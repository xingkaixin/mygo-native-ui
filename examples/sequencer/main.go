// Sequencer is a drum machine: choose a pattern to start from, click steps
// to change it, and play it at any tempo.
//
//	go run ./examples/sequencer
package main

import (
	"log"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/sequencer"
)

// patterns are written a row a drum, from Kick to Rim, x for a step that
// sounds.
var patterns = []struct {
	name, rows string
	bpm        float64
}{
	{"House", `
x...x...x...x...
....x.......x...
................
..x...x...x...x.
................
................
................
................`, 124},
	{"Breakbeat", `
x.x.......x.....
....x..x.x..x..x
................
x.x.x.x.x.x.x.x.
.............x..
................
................
................`, 132},
	{"Boom bap", `
x......x..x.....
....x.......x...
....x.......x...
x.x.x.x.x.x.x.x.
................
................
................
...x.......x....`, 90},
	{"Fill", `
x...x...x...x...
....x.......x.xx
................
x.x.x.x.........
................
........x..x....
.........xx..x..
................`, 110},
}

func parse(rows string) sequencer.Pattern {
	var p sequencer.Pattern
	for d, row := range strings.Fields(rows) {
		for k, r := range row {
			p[d][k] = r == 'x'
		}
	}
	return p
}

type app struct {
	pattern sequencer.Pattern
	bpm     float64
	drums   sequencer.State
	preset  int
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(6)).Gap(t.Space(5)).Children(func() {
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "Drum machine").Bold().FontSize(t.FontSize + 6).Grow(1)
			names := make([]string, len(patterns))
			for i, p := range patterns {
				names[i] = p.name
			}
			if ui.Segmented(c, &a.preset, names...).Changed() {
				a.pattern, a.bpm = parse(patterns[a.preset].rows), patterns[a.preset].bpm
			}
		})
		sequencer.View(c, &a.drums, &a.pattern, &a.bpm)
	})
}

func main() {
	a := &app{pattern: parse(patterns[0].rows), bpm: patterns[0].bpm}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Sequencer",
			Width:   840,
			Height:  520,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
