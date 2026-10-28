// Fifteen is the fifteen puzzle, or a smaller or bigger one: click a tile
// in the gap's row or column, or use the arrow keys, to put the tiles
// back in order.
//
//	go run ./examples/fifteen
package main

import (
	"log"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/fifteen"
)

type app struct {
	puzzle fifteen.State
	board  fifteen.Board
	size   int
}

var sizes = []int{3, 4, 5}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(6)).Gap(t.Space(5)).AlignItems(ui.Center).Children(func() {
		if ui.Segmented(c, &a.size, "3 × 3", "4 × 4", "5 × 5").Label("Size").Changed() {
			a.board = fifteen.New(sizes[a.size], uint64(c.Now().UnixNano()))
		}
		fifteen.View(c, &a.puzzle, &a.board)
	})
}

func main() {
	a := &app{size: 1, board: fifteen.New(4, uint64(time.Now().UnixNano()))}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Fifteen",
			Width:   520,
			Height:  620,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
