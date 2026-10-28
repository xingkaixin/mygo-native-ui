// Orrery shows the planets on today's date, which plays forward a week a
// second or more, or goes back to today.
//
//	go run ./examples/orrery
package main

import (
	"log"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/orrery"
)

// speeds are how far the date moves for each second of playing.
var speeds = []time.Duration{7 * 24 * time.Hour, 30 * 24 * time.Hour, 365 * 24 * time.Hour}

type app struct {
	date    time.Time
	orrery  orrery.State
	playing bool
	speed   int
	last    time.Time
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	now := c.Now()
	if a.playing {
		if !a.last.IsZero() {
			a.date = a.date.Add(time.Duration(float64(speeds[a.speed]) * now.Sub(a.last).Seconds()))
		}
		a.last = now
		c.After(time.Second / 60)
	}
	ui.Column(c).Fill().Background(t.Background).Children(func() {
		ui.Row(c).Gap(t.Space(2)).Padding(t.Space(3)).AlignItems(ui.Center).Children(func() {
			label := "Play"
			if a.playing {
				label = "Pause"
			}
			if ui.PrimaryButton(c, label).Clicked() {
				a.playing, a.last = !a.playing, time.Time{}
			}
			ui.Segmented(c, &a.speed, "Week/s", "Month/s", "Year/s")
			ui.Box(c).Grow(1)
			if ui.Button(c, "Today").Clicked() {
				a.date, a.playing = now, false
			}
		})
		orrery.View(c, &a.orrery, &a.date).Grow(1)
	})
}

func main() {
	a := &app{date: time.Now()}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Orrery",
			Width:   760,
			Height:  760,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
