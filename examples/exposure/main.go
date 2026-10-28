// Exposure checks a frame of an interview by a window, drawn in code: a
// window that clips, a face, a gray card and a dark sofa. Drag Exposure
// to open up or stop down the frame, and switch between zebras, false
// color and focus peaking.
//
//	go run ./examples/exposure
package main

import (
	"image"
	"log"
	"math"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/exposure"
)

const w, h = 640, 400

type app struct {
	frame    *image.RGBA
	exposed  *image.RGBA
	stops    float64
	exposure exposure.State
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(6)).Gap(t.Space(4)).Children(func() {
		exposure.View(c, &a.exposure, a.exposed).Grow(1)
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "Exposure").TextColor(t.TextMuted)
			if ui.Slider(c, &a.stops, -2, 2).Step(1.0 / 3).Width(t.Space(60)).Label("Exposure, stops").Changed() {
				a.expose()
			}
			ui.Textf(c, "%+.1f stops", a.stops).FontFeatures("tnum")
		})
	})
}

// expose scales the light of the frame by 2 to the power of stops.
func (a *app) expose() {
	gain := math.Exp2(a.stops)
	out := image.NewRGBA(a.frame.Bounds())
	for i := 0; i < len(a.frame.Pix); i += 4 {
		for k := range 3 {
			lin := math.Pow(float64(a.frame.Pix[i+k])/255, 2.2) * gain
			out.Pix[i+k] = uint8(255 * min(1, math.Pow(lin, 1/2.2)))
		}
		out.Pix[i+3] = 255
	}
	a.exposed = out
}

// room draws the frame: a wall, a bright window with its bars, shelves of
// books, a sofa in shadow, a gray card and a face.
func room(c *ui.Context) {
	ui.Box(c).Fill().Draw(func(p *ui.Painter, r ui.Rect) {
		p.FillGradient(ui.Rect{W: w, H: h}, ui.LinearGradient{From: ui.Hex("#5E574D"), To: ui.Hex("#3A352F"), Angle: 90}, 0)
		p.Fill(ui.Rect{X: 30, Y: 40, W: 190, H: 230}, ui.Hex("#FFFFFF"), 0)
		p.FillGradient(ui.Rect{X: 30, Y: 180, W: 190, H: 90}, ui.LinearGradient{From: ui.Hex("#F4F7FA"), To: ui.Hex("#E2E8EE"), Angle: 180}, 0)
		p.Fill(ui.Rect{X: 122, Y: 40, W: 6, H: 230}, ui.Hex("#3B3530"), 0)
		p.Fill(ui.Rect{X: 30, Y: 152, W: 190, H: 6}, ui.Hex("#3B3530"), 0)
		for shelf := range 3 {
			y := float32(60 + shelf*70)
			p.Fill(ui.Rect{X: 440, Y: y + 52, W: 170, H: 6}, ui.Hex("#2C241D"), 0)
			for b := range 14 {
				hue := float32(b*47%360) / 360
				col := ui.Oklch(0.45+0.1*float32(b%3), 0.08, hue*360)
				p.Fill(ui.Rect{X: float32(444 + b*12), Y: y + float32(8+b%4*4), W: 10, H: float32(44 - b%4*4)}, col, 1)
			}
		}
		p.Fill(ui.Rect{X: 0, Y: 300, W: w, H: 100}, ui.Hex("#120F0D"), 0)
		p.Fill(ui.Rect{X: 40, Y: 280, W: 300, H: 70}, ui.Hex("#080707"), 16)
		p.Fill(ui.Rect{X: 360, Y: 250, W: 56, H: 72}, ui.Hex("#777777"), 2)
		p.Fill(ui.Rect{X: 250, Y: 210, W: 120, H: 120}, ui.Hex("#2F3A4A"), 40)
		var face ui.Path
		p.FillPath(face.Circle(310, 160, 52), ui.Hex("#C48A6C"))
		var hair ui.Path
		hair.MoveTo(258, 150).QuadTo(262, 96, 312, 104).QuadTo(360, 100, 362, 150).QuadTo(345, 120, 310, 122).QuadTo(275, 122, 258, 150).Close()
		p.FillPath(&hair, ui.Hex("#2A1C14"))
	})
}

func main() {
	a := &app{frame: ui.Render(room, w, h, 1)}
	a.expose()
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Exposure",
			Width:   880,
			Height:  640,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
