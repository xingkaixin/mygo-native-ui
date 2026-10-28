// Package slopegraph compares two values of each item, before and after,
// as a slope graph or a dumbbell chart of the same data. Slope draws the
// items as lines from a left axis to a right one, labeled at both ends
// and colored by whether they rise or fall. Dumbbell gives each item a
// row, in the order given, with its two values as dots on a shared axis
// joined by a bar. Pointing at an item marks it, dims the others and
// shows its change.
//
//	slopegraph.Slope(c, &app.chart, "2020", "2025", items)
//	slopegraph.Dumbbell(c, &app.chart, "2020", "2025", items)
package slopegraph

import (
	"fmt"
	"math"
	"slices"
	"strconv"

	"github.com/egoist/mygo/ui"
)

// Item is what a chart compares: a name and its value before and after.
type Item struct {
	Name          string
	Before, After float64
}

// State is what a chart keeps: the item pointed at. Keep it in the app's
// state, one for each chart.
type State struct {
	hovered string
}

var (
	rise = ui.Hex("#3E7BFA")
	fall = ui.Hex("#F07A2B")
)

func color(it Item) ui.Color {
	if it.After < it.Before {
		return fall
	}
	return rise
}

func format(v float64) string { return strconv.FormatFloat(v, 'f', -1, 64) }

func change(it Item) string {
	d := it.After - it.Before
	s := fmt.Sprintf("%+.4g", d)
	if it.Before != 0 {
		s += fmt.Sprintf(" (%+.0f%%)", 100*d/math.Abs(it.Before))
	}
	return s
}

func extent(items []Item) (lo, hi float64) {
	lo, hi = math.Inf(1), math.Inf(-1)
	for _, it := range items {
		lo, hi = min(lo, it.Before, it.After), max(hi, it.Before, it.After)
	}
	if lo == hi {
		lo, hi = lo-1, hi+1
	}
	return lo, hi
}

// dodge moves ys apart by gap at least, keeping them between top and
// bottom where there is room, and returns where they go.
func dodge(ys []float32, gap, top, bottom float32) []float32 {
	order := make([]int, len(ys))
	for i := range order {
		order[i] = i
	}
	slices.SortStableFunc(order, func(a, b int) int { return int(math.Copysign(1, float64(ys[a]-ys[b]))) })
	out := slices.Clone(ys)
	for k, i := range order {
		if k > 0 {
			out[i] = max(out[i], out[order[k-1]]+gap)
		}
	}
	for k := len(order) - 1; k >= 0; k-- {
		i := order[k]
		limit := bottom
		if k < len(order)-1 {
			limit = out[order[k+1]] - gap
		}
		out[i] = max(min(out[i], limit), top)
	}
	return out
}

func text(p *ui.Painter, x, y float32, s string, size float32, ink ui.Color, align float32) float32 {
	w, h := p.MeasureText(0, ui.Span{Text: s, Size: size})
	p.Text(x-w*align, y-h/2, s, size, ink)
	return w
}

// slope is where a slope graph puts things in a box of w by h.
type slope struct {
	left, right, top, bottom float32
	y                        func(float64) float32
	ly, ry                   []float32
}

func layoutSlope(t *ui.Theme, items []Item, w, h float32) slope {
	lo, hi := extent(items)
	side := min(t.Space(44), w/3)
	g := slope{left: side, right: w - side, top: t.Space(12), bottom: h - t.Space(3)}
	g.y = func(v float64) float32 { return g.bottom - float32((v-lo)/(hi-lo))*(g.bottom-g.top) }
	var ly, ry []float32
	for _, it := range items {
		ly, ry = append(ly, g.y(it.Before)), append(ry, g.y(it.After))
	}
	gap := t.FontSize + 4
	g.ly = dodge(ly, gap, g.top-t.Space(2), g.bottom+t.Space(2))
	g.ry = dodge(ry, gap, g.top-t.Space(2), g.bottom+t.Space(2))
	return g
}

// Slope creates the slope graph of items, their values before titled
// before and after titled after. It is as tall as Space(90) unless given
// a height.
func Slope(c *ui.Context, s *State, before, after string, items []Item) ui.Element {
	t := c.Theme()
	box := ui.Box(c).Height(t.Space(90)).AlignSelf(ui.Stretch).Label(before + " to " + after)
	lb := box.LayoutBounds()
	x, y, over := box.PointerPosition()
	s.hovered = ""
	if over && len(items) > 0 {
		g := layoutSlope(t, items, lb.W, lb.H)
		best := t.Space(2)
		for i, it := range items {
			var d float32
			switch {
			case x < g.left:
				d = abs(y - g.ly[i])
			case x > g.right:
				d = abs(y - g.ry[i])
			default:
				f := (x - g.left) / (g.right - g.left)
				d = abs(y - (g.y(it.Before)*(1-f) + g.y(it.After)*f))
			}
			if d < best {
				best, s.hovered = d, it.Name
			}
		}
	}
	hovered := s.hovered
	box.Draw(func(p *ui.Painter, b ui.Rect) {
		if len(items) == 0 {
			return
		}
		g := layoutSlope(t, items, b.W, b.H)
		fs := t.FontSize - 1
		for _, ax := range []struct {
			x     float32
			title string
		}{{g.left, before}, {g.right, after}} {
			p.Line(b.X+ax.x, b.Y+g.top-t.Space(2), b.X+ax.x, b.Y+g.bottom+t.Space(2), 1, t.Border)
			text(p, b.X+ax.x, b.Y+t.Space(3), ax.title, t.FontSize, t.Text, 0.5)
		}
		order := make([]int, 0, len(items))
		for i, it := range items {
			if it.Name != hovered {
				order = append(order, i)
			}
		}
		for i, it := range items {
			if it.Name == hovered {
				order = append(order, i)
			}
		}
		for _, i := range order {
			it := items[i]
			col, ink, muted, width := color(it), t.Text, t.TextMuted, float32(2)
			if hovered != "" && it.Name != hovered {
				col, ink, muted = col.Alpha(0.2), ink.Alpha(0.3), muted.Alpha(0.3)
			} else if it.Name == hovered {
				width = 3
			}
			x0, y0, x1, y1 := b.X+g.left, b.Y+g.y(it.Before), b.X+g.right, b.Y+g.y(it.After)
			var line, d0, d1 ui.Path
			line.MoveTo(x0, y0).LineTo(x1, y1)
			p.StrokePath(&line, width, col)
			p.FillPath(d0.Circle(x0, y0, 3.5), col)
			p.FillPath(d1.Circle(x1, y1, 3.5), col)
			w := text(p, x0-t.Space(2.5), b.Y+g.ly[i], format(it.Before), fs, ink, 1)
			text(p, x0-t.Space(4.5)-w, b.Y+g.ly[i], it.Name, fs, muted, 1)
			w = text(p, x1+t.Space(2.5), b.Y+g.ry[i], format(it.After), fs, ink, 0)
			w += text(p, x1+t.Space(4.5)+w, b.Y+g.ry[i], it.Name, fs, muted, 0)
			if it.Name == hovered {
				text(p, x1+t.Space(6.5)+w, b.Y+g.ry[i], change(it), fs, color(it), 0)
			}
		}
	})
	return box
}

// ticks returns round values to mark from lo to hi, about n of them.
func ticks(lo, hi float64, n int) []float64 {
	raw := (hi - lo) / float64(n)
	step := math.Pow(10, math.Floor(math.Log10(raw)))
	for _, m := range []float64{1, 2, 5, 10} {
		if step*m >= raw {
			step *= m
			break
		}
	}
	var out []float64
	for v := math.Floor(lo/step) * step; v <= math.Ceil(hi/step)*step+step/2; v += step {
		out = append(out, v)
	}
	return out
}

// Dumbbell creates the dumbbell chart of items, a row each, their values
// before titled before and after titled after.
func Dumbbell(c *ui.Context, s *State, before, after string, items []Item) ui.Element {
	t := c.Theme()
	row, header := t.Space(9), t.Space(10)
	box := ui.Box(c).Height(header + row*float32(len(items))).AlignSelf(ui.Stretch).Label(before + " to " + after)
	_, y, over := box.PointerPosition()
	s.hovered = ""
	if i := int(math.Floor(float64((y - header) / row))); over && y >= header && i < len(items) {
		s.hovered = items[i].Name
	}
	hovered := s.hovered
	box.Draw(func(p *ui.Painter, b ui.Rect) {
		if len(items) == 0 {
			return
		}
		lo, hi := extent(items)
		marks := ticks(lo, hi, 5)
		lo, hi = marks[0], marks[len(marks)-1]
		name := t.Space(30)
		x0, x1 := b.X+name+t.Space(10), b.X+b.W-t.Space(30)
		x := func(v float64) float32 { return x0 + float32((v-lo)/(hi-lo))*(x1-x0) }
		fs := t.FontSize - 1
		for _, m := range marks {
			p.Line(x(m), b.Y+header-t.Space(1), x(m), b.Y+b.H, 1, t.Border.Alpha(0.6))
			text(p, x(m), b.Y+header-t.Space(3), format(m), fs-1, t.TextMuted, 0.5)
		}
		var ring, dot ui.Path
		ly := b.Y + t.Space(2.5)
		p.StrokePath(ring.Circle(b.X+t.Space(1.5), ly, 3.5), 1.5, t.TextMuted)
		w := text(p, b.X+t.Space(3.5), ly, before, fs, t.TextMuted, 0)
		p.FillPath(dot.Circle(b.X+t.Space(6.5)+w, ly, 4), t.TextMuted)
		text(p, b.X+t.Space(8.5)+w, ly, after, fs, t.TextMuted, 0)
		for i, it := range items {
			cy := b.Y + header + row*(float32(i)+0.5)
			col, ink, muted := color(it), t.Text, t.TextMuted
			if hovered != "" && it.Name != hovered {
				col, ink, muted = col.Alpha(0.25), ink.Alpha(0.35), muted.Alpha(0.35)
			}
			if it.Name == hovered {
				p.Fill(ui.Rect{X: b.X, Y: cy - row/2, W: b.W, H: row}, t.SurfaceHover, t.Radius)
			}
			text(p, b.X, cy, it.Name, fs, ink, 0)
			xb, xa := x(it.Before), x(it.After)
			p.Line(xb, cy, xa, cy, 4, col.Alpha(0.35))
			var r, d ui.Path
			p.FillPath(r.Circle(xb, cy, 4.5), t.Background)
			p.StrokePath(&r, 1.5, muted)
			p.FillPath(d.Circle(xa, cy, 5), col)
			left, right := it.Before, it.After
			if left > right {
				left, right = right, left
			}
			text(p, x(left)-t.Space(2.5), cy, format(left), fs, muted, 1)
			w := text(p, x(right)+t.Space(2.5), cy, format(right), fs, ink, 0)
			if it.Name == hovered {
				text(p, x(right)+t.Space(4.5)+w, cy, change(it), fs, color(it), 0)
			}
		}
	})
	return box
}

func abs(v float32) float32 { return max(v, -v) }
