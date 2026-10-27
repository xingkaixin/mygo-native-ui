// Stopwatch times laps of a run and counts down a tea timer: start the
// stopwatch and tap Lap, or set the timer from a preset and wait for it
// to ring.
//
//	go run ./examples/stopwatch
package main

import (
	"log"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/stopwatch"
)

type app struct {
	tab   int
	watch stopwatch.State
	timer stopwatch.TimerState
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(6)).Gap(t.Space(6)).AlignItems(ui.Center).Children(func() {
		ui.Segmented(c, &a.tab, "Stopwatch", "Timer").Label("Mode")
		ui.Column(c).Width(t.Space(90)).Grow(1).Children(func() {
			if a.tab == 0 {
				stopwatch.View(c, &a.watch).Grow(1)
				return
			}
			if stopwatch.Timer(c, &a.timer).Finished {
				c.Toast("Your tea is ready")
			}
		})
	})
}

func main() {
	a := &app{timer: stopwatch.TimerState{Duration: 3 * time.Minute}}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Stopwatch",
			Width:   480,
			Height:  640,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
