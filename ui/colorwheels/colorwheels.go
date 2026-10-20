// Package colorwheels is the color wheels of a grading app, as DaVinci
// Resolve's: Lift for the shadows, Gamma for the midtones and Gain for
// the highlights. Each is a disc of hues, stronger out to its rim, with a
// puck at its center: dragging on the disc moves the puck toward the hue
// to push the wheel's tones to, ten times finer with Shift. Under it are
// a slider of luminance and the wheel's amounts on red, green and blue.
// A double click on a disc resets its wheel, and one on a slider its
// luminance. Changed reports an edit; Apply grades a picture.
//
//	if colorwheels.View(c, &app.grade).Changed() {
//		app.graded.Update(colorwheels.Apply(app.pic, app.grade))
//	}
package colorwheels

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"
	"runtime"
	"sync"

	"github.com/egoist/mygo/ui"
)

// Wheel is the grade of one range of tones: X and Y the puck's place on
// the disc, to the right and up, within the circle of radius 1 around
// its center, and Lum the luminance, from -1 to 1. The zero Wheel leaves
// tones as they are.
type Wheel struct {
	X, Y, Lum float64
}

// Grade is the three wheels: Lift raises or sinks the shadows, Gamma
// bends the midtones and Gain scales the highlights. A wheel pushes each
// channel by its luminance and its puck, for a lift of half the push,
// and a gamma or a gain of 2 to its power.
type Grade struct {
	Lift, Gamma, Gain Wheel
}

// hues are where red, green and blue are on a disc, in radians
// counterclockwise from the right, a third of a turn apart with red at
// 103°, as on a vectorscope.
var hues = [3]float64{103 * math.Pi / 180, 223 * math.Pi / 180, 343 * math.Pi / 180}

// channels returns the wheel's push on red, green and blue: its
// luminance, plus half of its puck's toward each hue, so that the puck
// shifts the color and leaves the sum of the channels as it is.
func (w Wheel) channels() [3]float64 {
	var v [3]float64
	for c, h := range hues {
		v[c] = w.Lum + 0.5*(w.X*math.Cos(h)+w.Y*math.Sin(h))
	}
	return v
}

// amounts returns the lift, the gamma and the gain of each channel.
func (g Grade) amounts() [3][3]float64 {
	lift, gamma, gain := g.Lift.channels(), g.Gamma.channels(), g.Gain.channels()
	var a [3][3]float64
	for c := range 3 {
		a[0][c] = lift[c] / 2
		a[1][c] = math.Exp2(gamma[c])
		a[2][c] = math.Exp2(gain[c])
	}
	return a
}

// lut returns the tone out for each tone in, of each channel.
func (g Grade) lut() [3][256]uint8 {
	a := g.amounts()
	var l [3][256]uint8
	for c := range l {
		lift, gamma, gain := a[0][c], a[1][c], a[2][c]
		for i := range l[c] {
			x := float64(i) / 255
			y := math.Pow(max(0, gain*(x+lift*(1-x))), 1/gamma)
			l[c][i] = uint8(math.Round(min(1, y) * 255))
		}
	}
	return l
}

// Apply returns pic graded by g, each tone x of each channel to
// (gain × (x + lift × (1 − x)))^(1/gamma). It grades bands of the
// picture on all cores at once, a few milliseconds for a picture of a
// few million pixels; call it on a goroutine for a bigger one, and show
// what it returns through Window.Update.
func Apply(pic image.Image, g Grade) *image.RGBA {
	l := g.lut()
	b := pic.Bounds()
	out := image.NewRGBA(b)
	n := runtime.GOMAXPROCS(0)
	rows := max(1, (b.Dy()+n-1)/n)
	var wg sync.WaitGroup
	for y := b.Min.Y; y < b.Max.Y; y += rows {
		band := image.Rect(b.Min.X, y, b.Max.X, min(b.Max.Y, y+rows))
		wg.Go(func() {
			draw.Draw(out, band, pic, band.Min, draw.Src)
			pix := out.Pix[out.PixOffset(band.Min.X, band.Min.Y):out.PixOffset(band.Min.X, band.Max.Y)]
			for i := 0; i < len(pix); i += 4 {
				switch a := uint32(pix[i+3]); a {
				case 255:
					pix[i], pix[i+1], pix[i+2] = l[0][pix[i]], l[1][pix[i+1]], l[2][pix[i+2]]
				case 0:
				default:
					// The tones of a pixel partly see-through are its
					// color's, before its alpha multiplied them.
					for c := range 3 {
						v := min(255, uint32(pix[i+c])*255/a)
						pix[i+c] = uint8(uint32(l[c][v]) * a / 255)
					}
				}
			}
		})
	}
	wg.Wait()
	return out
}

// disc is a disc of the hues, see-through at its center and more opaque
// out to its rim, which a ring of them at full strength edges, for the
// wheels to draw over their surface.
var disc = sync.OnceValue(func() *ui.Bitmap {
	const n = 384
	img := image.NewNRGBA(image.Rect(0, 0, n, n))
	for y := range n {
		for x := range n {
			dx, dy := float64(x)+0.5-n/2, n/2-float64(y)-0.5
			d := math.Hypot(dx, dy) / (n / 2)
			edge := max(0, min(1, (1-d)*n/2))
			if edge == 0 {
				continue
			}
			ring := max(0, min(1, (d-0.93)*n/2))
			alpha := edge * max(ring, 0.55*d*d)
			a := math.Atan2(dy, dx)
			var v [3]float64
			for c, h := range hues {
				v[c] = math.Cos(a - h)
			}
			lo, hi := min(v[0], v[1], v[2]), max(v[0], v[1], v[2])
			px := color.NRGBA{A: uint8(255 * alpha)}
			px.R = uint8(255 * (v[0] - lo) / (hi - lo))
			px.G = uint8(255 * (v[1] - lo) / (hi - lo))
			px.B = uint8(255 * (v[2] - lo) / (hi - lo))
			img.SetNRGBA(x, y, px)
		}
	}
	return ui.NewBitmap(img)
})

var names = [3]string{"Lift", "Gamma", "Gain"}

// View creates the wheels of *g, side by side.
func View(c *ui.Context, g *Grade) ui.Element {
	t := c.Theme()
	root := ui.Row(c).Gap(t.Space(6))
	root.Children(func() {
		a := g.amounts()
		for i, w := range []*Wheel{&g.Lift, &g.Gamma, &g.Gain} {
			if wheel(c.Key(i), names[i], w, a[i], i > 0) {
				root.SetChanged()
			}
		}
	})
	return root
}

// wheel shows w, named name, over its amounts on each channel, written
// with their sign unless they are a scale, and reports an edit.
func wheel(c *ui.Context, name string, w *Wheel, amounts [3]float64, scale bool) bool {
	t := c.Theme()
	size := t.Space(36)
	changed := false
	ui.Column(c).Gap(t.Space(2.5)).AlignItems(ui.Center).Width(size).Children(func() {
		ui.Text(c, name).Bold()
		pad := ui.Box(c).Size(size, size).Cursor(ui.CursorGrab).Label(name)
		if dx, dy, ok := pad.Dragged(); ok && (dx != 0 || dy != 0) {
			k := float64(2 / size)
			if c.Modifiers()&ui.Shift != 0 {
				k /= 10
			}
			w.X, w.Y = w.X+float64(dx)*k, w.Y-float64(dy)*k
			if d := math.Hypot(w.X, w.Y); d > 1 {
				w.X, w.Y = w.X/d, w.Y/d
			}
			changed = true
		}
		if pad.DoubleClicked() && *w != (Wheel{}) {
			*w = Wheel{}
			changed = true
		}
		held := pad.Pressed()
		if held {
			pad.Cursor(ui.CursorGrabbing)
		}
		x, y := float32(w.X), float32(w.Y)
		pad.Draw(func(p *ui.Painter, r ui.Rect) {
			p.Fill(r, t.Surface, r.W/2)
			p.Image(disc(), r, ui.FillBox)
			p.Stroke(r, t.Border, r.W/2, 1)
			cx, cy := r.X+r.W/2, r.Y+r.H/2
			arm := r.W/2 - t.Space(2)
			p.Line(cx-t.Space(2), cy, cx+t.Space(2), cy, 1, t.Text.Alpha(0.3))
			p.Line(cx, cy-t.Space(2), cx, cy+t.Space(2), 1, t.Text.Alpha(0.3))
			px, py := cx+x*arm, cy-y*arm
			if x != 0 || y != 0 {
				p.Line(cx, cy, px, py, 1, t.Text.Alpha(0.35))
			}
			rad := t.Space(1.75)
			if held {
				rad = t.Space(2.25)
			}
			puck := ui.Rect{X: px - rad, Y: py - rad, W: 2 * rad, H: 2 * rad}
			p.Shadow(puck, rad, 0, 1, 3, 0, ui.RGB(0, 0, 0).Alpha(0.3))
			p.Fill(puck, ui.RGB(255, 255, 255), rad)
			p.Stroke(puck, ui.RGB(0, 0, 0).Alpha(0.55), rad, 1.5)
		})
		if lum(c, name, &w.Lum, size) {
			changed = true
		}
		ui.Row(c).Gap(t.Space(2)).Children(func() {
			for ch, v := range amounts {
				s := fmt.Sprintf("%+.2f", v)
				if scale {
					s = fmt.Sprintf("%.2f", v)
				}
				ui.Text(c, string("RGB"[ch])+" "+s).FontSize(t.FontSize - 2).FontFeatures("tnum").TextColor(t.TextMuted)
			}
		})
	})
	return changed
}

// lum shows the luminance slider of the wheel named name, size wide,
// filled from its middle, and reports an edit.
func lum(c *ui.Context, name string, v *float64, size float32) bool {
	t := c.Theme()
	kw, kh := t.Space(4), t.Space(3.5)
	s := ui.SliderBase(c, v, -1, 1).Width(size).Height(t.Space(5)).PaddingX(kw / 2).FocusRing(false).
		Label(name + " luminance").Value(fmt.Sprintf("%+.2f", *v))
	changed := s.Changed()
	if s.DoubleClicked() && *v != 0 {
		*v = 0
		changed = true
	}
	f := float32((*v + 1) / 2)
	if c.Direction() == ui.RightToLeft {
		f = 1 - f
	}
	focus := s.FocusVisible()
	s.Draw(func(p *ui.Painter, r ui.Rect) {
		h := t.Space(1)
		cy := r.Y + r.H/2
		x0, x1 := r.X+kw/2, r.X+r.W-kw/2
		mid, x := (x0+x1)/2, x0+(x1-x0)*f
		p.Fill(ui.Rect{X: x0, Y: cy - h/2, W: x1 - x0, H: h}, t.Border, h/2)
		p.Fill(ui.Rect{X: mid - 0.5, Y: cy - t.Space(1.5), W: 1, H: t.Space(3)}, t.TextMuted, 0)
		p.Fill(ui.Rect{X: min(mid, x), Y: cy - h/2, W: max(mid, x) - min(mid, x), H: h}, t.Accent, h/2)
		k := ui.Rect{X: x - kw/2, Y: cy - kh/2, W: kw, H: kh}
		p.Shadow(k, kh/2, 0, 1, 4, 0, ui.RGB(0, 0, 0).Alpha(0.2))
		p.Fill(k, ui.RGB(255, 255, 255), kh/2)
		p.Stroke(k, ui.RGB(0, 0, 0).Alpha(0.2), kh/2, 1)
		if focus {
			p.FocusRing(k, [4]float32{kh / 2, kh / 2, kh / 2, kh / 2})
		}
	})
	return changed
}
