// Bboxlabel labels a street for an object detector: some of its cars and
// people come labeled; drag to box the rest, press 1 to 4 for the class,
// drag a box or its corners to fit it, and Delete one that is wrong. The
// labels show below as a detector's training data takes them.
//
//	go run ./examples/bboxlabel
package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/bboxlabel"
)

type app struct {
	photo   *ui.Bitmap
	classes []bboxlabel.Class
	boxes   []bboxlabel.Box
	labeler bboxlabel.State
}

const w, h = 960, 600

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(5)).Gap(t.Space(3)).Children(func() {
		bboxlabel.View(c, &a.labeler, a.photo, a.classes, &a.boxes).Grow(1)
		var lines []string
		for _, b := range a.boxes {
			// YOLO's format: the class, then the box's center and size as
			// fractions of the picture.
			lines = append(lines, fmt.Sprintf("%d %.3f %.3f %.3f %.3f", b.Class,
				(b.X0+b.X1)/2/w, (b.Y0+b.Y1)/2/h, (b.X1-b.X0)/w, (b.Y1-b.Y0)/h))
		}
		ui.Text(c, strings.Join(lines, "\n")).Font("monospace").FontSize(t.FontSize - 2).TextColor(t.TextMuted).
			Height(t.Space(24)).Selectable()
	})
}

// street draws a street of cars, people, a bicycle and a traffic light.
func street(c *ui.Context) {
	ui.Box(c).Fill().Draw(func(p *ui.Painter, r ui.Rect) {
		p.FillGradient(ui.Rect{W: w, H: 360}, ui.LinearGradient{From: ui.Hex("#8EC5FF"), To: ui.Hex("#DCEBFA"), Angle: 180}, 0)
		for i, b := range []struct{ x, w, h float32 }{{0, 150, 210}, {160, 120, 280}, {290, 180, 190}, {620, 140, 250}, {770, 190, 300}} {
			col := []ui.Color{ui.Hex("#B9A99A"), ui.Hex("#9AA6B2"), ui.Hex("#C7B49B"), ui.Hex("#A3998F"), ui.Hex("#8F9AA6")}[i]
			p.Fill(ui.Rect{X: b.x, Y: 360 - b.h, W: b.w, H: b.h}, col, 0)
			for y := 360 - b.h + 20; y < 330; y += 40 {
				for x := b.x + 16; x < b.x+b.w-24; x += 36 {
					p.Fill(ui.Rect{X: x, Y: y, W: 20, H: 24}, ui.Hex("#E8EEF4").Alpha(0.8), 2)
				}
			}
		}
		p.Fill(ui.Rect{Y: 360, W: w, H: 40}, ui.Hex("#C9C5BE"), 0)
		p.Fill(ui.Rect{Y: 400, W: w, H: 200}, ui.Hex("#55595E"), 0)
		for x := float32(20); x < w; x += 120 {
			p.Fill(ui.Rect{X: x, Y: 496, W: 60, H: 6}, ui.Hex("#F2F2F2"), 1)
		}
		car := func(x, y, s float32, col ui.Color) {
			p.Fill(ui.Rect{X: x, Y: y + 30*s, W: 200 * s, H: 50 * s}, col, 12*s)
			p.Fill(ui.Rect{X: x + 40*s, Y: y, W: 110 * s, H: 45 * s}, col, 16*s)
			p.Fill(ui.Rect{X: x + 52*s, Y: y + 8*s, W: 40 * s, H: 26 * s}, ui.Hex("#CFE3F2"), 4*s)
			p.Fill(ui.Rect{X: x + 98*s, Y: y + 8*s, W: 40 * s, H: 26 * s}, ui.Hex("#CFE3F2"), 4*s)
			for _, wx := range []float32{45, 155} {
				var wheel ui.Path
				wheel.Circle(x+wx*s, y+80*s, 18*s)
				p.FillPath(&wheel, ui.Hex("#1E1E1E"))
			}
		}
		car(60, 420, 1, ui.Hex("#D64545"))
		car(560, 450, 1.2, ui.Hex("#2F6FD6"))
		car(330, 395, 0.7, ui.Hex("#F0B429"))
		person := func(x, y, s float32, shirt ui.Color) {
			var head ui.Path
			head.Circle(x, y, 10*s)
			p.FillPath(&head, ui.Hex("#E0B48F"))
			p.Fill(ui.Rect{X: x - 12*s, Y: y + 12*s, W: 24 * s, H: 40 * s}, shirt, 6*s)
			p.Fill(ui.Rect{X: x - 10*s, Y: y + 50*s, W: 8 * s, H: 34 * s}, ui.Hex("#2B3A55"), 3*s)
			p.Fill(ui.Rect{X: x + 2*s, Y: y + 50*s, W: 8 * s, H: 34 * s}, ui.Hex("#2B3A55"), 3*s)
		}
		person(520, 300, 1, ui.Hex("#3BA55C"))
		person(870, 290, 1.1, ui.Hex("#9B51E0"))
		person(250, 310, 0.9, ui.Hex("#EB5757"))
		bx, by := float32(400), float32(360)
		var frame ui.Path
		frame.MoveTo(bx, by).LineTo(bx+30, by-30).LineTo(bx+60, by).LineTo(bx+25, by).LineTo(bx+30, by-30)
		p.StrokePath(&frame, 3, ui.Hex("#2D9CDB"))
		for _, wx := range []float32{bx, bx + 60} {
			var wheel ui.Path
			wheel.Circle(wx, by, 16)
			p.StrokePath(&wheel, 3, ui.Hex("#222222"))
		}
		p.Fill(ui.Rect{X: 738, Y: 230, W: 6, H: 130}, ui.Hex("#3A3A3A"), 0)
		p.Fill(ui.Rect{X: 726, Y: 170, W: 30, H: 72}, ui.Hex("#2A2A2A"), 5)
		for i, col := range []string{"#E5484D", "#5A4A1A", "#2A4A2A"} {
			var lamp ui.Path
			lamp.Circle(741, 184+float32(i)*22, 8)
			p.FillPath(&lamp, ui.Hex(col))
		}
	})
}

func main() {
	a := &app{
		photo: ui.NewBitmap(ui.Render(street, w, h, 1)),
		classes: []bboxlabel.Class{
			{Name: "Car", Color: ui.Hex("#FF6B6B")},
			{Name: "Person", Color: ui.Hex("#4DABF7")},
			{Name: "Bicycle", Color: ui.Hex("#51CF66")},
			{Name: "Traffic light", Color: ui.Hex("#FCC419")},
		},
		boxes: []bboxlabel.Box{
			{X0: 60, Y0: 420, X1: 260, Y1: 518, Class: 0},
			{X0: 560, Y0: 450, X1: 800, Y1: 568, Class: 0},
			{X0: 508, Y0: 288, X1: 532, Y1: 384, Class: 1},
		},
	}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Box Labeler",
			Width:   1000,
			Height:  800,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
