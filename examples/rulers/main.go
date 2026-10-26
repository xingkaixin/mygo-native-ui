// Rulers shows an artboard on a canvas with rulers: drag from a ruler to
// pull out a guide, drag a guide to move it or back onto its ruler to
// remove it. Cmd-scroll to zoom, scroll or drag to pan.
//
//	go run ./examples/rulers
package main

import (
	"fmt"
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/rulers"
)

const boardW, boardH = 800, 560

type app struct {
	canvas rulers.State
	guides []rulers.Guide
}

func (a *app) artboard(c *ui.Context) {
	ink := ui.Oklch(0.25, 0.03, 260)
	ui.Column(c).Fill().Background(ui.Oklch(0.98, 0.01, 90)).Padding(40).Gap(24).Children(func() {
		ui.Row(c).Height(40).AlignItems(ui.Center).Gap(24).Children(func() {
			ui.Text(c, "Orchard").Bold().FontSize(20).TextColor(ink).Grow(1)
			for _, s := range []string{"Menu", "Visit", "About"} {
				ui.Text(c, s).TextColor(ink.Alpha(0.7))
			}
		})
		ui.Row(c).Grow(1).Gap(32).Children(func() {
			ui.Column(c).Grow(1).Gap(16).Justify(ui.Center).Children(func() {
				ui.Text(c, "Cider pressed this morning, two streets from the trees.").FontSize(36).Bold().TextColor(ink)
				ui.Text(c, "Open Thursday to Sunday, ten until the barrels run dry.").FontSize(16).TextColor(ink.Alpha(0.7))
				ui.Box(c).Size(160, 44).Radius(22).Background(ui.Oklch(0.62, 0.17, 35)).Children(func() {
					ui.Text(c, "Find us").Bold().TextColor(ui.Oklch(1, 0, 0)).Fill().TextAlign(ui.Center).Padding(12, 0)
				})
			})
			ui.Box(c).Size(280, 360).Radius(16).Background(ui.Oklch(0.78, 0.12, 130))
		})
	})
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Children(func() {
		ui.Row(c).Padding(8, 12).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "Landing page").Bold().Grow(1)
			ui.Text(c, fmt.Sprintf("%d guides", len(a.guides))).TextColor(t.TextMuted)
			if ui.Button(c, "Clear guides").Clicked() {
				a.guides = nil
			}
			if ui.Button(c, "−").Label("Zoom out").Clicked() {
				a.canvas.Zoom.ZoomOut()
			}
			ui.Textf(c, "%.0f%%", a.canvas.Zoom.Zoom()*100).FontFeatures("tnum").MinWidth(48).TextAlign(ui.Center)
			if ui.Button(c, "+").Label("Zoom in").Clicked() {
				a.canvas.Zoom.ZoomIn()
			}
			if ui.Button(c, "Fit").Clicked() {
				a.canvas.Zoom.Fit()
			}
		})
		ui.Divider(c)
		rulers.View(c, &a.canvas, boardW, boardH, &a.guides, func() { a.artboard(c) }).Grow(1)
	})
}

func main() {
	a := &app{guides: []rulers.Guide{
		{Vertical: true, Pos: 40}, {Vertical: true, Pos: boardW - 40}, {Pos: 80},
	}}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Rulers",
			Width:   960,
			Height:  680,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
