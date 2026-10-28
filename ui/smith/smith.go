// Package smith draws a Smith chart, as RF engineers plot an antenna's
// impedance across a sweep: circles of constant resistance and arcs of
// constant reactance, normalized to the system impedance and labeled,
// each trace a line in its own color from a dot at its first frequency
// to a ring at its last, both labeled with their frequency, and a
// shaded circle inside which the VSWR is within a limit. The pointer
// near a trace shows its nearest point: the frequency, the impedance in
// ohms, the reflection coefficient, VSWR and return loss, and the
// inductor or capacitor the reactance is at that frequency. Elsewhere
// inside the chart it shows the impedance under it.
//
//	smith.View(c, 50, []smith.Trace{{Name: "Antenna", Points: sweep}}, 2)
package smith

import (
	"fmt"
	"math"
	"math/cmplx"

	"github.com/egoist/mygo/ui"
)

// Point is an impedance in ohms at a frequency in hertz.
type Point struct {
	Freq float64
	Z    complex128
}

// Trace is a named sweep of impedances, in order of frequency.
type Trace struct {
	Name   string
	Points []Point
}

// Color returns the color of trace i.
func Color(t *ui.Theme, i int) ui.Color {
	l := float32(0.55)
	if t.Dark {
		l = 0.72
	}
	return ui.Oklch(l, 0.18, 255+float32(i)*110)
}

// Gamma returns the reflection coefficient of z on a line of impedance
// z0.
func Gamma(z complex128, z0 float64) complex128 {
	return (z - complex(z0, 0)) / (z + complex(z0, 0))
}

// Impedance returns the impedance whose reflection coefficient on a
// line of impedance z0 is g.
func Impedance(g complex128, z0 float64) complex128 {
	return complex(z0, 0) * (1 + g) / (1 - g)
}

var (
	resistances = []float64{0.2, 0.5, 1, 2, 5}
	reactances  = []float64{0.2, 0.5, 1, 2, 5}
)

// View creates the chart of traces on a line of impedance z0, with the
// circle of VSWR within vswr shaded when it is above 1. Give it a size,
// or grow it in its parent.
func View(c *ui.Context, z0 float64, traces []Trace, vswr float64) ui.Element {
	t := c.Theme()
	box := ui.Box(c).MinWidth(t.Space(90)).MinHeight(t.Space(90)).Role(ui.RoleImage).Label("Smith chart")
	b := box.LayoutBounds()
	pad := t.Space(9)
	rad := max(1, min(b.W, b.H)/2-pad)
	cx, cy := b.W/2, b.H/2
	toPx := func(g complex128) (float32, float32) {
		return cx + float32(real(g))*rad, cy - float32(imag(g))*rad
	}
	hotTrace, hotPoint := -1, -1
	var under complex128
	inside := false
	if px, py, over := box.PointerPosition(); over {
		best := float32(14)
		for i, tr := range traces {
			for k, p := range tr.Points {
				x, y := toPx(Gamma(p.Z, z0))
				if d := float32(math.Hypot(float64(x-px), float64(y-py))); d < best {
					best, hotTrace, hotPoint = d, i, k
				}
			}
		}
		g := complex(float64((px-cx)/rad), float64((cy-py)/rad))
		if cmplx.Abs(g) < 1 {
			under, inside = g, true
		}
	}
	box.Draw(func(p *ui.Painter, r ui.Rect) {
		if r.W != b.W || r.H != b.H {
			c.Invalidate()
		}
		at := func(g complex128) [2]float32 {
			x, y := toPx(g)
			return [2]float32{r.X + x, r.Y + y}
		}
		o := at(0)
		disc := ui.Rect{X: o[0] - rad, Y: o[1] - rad, W: 2 * rad, H: 2 * rad}
		p.Fill(disc, t.Surface, rad)
		if vswr > 1 {
			k := float32((vswr-1)/(vswr+1)) * rad
			ring := ui.Rect{X: o[0] - k, Y: o[1] - k, W: 2 * k, H: 2 * k}
			p.Fill(ring, t.Success.Alpha(0.12), k)
			p.StrokeDashed(ring, t.Success.Alpha(0.7), k, 1)
		}
		grid := t.Text.Alpha(0.16)
		small := t.FontSize - 4
		for _, rn := range resistances {
			k := rad / float32(rn+1)
			x := o[0] + rad - k
			p.Stroke(ui.Rect{X: x - k, Y: o[1] - k, W: 2 * k, H: 2 * k}, grid, k, 1)
			label := fmt.Sprint(rn)
			w, _ := p.MeasureText(0, ui.Span{Text: label, Size: small})
			p.RichText(x-k-w-2, o[1]+2, 0, ui.Span{Text: label, Size: small, Color: t.TextMuted, Features: "tnum"})
		}
		for _, xn := range reactances {
			for _, sign := range []float64{1, -1} {
				var pts [][2]float32
				for k := range 61 {
					rr := math.Tan(float64(k) / 60 * math.Pi / 2 * 0.995)
					pts = append(pts, at(Gamma(complex(rr, sign*xn), 1)))
				}
				p.StrokePolyline(pts, 1, grid)
				g := Gamma(complex(0, sign*xn), 1)
				e := at(g * 1.07)
				label := fmt.Sprint(sign * xn)
				if sign > 0 {
					label = "+" + label
				}
				w, h := p.MeasureText(0, ui.Span{Text: label, Size: small})
				p.RichText(e[0]-w/2, e[1]-h/2, 0, ui.Span{Text: label, Size: small, Color: t.TextMuted, Features: "tnum"})
			}
		}
		p.Line(o[0]-rad, o[1], o[0]+rad, o[1], 1, grid)
		p.Stroke(disc, t.Text.Alpha(0.5), rad, 1.25)
		for i, tr := range traces {
			if len(tr.Points) == 0 {
				continue
			}
			col := Color(t, i)
			pts := make([][2]float32, len(tr.Points))
			for k, pt := range tr.Points {
				pts[k] = at(Gamma(pt.Z, z0))
			}
			p.StrokePolyline(pts, 2, col)
			first, last := pts[0], pts[len(pts)-1]
			d := float32(3.5)
			p.Fill(ui.Rect{X: first[0] - d, Y: first[1] - d, W: 2 * d, H: 2 * d}, col, d)
			d = 4.5
			p.Fill(ui.Rect{X: last[0] - d, Y: last[1] - d, W: 2 * d, H: 2 * d}, t.Surface, d)
			p.Stroke(ui.Rect{X: last[0] - d, Y: last[1] - d, W: 2 * d, H: 2 * d}, col, d, 2)
			for k, e := range [][2]float32{first, last} {
				f := tr.Points[0].Freq
				if k == 1 {
					f = tr.Points[len(tr.Points)-1].Freq
				}
				sp := ui.Span{Text: Freq(f), Size: small + 1, Color: col, Weight: 600}
				w, h := p.MeasureText(0, sp)
				// Away from the center, where the traces crowd.
				x, y := e[0]+7, e[1]-h-2
				if e[0] < o[0] {
					x = e[0] - 7 - w
				}
				if e[1] > o[1] {
					y = e[1] + 2
				}
				p.RichText(x, y, 0, sp)
			}
		}
		switch {
		case hotTrace >= 0:
			pt := traces[hotTrace].Points[hotPoint]
			e := at(Gamma(pt.Z, z0))
			col := Color(t, hotTrace)
			d := float32(6)
			p.Fill(ui.Rect{X: e[0] - d, Y: e[1] - d, W: 2 * d, H: 2 * d}, col, d)
			p.Stroke(ui.Rect{X: e[0] - d, Y: e[1] - d, W: 2 * d, H: 2 * d}, t.Background, d, 2)
			lines := []ui.Span{{Text: traces[hotTrace].Name + " at " + Freq(pt.Freq), Weight: 600, Color: col}}
			card(p, t, r, e, append(lines, readout(t, pt.Z, z0, pt.Freq)...))
		case inside:
			e := at(under)
			p.Line(e[0]-6, e[1], e[0]+6, e[1], 1, t.Text.Alpha(0.6))
			p.Line(e[0], e[1]-6, e[0], e[1]+6, 1, t.Text.Alpha(0.6))
			card(p, t, r, e, readout(t, Impedance(under, z0), z0, 0))
		}
	})
	return box
}

// readout returns the lines of the card of impedance z: in ohms, as a
// reflection coefficient, VSWR and return loss, and at frequency f, if
// set, the inductor or capacitor of its reactance.
func readout(t *ui.Theme, z complex128, z0, f float64) []ui.Span {
	g := Gamma(z, z0)
	m := cmplx.Abs(g)
	sign := "+"
	if imag(z) < 0 {
		sign = "−"
	}
	lines := []ui.Span{
		{Text: fmt.Sprintf("Z  %.1f %s j%.1f Ω", real(z), sign, math.Abs(imag(z)))},
		{Text: fmt.Sprintf("Γ  %.3f ∠ %.1f°", m, cmplx.Phase(g)*180/math.Pi)},
		{Text: fmt.Sprintf("VSWR  %.2f", (1+m)/(1-m))},
		{Text: fmt.Sprintf("Return loss  %.1f dB", -20*math.Log10(m))},
	}
	if f > 0 && math.Abs(imag(z)) > 0.05 {
		w := 2 * math.Pi * f
		if imag(z) > 0 {
			lines = append(lines, ui.Span{Text: "Inductive  " + si(imag(z)/w, "H")})
		} else {
			lines = append(lines, ui.Span{Text: "Capacitive  " + si(-1/(w*imag(z)), "F")})
		}
	}
	for i := range lines {
		if lines[i].Color == (ui.Color{}) {
			lines[i].Color = t.Text
		}
		lines[i].Size = t.FontSize - 2
		lines[i].Features = "tnum"
	}
	return lines
}

// card paints lines in a card beside e, kept inside r.
func card(p *ui.Painter, t *ui.Theme, r ui.Rect, e [2]float32, lines []ui.Span) {
	pad := t.Space(2.5)
	w, h := float32(0), float32(0)
	for _, l := range lines {
		lw, lh := p.MeasureText(0, l)
		w, h = max(w, lw), h+lh+t.Space(0.5)
	}
	w, h = w+2*pad, h+2*pad
	x := e[0] + t.Space(4)
	if x+w > r.X+r.W {
		x = e[0] - t.Space(4) - w
	}
	y := max(r.Y, min(e[1]-h/2, r.Y+r.H-h))
	box := ui.Rect{X: x, Y: y, W: w, H: h}
	p.Shadow(box, t.Radius+2, 0, 2, 10, 0, ui.RGBA(0, 0, 0, 0.2))
	p.Fill(box, t.Background, t.Radius+2)
	p.Stroke(box, t.Border, t.Radius+2, 1)
	ty := y + pad
	for _, l := range lines {
		_, lh := p.RichText(x+pad, ty, 0, l)
		ty += lh + t.Space(0.5)
	}
}

// Freq writes f in hertz with the unit that suits it: 2.45 GHz.
func Freq(f float64) string {
	return si(f, "Hz")
}

// si writes v with an SI prefix and unit: 3.3 nH, 1.2 pF.
func si(v float64, unit string) string {
	prefixes := []struct {
		scale float64
		name  string
	}{{1e9, "G"}, {1e6, "M"}, {1e3, "k"}, {1, ""}, {1e-3, "m"}, {1e-6, "µ"}, {1e-9, "n"}, {1e-12, "p"}, {1e-15, "f"}}
	for _, pr := range prefixes {
		if math.Abs(v) >= pr.scale {
			return fmt.Sprintf("%.3g %s%s", v/pr.scale, pr.name, unit)
		}
	}
	return fmt.Sprintf("%.3g %s", v, unit)
}
