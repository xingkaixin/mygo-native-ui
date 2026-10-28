// Address asks for a shipping address at checkout: switch the country to
// watch the fields reorder and relabel, leave the postal code to see it
// tidied, and watch the envelope beside the form follow. Continue shows
// what is still missing.
//
//	go run ./examples/address
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/address"
)

type app struct {
	form    address.State
	address address.Address
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Row(c).Fill().Background(t.Background).Padding(t.Space(10)).Gap(t.Space(10)).AlignItems(ui.Start).Children(func() {
		ui.Column(c).Width(t.Space(110)).Gap(t.Space(5)).Children(func() {
			ui.Text(c, "Shipping address").FontSize(t.FontSize + 8).Bold()
			address.View(c, &a.form, &a.address)
			if ui.PrimaryButton(c, "Continue").Clicked() && a.form.Check(a.address) {
				c.Toast("Shipping to " + a.address.City)
			}
		})
		ui.Column(c).Grow(1).Gap(t.Space(3)).Children(func() {
			ui.Text(c, "Ship to").Bold().TextColor(t.TextMuted)
			ui.Column(c).Padding(t.Space(6)).Gap(t.Space(1)).Radius(t.Radius).Background(t.Surface).Border(1, t.Border).Children(func() {
				for i, l := range a.address.Lines() {
					ui.Text(c.Key(i), l).FontFeatures("tnum")
				}
			})
		})
	})
}

func main() {
	a := &app{address: address.Address{Country: "US", Line1: "1 Market Street", City: "San Francisco", Region: "California", Postal: "94105"}}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Checkout",
			Width:   900,
			Height:  640,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
