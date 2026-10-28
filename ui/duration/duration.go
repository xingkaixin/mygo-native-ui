// Package duration is a field for a length of time, as time trackers and
// issue estimates take one: typed the way people write it, "1h 30m",
// "90m", "1.5h", "1:30" or "2d 4h", and written back as 1h 30m once the
// focus leaves.
//
//	duration.Field(c, &app.estimate, &app.d)
//
// While the field has the focus, what the text reads as shows after it,
// and the duration changes with it; text it cannot read shows in the
// danger color and changes nothing. A bare number is minutes, and a day
// is 24 hours. Up and Down add or take 15 minutes, an hour with Shift.
package duration

import (
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/egoist/mygo/ui"
)

const day = 24 * time.Hour

var units = []struct {
	d     time.Duration
	names []string
}{
	{7 * day, []string{"w", "wk", "wks", "week", "weeks"}},
	{day, []string{"d", "day", "days"}},
	{time.Hour, []string{"h", "hr", "hrs", "hour", "hours"}},
	{time.Minute, []string{"m", "min", "mins", "minute", "minutes"}},
	{time.Second, []string{"s", "sec", "secs", "second", "seconds"}},
}

// Format writes d in days, hours, minutes and seconds, leaving out those
// that are zero: 2d 4h 30m.
func Format(d time.Duration) string {
	if d <= 0 {
		return "0m"
	}
	var parts []string
	for _, u := range units[1:] {
		if n := d / u.d; n > 0 {
			parts = append(parts, strconv.FormatInt(int64(n), 10)+u.names[0])
			d -= n * u.d
		}
	}
	return strings.Join(parts, " ")
}

// Parse reads a duration as people write it: numbers with units, "1h 30m",
// "1.5 hours"; hours and minutes, "1:30", and seconds, "1:30:15"; or a bare
// number of minutes.
func Parse(text string) (time.Duration, bool) {
	text = strings.ToLower(strings.TrimSpace(text))
	if text == "" {
		return 0, false
	}
	if strings.Contains(text, ":") {
		parts := strings.Split(text, ":")
		if len(parts) > 3 {
			return 0, false
		}
		var d time.Duration
		for i, p := range parts {
			n, err := strconv.Atoi(strings.TrimSpace(p))
			if err != nil || n < 0 || i > 0 && n > 59 {
				return 0, false
			}
			d += time.Duration(n) * []time.Duration{time.Hour, time.Minute, time.Second}[i]
		}
		return d, true
	}
	if n, err := strconv.ParseFloat(text, 64); err == nil && n >= 0 {
		return time.Duration(n * float64(time.Minute)), true
	}
	var d time.Duration
	rest := []rune(text)
	for len(rest) > 0 {
		i := 0
		for i < len(rest) && (unicode.IsDigit(rest[i]) || rest[i] == '.') {
			i++
		}
		j := i
		for j < len(rest) && rest[j] == ' ' {
			j++
		}
		k := j
		for k < len(rest) && unicode.IsLetter(rest[k]) {
			k++
		}
		n, err := strconv.ParseFloat(string(rest[:i]), 64)
		unit, ok := lookup(string(rest[j:k]))
		if err != nil || !ok {
			return 0, false
		}
		d += time.Duration(n * float64(unit))
		rest = []rune(strings.TrimLeft(string(rest[k:]), " ,"))
	}
	return d, true
}

func lookup(name string) (time.Duration, bool) {
	for _, u := range units {
		for _, n := range u.names {
			if n == name {
				return u.d, true
			}
		}
	}
	return 0, false
}

// State is what a field keeps: the text being typed. Keep it in the app's
// state, one for each field.
type State struct {
	text    string
	editing bool
}

// Field creates the field of *d.
func Field(c *ui.Context, s *State, d *time.Duration) ui.Element {
	t := c.Theme()
	if !s.editing {
		s.text = ""
		if *d > 0 {
			s.text = Format(*d)
		}
	}
	field := ui.Row(c).AlignItems(ui.Center).Radius(t.Radius).Background(t.Surface).Border(1, t.Border)
	var in ui.Element
	field.Children(func() {
		in = ui.TextInputBase(c, &s.text).Grow(1).Padding(t.Space(1.5), t.Space(2.5)).Label("Duration").Placeholder("1h 30m")
		in.OnTextKey(func(k ui.TextKey) bool {
			step := 15 * time.Minute
			if k.Mods&ui.Shift != 0 {
				step = time.Hour
			}
			switch {
			case k.Key == ui.KeyUp && k.Mods&^ui.Shift == 0:
				*d = (*d/step + 1) * step
			case k.Key == ui.KeyDown && k.Mods&^ui.Shift == 0:
				*d = max((*d-1)/step*step, 0)
			default:
				return false
			}
			s.text = Format(*d)
			k.SetText(s.text)
			return true
		})
		if !in.Focused() || strings.TrimSpace(s.text) == "" {
			return
		}
		if v, ok := Parse(s.text); ok && Format(v) != strings.TrimSpace(s.text) {
			ui.Text(c, Format(v)).FontSize(t.FontSize-1).TextColor(t.TextMuted).Padding(0, t.Space(2.5), 0, 0)
		}
	})
	if in.Focused() {
		field.Border(1, t.Accent)
		s.editing = true
	}
	if in.Changed() {
		if v, ok := Parse(s.text); ok {
			*d = v
		} else if strings.TrimSpace(s.text) == "" {
			*d = 0
		}
	}
	if s.editing && in.Focused() && strings.TrimSpace(s.text) != "" {
		if _, ok := Parse(s.text); !ok {
			field.Border(1, t.Danger)
			in.TextColor(t.Danger)
		}
	}
	if s.editing && (!in.Focused() || in.Submitted()) {
		s.editing = false
		c.Invalidate()
	}
	return field
}
