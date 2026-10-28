// Elevation shows the profile of a 48 km ride over two cols, colored by
// grade, with the climbs marked. Point at it for the elevation there, and
// drag across it to measure a stretch.
//
//	go run ./examples/elevation
package main

import (
	"fmt"
	"log"
	"math"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/elevation"
)

// route rises and falls as the sum of a few hills: where each tops out,
// how high, and how wide.
func route() []elevation.Point {
	hills := []struct{ at, height, width float64 }{
		{9000, 120, 2500},
		{19500, 520, 4200},
		{28000, 90, 1800},
		{37500, 380, 3000},
		{43500, 60, 1500},
	}
	var pts []elevation.Point
	for d := 0.0; d <= 48000; d += 100 {
		e := 210 + d/48000*40 + 6*math.Sin(d/700)
		for _, h := range hills {
			x := (d - h.at) / h.width
			e += h.height * math.Exp(-x*x)
		}
		pts = append(pts, elevation.Point{Dist: d, Elev: e})
	}
	return pts
}

var pts = route()

type app struct {
	profile elevation.State
	at      float64
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(8)).Gap(t.Space(4)).Children(func() {
		ui.Row(c).AlignItems(ui.Baseline).Gap(t.Space(3)).Children(func() {
			ui.Text(c, "Two Cols Loop").FontSize(t.FontSize + 8).Bold()
			ui.Text(c, "48 km").TextColor(t.TextMuted)
		})
		ev := elevation.View(c, &a.profile, pts)
		ev.Height(t.Space(70))
		if ev.At >= 0 {
			a.at = ev.At
		}
		ui.Text(c, fmt.Sprintf("Marker on the map at %.1f km", a.at/1000)).FontSize(t.FontSize - 1).TextColor(t.TextMuted)
	})
}

func main() {
	a := &app{}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Elevation",
			Width:   960,
			Height:  440,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
