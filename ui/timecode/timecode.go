// Package timecode is a timecode field, as video editors have over their
// viewer: a position in frames written as hours, minutes, seconds and
// frames, 01:00:12:08, at the footage's rate, a semicolon before the
// frames marking drop-frame. The zeros before the first figure that counts
// are dimmed.
//
//	timecode.Field(c, &app.tc, &app.frame, timecode.Rate{FPS: 30, Drop: true})
//
// Dragging the field sideways scrubs, a frame each 4 DIPs. With the focus,
// Up and Down step a frame, and a second with Shift. Typing digits enters
// a timecode from the right, as editors take it: 1, 0, 0 is 00:00:01:00;
// a + or - first moves by that much instead. Enter goes there, and Escape
// or the focus leaving drops what was typed.
package timecode

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
)

// Rate is the rate of the footage: the frames a second is numbered in,
// 24, 25, 30 or 60, and whether it is drop-frame, as 29.97 and 59.94 are,
// skipping the first numbers of each minute but every tenth.
type Rate struct {
	FPS  int
	Drop bool
}

// dropped returns the numbers skipped each minute.
func (r Rate) dropped() int {
	if !r.Drop {
		return 0
	}
	return r.FPS / 15
}

// Format writes frame, counted from 0, as a timecode at r.
func Format(frame int, r Rate) string {
	h, m, s, f := split(max(frame, 0), r)
	sep := ":"
	if r.Drop {
		sep = ";"
	}
	return fmt.Sprintf("%02d:%02d:%02d%s%02d", h, m, s, sep, f)
}

func split(frame int, r Rate) (h, m, s, f int) {
	if d := r.dropped(); d > 0 {
		perTen := r.FPS*600 - d*9
		perMinute := r.FPS*60 - d
		tens, rest := frame/perTen, frame%perTen
		frame += d * 9 * tens
		if rest > d {
			frame += d * ((rest - d) / perMinute)
		}
	}
	return frame / (r.FPS * 3600), frame / (r.FPS * 60) % 60, frame / r.FPS % 60, frame % r.FPS
}

// frames returns the frame of the timecode h:m:s:f at r, the reverse of
// split.
func frames(h, m, s, f int, r Rate) int {
	minutes := h*60 + m
	return (minutes*60+s)*r.FPS + f - r.dropped()*(minutes-minutes/10)
}

// State is what a field keeps: what is being typed. Keep it in the app's
// state, one for each field.
type State struct {
	typed  string
	sign   byte
	scrub  float32
	typing bool
}

// Field creates the field of *frame, at r.
func Field(c *ui.Context, s *State, frame *int, r Rate) ui.Element {
	t := c.Theme()
	field := ui.Box(c).Focusable().Label("Timecode").Cursor(ui.CursorResizeEW).
		Padding(t.Space(1.5), t.Space(3)).Radius(t.Radius).Background(t.Text.Alpha(0.05)).Border(1, t.Border)
	if !field.Focused() {
		s.typing, s.typed, s.sign = false, "", 0
	}
	if dx, _, ok := field.Dragged(); ok {
		s.scrub += dx
		if n := int(s.scrub / 4); n != 0 {
			*frame = max(*frame+n, 0)
			s.scrub -= float32(n) * 4
		}
	}
	if !field.Pressed() {
		s.scrub = 0
	}
	if field.Clicked() {
		field.Focus()
	}
	field.HandleInput(func(e ui.InputEvent) bool {
		switch e.Kind {
		case ui.InputText:
			took := false
			for _, ch := range e.Text {
				switch {
				case ch >= '0' && ch <= '9' && len(s.typed) < 8:
					s.typed += string(ch)
				case (ch == '+' || ch == '-') && s.typed == "":
					s.sign = byte(ch)
				default:
					continue
				}
				s.typing, took = true, true
			}
			return took
		case ui.InputKeyDown:
		default:
			return false
		}
		step := 1
		if e.Mods&ui.Shift != 0 {
			step = r.FPS
		}
		switch {
		case e.Key == ui.KeyEnter && s.typing:
			at := typedFrame(s.typed, r)
			switch s.sign {
			case '+':
				at = *frame + at
			case '-':
				at = *frame - at
			}
			*frame = max(at, 0)
			s.typing, s.typed, s.sign = false, "", 0
		case e.Key == ui.KeyEscape && s.typing:
			s.typing, s.typed, s.sign = false, "", 0
		case e.Key == ui.KeyBackspace && s.typing:
			if s.typed == "" {
				s.typing, s.sign = false, 0
			} else {
				s.typed = s.typed[:len(s.typed)-1]
			}
		case e.Key == ui.KeyUp && !s.typing:
			*frame += step
		case e.Key == ui.KeyDown && !s.typing:
			*frame = max(*frame-step, 0)
		default:
			return false
		}
		return true
	})

	size := t.FontSize + 8
	var spans []ui.Span
	add := func(text string, color ui.Color) {
		spans = append(spans, ui.Span{Text: text, Font: "monospace", Size: size, Weight: 500, Color: color, Features: "tnum"})
	}
	if s.typing {
		field.Background(t.Accent.Alpha(0.08)).Border(1, t.Accent)
		if s.sign != 0 {
			add(string(s.sign), t.Accent)
		}
		digits := strings.Repeat("0", 8-len(s.typed)) + s.typed
		for i := range 8 {
			if i > 0 && i%2 == 0 {
				add(separator(i, r), t.TextMuted)
			}
			color := t.TextMuted
			if i >= 8-len(s.typed) {
				color = t.Text
			}
			add(digits[i:i+1], color)
		}
	} else {
		text := Format(*frame, r)
		lead := 0
		for lead < len(text)-2 && (text[lead] == '0' || text[lead] == ':' || text[lead] == ';') {
			lead++
		}
		add(text[:lead], t.TextMuted.Alpha(0.6))
		add(text[lead:], t.Text)
	}
	field.Children(func() { ui.RichText(c, spans...).SingleLine() })
	return field
}

func separator(i int, r Rate) string {
	if i == 6 && r.Drop {
		return ";"
	}
	return ":"
}

// typedFrame returns the frame of digits typed, read from the right as
// frames, seconds, minutes and hours.
func typedFrame(typed string, r Rate) int {
	d := strings.Repeat("0", 8-len(typed)) + typed
	n := func(i int) int { return int(d[i]-'0')*10 + int(d[i+1]-'0') }
	h, m, sec, f := n(0), n(2), n(4), n(6)
	// The numbers drop-frame skips go to the first frame of that minute.
	if m%10 != 0 && sec == 0 {
		f = max(f, r.dropped())
	}
	return frames(h, m, sec, f, r)
}
