// Package flamegraph shows where a program spends its time, as profilers
// show a profile: each function a bar as wide as its share of the
// samples, the functions it called in bars below it, the root at the top,
// in warm colors of their names. A click on a bar zooms to it, a click on
// one above it or Escape zooms back out, and a search lights the
// functions matching it, with their share of the samples.
//
//	root, err := flamegraph.ParseFolded(f) // once, not every frame
//	...
//	ui.SearchField(c, &app.query)
//	flamegraph.View(c, &app.flame, root, app.query).Grow(1)
package flamegraph

import (
	"bufio"
	"fmt"
	"hash/fnv"
	"io"
	"slices"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
)

// Node is a function in a stack: its name, the samples in it and the
// functions it called, and those it called, its children.
type Node struct {
	Name     string
	Value    int64
	Children []*Node
}

// ParseFolded reads stacks folded a line each, as Brendan Gregg's
// stackcollapse scripts and many profilers write them: the frames from
// the root, separated by semicolons, a space and the samples,
// "main;run;parse 42". It returns the root, "all", the children of each
// node sorted by name.
func ParseFolded(r io.Reader) (*Node, error) {
	root := &Node{Name: "all"}
	sc := bufio.NewScanner(r)
	sc.Buffer(nil, 1<<20)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		i := strings.LastIndexByte(line, ' ')
		if i < 0 {
			continue
		}
		n, err := strconv.ParseInt(line[i+1:], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("flamegraph: %q: %w", line, err)
		}
		root.Value += n
		at := root
		for name := range strings.SplitSeq(line[:i], ";") {
			k := slices.IndexFunc(at.Children, func(c *Node) bool { return c.Name == name })
			if k < 0 {
				at.Children = append(at.Children, &Node{Name: name})
				k = len(at.Children) - 1
			}
			at = at.Children[k]
			at.Value += n
		}
	}
	var sortAll func(n *Node)
	sortAll = func(n *Node) {
		slices.SortFunc(n.Children, func(a, b *Node) int { return strings.Compare(a.Name, b.Name) })
		for _, c := range n.Children {
			sortAll(c)
		}
	}
	sortAll(root)
	return root, sc.Err()
}

// self returns the samples in n itself, not in the functions it called.
func (n *Node) self() int64 {
	v := n.Value
	for _, c := range n.Children {
		v -= c.Value
	}
	return v
}

// State is what a view keeps: the node it zoomed to, by the names from
// the root's child down, so that it holds as the app loads the profile
// again. Keep it in the app's state, one for each view.
type State struct {
	zoom []string
}

// path returns the nodes from root to the one zoomed to, root first,
// dropping the names past those root has.
func (s *State) path(root *Node) []*Node {
	path := []*Node{root}
	for i, name := range s.zoom {
		at := path[len(path)-1]
		k := slices.IndexFunc(at.Children, func(c *Node) bool { return c.Name == name })
		if k < 0 {
			s.zoom = s.zoom[:i]
			break
		}
		path = append(path, at.Children[k])
	}
	return path
}

// bar is a node laid out: its depth, and its left and width as fractions
// of the view.
type bar struct {
	node     *Node
	depth    int
	x, w     float64
	above    bool
	matching bool
}

// layout returns the bars of the view of path at least min of its width
// wide: the nodes of path full width, and the descendants of the last as
// wide as their share of it. matching reports whether a node matches the
// search.
func layout(path []*Node, min float64, matching func(*Node) bool) []bar {
	var bars []bar
	for d, n := range path[:len(path)-1] {
		bars = append(bars, bar{node: n, depth: d, w: 1, above: true, matching: matching(n)})
	}
	zoom := path[len(path)-1]
	var walk func(n *Node, depth int, x, w float64)
	walk = func(n *Node, depth int, x, w float64) {
		bars = append(bars, bar{node: n, depth: depth, x: x, w: w, matching: matching(n)})
		for _, c := range n.Children {
			cw := w * float64(c.Value) / float64(max(1, n.Value))
			if cw >= min {
				walk(c, depth+1, x, cw)
			}
			x += cw
		}
	}
	walk(zoom, len(path)-1, 0, 1)
	return bars
}

// View creates the view of the profile of root, lighting the functions
// whose names hold search, without regard to case, and a line below it of
// the function under the pointer, or the one zoomed to, or the share of
// the samples the search matches. Give it a height, or Grow; the bars
// scroll when they do not fit.
func View(c *ui.Context, s *State, root *Node, search string) ui.Element {
	t := c.Theme()
	rowH := t.FontSize + t.Space(2)
	search = strings.ToLower(search)
	matching := func(n *Node) bool { return search != "" && strings.Contains(strings.ToLower(n.Name), search) }
	path := s.path(root)
	zoom := path[len(path)-1]
	return ui.Column(c).Children(func() {
		var hovered *Node
		ui.Scroll(c).Grow(1).Children(func() {
			box := ui.Box(c).Focusable().Role(ui.RoleImage).Label("Flame graph of " + zoom.Name)
			w := max(1, box.LayoutBounds().W)
			bars := layout(path, 0.5/float64(w), matching)
			depth := 0
			for _, b := range bars {
				depth = max(depth, b.depth+1)
			}
			box.Height(float32(depth) * rowH)
			// at returns the bar at x, y in the box, if any.
			at := func(x, y float32) (bar, bool) {
				d := int(y / rowH)
				f := float64(x / w)
				for _, b := range bars {
					if b.depth == d && f >= b.x && f < b.x+b.w {
						return b, true
					}
				}
				return bar{}, false
			}
			if x, y, ok := box.PointerPosition(); ok {
				if b, ok := at(x, y); ok {
					hovered = b.node
				}
			}
			if box.Clicked() {
				if x, y, ok := box.PointerPosition(); ok {
					if b, ok := at(x, y); ok {
						s.zoomTo(path, b)
					}
				}
			}
			if box.Shortcut(0, ui.KeyEscape) && len(s.zoom) > 0 {
				s.zoom = s.zoom[:len(s.zoom)-1]
			}
			dark := t.Dark
			box.Draw(func(p *ui.Painter, r ui.Rect) {
				// The bars were laid out at the width of the last frame.
				if r.W != w {
					c.Invalidate()
				}
				vis := p.Visible()
				for _, b := range bars {
					y := r.Y + float32(b.depth)*rowH
					if y > vis.Y+vis.H || y+rowH < vis.Y {
						continue
					}
					rect := ui.Rect{X: r.X + float32(b.x)*r.W, Y: y, W: max(1, float32(b.w)*r.W-1), H: rowH - 1}
					fill := color(b.node.Name, dark)
					switch {
					case search != "" && b.matching:
						fill = ui.Oklch(0.7, 0.17, 310)
					case search != "":
						fill = t.TextMuted.Alpha(0.25)
					}
					if b.above {
						fill = fill.Alpha(0.45)
					}
					if b.node == hovered {
						fill = fill.Mix(t.Text, 0.15)
					}
					p.Fill(rect, fill, 2)
					if rect.W < 3*t.FontSize {
						continue
					}
					p.Clip(ui.Rect{X: rect.X + 4, Y: rect.Y, W: rect.W - 8, H: rect.H}, 0, func() {
						p.Text(rect.X+4, rect.Y+(rect.H-t.FontSize*1.2)/2, b.node.Name, t.FontSize-1, ui.RGB(0, 0, 0).Alpha(0.85))
					})
				}
			})
		})
		ui.Divider(c)
		ui.Text(c, status(root, zoom, hovered, search)).FontSize(t.FontSize-1).TextColor(t.TextMuted).
			FontFeatures("tnum").SingleLine().Padding(t.Space(1.5), t.Space(3))
	})
}

// zoomTo zooms to b, along path: to a node above the one zoomed to, out
// to it, and to one below, in.
func (s *State) zoomTo(path []*Node, b bar) {
	if b.above {
		s.zoom = s.zoom[:b.depth]
		return
	}
	var names []string
	var find func(n *Node) bool
	find = func(n *Node) bool {
		if n == b.node {
			return true
		}
		for _, c := range n.Children {
			names = append(names, c.Name)
			if find(c) {
				return true
			}
			names = names[:len(names)-1]
		}
		return false
	}
	if find(path[len(path)-1]) {
		s.zoom = append(s.zoom, names...)
	}
}

// status returns the line under the view: of hovered, or with a search,
// the share of the samples it matches, or of zoom.
func status(root, zoom, hovered *Node, search string) string {
	share := func(v int64) string {
		return fmt.Sprintf("%.1f%%", 100*float64(v)/float64(max(1, root.Value)))
	}
	describe := func(n *Node) string {
		return fmt.Sprintf("%s  ·  %s samples, %s  ·  self %s", n.Name, count(n.Value), share(n.Value), count(n.self()))
	}
	switch {
	case hovered != nil:
		return describe(hovered)
	case search != "":
		var v int64
		var walk func(n *Node)
		walk = func(n *Node) {
			if strings.Contains(strings.ToLower(n.Name), search) {
				v += n.Value
				return
			}
			for _, c := range n.Children {
				walk(c)
			}
		}
		walk(root)
		return fmt.Sprintf("Matched %s samples, %s", count(v), share(v))
	}
	return describe(zoom)
}

// count formats n with thousands separated, "12,345".
func count(n int64) string {
	s := strconv.FormatInt(n, 10)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}

// color returns the color of a function's bar, warm, from red to yellow,
// by its name, so that it keeps it from view to view.
func color(name string, dark bool) ui.Color {
	h := fnv.New32a()
	h.Write([]byte(name))
	v := h.Sum32()
	l := float32(0.78)
	if dark {
		l = 0.7
	}
	return ui.Oklch(l-float32(v>>8%8)/100, 0.14, 30+float32(v%55))
}
