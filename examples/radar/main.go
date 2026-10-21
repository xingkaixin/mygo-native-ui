// Radar compares four made-up laptops across seven measures, from battery
// life to fan noise, further out better on every axis. Point at an item
// or its legend entry to light it, near an axis to read every value on
// it, choose which laptops show beside the chart, and switch to the
// upgraded models to see the items move to their new values.
//
//	go run ./examples/radar
package main

import (
	"log"
	"math"
	"strconv"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/radar"
)

var axes = []radar.Axis{
	{Name: "Battery", Unit: "h"},
	{Name: "CPU", Unit: "pts", Format: grouped},
	{Name: "GPU", Unit: "TFLOPS"},
	{Name: "Display", Unit: "nits"},
	{Name: "Fan noise", Unit: "dB", Min: 20, Max: 45, LowerIsBetter: true},
	{Name: "Weight", Unit: "kg", Min: 0.5, Max: 3, LowerIsBetter: true},
	{Name: "Price", Min: 1000, Max: 3500, LowerIsBetter: true, Format: dollars},
}

// The indexes of the axes the list beside the chart reads.
const (
	weight = 5
	price  = 6
)

// laptop is a made-up laptop: its name, what it is, and its values on
// axes as it comes, and with its upgrades.
type laptop struct {
	name, about    string
	base, upgraded []float64
}

var laptops = []laptop{
	{"Aero 14", "14-inch ultralight, all-day battery",
		[]float64{18, 14200, 4.2, 600, 28, 1.35, 1499},
		[]float64{16.5, 16800, 5.6, 600, 30, 1.38, 1999}},
	{"Forge 16", "16-inch workstation with a discrete GPU",
		[]float64{9, 18900, 13.5, 500, 44, 2.25, 2399},
		[]float64{7.5, 22400, 19.8, 600, 43, 2.3, 3199}},
	{"Nimbus 13", "13-inch fanless, for the road",
		[]float64{15, 9800, 2.1, 400, 20, 1.12, 1099},
		[]float64{14, 11200, 2.6, 500, 20, 1.15, 1399}},
	{"Atlas 15", "15-inch creator laptop, bright screen",
		[]float64{12, 12500, 7.8, 1000, 36, 1.8, 1799},
		[]float64{11, 15100, 10.4, 1200, 38, 1.85, 2499}},
}

type app struct {
	radar    radar.State
	shown    []bool
	upgraded int
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	series := make([]radar.Series, len(laptops))
	for i, l := range laptops {
		values := l.base
		if a.upgraded == 1 {
			values = l.upgraded
		}
		series[i] = radar.Series{Name: l.name, Values: values}
	}
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(4), t.Space(5)).Gap(t.Space(4)).Children(func() {
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			ui.Column(c).Gap(t.Space(0.5)).Grow(1).Shrink(1).Children(func() {
				ui.Text(c, "Laptops compared").Bold().FontSize(t.FontSize + 4)
				ui.Text(c, "Four made-up laptops across seven measures, further out better on every axis").
					TextColor(t.TextMuted).SingleLine()
			})
			ui.Segmented(c, &a.upgraded, "As they come", "Upgraded").Label("Models")
		})
		ui.Row(c).Grow(1).AlignSelf(ui.Stretch).Gap(t.Space(6)).Children(func() {
			radar.View(c, &a.radar, axes, series, a.shown).Grow(1).AlignSelf(ui.Stretch)
			ui.Column(c).Width(t.Space(62)).Gap(t.Space(3)).Children(func() {
				ui.Text(c, "Show").Bold()
				for i, l := range laptops {
					values := series[i].Values
					ui.Column(c.Key(i)).Gap(t.Space(1)).Children(func() {
						ui.Checkbox(c, &a.shown[i], l.name)
						ui.Column(c).PaddingStart(t.Space(6)).Gap(t.Space(0.5)).Children(func() {
							ui.Text(c, l.about).FontSize(t.FontSize - 1).TextColor(t.TextMuted)
							ui.Text(c, dollars(values[price])+", "+strconv.FormatFloat(values[weight], 'f', -1, 64)+" kg").
								FontSize(t.FontSize - 1).TextColor(t.TextMuted).FontFeatures("tnum")
						})
					})
				}
			})
		})
	})
}

// grouped writes v rounded, with commas between groups of three digits.
func grouped(v float64) string {
	s := strconv.FormatFloat(math.Abs(math.Round(v)), 'f', 0, 64)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	if v <= -0.5 {
		s = "-" + s
	}
	return s
}

// dollars writes v as a price in whole dollars: $1,499.
func dollars(v float64) string { return "$" + grouped(v) }

func main() {
	a := &app{shown: []bool{true, true, true, false}}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Radar",
			Width:   1080,
			Height:  760,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
