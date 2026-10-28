// Snaplayout tiles two windows on a desktop: click a window's green
// button and pick a zone of a layout, and the window comes to the front
// and moves into it.
//
//	go run ./examples/snaplayout
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/snaplayout"
)

type window struct {
	title string
	color ui.Color
	frame ui.Rect
	open  bool
}

type app struct {
	windows []*window
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	desk := ui.Box(c).Fill().LinearGradient(ui.LinearGradient{From: ui.Hex("#3B5B92"), To: ui.Hex("#9A6FB0"), Angle: 135})
	b := desk.LayoutBounds()
	screen := ui.Rect{W: b.W, H: b.H}
	front := -1
	desk.Children(func() {
		for i, w := range a.windows {
			x := desk.AnimateTo([2]any{w.title, "x"}, w.frame.X, ui.Snappy)
			y := desk.AnimateTo([2]any{w.title, "y"}, w.frame.Y, ui.Snappy)
			fw := desk.AnimateTo([2]any{w.title, "w"}, w.frame.W, ui.Snappy)
			fh := desk.AnimateTo([2]any{w.title, "h"}, w.frame.H, ui.Snappy)
			ui.Column(c.Key(w.title)).Absolute().Left(x).Top(y).Size(fw, fh).Radius(t.Space(2.5)).Background(t.Background).Clip().
				Shadow(0, 12, 32, 0, ui.RGBA(0, 0, 0, 0.3)).Children(func() {
				ui.Row(c).Gap(t.Space(2)).Padding(t.Space(2.5), t.Space(3)).AlignItems(ui.Center).Background(t.Surface).Children(func() {
					for _, col := range []string{"#FF5F57", "#FEBC2E"} {
						ui.Box(c.Key(col)).Size(12, 12).Radius(6).Background(ui.Hex(col))
					}
					zoom := ui.ButtonBase(c).Size(12, 12).Radius(6).Background(ui.Hex("#28C840")).Label("Move & Resize")
					if zoom.Clicked() {
						w.open = !w.open
					}
					ui.Popover(c, zoom, &w.open, func() {
						if ev := snaplayout.Picker(c, snaplayout.Layouts); ev.Chosen != nil {
							w.frame, w.open = snaplayout.Frame(screen, *ev.Chosen, 10), false
							front = i
						}
					})
					ui.Text(c, w.title).Bold().FontSize(t.FontSize - 1).Grow(1).TextAlign(ui.Center).MarginEnd(t.Space(12))
				})
				ui.Box(c).Grow(1).Background(w.color.Alpha(0.12))
			})
		}
	})
	if front >= 0 {
		w := a.windows[front]
		a.windows = append(append(a.windows[:front:front], a.windows[front+1:]...), w)
	}
}

func main() {
	a := &app{windows: []*window{
		{title: "Notes", color: ui.Hex("#F5B700"), frame: ui.Rect{X: 60, Y: 50, W: 420, H: 320}},
		{title: "Mail", color: ui.Hex("#0A84FF"), frame: ui.Rect{X: 360, Y: 180, W: 460, H: 340}},
	}}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Snap layouts",
			Width:   960,
			Height:  620,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
