// Package jsonview shows JSON as a tree to open and close, as Firefox's
// JSON viewer does: each member or element on a row, its name in one
// color and its value in the color of its type, objects and arrays with
// how many they hold.
//
//	doc, err := jsonview.Parse(data)
//	...
//	jsonview.View(c, doc, &app.tree).Grow(1)
//	ui.Text(c, app.tree.Path()) // $.store.books[0].title
//
// The view is a ui.Outline, which builds only the rows in view, so
// documents of any size scroll as fast as small ones.
package jsonview

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
)

// Document is parsed JSON, its objects' members in the order written.
type Document struct {
	roots []*node
}

type kind uint8

const (
	object kind = iota
	array
	str
	number
	boolean
	null
)

// node is a value of a document: its name in its object, or index in its
// array, with -1 for the top value, what it holds, and its parent.
type node struct {
	name   string
	index  int
	kind   kind
	value  string // a string's text, or a number's, true, false or null
	kids   []*node
	parent *node
}

// Parse parses a JSON document, a value and nothing after it but space.
func Parse(data []byte) (*Document, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	top, err := parse(dec, nil)
	if err != nil {
		return nil, err
	}
	if _, err := dec.Token(); err != io.EOF {
		return nil, fmt.Errorf("jsonview: data after the value at offset %d", dec.InputOffset())
	}
	top.index = -1
	doc := &Document{roots: []*node{top}}
	if top.kind == object || top.kind == array {
		doc.roots = top.kids
	}
	return doc, nil
}

// parse reads the next value of dec, a child of parent.
func parse(dec *json.Decoder, parent *node) (*node, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	n := &node{parent: parent}
	switch v := tok.(type) {
	case json.Delim:
		n.kind = object
		if v == '[' {
			n.kind = array
		}
		for dec.More() {
			name := ""
			if n.kind == object {
				t, err := dec.Token()
				if err != nil {
					return nil, err
				}
				name = t.(string)
			}
			kid, err := parse(dec, n)
			if err != nil {
				return nil, err
			}
			kid.name, kid.index = name, len(n.kids)
			n.kids = append(n.kids, kid)
		}
		if _, err := dec.Token(); err != nil {
			return nil, err
		}
	case string:
		n.kind, n.value = str, v
	case json.Number:
		n.kind, n.value = number, v.String()
	case bool:
		n.kind, n.value = boolean, strconv.FormatBool(v)
	case nil:
		n.kind, n.value = null, "null"
	}
	return n, nil
}

// path returns where n is in its document, as JSONPath writes it:
// $.store.books[0].title, with names that are not identifiers in
// brackets, $["a b"].
func (n *node) path() string {
	var parts []string
	for ; n != nil && n.index >= 0; n = n.parent {
		switch {
		case n.parent.kind == array:
			parts = append(parts, "["+strconv.Itoa(n.index)+"]")
		case identifier(n.name):
			parts = append(parts, "."+n.name)
		default:
			name, _ := json.Marshal(n.name)
			parts = append(parts, "["+string(name)+"]")
		}
	}
	var b strings.Builder
	b.WriteString("$")
	for i := len(parts) - 1; i >= 0; i-- {
		b.WriteString(parts[i])
	}
	return b.String()
}

func identifier(s string) bool {
	for i, r := range s {
		if r != '_' && r != '$' && !(r >= 'a' && r <= 'z') && !(r >= 'A' && r <= 'Z') && !(i > 0 && r >= '0' && r <= '9') {
			return false
		}
	}
	return s != ""
}

// text returns n as it copies: a string's text, a number or keyword as
// written, and an object or array as indented JSON.
func (n *node) text() string {
	if n.kind == str {
		return n.value
	}
	var b strings.Builder
	n.write(&b, "")
	return b.String()
}

// write writes n as JSON, indented by two spaces from indent.
func (n *node) write(b *strings.Builder, indent string) {
	switch n.kind {
	case str:
		v, _ := json.Marshal(n.value)
		b.Write(v)
		return
	case number, boolean, null:
		b.WriteString(n.value)
		return
	}
	open, end := "{", "}"
	if n.kind == array {
		open, end = "[", "]"
	}
	if len(n.kids) == 0 {
		b.WriteString(open + end)
		return
	}
	b.WriteString(open + "\n")
	for i, k := range n.kids {
		b.WriteString(indent + "  ")
		if n.kind == object {
			name, _ := json.Marshal(k.name)
			b.Write(name)
			b.WriteString(": ")
		}
		k.write(b, indent+"  ")
		if i < len(n.kids)-1 {
			b.WriteByte(',')
		}
		b.WriteByte('\n')
	}
	b.WriteString(indent + end)
}

// State is what a view of a document keeps: the rows open and chosen.
// Keep it in the app's state, one for each view.
type State struct {
	outline  ui.OutlineState[*node]
	selected int
	doc      *Document
}

// Path returns where the value of the row chosen is, as JSONPath writes
// it, "$.store.books[0].title", or "" with no row chosen.
func (s *State) Path() string {
	if n := s.chosen(); n != nil {
		return n.path()
	}
	return ""
}

func (s *State) chosen() *node {
	if s.selected < 0 || s.selected >= s.outline.Rows() {
		return nil
	}
	return s.outline.Item(s.selected)
}

// View creates a tree of doc, of which s says what is open and chosen:
// the members of the object at the top, or the elements of the array,
// each on a row, and below each object or array open what it holds. Give
// it a size, or grow it in its parent.
//
// A click on a row's arrow, or Right and Left while the view has the
// keyboard focus, opens and closes it, with Option (Alt) opening and
// closing all inside, as ui.Outline does. Cmd+C copies the value of the
// row chosen, and its context menu copies its value or its path: a
// string's text, a number or keyword as written, an object or array as
// indented JSON.
func View(c *ui.Context, doc *Document, s *State) ui.Element {
	t := c.Theme()
	if s.doc != doc {
		*s = State{doc: doc, selected: -1}
	}
	s.outline.List.Selected = &s.selected
	palette := lightColors
	if t.Dark {
		palette = darkColors
	}
	tree := ui.Outline(c, &s.outline, doc.roots, func(n *node) []*node { return n.kids }, func(n *node) {
		colors := palette
		colors.muted = t.TextMuted
		// Colors of types are unreadable on the accent of the row chosen.
		if s.chosen() == n {
			a := t.AccentText
			colors = syntax{name: a, str: a, number: a, keyword: a, muted: a}
		}
		row := ui.Row(c).Gap(t.Space(1)).AlignItems(ui.Center).ContextMenu(func(m *ui.Menu) {
			if m.Item("Copy Value").Chosen() {
				c.WriteClipboard(n.text())
			}
			if m.Item("Copy Path").Chosen() {
				c.WriteClipboard(n.path())
			}
		})
		row.Children(func() {
			if n.index >= 0 {
				name, color := n.name, colors.name
				if n.parent.kind == array {
					name, color = strconv.Itoa(n.index), colors.muted
				}
				ui.Text(c, name).TextColor(color).NoWrap()
				ui.Text(c, ":").TextColor(colors.muted)
			}
			value(c, n, colors)
		})
	}).Font("monospace").FontSize(t.FontSize - 1)
	if n := s.chosen(); n != nil && tree.Shortcut(ui.Cmd, ui.KeyC) {
		c.WriteClipboard(n.text())
	}
	return tree
}

// value shows the value of n: a scalar in the color of its type, an
// object or array as how many it holds.
func value(c *ui.Context, n *node, colors syntax) {
	switch n.kind {
	case object, array:
		count, open, end := len(n.kids), "{", "}"
		if n.kind == array {
			open, end = "[", "]"
		}
		label := strconv.Itoa(count) + " keys"
		switch {
		case n.kind == array && count == 1:
			label = "1 item"
		case n.kind == array:
			label = strconv.Itoa(count) + " items"
		case count == 1:
			label = "1 key"
		}
		ui.Text(c, open+label+end).TextColor(colors.muted).NoWrap()
	case str:
		v, _ := json.Marshal(n.value)
		ui.Text(c, string(v)).TextColor(colors.str).SingleLine().Shrink(1)
	case number:
		ui.Text(c, n.value).TextColor(colors.number).NoWrap()
	default:
		ui.Text(c, n.value).TextColor(colors.keyword).NoWrap()
	}
}

// syntax is the colors of names and values, as Visual Studio Code colors
// JSON, and of the rest, indexes, counts and punctuation.
type syntax struct {
	name, str, number, keyword, muted ui.Color
}

var (
	lightColors = syntax{name: ui.Hex("#0451a5"), str: ui.Hex("#a31515"), number: ui.Hex("#098658"), keyword: ui.Hex("#0000ff")}
	darkColors  = syntax{name: ui.Hex("#9cdcfe"), str: ui.Hex("#ce9178"), number: ui.Hex("#b5cea8"), keyword: ui.Hex("#569cd6")}
)
