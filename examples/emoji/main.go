// Emoji writes a message with emoji from a picker, as a chat app does:
// the button by the field opens the picker in a popover, and each emoji
// picked goes at the end of the message. Send adds the message above.
//
//	go run ./examples/emoji
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/emoji"
)

// app is the messages sent, the one being written, and the picker.
type app struct {
	sent    []string
	message string
	picking bool
	emoji   emoji.State
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Children(func() {
		ui.Scroll(c).Grow(1).Children(func() {
			ui.Column(c).Padding(16).Gap(8).Children(func() {
				for _, m := range a.sent {
					ui.Text(c, m).Padding(8, 12).Radius(14).Background(t.Accent).TextColor(t.AccentText).AlignSelf(ui.End)
				}
			})
		})
		ui.Divider(c)
		ui.Row(c).Padding(10, 12).Gap(8).AlignItems(ui.Center).Children(func() {
			button := ui.Button(c, "😀").Label("Emoji")
			if button.Clicked() {
				a.picking = !a.picking
			}
			ui.Popover(c, button, &a.picking, func() {
				emoji.Picker(c, &a.emoji).Size(352, 400)
			})
			if e, ok := a.emoji.Picked(); ok {
				a.message += e
			}
			field := ui.TextInput(c, &a.message).Grow(1).Placeholder("Message").Label("Message")
			if (field.Submitted() || ui.Button(c, "Send").Clicked()) && a.message != "" {
				a.sent = append(a.sent, a.message)
				a.message = ""
			}
		})
	})
}

func main() {
	a := &app{
		sent:  []string{"Lunch at noon? 🍜", "Ship it 🚀"},
		emoji: emoji.State{Recent: []string{"👍", "🎉", "❤️", "😂", "🙏"}},
	}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Emoji",
			Width:   520,
			Height:  560,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
