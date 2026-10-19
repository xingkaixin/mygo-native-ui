// Kanban tracks issues on a board, as Linear does: drag them between
// lanes, or choose one and move it with Cmd and the arrows. A field adds
// issues to the backlog, and Backspace deletes the one chosen.
//
//	go run ./examples/kanban
package main

import (
	"fmt"
	"log"
	"slices"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/kanban"
)

// issue is a card of the board.
type issue struct {
	id     int
	title  string
	label  string
	points int
}

// app is the state the window shows: the board and the issue typed.
type app struct {
	lanes []kanban.Lane[issue]
	board kanban.State
	title string
	next  int
	moves int
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Children(func() {
		ui.Row(c).Padding(8, 12).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "Issues").Bold().Grow(1)
			if ui.TextInput(c, &a.title).Width(260).Placeholder("New issue").Label("New issue").Submitted() && a.title != "" {
				a.lanes[0].Cards = append(a.lanes[0].Cards, issue{id: a.next, title: a.title, label: "Feature", points: 1})
				a.board.Choose(0, len(a.lanes[0].Cards)-1)
				a.title, a.next = "", a.next+1
			}
		})
		ui.Divider(c)
		board := kanban.View(c, &a.board, a.lanes, func(i issue) { card(c, i) }).Grow(1)
		if board.Changed() {
			a.moves++
		}
		l, n := a.board.Chosen()
		if board.Shortcut(0, ui.KeyBackspace) && n < len(a.lanes[l].Cards) {
			a.lanes[l].Cards = slices.Delete(a.lanes[l].Cards, n, n+1)
		}
		ui.Divider(c)
		status := fmt.Sprintf("Moves: %d", a.moves)
		if n < len(a.lanes[l].Cards) {
			status = fmt.Sprintf("ENG-%d in %s · %s", a.lanes[l].Cards[n].id, a.lanes[l].Title, status)
		}
		ui.Text(c, status).FontSize(12).TextColor(t.TextMuted).Padding(6, 12)
	})
}

// card shows an issue: its id, title, label and estimate.
func card(c *ui.Context, i issue) {
	t := c.Theme()
	ui.Text(c, fmt.Sprintf("ENG-%d", i.id)).FontSize(11).TextColor(t.TextMuted)
	ui.Text(c, i.title)
	ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
		ui.Badge(c, i.label)
		ui.Spacer(c)
		ui.Text(c, fmt.Sprintf("%d pt", i.points)).FontSize(11).TextColor(t.TextMuted).FontFeatures("tnum")
	})
}

func main() {
	a := &app{next: 113}
	add := func(title, label string, points int) issue {
		a.next++
		return issue{id: a.next - 1, title: title, label: label, points: points}
	}
	a.lanes = []kanban.Lane[issue]{
		{Title: "Backlog", Cards: []issue{
			add("Export boards as CSV", "Feature", 3),
			add("Keyboard shortcut sheet", "Feature", 2),
			add("Dark mode for emails", "Design", 2),
		}},
		{Title: "Todo", Cards: []issue{
			add("Crash on empty workspace", "Bug", 1),
			add("Rate limit the search API", "Infra", 5),
		}},
		{Title: "In Progress", Cards: []issue{
			add("Sync cards across windows", "Feature", 8),
		}},
		{Title: "Done", Cards: []issue{
			add("Retina tray icon", "Design", 1),
			add("Upgrade to Go 1.27", "Infra", 2),
		}},
	}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Kanban",
			Width:   1200,
			Height:  640,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
