// Package ramp edits a tonal ramp of a color in OKLCH, as Radix Colors and
// Tailwind's palettes are made: a row of swatches from light to dark, and
// a curve each for their lightness, chroma and hue, whose points are
// dragged up and down. The chroma curve shades what sRGB cannot show at
// each step's lightness and hue, and a swatch past it is marked; such a
// step shows with less chroma on screens of sRGB. Each swatch shows text
// in white or black, whichever reads better, with its WCAG contrast.
// Below them, the chosen step: its name, its oklch() and its hex.
//
//	ramp.View(c, &app.ramp, &app.blue)
//
// A click on a swatch or a point chooses that step; ↑ and ↓ then nudge
// its lightness, and with Shift its chroma.
package ramp

import (
	"fmt"
	"math"

	"github.com/egoist/mygo/ui"
)

// maxC is the top of the chroma curve, past the chroma sRGB shows at
// any lightness and hue.
const maxC = 0.37

// Step is a color of a ramp in OKLCH: lightness from 0 to 1, chroma from
// 0 to about 0.37, and hue in degrees.
type Step struct {
	Name    string
	L, C, H float32
}

// Color returns the step's color.
func (s Step) Color() ui.Color { return ui.Oklch(s.L, s.C, s.H) }

// InGamut reports whether sRGB shows the step as it is.
func (s Step) InGamut() bool {
	r, g, b := linear(s.L, s.C, s.H)
	const eps = 1e-4
	return r >= -eps && r <= 1+eps && g >= -eps && g <= 1+eps && b >= -eps && b <= 1+eps
}

// Fit returns the step with its chroma lowered as far as sRGB needs to
// show it.
func (s Step) Fit() Step {
	s.C = min(s.C, ceiling(s.L, s.H))
	return s
}

// CSS returns the step as CSS's oklch().
func (s Step) CSS() string {
	return fmt.Sprintf("oklch(%.1f%% %.3f %.1f)", s.L*100, s.C, s.H)
}

// linear returns the linear sRGB of an OKLCH color, as Björn Ottosson's
// OKLab defines it, outside 0 to 1 where sRGB cannot show it.
func linear(l, c, h float32) (r, g, b float64) {
	hr := float64(h) * math.Pi / 180
	A, B := float64(c)*math.Cos(hr), float64(c)*math.Sin(hr)
	L := float64(l)
	l_ := L + 0.3963377774*A + 0.2158037573*B
	m_ := L - 0.1055613458*A - 0.0638541728*B
	s_ := L - 0.0894841775*A - 1.2914855480*B
	lc, mc, sc := l_*l_*l_, m_*m_*m_, s_*s_*s_
	return 4.0767416621*lc - 3.3077115913*mc + 0.2309699292*sc,
		-1.2684380046*lc + 2.6097574011*mc - 0.3413193965*sc,
		-0.0041960863*lc - 0.7034186147*mc + 1.7076147010*sc
}

// ceiling returns the most chroma sRGB shows at lightness l and hue h.
func ceiling(l, h float32) float32 {
	lo, hi := float32(0), float32(maxC)
	for range 16 {
		mid := (lo + hi) / 2
		if (Step{L: l, C: mid, H: h}).InGamut() {
			lo = mid
		} else {
			hi = mid
		}
	}
	return lo
}

// contrast returns the WCAG contrast of c against white and against
// black.
func contrast(c ui.Color) (white, black float64) {
	c = c.SRGB()
	lin := func(v uint8) float64 {
		x := float64(v) / 255
		if x <= 0.04045 {
			return x / 12.92
		}
		return math.Pow((x+0.055)/1.055, 2.4)
	}
	y := 0.2126*lin(c.R) + 0.7152*lin(c.G) + 0.0722*lin(c.B)
	return 1.05 / (y + 0.05), (y + 0.05) / 0.05
}

// curve is one of the three curves: what it shows of a step, and sets.
type curve struct {
	name     string
	top      float32
	get      func(*Step) *float32
	ceilings bool
}

var curves = []curve{
	{"Lightness", 1, func(s *Step) *float32 { return &s.L }, false},
	{"Chroma", maxC, func(s *Step) *float32 { return &s.C }, true},
	{"Hue", 360, func(s *Step) *float32 { return &s.H }, false},
}

// State is what an editor keeps: the step chosen and the point dragged.
// Keep it in the app's state, one for each editor.
type State struct {
	selected int
	// curve is the curve whose point is dragged, from 1, 0 for none.
	curve int
}

// View creates an editor of *steps, from the lightest to the darkest.
func View(c *ui.Context, s *State, steps *[]Step) ui.Element {
	t := c.Theme()
	view := ui.Column(c).Gap(t.Space(5)).Focusable().Label("Color ramp")
	st := *steps
	if len(st) == 0 {
		return view
	}
	s.selected = max(0, min(len(st)-1, s.selected))
	sel := &st[s.selected]
	switch {
	case view.Shortcut(0, ui.KeyUp):
		sel.L = min(1, sel.L+0.01)
	case view.Shortcut(0, ui.KeyDown):
		sel.L = max(0, sel.L-0.01)
	case view.Shortcut(ui.Shift, ui.KeyUp):
		sel.C = min(maxC, sel.C+0.005)
	case view.Shortcut(ui.Shift, ui.KeyDown):
		sel.C = max(0, sel.C-0.005)
	}
	view.Children(func() {
		ui.Row(c).Gap(t.Space(1)).Children(func() {
			for i := range st {
				if swatch(c, &st[i], i == s.selected) {
					s.selected = i
				}
			}
		})
		ui.Row(c).Gap(t.Space(3)).Children(func() {
			for k, cv := range curves {
				panel(c, s, st, k, cv)
			}
		})
		details(c, *sel)
	})
	return view
}

// swatch builds a step's swatch, and reports whether it was clicked.
func swatch(c *ui.Context, st *Step, chosen bool) bool {
	t := c.Theme()
	col := st.Color()
	white, black := contrast(col)
	ink, ratio := ui.RGB(255, 255, 255), white
	if black > white {
		ink, ratio = ui.RGB(0, 0, 0), black
	}
	b := ui.Column(c.Key(st.Name)).Grow(1).Basis(0).Height(t.Space(22)).Radius(t.Radius).Background(col).
		Padding(t.Space(2)).Justify(ui.End).Cursor(ui.CursorPointer).Role(ui.RoleButton).Label(st.Name)
	if chosen {
		b.Border(2, t.Text)
	}
	gamut := st.InGamut()
	b.Children(func() {
		ui.Text(c, "Aa").Bold().TextColor(ink)
		ui.Text(c, fmt.Sprintf("%.1f", ratio)).FontSize(t.FontSize - 2).TextColor(ink.Alpha(0.75))
		ui.Text(c, st.Name).FontSize(t.FontSize - 2).Bold().TextColor(ink.Alpha(0.75))
		if !gamut {
			ui.Text(c, "!").Bold().TextColor(ink).Attach(ui.AnchorTopRight, ui.AnchorTopRight).Top(t.Space(1)).Right(t.Space(2)).
				Tooltip("Past sRGB: shows with less chroma")
		}
	})
	return b.Clicked()
}

// panel builds curve k of the steps, its points dragged up and down.
func panel(c *ui.Context, s *State, st []Step, k int, cv curve) {
	t := c.Theme()
	n := len(st)
	ui.Column(c).Grow(1).Basis(0).Gap(t.Space(1)).Children(func() {
		sel := st[s.selected]
		ui.Row(c).Children(func() {
			ui.Text(c, cv.name).FontSize(t.FontSize - 1).Bold().Grow(1)
			ui.Text(c, fmt.Sprintf("%.3g", *cv.get(&sel))).FontSize(t.FontSize - 1).TextColor(t.TextMuted).FontFeatures("tnum")
		})
		box := ui.Box(c).Height(t.Space(40)).Radius(t.Radius).Border(1, t.Border).Background(t.Background).Cursor(ui.CursorResizeNS)
		b := box.LayoutBounds()
		pad := float32(10)
		x := func(i int) float32 { return pad + float32(i)*(b.W-2*pad)/float32(max(1, n-1)) }
		y := func(v float32) float32 { return pad + (1-v/cv.top)*(b.H-2*pad) }
		px, py, _ := box.PointerPosition()
		if box.Pressed() {
			if s.curve != k+1 {
				near := 0
				for i := range n {
					if abs(x(i)-px) < abs(x(near)-px) {
						near = i
					}
				}
				s.curve, s.selected = k+1, near
			}
			v := (1 - (py-pad)/max(1, b.H-2*pad)) * cv.top
			*cv.get(&st[s.selected]) = max(0, min(cv.top, v))
		} else if s.curve == k+1 {
			s.curve = 0
		}

		ceil := make([]float32, n)
		if cv.ceilings {
			for i, p := range st {
				ceil[i] = ceiling(p.L, p.H)
			}
		}
		chosen := s.selected
		box.Draw(func(p *ui.Painter, r ui.Rect) {
			at := func(i int, v float32) (float32, float32) { return r.X + x(i), r.Y + y(v) }
			if cv.ceilings {
				out := new(ui.Path)
				out.MoveTo(at(0, cv.top))
				for i := range n {
					out.LineTo(at(i, ceil[i]))
				}
				out.LineTo(at(n-1, cv.top)).Close()
				p.FillPath(out, t.Danger.Alpha(0.1))
			}
			line := new(ui.Path)
			for i := range n {
				px, py := at(i, *cv.get(&st[i]))
				if i == 0 {
					line.MoveTo(px, py)
				} else {
					line.LineTo(px, py)
				}
			}
			p.StrokePath(line, 1.5, t.TextMuted)
			for i := range n {
				px, py := at(i, *cv.get(&st[i]))
				rad := float32(5)
				if i == chosen {
					rad = 7
				}
				dot := ui.Rect{X: px - rad, Y: py - rad, W: 2 * rad, H: 2 * rad}
				p.Fill(dot, t.Background, rad)
				p.Fill(ui.Rect{X: dot.X + 1.5, Y: dot.Y + 1.5, W: dot.W - 3, H: dot.H - 3}, st[i].Color(), rad-1.5)
				if i == chosen {
					p.Stroke(dot, t.Text, rad, 1.5)
				}
			}
		})
	})
}

// details shows the chosen step's name, oklch(), hex and gamut.
func details(c *ui.Context, st Step) {
	t := c.Theme()
	col := st.Color().SRGB()
	ui.Row(c).Gap(t.Space(4)).AlignItems(ui.Center).Children(func() {
		ui.Box(c).Size(t.Space(8), t.Space(8)).Radius(t.Radius).Background(st.Color())
		ui.Text(c, st.Name).Bold()
		ui.Text(c, st.CSS()).Font("monospace").Selectable()
		ui.Text(c, fmt.Sprintf("#%02x%02x%02x", col.R, col.G, col.B)).Font("monospace").Selectable().TextColor(t.TextMuted)
		if !st.InGamut() {
			ui.Text(c, "Past sRGB").TextColor(t.Danger)
		}
	})
}

func abs(v float32) float32 { return float32(math.Abs(float64(v))) }
