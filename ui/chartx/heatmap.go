package chartx

import (
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/chart"
)

// Heatmap paints a heatmap in the area of a plot: a cell where each band
// of x meets each of y, cell (i, j) in color(i, j), and none where it is
// ui.Transparent. Padding the scales leaves gaps between the cells. The
// cells are painted rather than built, so thousands of them cost little.
// Heatmap returns the cell under the pointer, which it outlines, for a
// tooltip by it:
//
//	x, y := chart.NewBand(hours...), chart.NewBand(days...)
//	color := chartx.NewSequential(0, most, chartx.Viridis...)
//	chart.Plot(c, chart.Axes{Left: y.Ticks(), Bottom: x.Ticks()}, func(area ui.Element) {
//		i, j, ok := chartx.Heatmap(c, area, x, y, func(i, j int) ui.Color {
//			return color.At(counts[j][i])
//		})
//		if ok {
//			chart.Anchor(c, x.At(i), y.Band(j).Hi, ui.Center, ui.End, func() { ... })
//		}
//	})
func Heatmap(c *ui.Context, area ui.Element, x, y chart.Band, color func(i, j int) ui.Color) (i, j int, ok bool) {
	t := c.Theme()
	if px, py, over := area.PointerPosition(); over {
		if r := area.Bounds(); r.W > 0 && r.H > 0 {
			var okX, okY bool
			i, okX = x.Index(px / r.W)
			j, okY = y.Index(1 - py/r.H)
			ok = okX && okY
		}
	}
	hovered := ok
	area.Draw(func(p *ui.Painter, r ui.Rect) {
		cell := func(i, j int) ui.Rect {
			xs, ys := x.Band(i), y.Band(j)
			x0, y0 := chart.X(r, xs.Lo), chart.Y(r, ys.Hi)
			return ui.Rect{X: x0, Y: y0, W: chart.X(r, xs.Hi) - x0, H: chart.Y(r, ys.Lo) - y0}
		}
		for b := range y.Labels {
			for a := range x.Labels {
				if col := color(a, b); col.A > 0 {
					p.Fill(cell(a, b), col, 0)
				}
			}
		}
		if hovered {
			p.Stroke(cell(i, j), t.Text, 0, 1.5)
		}
	})
	return i, j, ok
}
