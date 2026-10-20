// Package treemap shows a tree of sizes as rectangles in rectangles, as
// disk usage apps show a folder: each item of the folder shown a
// rectangle of an area its share of the folder, kept near square, with its
// name and size, and the items in it drawn inside it. A click on an item
// with items in it goes into it, the path above goes back up, as do
// Escape and Backspace, and a line below tells of the item under the
// pointer.
//
//	treemap.View(c, &app.tree, app.root, humanBytes).Grow(1)
package treemap

import (
	"cmp"
	"fmt"
	"slices"
	"strconv"

	"github.com/egoist/mygo/ui"
)

// Node is an item of a tree: its name, its size, of it and the items in
// it together, and those items, its children.
type Node struct {
	Name     string
	Size     int64
	Children []*Node
}

// State is what a view keeps: the item it shows, by the names of the
// items on the way from the root, so that it holds as the app builds the
// tree again. Keep it in the app's state, one for each view.
type State struct {
	path []string
}

// nodes returns the nodes from root to the one shown, root first,
// dropping the names past those root has.
func (s *State) nodes(root *Node) []*Node {
	nodes := []*Node{root}
	for i, name := range s.path {
		at := nodes[len(nodes)-1]
		k := slices.IndexFunc(at.Children, func(c *Node) bool { return c.Name == name })
		if k < 0 || len(at.Children[k].Children) == 0 {
			s.path = s.path[:i]
			break
		}
		nodes = append(nodes, at.Children[k])
	}
	return nodes
}

// tile is a node laid out: its rectangle in the view, the index of the
// tile it is in, -1 for the items of the node shown, and the hue of that
// one.
type tile struct {
	node   *Node
	r      ui.Rect
	parent int
	hue    float32
}

// header is how tall the strip of a tile's name is, above the tiles of
// its children, and gap the room around tiles.
const (
	header = 18
	gap    = 2
)

// View creates the view of the item of root s shows, its sizes formatted
// with format, the numbers with none. Give it a size, or Grow.
func View(c *ui.Context, s *State, root *Node, format func(size int64) string) ui.Element {
	t := c.Theme()
	if format == nil {
		format = func(n int64) string { return strconv.FormatInt(n, 10) }
	}
	nodes := s.nodes(root)
	shown := nodes[len(nodes)-1]
	return ui.Column(c).Children(func() {
		names := make([]string, len(nodes))
		for i, n := range nodes {
			names[i] = n.Name
		}
		chosen := len(names) - 1
		if ui.Breadcrumbs(c, names, &chosen).Label("Path").Padding(t.Space(2), t.Space(3)).Changed() {
			s.path = s.path[:chosen]
		}
		box := ui.Box(c).Grow(1).Clip().Focusable().Role(ui.RoleImage).Label("Treemap of " + shown.Name)
		bounds := box.LayoutBounds()
		tiles := layout(shown, ui.Rect{W: bounds.W, H: bounds.H})
		hovered := -1
		if x, y, ok := box.PointerPosition(); ok {
			for i := len(tiles) - 1; i >= 0 && hovered < 0; i-- {
				if tiles[i].r.Contains(x, y) {
					hovered = i
				}
			}
		}
		if box.Clicked() && hovered >= 0 {
			s.enter(tiles, hovered)
		}
		if (box.Shortcut(0, ui.KeyEscape) || box.Shortcut(0, ui.KeyBackspace)) && len(s.path) > 0 {
			s.path = s.path[:len(s.path)-1]
		}
		dark := t.Dark
		box.Draw(func(p *ui.Painter, r ui.Rect) {
			// The tiles were laid out at the size of the last frame.
			if r.W != bounds.W || r.H != bounds.H {
				c.Invalidate()
			}
			for i, tl := range tiles {
				rect := ui.Rect{X: r.X + tl.r.X, Y: r.Y + tl.r.Y, W: tl.r.W, H: tl.r.H}
				fill, edge := colors(tl.hue, tl.parent >= 0, dark)
				if hovered >= 0 && (i == hovered || i == tiles[hovered].parent) {
					fill = fill.Mix(t.Text, 0.08)
				}
				p.Fill(rect, fill, 3)
				p.Stroke(rect, edge, 3, 1)
				label(p, t, tl, rect, format)
			}
		})
		line := fmt.Sprintf("%s  ·  %s  ·  %d items", shown.Name, format(shown.Size), len(shown.Children))
		if hovered >= 0 {
			line = describe(tiles, hovered, shown, format)
		}
		ui.Divider(c)
		ui.Text(c, line).FontSize(t.FontSize-1).TextColor(t.TextMuted).FontFeatures("tnum").SingleLine().
			Padding(t.Space(1.5), t.Space(3))
	})
}

// enter goes into the item of tile i, or of the tile it is in when it has
// no items itself.
func (s *State) enter(tiles []tile, i int) {
	var names []string
	for ; i >= 0; i = tiles[i].parent {
		if len(tiles[i].node.Children) > 0 {
			names = append(names, tiles[i].node.Name)
		}
	}
	slices.Reverse(names)
	s.path = append(s.path, names...)
}

// describe returns the line telling of the item of tile i: its name,
// after that of the tile it is in, its size and its share of shown.
func describe(tiles []tile, i int, shown *Node, format func(int64) string) string {
	tl := tiles[i]
	name := tl.node.Name
	if tl.parent >= 0 {
		name = tiles[tl.parent].node.Name + "/" + name
	}
	share := 100 * float64(tl.node.Size) / float64(max(1, shown.Size))
	line := fmt.Sprintf("%s  ·  %s  ·  %.1f%% of %s", name, format(tl.node.Size), share, shown.Name)
	if n := len(tl.node.Children); n > 0 {
		line += fmt.Sprintf("  ·  %d items", n)
	}
	return line
}

// layout returns the tiles of the items of n in r, and inside those with
// room, the tiles of their own items, each after the one it is in.
func layout(n *Node, r ui.Rect) []tile {
	var tiles []tile
	for i, rect := range squarify(n.Children, inset(r, gap)) {
		if rect.r.W < 1 || rect.r.H < 1 {
			continue
		}
		parent := len(tiles)
		tiles = append(tiles, tile{node: rect.node, r: rect.r, parent: -1, hue: hue(i)})
		inner := ui.Rect{X: rect.r.X, Y: rect.r.Y + header, W: rect.r.W, H: rect.r.H - header}
		if inner.W < 24 || inner.H < 12 {
			continue
		}
		for _, kid := range squarify(rect.node.Children, inset(inner, gap+1)) {
			if kid.r.W >= 2 && kid.r.H >= 2 {
				tiles = append(tiles, tile{node: kid.node, r: kid.r, parent: parent, hue: hue(i)})
			}
		}
	}
	return tiles
}

// placed is a node with its rectangle.
type placed struct {
	node *Node
	r    ui.Rect
}

// squarify lays nodes out in r, the largest first, each with an area of
// its share of their sizes, in rows that keep them as near square as they
// can, as Bruls, Huizing and van Wijk's squarified treemaps do.
func squarify(nodes []*Node, r ui.Rect) []placed {
	items := slices.Clone(nodes)
	items = slices.DeleteFunc(items, func(n *Node) bool { return n.Size <= 0 })
	slices.SortStableFunc(items, func(a, b *Node) int { return cmp.Compare(b.Size, a.Size) })
	var total float64
	for _, n := range items {
		total += float64(n.Size)
	}
	if total == 0 || r.W <= 0 || r.H <= 0 {
		return nil
	}
	area := make([]float64, len(items))
	for i, n := range items {
		area[i] = float64(n.Size) / total * float64(r.W) * float64(r.H)
	}
	out := make([]placed, 0, len(items))
	x, y, w, h := float64(r.X), float64(r.Y), float64(r.W), float64(r.H)
	for i := 0; i < len(items); {
		side := min(w, h)
		j, best := i+1, worst(area[i:i+1], side)
		for j < len(items) {
			if next := worst(area[i:j+1], side); next <= best {
				best, j = next, j+1
				continue
			}
			break
		}
		var sum float64
		for _, a := range area[i:j] {
			sum += a
		}
		if w >= h {
			cw := sum / h
			at := y
			for k := i; k < j; k++ {
				kh := area[k] / cw
				out = append(out, placed{items[k], rect(x, at, cw, kh)})
				at += kh
			}
			x, w = x+cw, w-cw
		} else {
			rh := sum / w
			at := x
			for k := i; k < j; k++ {
				kw := area[k] / rh
				out = append(out, placed{items[k], rect(at, y, kw, rh)})
				at += kw
			}
			y, h = y+rh, h-rh
		}
		i = j
	}
	return out
}

// worst returns the worst aspect ratio of a row of areas, the largest
// first, along a side side long.
func worst(row []float64, side float64) float64 {
	var sum float64
	for _, a := range row {
		sum += a
	}
	hi, lo := row[0], row[len(row)-1]
	return max(side*side*hi/(sum*sum), sum*sum/(side*side*lo))
}

// rect returns the rectangle, less the gap between tiles.
func rect(x, y, w, h float64) ui.Rect {
	return inset(ui.Rect{X: float32(x), Y: float32(y), W: float32(w), H: float32(h)}, gap/2)
}

func inset(r ui.Rect, d float32) ui.Rect {
	return ui.Rect{X: r.X + d, Y: r.Y + d, W: max(0, r.W-2*d), H: max(0, r.H-2*d)}
}

// hue returns the hue of the ith item, ever apart from those before it.
func hue(i int) float32 {
	return float32(int(230+float64(i)*137.5) % 360)
}

// colors returns the fill and the edge of a tile of hue, deeper for the
// tiles of a tile's items.
func colors(hue float32, inner, dark bool) (fill, edge ui.Color) {
	l, c := float32(0.9), float32(0.05)
	if inner {
		l, c = 0.82, 0.08
	}
	if dark {
		l = 1.2 - l
	}
	return ui.Oklch(l, c, hue), ui.Oklch(l-0.12, c+0.03, hue).Alpha(0.6)
}

// label draws the name and the size of tl's item in rect where it fits:
// above the tiles of its items, or in the middle of a tile in a tile.
func label(p *ui.Painter, t *ui.Theme, tl tile, rect ui.Rect, format func(int64) string) {
	size := t.FontSize - 2
	if tl.parent >= 0 {
		size = t.FontSize - 3
		if rect.W < 40 || rect.H < size*1.6 {
			return
		}
		p.Clip(inset(rect, 3), 0, func() {
			p.Text(rect.X+4, rect.Y+3, tl.node.Name, size, t.Text.Alpha(0.75))
		})
		return
	}
	if rect.W < 30 || rect.H < header {
		return
	}
	name := ui.Span{Text: tl.node.Name, Size: size, Weight: 600, Color: t.Text}
	amount := ui.Span{Text: "  " + format(tl.node.Size), Size: size, Color: t.TextMuted}
	p.Clip(ui.Rect{X: rect.X + 4, Y: rect.Y, W: rect.W - 8, H: header}, 0, func() {
		p.RichText(rect.X+5, rect.Y+(header-size*1.3)/2, 0, name, amount)
	})
}
