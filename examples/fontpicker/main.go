// Fontpicker sets a type specimen in any font the system has: choose a
// family, its size and weight, and read the specimen in them.
//
//	go run ./examples/fontpicker
package main

import (
	"fmt"
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/fontpicker"
)

var weights = []int{300, 400, 600, 800}

type app struct {
	font   string
	size   float64
	weight int
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(6)).Gap(t.Space(6)).Children(func() {
		ui.Row(c).Gap(t.Space(4)).AlignItems(ui.Center).Children(func() {
			fontpicker.View(c, &a.font).Width(t.Space(64))
			ui.Segmented(c, &a.weight, "Light", "Regular", "Semibold", "Heavy").Label("Weight")
			ui.Slider(c, &a.size, 12, 96).Width(t.Space(40)).Label("Size")
			ui.Text(c, fmt.Sprintf("%.0f pt", a.size)).TextColor(t.TextMuted).FontFeatures("tnum").Width(t.Space(12))
		})
		ui.Scroll(c).Grow(1).Children(func() {
			ui.Column(c).Gap(t.Space(5)).Children(func() {
				specimen := func(s string, size float32) {
					ui.Text(c, s).Font(a.font).FontSize(size).FontWeight(weights[a.weight])
				}
				specimen("Sphinx of black quartz, judge my vow.", float32(a.size))
				specimen("ABCDEFGHIJKLMNOPQRSTUVWXYZ\nabcdefghijklmnopqrstuvwxyz\n0123456789 &@?!%(){}", float32(a.size)*0.6)
				specimen("The quick brown fox jumps over the lazy dog, and the five boxing wizards jump quickly. "+
					"Pack my box with five dozen liquor jugs; how vexingly quick daft zebras jump!", t.FontSize+2)
			})
		})
	})
}

func main() {
	a := &app{font: "Georgia", size: 48, weight: 1}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Font Picker",
			Width:   900,
			Height:  600,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
