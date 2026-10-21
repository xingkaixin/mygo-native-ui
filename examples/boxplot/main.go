// Boxplot compares the response times of six made-up API endpoints over
// an hour, as box plots, violins or points: a health check fast and tight,
// lookups with a long tail, a feed and a checkout with two modes, as a
// cache's hits and misses, and uploads that sometimes time out. Hover an
// endpoint for its quartiles and a dot for its request; switch to a log
// scale to see the fast endpoints beside the slow, or lay the chart on its
// side.
//
//	go run ./examples/boxplot
package main

import (
	"fmt"
	"log"
	"math"
	"math/rand/v2"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/boxplot"
)

type app struct {
	endpoints []boxplot.Group
	chart     boxplot.State
	mode      int
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	a.chart.Mode = boxplot.Mode(a.mode)
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(4), t.Space(5)).Gap(t.Space(4)).Children(func() {
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			ui.Column(c).Gap(t.Space(0.5)).Grow(1).Shrink(1).Children(func() {
				ui.Text(c, "API latency").Bold().FontSize(t.FontSize + 4)
				ui.Text(c, "Response times of six endpoints over the last hour").TextColor(t.TextMuted).SingleLine()
			})
			ui.Segmented(c, &a.mode, "Box", "Violin", "Points").Label("Show as")
			ui.Text(c, "Log scale").TextColor(t.TextMuted)
			ui.Switch(c, &a.chart.Log).Label("Log scale")
			ui.Text(c, "Horizontal").TextColor(t.TextMuted)
			ui.Switch(c, &a.chart.Horizontal).Label("Horizontal")
		})
		boxplot.View(c, &a.chart, boxplot.Axis{Title: "Response time", Unit: "ms"}, a.endpoints).
			Grow(1).AlignSelf(ui.Stretch).MinHeight(t.Space(80))
	})
}

var regions = []string{"us-east", "us-west", "eu-west", "ap-south"}

// endpoints makes up an hour of requests to six endpoints, the same each
// run. Each takes a log-normal time around its median, some with a second,
// slower mode or rare spikes; each request is labeled with an id and the
// region that served it.
func endpoints() []boxplot.Group {
	r := rand.New(rand.NewPCG(2026, 10))
	lognormal := func(median, sigma float64) float64 { return median * math.Exp(sigma*r.NormFloat64()) }
	type spec struct {
		name string
		n    int
		draw func() (float64, string)
	}
	specs := []spec{
		{"GET /health", 600, func() (float64, string) {
			if r.Float64() < 0.012 {
				return lognormal(38, 0.3), "GC pause"
			}
			return lognormal(3.2, 0.22), ""
		}},
		{"GET /users/:id", 900, func() (float64, string) {
			if r.Float64() < 0.03 {
				return lognormal(150, 0.25), "cache miss"
			}
			return lognormal(24, 0.3), ""
		}},
		{"GET /search", 700, func() (float64, string) {
			return lognormal(110, 0.55), ""
		}},
		{"GET /feed", 800, func() (float64, string) {
			if r.Float64() < 0.35 {
				return lognormal(88, 0.2), "cache miss"
			}
			return lognormal(17, 0.25), ""
		}},
		{"POST /orders", 420, func() (float64, string) {
			if r.Float64() < 0.45 {
				return lognormal(245, 0.18), "card payment"
			}
			return lognormal(72, 0.18), ""
		}},
		{"POST /upload", 260, func() (float64, string) {
			if r.Float64() < 0.025 {
				return 3000, "timed out"
			}
			return lognormal(420, 0.35), ""
		}},
	}
	out := make([]boxplot.Group, len(specs))
	for i, s := range specs {
		g := boxplot.Group{Name: s.name, Values: make([]float64, s.n), Labels: make([]string, s.n)}
		for j := range s.n {
			v, why := s.draw()
			g.Values[j] = math.Round(v*10) / 10
			g.Labels[j] = fmt.Sprintf("req %06x · %s", r.Uint32()&0xffffff, regions[r.IntN(len(regions))])
			if why != "" {
				g.Labels[j] += " · " + why
			}
		}
		out[i] = g
	}
	return out
}

func main() {
	a := &app{endpoints: endpoints(), chart: boxplot.State{Log: true}}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Box plot",
			Width:   1100,
			Height:  760,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
