// Package packet shows a packet as Wireshark's details and bytes panes
// do: its protocols' fields in a tree, above its bytes in hex. Choosing a
// field selects the bytes it covers, and a click on a byte chooses the
// innermost field holding it, opening the fields round it. Dissect reads
// an Ethernet frame into fields, and sums it up for a list of packets:
//
//	fields, _ := packet.Dissect(frame)
//	packet.View(c, &app.packet, fields, frame).Grow(1)
package packet

import (
	"strconv"

	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/hexview"
)

// Field is a field of a packet: what it says, as "Source Port: 443", the
// bytes it covers, Length of them from Offset, and the fields within it.
type Field struct {
	Label          string
	Offset, Length int
	Fields         []Field
}

// State is what a view keeps: the fields open and the one chosen, the
// bytes selected and where the panes divide. Keep it in the app's state,
// one for each view; showing another packet with it keeps the fields open
// and chosen where it has them too, as Wireshark does.
type State struct {
	tree   ui.OutlineState[string]
	row    int
	hex    hexview.State
	split  float32
	synced string
	picked [2]int
}

// node is a field and the paths of the fields within it.
type node struct {
	field *Field
	kids  []string
}

// View creates the panes of a packet of data, whose fields say what its
// bytes are. Give it a size, or grow it in its parent.
func View(c *ui.Context, s *State, fields []Field, data []byte) ui.Element {
	t := c.Theme()
	nodes := map[string]node{}
	var walk func(prefix string, fs []Field) []string
	walk = func(prefix string, fs []Field) []string {
		if len(fs) == 0 {
			return nil
		}
		paths := make([]string, len(fs))
		for i := range fs {
			p := prefix + strconv.Itoa(i)
			paths[i] = p
			nodes[p] = node{&fs[i], walk(p+"/", fs[i].Fields)}
		}
		return paths
	}
	roots := walk("", fields)
	rows := s.visible(roots, nodes)
	if s.split == 0 {
		s.split = 50
	}

	if s.row >= 0 && s.row < len(rows) && rows[s.row] != s.synced {
		s.pick(rows[s.row], nodes)
	}

	return ui.SplitVerticalPercent(c, &s.split, func() {
		s.tree.List.Selected = &s.row
		ui.Outline(c, &s.tree, roots, func(p string) []string {
			return nodes[p].kids
		}, func(p string) {
			ui.Text(c, nodes[p].field.Label).SingleLine().Ellipsis("…")
		}).Grow(1).Label("Packet details")
	}, func() {
		hexview.View(c, &s.hex, data).Grow(1)
		start, end := s.hex.Selection()
		if [2]int{start, end} == s.picked || end-start != 1 {
			return
		}
		s.picked = [2]int{start, end}
		p := innermost(fields, "", start)
		if p == "" {
			return
		}
		for i := range p {
			if p[i] == '/' {
				s.tree.Open.Add(p[:i])
			}
		}
		for i, r := range s.visible(roots, nodes) {
			if r == p {
				s.row = i
				s.tree.List.ScrollIntoView(i)
			}
		}
		s.pick(p, nodes)
		// The tree came before the bytes in this frame.
		c.Invalidate()
	}).Background(t.Background)
}

// pick selects the bytes of the field at path p.
func (s *State) pick(p string, nodes map[string]node) {
	f := nodes[p].field
	s.synced = p
	s.hex.Select(f.Offset, f.Offset+max(1, f.Length))
	s.picked[0], s.picked[1] = s.hex.Selection()
}

// visible returns the paths of the rows the tree shows: the roots, and
// below each open field those within it.
func (s *State) visible(roots []string, nodes map[string]node) []string {
	var rows []string
	var add func(paths []string)
	add = func(paths []string) {
		for _, p := range paths {
			rows = append(rows, p)
			if s.tree.Open.Has(p) {
				add(nodes[p].kids)
			}
		}
	}
	add(roots)
	return rows
}

// innermost returns the path of the deepest field covering byte at, ""
// for none.
func innermost(fields []Field, prefix string, at int) string {
	for i, f := range fields {
		if at < f.Offset || at >= f.Offset+f.Length {
			continue
		}
		p := prefix + strconv.Itoa(i)
		if in := innermost(f.Fields, p+"/", at); in != "" {
			return in
		}
		return p
	}
	return ""
}
