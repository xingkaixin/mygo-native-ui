// Package draughts plays English draughts, or checkers, on the dark
// squares of an eight-by-eight board: twelve men a side, black at the
// bottom moving first. Men step diagonally forward and kings, crowned on
// reaching the far row, either way; a piece takes by jumping an
// opponent's next to it, and goes on jumping while it can. Taking is a
// must. A click on a piece that can move lifts it and marks where it can
// land, the squares it jumps through fainter; a click on one plays the
// move, the piece sliding along it and the taken pieces fading. A side
// with no move left loses. Beside the board, each side's pieces, whose
// turn it is, and the result.
//
//	ev := draughts.View(c, &app.board, &app.game)
//	if ev.Played && app.game.Turn == draughts.White {
//		app.game.Play(app.computer())
//	}
package draughts

import (
	"fmt"
	"math"
	"slices"
	"time"

	"github.com/egoist/mygo/ui"
)

// Side is a player, or none once the game is over.
type Side int8

const (
	Nobody Side = iota
	Black
	White
)

func (s Side) String() string { return [...]string{"Nobody", "Black", "White"}[s] }

func (s Side) other() Side { return 3 - s }

// Piece is a square's piece, or none.
type Piece struct {
	Side Side
	King bool
}

// Square is a square of the board, by row and column from the top left.
type Square struct{ Row, Col int }

func (q Square) on() bool { return q.Row >= 0 && q.Row < 8 && q.Col >= 0 && q.Col < 8 }

// Move is a move: the square the piece leaves and the squares it lands
// on, one for a step and one for each jump.
type Move struct {
	From Square
	Path []Square
}

// To returns where the move ends.
func (m Move) To() Square { return m.Path[len(m.Path)-1] }

// Jumps reports whether the move takes.
func (m Move) Jumps() bool {
	return abs(m.Path[0].Row-m.From.Row) == 2
}

// Game is a game: the board, whose turn it is, Nobody once it is over.
type Game struct {
	Board [8][8]Piece
	Turn  Side
	// last, piece and taken are the last move, the piece that made it
	// before any crowning, and the pieces it took, for showing it.
	last  Move
	piece Piece
	taken []Piece
	moves int
}

// New returns a game about to start, Black to move.
func New() Game {
	g := Game{Turn: Black}
	for r := range 8 {
		for c := range 8 {
			if (r+c)%2 == 1 {
				switch {
				case r < 3:
					g.Board[r][c] = Piece{Side: White}
				case r > 4:
					g.Board[r][c] = Piece{Side: Black}
				}
			}
		}
	}
	return g
}

func (g *Game) at(q Square) Piece { return g.Board[q.Row][q.Col] }

// dirs returns the ways a piece moves: forward for a man, up for black
// and down for white, and every way for a king.
func dirs(p Piece) [][2]int {
	if p.King {
		return [][2]int{{-1, -1}, {-1, 1}, {1, -1}, {1, 1}}
	}
	if p.Side == Black {
		return [][2]int{{-1, -1}, {-1, 1}}
	}
	return [][2]int{{1, -1}, {1, 1}}
}

// Moves returns the moves the side to move can make: when any piece can
// take, only moves that take, each jumping on as long as it can.
func (g *Game) Moves() []Move {
	var steps, jumps []Move
	for r := range 8 {
		for c := range 8 {
			from := Square{r, c}
			p := g.at(from)
			if p.Side != g.Turn || g.Turn == Nobody {
				continue
			}
			jumps = append(jumps, g.jumps(from, p, from, nil, map[Square]bool{})...)
			for _, d := range dirs(p) {
				to := Square{r + d[0], c + d[1]}
				if to.on() && g.at(to).Side == Nobody {
					steps = append(steps, Move{From: from, Path: []Square{to}})
				}
			}
		}
	}
	if len(jumps) > 0 {
		return jumps
	}
	return steps
}

// jumps returns the full jumping moves of p from at, having come along
// path from start and taken the pieces in taken.
func (g *Game) jumps(start Square, p Piece, at Square, path []Square, taken map[Square]bool) []Move {
	var out []Move
	for _, d := range dirs(p) {
		over, to := Square{at.Row + d[0], at.Col + d[1]}, Square{at.Row + 2*d[0], at.Col + 2*d[1]}
		if !to.on() || taken[over] || g.at(over).Side != p.Side.other() || (g.at(to).Side != Nobody && to != start) {
			continue
		}
		next := append(append([]Square{}, path...), to)
		taken[over] = true
		// A man crowned by a jump ends its move there.
		crowned := !p.King && (to.Row == 0 && p.Side == Black || to.Row == 7 && p.Side == White)
		var more []Move
		if !crowned {
			more = g.jumps(start, p, to, next, taken)
		}
		delete(taken, over)
		if len(more) == 0 {
			out = append(out, Move{From: start, Path: next})
		}
		out = append(out, more...)
	}
	return out
}

// Play makes move m for the side to move and reports whether it could.
func (g *Game) Play(m Move) bool {
	legal := false
	for _, l := range g.Moves() {
		if l.From == m.From && slices.Equal(l.Path, m.Path) {
			legal = true
		}
	}
	if !legal {
		return false
	}
	p := g.at(m.From)
	g.last, g.piece, g.taken = Move{From: m.From, Path: append([]Square{}, m.Path...)}, p, nil
	g.Board[m.From.Row][m.From.Col] = Piece{}
	at := m.From
	for _, to := range m.Path {
		if abs(to.Row-at.Row) == 2 {
			over := Square{(at.Row + to.Row) / 2, (at.Col + to.Col) / 2}
			g.taken = append(g.taken, g.at(over))
			g.Board[over.Row][over.Col] = Piece{}
		}
		at = to
	}
	if at.Row == 0 && p.Side == Black || at.Row == 7 && p.Side == White {
		p.King = true
	}
	g.Board[at.Row][at.Col] = p
	g.moves++
	g.Turn = g.Turn.other()
	if len(g.Moves()) == 0 {
		g.Turn = Nobody
	}
	return true
}

// Winner returns who won once the game is over: the side that made the
// last move, the other having none.
func (g *Game) Winner() Side {
	if g.Turn != Nobody || g.moves == 0 {
		return Nobody
	}
	return g.piece.Side
}

// Count returns how many pieces each side has.
func (g *Game) Count() (black, white int) {
	for _, row := range g.Board {
		for _, p := range row {
			switch p.Side {
			case Black:
				black++
			case White:
				white++
			}
		}
	}
	return black, white
}

// State is what a board keeps: the lifted piece and when the last move
// began to show. Keep it in the app's state, one for each board.
type State struct {
	lifted   Square
	isLifted bool
	moves    int
	shown    time.Time
}

// Events is a board, and whether a move was played on it in the frame.
type Events struct {
	ui.Element
	Played bool
}

const hop = 220 * time.Millisecond

var (
	light = ui.RGB(240, 222, 186)
	dark  = ui.RGB(120, 84, 56)
	ink   = ui.RGB(36, 32, 32)
	pale  = ui.RGB(236, 230, 214)
)

// View creates the board of *g and its score.
func View(c *ui.Context, s *State, g *Game) Events {
	t := c.Theme()
	var ev Events
	now := c.Now()
	if s.moves != g.moves {
		s.moves, s.shown, s.isLifted = g.moves, now, false
		if c.Preferences().ReduceMotion {
			s.shown = now.Add(-time.Hour)
		}
	}
	// progress runs from 0 to the number of hops of the last move.
	progress := float32(0)
	animating := false
	if g.moves > 0 {
		progress = float32(now.Sub(s.shown)) / float32(hop)
		if progress < float32(len(g.last.Path)) {
			animating = true
			c.After(time.Second / 60)
		}
	}
	cell := t.Space(13)
	moves := g.Moves()
	var lifted []Move
	if s.isLifted {
		for _, m := range moves {
			if m.From == s.lifted {
				lifted = append(lifted, m)
			}
		}
	}
	root := ui.Row(c).Gap(t.Space(8)).AlignItems(ui.Center)
	root.Children(func() {
		board := ui.Box(c).Size(cell*8+8, cell*8+8).Radius(t.Space(2)).Background(dark.Mix(ink, 0.4)).Role(ui.RoleImage).Label("Board")
		px, py, over := board.PointerPosition()
		hover := Square{-1, -1}
		if over {
			hover = Square{int((py - 4) / cell), int((px - 4) / cell)}
		}
		canLift := func(q Square) bool {
			for _, m := range moves {
				if m.From == q {
					return true
				}
			}
			return false
		}
		landing := func(q Square) (Move, bool) {
			for _, m := range lifted {
				if m.To() == q {
					return m, true
				}
			}
			return Move{}, false
		}
		if !animating {
			if _, ok := landing(hover); ok || canLift(hover) {
				board.Cursor(ui.CursorPointer)
			}
			if board.Clicked() {
				if m, ok := landing(hover); ok {
					if g.Play(m) {
						ev.Played = true
					}
				} else if canLift(hover) {
					s.lifted, s.isLifted = hover, !(s.isLifted && s.lifted == hover)
				} else {
					s.isLifted = false
				}
			}
		}
		board.Draw(func(p *ui.Painter, r ui.Rect) {
			x0, y0 := r.X+4, r.Y+4
			center := func(q Square) (float32, float32) {
				return x0 + (float32(q.Col)+0.5)*cell, y0 + (float32(q.Row)+0.5)*cell
			}
			for row := range 8 {
				for col := range 8 {
					sq := light
					if (row+col)%2 == 1 {
						sq = dark
					}
					p.Fill(ui.Rect{X: x0 + float32(col)*cell, Y: y0 + float32(row)*cell, W: cell, H: cell}, sq, 0)
				}
			}
			if s.isLifted {
				x, y := center(s.lifted)
				p.Fill(ui.Rect{X: x - cell/2, Y: y - cell/2, W: cell, H: cell}, t.Accent.Alpha(0.35), 0)
				for _, m := range lifted {
					for k, q := range m.Path {
						qx, qy := center(q)
						a := float32(0.9)
						if k < len(m.Path)-1 {
							a = 0.35
						}
						var dot ui.Path
						p.FillPath(dot.Circle(qx, qy, cell*0.14), ui.RGBA(255, 255, 255, a))
					}
				}
			}
			if !animating {
				if g.moves > 0 {
					for _, q := range append([]Square{g.last.From}, g.last.Path...) {
						x, y := center(q)
						p.Stroke(ui.Rect{X: x - cell/2 + 1.5, Y: y - cell/2 + 1.5, W: cell - 3, H: cell - 3}, ui.RGBA(255, 230, 120, 0.55), 0, 2)
					}
				}
				for row := range 8 {
					for col := range 8 {
						if pc := g.Board[row][col]; pc.Side != Nobody {
							x, y := center(Square{row, col})
							disc(p, x, y, cell, pc, 1)
						}
					}
				}
				return
			}
			// While the last move shows, the board is as it was before it:
			// the moving piece slides hop by hop and each taken piece fades
			// once the piece is past it.
			moving := g.last.Path[len(g.last.Path)-1]
			hops := append([]Square{g.last.From}, g.last.Path...)
			k := min(int(progress), len(hops)-2)
			f := progress - float32(k)
			ax, ay := center(hops[k])
			bx, by := center(hops[k+1])
			for row := range 8 {
				for col := range 8 {
					q := Square{row, col}
					if pc := g.Board[row][col]; pc.Side != Nobody && q != moving {
						x, y := center(q)
						disc(p, x, y, cell, pc, 1)
					}
				}
			}
			if g.last.Jumps() {
				at := g.last.From
				for j, to := range g.last.Path {
					over := Square{(at.Row + to.Row) / 2, (at.Col + to.Col) / 2}
					fade := min(1, max(0, float32(j+1)-progress))
					if fade > 0 {
						x, y := center(over)
						disc(p, x, y, cell, g.taken[j], fade)
					}
					at = to
				}
			}
			// The piece lifts off each square and drops on the next.
			lift := float32(math.Sin(float64(f)*math.Pi)) * cell * 0.12
			disc(p, ax+(bx-ax)*f, ay+(by-ay)*f-lift, cell, g.piece, 1)
		})
		score(c, g)
	})
	if ev.Played {
		root.SetChanged()
	}
	ev.Element = root
	return ev
}

// disc paints a piece at x, y, a king as two pieces stacked.
func disc(p *ui.Painter, x, y, cell float32, pc Piece, alpha float32) {
	face, rim := ink, ui.RGB(80, 74, 70)
	if pc.Side == White {
		face, rim = pale, ui.RGB(190, 180, 160)
	}
	r := cell * 0.38
	layers := 1
	if pc.King {
		layers = 2
	}
	for k := range layers {
		dy := -float32(k) * cell * 0.08
		var shadow, body, groove ui.Path
		p.FillPath(shadow.Circle(x, y+dy+cell*0.04, r), ui.RGBA(0, 0, 0, 0.35*alpha))
		p.FillPath(body.Circle(x, y+dy, r), face.Alpha(alpha))
		p.StrokePath(groove.Circle(x, y+dy, r*0.68), 1.5, rim.Alpha(alpha))
	}
	if pc.King {
		dy := -cell * 0.08
		var crown ui.Path
		w := r * 0.5
		crown.MoveTo(x-w, y+dy+w*0.45).LineTo(x-w, y+dy-w*0.3).LineTo(x-w*0.5, y+dy+w*0.05).LineTo(x, y+dy-w*0.55).
			LineTo(x+w*0.5, y+dy+w*0.05).LineTo(x+w, y+dy-w*0.3).LineTo(x+w, y+dy+w*0.45).Close()
		p.FillPath(&crown, ui.RGB(232, 184, 60).Alpha(alpha))
	}
}

func score(c *ui.Context, g *Game) {
	t := c.Theme()
	black, white := g.Count()
	ui.Column(c).Gap(t.Space(4)).Width(t.Space(40)).Children(func() {
		for _, side := range []Side{White, Black} {
			n := black
			if side == White {
				n = white
			}
			row := ui.Row(c.Key(side)).Gap(t.Space(3)).AlignItems(ui.Center).Padding(t.Space(2)).Radius(t.Radius)
			if g.Turn == side {
				row.Background(t.Accent.Alpha(0.12))
			}
			row.Children(func() {
				ui.Text(c, side.String()).FontWeight(600).Grow(1)
				ui.Text(c, fmt.Sprint(n)).FontSize(t.FontSize + 6).FontWeight(700).FontFeatures("tnum")
			})
		}
		msg := g.Turn.String() + " to move"
		if g.Turn == Nobody {
			msg = g.Winner().String() + " wins"
		} else if len(g.Moves()) > 0 && g.Moves()[0].Jumps() {
			msg += ", and must take"
		}
		ui.Text(c, msg).TextColor(t.TextMuted)
	})
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
