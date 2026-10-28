// Package reversi plays reversi on an eight-by-eight board: discs black
// on one side and white on the other, black moving first, each move
// flanking a line of the other's discs, which turn over. The squares the
// player to move can take show a dot; pointing at one shows the disc there
// faintly and rings the discs it would turn. A click plays it, and the
// discs turn one after another outward from it. A player with no move
// passes, and the game ends when neither has one. Beside the board, each
// player's count of discs, whose turn it is, and the result.
//
//	ev := reversi.View(c, &app.board, &app.game)
//	if ev.Played && app.game.Turn == reversi.White {
//		app.game.Play(app.computer())
//	}
package reversi

import (
	"fmt"
	"math"
	"time"

	"github.com/egoist/mygo/ui"
)

// Disc is a square's disc, or none.
type Disc int8

const (
	Empty Disc = iota
	Black
	White
)

func (d Disc) other() Disc { return 3 - d }

func (d Disc) String() string { return [...]string{"Empty", "Black", "White"}[d] }

// Square is a square of the board, by row and column from the top left.
type Square struct{ Row, Col int }

// Game is a game: the board, whose turn it is, Empty once it is over, and
// whether the last turn was passed.
type Game struct {
	Board  [8][8]Disc
	Turn   Disc
	Passed bool
	// last is the square played last and turned the discs turned by it.
	last   Square
	turned []Square
	moves  int
}

// New returns a game at its start: two discs of each color crossed in the
// middle, black to move.
func New() Game {
	var g Game
	g.Board[3][3], g.Board[4][4] = White, White
	g.Board[3][4], g.Board[4][3] = Black, Black
	g.Turn = Black
	return g
}

var dirs = [8][2]int{{-1, -1}, {-1, 0}, {-1, 1}, {0, -1}, {0, 1}, {1, -1}, {1, 0}, {1, 1}}

// Flips returns the discs that playing sq for who would turn.
func (g *Game) Flips(sq Square, who Disc) []Square {
	if sq.Row < 0 || sq.Row > 7 || sq.Col < 0 || sq.Col > 7 || g.Board[sq.Row][sq.Col] != Empty {
		return nil
	}
	var out []Square
	for _, d := range dirs {
		var line []Square
		r, c := sq.Row+d[0], sq.Col+d[1]
		for r >= 0 && r < 8 && c >= 0 && c < 8 && g.Board[r][c] == who.other() {
			line = append(line, Square{r, c})
			r, c = r+d[0], c+d[1]
		}
		if len(line) > 0 && r >= 0 && r < 8 && c >= 0 && c < 8 && g.Board[r][c] == who {
			out = append(out, line...)
		}
	}
	return out
}

// Moves returns the squares who can play.
func (g *Game) Moves(who Disc) []Square {
	var out []Square
	for r := range 8 {
		for c := range 8 {
			if len(g.Flips(Square{r, c}, who)) > 0 {
				out = append(out, Square{r, c})
			}
		}
	}
	return out
}

// Play plays sq for the player to move and passes the turn on, past a
// player with no move, and reports whether sq was a move.
func (g *Game) Play(sq Square) bool {
	flips := g.Flips(sq, g.Turn)
	if g.Turn == Empty || len(flips) == 0 {
		return false
	}
	g.Board[sq.Row][sq.Col] = g.Turn
	for _, f := range flips {
		g.Board[f.Row][f.Col] = g.Turn
	}
	g.last, g.turned = sq, flips
	g.moves++
	g.Passed = false
	switch {
	case len(g.Moves(g.Turn.other())) > 0:
		g.Turn = g.Turn.other()
	case len(g.Moves(g.Turn)) > 0:
		g.Passed = true
	default:
		g.Turn = Empty
	}
	return true
}

// Count returns how many discs each player has.
func (g *Game) Count() (black, white int) {
	for _, row := range g.Board {
		for _, d := range row {
			switch d {
			case Black:
				black++
			case White:
				white++
			}
		}
	}
	return
}

// State is what a board keeps: when the last move's discs began to turn.
// Keep it in the app's state, one for each board.
type State struct {
	moves  int
	turned time.Time
}

// Events is a board, and whether a move was played on it in the frame.
type Events struct {
	ui.Element
	Played bool
}

const (
	turn    = 260 * time.Millisecond
	stagger = 70 * time.Millisecond
)

var (
	felt = ui.Hex("#2E7D4F")
	ink  = ui.Hex("#151515")
	pale = ui.Hex("#F4F4F0")
)

// View creates the board of *g and its score.
func View(c *ui.Context, s *State, g *Game) Events {
	t := c.Theme()
	var ev Events
	now := c.Now()
	if s.moves != g.moves {
		s.moves, s.turned = g.moves, now
		if c.Preferences().ReduceMotion {
			s.turned = now.Add(-time.Hour)
		}
	}
	cell := t.Space(13)
	root := ui.Row(c).Gap(t.Space(8)).AlignItems(ui.Center)
	root.Children(func() {
		board(c, s, g, &ev, cell)
		score(c, g)
	})
	if ev.Played {
		root.SetChanged()
	}
	ev.Element = root
	return ev
}

// board creates the board of g, cell DIPs a square.
func board(c *ui.Context, s *State, g *Game, ev *Events, cell float32) {
	t := c.Theme()
	board := ui.Box(c).Size(cell*8+8, cell*8+8).Radius(t.Space(2)).Background(felt).Cursor(ui.CursorPointer).
		Role(ui.RoleImage).Label("Board")
	px, py, over := board.PointerPosition()
	hover := Square{-1, -1}
	if over {
		hover = Square{int((py - 4) / cell), int((px - 4) / cell)}
	}
	moves := g.Moves(g.Turn)
	var flips []Square
	if g.Turn != Empty {
		flips = g.Flips(hover, g.Turn)
	}
	if board.Clicked() && g.Play(hover) {
		ev.Played = true
		s.moves, s.turned = g.moves, c.Now()
	}
	game := *g
	since := s.turned
	board.Draw(func(p *ui.Painter, r ui.Rect) {
		x0, y0 := r.X+4, r.Y+4
		for i := 1; i < 8; i++ {
			p.Fill(ui.Rect{X: x0 + float32(i)*cell - 0.5, Y: y0, W: 1, H: cell * 8}, ui.RGBA(0, 0, 0, 0.35), 0)
			p.Fill(ui.Rect{X: x0, Y: y0 + float32(i)*cell - 0.5, W: cell * 8, H: 1}, ui.RGBA(0, 0, 0, 0.35), 0)
		}
		for _, k := range [][2]int{{2, 2}, {2, 6}, {6, 2}, {6, 6}} {
			var dot ui.Path
			dot.Circle(x0+float32(k[1])*cell, y0+float32(k[0])*cell, 3)
			p.FillPath(&dot, ui.RGBA(0, 0, 0, 0.45))
		}
		center := func(sq Square) (float32, float32) {
			return x0 + (float32(sq.Col)+0.5)*cell, y0 + (float32(sq.Row)+0.5)*cell
		}
		disc := func(sq Square, d Disc, squash float32, alpha float32) {
			x, y := center(sq)
			rad := cell * 0.4
			p.Shadow(ui.Rect{X: x - rad, Y: y - rad, W: 2 * rad, H: 2 * rad}, rad, 0, 2, 4, 0, ui.RGBA(0, 0, 0, 0.35*alpha))
			col := ink
			if d == White {
				col = pale
			}
			p.FillPath(ellipse(x, y, rad*max(0.04, squash), rad), col.Alpha(alpha))
			p.FillPath(ellipse(x-rad*0.3*squash, y-rad*0.3, rad*0.35*squash, rad*0.35), ui.RGBA(255, 255, 255, 0.08*alpha))
		}
		animating := false
		for row := range 8 {
			for col := range 8 {
				d := game.Board[row][col]
				if d == Empty {
					continue
				}
				sq := Square{row, col}
				squash := float32(1)
				for _, f := range game.turned {
					if f != sq {
						continue
					}
					steps := max(abs(f.Row-game.last.Row), abs(f.Col-game.last.Col))
					k := float64(p.Now().Sub(since)-time.Duration(steps-1)*stagger) / float64(turn)
					k = max(0, min(1, k))
					if k < 1 {
						animating = true
					}
					// The disc shows its old side until it stands on edge.
					if k < 0.5 {
						d = d.other()
					}
					squash = float32(math.Abs(math.Cos(math.Pi * k)))
				}
				disc(sq, d, squash, 1)
			}
		}
		if animating {
			p.AnimationFrame()
		}
		for _, m := range moves {
			if m == hover {
				disc(m, game.Turn, 1, 0.45)
				continue
			}
			x, y := center(m)
			var dot ui.Path
			dot.Circle(x, y, 4)
			p.FillPath(&dot, ui.RGBA(0, 0, 0, 0.28))
		}
		for _, f := range flips {
			x, y := center(f)
			var ring ui.Path
			ring.Circle(x, y, cell*0.4+2)
			p.StrokePath(&ring, 2, ui.Hex("#FFD43B"))
		}
		if game.moves > 0 {
			x, y := center(game.last)
			var mark ui.Path
			mark.Circle(x, y, 3)
			p.FillPath(&mark, ui.Hex("#E03131"))
		}
	})
}

// score shows each player's discs and whose turn it is.
func score(c *ui.Context, g *Game) {
	t := c.Theme()
	black, white := g.Count()
	ui.Column(c).Gap(t.Space(4)).Width(t.Space(44)).Children(func() {
		score := func(d Disc, n int) {
			row := ui.Row(c.Key(d)).Gap(t.Space(3)).AlignItems(ui.Center).Padding(t.Space(2), t.Space(3)).Radius(t.Space(2))
			if g.Turn == d {
				row.Background(t.Accent.Alpha(0.12))
			}
			row.Children(func() {
				col := ink
				if d == White {
					col = pale
				}
				ui.Box(c).Size(t.Space(6), t.Space(6)).Radius(t.Space(3)).Background(col).Border(1, t.Border)
				ui.Text(c, d.String()).Grow(1)
				ui.Text(c, fmt.Sprint(n)).FontSize(t.FontSize + 8).Bold().FontFeatures("tnum")
			})
		}
		score(Black, black)
		score(White, white)
		var note string
		switch {
		case g.Turn == Empty && black > white:
			note = "Black wins"
		case g.Turn == Empty && white > black:
			note = "White wins"
		case g.Turn == Empty:
			note = "A draw"
		case g.Passed:
			note = fmt.Sprintf("%s had no move and passed. %s to move.", g.Turn.other(), g.Turn)
		default:
			note = fmt.Sprintf("%s to move", g.Turn)
		}
		ui.Text(c, note).TextColor(t.TextMuted)
	})
}

// ellipse returns an ellipse about x, y.
func ellipse(x, y, rx, ry float32) *ui.Path {
	const k = 0.5523
	var e ui.Path
	e.MoveTo(x+rx, y)
	e.CubeTo(x+rx, y+ry*k, x+rx*k, y+ry, x, y+ry)
	e.CubeTo(x-rx*k, y+ry, x-rx, y+ry*k, x-rx, y)
	e.CubeTo(x-rx, y-ry*k, x-rx*k, y-ry, x, y-ry)
	e.CubeTo(x+rx*k, y-ry, x+rx, y-ry*k, x+rx, y)
	e.Close()
	return &e
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
