// Crop crops a picture, as a photo editor does, to a free shape or an
// aspect ratio, the part kept shown on the right at its size. Without a
// picture, it paints one of mountains at dusk.
//
//	go run ./examples/crop [picture.png|.jpg]
package main

import (
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"math"
	"os"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/crop"
)

// aspects are the shapes to crop to, 0 for any and -1 for the picture's.
var aspects = []struct {
	name  string
	ratio float32
}{{"Free", 0}, {"Original", -1}, {"1:1", 1}, {"4:3", 4.0 / 3}, {"16:9", 16.0 / 9}}

type app struct {
	picture *ui.Bitmap
	area    ui.Rect
	aspect  int
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	w, h := a.picture.Size()
	ratio := aspects[a.aspect].ratio
	if ratio < 0 {
		ratio = float32(w) / float32(h)
	}
	ui.Row(c).Fill().Background(t.Background).Children(func() {
		crop.View(c, a.picture, &a.area, ratio).Grow(1).AlignSelf(ui.Stretch).Background(t.Surface)
		ui.Divider(c).Vertical()
		ui.Column(c).Width(t.Space(84)).AlignSelf(ui.Stretch).Gap(t.Space(4)).Padding(t.Space(5)).Children(func() {
			ui.Text(c, "Aspect").Bold()
			names := make([]string, len(aspects))
			for i, as := range aspects {
				names[i] = as.name
			}
			ui.Segmented(c, &a.aspect, names...)
			ui.Text(c, "Kept").Bold()
			ar := a.area
			ui.Box(c).FillWidth().AspectRatio(ar.W / max(1, ar.H)).Radius(t.Space(1)).Clip().
				Draw(func(p *ui.Painter, r ui.Rect) {
					p.ImageRegion(a.picture, ar, r, ui.ImageOptions{})
				})
			ui.Text(c, fmt.Sprintf("%d × %d px", int(math.Round(float64(ar.W))), int(math.Round(float64(ar.H))))).
				FontFeatures("tnum")
			ui.Text(c, fmt.Sprintf("from %d, %d", int(ar.X), int(ar.Y))).FontFeatures("tnum").TextColor(t.TextMuted)
			ui.Box(c).Grow(1)
			if ui.Button(c, "Reset").Clicked() {
				a.area = ui.Rect{}
			}
		})
	})
}

// dusk paints mountains at dusk, w by h pixels.
func dusk(w, h int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	mix := func(a, b [3]float64, f float64) [3]float64 {
		return [3]float64{a[0] + (b[0]-a[0])*f, a[1] + (b[1]-a[1])*f, a[2] + (b[2]-a[2])*f}
	}
	top, horizon := [3]float64{38, 42, 110}, [3]float64{255, 160, 110}
	ranges := []struct {
		base, amp, freq float64
		color           [3]float64
	}{
		{0.58, 0.08, 3.1, [3]float64{120, 88, 140}},
		{0.66, 0.07, 5.3, [3]float64{78, 58, 110}},
		{0.76, 0.06, 8.7, [3]float64{42, 34, 70}},
	}
	sx, sy, sr := 0.68*float64(w), 0.52*float64(h), 0.06*float64(h)
	for y := range h {
		fy := float64(y) / float64(h)
		for x := range w {
			fx := float64(x) / float64(w)
			c := mix(top, horizon, math.Min(1, fy/0.6))
			if d := math.Hypot(float64(x)-sx, float64(y)-sy); d < sr {
				c = [3]float64{255, 226, 170}
			} else if d < 4*sr {
				c = mix(c, [3]float64{255, 210, 160}, 0.35*(1-(d-sr)/(3*sr)))
			}
			for _, r := range ranges {
				ridge := r.base - r.amp*(0.6*math.Sin(fx*r.freq*math.Pi+1.3)+0.4*math.Sin(fx*r.freq*2.7+0.4))
				if fy > ridge {
					c = mix(r.color, [3]float64{20, 18, 40}, (fy-ridge)*1.2)
				}
			}
			img.Set(x, y, color.RGBA{uint8(c[0]), uint8(c[1]), uint8(c[2]), 255})
		}
	}
	return img
}

func main() {
	var picture *ui.Bitmap
	if len(os.Args) > 1 {
		data, err := os.ReadFile(os.Args[1])
		if err != nil {
			log.Fatal(err)
		}
		if picture, err = ui.DecodeBitmap(data); err != nil {
			log.Fatal(err)
		}
	} else {
		picture = ui.NewBitmap(dusk(1600, 1000))
	}
	a := &app{picture: picture}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Crop",
			Width:   1120,
			Height:  700,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
