package terminal

import (
	"math"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
)

// blink is how long the cursor shows, and then hides, as it blinks.
const blink = 600 * time.Millisecond

// clipMods are the modifiers of Copy and Paste: Command on macOS, and
// Control with Shift elsewhere, where Control+C interrupts the program.
var clipMods = func() ui.Modifiers {
	if runtime.GOOS == "darwin" {
		return ui.Super
	}
	return ui.Ctrl | ui.Shift
}()

// View shows t in the system's monospaced font, as many columns and rows
// as its size holds, which it gives t.Resize when it first sizes t and
// when the size changes. Give it a size, or grow it in its parent.
//
// Once the view has the keyboard focus, which a click gives it, keys go
// to the program as xterm sends them: text, as input methods compose it
// too, Enter, Backspace, Tab, Escape, the arrows, Home, End, Page Up and
// Down, Insert, Delete, F1 to F12, and Control with a letter, all with
// Escape before them while Alt is held. Command+C copies the text
// selected and Command+V pastes, bracketed when the program asks
// (Control+Shift+C and V elsewhere); other Command keys are left to the
// app's shortcuts. Dragging selects cells, and a double click a word. The
// wheel scrolls back through the lines that scrolled off the screen,
// until the program writes again. The cursor is a block that blinks
// while the view has the focus, and an outline while it does not. The
// default colors and the 16 of the palette follow the theme, light or
// dark.
func View(c *ui.Context, t *Terminal) ui.Element {
	th := c.Theme()
	box := ui.Box(c).Focusable().FocusRing(false).Clip().Cursor(ui.CursorText).Label("Terminal")
	b := box.LayoutBounds()
	focused := box.Focused()
	t.metrics(th)
	now := c.Now()
	if focused != t.focused || t.blinked.IsZero() {
		t.focused, t.blinked = focused, now
	}
	shown := true
	if focused {
		since := now.Sub(t.blinked)
		shown = since/blink%2 == 0
		c.After(blink - since%blink)
	}

	t.mu.Lock()
	t.init()
	t.services = c.Services()
	cols, rows := t.cols, t.rows
	if b.W > 0 && b.H > 0 {
		cols, rows = max(int((b.W-2*t.pad)/t.cw), 2), max(int((b.H-2*t.pad)/t.ch), 1)
		if cols != t.cols || rows != t.rows {
			t.resize(cols, rows)
		}
	}
	answers := t.answers
	t.answers = nil
	caret := ui.Rect{X: t.pad + float32(t.cur.x)*t.cw, Y: t.pad + float32(t.cur.y+t.back)*t.ch, W: t.cw, H: t.ch}
	t.mu.Unlock()

	if size := [2]int{cols, rows}; b.W > 0 && size != t.told {
		t.told = size
		if t.Resize != nil {
			t.Resize(cols, rows)
		}
	}
	t.send(answers)
	box.HandleInput(t.input).TextCaret(caret).InputMethod(c.Modifiers()&ui.Alt == 0)
	box.Draw(func(p *ui.Painter, r ui.Rect) {
		// The size in cells follows the box's in the last frame: changed
		// since, the view builds again.
		if r.W != b.W || r.H != b.H {
			c.Invalidate()
		}
		t.paint(p, r, th, focused, shown)
	})
	return box
}

// metrics measures the cells in the theme's font size.
func (t *Terminal) metrics(th *ui.Theme) {
	if th.FontSize == t.size {
		return
	}
	t.size, t.pad, t.shaped = th.FontSize, th.Space(2), nil
	f := ui.Font{Family: "monospace", Size: t.size}
	t.cw = 1
	for _, g := range ui.Shape("M", f) {
		t.cw = max(g.Advance, 1)
	}
	m := f.Metrics()
	t.ch = max(float32(math.Ceil(float64(m.Ascent+m.Descent+m.LineGap))), 1)
	t.ascent = (t.ch-m.Ascent-m.Descent)/2 + m.Ascent
}

func (t *Terminal) send(b []byte) {
	if len(b) > 0 && t.Send != nil {
		t.Send(b)
	}
}

func (t *Terminal) input(ev ui.InputEvent) bool {
	switch ev.Kind {
	case ui.InputKeyDown:
		return t.key(ev)
	case ui.InputKeyUp:
		t.swallow = false
	case ui.InputText:
		if t.swallow {
			t.swallow = false
			return true
		}
		t.preedit, t.blinked = "", ev.Time
		t.send([]byte(ev.Text))
		return true
	case ui.InputCompose:
		t.preedit = ev.Text
		return true
	case ui.InputCommand:
		switch ev.Text {
		case "copy":
			t.copy()
			return true
		case "paste":
			t.paste()
			return true
		}
	case ui.InputPointerDown, ui.InputPointerMove, ui.InputPointerUp:
		return t.pointer(ev)
	case ui.InputScroll:
		return t.wheel(ev)
	}
	return false
}

func (t *Terminal) key(ev ui.InputEvent) bool {
	mods, k := ev.Mods, ev.Key
	if mods == clipMods && (k == ui.KeyC || k == ui.KeyV) {
		if k == ui.KeyC {
			t.copy()
		} else {
			t.paste()
		}
		return true
	}
	if mods&ui.Super != 0 {
		return false
	}
	t.mu.Lock()
	app := t.appCursor
	t.mu.Unlock()
	s := keySeq(k, mods, app)
	if s == "" {
		// The text the key types comes next, as Shift and the keyboard's
		// layout make it.
		return mods&(ui.Ctrl|ui.Alt) == 0 && usChar(k, false) != 0
	}
	if mods&ui.Alt != 0 {
		// What Option types with the key on macOS comes next: the
		// character without it goes instead.
		s, t.swallow = "\x1b"+s, true
	}
	t.blinked = ev.Time
	t.send([]byte(s))
	return true
}

// keySeq returns what key k sends with mods held, as xterm sends it, the
// arrows, Home and End as the program's application mode has them with
// app; "" for a key that types text.
func keySeq(k ui.Key, mods ui.Modifiers, app bool) string {
	s, ok := keys[k]
	switch {
	case k == ui.KeyTab && mods&ui.Shift != 0:
		return "\x1b[Z"
	case ok && app && len(s) == 3:
		// The arrows, Home and End send SS3 in application mode; F1 to F4,
		// the other keys of 3 bytes, send it already.
		return "\x1bO" + s[2:]
	case ok:
		return s
	case mods&ui.Ctrl != 0 && k >= ui.KeyA && k <= ui.KeyZ:
		return string(rune(k-ui.KeyA) + 1)
	case mods&ui.Alt != 0 && usChar(k, false) != 0:
		return string(usChar(k, mods&ui.Shift != 0))
	}
	return ""
}

// keys are what the keys that type no text send.
var keys = map[ui.Key]string{
	ui.KeyEnter: "\r", ui.KeyBackspace: "\x7f", ui.KeyTab: "\t", ui.KeyEscape: "\x1b",
	ui.KeyUp: "\x1b[A", ui.KeyDown: "\x1b[B", ui.KeyRight: "\x1b[C", ui.KeyLeft: "\x1b[D",
	ui.KeyHome: "\x1b[H", ui.KeyEnd: "\x1b[F", ui.KeyInsert: "\x1b[2~", ui.KeyDelete: "\x1b[3~",
	ui.KeyPageUp: "\x1b[5~", ui.KeyPageDown: "\x1b[6~",
	ui.KeyF1: "\x1bOP", ui.KeyF2: "\x1bOQ", ui.KeyF3: "\x1bOR", ui.KeyF4: "\x1bOS",
	ui.KeyF5: "\x1b[15~", ui.KeyF6: "\x1b[17~", ui.KeyF7: "\x1b[18~", ui.KeyF8: "\x1b[19~",
	ui.KeyF9: "\x1b[20~", ui.KeyF10: "\x1b[21~", ui.KeyF11: "\x1b[23~", ui.KeyF12: "\x1b[24~",
}

// symbolKeys are the keys of symbols, in the order of their characters in
// usChar.
var symbolKeys = []ui.Key{
	ui.KeyMinus, ui.KeyEqual, ui.KeyComma, ui.KeyPeriod, ui.KeySlash, ui.KeySemicolon,
	ui.KeyQuote, ui.KeyBracketLeft, ui.KeyBracketRight, ui.KeyBackslash, ui.KeyBackquote, ui.KeySpace,
}

// usChar returns the character key k types on a US keyboard, with Shift
// held or not, or 0 for a key that types none.
func usChar(k ui.Key, shift bool) byte {
	switch {
	case k >= ui.KeyA && k <= ui.KeyZ && shift:
		return 'A' + byte(k-ui.KeyA)
	case k >= ui.KeyA && k <= ui.KeyZ:
		return 'a' + byte(k-ui.KeyA)
	case k >= ui.Key0 && k <= ui.Key9 && shift:
		return ")!@#$%^&*("[k-ui.Key0]
	case k >= ui.Key0 && k <= ui.Key9:
		return '0' + byte(k-ui.Key0)
	}
	i := slices.Index(symbolKeys, k)
	switch {
	case i < 0:
		return 0
	case shift:
		return "_+<>?:\"{}|~ "[i]
	}
	return "-=,./;'[]\\` "[i]
}

func (t *Terminal) copy() {
	t.mu.Lock()
	text := t.selected()
	t.mu.Unlock()
	if text != "" {
		t.services.WriteClipboard(text)
	}
}

// paste sends the clipboard's text, its line breaks as Enter sends them.
func (t *Terminal) paste() {
	text := strings.NewReplacer("\r\n", "\r", "\n", "\r").Replace(t.services.ReadClipboard())
	if text == "" {
		return
	}
	t.mu.Lock()
	bracketed := t.bracketed
	t.mu.Unlock()
	if bracketed {
		// The end of a bracketed paste in the text would end it early,
		// and the program would take what follows as typed.
		text = "\x1b[200~" + strings.ReplaceAll(text, "\x1b[201~", "") + "\x1b[201~"
	}
	t.send([]byte(text))
}

// pointer selects the cells the pointer drags across, or the word it
// double-clicks.
func (t *Terminal) pointer(ev ui.InputEvent) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	col := max(0, min(int((ev.X-t.pad)/t.cw), t.cols-1))
	row := max(0, min(int((ev.Y-t.pad)/t.ch), t.rows-1))
	at := point{t.firstShown() + row, col}
	switch {
	case ev.Kind == ui.InputPointerDown && ev.Button == 0:
		t.sel, t.selecting = selection{a: at, b: at}, ev.Clicks < 2
		if !t.selecting {
			t.sel.a, t.sel.b = t.word(at)
			t.sel.on = true
		}
		return true
	case ev.Kind == ui.InputPointerMove && t.selecting:
		t.sel.b, t.sel.on = at, true
		return true
	case ev.Kind == ui.InputPointerUp && t.selecting:
		t.selecting = false
		return true
	}
	return false
}

// wheel scrolls the view back through the scrollback, or on to the
// bottom.
func (t *Terminal) wheel(ev ui.InputEvent) bool {
	dy := ev.DY
	if !ev.Precise {
		// A notch of a wheel comes as 40 DIPs; it scrolls three rows.
		dy = dy / 40 * 3 * t.ch
	}
	t.scrolled += dy
	rows := int(t.scrolled / t.ch)
	t.scrolled -= float32(rows) * t.ch
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.alt {
		t.back = max(0, min(t.back-rows, len(t.sb)))
	}
	return true
}

// paint paints the rows the view shows, the selection and the cursor.
func (t *Terminal) paint(p *ui.Painter, r ui.Rect, th *ui.Theme, focused, shown bool) {
	pal := lightPalette
	if th.Dark {
		pal = darkPalette
	}
	p.Fill(r, th.Background, 0)
	t.mu.Lock()
	defer t.mu.Unlock()
	x0, y0 := r.X+t.pad, r.Y+t.pad
	first := t.firstShown()
	a, b := t.sel.ordered()
	for row := range t.rows {
		n := first + row
		from, to := -1, -1
		if t.sel.on && n >= a.line && n <= b.line {
			from, to = 0, t.cols-1
			if n == a.line {
				from = a.col
			}
			if n == b.line {
				to = b.col
			}
		}
		t.paintLine(p, t.lineAt(n), x0, y0+float32(row)*t.ch, from, to, pal, th)
	}

	cur := t.cur
	if t.hidden || cur.y+t.back >= t.rows {
		return
	}
	line := t.lines[cur.y]
	w := 1
	if cur.x+1 < t.cols && line[cur.x+1].r == wideTail {
		w = 2
	}
	box := ui.Rect{X: x0 + float32(cur.x)*t.cw, Y: y0 + float32(cur.y+t.back)*t.ch, W: float32(w) * t.cw, H: t.ch}
	switch {
	case t.preedit != "":
		var rs []rune
		var cols []int
		x := cur.x
		for _, r := range t.preedit {
			if w := runeWidth(r); w > 0 {
				rs, cols = append(rs, r), append(cols, x)
				x += w
			}
		}
		box.W = float32(x-cur.x) * t.cw
		p.Fill(box, th.Background, 0)
		t.glyphs(p, rs, cols, x0, box.Y, th.Text, 0)
		p.Fill(ui.Rect{X: box.X, Y: box.Y + t.ascent + 2, W: box.W, H: 1}, th.Text, 0)
	case !focused:
		p.Stroke(box, th.Text, 0, 1)
	case shown:
		p.Fill(box, th.Text, 0)
		c := line[cur.x]
		if arms, ok := boxArms[c.r]; ok {
			t.boxLines(p, arms, box.X, box.Y, th.Background)
		} else if c.r > ' ' {
			t.glyphs(p, []rune{c.r}, []int{cur.x}, x0, box.Y, th.Background, c.st.attrs&(bold|italic))
		}
	}
}

// boxArms are the box drawing characters of light lines, which the view
// draws so that they join across rows, as the arms each has from the
// middle of its cell: up 1, down 2, left 4 and right 8. Rounded corners
// draw square.
var boxArms = map[rune]uint8{
	'─': 12, '│': 3, '┌': 10, '┐': 6, '└': 9, '┘': 5, '├': 11, '┤': 7,
	'┬': 14, '┴': 13, '┼': 15, '╭': 10, '╮': 6, '╰': 9, '╯': 5,
}

// boxLines draws the arms of a box drawing character in the cell at x, y.
func (t *Terminal) boxLines(p *ui.Painter, arms uint8, x, y float32, c ui.Color) {
	cx, cy := x+t.cw/2, y+t.ch/2
	if arms&12 != 0 {
		x1, x2 := cx, cx+1
		if arms&4 != 0 {
			x1 = x
		}
		if arms&8 != 0 {
			x2 = x + t.cw
		}
		p.Fill(ui.Rect{X: x1, Y: cy, W: x2 - x1, H: 1}, c, 0)
	}
	if arms&3 != 0 {
		y1, y2 := cy, cy+1
		if arms&1 != 0 {
			y1 = y
		}
		if arms&2 != 0 {
			y2 = y + t.ch
		}
		p.Fill(ui.Rect{X: cx, Y: y1, W: 1, H: y2 - y1}, c, 0)
	}
}

// paintLine paints a line of cells from x0 on the row at y: their
// backgrounds, the selection from column from to column to, and their
// text.
func (t *Terminal) paintLine(p *ui.Painter, line []cell, x0, y float32, from, to int, pal *[16]ui.Color, th *ui.Theme) {
	line = line[:min(len(line), t.cols)]
	cells := func(x, w int) ui.Rect {
		return ui.Rect{X: x0 + float32(x)*t.cw, Y: y, W: float32(w) * t.cw, H: t.ch}
	}
	start, has, last := 0, false, ui.Color{}
	for x := 0; x <= len(line); x++ {
		var bg ui.Color
		on := false
		if x < len(line) {
			_, bg, on = colors(line[x].st, pal, th)
		}
		if x == len(line) || on != has || on && bg != last {
			if has {
				p.Fill(cells(start, x-start), last, 0)
			}
			start, has, last = x, on, bg
		}
	}
	if from >= 0 && to >= from {
		p.Fill(cells(from, to-from+1), th.Selection, 0)
	}

	end := len(line)
	for end > 0 && (line[end-1].r == 0 || line[end-1].r == ' ') {
		end--
	}
	var run []rune
	var cols []int
	var fg ui.Color
	var variant uint8
	for x, c := range line[:end] {
		if c.r == wideTail {
			continue
		}
		f, _, _ := colors(c.st, pal, th)
		v := c.st.attrs & (bold | italic)
		if len(run) > 0 && (f != fg || v != variant) {
			t.glyphs(p, run, cols, x0, y, fg, variant)
			run, cols = run[:0], cols[:0]
		}
		fg, variant = f, v
		if arms, ok := boxArms[c.r]; ok {
			t.boxLines(p, arms, x0+float32(x)*t.cw, y, f)
			continue
		}
		r := c.r
		if r == 0 {
			r = ' '
		}
		run, cols = append(run, r), append(cols, x)
		w := 1
		if x+1 < len(line) && line[x+1].r == wideTail {
			w = 2
		}
		if c.st.attrs&underline != 0 {
			p.Fill(ui.Rect{X: x0 + float32(x)*t.cw, Y: y + t.ascent + 2, W: float32(w) * t.cw, H: 1}, f, 0)
		}
		if c.st.attrs&strike != 0 {
			p.Fill(ui.Rect{X: x0 + float32(x)*t.cw, Y: y + t.ch/2, W: float32(w) * t.cw, H: 1}, f, 0)
		}
	}
	if len(run) > 0 {
		t.glyphs(p, run, cols, x0, y, fg, variant)
	}
}

// shapeKey is text shaped bold, italic or both, as variant has it.
type shapeKey struct {
	text    string
	variant uint8
}

// glyphs draws runes in color c, each centered in its cells from its
// column of cols, from x0 on the row at y.
func (t *Terminal) glyphs(p *ui.Painter, runes []rune, cols []int, x0, y float32, c ui.Color, variant uint8) {
	k := shapeKey{string(runes), variant}
	shaped, ok := t.shaped[k]
	if !ok {
		if t.shaped == nil || len(t.shaped) > 4096 {
			t.shaped = map[shapeKey][]ui.Glyph{}
		}
		f := ui.Font{Family: "monospace", Size: t.size, Italic: variant&italic != 0}
		if variant&bold != 0 {
			f.Weight = 700
		}
		shaped = ui.Shape(k.text, f)
		t.shaped[k] = shaped
	}
	t.placed = append(t.placed[:0], shaped...)
	for i, g := range t.placed {
		if g.Cluster < len(cols) {
			w := float32(runeWidth(runes[g.Cluster])) * t.cw
			t.placed[i].X = float32(cols[g.Cluster])*t.cw + (w-g.Advance)/2
		}
	}
	p.Glyphs(t.placed, x0, y+t.ascent, c)
}

// colors returns the colors of a cell of style st, and whether its
// background is other than the terminal's.
func colors(st style, pal *[16]ui.Color, th *ui.Theme) (fg, bg ui.Color, hasBg bool) {
	fg, bg, hasBg = colorOf(st.fg, pal, th.Text), colorOf(st.bg, pal, th.Background), st.bg != 0
	if st.attrs&inverse != 0 {
		fg, bg, hasBg = bg, fg, true
	}
	if st.attrs&faint != 0 {
		fg = fg.Alpha(0.6)
	}
	return fg, bg, hasBg
}

// colorOf returns color c of a style, or def for the default color.
func colorOf(c uint32, pal *[16]ui.Color, def ui.Color) ui.Color {
	switch {
	case c == 0:
		return def
	case c&rgbColor != 0:
		return ui.RGB(uint8(c>>16), uint8(c>>8), uint8(c))
	}
	i := c & 0xff
	switch {
	case i < 16:
		return pal[i]
	case i < 232:
		// xterm's 6×6×6 cube.
		i -= 16
		level := func(v uint32) uint8 {
			if v == 0 {
				return 0
			}
			return uint8(55 + 40*v)
		}
		return ui.RGB(level(i/36), level(i/6%6), level(i%6))
	}
	g := uint8(8 + 10*(i-232))
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
