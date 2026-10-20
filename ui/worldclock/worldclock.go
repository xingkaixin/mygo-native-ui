// Package worldclock finds a time for a meeting across places, as World
// Time Buddy does: a row for each place, the first home, with its name,
// its offset from UTC and the abbreviation of its zone, and its date and
// time at the time chosen; to the right 24 hours, from midnight of the
// day chosen at home, each column the same instant in every row. A cell
// shows the hour there, in the user's 12- or 24-hour format, or the day
// where the date there changes, deep at night and light in working
// hours. A line shows the time now, and a band the time chosen, which a
// click or a drag across the hours sets, and Left and Right move by an
// hour, with Shift by 15 minutes. Changed reports a new time.
//
//	places := []worldclock.Place{{Name: "Berlin", Loc: berlin}, {Name: "Tokyo", Loc: tokyo}}
//	if worldclock.View(c, &app.at, places).Changed() {
//		app.save()
//	}
package worldclock

import (
	"fmt"
	"strconv"
	"time"

	"github.com/egoist/mygo/i18n"
	"github.com/egoist/mygo/ui"
)

// Place is a row of the planner: a name, as "Tokyo", and its time zone.
type Place struct {
	Name string
	Loc  *time.Location
}

// View creates the planner of places at *at, which it sets as the user
// chooses another time. The day shown is that of *at at home, the first
// place, so moving it past midnight shows the next day. Give it a width,
// or grow it in its parent: the hours share what the names leave.
func View(c *ui.Context, at *time.Time, places []Place) ui.Element {
	t := c.Theme()
	view := ui.Row(c).Gap(t.Space(4))
	if len(places) == 0 {
		return view
	}
	locale := c.Region().FormatLocale(c.Locale())
	now := c.Now()
	c.After(now.Truncate(time.Minute).Add(time.Minute).Sub(now))
	rowH, gap, pad := t.Space(11), t.Space(1), t.Space(2)
	height := float32(len(places))*(rowH+gap) - gap + 2*pad
	changed := false
	set := func(tm time.Time) {
		if !tm.Equal(*at) {
			*at, changed = tm, true
		}
	}
	view.Children(func() {
		labels := ui.Column(c).Width(t.Space(60)).Shrink(0).Gap(gap).Padding(pad, 0)
		grid := ui.Box(c).Grow(1).MinWidth(24 * t.Space(8)).Height(height).Cursor(ui.CursorPointer).
			Focusable().Role(ui.RoleSlider).Label("Time")
		start := midnight(*at, places[0].Loc)
		w := grid.LayoutBounds().W
		_, _, dragging := grid.Dragged()
		if (grid.Pressed() || dragging) && w > 0 {
			x, _, _ := grid.PointerPosition()
			set(start.Add(time.Duration(max(0, min(23, int(x/w*24)))) * time.Hour))
		}
		for _, k := range []struct {
			mods ui.Modifiers
			key  ui.Key
			by   time.Duration
		}{{0, ui.KeyLeft, -time.Hour}, {0, ui.KeyRight, time.Hour}, {ui.Shift, ui.KeyLeft, -15 * time.Minute}, {ui.Shift, ui.KeyRight, 15 * time.Minute}} {
			if grid.Shortcut(k.mods, k.key) {
				set(at.Add(k.by))
			}
		}
		chosen := *at
		start = midnight(chosen, places[0].Loc)
		grid.Value(i18n.FormatDate(locale, chosen.In(places[0].Loc), i18n.DateOptions{Skeleton: "MMMEdjm"}))
		labels.Children(func() {
			for i, p := range places {
				label(c, p, chosen, i == 0, locale, rowH)
			}
		})
		am, pm := i18n.DayPeriods(locale, i18n.Abbreviated)
		f := format{i18n.HourCycle(locale) == 12, am, pm, i18n.Weekdays(locale, i18n.Abbreviated, false)}
		dark := t.Dark
		grid.Draw(func(p *ui.Painter, r ui.Rect) {
			cw := r.W / 24
			radius := t.Space(1.5)
			for i, pl := range places {
				y := r.Y + pad + float32(i)*(rowH+gap)
				hour := func(k int) time.Time { return start.Add(time.Duration(k) * time.Hour).In(pl.Loc) }
				// The hours of each day there are a run with round ends, so
				// that where the date changes shows.
				from := 0
				for k := 1; k <= 24; k++ {
					if k < 24 && sameDay(hour(k), hour(k-1)) {
						continue
					}
					run := ui.Rect{X: r.X + float32(from)*cw, Y: y, W: float32(k-from)*cw - 1, H: rowH}
					p.Clip(run, radius, func() {
						for j := from; j < k; j++ {
							tm := hour(j)
							cell := ui.Rect{X: r.X + float32(j)*cw, Y: y, W: cw - 1, H: rowH}
							fill, text := tint(tm.Hour(), dark, t)
							p.Fill(cell, fill, 0)
							centered(p, cell, f.lines(tm, t.FontSize, text))
						}
					})
					from = k
				}
			}
			p.Clip(r, 0, func() {
				bx := r.X + float32(chosen.Sub(start).Hours())*cw
				band := ui.Rect{X: bx - 1, Y: r.Y + pad/2, W: cw + 1, H: r.H - pad}
				p.Fill(band, t.Accent.Alpha(0.16), radius)
				p.Stroke(band, t.Accent, radius, 2)
				if d := now.Sub(start); d >= 0 && d < 24*time.Hour {
					x := r.X + float32(d.Hours())*cw
					p.Fill(ui.Rect{X: x - 1, Y: r.Y + pad/2, W: 2, H: r.H - pad/2}, t.Danger, 1)
					var dot ui.Path
					dot.Circle(x, r.Y+pad/2, 4)
					p.FillPath(&dot, t.Danger)
				}
			})
		})
	})
	if changed {
		view.SetChanged()
	}
	return view
}

// label shows the name of p, its zone, and its time and date at at, a row
// h high.
func label(c *ui.Context, p Place, at time.Time, home bool, locale string, h float32) {
	t := c.Theme()
	tm := at.In(p.Loc)
	ui.Row(c).Height(h).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
		ui.Column(c).Grow(1).Shrink(1).Gap(t.Space(0.5)).Children(func() {
			ui.Text(c, p.Name).Bold().SingleLine()
			z := zone(tm)
			if home {
				z += " · Home"
			}
			ui.Text(c, z).FontSize(t.FontSize - 2).FontFeatures("tnum").TextColor(t.TextMuted).SingleLine()
		})
		ui.Column(c).AlignItems(ui.End).Gap(t.Space(0.5)).Children(func() {
			ui.Text(c, i18n.FormatDate(locale, tm, i18n.DateOptions{Skeleton: "jm"})).Bold().FontFeatures("tnum").NoWrap()
			ui.Text(c, i18n.FormatDate(locale, tm, i18n.DateOptions{Skeleton: "MMMEd"})).FontSize(t.FontSize - 2).TextColor(t.TextMuted).NoWrap()
		})
	})
}

// zone writes the offset from UTC of tm and the abbreviation of its zone:
// "UTC+5:30 · IST".
func zone(tm time.Time) string {
	name, off := tm.Zone()
	s := "UTC"
	if off != 0 {
		sign := "+"
		if off < 0 {
			sign, off = "−", -off
		}
		s += sign + strconv.Itoa(off/3600)
		if m := off % 3600 / 60; m != 0 {
			s += fmt.Sprintf(":%02d", m)
		}
	}
	// Zones without an abbreviation of their own, as Kathmandu's, are
	// named by their offset ("+0545"), which shows already.
	if name != "" && name != "UTC" && name[0] != '+' && name[0] != '-' {
		s += " · " + name
	}
	return s
}

// format is how cells write hours and days, in the user's regional
// formats.
type format struct {
	twelve bool
	am, pm string
	days   [7]string
}

// lines returns the lines of the cell of the hour from tm, in text about
// size high and of color: the day where the date changes ("Tue", "21"),
// else the hour ("5:30", "PM").
func (f format) lines(tm time.Time, size float32, color ui.Color) []ui.Span {
	if !sameDay(tm, tm.Add(-time.Hour)) {
		return []ui.Span{
			{Text: f.days[tm.Weekday()], Size: size - 3, Weight: 600, Color: color},
			{Text: strconv.Itoa(tm.Day()), Size: size, Weight: 700, Color: color},
		}
	}
	if !f.twelve {
		return []ui.Span{{Text: clock(tm.Hour(), tm.Minute()), Size: size - 1, Weight: 500, Color: color}}
	}
	period := f.am
	if tm.Hour() >= 12 {
		period = f.pm
	}
	return []ui.Span{
		{Text: clock((tm.Hour()+11)%12+1, tm.Minute()), Size: size - 1, Weight: 500, Color: color},
		{Text: period, Size: size - 4, Color: color.Alpha(0.65)},
	}
}

// tint returns the fill of a cell of hour h there and the color of its
// text: deep at night, from 22 to 6, light in working hours, from 9 to
// 18, and between them else.
func tint(h int, dark bool, t *ui.Theme) (fill, text ui.Color) {
	k := 1
	switch {
	case h >= 22 || h < 6:
		k = 0
	case h >= 9 && h < 18:
		k = 2
	}
	if dark {
		return ui.Oklch([]float32{0.28, 0.38, 0.5}[k], []float32{0.05, 0.07, 0.08}[k], 262), t.Text
	}
	fill = ui.Oklch([]float32{0.42, 0.80, 0.95}[k], []float32{0.1, 0.07, 0.03}[k], 262)
	if k == 0 {
		return fill, ui.RGB(255, 255, 255)
	}
	return fill, t.Text
}

// clock writes an hour, with its minutes unless they are 0: "5", "5:30".
func clock(h, m int) string {
	if m == 0 {
		return strconv.Itoa(h)
	}
	return fmt.Sprintf("%d:%02d", h, m)
}

// centered draws lines of text, each on its own, in the middle of r.
func centered(p *ui.Painter, r ui.Rect, lines []ui.Span) {
	var ws, hs []float32
	total := float32(0)
	for _, l := range lines {
		w, h := p.MeasureText(0, l)
		ws, hs = append(ws, w), append(hs, h)
		total += h
	}
	y := r.Y + (r.H-total)/2
	for i, l := range lines {
		p.RichText(r.X+(r.W-ws[i])/2, y, 0, l)
		y += hs[i]
	}
}

// midnight returns the start of the day of tm in loc.
func midnight(tm time.Time, loc *time.Location) time.Time {
	y, m, d := tm.In(loc).Date()
	return time.Date(y, m, d, 0, 0, 0, 0, loc)
}

func sameDay(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}
