// Resistor reads a resistor's color bands: click a band and pick its
// color, switch between four, five and six bands, or type a value such
// as 4k7 to see its bands.
//
//	go run ./examples/resistor
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/resistor"
)

type app struct {
	resistor resistor.State
	bands    []resistor.Color
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(8)).Children(func() {
		resistor.View(c, &a.resistor, &a.bands)
	})
}

func main() {
	a := &app{bands: []resistor.Color{resistor.Brown, resistor.Black, resistor.Black, resistor.Red, resistor.Brown}}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Resistor",
			Width:   760,
			Height:  560,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
