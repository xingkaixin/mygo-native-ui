// Automaton runs Conway's Game of Life from Gosper's glider gun and an
// R-pentomino, or four other rules from a random soup.
//
//	go run ./examples/automaton
package main

import (
	"fmt"
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/automaton"
)

var rules = []struct {
	name string
	rule automaton.Rule
}{
	{"Life", automaton.Life},
	{"HighLife", automaton.HighLife},
	{"Seeds", automaton.Seeds},
	{"Day & Night", automaton.DayNight},
	{"Maze", automaton.Maze},
}

type app struct {
	board   *automaton.Board
	life    automaton.State
	rule    int
	playing bool
	rate    float64
}

func (a *app) start() {
	a.board.Clear()
	a.board.Place(automaton.Gosper, 4, 4)
	a.board.Place(automaton.RPento, 80, 50)
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	rate := 0.0
	if a.playing {
		rate = a.rate
	}
	ui.Column(c).Fill().Background(t.Background).Children(func() {
		ui.Row(c).Gap(t.Space(2)).Padding(t.Space(3)).AlignItems(ui.Center).Children(func() {
			label := "Play"
			if a.playing {
				label = "Pause"
			}
			if ui.PrimaryButton(c, label).Clicked() {
				a.playing = !a.playing
			}
			if ui.Button(c, "Step").Clicked() {
				a.board.Step(rules[a.rule].rule)
			}
			names := make([]string, len(rules))
			for i, r := range rules {
				names[i] = r.name
			}
			prev := a.rule
			ui.Segmented(c, &a.rule, names...)
			if a.rule != prev {
				if a.rule == 0 {
					a.start()
				} else {
					a.board.Scatter(0.3)
				}
			}
			ui.Box(c).Grow(1)
			if ui.Button(c, "Random").Clicked() {
				a.board.Scatter(0.3)
			}
			if ui.Button(c, "Clear").Clicked() {
				a.board.Clear()
			}
		})
		automaton.View(c, &a.life, a.board, rules[a.rule].rule, rate).Grow(1).MarginX(t.Space(3))
		ui.Row(c).Gap(t.Space(4)).Padding(t.Space(3)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, fmt.Sprintf("Generation %d", a.board.Generation())).FontFeatures("tnum").Width(t.Space(32))
			ui.Text(c, fmt.Sprintf("Population %d", a.board.Population())).FontFeatures("tnum").Width(t.Space(32))
			ui.Text(c, rules[a.rule].rule.String()).TextColor(t.TextMuted).Font("monospace")
			ui.Box(c).Grow(1)
			ui.Text(c, fmt.Sprintf("%.0f/s", a.rate)).TextColor(t.TextMuted).FontFeatures("tnum")
			ui.Slider(c, &a.rate, 1, 60).Width(t.Space(40))
		})
	})
}

func main() {
	a := &app{board: automaton.NewBoard(120, 80), playing: true, rate: 15}
	a.start()
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Automaton",
			Width:   900,
			Height:  720,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
