// Xyplot compares an image model's pictures across four seeds and five
// guidance scales, the cells filling in as each picture comes in: low
// guidance washes the picture out, and high guidance burns it.
//
//	go run ./examples/xyplot
package main

import (
	"fmt"
	"image"
	"log"
	"math"
	"math/rand/v2"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/xyplot"
)

var (
	seeds = []uint64{1234, 52809, 777, 90210}
	cfgs  = []float64{2, 4.5, 7, 10, 14}
)

type app struct {
	plot     xyplot.State
	pictures [][]*ui.Bitmap
	start    time.Time
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	if a.start.IsZero() {
		a.start = c.Now()
	}
	ready := int(c.Now().Sub(a.start) / (150 * time.Millisecond))
	if ready < len(seeds)*len(cfgs) {
		c.After(150 * time.Millisecond)
	}
	xs := xyplot.Axis{Name: "CFG"}
	for _, g := range cfgs {
		xs.Values = append(xs.Values, fmt.Sprint(g))
	}
	ys := xyplot.Axis{Name: "Seed"}
	for _, s := range seeds {
		ys.Values = append(ys.Values, fmt.Sprint(s))
	}
	ui.Column(c).Fill().Padding(t.Space(6)).Background(t.Background).Children(func() {
		xyplot.View(c, &a.plot, xs, ys, func(x, y int) *ui.Bitmap {
			if y*len(cfgs)+x >= ready {
				return nil
			}
			return a.pictures[y][x]
		}).Grow(1)
	})
}

// picture draws the hills that seed makes, washed out or burned by cfg.
func picture(seed uint64, cfg float64) *image.RGBA {
	r := rand.New(rand.NewPCG(seed, seed))
	hue := r.Float32() * 360
	sunX, sunY := 40+r.Float32()*176, 50+r.Float32()*60
	phase := r.Float64() * 6
	img := ui.Render(func(c *ui.Context) {
		ui.Box(c).Fill().Draw(func(p *ui.Painter, b ui.Rect) {
			p.FillGradient(ui.Rect{W: 256, H: 256}, ui.LinearGradient{From: ui.Oklch(0.55, 0.12, hue), To: ui.Oklch(0.9, 0.08, hue+40), Angle: 180}, 0)
			var sun ui.Path
			sun.Circle(sunX, sunY, 22)
			p.FillPath(&sun, ui.Oklch(0.95, 0.1, hue+80))
			for k := range 3 {
				var hill ui.Path
				hill.MoveTo(0, 256)
				for x := float32(0); x <= 256; x += 8 {
					y := 150 + float32(k)*30 - 26*float32(math.Sin(float64(x)/(40+float64(k)*12)+phase*float64(k+1)))
					hill.LineTo(x, y)
				}
				hill.LineTo(256, 256)
				hill.Close()
				p.FillPath(&hill, ui.Oklch(0.5-float32(k)*0.1, 0.1, hue+150+float32(k)*20))
			}
		})
	}, 256, 256, 1)
	// Guidance of about 7 is faithful; lower flattens the colors toward
	// gray, and higher pushes them apart until they clip.
	gain := 0.35 + cfg/7*0.65
	for i := 0; i < len(img.Pix); i += 4 {
		gray := (float64(img.Pix[i]) + float64(img.Pix[i+1]) + float64(img.Pix[i+2])) / 3
		for k := range 3 {
			v := gray + (float64(img.Pix[i+k])-gray)*gain
			v = 128 + (v-128)*(0.7+gain*0.3)
			img.Pix[i+k] = uint8(max(0, min(255, v)))
		}
	}
	return img
}

func main() {
	a := &app{}
	for _, s := range seeds {
		var row []*ui.Bitmap
		for _, g := range cfgs {
			row = append(row, ui.NewBitmap(picture(s, g)))
		}
		a.pictures = append(a.pictures, row)
	}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "XY plot",
			Width:   880,
			Height:  720,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
