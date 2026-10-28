// Patternlock sets an unlock pattern, drawn twice to confirm it, and then
// asks for it to unlock.
//
//	go run ./examples/patternlock
package main

import (
	"log"
	"slices"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/patternlock"
)

type app struct {
	lock    patternlock.State
	pattern []int
	first   []int
	note    string
	open    bool
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Gap(t.Space(6)).AlignItems(ui.Center).Justify(ui.Center).Background(t.Background).Children(func() {
		title := "Draw your pattern to unlock"
		switch {
		case a.open:
			title = "Unlocked"
		case a.pattern == nil && a.first == nil:
			title = "Draw an unlock pattern"
		case a.pattern == nil:
			title = "Draw it again to confirm"
		}
		ui.Text(c, title).FontSize(t.FontSize + 6).Bold()
		ui.Text(c, a.note).TextColor(t.TextMuted).Height(t.Space(5))
		ev := patternlock.View(c, &a.lock)
		if drawn := ev.Drawn; drawn != nil {
			a.note = ""
			switch {
			case len(drawn) < 4:
				a.note = "Join at least 4 dots"
				a.lock.Wrong(c)
			case a.pattern == nil && a.first == nil:
				a.first = drawn
			case a.pattern == nil && slices.Equal(drawn, a.first):
				a.pattern, a.note = drawn, "Pattern set"
			case a.pattern == nil:
				a.first, a.note = nil, "Those differ. Start again"
				a.lock.Wrong(c)
			case slices.Equal(drawn, a.pattern):
				a.open = true
			default:
				a.note = "Wrong pattern"
				a.lock.Wrong(c)
			}
		}
		if ui.Button(c, "Lock").Disabled(!a.open).Clicked() {
			a.open, a.lock = false, patternlock.State{}
		}
	})
}

func main() {
	a := &app{}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Pattern lock",
			Width:   480,
			Height:  600,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
