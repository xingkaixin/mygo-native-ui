// Histslider filters 600 made-up stays by their price a night, over the
// spread of their prices: drag either end, and see how many stays are
// left.
//
//	go run ./examples/histslider
package main

import (
	"fmt"
	"log"
	"math"
	"math/rand/v2"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/histslider"
)

const top = 500

type app struct {
	prices    []float64
	low, high float64
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	price := histslider.Range{Min: 0, Max: top, Bins: 40, Format: func(v float64) string {
		if v >= top {
			return fmt.Sprintf("$%d+", top)
		}
		return fmt.Sprintf("$%.0f", v)
	}}
	left := 0
	for _, p := range a.prices {
		if p >= a.low && (p <= a.high || a.high >= top) {
			left++
		}
	}
	ui.Column(c).Fill().Background(t.Background).Justify(ui.Center).AlignItems(ui.Center).Children(func() {
		ui.Column(c).Width(t.Space(120)).Gap(t.Space(4)).Padding(t.Space(6)).Radius(t.Space(3)).Background(t.Surface).Border(1, t.Border).Children(func() {
			ui.Column(c).Gap(t.Space(1)).Children(func() {
				ui.Text(c, "Price range").Bold().FontSize(t.FontSize + 4)
				ui.Text(c, "Nightly prices before fees and taxes").TextColor(t.TextMuted)
			})
			histslider.View(c, &a.low, &a.high, a.prices, price)
			ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
				ui.Text(c, fmt.Sprintf("%d of %d stays", left, len(a.prices))).FontFeatures("tnum").Grow(1)
				if ui.Button(c, "Clear").Disabled(a.low == 0 && a.high == top).Clicked() {
					a.low, a.high = 0, top
				}
			})
		})
	})
}

func main() {
	r := rand.New(rand.NewPCG(7, 11))
	a := &app{low: 60, high: 240}
	for range 600 {
		a.prices = append(a.prices, math.Round(math.Exp(5+0.55*r.NormFloat64())))
	}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Price Range",
			Width:   640,
			Height:  480,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
