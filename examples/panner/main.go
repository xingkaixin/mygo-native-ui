// Panner places a sound around a listener in 5.1 or 7.1. Drag the puck,
// nudge it with the arrows, double-click to bring it to the middle, or
// turn on Orbit to send it around the room.
//
//	go run ./examples/panner
package main

import (
	"log"
	"math"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/panner"
)

type app struct {
	pan    panner.Position
	layout int
	orbit  bool
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	if a.orbit {
		turn := float64(c.Now().UnixMilli()%6000) / 6000 * 2 * math.Pi
		a.pan = panner.Position{X: 0.85 * math.Sin(turn), Y: 0.85 * math.Cos(turn)}
		c.AnimationFrame()
	}
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(8)).Gap(t.Space(5)).Children(func() {
		ui.Row(c).Gap(t.Space(4)).AlignItems(ui.Center).Children(func() {
			ui.Segmented(c, &a.layout, "5.1", "7.1").Label("Layout")
			ui.Toggle(c, &a.orbit, "Orbit")
		})
		layout := []panner.Layout{panner.Surround51, panner.Surround71}[a.layout]
		if panner.View(c, &a.pan, layout).Changed() {
			a.orbit = false
		}
	})
}

func main() {
	a := &app{pan: panner.Position{X: -0.4, Y: 0.6}}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Surround Panner",
			Width:   760,
			Height:  560,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
