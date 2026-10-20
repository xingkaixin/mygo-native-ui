// Pentool draws paths with a pen: a leaf to start from, and its SVG on
// the right, to copy.
//
//	go run ./examples/pentool
package main

import (
	"fmt"
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/pentool"
)

func leaf() []pentool.Path {
	return []pentool.Path{
		{Closed: true, Anchors: []pentool.Anchor{
			{X: 120, Y: 300, InX: 10, InY: 60, OutX: -10, OutY: -110, Corner: true},
			{X: 300, Y: 120, InX: -120, InY: -10, OutX: 10, OutY: 120, Corner: true},
		}},
		{Anchors: []pentool.Anchor{
			{X: 90, Y: 330, Corner: true},
			{X: 230, Y: 190, Corner: true},
		}},
	}
}

var hints = []string{
	"Click to add a corner",
	"Drag to pull out a curve, Alt for one handle",
	"Click the first point to close a path",
	"Escape or Enter ends a path open",
	"Drag a point to move it, its handles to bend it",
	"Delete removes the point chosen",
	"Double-click toggles corner and smooth",
}

type app struct {
	pen    pentool.State
	paths  []pentool.Path
	copied bool
}

func (a *app) svg() string {
	return fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 480 480"><path d="%s" fill="#cde8c4" stroke="#2f5d27" stroke-width="3"/></svg>`, pentool.SVG(a.paths))
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Row(c).Fill().Background(t.Background).Padding(t.Space(6)).Gap(t.Space(6)).AlignItems(ui.Stretch).Children(func() {
		if pentool.View(c, &a.pen, &a.paths).Size(480, 480).Changed() {
			a.copied = false
		}
		ui.Column(c).Grow(1).Shrink(1).Gap(t.Space(4)).Children(func() {
			ui.Text(c, "Pen").Bold().FontSize(t.FontSize + 6)
			ui.Column(c).Gap(t.Space(1)).Children(func() {
				for _, h := range hints {
					ui.Text(c, h).FontSize(t.FontSize - 1).TextColor(t.TextMuted)
				}
			})
			ui.Text(c, pentool.SVG(a.paths)).Font("monospace").FontSize(t.FontSize-2).
				Padding(t.Space(3)).Background(t.Surface).Border(1, t.Border).Radius(t.Space(1.5)).AlignSelf(ui.Stretch)
			ui.Row(c).Gap(t.Space(2)).Children(func() {
				label := "Copy SVG"
				if a.copied {
					label = "Copied"
				}
				if ui.Button(c, label).Clicked() {
					c.WriteClipboard(a.svg())
					a.copied = true
				}
				if ui.Button(c, "Clear").Clicked() {
					a.paths, a.pen, a.copied = nil, pentool.State{}, false
				}
			})
		})
	})
}

func main() {
	a := &app{paths: leaf()}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Pen",
			Width:   900,
			Height:  540,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
