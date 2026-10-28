// Braille shows a line in braille, cell by cell, and a writer to type
// more as a Perkins brailler does: click it, then hold F D S J K L
// together and let go.
//
//	go run ./examples/braille
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/braille"
)

type app struct {
	braille braille.State
	text    string
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(8)).Children(func() {
		braille.View(c, &a.braille, &a.text)
	})
}

func main() {
	a := &app{text: "Louis Braille, born 1809."}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Braille",
			Width:   900,
			Height:  560,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
