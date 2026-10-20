// Package outline is an outliner, as Workflowy and Logseq are: items of a
// line of text each, nested under the item above by their depth, with
// lines down the levels and a bullet that folds the items under it,
// ringed while they are folded. As the user types in an item:
//
//   - Enter adds an item after it, the text after the caret moving to it;
//     after an item with items unfolded under it, as the first of them.
//
//   - Backspace at its start joins it to the item above; an empty one
//     with nothing under it goes.
//
//   - Tab nests it, with the items under it, under the item above, and
//     Shift+Tab takes it out a level.
//
//   - Up and Down go to the item above and below, and Cmd+Up and Cmd+Down
//     move it, with the items under it, before or after the item beside
//     it.
//
//     outline.View(c, &app.outliner, &app.items)
package outline

import (
	"slices"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
)

// Item is an item of an outline: its text, and how deep it is, 0 at the
// top, one more than the item it is under.
type Item struct {
	Text  string
	Depth int
}

// State is what an outliner keeps: the identities of the items, as they
// move, the items folded, and the caret of the item being typed in, and
// where it goes next. Keep it in the app's state, one for each outliner.
type State struct {
	ids    []int
	next   int
	closed map[int]bool
	// focus is the identity of the item to give the focus to, plus 1, and
	// caret where its caret goes.
	focus, caret int
	// at and to are the selection of the item with the focus, as of the
	// last frame.
	at, to int
}

// end returns the index after the items under item i.
func end(items []Item, i int) int {
	j := i + 1
	for j < len(items) && items[j].Depth > items[i].Depth {
		j++
	}
	return j
}

// sibling returns the item at item i's depth before it, or after, -1 for
// none.
func sibling(items []Item, i int, after bool) int {
	d := items[i].Depth
	if after {
		if j := end(items, i); j < len(items) && items[j].Depth == d {
			return j
		}
		return -1
	}
	k := i - 1
	for k >= 0 && items[k].Depth > d {
		k--
	}
	if k >= 0 && items[k].Depth == d {
		return k
	}
	return -1
}

// View creates the outliner of *items, with an empty item to type in
// when there is none.
func View(c *ui.Context, s *State, items *[]Item) ui.Element {
	t := c.Theme()
	if len(*items) == 0 {
		*items = []Item{{}}
	}
	if len(s.ids) != len(*items) {
		s.ids = make([]int, len(*items))
		for i := range s.ids {
			s.ids[i] = i
		}
		s.next = len(s.ids)
	}
	if s.closed == nil {
		s.closed = map[int]bool{}
	}
	indent := t.Space(6)
	return ui.Column(c).Children(func() {
		its := *items
		for i := 0; i < len(its); i++ {
			id, it := s.ids[i], its[i]
			parent := end(its, i) > i+1
			closed := parent && s.closed[id]
			ui.Row(c.Key(id)).PaddingStart(float32(it.Depth) * indent).AlignItems(ui.Center).Children(func() {
				b := ui.ButtonBase(c).Size(indent, indent).Shrink(0).Label("Fold")
				if closed {
					b.Label("Unfold")
				}
				if !parent {
					b.Disabled(true)
				}
				if b.Clicked() {
					s.closed[id] = !closed
				}
				hot := b.Hovered() && parent
				b.Draw(func(p *ui.Painter, r ui.Rect) {
					cx, cy := r.X+r.W/2, r.Y+r.H/2
					if closed || hot {
						var halo ui.Path
						halo.Circle(cx, cy, 7)
						p.FillPath(&halo, t.Text.Alpha(0.12))
					}
					var dot ui.Path
					dot.Circle(cx, cy, 3)
					p.FillPath(&dot, t.TextMuted)
				})
				in := ui.TextInputBase(c, &its[i].Text).Grow(1).Padding(t.Space(1), t.Space(1)).Label("Item")
				if s.focus == id+1 {
					in.Focus().SetTextSelection(s.caret, s.caret)
					s.focus = 0
				}
				if in.Focused() {
					s.at, s.to = in.TextSelection()
				}
				in.OnTextKey(func(k ui.TextKey) bool { return s.key(items, id, k) })
			}).Draw(func(p *ui.Painter, r ui.Rect) {
				for d := range it.Depth {
					x := r.X + float32(d)*indent + indent/2
					p.Line(x, r.Y, x, r.Y+r.H, 1, t.Text.Alpha(0.1))
				}
			})
			if closed {
				i = end(its, i) - 1
			}
		}
	})
}

// key does what key k does in the item of identity id, and reports
// whether it did anything.
func (s *State) key(items *[]Item, id int, k ui.TextKey) bool {
	its := *items
	i := slices.Index(s.ids, id)
	if i < 0 {
		return false
	}
	d := its[i].Depth
	switch {
	case k.Key == ui.KeyEnter && k.Mods == 0:
		text := k.Text()
		at := byteAt(text, s.at)
		k.SetText(text[:at])
		its[i].Text = text[:at]
		next, depth := i+1, d
		switch {
		case end(its, i) > i+1 && !s.closed[id]:
			depth = d + 1
		case end(its, i) > i+1:
			next = end(its, i)
		}
		s.insert(items, next, Item{Text: text[at:], Depth: depth})
		s.focus, s.caret = s.ids[next]+1, 0
	case k.Key == ui.KeyBackspace && k.Mods == 0 && s.to == 0 && i > 0:
		if end(its, i) > i+1 {
			return false
		}
		prev := s.above(its, i)
		text := k.Text()
		s.focus, s.caret = s.ids[prev]+1, utf8.RuneCountInString(its[prev].Text)
		its[prev].Text += text
		*items = slices.Delete(its, i, i+1)
		s.ids = slices.Delete(s.ids, i, i+1)
	case k.Key == ui.KeyTab && k.Mods == 0:
		p := sibling(its, i, false)
		if p < 0 {
			return true
		}
		delete(s.closed, s.ids[p])
		for j := i; j < end(its, i); j++ {
			its[j].Depth++
		}
	case k.Key == ui.KeyTab && k.Mods == ui.Shift:
		if d == 0 {
			return true
		}
		for j := end(its, i) - 1; j >= i; j-- {
			its[j].Depth--
		}
	case (k.Key == ui.KeyUp || k.Key == ui.KeyDown) && k.Mods == ui.Cmd:
		down := k.Key == ui.KeyDown
		o := sibling(its, i, down)
		if o < 0 {
			return true
		}
		a, b := o, i
		if down {
			a, b = i, o
		}
		s.swap(items, a, b, end(its, b))
		s.focus, s.caret = id+1, s.at
	case k.Key == ui.KeyUp && k.Mods == 0:
		if i > 0 {
			prev := s.above(its, i)
			s.focus, s.caret = s.ids[prev]+1, min(s.at, utf8.RuneCountInString(its[prev].Text))
		}
	case k.Key == ui.KeyDown && k.Mods == 0:
		next := i + 1
		if s.closed[id] {
			next = end(its, i)
		}
		if next < len(its) {
			s.focus, s.caret = s.ids[next]+1, min(s.at, utf8.RuneCountInString(its[next].Text))
		}
	default:
		return false
	}
	return true
}

// above returns the item showing above item i: the one before it, or
// the outermost folded item it is under.
func (s *State) above(its []Item, i int) int {
	k := i - 1
	depth := its[k].Depth
	for j := k - 1; j >= 0 && depth > 0; j-- {
		if its[j].Depth < depth {
			depth = its[j].Depth
			if s.closed[s.ids[j]] {
				k = j
			}
		}
	}
	return k
}

// insert puts it at i, with a new identity.
func (s *State) insert(items *[]Item, i int, it Item) {
	*items = slices.Insert(*items, i, it)
	s.ids = slices.Insert(s.ids, i, s.next)
	s.next++
}

// swap puts the items from b to e before those from a to b.
func (s *State) swap(items *[]Item, a, b, e int) {
	its := *items
	moved := slices.Concat(its[b:e], its[a:b])
	copy(its[a:e], moved)
	ids := slices.Concat(s.ids[b:e], s.ids[a:b])
	copy(s.ids[a:e], ids)
}

// byteAt returns the byte offset of rune r in s.
func byteAt(s string, r int) int {
	for i := range s {
		if r == 0 {
			return i
		}
		r--
	}
	return len(s)
}
