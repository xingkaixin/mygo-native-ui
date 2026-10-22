// Package charmap shows Unicode's characters, as macOS's Character Viewer
// does: a sidebar of groups, as Arrows and Hiragana, the characters of the
// group chosen in a grid, a field searching them by name, by code point or
// by the character pasted, and beside them what the character chosen is:
// its name, its code point, its bytes in UTF-8 and UTF-16, its category,
// its block and its HTML entity.
//
//	charmap.View(c, &app.chars).Grow(1)
//	if r, ok := app.chars.Picked(); ok {
//		app.text += string(r)
//	}
//
// Names come from golang.org/x/text/unicode/runenames, indexed in the
// background once for all viewers. Emoji are left out: ui/emoji picks them.
package charmap

import (
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode/utf16"

	"github.com/egoist/mygo/ui"
)

// maxRecent is how many characters Recent keeps.
const maxRecent = 12

// State is what a viewer keeps: the group chosen, the search, the
// character chosen, the characters used lately, and the character picked
// in the frame. Keep it in the app's state, one for each viewer.
type State struct {
	// Recent is the characters picked lately, the latest first, which the
	// viewer shows above the grid. Save it to keep it.
	Recent []rune
	group  string
	query  string
	chosen rune
	picked rune
	grid   ui.GridState
	// found is the characters found for searched, while the names were
	// indexed or not.
	found    []rune
	searched string
	indexed  bool
}

// Picked returns the character picked in the viewer built last, in this
// frame.
func (s *State) Picked() (rune, bool) { return s.picked, s.picked != 0 }

func (s *State) pick(r rune) {
	s.picked, s.chosen = r, r
	s.Recent = slices.Insert(slices.DeleteFunc(s.Recent, func(x rune) bool { return x == r }), 0, r)
	s.Recent = s.Recent[:min(len(s.Recent), maxRecent)]
}

// chars returns the characters the grid shows: those found for the
// search, or those of the group.
func (s *State) chars(group []rune, names []entry, indexed bool) []rune {
	if s.query == "" {
		return group
	}
	if s.query != s.searched || indexed != s.indexed {
		s.found, s.searched, s.indexed = search(s.query, names), s.query, indexed
	}
	return s.found
}

// View creates a character viewer of which s keeps the state: a sidebar
// of groups, a search field, the characters used lately, the characters
// of the group or those found in a grid built only as it shows, and what
// the character chosen is, with Copy. A click chooses a character, as the
// arrows do in the grid, and a search chooses the first found; a double
// click or Enter, in the grid or the field, picks it, which Picked
// reports. Give it a size, or grow it in its parent.
func View(c *ui.Context, s *State) ui.Element {
	t := c.Theme()
	groups := load()
	names, indexed := nameIndex()
	s.picked = 0
	g := slices.IndexFunc(groups, func(g group) bool { return g.name == s.group })
	if g < 0 {
		g, s.group = 0, groups[0].name
	}
	if s.query != "" && !indexed {
		c.After(50 * time.Millisecond)
	}
	view := ui.Row(c).AlignItems(ui.Stretch).Clip().Background(t.Background)
	view.Children(func() {
		shown := s.group
		if s.query != "" {
			shown = ""
		}
		sidebar := ui.Sidebar(c, &shown, func() {
			for _, section := range []string{"Scripts", "Symbols"} {
				ui.SidebarSection(c, section, nil, func() {
					for _, g := range groups {
						if g.section == section {
							ui.SidebarItem(c, g.name, nil, g.name)
						}
					}
				})
			}
		}).Width(t.Space(52))
		if sidebar.Changed() {
			s.group, s.query = shown, ""
			s.grid.ScrollTo(0, ui.Start)
		}
		ui.Divider(c)
		ui.Column(c).Grow(1).MinWidth(0).Children(func() {
			var chars []rune
			ui.Row(c).Padding(t.Space(2), t.Space(3)).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
				title := s.group
				if s.query != "" {
					title = "Search Results"
				}
				ui.Text(c, title).Bold().SingleLine().Grow(1).Shrink(1)
				field := ui.SearchField(c, &s.query).Width(t.Space(55)).Label("Search characters")
				changed, submitted := field.Changed(), field.Submitted()
				chars = s.chars(groups[g].chars, names, indexed)
				if changed {
					s.grid.ScrollTo(0, ui.Start)
					if s.query != "" && len(chars) > 0 {
						s.chosen = chars[0]
					}
				}
				if submitted && s.query != "" && len(chars) > 0 {
					s.pick(s.chosen)
				}
			})
			if len(s.Recent) > 0 {
				s.recent(c)
			}
			ui.Divider(c)
			ui.Row(c).Grow(1).MinHeight(0).AlignItems(ui.Stretch).Children(func() {
				s.characters(c, chars, indexed)
				ui.Divider(c)
				details(c, s.chosen)
			})
		})
	})
	return view
}

// characters shows chars in a grid, which a click chooses from and a
// double click or Enter picks from.
func (s *State) characters(c *ui.Context, chars []rune, indexed bool) {
	t := c.Theme()
	if len(chars) == 0 {
		text := "No Characters Found"
		if !indexed {
			text = "Searching…"
		}
		ui.Text(c, text).TextColor(t.TextMuted).Padding(t.Space(4)).Grow(1).TextAlign(ui.Center)
		return
	}
	at := slices.Index(chars, s.chosen)
	s.grid.Selected = &at
	s.grid.Label = func(i int) string { return name(chars[i]) }
	grid := ui.GridView(c, &s.grid, len(chars), t.Space(10), t.Space(10), func(i int) {
		ui.Box(c).Grow(1).Justify(ui.Center).AlignItems(ui.Center).Children(func() {
			ui.Text(c, glyph(chars[i])).FontSize(t.FontSize * 1.6)
		})
	}).Grow(1).Padding(t.Space(2))
	if grid.Changed() && at >= 0 {
		s.chosen = chars[at]
	}
	if grid.Submitted() && at >= 0 {
		s.pick(chars[at])
	}
}

// recent shows the characters picked lately in a row, which a click
// chooses from and a double click picks from.
func (s *State) recent(c *ui.Context) {
	t := c.Theme()
	var picked rune
	ui.Row(c).Padding(0, t.Space(3), t.Space(2)).Gap(t.Space(1)).AlignItems(ui.Center).Clip().Children(func() {
		ui.Text(c, "Recently Used").FontSize(t.FontSize - 1).TextColor(t.TextMuted).SingleLine().MarginEnd(t.Space(1))
		for _, r := range s.Recent {
			b := ui.ButtonBase(c.Key(r)).Size(t.Space(8), t.Space(8)).Radius(t.Space(1.5)).Label(name(r)).KeepFocus()
			switch {
			case r == s.chosen:
				b.Background(t.Text.Alpha(0.1))
			case b.Hovered():
				b.Background(t.Text.Alpha(0.06))
			}
			if b.Clicked() {
				s.chosen = r
			}
			if b.DoubleClicked() {
				picked = r
			}
			b.Children(func() { ui.Text(c, glyph(r)).FontSize(t.FontSize * 1.3) })
		}
	})
	if picked != 0 {
		s.pick(picked)
	}
}

// details shows what r is: its glyph large, its name, its code point, its
// bytes, its category, its block and its HTML entity, and Copy, which
// copies it.
func details(c *ui.Context, r rune) {
	t := c.Theme()
	ui.Column(c).Width(t.Space(64)).Shrink(0).Padding(t.Space(4)).Gap(t.Space(3)).Children(func() {
		if r == 0 {
			ui.Text(c, "No Character Chosen").TextColor(t.TextMuted).AlignSelf(ui.Center).MarginY(t.Space(8))
			return
		}
		ui.Box(c).Height(t.Space(36)).Radius(t.Space(2)).Background(t.Surface).Justify(ui.Center).AlignItems(ui.Center).Children(func() {
			ui.Text(c, glyph(r)).FontSize(t.Space(18))
		})
		ui.Text(c, name(r)).Bold().Selectable()
		var units []string
		for _, u := range utf16.AppendRune(nil, r) {
			units = append(units, fmt.Sprintf("%04X", u))
		}
		ui.Column(c).Gap(t.Space(1.5)).Children(func() {
			fact(c, "Code point", fmt.Sprintf("U+%04X", r))
			fact(c, "UTF-8", fmt.Sprintf("% X", string(r)))
			fact(c, "UTF-16", strings.Join(units, " "))
			fact(c, "Category", category(r))
			if b := blockOf(r); b != "" {
				fact(c, "Block", b)
			}
			fact(c, "HTML", fmt.Sprintf("&#x%X;", r))
		})
		if ui.Button(c, "Copy").AlignSelf(ui.Start).Clicked() {
			c.WriteClipboard(string(r))
		}
	})
}

// fact shows a fact of a character: what it is, and its value, which can
// be selected.
func fact(c *ui.Context, label, value string) {
	t := c.Theme()
	ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Start).Children(func() {
		ui.Text(c, label).FontSize(t.FontSize - 1).TextColor(t.TextMuted).Width(t.Space(19)).Shrink(0)
		ui.Text(c, value).FontSize(t.FontSize - 1).Selectable().Grow(1).Shrink(1)
	})
}
