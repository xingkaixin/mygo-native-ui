// Tangram sets five silhouettes to fill with the seven tans, timing each.
//
//	go run ./examples/tangram
package main

import (
	"fmt"
	"log"
	"math"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/tangram"
)

// side is the side of the square the tans are cut from.
var side = 2 * math.Sqrt2

var figures = []struct {
	name    string
	outline [][2]float64
}{
	{"Square", [][2]float64{{0, 0}, {side, 0}, {side, side}, {0, side}}},
	{"Triangle", [][2]float64{{0, 0}, {4, 4}, {0, 4}}},
	{"Arrow", [][2]float64{{0, 1}, {2, 1}, {2, 0}, {4, 2}, {2, 4}, {2, 3}, {0, 3}}},
	{"Boat", [][2]float64{{3, 1}, {5, 3}, {4, 4}, {1, 4}, {0, 3}, {1, 3}}},
	{"Rocket", [][2]float64{{1, 0}, {2, 1}, {2, 3}, {3, 4}, {-1, 4}, {0, 3}, {0, 1}}},
}

type app struct {
	figure   int
	shown    int
	board    tangram.State
	puzzle   tangram.Puzzle
	started  time.Time
	solvedIn time.Duration
}

func (a *app) start(c *ui.Context) {
	a.shown, a.board = a.figure, tangram.State{}
	a.puzzle = tangram.New(figures[a.figure].outline)
	a.started, a.solvedIn = c.Now(), 0
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	if a.figure != a.shown || a.started.IsZero() {
		a.start(c)
	}
	names := make([]string, len(figures))
	for i, f := range figures {
		names[i] = f.name
	}
	ui.Column(c).Fill().Padding(t.Space(8)).Gap(t.Space(5)).AlignItems(ui.Center).Justify(ui.Center).Background(t.Background).Children(func() {
		ui.Segmented(c, &a.figure, names...)
		tangram.View(c, &a.board, &a.puzzle)
		if a.puzzle.Solved() && a.solvedIn == 0 {
			a.solvedIn = c.Now().Sub(a.started)
		} else if !a.puzzle.Solved() {
			a.solvedIn = 0
		}
		ui.Row(c).Gap(t.Space(4)).AlignItems(ui.Center).Children(func() {
			status := "Click to turn, shift-click to turn back, right-click to turn over"
			if a.solvedIn > 0 {
				d := a.solvedIn.Round(time.Second)
				status = fmt.Sprintf("Solved in %d:%02d", int(d.Minutes()), int(d.Seconds())%60)
			}
			ui.Text(c, status).TextColor(t.TextMuted).FontFeatures("tnum")
			if ui.Button(c, "Start over").Clicked() {
				a.start(c)
			}
		})
	})
}

func main() {
	a := &app{figure: 3}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Tangram",
			Width:   760,
			Height:  560,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
