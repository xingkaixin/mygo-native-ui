// Hexbin bins 5,000 points drawn from two overlapping clusters. Drag the
// slider to change the size of the bins.
//
//	go run ./examples/hexbin
package main

import (
	"fmt"
	"log"
	"math/rand/v2"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/hexbin"
)

type app struct {
	bins   hexbin.State
	points []hexbin.Point
	radius float64
}

func sample() []hexbin.Point {
	r := rand.New(rand.NewPCG(1, 2))
	out := make([]hexbin.Point, 0, 5000)
	for i := range 5000 {
		a, b := r.NormFloat64(), r.NormFloat64()
		if i%3 == 0 {
			out = append(out, hexbin.Point{X: 70 + 9*a, Y: 4.4 + 0.5*(0.4*a+0.9*b)})
		} else {
			out = append(out, hexbin.Point{X: 45 + 6*a, Y: 2.1 + 0.35*(0.6*a+0.8*b)})
		}
	}
	return out
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(8)).Gap(t.Space(4)).Children(func() {
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "Bin size").TextColor(t.TextMuted)
			ui.Slider(c, &a.radius, 4, 28).Step(1).Width(t.Space(48)).Label("Bin size")
			ui.Text(c, fmt.Sprintf("%.0f", a.radius)).FontFeatures("tnum")
		})
		hexbin.View(c, &a.bins, a.points, float32(a.radius)).Grow(1)
	})
}

func main() {
	a := &app{points: sample(), radius: 10}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Hexbin",
			Width:   820,
			Height:  600,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
