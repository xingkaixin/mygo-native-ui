// Package pareto ranks causes by how often they happen, as a Pareto
// chart: a bar a cause from the largest down, against counts up to their
// total on the left, and a line of the cumulative share over them against
// percentages on the right, with a dashed line at 80%. The vital few, the
// causes up to the one where the cumulative line reaches 80%, show in the
// accent and the rest muted. Pointing at a bar, or the arrows once the
// chart has the focus, reads out a cause's count, share and cumulative
// share.
//
//	pareto.View(c, causes)
package pareto

import (
	"cmp"
	"fmt"
	"slices"
	"strconv"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/chart"
)

// Cause is something that happened, and how many times.
type Cause struct {
	Name  string
	Count float64
}

// vital is the cumulative share up to which causes are the vital few.
const vital = 0.8

// View creates the chart of causes, which it sorts by count.
func View(c *ui.Context, causes []Cause) ui.Element {
	t := c.Theme()
	sorted := slices.Clone(causes)
	slices.SortStableFunc(sorted, func(a, b Cause) int { return cmp.Compare(b.Count, a.Count) })
	total := 0.0
	for _, cs := range sorted {
		total += cs.Count
	}
	cum := make([]float64, len(sorted))
	run := 0.0
	few := len(sorted)
	for i, cs := range sorted {
		run += cs.Count
		cum[i] = run / max(total, 1e-12)
		if cum[i] >= vital && few == len(sorted) {
			few = i + 1
		}
	}
	names := make([]string, len(sorted))
	for i, cs := range sorted {
		names[i] = cs.Name
	}
	x := chart.NewBand(names...).Padding(0.25, 0.1)
	var left, right []chart.Tick
	for k := range 6 {
		f := float32(k) / 5
		left = append(left, chart.Tick{At: f, Value: total * float64(f), Label: strconv.FormatFloat(total*float64(f), 'f', 0, 64)})
		right = append(right, chart.Tick{At: f, Value: float64(f), Label: fmt.Sprintf("%.0f%%", 100*f)})
	}
	root := ui.Column(c)
	// Names wrap to the width of a band in the last frame, and the chart
	// builds again when it changes.
	width := root.LayoutBounds().W
	root.Draw(func(_ *ui.Painter, r ui.Rect) {
		if r.W != width {
			c.Invalidate()
		}
	})
	nameW := max(width/float32(max(len(sorted), 1))*0.9, t.Space(10))
	small := t.FontSize - 2
	label := func(side chart.Side, tk chart.Tick) {
		if side != chart.Bottom {
			ui.Text(c, tk.Label).FontSize(small).TextColor(t.TextMuted).NoWrap()
			return
		}
		ui.Text(c, tk.Label).FontSize(small).TextColor(t.TextMuted).Width(nameW).TextAlign(ui.Center).MaxLines(2).MarginY(t.Space(1))
	}
	xs := make([]float32, len(sorted))
	for i := range sorted {
		xs[i] = x.At(i)
	}

	plot := func(area ui.Element) {
		area.Label("Pareto chart")
		sel := ui.Local(area, "pareto.selected", func() int { return -1 })
		chart.Select(c, area, sel, xs, func(i int) string {
			return fmt.Sprintf("%s: %s, %.0f%% cumulative", sorted[i].Name, strconv.FormatFloat(sorted[i].Count, 'f', -1, 64), 100*cum[i])
		})
		i := *sel
		area.Draw(func(p *ui.Painter, r ui.Rect) {
			for _, tk := range left {
				p.Line(r.X, chart.Y(r, tk.At), r.X+r.W, chart.Y(r, tk.At), 1, t.Border.Alpha(0.6))
			}
			for k, cs := range sorted {
				band := x.Band(k)
				h := float32(cs.Count / max(total, 1e-12))
				color := t.Accent
				if k >= few {
					color = t.TextMuted.Alpha(0.35)
				}
				if k == i {
					color = color.Mix(t.Text, 0.25)
				}
				x0, x1 := chart.X(r, band.Lo), chart.X(r, band.Hi)
				p.Fill(ui.Rect{X: x0, Y: chart.Y(r, h), W: x1 - x0, H: h * r.H}, color, 2)
			}
			vy := chart.Y(r, vital)
			for dx := r.X; dx < r.X+r.W; dx += 7 {
				p.Fill(ui.Rect{X: dx, Y: vy, W: min(4, r.X+r.W-dx), H: 1}, t.TextMuted, 0)
			}
			pts := make([][2]float32, len(sorted))
			for k := range sorted {
				pts[k] = [2]float32{chart.X(r, xs[k]), chart.Y(r, float32(cum[k]))}
			}
			p.StrokePolyline(pts, 2, t.Danger)
			for k, pt := range pts {
				var dot ui.Path
				radius := float32(3)
				if k == i {
					radius = 5
				}
				dot.Circle(pt[0], pt[1], radius)
				p.FillPath(&dot, t.Danger)
				p.StrokePath(&dot, 1.5, t.Background)
			}
		})
		if i < 0 || i >= len(sorted) {
			return
		}
		h := ui.Center
		switch {
		case xs[i] < 0.2:
			h = ui.Start
		case xs[i] > 0.8:
			h = ui.End
		}
		chart.Anchor(c, xs[i], float32(cum[i]), h, ui.Start, func() {
			ui.Column(c).Gap(t.Space(0.5)).Padding(t.Space(2), t.Space(2.5)).Margin(t.Space(2)).
				Radius(t.Space(1.5)).Background(t.Background).Border(1, t.Border).
				Shadow(0, 2, 8, 0, ui.RGBA(0, 0, 0, 0.12)).PassThrough().Children(func() {
				ui.Text(c, sorted[i].Name).Bold().NoWrap()
				for _, line := range []string{
					fmt.Sprintf("%s times, %.1f%%", strconv.FormatFloat(sorted[i].Count, 'f', -1, 64), 100*sorted[i].Count/total),
					fmt.Sprintf("%.1f%% cumulative", 100*cum[i]),
				} {
					ui.Text(c, line).FontSize(t.FontSize - 1).TextColor(t.TextMuted).FontFeatures("tnum").NoWrap()
				}
			})
		})
	}
	return root.Children(func() {
		chart.Plot(c, chart.Axes{Left: left, Right: right, Bottom: x.Ticks(), Label: label}, plot).Grow(1).Gap(t.Space(2))
	})
}
