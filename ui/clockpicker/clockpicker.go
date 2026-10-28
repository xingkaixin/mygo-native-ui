// Package clockpicker picks a time of day on a clock face, as phones do:
// the time large above the face, its hour and its minute each a button,
// and AM and PM beside them. The face shows the hours, or the minutes in
// fives, with a hand to the one chosen. A click or a drag on the face sets
// the hour, then turns to the minutes; the minutes take any minute between
// the fives. With Clock24 set, the hours from 13 to 23 and 00 ring inside
// the others, and AM and PM go away. ← and → move between the hour and the
// minute, and ↑ and ↓ change the one chosen.
//
//	clockpicker.View(c, &app.picker, &app.alarm)
package clockpicker

import (
	"fmt"
	"math"
	"time"

	"github.com/egoist/mygo/ui"
)

// State is what a picker keeps: whether it shows the minutes, and
// whether the app wants a 24-hour clock. Keep it in the app's state, one
// for each picker.
type State struct {
	// Clock24 shows the hours from 0 to 23, without AM and PM.
	Clock24 bool
	minutes bool
	down    bool
}

// View creates the picker of the time of day of *at, keeping its date.
func View(c *ui.Context, s *State, at *time.Time) ui.Element {
	t := c.Theme()
	h, m := at.Hour(), at.Minute()
	set := func(hour, minute int) {
		*at = time.Date(at.Year(), at.Month(), at.Day(), (hour+24)%24, (minute+60)%60, 0, 0, at.Location())
	}
	root := ui.Column(c).Gap(t.Space(5)).AlignItems(ui.Center).Focusable().FocusRing(false).
		Label("Time").Value(at.Format("15:04"))
	switch {
	case root.Shortcut(0, ui.KeyLeft):
		s.minutes = false
	case root.Shortcut(0, ui.KeyRight):
		s.minutes = true
	case root.Shortcut(0, ui.KeyUp) && s.minutes:
		set(h, m+1)
	case root.Shortcut(0, ui.KeyDown) && s.minutes:
		set(h, m-1)
	case root.Shortcut(0, ui.KeyUp):
		set(h+1, m)
	case root.Shortcut(0, ui.KeyDown):
		set(h-1, m)
	}
	root.Children(func() {
		ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
			shown := h
			if !s.Clock24 {
				shown = (h+11)%12 + 1
			}
			part := func(text, label string, on bool) bool {
				b := ui.ButtonBase(c.Key(label)).Padding(t.Space(1), t.Space(3)).Radius(t.Space(2.5)).Label(label).Cursor(ui.CursorPointer)
				col := t.Text
				switch {
				case on:
					b.Background(t.Accent.Alpha(0.15))
					col = t.Accent
				case b.Hovered():
					b.Background(t.SurfaceHover)
				}
				b.Children(func() {
					ui.Text(c, text).FontSize(t.FontSize + 30).FontFeatures("tnum").TextColor(col)
				})
				return b.Clicked()
			}
			if part(fmt.Sprintf("%02d", shown), "Hour", !s.minutes) {
				s.minutes = false
			}
			ui.Text(c, ":").FontSize(t.FontSize + 30)
			if part(fmt.Sprintf("%02d", m), "Minute", s.minutes) {
				s.minutes = true
			}
			if s.Clock24 {
				return
			}
			ui.Column(c).Margin(0, 0, 0, t.Space(2)).Radius(t.Space(2)).Border(1, t.Border).Clip().Children(func() {
				for i, name := range []string{"AM", "PM"} {
					on := (h >= 12) == (i == 1)
					b := ui.ButtonBase(c.Key(name)).Padding(t.Space(1.5), t.Space(3)).Label(name).Cursor(ui.CursorPointer)
					col := t.TextMuted
					if on {
						b.Background(t.Accent.Alpha(0.15))
						col = t.Accent
					}
					if b.Clicked() && !on {
						set(h+12, m)
					}
					b.Children(func() {
						ui.Text(c, name).Bold().FontSize(t.FontSize - 1).TextColor(col)
					})
				}
			})
		})
		face(c, s, h, m, set)
	})
	return root
}

// face creates the clock face, setting the time with set.
func face(c *ui.Context, s *State, h, m int, set func(hour, minute int)) {
	t := c.Theme()
	size := t.Space(64)
	e := ui.Box(c).Size(size, size).Radius(size / 2).Background(t.Surface).Cursor(ui.CursorPointer).Role(ui.RoleSlider)
	if s.minutes {
		e.Label("Minute").Range(0, 59, float64(m))
	} else {
		e.Label("Hour").Range(0, 23, float64(h))
	}
	outer, inner := size/2-t.Space(6), size/2-t.Space(14)
	px, py, _ := e.PointerPosition()
	pressed := e.Pressed()
	if pressed {
		dx, dy := px-size/2, py-size/2
		turn := math.Mod(math.Atan2(float64(dx), float64(-dy))/(2*math.Pi)+1, 1)
		if s.minutes {
			set(h, int(math.Round(turn*60))%60)
		} else {
			hour := int(math.Round(turn*12)) % 12
			switch {
			case s.Clock24 && math.Hypot(float64(dx), float64(dy)) < float64(outer+inner)/2:
				if hour != 0 {
					hour += 12
				}
			case s.Clock24:
				if hour == 0 {
					hour = 12
				}
			case h >= 12:
				hour += 12
			}
			set(hour, m)
		}
		e.SetChanged()
	}
	if !pressed && s.down && !s.minutes {
		s.minutes = true
	}
	s.down = pressed
	minutes, clock24 := s.minutes, s.Clock24
	e.Draw(func(p *ui.Painter, r ui.Rect) {
		cx, cy := r.X+r.W/2, r.Y+r.H/2
		at := func(turn float64, rad float32) (float32, float32) {
			a := turn*2*math.Pi - math.Pi/2
			return cx + rad*float32(math.Cos(a)), cy + rad*float32(math.Sin(a))
		}
		type mark struct {
			text string
			turn float64
			rad  float32
			on   bool
		}
		var marks []mark
		var hand mark
		if minutes {
			for k := range 12 {
				marks = append(marks, mark{fmt.Sprintf("%02d", k*5), float64(k) / 12, outer, m == k*5})
			}
			hand = mark{"", float64(m) / 60, outer, m%5 == 0}
		} else {
			for k := range 12 {
				hour := k
				if hour == 0 {
					hour = 12
				}
				marks = append(marks, mark{fmt.Sprint(hour), float64(k) / 12, outer, false})
				if clock24 {
					in := k + 12
					if k == 0 {
						in = 0
					}
					marks = append(marks, mark{fmt.Sprintf("%02d", in), float64(k) / 12, inner, false})
				}
			}
			rad := outer
			if clock24 && (h == 0 || h > 12) {
				rad = inner
			}
			hand = mark{"", float64(h%12) / 12, rad, true}
		}
		knob := t.Space(5)
		hx, hy := at(hand.turn, hand.rad)
		var arm ui.Path
		arm.MoveTo(cx, cy)
		arm.LineTo(hx, hy)
		p.StrokePath(&arm, 2, t.Accent)
		var pin ui.Path
		pin.Circle(cx, cy, 3.5)
		p.FillPath(&pin, t.Accent)
		var ball ui.Path
		ball.Circle(hx, hy, knob)
		p.FillPath(&ball, t.Accent)
		if !hand.on {
			var dot ui.Path
			dot.Circle(hx, hy, 2.5)
			p.FillPath(&dot, t.AccentText)
		}
		for _, mk := range marks {
			x, y := at(mk.turn, mk.rad)
			size := t.FontSize
			col := t.Text
			if mk.rad == inner {
				size, col = t.FontSize-2, t.TextMuted
			}
			if math.Hypot(float64(x-hx), float64(y-hy)) < float64(knob)/2 {
				col = t.AccentText
			}
			span := ui.Span{Text: mk.text, Size: size, Color: col}
			w, th := p.MeasureText(0, span)
			p.RichText(x-w/2, y-th/2, 0, span)
		}
	})
}
