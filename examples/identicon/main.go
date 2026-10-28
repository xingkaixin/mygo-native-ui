// Identicon draws the avatars of a team from their names, one large for
// a name you type, and a wall of them for made-up accounts.
//
//	go run ./examples/identicon
package main

import (
	"fmt"
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/identicon"
)

var team = []string{"ada", "grace", "linus", "margaret", "dennis", "barbara", "ken", "frances"}

type app struct {
	name string
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Scroll(c).Fill().Background(t.Background).Children(func() {
		ui.Column(c).Padding(t.Space(6)).Gap(t.Space(6)).Children(func() {
			ui.Row(c).Gap(t.Space(5)).AlignItems(ui.Center).Children(func() {
				identicon.View(c, a.name, 120)
				ui.Column(c).Gap(t.Space(2)).Grow(1).Children(func() {
					ui.Text(c, "Name").TextColor(t.TextMuted)
					ui.TextInput(c, &a.name).Placeholder("Type a name")
				})
			})
			ui.Column(c).Gap(t.Space(2)).Children(func() {
				for _, n := range team {
					ui.Row(c.Key(n)).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
						identicon.View(c, n, 32)
						ui.Text(c, n).FontWeight(500)
					})
				}
			})
			ui.Row(c).Wrap().Gap(t.Space(2)).Children(func() {
				for i := range 60 {
					identicon.View(c.Key(i), fmt.Sprintf("user-%d", 1000+i), 44)
				}
			})
		})
	})
}

func main() {
	a := &app{name: "octocat"}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Identicon",
			Width:   560,
			Height:  720,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
