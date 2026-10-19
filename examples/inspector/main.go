// Inspector edits a shape on a canvas with an inspector beside it, as
// design apps do: its position, size, rotation and corners, its fill,
// opacity and shadow. Drag the labels of the number fields sideways to
// change them, Shift for ten times faster, Option (Alt) for ten times
// slower.
//
//	go run ./examples/inspector
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/inspector"
)

// shape is the shape the canvas shows and the inspector edits.
type shape struct {
	name                string
	x, y, w, h          float64
	angle, radius       float64
	fill                ui.Color
	opacity             float64
	shadow              bool
	shadowY, shadowBlur float64
}

// app is the state the window shows: the shape, and the sections open.
type app struct {
	shape                     shape
	frame, appearance, effect bool
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	s := &a.shape
	ui.Row(c).Fill().AlignItems(ui.Stretch).Background(t.Background).Children(func() {
		ui.Box(c).Grow(1).Clip().Background(t.Surface).Children(func() {
			box := ui.Box(c).Absolute().Left(float32(s.x)).Top(float32(s.y)).Size(float32(s.w), float32(s.h)).
				Radius(float32(s.radius)).Background(s.fill).Opacity(float32(s.opacity / 100)).Rotate(float32(s.angle))
			if s.shadow {
				box.Shadow(0, float32(s.shadowY), float32(s.shadowBlur), 0, ui.RGBA(0, 0, 0, 0.3))
			}
		})
		ui.Divider(c)
		ui.Scroll(c).Width(280).Children(func() {
			ui.Column(c).Children(func() {
				ui.Column(c).Padding(12).Gap(6).Children(func() {
					ui.Text(c, "Name").FontSize(12).TextColor(t.TextMuted)
					ui.TextInput(c, &s.name).Label("Name")
				})
				ui.Divider(c)
				inspector.Section(c, "Frame", &a.frame, func() {
					inspector.Row(c, "Position", func() {
						inspector.Number(c, "X", &s.x, -2000, 2000, 1)
						inspector.Number(c, "Y", &s.y, -2000, 2000, 1)
					})
					inspector.Row(c, "Size", func() {
						inspector.Number(c, "W", &s.w, 1, 2000, 1)
						inspector.Number(c, "H", &s.h, 1, 2000, 1)
					})
					inspector.Row(c, "Rotation", func() {
						inspector.Number(c, "°", &s.angle, -360, 360, 0.5)
					})
					inspector.Row(c, "Corners", func() {
						inspector.Number(c, "R", &s.radius, 0, 500, 1)
					})
				})
				inspector.Section(c, "Appearance", &a.appearance, func() {
					inspector.Row(c, "Fill", func() { ui.ColorWell(c, &s.fill) })
					inspector.Row(c, "Opacity", func() {
						inspector.Number(c, "%", &s.opacity, 0, 100, 1)
					})
				})
				inspector.Section(c, "Shadow", &a.effect, func() {
					inspector.Row(c, "Show", func() { ui.Switch(c, &s.shadow).Label("Show shadow") })
					inspector.Row(c, "Offset", func() {
						inspector.Number(c, "Y", &s.shadowY, -100, 100, 1)
						inspector.Number(c, "Blur", &s.shadowBlur, 0, 200, 1)
					})
				})
			})
		})
	})
}

func main() {
	a := &app{
		shape: shape{name: "Card", x: 120, y: 110, w: 220, h: 140, radius: 16, fill: ui.Hex("#6366f1"),
			opacity: 100, shadow: true, shadowY: 8, shadowBlur: 24},
		frame: true, appearance: true, effect: true,
	}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Inspector",
			Width:   860,
			Height:  560,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
