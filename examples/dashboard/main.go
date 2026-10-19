// Dashboard shows a made-up service's health with the charts chartx adds
// to package chart: sparklines of the last hour by the numbers, a heatmap
// of requests by weekday and hour, a histogram of response times, and a
// calendar of a year of deploys. The heatmap's cells, the histogram's
// bars and the calendar's days say what they are under the pointer.
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
)

// days are the heatmap's rows, from the bottom up, so that Monday is at
// the top.
var days = []string{"Sun", "Sat", "Fri", "Thu", "Wed", "Tue", "Mon"}

// service is the data the window shows, the same at every start.
type service struct {
	requests, latency, errors []float64      // a minute each, the last hour
	traffic                   [7][24]float64 // requests by row of days and hour
	times                     []float64      // response times in ms
	firstDay                  time.Time
	deploys                   []float64 // a day each, the last year
}

func newService() *service {
	r := rand.New(rand.NewPCG(3, 4))
	s := &service{}
	for i := range 60 {
		wave := math.Sin(float64(i) / 9)
		s.requests = append(s.requests, 1200+300*wave+80*r.NormFloat64())
		s.latency = append(s.latency, 180+40*wave+25*r.NormFloat64())
		s.errors = append(s.errors, math.Max(0, 3+2*r.NormFloat64()))
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
	return s
}

func (s *service) view(c *ui.Context) {
	t := c.Theme()
	ui.Scroll(c).Fill().Background(t.Background).Children(func() {
		ui.Column(c).Padding(20).Gap(16).Children(func() {
			ui.Text(c, "Notes API").FontSize(18).Bold()
			ui.Row(c).Gap(16).Children(func() {
				s.stat(c, "Requests / min", s.requests, "%.0f", t.Accent)
				s.stat(c, "p95 latency", s.latency, "%.0f ms", ui.Hex("#f59e0b"))
				s.stat(c, "Errors / min", s.errors, "%.1f", ui.Hex("#ef4444"))
			})
			ui.Row(c).Gap(16).Children(func() {
				card(c, "Requests by hour", func() { s.heatmap(c) }).Grow(3).Basis(0)
				card(c, "Response times", func() { s.histogram(c) }).Grow(2).Basis(0)
			})
			card(c, "Deploys", func() {
				chartx.Calendar(c, s.firstDay, s.deploys, nil, func(day time.Time, v float64) string {
					if v == 1 {
						return "1 deploy on " + day.Format("Jan 2")
					}
					return fmt.Sprintf("%.0f deploys on %s", v, day.Format("Jan 2"))
				})
			})
		})
	})
}

// card builds a box of a part of the dashboard, titled.
func card(c *ui.Context, title string, build func()) ui.Element {
	t := c.Theme()
	return ui.Column(c).Padding(16).Gap(12).Radius(10).Border(1, t.Border).Children(func() {
		ui.Text(c, title).FontSize(13).Bold()
		build()
	})
}

// stat builds a number of the last minute with a sparkline of the hour.
func (s *service) stat(c *ui.Context, label string, values []float64, format string, color ui.Color) {
	t := c.Theme()
	ui.Row(c).Padding(16).Gap(12).Radius(10).Border(1, t.Border).AlignItems(ui.End).Grow(1).Basis(0).Children(func() {
		ui.Column(c).Gap(4).Grow(1).Children(func() {
			ui.Text(c, label).FontSize(12).TextColor(t.TextMuted)
			ui.Text(c, fmt.Sprintf(format, values[len(values)-1])).FontSize(22).Bold().FontFeatures("tnum")
		})
		chartx.Sparkline(c, values, color).Size(96, 32)
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
	}).Label("Requests by weekday and hour").Height(200).GapX(8).GapY(6)
	ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
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
	}).Label("Response times").Height(200).GapX(8).GapY(6)
	ui.Text(c, "ms").FontSize(11).TextColor(t.TextMuted).AlignSelf(ui.End)
}

func main() {
	s := newService()
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Dashboard",
			Width:   960,
			Height:  780,
			Content: ui.View(s.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
