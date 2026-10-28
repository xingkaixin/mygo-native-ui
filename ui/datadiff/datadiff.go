// Package datadiff shows how two versions of a table differ, as data diff
// tools show a table's rows before and after a change. Rows are matched
// by their key columns, and the table shows them in the new order, the
// removed ones where they stood: an added row green with a +, a removed
// row red and struck through with a −, and in a changed row each changed
// cell as its old value struck through beside its new one, numbers with
// how much they moved. Columns added and removed are marked in the header,
// and the key columns with a key.
//
// Above the table, how many rows were added, removed and changed, and how
// many cells. Runs of rows with no change fold to a row saying how many
// there are, which a click opens, and Show all opens them all. J and K, or
// ↓ and ↑, go to the next and previous changed row.
//
//	app.diff = datadiff.Compare(before, after, "id")
//	...
//	datadiff.View(c, &app.state, app.diff).Grow(1)
package datadiff

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
)

// Table is a table: its columns' names and its rows, a value for each
// column.
type Table struct {
	Columns []string
	Rows    [][]string
}

// Kind is how a row or a column changed.
type Kind uint8

const (
	Same Kind = iota
	Added
	Removed
	Changed
)

type column struct {
	name string
	kind Kind
	key  bool
	old  int // index in the old table, or -1
	new  int // index in the new table, or -1
}

type row struct {
	kind     Kind
	old, new []string
}

// Diff is how two tables differ: how many rows were added, removed and
// changed, and how many cells changed in the rows in both.
type Diff struct {
	Added, Removed, Changed, Cells int

	columns []column
	rows    []row
}

// Compare returns how after differs from before, its rows matched by the
// values in the key columns, the first column without any.
func Compare(before, after Table, key ...string) Diff {
	if len(key) == 0 && len(after.Columns) > 0 {
		key = after.Columns[:1]
	}
	var d Diff
	for j, name := range after.Columns {
		col := column{name: name, kind: Added, key: slices.Contains(key, name), old: -1, new: j}
		if i := slices.Index(before.Columns, name); i >= 0 {
			col.kind, col.old = Same, i
		}
		d.columns = append(d.columns, col)
	}
	// A removed column goes after the column that came before it.
	for i, name := range before.Columns {
		if slices.Contains(after.Columns, name) {
			continue
		}
		at := 0
		if i > 0 {
			at = slices.IndexFunc(d.columns, func(c column) bool { return c.old == i-1 }) + 1
		}
		d.columns = slices.Insert(d.columns, at, column{name: name, kind: Removed, old: i, new: -1})
	}
	keyOf := func(cols []string, r []string) string {
		var parts []string
		for _, k := range key {
			if i := slices.Index(cols, k); i >= 0 && i < len(r) {
				parts = append(parts, r[i])
			}
		}
		return strings.Join(parts, "\x00")
	}
	olds := map[string]int{}
	for i, r := range before.Rows {
		olds[keyOf(before.Columns, r)] = i
	}
	matched := make([]bool, len(before.Rows))
	news := map[string]bool{}
	for _, r := range after.Rows {
		news[keyOf(after.Columns, r)] = true
	}
	next := 0
	removed := func(upto int) {
		for ; next < upto; next++ {
			if !news[keyOf(before.Columns, before.Rows[next])] {
				d.rows = append(d.rows, row{kind: Removed, old: before.Rows[next]})
				d.Removed++
			}
		}
	}
	for _, r := range after.Rows {
		i, ok := olds[keyOf(after.Columns, r)]
		if !ok || matched[i] {
			d.rows = append(d.rows, row{kind: Added, new: r})
			d.Added++
			continue
		}
		matched[i] = true
		removed(i)
		next = max(next, i+1)
		rw := row{kind: Same, old: before.Rows[i], new: r}
		for _, col := range d.columns {
			if col.kind == Same && cell(rw.old, col.old) != cell(r, col.new) {
				rw.kind = Changed
				d.Cells++
			}
		}
		if rw.kind == Changed {
			d.Changed++
		}
		d.rows = append(d.rows, rw)
	}
	removed(len(before.Rows))
	return d
}

func cell(r []string, i int) string {
	if i < 0 || i >= len(r) {
		return ""
	}
	return r[i]
}

// State is what a diff view keeps: the folds opened and the change gone
// to. Keep it in the app's state, one for each view.
type State struct {
	all    bool
	open   map[int]bool
	at     int // the row gone to, plus 1
	jumped bool
}

const fold = 3

// item is a row of the table, or when count > 0 a fold of count rows
// from row, in the run of unchanged rows from start.
type item struct {
	row, count, start int
}

var (
	addColor    = ui.Hex("#2F9E44")
	removeColor = ui.Hex("#E03131")
	changeColor = ui.Hex("#F08C00")
)

// View creates the view of d.
func View(c *ui.Context, s *State, d Diff) ui.Element {
	t := c.Theme()
	if s.at > len(d.rows) {
		s.at = 0
	}
	var items []item
	for i := 0; i < len(d.rows); {
		j := i
		for j < len(d.rows) && d.rows[j].kind == Same {
			j++
		}
		if j == i {
			items = append(items, item{row: i})
			i++
			continue
		}
		lead, tail := 1, 1
		if i == 0 {
			lead = 0
		}
		if j == len(d.rows) {
			tail = 0
		}
		if s.all || s.open[i] || j-i-lead-tail < fold {
			for k := i; k < j; k++ {
				items = append(items, item{row: k})
			}
		} else {
			for k := i; k < i+lead; k++ {
				items = append(items, item{row: k})
			}
			items = append(items, item{row: i + lead, count: j - i - lead - tail, start: i})
			for k := j - tail; k < j; k++ {
				items = append(items, item{row: k})
			}
		}
		i = j
	}
	jump := func(dir int) {
		for k := s.at - 1 + dir; k >= 0 && k < len(d.rows); k += dir {
			if d.rows[k].kind != Same {
				s.at, s.jumped = k+1, true
				return
			}
		}
	}
	root := ui.Column(c).Gap(t.Space(3)).Focusable().FocusRing(false).Label("Table diff")
	switch {
	case root.Shortcut(0, ui.KeyJ), root.Shortcut(0, ui.KeyDown):
		jump(1)
	case root.Shortcut(0, ui.KeyK), root.Shortcut(0, ui.KeyUp):
		if s.at == 0 {
			s.at = len(d.rows) + 1
		}
		jump(-1)
	}
	widths := measure(c, d)
	root.Children(func() {
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			count := func(n int, sign, what string, col ui.Color) {
				e := ui.Text(c, fmt.Sprintf("%s%d %s", sign, n, what)).FontFeatures("tnum")
				if n > 0 {
					e.TextColor(col).Bold()
				} else {
					e.TextColor(t.TextMuted)
				}
			}
			count(d.Added, "+", "added", addColor)
			count(d.Removed, "−", "removed", removeColor)
			count(d.Changed, "~", "changed", changeColor)
			ui.Text(c, fmt.Sprintf("%d cells", d.Cells)).TextColor(t.TextMuted).FontFeatures("tnum")
			ui.Box(c).Grow(1)
			ui.Checkbox(c, &s.all, "Show all rows")
		})
		ui.ScrollBoth(c).Grow(1).Radius(t.Space(1.5)).Border(1, t.Border).Children(func() {
			ui.Column(c).Children(func() {
				header(c, d, widths)
				for _, it := range items {
					if it.count > 0 {
						f := ui.ButtonBase(c.Key(fmt.Sprint("fold", it.row))).Row().Height(t.Space(7)).AlignItems(ui.Center).
							PaddingStart(t.Space(9)).Background(t.Text.Alpha(0.03)).Label(fmt.Sprintf("%d unchanged rows", it.count))
						if f.Hovered() {
							f.Background(t.Text.Alpha(0.07))
						}
						if f.Clicked() {
							if s.open == nil {
								s.open = map[int]bool{}
							}
							s.open[it.start] = true
						}
						f.Children(func() {
							ui.Text(c, fmt.Sprintf("⋯ %d unchanged rows", it.count)).TextColor(t.TextMuted).FontSize(t.FontSize - 1)
						})
						continue
					}
					e := line(c.Key(it.row), d, d.rows[it.row], widths, s.at == it.row+1)
					if s.jumped && s.at == it.row+1 {
						e.ScrollIntoView()
					}
				}
			})
		})
	})
	if s.jumped {
		s.jumped = false
		if s.at > 0 && d.rows[s.at-1].kind != Same {
			c.Announce(describe(d, d.rows[s.at-1]))
		}
	}
	return root
}

// measure returns how wide each column is: its widest value, or its name,
// an old value beside a new one where they differ, within bounds.
func measure(c *ui.Context, d Diff) []float32 {
	t := c.Theme()
	size := t.FontSize - 1
	text := func(s string) float32 {
		w, _ := c.MeasureText(0, ui.Span{Text: s, Size: size})
		return w
	}
	widths := make([]float32, len(d.columns))
	for k, col := range d.columns {
		w := text(col.name) + t.Space(6)
		for _, r := range d.rows {
			o, n := cell(r.old, col.old), cell(r.new, col.new)
			v := text(o)
			if r.kind != Removed {
				v = text(n)
			}
			if r.kind == Changed && col.kind == Same && o != n {
				v = text(o) + text(n) + t.Space(3) + text(delta(o, n))
			}
			w = max(w, v+t.Space(5))
		}
		widths[k] = min(t.Space(90), max(t.Space(16), w))
	}
	return widths
}

func header(c *ui.Context, d Diff, widths []float32) {
	t := c.Theme()
	ui.Row(c).Height(t.Space(8)).AlignItems(ui.Center).Background(t.Surface).Children(func() {
		ui.Box(c).Width(t.Space(7))
		for k, col := range d.columns {
			cl := ui.Row(c.Key(k)).Width(widths[k]).PaddingX(t.Space(2)).Gap(t.Space(1)).AlignItems(ui.Center).Shrink(0)
			cl.Children(func() {
				name := ui.Text(c, col.name).Bold().FontSize(t.FontSize - 1).Ellipsis("…")
				switch col.kind {
				case Added:
					ui.Text(c, "+").Bold().TextColor(addColor)
					name.TextColor(addColor)
				case Removed:
					ui.Text(c, "−").Bold().TextColor(removeColor)
					name.TextColor(removeColor).Strikethrough()
				}
				if col.key {
					ui.Text(c, "key").FontSize(t.FontSize-3).TextColor(t.TextMuted).Padding(0, t.Space(1)).
						Radius(t.Space(1)).Background(t.Text.Alpha(0.07))
				}
			})
		}
	})
	ui.Divider(c)
}

// line creates a row of the table.
func line(c *ui.Context, d Diff, r row, widths []float32, current bool) ui.Element {
	t := c.Theme()
	mark, col := "", t.TextMuted
	var bg ui.Color
	switch r.kind {
	case Added:
		mark, col, bg = "+", addColor, addColor.Alpha(0.09)
	case Removed:
		mark, col, bg = "−", removeColor, removeColor.Alpha(0.08)
	case Changed:
		mark, col, bg = "~", changeColor, changeColor.Alpha(0.05)
	}
	e := ui.Row(c).Height(t.Space(7)).AlignItems(ui.Center).Label(describe(d, r))
	if r.kind != Same {
		e.Background(bg)
	}
	if current {
		e.Background(t.Selection)
	}
	size := t.FontSize - 1
	e.Children(func() {
		ui.Text(c, mark).Width(t.Space(7)).TextAlign(ui.Center).Bold().TextColor(col)
		for k, cl := range d.columns {
			o, n := cell(r.old, cl.old), cell(r.new, cl.new)
			box := ui.Box(c.Key(k)).Width(widths[k]).PaddingX(t.Space(2)).Shrink(0).Clip()
			box.Children(func() {
				switch {
				case r.kind == Removed || cl.kind == Removed:
					v := o
					if r.kind == Added {
						v = ""
					}
					ui.Text(c, v).FontSize(size).TextColor(removeColor.Alpha(0.85)).Strikethrough().NoWrap()
				case r.kind == Changed && cl.kind == Same && o != n:
					ui.RichText(c,
						ui.Span{Text: o, Size: size, Color: removeColor, Strikethrough: true},
						ui.Span{Text: "  " + n, Size: size, Weight: 600, Color: t.Text},
						ui.Span{Text: delta(o, n), Size: size - 1, Color: t.TextMuted},
					).NoWrap()
				default:
					v := ui.Text(c, n).FontSize(size).NoWrap()
					if cl.kind == Added {
						v.TextColor(addColor)
					}
				}
			})
		}
	})
	return e
}

// delta returns how much a number moved, as " +2.5", or nothing for
// values that are not both numbers.
func delta(o, n string) string {
	a, err1 := strconv.ParseFloat(strings.ReplaceAll(o, ",", ""), 64)
	b, err2 := strconv.ParseFloat(strings.ReplaceAll(n, ",", ""), 64)
	if err1 != nil || err2 != nil || a == b {
		return ""
	}
	return fmt.Sprintf("  %+g", b-a)
}

// describe says how a row changed, for screen readers.
func describe(d Diff, r row) string {
	var key []string
	vals := r.new
	if r.kind == Removed {
		vals = r.old
	}
	for _, col := range d.columns {
		if col.key {
			i := col.new
			if r.kind == Removed {
				i = col.old
			}
			key = append(key, cell(vals, i))
		}
	}
	name := strings.Join(key, ", ")
	switch r.kind {
	case Added:
		return "Added " + name
	case Removed:
		return "Removed " + name
	case Changed:
		var parts []string
		for _, col := range d.columns {
			if o, n := cell(r.old, col.old), cell(r.new, col.new); col.kind == Same && o != n {
				parts = append(parts, fmt.Sprintf("%s from %s to %s", col.name, o, n))
			}
		}
		return "Changed " + name + ": " + strings.Join(parts, "; ")
	}
	return name
}
