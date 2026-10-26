// Package schemaeditor edits the JSON Schema of an object, as the
// parameters of a model's tool or the shape of a structured output are
// written: a row for each field, with its name, its type, whether it is
// required and what it is for, nested under an object's or an array's
// row as deep as they go. A string's row opens to the values it may
// only take. A name that another field of the same object has shows
// red; the schema leaves out that field, and fields without a name.
//
//	if schemaeditor.Editor(c, &app.params).Changed() {
//		app.tool.InputSchema = schemaeditor.JSON(app.params)
//	}
//
// Parse reads the fields of a schema written elsewhere.
package schemaeditor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/egoist/mygo/ui"
)

// Type is the type of a field's values.
type Type string

const (
	String  Type = "string"
	Number  Type = "number"
	Integer Type = "integer"
	Boolean Type = "boolean"
	Object  Type = "object"
	Array   Type = "array"
)

var types = []Type{String, Number, Integer, Boolean, Object, Array}

// Field is a property of an object, or the items of an array, which have
// no Name and are never Required. An object's Fields are its properties;
// an array's Items is what it holds, a String when nil. Enum lists the
// only values a String may take.
type Field struct {
	Name        string
	Type        Type
	Description string
	Required    bool
	Enum        []string
	Fields      []Field
	Items       *Field
}

// Editor creates the editor of *fields, an object's properties. Changed
// reports an edit.
func Editor(c *ui.Context, fields *[]Field) ui.Element {
	t := c.Theme()
	changed := false
	el := ui.Column(c).Border(1, t.Border).Radius(t.Radius).Clip()
	el.Children(func() {
		ui.Row(c).Background(t.Surface).Children(func() {
			ui.Box(c).Width(t.Space(6))
			head(c, "Name").Width(t.Space(44))
			head(c, "Type").Width(t.Space(28))
			head(c, "Required").Width(t.Space(20))
			head(c, "Description").Grow(1)
			ui.Box(c).Width(t.Space(16))
		})
		changed = properties(c, fields, 0)
	})
	if changed {
		el.SetChanged()
	}
	return el
}

// head creates the head of a column.
func head(c *ui.Context, text string) ui.Element {
	t := c.Theme()
	return ui.Text(c, text).FontSize(t.FontSize-1).Bold().TextColor(t.TextMuted).SingleLine().Padding(t.Space(1.5), t.Space(2))
}

// properties builds the rows of *fields, depth levels down, and the row
// adding one, and reports an edit.
func properties(c *ui.Context, fields *[]Field, depth int) bool {
	changed, remove := false, -1
	list := ui.Column(c)
	// The field added last frame, whose name takes the focus.
	added := ui.Local(list, "schemaeditor.added", func() int { return -1 })
	list.Children(func() {
		for i := range *fields {
			f := &(*fields)[i]
			dup := f.Name != "" && slices.ContainsFunc((*fields)[:i], func(g Field) bool { return g.Name == f.Name })
			e := row(c.Key(i), f, depth, false, dup, i == *added)
			changed = changed || e.changed
			if e.remove {
				remove = i
			}
		}
		*added = -1
		if add(c.Key("add"), depth) {
			*fields = append(*fields, Field{Type: String})
			*added = len(*fields) - 1
			c.Invalidate()
			changed = true
		}
	})
	if remove >= 0 {
		*fields = slices.Delete(*fields, remove, remove+1)
		changed = true
	}
	return changed
}

// edit is what a row did.
type edit struct{ changed, remove bool }

// row builds the row of f, depth levels down, and the rows under it, its
// name marked taken if dup and given the focus if focus. The items of an
// array show "items" for a name.
func row(c *ui.Context, f *Field, depth int, items, dup, focus bool) edit {
	t := c.Theme()
	var e edit
	if f.Type == "" {
		f.Type = String
	}
	box := ui.Column(c)
	open := ui.Local(box, "schemaeditor.open", func() bool { return len(f.Enum) > 0 })
	box.Children(func() {
		ui.Divider(c)
		r := ui.Row(c).AlignItems(ui.Center).MinHeight(t.Space(9))
		r.Children(func() {
			ui.Box(c).Width(t.Space(6)).Center().Children(func() {
				if f.Type == String {
					arrow := "▸"
					if *open {
						arrow = "▾"
					}
					if button(c, arrow, "Values").Size(t.Space(5), t.Space(5)).Clicked() {
						*open = !*open
					}
				}
			})
			ui.Row(c).Width(t.Space(44)).AlignItems(ui.Center).Children(func() {
				guides(c, depth)
				if items {
					ui.Text(c, "items").TextColor(t.TextMuted).Font("monospace").Padding(t.Space(1.5), t.Space(2))
					return
				}
				in := ui.TextInputBase(c, &f.Name).Grow(1).MinWidth(0).Padding(t.Space(1.5), t.Space(2)).
					Font("monospace").Label("Name").Placeholder("name")
				if dup {
					in.TextColor(t.Danger).Error("Another field has this name")
				}
				if focus {
					in.Focus()
				}
				e.changed = in.Changed() || e.changed
			})
			e.changed = kind(c, f) || e.changed
			ui.Box(c).Width(t.Space(20)).Padding(0, t.Space(2)).Children(func() {
				if !items {
					e.changed = ui.Checkbox(c, &f.Required, "").Label("Required").Changed() || e.changed
				}
			})
			desc := ui.TextInputBase(c, &f.Description).Grow(1).MinWidth(0).Padding(t.Space(1.5), t.Space(2)).
				Label("Description").Placeholder("Description")
			e.changed = desc.Changed() || e.changed
			ui.Row(c).Width(t.Space(16)).Justify(ui.End).Padding(0, t.Space(2)).Children(func() {
				if !items && r.Hovered() {
					e.remove = button(c, "×", "Remove").Size(t.Space(5), t.Space(5)).Clicked()
				}
			})
		})
		if f.Type == String && *open {
			ui.Row(c).AlignItems(ui.Center).Padding(t.Space(1), 0).Children(func() {
				ui.Box(c).Width(t.Space(6))
				ui.Row(c).Width(t.Space(44)).AlignItems(ui.Center).Children(func() {
					guides(c, depth+1)
					ui.Text(c, "One of").FontSize(t.FontSize-1).TextColor(t.TextMuted).Padding(0, t.Space(2))
				})
				before := len(f.Enum)
				ui.TokenField(c, &f.Enum, nil).Grow(1).MarginEnd(t.Space(2)).Label("Values").Placeholder("Any value")
				e.changed = len(f.Enum) != before || e.changed
			})
		}
		switch f.Type {
		case Object:
			e.changed = properties(c.Key("fields"), &f.Fields, depth+1) || e.changed
		case Array:
			if f.Items == nil {
				f.Items = &Field{Type: String}
			}
			e.changed = row(c.Key("items"), f.Items, depth+1, true, false, false).changed || e.changed
		}
	})
	return e
}

// kind creates the cell choosing f's type, and reports a new choice.
func kind(c *ui.Context, f *Field) bool {
	t := c.Theme()
	sel := ui.SelectBase(c, &f.Type)
	sel.Trigger.Width(t.Space(28)).Padding(t.Space(1.5), t.Space(2)).Justify(ui.SpaceBetween).AlignItems(ui.Center).
		Label("Type").Children(func() {
		ui.Text(c, string(f.Type)).Font("monospace").TextColor(color(t, f.Type)).SingleLine()
		ui.Text(c, "▾").FontSize(t.FontSize - 2).TextColor(t.TextMuted)
	})
	if sel.Trigger.Hovered() {
		sel.Trigger.Background(t.SurfaceHover)
	}
	sel.Popup(func(panel ui.Element) {
		panel.Margin(t.Space(1), 0, 0, 0).Padding(t.Space(1)).Radius(t.Radius).Background(t.Background).
			Border(1, t.Border).Shadow(0, 8, 24, 0, ui.RGBA(0, 0, 0, 0.16))
		for _, ty := range types {
			item := sel.Item(ty).Padding(t.Space(1.5), t.Space(2.5)).Radius(t.Space(1))
			text := color(t, ty)
			if item.Highlighted() {
				item.Background(t.Accent)
				text = t.AccentText
			}
			item.Children(func() { ui.Text(c, string(ty)).Font("monospace").TextColor(text) })
		}
	})
	return sel.Trigger.Changed()
}

// color returns the color of a type, as editors color JSON's values.
func color(t *ui.Theme, ty Type) ui.Color {
	hue := map[Type]float32{String: 145, Number: 60, Integer: 60, Boolean: 300, Object: 250, Array: 200}[ty]
	l := float32(0.5)
	if t.Dark {
		l = 0.78
	}
	return ui.Oklch(l, 0.13, hue)
}

// guides draws a guide for each level of depth.
func guides(c *ui.Context, depth int) {
	t := c.Theme()
	if depth == 0 {
		return
	}
	ui.Box(c).Width(float32(depth) * t.Space(5)).AlignSelf(ui.Stretch).Draw(func(p *ui.Painter, r ui.Rect) {
		for d := range depth {
			x := r.X + float32(d)*t.Space(5) + t.Space(2.5)
			p.Line(x, r.Y, x, r.Y+r.H, 1, t.Text.Alpha(0.12))
		}
	})
}

// add creates the row adding a field, depth levels down, and reports a
// click.
func add(c *ui.Context, depth int) bool {
	t := c.Theme()
	clicked := false
	ui.Column(c).Children(func() {
		ui.Divider(c)
		ui.Row(c).AlignItems(ui.Center).Children(func() {
			ui.Box(c).Width(t.Space(6))
			guides(c, depth)
			b := ui.ButtonBase(c).Padding(t.Space(1.5), t.Space(2)).Radius(t.Space(1)).Label("Add field")
			if b.Hovered() {
				b.Background(t.SurfaceHover)
			}
			b.Children(func() { ui.Text(c, "+ Add field").FontSize(t.FontSize - 1).TextColor(t.Accent) })
			clicked = b.Clicked()
		})
	})
	return clicked
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

// JSON returns the schema of an object of fields, indented, its
// properties in their order.
func JSON(fields []Field) []byte {
	var b bytes.Buffer
	object(&b, fields, "", "")
	return b.Bytes()
}

// object writes the schema of an object of fields, described as desc,
// indent deep.
func object(b *bytes.Buffer, fields []Field, desc, indent string) {
	in := indent + "  "
	b.WriteString("{\n" + in + `"type": "object",`)
	if desc != "" {
		b.WriteString("\n" + in + `"description": ` + quote(desc) + ",")
	}
	b.WriteString("\n" + in + `"properties": {`)
	var required []string
	var named []Field
	for _, f := range fields {
		if f.Name != "" && !slices.ContainsFunc(named, func(g Field) bool { return g.Name == f.Name }) {
			named = append(named, f)
		}
	}
	fields = named
	for i, f := range fields {
		if i > 0 {
			b.WriteString(",")
		}
		b.WriteString("\n" + in + "  " + quote(f.Name) + ": ")
		schema(b, f, in+"  ")
		if f.Required {
			required = append(required, f.Name)
		}
	}
	if len(fields) > 0 {
		b.WriteString("\n" + in)
	}
	b.WriteString("}")
	if len(required) > 0 {
		b.WriteString(",\n" + in + `"required": [`)
		for i, name := range required {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(quote(name))
		}
		b.WriteString("]")
	}
	b.WriteString(",\n" + in + `"additionalProperties": false` + "\n" + indent + "}")
}

// schema writes the schema of f, indent deep.
func schema(b *bytes.Buffer, f Field, indent string) {
	if f.Type == Object {
		object(b, f.Fields, f.Description, indent)
		return
	}
	in := indent + "  "
	b.WriteString("{\n" + in + `"type": ` + quote(string(orString(f.Type))))
	if f.Description != "" {
		b.WriteString(",\n" + in + `"description": ` + quote(f.Description))
	}
	if f.Type == String && len(f.Enum) > 0 {
		b.WriteString(",\n" + in + `"enum": [`)
		for i, v := range f.Enum {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(quote(v))
		}
		b.WriteString("]")
	}
	if f.Type == Array {
		items := Field{Type: String}
		if f.Items != nil {
			items = *f.Items
		}
		b.WriteString(",\n" + in + `"items": `)
		schema(b, items, in)
	}
	b.WriteString("\n" + indent + "}")
}

// orString returns t, or String when it is empty.
func orString(t Type) Type {
	if t == "" {
		return String
	}
	return t
}

func quote(s string) string {
	q, _ := json.Marshal(s)
	return string(q)
}

// Parse returns the fields of the schema of an object in data, in the
// order its properties are written. It takes the parts of JSON Schema
// the editor edits, and leaves the rest.
func Parse(data []byte) ([]Field, error) {
	var root node
	if err := json.Unmarshal(data, &root); err != nil {
		return nil, err
	}
	if root.Type != "" && root.Type != Object {
		return nil, fmt.Errorf("schemaeditor: the schema is of %s, not of an object", root.Type)
	}
	return root.fields(), nil
}

// node is a schema as JSON writes it, its properties in order.
type node struct {
	Type        Type     `json:"type"`
	Description string   `json:"description"`
	Enum        []string `json:"enum"`
	Required    []string `json:"required"`
	Items       *node    `json:"items"`
	Properties  props    `json:"properties"`
}

// props are an object's properties in the order written.
type props []struct {
	name string
	node node
}

func (p *props) UnmarshalJSON(data []byte) error {
	d := json.NewDecoder(bytes.NewReader(data))
	if _, err := d.Token(); err != nil {
		return err
	}
	for d.More() {
		tok, err := d.Token()
		if err != nil {
			return err
		}
		var n node
		if err := d.Decode(&n); err != nil {
			return err
		}
		*p = append(*p, struct {
			name string
			node node
		}{tok.(string), n})
	}
	return nil
}

func (n node) fields() []Field {
	var out []Field
	for _, p := range n.Properties {
		f := p.node.field()
		f.Name, f.Required = p.name, slices.Contains(n.Required, p.name)
		out = append(out, f)
	}
	return out
}

func (n node) field() Field {
	f := Field{Type: orString(n.Type), Description: n.Description, Enum: n.Enum}
	switch f.Type {
	case Object:
		f.Fields = n.fields()
	case Array:
		items := Field{Type: String}
		if n.Items != nil {
			items = n.Items.field()
		}
		f.Items = &items
	}
	return f
}
