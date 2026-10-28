// Package dominoes plays the draw game of dominoes for two with a
// double-six set: seven tiles each, then in turn a tile matching an end
// of the line is played there, doubles across it. A player with nothing
// to play draws from the boneyard until something fits, and passes once
// it is empty. The first to play out their hand wins the pips left in the
// other's; if neither can play, the lighter hand wins the difference.
//
// The view shows the other side's hand face down, the line, and your
// hand, the tiles that fit lifted. Clicking one plays it; one fitting
// both ends lights both, to click where it goes. Draw and Pass show when
// they are all that is left to do. The app plays the other side, as
// Auto does.
//
//	if app.game.Turn == 1 {
//		app.game.Auto()
//	}
//	dominoes.View(c, &app.table, &app.game)
package dominoes

import (
	"fmt"
	"math/rand/v2"

	"github.com/egoist/mygo/ui"
)

// Tile is a domino: the pips on its two halves, 0 to 6.
type Tile struct{ A, B int }

// Double reports whether both halves are the same.
func (t Tile) Double() bool { return t.A == t.B }

// Pips returns the pips on both halves.
func (t Tile) Pips() int { return t.A + t.B }

func (t Tile) flip() Tile { return Tile{t.B, t.A} }

// Set returns the 28 tiles of a double-six set.
func Set() []Tile {
	var out []Tile
	for a := range 7 {
		for b := a; b < 7; b++ {
			out = append(out, Tile{a, b})
		}
	}
	return out
}

// Game is a game: the line, left to right with each tile's B meeting
// the next's A; the hands, yours first; the tiles left to draw; whose
// turn it is; and how many turns in a row were passed.
type Game struct {
	Line     []Tile
	Hands    [2][]Tile
	Boneyard []Tile
	Turn     int
	Passes   int
}

// New deals a game shuffled from seed. The hand with the highest double,
// or else the heaviest tile, plays first.
func New(seed uint64) Game {
	set := Set()
	r := rand.New(rand.NewPCG(seed, seed^0xa0761d6478bd642f))
	r.Shuffle(len(set), func(i, j int) { set[i], set[j] = set[j], set[i] })
	g := Game{Boneyard: set[14:]}
	g.Hands[0], g.Hands[1] = append([]Tile(nil), set[:7]...), append([]Tile(nil), set[7:14]...)
	best := -1
	for h, hand := range g.Hands {
		for _, t := range hand {
			score := t.Pips()
			if t.Double() {
				score += 100
			}
			if score > best {
				best, g.Turn = score, h
			}
		}
	}
	return g
}

// Fits reports whether t can be played at the left end of the line and
// at its right. Either fits an empty line.
func (g Game) Fits(t Tile) (left, right bool) {
	if len(g.Line) == 0 {
		return true, true
	}
	l, r := g.Line[0].A, g.Line[len(g.Line)-1].B
	return t.A == l || t.B == l, t.A == r || t.B == r
}

func (g Game) canPlay() bool {
	for _, t := range g.Hands[g.Turn] {
		if l, r := g.Fits(t); l || r {
			return true
		}
	}
	return false
}

// Play plays tile i of the hand whose turn it is at the right end of the
// line or the left.
func (g *Game) Play(i int, right bool) bool {
	hand := g.Hands[g.Turn]
	if _, over := g.Over(); over || i < 0 || i >= len(hand) {
		return false
	}
	t := hand[i]
	l, r := g.Fits(t)
	switch {
	case len(g.Line) == 0:
		g.Line = []Tile{t}
	case right && r:
		if t.A != g.Line[len(g.Line)-1].B {
			t = t.flip()
		}
		g.Line = append(g.Line, t)
	case !right && l:
		if t.B != g.Line[0].A {
			t = t.flip()
		}
		g.Line = append([]Tile{t}, g.Line...)
	default:
		return false
	}
	g.Hands[g.Turn] = append(hand[:i:i], hand[i+1:]...)
	g.Turn, g.Passes = 1-g.Turn, 0
	return true
}

// Draw takes a tile from the boneyard into the hand whose turn it is, if
// nothing in it fits.
func (g *Game) Draw() bool {
	if g.canPlay() || len(g.Boneyard) == 0 {
		return false
	}
	g.Hands[g.Turn] = append(g.Hands[g.Turn], g.Boneyard[0])
	g.Boneyard = g.Boneyard[1:]
	return true
}

// Pass passes the turn, if nothing fits and the boneyard is empty.
func (g *Game) Pass() bool {
	if _, over := g.Over(); over || g.canPlay() || len(g.Boneyard) > 0 {
		return false
	}
	g.Turn, g.Passes = 1-g.Turn, g.Passes+1
	return true
}

// Over reports whether the game is over and who won, or -1 for a tie.
func (g Game) Over() (winner int, over bool) {
	for h, hand := range g.Hands {
		if len(hand) == 0 {
			return h, true
		}
	}
	if g.Passes < 2 {
		return 0, false
	}
	a, b := pips(g.Hands[0]), pips(g.Hands[1])
	switch {
	case a < b:
		return 0, true
	case b < a:
		return 1, true
	}
	return -1, true
}

// Score returns what the winner of a game over scores.
func (g Game) Score() int {
	w, over := g.Over()
	if !over || w < 0 {
		return 0
	}
	if len(g.Hands[w]) == 0 {
		return pips(g.Hands[1-w])
	}
	return pips(g.Hands[1-w]) - pips(g.Hands[w])
}

func pips(hand []Tile) int {
	n := 0
	for _, t := range hand {
		n += t.Pips()
	}
	return n
}

// Move returns a move for the hand whose turn it is: its heaviest tile
// that fits, doubles first among equals, and the end it goes at.
func (g Game) Move() (i int, right, ok bool) {
	best := -1
	for k, t := range g.Hands[g.Turn] {
		l, r := g.Fits(t)
		if !l && !r {
			continue
		}
		score := 2 * t.Pips()
		if t.Double() {
			score++
		}
		if score > best {
			best, i, right, ok = score, k, r, true
		}
	}
	return i, right, ok
}

// Auto takes the turn of the hand whose turn it is: it plays Move,
// drawing first until something fits, or passes.
func (g *Game) Auto() {
	for g.Draw() {
	}
	if i, right, ok := g.Move(); ok {
		g.Play(i, right)
	} else {
		g.Pass()
	}
}

// State is what a table keeps: the tile picked that fits both ends,
// plus one, or 0. Keep it in the app's state, one for each table.
type State struct {
	picked int
}

// Events is a table, and whether you played, drew or passed in the
// frame.
type Events struct {
	ui.Element
	Moved bool
}

// View creates the table of *g, where you are hand 0.
func View(c *ui.Context, s *State, g *Game) Events {
	t := c.Theme()
	var ev Events
	winner, over := g.Over()
	mine := g.Turn == 0 && !over
	if !mine || s.picked > len(g.Hands[0]) {
		s.picked = 0
	}
	unit := t.Space(9)
	root := ui.Column(c).Gap(t.Space(5))
	root.Children(func() {
		ui.Row(c).Gap(t.Space(1)).Children(func() {
			for i, tile := range g.Hands[1] {
				ui.Box(c.Key(i)).Size(unit*0.7, unit*1.4).Draw(func(p *ui.Painter, r ui.Rect) {
					face(p, r, tile, true, !over)
				})
			}
		})
		ui.ScrollHorizontal(c).FillWidth().Radius(t.Radius + 4).Background(t.Surface).Children(func() {
			ui.Row(c).Gap(2).AlignItems(ui.Center).Padding(t.Space(3)).Height(unit*2 + t.Space(6)).Children(func() {
				end := func(right bool) {
					if s.picked == 0 {
						return
					}
					b := ui.ButtonBase(c.Key(right)).Size(unit*2, unit).Radius(5).Border(2, t.Accent).BorderStyle(ui.BorderDashed).Label(map[bool]string{false: "Play at the left end", true: "Play at the right end"}[right])
					b.Background(t.Accent.Alpha(0.12))
					if b.Clicked() && g.Play(s.picked-1, right) {
						s.picked, ev.Moved = 0, true
					}
				}
				end(false)
				for i, tile := range g.Line {
					w, h := unit*2, unit
					if tile.Double() {
						w, h = h, w
					}
					ui.Box(c.Key([2]any{"line", i})).Size(w, h).Draw(func(p *ui.Painter, r ui.Rect) {
						face(p, r, tile, tile.Double(), false)
					})
				}
				end(true)
				if len(g.Line) == 0 {
					ui.Text(c, "Play any tile to start the line").TextColor(t.TextMuted)
				}
			})
		})
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, status(g, winner, over)).Bold()
			ui.Text(c, fmt.Sprintf("%d in the boneyard", len(g.Boneyard))).TextColor(t.TextMuted)
			ui.Spacer(c)
			if mine && !g.canPlay() {
				if len(g.Boneyard) > 0 {
					if ui.Button(c, "Draw").Clicked() && g.Draw() {
						ev.Moved = true
					}
				} else if ui.Button(c, "Pass").Clicked() && g.Pass() {
					ev.Moved = true
				}
			}
		})
		ui.Row(c).Gap(t.Space(1.5)).AlignItems(ui.End).Wrap().Children(func() {
			for i, tile := range g.Hands[0] {
				l, r := g.Fits(tile)
				fits := mine && (l || r)
				b := ui.ButtonBase(c.Key(tile)).Size(unit, unit*2).Label(fmt.Sprintf("%d–%d", tile.A, tile.B)).Disabled(!fits)
				lift := float32(0)
				if fits {
					lift = t.Space(2)
				}
				b.Margin(0, 0, b.AnimateTo("dominoes.lift", lift, ui.Snappy), 0)
				if mine && !fits {
					b.Opacity(0.5)
				}
				picked := s.picked == i+1
				b.Draw(func(p *ui.Painter, r ui.Rect) {
					face(p, r, tile, true, false)
					if picked {
						p.Stroke(r, t.Accent, 5, 2.5)
					}
				})
				if b.Clicked() && fits {
					switch {
					case l && r && len(g.Line) > 0 && g.Line[0].A != g.Line[len(g.Line)-1].B:
						s.picked = i + 1
					case g.Play(i, r):
						s.picked, ev.Moved = 0, true
					}
				}
			}
		})
	})
	if ev.Moved {
		root.SetChanged()
	}
	ev.Element = root
	return ev
}

func status(g *Game, winner int, over bool) string {
	switch {
	case !over && g.Turn == 0:
		return "Your turn"
	case !over:
		return "Their turn"
	case winner == 0:
		return fmt.Sprintf("You win %d", g.Score())
	case winner == 1:
		return fmt.Sprintf("They win %d", g.Score())
	}
	return "Blocked: a tie"
}

var (
	ivory    = ui.Hex("#F7F2E6")
	edge     = ui.Hex("#CFC6B2")
	ink      = ui.Hex("#22201C")
	backFill = ui.Hex("#2F5D50")
)

// face draws tile in r, its halves top and bottom if vertical, else left
// and right; or its back.
func face(p *ui.Painter, r ui.Rect, tile Tile, vertical, back bool) {
	radius := min(r.W, r.H) * 0.12
	p.Shadow(r, radius, 0, 1, 3, 0, ui.RGBA(0, 0, 0, 0.25))
	if back {
		p.Fill(r, backFill, radius)
		p.Stroke(ui.Rect{X: r.X + 3, Y: r.Y + 3, W: r.W - 6, H: r.H - 6}, ivory.Alpha(0.25), radius*0.6, 1)
		return
	}
	p.Fill(r, ivory, radius)
	p.Stroke(r, edge, radius, 1)
	halves := [2]ui.Rect{{X: r.X, Y: r.Y, W: r.W / 2, H: r.H}, {X: r.X + r.W/2, Y: r.Y, W: r.W / 2, H: r.H}}
	if vertical {
		halves = [2]ui.Rect{{X: r.X, Y: r.Y, W: r.W, H: r.H / 2}, {X: r.X, Y: r.Y + r.H/2, W: r.W, H: r.H / 2}}
		p.Fill(ui.Rect{X: r.X + r.W*0.15, Y: r.Y + r.H/2 - 0.5, W: r.W * 0.7, H: 1}, edge, 0)
	} else {
		p.Fill(ui.Rect{X: r.X + r.W/2 - 0.5, Y: r.Y + r.H*0.15, W: 1, H: r.H * 0.7}, edge, 0)
	}
	for k, n := range [2]int{tile.A, tile.B} {
		h := halves[k]
		side := min(h.W, h.H)
		dot := side * 0.09
		for _, at := range pipsAt[n] {
			x, y := at[0], at[1]
			if !vertical && n == 6 {
				x, y = y, x
			}
			var pip ui.Path
			p.FillPath(pip.Circle(h.X+h.W/2+float32(x)*side*0.27, h.Y+h.H/2+float32(y)*side*0.27, dot), ink)
		}
	}
}

// pipsAt places each number's pips on a 3 by 3 grid around the middle
// of a half.
var pipsAt = [7][][2]int{
	{},
	{{0, 0}},
	{{-1, -1}, {1, 1}},
	{{-1, -1}, {0, 0}, {1, 1}},
	{{-1, -1}, {1, -1}, {-1, 1}, {1, 1}},
	{{-1, -1}, {1, -1}, {0, 0}, {-1, 1}, {1, 1}},
	{{-1, -1}, {1, -1}, {-1, 0}, {1, 0}, {-1, 1}, {1, 1}},
}
