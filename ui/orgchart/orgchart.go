// Package orgchart shows an organization chart, top down: each person a
// card of their avatar, name and title in their team's color, under
// their manager, with elbow lines from each manager to their reports.
// The reports of a manager who have no reports of their own stack one
// under another, as most org charts stack them, to keep the chart
// narrow. The button under a manager's card folds the people under them
// and unfolds them again, and shows how many there are.
//
//	orgchart.View(c, &app.chart, app.people, &app.chosen).Grow(1)
//	if app.chosen >= 0 {
//		... // show app.people[app.chosen]
//	}
//
// The chart opens at its own size, the top person at the middle of the
// top. Dragging pans, as scrolling does, and scrolling with Cmd held,
// Command on macOS and Control elsewhere, zooms around the pointer. A
// double click on the background fits the chart in the view, and zooms
// in there again, to the chart's own size. While the chart has the
// keyboard focus, Cmd+= and Cmd+- zoom in and out, Cmd+0 shows the chart
// at its own size and Cmd+9 fits it, as in zoomview.
//
// A click chooses a person, and the arrows move from them: Up to their
// manager, Down to their first report, unfolding them, and Left and Right
// to the people beside them; in a stack, Up and Down go along the stack.
// The chart pans to keep the person they go to in view. Changed reports
// the choice changing.
package orgchart

import (
	"math"
	"strconv"

	"github.com/egoist/mygo/ui"
)

// Person is a person of a chart: their name, title and team, which gives
// their card its color, and the index of their manager. A person whose
// manager is themselves, or not in the chart, as -1, is at the top.
type Person struct {
	Name, Title, Team string
	Manager           int
}

// maxZoom is how far a chart zooms in: twice its own size.
const maxZoom = 2

// step is how far ZoomIn and ZoomOut zoom: two steps double the size.
var step = float32(math.Sqrt2)

// State is what a chart keeps: the people folded, by index, how many
// DIPs of the view a DIP of the chart takes, and the point of the chart
// at the view's center. The zero State shows the chart at its own size,
// the top person at the middle of the view's top. Keep it in the app's
// state, one for each chart.
type State struct {
	closed map[int]bool
	// zoom is 0 while the chart fits, and before the view first shows
	// it, which started records.
	zoom, x, y float32
	started    bool
	// fit is the zoom fitting the chart in the view, and w, h, vw and vh
	// the sizes of the chart and the view, as of the last frame.
	fit, w, h, vw, vh float32
	// travel is how far the pointer moved since the press began, which
	// keeps a drag from choosing the person it began on.
	travel float32
	// anchor is the person last folded or unfolded, plus 1, and ax, ay
	// where their card was then, to keep it there as the chart changes.
	anchor int
	ax, ay float32
}

// Zoom returns how many DIPs of the view a DIP of the chart takes, as a
// label shows it: 1 at the chart's own size.
func (s *State) Zoom() float32 {
	if s.zoom == 0 {
		return s.fit
	}
	return s.zoom
}

// Fit fits the chart in the view, no larger than its own size, and
// follows the view's size until the user zooms or pans again.
func (s *State) Fit() { s.zoom, s.started = 0, true }

// ZoomIn zooms in a step around the view's center; two steps double the
// size.
func (s *State) ZoomIn() { s.zoomAt(s.Zoom()*step, s.vw/2, s.vh/2) }

// ZoomOut zooms out a step around the view's center.
func (s *State) ZoomOut() { s.zoomAt(s.Zoom()/step, s.vw/2, s.vh/2) }

// zoomAt sets the zoom to z, between the fit and maxZoom, keeping the
// point of the chart at (px, py) of the view where it is.
func (s *State) zoomAt(z, px, py float32) {
	from := s.Zoom()
	if s.zoom == 0 {
		s.x, s.y = s.w/2, s.h/2
	}
	if from <= 0 {
		return
	}
	z = max(s.fit, min(maxZoom, z))
	cx, cy := s.x+(px-s.vw/2)/from, s.y+(py-s.vh/2)/from
	s.zoom, s.x, s.y = z, cx-(px-s.vw/2)/z, cy-(py-s.vh/2)/z
	s.clamp()
}

// pan moves the chart by dx, dy DIPs of the view, and reports whether it
// moved.
func (s *State) pan(dx, dy float32) bool {
	if s.zoom == 0 {
		s.zoom, s.x, s.y = s.fit, s.w/2, s.h/2
	}
	x, y := s.x, s.y
	s.x -= dx / s.zoom
	s.y -= dy / s.zoom
	s.clamp()
	return s.x != x || s.y != y
}

// show pans the least that brings the box x0, y0 to x1, y1 of the chart
// m DIPs inside the view.
func (s *State) show(x0, y0, x1, y1, m float32) {
	if s.zoom == 0 {
		return
	}
	hw, hh := max(0, s.vw/2-m)/s.zoom, max(0, s.vh/2-m)/s.zoom
	s.x = min(max(s.x, x1-hw), x0+hw)
	s.y = min(max(s.y, y1-hh), y0+hh)
	s.clamp()
}

// clamp keeps the chart's edges outside the view, or centers it along an
// axis it does not fill.
func (s *State) clamp() {
	s.x = clampAxis(s.x, s.w, s.vw/s.zoom)
	s.y = clampAxis(s.y, s.h, s.vh/s.zoom)
}

// clampAxis keeps the center c of a span of the view seen showing span
// DIPs of a chart that is size long within it, or at its middle.
func clampAxis(c, size, span float32) float32 {
	if span >= size {
		return size / 2
	}
	return max(span/2, min(size-span/2, c))
}

// layout fits s to a chart w by h in a view vw by vh: the fit's zoom,
// and where the chart is while it fits.
func (s *State) layout(w, h, vw, vh float32) {
	s.w, s.h, s.vw, s.vh = w, h, vw, vh
	s.fit = 1
	if w > 0 && h > 0 && vw > 0 && vh > 0 {
		s.fit = min(1, vw/w, vh/h)
	}
	if s.zoom == 0 {
		s.x, s.y = w/2, h/2
		return
	}
	s.zoom = max(s.fit, s.zoom)
	s.clamp()
}

type node struct {
	parent int
	// slots are the person's reports as they line up: one for each with
	// reports of their own, and one stack of those with none, where the
	// first of them is.
	slots [][]int
	under int
	// w is how wide the person's tree is, off where their card is across
	// it, and x, y the card's top-left corner in the chart.
	w, off, x, y float32
	in, shown    bool
}

// View creates the chart of people, choosing *chosen, an index of people
// or -1 for none. Give it a size, or Grow.
func View(c *ui.Context, s *State, people []Person, chosen *int) ui.Element {
	t := c.Theme()
	if s.closed == nil {
		s.closed = map[int]bool{}
	}
	cardW, cardH, radius := t.Space(56), t.Space(14), t.Space(2)
	gapX, gapY, indent, stackGap, pad := t.Space(6), t.Space(12), t.Space(6), t.Space(3), t.Space(8)

	n := len(people)
	nodes := make([]node, n)
	kids := make([][]int, n)
	var top [][]int
	teams := map[string]int{}
	for i, p := range people {
		nodes[i].parent = p.Manager
		if p.Manager < 0 || p.Manager >= n || p.Manager == i {
			nodes[i].parent = -1
			top = append(top, []int{i})
		} else {
			kids[p.Manager] = append(kids[p.Manager], i)
		}
		if _, ok := teams[p.Team]; !ok {
			teams[p.Team] = len(teams)
		}
	}
	stacked := func(slot []int) bool { return len(kids[slot[0]]) == 0 }
	for i := range nodes {
		at := -1
		for _, j := range kids[i] {
			switch {
			case len(kids[j]) > 0:
				nodes[i].slots = append(nodes[i].slots, []int{j})
			case at < 0:
				at = len(nodes[i].slots)
				nodes[i].slots = append(nodes[i].slots, []int{j})
			default:
				nodes[i].slots[at] = append(nodes[i].slots[at], j)
			}
		}
	}
	// People in a cycle of managers are under no one at the top, and do
	// not show.
	var count func(i int) int
	count = func(i int) int {
		nodes[i].in = true
		for _, j := range kids[i] {
			nodes[i].under += 1 + count(j)
		}
		return nodes[i].under
	}
	for _, slot := range top {
		count(slot[0])
	}

	open := func(i int) bool { return len(nodes[i].slots) > 0 && !s.closed[i] }
	var measure func(i int)
	measure = func(i int) {
		nd := &nodes[i]
		nd.w, nd.off = cardW, 0
		if !open(i) {
			return
		}
		if len(nd.slots) == 1 && stacked(nd.slots[0]) {
			nd.w = indent + cardW
			return
		}
		var x, first, last float32
		for k, slot := range nd.slots {
			if k > 0 {
				x += gapX
			}
			a, w := x+indent/2, indent+cardW
			if !stacked(slot) {
				measure(slot[0])
				a, w = x+nodes[slot[0]].off+cardW/2, nodes[slot[0]].w
			}
			if k == 0 {
				first = a
			}
			last, x = a, x+w
		}
		nd.off = (first+last)/2 - cardW/2
		nd.w = max(x, nd.off+cardW)
	}
	var place func(i int, x, y float32)
	place = func(i int, x, y float32) {
		nd := &nodes[i]
		nd.shown, nd.x, nd.y = true, x+nd.off, y
		if !open(i) {
			return
		}
		y += cardH + gapY
		for _, slot := range nd.slots {
			if !stacked(slot) {
				place(slot[0], x, y)
				x += nodes[slot[0]].w + gapX
				continue
			}
			for k, j := range slot {
				nodes[j].shown, nodes[j].x, nodes[j].y = true, x+indent, y+float32(k)*(cardH+stackGap)
			}
			x += indent + cardW + gapX
		}
	}

	view := ui.Box(c).Clip().Focusable().Role(ui.RoleGroup).Label("Organization chart")
	b := view.LayoutBounds()
	// settle lays the chart out, keeps the card of the person last folded
	// or unfolded where it was, and fits the view to the chart.
	settle := func() {
		var w, h float32
		for i := range nodes {
			nodes[i].shown = false
		}
		for k, slot := range top {
			if k > 0 {
				w += gapX
			}
			measure(slot[0])
			place(slot[0], w, 0)
			w += nodes[slot[0]].w
		}
		for _, nd := range nodes {
			if nd.shown {
				h = max(h, nd.y+cardH)
			}
		}
		if a := s.anchor - 1; a >= 0 && a < n && nodes[a].shown && s.zoom != 0 {
			s.x += nodes[a].x - s.ax
			s.y += nodes[a].y - s.ay
		}
		s.anchor = 0
		if !s.started && b.W > 0 && b.H > 0 && len(top) > 0 {
			s.started, s.zoom = true, 1
			s.x, s.y = pad+nodes[top[0][0]].x+cardW/2, 0
		}
		s.layout(w+2*pad, h+2*pad, b.W, b.H)
	}
	settle()

	if s.w*s.Zoom() > s.vw || s.h*s.Zoom() > s.vh {
		view.Cursor(ui.CursorGrab)
	}
	pressing := false
	if dx, dy, ok := view.Dragged(); ok {
		pressing = true
		if dx != 0 || dy != 0 {
			s.travel += abs(dx) + abs(dy)
			s.pan(dx, dy)
			view.Cursor(ui.CursorGrabbing)
		}
	}
	if view.DoubleClicked() {
		if s.Zoom() > s.fit {
			s.Fit()
		} else {
			px, py, _ := view.PointerPosition()
			s.zoomAt(max(1, 2*s.fit), px, py)
		}
	}
	for _, k := range []struct {
		mods ui.Modifiers
		key  ui.Key
		do   func()
	}{
		{ui.Cmd, ui.KeyEqual, s.ZoomIn},
		{ui.Cmd | ui.Shift, ui.KeyEqual, s.ZoomIn},
		{ui.Cmd, ui.KeyMinus, s.ZoomOut},
		{ui.Cmd, ui.Key0, func() { s.zoomAt(1, s.vw/2, s.vh/2) }},
		{ui.Cmd, ui.Key9, s.Fit},
	} {
		if view.Shortcut(k.mods, k.key) {
			k.do()
		}
	}
	changed := false
	for _, key := range []ui.Key{ui.KeyUp, ui.KeyDown, ui.KeyLeft, ui.KeyRight} {
		if !view.Shortcut(0, key) || len(top) == 0 {
			continue
		}
		cur, next := *chosen, top[0][0]
		if cur >= 0 && cur < n && nodes[cur].in {
			// From one folded away, the arrows go from the folded card
			// that hides them.
			for a := nodes[cur].parent; a >= 0; a = nodes[a].parent {
				if s.closed[a] {
					cur = a
				}
			}
			next = move(nodes, top, cur, key)
			if key == ui.KeyDown && s.closed[cur] && next >= 0 {
				s.closed[cur] = false
				s.anchor, s.ax, s.ay = cur+1, nodes[cur].x, nodes[cur].y
				settle()
			}
		}
		if next >= 0 && next != *chosen {
			*chosen, changed = next, true
			nd := nodes[next]
			s.show(pad+nd.x, pad+nd.y, pad+nd.x+cardW, pad+nd.y+cardH, t.Space(6))
		}
	}

	// Scrolling comes between builds, at the box laid out since, which
	// only a handle reads then.
	ref := ui.Local(view, "orgchart.handle", func() ui.Handle { return ui.Handle{} })
	view.Bind(ref)
	view.HandleInput(func(ev ui.InputEvent) bool {
		if ev.Kind != ui.InputScroll {
			return false
		}
		b := ref.Bounds(c)
		s.vw, s.vh = b.W, b.H
		if ev.Mods&ui.Cmd != 0 {
			// Scrolling 300 DIPs zooms by 2, as zoomview does.
			s.zoomAt(s.Zoom()/float32(math.Exp2(float64(ev.DY)/300)), ev.X, ev.Y)
			return true
		}
		return s.pan(-ev.DX, -ev.DY)
	})

	hue := func(team string) ui.Color {
		// Around the wheel by the golden angle, so that teams that come
		// one after another differ.
		return ui.Oklch(0.62, 0.15, float32((20+teams[team]*137)%360))
	}
	z := s.Zoom()
	line := t.TextMuted.Mix(t.Border, 0.5)
	view.Children(func() {
		chart := ui.Box(c).Absolute().Left(0).Top(0).Size(s.w, s.h).TransformOrigin(0, 0).
			Scale(z, z).Translate(s.vw/2-s.x*z, s.vh/2-s.y*z)
		chart.Draw(func(p *ui.Painter, r ui.Rect) {
			ox, oy, bend, width := r.X+pad, r.Y+pad, t.Space(2), float32(1.5)
			// Paths draw no larger than 4096 pixels, which a chart's lines
			// pass, so the straight parts are lines, opaque where they meet.
			corner := func(x0, y0, cx, cy, x1, y1 float32) {
				var path ui.Path
				path.MoveTo(x0, y0).QuadTo(cx, cy, x1, y1)
				p.StrokePath(&path, width, line)
			}
			// stack draws the line down from y beside a stack of cards,
			// and to each of them.
			stack := func(slot []int, y float32) {
				x, tick := ox+nodes[slot[0]].x-indent/2, indent/2
				end := oy + nodes[slot[len(slot)-1]].y + cardH/2
				for _, j := range slot[:len(slot)-1] {
					mid := oy + nodes[j].y + cardH/2
					p.Line(x, mid, x+tick, mid, width, line)
				}
				p.Line(x, y, x, end-bend, width, line)
				corner(x, end-bend, x, end, x+bend, end)
				p.Line(x+bend, end, x+tick, end, width, line)
			}
			anchor := func(slot []int) float32 {
				if stacked(slot) {
					return ox + nodes[slot[0]].x - indent/2
				}
				return ox + nodes[slot[0]].x + cardW/2
			}
			for i, nd := range nodes {
				if !nd.shown || !open(i) {
					continue
				}
				bottom := oy + nd.y + cardH
				if len(nd.slots) == 1 && stacked(nd.slots[0]) {
					stack(nd.slots[0], bottom)
					continue
				}
				cx, bus, last := ox+nd.x+cardW/2, bottom+gapY/2, len(nd.slots)-1
				p.Line(cx, bottom, cx, bus, width, line)
				if last > 0 {
					x0, x1 := anchor(nd.slots[0]), anchor(nd.slots[last])
					corner(x0, bus+bend, x0, bus, x0+bend, bus)
					p.Line(x0+bend, bus, x1-bend, bus, width, line)
					corner(x1-bend, bus, x1, bus, x1, bus+bend)
				}
				for k, slot := range nd.slots {
					x, y := anchor(slot), bus
					if last > 0 && (k == 0 || k == last) {
						y += bend
					}
					if stacked(slot) {
						stack(slot, y)
					} else {
						p.Line(x, y, x, oy+nodes[slot[0]].y, width, line)
					}
				}
			}
		})
		chart.Children(func() {
			for i, p := range people {
				nd := nodes[i]
				if !nd.shown {
					continue
				}
				color := hue(p.Team)
				card := ui.Box(c.Key(i)).Absolute().Left(pad+nd.x).Top(pad+nd.y).Size(cardW, cardH).
					Radius(radius).Background(t.Background).Border(1, t.Border).
					Shadow(0, 1, 3, 0, ui.RGB(0, 0, 0).Alpha(0.08)).Role(ui.RoleButton).Label(p.Name)
				if card.Hovered() {
					card.Border(1, t.TextMuted)
				}
				if *chosen == i {
					card.Border(2, t.Accent)
				}
				if dx, dy, ok := card.Dragged(); ok {
					pressing = true
					if dx != 0 || dy != 0 {
						s.travel += (abs(dx) + abs(dy)) * z
						s.pan(dx*z, dy*z)
					}
				}
				// A press that moves a little, as a trackpad's may, is
				// still a click.
				if card.Clicked() && s.travel <= 3 && *chosen != i {
					*chosen, changed = i, true
				}
				card.Children(func() {
					ui.Box(c).Absolute().Left(0).Top(0).Right(0).Height(t.Space(0.75)).
						Radius(radius-1, radius-1, 0, 0).Background(color)
					ui.Row(c).Fill().Padding(0, t.Space(3)).Gap(t.Space(2.5)).AlignItems(ui.Center).Children(func() {
						ui.Avatar(c, p.Name, nil).Background(color)
						ui.Column(c).Gap(t.Space(0.5)).Shrink(1).Children(func() {
							ui.Text(c, p.Name).FontWeight(600).SingleLine()
							ui.Text(c, p.Title).FontSize(t.FontSize - 1).TextColor(t.TextMuted).SingleLine()
						})
					})
				})
				if len(nd.slots) > 0 {
					fold(c.Key(-1-i), s, i, nd, color, pad+nd.x+cardW/2, pad+nd.y+cardH)
				}
			}
		})
	})
	if !pressing {
		s.travel = 0
	}
	if changed {
		view.SetChanged()
	}
	return view
}

// move returns the person the arrow key goes to from cur, or -1.
func move(nodes []node, top [][]int, cur int, key ui.Key) int {
	m := nodes[cur].parent
	slots := top
	if m >= 0 {
		slots = nodes[m].slots
	}
	at, k := 0, 0
	for a, slot := range slots {
		for b, j := range slot {
			if j == cur {
				at, k = a, b
			}
		}
	}
	slot := slots[at]
	switch {
	case key == ui.KeyUp && k > 0:
		return slot[k-1]
	case key == ui.KeyUp:
		return m
	case key == ui.KeyDown && k < len(slot)-1:
		return slot[k+1]
	case key == ui.KeyDown && len(nodes[cur].slots) > 0:
		return nodes[cur].slots[0][0]
	case key == ui.KeyLeft && at > 0:
		return slots[at-1][0]
	case key == ui.KeyRight && at < len(slots)-1:
		return slots[at+1][0]
	}
	return -1
}

// fold shows the button at x, y, the middle of the bottom of person i's
// card, that folds the people under them, with how many there are.
func fold(c *ui.Context, s *State, i int, nd node, color ui.Color, x, y float32) {
	t := c.Theme()
	closed := s.closed[i]
	text := strconv.Itoa(nd.under)
	size := t.FontSize - 2
	tw, _ := c.MeasureText(0, ui.Span{Text: text, Size: size, Weight: 600})
	chevron := t.Space(2)
	h := t.Space(5)
	w := tw + chevron + t.Space(4)
	b := ui.ButtonBase(c).Absolute().Left(x-w/2).Top(y-h/2).Size(w, h).Radius(h / 2).
		Justify(ui.Center).AlignItems(ui.Center).Gap(t.Space(1)).Label("Fold")
	if b.Clicked() {
		s.closed[i] = !closed
		s.anchor, s.ax, s.ay = i+1, nd.x, nd.y
	}
	ink := t.TextMuted
	switch {
	case closed:
		b.Label("Unfold").Background(color)
		ink = ui.RGB(255, 255, 255)
	case b.Hovered():
		b.Background(t.SurfaceHover).Border(1, t.Border)
	default:
		b.Background(t.Background).Border(1, t.Border)
	}
	b.Children(func() {
		ui.Text(c, text).FontSize(size).FontWeight(600).FontFeatures("tnum").TextColor(ink)
		ui.Box(c).Size(chevron, chevron).Draw(func(p *ui.Painter, r ui.Rect) {
			// Down while folded, to unfold; up while open, to fold.
			d := r.H / 4
			cy := r.Y + r.H/2
			if closed {
				d = -d
			}
			p.StrokePolyline([][2]float32{{r.X, cy + d}, {r.X + r.W/2, cy - d}, {r.X + r.W, cy + d}}, 1.5, ink)
		})
	})
}

func abs(v float32) float32 { return max(v, -v) }
