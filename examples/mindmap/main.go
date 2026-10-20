// Mindmap edits an outline on the left and shows it as a mind map on the
// right, as outliners with a map view do. Type in the outline to change
// the map; fold branches and choose items on the map.
//
//	go run ./examples/mindmap
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/mindmap"
	"github.com/xingkaixin/mygo-native-ui/ui/outline"
)

type app struct {
	title   string
	items   []outline.Item
	outline outline.State
	mindmap mindmap.State
	split   float32
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Split(c, &a.split, func() {
		ui.Column(c).Fill().Padding(t.Space(4)).Gap(t.Space(3)).Children(func() {
			ui.TextInput(c, &a.title).Label("Title")
			ui.Scroll(c).Grow(1).Children(func() {
				outline.View(c, &a.outline, &a.items)
			})
		})
	}, func() {
		items := make([]mindmap.Item, len(a.items))
		for i, it := range a.items {
			items[i] = mindmap.Item(it)
		}
		ui.Column(c).Fill().Background(t.Surface).Children(func() {
			sc := ui.ScrollBoth(c).Grow(1)
			view := sc.LayoutBounds()
			sc.Children(func() {
				ui.Column(c).MinWidth(view.W).MinHeight(view.H).Justify(ui.Center).AlignItems(ui.Center).Children(func() {
					mindmap.View(c, &a.mindmap, a.title, items)
				})
			})
			ui.Divider(c)
			ui.Row(c).Padding(t.Space(2), t.Space(4)).Children(func() {
				text := "Choose an item"
				if i, ok := a.mindmap.Chosen(); ok && i < len(items) {
					text = items[i].Text
				}
				ui.Text(c, text).TextColor(t.TextMuted)
			})
		})
	}).Fill()
}

func newApp() *app {
	return &app{title: "Product launch", split: 320, items: []outline.Item{
		{Text: "Research", Depth: 0},
		{Text: "Interviews", Depth: 1},
		{Text: "Survey", Depth: 1},
		{Text: "Competitors", Depth: 1},
		{Text: "Design", Depth: 0},
		{Text: "Flows", Depth: 1},
		{Text: "Prototype", Depth: 1},
		{Text: "Usability test", Depth: 2},
		{Text: "Build", Depth: 0},
		{Text: "API", Depth: 1},
		{Text: "Web app", Depth: 1},
		{Text: "Mobile app", Depth: 1},
		{Text: "iOS", Depth: 2},
		{Text: "Android", Depth: 2},
		{Text: "Marketing", Depth: 0},
		{Text: "Landing page", Depth: 1},
		{Text: "Press kit", Depth: 1},
		{Text: "Launch", Depth: 0},
		{Text: "Beta", Depth: 1},
		{Text: "Release", Depth: 1},
	}}
}

func main() {
	a := newApp()
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Mind map",
			Width:   1280,
			Height:  760,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
