// Package logview shows the output of programs as it comes, in the colors
// and styles their ANSI escape sequences give it, as a terminal shows a
// build's log, without being one: there is no shell, cursor or input.
//
//	cmd.Stdout = pipe // read in a goroutine:
//	for {
//		n, err := pipe.Read(buf)
//		chunk := bytes.Clone(buf[:n])
//		win.Update(func() { app.log.Write(chunk) })
//		...
//	}
//
//	logview.View(c, &app.log, app.filter).Grow(1)
//
// A Log keeps the lines written to it, and a View shows them, following
// the end as lines come while the user has not scrolled away from it.
package logview

import (
	"bytes"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
)

// Log is the text of a log, in lines, with its colors and styles. Write
// to it on the main thread, as from a goroutine through Window.Update, and
// show it in one View at a time. The zero Log is empty and keeps every
// line.
type Log struct {
	// Max, when above 0, is how many lines the log keeps, the last ones.
	Max int

	lines []line
	// open reports whether the last line goes on: written since its
	// newline, or not ended by one yet.
	open bool
	// dropped counts the lines taken from the front, which numbers the
	// lines from the first written.
	dropped int
	// cur is the style the next text takes, pending the bytes of an
	// escape sequence or a rune cut by the end of a Write, and cr a
	// carriage return whose line goes back to its start unless a newline
	// follows.
	cur     style
	pending []byte
	cr      bool

	list   ui.ListState
	filter filtered
}

// line is a line of text in runs of a style: each run ends at byte end.
type line struct {
	text string
	runs []run
}

type run struct {
	end int
	st  style
}

// style is how a run of text shows. Colors are a palette's index from 0
// to 255, an RGB color with rgbFlag set, or defaultColor.
type style struct {
	fg, bg  int32
	bold    bool
	dim     bool
	italic  bool
	under   bool
	strike  bool
	inverse bool
}

const (
	defaultColor = -1
	rgbFlag      = 1 << 24
)

var plain = style{fg: defaultColor, bg: defaultColor}

// Write adds p to the log: text, newlines, carriage returns, which start
// their line over as progress bars do, tabs to every eighth column, and
// the SGR escape sequences setting colors (the 16 of the palette, the 256
// of xterm, and RGB) and bold, faint, italic, underlined, crossed out and
// inverse text. It leaves out other escape sequences, as those moving the
// cursor and setting the title. A sequence or a character that p cuts
// short waits for the rest in the next Write. It never fails.
func (l *Log) Write(p []byte) (int, error) {
	if l.cur == (style{}) {
		l.cur = plain
	}
	data := p
	if len(l.pending) > 0 {
		data = append(l.pending, p...)
		l.pending = nil
	}
	var text []byte
	flush := func() {
		if len(text) > 0 {
			l.add(string(text))
			text = text[:0]
		}
	}
	for i := 0; i < len(data); {
		b := data[i]
		if l.cr && b != '\n' {
			flush()
			l.restart()
		}
		l.cr = false
		switch {
		case b == '\n':
			flush()
			l.newline()
			i++
		case b == '\r':
			flush()
			l.cr = true
			i++
		case b == '\t':
			col := utf8.RuneCount(text)
			if l.open {
				col += utf8.RuneCountInString(l.lines[len(l.lines)-1].text)
			}
			text = append(text, "        "[:8-col%8]...)
			i++
		case b == 0x1b:
			n, ok := l.escape(data[i:], flush)
			if !ok {
				l.pending = bytes.Clone(data[i:])
				i = len(data)
				break
			}
			i += n
		case b < 0x20 || b == 0x7f:
			i++
		case b < utf8.RuneSelf:
			text = append(text, b)
			i++
		default:
			if !utf8.FullRune(data[i:]) {
				l.pending = bytes.Clone(data[i:])
				i = len(data)
				break
			}
			_, n := utf8.DecodeRune(data[i:])
			text = append(text, data[i:i+n]...)
			i += n
		}
	}
	flush()
	return len(p), nil
}

// escape reads the escape sequence at the start of b, of n bytes, setting
// the style for an SGR sequence once the text before it is flushed. It
// reports false for a sequence cut short.
func (l *Log) escape(b []byte, flush func()) (n int, ok bool) {
	if len(b) < 2 {
		return 0, false
	}
	switch b[1] {
	case '[':
		for i := 2; i < len(b); i++ {
			if c := b[i]; c >= 0x40 && c <= 0x7e {
				if c == 'm' {
					flush()
					l.cur = sgr(l.cur, string(b[2:i]))
				}
				return i + 1, true
			}
		}
		return 0, false
	case ']', 'P', '_', '^':
		// Strings, as OSC's titles and links, end with BEL or ST.
		for i := 2; i < len(b); i++ {
			if b[i] == 0x07 {
				return i + 1, true
			}
			if b[i] == 0x1b && i+1 < len(b) && b[i+1] == '\\' {
				return i + 2, true
			}
		}
		return 0, false
	}
	return 2, true
}

// sgr returns st as the parameters of an SGR sequence set it.
func sgr(st style, params string) style {
	if params == "" {
		return plain
	}
	ps := strings.FieldsFunc(params, func(r rune) bool { return r == ';' || r == ':' })
	num := func(i int) int {
		if i >= len(ps) {
			return 0
		}
		n, _ := strconv.Atoi(ps[i])
		return n
	}
	for i := 0; i < len(ps); i++ {
		switch n := num(i); {
		case n == 0:
			st = plain
		case n == 1:
			st.bold = true
		case n == 2:
			st.dim = true
		case n == 3:
			st.italic = true
		case n == 4:
			st.under = true
		case n == 7:
			st.inverse = true
		case n == 9:
			st.strike = true
		case n == 21 || n == 22:
			st.bold, st.dim = false, false
		case n == 23:
			st.italic = false
		case n == 24:
			st.under = false
		case n == 27:
			st.inverse = false
		case n == 29:
			st.strike = false
		case n >= 30 && n <= 37:
			st.fg = int32(n - 30)
		case n >= 90 && n <= 97:
			st.fg = int32(n - 90 + 8)
		case n == 39:
			st.fg = defaultColor
		case n >= 40 && n <= 47:
			st.bg = int32(n - 40)
		case n >= 100 && n <= 107:
			st.bg = int32(n - 100 + 8)
		case n == 49:
			st.bg = defaultColor
		case n == 38 || n == 48:
			var c int32
			switch num(i + 1) {
			case 5:
				c, i = int32(num(i+2)&0xff), i+2
			case 2:
				// 38;2;r;g;b, or 38:2::r:g:b with a color space id, which
				// FieldsFunc leaves out with the empty field.
				c, i = rgbFlag|int32(num(i+2)&0xff)<<16|int32(num(i+3)&0xff)<<8|int32(num(i+4)&0xff), i+4
			default:
				continue
			}
			if n == 38 {
				st.fg = c
			} else {
				st.bg = c
			}
		}
	}
	return st
}

// add adds s in the current style to the last line, opening a line for
// it if the last one ended.
func (l *Log) add(s string) {
	if !l.open {
		l.lines = append(l.lines, line{})
		l.open = true
	}
	ln := &l.lines[len(l.lines)-1]
	ln.text += s
	if k := len(ln.runs); k > 0 && ln.runs[k-1].st == l.cur {
		ln.runs[k-1].end = len(ln.text)
	} else {
		ln.runs = append(ln.runs, run{end: len(ln.text), st: l.cur})
	}
}

// newline ends the last line, and drops the first lines past Max.
func (l *Log) newline() {
	if !l.open {
		l.lines = append(l.lines, line{})
	}
	l.open = false
	// Dropping a quarter of Max at a time keeps writes from copying the
	// lines kept every time; rows shows the last Max meanwhile.
	if l.Max > 0 && len(l.lines) > l.Max+l.Max/4 {
		n := len(l.lines) - l.Max
		l.lines = slices.Delete(l.lines, 0, n)
		l.dropped += n
	}
}

// restart takes the last line back to its start, for a carriage return.
func (l *Log) restart() {
	if l.open {
		l.lines[len(l.lines)-1] = line{}
	}
}

// Len returns how many lines the log shows.
func (l *Log) Len() int {
	n := len(l.lines)
	if l.Max > 0 {
		n = min(n, l.Max)
	}
	return n
}

// Clear empties the log.
func (l *Log) Clear() {
	l.dropped += len(l.lines)
	l.lines, l.open, l.cr, l.pending = nil, false, false, nil
	l.cur = plain
	l.filter = filtered{}
}

// Text returns the lines the log shows, without their styles, each ended
// by a newline but an open last one: the whole log for Copy, where a
// selection in a View copies only the lines it built.
func (l *Log) Text() string {
	var b strings.Builder
	for i := len(l.lines) - l.Len(); i < len(l.lines); i++ {
		b.WriteString(l.lines[i].text)
		if i < len(l.lines)-1 || !l.open {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

// filtered is the lines matching a filter, by their numbers from the
// first line written, as far as the lines ended up to scanned.
type filtered struct {
	query   string
	matches []int
	scanned int
}

// rows returns how many rows a View shows for filter, and the number of
// the line in each: the lines shown, or those of them holding filter,
// case aside. It scans only the lines ended since the last call, and the
// open line every time.
func (l *Log) rows(filter string) (int, func(row int) int) {
	first := l.dropped + len(l.lines) - l.Len()
	if filter == "" {
		return l.Len(), func(row int) int { return first + row }
	}
	f := &l.filter
	if f.query != filter {
		*f = filtered{query: filter, scanned: first}
	}
	q := strings.ToLower(filter)
	ended := l.dropped + len(l.lines)
	if l.open {
		ended--
	}
	for n := max(f.scanned, first); n < ended; n++ {
		if strings.Contains(strings.ToLower(l.lines[n-l.dropped].text), q) {
			f.matches = append(f.matches, n)
		}
	}
	f.scanned = ended
	k, _ := slices.BinarySearch(f.matches, first)
	f.matches = f.matches[k:]
	matches := f.matches
	if l.open && strings.Contains(strings.ToLower(l.lines[len(l.lines)-1].text), q) {
		matches = append(slices.Clip(matches), ended)
	}
	return len(matches), func(row int) int { return matches[row] }
}

// View creates a list of the lines of l that hold filter, case aside, or
// all of them for "", in the system's monospaced font, wrapping where
// they are wider than it, and the text that matches filter highlighted.
// Give it a size, or grow it in its parent.
//
// The view starts at the end of the log and follows it as lines come,
// until the user scrolls away from it; a button then jumps back to the
// end. It builds only the lines in view, as ui.List does, so logs of any
// length scroll as fast as short ones, and the user selects and copies
// text across the lines it built; Log.Text has all of it.
func View(c *ui.Context, l *Log, filter string) ui.Element {
	t := c.Theme()
	n, lineAt := l.rows(filter)
	l.list.FollowEnd = true
	l.list.Key = func(row int) any { return lineAt(row) }
	palette := lightPalette
	if t.Dark {
		palette = darkPalette
	}
	view := ui.Column(c)
	view.Children(func() {
		ui.List(c, &l.list, n, func(row int) {
			ln := l.lines[lineAt(row)-l.dropped]
			ui.RichText(c, spans(ln, filter, palette, t)...).Padding(0, t.Space(3))
		}).Grow(1).Font("monospace").FontSize(t.FontSize-1).LineHeight(1.4).Padding(t.Space(2), 0).Selectable().Children(func() {
			if n == 0 && filter != "" {
				ui.Text(c, "No lines match").TextColor(t.TextMuted).Padding(t.Space(3))
			}
		})
		if n > 0 && !l.list.AtEnd() {
			if ui.Button(c, "Jump to latest ↓").Absolute().Right(t.Space(4)).Bottom(t.Space(4)).Clicked() {
				l.list.ScrollToEnd()
			}
		}
	})
	return view
}

// spans returns ln in spans of its styles, with what matches filter
// highlighted.
func spans(ln line, filter string, palette *[16]ui.Color, t *ui.Theme) []ui.Span {
	if ln.text == "" {
		return []ui.Span{{Text: " "}}
	}
	var hits [][2]int
	if filter != "" {
		lower, q := strings.ToLower(ln.text), strings.ToLower(filter)
		// Lowering keeps the byte offsets of ASCII, which the matches of
		// most filters are; others highlight nothing rather than wrong
		// bytes.
		if len(lower) == len(ln.text) {
			for i := 0; ; {
				k := strings.Index(lower[i:], q)
				if k < 0 {
					break
				}
				hits = append(hits, [2]int{i + k, i + k + len(q)})
				i += k + len(q)
			}
		}
	}
	var out []ui.Span
	start := 0
	for _, r := range ln.runs {
		base := spanOf(r.st, palette, t)
		for start < r.end {
			end, hit := r.end, false
			for _, h := range hits {
				if h[1] <= start || h[0] >= end {
					continue
				}
				if h[0] > start {
					end = h[0]
				} else {
					end, hit = min(end, h[1]), true
				}
				break
			}
			s := base
			s.Text = ln.text[start:end]
			if hit {
				s.Background = t.Warning.Alpha(0.45)
			}
			out = append(out, s)
			start = end
		}
	}
	return out
}

// spanOf returns the span of a run in style st.
func spanOf(st style, palette *[16]ui.Color, t *ui.Theme) ui.Span {
	fg, bg := color(st.fg, palette, ui.Transparent), color(st.bg, palette, ui.Transparent)
	if st.inverse {
		if fg == ui.Transparent {
			fg = t.Text
		}
		if bg == ui.Transparent {
			bg = t.Background
		}
		fg, bg = bg, fg
	}
	if st.dim {
		if fg == ui.Transparent {
			fg = t.Text
		}
		fg = fg.Alpha(0.6)
	}
	s := ui.Span{Color: fg, Background: bg, Italic: st.italic, Underline: st.under, Strikethrough: st.strike}
	if st.bold {
		s.Weight = 700
	}
	return s
}

// color returns color c of a style, or otherwise for defaultColor.
func color(c int32, palette *[16]ui.Color, otherwise ui.Color) ui.Color {
	switch {
	case c == defaultColor:
		return otherwise
	case c&rgbFlag != 0:
		return ui.RGB(uint8(c>>16), uint8(c>>8), uint8(c))
	case c < 16:
		return palette[c]
	case c < 232:
		// xterm's 6×6×6 cube.
		c -= 16
		level := func(v int32) uint8 {
			if v == 0 {
				return 0
			}
			return uint8(55 + 40*v)
		}
		return ui.RGB(level(c/36), level(c/6%6), level(c%6))
	}
	g := uint8(8 + 10*(c-232))
	return ui.RGB(g, g, g)
}

// The 16 colors of the palette in light and dark windows, as Visual
// Studio Code's terminal has them, which read on its background.
var (
	lightPalette = &[16]ui.Color{
		ui.Hex("#000000"), ui.Hex("#cd3131"), ui.Hex("#00bc00"), ui.Hex("#949800"),
		ui.Hex("#0451a5"), ui.Hex("#bc05bc"), ui.Hex("#0598bc"), ui.Hex("#555555"),
		ui.Hex("#666666"), ui.Hex("#cd3131"), ui.Hex("#14ce14"), ui.Hex("#b5ba00"),
		ui.Hex("#0451a5"), ui.Hex("#bc05bc"), ui.Hex("#0598bc"), ui.Hex("#a5a5a5"),
	}
	darkPalette = &[16]ui.Color{
		ui.Hex("#000000"), ui.Hex("#cd3131"), ui.Hex("#0dbc79"), ui.Hex("#e5e510"),
		ui.Hex("#2472c8"), ui.Hex("#bc3fbc"), ui.Hex("#11a8cd"), ui.Hex("#e5e5e5"),
		ui.Hex("#666666"), ui.Hex("#f14c4c"), ui.Hex("#23d18b"), ui.Hex("#f5f543"),
		ui.Hex("#3b8eea"), ui.Hex("#d670d6"), ui.Hex("#29b8db"), ui.Hex("#e5e5e5"),
	}
)
