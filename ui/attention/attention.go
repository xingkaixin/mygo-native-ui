// Package attention shows where a transformer's attention heads look, as
// BertViz draws them: a layer chosen above, a chip for each head in its
// own color, which shows or hides it, and the tokens twice, side by
// side, with a line from each token on the left to each token on the
// right it attends to, as strong as the weight, in the color of its head.
// The pointer on a token on the left keeps only its lines and shades the
// tokens on the right by how much it attends to each; on a token on the
// right, only the lines into it. Matrix shows the heads shown, averaged,
// as a grid of queries down the side and keys across the top, with a
// cell's weight on hover.
//
//	attention.View(c, &app.attention, attention.Attention{Tokens: toks, Weights: w})
package attention

import (
	"fmt"
	"strconv"

	"github.com/egoist/mygo/ui"
)

// Attention is what a model attended to: its tokens and its weights,
// Weights[layer][head][query][key], each query's row summing to 1.
type Attention struct {
	Tokens  []string
	Weights [][][][]float32
}

// State is what a view keeps: the layer shown, the heads hidden, and
// whether it shows the matrix. Keep it in the app's state, one for each
// view.
type State struct {
	Layer  int
	hidden uint64
	matrix bool
}

// Color returns the color of head h of n.
func Color(t *ui.Theme, h, n int) ui.Color {
	l := float32(0.6)
	if t.Dark {
		l = 0.72
	}
	return ui.Oklch(l, 0.16, 20+float32(h)*360/float32(max(n, 1)))
}

// View creates the view of a. Give it a size, or grow it in its parent.
func View(c *ui.Context, s *State, a Attention) ui.Element {
	t := c.Theme()
	layers := len(a.Weights)
	s.Layer = max(0, min(s.Layer, layers-1))
	heads := 0
	if layers > 0 {
		heads = len(a.Weights[s.Layer])
	}
	return ui.Column(c).Gap(t.Space(3)).Children(func() {
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Wrap().Children(func() {
			ui.Row(c).Gap(t.Space(1)).AlignItems(ui.Center).Children(func() {
				if ui.Button(c, "‹").Label("Previous layer").Disabled(s.Layer == 0).Clicked() {
					s.Layer--
				}
				ui.Text(c, fmt.Sprintf("Layer %d of %d", s.Layer+1, layers)).FontFeatures("tnum").MinWidth(t.Space(24)).TextAlign(ui.Center)
				if ui.Button(c, "›").Label("Next layer").Disabled(s.Layer >= layers-1).Clicked() {
					s.Layer++
				}
			})
			ui.Row(c).Gap(t.Space(1)).Wrap().Grow(1).Children(func() {
				for h := range heads {
					chip(c.Key(h), s, h, heads)
				}
			})
			view := 0
			if s.matrix {
				view = 1
			}
			ui.Segmented(c, &view, "Lines", "Matrix")
			s.matrix = view == 1
		})
		if heads == 0 {
			return
		}
		w := a.Weights[s.Layer]
		if s.matrix {
			matrix(c, a.Tokens, average(w, s.hidden))
		} else {
			lines(c, s, a.Tokens, w)
		}
	})
}

// chip creates the chip of head h of n, which shows or hides it.
func chip(c *ui.Context, s *State, h, n int) {
	t := c.Theme()
	bit := uint64(1) << h
	on := s.hidden&bit == 0
	col := Color(t, h, n)
	b := ui.ButtonBase(c).Row().Gap(t.Space(1)).AlignItems(ui.Center).Padding(t.Space(0.5), t.Space(2)).
		Radius(t.Space(3)).Border(1, t.Border).Checked(on).Label("Head " + strconv.Itoa(h+1))
	if on {
		b.Background(col.Alpha(0.14)).BorderColor(col.Alpha(0.5))
	} else if b.Hovered() {
		b.Background(t.SurfaceHover)
	}
	if b.Clicked() {
		s.hidden ^= bit
	}
	b.Children(func() {
		dot := ui.Box(c).Size(t.Space(2), t.Space(2)).Radius(t.Space(1))
		if on {
			dot.Background(col)
		} else {
			dot.Border(1.5, col)
		}
		ui.Text(c, strconv.Itoa(h+1)).FontSize(t.FontSize - 2).FontFeatures("tnum")
	})
}

// lines creates the two columns of toks and the lines of the heads shown
// of w between them.
func lines(c *ui.Context, s *State, toks []string, w [][][]float32) {
	t := c.Theme()
	n := len(toks)
	size := t.FontSize
	_, rowH := c.MeasureText(0, ui.Span{Text: "Mg", Size: size})
	rowH += t.Space(1.5)
	colW := float32(0)
	for _, tk := range toks {
		tw, _ := c.MeasureText(0, ui.Span{Text: tk, Size: size, Font: "monospace"})
		colW = max(colW, tw)
	}
	colW += t.Space(3)
	gap := t.Space(48)
	box := ui.Box(c).Size(2*colW+gap, rowH*float32(n)).Role(ui.RoleImage).Label("Attention lines")
	px, py, over := box.PointerPosition()
	from, to := -1, -1
	if over && py >= 0 {
		i := min(int(py/rowH), n-1)
		switch {
		case px < colW:
			from = i
		case px > colW+gap:
			to = i
		}
	}
	box.Draw(func(p *ui.Painter, r ui.Rect) {
		lx, rx := r.X+colW, r.X+colW+gap
		y := func(i int) float32 { return r.Y + (float32(i)+0.5)*rowH }
		if from >= 0 {
			for k := range n {
				v := mean(w, s.hidden, from, k)
				p.Fill(ui.Rect{X: rx, Y: r.Y + float32(k)*rowH + 1, W: colW, H: rowH - 2}, t.Accent.Alpha(v*0.8), t.Radius)
			}
		}
		for h, head := range w {
			if s.hidden&(1<<h) != 0 {
				continue
			}
			col := Color(t, h, len(w))
			for q := range n {
				if from >= 0 && q != from {
					continue
				}
				for k := range n {
					if to >= 0 && k != to {
						continue
					}
					v := head[q][k]
					if v < 0.02 {
						continue
					}
					p.StrokePolyline([][2]float32{{lx + t.Space(1), y(q)}, {rx - t.Space(1), y(k)}}, 1+2*v, col.Alpha(min(1, v*1.2)))
				}
			}
		}
		for i, tk := range toks {
			for side, x := range []float32{r.X, rx} {
				lit := side == 0 && i == from || side == 1 && i == to
				sp := ui.Span{Text: tk, Size: size, Font: "monospace", Color: t.Text}
				if lit {
					sp.Weight = 700
				}
				tw, th := p.MeasureText(0, sp)
				tx := x + t.Space(1.5)
				if side == 0 {
					tx = x + colW - t.Space(1.5) - tw
				}
				if lit {
					p.Fill(ui.Rect{X: x, Y: r.Y + float32(i)*rowH + 1, W: colW, H: rowH - 2}, t.Text.Alpha(0.08), t.Radius)
				}
				p.RichText(tx, y(i)-th/2, 0, sp)
			}
		}
	})
}

// matrix creates the grid of m, queries down the side and keys across
// the top, shaded by weight, with a cell's weight on hover.
func matrix(c *ui.Context, toks []string, m [][]float32) {
	t := c.Theme()
	n := len(toks)
	size := t.FontSize - 1
	label := float32(0)
	for _, tk := range toks {
		tw, _ := c.MeasureText(0, ui.Span{Text: tk, Size: size, Font: "monospace"})
		label = max(label, tw)
	}
	label += t.Space(2)
	cell := t.Space(8)
	box := ui.Box(c).Size(label+cell*float32(n), label+cell*float32(n)).Role(ui.RoleImage).Label("Attention matrix")
	px, py, over := box.PointerPosition()
	hq, hk := -1, -1
	if over && px > label && py > label {
		hq, hk = min(int((py-label)/cell), n-1), min(int((px-label)/cell), n-1)
		box.Tooltip(fmt.Sprintf("%s → %s  %.3f", toks[hq], toks[hk], m[hq][hk]))
	}
	peak := float32(0)
	for _, row := range m {
		for _, v := range row {
			peak = max(peak, v)
		}
	}
	box.Draw(func(p *ui.Painter, r ui.Rect) {
		ox, oy := r.X+label, r.Y+label
		for q := range n {
			for k := range n {
				v := m[q][k] / max(peak, 1e-6)
				cr := ui.Rect{X: ox + float32(k)*cell, Y: oy + float32(q)*cell, W: cell - 1, H: cell - 1}
				p.Fill(cr, t.Text.Alpha(0.04), 2)
				p.Fill(cr, t.Accent.Alpha(v), 2)
			}
		}
		if hq >= 0 {
			p.Stroke(ui.Rect{X: ox + float32(hk)*cell - 1, Y: oy + float32(hq)*cell - 1, W: cell + 1, H: cell + 1}, t.Text, 3, 1.5)
		}
		for i, tk := range toks {
			col := t.TextMuted
			if i == hq {
				col = t.Text
			}
			sp := ui.Span{Text: tk, Size: size, Font: "monospace", Color: col}
			tw, th := p.MeasureText(0, sp)
			p.RichText(ox-t.Space(1.5)-tw, oy+(float32(i)+0.5)*cell-th/2, 0, sp)
			sp.Color = t.TextMuted
			if i == hk {
				sp.Color = t.Text
			}
			cx, cy := ox+(float32(i)+0.5)*cell, oy-t.Space(1.5)
			p.Transform(ui.Rotation(-90), cx, cy, func() {
				p.RichText(cx, cy-th/2, 0, sp)
			})
		}
	})
}

// mean returns the mean weight from q to k over the heads of w shown.
func mean(w [][][]float32, hidden uint64, q, k int) float32 {
	sum, n := float32(0), 0
	for h, head := range w {
		if hidden&(1<<h) == 0 {
			sum += head[q][k]
			n++
		}
	}
	if n == 0 {
		return 0
	}
	return sum / float32(n)
}

// average returns the mean of the heads of w shown.
func average(w [][][]float32, hidden uint64) [][]float32 {
	n := len(w[0])
	out := make([][]float32, n)
	for q := range n {
		out[q] = make([]float32, n)
		for k := range n {
			out[q][k] = mean(w, hidden, q, k)
		}
	}
	return out
}
