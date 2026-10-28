// Package mancala plays mancala by the rules of Kalah: two rows of six
// pits with four seeds each and a store at either end, South owning the
// bottom row and the store on the right, North the top row and the store
// on the left. A move picks up a pit of one's own and sows its seeds one
// by one counterclockwise, into one's own store but not the other's. A
// last seed in one's own store earns another move; a last seed in an
// empty pit of one's own takes it and the seeds across from it. When a
// row runs out, the other row's seeds go to its owner's store, and the
// fuller store wins. The pits the player to move can pick light up under
// the pointer, a click plays one, and the seeds drop in one after
// another. Beside the board, each store's count, whose turn it is, and
// the result.
//
//	ev := mancala.View(c, &app.board, &app.game)
//	if ev.Played && app.game.Turn == mancala.North {
//		app.game.Play(app.computer())
//	}
package mancala

import (
	"fmt"
	"math"
	"time"

	"github.com/egoist/mygo/ui"
)

// Player is a side of the board, or none once the game is over.
type Player int8

const (
	Nobody Player = iota
	South
	North
)

func (p Player) String() string { return [...]string{"Nobody", "South", "North"}[p] }

func (p Player) other() Player { return 3 - p }

// Stores are where South's and North's stores are in Game.Pits.
const (
	SouthStore = 6
	NorthStore = 13
)

// Game is a game: the seeds in each pit, counterclockwise from South's
// leftmost pit, South's store at 6 and North's at 13, and whose turn it
// is, Nobody once it is over.
type Game struct {
	Pits [14]int
	Turn Player
	// before, from and sown are the pits before the last move, the pit it
	// took, and where its seeds went, for showing them drop.
	before [14]int
	from   int
	sown   []int
	moves  int
}

// New returns a game about to start, four seeds in each pit and South to
// move.
func New() Game {
	g := Game{Turn: South}
	for i := range g.Pits {
		if i != SouthStore && i != NorthStore {
			g.Pits[i] = 4
		}
	}
	return g
}

// Owner returns who owns pit i.
func Owner(i int) Player {
	if i <= SouthStore {
		return South
	}
	return North
}

func store(p Player) int {
	if p == South {
		return SouthStore
	}
	return NorthStore
}

// Moves returns the pits the player to move can sow.
func (g *Game) Moves() []int {
	var out []int
	for i := range g.Pits {
		if i != SouthStore && i != NorthStore && Owner(i) == g.Turn && g.Pits[i] > 0 {
			out = append(out, i)
		}
	}
	return out
}

// Play sows pit i for the player to move and reports whether it could.
func (g *Game) Play(i int) bool {
	if i < 0 || i >= len(g.Pits) || i == SouthStore || i == NorthStore || Owner(i) != g.Turn || g.Pits[i] == 0 {
		return false
	}
	g.before, g.from, g.sown = g.Pits, i, nil
	g.moves++
	me := g.Turn
	seeds := g.Pits[i]
	g.Pits[i] = 0
	at := i
	for seeds > 0 {
		at = (at + 1) % len(g.Pits)
		if at == store(me.other()) {
			continue
		}
		g.Pits[at]++
		g.sown = append(g.sown, at)
		seeds--
	}
	if across := 12 - at; at != store(me) && Owner(at) == me && g.Pits[at] == 1 && g.Pits[across] > 0 {
		g.Pits[store(me)] += g.Pits[across] + 1
		g.Pits[at], g.Pits[across] = 0, 0
	}
	if at != store(me) {
		g.Turn = me.other()
	}
	south, north := 0, 0
	for k := range 6 {
		south += g.Pits[k]
		north += g.Pits[7+k]
	}
	if south == 0 || north == 0 {
		for k := range 6 {
			g.Pits[SouthStore] += g.Pits[k]
			g.Pits[NorthStore] += g.Pits[7+k]
			g.Pits[k], g.Pits[7+k] = 0, 0
		}
		g.Turn = Nobody
	}
	return true
}

// Winner returns who has more seeds in their store once the game is
// over, Nobody for a tie.
func (g *Game) Winner() Player {
	switch s, n := g.Pits[SouthStore], g.Pits[NorthStore]; {
	case g.Turn != Nobody || s == n:
		return Nobody
	case s > n:
		return South
	}
	return North
}

// State is what a board keeps: when the last move's seeds began to drop.
// Keep it in the app's state, one for each board.
type State struct {
	moves int
	sown  time.Time
}

// Events is a board, and whether a move was played on it in the frame.
type Events struct {
	ui.Element
	Played bool
}

const drop = 140 * time.Millisecond

var (
	wood     = ui.RGB(156, 104, 62)
	woodDark = ui.RGB(112, 70, 38)
	hole     = ui.RGB(92, 56, 30)
	colors   = []ui.Color{ui.RGB(232, 90, 80), ui.RGB(80, 160, 230), ui.RGB(250, 200, 70), ui.RGB(110, 200, 120), ui.RGB(200, 130, 220)}
)

// View creates the board of *g and its score.
func View(c *ui.Context, s *State, g *Game) Events {
	t := c.Theme()
	var ev Events
	now := c.Now()
	if s.moves != g.moves {
		s.moves, s.sown = g.moves, now
		if c.Preferences().ReduceMotion {
			s.sown = now.Add(-time.Hour)
		}
	}
	shown := g.Pits
	if n := int(now.Sub(s.sown) / drop); g.moves > 0 && n < len(g.sown) {
		shown = g.before
		shown[g.from] = 0
		for _, at := range g.sown[:n+1] {
			shown[at]++
		}
		c.After(drop - now.Sub(s.sown)%drop)
	}
	pit := t.Space(16)
	gap := t.Space(3)
	w := 8*pit + 9*gap
	h := 2*pit + 3*gap + t.Space(4)
	root := ui.Row(c).Gap(t.Space(8)).AlignItems(ui.Center)
	root.Children(func() {
		board := ui.Box(c).Size(w, h).Role(ui.RoleImage).Label("Mancala board")
		center := func(i int) (float32, float32) {
			switch {
			case i == NorthStore:
				return gap + pit/2, h / 2
			case i == SouthStore:
				return w - gap - pit/2, h / 2
			case i < SouthStore:
				return gap + pit + gap + float32(i)*(pit+gap) + pit/2, h - gap - pit/2 - t.Space(2)
			}
			return gap + pit + gap + float32(12-i)*(pit+gap) + pit/2, gap + pit/2 + t.Space(2)
		}
		hover := -1
		if px, py, ok := board.PointerPosition(); ok {
			for _, i := range g.Moves() {
				x, y := center(i)
				if math.Hypot(float64(px-x), float64(py-y)) < float64(pit/2) {
					hover = i
				}
			}
		}
		if hover >= 0 {
			board.Cursor(ui.CursorPointer)
			if board.Clicked() && g.Play(hover) {
				ev.Played = true
			}
		}
		board.Draw(func(p *ui.Painter, r ui.Rect) {
			p.FillGradient(r, ui.LinearGradient{From: wood, To: woodDark, Angle: 180}, h/2)
			for i := range g.Pits {
				x, y := center(i)
				x, y = r.X+x, r.Y+y
				box := ui.Rect{X: x - pit/2, Y: y - pit/2, W: pit, H: pit}
				radius := pit / 2
				if i == SouthStore || i == NorthStore {
					box = ui.Rect{X: x - pit/2, Y: r.Y + gap + t.Space(2), W: pit, H: h - 2*gap - t.Space(4)}
				}
				p.Fill(box, hole, radius)
				p.Shadow(box, radius, 0, 2, 6, 0, ui.RGBA(0, 0, 0, 0.35))
				if i == hover {
					p.Stroke(box, ui.RGBA(255, 236, 180, 0.9), radius, 2.5)
				}
				for k := range min(shown[i], 40) {
					sx, sy := seedAt(i, k, box)
					var seed ui.Path
					col := colors[(i*7+k*3)%len(colors)]
					p.FillPath(seed.Circle(sx, sy, pit*0.09), col)
					var shine ui.Path
					p.FillPath(shine.Circle(sx-pit*0.03, sy-pit*0.03, pit*0.03), ui.RGBA(255, 255, 255, 0.55))
				}
				label := ui.Span{Text: fmt.Sprint(shown[i]), Size: t.FontSize - 1, Weight: 700, Color: ui.RGBA(255, 244, 220, 0.9), Features: "tnum"}
				lw, lh := p.MeasureText(0, label)
				// Counts sit outside the pits, toward the board's edge, and
				// inside the stores at their foot.
				ly := box.Y - lh - 1
				switch {
				case i == SouthStore || i == NorthStore:
					ly = box.Y + box.H - lh - t.Space(1)
				case i < SouthStore:
					ly = box.Y + box.H + 1
				}
				p.RichText(x-lw/2, ly, 0, label)
			}
		})
		score(c, g)
	})
	if ev.Played {
		root.SetChanged()
	}
	ev.Element = root
	return ev
}

// seedAt returns where seed k sits in a pit or store: in rings from the
// middle, a little off true so that they look dropped.
func seedAt(i, k int, box ui.Rect) (float32, float32) {
	cx, cy := box.X+box.W/2, box.Y+box.H/2
	ring, idx, count := 0, k, 1
	for idx >= count {
		idx -= count
		ring++
		count = ring * 6
	}
	r := float64(ring) * float64(min(box.W, box.H)) * 0.15
	a := float64(idx)/float64(max(1, count))*2*math.Pi + float64(i)*0.7 + float64(ring)*0.4
	jx := float32(math.Sin(float64(i*31+k*17))) * box.W * 0.03
	jy := float32(math.Cos(float64(i*13+k*29))) * box.W * 0.03
	stretch := box.H / box.W
	return cx + float32(r*math.Cos(a)) + jx, cy + float32(r*math.Sin(a))*float32(max(1, stretch*0.8)) + jy
}

func score(c *ui.Context, g *Game) {
	t := c.Theme()
	ui.Column(c).Gap(t.Space(4)).Width(t.Space(40)).Children(func() {
		for _, p := range []Player{North, South} {
			row := ui.Row(c.Key(p)).Gap(t.Space(3)).AlignItems(ui.Center).Padding(t.Space(2)).Radius(t.Radius)
			if g.Turn == p {
				row.Background(t.Accent.Alpha(0.12))
			}
			row.Children(func() {
				ui.Text(c, p.String()).FontWeight(600).Grow(1)
				ui.Text(c, fmt.Sprint(g.Pits[store(p)])).FontSize(t.FontSize + 6).FontWeight(700).FontFeatures("tnum")
			})
		}
		msg := g.Turn.String() + " to move"
		switch {
		case g.Turn == Nobody && g.Winner() == Nobody:
			msg = "A tie"
		case g.Turn == Nobody:
			msg = g.Winner().String() + " wins"
		}
		ui.Text(c, msg).TextColor(t.TextMuted)
	})
}
