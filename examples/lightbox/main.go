// Lightbox shows a gallery of landscapes it paints at startup: click one
// to see it over the window, then Left and Right, or the strip below, for
// the others. Cmd-scroll to zoom, drag to pan, Escape to close.
//
//	go run ./examples/lightbox
package main

import (
	"image"
	"image/color"
	"log"
	"math"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/lightbox"
)

// scene is a landscape to paint: its sky from top to horizon, its sun,
// and its ridges from far to near.
type scene struct {
	caption       string
	top, horizon  color.RGBA
	sun           color.RGBA
	sunX, sunY    float64
	far, mid, low color.RGBA
	seed          float64
}

var scenes = []scene{
	{"Dawn over the eastern ridge", rgb(0x2E3A6B), rgb(0xF6B48F), rgb(0xFFE3B0), 0.7, 0.62, rgb(0x6C5A86), rgb(0x47406A), rgb(0x272744), 1},
	{"Noon above the valley", rgb(0x3F8EDB), rgb(0xBFE3F7), rgb(0xFFFBEA), 0.3, 0.18, rgb(0x7FA7B8), rgb(0x4F7F6A), rgb(0x2F5A3D), 2},
	{"Late afternoon, west face", rgb(0x6EA4D8), rgb(0xF4D9A6), rgb(0xFFF1C9), 0.82, 0.4, rgb(0xA79A8E), rgb(0x7A6A55), rgb(0x4A3D2E), 3},
	{"Dusk from the saddle", rgb(0x3B2A5A), rgb(0xF07C5A), rgb(0xFFC48A), 0.45, 0.66, rgb(0x8A4E6E), rgb(0x5A3253), rgb(0x2A1B34), 4},
	{"Moonrise in the high pass", rgb(0x0B1530), rgb(0x2D3F6E), rgb(0xE8ECF5), 0.25, 0.3, rgb(0x34466E), rgb(0x22304F), rgb(0x121A2E), 5},
	{"Fog in the lower meadows", rgb(0xA9B4BE), rgb(0xE7E9EA), rgb(0xF5F5F2), 0.6, 0.35, rgb(0xC5CBD0), rgb(0x9BA6AE), rgb(0x6F7D86), 6},
	{"First snow on the north ridge", rgb(0x8DB5E0), rgb(0xE9F1F8), rgb(0xFFFFFF), 0.15, 0.22, rgb(0xDCE6EF), rgb(0xAFC2D3), rgb(0x6B829A), 7},
	{"Storm light over the lake", rgb(0x3C4652), rgb(0xC9B88A), rgb(0xFFF0C0), 0.55, 0.5, rgb(0x5E6670), rgb(0x3F4750), rgb(0x22282F), 8},
}

func rgb(v uint32) color.RGBA {
	return color.RGBA{uint8(v >> 16), uint8(v >> 8), uint8(v), 255}
}

func mix(a, b color.RGBA, t float64) color.RGBA {
	m := func(x, y uint8) uint8 { return uint8(float64(x)*(1-t) + float64(y)*t) }
	return color.RGBA{m(a.R, b.R), m(a.G, b.G), m(a.B, b.B), 255}
}

// ridge returns the height of a ridge at x, from 0 to 1, rough by seed.
func ridge(x, seed, base, rough float64) float64 {
	y := base
	for k := 1.0; k <= 5; k++ {
		y += rough / k * math.Sin(x*k*3.1+seed*k*1.7+math.Cos(x*k*1.3+seed))
	}
	return y
}

func paint(s scene, w, h int) image.Image {
	m := image.NewRGBA(image.Rect(0, 0, w, h))
	horizon := 0.72
	for py := range h {
		for px := range w {
			x, y := float64(px)/float64(w), float64(py)/float64(h)
			c := mix(s.top, s.horizon, math.Min(1, y/horizon))
			dx, dy := (x-s.sunX)*float64(w)/float64(h), y-s.sunY
			if d := math.Hypot(dx, dy); d < 0.06 {
				c = s.sun
			} else if d < 0.25 {
				c = mix(c, s.sun, 0.35*(1-(d-0.06)/0.19))
			}
			for i, r := range []struct {
				col         color.RGBA
				base, rough float64
			}{{s.far, 0.55, 0.06}, {s.mid, 0.66, 0.05}, {s.low, 0.8, 0.04}} {
				if y > ridge(x, s.seed+float64(i)*2.3, r.base, r.rough) {
					c = r.col
				}
			}
			m.SetRGBA(px, py, c)
		}
	}
	return m
}

type app struct {
	photos []lightbox.Photo
	box    lightbox.State
	open   bool
	index  int
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(6)).Gap(t.Space(4)).Children(func() {
		ui.Text(c, "Ridge walk").Bold().FontSize(t.FontSize + 4)
		ui.Grid(c).Columns(4).Gap(t.Space(3)).Children(func() {
			for i, p := range a.photos {
				thumb := ui.Column(c.Key(i)).Gap(t.Space(1.5)).Cursor(ui.CursorPointer).Label(p.Caption)
				thumb.Children(func() {
					ui.Box(c).Height(120).Radius(t.Radius).Clip().Children(func() {
						ui.Image(c, p.Image).Fit(ui.Cover).Fill()
					})
					ui.Text(c, p.Caption).FontSize(t.FontSize - 1).TextColor(t.TextMuted).SingleLine()
				})
				if thumb.Clicked() {
					a.open, a.index = true, i
				}
			}
		})
	})
	lightbox.Show(c, &a.box, &a.open, &a.index, a.photos)
}

func newApp() *app {
	a := &app{}
	for _, s := range scenes {
		a.photos = append(a.photos, lightbox.Photo{Image: ui.NewBitmap(paint(s, 1500, 1000)), Caption: s.caption})
	}
	return a
}

func main() {
	a := newApp()
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Lightbox",
			Width:   960,
			Height:  640,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
