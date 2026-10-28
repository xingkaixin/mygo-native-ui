// Gomoku plays five in a row on a 15-line board, against the computer as
// white or with two at the keyboard: make an unbroken line of five
// before the other side does.
//
//	go run ./examples/gomoku
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/goban"
	"github.com/xingkaixin/mygo-native-ui/ui/gomoku"
)

type app struct {
	gomoku   gomoku.State
	game     gomoku.Game
	computer bool
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(6)).Gap(t.Space(3)).Children(func() {
		ui.Row(c).Gap(t.Space(4)).AlignItems(ui.Center).Children(func() {
			ui.Checkbox(c, &a.computer, "Computer plays white")
			ui.Spacer(c)
			if ui.Button(c, "New Game").Clicked() {
				a.game = gomoku.New(15)
			}
		})
		ev := gomoku.View(c, &a.gomoku, &a.game)
		ev.Grow(1).Basis(0)
		if w, _ := a.game.Winner(); a.computer && w == goban.Empty && a.game.Turn() == goban.White {
			if ev.Undone {
				a.game.Undo()
			} else {
				a.game.Play(a.game.Suggest())
			}
		}
	})
}

func main() {
	a := &app{game: gomoku.New(15), computer: true}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Gomoku",
			Width:   640,
			Height:  720,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
