// Curves grades a photo with tone curves: drag across the picture to
// compare it before and after. Give it a photo to grade one of your own.
//
//	go run ./examples/curves [photo.jpg]
package main

import (
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"math"
	"os"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/curves"
)

// dusk paints a flat, low-contrast landscape at dusk, the kind of picture
// a curve brings to life.
func dusk(w, h int) image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	mix := func(a, b [3]float64, t float64) [3]float64 {
		t = max(0, min(1, t))
		return [3]float64{a[0] + (b[0]-a[0])*t, a[1] + (b[1]-a[1])*t, a[2] + (b[2]-a[2])*t}
	}
	ridge := func(x float64, k int) float64 {
		f := float64(k + 1)
		return 0.45 + 0.08*f + 0.05*math.Sin(x*7/f+f*2) + 0.03*math.Sin(x*19/f+f)
	}
	for y := range h {
		for x := range w {
			u, v := float64(x)/float64(w), float64(y)/float64(h)
			px := mix([3]float64{96, 108, 140}, [3]float64{196, 160, 130}, v/0.6)
			if d := math.Hypot(u-0.7, v-0.42); d < 0.06 {
				px = mix([3]float64{225, 205, 170}, px, d/0.06)
			}
			for k := range 3 {
				if v > ridge(u, k) {
					l := 70 - float64(k)*18
					px = [3]float64{l + 20, l + 18, l + 30}
				}
			}
			if v > 0.82 {
				px = mix([3]float64{120, 110, 115}, [3]float64{60, 58, 70}, (v-0.82)/0.18)
				px = mix(px, [3]float64{200, 180, 150}, 0.25*math.Max(0, 1-math.Abs(u-0.7)*8)*(0.5+0.5*math.Sin(v*300)))
			}
			n := float64((x*7919+y*104729)%23) - 11
			img.Set(x, y, color.NRGBA{uint8(max(0, min(255, px[0]+n))), uint8(max(0, min(255, px[1]+n))), uint8(max(0, min(255, px[2]+n))), 255})
		}
	}
	return img
}

type app struct {
	pic    image.Image
	before *ui.Bitmap
	after  *ui.Bitmap
	tone   curves.State
	curves curves.Curves
	split  float32
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Row(c).Fill().Background(t.Background).Padding(t.Space(6)).Gap(t.Space(6)).AlignItems(ui.Start).Children(func() {
		b := a.pic.Bounds()
		w := float32(620)
		h := w * float32(b.Dy()) / float32(b.Dx())
		photo := ui.Box(c).Size(w, h).Radius(t.Space(2)).Clip().Cursor(ui.CursorResizeEW).Label("Compare")
		if x, _, ok := photo.PointerPosition(); ok && photo.Pressed() {
			a.split = max(0, min(1, x/w))
		}
		photo.Draw(func(p *ui.Painter, r ui.Rect) {
			cut := r.X + r.W*a.split
			p.Clip(ui.Rect{X: r.X, Y: r.Y, W: cut - r.X, H: r.H}, 0, func() { p.Image(a.before, r, ui.FillBox) })
			p.Clip(ui.Rect{X: cut, Y: r.Y, W: r.X + r.W - cut, H: r.H}, 0, func() { p.Image(a.after, r, ui.FillBox) })
			p.Line(cut, r.Y, cut, r.Y+r.H, 2, ui.RGB(255, 255, 255))
			white := ui.RGB(255, 255, 255)
			if a.split > 0.1 {
				p.Text(r.X+10, r.Y+8, "Before", t.FontSize-1, white)
			}
			if a.split < 0.9 {
				p.Text(r.X+r.W-52, r.Y+8, "After", t.FontSize-1, white)
			}
		})
		ui.Column(c).Gap(t.Space(4)).Children(func() {
			ui.Text(c, "Curves").Bold().FontSize(t.FontSize + 6)
			if curves.View(c, &a.tone, a.pic, &a.curves).Changed() {
				a.after = ui.NewBitmap(curves.Apply(a.pic, a.curves))
			}
			if ui.Button(c, "S-curve").AlignSelf(ui.Start).Clicked() {
				a.curves.RGB = curves.Curve{{X: 0, Y: 0}, {X: 0.25, Y: 0.15}, {X: 0.75, Y: 0.88}, {X: 1, Y: 1}}
				a.after = ui.NewBitmap(curves.Apply(a.pic, a.curves))
			}
		})
	})
}

func main() {
	a := &app{split: 0.5}
	if len(os.Args) > 1 {
		f, err := os.Open(os.Args[1])
		if err != nil {
			log.Fatal(err)
		}
		a.pic, _, err = image.Decode(f)
		f.Close()
		if err != nil {
			log.Fatal(err)
		}
	} else {
		a.pic = dusk(1240, 820)
	}
	a.before = ui.NewBitmap(a.pic)
	a.after = a.before
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Curves",
			Width:   1000,
			Height:  520,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
