// Package sunburst shows a tree of sizes as rings around a disc, as disk
// usage apps show a folder: the item shown in the middle, the items in it
// in the ring around it, each an arc as wide as its share of it, and
// their own items in the rings beyond, four rings at most. Each arc takes
// the hue of the arc of the first ring it is in, lighter further out, and is
// named where its name fits. Hovering an item lights it and the items it
// is in, and the middle tells its path, size and share. A click on an
// item with items in it zooms in, the rings moving until it is the
// middle and its items show around it, and a click on the middle zooms
// back out; while the desktop asks for less motion, the zoom is at once.
//
//	sunburst.View(c, &app.sunburst, app.root, humanBytes).Grow(1)
package sunburst

import (
	"cmp"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
)

// Node is an item of a tree: its name, its size, of it and the items in
// it together, and those items, its children.
type Node struct {
	Name     string
	Size     int64
	Children []*Node
}

// State is what a view keeps: the item in the middle, by the names of the
// items on the way from the root, so that it holds as the app builds the
// tree again, and the zoom on its way to it. Keep it in the app's state,
// one for each view.
type State struct {
	path     []string
	from, to window
	start    time.Time
}

// window is what the view shows: the part of the root's turn from a0 to
// a1 all around, and the depth below the root of the item in the middle.
type window struct{ a0, a1, depth float64 }

// rings is how many rings show around the middle, gap the room between
// arcs, and zoom how long a zoom takes.
const (
	rings = 4
	gap   = 1.5
	zoom  = 600 * time.Millisecond
)

// span is an item with its part of the root's turn.
type span struct {
	node   *Node
	a0, a1 float64
}

// split returns the items of n with a size, the largest first, each with
// its share of n's part of the turn, from a0 to a1.
func split(n *Node, a0, a1 float64) []span {
	kids := slices.DeleteFunc(slices.Clone(n.Children), func(k *Node) bool { return k.Size <= 0 })
	slices.SortStableFunc(kids, func(a, b *Node) int { return cmp.Compare(b.Size, a.Size) })
	var sum int64
	for _, k := range kids {
		sum += k.Size
	}
	total := max(n.Size, sum)
	out := make([]span, len(kids))
	at := a0
	for i, k := range kids {
		w := (a1 - a0) * float64(k.Size) / float64(total)
		out[i] = span{k, at, at + w}
		at += w
	}
	return out
}

// target returns the window of the item in the middle and the items from
// root to it, root first, dropping the names past those root has.
func (s *State) target(root *Node) (window, []*Node) {
	nodes := []*Node{root}
	w := window{0, 1, 0}
	for i, name := range s.path {
		kids := split(nodes[len(nodes)-1], w.a0, w.a1)
		k := slices.IndexFunc(kids, func(k span) bool { return k.node.Name == name && len(k.node.Children) > 0 })
		if k < 0 {
			s.path = s.path[:i]
			break
		}
		nodes = append(nodes, kids[k].node)
		w = window{kids[k].a0, kids[k].a1, float64(i + 1)}
	}
	return w, nodes
}

// shown returns the window to show in this frame, on the way to target,
// or at it at once while the desktop asks for less motion.
func (s *State) shown(c *ui.Context, target window) window {
	now := c.Now()
	if s.start.IsZero() || c.Preferences().ReduceMotion {
		s.from, s.to, s.start = target, target, now
	}
	if target != s.to {
		s.from, s.to, s.start = s.at(now), target, now
	}
	if now.Sub(s.start) < zoom {
		c.AnimationFrame()
	} else {
		s.from = s.to
	}
	return s.at(now)
}

// at returns the window at now, eased from the one the zoom started at to
// the one it goes to.
func (s *State) at(now time.Time) window {
	f := float64(now.Sub(s.start)) / float64(zoom)
	if f >= 1 {
		return s.to
	}
	e := float64(ui.EaseInOut(float32(max(0, f))))
	lerp := func(a, b float64) float64 { return a + (b-a)*e }
	return window{lerp(s.from.a0, s.to.a0), lerp(s.from.a1, s.to.a1), lerp(s.from.depth, s.to.depth)}
}

// arc is an item laid out: its part of the root's turn, its depth below
// the root, the index of the arc it is in, -1 for the root, and that of
// the item it is in at the depth below the middle, which gives its hue,
// -1 for the middle and those it is in.
type arc struct {
	node   *Node
	a0, a1 float64
	depth  int
	parent int
	branch int
}

// layout returns the arcs of root's items that show in w, those it is in
// before them, leaving out those narrower than least of the turn shown,
// their hues numbered among the items of those at depth below the root.
func layout(root *Node, w window, least float64, depth int) []arc {
	arcs := []arc{{node: root, a1: 1, parent: -1, branch: -1}}
	least *= w.a1 - w.a0
	for i := 0; i < len(arcs); i++ {
		a := arcs[i]
		if float64(a.depth)-w.depth >= rings {
			continue
		}
		for k, kid := range split(a.node, a.a0, a.a1) {
			if min(kid.a1, w.a1)-max(kid.a0, w.a0) < least {
				continue
			}
			branch := a.branch
			if a.depth == depth {
				branch = k
			}
			arcs = append(arcs, arc{kid.node, kid.a0, kid.a1, a.depth + 1, i, branch})
		}
	}
	return arcs
}

// geometry is the size of a view: its middle, the radius of the disc in
// it, how wide each ring around is, and the radius of the outermost.
type geometry struct {
	cx, cy, disc, ring, outer float64
}

// radius returns how far from the middle ring x starts, the disc being
// ring 0, so that an arc between rings is between their radii.
func (g geometry) radius(x float64) float64 {
	if x <= 1 {
		return max(0, x) * g.disc
	}
	return min(g.outer, g.disc+(x-1)*g.ring)
}

// hit returns the index of the arc at x, y from the middle, or -1.
func hit(arcs []arc, w window, g geometry, x, y float64) int {
	rho := math.Hypot(x, y)
	a := math.Atan2(x, -y) / (2 * math.Pi)
	if a < 0 {
		a++
	}
	a = w.a0 + a*(w.a1-w.a0)
	for i := len(arcs) - 1; i >= 0; i-- {
		ring := float64(arcs[i].depth) - w.depth
		if ring <= rings && rho >= g.radius(ring) && rho < g.radius(ring+1) && a >= arcs[i].a0 && a < arcs[i].a1 {
			return i
		}
	}
	return -1
}

// View creates the sunburst of the item of root s shows in the middle,
// its sizes formatted with format, the numbers with none. Give it a size,
// or Grow.
func View(c *ui.Context, s *State, root *Node, format func(size int64) string) ui.Element {
	t := c.Theme()
	if format == nil {
		format = func(n int64) string { return strconv.FormatInt(n, 10) }
	}
	target, nodes := s.target(root)
	middle := nodes[len(nodes)-1]
	w := s.shown(c, target)
	box := ui.Box(c).Role(ui.RoleImage).Label("Sunburst of " + middle.Name)
	bounds := box.LayoutBounds()
	g := geometry{cx: float64(bounds.W) / 2, cy: float64(bounds.H) / 2}
	g.outer = max(0, min(g.cx, g.cy)-float64(t.Space(4)))
	g.ring = g.outer / (rings + 1.4)
	g.disc = 1.4 * g.ring
	// The hues are those around the shallower of the middles a zoom goes
	// between, which holds every arc it shows: they change as a zoom out
	// starts, or once a zoom in ends.
	arcs := layout(root, w, 1/(2*math.Pi*max(1, g.outer)), int(min(s.from.depth, s.to.depth)))
	hovered := -1
	if x, y, ok := box.PointerPosition(); ok {
		hovered = hit(arcs, w, g, float64(x)-g.cx, float64(y)-g.cy)
	}
	if hovered >= 0 {
		h := arcs[hovered].node
		if h == middle && len(s.path) > 0 || h != middle && len(h.Children) > 0 {
			box.Cursor(ui.CursorPointer)
		}
		if box.Clicked() {
			switch {
			case h == middle && len(s.path) > 0:
				s.path = s.path[:len(s.path)-1]
			case h != middle && len(h.Children) > 0:
				s.path = names(arcs, hovered)
			}
		}
	}
	lit := make([]bool, len(arcs))
	for i := hovered; i >= 0; i = arcs[i].parent {
		lit[i] = true
	}
	dim := hovered >= 0 && arcs[hovered].node != middle
	box.Children(func() {
		width := float32(1.5 * g.disc)
		ui.Column(c).Absolute().Left(float32(g.cx)-width/2).Top(float32(g.cy-0.7*g.disc)).
			Size(width, float32(1.4*g.disc)).Center().Gap(t.Space(0.5)).PassThrough().Children(func() {
			n, name, line := middle, middle.Name, fmt.Sprintf("%d items", len(middle.Children))
			if dim {
				n, name = arcs[hovered].node, path(arcs, hovered, middle)
				share := 100 * float64(n.Size) / float64(max(1, middle.Size))
				line = fmt.Sprintf("%.1f%% of %s", share, middle.Name)
			}
			ui.Text(c, name).Bold().TextAlign(ui.Center).MaxLines(2)
			ui.Text(c, format(n.Size)).FontSize(t.FontSize + 6).FontWeight(600).FontFeatures("tnum").SingleLine()
			ui.Text(c, line).FontSize(t.FontSize - 1).TextColor(t.TextMuted).FontFeatures("tnum").SingleLine()
		})
	})
	dark := t.Dark
	box.Draw(func(p *ui.Painter, r ui.Rect) {
		// The arcs were laid out at the size of the last frame.
		if r.W != bounds.W || r.H != bounds.H {
			c.Invalidate()
		}
		cx, cy := float64(r.X)+g.cx, float64(r.Y)+g.cy
		for i, a := range arcs {
			x := float64(a.depth) - w.depth
			r0, r1 := g.radius(x), g.radius(x+1)
			a0, a1 := (max(a.a0, w.a0)-w.a0)/(w.a1-w.a0), (min(a.a1, w.a1)-w.a0)/(w.a1-w.a0)
			if r1-r0 < gap || x > rings+1 {
				continue
			}
			shade, text := color(a.branch, x, dark), t.Text.Alpha(0.8)
			if dim && !lit[i] {
				shade, text = shade.Mix(t.Background, 0.55), text.Alpha(0.3)
			}
			if x > rings {
				shade, text = shade.Alpha(float32(rings+1-x)), text.Alpha(float32(rings+1-x))
			}
			fill(p, shade, cx, cy, r0, r1, a0, a1)
			if x >= 1 {
				label(p, a.node.Name, t.FontSize-2, text, cx, cy, r0, r1, a0, a1)
			}
		}
	})
	return box
}

// names returns the names of the items from the root's to that of arc i.
func names(arcs []arc, i int) []string {
	var out []string
	for ; arcs[i].parent >= 0; i = arcs[i].parent {
		out = append(out, arcs[i].node.Name)
	}
	slices.Reverse(out)
	return out
}

// path returns the path of the item of arc i from middle, "src/cmd/go".
func path(arcs []arc, i int, middle *Node) string {
	var out []string
	for ; i >= 0; i = arcs[i].parent {
		out = append(out, arcs[i].node.Name)
		if arcs[i].node == middle {
			break
		}
	}
	slices.Reverse(out)
	return strings.Join(out, "/")
}

// fill fills the arc from radius r0 to r1 and from a0 to a1 of the turn,
// clockwise from the top, less the gap around it, with c.
func fill(p *ui.Painter, c ui.Color, cx, cy, r0, r1, a0, a1 float64) {
	if r0 > 0 {
		r0 += gap / 2
	}
	r1 -= gap / 2
	t0, t1 := 2*math.Pi*a0-math.Pi/2, 2*math.Pi*a1-math.Pi/2
	full := a1-a0 > 1-1e-9
	// The sides keep the gap between them all along, so they meet where
	// the arc is as long as the gap is wide.
	inner := max(r0, gap/(t1-t0))
	d1, d0 := gap/2/r1, gap/2/inner
	if full {
		inner, d1, d0 = r0, 0, 0
	}
	if inner >= r1 {
		return
	}
	// The mask of a path takes the room of its bounds, and those of whole
	// rings fill what the window keeps for masks, leaving arcs out: pieces
	// of an eighth of a turn bound the arc closely, each a DIP into the
	// next so that no seam shows between them.
	n := int(math.Ceil((t1 - t0) / (math.Pi / 4)))
	step := (t1 - t0) / float64(n)
	for i := range n {
		o0, o1 := t0+float64(i)*step, t0+float64(i+1)*step+1/r1
		i0, i1 := o0, o1
		if i == 0 && !full {
			o0, i0 = t0+d1, min(t0+d0, o1)
		}
		if i == n-1 && !full {
			o1, i1 = t1-d1, max(t1-d0, i0)
		}
		var path ui.Path
		path.MoveTo(float32(cx+r1*math.Cos(o0)), float32(cy+r1*math.Sin(o0)))
		arcTo(&path, cx, cy, r1, o0, o1)
		path.LineTo(float32(cx+inner*math.Cos(i1)), float32(cy+inner*math.Sin(i1)))
		arcTo(&path, cx, cy, inner, i1, i0)
		path.Close()
		p.FillPath(&path, c)
	}
}

// arcTo adds the arc of radius r around cx, cy from angle from to to, in
// radians, as cubic Béziers of a quarter turn at most.
func arcTo(p *ui.Path, cx, cy, r, from, to float64) {
	n := math.Ceil(math.Abs(to-from) / (math.Pi / 2))
	if n == 0 {
		return
	}
	d := (to - from) / n
	k := 4.0 / 3 * math.Tan(d/4)
	for i := range int(n) {
		a, b := from+float64(i)*d, from+float64(i+1)*d
		ca, sa, cb, sb := math.Cos(a), math.Sin(a), math.Cos(b), math.Sin(b)
		p.CubeTo(float32(cx+r*(ca-k*sa)), float32(cy+r*(sa+k*ca)),
			float32(cx+r*(cb+k*sb)), float32(cy+r*(sb-k*cb)),
			float32(cx+r*cb), float32(cy+r*sb))
	}
}

// label draws name along the radius through the middle of the arc from r0
// to r1 and from a0 to a1, reading left to right on either side, cut
// short to fit, or not at all where a few letters do not fit.
func label(p *ui.Painter, name string, size float32, color ui.Color, cx, cy, r0, r1, a0, a1 float64) {
	mid := (r0 + r1) / 2
	if (a1-a0)*2*math.Pi*mid < float64(size)*1.4 {
		return
	}
	name = fit(p, name, size, float32(r1-r0)-8)
	if name == "" {
		return
	}
	span := ui.Span{Text: name, Size: size, Color: color}
	w, h := p.MeasureText(0, span)
	deg := float32(((a0+a1)/2)*360 - 90)
	x := float32(cx+mid) - w/2
	if deg > 90 {
		deg -= 180
		x = float32(cx-mid) - w/2
	}
	p.Transform(ui.Rotation(deg), float32(cx), float32(cy), func() {
		p.RichText(x, float32(cy)-h/2, 0, span)
	})
}

// fit returns name, cut short with an ellipsis to be width wide at most,
// or "" when three letters do not fit.
func fit(p *ui.Painter, name string, size, width float32) string {
	w, _ := p.MeasureText(0, ui.Span{Text: name, Size: size})
	if w <= width {
		return name
	}
	runes := []rune(name)
	for n := min(len(runes)-1, int(float32(len(runes))*width/w)); n >= 3; n-- {
		cut := string(runes[:n]) + "…"
		if w, _ := p.MeasureText(0, ui.Span{Text: cut, Size: size}); w <= width {
			return cut
		}
	}
	return ""
}

// color returns the fill of an arc at ring x of the item numbered branch
// around the middle: its hue, deepest in the first ring and lighter
// further out, and palest in the middle, gray for none.
func color(branch int, x float64, dark bool) ui.Color {
	l, ch := 0.72+0.05*(x-1), 0.13-0.02*(x-1)
	if x < 1 {
		l, ch = 0.94-0.22*max(0, x), 0.03+0.1*max(0, x)
	}
	h := float32(0)
	if branch < 0 {
		ch = 0
	} else {
		h = float32(int(230+float64(branch)*137.5) % 360)
	}
	if dark {
		l = 1.2 - l
	}
	return ui.Oklch(float32(l), float32(ch), h)
}
