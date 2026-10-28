// Kakuro is a seven by seven kakuro with one answer: pick a cell, look at
// the sets of digits beside the grid, and type the digit.
//
//	go run ./examples/kakuro
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/kakuro"
)

// The answer, which the clues are worked out from.
const answer = `
#######
#79#981
#219834
##21#98
#98#21#
#142978
#839#21
`

type app struct {
	kakuro kakuro.State
	puzzle kakuro.Puzzle
	digits []int
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(8)).Gap(t.Space(5)).Children(func() {
		kakuro.View(c, &a.kakuro, a.puzzle, &a.digits)
		if ui.Button(c, "Clear").AlignSelf(ui.Start).Clicked() {
			a.digits = nil
		}
	})
}

func main() {
	p, err := kakuro.Parse(answer)
	if err != nil {
		log.Fatal(err)
	}
	a := &app{puzzle: p}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Kakuro",
			Width:   720,
			Height:  520,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
