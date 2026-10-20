// Package dock lays panels out as IDEs do: splits side by side and one
// above another, down to groups of panels behind a bar of tabs. The
// divider between two sides of a split drags to resize them. A click on a
// tab shows its panel; a press in a group, or the keyboard focus in it,
// makes it the active group, whose tab shown has a line in the accent
// color.
//
// Dragging a tab moves its panel: let go over the tabs or the middle of a
// group, it joins the group; near one of the group's four edges, it splits
// the group there; on an outer edge of the dock, it docks across that side
// of the whole dock. While dragging, a translucent area shows where the
// panel would land. A tab closes with its close button or a click of the
// middle button; a group left without panels goes, and its split gives
// its room to the other side.
//
//	root := dock.View(c, &app.dock, &app.layout, func(id string) string {
//		return app.panels[id].title
//	}, func(id string) {
//		app.panels[id].build(c)
//	}).Grow(1)
//	if root.Changed() {
//		app.save()
//	}
//
// The app keeps the layout, a tree of Layout it can save as it is, and
// builds each panel by its id. The dock changes the layout in place as the
// user resizes, moves and closes panels, and reports each change as
// Changed.
package dock

import (
	"slices"

	"github.com/egoist/mygo/ui"
)

// Layout is a node of a dock's tree: a split of First and Second when
// both are set, else a group of panels.
type Layout struct {
	// Panels are the ids of a group's panels, in the order of their tabs,
	// unique in the dock, and Active the index of the one shown.
	Panels []string
	Active int

	// Vertical puts First above Second, rather than beside it, and Percent
	// is the share of the room First takes, from 0 to 100.
	Vertical      bool
	Percent       float32
	First, Second *Layout
}

// State is what a dock keeps: the active group. Keep it in the app's
// state, one for each dock. The zero State makes the first group active.
type State struct {
	// active is the panel the active group shows, and middle the tab the
	// middle button clicked since the last frame, "" for none. focus asks
	// the tab of active to take the keyboard focus, as a tab just dropped.
	active string
	middle string
	focus  bool
}

// Active returns the id of the panel the active group shows, "" when the
// dock has none.
func (s *State) Active() string { return s.active }

// side is where a panel dropped on a group, or on the dock, goes.
type side uint8

const (
	center side = iota
	left
	right
	top
	bottom
)

// edgeShare and outerShare are the percentage of the room a panel docked
// on an edge takes, of a group or of the whole dock.
const edgeShare, outerShare = 50, 25

// middleButton is InputEvent.Button for the middle button of the mouse.
const middleButton = 2

// ref is a panel dragged: its dock, so that docks take only their own
// panels, and its id.
type ref struct {
	dock *State
	id   string
}

// target is where a panel dragged would land: on a side of the group
// node, or with outer of the whole dock, node then its root.
type target struct {
	node  *Layout
	side  side
	outer bool
}

// dock is a View being built.
type dock struct {
	c       *ui.Context
	s       *State
	title   func(id string) string
	content func(id string)
	// dragged is the panel dragged over the dock, or dropped on it, and
	// over whether it is still dragged. to is where it would land, set as
	// the groups see the pointer and read as they draw. closed is the tab
	// whose close button was clicked, and resized whether a divider moved.
	dragged string
	over    bool
	to      target
	closed  string
	resized bool
}

// View creates a dock of the panels of layout, title(id) the title of a
// panel's tab and content(id) building the panel each group shows. Give it
// a size, or grow it in its parent.
func View(c *ui.Context, s *State, layout *Layout, title func(id string) string, content func(id string)) ui.Element {
	t := c.Theme()
	d := &dock{c: c, s: s, title: title, content: content}
	// moved is whether a panel closed or moved, the tree changing.
	moved := false
	if id := s.middle; id != "" {
		s.middle = ""
		moved = s.close(layout, id)
	}
	root := ui.Box(c).Background(t.Background)
	drop, dropped := ui.Drop[ref](root)
	dropped = dropped && drop.dock == s
	drag, over := ui.DragOver[ref](root)
	d.over = over && drag.dock == s
	switch {
	case dropped:
		d.dragged = drop.id
	case d.over:
		d.dragged = drag.id
	}
	// Only while dragging: an element asking where the pointer is gets a
	// frame each time it moves.
	if d.dragged != "" {
		x, y, in := root.PointerPosition()
		b := root.LayoutBounds()
		band := t.Space(3)
		if sd := edge(x, y, b.W, b.H, band, band); in && sd != center {
			d.to = target{node: layout, side: sd, outer: true}
		}
	}
	root.DrawOver(func(p *ui.Painter, r ui.Rect) {
		if d.over && d.to.outer {
			preview(p, t, area(r, d.to.side, outerShare))
		}
	})
	root.Children(func() { d.node(layout) })

	switch {
	case d.closed != "":
		moved = s.close(layout, d.closed) || moved
	case dropped && d.to.node != nil:
		s.move(layout, d.dragged, d.to, c.Direction() == ui.RightToLeft)
		moved = true
	}
	g := find(layout, s.active)
	if g == nil {
		g = first(layout)
	}
	s.active = ""
	if len(g.Panels) > 0 {
		s.active = g.Panels[min(g.Active, len(g.Panels)-1)]
	}
	if moved || d.resized {
		root.SetChanged()
	}
	if moved {
		// Another frame, once this one has laid the tree out, finds the
		// tabs that came under the pointer.
		c.Invalidate()
	}
	return root
}

// node builds n: a split, its sides built in turn, or a group.
func (d *dock) node(n *Layout) {
	if n.First == nil || n.Second == nil {
		d.group(n)
		return
	}
	split := ui.SplitPercent
	if n.Vertical {
		split = ui.SplitVerticalPercent
	}
	if split(d.c, &n.Percent, func() { d.node(n.First) }, func() { d.node(n.Second) }).Fill().Changed() {
		d.resized = true
	}
}

// group builds the group n: its bar of tabs over the panel it shows.
func (d *dock) group(n *Layout) {
	c, s := d.c, d.s
	t := c.Theme()
	g := ui.Column(c).Fill()
	if len(n.Panels) == 0 {
		return
	}
	n.Active = max(0, min(n.Active, len(n.Panels)-1))
	shown := n.Panels[n.Active]
	// Until a tab just dropped takes the focus, FocusWithin may still find
	// it in the elements the tab had before it moved.
	if g.FocusWithin() && !s.focus {
		s.active = shown
	}
	h := t.Space(9)
	if d.dragged != "" && d.to.node == nil {
		if x, y, in := g.PointerPosition(); in {
			b := g.LayoutBounds()
			sd := center
			if y > h {
				sd = edge(x, y-h, b.W, b.H-h, b.W/4, (b.H-h)/4)
			}
			// A group's own panel splits it only when it leaves another
			// behind, and joins it never.
			if !slices.Contains(n.Panels, d.dragged) || sd != center && len(n.Panels) > 1 {
				d.to = target{node: n, side: sd}
			}
		}
	}
	g.DrawOver(func(p *ui.Painter, r ui.Rect) {
		if d.over && d.to.node == n && !d.to.outer {
			preview(p, t, area(r, d.to.side, edgeShare))
		}
	})
	g.Children(func() {
		d.tabs(n, h)
		// Keyed by the panel, a group showing another builds it anew.
		body := ui.Column(c.Key(shown)).Grow(1).Clip()
		// Left to go on, a press anywhere in the panel makes the group
		// active, without taking it from what it presses.
		body.HandleInput(func(e ui.InputEvent) bool {
			if e.Kind == ui.InputPointerDown {
				s.active = shown
			}
			return false
		})
		body.Children(func() { d.content(shown) })
	})
}

// tabs builds the bar of tabs of the group n, h high, which scrolls
// sideways when the group is narrower than its tabs.
func (d *dock) tabs(n *Layout, h float32) {
	c, s := d.c, d.s
	t := c.Theme()
	button := t.Space(5)
	bar := ui.ScrollHorizontal(c).Height(h).Background(t.Surface).Draw(func(p *ui.Painter, r ui.Rect) {
		p.Fill(ui.Rect{X: r.X, Y: r.Y + r.H - 1, W: r.W, H: 1}, t.Border, 0)
	})
	bar.Children(func() {
		parts := ui.TabsBase(c, &n.Active, len(n.Panels))
		parts.List.Height(h).AlignItems(ui.Stretch).Children(func() {
			for i, id := range n.Panels {
				title := d.title(id)
				on := i == n.Active
				w := ui.Row(c.Key(id)).AlignItems(ui.Stretch).BorderEndWidth(1).BorderColor(t.Border).Drag(ref{dock: s, id: id})
				size := w.LayoutBounds()
				_, _, hovered := w.PointerPosition()
				hovered = hovered && d.dragged == ""
				color := t.TextMuted
				if on || hovered {
					color = t.Text
				}
				// A background of its own gives the copy dragged the tab's look.
				w.Background(t.Surface)
				if on {
					w.Background(t.Background)
				}
				if w.Dragging() {
					w.Opacity(0.4)
				}
				w.DrawOver(func(p *ui.Painter, r ui.Rect) {
					switch {
					case !on:
						p.Fill(ui.Rect{X: r.X, Y: r.Y + r.H - 1, W: r.W, H: 1}, t.Border, 0)
					case s.active == id:
						p.Fill(ui.Rect{X: r.X, Y: r.Y, W: r.W, H: 2}, t.Accent, 0)
					}
				})
				w.HandleInput(func(e ui.InputEvent) bool {
					if e.Button != middleButton {
						return false
					}
					if e.Kind == ui.InputPointerUp && e.X >= 0 && e.Y >= 0 && e.X < size.W && e.Y < size.H {
						s.middle = id
					}
					return e.Kind == ui.InputPointerDown || e.Kind == ui.InputPointerUp
				})
				w.Children(func() {
					head := parts.Tab(i).AlignItems(ui.Center).PaddingStart(t.Space(3)).PaddingEnd(button + t.Space(3)).
						TextColor(color).FocusRing(false)
					if s.focus && s.active == id {
						head.Focus()
						s.focus = false
					}
					if head.FocusVisible() {
						head.DrawOver(func(p *ui.Painter, r ui.Rect) {
							p.FocusRing(ui.Rect{X: r.X + 5, Y: r.Y + 5, W: r.W - 10, H: r.H - 10}, [4]float32{t.Radius, t.Radius, t.Radius, t.Radius})
						})
					}
					head.Children(func() { ui.Text(c, title).SingleLine() })

					// Not focusable, the close button keeps the arrows of the
					// tab list on the tabs.
					x := ui.Box(c).Attach(ui.AnchorEnd, ui.AnchorEnd).End(t.Space(1.5)).Size(button, button).Radius(t.Space(1)).
						Role(ui.RoleButton).Label("Close " + title).KeepFocus()
					if x.Clicked() {
						d.closed = id
					}
					_, _, onX := x.PointerPosition()
					switch {
					case hovered && x.Pressed():
						x.Background(t.SurfacePressed)
					case hovered && onX:
						x.Background(t.SurfaceHover)
					}
					x.Draw(func(p *ui.Painter, r ui.Rect) {
						cx, cy, k := r.X+r.W/2, r.Y+r.H/2, r.W/5
						var path ui.Path
						path.MoveTo(cx-k, cy-k).LineTo(cx+k, cy+k).MoveTo(cx+k, cy-k).LineTo(cx-k, cy+k)
						p.StrokePath(&path, 1.5, color)
					})
				})
			}
		})
	})
}

// edge returns the edge of a w by h box nearest to (x, y), measured
// against how far the room of each edge reaches into the box, rx across
// it and ry down it, or center when the point is beyond them all.
func edge(x, y, w, h, rx, ry float32) side {
	sd, near := center, float32(1)
	for _, e := range [...]struct {
		side side
		d    float32
	}{{left, x / rx}, {right, (w - x) / rx}, {top, y / ry}, {bottom, (h - y) / ry}} {
		if e.d < near {
			sd, near = e.side, e.d
		}
	}
	return sd
}

// area returns the part of r a panel docked on side of it takes, share
// percent of it, or all of r for center.
func area(r ui.Rect, sd side, share float32) ui.Rect {
	f := share / 100
	switch sd {
	case left:
		r.W *= f
	case right:
		r.X, r.W = r.X+r.W*(1-f), r.W*f
	case top:
		r.H *= f
	case bottom:
		r.Y, r.H = r.Y+r.H*(1-f), r.H*f
	}
	return r
}

// preview shows r as where a panel dragged would land.
func preview(p *ui.Painter, t *ui.Theme, r ui.Rect) {
	in := t.Space(1)
	r = ui.Rect{X: r.X + in, Y: r.Y + in, W: r.W - 2*in, H: r.H - 2*in}
	p.Fill(r, t.Accent.Alpha(0.16), t.Radius)
	p.Stroke(r, t.Accent, t.Radius, 1.5)
}

// close takes the panel id out of layout and reports whether it was in
// it. A group shown active stays so with the panel shown next.
func (s *State) close(layout *Layout, id string) bool {
	g := find(layout, id)
	if g == nil {
		return false
	}
	remove(g, slices.Index(g.Panels, id))
	if s.active == id && len(g.Panels) > 0 {
		s.active = g.Panels[g.Active]
	}
	prune(layout)
	return true
}

// move takes the panel id from its group to target to, and makes the
// group it lands in active. Right to left, a split's first side is on the
// right.
func (s *State) move(layout *Layout, id string, to target, rtl bool) {
	g := find(layout, id)
	if g == nil {
		return
	}
	remove(g, slices.Index(g.Panels, id))
	n := to.node
	if to.side == center {
		n.Panels = append(n.Panels, id)
		n.Active = len(n.Panels) - 1
	} else {
		share := float32(edgeShare)
		if to.outer {
			share = outerShare
		}
		rest, moved := *n, &Layout{Panels: []string{id}}
		*n = Layout{Vertical: to.side == top || to.side == bottom, Percent: share, First: moved, Second: &rest}
		if (to.side == right || to.side == bottom) != (rtl && !n.Vertical) {
			n.Percent, n.First, n.Second = 100-share, &rest, moved
		}
	}
	prune(layout)
	s.active, s.focus = id, true
}

// remove takes panel i out of the group g, which then shows the same
// panel, or the one after i when it showed i, or the one before when i was
// the last.
func remove(g *Layout, i int) {
	g.Panels = slices.Delete(g.Panels, i, i+1)
	if i < g.Active || g.Active >= len(g.Panels) {
		g.Active = max(g.Active-1, 0)
	}
}

// prune takes the groups left without panels out of n, each split of one
// giving its place to its other side.
func prune(n *Layout) {
	if n.First == nil || n.Second == nil {
		return
	}
	prune(n.First)
	prune(n.Second)
	switch {
	case empty(n.First):
		*n = *n.Second
	case empty(n.Second):
		*n = *n.First
	}
}

// empty reports whether n is a group without panels.
func empty(n *Layout) bool {
	return (n.First == nil || n.Second == nil) && len(n.Panels) == 0
}

// find returns the group of n with the panel id, nil for none.
func find(n *Layout, id string) *Layout {
	if n.First == nil || n.Second == nil {
		if slices.Contains(n.Panels, id) {
			return n
		}
		return nil
	}
	if g := find(n.First, id); g != nil {
		return g
	}
	return find(n.Second, id)
}

// first returns the first group of n, down its First sides.
func first(n *Layout) *Layout {
	for n.First != nil && n.Second != nil {
		n = n.First
	}
	return n
}
