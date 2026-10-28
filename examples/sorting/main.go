// Sorting races four algorithms on the same shuffled values, or shows any
// one of the seven alone.
//
//	go run ./examples/sorting
package main

import (
	"fmt"
	"log"
	"math/rand/v2"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/sorting"
)

var race = [4]sorting.Algorithm{sorting.Insertion, sorting.Shell, sorting.Merge, sorting.Quick}

type app struct {
	values  []int
	states  [4]sorting.State
	one     sorting.State
	algo    string
	mode    int
	playing bool
	rate    float64
}

func (a *app) shuffle(n int) {
	a.values = rand.Perm(n)
	for i := range a.values {
		a.values[i]++
	}
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	rate := 0.0
	if a.playing {
		rate = a.rate
	}
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(3)).Gap(t.Space(3)).Children(func() {
		ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
			label := "Play"
			if a.playing {
				label = "Pause"
			}
			if ui.PrimaryButton(c, label).Clicked() {
				a.playing = !a.playing
			}
			if ui.Button(c, "Shuffle").Clicked() {
				a.shuffle(len(a.values))
			}
			ui.Segmented(c, &a.mode, "Race", "One")
			if a.mode == 1 {
				names := make([]string, len(sorting.Algorithms))
				for i, al := range sorting.Algorithms {
					names[i] = al.String()
				}
				ui.Select(c, &a.algo, names)
			}
			ui.Box(c).Grow(1)
			ui.Text(c, fmt.Sprintf("%.0f steps/s", a.rate)).TextColor(t.TextMuted).FontFeatures("tnum")
			ui.Slider(c, &a.rate, 10, 2000).Width(t.Space(40))
		})
		if a.mode == 1 {
			algo := sorting.Quick
			for _, al := range sorting.Algorithms {
				if al.String() == a.algo {
					algo = al
				}
			}
			sorting.View(c, &a.one, a.values, algo, rate).Grow(1)
			return
		}
		ui.Grid(c).Columns(2).Gap(t.Space(4)).Grow(1).Children(func() {
			for i, al := range race {
				sorting.View(c.Key(i), &a.states[i], a.values, al, rate)
			}
		})
	})
}

func main() {
	a := &app{playing: true, rate: 400, algo: sorting.Heap.String()}
	a.shuffle(80)
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Sorting",
			Width:   900,
			Height:  640,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
