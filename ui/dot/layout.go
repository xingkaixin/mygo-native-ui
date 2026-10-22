package dot

import (
	"cmp"
	"math"
	"slices"
	"strings"

	"github.com/egoist/mygo/ui"
)

// sizes are the sizes of the parts of a graph, from the theme.
type sizes struct {
	th, sh, padX, padY, minW, minH, radius, ring, point, fold, head, gap, lane, band, port, inset, loop, pad float32
}

func measure(c *ui.Context) sizes {
	t := c.Theme()
	_, th := c.MeasureText(0, ui.Span{Text: "Ag", Size: t.FontSize})
	_, sh := c.MeasureText(0, ui.Span{Text: "Ag", Size: t.FontSize - 1})
	return sizes{th: th, sh: sh, padX: t.Space(3), padY: t.Space(1.5), minW: t.Space(12), minH: th + t.Space(4),
		radius: t.Space(2), ring: t.Space(1), point: t.Space(2.5), fold: t.Space(2.5), head: t.Space(2.5),
		gap: t.Space(6), lane: t.Space(2), band: t.Space(5), port: t.Space(2.5), inset: t.Space(3),
		loop: t.Space(6), pad: t.Space(5)}
}

// rim returns how tall the ellipse at either end of a cylinder w wide is,
// half way across.
func rim(w float32, z sizes) float32 { return max(z.fold*0.6, min(z.fold, w/10)) }

// fit returns the size of node n around its text.
func fit(n *node, z sizes) (float32, float32) {
	tw, tb := n.tw, n.tb
	switch n.shape {
	case point:
		return z.point, z.point
	case plain:
		return tw, tb
	case plaintext:
		return tw + 2*z.padX, tb + 2*z.padY
	case ellipse:
		return max((tw+2*z.padY)*math.Sqrt2, z.minW), max((tb+z.padY)*math.Sqrt2, z.minH)
	case circle, doublecircle:
		d := max(float32(math.Hypot(float64(tw), float64(tb)))+2*z.padY, z.minH)
		if n.shape == doublecircle {
			d += 2 * z.ring
		}
		return d, d
	case diamond:
		// A diamond k times as wide as it is tall holds a box bw by bh
		// when it is bw + k·bh wide.
		bw, bh := tw+2*z.padY, tb+z.padY
		k := max(1, min(2, bw/bh))
		w := max(bw+k*bh, z.minW)
		return w, max(w/k, z.minH)
	case cylinder:
		w := max(tw+2*z.padX, z.minW)
		return w, max(tb+2*z.padY, z.minH) + 2*rim(w, z)
	case note:
		return max(tw+2*z.padX+z.fold/2, z.minW), max(tb+2*z.padY, z.minH)
	}
	return max(tw+2*z.padX, z.minW), max(tb+2*z.padY, z.minH)
}

// outline returns the outline of n about its center, round shapes as many
// points along their curves.
func outline(n node, z sizes) [][2]float32 {
	w, h := n.w/2, n.h/2
	arc := func(cx, cy, rx, ry float32, a0, a1 float64, steps int, out [][2]float32) [][2]float32 {
		for k := 0; k <= steps; k++ {
			a := a0 + (a1-a0)*float64(k)/float64(steps)
			out = append(out, [2]float32{cx + rx*float32(math.Cos(a)), cy + ry*float32(math.Sin(a))})
		}
		return out
	}
	switch n.shape {
	case ellipse:
		return arc(0, 0, w, h, 0, 2*math.Pi, 48, nil)[:48]
	case circle, doublecircle, point:
		return arc(0, 0, w, w, 0, 2*math.Pi, 40, nil)[:40]
	case diamond:
		return [][2]float32{{0, -h}, {w, 0}, {0, h}, {-w, 0}}
	case note:
		f := min(z.fold, h)
		return [][2]float32{{-w, -h}, {w - f, -h}, {w, -h + f}, {w, h}, {-w, h}}
	case cylinder:
		ry := rim(n.w, z)
		out := arc(0, -h+ry, w, ry, math.Pi, 2*math.Pi, 16, nil)
		return arc(0, h-ry, w, ry, 0, math.Pi, 16, out)
	case box:
		if n.style&rounded != 0 {
			r := min(z.radius, h/2, w/2)
			out := arc(w-r, -h+r, r, r, -math.Pi/2, 0, 6, nil)
			out = arc(w-r, h-r, r, r, 0, math.Pi/2, 6, out)
			out = arc(-w+r, h-r, r, r, math.Pi/2, math.Pi, 6, out)
			return arc(-w+r, -h+r, r, r, math.Pi, 3*math.Pi/2, 6, out)
		}
	}
	return [][2]float32{{-w, -h}, {w, -h}, {w, h}, {-w, h}}
}

// reach returns where the outline poly meets the line across it at u,
// first and last along it.
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

// lines returns the lines of s and the width of the widest.
func lines(c *ui.Context, s string, size float32, weight int) ([]line, float32) {
	var out []line
	var widest float32
	for _, l := range strings.Split(s, "\n") {
		w, _ := c.MeasureText(0, ui.Span{Text: l, Size: size, Weight: weight})
		out = append(out, line{l, w})
		widest = max(widest, w)
	}
	return out, widest
}

// layout sizes the nodes, the labels and the clusters' titles, and places
// them.
func layout(c *ui.Context, g *graph) {
	t := c.Theme()
	z := measure(c)
	g.z = z
	for i := range g.nodes {
		n := &g.nodes[i]
		if n.shape != point {
			n.lines, n.tw = lines(c, n.text, t.FontSize, 400)
		}
		n.tb = float32(len(n.lines)) * z.th
		n.w, n.h = fit(n, z)
		// The edges of a node back to itself share a loop and a label.
		var texts []string
		for _, k := range n.loops {
			if s := g.edges[k].text; s != "" && g.edges[k].style&invis == 0 {
				texts = append(texts, s)
			}
			g.edges[k].text = ""
		}
		if len(n.loops) > 0 {
			g.edges[n.loops[0]].text = strings.Join(texts, ", ")
		}
	}
	for i := range g.edges {
		if e := &g.edges[i]; e.text != "" {
			var w float32
			e.lines, w = lines(c, e.text, t.FontSize-1, 400)
			e.lw, e.lh = w+2*t.Space(1), float32(len(e.lines))*z.sh+t.Space(1)
		}
	}
	for k := range g.clusters {
		if cl := &g.clusters[k]; cl.text != "" {
			cl.lines, cl.tw = lines(c, cl.text, t.FontSize-1, 600)
			cl.band = float32(len(cl.lines))*z.sh + t.Space(3)
		}
	}
	place(g, z)
}

const (
	isNode uint8 = iota
	isDummy
	isLeft
	isRight
)

// vert is a vertex of the layered graph: a node, a point an edge passes in
// a layer between its ends, the label of an edge, or an edge of a
// cluster's box in a layer it spans. lo and hi are how far it reaches
// before and after its center u across the layers, size how far along
// them; at is its place in its layer, scope the cluster it is in.
type vert struct {
	kind                 uint8
	node, cluster, scope int
	layer                int
	lo, hi, size, u, v   float32
	at                   float64
}

// route is an edge between ranks as laid out: its ends in rank order,
// whether that is against the edge, the vertices it passes, and that of
// its label, -1 for none.
type route struct {
	edge, a, b int
	rev        bool
	path       []int
	label      int
}

// rank turns around the routes that close a cycle and returns the rank of
// each node, after the nodes that link to it; find gives the node that
// stands for the nodes kept in one rank with it.
func rank(n int, routes []route, find func(int) int) []int {
	out := make([][]int, n)
	in := make([]int, n)
	for k, r := range routes {
		out[find(r.a)] = append(out[find(r.a)], k)
		in[find(r.b)]++
	}
	mark := make([]uint8, n)
	var walk func(u int)
	walk = func(u int) {
		mark[u] = 1
		for _, k := range out[u] {
			switch v := find(routes[k].b); mark[v] {
			case 0:
				walk(v)
			case 1:
				routes[k].rev = true
			}
		}
		mark[u] = 2
	}
	for _, sources := range []bool{true, false} {
		for u := range n {
			if find(u) == u && mark[u] == 0 && (!sources || in[u] == 0) {
				walk(u)
			}
		}
	}
	for k := range routes {
		if r := &routes[k]; r.rev {
			r.a, r.b = r.b, r.a
		}
	}
	rk := make([]int, n)
	for range n {
		moved := false
		for _, r := range routes {
			if a, b := find(r.a), find(r.b); rk[a]+1 > rk[b] {
				rk[b], moved = rk[a]+1, true
			}
		}
		if !moved {
			break
		}
	}
	// A node with more edges out than in moves down as far as they let it,
	// as one that starts a chain does to just before the node it links to,
	// and one with more in than out up.
	for range 4 {
		for u := range n {
			if find(u) != u {
				continue
			}
			lo, hi, ins, outs := 0, math.MaxInt, 0, 0
			for _, r := range routes {
				if find(r.b) == u {
					lo, ins = max(lo, rk[find(r.a)]+1), ins+1
				}
				if find(r.a) == u {
					hi, outs = min(hi, rk[find(r.b)]-1), outs+1
				}
			}
			switch {
			case outs > ins:
				rk[u] = hi
			case ins > outs:
				rk[u] = lo
			}
		}
	}
	low := math.MaxInt
	for u := range n {
		low = min(low, rk[find(u)])
	}
	ranks := make([]int, n)
	for u := range n {
		ranks[u] = rk[find(u)] - low
	}
	return ranks
}

// place lays out the nodes, the edges and the clusters as Graphviz's dot
// does, after Sugiyama: it ranks the nodes, gives each edge a vertex in
// each layer it passes and its label one in a layer between ranks, orders
// each layer so that few edges cross, with the nodes of a cluster
// together, sets each vertex across as near those it links to as the ones
// beside it let it, and runs edges as curves through their vertices. It
// lays out down the ranks, and turns the graph the way rankdir says as it
// places it.
func place(g *graph, z sizes) {
	n, nc := len(g.nodes), len(g.clusters)
	g.w, g.h = 0, 0
	if n == 0 {
		return
	}
	vertical := g.dir == tb || g.dir == bt
	across := func(w, h float32) (float32, float32) {
		if vertical {
			return w, h
		}
		return h, w
	}
	// canon turns a point about a node's center from the graph's way to
	// the layout's, and final a point of the layout to the graph's.
	canon := func(p [2]float32) [2]float32 {
		switch g.dir {
		case bt:
			return [2]float32{p[0], -p[1]}
		case lr:
			return [2]float32{p[1], p[0]}
		case rl:
			return [2]float32{p[1], -p[0]}
		}
		return p
	}
	final := func(p [2]float32) [2]float32 {
		switch g.dir {
		case bt:
			return [2]float32{p[0], -p[1]}
		case lr:
			return [2]float32{p[1], p[0]}
		case rl:
			return [2]float32{-p[1], p[0]}
		}
		return p
	}

	root := make([]int, n)
	for u := range root {
		root[u] = u
	}
	var find func(u int) int
	find = func(u int) int {
		if root[u] != u {
			root[u] = find(root[u])
		}
		return root[u]
	}
	for _, set := range g.sames {
		for _, u := range set[1:] {
			root[find(u)] = find(set[0])
		}
	}
	var routes []route
	var flats []int
	for k, e := range g.edges {
		g.edges[k].scope = g.common(g.nodes[e.from].cluster, g.nodes[e.to].cluster)
		switch {
		case e.from == e.to:
		case find(e.from) == find(e.to):
			flats = append(flats, k)
		default:
			routes = append(routes, route{edge: k, a: e.from, b: e.to, label: -1})
		}
	}
	ranks := rank(n, routes, find)

	// Nodes are in the even layers, and labels in the odd ones between.
	first, last := make([]int, nc), make([]int, nc)
	for k := range first {
		first[k], last[k] = math.MaxInt, -1
	}
	for u, nd := range g.nodes {
		for k := nd.cluster; k >= 0; k = g.clusters[k].parent {
			first[k], last[k] = min(first[k], 2*ranks[u]), max(last[k], 2*ranks[u])
		}
	}
	var vs []vert
	for u, nd := range g.nodes {
		cw, mw := across(nd.w, nd.h)
		v := vert{kind: isNode, node: u, cluster: -1, scope: nd.cluster, layer: 2 * ranks[u], lo: cw / 2, hi: cw / 2, size: mw}
		if len(nd.loops) > 0 {
			v.hi += z.loop
			if e := g.edges[nd.loops[0]]; len(e.lines) > 0 {
				lw, lh := across(e.lw, e.lh)
				v.hi += lw + z.lane
				v.size = max(v.size, lh)
			}
		}
		vs = append(vs, v)
	}
	for k := range routes {
		r := &routes[k]
		e := g.edges[r.edge]
		la, lb := vs[r.a].layer, vs[r.b].layer
		mid := (la + lb) / 2
		if mid%2 == 0 {
			mid--
		}
		r.path = []int{r.a}
		for L := la + 1; L < lb; L++ {
			d := vert{kind: isDummy, node: -1, cluster: -1, scope: e.scope, layer: L}
			if L == mid && len(e.lines) > 0 && e.style&invis == 0 {
				// The label is beside the edge, after it across the layers.
				cw, mw := across(e.lw, e.lh)
				d.hi, d.size = z.lane+cw, mw
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
	lefts, rights := make([][]int, nc), make([][]int, nc)
	for k, cl := range g.clusters {
		for L := first[k]; L <= last[k]; L++ {
			lefts[k] = append(lefts[k], len(vs))
			vs = append(vs, vert{kind: isLeft, node: -1, cluster: k, scope: cl.parent, layer: L})
			rights[k] = append(rights[k], len(vs))
			vs = append(vs, vert{kind: isRight, node: -1, cluster: k, scope: cl.parent, layer: L})
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

	// Each layer is ordered as a tree: the vertices of a cluster come
	// together between its box's edges, and clusters beside each other
	// keep one order in every layer, so that their boxes are rectangles
	// that do not overlap.
	kids := make([][]int, nc+1)
	for k, cl := range g.clusters {
		if first[k] <= last[k] {
			kids[cl.parent+1] = append(kids[cl.parent+1], k)
		}
	}
	order := make([]float64, nc)
	for k := range order {
		order[k] = float64(k)
	}
	key := make([]float64, len(vs))
	cols := make([][]int, layers)
	var arrange func(L, scope int, out []int) []int
	arrange = func(L, scope int, out []int) []int {
		type item struct {
			u, k     int
			key, cur float64
		}
		var items []item
		for _, u := range members[L] {
			if vs[u].scope == scope {
				items = append(items, item{u, -1, key[u], vs[u].at})
			}
		}
		for _, k := range kids[scope+1] {
			if L < first[k] || L > last[k] {
				continue
			}
			sum, cnt := 0.0, 0
			for _, u := range members[L] {
				if g.within(vs[u].scope, k) {
					sum, cnt = sum+key[u], cnt+1
				}
			}
			cur := vs[lefts[k][L-first[k]]].at
			at := cur
			if cnt > 0 {
				at = sum / float64(cnt)
			}
			items = append(items, item{-1, k, at, cur})
		}
		slices.SortStableFunc(items, func(a, b item) int { return cmp.Or(cmp.Compare(a.key, b.key), cmp.Compare(a.cur, b.cur)) })
		var slots, ks []int
		for i, it := range items {
			if it.k >= 0 {
				slots, ks = append(slots, i), append(ks, it.k)
			}
		}
		slices.SortStableFunc(ks, func(a, b int) int { return cmp.Compare(order[a], order[b]) })
		for j, i := range slots {
			items[i].k = ks[j]
		}
		for _, it := range items {
			if it.k < 0 {
				out = append(out, it.u)
				continue
			}
			out = append(out, lefts[it.k][L-first[it.k]])
			out = arrange(L, it.k, out)
			out = append(out, rights[it.k][L-first[it.k]])
		}
		return out
	}
	rearrange := func(L int) {
		cols[L] = arrange(L, -1, cols[L][:0])
		for i, u := range cols[L] {
			vs[u].at = float64(i)
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
		at := make([]float64, len(vs))
		for u := range vs {
			at[u] = vs[u].at
		}
		return at
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
		sum, cnt := make([]float64, nc), make([]int, nc)
		for _, v := range vs {
			if v.kind <= isDummy {
				for k := v.scope; k >= 0; k = g.clusters[k].parent {
					sum[k], cnt[k] = sum[k]+v.at, cnt[k]+1
				}
			}
		}
		for k := range order {
			order[k] = sum[k] / float64(max(cnt[k], 1))
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
	// apart by a gap, and more where a cluster's box starts or ends, for
	// its edge, its title and those of the clusters in it that start or
	// end there too.
	depth := make([]float32, layers)
	for _, v := range vs {
		depth[v.layer] = max(depth[v.layer], v.size)
	}
	before, after := make([]float32, nc), make([]float32, nc)
	b0, b1 := make([]float32, layers), make([]float32, layers)
	spread := func() {
		starts, ends := make([]float32, layers), make([]float32, layers)
		for k := range nc {
			if first[k] <= last[k] {
				starts[first[k]] = max(starts[first[k]], before[k])
				ends[last[k]] = max(ends[last[k]], after[k])
			}
		}
		for L := range layers {
			b0[L] = 0
			if L > 0 {
				b0[L] = b1[L-1] + z.band + ends[L-1] + starts[L]
			}
			b1[L] = b0[L] + depth[L]
		}
	}
	// A cluster's index is after that of the cluster around it, so the
	// clusters inside come first.
	for k := nc - 1; k >= 0; k-- {
		if first[k] > last[k] {
			continue
		}
		var b, a float32
		for _, j := range kids[k+1] {
			if first[j] == first[k] {
				b = max(b, before[j])
			}
			if last[j] == last[k] {
				a = max(a, after[j])
			}
		}
		before[k], after[k] = z.inset+b, z.inset+a
		switch g.dir {
		case tb:
			before[k] += g.clusters[k].band
		case bt:
			after[k] += g.clusters[k].band
		default:
			// The title runs along the ranks: the box is as long as it.
			spread()
			if short := g.clusters[k].tw + 2*z.inset - (b1[last[k]] + after[k] - b0[first[k]] + before[k]); short > 0 {
				before[k] += short / 2
				after[k] += short / 2
			}
		}
	}
	spread()
	for u := range vs {
		vs[u].v = (b0[vs[u].layer] + b1[vs[u].layer]) / 2
	}

	// Across the ranks: the vertices of each layer in order, apart by
	// their sizes and gaps, first pulled toward those they link to and
	// the edges of a box toward the box's other edges in the layers it
	// spans, and then pushed apart where they still overlap, the edges of
	// a box as one.
	flatGap := map[[2]int]float32{}
	for _, k := range flats {
		if e := g.edges[k]; len(e.lines) > 0 && e.style&invis == 0 {
			cw, _ := across(e.lw, e.lh)
			pair := [2]int{min(e.from, e.to), max(e.from, e.to)}
			flatGap[pair] = max(flatGap[pair], cw+2*z.lane)
		}
	}
	sep := func(a, b int) float32 {
		va, vb := vs[a], vs[b]
		var gap float32
		switch {
		case va.kind == isLeft:
			gap = z.inset
			if !vertical {
				gap += g.clusters[va.cluster].band
			}
		case vb.kind == isRight:
			gap = z.inset
		case va.kind == isRight || vb.kind == isLeft:
			gap = z.gap / 2
		case va.kind == isNode && vb.kind == isNode:
			gap = max(z.gap, flatGap[[2]int{min(a, b), max(a, b)}])
		case va.kind == isNode || vb.kind == isNode:
			gap = z.gap / 2
		default:
			gap = z.lane
		}
		return va.hi + vb.lo + gap
	}
	// boxes returns how far the box of each cluster wants to reach before
	// and after across the layers: as far as what is in it does, and its
	// edge and title.
	boxes := func() ([]float32, []float32) {
		lo, hi := make([]float32, nc), make([]float32, nc)
		for k := range lo {
			lo[k], hi[k] = float32(math.Inf(1)), float32(math.Inf(-1))
		}
		for _, v := range vs {
			for k := v.scope; k >= 0; k = g.clusters[k].parent {
				lo[k], hi[k] = min(lo[k], v.u-v.lo), max(hi[k], v.u+v.hi)
			}
		}
		for k := range lo {
			lo[k] -= z.inset
			hi[k] += z.inset
			if !vertical {
				lo[k] -= g.clusters[k].band
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
				want = blo[vs[u].cluster]
			case isRight:
				want = bhi[vs[u].cluster]
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
				top := ps[len(ps)-1]
				ps = ps[:len(ps)-1]
				ps[len(ps)-1].sum += top.sum
				ps[len(ps)-1].n += top.n
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
	at := make([]float32, len(vs))
	for u, v := range vs {
		one[u], at[u] = u, v.u
		switch v.kind {
		case isLeft:
			one[u] = lefts[v.cluster][0]
		case isRight:
			one[u] = rights[v.cluster][0]
		}
	}
	for k := range g.clusters {
		for _, u := range lefts[k] {
			at[lefts[k][0]] = min(at[lefts[k][0]], vs[u].u)
		}
		for _, u := range rights[k] {
			at[rights[k][0]] = max(at[rights[k][0]], vs[u].u)
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
		for k, cl := range g.clusters {
			if len(lefts[k]) > 0 {
				add(lefts[k][0], rights[k][0], cl.tw+2*z.inset)
			}
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

	// Each edge leaves the side of its node facing the next rank, the
	// edges of a side spread across it in the order they go, and runs
	// through its vertices, straight across the bands of the layers and
	// curving in the gaps between.
	polys := make([][][2]float32, n)
	for u, nd := range g.nodes {
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
		p := r.path
		sides[r.a][1] = append(sides[r.a][1], port{k, vs[p[1]].u})
		sides[r.b][0] = append(sides[r.b][0], port{k, vs[p[len(p)-2]].u})
	}
	offs := make([][2]float32, len(routes))
	for u, nd := range g.nodes {
		cw, _ := across(nd.w, nd.h)
		for j, ports := range sides[u] {
			slices.SortStableFunc(ports, func(a, b port) int { return cmp.Compare(a.toward, b.toward) })
			step := min(z.port, cw*0.6/float32(max(len(ports)-1, 1)))
			for i, p := range ports {
				offs[p.route][1-j] = (float32(i) - float32(len(ports)-1)/2) * step
			}
		}
	}
	for k, r := range routes {
		e := &g.edges[r.edge]
		p := r.path
		a, b := vs[p[0]], vs[p[len(p)-1]]
		ua, ub := a.u+offs[k][0], b.u+offs[k][1]
		_, ha := reach(polys[r.a], offs[k][0])
		lb, _ := reach(polys[r.b], offs[k][1])
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
			slices.Reverse(e.pts)
		}
		if r.label >= 0 {
			v := vs[r.label]
			cw, _ := across(e.lw, e.lh)
			c := final([2]float32{v.u + z.lane + cw/2, v.v})
			e.lx, e.ly = c[0], c[1]
		}
	}

	// An edge between nodes of one rank runs straight between them when
	// they are side by side, and arcs over the rank when others are
	// between, its label over it.
	for _, k := range flats {
		e := &g.edges[k]
		a, b := e.from, e.to
		if vs[a].u > vs[b].u {
			a, b = b, a
		}
		va, vb := vs[a], vs[b]
		_, mw := across(e.lw, e.lh)
		var sg seg
		var top [2]float32
		if math.Abs(vb.at-va.at) == 1 {
			flip := func(poly [][2]float32) [][2]float32 {
				out := make([][2]float32, len(poly))
				for i, p := range poly {
					out[i] = [2]float32{p[1], p[0]}
				}
				return out
			}
			_, ra := reach(flip(polys[a]), 0)
			lb, _ := reach(flip(polys[b]), 0)
			sg = straight([2]float32{va.u + ra, va.v}, [2]float32{vb.u + lb, vb.v})
			top = [2]float32{(va.u + ra + vb.u + lb) / 2, va.v}
		} else {
			wa, _ := across(g.nodes[a].w, g.nodes[a].h)
			wb, _ := across(g.nodes[b].w, g.nodes[b].h)
			ta, _ := reach(polys[a], wa/4)
			tb, _ := reach(polys[b], -wb/4)
			s, t := [2]float32{va.u + wa/4, va.v + ta}, [2]float32{vb.u - wb/4, vb.v + tb}
			lift := min(s[1], t[1]) - z.band*1.5
			sg = seg{s, {s[0], lift}, {t[0], lift}, t}
			top = [2]float32{(s[0] + t[0]) / 2, (s[1]+t[1])/8 + lift*3/4}
		}
		for i := range sg {
			sg[i] = final(sg[i])
		}
		e.pts = flatten([]seg{sg})
		if a != e.from {
			slices.Reverse(e.pts)
		}
		c := final([2]float32{top[0], top[1] - z.lane/2 - mw/2})
		e.lx, e.ly = c[0], c[1]
	}

	for u := range g.nodes {
		c := final([2]float32{vs[u].u, vs[u].v})
		g.nodes[u].x, g.nodes[u].y = c[0], c[1]
	}
	for k := range g.clusters {
		cl := &g.clusters[k]
		if cl.shown = len(lefts[k]) > 0; !cl.shown {
			continue
		}
		p0 := final([2]float32{vs[lefts[k][0]].u, b0[first[k]] - before[k]})
		p1 := final([2]float32{vs[rights[k][0]].u, b1[last[k]] + after[k]})
		cl.x0, cl.y0, cl.x1, cl.y1 = min(p0[0], p1[0]), min(p0[1], p1[1]), max(p0[0], p1[0]), max(p0[1], p1[1])
	}
	for _, nd := range g.nodes {
		if len(nd.loops) > 0 {
			for i, k := range nd.loops {
				g.loop(&g.edges[k], i == 0)
			}
		}
	}

	x0, y0 := float32(math.Inf(1)), float32(math.Inf(1))
	x1, y1 := float32(math.Inf(-1)), float32(math.Inf(-1))
	grow := func(x, y, w, h float32) {
		x0, y0, x1, y1 = min(x0, x-w), min(y0, y-h), max(x1, x+w), max(y1, y+h)
	}
	for _, nd := range g.nodes {
		grow(nd.x, nd.y, nd.w/2, nd.h/2)
	}
	for _, cl := range g.clusters {
		if cl.shown {
			grow(cl.x0, cl.y0, 0, 0)
			grow(cl.x1, cl.y1, 0, 0)
		}
	}
	for _, e := range g.edges {
		if len(e.lines) > 0 {
			grow(e.lx, e.ly, e.lw/2, e.lh/2)
		}
		for _, p := range e.pts {
			grow(p[0], p[1], 0, 0)
		}
	}
	dx, dy := z.pad-x0, z.pad-y0
	for u := range g.nodes {
		g.nodes[u].x += dx
		g.nodes[u].y += dy
	}
	for k := range g.clusters {
		cl := &g.clusters[k]
		cl.x0, cl.x1, cl.y0, cl.y1 = cl.x0+dx, cl.x1+dx, cl.y0+dy, cl.y1+dy
	}
	for k := range g.edges {
		e := &g.edges[k]
		e.lx, e.ly = e.lx+dx, e.ly+dy
		for i := range e.pts {
			e.pts[i][0] += dx
			e.pts[i][1] += dy
		}
	}
	g.w, g.h = x1-x0+2*z.pad, y1-y0+2*z.pad
}

// loop lays out e, an edge of a node back to itself, as a loop on the side
// of the node across the ranks, and its label beside it when labeled.
func (g *graph) loop(e *edge, labeled bool) {
	z := g.z
	n := g.nodes[e.from]
	poly := outline(n, z)
	var a, b, c1, c2 [2]float32
	if g.dir == tb || g.dir == bt {
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
	e.pts = flatten([]seg{{a, c1, c2, b}})
	if !labeled {
		e.lines = nil
	}
}

// seg is a cubic Bézier curve: its ends, and its control points between.
type seg [4][2]float32

// basis returns the curve near pts as d3's curveBasis draws it: a uniform
// cubic B-spline from the first point to the last, which passes through
// the middle of three in a line.
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

func straight(a, b [2]float32) seg { return seg{a, lerp(a, b, 1.0/3), lerp(a, b, 2.0/3), b} }

// flatten returns points along the curves sg, more where they are longer.
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
