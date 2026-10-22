// Ridgeline compares the daily highs of each month over twenty years in a
// made-up temperate city, a ridge for each month: tight and warm in
// summer, wide in winter, with cold snaps that leave a long tail below.
// Drag the sliders to overlap the ridges more or less and to smooth or
// sharpen them, mark the medians or the quartiles, color the ridges by
// temperature, or sort the months by their medians. Hover a ridge for its
// density and the share of days colder.
//
//	go run ./examples/ridgeline
package main

import (
	"fmt"
	"log"
	"math"
	"math/rand/v2"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/ridgeline"
)

type app struct {
	months    []ridgeline.Group
	chart     ridgeline.State
	overlap   float64
	bandwidth float64
	marks     int
	gradient  bool
}

// warm runs from deep blue through pale yellow to dark red, as
// ColorBrewer's RdYlBu reversed, for temperatures.
var warm = []ui.Color{
	ui.Hex("#313695"), ui.Hex("#4575b4"), ui.Hex("#74add1"), ui.Hex("#abd9e9"), ui.Hex("#fee090"),
	ui.Hex("#fdae61"), ui.Hex("#f46d43"), ui.Hex("#d73027"), ui.Hex("#a50026"),
}

var marks = [][]float64{nil, {0.5}, {0.25, 0.5, 0.75}}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	a.chart.Overlap = float32(a.overlap)
	a.chart.Bandwidth = a.bandwidth
	a.chart.Quantiles = marks[a.marks]
	a.chart.Gradient = nil
	if a.gradient {
		a.chart.Gradient = warm
	}
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(4), t.Space(5)).Gap(t.Space(4)).Children(func() {
		ui.Column(c).Gap(t.Space(0.5)).Children(func() {
			ui.Text(c, "Daily highs by month").Bold().FontSize(t.FontSize + 4)
			ui.Text(c, "Twenty years of daily high temperatures in a made-up city").TextColor(t.TextMuted).SingleLine()
		})
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "Overlap").TextColor(t.TextMuted)
			ui.Slider(c, &a.overlap, 0, 4).Label("Overlap").Width(t.Space(30))
			ui.Text(c, fmt.Sprintf("%.1f", a.overlap)).FontFeatures("tnum").Width(t.Space(8))
			ui.Text(c, "Bandwidth").TextColor(t.TextMuted)
			ui.Slider(c, &a.bandwidth, 0.2, 3).Label("Bandwidth").Width(t.Space(30))
			ui.Text(c, fmt.Sprintf("×%.1f", a.bandwidth)).FontFeatures("tnum").Width(t.Space(8))
			ui.Box(c).Grow(1)
			ui.Segmented(c, &a.marks, "None", "Median", "Quartiles").Label("Marks")
			ui.Text(c, "Gradient").TextColor(t.TextMuted)
			ui.Switch(c, &a.gradient).Label("Gradient")
			ui.Text(c, "Sort").TextColor(t.TextMuted)
			ui.Switch(c, &a.chart.Sorted).Label("Sort by median")
		})
		ridgeline.View(c, &a.chart, ridgeline.Axis{Title: "Daily high", Unit: "°C"}, a.months).
			Grow(1).AlignSelf(ui.Stretch).MinHeight(t.Space(80))
	})
}

var names = []string{"January", "February", "March", "April", "May", "June",
	"July", "August", "September", "October", "November", "December"}

// months makes up twenty years of daily highs, the same each run: each
// day's high is normal about its month's mean, wider in winter, with cold
// snaps in winter and heat waves in summer.
func months() []ridgeline.Group {
	r := rand.New(rand.NewPCG(2026, 10))
	mean := []float64{2, 4, 9, 15, 21, 26, 29, 28, 23, 16, 9, 3}
	days := []int{31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}
	out := make([]ridgeline.Group, 12)
	for m := range 12 {
		// Winter is 1 at January and 0 at July.
		winter := (1 + math.Cos(2*math.Pi*float64(m)/12)) / 2
		sd := 3.5 + 2.5*winter
		g := ridgeline.Group{Name: names[m]}
		for range 20 * days[m] {
			v := mean[m] + sd*r.NormFloat64()
			switch u := r.Float64(); {
			case u < 0.12*winter:
				v -= 7 + 3*r.Float64()
			case u > 1-0.08*(1-winter):
				v += 4 + 3*r.Float64()
			}
			g.Values = append(g.Values, math.Round(v*10)/10)
		}
		out[m] = g
	}
	return out
}

func main() {
	a := &app{months: months(), overlap: 1.5, bandwidth: 1, marks: 1}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Ridgeline",
			Width:   1100,
			Height:  820,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
