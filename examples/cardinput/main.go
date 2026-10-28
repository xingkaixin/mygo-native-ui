// Cardinput takes a card at the checkout of a yearly plan. Try the test
// number 4242 4242 4242 4242, or 3782 822463 10005 for a card whose code
// has four digits.
//
//	go run ./examples/cardinput
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/cardinput"
)

type app struct {
	field cardinput.State
	card  cardinput.Card
	paid  bool
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Scroll(c).Fill().Background(t.Background).Children(func() {
		ui.Column(c).Padding(t.Space(8)).Gap(t.Space(6)).AlignItems(ui.Center).Children(func() {
			ui.Column(c).Gap(t.Space(1)).AlignItems(ui.Center).Children(func() {
				ui.Text(c, "Pro, yearly").FontSize(t.FontSize + 4).Bold()
				ui.Text(c, "$96.00 a year").TextColor(t.TextMuted)
			})
			cardinput.View(c, &a.field, &a.card)
			label := "Pay $96.00"
			if a.paid {
				label = "Paid"
			}
			if ui.PrimaryButton(c, label).Width(t.Space(92)).Disabled(a.paid || !a.card.Valid(c.Now())).Clicked() {
				a.paid = true
			}
		})
	})
}

func main() {
	a := &app{}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Card",
			Width:   560,
			Height:  760,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
