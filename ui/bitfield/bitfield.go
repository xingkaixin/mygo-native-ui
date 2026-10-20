// Package bitfield edits a register bit by bit, as datasheets draw
// registers: a cell for each bit under its number, the most significant
// on the left, in nibbles and bytes; the register's fields as colored
// brackets below their bits, each named with its value; and the whole
// value above as hexadecimal, decimal and binary text that edits it.
//
//	bitfield.View(c, &app.ctrl, &app.value, 32, []bitfield.Field{
//		{Name: "EN", Lsb: 0, Width: 1},
//		{Name: "MODE", Lsb: 1, Width: 2, Values: map[uint64]string{0: "Off", 1: "Slow", 2: "Fast"}},
//		{Name: "DIV", Lsb: 8, Width: 12},
//	})
package bitfield

import (
	"cmp"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
)

// Field is a field of a register: Width bits from bit Lsb up, named Name,
// and the names of some of its values, as "Fast" for 2, which show with
// them.
type Field struct {
	Name       string
	Lsb, Width int
	Values     map[uint64]string
}

// State is what a view keeps: the bit with the keyboard focus, a drag
// across bits, and the text typed in each field. Keep it in the app's
// state, one for each view. The zero State focuses the most significant
// bit.
type State struct {
	// focus is the bit with the focus, counted from the most significant.
	focus int
	// down is whether the pointer presses the bits, last the bit it went
	// over last, counted as focus is, or -1 for a press that began off
	// the bits, and set what it sets them to.
	down   bool
	last   int
	set    bool
	texts  [3]string
	inputs [3]ui.Handle
}

// bases are the fields above the bits: the value in hexadecimal, decimal
// and binary.
var bases = [3]struct {
	name, word, prefix string
	base               int
}{{"Hex", "hexadecimal", "0x", 16}, {"Dec", "decimal", "", 10}, {"Bin", "binary", "0b", 2}}

// View creates the editor of *value as a register of bits bits, 8, 16, 32
// or 64, whose fields are fields; the bits above the register's are
// cleared. Above the bits, the value shows in hexadecimal, decimal and
// binary, each in a field where typing a number that fits the register
// changes it, and anything else is marked invalid. Each bit shows 0 or 1
// under its number, set bits standing out, in rows of 32 at most. A
// field's set bits fill with its color, and its bracket below its bits
// carries its name, its value and the value's name; bits of no field show
// muted. The pointer over a field's bits or its label lights its bits.
//
// A click toggles a bit, and dragging on from it sets the bits it crosses
// to what the first became. While the bits have the keyboard focus, the
// arrows move it from bit to bit and Space toggles the bit it is on.
// Changed reports a new value.
func View(c *ui.Context, s *State, value *uint64, bits int, fields []Field) ui.Element {
	t := c.Theme()
	bits = max(1, min(64, bits))
	*value &= ^uint64(0) >> (64 - bits)
	g := layout(c, bits, fields, *value)
	changed := false
	view := ui.Column(c).Gap(t.Space(4)).Width(g.w)
	view.Children(func() {
		ui.Row(c).Wrap().Gap(t.Space(3)).Children(func() {
			for i := range bases {
				changed = s.text(c.Key(i), i, value, bits) || changed
			}
		})
		changed = s.grid(c, g, value) || changed
	})
	if changed {
		view.SetChanged()
	}
	return view
}

// text builds the field writing *value in bases[i], and sets *value to
// what is typed in it while that fits bits bits.
func (s *State) text(c *ui.Context, i int, value *uint64, bits int) bool {
	t := c.Theme()
	b := bases[i]
	if !s.inputs[i].Focused(c) {
		s.texts[i] = format(*value, bits, b.base, b.prefix)
	}
	widest := format(^uint64(0)>>(64-bits), bits, b.base, b.prefix)
	w, _ := c.MeasureText(0, ui.Span{Text: strings.Repeat("0", len(widest)), Font: "monospace", Size: t.FontSize})
	changed := false
	field := ui.Field(c, b.name, func() {
		in := ui.TextInput(c, &s.texts[i]).Bind(&s.inputs[i]).Font("monospace").Label(b.name)
		if !in.Focused() {
			return
		}
		v, ok := parse(s.texts[i], bits, b.base, b.prefix)
		if !ok {
			in.Error(fmt.Sprintf("Not a %d-bit %s number", bits, b.word))
		}
		if ok && in.Changed() && v != *value {
			*value, changed = v, true
		}
	}).Basis(w + t.Space(6))
	if b.base == 2 {
		field.Grow(1)
	}
	return changed
}

// format writes v of bits bits in base, with prefix, padded with zeros to
// the register's width, and binary in groups of four.
func format(v uint64, bits, base int, prefix string) string {
	if base == 10 {
		return strconv.FormatUint(v, 10)
	}
	digits := bits
	if base == 16 {
		digits = (bits + 3) / 4
	}
	s := strings.ToUpper(strconv.FormatUint(v, base))
	s = strings.Repeat("0", digits-len(s)) + s
	if base != 2 {
		return prefix + s
	}
	var out strings.Builder
	for i := range len(s) {
		if i > 0 && (len(s)-i)%4 == 0 {
			out.WriteByte(' ')
		}
		out.WriteByte(s[i])
	}
	return out.String()
}

// parse reads a number of bits bits in base, with or without prefix,
// ignoring spaces and underscores.
func parse(s string, bits, base int, prefix string) (uint64, bool) {
	s = strings.NewReplacer(" ", "", "_", "").Replace(strings.ToLower(s))
	v, err := strconv.ParseUint(strings.TrimPrefix(s, prefix), base, bits)
	return v, err == nil
}

// grid builds the bits, with the brackets and labels of their fields,
// and changes *value as the pointer and keys say.
func (s *State) grid(c *ui.Context, g *geometry, value *uint64) bool {
	t := c.Theme()
	n := g.bits
	old := *value
	mask := func(k int) uint64 { return 1 << (n - 1 - k) }
	set := func(k int, on bool) {
		if on {
			*value |= mask(k)
		} else {
			*value &^= mask(k)
		}
	}
	box := ui.Box(c).Size(g.w, g.h).Focusable().FocusRing(false).Label("Bits")
	s.focus = max(0, min(n-1, s.focus))
	x, y, over := box.PointerPosition()
	// Dragged goes on while the press does, past the box, where Pressed
	// stops.
	if _, _, held := box.Dragged(); held {
		k, _ := g.at(x, y)
		switch {
		case !s.down:
			s.down, s.last = true, k
			if k >= 0 {
				s.set = *value&mask(k) == 0
				set(k, s.set)
				s.focus = k
			}
		case s.last >= 0 && k >= 0:
			// A quick drag skips bits between frames; those in the row
			// it crossed are set too.
			from, to := min(s.last, k), max(s.last, k)
			if from/g.perRow != to/g.perRow {
				from, to = k, k
			}
			for i := from; i <= to; i++ {
				set(i, s.set)
			}
			s.last, s.focus = k, k
		}
	} else {
		s.down = false
	}
	for _, m := range []struct {
		key ui.Key
		by  int
	}{{ui.KeyLeft, -1}, {ui.KeyRight, 1}, {ui.KeyUp, -g.perRow}, {ui.KeyDown, g.perRow}} {
		if to := s.focus + m.by; box.Shortcut(0, m.key) && to >= 0 && to < n {
			s.focus = to
		}
	}
	if box.Shortcut(0, ui.KeySpace) {
		*value ^= mask(s.focus)
	}
	hoverBit, hovered := -1, -1
	if over {
		hoverBit, hovered = g.at(x, y)
	}
	if hoverBit >= 0 {
		box.Cursor(ui.CursorPointer)
	}
	bit := n - 1 - s.focus
	box.Value(fmt.Sprintf("Bit %d, %d", bit, *value>>bit&1))
	v, focus, focused := *value, s.focus, box.Focused()
	box.Draw(func(p *ui.Painter, r ui.Rect) {
		radius := t.Space(1)
		white := ui.RGB(255, 255, 255)
		for k := range n {
			row, col := k/g.perRow, k%g.perRow
			top := r.Y + g.top[row]
			cx := r.X + g.x[col]
			num := strconv.Itoa(n - 1 - k)
			p.RichText(cx+(g.cellW-float32(len(num))*g.nw)/2, top, 0, small(t, num))
			if col > 0 && (n-k)%8 == 0 {
				lx := (r.X + g.x[col-1] + g.cellW + cx) / 2
				p.Line(lx, top, lx, top+g.numH+g.cellH, 1, t.Border)
			}
			cell := ui.Rect{X: cx, Y: top + g.numH, W: g.cellW, H: g.cellH}
			on := v&mask(k) != 0
			f := g.owner[k]
			digit := ui.Span{Text: "0", Font: "monospace", Size: t.FontSize, Color: t.TextMuted}
			switch {
			case f < 0 && on:
				p.Fill(cell, t.Text.Alpha(0.12), radius)
				digit.Text, digit.Weight = "1", 700
			case f < 0:
				p.StrokeDashed(cell, t.Border, radius, 1)
				digit.Color = t.TextMuted.Alpha(0.55)
			case on:
				fill, _ := colors(t, f)
				p.Fill(cell, fill, radius)
				digit.Text, digit.Weight, digit.Color = "1", 700, white
			default:
				p.Fill(cell, t.Background, radius)
				p.Stroke(cell, t.Border, radius, 1)
			}
			if f >= 0 && f == hovered && !on {
				fill, _ := colors(t, f)
				p.Fill(cell, fill.Alpha(0.2), radius)
			}
			p.RichText(cell.X+(cell.W-g.cw)/2, cell.Y+(cell.H-g.ch)/2, 0, digit)
		}
		for _, sg := range g.segs {
			fill, ink := colors(t, sg.field)
			width := float32(1.5)
			if sg.field == hovered {
				width = 2.5
				o := t.Space(0.5)
				p.Stroke(ui.Rect{X: r.X + sg.x0 - o, Y: r.Y + g.top[sg.row] + g.numH - o, W: sg.x1 - sg.x0 + 2*o, H: g.cellH + 2*o}, fill, radius+o, 2)
			}
			y0 := r.Y + g.top[sg.row] + g.numH + g.cellH + t.Space(1)
			y1 := y0 + t.Space(1.5)
			p.StrokePolyline([][2]float32{{r.X + sg.x0 + 1, y0}, {r.X + sg.x0 + 1, y1}, {r.X + sg.x1 - 1, y1}, {r.X + sg.x1 - 1, y0}}, width, ink)
		}
		for _, l := range g.labels {
			_, ink := colors(t, l.field)
			top := r.Y + g.top[l.row] + g.stripH
			y := top + float32(l.lane)*g.laneH
			if l.side != 0 {
				p.Line(r.X+l.at, top-t.Space(0.5), r.X+l.at, y+g.textH/2, 1, ink)
			}
			p.RichText(r.X+l.x, y, 0, l.spans...)
		}
		if focused {
			row, col := focus/g.perRow, focus%g.perRow
			p.FocusRing(ui.Rect{X: r.X + g.x[col], Y: r.Y + g.top[row] + g.numH, W: g.cellW, H: g.cellH}, [4]float32{radius, radius, radius, radius})
		}
	})
	return *value != old
}

// small is the span of a bit's number.
func small(t *ui.Theme, s string) ui.Span {
	return ui.Span{Text: s, Font: "monospace", Size: t.FontSize - 3, Color: t.TextMuted}
}

// colors returns the color of field i, to fill its set bits with, and
// the darker or lighter one of its bracket and name, which reads on the
// theme's background.
func colors(t *ui.Theme, i int) (fill, ink ui.Color) {
	// The golden angle keeps neighbors apart in hue, however many fields.
	h := float32(math.Mod(25+float64(i)*137.5, 360))
	if t.Dark {
		return ui.Oklch(0.6, 0.14, h), ui.Oklch(0.78, 0.12, h)
	}
	return ui.Oklch(0.6, 0.15, h), ui.Oklch(0.5, 0.15, h)
}

// geometry is where a view draws, in DIPs from its top left: rows of
// cells, counted k from the most significant bit, each row under the
// bits' numbers and over the brackets of their fields and the fields'
// labels.
type geometry struct {
	bits, perRow int
	cellW, cellH float32
	// numH is the height of the numbers, stripH of the numbers, cells and
	// brackets, and laneH of a lane of labels below them.
	numH, stripH, laneH float32
	// cw and ch are the size of a cell's digit, nw the width of a digit
	// of a number, and textH the height of a label.
	cw, ch, nw, textH float32
	x                 []float32 // the left of each column
	top               []float32 // the top of each row
	bottom            []float32 // and its bottom
	owner             []int     // the field of each bit, or -1
	segs              []seg
	labels            []label
	w, h              float32
}

// seg is the part of a field in a row, from x0 to x1.
type seg struct {
	field, row int
	x0, x1     float32
}

// label is a field's name and value in its first row, in a lane below
// its bracket: centered under the bracket when it fits (side 0), or else
// to the left (-1) or right (1) of a line down from the bracket's middle,
// at, from x0 to x1 with the line.
type label struct {
	field, row, lane, side int
	at, x, w, x0, x1       float32
	spans                  []ui.Span
}

// covers reports whether l is over the line at x.
func (l *label) covers(x float32) bool { return x >= l.x0 && x <= l.x1 }

// layout places the bits of a register of bits bits, and the brackets
// and labels of its fields, of value.
func layout(c *ui.Context, bits int, fields []Field, value uint64) *geometry {
	t := c.Theme()
	g := &geometry{bits: bits, perRow: min(bits, 32), cellW: t.Space(8), cellH: t.Space(7), numH: t.Space(4)}
	if g.perRow > 16 {
		g.cellW = t.Space(5)
	}
	g.stripH = g.numH + g.cellH + t.Space(3)
	g.cw, g.ch = c.MeasureText(0, ui.Span{Text: "0", Font: "monospace", Size: t.FontSize})
	g.nw, _ = c.MeasureText(0, small(t, "0"))
	_, g.textH = c.MeasureText(0, ui.Span{Text: "M", Size: t.FontSize - 1})
	g.laneH = g.textH + t.Space(1)
	g.x = make([]float32, g.perRow)
	for col := 1; col < g.perRow; col++ {
		gap := t.Space(0.5)
		// bits-col is the bit left of the gap, in the first row and, 32
		// apart, in the others.
		switch b := bits - col; {
		case b%8 == 0:
			gap = t.Space(3.5)
		case b%4 == 0:
			gap = t.Space(2)
		}
		g.x[col] = g.x[col-1] + g.cellW + gap
	}
	g.w = g.x[g.perRow-1] + g.cellW
	rows := (bits + g.perRow - 1) / g.perRow
	g.owner = slices.Repeat([]int{-1}, bits)
	for i, f := range fields {
		lo, hi := max(0, f.Lsb), min(bits-1, f.Lsb+f.Width-1)
		if hi < lo {
			continue
		}
		for k := bits - 1 - hi; k <= bits-1-lo; k++ {
			if g.owner[k] < 0 {
				g.owner[k] = i
			}
		}
		labeled := false
		for row := range rows {
			c0, c1 := max(bits-1-hi, row*g.perRow), min(bits-1-lo, (row+1)*g.perRow-1)
			if c0 > c1 {
				continue
			}
			sg := seg{i, row, g.x[c0%g.perRow], g.x[c1%g.perRow] + g.cellW}
			g.segs = append(g.segs, sg)
			if !labeled {
				labeled = true
				l := label{field: i, row: row, at: (sg.x0 + sg.x1) / 2, spans: spans(t, i, f, value>>lo&(^uint64(0)>>(63-hi+lo)))}
				l.w, _ = c.MeasureText(0, l.spans...)
				if l.w > sg.x1-sg.x0 {
					l.side = -1
				}
				g.labels = append(g.labels, l)
			}
		}
	}
	slices.SortFunc(g.labels, func(a, b label) int {
		return cmp.Or(cmp.Compare(a.row, b.row), cmp.Compare(a.at, b.at))
	})
	y, next := float32(0), 0
	for row := range rows {
		from := next
		for next < len(g.labels) && g.labels[next].row == row {
			next++
		}
		lanes := place(g.labels[from:next], t.Space(1))
		for _, l := range g.labels[from:next] {
			g.w = max(g.w, l.x1)
		}
		g.top = append(g.top, y)
		g.bottom = append(g.bottom, y+g.stripH+float32(lanes)*g.laneH)
		y = g.bottom[row] + t.Space(4)
	}
	g.h = g.bottom[rows-1]
	return g
}

// spans returns the label of field f, the i-th, of value v.
func spans(t *ui.Theme, i int, f Field, v uint64) []ui.Span {
	_, ink := colors(t, i)
	size := t.FontSize - 1
	out := []ui.Span{
		{Text: f.Name, Size: size, Weight: 600, Color: ink},
		{Text: " = " + strconv.FormatUint(v, 10), Size: size, Color: t.Text, Features: "tnum"},
	}
	if name, ok := f.Values[v]; ok {
		out = append(out, ui.Span{Text: " (" + name + ")", Size: size, Color: t.TextMuted})
	}
	return out
}

// place puts the labels of a row, left to right, in lanes, and returns
// how many: a label too wide for its bracket goes to the left of its
// line, or to the right where the left has no room, and in a lane below
// any it meets and any whose line it is over, so that no label is over
// another or over another's line.
func place(ls []label, pad float32) int {
	turn := func(l *label, side int) {
		l.side = side
		switch side {
		case 0:
			l.x = l.at - l.w/2
			l.x0, l.x1 = l.x, l.x+l.w
		case -1:
			l.x = l.at - pad - l.w
			l.x0, l.x1 = l.x, l.at
		default:
			l.x = l.at + pad
			l.x0, l.x1 = l.at, l.x+l.w
		}
	}
	for i := range ls {
		l := &ls[i]
		if l.side != 0 && l.at-pad-l.w < 0 {
			l.side = 1
		}
		turn(l, l.side)
	}
	// Two labels facing each other, each over the other's line, fit in no
	// lanes: the right one turns right too.
	for j := range ls {
		for i := range j {
			a, b := &ls[i], &ls[j]
			if a.side == 1 && b.side == -1 && a.covers(b.at) && b.covers(a.at) {
				turn(b, 1)
			}
		}
	}
	for range ls {
		moved := false
		for i := range ls {
			for j := range ls {
				a, b := &ls[i], &ls[j]
				// Of two that meet, neither over the other's line, the one
				// beside its line goes below one under its bracket, or else
				// the one to the right does.
				meet := a.x0 < b.x1+pad && b.x0 < a.x1+pad && !a.covers(b.at) && !b.covers(a.at)
				lower := b.side != 0 && (a.side == 0 || i < j)
				if i != j && b.lane <= a.lane && (b.covers(a.at) || meet && lower) {
					b.lane, moved = a.lane+1, true
				}
			}
		}
		if !moved {
			break
		}
	}
	lanes := 0
	for _, l := range ls {
		lanes = max(lanes, l.lane+1)
	}
	return lanes
}

// at returns the bit at x, y, counted from the most significant, or -1,
// and the field there, of the bit or a label, or -1.
func (g *geometry) at(x, y float32) (int, int) {
	for row, top := range g.top {
		switch {
		case y >= top && y < top+g.stripH:
			for col, cx := range g.x {
				if k := row*g.perRow + col; x >= cx && x < cx+g.cellW && k < g.bits {
					return k, g.owner[k]
				}
			}
		case y >= top+g.stripH && y < g.bottom[row]:
			for _, l := range g.labels {
				ly := top + g.stripH + float32(l.lane)*g.laneH
				if l.row == row && x >= l.x && x < l.x+l.w && y >= ly && y < ly+g.laneH {
					return -1, l.field
				}
			}
		}
	}
	return -1, -1
}
