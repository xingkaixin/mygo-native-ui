// Density plots 4,000 eruptions of a made-up geyser, how long each lasted
// against how long the wait before it was: short eruptions after short
// waits, long ones after long waits, and a few between. Switch between
// hexagonal bins and contours, and hover a hexagon for its count.
//
//	go run ./examples/density
package main

import (
	"log"
	"math/rand/v2"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/density"
)

func eruptions(n int) [][2]float64 {
	r := rand.New(rand.NewPCG(1, 2))
	points := make([][2]float64, n)
	for i := range points {
		switch f := r.Float64(); {
		case f < 0.34:
			points[i] = [2]float64{2.0 + r.NormFloat64()*0.28, 54 + r.NormFloat64()*5.5}
		case f < 0.95:
			points[i] = [2]float64{4.3 + r.NormFloat64()*0.38, 80 + r.NormFloat64()*6}
		default:
			points[i] = [2]float64{2.6 + r.Float64()*1.6, 60 + r.Float64()*20}
		}
	}
	return points
}

type app struct {
	points [][2]float64
	plot   density.State
	mode   int
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	a.plot.Mode = density.Mode(a.mode)
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(6)).Gap(t.Space(4)).Children(func() {
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "Geyser eruptions").Bold().FontSize(t.FontSize + 4).Grow(1)
			ui.Segmented(c, &a.mode, "Hexbin", "Contour").Label("Show as")
		})
		density.View(c, &a.plot, density.Axis{Title: "Eruption", Unit: "min"},
			density.Axis{Title: "Wait before", Unit: "min"}, a.points).Grow(1)
	})
}

func main() {
	a := &app{points: eruptions(4000)}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Density",
			Width:   760,
			Height:  600,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
