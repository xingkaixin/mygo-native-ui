// Masked shows a form of masked fields: a phone number, a date, a product
// key, a MAC address and a timecode. Beside each field are the value the
// app keeps, the characters of the slots alone, and whether it fills
// them all; under the form, the field completed last, formatted again.
//
//	go run ./examples/masked
package main

import (
	"fmt"
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/masked"
)

// field is a field of the form: its label, its pattern and what its empty
// slots show.
type field struct {
	label, pattern string
	placeholder    rune
}

var fields = []field{
	{"Phone", "(999) 999-9999", '_'},
	{"Date", "9999-99-99", '_'},
	{"Product key", "*****-*****-*****", 0},
	{"MAC address", "hh:hh:hh:hh:hh:hh", 0},
	{"Timecode", "99:99:99:99", '0'},
}

type app struct {
	values [5]string
	last   string
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Padding(24).Gap(20).Background(t.Background).Children(func() {
		ui.Text(c, "Masked fields").FontSize(16).Bold()
		ui.Form(c, func() {
			for i, f := range fields {
				ui.Field(c.Key(i), f.label, func() {
					ui.Row(c).Gap(t.Space(4)).AlignItems(ui.Center).Children(func() {
						ui.Box(c).Width(t.Space(50)).Children(func() {
							if v, ok := masked.Field(c, &a.values[i], f.pattern, f.placeholder).Done(); ok {
								a.last = fmt.Sprintf("%s %s", f.label, masked.Format(f.pattern, v))
							}
						})
						ui.Text(c, fmt.Sprintf("%q", a.values[i])).Font("monospace").Width(t.Space(40))
						if masked.Complete(f.pattern, a.values[i]) {
							ui.Text(c, "Complete").TextColor(t.Success)
						} else {
							ui.Text(c, "Incomplete").TextColor(t.TextMuted)
						}
					})
				})
			}
		})
		last := "Fill a field to see it here."
		if a.last != "" {
			last = "Completed last: " + a.last
		}
		ui.Text(c, last).TextColor(t.TextMuted)
	})
}

func main() {
	a := &app{values: [5]string{"4155550132", "202610", "", "00005e005301", "0100"}}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Masked",
			Width:   640,
			Height:  400,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
