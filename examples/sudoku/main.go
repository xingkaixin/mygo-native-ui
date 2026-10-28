// Sudoku plays a classic puzzle: type digits or use the pad, turn Notes
// on with N or its toggle to pencil in candidates, and undo with Cmd+Z.
// A digit placed nine times dims on the pad.
//
//	go run ./examples/sudoku
package main

import (
	"fmt"
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/sudoku"
)

const puzzle = "53..7....6..195....98....6.8...6...34..8.3..17...2...6.6....28....419..5....8..79"

type app struct {
	play sudoku.State
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(8)).Gap(t.Space(6)).Children(func() {
		ui.Text(c, "Sudoku").FontSize(t.FontSize + 8).Bold()
		ui.Row(c).Gap(t.Space(8)).AlignItems(ui.Start).Children(func() {
			if sudoku.View(c, &a.play, puzzle).Solved {
				c.Toast("Solved!")
			}
			ui.Column(c).Gap(t.Space(3)).Width(t.Space(66)).Children(func() {
				ui.Grid(c).Columns(3).Gap(t.Space(2)).Children(func() {
					for d := 1; d <= 9; d++ {
						b := ui.Button(c.Key(d), fmt.Sprint(d)).Height(t.Space(12)).FontSize(t.FontSize + 6).KeepFocus()
						if a.play.Count(d) >= 9 {
							b.Opacity(0.35)
						}
						if b.Clicked() {
							a.play.Put(d)
						}
					}
				})
				ui.Row(c).Gap(t.Space(2)).Children(func() {
					ui.Toggle(c, &a.play.Notes, "Notes").Grow(1).KeepFocus()
					if ui.Button(c, "Erase").Grow(1).KeepFocus().Clicked() {
						a.play.Put(0)
					}
					if ui.Button(c, "Undo").Grow(1).KeepFocus().Clicked() {
						a.play.Undo()
					}
				})
			})
		})
	})
}

func main() {
	a := &app{}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Sudoku",
			Width:   820,
			Height:  620,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
