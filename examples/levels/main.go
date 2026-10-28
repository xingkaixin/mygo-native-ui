// Levels brings back a hazy, faded photo of hills at dusk: press Auto, or
// drag the triangles, and the photo beside the panel follows.
//
//	go run ./examples/levels
package main

import (
	"image"
	"log"
	"math"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/levels"
)

const w, h = 640, 420

type app struct {
	photo   *image.RGBA
	preview *ui.Bitmap
	adjust  levels.Adjustment
	levels  levels.State
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Row(c).Fill().Padding(t.Space(6)).Gap(t.Space(6)).Background(t.Background).Children(func() {
		ui.Image(c, a.preview).Grow(1).Fit(ui.Contain).Radius(t.Space(2))
		if levels.View(c, &a.levels, a.photo, &a.adjust).Changed() {
			a.preview = ui.NewBitmap(levels.Apply(a.photo, a.adjust))
		}
	})
}

// hills draws hills under a dusk sky.
func hills(c *ui.Context) {
	ui.Box(c).Fill().Draw(func(p *ui.Painter, r ui.Rect) {
		p.FillGradient(ui.Rect{W: w, H: h}, ui.LinearGradient{From: ui.Hex("#2B4C7E"), To: ui.Hex("#F4A261"), Angle: 180}, 0)
		var sun ui.Path
		sun.Circle(440, 230, 46)
		p.FillPath(&sun, ui.Hex("#FFE8A3"))
		layers := []struct {
			y, amp float32
			col    ui.Color
		}{{240, 30, ui.Hex("#6D597A")}, {290, 40, ui.Hex("#355070")}, {340, 30, ui.Hex("#1D3557")}}
		for k, l := range layers {
			var hill ui.Path
			hill.MoveTo(0, h)
			for x := float32(0); x <= w; x += 20 {
				y := l.y - l.amp*float32(0.5+0.5*math.Sin(float64(x)/90+float64(k)*2))
				hill.LineTo(x, y)
			}
			hill.LineTo(w, h)
			hill.Close()
			p.FillPath(&hill, l.col)
		}
	})
}

// fade flattens a picture toward a warm gray, as haze and an old print do.
func fade(img *image.RGBA) {
	for i := 0; i < len(img.Pix); i += 4 {
		for k, lift := range []int{78, 66, 50} {
			img.Pix[i+k] = uint8(lift + int(img.Pix[i+k])*55/100)
		}
	}
}

func main() {
	photo := ui.Render(hills, w, h, 1)
	fade(photo)
	a := &app{photo: photo, preview: ui.NewBitmap(photo)}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Levels",
			Width:   1040,
			Height:  520,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
