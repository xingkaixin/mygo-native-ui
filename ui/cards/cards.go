// Package cards draws playing cards and holds a hand of them. A card shows
// its rank and suit in two corners and its pips in the middle, the court
// cards a framed letter, or its back, and turns over as it changes sides.
// A hand fans its cards out; the card pointed at rises, a card dragged
// along the hand takes another place in it, the others making room, and a
// card dragged up out of the hand, or double-clicked, is played.
//
//	ev := cards.Hand(c, &app.hand, &app.cards)
//	if ev.Played != nil {
//		app.table = append(app.table, *ev.Played)
//	}
package cards

import (
	"math"
	"slices"

	"github.com/egoist/mygo/ui"
)

// Suit is a card's suit.
type Suit uint8

const (
	Spades Suit = iota
	Hearts
	Diamonds
	Clubs
)

var suits = [...]string{"♠", "♥", "♦", "♣"}
var suitNames = [...]string{"Spades", "Hearts", "Diamonds", "Clubs"}

// Card is a playing card: its rank, from 1 for the ace to 13 for the king,
// and its suit.
type Card struct {
	Rank int
	Suit Suit
}

var ranks = [...]string{"", "A", "2", "3", "4", "5", "6", "7", "8", "9", "10", "J", "Q", "K"}
var rankNames = [...]string{"", "Ace", "Two", "Three", "Four", "Five", "Six", "Seven", "Eight", "Nine", "Ten", "Jack", "Queen", "King"}

// String writes c as its rank and suit, such as "Q♥".
func (c Card) String() string { return ranks[c.Rank] + suits[c.Suit] }

func (c Card) name() string { return rankNames[c.Rank] + " of " + suitNames[c.Suit] }

// Deck returns the 52 cards, suit by suit, each from the ace to the king.
func Deck() []Card {
	var d []Card
	for s := Spades; s <= Clubs; s++ {
		for r := 1; r <= 13; r++ {
			d = append(d, Card{r, s})
		}
	}
	return d
}

var (
	red   = ui.Hex("#c8302c")
	black = ui.Hex("#1c1c1e")
	paper = ui.Hex("#fdfcf8")
)

func (c Card) color() ui.Color {
	if c.Suit == Hearts || c.Suit == Diamonds {
		return red
	}
	return black
}

// pips are where a card's pips stand, as fractions across and down its
// middle, by rank.
var pips = [...][][2]float32{
	2:  {{0.5, 0}, {0.5, 1}},
	3:  {{0.5, 0}, {0.5, 0.5}, {0.5, 1}},
	4:  {{0, 0}, {1, 0}, {0, 1}, {1, 1}},
	5:  {{0, 0}, {1, 0}, {0.5, 0.5}, {0, 1}, {1, 1}},
	6:  {{0, 0}, {1, 0}, {0, 0.5}, {1, 0.5}, {0, 1}, {1, 1}},
	7:  {{0, 0}, {1, 0}, {0.5, 0.25}, {0, 0.5}, {1, 0.5}, {0, 1}, {1, 1}},
	8:  {{0, 0}, {1, 0}, {0.5, 0.25}, {0, 0.5}, {1, 0.5}, {0.5, 0.75}, {0, 1}, {1, 1}},
	9:  {{0, 0}, {1, 0}, {0, 1.0 / 3}, {1, 1.0 / 3}, {0.5, 0.5}, {0, 2.0 / 3}, {1, 2.0 / 3}, {0, 1}, {1, 1}},
	10: {{0, 0}, {1, 0}, {0.5, 1.0 / 6}, {0, 1.0 / 3}, {1, 1.0 / 3}, {0, 2.0 / 3}, {1, 2.0 / 3}, {0.5, 5.0 / 6}, {0, 1}, {1, 1}},
}

// Size returns how wide and tall a card is.
func Size(c *ui.Context) (float32, float32) {
	t := c.Theme()
	return t.Space(24), t.Space(33.5)
}

// Face creates card, face up or down, turning over when up changes.
func Face(c *ui.Context, card Card, up bool) ui.Element {
	w, h := Size(c)
	holder := ui.Box(c).Size(w, h).Label(card.name()).Role(ui.RoleImage)
	if !up {
		holder.Label("Card face down")
	}
	target := float32(0)
	if !up {
		target = 180
	}
	turn := holder.AnimateTo("cards.turn", target, ui.Snappy)
	holder.Children(func() {
		front(c, card, w, h).RotateY(turn).Perspective(700).BackfaceHidden()
		back(c, w, h).RotateY(turn + 180).Perspective(700).BackfaceHidden()
	})
	return holder
}

// front creates the face of card.
func front(c *ui.Context, card Card, w, h float32) ui.Element {
	col := card.color()
	face := ui.Box(c).Fill().Absolute().Radius(w*0.08).Background(paper).Border(1, ui.RGBA(0, 0, 0, 0.15)).PassThrough()
	face.Children(func() {
		corner := func(flip bool) {
			e := ui.Column(c).Absolute().AlignItems(ui.Center).Width(w * 0.2).PassThrough()
			if flip {
				e.Right(w * 0.03).Bottom(h * 0.02).Rotate(180)
			} else {
				e.Left(w * 0.03).Top(h * 0.02)
			}
			e.Children(func() {
				ui.Text(c, ranks[card.Rank]).FontSize(w * 0.17).Bold().TextColor(col).LineHeight(1)
				ui.Text(c, suits[card.Suit]).FontSize(w * 0.15).TextColor(col).LineHeight(1)
			})
		}
		corner(false)
		corner(true)
		glyph := func(x, y, size float32, flip bool) {
			e := ui.Box(c).Absolute().Left(x-size/2).Top(y-size/2).Size(size, size).Center().PassThrough()
			if flip {
				e.Rotate(180)
			}
			e.Children(func() { ui.Text(c, suits[card.Suit]).FontSize(size).TextColor(col).LineHeight(1) })
		}
		switch {
		case card.Rank == 1:
			glyph(w/2, h/2, w*0.42, false)
		case card.Rank > 10:
			ui.Column(c).Absolute().Left(w*0.22).Top(h*0.18).Size(w*0.56, h*0.64).Radius(w*0.04).
				Border(1.5, col.Alpha(0.5)).Background(col.Alpha(0.06)).Center().PassThrough().Children(func() {
				ui.Text(c, ranks[card.Rank]).Font("serif").FontSize(w * 0.4).Bold().TextColor(col).LineHeight(1)
				ui.Text(c, suits[card.Suit]).FontSize(w * 0.2).TextColor(col).LineHeight(1)
			})
		default:
			x0, x1 := w*0.34, w*0.66
			y0, y1 := h*0.22, h*0.78
			for _, p := range pips[card.Rank] {
				glyph(x0+(x1-x0)*p[0], y0+(y1-y0)*p[1], w*0.17, p[1] > 0.5)
			}
		}
	})
	return face
}

// back creates the back of a card.
func back(c *ui.Context, w, h float32) ui.Element {
	t := c.Theme()
	return ui.Box(c).Fill().Absolute().Radius(w*0.08).Background(paper).Border(1, ui.RGBA(0, 0, 0, 0.15)).PassThrough().
		Draw(func(p *ui.Painter, r ui.Rect) {
			in := ui.Rect{X: r.X + w*0.07, Y: r.Y + w*0.07, W: r.W - w*0.14, H: r.H - w*0.14}
			p.Fill(in, t.Accent, w*0.04)
			line := ui.RGBA(255, 255, 255, 0.28)
			step := w * 0.1
			for d := -in.H; d < in.W; d += step {
				var a, b ui.Path
				x0, x1 := max(in.X, in.X+d), min(in.X+in.W, in.X+d+in.H)
				a.MoveTo(x0, in.Y+(x0-in.X-d))
				a.LineTo(x1, in.Y+(x1-in.X-d))
				p.StrokePath(&a, 1, line)
				x0, x1 = max(in.X, in.X+in.W-d-in.H), min(in.X+in.W, in.X+in.W-d)
				b.MoveTo(x0, in.Y+(in.X+in.W-d-x0))
				b.LineTo(x1, in.Y+(in.X+in.W-d-x1))
				p.StrokePath(&b, 1, line)
			}
		})
}

// State is what a hand keeps: the card being dragged, and where it was
// taken hold of. Keep it in the app's state, one for each hand.
type State struct {
	dragging     bool
	dragged      Card
	grabX, grabY float32
}

// Events is a hand, and the card played from it in the frame: Played, nil
// when none was. The card is out of the hand by then.
type Events struct {
	ui.Element
	Played *Card
}

// Hand creates the hand of *hand.
func Hand(c *ui.Context, s *State, hand *[]Card) Events {
	t := c.Theme()
	w, h := Size(c)
	var ev Events
	changed := false
	cards := *hand
	holder := ui.Box(c).Height(h + t.Space(20)).Shrink(0).Label("Hand").Role(ui.RoleGroup)
	b := holder.LayoutBounds()
	n := len(cards)
	px, py, _ := holder.PointerPosition()
	if s.dragging && !slices.Contains(cards, s.dragged) {
		s.dragging = false
	}
	step := float32(0)
	if n > 1 {
		step = min(w*0.62, (b.W-w-t.Space(4))/float32(n-1))
	}
	spread := min(7, 42/float32(max(1, n)))
	place := func(k int) (x, y, angle float32) {
		off := float32(k) - float32(n-1)/2
		return b.W/2 + off*step - w/2, t.Space(8) + off*off*step*0.035, off * spread
	}
	order := cards
	to := -1
	if s.dragging {
		order = slices.DeleteFunc(slices.Clone(cards), func(x Card) bool { return x == s.dragged })
		to = n / 2
		if step > 0 {
			k := math.Round(float64((px-s.grabX+w/2-b.W/2)/step + float32(n-1)/2))
			to = max(0, min(n-1, int(k)))
		}
		order = slices.Insert(order, to, s.dragged)
	}
	hovered := -1
	if holder.Hovered() && !s.dragging {
		for k := range order {
			x, y, _ := place(k)
			if px >= x && px < x+w && py >= y-t.Space(6) && py < y+h {
				hovered = k
			}
		}
	}
	// A drag that starts while the cards are built takes them over from
	// the next frame.
	dragging := s.dragging
	holder.Children(func() {
		for k, card := range order {
			if dragging && card == s.dragged {
				continue
			}
			x, y, a := place(k)
			if k == hovered {
				y, a = y-t.Space(6), a*0.4
			}
			e := Face(c.Key(card), card, true)
			x = e.AnimateTo("cards.x", x, ui.Snappy)
			y = e.AnimateTo("cards.y", y, ui.Snappy)
			a = e.AnimateTo("cards.a", a, ui.Snappy)
			e.Absolute().Left(x).Top(y).Rotate(a).Shadow(0, 2, 6, 0, ui.RGBA(0, 0, 0, 0.18)).Cursor(ui.CursorGrab)
			if _, _, moved := e.Dragged(); moved && !s.dragging {
				s.dragging, s.dragged, s.grabX, s.grabY = true, card, px-x, py-y
			}
			if e.DoubleClicked() {
				played := card
				*hand = slices.DeleteFunc(slices.Clone(cards), func(x Card) bool { return x == card })
				ev.Played, changed = &played, true
			}
		}
		if !dragging {
			return
		}
		// The card dragged is built last, so that it passes over the others.
		x, y := px-s.grabX, py-s.grabY
		e := Face(c.Key(s.dragged), s.dragged, true)
		e.AnimateTo("cards.x", x, ui.Snappy)
		e.AnimateTo("cards.y", y, ui.Snappy)
		e.AnimateTo("cards.a", 0, ui.Snappy)
		e.Absolute().Left(x).Top(y).Shadow(0, 12, 24, 0, ui.RGBA(0, 0, 0, 0.3)).Cursor(ui.CursorGrabbing)
		if e.Pressed() {
			return
		}
		s.dragging, changed = false, true
		rest := slices.DeleteFunc(slices.Clone(cards), func(x Card) bool { return x == s.dragged })
		if y < -h*0.4 {
			played := s.dragged
			*hand, ev.Played = rest, &played
		} else {
			*hand = slices.Insert(rest, to, s.dragged)
		}
	})
	if changed {
		holder.SetChanged()
	}
	ev.Element = holder
	return ev
}
