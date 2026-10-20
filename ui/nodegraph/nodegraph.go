// Package nodegraph is a node editor, as Blender's shader editor and
// ComfyUI are: nodes on a canvas, each a title, its inputs down its left
// and its outputs down its right, and what the app builds in it, and
// links from outputs to inputs, drawn as curves.
//
//	nodegraph.View(c, &app.canvas, &app.graph, func(n *nodegraph.Node[Op]) {
//		ui.Slider(c, &n.Data.Amount, 0, 1)
//	}).Grow(1)
//
// Dragging a node moves it; a click chooses it, and Delete or
// Backspace removes it with its links. Dragging from an output to an
// input links them, in place of the link the input had, and dragging the
// end of a link off its input takes it out, to drop on another input or
// let go of. Links that would make a cycle do not take. Dragging the
// canvas or scrolling pans, and scrolling with Cmd held, Command on macOS
// and Control elsewhere, zooms around the pointer, as do Cmd+= and Cmd+-
// around the middle; Cmd+0 shows the graph at its size.
package nodegraph

import (
	"math"
	"slices"

	"github.com/egoist/mygo/ui"
)

// Node is a node of a graph: its title, where its top-left corner is in
// the graph, the names of its inputs and its outputs, and the app's data
// of it, which the node's body shows.
type Node[T any] struct {
	Title   string
	X, Y    float32
	Inputs  []string
	Outputs []string
	Data    T
}

// Link links output Out of node From to input In of node To, by their
// indexes.
type Link struct {
	From, Out int
	To, In    int
}

// Graph is the nodes and the links between them, which the view changes
// as the user edits it.
type Graph[T any] struct {
	Nodes []Node[T]
	Links []Link
}

// Input returns the link into input in of node to, if any.
func (g *Graph[T]) Input(to, in int) (Link, bool) {
	i := slices.IndexFunc(g.Links, func(l Link) bool { return l.To == to && l.In == in })
	if i < 0 {
		return Link{}, false
	}
	return g.Links[i], true
}

// remove removes node i and its links, the links after it following it.
func (g *Graph[T]) remove(i int) {
	g.Nodes = slices.Delete(g.Nodes, i, i+1)
	g.Links = slices.DeleteFunc(g.Links, func(l Link) bool { return l.From == i || l.To == i })
	for k := range g.Links {
		l := &g.Links[k]
		if l.From > i {
			l.From--
		}
		if l.To > i {
			l.To--
		}
	}
}

// reaches reports whether links lead from node from to node to.
func (g *Graph[T]) reaches(from, to int) bool {
	seen := map[int]bool{}
	var walk func(n int) bool
	walk = func(n int) bool {
		if n == to {
			return true
		}
		if seen[n] {
			return false
		}
		seen[n] = true
		for _, l := range g.Links {
			if l.From == n && walk(l.To) {
				return true
			}
		}
		return false
	}
	return walk(from)
}

// State is what a view keeps: how far it is zoomed and panned, the node
// chosen, the link being dragged, and how wide the nodes were. Keep it in
// the app's state, one for each view.
type State struct {
	zoom, panX, panY float32
	w, h             float32
	// chosen is the index of the node chosen, plus 1, so that the zero
	// State chooses none.
	chosen int
	wire   *wire
	widths []float32
}

// wire is a link being dragged from output out of node from, its end at
// x, y in the graph.
type wire struct {
	from, out int
	x, y      float32
}

// Chosen returns the index of the node chosen.
func (s *State) Chosen() (int, bool) { return s.chosen - 1, s.chosen > 0 }

// Center returns the point of the graph in the middle of the view, as
// where to put a node the user adds.
func (s *State) Center() (x, y float32) {
	z := s.scale()
	return (s.w/2 - s.panX) / z, (s.h/2 - s.panY) / z
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
	z = max(0.25, min(2, z))
	s.panX = x - (x-s.panX)*z/old
	s.panY = y - (y-s.panY)*z/old
	s.zoom = z
}

// The sizes of a node's parts, in DIPs of the graph: its title, a row of
// its ports, and a port's dot.
const (
	titleH = 30
	rowH   = 24
	dot    = 10
)

// View creates the editor of g, building what body builds in each node,
// below its ports, for the node's data; nil builds nothing. Give it a
// size, or Grow. Changed reports the user linking, unlinking or removing
// nodes.
func View[T any](c *ui.Context, s *State, g *Graph[T], body func(n *Node[T])) ui.Element {
	t := c.Theme()
	canvas := ui.Box(c).Clip().Focusable().Role(ui.RoleGroup).Label("Node graph").Background(t.Surface)
	b := canvas.LayoutBounds()
	s.w, s.h = b.W, b.H
	before := len(g.Links)
	z := s.scale()

	// graph returns the point of the graph at x, y in the view.
	graph := func(x, y float32) (float32, float32) { return (x - s.panX) / z, (y - s.panY) / z }
	if dx, dy, ok := canvas.Dragged(); ok && s.wire == nil {
		s.panX, s.panY = s.panX+dx, s.panY+dy
	}
	if canvas.Pressed() {
		s.chosen = 0
	}
	if i, ok := s.Chosen(); ok && i < len(g.Nodes) &&
		(canvas.Shortcut(0, ui.KeyDelete) || canvas.Shortcut(0, ui.KeyBackspace)) {
		g.remove(i)
		s.chosen = 0
		canvas.SetChanged()
	}
	for _, k := range []struct {
		key ui.Key
		z   float32
	}{{ui.KeyEqual, z * 1.25}, {ui.KeyMinus, z / 1.25}, {ui.Key0, 1}} {
		if canvas.Shortcut(ui.Cmd, k.key) {
			s.zoomAt(k.z, s.w/2, s.h/2)
		}
	}
	canvas.HandleInput(func(ev ui.InputEvent) bool {
		if ev.Kind != ui.InputScroll {
			return false
		}
		if ev.Mods&ui.Cmd != 0 {
			// Scrolling 300 DIPs zooms by 2, as zoomview does.
			s.zoomAt(s.scale()*float32(math.Exp2(float64(-ev.DY)/300)), ev.X, ev.Y)
			return true
		}
		s.panX, s.panY = s.panX-ev.DX, s.panY-ev.DY
		return true
	})
	if s.wire != nil {
		px, py, _ := canvas.PointerPosition()
		s.wire.x, s.wire.y = graph(px, py)
	}

	s.widths = slices.Grow(s.widths[:0], len(g.Nodes))[:len(g.Nodes)]
	canvas.Children(func() {
		for i := range g.Nodes {
			node(c.Key(i), s, g, i, body)
		}
	})
	if len(g.Links) != before {
		canvas.SetChanged()
	}

	nodes := g.Nodes
	links := slices.Clone(g.Links)
	widths := slices.Clone(s.widths)
	wire := s.wire
	chosen := s.chosen - 1
	panX, panY := s.panX, s.panY
	canvas.Draw(func(p *ui.Painter, r ui.Rect) {
		at := func(x, y float32) (float32, float32) { return r.X + panX + x*z, r.Y + panY + y*z }
		grid(p, r, panX, panY, z, t.Text.Alpha(0.12))
		for _, l := range links {
			if l.From >= len(nodes) || l.To >= len(nodes) {
				continue
			}
			x0, y0 := at(port(nodes[l.From], widths[l.From], l.Out, true))
			x1, y1 := at(port(nodes[l.To], 0, l.In, false))
			color := t.TextMuted
			if l.From == chosen || l.To == chosen {
				color = t.Accent
			}
			curve(p, x0, y0, x1, y1, 2*z, color)
		}
		if wire != nil && wire.from < len(nodes) {
			x0, y0 := at(port(nodes[wire.from], widths[wire.from], wire.out, true))
			x1, y1 := at(wire.x, wire.y)
			curve(p, x0, y0, x1, y1, 2*z, t.Accent)
		}
	})
	return canvas
}

// port returns where port k of n, an output or an input, is in the graph,
// w the width of n.
func port[T any](n Node[T], w float32, k int, output bool) (float32, float32) {
	x := n.X
	if output {
		x += w
	}
	// The border, 1 DIP wide, is inside the node.
	return x, n.Y + 1 + titleH + float32(k)*rowH + rowH/2
}

// node builds node i of g: dragging it moves it, and its ports start
// and end links.
func node[T any](c *ui.Context, s *State, g *Graph[T], i int, body func(n *Node[T])) {
	t := c.Theme()
	n := &g.Nodes[i]
	z := s.scale()
	r := t.Space(2)
	border := t.Border
	if s.chosen == i+1 {
		border = t.Accent
	}
	el := ui.Column(c).Absolute().Left(s.panX+n.X*z).Top(s.panY+n.Y*z).TransformOrigin(0, 0).Scale(z, z).
		MinWidth(t.Space(40)).Radius(r).Background(t.Background).Border(1, border).
		Shadow(0, 2, 8, 0, ui.RGB(0, 0, 0).Alpha(0.12)).Role(ui.RoleGroup).Label(n.Title)
	s.widths[i] = el.LayoutBounds().W
	pressed := el.Pressed()
	if pressed {
		s.chosen = i + 1
	}
	if dx, dy, ok := el.Dragged(); ok {
		n.X, n.Y = n.X+dx, n.Y+dy
	}
	el.Children(func() {
		title := ui.Row(c).Height(titleH).Padding(0, t.Space(3)).AlignItems(ui.Center).
			Radius(r-1, r-1, 0, 0).Background(t.Text.Alpha(0.05)).Cursor(ui.CursorGrab)
		if pressed {
			title.Cursor(ui.CursorGrabbing)
		}
		title.Children(func() {
			ui.Text(c, n.Title).Bold().FontSize(t.FontSize - 1).SingleLine()
		})
		for k := range max(len(n.Inputs), len(n.Outputs)) {
			ui.Row(c.Key(k)).Height(rowH).Gap(t.Space(4)).AlignItems(ui.Center).Children(func() {
				if k < len(n.Inputs) {
					input(c, s, g, i, k)
				}
				ui.Box(c).Grow(1)
				if k < len(n.Outputs) {
					output(c, s, g, i, k)
				}
			})
		}
		if body != nil {
			ui.Column(c).Gap(t.Space(2)).Padding(t.Space(1), t.Space(3), t.Space(3)).Children(func() { body(n) })
		}
	})
}

// input shows input k of node i: its dot, on the node's edge, and its
// name. Dragging the dot of a linked input takes its link out.
func input[T any](c *ui.Context, s *State, g *Graph[T], i, k int) {
	t := c.Theme()
	link, linked := g.Input(i, k)
	ui.Row(c).Gap(t.Space(1.5)).AlignItems(ui.Center).Children(func() {
		d := portDot(c, linked).MarginStart(-dot/2 - 1).Label("Input " + g.Nodes[i].Inputs[k])
		if _, _, ok := d.Dragged(); ok && linked && s.wire == nil {
			g.Links = slices.DeleteFunc(g.Links, func(l Link) bool { return l == link })
			x, y := port(g.Nodes[i], 0, k, false)
			s.wire = &wire{from: link.From, out: link.Out, x: x, y: y}
		}
		if _, _, ok := d.DragEnded(); ok {
			drop(s, g)
		}
		ui.Text(c, g.Nodes[i].Inputs[k]).FontSize(t.FontSize - 2).TextColor(t.TextMuted).SingleLine()
	})
}

// output shows output k of node i: its name and its dot, on the node's
// edge, which dragging starts a link from.
func output[T any](c *ui.Context, s *State, g *Graph[T], i, k int) {
	t := c.Theme()
	linked := slices.ContainsFunc(g.Links, func(l Link) bool { return l.From == i && l.Out == k })
	ui.Row(c).Gap(t.Space(1.5)).AlignItems(ui.Center).Children(func() {
		ui.Text(c, g.Nodes[i].Outputs[k]).FontSize(t.FontSize - 2).TextColor(t.TextMuted).SingleLine()
		d := portDot(c, linked).MarginEnd(-dot/2 - 1).Label("Output " + g.Nodes[i].Outputs[k])
		if _, _, ok := d.Dragged(); ok && s.wire == nil {
			x, y := port(g.Nodes[i], s.widths[i], k, true)
			s.wire = &wire{from: i, out: k, x: x, y: y}
		}
		if _, _, ok := d.DragEnded(); ok {
			drop(s, g)
		}
	})
}

// portDot creates the dot of a port, filled when linked.
func portDot(c *ui.Context, linked bool) ui.Element {
	t := c.Theme()
	d := ui.Box(c).Size(dot, dot).Cursor(ui.CursorCrosshair).Role(ui.RoleButton)
	return d.Draw(func(p *ui.Painter, r ui.Rect) {
		var circle ui.Path
		circle.Circle(r.X+r.W/2, r.Y+r.H/2, r.W/2-0.75)
		fill := t.Background
		if linked || d.Hovered() {
			fill = t.Accent
		}
		p.FillPath(&circle, fill)
		p.StrokePath(&circle, 1.5, t.Accent)
	})
}

// reach is how near an input a link dropped takes, in DIPs of the graph.
const reach = 16

// drop ends the link being dragged on the input nearest its end, unless
// none is near, it is of the node the link comes from, or it makes a
// cycle; it takes the input's place of any link it had.
func drop[T any](s *State, g *Graph[T]) {
	w := s.wire
	s.wire = nil
	if w == nil {
		return
	}
	to, in, best := -1, -1, float32(reach*reach)
	for i, n := range g.Nodes {
		for k := range n.Inputs {
			x, y := port(n, 0, k, false)
			if d := (x-w.x)*(x-w.x) + (y-w.y)*(y-w.y); d < best {
				to, in, best = i, k, d
			}
		}
	}
	if to < 0 || to == w.from || g.reaches(to, w.from) {
		return
	}
	g.Links = slices.DeleteFunc(g.Links, func(l Link) bool { return l.To == to && l.In == in })
	g.Links = append(g.Links, Link{From: w.from, Out: w.out, To: to, In: in})
}

// grid draws the dots of the canvas r, a dot every 24 DIPs of the graph,
// panned and zoomed as it is.
func grid(p *ui.Painter, r ui.Rect, panX, panY, z float32, color ui.Color) {
	step := 24 * z
	if step < 8 {
		return
	}
	mod := func(v float32) float32 { return v - step*float32(math.Floor(float64(v/step))) }
	for x := r.X + mod(panX); x < r.X+r.W; x += step {
		for y := r.Y + mod(panY); y < r.Y+r.H; y += step {
			p.Fill(ui.Rect{X: x - 0.75, Y: y - 0.75, W: 1.5, H: 1.5}, color, 0.75)
		}
	}
}

// curve draws a link from an output at x0, y0 to an input at x1, y1,
// leaving and coming in level.
func curve(p *ui.Painter, x0, y0, x1, y1, width float32, color ui.Color) {
	dx := max(40, abs(x1-x0)/2)
	var path ui.Path
	path.MoveTo(x0, y0).CubeTo(x0+dx, y0, x1-dx, y1, x1, y1)
	p.StrokePath(&path, width, color)
}

func abs(v float32) float32 { return max(v, -v) }
