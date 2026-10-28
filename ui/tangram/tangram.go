// Package tangram is the puzzle of the seven tans cut from a square, two
// large triangles, a medium one, two small ones, a square and a
// parallelogram, to be laid edge to edge, none on another, so that they
// fill a silhouette. The silhouette shows on the left and the tans start
// on the right, as the square they were cut from, pulled apart. A
// dragged tan whose corner is dropped near a corner of the silhouette or
// of another tan snaps to it. A click turns a tan an eighth of a turn
// clockwise, a shift-click back, and a right-click turns it over. Once
// the tans fill the silhouette they fade into one shape in the color of
// text.
//
//	app.puzzle = tangram.New(house) // the silhouette's corners
//	tangram.View(c, &app.board, &app.puzzle)
//	if app.puzzle.Solved() { … }
package tangram

import (
	"math"
	"slices"
	"time"

	"github.com/egoist/mygo/ui"
)

// Kind is the shape of a tan.
type Kind int8

const (
	LargeTriangle Kind = iota
	MediumTriangle
	SmallTriangle
	Square
	Parallelogram
)

// shapes are the tans' corners, in units of a small triangle's legs,
// with y down.
var shapes = [...][][2]float64{
	LargeTriangle:  {{0, 0}, {2, 0}, {0, 2}},
	MediumTriangle: {{0, 0}, {2, 0}, {1, 1}},
	SmallTriangle:  {{0, 0}, {1, 0}, {0, 1}},
	Square:         {{0, 0}, {1, 0}, {1, 1}, {0, 1}},
	Parallelogram:  {{0, 0}, {1, 0}, {2, 1}, {1, 1}},
}

// Tan is a piece where it lies: its centroid at X, Y from the board's top
// left, in units of a small triangle's legs, turned over first when
// Flipped, then turned Turn eighths of a turn clockwise.
type Tan struct {
	Kind    Kind
	X, Y    float64
	Turn    int
	Flipped bool
}

// Corners returns the tan's corners where it lies.
func (t Tan) Corners() [][2]float64 {
	return corners(t.Kind, t.Flipped, float64(t.Turn)*45, t.X, t.Y, 1)
}

// corners returns the corners of a tan of kind k turned deg degrees
// clockwise about its centroid at x, y, then squeezed sideways by sx.
func corners(k Kind, flipped bool, deg, x, y, sx float64) [][2]float64 {
	shape := shapes[k]
	var cx, cy float64
	for _, v := range shape {
		cx, cy = cx+v[0], cy+v[1]
	}
	cx, cy = cx/float64(len(shape)), cy/float64(len(shape))
	sin, cos := math.Sincos(deg * math.Pi / 180)
	out := make([][2]float64, len(shape))
	for i, v := range shape {
		dx, dy := v[0]-cx, v[1]-cy
		if flipped {
			dx = -dx
		}
		out[i] = [2]float64{x + (dx*cos-dy*sin)*sx, y + dx*sin + dy*cos}
	}
	return out
}

// Puzzle is a silhouette and the seven tans: the two large triangles,
// the medium one, the two small ones, the square and the parallelogram.
type Puzzle struct {
	// Outline is the silhouette's corners from the board's top left.
	Outline [][2]float64
	Tans    [7]Tan
}

// home is where New lays the tans out, from the middle of the tray: the
// square they are cut from, its sides 2√2, pulled apart by a quarter.
var home = [7]Tan{
	{LargeTriangle, 0, -1.1785, 5, false},
	{LargeTriangle, -1.1785, 0, 3, false},
	{MediumTriangle, 1.1785, 1.1785, 7, false},
	{SmallTriangle, 1.4731, -0.8839, 7, false},
	{SmallTriangle, 0, 0.5893, 1, false},
	{Square, 0.8839, 0, 1, false},
	{Parallelogram, -0.4419, 1.3258, 3, false},
}

// New returns the puzzle of outline, given in units of a small
// triangle's legs, which is moved to the middle of the board's left, with
// the tans laid out on its right.
func New(outline [][2]float64) Puzzle {
	minX, minY, maxX, maxY := bounds(outline)
	left, h := board(maxX-minX, maxY-minY)
	dx, dy := left/2-(minX+maxX)/2, h/2-(minY+maxY)/2
	p := Puzzle{Outline: make([][2]float64, len(outline))}
	for i, v := range outline {
		p.Outline[i] = [2]float64{v[0] + dx, v[1] + dy}
	}
	for i, t := range home {
		t.X, t.Y = t.X+left+tray/2, t.Y+h/2
		p.Tans[i] = t
	}
	return p
}

// tray is how wide the board's right is, where the tans start.
const tray = 7.0

// board returns how wide the board's left is, where a silhouette w by h
// shows, and how tall the board is.
func board(w, h float64) (left, height float64) {
	return max(7, w+2), max(7, h+2)
}

func bounds(pts [][2]float64) (minX, minY, maxX, maxY float64) {
	minX, minY, maxX, maxY = math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	for _, v := range pts {
		minX, maxX = min(minX, v[0]), max(maxX, v[0])
		minY, maxY = min(minY, v[1]), max(maxY, v[1])
	}
	return
}

func inside(poly [][2]float64, x, y float64) bool {
	in := false
	for i, a := range poly {
		b := poly[(i+1)%len(poly)]
		if (a[1] > y) != (b[1] > y) && x < a[0]+(y-a[1])*(b[0]-a[0])/(b[1]-a[1]) {
			in = !in
		}
	}
	return in
}

// Solved reports whether the tans fill the silhouette, none on another.
func (p *Puzzle) Solved() bool {
	if len(p.Outline) < 3 {
		return false
	}
	var tans [7][][2]float64
	for i, t := range p.Tans {
		tans[i] = t.Corners()
	}
	minX, minY, maxX, maxY := bounds(p.Outline)
	// Samples a tenth of a unit apart, offset so that none falls on the
	// edges of tans laid on whole and half units.
	var in, wrong int
	for y := minY + 0.0137; y < maxY; y += 0.1 {
		for x := minX + 0.0291; x < maxX; x += 0.1 {
			n := 0
			for _, c := range tans {
				if inside(c, x, y) {
					n++
				}
			}
			if inside(p.Outline, x, y) {
				in++
				if n != 1 {
					wrong++
				}
			} else if n > 0 {
				wrong++
			}
		}
	}
	// Corners snap exactly, but tans laid by hand may be a hair off.
	return in > 0 && wrong*100 < in
}

// snap moves tan i so that its corner nearest a corner of the silhouette
// or of another tan, if near enough, lies on it.
func (p *Puzzle) snap(i int) {
	best, dx, dy := 0.35, 0.0, 0.0
	mine := p.Tans[i].Corners()
	try := func(pts [][2]float64) {
		for _, q := range pts {
			for _, v := range mine {
				if d := math.Hypot(q[0]-v[0], q[1]-v[1]); d < best {
					best, dx, dy = d, q[0]-v[0], q[1]-v[1]
				}
			}
		}
	}
	try(p.Outline)
	for j, t := range p.Tans {
		if j != i {
			try(t.Corners())
		}
	}
	p.Tans[i].X += dx
	p.Tans[i].Y += dy
}

// State is what a board keeps: which tan is on top, the one being
// dragged, and how the last turned one was. Keep it in the app's state,
// one for each board.
type State struct {
	order        []int
	held         int
	grabX, grabY float64
	downX, downY float32
	moved        bool
	from         [7]Tan
	at           [7]time.Time
	solved       bool
	solvedAt     time.Time
}

const (
	unit     = 44
	turnTime = 180 * time.Millisecond
	fadeTime = 600 * time.Millisecond
)

// colors are the tans' hues, in the order of Puzzle.Tans.
var colors = [7]float32{25, 250, 150, 60, 310, 95, 200}

// View creates the board of p.
func View(c *ui.Context, s *State, p *Puzzle) ui.Element {
	t := c.Theme()
	now := c.Now()
	still := c.Preferences().ReduceMotion
	if len(s.order) != len(p.Tans) {
		s.order = []int{0, 1, 2, 3, 4, 5, 6}
	}
	if solved := p.Solved(); solved != s.solved {
		s.solved, s.solvedAt = solved, now
	}
	minX, minY, maxX, maxY := bounds(p.Outline)
	left, h := board(maxX-minX, maxY-minY)
	w := left + tray
	label := "Tangram"
	if s.solved {
		label = "Tangram, solved"
	}
	e := ui.Box(c).Size(float32(w)*unit, float32(h)*unit).Radius(t.Space(3)).Background(t.Surface).
		Border(1, t.Border).Role(ui.RoleImage).Label(label)

	// shown returns tan i's corners as drawn, partway through its last
	// turn.
	animating := false
	shown := func(i int) [][2]float64 {
		tn := p.Tans[i]
		k := float64(now.Sub(s.at[i])) / float64(turnTime)
		if still || k >= 1 {
			return tn.Corners()
		}
		animating = true
		k = 1 - (1-k)*(1-k)
		f := s.from[i]
		x, y := f.X+(tn.X-f.X)*k, f.Y+(tn.Y-f.Y)*k
		if f.Flipped != tn.Flipped {
			return corners(f.Kind, f.Flipped, float64(f.Turn)*45, x, y, 1-2*k)
		}
		d := float64((tn.Turn-f.Turn+12)%8 - 4)
		return corners(f.Kind, f.Flipped, (float64(f.Turn)+d*k)*45, x, y, 1)
	}
	at := func(x, y float32) int {
		for _, i := range slices.Backward(s.order) {
			if inside(shown(i), float64(x/unit), float64(y/unit)) {
				return i
			}
		}
		return -1
	}
	// turn sets tan i's turn and side, keeping where it was to animate
	// from.
	turn := func(i, by int, flip bool) {
		tn := &p.Tans[i]
		s.from[i], s.at[i] = *tn, now
		if flip {
			// Turning a tan over left to right undoes its turn.
			tn.Flipped, tn.Turn = !tn.Flipped, -tn.Turn
		}
		tn.Turn = ((tn.Turn+by)%8 + 8) % 8
		p.snap(i)
	}
	raise := func(i int) {
		s.order = append(slices.DeleteFunc(s.order, func(j int) bool { return j == i }), i)
	}

	if px, py, over := e.PointerPosition(); s.held > 0 {
		e.Cursor(ui.CursorGrabbing)
	} else if over && at(px, py) >= 0 {
		e.Cursor(ui.CursorGrab)
	}
	e.HandleInput(func(in ui.InputEvent) bool {
		switch in.Kind {
		case ui.InputPointerDown:
			i := at(in.X, in.Y)
			if i < 0 {
				return false
			}
			raise(i)
			if in.Button == 1 {
				turn(i, 0, true)
				return true
			}
			s.held, s.moved = i+1, false
			s.downX, s.downY = in.X, in.Y
			s.grabX, s.grabY = float64(in.X/unit)-p.Tans[i].X, float64(in.Y/unit)-p.Tans[i].Y
		case ui.InputPointerMove:
			if s.held == 0 {
				return false
			}
			if !s.moved && math.Hypot(float64(in.X-s.downX), float64(in.Y-s.downY)) < 4 {
				return true
			}
			s.moved = true
			tn := &p.Tans[s.held-1]
			s.at[s.held-1] = time.Time{}
			tn.X = max(0, min(w, float64(in.X/unit)-s.grabX))
			tn.Y = max(0, min(h, float64(in.Y/unit)-s.grabY))
		case ui.InputPointerUp:
			if s.held == 0 {
				return false
			}
			i := s.held - 1
			s.held = 0
			if s.moved {
				p.snap(i)
			} else if in.Mods&ui.Shift != 0 {
				turn(i, -1, false)
			} else {
				turn(i, 1, false)
			}
		default:
			return false
		}
		return true
	})

	e.Draw(func(pt *ui.Painter, r ui.Rect) {
		path := func(pts [][2]float64) *ui.Path {
			var path ui.Path
			for i, v := range pts {
				x, y := r.X+float32(v[0])*unit, r.Y+float32(v[1])*unit
				if i == 0 {
					path.MoveTo(x, y)
				} else {
					path.LineTo(x, y)
				}
			}
			path.Close()
			return &path
		}
		if len(p.Outline) > 2 {
			pt.FillPath(path(p.Outline), t.Text.Alpha(0.1))
		}
		ink := t.Text
		k := float32(1)
		if !still {
			k = min(1, float32(now.Sub(s.solvedAt))/float32(fadeTime))
		}
		if k < 1 {
			pt.AnimationFrame()
		}
		for _, i := range s.order {
			col := ui.Oklch(0.74, 0.13, colors[i])
			if t.Dark {
				col = ui.Oklch(0.66, 0.12, colors[i])
			}
			switch {
			case s.solved:
				col = col.Mix(ink, k)
			case !s.solvedAt.IsZero():
				col = ink.Mix(col, k)
			}
			pts := shown(i)
			if s.held == i+1 && s.moved {
				lifted := make([][2]float64, len(pts))
				for j, v := range pts {
					lifted[j] = [2]float64{v[0] + 0.06, v[1] + 0.1}
				}
				pt.FillPath(path(lifted), ui.RGB(0, 0, 0).Alpha(0.18))
			}
			pt.FillPath(path(pts), col)
			pt.StrokePath(path(pts), 1, t.Surface.Alpha(0.9))
		}
		if animating {
			pt.AnimationFrame()
		}
	})
	return e
}
