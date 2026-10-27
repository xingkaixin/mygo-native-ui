// Package recurrence edits how an event repeats, as Calendar's Custom
// repeat does: daily, weekly, monthly or yearly, every so many of them;
// on days of the week chosen as toggles; on days of the month chosen in
// a grid, or on the second Tuesday or the last weekday; in months chosen
// in a grid; and ending never, on a day or after so many times. Under it
// show what the rule means, "Every 2 weeks on Monday and Thursday", its
// next times, and the rule as RFC 5545's RRULE.
//
//	recurrence.View(c, &app.rule, app.start)
//	...
//	rule, err := recurrence.Parse("FREQ=MONTHLY;BYDAY=-1FR")
//	next := rule.Times(app.start, time.Now(), 5)
//
// Changing the frequency starts its days afresh from the start's.
package recurrence

import (
	"errors"
	"fmt"
	"iter"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
)

// Freq is how often a rule repeats.
type Freq int

const (
	Daily Freq = iota
	Weekly
	Monthly
	Yearly
)

// Rule is how an event repeats from its start, as an RRULE says it. The
// zero Rule repeats every day.
type Rule struct {
	Freq Freq
	// Interval is how many days, weeks, months or years apart the times
	// are; 0 is 1.
	Interval int
	// Days are the days of the week it repeats on: weekly, those of each
	// week, none the start's; monthly and yearly with Pos, those of which
	// Pos picks one.
	Days []time.Weekday
	// Pos picks, monthly and yearly, the Pos-th of the month's Days, from
	// 1, or from its end, -1 the last. 0 repeats on MonthDays instead.
	Pos int
	// MonthDays are the days of the month it repeats on, monthly and
	// yearly without Pos; none the start's. A month without the day skips
	// it.
	MonthDays []int
	// Months are the months it repeats in, yearly; none the start's.
	Months []time.Month
	// Count ends it after so many times, the start the first; 0 never.
	Count int
	// Until ends it after that day; the zero time never.
	Until time.Time
}

var (
	freqNames = []string{"DAILY", "WEEKLY", "MONTHLY", "YEARLY"}
	dayCodes  = []string{"SU", "MO", "TU", "WE", "TH", "FR", "SA"}
)

// String returns the rule as an RRULE's value, as
// "FREQ=WEEKLY;INTERVAL=2;BYDAY=MO,TH;COUNT=10".
func (r Rule) String() string {
	parts := []string{"FREQ=" + freqNames[r.Freq]}
	if r.Interval > 1 {
		parts = append(parts, "INTERVAL="+strconv.Itoa(r.Interval))
	}
	if r.Freq == Yearly && len(r.Months) > 0 {
		parts = append(parts, "BYMONTH="+join(r.Months, func(m time.Month) string { return strconv.Itoa(int(m)) }))
	}
	code := func(d time.Weekday) string { return dayCodes[d] }
	switch {
	case r.Freq == Weekly && len(r.Days) > 0:
		parts = append(parts, "BYDAY="+join(weekOrder(r.Days, time.Monday), code))
	case r.Freq >= Monthly && r.Pos != 0 && len(r.Days) == 1:
		parts = append(parts, "BYDAY="+strconv.Itoa(r.Pos)+code(r.Days[0]))
	case r.Freq >= Monthly && r.Pos != 0 && len(r.Days) > 1:
		parts = append(parts, "BYDAY="+join(weekOrder(r.Days, time.Monday), code), "BYSETPOS="+strconv.Itoa(r.Pos))
	case r.Freq >= Monthly && len(r.MonthDays) > 0:
		parts = append(parts, "BYMONTHDAY="+join(sorted(r.MonthDays), strconv.Itoa))
	}
	if r.Count > 0 {
		parts = append(parts, "COUNT="+strconv.Itoa(r.Count))
	} else if !r.Until.IsZero() {
		parts = append(parts, "UNTIL="+r.Until.Format("20060102"))
	}
	return strings.Join(parts, ";")
}

func join[T any](items []T, text func(T) string) string {
	out := make([]string, len(items))
	for i, it := range items {
		out[i] = text(it)
	}
	return strings.Join(out, ",")
}

func sorted[T int | time.Month](items []T) []T {
	out := slices.Clone(items)
	slices.Sort(out)
	return slices.Compact(out)
}

// weekOrder returns days in the order of a week starting on first.
func weekOrder(days []time.Weekday, first time.Weekday) []time.Weekday {
	out := slices.Clone(days)
	slices.SortFunc(out, func(a, b time.Weekday) int { return (int(a-first)+7)%7 - (int(b-first)+7)%7 })
	return slices.Compact(out)
}

// Parse parses an RRULE's value, with "RRULE:" before it or not. It takes
// FREQ from DAILY to YEARLY, INTERVAL, BYDAY, BYSETPOS, BYMONTHDAY,
// BYMONTH, COUNT, UNTIL and WKST, which it ignores.
func Parse(s string) (Rule, error) {
	var r Rule
	s = strings.TrimPrefix(strings.TrimSpace(s), "RRULE:")
	freq, setpos, numbered := -1, 0, false
	for part := range strings.SplitSeq(s, ";") {
		key, value, ok := strings.Cut(part, "=")
		if !ok {
			return Rule{}, fmt.Errorf("recurrence: %q is not KEY=VALUE", part)
		}
		var err error
		switch strings.ToUpper(key) {
		case "FREQ":
			freq = slices.Index(freqNames, strings.ToUpper(value))
			if freq < 0 {
				return Rule{}, fmt.Errorf("recurrence: FREQ=%s is not supported", value)
			}
			r.Freq = Freq(freq)
		case "INTERVAL":
			r.Interval, err = positive(value)
		case "COUNT":
			r.Count, err = positive(value)
		case "BYSETPOS":
			setpos, err = strconv.Atoi(value)
		case "BYMONTHDAY":
			for v := range strings.SplitSeq(value, ",") {
				d, e := strconv.Atoi(v)
				if e != nil || d < 1 || d > 31 {
					return Rule{}, fmt.Errorf("recurrence: BYMONTHDAY %q is not a day from 1 to 31", v)
				}
				r.MonthDays = append(r.MonthDays, d)
			}
		case "BYMONTH":
			for v := range strings.SplitSeq(value, ",") {
				m, e := strconv.Atoi(v)
				if e != nil || m < 1 || m > 12 {
					return Rule{}, fmt.Errorf("recurrence: BYMONTH %q is not a month from 1 to 12", v)
				}
				r.Months = append(r.Months, time.Month(m))
			}
		case "BYDAY":
			for v := range strings.SplitSeq(strings.ToUpper(value), ",") {
				d := slices.Index(dayCodes, v[max(0, len(v)-2):])
				if d < 0 {
					return Rule{}, fmt.Errorf("recurrence: BYDAY %q is not a day of the week", v)
				}
				if n := v[:len(v)-2]; n != "" {
					numbered = true
					if r.Pos, err = strconv.Atoi(strings.TrimPrefix(n, "+")); err != nil {
						return Rule{}, fmt.Errorf("recurrence: BYDAY %q has no number before its day", v)
					}
				}
				r.Days = append(r.Days, time.Weekday(d))
			}
		case "UNTIL":
			r.Until, err = time.Parse("20060102", value[:min(8, len(value))])
		case "WKST":
		default:
			return Rule{}, fmt.Errorf("recurrence: %s is not supported", key)
		}
		if err != nil {
			return Rule{}, fmt.Errorf("recurrence: %s=%s: %w", key, value, err)
		}
	}
	if freq < 0 {
		return Rule{}, errors.New("recurrence: FREQ is missing")
	}
	if numbered && len(r.Days) > 1 {
		return Rule{}, errors.New("recurrence: BYDAY numbers more than one day")
	}
	if setpos != 0 {
		r.Pos = setpos
	}
	if r.Pos < -5 || r.Pos > 5 || r.Pos != 0 && r.Freq < Monthly {
		return Rule{}, fmt.Errorf("recurrence: position %d is not supported", r.Pos)
	}
	return r, nil
}

func positive(s string) (int, error) {
	n, err := strconv.Atoi(s)
	if err == nil && n < 1 {
		err = errors.New("not above 0")
	}
	return n, err
}

// Times returns the first n times on or after from of the event that
// starts at start: its day at the start's time of day.
func (r Rule) Times(start, from time.Time, n int) []time.Time {
	var out []time.Time
	for at := range r.all(start) {
		if len(out) == n {
			break
		}
		if !at.Before(from) {
			out = append(out, at)
		}
	}
	return out
}

// periods is how many days, weeks, months or years apart from the start
// the times stop being looked for, as for a rule on the 31st of February.
const periods = 50000

// all returns the times of the event that starts at start, the start the
// first, as RFC 5545's DTSTART.
func (r Rule) all(start time.Time) iter.Seq[time.Time] {
	return func(yield func(time.Time) bool) {
		count := 0
		emit := func(at time.Time) bool {
			if !r.Until.IsZero() && dayOf(at).After(dayOf(r.Until)) {
				return false
			}
			count++
			return yield(at) && (r.Count == 0 || count < r.Count)
		}
		if !emit(start) {
			return
		}
		step := max(1, r.Interval)
		for k := 0; k < periods; k += step {
			for _, at := range r.period(start, k) {
				if at.After(start) && !emit(at) {
					return
				}
			}
		}
	}
}

func dayOf(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// period returns the times, in order, of the period k days, weeks,
// months or years after the start's.
func (r Rule) period(start time.Time, k int) []time.Time {
	at := func(y int, m time.Month, d int) time.Time {
		return time.Date(y, m, d, start.Hour(), start.Minute(), start.Second(), 0, start.Location())
	}
	switch r.Freq {
	case Weekly:
		monday := start.AddDate(0, 0, -(int(start.Weekday())+6)%7+7*k)
		var out []time.Time
		for _, d := range weekOrder(r.days(start), time.Monday) {
			day := monday.AddDate(0, 0, (int(d)+6)%7)
			out = append(out, at(day.Year(), day.Month(), day.Day()))
		}
		return out
	case Monthly:
		first := time.Date(start.Year(), start.Month()+time.Month(k), 1, 0, 0, 0, 0, time.UTC)
		return r.inMonth(first.Year(), first.Month(), start, at)
	case Yearly:
		months := sorted(r.Months)
		if len(months) == 0 {
			months = []time.Month{start.Month()}
		}
		var out []time.Time
		for _, m := range months {
			out = append(out, r.inMonth(start.Year()+k, m, start, at)...)
		}
		return out
	}
	day := start.AddDate(0, 0, k)
	return []time.Time{at(day.Year(), day.Month(), day.Day())}
}

func (r Rule) days(start time.Time) []time.Weekday {
	if len(r.Days) == 0 {
		return []time.Weekday{start.Weekday()}
	}
	return r.Days
}

// inMonth returns the times of month m of year y: the Pos-th of its
// Days, or its MonthDays.
func (r Rule) inMonth(y int, m time.Month, start time.Time, at func(int, time.Month, int) time.Time) []time.Time {
	last := daysIn(y, m)
	if r.Pos != 0 {
		days := r.days(start)
		var match []int
		for d := 1; d <= last; d++ {
			if slices.Contains(days, time.Date(y, m, d, 0, 0, 0, 0, time.UTC).Weekday()) {
				match = append(match, d)
			}
		}
		i := r.Pos - 1
		if r.Pos < 0 {
			i = len(match) + r.Pos
		}
		if i < 0 || i >= len(match) {
			return nil
		}
		return []time.Time{at(y, m, match[i])}
	}
	days := sorted(r.MonthDays)
	if len(days) == 0 {
		days = []int{start.Day()}
	}
	var out []time.Time
	for _, d := range days {
		if d <= last {
			out = append(out, at(y, m, d))
		}
	}
	return out
}

func daysIn(y int, m time.Month) int { return time.Date(y, m+1, 0, 0, 0, 0, 0, time.UTC).Day() }

var (
	units     = []string{"day", "week", "month", "year"}
	ordinals  = []string{"first", "second", "third", "fourth", "fifth"}
	workweek  = []time.Weekday{time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday}
	weekend   = []time.Weekday{time.Saturday, time.Sunday}
	everyDay  = []time.Weekday{time.Sunday, time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday, time.Saturday}
	dayKinds  = []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "day", "weekday", "weekend day"}
	positions = []string{"first", "second", "third", "fourth", "fifth", "last"}
)

// Describe returns what the rule means for an event that starts at
// start, in words: "Every 2 weeks on Monday and Thursday, 10 times".
func (r Rule) Describe(start time.Time) string {
	n := max(1, r.Interval)
	weekdaysOnly := r.Freq == Weekly && n == 1 && same(r.days(start), workweek)
	var b strings.Builder
	switch {
	case weekdaysOnly:
		b.WriteString("Every weekday")
	case n == 1:
		b.WriteString("Every " + units[r.Freq])
	default:
		fmt.Fprintf(&b, "Every %d %ss", n, units[r.Freq])
	}
	switch {
	case r.Freq == Weekly && !weekdaysOnly:
		names := make([]string, 0, 7)
		for _, d := range weekOrder(r.days(start), time.Monday) {
			names = append(names, d.String())
		}
		b.WriteString(" on " + list(names))
	case r.Freq == Monthly && r.Pos != 0:
		b.WriteString(" on the " + position(r.Pos) + " " + dayKind(r.days(start)))
	case r.Freq == Monthly:
		b.WriteString(" on the " + list(r.monthDays(start)))
	case r.Freq == Yearly:
		months := sorted(r.Months)
		if len(months) == 0 {
			months = []time.Month{start.Month()}
		}
		names := make([]string, len(months))
		for i, m := range months {
			names[i] = m.String()
		}
		if r.Pos != 0 {
			b.WriteString(" on the " + position(r.Pos) + " " + dayKind(r.days(start)) + " of " + list(names))
		} else {
			b.WriteString(" in " + list(names) + " on the " + list(r.monthDays(start)))
		}
	}
	switch {
	case r.Count == 1:
		b.WriteString(", once")
	case r.Count > 1:
		fmt.Fprintf(&b, ", %d times", r.Count)
	case !r.Until.IsZero():
		b.WriteString(", until " + r.Until.Format("Jan 2, 2006"))
	}
	return b.String()
}

func (r Rule) monthDays(start time.Time) []string {
	days := sorted(r.MonthDays)
	if len(days) == 0 {
		days = []int{start.Day()}
	}
	out := make([]string, len(days))
	for i, d := range days {
		out[i] = ordinal(d)
	}
	return out
}

func same(a, b []time.Weekday) bool {
	return slices.Equal(weekOrder(a, time.Sunday), weekOrder(b, time.Sunday))
}

func position(pos int) string {
	if pos < 0 {
		return "last"
	}
	return ordinals[pos-1]
}

// dayKind names days as the menu of "On the" does: one day's name, or
// "day", "weekday" or "weekend day".
func dayKind(days []time.Weekday) string {
	switch {
	case same(days, everyDay):
		return "day"
	case same(days, workweek):
		return "weekday"
	case same(days, weekend):
		return "weekend day"
	}
	names := make([]string, len(days))
	for i, d := range weekOrder(days, time.Monday) {
		names[i] = d.String()
	}
	return list(names)
}

func ordinal(n int) string {
	suffix := "th"
	switch {
	case n%100 >= 11 && n%100 <= 13:
	case n%10 == 1:
		suffix = "st"
	case n%10 == 2:
		suffix = "nd"
	case n%10 == 3:
		suffix = "rd"
	}
	return strconv.Itoa(n) + suffix
}

func list(items []string) string {
	switch len(items) {
	case 0:
		return ""
	case 1:
		return items[0]
	}
	return strings.Join(items[:len(items)-1], ", ") + " and " + items[len(items)-1]
}

// View creates the editor of *r for an event that starts at start, with
// what it means, its next times and its RRULE under it. Changed reports
// a change.
func View(c *ui.Context, r *Rule, start time.Time) ui.Element {
	t := c.Theme()
	root := ui.Column(c).Gap(t.Space(4))
	changed := false
	edit := func(fn func()) {
		fn()
		changed = true
	}
	root.Children(func() {
		labeled(c, "freq", "Frequency", func() {
			freq := []string{"Daily", "Weekly", "Monthly", "Yearly"}[r.Freq]
			if ui.Select(c, &freq, []string{"Daily", "Weekly", "Monthly", "Yearly"}).Width(t.Space(36)).Label("Frequency").Changed() {
				edit(func() {
					r.Freq = Freq(slices.Index([]string{"Daily", "Weekly", "Monthly", "Yearly"}, freq))
					r.Days, r.Pos, r.MonthDays, r.Months = nil, 0, nil, nil
				})
			}
		})
		labeled(c, "every", "Every", func() {
			n := float64(max(1, r.Interval))
			if ui.NumberInput(c, &n, 1, 999, 1).Label("Every").Changed() {
				edit(func() { r.Interval = int(n) })
			}
			unit := units[r.Freq]
			if n != 1 {
				unit += "s"
			}
			ui.Text(c, unit)
		})
		switch r.Freq {
		case Weekly:
			labeled(c, "on", "On", func() { weekdays(c, r, start, edit) })
		case Monthly:
			labeled(c, "mode", "", func() {
				mode := 0
				if r.Pos != 0 {
					mode = 1
				}
				if ui.Segmented(c, &mode, "Each", "On the").Label("Repeat on").Changed() {
					edit(func() { r.byPosition(mode == 1, start) })
				}
			})
			if r.Pos == 0 {
				labeled(c, "days", "", func() { monthDays(c, r, start, edit) })
			} else {
				labeled(c, "days", "", func() { positionPicker(c, r, edit) })
			}
		case Yearly:
			labeled(c, "in", "In", func() { months(c, r, start, edit) })
			labeled(c, "pos", "", func() {
				on := r.Pos != 0
				if ui.Checkbox(c, &on, "On the").Changed() {
					edit(func() { r.byPosition(on, start) })
				}
				if on {
					positionPicker(c, r, edit)
				}
			})
		}
		labeled(c, "end", "End", func() { end(c, r, start, edit) })
		ui.Divider(c)
		summary(c, r, start)
	})
	if changed {
		root.SetChanged()
	}
	return root
}

// byPosition repeats the rule on a day of the week at a position in the
// month, the start's, or on days of the month.
func (r *Rule) byPosition(on bool, start time.Time) {
	r.Days, r.Pos, r.MonthDays = nil, 0, nil
	if on {
		r.Days = []time.Weekday{start.Weekday()}
		r.Pos = (start.Day()-1)/7 + 1
		if r.Pos == 5 {
			r.Pos = -1
		}
	}
}

func labeled(c *ui.Context, key, label string, fn func()) {
	t := c.Theme()
	ui.Row(c.Key(key)).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
		ui.Text(c, label).Width(t.Space(22)).TextAlign(ui.End).TextColor(t.TextMuted)
		fn()
	})
}

// chip creates a toggle of a picker's grid, size wide.
func chip(c *ui.Context, label string, on bool, size float32) ui.Element {
	t := c.Theme()
	e := ui.ButtonBase(c).Size(size, t.Space(8)).Radius(t.Space(1.5)).Center().Checked(on).Label(label)
	fill, text := t.Surface, t.Text
	switch {
	case on && e.Pressed():
		fill, text = t.AccentPressed, t.AccentText
	case on:
		fill, text = t.Accent, t.AccentText
	case e.Pressed():
		fill = t.SurfacePressed
	case e.Hovered():
		fill = t.SurfaceHover
	}
	return e.Background(fill).Border(1, t.Border).Children(func() {
		ui.Text(c, label).TextColor(text).FontFeatures("tnum").SingleLine()
	})
}

// toggle returns set with v added or taken out, but not its last.
func toggle[T comparable](set []T, v T) []T {
	if i := slices.Index(set, v); i >= 0 {
		if len(set) == 1 {
			return set
		}
		return slices.Delete(slices.Clone(set), i, i+1)
	}
	return append(slices.Clone(set), v)
}

func weekdays(c *ui.Context, r *Rule, start time.Time, edit func(func())) {
	t := c.Theme()
	first := c.Region().FirstWeekday
	ui.Row(c).Gap(t.Space(1)).Children(func() {
		for i := range 7 {
			d := (first + time.Weekday(i)) % 7
			if chip(c.Key(i), d.String()[:3], slices.Contains(r.days(start), d), t.Space(12)).Clicked() {
				edit(func() { r.Days = toggle(r.days(start), d) })
			}
		}
	})
}

func monthDays(c *ui.Context, r *Rule, start time.Time, edit func(func())) {
	t := c.Theme()
	days := r.MonthDays
	if len(days) == 0 {
		days = []int{start.Day()}
	}
	ui.Grid(c).Columns(7).Gap(t.Space(1)).Children(func() {
		for d := 1; d <= 31; d++ {
			if chip(c.Key(d), strconv.Itoa(d), slices.Contains(days, d), t.Space(9)).Clicked() {
				edit(func() { r.MonthDays = toggle(days, d) })
			}
		}
	})
}

func months(c *ui.Context, r *Rule, start time.Time, edit func(func())) {
	t := c.Theme()
	in := r.Months
	if len(in) == 0 {
		in = []time.Month{start.Month()}
	}
	ui.Grid(c).Columns(4).Gap(t.Space(1)).Children(func() {
		for m := time.January; m <= time.December; m++ {
			if chip(c.Key(m), m.String()[:3], slices.Contains(in, m), t.Space(15)).Clicked() {
				edit(func() { r.Months = toggle(in, m) })
			}
		}
	})
}

// positionPicker creates the menus of "On the": the position in the
// month and the days.
func positionPicker(c *ui.Context, r *Rule, edit func(func())) {
	t := c.Theme()
	pos := position(r.Pos)
	if ui.Select(c, &pos, positions).Width(t.Space(30)).Label("Position").Changed() {
		edit(func() {
			r.Pos = slices.Index(positions, pos) + 1
			if pos == "last" {
				r.Pos = -1
			}
		})
	}
	kind := dayKind(r.Days)
	if ui.Select(c, &kind, dayKinds).Width(t.Space(36)).Label("Day").Changed() {
		edit(func() {
			switch kind {
			case "day":
				r.Days = slices.Clone(everyDay)
			case "weekday":
				r.Days = slices.Clone(workweek)
			case "weekend day":
				r.Days = slices.Clone(weekend)
			default:
				r.Days = []time.Weekday{time.Weekday(slices.Index(dayKinds, kind))}
			}
		})
	}
}

func end(c *ui.Context, r *Rule, start time.Time, edit func(func())) {
	t := c.Theme()
	ends := []string{"Never", "On date", "After"}
	how := "Never"
	switch {
	case r.Count > 0:
		how = "After"
	case !r.Until.IsZero():
		how = "On date"
	}
	if ui.Select(c, &how, ends).Width(t.Space(32)).Label("End").Changed() {
		edit(func() {
			r.Count, r.Until = 0, time.Time{}
			switch how {
			case "On date":
				r.Until = dayOf(start.AddDate(0, 3, 0))
			case "After":
				r.Count = 10
			}
		})
	}
	switch how {
	case "On date":
		until := r.Until
		if ui.DateInput(c, &until).Label("End date").Changed() {
			edit(func() { r.Until = dayOf(until) })
		}
	case "After":
		n := float64(r.Count)
		if ui.NumberInput(c, &n, 1, 999, 1).Label("Times").Changed() {
			edit(func() { r.Count = int(n) })
		}
		word := "times"
		if r.Count == 1 {
			word = "time"
		}
		ui.Text(c, word)
	}
}

// summary creates what the rule means, its next times and its RRULE.
func summary(c *ui.Context, r *Rule, start time.Time) {
	t := c.Theme()
	ui.Text(c, r.Describe(start)).Bold()
	next := r.Times(start, c.Now(), 5)
	ui.Column(c).Gap(t.Space(1)).Children(func() {
		if len(next) == 0 {
			ui.Text(c, "No more times").TextColor(t.TextMuted)
		}
		for i, at := range next {
			ui.Text(c.Key(i), at.Format("Mon, Jan 2 2006 at 15:04")).FontFeatures("tnum")
		}
	})
	ui.Text(c, "RRULE:"+r.String()).Font("monospace").FontSize(t.FontSize - 1).TextColor(t.TextMuted).Selectable()
}
