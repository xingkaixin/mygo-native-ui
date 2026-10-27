// Package goban shows a board of Go, as Go servers and editors do: 9, 13
// or 19 lines on wood with their star points, shaded stones, a ring on
// the last stone played, the stone to play following the pointer as a
// ghost, and letters and numbers along the four edges, which skip I as
// Go's do. A click on an empty point reports it for the app to play.
//
// Board plays stones by the rules: a stone takes the stones of the other
// color it leaves without liberties, and may not leave itself without
// any. Ko, which forbids a board repeating the one before the last move,
// is the app's to check, as the moves are. ParseSGF reads a game's main
// line.
//
//	ev := goban.View(c, &app.goban, app.board, goban.Options{Turn: app.turn, Last: app.last})
//	if ev.Played {
//		if next, _, ok := app.board.Play(ev.Point, app.turn); ok {
//			app.board = next
//		}
//	}
package goban

import (
	"errors"
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
)

// Stone is what a point holds.
type Stone byte

const (
	Empty Stone = iota
	Black
	White
)

// Other returns the stone of the other color.
func (s Stone) Other() Stone { return 3 - s }

// Point is an intersection of the board, X from the left and Y from the
// top, both from 0, as SGF counts them.
type Point struct{ X, Y int }

// Name returns the point's name as Go players write it, its column's
// letter, I skipped, and its row from the bottom: "D4" on a board of
// size.
func (p Point) Name(size int) string {
	col := 'A' + rune(p.X)
	if col >= 'I' {
		col++
	}
	return string(col) + strconv.Itoa(size-p.Y)
}

// Board is the stones on a board of Size lines each way.
type Board struct {
	Size   int
	stones []Stone
}

// NewBoard returns an empty board of size lines each way.
func NewBoard(size int) Board { return Board{Size: size, stones: make([]Stone, size*size)} }

func (b Board) on(p Point) bool { return p.X >= 0 && p.Y >= 0 && p.X < b.Size && p.Y < b.Size }

// At returns the stone at p, Empty off the board.
func (b Board) At(p Point) Stone {
	if !b.on(p) || b.stones == nil {
		return Empty
	}
	return b.stones[p.Y*b.Size+p.X]
}

// Set returns the board with s at p, the stones around left as they are.
func (b Board) Set(p Point, s Stone) Board {
	if !b.on(p) {
		return b
	}
	b.stones = slices.Clone(b.stones)
	if b.stones == nil {
		b.stones = make([]Stone, b.Size*b.Size)
	}
	b.stones[p.Y*b.Size+p.X] = s
	return b
}

// Equal reports whether b and o hold the same stones, as ko asks.
func (b Board) Equal(o Board) bool {
	return b.Size == o.Size && slices.Equal(b.stones, o.stones)
}

func (b Board) around(p Point) []Point {
	var out []Point
	for _, n := range []Point{{p.X - 1, p.Y}, {p.X + 1, p.Y}, {p.X, p.Y - 1}, {p.X, p.Y + 1}} {
		if b.on(n) {
			out = append(out, n)
		}
	}
	return out
}

// group returns the stones joined to the one at p, and whether they have
// a liberty.
func (b Board) group(p Point) ([]Point, bool) {
	color := b.At(p)
	seen := map[Point]bool{p: true}
	stack, out, free := []Point{p}, []Point{}, false
	for len(stack) > 0 {
		q := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		out = append(out, q)
		for _, n := range b.around(q) {
			switch at := b.At(n); {
			case at == Empty:
				free = true
			case at == color && !seen[n]:
				seen[n] = true
				stack = append(stack, n)
			}
		}
	}
	return out, free
}

// Play returns the board after a stone s at p, with the stones it takes
// off and how many, and false where p is taken or the stone would have no
// liberty.
func (b Board) Play(p Point, s Stone) (Board, int, bool) {
	if !b.on(p) || b.At(p) != Empty || s == Empty {
		return b, 0, false
	}
	next := b.Set(p, s)
	taken := 0
	for _, n := range next.around(p) {
		if next.At(n) != s.Other() {
			continue
		}
		if g, free := next.group(n); !free {
			for _, q := range g {
				next.stones[q.Y*next.Size+q.X] = Empty
			}
			taken += len(g)
		}
	}
	if _, free := next.group(p); !free {
		return b, 0, false
	}
	return next, taken, true
}

// Move is a stone played at a point, or a pass.
type Move struct {
	Stone Stone
	Point Point
	Pass  bool
}

// Game is a game an SGF holds: the board's size, the stones set on it
// before the first move, and the moves of its main line.
type Game struct {
	Size  int
	Setup Board
	Moves []Move
}

// ParseSGF reads the main line of a game of Go in SGF: SZ, the stones AB
// and AW set, and the moves B and W, "tt" or empty a pass. The variations
// after the first are left out.
func ParseSGF(sgf string) (Game, error) {
	g := Game{Size: 19}
	type prop struct {
		key    string
		values []string
	}
	var props []prop
	// first[d] is whether a variation at depth d+1 opened already: those
	// after the first are skipped, from the depth skip.
	depth, skip := 0, 0
	first := []bool{false}
	for i := 0; i < len(sgf); i++ {
		switch ch := sgf[i]; {
		case ch == '(':
			depth++
			first = append(first[:depth], false)
			if skip == 0 && first[depth-1] {
				skip = depth
			}
			first[depth-1] = true
		case ch == ')':
			if skip == depth {
				skip = 0
			}
			depth = max(0, depth-1)
		case ch >= 'A' && ch <= 'Z':
			j := i
			for j < len(sgf) && sgf[j] >= 'A' && sgf[j] <= 'Z' {
				j++
			}
			p := prop{key: sgf[i:j]}
			for j < len(sgf) && sgf[j] == '[' {
				end := closing(sgf, j+1)
				if end < 0 {
					return g, errors.New("goban: a value of " + p.key + " has no ]")
				}
				p.values = append(p.values, sgf[j+1:end])
				j = end + 1
				for j < len(sgf) && strings.ContainsRune(" \t\r\n", rune(sgf[j])) {
					j++
				}
			}
			if skip == 0 {
				props = append(props, p)
			}
			i = j - 1
		}
	}
	for _, p := range props {
		if p.key == "SZ" && len(p.values) > 0 {
			n, err := strconv.Atoi(p.values[0])
			if err != nil || n < 2 || n > 25 {
				return g, errors.New("goban: SZ " + strconv.Quote(p.values[0]) + " is not a size from 2 to 25")
			}
			g.Size = n
		}
	}
	g.Setup = NewBoard(g.Size)
	point := func(v string) (Point, bool) {
		if len(v) != 2 {
			return Point{}, false
		}
		p := Point{int(v[0] - 'a'), int(v[1] - 'a')}
		return p, g.Setup.on(p)
	}
	for _, p := range props {
		stone := map[string]Stone{"B": Black, "W": White, "AB": Black, "AW": White}[p.key]
		if stone == Empty {
			continue
		}
		for _, v := range p.values {
			at, on := point(v)
			switch {
			case p.key == "AB" || p.key == "AW":
				if on {
					g.Setup = g.Setup.Set(at, stone)
				}
			case on:
				g.Moves = append(g.Moves, Move{Stone: stone, Point: at})
			case v == "" || v == "tt":
				g.Moves = append(g.Moves, Move{Stone: stone, Pass: true})
			default:
				return g, errors.New("goban: " + strconv.Quote(v) + " is not a point")
			}
		}
	}
	return g, nil
}

// closing returns where the value from i ends, at the first ] not
// escaped by a backslash, or -1.
func closing(s string, i int) int {
	for ; i < len(s); i++ {
		switch s[i] {
		case '\\':
			i++
		case ']':
			return i
		}
	}
	return -1
}

// Options are what a board shows and plays.
type Options struct {
	// Turn is the stone a click plays, its ghost under the pointer; Empty
	// plays none.
	Turn Stone
	// Last is the stone played last, ringed; nil is none.
	Last *Point
}

// Events are what happened on the board since the last frame.
type Events struct {
	ui.Element
	// Played reports a click on the empty Point, for the app to play.
	Played bool
	Point  Point
}

// State is what a board keeps: its lines' place as of the last layout.
// Keep it in the app's state, one for each board.
type State struct {
	gap, ox, oy float32
	size        int
}

// pointAt returns the point nearest x, y, and whether it is within half a
// line's gap.
func (s *State) pointAt(x, y float32) (Point, bool) {
	if s.gap <= 0 {
		return Point{}, false
	}
	p := Point{int(math.Round(float64((x - s.ox) / s.gap))), int(math.Round(float64((y - s.oy) / s.gap)))}
	return p, p.X >= 0 && p.Y >= 0 && p.X < s.size && p.Y < s.size
}

// stars returns the star points of a board of size lines.
func stars(size int) []Point {
	var at []int
	switch {
	case size >= 15:
		at = []int{3, size / 2, size - 4}
	case size >= 9:
		edge := 2
		if size >= 13 {
			edge = 3
		}
		at = []int{edge, size - 1 - edge}
	}
	var out []Point
	for _, x := range at {
		for _, y := range at {
			out = append(out, Point{x, y})
		}
	}
	if size%2 == 1 && size >= 9 && size < 15 {
		out = append(out, Point{size / 2, size / 2})
	}
	return out
}

// View creates the board of b, as big as the room it is given allows.
func View(c *ui.Context, s *State, b Board, o Options) Events {
	ev := Events{}
	board := ui.Box(c).Role(ui.RoleGroup).Label("Go board")
	bounds := board.LayoutBounds()
	s.size = max(2, b.Size)
	// A line and a half of wood around the edge lines holds the
	// coordinates.
	s.gap = float32(math.Floor(float64(min(bounds.W, bounds.H) / float32(s.size+2))))
	s.ox = (bounds.W-s.gap*float32(s.size-1))/2 + 0.5
	s.oy = (bounds.H-s.gap*float32(s.size-1))/2 + 0.5
	px, py, over := board.PointerPosition()
	hover, on := s.pointAt(px, py)
	ghost := over && on && o.Turn != Empty && b.At(hover) == Empty
	if ghost {
		board.Cursor(ui.CursorPointer)
	}
	if board.Clicked() && ghost {
		ev.Played, ev.Point = true, hover
	}
	board.Draw(func(p *ui.Painter, r ui.Rect) {
		s.draw(p, r, b, o, hover, ghost)
	})
	ev.Element = board
	return ev
}

func (s *State) draw(p *ui.Painter, r ui.Rect, b Board, o Options, hover Point, ghost bool) {
	g, n := s.gap, s.size
	if g <= 0 {
		return
	}
	x0, y0 := r.X+s.ox, r.Y+s.oy
	span := g * float32(n-1)
	wood := ui.Rect{X: x0 - g*1.2, Y: y0 - g*1.2, W: span + g*2.4, H: span + g*2.4}
	p.Shadow(wood, 3, 0, 2, 10, 0, ui.RGB(0, 0, 0).Alpha(0.3))
	p.FillGradient(wood, ui.LinearGradient{From: ui.Hex("#e3bd6f"), To: ui.Hex("#d4a653"), Angle: 160}, 3)
	ink := ui.RGB(40, 30, 15)
	line := max(1, g/30)
	for i := range n {
		at := float32(i) * g
		w := line
		if i == 0 || i == n-1 {
			w = line * 1.6
		}
		p.Line(x0+at, y0, x0+at, y0+span, w, ink)
		p.Line(x0, y0+at, x0+span, y0+at, w, ink)
	}
	for _, st := range stars(n) {
		var dot ui.Path
		p.FillPath(dot.Circle(x0+float32(st.X)*g, y0+float32(st.Y)*g, max(2, g*0.1)), ink)
	}
	size := g * 0.3
	for i := range n {
		at := float32(i) * g
		label := Point{i, 0}.Name(n)[:1]
		w, h := p.MeasureText(0, ui.Span{Text: label, Size: size})
		p.Text(x0+at-w/2, y0+span+g*0.55, label, size, ink.Alpha(0.75))
		p.Text(x0+at-w/2, y0-g*0.55-h, label, size, ink.Alpha(0.75))
		num := strconv.Itoa(n - i)
		w, h = p.MeasureText(0, ui.Span{Text: num, Size: size})
		p.Text(x0-g*0.55-w, y0+at-h/2, num, size, ink.Alpha(0.75))
		p.Text(x0+span+g*0.55, y0+at-h/2, num, size, ink.Alpha(0.75))
	}
	for y := range n {
		for x := range n {
			if st := b.At(Point{x, y}); st != Empty {
				stone(p, x0+float32(x)*g, y0+float32(y)*g, g, st, 1)
			}
		}
	}
	if ghost {
		stone(p, x0+float32(hover.X)*g, y0+float32(hover.Y)*g, g, o.Turn, 0.5)
	}
	if o.Last != nil && b.At(*o.Last) != Empty {
		var ring ui.Path
		color := ui.RGB(255, 255, 255)
		if b.At(*o.Last) == White {
			color = ui.RGB(20, 20, 20)
		}
		p.StrokePath(ring.Circle(x0+float32(o.Last.X)*g, y0+float32(o.Last.Y)*g, g*0.22), max(1.5, g*0.06), color)
	}
}

// stone draws a stone at x, y on lines gap apart, lit from the top left,
// alpha opaque.
func stone(p *ui.Painter, x, y, gap float32, s Stone, alpha float32) {
	r := gap * 0.48
	box := ui.Rect{X: x - r, Y: y - r, W: 2 * r, H: 2 * r}
	if alpha == 1 {
		p.Shadow(box, r, gap*0.04, gap*0.06, gap*0.1, 0, ui.RGB(0, 0, 0).Alpha(0.35))
	}
	var disc ui.Path
	disc.Circle(x, y, r)
	if s == Black {
		p.FillPathGradient(&disc, ui.LinearGradient{From: ui.RGB(85, 85, 85).Alpha(alpha), To: ui.RGB(10, 10, 10).Alpha(alpha), Angle: 135})
		return
	}
	p.FillPathGradient(&disc, ui.LinearGradient{From: ui.RGB(255, 255, 255).Alpha(alpha), To: ui.RGB(205, 205, 200).Alpha(alpha), Angle: 135})
	p.StrokePath(&disc, max(0.5, gap*0.02), ui.RGB(120, 120, 110).Alpha(alpha*0.6))
}
