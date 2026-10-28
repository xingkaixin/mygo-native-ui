// Package splom draws a scatterplot matrix: for every pair of a table's
// columns, a scatterplot of one against the other, the column's own
// histogram on the diagonal with its name and range, and the points
// colored by their group. Dragging a rectangle in any plot brushes the
// rows inside it, which then stand out in every plot while the rest fade;
// on the diagonal the drag brushes a range of the column. A click
// without a drag clears the brush.
//
//	splom.View(c, &app.matrix, columns, rows, groups)
package splom

import (
	"fmt"
	"math"

	"github.com/egoist/mygo/ui"
)

// State is what a matrix keeps: its brush. Keep it in the app's state,
// one for each matrix.
type State struct {
	brushing bool
	// The brushed plot, row i and column j, and the brushed ranges of
	// its columns, in their values.
	i, j           int
	x0, x1, y0, y1 float64
	on             bool
	fromX, fromY   float32
}

var palette = []ui.Color{
	ui.RGB(66, 133, 244), ui.RGB(234, 134, 46), ui.RGB(52, 168, 83), ui.RGB(217, 72, 92),
	ui.RGB(142, 92, 214), ui.RGB(23, 162, 184),
}

// View creates the matrix of the columns named columns, each row of rows
// a value for each column, and groups the group of each row, or nil.
func View(c *ui.Context, s *State, columns []string, rows [][]float64, groups []string) ui.Element {
	t := c.Theme()
	n := len(columns)
	root := ui.Column(c).Gap(t.Space(3)).Role(ui.RoleGroup).Label("Scatterplot matrix")
	if n == 0 {
		return root
	}
	lo, hi := make([]float64, n), make([]float64, n)
	for k := range n {
		lo[k], hi[k] = math.Inf(1), math.Inf(-1)
		for _, r := range rows {
			lo[k], hi[k] = min(lo[k], r[k]), max(hi[k], r[k])
		}
		pad := (hi[k] - lo[k]) * 0.06
		if pad == 0 {
			pad = 1
		}
		lo[k], hi[k] = lo[k]-pad, hi[k]+pad
	}
	colors := map[string]ui.Color{}
	var names []string
	for _, g := range groups {
		if _, ok := colors[g]; !ok {
			colors[g] = palette[len(names)%len(palette)]
			names = append(names, g)
		}
	}
	color := func(row int) ui.Color {
		if groups == nil {
			return t.Accent
		}
		return colors[groups[row]]
	}
	picked := make([]bool, len(rows))
	count := 0
	for r, row := range rows {
		x := row[s.j]
		in := !s.on || (x >= min(s.x0, s.x1) && x <= max(s.x0, s.x1))
		if in && s.on && s.i != s.j {
			y := row[s.i]
			in = y >= min(s.y0, s.y1) && y <= max(s.y0, s.y1)
		}
		picked[r] = in
		if in {
			count++
		}
	}
	root.Children(func() {
		ui.Row(c).Gap(t.Space(4)).AlignItems(ui.Center).Children(func() {
			for _, g := range names {
				ui.Row(c.Key(g)).Gap(t.Space(1.5)).AlignItems(ui.Center).Children(func() {
					ui.Box(c).Size(t.Space(2.5), t.Space(2.5)).Radius(t.Space(1.25)).Background(colors[g])
					ui.Text(c, g).FontSize(t.FontSize - 1).TextColor(t.TextMuted)
				})
			}
			ui.Box(c).Grow(1)
			if s.on {
				ui.Text(c, fmt.Sprintf("%d of %d rows", count, len(rows))).FontSize(t.FontSize - 1).FontFeatures("tnum")
			}
		})
		view := ui.Box(c).Grow(1).Cursor(ui.CursorCrosshair).Role(ui.RoleImage)
		b := view.LayoutBounds()
		gap := t.Space(1.5)
		size := (min(b.W, b.H) - gap*float32(n-1)) / float32(n)
		cellAt := func(x, y float32) (int, int, bool) {
			j, i := int(x/(size+gap)), int(y/(size+gap))
			return i, j, i >= 0 && j >= 0 && i < n && j < n
		}
		origin := func(i, j int) (float32, float32) { return float32(j) * (size + gap), float32(i) * (size + gap) }
		value := func(k int, at, from float32, flip bool) float64 {
			f := float64((at - from) / size)
			if flip {
				f = 1 - f
			}
			return lo[k] + max(0, min(1, f))*(hi[k]-lo[k])
		}
		view.HandleInput(func(in ui.InputEvent) bool {
			switch in.Kind {
			case ui.InputPointerDown:
				i, j, ok := cellAt(in.X, in.Y)
				if !ok {
					return false
				}
				s.brushing, s.i, s.j, s.fromX, s.fromY = true, i, j, in.X, in.Y
				s.on = false
			case ui.InputPointerMove:
				if !s.brushing {
					return false
				}
				ox, oy := origin(s.i, s.j)
				if math.Hypot(float64(in.X-s.fromX), float64(in.Y-s.fromY)) < 3 {
					return true
				}
				s.on = true
				s.x0, s.x1 = value(s.j, s.fromX, ox, false), value(s.j, in.X, ox, false)
				s.y0, s.y1 = value(s.i, s.fromY, oy, true), value(s.i, in.Y, oy, true)
			case ui.InputPointerUp:
				s.brushing = false
			default:
				return false
			}
			return true
		})
		view.Draw(func(p *ui.Painter, r ui.Rect) {
			small := t.FontSize - 2
			dot := max(1.5, min(3, size/60))
			for i := range n {
				for j := range n {
					ox, oy := origin(i, j)
					cell := ui.Rect{X: r.X + ox, Y: r.Y + oy, W: size, H: size}
					p.Fill(cell, t.Surface, t.Radius)
					px := func(k int, v float64) float32 { return float32((v - lo[k]) / (hi[k] - lo[k])) }
					if i == j {
						histogram(p, t, cell, rows, j, lo[j], hi[j], picked, s.on)
						p.RichText(cell.X+t.Space(2), cell.Y+t.Space(1.5), 0, ui.Span{Text: columns[j], Size: t.FontSize - 1, Weight: 600, Color: t.Text})
						for _, end := range []struct {
							v float64
							x float32
						}{{lo[j], 0}, {hi[j], 1}} {
							sp := ui.Span{Text: short(end.v), Size: small, Color: t.TextMuted, Features: "tnum"}
							w, h := p.MeasureText(0, sp)
							p.RichText(cell.X+t.Space(1.5)+(cell.W-w-t.Space(3))*end.x, cell.Y+cell.H-h-t.Space(1), 0, sp)
						}
					} else {
						for pass := range 2 {
							for ri, row := range rows {
								if picked[ri] != (pass == 1) {
									continue
								}
								col := color(ri)
								if s.on && !picked[ri] {
									col = t.TextMuted.Alpha(0.25)
								} else {
									col = col.Alpha(0.8)
								}
								x := cell.X + px(j, row[j])*cell.W
								y := cell.Y + (1-px(i, row[i]))*cell.H
								p.Fill(ui.Rect{X: x - dot, Y: y - dot, W: 2 * dot, H: 2 * dot}, col, dot)
							}
						}
					}
					if s.on && i == s.i && j == s.j {
						x0, x1 := cell.X+px(j, s.x0)*cell.W, cell.X+px(j, s.x1)*cell.W
						y0, y1 := cell.Y+(1-px(i, s.y0))*cell.H, cell.Y+(1-px(i, s.y1))*cell.H
						if i == j {
							y0, y1 = cell.Y, cell.Y+cell.H
						}
						box := ui.Rect{X: min(x0, x1), Y: min(y0, y1), W: max(1, max(x0, x1)-min(x0, x1)), H: max(1, max(y0, y1)-min(y0, y1))}
						p.Fill(box, t.Accent.Alpha(0.1), 0)
						p.Stroke(box, t.Accent, 0, 1)
					}
				}
			}
		})
	})
	return root
}

// histogram paints the counts of column k's values in 12 bins, the
// brushed rows' over all rows'.
func histogram(p *ui.Painter, t *ui.Theme, cell ui.Rect, rows [][]float64, k int, lo, hi float64, picked []bool, on bool) {
	const bins = 12
	var all, sel [bins]int
	top := 1
	for ri, row := range rows {
		b := min(bins-1, int((row[k]-lo)/(hi-lo)*bins))
		all[b]++
		if picked[ri] {
			sel[b]++
		}
		top = max(top, all[b])
	}
	area := ui.Rect{X: cell.X, Y: cell.Y + cell.H*0.3, W: cell.W, H: cell.H * 0.55}
	w := area.W / bins
	for b := range bins {
		h := area.H * float32(all[b]) / float32(top)
		col := t.TextMuted.Alpha(0.35)
		if !on {
			col = t.Accent.Alpha(0.55)
		}
		p.Fill(ui.Rect{X: area.X + float32(b)*w + 1, Y: area.Y + area.H - h, W: w - 2, H: h}, col, 1)
		if on {
			hs := area.H * float32(sel[b]) / float32(top)
			p.Fill(ui.Rect{X: area.X + float32(b)*w + 1, Y: area.Y + area.H - hs, W: w - 2, H: hs}, t.Accent, 1)
		}
	}
}

// short writes a value with three significant digits, or whole when it
// has more digits before the point.
func short(v float64) string {
	if math.Abs(v) >= 1000 {
		return fmt.Sprintf("%.0f", v)
	}
	return fmt.Sprintf("%.3g", v)
}
