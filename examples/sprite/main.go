// Sprite plays a sheet of 16 by 16 pixel art drawn in code: a slime idling
// and hopping, and a coin spinning. Drag across the sheet to make other
// frames the animation's, and turn on the onion skin to see the frame
// before.
//
//	go run ./examples/sprite
package main

import (
	"image"
	"image/color"
	"log"
	"math"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/sprite"
)

type app struct {
	preview    sprite.State
	sheet      *ui.Bitmap
	animations []sprite.Animation
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(6)).Children(func() {
		sprite.View(c, &a.preview, a.sheet, 8, 3, &a.animations)
	})
}

const size = 16

var (
	green     = color.RGBA{0x5c, 0xc8, 0x5a, 0xff}
	darkGreen = color.RGBA{0x2a, 0x6e, 0x3a, 0xff}
	shine     = color.RGBA{0xc8, 0xf5, 0xb0, 0xff}
	ink       = color.RGBA{0x1b, 0x24, 0x2c, 0xff}
	shadow    = color.RGBA{0, 0, 0, 0x40}
	gold      = color.RGBA{0xf5, 0xc2, 0x3d, 0xff}
	darkGold  = color.RGBA{0xa8, 0x6a, 0x10, 0xff}
	paleGold  = color.RGBA{0xff, 0xec, 0x9a, 0xff}
)

// slime draws a slime w wide and h tall, lifted off the ground, into the
// frame at ox, oy.
func slime(img *image.RGBA, ox, oy int, w, h float64, lift int, blink bool) {
	base := float64(size - 1 - lift)
	inside := func(x, y int) bool {
		dx := (float64(x) + 0.5 - size/2) / w
		dy := (base - float64(y) - 0.5) / h
		return dy >= -0.1 && dx*dx+dy*dy <= 1
	}
	sw := int(5 - float64(lift)/2)
	for x := size/2 - sw; x < size/2+sw; x++ {
		img.Set(ox+x, oy+size-1, shadow)
	}
	for y := range size {
		for x := range size {
			if !inside(x, y) {
				continue
			}
			c := green
			if !inside(x-1, y) || !inside(x+1, y) || !inside(x, y-1) || !inside(x, y+1) {
				c = darkGreen
			}
			img.Set(ox+x, oy+y, c)
		}
	}
	top := int(base - h + 1)
	img.Set(ox+size/2-int(w/2)+1, oy+top+1, shine)
	eye := int(base - h*0.45)
	for _, x := range []int{size/2 - 2, size/2 + 1} {
		img.Set(ox+x, oy+eye, ink)
		if !blink {
			img.Set(ox+x, oy+eye-1, ink)
		}
	}
}

// coin draws a coin turned so that it shows w of its width.
func coin(img *image.RGBA, ox, oy int, w float64) {
	const h = 6.5
	for y := range size {
		for x := range size {
			dx := (float64(x) + 0.5 - size/2) / max(w, 1)
			dy := (float64(y) + 0.5 - size/2) / h
			if dx*dx+dy*dy > 1 {
				continue
			}
			c := gold
			switch {
			case dx*dx+dy*dy > 0.6:
				c = darkGold
			case math.Abs(dx) < 0.25 && math.Abs(dy) < 0.5:
				c = paleGold
			}
			img.Set(ox+x, oy+y, c)
		}
	}
}

func drawSheet() *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, 8*size, 3*size))
	for i := range 8 {
		k := math.Sin(2 * math.Pi * float64(i) / 8)
		slime(img, i*size, 0, 6.5+0.6*k, 7-0.6*k, 0, i == 5)
	}
	hop := []struct {
		w, h float64
		lift int
	}{{7.5, 5.5, 0}, {5.5, 8.5, 2}, {6, 7.5, 5}, {6.5, 7, 6}, {6, 7.5, 4}, {5.8, 8, 1}, {7.6, 5.4, 0}, {6.8, 6.6, 0}}
	for i, f := range hop {
		slime(img, i*size, size, f.w, f.h, f.lift, false)
	}
	for i := range 8 {
		coin(img, i*size, 2*size, 6*math.Abs(math.Cos(math.Pi*float64(i)/8)))
	}
	return img
}

func main() {
	a := &app{
		sheet: ui.NewBitmap(drawSheet()).Pixelated(),
		animations: []sprite.Animation{
			{Name: "Idle", From: 0, To: 7, FPS: 8},
			{Name: "Hop", From: 8, To: 15, FPS: 12},
			{Name: "Coin", From: 16, To: 23, FPS: 14},
		},
	}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Sprite",
			Width:   820,
			Height:  500,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
