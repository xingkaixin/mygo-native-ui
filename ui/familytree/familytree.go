// Package familytree draws a family's descendants, as genealogy charts
// do: a card for each person with their years, each married couple side
// by side joined by a line, and their children in a row below, joined to
// them by a bracket. Above the chart, whose descendants they are and how
// many. The chart takes the room the family needs; put it in a
// ui.ScrollBoth when it may not fit. Clicking a person reads out their
// life, children and grandchildren instead, and lights their line of
// descent from the eldest generation.
//
//	familytree.View(c, &app.tree, family)
package familytree

import (
	"fmt"
	"slices"
	"strings"

	"github.com/egoist/mygo/ui"
)

// Person is a member of the family.
type Person struct {
	Name string
	// Born and Died are years; Died is 0 for the living.
	Born, Died int
}

// Family is a descendant, the spouse they married, if any, and their
// children's families.
type Family struct {
	Person   Person
	Spouse   *Person
	Children []Family
}

// State is what a chart keeps: the person picked. Keep it in the app's
// state, one for each chart.
type State struct {
	// picked is the path of child indexes to the picked family, and
	// spouse whether its spouse is the one picked.
	picked []int
	spouse bool
	chosen bool
}

type node struct {
	f        *Family
	path     []int
	x, y     float32
	width    float32
	children []*node
}

// View creates the chart of the family descending from root.
func View(c *ui.Context, s *State, root Family) ui.Element {
	t := c.Theme()
	cardW, cardH := t.Space(30), t.Space(12)
	spouseGap, siblingGap, levelGap := t.Space(3), t.Space(4), t.Space(10)
	var build func(f *Family, path []int, depth int) *node
	build = func(f *Family, path []int, depth int) *node {
		n := &node{f: f, path: path, y: float32(depth) * (cardH + levelGap), width: cardW}
		if f.Spouse != nil {
			n.width = 2*cardW + spouseGap
		}
		kids := float32(0)
		for i := range f.Children {
			ch := build(&f.Children[i], append(append([]int{}, path...), i), depth+1)
			n.children = append(n.children, ch)
			if i > 0 {
				kids += siblingGap
			}
			kids += ch.width
		}
		n.width = max(n.width, kids)
		return n
	}
	tree := build(&root, nil, 0)
	var place func(n *node, left float32)
	place = func(n *node, left float32) {
		n.x = left
		kids := float32(0)
		for i, ch := range n.children {
			if i > 0 {
				kids += siblingGap
			}
			kids += ch.width
		}
		at := left + (n.width-kids)/2
		for _, ch := range n.children {
			place(ch, at)
			at += ch.width + siblingGap
		}
	}
	place(tree, 0)
	depth := 0
	var walk func(n *node, d int, fn func(n *node))
	walk = func(n *node, d int, fn func(n *node)) {
		depth = max(depth, d)
		fn(n)
		for _, ch := range n.children {
			walk(ch, d+1, fn)
		}
	}
	var nodes []*node
	walk(tree, 0, func(n *node) { nodes = append(nodes, n) })
	// cards returns where a node's cards go: the descendant's, centered
	// over the children when alone, and the spouse's to its right.
	cards := func(n *node) (self, spouse ui.Rect) {
		w := cardW
		if n.f.Spouse != nil {
			w = 2*cardW + spouseGap
		}
		x := n.x + (n.width-w)/2
		self = ui.Rect{X: x, Y: n.y, W: cardW, H: cardH}
		spouse = ui.Rect{X: x + cardW + spouseGap, Y: n.y, W: cardW, H: cardH}
		return self, spouse
	}
	lit := map[*node]bool{}
	var picked *node
	var pickedPerson *Person
	if s.chosen {
		for _, n := range nodes {
			if slices.Equal(n.path, s.picked) {
				picked = n
				pickedPerson = &n.f.Person
				if s.spouse && n.f.Spouse != nil {
					pickedPerson = n.f.Spouse
				}
			}
		}
		for _, n := range nodes {
			if len(n.path) <= len(s.picked) && slices.Equal(n.path, s.picked[:len(n.path)]) {
				lit[n] = true
			}
		}
	}
	view := ui.Column(c).Gap(t.Space(4)).AlignItems(ui.Start).Role(ui.RoleGroup).Label("Family tree")
	view.Children(func() {
		line := fmt.Sprintf("Descendants of %s · %d people", root.Person.Name, len(nodes)-1)
		if root.Spouse != nil {
			line = fmt.Sprintf("Descendants of %s and %s · %d people", root.Person.Name, root.Spouse.Name, len(nodes)-1)
		}
		if picked != nil {
			line = about(*pickedPerson, picked, s.spouse)
		}
		ui.Text(c, line).FontFeatures("tnum")
		chart := ui.Box(c).Size(tree.width, float32(depth+1)*(cardH+levelGap)-levelGap).Role(ui.RoleImage)
		if chart.Clicked() {
			if px, py, ok := chart.PointerPosition(); ok {
				s.chosen = false
				for _, n := range nodes {
					self, sp := cards(n)
					switch {
					case contains(self, px, py):
						s.picked, s.spouse, s.chosen = n.path, false, true
					case n.f.Spouse != nil && contains(sp, px, py):
						s.picked, s.spouse, s.chosen = n.path, true, true
					}
				}
			}
		}
		chart.Draw(func(p *ui.Painter, r ui.Rect) {
			off := func(b ui.Rect) ui.Rect { return ui.Rect{X: r.X + b.X, Y: r.Y + b.Y, W: b.W, H: b.H} }
			for _, n := range nodes {
				self, sp := cards(n)
				self, sp = off(self), off(sp)
				if n.f.Spouse != nil {
					p.StrokePolyline([][2]float32{{self.X + self.W, self.Y + self.H/2}, {sp.X, sp.Y + sp.H/2}}, 1.5, t.TextMuted)
				}
				if len(n.children) == 0 {
					continue
				}
				fromX := self.X + self.W/2
				if n.f.Spouse != nil {
					fromX = self.X + self.W + spouseGap/2
				}
				fromY := self.Y + self.H/2
				if n.f.Spouse == nil {
					fromY = self.Y + self.H
				}
				midY := self.Y + self.H + levelGap/2
				var xs []float32
				for _, ch := range n.children {
					cs, _ := cards(ch)
					xs = append(xs, r.X+cs.X+cs.W/2)
				}
				ink := t.TextMuted
				p.StrokePolyline([][2]float32{{fromX, fromY}, {fromX, midY}}, 1.5, ink)
				p.StrokePolyline([][2]float32{{min(fromX, xs[0]), midY}, {max(fromX, xs[len(xs)-1]), midY}}, 1.5, ink)
				for i, x := range xs {
					col, w := ink, float32(1.5)
					if lit[n.children[i]] {
						col, w = t.Accent, 2.5
						p.StrokePolyline([][2]float32{{fromX, fromY}, {fromX, midY}, {x, midY}}, w, col)
					}
					p.StrokePolyline([][2]float32{{x, midY}, {x, midY + levelGap/2}}, w, col)
				}
			}
			for _, n := range nodes {
				self, sp := cards(n)
				card(p, t, off(self), n.f.Person, lit[n], picked == n && !s.spouse)
				if n.f.Spouse != nil {
					card(p, t, off(sp), *n.f.Spouse, false, picked == n && s.spouse)
				}
			}
		})
	})
	return view
}

func card(p *ui.Painter, t *ui.Theme, b ui.Rect, who Person, lit, picked bool) {
	bg, border, bw := t.Surface, t.Border, float32(1)
	switch {
	case picked:
		bg, border, bw = t.Accent.Alpha(0.12), t.Accent, 2
	case lit:
		border, bw = t.Accent, 1.5
	}
	p.Fill(b, bg, t.Radius+2)
	p.Stroke(b, border, t.Radius+2, bw)
	name := ui.Span{Text: who.Name, Size: t.FontSize - 1, Weight: 600, Color: t.Text}
	years := ui.Span{Text: lifespan(who), Size: t.FontSize - 2, Color: t.TextMuted, Features: "tnum"}
	_, nh := p.MeasureText(b.W-t.Space(4), name)
	_, yh := p.MeasureText(0, years)
	y := b.Y + (b.H-nh-yh)/2
	p.RichText(b.X+t.Space(2), y, b.W-t.Space(4), name)
	p.RichText(b.X+t.Space(2), y+nh, 0, years)
}

func lifespan(p Person) string {
	if p.Died == 0 {
		return fmt.Sprintf("b. %d", p.Born)
	}
	return fmt.Sprintf("%d–%d", p.Born, p.Died)
}

// about writes a picked person's life and, for a descendant or spouse,
// how many children and grandchildren the couple had.
func about(who Person, n *node, spouse bool) string {
	parts := []string{who.Name, lifespan(who)}
	if who.Died != 0 {
		parts = append(parts, fmt.Sprintf("lived %d years", who.Died-who.Born))
	}
	kids, grand := len(n.f.Children), 0
	for _, ch := range n.f.Children {
		grand += len(ch.Children)
	}
	if kids > 0 {
		parts = append(parts, plural(kids, "child", "children"))
	}
	if grand > 0 {
		parts = append(parts, plural(grand, "grandchild", "grandchildren"))
	}
	if spouse {
		parts = append(parts, "married "+n.f.Person.Name)
	}
	return strings.Join(parts, " · ")
}

func plural(n int, one, many string) string {
	if n == 1 {
		return "1 " + one
	}
	return fmt.Sprintf("%d %s", n, many)
}

func contains(r ui.Rect, x, y float32) bool {
	return x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H
}
