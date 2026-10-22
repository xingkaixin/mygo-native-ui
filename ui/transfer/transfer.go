// Package transfer is a transfer list, a dual list box, as for choosing
// the columns of a table or the permissions of a role: the items available
// at the left and those chosen at the right, in the order the app keeps
// them. Each list has a title, how many of its rows are selected, "3 of
// 40", a search field, and a check box selecting all its rows in view,
// mixed while some are.
//
//	if transfer.View(c, &app.picker, &app.columns, fields).Grow(1).Changed() {
//		app.save()
//	}
//
// A click selects a row, Cmd-click adds it or takes it away, and
// Shift-click selects the rows from the one clicked last. Up and Down
// select the row above and below, Shift with them adding it. The buttons
// between the lists move the rows selected to the other list, as Enter
// does, and a double click moves the row clicked.
//
// Dragging a row moves it, with the others selected when it is one of
// them: to the other list, or within Chosen to another place, which a line
// shows, since the order of the chosen items matters. Disabled items
// neither move nor select. Each list builds only the rows in view, so it
// holds thousands of items.
package transfer

import (
	"fmt"
	"slices"
	"strings"

	"github.com/egoist/mygo/ui"
)

// Item is an item to choose: its key, unique, its title, a detail shown
// muted after it, and whether it stays in the list it is in.
type Item struct {
	Key      string
	Title    string
	Detail   string
	Disabled bool
}

// State is what a transfer list keeps: the search typed and the rows
// selected in each of its lists. Keep it in the app's state, one for each
// view. The zero State searches and selects nothing.
type State struct {
	sides [2]side
}

// side is what a list of a View keeps.
type side struct {
	list  ui.ListState
	query string
	sel   map[string]bool
	// anchor is the item Shift selects from, and lead the one the keys
	// move from.
	anchor, lead string
	// reveal asks the next View to scroll the lead into view.
	reveal bool
}

// The lists of a View: Available at the left, Chosen at the right.
const (
	left = iota
	right
)

var titles = [2]string{"Available", "Chosen"}

// ref is an item dragged, with its view, so that views take only their
// own, and the list it is dragged from.
type ref struct {
	s    *State
	from int
	key  string
}

// view is what a View builds its lists with.
type view struct {
	c      *ui.Context
	s      *State
	chosen *[]string
	items  []Item
	// rows holds the items each list shows, in order, as indices of items.
	rows [2][]int
	// over is the row of Chosen a drag would drop at, before it with
	// above, and end whether it would drop past the last row.
	over       int
	above, end bool
	// moving is what the frame moves to the list to, before the chosen
	// item before, "" for the end.
	moving []string
	to     int
	before string
}

// View creates the transfer list of items, the keys of those chosen in
// *chosen, in order, which it edits in place as the user moves them and
// reports as a change. Keys of *chosen not in items stay there, unseen.
// Give it a size, or grow it in its parent.
func View(c *ui.Context, s *State, chosen *[]string, items []Item) ui.Element {
	t := c.Theme()
	v := &view{c: c, s: s, chosen: chosen, items: items, over: -1}
	v.filter()
	e := ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Stretch)
	e.Children(func() {
		v.list(left)
		ui.Column(c).Gap(t.Space(2)).Justify(ui.Center).Children(func() {
			if v.button("Add", chevrons[1], left) {
				v.ask(v.selected(left), right, "")
			}
			if v.button("Remove", chevrons[0], right) {
				v.ask(v.selected(right), left, "")
			}
		})
		v.list(right)
	})
	if v.apply() {
		e.SetChanged()
	}
	return e
}

// filter lists the items of each list that match its search, and keeps
// selected only those listed.
func (v *view) filter() {
	index := make(map[string]int, len(v.items))
	for i := range v.items {
		index[v.items[i].Key] = i
	}
	in := make(map[string]bool, len(*v.chosen))
	for _, k := range *v.chosen {
		if i, ok := index[k]; ok {
			in[k] = true
			if v.match(right, i) {
				v.rows[right] = append(v.rows[right], i)
			}
		}
	}
	for i := range v.items {
		if !in[v.items[i].Key] && v.match(left, i) {
			v.rows[left] = append(v.rows[left], i)
		}
	}
	for side := range v.s.sides {
		sd := &v.s.sides[side]
		if sd.sel == nil {
			sd.sel = map[string]bool{}
		}
		if keys := v.selected(side); len(keys) != len(sd.sel) {
			clear(sd.sel)
			for _, k := range keys {
				sd.sel[k] = true
			}
		}
	}
}

// match reports whether item i matches the search of list side.
func (v *view) match(side, i int) bool {
	q := strings.ToLower(strings.TrimSpace(v.s.sides[side].query))
	it := &v.items[i]
	return q == "" || strings.Contains(strings.ToLower(it.Title), q) || strings.Contains(strings.ToLower(it.Detail), q)
}

// selected returns the keys of the rows of list side selected, in order.
func (v *view) selected(side int) []string {
	var keys []string
	for _, i := range v.rows[side] {
		if it := &v.items[i]; v.s.sides[side].sel[it.Key] && !it.Disabled {
			keys = append(keys, it.Key)
		}
	}
	return keys
}

// Sizes, in the theme's spaces.
const (
	rowHeight = 7
	boxSize   = 4
)

// list builds list side in its panel: its title, count and search over
// its rows.
func (v *view) list(side int) {
	c, s, t := v.c, v.s, v.c.Theme()
	sd := &s.sides[side]
	rows := v.rows[side]
	if sd.reveal {
		if i := v.index(side, sd.lead); i >= 0 {
			sd.list.ScrollIntoView(i)
		}
		sd.reveal = false
	}
	ui.Column(c.Key(side)).Grow(1).MinWidth(t.Space(50)).Radius(t.Space(2)).Border(1, t.Border).Background(t.Background).Clip().Children(func() {
		ui.Row(c).Gap(t.Space(2)).Padding(t.Space(2), t.Space(3)).AlignItems(ui.Center).Children(func() {
			v.check(side)
			ui.Text(c, titles[side]).Bold().SingleLine().Grow(1).Shrink(1)
			count := fmt.Sprint(len(rows))
			if n := len(sd.sel); n > 0 {
				count = fmt.Sprintf("%d of %d", n, len(rows))
			}
			ui.Text(c, count).TextColor(t.TextMuted).FontFeatures("tnum").SingleLine()
		})
		ui.SearchField(c, &sd.query).Label("Search "+strings.ToLower(titles[side])).Margin(0, t.Space(2), t.Space(1))
		sd.list.Key = func(i int) any { return v.items[rows[i]].Key }
		sd.list.Label = func(i int) string { return v.items[rows[i]].Title }
		list := ui.List(c, &sd.list, len(rows), func(i int) { v.row(side, i) }).Grow(1).
			Padding(t.Space(1), t.Space(1), t.Space(2)).Focusable().Label(titles[side])
		list.Children(func() {
			if len(rows) == 0 {
				empty := "No items"
				if strings.TrimSpace(sd.query) != "" {
					empty = "No matches"
				}
				ui.Text(c, empty).TextColor(t.TextMuted).Padding(t.Space(2))
			}
		})
		r, over := ui.DragOver[ref](list)
		d, dropped := ui.Drop[ref](list)
		if dropped {
			r, over = d, true
		}
		switch {
		case !over || r.s != s:
		case side == right:
			v.end = len(rows) > 0
			if dropped {
				v.ask(v.carried(r), right, "")
			}
		case r.from == right:
			list.Background(t.Accent.Alpha(0.08))
			if dropped {
				v.ask(v.carried(r), left, "")
			}
		}
		v.keys(list, side)
	})
}

// row builds row i of list side.
func (v *view) row(side, i int) {
	c, s, t := v.c, v.s, v.c.Theme()
	sd := &s.sides[side]
	it := &v.items[v.rows[side][i]]
	on := sd.sel[it.Key]
	box := ui.Row(c).Height(t.Space(rowHeight)).Padding(0, t.Space(2)).Gap(t.Space(2)).AlignItems(ui.Center).
		Label(it.Title).Selected(on)
	// Rows selected together make one block, as in Finder.
	top, bottom := t.Radius, t.Radius
	if on && i > 0 && sd.sel[v.key(side, i-1)] {
		top = 0
	}
	if on && i+1 < len(v.rows[side]) && sd.sel[v.key(side, i+1)] {
		bottom = 0
	}
	box.Radius(top, top, bottom, bottom)
	if !it.Disabled {
		box.Drag(ref{s, side, it.Key})
		if box.Pressed() && !on && c.Modifiers()&(ui.Shift|ui.Cmd) == 0 {
			// A row not selected is selected as the press begins, to drag
			// it at once.
			v.only(side, i)
		}
		if box.Clicked() {
			switch m := box.ClickModifiers(); {
			case m&ui.Shift != 0:
				v.span(side, i)
			case m&ui.Cmd != 0:
				v.toggle(side, i)
			default:
				v.only(side, i)
			}
		}
		if box.DoubleClicked() {
			v.ask([]string{it.Key}, 1-side, "")
		}
		if box.Hovered() && !on {
			box.Background(t.Text.Alpha(0.05))
		}
		if box.Dragging() {
			box.Opacity(0.4)
		}
	}
	if side == right {
		_, py, _ := box.PointerPosition()
		above := py < box.Bounds().H/2
		r, over := ui.DragOver[ref](box)
		if d, ok := ui.Drop[ref](box); ok && d.s == s {
			before := it.Key
			if !above {
				before = v.next(it.Key)
			}
			v.ask(v.carried(d), right, before)
		}
		if over && r.s == s {
			v.over, v.above = i, above
		}
		last := i == len(v.rows[side])-1
		box.DrawOver(func(p *ui.Painter, b ui.Rect) {
			y := b.Y
			switch {
			case v.over == i && v.above:
			case v.over == i, v.end && last:
				y += b.H
			default:
				return
			}
			p.Fill(ui.Rect{X: b.X + 4, Y: y - 1, W: b.W - 4, H: 2}, t.Accent, 1)
			p.Fill(ui.Rect{X: b.X, Y: y - 4, W: 8, H: 8}, t.Accent, 4)
		})
	}
	muted := t.TextMuted
	if on {
		muted = t.AccentText
	}
	box.Children(func() {
		if side == right {
			grip(c, muted, !it.Disabled)
		}
		dim := float32(1)
		if it.Disabled {
			dim = 0.45
		}
		ui.Text(c, it.Title).SingleLine().Grow(1).Shrink(1).Opacity(dim)
		if it.Detail != "" {
			ui.Text(c, it.Detail).SingleLine().FontSize(t.FontSize - 1).TextColor(muted).Opacity(dim)
		}
	})
}

// grip builds the handle of a chosen row, which shows it reorders, or
// room for one.
func grip(c *ui.Context, color ui.Color, show bool) {
	t := c.Theme()
	k := t.Space(3)
	g := ui.Box(c).Size(k, k).Shrink(0)
	if !show {
		return
	}
	g.Draw(func(p *ui.Painter, r ui.Rect) {
		for j := range 3 {
			for i := range 2 {
				p.Fill(ui.Rect{X: r.X + r.W/2 - 3 + float32(i)*4, Y: r.Y + r.H/2 - 4.5 + float32(j)*4, W: 2, H: 2}, color.Alpha(0.7), 1)
			}
		}
	})
}

// check builds the check box selecting all the rows of list side that
// move, mixed while some are.
func (v *view) check(side int) {
	c, t := v.c, v.c.Theme()
	sd := &v.s.sides[side]
	movable := 0
	for _, i := range v.rows[side] {
		if !v.items[i].Disabled {
			movable++
		}
	}
	n := len(sd.sel)
	all := n > 0 && n == movable
	on := all
	b := ui.CheckboxBase(c, &on).FocusRing(false).Label("Select all " + strings.ToLower(titles[side])).Disabled(movable == 0)
	if n > 0 && !all {
		b.Mixed()
	}
	if b.Clicked() {
		clear(sd.sel)
		if !all {
			for _, i := range v.rows[side] {
				if !v.items[i].Disabled {
					sd.sel[v.items[i].Key] = true
				}
			}
		}
		sd.anchor, sd.lead = "", ""
	}
	b.Children(func() {
		box := ui.Box(c).Size(t.Space(boxSize), t.Space(boxSize)).Radius(t.Space(1)).Shrink(0)
		if n > 0 {
			box.Background(t.Accent)
		} else {
			box.Background(t.Background).Border(1, t.Border.Mix(t.Text, 0.25))
		}
		if movable == 0 {
			box.Opacity(0.5)
		}
		box.DrawOver(func(p *ui.Painter, r ui.Rect) {
			switch {
			case all:
				var path ui.Path
				path.MoveTo(r.X+r.W*0.22, r.Y+r.H*0.52).LineTo(r.X+r.W*0.42, r.Y+r.H*0.71).LineTo(r.X+r.W*0.78, r.Y+r.H*0.31)
				p.StrokePath(&path, t.Space(0.5), t.AccentText)
			case n > 0:
				p.Fill(ui.Rect{X: r.X + r.W*0.25, Y: r.Y + r.H/2 - 1, W: r.W * 0.5, H: 2}, t.AccentText, 1)
			}
			if b.FocusVisible() {
				p.FocusRing(r, [4]float32{t.Space(1), t.Space(1), t.Space(1), t.Space(1)})
			}
		})
	})
}

// button builds the button moving the rows selected in list from to the
// other list, and reports a click.
func (v *view) button(label string, icon *ui.SVG, from int) bool {
	c, t := v.c, v.c.Theme()
	k := t.Space(4)
	return ui.Button(c, "").Label(label).Tooltip(label).Disabled(len(v.s.sides[from].sel) == 0).Children(func() {
		ui.Icon(c, icon).Size(k, k)
	}).Clicked()
}

// keys does what the keys pressed on list side do.
func (v *view) keys(list ui.Element, side int) {
	sd := &v.s.sides[side]
	at := v.index(side, sd.lead)
	switch {
	case list.Shortcut(0, ui.KeyUp):
		if i := v.step(side, at, -1); i >= 0 {
			v.only(side, i)
		}
	case list.Shortcut(0, ui.KeyDown):
		if i := v.step(side, at, 1); i >= 0 {
			v.only(side, i)
		}
	case list.Shortcut(ui.Shift, ui.KeyUp):
		if i := v.step(side, at, -1); i >= 0 {
			v.span(side, i)
		}
	case list.Shortcut(ui.Shift, ui.KeyDown):
		if i := v.step(side, at, 1); i >= 0 {
			v.span(side, i)
		}
	case list.Shortcut(0, ui.KeyEnter):
		v.ask(v.selected(side), 1-side, "")
		return
	default:
		return
	}
	if i := v.index(side, sd.lead); i >= 0 {
		sd.list.ScrollIntoView(i)
	}
}

// step returns the row of list side that moves, nearest to row at in
// direction dir, from the end without at, or -1 for none.
func (v *view) step(side, at, dir int) int {
	rows := v.rows[side]
	if at < 0 {
		at = len(rows)
		if dir > 0 {
			at = -1
		}
	}
	for i := at + dir; i >= 0 && i < len(rows); i += dir {
		if !v.items[rows[i]].Disabled {
			return i
		}
	}
	return -1
}

// index returns the row of list side showing the item key, -1 for none.
func (v *view) index(side int, key string) int {
	return slices.IndexFunc(v.rows[side], func(i int) bool { return v.items[i].Key == key })
}

// key returns the key of row i of list side.
func (v *view) key(side, i int) string { return v.items[v.rows[side][i]].Key }

// only selects row i of list side alone.
func (v *view) only(side, i int) {
	sd, k := &v.s.sides[side], v.key(side, i)
	clear(sd.sel)
	sd.sel[k] = true
	sd.anchor, sd.lead = k, k
}

// toggle selects row i of list side, or takes it out of the selection.
func (v *view) toggle(side, i int) {
	sd, k := &v.s.sides[side], v.key(side, i)
	if sd.sel[k] {
		delete(sd.sel, k)
	} else {
		sd.sel[k] = true
	}
	sd.anchor, sd.lead = k, k
}

// span selects the rows of list side that move from the anchor to row i.
func (v *view) span(side, i int) {
	sd := &v.s.sides[side]
	a := v.index(side, sd.anchor)
	if a < 0 {
		v.only(side, i)
		return
	}
	clear(sd.sel)
	for j := min(a, i); j <= max(a, i); j++ {
		if it := &v.items[v.rows[side][j]]; !it.Disabled {
			sd.sel[it.Key] = true
		}
	}
	sd.lead = v.key(side, i)
}

// carried returns the keys a drag of r moves: those selected in its list
// when it is one of them, else its own.
func (v *view) carried(r ref) []string {
	if v.s.sides[r.from].sel[r.key] {
		return v.selected(r.from)
	}
	return []string{r.key}
}

// next returns the chosen key after key, "" for the last.
func (v *view) next(key string) string {
	ks := *v.chosen
	if i := slices.Index(ks, key); i >= 0 && i+1 < len(ks) {
		return ks[i+1]
	}
	return ""
}

// ask asks the frame to move keys to list to, in Chosen before the key
// before, "" for the end.
func (v *view) ask(keys []string, to int, before string) {
	if len(keys) > 0 {
		v.moving, v.to, v.before = keys, to, before
	}
}

// apply moves what the frame asked, selects what moved in the list it went
// to, and reports whether *chosen changed.
func (v *view) apply() bool {
	keys := v.moving
	if keys == nil {
		return false
	}
	moving := make(map[string]bool, len(keys))
	for _, k := range keys {
		moving[k] = true
	}
	was := *v.chosen
	// Before the first chosen key from before on that does not move too.
	before := ""
	if j := slices.Index(was, v.before); v.before != "" && j >= 0 {
		for ; j < len(was) && moving[was[j]]; j++ {
		}
		if j < len(was) {
			before = was[j]
		}
	}
	out := slices.DeleteFunc(slices.Clone(was), func(k string) bool { return moving[k] })
	if v.to == right {
		j := len(out)
		if before != "" {
			j = slices.Index(out, before)
		}
		out = slices.Insert(out, j, keys...)
	}
	for side := range v.s.sides {
		clear(v.s.sides[side].sel)
	}
	sd := &v.s.sides[v.to]
	for _, k := range keys {
		sd.sel[k] = true
	}
	sd.anchor, sd.lead, sd.reveal = keys[0], keys[0], true
	if slices.Equal(was, out) {
		return false
	}
	*v.chosen = out
	return true
}

func svg(body string) *ui.SVG {
	return ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">` + body + `</svg>`))
}

var chevrons = [2]*ui.SVG{
	svg(`<path d="M15 6l-6 6 6 6"/>`),
	svg(`<path d="M9 6l6 6-6 6"/>`),
}
