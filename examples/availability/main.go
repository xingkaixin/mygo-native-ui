// Availability picks a time for a planning call this week, as When2meet
// does: paint the times you are free on your grid, and the group's grid
// beside it shades each half hour by how many of six people are free,
// outlining the best. Hover a slot to see who is free then, and a person
// to see when they are.
//
//	go run ./examples/availability
package main

import (
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/availability"
)

// others are five made-up people and the hours they are free each day of
// the week, from Monday, as "9-12 13:30-17".
var others = []struct {
	name string
	free [7]string
}{
	{"Ana Lima", [7]string{"9-12 14-17", "10-16", "9-11 13-16", "13-18", "9-15", "10-12", ""}},
	{"Ben Okafor", [7]string{"10-13", "9-11 14-18", "13:30-17", "9-12", "11-16", "", "14-17"}},
	{"Chloé Martin", [7]string{"13-18", "9-12", "9-10:30 14-18", "10-14", "9-12 15-17", "10-13", ""}},
	{"Dev Patel", [7]string{"9-11 15-18", "11-15", "12-16", "14-18", "10-13", "", "11-15"}},
	{"Emi Sato", [7]string{"9-10:30", "13-17", "10-12 14-15:30", "9-11 15-17", "12-17", "9-11", "13-16"}},
}

// mine is when I am free to begin with.
var mine = [7]string{"", "10-12", "14-16", "15-17", "", "", ""}

type app struct {
	grid   availability.Grid
	group  availability.State
	mine   []time.Time
	others []availability.Person
	saved  time.Time
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	everyone := append([]availability.Person{{Name: "You", Free: a.mine}}, a.others...)
	ui.Scroll(c).Fill().Background(t.Background).Children(func() {
		ui.Column(c).Padding(t.Space(6)).Gap(t.Space(6)).Children(func() {
			ui.Column(c).Gap(t.Space(1)).Children(func() {
				ui.Text(c, "Q4 planning call").Bold().FontSize(t.FontSize + 6)
				ui.Text(c, "Find an hour this week that suits the team").TextColor(t.TextMuted)
			})
			ui.Row(c).Gap(t.Space(8)).AlignItems(ui.Start).Children(func() {
				ui.Column(c).Grow(1).Basis(0).Gap(t.Space(3)).Children(func() {
					note := "Drag across the times you are free"
					if !a.saved.IsZero() {
						note = "Saved at " + a.saved.Format("15:04:05")
					}
					heading(c, "Your availability", note)
					if availability.Mine(c, &a.group, a.grid, &a.mine).AlignSelf(ui.Stretch).Changed() {
						a.saved = c.Now()
					}
				})
				ui.Column(c).Grow(1).Basis(0).Gap(t.Space(3)).Children(func() {
					heading(c, "Group's availability", strconv.Itoa(len(everyone))+" people")
					availability.Group(c, &a.group, a.grid, everyone).AlignSelf(ui.Stretch)
				})
				ui.Column(c).Width(t.Space(56)).Gap(t.Space(3)).Children(func() {
					heading(c, "People", "Hover to see when")
					availability.People(c, &a.group, a.grid, everyone).AlignSelf(ui.Stretch)
				})
			})
		})
	})
}

// heading titles a part of the window, with a note below.
func heading(c *ui.Context, title, note string) {
	t := c.Theme()
	ui.Column(c).Gap(t.Space(0.5)).Children(func() {
		ui.Text(c, title).Bold()
		ui.Text(c, note).FontSize(t.FontSize - 1).TextColor(t.TextMuted)
	})
}

// slots returns the starts of the half hours in the hours of each day of
// week, from monday.
func slots(monday time.Time, week [7]string) []time.Time {
	var out []time.Time
	for d, spans := range week {
		day := monday.AddDate(0, 0, d)
		for _, span := range strings.Fields(spans) {
			from, to, _ := strings.Cut(span, "-")
			for m := minutes(from); m < minutes(to); m += 30 {
				out = append(out, time.Date(day.Year(), day.Month(), day.Day(), 0, m, 0, 0, day.Location()))
			}
		}
	}
	return out
}

// minutes returns the minutes after midnight of a time as "9" or "13:30".
func minutes(s string) int {
	h, m, _ := strings.Cut(s, ":")
	hours, _ := strconv.Atoi(h)
	mins, _ := strconv.Atoi(m)
	return hours*60 + mins
}

func main() {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	monday := today.AddDate(0, 0, -(int(today.Weekday())+6)%7)
	a := &app{grid: availability.Grid{From: 9, To: 18}, mine: slots(monday, mine)}
	for d := range 7 {
		a.grid.Days = append(a.grid.Days, monday.AddDate(0, 0, d))
	}
	for _, p := range others {
		a.others = append(a.others, availability.Person{Name: p.name, Free: slots(monday, p.free)})
	}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Availability",
			Width:   1280,
			Height:  640,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
