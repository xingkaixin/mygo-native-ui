// Package solitaire plays Klondike on a felt table: the stock, the waste,
// four foundations to build up by suit from the ace to the king, and
// seven tableau piles to build down in alternating colors, a king alone
// on an empty one. A click on the stock deals one card, or three, to the
// waste, and turns the waste back over once the stock is empty. Cards
// are dragged where they go, a face-up card in a tableau pile taking
// those on it along, and a double click sends a card to its foundation.
// A card left face down on top of a pile turns over, cards fly to where
// they land and back when they cannot, and once every card is face up the
// rest go to the foundations by themselves. Above the table, the moves,
// the time, and Undo, as ⌘Z.
//
//	solitaire.View(c, &app.table, &app.game)
//	if app.game.Won() {
//		...
//	}
package solitaire

import (
	"fmt"
	"math"
	"math/rand/v2"
	"slices"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/cards"
)

// Placed is a card where it lies, face up or down.
type Placed struct {
	cards.Card
	Up bool
}

// Game is a game: the stock, its top last; the waste, face up, its top
// last; the foundations and the tableau piles, each from the bottom; how
// many cards a deal turns, 1 or 3; and how many moves were made.
type Game struct {
	Stock, Waste []Placed
	Foundations  [4][]Placed
	Tableau      [7][]Placed
	Draw         int
	Moves        int
	history      []Game
}

// New returns a game shuffled by seed, dealing draw cards at a time:
// seven tableau piles of one to seven cards, the top one face up, and the
// rest of the deck in the stock.
func New(seed uint64, draw int) Game {
	deck := cards.Deck()
	rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15)).Shuffle(len(deck), func(i, j int) { deck[i], deck[j] = deck[j], deck[i] })
	g := Game{Draw: max(1, draw)}
	for i := range g.Tableau {
		for j := 0; j <= i; j++ {
			g.Tableau[i] = append(g.Tableau[i], Placed{deck[0], j == i})
			deck = deck[1:]
		}
	}
	for _, card := range deck {
		g.Stock = append(g.Stock, Placed{card, false})
	}
	return g
}

type kind uint8

const (
	waste kind = iota
	foundation
	tableau
)

// place is a pile: the waste, or a foundation or tableau pile by number.
type place struct {
	kind kind
	i    int
}

func (g *Game) at(p place) *[]Placed {
	switch p.kind {
	case waste:
		return &g.Waste
	case foundation:
		return &g.Foundations[p.i]
	}
	return &g.Tableau[p.i]
}

func red(c cards.Card) bool { return c.Suit == cards.Hearts || c.Suit == cards.Diamonds }

// movable reports whether the cards from k up in pile p can move: the top
// of the waste or a foundation, or any face-up card in a tableau pile,
// whose cards on it are always in sequence.
func (g *Game) movable(p place, k int) bool {
	pile := *g.at(p)
	if k < 0 || k >= len(pile) || !pile[k].Up {
		return false
	}
	return p.kind == tableau || k == len(pile)-1
}

// fits reports whether run can go on pile to.
func (g *Game) fits(run []Placed, to place) bool {
	pile := *g.at(to)
	first := run[0].Card
	switch to.kind {
	case foundation:
		if len(run) != 1 {
			return false
		}
		if len(pile) == 0 {
			return first.Rank == 1
		}
		top := pile[len(pile)-1].Card
		return top.Suit == first.Suit && first.Rank == top.Rank+1
	case tableau:
		if len(pile) == 0 {
			return first.Rank == 13
		}
		top := pile[len(pile)-1]
		return top.Up && red(top.Card) != red(first) && first.Rank == top.Rank-1
	}
	return false
}

func (g *Game) clone() Game {
	c := *g
	c.history = nil
	c.Stock, c.Waste = slices.Clone(g.Stock), slices.Clone(g.Waste)
	for i := range c.Foundations {
		c.Foundations[i] = slices.Clone(g.Foundations[i])
	}
	for i := range c.Tableau {
		c.Tableau[i] = slices.Clone(g.Tableau[i])
	}
	return c
}

func (g *Game) save() { g.history = append(g.history, g.clone()) }

// move moves the cards from k up in pile from onto pile to, turning over
// the card left on top, and reports whether it could.
func (g *Game) move(from place, k int, to place) bool {
	src := g.at(from)
	if from == to || !g.movable(from, k) || !g.fits((*src)[k:], to) {
		return false
	}
	g.save()
	run := slices.Clone((*src)[k:])
	*src = (*src)[:k]
	if from.kind == tableau && k > 0 {
		(*src)[k-1].Up = true
	}
	*g.at(to) = append(*g.at(to), run...)
	g.Moves++
	return true
}

// home moves the top card of pile p to the foundation it fits, and
// reports whether there was one.
func (g *Game) home(p place) bool {
	k := len(*g.at(p)) - 1
	for i := range g.Foundations {
		if g.move(p, k, place{foundation, i}) {
			return true
		}
	}
	return false
}

// Deal turns the next card, or three, of the stock onto the waste, or the
// waste back over into the stock once the stock is empty.
func (g *Game) Deal() {
	if len(g.Stock) == 0 && len(g.Waste) == 0 {
		return
	}
	g.save()
	if len(g.Stock) == 0 {
		for i := len(g.Waste) - 1; i >= 0; i-- {
			g.Stock = append(g.Stock, Placed{g.Waste[i].Card, false})
		}
		g.Waste = nil
	} else {
		for range min(max(1, g.Draw), len(g.Stock)) {
			top := g.Stock[len(g.Stock)-1]
			g.Stock = g.Stock[:len(g.Stock)-1]
			g.Waste = append(g.Waste, Placed{top.Card, true})
		}
	}
	g.Moves++
}

// CanUndo reports whether there is a move to undo.
func (g *Game) CanUndo() bool { return len(g.history) > 0 }

// Undo takes back the last move.
func (g *Game) Undo() {
	if n := len(g.history); n > 0 {
		prev := g.history[n-1]
		prev.history = g.history[:n-1]
		*g = prev
	}
}

// Won reports whether every card is on a foundation.
func (g *Game) Won() bool {
	for _, f := range g.Foundations {
		if len(f) < 13 {
			return false
		}
	}
	return true
}

// open reports whether every card left is face up in the tableau, when
// the game plays itself out.
func (g *Game) open() bool {
	if len(g.Stock) > 0 || len(g.Waste) > 0 {
		return false
	}
	for _, pile := range g.Tableau {
		for _, card := range pile {
			if !card.Up {
				return false
			}
		}
	}
	return true
}

// State is what a table keeps: the cards being dragged and where they
// were taken hold of, when play began and ended, and when the next card
// goes home by itself. Keep it in the app's state, one for each table.
type State struct {
	dragging     bool
	from         place
	k            int
	grabX, grabY float32
	started      time.Time
	ended        time.Time
	next         time.Time
}

var felt = ui.Hex("#2E6B40")

// View creates the table of *g and, above it, its moves, time and Undo.
func View(c *ui.Context, s *State, g *Game) ui.Element {
	t := c.Theme()
	now := c.Now()
	w, h := cards.Size(c)
	gap := t.Space(3)
	changed := false
	if g.Moves == 0 {
		s.started, s.ended = time.Time{}, time.Time{}
	} else if s.started.IsZero() {
		s.started = now
	}
	if g.Won() && s.ended.IsZero() {
		s.ended = now
	}
	if g.open() && !g.Won() {
		if now.Before(s.next) {
			c.After(s.next.Sub(now))
		} else {
			for i := range g.Tableau {
				if len(g.Tableau[i]) > 0 && g.home(place{tableau, i}) {
					break
				}
			}
			s.next, changed = now.Add(90*time.Millisecond), true
			c.After(90 * time.Millisecond)
		}
	}
	root := ui.Column(c).Gap(t.Space(3)).Width(7*w + 8*gap)
	if root.Shortcut(ui.Cmd, ui.KeyZ) && g.CanUndo() {
		g.Undo()
		s.dragging, changed = false, true
	}
	root.Children(func() {
		ui.Row(c).Gap(t.Space(4)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, fmt.Sprintf("Moves %d", g.Moves)).FontFeatures("tnum")
			elapsed := time.Duration(0)
			switch {
			case !s.ended.IsZero():
				elapsed = s.ended.Sub(s.started)
			case !s.started.IsZero():
				elapsed = now.Sub(s.started)
				c.After(time.Second - elapsed%time.Second)
			}
			ui.Text(c, fmt.Sprintf("%d:%02d", int(elapsed.Minutes()), int(elapsed.Seconds())%60)).FontFeatures("tnum").TextColor(t.TextMuted)
			if g.Won() {
				ui.Text(c, "Solved").Bold().TextColor(t.Success)
			}
			ui.Spacer(c)
			if ui.Button(c, "Undo").Disabled(!g.CanUndo()).Clicked() {
				g.Undo()
				s.dragging, changed = false, true
			}
		})
		if s.table(c, g, w, h, gap) {
			changed = true
		}
	})
	if changed {
		root.SetChanged()
	}
	return root
}

// table creates the felt and the cards on it, and reports whether a move
// was made.
func (s *State) table(c *ui.Context, g *Game, w, h, gap float32) bool {
	t := c.Theme()
	var act func() bool
	board := ui.Box(c).FillWidth().Grow(1).MinHeight(h*3 + gap*4).Radius(t.Radius * 2).Background(felt).Label("Klondike table").Role(ui.RoleGroup)
	b := board.LayoutBounds()
	px, py, _ := board.PointerPosition()
	col := func(i int) float32 { return gap + float32(i)*(w+gap) }
	row := gap*2 + h
	// fans returns how far apart pile i fans its face-down and face-up
	// cards, closer when it would run off the table.
	fans := func(i int) (float32, float32) {
		down, up := h*0.1, h*0.25
		pile := g.Tableau[i]
		var need float32
		for _, pc := range pile[:max(0, len(pile)-1)] {
			if pc.Up {
				need += up
			} else {
				need += down
			}
		}
		if room := b.H - row - h - gap; need > room && room > 0 {
			f := room / need
			down, up = down*f, up*f
		}
		return down, up
	}
	if s.dragging && !g.movable(s.from, s.k) {
		s.dragging = false
	}
	dragging := s.dragging
	board.Children(func() {
		slot := func(key string, x, y float32, mark string) ui.Element {
			e := ui.Box(c.Key(key)).Absolute().Left(x).Top(y).Size(w, h).Radius(w*0.08).Border(1.5, ui.RGBA(255, 255, 255, 0.28)).Center()
			e.Children(func() {
				ui.Text(c, mark).FontSize(w * 0.3).TextColor(ui.RGBA(255, 255, 255, 0.3))
			})
			return e
		}
		if slot("stock", col(0), gap, "↻").Clicked() {
			act = deal(g)
		}
		for i := range g.Foundations {
			slot(fmt.Sprint("foundation", i), col(3+i), gap, "A")
		}
		for i := range g.Tableau {
			slot(fmt.Sprint("tableau", i), col(i), row, "K")
		}
		card := func(p place, k int, pc Placed, x, y float32) {
			e := cards.Face(c.Key(pc.Card), pc.Card, pc.Up)
			x = e.AnimateTo("solitaire.x", x, ui.Snappy)
			y = e.AnimateTo("solitaire.y", y, ui.Snappy)
			e.Absolute().Left(x).Top(y)
			if !g.movable(p, k) {
				return
			}
			e.Cursor(ui.CursorGrab)
			if _, _, drag := e.Dragged(); drag && !s.dragging {
				s.dragging, s.from, s.k, s.grabX, s.grabY = true, p, k, px-x, py-y
			}
			if e.DoubleClicked() && k == len(*g.at(p))-1 {
				act = func() bool { return g.home(p) }
			}
		}
		for k, st := range g.Stock {
			e := cards.Face(c.Key(st.Card), st.Card, false)
			x := e.AnimateTo("solitaire.x", col(0), ui.Snappy)
			y := e.AnimateTo("solitaire.y", gap, ui.Snappy)
			e.Absolute().Left(x).Top(y)
			if k == len(g.Stock)-1 && e.Clicked() {
				act = deal(g)
			}
		}
		// Of the waste, the last cards dealt fan out, as many as a deal
		// turns.
		fanned := len(g.Waste) - min(len(g.Waste), max(1, g.Draw))
		for k, wc := range g.Waste {
			if dragging && s.from.kind == waste && k >= s.k {
				continue
			}
			x := col(1) + float32(max(0, k-fanned))*w*0.24
			card(place{waste, 0}, k, wc, x, gap)
		}
		for i, pile := range g.Foundations {
			for k, fc := range pile {
				if dragging && s.from == (place{foundation, i}) && k >= s.k {
					continue
				}
				card(place{foundation, i}, k, fc, col(3+i), gap)
			}
		}
		for i, pile := range g.Tableau {
			down, up := fans(i)
			y := row
			for k, tc := range pile {
				if dragging && s.from == (place{tableau, i}) && k >= s.k {
					break
				}
				card(place{tableau, i}, k, tc, col(i), y)
				if tc.Up {
					y += up
				} else {
					y += down
				}
			}
		}
		if !dragging {
			return
		}
		// The cards dragged are built last, so that they pass over the
		// rest.
		run := (*g.at(s.from))[s.k:]
		_, up := fans(0)
		up = max(up, h*0.25)
		x, y := px-s.grabX, py-s.grabY
		held := false
		for j, rc := range run {
			e := cards.Face(c.Key(rc.Card), rc.Card, true)
			e.AnimateTo("solitaire.x", x, ui.Snappy)
			e.AnimateTo("solitaire.y", y+float32(j)*up, ui.Snappy)
			e.Absolute().Left(x).Top(y+float32(j)*up).Shadow(0, 10, 20, 0, ui.RGBA(0, 0, 0, 0.35)).Cursor(ui.CursorGrabbing)
			if j == 0 {
				held = e.Pressed()
			}
		}
		if held {
			return
		}
		s.dragging = false
		cx, cy := x+w/2, y+h/2
		i := max(0, min(6, int(math.Floor(float64((cx-gap/2)/(w+gap))))))
		to := place{tableau, i}
		if cy < row {
			if i < 3 {
				return
			}
			to = place{foundation, i - 3}
		}
		from, k := s.from, s.k
		act = func() bool { return g.move(from, k, to) }
	})
	// Moves wait until the cards are built, which a card moved while they
	// are would be twice.
	return act != nil && act()
}

func deal(g *Game) func() bool {
	return func() bool {
		g.Deal()
		return true
	}
}
