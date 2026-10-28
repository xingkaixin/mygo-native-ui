// Package survival plots Kaplan–Meier survival curves: the share of a
// group still free of an event over time, a line a group stepping down
// where events happen, in a shaded 95% confidence band, with a tick where
// a subject left the study without one. The legend gives each group's
// median survival, and the numbers under each time on the axis how many
// of each group were still at risk then. Pointing along the plot reads
// out each group's survival at the time under the pointer.
//
//	survival.View(c, groups, "Months")
package survival

import (
	"fmt"
	"math"
	"slices"
	"strconv"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/chart"
)

// Subject is a subject of a study: when it had the event, or when it was
// last seen without it.
type Subject struct {
	Time  float64
	Event bool
}

// Group is a group of subjects, named, and drawn in Color, zero for one
// of its own.
type Group struct {
	Name     string
	Color    ui.Color
	Subjects []Subject
}

var palette = []ui.Color{ui.Hex("#4C8DF6"), ui.Hex("#F2994A"), ui.Hex("#27AE60"), ui.Hex("#9B51E0"), ui.Hex("#EB5757")}

// step is the survival from a time on, and its 95% interval.
type step struct{ time, s, lo, hi float64 }

// curve is a group's estimate: its steps, the first at time 0, the
// times subjects left without the event with the survival then, and the
// subjects' times in order.
type curve struct {
	steps    []step
	censored [][2]float64
	times    []float64
	end      float64
}

func estimate(subjects []Subject) curve {
	sorted := slices.Clone(subjects)
	slices.SortFunc(sorted, func(a, b Subject) int {
		switch {
		case a.Time < b.Time:
			return -1
		case a.Time > b.Time:
			return 1
		}
		return 0
	})
	cv := curve{steps: []step{{0, 1, 1, 1}}}
	s, greenwood := 1.0, 0.0
	n := len(sorted)
	for i := 0; i < len(sorted); {
		tm := sorted[i].Time
		d, left := 0, 0
		for ; i < len(sorted) && sorted[i].Time == tm; i++ {
			cv.times = append(cv.times, tm)
			if sorted[i].Event {
				d++
			} else {
				left++
			}
		}
		if d > 0 {
			s *= 1 - float64(d)/float64(n)
			if n > d {
				greenwood += float64(d) / float64(n*(n-d))
			}
			se := s * math.Sqrt(greenwood)
			cv.steps = append(cv.steps, step{tm, s, max(0, s-1.96*se), min(1, s+1.96*se)})
		}
		if left > 0 {
			cv.censored = append(cv.censored, [2]float64{tm, s})
		}
		n -= d + left
		cv.end = tm
	}
	return cv
}

// at returns the step in force at time tm.
func (cv curve) at(tm float64) step {
	k := 0
	for i, st := range cv.steps {
		if st.time <= tm {
			k = i
		}
	}
	return cv.steps[k]
}

// atRisk returns how many subjects were still in the study just before
// time tm.
func (cv curve) atRisk(tm float64) int {
	i, _ := slices.BinarySearch(cv.times, tm)
	return len(cv.times) - i
}

// median returns the first time survival falls to a half or below.
func (cv curve) median() (float64, bool) {
	for _, st := range cv.steps {
		if st.s <= 0.5 {
			return st.time, true
		}
	}
	return 0, false
}

func number(v float64) string { return strconv.FormatFloat(v, 'f', -1, 64) }

// View creates the curves of groups over time, measured in unit.
func View(c *ui.Context, groups []Group, unit string) ui.Element {
	t := c.Theme()
	curves := make([]curve, len(groups))
	colors := make([]ui.Color, len(groups))
	end := 0.0
	for i, g := range groups {
		curves[i] = estimate(g.Subjects)
		end = max(end, curves[i].end)
		colors[i] = g.Color
		if colors[i] == (ui.Color{}) {
			colors[i] = palette[i%len(palette)]
		}
	}
	x := chart.NewLinear(0, max(end, 1)).Nice(6)
	y := chart.NewLinear(0, 1)
	xticks, yticks := x.Ticks(6), y.Ticks(4)
	for i := range yticks {
		yticks[i].Label = fmt.Sprintf("%.0f%%", 100*yticks[i].Value)
	}
	small := t.FontSize - 2
	label := func(side chart.Side, tk chart.Tick) {
		if side != chart.Bottom {
			ui.Text(c, tk.Label).FontSize(small).TextColor(t.TextMuted).NoWrap()
			return
		}
		ui.Column(c).AlignItems(ui.Center).Children(func() {
			ui.Text(c, tk.Label).FontSize(small).TextColor(t.TextMuted).NoWrap().MarginY(t.Space(1))
			for i, cv := range curves {
				ui.Text(c.Key(i), strconv.Itoa(cv.atRisk(tk.Value))).FontSize(small).TextColor(colors[i]).FontFeatures("tnum").NoWrap()
			}
		})
	}

	return ui.Column(c).Gap(t.Space(3)).Children(func() {
		ui.Row(c).Gap(t.Space(5)).Wrap().Children(func() {
			for i, g := range groups {
				ui.Row(c.Key(i)).Gap(t.Space(1.5)).AlignItems(ui.Center).Children(func() {
					ui.Box(c).Size(t.Space(3), t.Space(1)).Radius(t.Space(0.5)).Background(colors[i])
					ui.Text(c, g.Name)
					median := "median not reached"
					if m, ok := curves[i].median(); ok {
						median = "median " + number(m)
					}
					ui.Text(c, median).TextColor(t.TextMuted)
				})
			}
		})
		chart.Plot(c, chart.Axes{Left: yticks, Bottom: xticks, Label: label}, func(area ui.Element) {
			area.Label("Survival over " + unit)
			px, _, over := area.PointerPosition()
			b := area.Bounds()
			tm := -1.0
			if over && b.W > 0 {
				tm = max(0, x.Invert(px/b.W))
			}
			area.Draw(func(p *ui.Painter, r ui.Rect) {
				for _, tk := range yticks {
					p.Line(r.X, chart.Y(r, tk.At), r.X+r.W, chart.Y(r, tk.At), 1, t.Border.Alpha(0.6))
				}
				for _, tk := range xticks {
					p.Line(chart.X(r, tk.At), r.Y, chart.X(r, tk.At), r.Y+r.H, 1, t.Border.Alpha(0.6))
				}
				px := func(v float64) float32 { return chart.X(r, x.At(v)) }
				py := func(v float64) float32 { return chart.Y(r, y.At(v)) }
				for i, cv := range curves {
					var band ui.Path
					var line [][2]float32
					for k, st := range cv.steps {
						if k > 0 {
							line = append(line, [2]float32{px(st.time), py(cv.steps[k-1].s)})
						}
						line = append(line, [2]float32{px(st.time), py(st.s)})
					}
					line = append(line, [2]float32{px(cv.end), py(cv.steps[len(cv.steps)-1].s)})
					for k, st := range cv.steps {
						next := cv.end
						if k+1 < len(cv.steps) {
							next = cv.steps[k+1].time
						}
						if k == 0 {
							band.MoveTo(px(st.time), py(st.hi))
						} else {
							band.LineTo(px(st.time), py(st.hi))
						}
						band.LineTo(px(next), py(st.hi))
					}
					for k := len(cv.steps) - 1; k >= 0; k-- {
						st := cv.steps[k]
						next := cv.end
						if k+1 < len(cv.steps) {
							next = cv.steps[k+1].time
						}
						band.LineTo(px(next), py(st.lo))
						band.LineTo(px(st.time), py(st.lo))
					}
					band.Close()
					p.FillPath(&band, colors[i].Alpha(0.12))
					p.StrokePolyline(line, 2, colors[i])
					for _, cs := range cv.censored {
						cx, cy := px(cs[0]), py(cs[1])
						p.Fill(ui.Rect{X: cx - 0.75, Y: cy - 4, W: 1.5, H: 8}, colors[i], 0)
					}
				}
				if tm >= 0 {
					hx := r.X + float32(x.At(tm))*r.W
					p.Line(hx, r.Y, hx, r.Y+r.H, 1, t.TextMuted.Alpha(0.5))
					for i, cv := range curves {
						if tm > cv.end {
							continue
						}
						var dot ui.Path
						dot.Circle(hx, py(cv.at(tm).s), 4)
						p.FillPath(&dot, colors[i])
						p.StrokePath(&dot, 1.5, t.Background)
					}
				}
			})
			if tm < 0 {
				return
			}
			f := x.At(tm)
			h := ui.Start
			if f > 0.6 {
				h = ui.End
			}
			chart.Anchor(c, f, 1, h, ui.Start, func() {
				ui.Column(c).Gap(t.Space(1)).Padding(t.Space(2), t.Space(2.5)).Margin(t.Space(1), t.Space(2)).
					Radius(t.Space(1.5)).Background(t.Background).Border(1, t.Border).
					Shadow(0, 2, 8, 0, ui.RGBA(0, 0, 0, 0.12)).PassThrough().Children(func() {
					ui.Text(c, fmt.Sprintf("%s %.1f", unit, tm)).FontSize(small).TextColor(t.TextMuted).NoWrap()
					for i, cv := range curves {
						if tm > cv.end {
							continue
						}
						st := cv.at(tm)
						ui.Row(c.Key(i)).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
							ui.Box(c).Size(t.Space(2), t.Space(2)).Radius(t.Space(1)).Background(colors[i])
							ui.Text(c, groups[i].Name).FontSize(t.FontSize - 1).NoWrap().Grow(1)
							ui.Text(c, fmt.Sprintf("%.1f%% (%.0f–%.0f)", 100*st.s, 100*st.lo, 100*st.hi)).
								FontSize(t.FontSize - 1).FontFeatures("tnum").NoWrap()
						})
					}
				})
			})
		}).Height(t.Space(80)).Gap(t.Space(2))
		ui.Text(c, unit).FontSize(small).TextColor(t.TextMuted).AlignSelf(ui.Center)
	})
}
