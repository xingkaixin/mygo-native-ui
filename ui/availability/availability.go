// Package availability finds a time when a group is free, as When2meet
// does: the days across and slots of time down them, 30 minutes each
// unless the app says otherwise, the whole hours on the left in the
// user's regional time format and the days on top.
//
// On my grid, pressing and dragging paints a rectangle of slots: free
// when the first slot was not, and not free when it was. The arrows move
// a focus cell and Space toggles it. Changed reports a change in the
// frame it happens, the slots the app keeps already changed.
//
// The group grid shades each slot by how many people are free then, from
// none to all, with a legend below ("0/6 available … 6/6 available"), and
// outlines the slots the most people are free. Hovering a slot shows who
// is free then and who is not, and hovering a person in the list of
// people lights the slots they are free.
//
//	g := availability.Grid{Days: week, From: 9, To: 18}
//	if availability.Mine(c, &app.group, g, &app.mine).Changed() {
//		app.save()
//	}
//	everyone := append([]availability.Person{{Name: "You", Free: app.mine}}, app.others...)
//	availability.Group(c, &app.group, g, everyone)
//	availability.People(c, &app.group, g, everyone)
package availability

import (
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo/i18n"
	"github.com/egoist/mygo/ui"
)

// Grid is the times a group picks from: the days across, and down each
// the slots from the hour From to the hour To, Step long.
type Grid struct {
	// Days are the columns, each the day of a time. The grid shows times
	// in the location of the first.
	Days []time.Time
	// From and To are the hours the slots start and end at, as 9 and 18.
	From, To int
	// Step is how long a slot is; 0 is 30 minutes.
	Step time.Duration
}

// Person is someone in the group, by name, and the slots they are free,
// each by the time it starts.
type Person struct {
	Name string
	Free []time.Time
}

// State is what a group's grids keep: the press painting my grid, its
// focus cell, and the person the pointer is on in the list of people.
// Keep it in the app's state, one for each group, and give it to its
// grids and its list alike.
type State struct {
	// held is whether the pointer held my grid in the last frame, so that
	// a press starts painting only in the frame it begins.
	held     bool
	painting bool
	// from is the cell a paint began on and to the one it is over; adding
	// is whether it frees the slots, and before is my slots as it began,
	// which each frame paints the rectangle over anew.
	from, to cell
	adding   bool
	before   []time.Time
	focus    cell
	// keyed is whether the keys moved the focus cell since the pointer
	// last pressed, which shows it.
	keyed bool
	// person is the index of the person the pointer is on in the list,
	// plus one, so that the zero State has none.
	person int
}

// cell is a slot of the grid by its column and its row.
type cell struct{ day, row int }

// Mine creates my grid of g, the slots I am free in *free, each by the
// time it starts, which it sets to a new slice, in order, as I paint or
// toggle them. Give it a width, or stretch it in its parent; its height
// follows the rows.
func Mine(c *ui.Context, s *State, g Grid, free *[]time.Time) ui.Element {
	t := c.Theme()
	f := newFrame(c, g)
	box := ui.Box(c).Height(f.height).MinWidth(f.minWidth(t)).Focusable().FocusRing(false).
		Cursor(ui.CursorPointer).Label("Your availability")
	f.fit(box.LayoutBounds().W)
	if len(f.days) == 0 || f.rows == 0 {
		return box
	}
	s.focus = cell{max(0, min(len(f.days)-1, s.focus.day)), max(0, min(f.rows-1, s.focus.row))}
	changed := false
	x, y, over := box.PointerPosition()
	_, _, held := box.Dragged()
	switch {
	case held && !s.held:
		if at, ok := f.at(x, y); ok && over {
			s.painting, s.from, s.to, s.focus, s.keyed = true, at, at, at, false
			s.adding = !f.mark(*free)[f.index(at)]
			s.before = slices.Clone(*free)
			changed = s.paint(f.slots, free)
		}
	case held && s.painting:
		if at := f.nearest(x, y); at != s.to {
			s.to, s.focus = at, at
			changed = s.paint(f.slots, free)
		}
	case !held:
		s.painting, s.before = false, nil
	}
	s.held = held
	for _, k := range []struct {
		key        ui.Key
		days, rows int
	}{{ui.KeyLeft, -1, 0}, {ui.KeyRight, 1, 0}, {ui.KeyUp, 0, -1}, {ui.KeyDown, 0, 1}} {
		if box.Shortcut(0, k.key) {
			s.focus = cell{max(0, min(len(f.days)-1, s.focus.day+k.days)), max(0, min(f.rows-1, s.focus.row+k.rows))}
			s.keyed = true
		}
	}
	if !s.painting && box.Shortcut(0, ui.KeySpace) {
		s.from, s.to, s.adding, s.before, s.keyed = s.focus, s.focus, !f.mark(*free)[f.index(s.focus)], *free, true
		changed = s.paint(f.slots, free) || changed
		s.before = nil
	}
	marks := f.mark(*free)
	focus := f.index(s.focus)
	state := "not free"
	if marks[focus] {
		state = "free"
	}
	box.Value(f.describe(focus) + ", " + state)
	hover, hovered := f.at(x, y)
	hovered = hovered && over && !s.painting
	ring := box.Focused() && (box.FocusVisible() || s.keyed)
	on, off := shade(1, 1, t), shade(0, 1, t)
	box.Draw(func(p *ui.Painter, r ui.Rect) {
		f.draw(p, r, t, func(k int) ui.Color {
			if marks[k] {
				return on
			}
			return off
		})
		if s.painting {
			a, b := f.rect(r, f.index(s.from)), f.rect(r, f.index(s.to))
			x0, y0 := min(a.X, b.X), min(a.Y, b.Y)
			p.Stroke(ui.Rect{X: x0, Y: y0, W: max(a.X, b.X) + a.W - x0, H: max(a.Y, b.Y) + a.H - y0}, t.Text.Alpha(0.6), 3, 1.5)
		}
		if hovered {
			p.Stroke(f.rect(r, f.index(hover)), t.Text.Alpha(0.45), 2, 1.5)
		}
		if ring {
			p.FocusRing(f.rect(r, focus), [4]float32{2, 2, 2, 2})
		}
	})
	if changed {
		box.SetChanged()
	}
	return box
}

// paint sets *free to my slots as the paint began, with the rectangle
// from its first cell to the one it is over freed or cleared, and reports
// whether that changed them.
func (s *State) paint(sl slots, free *[]time.Time) bool {
	d0, d1 := min(s.from.day, s.to.day), max(s.from.day, s.to.day)
	r0, r1 := min(s.from.row, s.to.row), max(s.from.row, s.to.row)
	var out []time.Time
	for _, tm := range s.before {
		if k, ok := sl.cell(tm); ok && k/sl.rows >= d0 && k/sl.rows <= d1 && k%sl.rows >= r0 && k%sl.rows <= r1 {
			continue
		}
		out = append(out, tm)
	}
	if s.adding {
		for d := d0; d <= d1; d++ {
			for r := r0; r <= r1; r++ {
				out = append(out, sl.start(d*sl.rows+r))
			}
		}
	}
	slices.SortFunc(out, time.Time.Compare)
	if slices.EqualFunc(out, *free, time.Time.Equal) {
		return false
	}
	*free = out
	return true
}

// Group creates the group grid of g, each slot shaded by how many of
// people are free then, with its legend below. Give it a width, or
// stretch it in its parent; its height follows the rows.
func Group(c *ui.Context, s *State, g Grid, people []Person) ui.Element {
	t := c.Theme()
	f := newFrame(c, g)
	n := len(people)
	marks := make([][]bool, n)
	counts := make([]int, len(f.days)*f.rows)
	for i, p := range people {
		marks[i] = f.mark(p.Free)
		for k, on := range marks[i] {
			if on {
				counts[k]++
			}
		}
	}
	best := 0
	for _, k := range counts {
		best = max(best, k)
	}
	view := ui.Column(c).Gap(t.Space(3))
	view.Children(func() {
		box := ui.Box(c).Height(f.height).MinWidth(f.minWidth(t)).AlignSelf(ui.Stretch).Label("Group availability")
		f.fit(box.LayoutBounds().W)
		x, y, over := box.PointerPosition()
		at, hovered := f.at(x, y)
		hovered = hovered && over
		k := f.index(at)
		box.Draw(func(p *ui.Painter, r ui.Rect) {
			f.draw(p, r, t, func(k int) ui.Color { return shade(counts[k], n, t) })
			// The person the pointer is on in the list is read as the
			// grid paints, as the list may be built after the grid.
			if i := s.person - 1; i >= 0 && i < n {
				wash := t.Background.Alpha(0.86)
				for k, on := range marks[i] {
					if !on {
						p.Fill(f.rect(r, k), wash, 0)
					}
				}
			}
			if best > 0 {
				f.outline(p, r, t, func(k int) bool { return counts[k] == best })
			}
			if hovered {
				p.Stroke(f.rect(r, k), t.Text.Alpha(0.6), 2, 1.5)
			}
		})
		if hovered {
			box.Children(func() {
				rect := f.rect(ui.Rect{}, k)
				anchor := ui.Box(c).Absolute().Left(rect.X).Top(rect.Y).Size(rect.W, rect.H).PassThrough()
				card(c, anchor, &f, k, people, marks, counts[k])
			})
		}
		if n > 0 {
			legend(c, n, best, f.left)
		}
	})
	return view
}

// card shows by anchor, on cell k, when its slot is, how many are free
// then, and who of people is and who is not.
func card(c *ui.Context, anchor ui.Element, f *frame, k int, people []Person, marks [][]bool, count int) {
	t := c.Theme()
	var yes, no []string
	for i, p := range people {
		if marks[i][k] {
			yes = append(yes, p.Name)
		} else {
			no = append(no, p.Name)
		}
	}
	ui.Overlay(c, func() {
		// The card and all in it let the pointer through to the grid,
		// which would otherwise lose the hover and the card.
		tip := ui.Column(c).Absolute().PassThrough().Role(ui.RoleTooltip).
			AttachTo(anchor, ui.AnchorRight, ui.AnchorLeft).Margin(0, t.Space(2), 0, t.Space(2)).
			Padding(t.Space(2.5), t.Space(3)).Gap(t.Space(2.5)).Radius(t.Space(2)).MinWidth(t.Space(48)).
			Background(t.Background).Border(1, t.Border).Shadow(0, 4, 16, 0, ui.RGBA(0, 0, 0, 0.18))
		tip.Children(func() {
			ui.Column(c).PassThrough().Gap(t.Space(0.5)).Children(func() {
				ui.Text(c, f.when(k)).PassThrough().Bold().NoWrap()
				ui.Text(c, strconv.Itoa(count)+"/"+strconv.Itoa(len(people))+" available").PassThrough().
					FontSize(t.FontSize - 1).FontFeatures("tnum").TextColor(t.TextMuted)
			})
			ui.Row(c).PassThrough().Gap(t.Space(6)).AlignItems(ui.Start).Children(func() {
				names(c, "Available", yes, t.Text, shade(1, 1, t))
				names(c, "Unavailable", no, t.TextMuted, t.Text.Alpha(0.2))
			})
		})
	})
}

// names lists people under a heading with a dot of mark, in color, or a
// dash for none.
func names(c *ui.Context, heading string, people []string, color, mark ui.Color) {
	t := c.Theme()
	ui.Column(c.Key(heading)).PassThrough().Gap(t.Space(1)).Children(func() {
		ui.Row(c).PassThrough().Gap(t.Space(1.5)).AlignItems(ui.Center).Children(func() {
			ui.Box(c).PassThrough().Size(t.Space(2), t.Space(2)).Radius(t.Space(1)).Background(mark)
			ui.Text(c, heading).PassThrough().FontSize(t.FontSize - 2).Bold().TextColor(t.TextMuted)
		})
		if len(people) == 0 {
			ui.Text(c, "–").PassThrough().TextColor(t.TextMuted)
		}
		for i, name := range people {
			ui.Text(c.Key(i), name).PassThrough().TextColor(color).NoWrap()
		}
	})
}

// legend shows the shades from none of n people free to all, under the
// cells, left from the grid's edge, and the outline of the best slots,
// best of n free.
func legend(c *ui.Context, n, best int, left float32) {
	t := c.Theme()
	small := t.FontSize - 2
	ui.Row(c).PaddingStart(left).Gap(t.Space(2)).AlignItems(ui.Center).Wrap().Children(func() {
		ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "0/"+strconv.Itoa(n)+" available").FontSize(small).FontFeatures("tnum").TextColor(t.TextMuted).NoWrap()
			ui.Box(c).Size(t.Space(28), t.Space(3)).Draw(func(p *ui.Painter, r ui.Rect) {
				w := (r.W - float32(n)) / float32(n+1)
				for k := range n + 1 {
					p.Fill(ui.Rect{X: r.X + float32(k)*(w+1), Y: r.Y, W: w, H: r.H}, shade(k, n, t), 2)
				}
			})
			ui.Text(c, strconv.Itoa(n)+"/"+strconv.Itoa(n)+" available").FontSize(small).FontFeatures("tnum").TextColor(t.TextMuted).NoWrap()
		})
		if best > 0 {
			ui.Row(c).MarginStart(t.Space(3)).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
				ui.Box(c).Size(t.Space(3.5), t.Space(3)).Radius(2).Background(shade(best, n, t)).Border(2, t.Text)
				ui.Text(c, "Most available, "+strconv.Itoa(best)+"/"+strconv.Itoa(n)).FontSize(small).FontFeatures("tnum").TextColor(t.TextMuted).NoWrap()
			})
		}
	})
}

// People creates the list of people, each with how long they are free on
// g. Hovering one lights the slots they are free on the group grid.
func People(c *ui.Context, s *State, g Grid, people []Person) ui.Element {
	t := c.Theme()
	sl := newSlots(g)
	locale := c.Region().FormatLocale(c.Locale())
	s.person = 0
	return ui.Column(c).Gap(t.Space(0.5)).Children(func() {
		for i, p := range people {
			free := 0
			for _, on := range sl.mark(p.Free) {
				if on {
					free++
				}
			}
			hours := i18n.FormatNumber(locale, float64(free*sl.step)/60, i18n.NumberOptions{MaxFractionDigits: 2}) + " h"
			row := ui.Row(c.Key(i)).Gap(t.Space(2.5)).Padding(t.Space(1.5), t.Space(2)).Radius(t.Space(1.5)).
				AlignItems(ui.Center).Label(p.Name).Description(hours + " free")
			if row.Hovered() {
				s.person = i + 1
				row.Background(t.SurfaceHover)
			}
			row.Children(func() {
				d := t.Space(7)
				ui.Row(c).Size(d, d).Shrink(0).Radius(d / 2).Background(hue(i)).Justify(ui.Center).AlignItems(ui.Center).Children(func() {
					ui.Text(c, initial(p.Name)).FontSize(t.FontSize - 2).Bold().TextColor(ui.RGB(255, 255, 255))
				})
				ui.Text(c, p.Name).Grow(1).Shrink(1).SingleLine()
				ui.Text(c, hours).FontSize(t.FontSize - 1).FontFeatures("tnum").TextColor(t.TextMuted).NoWrap()
			})
		}
	})
}

// initial returns the first letter of name, in upper case.
func initial(name string) string {
	for _, r := range name {
		return strings.ToUpper(string(r))
	}
	return ""
}

// hue returns the color of the i-th person.
func hue(i int) ui.Color {
	return ui.Oklch(0.6, 0.13, float32((255+i*67)%360))
}

// shade returns the fill of a slot k of n people are free in, from a
// light gray at none to a deep green at all, lighter at all in a dark
// theme.
func shade(k, n int, t *ui.Theme) ui.Color {
	if k <= 0 || n <= 0 {
		return t.Text.Alpha(0.07)
	}
	f := float32(k) / float32(n)
	if t.Dark {
		return ui.Oklch(0.34+0.38*f, 0.05+0.1*f, 152)
	}
	return ui.Oklch(0.92-0.4*f, 0.05+0.11*f, 152)
}

// slots maps the times of a grid to its cells, each day's rows one after
// the other, and back.
type slots struct {
	// days are the midnights of the days, in the location of the first.
	days []time.Time
	// from is the minute of the day the first row starts at, and step
	// how many minutes a row lasts.
	from, step int
	rows       int
}

// newSlots maps the times of g.
func newSlots(g Grid) slots {
	step := int(g.Step / time.Minute)
	if step <= 0 {
		step = 30
	}
	sl := slots{from: g.From * 60, step: step, rows: max(0, (g.To-g.From)*60/step)}
	for _, d := range g.Days {
		sl.days = append(sl.days, time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, g.Days[0].Location()))
	}
	return sl
}

// cell returns the index of the cell tm is in, or false when it is in
// none.
func (sl slots) cell(tm time.Time) (int, bool) {
	if len(sl.days) == 0 {
		return 0, false
	}
	tm = tm.In(sl.days[0].Location())
	y, m, dd := tm.Date()
	minute := tm.Hour()*60 + tm.Minute() - sl.from
	if minute < 0 || minute/sl.step >= sl.rows {
		return 0, false
	}
	for d, day := range sl.days {
		if day.Day() == dd && day.Month() == m && day.Year() == y {
			return d*sl.rows + minute/sl.step, true
		}
	}
	return 0, false
}

// start returns when the slot of cell k starts.
func (sl slots) start(k int) time.Time {
	day := sl.days[k/sl.rows]
	return time.Date(day.Year(), day.Month(), day.Day(), 0, sl.from+k%sl.rows*sl.step, 0, 0, day.Location())
}

// mark returns, for each cell, whether one of times is in it.
func (sl slots) mark(times []time.Time) []bool {
	marks := make([]bool, len(sl.days)*sl.rows)
	for _, tm := range times {
		if k, ok := sl.cell(tm); ok {
			marks[k] = true
		}
	}
	return marks
}

// index returns the index of cell at.
func (sl slots) index(at cell) int { return at.day*sl.rows + at.row }

// frame is a grid laid out in a box: the hours in a gutter left wide,
// the days in a row top high, and the cells w by h, the days gap apart.
// The box is height high, the last hour's label hanging below the cells.
type frame struct {
	slots
	locale                       string
	left, top, w, h, gap, height float32
	// hours are the labels of the whole hours from the minute of the
	// day first on, and weekdays, dates and numbers those of the days, as
	// "Tue", "Oct 21" and "21", the last where the date does not fit.
	hours                    []string
	first                    int
	weekdays, dates, numbers []string
	small, dayH, weekdayH    float32
}

// newFrame lays g out in the theme of c, the cells' width left for fit.
func newFrame(c *ui.Context, g Grid) frame {
	t := c.Theme()
	f := frame{slots: newSlots(g), locale: c.Region().FormatLocale(c.Locale()), h: t.Space(5), gap: t.Space(1), small: t.FontSize - 2}
	f.first = (f.from + 59) / 60 * 60
	widest := float32(0)
	for h := f.first / 60; h*60 <= f.from+f.rows*f.step; h++ {
		label := i18n.FormatDate(f.locale, time.Date(2000, 1, 1, h, 0, 0, 0, time.UTC), i18n.DateOptions{Skeleton: "j"})
		w, _ := c.MeasureText(0, ui.Span{Text: label, Size: f.small})
		widest = max(widest, w)
		f.hours = append(f.hours, label)
	}
	for _, d := range f.days {
		f.weekdays = append(f.weekdays, i18n.FormatDate(f.locale, d, i18n.DateOptions{Skeleton: "E"}))
		f.dates = append(f.dates, i18n.FormatDate(f.locale, d, i18n.DateOptions{Skeleton: "MMMd"}))
		f.numbers = append(f.numbers, i18n.FormatDate(f.locale, d, i18n.DateOptions{Skeleton: "d"}))
	}
	_, f.weekdayH = c.MeasureText(0, ui.Span{Text: "M", Size: f.small})
	_, f.dayH = c.MeasureText(0, ui.Span{Text: "M", Size: t.FontSize - 1, Weight: 600})
	_, labelH := c.MeasureText(0, ui.Span{Text: "0", Size: f.small})
	f.left = widest + t.Space(2.5)
	f.top = f.weekdayH + f.dayH + t.Space(2.5)
	f.height = f.top + float32(f.rows)*f.h + labelH/2
	return f
}

// minWidth returns how narrow the grid can be, its cells still wide
// enough to aim at.
func (f *frame) minWidth(t *ui.Theme) float32 {
	return f.left + float32(len(f.days))*(t.Space(6)+f.gap)
}

// fit sizes the cells to fill a box width wide.
func (f *frame) fit(width float32) {
	if n := float32(len(f.days)); n > 0 {
		f.w = max(1, (width-f.left-f.gap*(n-1))/n)
	}
}

// rect returns the box of cell k in the grid's box r.
func (f *frame) rect(r ui.Rect, k int) ui.Rect {
	d, row := k/f.rows, k%f.rows
	return ui.Rect{X: r.X + f.left + float32(d)*(f.w+f.gap), Y: r.Y + f.top + float32(row)*f.h, W: f.w, H: f.h}
}

// at returns the cell at x, y in the grid's box, or false where there is
// none, a gap between days going with the day to its left.
func (f *frame) at(x, y float32) (cell, bool) {
	if x < f.left || y < f.top || f.w <= 0 {
		return cell{}, false
	}
	at := cell{int((x - f.left) / (f.w + f.gap)), int((y - f.top) / f.h)}
	return at, at.day < len(f.days) && at.row < f.rows
}

// nearest returns the cell at x, y, or the cell nearest it when it is
// outside the cells.
func (f *frame) nearest(x, y float32) cell {
	day := int(max(0, x-f.left) / (f.w + f.gap))
	row := int(max(0, y-f.top) / f.h)
	return cell{min(len(f.days)-1, day), min(f.rows-1, row)}
}

// when writes when the slot of cell k is: "Tue, Oct 21 · 10:00 – 10:30".
func (f *frame) when(k int) string {
	start := f.start(k)
	end := start.Add(time.Duration(f.step) * time.Minute)
	jm := i18n.DateOptions{Skeleton: "jm"}
	return i18n.FormatDate(f.locale, start, i18n.DateOptions{Skeleton: "MMMEd"}) + " · " +
		i18n.FormatDate(f.locale, start, jm) + " – " + i18n.FormatDate(f.locale, end, jm)
}

// describe writes when the slot of cell k starts, for screen readers.
func (f *frame) describe(k int) string {
	return i18n.FormatDate(f.locale, f.start(k), i18n.DateOptions{Skeleton: "MMMMEEEEdjm"})
}

// draw paints the grid in its box r: the days on top, the hours beside,
// and each day's cells, cell k in fill(k), with lines between the slots,
// stronger at whole hours.
func (f *frame) draw(p *ui.Painter, r ui.Rect, t *ui.Theme, fill func(k int) ui.Color) {
	for d := range f.days {
		x := r.X + f.left + float32(d)*(f.w+f.gap)
		center := func(s ui.Span, y float32) {
			w, _ := p.MeasureText(0, s)
			p.RichText(x+(f.w-w)/2, y, 0, s)
		}
		center(ui.Span{Text: f.weekdays[d], Size: f.small, Weight: 500, Color: t.TextMuted}, r.Y)
		date := ui.Span{Text: f.dates[d], Size: t.FontSize - 1, Weight: 600, Color: t.Text}
		if w, _ := p.MeasureText(0, date); w > f.w-t.Space(1) {
			date.Text = f.numbers[d]
		}
		center(date, r.Y+f.weekdayH)
	}
	for i, label := range f.hours {
		y := r.Y + f.top + float32(f.first+i*60-f.from)/float32(f.step)*f.h
		s := ui.Span{Text: label, Size: f.small, Color: t.TextMuted, Features: "tnum"}
		w, th := p.MeasureText(0, s)
		p.RichText(r.X+f.left-w-t.Space(2), y-th/2, 0, s)
	}
	for d := range f.days {
		col := f.rect(r, d*f.rows)
		col.H = float32(f.rows) * f.h
		p.Clip(col, t.Space(1), func() {
			for row := range f.rows {
				k := d*f.rows + row
				p.Fill(f.rect(r, k), fill(k), 0)
			}
			for row := 1; row < f.rows; row++ {
				line := t.Background.Alpha(0.55)
				if (f.from+row*f.step)%60 == 0 {
					line = t.Background
				}
				p.Fill(ui.Rect{X: col.X, Y: col.Y + float32(row)*f.h - 0.5, W: col.W, H: 1}, line, 0)
			}
		})
	}
}

// outline strokes around each run of cells down a day that are in it.
func (f *frame) outline(p *ui.Painter, r ui.Rect, t *ui.Theme, in func(k int) bool) {
	for d := range f.days {
		for row := 0; row < f.rows; row++ {
			if !in(d*f.rows + row) {
				continue
			}
			end := row
			for end+1 < f.rows && in(d*f.rows+end+1) {
				end++
			}
			a := f.rect(r, d*f.rows+row)
			a.H = float32(end-row+1) * f.h
			p.Stroke(ui.Rect{X: a.X + 1, Y: a.Y + 1, W: a.W - 2, H: a.H - 2}, t.Text, 3, 2)
			row = end
		}
	}
}
