// Package morris plays nine men's morris: white and black each put their
// nine pieces on the 24 points of three nested squares, then move them
// along the lines to a point next to them, or anywhere once down to
// three. Three in a row along a line is a mill, and makes its player take
// one of the other's pieces, not one in a mill unless all are. A player
// left with two pieces, or with no move, loses.
//
// The view shows the board with its mills lit, the pieces still to place
// beside it, and whose turn it is and what to do. A click puts a piece on
// an empty point; to move, click a piece and then one of the points
// marked for it; after a mill, the pieces that can be taken are ringed.
// The app can play a side with Auto.
//
//	if app.game.Turn == morris.Black {
//		app.game.Auto()
//	}
//	morris.View(c, &app.board, &app.game)
package morris

import (
	"fmt"
	"slices"

	"github.com/egoist/mygo/ui"
)

// Piece is what is on a point.
type Piece uint8

const (
	Empty Piece = iota
	White
	Black
)

// Other returns the other side.
func (p Piece) Other() Piece { return 3 - p }

func (p Piece) String() string { return [...]string{"", "White", "Black"}[p] }

// Points is how many points the board has: eight on each of three
// squares, from the outer, clockwise from the top left corner.
const Points = 24

var (
	neighbors [Points][]int
	mills     [16][3]int
)

func init() {
	n := 0
	for r := range 3 {
		for k := range 8 {
			i := r*8 + k
			neighbors[i] = append(neighbors[i], r*8+(k+7)%8, r*8+(k+1)%8)
			if k%2 == 1 {
				if r > 0 {
					neighbors[i] = append(neighbors[i], i-8)
				}
				if r < 2 {
					neighbors[i] = append(neighbors[i], i+8)
				}
			}
			if k%2 == 0 {
				mills[n] = [3]int{i, r*8 + k + 1, r*8 + (k+2)%8}
				n++
			}
		}
	}
	for _, k := range []int{1, 3, 5, 7} {
		mills[n] = [3]int{k, 8 + k, 16 + k}
		n++
	}
}

// at returns where point i is on a board 6 units across.
func at(i int) (x, y float32) {
	r, k := i/8, i%8
	o, f := float32(r), float32(6-r)
	return [8]float32{o, 3, f, f, f, 3, o, o}[k], [8]float32{o, o, o, 3, f, f, f, 3}[k]
}

// Game is a game: what is on each point, whose turn it is, how many
// pieces each side has yet to place, and whether the side to play must
// take a piece for a mill it just made.
type Game struct {
	Board    [Points]Piece
	Turn     Piece
	InHand   [3]int
	Removing bool
}

// New returns a game with nine pieces each to place, white first.
func New() Game { return Game{Turn: White, InHand: [3]int{0, 9, 9}} }

func (g Game) count(p Piece) int {
	n := g.InHand[p]
	for _, q := range g.Board {
		if q == p {
			n++
		}
	}
	return n
}

// inMill reports whether the piece on i is part of a mill.
func (g Game) inMill(i int) bool {
	p := g.Board[i]
	for _, m := range mills {
		if slices.Contains(m[:], i) && g.Board[m[0]] == p && g.Board[m[1]] == p && g.Board[m[2]] == p {
			return true
		}
	}
	return false
}

// Targets returns the points the piece on from can move to: those next
// to it, or every empty one when its side is down to three.
func (g Game) Targets(from int) []int {
	if g.Board[from] != g.Turn || g.InHand[g.Turn] > 0 || g.Removing {
		return nil
	}
	var out []int
	if g.count(g.Turn) == 3 {
		for i, q := range g.Board {
			if q == Empty {
				out = append(out, i)
			}
		}
		return out
	}
	for _, n := range neighbors[from] {
		if g.Board[n] == Empty {
			out = append(out, n)
		}
	}
	return out
}

// Removable returns the pieces the side to play may take after a mill.
func (g Game) Removable() []int {
	if !g.Removing {
		return nil
	}
	var free, all []int
	for i, q := range g.Board {
		if q == g.Turn.Other() {
			all = append(all, i)
			if !g.inMill(i) {
				free = append(free, i)
			}
		}
	}
	if len(free) > 0 {
		return free
	}
	return all
}

// Winner returns the side that has won, or Empty.
func (g Game) Winner() Piece {
	if g.Removing {
		return Empty
	}
	if g.count(g.Turn) < 3 {
		return g.Turn.Other()
	}
	if g.InHand[g.Turn] > 0 {
		return Empty
	}
	for i := range g.Board {
		if len(g.Targets(i)) > 0 {
			return Empty
		}
	}
	return g.Turn.Other()
}

func (g *Game) after(to int) {
	if g.inMill(to) {
		g.Removing = true
		return
	}
	g.Turn = g.Turn.Other()
}

// Place puts a piece of the side to play on the empty point to.
func (g *Game) Place(to int) bool {
	if g.Winner() != Empty || g.Removing || g.InHand[g.Turn] == 0 || g.Board[to] != Empty {
		return false
	}
	g.Board[to] = g.Turn
	g.InHand[g.Turn]--
	g.after(to)
	return true
}

// Move moves the piece on from to to, if it may go there.
func (g *Game) Move(from, to int) bool {
	if g.Winner() != Empty || !slices.Contains(g.Targets(from), to) {
		return false
	}
	g.Board[to], g.Board[from] = g.Board[from], Empty
	g.after(to)
	return true
}

// Remove takes the piece on i for a mill.
func (g *Game) Remove(i int) bool {
	if !slices.Contains(g.Removable(), i) {
		return false
	}
	g.Board[i] = Empty
	g.Removing = false
	g.Turn = g.Turn.Other()
	return true
}

// closes reports whether p on to makes a mill, with from left empty.
func (g Game) closes(p Piece, from, to int) bool {
	for _, m := range mills {
		if !slices.Contains(m[:], to) {
			continue
		}
		full := true
		for _, i := range m {
			full = full && (i == to || i != from && g.Board[i] == p)
		}
		if full {
			return true
		}
	}
	return false
}

// Auto plays a turn for the side to play: it makes a mill if it can, or
// else blocks one of the other side's, and takes the piece that most
// nearly makes a mill.
func (g *Game) Auto() {
	if g.Winner() != Empty {
		return
	}
	me, them := g.Turn, g.Turn.Other()
	if g.Removing {
		best, score := -1, -1
		for _, i := range g.Removable() {
			s := len(neighbors[i])
			for _, m := range mills {
				if slices.Contains(m[:], i) {
					for _, j := range m {
						if g.Board[j] == them {
							s += 2
						}
					}
				}
			}
			if s > score {
				best, score = i, s
			}
		}
		g.Remove(best)
		return
	}
	type move struct{ from, to int }
	var moves []move
	if g.InHand[me] > 0 {
		for i, q := range g.Board {
			if q == Empty {
				moves = append(moves, move{-1, i})
			}
		}
	} else {
		for i := range g.Board {
			for _, t := range g.Targets(i) {
				moves = append(moves, move{i, t})
			}
		}
	}
	best, score := moves[0], -1
	for _, m := range moves {
		s := len(neighbors[m.to])
		if g.closes(me, m.from, m.to) {
			s += 100
		}
		if g.closes(them, -1, m.to) {
			s += 50
		}
		if m.from >= 0 && g.closes(them, -1, m.from) {
			s -= 30
		}
		if s > score {
			best, score = m, s
		}
	}
	if best.from < 0 {
		g.Place(best.to)
	} else {
		g.Move(best.from, best.to)
	}
}

// State is what a board keeps: the piece picked to move, plus one, or 0.
// Keep it in the app's state, one for each board.
type State struct {
	picked int
}

// Events is a board, and whether a piece was placed, moved or taken in
// the frame.
type Events struct {
	ui.Element
	Moved bool
}

// View creates the board of *g.
func View(c *ui.Context, s *State, g *Game) Events {
	t := c.Theme()
	var ev Events
	winner := g.Winner()
	if s.picked > 0 && (g.Board[s.picked-1] != g.Turn || g.Removing || winner != Empty) {
		s.picked = 0
	}
	var targets []int
	if s.picked > 0 {
		targets = g.Targets(s.picked - 1)
	}
	removable := g.Removable()
	root := ui.Row(c).Gap(t.Space(6)).AlignItems(ui.Stretch)
	root.Children(func() {
		board := ui.Box(c).Grow(1).Basis(0).Label("Morris board")
		bounds := board.LayoutBounds()
		side := min(bounds.W, bounds.H)
		unit := side / 7
		ox, oy := (bounds.W-side)/2+unit/2, (bounds.H-side)/2+unit/2
		px, py, over := board.PointerPosition()
		hover := -1
		for i := range Points {
			x, y := at(i)
			dx, dy := px-(ox+x*unit), py-(oy+y*unit)
			if over && dx*dx+dy*dy < unit*unit*0.16 {
				hover = i
			}
		}
		actionable := func(i int) bool {
			switch {
			case i < 0 || winner != Empty:
				return false
			case g.Removing:
				return slices.Contains(removable, i)
			case g.InHand[g.Turn] > 0:
				return g.Board[i] == Empty
			}
			return g.Board[i] == g.Turn && len(g.Targets(i)) > 0 || slices.Contains(targets, i)
		}
		if actionable(hover) {
			board.Cursor(ui.CursorPointer)
		}
		if board.Clicked() && actionable(hover) {
			switch {
			case g.Removing:
				ev.Moved = g.Remove(hover)
			case g.InHand[g.Turn] > 0:
				ev.Moved = g.Place(hover)
			case slices.Contains(targets, hover):
				ev.Moved = g.Move(s.picked-1, hover)
				s.picked = 0
			default:
				s.picked = hover + 1
			}
		}
		game, picked := *g, s.picked-1
		board.Draw(func(p *ui.Painter, r ui.Rect) {
			pt := func(i int) (float32, float32) {
				x, y := at(i)
				return r.X + ox + x*unit, r.Y + oy + y*unit
			}
			wood := ui.Rect{X: r.X + ox - unit*0.5, Y: r.Y + oy - unit*0.5, W: unit * 7, H: unit * 7}
			p.Shadow(wood, 6, 0, 2, 10, 0, ui.RGBA(0, 0, 0, 0.3))
			p.FillGradient(wood, ui.LinearGradient{From: ui.Hex("#E3C893"), To: ui.Hex("#D2AE72"), Angle: 160}, 6)
			ink := ui.RGBA(58, 42, 22, 0.8)
			w := max(1.5, unit*0.04)
			var lines ui.Path
			for _, m := range mills {
				lines.MoveTo(pt(m[0]))
				lines.LineTo(pt(m[2]))
			}
			p.StrokePath(&lines, w, ink)
			for _, m := range mills {
				q := game.Board[m[0]]
				if q == Empty || game.Board[m[1]] != q || game.Board[m[2]] != q {
					continue
				}
				x0, y0 := pt(m[0])
				x2, y2 := pt(m[2])
				var mill ui.Path
				mill.MoveTo(x0, y0).LineTo(x2, y2).Cap(ui.RoundCap)
				p.StrokePath(&mill, w*3, t.Accent.Alpha(0.7))
			}
			for i, q := range game.Board {
				x, y := pt(i)
				var dot ui.Path
				if q == Empty {
					dot.Circle(x, y, max(3, unit*0.07))
					p.FillPath(&dot, ink)
					continue
				}
				piece(p, x, y, unit*0.34, q, t)
				if i == picked {
					var ring ui.Path
					p.StrokePath(ring.Circle(x, y, unit*0.42), 2.5, t.Accent)
				}
				if slices.Contains(removable, i) {
					var ring ui.Path
					p.StrokePath(ring.Circle(x, y, unit*0.42), 2.5, t.Danger)
				}
			}
			for _, i := range targets {
				x, y := pt(i)
				var dot ui.Path
				p.FillPath(dot.Circle(x, y, unit*0.14), t.Accent.Alpha(0.5))
			}
			if hover >= 0 && game.Board[hover] == Empty && game.InHand[game.Turn] > 0 && !game.Removing && winner == Empty {
				x, y := pt(hover)
				var ghost ui.Path
				p.FillPath(ghost.Circle(x, y, unit*0.34), t.Accent.Alpha(0.25))
			}
		})
		ui.Column(c).Width(t.Space(44)).Gap(t.Space(5)).Children(func() {
			ui.Text(c, status(g, winner)).Bold().FontSize(t.FontSize + 1)
			for _, side := range []Piece{White, Black} {
				ui.Column(c.Key(side)).Gap(t.Space(1.5)).Children(func() {
					ui.Text(c, fmt.Sprintf("%s · %d on the board", side, g.count(side)-g.InHand[side])).TextColor(t.TextMuted).FontSize(t.FontSize - 1)
					ui.Row(c).Gap(2).Height(t.Space(5)).Children(func() {
						for range g.InHand[side] {
							ui.Box(c).Size(t.Space(5), t.Space(5)).Draw(func(p *ui.Painter, r ui.Rect) {
								piece(p, r.X+r.W/2, r.Y+r.H/2, r.W*0.42, side, t)
							})
						}
					})
				})
			}
		})
	})
	if ev.Moved {
		root.SetChanged()
	}
	ev.Element = root
	return ev
}

func status(g *Game, winner Piece) string {
	switch {
	case winner != Empty:
		return winner.String() + " wins"
	case g.Removing:
		return fmt.Sprintf("%s: take a %s piece", g.Turn, g.Turn.Other())
	case g.InHand[g.Turn] > 0:
		return g.Turn.String() + " to place"
	case g.count(g.Turn) == 3:
		return g.Turn.String() + " flies"
	}
	return g.Turn.String() + " to move"
}

// piece draws a piece of side q at x, y, rad across.
func piece(p *ui.Painter, x, y, rad float32, q Piece, t *ui.Theme) {
	fill, rim := ui.Hex("#F4F1EA"), ui.Hex("#B9B2A3")
	if q == Black {
		fill, rim = ui.Hex("#2A2A2E"), ui.Hex("#55555C")
	}
	box := ui.Rect{X: x - rad, Y: y - rad, W: 2 * rad, H: 2 * rad}
	p.Shadow(box, rad, 0, 1.5, 4, 0, ui.RGBA(0, 0, 0, 0.3))
	var disc ui.Path
	disc.Circle(x, y, rad)
	p.FillPath(&disc, fill)
	p.StrokePath(&disc, 1.5, rim)
	var inner ui.Path
	p.StrokePath(inner.Circle(x, y, rad*0.6), 1, rim.Alpha(0.6))
}
