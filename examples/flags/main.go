// Flags switches the feature flags of a notes app across development,
// staging and production, rolling the new ones out a share of users at a
// time.
//
//	go run ./examples/flags
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/flags"
)

type app struct {
	list  flags.State
	flags []flags.Flag
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Scroll(c).Fill().Background(t.Background).Children(func() {
		ui.Column(c).Padding(t.Space(8)).Gap(t.Space(4)).Children(func() {
			ui.Text(c, "Feature flags").FontSize(t.FontSize + 4).Bold()
			flags.View(c, &a.list, []string{"Development", "Staging", "Production"}, &a.flags)
		})
	})
}

func env(on bool, percent int) flags.Env { return flags.Env{On: on, Percent: percent} }

func flag(key, description string, envs ...flags.Env) flags.Flag {
	return flags.Flag{Key: key, Description: description, Envs: envs}
}

func main() {
	a := &app{flags: []flags.Flag{
		flag("new-editor", "The block editor in place of the plain one",
			env(true, 100), env(true, 100), env(true, 25)),
		flag("offline-sync", "Keep notes on the device and sync when back online",
			env(true, 100), env(true, 50), env(false, 0)),
		flag("ai-summaries", "Summarize long notes at the top",
			env(true, 100), env(false, 100), env(false, 0)),
		flag("shared-folders", "Folders more than one person can edit",
			env(true, 100), env(true, 100), env(true, 100)),
		flag("dark-icons", "The app icon follows the system appearance",
			env(true, 100), env(true, 100), env(true, 100)),
		flag("legacy-export", "Export to the old .notes format",
			env(false, 100), env(false, 100), env(true, 10)),
		flag("checkout-v2", "The new upgrade page with yearly plans",
			env(true, 100), env(true, 100), env(false, 5)),
	}}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Flags",
			Width:   900,
			Height:  720,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
