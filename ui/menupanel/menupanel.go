// Package menupanel builds menus of native UI for the panels of menu bar
// apps, as macOS's own menus and Control Center look: rows of a title,
// with an icon before it, a check mark, and a value or a shortcut after
// it; headers and separators between groups; submenus that open in place,
// below their row; and rows of your own, as a slider. The pointer and the
// keyboard choose rows as in a menu.
//
//	menupanel.View(c, &app.menu, func(m *menupanel.Menu) {
//		m.Header("Wi-Fi")
//		for i, n := range app.networks {
//			if m.Item(n.Name).Checked(i == app.joined).Detail(n.Signal).Chosen() {
//				app.join(i)
//			}
//		}
//		m.Separator()
//		m.Row(func() { ui.Slider(c, &app.volume, 0, 1).Label("Volume") })
//		if m.Item("Quit").Shortcut(ui.Cmd, ui.KeyQ).Chosen() {
//			mygo.App.Quit()
//		}
//	})
//
// As with ui.Menu, Chosen reports a choice in the frame after the user
// made it, and only for the item of the same place and title, so that a
// menu changing in between reports none.
package menupanel

import (
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/shortcut"
)

// State is what a menu keeps from frame to frame: the row highlighted, the
// submenus open, and the item chosen. Keep it in the app's state, one for
// each menu.
type State struct {
	active int // the item highlighted, -1 for none
	hover  int // the item the pointer was over in the last frame
	chosen int // the item chosen, for the next frame to report
	label  string
	open   map[string]bool
}

// Menu is a menu as View builds it, to add rows to.
type Menu struct {
	c       *ui.Context
	s       *State
	chosen  int
	label   string
	entries []*entry
	items   int // items so far, which numbers the next
	depth   int
	parent  int // the submenu being built, -1 outside any
}

type kind uint8

const (
	item kind = iota
	header
	separator
	row
)

// entry is a row of a menu as built.
type entry struct {
	kind     kind
	title    string
	index    int // an item's number, in the order built
	parent   int // the submenu it is in, -1 for none
	depth    int
	icon     *ui.SVG
	detail   string
	keys     shortcut.Shortcut
	checked  int8 // 0 none, 1 off, 2 on
	disabled bool
	submenu  bool
	build    func()
}

// Item is an item of a menu, to set up and ask whether it was chosen.
type Item struct {
	m *Menu
	e *entry
}

// Item adds an item titled title.
func (m *Menu) Item(title string) Item {
	e := &entry{kind: item, title: title, index: m.items, parent: m.parent, depth: m.depth}
	m.items++
	m.entries = append(m.entries, e)
	return Item{m, e}
}

// Header adds the title of the group of rows after it.
func (m *Menu) Header(title string) {
	m.entries = append(m.entries, &entry{kind: header, title: title, depth: m.depth})
}

// Separator adds a line between groups of rows.
func (m *Menu) Separator() {
	m.entries = append(m.entries, &entry{kind: separator, depth: m.depth})
}

// Row adds a row that build builds, as a slider or a meter, inset as the
// items' titles are; the keyboard passes it by.
func (m *Menu) Row(build func()) {
	m.entries = append(m.entries, &entry{kind: row, build: build, depth: m.depth})
}

// Submenu adds an item titled title that opens and closes the rows build
// adds, below it and indented, as a click on it or Right and Left do. The
// rows are built only while open.
func (m *Menu) Submenu(title string, build func(m *Menu)) Item {
	it := m.Item(title)
	it.e.submenu = true
	if m.s.open[title] {
		parent := m.parent
		m.parent, m.depth = it.e.index, m.depth+1
		build(m)
		m.parent, m.depth = parent, m.depth-1
	}
	return it
}

// Icon shows icon before the title, in the color of the text.
func (it Item) Icon(icon *ui.SVG) Item { it.e.icon = icon; return it }

// Detail shows text after the title, muted, as a value or a status:
// "Connected", "42%".
func (it Item) Detail(text string) Item { it.e.detail = text; return it }

// Checked shows a check mark before the item while on, and room for one
// while off.
func (it Item) Checked(on bool) Item {
	it.e.checked = 1
	if on {
		it.e.checked = 2
	}
	return it
}

// Shortcut shows the keys that do the same as the item, at its right; as
// with ui.MenuItem, the view handles them itself.
func (it Item) Shortcut(mods ui.Modifiers, key ui.Key) Item {
	it.e.keys = shortcut.Shortcut{Mods: mods, Key: key}
	return it
}

// Disabled shows the item muted, and lets no one choose it.
func (it Item) Disabled(d bool) Item { it.e.disabled = d; return it }

// Chosen reports whether the item was chosen, in the frame after the user
// chose it. A submenu's item never is: choosing it opens it.
func (it Item) Chosen() bool {
	e := it.e
	return !e.disabled && !e.submenu && e.index == it.m.chosen && e.title == it.m.label
}

// View creates a menu of the rows build adds, of which s says what is
// highlighted and open. It takes the keyboard focus as it appears, as a
// menu does: Up and Down move from item to item, Return and Space choose
// one, and Right and Left open and close a submenu. The pointer
// highlights the item it is over, and a click chooses it. Give it the
// width of the panel; it is as tall as its rows.
func View(c *ui.Context, s *State, build func(m *Menu)) ui.Element {
	t := c.Theme()
	if s.open == nil {
		*s = State{active: -1, hover: -1, chosen: -1, open: map[string]bool{}}
	}
	m := &Menu{c: c, s: s, chosen: s.chosen, label: s.label, parent: -1}
	s.chosen, s.label = -1, ""
	build(m)
	items := make([]*entry, m.items)
	checks := false
	for _, e := range m.entries {
		if e.kind == item {
			items[e.index] = e
			checks = checks || e.checked > 0
		}
	}
	if s.active >= len(items) {
		s.active = -1
	}

	panel := ui.Column(c).Padding(t.Space(1.25), 0).Role(ui.RoleMenu).Focusable().AutoFocus()
	s.keys(c, panel, items)
	hovered := -1
	panel.Children(func() {
		for i, e := range m.entries {
			switch e.kind {
			case header:
				ui.Text(c.Key(i), e.title).Bold().FontSize(t.FontSize-2).TextColor(t.TextMuted).
					NoWrap().Padding(t.Space(1.5), inset(t, e.depth, checks), t.Space(0.5), inset(t, e.depth, checks)).Role(ui.RoleHeading)
			case separator:
				ui.Box(c.Key(i)).Height(1).Background(t.Border).Margin(t.Space(1.25), t.Space(2.5))
			case row:
				ui.Column(c.Key(i)).Padding(t.Space(1), inset(t, e.depth, checks)).Children(e.build)
			case item:
				if s.itemRow(c.Key(i), e, checks) {
					hovered = e.index
				}
			}
		}
	})
	// The pointer moves the highlight as it comes to an item, or leaves
	// them, so that the keys can move it from under a pointer at rest.
	if hovered != s.hover {
		s.active, s.hover = hovered, hovered
	}
	return panel
}

// inset is how far the text of rows of depth starts from the panel's
// edge: past the highlight's own inset, a step further for each submenu,
// and past the check marks of a menu with checks.
func inset(t *ui.Theme, depth int, checks bool) float32 {
	x := t.Space(3.5) + float32(depth)*t.Space(4)
	if checks {
		x += checkW(t)
	}
	return x
}

// checkW is the room a check mark takes before a title, with its gap.
func checkW(t *ui.Theme) float32 { return t.Space(4.5) }

// keys moves the highlight among the items enabled, and chooses, opens and
// closes them, with the keys pressed in the panel.
func (s *State) keys(c *ui.Context, panel ui.Element, items []*entry) {
	step := func(d int) {
		for i, n := s.active, 0; n < len(items); n++ {
			i = (i + d + len(items)) % len(items)
			if !items[i].disabled {
				s.active = i
				return
			}
		}
	}
	if panel.Shortcut(0, ui.KeyDown) {
		step(1)
	}
	if panel.Shortcut(0, ui.KeyUp) {
		if s.active < 0 {
			s.active = 0
		}
		step(-1)
	}
	if s.active < 0 {
		return
	}
	e := items[s.active]
	if panel.Shortcut(0, ui.KeyEnter) || panel.Shortcut(0, ui.KeySpace) {
		s.choose(c, e)
	}
	if panel.Shortcut(0, ui.KeyRight) && e.submenu {
		s.open[e.title] = true
	}
	if panel.Shortcut(0, ui.KeyLeft) {
		switch {
		case e.submenu && s.open[e.title]:
			delete(s.open, e.title)
		case e.parent >= 0:
			s.active = e.parent
			delete(s.open, items[e.parent].title)
		}
	}
}

// choose chooses e: opens or closes it, a submenu, or has the next frame
// report it.
func (s *State) choose(c *ui.Context, e *entry) {
	switch {
	case e.disabled:
	case e.submenu:
		if s.open[e.title] {
			delete(s.open, e.title)
		} else {
			s.open[e.title] = true
		}
	default:
		s.chosen, s.label = e.index, e.title
		c.Invalidate()
	}
}

// itemRow builds the row of e, with room for check marks if checks, and
// reports whether the pointer is over it.
func (s *State) itemRow(c *ui.Context, e *entry, checks bool) bool {
	t := c.Theme()
	on := s.active == e.index && !e.disabled
	text, muted := t.Text, t.TextMuted
	switch {
	case on:
		text, muted = t.AccentText, t.AccentText
	case e.disabled:
		text = t.TextMuted
	}
	role := ui.RoleMenuItem
	if e.checked > 0 {
		role = ui.RoleMenuItemCheckBox
	}
	// The row's margin is the highlight's inset; the check mark goes in
	// the inset of the text.
	start := inset(t, e.depth, checks) - t.Space(1.25)
	if checks {
		start -= checkW(t)
	}
	r := ui.Row(c).Height(t.Space(5.5)).Margin(0, t.Space(1.25)).Padding(0, t.Space(2.25), 0, start).
		Gap(t.Space(1.5)).AlignItems(ui.Center).Radius(t.Space(1)).Role(role).Label(e.title).Checked(e.checked == 2).
		Disabled(e.disabled)
	if on {
		r.Background(t.Accent)
	}
	if r.Clicked() {
		s.choose(c, e)
	}
	r.Children(func() {
		if checks {
			mark := ui.Box(c).Size(checkW(t)-t.Space(1.5), t.Space(3))
			if e.checked == 2 {
				mark.Draw(func(p *ui.Painter, r ui.Rect) {
					p.StrokePolyline([][2]float32{
						{r.X + r.W*0.15, r.Y + r.H*0.55}, {r.X + r.W*0.4, r.Y + r.H*0.8}, {r.X + r.W*0.85, r.Y + r.H*0.2},
					}, 1.6, text)
				})
			}
		}
		if e.icon != nil {
			ui.Icon(c, e.icon).Size(t.Space(4), t.Space(4)).TextColor(text)
		}
		ui.Text(c, e.title).TextColor(text).SingleLine().Grow(1).Shrink(1)
		if e.detail != "" {
			ui.Text(c, e.detail).TextColor(muted).NoWrap()
		}
		if k := e.keys.MenuText(); k != "" {
			ui.Text(c, k).TextColor(muted).NoWrap()
		}
		if e.submenu {
			open := s.open[e.title]
			ui.Box(c).Size(t.Space(2.5), t.Space(2.5)).Draw(func(p *ui.Painter, r ui.Rect) {
				pts := [][2]float32{{r.X + r.W*0.3, r.Y + r.H*0.1}, {r.X + r.W*0.7, r.Y + r.H*0.5}, {r.X + r.W*0.3, r.Y + r.H*0.9}}
				if open {
					pts = [][2]float32{{r.X + r.W*0.1, r.Y + r.H*0.3}, {r.X + r.W*0.5, r.Y + r.H*0.7}, {r.X + r.W*0.9, r.Y + r.H*0.3}}
				}
				p.StrokePolyline(pts, 1.5, muted)
			})
		}
	})
	return r.Hovered()
}
