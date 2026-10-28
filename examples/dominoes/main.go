// Dominoes plays the draw game against the computer, round after round,
// keeping the score: play a lifted tile, pick its end when it fits both,
// and draw when nothing fits.
//
//	go run ./examples/dominoes
package main

import (
	"fmt"
	"log"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/dominoes"
)

type app struct {
	table  dominoes.State
	game   dominoes.Game
	scores [2]int
	due    time.Time
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	now := c.Now()
	if _, over := a.game.Over(); !over && a.game.Turn == 1 {
		// A pause before the other side plays, to see it happen.
		if a.due.IsZero() {
			a.due = now.Add(700 * time.Millisecond)
		}
		if now.Before(a.due) {
			c.After(a.due.Sub(now))
		} else {
			a.game.Auto()
			a.due = time.Time{}
			if w, over := a.game.Over(); over && w >= 0 {
				a.scores[w] += a.game.Score()
			}
		}
	}
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(6)).Gap(t.Space(5)).Children(func() {
		ui.Row(c).Gap(t.Space(4)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, fmt.Sprintf("You %d · Them %d", a.scores[0], a.scores[1])).FontFeatures("tnum").Bold()
			ui.Spacer(c)
			if _, over := a.game.Over(); over && ui.Button(c, "Next Round").Clicked() {
				a.game, a.due = dominoes.New(uint64(now.UnixNano())), time.Time{}
			}
		})
		ev := dominoes.View(c, &a.table, &a.game)
		ev.Grow(1)
		if w, over := a.game.Over(); over && ev.Moved && w >= 0 {
			a.scores[w] += a.game.Score()
		}
	})
}

func main() {
	a := &app{game: dominoes.New(uint64(time.Now().UnixNano()))}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Dominoes",
			Width:   900,
			Height:  520,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
