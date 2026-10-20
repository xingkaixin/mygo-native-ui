// Package pentool is the pen tool of a vector editor. A click adds a
// corner to the path being drawn, starting one when there is none; a
// press and drag adds a smooth point, pulling its handles out together,
// one alone with Alt held. A click on the first point closes the path,
// and Escape or Enter ends it open. Dragging a point moves it, dragging
// the handles of the point chosen bends its curves, the other handle
// turning along unless Alt breaks them apart. Delete removes the point
// chosen, and a double click makes it a corner or a smooth point again.
// Changed reports an edit; SVG writes the paths as the d of an SVG path.
//
//	pentool.View(c, &app.pen, &app.paths).Grow(1)
//	d := pentool.SVG(app.paths)
package pentool

import (
	"math"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
)

// Anchor is a point of a path, with its handles from it: In bends the
// curve coming to it, Out the one leaving it. A Corner's handles turn
// apart; a smooth point's turn together.
type Anchor struct {
	X, Y       float32
	InX, InY   float32
	OutX, OutY float32
	Corner     bool
}

// Path is a path through anchors, back to the first when Closed.
type Path struct {
	Anchors []Anchor
	Closed  bool
}

type mode uint8

const (
	idle mode = iota
	pulling
	moving
	handleIn
	handleOut
)

// State is what a pen keeps: the path being drawn, the point chosen and
// the drag going on. Keep it in the app's state, one for each canvas.
type State struct {
	// drawing, path and anchor are indexes plus 1, so that the zero
	// State draws and chooses nothing.
	drawing      int
	path, anchor int
	mode         mode
	down, moved  bool
	x, y         float32
	px, py       float32
	ax, ay       float32
}

const reach = 7

func near(x0, y0, x1, y1 float32) bool {
	return math.Hypot(float64(x1-x0), float64(y1-y0)) <= reach
}

func (s *State) chosen(paths []Path) *Anchor {
	if s.path == 0 || s.path > len(paths) || s.anchor == 0 || s.anchor > len(paths[s.path-1].Anchors) {
		return nil
	}
	return &paths[s.path-1].Anchors[s.anchor-1]
}

// press starts what a press at x, y does: bending a handle of the point
// chosen, closing the path drawn, moving a point, or adding one.
func (s *State) press(paths *[]Path, x, y float32) {
	ps := *paths
	s.px, s.py, s.moved = x, y, false
	if a := s.chosen(ps); a != nil {
		switch {
		case (a.InX != 0 || a.InY != 0) && near(a.X+a.InX, a.Y+a.InY, x, y):
			s.mode = handleIn
			return
		case (a.OutX != 0 || a.OutY != 0) && near(a.X+a.OutX, a.Y+a.OutY, x, y):
			s.mode = handleOut
			return
		}
	}
	if s.drawing > 0 {
		p := &ps[s.drawing-1]
		if len(p.Anchors) >= 2 && near(p.Anchors[0].X, p.Anchors[0].Y, x, y) {
			p.Closed = true
			s.path, s.anchor, s.drawing, s.mode = s.drawing, 1, 0, pulling
			return
		}
	}
	for i := len(ps) - 1; i >= 0; i-- {
		for j, a := range ps[i].Anchors {
			if near(a.X, a.Y, x, y) {
				s.path, s.anchor, s.mode = i+1, j+1, moving
				s.ax, s.ay = a.X, a.Y
				return
			}
		}
	}
	if s.drawing == 0 {
		*paths = append(*paths, Path{})
		s.drawing = len(*paths)
	}
	p := &(*paths)[s.drawing-1]
	p.Anchors = append(p.Anchors, Anchor{X: x, Y: y, Corner: true})
	s.path, s.anchor, s.mode = s.drawing, len(p.Anchors), pulling
	s.moved = true
}

// drag carries on what the press started, the pointer at x, y.
func (s *State) drag(paths []Path, x, y float32, alt bool) {
	a := s.chosen(paths)
	if a == nil || s.mode == idle || math.Hypot(float64(x-s.px), float64(y-s.py)) < 3 {
		return
	}
	s.moved = true
	switch s.mode {
	case pulling:
		a.OutX, a.OutY = x-a.X, y-a.Y
		if alt {
			a.Corner = true
		} else {
			a.InX, a.InY, a.Corner = -a.OutX, -a.OutY, false
		}
	case moving:
		a.X, a.Y = s.ax+x-s.px, s.ay+y-s.py
	case handleIn, handleOut:
		hx, hy := x-a.X, y-a.Y
		own, other := [2]*float32{&a.InX, &a.InY}, [2]*float32{&a.OutX, &a.OutY}
		if s.mode == handleOut {
			own, other = other, own
		}
		*own[0], *own[1] = hx, hy
		if alt {
			a.Corner = true
		}
		if !a.Corner {
			l := float32(math.Hypot(float64(hx), float64(hy)))
			o := float32(math.Hypot(float64(*other[0]), float64(*other[1])))
			if l > 0 {
				*other[0], *other[1] = -hx/l*o, -hy/l*o
			}
		}
	}
}

// remove takes out the point chosen, then its path once it has no point.
func (s *State) remove(paths *[]Path) {
	if s.chosen(*paths) == nil {
		return
	}
	p := &(*paths)[s.path-1]
	p.Anchors = append(p.Anchors[:s.anchor-1], p.Anchors[s.anchor:]...)
	if len(p.Anchors) < 3 {
		p.Closed = false
	}
	if len(p.Anchors) == 0 {
		*paths = append((*paths)[:s.path-1], (*paths)[s.path:]...)
		if s.drawing == s.path {
			s.drawing = 0
		} else if s.drawing > s.path {
			s.drawing--
		}
	}
	s.path, s.anchor = 0, 0
}

// toggle makes a smooth point a corner with no handles, and a corner a
// smooth point with handles along its neighbors.
func toggle(p *Path, i int) {
	a := &p.Anchors[i]
	if a.InX != 0 || a.InY != 0 || a.OutX != 0 || a.OutY != 0 {
		a.InX, a.InY, a.OutX, a.OutY, a.Corner = 0, 0, 0, 0, true
		return
	}
	n := len(p.Anchors)
	prev, next := p.Anchors[max(i-1, 0)], p.Anchors[min(i+1, n-1)]
	if p.Closed {
		prev, next = p.Anchors[(i+n-1)%n], p.Anchors[(i+1)%n]
	}
	dx, dy := (next.X-prev.X)/4, (next.Y-prev.Y)/4
	a.InX, a.InY, a.OutX, a.OutY, a.Corner = -dx, -dy, dx, dy, false
}

func trace(out *ui.Path, p Path, x, y float32) {
	as := p.Anchors
	if len(as) == 0 {
		return
	}
	out.MoveTo(x+as[0].X, y+as[0].Y)
	seg := func(a, b Anchor) {
		out.CubeTo(x+a.X+a.OutX, y+a.Y+a.OutY, x+b.X+b.InX, y+b.Y+b.InY, x+b.X, y+b.Y)
	}
	for i := 1; i < len(as); i++ {
		seg(as[i-1], as[i])
	}
	if p.Closed && len(as) > 1 {
		seg(as[len(as)-1], as[0])
		out.Close()
	}
}

// View creates the canvas of *paths.
func View(c *ui.Context, s *State, paths *[]Path) ui.Element {
	t := c.Theme()
	canvas := ui.Box(c).Background(t.Surface).Border(1, t.Border).Radius(t.Space(2)).Clip().
		Focusable().Cursor(ui.CursorCrosshair).Label("Canvas")
	switch {
	case canvas.Shortcut(0, ui.KeyEscape), canvas.Shortcut(0, ui.KeyEnter):
		s.drawing = 0
	case canvas.Shortcut(0, ui.KeyBackspace), canvas.Shortcut(0, ui.KeyDelete):
		if s.chosen(*paths) != nil {
			s.remove(paths)
			canvas.SetChanged()
		}
	}
	x, y, over := canvas.PointerPosition()
	// Pressed ends as the pointer leaves the canvas, but the drag goes on
	// until it is let go, the pointer where its moves take it.
	dx, dy, held := canvas.Dragged()
	switch {
	case canvas.Pressed() && !s.down:
		s.down, s.x, s.y = true, x, y
		s.press(paths, x, y)
	case held && s.down:
		s.x, s.y = s.x+dx, s.y+dy
		s.drag(*paths, s.x, s.y, c.Modifiers()&ui.Alt != 0)
	case !held && s.down:
		s.down = false
		if s.moved || s.mode == pulling {
			canvas.SetChanged()
		}
		s.mode = idle
	}
	if canvas.DoubleClicked() {
		if a := s.chosen(*paths); a != nil && near(a.X, a.Y, x, y) {
			toggle(&(*paths)[s.path-1], s.anchor-1)
			canvas.SetChanged()
		}
	}
	ps := *paths
	canvas.Draw(func(p *ui.Painter, r ui.Rect) {
		for gy := float32(20); gy < r.H; gy += 20 {
			for gx := float32(20); gx < r.W; gx += 20 {
				p.Fill(ui.Rect{X: r.X + gx - 0.75, Y: r.Y + gy - 0.75, W: 1.5, H: 1.5}, t.Text.Alpha(0.12), 0.75)
			}
		}
		for _, path := range ps {
			var pp ui.Path
			trace(&pp, path, r.X, r.Y)
			if path.Closed {
				p.FillPath(&pp, t.Accent.Alpha(0.12))
			}
			p.StrokePath(&pp, 1.5, t.Text)
		}
		if s.drawing > 0 && over && !held {
			as := ps[s.drawing-1].Anchors
			last := as[len(as)-1]
			var band ui.Path
			band.MoveTo(r.X+last.X, r.Y+last.Y).CubeTo(r.X+last.X+last.OutX, r.Y+last.Y+last.OutY, r.X+x, r.Y+y, r.X+x, r.Y+y)
			p.StrokePath(&band, 1, t.Accent.Alpha(0.6))
			if first := as[0]; len(as) >= 2 && near(first.X, first.Y, x, y) {
				p.Stroke(ui.Rect{X: r.X + first.X - 8, Y: r.Y + first.Y - 8, W: 16, H: 16}, t.Accent, 8, 1.5)
			}
		}
		if a := s.chosen(ps); a != nil {
			for _, h := range [][2]float32{{a.InX, a.InY}, {a.OutX, a.OutY}} {
				if h[0] == 0 && h[1] == 0 {
					continue
				}
				hx, hy := r.X+a.X+h[0], r.Y+a.Y+h[1]
				p.StrokePolyline([][2]float32{{r.X + a.X, r.Y + a.Y}, {hx, hy}}, 1, t.Accent)
				knob := ui.Rect{X: hx - 3.5, Y: hy - 3.5, W: 7, H: 7}
				p.Fill(knob, t.Surface, 3.5)
				p.Stroke(knob, t.Accent, 3.5, 1.5)
			}
		}
		for i, path := range ps {
			for j, a := range path.Anchors {
				box := ui.Rect{X: r.X + a.X - 4, Y: r.Y + a.Y - 4, W: 8, H: 8}
				round := float32(1)
				if !a.Corner {
					round = 4
				}
				if s.path == i+1 && s.anchor == j+1 {
					p.Fill(box, t.Accent, round)
				} else {
					p.Fill(box, t.Surface, round)
				}
				p.Stroke(box, t.Accent, round, 1.5)
			}
		}
	})
	return canvas
}

func num(v float32) string {
	return strconv.FormatFloat(math.Round(float64(v)*100)/100, 'f', -1, 64)
}

// SVG writes paths as the d attribute of an SVG path element: a line
// where a segment has no handles, a cubic curve where it has.
func SVG(paths []Path) string {
	var b strings.Builder
	for _, p := range paths {
		as := p.Anchors
		if len(as) == 0 {
			continue
		}
		if b.Len() > 0 {
			b.WriteByte(' ')
		}
		b.WriteString("M" + num(as[0].X) + " " + num(as[0].Y))
		seg := func(a, z Anchor) {
			if a.OutX == 0 && a.OutY == 0 && z.InX == 0 && z.InY == 0 {
				b.WriteString(" L" + num(z.X) + " " + num(z.Y))
				return
			}
			b.WriteString(" C" + num(a.X+a.OutX) + " " + num(a.Y+a.OutY) + " " + num(z.X+z.InX) + " " + num(z.Y+z.InY) + " " + num(z.X) + " " + num(z.Y))
		}
		for i := 1; i < len(as); i++ {
			seg(as[i-1], as[i])
		}
		if p.Closed && len(as) > 1 {
			if last := as[len(as)-1]; last.OutX != 0 || last.OutY != 0 || as[0].InX != 0 || as[0].InY != 0 {
				seg(last, as[0])
			}
			b.WriteString(" Z")
		}
	}
	return b.String()
}
