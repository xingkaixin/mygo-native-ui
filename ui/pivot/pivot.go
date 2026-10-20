// Package pivot is a pivot table over records, as spreadsheet apps have
// one: the records grouped into rows by the fields in Rows and into
// columns by those in Columns, the first outermost, and each cell summing
// up the records in its row and its column by each field in Values, as
// their sum, count, average, least or greatest. A group with groups in it
// shows its subtotal, and a click on its name folds and unfolds it; Total
// sums up every record, at the bottom and at the right. Numbers are
// written in the user's regional format, right-aligned, and the headers
// stay as the table scrolls.
//
// Beside the table, the fields are dragged from the list to Rows, Columns
// or Values, between them and within them to reorder them, and back to the
// list to take them out. A click on a field in the list adds it, to Values
// when its values are numbers and to Rows when not, or takes it out of
// them all. The aggregation by a value's name opens a menu of the others.
//
// A click on the header above the row names sorts the rows by their names,
// and a click on a column's header sorts them by its values, largest
// first; a second click reverses the order.
//
//	app.sales.Rows = []string{"Region", "Country"}
//	app.sales.Columns = []string{"Quarter"}
//	app.sales.Values = []pivot.Value{{Field: "Revenue", Agg: pivot.Sum}}
//	pivot.View(c, &app.sales, fields, records).Grow(1)
package pivot

import (
	"math"
	"slices"
	"strconv"

	"github.com/egoist/mygo/ui"
)

// Agg is how a cell sums up the values of a field in its records.
type Agg uint8

const (
	Sum Agg = iota
	// Count counts the records with a value.
	Count
	Average
	Min
	Max
)

var aggNames = [...]string{"Sum", "Count", "Average", "Min", "Max"}

// String returns the name of a, as "Sum".
func (a Agg) String() string { return aggNames[a] }

// Value is a field in Values, which the cells sum up as Agg says.
type Value struct {
	Field string
	Agg   Agg
}

// State is what a table keeps: the fields grouping its rows and its
// columns, outermost first, and those its cells sum up, which the app sets
// to start with a layout; the groups folded, how the rows are sorted and
// where the table is scrolled. Keep it in the app's state, one for each
// table.
type State struct {
	Rows, Columns []string
	Values        []Value
	folded        map[string]bool
	order         order
	scroll        ui.ScrollState
}

// Where a field is: the list of fields, or a well.
const (
	inList = iota
	inRows
	inColumns
	inValues
)

var wellNames = [...]string{"Fields", "Rows", "Columns", "Values"}

// ref is a field dragged: its table, so that tables take only their own
// fields, where it is, its index there, and its name.
type ref struct {
	s     *State
	from  int
	i     int
	field string
}

// kind is what the values of a field are: numbers or not, and whole
// numbers or not.
type kind struct{ number, whole bool }

// kinds returns the kind of each field of records.
func kinds(fields []string, records [][]string) []kind {
	ks := make([]kind, len(fields))
	for f := range fields {
		k, seen := kind{number: true, whole: true}, false
		for _, r := range records {
			if f >= len(r) || r[f] == "" {
				continue
			}
			x, err := strconv.ParseFloat(r[f], 64)
			if err != nil {
				k = kind{}
				break
			}
			seen, k.whole = true, k.whole && x == math.Trunc(x)
		}
		k.number = k.number && seen
		ks[f] = k
	}
	return ks
}

// View creates the list of fields and the wells beside the table of
// records, each record a value for each of fields, in their order; a field
// whose values all read as numbers is a number field. Give it a size, or
// Grow it within its parent.
func View(c *ui.Context, s *State, fields []string, records [][]string) ui.Element {
	t := c.Theme()
	if s.folded == nil {
		s.folded = map[string]bool{}
	}
	ks := kinds(fields, records)
	return ui.Row(c).Gap(t.Space(4)).AlignItems(ui.Stretch).Children(func() {
		s.panel(c, fields, ks)
		if len(s.Values) == 0 {
			ui.Text(c, "Drag a field to Values to sum it up").TextColor(t.TextMuted).Grow(1).AlignSelf(ui.Center).TextAlign(ui.Center)
			return
		}
		s.table(c, fields, records, ks)
	})
}

// panel shows the list of fields and the wells, and moves the field
// dropped on them.
func (s *State) panel(c *ui.Context, fields []string, ks []kind) {
	t := c.Theme()
	var drop *ref
	to, at := 0, 0
	ui.Scroll(c).Width(t.Space(56)).Shrink(0).Children(func() {
		ui.Column(c).Gap(t.Space(1.5)).Children(func() {
			title(c, wellNames[inList])
			list := ui.Column(c).Padding(t.Space(1)).Radius(t.Radius).Border(1, t.Border).Background(t.Surface)
			if r, ok := ui.Drop[ref](list); ok && r.s == s {
				drop, to = &r, inList
			}
			if r, over := ui.DragOver[ref](list); over && r.s == s && r.from != inList {
				list.Border(1, t.Accent)
			}
			list.Children(func() {
				for i, f := range fields {
					s.field(c.Key(i), f, ks[i].number)
				}
			})
			for w := inRows; w <= inValues; w++ {
				ui.Box(c).Height(t.Space(1.5))
				title(c, wellNames[w])
				if r, i, ok := s.well(c.Key(w), w, fields, ks); ok {
					drop, to, at = &r, w, i
				}
			}
		})
	})
	if drop != nil {
		agg := Count
		if f := slices.Index(fields, drop.field); f >= 0 && ks[f].number {
			agg = Sum
		}
		s.place(*drop, to, at, agg)
	}
}

func title(c *ui.Context, s string) {
	t := c.Theme()
	ui.Text(c, s).FontSize(t.FontSize - 1).FontWeight(600).TextColor(t.TextMuted)
}

// field shows a field of the list, checked while a well has it, which
// drags and a click adds or takes out.
func (s *State) field(c *ui.Context, name string, number bool) {
	t := c.Theme()
	used := s.uses(name)
	b := ui.ButtonBase(c).Gap(t.Space(2)).Padding(t.Space(1.5), t.Space(2)).Radius(t.Radius).
		Role(ui.RoleCheckBox).Checked(used).Label(name).Drag(ref{s: s, from: inList, field: name})
	if b.Hovered() {
		b.Background(t.SurfaceHover)
	}
	if b.Clicked() {
		s.toggle(name, number)
	}
	b.Children(func() {
		size := t.Space(3.5)
		ui.Box(c).Size(size, size).Shrink(0).Draw(func(p *ui.Painter, r ui.Rect) {
			if !used {
				p.Fill(r, t.Background, 3)
				p.Stroke(r, t.Border, 3, 1)
				return
			}
			p.Fill(r, t.Accent, 3)
			p.StrokePolyline([][2]float32{{r.X + r.W*0.25, r.Y + r.H*0.52}, {r.X + r.W*0.43, r.Y + r.H*0.7}, {r.X + r.W*0.76, r.Y + r.H*0.32}}, 1.6, t.AccentText)
		})
		ui.Text(c, name).SingleLine().Grow(1).Shrink(1)
		kind := "Abc"
		if number {
			kind = "123"
		}
		ui.Text(c, kind).FontSize(t.FontSize - 3).TextColor(t.TextMuted)
	})
}

// well shows the fields in well w, each dragged to move it, and returns
// the field dropped on it and where: before or after the field under the
// pointer, or at the end.
func (s *State) well(c *ui.Context, w int, fields []string, ks []kind) (dropped ref, at int, ok bool) {
	t := c.Theme()
	names := s.Rows
	switch w {
	case inColumns:
		names = s.Columns
	case inValues:
		names = make([]string, len(s.Values))
		for i, v := range s.Values {
			names[i] = v.Field
		}
	}
	box := ui.Column(c).MinHeight(t.Space(11)).Padding(t.Space(1)).Radius(t.Radius).Border(1, t.Border).BorderStyle(ui.BorderDashed)
	if r, yes := ui.Drop[ref](box); yes && r.s == s {
		dropped, at, ok = r, len(names), true
	}
	r, over := ui.DragOver[ref](box)
	over = over && r.s == s
	box.Children(func() {
		if len(names) == 0 {
			ui.Text(c, "Drag fields here").FontSize(t.FontSize-1).TextColor(t.TextMuted).Padding(t.Space(1.5), t.Space(2))
		}
		for i, name := range names {
			number := false
			if f := slices.Index(fields, name); f >= 0 {
				number = ks[f].number
			}
			r, before, hovered := s.chip(c.Key(i), w, i, name, number, over && i == len(names)-1)
			over = over || hovered
			if r != nil {
				dropped, at, ok = *r, i, true
				if !before {
					at++
				}
			}
		}
	})
	if over {
		box.Border(1, t.Accent).BorderStyle(ui.BorderSolid).Background(t.Accent.Alpha(0.06))
	}
	return dropped, at, ok
}

// chip shows field name at index i of well w, with its aggregation in
// Values: a box around it that drags it and takes fields dropped on either
// half of it, before or after it, shown with a line, below it with end.
// It returns the field dropped on it, whether before, and whether one is
// dragged over it.
func (s *State) chip(c *ui.Context, w, i int, name string, number, end bool) (dropped *ref, before, over bool) {
	t := c.Theme()
	slot := ui.Column(c).Padding(t.Space(0.5), 0).Drag(ref{s: s, from: w, i: i, field: name})
	_, py, _ := slot.PointerPosition()
	before = py < slot.Bounds().H/2
	if r, ok := ui.Drop[ref](slot); ok && r.s == s {
		dropped = &r
	}
	r, over := ui.DragOver[ref](slot)
	over = over && r.s == s
	slot.Draw(func(p *ui.Painter, b ui.Rect) {
		switch {
		case over && before:
			p.Fill(ui.Rect{X: b.X, Y: b.Y - 1, W: b.W, H: 2}, t.Accent, 1)
		case over || end:
			p.Fill(ui.Rect{X: b.X, Y: b.Y + b.H - 1, W: b.W, H: 2}, t.Accent, 1)
		}
	})
	slot.Children(func() {
		face := ui.Row(c).Gap(t.Space(1.5)).Padding(t.Space(1.5), t.Space(2)).Radius(t.Radius).AlignItems(ui.Center).
			Background(t.Background).Border(1, t.Border).Shadow(0, 1, 2, 0, ui.RGB(0, 0, 0).Alpha(0.06))
		if slot.Dragging() {
			face.Opacity(0.4)
		}
		face.Children(func() {
			if w == inValues {
				s.aggMenu(c, i, name, number)
			}
			ui.Text(c, name).SingleLine().Shrink(1)
		})
	})
	return dropped, before, over
}

// aggMenu shows the aggregation of value i, a button opening a menu of
// them; those other than Count take numbers.
func (s *State) aggMenu(c *ui.Context, i int, name string, number bool) {
	t := c.Theme()
	b := ui.ButtonBase(c).Gap(t.Space(1)).Padding(t.Space(0.5), t.Space(1.5)).Radius(t.Radius).Background(t.Surface).
		Label(s.Values[i].Agg.String() + " of " + name)
	if b.Hovered() {
		b.Background(t.SurfaceHover)
	}
	b.Menu(func(m *ui.Menu) {
		if i >= len(s.Values) {
			return
		}
		for a := Sum; a <= Max; a++ {
			if m.Item(a.String()).Checked(s.Values[i].Agg == a).Disabled(!number && a != Count).Chosen() {
				s.Values[i].Agg = a
			}
		}
	})
	b.Children(func() {
		ui.Text(c, s.Values[i].Agg.String()).FontSize(t.FontSize - 1).TextColor(t.Accent)
		ui.Box(c).Size(t.Space(2), t.Space(2)).Draw(func(p *ui.Painter, r ui.Rect) {
			cx, cy, d := r.X+r.W/2, r.Y+r.H/2, r.W*0.3
			p.StrokePolyline([][2]float32{{cx - d, cy - d/2}, {cx, cy + d/2}, {cx + d, cy - d/2}}, 1.4, t.Accent)
		})
	})
}

// uses reports whether a well has field.
func (s *State) uses(field string) bool {
	return slices.Contains(s.Rows, field) || slices.Contains(s.Columns, field) ||
		slices.ContainsFunc(s.Values, func(v Value) bool { return v.Field == field })
}

// toggle takes field out of the wells, or adds it to Values when its
// values are numbers and to Rows when not.
func (s *State) toggle(field string, number bool) {
	switch {
	case s.uses(field):
		same := func(f string) bool { return f == field }
		s.Rows = slices.DeleteFunc(s.Rows, same)
		s.Columns = slices.DeleteFunc(s.Columns, same)
		s.Values = slices.DeleteFunc(s.Values, func(v Value) bool { return v.Field == field })
	case number:
		s.Values = append(s.Values, Value{Field: field})
	default:
		s.Rows = append(s.Rows, field)
	}
}

// place moves the field r drags to index at of well to, counted before the
// move, a new value summing it up as agg; a field groups either rows or
// columns, not both. Dropped on the list, it is taken out of its well.
func (s *State) place(r ref, to, at int, agg Agg) {
	dims := func(w int) *[]string {
		if w == inRows {
			return &s.Rows
		}
		return &s.Columns
	}
	switch r.from {
	case inRows, inColumns:
		fs := dims(r.from)
		*fs = slices.Delete(*fs, r.i, r.i+1)
		if to == r.from && at > r.i {
			at--
		}
	case inValues:
		agg = s.Values[r.i].Agg
		s.Values = slices.Delete(s.Values, r.i, r.i+1)
		if to == inValues && at > r.i {
			at--
		}
	}
	switch to {
	case inRows, inColumns:
		other := dims(inRows + inColumns - to)
		*other = slices.DeleteFunc(*other, func(f string) bool { return f == r.field })
		fs := dims(to)
		if i := slices.Index(*fs, r.field); i >= 0 {
			*fs = slices.Delete(*fs, i, i+1)
			if at > i {
				at--
			}
		}
		*fs = slices.Insert(*fs, min(at, len(*fs)), r.field)
	case inValues:
		s.Values = slices.Insert(s.Values, min(at, len(s.Values)), Value{Field: r.field, Agg: agg})
	}
}
