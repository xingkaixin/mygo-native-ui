// Package lunar shows a month with the Chinese calendar, 农历, as Chinese
// calendar apps do: under each day's date the day of the Chinese month,
// 初一 to 三十, or on 初一 the month, as 闰四月 or 腊月, and instead a solar
// term, as 清明, or a festival, as 中秋, in the accent. The days of the
// months around fill the weeks, dimmed, and the weeks start on the first
// day of the user's regional formats. Above are the year and month, the
// Chinese year of the day chosen with its 干支 and animal, as 乙巳年 · 蛇,
// and buttons to the months before and after and to today. Its text is
// Chinese whatever the window's language.
//
// Detail writes a day out: its date in the Chinese calendar, the 干支 of
// its year, month and day, and its solar term. The Chinese year changes at
// 春节, and so does the year of the header and of Date, but the 干支 of the
// year in Detail changes at 立春, as the month's changes at a solar term:
// from 立春 to 春节 they are a year apart.
//
// Of and Term convert days without a view, from 1900 to 2100.
//
//	if lunar.View(c, &app.day, app.marks).Grow(1).Changed() {
//		app.save()
//	}
//	lunar.Detail(c, app.day)
package lunar

import (
	"fmt"
	"time"

	"github.com/egoist/mygo/ui"
)

// Mark is something of the app's on a day, as a birthday: its title, in
// its color; the zero Color is the accent.
type Mark struct {
	Title string
	Color ui.Color
}

// View creates the month of *day, with no day chosen while it is zero.
// A click on a day sets *day to its midnight, as do the arrows, moving a
// day or a week, while the month has the keyboard focus; Enter and Space
// choose the day they move from, today with none chosen. The buttons move
// the day chosen to the months before and after, and to today. Changed
// reports a new day.
//
// marks, unless nil, returns the app's marks of a day, which show under
// it. Give the view a size, or grow it in its parent.
func View(c *ui.Context, day *time.Time, marks func(day time.Time) []Mark) ui.Element {
	t := c.Theme()
	now := c.Now()
	loc := now.Location()
	if !day.IsZero() {
		loc = day.Location()
	}
	today := midnight(now.In(loc))
	c.After(today.AddDate(0, 0, 1).Sub(now))
	changed := false
	choose := func(d time.Time) {
		if !d.Equal(*day) {
			*day, changed = d, true
		}
	}
	cursor := func() time.Time {
		if day.IsZero() {
			return today
		}
		return midnight(*day)
	}
	root := ui.Column(c).Gap(t.Space(3)).Lang("zh-Hans")
	root.Children(func() {
		ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
			cur := cursor()
			ui.Text(c, fmt.Sprintf("%d年%d月", cur.Year(), cur.Month())).Bold().FontSize(t.FontSize + 8).FontFeatures("tnum")
			if l, ok := Of(cur); ok {
				ui.Text(c, yearName(l.Year)).FontSize(t.FontSize + 1).TextColor(t.TextMuted)
			}
			ui.Box(c).Grow(1)
			if ui.Button(c, "‹").Label("上个月").Clicked() {
				choose(addMonths(cur, -1))
			}
			if ui.Button(c, "今天").Clicked() {
				choose(today)
			}
			if ui.Button(c, "›").Label("下个月").Clicked() {
				choose(addMonths(cur, 1))
			}
		})
		grid := ui.Column(c).Grow(1).Gap(t.Space(1)).Focusable().FocusRing(false).Role(ui.RoleTable)
		for _, k := range []struct {
			key ui.Key
			by  int
		}{{ui.KeyLeft, -1}, {ui.KeyRight, 1}, {ui.KeyUp, -7}, {ui.KeyDown, 7}, {ui.KeyEnter, 0}, {ui.KeySpace, 0}} {
			if grid.Shortcut(0, k.key) {
				choose(cursor().AddDate(0, 0, k.by))
			}
		}
		cur := cursor()
		first := time.Date(cur.Year(), cur.Month(), 1, 0, 0, 0, 0, loc)
		grid.Label(fmt.Sprintf("%d年%d月", first.Year(), first.Month()))
		weekStart := c.Region().FirstWeekday
		start := first.AddDate(0, 0, -((int(first.Weekday()) - int(weekStart) + 7) % 7))
		termOn := map[int]string{}
		for n := -1; n <= 1; n++ {
			m := first.AddDate(0, n, 0)
			for i := range 2 {
				termOn[days(m)+termDay(m.Year(), m.Month(), i)-1] = terms[2*(int(m.Month())-1)+i]
			}
		}
		ring := grid.FocusVisible()
		grid.Children(func() {
			ui.Row(c).Gap(t.Space(1)).Children(func() {
				for i := range 7 {
					ui.Text(c, string(weekdays[(int(weekStart)+i)%7])).Grow(1).Basis(0).TextAlign(ui.Center).
						FontSize(t.FontSize - 1).TextColor(t.TextMuted)
				}
			})
			for w := range 6 {
				ui.Row(c.Key(w)).Grow(1).Basis(0).MinHeight(t.Space(16)).Gap(t.Space(1)).AlignItems(ui.Stretch).Role(ui.RoleRow).Children(func() {
					for i := range 7 {
						d := start.AddDate(0, 0, w*7+i)
						var ms []Mark
						if marks != nil {
							ms = marks(d)
						}
						cell := dayCell(c.Key(i), d, ms, termOn[days(d)], cellState{
							chosen: !day.IsZero() && d.Equal(cur),
							today:  d.Equal(today),
							other:  d.Month() != first.Month(),
							ring:   ring && d.Equal(cur),
						})
						if cell.Clicked() {
							choose(d)
						}
						if d.Equal(cur) {
							grid.ActiveDescendant(cell)
						}
					}
				})
			}
		})
	})
	if changed {
		root.SetChanged()
	}
	return root
}

// cellState is how a day shows: chosen, today, of another month, and with
// the focus ring, as the day the keys move from while they have the focus.
type cellState struct {
	chosen, today, other, ring bool
}

// dayCell builds the cell of d: its date, below it its day of the Chinese
// calendar, or term, its solar term, or its festival, and its marks.
func dayCell(c *ui.Context, d time.Time, marks []Mark, term string, s cellState) ui.Element {
	t := c.Theme()
	l, ok := Of(d)
	next, _ := Of(d.AddDate(0, 0, 1))
	small, special := "", false
	if ok {
		small = l.dayName()
		if l.Day == 1 {
			small = l.monthName()
		}
	}
	if term != "" {
		small, special = term, true
	}
	if f := festival(l, next); ok && f != "" {
		small, special = f, true
	}
	label := fmt.Sprintf("%d年%d月%d日", d.Year(), d.Month(), d.Day())
	if ok {
		label += " 农历" + l.monthName() + l.dayName()
	}
	if special {
		label += " " + small
	}
	for _, m := range marks {
		label += " " + m.Title
	}
	cell := ui.Column(c).Grow(1).Basis(0).Padding(t.Space(1.5), t.Space(1)).Gap(t.Space(0.5)).AlignItems(ui.Center).
		Radius(t.Space(2)).Role(ui.RoleButton).Label(label).Selected(s.chosen)
	number, text, accent := t.Text, t.TextMuted, t.Accent
	if s.other {
		number, text, accent = t.TextMuted.Alpha(0.6), t.TextMuted.Alpha(0.5), t.Accent.Alpha(0.5)
	}
	switch {
	case s.chosen:
		cell.Background(t.Accent.Alpha(0.12))
	case cell.Hovered():
		cell.Background(t.SurfaceHover)
	}
	if s.ring {
		radius := t.Space(2)
		cell.DrawOver(func(p *ui.Painter, r ui.Rect) { p.FocusRing(r, [4]float32{radius, radius, radius, radius}) })
	}
	cell.Children(func() {
		size := t.Space(7.5)
		pill := ui.Row(c).Height(size).MinWidth(size).Padding(0, t.Space(1.5)).Radius(size / 2).Justify(ui.Center).AlignItems(ui.Center)
		pill.Children(func() {
			n := ui.Text(c, fmt.Sprint(d.Day())).FontSize(t.FontSize + 4).FontFeatures("tnum").TextColor(number).NoWrap()
			switch {
			case s.today:
				pill.Background(t.Accent)
				n.TextColor(t.AccentText).Bold()
			case s.chosen:
				n.TextColor(t.Accent).Bold()
			}
		})
		if special {
			text = accent
		}
		ui.Text(c, small).FontSize(t.FontSize - 2).TextColor(text).SingleLine()
		for i, m := range marks {
			color := m.Color
			if color == (ui.Color{}) {
				color = t.Accent
			}
			bg := color.Alpha(0.16)
			if s.other {
				color, bg = color.Alpha(0.5), color.Alpha(0.08)
			}
			ui.Row(c.Key(i)).AlignSelf(ui.Stretch).Justify(ui.Center).Padding(t.Space(0.25), t.Space(1)).Radius(t.Space(1)).
				Background(bg).Children(func() {
				ui.Text(c, m.Title).FontSize(t.FontSize - 3).TextColor(color).SingleLine().Shrink(1)
			})
		}
	})
	return cell
}

// Detail creates the lines of day in the Chinese calendar: its date, as
// 农历乙巳年腊月廿三; the 干支 of its year, from 立春, of its month, from the
// solar term that begins it, and of its day, as 丙午年 庚寅月 乙卯日; and its
// solar term, in the accent. It is empty outside 1900 to 2100.
func Detail(c *ui.Context, day time.Time) ui.Element {
	t := c.Theme()
	col := ui.Column(c).Gap(t.Space(1.5)).Lang("zh-Hans")
	l, ok := Of(day)
	if !ok {
		return col
	}
	year, month, d := pillars(day)
	col.Children(func() {
		ui.Text(c, "农历"+l.String()).Bold().FontSize(t.FontSize + 3)
		ui.Text(c, year+"年 "+month+"月 "+d+"日").TextColor(t.TextMuted)
		if term := Term(day); term != "" {
			ui.Row(c).AlignSelf(ui.Start).Padding(t.Space(0.5), t.Space(2)).Radius(t.Space(1)).Background(t.Accent.Alpha(0.14)).Children(func() {
				ui.Text(c, term).Bold().TextColor(t.Accent)
			})
		}
	})
	return col
}

// addMonths returns d moved by n months, on its day or on the month's last
// when it has fewer days.
func addMonths(d time.Time, n int) time.Time {
	first := time.Date(d.Year(), d.Month()+time.Month(n), 1, 0, 0, 0, 0, d.Location())
	return first.AddDate(0, 0, min(d.Day(), first.AddDate(0, 1, -1).Day())-1)
}

// midnight returns the midnight of t's day.
func midnight(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}
