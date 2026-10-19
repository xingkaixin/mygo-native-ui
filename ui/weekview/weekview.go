// Package weekview shows the events of a week on a grid of days and
// hours, as macOS Calendar's week view does: events overlapping side by
// side, dragged to move them to another time or day and by their bottom
// edge to change when they end.
//
//	monday := time.Date(2026, 10, 19, 0, 0, 0, 0, time.Local)
//	if weekview.View(c, &app.week, monday, 7, app.events).Grow(1).Changed() {
//		app.save()
//	}
package weekview

import (
	"math"
	"slices"
	"time"

	"github.com/egoist/mygo/ui"
)

// snap is the step events move and stretch by.
const snap = 15 * time.Minute

// Event is a stretch of time on the calendar.
type Event struct {
	Title      string
	Start, End time.Time
	// Color colors the event; the zero Color is the accent.
	Color ui.Color
}

// State is what a week view keeps: where it is scrolled, the event
// chosen, and the event dragged. Keep it in the app's state, one for each
// view. The zero State shows the morning from 8 o'clock.
type State struct {
	scroll  ui.ScrollState
	started bool
	chosen  int
	// drag is the event dragged, its index plus one, and how far, and
	// whether by its bottom edge.
	drag    int
	dx, dy  float32
	stretch bool
}

// Chosen returns the index of the event chosen, or -1 with none.
func (s *State) Chosen() int { return s.chosen - 1 }

// View creates a week view of days from the day of start, of events,
// which it moves in place and reports moving as changes. Give it a size,
// or grow it in its parent.
//
// A click chooses an event; dragging moves it, in steps of 15 minutes and
// to other days, and dragging its bottom edge changes its end. A line
// shows the time now on today's column.
func View(c *ui.Context, s *State, start time.Time, days int, events []Event) ui.Element {
	t := c.Theme()
	hourH := t.Space(12)
	gutter := t.Space(14)
	start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, start.Location())
	now := c.Now()
	c.After(now.Truncate(time.Minute).Add(time.Minute).Sub(now))
	if !s.started {
		s.scroll.Y, s.started = 8*hourH, true
	}
	changed := false
	view := ui.Column(c)
	view.Children(func() {
		ui.Row(c).Padding(t.Space(2), 0).Children(func() {
			ui.Box(c).Width(gutter)
			for d := range days {
				day := start.AddDate(0, 0, d)
				header(c, day, sameDay(day, now))
			}
		})
		ui.Divider(c)
		ui.Scroll(c).Grow(1).TrackScroll(&s.scroll).Children(func() {
			ui.Row(c).Height(24 * hourH).AlignItems(ui.Stretch).Children(func() {
				ui.Box(c).Width(gutter).Draw(func(p *ui.Painter, r ui.Rect) {
					for h := 1; h < 24; h++ {
						label := time.Date(2000, 1, 1, h, 0, 0, 0, time.UTC).Format("3 PM")
						w, th := c.MeasureText(0, ui.Span{Text: label, Size: t.FontSize - 2})
						p.RichText(r.X+r.W-w-t.Space(2), r.Y+float32(h)*hourH-th/2, 0, ui.Span{Text: label, Size: t.FontSize - 2, Color: t.TextMuted})
					}
				})
				for d := range days {
					day := start.AddDate(0, 0, d)
					if column(c.Key(d), s, day, d, events, hourH, now) {
						changed = true
					}
				}
			})
		})
	})
	if changed {
		view.SetChanged()
	}
	return view
}

// header shows the weekday and the date of day, the date in a circle of
// the accent today.
func header(c *ui.Context, day time.Time, today bool) {
	t := c.Theme()
	ui.Row(c).Grow(1).Basis(0).Gap(t.Space(1.5)).Justify(ui.Center).AlignItems(ui.Center).Children(func() {
		ui.Text(c, day.Format("Mon")).TextColor(t.TextMuted)
		d := t.Space(7)
		circle := ui.Row(c).Size(d, d).Radius(d / 2).Justify(ui.Center).AlignItems(ui.Center)
		circle.Children(func() {
			date := ui.Text(c, day.Format("2")).FontFeatures("tnum")
			if today {
				circle.Background(t.Danger)
				date.TextColor(ui.RGB(255, 255, 255)).Bold()
			}
		})
	})
}

func sameDay(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.In(a.Location()).Date()
	return ay == by && am == bm && ad == bd
}

// column builds the column of day, the d-th, with its hours and the
// events in it, and reports whether one moved.
func column(c *ui.Context, s *State, day time.Time, d int, events []Event, hourH float32, now time.Time) bool {
	t := c.Theme()
	col := ui.Box(c).Grow(1).Basis(0)
	w := col.LayoutBounds().W
	today := sameDay(day, now)
	col.Draw(func(p *ui.Painter, r ui.Rect) {
		line := t.Text.Alpha(0.08)
		p.Fill(ui.Rect{X: r.X, Y: r.Y, W: 1, H: r.H}, line, 0)
		for h := 1; h < 24; h++ {
			p.Fill(ui.Rect{X: r.X, Y: r.Y + float32(h)*hourH, W: r.W, H: 1}, line, 0)
		}
	})
	var in []int
	for i, e := range events {
		if sameDay(day, e.Start) {
			in = append(in, i)
		}
	}
	lanes, count := layout(events, in)
	moved := false
	col.Children(func() {
		for k, i := range in {
			if event(c.Key(i), s, &events[i], i, d, lanes[k], count[k], w, hourH, day) {
				moved = true
			}
		}
		if today {
			y := hours(now, day) * hourH
			ui.Box(c).Absolute().Left(0).Right(0).Top(y - 1).Height(2).Background(t.Danger)
			ui.Box(c).Absolute().Left(-4).Top(y-4).Size(8, 8).Radius(4).Background(t.Danger)
		}
	})
	return moved
}

// hours returns how many hours after the midnight starting day tm is.
func hours(tm, day time.Time) float32 {
	return float32(tm.Sub(day).Hours())
}

// layout places the events in of events side by side where they
// overlap: the lane of each, from the left, and how many lanes the
// cluster of events overlapping one another it is in has.
func layout(events []Event, in []int) (lanes, count []int) {
	order := make([]int, len(in))
	for k := range order {
		order[k] = k
	}
	slices.SortStableFunc(order, func(a, b int) int { return events[in[a]].Start.Compare(events[in[b]].Start) })
	lanes, count = make([]int, len(in)), make([]int, len(in))
	// ends is when the last event of each lane of the cluster ends.
	var ends []time.Time
	var cluster []int
	var end time.Time
	flush := func() {
		for _, k := range cluster {
			count[k] = len(ends)
		}
		cluster, ends = nil, nil
	}
	for _, k := range order {
		e := events[in[k]]
		if len(cluster) > 0 && !e.Start.Before(end) {
			flush()
		}
		l := slices.IndexFunc(ends, func(end time.Time) bool { return !end.After(e.Start) })
		if l < 0 {
			l, ends = len(ends), append(ends, e.End)
		} else {
			ends[l] = e.End
		}
		lanes[k] = l
		if len(cluster) == 0 || e.End.After(end) {
			end = e.End
		}
		cluster = append(cluster, k)
	}
	flush()
	return lanes, count
}

// event builds event e, the i-th, on the d-th day, in lane of count lanes
// of a column w wide, which dragging moves and its bottom edge stretches,
// and reports whether it moved.
func event(c *ui.Context, s *State, e *Event, i, d, lane, count int, w, hourH float32, day time.Time) bool {
	t := c.Theme()
	color := e.Color
	if color == (ui.Color{}) {
		color = t.Accent
	}
	laneW := max(0, w-t.Space(2)) / float32(count)
	top, bottom := hours(e.Start, day)*hourH, hours(e.End, day)*hourH
	dragged := s.drag == i+1
	x, y := float32(lane)*laneW, top
	if dragged && !s.stretch {
		x, y = x+snapX(s.dx, w), y+snapY(s.dy, hourH)
	}
	if dragged && s.stretch {
		bottom = max(top+hourH/4, bottom+snapY(s.dy, hourH))
	}
	chosen := s.chosen == i+1
	short := bottom-top < hourH*3/4
	box := ui.Column(c).Absolute().Left(x+1).Top(y+1).Width(laneW-2).Height(max(hourH/3, bottom-top)-2).
		Padding(t.Space(1), t.Space(1.5)).Radius(t.Space(1)).Clip().Role(ui.RoleButton).Label(e.Title)
	if short {
		box.Padding(0, t.Space(1.5)).Justify(ui.Center)
	}
	text := t.Text
	if chosen {
		box.Background(color)
		text = ui.RGB(255, 255, 255)
	} else {
		box.Background(color.Alpha(0.18))
	}
	box.Draw(func(p *ui.Painter, r ui.Rect) {
		p.Fill(ui.Rect{X: r.X, Y: r.Y, W: 3, H: r.H}, color, 0)
	})
	moved := false
	if dx, dy, ok := box.Dragged(); ok {
		if !dragged {
			s.drag, s.dx, s.dy, s.stretch, s.chosen = i+1, 0, 0, false, i+1
		}
		s.dx, s.dy = s.dx+dx, s.dy+dy
	} else if dragged && !s.stretch {
		days := int(math.Round(float64(s.dx / w)))
		shift := time.Duration(math.Round(float64(s.dy/hourH*4))) * snap
		if days != 0 || shift != 0 {
			e.Start, e.End = e.Start.AddDate(0, 0, days).Add(shift), e.End.AddDate(0, 0, days).Add(shift)
			moved = true
		}
		s.drag = 0
	}
	box.Children(func() {
		lines := func() {
			ui.Text(c, e.Title).Bold().FontSize(t.FontSize - 1).TextColor(text).SingleLine().Shrink(1)
			ui.Text(c, e.Start.Format("15:04")).FontSize(t.FontSize - 2).TextColor(text.Alpha(0.7)).FontFeatures("tnum").NoWrap()
		}
		// Too short for two lines, an event shows its time after its title.
		if short {
			ui.Row(c).Gap(t.Space(1)).AlignItems(ui.Center).Children(lines)
		} else {
			lines()
		}
		edge := ui.Box(c).Absolute().Left(0).Right(0).Bottom(0).Height(t.Space(1.5)).Cursor(ui.CursorResizeNS)
		if _, dy, ok := edge.Dragged(); ok {
			if !dragged {
				s.drag, s.dy, s.stretch, s.chosen = i+1, 0, true, i+1
			}
			s.dy += dy
		} else if dragged && s.stretch {
			shift := time.Duration(math.Round(float64(s.dy/hourH*4))) * snap
			if shift != 0 {
				e.End = e.End.Add(shift)
				if e.End.Before(e.Start.Add(snap)) {
					e.End = e.Start.Add(snap)
				}
				moved = true
			}
			s.drag = 0
		}
	})
	return moved
}

// snapY rounds dy to the DIPs of a step of 15 minutes, hours hourH high.
func snapY(dy, hourH float32) float32 {
	step := hourH / 4
	return float32(math.Round(float64(dy/step))) * step
}

// snapX rounds dx to whole days, columns w wide.
func snapX(dx, w float32) float32 {
	return float32(math.Round(float64(dx/max(1, w)))) * w
}
