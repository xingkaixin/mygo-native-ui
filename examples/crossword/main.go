// Crossword plays a five by five mini crossword: type the answers, find
// your way with the arrows and Tab, and Check the letters you are unsure
// of. The clock stops when the last letter is right.
//
//	go run ./examples/crossword
package main

import (
	"fmt"
	"log"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/crossword"
)

var mini = crossword.Puzzle{
	Grid: []string{
		"ARM##",
		"ROOF#",
		"TOTAL",
		"#MODE",
		"##RED",
	},
	Across: map[int]string{
		1: "Limb with an elbow",
		4: "It's over your head at home",
		6: "Sum at the bottom of a bill",
		8: "Dark ___ (a setting for night owls)",
		9: "Color of a stop sign",
	},
	Down: map[int]string{
		1: "What a gallery hangs",
		2: "Kitchen or den",
		3: "What spins a fan",
		5: "Slowly go quiet, as a song's end",
		7: "Tiny light on a power button, for short",
	},
}

type app struct {
	play    crossword.State
	started time.Time
	took    time.Duration
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	now := c.Now()
	if a.started.IsZero() {
		a.started = now
	}
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(8)).Gap(t.Space(6)).Children(func() {
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "The Mini").FontSize(t.FontSize + 8).Bold().Grow(1)
			elapsed := a.took
			if elapsed == 0 {
				elapsed = now.Sub(a.started)
				c.After(time.Second - elapsed%time.Second)
			}
			ui.Text(c, fmt.Sprintf("%d:%02d", int(elapsed.Minutes()), int(elapsed.Seconds())%60)).FontFeatures("tnum").TextColor(t.TextMuted)
			if ui.Button(c, "Check").Clicked() {
				a.play.Check()
			}
			if ui.Button(c, "Clear").Clicked() {
				a.play, a.started, a.took = crossword.State{}, now, 0
			}
		})
		if crossword.View(c, &a.play, mini).Solved {
			a.took = now.Sub(a.started)
			c.Toast(fmt.Sprintf("Solved in %d:%02d", int(a.took.Minutes()), int(a.took.Seconds())%60))
		}
	})
}

func main() {
	a := &app{}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Crossword",
			Width:   860,
			Height:  640,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
