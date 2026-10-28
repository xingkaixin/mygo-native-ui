// Solitaire deals a game of Klondike: click the stock, drag cards between
// the piles, double-click to send a card home, and undo with ⌘Z. New game
// shuffles again, dealing one card or three at a time.
//
//	go run ./examples/solitaire
package main

import (
	"fmt"
	"log"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/solitaire"
)

type app struct {
	table solitaire.State
	game  solitaire.Game
	draw  int
	won   bool
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Padding(t.Space(6)).Gap(t.Space(4)).AlignItems(ui.Center).Background(t.Background).Children(func() {
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			ui.Segmented(c, &a.draw, "Draw 1", "Draw 3").Label("Cards dealt")
			if ui.Button(c, "New game").Clicked() {
				a.game, a.won = solitaire.New(uint64(time.Now().UnixNano()), 1+2*a.draw), false
			}
		})
		ui.Column(c).Grow(1).Children(func() {
			solitaire.View(c, &a.table, &a.game).Grow(1)
		})
		if a.game.Won() && !a.won {
			a.won = true
			c.Toast(fmt.Sprintf("Solved in %d moves", a.game.Moves))
		}
	})
}

func main() {
	a := &app{game: solitaire.New(7, 1)}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Solitaire",
			Width:   900,
			Height:  760,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
