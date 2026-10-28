// Fifths shows the circle of fifths in D major: click a key, or use the
// arrows, to see its scale and chords.
//
//	go run ./examples/fifths
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/fifths"
)

type app struct {
	key fifths.Key
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Padding(t.Space(8)).AlignItems(ui.Center).Justify(ui.Center).Background(t.Background).Children(func() {
		fifths.View(c, &a.key).AutoFocus()
	})
}

func main() {
	a := &app{key: fifths.Key{At: 2}}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Circle of Fifths",
			Width:   760,
			Height:  480,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
