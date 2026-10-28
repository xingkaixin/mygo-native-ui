// Backgammon sets up a game for two players taking turns at one board.
//
//	go run ./examples/backgammon
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/backgammon"
)

type app struct {
	board backgammon.State
	game  backgammon.Game
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Padding(t.Space(8)).AlignItems(ui.Center).Justify(ui.Center).Background(t.Background).Children(func() {
		backgammon.View(c, &a.board, &a.game)
	})
}

func main() {
	a := &app{game: backgammon.New()}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Backgammon",
			Width:   860,
			Height:  720,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
