// Typing times the typing of a passage chosen from three, and keeps the
// best speed of the session. Start typing; Escape starts over.
//
//	go run ./examples/typing
package main

import (
	"log"
	"strconv"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/typing"
)

var passages = []string{
	"The lighthouse keeper climbed the stairs each evening with a can of oil and a cloth. Ships he would never meet passed in the dark, and every one of them knew where the rocks were because of him.",
	"A good map leaves things out. It keeps the roads you need and drops the rest, so that you can fold the whole valley into a pocket and still find the bridge before the rain sets in.",
	"Bread asks for patience more than skill: flour, water, salt and time. Most of the work happens while you sleep, and the rest is knowing when to leave the dough alone.",
}

type app struct {
	test   typing.State
	chosen int
	best   float64
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(10)).Gap(t.Space(8)).Children(func() {
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			ui.Segmented(c, &a.chosen, "Lighthouse", "Maps", "Bread").Label("Passage")
			ui.Spacer(c)
			if a.best > 0 {
				ui.Text(c, "Best "+strconv.Itoa(int(a.best))+" wpm").TextColor(t.TextMuted)
			}
		})
		ev := typing.View(c, &a.test, passages[a.chosen])
		ev.MaxWidth(t.Space(200))
		if ev.Finished {
			a.best = max(a.best, ev.WPM)
		}
	})
}

func main() {
	a := &app{}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Typing",
			Width:   960,
			Height:  560,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
