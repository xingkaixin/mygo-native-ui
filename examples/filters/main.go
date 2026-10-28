// Filters adjusts a drawn meadow at noon, or the photo given: move the
// sliders, double-click one to set it back, and choose Split to compare
// the photo before and after across a line you drag.
//
//	go run ./examples/filters [image]
package main

import (
	"image"
	"image/color"
	"log"
	"math"
	"math/rand/v2"
	"os"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/filters"
)

func mix(a, b color.RGBA, t float64) color.RGBA {
	t = math.Max(0, math.Min(1, t))
	l := func(x, y uint8) uint8 { return uint8(float64(x) + (float64(y)-float64(x))*t) }
	return color.RGBA{l(a.R, b.R), l(a.G, b.G), l(a.B, b.B), 255}
}

// meadow draws a sky with a cloud, far mountains, a green meadow strewn
// with flowers, and a tree.
func meadow() image.Image {
	const w, h = 600, 400
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	rnd := rand.New(rand.NewPCG(7, 7))
	for y := range h {
		for x := range w {
			fx, fy := float64(x), float64(y)
			c := mix(color.RGBA{86, 150, 220, 255}, color.RGBA{196, 224, 244, 255}, fy/220)
			if d := math.Hypot((fx-170)/2.2, fy-70); d < 26 || math.Hypot((fx-210)/2, fy-60) < 22 {
				c = color.RGBA{250, 250, 252, 255}
			}
			if m := 190 + 30*math.Sin(fx/60) + 12*math.Sin(fx/17); fy > m {
				c = mix(color.RGBA{120, 140, 170, 255}, color.RGBA{96, 118, 150, 255}, (fy-m)/40)
			}
			if g := 240 + 10*math.Sin(fx/80); fy > g {
				c = mix(color.RGBA{120, 176, 72, 255}, color.RGBA{58, 116, 40, 255}, (fy-g)/160)
			}
			img.Set(x, y, c)
		}
	}
	for range 900 {
		x, y := rnd.IntN(w), 250+rnd.IntN(h-250)
		c := []color.RGBA{{250, 220, 80, 255}, {255, 255, 255, 255}, {230, 90, 110, 255}}[rnd.IntN(3)]
		for dy := range 2 {
			for dx := range 2 {
				img.Set(x+dx, y+dy, c)
			}
		}
	}
	for y := 200; y < 300; y++ {
		for x := 446; x < 458; x++ {
			img.Set(x, y, color.RGBA{92, 64, 44, 255})
		}
	}
	for y := 120; y < 240; y++ {
		for x := 380; x < 530; x++ {
			if math.Hypot(float64(x-452)/1.2, float64(y-180)) < 58 {
				img.Set(x, y, mix(color.RGBA{60, 120, 50, 255}, color.RGBA{34, 80, 32, 255}, float64(y-120)/120))
			}
		}
	}
	return img
}

type app struct {
	adjuster filters.State
	adjust   filters.Adjust
	photo    *ui.Bitmap
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(6)).Children(func() {
		filters.View(c, &a.adjuster, a.photo, &a.adjust).Grow(1)
	})
}

func main() {
	a := &app{adjust: filters.Adjust{Exposure: 0.3, Contrast: 0.15, Warmth: 0.4, Vignette: 0.5}}
	if len(os.Args) > 1 {
		data, err := os.ReadFile(os.Args[1])
		if err != nil {
			log.Fatal(err)
		}
		if a.photo, err = ui.DecodeBitmap(data); err != nil {
			log.Fatal(err)
		}
	} else {
		a.photo = ui.NewBitmap(meadow())
	}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Filters",
			Width:   1000,
			Height:  640,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
