// Package cron is a field for a cron schedule, as CI services and job
// schedulers take one: five fields, the minute, the hour, the day of the
// month, the month and the day of the week, each a value, a range, a step
// or a list of them, "*/15", "MON-FRI", "1,15", or a macro, "@daily".
// Under the field show its five fields, the one that is wrong in red,
// what the schedule means, "At 09:00, on Monday through Friday", and the
// next times it runs.
//
//	cron.Field(c, &app.schedule)
//	...
//	s, err := cron.Parse(app.schedule)
//	next := s.Next(time.Now())
package cron

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
)

// Schedule is a parsed cron schedule.
type Schedule struct {
	fields [5]field
}

// field is a field of a schedule: what it says, its items, and the
// values they take.
type field struct {
	text  string
	items []item
	set   uint64
}

// item is a value, a range or a step of a field: from lo to hi every
// step, star for "*".
type item struct {
	lo, hi, step int
	star         bool
}

// star reports whether f is "*", every value.
func (f field) star() bool { return len(f.items) == 1 && f.items[0].star && f.items[0].step == 1 }

func (f field) has(v int) bool { return f.set&(1<<v) != 0 }

// specs are the fields of a schedule: their names, the values they take
// and the names those may go by.
var specs = [5]struct {
	name   string
	lo, hi int
	names  []string
}{
	{"minute", 0, 59, nil},
	{"hour", 0, 23, nil},
	{"day", 1, 31, nil},
	{"month", 1, 12, []string{"", "JAN", "FEB", "MAR", "APR", "MAY", "JUN", "JUL", "AUG", "SEP", "OCT", "NOV", "DEC"}},
	{"weekday", 0, 7, []string{"SUN", "MON", "TUE", "WED", "THU", "FRI", "SAT"}},
}

var macros = map[string]string{
	"@yearly":   "0 0 1 1 *",
	"@annually": "0 0 1 1 *",
	"@monthly":  "0 0 1 * *",
	"@weekly":   "0 0 * * 0",
	"@daily":    "0 0 * * *",
	"@midnight": "0 0 * * *",
	"@hourly":   "0 * * * *",
}

// Error is what is wrong with a schedule, and the index of the field it
// is in, from 0 for the minute, -1 for the schedule as a whole.
type Error struct {
	Field int
	Msg   string
}

func (e *Error) Error() string {
	if e.Field < 0 {
		return e.Msg
	}
	return specs[e.Field].name + ": " + e.Msg
}

// Parse parses a schedule of five fields or a macro. A day of the week
// is from 0 to 7, both Sunday, or SUN to SAT, and a month from 1 to 12 or
// JAN to DEC. With both the day of the month and of the week given, a day
// either is takes, as cron runs them.
func Parse(expr string) (*Schedule, error) {
	expr = strings.TrimSpace(expr)
	if m, ok := macros[strings.ToLower(expr)]; ok {
		expr = m
	} else if strings.HasPrefix(expr, "@") {
		return nil, &Error{-1, fmt.Sprintf("%s is not a macro", expr)}
	}
	parts := strings.Fields(expr)
	if len(parts) != 5 {
		return nil, &Error{-1, fmt.Sprintf("takes 5 fields, not %d", len(parts))}
	}
	s := &Schedule{}
	for i, p := range parts {
		f, err := parseField(i, p)
		if err != nil {
			return nil, &Error{i, err.Error()}
		}
		s.fields[i] = f
	}
	// Sunday is 0 and 7.
	if s.fields[4].has(7) {
		s.fields[4].set |= 1
	}
	return s, nil
}

func parseField(i int, text string) (field, error) {
	spec := specs[i]
	f := field{text: text}
	for part := range strings.SplitSeq(text, ",") {
		it := item{step: 1}
		rng, step, hasStep := strings.Cut(part, "/")
		if hasStep {
			n, err := strconv.Atoi(step)
			if err != nil || n <= 0 {
				return f, fmt.Errorf("%q is not a step", step)
			}
			it.step = n
		}
		switch lo, hi, isRange := strings.Cut(rng, "-"); {
		case rng == "*":
			it.lo, it.hi, it.star = spec.lo, spec.hi, true
		case isRange:
			var err error
			if it.lo, err = value(spec.lo, spec.hi, spec.names, lo); err != nil {
				return f, err
			}
			if it.hi, err = value(spec.lo, spec.hi, spec.names, hi); err != nil {
				return f, err
			}
			if it.hi < it.lo {
				return f, fmt.Errorf("%s comes after %s", lo, hi)
			}
		default:
			v, err := value(spec.lo, spec.hi, spec.names, rng)
			if err != nil {
				return f, err
			}
			it.lo, it.hi = v, v
			if hasStep {
				it.hi = spec.hi
			}
		}
		for v := it.lo; v <= it.hi; v += it.step {
			f.set |= 1 << v
		}
		f.items = append(f.items, it)
	}
	return f, nil
}

// value parses a value from lo to hi, a number or one of names.
func value(lo, hi int, names []string, s string) (int, error) {
	if k := slices.Index(names, strings.ToUpper(s)); k >= 0 && s != "" {
		return k, nil
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("%q is not a value", s)
	}
	if v < lo || v > hi {
		return 0, fmt.Errorf("%d is not from %d to %d", v, lo, hi)
	}
	return v, nil
}

// day reports whether s runs on the day of t.
func (s *Schedule) day(t time.Time) bool {
	dom, dow := s.fields[2], s.fields[4]
	inMonth, inWeek := dom.has(t.Day()), dow.has(int(t.Weekday()))
	switch {
	case dom.star() || dow.star():
		return inMonth && inWeek
	}
	return inMonth || inWeek
}

// Next returns the first time after t that s runs, in t's location, or
// the zero time when it runs in none of the next five years, as for
// February 30th.
func (s *Schedule) Next(t time.Time) time.Time {
	loc := t.Location()
	t = t.Add(time.Minute - time.Duration(t.Second())*time.Second - time.Duration(t.Nanosecond()))
	limit := t.AddDate(5, 0, 0)
	for t.Before(limit) {
		y, m, d := t.Date()
		switch {
		case !s.fields[3].has(int(m)):
			t = time.Date(y, m+1, 1, 0, 0, 0, 0, loc)
		case !s.day(t):
			t = time.Date(y, m, d+1, 0, 0, 0, 0, loc)
		case !s.fields[1].has(t.Hour()):
			t = time.Date(y, m, d, t.Hour()+1, 0, 0, 0, loc)
		case !s.fields[0].has(t.Minute()):
			t = t.Add(time.Minute)
		default:
			return t
		}
	}
	return time.Time{}
}

// Field creates the field editing *expr, with what it means under it.
func Field(c *ui.Context, expr *string) ui.Element {
	t := c.Theme()
	s, err := Parse(*expr)
	bad := -2
	if e, ok := err.(*Error); ok {
		bad = e.Field
	}
	texts := strings.Fields(*expr)
	if m, ok := macros[strings.ToLower(strings.TrimSpace(*expr))]; ok {
		texts = strings.Fields(m)
	}
	return ui.Column(c).Gap(t.Space(3)).Children(func() {
		in := ui.TextInput(c, expr).Font("monospace").Placeholder("*/15 9-17 * * MON-FRI").Label("Schedule")
		if bad == -1 {
			in.Border(1, t.Danger)
		}
		ui.Row(c).Gap(t.Space(2)).Children(func() {
			for i, spec := range specs {
				text := ""
				if i < len(texts) {
					text = texts[i]
				}
				color := t.Border
				if i == bad {
					color = t.Danger
				}
				ui.Column(c.Key(i)).Gap(t.Space(0.5)).Padding(t.Space(1.5), t.Space(2)).Radius(t.Space(1.5)).
					Border(1, color).MinWidth(t.Space(18)).Children(func() {
					ui.Text(c, text).Font("monospace").SingleLine()
					ui.Text(c, spec.name).FontSize(t.FontSize - 2).TextColor(t.TextMuted)
				})
			}
		})
		if err != nil {
			ui.Text(c, err.Error()).FontSize(t.FontSize - 1).TextColor(t.Danger)
			return
		}
		ui.Text(c, s.describe()).Bold()
		now := c.Now()
		at := now
		ui.Column(c).Gap(t.Space(1)).Children(func() {
			for i := range 5 {
				at = s.Next(at)
				if at.IsZero() {
					if i == 0 {
						ui.Text(c, "Never runs").TextColor(t.TextMuted)
					}
					break
				}
				if i == 0 {
					c.After(at.Sub(now))
				}
				ui.Row(c.Key(i)).Gap(t.Space(3)).Children(func() {
					ui.Text(c, at.Format("Mon, Jan 2 2006 at 15:04")).FontFeatures("tnum").Width(t.Space(52))
					ui.Text(c, until(at.Sub(now))).TextColor(t.TextMuted)
				})
			}
		})
	})
}

// until returns how long d is, as "in 3 hours".
func until(d time.Duration) string {
	unit := func(n int, word string) string {
		if n == 1 {
			return "in 1 " + word
		}
		return fmt.Sprintf("in %d %ss", n, word)
	}
	switch {
	case d < time.Hour:
		return unit(max(1, int(d.Round(time.Minute).Minutes())), "minute")
	case d < 48*time.Hour:
		return unit(int(d.Hours()), "hour")
	}
	return unit(int(d.Hours()/24), "day")
}

var (
	weekdays = []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"}
	months   = []string{"", "January", "February", "March", "April", "May", "June", "July", "August", "September",
		"October", "November", "December"}
)

// describe returns what s means in words: "At 09:00, on Monday through
// Friday".
func (s *Schedule) describe() string {
	minute, hour, dom, month, dow := s.fields[0], s.fields[1], s.fields[2], s.fields[3], s.fields[4]
	out := clock(minute, hour)
	var days []string
	if !dom.star() {
		days = append(days, "on day "+words(dom, nil, "day")+" of the month")
	}
	if !dow.star() {
		days = append(days, "on "+words(dow, weekdays, "day of the week"))
	}
	if len(days) > 0 {
		out += ", " + strings.Join(days, " or ")
	}
	if !month.star() {
		out += ", in " + words(month, months, "month")
	}
	return out
}

// clock returns when in a day the minute and hour fields run: "At 09:00",
// "Every 15 minutes", "At minute 30 past hours 9 through 17".
func clock(minute, hour field) string {
	single := func(f field) bool { return len(f.items) == 1 && !f.items[0].star && f.items[0].lo == f.items[0].hi }
	values := func(f field) bool {
		for _, it := range f.items {
			if it.star || it.lo != it.hi {
				return false
			}
		}
		return true
	}
	switch {
	case single(minute) && values(hour):
		var times []string
		for _, it := range hour.items {
			times = append(times, fmt.Sprintf("%02d:%02d", it.lo, minute.items[0].lo))
		}
		return "At " + list(times)
	case minute.star() && hour.star():
		return "Every minute"
	case len(minute.items) == 1 && minute.items[0].star && hour.star():
		return fmt.Sprintf("Every %d minutes", minute.items[0].step)
	case single(minute) && hour.star():
		if minute.items[0].lo == 0 {
			return "Every hour"
		}
		return fmt.Sprintf("At minute %d past every hour", minute.items[0].lo)
	}
	m := "Every minute"
	switch {
	case len(minute.items) == 1 && minute.items[0].star && minute.items[0].step > 1:
		m = fmt.Sprintf("Every %d minutes", minute.items[0].step)
	case !minute.star():
		m = "At minute " + words(minute, nil, "minute")
	}
	if hour.star() {
		return m + " of every hour"
	}
	if len(hour.items) == 1 && hour.items[0].star {
		return m + fmt.Sprintf(" past every %d hours", hour.items[0].step)
	}
	return m + " past hour " + words(hour, nil, "hour")
}

// words returns the items of f in words, values by names when it has
// them: "1 and 15", "Monday through Friday", "every 2nd day".
func words(f field, names []string, unit string) string {
	name := func(v int) string {
		if names != nil {
			return names[v]
		}
		return strconv.Itoa(v)
	}
	var parts []string
	for _, it := range f.items {
		var p string
		switch {
		case it.star:
			p = fmt.Sprintf("every %s %s", ordinal(it.step), unit)
		case it.lo == it.hi:
			p = name(it.lo)
		default:
			p = name(it.lo) + " through " + name(it.hi)
		}
		if it.step > 1 && !it.star {
			p = fmt.Sprintf("every %s %s from %s", ordinal(it.step), unit, p)
		}
		parts = append(parts, p)
	}
	return list(parts)
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

// list joins parts as a sentence does: "a, b and c".
func list(parts []string) string {
	if len(parts) < 2 {
		return strings.Join(parts, "")
	}
	return strings.Join(parts[:len(parts)-1], ", ") + " and " + parts[len(parts)-1]
}
