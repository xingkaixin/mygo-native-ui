// Package whiteboard is an infinite whiteboard, as tldraw and FigJam are:
// sticky notes, rectangles, ellipses, text, arrows and pen strokes on a
// board of dots that zooms and pans without end. The board is the app's
// slice of shapes, which the view edits and the app saves.
//
//	if whiteboard.View(c, &app.board, &app.shapes).Grow(1).Changed() {
//		app.save()
//	}
//
// The wheel zooms around the pointer, as does scrolling with Cmd held,
// Command on macOS and Control elsewhere; scrolling on a touchpad pans,
// as do dragging with the middle button and dragging with Space held.
// The toolbar below chooses a tool, as do keys: Select (V), Sticky note
// (N), Rectangle (R), Ellipse (O), Arrow (A), Pen (P) and Text (T).
// Dragging draws a rectangle or an ellipse, and a click puts one of its
// own size. A sticky note goes where it is let go, and its text, as that
// of text put with a click, is typed at once. An arrow drawn from a shape
// to a shape joins them and follows them as they move; an end let go
// elsewhere stays where it is. The pen draws smoothed strokes.
//
// Select chooses the shape clicked, Shift adds it or takes it away, and
// dragging across the board chooses the shapes it touches. Dragging the
// chosen shapes moves them, lining their edges and centers up with those
// of the other shapes, with guides where they line up; the handles of
// their box resize them, and those of an arrow move its ends, to join
// another shape or none. A double click on a sticky note, a shape or text
// edits its text in place, until Escape or a click elsewhere. Delete or
// Backspace removes the shapes chosen, Cmd+D duplicates them, Cmd+A
// chooses all, Cmd+Z undoes and Shift+Cmd+Z redoes. A color of the
// toolbar colors the sticky notes chosen and those put next. Changed
// reports an edit.
package whiteboard

import (
	"math"
	"reflect"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
)

// Kind is what a shape is.
type Kind uint8

const (
	Sticky Kind = iota
	Rectangle
	Ellipse
	Text
	Arrow
	Pen
)

// Color is the color of a sticky note.
type Color uint8

const (
	Yellow Color = iota
	Orange
	Pink
	Green
	Blue
	Violet
)

var palette = [...]struct {
	name  string
	color ui.Color
}{
	{"Yellow", ui.Oklch(0.92, 0.12, 98)},
	{"Orange", ui.Oklch(0.86, 0.1, 60)},
	{"Pink", ui.Oklch(0.85, 0.09, 355)},
	{"Green", ui.Oklch(0.88, 0.1, 150)},
	{"Blue", ui.Oklch(0.86, 0.07, 240)},
	{"Violet", ui.Oklch(0.84, 0.08, 300)},
}

// ink is the text of sticky notes, which are light in either theme.
var ink = ui.RGB(34, 34, 38)

// Shape is a shape on a board, in the board's DIPs. A sticky note, a
// rectangle or an ellipse fills X, Y, W, H, with its Text in the middle;
// text starts at X, Y, as wide and high as it takes, which the view sets.
// A pen stroke goes through Points, from X, Y, within W by H. An arrow
// points from X, Y to X+W, Y+H, or from and to the edges of the shapes
// whose IDs are From and To, where the view keeps its ends. ID names a
// shape for arrows; the view numbers the shapes without one. Shapes later
// in the slice show above those before them.
type Shape struct {
	ID         int
	Kind       Kind
	X, Y, W, H float32
	Text       string
	Color      Color
	Points     []ui.Point
	From, To   int
}

func svg(body string) *ui.SVG {
	return ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">` + body + `</svg>`))
}

// selecting is the tool that chooses and moves shapes; each other puts
// the kind of shape it names.
const selecting = 0

var tools = []struct {
	name, hint string
	key        ui.Key
	kind       Kind
	icon       *ui.SVG
}{
	{"Select", "V", ui.KeyV, 0, svg(`<path d="M6 3.5l12.5 7.2-5.6 1.4-2.6 5.4z"/>`)},
	{"Sticky note", "N", ui.KeyN, Sticky, svg(`<path d="M5 4.5h14v9.5l-5 5H5z"/><path d="M14 19v-5h5"/>`)},
	{"Rectangle", "R", ui.KeyR, Rectangle, svg(`<rect x="4" y="6" width="16" height="12" rx="2"/>`)},
	{"Ellipse", "O", ui.KeyO, Ellipse, svg(`<ellipse cx="12" cy="12" rx="8.5" ry="6.5"/>`)},
	{"Arrow", "A", ui.KeyA, Arrow, svg(`<path d="M5 19L19 5M10 5h9v9"/>`)},
	{"Pen", "P", ui.KeyP, Pen, svg(`<path d="M4 20l1.2-4.4L16 4.8l3.2 3.2L8.4 18.8z"/><path d="M13.8 7l3.2 3.2"/>`)},
	{"Text", "T", ui.KeyT, Text, svg(`<path d="M5 7V5h14v2M12 5v14M9 19h6"/>`)},
}

const (
	minZoom, maxZoom = 0.1, 8
	// reach is how near the pointer takes a shape, a handle or a line to
	// snap to, in DIPs of the view.
	reach = 6
	// gap is how far an arrow's end stays from the shape it joins.
	gap       = 6
	lineWidth = 2
	penWidth  = 3
)

type gesture uint8

const (
	idle gesture = iota
	panning
	marquee
	moving
	resizing
	creating
	drawing
	pointing
)

// State is what a view keeps: how far it is zoomed and panned, the tool
// and the color of new sticky notes, the shapes chosen and the one whose
// text is edited, what the pointer is doing, and what undo and redo go
// back to. Keep it in the app's state, one for each view.
type State struct {
	// zoom is the DIPs of the view a DIP of the board takes, 0 for 1, and
	// panX, panY where the board's origin is in the view.
	zoom, panX, panY float32
	tool             int
	color            Color
	// chosen holds the IDs of the shapes chosen, and editing that of the
	// shape whose text is typed.
	chosen  []int
	editing int
	text    ui.Handle
	// focus gives the text the focus, and refocus the canvas once it is
	// typed.
	focus, refocus bool
	space, dirty   bool
	gesture        gesture
	moved          bool
	// px, py is where the pointer was in the view as of its last event,
	// and ax, ay the point of the board where the press began.
	px, py, ax, ay float32
	// id is the shape the gesture makes or works on, pick the one a click
	// on shapes chosen chooses alone unless it moves them, and target the
	// shape the arrow end dragged would join.
	id, pick, target int
	// end is the end of the arrow dragged, 0 its start; fx, fy are where
	// the handle dragged is on box, the chosen shapes' bounds as the drag
	// began; and mx, my is the far corner of the marquee from ax, ay.
	end    int
	fx, fy float32
	box    box
	mx, my float32
	// base is what was chosen as the marquee began.
	base   []int
	guides []box
	// before is the board as a gesture began and start as it began to
	// move, after the shape it puts, and edited the board before the text
	// typed.
	before, start, edited []Shape
	undo, redo            [][]Shape
	// layouts are the texts drawn this frame, and old those of the last.
	layouts, old map[layoutKey]*ui.TextLayout
}

type box struct{ x0, y0, x1, y1 float32 }

func (b box) union(o box) box {
	return box{min(b.x0, o.x0), min(b.y0, o.y0), max(b.x1, o.x1), max(b.y1, o.y1)}
}

func (b box) overlaps(o box) bool {
	return b.x0 <= o.x1 && o.x0 <= b.x1 && b.y0 <= o.y1 && o.y0 <= b.y1
}

func (b box) xs() [3]float32 { return [3]float32{b.x0, (b.x0 + b.x1) / 2, b.x1} }
func (b box) ys() [3]float32 { return [3]float32{b.y0, (b.y0 + b.y1) / 2, b.y1} }

func (b box) shift(dx, dy float32) box { return box{b.x0 + dx, b.y0 + dy, b.x1 + dx, b.y1 + dy} }

func bounds(sh Shape) box {
	if sh.Kind == Arrow {
		return box{min(sh.X, sh.X+sh.W), min(sh.Y, sh.Y+sh.H), max(sh.X, sh.X+sh.W), max(sh.Y, sh.Y+sh.H)}
	}
	return box{sh.X, sh.Y, sh.X + sh.W, sh.Y + sh.H}
}

func hypot(x, y float32) float32 { return float32(math.Hypot(float64(x), float64(y))) }

func abs(v float32) float32 { return max(v, -v) }

// look returns how the text of a shape of kind k shows.
func look(k Kind) (size float32, weight int, align ui.Align) {
	switch k {
	case Sticky:
		return 16, 400, ui.Center
	case Text:
		return 24, 600, ui.Start
	}
	return 18, 500, ui.Center
}

// wrap returns where the lines of a shape's text start on the board, and
// the width they wrap at, 0 for none.
func wrap(sh Shape) (x, w float32) {
	switch sh.Kind {
	case Text:
		return sh.X, 0
	case Ellipse:
		// The widest box inside an ellipse is √½ of its width.
		in := sh.W * 0.15
		return sh.X + in, sh.W - 2*in
	case Sticky:
		return sh.X + 14, sh.W - 28
	}
	return sh.X + 12, sh.W - 24
}

// size returns the size a click puts a shape of kind k in.
func size(k Kind) (w, h float32) {
	switch k {
	case Sticky:
		return 160, 160
	case Ellipse:
		return 150, 110
	}
	return 180, 110
}

// view maps the board to the view: a point x, y of the board shows at
// ox+x*z, oy+y*z.
type view struct{ ox, oy, z float32 }

func (v view) at(x, y float32) (float32, float32) { return v.ox + x*v.z, v.oy + y*v.z }

func (v view) rect(b box) ui.Rect {
	x, y := v.at(b.x0, b.y0)
	return ui.Rect{X: x, Y: y, W: (b.x1 - b.x0) * v.z, H: (b.y1 - b.y0) * v.z}
}

func (s *State) scale() float32 {
	if s.zoom == 0 {
		return 1
	}
	return s.zoom
}

func (s *State) board(x, y float32) (float32, float32) {
	z := s.scale()
	return (x - s.panX) / z, (y - s.panY) / z
}

// zoomAt zooms to z keeping the point of the board at x, y of the view
// where it is.
func (s *State) zoomAt(z, x, y float32) {
	old := s.scale()
	z = max(minZoom, min(maxZoom, z))
	s.panX = x - (x-s.panX)*z/old
	s.panY = y - (y-s.panY)*z/old
	s.zoom = z
}

func (s *State) isChosen(id int) bool { return id != 0 && slices.Contains(s.chosen, id) }

func find(shapes []Shape, id int) int {
	if id == 0 {
		return -1
	}
	return slices.IndexFunc(shapes, func(sh Shape) bool { return sh.ID == id })
}

func nextID(shapes []Shape) int {
	n := 1
	for _, sh := range shapes {
		n = max(n, sh.ID+1)
	}
	return n
}

func same(a, b []Shape) bool { return len(a) == len(b) && (len(a) == 0 || reflect.DeepEqual(a, b)) }

// commit makes what changed since before a step of undo. The view never
// changes Points in place, so a shallow copy of the board keeps it.
func (s *State) commit(shapes, before []Shape) {
	if before == nil || same(before, shapes) {
		return
	}
	s.undo = append(s.undo, before)
	s.redo = nil
	s.dirty = true
}

func (s *State) restore(shapes *[]Shape, from, to *[][]Shape) {
	if len(*from) == 0 || s.gesture != idle {
		return
	}
	*to = append(*to, slices.Clone(*shapes))
	*shapes = (*from)[len(*from)-1]
	*from = (*from)[:len(*from)-1]
	s.chosen = slices.DeleteFunc(s.chosen, func(id int) bool { return find(*shapes, id) < 0 })
	s.dirty = true
}

// tidy gives the shapes without an ID one, sizes text to what it holds,
// and puts the joined ends of arrows on the edges of their shapes.
func tidy(c *ui.Context, shapes []Shape) {
	next := nextID(shapes)
	for i := range shapes {
		sh := &shapes[i]
		if sh.ID == 0 {
			sh.ID, next = next, next+1
		}
		if sh.Kind == Text {
			size, weight, _ := look(Text)
			w, h := c.MeasureText(0, ui.Span{Text: sh.Text, Size: size, Weight: weight})
			_, line := c.MeasureText(0, ui.Span{Text: "M", Size: size, Weight: weight})
			sh.W, sh.H = max(w, size/2), max(h, line)
		}
	}
	center := func(id int) (float32, float32, int) {
		i := find(shapes, id)
		if i < 0 || shapes[i].Kind == Arrow || shapes[i].Kind == Pen {
			return 0, 0, -1
		}
		sh := shapes[i]
		return sh.X + sh.W/2, sh.Y + sh.H/2, i
	}
	for i := range shapes {
		a := &shapes[i]
		if a.Kind != Arrow {
			continue
		}
		x0, y0, x1, y1 := a.X, a.Y, a.X+a.W, a.Y+a.H
		fx, fy, from := center(a.From)
		tx, ty, to := center(a.To)
		if from >= 0 {
			if to < 0 {
				tx, ty = x1, y1
			}
			x0, y0 = edge(shapes[from], tx, ty)
		}
		if to >= 0 {
			if from < 0 {
				fx, fy = x0, y0
			}
			x1, y1 = edge(shapes[to], fx, fy)
		}
		a.X, a.Y, a.W, a.H = x0, y0, x1-x0, y1-y0
	}
}

// edge returns where the line from the center of sh to x, y leaves it, a
// gap beyond its edge.
func edge(sh Shape, x, y float32) (float32, float32) {
	cx, cy := sh.X+sh.W/2, sh.Y+sh.H/2
	dx, dy := x-cx, y-cy
	l := hypot(dx, dy)
	if l == 0 {
		return cx, cy
	}
	rx, ry := sh.W/2, sh.H/2
	var k float32
	if sh.Kind == Ellipse {
		k = 1 / hypot(dx/max(rx, 1e-3), dy/max(ry, 1e-3))
	} else {
		k = min(rx/max(abs(dx), 1e-6), ry/max(abs(dy), 1e-6))
	}
	return cx + dx*k + dx/l*gap, cy + dy*k + dy/l*gap
}

// line returns the points of an arrow or a pen stroke on the board.
func line(sh Shape) []ui.Point {
	if sh.Kind == Arrow {
		return []ui.Point{{X: sh.X, Y: sh.Y}, {X: sh.X + sh.W, Y: sh.Y + sh.H}}
	}
	pts := make([]ui.Point, len(sh.Points))
	for i, p := range sh.Points {
		pts[i] = ui.Point{X: sh.X + p.X, Y: sh.Y + p.Y}
	}
	return pts
}

func segment(x, y float32, a, b ui.Point) float32 {
	dx, dy := b.X-a.X, b.Y-a.Y
	u := float32(0)
	if l2 := dx*dx + dy*dy; l2 > 0 {
		u = max(0, min(1, ((x-a.X)*dx+(y-a.Y)*dy)/l2))
	}
	return hypot(x-a.X-u*dx, y-a.Y-u*dy)
}

// hit returns the index of the topmost shape within tol of x, y of the
// board, -1 for none; joinable leaves out arrows and pen strokes, which
// arrows do not join.
func hit(shapes []Shape, x, y, tol float32, joinable bool) int {
	for i := len(shapes) - 1; i >= 0; i-- {
		sh := shapes[i]
		switch sh.Kind {
		case Arrow, Pen:
			if joinable {
				continue
			}
			pts := line(sh)
			for k := range pts {
				if segment(x, y, pts[max(k-1, 0)], pts[k]) <= tol+penWidth/2 {
					return i
				}
			}
		case Ellipse:
			dx, dy := (x-sh.X-sh.W/2)/(sh.W/2+tol), (y-sh.Y-sh.H/2)/(sh.H/2+tol)
			if dx*dx+dy*dy <= 1 {
				return i
			}
		default:
			if x >= sh.X-tol && x <= sh.X+sh.W+tol && y >= sh.Y-tol && y <= sh.Y+sh.H+tol {
				return i
			}
		}
	}
	return -1
}

// grips are the handles of a box, as fractions of its width and height.
var grips = [8][2]float32{{0, 0}, {0.5, 0}, {1, 0}, {1, 0.5}, {1, 1}, {0.5, 1}, {0, 1}, {0, 0.5}}

// lone returns the index of the arrow chosen, when it is the only shape
// chosen.
func (s *State) lone(shapes []Shape) (int, bool) {
	if len(s.chosen) != 1 {
		return -1, false
	}
	i := find(shapes, s.chosen[0])
	return i, i >= 0 && shapes[i].Kind == Arrow
}

// frame returns the bounds of the shapes chosen, and whether its handles
// show: not for a lone arrow, which has its own, nor for text alone,
// which sizes itself.
func (s *State) frame(shapes []Shape) (box, bool) {
	var b box
	n, sized := 0, false
	for _, sh := range shapes {
		if !s.isChosen(sh.ID) {
			continue
		}
		if n == 0 {
			b = bounds(sh)
		} else {
			b = b.union(bounds(sh))
		}
		n++
		sized = sized || sh.Kind != Text
	}
	_, arrow := s.lone(shapes)
	return b, n > 0 && sized && !arrow
}

// grab returns the handle of the chosen shapes at x, y of the view: an end
// of a lone arrow, or one of their box, at fx, fy of it, with end -1.
func (s *State) grab(shapes []Shape, x, y float32) (end int, fx, fy float32, ok bool) {
	v := view{s.panX, s.panY, s.scale()}
	if i, ok := s.lone(shapes); ok {
		for end, p := range line(shapes[i]) {
			if px, py := v.at(p.X, p.Y); hypot(px-x, py-y) <= reach {
				return end, 0, 0, true
			}
		}
		return 0, 0, 0, false
	}
	b, ok := s.frame(shapes)
	if !ok {
		return 0, 0, 0, false
	}
	for _, g := range grips {
		gx, gy := v.at(b.x0+(b.x1-b.x0)*g[0], b.y0+(b.y1-b.y0)*g[1])
		if abs(gx-x) <= reach && abs(gy-y) <= reach {
			return -1, g[0], g[1], true
		}
	}
	return 0, 0, 0, false
}

// input takes the canvas's pointer, wheel and Space as they come, so that
// a pen stroke gets every sample of the pointer and not one a frame.
func (s *State) input(shapes *[]Shape, ev ui.InputEvent) bool {
	switch ev.Kind {
	case ui.InputKeyDown, ui.InputKeyUp:
		if ev.Key != ui.KeySpace {
			return false
		}
		s.space = ev.Kind == ui.InputKeyDown
	case ui.InputScroll:
		if ev.Precise && ev.Mods&ui.Cmd == 0 {
			s.panX, s.panY = s.panX-ev.DX, s.panY-ev.DY
			return true
		}
		// Scrolling 200 DIPs zooms by 2.
		s.zoomAt(s.scale()*float32(math.Exp2(float64(-ev.DY)/200)), ev.X, ev.Y)
	case ui.InputPointerDown:
		if s.gesture != idle {
			return true
		}
		s.px, s.py, s.moved = ev.X, ev.Y, false
		if ev.Button == 2 || ev.Button == 0 && s.space {
			s.gesture = panning
			return true
		}
		if ev.Button != 0 {
			return false
		}
		s.press(shapes, ev.X, ev.Y, ev.Mods&ui.Shift != 0, ev.Clicks)
	case ui.InputPointerMove:
		if s.gesture == idle {
			return false
		}
		s.drag(*shapes, ev.X, ev.Y)
	case ui.InputPointerUp:
		if s.gesture == idle {
			return false
		}
		s.release(shapes)
	default:
		return false
	}
	return true
}

// press starts what a press at x, y of the view does with the tool.
func (s *State) press(shapes *[]Shape, x, y float32, shift bool, clicks int) {
	s.finish(shapes)
	z := s.scale()
	bx, by := s.board(x, y)
	s.ax, s.ay, s.pick = bx, by, 0
	s.before = slices.Clone(*shapes)
	s.start = s.before
	tol := reach / z
	put := func(sh Shape) {
		sh.ID = nextID(*shapes)
		*shapes = append(*shapes, sh)
		s.id, s.chosen = sh.ID, []int{sh.ID}
	}
	switch kind := tools[s.tool].kind; {
	case s.tool == selecting:
		i := hit(*shapes, bx, by, tol, false)
		if clicks == 2 && i >= 0 && (*shapes)[i].Kind <= Text {
			s.edit((*shapes)[i].ID, *shapes)
			return
		}
		if end, fx, fy, ok := s.grab(*shapes, x, y); ok {
			if end >= 0 {
				s.gesture, s.id, s.end = pointing, s.chosen[0], end
				return
			}
			s.gesture, s.fx, s.fy = resizing, fx, fy
			s.box, _ = s.frame(*shapes)
			return
		}
		if i < 0 {
			if !shift {
				s.chosen = nil
			}
			s.gesture, s.base, s.mx, s.my = marquee, slices.Clone(s.chosen), bx, by
			return
		}
		id := (*shapes)[i].ID
		switch {
		case shift && s.isChosen(id):
			s.chosen = slices.DeleteFunc(s.chosen, func(c int) bool { return c == id })
			return
		case shift:
			s.chosen = append(s.chosen, id)
		case s.isChosen(id):
			s.pick = id
		default:
			s.chosen = []int{id}
		}
		s.gesture = moving
	case kind == Sticky:
		w, h := size(Sticky)
		put(Shape{Kind: Sticky, X: bx - w/2, Y: by - h/2, W: w, H: h, Color: s.color})
		s.start, s.gesture = slices.Clone(*shapes), moving
	case kind == Text:
		size, _, _ := look(Text)
		put(Shape{Kind: Text, X: bx, Y: by - size*0.65})
		s.tool = selecting
		s.edit(s.id, s.before)
		s.before = nil
	case kind == Pen:
		put(Shape{Kind: Pen, X: bx, Y: by, Points: []ui.Point{{}}})
		s.gesture = drawing
	case kind == Arrow:
		from := 0
		if i := hit(*shapes, bx, by, tol, true); i >= 0 {
			from = (*shapes)[i].ID
		}
		put(Shape{Kind: Arrow, X: bx, Y: by, From: from})
		s.gesture, s.end = pointing, 1
	default:
		put(Shape{Kind: kind, X: bx, Y: by})
		s.gesture = creating
	}
}

// drag carries on what the press started, the pointer at x, y of the
// view.
func (s *State) drag(shapes []Shape, x, y float32) {
	z := s.scale()
	dx, dy := x-s.px, y-s.py
	s.px, s.py = x, y
	if s.gesture == panning {
		s.panX, s.panY = s.panX+dx, s.panY+dy
		return
	}
	bx, by := s.board(x, y)
	if !s.moved {
		vx, vy := view{s.panX, s.panY, z}.at(s.ax, s.ay)
		// A press shaking by a DIP or two is still a click.
		if s.gesture != drawing && hypot(x-vx, y-vy) < 3 {
			return
		}
		s.moved = true
	}
	i := find(shapes, s.id)
	switch s.gesture {
	case marquee:
		s.mx, s.my = bx, by
		m := box{min(s.ax, bx), min(s.ay, by), max(s.ax, bx), max(s.ay, by)}
		s.chosen = slices.Clone(s.base)
		for _, sh := range shapes {
			if !s.isChosen(sh.ID) && bounds(sh).overlaps(m) {
				s.chosen = append(s.chosen, sh.ID)
			}
		}
	case moving:
		s.move(shapes, bx-s.ax, by-s.ay)
	case resizing:
		s.resize(shapes, bx-s.ax, by-s.ay)
	case creating:
		if i >= 0 {
			sh := &shapes[i]
			sh.X, sh.Y, sh.W, sh.H = min(s.ax, bx), min(s.ay, by), abs(bx-s.ax), abs(by-s.ay)
		}
	case drawing:
		if i < 0 {
			return
		}
		sh := &shapes[i]
		last := sh.Points[len(sh.Points)-1]
		px, py := bx-sh.X, by-sh.Y
		if hypot(px-last.X, py-last.Y) < 2/z {
			return
		}
		// Each point leans to the one before, which takes out the shake
		// of a hand.
		sh.Points = append(sh.Points, ui.Point{X: last.X + (px-last.X)*0.6, Y: last.Y + (py-last.Y)*0.6})
	case pointing:
		if i < 0 {
			return
		}
		a := &shapes[i]
		other := a.From
		if s.end == 0 {
			other = a.To
		}
		s.target = 0
		if j := hit(shapes, bx, by, reach/z, true); j >= 0 && shapes[j].ID != other {
			s.target = shapes[j].ID
		}
		if s.end == 0 {
			ex, ey := a.X+a.W, a.Y+a.H
			a.X, a.Y, a.W, a.H, a.From = bx, by, ex-bx, ey-by, s.target
		} else {
			a.W, a.H, a.To = bx-a.X, by-a.Y, s.target
		}
	}
}

// snap returns how far to move the lines at further to line one of them
// up with one of lines, within tol.
func snap(at [3]float32, lines []float32, tol float32) float32 {
	best, d := tol, float32(0)
	for _, a := range at {
		for _, l := range lines {
			if v := l - a; abs(v) < best {
				best, d = abs(v), v
			}
		}
	}
	return d
}

// move moves the shapes chosen by dx, dy from where they were, snapping
// their box to the edges and centers of the other shapes, and keeps the
// guides of where it lines up. An arrow's joined ends stay with their
// shapes.
func (s *State) move(shapes []Shape, dx, dy float32) {
	var m box
	var others []box
	var xs, ys []float32
	moving := false
	for _, sh := range s.start {
		b := bounds(sh)
		switch {
		case s.isChosen(sh.ID) && sh.Kind != Arrow:
			if moving {
				m = m.union(b)
			} else {
				m = b
			}
			moving = true
		case !s.isChosen(sh.ID) && sh.Kind != Arrow:
			others = append(others, b)
			x, y := b.xs(), b.ys()
			xs, ys = append(xs, x[:]...), append(ys, y[:]...)
		}
	}
	s.guides = s.guides[:0]
	if moving {
		tol := reach / s.scale()
		dx += snap(m.shift(dx, dy).xs(), xs, tol)
		dy += snap(m.shift(dx, dy).ys(), ys, tol)
		m = m.shift(dx, dy)
		for _, o := range others {
			for _, a := range m.xs() {
				for _, b := range o.xs() {
					if abs(a-b) < 0.01 {
						s.guides = append(s.guides, box{a, min(m.y0, o.y0), a, max(m.y1, o.y1)})
					}
				}
			}
			for _, a := range m.ys() {
				for _, b := range o.ys() {
					if abs(a-b) < 0.01 {
						s.guides = append(s.guides, box{min(m.x0, o.x0), a, max(m.x1, o.x1), a})
					}
				}
			}
		}
	}
	for i, sh := range s.start {
		if i >= len(shapes) || !s.isChosen(sh.ID) {
			continue
		}
		n := &shapes[i]
		switch {
		case sh.Kind != Arrow, sh.From == 0 && sh.To == 0:
			n.X, n.Y = sh.X+dx, sh.Y+dy
		case sh.From == 0:
			n.X, n.Y, n.W, n.H = sh.X+dx, sh.Y+dy, sh.W-dx, sh.H-dy
		case sh.To == 0:
			n.W, n.H = sh.W+dx, sh.H+dy
		}
	}
}

// resize moves the side or corner of the chosen shapes' box at s.fx, s.fy
// by dx, dy, and scales the shapes with the box.
func (s *State) resize(shapes []Shape, dx, dy float32) {
	const least = 8
	b, n := s.box, s.box
	switch s.fx {
	case 0:
		n.x0 = min(b.x0+dx, b.x1-least)
	case 1:
		n.x1 = max(b.x1+dx, b.x0+least)
	}
	switch s.fy {
	case 0:
		n.y0 = min(b.y0+dy, b.y1-least)
	case 1:
		n.y1 = max(b.y1+dy, b.y0+least)
	}
	ratio := func(a, b float32) float32 {
		if b == 0 {
			return 1
		}
		return a / b
	}
	kx, ky := ratio(n.x1-n.x0, b.x1-b.x0), ratio(n.y1-n.y0, b.y1-b.y0)
	at := func(x, y float32) (float32, float32) { return n.x0 + (x-b.x0)*kx, n.y0 + (y-b.y0)*ky }
	for i, sh := range s.start {
		if i >= len(shapes) || !s.isChosen(sh.ID) {
			continue
		}
		m := sh
		m.X, m.Y = at(sh.X, sh.Y)
		switch sh.Kind {
		case Text:
		case Arrow:
			ex, ey := at(sh.X+sh.W, sh.Y+sh.H)
			m.W, m.H = ex-m.X, ey-m.Y
		default:
			m.W, m.H = sh.W*kx, sh.H*ky
			if sh.Kind == Pen {
				m.Points = make([]ui.Point, len(sh.Points))
				for k, p := range sh.Points {
					m.Points[k] = ui.Point{X: p.X * kx, Y: p.Y * ky}
				}
			}
		}
		shapes[i] = m
	}
}

// release ends the gesture, and makes what it changed a step of undo.
func (s *State) release(shapes *[]Shape) {
	g := s.gesture
	s.gesture, s.guides, s.target = idle, nil, 0
	z := s.scale()
	i := find(*shapes, s.id)
	switch {
	case g == moving && !s.moved && s.pick != 0:
		s.chosen = []int{s.pick}
	case g == creating && i >= 0:
		if sh := &(*shapes)[i]; max(sh.W, sh.H)*z < 4 {
			sh.W, sh.H = size(sh.Kind)
			sh.X, sh.Y = s.ax-sh.W/2, s.ay-sh.H/2
		}
		s.tool = selecting
	case g == drawing && i >= 0:
		sh := &(*shapes)[i]
		bx, by := s.board(s.px, s.py)
		sh.Points = append(sh.Points, ui.Point{X: bx - sh.X, Y: by - sh.Y})
		fit(sh)
	case g == pointing && i >= 0 && len(*shapes) > len(s.before):
		if a := (*shapes)[i]; hypot(a.W, a.H)*z < 8 {
			*shapes = slices.Delete(*shapes, i, i+1)
			s.chosen = nil
		} else {
			s.tool = selecting
		}
	}
	s.commit(*shapes, s.before)
	if g == moving && i >= 0 && len(*shapes) > len(s.before) {
		s.tool = selecting
		s.edit(s.id, *shapes)
	}
	s.before, s.start, s.id = nil, nil, 0
}

// fit moves a pen stroke's X, Y to the top left of its points, and its W,
// H to their size.
func fit(sh *Shape) {
	x0, y0 := float32(math.Inf(1)), float32(math.Inf(1))
	x1, y1 := float32(math.Inf(-1)), float32(math.Inf(-1))
	for _, p := range sh.Points {
		x0, y0, x1, y1 = min(x0, p.X), min(y0, p.Y), max(x1, p.X), max(y1, p.Y)
	}
	for k := range sh.Points {
		sh.Points[k].X -= x0
		sh.Points[k].Y -= y0
	}
	sh.X, sh.Y, sh.W, sh.H = sh.X+x0, sh.Y+y0, x1-x0, y1-y0
}

// edit starts typing the text of shape id, which commits as one step of
// undo from base.
func (s *State) edit(id int, base []Shape) {
	s.editing, s.focus, s.chosen = id, true, []int{id}
	s.edited = slices.Clone(base)
}

// finish ends typing, removing text left empty.
func (s *State) finish(shapes *[]Shape) {
	if s.editing == 0 {
		return
	}
	if i := find(*shapes, s.editing); i >= 0 && (*shapes)[i].Kind == Text && strings.TrimSpace((*shapes)[i].Text) == "" {
		s.chosen = []int{s.editing}
		s.drop(shapes)
	}
	s.editing = 0
	s.commit(*shapes, s.edited)
	s.edited = nil
}

// drop removes the shapes chosen, leaving the ends of the arrows joined to
// them where they are.
func (s *State) drop(shapes *[]Shape) {
	*shapes = slices.DeleteFunc(*shapes, func(sh Shape) bool { return s.isChosen(sh.ID) })
	for i := range *shapes {
		a := &(*shapes)[i]
		if s.isChosen(a.From) {
			a.From = 0
		}
		if s.isChosen(a.To) {
			a.To = 0
		}
	}
	s.chosen = nil
}

// duplicate adds copies of the shapes chosen a little below and to the
// right, the copies of arrows joining the copies of their shapes, and
// chooses them.
func (s *State) duplicate(shapes *[]Shape) {
	before := slices.Clone(*shapes)
	next := nextID(before)
	ids := map[int]int{}
	var chosen []int
	for _, sh := range before {
		if s.isChosen(sh.ID) {
			ids[sh.ID] = next
			chosen = append(chosen, next)
			next++
		}
	}
	for _, sh := range before {
		id, ok := ids[sh.ID]
		if !ok {
			continue
		}
		sh.ID, sh.X, sh.Y = id, sh.X+20, sh.Y+20
		if to, ok := ids[sh.From]; ok {
			sh.From = to
		}
		if to, ok := ids[sh.To]; ok {
			sh.To = to
		}
		*shapes = append(*shapes, sh)
	}
	s.chosen = chosen
	s.commit(*shapes, before)
}

// keys does what the keys do while the canvas has the focus and no text
// is typed.
func (s *State) keys(root ui.Element, shapes *[]Shape) {
	for i, tl := range tools {
		if root.Shortcut(0, tl.key) {
			s.tool = i
		}
	}
	if root.Shortcut(0, ui.KeyEscape) {
		s.chosen, s.tool = nil, selecting
	}
	del := root.Shortcut(0, ui.KeyDelete)
	if root.Shortcut(0, ui.KeyBackspace) || del {
		before := slices.Clone(*shapes)
		s.drop(shapes)
		s.commit(*shapes, before)
	}
	if root.Shortcut(ui.Cmd, ui.KeyD) {
		s.duplicate(shapes)
	}
	if root.Shortcut(ui.Cmd, ui.KeyA) {
		s.chosen = s.chosen[:0]
		for _, sh := range *shapes {
			s.chosen = append(s.chosen, sh.ID)
		}
	}
	if root.Shortcut(ui.Cmd, ui.KeyZ) {
		s.restore(shapes, &s.undo, &s.redo)
	}
	if root.Shortcut(ui.Cmd|ui.Shift, ui.KeyZ) {
		s.restore(shapes, &s.redo, &s.undo)
	}
}

// View creates the board of *shapes, with its toolbar. Give it a size, or
// Grow. Changed reports an edit.
func View(c *ui.Context, s *State, shapes *[]Shape) ui.Element {
	tidy(c, *shapes)
	s.old, s.layouts = s.layouts, nil
	root := ui.Box(c).Clip()
	if s.editing == 0 {
		s.keys(root, shapes)
	}
	root.Children(func() {
		s.canvas(c, shapes)
		s.toolbar(c, shapes)
	})
	if s.dirty {
		root.SetChanged()
		s.dirty = false
	}
	return root
}

// canvas shows the board, and takes the pointer on it.
func (s *State) canvas(c *ui.Context, shapes *[]Shape) {
	t := c.Theme()
	canvas := ui.Box(c).Fill().Clip().Focusable().AutoFocus().Role(ui.RoleGroup).Label("Whiteboard").
		Background(t.Background)
	canvas.HandleInput(func(ev ui.InputEvent) bool { return s.input(shapes, ev) })
	if s.refocus {
		canvas.Focus()
		s.refocus = false
	}
	if !canvas.Focused() {
		// The key going up goes to what has the focus now.
		s.space = false
	}
	px, py, over := canvas.PointerPosition()
	switch {
	case s.gesture == panning:
		canvas.Cursor(ui.CursorGrabbing)
	case s.space:
		canvas.Cursor(ui.CursorGrab)
	case tools[s.tool].kind == Text:
		canvas.Cursor(ui.CursorText)
	case s.tool != selecting:
		canvas.Cursor(ui.CursorCrosshair)
	case over:
		if end, fx, fy, ok := s.grab(*shapes, px, py); ok && end < 0 {
			switch {
			case fx == 0.5:
				canvas.Cursor(ui.CursorResizeNS)
			case fy == 0.5:
				canvas.Cursor(ui.CursorResizeEW)
			case fx == fy:
				canvas.Cursor(ui.CursorResizeNWSE)
			default:
				canvas.Cursor(ui.CursorResizeNESW)
			}
		}
	}

	list := slices.Clone(*shapes)
	chosen := slices.Clone(s.chosen)
	guides := slices.Clone(s.guides)
	frame, handles := s.frame(list)
	lone, arrow := s.lone(list)
	editing, target := s.editing, s.target
	mark := s.gesture == marquee && s.moved
	m := box{min(s.ax, s.mx), min(s.ay, s.my), max(s.ax, s.mx), max(s.ay, s.my)}
	panX, panY, z := s.panX, s.panY, s.scale()
	canvas.Draw(func(p *ui.Painter, r ui.Rect) {
		v := view{r.X + panX, r.Y + panY, z}
		// Clip clips the children, not what Draw paints.
		p.Clip(r, 0, func() {
			grid(p, r, v, t.Text.Alpha(0.16))
			for _, sh := range list {
				color := t.Text
				if sh.Kind == Arrow && slices.Contains(chosen, sh.ID) {
					color = t.Accent
				}
				s.paint(p, t, v, sh, color, sh.ID != editing)
			}
			if i := find(list, target); i >= 0 {
				b := bounds(list[i])
				p.Stroke(v.rect(box{b.x0 - 4/z, b.y0 - 4/z, b.x1 + 4/z, b.y1 + 4/z}), t.Accent, 6, 2)
			}
			for _, sh := range list {
				if sh.Kind != Arrow && slices.Contains(chosen, sh.ID) {
					p.Stroke(v.rect(bounds(sh)), t.Accent, 0, 1.5)
				}
			}
			if handles {
				f := v.rect(frame)
				if len(chosen) > 1 {
					p.Stroke(f, t.Accent, 0, 1)
				}
				for _, g := range grips {
					k := ui.Rect{X: f.X + f.W*g[0] - 4, Y: f.Y + f.H*g[1] - 4, W: 8, H: 8}
					p.Fill(k, t.Background, 1.5)
					p.Stroke(k, t.Accent, 1.5, 1.5)
				}
			}
			if arrow {
				for _, pt := range line(list[lone]) {
					x, y := v.at(pt.X, pt.Y)
					k := ui.Rect{X: x - 5, Y: y - 5, W: 10, H: 10}
					p.Fill(k, t.Background, 5)
					p.Stroke(k, t.Accent, 5, 1.5)
				}
			}
			for _, g := range guides {
				x0, y0 := v.at(g.x0, g.y0)
				x1, y1 := v.at(g.x1, g.y1)
				p.Line(x0, y0, x1, y1, 1, t.Danger)
			}
			if mark {
				mr := v.rect(m)
				p.Fill(mr, t.Accent.Alpha(0.08), 0)
				p.Stroke(mr, t.Accent.Alpha(0.7), 0, 1)
			}
		})
	})
	canvas.Children(func() { s.editor(c, shapes) })
}

// grid draws a dot every 24 DIPs of the board, or every so many more that
// they stay apart as it zooms out.
func grid(p *ui.Painter, r ui.Rect, v view, color ui.Color) {
	step := 24 * v.z
	for step < 16 {
		step *= 2
	}
	mod := func(a float32) float32 { return a - step*float32(math.Floor(float64(a/step))) }
	for x := r.X + mod(v.ox-r.X); x < r.X+r.W; x += step {
		for y := r.Y + mod(v.oy-r.Y); y < r.Y+r.H; y += step {
			p.Fill(ui.Rect{X: x - 0.75, Y: y - 0.75, W: 1.5, H: 1.5}, color, 0.75)
		}
	}
}

func ellipse(path *ui.Path, r ui.Rect) {
	// A quarter of a circle is a cubic curve with its handles 0.5523 of
	// the radius long.
	const k = 0.5523
	cx, cy, rx, ry := r.X+r.W/2, r.Y+r.H/2, r.W/2, r.H/2
	path.MoveTo(cx+rx, cy).
		CubeTo(cx+rx, cy+ry*k, cx+rx*k, cy+ry, cx, cy+ry).
		CubeTo(cx-rx*k, cy+ry, cx-rx, cy+ry*k, cx-rx, cy).
		CubeTo(cx-rx, cy-ry*k, cx-rx*k, cy-ry, cx, cy-ry).
		CubeTo(cx+rx*k, cy-ry, cx+rx, cy-ry*k, cx+rx, cy).Close()
}

type layoutKey struct {
	text        string
	size, width float32
	weight      int
	align       ui.Align
}

// layout returns the text of k laid out, as drawn last frame if it was.
func (s *State) layout(k layoutKey, family string) *ui.TextLayout {
	l, ok := s.layouts[k]
	if ok {
		return l
	}
	if l, ok = s.old[k]; !ok {
		l = ui.ShapeRichTextWith([]ui.Span{{Text: k.text}}, ui.Font{Family: family, Size: k.size, Weight: k.weight},
			ui.TextShape{Width: k.width, Align: k.align})
	}
	if s.layouts == nil {
		s.layouts = map[layoutKey]*ui.TextLayout{}
	}
	s.layouts[k] = l
	return l
}

// paint draws sh in color, and its text when label is set.
func (s *State) paint(p *ui.Painter, t *ui.Theme, v view, sh Shape, color ui.Color, label bool) {
	z := v.z
	r := v.rect(bounds(sh))
	switch sh.Kind {
	case Sticky:
		p.Shadow(r, 4*z, 0, 2*z, 10*z, 0, ui.RGB(0, 0, 0).Alpha(0.18))
		p.Fill(r, palette[int(sh.Color)%len(palette)].color, 4*z)
	case Rectangle:
		p.Fill(r, t.Background, 10*z)
		p.Stroke(r, color, 10*z, lineWidth*z)
	case Ellipse:
		var e ui.Path
		ellipse(&e, r)
		p.FillPath(&e, t.Background)
		p.StrokePath(&e, lineWidth*z, color)
	case Arrow:
		x0, y0 := v.at(sh.X, sh.Y)
		x1, y1 := v.at(sh.X+sh.W, sh.Y+sh.H)
		dx, dy := x1-x0, y1-y0
		l := hypot(dx, dy)
		if l < 1 {
			return
		}
		ux, uy := dx/l, dy/l
		head := min(12*z, l*0.5)
		bx, by := x1-ux*head, y1-uy*head
		p.StrokePolyline([][2]float32{{x0, y0}, {bx + ux*head*0.4, by + uy*head*0.4}}, lineWidth*z, color)
		var tip ui.Path
		w := head * 0.55
		tip.MoveTo(x1, y1).LineTo(bx-uy*w, by+ux*w).LineTo(bx+uy*w, by-ux*w).Close()
		p.FillPath(&tip, color)
		return
	case Pen:
		pts := line(sh)
		for k := range pts {
			pts[k].X, pts[k].Y = v.at(pts[k].X, pts[k].Y)
		}
		var path ui.Path
		if len(pts) == 1 {
			path.Circle(pts[0].X, pts[0].Y, penWidth*z/2)
			p.FillPath(&path, color)
			return
		}
		// Curves through the middles of the segments, bent at the points,
		// round the corners of the polyline.
		path.MoveTo(pts[0].X, pts[0].Y)
		for k := 1; k < len(pts)-1; k++ {
			path.QuadTo(pts[k].X, pts[k].Y, (pts[k].X+pts[k+1].X)/2, (pts[k].Y+pts[k+1].Y)/2)
		}
		path.LineTo(pts[len(pts)-1].X, pts[len(pts)-1].Y)
		p.StrokePath(&path, penWidth*z, color)
		return
	}
	if !label || sh.Text == "" {
		return
	}
	size, weight, align := look(sh.Kind)
	x, w := wrap(sh)
	l := s.layout(layoutKey{sh.Text, size * z, w * z, weight, align}, t.Font)
	_, h := l.Size()
	tx, _ := v.at(x, 0)
	text := t.Text
	if sh.Kind == Sticky {
		text = ink
	}
	p.TextLayout(l, tx, r.Y+(r.H-h)/2, text)
}

// editor shows the text of the shape edited in an input over it, in the
// place and the style it is drawn in.
func (s *State) editor(c *ui.Context, shapes *[]Shape) {
	i := find(*shapes, s.editing)
	if i < 0 {
		s.editing = 0
		return
	}
	if !s.focus && !s.text.Focused(c) {
		s.finish(shapes)
		return
	}
	t := c.Theme()
	sh := &(*shapes)[i]
	v := view{s.panX, s.panY, s.scale()}
	z := v.z
	size, weight, align := look(sh.Kind)
	x, w := wrap(*sh)
	r := v.rect(bounds(*sh))
	_, h := s.layout(layoutKey{sh.Text, size * z, w * z, weight, align}, t.Font).Size()
	_, line := c.MeasureText(0, ui.Span{Text: "M", Size: size * z, Weight: weight})
	h = max(h, line)
	tx, _ := v.at(x, 0)
	in := ui.TextAreaBase(c.Key(s.editing), &sh.Text).Bind(&s.text).Absolute().Left(tx).Top(r.Y+(r.H-h)/2).
		FontSize(size*z).FontWeight(weight).TextAlign(align).Lines(1, 1000).Label("Text")
	if sh.Kind == Text {
		in.Width(r.W + size*z)
	} else {
		in.Width(w * z)
	}
	if sh.Kind == Sticky {
		in.TextColor(ink)
	}
	if s.focus {
		in.Focus().SetTextSelection(0, utf8.RuneCountInString(sh.Text))
		s.focus = false
	}
	if in.Changed() {
		s.dirty = true
	}
	in.OnTextKey(func(k ui.TextKey) bool {
		if k.Key == ui.KeyEscape && k.Mods == 0 {
			s.finish(shapes)
			s.refocus = true
			return true
		}
		return false
	})
}

// toolbar shows the tools and the colors of sticky notes, floating at the
// bottom of the board.
func (s *State) toolbar(c *ui.Context, shapes *[]Shape) {
	t := c.Theme()
	ui.Row(c).Attach(ui.AnchorBottom, ui.AnchorBottom).Bottom(t.Space(4)).Padding(t.Space(1.5)).Gap(t.Space(1)).
		AlignItems(ui.Center).Radius(t.Space(3)).Background(t.Background).Border(1, t.Border).
		Shadow(0, 4, 16, 0, ui.RGB(0, 0, 0).Alpha(0.14)).Role(ui.RoleToolbar).Label("Tools").Children(func() {
		for i, tl := range tools {
			b := ui.ButtonBase(c.Key(tl.name)).Size(t.Space(9), t.Space(9)).Center().Radius(t.Space(2)).KeepFocus().
				Role(ui.RoleRadio).Checked(i == s.tool).Label(tl.name).Tooltip(tl.name + "  " + tl.hint)
			switch {
			case i == s.tool:
				b.Background(t.Accent).TextColor(t.AccentText)
			case b.Hovered():
				b.Background(t.SurfaceHover)
			}
			if b.Clicked() {
				s.tool = i
			}
			b.Children(func() { ui.Icon(c, tl.icon).FontSize(t.Space(5)) })
		}
		ui.Box(c).Size(1, t.Space(6)).Background(t.Border).Margin(0, t.Space(1.5))
		for i, pc := range palette {
			sw := ui.ButtonBase(c.Key(pc.name)).Size(t.Space(8), t.Space(8)).Center().Radius(t.Space(4)).KeepFocus().
				Role(ui.RoleRadio).Checked(Color(i) == s.color).Label(pc.name)
			switch {
			case Color(i) == s.color:
				sw.Border(2, t.Accent)
			case sw.Hovered():
				sw.Background(t.SurfaceHover)
			}
			if sw.Clicked() {
				s.color = Color(i)
				before := slices.Clone(*shapes)
				for k := range *shapes {
					if sh := &(*shapes)[k]; sh.Kind == Sticky && s.isChosen(sh.ID) {
						sh.Color = s.color
					}
				}
				s.commit(*shapes, before)
			}
			sw.Children(func() {
				ui.Box(c).Size(t.Space(5), t.Space(5)).Radius(t.Space(2.5)).Background(pc.color).Border(1, ink.Alpha(0.15))
			})
		}
	})
}
