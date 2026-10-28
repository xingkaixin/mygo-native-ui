// Ternary places made-up soil samples from three fields on the USDA soil
// texture triangle: a river meadow of silty soils, a hillside of loams,
// and a sandy plot by the dunes. Hover a sample for its parts and class,
// or a class to light it.
//
//	go run ./examples/ternary
package main

import (
	"fmt"
	"log"
	"math/rand/v2"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/ternary"
)

var fields = []struct {
	name             string
	clay, sand, silt float64
	hue              float32
}{
	{"Meadow", 22, 15, 63, 250},
	{"Hillside", 24, 40, 36, 145},
	{"Dunes", 6, 82, 12, 60},
}

func samples() []ternary.Point {
	rnd := rand.New(rand.NewPCG(4, 4))
	var out []ternary.Point
	for _, f := range fields {
		for i := range 10 {
			part := func(v float64) float64 { return max(0.5, v+rnd.NormFloat64()*4) }
			out = append(out, ternary.Point{
				Label: fmt.Sprintf("%s %d", f.name, i+1),
				A:     part(f.clay), B: part(f.sand), C: part(f.silt),
				Color: ui.Oklch(0.6, 0.15, f.hue),
			})
		}
	}
	return out
}

var points = samples()

func view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(6)).Gap(t.Space(3)).Children(func() {
		ui.Row(c).Gap(t.Space(4)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "Soil texture of three fields").Bold().FontSize(t.FontSize + 4).Grow(1)
			for _, f := range fields {
				ui.Row(c.Key(f.name)).Gap(t.Space(1.5)).AlignItems(ui.Center).Children(func() {
					ui.Box(c).Size(t.Space(2.5), t.Space(2.5)).Radius(t.Space(1.25)).Background(ui.Oklch(0.6, 0.15, f.hue))
					ui.Text(c, f.name)
				})
			}
		})
		ternary.View(c, ternary.Axes{A: "Clay", B: "Sand", C: "Silt"}, ternary.SoilTexture, points).Grow(1)
	})
}

func main() {
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Ternary plot",
			Width:   900,
			Height:  820,
			Content: ui.View(view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
