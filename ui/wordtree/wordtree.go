// Package wordtree shows the contexts of a phrase in a body of text as a
// word tree: the phrase in the middle, the words that follow it branching
// to the right and those before it branching to the left, a branch for
// each different word, merged while sentences agree and as large as the
// number of sentences through it. A branch that only one run of
// sentences takes shows as a phrase. A click on a word makes the phrase
// up to it the root.
//
//	wordtree.View(c, &app.root, app.sentences)
package wordtree

import (
	"cmp"
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/egoist/mygo/ui"
)

const (
	// depth is how many words from the root a branch follows.
	depth = 10
	// branches is how many branches a word shows, the most taken.
	branches = 7
	// phrase is how many words a branch joins before cutting with "…".
	phrase = 6
)

// node is a branch: its words, the sentences through it, and the
// branches after it. side is 1 for
// what follows the root and -1 for what comes before it, and x, y, w and
// h its box once laid out, span the height of the branches it holds.
type node struct {
	key          string
	words        []string
	count        int
	kids         []*node
	parent       *node
	side         int
	size         float32
	x, y, w, h   float32
	span, extent float32
}

// key is how a word is matched: in lower case, without punctuation
// around it.
func key(w string) string {
	return strings.ToLower(strings.Trim(w, ".,;:!?\"'“”‘’()[]—–-"))
}

// contexts returns, for each place sentences have the phrase root, the
// words after it and the words before it, nearest first.
func contexts(sentences []string, root []string) (after, before [][]string) {
	for _, s := range sentences {
		words := strings.Fields(s)
	next:
		for i := 0; i+len(root) <= len(words); i++ {
			for k, r := range root {
				if key(words[i+k]) != r {
					continue next
				}
			}
			after = append(after, words[i+len(root):])
			back := slices.Clone(words[:i])
			slices.Reverse(back)
			before = append(before, back)
		}
	}
	return after, before
}

// grow returns the tree of seqs, nearest word first, for side.
func grow(seqs [][]string, side int) *node {
	top := &node{side: side}
	for _, seq := range seqs {
		n := top
		n.count++
		for _, w := range seq[:min(len(seq), depth)] {
			k := key(w)
			i := slices.IndexFunc(n.kids, func(kid *node) bool { return kid.key == k })
			if i < 0 {
				i = len(n.kids)
				n.kids = append(n.kids, &node{key: k, words: []string{w}, parent: n, side: side})
			}
			n = n.kids[i]
			n.count++
		}
	}
	prune(top)
	return top
}

// prune keeps the branches most taken and joins each branch that all its
// sentences go on through into one.
func prune(n *node) {
	slices.SortStableFunc(n.kids, func(a, b *node) int { return cmp.Compare(b.count, a.count) })
	n.kids = n.kids[:min(len(n.kids), branches)]
	for _, kid := range n.kids {
		for len(kid.kids) == 1 && kid.kids[0].count == kid.count && len(kid.words) < phrase {
			only := kid.kids[0]
			kid.words = append(kid.words, only.words...)
			kid.kids = only.kids
			for _, k := range kid.kids {
				k.parent = kid
			}
		}
		if len(kid.words) >= phrase && len(kid.kids) > 0 && kid.count == 1 {
			kid.kids = nil
			kid.words = append(kid.words, "…")
		}
		prune(kid)
	}
}

// text is what a branch shows, its words in reading order.
func (n *node) text() string {
	words := n.words
	if n.side < 0 {
		words = slices.Clone(words)
		slices.Reverse(words)
	}
	return strings.Join(words, " ")
}

// path returns the words from the root to n, n's included, in reading
// order, as keys.
func (n *node) path() []string {
	var keys []string
	for m := n; m.parent != nil; m = m.parent {
		ws := slices.Clone(m.words)
		slices.Reverse(ws)
		for _, w := range ws {
			if k := key(w); k != "" && w != "…" {
				keys = append(keys, k)
			}
		}
	}
	if n.side > 0 {
		slices.Reverse(keys)
	}
	return keys
}

// View creates the word tree of *root, a phrase, in sentences. A click on
// a word sets *root to the phrase up to it and reports Changed.
func View(c *ui.Context, root *string, sentences []string) ui.Element {
	t := c.Theme()
	var keys []string
	for _, w := range strings.Fields(*root) {
		keys = append(keys, key(w))
	}
	after, before := contexts(sentences, keys)
	canvas := ui.Box(c).Role(ui.RoleImage).Label(fmt.Sprintf("Word tree of “%s”, in %d places", *root, len(after)))
	if len(keys) == 0 || len(after) == 0 {
		return canvas.Children(func() {
			ui.Text(c, fmt.Sprintf("No sentence has “%s”.", *root)).TextColor(t.TextMuted)
		})
	}
	right, left := grow(after, 1), grow(before, -1)
	most := len(after)
	sizeOf := func(count int) float32 {
		return t.FontSize * (0.85 + 1.15*float32(math.Log(float64(count))/math.Log(float64(max(most, 2)))))
	}
	gapX, gapY := t.Space(6), t.Space(1)
	var measure func(n *node)
	measure = func(n *node) {
		n.size = sizeOf(n.count)
		n.w, n.h = c.MeasureText(0, ui.Span{Text: n.text(), Size: n.size})
		kids, widest := float32(0), float32(0)
		for i, kid := range n.kids {
			measure(kid)
			kids += kid.span
			if i > 0 {
				kids += gapY
			}
			widest = max(widest, kid.extent)
		}
		n.span, n.extent = max(n.h, kids), n.w
		if len(n.kids) > 0 {
			n.extent += gapX + widest
		}
	}
	var placeKids func(n *node, edge, top float32, out *[]*node)
	placeKids = func(n *node, edge, top float32, out *[]*node) {
		total := float32(0)
		for i, kid := range n.kids {
			total += kid.span
			if i > 0 {
				total += gapY
			}
		}
		y := top + (n.span-total)/2
		for _, kid := range n.kids {
			kid.y = y + (kid.span-kid.h)/2
			kid.x = edge
			if n.side < 0 {
				kid.x = edge - kid.w
			}
			*out = append(*out, kid)
			next := kid.x + kid.w + gapX
			if n.side < 0 {
				next = kid.x - gapX
			}
			placeKids(kid, next, y, out)
			y += kid.span + gapY
		}
	}

	rootSize := t.FontSize * 2.2
	rootW, rootH := c.MeasureText(0, ui.Span{Text: *root, Size: rootSize, Weight: 700})
	measure(right)
	measure(left)
	right.span, left.span = max(right.span, rootH), max(left.span, rootH)
	leftW, rightW := left.extent-left.w, right.extent-right.w
	if len(left.kids) > 0 {
		leftW -= gapX
	}
	if len(right.kids) > 0 {
		rightW -= gapX
	}
	pad := t.Space(4)
	h := max(left.span, right.span) + 2*pad
	rootX := pad + leftW + gapX
	rootY := h/2 - rootH/2
	var nodes []*node
	left.x, left.y, left.w, left.h = rootX, rootY, rootW, rootH
	right.x, right.y, right.w, right.h = rootX, rootY, rootW, rootH
	placeKids(right, rootX+rootW+gapX, h/2-right.span/2, &nodes)
	placeKids(left, rootX-gapX, h/2-left.span/2, &nodes)
	canvas.Size(rootX+rootW+gapX+rightW+pad, h)

	px, py, over := canvas.PointerPosition()
	var hovered *node
	if over {
		for _, n := range nodes {
			if px >= n.x && px <= n.x+n.w && py >= n.y && py <= n.y+n.h {
				hovered = n
			}
		}
	}
	if hovered != nil {
		canvas.Cursor(ui.CursorPointer)
	}
	if canvas.Clicked() && hovered != nil {
		if hovered.side > 0 {
			*root = strings.Join(append(slices.Clone(keys), hovered.path()...), " ")
		} else {
			*root = strings.Join(append(hovered.path(), keys...), " ")
		}
		canvas.SetChanged()
	}

	shown := *root
	canvas.Draw(func(p *ui.Painter, r ui.Rect) {
		link := t.TextMuted.Alpha(0.45)
		for _, n := range nodes {
			par := n.parent
			var x0, x1 float32
			if n.side > 0 {
				x0, x1 = par.x+par.w+t.Space(1), n.x-t.Space(1)
			} else {
				x0, x1 = par.x-t.Space(1), n.x+n.w+t.Space(1)
			}
			y0, y1 := par.y+par.h/2, n.y+n.h/2
			var curve ui.Path
			mid := (x0 + x1) / 2
			curve.MoveTo(r.X+x0, r.Y+y0).CubeTo(r.X+mid, r.Y+y0, r.X+mid, r.Y+y1, r.X+x1, r.Y+y1)
			p.StrokePath(&curve, 1, link)
		}
		p.RichText(r.X+rootX, r.Y+rootY, 0, ui.Span{Text: shown, Size: rootSize, Weight: 700, Color: t.Text})
		for _, n := range nodes {
			color := t.Text
			if n == hovered {
				color = t.Accent
			}
			p.RichText(r.X+n.x, r.Y+n.y, 0, ui.Span{Text: n.text(), Size: n.size, Color: color})
		}
	})
	return canvas
}
