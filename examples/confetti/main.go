// Confetti cheers a release checklist: ticking off the last item showers
// confetti over the window, and Ship it bursts it from the button.
//
//	go run ./examples/confetti
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/confetti"
)

type app struct {
	party confetti.State
	items []string
	done  []bool
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(10)).Gap(t.Space(5)).Children(func() {
		ui.Text(c, "Release 3.2").FontSize(t.FontSize + 8).Bold()
		ui.Column(c).Gap(t.Space(3)).Children(func() {
			for i, item := range a.items {
				before := a.allDone()
				if ui.Checkbox(c.Key(i), &a.done[i], item).Changed() && !before && a.allDone() {
					w, _ := c.Size()
					a.party.Shower(w)
				}
			}
		})
		b := ui.PrimaryButton(c, "Ship it").AlignSelf(ui.Start).Disabled(!a.allDone())
		if b.Clicked() {
			r := b.Bounds()
			a.party.Burst(ui.Point{X: r.X + r.W/2, Y: r.Y + r.H/2})
		}
	})
	confetti.View(c, &a.party)
}

func (a *app) allDone() bool {
	for _, d := range a.done {
		if !d {
			return false
		}
	}
	return true
}

func main() {
	a := &app{items: []string{
		"Changelog written",
		"Migrations run on staging",
		"Release notes reviewed",
		"Builds signed and notarized",
		"Status page updated",
	}}
	a.done = make([]bool, len(a.items))
	a.done[0], a.done[1] = true, true
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Confetti",
			Width:   720,
			Height:  560,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
