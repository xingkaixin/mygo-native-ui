// Package sankey shows how amounts flow from one stage to the next, as
// funnel and budget reports do: the stages in columns from the sources to
// where the flows end, each a bar as tall as the most that goes in or out
// of it, and a band for each flow as wide as its amount, in the color it
// comes from. The pointer lights a band or the bands of a stage, and the
// line below says what it is.
//
//	sankey.View(c, stages, flows, "users").Grow(1)
package sankey

import (
	"cmp"
	"fmt"
	"slices"

	"github.com/egoist/mygo/ui"
)

// Flow is an amount from stage From to stage To, by their indexes.
type Flow struct {
	From, To int
	Value    float64
}

type node struct {
	col         int
	x, y, h     float32
	in, out     float64
	inAt, outAt float32
}

// band is where a flow is: its center at each end, and its width.
type band struct {
	x0, y0, x1, y1, w float32
}

// layout places the stages in w×h, those that lead nowhere in the last
// column, and orders each column by where its flows come from and go to,
// so that few bands cross.
func layout(n int, flows []Flow, w, h, barW, gap float32) ([]node, []band) {
	nodes := make([]node, n)
	ok := func(f Flow) bool {
		return f.From >= 0 && f.From < n && f.To >= 0 && f.To < n && f.From != f.To && f.Value > 0
	}
	state := make([]uint8, n)
	var place func(i int) int
	place = func(i int) int {
		if state[i] == 1 {
			return 0
		}
		if state[i] == 2 {
			return nodes[i].col
		}
		state[i] = 1
		col := 0
		for _, f := range flows {
			if ok(f) && f.To == i {
				col = max(col, place(f.From)+1)
			}
		}
		nodes[i].col, state[i] = col, 2
		return col
	}
	cols := 0
	for i := range n {
		cols = max(cols, place(i)+1)
	}
	for _, f := range flows {
		if ok(f) {
			nodes[f.From].out += f.Value
			nodes[f.To].in += f.Value
		}
	}
	for i := range nodes {
		if nodes[i].out == 0 && nodes[i].in > 0 {
			nodes[i].col = cols - 1
		}
	}
	ids := make([][]int, cols)
	for i, nd := range nodes {
		ids[nd.col] = append(ids[nd.col], i)
	}
	value := func(i int) float64 { return max(nodes[i].in, nodes[i].out) }
	k := float32(0)
	for _, col := range ids {
		var sum float64
		for _, i := range col {
			sum += value(i)
		}
		if sum > 0 {
			s := (h - gap*float32(len(col)-1)) / float32(sum)
			if k == 0 || s < k {
				k = s
			}
		}
	}
	stack := func(col []int) {
		total := gap * float32(len(col)-1)
		for _, i := range col {
			total += float32(value(i)) * k
		}
		y := (h - total) / 2
		for _, i := range col {
			nodes[i].y, nodes[i].h = y, float32(value(i))*k
			y += nodes[i].h + gap
		}
	}
	for _, col := range ids {
		stack(col)
	}
	mid := func(i int) float32 { return nodes[i].y + nodes[i].h/2 }
	sweep := func(c int, back bool) {
		key := map[int]float32{}
		for _, i := range ids[c] {
			var sum, wt float32
			for _, f := range flows {
				if !ok(f) {
					continue
				}
				switch {
				case !back && f.To == i:
					sum, wt = sum+mid(f.From)*float32(f.Value), wt+float32(f.Value)
				case back && f.From == i:
					sum, wt = sum+mid(f.To)*float32(f.Value), wt+float32(f.Value)
				}
			}
			key[i] = mid(i)
			if wt > 0 {
				key[i] = sum / wt
			}
		}
		slices.SortStableFunc(ids[c], func(a, b int) int { return cmp.Compare(key[a], key[b]) })
		stack(ids[c])
	}
	for range 3 {
		for c := 1; c < cols; c++ {
			sweep(c, false)
		}
		for c := cols - 2; c >= 0; c-- {
			sweep(c, true)
		}
	}
	for i := range nodes {
		if cols > 1 {
			nodes[i].x = float32(nodes[i].col) * (w - barW) / float32(cols-1)
		}
	}
	// Each stage's bands leave and arrive in the order of their other
	// ends, top to bottom, so that they do not cross at the bars.
	order := make([]int, 0, len(flows))
	for i, f := range flows {
		if ok(f) {
			order = append(order, i)
		}
	}
	bands := make([]band, len(flows))
	slices.SortStableFunc(order, func(a, b int) int { return cmp.Compare(mid(flows[a].To), mid(flows[b].To)) })
	for _, i := range order {
		f := flows[i]
		src := &nodes[f.From]
		wd := float32(f.Value) * k
		bands[i].x0, bands[i].y0, bands[i].w = src.x+barW, src.y+src.outAt+wd/2, wd
		src.outAt += wd
	}
	slices.SortStableFunc(order, func(a, b int) int { return cmp.Compare(mid(flows[a].From), mid(flows[b].From)) })
	for _, i := range order {
		f := flows[i]
		dst := &nodes[f.To]
		bands[i].x1, bands[i].y1 = dst.x, dst.y+dst.inAt+bands[i].w/2
		dst.inAt += bands[i].w
	}
	return nodes, bands
}

// center returns where the middle of b is at x, and whether x is along
// it.
func (b band) center(x float32) (float32, bool) {
	if x < b.x0 || x > b.x1 || b.x1 <= b.x0 {
		return 0, false
	}
	// x runs along the curve as the smoothstep of t does, which the
	// bisection inverts.
	lo, hi := float32(0), float32(1)
	for range 20 {
		t := (lo + hi) / 2
		if b.x0+(b.x1-b.x0)*t*t*(3-2*t) < x {
			lo = t
		} else {
			hi = t
		}
	}
	t := (lo + hi) / 2
	return b.y0 + (b.y1-b.y0)*t*t*(3-2*t), true
}

func (b band) path() *ui.Path {
	dx := (b.x1 - b.x0) / 2
	top, bot := b.w/2, b.w/2
	var p ui.Path
	p.MoveTo(b.x0, b.y0-top).CubeTo(b.x0+dx, b.y0-top, b.x1-dx, b.y1-top, b.x1, b.y1-top).
		LineTo(b.x1, b.y1+bot).CubeTo(b.x1-dx, b.y1+bot, b.x0+dx, b.y0+bot, b.x0, b.y0+bot).Close()
	return &p
}

// View creates the diagram of flows between stages. Give it a size, or
// Grow.
func View(c *ui.Context, stages []string, flows []Flow, unit string) ui.Element {
	t := c.Theme()
	barW, gap := t.Space(3), t.Space(4)
	small := t.FontSize - 1
	return ui.Column(c).Gap(t.Space(3)).Children(func() {
		chart := ui.Box(c).Grow(1).Shrink(1).AlignSelf(ui.Stretch)
		r := chart.LayoutBounds()
		nodes, bands := layout(len(stages), flows, r.W, r.H, barW, gap)
		hotNode, hotFlow := -1, -1
		if px, py, ok := chart.PointerPosition(); ok {
			for i, nd := range nodes {
				if px >= nd.x-2 && px <= nd.x+barW+2 && py >= nd.y && py <= nd.y+nd.h {
					hotNode = i
				}
			}
			if hotNode < 0 {
				for i, b := range bands {
					if y, ok := b.center(px); ok && py >= y-b.w/2 && py <= y+b.w/2 {
						hotFlow = i
					}
				}
			}
		}
		lit := func(i int) bool {
			f := flows[i]
			return i == hotFlow || f.From == hotNode || f.To == hotNode
		}
		chart.Draw(func(p *ui.Painter, r ui.Rect) {
			nodes, bands := layout(len(stages), flows, r.W, r.H, barW, gap)
			for i := range nodes {
				nodes[i].x, nodes[i].y = nodes[i].x+r.X, nodes[i].y+r.Y
			}
			for i := range bands {
				b := &bands[i]
				b.x0, b.y0, b.x1, b.y1 = b.x0+r.X, b.y0+r.Y, b.x1+r.X, b.y1+r.Y
			}
			for i, b := range bands {
				if b.w == 0 {
					continue
				}
				alpha := float32(0.35)
				switch {
				case lit(i):
					alpha = 0.65
				case hotNode >= 0 || hotFlow >= 0:
					alpha = 0.12
				}
				p.FillPath(b.path(), hue(flows[i].From).Alpha(alpha))
			}
			last := 0
			for _, nd := range nodes {
				last = max(last, nd.col)
			}
			for i, nd := range nodes {
				if nd.h == 0 {
					continue
				}
				p.Fill(ui.Rect{X: nd.x, Y: nd.y, W: barW, H: nd.h}, hue(i), 2)
				label := []ui.Span{
					{Text: stages[i], Size: small, Weight: 600, Color: t.Text},
					{Text: "  " + number(max(nd.in, nd.out)), Size: small, Color: t.TextMuted},
				}
				w, h := p.MeasureText(0, label...)
				x := nd.x + barW + t.Space(2)
				if nd.col == last && last > 0 {
					x = nd.x - t.Space(2) - w
				}
				p.RichText(x, nd.y+nd.h/2-h/2, 0, label...)
			}
		})
		line := "Point at a stage or a flow"
		switch {
		case hotFlow >= 0:
			f := flows[hotFlow]
			line = fmt.Sprintf("%s → %s: %s %s", stages[f.From], stages[f.To], number(f.Value), unit)
			if out := nodes[f.From].out; out > 0 {
				line += fmt.Sprintf(", %.0f%% of %s", f.Value/out*100, stages[f.From])
			}
		case hotNode >= 0:
			nd := nodes[hotNode]
			line = stages[hotNode] + ": "
			switch {
			case nd.in > 0 && nd.out > 0:
				line += fmt.Sprintf("%s %s in, %s out", number(nd.in), unit, number(nd.out))
			case nd.in > 0:
				line += number(nd.in) + " " + unit
			default:
				line += number(nd.out) + " " + unit
			}
		}
		ui.Text(c, line).FontFeatures("tnum").TextColor(t.TextMuted)
	})
}

// hue returns stage i's color, around the wheel by the golden angle so
// that stages near one another differ.
func hue(i int) ui.Color {
	return ui.Oklch(0.66, 0.14, float32((250+i*137)%360))
}

// number writes n rounded, with commas: 12,400.
func number(n float64) string {
	s := fmt.Sprintf("%.0f", n)
	neg := s[0] == '-'
	if neg {
		s = s[1:]
	}
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	if neg {
		s = "-" + s
	}
	return s
}
