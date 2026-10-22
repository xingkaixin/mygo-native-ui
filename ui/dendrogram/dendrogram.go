// Package dendrogram shows a hierarchical clustering as a tree, as
// statistics tools draw one: the items as leaves, in an order where no
// links cross, joined by right-angle links at the distance their clusters
// merged, read on an axis. Cluster, from vectors, and ClusterDistances,
// from a matrix of distances, make the tree, with single, complete,
// average or Ward linkage.
//
// A line across the tree cuts it into clusters, each in a color of its
// own, their count on the axis. Dragging the line, or along the axis,
// moves the cut; Tree.Cut gives the height that makes k clusters. The
// pointer on a link lights the cluster it makes and lists its items below,
// and on a name or a cell of the heatmap, a line below tells of it.
// The tree reads from left to right, the names at the right, which suits
// many items, or with TopDown from the top down. With columns, the items'
// values show as a heatmap beside the leaves, each column shaded from its
// lowest value to its highest.
//
//	if app.tree == nil {
//		app.tree = dendrogram.Cluster(vectors, dendrogram.Euclidean, dendrogram.Average)
//		app.cut = app.tree.Cut(4)
//	}
//	dendrogram.View(c, &app.chart, app.tree, app.items, columns, &app.cut).Grow(1)
package dendrogram

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/chart"
)

// Item is an item of the tree: its name, at its leaf, and its values,
// which show in the heatmap, one for each column.
type Item struct {
	Name   string
	Values []float64
}

// State is what a view keeps: the press dragging the cut. TopDown, which
// the app sets, puts the root at the top and the leaves below. Keep it in
// the app's state, one for each view.
type State struct {
	TopDown bool
	// drag is what the press under way began on: 0 before one does, 1 the
	// cut and 2 anything else.
	drag int
}

// geometry places the tree in the box. Along the leaves, u, they are step
// apart from u0; across them, v, a node is at its height's share of vlen
// from base, toward the root, and the heatmap and names are past base, at
// negative v.
type geometry struct {
	topDown              bool
	u0, step, base, vlen float32
}

func (g geometry) pt(u, v float32) (x, y float32) {
	if g.topDown {
		return u, g.base - v
	}
	return g.base - v, u
}

func (g geometry) uv(x, y float32) (u, v float32) {
	if g.topDown {
		return x, g.base - y
	}
	return y, g.base - x
}

// line draws the straight line from u0, v0 to u1, v1.
func (g geometry) line(p *ui.Painter, r ui.Rect, u0, v0, u1, v1, width float32, c ui.Color) {
	x0, y0 := g.pt(u0, v0)
	x1, y1 := g.pt(u1, v1)
	p.Line(r.X+x0, r.Y+y0, r.X+x1, r.Y+y1, width, c)
}

// rect returns the rectangle from u0, v0 to u1, v1.
func (g geometry) rect(r ui.Rect, u0, v0, u1, v1 float32) ui.Rect {
	x0, y0 := g.pt(u0, v0)
	x1, y1 := g.pt(u1, v1)
	return ui.Rect{X: r.X + min(x0, x1), Y: r.Y + min(y0, y1), W: abs(x1 - x0), H: abs(y1 - y0)}
}

// View creates the view of tree, its leaves named by items, in the order
// it was clustered from, cut at the height *cut. With columns, the items'
// values show beside the leaves, a column of the heatmap for each. Changed
// reports the user moving the cut. Give it a size, or Grow.
func View(c *ui.Context, s *State, tree *Tree, items []Item, columns []string, cut *float64) ui.Element {
	column := ui.Column(c)
	column.Children(func() {
		build(c, s, tree, items, columns, cut, column)
	})
	return column
}

// build builds the inside of the view, which reports in column a change of
// the cut.
func build(c *ui.Context, s *State, tree *Tree, items []Item, columns []string, cut *float64, column ui.Element) {
	t := c.Theme()
	n := tree.n
	name := func(i int) string {
		if i < len(items) {
			return items[i].Name
		}
		return ""
	}
	value := func(i, j int) float64 {
		if i < len(items) && j < len(items[i].Values) {
			return items[i].Values[j]
		}
		return math.NaN()
	}
	var top float64
	if m := len(tree.merges); m > 0 {
		top = tree.merges[m-1].height
	}
	scale := chart.NewLinear(0, top).Nice(5)
	if scale.Hi <= 0 {
		scale = chart.NewLinear(0, 1)
	}
	ticks := scale.Ticks(5)
	of, k := tree.Clusters(*cut)

	small := t.FontSize - 2
	pad, gap, grip, swatch := t.Space(3), t.Space(2), t.Space(1.5), t.Space(1.5)
	measure := func(s string, size float32) float32 {
		w, _ := c.MeasureText(0, ui.Span{Text: s, Size: size})
		return w
	}
	_, lh := c.MeasureText(0, ui.Span{Text: "0", Size: small})
	var tickW float32
	for _, tk := range ticks {
		tickW = max(tickW, measure(tk.Label, small))
	}
	count := plural(k, "cluster")
	pillW, pillH := measure(count, small)+t.Space(3), lh+t.Space(1)
	var colW float32
	for _, col := range columns {
		colW = max(colW, measure(col, small))
	}

	view := ui.Box(c).Grow(1).Clip().Role(ui.RoleImage).
		Label(fmt.Sprintf("Dendrogram of %s in %s", plural(n, "item"), count))
	b := view.LayoutBounds()
	g := geometry{topDown: s.TopDown}
	// The axis and the pill run along the leaves' far side when the tree
	// reads left to right, and down their near side from the top.
	var axisU, uEnd, cell float32
	if g.topDown {
		g.u0 = pad + max(tickW, pillW, colW) + t.Space(2)
		g.step = (b.W - g.u0 - pad) / float32(max(1, n))
		cell = min(t.Space(3.5), max(t.Space(1.5), 0.3*b.H/float32(max(1, len(columns)))))
	} else {
		g.u0 = pad
		if len(columns) > 0 {
			g.u0 += colW + t.Space(1)
		}
		g.step = (b.H - g.u0 - pad - t.Space(2) - max(lh, pillH)) / float32(max(1, n))
		cell = min(t.Space(3.5), max(t.Space(1.5), 0.3*b.W/float32(max(1, len(columns)))))
	}
	g.step = max(1, g.step)
	uEnd = g.u0 + float32(n)*g.step
	nameSize := min(t.FontSize-1, g.step/1.2)
	stride := 1
	if nameSize < 8 {
		nameSize, stride = 8, int(math.Ceil(float64(8*1.2/g.step)))
	}
	var nameW float32
	for i := range n {
		nameW = max(nameW, measure(name(i), nameSize))
	}
	heatLen := float32(len(columns)) * cell
	namesAt := gap
	if len(columns) > 0 {
		namesAt += heatLen + gap
	}
	if g.topDown {
		axisU = g.u0 - t.Space(1)
		g.base = b.H - pad - namesAt - swatch - t.Space(1.5) - nameW
		g.vlen = g.base - pad - max(lh, pillH)/2
	} else {
		axisU = uEnd + t.Space(1)
		g.base = b.W - pad - namesAt - swatch - t.Space(1.5) - nameW
		g.vlen = g.base - pad - max(tickW, pillW)/2
	}
	g.vlen = max(1, g.vlen)
	vOf := func(h float64) float32 { return float32(h/scale.Hi) * g.vlen }

	// u and lead are each node's place along the leaves and its first
	// leaf, which gives a cluster's color.
	nodes := n + len(tree.merges)
	u := make([]float32, nodes)
	lead := make([]int, nodes)
	for rank, leaf := range tree.order {
		u[leaf], lead[leaf] = g.u0+(float32(rank)+0.5)*g.step, leaf
	}
	for i, m := range tree.merges {
		u[n+i], lead[n+i] = (u[m.a]+u[m.b])/2, lead[m.a]
	}

	hovered, row, col := -1, -1, -1
	px, py, over := view.PointerPosition()
	pu, pv := g.uv(px, py)
	inLeaves := pu >= g.u0 && pu < uEnd
	onAxis := pv >= -gap && pv <= g.vlen+gap && (g.topDown && pu < g.u0 || !g.topDown && pu >= uEnd)
	onCut := onAxis || inLeaves && pv >= -gap && abs(pv-vOf(*cut)) <= grip
	switch {
	case !over || n == 0:
	case inLeaves && len(columns) > 0 && pv < -gap && pv >= -gap-heatLen:
		row, col = int((pu-g.u0)/g.step), int((-pv-gap)/cell)
	case inLeaves && pv < -namesAt+gap/2:
		hovered = tree.order[int((pu-g.u0)/g.step)]
	case !onCut:
		// The innermost merge whose links frame the pointer.
		for i, m := range tree.merges {
			lo, hi := min(u[m.a], u[m.b]), max(u[m.a], u[m.b])
			if pu >= lo-grip && pu <= hi+grip && pv >= min(vOf(tree.height(m.a)), vOf(tree.height(m.b))) &&
				pv <= vOf(m.height)+grip && (hovered < 0 || m.height < tree.height(hovered)) {
				hovered = n + i
			}
		}
	}

	_, _, pressing := view.Dragged()
	switch {
	case !pressing:
		s.drag = 0
	case s.drag == 0 && onCut:
		s.drag = 1
	case s.drag == 0:
		s.drag = 2
	}
	changed := false
	if clicked := view.Clicked(); s.drag == 1 || clicked && onAxis {
		h := scale.Hi * float64(max(0, min(1, pv/g.vlen)))
		if h != *cut {
			*cut, changed = h, true
			of, k = tree.Clusters(h)
			count = plural(k, "cluster")
			pillW = measure(count, small) + t.Space(3)
		}
	}
	if s.drag == 1 || onCut && s.drag == 0 && over {
		if g.topDown {
			view.Cursor(ui.CursorResizeNS)
		} else {
			view.Cursor(ui.CursorResizeEW)
		}
	}

	inside := make([]bool, nodes)
	if hovered >= 0 {
		var walk func(node int)
		walk = func(node int) {
			inside[node] = true
			if node >= n {
				walk(tree.merges[node-n].a)
				walk(tree.merges[node-n].b)
			}
		}
		walk(hovered)
	}
	lows, highs := ranges(len(columns), n, value)

	dark := t.Dark
	hue := func(cluster int) ui.Color {
		l := float32(0.6)
		if dark {
			l = 0.72
		}
		// Around the wheel by the golden angle, so that clusters side by
		// side differ.
		return ui.Oklch(l, 0.15, float32((20+cluster*137)%360))
	}
	neutral := t.TextMuted.Mix(t.Border, 0.4)
	cutAt := *cut
	view.Draw(func(p *ui.Painter, r ui.Rect) {
		if r.W != b.W || r.H != b.H {
			c.Invalidate()
		}
		if n == 0 {
			return
		}
		vc := vOf(min(cutAt, scale.Hi))
		ink := func(node int, color ui.Color) (ui.Color, float32) {
			switch {
			case hovered < 0:
				return color, 1.5
			case inside[node]:
				return color, 2.5
			}
			return color.Alpha(0.3), 1.5
		}
		for i, m := range tree.merges {
			node := n + i
			vm := vOf(m.height)
			color, width := ink(node, neutral)
			if m.height <= cutAt {
				color, width = ink(node, hue(of[lead[node]]))
			}
			// The bar runs past the arms by half their width, to square
			// the corners.
			lo, hi := min(u[m.a], u[m.b]), max(u[m.a], u[m.b])
			g.line(p, r, lo-width/2, vm, hi+width/2, vm, width, color)
			for _, kid := range []int{m.a, m.b} {
				vk := vOf(tree.height(kid))
				if m.height > cutAt && tree.height(kid) <= cutAt {
					// Below the cut, the arm is the kid's cluster's.
					below, w := ink(kid, hue(of[lead[kid]]))
					g.line(p, r, u[kid], vk, u[kid], vc, w, below)
					g.line(p, r, u[kid], vc, u[kid], vm, width, color)
					continue
				}
				g.line(p, r, u[kid], vk, u[kid], vm, width, color)
			}
		}

		for rank, leaf := range tree.order {
			lo := g.u0 + float32(rank)*g.step
			mid := lo + g.step/2
			for j := range columns {
				v0 := -gap - float32(j)*cell
				cr := g.rect(r, lo, v0, lo+g.step, v0-cell)
				p.Fill(cr, shade(value(leaf, j), lows[j], highs[j], dark, t), 0)
			}
			half := min(g.step*0.35, t.Space(1))
			p.Fill(g.rect(r, mid-half, -namesAt, mid+half, -namesAt-swatch), hue(of[leaf]), 1)
			if rank%stride != 0 {
				continue
			}
			color, weight := t.Text, 400
			if hovered >= 0 && !inside[leaf] {
				color = t.TextMuted.Alpha(0.6)
			} else if hovered >= 0 {
				weight = 600
			}
			text := ui.Span{Text: name(leaf), Size: nameSize, Weight: weight, Color: color}
			_, th := p.MeasureText(0, text)
			x, y := g.pt(mid, -namesAt-swatch-t.Space(1.5))
			x, y = r.X+x, r.Y+y
			if g.topDown {
				p.Transform(ui.Rotation(90), x, y, func() { p.RichText(x, y-th/2, 0, text) })
			} else {
				p.RichText(x, y-th/2, 0, text)
			}
		}
		if row >= 0 {
			lo, v0 := g.u0+float32(row)*g.step, -gap-float32(col)*cell
			p.Stroke(g.rect(r, lo, v0, lo+g.step, v0-cell), t.Text, 0, 1.5)
		}
		every := int(math.Ceil(float64(small / cell)))
		for j, label := range columns {
			if j%every != 0 {
				continue
			}
			mid := -gap - (float32(j)+0.5)*cell
			if g.topDown {
				x, y := g.pt(g.u0-t.Space(1), mid)
				w, _ := p.MeasureText(0, ui.Span{Text: label, Size: small})
				p.Text(r.X+x-w, r.Y+y-lh/2, label, small, t.TextMuted)
				continue
			}
			x, y := g.pt(g.u0-t.Space(1), mid)
			x, y = r.X+x, r.Y+y
			p.Transform(ui.Rotation(-90), x, y, func() { p.Text(x, y-lh/2, label, small, t.TextMuted) })
		}

		g.line(p, r, axisU, 0, axisU, g.vlen, 1, t.Border)
		tick := t.Space(1)
		for _, tk := range ticks {
			v := vOf(tk.Value)
			w, _ := p.MeasureText(0, ui.Span{Text: tk.Label, Size: small})
			if g.topDown {
				g.line(p, r, axisU-tick, v, axisU, v, 1, t.Border)
				x, y := g.pt(axisU-tick-t.Space(1), v)
				p.Text(r.X+x-w, r.Y+y-lh/2, tk.Label, small, t.TextMuted)
			} else {
				g.line(p, r, axisU, v, axisU+tick, v, 1, t.Border)
				x, y := g.pt(axisU+tick+t.Space(0.5), v)
				p.Text(r.X+x-w/2, r.Y+y, tk.Label, small, t.TextMuted)
			}
		}

		g.line(p, r, g.u0-t.Space(1), vc, uEnd, vc, 2, t.Accent)
		var pill ui.Rect
		if g.topDown {
			x, y := g.pt(axisU-tick, vc)
			pill = ui.Rect{X: r.X + x - pillW, Y: r.Y + y - pillH/2, W: pillW, H: pillH}
		} else {
			x, y := g.pt(axisU, vc)
			pill = ui.Rect{X: r.X + max(0, min(r.W-pillW, x-pillW/2)), Y: r.Y + y, W: pillW, H: pillH}
		}
		p.Fill(pill, t.Accent, pillH/2)
		p.Text(pill.X+t.Space(1.5), pill.Y+(pillH-lh)/2, count, small, t.AccentText)
	})
	line := fmt.Sprintf("%s, %s linkage  ·  cut at %s: %s", plural(n, "item"), tree.link, format(cutAt, top), count)
	switch {
	case row >= 0 && col < len(columns):
		leaf := tree.order[row]
		line = fmt.Sprintf("%s  ·  %s  ·  %s", name(leaf), columns[col], strconv.FormatFloat(value(leaf, col), 'g', 4, 64))
	case hovered >= 0 && hovered < n:
		line = fmt.Sprintf("%s  ·  cluster %d of %d", name(hovered), of[hovered]+1, k)
	case hovered >= 0:
		m := tree.merges[hovered-n]
		var names []string
		for _, leaf := range tree.order {
			if inside[leaf] {
				names = append(names, name(leaf))
			}
		}
		line = fmt.Sprintf("%s joined at %s: %s", plural(m.size, "item"), format(m.height, top), strings.Join(names, ", "))
	}
	ui.Divider(c)
	ui.Text(c, line).FontSize(t.FontSize-1).TextColor(t.TextMuted).FontFeatures("tnum").SingleLine().
		Padding(t.Space(1.5), t.Space(3))
	if changed {
		column.SetChanged()
	}
}

// ranges returns the lowest and the highest value of each of cols columns
// across n items.
func ranges(cols, n int, value func(i, j int) float64) (lows, highs []float64) {
	lows, highs = make([]float64, cols), make([]float64, cols)
	for j := range cols {
		lows[j], highs[j] = math.Inf(1), math.Inf(-1)
		for i := range n {
			if v := value(i, j); !math.IsNaN(v) {
				lows[j], highs[j] = min(lows[j], v), max(highs[j], v)
			}
		}
	}
	return lows, highs
}

// shade returns the color of v in a column from lo to hi: from pale to
// deep blue, deeper for more in either theme, and none for NaN.
func shade(v, lo, hi float64, dark bool, t *ui.Theme) ui.Color {
	if math.IsNaN(v) {
		return t.Text.Alpha(0.05)
	}
	f := float32(0.5)
	if hi > lo {
		f = float32((v - lo) / (hi - lo))
	}
	l, c := 0.96-0.56*f, 0.03+0.13*f
	if dark {
		l, c = 0.26+0.5*f, 0.03+0.08*f
	}
	return ui.Oklch(l, c, 255-40*f)
}

// format returns height with as many decimals as the tree's top needs for
// three figures.
func format(height, top float64) string {
	decimals := 0
	if top > 0 {
		decimals = max(0, 2-int(math.Floor(math.Log10(top))))
	}
	return strconv.FormatFloat(height, 'f', decimals, 64)
}

// plural returns n of unit: "1 item", "40 items".
func plural(n int, unit string) string {
	if n != 1 {
		unit += "s"
	}
	return strconv.Itoa(n) + " " + unit
}

func abs(v float32) float32 { return max(v, -v) }
