// Filterbuilder filters an issue tracker's list as Linear and Notion do:
// 240 issues, made up, below a builder of conditions on their title,
// status, priority, label, assignee, estimate, date and whether they are
// blocked, which the list follows as it changes, counting the issues that
// match.
//
//	go run ./examples/filterbuilder
package main

import (
	"fmt"
	"log"
	"math/rand/v2"
	"slices"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/filterbuilder"
)

var (
	statuses   = []string{"Backlog", "Todo", "In progress", "In review", "Done", "Canceled"}
	priorities = []string{"Urgent", "High", "Medium", "Low", "None"}
	labels     = []string{"Bug", "Feature", "Chore", "Docs", "Performance"}
	people     = []string{"Ada", "Bruno", "Chen", "Dara", "Emeka", "Freya"}
)

var fields = []filterbuilder.Field{
	{Name: "Title", Kind: filterbuilder.Text},
	{Name: "Status", Kind: filterbuilder.Select, Options: statuses},
	{Name: "Priority", Kind: filterbuilder.Select, Options: priorities},
	{Name: "Label", Kind: filterbuilder.Select, Options: labels},
	{Name: "Assignee", Kind: filterbuilder.Select, Options: people},
	{Name: "Estimate", Kind: filterbuilder.Number},
	{Name: "Created", Kind: filterbuilder.Date},
	{Name: "Blocked", Kind: filterbuilder.Bool},
}

type issue struct {
	id                                   int
	title, status, priority, label, user string
	estimate                             int
	created                              time.Time
	blocked                              bool
	record                               map[string]any
}

type app struct {
	issues []*issue
	filter filterbuilder.Filter
	list   ui.ListState
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	var shown []*issue
	for _, is := range a.issues {
		if filterbuilder.Match(&a.filter, fields, is.record) {
			shown = append(shown, is)
		}
	}
	ui.Column(c).Fill().Background(t.Background).Children(func() {
		ui.Scroll(c).MaxHeight(t.Space(100)).Children(func() {
			ui.Column(c).Padding(t.Space(4), t.Space(5)).Gap(t.Space(3)).Children(func() {
				ui.Text(c, "Filter").Bold()
				filterbuilder.View(c, &a.filter, fields)
			})
		})
		ui.Divider(c)
		ui.Textf(c, "%d of %d issues", len(shown), len(a.issues)).Bold().FontFeatures("tnum").
			Padding(t.Space(3), t.Space(5))
		ui.Divider(c)
		ui.List(c, &a.list, len(shown), func(i int) { row(c, shown[i]) }).Grow(1).Children(func() {
			if len(shown) == 0 {
				ui.Text(c, "No issues match").TextColor(t.TextMuted).Padding(t.Space(4), t.Space(5))
			}
		})
	})
}

// row shows an issue: its id, status, title, label, priority, assignee,
// estimate and date.
func row(c *ui.Context, is *issue) {
	t := c.Theme()
	small := t.FontSize - 1
	ui.Column(c).Children(func() {
		ui.Row(c).Padding(t.Space(2), t.Space(5)).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, fmt.Sprintf("ENG-%d", is.id)).FontSize(small).TextColor(t.TextMuted).FontFeatures("tnum").Width(t.Space(16))
			ui.Row(c).Width(t.Space(24)).Gap(t.Space(1.5)).AlignItems(ui.Center).Children(func() {
				ui.Box(c).Size(t.Space(2), t.Space(2)).Radius(t.Space(1)).Background(statusColor(t, is.status))
				ui.Text(c, is.status).FontSize(small).SingleLine()
			})
			ui.Text(c, is.title).SingleLine().Grow(1).Shrink(1).MinWidth(0)
			if is.blocked {
				ui.Text(c, "Blocked").FontSize(small).TextColor(t.Danger)
			}
			ui.Badge(c, is.label)
			ui.Text(c, is.priority).FontSize(small).TextColor(t.TextMuted).Width(t.Space(14))
			ui.Text(c, is.user).FontSize(small).Width(t.Space(13))
			ui.Text(c, fmt.Sprintf("%d pt", is.estimate)).FontSize(small).TextColor(t.TextMuted).FontFeatures("tnum").
				Width(t.Space(9)).TextAlign(ui.End)
			ui.Text(c, is.created.Format("Jan 2")).FontSize(small).TextColor(t.TextMuted).FontFeatures("tnum").
				Width(t.Space(13)).TextAlign(ui.End)
		})
		ui.Divider(c)
	})
}

func statusColor(t *ui.Theme, status string) ui.Color {
	switch status {
	case "Todo":
		return t.Text
	case "In progress":
		return t.Warning
	case "In review":
		return t.Accent
	case "Done":
		return t.Success
	}
	return t.TextMuted
}

// issues makes up n issues, the newest first.
func issues(n int) []*issue {
	r := rand.New(rand.NewPCG(7, 240))
	verbs := []string{"Fix", "Add", "Remove", "Improve", "Refactor", "Document", "Speed up", "Investigate"}
	things := []string{
		"login redirect", "CSV export", "dark mode colors", "search indexing", "webhook retries",
		"billing page", "rate limiter", "onboarding emails", "file uploads", "audit log",
		"keyboard shortcuts", "SSO setup", "API pagination", "mobile layout", "notification settings",
		"project templates", "time zone handling", "image previews",
	}
	contexts := []string{"", "", "", " on Safari", " for large teams", " in the desktop app", " after the migration"}
	pick := func(xs []string) string { return xs[r.IntN(len(xs))] }
	now := time.Now()
	out := make([]*issue, n)
	for i := range out {
		is := &issue{
			id:       1240 - i,
			title:    pick(verbs) + " " + pick(things) + pick(contexts),
			status:   pick(statuses),
			priority: pick(priorities),
			label:    pick(labels),
			user:     pick(people),
			estimate: []int{1, 2, 3, 5, 8}[r.IntN(5)],
			created:  now.Add(-time.Duration(r.IntN(120*24)) * time.Hour),
			blocked:  r.IntN(8) == 0,
		}
		is.record = map[string]any{
			"Title": is.title, "Status": is.status, "Priority": is.priority, "Label": is.label,
			"Assignee": is.user, "Estimate": is.estimate, "Created": is.created, "Blocked": is.blocked,
		}
		out[i] = is
	}
	slices.SortFunc(out, func(a, b *issue) int { return b.created.Compare(a.created) })
	return out
}

func main() {
	a := &app{
		issues: issues(240),
		filter: filterbuilder.Filter{Conditions: []filterbuilder.Condition{
			{Field: "Status", Op: filterbuilder.AnyOf, Values: []string{"Todo", "In progress"}},
			{Group: &filterbuilder.Filter{Or: true, Conditions: []filterbuilder.Condition{
				{Field: "Priority", Op: filterbuilder.AnyOf, Values: []string{"Urgent", "High"}},
				{Field: "Label", Op: filterbuilder.AnyOf, Values: []string{"Bug"}},
			}}},
			{Field: "Created", Op: filterbuilder.WithinLast, Number: 60},
		}},
	}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Issues",
			Width:   1080,
			Height:  760,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
