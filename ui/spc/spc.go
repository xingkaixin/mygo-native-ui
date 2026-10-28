// Package spc draws a control chart of a process, as Shewhart's
// individuals chart is read on a shop floor: the measurements in order,
// joined by a line, around a center line at their mean, dashed control
// limits three sigmas away, and fainter lines at one and two sigmas. The
// mean and sigma come from the first points, the baseline, shaded behind
// them, sigma from their average moving range. A point breaking one of
// the Western Electric rules is red, with the number of the rule above
// it:
//
//  1. a point beyond three sigmas;
//  2. two of three points in a row beyond two sigmas on one side;
//  3. four of five points in a row beyond one sigma on one side;
//  4. eight points in a row on one side of the center.
//
// The limits are named at the right. The pointer on a point shows its
// label, its value, how many sigmas it is from the center, and the rules
// it breaks.
//
//	spc.View(c, spc.Chart{Values: app.diameters, Labels: app.times, Baseline: 20, Unit: "mm"}).Grow(1)
package spc

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/chart"
)

// Chart is a process: its measurements in order, a label for each, as a
// time, how many of the first measurements the limits come from, 0 for
// all of them, and the unit of the measurements.
type Chart struct {
	Values   []float64
	Labels   []string
	Baseline int
	Unit     string
}

// Limits are the center of a process and its sigma.
type Limits struct {
	Center, Sigma float64
}

// Estimate returns the limits of values: their mean, and sigma as their
// average moving range over d2, 1.128 for ranges of two.
func Estimate(values []float64) Limits {
	if len(values) == 0 {
		return Limits{}
	}
	sum, mr := 0.0, 0.0
	for i, v := range values {
		sum += v
		if i > 0 {
			mr += math.Abs(v - values[i-1])
		}
	}
	l := Limits{Center: sum / float64(len(values))}
	if len(values) > 1 {
		l.Sigma = mr / float64(len(values)-1) / 1.128
	}
	return l
}

// Check returns the Western Electric rules, from 1 to 4, that each of
// values breaks against l. A run breaking a rule marks each of its points
// that is on the run's side.
func Check(values []float64, l Limits) [][]int {
	out := make([][]int, len(values))
	if l.Sigma <= 0 {
		return out
	}
	z := make([]float64, len(values))
	for i, v := range values {
		z[i] = (v - l.Center) / l.Sigma
	}
	mark := func(i, rule int) {
		for _, r := range out[i] {
			if r == rule {
				return
			}
		}
		out[i] = append(out[i], rule)
	}
	// runs marks the points beyond sigmas on one side when at least need of
	// a window of n points are.
	runs := func(rule, need, n int, sigmas float64) {
		for end := n - 1; end < len(z); end++ {
			for _, side := range []float64{1, -1} {
				count := 0
				for i := end - n + 1; i <= end; i++ {
					if z[i]*side > sigmas {
						count++
					}
				}
				if count < need {
					continue
				}
				for i := end - n + 1; i <= end; i++ {
					if z[i]*side > sigmas {
						mark(i, rule)
					}
				}
			}
		}
	}
	for i := range z {
		if math.Abs(z[i]) > 3 {
			mark(i, 1)
		}
	}
	runs(2, 2, 3, 2)
	runs(3, 4, 5, 1)
	runs(4, 8, 8, 0)
	return out
}

// View creates the chart of ch. Give it a size, or grow it in its parent.
func View(c *ui.Context, ch Chart) ui.Element {
	t := c.Theme()
	n := len(ch.Values)
	base := ch.Values
	if ch.Baseline > 0 && ch.Baseline < n {
		base = ch.Values[:ch.Baseline]
	}
	l := Estimate(base)
	broken := Check(ch.Values, l)
	lo, hi := l.Center-3.6*l.Sigma, l.Center+3.6*l.Sigma
	for _, v := range ch.Values {
		lo, hi = min(lo, v), max(hi, v)
	}
	y := chart.NewLinear(lo, hi).Nice(6)
	small := t.FontSize - 2
	left, right := t.Space(12), t.Space(22)
	top, bottom := t.Space(5), t.Space(8)
	box := ui.Box(c).MinWidth(t.Space(100)).MinHeight(t.Space(60)).Role(ui.RoleImage).Label("Control chart")
	b := box.LayoutBounds()
	plot := ui.Rect{X: left, Y: top, W: max(1, b.W-left-right), H: max(1, b.H-top-bottom)}
	xAt := func(i int) float32 {
		if n < 2 {
			return plot.X + plot.W/2
		}
		return plot.X + float32(i)*plot.W/float32(n-1)
	}
	yAt := func(v float64) float32 { return plot.Y + (1-y.At(v))*plot.H }
	hot := -1
	if px, _, over := box.PointerPosition(); over && n > 0 {
		hot = max(0, min(n-1, int(math.Round(float64((px-plot.X)/plot.W*float32(max(n-1, 1)))))))
	}
	violations := 0
	for _, r := range broken {
		if len(r) > 0 {
			violations++
		}
	}
	box.Description(fmt.Sprintf("Center %s, limits %s to %s, %d points out of control",
		num(l.Center), num(l.Center-3*l.Sigma), num(l.Center+3*l.Sigma), violations))
	box.Draw(func(p *ui.Painter, r ui.Rect) {
		if r.W != b.W || r.H != b.H {
			c.Invalidate()
		}
		at := func(x, yy float32) (float32, float32) { return r.X + x, r.Y + yy }
		pr := ui.Rect{X: r.X + plot.X, Y: r.Y + plot.Y, W: plot.W, H: plot.H}
		if ch.Baseline > 0 && ch.Baseline < n {
			x0, _ := at(xAt(0), 0)
			x1, _ := at((xAt(ch.Baseline-1)+xAt(ch.Baseline))/2, 0)
			p.Fill(ui.Rect{X: x0, Y: pr.Y, W: x1 - x0, H: pr.H}, t.Text.Alpha(0.04), 0)
			p.Text(x0+t.Space(1), pr.Y+t.Space(1), "Baseline", small, t.TextMuted)
		}
		for _, tk := range y.Ticks(6) {
			_, ty := at(0, yAt(tk.Value))
			p.Line(pr.X, ty, pr.X+pr.W, ty, 1, t.Text.Alpha(0.05))
			w, h := p.MeasureText(0, ui.Span{Text: tk.Label, Size: small})
			p.Text(pr.X-t.Space(2)-w, ty-h/2, tk.Label, small, t.TextMuted)
		}
		if l.Sigma > 0 {
			for k := -3; k <= 3; k++ {
				_, ly := at(0, yAt(l.Center+float64(k)*l.Sigma))
				var path ui.Path
				path.MoveTo(pr.X, ly).LineTo(pr.X+pr.W, ly)
				col, width := t.Text.Alpha(0.15), float32(1)
				switch k {
				case 0:
					col, width = t.Text.Alpha(0.6), 1.5
				case -3, 3:
					col, width = t.Danger, 1.5
					path.Dash(0, 6, 4)
				default:
					path.Dash(0, 2, 3)
				}
				p.StrokePath(&path, width, col)
				name := ""
				switch k {
				case 3:
					name = "UCL "
				case 0:
					name = "CL "
				case -3:
					name = "LCL "
				case 1, -1, 2, -2:
					name = fmt.Sprintf("%+dσ ", k)
				}
				sp := []ui.Span{{Text: name, Size: small, Color: t.TextMuted}}
				if k%3 == 0 {
					sp = append(sp, ui.Span{Text: num(l.Center + float64(k)*l.Sigma), Size: small, Color: t.Text, Features: "tnum"})
				}
				_, h := p.MeasureText(0, sp...)
				p.RichText(pr.X+pr.W+t.Space(2), ly-h/2, 0, sp...)
			}
		}
		pts := make([][2]float32, n)
		for i, v := range ch.Values {
			pts[i][0], pts[i][1] = at(xAt(i), yAt(v))
		}
		p.StrokePolyline(pts, 1.5, t.Text.Alpha(0.45))
		if hot >= 0 {
			p.Line(pts[hot][0], pr.Y, pts[hot][0], pr.Y+pr.H, 1, t.Text.Alpha(0.15))
		}
		for i, pt := range pts {
			d := t.Space(1)
			col := t.Accent
			if len(broken[i]) > 0 {
				col = t.Danger
				d = t.Space(1.4)
				var nums []string
				for _, r := range broken[i] {
					nums = append(nums, strconv.Itoa(r))
				}
				rules := strings.Join(nums, ",")
				w, h := p.MeasureText(0, ui.Span{Text: rules, Size: small - 1, Weight: 700})
				above := pt[1] - d - t.Space(1) - h
				if ch.Values[i] < l.Center {
					above = pt[1] + d + t.Space(1)
				}
				p.RichText(pt[0]-w/2, above, 0, ui.Span{Text: rules, Size: small - 1, Weight: 700, Color: t.Danger})
			}
			if i == hot {
				d += 1.5
			}
			p.Fill(ui.Rect{X: pt[0] - d - 1.5, Y: pt[1] - d - 1.5, W: 2*d + 3, H: 2*d + 3}, t.Background, d+1.5)
			p.Fill(ui.Rect{X: pt[0] - d, Y: pt[1] - d, W: 2 * d, H: 2 * d}, col, d)
		}
		step := max(1, int(math.Ceil(float64(n)/float64(max(1, int(pr.W/t.Space(18)))))))
		for i := 0; i < n && i < len(ch.Labels); i += step {
			w, _ := p.MeasureText(0, ui.Span{Text: ch.Labels[i], Size: small})
			p.Text(pts[i][0]-w/2, pr.Y+pr.H+t.Space(2), ch.Labels[i], small, t.TextMuted)
		}
		if hot >= 0 {
			card(p, t, r, pts[hot], ch, hot, l, broken[hot])
		}
	})
	return box
}

// card paints the card of point i at pt.
func card(p *ui.Painter, t *ui.Theme, r ui.Rect, pt [2]float32, ch Chart, i int, l Limits, rules []int) {
	small := t.FontSize - 2
	v := ch.Values[i]
	value := num(v)
	if ch.Unit != "" {
		value += " " + ch.Unit
	}
	title := "#" + strconv.Itoa(i+1)
	if i < len(ch.Labels) {
		title = ch.Labels[i]
	}
	lines := [][]ui.Span{
		{{Text: title, Size: small, Color: t.TextMuted}},
		{{Text: value, Size: t.FontSize, Weight: 600, Color: t.Text, Features: "tnum"}},
	}
	if l.Sigma > 0 {
		lines = append(lines, []ui.Span{{Text: fmt.Sprintf("%+.2fσ from center", (v-l.Center)/l.Sigma), Size: small, Color: t.TextMuted, Features: "tnum"}})
	}
	if len(rules) > 0 {
		var names []string
		for _, r := range rules {
			names = append(names, ruleNames[r-1])
		}
		lines = append(lines, []ui.Span{{Text: strings.Join(names, "\n"), Size: small, Color: t.Danger}})
	}
	pad := t.Space(2.5)
	w, h := float32(0), float32(0)
	for _, ln := range lines {
		lw, lh := p.MeasureText(t.Space(70), ln...)
		w, h = max(w, lw), h+lh+t.Space(0.5)
	}
	w, h = w+2*pad, h+2*pad
	x, y := pt[0]+t.Space(3), pt[1]-h/2
	if x+w > r.X+r.W {
		x = pt[0] - t.Space(3) - w
	}
	y = max(r.Y, min(y, r.Y+r.H-h))
	box := ui.Rect{X: x, Y: y, W: w, H: h}
	p.Shadow(box, t.Radius+2, 0, 2, 10, 0, ui.RGBA(0, 0, 0, 0.2))
	p.Fill(box, t.Background, t.Radius+2)
	p.Stroke(box, t.Border, t.Radius+2, 1)
	ty := y + pad
	for _, ln := range lines {
		_, lh := p.RichText(x+pad, ty, t.Space(70), ln...)
		ty += lh + t.Space(0.5)
	}
}

var ruleNames = [4]string{
	"Rule 1: beyond 3σ",
	"Rule 2: 2 of 3 beyond 2σ",
	"Rule 3: 4 of 5 beyond 1σ",
	"Rule 4: 8 in a row on one side",
}

// num writes v to the precision a chart reads.
func num(v float64) string {
	return strconv.FormatFloat(v, 'g', 5, 64)
}
