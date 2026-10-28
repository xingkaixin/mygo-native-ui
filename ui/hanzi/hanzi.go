// Package hanzi shows how a Chinese character is written, stroke by
// stroke, as Hanzi Writer and the stroke order pages of dictionaries do,
// on a 米字格 practice grid. Animate draws the strokes in order, each
// along the way the brush goes. Quiz shows the character faintly to trace
// over, 描红: the user draws each stroke in order with the pointer, a
// stroke drawn near enough and the right way turns to ink, and after
// three misses on a stroke it shows where that stroke goes.
//
//	ev := hanzi.View(c, &app.writer, hanzi.Char{Strokes: app.strokes}).Size(320, 320)
//	if ev.Done {
//		c.Toast(fmt.Sprintf("Written with %d mistakes", ev.Mistakes))
//	}
//
// A Char's strokes are lines along their middles, from where the brush
// sets down to where it lifts, in a square from 0, 0 at its top left to
// 1, 1, smoothed between their points: the medians of Make Me a Hanzi
// fit, turned upright and scaled from 1024. Call Animate or Quiz again
// when the character changes.
package hanzi

import (
	"math"
	"time"

	"github.com/egoist/mygo/ui"
)

// hint is how many misses on a stroke show where it goes.
const hint = 3

var (
	ink   = ui.RGB(28, 25, 23)
	grid  = ui.RGB(220, 70, 60)
	trace = ui.RGB(37, 99, 235)
)

// Char is a character as its strokes, in the order they are written.
type Char struct {
	Strokes [][]ui.Point
}

// State is what a view keeps: whether it plays the strokes or quizzes
// them, and how far the quiz went. Keep it in the app's state, one for
// each view.
type State struct {
	run      uint64
	quiz     bool
	next     int
	misses   int
	mistakes int
	trace    []ui.Point
}

// Animate draws the character's strokes in order, from none.
func (s *State) Animate() {
	*s = State{run: s.run + 1}
}

// Quiz starts a quiz of the character's strokes, from the first.
func (s *State) Quiz() {
	*s = State{run: s.run, quiz: true}
}

// Events is a view, and whether its quiz ended this frame.
type Events struct {
	ui.Element
	// Done reports that the last stroke of a quiz was traced, after
	// Mistakes misses in all.
	Done     bool
	Mistakes int
}

// View creates a view of ch on a practice grid, as large as fits in the
// view's box. Give the view a size.
func View(c *ui.Context, s *State, ch Char) Events {
	view := ui.Box(c).Role(ui.RoleImage)
	ev := Events{Element: view}
	strokes := make([][]ui.Point, len(ch.Strokes))
	for i, st := range ch.Strokes {
		strokes[i] = smooth(st)
	}
	shown := make([]float32, len(strokes))
	switch {
	case s.quiz:
		for i := range min(s.next, len(shown)) {
			shown[i] = 1
		}
		ev.Done, ev.Mistakes = quiz(s, view, strokes), s.mistakes
	case s.run == 0:
		for i := range shown {
			shown[i] = 1
		}
	default:
		tl := view.Timeline("hanzi.strokes", ui.TimelineOptions{Run: s.run, Delay: 200 * time.Millisecond})
		for i, st := range strokes {
			d := 250*time.Millisecond + time.Duration(length(st)*float32(time.Second))
			shown[i] = tl.Tween(ui.Tween{To: 1, Duration: d, Ease: ui.EaseInOut}, ui.AfterPrevious(120*time.Millisecond))
		}
	}

	quizzing, next, misses, traced := s.quiz, s.next, s.misses, s.trace
	view.Draw(func(p *ui.Painter, r ui.Rect) {
		side := min(r.W, r.H)
		box := ui.Rect{X: r.X + (r.W-side)/2, Y: r.Y + (r.H-side)/2, W: side, H: side}
		practiceGrid(p, box)
		width := side * 0.075
		at := func(q ui.Point) (float32, float32) { return box.X + q.X*side, box.Y + q.Y*side }
		for i, st := range strokes {
			faint := ink.Alpha(0.12)
			if quizzing {
				faint = grid.Alpha(0.18)
			}
			p.StrokePath(path(st, 1, at), width, faint)
			if quizzing && i == next && misses >= hint {
				p.StrokePath(path(st, 1, at), width, trace.Alpha(0.3))
			}
			if shown[i] > 0 {
				p.StrokePath(path(st, shown[i], at), width, ink)
			}
		}
		if len(traced) > 1 {
			p.StrokePath(path(traced, 1, at), width*0.6, trace)
		}
	})
	return ev
}

// quiz follows the pointer drawing on view, and judges each stroke drawn
// against the next of strokes once the pointer lets go. It reports
// whether the last stroke was traced.
func quiz(s *State, view ui.Element, strokes [][]ui.Point) bool {
	view.Cursor(ui.CursorCrosshair)
	b := view.LayoutBounds()
	side := min(b.W, b.H)
	if side <= 0 || s.next >= len(strokes) {
		return false
	}
	x, y, _ := view.PointerPosition()
	q := ui.Point{X: (x - (b.W-side)/2) / side, Y: (y - (b.H-side)/2) / side}
	if view.Pressed() {
		if n := len(s.trace); n == 0 || dist(s.trace[n-1], q) > 0.01 {
			s.trace = append(s.trace, q)
		}
		return false
	}
	if len(s.trace) == 0 {
		return false
	}
	ok := matches(s.trace, strokes[s.next])
	s.trace = nil
	if !ok {
		s.misses++
		s.mistakes++
		return false
	}
	s.next++
	s.misses = 0
	return s.next == len(strokes)
}

// matches reports whether drawn traces stroke: starting and ending near
// its ends, going its way, and keeping near it all along.
func matches(drawn, stroke []ui.Point) bool {
	if len(drawn) < 2 {
		return false
	}
	a, z := drawn[0], drawn[len(drawn)-1]
	sa, sz := stroke[0], stroke[len(stroke)-1]
	if dist(a, sa) > 0.2 || dist(z, sz) > 0.2 {
		return false
	}
	var off float32
	for _, q := range drawn {
		off += near(stroke, q)
	}
	return off/float32(len(drawn)) < 0.08
}

// near returns how far q is from the polyline line.
func near(line []ui.Point, q ui.Point) float32 {
	best := float32(math.MaxFloat32)
	for i := 1; i < len(line); i++ {
		a, b := line[i-1], line[i]
		dx, dy := b.X-a.X, b.Y-a.Y
		t := float32(0)
		if l := dx*dx + dy*dy; l > 0 {
			t = max(0, min(1, ((q.X-a.X)*dx+(q.Y-a.Y)*dy)/l))
		}
		best = min(best, dist(q, ui.Point{X: a.X + t*dx, Y: a.Y + t*dy}))
	}
	return best
}

// practiceGrid draws a 米字格 over box: a frame, and dashed lines across
// its middle and its diagonals.
func practiceGrid(p *ui.Painter, box ui.Rect) {
	p.Fill(box, ui.RGB(255, 253, 248), 0)
	p.Stroke(box, grid.Alpha(0.8), 0, 2)
	l, t, r, b := box.X, box.Y, box.X+box.W, box.Y+box.H
	cx, cy := l+box.W/2, t+box.H/2
	for _, seg := range [][4]float32{{l, cy, r, cy}, {cx, t, cx, b}, {l, t, r, b}, {r, t, l, b}} {
		dash := new(ui.Path).MoveTo(seg[0], seg[1]).LineTo(seg[2], seg[3]).Dash(0, 6, 5)
		p.StrokePath(dash, 1, grid.Alpha(0.45))
	}
}

// path returns the part of line from its start to part of its length, in
// the view's coordinates by at.
func path(line []ui.Point, part float32, at func(ui.Point) (float32, float32)) *ui.Path {
	pa := new(ui.Path)
	left := length(line) * part
	x, y := at(line[0])
	pa.MoveTo(x, y)
	for i := 1; i < len(line) && left > 0; i++ {
		a, b := line[i-1], line[i]
		d := dist(a, b)
		if d > left {
			b = ui.Point{X: a.X + (b.X-a.X)*left/d, Y: a.Y + (b.Y-a.Y)*left/d}
		}
		left -= d
		x, y := at(b)
		pa.LineTo(x, y)
	}
	return pa
}

// smooth returns points along a Catmull-Rom spline through pts, eight
// for each span between two of them. It keeps the sharp turns of strokes
// as 横折 sharp, as a spline through them would swing past them.
func smooth(pts []ui.Point) []ui.Point {
	if len(pts) < 3 {
		return pts
	}
	sharp := func(i int) bool {
		if i <= 0 || i >= len(pts)-1 {
			return true
		}
		a, b, c := pts[i-1], pts[i], pts[i+1]
		ux, uy, vx, vy := b.X-a.X, b.Y-a.Y, c.X-b.X, c.Y-b.Y
		cos := (ux*vx + uy*vy) / (dist(a, b) * dist(b, c))
		return cos < 0.5
	}
	out := []ui.Point{pts[0]}
	for i := 0; i+1 < len(pts); i++ {
		p0, p1, p2, p3 := pts[max(0, i-1)], pts[i], pts[i+1], pts[min(len(pts)-1, i+2)]
		if sharp(i) {
			p0 = p1
		}
		if sharp(i + 1) {
			p3 = p2
		}
		for k := 1; k <= 8; k++ {
			t := float32(k) / 8
			t2, t3 := t*t, t*t*t
			f := func(a, b, c, d float32) float32 {
				return 0.5 * (2*b + (c-a)*t + (2*a-5*b+4*c-d)*t2 + (3*b-a-3*c+d)*t3)
			}
			out = append(out, ui.Point{X: f(p0.X, p1.X, p2.X, p3.X), Y: f(p0.Y, p1.Y, p2.Y, p3.Y)})
		}
	}
	return out
}

func length(line []ui.Point) float32 {
	var l float32
	for i := 1; i < len(line); i++ {
		l += dist(line[i-1], line[i])
	}
	return l
}

func dist(a, b ui.Point) float32 { return float32(math.Hypot(float64(a.X-b.X), float64(a.Y-b.Y))) }
