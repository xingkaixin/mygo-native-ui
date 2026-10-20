// Package forcegraph lays out a graph by simulating forces, as the graph
// view of a notes app does: nodes push one another apart, links pull the
// nodes they join together as springs do, a weak pull keeps the graph in
// the middle, and nodes do not overlap. The layout moves until it settles,
// and settles at once while the desktop asks for less motion. A node is
// larger the more links it has, and colored by its group.
//
//	forcegraph.View(c, &app.graph, nodes, links, &app.chosen).Grow(1)
//
// Hovering a node lights it, its links and its neighbors, dims the rest
// and names them. The chosen node and its neighbors are named too, and
// every node once zoomed in far enough. Dragging a node moves it, the
// others following, and holds it there until it is let go. Dragging the
// background pans, scrolling zooms around the pointer, and a double click
// on the background fits the graph in the view again. A click chooses a
// node, or none on the background; Changed reports it.
//
// Where the nodes are is kept by their IDs, so the app may change the
// nodes and links: the layout moves again, and new nodes start by their
// neighbors.
package forcegraph

import (
	"math"
	"slices"

	"github.com/egoist/mygo/ui"
)

// Node is a node of a graph: its ID, which keeps its place as the nodes
// change, the label it shows, and its group, which gives it its color.
type Node struct {
	ID, Label, Group string
}

// Link links two nodes, by their indexes.
type Link struct {
	From, To int
}

// State is what a view keeps: where each node is and how fast it moves,
// by ID, how far the layout is from settling, how far the view is zoomed
// and panned, and the node held. The zero State fits the graph in the
// view, following it as it moves, until the user zooms, pans or drags a
// node. Keep it in the app's state, one for each view.
type State struct {
	bodies map[string]*body
	// alpha is how far the layout is from settling: the forces act in
	// proportion to it, and it cools tick by tick, as in d3-force.
	alpha float32
	links uint64
	frame uint32

	zoom, panX, panY float32
	free             bool

	pressing bool
	held     string
	// travel is how far the pointer moved since the press began.
	travel float32

	xs, ys []float32
	tree   tree
}

type body struct {
	x, y, vx, vy float32
	seen         uint32
}

// The simulation's constants are d3-force's defaults, the pull to the
// middle that of its x and y forces.
const (
	alphaMin   = 0.001
	alphaDecay = 0.0228 // cools to alphaMin in 300 ticks
	reheat     = 0.3
	keep       = 0.6 // of a body's velocity from one tick to the next
	charge     = 30
	theta2     = 0.81 // Barnes–Hut: a quad as wide as 0.9 of its distance acts as one body
	distance   = 30
	gravity    = 0.1
	gap        = 2 // kept between nodes, besides their radii
)

// The view's zoom limits, and the zoom from which every node is named.
const (
	minZoom   = 0.05
	maxZoom   = 8
	labelZoom = 3
)

// graph is a frame's graph: the nodes' bodies and radii, the links between
// nodes that are there, and the neighbors of node i, nbs[start[i]:start[i+1]].
type graph struct {
	bodies     []*body
	r          []float32
	links      []Link
	start, nbs []int32
}

func (g *graph) neighbors(i int) []int32 { return g.nbs[g.start[i]:g.start[i+1]] }

func (g *graph) degree(i int) float32 { return float32(g.start[i+1] - g.start[i]) }

// resolve finds the bodies of nodes, placing those it has not seen by
// their neighbors, and forgets those gone. A change to the nodes or the
// links warms the layout up.
func (s *State) resolve(nodes []Node, links []Link) *graph {
	if s.bodies == nil {
		s.bodies = map[string]*body{}
	}
	first := len(s.bodies) == 0
	s.frame++
	n := len(nodes)
	g := &graph{bodies: make([]*body, n), r: make([]float32, n), start: make([]int32, n+1)}
	sig := uint64(14695981039346656037)
	for _, l := range links {
		if l.From < 0 || l.To < 0 || l.From >= n || l.To >= n || l.From == l.To {
			continue
		}
		g.links = append(g.links, l)
		g.start[l.From+1]++
		g.start[l.To+1]++
		sig = (sig ^ uint64(l.From)<<32 ^ uint64(l.To)) * 1099511628211
	}
	for i := range n {
		g.start[i+1] += g.start[i]
	}
	g.nbs = make([]int32, g.start[n])
	fill := slices.Clone(g.start[:n])
	for _, l := range g.links {
		g.nbs[fill[l.From]], g.nbs[fill[l.To]] = int32(l.To), int32(l.From)
		fill[l.From]++
		fill[l.To]++
	}

	placed := make([]bool, n)
	var fresh []int
	live := 0
	for i, nd := range nodes {
		b := s.bodies[nd.ID]
		if b == nil {
			b = &body{}
			s.bodies[nd.ID] = b
			fresh = append(fresh, i)
		} else {
			placed[i] = true
		}
		if b.seen != s.frame {
			b.seen = s.frame
			live++
		}
		g.bodies[i] = b
		g.r[i] = 2.5 + sqrt(g.degree(i))
	}
	for k, i := range fresh {
		b := g.bodies[i]
		var sx, sy, m float32
		for _, j := range g.neighbors(i) {
			if placed[j] {
				sx, sy, m = sx+g.bodies[j].x, sy+g.bodies[j].y, m+1
			}
		}
		if m > 0 && !first {
			// Turning by the golden angle, about 2.4 radians, sets apart
			// the nodes that come at once by the same neighbors.
			a := float64(k) * 2.4
			b.x, b.y = sx/m+10*float32(math.Cos(a)), sy/m+10*float32(math.Sin(a))
		} else {
			// A sunflower's spiral, as d3-force starts nodes, spreads
			// them evenly without two in one place.
			rad, a := 10*math.Sqrt(0.5+float64(i)), float64(i)*math.Pi*(3-math.Sqrt(5))
			b.x, b.y = float32(rad*math.Cos(a)), float32(rad*math.Sin(a))
		}
		placed[i] = true
	}
	gone := len(s.bodies) - live
	if gone > 0 {
		for id, b := range s.bodies {
			if b.seen != s.frame {
				delete(s.bodies, id)
			}
		}
	}
	switch {
	case first && n > 0:
		s.alpha = 1
	case len(fresh) > 0 || gone > 0 || sig != s.links:
		s.alpha = max(s.alpha, reheat)
	}
	s.links = sig
	return g
}

// tick moves the bodies of g a step, as d3-force's simulation does with
// its link, many-body, x and y and collide forces, but for the body held,
// which stays where it is.
func (s *State) tick(g *graph, held int) {
	s.alpha -= s.alpha * alphaDecay
	a := s.alpha
	bs := g.bodies
	for _, l := range g.links {
		from, to := bs[l.From], bs[l.To]
		dx, dy := to.x+to.vx-from.x-from.vx, to.y+to.vy-from.y-from.vy
		d := sqrt(dx*dx + dy*dy)
		if d == 0 {
			continue
		}
		nf, nt := g.degree(l.From), g.degree(l.To)
		k := (d - distance) / d * a / min(nf, nt)
		dx, dy = dx*k, dy*k
		bias := nf / (nf + nt)
		to.vx, to.vy = to.vx-dx*bias, to.vy-dy*bias
		from.vx, from.vy = from.vx+dx*(1-bias), from.vy+dy*(1-bias)
	}
	s.xs = slices.Grow(s.xs[:0], len(bs))[:len(bs)]
	s.ys = slices.Grow(s.ys[:0], len(bs))[:len(bs)]
	for i, b := range bs {
		s.xs[i], s.ys[i] = b.x, b.y
	}
	s.tree.build(s.xs, s.ys, g.r)
	for i, b := range bs {
		fx, fy := s.tree.repel(int32(i), s.xs, s.ys)
		b.vx += (fx*charge - b.x*gravity) * a
		b.vy += (fy*charge - b.y*gravity) * a
	}
	for i, b := range bs {
		s.xs[i], s.ys[i] = b.x+b.vx, b.y+b.vy
	}
	s.tree.build(s.xs, s.ys, g.r)
	s.tree.collide(bs, s.xs, s.ys, g.r)
	for i, b := range bs {
		if i == held {
			b.vx, b.vy = 0, 0
			continue
		}
		b.vx, b.vy = b.vx*keep, b.vy*keep
		b.x, b.y = b.x+b.vx, b.y+b.vy
	}
}

// quad is a square of a quadtree: its top-left corner and side, the quads
// of its quarters, 0 for none, and, for a leaf, the first of its bodies;
// and how many bodies it has, their middle, and the largest radius.
type quad struct {
	x, y, size float32
	kids       [4]int32
	first      int32
	n, cx, cy  float32
	r          float32
}

// tree is a quadtree of bodies, at xs, ys, in which next links the bodies
// of a leaf, -1 ending them.
type tree struct {
	quads []quad
	next  []int32
	stack []int32
}

func (t *tree) build(xs, ys, rs []float32) {
	t.quads = t.quads[:0]
	t.next = slices.Grow(t.next[:0], len(xs))[:len(xs)]
	if len(xs) == 0 {
		return
	}
	x0, y0, x1, y1 := xs[0], ys[0], xs[0], ys[0]
	for i := range xs {
		x0, y0, x1, y1 = min(x0, xs[i]), min(y0, ys[i]), max(x1, xs[i]), max(y1, ys[i])
	}
	t.quads = append(t.quads, quad{x: x0, y: y0, size: max(x1-x0, y1-y0) + 1, first: -1})
	for i := range xs {
		t.insert(int32(i), xs, ys)
	}
	t.sum(0, xs, ys, rs)
}

// kid returns the quarter of quad q that x, y is in, adding it if there is
// none.
func (t *tree) kid(q int32, x, y float32) int32 {
	p := t.quads[q]
	h := p.size / 2
	k, kx, ky := 0, p.x, p.y
	if x >= p.x+h {
		k, kx = k|1, kx+h
	}
	if y >= p.y+h {
		k, ky = k|2, ky+h
	}
	if p.kids[k] == 0 {
		t.quads = append(t.quads, quad{x: kx, y: ky, size: h, first: -1})
		t.quads[q].kids[k] = int32(len(t.quads) - 1)
	}
	return t.quads[q].kids[k]
}

func (t *tree) insert(i int32, xs, ys []float32) {
	t.next[i] = -1
	q := int32(0)
	for {
		p := &t.quads[q]
		if p.kids != [4]int32{} {
			q = t.kid(q, xs[i], ys[i])
			continue
		}
		j := p.first
		if j < 0 {
			p.first = i
			return
		}
		// Bodies in one place, or nearly, share a leaf rather than
		// splitting it without end.
		if p.size < 1e-2 || xs[j] == xs[i] && ys[j] == ys[i] {
			t.next[i], p.first = j, i
			return
		}
		p.first = -1
		t.quads[t.kid(q, xs[j], ys[j])].first = j
	}
}

func (t *tree) sum(q int32, xs, ys, rs []float32) {
	p := &t.quads[q]
	var n, sx, sy, r float32
	for j := p.first; j >= 0; j = t.next[j] {
		n, sx, sy, r = n+1, sx+xs[j], sy+ys[j], max(r, rs[j])
	}
	for _, k := range p.kids {
		if k == 0 {
			continue
		}
		t.sum(k, xs, ys, rs)
		kq := &t.quads[k]
		n, sx, sy, r = n+kq.n, sx+kq.cx*kq.n, sy+kq.cy*kq.n, max(r, kq.r)
	}
	p.n, p.cx, p.cy, p.r = n, sx/n, sy/n, r
}

// repel returns the push on body i from the others, falling with their
// distance, from a far quad's bodies as from one at their middle.
func (t *tree) repel(i int32, xs, ys []float32) (fx, fy float32) {
	push := func(dx, dy, n float32) {
		d2 := dx*dx + dy*dy
		if d2 < 1 {
			// As d3's distanceMin, so that bodies nearly in one place do
			// not fly apart.
			d2 = sqrt(d2)
		}
		fx, fy = fx+dx*n/d2, fy+dy*n/d2
	}
	t.stack = append(t.stack[:0], 0)
	for len(t.stack) > 0 {
		q := t.quads[t.stack[len(t.stack)-1]]
		t.stack = t.stack[:len(t.stack)-1]
		dx, dy := xs[i]-q.cx, ys[i]-q.cy
		if q.first < 0 && q.size*q.size < theta2*(dx*dx+dy*dy) {
			push(dx, dy, q.n)
			continue
		}
		for j := q.first; j >= 0; j = t.next[j] {
			if j != i {
				dx, dy := xs[i]-xs[j], ys[i]-ys[j]
				if dx == 0 && dy == 0 {
					dx = 1e-3 * float32(i-j)
				}
				push(dx, dy, 1)
			}
		}
		for _, k := range q.kids {
			if k != 0 {
				t.stack = append(t.stack, k)
			}
		}
	}
	return fx, fy
}

// collide pushes apart the bodies that overlap, as they will be at xs,
// ys, by how far they do, the smaller moving the more.
func (t *tree) collide(bs []*body, xs, ys, rs []float32) {
	for i := range bs {
		xi, yi, ri := xs[i], ys[i], rs[i]+gap/2
		t.stack = append(t.stack[:0], 0)
		for len(t.stack) > 0 {
			q := t.quads[t.stack[len(t.stack)-1]]
			t.stack = t.stack[:len(t.stack)-1]
			reach := ri + q.r + gap/2
			if q.x > xi+reach || q.y > yi+reach || q.x+q.size < xi-reach || q.y+q.size < yi-reach {
				continue
			}
			for j := q.first; j >= 0; j = t.next[j] {
				if int(j) <= i {
					continue
				}
				rj := rs[j] + gap/2
				dx, dy, rr := xi-xs[j], yi-ys[j], ri+rj
				d2 := dx*dx + dy*dy
				if d2 >= rr*rr {
					continue
				}
				if d2 == 0 {
					dx = 1e-3 * float32(int(j)-i)
					d2 = dx * dx
				}
				d := sqrt(d2)
				k := (rr - d) / d
				dx, dy = dx*k, dy*k
				w := rj * rj / (ri*ri + rj*rj)
				bs[i].vx, bs[i].vy = bs[i].vx+dx*w, bs[i].vy+dy*w
				bs[j].vx, bs[j].vy = bs[j].vx-dx*(1-w), bs[j].vy-dy*(1-w)
			}
			for _, k := range q.kids {
				if k != 0 {
					t.stack = append(t.stack, k)
				}
			}
		}
	}
}

func (s *State) scale() float32 {
	if s.zoom == 0 {
		return 1
	}
	return s.zoom
}

// zoomAt zooms to z keeping the point of the graph at x, y in the view
// where it is.
func (s *State) zoomAt(z, x, y float32) {
	old := s.scale()
	z = max(minZoom, min(maxZoom, z))
	s.panX = x - (x-s.panX)*z/old
	s.panY = y - (y-s.panY)*z/old
	s.zoom = z
}

// fit zooms and pans a view w by h to show all of g inside pad, no
// closer than twice its size.
func (s *State) fit(g *graph, w, h, pad float32) {
	b := g.bodies[0]
	x0, y0, x1, y1 := b.x, b.y, b.x, b.y
	for i, b := range g.bodies {
		x0, y0 = min(x0, b.x-g.r[i]), min(y0, b.y-g.r[i])
		x1, y1 = max(x1, b.x+g.r[i]), max(y1, b.y+g.r[i])
	}
	s.zoom = max(minZoom, min(2, (w-2*pad)/(x1-x0), (h-2*pad)/(y1-y0)))
	s.panX, s.panY = w/2-(x0+x1)/2*s.zoom, h/2-(y0+y1)/2*s.zoom
}

// nodeAt returns the node at x, y of the view, the nearest if several are,
// or -1.
func (s *State) nodeAt(g *graph, x, y float32) int {
	z := s.scale()
	at, best := -1, float32(math.MaxFloat32)
	for i, b := range g.bodies {
		dx, dy := s.panX+b.x*z-x, s.panY+b.y*z-y
		reach := max(1.5, g.r[i]*z) + 3
		if d := dx*dx + dy*dy; d < reach*reach && d < best {
			at, best = i, d
		}
	}
	return at
}

// View creates the graph of nodes and links, choosing *chosen, a node's
// ID or "" for none. Give it a size, or Grow.
func View(c *ui.Context, s *State, nodes []Node, links []Link, chosen *string) ui.Element {
	t := c.Theme()
	canvas := ui.Box(c).Role(ui.RoleGroup).Label("Graph")
	b := canvas.LayoutBounds()
	g := s.resolve(nodes, links)
	n := len(nodes)
	index := func(id string) int { return slices.IndexFunc(nodes, func(nd Node) bool { return nd.ID == id }) }

	px, py, _ := canvas.PointerPosition()
	hit := -1
	// Hovered, unlike PointerPosition, is false under an overlay, as the
	// combobox's options over the graph.
	if canvas.Hovered() {
		hit = s.nodeAt(g, px, py)
	}
	dx, dy, pressing := canvas.Dragged()
	if pressing && !s.pressing {
		s.held, s.travel = "", 0
		if hit >= 0 {
			s.held = nodes[hit].ID
		}
	}
	held := -1
	if pressing && s.held != "" {
		held = index(s.held)
	}
	if pressing && (dx != 0 || dy != 0) {
		s.travel += abs(dx) + abs(dy)
		s.free = true
		if held >= 0 {
			hb := g.bodies[held]
			hb.x, hb.y = hb.x+dx/s.scale(), hb.y+dy/s.scale()
		} else {
			s.panX, s.panY = s.panX+dx, s.panY+dy
		}
	}
	if held >= 0 && s.travel > 0 {
		s.alpha = max(s.alpha, reheat)
	}
	// A press that moves a little, as a trackpad's may, is still a click.
	if canvas.Clicked() && s.travel <= 3 {
		id := ""
		if hit >= 0 {
			id = nodes[hit].ID
		}
		if *chosen != id {
			*chosen = id
			canvas.SetChanged()
		}
	}
	if canvas.DoubleClicked() && hit < 0 {
		s.free = false
	}
	s.pressing = pressing
	canvas.HandleInput(func(ev ui.InputEvent) bool {
		if ev.Kind != ui.InputScroll {
			return false
		}
		s.free = true
		// Scrolling 300 DIPs zooms by 2, as zoomview does.
		s.zoomAt(s.scale()*float32(math.Exp2(float64(-ev.DY)/300)), ev.X, ev.Y)
		return true
	})

	if n > 0 {
		if c.Preferences().ReduceMotion {
			for held < 0 && s.alpha >= alphaMin {
				s.tick(g, -1)
			}
		} else if s.alpha >= alphaMin {
			s.tick(g, held)
			c.AnimationFrame()
		}
		if !s.free && b.W > 0 && b.H > 0 {
			s.fit(g, b.W, b.H, t.Space(6))
		}
	}

	hover := hit
	if pressing {
		hover = held
	}
	switch {
	case pressing && s.travel > 0:
		canvas.Cursor(ui.CursorGrabbing)
	case hover >= 0:
		canvas.Cursor(ui.CursorPointer)
	}
	lit, named := make([]bool, n), make([]bool, n)
	mark := func(i int, set []bool) {
		set[i] = true
		for _, j := range g.neighbors(i) {
			set[j] = true
		}
	}
	ch := index(*chosen)
	if hover >= 0 {
		mark(hover, lit)
		mark(hover, named)
	}
	if ch >= 0 {
		mark(ch, named)
	}
	focus := hover
	if focus < 0 {
		focus = ch
	}

	z := s.scale()
	all := z >= labelZoom
	type dot struct {
		x, y, r float32
		color   ui.Color
	}
	dots := make([]dot, n)
	for i, bd := range g.bodies {
		color := hue(nodes[i].Group)
		if hover >= 0 && !lit[i] {
			color = color.Alpha(0.2)
		}
		dots[i] = dot{s.panX + bd.x*z, s.panY + bd.y*z, max(1.5, g.r[i]*z), color}
	}
	gl := g.links
	// Element.Clip hides only what the children draw outside the element.
	canvas.Draw(func(p *ui.Painter, r ui.Rect) {
		p.Clip(r, 0, func() {
			var faint, bright ui.Path
			for _, l := range gl {
				a, b := dots[l.From], dots[l.To]
				path := &faint
				if l.From == focus || l.To == focus {
					path = &bright
				}
				path.MoveTo(r.X+a.x, r.Y+a.y).LineTo(r.X+b.x, r.Y+b.y)
			}
			faintColor := t.Text.Alpha(0.16)
			if hover >= 0 {
				faintColor = t.Text.Alpha(0.06)
			}
			p.StrokePath(&faint, 1, faintColor)
			p.StrokePath(&bright, 1.5, t.Accent)
			shows := func(d dot, margin float32) bool {
				return d.x+margin > 0 && d.x-margin < r.W && d.y+margin > 0 && d.y-margin < r.H
			}
			// The nodes lit go over those dimmed.
			for _, front := range []bool{false, true} {
				for i, d := range dots {
					if front == (hover < 0 || lit[i]) && shows(d, d.r) {
						var circle ui.Path
						circle.Circle(r.X+d.x, r.Y+d.y, d.r)
						p.FillPath(&circle, d.color)
					}
				}
			}
			if ch >= 0 {
				var ring ui.Path
				ring.Circle(r.X+dots[ch].x, r.Y+dots[ch].y, dots[ch].r+3)
				p.StrokePath(&ring, 2, t.Accent)
			}
			size := t.FontSize - 2
			// label draws the label of node i below it, on back, which keeps it
			// apart from the nodes and links under it.
			label := func(i int, color, back ui.Color) {
				d := dots[i]
				if !shows(d, d.r+t.Space(30)) {
					return
				}
				w, h := p.MeasureText(0, ui.Span{Text: nodes[i].Label, Size: size})
				x, y := r.X+d.x-w/2, r.Y+d.y+d.r+2
				p.Fill(ui.Rect{X: x - 4, Y: y, W: w + 8, H: h}, back, h/2)
				p.Text(x, y, nodes[i].Label, size, color)
			}
			for i := range dots {
				if all && !named[i] {
					color := t.TextMuted
					if hover >= 0 {
						color = color.Alpha(0.4)
					}
					label(i, color, ui.Transparent)
				}
			}
			for i := range dots {
				if named[i] {
					color, back := t.Text, t.Background.Alpha(0.85)
					if hover >= 0 && !lit[i] {
						color, back = t.TextMuted.Alpha(0.6), back.Alpha(0.4)
					}
					label(i, color, back)
				}
			}
		})
	})
	return canvas
}

// hue returns the color of group, its hue from a hash of its name so that
// it keeps it as other groups come and go.
func hue(group string) ui.Color {
	h := uint32(2166136261)
	for i := range len(group) {
		h = (h ^ uint32(group[i])) * 16777619
	}
	// FNV-1a leaves names that differ only at their end close; mixing the
	// bits as MurmurHash3 finishes spreads them around the wheel.
	h = (h ^ h>>16) * 0x85ebca6b
	h = (h ^ h>>13) * 0xc2b2ae35
	return ui.Oklch(0.68, 0.14, float32((h^h>>16)%360))
}

func sqrt(v float32) float32 { return float32(math.Sqrt(float64(v))) }

func abs(v float32) float32 { return max(v, -v) }
