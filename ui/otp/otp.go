// Package otp is a field for a one-time code, as sign-in pages take one
// sent by text or email: a box a digit, the box to type in lit, a digit
// typed going to the next and Backspace taking back the last; a code
// pasted fills the boxes, what is not a digit left out. Done reports the
// code in the frame it is complete.
//
//	if code, ok := otp.Field(c, &app.code, 6).Done(); ok {
//		app.verify(code)
//	}
package otp

import (
	"strings"

	"github.com/egoist/mygo/ui"
)

// Events is what the user did to a field in a frame.
type Events struct {
	code string
	done bool
}

// Done returns the code, and true in the frame it is complete.
func (e Events) Done() (string, bool) { return e.code, e.done }

// Field creates the field of a code of n digits, *code what is typed.
func Field(c *ui.Context, code *string, n int) Events {
	t := c.Theme()
	digits := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, *code)
	if len(digits) > n {
		digits = digits[:n]
	}
	*code = digits
	w, h := t.Space(11), t.Space(13)
	gap := t.Space(2)
	var ev Events
	ui.Box(c).Children(func() {
		// The input takes the keys and the clicks under the boxes, which
		// show what it holds.
		in := ui.TextInputBase(c, code).Absolute().Left(0).Top(0).Fill().Opacity(0).Label("Code").
			Cursor(ui.CursorText)
		last := ui.Local(in, "otp.last", func() string { return "" })
		if digits != *last && len(digits) == n {
			ev = Events{code: digits, done: true}
		}
		*last = digits
		focused := in.Focused()
		ui.Row(c).Gap(gap).AlignItems(ui.Center).Children(func() {
			for k := range n {
				if n >= 6 && n%2 == 0 && k == n/2 {
					ui.Box(c.Key(-1)).Size(t.Space(3), 2).Background(t.TextMuted)
				}
				text := ""
				if k < len(digits) {
					text = digits[k : k+1]
				}
				box := ui.Column(c.Key(k)).Size(w, h).Radius(t.Space(2)).Background(t.Background).Border(1, t.Border).
					Justify(ui.Center).AlignItems(ui.Center)
				if focused && k == min(len(digits), n-1) {
					box.Border(2, t.Accent)
				}
				box.Children(func() {
					ui.Text(c, text).Font("monospace").FontSize(t.FontSize + 8).Bold()
				})
			}
		})
	})
	return ev
}
