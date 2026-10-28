// Swatches pulls a palette from a drawn harbour at dusk, or from the
// picture given or dropped on the window: point at a swatch to see where
// its color is, and click it to copy its hex.
//
//	go run ./examples/swatches [image]
package main

import (
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"math"
	"os"
	"path/filepath"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/swatches"
)

func mix(a, b color.RGBA, t float64) color.RGBA {
	t = math.Max(0, math.Min(1, t))
	l := func(x, y uint8) uint8 { return uint8(float64(x) + (float64(y)-float64(x))*t) }
	return color.RGBA{l(a.R, b.R), l(a.G, b.G), l(a.B, b.B), 255}
}

// harbour draws a dusk sky, a low sun, a town on a hill, the sea, and
// boats moored in a row.
func harbour() image.Image {
	const w, h = 480, 320
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	top, dusk, glow := color.RGBA{44, 52, 104, 255}, color.RGBA{214, 106, 112, 255}, color.RGBA{252, 196, 120, 255}
	sea := color.RGBA{36, 62, 92, 255}
	const shore = 200
	for y := range h {
		for x := range w {
			fx, fy := float64(x), float64(y)
			c := mix(top, dusk, fy/150)
			if fy > 120 {
				c = mix(dusk, glow, (fy-120)/80)
			}
			if d := math.Hypot(fx-330, fy-180); d < 26 {
				c = color.RGBA{255, 226, 160, 255}
			}
			hill := 150 + 30*math.Cos((fx-120)/90)
			if fx < 260 && fy > hill {
				c = color.RGBA{58, 48, 70, 255}
				if int(fx)%14 < 9 && int(fy)%12 < 6 && fy < shore-6 {
					c = color.RGBA{238, 188, 108, 255}
				}
			}
			if fy >= shore {
				c = mix(sea, color.RGBA{20, 34, 56, 255}, (fy-shore)/120)
				if math.Abs(fx-330) < 30-(fy-shore)/5 && int(fy)%5 < 2 {
					c = mix(c, glow, 0.6)
				}
			}
			img.Set(x, y, c)
		}
	}
	hulls := []color.RGBA{{226, 84, 70, 255}, {240, 236, 222, 255}, {64, 140, 160, 255}, {240, 236, 222, 255}, {226, 84, 70, 255}}
	for i, hull := range hulls {
		x0 := 40 + i*80
		for y := 232; y < 246; y++ {
			for x := x0 + (y-232)/2; x < x0+56-(y-232)/2; x++ {
				img.Set(x, y, hull)
			}
		}
		for y := 196; y < 232; y++ {
			img.Set(x0+28, y, color.RGBA{30, 30, 40, 255})
		}
	}
	return img
}

type app struct {
	palette swatches.State
	picture image.Image
	caption string
	count   int
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	root := ui.Scroll(c).Fill().Background(t.Background)
	if files := root.DroppedFiles(); len(files) > 0 {
		if f, err := os.Open(files[0]); err == nil {
			if img, _, err := image.Decode(f); err == nil {
				a.picture, a.caption = img, filepath.Base(files[0])
			}
			f.Close()
		}
	}
	root.Children(func() {
		ui.Column(c).Padding(t.Space(8)).Gap(t.Space(5)).MaxWidth(t.Space(180)).Children(func() {
			ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
				ui.Text(c, a.caption).FontSize(t.FontSize + 4).Bold().Grow(1)
				ui.Segmented(c, &a.count, "4 colors", "6 colors", "8 colors").Label("Colors")
			})
			swatches.View(c, &a.palette, a.picture, []int{4, 6, 8}[a.count])
		})
	})
}

func main() {
	a := &app{picture: harbour(), caption: "Harbour at dusk", count: 1}
	if len(os.Args) > 1 {
		f, err := os.Open(os.Args[1])
		if err != nil {
			log.Fatal(err)
		}
		if a.picture, _, err = image.Decode(f); err != nil {
			log.Fatal(err)
		}
		f.Close()
		a.caption = filepath.Base(os.Args[1])
	}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Swatches",
			Width:   820,
			Height:  860,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
