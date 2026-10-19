// Package columns shows a hierarchy as Finder's column view does: a column
// for each level, the roots at the left, the item chosen in each opening
// a column of what it holds at its right, and past the last column, for
// an item that holds nothing, a preview of it. The columns scroll
// sideways as they go past the view's right.
//
//	columns.View(c, &app.browse, app.roots, children, func(f *File) {
//		ui.Icon(c, f.icon)
//		ui.Text(c, f.name).SingleLine()
//	}, preview).Grow(1)
//	if f, ok := app.browse.Opened(); ok {
//		open(f)
//	}
//
// Each column is a ui.List, which builds only the rows in view.
package columns

import (
	"slices"

	"github.com/egoist/mygo/ui"
)

// State is what a view of columns keeps: the path of items chosen, from
// the first column, and the columns' lists. Keep it in the app's state,
// one for each view.
type State[K comparable] struct {
	path   []K
	lists  []ui.ListState
	sel    []int
	focus  int  // the column to give the keyboard focus, 0 for none, else 1 more than it
	reveal bool // whether to scroll the last column into view
	opened K
	open   bool
}

// Path returns the items chosen, one in each column from the first.
func (s *State[K]) Path() []K { return s.path }

// SetPath chooses the items of path, one in each column from the first,
// as to reveal an item where it is, gives the last column chosen in the
// keyboard focus, and scrolls it into view.
func (s *State[K]) SetPath(path []K) {
	s.path = slices.Clone(path)
	s.focus, s.reveal = len(path), true
}

// Opened returns the item opened in the last frame, by a double click on
// it or Return while its column had the keyboard focus.
func (s *State[K]) Opened() (K, bool) { return s.opened, s.open }

// View creates the columns of the hierarchy from roots, which children
// returns the items of, none for an item that holds none: row builds the
// content of an item's row, as its icon and its name, after which the
// view puts an arrow for items that hold some, and preview, unless nil,
// builds the preview of an item chosen that holds none. children is
// called for each row in view, in each frame: cache what is slow to get,
// as the entries of a directory. Give the view a size, or grow it in its
// parent.
//
// A click chooses an item, as Up and Down do in the column with the
// keyboard focus, and Right goes into the next column, Left back to the
// one before. Items gone from their column end the path there.
func View[K comparable](c *ui.Context, s *State[K], roots []K, children func(K) []K, row func(K), preview func(K)) ui.Element {
	t := c.Theme()
	var zero K
	s.opened, s.open = zero, false
	cols := [][]K{roots}
	for i := 0; i < len(s.path); i++ {
		if !slices.Contains(cols[i], s.path[i]) {
			s.path = s.path[:i]
			break
		}
		kids := children(s.path[i])
		if len(kids) == 0 {
			s.path = s.path[:i+1]
			break
		}
		cols = append(cols, kids)
	}
	for len(s.lists) < len(cols) {
		s.lists = append(s.lists, ui.ListState{})
		s.sel = append(s.sel, -1)
	}
	for i := range cols {
		s.sel[i] = -1
		if i < len(s.path) {
			s.sel[i] = slices.Index(cols[i], s.path[i])
		}
	}
	width := t.Space(55)
	leaf := len(s.path) > 0 && len(s.path) == len(cols)
	reveal := s.reveal
	s.reveal = false

	scroller := ui.ScrollHorizontal(c)
	scroller.Children(func() {
		ui.Row(c).Fill().AlignItems(ui.Stretch).Children(func() {
			for i, items := range cols {
				ls := &s.lists[i]
				ls.Selected = &s.sel[i]
				list := ui.List(c.Key(i), ls, len(items), func(r int) {
					k := items[r]
					on := r == s.sel[i]
					ui.Row(c).Gap(t.Space(1.5)).AlignItems(ui.Center).Padding(t.Space(1), t.Space(2)).Children(func() {
						ui.Row(c).Gap(t.Space(1.5)).AlignItems(ui.Center).Grow(1).Shrink(1).Children(func() { row(k) })
						if len(children(k)) > 0 {
							arrow(c, on)
						}
					})
				}).Width(width).Padding(t.Space(1))
				if list.Changed() && s.sel[i] >= 0 {
					s.path = append(s.path[:i], items[s.sel[i]])
					s.reveal = true
				}
				if (list.Submitted() || list.DoubleClicked()) && s.sel[i] >= 0 {
					s.opened, s.open = items[s.sel[i]], true
				}
				if ls.Shortcut(c, 0, ui.KeyRight) && i+1 < len(cols) && len(s.path) > i {
					s.path = append(s.path[:i+1], cols[i+1][0])
					s.focus, s.reveal = i+2, true
				}
				if ls.Shortcut(c, 0, ui.KeyLeft) && i > 0 {
					s.path = s.path[:i]
					s.focus = i
				}
				if s.focus == i+1 && ls.Focus(c) {
					s.focus = 0
				}
				if reveal && i == len(cols)-1 && !leaf {
					list.ScrollIntoView()
				}
				ui.Divider(c)
			}
			if leaf && preview != nil {
				last := s.path[len(s.path)-1]
				p := ui.Column(c).Grow(1).MinWidth(width).Padding(t.Space(4)).Gap(t.Space(2)).Children(func() { preview(last) })
				if reveal {
					p.ScrollIntoView()
				}
			}
		})
	})
	return scroller
}

// arrow builds the arrow of an item that holds others, in the accent's
// text color on the item chosen.
func arrow(c *ui.Context, on bool) {
	t := c.Theme()
	color := t.TextMuted
	if on {
		color = t.AccentText
	}
	ui.Box(c).Size(t.Space(2), t.Space(2.5)).Draw(func(p *ui.Painter, r ui.Rect) {
		p.StrokePolyline([][2]float32{{r.X + r.W*0.25, r.Y}, {r.X + r.W*0.75, r.Y + r.H/2}, {r.X + r.W*0.25, r.Y + r.H}}, 1.5, color)
	})
}
