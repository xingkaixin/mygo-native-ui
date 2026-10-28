// Package circlepack draws a tree as circles packed in circles, as d3's
// zoomable circle packing: each leaf a circle whose area is its value,
// each parent the circle holding its children, colored by the branch of
// the top it is in. The children of the circle in view are named where
// their names fit, and the way down to it shows in the top corner.
//
//	circlepack.View(c, &app.pack, app.tree).Grow(1)
//
// A click on a circle holding others zooms into it, and a click on the
// circle in view, outside it or Escape zooms back out. The pointer over a
// circle outlines it and shows its name and value.
package circlepack

import (
	"fmt"
	"math"
	"slices"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
)

// Node is a node of the tree: its name, and its value for a leaf, or its
// children; a parent's value is the sum of its children's.
type Node struct {
	Name  string
	Value float64
	Kids  []Node
}

func (n Node) total() float64 {
	if len(n.Kids) == 0 {
		return n.Value
	}
	sum := 0.0
	for _, k := range n.Kids {
		sum += k.total()
	}
	return sum
}

// State is what a view keeps: the circle zoomed into, by its path of
// child indices from the top. Keep it in the app's state, one for each
// view.
type State struct {
	focus []int
}

// circle is a node laid out: its center and radius, its depth and path,
// the index of its parent in the layout, and its branch, the top child it
// is in.
type circle struct {
	x, y, r float64
	node    *Node
	depth   int
	path    []int
	parent  int
	branch  int
	value   float64
}

// pack places circles of radii rs touching each other around the origin,
// the largest first, each where it is nearest the middle without
// overlapping, and returns their centers.
func pack(rs []float64) [][2]float64 {
	order := make([]int, len(rs))
	for i := range order {
		order[i] = i
	}
	slices.SortStableFunc(order, func(a, b int) int {
		switch {
		case rs[a] > rs[b]:
			return -1
		case rs[a] < rs[b]:
			return 1
		}
		return 0
	})
	at := make([][2]float64, len(rs))
	var placed []int
	free := func(x, y, r float64) bool {
		for _, j := range placed {
			if math.Hypot(x-at[j][0], y-at[j][1]) < r+rs[j]-1e-6 {
				return false
			}
		}
		return true
	}
	for n, i := range order {
		r := rs[i]
		switch n {
		case 0:
			at[i] = [2]float64{0, 0}
		case 1:
			j := order[0]
			at[i] = [2]float64{rs[j] + r, 0}
		default:
			best, bd := [2]float64{}, math.Inf(1)
			for a := 0; a < len(placed); a++ {
				for b := a + 1; b < len(placed); b++ {
					A, B := placed[a], placed[b]
					da, db := rs[A]+r, rs[B]+r
					dx, dy := at[B][0]-at[A][0], at[B][1]-at[A][1]
					d := math.Hypot(dx, dy)
					if d == 0 || d > da+db || d < math.Abs(da-db) {
						continue
					}
					l := (da*da - db*db + d*d) / (2 * d)
					h := math.Sqrt(max(da*da-l*l, 0))
					mx, my := at[A][0]+l*dx/d, at[A][1]+l*dy/d
					for _, s := range []float64{1, -1} {
						x, y := mx-s*h*dy/d, my+s*h*dx/d
						if dist := math.Hypot(x, y); dist < bd && free(x, y, r) {
							best, bd = [2]float64{x, y}, dist
						}
					}
				}
			}
			at[i] = best
		}
		placed = append(placed, i)
	}
	return at
}

// enclose returns the center of a small circle holding the circles at at
// of radii rs, stepping toward the point farthest from it ever less, as
// Bădoiu and Clarkson's approximation does.
func enclose(at [][2]float64, rs []float64) (x, y float64) {
	for _, p := range at {
		x += p[0] / float64(len(at))
		y += p[1] / float64(len(at))
	}
	for k := 1; k <= 400; k++ {
		far, fx, fy := -1.0, 0.0, 0.0
		for i, p := range at {
			dx, dy := p[0]-x, p[1]-y
			d := math.Hypot(dx, dy)
			ex, ey := p[0], p[1]
			if d > 0 {
				ex, ey = p[0]+dx/d*rs[i], p[1]+dy/d*rs[i]
			}
			if e := d + rs[i]; e > far {
				far, fx, fy = e, ex, ey
			}
		}
		x += (fx - x) / float64(k+1)
		y += (fy - y) / float64(k+1)
	}
	return x, y
}

// layout lays out the tree in circles, the top one of radius 1 at the
// origin.
func layout(root *Node) []circle {
	type local struct{ x, y, r float64 }
	var size func(n *Node) float64
	offsets := map[*Node][]local{}
	size = func(n *Node) float64 {
		if len(n.Kids) == 0 {
			return math.Sqrt(max(n.Value, 0))
		}
		rs := make([]float64, len(n.Kids))
		for i := range n.Kids {
			rs[i] = size(&n.Kids[i])
		}
		at := pack(rs)
		cx, cy := enclose(at, rs)
		R := 0.0
		for i, p := range at {
			R = max(R, math.Hypot(p[0]-cx, p[1]-cy)+rs[i])
		}
		ls := make([]local, len(at))
		for i, p := range at {
			ls[i] = local{p[0] - cx, p[1] - cy, rs[i]}
		}
		offsets[n] = ls
		return R * 1.04
	}
	R := size(root)
	if R == 0 {
		return nil
	}
	out := []circle{{x: 0, y: 0, r: 1, node: root, parent: -1, branch: -1, value: root.total()}}
	for i := 0; i < len(out); i++ {
		c := out[i]
		for k, l := range offsets[c.node] {
			kid := &c.node.Kids[k]
			branch := c.branch
			if c.depth == 0 {
				branch = k
			}
			out = append(out, circle{
				x: c.x + l.x/R, y: c.y + l.y/R, r: l.r / R,
				node: kid, depth: c.depth + 1, path: append(slices.Clone(c.path), k),
				parent: i, branch: branch, value: kid.total(),
			})
		}
	}
	return out
}

var palette = []ui.Color{
	ui.Hex("#4e79a7"), ui.Hex("#f28e2b"), ui.Hex("#59a14f"), ui.Hex("#e15759"),
	ui.Hex("#b07aa1"), ui.Hex("#76b7b2"), ui.Hex("#edc948"), ui.Hex("#9c755f"),
}

// View creates the view of the tree under root. Give it a size, or Grow.
func View(c *ui.Context, s *State, root Node) ui.Element {
	t := c.Theme()
	circles := layout(&root)
	view := ui.Box(c).Focusable().Label(root.Name)
	if len(circles) == 0 {
		return view
	}
	focus := 0
	for i, ci := range circles {
		if slices.Equal(ci.path, s.focus) {
			focus = i
		}
	}
	s.focus = circles[focus].path
	b := view.Bounds()
	side := float64(min(b.W, b.H))/2 - 8
	f := circles[focus]
	zoom := 200 * time.Millisecond
	k := float64(view.Animate("circlepack.k", float32(side/f.r), zoom))
	fx := float64(view.Animate("circlepack.x", float32(f.x), zoom))
	fy := float64(view.Animate("circlepack.y", float32(f.y), zoom))
	cx, cy := float64(b.W)/2, float64(b.H)/2
	screen := func(ci circle) (x, y, r float64) {
		return cx + (ci.x-fx)*k, cy + (ci.y-fy)*k, ci.r * k
	}

	hovered := -1
	if px, py, ok := view.PointerPosition(); ok {
		for i, ci := range circles {
			x, y, r := screen(ci)
			if math.Hypot(float64(px)-x, float64(py)-y) <= r {
				hovered = i
			}
		}
	}
	if hovered >= 0 {
		h := circles[hovered]
		view.Tooltip(fmt.Sprintf("%s: %s", h.node.Name, format(h.value)))
	}
	out := func() {
		if focus > 0 {
			s.focus = circles[circles[focus].parent].path
		}
	}
	if view.Clicked() {
		view.Focus()
		target := hovered
		for target >= 0 && len(circles[target].node.Kids) == 0 {
			target = circles[target].parent
		}
		switch {
		case target < 0, target == focus:
			out()
		default:
			// Zooming goes one level at a time, into the child of the
			// circle in view that holds the one clicked.
			for target >= 0 && circles[target].parent != focus && target != focus {
				target = circles[target].parent
			}
			if target > 0 && circles[target].parent == focus {
				s.focus = circles[target].path
			} else {
				out()
			}
		}
	}
	if view.Shortcut(0, ui.KeyEscape) {
		out()
	}

	var trail []string
	for i := focus; i >= 0; i = circles[i].parent {
		trail = append([]string{circles[i].node.Name}, trail...)
	}
	draw := func(p *ui.Painter, r ui.Rect) {
		ox, oy := float64(r.X), float64(r.Y)
		for i, ci := range circles {
			x, y, rad := screen(ci)
			if rad < 0.5 {
				continue
			}
			var path ui.Path
			path.Circle(float32(ox+x), float32(oy+y), float32(rad))
			color := t.Text.Alpha(0.05)
			if ci.branch >= 0 {
				base := palette[ci.branch%len(palette)]
				color = base.Alpha(0.14)
				if len(ci.node.Kids) == 0 {
					color = base.Alpha(0.7)
				}
			}
			p.FillPath(&path, color)
			if i == hovered {
				p.StrokePath(&path, 1.5, t.Text)
			}
		}
		for _, ci := range circles {
			if ci.parent != focus {
				continue
			}
			x, y, rad := screen(ci)
			name := ui.Span{Text: ci.node.Name, Size: t.FontSize - 1, Weight: 600, Color: t.Text}
			w, h := p.MeasureText(0, name)
			if float64(w) > rad*1.8 || float64(h)*2 > rad*1.6 {
				continue
			}
			p.RichText(float32(ox+x)-w/2, float32(oy+y)-h, 0, name)
			value := ui.Span{Text: format(ci.value), Size: t.FontSize - 2, Color: t.TextMuted}
			vw, _ := p.MeasureText(0, value)
			p.RichText(float32(ox+x)-vw/2, float32(oy+y), 0, value)
		}
		label := ui.Span{Text: strings.Join(trail, " / "), Size: t.FontSize - 1, Weight: 500, Color: t.TextMuted}
		p.RichText(r.X+8, r.Y+8, 0, label)
	}
	view.Draw(func(p *ui.Painter, r ui.Rect) { p.Clip(r, 0, func() { draw(p, r) }) })
	return view
}

// format writes v briefly: 1.2M, 34.5K, 120.
func format(v float64) string {
	switch {
	case v >= 1e6:
		return fmt.Sprintf("%.1fM", v/1e6)
	case v >= 1e4:
		return fmt.Sprintf("%.0fK", v/1e3)
	case v >= 1e3:
		return fmt.Sprintf("%.1fK", v/1e3)
	}
	return fmt.Sprintf("%.0f", v)
}
