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
package kvtable

import (
	"slices"

	"github.com/egoist/mygo/ui"
)

// Pair is a key and its value, which counts while On.
type Pair struct {
	Key, Value string
	On         bool
}

// Editor creates a table editing *pairs, under a header naming its
// columns: a row for each pair, which its check box turns on and off and
// its button, shown as the pointer is over the row, removes, and a last
// row adding the pair typed in it, on. Tab goes from field to field.
// Changed reports any edit.
func Editor(c *ui.Context, pairs *[]Pair) ui.Element {
	t := c.Theme()
	table := ui.Column(c).Border(1, t.Border).Radius(t.Radius).Clip()
	draft := ui.Local(table, "kvtable.draft", func() Pair { return Pair{On: true} })
	changed := false
	remove := -1
	table.Children(func() {
		ui.Row(c).Background(t.Surface).Children(func() {
			cell(c, "", false).Width(t.Space(9))
			cell(c, "Key", true).Grow(1).Basis(0)
			ui.Divider(c)
			cell(c, "Value", true).Grow(1.5).Basis(0)
			ui.Box(c).Width(t.Space(8))
		})
		for i := range *pairs {
			ui.Divider(c)
			p := &(*pairs)[i]
			r := row(c.Key(i), p, false)
			changed = changed || r.changed
			if r.remove {
				remove = i
			}
		}
		ui.Divider(c)
		// The draft row has the key the pair it adds will, so that its
		// field keeps the keyboard focus as it becomes that pair's.
		if row(c.Key(len(*pairs)), draft, true).changed && (draft.Key != "" || draft.Value != "") {
			*pairs = append(*pairs, *draft)
			*draft = Pair{On: true}
			changed = true
		}
	})
	if remove >= 0 {
		*pairs = slices.Delete(*pairs, remove, remove+1)
		changed = true
	}
	if changed {
		table.SetChanged()
	}
	return table
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

// edit is what a row's input did.
type edit struct{ changed, remove bool }

// row builds the row of p, the draft row if draft, which has no check box
// or button.
func row(c *ui.Context, p *Pair, draft bool) edit {
	t := c.Theme()
	var e edit
	r := ui.Row(c).AlignItems(ui.Stretch)
	hovered := r.Hovered()
	r.Children(func() {
		ui.Box(c).Width(t.Space(9)).Center().Children(func() {
			if !draft {
				e.changed = ui.Checkbox(c, &p.On, "").Label("On").Changed() || e.changed
			}
		})
		color := t.Text
		if !p.On {
			color = t.TextMuted
		}
		key := ui.TextInputBase(c, &p.Key).Grow(1).Basis(0).MinWidth(0).Padding(t.Space(1.5), t.Space(2.5)).
			TextColor(color).Label("Key").Placeholder("Key")
		ui.Divider(c)
		value := ui.TextInputBase(c, &p.Value).Grow(1.5).Basis(0).MinWidth(0).Padding(t.Space(1.5), t.Space(2.5)).
			TextColor(color).Label("Value").Placeholder("Value")
		e.changed = key.Changed() || value.Changed() || e.changed
		ui.Box(c).Width(t.Space(8)).Center().Children(func() {
			if !draft && (hovered || key.Focused() || value.Focused()) {
				x := ui.ButtonBase(c).Size(t.Space(5), t.Space(5)).Radius(t.Space(1)).Center().Label("Remove")
				if x.Hovered() {
					x.Background(t.SurfaceHover)
				}
				e.remove = x.Clicked()
				x.Children(func() { ui.Text(c, "×").TextColor(t.TextMuted) })
			}
		})
	})
	return e
}
