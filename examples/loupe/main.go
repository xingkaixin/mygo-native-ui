// Loupe picks colors from a small drawn landscape, 160 by 100 pixels:
// move over it to see the pixels under the loupe, click to keep a color,
// and use the arrows for one pixel at a time. The colors kept line up on
// the right.
//
//	go run ./examples/loupe
package main

import (
	"image"
	"image/color"
	"log"
	"math"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/loupe"
)

func mix(a, b color.RGBA, t float64) color.RGBA {
	l := func(x, y uint8) uint8 { return uint8(float64(x) + (float64(y)-float64(x))*t) }
	return color.RGBA{l(a.R, b.R), l(a.G, b.G), l(a.B, b.B), 255}
}

// landscape draws a sunset: a sky in bands, a sun, two ranges of hills
// and a lake.
func landscape() image.Image {
	const w, h = 160, 100
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	top, low := color.RGBA{38, 41, 102, 255}, color.RGBA{250, 140, 90, 255}
	for y := range h {
		band := float64(y/6*6) / 60
		for x := range w {
			img.Set(x, y, mix(top, low, math.Min(band, 1)))
		}
	}
	for y := range h {
		for x := range w {
			if dx, dy := float64(x-104), float64(y-58); dx*dx+dy*dy < 15*15 {
				img.Set(x, y, color.RGBA{255, 214, 102, 255})
			}
		}
	}
	for x := range w {
		far := 62 + int(8*math.Sin(float64(x)/13)+4*math.Sin(float64(x)/5))
		near := 72 + int(6*math.Sin(float64(x)/9+2))
		for y := far; y < h; y++ {
			img.Set(x, y, color.RGBA{88, 62, 112, 255})
		}
		for y := near; y < h; y++ {
			img.Set(x, y, color.RGBA{46, 38, 72, 255})
		}
		for y := 84; y < h; y++ {
			if (x+y)%7 == 0 {
				img.Set(x, y, color.RGBA{250, 160, 110, 255})
			} else {
				img.Set(x, y, color.RGBA{60, 70, 130, 255})
			}
		}
	}
	return img
}

type app struct {
	loupe loupe.State
	img   image.Image
	kept  []ui.Color
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Row(c).Fill().Background(t.Background).Padding(t.Space(6)).Gap(t.Space(6)).AlignItems(ui.Start).Children(func() {
		ev := loupe.View(c, &a.loupe, a.img)
		ev.Grow(1).FillHeight().Radius(t.Radius).Background(t.Surface)
		if ev.Picked {
			a.kept = append([]ui.Color{ev.Color}, a.kept...)
		}
		ui.Column(c).Width(t.Space(44)).Gap(t.Space(2)).Children(func() {
			ui.Text(c, "Kept").Bold()
			if len(a.kept) == 0 {
				ui.Text(c, "Click the image to keep a color").TextColor(t.TextMuted).Wrap()
			}
			for i, k := range a.kept {
				ui.Row(c.Key(i)).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
					ui.Box(c).Size(t.Space(6), t.Space(6)).Radius(t.Space(1.5)).Background(k).Border(1, t.Border)
					ui.Text(c, loupe.Hex(k)).Font("monospace")
				})
			}
		})
	})
}

func main() {
	a := &app{img: landscape()}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Loupe",
			Width:   900,
			Height:  560,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
