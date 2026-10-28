// Ramp tunes the color ramps of a design system in OKLCH: choose a hue,
// drag the points of its curves, and copy the CSS variables below. Fit to
// sRGB lowers the chroma of the steps that screens of sRGB cannot show.
//
//	go run ./examples/ramp
package main

import (
	"fmt"
	"log"
	"math"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/ramp"
)

var names = []string{"50", "100", "200", "300", "400", "500", "600", "700", "800", "900", "950"}

// generate returns a ramp of hue h whose chroma peaks at peak in its
// middle steps, and whose hue turns by turn from the lightest to the
// darkest.
func generate(h, peak, turn float32) []ramp.Step {
	steps := make([]ramp.Step, len(names))
	for i, n := range names {
		f := float32(i) / float32(len(names)-1)
		steps[i] = ramp.Step{
			Name: n,
			L:    0.975 - f*0.71,
			C:    peak * float32(math.Sin(math.Pi*(0.08+0.84*math.Pow(float64(f), 0.8)))),
			H:    h + turn*f,
		}
	}
	return steps
}

type palette struct {
	name  string
	steps []ramp.Step
}

type app struct {
	chosen   int
	palettes []palette
	editor   ramp.State
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	p := &a.palettes[a.chosen]
	ui.Column(c).Fill().Background(t.Surface).Padding(t.Space(6)).Gap(t.Space(5)).Children(func() {
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			labels := make([]string, len(a.palettes))
			for i, p := range a.palettes {
				labels[i] = p.name
			}
			ui.Segmented(c, &a.chosen, labels...).Label("Palette")
			ui.Box(c).Grow(1)
			if ui.Button(c, "Fit to sRGB").Clicked() {
				for i := range p.steps {
					p.steps[i] = p.steps[i].Fit()
				}
			}
		})
		ramp.View(c, &a.editor, &p.steps)
		var css strings.Builder
		for _, s := range p.steps {
			fmt.Fprintf(&css, "--%s-%s: %s;\n", strings.ToLower(p.name), s.Name, s.CSS())
		}
		ui.Text(c, strings.TrimSuffix(css.String(), "\n")).Font("monospace").FontSize(t.FontSize-1).Selectable().
			Padding(t.Space(3)).Radius(t.Radius).Background(t.Background).Border(1, t.Border)
	})
}

func main() {
	a := &app{palettes: []palette{
		{"Blue", generate(258, 0.2, -4)},
		{"Teal", generate(185, 0.13, 8)},
		{"Orange", generate(55, 0.19, -20)},
		{"Grape", generate(305, 0.24, 10)},
	}}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Color Ramps",
			Width:   1000,
			Height:  860,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
