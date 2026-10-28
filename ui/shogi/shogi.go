// Package shogi shows a position of Japanese chess, 将棋, as shogi apps
// do: the pentagonal pieces on a nine by nine board, pointing at the
// opponent, the promoted ones in red, each side's captured pieces in hand
// beside it, gote's above and sente's below, and the files numbered 9 to
// 1 across the top and the ranks 一 to 九 down the side. A piece moves by
// dragging it, or by clicking it and then its square, and one in hand
// drops the same way; the squares it may go to show dots. A move that may
// promote asks, with the piece both ways at its square, and one that must
// promotes. The last move and a king in check are lit, and the board
// turns for gote.
//
// The board knows no rules beyond that: the app says where each piece may
// go, and plays the moves it reports.
//
//	ev := shogi.View(c, &app.board, app.pos, shogi.Options{Legal: app.legal, Last: app.last})
//	if ev.Moved {
//		app.pos.Play(ev.Move)
//	}
package shogi

import (
	"errors"
	"math"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"github.com/egoist/mygo/ui"
)

// Kinds are the kinds of pieces, by their letters in SFEN: king, rook,
// bishop, gold, silver, knight, lance and pawn. A hand counts them from
// the rook, as Kinds[1:] has them.
const Kinds = "KRBGSNLP"

// Piece is a piece: its kind, a letter of Kinds, whether it is gote's,
// and whether it is promoted. The zero Piece is an empty square.
type Piece struct {
	Kind     byte
	Gote     bool
	Promoted bool
}

var names = map[byte][2]string{
	'K': {"玉", ""}, 'R': {"飛", "龍"}, 'B': {"角", "馬"}, 'G': {"金", ""},
	'S': {"銀", "全"}, 'N': {"桂", "圭"}, 'L': {"香", "杏"}, 'P': {"歩", "と"},
}

func (p Piece) name() string {
	if p.Kind == 'K' && p.Gote {
		return "王"
	}
	if p.Promoted {
		return names[p.Kind][1]
	}
	return names[p.Kind][0]
}

// promotes reports whether the piece may turn over.
func (p Piece) promotes() bool { return !p.Promoted && names[p.Kind][1] != "" }

// Square is a square of the board: 0 is 9a, at gote's right as sente
// sees it top left, 8 is 1a, and 80 is 1i.
type Square int

// At returns the square on file f and rank r, both from 1 to 9.
func At(f, r int) Square { return Square((r-1)*9 + 9 - f) }

func (s Square) File() int { return 9 - int(s)%9 }
func (s Square) Rank() int { return int(s)/9 + 1 }

// String returns the square as USI names it, as "7g".
func (s Square) String() string { return strconv.Itoa(s.File()) + string(rune('a'+s.Rank()-1)) }

// Position is the pieces on the board, those in each side's hand, sente's
// first, and whether gote is to move.
type Position struct {
	Board [81]Piece
	Hands [2][7]int
	Gote  bool
}

// Start is the position shogi starts from.
var Start, _ = ParseSFEN("lnsgkgsnl/1r5b1/ppppppppp/9/9/9/PPPPPPPPP/1B5R1/LNSGKGSNL b -")

// ParseSFEN parses the board, the side to move and the hands of an SFEN;
// the move number after them is left for the app.
func ParseSFEN(sfen string) (Position, error) {
	var p Position
	fields := strings.Fields(sfen)
	if len(fields) < 1 {
		return p, errors.New("shogi: an SFEN has a board")
	}
	ranks := strings.Split(fields[0], "/")
	if len(ranks) != 9 {
		return p, errors.New("shogi: an SFEN has 9 ranks")
	}
	for r, rank := range ranks {
		col, promoted := 0, false
		for _, ch := range rank {
			switch {
			case ch >= '1' && ch <= '9':
				col += int(ch - '0')
			case ch == '+':
				promoted = true
				continue
			case ch < 0x80 && unicode.IsLetter(ch) && strings.IndexByte(Kinds, byte(ch)&^0x20) >= 0 && col < 9:
				p.Board[r*9+col] = Piece{Kind: byte(ch) &^ 0x20, Gote: ch >= 'a', Promoted: promoted}
				col++
			default:
				return p, errors.New("shogi: " + strconv.Quote(rank) + " is not a rank of an SFEN")
			}
			promoted = false
		}
	}
	if len(fields) > 1 {
		p.Gote = fields[1] == "w"
	}
	if len(fields) > 2 && fields[2] != "-" {
		n := 0
		for _, ch := range fields[2] {
			if ch >= '0' && ch <= '9' {
				n = n*10 + int(ch-'0')
				continue
			}
			i := strings.IndexByte(Kinds, byte(ch)&^0x20)
			if i < 1 {
				return p, errors.New("shogi: " + strconv.Quote(fields[2]) + " is not a hand of an SFEN")
			}
			side := 0
			if ch >= 'a' {
				side = 1
			}
			p.Hands[side][i-1] += max(n, 1)
			n = 0
		}
	}
	return p, nil
}

// SFEN returns the board, the side to move and the hands, as the first
// three fields of an SFEN.
func (p Position) SFEN() string {
	var b strings.Builder
	for r := range 9 {
		empty := 0
		for col := range 9 {
			pc := p.Board[r*9+col]
			if pc.Kind == 0 {
				empty++
				continue
			}
			if empty > 0 {
				b.WriteByte(byte('0' + empty))
				empty = 0
			}
			b.WriteString(pc.letter())
		}
		if empty > 0 {
			b.WriteByte(byte('0' + empty))
		}
		if r < 8 {
			b.WriteByte('/')
		}
	}
	b.WriteString(map[bool]string{false: " b ", true: " w "}[p.Gote])
	hand := ""
	for side := range 2 {
		for i, n := range p.Hands[side] {
			if n == 0 {
				continue
			}
			if n > 1 {
				hand += strconv.Itoa(n)
			}
			hand += Piece{Kind: Kinds[i+1], Gote: side == 1}.letter()
		}
	}
	if hand == "" {
		hand = "-"
	}
	b.WriteString(hand)
	return b.String()
}

func (p Piece) letter() string {
	s := string(p.Kind)
	if p.Gote {
		s = strings.ToLower(s)
	}
	if p.Promoted {
		s = "+" + s
	}
	return s
}

// Move is a move of a piece from a square to another, turning it over
// when Promote, or a drop of a piece of kind Drop from the hand of the
// side to move.
type Move struct {
	From, To Square
	Drop     byte
	Promote  bool
}

// String returns the move as USI writes it, as "7g7f", "8h2b+" or "P*5e".
func (m Move) String() string {
	if m.Drop != 0 {
		return string(m.Drop) + "*" + m.To.String()
	}
	s := m.From.String() + m.To.String()
	if m.Promote {
		s += "+"
	}
	return s
}

// Play plays m for the side to move: a piece taken goes to its hand,
// turned back, and the other side is to move.
func (p *Position) Play(m Move) {
	side := 0
	if p.Gote {
		side = 1
	}
	if m.Drop != 0 {
		p.Hands[side][strings.IndexByte(Kinds, m.Drop)-1]--
		p.Board[m.To] = Piece{Kind: m.Drop, Gote: p.Gote}
	} else {
		if taken := p.Board[m.To]; taken.Kind != 0 && taken.Kind != 'K' {
			p.Hands[side][strings.IndexByte(Kinds, taken.Kind)-1]++
		}
		pc := p.Board[m.From]
		pc.Promoted = pc.Promoted || m.Promote
		p.Board[m.To], p.Board[m.From] = pc, Piece{}
	}
	p.Gote = !p.Gote
}

// Options are how a board shows and what may move.
type Options struct {
	// Flipped shows gote's side at the bottom.
	Flipped bool
	// Legal returns the squares the piece on from may go to, or, when drop
	// is a kind, those a piece of it in the hand of the side to move may
	// drop on. A piece with none does not move. Nil lets every piece of
	// the side to move go to any square but its own pieces'.
	Legal func(from Square, drop byte) []Square
	// Last is the move before, lit; one from a square to itself is none.
	Last Move
	// Check lights the king of the side to move.
	Check bool
}

// Events are what happened on the board since the last frame.
type Events struct {
	ui.Element
	// Moved reports a move, which the app plays on its position.
	Moved bool
	Move  Move
}

// source is where a piece moves from: a square, or a side's hand, by the
// kind of piece it drops; ok is false for none.
type source struct {
	sq   Square
	drop byte
	gote bool
	ok   bool
}

type slot struct {
	kind byte
	r    ui.Rect
}

// State is what a board keeps: the piece chosen, the one dragged, and the
// move asking whether to promote. Keep it in the app's state, one for
// each board.
type State struct {
	chosen   source
	again    bool
	held     source
	dragging bool
	asking   *Move
	played   *Move
	// px, py are where the pointer is, and hx, hy where it pressed, from
	// the board's corner.
	px, py, hx, hy float32
	// cw and ch are the size of a square, ox, oy where the board's top
	// left shows, and hands the pieces in each side's hand, as of the last
	// layout.
	cw, ch, ox, oy float32
	hands          [2][]slot
}

// View creates the board of pos, as big as the room it is given allows.
func View(c *ui.Context, s *State, pos Position, o Options) Events {
	ev := Events{}
	if s.played != nil {
		ev.Moved, ev.Move, s.played = true, *s.played, nil
	}
	board := ui.Box(c).Focusable().FocusRing(false).Role(ui.RoleGroup).Label("Shogi board").Value(pos.SFEN())
	board.HandleInput(func(in ui.InputEvent) bool { return s.input(&pos, &o, in) })
	if board.Shortcut(0, ui.KeyEscape) {
		s.chosen, s.asking = source{}, nil
	}
	s.layout(board.LayoutBounds(), &pos, &o)
	if s.held.ok && s.dragging {
		board.Cursor(ui.CursorGrabbing)
	} else if px, py, over := board.PointerPosition(); over {
		if src, ok := s.hit(&o, px, py); ok && len(targets(&pos, &o, src)) > 0 {
			board.Cursor(ui.CursorGrab)
		}
	}
	board.Draw(func(p *ui.Painter, r ui.Rect) { s.draw(p, r, &pos, &o) })
	ev.Element = board
	return ev
}

// layout places the board and the hands in b, the board's squares a
// little taller than wide, as on a real board.
func (s *State) layout(b ui.Rect, pos *Position, o *Options) {
	s.cw = float32(math.Floor(float64(min(b.W/9.8, b.H/(9*1.09+3.6)))))
	s.ch = s.cw * 1.09
	hand := s.cw * 1.25
	s.ox = (b.W - s.cw*9.6) / 2
	s.oy = (b.H-s.ch*9-s.cw*1.2)/2 + hand*0.1 + s.cw*0.6
	for side := range 2 {
		s.hands[side] = s.hands[side][:0]
		for i, n := range pos.Hands[side] {
			if n > 0 {
				s.hands[side] = append(s.hands[side], slot{kind: Kinds[i+1]})
			}
		}
		bottom := side == 0 != o.Flipped
		w := s.cw * 1.05
		for i := range s.hands[side] {
			x := s.ox + float32(i)*w
			y := s.oy - s.cw*0.55 - hand
			if bottom {
				x = s.ox + s.cw*9 - float32(len(s.hands[side])-i)*w
				y = s.oy + s.ch*9 + s.cw*0.25
			}
			s.hands[side][i].r = ui.Rect{X: x, Y: y, W: w, H: hand}
		}
	}
}

// shown returns the square in column col and row row, from the board's
// top left.
func shown(o *Options, col, row int) Square {
	if o.Flipped {
		col, row = 8-col, 8-row
	}
	return Square(row*9 + col)
}

func (s *State) box(o *Options, sq Square) ui.Rect {
	col, row := int(sq)%9, int(sq)/9
	if o.Flipped {
		col, row = 8-col, 8-row
	}
	return ui.Rect{X: s.ox + float32(col)*s.cw, Y: s.oy + float32(row)*s.ch, W: s.cw, H: s.ch}
}

// square returns the square under x, y from the board's corner.
func (s *State) square(o *Options, x, y float32) (Square, bool) {
	if s.cw <= 0 || x < s.ox || y < s.oy {
		return 0, false
	}
	col, row := int((x-s.ox)/s.cw), int((y-s.oy)/s.ch)
	if col > 8 || row > 8 {
		return 0, false
	}
	return shown(o, col, row), true
}

// hit returns the piece of the side to move under x, y, on the board or
// in its hand.
func (s *State) hit(o *Options, x, y float32) (source, bool) {
	if sq, ok := s.square(o, x, y); ok {
		return source{sq: sq, ok: true}, true
	}
	for side := range 2 {
		for _, h := range s.hands[side] {
			if x >= h.r.X && x < h.r.X+h.r.W && y >= h.r.Y && y < h.r.Y+h.r.H {
				return source{drop: h.kind, gote: side == 1, ok: true}, true
			}
		}
	}
	return source{}, false
}

// targets returns where the piece of src may go: none unless it is the
// side to move's.
func targets(pos *Position, o *Options, src source) []Square {
	if src.drop != 0 {
		if src.gote != pos.Gote {
			return nil
		}
		if o.Legal != nil {
			return o.Legal(0, src.drop)
		}
		var out []Square
		for sq := Square(0); sq < 81; sq++ {
			if pos.Board[sq].Kind == 0 {
				out = append(out, sq)
			}
		}
		return out
	}
	pc := pos.Board[src.sq]
	if pc.Kind == 0 || pc.Gote != pos.Gote {
		return nil
	}
	if o.Legal != nil {
		return o.Legal(src.sq, 0)
	}
	var out []Square
	for sq := Square(0); sq < 81; sq++ {
		if q := pos.Board[sq]; q.Kind == 0 || q.Gote != pc.Gote {
			out = append(out, sq)
		}
	}
	return out
}

// far returns how many ranks from the far side sq is, for gote or sente:
// 1 on the far rank.
func far(sq Square, gote bool) int {
	if gote {
		return 10 - sq.Rank()
	}
	return sq.Rank()
}

// move plays a piece of src to sq, asking whether to promote where it may
// and promoting where it must.
func (s *State) move(pos *Position, src source, to Square) {
	s.chosen = source{}
	m := Move{From: src.sq, To: to, Drop: src.drop}
	if src.drop != 0 {
		s.played = &m
		return
	}
	pc := pos.Board[src.sq]
	if pc.promotes() && (far(src.sq, pc.Gote) <= 3 || far(to, pc.Gote) <= 3) {
		must := far(to, pc.Gote) == 1 && (pc.Kind == 'P' || pc.Kind == 'L' || pc.Kind == 'N') ||
			far(to, pc.Gote) == 2 && pc.Kind == 'N'
		if !must {
			s.asking = &m
			return
		}
		m.Promote = true
	}
	s.played = &m
}

// choice returns the two pieces asking whether to promote, the promoted
// one first, over the asking move's square and the one beside it toward
// the middle of the board.
func (s *State) choice(o *Options) [2]ui.Rect {
	r := s.box(o, s.asking.To)
	other := r
	if r.Y+r.H*2 <= s.oy+s.ch*9 {
		other.Y += r.H
	} else {
		other.Y -= r.H
		r, other = other, r
	}
	return [2]ui.Rect{r, other}
}

func (s *State) input(pos *Position, o *Options, in ui.InputEvent) bool {
	switch in.Kind {
	case ui.InputPointerDown:
		if in.Button != 0 {
			return false
		}
		if s.asking != nil {
			m, choice := *s.asking, s.choice(o)
			s.asking = nil
			for i, r := range choice {
				if in.X >= r.X && in.X < r.X+r.W && in.Y >= r.Y && in.Y < r.Y+r.H {
					m.Promote = i == 0
					s.played = &m
				}
			}
			return true
		}
		src, on := s.hit(o, in.X, in.Y)
		if !on {
			s.chosen = source{}
			return true
		}
		if s.chosen.ok && src.drop == 0 && slices.Contains(targets(pos, o, s.chosen), src.sq) {
			s.move(pos, s.chosen, src.sq)
			return true
		}
		s.again = s.chosen == src
		s.chosen = source{}
		if len(targets(pos, o, src)) > 0 {
			s.held, s.dragging, s.chosen = src, false, src
			s.hx, s.hy, s.px, s.py = in.X, in.Y, in.X, in.Y
		}
	case ui.InputPointerMove:
		if !s.held.ok {
			return false
		}
		s.px, s.py = in.X, in.Y
		if math.Hypot(float64(in.X-s.hx), float64(in.Y-s.hy)) > 4 {
			s.dragging = true
		}
	case ui.InputPointerUp:
		if !s.held.ok {
			return false
		}
		if !s.dragging && s.again {
			s.chosen = source{}
		}
		if s.dragging {
			sq, on := s.square(o, in.X, in.Y)
			if on && slices.Contains(targets(pos, o, s.held), sq) {
				s.move(pos, s.held, sq)
			} else if !on || s.held.drop != 0 || sq != s.held.sq {
				s.chosen = source{}
			}
		}
		s.held, s.dragging = source{}, false
	default:
		return false
	}
	return true
}

var (
	wood   = ui.Hex("#e9c983")
	ink    = ui.Hex("#3b2a17")
	face   = ui.LinearGradient{From: ui.Hex("#fbe9c4"), To: ui.Hex("#e8c88f"), Angle: 90}
	edge   = ui.Hex("#9c7a45")
	red    = ui.Hex("#b4231a")
	lit    = ui.RGBA(240, 190, 40, 0.45)
	choose = ui.RGBA(40, 140, 70, 0.35)
	dot    = ui.RGBA(30, 110, 50, 0.65)
)

func (s *State) draw(p *ui.Painter, r ui.Rect, pos *Position, o *Options) {
	cw, ch := s.cw, s.ch
	if cw <= 0 {
		return
	}
	at := func(b ui.Rect) ui.Rect { return ui.Rect{X: r.X + b.X, Y: r.Y + b.Y, W: b.W, H: b.H} }
	board := ui.Rect{X: r.X + s.ox, Y: r.Y + s.oy, W: cw * 9, H: ch * 9}
	frame := ui.Rect{X: board.X - cw*0.25, Y: board.Y - cw*0.25, W: board.W + cw*0.5, H: board.H + cw*0.5}
	p.Shadow(frame, 3, 0, 2, 10, 0, ui.RGB(0, 0, 0).Alpha(0.3))
	p.Fill(frame, wood, 3)
	line := max(1, cw/45)
	for i := range 10 {
		y := board.Y + float32(i)*ch
		p.Line(board.X, y, board.X+board.W, y, line, ink)
		x := board.X + float32(i)*cw
		p.Line(x, board.Y, x, board.Y+board.H, line, ink)
	}
	for _, c := range [][2]float32{{3, 3}, {6, 3}, {3, 6}, {6, 6}} {
		var star ui.Path
		p.FillPath(star.Circle(board.X+c[0]*cw, board.Y+c[1]*ch, cw*0.06), ink)
	}
	num := cw * 0.3
	kanji := []string{"一", "二", "三", "四", "五", "六", "七", "八", "九"}
	for i := range 9 {
		file, rank := strconv.Itoa(9-i), kanji[i]
		if o.Flipped {
			file, rank = strconv.Itoa(i+1), kanji[8-i]
		}
		w, h := p.MeasureText(0, ui.Span{Text: file, Size: num})
		p.Text(board.X+(float32(i)+0.5)*cw-w/2, frame.Y-h-cw*0.06, file, num, ink.Alpha(0.75))
		w, h = p.MeasureText(0, ui.Span{Text: rank, Size: num})
		p.Text(frame.X+frame.W+cw*0.08, board.Y+(float32(i)+0.5)*ch-h/2, rank, num, ink.Alpha(0.75))
	}

	if o.Last.From != o.Last.To || o.Last.Drop != 0 {
		p.Fill(at(s.box(o, o.Last.To)), lit, 0)
		if o.Last.Drop == 0 {
			p.Fill(at(s.box(o, o.Last.From)), lit.Alpha(0.2), 0)
		}
	}
	if o.Check {
		for sq, pc := range pos.Board {
			if pc.Kind == 'K' && pc.Gote == pos.Gote {
				b := at(s.box(o, Square(sq)))
				p.Fill(b, ui.RGBA(230, 40, 30, 0.35), 0)
			}
		}
	}
	if s.chosen.ok && s.chosen.drop == 0 {
		p.Fill(at(s.box(o, s.chosen.sq)), choose, 0)
	}

	held := s.held.ok && s.dragging
	for sq, pc := range pos.Board {
		if pc.Kind == 0 || held && s.held.drop == 0 && s.held.sq == Square(sq) {
			continue
		}
		if s.asking != nil && s.asking.From == Square(sq) {
			continue
		}
		b := at(s.box(o, Square(sq)))
		piece(p, pc, b.X+b.W/2, b.Y+b.H/2, cw, o.Flipped)
	}

	for side := range 2 {
		for _, h := range s.hands[side] {
			b := at(h.r)
			if s.chosen.ok && s.chosen.drop == h.kind && s.chosen.gote == (side == 1) {
				p.Fill(b, choose, 4)
			}
			n := pos.Hands[side][strings.IndexByte(Kinds, h.kind)-1]
			if held && s.held.drop == h.kind && s.held.gote == (side == 1) {
				n--
			}
			if n > 0 {
				piece(p, Piece{Kind: h.kind, Gote: side == 1}, b.X+b.W/2, b.Y+b.H/2, cw*0.92, o.Flipped)
			}
			if n > 1 {
				label := ui.Span{Text: strconv.Itoa(n), Size: cw * 0.26, Weight: 700, Color: ui.RGB(255, 255, 255)}
				w, th := p.MeasureText(0, label)
				d := max(w+cw*0.12, th)
				badge := ui.Rect{X: b.X + b.W - d, Y: b.Y + b.H - d, W: d, H: d}
				p.Fill(badge, ink, d/2)
				p.RichText(badge.X+(d-w)/2, badge.Y+(d-th)/2, 0, label)
			}
		}
	}

	if s.chosen.ok {
		for _, sq := range targets(pos, o, s.chosen) {
			b := at(s.box(o, sq))
			var d ui.Path
			if pos.Board[sq].Kind != 0 {
				p.Stroke(ui.Rect{X: b.X + 2, Y: b.Y + 2, W: b.W - 4, H: b.H - 4}, dot, 2, cw*0.06)
			} else {
				p.FillPath(d.Circle(b.X+b.W/2, b.Y+b.H/2, cw*0.12), dot)
			}
		}
	}

	if s.asking != nil {
		pc := pos.Board[s.asking.From]
		for i, b := range s.choice(o) {
			b = at(b)
			p.Shadow(b, 4, 0, 3, 12, 0, ui.RGB(0, 0, 0).Alpha(0.35))
			p.Fill(b, ui.Hex("#fffaf0"), 4)
			shown := pc
			shown.Promoted = i == 0
			piece(p, shown, b.X+b.W/2, b.Y+b.H/2, cw, o.Flipped)
		}
	}

	if held {
		pc := Piece{Kind: s.held.drop, Gote: pos.Gote}
		if s.held.drop == 0 {
			pc = pos.Board[s.held.sq]
		}
		piece(p, pc, r.X+s.px, r.Y+s.py, cw*1.1, o.Flipped)
	}
}

// size is how big each kind of piece is against the king's, as on a real
// set.
var size = map[byte]float32{'K': 1, 'R': 0.97, 'B': 0.97, 'G': 0.94, 'S': 0.94, 'N': 0.9, 'L': 0.88, 'P': 0.84}

// piece draws pc centered at x, y on squares cw wide, pointing away from
// its side: up for the side at the bottom.
func piece(p *ui.Painter, pc Piece, x, y, cw float32, flipped bool) {
	k := size[pc.Kind]
	w, h := cw*0.84*k, cw*0.98*k
	draw := func() {
		var shape ui.Path
		shape.MoveTo(x, y-h/2).
			LineTo(x+w*0.4, y-h/2+h*0.2).
			LineTo(x+w/2, y+h/2).
			LineTo(x-w/2, y+h/2).
			LineTo(x-w*0.4, y-h/2+h*0.2).Close()
		p.FillPathGradient(&shape, face)
		p.StrokePath(&shape, max(1, cw/40), edge)
		color := ink
		if pc.Promoted {
			color = red
		}
		span := ui.Span{Text: pc.name(), Size: h * 0.48, Weight: 700, Color: color}
		tw, th := p.MeasureText(0, span)
		p.RichText(x-tw/2, y-th/2+h*0.06, tw+1, span)
	}
	if pc.Gote != flipped {
		p.Transform(ui.Rotation(180), x, y, draw)
		return
	}
	draw()
}
