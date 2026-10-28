// Karnaugh simplifies the segments of a BCD to seven-segment decoder:
// inputs A B C D are a digit from 0 to 9, codes 10 to 15 never come and
// are don't cares. Pick a segment to load its function, click cells or
// rows to change it, and the sum of products follows.
//
//	go run ./examples/karnaugh
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/karnaugh"
)

var (
	vars = []string{"A", "B", "C", "D"}
	// lit lists the digits each segment, a to g, is lit for.
	lit = [][]int{
		{0, 2, 3, 5, 6, 7, 8, 9},
		{0, 1, 2, 3, 4, 7, 8, 9},
		{0, 1, 3, 4, 5, 6, 7, 8, 9},
		{0, 2, 3, 5, 6, 8, 9},
		{0, 2, 6, 8},
		{0, 4, 5, 6, 8, 9},
		{2, 3, 4, 5, 6, 8, 9},
	}
	segment = 4
	cells   []karnaugh.Cell
)

func load() {
	cells = make([]karnaugh.Cell, 16)
	for _, d := range lit[segment] {
		cells[d] = karnaugh.One
	}
	for m := 10; m < 16; m++ {
		cells[m] = karnaugh.DontCare
	}
}

func view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(6)).Gap(t.Space(6)).Children(func() {
		ui.Row(c).Gap(t.Space(4)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "Segment").Bold().FontSize(t.FontSize + 4)
			if ui.Segmented(c, &segment, "a", "b", "c", "d", "e", "f", "g").Label("Segment").Changed() {
				load()
			}
		})
		karnaugh.View(c, vars, &cells)
	})
}

func main() {
	load()
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Karnaugh",
			Width:   900,
			Height:  760,
			Content: ui.View(view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
