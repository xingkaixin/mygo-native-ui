// Trimbar trims 48 seconds of a sunset drawn in code: drag the yellow
// handles to keep part of it, drag between them to scrub, and play what
// is kept. I and O set the in and out points at the playhead.
//
//	go run ./examples/trimbar
package main

import (
	"image"
	"image/color"
	"log"
	"math"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/trimbar"
)

const length = 48 * time.Second

type app struct {
	trim              trimbar.State
	in, out, playhead time.Duration
	frames            []*ui.Bitmap
	playing           bool
	playedFrom        time.Duration
	started           time.Time
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	now := c.Now()
	if a.playing {
		a.playhead = a.playedFrom + now.Sub(a.started)
		if a.playhead >= a.out {
			a.playhead, a.playing = a.out, false
		} else {
			c.After(time.Second / 30)
		}
	}
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(6)).Gap(t.Space(5)).Children(func() {
		frame := a.frames[min(len(a.frames)-1, int(int64(a.playhead)*int64(len(a.frames))/int64(length)))]
		ui.Image(c, frame).Grow(1).Basis(0).FillWidth().Fit(ui.Contain).Radius(t.Radius)
		trimbar.View(c, &a.trim, length, &a.in, &a.out, &a.playhead, a.frames)
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			label := "Play"
			if a.playing {
				label = "Pause"
			}
			if ui.Button(c, label).Clicked() {
				a.playing = !a.playing
				if a.playhead >= a.out {
					a.playhead = a.in
				}
				a.playedFrom, a.started = a.playhead, now
			}
			ui.Text(c, "I and O set the in and out points at the playhead").TextColor(t.TextMuted)
		})
	})
}

func lerp(a, b color.RGBA, u float64) color.RGBA {
	m := func(x, y uint8) uint8 { return uint8(float64(x) + (float64(y)-float64(x))*u) }
	return color.RGBA{m(a.R, b.R), m(a.G, b.G), m(a.B, b.B), 255}
}

// sunset draws the scene u of the way through, from afternoon at 0 to
// dusk at 1.
func sunset(u float64) image.Image {
	const w, h = 320, 180
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	top := lerp(color.RGBA{0x4a, 0x86, 0xc8, 0}, color.RGBA{0x24, 0x1a, 0x52, 0}, u)
	glow := lerp(color.RGBA{0xff, 0xe2, 0xa6, 0}, color.RGBA{0xff, 0x5e, 0x3a, 0}, u)
	sun := lerp(color.RGBA{0xff, 0xf8, 0xdc, 0}, color.RGBA{0xff, 0x9a, 0x5a, 0}, u)
	sea := lerp(color.RGBA{0x2c, 0x5a, 0x86, 0}, color.RGBA{0x1a, 0x12, 0x3a, 0}, u)
	horizon := 0.62 * h
	sx, sy, sr := w*(0.3+0.4*u), h*(0.2+0.5*u), 0.09*h
	for y := range h {
		for x := range w {
			fx, fy := float64(x)+0.5, float64(y)+0.5
			var c color.RGBA
			if fy < horizon {
				c = lerp(top, glow, math.Pow(fy/horizon, 1.6))
				if d := math.Hypot(fx-sx, fy-sy); d < sr+0.5 {
					c = lerp(c, sun, min(1, sr+0.5-d))
				}
			} else {
				c = lerp(sea, glow, 0.35*math.Exp(-math.Abs(fx-sx)/20)*(1-(fy-horizon)/(h-horizon)))
			}
			hill := horizon - 12 - 10*math.Sin(fx/36) - 6*math.Sin(fx/14+1)
			if fx > 190 && fy > hill && fy < horizon+1 {
				c = lerp(color.RGBA{0x1c, 0x22, 0x30, 0}, color.RGBA{0x0c, 0x0a, 0x18, 0}, u)
			}
			img.Set(x, y, c)
		}
	}
	return img
}

func main() {
	a := &app{in: 6 * time.Second, out: 36 * time.Second, playhead: 14 * time.Second}
	for i := range 24 {
		a.frames = append(a.frames, ui.NewBitmap(sunset(float64(i)/23)))
	}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Trim",
			Width:   880,
			Height:  640,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
