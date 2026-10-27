// Phoneinput asks for two phone numbers in a sign-up form: choose the
// country from the flag, type the digits and watch them group, or paste
// a number starting with + to switch country. Send code is enabled once
// the number is complete.
//
//	go run ./examples/phoneinput
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/phoneinput"
)

type app struct {
	name             string
	phone, emergency phoneinput.Number
	phoneField       phoneinput.State
	emergencyField   phoneinput.State
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(10)).AlignItems(ui.Center).Children(func() {
		ui.Column(c).Width(t.Space(100)).Gap(t.Space(5)).Children(func() {
			ui.Text(c, "Create your account").FontSize(t.FontSize + 8).Bold()
			ui.Field(c, "Name", func() {
				ui.TextInput(c, &a.name).Placeholder("Your name")
			})
			ui.Field(c, "Mobile number", func() {
				phoneinput.View(c, &a.phoneField, &a.phone)
			})
			ui.Field(c, "Emergency contact", func() {
				phoneinput.View(c, &a.emergencyField, &a.emergency)
			})
			ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
				if ui.PrimaryButton(c, "Send code").Disabled(!a.phone.Valid()).Clicked() {
					c.Toast("Sent a code to " + a.phone.String())
				}
				if a.phone.Valid() {
					ui.Text(c, a.phone.E164()).TextColor(t.TextMuted).FontFeatures("tnum")
				}
			})
		})
	})
}

func main() {
	a := &app{
		phone:     phoneinput.Number{Country: "US"},
		emergency: phoneinput.Number{Country: "GB", National: "2079460958"},
	}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Sign up",
			Width:   640,
			Height:  560,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
