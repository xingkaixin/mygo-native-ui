// Vectorscope grades a picture of a beach at sunset: drag Warmth and
// Saturation and watch the vectorscope swing and stretch, the waveform
// and the RGB parade follow, and point at a scope to read it.
//
//	go run ./examples/vectorscope
package main

import (
	"image"
	"log"
	"math"
	"math/rand/v2"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/vectorscope"
)

const w, h = 640, 420

type app struct {
	photo      *image.RGBA
	graded     *image.RGBA
	preview    *ui.Bitmap
	warmth     float64
	saturation float64
	scopes     vectorscope.State
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Row(c).Fill().Padding(t.Space(6)).Gap(t.Space(6)).Background(t.Background).Children(func() {
		ui.Column(c).Grow(1).Basis(0).Gap(t.Space(4)).Children(func() {
			ui.Image(c, a.preview).Grow(1).Fit(ui.Contain).Radius(t.Space(2))
			changed := false
			ui.Field(c, "Warmth", func() {
				changed = ui.Slider(c, &a.warmth, -1, 1).Changed() || changed
			})
			ui.Field(c, "Saturation", func() {
				changed = ui.Slider(c, &a.saturation, 0, 2).Changed() || changed
			})
			if changed {
				a.grade()
			}
		})
		ui.Column(c).Width(t.Space(100)).AlignSelf(ui.Start).Children(func() {
			vectorscope.View(c, &a.scopes, a.graded)
		})
	})
}

// grade warms or cools the photo and scales its saturation around each
// pixel's luma.
func (a *app) grade() {
	out := image.NewRGBA(a.photo.Bounds())
	for i := 0; i < len(a.photo.Pix); i += 4 {
		r, g, b := float64(a.photo.Pix[i]), float64(a.photo.Pix[i+1]), float64(a.photo.Pix[i+2])
		r, b = r*(1+0.25*a.warmth), b*(1-0.25*a.warmth)
		y := 0.2126*r + 0.7152*g + 0.0722*b
		for k, v := range []float64{r, g, b} {
			out.Pix[i+k] = uint8(max(0, min(255, y+(v-y)*a.saturation)))
		}
		out.Pix[i+3] = 255
	}
	a.graded, a.preview = out, ui.NewBitmap(out)
}

// beach draws a beach at sunset, with a sunbather's warm skin in the
// foreground.
func beach(c *ui.Context) {
	ui.Box(c).Fill().Draw(func(p *ui.Painter, r ui.Rect) {
		p.FillGradient(ui.Rect{W: w, H: 250}, ui.LinearGradient{From: ui.Hex("#3A5BA0"), To: ui.Hex("#F6A15B"), Angle: 180}, 0)
		var sun ui.Path
		p.FillPath(sun.Circle(470, 215, 40), ui.Hex("#FFE3A1"))
		p.FillGradient(ui.Rect{Y: 230, W: w, H: 60}, ui.LinearGradient{From: ui.Hex("#2E6F8E"), To: ui.Hex("#1F4E6B"), Angle: 180}, 0)
		var sand ui.Path
		sand.MoveTo(0, 290)
		for x := float32(0); x <= w; x += 20 {
			sand.LineTo(x, 280+8*float32(math.Sin(float64(x)/70)))
		}
		sand.LineTo(w, h)
		sand.LineTo(0, h)
		sand.Close()
		p.FillPath(&sand, ui.Hex("#D9B98C"))
		p.Fill(ui.Rect{X: 120, Y: 330, W: 210, H: 46}, ui.Hex("#C8323C"), 6)
		p.Fill(ui.Rect{X: 150, Y: 318, W: 150, H: 30}, ui.Hex("#C98B67"), 15)
		var head ui.Path
		p.FillPath(head.Circle(320, 330, 20), ui.Hex("#B97A57"))
		p.Fill(ui.Rect{X: 380, Y: 300, W: 6, H: 90}, ui.Hex("#5B4636"), 2)
		var umbrella ui.Path
		umbrella.MoveTo(318, 304)
		umbrella.QuadTo(383, 230, 448, 304)
		umbrella.Close()
		p.FillPath(&umbrella, ui.Hex("#2BA36B"))
	})
}

// grain adds the noise of a camera's sensor, which spreads each flat color
// into a cloud on the scopes as a real photo's are.
func grain(img *image.RGBA) {
	rng := rand.New(rand.NewPCG(1, 2))
	for i := 0; i < len(img.Pix); i += 4 {
		n := rng.NormFloat64() * 7
		for k := range 3 {
			img.Pix[i+k] = uint8(max(0, min(255, float64(img.Pix[i+k])+n+rng.NormFloat64()*3)))
		}
	}
}

func main() {
	photo := ui.Render(beach, w, h, 1)
	grain(photo)
	a := &app{photo: photo, saturation: 1}
	a.grade()
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Vectorscope",
			Width:   1120,
			Height:  620,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
