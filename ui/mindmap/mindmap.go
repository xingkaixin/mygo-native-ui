// Package mindmap shows an outline as a mind map, as XMind and MindNode
// do: the title in the middle, its branches to the right and the left,
// half the map on each side, each in its own color, and the items under
// each laid out as a tree away from it. The dot at the end of an item
// with items under it folds them, and shows how many there are while they
// are folded. A click chooses an item, which Chosen returns.
//
// The map is as big as its items; put it in a ScrollBoth.
//
//	ui.ScrollBoth(c).Grow(1).Children(func() {
//		mindmap.View(c, &app.mindmap, "Launch", app.items)
//	})
package mindmap

import (
	"fmt"

	"github.com/egoist/mygo/ui"
)

// Item is an item of the map: its text, and how deep it is, 0 for a
// branch of the title, one more than the item it is under.
type Item struct {
	Text  string
	Depth int
}

// State is what a map keeps: the items folded and the one chosen, by
// index. Keep it in the app's state, one for each map.
type State struct {
	closed map[int]bool
	// chosen is the index of the item chosen, plus 1, so that the zero
	// State chooses none.
	chosen int
}

// Chosen returns the index of the item chosen.
func (s *State) Chosen() (int, bool) { return s.chosen - 1, s.chosen > 0 }

type node struct {
	kids []int
	// w and h are the item's size, tall the height of the tree under it.
	w, h, tall float32
	x, y       float32
	left       bool
	branch     int
	shown      bool
}

// View creates the map of items under title.
func View(c *ui.Context, s *State, title string, items []Item) ui.Element {
	t := c.Theme()
	if s.closed == nil {
		s.closed = map[int]bool{}
	}
	gapX, gapY, pad := t.Space(10), t.Space(2), t.Space(8)
	padX, padY := t.Space(3), t.Space(1.5)
	// The title is node n, after the items.
	n := len(items)
	nodes := make([]node, n+1)
	var stack []int
	for i, it := range items {
		for len(stack) > 0 && items[stack[len(stack)-1]].Depth >= it.Depth {
			stack = stack[:len(stack)-1]
		}
		parent := n
		if len(stack) > 0 {
			parent = stack[len(stack)-1]
		}
		nodes[parent].kids = append(nodes[parent].kids, i)
		stack = append(stack, i)
	}
	size := func(i int) (float32, int) {
		if i == n {
			return t.FontSize + 6, 700
		}
		if items[i].Depth == 0 {
			return t.FontSize + 1, 600
		}
		return t.FontSize, 400
	}
	for i := range nodes {
		text := title
		if i < n {
			text = items[i].Text
		}
		fs, wt := size(i)
		w, h := c.MeasureText(0, ui.Span{Text: text, Size: fs, Weight: wt})
		px, py := padX, padY
		if i == n {
			px, py = t.Space(5), t.Space(3)
		}
		nodes[i].w, nodes[i].h = max(w, t.Space(4))+2*px, h+2*py
	}
	var measure func(i int) float32
	measure = func(i int) float32 {
		nd := &nodes[i]
		nd.tall = nd.h
		if i < n && s.closed[i] {
			return nd.tall
		}
		var sum float32
		for k, j := range nd.kids {
			if k > 0 {
				sum += gapY
			}
			sum += measure(j)
		}
		nd.tall = max(nd.h, sum)
		return nd.tall
	}
	root := &nodes[n]
	var right, left []int
	// The sides split by items, folded or not, so that folding a branch
	// leaves the others where they are.
	var leaves func(i int) int
	leaves = func(i int) int {
		k := 0
		for _, j := range nodes[i].kids {
			k += leaves(j)
		}
		return max(1, k)
	}
	total, acc := leaves(n), 0
	for k, j := range root.kids {
		measure(j)
		nodes[j].branch = k
		w := leaves(j)
		if len(left) == 0 && (k == 0 || 2*acc+w <= total) {
			right = append(right, j)
			acc += w
		} else {
			left = append(left, j)
		}
	}
	// place puts node i with its middle at y, its side toward the title
	// at x, and the tree under it beyond.
	var place func(i int, x, y float32, left bool, branch int)
	place = func(i int, x, y float32, left bool, branch int) {
		nd := &nodes[i]
		nd.left, nd.branch, nd.shown = left, branch, true
		nd.y = y - nd.h/2
		nd.x = x
		if left {
			nd.x = x - nd.w
		}
		if s.closed[i] {
			return
		}
		next := x + nd.w + gapX
		if left {
			next = x - nd.w - gapX
		}
		stackKids(nodes, nd.kids, next, y, gapY, left, branch, place)
	}
	root.shown = true
	root.x, root.y = -root.w/2, -root.h/2
	stackKids(nodes, right, root.w/2+gapX, 0, gapY, false, -1, place)
	stackKids(nodes, left, -root.w/2-gapX, 0, gapY, true, -1, place)
	minX, minY, maxX, maxY := root.x, root.y, root.x+root.w, root.y+root.h
	for _, nd := range nodes {
		if nd.shown {
			minX, minY = min(minX, nd.x), min(minY, nd.y)
			maxX, maxY = max(maxX, nd.x+nd.w), max(maxY, nd.y+nd.h)
		}
	}
	ox, oy := pad-minX, pad-minY
	box := ui.Box(c).Size(maxX-minX+2*pad, maxY-minY+2*pad)
	var under func(i int) int
	under = func(i int) int {
		k := 0
		for _, j := range nodes[i].kids {
			k += 1 + under(j)
		}
		return k
	}
	box.Draw(func(p *ui.Painter, r ui.Rect) {
		for i, nd := range nodes {
			if nd.shown && i < n && items[i].Depth > 0 {
				x, y := r.X+ox+nd.x, r.Y+oy+nd.y+nd.h
				p.Line(x, y, x+nd.w, y, 1.5, hue(nd.branch))
			}
			if !nd.shown || (i < n && s.closed[i]) {
				continue
			}
			for _, j := range nd.kids {
				kid := nodes[j]
				x0, x1 := nd.x+nd.w, kid.x
				if kid.left {
					x0, x1 = nd.x, kid.x+kid.w
				}
				y0, y1 := nd.y+nd.h/2, kid.y+kid.h/2
				if i < n && items[i].Depth > 0 {
					y0 = nd.y + nd.h
				}
				if items[j].Depth > 0 {
					y1 = kid.y + kid.h
				}
				x0, y0, x1, y1 = r.X+ox+x0, r.Y+oy+y0, r.X+ox+x1, r.Y+oy+y1
				dx := (x1 - x0) / 2
				var path ui.Path
				path.MoveTo(x0, y0).CubeTo(x0+dx, y0, x1-dx, y1, x1, y1)
				width := float32(2.5)
				if i < n {
					width = 1.5
				}
				p.StrokePath(&path, width, hue(kid.branch))
			}
		}
	})
	box.Children(func() {
		for i, nd := range nodes {
			if !nd.shown {
				continue
			}
			text := title
			if i < n {
				text = items[i].Text
			}
			fs, wt := size(i)
			el := ui.ButtonBase(c.Key(i)).Absolute().Left(ox+nd.x).Top(oy+nd.y).Size(nd.w, nd.h).
				Justify(ui.Center).AlignItems(ui.Center).Label(text)
			color := t.Text
			switch {
			case i == n:
				el.Background(t.Text).Radius(nd.h / 2)
				color = t.Background
			case items[i].Depth == 0:
				el.Background(hue(nd.branch).Alpha(0.16)).Border(1.5, hue(nd.branch)).Radius(t.Space(2))
			default:
				el.Radius(t.Space(1))
				if el.Hovered() {
					el.Background(t.SurfaceHover)
				}
			}
			if i < n {
				if s.chosen == i+1 {
					el.Border(2, t.Accent)
				}
				if el.Clicked() {
					s.chosen = i + 1
				}
			}
			el.Children(func() {
				ui.Text(c, text).FontSize(fs).FontWeight(wt).TextColor(color).SingleLine()
			})
			if i < n && len(nd.kids) > 0 {
				fold(c.Key(fmt.Sprint("fold", i)), s, i, nd, ox, oy, items[i].Depth == 0, under(i))
			}
		}
	})
	return box
}

// stackKids places kids one under another, the middle of them all at y.
func stackKids(nodes []node, kids []int, x, y, gapY float32, left bool, branch int, place func(int, float32, float32, bool, int)) {
	var total float32
	for k, j := range kids {
		if k > 0 {
			total += gapY
		}
		total += nodes[j].tall
	}
	top := y - total/2
	for _, j := range kids {
		b := branch
		if b < 0 {
			b = nodes[j].branch
		}
		place(j, x, top+nodes[j].tall/2, left, b)
		top += nodes[j].tall + gapY
	}
}

// fold shows the dot at the outer end of item i that folds the count
// items under it, with the count while they are folded.
func fold(c *ui.Context, s *State, i int, nd node, ox, oy float32, branch bool, count int) {
	t := c.Theme()
	closed := s.closed[i]
	d := t.Space(3.5)
	if closed {
		d = t.Space(5)
	}
	cx := ox + nd.x + nd.w + t.Space(1) + d/2
	if nd.left {
		cx = ox + nd.x - t.Space(1) - d/2
	}
	cy := oy + nd.y + nd.h
	if branch {
		cy = oy + nd.y + nd.h/2
	}
	b := ui.ButtonBase(c).Absolute().Left(cx-d/2).Top(cy-d/2).Size(d, d).Radius(d / 2).
		Justify(ui.Center).AlignItems(ui.Center).Label("Fold")
	if b.Clicked() {
		s.closed[i] = !closed
	}
	color := hue(nd.branch)
	if !closed {
		b.Background(t.Background).Border(1.5, color)
		return
	}
	b.Label("Unfold").Background(color).Children(func() {
		ui.Text(c, fmt.Sprint(count)).FontSize(t.FontSize - 3).Bold().TextColor(ui.RGB(255, 255, 255))
	})
}

// hue returns branch k's color, around the wheel by the golden angle so
// that branches beside one another differ.
func hue(k int) ui.Color {
	return ui.Oklch(0.64, 0.15, float32((20+k*137)%360))
}
