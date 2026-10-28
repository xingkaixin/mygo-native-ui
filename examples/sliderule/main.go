// Sliderule multiplies on a slide rule: C's 1 sits over 2 on D, and the
// hairline over 3 on C reads 6 on D. Drag the slide and the cursor to
// work out more.
//
//	go run ./examples/sliderule
package main

import (
	"log"
	"math"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/sliderule"
)

type app struct {
	rule sliderule.State
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(8)).Children(func() {
		sliderule.View(c, &a.rule)
	})
}

func main() {
	a := &app{rule: sliderule.State{Slide: math.Log10(2), Cursor: math.Log10(6)}}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Slide Rule",
			Width:   1000,
			Height:  400,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
