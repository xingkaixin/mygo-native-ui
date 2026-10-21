package flowchart

import (
	"cmp"
	"math"
	"slices"
	"strings"

	"github.com/egoist/mygo/ui"
)

// sizes are the sizes of the parts of a chart, from the theme.
type sizes struct {
	th, sh, padX, padY, wrap, min, radius, sub, ring, dot, head, gap, lane, band, port, inset, loop, pad float32
}

func measure(c *ui.Context) sizes {
	t := c.Theme()
	_, th := c.MeasureText(0, ui.Span{Text: "Ag", Size: t.FontSize})
	_, sh := c.MeasureText(0, ui.Span{Text: "Ag", Size: t.FontSize - 1})
	return sizes{th: th, sh: sh, padX: t.Space(4), padY: t.Space(2.5), wrap: t.Space(48), min: t.Space(12),
		radius: t.Space(2), sub: t.Space(2), ring: t.Space(1.25), dot: t.Space(1.1), head: t.Space(2.25),
		gap: t.Space(8), lane: t.Space(3), band: t.Space(6), port: t.Space(2.5),
		inset: t.Space(3), loop: t.Space(6), pad: t.Space(5)}
}

// rim returns how tall the ellipse at either end of a cylinder w wide is,
// half way across.
func rim(w float32, z sizes) float32 { return max(z.sub*0.75, min(z.sub*1.25, w/10)) }

// fit returns the size of a node of shape sh around text tw wide and tb
// tall.
func fit(sh shape, tw, tb float32, z sizes) (float32, float32) {
	w, h := tw+2*z.padX, tb+2*z.padY
	switch sh {
	case stadium:
		return max(w+h/3, z.min), h
	case subroutine:
		return max(w+2*z.sub, z.min), h
	case cylinder:
		w = max(w, z.min)
		return w, h + 2*rim(w, z)
	case circle, double:
		d := max(float32(math.Hypot(float64(tw), float64(tb)))+2*z.padY, z.min)
		if sh == double {
			d += 2 * z.ring
		}
		return d, d
	case diamond:
		// A diamond k times as wide as it is tall holds a box bw by bh
		// when it is bw + k·bh wide.
		bw, bh := tw+2*z.sub, tb+1.5*z.sub
		k := max(1, min(2, bw/bh))
		w = max(bw+k*bh, z.min)
		return w, w / k
	case hexagon:
		return w + h/2, h
	case flag:
		return w + h/3, h
	case lean, leanLeft:
		return w + h*0.4, h
	case trapezoid, inverted:
		return w + h*0.5, h
	}
	return max(w, z.min), h
}

// outline returns the corners of the shape of n about its center, those of
// round shapes many along their curves.
func outline(n node, z sizes) [][2]float32 {
	w, h := n.w/2, n.h/2
	arc := func(cx, cy, rx, ry, a0, a1 float64, out [][2]float32) [][2]float32 {
		const steps = 12
		for k := 0; k <= steps; k++ {
			a := a0 + (a1-a0)*float64(k)/steps
			out = append(out, [2]float32{float32(cx + rx*math.Cos(a)), float32(cy + ry*math.Sin(a))})
		}
		return out
	}
	switch n.shape {
	case circle, double:
		return arc(0, 0, float64(w), float64(w), 0, 2*math.Pi, nil)[:12]
	case stadium:
		r := float64(h)
		out := arc(float64(w)-r, 0, r, r, -math.Pi/2, math.Pi/2, nil)
		return arc(r-float64(w), 0, r, r, math.Pi/2, 3*math.Pi/2, out)
	case cylinder:
		ry := float64(rim(n.w, z))
		out := arc(0, float64(-h)+ry, float64(w), ry, math.Pi, 2*math.Pi, nil)
		return arc(0, float64(h)-ry, float64(w), ry, 0, math.Pi, out)
	case diamond:
		return [][2]float32{{0, -h}, {w, 0}, {0, h}, {-w, 0}}
	case hexagon:
		in := n.h / 4
		return [][2]float32{{-w + in, -h}, {w - in, -h}, {w, 0}, {w - in, h}, {-w + in, h}, {-w, 0}}
	case flag:
		return [][2]float32{{-w, -h}, {w, -h}, {w, h}, {-w, h}, {-w + n.h/3, 0}}
	case lean:
		s := n.h * 0.4
		return [][2]float32{{-w + s, -h}, {w, -h}, {w - s, h}, {-w, h}}
	case leanLeft:
		s := n.h * 0.4
		return [][2]float32{{-w, -h}, {w - s, -h}, {w, h}, {-w + s, h}}
	case trapezoid:
		s := n.h * 0.4
		return [][2]float32{{-w + s, -h}, {w - s, -h}, {w, h}, {-w, h}}
	case inverted:
		s := n.h * 0.4
		return [][2]float32{{-w, -h}, {w, -h}, {w - s, h}, {-w + s, h}}
	}
	return [][2]float32{{-w, -h}, {w, -h}, {w, h}, {-w, h}}
}

// reach returns where the outline poly, as u, v pairs about its center,
// meets the line across it at u, first and last along v.
func reach(poly [][2]float32, u float32) (float32, float32) {
	lo, hi := float32(math.Inf(1)), float32(math.Inf(-1))
	for i, a := range poly {
		b := poly[(i+1)%len(poly)]
		if (a[0] < u) == (b[0] < u) {
			continue
		}
		v := a[1] + (u-a[0])/(b[0]-a[0])*(b[1]-a[1])
		lo, hi = min(lo, v), max(hi, v)
	}
	if lo > hi {
		return 0, 0
	}
	return lo, hi
}

// wrap breaks s into lines no wider than width where it can, at its
// newlines and then between words, and returns them and the widest.
func wrap(c *ui.Context, s string, size float32, weight int, width float32) ([]line, float32) {
	measure := func(s string) float32 {
		w, _ := c.MeasureText(0, ui.Span{Text: s, Size: size, Weight: weight})
		return w
	}
	var out []line
	var widest float32
	for _, para := range strings.Split(s, "\n") {
		cur := ""
		for _, word := range strings.Fields(para) {
			switch {
			case cur == "":
				cur = word
			case measure(cur+" "+word) > width:
				out = append(out, line{cur, measure(cur)})
				cur = word
			default:
				cur += " " + word
			}
		}
		out = append(out, line{cur, measure(cur)})
		widest = max(widest, out[len(out)-1].w)
	}
	for _, l := range out {
		widest = max(widest, l.w)
	}
	return out, widest
}

// layout sizes the nodes, the labels and the titles, and places them.
func layout(c *ui.Context, ch *chart) {
	t := c.Theme()
	z := measure(c)
	ch.z = z
	label := func(e *edge, text string) {
		var w float32
		e.lines, w = wrap(c, text, t.FontSize-1, 400, z.wrap)
		e.lw, e.lh = w+2*t.Space(1.5), float32(len(e.lines))*z.sh+t.Space(1)
	}
	for i := range ch.edges {
		if e := &ch.edges[i]; e.text != "" && e.from != e.to {
			label(e, e.text)
		}
	}
	for i := range ch.nodes {
		n := &ch.nodes[i]
		n.lines, n.tw = wrap(c, n.text, t.FontSize, 500, z.wrap)
		n.tb = float32(len(n.lines)) * z.th
		n.w, n.h = fit(n.shape, n.tw, n.tb, z)
		// The links of a node back to itself share a loop and a label.
		var texts []string
		for _, k := range n.loops {
			if s := ch.edges[k].text; s != "" {
				texts = append(texts, s)
			}
		}
		if len(texts) > 0 {
			label(&ch.edges[n.loops[0]], strings.Join(texts, ", "))
		}
	}
	title := func(g *group, width float32) {
		if g.title == "" {
			return
		}
		g.lines, g.tw = wrap(c, g.title, t.FontSize-1, 600, width)
		g.band = float32(len(g.lines))*z.sh + t.Space(3)
	}
	vertical := ch.dir == down || ch.dir == up
	if vertical {
		for i := range ch.groups {
			title(&ch.groups[i], z.wrap)
		}
	}
	place(ch, z, title)
}

const (
	isNode uint8 = iota
	isDummy
	isLeft
	isRight
)

// vert is a vertex of the layered graph: a node, a point a link passes in
// a rank between its ends, the label of a link, or an edge of a subgraph's
// box in a rank it spans. lo and hi are how far it reaches before and
// after its center u across the ranks, size how far along them; at is its
// place in its rank, scope the subgraph it is in.
type vert struct {
	kind               uint8
	node, group, scope int
	layer              int
	lo, hi, size, u, v float32
	at                 float64
}

// route is a link as laid out: its edge, its ends in rank order, the
// ranks it spans at least, whether it goes against the ranks, and the
// vertices it passes, with that of its label, -1 for none.
type route struct {
	edge, a, b, min int
	rev             bool
	path            []int
	label           int
}

// place lays out the nodes, the links and the subgraphs, as Sugiyama's
// method does: it turns around the links that close a cycle, puts each
// node a rank after those that link to it, moves those it can to shorten
// their links, gives each link a vertex in each rank it passes and its
// label one in a rank between nodes, orders each rank so that few links
// cross, with the nodes of a subgraph together, sets each vertex across
// as near those it links to as the ones beside it let it, and runs links
// as curves through their vertices. It lays out down the ranks, and turns
// the chart the way it goes as it places it.
func place(ch *chart, z sizes, title func(g *group, width float32)) {
	n, ng := len(ch.nodes), len(ch.groups)
	if n == 0 {
		ch.w, ch.h = 0, 0
		return
	}
	vertical := ch.dir == down || ch.dir == up
	across := func(w, h float32) (float32, float32) {
		if vertical {
			return w, h
		}
		return h, w
	}
	// canon turns a point about a node's center from the chart's way to
	// the layout's, and final a point of the layout to the chart's.
	canon := func(p [2]float32) [2]float32 {
		switch ch.dir {
		case up:
			return [2]float32{p[0], -p[1]}
		case right:
			return [2]float32{p[1], p[0]}
		case left:
			return [2]float32{p[1], -p[0]}
		}
		return p
	}
	final := func(p [2]float32) [2]float32 {
		switch ch.dir {
		case up:
			return [2]float32{p[0], -p[1]}
		case right:
			return [2]float32{p[1], p[0]}
		case left:
			return [2]float32{-p[1], p[0]}
		}
		return p
	}

	var routes []route
	for k, e := range ch.edges {
		if e.from != e.to {
			routes = append(routes, route{edge: k, a: e.from, b: e.to, min: max(1, e.length), label: -1})
		}
	}
	out := make([][]int, n)
	in := make([]int, n)
	for k, r := range routes {
		out[r.a] = append(out[r.a], k)
		in[r.b]++
	}
	mark := make([]uint8, n)
	var walk func(u int)
	walk = func(u int) {
		mark[u] = 1
		for _, k := range out[u] {
			switch v := routes[k].b; mark[v] {
			case 0:
				walk(v)
			case 1:
				routes[k].rev = true
			}
		}
		mark[u] = 2
	}
	for _, roots := range []bool{true, false} {
		for u := range n {
			if mark[u] == 0 && (!roots || in[u] == 0) {
				walk(u)
			}
		}
	}
	for k := range routes {
		if r := &routes[k]; r.rev {
			r.a, r.b = r.b, r.a
		}
	}
	rank := make([]int, n)
	for range n {
		moved := false
		for _, r := range routes {
			if rank[r.a]+r.min > rank[r.b] {
				rank[r.b], moved = rank[r.a]+r.min, true
			}
		}
		if !moved {
			break
		}
	}
	// A node with more links out than in moves down as far as they let
	// it, as one that starts a chain does to just before the node it links
	// to, and one with more in than out up.
	for range 4 {
		for u := range n {
			lo, hi, ins, outs := 0, math.MaxInt, 0, 0
			for _, r := range routes {
				if r.b == u {
					lo, ins = max(lo, rank[r.a]+r.min), ins+1
				}
				if r.a == u {
					hi, outs = min(hi, rank[r.b]-r.min), outs+1
				}
			}
			switch {
			case outs > ins:
				rank[u] = hi
			case ins > outs:
				rank[u] = lo
			}
		}
	}
	low := slices.Min(rank)
	for u := range rank {
		rank[u] -= low
	}

	// Nodes are in the even layers, and labels in the odd ones between.
	first, last := make([]int, ng), make([]int, ng)
	for g := range first {
		first[g], last[g] = math.MaxInt, -1
	}
	for u, nd := range ch.nodes {
		for g := nd.group; g >= 0; g = ch.groups[g].parent {
			first[g], last[g] = min(first[g], 2*rank[u]), max(last[g], 2*rank[u])
		}
	}
	common := func(a, b int) int {
		for a >= 0 && !ch.within(b, a) {
			a = ch.groups[a].parent
		}
		return a
	}

	var vs []vert
	for u, nd := range ch.nodes {
		cw, mw := across(nd.w, nd.h)
		v := vert{kind: isNode, node: u, group: -1, scope: nd.group, layer: 2 * rank[u], lo: cw / 2, hi: cw / 2, size: mw}
		if len(nd.loops) > 0 {
			e := ch.edges[nd.loops[0]]
			v.hi += z.loop
			if len(e.lines) > 0 {
				lw, _ := across(e.lw, e.lh)
				v.hi += lw + z.lane
			}
		}
		vs = append(vs, v)
	}
	for k := range routes {
		r := &routes[k]
		e := ch.edges[r.edge]
		la, lb := 2*rank[r.a], 2*rank[r.b]
		mid := (la + lb) / 2
		if mid%2 == 0 {
			mid--
		}
		scope := common(ch.nodes[r.a].group, ch.nodes[r.b].group)
		r.path = []int{r.a}
		for L := la + 1; L < lb; L++ {
			d := vert{kind: isDummy, node: -1, group: -1, scope: scope, layer: L}
			if L == mid && len(e.lines) > 0 && e.stroke != hidden {
				cw, mw := across(e.lw, e.lh)
				d.lo, d.hi, d.size = cw/2, cw/2, mw
				r.label = len(vs)
			}
			r.path = append(r.path, len(vs))
			vs = append(vs, d)
		}
		r.path = append(r.path, r.b)
	}
	layers := 0
	for _, v := range vs {
		layers = max(layers, v.layer+1)
	}
	members := make([][]int, layers)
	for u, v := range vs {
		vs[u].at = float64(len(members[v.layer]))
		members[v.layer] = append(members[v.layer], u)
	}
	lefts, rights := make([][]int, ng), make([][]int, ng)
	for g, gr := range ch.groups {
		for L := first[g]; L <= last[g]; L++ {
			lefts[g] = append(lefts[g], len(vs))
			vs = append(vs, vert{kind: isLeft, node: -1, group: g, scope: gr.parent, layer: L})
			rights[g] = append(rights[g], len(vs))
			vs = append(vs, vert{kind: isRight, node: -1, group: g, scope: gr.parent, layer: L})
		}
	}
	ups, downs := make([][]int, len(vs)), make([][]int, len(vs))
	type segment struct{ a, b int }
	var segs []segment
	for _, r := range routes {
		for i := 1; i < len(r.path); i++ {
			a, b := r.path[i-1], r.path[i]
			downs[a], ups[b] = append(downs[a], b), append(ups[b], a)
			segs = append(segs, segment{a, b})
		}
	}

	// Each rank is ordered as a tree: the vertices of a subgraph come
	// together between its box's edges, and the subgraphs beside each
	// other keep one order in every rank, so that their boxes can be
	// rectangles that do not overlap.
	kids := make([][]int, ng+1)
	for g, gr := range ch.groups {
		kids[gr.parent+1] = append(kids[gr.parent+1], g)
	}
	order := make([]float64, ng)
	for g := range order {
		order[g] = float64(g)
	}
	key := make([]float64, len(vs))
	cols := make([][]int, layers)
	var arrange func(L, scope int, out []int) []int
	arrange = func(L, scope int, out []int) []int {
		type item struct {
			u, g     int
			key, cur float64
		}
		var items []item
		for _, u := range members[L] {
			if vs[u].scope == scope {
				items = append(items, item{u, -1, key[u], vs[u].at})
			}
		}
		for _, g := range kids[scope+1] {
			if L < first[g] || L > last[g] {
				continue
			}
			sum, cnt := 0.0, 0
			for _, u := range members[L] {
				if ch.within(vs[u].scope, g) {
					sum, cnt = sum+key[u], cnt+1
				}
			}
			cur := vs[lefts[g][L-first[g]]].at
			k := cur
			if cnt > 0 {
				k = sum / float64(cnt)
			}
			items = append(items, item{-1, g, k, cur})
		}
		slices.SortStableFunc(items, func(a, b item) int { return cmp.Or(cmp.Compare(a.key, b.key), cmp.Compare(a.cur, b.cur)) })
		var slots, gs []int
		for i, it := range items {
			if it.g >= 0 {
				slots, gs = append(slots, i), append(gs, it.g)
			}
		}
		slices.SortStableFunc(gs, func(a, b int) int { return cmp.Compare(order[a], order[b]) })
		for k, i := range slots {
			items[i].g = gs[k]
		}
		for _, it := range items {
			if it.g < 0 {
				out = append(out, it.u)
				continue
			}
			out = append(out, lefts[it.g][L-first[it.g]])
			out = arrange(L, it.g, out)
			out = append(out, rights[it.g][L-first[it.g]])
		}
		return out
	}
	rearrange := func(L int) {
		cols[L] = arrange(L, -1, cols[L][:0])
		for k, u := range cols[L] {
			vs[u].at = float64(k)
		}
	}
	sweep := func(L int, near [][]int) {
		for _, u := range members[L] {
			key[u] = vs[u].at
			if len(near[u]) > 0 {
				sum := 0.0
				for _, w := range near[u] {
					sum += vs[w].at
				}
				key[u] = sum / float64(len(near[u]))
			}
		}
		rearrange(L)
	}
	crossings := func() int {
		c := 0
		for i, s := range segs {
			for _, r := range segs[i+1:] {
				if vs[s.a].layer == vs[r.a].layer && (vs[s.a].at-vs[r.a].at)*(vs[s.b].at-vs[r.b].at) < 0 {
					c++
				}
			}
		}
		return c
	}
	save := func() []float64 {
		o := make([]float64, len(vs))
		for u := range vs {
			o[u] = vs[u].at
		}
		return o
	}
	for u := range vs {
		key[u] = vs[u].at
	}
	for L := range layers {
		rearrange(L)
	}
	best, least := save(), crossings()
	for range 8 {
		if least == 0 {
			break
		}
		sum, cnt := make([]float64, ng), make([]int, ng)
		for _, v := range vs {
			if v.kind <= isDummy {
				for g := v.scope; g >= 0; g = ch.groups[g].parent {
					sum[g], cnt[g] = sum[g]+v.at, cnt[g]+1
				}
			}
		}
		for g := range order {
			order[g] = sum[g] / float64(max(cnt[g], 1))
		}
		for L := 1; L < layers; L++ {
			sweep(L, ups)
		}
		for L := layers - 2; L >= 0; L-- {
			sweep(L, downs)
		}
		if c := crossings(); c < least {
			best, least = save(), c
		}
	}
	for u := range vs {
		vs[u].at = best[u]
	}
	for _, col := range cols {
		slices.SortFunc(col, func(a, b int) int { return cmp.Compare(vs[a].at, vs[b].at) })
	}

	// Along the ranks: each layer is a band as deep as its deepest vertex,
	// apart by a gap, and more where a subgraph's box starts or ends, for
	// its edge, its title and those of the subgraphs in it that start or
	// end there too.
	depth := make([]float32, layers)
	for _, v := range vs {
		depth[v.layer] = max(depth[v.layer], v.size)
	}
	before, after := make([]float32, ng), make([]float32, ng)
	for g := ng - 1; g >= 0; g-- {
		var b, a float32
		for _, k := range kids[g+1] {
			if first[k] == first[g] {
				b = max(b, before[k])
			}
			if last[k] == last[g] {
				a = max(a, after[k])
			}
		}
		before[g], after[g] = z.inset+b, z.inset+a
		switch ch.dir {
		case down:
			before[g] += ch.groups[g].band
		case up:
			after[g] += ch.groups[g].band
		}
	}
	starts, ends := make([]float32, layers), make([]float32, layers)
	for g := range ch.groups {
		starts[first[g]] = max(starts[first[g]], before[g])
		ends[last[g]] = max(ends[last[g]], after[g])
	}
	b0, b1 := make([]float32, layers), make([]float32, layers)
	for L := range layers {
		if L > 0 {
			b0[L] = b1[L-1] + z.band + ends[L-1] + starts[L]
		}
		b1[L] = b0[L] + depth[L]
	}
	for u := range vs {
		vs[u].v = (b0[vs[u].layer] + b1[vs[u].layer]) / 2
	}
	if !vertical {
		for g := range ch.groups {
			title(&ch.groups[g], b1[last[g]]+after[g]-b0[first[g]]+before[g]-2*z.inset)
		}
	}

	// Across the ranks: the vertices of each rank in order, apart by
	// their sizes and gaps, first pulled toward those they link to and
	// the edges of a box toward the box's other edges in the ranks it
	// spans, and then pushed apart where they still overlap, the edges of
	// a box as one.
	sep := func(a, b int) float32 {
		va, vb := vs[a], vs[b]
		var gap float32
		switch {
		case va.kind == isLeft:
			gap = z.inset
			if !vertical {
				gap += ch.groups[va.group].band
			}
		case vb.kind == isRight:
			gap = z.inset
		case va.kind == isRight || vb.kind == isLeft:
			gap = z.gap / 2
		case va.kind == isNode && vb.kind == isNode:
			gap = z.gap
		case va.kind == isNode || vb.kind == isNode:
			gap = z.gap / 2
		default:
			gap = z.lane
		}
		return va.hi + vb.lo + gap
	}
	// boxes returns how far the box of each subgraph wants to reach before
	// and after across the ranks: as far as what is in it does, and its
	// edge and title.
	boxes := func() ([]float32, []float32) {
		lo, hi := make([]float32, ng), make([]float32, ng)
		for g := range lo {
			lo[g], hi[g] = float32(math.Inf(1)), float32(math.Inf(-1))
		}
		for _, v := range vs {
			for g := v.scope; g >= 0; g = ch.groups[g].parent {
				lo[g], hi[g] = min(lo[g], v.u-v.lo), max(hi[g], v.u+v.hi)
			}
		}
		for g := range lo {
			lo[g] -= z.inset
			hi[g] += z.inset
			if !vertical {
				lo[g] -= ch.groups[g].band
			}
		}
		return lo, hi
	}
	// level sets the vertices of col as near where they want to be as they
	// can be in their order, apart by sep, by the pool adjacent violators
	// algorithm: where a vertex wants to be less how far it must be after
	// the first must not fall along the column.
	level := func(col []int, down, up bool) {
		off := make([]float32, len(col))
		for i := 1; i < len(col); i++ {
			off[i] = off[i-1] + sep(col[i-1], col[i])
		}
		type pool struct {
			sum, n float32
			first  int
		}
		var ps []pool
		blo, bhi := boxes()
		for i, u := range col {
			want := vs[u].u
			switch vs[u].kind {
			case isLeft:
				want = blo[vs[u].group]
			case isRight:
				want = bhi[vs[u].group]
			default:
				var sum float32
				k := 0
				if down {
					for _, w := range ups[u] {
						sum, k = sum+vs[w].u, k+1
					}
				}
				if up {
					for _, w := range downs[u] {
						sum, k = sum+vs[w].u, k+1
					}
				}
				if k > 0 {
					want = sum / float32(k)
				}
			}
			ps = append(ps, pool{want - off[i], 1, i})
			for len(ps) > 1 && ps[len(ps)-2].sum/ps[len(ps)-2].n > ps[len(ps)-1].sum/ps[len(ps)-1].n {
				last := ps[len(ps)-1]
				ps = ps[:len(ps)-1]
				ps[len(ps)-1].sum += last.sum
				ps[len(ps)-1].n += last.n
			}
		}
		for k, p := range ps {
			end := len(col)
			if k+1 < len(ps) {
				end = ps[k+1].first
			}
			for i := p.first; i < end; i++ {
				vs[col[i]].u = p.sum/p.n + off[i]
			}
		}
	}
	for _, col := range cols {
		level(col, false, false)
	}
	for range 10 {
		for L := 1; L < layers; L++ {
			level(cols[L], true, false)
		}
		for L := layers - 2; L >= 0; L-- {
			level(cols[L], false, true)
		}
	}
	for range 4 {
		for _, col := range cols {
			level(col, true, true)
		}
	}
	one := make([]int, len(vs))
	for u, v := range vs {
		one[u] = u
		switch v.kind {
		case isLeft:
			one[u] = lefts[v.group][0]
		case isRight:
			one[u] = rights[v.group][0]
		}
	}
	at := make([]float32, len(vs))
	for u, v := range vs {
		at[u] = v.u
	}
	for g := range ch.groups {
		for _, u := range lefts[g] {
			at[lefts[g][0]] = min(at[lefts[g][0]], vs[u].u)
		}
		for _, u := range rights[g] {
			at[rights[g][0]] = max(at[rights[g][0]], vs[u].u)
		}
	}
	type need struct {
		to int
		d  float32
	}
	next, deg := make([][]need, len(vs)), make([]int, len(vs))
	add := func(a, b int, d float32) {
		next[a] = append(next[a], need{b, d})
		deg[b]++
	}
	for _, col := range cols {
		for i := 1; i < len(col); i++ {
			add(one[col[i-1]], one[col[i]], sep(col[i-1], col[i]))
		}
	}
	if vertical {
		for g, gr := range ch.groups {
			add(lefts[g][0], rights[g][0], gr.tw+2*z.inset)
		}
	}
	var queue []int
	for u := range vs {
		if one[u] == u && deg[u] == 0 {
			queue = append(queue, u)
		}
	}
	for len(queue) > 0 {
		a := queue[0]
		queue = queue[1:]
		for _, nd := range next[a] {
			at[nd.to] = max(at[nd.to], at[a]+nd.d)
			if deg[nd.to]--; deg[nd.to] == 0 {
				queue = append(queue, nd.to)
			}
		}
	}
	for u := range vs {
		vs[u].u = at[one[u]]
	}

	// Each link leaves the side of its node facing the next rank, the
	// links of a side spread across it in the order they go, and runs
	// through its vertices, straight across the bands of the ranks and
	// curving in the gaps between.
	polys := make([][][2]float32, n)
	for u, nd := range ch.nodes {
		for _, p := range outline(nd, z) {
			polys[u] = append(polys[u], canon(p))
		}
	}
	type port struct {
		route  int
		toward float32
	}
	sides := make([][2][]port, n)
	for k, r := range routes {
		if ch.edges[r.edge].stroke == hidden {
			continue
		}
		p := r.path
		sides[r.a][1] = append(sides[r.a][1], port{k, vs[p[1]].u})
		sides[r.b][0] = append(sides[r.b][0], port{k, vs[p[len(p)-2]].u})
	}
	offs := make([][2]float32, len(routes))
	for u := range n {
		for j, ports := range sides[u] {
			slices.SortStableFunc(ports, func(a, b port) int { return cmp.Compare(a.toward, b.toward) })
			step := min(z.port, max(0, 2*vs[u].lo-2*z.radius)/float32(max(len(ports)-1, 1)))
			for i, p := range ports {
				offs[p.route][1-j] = (float32(i) - float32(len(ports)-1)/2) * step
			}
		}
	}
	for k, r := range routes {
		e := &ch.edges[r.edge]
		if e.stroke == hidden {
			continue
		}
		p := r.path
		a, b := vs[p[0]], vs[p[len(p)-1]]
		ua, ub := a.u+offs[k][0], b.u+offs[k][1]
		_, ha := reach(polys[r.a], offs[k][0])
		lb, _ := reach(polys[r.b], offs[k][1])
		// The link runs straight from a node to the edge of its band, and
		// on as a B-spline through the vertices it passes, through the
		// center of its label.
		pts := [][2]float32{{ua, a.v + ha}}
		if b1[a.layer] > a.v+ha {
			pts = append(pts, [2]float32{ua, b1[a.layer]})
		}
		for _, d := range p[1 : len(p)-1] {
			v := vs[d]
			if d == r.label {
				pts = append(pts, [2]float32{v.u, b0[v.layer]}, [2]float32{v.u, v.v}, [2]float32{v.u, b1[v.layer]})
			} else {
				pts = append(pts, [2]float32{v.u, v.v})
			}
		}
		if b0[b.layer] < b.v+lb {
			pts = append(pts, [2]float32{ub, b0[b.layer]})
		}
		pts = append(pts, [2]float32{ub, b.v + lb})
		for i := range pts {
			pts[i] = final(pts[i])
		}
		e.pts = flatten(basis(pts))
		if r.rev {
			e.pts = reversed(e.pts)
		}
		if r.label >= 0 {
			v := vs[r.label]
			c := final([2]float32{v.u, v.v})
			e.lx, e.ly, e.scope = c[0], c[1], v.scope
		}
	}

	for u := range ch.nodes {
		c := final([2]float32{vs[u].u, vs[u].v})
		ch.nodes[u].x, ch.nodes[u].y = c[0], c[1]
	}
	for g := range ch.groups {
		gr := &ch.groups[g]
		p0 := final([2]float32{vs[lefts[g][0]].u, b0[first[g]] - before[g]})
		p1 := final([2]float32{vs[rights[g][0]].u, b1[last[g]] + after[g]})
		gr.x0, gr.y0, gr.x1, gr.y1 = min(p0[0], p1[0]), min(p0[1], p1[1]), max(p0[0], p1[0]), max(p0[1], p1[1])
	}
	for k := range ch.edges {
		e := &ch.edges[k]
		if e.from == e.to {
			ch.loop(e, k == ch.nodes[e.from].loops[0])
			continue
		}
		for j, g := range e.clip {
			if g >= 0 && len(e.pts) > 1 {
				gr := ch.groups[g]
				e.pts = clip(e.pts, ui.Rect{X: gr.x0, Y: gr.y0, W: gr.x1 - gr.x0, H: gr.y1 - gr.y0}, j == 1)
			}
		}
	}

	x0, y0 := float32(math.Inf(1)), float32(math.Inf(1))
	x1, y1 := float32(math.Inf(-1)), float32(math.Inf(-1))
	grow := func(x, y, w, h float32) {
		x0, y0, x1, y1 = min(x0, x-w), min(y0, y-h), max(x1, x+w), max(y1, y+h)
	}
	for _, nd := range ch.nodes {
		grow(nd.x, nd.y, nd.w/2, nd.h/2)
	}
	for _, gr := range ch.groups {
		grow(gr.x0, gr.y0, 0, 0)
		grow(gr.x1, gr.y1, 0, 0)
	}
	for _, e := range ch.edges {
		if len(e.lines) > 0 {
			grow(e.lx, e.ly, e.lw/2, e.lh/2)
		}
		for _, p := range e.pts {
			grow(p[0], p[1], 0, 0)
		}
	}
	dx, dy := z.pad-x0, z.pad-y0
	for u := range ch.nodes {
		ch.nodes[u].x += dx
		ch.nodes[u].y += dy
	}
	for g := range ch.groups {
		gr := &ch.groups[g]
		gr.x0, gr.x1, gr.y0, gr.y1 = gr.x0+dx, gr.x1+dx, gr.y0+dy, gr.y1+dy
	}
	for k := range ch.edges {
		e := &ch.edges[k]
		e.lx, e.ly = e.lx+dx, e.ly+dy
		for i := range e.pts {
			e.pts[i][0] += dx
			e.pts[i][1] += dy
		}
	}
	ch.w, ch.h = x1-x0+2*z.pad, y1-y0+2*z.pad
}

// loop lays out e, a link of a node back to itself, as a loop on the side
// of the node across the ranks, and its label beside it when labeled.
func (ch *chart) loop(e *edge, labeled bool) {
	z := ch.z
	n := ch.nodes[e.from]
	poly := outline(n, z)
	var a, b, c1, c2 [2]float32
	if ch.dir == down || ch.dir == up {
		flipped := make([][2]float32, len(poly))
		for i, p := range poly {
			flipped[i] = [2]float32{p[1], p[0]}
		}
		_, ra := reach(flipped, -n.h/4)
		_, rb := reach(flipped, n.h/4)
		a, b = [2]float32{n.x + ra, n.y - n.h/4}, [2]float32{n.x + rb, n.y + n.h/4}
		c1, c2 = [2]float32{a[0] + z.loop*4/3, a[1] - z.loop/3}, [2]float32{b[0] + z.loop*4/3, b[1] + z.loop/3}
		e.lx, e.ly = max(a[0], b[0])+z.loop+z.lane/2+e.lw/2, n.y
	} else {
		_, ra := reach(poly, -n.w/4)
		_, rb := reach(poly, n.w/4)
		a, b = [2]float32{n.x - n.w/4, n.y + ra}, [2]float32{n.x + n.w/4, n.y + rb}
		c1, c2 = [2]float32{a[0] - z.loop/3, a[1] + z.loop*4/3}, [2]float32{b[0] + z.loop/3, b[1] + z.loop*4/3}
		e.lx, e.ly = n.x, max(a[1], b[1])+z.loop+z.lane/2+e.lh/2
	}
	e.pts, e.scope = flatten([]seg{{a, c1, c2, b}}), n.group
	if !labeled {
		e.lines = nil
	}
}

// seg is a cubic Bézier curve: its ends, and its control points between.
type seg [4][2]float32

// basis returns the curve near pts as d3's curveBasis draws it: a
// uniform cubic B-spline from the first point to the last, which passes
// through the middle of three in a line.
func basis(pts [][2]float32) []seg {
	if len(pts) < 3 {
		return []seg{straight(pts[0], pts[len(pts)-1])}
	}
	pts = append(pts, pts[len(pts)-1])
	at := lerp(pts[0], pts[1], 1.0/6)
	out := []seg{straight(pts[0], at)}
	for i := 2; i < len(pts); i++ {
		a, b, c := pts[i-2], pts[i-1], pts[i]
		end := [2]float32{(a[0] + 4*b[0] + c[0]) / 6, (a[1] + 4*b[1] + c[1]) / 6}
		out = append(out, seg{at, lerp(a, b, 1.0/3), lerp(a, b, 2.0/3), end})
		at = end
	}
	return append(out, straight(at, pts[len(pts)-1]))
}

// straight returns the straight curve from a to b.
func straight(a, b [2]float32) seg { return seg{a, lerp(a, b, 1.0/3), lerp(a, b, 2.0/3), b} }

// flatten returns points along the curve sg, more where it is longer.
func flatten(sg []seg) [][2]float32 {
	pts := [][2]float32{sg[0][0]}
	for _, s := range sg {
		l := dist(s[0], s[1]) + dist(s[1], s[2]) + dist(s[2], s[3])
		steps := int(min(16, max(1, l/6)))
		for k := 1; k <= steps; k++ {
			u := float32(k) / float32(steps)
			v := 1 - u
			a, b, c, d := v*v*v, 3*v*v*u, 3*v*u*u, u*u*u
			pts = append(pts, [2]float32{a*s[0][0] + b*s[1][0] + c*s[2][0] + d*s[3][0], a*s[0][1] + b*s[1][1] + c*s[2][1] + d*s[3][1]})
		}
	}
	return pts
}

// clip returns pts cut where it comes into r, at its end, or where it
// last leaves r, at its start.
func clip(pts [][2]float32, r ui.Rect, atEnd bool) [][2]float32 {
	inside := func(p [2]float32) bool { return p[0] > r.X && p[0] < r.X+r.W && p[1] > r.Y && p[1] < r.Y+r.H }
	cross := func(a, b [2]float32) [2]float32 {
		// a is outside and b inside: halve the way between until it is on
		// the edge.
		for range 20 {
			if m := lerp(a, b, 0.5); inside(m) {
				b = m
			} else {
				a = m
			}
		}
		return a
	}
	if atEnd {
		for i := 1; i < len(pts); i++ {
			if inside(pts[i]) {
				return append(pts[:i:i], cross(pts[i-1], pts[i]))
			}
		}
		return pts
	}
	for i := len(pts) - 2; i >= 0; i-- {
		if inside(pts[i]) {
			return append([][2]float32{cross(pts[i+1], pts[i])}, pts[i+1:]...)
		}
	}
	return pts
}
