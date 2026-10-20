// Package heatmap shows counts by day over a year, as GitHub shows a
// person's contributions: a square a day, a column a week, deeper for
// more, the months above and the days of the week beside, the count of
// the year above and a legend below. The pointer on a day tells of it; a
// click chooses it, which Chosen returns, and a click on it again lets it
// go.
//
//	heatmap.View(c, &app.activity, app.counts, c.Now(), "commit")
//	if day, ok := app.activity.Chosen(); ok {
//		... // show the commits of day
//	}
package heatmap

import (
	"fmt"
	"slices"
	"strconv"
	"time"

	"github.com/egoist/mygo/ui"
)

// Date is a day of the calendar.
type Date struct {
	Year  int
	Month time.Month
	Day   int
}

// DateOf returns the day of t, in its location.
func DateOf(t time.Time) Date {
	y, m, d := t.Date()
	return Date{y, m, d}
}

func (d Date) time() time.Time { return time.Date(d.Year, d.Month, d.Day, 12, 0, 0, 0, time.UTC) }

// State is what a view keeps: the day chosen. Keep it in the app's state,
// one for each view.
type State struct {
	chosen Date
}

// Chosen returns the day chosen, or false with none.
func (s *State) Chosen() (Date, bool) { return s.chosen, s.chosen != Date{} }

// weeks is how many weeks show, the last that of the day the view ends.
const weeks = 53

// View creates the view of counts, the count of each day, over the year
// to the day of end, unit naming what is counted, as "commit".
func View(c *ui.Context, s *State, counts map[Date]int, end time.Time, unit string) ui.Element {
	t := c.Theme()
	last := DateOf(end).time()
	first := last.AddDate(0, 0, -int(last.Weekday())-7*(weeks-1))
	total := 0
	var values []int
	for d, n := range counts {
		if tm := d.time(); n > 0 && !tm.Before(first) && !tm.After(last) {
			total += n
			values = append(values, n)
		}
	}
	slices.Sort(values)
	level := levels(values)
	cell, gap := t.Space(2.75), t.Space(0.75)
	step := cell + gap
	top := t.FontSize + t.Space(1)
	left := t.Space(8)
	dark := t.Dark
	var hovered Date
	return ui.Column(c).Gap(t.Space(2)).Children(func() {
		ui.Text(c, fmt.Sprintf("%s in the last year", count(total, unit))).Bold()
		box := ui.Box(c).Size(left+weeks*step, top+7*step).Role(ui.RoleImage).
			Label(fmt.Sprintf("%s in the last year", count(total, unit)))
		// at returns the day at x, y in the box, if one shows there.
		at := func(x, y float32) (Date, bool) {
			col, row := int((x-left)/step), int((y-top)/step)
			if x < left || y < top || col >= weeks || row >= 7 {
				return Date{}, false
			}
			d := first.AddDate(0, 0, col*7+row)
			if d.After(last) {
				return Date{}, false
			}
			return DateOf(d), true
		}
		if x, y, ok := box.PointerPosition(); ok {
			hovered, _ = at(x, y)
		}
		if box.Clicked() && hovered != (Date{}) {
			if s.chosen == hovered {
				s.chosen = Date{}
			} else {
				s.chosen = hovered
			}
		}
		chosen := s.chosen
		box.Draw(func(p *ui.Painter, r ui.Rect) {
			small := t.FontSize - 3
			for row, name := range []string{"", "Mon", "", "Wed", "", "Fri", ""} {
				if name != "" {
					p.Text(r.X, r.Y+top+float32(row)*step+(cell-small*1.2)/2, name, small, t.TextMuted)
				}
			}
			month := time.Month(0)
			for col := range weeks {
				day := first.AddDate(0, 0, col*7)
				x := r.X + left + float32(col)*step
				// A month's name goes over the first week that starts in it.
				if m := day.Month(); m != month && day.Day() <= 7 && col < weeks-2 {
					month = m
					p.Text(x, r.Y, day.Format("Jan"), small, t.TextMuted)
				}
				for row := range 7 {
					d := day.AddDate(0, 0, row)
					if d.After(last) {
						break
					}
					rect := ui.Rect{X: x, Y: r.Y + top + float32(row)*step, W: cell, H: cell}
					p.Fill(rect, shade(level(counts[DateOf(d)]), dark, t), 2)
					if date := DateOf(d); date == chosen || date == hovered {
						p.Stroke(rect, t.Text.Alpha(0.6), 2, 1.5)
					}
				}
			}
		})
		ui.Row(c).Gap(t.Space(1)).AlignItems(ui.Center).Width(left + weeks*step).Children(func() {
			line := ""
			switch {
			case hovered != (Date{}):
				line = describe(hovered, counts[hovered], unit)
			case chosen != (Date{}):
				line = describe(chosen, counts[chosen], unit)
			}
			ui.Text(c, line).FontSize(t.FontSize - 1).TextColor(t.TextMuted).Grow(1).Shrink(1).SingleLine()
			ui.Text(c, "Less").FontSize(t.FontSize - 2).TextColor(t.TextMuted)
			for k := range 5 {
				ui.Box(c.Key(k)).Size(cell, cell).Radius(2).Background(shade(k, dark, t))
			}
			ui.Text(c, "More").FontSize(t.FontSize - 2).TextColor(t.TextMuted)
		})
	})
}

// levels returns the level of a count, 0 for none and from 1 to 4 by the
// quarter of values, the sorted counts of the days with any, it falls in.
func levels(values []int) func(n int) int {
	quarter := func(q int) int {
		if len(values) == 0 {
			return 0
		}
		return values[min(len(values)-1, len(values)*q/4)]
	}
	q1, q2, q3 := quarter(1), quarter(2), quarter(3)
	return func(n int) int {
		switch {
		case n <= 0:
			return 0
		case n < q1:
			return 1
		case n < q2:
			return 2
		case n < q3:
			return 3
		}
		return 4
	}
}

// shade returns the color of level k, from 0, an empty day, to 4.
func shade(k int, dark bool, t *ui.Theme) ui.Color {
	if k == 0 {
		return t.Text.Alpha(0.07)
	}
	l := []float32{0, 0.86, 0.72, 0.58, 0.45}[k]
	if dark {
		l = []float32{0, 0.38, 0.5, 0.64, 0.8}[k]
	}
	return ui.Oklch(l, 0.06+0.025*float32(k), 150)
}

// describe returns the line of day: "3 commits on Tuesday, March 4, 2026".
func describe(d Date, n int, unit string) string {
	if n == 0 {
		return "No " + unit + "s on " + d.time().Format("Monday, January 2, 2006")
	}
	return count(n, unit) + " on " + d.time().Format("Monday, January 2, 2006")
}

// count returns n of unit, "1 commit", "1,204 commits".
func count(n int, unit string) string {
	s := strconv.Itoa(n)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	if n != 1 {
		unit += "s"
	}
	return s + " " + unit
}
