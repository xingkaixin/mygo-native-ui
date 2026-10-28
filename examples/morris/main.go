// Morris plays nine men's morris against the computer, which plays
// black: place your nine pieces, then move them, and take one of black's
// whenever you make a mill.
//
//	go run ./examples/morris
package main

import (
	"log"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/morris"
)

type app struct {
	board    morris.State
	game     morris.Game
	computer bool
	due      time.Time
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	now := c.Now()
	if a.computer && a.game.Turn == morris.Black && a.game.Winner() == morris.Empty {
		// A pause before black plays, to see it happen.
		if a.due.IsZero() {
			a.due = now.Add(600 * time.Millisecond)
		}
		if now.Before(a.due) {
			c.After(a.due.Sub(now))
		} else {
			a.game.Auto()
			a.due = time.Time{}
		}
	}
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(6)).Gap(t.Space(4)).Children(func() {
		ui.Row(c).Gap(t.Space(4)).AlignItems(ui.Center).Children(func() {
			ui.Checkbox(c, &a.computer, "Computer plays black")
			ui.Spacer(c)
			if ui.Button(c, "New Game").Clicked() {
				a.game, a.due = morris.New(), time.Time{}
			}
		})
		morris.View(c, &a.board, &a.game).Grow(1).Basis(0)
	})
}

func main() {
	a := &app{game: morris.New(), computer: true}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Morris",
			Width:   820,
			Height:  620,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
