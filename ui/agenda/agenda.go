// Package agenda lists the events of the days ahead under their days, as
// calendar apps' list and schedule views do: a heading for each day with
// events, "Today" and "Tomorrow" by name, that stays at the top while its
// events scroll under it; and for each event its time or "All day", a bar
// of its color, its title and where it is. An event over several days
// shows on each, as "Day 2 of 3". Events that have ended are muted, and
// one going on is tinted in its color and marked Now. A click chooses an
// event, which Chosen returns.
//
//	if agenda.View(c, &app.agenda, time.Now(), 30, app.events).Grow(1).Changed() {
//		app.open(app.agenda.Chosen())
//	}
package agenda

import (
	"slices"
	"strconv"
	"time"

	"github.com/egoist/mygo/i18n"
	"github.com/egoist/mygo/ui"
)

// Event is a stretch of time on the calendar. One from midnight to
// midnight, or over more than one day, is all day on its days.
type Event struct {
	Title      string
	Start, End time.Time
	// Location says where the event is, under its title.
	Location string
	// Color colors the event's bar; the zero Color is the accent.
	Color ui.Color
}

// State is what an agenda keeps: the event chosen and where the list is.
// Keep it in the app's state, one for each agenda.
type State struct {
	chosen int
	row    int
	list   ui.ListState
	rows   []row
}

// Chosen returns the index of the event chosen, or -1 with none.
func (s *State) Chosen() int { return s.chosen - 1 }

// row is a row of the list: a day's heading, or an event on it.
type row struct {
	day   time.Time
	event int
	// nth and of number the day among an event's days, when more than
	// one.
	nth, of int
}

// View creates the agenda of events on the days days from the day of
// from. Changed reports an event chosen. Give it a size, or grow it in its
// parent.
func View(c *ui.Context, s *State, from time.Time, days int, events []Event) ui.Element {
	t := c.Theme()
	locale := c.Region().FormatLocale(c.Locale())
	loc := from.Location()
	now := c.Now().In(loc)
	c.After(now.Truncate(time.Minute).Add(time.Minute).Sub(now))
	today := midnight(now)
	start := midnight(from)
	s.rows = s.rows[:0]
	for d := range days {
		day := start.AddDate(0, 0, d)
		next := day.AddDate(0, 0, 1)
		var on []int
		for i, e := range events {
			if e.Start.Before(next) && end(e).After(day) {
				on = append(on, i)
			}
		}
		if len(on) == 0 {
			continue
		}
		slices.SortStableFunc(on, func(a, b int) int {
			aa, ba := allDay(events[a], day), allDay(events[b], day)
			if aa != ba {
				if aa {
					return -1
				}
				return 1
			}
			return events[a].Start.Compare(events[b].Start)
		})
		s.rows = append(s.rows, row{day: day, event: -1})
		for _, i := range on {
			e := events[i]
			first, last := midnight(e.Start.In(loc)), midnight(end(e).Add(-time.Nanosecond).In(loc))
			of := int(last.Sub(first).Hours()/24+0.5) + 1
			nth := int(day.Sub(first).Hours()/24+0.5) + 1
			s.rows = append(s.rows, row{day: day, event: i, nth: nth, of: of})
		}
	}
	s.row = -1
	for r, rw := range s.rows {
		if rw.event >= 0 && rw.event == s.chosen-1 {
			s.row = r
		}
	}
	s.list.Selected = &s.row
	s.list.Header = func(r int) bool { return s.rows[r].event < 0 }
	s.list.Key = func(r int) any {
		rw := s.rows[r]
		return [2]int64{rw.day.Unix(), int64(rw.event)}
	}
	s.list.RoundRows = true
	list := ui.List(c, &s.list, len(s.rows), func(r int) {
		rw := s.rows[r]
		if rw.event < 0 {
			heading(c, rw.day, today, locale)
			return
		}
		item(c, &events[rw.event], rw, now, locale, r == s.row)
	}).Label("Agenda")
	if list.Changed() {
		s.chosen = 0
		if s.row >= 0 {
			s.chosen = s.rows[s.row].event + 1
		}
	}
	list.Children(func() {
		if len(s.rows) == 0 {
			ui.Text(c, "No upcoming events").TextColor(t.TextMuted).AlignSelf(ui.Center).MarginY(t.Space(10))
		}
	})
	return list
}

// midnight returns the start of t's day.
func midnight(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// end returns when e ends, a moment after it starts at least.
func end(e Event) time.Time {
	if e.End.After(e.Start) {
		return e.End
	}
	return e.Start.Add(time.Nanosecond)
}

// allDay reports whether e takes the whole of day.
func allDay(e Event, day time.Time) bool {
	return !e.Start.After(day) && !end(e).Before(day.AddDate(0, 0, 1))
}

// heading creates the heading of day.
func heading(c *ui.Context, day, today time.Time, locale string) {
	t := c.Theme()
	name := i18n.FormatDate(locale, day, i18n.DateOptions{Skeleton: "MMMMEEEEd"})
	col := t.Text
	switch {
	case day.Equal(today):
		name, col = "Today · "+i18n.FormatDate(locale, day, i18n.DateOptions{Skeleton: "MMMMd"}), t.Accent
	case day.Equal(today.AddDate(0, 0, 1)):
		name = "Tomorrow · " + i18n.FormatDate(locale, day, i18n.DateOptions{Skeleton: "MMMMd"})
	case day.Year() != today.Year():
		name = i18n.FormatDate(locale, day, i18n.DateOptions{Skeleton: "yMMMMEEEEd"})
	}
	ui.Text(c, name).Bold().TextColor(col).Background(t.Background).Padding(t.Space(4), t.Space(4), t.Space(1.5)).
		Role(ui.RoleHeading)
}

// item creates the row of e on the day of rw, muted once it has ended
// and tinted while it goes on.
func item(c *ui.Context, e *Event, rw row, now time.Time, locale string, chosen bool) {
	t := c.Theme()
	col := e.Color
	if col == (ui.Color{}) {
		col = t.Accent
	}
	ink, muted := t.Text, t.TextMuted
	over := !end(*e).After(now)
	if over {
		ink, muted = t.TextMuted, t.TextMuted.Alpha(0.7)
	}
	if chosen {
		ink, muted = t.AccentText, t.AccentText.Alpha(0.8)
	}
	when, until := "All day", ""
	if !allDay(*e, rw.day) {
		hm := i18n.DateOptions{Skeleton: "jm", HourCycle: c.Region().HourCycle}
		switch {
		case e.Start.Before(rw.day):
			when, until = "Until", i18n.FormatDate(locale, e.End, hm)
		case end(*e).After(rw.day.AddDate(0, 0, 1)):
			when, until = i18n.FormatDate(locale, e.Start, hm), "Overnight"
		default:
			when = i18n.FormatDate(locale, e.Start, hm)
			if e.End.After(e.Start) {
				until = i18n.FormatDate(locale, e.End, hm)
			}
		}
	}
	live := !over && !e.Start.After(now) && rw.day.Equal(midnight(now))
	r := ui.Row(c).Padding(t.Space(2), t.Space(4)).Gap(t.Space(3)).AlignItems(ui.Stretch)
	if live && !chosen {
		r.Background(col.Alpha(0.08))
	}
	r.Children(func() {
		ui.Column(c).Width(t.Space(18)).Children(func() {
			ui.Text(c, when).FontSize(t.FontSize - 1).TextColor(ink).FontFeatures("tnum").SingleLine()
			if until != "" {
				ui.Text(c, until).FontSize(t.FontSize - 2).TextColor(muted).FontFeatures("tnum").SingleLine()
			}
		})
		bar := col
		if over && !chosen {
			bar = col.Alpha(0.4)
		}
		ui.Box(c).Width(t.Space(1)).Radius(t.Space(0.5)).Background(bar)
		ui.Column(c).Grow(1).Shrink(1).MinWidth(0).Gap(t.Space(0.5)).Children(func() {
			title := e.Title
			if rw.of > 1 {
				title += " (Day " + strconv.Itoa(rw.nth) + " of " + strconv.Itoa(rw.of) + ")"
			}
			ui.Text(c, title).TextColor(ink).FontWeight(500).SingleLine()
			if e.Location != "" {
				ui.Text(c, e.Location).FontSize(t.FontSize - 1).TextColor(muted).SingleLine()
			}
		})
		if live {
			ui.Text(c, "Now").FontSize(t.FontSize - 2).Bold().TextColor(col).AlignSelf(ui.Center)
		}
	})
}
