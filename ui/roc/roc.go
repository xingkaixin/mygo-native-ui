// Package roc shows how well a binary classifier's scores tell positives
// from negatives: the ROC curve, the true positive rate over the false
// positive rate, beside the precision-recall curve, with the area under
// the first and the average precision of the second. The threshold, set
// with the slider or by pressing and dragging along either curve, marks
// its point on both, and the counts and rates below follow it. The view
// starts at the threshold that best separates the classes, where the true
// positive rate most exceeds the false positive rate.
//
//	roc.View(c, &app.roc, app.samples)
package roc

import (
	"fmt"
	"math"
	"slices"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/chart"
)

// Sample is the score a classifier gave an example, and whether the
// example is positive.
type Sample struct {
	Score    float64
	Positive bool
}

// State is what a view keeps: the threshold. Keep it in the app's state,
// one for each view.
type State struct {
	threshold float64
	set       bool
}

// Threshold returns the threshold: scores at or above it count as
// positive.
func (s *State) Threshold() float64 { return s.threshold }

// point is the classifier at a threshold: the true and false positives at
// or above it.
type point struct {
	threshold float64
	tp, fp    int
}

// curve returns the points of samples from the highest threshold down,
// the first above every score, and the counts of positives and negatives.
func curve(samples []Sample) (pts []point, pos, neg int) {
	sorted := slices.Clone(samples)
	slices.SortFunc(sorted, func(a, b Sample) int {
		switch {
		case a.Score > b.Score:
			return -1
		case a.Score < b.Score:
			return 1
		}
		return 0
	})
	pts = []point{{threshold: math.Inf(1)}}
	tp, fp := 0, 0
	for i, s := range sorted {
		if s.Positive {
			tp++
		} else {
			fp++
		}
		if i == len(sorted)-1 || sorted[i+1].Score != s.Score {
			pts = append(pts, point{s.Score, tp, fp})
		}
	}
	return pts, tp, fp
}

// at returns the point of threshold th.
func at(pts []point, th float64) int {
	k := 0
	for i, p := range pts {
		if p.threshold >= th {
			k = i
		}
	}
	return k
}

func precision(p point) float64 {
	if p.tp+p.fp == 0 {
		return 1
	}
	return float64(p.tp) / float64(p.tp+p.fp)
}

// View creates the curves of samples.
func View(c *ui.Context, s *State, samples []Sample) ui.Element {
	t := c.Theme()
	pts, pos, neg := curve(samples)
	root := ui.Column(c).Gap(t.Space(4))
	if pos == 0 || neg == 0 {
		return root.Children(func() {
			ui.Text(c, "The curves need both positive and negative samples.").TextColor(t.TextMuted)
		})
	}
	lo, hi := pts[len(pts)-1].threshold, pts[1].threshold
	if !s.set {
		youden := func(p point) float64 { return float64(p.tp)/float64(pos) - float64(p.fp)/float64(neg) }
		best := 1
		for i := 2; i < len(pts); i++ {
			if youden(pts[i]) > youden(pts[best]) {
				best = i
			}
		}
		s.threshold, s.set = pts[best].threshold, true
	}

	fpr, tpr, prec := make([]float32, len(pts)), make([]float32, len(pts)), make([]float32, len(pts))
	auc, ap := 0.0, 0.0
	for i, p := range pts {
		fpr[i], tpr[i], prec[i] = float32(p.fp)/float32(neg), float32(p.tp)/float32(pos), float32(precision(p))
		if i > 0 {
			auc += float64(fpr[i]-fpr[i-1]) * float64(tpr[i]+tpr[i-1]) / 2
			ap += float64(tpr[i]-tpr[i-1]) * precision(p)
		}
	}
	// The precision-recall curve starts at the precision of the first
	// threshold, not that of predicting nothing.
	prec[0] = prec[1]
	pick := func(i int) { s.threshold = pts[max(i, 1)].threshold }

	return root.Children(func() {
		k := at(pts, s.threshold)
		ui.Row(c).Gap(t.Space(8)).Children(func() {
			plot(c, fmt.Sprintf("ROC · AUC %.3f", auc), "False positive rate", "True positive rate", fpr, tpr, k, true, pick)
			plot(c, fmt.Sprintf("Precision–recall · AP %.3f", ap), "Recall", "Precision", tpr, prec, k, false, pick)
		})
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "Threshold").TextColor(t.TextMuted)
			ui.Slider(c, &s.threshold, lo, hi).Grow(1).Label("Threshold")
			ui.Text(c, fmt.Sprintf("%.3f", s.threshold)).FontFeatures("tnum").Width(t.Space(14))
		})
		p := pts[k]
		fn, tn := pos-p.tp, neg-p.fp
		ui.Row(c).Gap(t.Space(10)).Children(func() {
			ui.Grid(c).ColumnTracks(ui.FitContent(), ui.Fixed(t.Space(26)), ui.Fixed(t.Space(26))).GapX(t.Space(1)).GapY(t.Space(1)).AlignItems(ui.Center).Children(func() {
				head := func(text string) { ui.Text(c, text).FontSize(t.FontSize - 1).TextColor(t.TextMuted).NoWrap() }
				head("")
				head("Predicted positive")
				head("Predicted negative")
				head("Actual positive")
				cell(c, "True positive", p.tp, t.Success)
				cell(c, "False negative", fn, t.Danger)
				head("Actual negative")
				cell(c, "False positive", p.fp, t.Danger)
				cell(c, "True negative", tn, t.Success)
			})
			r := float64(p.tp) / float64(pos)
			pr := precision(p)
			ui.Grid(c).ColumnTracks(ui.FitContent(), ui.FitContent()).GapX(t.Space(4)).GapY(t.Space(1)).Children(func() {
				for _, m := range []struct {
					name string
					v    float64
				}{
					{"Precision", pr},
					{"Recall", r},
					{"F1", 2 * pr * r / max(pr+r, 1e-12)},
					{"Accuracy", float64(p.tp+tn) / float64(pos+neg)},
					{"False positive rate", float64(p.fp) / float64(neg)},
				} {
					ui.Text(c, m.name).TextColor(t.TextMuted).NoWrap()
					ui.Text(c, fmt.Sprintf("%.3f", m.v)).FontFeatures("tnum")
				}
			})
		})
	})
}

// cell creates a count of the confusion matrix, tinted with color.
func cell(c *ui.Context, name string, n int, color ui.Color) {
	t := c.Theme()
	ui.Column(c).Padding(t.Space(2), t.Space(3)).Radius(t.Radius).Background(color.Alpha(0.12)).Label(fmt.Sprintf("%s: %d", name, n)).Children(func() {
		ui.Text(c, name).FontSize(t.FontSize - 2).TextColor(t.TextMuted).NoWrap()
		ui.Text(c, fmt.Sprint(n)).FontSize(t.FontSize + 2).Bold().FontFeatures("tnum")
	})
}

// plot creates a curve through xs and ys, fractions of the plot, with the
// point k marked, the diagonal of chance under it if diagonal, and pick
// called with the point nearest where the pointer presses.
func plot(c *ui.Context, title, xname, yname string, xs, ys []float32, k int, diagonal bool, pick func(i int)) {
	t := c.Theme()
	scale := chart.NewLinear(0, 1)
	ticks := scale.Ticks(5)
	ui.Column(c).Grow(1).Basis(0).Gap(t.Space(2)).Children(func() {
		ui.Text(c, title).Bold()
		ui.Text(c, yname).FontSize(t.FontSize - 2).TextColor(t.TextMuted)
		chart.Plot(c, chart.Axes{Left: ticks, Bottom: ticks}, func(area ui.Element) {
			area.Label(title).Cursor(ui.CursorCrosshair)
			if area.Pressed() {
				px, py, _ := area.PointerPosition()
				b := area.Bounds()
				fx, fy := px/max(b.W, 1), 1-py/max(b.H, 1)
				best, dist := 0, float32(math.Inf(1))
				for i := 1; i < len(xs); i++ {
					if d := (xs[i]-fx)*(xs[i]-fx) + (ys[i]-fy)*(ys[i]-fy); d < dist {
						best, dist = i, d
					}
				}
				pick(best)
			}
			area.Draw(func(p *ui.Painter, r ui.Rect) {
				grid := t.Border.Alpha(0.6)
				for _, tk := range ticks {
					p.Line(r.X, chart.Y(r, tk.At), r.X+r.W, chart.Y(r, tk.At), 1, grid)
					p.Line(chart.X(r, tk.At), r.Y, chart.X(r, tk.At), r.Y+r.H, 1, grid)
				}
				zero := make([]float32, len(xs))
				p.FillPath(chart.Area(r, xs, zero, ys, chart.CurveLinear), t.Accent.Alpha(0.1))
				if diagonal {
					p.StrokePolyline([][2]float32{{r.X, r.Y + r.H}, {r.X + r.W, r.Y}}, 1, t.TextMuted.Alpha(0.6))
				}
				p.StrokePath(chart.Line(r, xs, ys, chart.CurveLinear), 2, t.Accent)
				cx, cy := chart.X(r, xs[k]), chart.Y(r, ys[k])
				p.Line(cx, cy, cx, r.Y+r.H, 1, t.Accent.Alpha(0.5))
				p.Line(r.X, cy, cx, cy, 1, t.Accent.Alpha(0.5))
				var dot ui.Path
				dot.Circle(cx, cy, 5)
				p.FillPath(&dot, t.Accent)
				p.StrokePath(&dot, 2, t.Background)
			})
		}).Height(t.Space(64)).Gap(t.Space(2))
		ui.Text(c, xname).FontSize(t.FontSize - 2).TextColor(t.TextMuted).AlignSelf(ui.End)
	})
}
