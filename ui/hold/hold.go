// Package hold confirms a dangerous action by holding or sliding rather
// than a click, which a stray click cannot set off. Button fills while
// pressed and confirms when full; letting go early drains it. Slider
// confirms when its knob is dragged to the end; let go before, and the
// knob springs back. Either shows a check for a moment after.
//
//	if hold.Button(c, &app.wipe, "Hold to erase", 1500*time.Millisecond).Confirmed {
//		app.erase()
//	}
//	if hold.Slider(c, &app.unlock, "Slide to unlock").Confirmed {
//		app.unlock()
//	}
package hold

import (
	"time"

	"github.com/egoist/mygo/ui"
)

// shown is how long the check shows once confirmed.
const shown = 1200 * time.Millisecond

// State is what a control keeps: how far it is filled or slid, and when
// it was last confirmed. Keep it in the app's state, one for each
// control.
type State struct {
	progress float64
	at       time.Time
	done     time.Time
	held     bool
	grab     bool
	// wait holds a control that confirmed until the pointer lets go, so
	// that holding on does not confirm again.
	wait bool
}

// Events is a control, and whether it confirmed this frame.
type Events struct {
	ui.Element
	Confirmed bool
}

// step returns the time since the last frame that s saw, at most a tenth
// of a second, as a frame after a pause should not jump.
func (s *State) step(now time.Time) float64 {
	d := now.Sub(s.at)
	s.at = now
	return min(0.1, max(0, d.Seconds()))
}

// Button creates a button showing label that confirms once held for d.
func Button(c *ui.Context, s *State, label string, d time.Duration) Events {
	t := c.Theme()
	now := c.Now()
	dt := s.step(now)
	btn := ui.ButtonBase(c).Height(t.Space(10)).Padding(0, t.Space(5)).Radius(t.Radius).Border(1, t.Danger.Alpha(0.5)).
		Background(t.Danger.Alpha(0.08)).Role(ui.RoleButton).Label(label).Cursor(ui.CursorPointer)
	ev := Events{Element: btn}
	done := now.Sub(s.done) < shown
	pressed := btn.Pressed()
	if !pressed {
		s.wait = false
	}
	switch {
	case done || s.wait:
	case pressed:
		s.progress = min(1, s.progress+dt/max(0.05, d.Seconds()))
		if s.progress >= 1 {
			ev.Confirmed, s.done, done, s.wait = true, now, true, true
		}
	case s.progress > 0:
		// Letting go drains the button four times as fast as it filled.
		s.progress = max(0, s.progress-4*dt/max(0.05, d.Seconds()))
	}
	if done || s.wait {
		s.progress = 0
	}
	if pressed || s.progress > 0 || done {
		c.AnimationFrame()
	}
	fill, check := float32(s.progress), done
	btn.Draw(func(p *ui.Painter, r ui.Rect) {
		if fill > 0 {
			p.Clip(r, t.Radius, func() {
				p.Fill(ui.Rect{X: r.X, Y: r.Y, W: r.W * fill, H: r.H}, t.Danger.Alpha(0.85), 0)
			})
		}
	}).Children(func() {
		text := label
		if check {
			text = "✓"
		}
		col := t.Danger
		if fill > 0.5 {
			col = ui.RGB(255, 255, 255)
		}
		ui.Text(c, text).TextColor(col).Bold().Center()
	})
	return ev
}

// Slider creates a track showing label whose knob confirms once dragged
// to the end. Give it a width.
func Slider(c *ui.Context, s *State, label string) Events {
	t := c.Theme()
	now := c.Now()
	dt := s.step(now)
	h := t.Space(12)
	pad := float32(4)
	knob := h - 2*pad
	track := ui.Box(c).Height(h).Radius(h/2).Background(t.Danger.Alpha(0.1)).Border(1, t.Danger.Alpha(0.25)).
		Role(ui.RoleSlider).Label(label)
	b := track.LayoutBounds()
	run := max(1, b.W-2*pad-knob)
	ev := Events{Element: track}
	done := now.Sub(s.done) < shown
	px, _, _ := track.PointerPosition()
	pressed := track.Pressed()
	// The knob follows the drag outside the track too, where the track is
	// no longer pressed, until the pointer lets go.
	_, _, ended := track.DragEnded()
	released := ended || track.Clicked()
	switch {
	case done:
		s.progress = 1
	case s.grab:
		if dx, _, ok := track.Dragged(); ok {
			s.progress = max(0, min(1, s.progress+float64(dx/run)))
		}
		if s.progress >= 0.98 {
			ev.Confirmed, s.done, done = true, now, true
			s.progress, s.grab, s.wait = 1, false, true
		}
	case pressed && !s.held && !s.wait:
		x := float32(s.progress)*run + pad
		s.grab = px >= x && px <= x+knob
	}
	if !s.grab && !done && s.progress > 0 {
		if c.Preferences().ReduceMotion {
			s.progress = 0
		} else {
			s.progress = max(0, s.progress-dt*4)
		}
	}
	if released {
		s.grab, s.wait = false, false
	}
	s.held = pressed
	if !s.grab && s.progress > 0 || done {
		c.AnimationFrame()
	}
	if s.grab {
		track.Cursor(ui.CursorGrabbing)
	} else {
		track.Cursor(ui.CursorGrab)
	}
	at, check := float32(s.progress), done
	track.Draw(func(p *ui.Painter, r ui.Rect) {
		x := r.X + pad + at*run
		p.Fill(ui.Rect{X: r.X + pad, Y: r.Y + pad, W: x - r.X - pad + knob, H: knob}, t.Danger.Alpha(0.25), knob/2)
		// A span of no color takes the theme's, so the label goes once it
		// fades out.
		if fade := 1 - at*1.6; fade > 0.02 {
			span := ui.Span{Text: label, Weight: 600, Color: t.Danger.Alpha(fade)}
			w, th := p.MeasureText(0, span)
			p.RichText(r.X+knob+(r.W-knob)/2-w/2, r.Y+r.H/2-th/2, 0, span)
		}
		k := ui.Rect{X: x, Y: r.Y + pad, W: knob, H: knob}
		p.Shadow(k, knob/2, 0, 1, 4, 0, ui.RGB(0, 0, 0).Alpha(0.25))
		p.Fill(k, t.Danger, knob/2)
		mark := ui.Span{Text: "→", Size: knob * 0.45, Weight: 700, Color: ui.RGB(255, 255, 255)}
		if check {
			mark.Text = "✓"
		}
		mw, mh := p.MeasureText(0, mark)
		p.RichText(k.X+knob/2-mw/2, k.Y+knob/2-mh/2, 0, mark)
	})
	return ev
}
