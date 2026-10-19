package chartx

import (
	"math"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/chart"
)

// Sparkline builds a sparkline, as Edward Tufte named them: a line of
// values small enough to sit in a sentence or a table's cell, without
// axes, the area below it tinted and its last value marked with a dot. It
// is 80 by 20 DIPs, or as Size sets, the line spanning its height from the
// least value to the greatest:
//
//	chartx.Sparkline(c, app.latency, t.Accent).Size(120, 24)
//
// NaN values break the line, as in chart.Line.
func Sparkline(c *ui.Context, values []float64, color ui.Color) ui.Element {
	y := chart.NewLinear(chart.Extent(values...))
	n := len(values)
	xs, ys := make([]float32, n), make([]float32, n)
	for i, v := range values {
		xs[i] = 0.5
		if n > 1 {
			xs[i] = float32(i) / float32(n-1)
		}
		ys[i] = y.At(v)
	}
	return ui.Box(c).Size(80, 20).Draw(func(p *ui.Painter, r ui.Rect) {
		if n == 0 {
			return
		}
		// Inset by the dot's radius, which would cross the edges at the
		// ends and extremes.
		const dot = 2
		r = ui.Rect{X: r.X + dot, Y: r.Y + dot, W: r.W - 2*dot, H: r.H - 2*dot}
		p.FillPath(chart.Area(r, xs, nil, ys, chart.CurveLinear), color.Alpha(0.15))
		p.StrokePath(chart.Line(r, xs, ys, chart.CurveLinear), 1.5, color)
		if last := values[n-1]; !math.IsNaN(last) {
			x, y := chart.X(r, xs[n-1]), chart.Y(r, ys[n-1])
			p.Fill(ui.Rect{X: x - dot, Y: y - dot, W: 2 * dot, H: 2 * dot}, color, dot)
		}
	})
}
