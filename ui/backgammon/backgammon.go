// Package backgammon plays backgammon between two players at one board:
// the twenty-four points as triangles in four quarters, the bar between
// them, and a tray beside them for the checkers borne off. White moves
// from its 24 point down to its home board, points 1 to 6, at the bottom
// right, and black the other way round to its home at the top right.
//
// Roll, or Space, throws the dice. A click on a point with the player's
// checkers, or on the bar, chooses its top checker and lights where it
// can go with each die left; a click there moves it, hitting a lone
// checker of the other's to the bar, and a double-click moves a checker by
// the first die it can use. A checker on the bar must come in first, and
// checkers bear off once all of a player's are home. The dice used fade,
// and Undo takes back the turn's moves. Done, or Space again, passes the
// turn once no move is left.
//
//	backgammon.View(c, &app.board, &app.game)
package backgammon

import (
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"

	"github.com/egoist/mygo/ui"
)

// Side is a player.
type Side int8

const (
	White Side = 1
	Black Side = -1
)

func (s Side) String() string {
	if s == Black {
		return "Black"
	}
	return "White"
}

func (s Side) index() int {
	if s == Black {
		return 1
	}
	return 0
}

// The ends of a move other than the points.
const (
	Bar = 0  // where a move from the bar starts
	Off = -1 // where a move bearing off ends
)

// Game is a game: the checkers on each point from 1 to 24, white's as
// positive counts and black's as negative, on the bar and borne off, by
// side, whose turn it is, and the dice left to move by.
type Game struct {
	Points   [25]int
	OnBar    [2]int
	BorneOff [2]int
	Turn     Side
	Dice     []int
	// rolled are the dice as thrown, for showing which are used.
	rolled []int
}

// New returns a game set up to start, white to roll.
func New() Game {
	var g Game
	g.Points[24], g.Points[13], g.Points[8], g.Points[6] = 2, 5, 3, 5
	g.Points[1], g.Points[12], g.Points[17], g.Points[19] = -2, -5, -3, -5
	g.Turn = White
	return g
}

// Roll throws the dice for the player to move.
func (g *Game) Roll() { g.Throw(1+rand.IntN(6), 1+rand.IntN(6)) }

// Throw sets the dice to a and b, for dice thrown elsewhere: two to move
// by, or four of a double.
func (g *Game) Throw(a, b int) {
	g.Dice = []int{a, b}
	if a == b {
		g.Dice = []int{a, a, a, a}
	}
	g.rolled = slices.Clone(g.Dice)
}

func (g *Game) own(p int) int { return g.Points[p] * int(g.Turn) }

// home reports whether all the player's checkers are in their home board
// or borne off.
func (g *Game) home() bool {
	if g.OnBar[g.Turn.index()] > 0 {
		return false
	}
	for p := 1; p <= 24; p++ {
		if g.own(p) > 0 && !inHome(g.Turn, p) {
			return false
		}
	}
	return true
}

func inHome(s Side, p int) bool {
	if s == White {
		return p <= 6
	}
	return p >= 19
}

// Target returns where a checker of the player to move goes from from by
// die, and whether it can.
func (g *Game) Target(from, die int) (int, bool) {
	s := g.Turn
	if g.OnBar[s.index()] > 0 && from != Bar {
		return 0, false
	}
	var to int
	switch {
	case from == Bar && g.OnBar[s.index()] > 0:
		to = 25 - die
		if s == Black {
			to = die
		}
	case from >= 1 && from <= 24 && g.own(from) > 0:
		to = from - die*int(s)
	default:
		return 0, false
	}
	if to >= 1 && to <= 24 {
		return to, g.own(to) >= -1
	}
	if !g.home() {
		return 0, false
	}
	exact := to == 0 || to == 25
	if exact {
		return Off, true
	}
	// A die higher than needed bears off only the checker farthest out.
	for p := 1; p <= 24; p++ {
		if g.own(p) > 0 && inHome(s, p) && (s == White && p > from || s == Black && p < from) {
			return 0, false
		}
	}
	return Off, true
}

// Moves returns where a checker can go from from, by each die left.
func (g *Game) Moves(from int) map[int]int {
	out := map[int]int{}
	for _, d := range g.Dice {
		if to, ok := g.Target(from, d); ok {
			if _, seen := out[to]; !seen {
				out[to] = d
			}
		}
	}
	return out
}

// CanMove reports whether any checker of the player to move can move.
func (g *Game) CanMove() bool {
	for from := 0; from <= 24; from++ {
		if len(g.Moves(from)) > 0 {
			return true
		}
	}
	return false
}

// Move moves a checker from from by die, and reports whether it could.
func (g *Game) Move(from, die int) bool {
	to, ok := g.Target(from, die)
	if !ok || !slices.Contains(g.Dice, die) {
		return false
	}
	s := g.Turn
	if from == Bar {
		g.OnBar[s.index()]--
	} else {
		g.Points[from] -= int(s)
	}
	if to == Off {
		g.BorneOff[s.index()]++
	} else {
		if g.own(to) == -1 {
			g.Points[to] = 0
			g.OnBar[(-s).index()]++
		}
		g.Points[to] += int(s)
	}
	i := slices.Index(g.Dice, die)
	g.Dice = slices.Delete(g.Dice, i, i+1)
	return true
}

// Winner returns the side that has borne off all fifteen checkers, or 0.
func (g *Game) Winner() Side {
	switch {
	case g.BorneOff[0] == 15:
		return White
	case g.BorneOff[1] == 15:
		return Black
	}
	return 0
}

// State is what a board keeps: the checker chosen and the game as the
// turn began, for Undo. Keep it in the app's state, one for each board.
type State struct {
	from   int
	chosen bool
	start  *Game
}

var (
	wood   = ui.Hex("#6B4226")
	cloth  = ui.Hex("#E9DCC2")
	dark   = ui.Hex("#8C3B2E")
	light  = ui.Hex("#D9A066")
	ivory  = ui.Hex("#F7F3EA")
	ebony  = ui.Hex("#2A2420")
	target = ui.Hex("#2F9E44")
)

// View creates the board of *g, with its dice and buttons.
func View(c *ui.Context, s *State, g *Game) ui.Element {
	t := c.Theme()
	if g.Turn == 0 {
		*g = New()
	}
	if s.start == nil || len(g.rolled) == 0 {
		snap := *g
		snap.Dice, snap.rolled = nil, nil
		s.start = &snap
	}
	pw := t.Space(11)
	ph := pw * 5
	barW := pw * 0.9
	trayW := pw * 1.1
	m := t.Space(3)
	bw := m*2 + pw*12 + barW + trayW + m
	bh := m*2 + ph*2 + pw*1.6
	// place returns the left of point p and whether it is in the top row.
	place := func(p int) (float32, bool) {
		k := 12 - p
		top := p >= 13
		if top {
			k = p - 13
		}
		x := m + float32(k)*pw
		if k >= 6 {
			x += barW
		}
		return x, top
	}
	rolled := len(g.rolled) > 0
	winner := g.Winner()
	done := rolled && !g.CanMove()
	next := func() {
		switch {
		case winner != 0:
			*g = New()
		case !rolled:
			g.Roll()
		case done:
			g.Turn, g.Dice, g.rolled = -g.Turn, nil, nil
		}
		s.chosen, s.start = false, nil
	}
	root := ui.Column(c).Gap(t.Space(4)).AlignItems(ui.Center).Focusable().FocusRing(false).Label("Backgammon")
	if root.Shortcut(0, ui.KeySpace) {
		next()
	}
	var dests map[int]int
	if s.chosen {
		dests = g.Moves(s.from)
	}
	root.Children(func() {
		board := ui.Box(c).Size(bw, bh).Radius(t.Space(2)).Background(wood).Role(ui.RoleImage).
			Label("Board").Value(g.Turn.String() + " to move")
		px, py, over := board.PointerPosition()
		hit := -2
		if over {
			switch {
			case px >= m+pw*6 && px < m+pw*6+barW:
				hit = Bar
			case px >= bw-m-trayW:
				hit = Off
			default:
				for p := 1; p <= 24; p++ {
					x, top := place(p)
					if px >= x && px < x+pw && top == (py < bh/2) {
						hit = p
					}
				}
			}
		}
		if s.chosen && dests[hit] > 0 || hit >= Bar && len(g.Moves(hit)) > 0 {
			board.Cursor(ui.CursorPointer)
		}
		if board.DoubleClicked() && hit >= Bar {
			for _, d := range g.Dice {
				if g.Move(hit, d) {
					break
				}
			}
			s.chosen = false
		} else if board.Clicked() {
			switch {
			case s.chosen && dests[hit] > 0:
				g.Move(s.from, dests[hit])
				s.chosen = false
			case hit >= Bar && len(g.Moves(hit)) > 0:
				s.from, s.chosen = hit, !(s.chosen && s.from == hit)
			default:
				s.chosen = false
			}
			board.SetChanged()
		}
		if s.chosen {
			dests = g.Moves(s.from)
		} else {
			dests = nil
		}
		game, chosen, from := *g, s.chosen, s.from
		board.Draw(func(p *ui.Painter, r ui.Rect) {
			p.Fill(ui.Rect{X: r.X + m, Y: r.Y + m, W: pw * 6, H: r.H - 2*m}, cloth, 2)
			p.Fill(ui.Rect{X: r.X + m + pw*6 + barW, Y: r.Y + m, W: pw * 6, H: r.H - 2*m}, cloth, 2)
			tx := r.X + r.W - m - trayW
			p.Fill(ui.Rect{X: tx, Y: r.Y + m, W: trayW, H: r.H - 2*m}, wood.Mix(ui.RGB(0, 0, 0), 0.3), 2)
			rad := pw * 0.46
			for pt := 1; pt <= 24; pt++ {
				x, top := place(pt)
				x += r.X
				y0, tip := r.Y+m, r.Y+m+ph
				if !top {
					y0, tip = r.Y+r.H-m, r.Y+r.H-m-ph
				}
				var tri ui.Path
				tri.MoveTo(x, y0)
				tri.LineTo(x+pw/2, tip)
				tri.LineTo(x+pw, y0)
				tri.Close()
				col := dark
				if pt%2 == 0 {
					col = light
				}
				p.FillPath(&tri, col)
				if _, ok := dests[pt]; ok {
					p.FillPath(&tri, target.Alpha(0.6))
				}
				num := ui.Span{Text: fmt.Sprint(pt), Size: 9, Color: ui.RGBA(255, 255, 255, 0.6)}
				nw, nh := p.MeasureText(0, num)
				ny := r.Y + (m-nh)/2
				if !top {
					ny = r.Y + r.H - (m+nh)/2
				}
				p.RichText(x+(pw-nw)/2, ny, 0, num)
				n := game.Points[pt]
				side := White
				if n < 0 {
					side, n = Black, -n
				}
				shown := min(n, 5)
				for k := range shown {
					cy := y0 + rad + float32(k)*rad*2
					if !top {
						cy = y0 - rad - float32(k)*rad*2
					}
					lit := chosen && from == pt && k == shown-1
					checker(p, x+pw/2, cy, rad, side, lit, n > 5 && k == 4, n)
				}
			}
			bx := r.X + m + pw*6 + barW/2
			for i, side := range []Side{White, Black} {
				n := game.OnBar[i]
				shown := min(n, 4)
				for k := range shown {
					dir := float32(1)
					if side == Black {
						dir = -1
					}
					cy := r.Y + r.H/2 + dir*(rad*1.6+float32(k)*rad*2)
					lit := chosen && from == Bar && side == game.Turn && k == shown-1
					checker(p, bx, cy, rad, side, lit, n > 4 && k == 3, n)
				}
			}
			if _, ok := dests[Off]; ok {
				p.Fill(ui.Rect{X: tx, Y: r.Y + m, W: trayW, H: r.H - 2*m}, target.Alpha(0.5), 2)
			}
			for i, side := range []Side{White, Black} {
				col := ivory
				if side == Black {
					col = ebony
				}
				for k := range game.BorneOff[i] {
					y := r.Y + r.H - m - 4 - float32(k+1)*6
					if side == Black {
						y = r.Y + m + 4 + float32(k)*6
					}
					p.Fill(ui.Rect{X: tx + 4, Y: y, W: trayW - 8, H: 5}, col, 1.5)
				}
			}
			// The dice lie in the half of the board away from the player's home.
			dx := r.X + m + pw*3
			if game.Turn == Black {
				dx = r.X + m + pw*9 + barW
			}
			left := slices.Clone(game.Dice)
			for i, d := range game.rolled {
				used := true
				if k := slices.Index(left, d); k >= 0 {
					left, used = slices.Delete(left, k, k+1), false
				}
				off := (float32(i) - float32(len(game.rolled)-1)/2) * pw * 1.1
				die(p, dx+off, r.Y+r.H/2, pw*0.8, d, used, game.Turn)
			}
		})
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			note, label := g.Turn.String()+" to roll", "Roll"
			switch {
			case winner != 0:
				note, label = winner.String()+" wins", "New game"
			case done && len(g.Dice) == len(g.rolled):
				note, label = g.Turn.String()+" has no move", "Done"
			case done:
				note, label = g.Turn.String()+" has moved", "Done"
			case rolled:
				dice := make([]string, len(g.Dice))
				for i, d := range g.Dice {
					dice[i] = fmt.Sprint(d)
				}
				note, label = g.Turn.String()+" to move "+strings.Join(dice, ", "), "Done"
			}
			ui.Text(c, note).Width(t.Space(48))
			if ui.Button(c, "Undo").Disabled(!rolled || len(g.Dice) == len(g.rolled)).Clicked() {
				dice := g.rolled
				*g = *s.start
				g.rolled, g.Dice = dice, slices.Clone(dice)
				s.chosen = false
			}
			if ui.PrimaryButton(c, label).Disabled(rolled && !done && winner == 0).Clicked() {
				next()
			}
		})
	})
	return root
}

// checker draws a checker at x, y, ringed when chosen, and on the top of
// a stack taller than shows, how many it holds.
func checker(p *ui.Painter, x, y, rad float32, side Side, lit, count bool, n int) {
	face, edge, ink := ivory, ui.RGBA(0, 0, 0, 0.3), ebony
	if side == Black {
		face, edge, ink = ebony, ui.RGBA(255, 255, 255, 0.2), ivory
	}
	var d ui.Path
	d.Circle(x, y, rad-0.5)
	p.FillPath(&d, face)
	p.StrokePath(&d, 1, edge)
	var groove ui.Path
	groove.Circle(x, y, rad*0.62)
	p.StrokePath(&groove, 1, edge)
	if lit {
		var ring ui.Path
		ring.Circle(x, y, rad+1.5)
		p.StrokePath(&ring, 2.5, target)
	}
	if count {
		span := ui.Span{Text: fmt.Sprint(n), Size: rad * 0.9, Weight: 700, Color: ink}
		w, h := p.MeasureText(0, span)
		p.RichText(x-w/2, y-h/2, 0, span)
	}
}

// die draws a die showing n, faded once used.
func die(p *ui.Painter, x, y, size float32, n int, used bool, side Side) {
	r := ui.Rect{X: x - size/2, Y: y - size/2, W: size, H: size}
	face, pip := ivory, ebony
	if side == Black {
		face, pip = ebony, ivory
	}
	alpha := float32(1)
	if used {
		alpha = 0.35
	}
	p.Shadow(r, size*0.18, 0, 2, 4, 0, ui.RGBA(0, 0, 0, 0.3*alpha))
	p.Fill(r, face.Alpha(alpha), size*0.18)
	spots := map[int][][2]float32{
		1: {{0.5, 0.5}}, 2: {{0.27, 0.27}, {0.73, 0.73}}, 3: {{0.27, 0.27}, {0.5, 0.5}, {0.73, 0.73}},
		4: {{0.27, 0.27}, {0.73, 0.27}, {0.27, 0.73}, {0.73, 0.73}},
		5: {{0.27, 0.27}, {0.73, 0.27}, {0.5, 0.5}, {0.27, 0.73}, {0.73, 0.73}},
		6: {{0.27, 0.25}, {0.73, 0.25}, {0.27, 0.5}, {0.73, 0.5}, {0.27, 0.75}, {0.73, 0.75}},
	}
	for _, s := range spots[n] {
		var d ui.Path
		d.Circle(r.X+r.W*s[0], r.Y+r.H*s[1], size*0.08)
		p.FillPath(&d, pip.Alpha(alpha))
	}
}
