// Outline plans a launch in an outliner, as Workflowy does, the outline
// beside it as Markdown, as the app would save it. Enter adds an item,
// Tab and Shift+Tab nest it and take it out, Cmd+Up and Cmd+Down move
// it, and a bullet folds what is under it.
//
//	go run ./examples/outline
package main

import (
	"log"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/outline"
)

type app struct {
	items    []outline.Item
	outliner outline.State
	split    float32
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Split(c, &a.split, func() {
		ui.Scroll(c).Fill().Background(t.Background).Children(func() {
			ui.Column(c).Gap(t.Space(3)).Padding(t.Space(6), t.Space(8)).Children(func() {
				ui.Text(c, "Launch plan").Bold().FontSize(t.FontSize + 8)
				outline.View(c, &a.outliner, &a.items)
			})
		})
	}, func() {
		ui.Scroll(c).Fill().Background(t.Surface).Children(func() {
			ui.Column(c).Gap(t.Space(2)).Padding(t.Space(5)).Children(func() {
				ui.Text(c, "Markdown").Bold()
				ui.Text(c, markdown(a.items)).Font("monospace").FontSize(t.FontSize - 1).Selectable()
			})
		})
	}).Fill()
}

// markdown returns items as a Markdown list, two spaces a level.
func markdown(items []outline.Item) string {
	var b strings.Builder
	for _, it := range items {
		b.WriteString(strings.Repeat("  ", it.Depth) + "- " + it.Text + "\n")
	}
	return b.String()
}

func main() {
	a := &app{split: 640}
	for _, line := range strings.Split(strings.TrimSpace(`
Ship the beta
	Fix the sync conflicts
		Merge edits made offline
		Keep the newer title
	Write the release notes
	Record a two-minute demo
Tell people
	Post on the blog
	Email the waitlist
		Draft
		Send in two batches
Afterwards
	Read the feedback
	Plan the next release`), "\n") {
		text := strings.TrimLeft(line, "\t")
		a.items = append(a.items, outline.Item{Text: text, Depth: len(line) - len(text)})
	}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Outline",
			Width:   1000,
			Height:  640,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
