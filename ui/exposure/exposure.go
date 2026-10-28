// Package exposure shows a picture with the exposure aids of camera
// monitors over it: zebra stripes over the pixels at or above a level,
// false color painting each pixel by its luma, in colors at the levels
// that matter, from crushed blacks in purple through middle gray in green
// and skin in pink to clipped whites in red, and gray between them, with
// a key below; or focus peaking, outlining the sharp edges in red. A
// segmented control chooses the aid and a slider the level of the zebras,
// and the pointer reads out the level under it in IRE, from 0 to 100. The
// overlay is worked out again only when the app passes another picture.
//
//	exposure.View(c, &app.exposure, app.frame)
package exposure

import (
	"fmt"
	"image"
	"image/color"
	"math"

	"github.com/egoist/mygo/ui"
)

const (
	off = iota
	zebra
	falseColor
	peaking
)

// band is a stretch of levels false color paints in a color.
type band struct {
	lo, hi float32
	color  color.NRGBA
	name   string
}

var bands = []band{
	{0, 2, color.NRGBA{120, 40, 170, 255}, "Crushed"},
	{2, 10, color.NRGBA{40, 90, 220, 255}, "Shadows"},
	{38, 48, color.NRGBA{60, 180, 80, 255}, "Middle gray"},
	{52, 62, color.NRGBA{235, 120, 170, 255}, "Skin"},
	{92, 98, color.NRGBA{240, 210, 40, 255}, "Near clipping"},
	{98, 101, color.NRGBA{220, 40, 40, 255}, "Clipped"},
}

// peak is how steep luma must change across a pixel, over its 3 by 3
// neighbors, for peaking to outline it.
const peak = 0.6

// State is what a view keeps: the aid shown, the level of the zebras, and
// the picture's luma and overlays. Keep it in the app's state, one for
// each picture shown.
type State struct {
	aid   int
	level float64
	set   bool

	img     image.Image
	picture *ui.Bitmap
	luma    []float32
	w, h    int
	// made is the aid and level the overlay was made for.
	made    [2]float64
	overlay *ui.Bitmap
}

// load works out the luma of img, from 0 to 100.
func (s *State) load(img image.Image) {
	b := img.Bounds()
	s.img, s.picture, s.w, s.h = img, ui.NewBitmap(img), b.Dx(), b.Dy()
	s.luma = make([]float32, s.w*s.h)
	for y := range s.h {
		for x := range s.w {
			r, g, bl, _ := img.At(b.Min.X+x, b.Min.Y+y).RGBA()
			s.luma[y*s.w+x] = 100 * (0.2126*float32(r) + 0.7152*float32(g) + 0.0722*float32(bl)) / 65535
		}
	}
	s.overlay = nil
}

// draw draws the overlay of the aid chosen.
func (s *State) draw() {
	s.made = [2]float64{float64(s.aid), s.level}
	s.overlay = nil
	if s.aid == off {
		return
	}
	out := image.NewNRGBA(image.Rect(0, 0, s.w, s.h))
	at := func(x, y int) float32 {
		return s.luma[min(max(y, 0), s.h-1)*s.w+min(max(x, 0), s.w-1)]
	}
	for y := range s.h {
		for x := range s.w {
			v := s.luma[y*s.w+x]
			switch s.aid {
			case zebra:
				if v >= float32(s.level) {
					c := color.NRGBA{255, 255, 255, 220}
					if (x+y)/4%2 == 0 {
						c = color.NRGBA{0, 0, 0, 160}
					}
					out.SetNRGBA(x, y, c)
				}
			case falseColor:
				g := uint8(v * 2.55 * 0.8)
				c := color.NRGBA{g, g, g, 255}
				for _, b := range bands {
					if v >= b.lo && v < b.hi {
						c = b.color
					}
				}
				out.SetNRGBA(x, y, c)
			case peaking:
				gx := at(x+1, y-1) + 2*at(x+1, y) + at(x+1, y+1) - at(x-1, y-1) - 2*at(x-1, y) - at(x-1, y+1)
				gy := at(x-1, y+1) + 2*at(x, y+1) + at(x+1, y+1) - at(x-1, y-1) - 2*at(x, y-1) - at(x+1, y-1)
				if float32(math.Hypot(float64(gx), float64(gy)))/100 > peak {
					out.SetNRGBA(x, y, color.NRGBA{255, 40, 40, 255})
				}
			}
		}
	}
	s.overlay = ui.NewBitmap(out)
}

// View creates the picture img with the aid chosen over it. Give it a
// size, or grow it in its parent.
func View(c *ui.Context, s *State, img image.Image) ui.Element {
	t := c.Theme()
	if !s.set {
		s.level, s.set = 95, true
	}
	if img != nil && s.img != img {
		s.load(img)
	}
	if s.img != nil && (s.made != [2]float64{float64(s.aid), s.level} || (s.aid != off && s.overlay == nil)) {
		s.draw()
	}
	return ui.Column(c).Gap(t.Space(3)).Children(func() {
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			ui.Segmented(c, &s.aid, "Off", "Zebra", "False color", "Peaking").Label("Exposure aid")
			if s.aid == zebra {
				ui.Slider(c, &s.level, 50, 100).Step(1).Width(t.Space(36)).Label("Zebra level")
				ui.Text(c, fmt.Sprintf("%.0f IRE", s.level)).FontFeatures("tnum")
			}
		})
		if s.picture == nil {
			return
		}
		overlay := s.overlay
		pic := ui.Image(c, s.picture).Fit(ui.Contain).Grow(1).AlignSelf(ui.Stretch).Label("Picture")
		pic.DrawOver(func(p *ui.Painter, r ui.Rect) {
			if overlay != nil {
				p.Image(overlay, r, ui.Contain)
			}
		})
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			if s.aid == falseColor {
				for i, b := range bands {
					ui.Row(c.Key(i)).Gap(t.Space(1.5)).AlignItems(ui.Center).Children(func() {
						ui.Box(c).Size(t.Space(3), t.Space(3)).Radius(t.Space(0.75)).
							Background(ui.RGB(b.color.R, b.color.G, b.color.B))
						ui.Text(c, b.name).FontSize(t.FontSize - 1).NoWrap()
						ui.Text(c, fmt.Sprintf("%.0f–%.0f", b.lo, min(b.hi, 100))).FontSize(t.FontSize - 1).
							TextColor(t.TextMuted).FontFeatures("tnum").NoWrap()
					})
				}
			}
			ui.Spacer(c)
			px, py, over := pic.PointerPosition()
			r := pic.Bounds()
			k := min(r.W/float32(s.w), r.H/float32(s.h))
			x := int((px - (r.W-float32(s.w)*k)/2) / k)
			y := int((py - (r.H-float32(s.h)*k)/2) / k)
			reading := ""
			if over && k > 0 && x >= 0 && y >= 0 && x < s.w && y < s.h {
				reading = fmt.Sprintf("%.0f IRE", s.luma[y*s.w+x])
			}
			ui.Text(c, reading).FontFeatures("tnum").TextColor(t.TextMuted).Width(t.Space(16)).TextAlign(ui.End)
		})
	})
}
