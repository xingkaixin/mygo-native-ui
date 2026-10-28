// Typescale sets the type scale of a design system on a base of 16
// pixels and a major third.
//
//	go run ./examples/typescale
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/typescale"
)

type app struct {
	preview typescale.State
	scale   typescale.Scale
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Scroll(c).Fill().Background(t.Background).Children(func() {
		ui.Column(c).FillWidth().Padding(t.Space(8)).Children(func() {
			typescale.View(c, &a.preview, &a.scale)
		})
	})
}

func main() {
	a := &app{scale: typescale.Scale{Base: 16, Ratio: 1.25, Below: 2, Above: 6}}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Type scale",
			Width:   960,
			Height:  860,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
