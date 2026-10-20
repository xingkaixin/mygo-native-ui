// Package daterange chooses a range of days, as analytics dashboards and
// booking sites do: two months side by side, where a click chooses the
// first day and another the last, the days between lit as the pointer
// moves; ranges such as Last 7 days and This month on the left, and the
// range above, with how many days it is. Weeks start on the first day of
// the user's regional formats, and dates are written in them. Changed
// reports a new range.
//
//	if daterange.Picker(c, &app.picker, &app.from, &app.to, c.Now()).Changed() {
//		app.reload()
//	}
package daterange

import (
	"fmt"
	"math"
	"time"

	"github.com/egoist/mygo/i18n"
	"github.com/egoist/mygo/ui"
)

// State is what a picker keeps: the months shown, and the first day while
// the last is being chosen. Keep it in the app's state, one for each
// picker.
type State struct {
	month   time.Time
	picking bool
	anchor  time.Time
	// hover is the day under the pointer as of the last frame.
	hover time.Time
}

type preset struct {
	name     string
	from, to time.Time
}

func presets(today time.Time) []preset {
	y, m, _ := today.Date()
	month := time.Date(y, m, 1, 0, 0, 0, 0, today.Location())
	return []preset{
		{"Today", today, today},
		{"Yesterday", today.AddDate(0, 0, -1), today.AddDate(0, 0, -1)},
		{"Last 7 days", today.AddDate(0, 0, -6), today},
		{"Last 30 days", today.AddDate(0, 0, -29), today},
		{"This month", month, today},
		{"Last month", month.AddDate(0, -1, 0), month.AddDate(0, 0, -1)},
		{"This year", time.Date(y, 1, 1, 0, 0, 0, 0, today.Location()), today},
	}
}

func day(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}

// shown returns the first month to show, the one before last's.
func shown(last time.Time) time.Time {
	y, m, _ := last.Date()
	return time.Date(y, m-1, 1, 0, 0, 0, 0, last.Location())
}

// Picker creates the picker of the days from *from to *to, both included,
// around today.
func Picker(c *ui.Context, s *State, from, to *time.Time, today time.Time) ui.Element {
	t := c.Theme()
	locale := c.Region().FormatLocale(c.Locale())
	today = day(today)
	if s.month.IsZero() {
		s.month = shown(today)
		if !to.IsZero() {
			s.month = shown(*to)
		}
	}
	changed := false
	set := func(a, b time.Time) {
		if b.Before(a) {
			a, b = b, a
		}
		*from, *to = a, b
		s.picking, changed = false, true
	}
	root := ui.Row(c).Background(t.Background).Border(1, t.Border).Radius(t.Space(2))
	root.Children(func() {
		ui.Column(c).Width(t.Space(36)).Padding(t.Space(2)).Gap(t.Space(0.5)).AlignSelf(ui.Start).Children(func() {
			for i, p := range presets(today) {
				on := !s.picking && day(*from).Equal(p.from) && day(*to).Equal(p.to)
				b := ui.ButtonBase(c.Key(i)).Padding(t.Space(1.5), t.Space(2.5)).Radius(t.Radius).Justify(ui.Start).Label(p.name)
				switch {
				case on:
					b.Background(t.Accent.Alpha(0.12))
				case b.Hovered():
					b.Background(t.SurfaceHover)
				}
				if b.Clicked() {
					set(p.from, p.to)
					s.month = shown(p.to)
				}
				b.Children(func() {
					color := t.Text
					if on {
						color = t.Accent
					}
					ui.Text(c, p.name).TextColor(color)
				})
			}
		})
		ui.Divider(c).Vertical().AlignSelf(ui.Stretch)
		ui.Column(c).Padding(t.Space(4)).Gap(t.Space(4)).Children(func() {
			ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
				if s.picking {
					ui.Text(c, written(locale, s.anchor, s.anchor)).Bold().FontSize(t.FontSize + 2)
					ui.Text(c, "Choose the last day").TextColor(t.TextMuted)
					return
				}
				if from.IsZero() || to.IsZero() {
					ui.Text(c, "Choose the first day").TextColor(t.TextMuted)
					return
				}
				ui.Text(c, written(locale, *from, *to)).Bold().FontSize(t.FontSize + 2)
				n := int(math.Round(day(*to).Sub(day(*from)).Hours()/24)) + 1
				days := fmt.Sprintf("%d days", n)
				if n == 1 {
					days = "1 day"
				}
				ui.Text(c, days).TextColor(t.TextMuted)
			})
			lo, hi := day(*from), day(*to)
			if s.picking {
				lo, hi = s.anchor, s.anchor
				if !s.hover.IsZero() {
					lo, hi = minDay(s.anchor, s.hover), maxDay(s.anchor, s.hover)
				}
			}
			var hover time.Time
			ui.Row(c).Gap(t.Space(8)).Children(func() {
				for k := range 2 {
					m := s.month.AddDate(0, k, 0)
					ui.Column(c.Key(k)).Gap(t.Space(1)).Children(func() {
						ui.Row(c).AlignItems(ui.Center).Children(func() {
							arrow := func(text, label string, by int) {
								b := ui.Button(c, text).Padding(t.Space(0.5), t.Space(2.5)).Label(label)
								if b.Clicked() {
									s.month = s.month.AddDate(0, by, 0)
								}
							}
							if k == 0 {
								arrow("‹", "Previous month", -1)
							}
							ui.Text(c, i18n.FormatDate(locale, m, i18n.DateOptions{Skeleton: "yMMMM"})).Bold().Grow(1).TextAlign(ui.Center)
							if k == 1 {
								arrow("›", "Next month", 1)
							}
						})
						if d, ok := s.grid(c, m, locale, today, lo, hi, func(d time.Time) {
							if s.picking {
								set(s.anchor, d)
							} else {
								s.picking, s.anchor = true, d
							}
						}); ok {
							hover = d
						}
					})
				}
			})
			if s.picking && !hover.Equal(s.hover) {
				c.Invalidate()
			}
			s.hover = hover
		})
	})
	if changed {
		root.SetChanged()
	}
	return root
}

// grid shows month's days, those from lo to hi lit, and returns the day
// under the pointer.
func (s *State) grid(c *ui.Context, month time.Time, locale string, today, lo, hi time.Time, choose func(time.Time)) (time.Time, bool) {
	t := c.Theme()
	w, h := t.Space(9), t.Space(8)
	first := c.Region().FirstWeekday
	names := i18n.Weekdays(locale, i18n.Short, true)
	var hover time.Time
	found := false
	ui.Row(c).Children(func() {
		for i := range 7 {
			ui.Text(c, names[(int(first)+i)%7]).Width(w).TextAlign(ui.Center).FontSize(t.FontSize - 2).TextColor(t.TextMuted)
		}
	})
	start := month.AddDate(0, 0, -((int(month.Weekday()) - int(first) + 7) % 7))
	for wk := range 6 {
		ui.Row(c.Key(wk)).Children(func() {
			for k := range 7 {
				d := start.AddDate(0, 0, wk*7+k)
				if d.Month() != month.Month() {
					ui.Box(c.Key(k)).Size(w, h)
					continue
				}
				in := !d.Before(lo) && !d.After(hi)
				end := d.Equal(lo) || d.Equal(hi)
				cell := ui.ButtonBase(c.Key(k)).Size(w, h).Label(i18n.FormatDate(locale, d, i18n.DateOptions{Skeleton: "yMMMMd"}))
				if cell.Hovered() {
					hover, found = d, true
				}
				if cell.Clicked() {
					choose(d)
				}
				hot := cell.Hovered()
				cell.Draw(func(p *ui.Painter, r ui.Rect) {
					if in && !lo.Equal(hi) {
						band := r
						if d.Equal(lo) {
							band.X, band.W = r.X+r.W/2, r.W/2
						}
						if d.Equal(hi) {
							band.W = r.W / 2
						}
						p.Fill(band, t.Accent.Alpha(0.14), 0)
					}
					switch {
					case end:
						p.Fill(r, t.Accent, t.Radius)
					case hot:
						p.Fill(r, t.SurfaceHover, t.Radius)
					}
				})
				cell.Children(func() {
					text := ui.Text(c, fmt.Sprint(d.Day())).FontFeatures("tnum")
					switch {
					case end:
						text.TextColor(t.AccentText)
					case d.Equal(today):
						text.TextColor(t.Accent).Bold()
					}
				})
			}
		})
	}
	return hover, found
}

// written writes the days from a to b, the year once when it is the same.
func written(locale string, a, b time.Time) string {
	full := i18n.DateOptions{Skeleton: "yMMMd"}
	if day(a).Equal(day(b)) {
		return i18n.FormatDate(locale, a, full)
	}
	first := i18n.FormatDate(locale, a, i18n.DateOptions{Skeleton: "MMMd"})
	if a.Year() != b.Year() {
		first = i18n.FormatDate(locale, a, full)
	}
	return first + " – " + i18n.FormatDate(locale, b, full)
}

func minDay(a, b time.Time) time.Time {
	if b.Before(a) {
		return b
	}
	return a
}

func maxDay(a, b time.Time) time.Time {
	if b.After(a) {
		return b
	}
	return a
}
