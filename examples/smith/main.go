// Smith plots two revisions of a made-up 2.4 GHz antenna swept from
// 2.3 to 2.6 GHz on a 50 Ω line, each a parallel resonance behind a bit
// of series inductance: the prototype never comes within VSWR 2, and
// revision B, with a shorter feed and a broader resonance, stays inside
// it across the 2.4 GHz band.
//
//	go run ./examples/smith
package main

import (
	"log"
	"math"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/smith"
)

// sweep returns the impedance of a parallel resonance r, l, c in series
// with ls, from 2.3 to 2.6 GHz.
func sweep(r, l, c, ls float64) []smith.Point {
	var pts []smith.Point
	for k := range 61 {
		f := 2.3e9 + float64(k)*5e6
		w := 2 * math.Pi * f
		y := complex(1/r, w*c-1/(w*l))
		pts = append(pts, smith.Point{Freq: f, Z: 1/y + complex(0, w*ls)})
	}
	return pts
}

var traces = []smith.Trace{
	{Name: "Prototype", Points: sweep(110, 0.55e-9, 7.6e-12, 0.9e-9)},
	{Name: "Revision B", Points: sweep(62, 0.6e-9, 7.05e-12, 0.3e-9)},
}

func view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(6)).Gap(t.Space(4)).Children(func() {
		ui.Row(c).Gap(t.Space(5)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "Antenna match").Bold().FontSize(t.FontSize + 4).Grow(1)
			for i, tr := range traces {
				ui.Row(c.Key(i)).Gap(t.Space(1.5)).AlignItems(ui.Center).Children(func() {
					ui.Box(c).Size(t.Space(4), t.Space(1)).Radius(t.Space(0.5)).Background(smith.Color(t, i))
					ui.Text(c, tr.Name).FontSize(t.FontSize - 1)
				})
			}
		})
		smith.View(c, 50, traces, 2).Grow(1)
	})
}

func main() {
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Smith Chart",
			Width:   760,
			Height:  780,
			Content: ui.View(view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
