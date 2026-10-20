// Dashboard shows a made-up service's health on a grid of widgets, which
// drag by their titles to move them and by their bottom right corners to
// resize them: numbers of the last minute with sparklines of the hour, a
// heatmap of requests by weekday and hour, a histogram of response times,
// recent alerts, and a calendar of a year of deploys, drawn with the
// charts chartx adds to package chart. The heatmap's cells, the
// histogram's bars and the calendar's days say what they are under the
// pointer.
//
//	go run ./examples/dashboard
package main

import (
	"fmt"
	"log"
	"math"
	"math/rand/v2"
	"strconv"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/chart"

	"github.com/xingkaixin/mygo-native-ui/ui/chartx"
	"github.com/xingkaixin/mygo-native-ui/ui/dashboard"
)

// days are the heatmap's rows, from the bottom up, so that Monday is at
// the top.
var days = []string{"Sun", "Sat", "Fri", "Thu", "Wed", "Tue", "Mon"}

// severity is how bad an alert is.
type severity uint8

const (
	critical severity = iota
	warning
	resolved
)

// alert is an alert that fired: how bad, what, where and when.
type alert struct {
	severity severity
	title    string
	where    string
	ago      string
}

var alerts = []alert{
	{critical, "Error rate above 2%", "api-gateway · eu-west-1", "2m"},
	{warning, "p95 latency above 250 ms", "notes-api · us-east-1", "9m"},
	{warning, "Disk 85% full", "postgres-primary", "24m"},
	{resolved, "Pod restarting", "worker-7f9c4", "41m"},
	{critical, "Certificate expires in 3 days", "notes.example.com", "1h"},
	{resolved, "Queue depth above 10k", "jobs-queue", "2h"},
	{warning, "Memory above 90%", "search-indexer", "3h"},
	{resolved, "Health check failing", "cache-2 · us-east-1", "5h"},
	{resolved, "Replication lag above 30 s", "postgres-replica-1", "7h"},
}

// service is the data the window shows, the same at every start, and the
// widgets that show it.
type service struct {
	requests, latency, errors, cpu []float64      // a minute each, the last hour
	traffic                        [7][24]float64 // requests by row of days and hour
	times                          []float64      // response times in ms
	firstDay                       time.Time
	deploys                        []float64 // a day each, the last year

	board   dashboard.State
	widgets []dashboard.Widget
	panels  []func(c *ui.Context) // what each widget shows
}

func newService() *service {
	r := rand.New(rand.NewPCG(3, 4))
	s := &service{}
	for i := range 60 {
		wave := math.Sin(float64(i) / 9)
		s.requests = append(s.requests, 1200+300*wave+80*r.NormFloat64())
		s.latency = append(s.latency, 180+40*wave+25*r.NormFloat64())
		s.errors = append(s.errors, math.Max(0, 3+2*r.NormFloat64()))
		s.cpu = append(s.cpu, 52+14*wave+4*r.NormFloat64())
	}
	for j, day := range days {
		weekend := day == "Sat" || day == "Sun"
		for h := range 24 {
			// Busiest in working hours, least at night and at weekends.
			v := 200 + 1800*math.Exp(-math.Pow(float64(h)-14, 2)/18)
			if weekend {
				v *= 0.35
			}
			s.traffic[j][h] = math.Round(v * (0.85 + 0.3*r.Float64()))
		}
	}
	for range 3000 {
		s.times = append(s.times, math.Exp(5+0.45*r.NormFloat64()))
	}
	today := time.Date(2026, 10, 19, 0, 0, 0, 0, time.UTC)
	s.firstDay = today.AddDate(-1, 0, 1)
	for k := 0; !s.firstDay.AddDate(0, 0, k).After(today); k++ {
		n := 0.0
		if wd := s.firstDay.AddDate(0, 0, k).Weekday(); wd != time.Saturday && wd != time.Sunday && r.Float64() < 0.7 {
			n = math.Floor(math.Abs(r.NormFloat64()) * 4)
		}
		s.deploys = append(s.deploys, n)
	}

	s.widgets = []dashboard.Widget{
		{Title: "Requests / min", X: 0, Y: 0, W: 3, H: 2, MinW: 2, MinH: 2},
		{Title: "p95 latency", X: 3, Y: 0, W: 3, H: 2, MinW: 2, MinH: 2},
		{Title: "Errors / min", X: 6, Y: 0, W: 3, H: 2, MinW: 2, MinH: 2},
		{Title: "CPU", X: 9, Y: 0, W: 3, H: 2, MinW: 2, MinH: 2},
		{Title: "Requests by hour", X: 0, Y: 2, W: 8, H: 4, MinW: 5, MinH: 3},
		{Title: "Recent alerts", X: 8, Y: 2, W: 4, H: 8, MinW: 3, MinH: 3},
		{Title: "Response times", X: 0, Y: 6, W: 8, H: 4, MinW: 4, MinH: 3},
		{Title: "Deploys", X: 0, Y: 10, W: 12, H: 3, MinW: 9, MinH: 3},
	}
	s.panels = []func(c *ui.Context){
		func(c *ui.Context) { stat(c, s.requests, "%.0f", false, c.Theme().Accent) },
		func(c *ui.Context) { stat(c, s.latency, "%.0f ms", true, c.Theme().Warning) },
		func(c *ui.Context) { stat(c, s.errors, "%.1f", true, c.Theme().Danger) },
		func(c *ui.Context) { stat(c, s.cpu, "%.0f%%", true, ui.Oklch(0.62, 0.16, 300)) },
		s.heatmap,
		alertList,
		s.histogram,
		func(c *ui.Context) {
			chartx.Calendar(c, s.firstDay, s.deploys, nil, func(day time.Time, v float64) string {
				if v == 1 {
					return "1 deploy on " + day.Format("Jan 2")
				}
				return fmt.Sprintf("%.0f deploys on %s", v, day.Format("Jan 2"))
			}).AlignSelf(ui.Start)
		},
	}
	return s
}

func (s *service) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Surface).Children(func() {
		ui.Row(c).Gap(t.Space(2)).Padding(t.Space(4), t.Space(3), 0).AlignItems(ui.End).Children(func() {
			ui.Text(c, "Notes API").FontSize(18).Bold()
			ui.Text(c, "Production · the last hour").TextColor(t.TextMuted)
		})
		dashboard.View(c, &s.board, &s.widgets, 12, 60, func(i int) { s.panels[i](c) }).Grow(1)
	})
}

// stat builds the number of the last minute in format, how it changed
// over the hour, red when it rose and up is worse or it fell and up is
// better, and a sparkline of the hour.
func stat(c *ui.Context, values []float64, format string, upWorse bool, color ui.Color) {
	t := c.Theme()
	last, first := values[len(values)-1], values[0]
	change := (last - first) / first * 100
	ui.Column(c).Grow(1).Gap(t.Space(2)).Children(func() {
		ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, fmt.Sprintf(format, last)).FontSize(t.FontSize + 8).Bold().FontFeatures("tnum")
			badge := t.Success
			if (change > 0) == upWorse {
				badge = t.Danger
			}
			ui.Text(c, fmt.Sprintf("%+.0f%%", change)).FontSize(t.FontSize-1).FontFeatures("tnum").TextColor(badge).
				Padding(t.Space(0.5), t.Space(1.5)).Radius(t.Space(1)).Background(badge.Alpha(0.12))
		})
		chartx.Sparkline(c, values, color).WidthPercent(100).Grow(1)
	})
}

// alertList builds the alerts, the latest first: a dot of how bad each
// is, what, where and how long ago.
func alertList(c *ui.Context) {
	t := c.Theme()
	ui.Scroll(c).Grow(1).Children(func() {
		ui.Column(c).Children(func() {
			for i, a := range alerts {
				color, state := t.Danger, "Critical"
				switch a.severity {
				case warning:
					color, state = t.Warning, "Warning"
				case resolved:
					color, state = t.Success, "Resolved"
				}
				if i > 0 {
					ui.Divider(c)
				}
				ui.Row(c.Key(i)).Gap(t.Space(2.5)).Padding(t.Space(2), 0).AlignItems(ui.Center).Children(func() {
					ui.Box(c).Size(t.Space(2), t.Space(2)).Radius(t.Space(1)).Background(color).Shrink(0)
					ui.Column(c).Gap(t.Space(0.5)).Grow(1).Shrink(1).Children(func() {
						title := ui.Text(c, a.title).SingleLine()
						if a.severity == resolved {
							title.TextColor(t.TextMuted)
						}
						ui.Text(c, state+" · "+a.where).FontSize(t.FontSize - 2).TextColor(t.TextMuted).SingleLine()
					})
					ui.Text(c, a.ago).FontSize(t.FontSize - 2).TextColor(t.TextMuted).FontFeatures("tnum").NoWrap()
				})
			}
		})
	})
}

func (s *service) heatmap(c *ui.Context) {
	t := c.Theme()
	most := 0.0
	for _, row := range s.traffic {
		for _, v := range row {
			most = max(most, v)
		}
	}
	color := chartx.NewSequential(0, most, chartx.Viridis...)
	hours := make([]string, 24)
	for h := range hours {
		hours[h] = strconv.Itoa(h)
	}
	x, y := chart.NewBand(hours...).Padding(0.08, 0), chart.NewBand(days...).Padding(0.08, 0)
	xt := x.Ticks()
	for h := range xt {
		if h%3 != 0 {
			xt[h].Label = ""
		}
	}
	label := func(_ chart.Side, tick chart.Tick) {
		ui.Text(c, tick.Label).NoWrap().FontSize(11).TextColor(t.TextMuted)
	}
	chart.Plot(c, chart.Axes{Left: y.Ticks(), Bottom: xt, Label: label}, func(area ui.Element) {
		i, j, ok := chartx.Heatmap(c, area, x, y, func(i, j int) ui.Color { return color.At(s.traffic[j][i]) })
		if ok {
			text := fmt.Sprintf("%s %02d:00 · %.0f requests", days[j], i, s.traffic[j][i])
			chart.Anchor(c, x.At(i), y.Band(j).Hi, ui.Center, ui.End, func() {
				ui.Text(c, text).NoWrap().FontSize(12).Padding(4, 8).Radius(6).
					Background(t.Text).TextColor(t.Background).Margin(0, 0, 6, 0)
			})
		}
	}).Label("Requests by weekday and hour").Grow(1).GapX(8).GapY(6)
	ui.Row(c).Gap(8).Margin(t.Space(2), 0, 0).AlignItems(ui.Center).Children(func() {
		ui.Text(c, "0").FontSize(11).TextColor(t.TextMuted)
		ui.Box(c).Size(120, 8).Radius(4).LinearGradient(color.Gradient())
		ui.Text(c, fmt.Sprintf("%.0f requests", most)).FontSize(11).TextColor(t.TextMuted)
	})
}

func (s *service) histogram(c *ui.Context) {
	t := c.Theme()
	bins := chartx.Bins(s.times, 24)
	most := 0
	for _, b := range bins {
		most = max(most, b.Count)
	}
	x := chart.NewLinear(bins[0].Lo, bins[len(bins)-1].Hi)
	y := chart.NewLinear(0, float64(most)).Nice(4)
	label := func(_ chart.Side, tick chart.Tick) {
		ui.Text(c, tick.Label).NoWrap().FontSize(11).TextColor(t.TextMuted)
	}
	chart.Plot(c, chart.Axes{Left: y.Ticks(4), Bottom: x.Ticks(5), Label: label}, func(area ui.Element) {
		for i, b := range bins {
			bar := chart.Rect(c.Key(i), x.Span(b.Lo, b.Hi), y.Span(0, float64(b.Count))).
				Tooltip(fmt.Sprintf("%g–%g ms · %d requests", b.Lo, b.Hi, b.Count))
			fill := t.Accent
			if bar.Hovered() {
				fill = t.Accent.Mix(t.Text, 0.3)
			}
			bar.Draw(func(p *ui.Painter, r ui.Rect) {
				// A gap of a pixel between bars, which touch.
				p.Fill(ui.Rect{X: r.X + 0.5, Y: r.Y, W: r.W - 1, H: r.H}, fill, 0)
			})
		}
	}).Label("Response times").Grow(1).GapX(8).GapY(6).Margin(t.Space(2), 0, 0)
	ui.Text(c, "ms").FontSize(11).TextColor(t.TextMuted).AlignSelf(ui.End)
}

func main() {
	s := newService()
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Dashboard",
			Width:   1120,
			Height:  880,
			Content: ui.View(s.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
