// Streamgraph shows made-up listening on a music service by genre, month
// by month for twenty years, as a streamgraph. Point at a genre to read
// every genre's hours that month, click it to see it alone, and switch
// the baseline and the order of the layers.
//
//	go run ./examples/streamgraph
package main

import (
	"log"
	"math"
	"math/rand/v2"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/streamgraph"
)

// genre is how a genre's listening goes: it peaks in the year peak, as
// high as height, rising and falling over about spread years, above a
// floor it never drops below.
type genre struct {
	name                 string
	peak, spread, height float64
	floor                float64
}

var genres = []genre{
	{"Rock", 2006, 6, 32, 6},
	{"Pop", 2014, 9, 38, 10},
	{"Hip-hop", 2020, 6, 44, 4},
	{"Electronic", 2013, 4, 22, 4},
	{"R&B", 2016, 7, 16, 4},
	{"Country", 2010, 8, 12, 5},
	{"Indie", 2011, 3.5, 18, 2},
	{"Latin", 2022, 4, 26, 2},
	{"K-pop", 2022, 2.5, 18, 0.5},
	{"Jazz", 2005, 10, 3, 2},
	{"Afrobeats", 2026, 2.5, 18, 0.3},
}

// listening returns the first of each month from 2005 to 2024 and each
// genre's hours in it, in millions: more each year as the service grows,
// more around December, and wandering a little from month to month.
func listening() ([]time.Time, []streamgraph.Layer) {
	r := rand.New(rand.NewPCG(7, 11))
	var months []time.Time
	for y := 2005; y < 2025; y++ {
		for m := time.January; m <= time.December; m++ {
			months = append(months, time.Date(y, m, 1, 0, 0, 0, 0, time.UTC))
		}
	}
	layers := make([]streamgraph.Layer, len(genres))
	for k, g := range genres {
		layers[k].Name = g.name
		drift := 0.0
		for _, m := range months {
			year := float64(m.Year()) + float64(m.Month()-1)/12
			d := (year - g.peak) / g.spread
			drift = 0.85*drift + 0.15*(r.Float64()*2-1)
			v := (g.floor + g.height*math.Exp(-d*d)) * (0.7 + 0.03*(year-2005))
			v *= 1 + 0.03*math.Cos(2*math.Pi*float64(m.Month())/12) + 0.25*drift
			layers[k].Values = append(layers[k].Values, math.Round(v*10)/10)
		}
	}
	return months, layers
}

type app struct {
	months []time.Time
	layers []streamgraph.Layer
	base   int
	order  int
	stream streamgraph.State
}

func newApp() *app {
	a := &app{}
	a.months, a.layers = listening()
	return a
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(5), t.Space(6)).Gap(t.Space(4)).Children(func() {
		ui.Row(c).Gap(t.Space(4)).AlignItems(ui.Center).Children(func() {
			ui.Column(c).Gap(t.Space(0.5)).Grow(1).Shrink(1).Children(func() {
				ui.Text(c, "Listening by genre, 2005–2024").Bold().FontSize(t.FontSize + 4)
				ui.Text(c, "Hours streamed each month, in millions").TextColor(t.TextMuted).SingleLine()
			})
			ui.Segmented(c, &a.base, "Wiggle", "Silhouette", "Zero").Label("Baseline")
			ui.Segmented(c, &a.order, "Inside out", "As given").Label("Order")
		})
		bases := []streamgraph.Baseline{streamgraph.Wiggle, streamgraph.Silhouette, streamgraph.Zero}
		orders := []streamgraph.Order{streamgraph.InsideOut, streamgraph.Given}
		streamgraph.View(c, &a.stream, a.months, a.layers, bases[a.base], orders[a.order], "million hours").
			Grow(1).AlignSelf(ui.Stretch)
	})
}

func main() {
	a := newApp()
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Streamgraph",
			Width:   1120,
			Height:  720,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
