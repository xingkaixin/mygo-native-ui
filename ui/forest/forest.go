// Package forest shows a meta-analysis as a forest plot: a row a study,
// its estimate a square as large as its weight on a line across its 95%
// confidence interval, beside its name, its numbers and its share of the
// weight, on an axis shared by the rows with a line where there is no
// effect. The last row is the pooled estimate, a diamond across its
// interval, with a dashed line up through the studies. Pointing at a row
// highlights it.
//
// The pooled estimate is the fixed-effect one: each study is weighted by
// the inverse of its variance, read from its interval.
//
//	forest.View(c, studies, forest.Ratio)
package forest

import (
	"fmt"
	"math"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/chart"
)

// Measure is what the estimates measure, which sets where there is no
// effect and the axis.
type Measure uint8

const (
	// Difference is a measure of a difference, as a mean difference: no
	// effect is 0, on a linear axis.
	Difference Measure = iota
	// Ratio is a measure of a ratio, as an odds or a hazard ratio: no
	// effect is 1, on a log axis.
	Ratio
)

// Study is a study's estimate and the 95% confidence interval around it.
type Study struct {
	Name             string
	Estimate, Lo, Hi float64
}

// z is how many standard errors a 95% interval reaches either side.
const z = 1.959964

// row is a row of the plot, in the units of the axis: logs for a ratio.
type row struct {
	study                    Study
	est, lo, hi, weight      float64
	pooled                   bool
	label, numbers, weighted string
}

// View creates the forest plot of studies, which estimate m.
func View(c *ui.Context, studies []Study, m Measure) ui.Element {
	t := c.Theme()
	f := func(v float64) float64 { return v }
	if m == Ratio {
		f = math.Log
	}
	none := f(map[Measure]float64{Difference: 0, Ratio: 1}[m])
	rows := make([]row, 0, len(studies)+1)
	sum, sumW := 0.0, 0.0
	for _, s := range studies {
		r := row{study: s, est: f(s.Estimate), lo: f(s.Lo), hi: f(s.Hi)}
		if se := (r.hi - r.lo) / (2 * z); se > 0 {
			r.weight = 1 / (se * se)
		}
		sum, sumW = sum+r.weight*r.est, sumW+r.weight
		rows = append(rows, r)
	}
	if sumW > 0 {
		est, se := sum/sumW, 1/math.Sqrt(sumW)
		back := func(v float64) float64 { return v }
		if m == Ratio {
			back = math.Exp
		}
		pooled := Study{Name: "Pooled", Estimate: back(est), Lo: back(est - z*se), Hi: back(est + z*se)}
		rows = append(rows, row{study: pooled, est: est, lo: est - z*se, hi: est + z*se, weight: sumW, pooled: true})
	}
	lo, hi := none, none
	most := 0.0
	for i := range rows {
		r := &rows[i]
		lo, hi = min(lo, r.lo), max(hi, r.hi)
		if !r.pooled {
			most = max(most, r.weight)
		}
		r.numbers = fmt.Sprintf("%.2f [%.2f, %.2f]", r.study.Estimate, r.study.Lo, r.study.Hi)
		r.weighted = fmt.Sprintf("%.1f%%", 100*r.weight/sumW)
		r.label = fmt.Sprintf("%s: %.2f, 95%% interval %.2f to %.2f, weight %s", r.study.Name, r.study.Estimate, r.study.Lo, r.study.Hi, r.weighted)
	}
	pad := (hi - lo) * 0.05
	scale, ticks := axis(m, lo-pad, hi+pad)
	pooledAt := float32(-1)
	if sumW > 0 {
		pooledAt = scale.At(rows[len(rows)-1].est)
	}
	noneAt := scale.At(none)

	nameW, numW, weightW, rowH := t.Space(40), t.Space(36), t.Space(14), t.Space(7)
	header := map[Measure]string{Difference: "Difference [95% CI]", Ratio: "Ratio [95% CI]"}[m]
	return ui.Column(c).Label("Forest plot").Children(func() {
		ui.Row(c).Height(rowH).AlignItems(ui.Center).PaddingX(t.Space(2)).Children(func() {
			muted := func(s string, w float32, a ui.Align) {
				ui.Text(c, s).FontSize(t.FontSize - 1).Bold().TextColor(t.TextMuted).Width(w).TextAlign(a).NoWrap()
			}
			muted("Study", nameW, ui.Start)
			ui.Box(c).Grow(1)
			muted(header, numW, ui.End)
			muted("Weight", weightW, ui.End)
		})
		for i, r := range rows {
			line := ui.Row(c.Key(i)).Height(rowH).AlignItems(ui.Center).PaddingX(t.Space(2)).Label(r.label)
			if r.pooled {
				line.BorderWidth(1, 0, 0, 0).BorderColor(t.Border)
			}
			if line.Hovered() {
				line.Background(t.SurfaceHover)
			}
			line.Children(func() {
				name := ui.Text(c, r.study.Name).Width(nameW).SingleLine()
				if r.pooled {
					name.Bold()
				}
				ui.Box(c).Grow(1).AlignSelf(ui.Stretch).Draw(func(p *ui.Painter, b ui.Rect) {
					x := func(v float64) float32 { return chart.X(b, scale.At(v)) }
					nx := chart.X(b, noneAt)
					p.Fill(ui.Rect{X: nx, Y: b.Y, W: 1, H: b.H}, t.TextMuted, 0)
					if pooledAt >= 0 && !r.pooled {
						px := chart.X(b, pooledAt)
						for y := b.Y; y < b.Y+b.H; y += 5 {
							p.Fill(ui.Rect{X: px, Y: y, W: 1, H: min(3, b.Y+b.H-y)}, t.Danger.Alpha(0.7), 0)
						}
					}
					cy := b.Y + b.H/2
					p.Clip(b, 0, func() {
						if r.pooled {
							h := b.H * 0.32
							var d ui.Path
							d.MoveTo(x(r.lo), cy)
							d.LineTo(x(r.est), cy-h)
							d.LineTo(x(r.hi), cy)
							d.LineTo(x(r.est), cy+h)
							d.Close()
							p.FillPath(&d, t.Danger)
							return
						}
						p.Fill(ui.Rect{X: x(r.lo), Y: cy - 0.75, W: x(r.hi) - x(r.lo), H: 1.5}, t.Text, 0)
						side := t.Space(1.5) + t.Space(3.5)*float32(math.Sqrt(r.weight/most))
						p.Fill(ui.Rect{X: x(r.est) - side/2, Y: cy - side/2, W: side, H: side}, t.Accent, 0)
					})
				})
				num := ui.Text(c, r.numbers).Width(numW).TextAlign(ui.End).FontFeatures("tnum").NoWrap()
				wt := ui.Text(c, r.weighted).Width(weightW).TextAlign(ui.End).FontFeatures("tnum").NoWrap()
				if r.pooled {
					num.Bold()
					wt.Bold()
				}
			})
		}
		ui.Row(c).Height(t.Space(8)).PaddingX(t.Space(2)).Children(func() {
			ui.Box(c).Width(nameW)
			ui.Box(c).Grow(1).AlignSelf(ui.Stretch).Draw(func(p *ui.Painter, b ui.Rect) {
				fs := t.FontSize - 2
				p.Fill(ui.Rect{X: b.X, Y: b.Y, W: b.W, H: 1}, t.TextMuted, 0)
				for _, tk := range ticks {
					tx := chart.X(b, tk.At)
					p.Fill(ui.Rect{X: tx, Y: b.Y, W: 1, H: t.Space(1)}, t.TextMuted, 0)
					w, _ := p.MeasureText(0, ui.Span{Text: tk.Label, Size: fs})
					p.Text(tx-w/2, b.Y+t.Space(1.5), tk.Label, fs, t.TextMuted)
				}
			})
			ui.Box(c).Width(numW + weightW)
		})
	})
}

// axis returns the scale from lo to hi, in the units of m's axis, and its
// ticks labeled in m's own units.
func axis(m Measure, lo, hi float64) (chart.Linear, []chart.Tick) {
	if m == Difference {
		s := chart.NewLinear(lo, hi).Nice(6)
		ticks := s.Ticks(6)
		for i := range ticks {
			ticks[i].Label = fmt.Sprint(ticks[i].Value)
		}
		return s, ticks
	}
	s := chart.NewLinear(lo, hi)
	var ticks []chart.Tick
	for _, v := range []float64{0.01, 0.02, 0.05, 0.1, 0.2, 0.5, 1, 2, 5, 10, 20, 50, 100} {
		if l := math.Log(v); l >= lo && l <= hi {
			ticks = append(ticks, chart.Tick{At: s.At(l), Value: v, Label: fmt.Sprint(v)})
		}
	}
	return s, ticks
}
