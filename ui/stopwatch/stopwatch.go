// Package stopwatch times things, as the Clock app does: a stopwatch
// with laps, and a timer counting down. The stopwatch starts and stops
// with its right button, and its left one marks a lap while it runs and
// resets it while stopped; the laps list below, the newest first, the
// fastest in green and the slowest in red once there are three. The
// timer is set with hours, minutes and seconds or a preset, and counts
// down around a ring; Finished reports the frame it reaches zero.
//
//	stopwatch.View(c, &app.watch)
//	if stopwatch.Timer(c, &app.timer).Finished {
//		app.ring()
//	}
package stopwatch

import (
	"fmt"
	"math"
	"time"

	"github.com/egoist/mygo/ui"
)

// State is what a stopwatch keeps: the time it ran before it last
// started, when that was, and its laps. Keep it in the app's state, one
// for each stopwatch.
type State struct {
	before  time.Duration
	started time.Time
	running bool
	laps    []time.Duration
}

// Elapsed returns how long the stopwatch has run at now.
func (s *State) Elapsed(now time.Time) time.Duration {
	if s.running {
		return s.before + now.Sub(s.started)
	}
	return s.before
}

// Laps returns the laps' times, the first lap first.
func (s *State) Laps() []time.Duration { return s.laps }

// clock returns d as minutes, seconds and hundredths, with hours past an
// hour.
func clock(d time.Duration, hundredths bool) string {
	cs := int(d / (10 * time.Millisecond))
	h, m, sec := cs/360000, cs/6000%60, cs/100%60
	out := fmt.Sprintf("%02d:%02d", m, sec)
	if h > 0 {
		out = fmt.Sprintf("%d:%s", h, out)
	}
	if hundredths {
		out += fmt.Sprintf(".%02d", cs%100)
	}
	return out
}

// round creates a round button showing label in col, and reports a
// click.
func round(c *ui.Context, label string, col ui.Color, disabled bool) bool {
	t := c.Theme()
	b := ui.ButtonBase(c).Size(t.Space(20), t.Space(20)).Radius(t.Space(10)).Background(col.Alpha(0.16)).Label(label).Disabled(disabled)
	if b.Hovered() {
		b.Background(col.Alpha(0.24))
	}
	b.Children(func() {
		ui.Text(c, label).TextColor(col).Bold().Center()
	})
	return b.Clicked()
}

// digits creates the time f returns as large digits, painted again while
// live.
func digits(c *ui.Context, size float32, live bool, f func(now time.Time) string) ui.Element {
	t := c.Theme()
	return ui.Box(c).Height(size * 1.3).FillWidth().Draw(func(p *ui.Painter, r ui.Rect) {
		span := ui.Span{Text: f(p.Now()), Size: size, Weight: 300, Color: t.Text, Features: "tnum"}
		w, h := p.MeasureText(0, span)
		p.RichText(r.X+r.W/2-w/2, r.Y+r.H/2-h/2, 0, span)
		if live {
			p.After(10 * time.Millisecond)
		}
	})
}

// View creates the stopwatch. Changed reports a start, stop, lap or
// reset.
func View(c *ui.Context, s *State) ui.Element {
	t := c.Theme()
	root := ui.Column(c).Gap(t.Space(6)).AlignItems(ui.Center)
	changed := false
	root.Children(func() {
		running := s.running
		digits(c, 64, running, func(now time.Time) string { return clock(s.Elapsed(now), true) })
		ui.Row(c).FillWidth().Children(func() {
			left := "Lap"
			if !s.running && s.before > 0 {
				left = "Reset"
			}
			if round(c, left, t.Text, !s.running && s.before == 0) {
				now := c.Now()
				if s.running {
					var sum time.Duration
					for _, l := range s.laps {
						sum += l
					}
					s.laps = append(s.laps, s.Elapsed(now)-sum)
				} else {
					*s = State{}
				}
				changed = true
			}
			ui.Spacer(c)
			right, col := "Start", t.Success
			if s.running {
				right, col = "Stop", t.Danger
			}
			if round(c, right, col, false) {
				now := c.Now()
				if s.running {
					s.before, s.running = s.Elapsed(now), false
				} else {
					s.started, s.running = now, true
				}
				changed = true
			}
		})
		s.lapList(c)
	})
	if changed {
		root.SetChanged()
	}
	return root
}

// lapList creates the laps, the one running on top.
func (s *State) lapList(c *ui.Context) {
	t := c.Theme()
	if len(s.laps) == 0 && !s.running {
		return
	}
	fast, slow := -1, -1
	if len(s.laps) >= 3 {
		fast, slow = 0, 0
		for i, l := range s.laps {
			if l < s.laps[fast] {
				fast = i
			}
			if l > s.laps[slow] {
				slow = i
			}
		}
	}
	var sum time.Duration
	for _, l := range s.laps {
		sum += l
	}
	row := func(key string, name string, col ui.Color, lap, total func(now time.Time) string, live bool) {
		ui.Row(c.Key(key)).FillWidth().Padding(t.Space(2.5), 0).BorderWidth(0, 0, 1, 0).BorderColor(t.Border).Children(func() {
			ui.Text(c, name).TextColor(col).Grow(1)
			ui.Box(c).Width(t.Space(24)).Height(t.FontSize * 1.4).Draw(func(p *ui.Painter, r ui.Rect) {
				for i, f := range []func(time.Time) string{lap, total} {
					span := ui.Span{Text: f(p.Now()), Color: col, Features: "tnum"}
					if i == 1 {
						span.Color = t.TextMuted
					}
					w, h := p.MeasureText(0, span)
					x := r.X + r.W - w
					if i == 0 {
						x -= t.Space(26)
					}
					p.RichText(x, r.Y+r.H/2-h/2, 0, span)
				}
				if live {
					p.After(10 * time.Millisecond)
				}
			})
		})
	}
	ui.Scroll(c).FillWidth().Grow(1).Children(func() {
		ui.Column(c).FillWidth().Children(func() {
			if s.running || s.before > sum {
				n := len(s.laps) + 1
				row("live", fmt.Sprintf("Lap %d", n), t.Text,
					func(now time.Time) string { return clock(s.Elapsed(now)-sum, true) },
					func(now time.Time) string { return clock(s.Elapsed(now), true) }, s.running)
			}
			total := sum
			for i := len(s.laps) - 1; i >= 0; i-- {
				col := t.Text
				switch i {
				case fast:
					col = t.Success
				case slow:
					col = t.Danger
				}
				lap, at := s.laps[i], total
				row(fmt.Sprint(i), fmt.Sprintf("Lap %d", i+1), col,
					func(time.Time) string { return clock(lap, true) },
					func(time.Time) string { return clock(at, true) }, false)
				total -= lap
			}
		})
	})
}

// TimerState is what a timer keeps: how long it is set for, and while
// it runs, when it ends. Keep it in the app's state, one for each timer.
type TimerState struct {
	// Duration is how long the timer is set for, which the app may preset.
	Duration time.Duration
	end      time.Time
	left     time.Duration
	running  bool
	paused   bool
	rang     bool
}

// Remaining returns how long the timer has left at now, 0 when it is not
// set going.
func (s *TimerState) Remaining(now time.Time) time.Duration {
	switch {
	case s.running:
		return max(0, s.end.Sub(now))
	case s.paused:
		return s.left
	}
	return 0
}

// TimerEvents is a timer, and whether it reached zero this frame.
type TimerEvents struct {
	ui.Element
	Finished bool
}

var presets = []time.Duration{time.Minute, 3 * time.Minute, 5 * time.Minute, 10 * time.Minute, 25 * time.Minute}

// Timer creates the timer: while set going, the time left around a ring
// that empties; while not, fields to set it.
func Timer(c *ui.Context, s *TimerState) TimerEvents {
	t := c.Theme()
	now := c.Now()
	ev := TimerEvents{}
	if s.running && !now.Before(s.end) {
		s.running, s.rang, ev.Finished = false, true, true
	}
	if s.running {
		c.After(s.end.Sub(now))
	}
	ev.Element = ui.Column(c).Gap(t.Space(6)).AlignItems(ui.Center).Children(func() {
		going := s.running || s.paused
		if going {
			total, running := s.Duration, s.running
			ui.Box(c).Size(t.Space(64), t.Space(64)).Draw(func(p *ui.Painter, r ui.Rect) {
				left := s.Remaining(p.Now())
				frac := float32(left) / float32(max(1, total))
				cx, cy, rad := r.X+r.W/2, r.Y+r.H/2, min(r.W, r.H)/2-6
				var ring ui.Path
				p.StrokePath(ring.Circle(cx, cy, rad), 6, t.Text.Alpha(0.1))
				if frac > 0 {
					pts := make([][2]float32, 0, 121)
					for i := range 121 {
						a := -math.Pi/2 + 2*math.Pi*float64(frac)*float64(i)/120
						pts = append(pts, [2]float32{cx + rad*float32(math.Cos(a)), cy + rad*float32(math.Sin(a))})
					}
					p.StrokePolyline(pts, 6, ui.RGB(255, 159, 10))
				}
				// The time left rounds up, so that 0:00 shows only at the end.
				span := ui.Span{Text: clock((left+time.Second-1)/time.Second*time.Second, false), Size: 52, Weight: 300, Color: t.Text, Features: "tnum"}
				w, h := p.MeasureText(0, span)
				p.RichText(cx-w/2, cy-h/2, 0, span)
				if running {
					p.After(time.Second / 30)
				}
			})
		} else {
			if s.rang {
				ui.Text(c, "Done").FontSize(t.FontSize + 10).Bold().TextColor(ui.RGB(255, 159, 10))
			}
			ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
				parts := []struct {
					unit time.Duration
					n    int
					name string
				}{{time.Hour, 24, "h"}, {time.Minute, 60, "min"}, {time.Second, 60, "s"}}
				for _, part := range parts {
					options := make([]string, part.n)
					for i := range options {
						options[i] = fmt.Sprintf("%d %s", i, part.name)
					}
					v := s.Duration / part.unit % time.Duration(part.n)
					cur := options[v]
					if ui.Select(c.Key(part.name), &cur, options).Width(t.Space(24)).Label(part.name).Changed() {
						var n int
						fmt.Sscanf(cur, "%d", &n)
						s.Duration += time.Duration(n)*part.unit - v*part.unit
					}
				}
			})
			ui.Row(c).Gap(t.Space(2)).Children(func() {
				for _, d := range presets {
					if ui.Button(c.Key(d), fmt.Sprintf("%d min", int(d.Minutes()))).Clicked() {
						s.Duration = d
					}
				}
			})
		}
		ui.Row(c).Width(t.Space(64)).Children(func() {
			if round(c, "Cancel", t.Text, !going) {
				s.running, s.paused, s.rang = false, false, false
			}
			ui.Spacer(c)
			switch {
			case s.running:
				if round(c, "Pause", ui.RGB(255, 159, 10), false) {
					s.left, s.running, s.paused = s.end.Sub(now), false, true
				}
			case s.paused:
				if round(c, "Resume", t.Success, false) {
					s.end, s.running, s.paused = now.Add(s.left), true, false
				}
			default:
				if round(c, "Start", t.Success, s.Duration <= 0) {
					s.end, s.running, s.rang = now.Add(s.Duration), true, false
				}
			}
		})
	})
	return ev
}
