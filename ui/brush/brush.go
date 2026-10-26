// Package brush shows counts over time as bars, as log explorers such
// as Kibana and Grafana show events above the list of them, and chooses
// a window of time on them by dragging: across the bars to choose one,
// inside it to move it, and on its edges to widen or narrow it, always
// by whole bars. The bars outside the window are dimmed, a double click
// lets the window go, and a line above tells what it holds. The pointer
// on a bar tells its time and count. Several series stack, each in its
// color, with a legend.
//
//	if brush.View(c, &app.brush, app.start, time.Minute, app.series, &app.window).Changed() {
//		app.query(app.window)
//	}
package brush

import (
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/egoist/mygo/ui"
)

// Series is counts in a bar each, stacked over those of the series
// before it, named and in a color: the theme's accent when it is zero.
type Series struct {
	Name   string
	Color  ui.Color
	Counts []int
}

// Range is a window of time from From up to To, all of it when zero.
type Range struct {
	From, To time.Time
}

// IsZero reports whether r is no window: all of the time.
func (r Range) IsZero() bool { return r.From.IsZero() && r.To.IsZero() }

// State is what a view keeps: the drag going on. Keep it in the app's
// state, one for each view.
type State struct {
	drag   drag
	anchor int     // the bar edge a new window started at
	grab   float32 // where the window was taken, in bars from its start
	was    [2]int  // the window's bar edges as the drag started
}

// drag is what dragging does.
type drag uint8

const (
	none drag = iota
	making
	moving
	fromEdge
	toEdge
)

// View creates the bars of series, counted in bars of step from start,
// and the window *window chosen on them. Changed reports a new window,
// as a drag ends.
func View(c *ui.Context, s *State, start time.Time, step time.Duration, series []Series, window *Range) ui.Element {
	t := c.Theme()
	n := 0
	for _, se := range series {
		n = max(n, len(se.Counts))
	}
	totals := make([]int, n)
	for _, se := range series {
		for i, v := range se.Counts {
			totals[i] += v
		}
	}
	peak := 0
	for _, v := range totals {
		peak = max(peak, v)
	}
	// edge returns the bar edge of tm, from 0 to n.
	edge := func(tm time.Time) int {
		return max(0, min(n, int(math.Round(float64(tm.Sub(start))/float64(step)))))
	}
	at := func(e int) time.Time { return start.Add(time.Duration(e) * step) }
	win := [2]int{0, n}
	if !window.IsZero() {
		win = [2]int{edge(window.From), edge(window.To)}
	}
	changed := false
	small := t.FontSize - 2
	axisW := t.Space(10)
	axisH := small*1.4 + t.Space(1)
	el := ui.Column(c).Gap(t.Space(2))
	el.Children(func() {
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			sum := 0
			for _, v := range totals[win[0]:win[1]] {
				sum += v
			}
			from, to := at(win[0]), at(win[1])
			ui.Text(c, fmt.Sprintf("%s events", group(sum))).Bold().FontFeatures("tnum")
			ui.Text(c, span(from, to)).FontSize(t.FontSize - 1).TextColor(t.TextMuted).FontFeatures("tnum").Shrink(1).SingleLine()
			ui.Spacer(c)
			if len(series) > 1 {
				for i, se := range series {
					ui.Row(c.Key(i)).Gap(t.Space(1.5)).AlignItems(ui.Center).Children(func() {
						ui.Box(c).Size(t.Space(2.5), t.Space(2.5)).Radius(2).Background(colorOf(t, se.Color))
						ui.Text(c, se.Name).FontSize(t.FontSize - 1).TextColor(t.TextMuted)
					})
				}
			}
			if !window.IsZero() && ui.Button(c, "Clear").Clicked() {
				*window = Range{}
				changed = true
			}
		})
		box := ui.Box(c).AlignSelf(ui.Stretch).Height(t.Space(30)).Role(ui.RoleImage).
			Label("Events over time").Description(span(at(win[0]), at(win[1])))
		r := box.LayoutBounds()
		plotW := max(r.W-axisW, 1)
		bw := plotW / float32(max(n, 1))
		x, y, in := box.PointerPosition()
		px := x - axisW
		edgeAt := func(px float32) int { return max(0, min(n, int(math.Round(float64(px/bw))))) }
		barAt := func(px float32) int { return max(0, min(n-1, int(px/bw))) }
		chosen := !window.IsZero()
		near := func(e int) bool { return chosen && abs(px-float32(e)*bw) < t.Space(1.5) }
		switch {
		case s.drag != none:
		case in && near(win[0]), in && near(win[1]):
			box.Cursor(ui.CursorResizeEW)
		case in && chosen && px > float32(win[0])*bw && px < float32(win[1])*bw:
			box.Cursor(ui.CursorGrab)
		case in:
			box.Cursor(ui.CursorCrosshair)
		}
		if in && s.drag == none && n > 0 && px >= 0 && y < r.H-axisH {
			i := barAt(px)
			box.Tooltip(fmt.Sprintf("%s · %s", span(at(i), at(i+1)), group(totals[i])))
		}
		switch {
		case box.DoubleClicked() && chosen:
			*window = Range{}
			changed = true
		case box.Pressed() && n > 0:
			if s.drag == none {
				s.was = win
				switch {
				case near(win[0]):
					s.drag = fromEdge
				case near(win[1]):
					s.drag = toEdge
				case chosen && px > float32(win[0])*bw && px < float32(win[1])*bw:
					s.drag, s.grab = moving, px/bw-float32(win[0])
				default:
					s.drag, s.anchor = making, edgeAt(px)
				}
			}
			e := edgeAt(px)
			switch s.drag {
			case making:
				win = [2]int{min(s.anchor, e), max(s.anchor, e)}
			case fromEdge:
				win = [2]int{min(e, s.was[1]-1), s.was[1]}
			case toEdge:
				win = [2]int{s.was[0], max(e, s.was[0]+1)}
			case moving:
				w := s.was[1] - s.was[0]
				from := max(0, min(n-w, int(math.Round(float64(px/bw-s.grab)))))
				win = [2]int{from, from + w}
			}
			if win[0] < win[1] && (s.drag != making || win != s.was) {
				*window = Range{at(win[0]), at(win[1])}
			}
			box.Cursor(map[drag]ui.Cursor{making: ui.CursorCrosshair, moving: ui.CursorGrabbing, fromEdge: ui.CursorResizeEW, toEdge: ui.CursorResizeEW}[s.drag])
		case s.drag != none:
			changed = s.was != win || s.drag == making && win[0] < win[1]
			s.drag = none
		}
		chosen = !window.IsZero()
		box.Draw(func(p *ui.Painter, r ui.Rect) {
			plot := ui.Rect{X: r.X + axisW, Y: r.Y + t.Space(1), W: r.W - axisW, H: r.H - axisH - t.Space(1)}
			p.Line(plot.X, plot.Y+plot.H, plot.X+plot.W, plot.Y+plot.H, 1, t.Border)
			top := nice(peak)
			for _, f := range []float32{0.5, 1} {
				gy := plot.Y + plot.H*(1-f)
				p.Line(plot.X, gy, plot.X+plot.W, gy, 1, t.Text.Alpha(0.06))
				label := short(int(float32(top) * f))
				w, _ := p.MeasureText(0, ui.Span{Text: label, Size: small})
				p.Text(plot.X-w-t.Space(1.5), gy-small*0.6, label, small, t.TextMuted)
			}
			if n == 0 || top == 0 {
				return
			}
			bw := plot.W / float32(n)
			gap := min(1, bw*0.2)
			for i := range n {
				dim := chosen && (i < win[0] || i >= win[1])
				base := plot.Y + plot.H
				for _, se := range series {
					if i >= len(se.Counts) || se.Counts[i] == 0 {
						continue
					}
					h := plot.H * float32(se.Counts[i]) / float32(top)
					col := colorOf(t, se.Color)
					if dim {
						col = col.Alpha(0.25)
					}
					p.Fill(ui.Rect{X: plot.X + float32(i)*bw + gap/2, Y: base - h, W: bw - gap, H: h}, col, 0)
					base -= h
				}
			}
			if chosen {
				x0, x1 := plot.X+float32(win[0])*bw, plot.X+float32(win[1])*bw
				p.Fill(ui.Rect{X: x0, Y: plot.Y, W: x1 - x0, H: plot.H}, t.Accent.Alpha(0.08), 0)
				for _, x := range []float32{x0, x1} {
					p.Line(x, plot.Y, x, plot.Y+plot.H, 1.5, t.Accent)
					p.Fill(ui.Rect{X: x - 3, Y: plot.Y + plot.H/2 - 10, W: 6, H: 20}, t.Accent, 3)
				}
			}
			ticks(p, t, plot, start, step, n, small)
		})
	})
	if changed {
		el.SetChanged()
	}
	return el
}

// ticks draws the times under plot, as far apart as fits labels.
func ticks(p *ui.Painter, t *ui.Theme, plot ui.Rect, start time.Time, step time.Duration, n int, size float32) {
	total := time.Duration(n) * step
	every := time.Duration(0)
	for _, d := range []time.Duration{
		time.Second, 5 * time.Second, 15 * time.Second, 30 * time.Second,
		time.Minute, 5 * time.Minute, 15 * time.Minute, 30 * time.Minute,
		time.Hour, 3 * time.Hour, 6 * time.Hour, 12 * time.Hour, 24 * time.Hour, 7 * 24 * time.Hour,
	} {
		every = d
		if plot.W*float32(d)/float32(total) >= t.Space(22) {
			break
		}
	}
	layout := "15:04"
	switch {
	case every < time.Minute:
		layout = "15:04:05"
	case every >= 24*time.Hour:
		layout = "Jan 2"
	}
	first := start.Truncate(every)
	if first.Before(start) {
		first = first.Add(every)
	}
	for tm := first; !tm.After(start.Add(total)); tm = tm.Add(every) {
		x := plot.X + plot.W*float32(tm.Sub(start))/float32(total)
		p.Line(x, plot.Y+plot.H, x, plot.Y+plot.H+3, 1, t.Border)
		label := tm.Format(layout)
		if layout != "Jan 2" && tm.Hour() == 0 && tm.Minute() == 0 && tm.Second() == 0 {
			label = tm.Format("Jan 2")
		}
		w, _ := p.MeasureText(0, ui.Span{Text: label, Size: size})
		if x-w/2 < plot.X || x+w/2 > plot.X+plot.W {
			continue
		}
		p.Text(x-w/2, plot.Y+plot.H+t.Space(1), label, size, t.TextMuted)
	}
}

// colorOf returns col, or the theme's accent when it is zero.
func colorOf(t *ui.Theme, col ui.Color) ui.Color {
	if col == (ui.Color{}) {
		return t.Accent
	}
	return col
}

// nice returns a round number at least v: 1, 2 or 5 times a power of 10.
func nice(v int) int {
	if v <= 0 {
		return 0
	}
	p := int(math.Pow(10, math.Floor(math.Log10(float64(v)))))
	for _, m := range []int{1, 2, 5, 10} {
		if m*p >= v {
			return m * p
		}
	}
	return 10 * p
}

// span writes from up to to: "Oct 26, 10:12:00 – 10:18:00 (6m)".
func span(from, to time.Time) string {
	layout := "Jan 2, 15:04:05"
	end := "15:04:05"
	if from.YearDay() != to.Add(-time.Nanosecond).YearDay() {
		end = layout
	}
	return fmt.Sprintf("%s – %s (%s)", from.Format(layout), to.Format(end), length(to.Sub(from)))
}

// length writes d short: "45s", "6m", "1h 30m", "2d 4h".
func length(d time.Duration) string {
	switch {
	case d < time.Minute:
		return strconv.Itoa(int(d.Seconds())) + "s"
	case d < time.Hour:
		if s := int(d.Seconds()) % 60; s != 0 {
			return fmt.Sprintf("%dm %ds", int(d.Minutes()), s)
		}
		return strconv.Itoa(int(d.Minutes())) + "m"
	case d < 24*time.Hour:
		if m := int(d.Minutes()) % 60; m != 0 {
			return fmt.Sprintf("%dh %dm", int(d.Hours()), m)
		}
		return strconv.Itoa(int(d.Hours())) + "h"
	}
	days := int(d.Hours()) / 24
	if h := int(d.Hours()) % 24; h != 0 {
		return fmt.Sprintf("%dd %dh", days, h)
	}
	return strconv.Itoa(days) + "d"
}

// short writes n short: 950, 1.2k, 34k, 1.5M.
func short(n int) string {
	switch {
	case n < 1000:
		return strconv.Itoa(n)
	case n < 10000:
		return strconv.FormatFloat(math.Round(float64(n)/100)/10, 'f', -1, 64) + "k"
	case n < 1000000:
		return strconv.Itoa(n/1000) + "k"
	}
	return strconv.FormatFloat(math.Round(float64(n)/100000)/10, 'f', -1, 64) + "M"
}

// group writes n with commas: 12,400.
func group(n int) string {
	s := strconv.Itoa(n)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}

func abs(v float32) float32 { return max(v, -v) }
