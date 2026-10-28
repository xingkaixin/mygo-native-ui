// Rebase plans an interactive rebase of a feature branch onto main: choose
// what to do with each commit, drag them into order, and read the todo git
// would run below.
//
//	go run ./examples/rebase
package main

import (
	"log"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/rebase"
)

type app struct {
	editor rebase.State
	steps  []rebase.Step
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Padding(t.Space(8)).Gap(t.Space(4)).Background(t.Background).Children(func() {
		ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "Rebase").FontSize(t.FontSize + 4).Bold()
			ui.Text(c, "feature/search").Font("monospace").FontSize(t.FontSize + 2)
			ui.Text(c, "onto").FontSize(t.FontSize + 2).TextColor(t.TextMuted)
			ui.Text(c, "main").Font("monospace").FontSize(t.FontSize + 2)
		})
		rebase.View(c, &a.editor, &a.steps).Grow(1)
		ui.Column(c).Gap(t.Space(1)).Padding(t.Space(3)).Radius(t.Space(2)).Background(t.Surface).Children(func() {
			ui.Text(c, "git-rebase-todo").FontSize(t.FontSize - 2).TextColor(t.TextMuted)
			ui.Text(c, strings.TrimSpace(rebase.Todo(a.steps))).Font("monospace").FontSize(t.FontSize - 1).Selectable()
		})
	})
}

func main() {
	a := &app{steps: []rebase.Step{
		{Hash: "a1b2c3d", Subject: "Add a search index for notes"},
		{Hash: "4e5f6a7", Subject: "WIP"},
		{Hash: "8b9c0d1", Subject: "Fix typo in search index"},
		{Hash: "2e3f4a5", Subject: "Add a search box to the sidebar"},
		{Hash: "6b7c8d9", Subject: "console.log everything"},
		{Hash: "0e1f2a3", Subject: "Highlight matches in results"},
		{Hash: "4b5c6d7", Subject: "Address review comments"},
	}}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Rebase",
			Width:   760,
			Height:  780,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
