// Package ecdf compares distributions as empirical cumulative
// distribution functions: a line a series, stepping up by its share at
// each of its values, so that every value shows and no bins hide any.
// Pointing along the plot reads out each series' share at or below the
// value under the pointer. With QQ on, each series' values go up against
// the quantiles of a normal distribution across, with a dashed line
// through its quartiles: points on the line are normal, and bends away
// from it at the ends show tails heavier or lighter than normal.
//
//	ecdf.View(c, &app.dist, "Latency, ms", series)
package ecdf

import (
	"fmt"
	"math"
	"slices"
	"strconv"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/chart"
)

// Series is a sample of values, named, and drawn in Color, zero for one
// of its own.
type Series struct {
	Name   string
	Color  ui.Color
	Values []float64
}

var palette = []ui.Color{ui.Hex("#4C8DF6"), ui.Hex("#F2994A"), ui.Hex("#27AE60"), ui.Hex("#9B51E0"), ui.Hex("#EB5757")}

// State is what a view keeps: whether it shows the QQ plot. Set QQ for
// how the view first shows; keep it in the app's state, one for each view.
type State struct {
	QQ bool
}

// quantile returns the value a share p of sorted is at or below.
func quantile(sorted []float64, p float64) float64 {
	f := p * float64(len(sorted)-1)
	i := int(f)
	if i+1 >= len(sorted) {
		return sorted[len(sorted)-1]
	}
	return sorted[i] + (sorted[i+1]-sorted[i])*(f-float64(i))
}

// normal returns the quantile p of the standard normal distribution.
func normal(p float64) float64 { return math.Sqrt2 * math.Erfinv(2*p-1) }

func number(v float64) string { return strconv.FormatFloat(v, 'g', 4, 64) }

// View creates the plot of series, whose values are of what name says.
func View(c *ui.Context, s *State, name string, series []Series) ui.Element {
	t := c.Theme()
	type line struct {
		Series
		color  ui.Color
		sorted []float64
	}
	var lines []line
	var all []float64
	for i, sr := range series {
		if len(sr.Values) == 0 {
			continue
		}
		l := line{sr, sr.Color, slices.Sorted(slices.Values(sr.Values))}
		if l.color == (ui.Color{}) {
			l.color = palette[i%len(palette)]
		}
		lines = append(lines, l)
		all = append(all, sr.Values...)
	}
	vals := chart.NewLinear(chart.Extent(all...)).Nice(6)
	most := 0
	for _, l := range lines {
		most = max(most, len(l.sorted))
	}
	edge := math.Ceil(max(math.Abs(normal(0.5/float64(max(most, 1)))), 1))
	x, y := vals, chart.NewLinear(0, 1)
	if s.QQ {
		x, y = chart.NewLinear(-edge, edge), vals
	}
	xticks, yticks := x.Ticks(6), y.Ticks(5)
	if !s.QQ {
		for i := range yticks {
			yticks[i].Label = fmt.Sprintf("%.0f%%", 100*yticks[i].Value)
		}
	}
	xname, yname := name, "Share at or below"
	if s.QQ {
		xname, yname = "Normal quantile", name
	}

	return ui.Column(c).Gap(t.Space(3)).Children(func() {
		ui.Row(c).Gap(t.Space(5)).AlignItems(ui.Center).Children(func() {
			ui.Row(c).Gap(t.Space(5)).Wrap().Grow(1).Children(func() {
				for i, l := range lines {
					ui.Row(c.Key(i)).Gap(t.Space(1.5)).AlignItems(ui.Center).Children(func() {
						ui.Box(c).Size(t.Space(3), t.Space(1)).Radius(t.Space(0.5)).Background(l.color)
						ui.Text(c, l.Name)
						ui.Text(c, fmt.Sprintf("n %d · median %s", len(l.sorted), number(quantile(l.sorted, 0.5)))).
							TextColor(t.TextMuted).FontFeatures("tnum")
					})
				}
			})
			ui.Toggle(c, &s.QQ, "QQ")
		})
		ui.Text(c, yname).FontSize(t.FontSize - 2).TextColor(t.TextMuted)
		chart.Plot(c, chart.Axes{Left: yticks, Bottom: xticks}, func(area ui.Element) {
			area.Label(name)
			px, _, over := area.PointerPosition()
			b := area.Bounds()
			at := math.NaN()
			if over && !s.QQ && b.W > 0 {
				at = x.Invert(px / b.W)
			}
			qq := s.QQ
			area.Draw(func(p *ui.Painter, r ui.Rect) {
				for _, tk := range yticks {
					p.Line(r.X, chart.Y(r, tk.At), r.X+r.W, chart.Y(r, tk.At), 1, t.Border.Alpha(0.6))
				}
				for _, tk := range xticks {
					p.Line(chart.X(r, tk.At), r.Y, chart.X(r, tk.At), r.Y+r.H, 1, t.Border.Alpha(0.6))
				}
				pt := func(vx, vy float64) [2]float32 { return [2]float32{chart.X(r, x.At(vx)), chart.Y(r, y.At(vy))} }
				p.Clip(r, 0, func() {
					for _, l := range lines {
						n := float64(len(l.sorted))
						if qq {
							q1, q3 := quantile(l.sorted, 0.25), quantile(l.sorted, 0.75)
							slope := (q3 - q1) / (normal(0.75) - normal(0.25))
							mid := q1 - slope*normal(0.25)
							a, b := pt(-edge, mid-slope*edge), pt(edge, mid+slope*edge)
							for k := float32(0); k < 1; k += 0.02 {
								p.StrokePolyline([][2]float32{
									{a[0] + (b[0]-a[0])*k, a[1] + (b[1]-a[1])*k},
									{a[0] + (b[0]-a[0])*(k+0.01), a[1] + (b[1]-a[1])*(k+0.01)},
								}, 1.5, l.color.Alpha(0.7))
							}
							for i, v := range l.sorted {
								q := pt(normal((float64(i)+0.5)/n), v)
								p.Fill(ui.Rect{X: q[0] - 2, Y: q[1] - 2, W: 4, H: 4}, l.color.Alpha(0.7), 2)
							}
							continue
						}
						steps := [][2]float32{pt(x.Lo, 0)}
						for i, v := range l.sorted {
							steps = append(steps, pt(v, float64(i)/n), pt(v, float64(i+1)/n))
						}
						steps = append(steps, pt(x.Hi, 1))
						p.StrokePolyline(steps, 2, l.color)
					}
				})
				if math.IsNaN(at) {
					return
				}
				hx := chart.X(r, x.At(at))
				p.Line(hx, r.Y, hx, r.Y+r.H, 1, t.TextMuted.Alpha(0.5))
				for _, l := range lines {
					share := float64(atOrBelow(l.sorted, at)) / float64(len(l.sorted))
					var dot ui.Path
					dot.Circle(hx, chart.Y(r, y.At(share)), 4)
					p.FillPath(&dot, l.color)
					p.StrokePath(&dot, 1.5, t.Background)
				}
			})
			if math.IsNaN(at) {
				return
			}
			f := x.At(at)
			h := ui.Start
			if f > 0.6 {
				h = ui.End
			}
			chart.Anchor(c, f, 0, h, ui.End, func() {
				ui.Column(c).Gap(t.Space(1)).Padding(t.Space(2), t.Space(2.5)).Margin(t.Space(2)).
					Radius(t.Space(1.5)).Background(t.Background).Border(1, t.Border).
					Shadow(0, 2, 8, 0, ui.RGBA(0, 0, 0, 0.12)).PassThrough().Children(func() {
					ui.Text(c, "At or below "+number(at)).FontSize(t.FontSize - 2).TextColor(t.TextMuted).NoWrap()
					for i, l := range lines {
						share := float64(atOrBelow(l.sorted, at)) / float64(len(l.sorted))
						ui.Row(c.Key(i)).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
							ui.Box(c).Size(t.Space(2), t.Space(2)).Radius(t.Space(1)).Background(l.color)
							ui.Text(c, l.Name).FontSize(t.FontSize - 1).NoWrap().Grow(1)
							ui.Text(c, fmt.Sprintf("%.1f%%", 100*share)).FontSize(t.FontSize - 1).FontFeatures("tnum").NoWrap()
						})
					}
				})
			})
		}).Height(t.Space(80)).Gap(t.Space(2))
		ui.Text(c, xname).FontSize(t.FontSize - 2).TextColor(t.TextMuted).AlignSelf(ui.End)
	})
}

// atOrBelow returns how many of sorted are at or below v.
func atOrBelow(sorted []float64, v float64) int {
	i, found := slices.BinarySearch(sorted, v)
	for found && i < len(sorted) && sorted[i] == v {
		i++
	}
	return i
}
