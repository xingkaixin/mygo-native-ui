// Reversi plays black against the computer's white, which takes corners
// and edges where it can and keeps off the squares beside them.
//
//	go run ./examples/reversi
package main

import (
	"log"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/reversi"
)

// weights are what each square is worth to hold.
var weights = [8][8]int{
	{100, -20, 10, 5, 5, 10, -20, 100},
	{-20, -50, -2, -2, -2, -2, -50, -20},
	{10, -2, 1, 1, 1, 1, -2, 10},
	{5, -2, 1, 0, 0, 1, -2, 5},
	{5, -2, 1, 0, 0, 1, -2, 5},
	{10, -2, 1, 1, 1, 1, -2, 10},
	{-20, -50, -2, -2, -2, -2, -50, -20},
	{100, -20, 10, 5, 5, 10, -20, 100},
}

type app struct {
	board  reversi.State
	game   reversi.Game
	moveAt time.Time
}

// computer picks white's move: the square worth most, with the discs it
// turns.
func (a *app) computer() reversi.Square {
	best, score := reversi.Square{}, -1<<30
	for _, m := range a.game.Moves(reversi.White) {
		v := weights[m.Row][m.Col]
		for _, f := range a.game.Flips(m, reversi.White) {
			v += 1 + weights[f.Row][f.Col]/10
		}
		if v > score {
			best, score = m, v
		}
	}
	return best
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	if a.game.Turn == reversi.White {
		if a.moveAt.IsZero() {
			a.moveAt = c.Now().Add(700 * time.Millisecond)
		}
		if wait := a.moveAt.Sub(c.Now()); wait > 0 {
			c.After(wait)
		} else {
			a.game.Play(a.computer())
			a.moveAt = time.Time{}
		}
	}
	ui.Column(c).Fill().Padding(t.Space(8)).Gap(t.Space(6)).AlignItems(ui.Center).Justify(ui.Center).Background(t.Background).Children(func() {
		reversi.View(c, &a.board, &a.game)
		if ui.Button(c, "New game").Clicked() {
			a.game, a.moveAt = reversi.New(), time.Time{}
		}
	})
}

func main() {
	a := &app{game: reversi.New()}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Reversi",
			Width:   780,
			Height:  560,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
