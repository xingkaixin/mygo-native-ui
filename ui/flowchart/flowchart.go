// Package flowchart draws a flowchart from text, as Mermaid's flowchart
// does: the nodes in ranks down the chart, or across it, each after the
// nodes that link to it, and ordered so that few links cross; boxes,
// rounded boxes, stadiums, cylinders, circles, diamonds, hexagons and the
// other shapes Mermaid has; links as curves between them, solid, dotted
// or thick, ending in arrowheads, circles or crosses, with their text on
// them; and subgraphs as titled boxes around their nodes. The pointer
// lights the node under it and its links, and a click chooses a node,
// which Chosen returns. A chart too big to fit scrolls both ways. The
// lines that are wrong are listed under it, by number with what is wrong,
// and the chart shows what the others say.
//
//	flowchart.View(c, &app.chart, app.source).Grow(1)
//	if n, ok := app.chart.Chosen(); ok {
//		... // show what n.ID stands for
//	}
//
// The source has a statement a line, or several apart by semicolons;
// lines starting with %% are comments:
//
//	flowchart TD           the header: TD or TB down, BT up, LR right, RL left
//	A                      a node, a box of its id
//	A[text], A(text)       a box of text, a rounded one
//	A([text]) A[[text]] A[(text)]     a stadium, a subroutine, a cylinder
//	A((text)) A(((text))) A{text}     a circle, a double one, a diamond
//	A{{text}} A>text]                 a hexagon, a flag
//	A[/text/] A[\text\] A[/text\] A[\text/]   leaning boxes, trapezoids
//	A["text"]              text in quotes, which may hold brackets
//	A --> B, A --- B       a link with an arrowhead, one without
//	A -.-> B, A ==> B      a dotted one, a thick one
//	A --o B, A --x B       one ending in a circle, in a cross
//	A <--> B               one with an arrowhead at each end
//	A ~~~ B                one that places B after A but does not show
//	A -- text --> B        a link with text, as A -->|text| B
//	A ---> B               a longer link, a rank more for each dash more
//	A --> B --> C          a chain of links
//	A & B --> C & D        links from each of A and B to each of C and D
//	subgraph id [title]    a box around the nodes first named in it,
//	end                    up to end; subgraphs nest
//
// <br> breaks the text of a node or a link. A node keeps its shape and
// text when it is named again. A link may go to or from a subgraph's id,
// and then ends at its box. Lines of classDef, class, style, linkStyle,
// click and direction are skipped.
package flowchart

import (
	"fmt"
	"math"

	"github.com/egoist/mygo/ui"
)

// Node is a node of a chart: its id, as the source names it, and its
// text, its lines apart by newlines.
type Node struct {
	ID, Text string
}

// State is what a view keeps: the chart as it parsed and laid out the
// source it was last given, and the node chosen. Keep it in the app's
// state, one for each view.
type State struct {
	key key
	ch  *chart
	// chosen is the id of the node chosen, "" for none.
	chosen string
}

// Chosen returns the node chosen with a click, while the chart has it.
func (s *State) Chosen() (Node, bool) {
	if s.ch == nil {
		return Node{}, false
	}
	i := s.ch.find(s.chosen)
	if i < 0 {
		return Node{}, false
	}
	return Node{s.ch.nodes[i].id, s.ch.nodes[i].text}, true
}

// key is what a layout is for: the source, and the theme's sizes.
type key struct {
	src         string
	font, space float32
}

// line is a line of text as it wraps, and its width.
type line struct {
	text string
	w    float32
}

type node struct {
	id, text string
	shape    shape
	// group is the subgraph the node is in, -1 for none; loops are its
	// links back to itself.
	group int
	loops []int
	// lines are its text as it wraps, tw wide and tb tall; w and h are
	// its size, x and y its center.
	lines              []line
	tw, tb, w, h, x, y float32
}

type edge struct {
	from, to int
	link
	line int
	// clip is the subgraph where the edge starts and where it ends, -1
	// for none: from or to is then a node in it, and the edge is cut at
	// its box.
	clip [2]int
	// lines are its text as it wraps, on a label lw by lh centered at lx,
	// ly, in subgraph scope; pts are its line, from where it leaves to
	// where it arrives.
	lines          []line
	lw, lh, lx, ly float32
	scope          int
	pts            [][2]float32
}

type group struct {
	id, title    string
	parent, line int
	// lines are its title as it wraps, tw wide, in a band as tall as band
	// at the top of its box.
	lines          []line
	tw, band       float32
	x0, y0, x1, y1 float32
}

type chart struct {
	dir    direction
	nodes  []node
	edges  []edge
	groups []group
	errs   []problem
	w, h   float32
	z      sizes
}

// find returns the node of id, -1 for none.
func (ch *chart) find(id string) int {
	for i, n := range ch.nodes {
		if id != "" && n.id == id {
			return i
		}
	}
	return -1
}

// at returns the node at x, y, -1 for none.
func (ch *chart) at(x, y float32) int {
	for i, n := range ch.nodes {
		if abs(x-n.x) <= n.w/2 && abs(y-n.y) <= n.h/2 {
			return i
		}
	}
	return -1
}

// lit reports whether edge e is a link of node n.
func (e edge) lit(n int) bool {
	return n >= 0 && (e.from == n && e.clip[0] < 0 || e.to == n && e.clip[1] < 0)
}

// View creates the chart that src describes. Give it a size, or Grow; it
// scrolls when it does not fit, and the lines that are wrong show under
// it.
func View(c *ui.Context, s *State, src string) ui.Element {
	t := c.Theme()
	k := key{src, t.FontSize, t.Space(1)}
	if s.ch == nil || s.key != k {
		s.key, s.ch = k, parse(src)
		layout(c, s.ch)
	}
	ch := s.ch
	return ui.Column(c).Children(func() {
		ui.ScrollBoth(c).Grow(1).Shrink(1).Children(func() {
			box := ui.Box(c).Size(ch.w, ch.h)
			hot := -1
			if px, py, ok := box.PointerPosition(); ok {
				hot = ch.at(px, py)
			}
			if hot >= 0 {
				box.Cursor(ui.CursorPointer)
			}
			if box.Clicked() {
				s.chosen = ""
				if hot >= 0 {
					s.chosen = ch.nodes[hot].id
				}
			}
			chosen := ch.find(s.chosen)
			box.Draw(func(p *ui.Painter, r ui.Rect) {
				draw(p, t, ch, r.X, r.Y, hot, chosen)
			})
		})
		if len(ch.errs) == 0 {
			return
		}
		ui.Column(c).Gap(t.Space(1)).Padding(t.Space(2), t.Space(3)).Children(func() {
			for i, e := range ch.errs {
				ui.Text(c.Key(i), fmt.Sprintf("Line %d: %s", e.line, e.msg)).FontSize(t.FontSize - 1).TextColor(t.Danger)
			}
		})
	})
}

// draw paints ch with its left at ox and top at oy, lighting node hot and
// node chosen and their links.
func draw(p *ui.Painter, t *ui.Theme, ch *chart, ox, oy float32, hot, chosen int) {
	z := ch.z
	tint := t.Text.Alpha(0.035)
	text := func(x, y float32, lines []line, size, lh float32, weight int, color ui.Color, center bool) {
		for i, l := range lines {
			lx := x
			if center {
				lx -= l.w / 2
			}
			p.RichText(lx, y+float32(i)*lh, 0, ui.Span{Text: l.text, Size: size, Weight: weight, Color: color})
		}
	}
	depth := make([]int, len(ch.groups))
	for g, gr := range ch.groups {
		if gr.parent >= 0 {
			depth[g] = depth[gr.parent] + 1
		}
		r := ui.Rect{X: ox + gr.x0, Y: oy + gr.y0, W: gr.x1 - gr.x0, H: gr.y1 - gr.y0}
		p.Fill(r, tint, z.radius*1.5)
		p.Stroke(r, t.Border, z.radius*1.5, 1)
		text(r.X+z.inset, r.Y+(gr.band-float32(len(gr.lines))*z.sh)/2, gr.lines, t.FontSize-1, z.sh, 600, t.TextMuted, false)
	}

	lit := func(e edge) bool { return e.lit(hot) || e.lit(chosen) }
	for _, on := range []bool{false, true} {
		for _, e := range ch.edges {
			if e.stroke == hidden || lit(e) != on {
				continue
			}
			// Opaque, so that a line under its arrowhead or another line
			// does not show darker.
			color, width := t.Background.Mix(t.Text, 0.5), float32(1.25)
			if e.stroke == thick {
				width = 2.75
			}
			if on {
				color, width = t.Accent, width+0.75
			}
			ch.drawEdge(p, t, e, ox, oy, color, width)
		}
	}
	for _, e := range ch.edges {
		if len(e.lines) == 0 || e.stroke == hidden {
			continue
		}
		r := ui.Rect{X: ox + e.lx - e.lw/2, Y: oy + e.ly - e.lh/2, W: e.lw, H: e.lh}
		p.Fill(r, t.Background, t.Space(1.5))
		if e.scope >= 0 {
			for range depth[e.scope] + 1 {
				p.Fill(r, tint, t.Space(1.5))
			}
		}
		ink := t.TextMuted
		if lit(e) {
			ink = t.Accent
		}
		text(ox+e.lx, oy+e.ly-float32(len(e.lines))*z.sh/2, e.lines, t.FontSize-1, z.sh, 400, ink, true)
	}

	for i, n := range ch.nodes {
		fill, border, width := t.Surface, t.Border, float32(1)
		switch i {
		case chosen:
			fill, border, width = t.Surface.Mix(t.Accent, 0.16), t.Accent, 2
		case hot:
			fill, border, width = t.Surface.Mix(t.Accent, 0.06), t.Accent, 1.5
		}
		x, y := ox+n.x, oy+n.y
		ch.drawNode(p, t, n, x, y, fill, border, width)
		dx, dy := float32(0), float32(0)
		switch n.shape {
		case flag:
			dx = n.h / 6
		case cylinder:
			dy = rim(n.w, z) / 2
		}
		text(x+dx, y+dy-n.tb/2, n.lines, t.FontSize, z.th, 500, t.Text, true)
	}
}

// drawNode draws the shape of n centered at x, y.
func (ch *chart) drawNode(p *ui.Painter, t *ui.Theme, n node, x, y float32, fill, border ui.Color, width float32) {
	z := ch.z
	r := ui.Rect{X: x - n.w/2, Y: y - n.h/2, W: n.w, H: n.h}
	box := func(radius float32) {
		p.Fill(r, t.Background, radius)
		p.Fill(r, fill, radius)
		p.Stroke(r, border, radius, width)
	}
	shape := func(path *ui.Path) {
		p.FillPath(path, t.Background)
		p.FillPath(path, fill)
		p.StrokePath(path, width, border)
	}
	switch n.shape {
	case rect:
		box(t.Space(0.5))
	case round:
		box(z.radius)
	case stadium:
		box(n.h / 2)
	case subroutine:
		box(t.Space(0.5))
		p.Line(r.X+z.sub, r.Y, r.X+z.sub, r.Y+r.H, width, border)
		p.Line(r.X+r.W-z.sub, r.Y, r.X+r.W-z.sub, r.Y+r.H, width, border)
	case circle, double:
		var ring ui.Path
		ring.Circle(x, y, n.w/2)
		shape(&ring)
		if n.shape == double {
			var inner ui.Path
			inner.Circle(x, y, n.w/2-z.ring)
			p.StrokePath(&inner, width, border)
		}
	case cylinder:
		const k = 0.5523
		rx, ry := n.w/2, rim(n.w, z)
		x0, x1, top, bottom := r.X, r.X+r.W, r.Y+ry, r.Y+r.H-ry
		var body, front ui.Path
		body.MoveTo(x0, top).CubeTo(x0, top-k*ry, x-k*rx, top-ry, x, top-ry).CubeTo(x+k*rx, top-ry, x1, top-k*ry, x1, top).
			LineTo(x1, bottom).CubeTo(x1, bottom+k*ry, x+k*rx, bottom+ry, x, bottom+ry).CubeTo(x-k*rx, bottom+ry, x0, bottom+k*ry, x0, bottom).Close()
		shape(&body)
		front.MoveTo(x0, top).CubeTo(x0, top+k*ry, x-k*rx, top+ry, x, top+ry).CubeTo(x+k*rx, top+ry, x1, top+k*ry, x1, top)
		p.StrokePath(&front, width, border)
	default:
		var path ui.Path
		for i, pt := range outline(n, z) {
			if i == 0 {
				path.MoveTo(x+pt[0], y+pt[1])
			} else {
				path.LineTo(x+pt[0], y+pt[1])
			}
		}
		path.Close()
		shape(&path)
	}
}

// drawEdge draws e moved by ox, oy, with what ends it.
func (ch *chart) drawEdge(p *ui.Painter, t *ui.Theme, e edge, ox, oy float32, color ui.Color, width float32) {
	z := ch.z
	if len(e.pts) < 2 {
		return
	}
	pts := make([][2]float32, len(e.pts))
	for i, pt := range e.pts {
		pts[i] = [2]float32{ox + pt[0], oy + pt[1]}
	}
	scale := max(1, width/2)
	back := func(h head) float32 {
		switch h {
		case arrow:
			return z.head * scale / 2
		case dot:
			return 2 * z.dot * scale
		case cross:
			return z.head * scale / 2
		}
		return 0
	}
	body := cut(pts, back(e.heads[0]), back(e.heads[1]))
	if e.stroke == dotted {
		dashes(p, body, 3, 3, width, color)
	} else if len(body) > 1 {
		p.StrokePath(polyline(body), width, color)
	}
	for j, h := range e.heads {
		if h == bare {
			continue
		}
		end := pts
		if j == 0 {
			end = reversed(pts)
		}
		tip := end[len(end)-1]
		from := point(end, 2)
		dx, dy := tip[0]-from[0], tip[1]-from[1]
		if d := float32(math.Hypot(float64(dx), float64(dy))); d > 0 {
			dx, dy = dx/d, dy/d
		} else {
			dx, dy = 1, 0
		}
		switch h {
		case arrow:
			l, w := z.head*scale, t.Space(1.25)*scale
			bx, by := tip[0]-dx*l, tip[1]-dy*l
			var tri ui.Path
			tri.MoveTo(tip[0], tip[1]).LineTo(bx-dy*w, by+dx*w).LineTo(bx+dy*w, by-dx*w).Close()
			p.FillPath(&tri, color)
		case dot:
			r := z.dot * scale
			var ring ui.Path
			ring.Circle(tip[0]-dx*r, tip[1]-dy*r, r)
			p.FillPath(&ring, color)
		case cross:
			r := z.head * scale / 2
			cx, cy := tip[0]-dx*r, tip[1]-dy*r
			ax, ay := (dx-dy)*r, (dy+dx)*r
			p.Line(cx-ax, cy-ay, cx+ax, cy+ay, width, color)
			p.Line(cx+ay, cy-ax, cx-ay, cy+ax, width, color)
		}
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

// point returns the point d along pts back from its end.
func point(pts [][2]float32, d float32) [2]float32 {
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

// dashes draws the line through pts in dashes on long, off apart.
func dashes(p *ui.Painter, pts [][2]float32, on, off, width float32, color ui.Color) {
	if len(pts) < 2 {
		return
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
				p.StrokePolyline(append(dash, q), width, color)
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
		p.StrokePolyline(dash, width, color)
	}
}

func abs(v float32) float32 { return max(v, -v) }
