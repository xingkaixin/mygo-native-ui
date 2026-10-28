// Draughts plays black against the computer's white, which looks four
// moves ahead and counts a king as one and a half men.
//
//	go run ./examples/draughts
package main

import (
	"log"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/draughts"
)

type app struct {
	board  draughts.State
	game   draughts.Game
	moveAt time.Time
}

// worth returns white's material lead.
func worth(g *draughts.Game) float64 {
	v := 0.0
	for _, row := range g.Board {
		for _, p := range row {
			w := 1.0
			if p.King {
				w = 1.5
			}
			switch p.Side {
			case draughts.White:
				v += w
			case draughts.Black:
				v -= w
			}
		}
	}
	return v
}

// best returns white's lead after depth moves, each side taking the move
// best for it, and the move that gets there.
func best(g draughts.Game, depth int) (float64, draughts.Move) {
	if g.Turn == draughts.Nobody {
		if g.Winner() == draughts.White {
			return 100, draughts.Move{}
		}
		return -100, draughts.Move{}
	}
	if depth == 0 {
		return worth(&g), draughts.Move{}
	}
	var pick draughts.Move
	score := 0.0
	for i, m := range g.Moves() {
		next := g
		next.Play(m)
		v, _ := best(next, depth-1)
		if i == 0 || (g.Turn == draughts.White && v > score) || (g.Turn == draughts.Black && v < score) {
			pick, score = m, v
		}
	}
	return score, pick
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	if a.game.Turn == draughts.White {
		if a.moveAt.IsZero() {
			a.moveAt = c.Now().Add(900 * time.Millisecond)
		}
		if wait := a.moveAt.Sub(c.Now()); wait > 0 {
			c.After(wait)
		} else {
			_, m := best(a.game, 4)
			a.game.Play(m)
			a.moveAt = time.Time{}
		}
	}
	ui.Column(c).Fill().Padding(t.Space(8)).Gap(t.Space(6)).AlignItems(ui.Center).Justify(ui.Center).Background(t.Background).Children(func() {
		draughts.View(c, &a.board, &a.game)
		if ui.Button(c, "New game").Clicked() {
			a.game, a.moveAt = draughts.New(), time.Time{}
		}
	})
}

func main() {
	a := &app{game: draughts.New()}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Draughts",
			Width:   820,
			Height:  620,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
