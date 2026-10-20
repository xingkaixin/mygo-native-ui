// Package terminal is a terminal emulator, enough of xterm for shells and
// the programs they run, as less, vim, top and htop. A Terminal keeps the
// screen the program draws, in its colors and styles, with the 5000 lines
// that scrolled off its top, and the title it set; a View shows it and
// sends the program what the user types.
//
//	app.term.Send = func(b []byte) { pty.Write(b) }
//	app.term.Resize = func(cols, rows int) { setSize(pty, cols, rows) }
//	go io.Copy(&app.term, pty) // Write is safe from the reading goroutine
//
//	terminal.View(c, &app.term).Fill().AutoFocus()
//
// It runs no program itself: the app starts one in a pseudo-terminal and
// connects the two, as examples/terminal does.
package terminal

import (
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
)

// maxScrollback is how many lines that scrolled off the screen a
// terminal keeps.
const maxScrollback = 5000

// Terminal is a terminal's screen, the lines that scrolled off it, and the
// cursor, modes and title the program set. Write the program's output to
// it from any goroutine, as the one reading the pseudo-terminal: each
// Write asks its view for a frame. Show it in one View at a time. The zero
// Terminal is an empty screen of 80 columns and 24 rows until a view sizes
// it.
type Terminal struct {
	// Send gets what the terminal sends the program, to write to the
	// pseudo-terminal: the keys typed, pastes, and the answers to the
	// program's queries of the cursor's position and the terminal's
	// attributes. The view calls it on the main thread.
	Send func(b []byte)
	// Resize gets the terminal's size in cells when a view changes it,
	// on the main thread, to set the pseudo-terminal's.
	Resize func(cols, rows int)

	// mu guards what follows against Write, which may run on another
	// goroutine.
	mu         sync.Mutex
	cols, rows int
	// screens are the main screen and the alternate one of full-screen
	// programs, lines the one shown.
	screens [2][][]cell
	lines   [][]cell
	alt     bool
	// sb is the lines scrolled off the top of the main screen, oldest
	// first, and dropped counts those let go of before them, which
	// numbers lines from the first written: a selection holds to them.
	sb          [][]cell
	dropped     int
	cur, saved  cursor
	top, bottom int // the scroll region's first and last rows
	appCursor   bool
	noWrap      bool
	hidden      bool
	bracketed   bool
	title       string
	answers     []byte
	// last is the character written last, which REP repeats.
	last rune

	// The parser's state, the bytes of the sequence or string it reads,
	// with a CSI's intermediate byte, whether the string is an OSC, and
	// the bytes of a character cut short so far.
	state int
	seq   []byte
	inter byte
	osc   bool
	u8    []byte

	// back is how many rows the view is scrolled back from the bottom,
	// and sel what the pointer selects.
	back      int
	sel       selection
	selecting bool
	// services are the services of the window showing the terminal,
	// whose frame Write asks for.
	services ui.Services

	// What only the main thread uses: the size last given to Resize, the
	// font's size and the cells', the text shaped, the cursor's blinking,
	// and the input in flight.
	told     [2]int
	size     float32
	cw, ch   float32
	ascent   float32
	pad      float32
	shaped   map[shapeKey][]ui.Glyph
	placed   []ui.Glyph
	focused  bool
	blinked  time.Time
	preedit  string
	swallow  bool
	scrolled float32
}

// cell is a character on the screen: 0 for none, as blank, and wideTail
// for the right half of a wide character.
type cell struct {
	r  rune
	st style
}

const wideTail = -1

// style is how a cell shows. A color is 0 for the default one, a
// palette's index from 0 to 255 with paletteColor set, or an RGB color
// with rgbColor set.
type style struct {
	fg, bg uint32
	attrs  uint8
}

const (
	paletteColor = 1 << 24
	rgbColor     = 2 << 24
)

const (
	bold uint8 = 1 << iota
	faint
	italic
	underline
	inverse
	strike
)

// cursor is where the next character goes and how it shows: in DEC's
// line drawing characters, and on the next line when wrap is set, as it
// is after the last column was written.
type cursor struct {
	x, y     int
	st       style
	graphics bool
	wrap     bool
}

// The parser's states.
const (
	ground = iota
	escape
	charset
	csi
	str
	strEscape
)

// Write takes the program's output: text, control characters, and the
// escape sequences that move the cursor, edit and scroll the screen, set
// colors, styles, modes and the title, and query the terminal, whose
// answers go to Send. A sequence or a character that p cuts short waits
// for the rest in the next Write. It scrolls the view back to the bottom,
// and never fails.
func (t *Terminal) Write(p []byte) (int, error) {
	t.mu.Lock()
	t.init()
	for _, b := range p {
		t.feed(b)
	}
	t.back = 0
	services := t.services
	t.mu.Unlock()
	services.Invalidate()
	return len(p), nil
}

// Title returns the title the program set last, with OSC 0 or 2.
func (t *Terminal) Title() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.title
}

func (t *Terminal) init() {
	if t.cols == 0 {
		t.cols, t.rows = 80, 24
		t.reset()
	}
}

// reset clears the screens and the modes, as ESC c does; the scrollback
// stays.
func (t *Terminal) reset() {
	for i := range t.screens {
		t.screens[i] = make([][]cell, t.rows)
		for y := range t.screens[i] {
			t.screens[i][y] = make([]cell, t.cols)
		}
	}
	t.lines, t.alt = t.screens[0], false
	t.cur, t.saved = cursor{}, cursor{}
	t.top, t.bottom = 0, t.rows-1
	t.appCursor, t.noWrap, t.hidden, t.bracketed = false, false, false, false
}

// resize sizes the screens to cols and rows, moving the lines above the
// cursor into the scrollback when fewer rows leave it below them.
func (t *Terminal) resize(cols, rows int) {
	if n := t.cur.y - rows + 1; !t.alt && n > 0 {
		for _, l := range t.screens[0][:n] {
			t.keep(l)
		}
		t.screens[0] = t.screens[0][n:]
		t.cur.y -= n
	}
	for i, lines := range t.screens {
		lines = lines[:min(len(lines), rows)]
		for len(lines) < rows {
			lines = append(lines, nil)
		}
		for y, l := range lines {
			l = l[:min(len(l), cols)]
			for len(l) < cols {
				l = append(l, cell{})
			}
			lines[y] = l
		}
		t.screens[i] = lines
	}
	t.cols, t.rows = cols, rows
	t.lines = t.screens[0]
	if t.alt {
		t.lines = t.screens[1]
	}
	t.top, t.bottom = 0, rows-1
	t.moveTo(t.cur.x, t.cur.y)
}

func (t *Terminal) feed(b byte) {
	switch t.state {
	case ground:
		if len(t.u8) > 0 && b < utf8.RuneSelf {
			t.u8 = t.u8[:0]
			t.print(utf8.RuneError)
		}
		switch {
		case b == 0x1b:
			t.state = escape
		case b < 0x20:
			t.control(b)
		case b == 0x7f:
		case b < utf8.RuneSelf:
			t.print(rune(b))
		default:
			t.u8 = append(t.u8, b)
			if !utf8.FullRune(t.u8) {
				return
			}
			r, n := utf8.DecodeRune(t.u8)
			rest := string(t.u8[n:])
			t.u8 = t.u8[:0]
			t.print(r)
			for i := range len(rest) {
				t.feed(rest[i])
			}
		}
	case escape:
		t.state = ground
		t.escape(b)
	case charset:
		if t.inter == '(' {
			t.cur.graphics = b == '0'
		}
		t.state = ground
	case csi:
		switch {
		case b >= 0x40 && b <= 0x7e:
			t.state = ground
			t.dispatch(b)
		case b >= 0x30:
			t.seq = append(t.seq, b)
		case b >= 0x20:
			t.inter = b
		case b == 0x1b:
			t.state = escape
		default:
			t.control(b)
		}
	case str:
		switch {
		case b == 0x07:
			t.state = ground
			t.endString()
		case b == 0x1b:
			t.state = strEscape
		case t.osc && len(t.seq) < 4096:
			t.seq = append(t.seq, b)
		}
	case strEscape:
		t.endString()
		t.state = ground
		if b != '\\' {
			// An escape that is not ST ends the string and starts a
			// sequence of its own.
			t.state = escape
			t.feed(b)
		}
	}
}

func (t *Terminal) escape(b byte) {
	switch b {
	case '[':
		t.state, t.seq, t.inter = csi, t.seq[:0], 0
	case ']', 'P', 'X', '^', '_':
		t.state, t.seq, t.osc = str, t.seq[:0], b == ']'
	case '(', ')', '*', '+', '#', '%':
		t.state, t.inter = charset, b
	case '7':
		t.saved = t.cur
	case '8':
		t.restore()
	case 'M':
		t.reverseIndex()
	case 'c':
		t.reset()
	}
}

func (t *Terminal) control(b byte) {
	switch b {
	case '\b':
		t.moveTo(t.cur.x-1, t.cur.y)
	case '\t':
		t.cur.x = min((t.cur.x/8+1)*8, t.cols-1)
	case '\n', '\v', '\f':
		t.index()
	case '\r':
		t.moveTo(0, t.cur.y)
	}
}

func (t *Terminal) endString() {
	if !t.osc {
		return
	}
	if code, title, _ := strings.Cut(string(t.seq), ";"); code == "0" || code == "2" {
		t.title = title
	}
}

// decGraphics are the characters of DEC's line drawing set for ` to ~.
var decGraphics = []rune("◆▒␉␌␍␊°±␤␋┘┐┌└┼⎺⎻─⎼⎽├┤┴┬│≤≥π≠£·")

// print writes r at the cursor and moves it on, to the next line first
// when the last character filled the line.
func (t *Terminal) print(r rune) {
	if t.cur.graphics && r >= '`' && r <= '~' {
		r = decGraphics[r-'`']
	}
	w := runeWidth(r)
	if w == 0 {
		return
	}
	c := &t.cur
	if c.wrap && !t.noWrap {
		c.x = 0
		t.index()
	}
	c.wrap = false
	if c.x+w > t.cols {
		if t.noWrap {
			c.x = t.cols - w
		} else {
			t.erase(t.lines[c.y], c.x, t.cols)
			c.x = 0
			t.index()
		}
	}
	line := t.lines[c.y]
	for x := c.x; x < c.x+w; x++ {
		// Writing over half a wide character blanks the other half.
		if line[x].r == wideTail && x > 0 {
			line[x-1].r = 0
		}
		if x+1 < t.cols && line[x+1].r == wideTail {
			line[x+1].r = 0
		}
	}
	line[c.x], t.last = cell{r, c.st}, r
	if w == 2 {
		line[c.x+1] = cell{wideTail, c.st}
	}
	if c.x+w >= t.cols {
		c.x, c.wrap = t.cols-1, true
	} else {
		c.x += w
	}
}

// wide are the ranges of East Asian wide characters and emoji, which take
// two cells.
var wide = [][2]rune{
	{0x1100, 0x115f}, {0x2e80, 0x303e}, {0x3041, 0x33ff}, {0x3400, 0x4dbf},
	{0x4e00, 0x9fff}, {0xa000, 0xa4cf}, {0xa960, 0xa97f}, {0xac00, 0xd7a3},
	{0xf900, 0xfaff}, {0xfe10, 0xfe19}, {0xfe30, 0xfe6f}, {0xff00, 0xff60},
	{0xffe0, 0xffe6}, {0x1f300, 0x1f64f}, {0x1f680, 0x1f6ff}, {0x1f900, 0x1faff},
	{0x20000, 0x3fffd},
}

// runeWidth returns the cells r takes: 2 for wide characters, and 0 for
// marks that combine with the character before, which the screen leaves
// out.
func runeWidth(r rune) int {
	if r < 0x300 {
		return 1
	}
	if unicode.In(r, unicode.Mn, unicode.Me, unicode.Cf) {
		return 0
	}
	for _, w := range wide {
		if r < w[0] {
			break
		}
		if r <= w[1] {
			return 2
		}
	}
	return 1
}

// moveTo moves the cursor to x, y, within the screen.
func (t *Terminal) moveTo(x, y int) {
	t.cur.x, t.cur.y = max(0, min(x, t.cols-1)), max(0, min(y, t.rows-1))
	t.cur.wrap = false
}

// index moves the cursor down a line, scrolling the region up at its
// bottom.
func (t *Terminal) index() {
	t.cur.wrap = false
	switch {
	case t.cur.y == t.bottom:
		t.scroll(t.top, t.bottom, 1)
	case t.cur.y < t.rows-1:
		t.cur.y++
	}
}

func (t *Terminal) reverseIndex() {
	t.cur.wrap = false
	switch {
	case t.cur.y == t.top:
		t.scroll(t.top, t.bottom, -1)
	case t.cur.y > 0:
		t.cur.y--
	}
}

// scroll moves the lines from row top to row bottom up by n, or down for
// n below 0, with blank lines where they leave. Lines leaving the top of
// the main screen go to the scrollback.
func (t *Terminal) scroll(top, bottom, n int) {
	region := t.lines[top : bottom+1]
	if n > 0 {
		n = min(n, len(region))
		if top == 0 && !t.alt {
			for _, l := range region[:n] {
				t.keep(l)
			}
		}
		copy(region, region[n:])
		for i := len(region) - n; i < len(region); i++ {
			region[i] = t.blankLine()
		}
		return
	}
	n = min(-n, len(region))
	copy(region[n:], region)
	for i := range n {
		region[i] = t.blankLine()
	}
}

// keep adds l to the scrollback, letting go of the oldest line past
// maxScrollback.
func (t *Terminal) keep(l []cell) {
	if len(t.sb) == maxScrollback {
		t.sb[0] = nil
		t.sb = t.sb[1:]
		t.dropped++
	}
	t.sb = append(t.sb, l)
}

func (t *Terminal) blankLine() []cell {
	l := make([]cell, t.cols)
	t.erase(l, 0, t.cols)
	return l
}

// erase blanks the cells of l from from to to, in the background color
// set, as xterm does.
func (t *Terminal) erase(l []cell, from, to int) {
	blank := cell{st: style{bg: t.cur.st.bg}}
	for i := max(from, 0); i < min(to, len(l)); i++ {
		l[i] = blank
	}
}

func (t *Terminal) restore() {
	t.cur = t.saved
	t.moveTo(t.cur.x, t.cur.y)
}

// dispatch carries out the CSI sequence that final ends.
func (t *Terminal) dispatch(final byte) {
	if t.inter != 0 {
		return
	}
	params := string(t.seq)
	var private byte
	if params != "" && params[0] >= '<' {
		private, params = params[0], params[1:]
	}
	if final == 'm' && private == 0 {
		t.cur.st = sgr(t.cur.st, params)
		return
	}
	var ps []int
	for _, f := range strings.Split(params, ";") {
		n, _ := strconv.Atoi(f)
		ps = append(ps, n)
	}
	arg := func(i, def int) int {
		if i < len(ps) && ps[i] > 0 {
			return ps[i]
		}
		return def
	}
	if private == '?' && (final == 'h' || final == 'l') {
		for _, m := range ps {
			t.setMode(m, final == 'h')
		}
		return
	}
	if private != 0 {
		return
	}
	c, n := &t.cur, arg(0, 1)
	line := t.lines[c.y]
	switch final {
	case 'A':
		limit := 0
		if c.y >= t.top {
			limit = t.top
		}
		t.moveTo(c.x, max(c.y-n, limit))
	case 'B':
		limit := t.rows - 1
		if c.y <= t.bottom {
			limit = t.bottom
		}
		t.moveTo(c.x, min(c.y+n, limit))
	case 'C':
		t.moveTo(c.x+n, c.y)
	case 'D':
		t.moveTo(c.x-n, c.y)
	case 'E':
		t.moveTo(0, c.y+n)
	case 'F':
		t.moveTo(0, c.y-n)
	case 'G':
		t.moveTo(n-1, c.y)
	case 'H', 'f':
		t.moveTo(arg(1, 1)-1, n-1)
	case 'd':
		t.moveTo(c.x, n-1)
	case 'J':
		t.eraseDisplay(arg(0, 0))
	case 'K':
		switch arg(0, 0) {
		case 0:
			t.erase(line, c.x, t.cols)
		case 1:
			t.erase(line, 0, c.x+1)
		case 2:
			t.erase(line, 0, t.cols)
		}
	case 'X':
		t.erase(line, c.x, c.x+n)
	case '@':
		n = min(n, t.cols-c.x)
		copy(line[c.x+n:], line[c.x:])
		t.erase(line, c.x, c.x+n)
	case 'P':
		n = min(n, t.cols-c.x)
		copy(line[c.x:], line[c.x+n:])
		t.erase(line, t.cols-n, t.cols)
	case 'L', 'M':
		if c.y >= t.top && c.y <= t.bottom {
			if final == 'L' {
				n = -n
			}
			t.scroll(c.y, t.bottom, n)
			t.moveTo(0, c.y)
		}
	case 'S':
		t.scroll(t.top, t.bottom, n)
	case 'T':
		t.scroll(t.top, t.bottom, -n)
	case 'b':
		// Newer terminfo entries of xterm-256color have ncurses repeat a
		// character with REP.
		for range min(n, t.cols*t.rows) {
			t.print(t.last)
		}
	case 'r':
		if top, bottom := arg(0, 1)-1, arg(1, t.rows)-1; top < bottom && bottom < t.rows {
			t.top, t.bottom = top, bottom
			t.moveTo(0, 0)
		}
	case 'n':
		switch arg(0, 0) {
		case 5:
			t.answers = append(t.answers, "\x1b[0n"...)
		case 6:
			t.answers = append(t.answers, "\x1b["+strconv.Itoa(c.y+1)+";"+strconv.Itoa(c.x+1)+"R"...)
		}
	case 'c':
		if arg(0, 0) == 0 {
			t.answers = append(t.answers, "\x1b[?1;2c"...)
		}
	}
}

func (t *Terminal) eraseDisplay(mode int) {
	c := t.cur
	switch mode {
	case 0:
		t.erase(t.lines[c.y], c.x, t.cols)
		for _, l := range t.lines[c.y+1:] {
			t.erase(l, 0, t.cols)
		}
	case 1:
		for _, l := range t.lines[:c.y] {
			t.erase(l, 0, t.cols)
		}
		t.erase(t.lines[c.y], 0, c.x+1)
	case 2:
		for _, l := range t.lines {
			t.erase(l, 0, t.cols)
		}
	case 3:
		t.dropped += len(t.sb)
		t.sb = nil
	}
}

// setMode sets DEC private mode m on or off.
func (t *Terminal) setMode(m int, on bool) {
	switch m {
	case 1:
		t.appCursor = on
	case 7:
		t.noWrap = !on
	case 25:
		t.hidden = !on
	case 47, 1047, 1049:
		if on == t.alt {
			return
		}
		if on && m == 1049 {
			t.saved = t.cur
		}
		t.alt, t.sel, t.selecting = on, selection{}, false
		t.lines = t.screens[0]
		if on {
			t.lines = t.screens[1]
			for _, l := range t.lines {
				t.erase(l, 0, t.cols)
			}
		}
		if !on && m == 1049 {
			t.restore()
		}
	case 2004:
		t.bracketed = on
	}
}

// sgr returns st as the parameters of an SGR sequence set it.
func sgr(st style, params string) style {
	ps := strings.FieldsFunc(params, func(r rune) bool { return r == ';' || r == ':' })
	if len(ps) == 0 {
		return style{}
	}
	num := func(i int) uint32 {
		if i >= len(ps) {
			return 0
		}
		n, _ := strconv.Atoi(ps[i])
		return uint32(n)
	}
	for i := 0; i < len(ps); i++ {
		switch n := num(i); {
		case n == 0:
			st = style{}
		case n == 1:
			st.attrs |= bold
		case n == 2:
			st.attrs |= faint
		case n == 3:
			st.attrs |= italic
		case n == 4:
			st.attrs |= underline
		case n == 7:
			st.attrs |= inverse
		case n == 9:
			st.attrs |= strike
		case n == 21 || n == 22:
			st.attrs &^= bold | faint
		case n == 23:
			st.attrs &^= italic
		case n == 24:
			st.attrs &^= underline
		case n == 27:
			st.attrs &^= inverse
		case n == 29:
			st.attrs &^= strike
		case n >= 30 && n <= 37:
			st.fg = paletteColor | (n - 30)
		case n >= 90 && n <= 97:
			st.fg = paletteColor | (n - 90 + 8)
		case n == 39:
			st.fg = 0
		case n >= 40 && n <= 47:
			st.bg = paletteColor | (n - 40)
		case n >= 100 && n <= 107:
			st.bg = paletteColor | (n - 100 + 8)
		case n == 49:
			st.bg = 0
		case n == 38 || n == 48:
			var c uint32
			switch num(i + 1) {
			case 5:
				c, i = paletteColor|num(i+2)&0xff, i+2
			case 2:
				// 38;2;r;g;b, or 38:2::r:g:b with a color space id, which
				// FieldsFunc leaves out with the empty field.
				c, i = rgbColor|(num(i+2)&0xff)<<16|(num(i+3)&0xff)<<8|num(i+4)&0xff, i+4
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

// lineAt returns line n, numbered from the first written, of the
// scrollback or the screen shown; nil for one let go of.
func (t *Terminal) lineAt(n int) []cell {
	i := n - t.dropped
	switch {
	case i < 0:
		return nil
	case i < len(t.sb):
		return t.sb[i]
	case i-len(t.sb) < len(t.lines):
		return t.lines[i-len(t.sb)]
	}
	return nil
}

// firstShown returns the number of the line the view shows at its top.
func (t *Terminal) firstShown() int {
	return t.dropped + len(t.sb) - t.back
}

// point is a cell of a line, numbered from the first written.
type point struct{ line, col int }

// selection is the cells from a to b, both included, in either order.
type selection struct {
	a, b point
	on   bool
}

func (s selection) ordered() (point, point) {
	if s.b.line < s.a.line || s.b.line == s.a.line && s.b.col < s.a.col {
		return s.b, s.a
	}
	return s.a, s.b
}

// isWord reports whether a double click selects r with the characters
// around it: not a space, a quote or a bracket.
func isWord(r rune) bool {
	return r == wideTail || r > ' ' && !strings.ContainsRune("\"'`()[]{}<>|,;", r)
}

// word returns the ends of the word at p.
func (t *Terminal) word(p point) (point, point) {
	line := t.lineAt(p.line)
	if p.col >= len(line) || !isWord(line[p.col].r) {
		return p, p
	}
	a, b := p, p
	for a.col > 0 && isWord(line[a.col-1].r) {
		a.col--
	}
	for b.col+1 < len(line) && isWord(line[b.col+1].r) {
		b.col++
	}
	return a, b
}

// selected returns the text selected, its lines without the spaces that
// end them.
func (t *Terminal) selected() string {
	if !t.sel.on {
		return ""
	}
	a, b := t.sel.ordered()
	var lines []string
	for n := a.line; n <= b.line; n++ {
		line := t.lineAt(n)
		from, to := 0, len(line)-1
		if n == a.line {
			from = a.col
		}
		if n == b.line {
			to = min(to, b.col)
		}
		var rs []rune
		for x := from; x <= to; x++ {
			switch r := line[x].r; r {
			case wideTail:
			case 0:
				rs = append(rs, ' ')
			default:
				rs = append(rs, r)
			}
		}
		lines = append(lines, strings.TrimRight(string(rs), " "))
	}
	return strings.Join(lines, "\n")
}
