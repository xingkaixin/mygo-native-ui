// Shadows edits the elevation of a card, starting from one of three
// stacks: a soft shadow, a layered one as design systems build them, and
// a pressed card's inner shadow. Drag across the stage to move the light.
//
//	go run ./examples/shadows
package main

import (
	"log"
	"slices"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/shadows"
)

var presets = [][]shadows.Shadow{
	{{Y: 8, Blur: 24, Color: ui.RGBA(15, 23, 42, 0.12)}},
	{
		{Y: 1, Blur: 2, Color: ui.RGBA(15, 23, 42, 0.08)},
		{Y: 4, Blur: 8, Spread: -2, Color: ui.RGBA(15, 23, 42, 0.08)},
		{Y: 16, Blur: 32, Spread: -8, Color: ui.RGBA(15, 23, 42, 0.16)},
	},
	{
		{Y: 2, Blur: 6, Color: ui.RGBA(15, 23, 42, 0.18), Inset: true},
		{Y: -1, Color: ui.RGBA(255, 255, 255, 0.6), Inset: true},
	},
}

type app struct {
	editor  shadows.State
	shadows []shadows.Shadow
	preset  int
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(8)).Gap(t.Space(5)).Children(func() {
		ui.Row(c).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "Card elevation").FontSize(t.FontSize + 6).Bold().Grow(1)
			if ui.Segmented(c, &a.preset, "Soft", "Layered", "Pressed").Label("Preset").Changed() {
				a.shadows = slices.Clone(presets[a.preset])
			}
		})
		shadows.View(c, &a.editor, &a.shadows)
	})
}

func main() {
	a := &app{preset: 1, shadows: slices.Clone(presets[1])}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Shadows",
			Width:   920,
			Height:  640,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
