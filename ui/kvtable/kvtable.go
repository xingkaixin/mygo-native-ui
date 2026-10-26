// Package kvtable edits pairs of keys and values as a table, as Postman
// edits the parameters and headers of a request and as apps edit
// environment variables: a row for each pair, with a check box that turns
// it on and off, its key and its value in fields, and a button that
// removes it; and a last row, empty, which typing in adds a pair from.
//
//	if kvtable.Editor(c, &app.headers).Changed() {
//		app.send()
//	}
//	for _, h := range app.headers {
//		if h.On {
//			req.Header.Add(h.Key, h.Value)
//		}
//	}
//
// EditorWith takes Options for more: a kind for each pair, as a form's
// fields are text or files, suggested keys and values, rows dragged into
// another order, and the pairs edited as text.
//
//	kvtable.EditorWith(c, &app.form, kvtable.Options{
//		Kinds:  []kvtable.Kind{{Name: "Text"}, {Name: "File", Cell: kvtable.File}},
//		Choose: app.chooseFile,
//	})
package kvtable

import (
	"cmp"
	"encoding/json"
	"path/filepath"
	"slices"
	"strings"

	"github.com/egoist/mygo/ui"
)

// Pair is a key and its value, which counts while On, of the kind named
// Kind in a table with Options.Kinds.
type Pair struct {
	Key, Value string
	On         bool
	Kind       string
}

// Cell is how a kind's values show and are edited.
type Cell uint8

const (
	// Text is a field of text.
	Text Cell = iota
	// File is the path of a file, which shows by its name: a file dropped
	// on it takes its place, as one Options.Choose picks does.
	File
	// Secret is a field showing dots, until its Show button shows the text.
	Secret
	// JSON is a field in a monospace font, in the danger color while what
	// it holds is not JSON.
	JSON
)

// Kind is a kind of value: its name, which a pair's Kind holds and the
// kind column shows, and how its values show.
type Kind struct {
	Name string
	Cell Cell
}

// Options are what a table does besides Editor's.
type Options struct {
	// KeyLabel and ValueLabel name the columns, in the header and as the
	// placeholders of the fields: Key and Value when empty.
	KeyLabel, ValueLabel string
	// Kinds, when any, show a column choosing each pair's kind among them.
	// A pair added is of the first.
	Kinds []Kind
	// Keys are suggested below a key as it is typed: those containing what
	// was typed, those starting with it first. Up and Down move among them,
	// and Enter or a click takes one.
	Keys []string
	// Values returns the values to suggest for key, as Keys are.
	Values func(key string) []string
	// Ranges styles runs of a value in a field, as templatefield.Ranges
	// colors its variables.
	Ranges func(value string) []ui.TextRange
	// Choose, when set, puts a Choose button in File values, which calls
	// it: it shows a dialog, and calls set with the path picked on the main
	// thread, as Window.Update runs it, or not at all when canceled.
	Choose func(set func(path string))
	// Reorder puts a handle at the start of each row, which drags it to
	// another place.
	Reorder bool
	// Bulk, when set, shows the pairs as text while *Bulk is true: a line
	// for each, "key: value", starting "# " while off. Editing the text
	// edits the pairs, which keep their kinds by their keys.
	Bulk *bool
}

// Editor creates a table editing *pairs, under a header naming its
// columns: a row for each pair, which its check box turns on and off and
// its button, shown as the pointer is over the row, removes, and a last
// row adding the pair typed in it, on. Tab goes from field to field.
// Changed reports any edit.
func Editor(c *ui.Context, pairs *[]Pair) ui.Element {
	return EditorWith(c, pairs, Options{})
}

// EditorWith creates the table of Editor, which o adds to. Changed reports
// any edit: of a key, a value or a kind, a file dropped or chosen, a row
// moved, or the text while Bulk.
func EditorWith(c *ui.Context, pairs *[]Pair, o Options) ui.Element {
	if o.Bulk != nil && *o.Bulk {
		return bulk(c, pairs, o)
	}
	t := c.Theme()
	tb := &table{c: c, o: o, keyLabel: cmp.Or(o.KeyLabel, "Key"), valueLabel: cmp.Or(o.ValueLabel, "Value")}
	el := ui.Column(c).Border(1, t.Border).Radius(t.Radius).Clip()
	tb.st = ui.Local(el, "kvtable.state", func() state { return state{picked: -1} })
	draft := &tb.st.draft
	draft.On = true
	if draft.Kind == "" && len(o.Kinds) > 0 {
		draft.Kind = o.Kinds[0].Name
	}
	changed, picked := false, false
	if i := tb.st.picked; i >= 0 && i < len(*pairs) {
		(*pairs)[i].Value = tb.st.path
		changed = true
	} else if i == len(*pairs) {
		draft.Value, picked = tb.st.path, true
	}
	tb.st.picked = -1
	remove, move := -1, [2]int{-1, -1}
	el.Children(func() {
		ui.Row(c).Background(t.Surface).Children(func() {
			if o.Reorder {
				ui.Box(c).Width(t.Space(5))
			}
			cell(c, "", false).Width(t.Space(9))
			if len(o.Kinds) > 0 {
				cell(c, "", false).Width(t.Space(26))
			}
			cell(c, tb.keyLabel, true).Grow(1).Basis(0)
			ui.Divider(c)
			cell(c, tb.valueLabel, true).Grow(1.5).Basis(0)
			ui.Box(c).Width(t.Space(8))
		})
		for i := range *pairs {
			ui.Divider(c)
			e := tb.row(c.Key(i), &(*pairs)[i], i, false)
			changed = changed || e.changed
			if e.remove {
				remove = i
			}
			if e.from >= 0 {
				move = [2]int{e.from, i}
			}
		}
		ui.Divider(c)
		// The draft row has the key the pair it adds will, so that its
		// field keeps the keyboard focus as it becomes that pair's.
		e := tb.row(c.Key(len(*pairs)), draft, len(*pairs), true)
		if (e.changed || picked) && (draft.Key != "" || draft.Value != "") {
			*pairs = append(*pairs, *draft)
			*draft = Pair{}
			changed = true
		}
	})
	if remove >= 0 {
		*pairs = slices.Delete(*pairs, remove, remove+1)
		changed = true
	} else if from, to := move[0], move[1]; from >= 0 {
		p := (*pairs)[from]
		*pairs = slices.Insert(slices.Delete(*pairs, from, from+1), to, p)
		changed = true
	}
	if changed {
		el.SetChanged()
	}
	return el
}

// state is what a table keeps: its draft row, and the path Choose picked
// for row picked, -1 for none, which the next frame puts there.
type state struct {
	draft  Pair
	picked int
	path   string
}

// moving is a row dragged by its handle, from row from of the table of st.
type moving struct {
	st   *state
	from int
}

// table is what the rows of a table share: its options, and the names of
// its columns.
type table struct {
	c                    *ui.Context
	o                    Options
	st                   *state
	keyLabel, valueLabel string
}

// cell creates a cell of the header, of text.
func cell(c *ui.Context, text string, muted bool) ui.Element {
	t := c.Theme()
	e := ui.Text(c, text).FontSize(t.FontSize-1).Bold().SingleLine().Padding(t.Space(1.5), t.Space(2.5))
	if muted {
		e.TextColor(t.TextMuted)
	}
	return e
}

// edit is what a row's input did: from is the row dropped on it, -1 for
// none.
type edit struct {
	changed, remove bool
	from            int
}

// row builds the row of p, row i of the table, the draft row if draft,
// which has no handle, check box or button.
func (tb *table) row(c *ui.Context, p *Pair, i int, draft bool) edit {
	t := c.Theme()
	o := tb.o
	e := edit{from: -1}
	r := ui.Row(c).AlignItems(ui.Stretch)
	hovered := r.Hovered() || r.FocusWithin()
	if !draft && o.Reorder {
		if m, ok := ui.Drop[moving](r); ok && m.st == tb.st && m.from != i {
			e.from = m.from
		}
		// The line shows where the row dragged goes: after this one when it
		// comes from above, before it otherwise.
		if m, ok := ui.DragOver[moving](r); ok && m.st == tb.st && m.from != i {
			below := m.from < i
			r.DrawOver(func(p *ui.Painter, b ui.Rect) {
				y := b.Y - 1
				if below {
					y = b.Y + b.H - 1
				}
				p.Fill(ui.Rect{X: b.X, Y: y, W: b.W, H: 2}, t.Accent, 0)
			})
		}
	}
	color := t.Text
	if !p.On {
		color = t.TextMuted
	}
	r.Children(func() {
		if o.Reorder {
			grip := ui.Box(c).Width(t.Space(5)).Center()
			if !draft {
				grip.Cursor(ui.CursorGrab).Drag(moving{st: tb.st, from: i}).Children(func() {
					ui.Text(c, "⋮⋮").FontSize(t.FontSize - 2).TextColor(t.TextMuted)
				})
				if grip.Dragging() {
					r.Opacity(0.4)
				}
			}
		}
		ui.Box(c).Width(t.Space(9)).Center().Children(func() {
			if !draft {
				e.changed = ui.Checkbox(c, &p.On, "").Label("On").Changed() || e.changed
			}
		})
		if len(o.Kinds) > 0 {
			e.changed = tb.kind(c, p, color) || e.changed
		}
		key := tb.input(c, &p.Key, o.Keys).Grow(1).Basis(0).MinWidth(0).Padding(t.Space(1.5), t.Space(2.5)).
			TextColor(color).Label(tb.keyLabel).Placeholder(tb.keyLabel)
		e.changed = key.Changed() || e.changed
		ui.Divider(c)
		e.changed = tb.value(c, p, i, color) || e.changed
		ui.Box(c).Width(t.Space(8)).Center().Children(func() {
			if !draft && hovered {
				e.remove = button(c, "×", "Remove").Size(t.Space(5), t.Space(5)).Clicked()
			}
		})
	})
	return e
}

// kind creates the cell choosing p's kind, and reports a new choice.
func (tb *table) kind(c *ui.Context, p *Pair, color ui.Color) bool {
	t := c.Theme()
	sel := ui.SelectBase(c, &p.Kind)
	sel.Trigger.Width(t.Space(26)).Padding(t.Space(1.5), t.Space(2.5)).Justify(ui.SpaceBetween).AlignItems(ui.Center).
		Label("Kind").Children(func() {
		ui.Text(c, p.Kind).TextColor(color).SingleLine().Shrink(1)
		ui.Text(c, "▾").FontSize(t.FontSize - 2).TextColor(t.TextMuted)
	})
	if sel.Trigger.Hovered() {
		sel.Trigger.Background(t.SurfaceHover)
	}
	sel.Popup(func(panel ui.Element) {
		popup(c, panel)
		for _, k := range tb.o.Kinds {
			option(c, sel.Item(k.Name), k.Name)
		}
	})
	return sel.Trigger.Changed()
}

// value creates the cell of p's value, row i's, as its kind shows it, and
// reports an edit.
func (tb *table) value(c *ui.Context, p *Pair, i int, color ui.Color) bool {
	t := c.Theme()
	o := tb.o
	kind := Text
	if k := slices.IndexFunc(o.Kinds, func(k Kind) bool { return k.Name == p.Kind }); k >= 0 {
		kind = o.Kinds[k].Cell
	}
	if kind == File {
		return tb.file(c, p, i, color)
	}
	var suggest []string
	if o.Values != nil {
		suggest = o.Values(p.Key)
	}
	changed := false
	// The cell, not the input, has the padding at the sides, as the file
	// cell has, for the columns to share the row alike in every row.
	ui.Row(c).Grow(1.5).Basis(0).MinWidth(0).AlignItems(ui.Center).Gap(t.Space(2)).Padding(0, t.Space(2.5)).Children(func() {
		in := tb.input(c, &p.Value, suggest).Grow(1).MinWidth(0).Padding(t.Space(1.5), 0).
			TextColor(color).Label(tb.valueLabel).Placeholder(tb.valueLabel)
		changed = in.Changed()
		switch kind {
		case JSON:
			in.Font("monospace")
			if strings.TrimSpace(p.Value) != "" && !json.Valid([]byte(p.Value)) {
				in.TextColor(t.Danger)
			}
		case Secret:
			shown := ui.Local(in, "kvtable.shown", func() bool { return false })
			if !*shown {
				in.Password()
			}
			label := "Show"
			if *shown {
				label = "Hide"
			}
			if button(c, label, label).Padding(t.Space(0.5), t.Space(1.5)).Clicked() {
				*shown = !*shown
			}
		}
		if o.Ranges != nil {
			in.TextRanges(o.Ranges(p.Value)...)
		}
	})
	return changed
}

// file creates the cell of a File value, p's in row i, which takes files
// dropped on it, and reports one dropped.
func (tb *table) file(c *ui.Context, p *Pair, i int, color ui.Color) bool {
	t := c.Theme()
	changed := false
	box := ui.Row(c).Grow(1.5).Basis(0).MinWidth(0).AlignItems(ui.Center).Gap(t.Space(2)).Padding(t.Space(1.5), t.Space(2.5))
	if files := box.DroppedFiles(); len(files) > 0 {
		p.Value, changed = files[0], true
	}
	if box.FileDragOver() {
		box.Background(t.Accent.Alpha(0.12))
	}
	box.Children(func() {
		if p.Value == "" {
			ui.Text(c, "No file").TextColor(t.TextMuted).SingleLine().Grow(1)
		} else {
			ui.Text(c, filepath.Base(p.Value)).TextColor(color).SingleLine().Grow(1).Shrink(1).Tooltip(p.Value)
		}
		if tb.o.Choose != nil && button(c, "Choose…", "Choose a file").Padding(t.Space(0.5), t.Space(1.5)).Clicked() {
			st := tb.st
			tb.o.Choose(func(path string) { st.picked, st.path = i, path })
		}
	})
	return changed
}

// input creates a text input editing *value without a look, suggesting
// those of suggestions containing what was typed, as Options.Keys are.
func (tb *table) input(c *ui.Context, value *string, suggestions []string) ui.Element {
	if len(suggestions) == 0 {
		return ui.TextInputBase(c, value)
	}
	cb := ui.ComboboxBase(c, value)
	var shown []string
	if q := strings.ToLower(strings.TrimSpace(*value)); q != "" && cb.Filtering() {
		var starts, contains []string
		for _, s := range suggestions {
			switch l := strings.ToLower(s); {
			case l == q:
			case strings.HasPrefix(l, q):
				starts = append(starts, s)
			case strings.Contains(l, q):
				contains = append(contains, s)
			}
		}
		shown = append(starts, contains...)
		shown = shown[:min(len(shown), 8)]
	}
	cb.Popup(func(panel ui.Element) {
		popup(c, panel)
		for _, s := range shown {
			option(c, cb.Item(s), s)
		}
	})
	if v, ok := cb.Chosen(); ok && v != *value {
		*value = v
		cb.Input.SetChanged()
		c.Invalidate()
	}
	return cb.Input
}

// popup styles the panel of a select's or a field's options.
func popup(c *ui.Context, panel ui.Element) {
	t := c.Theme()
	panel.Margin(t.Space(1), 0, 0, 0).Padding(t.Space(1)).Radius(t.Radius).Background(t.Background).
		Border(1, t.Border).Shadow(0, 8, 24, 0, ui.RGBA(0, 0, 0, 0.16))
}

// option styles item, an option showing text.
func option(c *ui.Context, item ui.Element, text string) {
	t := c.Theme()
	item.Padding(t.Space(1.5), t.Space(2.5)).Radius(t.Space(1))
	if item.Highlighted() {
		item.Background(t.Accent).TextColor(t.AccentText)
	}
	item.Children(func() { ui.Text(c, text).SingleLine() })
}

// button creates a plain button showing text, named label.
func button(c *ui.Context, text, label string) ui.Element {
	t := c.Theme()
	b := ui.ButtonBase(c).Radius(t.Space(1)).Center().Label(label)
	if b.Hovered() {
		b.Background(t.SurfaceHover)
	}
	return b.Children(func() { ui.Text(c, text).FontSize(t.FontSize - 1).TextColor(t.TextMuted) })
}

// bulkText is the text a table in bulk shows, and the pairs it last showed
// or made, as format writes them: pairs changed since show anew.
type bulkText struct{ text, seen string }

// bulk creates the text area editing *pairs as text.
func bulk(c *ui.Context, pairs *[]Pair, o Options) ui.Element {
	box := ui.Column(c)
	b := ui.Local(box, "kvtable.bulk", func() bulkText { return bulkText{} })
	if f := format(*pairs); f != b.seen {
		b.text, b.seen = f, f
	}
	changed := false
	box.Children(func() {
		placeholder := cmp.Or(o.KeyLabel, "Key") + ": " + cmp.Or(o.ValueLabel, "Value")
		area := ui.TextArea(c, &b.text).Font("monospace").Lines(6, 20).Label("Pairs").Placeholder(placeholder)
		if area.Changed() {
			*pairs = parse(b.text, *pairs, o.Kinds)
			b.seen, changed = format(*pairs), true
		}
	})
	if changed {
		box.SetChanged()
	}
	return box
}

// format writes pairs as bulk shows them.
func format(pairs []Pair) string {
	var s strings.Builder
	for i, p := range pairs {
		if i > 0 {
			s.WriteByte('\n')
		}
		if !p.On {
			s.WriteString("# ")
		}
		s.WriteString(p.Key + ": " + p.Value)
	}
	return s.String()
}

// parse reads the pairs of text, as bulk shows them, each of the kind of
// the pair of old with its key, or of the first of kinds.
func parse(text string, old []Pair, kinds []Kind) []Pair {
	var out []Pair
	for line := range strings.Lines(text) {
		line = strings.TrimSpace(line)
		rest, off := strings.CutPrefix(line, "#")
		if rest = strings.TrimSpace(rest); rest == "" {
			continue
		}
		k, v, _ := strings.Cut(rest, ":")
		p := Pair{Key: strings.TrimSpace(k), Value: strings.TrimSpace(v), On: !off}
		if i := slices.IndexFunc(old, func(q Pair) bool { return q.Key == p.Key }); i >= 0 {
			p.Kind = old[i].Kind
		} else if len(kinds) > 0 {
			p.Kind = kinds[0].Name
		}
		out = append(out, p)
	}
	return out
}
