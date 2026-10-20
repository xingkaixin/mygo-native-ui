// Monthview is a calendar's month, as macOS Calendar shows it: a
// conference, a trip and birthdays as bars, meetings by their time. Drag
// an event to another day, and click a day to list its events beside the
// month.
//
//	go run ./examples/monthview
package main

import (
	"fmt"
	"log"
	"slices"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/i18n"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/monthview"
)

// app is the events, the month shown, the view's state and how many
// events were moved.
type app struct {
	events   []monthview.Event
	month    time.Time
	calendar monthview.State
	moves    int
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Row(c).Fill().Background(t.Background).AlignItems(ui.Stretch).Children(func() {
		if monthview.View(c, &a.calendar, &a.month, &a.events).Grow(1).Changed() {
			a.moves++
		}
		ui.Divider(c).Vertical()
		a.agenda(c)
	})
}

// agenda lists the events of the day chosen, or of today.
func (a *app) agenda(c *ui.Context) {
	t := c.Theme()
	locale := c.Region().FormatLocale(c.Locale())
	d, ok := a.calendar.Chosen()
	if !ok {
		d = midnight(c.Now())
	}
	next := d.AddDate(0, 0, 1)
	var on []monthview.Event
	for _, e := range a.events {
		if e.Start.Before(next) && e.End.After(d) {
			on = append(on, e)
		}
	}
	slices.SortStableFunc(on, func(x, y monthview.Event) int { return x.Start.Compare(y.Start) })
	ui.Column(c).Width(t.Space(64)).Padding(t.Space(5)).Gap(t.Space(4)).Children(func() {
		ui.Column(c).Gap(t.Space(0.5)).Children(func() {
			ui.Text(c, d.Format("Monday")).FontSize(t.FontSize - 1).Bold().TextColor(t.Accent)
			ui.Text(c, d.Format("January 2, 2006")).FontSize(t.FontSize + 4).Bold()
		})
		if len(on) == 0 {
			ui.Text(c, "No events").TextColor(t.TextMuted)
		}
		for i, e := range on {
			ui.Row(c.Key(i)).Gap(t.Space(2.5)).AlignItems(ui.Stretch).Children(func() {
				color := e.Color
				if color == (ui.Color{}) {
					color = t.Accent
				}
				ui.Box(c).Width(3).Radius(1.5).Background(color)
				ui.Column(c).Gap(t.Space(0.5)).Shrink(1).Children(func() {
					ui.Text(c, e.Title).Bold()
					ui.Text(c, when(locale, e)).FontSize(t.FontSize - 1).FontFeatures("tnum").TextColor(t.TextMuted)
				})
			})
		}
		ui.Box(c).Grow(1)
		ui.Text(c, fmt.Sprintf("%d events · %d moved", len(a.events), a.moves)).FontSize(t.FontSize - 2).TextColor(t.TextMuted)
	})
}

// when writes when e is in the regional formats of locale, as the month
// does: "All day", "Oct 28 – Nov 3" or "3:00 PM – 4:00 PM".
func when(locale string, e monthview.Event) string {
	whole := e.Start.Equal(midnight(e.Start)) && e.End.Equal(midnight(e.End))
	last := e.End.Add(-time.Nanosecond)
	format := func(tm time.Time, skeleton string) string {
		return i18n.FormatDate(locale, tm, i18n.DateOptions{Skeleton: skeleton})
	}
	switch {
	case whole && midnight(last).Equal(e.Start):
		return "All day"
	case whole:
		return format(e.Start, "MMMd") + " – " + format(last, "MMMd")
	}
	return format(e.Start, "jm") + " – " + format(e.End, "jm")
}

func midnight(tm time.Time) time.Time {
	y, m, d := tm.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, tm.Location())
}

// events makes up a month of events around the month of today: weekly
// meetings, a conference over a weekend, a holiday, birthdays, and a trip
// into the next month.
func events(today time.Time) []monthview.Event {
	first := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, today.Location())
	purple, green, orange, red, teal, indigo := ui.RGB(175, 82, 222), ui.RGB(52, 199, 89), ui.RGB(255, 149, 0), ui.RGB(255, 59, 48), ui.RGB(48, 176, 199), ui.RGB(88, 86, 214)
	var out []monthview.Event
	timed := func(title string, day, hour, minute, length int, color ui.Color) {
		start := first.AddDate(0, 0, day-1).Add(time.Duration(hour)*time.Hour + time.Duration(minute)*time.Minute)
		out = append(out, monthview.Event{Title: title, Start: start, End: start.Add(time.Duration(length) * time.Minute), Color: color})
	}
	allDay := func(title string, day, days int, color ui.Color) {
		start := first.AddDate(0, 0, day-1)
		out = append(out, monthview.Event{Title: title, Start: start, End: start.AddDate(0, 0, days), Color: color})
	}
	// weekday returns the day of the month of the n-th wd, from 0.
	weekday := func(wd time.Weekday, n int) int {
		return 1 + (int(wd)-int(first.Weekday())+7)%7 + 7*n
	}
	last := first.AddDate(0, 1, -1).Day()
	for n := 0; weekday(time.Monday, n) <= last; n++ {
		timed("Team sync", weekday(time.Monday, n), 10, 0, 60, ui.Color{})
	}
	for n := 0; weekday(time.Thursday, n) <= last; n++ {
		timed("Design review", weekday(time.Thursday, n), 15, 0, 60, purple)
	}
	for n := 0; weekday(time.Wednesday, n) <= last; n += 2 {
		timed("1:1 with Sam", weekday(time.Wednesday, n), 11, 30, 30, green)
	}
	// Saturday to Monday crosses into the next week whether weeks start
	// on Sunday or on Monday.
	allDay("Systems Conference", weekday(time.Saturday, 1), 3, indigo)
	allDay("Company holiday", weekday(time.Friday, 2), 1, red)
	thursday := weekday(time.Thursday, 2)
	allDay("Maya's birthday", thursday, 1, orange)
	timed("Gym", thursday, 7, 0, 60, green)
	timed("Code freeze", thursday, 9, 0, 30, red)
	timed("Release v2.4", thursday, 16, 0, 30, red)
	timed("Dinner with Alex", thursday, 19, 30, 120, orange)
	allDay("Dad's birthday", 24, 1, orange)
	allDay("Trip to Lisbon", last-3, 7, teal)
	timed("Flight to Lisbon", last-3, 7, 45, 165, teal)
	timed("Book club", 2, 19, 0, 90, purple)
	timed("Lunch with Priya", 9, 12, 30, 60, orange)
	timed("Dentist", 13, 9, 0, 45, ui.Color{})
	timed("Interview", 20, 14, 0, 60, red)
	timed("Quarterly planning", 21, 9, 30, 120, ui.Color{})
	return out
}

func main() {
	now := time.Now()
	a := &app{month: now, events: events(now)}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Calendar",
			Width:   1280,
			Height:  760,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
