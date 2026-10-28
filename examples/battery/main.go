// Battery shows a day of a laptop's battery: a night on the charger, a
// day running down to red, a charge at lunch, and how long the screen
// was on each hour. Point at either chart to read a time.
//
//	go run ./examples/battery
package main

import (
	"log"
	"math"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/battery"
)

type app struct {
	samples []battery.Sample
	screen  []battery.Usage
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(8)).Children(func() {
		battery.View(c, a.samples, a.screen)
	})
}

// day makes a day ending now: charging overnight, draining through the
// morning, charging over lunch, and draining again.
func day(now time.Time) ([]battery.Sample, []battery.Usage) {
	var samples []battery.Sample
	var screen []battery.Usage
	start := now.Truncate(time.Hour).Add(-23 * time.Hour)
	level := 0.35
	for at := start; !at.After(now); at = at.Add(5 * time.Minute) {
		h := at.Hour()
		charging := h < 7 || h >= 12 && h < 14
		switch {
		case charging:
			level = math.Min(1, level+0.025)
		case h >= 8 && h < 12:
			level -= 0.017
		case h >= 14 && h < 20:
			level -= 0.008
		default:
			level -= 0.001
		}
		level = math.Max(0.05, level)
		samples = append(samples, battery.Sample{At: at, Level: level, Charging: charging})
	}
	for at := start; !at.After(now); at = at.Add(time.Hour) {
		on := 0.0
		if h := at.Hour(); h >= 8 && h < 23 {
			on = 0.45 + 0.5*math.Abs(math.Sin(float64(h)))
		}
		screen = append(screen, battery.Usage{At: at, Screen: time.Duration(on * float64(time.Hour))})
	}
	return samples, screen
}

func main() {
	a := &app{}
	a.samples, a.screen = day(time.Date(2026, 10, 28, 18, 40, 0, 0, time.Local))
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Battery",
			Width:   760,
			Height:  520,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
