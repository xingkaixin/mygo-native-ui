package classdiagram

import (
	"cmp"
	"math"
	"slices"

	"github.com/egoist/mygo/ui"
)

// sizes are the sizes of the parts of a diagram, from the theme.
type sizes struct {
	th, sh, row, padX, padY, inset, min, radius, gap, lane, band, port, stub, loop, pad, tri, dia, arrow float32
}

func measure(c *ui.Context) sizes {
	t := c.Theme()
	_, th := c.MeasureText(0, ui.Span{Text: "Ag", Size: t.FontSize, Weight: 600})
	_, sh := c.MeasureText(0, ui.Span{Text: "Ag", Size: t.FontSize - 1})
	return sizes{th: th, sh: sh, row: sh + t.Space(1), padX: t.Space(3), padY: t.Space(2), inset: t.Space(1.5),
		min: t.Space(28), radius: t.Space(1.5), gap: t.Space(10), lane: t.Space(3), band: t.Space(9),
		port: t.Space(8), stub: t.Space(5), loop: t.Space(7), pad: t.Space(5),
		tri: t.Space(3.5), dia: t.Space(4.5), arrow: t.Space(2.75)}
}

// weight is how far up the ranks an end pulls its class: a parent above
// those that inherit from it, a whole above its parts, and what an arrow
// points to below where it comes from.
func weight(e end) int {
	switch e {
	case triangle:
		return 3
	case filled, hollow:
		return 2
	case arrow:
		return 0
	}
	return 1
}

// layout sizes the classes, the labels and the cardinalities, and places
// them.
func layout(c *ui.Context, d *diagram) {
	t := c.Theme()
	z := measure(c)
	d.z = z
	width := func(spans ...ui.Span) float32 {
		w, _ := c.MeasureText(0, spans...)
		return float32(math.Ceil(float64(w)))
	}
	size := t.FontSize - 1
	for i := range d.classes {
		cl := &d.classes[i]
		w := width(ui.Span{Text: cl.name, Size: t.FontSize, Weight: 600})
		cl.head = z.th + 2*z.padY
		if cl.note != "" {
			w = max(w, width(ui.Span{Text: "«" + cl.note + "»", Size: size}))
			cl.head += z.sh
		}
		for _, m := range slices.Concat(cl.fields, cl.methods) {
			w = max(w, width(spans(m, size, ui.Color{}, ui.Color{})...))
		}
		cl.fh = float32(len(cl.fields))*z.row + 2*z.inset
		cl.mh = float32(len(cl.methods))*z.row + 2*z.inset
		cl.w, cl.h = max(z.min, w+2*z.padX), cl.head+cl.fh+cl.mh
	}
	for i := range d.rels {
		r := &d.rels[i]
		for j, card := range r.cards {
			if card != "" {
				r.cw[j] = width(ui.Span{Text: card, Size: size})
			}
		}
		if r.label != "" {
			r.lw, r.lh = width(ui.Span{Text: r.label, Size: size})+2*t.Space(1.5), z.sh+t.Space(1)
		}
	}
	place(d, z)
}

// vert is a vertex of the layered graph: a class, or a point a relation
// passes in a rank between its ends, which may hold its label. lo and hi
// are how far it reaches before and after its center u across the ranks,
// size how far along them; at is its place in its rank.
type vert struct {
	class              int
	layer              int
	lo, hi, size, u, v float32
	at                 float64
}

// route is a relation as laid out: a and b are its classes in rank order,
// flip whether a is the class it goes to, path the vertices it passes
// and label that of its label, -1 for none.
type route struct {
	rel, a, b int
	flip      bool
	path      []int
	label     int
}

// place lays out the classes and the relations between them, as
// Sugiyama's method does: it puts each class a rank below those its
// relations pull above it, turning around those that close a cycle,
// moves those it can to shorten their relations, gives each relation a
// vertex in each rank it passes and its label one in a rank between
// classes, orders each rank so that few relations cross, sets each vertex
// across as near those it links to as the ones beside it let it, and runs
// relations through their vertices, leaving and meeting their classes
// square. It lays out down the ranks, and turns the diagram the way it
// goes as it places it.
func place(d *diagram, z sizes) {
	n := len(d.classes)
	if n == 0 {
		d.w, d.h = 0, 0
		return
	}
	vertical := d.dir == down || d.dir == up
	across := func(w, h float32) (float32, float32) {
		if vertical {
			return w, h
		}
		return h, w
	}
	final := func(p [2]float32) [2]float32 {
		switch d.dir {
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
	for k, r := range d.rels {
		if r.from == r.to {
			continue
		}
		rt := route{rel: k, a: r.from, b: r.to, label: -1}
		if weight(r.ends[1]) > weight(r.ends[0]) {
			rt.a, rt.b, rt.flip = r.to, r.from, true
		}
		routes = append(routes, rt)
	}
	out := make([][]int, n)
	in := make([]int, n)
	for k, rt := range routes {
		out[rt.a] = append(out[rt.a], k)
		in[rt.b]++
	}
	mark := make([]uint8, n)
	back := make([]bool, len(routes))
	var walk func(u int)
	walk = func(u int) {
		mark[u] = 1
		for _, k := range out[u] {
			switch v := routes[k].b; mark[v] {
			case 0:
				walk(v)
			case 1:
				back[k] = true
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
		if rt := &routes[k]; back[k] {
			rt.a, rt.b, rt.flip = rt.b, rt.a, !rt.flip
		}
	}
	rank := make([]int, n)
	for range n {
		moved := false
		for _, rt := range routes {
			if rank[rt.a]+1 > rank[rt.b] {
				rank[rt.b], moved = rank[rt.a]+1, true
			}
		}
		if !moved {
			break
		}
	}
	// A class with more relations below than above moves down as far as
	// they let it, and one with more above than below up.
	for range 4 {
		for u := range n {
			lo, hi, ins, outs := 0, math.MaxInt, 0, 0
			for _, rt := range routes {
				if rt.b == u {
					lo, ins = max(lo, rank[rt.a]+1), ins+1
				}
				if rt.a == u {
					hi, outs = min(hi, rank[rt.b]-1), outs+1
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

	// Classes are in the even layers, and labels in the odd ones between.
	vs := make([]vert, 0, n)
	for u, cl := range d.classes {
		cw, mw := across(cl.w, cl.h)
		v := vert{class: u, layer: 2 * rank[u], lo: cw / 2, hi: cw / 2, size: mw}
		if len(cl.loops) > 0 {
			_, _, out := d.loops(cl)
			v.hi += slices.Max(out)
		}
		vs = append(vs, v)
	}
	for k := range routes {
		rt := &routes[k]
		r := d.rels[rt.rel]
		la, lb := 2*rank[rt.a], 2*rank[rt.b]
		mid := (la + lb) / 2
		if mid%2 == 0 {
			mid--
		}
		rt.path = []int{rt.a}
		for L := la + 1; L < lb; L++ {
			x := vert{class: -1, layer: L}
			if L == mid && r.label != "" {
				cw, mw := across(r.lw, r.lh)
				x.lo, x.hi, x.size = cw/2, cw/2, mw
				rt.label = len(vs)
			}
			rt.path = append(rt.path, len(vs))
			vs = append(vs, x)
		}
		rt.path = append(rt.path, rt.b)
	}
	layers := 0
	for _, v := range vs {
		layers = max(layers, v.layer+1)
	}
	cols := make([][]int, layers)
	for u, v := range vs {
		vs[u].at = float64(len(cols[v.layer]))
		cols[v.layer] = append(cols[v.layer], u)
	}
	ups, downs := make([][]int, len(vs)), make([][]int, len(vs))
	type segment struct{ a, b int }
	var segs []segment
	for _, rt := range routes {
		for i := 1; i < len(rt.path); i++ {
			a, b := rt.path[i-1], rt.path[i]
			downs[a], ups[b] = append(downs[a], b), append(ups[b], a)
			segs = append(segs, segment{a, b})
		}
	}

	// Each rank in turn is ordered by where the vertices it links to in
	// the rank before are, down the ranks and back up, keeping the order
	// with the fewest crossings.
	sweep := func(L int, near [][]int) {
		key := make(map[int]float64, len(cols[L]))
		for _, u := range cols[L] {
			key[u] = vs[u].at
			if len(near[u]) > 0 {
				sum := 0.0
				for _, w := range near[u] {
					sum += vs[w].at
				}
				key[u] = sum / float64(len(near[u]))
			}
		}
		slices.SortStableFunc(cols[L], func(a, b int) int { return cmp.Compare(key[a], key[b]) })
		for k, u := range cols[L] {
			vs[u].at = float64(k)
		}
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
	best, least := save(), crossings()
	for range 12 {
		if least == 0 {
			break
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
	// apart by a gap.
	depth := make([]float32, layers)
	for _, v := range vs {
		depth[v.layer] = max(depth[v.layer], v.size)
	}
	b0, b1 := make([]float32, layers), make([]float32, layers)
	for L := range layers {
		if L > 0 {
			b0[L] = b1[L-1] + z.band
		}
		b1[L] = b0[L] + depth[L]
	}
	for u := range vs {
		vs[u].v = (b0[vs[u].layer] + b1[vs[u].layer]) / 2
	}

	// Across the ranks: the vertices of each rank in order, apart by
	// their sizes and gaps, each pulled toward those it links to, by the
	// pool adjacent violators algorithm: where a vertex wants to be less
	// how far it must be after the first must not fall along the rank.
	sep := func(a, b int) float32 {
		va, vb := vs[a], vs[b]
		gap := z.lane
		switch {
		case va.class >= 0 && vb.class >= 0:
			gap = z.gap
		case va.class >= 0 || vb.class >= 0:
			gap = z.gap / 2
		}
		return va.hi + vb.lo + gap
	}
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
		for i, u := range col {
			want, k := float32(0), 0
			if down {
				for _, w := range ups[u] {
					want, k = want+vs[w].u, k+1
				}
			}
			if up {
				for _, w := range downs[u] {
					want, k = want+vs[w].u, k+1
				}
			}
			if k > 0 {
				want /= float32(k)
			} else {
				want = vs[u].u
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

	// Each relation leaves the side of its class facing the next rank, the
	// relations of a side spread across it in the order they go, runs
	// straight out of the band of its rank, curves between the bands and
	// straight across those it passes, and comes in straight.
	type port struct {
		route  int
		toward float32
	}
	sides := make([][2][]port, n)
	for k, rt := range routes {
		p := rt.path
		sides[rt.a][1] = append(sides[rt.a][1], port{k, vs[p[1]].u})
		sides[rt.b][0] = append(sides[rt.b][0], port{k, vs[p[len(p)-2]].u})
	}
	offs := make([][2]float32, len(routes))
	for u, cl := range d.classes {
		cw, _ := across(cl.w, cl.h)
		for j, ports := range sides[u] {
			slices.SortStableFunc(ports, func(a, b port) int { return cmp.Compare(a.toward, b.toward) })
			step := min(z.port, max(0, cw-4*z.radius)/float32(max(len(ports)-1, 1)))
			for i, p := range ports {
				offs[p.route][1-j] = (float32(i) - float32(len(ports)-1)/2) * step
			}
		}
	}
	for k, rt := range routes {
		r := &d.rels[rt.rel]
		p := rt.path
		a, b := vs[p[0]], vs[p[len(p)-1]]
		start := [2]float32{a.u + offs[k][0], a.v + a.size/2}
		end := [2]float32{b.u + offs[k][1], b.v - b.size/2}
		at := [2]float32{start[0], max(b1[a.layer], start[1]+z.stub)}
		pts := [][2]float32{start, at}
		curve := func(to [2]float32) {
			pts = append(pts, bend(at, to)[1:]...)
			at = to
		}
		// A rank of no depth, between classes with no labels, the curve
		// passes in one S rather than two.
		for _, x := range p[1 : len(p)-1] {
			if v := vs[x]; b1[v.layer] > b0[v.layer] {
				curve([2]float32{v.u, b0[v.layer]})
				at = [2]float32{v.u, b1[v.layer]}
				pts = append(pts, at)
			}
		}
		curve([2]float32{end[0], min(b0[b.layer], end[1]-z.stub)})
		pts = append(pts, end)
		for i := range pts {
			pts[i] = final(pts[i])
		}
		if rt.label >= 0 {
			v := vs[rt.label]
			c := final([2]float32{v.u, v.v})
			r.lx, r.ly = c[0], c[1]
		}
		// A cardinality sits by its end, on the side away from the middle
		// of the class, past the marker.
		for j, e := range [2][2]float32{start, end} {
			i := j
			if rt.flip {
				i = 1 - j
			}
			if r.cards[i] == "" {
				continue
			}
			pw, aw := across(r.cw[i], z.sh)
			side, out := float32(1), float32(1-2*j)
			if offs[k][j] < 0 {
				side = -1
			}
			r.at[i] = final([2]float32{e[0] + side*(z.tri*0.6+z.inset+pw/2), e[1] + out*(z.inset+aw/2)})
		}
		if rt.flip {
			pts = reversed(pts)
		}
		r.pts = pts
	}

	for u := range d.classes {
		c := final([2]float32{vs[u].u, vs[u].v})
		d.classes[u].x, d.classes[u].y = c[0], c[1]
	}
	for _, cl := range d.classes {
		for k, i := range cl.loops {
			d.loop(&d.rels[i], cl, k)
		}
	}

	x0, y0 := float32(math.Inf(1)), float32(math.Inf(1))
	x1, y1 := float32(math.Inf(-1)), float32(math.Inf(-1))
	grow := func(x, y, w, h float32) {
		x0, y0, x1, y1 = min(x0, x-w), min(y0, y-h), max(x1, x+w), max(y1, y+h)
	}
	for _, cl := range d.classes {
		grow(cl.x, cl.y, cl.w/2, cl.h/2)
	}
	for _, r := range d.rels {
		if r.label != "" {
			grow(r.lx, r.ly, r.lw/2, r.lh/2)
		}
		for j, card := range r.cards {
			if card != "" {
				grow(r.at[j][0], r.at[j][1], r.cw[j]/2, z.sh/2)
			}
		}
		for _, p := range r.pts {
			grow(p[0], p[1], 0, 0)
		}
	}
	dx, dy := z.pad-x0, z.pad-y0
	for u := range d.classes {
		d.classes[u].x += dx
		d.classes[u].y += dy
	}
	for k := range d.rels {
		r := &d.rels[k]
		r.lx, r.ly = r.lx+dx, r.ly+dy
		for j := range r.at {
			r.at[j][0] += dx
			r.at[j][1] += dy
		}
		for i := range r.pts {
			r.pts[i][0] += dx
			r.pts[i][1] += dy
		}
	}
	d.w, d.h = x1-x0+2*z.pad, y1-y0+2*z.pad
}

// loops returns how far apart the loops of cl are along its side across
// the ranks, with room between for their cardinalities, how tall each is,
// and how far each reaches out of the side, with its label beside it.
func (d *diagram) loops(cl class) (step, tall float32, out []float32) {
	z := d.z
	vertical := d.dir == down || d.dir == up
	length := cl.h
	if !vertical {
		length = cl.w
	}
	// room returns how far along the side the end j of r and its
	// cardinality reach from where it meets the side.
	room := func(r relation, j int) float32 {
		w := z.tri * 0.6
		if r.cards[j] != "" {
			along, _ := d.card(r.cw[j])
			w += z.inset + along
		}
		return w
	}
	tall, gap := z.port*1.25, float32(0)
	for i, k := range cl.loops {
		r := d.rels[k]
		if i > 0 {
			gap = max(gap, room(d.rels[cl.loops[i-1]], 1)+room(r, 0)+z.inset)
		}
		o := z.loop
		for j, card := range r.cards {
			if card != "" {
				_, away := d.card(r.cw[j])
				o = max(o, z.inset+away)
			}
		}
		if r.label != "" {
			lw := r.lw
			if !vertical {
				lw = r.lh
			}
			o = max(o, z.loop+z.inset+lw)
		}
		out = append(out, o)
	}
	n := float32(len(cl.loops))
	step = tall + gap
	if total, space := (n-1)*step+tall, length-2*z.radius; total > space {
		step, tall = step*space/total, tall*space/total
	}
	return step, tall, out
}

// card returns how far a cardinality w wide reaches along the side of a
// class across the ranks, and away from it.
func (d *diagram) card(w float32) (float32, float32) {
	if d.dir == down || d.dir == up {
		return d.z.sh, w
	}
	return w, d.z.sh
}

// loop lays out r, the k-th relation of cl to itself, as a loop out of
// the side of cl across the ranks, after the loops before it, with its
// cardinalities by its ends and its label beside it.
func (d *diagram) loop(r *relation, cl class, k int) {
	z := d.z
	n := len(cl.loops)
	vertical := d.dir == down || d.dir == up
	step, tall, _ := d.loops(cl)
	// Points are u along the side and v away from it, turned where they
	// show by at.
	side := cl.w / 2
	if !vertical {
		side = cl.h / 2
	}
	at := func(u, v float32) [2]float32 {
		if vertical {
			return [2]float32{cl.x + v, cl.y + u}
		}
		return [2]float32{cl.x + u, cl.y + v}
	}
	c := (float32(k) - float32(n-1)/2) * step
	a, b, out := c-tall/2, c+tall/2, side+z.loop
	r.pts = rounded([][2]float32{at(a, side), at(a, out), at(b, out), at(b, side)}, z.radius*2)
	if r.label != "" {
		lw := r.lw
		if !vertical {
			lw = r.lh
		}
		p := at(c, out+z.inset+lw/2)
		r.lx, r.ly = p[0], p[1]
	}
	for j, card := range r.cards {
		if card == "" {
			continue
		}
		along, away := d.card(r.cw[j])
		u := a - z.tri*0.6 - z.inset/2 - along/2
		if j == 1 {
			u = b + z.tri*0.6 + z.inset/2 + along/2
		}
		r.at[j] = at(u, side+z.inset+away/2)
	}
}

// rounded returns the line through pts with its corners rounded by
// radius.
func rounded(pts [][2]float32, radius float32) [][2]float32 {
	out := [][2]float32{pts[0]}
	for i := 1; i < len(pts)-1; i++ {
		a, p, b := pts[i-1], pts[i], pts[i+1]
		s, e := lerp(p, a, radius/dist(p, a)), lerp(p, b, radius/dist(p, b))
		const steps = 6
		for k := range steps + 1 {
			u := float32(k) / steps
			out = append(out, lerp(lerp(s, p, u), lerp(p, e, u), u))
		}
	}
	return append(out, pts[len(pts)-1])
}

// bend returns points along the curve from a to b that leaves a and
// comes into b along the ranks, more where it is longer.
func bend(a, b [2]float32) [][2]float32 {
	if a[0] == b[0] {
		return [][2]float32{a, b}
	}
	m := (b[1] - a[1]) / 2
	c1, c2 := [2]float32{a[0], a[1] + m}, [2]float32{b[0], b[1] - m}
	l := dist(a, c1) + dist(c1, c2) + dist(c2, b)
	steps := int(min(24, max(2, l/6)))
	pts := make([][2]float32, 0, steps+1)
	for k := 0; k <= steps; k++ {
		u := float32(k) / float32(steps)
		v := 1 - u
		p, q, r, s := v*v*v, 3*v*v*u, 3*v*u*u, u*u*u
		pts = append(pts, [2]float32{p*a[0] + q*c1[0] + r*c2[0] + s*b[0], p*a[1] + q*c1[1] + r*c2[1] + s*b[1]})
	}
	return pts
}
