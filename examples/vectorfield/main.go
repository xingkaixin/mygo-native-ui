// Vectorfield plots five fields of the plane, from a whirlpool to a
// swinging pendulum, with a few streamlines drawn through them.
//
//	go run ./examples/vectorfield
package main

import (
	"log"
	"math"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/vectorfield"
)

var fields = []struct {
	name  string
	span  float64
	f     vectorfield.Field
	seeds [][2]float64
}{
	{"Vortex", 2, func(x, y float64) (float64, float64) {
		return -y - 0.15*x, x - 0.15*y
	}, [][2]float64{{1.6, 0}}},
	{"Saddle", 2, func(x, y float64) (float64, float64) {
		return x, -y
	}, [][2]float64{{0.2, 1.8}, {-0.2, -1.8}}},
	{"Dipole", 2, func(x, y float64) (float64, float64) {
		ex, ey := 0.0, 0.0
		for _, q := range []struct{ x, s float64 }{{-0.6, 1}, {0.6, -1}} {
			dx, dy := x-q.x, y
			r3 := math.Pow(dx*dx+dy*dy+0.01, 1.5)
			ex, ey = ex+q.s*dx/r3, ey+q.s*dy/r3
		}
		return ex, ey
	}, [][2]float64{{-0.5, 0.1}, {-0.5, -0.1}, {-0.7, 0.1}}},
	{"Van der Pol", 4, func(x, y float64) (float64, float64) {
		return y, (1-x*x)*y - x
	}, [][2]float64{{0.1, 0}}},
	{"Pendulum", 7, func(x, y float64) (float64, float64) {
		return y, -math.Sin(x) - 0.1*y
	}, [][2]float64{{-6, 3}, {0, 1.8}}},
}

type app struct {
	field int
	shown int
	plot  vectorfield.State
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	if a.field != a.shown || a.plot.Seeds == nil {
		a.shown = a.field
		a.plot = vectorfield.State{Seeds: append([][2]float64{}, fields[a.field].seeds...)}
	}
	fd := fields[a.field]
	ui.Column(c).Fill().Background(t.Background).Children(func() {
		ui.Row(c).Gap(t.Space(2)).Padding(t.Space(3)).AlignItems(ui.Center).Children(func() {
			names := make([]string, len(fields))
			for i, f := range fields {
				names[i] = f.name
			}
			ui.Segmented(c, &a.field, names...)
			ui.Box(c).Grow(1)
			if ui.Button(c, "Clear").Clicked() {
				a.plot.Seeds = [][2]float64{}
			}
		})
		vectorfield.View(c, &a.plot, fd.f, fd.span).Grow(1)
	})
}

func main() {
	a := &app{}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Vector field",
			Width:   760,
			Height:  720,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
