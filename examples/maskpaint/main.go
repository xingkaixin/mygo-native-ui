// Maskpaint masks part of a drawn lakeside, 640 by 400 pixels, for an
// image model to paint anew: brush over the boat to mask it, erase what
// spilled over, and see beside the picture the mask as the model gets it,
// white where to paint, with the prompt to send.
//
//	go run ./examples/maskpaint
package main

import (
	"image"
	"image/color"
	"log"
	"math"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/maskpaint"
)

const w, h = 640, 400

func mix(a, b color.RGBA, t float64) color.RGBA {
	t = math.Max(0, math.Min(1, t))
	l := func(x, y uint8) uint8 { return uint8(float64(x) + (float64(y)-float64(x))*t) }
	return color.RGBA{l(a.R, b.R), l(a.G, b.G), l(a.B, b.B), 255}
}

// lakeside draws a morning sky, hills, a lake with the hills mirrored in
// it, a shore, and a red boat on the water.
func lakeside() image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	sky, dawn := color.RGBA{120, 170, 220, 255}, color.RGBA{250, 214, 170, 255}
	hill, far := color.RGBA{70, 112, 88, 255}, color.RGBA{128, 156, 150, 255}
	water := color.RGBA{70, 120, 150, 255}
	const lake = 230
	ridge := func(x float64) (near, back float64) {
		back = 150 + 22*math.Sin(x/70) + 10*math.Sin(x/23)
		near = 190 + 18*math.Sin(x/55+1) + 6*math.Sin(x/17)
		return
	}
	for y := range h {
		for x := range w {
			fx, fy := float64(x), float64(y)
			near, back := ridge(fx)
			c := mix(sky, dawn, fy/lake)
			switch {
			case y >= lake+120+int(8*math.Sin(fx/40)):
				c = mix(color.RGBA{214, 196, 150, 255}, color.RGBA{180, 160, 116, 255}, (fy-lake-120)/50)
			case y >= lake:
				m := 2*lake - fy
				c = mix(water, color.RGBA{40, 80, 110, 255}, (fy-lake)/170)
				if m < near && m > back-30 {
					c = mix(c, hill, 0.35)
				}
				if y%9 == 0 && math.Sin(fx/13+fy) > 0.6 {
					c = mix(c, color.RGBA{220, 235, 240, 255}, 0.25)
				}
			case fy > near:
				c = mix(hill, color.RGBA{40, 70, 52, 255}, (fy-near)/50)
			case fy > back:
				c = mix(far, hill, (fy-back)/80)
			}
			img.Set(x, y, c)
		}
	}
	for y := 262; y < 282; y++ {
		inset := (y - 262) / 2
		for x := 380 + inset; x < 470-inset; x++ {
			img.Set(x, y, color.RGBA{196, 64, 52, 255})
		}
	}
	for y := 222; y < 262; y++ {
		for x := 424; x < 427; x++ {
			img.Set(x, y, color.RGBA{60, 44, 36, 255})
		}
		for x := 427; x < 427+(y-222)*3/4; x++ {
			img.Set(x, y, color.RGBA{244, 240, 228, 255})
		}
	}
	return img
}

type app struct {
	painter maskpaint.State
	photo   *ui.Bitmap
	mask    *image.Alpha
	preview *image.Gray
	sent    *ui.Bitmap
	prompt  string
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Row(c).Fill().Background(t.Background).Padding(t.Space(6)).Gap(t.Space(6)).AlignItems(ui.Stretch).Children(func() {
		if maskpaint.View(c, &a.painter, a.photo, a.mask).Grow(1).Changed() {
			copy(a.preview.Pix, a.mask.Pix)
			a.sent.Update(a.preview)
		}
		ui.Column(c).Width(t.Space(56)).Gap(t.Space(3)).Children(func() {
			ui.Text(c, "Mask").Bold()
			sent := a.sent
			ui.Box(c).AspectRatio(float32(w)/float32(h)).Radius(t.Space(1.5)).Clip().Border(1, t.Border).
				Draw(func(p *ui.Painter, r ui.Rect) { p.Image(sent, r, ui.FillBox) })
			ui.Text(c, "Prompt").Bold().Margin(t.Space(2), 0, 0, 0)
			ui.TextArea(c, &a.prompt).Height(t.Space(24))
			if ui.PrimaryButton(c, "Generate").Clicked() {
				c.Toast("Sent the mask and prompt to the model")
			}
		})
	})
}

func main() {
	a := &app{
		photo:   ui.NewBitmap(lakeside()),
		mask:    image.NewAlpha(image.Rect(0, 0, w, h)),
		preview: image.NewGray(image.Rect(0, 0, w, h)),
		prompt:  "A wooden rowboat with two oars, morning light",
	}
	a.sent = ui.NewBitmap(a.preview)
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Maskpaint",
			Width:   1080,
			Height:  640,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
