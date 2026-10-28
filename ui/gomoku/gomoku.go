// Package gomoku plays five in a row on a Go board: black and white take
// turns putting a stone on an empty point, and the first to make an
// unbroken line of five or more, across, down or diagonally, wins, struck
// through. Above the board, whose turn it is or who won; ⌘Z and Undo take
// back the last stone. Suggest picks a point for the side to play, so an
// app can play one side itself.
//
//	ev := gomoku.View(c, &app.gomoku, &app.game)
//	if ev.Played && app.computer && app.game.Turn() == goban.White {
//		app.game.Play(app.game.Suggest())
//	}
package gomoku

import (
	"math"

	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/goban"
)

// Game is a game: its board's size and the points played, black first.
type Game struct {
	Size  int
	Moves []goban.Point
}

// New returns a game on a board of size lines, as 15.
func New(size int) Game { return Game{Size: size} }

// Board returns the stones on the board.
func (g Game) Board() goban.Board {
	b := goban.NewBoard(g.Size)
	for i, p := range g.Moves {
		b = b.Set(p, goban.Black+goban.Stone(i%2))
	}
	return b
}

// Turn returns the stone to play next.
func (g Game) Turn() goban.Stone { return goban.Black + goban.Stone(len(g.Moves)%2) }

var directions = [4]goban.Point{{X: 1}, {Y: 1}, {X: 1, Y: 1}, {X: 1, Y: -1}}

// Winner returns the stone that made five or more in a row and the ends
// of its line, or Empty.
func (g Game) Winner() (goban.Stone, [2]goban.Point) {
	b := g.Board()
	for _, p := range g.Moves {
		s := b.At(p)
		for _, d := range directions {
			lo, hi := p, p
			for b.At(goban.Point{X: lo.X - d.X, Y: lo.Y - d.Y}) == s {
				lo = goban.Point{X: lo.X - d.X, Y: lo.Y - d.Y}
			}
			for b.At(goban.Point{X: hi.X + d.X, Y: hi.Y + d.Y}) == s {
				hi = goban.Point{X: hi.X + d.X, Y: hi.Y + d.Y}
			}
			if max(abs(hi.X-lo.X), abs(hi.Y-lo.Y)) >= 4 {
				return s, [2]goban.Point{lo, hi}
			}
		}
	}
	return goban.Empty, [2]goban.Point{}
}

// Play puts the stone to play on p, if p is empty and no one has won.
func (g *Game) Play(p goban.Point) bool {
	if w, _ := g.Winner(); w != goban.Empty || g.Board().At(p) != goban.Empty {
		return false
	}
	g.Moves = append(g.Moves, p)
	return true
}

// Undo takes back the last stone.
func (g *Game) Undo() {
	if len(g.Moves) > 0 {
		g.Moves = g.Moves[:len(g.Moves)-1]
	}
}

// Suggest returns a point for the stone to play: the one that makes its
// own best line or blocks the other's, counting how long and how open
// each line through it would be.
func (g Game) Suggest() goban.Point {
	b, me := g.Board(), g.Turn()
	if len(g.Moves) == 0 {
		return goban.Point{X: g.Size / 2, Y: g.Size / 2}
	}
	best, bestScore := goban.Point{X: -1}, -1.0
	for y := range g.Size {
		for x := range g.Size {
			p := goban.Point{X: x, Y: y}
			if b.At(p) != goban.Empty || !near(b, p) {
				continue
			}
			// Making a line counts a little more than blocking one as
			// long, so that a win is taken before a loss is stopped.
			score := 1.1*value(b, p, me) + value(b, p, me.Other())
			if score > bestScore {
				best, bestScore = p, score
			}
		}
	}
	return best
}

// near reports whether a stone is within two points of p.
func near(b goban.Board, p goban.Point) bool {
	for dy := -2; dy <= 2; dy++ {
		for dx := -2; dx <= 2; dx++ {
			if b.At(goban.Point{X: p.X + dx, Y: p.Y + dy}) != goban.Empty {
				return true
			}
		}
	}
	return false
}

// value scores s playing p: for each direction, the stones of s it would
// join and how many of the line's two ends stay open.
func value(b goban.Board, p goban.Point, s goban.Stone) float64 {
	total := 0.0
	for _, d := range directions {
		run, open := 1, 0
		for _, sign := range [2]int{-1, 1} {
			q := goban.Point{X: p.X + sign*d.X, Y: p.Y + sign*d.Y}
			for q.X >= 0 && q.Y >= 0 && q.X < b.Size && q.Y < b.Size && b.At(q) == s {
				run++
				q = goban.Point{X: q.X + sign*d.X, Y: q.Y + sign*d.Y}
			}
			if q.X >= 0 && q.Y >= 0 && q.X < b.Size && q.Y < b.Size && b.At(q) == goban.Empty {
				open++
			}
		}
		switch {
		case run >= 5:
			total += 1e6
		case open > 0:
			total += math.Pow(10, float64(run-1)) * float64(open*open)
		}
	}
	return total
}

func abs(x int) int { return max(x, -x) }

// State is what a game's view keeps: its board's. Keep it in the app's
// state, one for each game.
type State struct {
	board goban.State
}

// Events is a game's view, and whether a stone was played or taken back
// in the frame.
type Events struct {
	ui.Element
	Played, Undone bool
}

// View creates the view of *g, which a click on the board plays.
func View(c *ui.Context, s *State, g *Game) Events {
	t := c.Theme()
	var ev Events
	winner, line := g.Winner()
	root := ui.Column(c).Gap(t.Space(3))
	root.Children(func() {
		ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
			status, who := "", g.Turn()
			switch {
			case winner != goban.Empty:
				status, who = name(winner)+" wins", winner
			case len(g.Moves) == g.Size*g.Size:
				status, who = "Draw", goban.Empty
			default:
				status = name(who) + " to play"
			}
			if who != goban.Empty {
				ui.Box(c).Size(14, 14).Radius(7).Background(swatch(who)).Border(1, t.Border)
			}
			ui.Text(c, status).Bold()
			ui.Spacer(c)
			if (ui.Button(c, "Undo").Disabled(len(g.Moves) == 0).Clicked() || c.Shortcut(ui.Cmd, ui.KeyZ)) && len(g.Moves) > 0 {
				g.Undo()
				ev.Undone = true
			}
		})
		var last *goban.Point
		if len(g.Moves) > 0 {
			last = &g.Moves[len(g.Moves)-1]
		}
		turn := g.Turn()
		if winner != goban.Empty {
			turn = goban.Empty
		}
		board := goban.View(c, &s.board, g.Board(), goban.Options{Turn: turn, Last: last})
		board.Grow(1).Basis(0).FillWidth()
		if board.Played && g.Play(board.Point) {
			ev.Played = true
		}
		if winner != goban.Empty {
			board.Children(func() {
				ui.Box(c).Fill().PassThrough().Draw(func(p *ui.Painter, r ui.Rect) {
					// Where goban puts its lines in a box of this size.
					gap := float32(math.Floor(float64(min(r.W, r.H) / float32(g.Size+2))))
					ox := r.X + (r.W-gap*float32(g.Size-1))/2 + 0.5
					oy := r.Y + (r.H-gap*float32(g.Size-1))/2 + 0.5
					var strike ui.Path
					strike.MoveTo(ox+float32(line[0].X)*gap, oy+float32(line[0].Y)*gap).
						LineTo(ox+float32(line[1].X)*gap, oy+float32(line[1].Y)*gap).Cap(ui.RoundCap)
					p.StrokePath(&strike, max(3, gap*0.14), ui.Hex("#E5484D"))
				})
			})
		}
	})
	if ev.Played || ev.Undone {
		root.SetChanged()
	}
	ev.Element = root
	return ev
}

func name(s goban.Stone) string {
	if s == goban.White {
		return "White"
	}
	return "Black"
}

func swatch(s goban.Stone) ui.Color {
	if s == goban.White {
		return ui.RGB(245, 245, 240)
	}
	return ui.RGB(25, 25, 25)
}
