// Package filters adjusts a photo, as the Adjust panels of photo editors
// do: sliders for exposure, contrast, saturation, warmth, tint, vignette
// and blur over the photo, which a shader redraws on the GPU as they move.
// After shows the photo adjusted, Before as it was, and Split both, either
// side of a line dragged across it. A double click on a slider sets it
// back to none, and Reset sets them all.
//
//	filters.View(c, &app.adjuster, app.photo, &app.adjust).Grow(1)
//
// The app reads *Adjust to apply it as it saves. Changed reports a change.
package filters

import (
	"strconv"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/shader"
)

// Adjust is how a photo is adjusted, each at 0 for none: Exposure in
// stops, from -2 to 2, Blur in pixels of the photo, from 0 to 20, Vignette
// from 0 to 1, and the rest from -1 to 1.
type Adjust struct {
	Exposure, Contrast, Saturation float64
	Warmth, Tint                   float64
	Vignette, Blur                 float64
}

// slider is an adjustment's slider: its name, its range, and its value.
type slider struct {
	name   string
	lo, hi float64
	value  func(a *Adjust) *float64
}

var sliders = []slider{
	{"Exposure", -2, 2, func(a *Adjust) *float64 { return &a.Exposure }},
	{"Contrast", -1, 1, func(a *Adjust) *float64 { return &a.Contrast }},
	{"Saturation", -1, 1, func(a *Adjust) *float64 { return &a.Saturation }},
	{"Warmth", -1, 1, func(a *Adjust) *float64 { return &a.Warmth }},
	{"Tint", -1, 1, func(a *Adjust) *float64 { return &a.Tint }},
	{"Vignette", 0, 1, func(a *Adjust) *float64 { return &a.Vignette }},
	{"Blur", 0, 20, func(a *Adjust) *float64 { return &a.Blur }},
}

// adjust draws the photo, image 0, filling the rectangle, adjusted right
// of Split, in DIPs from its left, and as it was left of it. The colors
// change in the photo's sRGB, as quick editors change them; exposure
// scales them as a stop would in linear light.
var adjust = shader.MustCompile(`package main

var Exposure float
var Contrast float
var Saturation float
var Warmth float
var Tint float
var Vignette float
var Blur float
var Split float

func Fragment(dst vec4, pos vec2) vec4 {
	size := rectSize()
	at := pos / size * image0Size()
	c := image0Sample(at)
	if pos.x < Split {
		return c
	}
	if Blur > 0 {
		sum := vec4(0)
		total := 0.0
		for i := -3; i <= 3; i++ {
			for j := -3; j <= 3; j++ {
				o := vec2(float(i), float(j))
				w := exp(-dot(o, o) / 4.5)
				sum += image0Sample(at+o*Blur/3) * w
				total += w
			}
		}
		c = sum / total
	}
	if c.a == 0 {
		return c
	}
	rgb := c.rgb / c.a
	rgb *= pow(2.0, Exposure/2.2)
	rgb.r *= 1 + 0.12*Warmth
	rgb.b *= 1 - 0.12*Warmth
	rgb.g *= 1 - 0.1*Tint
	rgb = (rgb-0.5)*(1+Contrast) + 0.5
	l := dot(rgb, vec3(0.2126, 0.7152, 0.0722))
	rgb = mix(vec3(l), rgb, 1+Saturation)
	aspect := vec2(size.x/size.y, 1)
	d := length((pos/size-0.5)*aspect) / length(aspect*0.5)
	rgb *= 1 - Vignette*smoothstep(0.35, 1.05, d)
	rgb = clamp(rgb, vec3(0), vec3(1))
	return vec4(rgb*c.a, c.a)
}
`)

// State is what an adjuster keeps: what the photo shows, and where the
// line of Split is across it, from 0 to 1. Keep it in the app's state, one
// for each adjuster.
type State struct {
	show  int
	split float32
}

// View creates the adjuster of pic, adjusted by *a.
func View(c *ui.Context, s *State, pic *ui.Bitmap, a *Adjust) ui.Element {
	t := c.Theme()
	if s.split == 0 {
		s.split = 0.5
	}
	changed := false
	iw, ih := pic.Size()
	root := ui.Row(c).Gap(t.Space(5)).AlignItems(ui.Stretch)
	root.Children(func() {
		stage := ui.Column(c).Grow(1).MinWidth(t.Space(60)).Gap(t.Space(3))
		stage.Children(func() {
			view := ui.Box(c).Grow(1).MinHeight(t.Space(50)).Role(ui.RoleImage).Label("Photo")
			b := view.LayoutBounds()
			k := min(b.W/float32(iw), b.H/float32(ih))
			fit := ui.Rect{X: (b.W - float32(iw)*k) / 2, Y: (b.H - float32(ih)*k) / 2, W: float32(iw) * k, H: float32(ih) * k}
			if s.show == 1 {
				view.Cursor(ui.CursorResizeEW)
				if x, _, _ := view.PointerPosition(); view.Pressed() && fit.W > 0 {
					s.split = max(0, min(1, (x-fit.X)/fit.W))
				}
			}
			adj, show, split := *a, s.show, s.split
			view.Draw(func(p *ui.Painter, r ui.Rect) {
				k := min(r.W/float32(iw), r.H/float32(ih))
				fr := ui.Rect{X: r.X + (r.W-float32(iw)*k)/2, Y: r.Y + (r.H-float32(ih)*k)/2, W: float32(iw) * k, H: float32(ih) * k}
				cut := float32(0)
				switch show {
				case 1:
					cut = fr.W * split
				case 2:
					cut = fr.W + 1
				}
				adjust.Draw(p, fr, shader.Options{
					Images: [4]*ui.Bitmap{pic},
					Uniforms: map[string]any{
						"Exposure": adj.Exposure, "Contrast": adj.Contrast, "Saturation": adj.Saturation,
						"Warmth": adj.Warmth, "Tint": adj.Tint, "Vignette": adj.Vignette, "Blur": adj.Blur,
						"Split": cut,
					},
				})
				if show != 1 {
					return
				}
				x := fr.X + cut
				p.Line(x, fr.Y, x, fr.Y+fr.H, 2, ui.RGB(255, 255, 255))
				var knob ui.Path
				knob.Circle(x, fr.Y+fr.H/2, 12)
				p.FillPath(&knob, ui.RGB(255, 255, 255))
				p.StrokePath(&knob, 1, ui.RGBA(0, 0, 0, 0.2))
				p.Line(x-4, fr.Y+fr.H/2-5, x-4, fr.Y+fr.H/2+5, 1.5, ui.RGBA(0, 0, 0, 0.5))
				p.Line(x+4, fr.Y+fr.H/2-5, x+4, fr.Y+fr.H/2+5, 1.5, ui.RGBA(0, 0, 0, 0.5))
				p.Text(fr.X+8, fr.Y+8, "Before", t.FontSize-2, ui.RGB(255, 255, 255))
				w, _ := p.MeasureText(0, ui.Span{Text: "After", Size: t.FontSize - 2})
				p.Text(fr.X+fr.W-w-8, fr.Y+8, "After", t.FontSize-2, ui.RGB(255, 255, 255))
			})
			ui.Segmented(c, &s.show, "After", "Split", "Before").Label("Show").AlignSelf(ui.Center)
		})
		ui.Column(c).Width(t.Space(64)).Gap(t.Space(3)).Children(func() {
			ui.Row(c).AlignItems(ui.Center).Children(func() {
				ui.Text(c, "Adjust").Bold().Grow(1)
				if ui.Button(c, "Reset").Disabled(*a == Adjust{}).Clicked() {
					*a, changed = Adjust{}, true
				}
			})
			for i, sl := range sliders {
				v := sl.value(a)
				ui.Column(c.Key(i)).Gap(t.Space(1)).Children(func() {
					ui.Row(c).Children(func() {
						ui.Text(c, sl.name).FontSize(t.FontSize - 1).Grow(1)
						ui.Text(c, number(*v)).FontSize(t.FontSize - 1).FontFeatures("tnum").TextColor(t.TextMuted)
					})
					in := track(c, v, sl.lo, sl.hi).Label(sl.name)
					if in.Changed() {
						changed = true
					}
					if in.DoubleClicked() {
						*v, changed = 0, true
					}
				})
			}
		})
	})
	if changed {
		root.SetChanged()
	}
	return root
}

// track creates a slider of *v from lo to hi whose fill runs from 0, so
// that one going both ways fills from its middle.
func track(c *ui.Context, v *float64, lo, hi float64) ui.Element {
	t := c.Theme()
	const knob = 8
	in := ui.SliderBase(c, v, lo, hi).Step((hi - lo) / 200).Height(t.Space(5)).PaddingX(knob)
	val, focus := *v, in.FocusVisible()
	in.FocusRing(false).Draw(func(p *ui.Painter, r ui.Rect) {
		cy := r.Y + r.H/2
		at := func(x float64) float32 { return r.X + knob + (r.W-2*knob)*float32((x-lo)/(hi-lo)) }
		p.Fill(ui.Rect{X: r.X + knob, Y: cy - 2, W: r.W - 2*knob, H: 4}, t.Text.Alpha(0.12), 2)
		z, x := at(max(lo, 0)), at(val)
		p.Fill(ui.Rect{X: min(z, x), Y: cy - 2, W: max(z, x) - min(z, x), H: 4}, t.Accent, 2)
		if lo < 0 {
			p.Fill(ui.Rect{X: z - 1, Y: cy - 5, W: 2, H: 10}, t.TextMuted.Alpha(0.6), 1)
		}
		var dot ui.Path
		dot.Circle(x, cy, knob)
		p.Shadow(ui.Rect{X: x - knob, Y: cy - knob, W: 2 * knob, H: 2 * knob}, knob, 0, 1, 3, 0, ui.RGBA(0, 0, 0, 0.25))
		p.FillPath(&dot, ui.RGB(255, 255, 255))
		ring := t.Border
		if focus {
			ring = t.Focus
		}
		p.StrokePath(&dot, 1, ring)
	})
	return in
}

// number writes v with two decimals, a plus before those above zero.
func number(v float64) string {
	s := strconv.FormatFloat(v, 'f', 2, 64)
	if v > 0.004 {
		return "+" + s
	}
	if s == "-0.00" {
		return "0.00"
	}
	return s
}
