// Package emoji picks emoji, as Slack's and macOS's pickers do: a field
// searching them by name, tabs going to their groups, the emoji used
// lately first, and the name of the emoji under the pointer below.
//
//	ui.Popover(c, button, &app.picking, func() {
//		emoji.Picker(c, &app.emoji).Size(352, 400)
//	})
//	if e, ok := app.emoji.Picked(); ok {
//		app.message += e
//	}
//
// It holds the 1,898 emoji of Unicode Emoji 15.1 and before, without
// skin tones, which systems' fonts draw.
package emoji

import (
	_ "embed"
	"slices"
	"strings"
	"sync"
	"unicode"

	"github.com/egoist/mygo/ui"
)

//go:embed emoji.txt
var data string

// maxRecent is how many emoji Recent keeps.
const maxRecent = 24

// group is a group of emoji, in their order: its name, the emoji of its
// tab, and its emoji.
type group struct {
	name, icon string
	emoji      []entry
}

// entry is an emoji, its name, and the words of its name and group,
// lowercase, for searching.
type entry struct {
	char, name string
	words      []string
}

var icons = map[string]string{
	"Smileys & Emotion": "😀", "People & Body": "👋", "Animals & Nature": "🐻", "Food & Drink": "🍔",
	"Travel & Places": "🚗", "Activities": "⚽", "Objects": "💡", "Symbols": "🔣", "Flags": "🏁",
}

// load parses data: its groups, and the emoji by character.
var load = sync.OnceValues(func() ([]group, map[string]entry) {
	var groups []group
	byChar := map[string]entry{}
	for line := range strings.Lines(data) {
		line = strings.TrimSpace(line)
		switch {
		case line == "" || strings.HasPrefix(line, "#"):
		case strings.HasPrefix(line, "= "):
			name := line[2:]
			groups = append(groups, group{name: name, icon: icons[name]})
		default:
			char, name, _ := strings.Cut(line, " ")
			g := &groups[len(groups)-1]
			e := entry{char: char, name: name, words: words(name + " " + g.name)}
			g.emoji = append(g.emoji, e)
			byChar[char] = e
		}
	}
	return groups, byChar
})

// words returns the words of s, lowercase.
func words(s string) []string {
	return strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
}

// State is what a picker keeps: the search, the emoji used lately, and
// the emoji picked in the frame. Keep it in the app's state, one for each
// picker.
type State struct {
	// Recent is the emoji picked lately, the latest first, which the
	// picker shows first. Save it to keep it.
	Recent []string
	query  string
	list   ui.ListState
	picked string
	hover  string
	// refocus gives the search field the focus back in the next frame,
	// after a click on a tab or an emoji took it, so typing searches.
	refocus bool
}

// Picked returns the emoji picked in the picker built last, in this
// frame.
func (s *State) Picked() (string, bool) { return s.picked, s.picked != "" }

func (s *State) pick(char string) {
	s.picked = char
	s.Recent = slices.Insert(slices.DeleteFunc(s.Recent, func(r string) bool { return r == char }), 0, char)
	s.Recent = s.Recent[:min(len(s.Recent), maxRecent)]
}

// row is a row of the picker's list: the header of a group, or its emoji
// from the row's.
type row struct {
	header string
	group  int
	emoji  []entry
}

// rows lays out the groups in rows of cols emoji, after a header each,
// with the emoji used lately first, or the emoji matching query in one
// group.
func (s *State) rows(cols int) (rows []row) {
	groups, byChar := load()
	add := func(header string, group int, emoji []entry) {
		if len(emoji) == 0 {
			return
		}
		rows = append(rows, row{header: header, group: group})
		for chunk := range slices.Chunk(emoji, cols) {
			rows = append(rows, row{group: group, emoji: chunk})
		}
	}
	if q := words(s.query); len(q) > 0 {
		var found []entry
		for _, g := range groups {
			for _, e := range g.emoji {
				if matches(e.words, q) {
					found = append(found, e)
				}
			}
		}
		add("Search Results", -1, found)
		return rows
	}
	var recent []entry
	for _, r := range s.Recent {
		if e, ok := byChar[r]; ok {
			recent = append(recent, e)
		}
	}
	add("Frequently Used", -1, recent)
	for i, g := range groups {
		add(g.name, i, g.emoji)
	}
	return rows
}

// matches reports whether every word of query starts a word of words.
func matches(words, query []string) bool {
	for _, q := range query {
		if !slices.ContainsFunc(words, func(w string) bool { return strings.HasPrefix(w, q) }) {
			return false
		}
	}
	return true
}

// Picker creates an emoji picker of which s keeps the state: a search
// field, which takes the keyboard focus, a tab for each group, the emoji
// in a list built only as it shows, and the name of the emoji under the
// pointer. A click picks an emoji, as Enter in the field picks the first
// found; Picked reports it. Give it a size.
func Picker(c *ui.Context, s *State) ui.Element {
	t := c.Theme()
	groups, byChar := load()
	s.picked = ""
	cell := t.Space(9)
	picker := ui.Column(c).Clip()
	w := picker.LayoutBounds().W - 2*t.Space(2)
	cols := max(1, int(w/cell))
	rows := s.rows(cols)
	s.list.Header = func(i int) bool { return rows[i].header != "" }
	hover := ""
	picker.Children(func() {
		ui.Box(c).Padding(t.Space(2), t.Space(2), t.Space(1)).Children(func() {
			// A SearchField would take the focus in its box, not its text.
			search := ui.TextInput(c, &s.query).Placeholder("Search").Label("Search emoji").AutoFocus()
			if s.refocus {
				search.Focus()
				s.refocus = false
			}
			if search.Shortcut(0, ui.KeyEscape) && s.query != "" {
				s.query = ""
				search.SetChanged()
			}
			if search.Changed() {
				s.list.ScrollTo(0, ui.Start)
			}
			if search.Submitted() && len(rows) > 1 {
				s.pick(rows[1].emoji[0].char)
			}
		})
		current := -1
		if first, last := s.list.Visible(); first <= last && first < len(rows) {
			current = rows[first].group
		}
		ui.Row(c).Padding(0, t.Space(2)).Justify(ui.SpaceBetween).Children(func() {
			for i, g := range groups {
				tab := ui.ButtonBase(c).Size(t.Space(7), t.Space(7)).Radius(t.Space(1.5)).Label(g.name)
				if i == current && s.query == "" {
					tab.Background(t.Text.Alpha(0.1))
				} else {
					tab.Opacity(0.6)
				}
				if tab.Clicked() {
					s.query, s.refocus = "", true
					s.list.ScrollTo(slices.IndexFunc(rows, func(r row) bool { return r.group == i }), ui.Start)
				}
				tab.Children(func() { ui.Text(c, g.icon).FontSize(t.FontSize + 2) })
			}
		})
		ui.Divider(c).Margin(t.Space(1), 0, 0)
		ui.List(c, &s.list, len(rows), func(i int) {
			r := rows[i]
			if r.header != "" {
				ui.Text(c, r.header).Bold().FontSize(t.FontSize-1).TextColor(t.TextMuted).
					Padding(t.Space(2), t.Space(1), t.Space(1)).Background(t.Background)
				return
			}
			ui.Row(c).Children(func() {
				for _, e := range r.emoji {
					b := ui.ButtonBase(c.Key(e.char)).Size(cell, cell).Radius(t.Space(1.5)).Label(e.name)
					if b.Hovered() {
						b.Background(t.Text.Alpha(0.08))
						hover = e.char
					}
					if b.Clicked() {
						s.pick(e.char)
						s.refocus = true
					}
					b.Children(func() { ui.Text(c, e.char).FontSize(t.FontSize * 1.7) })
				}
			})
		}).Grow(1).Padding(0, t.Space(2), t.Space(2))
		if len(rows) == 0 {
			ui.Text(c, "No Emoji Found").TextColor(t.TextMuted).Padding(t.Space(4)).AlignSelf(ui.Center)
		}
		ui.Divider(c)
		if hover != "" {
			s.hover = hover
		}
		ui.Row(c).Height(t.Space(11)).Padding(0, t.Space(3)).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
			if e, ok := byChar[s.hover]; ok {
				ui.Text(c, e.char).FontSize(t.FontSize * 2)
				ui.Text(c, e.name).SingleLine().Shrink(1)
			} else {
				ui.Text(c, "Pick an emoji").TextColor(t.TextMuted)
			}
		})
	})
	return picker
}
