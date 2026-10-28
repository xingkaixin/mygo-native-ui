// Polygonlabel outlines a lakeside for a segmentation model: the lake and
// the cabin come outlined; click around the rest, press 1 to 4 for the
// class, and drag points to fit them. The labels show below as a
// segmentation model's training data takes them.
//
//	go run ./examples/polygonlabel
package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/polygonlabel"
)

type app struct {
	photo    *ui.Bitmap
	classes  []polygonlabel.Class
	polygons []polygonlabel.Polygon
	labeler  polygonlabel.State
}

const w, h = 960, 600

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(5)).Gap(t.Space(3)).Children(func() {
		polygonlabel.View(c, &a.labeler, a.photo, a.classes, &a.polygons).Grow(1)
		var lines []string
		for _, pg := range a.polygons {
			// YOLO's format for segments: the class, then each point as
			// fractions of the picture.
			line := fmt.Sprint(pg.Class)
			for _, p := range pg.Points {
				line += fmt.Sprintf(" %.3f %.3f", p[0]/w, p[1]/h)
			}
			lines = append(lines, line)
		}
		ui.Text(c, strings.Join(lines, "\n")).Font("monospace").FontSize(t.FontSize - 2).TextColor(t.TextMuted).
			Height(t.Space(20)).Selectable()
	})
}

// lakeside draws a cabin and pines by a lake under mountains.
func lakeside(c *ui.Context) {
	ui.Box(c).Fill().Draw(func(p *ui.Painter, r ui.Rect) {
		p.FillGradient(ui.Rect{W: w, H: 330}, ui.LinearGradient{From: ui.Hex("#7FB7E6"), To: ui.Hex("#E4F0F8"), Angle: 180}, 0)
		var peaks ui.Path
		peaks.MoveTo(0, 330)
		for _, q := range [][2]float32{{0, 230}, {140, 120}, {260, 210}, {400, 90}, {560, 220}, {700, 130}, {840, 200}, {960, 150}, {960, 330}} {
			peaks.LineTo(q[0], q[1])
		}
		peaks.Close()
		p.FillPath(&peaks, ui.Hex("#7D8DA6"))
		p.Fill(ui.Rect{Y: 300, W: w, H: 300}, ui.Hex("#6B9E5E"), 0)
		var lake ui.Path
		lake.MoveTo(330, 380)
		lake.CubeTo(420, 330, 760, 330, 860, 390)
		lake.CubeTo(930, 440, 860, 540, 700, 550)
		lake.CubeTo(520, 565, 340, 520, 320, 460)
		lake.CubeTo(305, 420, 310, 395, 330, 380)
		lake.Close()
		p.FillPath(&lake, ui.Hex("#3E7CB1"))
		p.Fill(ui.Rect{X: 110, Y: 330, W: 150, H: 100}, ui.Hex("#8B5A3C"), 0)
		var roof ui.Path
		roof.MoveTo(95, 335)
		roof.LineTo(185, 270)
		roof.LineTo(275, 335)
		roof.Close()
		p.FillPath(&roof, ui.Hex("#5B3A29"))
		p.Fill(ui.Rect{X: 165, Y: 375, W: 36, H: 55}, ui.Hex("#3B2418"), 0)
		for _, x := range []float32{40, 300, 900, 870} {
			var pine ui.Path
			pine.MoveTo(x, 260)
			pine.LineTo(x+32, 400)
			pine.LineTo(x-32, 400)
			pine.Close()
			p.FillPath(&pine, ui.Hex("#2F5D3A"))
			p.Fill(ui.Rect{X: x - 5, Y: 400, W: 10, H: 22}, ui.Hex("#4A3426"), 0)
		}
		var boat ui.Path
		boat.MoveTo(560, 440)
		boat.LineTo(660, 440)
		boat.LineTo(640, 462)
		boat.LineTo(578, 462)
		boat.Close()
		p.FillPath(&boat, ui.Hex("#D9483B"))
	})
}

func main() {
	a := &app{
		photo: ui.NewBitmap(ui.Render(lakeside, w, h, 1)),
		classes: []polygonlabel.Class{
			{Name: "Water", Color: ui.Hex("#228BE6")},
			{Name: "Building", Color: ui.Hex("#E8590C")},
			{Name: "Tree", Color: ui.Hex("#2F9E44")},
			{Name: "Boat", Color: ui.Hex("#C2255C")},
		},
		polygons: []polygonlabel.Polygon{
			{Class: 0, Points: [][2]float32{{330, 380}, {437, 352}, {591, 344}, {748, 356}, {860, 390}, {885, 451}, {828, 515}, {700, 550}, {568, 551}, {450, 533}, {362, 501}, {320, 460}, {312, 411}}},
			{Class: 1, Points: [][2]float32{{110, 430}, {110, 335}, {95, 335}, {185, 270}, {275, 335}, {260, 335}, {260, 430}}},
		},
	}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Polygon labels",
			Width:   1040,
			Height:  860,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
