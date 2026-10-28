// Cards deals from a shuffled deck: click the deck to draw, drag the cards
// of the hand into order, and drag one up to the table, or double-click
// it, to play it there face up.
//
//	go run ./examples/cards
package main

import (
	"cmp"
	"log"
	"math/rand/v2"
	"slices"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/cards"
)

type played struct {
	card cards.Card
	up   bool
}

type app struct {
	deck  []cards.Card
	hand  []cards.Card
	held  cards.State
	table []played
}

func (a *app) deal() {
	a.deck = cards.Deck()
	rand.Shuffle(len(a.deck), func(i, j int) { a.deck[i], a.deck[j] = a.deck[j], a.deck[i] })
	a.hand, a.table = a.deck[:7:7], nil
	a.deck = a.deck[7:]
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	w, h := cards.Size(c)
	felt := ui.Hex("#1f5f43")
	ui.Column(c).Fill().Background(felt).Padding(t.Space(8)).Gap(t.Space(6)).Children(func() {
		ui.Row(c).Gap(t.Space(2)).Children(func() {
			ui.Box(c).Grow(1)
			if ui.Button(c, "By suit").Clicked() {
				slices.SortFunc(a.hand, func(x, y cards.Card) int { return cmp.Or(cmp.Compare(x.Suit, y.Suit), cmp.Compare(x.Rank, y.Rank)) })
			}
			if ui.Button(c, "By rank").Clicked() {
				slices.SortFunc(a.hand, func(x, y cards.Card) int { return cmp.Or(cmp.Compare(x.Rank, y.Rank), cmp.Compare(x.Suit, y.Suit)) })
			}
			if ui.Button(c, "Deal again").Clicked() {
				a.deal()
			}
		})
		ui.Row(c).Grow(1).Gap(t.Space(12)).AlignItems(ui.Center).Justify(ui.Center).Children(func() {
			deck := ui.ButtonBase(c).Size(w+6, h+6).Label("Draw a card").Disabled(len(a.deck) == 0 || len(a.hand) >= 12)
			deck.Children(func() {
				for i := range min(len(a.deck), 4) {
					cards.Face(c.Key(i), cards.Card{Rank: 1}, false).Absolute().Left(float32(i) * 1.5).Top(float32(i) * 1.5)
				}
			})
			if deck.Clicked() {
				a.hand = append(a.hand, a.deck[0])
				a.deck = a.deck[1:]
			}
			ui.Box(c).Size(w+t.Space(30), h+t.Space(8)).Children(func() {
				first := max(0, len(a.table)-6)
				for i, p := range a.table[first:] {
					e := cards.Face(c.Key(p.card), p.card, p.up).Absolute().Left(float32(i) * t.Space(6)).Top(float32(i%2) * t.Space(2)).
						Rotate(float32((i*37)%11) - 5)
					e.Shadow(0, 2, 6, 0, ui.RGBA(0, 0, 0, 0.25))
				}
			})
		})
		ev := cards.Hand(c, &a.held, &a.hand)
		if ev.Played != nil {
			a.table = append(a.table, played{card: *ev.Played})
		}
	})
	for i := range a.table {
		if !a.table[i].up {
			a.table[i].up = true
			c.Invalidate()
		}
	}
}

func main() {
	a := &app{}
	a.deal()
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Cards",
			Width:   900,
			Height:  640,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
