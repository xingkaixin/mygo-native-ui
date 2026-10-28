// Querybar filters the issues of a project: is:, author:, label: and
// assignee: narrow them, a minus excludes, and the other words search the
// titles. Type a key, or a colon after one, to see what it takes.
//
//	go run ./examples/querybar
package main

import (
	"fmt"
	"log"
	"slices"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/querybar"
)

type issue struct {
	n        int
	title    string
	open     bool
	author   string
	assignee string
	labels   []string
}

var issues = []issue{
	{412, "Window flickers when resized on Wayland", true, "mira", "jon", []string{"bug", "linux"}},
	{409, "Add a dark variant of the toolbar icons", true, "sam", "", []string{"enhancement", "design"}},
	{405, "Typo in the getting started guide", true, "lee", "", []string{"docs", "good first issue"}},
	{401, "Crash when the clipboard holds an image", false, "jon", "mira", []string{"bug"}},
	{398, "Support right to left text in tables", true, "ana", "ana", []string{"enhancement"}},
	{396, "Tooltip stays after the window loses focus", true, "sam", "jon", []string{"bug"}},
	{390, "Document how to sign builds on Windows", false, "mira", "lee", []string{"docs"}},
	{388, "Scroll position resets after a hot reload", true, "lee", "", []string{"bug", "good first issue"}},
	{381, "Faster text shaping for long lines", false, "ana", "ana", []string{"performance"}},
	{377, "Menu shortcuts missing in the Linux build", true, "jon", "", []string{"bug", "linux"}},
	{372, "Explain the frame budget in the guide", true, "sam", "lee", []string{"docs"}},
}

var fields = []querybar.Field{
	{Key: "is", Hint: "Open or closed", Values: []string{"open", "closed"}},
	{Key: "author", Hint: "Who opened it", Values: []string{"ana", "jon", "lee", "mira", "sam"}},
	{Key: "label", Hint: "Has the label", Values: []string{"bug", "design", "docs", "enhancement", "good first issue", "linux", "performance"}},
	{Key: "assignee", Hint: "Who works on it", Values: []string{"ana", "jon", "lee", "mira", "none"}},
}

func (i issue) has(t querybar.Term) bool {
	v := strings.ToLower(t.Value)
	switch strings.ToLower(t.Key) {
	case "is":
		return (v == "open") == i.open
	case "author":
		return i.author == v
	case "label":
		return slices.Contains(i.labels, v)
	case "assignee":
		return i.assignee == v || v == "none" && i.assignee == ""
	}
	return true
}

func (i issue) matches(q querybar.Query) bool {
	for _, t := range q.Terms {
		if i.has(t) == t.Not {
			return false
		}
	}
	title := strings.ToLower(i.title)
	for _, w := range strings.Fields(strings.ToLower(q.Text)) {
		if !strings.Contains(title, w) {
			return false
		}
	}
	return true
}

type app struct {
	query string
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	q := querybar.Parse(a.query)
	var shown []issue
	for _, i := range issues {
		if i.matches(q) {
			shown = append(shown, i)
		}
	}
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(8)).Gap(t.Space(4)).Children(func() {
		ui.Text(c, "Issues").FontSize(t.FontSize + 8).Bold()
		ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
			querybar.Input(c, &a.query, fields).Grow(1).Padding(t.Space(2), t.Space(3)).Radius(t.Radius).
				Border(1, t.Border).Placeholder("Filter issues").Label("Filter")
			if ui.Button(c, "Clear").Disabled(a.query == "").Clicked() {
				a.query = ""
			}
		})
		ui.Text(c, fmt.Sprintf("%d of %d issues", len(shown), len(issues))).TextColor(t.TextMuted)
		if len(shown) == 0 {
			return
		}
		ui.Column(c).Border(1, t.Border).Radius(t.Radius).Clip().Dividers(1, t.Border).Children(func() {
			for _, i := range shown {
				row(c, i)
			}
		})
	})
}

func row(c *ui.Context, i issue) {
	t := c.Theme()
	ui.Row(c.Key(i.n)).Gap(t.Space(3)).Padding(t.Space(3), t.Space(4)).AlignItems(ui.Start).Children(func() {
		state := t.Success
		if !i.open {
			state = ui.RGB(130, 80, 223)
		}
		ui.Box(c).Size(10, 10).Radius(5).Background(state).Margin(5, 0, 0, 0)
		ui.Column(c).Gap(t.Space(1)).Grow(1).Shrink(1).Children(func() {
			ui.Row(c).Gap(t.Space(2)).Wrap().AlignItems(ui.Center).Children(func() {
				ui.Text(c, i.title).Bold()
				for _, l := range i.labels {
					ui.Text(c, l).FontSize(t.FontSize-2).Padding(1, t.Space(2)).Radius(999).Background(t.Accent.Alpha(0.12)).TextColor(t.Accent)
				}
			})
			by := fmt.Sprintf("#%d opened by %s", i.n, i.author)
			if i.assignee != "" {
				by += ", assigned to " + i.assignee
			}
			ui.Text(c, by).FontSize(t.FontSize - 1).TextColor(t.TextMuted)
		})
	})
}

func main() {
	a := &app{query: "is:open label:bug "}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Querybar",
			Width:   760,
			Height:  640,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
