// Package jsondiff shows how two JSON documents differ, as one tree of
// their members in the order written: a member added with a + on green,
// one removed with a − on red, and a value changed as old → new. Objects
// are compared by their members' names and arrays by index, and values
// in the colors of their types, as jsonview colors them. Objects and
// arrays with nothing changed inside start closed, with how many members
// they hold.
//
//	app.diff = jsondiff.Compare(old, new)
//	...
//	jsondiff.View(c, &app.state, app.diff).Grow(1)
//	ui.Text(c, app.state.Path()) // $.owner.login
//
// A header counts the additions, removals and changes, and its Only
// changes hides the members with no change in them. A document that does
// not parse shows in place of the tree as the error, its line and column,
// and a caret under where it is.
package jsondiff

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
)

// Diff is how two JSON documents differ, or why they did not parse.
type Diff struct {
	// root holds the rows at the top: the members of the objects, or the
	// elements of the arrays, at the top of both, or else the top values.
	root *node
	// changes is the members added, removed and changed, in the order
	// they show.
	changes []*node
	nodes   int
	errs    []*parseError
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

// value is a value of a document: a scalar as written, or an object's or
// array's members, its object's names in order.
type value struct {
	kind  kind
	text  string
	names []string
	kids  []*value
}

type op uint8

const (
	same op = iota
	added
	removed
	changed
)

// node is a row of a diff: a member of both documents, or of one, by its
// name in its object or its index in its array, -1 in an object.
type node struct {
	name     string
	index    int
	parent   *node
	op       op
	old, new *value
	// kids is the rows inside, nil for a row that does not open, and
	// inner those with a change in them.
	kids, inner []*node
	// order is where the row is among all rows, open or not.
	order int
}

// Compare parses old and new, JSON documents, and finds how they differ.
// When either does not parse, the diff holds why, which View shows.
func Compare(old, new []byte) *Diff {
	d := &Diff{}
	a, err := parse("Old", old)
	if err != nil {
		d.errs = append(d.errs, err)
	}
	b, err := parse("New", new)
	if err != nil {
		d.errs = append(d.errs, err)
	}
	if d.errs != nil {
		return d
	}
	top := d.compare(nil, "", -1, a, b)
	d.root = top
	if top.op != same || top.kids == nil {
		d.root = &node{kids: []*node{top}}
		if top.op != same {
			d.root.inner = d.root.kids
		}
	}
	return d
}

// compare makes the row of a member of parent, old a in the old document
// and b in the new, nil where it is not.
func (d *Diff) compare(parent *node, name string, index int, a, b *value) *node {
	n := &node{name: name, index: index, parent: parent, old: a, new: b, order: d.nodes}
	d.nodes++
	switch {
	case a == nil:
		n.op = added
	case b == nil:
		n.op = removed
	case a.kind != b.kind || a.text != b.text:
		n.op = changed
	}
	if n.op != same {
		d.changes = append(d.changes, n)
		if n.op != added {
			d.whole(n, a, removed)
		}
		if n.op != removed {
			d.whole(n, b, added)
		}
		n.inner = n.kids
		return n
	}
	switch a.kind {
	case object:
		at := map[string]int{}
		for i, name := range a.names {
			at[name] = i
		}
		kept := map[string]bool{}
		for _, name := range b.names {
			kept[name] = true
		}
		// upto[i] is where in a the member kept at or after b's i is, so
		// that members removed show where they were, before those added
		// there.
		upto := make([]int, len(b.names)+1)
		upto[len(b.names)] = len(a.names)
		for i := len(b.names) - 1; i >= 0; i-- {
			upto[i] = upto[i+1]
			if k, ok := at[b.names[i]]; ok {
				upto[i] = k
			}
		}
		next := 0
		gone := func(to int) {
			for ; next < to; next++ {
				if !kept[a.names[next]] {
					n.kids = append(n.kids, d.compare(n, a.names[next], -1, a.kids[next], nil))
				}
			}
		}
		for i, name := range b.names {
			gone(upto[i])
			var old *value
			if k, ok := at[name]; ok {
				next = max(next, k+1)
				old = a.kids[k]
			}
			n.kids = append(n.kids, d.compare(n, name, -1, old, b.kids[i]))
		}
		gone(len(a.names))
	case array:
		for i := range max(len(a.kids), len(b.kids)) {
			var x, y *value
			if i < len(a.kids) {
				x = a.kids[i]
			}
			if i < len(b.kids) {
				y = b.kids[i]
			}
			n.kids = append(n.kids, d.compare(n, "", i, x, y))
		}
	}
	for _, k := range n.kids {
		if k.op != same || k.inner != nil {
			n.inner = append(n.inner, k)
		}
	}
	return n
}

// whole adds to parent the rows of the members of v, each and all inside
// it added or removed with it.
func (d *Diff) whole(parent *node, v *value, o op) {
	for i, k := range v.kids {
		kid := &node{index: i, parent: parent, op: o, order: d.nodes}
		d.nodes++
		if v.kind == object {
			kid.name, kid.index = v.names[i], -1
		}
		if o == added {
			kid.new = k
		} else {
			kid.old = k
		}
		d.whole(kid, k, o)
		kid.inner = kid.kids
		parent.kids = append(parent.kids, kid)
	}
}

// parseError is where a document did not parse: its line and column, the
// text of the line around the column and where the column is in it.
type parseError struct {
	doc       string
	line, col int
	text      string
	at        int
	msg       string
}

// parse parses a JSON document, a value and nothing after it but space.
func parse(doc string, data []byte) (*value, *parseError) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	v, err := read(dec)
	if err == nil {
		off := int(dec.InputOffset())
		if _, err := dec.Token(); err != io.EOF {
			off += len(data[off:]) - len(bytes.TrimLeft(data[off:], " \t\r\n"))
			return nil, locate(doc, data, off, "data after the value")
		}
		return v, nil
	}
	if se := (*json.SyntaxError)(nil); errors.As(err, &se) {
		// Offset is past the byte that did not parse.
		return nil, locate(doc, data, int(se.Offset)-1, se.Error())
	}
	return nil, locate(doc, data, len(data), "unexpected end of JSON input")
}

// read reads the next value of dec.
func read(dec *json.Decoder) (*value, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	v := &value{}
	switch t := tok.(type) {
	case json.Delim:
		v.kind = object
		if t == '[' {
			v.kind = array
		}
		for dec.More() {
			if v.kind == object {
				name, err := dec.Token()
				if err != nil {
					return nil, err
				}
				v.names = append(v.names, name.(string))
			}
			kid, err := read(dec)
			if err != nil {
				return nil, err
			}
			v.kids = append(v.kids, kid)
		}
		if _, err := dec.Token(); err != nil {
			return nil, err
		}
	case string:
		v.kind, v.text = str, t
	case json.Number:
		v.kind, v.text = number, t.String()
	case bool:
		v.kind, v.text = boolean, strconv.FormatBool(t)
	case nil:
		v.kind, v.text = null, "null"
	}
	return v, nil
}

// locate returns the error msg at byte off of data, with the text of its
// line up to 40 runes on each side of it.
func locate(doc string, data []byte, off int, msg string) *parseError {
	off = max(0, min(off, len(data)))
	start := bytes.LastIndexByte(data[:off], '\n') + 1
	end := bytes.IndexByte(data[off:], '\n')
	if end < 0 {
		end = len(data) - off
	}
	before := []rune(string(data[start:off]))
	after := []rune(string(bytes.TrimRight(data[off:off+end], "\r")))
	e := &parseError{doc: doc, line: bytes.Count(data[:off], []byte("\n")) + 1, col: len(before) + 1, msg: msg}
	if len(before) > 40 {
		before = append([]rune("…"), before[len(before)-39:]...)
	}
	if len(after) > 40 {
		after = append(after[:39], '…')
	}
	// A tab is one column, as the caret below it counts.
	e.text = strings.ReplaceAll(string(before)+string(after), "\t", " ")
	e.at = len(before)
	return e
}

// path returns where n is in its documents, as JSONPath writes it:
// $.owner.login, with names that are not identifiers in brackets,
// $["a b"].
func (n *node) path() string {
	var parts []string
	for ; n.parent != nil; n = n.parent {
		switch {
		case n.index >= 0:
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
	for _, p := range slices.Backward(parts) {
		b.WriteString(p)
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

// State is what a view of a diff keeps: the rows open and chosen, and
// whether only changes show. Keep it in the app's state, one for each
// view.
type State struct {
	outline  ui.OutlineState[*node]
	selected int
	only     bool
	diff     *Diff
}

// Path returns where the member of the row chosen is, as JSONPath writes
// it, "$.owner.login", or "" with no row chosen.
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

// kids returns the rows of n that show.
func (s *State) kids(n *node) []*node {
	if s.only {
		return n.inner
	}
	return n.kids
}

// step chooses the change after the row chosen, or before it, going
// around past the last and first, and opens the rows around it.
func (s *State) step(forward bool) {
	ch := s.diff.changes
	if len(ch) == 0 {
		return
	}
	at := -1
	if n := s.chosen(); n != nil {
		at = n.order
	}
	to := ch[len(ch)-1]
	if forward {
		to = ch[0]
		if i := slices.IndexFunc(ch, func(n *node) bool { return n.order > at }); i >= 0 {
			to = ch[i]
		}
	} else if i := slices.IndexFunc(ch, func(n *node) bool { return n.order >= at }); i > 0 {
		to = ch[i-1]
	}
	for p := to.parent; p != nil; p = p.parent {
		s.outline.Open.Add(p)
	}
	row := 0
	var find func(rows []*node) bool
	find = func(rows []*node) bool {
		for _, n := range rows {
			if n == to {
				return true
			}
			row++
			if s.outline.Open.Has(n) && find(s.kids(n)) {
				return true
			}
		}
		return false
	}
	if find(s.kids(s.diff.root)) {
		s.selected = row
		s.outline.List.ScrollIntoView(row)
	}
}

// View creates the view of d, of which s says what is open and chosen: a
// header of how many members were added, removed and changed, with Only
// changes, and the tree. Give it a size, or grow it in its parent.
//
// A click on a row's arrow, or Right and Left while the tree has the
// keyboard focus, opens and closes it, with Option (Alt) opening and
// closing all inside, as ui.Outline does. N and Shift+N choose the next
// and previous change, opening the rows around it, and go around past the
// last and first.
func View(c *ui.Context, s *State, d *Diff) ui.Element {
	t := c.Theme()
	if s.diff != d {
		*s = State{diff: d, selected: -1, only: s.only}
		var open func(n *node)
		open = func(n *node) {
			if n.op == same && n.inner != nil {
				s.outline.Open.Add(n)
				for _, k := range n.inner {
					open(k)
				}
			}
		}
		if d.root != nil {
			open(d.root)
		}
	}
	if d.errs != nil {
		return failures(c, d.errs)
	}
	s.outline.List.Selected = &s.selected
	palette := lightColors
	if t.Dark {
		palette = darkColors
	}
	palette.muted = t.TextMuted
	return ui.Column(c).Children(func() {
		header(c, s)
		ui.Divider(c)
		tree := ui.Outline(c, &s.outline, s.kids(d.root), s.kids, func(n *node) {
			colors := palette
			chosen := s.chosen() == n
			// Colors of types are unreadable on the accent of the row chosen.
			if chosen {
				a := t.AccentText
				colors = syntax{name: a, str: a, number: a, keyword: a, muted: a}
			}
			row(c, n, colors, chosen)
		}).Font("monospace").FontSize(t.FontSize - 1).Grow(1)
		if tree.Shortcut(0, ui.KeyN) {
			s.step(true)
		}
		if tree.Shortcut(ui.Shift, ui.KeyN) {
			s.step(false)
		}
	})
}

// header shows the counts of the changes and Only changes.
func header(c *ui.Context, s *State) {
	t := c.Theme()
	var count [4]int
	for _, n := range s.diff.changes {
		count[n.op]++
	}
	ui.Row(c).Gap(t.Space(3)).Padding(t.Space(1.5), t.Space(3)).AlignItems(ui.Center).Children(func() {
		if len(s.diff.changes) == 0 {
			ui.Text(c, "No differences").TextColor(t.TextMuted)
		}
		for _, k := range []struct {
			op   op
			noun string
		}{{added, "added"}, {removed, "removed"}, {changed, "changed"}} {
			if count[k.op] > 0 {
				ui.Row(c).Gap(t.Space(1)).AlignItems(ui.Center).Children(func() {
					ui.Text(c, mark(k.op)+strconv.Itoa(count[k.op])).Bold().TextColor(opColor(t, k.op)).FontFeatures("tnum")
					ui.Text(c, k.noun).TextColor(t.TextMuted)
				})
			}
		}
		ui.Box(c).Grow(1)
		ui.Toggle(c, &s.only, "Only changes")
	})
}

// row shows n: a mark of how it changed, its name or index and its value,
// or its old and new values, on the colors of what was removed and added
// unless chosen.
func row(c *ui.Context, n *node, colors syntax, chosen bool) {
	t := c.Theme()
	pad := t.Space(1)
	// Rows not on a tint start as far in as those on one.
	part := func(o op, build func()) {
		r := ui.Row(c).Gap(t.Space(1)).AlignItems(ui.Center).Shrink(1).PaddingStart(pad)
		if o != same {
			r.PaddingEnd(pad).Radius(pad)
			if !chosen {
				r.Background(opColor(t, o).Alpha(0.16))
			}
		}
		r.Children(build)
	}
	label := func() {
		if n.parent == nil {
			return
		}
		name, color := n.name, colors.name
		if n.index >= 0 {
			name, color = strconv.Itoa(n.index), colors.muted
		}
		ui.Row(c).Children(func() {
			ui.Text(c, name).TextColor(color).NoWrap()
			ui.Text(c, ":").TextColor(colors.muted)
		})
	}
	ui.Row(c).Gap(t.Space(1)).AlignItems(ui.Center).Shrink(1).Children(func() {
		m := ui.Text(c, mark(n.op)).Bold().Width(t.Space(2)).Shrink(0)
		if !chosen {
			m.TextColor(opColor(t, n.op))
		} else {
			m.TextColor(colors.muted)
		}
		switch n.op {
		case same:
			part(same, func() { label(); show(c, n.new, colors) })
		case added:
			part(added, func() { label(); show(c, n.new, colors) })
		case removed:
			part(removed, func() { label(); show(c, n.old, colors) })
		case changed:
			part(same, label)
			part(removed, func() { show(c, n.old, colors) })
			ui.Text(c, "→").TextColor(colors.muted)
			part(added, func() { show(c, n.new, colors) })
		}
	})
}

// show shows v: a scalar in the color of its type, an object or array as
// how many it holds.
func show(c *ui.Context, v *value, colors syntax) {
	switch v.kind {
	case object, array:
		count, open, end := len(v.kids), "{", "}"
		if v.kind == array {
			open, end = "[", "]"
		}
		label := strconv.Itoa(count) + " keys"
		switch {
		case v.kind == array && count == 1:
			label = "1 item"
		case v.kind == array:
			label = strconv.Itoa(count) + " items"
		case count == 1:
			label = "1 key"
		}
		ui.Text(c, open+label+end).TextColor(colors.muted).NoWrap()
	case str:
		q, _ := json.Marshal(v.text)
		ui.Text(c, string(q)).TextColor(colors.str).SingleLine().Shrink(1)
	case number:
		ui.Text(c, v.text).TextColor(colors.number).NoWrap()
	default:
		ui.Text(c, v.text).TextColor(colors.keyword).NoWrap()
	}
}

// mark returns the sign of o, a space for a row the same.
func mark(o op) string {
	return [...]string{" ", "+", "−", "~"}[o]
}

func opColor(t *ui.Theme, o op) ui.Color {
	return [...]ui.Color{t.TextMuted, t.Success, t.Danger, t.Warning}[o]
}

// failures shows why documents did not parse: which, the line and column,
// the message, and the line with a caret under the column.
func failures(c *ui.Context, errs []*parseError) ui.Element {
	t := c.Theme()
	return ui.Column(c).Gap(t.Space(5)).Padding(t.Space(4)).Children(func() {
		for _, e := range errs {
			ui.Column(c).Gap(t.Space(1.5)).Children(func() {
				ui.Text(c, fmt.Sprintf("%s document, line %d, column %d", e.doc, e.line, e.col)).Bold().TextColor(t.Danger)
				ui.Text(c, e.msg)
				ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Start).Padding(t.Space(2), t.Space(3)).Radius(t.Space(1.5)).Background(t.Text.Alpha(0.05)).
					Font("monospace").FontSize(t.FontSize - 1).Children(func() {
					ui.Text(c, strconv.Itoa(e.line)).TextColor(t.TextMuted)
					ui.Column(c).Shrink(1).Children(func() {
						ui.Text(c, e.text).NoWrap()
						ui.Text(c, strings.Repeat(" ", e.at)+"^").Bold().TextColor(t.Danger).NoWrap()
					})
				})
			})
		}
	})
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
