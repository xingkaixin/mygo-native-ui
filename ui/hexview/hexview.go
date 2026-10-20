// Package hexview shows bytes as a hex dump, as Hex Fiend and hexyl show
// them: sixteen to a row, each row's offset, the bytes in hex, and the
// same bytes as ASCII text, selected in both columns at once.
//
//	hexview.View(c, &app.hex, app.data).Grow(1)
//	start, end := app.hex.Selection()
//	ui.Textf(c, "Offset %#x, %d bytes selected", start, end-start)
//
// The view builds only the rows in view, as ui.List does, so it scrolls
// through bytes of any length as fast as through a few.
package hexview

import (
	"math"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
)

// perRow is how many bytes a row shows.
const perRow = 16

// State is what a view shows chosen: the byte where the selection began
// and the one where the cursor is, which end it, in the hex column or the
// ASCII one. Keep it in the app's state, one for each view. The zero
// State selects the first byte.
type State struct {
	anchor, cursor int
	ascii          bool
	list           ui.ListState
	// scroll is a row to scroll into view in the next frame.
	scroll int
}

// Selection returns the bytes selected, from start to end, end excluded:
// one at least, the cursor's, in a view of bytes.
func (s *State) Selection() (start, end int) {
	return min(s.anchor, s.cursor), max(s.anchor, s.cursor) + 1
}

// Select selects the bytes from start to end, end excluded, as the result
// of a search, and scrolls them into view.
func (s *State) Select(start, end int) {
	s.anchor, s.cursor = start, max(start, end-1)
	s.scroll = s.cursor/perRow + 1
}

// moveTo moves the cursor to i, within n bytes, and the anchor with it
// unless extend.
func (s *State) moveTo(i, n int, extend bool) {
	s.cursor = max(0, min(n-1, i))
	if !extend {
		s.anchor = s.cursor
	}
	s.scroll = s.cursor/perRow + 1
}

// layout is where things are along a row, in DIPs from its left: the
// offset, the bytes in hex, and the bytes as text, of characters cw wide.
type layout struct {
	cw, hexX, textX, width float32
	digits                 int
}

func newLayout(cw float32, n int) layout {
	digits := max(8, len(strconv.FormatInt(int64(max(n-1, 0)), 16)))
	l := layout{cw: cw, digits: digits}
	l.hexX = float32(digits+2) * cw
	l.textX = l.hexX + float32(perRow*3+1+1)*cw
	l.width = l.textX + perRow*cw
	return l
}

// hexAt returns where byte i of a row starts in the hex column: pairs with
// a space between, and another after the eighth.
func (l layout) hexAt(i int) float32 {
	x := l.hexX + float32(3*i)*l.cw
	if i >= perRow/2 {
		x += l.cw
	}
	return x
}

// byteAt returns the byte of a row under x, and whether it is in the
// ASCII column; past the ends of a column, its first or last byte.
func (l layout) byteAt(x float32) (int, bool) {
	if x >= l.textX-l.cw {
		return max(0, min(perRow-1, int((x-l.textX)/l.cw))), true
	}
	for i := perRow - 1; i > 0; i-- {
		if x >= l.hexAt(i)-l.cw/2 {
			return i, false
		}
	}
	return 0, false
}

// View creates a hex dump of data, of which s says what is chosen. Give
// it a size, or grow it in its parent.
//
// A click chooses a byte, in either column, Shift-click and dragging
// select bytes up to it, and the arrows, Page Up and Down, Home and End
// (of a row) move the cursor while the view has the keyboard focus, with
// Shift held selecting; Cmd+Up and Cmd+Down go to the first and last
// byte, Cmd+A selects them all, and Cmd+C copies the bytes selected, in
// hex, or as text from the ASCII column. Bytes of zero show muted, and
// bytes that are not printable ASCII show as dots in the ASCII column.
func View(c *ui.Context, s *State, data []byte) ui.Element {
	t := c.Theme()
	n := len(data)
	size := t.FontSize
	cw, textH := c.MeasureText(0, ui.Span{Text: "0", Font: "monospace", Size: size})
	rowH := float32(math.Ceil(float64(textH * 1.35)))
	l := newLayout(cw, n)
	rows := (n + perRow - 1) / perRow
	if n > 0 {
		s.anchor, s.cursor = max(0, min(n-1, s.anchor)), max(0, min(n-1, s.cursor))
	}

	view := ui.Column(c)
	view.Children(func() {
		ui.Box(c).Height(rowH+t.Space(2)).Padding(0, t.Space(3)).Draw(func(p *ui.Painter, r ui.Rect) {
			y := r.Y + (r.H-textH)/2
			p.RichText(r.X+t.Space(3), y, 0, ui.Span{Text: "Offset", Font: "monospace", Size: size, Color: t.TextMuted})
			for i := range perRow {
				label := strings.ToUpper(strconv.FormatInt(int64(i), 16))
				p.RichText(r.X+t.Space(3)+l.hexAt(i), y, 0, ui.Span{Text: "0" + label, Font: "monospace", Size: size, Color: t.TextMuted})
			}
			p.RichText(r.X+t.Space(3)+l.textX, y, 0, ui.Span{Text: "ASCII", Font: "monospace", Size: size, Color: t.TextMuted})
		})
		ui.Divider(c)
		list := ui.List(c, &s.list, rows, func(row int) {
			s.row(c, data, row, rowH, textH, l)
		}).Grow(1).Padding(t.Space(1), 0).Focusable()
		focused := list.FocusWithin()
		if focused {
			keys(list, s, n)
		}
		if s.scroll > 0 {
			s.list.ScrollIntoView(s.scroll - 1)
			s.scroll = 0
		}
		if n > 0 && list.Shortcut(ui.Cmd, ui.KeyC) {
			c.WriteClipboard(s.copy(data))
		}
		list.Children(func() {
			if n == 0 {
				ui.Text(c, "No bytes").TextColor(t.TextMuted).Padding(t.Space(3))
			}
		})
	})
	return view
}

// keys moves the cursor with the keys pressed in the list of n bytes.
func keys(list ui.Element, s *State, n int) {
	if n == 0 {
		return
	}
	first, last := s.list.Visible()
	page := max(1, last-first) * perRow
	at := s.cursor
	for _, k := range []struct {
		key ui.Key
		to  int
	}{
		{ui.KeyLeft, at - 1}, {ui.KeyRight, at + 1},
		{ui.KeyUp, at - perRow}, {ui.KeyDown, at + perRow},
		{ui.KeyPageUp, at - page}, {ui.KeyPageDown, at + page},
		{ui.KeyHome, at - at%perRow}, {ui.KeyEnd, at - at%perRow + perRow - 1},
	} {
		if list.Shortcut(0, k.key) {
			s.moveTo(k.to, n, false)
		}
		if list.Shortcut(ui.Shift, k.key) {
			s.moveTo(k.to, n, true)
		}
	}
	for _, k := range []struct {
		mods ui.Modifiers
		key  ui.Key
		to   int
	}{
		{ui.Cmd, ui.KeyUp, 0}, {ui.Cmd, ui.KeyDown, n - 1},
		{ui.Cmd | ui.Shift, ui.KeyUp, 0}, {ui.Cmd | ui.Shift, ui.KeyDown, n - 1},
	} {
		if list.Shortcut(k.mods, k.key) {
			s.moveTo(k.to, n, k.mods&ui.Shift != 0)
		}
	}
	if list.Shortcut(ui.Cmd, ui.KeyA) {
		s.anchor, s.cursor = 0, n-1
	}
}

// row builds row of data: the offset, then the bytes in hex and as text,
// over the selection, which a press on the row's bytes changes.
func (s *State) row(c *ui.Context, data []byte, row int, rowH, textH float32, l layout) {
	t := c.Theme()
	n := len(data)
	from := row * perRow
	to := min(n, from+perRow)
	pad := t.Space(3)
	box := ui.Box(c).Height(rowH).Width(l.width + 2*pad)

	pressed := ui.Local(box, "hexview.pressed", func() bool { return false })
	// Dragged goes on while the press does, past the row, where Pressed
	// stops.
	if _, _, ok := box.Dragged(); ok {
		px, py, _ := box.PointerPosition()
		// Captured, the pointer reads in the row's axes past its box too,
		// which reach the rows above and below.
		i, ascii := l.byteAt(px - pad)
		at := from + int(math.Floor(float64(py/rowH)))*perRow + i
		if *pressed {
			s.moveTo(at, n, true)
		} else {
			s.ascii = ascii
			s.moveTo(at, n, box.ClickModifiers()&ui.Shift != 0)
		}
		*pressed, s.scroll = true, 0
	} else {
		*pressed = false
	}

	start, end := s.Selection()
	focused := s.list.FocusWithin(c)
	box.Draw(func(p *ui.Painter, r ui.Rect) {
		x0, y := r.X+pad, r.Y+(r.H-textH)/2
		mono := func(text string, color ui.Color) ui.Span {
			return ui.Span{Text: text, Font: "monospace", Size: t.FontSize, Color: color}
		}
		for i := range to - from {
			at := from + i
			if at < start || at >= end {
				continue
			}
			hexBg, textBg := t.Selection, t.Selection
			if focused {
				if s.ascii {
					textBg = t.Accent
				} else {
					hexBg = t.Accent
				}
			}
			w := 2 * l.cw
			if at+1 < end && i+1 < perRow {
				w = l.hexAt(i+1) - l.hexAt(i)
			}
			p.Fill(ui.Rect{X: x0 + l.hexAt(i), Y: r.Y, W: w, H: r.H}, hexBg, 0)
			p.Fill(ui.Rect{X: x0 + l.textX + float32(i)*l.cw, Y: r.Y, W: l.cw, H: r.H}, textBg, 0)
		}
		offset := strconv.FormatInt(int64(from), 16)
		p.RichText(x0, y, 0, mono(strings.Repeat("0", l.digits-len(offset))+offset, t.TextMuted))
		text := make([]ui.Span, 0, perRow)
		for i, b := range data[from:to] {
			at := from + i
			on := at >= start && at < end && focused
			color, textColor := t.Text, t.Text
			if b == 0 {
				color = t.TextMuted
			}
			ch := string(rune(b))
			if b < 0x20 || b > 0x7e {
				ch, textColor = "·", t.TextMuted
			}
			if on && !s.ascii {
				color = t.AccentText
			}
			if on && s.ascii {
				textColor = t.AccentText
			}
			p.RichText(x0+l.hexAt(i), y, 0, mono(hexByte(b), color))
			text = append(text, mono(ch, textColor))
		}
		p.RichText(x0+l.textX, y, 0, text...)
	})
}

// copy returns the bytes selected in data as the column of the cursor
// shows them: in hex, separated by spaces, or as text.
func (s *State) copy(data []byte) string {
	start, end := s.Selection()
	end = min(end, len(data))
	var b strings.Builder
	for i, c := range data[start:end] {
		switch {
		case !s.ascii:
			if i > 0 {
				b.WriteByte(' ')
			}
			b.WriteString(hexByte(c))
		case c < 0x20 || c > 0x7e:
			b.WriteByte('.')
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}

func hexByte(b byte) string {
	const digits = "0123456789abcdef"
	return string([]byte{digits[b>>4], digits[b&0xf]})
}
