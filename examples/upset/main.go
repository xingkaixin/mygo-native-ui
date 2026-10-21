// Upset shows which of six features 1,200 made-up users of a workspace app
// used in a month as an UpSet plot: a column for each combination of
// features, as many users as used exactly those. Point at a column to name
// its combination, click it to select it and Shift-click to add more, and
// the list beside the plot names the users selected; choose how the columns
// sort, how many show, and whether combinations no one used show too.
//
//	go run ./examples/upset
package main

import (
	"log"
	"math/rand/v2"
	"strconv"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/upset"
)

var features = []string{"Mail", "Calendar", "Docs", "Sheets", "Slides", "Chat"}

// The indexes of features.
const (
	mail = iota
	calendar
	docs
	sheets
	slides
	chat
)

// user is a made-up user: their name, and the indexes of the features they
// used.
type user struct {
	name string
	uses []int
}

var shows = []string{"Top 10", "Top 20", "All"}

type app struct {
	users []user
	uses  [][]int
	plot  upset.State
	order int
	show  string
	list  ui.ListState
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	a.plot.Order = upset.Order(a.order)
	a.plot.Limit = map[string]int{"Top 10": 10, "Top 20": 20}[a.show]
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(4), t.Space(5)).Gap(t.Space(4)).Children(func() {
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			ui.Column(c).Gap(t.Space(0.5)).Grow(1).Shrink(1).Children(func() {
				ui.Text(c, "Features used in October").Bold().FontSize(t.FontSize + 4)
				ui.Text(c, "1,200 made-up users of a workspace app and the six features they used").
					TextColor(t.TextMuted).SingleLine()
			})
			ui.Text(c, "Sort by").TextColor(t.TextMuted)
			ui.Segmented(c, &a.order, "Size", "Degree").Label("Sort by")
			ui.Text(c, "Show").TextColor(t.TextMuted)
			ui.Select(c, &a.show, shows).Label("Show")
			ui.Text(c, "Empty").TextColor(t.TextMuted)
			ui.Switch(c, &a.plot.Empty).Label("Show empty combinations")
		})
		var ev upset.Events
		ui.Row(c).Grow(1).AlignSelf(ui.Stretch).Gap(t.Space(5)).Children(func() {
			ui.Column(c).Grow(1).Shrink(1).AlignSelf(ui.Stretch).MinHeight(t.Space(80)).Children(func() {
				ev = upset.View(c, &a.plot, features, a.uses)
			})
			a.selection(c, ev.Selected())
		})
	})
}

// selection lists the users chosen, with the features each used.
func (a *app) selection(c *ui.Context, chosen []int) {
	t := c.Theme()
	ui.Column(c).Width(t.Space(72)).AlignSelf(ui.Stretch).Radius(t.Space(2.5)).Border(1, t.Border).Children(func() {
		ui.Column(c).Padding(t.Space(3), t.Space(4)).Gap(t.Space(0.5)).Children(func() {
			ui.Text(c, "Selected users").Bold()
			line := "Click a column to list its users"
			if len(chosen) > 0 {
				line = plural(len(chosen), "user")
			}
			ui.Text(c, line).FontSize(t.FontSize - 1).TextColor(t.TextMuted)
		})
		ui.Divider(c)
		ui.List(c, &a.list, len(chosen), func(i int) {
			u := a.users[chosen[i]]
			names := make([]string, len(u.uses))
			for k, f := range u.uses {
				names[k] = features[f]
			}
			ui.Row(c).Padding(t.Space(1.5), t.Space(4)).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
				ui.Text(c, u.name).SingleLine().Grow(1).Shrink(1)
				ui.Text(c, strings.Join(names, ", ")).FontSize(t.FontSize - 2).TextColor(t.TextMuted).SingleLine()
			})
		}).Grow(1).Padding(t.Space(1), 0)
	})
}

// plural writes n and noun, with an s for other than one: "1 user", "1,200
// users".
func plural(n int, noun string) string {
	s := grouped(n) + " " + noun
	if n != 1 {
		s += "s"
	}
	return s
}

// grouped writes n with commas between groups of three digits.
func grouped(n int) string {
	s := strconv.Itoa(n)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}

var first = []string{
	"Ada", "Ben", "Chloe", "Dev", "Elena", "Farid", "Grace", "Hugo", "Iris", "Jonas", "Kenji", "Lena",
	"Mateo", "Nora", "Omar", "Priya", "Quinn", "Rosa", "Sami", "Tara", "Uma", "Victor", "Wen", "Yara", "Zoe",
}

var last = []string{
	"Abbott", "Brandt", "Castillo", "Dubois", "Eriksen", "Fischer", "Gallo", "Haas", "Ibarra", "Jensen",
	"Kowalski", "Lindqvist", "Moreau", "Nakamura", "Okafor", "Petrov", "Quint", "Rossi", "Sato", "Tanaka",
	"Ueda", "Vidal", "Weber", "Xu", "Young", "Zhang",
}

// users makes up n users, the same each run: most use mail, and those
// who do mostly use the calendar too; sheets and slides go with docs, and
// chat with the calendar. Each uses one feature at least.
func users(n int) []user {
	r := rand.New(rand.NewPCG(2026, 10))
	// chance is true as often as with says, or without when given is false.
	chance := func(given bool, with, without float64) bool {
		if given {
			return r.Float64() < with
		}
		return r.Float64() < without
	}
	out := make([]user, n)
	for i := range out {
		used := [6]bool{}
		used[mail] = chance(true, 0.78, 0)
		used[calendar] = chance(used[mail], 0.62, 0.15)
		used[docs] = chance(true, 0.52, 0)
		used[sheets] = chance(used[docs], 0.48, 0.18)
		used[slides] = chance(used[docs], 0.3, 0.04)
		used[chat] = chance(used[calendar], 0.55, 0.3)
		u := user{name: first[r.IntN(len(first))] + " " + last[r.IntN(len(last))]}
		for f, on := range used {
			if on {
				u.uses = append(u.uses, f)
			}
		}
		if u.uses == nil {
			u.uses = []int{[]int{mail, docs, chat}[r.IntN(3)]}
		}
		out[i] = u
	}
	return out
}

func main() {
	a := &app{users: users(1200), show: "Top 20"}
	for _, u := range a.users {
		a.uses = append(a.uses, u.uses)
	}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "UpSet plot",
			Width:   1240,
			Height:  760,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
