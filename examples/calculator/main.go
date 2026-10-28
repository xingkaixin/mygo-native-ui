// Calculator adds up a few receipts on a calculator with a paper tape.
//
//	go run ./examples/calculator
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/calculator"
)

type app struct {
	calc calculator.State
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Row(c).Fill().Padding(t.Space(8)).AlignItems(ui.Center).Justify(ui.Center).Background(t.Background).Children(func() {
		calculator.View(c, &a.calc)
	})
}

func main() {
	a := &app{}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Calculator",
			Width:   720,
			Height:  640,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
