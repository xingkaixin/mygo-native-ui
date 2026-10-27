// Package xiangqi shows a position of Chinese chess, 象棋, as xiangqi
// apps do: the pieces on discs at the points of nine files and ten
// ranks, the river between the sides, the palaces' diagonals and marks
// where the cannons and soldiers start, with the files numbered along the
// edges, 九 to 一 on red's side and 1 to 9 on black's. A piece moves by
// dragging it, or by clicking it and then its point; the points it may go
// to show dots, and those where it takes a piece rings. The last move and
// a general in check are lit, and the board turns for black.
//
// The board knows no rules: the app says where each piece may go, and
// plays the moves it reports.
//
//	ev := xiangqi.View(c, &app.board, app.pos, xiangqi.Options{Legal: app.legal, Last: app.last})
//	if ev.Moved {
//		app.play(ev.Move)
//	}
package xiangqi

import (
	"errors"
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
)

// Piece is a piece by its letter in xiangqi's FEN: K the general, A an
// advisor, B an elephant, N a horse, R a chariot, C a cannon and P a
// soldier, upper case red's and lower case black's; 0 is an empty point.
type Piece byte

// Red reports whether the piece is red's.
func (p Piece) Red() bool { return p >= 'A' && p <= 'Z' }

// names are the characters the pieces show, red's and black's.
var names = map[Piece]string{
	'K': "帥", 'A': "仕", 'B': "相", 'N': "傌", 'R': "俥", 'C': "炮", 'P': "兵",
	'k': "將", 'a': "士", 'b': "象", 'n': "馬", 'r': "車", 'c': "砲", 'p': "卒",
}

// Point is a point of the board: 0 is a0 at red's left, 8 i0, 9 a1 and
// 89 i9 at black's right.
type Point int

// At returns the point on file f, from 0 to 8, and rank r, from 0 to 9.
func At(f, r int) Point { return Point(r*9 + f) }

func (p Point) File() int { return int(p) % 9 }
func (p Point) Rank() int { return int(p) / 9 }

// String returns the point as ICCS names it, as "h2".
func (p Point) String() string { return string(rune('a'+p.File())) + strconv.Itoa(p.Rank()) }

// Position is the pieces on the board's points.
type Position [90]Piece

// Start is the position xiangqi starts from.
var Start, _ = ParseFEN("rnbakabnr/9/1c5c1/p1p1p1p1p/9/9/P1P1P1P1P/1C5C1/9/RNBAKABNR")

// ParseFEN parses the placement of the pieces, the first field of a FEN,
// from black's rank 9 down; the fields after it are left for the app.
func ParseFEN(fen string) (Position, error) {
	var p Position
	ranks := strings.Split(strings.Fields(fen + " ")[0], "/")
	if len(ranks) != 10 {
		return p, errors.New("xiangqi: a FEN has 10 ranks")
	}
	for i, rank := range ranks {
		f := 0
		for _, ch := range rank {
			switch {
			case ch >= '1' && ch <= '9':
				f += int(ch - '0')
			case names[Piece(ch)] != "" && f < 9:
				p[At(f, 9-i)] = Piece(ch)
				f++
			default:
				return p, errors.New("xiangqi: " + strconv.Quote(rank) + " is not a rank of a FEN")
			}
		}
		if f != 9 {
			return p, errors.New("xiangqi: " + strconv.Quote(rank) + " is not 9 points")
		}
	}
	return p, nil
}

// FEN returns the placement of the pieces, as the first field of a FEN.
func (p Position) FEN() string {
	var b strings.Builder
	for r := 9; r >= 0; r-- {
		empty := 0
		for f := range 9 {
			pc := p[At(f, r)]
			if pc == 0 {
				empty++
				continue
			}
			if empty > 0 {
				b.WriteByte(byte('0' + empty))
				empty = 0
			}
			b.WriteByte(byte(pc))
		}
		if empty > 0 {
			b.WriteByte(byte('0' + empty))
		}
		if r > 0 {
			b.WriteByte('/')
		}
	}
	return b.String()
}

// Move is a move of a piece from a point to another.
type Move struct{ From, To Point }

// String returns the move as ICCS writes it, as "h2e2".
func (m Move) String() string { return m.From.String() + m.To.String() }

// Options are how a board shows and what may move.
type Options struct {
	// Flipped shows black's side at the bottom.
	Flipped bool
	// Legal returns the points the piece on from may go to; a piece with
	// none does not move. Nil lets every piece go anywhere.
	Legal func(from Point) []Point
	// Last is the move before, lit; one from a point to itself is none.
	Last Move
	// Check is the general in check, K or k, lit in red; 0 is none.
	Check Piece
}

// Events are what happened on the board since the last frame.
type Events struct {
	ui.Element
	// Moved reports a move, which the app plays on its position.
	Moved bool
	Move  Move
}

// State is what a board keeps: the piece chosen and the one dragged.
// Keep it in the app's state, one for each board.
type State struct {
	// chosen is the point chosen, plus 1; 0 none. again is a press on
	// it, which a click unchooses.
	chosen int
	again  bool
	// held is the point pressed, plus 1, hx, hy where, and px, py where
	// the pointer is, from the board's corner; dragging once it moved off
	// the press.
	held           int
	px, py, hx, hy float32
	dragging       bool
	played         *Move
	// gap is how far apart the lines are, and ox, oy where a9's point
	// shows, as of the last layout.
	gap, ox, oy float32
}

// View creates the board of pos, as big as the room it is given allows.
func View(c *ui.Context, s *State, pos Position, o Options) Events {
	ev := Events{}
	if s.played != nil {
		ev.Moved, ev.Move, s.played = true, *s.played, nil
	}
	board := ui.Box(c).Focusable().FocusRing(false).Role(ui.RoleGroup).Label("Xiangqi board").Value(pos.FEN())
	board.HandleInput(func(in ui.InputEvent) bool { return s.input(&pos, &o, in) })
	if board.Shortcut(0, ui.KeyEscape) {
		s.chosen = 0
	}
	b := board.LayoutBounds()
	// A line's gap of room around the edges holds the numbers.
	s.gap = float32(math.Floor(float64(min(b.W/10, b.H/11))))
	s.ox, s.oy = (b.W-s.gap*8)/2, (b.H-s.gap*9)/2
	if px, py, over := board.PointerPosition(); s.held != 0 && s.dragging {
		board.Cursor(ui.CursorGrabbing)
	} else if pt, on := s.pointAt(&o, px, py); over && on {
		if _, ok := targets(&pos, &o, pt); ok {
			board.Cursor(ui.CursorGrab)
		}
	}
	board.Draw(func(p *ui.Painter, r ui.Rect) { s.draw(p, r, &pos, &o) })
	ev.Element = board
	return ev
}

// shown returns the point in column col and row row, from the board's
// top left.
func shown(o *Options, col, row int) Point {
	if o.Flipped {
		return At(8-col, row)
	}
	return At(col, 9-row)
}

// pointAt returns the point nearest x, y from the board's corner, and
// whether it is within half a gap.
func (s *State) pointAt(o *Options, x, y float32) (Point, bool) {
	if s.gap <= 0 {
		return 0, false
	}
	col := int(math.Round(float64((x - s.ox) / s.gap)))
	row := int(math.Round(float64((y - s.oy) / s.gap)))
	if col < 0 || col > 8 || row < 0 || row > 9 {
		return 0, false
	}
	return shown(o, col, row), true
}

// center returns where pt shows, from the board's corner.
func (s *State) center(o *Options, pt Point) (float32, float32) {
	col, row := pt.File(), 9-pt.Rank()
	if o.Flipped {
		col, row = 8-col, 9-row
	}
	return s.ox + float32(col)*s.gap, s.oy + float32(row)*s.gap
}

func targets(pos *Position, o *Options, from Point) ([]Point, bool) {
	if pos[from] == 0 {
		return nil, false
	}
	if o.Legal == nil {
		return nil, true
	}
	to := o.Legal(from)
	return to, len(to) > 0
}

func allowed(pos *Position, o *Options, from, to Point) bool {
	if from == to {
		return false
	}
	t, ok := targets(pos, o, from)
	return ok && (o.Legal == nil || slices.Contains(t, to))
}

func (s *State) input(pos *Position, o *Options, in ui.InputEvent) bool {
	switch in.Kind {
	case ui.InputPointerDown:
		if in.Button != 0 {
			return false
		}
		pt, on := s.pointAt(o, in.X, in.Y)
		if !on {
			s.chosen = 0
			return true
		}
		if s.chosen != 0 && allowed(pos, o, Point(s.chosen-1), pt) {
			s.play(Point(s.chosen-1), pt)
			return true
		}
		s.again = s.chosen == int(pt)+1
		s.chosen = 0
		if _, ok := targets(pos, o, pt); ok {
			s.held, s.dragging, s.chosen = int(pt)+1, false, int(pt)+1
			s.hx, s.hy, s.px, s.py = in.X, in.Y, in.X, in.Y
		}
	case ui.InputPointerMove:
		if s.held == 0 {
			return false
		}
		s.px, s.py = in.X, in.Y
		if math.Hypot(float64(in.X-s.hx), float64(in.Y-s.hy)) > 4 {
			s.dragging = true
		}
	case ui.InputPointerUp:
		if s.held == 0 {
			return false
		}
		from := Point(s.held - 1)
		if !s.dragging && s.again {
			s.chosen = 0
		}
		if s.dragging {
			if pt, on := s.pointAt(o, in.X, in.Y); on && allowed(pos, o, from, pt) {
				s.play(from, pt)
			} else if pt != from {
				s.chosen = 0
			}
		}
		s.held, s.dragging = 0, false
	default:
		return false
	}
	return true
}

func (s *State) play(from, to Point) {
	s.chosen = 0
	s.played = &Move{From: from, To: to}
}

var (
	wood  = ui.Hex("#eccf98")
	ink   = ui.Hex("#6b3d1f")
	red   = ui.Hex("#b8261d")
	black = ui.Hex("#1f1f1f")
)

func (s *State) draw(p *ui.Painter, r ui.Rect, pos *Position, o *Options) {
	g := s.gap
	if g <= 0 {
		return
	}
	x0, y0 := r.X+s.ox, r.Y+s.oy
	xy := func(pt Point) (float32, float32) {
		x, y := s.center(o, pt)
		return r.X + x, r.Y + y
	}
	frame := ui.Rect{X: x0 - g*0.95, Y: y0 - g*0.95, W: g*8 + g*1.9, H: g*9 + g*1.9}
	p.Shadow(frame, 4, 0, 2, 10, 0, ui.RGB(0, 0, 0).Alpha(0.3))
	p.Fill(frame, wood, 4)
	line := max(1, g/40)
	p.Stroke(ui.Rect{X: x0 - g*0.12, Y: y0 - g*0.12, W: g*8 + g*0.24, H: g*9 + g*0.24}, ink, 0, line*2)
	for i := range 10 {
		y := y0 + float32(i)*g
		p.Line(x0, y, x0+g*8, y, line, ink)
	}
	for i := range 9 {
		x := x0 + float32(i)*g
		if i == 0 || i == 8 {
			p.Line(x, y0, x, y0+g*9, line, ink)
			continue
		}
		p.Line(x, y0, x, y0+g*4, line, ink)
		p.Line(x, y0+g*5, x, y0+g*9, line, ink)
	}
	for _, top := range []float32{0, 7} {
		y := y0 + top*g
		p.StrokePolyline([][2]float32{{x0 + 3*g, y}, {x0 + 5*g, y + 2*g}}, line, ink)
		p.StrokePolyline([][2]float32{{x0 + 5*g, y}, {x0 + 3*g, y + 2*g}}, line, ink)
	}
	for _, m := range []Point{At(1, 2), At(7, 2), At(1, 7), At(7, 7), At(0, 3), At(2, 3), At(4, 3), At(6, 3), At(8, 3),
		At(0, 6), At(2, 6), At(4, 6), At(6, 6), At(8, 6)} {
		x, y := xy(m)
		corners(p, x, y, g, m.File() > 0, m.File() < 8, line, ink)
	}
	river := []string{"楚 河", "漢 界"}
	if o.Flipped {
		river[0], river[1] = river[1], river[0]
	}
	size := g * 0.5
	for i, text := range river {
		w, h := p.MeasureText(0, ui.Span{Text: text, Size: size})
		cx := x0 + g*2 + float32(i)*g*4
		p.Text(cx-w/2, y0+g*4.5-h/2, text, size, ink.Alpha(0.8))
	}
	chinese := []string{"九", "八", "七", "六", "五", "四", "三", "二", "一"}
	num := g * 0.28
	for col := range 9 {
		bottom, top := chinese[col], strconv.Itoa(col+1)
		if o.Flipped {
			bottom, top = strconv.Itoa(col+1), chinese[col]
		}
		x := x0 + float32(col)*g
		w, h := p.MeasureText(0, ui.Span{Text: bottom, Size: num})
		p.Text(x-w/2, y0+g*9+g*0.72-h/2, bottom, num, ink)
		w, h = p.MeasureText(0, ui.Span{Text: top, Size: num})
		p.Text(x-w/2, y0-g*0.72-h/2, top, num, ink)
	}
	lit := ui.RGBA(40, 120, 220, 0.45)
	if o.Last.From != o.Last.To {
		for _, pt := range []Point{o.Last.From, o.Last.To} {
			x, y := xy(pt)
			ring(p, x, y, g*0.47, line*2, lit)
		}
	}
	held := Point(-1)
	if s.held != 0 && s.dragging {
		held = Point(s.held - 1)
	}
	for pt := Point(0); pt < 90; pt++ {
		pc := pos[pt]
		if pc == 0 || pt == held {
			continue
		}
		x, y := xy(pt)
		if o.Check != 0 && pc == o.Check {
			var glow ui.Path
			p.Shadow(ui.Rect{X: x - g*0.42, Y: y - g*0.42, W: g * 0.84, H: g * 0.84}, g*0.42, 0, 0, g*0.25, g*0.06, ui.RGB(230, 30, 30))
			p.FillPath(glow.Circle(x, y, g*0.46), ui.RGB(230, 30, 30))
		}
		piece(p, pc, x, y, g, s.chosen == int(pt)+1)
	}
	if s.chosen != 0 {
		if to, ok := targets(pos, o, Point(s.chosen-1)); ok {
			for _, pt := range to {
				x, y := xy(pt)
				var dot ui.Path
				if pos[pt] != 0 {
					p.StrokePath(dot.Circle(x, y, g*0.47), g*0.07, ui.RGBA(20, 110, 40, 0.7))
				} else {
					p.FillPath(dot.Circle(x, y, g*0.13), ui.RGBA(20, 110, 40, 0.7))
				}
			}
		}
	}
	if held >= 0 {
		piece(p, pos[held], r.X+s.px, r.Y+s.py, g*1.1, true)
	}
}

// corners draws the marks around a point where a cannon or a soldier
// starts, on its left and right sides unless at an edge.
func corners(p *ui.Painter, x, y, g float32, left, right bool, w float32, c ui.Color) {
	d, l := g*0.08, g*0.18
	for _, sx := range []float32{-1, 1} {
		if sx < 0 && !left || sx > 0 && !right {
			continue
		}
		for _, sy := range []float32{-1, 1} {
			cx, cy := x+sx*d, y+sy*d
			p.Line(cx, cy, cx+sx*l, cy, w, c)
			p.Line(cx, cy, cx, cy+sy*l, w, c)
		}
	}
}

func ring(p *ui.Painter, x, y, radius, w float32, c ui.Color) {
	var path ui.Path
	p.StrokePath(path.Circle(x, y, radius), w, c)
}

// piece draws pc on a disc at x, y on lines gap apart, raised when
// lifted.
func piece(p *ui.Painter, pc Piece, x, y, gap float32, lifted bool) {
	rad := gap * 0.44
	color := black
	if pc.Red() {
		color = red
	}
	box := ui.Rect{X: x - rad, Y: y - rad, W: 2 * rad, H: 2 * rad}
	shadow := gap * 0.06
	if lifted {
		shadow = gap * 0.16
	}
	p.Shadow(box, rad, 0, shadow*0.6, shadow, 0, ui.RGB(0, 0, 0).Alpha(0.4))
	var disc ui.Path
	disc.Circle(x, y, rad)
	p.FillPathGradient(&disc, ui.LinearGradient{From: ui.Hex("#fbf1dc"), To: ui.Hex("#e2c48f"), Angle: 135})
	ring(p, x, y, rad-gap*0.02, gap*0.03, ui.Hex("#a8824a"))
	ring(p, x, y, rad*0.8, gap*0.025, color)
	name := names[pc]
	size := gap * 0.5
	w, h := p.MeasureText(0, ui.Span{Text: name, Size: size, Weight: 700})
	p.RichText(x-w/2, y-h/2, w+1, ui.Span{Text: name, Size: size, Weight: 700, Color: color})
}
