// Package volcano plots the results of a differential expression test as
// a volcano plot: a point a gene at its log2 fold change across and the
// -log10 of its p-value up, dashed lines at the thresholds, and the genes
// past both colored by direction, up in red and down in blue, the rest
// gray. The most significant of them are named by their points. The
// sliders below set the thresholds, which start at a fold change of 2 and
// a p-value of 0.05, and pointing at a point reads out its gene.
//
//	volcano.View(c, &app.volcano, genes)
package volcano

import (
	"cmp"
	"fmt"
	"math"
	"slices"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/chart"
)

// Gene is a gene's result: its log2 fold change and its p-value.
type Gene struct {
	Name       string
	FoldChange float64
	P          float64
}

// named is how many of the most significant genes are named.
const named = 12

// State is what a view keeps: its thresholds, as a log2 fold change and
// as -log10 of a p-value, and the point pointed at. Keep it in the app's
// state, one for each view.
type State struct {
	fold, sig float64
	set       bool
	hovered   int
}

var (
	up   = ui.Hex("#E5484D")
	down = ui.Hex("#3E7BFA")
)

// View creates the plot of genes.
func View(c *ui.Context, s *State, genes []Gene) ui.Element {
	t := c.Theme()
	if !s.set {
		s.fold, s.sig, s.set = 1, -math.Log10(0.05), true
	}
	ys := make([]float64, len(genes))
	top, wide := 1.0, 1.0
	for i, g := range genes {
		ys[i] = -math.Log10(max(g.P, 1e-300))
		top = max(top, ys[i])
		wide = max(wide, math.Abs(g.FoldChange))
	}
	x := chart.NewLinear(-wide, wide).Nice(8)
	y := chart.NewLinear(0, top).Nice(6)
	xticks, yticks := x.Ticks(8), y.Ticks(6)
	side := func(i int) int {
		g := genes[i]
		switch {
		case ys[i] < s.sig || math.Abs(g.FoldChange) < s.fold:
			return 0
		case g.FoldChange > 0:
			return 1
		}
		return -1
	}
	var hits []int
	nUp, nDown := 0, 0
	for i := range genes {
		switch side(i) {
		case 1:
			nUp++
			hits = append(hits, i)
		case -1:
			nDown++
			hits = append(hits, i)
		}
	}
	slices.SortFunc(hits, func(a, b int) int { return cmp.Compare(ys[b], ys[a]) })
	hits = hits[:min(len(hits), named)]

	return ui.Column(c).Gap(t.Space(3)).Children(func() {
		ui.Row(c).Gap(t.Space(5)).Children(func() {
			legend := func(color ui.Color, text string) {
				ui.Row(c).Gap(t.Space(1.5)).AlignItems(ui.Center).Children(func() {
					ui.Box(c).Size(t.Space(2), t.Space(2)).Radius(t.Space(1)).Background(color)
					ui.Text(c, text).FontFeatures("tnum")
				})
			}
			legend(up, fmt.Sprintf("%d up", nUp))
			legend(down, fmt.Sprintf("%d down", nDown))
			legend(t.TextMuted.Alpha(0.5), fmt.Sprintf("%d not significant", len(genes)-nUp-nDown))
		})
		ui.Text(c, "−log10 p").FontSize(t.FontSize - 2).TextColor(t.TextMuted)
		chart.Plot(c, chart.Axes{Left: yticks, Bottom: xticks}, func(area ui.Element) {
			area.Label(fmt.Sprintf("Volcano plot of %d genes", len(genes)))
			px, py, over := area.PointerPosition()
			b := area.Bounds()
			s.hovered = -1
			if over && b.W > 0 && b.H > 0 {
				best := float32(t.Space(3) * t.Space(3))
				for i, g := range genes {
					dx := x.At(g.FoldChange)*b.W - px
					dy := (1-y.At(ys[i]))*b.H - py
					if d := dx*dx + dy*dy; d <= best {
						best, s.hovered = d, i
					}
				}
			}
			hovered := s.hovered
			fold, sig := s.fold, s.sig
			area.Draw(func(p *ui.Painter, r ui.Rect) {
				for _, tk := range yticks {
					p.Line(r.X, chart.Y(r, tk.At), r.X+r.W, chart.Y(r, tk.At), 1, t.Border.Alpha(0.5))
				}
				for _, tk := range xticks {
					p.Line(chart.X(r, tk.At), r.Y, chart.X(r, tk.At), r.Y+r.H, 1, t.Border.Alpha(0.5))
				}
				gray := t.TextMuted.Alpha(0.35)
				for pass := range 2 {
					for i, g := range genes {
						sd := side(i)
						if (sd == 0) != (pass == 0) {
							continue
						}
						color := map[int]ui.Color{-1: down, 0: gray, 1: up}[sd]
						cx, cy := chart.X(r, x.At(g.FoldChange)), chart.Y(r, y.At(ys[i]))
						p.Fill(ui.Rect{X: cx - 2, Y: cy - 2, W: 4, H: 4}, color.Alpha(0.75), 2)
					}
				}
				dash := t.TextMuted.Alpha(0.8)
				sy := chart.Y(r, y.At(sig))
				for dx := r.X; dx < r.X+r.W; dx += 7 {
					p.Fill(ui.Rect{X: dx, Y: sy, W: min(4, r.X+r.W-dx), H: 1}, dash, 0)
				}
				for _, f := range []float64{-fold, fold} {
					fx := chart.X(r, x.At(f))
					for dy := r.Y; dy < r.Y+r.H; dy += 7 {
						p.Fill(ui.Rect{X: fx, Y: dy, W: 1, H: min(4, r.Y+r.H-dy)}, dash, 0)
					}
				}
				fs := t.FontSize - 2
				var placed []ui.Rect
				for _, i := range hits {
					g := genes[i]
					cx, cy := chart.X(r, x.At(g.FoldChange)), chart.Y(r, y.At(ys[i]))
					w, h := p.MeasureText(0, ui.Span{Text: g.Name, Size: fs})
					lr := ui.Rect{X: cx + 4, Y: cy - h - 2, W: w, H: h}
					if g.FoldChange < 0 {
						lr.X = cx - 4 - w
					}
					lr.X = min(max(lr.X, r.X), r.X+r.W-w)
					lr.Y = max(lr.Y, r.Y)
					if slices.ContainsFunc(placed, func(o ui.Rect) bool {
						return lr.X < o.X+o.W && o.X < lr.X+lr.W && lr.Y < o.Y+o.H && o.Y < lr.Y+lr.H
					}) {
						continue
					}
					placed = append(placed, lr)
					p.Text(lr.X, lr.Y, g.Name, fs, t.Text)
				}
				if hovered >= 0 {
					g := genes[hovered]
					var ring ui.Path
					ring.Circle(chart.X(r, x.At(g.FoldChange)), chart.Y(r, y.At(ys[hovered])), 5)
					p.StrokePath(&ring, 2, t.Text)
				}
			})
			if hovered < 0 {
				return
			}
			g := genes[hovered]
			fx, fy := x.At(g.FoldChange), y.At(ys[hovered])
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
					ui.Text(c, g.Name).Bold().NoWrap()
					for _, line := range []string{
						fmt.Sprintf("log2 fold change %.2f (×%.2f)", g.FoldChange, math.Exp2(g.FoldChange)),
						fmt.Sprintf("p %.2g", g.P),
					} {
						ui.Text(c, line).FontSize(t.FontSize - 1).TextColor(t.TextMuted).FontFeatures("tnum").NoWrap()
					}
				})
			})
		}).Height(t.Space(90)).Gap(t.Space(2))
		ui.Text(c, "log2 fold change").FontSize(t.FontSize - 2).TextColor(t.TextMuted).AlignSelf(ui.End)
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "Fold change").TextColor(t.TextMuted)
			ui.Slider(c, &s.fold, 0, 4).Step(0.25).Width(t.Space(40)).Label("Fold change threshold, log2")
			ui.Text(c, fmt.Sprintf("×%.2f", math.Exp2(s.fold))).FontFeatures("tnum").Width(t.Space(14))
			ui.Text(c, "p").TextColor(t.TextMuted)
			ui.Slider(c, &s.sig, 1, 10).Step(0.1).Width(t.Space(40)).Label("p-value threshold, −log10")
			ui.Text(c, fmt.Sprintf("< %.2g", math.Pow(10, -s.sig))).FontFeatures("tnum")
		})
	})
}
