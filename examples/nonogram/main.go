// Nonogram plays three picross puzzles of growing size: a heart, a cat
// and a sailboat. Fill squares with clicks or drags, cross them out with
// right clicks, and pick another puzzle at the top.
//
//	go run ./examples/nonogram
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/nonogram"
)

var puzzles = []struct {
	name    string
	picture []string
}{
	{"Heart", []string{
		".##.##.",
		"#######",
		"#######",
		".#####.",
		"..###..",
		"...#...",
	}},
	{"Cat", []string{
		"#........#",
		"##......##",
		"###....###",
		"##########",
		"#.##..##.#",
		"##########",
		"####..####",
		".###..###.",
		"..######..",
		"...####...",
	}},
	{"Sailboat", []string{
		".......#.......",
		".......##......",
		"......####.....",
		"......#####....",
		".....#######...",
		".....########..",
		"....##########.",
		"....###########",
		".......#.......",
		".......#.......",
		"###############",
		".#############.",
		"..###########..",
		".##..##..##..##",
		"##..##..##..##.",
	}},
}

type app struct {
	games  [3]nonogram.State
	pick   int
	solved [3]bool
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(8)).Gap(t.Space(5)).Children(func() {
		ui.Row(c).Gap(t.Space(4)).AlignItems(ui.Center).Children(func() {
			title := "Picross"
			if a.solved[a.pick] {
				title = puzzles[a.pick].name
			}
			ui.Text(c, title).FontSize(t.FontSize + 8).Bold().Grow(1)
			ui.Segmented(c, &a.pick, "7×6", "10×10", "15×15")
		})
		ev := nonogram.View(c, &a.games[a.pick], puzzles[a.pick].picture)
		ev.Grow(1)
		if ev.Solved {
			a.solved[a.pick] = true
		}
	})
}

func main() {
	a := &app{pick: 1}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Nonogram",
			Width:   640,
			Height:  720,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
