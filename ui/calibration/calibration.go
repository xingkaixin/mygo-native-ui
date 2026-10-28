// Package calibration shows whether a classifier's probabilities mean
// what they say, as a reliability diagram: the predictions of a model put
// in bins of equal width by probability, each bin a point at its mean
// probability across and the share of positives in it up, joined in a line
// a model, with the diagonal where the two agree. Bars along the bottom
// count the predictions in each bin, and the legend gives each model's
// expected calibration error: how far its bins fall from the diagonal,
// weighted by their counts. Pointing at a point reads out its bin.
//
//	calibration.View(c, &app.cal, models, 10)
package calibration

import (
	"fmt"
	"math"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/chart"
)

// Prediction is the probability a model gave an example of being
// positive, from 0 to 1, and whether it is.
type Prediction struct {
	P        float64
	Positive bool
}

// Model is a model's predictions, named, and drawn in Color, zero for one
// of its own.
type Model struct {
	Name        string
	Color       ui.Color
	Predictions []Prediction
}

var palette = []ui.Color{ui.Hex("#4C8DF6"), ui.Hex("#F2994A"), ui.Hex("#27AE60"), ui.Hex("#9B51E0"), ui.Hex("#EB5757")}

// State is what a view keeps: the point pointed at. Keep it in the app's
// state, one for each view.
type State struct {
	model, bin int
	over       bool
}

// bin is the predictions of a model whose probabilities fall in one bin.
type bin struct {
	n, positives int
	sum          float64
}

func (b bin) mean() float64     { return b.sum / float64(b.n) }
func (b bin) observed() float64 { return float64(b.positives) / float64(b.n) }

// binned puts predictions in n bins and returns them with the expected
// calibration error.
func binned(predictions []Prediction, n int) ([]bin, float64) {
	bins := make([]bin, n)
	for _, p := range predictions {
		k := min(max(int(p.P*float64(n)), 0), n-1)
		bins[k].n++
		bins[k].sum += p.P
		if p.Positive {
			bins[k].positives++
		}
	}
	ece := 0.0
	for _, b := range bins {
		if b.n > 0 {
			ece += float64(b.n) / float64(len(predictions)) * math.Abs(b.mean()-b.observed())
		}
	}
	return bins, ece
}

// View creates the diagram of models with their predictions in bins of
// equal width, n of them across.
func View(c *ui.Context, s *State, models []Model, n int) ui.Element {
	t := c.Theme()
	n = max(n, 1)
	colors := make([]ui.Color, len(models))
	bins := make([][]bin, len(models))
	eces := make([]float64, len(models))
	most := 1
	for i, m := range models {
		colors[i] = m.Color
		if colors[i] == (ui.Color{}) {
			colors[i] = palette[i%len(palette)]
		}
		bins[i], eces[i] = binned(m.Predictions, n)
		for _, b := range bins[i] {
			most = max(most, b.n)
		}
	}
	scale := chart.NewLinear(0, 1)
	ticks := scale.Ticks(5)

	return ui.Column(c).Gap(t.Space(3)).Children(func() {
		ui.Row(c).Gap(t.Space(5)).Wrap().Children(func() {
			for i, m := range models {
				ui.Row(c.Key(i)).Gap(t.Space(1.5)).AlignItems(ui.Center).Children(func() {
					ui.Box(c).Size(t.Space(3), t.Space(1)).Radius(t.Space(0.5)).Background(colors[i])
					ui.Text(c, m.Name)
					ui.Text(c, fmt.Sprintf("ECE %.3f", eces[i])).TextColor(t.TextMuted).FontFeatures("tnum")
				})
			}
		})
		ui.Text(c, "Fraction of positives").FontSize(t.FontSize - 2).TextColor(t.TextMuted)
		chart.Plot(c, chart.Axes{Left: ticks, Bottom: ticks}, func(area ui.Element) {
			area.Label("Reliability diagram")
			px, py, over := area.PointerPosition()
			b := area.Bounds()
			s.over = false
			if over && b.W > 0 {
				best := float32(t.Space(4) * t.Space(4))
				for i, bs := range bins {
					for k, bn := range bs {
						if bn.n == 0 {
							continue
						}
						dx := float32(bn.mean())*b.W - px
						dy := (1-float32(bn.observed()))*b.H - py
						if d := dx*dx + dy*dy; d < best {
							best, s.model, s.bin, s.over = d, i, k, true
						}
					}
				}
			}
			hovered, hm, hb := s.over, s.model, s.bin
			area.Draw(func(p *ui.Painter, r ui.Rect) {
				grid := t.Border.Alpha(0.6)
				for _, tk := range ticks {
					p.Line(r.X, chart.Y(r, tk.At), r.X+r.W, chart.Y(r, tk.At), 1, grid)
					p.Line(chart.X(r, tk.At), r.Y, chart.X(r, tk.At), r.Y+r.H, 1, grid)
				}
				bw := r.W / float32(n)
				for i, bs := range bins {
					for k, bn := range bs {
						h := float32(bn.n) / float32(most) * r.H * 0.2
						w := bw * 0.8 / float32(len(models))
						x := r.X + float32(k)*bw + bw*0.1 + float32(i)*w
						p.Fill(ui.Rect{X: x, Y: r.Y + r.H - h, W: w, H: h}, colors[i].Alpha(0.3), 0)
					}
				}
				p.StrokePolyline([][2]float32{{r.X, r.Y + r.H}, {r.X + r.W, r.Y}}, 1, t.TextMuted.Alpha(0.6))
				for i, bs := range bins {
					var line [][2]float32
					for _, bn := range bs {
						if bn.n > 0 {
							line = append(line, [2]float32{chart.X(r, float32(bn.mean())), chart.Y(r, float32(bn.observed()))})
						}
					}
					p.StrokePolyline(line, 2, colors[i])
					for k, pt := range line {
						var dot ui.Path
						radius := float32(3.5)
						if hovered && hm == i && k == hb {
							radius = 5.5
						}
						dot.Circle(pt[0], pt[1], radius)
						p.FillPath(&dot, colors[i])
						p.StrokePath(&dot, 1.5, t.Background)
					}
				}
			})
			if !hovered {
				return
			}
			bn := bins[hm][hb]
			fx, fy := float32(bn.mean()), float32(bn.observed())
			h, v := ui.Start, ui.End
			if fx > 0.6 {
				h = ui.End
			}
			if fy > 0.7 {
				v = ui.Start
			}
			chart.Anchor(c, fx, fy, h, v, func() {
				ui.Column(c).Gap(t.Space(0.5)).Padding(t.Space(2), t.Space(2.5)).Margin(t.Space(2)).
					Radius(t.Space(1.5)).Background(t.Background).Border(1, t.Border).
					Shadow(0, 2, 8, 0, ui.RGBA(0, 0, 0, 0.12)).PassThrough().Children(func() {
					lo, hi := float64(hb)/float64(n), float64(hb+1)/float64(n)
					ui.Text(c, fmt.Sprintf("%s · %.2f–%.2f", models[hm].Name, lo, hi)).Bold().NoWrap()
					for _, line := range []string{
						fmt.Sprintf("%d predictions", bn.n),
						fmt.Sprintf("Mean probability %.3f", bn.mean()),
						fmt.Sprintf("Positives %.1f%%", 100*bn.observed()),
					} {
						ui.Text(c, line).FontSize(t.FontSize - 1).TextColor(t.TextMuted).FontFeatures("tnum").NoWrap()
					}
				})
			})
		}).Height(t.Space(90)).Gap(t.Space(2))
		ui.Text(c, "Mean predicted probability").FontSize(t.FontSize - 2).TextColor(t.TextMuted).AlignSelf(ui.End)
	})
}
