// Yearview shows this year of a made-up team's calendar: holidays, a
// conference trip over several days, two-week sprints and their reviews,
// and birthdays. Hover a day to list its events, and click one to show it
// below.
//
//	go run ./examples/yearview
package main

import (
	"log"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/yearview"
)

var (
	holiday  = ui.Hex("#ef4444")
	trip     = ui.Hex("#f59e0b")
	work     = ui.Hex("#3b82f6")
	birthday = ui.Hex("#a855f7")
)

func events(year int) []yearview.Event {
	day := func(m time.Month, d int) time.Time { return time.Date(year, m, d, 0, 0, 0, 0, time.Local) }
	all := func(title string, m time.Month, d, days int, col ui.Color) yearview.Event {
		return yearview.Event{Title: title, Start: day(m, d), End: day(m, d).AddDate(0, 0, days), Color: col}
	}
	es := []yearview.Event{
		all("New Year's Day", 1, 1, 1, holiday),
		all("Spring Festival", 2, 17, 3, holiday),
		all("Labour Day", 5, 1, 1, holiday),
		all("GopherCon", 6, 9, 4, trip),
		all("Summer offsite", 8, 18, 2, trip),
		all("National Day", 10, 1, 7, holiday),
		all("Christmas", 12, 25, 1, holiday),
		all("Ada's birthday", 3, 12, 1, birthday),
		all("Grace's birthday", 12, 9, 1, birthday),
		all("Alan's birthday", 6, 23, 1, birthday),
	}
	for d := day(1, 12); d.Year() == year; d = d.AddDate(0, 0, 14) {
		review := d.AddDate(0, 0, 11).Add(15 * time.Hour)
		es = append(es, yearview.Event{Title: "Sprint review", Start: review, End: review.Add(time.Hour), Color: work})
	}
	return es
}

type app struct {
	date   time.Time
	year   yearview.State
	events []yearview.Event
	shown  int
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	if a.date.Year() != a.shown {
		a.shown = a.date.Year()
		a.events = events(a.shown)
	}
	ui.Column(c).Fill().Background(t.Background).Children(func() {
		yearview.View(c, &a.year, &a.date, a.events).Grow(1)
		ui.Divider(c)
		ui.Row(c).Padding(t.Space(3), t.Space(4)).Gap(t.Space(4)).Children(func() {
			day, ok := a.year.Chosen()
			if !ok {
				ui.Text(c, "Click a day to see its events").TextColor(t.TextMuted)
				return
			}
			ui.Text(c, day.Format("Monday, January 2")).Bold()
			n := 0
			for _, e := range a.events {
				if e.Start.Before(day.AddDate(0, 0, 1)) && e.End.After(day) {
					ui.Row(c).Gap(t.Space(1.5)).AlignItems(ui.Center).Children(func() {
						ui.Box(c).Size(8, 8).Radius(4).Background(e.Color)
						ui.Text(c, e.Title)
					})
					n++
				}
			}
			if n == 0 {
				ui.Text(c, "No events").TextColor(t.TextMuted)
			}
		})
	})
}

func main() {
	a := &app{}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Year",
			Width:   1040,
			Height:  760,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
