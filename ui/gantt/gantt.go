// Package gantt shows a project's tasks over the days, as project
// planners do: a row for each task, its name beside a bar from its first
// day to its last, filled as far as it is done, or for a milestone, a
// diamond on its day; arrows from the tasks it waits for, red where it
// starts before one ends; the weekends shaded, and a line at today.
// Dragging a bar moves its task and dragging an end of it moves that
// end, by whole days; a click chooses a task, which Chosen returns.
//
//	gantt.View(c, &app.chart, &app.tasks, c.Now()).Grow(1)
//	if i, ok := app.chart.Chosen(); ok {
//		... // show app.tasks[i]
//	}
package gantt

import (
	"fmt"
	"math"
	"time"

	"github.com/egoist/mygo/ui"
)

// Task is a task of a project: its name, its first day, how many days
// it takes, 0 for a milestone, how much of it is done, from 0 to 1, and
// the indexes of the tasks it waits for.
type Task struct {
	Name  string
	Start time.Time
	Days  int
	Done  float32
	After []int
}

// State is what a chart keeps: the task chosen and the drag going on.
// Keep it in the app's state, one for each chart.
type State struct {
	// chosen is the index of the task chosen, plus 1, so that the zero
	// State chooses none.
	chosen int
	drag   drag
}

// drag is a drag of part of task's bar, by acc DIPs so far, from where
// it started.
type drag struct {
	on    bool
	task  int
	part  part
	acc   float32
	start time.Time
	days  int
}

type part uint8

const (
	whole part = iota
	first
	last
)

// Chosen returns the index of the task chosen.
func (s *State) Chosen() (int, bool) { return s.chosen - 1, s.chosen > 0 }

// day returns the midnight of t's day.
func day(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}

// between returns how many days b is after a, both midnights.
func between(a, b time.Time) int { return int(math.Round(b.Sub(a).Hours() / 24)) }

// View creates the chart of *tasks around today. Give it a size, or Grow;
// the days scroll across when they do not fit.
func View(c *ui.Context, s *State, tasks *[]Task, today time.Time) ui.Element {
	t := c.Theme()
	ts := *tasks
	today = day(today)
	from, to := today, today
	for _, tk := range ts {
		from = minTime(from, day(tk.Start))
		to = maxTime(to, day(tk.Start).AddDate(0, 0, tk.Days))
	}
	from = from.AddDate(0, 0, -3)
	n := between(from, to) + 10
	dayW, rowH, header := t.Space(7), t.Space(9), t.Space(12)
	x := func(d time.Time) float32 { return float32(between(from, day(d))) * dayW }
	return ui.Row(c).AlignItems(ui.Start).Children(func() {
		ui.Column(c).Width(t.Space(56)).Shrink(0).Padding(header, 0, 0).Children(func() {
			for i, tk := range ts {
				row := ui.ButtonBase(c.Key(i)).Height(rowH).PaddingX(t.Space(3)).Label(tk.Name)
				if s.chosen == i+1 {
					row.Background(t.Accent.Alpha(0.12))
				}
				if row.Clicked() {
					s.chosen = i + 1
				}
				row.Children(func() {
					ui.Row(c).Fill().Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
						ui.Text(c, tk.Name).SingleLine().Shrink(1).Grow(1)
						length := fmt.Sprintf("%dd", tk.Days)
						if tk.Days == 0 {
							length = "◆"
						}
						ui.Text(c, length).FontSize(t.FontSize - 2).FontFeatures("tnum").TextColor(t.TextMuted)
					})
				})
			}
		})
		ui.Divider(c).Vertical().AlignSelf(ui.Stretch)
		ui.ScrollHorizontal(c).Grow(1).Shrink(1).Children(func() {
			chart := ui.Box(c).Size(float32(n)*dayW, header+float32(len(ts))*rowH)
			chart.Children(func() {
				for i := range ts {
					s.bar(c.Key(i), ts, i, x(ts[i].Start), header+float32(i)*rowH, dayW, rowH)
				}
			})
			chosen := s.chosen - 1
			chart.Draw(func(p *ui.Painter, r ui.Rect) {
				grid(p, t, r, from, n, dayW, header)
				for i, tk := range ts {
					for _, k := range tk.After {
						if k < 0 || k >= len(ts) || k == i {
							continue
						}
						pre := ts[k]
						x0 := r.X + x(pre.Start.AddDate(0, 0, pre.Days))
						if pre.Days == 0 {
							x0 += dayW / 2
						}
						x1 := r.X + x(tk.Start)
						y0 := r.Y + header + (float32(k)+0.5)*rowH
						y1 := r.Y + header + (float32(i)+0.5)*rowH
						color := t.TextMuted.Alpha(0.7)
						if i == chosen || k == chosen {
							color = t.Accent
						}
						if day(tk.Start).Before(day(pre.Start).AddDate(0, 0, pre.Days)) {
							color = t.Danger
						}
						arrow(p, x0, y0, x1, y1, rowH, color)
					}
				}
				if k := between(from, today); k >= 0 && k < n {
					tx := r.X + float32(k)*dayW + dayW/2
					p.Line(tx, r.Y+header-t.Space(2), tx, r.Y+r.H, 2, t.Danger.Alpha(0.8))
				}
			})
		})
	})
}

// grid draws the months and days above the rows of r, n days from from,
// and the weekends shaded.
func grid(p *ui.Painter, t *ui.Theme, r ui.Rect, from time.Time, n int, dayW, header float32) {
	small := t.FontSize - 2
	month := time.Month(0)
	for k := range n {
		d := from.AddDate(0, 0, k)
		x := r.X + float32(k)*dayW
		if wd := d.Weekday(); wd == time.Saturday || wd == time.Sunday {
			p.Fill(ui.Rect{X: x, Y: r.Y + header, W: dayW, H: r.H - header}, t.Text.Alpha(0.04), 0)
		}
		if d.Month() != month {
			month = d.Month()
			p.Line(x, r.Y, x, r.Y+r.H, 1, t.Border)
			p.Text(x+t.Space(1.5), r.Y+t.Space(1), d.Format("January 2006"), small, t.Text)
		} else if d.Weekday() == time.Monday {
			p.Line(x, r.Y+header/2, x, r.Y+r.H, 1, t.Text.Alpha(0.06))
		}
		label := fmt.Sprint(d.Day())
		color := t.TextMuted
		if wd := d.Weekday(); wd == time.Saturday || wd == time.Sunday {
			color = color.Alpha(0.6)
		}
		p.Text(x+dayW/2-float32(len(label))*small*0.3, r.Y+header/2+t.Space(1), label, small, color)
	}
	p.Line(r.X, r.Y+header, r.X+r.W, r.Y+header, 1, t.Border)
}

// bar shows the bar of task i at x, y: its body moves it, its ends move
// them, and a press chooses it.
func (s *State) bar(c *ui.Context, ts []Task, i int, x, y, dayW, rowH float32) {
	t := c.Theme()
	tk := ts[i]
	h := rowH - t.Space(3)
	top := y + (rowH-h)/2
	color := t.Accent
	if tk.Done >= 1 {
		color = t.Success
	}
	if tk.Days == 0 {
		d := h * 0.8
		m := ui.Box(c).Absolute().Left(x+dayW/2-d/2).Top(y+(rowH-d)/2).Size(d, d).Cursor(ui.CursorGrab).
			Role(ui.RoleButton).Label(tk.Name)
		if m.Pressed() {
			s.chosen = i + 1
		}
		if dx, _, ok := m.Dragged(); ok {
			s.move(ts, i, whole, dx, dayW)
		}
		s.end(m)
		chosen := s.chosen == i+1
		m.Draw(func(p *ui.Painter, r ui.Rect) {
			var dia ui.Path
			cx, cy, k := r.X+r.W/2, r.Y+r.H/2, r.W/2
			dia.MoveTo(cx, cy-k).LineTo(cx+k, cy).LineTo(cx, cy+k).LineTo(cx-k, cy).Close()
			p.FillPath(&dia, t.Warning)
			if chosen {
				p.StrokePath(&dia, 2, t.Text)
			}
		})
		return
	}
	w := float32(tk.Days)*dayW - t.Space(1)
	body := ui.Box(c).Absolute().Left(x+t.Space(0.5)).Top(top).Size(w, h).Radius(t.Space(1.5)).
		Background(color.Alpha(0.25)).Cursor(ui.CursorGrab).Role(ui.RoleButton).Label(tk.Name)
	if s.chosen == i+1 {
		body.Border(2, color)
	}
	if body.Pressed() {
		s.chosen = i + 1
	}
	if dx, _, ok := body.Dragged(); ok {
		s.move(ts, i, whole, dx, dayW)
	}
	s.end(body)
	done := max(0, min(1, tk.Done))
	body.Draw(func(p *ui.Painter, r ui.Rect) {
		p.Fill(ui.Rect{X: r.X, Y: r.Y, W: r.W * done, H: r.H}, color, t.Space(1.5))
	})
	body.Children(func() {
		grip := t.Space(2)
		for k, pt := range []part{first, last} {
			e := ui.Box(c.Key(k)).Absolute().Top(0).Size(grip, h).Cursor(ui.CursorResizeEW)
			if pt == first {
				e.Left(0).Label("Start of " + tk.Name)
			} else {
				e.Left(w - grip).Label("End of " + tk.Name)
			}
			if e.Pressed() {
				s.chosen = i + 1
			}
			if dx, _, ok := e.Dragged(); ok {
				s.move(ts, i, pt, dx, dayW)
			}
			s.end(e)
		}
	})
}

// move moves part of task i's bar by dx more, to the whole day nearest.
func (s *State) move(ts []Task, i int, pt part, dx, dayW float32) {
	if !s.drag.on || s.drag.task != i || s.drag.part != pt {
		s.drag = drag{on: true, task: i, part: pt, start: ts[i].Start, days: ts[i].Days}
	}
	s.drag.acc += dx
	k := int(math.Round(float64(s.drag.acc / dayW)))
	tk := &ts[i]
	switch pt {
	case whole:
		tk.Start = s.drag.start.AddDate(0, 0, k)
	case first:
		k = min(k, s.drag.days-1)
		tk.Start, tk.Days = s.drag.start.AddDate(0, 0, k), s.drag.days-k
	case last:
		tk.Days = max(1, s.drag.days+k)
	}
}

// end ends the drag when e lets go.
func (s *State) end(e ui.Element) {
	if _, _, ok := e.DragEnded(); ok {
		s.drag = drag{}
	}
}

// arrow draws a link from the end of a task at x0, y0 to the start of
// one at x1, y1, round the bars where the second starts before the first
// ends.
func arrow(p *ui.Painter, x0, y0, x1, y1, rowH float32, color ui.Color) {
	const out, head = 8, 5
	pts := [][2]float32{{x0, y0}, {x0 + out, y0}}
	if x1-out >= x0+out {
		pts = append(pts, [2]float32{x0 + out, y1})
	} else {
		mid := y0 + rowH/2
		if y1 < y0 {
			mid = y0 - rowH/2
		}
		pts = append(pts, [2]float32{x0 + out, mid}, [2]float32{x1 - out, mid}, [2]float32{x1 - out, y1})
	}
	pts = append(pts, [2]float32{x1 - 1, y1})
	p.StrokePolyline(pts, 1.5, color)
	var tip ui.Path
	tip.MoveTo(x1, y1).LineTo(x1-head, y1-head*0.7).LineTo(x1-head, y1+head*0.7).Close()
	p.FillPath(&tip, color)
}

func minTime(a, b time.Time) time.Time {
	if b.Before(a) {
		return b
	}
	return a
}

func maxTime(a, b time.Time) time.Time {
	if b.After(a) {
		return b
	}
	return a
}
