// Hold guards the danger zone of an app's settings: hold the button to
// erase the library, or slide the knob to delete the account. Letting go
// early does nothing.
//
//	go run ./examples/hold
package main

import (
	"fmt"
	"log"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/hold"
)

type app struct {
	erase, remove hold.State
	photos        int
	deleted       bool
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(10)).AlignItems(ui.Center).Children(func() {
		ui.Column(c).Width(t.Space(110)).Gap(t.Space(6)).Children(func() {
			ui.Text(c, "Danger zone").FontSize(t.FontSize + 8).Bold()
			ui.Column(c).Gap(t.Space(3)).Children(func() {
				ui.Text(c, "Erase library").Bold()
				ui.Text(c, fmt.Sprintf("Removes all %d photos from this Mac. Hold the button until it fills.", a.photos)).TextColor(t.TextMuted)
				ui.Row(c).Children(func() {
					if hold.Button(c, &a.erase, "Hold to erase", 1500*time.Millisecond).Confirmed {
						a.photos = 0
						c.Toast("Erased the library")
					}
				})
			})
			ui.Divider(c)
			ui.Column(c).Gap(t.Space(3)).Children(func() {
				ui.Text(c, "Delete account").Bold()
				status := "Your account and its backups go for good. Slide to the end to confirm."
				if a.deleted {
					status = "Account deleted."
				}
				ui.Text(c, status).TextColor(t.TextMuted)
				ev := hold.Slider(c, &a.remove, "Slide to delete account")
				ev.Width(t.Space(80))
				if ev.Confirmed {
					a.deleted = true
				}
			})
			if ui.Button(c, "Restore demo").Clicked() {
				a.photos, a.deleted = 12480, false
			}
		})
	})
}

func main() {
	a := &app{photos: 12480}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Settings",
			Width:   640,
			Height:  560,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
