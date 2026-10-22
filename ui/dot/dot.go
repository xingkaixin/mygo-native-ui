// Package dot draws a graph from its source in Graphviz's DOT language,
// laid out in ranks as Graphviz's dot lays it out: each node a rank after
// the nodes that link to it, the edges that close a cycle turned around,
// and the nodes of each rank ordered so that few edges cross. Edges are
// smooth curves with their labels beside them, ending in the arrowheads
// they name; those of a graph, as against a digraph, end in none.
// Clusters are boxes around their nodes, titled with their labels. A
// click chooses a node, lighting its edges, and sets *chosen to its ID in
// the same frame. A graph too big to fit scrolls both ways. The lines that
// are wrong are listed under it, by number with what is wrong, and the
// graph shows what the others say.
//
//	dot.View(c, &app.graph, app.source, &app.chosen).Grow(1)
//	if app.chosen != "" {
//		... // show what the node stands for
//	}
//
// The source is DOT's statements, which semicolons may end; comments in
// // and /* */, and lines starting with #, are skipped:
//
//	digraph name { ... }          a graph whose edges have arrowheads;
//	graph name { ... }            one whose edges have none; strict
//	                              before either keeps one edge at most
//	                              between two nodes
//	a  "a node"  42               IDs: plain, in quotes, or numbers
//	a [shape=box, label="A\nB"]   a node, with attributes
//	a -> b -> c [color=red]       edges, in a graph a -- b -- c
//	{a b} -> c                    edges from each of a and b to c
//	node [shape=box]              attributes of the nodes and the edges
//	edge [style=dashed]           made after it, in its subgraph or graph
//	rankdir=LR                    an attribute of the graph or subgraph
//	subgraph cluster_x { ... }    a cluster: a box around its nodes
//	{ rank=same; a b }            nodes in one rank
//
// It draws these attributes: label, where \n breaks a line; shape: box,
// rect, ellipse, oval, circle, doublecircle, diamond, plaintext, plain,
// none, point, note and cylinder; style: filled, dashed, dotted, bold,
// rounded and invis; color, fillcolor, fontcolor and penwidth; arrowhead
// and arrowtail: normal, none, empty, dot, odot, diamond and vee; dir;
// rankdir: TB, LR, BT and RL; rank=same; and a cluster's label, color and
// fillcolor. Colors are #rrggbb, #rrggbbaa or X11 names, and show as
// given in either theme; what has none takes the theme's colors, and text
// on a color the source gives is dark or light to stand out from it. It
// skips the other attributes, and reads no HTML labels or records.
package dot

import (
	"fmt"
	"math"

	"github.com/egoist/mygo/ui"
)

// State is what a view keeps: the graph as it parsed and laid out the
// source it was last given. Keep it in the app's state, one for each
// view.
type State struct {
	key key
	g   *graph
}

// key is what a layout is for: the source, and the theme's sizes.
type key struct {
	src         string
	font, space float32
}

// line is a line of text, and its width.
type line struct {
	text string
	w    float32
}

type node struct {
	look
	id      string
	shape   shape
	cluster int
	// loops are its edges back to itself. text is its label as it shows,
	// in lines tw wide and tb tall; w and h are its size, x and y its
	// center.
	loops              []int
	text               string
	lines              []line
	tw, tb, w, h, x, y float32
}

type edge struct {
	look
	from, to int
	// heads are its arrowheads at its tail and its head, and dir the ends
	// that show them: 1 for the head, 2 for the tail.
	heads [2]arrow
	dir   uint8
	line  int
	// scope is the cluster around both its ends, -1 for none. text is its
	// label as it shows, in lines on a label lw by lh centered at lx, ly;
	// pts are its curve, from its tail to its head.
	scope          int
	text           string
	lines          []line
	lw, lh, lx, ly float32
	pts            [][2]float32
}

type cluster struct {
	look
	id     string
	parent int
	// text is its label as it shows, in lines tw wide in a band as tall as
	// band at the top of its box. A cluster with no nodes is not shown.
	text           string
	lines          []line
	tw, band       float32
	shown          bool
	x0, y0, x1, y1 float32
}

type problem struct {
	line int
	msg  string
}

type graph struct {
	id               string
	directed, strict bool
	dir              direction
	nodes            []node
	edges            []edge
	clusters         []cluster
	// sames are the sets of nodes kept in one rank.
	sames [][]int
	errs  []problem
	w, h  float32
	z     sizes
}

// within reports whether cluster a, -1 for the graph, is k or in it; all
// are in the graph.
func (g *graph) within(a, k int) bool {
	for ; a >= 0; a = g.clusters[a].parent {
		if a == k {
			return true
		}
	}
	return k < 0
}

// common returns the innermost cluster around both a and b, -1 for none.
func (g *graph) common(a, b int) int {
	for a >= 0 && !g.within(b, a) {
		a = g.clusters[a].parent
	}
	return a
}

// find returns the node of id, -1 for none.
func (g *graph) find(id string) int {
	for i, n := range g.nodes {
		if id != "" && n.id == id {
			return i
		}
	}
	return -1
}

// at returns the node shown at x, y, -1 for none.
func (g *graph) at(x, y float32) int {
	for i := len(g.nodes) - 1; i >= 0; i-- {
		n := g.nodes[i]
		if n.style&invis == 0 && abs(x-n.x) <= max(n.w/2, 6) && abs(y-n.y) <= max(n.h/2, 6) {
			return i
		}
	}
	return -1
}

// View creates the graph that src describes, and sets *chosen to the ID
// of the node a click chooses, "" for a click beside the nodes. Give it a
// size, or Grow; it scrolls when it does not fit, and the lines that are
// wrong show under it.
func View(c *ui.Context, s *State, src string, chosen *string) ui.Element {
	t := c.Theme()
	k := key{src, t.FontSize, t.Space(1)}
	if s.g == nil || s.key != k {
		s.key, s.g = k, parse(src)
		layout(c, s.g)
	}
	g := s.g
	return ui.Column(c).Children(func() {
		ui.ScrollBoth(c).Grow(1).Shrink(1).Children(func() {
			box := ui.Box(c).Size(g.w, g.h)
			hot := -1
			if px, py, ok := box.PointerPosition(); ok {
				hot = g.at(px, py)
			}
			if hot >= 0 {
				box.Cursor(ui.CursorPointer)
			}
			if box.Clicked() {
				*chosen = ""
				if hot >= 0 {
					*chosen = g.nodes[hot].id
				}
			}
			sel := g.find(*chosen)
			box.Draw(func(p *ui.Painter, r ui.Rect) {
				draw(p, t, g, r.X, r.Y, hot, sel)
			})
		})
		if len(g.errs) == 0 {
			return
		}
		ui.Column(c).Gap(t.Space(1)).Padding(t.Space(2), t.Space(3)).Children(func() {
			for i, e := range g.errs {
				ui.Text(c.Key(i), fmt.Sprintf("Line %d: %s", e.line, e.msg)).FontSize(t.FontSize - 1).TextColor(t.Danger)
			}
		})
	})
}

// width returns how wide the lines of l draw, from Graphviz's pen width:
// 0 draws none.
func width(l look) float32 {
	w := l.pen * 1.25
	if l.style&bold != 0 {
		w *= 2
	}
	return w
}

// draw paints g with its left at ox and top at oy, ringing node hot and
// node chosen and lighting chosen's edges.
func draw(p *ui.Painter, t *ui.Theme, g *graph, ox, oy float32, hot, chosen int) {
	z := g.z
	// pals are the palettes inside each cluster, after that of the graph:
	// a cluster filled with a color of the source's is a ground of its own.
	// Inside each cluster is the color it fills with over what is around
	// it, opaque, so that its title can cover the edges under it.
	pals := make([]palette, len(g.clusters)+1)
	inside := make([]ui.Color, len(g.clusters)+1)
	pals[0], inside[0] = themed(t), t.Background
	for k, cl := range g.clusters {
		pal, around := pals[cl.parent+1], inside[cl.parent+1]
		fill := t.Text.Alpha(0.03)
		if cl.style&filled != 0 {
			fill = pal.ground.Mix(pal.text, 0.07)
			if f := cl.filling(); f.ok {
				fill = f.c
				pal = on(t, f.c.Over(around))
			}
		}
		pals[k+1], inside[k+1] = pal, fill.Over(around)
	}
	text := func(x, y float32, lines []line, size, lh float32, weight int, color ui.Color) {
		for i, l := range lines {
			p.RichText(x-l.w/2, y+float32(i)*lh, 0, ui.Span{Text: l.text, Size: size, Weight: weight, Color: color})
		}
	}

	for k, cl := range g.clusters {
		if !cl.shown || cl.style&invis != 0 {
			continue
		}
		pal := pals[cl.parent+1]
		r := ui.Rect{X: ox + cl.x0, Y: oy + cl.y0, W: cl.x1 - cl.x0, H: cl.y1 - cl.y0}
		radius := t.Space(1)
		if cl.style&rounded != 0 {
			radius = t.Space(3)
		}
		p.Fill(r, inside[k+1], radius)
		border := cl.color.or(pal.border)
		if w := width(cl.look); w > 0 {
			if cl.style&(dashed|dotted) != 0 {
				corners := [][2]float32{{r.X, r.Y}, {r.X + r.W, r.Y}, {r.X + r.W, r.Y + r.H}, {r.X, r.Y + r.H}, {r.X, r.Y}}
				dashes(p, corners, cl.style, w, border)
			} else {
				p.Stroke(r, border, radius, w)
			}
		}
	}

	lit := func(e edge) bool { return chosen >= 0 && (e.from == chosen || e.to == chosen) }
	for _, on := range []bool{false, true} {
		for _, e := range g.edges {
			if e.style&invis != 0 || lit(e) != on {
				continue
			}
			color, w := e.color.or(pals[e.scope+1].line), width(e.look)
			if on {
				color, w = t.Accent, max(w, 1.25)+0.75
			}
			g.drawEdge(p, e, ox, oy, color, w)
		}
	}
	for _, e := range g.edges {
		if len(e.lines) == 0 || e.style&invis != 0 {
			continue
		}
		color := e.font.or(pals[e.scope+1].muted)
		if lit(e) {
			color = t.Accent
		}
		text(ox+e.lx, oy+e.ly-float32(len(e.lines))*z.sh/2, e.lines, t.FontSize-1, z.sh, 400, color)
	}

	for k, cl := range g.clusters {
		if !cl.shown || cl.style&invis != 0 || len(cl.lines) == 0 {
			continue
		}
		th := float32(len(cl.lines)) * z.sh
		x, y := ox+(cl.x0+cl.x1)/2, oy+cl.y0+(cl.band-th)/2
		chip := ui.Rect{X: x - cl.tw/2 - t.Space(1.5), Y: y - t.Space(0.5), W: cl.tw + t.Space(3), H: th + t.Space(1)}
		p.Fill(chip, inside[k+1], t.Space(1))
		text(x, y, cl.lines, t.FontSize-1, z.sh, 600, cl.font.or(pals[k+1].muted))
	}

	for i, n := range g.nodes {
		if n.style&invis != 0 {
			continue
		}
		pal := pals[n.cluster+1]
		x, y := ox+n.x, oy+n.y
		poly := outline(n, z)
		pts := make([][2]float32, len(poly)+1)
		for k, pt := range poly {
			pts[k] = [2]float32{x + pt[0], y + pt[1]}
		}
		pts[len(poly)] = pts[0]
		ink := pal.text
		var fill ui.Color
		switch {
		case n.shape == point:
			fill = n.filling().or(pal.text)
		case n.style&filled != 0:
			fill = pal.ground.Mix(pal.text, 0.1)
			if f := n.filling(); f.ok {
				fill = f.c.Over(pal.ground)
				ink = on(t, fill).text
			}
		}
		if fill.A > 0 {
			p.FillPath(polyline(pts), fill)
		}
		stroke, w := n.color.or(pal.stroke), width(n.look)
		if n.shape != plaintext && n.shape != plain && w > 0 {
			if n.style&(dashed|dotted) != 0 {
				dashes(p, pts, n.style, w, stroke)
			} else {
				p.StrokePath(polyline(pts), w, stroke)
			}
			switch n.shape {
			case doublecircle:
				var inner ui.Path
				inner.Circle(x, y, n.w/2-z.ring)
				p.StrokePath(&inner, w, stroke)
			case note:
				f := min(z.fold, n.h/2)
				x1, y0 := x+n.w/2, y-n.h/2
				p.StrokePolyline([][2]float32{{x1 - f, y0}, {x1 - f, y0 + f}, {x1, y0 + f}}, w, stroke)
			case cylinder:
				const k = 0.5523
				rx, ry := n.w/2, rim(n.w, z)
				top := y - n.h/2 + ry
				var front ui.Path
				front.MoveTo(x-rx, top).CubeTo(x-rx, top+k*ry, x-k*rx, top+ry, x, top+ry).CubeTo(x+k*rx, top+ry, x+rx, top+k*ry, x+rx, top)
				p.StrokePath(&front, w, stroke)
			}
		}
		if i == chosen || i == hot {
			ring, rw := t.Accent, float32(2)
			if i != chosen {
				ring, rw = t.Accent.Alpha(0.45), 1.5
			}
			d := t.Space(1) + w/2
			out := make([][2]float32, len(pts))
			for k, pt := range pts {
				out[k] = [2]float32{x + (pt[0]-x)*(n.w/2+d)/max(n.w/2, 1), y + (pt[1]-y)*(n.h/2+d)/max(n.h/2, 1)}
			}
			p.StrokePath(polyline(out), rw, ring)
		}
		dy := float32(0)
		if n.shape == cylinder {
			dy = rim(n.w, z) / 2
		}
		text(x, y+dy-n.tb/2, n.lines, t.FontSize, z.th, 400, n.font.or(ink))
	}
}

// drawEdge draws e moved by ox, oy, with the arrowheads its dir shows.
func (g *graph) drawEdge(p *ui.Painter, e edge, ox, oy float32, color ui.Color, w float32) {
	if len(e.pts) < 2 || w <= 0 {
		return
	}
	pts := make([][2]float32, len(e.pts))
	for i, pt := range e.pts {
		pts[i] = [2]float32{ox + pt[0], oy + pt[1]}
	}
	l := g.z.head * max(1, w/2)
	shown := [2]bool{e.dir&2 != 0 && e.heads[0] != noHead, e.dir&1 != 0 && e.heads[1] != noHead}
	back := func(j int) float32 {
		if !shown[j] {
			return 0
		}
		switch e.heads[j] {
		case veeHead:
			return l * 0.6
		case dotHead, odotHead:
			return l * 0.8
		case diamondHead:
			return l * 1.4
		}
		return l
	}
	body := cut(pts, back(0), back(1))
	if e.style&(dashed|dotted) != 0 {
		dashes(p, body, e.style, w, color)
	} else if len(body) > 1 {
		stroke(p, body, w, color)
	}
	for j := range 2 {
		if !shown[j] {
			continue
		}
		end := pts
		if j == 0 {
			end = reversed(pts)
		}
		tip := end[len(end)-1]
		from := behind(end, l)
		dx, dy := tip[0]-from[0], tip[1]-from[1]
		if d := float32(math.Hypot(float64(dx), float64(dy))); d > 0 {
			dx, dy = dx/d, dy/d
		} else {
			dx, dy = 1, 0
		}
		at := func(along, side float32) [2]float32 {
			return [2]float32{tip[0] - dx*along - dy*side, tip[1] - dy*along + dx*side}
		}
		half := l * 0.4
		switch e.heads[j] {
		case normalHead, emptyHead:
			tri := polyline([][2]float32{tip, at(l, half), at(l, -half), tip})
			if e.heads[j] == normalHead {
				p.FillPath(tri, color)
			} else {
				p.StrokePath(tri, max(1, w*0.8), color)
			}
		case veeHead:
			p.FillPath(polyline([][2]float32{tip, at(l, half*1.1), at(l*0.6, 0), at(l, -half*1.1), tip}), color)
		case diamondHead:
			p.FillPath(polyline([][2]float32{tip, at(l*0.7, half), at(l*1.4, 0), at(l*0.7, -half), tip}), color)
		case dotHead, odotHead:
			r := l * 0.4
			c := at(r, 0)
			var ring ui.Path
			ring.Circle(c[0], c[1], r)
			if e.heads[j] == dotHead {
				p.FillPath(&ring, color)
			} else {
				p.StrokePath(&ring, max(1, w*0.8), color)
			}
		}
	}
}

// stroke draws the line through pts in runs at most 256 DIPs across:
// mygo draws a line on the CPU as a mask as large as its bounds, and draws
// none larger than 4096 pixels, which an edge across a large graph may be.
func stroke(p *ui.Painter, pts [][2]float32, w float32, color ui.Color) {
	const most = 256
	var run [][2]float32
	var x0, y0, x1, y1 float32
	add := func(q [2]float32) {
		if len(run) > 0 && (max(x1, q[0])-min(x0, q[0]) > most || max(y1, q[1])-min(y0, q[1]) > most) {
			p.StrokePolyline(run, w, color)
			run = [][2]float32{run[len(run)-1]}
			x0, y0, x1, y1 = run[0][0], run[0][1], run[0][0], run[0][1]
		}
		if len(run) == 0 {
			x0, y0, x1, y1 = q[0], q[1], q[0], q[1]
		}
		run = append(run, q)
		x0, y0, x1, y1 = min(x0, q[0]), min(y0, q[1]), max(x1, q[0]), max(y1, q[1])
	}
	for i, q := range pts {
		if i > 0 {
			a := pts[i-1]
			n := int(dist(a, q) / (most / 2))
			for k := 1; k <= n; k++ {
				add(lerp(a, q, float32(k)/float32(n+1)))
			}
		}
		add(q)
	}
	if len(run) > 1 {
		p.StrokePolyline(run, w, color)
	}
}

// polyline returns the path through pts.
func polyline(pts [][2]float32) *ui.Path {
	var path ui.Path
	path.MoveTo(pts[0][0], pts[0][1])
	for _, pt := range pts[1:] {
		path.LineTo(pt[0], pt[1])
	}
	return &path
}

func reversed(pts [][2]float32) [][2]float32 {
	out := make([][2]float32, len(pts))
	for i, pt := range pts {
		out[len(pts)-1-i] = pt
	}
	return out
}

func dist(a, b [2]float32) float32 {
	return float32(math.Hypot(float64(b[0]-a[0]), float64(b[1]-a[1])))
}

func lerp(a, b [2]float32, f float32) [2]float32 {
	return [2]float32{a[0] + (b[0]-a[0])*f, a[1] + (b[1]-a[1])*f}
}

// behind returns the point d along pts back from its end.
func behind(pts [][2]float32, d float32) [2]float32 {
	for i := len(pts) - 1; i > 0; i-- {
		l := dist(pts[i-1], pts[i])
		if l >= d && l > 0 {
			return lerp(pts[i], pts[i-1], d/l)
		}
		d -= l
	}
	return pts[0]
}

// cut returns pts without d0 of its length at the start and d1 at the
// end.
func cut(pts [][2]float32, d0, d1 float32) [][2]float32 {
	start := func(pts [][2]float32, d float32) [][2]float32 {
		for i := 1; i < len(pts); i++ {
			l := dist(pts[i-1], pts[i])
			if l > d {
				return append([][2]float32{lerp(pts[i-1], pts[i], d/l)}, pts[i:]...)
			}
			d -= l
		}
		return nil
	}
	pts = start(pts, d0)
	if len(pts) < 2 {
		return nil
	}
	return reversed(start(reversed(pts), d1))
}

// dashes draws the line through pts in dashes, or in dots when st is
// dotted.
func dashes(p *ui.Painter, pts [][2]float32, st style, w float32, color ui.Color) {
	if len(pts) < 2 {
		return
	}
	on, off := 3*w+2, 2*w+2
	if st&dotted != 0 {
		on, off = w, 2*w+1
	}
	dash := [][2]float32{pts[0]}
	drawing, left := true, on
	for i := 1; i < len(pts); i++ {
		a, b := pts[i-1], pts[i]
		l, at := dist(a, b), float32(0)
		for l-at > left {
			at += left
			q := lerp(a, b, at/l)
			if drawing {
				p.StrokePolyline(append(dash, q), w, color)
				left = off
			} else {
				dash = [][2]float32{q}
				left = on
			}
			drawing = !drawing
		}
		left -= l - at
		if drawing {
			dash = append(dash, b)
		}
	}
	if drawing && len(dash) > 1 {
		p.StrokePolyline(dash, w, color)
	}
}

func abs(v float32) float32 { return max(v, -v) }
