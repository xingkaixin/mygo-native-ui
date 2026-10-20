// Package monthview shows the events of a month on a grid of its weeks,
// as macOS Calendar's month view does: the weeks from the first day of
// the week of the user's regional formats, the days of the months around
// muted and today's date in a circle of the accent; events of whole days
// and of several days as bars across them, going on into the weeks after,
// and other events as a dot of their color, their time and their title. A
// day that cannot show all its events ends in "+3 more", which lists
// them. Dragging an event moves it to another day, keeping its time and
// length; a click chooses a day, which Chosen returns.
//
//	if monthview.View(c, &app.calendar, &app.month, &app.events).Grow(1).Changed() {
//		app.save()
//	}
//	if day, ok := app.calendar.Chosen(); ok {
//		... // show the events of day
//	}
package monthview

import (
	"cmp"
	"fmt"
	"math"
	"slices"
	"strconv"
	"time"

	"github.com/egoist/mygo/i18n"
	"github.com/egoist/mygo/ui"
)

// Event is a stretch of time on the calendar. One from midnight to
// midnight, or over more than one day, shows as a bar across its days.
type Event struct {
	Title      string
	Start, End time.Time
	// Color colors the event; the zero Color is the accent.
	Color ui.Color
}

// State is what a month view keeps: the day chosen, the day whose events
// are listed, and the event dragged. Keep it in the app's state, one for
// each view.
type State struct {
	chosen  time.Time
	listed  time.Time
	listing bool
	// drag is the event dragged, its index plus one, by its piece in the
	// week-th week, from the grab-th day of the grid to the at-th.
	drag, week, grab, at int
}

// Chosen returns the day chosen, at its midnight, or false with none.
func (s *State) Chosen() (time.Time, bool) { return s.chosen, !s.chosen.IsZero() }

// View creates the month view of the month of *month, any day in it, and
// of *events, which it moves in place and reports moving as changes. Give
// it a size, or grow it in its parent.
//
// The buttons above go to the months before and after, and Today back to
// today's, choosing today. A click on a day chooses it; dragging an event
// moves it by whole days to the day it is dropped on.
func View(c *ui.Context, s *State, month *time.Time, events *[]Event) ui.Element {
	t := c.Theme()
	region := c.Region()
	locale := region.FormatLocale(c.Locale())
	if month.IsZero() {
		*month = c.Now()
	}
	loc := month.Location()
	now := c.Now().In(loc)
	today := day(now)
	c.After(today.AddDate(0, 0, 1).Sub(now))
	first := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, loc)
	start := first.AddDate(0, 0, -((int(first.Weekday()) - int(region.FirstWeekday) + 7) % 7))
	weeks := (between(start, first.AddDate(0, 1, 0)) + 6) / 7
	changed := false
	root := ui.Column(c)
	root.Children(func() {
		ui.Row(c).Padding(t.Space(3), t.Space(4)).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, i18n.FormatDate(locale, first, i18n.DateOptions{Skeleton: "yMMMM"})).Bold().FontSize(t.FontSize + 6).Grow(1)
			if ui.Button(c, "‹").Label("Previous month").Clicked() {
				*month = first.AddDate(0, -1, 0)
			}
			if ui.Button(c, "Today").Clicked() {
				*month, s.chosen = now, today
			}
			if ui.Button(c, "›").Label("Next month").Clicked() {
				*month = first.AddDate(0, 1, 0)
			}
		})
		names := i18n.Weekdays(locale, i18n.Abbreviated, true)
		ui.Row(c).Children(func() {
			for i := range 7 {
				ui.Text(c, names[(int(region.FirstWeekday)+i)%7]).Grow(1).Basis(0).Padding(t.Space(1.5), t.Space(2.5)).
					FontSize(t.FontSize - 1).TextColor(t.TextMuted)
			}
		})
		changed = s.grid(c, events, &sheet{start: start, weeks: weeks, month: first.Month(), today: today, locale: locale})
	})
	if changed {
		root.SetChanged()
	}
	return root
}

// sheet is how the weeks of a view are laid out: weeks rows from the day
// start, of days w wide and h high, the days of month bright; in each
// day, lanes lanes of events lane high from top.
type sheet struct {
	start, today time.Time
	month        time.Month
	weeks        int
	w, h         float32
	top, lane    float32
	lanes        int
	locale       string
}

// cell returns the box of the k-th day in the grid.
func (g *sheet) cell(k int) ui.Rect {
	return ui.Rect{X: float32(k%7) * g.w, Y: float32(k/7) * g.h, W: g.w, H: g.h}
}

// at returns the index of the day at x, y in the grid, or the nearest.
func (g *sheet) at(x, y float32) int {
	return min(max(int(y/g.h), 0), g.weeks-1)*7 + min(max(int(x/g.w), 0), 6)
}

// grid builds the days of g with the events on them, and reports whether
// one moved.
func (s *State) grid(c *ui.Context, events *[]Event, g *sheet) bool {
	t := c.Theme()
	box := ui.Box(c).Grow(1).Clip()
	b := box.LayoutBounds()
	g.w, g.h = b.W/7, b.H/float32(g.weeks)
	g.top, g.lane = t.Space(7.5), t.FontSize+t.Space(2)
	g.lanes = max(1, int((g.h-g.top-t.Space(1))/g.lane))
	es := *events
	shift, from, to := 0, 0, -1
	if s.drag > 0 && s.drag <= len(es) {
		shift = s.at - s.grab
		from, to, _ = days(es[s.drag-1], g.start)
	}
	chosen := s.chosen
	box.Draw(func(p *ui.Painter, r ui.Rect) {
		// The days are laid out in the box's size as of the last frame.
		if r.W != b.W || r.H != b.H {
			c.Invalidate()
		}
		for k := range g.weeks * 7 {
			cell := g.cell(k)
			cell.X, cell.Y = cell.X+r.X, cell.Y+r.Y
			switch d := g.start.AddDate(0, 0, k); {
			case d.Equal(chosen):
				p.Fill(cell, t.Accent.Alpha(0.1), 0)
			case d.Month() != g.month:
				p.Fill(cell, t.Text.Alpha(0.03), 0)
			}
		}
		line := t.Text.Alpha(0.08)
		for k := 1; k < 7; k++ {
			p.Fill(ui.Rect{X: r.X + float32(k)*g.w, Y: r.Y, W: 1, H: r.H}, line, 0)
		}
		for k := range g.weeks {
			p.Fill(ui.Rect{X: r.X, Y: r.Y + float32(k)*g.h, W: r.W, H: 1}, line, 0)
		}
	})
	box.DrawOver(func(p *ui.Painter, r ui.Rect) {
		if shift == 0 {
			return
		}
		for k := max(0, from+shift); k <= min(g.weeks*7-1, to+shift); k++ {
			cell := g.cell(k)
			cell.X, cell.Y = cell.X+r.X, cell.Y+r.Y
			p.Fill(cell, t.Accent.Alpha(0.16), 0)
		}
	})
	if b.W == 0 || b.H == 0 {
		return false
	}
	moved, listed := false, false
	box.Children(func() {
		for k := range g.weeks * 7 {
			s.day(c.Key(k), g, k)
		}
		for wk, pieces := range layout(es, g.start, g.weeks) {
			var need, hidden [7]int
			for _, p := range pieces {
				for d := p.from; d <= p.to; d++ {
					need[d] = max(need[d], p.lane+1)
				}
			}
			for _, p := range pieces {
				fits := p.lane < g.lanes-1
				if p.lane == g.lanes-1 {
					fits = slices.IndexFunc(need[p.from:p.to+1], func(n int) bool { return n > g.lanes }) < 0
				}
				if !fits {
					for d := p.from; d <= p.to; d++ {
						hidden[d]++
					}
					continue
				}
				if s.piece(c.Key([2]int{p.i, wk}), g, &es[p.i], p, wk) {
					moved = true
				}
			}
			for d, n := range hidden {
				if n > 0 && s.more(c.Key([2]int{-1, wk*7 + d}), g, es, pieces, wk, d, n) {
					listed = true
				}
			}
		}
	})
	if !listed {
		s.listing = false
	}
	return moved
}

// day builds the k-th day of g, its date in a circle of the accent today,
// which a click chooses.
func (s *State) day(c *ui.Context, g *sheet, k int) {
	t := c.Theme()
	d := g.start.AddDate(0, 0, k)
	r := g.cell(k)
	cell := ui.ButtonBase(c).Absolute().Left(r.X).Top(r.Y).Size(r.W, r.H).Justify(ui.Start).AlignItems(ui.Start).Padding(t.Space(1)).
		Label(i18n.FormatDate(g.locale, d, i18n.DateOptions{Skeleton: "yMMMMd"}))
	if cell.Clicked() {
		s.chosen = d
	}
	cell.Children(func() {
		number := strconv.Itoa(d.Day())
		if d.Day() == 1 {
			number = i18n.FormatDate(g.locale, d, i18n.DateOptions{Skeleton: "MMMd"})
		}
		size := t.Space(6)
		pill := ui.Row(c).Height(size).MinWidth(size).Padding(0, t.Space(1.5)).Radius(size / 2).Justify(ui.Center).AlignItems(ui.Center)
		pill.Children(func() {
			text := ui.Text(c, number).FontFeatures("tnum").NoWrap()
			switch {
			case d.Equal(g.today):
				pill.Background(t.Accent)
				text.TextColor(t.AccentText).Bold()
			case d.Month() != g.month:
				text.TextColor(t.TextMuted)
			}
		})
	})
}

// piece builds p, the piece of e in the wk-th week of g, which dragging
// moves to another day, and reports whether it moved.
func (s *State) piece(c *ui.Context, g *sheet, e *Event, p piece, wk int) bool {
	t := c.Theme()
	color := colorOf(*e, t)
	inset := t.Space(1)
	left, right := float32(0), float32(0)
	if p.head {
		left = inset
	}
	if p.tail {
		right = inset
	}
	x, y := float32(p.from)*g.w+left, float32(wk)*g.h+g.top+float32(p.lane)*g.lane
	box := ui.Row(c).Absolute().Left(x).Top(y).Size(float32(p.to-p.from+1)*g.w-left-right, g.lane-t.Space(0.5)).
		Padding(0, t.Space(1.5)).Gap(t.Space(1)).AlignItems(ui.Center).Role(ui.RoleButton).Label(e.Title)
	if p.bar {
		box.Background(color.Alpha(0.25)).Radius(left, right, right, left)
	}
	held := s.drag == p.i+1 && s.week == wk
	moved := false
	if _, _, ok := box.Dragged(); ok {
		px, py, _ := box.PointerPosition()
		at := g.at(x+px, y+py)
		if !held {
			s.drag, s.week, s.grab = p.i+1, wk, at
		}
		s.at = at
	} else if held {
		if k := s.at - s.grab; k != 0 {
			e.Start, e.End = e.Start.AddDate(0, 0, k), e.End.AddDate(0, 0, k)
			moved = true
		} else {
			s.chosen = g.start.AddDate(0, 0, s.at)
		}
		s.drag = 0
	}
	if s.drag == p.i+1 && s.at != s.grab {
		box.Opacity(0.4)
	}
	box.Children(func() { line(c, g, *e, p.bar, color) })
	return moved
}

// more builds the "+n more" of the d-th day of the wk-th week of g, whose
// pieces are pieces, which lists the day's events, and reports whether
// they are listed.
func (s *State) more(c *ui.Context, g *sheet, es []Event, pieces []piece, wk, d, n int) bool {
	t := c.Theme()
	k := wk*7 + d
	date := g.start.AddDate(0, 0, k)
	r := g.cell(k)
	open := s.listing && s.listed.Equal(date)
	more := ui.ButtonBase(c).Absolute().Left(r.X+t.Space(1)).Top(r.Y+g.top+float32(g.lanes-1)*g.lane).
		Height(g.lane-t.Space(0.5)).Padding(0, t.Space(1.5)).Radius(t.Space(1)).Label(fmt.Sprintf("%d more events", n))
	if more.Hovered() || open {
		more.Background(t.SurfaceHover)
	}
	if more.Clicked() {
		s.listing, s.listed = !open, date
		open = s.listing
	}
	more.Children(func() {
		ui.Text(c, fmt.Sprintf("+%d more", n)).FontSize(t.FontSize - 1).Bold().TextColor(t.TextMuted)
	})
	if !open {
		return false
	}
	ui.PopoverBase(c, more, &s.listing, func(panel ui.Element) {
		panel.Margin(t.Space(1), 0, 0, 0).Width(t.Space(64)).Padding(t.Space(3)).Gap(t.Space(1)).Radius(t.Space(2.5)).
			Background(t.Background).Border(1, t.Border).Shadow(0, 8, 24, 0, ui.RGB(0, 0, 0).Alpha(0.18))
		ui.Text(c, i18n.FormatDate(g.locale, date, i18n.DateOptions{Skeleton: "MMMMEEEEd"})).Bold().Padding(0, 0, t.Space(1))
		for _, p := range pieces {
			if p.from > d || p.to < d {
				continue
			}
			e := es[p.i]
			color := colorOf(e, t)
			row := ui.Row(c.Key(p.i)).Height(g.lane).Padding(0, t.Space(1.5)).Gap(t.Space(1)).AlignItems(ui.Center).Radius(t.Space(1))
			if p.bar {
				row.Background(color.Alpha(0.25))
			}
			row.Children(func() { line(c, g, e, p.bar, color) })
		}
	})
	return true
}

// line builds what an event shows on its line: a bar's title, or a dot of
// color, the time it starts and its title.
func line(c *ui.Context, g *sheet, e Event, bar bool, color ui.Color) {
	t := c.Theme()
	small := t.FontSize - 1
	if !bar {
		d := t.Space(1.5)
		ui.Box(c).Size(d, d).Radius(d / 2).Background(color).Shrink(0)
		when := i18n.FormatDate(g.locale, e.Start.In(g.start.Location()), i18n.DateOptions{Skeleton: "jm"})
		ui.Text(c, when).FontSize(small).FontFeatures("tnum").TextColor(t.TextMuted).NoWrap()
	}
	ui.Text(c, e.Title).FontSize(small).SingleLine().Shrink(1)
}

func colorOf(e Event, t *ui.Theme) ui.Color {
	if e.Color == (ui.Color{}) {
		return t.Accent
	}
	return e.Color
}

// piece is the part of an event in a week: the event's index, the days of
// the week it covers, from 0, whether it is a bar, whether the event
// starts and ends in them, and its lane, from the top.
type piece struct {
	i          int
	from, to   int
	bar        bool
	head, tail bool
	lane       int
}

// layout lays events out over weeks weeks from start: the pieces of each
// week, the bars first, those starting earlier and lasting longer before,
// then the other events by their time, each in the top lane free on all
// its days.
func layout(events []Event, start time.Time, weeks int) [][]piece {
	type span struct{ i, a, b int }
	var bars, dots []span
	for i, e := range events {
		a, b, bar := days(e, start)
		switch {
		case b < 0 || a >= weeks*7:
		case bar:
			bars = append(bars, span{i, a, b})
		default:
			dots = append(dots, span{i, a, b})
		}
	}
	slices.SortStableFunc(bars, func(x, y span) int { return cmp.Or(cmp.Compare(x.a, y.a), cmp.Compare(y.b-y.a, x.b-x.a)) })
	slices.SortStableFunc(dots, func(x, y span) int { return events[x.i].Start.Compare(events[y.i].Start) })
	out := make([][]piece, weeks)
	for wk := range out {
		var used [][7]bool
		take := func(from, to int) int {
			for l := 0; ; l++ {
				if l == len(used) {
					used = append(used, [7]bool{})
				}
				if !slices.Contains(used[l][from:to+1], true) {
					for d := from; d <= to; d++ {
						used[l][d] = true
					}
					return l
				}
			}
		}
		lo, hi := wk*7, wk*7+6
		for _, sp := range bars {
			from, to := max(sp.a, lo), min(sp.b, hi)
			if from <= to {
				out[wk] = append(out[wk], piece{sp.i, from - lo, to - lo, true, sp.a == from, sp.b == to, take(from-lo, to-lo)})
			}
		}
		for _, sp := range dots {
			if sp.a >= lo && sp.a <= hi {
				d := sp.a - lo
				out[wk] = append(out[wk], piece{sp.i, d, d, false, true, true, take(d, d)})
			}
		}
	}
	return out
}

// days returns the first and last days of e as days after start, and
// whether it shows as a bar: over more than one day, or from midnight to
// midnight.
func days(e Event, start time.Time) (first, last int, bar bool) {
	loc := start.Location()
	from, to := e.Start.In(loc), e.End.In(loc)
	a, b := day(from), day(from)
	if to.After(from) {
		b = day(to.Add(-time.Nanosecond))
	}
	return between(start, a), between(start, b), b.After(a) || from.Equal(a) && to.Equal(a.AddDate(0, 0, 1))
}

// day returns the midnight of t's day.
func day(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}

// between returns how many days b is after a, both midnights.
func between(a, b time.Time) int { return int(math.Round(b.Sub(a).Hours() / 24)) }
