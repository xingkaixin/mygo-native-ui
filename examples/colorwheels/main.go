// Colorwheels grades a flat picture with lift, gamma and gain wheels,
// grading it in the background as they turn, and shows it before or
// after. Give it a picture to grade one of your own.
//
//	go run ./examples/colorwheels [picture]
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

	"github.com/xingkaixin/mygo-native-ui/ui/colorwheels"
)

// lake paints a lake under mountains as a camera records it flat, with
// lifted shadows, dull highlights and a green cast, for the wheels to
// grade.
func lake(w, h int) image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	mix := func(a, b [3]float64, t float64) [3]float64 {
		t = max(0, min(1, t))
		return [3]float64{a[0] + (b[0]-a[0])*t, a[1] + (b[1]-a[1])*t, a[2] + (b[2]-a[2])*t}
	}
	far := func(u float64) float64 { return 0.40 + 0.06*math.Sin(u*5+1) + 0.025*math.Sin(u*17) }
	near := func(u float64) float64 { return 0.50 + 0.04*math.Sin(u*7+2) + 0.015*math.Sin(u*29) }
	shore := func(u float64) float64 { return 0.80 + 0.015*math.Sin(u*11) }
	tree := func(u, v float64) bool {
		for k, x := range []float64{0.06, 0.11, 0.15, 0.82, 0.88, 0.93} {
			ht, wd := 0.16+0.05*float64(k%3), 0.035
			top := shore(x) - ht
			if v > top && v < shore(x)+0.02 && math.Abs(u-x) < wd*(v-top)/ht {
				return true
			}
		}
		return false
	}
	sky := func(u, v float64) [3]float64 {
		px := mix([3]float64{128, 146, 160}, [3]float64{206, 204, 190}, v/0.55)
		if d := math.Hypot((u-0.62)*1.5, v-0.3); d < 0.25 {
			px = mix([3]float64{226, 222, 204}, px, d/0.25)
		}
		return px
	}
	for y := range h {
		for x := range w {
			u, v := float64(x)/float64(w), float64(y)/float64(h)
			px := sky(u, v)
			if v > far(u) {
				px = mix([3]float64{132, 142, 146}, [3]float64{112, 122, 122}, (v-far(u))/0.15)
			}
			if v > near(u) {
				px = mix([3]float64{92, 104, 96}, [3]float64{74, 86, 76}, (v-near(u))/0.1)
			}
			if v > 0.58 {
				m := 1.16 - v
				px = sky(u, m)
				if m > far(u) {
					px = [3]float64{118, 128, 128}
				}
				if m > near(u) {
					px = [3]float64{84, 96, 88}
				}
				ripple := 0.5 + 0.5*math.Sin(v*v*900+math.Sin(u*40))
				px = mix(px, [3]float64{70, 86, 88}, 0.3+0.06*ripple)
			}
			if v > shore(u) {
				px = mix([3]float64{78, 86, 66}, [3]float64{52, 58, 48}, (v-shore(u))/0.2)
			}
			if tree(u, v) {
				px = [3]float64{56, 66, 58}
			}
			n := float64((x*7919+y*104729)%17) - 8
			img.Set(x, y, color.NRGBA{uint8(max(0, min(255, px[0]+n))), uint8(max(0, min(255, px[1]+n+3))), uint8(max(0, min(255, px[2]+n))), 255})
		}
	}
	return img
}

type app struct {
	win    *mygo.Window
	pic    image.Image
	before *ui.Bitmap
	after  *ui.Bitmap
	grade  colorwheels.Grade
	shown  int
	busy   bool
	again  bool
}

// regrade grades the picture on a goroutine, one grade at a time: what
// the wheels do meanwhile waits for it, and goes in the next.
func (a *app) regrade() {
	if a.busy {
		a.again = true
		return
	}
	a.busy = true
	g := a.grade
	go func() {
		out := colorwheels.Apply(a.pic, g)
		a.win.Update(func() {
			a.after.Update(out)
			a.busy = false
			if a.again {
				a.again = false
				a.regrade()
			}
		})
	}()
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(6)).Gap(t.Space(5)).Children(func() {
		ui.Row(c).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "Color wheels").Bold().FontSize(t.FontSize + 6).Grow(1)
			ui.Segmented(c, &a.shown, "Before", "After").Label("Show")
		})
		shown := a.after
		if a.shown == 0 {
			shown = a.before
		}
		ui.Image(c, shown).Grow(1).AlignSelf(ui.Stretch).Fit(ui.Contain).Label("Picture")
		ui.Column(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			if colorwheels.View(c, &a.grade).Changed() {
				a.shown = 1
				a.regrade()
			}
			ui.Text(c, "Drag a wheel to push its tones toward a hue, with Shift for fine moves. Double-click to reset.").
				FontSize(t.FontSize - 1).TextColor(t.TextMuted)
		})
	})
}

func main() {
	a := &app{shown: 1}
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
		a.pic = lake(1440, 900)
	}
	a.before = ui.NewBitmap(a.pic)
	a.after = ui.NewBitmap(a.pic)
	mygo.App.WhenReady(func() {
		a.win = mygo.NewWindow(mygo.WindowOptions{
			Title:   "Color wheels",
			Width:   960,
			Height:  860,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
