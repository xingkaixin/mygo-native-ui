// Package karnaugh simplifies a Boolean function of two to four inputs,
// as a Karnaugh map is drawn in a logic course: the truth table on the
// left, the map on the right with its rows and columns in Gray code,
// and the simplest sum of products under them. Each product is a loop
// around the cells it covers in its own color, split where it wraps
// around an edge, and its term in the sum is in that color too. A click
// on a cell of the map or a row of the table turns its output from 0 to
// 1 to don't care (X) and back; don't cares join loops when that makes
// them larger. The pointer on a cell lights its row of the table, and
// the other way round. The map changes the app's cells in place and
// reports the changes:
//
//	if karnaugh.View(c, []string{"A", "B", "C", "D"}, &app.cells).Changed() {
//		app.formula = karnaugh.Format(vars, karnaugh.Minimize(len(vars), app.cells))
//	}
package karnaugh

import (
	"math/bits"
	"slices"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
)

// Cell is an output of the function.
type Cell uint8

const (
	Zero Cell = iota
	One
	DontCare
)

func (v Cell) String() string { return [...]string{"0", "1", "X"}[v] }

// Term is a product of inputs: Mask has a bit for each input it names,
// the first input the highest, and Value the bits those inputs must be.
type Term struct {
	Mask, Value uint
}

// Covers reports whether t is true for minterm m.
func (t Term) Covers(m uint) bool { return m&t.Mask == t.Value }

// Minimize returns the fewest, then smallest, products whose sum is the
// function of n inputs with outputs cells, 2ⁿ of them, the first input
// the highest bit of a cell's index: prime implicants, the essential
// ones first and then those covering the most ones left.
func Minimize(n int, cells []Cell) []Term {
	size := uint(1) << n
	valid := func(t Term) bool {
		for m := range size {
			if t.Covers(m) && cells[m] == Zero {
				return false
			}
		}
		return true
	}
	var all []Term
	for mask := range size {
		for value := range size {
			if value&^mask == 0 && valid(Term{mask, value}) {
				all = append(all, Term{mask, value})
			}
		}
	}
	var primes []Term
	for _, t := range all {
		prime := true
		for _, o := range all {
			// o is larger and contains t when it fixes fewer of the inputs,
			// and those as t does.
			if o != t && o.Mask&t.Mask == o.Mask && t.Value&o.Mask == o.Value {
				prime = false
				break
			}
		}
		if prime {
			primes = append(primes, t)
		}
	}
	left := map[uint]bool{}
	for m := range size {
		if cells[m] == One {
			left[m] = true
		}
	}
	var out []Term
	take := func(t Term) {
		out = append(out, t)
		for m := range left {
			if t.Covers(m) {
				delete(left, m)
			}
		}
	}
	for m := range size {
		if cells[m] != One {
			continue
		}
		var cover []Term
		for _, p := range primes {
			if p.Covers(m) {
				cover = append(cover, p)
			}
		}
		if len(cover) == 1 && !slices.Contains(out, cover[0]) {
			take(cover[0])
		}
	}
	for len(left) > 0 {
		best, score := Term{}, -1
		for _, p := range primes {
			n := 0
			for m := range left {
				if p.Covers(m) {
					n++
				}
			}
			// More ones first, then fewer inputs named.
			s := n*16 - bits.OnesCount(p.Mask)
			if n > 0 && s > score {
				best, score = p, s
			}
		}
		take(best)
	}
	slices.SortFunc(out, func(a, b Term) int {
		if d := bits.OnesCount(a.Mask) - bits.OnesCount(b.Mask); d != 0 {
			return d
		}
		if a.Mask != b.Mask {
			return int(b.Mask) - int(a.Mask)
		}
		return int(b.Value) - int(a.Value)
	})
	return out
}

// Format writes terms as a sum of products of vars, a complement with a
// prime, as A'B + CD'.
func Format(vars []string, terms []Term) string {
	if len(terms) == 0 {
		return "0"
	}
	parts := make([]string, len(terms))
	for i, t := range terms {
		parts[i] = product(vars, t)
	}
	return strings.Join(parts, " + ")
}

func product(vars []string, t Term) string {
	n := len(vars)
	var b strings.Builder
	for i, v := range vars {
		bit := uint(1) << (n - 1 - i)
		if t.Mask&bit == 0 {
			continue
		}
		b.WriteString(v)
		if t.Value&bit == 0 {
			b.WriteString("'")
		}
	}
	if b.Len() == 0 {
		return "1"
	}
	return b.String()
}

// gray returns the k-bit Gray code in order: 0 1, or 00 01 11 10.
func gray(k int) []uint {
	if k == 1 {
		return []uint{0, 1}
	}
	return []uint{0, 1, 3, 2}
}

// Color returns the color of the term at index i.
func Color(t *ui.Theme, i int) ui.Color {
	l := float32(0.6)
	if t.Dark {
		l = 0.72
	}
	return ui.Oklch(l, 0.17, 25+float32(i)*70)
}

// View creates the truth table and the map of the function of vars, two
// to four of them, with outputs *cells, which a click changes in place,
// and the sum of products under them. Changed reports a change.
func View(c *ui.Context, vars []string, cells *[]Cell) ui.Element {
	t := c.Theme()
	n := len(vars)
	size := 1 << n
	if len(*cells) != size {
		grown := make([]Cell, size)
		copy(grown, *cells)
		*cells = grown
	}
	terms := Minimize(n, *cells)
	root := ui.Column(c).Gap(t.Space(5))
	hover := ui.Local(root, "karnaugh.hover", func() int { return -1 })
	hot := -1
	changed := false
	cycle := func(m int) {
		(*cells)[m] = ((*cells)[m] + 1) % 3
		changed = true
	}
	root.Children(func() {
		ui.Row(c).Gap(t.Space(8)).AlignItems(ui.Start).Children(func() {
			table(c, vars, *cells, *hover, &hot, cycle)
			grid(c, vars, *cells, terms, *hover, &hot, cycle)
		})
		ui.Row(c).Wrap().Gap(t.Space(2)).Role(ui.RoleText).Label("F = " + Format(vars, terms)).Children(func() {
			math := func(c *ui.Context, s string) ui.Element {
				return ui.Text(c, s).Font("monospace").FontSize(t.FontSize + 4)
			}
			math(c, "F")
			math(c, "=").TextColor(t.TextMuted)
			if len(terms) == 0 {
				math(c, "0")
			}
			for i, tm := range terms {
				if i > 0 {
					math(c.Key(-i), "+").TextColor(t.TextMuted)
				}
				math(c.Key(i), product(vars, tm)).FontWeight(700).TextColor(Color(t, i))
			}
		})
	})
	// The table is built before the map, so a cell's hover reaches the
	// table a frame later.
	if hot != *hover {
		*hover = hot
		c.Invalidate()
	}
	if changed {
		root.SetChanged()
	}
	return root
}

// table creates the truth table, its row m lit when m is hot.
func table(c *ui.Context, vars []string, cells []Cell, lit int, hot *int, cycle func(int)) {
	t := c.Theme()
	n := len(vars)
	cellW := t.Space(8)
	ui.Column(c).Border(1, t.Border).Radius(t.Radius).Clip().Role(ui.RoleTable).Label("Truth table").Children(func() {
		ui.Row(c).Background(t.Surface).Children(func() {
			ui.Text(c, "#").Width(cellW).TextAlign(ui.Center).TextColor(t.TextMuted).Padding(t.Space(1), 0).FontSize(t.FontSize - 1)
			for _, v := range vars {
				ui.Text(c.Key(v), v).Width(cellW).TextAlign(ui.Center).FontWeight(600).Padding(t.Space(1), 0).Font("monospace")
			}
			ui.Text(c, "F").Width(cellW).TextAlign(ui.Center).FontWeight(700).Padding(t.Space(1), 0).Font("monospace")
		})
		for m := range len(cells) {
			row := ui.ButtonBase(c.Key(m)).Row().FocusRing(false).Label("Row " + strconv.Itoa(m) + ", F " + cells[m].String())
			if row.Hovered() {
				*hot = m
			}
			if m == lit {
				row.Background(t.Accent.Alpha(0.14))
			}
			if row.Clicked() {
				cycle(m)
			}
			row.Children(func() {
				ui.Text(c, strconv.Itoa(m)).Width(cellW).TextAlign(ui.Center).TextColor(t.TextMuted).FontSize(t.FontSize-2).FontFeatures("tnum").Padding(t.Space(0.5), 0)
				for i := range n {
					bit := m >> (n - 1 - i) & 1
					ui.Text(c.Key(i), strconv.Itoa(bit)).Width(cellW).TextAlign(ui.Center).Font("monospace").Padding(t.Space(0.5), 0)
				}
				out := ui.Text(c, cells[m].String()).Width(cellW).TextAlign(ui.Center).Font("monospace").Padding(t.Space(0.5), 0)
				if cells[m] == One {
					out.FontWeight(700)
				} else {
					out.TextColor(t.TextMuted)
				}
			})
		}
	})
}

// grid creates the map, its cell m lit when m is lit, with a loop for
// each of terms.
func grid(c *ui.Context, vars []string, cells []Cell, terms []Term, lit int, hot *int, cycle func(int)) {
	t := c.Theme()
	n := len(vars)
	rb := n / 2
	cb := n - rb
	rows, cols := gray(rb), gray(cb)
	cell := t.Space(16)
	head := t.Space(10)
	index := func(r, k int) int { return int(rows[r]<<cb | cols[k]) }
	bitsOf := func(v uint, k int) string {
		s := strconv.FormatUint(uint64(v), 2)
		return strings.Repeat("0", k-len(s)) + s
	}
	box := ui.Box(c).Size(head+cell*float32(len(cols)), head+cell*float32(len(rows))).Role(ui.RoleGroup).Label("Karnaugh map")
	box.Children(func() {
		for r := range rows {
			for k := range cols {
				m := index(r, k)
				e := ui.ButtonBase(c.Key(m)).Absolute().Left(head+cell*float32(k)).Top(head+cell*float32(r)).Size(cell, cell).
					FocusRing(false).Label("Cell " + strconv.Itoa(m) + ", " + cells[m].String())
				if e.Hovered() {
					*hot = m
				}
				if e.Clicked() {
					cycle(m)
				}
			}
		}
	})
	box.Draw(func(p *ui.Painter, b ui.Rect) {
		ox, oy := b.X+head, b.Y+head
		corner := ui.Span{Text: strings.Join(vars[:rb], "") + " \\ " + strings.Join(vars[rb:], ""), Size: t.FontSize - 2, Color: t.TextMuted, Font: "monospace"}
		cw, _ := p.MeasureText(0, corner)
		p.RichText(max(b.X, ox-cw-t.Space(1)), b.Y, 0, corner)
		for k, v := range cols {
			sp := ui.Span{Text: bitsOf(v, cb), Size: t.FontSize - 1, Color: t.TextMuted, Font: "monospace"}
			w, h := p.MeasureText(0, sp)
			p.RichText(ox+cell*float32(k)+(cell-w)/2, oy-h-t.Space(1), 0, sp)
		}
		for r, v := range rows {
			sp := ui.Span{Text: bitsOf(v, rb), Size: t.FontSize - 1, Color: t.TextMuted, Font: "monospace"}
			w, h := p.MeasureText(0, sp)
			p.RichText(ox-w-t.Space(1.5), oy+cell*float32(r)+(cell-h)/2, 0, sp)
		}
		for r := range rows {
			for k := range cols {
				m := index(r, k)
				cr := ui.Rect{X: ox + cell*float32(k), Y: oy + cell*float32(r), W: cell, H: cell}
				if m == lit {
					p.Fill(cr, t.Accent.Alpha(0.12), 0)
				}
				p.Stroke(cr, t.Border, 0, 1)
				sp := ui.Span{Text: cells[m].String(), Size: t.FontSize + 3, Font: "monospace", Weight: 700, Color: t.Text}
				switch cells[m] {
				case Zero:
					sp.Color = t.TextMuted.Alpha(0.6)
				case DontCare:
					sp.Color = t.TextMuted
				}
				w, h := p.MeasureText(0, sp)
				p.RichText(cr.X+(cr.W-w)/2, cr.Y+(cr.H-h)/2, 0, sp)
			}
		}
		for i, tm := range terms {
			col := Color(t, i)
			inset := t.Space(1) + float32(i%3)*t.Space(1.25)
			for _, rr := range runs(rows, rb, cb, tm) {
				for _, kk := range runs(cols, cb, 0, tm) {
					r := ui.Rect{
						X: ox + cell*float32(kk[0]) + inset, Y: oy + cell*float32(rr[0]) + inset,
						W: cell*float32(kk[1]-kk[0]+1) - 2*inset, H: cell*float32(rr[1]-rr[0]+1) - 2*inset,
					}
					p.Fill(r, col.Alpha(0.1), t.Space(3))
					p.Stroke(r, col, t.Space(3), 2)
				}
			}
		}
	})
}

// runs returns the runs of positions along an axis of codes, k bits
// shifted by shift into a minterm, that tm covers, as first and last
// positions; a run wrapping past the last position is split in two.
func runs(codes []uint, k, shift int, tm Term) [][2]int {
	axis := uint(1)<<k - 1
	mask, value := tm.Mask>>shift&axis, tm.Value>>shift&axis
	var out [][2]int
	for i := 0; i < len(codes); i++ {
		if codes[i]&mask != value {
			continue
		}
		j := i
		for j+1 < len(codes) && codes[j+1]&mask == value {
			j++
		}
		out = append(out, [2]int{i, j})
		i = j
	}
	return out
}
