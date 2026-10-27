// Package chessboard shows a chess position on a board, as chess sites
// do: a piece moves by dragging it, or by clicking it and then its
// square; the squares it may go to show dots, and those where it takes a
// piece rings. The last move and a king in check are lit. A pawn reaching
// the last rank chooses its piece from a column over the square. Arrows
// drawn with the right button and circles from a right click mark the
// board, and a click clears them. Files and ranks run along the edges, and
// the board turns for black.
//
// The board knows no rules: the app says where each piece may go, and
// plays the moves it reports, with castling's rook and en passant's pawn.
//
//	ev := chessboard.View(c, &app.board, app.pos, chessboard.Options{
//		Legal: app.legal, Last: app.last,
//	})
//	if ev.Moved {
//		app.play(ev.Move)
//	}
package chessboard

import (
	"errors"
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
)

// Piece is a piece by its letter in FEN: K, Q, R, B, N and P for white's,
// k, q, r, b, n and p for black's; 0 is an empty square.
type Piece byte

// White reports whether the piece is white's.
func (p Piece) White() bool { return p >= 'A' && p <= 'Z' }

// Square is a square of the board: 0 is a1, 7 h1, 8 a2 and 63 h8.
type Square int

// At returns the square on file f and rank r, both from 0.
func At(f, r int) Square { return Square(r*8 + f) }

func (s Square) File() int { return int(s) % 8 }
func (s Square) Rank() int { return int(s) / 8 }

// String returns the square's name, as "e4".
func (s Square) String() string { return string(rune('a'+s.File())) + strconv.Itoa(s.Rank()+1) }

// Position is the pieces on the board's squares.
type Position [64]Piece

// Start is the position chess starts from.
var Start, _ = ParseFEN("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR")

// ParseFEN parses the placement of the pieces, the first field of a FEN;
// the fields after it are left for the app.
func ParseFEN(fen string) (Position, error) {
	var p Position
	ranks := strings.Split(strings.Fields(fen + " ")[0], "/")
	if len(ranks) != 8 {
		return p, errors.New("chessboard: a FEN has 8 ranks")
	}
	for i, rank := range ranks {
		f := 0
		for _, ch := range rank {
			switch {
			case ch >= '1' && ch <= '8':
				f += int(ch - '0')
			case strings.ContainsRune("KQRBNPkqrbnp", ch) && f < 8:
				p[At(f, 7-i)] = Piece(ch)
				f++
			default:
				return p, errors.New("chessboard: " + strconv.Quote(rank) + " is not a rank of a FEN")
			}
		}
		if f != 8 {
			return p, errors.New("chessboard: " + strconv.Quote(rank) + " is not 8 squares")
		}
	}
	return p, nil
}

// FEN returns the placement of the pieces, as the first field of a FEN.
func (p Position) FEN() string {
	var b strings.Builder
	for r := 7; r >= 0; r-- {
		empty := 0
		for f := range 8 {
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

// Move is a move of a piece from a square to another, and the piece a
// pawn becomes on the last rank.
type Move struct {
	From, To  Square
	Promotion Piece
}

// String returns the move as UCI writes it, as "e2e4" or "e7e8q".
func (m Move) String() string {
	s := m.From.String() + m.To.String()
	if m.Promotion != 0 {
		s += strings.ToLower(string(rune(m.Promotion)))
	}
	return s
}

// Options are how a board shows and what may move.
type Options struct {
	// Flipped shows black's side at the bottom.
	Flipped bool
	// Legal returns the squares the piece on from may go to; a piece with
	// none does not move. Nil lets every piece go anywhere.
	Legal func(from Square) []Square
	// Last is the move before, lit; one from a square to itself is none.
	Last Move
	// Check is the king in check, K or k, lit in red; 0 is none.
	Check Piece
	// Light and Dark are the squares' colors; zero ones are wood's.
	Light, Dark ui.Color
}

// Events are what happened on the board since the last frame.
type Events struct {
	ui.Element
	// Moved reports a move, which the app plays on its position.
	Moved bool
	Move  Move
}

// State is what a board keeps: the piece chosen, the one dragged, the
// marks and the pawn choosing its piece. Keep it in the app's state, one
// for each board.
type State struct {
	// chosen is the square chosen, plus 1; 0 none.
	chosen int
	// held is the square pressed, plus 1, hx, hy where, and px, py where
	// the pointer is, from the board's corner; dragging once it moved off
	// the press. again is a press on the piece chosen, which a click
	// unchooses.
	held            int
	px, py, hx, hy  float32
	dragging, again bool
	// marking is the square the right button went down on, plus 1.
	marking int
	arrows  [][2]Square
	circles []Square
	// promoting is the pawn's move to the last rank, waiting for its
	// piece, when promote.
	promoting Move
	promote   bool
	played    *Move
	// square and corner are the board's square size and top left, as of
	// the last layout.
	square, ox, oy float32
	glyphs         map[glyphKey][]ui.Glyph
}

type glyphKey struct {
	piece Piece
	size  float32
}

// glyphs are the pieces' symbols in Unicode, the filled ones for both
// sides; the pawn asks for text over emoji.
var glyphs = map[byte]string{'k': "♚", 'q': "♛", 'r': "♜", 'b': "♝", 'n': "♞", 'p': "♟︎"}

// View creates the board of pos, as big as the room it is given allows.
func View(c *ui.Context, s *State, pos Position, o Options) Events {
	t := c.Theme()
	if o.Light == (ui.Color{}) {
		o.Light, o.Dark = ui.Hex("#f0d9b5"), ui.Hex("#b58863")
	}
	ev := Events{}
	if s.played != nil {
		ev.Moved, ev.Move, s.played = true, *s.played, nil
	}
	board := ui.Box(c).Focusable().FocusRing(false).Role(ui.RoleGroup).Label("Chessboard").Value(pos.FEN())
	board.HandleInput(func(in ui.InputEvent) bool { return s.input(&pos, &o, in) })
	if board.Shortcut(0, ui.KeyEscape) {
		s.chosen, s.promote = 0, false
	}
	b := board.LayoutBounds()
	s.square = float32(math.Floor(float64(min(b.W, b.H) / 8)))
	s.ox, s.oy = (b.W-s.square*8)/2, (b.H-s.square*8)/2
	if px, py, over := board.PointerPosition(); s.held != 0 && s.dragging {
		board.Cursor(ui.CursorGrabbing)
	} else if sq, on := s.squareAt(&o, px, py); over && on && !s.promote {
		if _, ok := targets(&pos, &o, sq); ok {
			board.Cursor(ui.CursorGrab)
		}
	}
	board.Draw(func(p *ui.Painter, r ui.Rect) { s.draw(p, t, r, &pos, &o) })
	ev.Element = board
	return ev
}

// square returns the square at x, y from the board's corner, and whether
// it is on the board.
func (s *State) squareAt(o *Options, x, y float32) (Square, bool) {
	if s.square <= 0 {
		return 0, false
	}
	f := int(math.Floor(float64((x - s.ox) / s.square)))
	r := 7 - int(math.Floor(float64((y-s.oy)/s.square)))
	if f < 0 || f > 7 || r < 0 || r > 7 {
		return 0, false
	}
	if o.Flipped {
		f, r = 7-f, 7-r
	}
	return At(f, r), true
}

// rect returns where sq shows, from the board's corner.
func (s *State) rect(o *Options, sq Square) ui.Rect {
	f, r := sq.File(), 7-sq.Rank()
	if o.Flipped {
		f, r = 7-f, 7-r
	}
	return ui.Rect{X: s.ox + float32(f)*s.square, Y: s.oy + float32(r)*s.square, W: s.square, H: s.square}
}

// targets returns the squares the piece on from may go to, and whether
// it moves at all.
func targets(pos *Position, o *Options, from Square) ([]Square, bool) {
	if pos[from] == 0 {
		return nil, false
	}
	if o.Legal == nil {
		return nil, true
	}
	to := o.Legal(from)
	return to, len(to) > 0
}

func allowed(pos *Position, o *Options, from, to Square) bool {
	if from == to {
		return false
	}
	t, ok := targets(pos, o, from)
	return ok && (o.Legal == nil || slices.Contains(t, to))
}

// promotions are the pieces a pawn may become, from the last rank in.
var promotions = []byte{'q', 'n', 'r', 'b'}

// promotionAt returns the piece of the column of choices at sq, and
// whether sq holds one.
func (s *State) promotionAt(sq Square) (Piece, bool) {
	to := s.promoting.To
	if sq.File() != to.File() {
		return 0, false
	}
	i := to.Rank() - sq.Rank()
	if to.Rank() == 0 {
		i = sq.Rank()
	}
	if i < 0 || i >= len(promotions) {
		return 0, false
	}
	p := Piece(promotions[i])
	if to.Rank() == 7 {
		p -= 'a' - 'A'
	}
	return p, true
}

func (s *State) input(pos *Position, o *Options, in ui.InputEvent) bool {
	switch in.Kind {
	case ui.InputPointerDown:
		sq, on := s.squareAt(o, in.X, in.Y)
		if in.Button == 1 {
			if on {
				s.marking = int(sq) + 1
			}
			return true
		}
		if in.Button != 0 {
			return false
		}
		s.arrows, s.circles = nil, nil
		if s.promote {
			s.promote = false
			if pc, ok := s.promotionAt(sq); on && ok {
				m := s.promoting
				m.Promotion = pc
				s.played = &m
			}
			return true
		}
		if !on {
			s.chosen = 0
			return true
		}
		if s.chosen != 0 && allowed(pos, o, Square(s.chosen-1), sq) {
			s.play(pos, Square(s.chosen-1), sq)
			return true
		}
		s.again = s.chosen == int(sq)+1
		s.chosen = 0
		if _, ok := targets(pos, o, sq); ok {
			s.held, s.dragging = int(sq)+1, false
			s.hx, s.hy, s.px, s.py = in.X, in.Y, in.X, in.Y
			s.chosen = int(sq) + 1
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
		if in.Button == 1 {
			if s.marking != 0 {
				if sq, on := s.squareAt(o, in.X, in.Y); on {
					s.mark(Square(s.marking-1), sq)
				}
			}
			s.marking = 0
			return true
		}
		if s.held == 0 {
			return false
		}
		from := Square(s.held - 1)
		if !s.dragging && s.again {
			s.chosen = 0
		}
		if s.dragging {
			if sq, on := s.squareAt(o, in.X, in.Y); on && allowed(pos, o, from, sq) {
				s.play(pos, from, sq)
			} else if sq != from {
				s.chosen = 0
			}
		}
		s.held, s.dragging = 0, false
	default:
		return false
	}
	return true
}

// mark draws an arrow from a to b, or a circle on a when b is a, or takes
// away the one there.
func (s *State) mark(a, b Square) {
	if a == b {
		if i := slices.Index(s.circles, a); i >= 0 {
			s.circles = slices.Delete(s.circles, i, i+1)
		} else {
			s.circles = append(s.circles, a)
		}
		return
	}
	arrow := [2]Square{a, b}
	if i := slices.Index(s.arrows, arrow); i >= 0 {
		s.arrows = slices.Delete(s.arrows, i, i+1)
	} else {
		s.arrows = append(s.arrows, arrow)
	}
}

// play reports the move, or has a pawn reaching the last rank choose its
// piece first.
func (s *State) play(pos *Position, from, to Square) {
	s.chosen = 0
	m := Move{From: from, To: to}
	if p := pos[from]; (p == 'P' && to.Rank() == 7) || (p == 'p' && to.Rank() == 0) {
		s.promoting, s.promote = m, true
		return
	}
	s.played = &m
}

func (s *State) glyph(p Piece, size float32) []ui.Glyph {
	k := glyphKey{p, size}
	if g, ok := s.glyphs[k]; ok {
		return g
	}
	if s.glyphs == nil {
		s.glyphs = map[glyphKey][]ui.Glyph{}
	}
	g := ui.Shape(glyphs[byte(p)|0x20], ui.Font{Size: size})
	s.glyphs[k] = g
	return g
}

// piece draws p centered on x, y in a square size wide: the filled symbol
// in its side's color, outlined in the other's.
func (s *State) piece(pt *ui.Painter, p Piece, x, y, size float32) {
	g := s.glyph(p, size*0.84)
	if len(g) == 0 {
		return
	}
	w := g[len(g)-1].X + g[len(g)-1].Advance
	bx, by := x-w/2, y+size*0.29
	fill, edge := ui.RGB(28, 28, 28), ui.RGBA(255, 255, 255, 0.85)
	if p.White() {
		fill, edge = ui.RGB(255, 255, 255), ui.RGB(28, 28, 28)
	}
	k := max(1, size/36)
	for i := range 16 {
		a := float64(i) * math.Pi / 8
		pt.Glyphs(g, bx+k*float32(math.Cos(a)), by+k*float32(math.Sin(a)), edge)
	}
	pt.Glyphs(g, bx, by, fill)
}

func (s *State) draw(p *ui.Painter, t *ui.Theme, r ui.Rect, pos *Position, o *Options) {
	q := s.square
	if q <= 0 {
		return
	}
	at := func(sq Square) ui.Rect {
		rc := s.rect(o, sq)
		rc.X, rc.Y = rc.X+r.X, rc.Y+r.Y
		return rc
	}
	board := ui.Rect{X: r.X + s.ox, Y: r.Y + s.oy, W: q * 8, H: q * 8}
	p.Shadow(board, 2, 0, 2, 8, 0, ui.RGB(0, 0, 0).Alpha(0.25))
	lit := ui.RGBA(205, 210, 60, 0.5)
	for sq := Square(0); sq < 64; sq++ {
		rc := at(sq)
		fill := o.Light
		if (sq.File()+sq.Rank())%2 == 0 {
			fill = o.Dark
		}
		p.Fill(rc, fill, 0)
		if o.Last.From != o.Last.To && (sq == o.Last.From || sq == o.Last.To) || s.chosen == int(sq)+1 {
			p.Fill(rc, lit, 0)
		}
		if o.Check != 0 && pos[sq] == o.Check {
			p.Clip(rc, 0, func() {
				core := ui.Rect{X: rc.X + q*0.3, Y: rc.Y + q*0.3, W: q * 0.4, H: q * 0.4}
				p.Shadow(core, q*0.2, 0, 0, q*0.35, q*0.1, ui.RGB(230, 30, 30))
				p.Fill(core, ui.RGB(230, 30, 30), q*0.2)
			})
		}
	}
	coord := q * 0.17
	for i := range 8 {
		bottom, left := s.shown(o, i, 7), s.shown(o, 0, i)
		rb, rl := at(bottom), at(left)
		p.Text(rb.X+rb.W-coord*0.9, rb.Y+rb.H-coord*1.35, string(rune('a'+bottom.File())), coord, ink(bottom, o))
		p.Text(rl.X+coord*0.35, rl.Y+coord*0.25, strconv.Itoa(left.Rank()+1), coord, ink(left, o))
	}
	if s.chosen != 0 {
		from := Square(s.chosen - 1)
		if to, ok := targets(pos, o, from); ok {
			for _, sq := range to {
				rc := at(sq)
				var path ui.Path
				if pos[sq] != 0 {
					path.Circle(rc.X+q/2, rc.Y+q/2, q*0.46)
					p.StrokePath(&path, q*0.08, ui.RGBA(20, 85, 30, 0.45))
				} else {
					p.FillPath(path.Circle(rc.X+q/2, rc.Y+q/2, q*0.15), ui.RGBA(20, 85, 30, 0.45))
				}
			}
		}
	}
	held := Square(-1)
	if s.held != 0 && s.dragging {
		held = Square(s.held - 1)
	}
	for sq := Square(0); sq < 64; sq++ {
		if pc := pos[sq]; pc != 0 && sq != held && !(s.promote && sq == s.promoting.From) {
			rc := at(sq)
			s.piece(p, pc, rc.X+q/2, rc.Y+q/2, q)
		}
	}
	for _, sq := range s.circles {
		rc := at(sq)
		var path ui.Path
		p.StrokePath(path.Circle(rc.X+q/2, rc.Y+q/2, q*0.44), q*0.07, ui.RGBA(21, 120, 27, 0.75))
	}
	for _, a := range s.arrows {
		arrow(p, at(a[0]), at(a[1]), q)
	}
	if held >= 0 {
		s.piece(p, pos[held], r.X+s.px, r.Y+s.py, q*1.1)
	}
	if s.promote {
		p.Fill(board, ui.RGB(0, 0, 0).Alpha(0.35), 0)
		for sq := Square(0); sq < 64; sq++ {
			if pc, ok := s.promotionAt(sq); ok {
				rc := at(sq)
				p.Fill(rc, t.Surface, q/2)
				s.piece(p, pc, rc.X+q/2, rc.Y+q/2, q*0.9)
			}
		}
	}
}

// shown returns the square shown in column col and row row, from the
// board's top left.
func (s *State) shown(o *Options, col, row int) Square {
	if o.Flipped {
		return At(7-col, row)
	}
	return At(col, 7-row)
}

// ink returns the color of a coordinate on sq: the other squares'.
func ink(sq Square, o *Options) ui.Color {
	if (sq.File()+sq.Rank())%2 == 0 {
		return o.Light
	}
	return o.Dark
}

// arrow draws an arrow from the middle of a to that of b, a square q wide,
// its head ending short of b's middle.
func arrow(p *ui.Painter, a, b ui.Rect, q float32) {
	x0, y0 := a.X+q/2, a.Y+q/2
	x1, y1 := b.X+q/2, b.Y+q/2
	dx, dy := x1-x0, y1-y0
	l := float32(math.Hypot(float64(dx), float64(dy)))
	ux, uy := dx/l, dy/l
	head, half := q*0.42, q*0.3
	x1, y1 = x1-ux*q*0.1, y1-uy*q*0.1
	bx, by := x1-ux*head, y1-uy*head
	color := ui.RGBA(21, 120, 27, 0.75)
	p.StrokePolyline([][2]float32{{x0 + ux*q*0.2, y0 + uy*q*0.2}, {bx, by}}, q*0.16, color)
	var tip ui.Path
	tip.MoveTo(x1, y1).LineTo(bx-uy*half, by+ux*half).LineTo(bx+uy*half, by-ux*half).Close()
	p.FillPath(&tip, color)
}
