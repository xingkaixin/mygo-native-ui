// Meshgradient designs the wallpaper of a phone's lock screen: drag the
// points, click one to change its color, start from a preset, and copy
// the mesh as SwiftUI code. The phone on the right paints the same mesh.
//
//	go run ./examples/meshgradient
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/meshgradient"
)

var presets = []struct {
	name       string
	cols, rows int
	colors     []string
}{
	{"Sunset", 3, 3, []string{"#2b1055", "#7a2b8c", "#d5487a", "#5b2a86", "#f0626f", "#ff9f68", "#ff7f50", "#ffc46b", "#ffe8a3"}},
	{"Lagoon", 3, 4, []string{"#03396c", "#005b96", "#6497b1", "#0a7e8c", "#1fb5a8", "#b3cde0", "#29c7ac", "#7fe0c4", "#e0f7f1", "#c1f0e4", "#f6fff9", "#ffffff"}},
	{"Aurora", 4, 3, []string{"#0b132b", "#1c2541", "#3a506b", "#0b132b", "#2bd9a6", "#5bc0be", "#6fffe9", "#7b5cff", "#1c2541", "#3a506b", "#a06cd5", "#0b132b"}},
}

func preset(i int) meshgradient.Mesh {
	p := presets[i]
	colors := make([]ui.Color, len(p.colors))
	for i, h := range p.colors {
		colors[i] = ui.Hex(h)
	}
	return meshgradient.New(p.cols, p.rows, colors)
}

type app struct {
	mesh   meshgradient.Mesh
	editor meshgradient.State
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Row(c).Fill().Background(t.Background).Padding(t.Space(6)).Gap(t.Space(6)).Children(func() {
		ui.Column(c).Grow(1).FillHeight().Gap(t.Space(4)).Children(func() {
			ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
				for i, p := range presets {
					if ui.Button(c.Key(i), p.name).Clicked() {
						a.mesh, a.editor = preset(i), meshgradient.State{}
					}
				}
				ui.Spacer(c)
				if ui.Button(c, "Copy SwiftUI").Clicked() {
					c.WriteClipboard(a.mesh.SwiftUI())
					c.Toast("Copied the MeshGradient")
				}
			})
			meshgradient.View(c, &a.editor, &a.mesh).Grow(1)
		})
		mesh := a.mesh
		ui.Column(c).Size(t.Space(70), t.Space(140)).AlignSelf(ui.Center).Radius(t.Space(9)).Clip().
			Border(6, ui.RGB(20, 20, 22)).Padding(t.Space(10), t.Space(4)).Gap(t.Space(1)).AlignItems(ui.Center).
			Draw(func(p *ui.Painter, r ui.Rect) {
				// Clip clips the children, not what Draw paints.
				p.Clip(r, t.Space(9), func() { meshgradient.Paint(p, r, mesh) })
			}).Children(func() {
			ui.Text(c, "Tuesday, 27 October").TextColor(ui.RGBA(255, 255, 255, 0.85)).Bold()
			ui.Text(c, "9:41").FontSize(64).FontWeight(600).TextColor(ui.RGB(255, 255, 255))
		})
	})
}

func main() {
	a := &app{mesh: preset(0)}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Mesh gradient",
			Width:   1000,
			Height:  680,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
