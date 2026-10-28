// Package scheduler books resources by the hours of a day, as room
// booking tools do: a row a resource, as a meeting room or a person, and
// the hours across. Dragging along an empty stretch of a row books it, in
// steps of 15 minutes; dragging a booking moves it in time and to another
// row, and dragging either end changes when it starts or ends. Bookings
// that overlap in a row show in red. A click chooses a booking, Delete
// removes it and Escape lets it go.
//
//	if scheduler.View(c, &app.sched, &app.bookings, rooms, 8, 20).Changed() {
//		app.save()
//	}
package scheduler

import (
	"fmt"
	"math"
	"time"

	"github.com/egoist/mygo/ui"
)

// snap is the step bookings start, end and move by.
const snap = 15 * time.Minute

// Booking is a resource taken for a stretch of the day.
type Booking struct {
	Title string
	// Resource indexes the resources of the view.
	Resource int
	// Start and End are times of the day, from midnight.
	Start, End time.Duration
}

type mode uint8

const (
	idle mode = iota
	booking
	moving
	startEdge
	endEdge
)

// State is what a view keeps: the booking chosen and the drag going on.
// Keep it in the app's state, one for each view.
type State struct {
	chosen int
	down   bool
	mode   mode
	// at is the time and the row the drag started at, and was the booking
	// dragged as it was then.
	at  time.Duration
	row int
	was Booking
}

// Chosen returns the index of the booking chosen, or -1 with none.
func (s *State) Chosen() int { return s.chosen - 1 }

// grid is where a view of a box w wide puts its hours and rows.
type grid struct {
	names, header, rowH, hourW float32
	from, to                   int
}

func (g grid) x(d time.Duration) float32 {
	return g.names + float32(d.Hours()-float64(g.from))*g.hourW
}

func (g grid) time(x float32) time.Duration {
	return time.Duration((float64(g.from) + float64((x-g.names)/g.hourW)) * float64(time.Hour))
}

func (g grid) rect(b Booking) ui.Rect {
	x0, x1 := g.x(b.Start), g.x(b.End)
	return ui.Rect{X: x0, Y: g.header + float32(b.Resource)*g.rowH + 3, W: x1 - x0, H: g.rowH - 6}
}

func round(d time.Duration) time.Duration {
	return time.Duration(math.Round(float64(d)/float64(snap))) * snap
}

// clash reports, for each booking, whether another in its row overlaps
// it.
func clash(bookings []Booking) []bool {
	out := make([]bool, len(bookings))
	for i, a := range bookings {
		for j, b := range bookings {
			if i != j && a.Resource == b.Resource && a.Start < b.End && b.Start < a.End {
				out[i] = true
			}
		}
	}
	return out
}

// View creates a view of resources for the hours from to to of a day,
// with bookings, which the user adds to, moves, stretches and removes in
// place, reported as changes. It is as wide as its parent.
func View(c *ui.Context, s *State, bookings *[]Booking, resources []string, from, to int) ui.Element {
	t := c.Theme()
	g := grid{names: t.Space(30), header: t.Space(8), rowH: t.Space(12), from: from, to: max(to, from+1)}
	lo, hi := time.Duration(g.from)*time.Hour, time.Duration(g.to)*time.Hour
	canvas := ui.Box(c).AlignSelf(ui.Stretch).Height(g.header + g.rowH*float32(len(resources))).
		Focusable().FocusRing(false).Role(ui.RoleImage).
		Label(fmt.Sprintf("Schedule of %d resources, %d bookings", len(resources), len(*bookings)))
	g.hourW = max(1, canvas.LayoutBounds().W-g.names) / float32(g.to-g.from)
	bs := *bookings
	changed := false
	if s.chosen > len(bs) {
		s.chosen = 0
	}

	x, y, over := canvas.PointerPosition()
	row := min(max(int(math.Floor(float64((y-g.header)/g.rowH))), 0), len(resources)-1)
	at := g.time(x)
	edge := t.Space(1.5)
	// hit returns the booking under the pointer, the last drawn on top, and
	// which part of it.
	hit := func() (int, mode) {
		for i := len(bs) - 1; i >= 0; i-- {
			r := g.rect(bs[i])
			if !r.Contains(x, y) {
				continue
			}
			switch {
			case x < r.X+edge:
				return i, startEdge
			case x > r.X+r.W-edge:
				return i, endEdge
			}
			return i, moving
		}
		return -1, idle
	}

	pressed := canvas.Pressed() && len(resources) > 0
	if pressed && !s.down {
		s.mode, s.chosen = idle, 0
		if y > g.header && x > g.names {
			s.at, s.row = round(at), row
			i, m := hit()
			if i >= 0 {
				s.chosen, s.mode, s.was = i+1, m, bs[i]
			} else {
				s.mode = booking
			}
		}
	}
	if _, _, dragged := canvas.Dragged(); pressed && dragged && s.mode != idle {
		now := min(max(round(at), lo), hi)
		var b Booking
		switch s.mode {
		case booking:
			b = Booking{Resource: s.row, Start: min(s.at, now), End: max(s.at, now)}
			if b.End == b.Start {
				b.End = b.Start + snap
			}
			if s.chosen == 0 {
				bs = append(bs, b)
				s.chosen = len(bs)
			}
		case moving:
			b = s.was
			shift := min(max(round(at)-s.at, lo-b.Start), hi-b.End)
			b.Start, b.End = b.Start+shift, b.End+shift
			b.Resource = min(max(b.Resource+row-s.row, 0), len(resources)-1)
		case startEdge:
			b = s.was
			b.Start = min(now, b.End-snap)
		case endEdge:
			b = s.was
			b.End = max(now, b.Start+snap)
		}
		if bs[s.chosen-1] != b {
			bs[s.chosen-1] = b
			changed = true
		}
	}
	if !pressed {
		s.mode = idle
	}
	s.down = pressed

	switch {
	case s.chosen > 0 && (canvas.Shortcut(0, ui.KeyDelete) || canvas.Shortcut(0, ui.KeyBackspace)):
		bs = append(bs[:s.chosen-1], bs[s.chosen:]...)
		s.chosen, changed = 0, true
	case canvas.Shortcut(0, ui.KeyEscape):
		s.chosen = 0
	}
	*bookings = bs

	m := s.mode
	if m == idle && over {
		_, m = hit()
	}
	cursor := ui.CursorResizeEW
	switch {
	case m == idle:
		cursor = ui.CursorDefault
	case m == moving && s.mode == moving:
		cursor = ui.CursorGrabbing
	case m == moving:
		cursor = ui.CursorGrab
	}
	canvas.Cursor(cursor)

	shown := bs
	clashes := clash(shown)
	chosen := s.chosen - 1
	canvas.Draw(func(p *ui.Painter, r ui.Rect) {
		fs := t.FontSize - 2
		line := t.Border.Alpha(0.6)
		for h := g.from; h <= g.to; h++ {
			hx := r.X + g.x(time.Duration(h)*time.Hour)
			p.Fill(ui.Rect{X: hx, Y: r.Y + g.header, W: 1, H: r.H - g.header}, line, 0)
			if h < g.to {
				label := fmt.Sprintf("%d:00", h)
				_, th := p.MeasureText(0, ui.Span{Text: label, Size: fs})
				p.Text(hx+t.Space(1), r.Y+(g.header-th)/2, label, fs, t.TextMuted)
			}
		}
		for i, name := range resources {
			ry := r.Y + g.header + float32(i)*g.rowH
			p.Fill(ui.Rect{X: r.X, Y: ry, W: r.W, H: 1}, line, 0)
			_, th := p.MeasureText(0, ui.Span{Text: name, Size: t.FontSize})
			p.Clip(ui.Rect{X: r.X, Y: ry, W: g.names - t.Space(2), H: g.rowH}, 0, func() {
				p.Text(r.X+t.Space(2), ry+(g.rowH-th)/2, name, t.FontSize, t.Text)
			})
		}
		p.Fill(ui.Rect{X: r.X, Y: r.Y + r.H - 1, W: r.W, H: 1}, line, 0)
		for i, b := range shown {
			br := g.rect(b)
			br.X += r.X
			br.Y += r.Y
			color := t.Accent
			if clashes[i] {
				color = t.Danger
			}
			text := t.Text
			if i == chosen {
				p.Fill(br, color, t.Radius)
				text = t.AccentText
			} else {
				p.Fill(br, color.Alpha(0.18), t.Radius)
				p.Fill(ui.Rect{X: br.X, Y: br.Y, W: 3, H: br.H}, color, 0)
			}
			span := fmt.Sprintf("%s–%s", clock(b.Start), clock(b.End))
			p.Clip(br, t.Radius, func() {
				tx := br.X + t.Space(2)
				_, th := p.MeasureText(0, ui.Span{Text: span, Size: fs})
				if b.Title == "" {
					p.Text(tx, br.Y+(br.H-th)/2, span, fs, text.Alpha(0.75))
					return
				}
				_, nh := p.MeasureText(0, ui.Span{Text: b.Title, Size: fs + 1})
				top := br.Y + (br.H-nh-th)/2
				p.RichText(tx, top, 0, ui.Span{Text: b.Title, Size: fs + 1, Color: text, Weight: 600})
				p.Text(tx, top+nh, span, fs, text.Alpha(0.75))
			})
		}
	})
	if changed {
		canvas.SetChanged()
	}
	return canvas
}

// clock writes a time of the day as 9:30.
func clock(d time.Duration) string {
	return fmt.Sprintf("%d:%02d", int(d.Hours()), int(d.Minutes())%60)
}
