// Package runs compares training runs, as W&B and TensorBoard chart them:
// a metric over the steps, a line a run in its color, smoothed with an
// exponential moving average over the raw values, which show faint
// behind. The slider sets the smoothing, from none to 0.99, and Log puts
// the metric on a log scale. The pointer over the chart reads out each
// run's value at the step under it. The legend below names the runs with
// their last values, and a click on one hides or shows its line.
//
//	runs.View(c, &app.loss, "Loss", app.runs).Height(360)
package runs

import (
	"math"
	"slices"
	"strconv"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/chart"
)

// Run is a training run: its name, the color of its line, zero for one of
// its own, and the metric's values at its steps, in order.
type Run struct {
	Name   string
	Color  ui.Color
	Steps  []float64
	Values []float64
}

var palette = []ui.Color{
	ui.Hex("#4C8DF6"), ui.Hex("#F2994A"), ui.Hex("#27AE60"), ui.Hex("#EB5757"),
	ui.Hex("#9B51E0"), ui.Hex("#2BB0C4"), ui.Hex("#D4A72C"), ui.Hex("#E36BB0"),
}

// State is what a chart keeps: the smoothing, from 0 to 0.99, whether the
// metric is on a log scale, and the runs hidden. Set Smoothing and Log for
// how the chart first shows; keep it in the app's state, one for each
// chart.
type State struct {
	Smoothing float64
	Log       bool
	hidden    map[string]bool
}

// View creates the chart of metric, named so, over runs.
func View(c *ui.Context, s *State, metric string, runs []Run) ui.Element {
	t := c.Theme()
	if s.hidden == nil {
		s.hidden = map[string]bool{}
	}
	s.Smoothing = max(0, min(0.99, s.Smoothing))
	type line struct {
		run      Run
		color    ui.Color
		smoothed []float64
	}
	var shown []line
	var all []line
	for i, r := range runs {
		col := r.Color
		if col == (ui.Color{}) {
			col = palette[i%len(palette)]
		}
		l := line{r, col, ema(r.Values, s.Smoothing)}
		all = append(all, l)
		if !s.hidden[r.Name] {
			shown = append(shown, l)
		}
	}
	at := func(v float64) float64 {
		if s.Log {
			return math.Log10(v)
		}
		return v
	}
	var xs, ys []float64
	for _, l := range shown {
		xs = append(xs, l.run.Steps...)
		for _, v := range l.run.Values {
			if !s.Log || v > 0 {
				ys = append(ys, at(v))
			}
		}
	}
	x := chart.NewLinear(chart.Extent(xs...))
	ylo, yhi := chart.Extent(ys...)
	var y chart.Linear
	var yticks []chart.Tick
	if s.Log {
		y = chart.NewLinear(math.Floor(ylo), math.Ceil(max(yhi, ylo+1)))
		for e := y.Lo; e <= y.Hi; e++ {
			yticks = append(yticks, chart.Tick{At: y.At(e), Value: e, Label: number(math.Pow(10, e))})
		}
	} else {
		y = chart.NewLinear(ylo, yhi).Nice(5)
		yticks = y.Ticks(5)
		for i := range yticks {
			yticks[i].Label = number(yticks[i].Value)
		}
	}
	xticks := x.Ticks(6)
	for i := range xticks {
		xticks[i].Label = number(xticks[i].Value)
	}

	return ui.Column(c).Gap(t.Space(3)).Children(func() {
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, metric).Bold().Grow(1)
			ui.Text(c, "Smoothing").TextColor(t.TextMuted)
			ui.Slider(c, &s.Smoothing, 0, 0.99).Width(t.Space(30)).Label("Smoothing")
			ui.Text(c, strconv.FormatFloat(s.Smoothing, 'f', 2, 64)).FontFeatures("tnum").Width(t.Space(9))
			ui.Toggle(c, &s.Log, "Log")
		})
		chart.Plot(c, chart.Axes{Left: yticks, Bottom: xticks}, func(area ui.Element) {
			area.Label(metric)
			px, _, over := area.PointerPosition()
			b := area.Bounds()
			step, hover := 0.0, over && b.W > 0 && len(shown) > 0
			if hover {
				step = x.Invert(px / b.W)
			}
			type reading struct {
				name  string
				color ui.Color
				value float64
				step  float64
			}
			var readings []reading
			if hover {
				for _, l := range shown {
					if i := nearest(l.run.Steps, step); i >= 0 {
						readings = append(readings, reading{l.run.Name, l.color, l.smoothed[i], l.run.Steps[i]})
					}
				}
				slices.SortStableFunc(readings, func(a, b reading) int {
					switch {
					case a.value > b.value:
						return -1
					case a.value < b.value:
						return 1
					}
					return 0
				})
			}
			area.Draw(func(p *ui.Painter, r ui.Rect) {
				for _, tk := range yticks {
					p.Line(r.X, chart.Y(r, tk.At), r.X+r.W, chart.Y(r, tk.At), 1, t.Border.Alpha(0.6))
				}
				p.Clip(r, 0, func() {
					for pass := range 2 {
						for _, l := range shown {
							vs, width, col := l.run.Values, float32(1), l.color.Alpha(0.25)
							if pass == 1 {
								vs, width, col = l.smoothed, 1.75, l.color
							}
							if pass == 0 && s.Smoothing == 0 {
								continue
							}
							var fx, fy []float32
							for i, v := range vs {
								if s.Log && v <= 0 {
									continue
								}
								fx, fy = append(fx, x.At(l.run.Steps[i])), append(fy, y.At(at(v)))
							}
							fx, fy = chart.MinMax(nil, nil, fx, fy, int(r.W*p.Scale()))
							p.StrokePath(chart.Line(r, fx, fy, chart.CurveLinear), width, col)
						}
					}
				})
				if hover {
					hx := r.X + px
					p.Line(hx, r.Y, hx, r.Y+r.H, 1, t.TextMuted.Alpha(0.5))
					for _, rd := range readings {
						if s.Log && rd.value <= 0 {
							continue
						}
						cx, cy := chart.X(r, x.At(rd.step)), chart.Y(r, y.At(at(rd.value)))
						var dot ui.Path
						dot.Circle(cx, cy, 3.5)
						p.FillPath(&dot, rd.color)
						p.StrokePath(&dot, 1.5, t.Background)
					}
				}
			})
			if hover && len(readings) > 0 {
				f := px / b.W
				h := ui.Start
				if f > 0.6 {
					h = ui.End
				}
				chart.Anchor(c, f, 1, h, ui.Start, func() {
					ui.Column(c).Gap(t.Space(1)).Padding(t.Space(2), t.Space(2.5)).Margin(t.Space(1), t.Space(2)).
						Radius(t.Space(1.5)).Background(t.Background).Border(1, t.Border).
						Shadow(0, 2, 8, 0, ui.RGBA(0, 0, 0, 0.12)).PassThrough().Children(func() {
						ui.Text(c, "Step "+number(readings[0].step)).FontSize(t.FontSize - 2).TextColor(t.TextMuted).NoWrap()
						for i, rd := range readings {
							ui.Row(c.Key(i)).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
								ui.Box(c).Size(t.Space(2), t.Space(2)).Radius(t.Space(1)).Background(rd.color)
								ui.Text(c, rd.name).FontSize(t.FontSize - 1).NoWrap().Grow(1)
								ui.Text(c, value(rd.value)).FontSize(t.FontSize - 1).FontFeatures("tnum").NoWrap()
							})
						}
					})
				})
			}
		}).Grow(1).Gap(t.Space(2))
		ui.Row(c).Gap(t.Space(1)).Wrap().Role(ui.RoleGroup).Label("Runs").Children(func() {
			for i, l := range all {
				on := !s.hidden[l.run.Name]
				chip := ui.ButtonBase(c.Key(i)).Gap(t.Space(2)).Padding(t.Space(1), t.Space(2)).Radius(t.Space(1.5)).
					Label(l.run.Name).Checked(on)
				if chip.Hovered() {
					chip.Background(t.SurfaceHover)
				}
				if chip.Clicked() {
					s.hidden[l.run.Name] = on
				}
				if !on {
					chip.Opacity(0.45)
				}
				chip.Children(func() {
					ui.Box(c).Size(t.Space(3), t.Space(1)).Radius(t.Space(0.5)).Background(l.color)
					ui.Text(c, l.run.Name)
					if n := len(l.smoothed); n > 0 {
						ui.Text(c, value(l.smoothed[n-1])).FontSize(t.FontSize - 1).FontFeatures("tnum").TextColor(t.TextMuted)
					}
				})
			}
		})
	})
}

// ema smooths vs with an exponential moving average of weight w, debiased
// as TensorBoard's so that the first values are not pulled toward 0.
func ema(vs []float64, w float64) []float64 {
	out := make([]float64, len(vs))
	last, n := 0.0, 0
	for i, v := range vs {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			out[i] = v
			continue
		}
		last = last*w + (1-w)*v
		n++
		out[i] = last
		if w > 0 {
			out[i] = last / (1 - math.Pow(w, float64(n)))
		}
	}
	return out
}

// nearest returns the index of the step nearest to at in steps, in order,
// or -1 for none.
func nearest(steps []float64, at float64) int {
	i, _ := slices.BinarySearch(steps, at)
	switch {
	case len(steps) == 0:
		return -1
	case i == len(steps):
		return i - 1
	case i > 0 && at-steps[i-1] < steps[i]-at:
		return i - 1
	}
	return i
}

// number writes v for an axis: 1.5k, 20k, 1M, 0.01.
func number(v float64) string {
	a := math.Abs(v)
	switch {
	case a >= 1e6:
		return strconv.FormatFloat(v/1e6, 'g', 3, 64) + "M"
	case a >= 1e3:
		return strconv.FormatFloat(v/1e3, 'g', 3, 64) + "k"
	}
	return strconv.FormatFloat(v, 'g', 3, 64)
}

// value writes a metric's value with four significant digits.
func value(v float64) string {
	return strconv.FormatFloat(v, 'g', 4, 64)
}
