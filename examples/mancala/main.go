// Mancala plays South against the computer's North, which looks a few
// moves ahead for the fullest store.
//
//	go run ./examples/mancala
package main

import (
	"log"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/mancala"
)

type app struct {
	board  mancala.State
	game   mancala.Game
	moveAt time.Time
}

// best returns North's store lead after looking depth moves ahead, each
// side taking the move best for it, and the move that gets there.
func best(g mancala.Game, depth int) (int, int) {
	if depth == 0 || g.Turn == mancala.Nobody {
		return g.Pits[mancala.NorthStore] - g.Pits[mancala.SouthStore], -1
	}
	pick, score := -1, 0
	for _, m := range g.Moves() {
		next := g
		next.Play(m)
		v, _ := best(next, depth-1)
		if pick < 0 || (g.Turn == mancala.North && v > score) || (g.Turn == mancala.South && v < score) {
			pick, score = m, v
		}
	}
	return score, pick
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	if a.game.Turn == mancala.North {
		if a.moveAt.IsZero() {
			a.moveAt = c.Now().Add(1200 * time.Millisecond)
		}
		if wait := a.moveAt.Sub(c.Now()); wait > 0 {
			c.After(wait)
		} else {
			_, m := best(a.game, 6)
			a.game.Play(m)
			a.moveAt = time.Time{}
		}
	}
	ui.Column(c).Fill().Padding(t.Space(8)).Gap(t.Space(6)).AlignItems(ui.Center).Justify(ui.Center).Background(t.Background).Children(func() {
		mancala.View(c, &a.board, &a.game)
		if ui.Button(c, "New game").Clicked() {
			a.game, a.moveAt = mancala.New(), time.Time{}
		}
	})
}

func main() {
	a := &app{game: mancala.New()}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Mancala",
			Width:   1000,
			Height:  440,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
