// Sparkline is a service's live dashboard: cards of its numbers over the
// last minute, a table of its endpoints each with its latency, and its
// deploys by day. Point at a line or a bar to read a value.
//
//	go run ./examples/sparkline
package main

import (
	"fmt"
	"log"
	"math/rand/v2"
	"slices"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/sparkline"
)

const keep = 60

// series is a value that wanders around base, as live numbers do.
type series struct {
	base, spread float64
	values       []float64
}

func (s *series) step(r *rand.Rand) {
	v := s.base
	if n := len(s.values); n > 0 {
		v = s.values[n-1]
	}
	v += (r.Float64()-0.5)*s.spread + (s.base-v)*0.1
	s.values = append(s.values, max(0, v))
	if len(s.values) > keep {
		s.values = s.values[1:]
	}
}

// change writes how much the last value is above the first, and whether
// it went up.
func (s *series) change() (string, bool) {
	first, last := s.values[0], s.values[len(s.values)-1]
	if first == 0 {
		return "", false
	}
	d := (last - first) / first * 100
	return fmt.Sprintf("%+.1f%%", d), d > 0
}

type app struct {
	rand      *rand.Rand
	next      time.Time
	requests  series
	latency   series
	errors    series
	users     series
	endpoints []*series
	deploys   []float64
}

var names = []string{"GET /api/projects", "POST /api/builds", "GET /api/builds/:id", "GET /api/logs", "POST /api/auth/token"}

func newApp() *app {
	a := &app{
		rand:     rand.New(rand.NewPCG(1, 2)),
		requests: series{base: 1200, spread: 160},
		latency:  series{base: 180, spread: 30},
		errors:   series{base: 0.6, spread: 0.25},
		users:    series{base: 3400, spread: 90},
		deploys:  []float64{3, 5, 2, 6, 4, 1, 0, 4, 7, 3, 5, 2, 0, 1},
	}
	for i := range names {
		a.endpoints = append(a.endpoints, &series{base: 60 + float64(i)*70, spread: 20 + float64(i)*6})
	}
	for range keep {
		a.tick()
	}
	return a
}

func (a *app) tick() {
	for _, s := range append([]*series{&a.requests, &a.latency, &a.errors, &a.users}, a.endpoints...) {
		s.step(a.rand)
	}
}

func (a *app) stat(label, unit string, s *series, upIsWorse bool, format string) sparkline.Stat {
	change, up := s.change()
	return sparkline.Stat{
		Label:  label,
		Value:  fmt.Sprintf(format, s.values[len(s.values)-1]) + unit,
		Change: change,
		Worse:  up == upIsWorse,
		Values: s.values,
	}
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	now := c.Now()
	if now.After(a.next) {
		if !a.next.IsZero() {
			a.tick()
		}
		a.next = now.Add(time.Second)
	}
	c.After(a.next.Sub(now))
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(6)).Gap(t.Space(5)).Children(func() {
		ui.Row(c).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "api-gateway").Bold().FontSize(t.FontSize + 6).Grow(1)
			ui.Text(c, "Last minute, live").TextColor(t.TextMuted)
		})
		ui.Grid(c).Columns(4).Gap(t.Space(4)).Children(func() {
			sparkline.Card(c, a.stat("Requests", "/s", &a.requests, false, "%.0f"))
			sparkline.Card(c, a.stat("p95 latency", " ms", &a.latency, true, "%.0f"))
			sparkline.Card(c, a.stat("Errors", "%", &a.errors, true, "%.2f"))
			sparkline.Card(c, a.stat("Active users", "", &a.users, false, "%.0f"))
		})
		ui.Row(c).Gap(t.Space(4)).AlignItems(ui.Start).Children(func() {
			ui.Column(c).Grow(1).Padding(t.Space(4)).Radius(t.Space(2.5)).Border(1, t.Border).Children(func() {
				ui.Text(c, "Endpoints").Bold().Padding(0, 0, t.Space(2))
				for i, s := range a.endpoints {
					if i > 0 {
						ui.Divider(c)
					}
					ui.Row(c.Key(i)).Gap(t.Space(4)).Padding(t.Space(2), 0).AlignItems(ui.Center).Children(func() {
						ui.Text(c, names[i]).Font("monospace").FontSize(t.FontSize - 1).Grow(1)
						sparkline.Line(c, s.values, ui.Color{}).Size(t.Space(40), t.Space(6))
						last := s.values[len(s.values)-1]
						ui.Text(c, fmt.Sprintf("%.0f ms", last)).FontFeatures("tnum").Width(t.Space(16)).TextAlign(ui.End)
					})
				}
			})
			ui.Column(c).Width(t.Space(80)).Gap(t.Space(2)).Padding(t.Space(4)).Radius(t.Space(2.5)).Border(1, t.Border).Children(func() {
				ui.Text(c, "Deploys").Bold()
				total := 0.0
				for _, d := range a.deploys {
					total += d
				}
				ui.Text(c, fmt.Sprintf("%.0f in the last %d days", total, len(a.deploys))).FontSize(t.FontSize - 1).TextColor(t.TextMuted)
				sparkline.Bars(c, a.deploys, ui.Oklch(0.65, 0.15, 150)).Height(t.Space(24)).AlignSelf(ui.Stretch)
				ui.Text(c, fmt.Sprintf("Busiest: %.0f in a day", slices.Max(a.deploys))).FontSize(t.FontSize - 1).TextColor(t.TextMuted)
			})
		})
	})
}

func main() {
	a := newApp()
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Sparkline",
			Width:   1120,
			Height:  640,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
