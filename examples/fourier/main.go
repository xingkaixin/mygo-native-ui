// Fourier draws a heart, a star and a square with epicycles, as few or as
// many as the slider asks for, or whatever you draw.
//
//	go run ./examples/fourier
package main

import (
	"fmt"
	"log"
	"math"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/fourier"
)

func heart() [][2]float64 {
	var pts [][2]float64
	for i := range 200 {
		a := float64(i) / 200 * 2 * math.Pi
		x := 16 * math.Pow(math.Sin(a), 3)
		y := 13*math.Cos(a) - 5*math.Cos(2*a) - 2*math.Cos(3*a) - math.Cos(4*a)
		pts = append(pts, [2]float64{x / 18, -y/18 - 0.1})
	}
	return pts
}

func star() [][2]float64 {
	var pts [][2]float64
	for i := range 10 {
		r := 0.9
		if i%2 == 1 {
			r = 0.38
		}
		a := float64(i)/10*2*math.Pi - math.Pi/2
		pts = append(pts, [2]float64{r * math.Cos(a), r*math.Sin(a) + 0.08})
	}
	return pts
}

var shapes = []func() [][2]float64{heart, star, func() [][2]float64 {
	return [][2]float64{{-0.75, -0.75}, {0.75, -0.75}, {0.75, 0.75}, {-0.75, 0.75}}
}}

type app struct {
	shape   int
	shown   int
	path    [][2]float64
	terms   float64
	fourier fourier.State
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	if a.shape != a.shown {
		a.shown = a.shape
		if a.shape < len(shapes) {
			a.path = shapes[a.shape]()
		}
	}
	ui.Column(c).Fill().Background(t.Background).Children(func() {
		ui.Row(c).Gap(t.Space(3)).Padding(t.Space(3)).AlignItems(ui.Center).Children(func() {
			ui.Segmented(c, &a.shape, "Heart", "Star", "Square")
			ui.Box(c).Grow(1)
			ui.Text(c, fmt.Sprintf("%d circles", int(a.terms)-1)).TextColor(t.TextMuted).FontFeatures("tnum")
			ui.Slider(c, &a.terms, 2, 256).Width(t.Space(50))
		})
		fourier.View(c, &a.fourier, &a.path, int(a.terms)).Grow(1)
	})
}

func main() {
	a := &app{path: heart(), terms: 30}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Fourier",
			Width:   720,
			Height:  760,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
