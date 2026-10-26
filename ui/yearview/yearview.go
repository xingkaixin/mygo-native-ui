// Package yearview shows a year as twelve small months, as macOS
// Calendar's year view does: the months in as many columns as fit, each
// with its name and its weeks from the first day of the week of the
// user's regional formats, today's date in a circle of the accent and the
// month it is in named in the accent. A day with events has its number
// bold and dots of their colors under it, three at most, and the pointer
// on it lists them. A click chooses a day, which Chosen returns, to show
// it in a month, a week or an agenda.
//
//	if yearview.View(c, &app.year, &app.date, app.events).Grow(1).Changed() {
//		day, _ := app.year.Chosen()
//		app.showMonth(day)
//	}
package yearview

import (
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo/i18n"
	"github.com/egoist/mygo/ui"
)

// Event is a stretch of time on the calendar, marking each day it is on.
type Event struct {
	Title      string
	Start, End time.Time
	// Color colors the event's dot; the zero Color is the accent.
	Color ui.Color
}

// State is what a year view keeps: the day chosen. Keep it in the app's
// state, one for each view.
type State struct {
	chosen time.Time
}

// Chosen returns the day chosen, at its midnight, or false with none.
func (s *State) Chosen() (time.Time, bool) { return s.chosen, !s.chosen.IsZero() }

// View creates the year view of the year of *date, any day in it, and of
// events. Changed reports a day chosen. Give it a size, or grow it in its
// parent.
//
// The buttons above go to the years before and after, and Today back to
// this year, choosing today.
func View(c *ui.Context, s *State, date *time.Time, events []Event) ui.Element {
	t := c.Theme()
	region := c.Region()
	locale := region.FormatLocale(c.Locale())
	if date.IsZero() {
		*date = c.Now()
	}
	loc := date.Location()
	now := c.Now().In(loc)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	c.After(today.AddDate(0, 0, 1).Sub(now))
	year := date.Year()
	marks := mark(events, year, loc)
	changed := false
	root := ui.Column(c)
	root.Children(func() {
		ui.Row(c).Padding(t.Space(3), t.Space(4)).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, strconv.Itoa(year)).Bold().FontSize(t.FontSize + 8).Grow(1)
			if ui.Button(c, "‹").Label("Previous year").Clicked() {
				*date = date.AddDate(-1, 0, 0)
			}
			if ui.Button(c, "Today").Clicked() {
				*date, s.chosen, changed = now, today, true
			}
			if ui.Button(c, "›").Label("Next year").Clicked() {
				*date = date.AddDate(1, 0, 0)
			}
		})
		months := i18n.Months(locale, i18n.Wide, true)
		names := i18n.Weekdays(locale, i18n.Narrow, true)
		grid := ui.Grid(c).Grow(1).Padding(0, t.Space(4), t.Space(4)).GapX(t.Space(6)).GapY(t.Space(3))
		b := grid.LayoutBounds()
		// The columns are those giving the days the most room.
		cols, best := 4, float32(0)
		for _, n := range []int{6, 4, 3, 2, 1} {
			rows := 12 / n
			w := (b.W - t.Space(8) - float32(n-1)*t.Space(6)) / float32(n) / 7
			h := (b.H - t.Space(4) - float32(rows-1)*t.Space(3)) / float32(rows) / 8.6
			if cell := min(w, h); cell > best {
				cols, best = n, cell
			}
		}
		tracks := make([]ui.Track, 12/cols)
		for i := range tracks {
			tracks[i] = ui.Fr(1)
		}
		grid.RowTracks(tracks...)
		grid.Columns(cols).Children(func() {
			for m := range 12 {
				first := time.Date(year, time.Month(m+1), 1, 0, 0, 0, 0, loc)
				mo := &month{first: first, today: today, chosen: s.chosen, weekday: region.FirstWeekday,
					name: months[m], days: names, marks: marks, events: events, locale: locale}
				if day, ok := mo.view(c.Key(m)); ok {
					s.chosen, changed = day, true
				}
			}
		})
	})
	if changed {
		root.SetChanged()
	}
	return root
}

// mark returns the events on each day of year, by the day's place in it
// from 0, in order.
func mark(events []Event, year int, loc *time.Location) map[int][]int {
	marks := map[int][]int{}
	start := time.Date(year, 1, 1, 0, 0, 0, 0, loc)
	end := start.AddDate(1, 0, 0)
	for i, e := range events {
		from, to := e.Start.In(loc), e.End.In(loc)
		if !to.After(from) {
			to = from.Add(time.Nanosecond)
		}
		if !to.After(start) || !from.Before(end) {
			continue
		}
		d := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, loc)
		if d.Before(start) {
			d = start
		}
		for ; d.Before(to) && d.Before(end); d = d.AddDate(0, 0, 1) {
			marks[d.YearDay()-1] = append(marks[d.YearDay()-1], i)
		}
	}
	return marks
}

// month is a month as a year view shows it.
type month struct {
	first, today, chosen time.Time
	weekday              time.Weekday
	name                 string
	days                 [7]string
	marks                map[int][]int
	events               []Event
	locale               string
}

// view creates the month, and returns the day clicked.
func (mo *month) view(c *ui.Context) (time.Time, bool) {
	t := c.Theme()
	lead := (int(mo.first.Weekday()) - int(mo.weekday) + 7) % 7
	n := mo.first.AddDate(0, 1, -1).Day()
	box := ui.Box(c).MinWidth(0).MinHeight(0).Label(mo.name)
	b := box.LayoutBounds()
	cell := max(1, min(b.W/7, (b.H-t.FontSize*2)/7.2))
	top := t.FontSize * 2
	at := func(d int) ui.Rect {
		k := lead + d - 1
		return ui.Rect{X: float32(k%7) * cell, Y: top + cell*(0.8+float32(k/7)), W: cell, H: cell}
	}
	hot := 0
	if x, y, ok := box.PointerPosition(); ok && box.Hovered() {
		for d := 1; d <= n; d++ {
			if at(d).Contains(x, y) {
				hot = d
			}
		}
	}
	if hot > 0 {
		box.Cursor(ui.CursorPointer)
		if list := mo.marks[mo.first.YearDay()-1+hot-1]; len(list) > 0 {
			box.Tooltip(mo.tip(hot, list))
		}
	}
	this := mo.first.Year() == mo.today.Year() && mo.first.Month() == mo.today.Month()
	box.Draw(func(p *ui.Painter, r ui.Rect) {
		title := t.Text
		if this {
			title = t.Accent
		}
		p.RichText(r.X+cell*0.25, r.Y, 0, ui.Span{Text: mo.name, Size: t.FontSize + 1, Weight: 700, Color: title})
		small := min(t.FontSize-2, cell*0.42)
		for i := range 7 {
			span := ui.Span{Text: mo.days[(int(mo.weekday)+i)%7], Size: small, Color: t.TextMuted}
			w, h := p.MeasureText(0, span)
			p.RichText(r.X+float32(i)*cell+(cell-w)/2, r.Y+top+(cell*0.8-h)/2, 0, span)
		}
		for d := 1; d <= n; d++ {
			cr := at(d)
			cr.X += r.X
			cr.Y += r.Y
			day := mo.first.AddDate(0, 0, d-1)
			marks := mo.marks[day.YearDay()-1]
			ink, weight := t.Text, 400
			if len(marks) > 0 {
				weight = 700
			}
			disc := ui.Rect{X: cr.X + cell*0.12, Y: cr.Y + cell*0.08, W: cell * 0.76, H: cell * 0.76}
			switch {
			case day.Equal(mo.today):
				p.Fill(disc, t.Accent, disc.W/2)
				ink = t.AccentText
			case day.Equal(mo.chosen):
				p.Fill(disc, t.Text.Alpha(0.12), disc.W/2)
			case d == hot:
				p.Fill(disc, t.Text.Alpha(0.06), disc.W/2)
			}
			span := ui.Span{Text: strconv.Itoa(d), Size: small, Weight: weight, Color: ink, Features: "tnum"}
			w, h := p.MeasureText(0, span)
			p.RichText(cr.X+(cr.W-w)/2, disc.Y+(disc.H-h)/2, 0, span)
			dots := min(len(marks), 3)
			dot := max(2, cell*0.1)
			x := cr.X + cr.W/2 - float32(dots)*dot*0.9 + dot*0.4
			for i := range dots {
				p.Fill(ui.Rect{X: x + float32(i)*dot*1.8, Y: cr.Y + cell*0.86, W: dot, H: dot}, mo.colorOf(t, marks[i]), dot/2)
			}
		}
	})
	if hot > 0 && box.Clicked() {
		return mo.first.AddDate(0, 0, hot-1), true
	}
	return time.Time{}, false
}

// colorOf returns the color of the i-th event.
func (mo *month) colorOf(t *ui.Theme, i int) ui.Color {
	if c := mo.events[i].Color; c != (ui.Color{}) {
		return c
	}
	return t.Accent
}

// tip lists the events on day d of the month, five at most.
func (mo *month) tip(d int, list []int) string {
	day := mo.first.AddDate(0, 0, d-1)
	var sb strings.Builder
	sb.WriteString(i18n.FormatDate(mo.locale, day, i18n.DateOptions{Skeleton: "MMMMEEEEd"}))
	for k, i := range list {
		if k == 5 {
			sb.WriteString("\n+" + strconv.Itoa(len(list)-5) + " more")
			break
		}
		sb.WriteString("\n• " + mo.events[i].Title)
	}
	return sb.String()
}
