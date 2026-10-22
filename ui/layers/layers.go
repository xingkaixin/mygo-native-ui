// Package layers is a design tool's layers panel, as Figma's, Sketch's
// and Photoshop's: the layers of a document as a tree, the front-most at
// the top, each row with an arrow that folds a frame or a group, an icon
// of its kind, its name, and buttons to lock and hide it, which show
// while the pointer is over the row and stay while the layer is locked or
// hidden. Hidden layers, and the layers in a hidden frame or group, are
// dimmed.
//
//	ev := layers.View(c, &app.panel, &app.doc)
//	if ev.Changed() {
//		app.save()
//	}
//	for _, id := range app.panel.Selected() {
//		app.outline(id) // on the canvas
//	}
//	if id, ok := app.clicked(); ok { // on the canvas
//		app.panel.Select(id)
//	}
//
// A click selects a layer, Cmd-click adds it or takes it away, and
// Shift-click selects the rows from the one clicked last. Up and Down
// select the row above and below, Shift with them adding it; Left folds a
// frame or group or goes to the one it is in, and Right unfolds it or goes
// to its first layer. A double click, or Enter, renames a layer in place:
// Enter keeps the name typed and Escape goes back. Delete and Backspace
// remove the layers selected.
//
// Dragging a layer moves it, with the others selected when it is one of
// them: a line shows where they would go between the rows, as deep as the
// pointer is far to the right, and a frame or a group lights up while
// they would go in it, at its top, as they do over the middle of its row.
// A folded frame or group the drag rests on unfolds. Layers never go into
// themselves.
package layers

import (
	"math"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
)

// Kind is what a layer is, which its icon shows.
type Kind uint8

const (
	Frame Kind = iota
	Group
	Rectangle
	Ellipse
	Text
	Image
	Path
)

// Layer is a layer of a document: its ID, unique and not empty, its name,
// what it is, whether it is hidden or locked, and for a frame or a group,
// the layers in it, the front-most first.
type Layer struct {
	ID       string
	Name     string
	Kind     Kind
	Hidden   bool
	Locked   bool
	Children []Layer
}

// holds reports whether layers go into l.
func (l *Layer) holds() bool { return l.Kind == Frame || l.Kind == Group }

// Events are what the user did with a panel in a frame. The zero Events
// report nothing.
type Events struct {
	selection        bool
	renamed, toggled string
	moved, deleted   []string
}

// SelectionChanged reports that the user changed the layers selected,
// which State.Selected returns.
func (e Events) SelectionChanged() bool { return e.selection }

// Renamed returns the layer the user renamed, whose Name is the new name.
func (e Events) Renamed() (string, bool) { return e.renamed, e.renamed != "" }

// Toggled returns the layer the user locked or unlocked, hid or showed,
// with its buttons.
func (e Events) Toggled() (string, bool) { return e.toggled, e.toggled != "" }

// Moved returns the layers the user dragged to another place, which are
// already there, each with the layers in it: the front-most first.
func (e Events) Moved() ([]string, bool) { return e.moved, e.moved != nil }

// Deleted returns the layers the user removed, which are already gone,
// with the layers that were in them.
func (e Events) Deleted() ([]string, bool) { return e.deleted, e.deleted != nil }

// Changed reports that the user changed the layers in any of these ways,
// for the app to save them or keep a step to undo.
func (e Events) Changed() bool {
	return e.renamed != "" || e.toggled != "" || e.moved != nil || e.deleted != nil
}

// State is what a panel keeps: the layers selected, the frames and groups
// folded, and the name typed while a layer is renamed. Keep it in the
// app's state, one for each panel. The zero State selects nothing and
// folds nothing.
type State struct {
	list ui.ListState
	sel  map[string]bool
	// chosen is sel in the order of the layers, as of the last View.
	chosen []string
	// anchor is the layer Shift selects from, and lead the one the keys
	// move from.
	anchor, lead string
	folded       map[string]bool
	// reveal asks the next View to unfold the selection and scroll the
	// lead into view.
	reveal bool
	// editing is the layer being renamed, and draft the name typed; done
	// is how the user ended it between frames, which the next frame does.
	// fresh gives the field the focus, focused tells it had it since, and
	// refocus gives it back to the list.
	editing, draft          string
	done                    int
	fresh, focused, refocus bool
	// moving is the layer whose row a drag moved in the last frame, and
	// resting the folded frame or group a drag rests on, since since.
	moving, resting string
	since           time.Time
}

// How a rename ends.
const (
	keep = iota + 1
	cancel
)

// unfoldDelay is how long a drag rests on a folded frame or group before
// it unfolds.
const unfoldDelay = 600 * time.Millisecond

// Selected returns the IDs of the layers selected, front-most first.
func (s *State) Selected() []string { return s.chosen }

// Select selects the layers of ids alone, none without any, as a click on
// the canvas does: the panel unfolds what holds them and scrolls the first
// into view.
func (s *State) Select(ids ...string) {
	s.sel = make(map[string]bool, len(ids))
	for _, id := range ids {
		s.sel[id] = true
	}
	s.chosen = slices.Clone(ids)
	s.anchor, s.lead = "", ""
	if len(ids) > 0 {
		s.anchor, s.lead = ids[0], ids[0]
	}
	s.reveal = true
}

// carried returns the layers a drag from id moves: those selected when it
// is one of them, else id.
func (s *State) carried(id string) map[string]bool {
	if s.sel[id] {
		return s.sel
	}
	return map[string]bool{id: true}
}

// rename starts renaming l.
func (s *State) rename(l *Layer) {
	s.editing, s.draft, s.done, s.fresh, s.focused = l.ID, l.Name, 0, true, false
}

// unfold unfolds the frames and groups holding a layer selected in ls, and
// reports whether ls holds one.
func (s *State) unfold(ls []Layer) bool {
	found := false
	for i := range ls {
		if s.unfold(ls[i].Children) {
			delete(s.folded, ls[i].ID)
			found = true
		}
		found = found || s.sel[ls[i].ID]
	}
	return found
}

// row is a row of the panel: its layer, how deep it is, the row of the
// frame or group it is in, -1 at the top, whether it or one it is in is
// hidden, and whether one it is in is selected.
type row struct {
	l           *Layer
	id          string
	depth, up   int
	hidden, lit bool
}

// flatten appends the rows of ls, depth deep in the row up, and those in
// them unfolded.
func (s *State) flatten(rows []row, ls []Layer, depth, up int, hidden, lit bool) []row {
	for i := range ls {
		l := &ls[i]
		rows = append(rows, row{l, l.ID, depth, up, hidden || l.Hidden, lit})
		if len(l.Children) > 0 && !s.folded[l.ID] {
			rows = s.flatten(rows, l.Children, depth+1, len(rows)-1, hidden || l.Hidden, lit || s.sel[l.ID])
		}
	}
	return rows
}

// spot is where layers dropped go: into the frame or group parent, "" for
// the top, before the layer before, "" for the end.
type spot struct{ parent, before string }

// target is where a drag over the panel would drop, and how it shows: in
// the frame or group of row, or a line above or below it, depth deep;
// rect is where row painted.
type target struct {
	ok          bool
	at          spot
	row, depth  int
	into, below bool
	rect        ui.Rect
}

// ref is a layer dragged, with its panel, so that panels take only their
// own.
type ref struct {
	panel *State
	id    string
}

// panel is what a View builds its rows with.
type panel struct {
	c       *ui.Context
	s       *State
	layers  *[]Layer
	rows    []row
	ev      Events
	tgt     target
	from    string
	dropped bool
	moving  string
	resting bool
	// moved holds the layers the drag under way moves.
	moved map[string]bool
}

// View creates the panel of *layers, which it edits in place as the user
// renames, locks, hides, moves and removes them, reporting each in the
// Events it returns. It fills its parent and scrolls its rows, building
// only those in view.
func View(c *ui.Context, s *State, layers *[]Layer) Events {
	t := c.Theme()
	if s.sel == nil {
		s.sel = map[string]bool{}
	}
	if s.folded == nil {
		s.folded = map[string]bool{}
	}
	p := &panel{c: c, s: s, layers: layers}
	if s.moving != "" {
		p.moved = s.carried(s.moving)
	}
	if s.editing != "" {
		switch sl, i := find(layers, s.editing); {
		case sl == nil:
			s.editing = ""
		case s.done != 0:
			p.finish(&(*sl)[i], s.done == keep)
			s.refocus = true
		}
	}
	if s.reveal {
		s.unfold(*layers)
	}
	p.rows = s.flatten(nil, *layers, 0, -1, false, false)
	if s.reveal {
		if i := p.index(s.lead); i >= 0 {
			s.list.ScrollIntoView(i)
		}
		s.reveal = false
	}
	rows := p.rows
	s.list.Key = func(i int) any { return rows[i].id }
	list := ui.List(c, &s.list, len(rows), p.row).Fill().Padding(0, 0, t.Space(4), 0).Focusable().Label("Layers")
	list.DrawOver(func(pt *ui.Painter, _ ui.Rect) { p.line(pt, t) })
	// Below the rows, layers go to the end.
	v, over := ui.DragOver[ref](list)
	if d, ok := ui.Drop[ref](list); ok {
		v, over, p.dropped = d, true, true
	}
	if over && v.panel == s && len(rows) > 0 {
		p.from, p.tgt = v.id, target{ok: true, row: len(rows) - 1, below: true}
	}
	if s.editing == "" {
		p.keys(list)
	}
	if p.dropped && p.tgt.ok {
		p.move(p.tgt.at, p.tgt.into)
	}
	if !p.resting {
		s.resting = ""
	}
	s.moving = p.moving
	if s.refocus {
		list.Focus()
		s.refocus = false
	}
	s.chosen = collect(*layers, s.sel, nil)
	if len(s.chosen) != len(s.sel) {
		// Layers gone since are no longer selected.
		clear(s.sel)
		for _, id := range s.chosen {
			s.sel[id] = true
		}
	}
	return p.ev
}

// Sizes of a row, in the theme's spaces.
const (
	rowHeight = 7
	inset     = 2
	indent    = 4
	arrowSize = 4
)

// row builds row i.
func (p *panel) row(i int) {
	c, s, t := p.c, p.s, p.c.Theme()
	r := p.rows[i]
	l := r.l
	h := t.Space(rowHeight)
	box := ui.Row(c).Height(h).PaddingStart(t.Space(inset) + float32(r.depth)*t.Space(indent)).PaddingEnd(t.Space(1)).
		Gap(t.Space(1)).AlignItems(ui.Center).FontSize(t.FontSize - 1).Drag(ref{s, r.id}).Label(l.Name).Level(r.depth + 1)
	px, py, _ := box.PointerPosition()
	if box.Pressed() && !s.sel[r.id] && c.Modifiers()&(ui.Shift|ui.Cmd) == 0 {
		// A layer not selected is selected as the press begins, to drag
		// it at once.
		p.only(i)
	}
	if box.Clicked() {
		switch m := box.ClickModifiers(); {
		case m&ui.Shift != 0:
			p.span(i)
		case m&ui.Cmd != 0:
			p.toggle(i)
		default:
			p.only(i)
		}
	}
	if box.DoubleClicked() {
		s.rename(l)
	}
	v, over := ui.DragOver[ref](box)
	if d, ok := ui.Drop[ref](box); ok {
		v, over, p.dropped = d, true, true
	}
	if over && v.panel == s {
		p.aim(i, v.id, px, py)
	}
	if box.Dragging() {
		p.moving = r.id
	}
	chosen, open := s.sel[r.id], len(l.Children) > 0 && !s.folded[r.id]
	box.Selected(chosen)
	if len(l.Children) > 0 {
		box.Expanded(open)
	}
	// Its buttons stay out of the copy a drag shows of the row.
	hovered := box.Hovered() && !box.Dragging()
	into := p.tgt.ok && p.tgt.into && p.tgt.row == i
	switch {
	case chosen:
	case into:
		box.Background(t.Accent.Alpha(0.16))
	case r.lit:
		box.Background(t.Accent.Alpha(0.12))
	case hovered:
		box.Background(t.Text.Alpha(0.05))
	}
	if p.within(i, p.moved) {
		box.Opacity(0.4)
	}
	box.Draw(func(_ *ui.Painter, b ui.Rect) {
		if i == p.tgt.row {
			p.tgt.rect = b
		}
	})
	if into {
		box.DrawOver(func(pt *ui.Painter, b ui.Rect) {
			pt.Stroke(ui.Rect{X: b.X + 1, Y: b.Y + 1, W: b.W - 2, H: b.H - 2}, t.Accent, t.Space(1), 2)
		})
	}
	muted := t.TextMuted
	if chosen {
		muted = t.AccentText
	}
	box.Children(func() {
		p.arrow(r, open, muted)
		icon := ui.Icon(c, kinds[l.Kind]).Size(t.Space(4), t.Space(4)).Shrink(0).TextColor(muted).MarginEnd(t.Space(0.5))
		var name ui.Element
		if s.editing == r.id {
			name = p.editor(l)
		} else {
			name = ui.Text(c, l.Name).SingleLine().Grow(1).Shrink(1)
			if l.Kind == Frame {
				name.FontWeight(600)
			}
		}
		if r.hidden {
			icon.Opacity(0.4)
			name.Opacity(0.4)
		}
		if button(c, "Lock", "Unlock", locks, l.Locked, hovered, muted).Clicked() {
			l.Locked = !l.Locked
			p.ev.toggled = r.id
		}
		if button(c, "Hide", "Show", eyes, l.Hidden, hovered, muted).Clicked() {
			l.Hidden = !l.Hidden
			p.ev.toggled = r.id
		}
	})
}

// within reports whether row i, or a row it is in, holds a layer of set.
func (p *panel) within(i int, set map[string]bool) bool {
	for ; i >= 0; i = p.rows[i].up {
		if set[p.rows[i].id] {
			return true
		}
	}
	return false
}

// arrow builds the arrow folding the layers of r, or room for one.
func (p *panel) arrow(r row, open bool, color ui.Color) {
	c, s, t := p.c, p.s, p.c.Theme()
	k := t.Space(arrowSize)
	if len(r.l.Children) == 0 {
		ui.Box(c).Size(k, k).Shrink(0)
		return
	}
	label := "Fold"
	if !open {
		label = "Unfold"
	}
	b := ui.ButtonBase(c).Size(k, k).Shrink(0).Label(label).Expanded(open)
	if b.Clicked() {
		if open {
			s.folded[r.id] = true
		} else {
			delete(s.folded, r.id)
		}
	}
	var turn float32
	if open {
		turn = 90
	}
	turn = b.Animate("turn", turn, 150*time.Millisecond)
	b.Draw(func(pt *ui.Painter, rect ui.Rect) {
		cx, cy := rect.X+rect.W/2, rect.Y+rect.H/2
		sin, cos := sincos(turn)
		var tri ui.Path
		for j, q := range [][2]float32{{-2.5, -4}, {3.5, 0}, {-2.5, 4}} {
			x, y := cx+q[0]*cos-q[1]*sin, cy+q[0]*sin+q[1]*cos
			if j == 0 {
				tri.MoveTo(x, y)
			} else {
				tri.LineTo(x, y)
			}
		}
		pt.FillPath(tri.Close(), color)
	})
}

// editor builds the field renaming l, in place of its name.
func (p *panel) editor(l *Layer) ui.Element {
	c, s, t := p.c, p.s, p.c.Theme()
	pad := t.Space(0.5)
	// The text stays where the name was, within the field's padding and
	// border.
	in := ui.TextInputBase(c, &s.draft).Grow(1).Shrink(1).MinWidth(t.Space(10)).Padding(pad, pad*2).
		Margin(-pad-1, -pad*2-1).Radius(t.Space(1)).Background(t.Background).Border(1, t.Accent).
		TextColor(t.Text).Label("Layer name")
	if s.fresh {
		in.Focus().SetTextSelection(0, utf8.RuneCountInString(s.draft))
		s.fresh = false
	}
	switch {
	case in.Focused():
		s.focused = true
	case s.focused:
		// The focus went elsewhere, as to another row: the name typed
		// stays.
		p.finish(l, true)
	}
	in.OnTextKey(func(k ui.TextKey) bool {
		switch {
		case k.Key == ui.KeyEnter && k.Mods == 0:
			s.draft, s.done = k.Text(), keep
		case k.Key == ui.KeyEscape && k.Mods == 0:
			s.done = cancel
		default:
			return false
		}
		return true
	})
	return in
}

// finish ends renaming l, keeping the name typed or not.
func (p *panel) finish(l *Layer, keep bool) {
	s := p.s
	if name := strings.TrimSpace(s.draft); keep && name != "" && name != l.Name {
		l.Name = name
		p.ev.renamed = l.ID
	}
	s.editing, s.done, s.focused = "", 0, false
}

// aim makes where the layers a drag from id carries would drop, with the
// pointer at (x, y) in row i, the panel's target: in the frame or group of
// the row, over the middle of its row, else between it and the row above
// or below, as deep as x is far to the right, among those the rows around
// allow.
func (p *panel) aim(i int, id string, x, y float32) {
	t, s, rows := p.c.Theme(), p.s, p.rows
	p.from = id
	carried := s.carried(id)
	r, h := rows[i], t.Space(rowHeight)
	if r.l.holds() && !p.within(i, carried) && y >= h/4 && y < h*3/4 {
		before := ""
		if len(r.l.Children) > 0 {
			before = r.l.Children[0].ID
		}
		p.tgt = target{ok: true, at: spot{r.id, before}, row: i, into: true}
		p.rest(r)
		return
	}
	below := y >= h/2
	a, b := i-1, i
	if below {
		a, b = i, i+1
	}
	lo, hi := 0, 0
	if b < len(rows) {
		lo = rows[b].depth
	}
	if a >= 0 {
		hi = rows[a].depth
	}
	if b < len(rows) && rows[b].up == a {
		// Below an unfolded frame or group, only its first layer.
		hi = lo
	}
	d := min(max(int((x-t.Space(inset))/t.Space(indent)), lo), hi)
	var at spot
	up := -1
	if b < len(rows) && d == rows[b].depth {
		up, at.before = rows[b].up, rows[b].id
	} else {
		// After the row above, or the frame or group it is in that is d
		// deep, which it ends.
		j := a
		for rows[j].depth > d {
			j = rows[j].up
		}
		up = rows[j].up
	}
	if p.within(up, carried) {
		p.tgt = target{}
		return
	}
	if up >= 0 {
		at.parent = rows[up].id
	}
	p.tgt = target{ok: true, at: at, row: i, depth: d, below: below}
}

// rest unfolds the folded frame or group of r once a drag rested on it
// long enough.
func (p *panel) rest(r row) {
	c, s := p.c, p.s
	if len(r.l.Children) == 0 || !s.folded[r.id] {
		return
	}
	p.resting = true
	if s.resting != r.id {
		s.resting, s.since = r.id, c.Now()
	}
	if wait := unfoldDelay - c.Now().Sub(s.since); wait > 0 {
		c.After(wait)
	} else {
		delete(s.folded, r.id)
		c.Invalidate()
	}
}

// line draws the line where the layers dragged would drop.
func (p *panel) line(pt *ui.Painter, t *ui.Theme) {
	g := p.tgt
	if !g.ok || g.into || g.rect.W == 0 {
		return
	}
	b := g.rect
	y := b.Y
	if g.below {
		y += b.H
	}
	x := b.X + t.Space(inset) + float32(g.depth)*t.Space(indent) + t.Space(arrowSize)
	pt.Stroke(ui.Rect{X: x, Y: y - 3, W: 6, H: 6}, t.Accent, 3, 1.5)
	pt.Fill(ui.Rect{X: x + 6, Y: y - 1, W: b.X + b.W - x - 6 - t.Space(1), H: 2}, t.Accent, 1)
}

// move moves the layers carried to at, unfolding the frame or group they
// went into.
func (p *panel) move(at spot, into bool) {
	s, layers := p.s, p.layers
	carried := s.carried(p.from)
	kids := children(layers, at.parent)
	if kids == nil {
		return
	}
	// Before the first layer from at.before on that does not move too.
	j := len(*kids)
	if at.before != "" {
		j = slices.IndexFunc(*kids, func(l Layer) bool { return l.ID == at.before })
	}
	for j >= 0 && j < len(*kids) && carried[(*kids)[j].ID] {
		j++
	}
	before := ""
	if j >= 0 && j < len(*kids) {
		before = (*kids)[j].ID
	}
	was := shape(*layers, 0, nil)
	ids := tops(*layers, carried, nil)
	var taken []Layer
	for _, id := range ids {
		sl, i := find(layers, id)
		taken = append(taken, (*sl)[i])
		*sl = slices.Delete(*sl, i, i+1)
	}
	// What was taken out moved the frame or group they go in.
	kids = children(layers, at.parent)
	j = len(*kids)
	if before != "" {
		j = slices.IndexFunc(*kids, func(l Layer) bool { return l.ID == before })
	}
	*kids = slices.Insert(*kids, j, taken...)
	if into {
		delete(s.folded, at.parent)
	}
	if !slices.Equal(was, shape(*layers, 0, nil)) {
		p.ev.moved = ids
	}
}

// remove removes the layers selected.
func (p *panel) remove() {
	s := p.s
	ids := tops(*p.layers, s.sel, nil)
	if len(ids) == 0 {
		return
	}
	var gone []string
	for _, id := range ids {
		sl, i := find(p.layers, id)
		gone = all((*sl)[i:i+1], gone)
		*sl = slices.Delete(*sl, i, i+1)
	}
	clear(s.sel)
	s.anchor, s.lead = "", ""
	p.ev.deleted, p.ev.selection = gone, true
}

// keys does what the keys pressed on the panel's list do.
func (p *panel) keys(list ui.Element) {
	s, rows := p.s, p.rows
	if len(rows) == 0 {
		return
	}
	at := p.index(s.lead)
	up, down := at-1, at+1
	if at < 0 {
		up, down = len(rows)-1, 0
	}
	switch {
	case list.Shortcut(0, ui.KeyUp) && up >= 0:
		p.only(up)
	case list.Shortcut(0, ui.KeyDown) && down < len(rows):
		p.only(down)
	case list.Shortcut(ui.Shift, ui.KeyUp) && up >= 0:
		p.span(up)
	case list.Shortcut(ui.Shift, ui.KeyDown) && down < len(rows):
		p.span(down)
	case list.Shortcut(0, ui.KeyLeft) && at >= 0:
		r := rows[at]
		switch {
		case len(r.l.Children) > 0 && !s.folded[r.id]:
			s.folded[r.id] = true
		case r.up >= 0:
			p.only(r.up)
		}
	case list.Shortcut(0, ui.KeyRight) && at >= 0:
		r := rows[at]
		switch {
		case len(r.l.Children) > 0 && s.folded[r.id]:
			delete(s.folded, r.id)
		case len(r.l.Children) > 0:
			p.only(at + 1)
		}
	case list.Shortcut(0, ui.KeyEnter) && at >= 0 && s.sel[s.lead]:
		s.rename(rows[at].l)
	case list.Shortcut(0, ui.KeyBackspace), list.Shortcut(0, ui.KeyDelete):
		p.remove()
	default:
		return
	}
	if i := p.index(s.lead); i >= 0 {
		s.list.ScrollIntoView(i)
	}
}

// index returns the row of the layer id, -1 for none.
func (p *panel) index(id string) int {
	return slices.IndexFunc(p.rows, func(r row) bool { return r.id == id })
}

// only selects row i alone.
func (p *panel) only(i int) {
	s, id := p.s, p.rows[i].id
	if len(s.sel) != 1 || !s.sel[id] {
		clear(s.sel)
		s.sel[id] = true
		p.ev.selection = true
	}
	s.anchor, s.lead = id, id
}

// toggle selects row i, or takes it out of the selection.
func (p *panel) toggle(i int) {
	s, id := p.s, p.rows[i].id
	if s.sel[id] {
		delete(s.sel, id)
	} else {
		s.sel[id] = true
	}
	s.anchor, s.lead = id, id
	p.ev.selection = true
}

// span selects the rows from the anchor to row i.
func (p *panel) span(i int) {
	s := p.s
	a := p.index(s.anchor)
	if a < 0 {
		p.only(i)
		return
	}
	clear(s.sel)
	for j := min(a, i); j <= max(a, i); j++ {
		s.sel[p.rows[j].id] = true
	}
	s.lead = p.rows[i].id
	p.ev.selection = true
}

// find returns the slice in layers holding the layer id, and its index
// there, or nil.
func find(layers *[]Layer, id string) (*[]Layer, int) {
	for i := range *layers {
		if (*layers)[i].ID == id {
			return layers, i
		}
		if sl, j := find(&(*layers)[i].Children, id); sl != nil {
			return sl, j
		}
	}
	return nil, -1
}

// children returns the layers in the frame or group id, those at the top
// for "", or nil.
func children(layers *[]Layer, id string) *[]Layer {
	if id == "" {
		return layers
	}
	sl, i := find(layers, id)
	if sl == nil {
		return nil
	}
	return &(*sl)[i].Children
}

// tops appends the IDs of the layers of ls in set, front-most first, but
// for those in another of set.
func tops(ls []Layer, set map[string]bool, out []string) []string {
	for i := range ls {
		if set[ls[i].ID] {
			out = append(out, ls[i].ID)
		} else {
			out = tops(ls[i].Children, set, out)
		}
	}
	return out
}

// collect appends the IDs of the layers of ls in set, at any depth,
// front-most first.
func collect(ls []Layer, set map[string]bool, out []string) []string {
	for i := range ls {
		if set[ls[i].ID] {
			out = append(out, ls[i].ID)
		}
		out = collect(ls[i].Children, set, out)
	}
	return out
}

// all appends the IDs of the layers of ls and those in them.
func all(ls []Layer, out []string) []string {
	for i := range ls {
		out = all(ls[i].Children, append(out, ls[i].ID))
	}
	return out
}

// place is a layer's place in a tree, as shape lists them.
type place struct {
	id    string
	depth int
}

// shape appends the layers of ls and those in them as they go, each with
// its depth, which tells the tree.
func shape(ls []Layer, depth int, out []place) []place {
	for i := range ls {
		out = shape(ls[i].Children, depth+1, append(out, place{ls[i].ID, depth}))
	}
	return out
}

// button creates a button turning a layer's on to true, named do, or back
// to false, named undo, its icon icons[0] or icons[1] while on; it shows
// only while on or show.
func button(c *ui.Context, do, undo string, icons [2]*ui.SVG, on, show bool, color ui.Color) ui.Element {
	t := c.Theme()
	label, icon := do, icons[0]
	if on {
		label, icon = undo, icons[1]
	}
	b := ui.ButtonBase(c).Size(t.Space(6), t.Space(6)).Shrink(0).Radius(t.Space(1)).Label(label).Tooltip(label)
	if !on && !show {
		b.Opacity(0)
	}
	if b.Hovered() {
		b.Background(t.Text.Alpha(0.08))
	}
	k := t.Space(4)
	return b.Draw(func(p *ui.Painter, r ui.Rect) {
		p.Icon(icon, ui.Rect{X: r.X + (r.W-k)/2, Y: r.Y + (r.H-k)/2, W: k, H: k}, color)
	})
}

// sincos returns the sine and cosine of deg degrees.
func sincos(deg float32) (float32, float32) {
	s, c := math.Sincos(float64(deg) * math.Pi / 180)
	return float32(s), float32(c)
}

func svg(body string) *ui.SVG {
	return ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">` + body + `</svg>`))
}

var (
	kinds = [...]*ui.SVG{
		Frame:     svg(`<path d="M8 3.5v17M16 3.5v17M3.5 8h17M3.5 16h17"/>`),
		Group:     svg(`<path d="M4 9V4h5M15 4h5v5M20 15v5h-5M9 20H4v-5"/>`),
		Rectangle: svg(`<rect x="4.5" y="4.5" width="15" height="15" rx="1.5"/>`),
		Ellipse:   svg(`<circle cx="12" cy="12" r="8"/>`),
		Text:      svg(`<path d="M5.5 7V5h13v2M12 5v14M9 19h6"/>`),
		Image:     svg(`<rect x="3.5" y="4.5" width="17" height="15" rx="2"/><circle cx="9" cy="10" r="1.6"/><path d="M20.5 16l-5-5L5 19.5"/>`),
		Path:      svg(`<path d="M12 3.5l6 8-6 9-6-9z"/><path d="M12 3.5v6.5"/><circle cx="12" cy="12" r="1.5"/>`),
	}
	locks = [2]*ui.SVG{
		svg(`<rect x="5" y="11" width="14" height="9.5" rx="2"/><path d="M8 11V8a4 4 0 0 1 7.6-1.8"/>`),
		svg(`<rect x="5" y="11" width="14" height="9.5" rx="2"/><path d="M8 11V8a4 4 0 0 1 8 0v3"/>`),
	}
	eyes = [2]*ui.SVG{
		svg(`<path d="M2.5 12S6 5.5 12 5.5 21.5 12 21.5 12 18 18.5 12 18.5 2.5 12 2.5 12z"/><circle cx="12" cy="12" r="3"/>`),
		svg(`<path d="M3 10c2.5 3.5 5.5 5 9 5s6.5-1.5 9-5M6.5 13.5L5 16M12 15v3M17.5 13.5L19 16"/>`),
	}
)
