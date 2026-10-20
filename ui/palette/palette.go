// Package palette shows a command palette over the window, as ⌘K opens
// one in Linear and ⌘⇧P in Visual Studio Code: a field to type in above
// the commands that match what is typed, best first, with the letters
// that match in bold, and each command's shortcut beside it.
//
//	if c.Shortcut(ui.Cmd, ui.KeyK) {
//		app.palette, app.query = true, ""
//	}
//	if i, ok := palette.Show(c, &app.palette, &app.query, app.items).Chosen(); ok {
//		app.run(i)
//	}
//
// The app keeps whether the palette shows, the text typed in it, and the
// commands it offers, which may follow the text, as files found for it.
package palette

import (
	"cmp"
	"slices"
	"strings"
	"unicode"

	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/shortcut"
)

// Item is a command the palette offers: its title, the section it is in,
// under a heading while nothing is typed, and the shortcut that runs it
// too, shown beside it.
type Item struct {
	Title    string
	Section  string
	Shortcut shortcut.Shortcut
}

// Events are what the user did with a palette in a frame.
type Events struct {
	chosen int
	ok     bool
}

// Chosen returns the item the user chose, with a click or Enter, as an
// index into the items given; the palette has closed.
func (e Events) Chosen() (int, bool) { return e.chosen, e.ok }

// row is a row of the palette: a heading, or an item with the runes of
// its title that match what is typed.
type row struct {
	heading string
	item    int
	matched []int
}

// state is what a palette keeps from frame to frame: its list's state,
// the row the arrows are on, the row last under the pointer, -1 for
// none, and the text the rows were found for.
type state struct {
	list      ui.ListState
	highlight int
	hovered   int
	query     string
}

// Show shows the palette while *open, editing *query, and offering items.
// Typing finds the items whose titles hold its letters in order, case
// aside: those starting with it first, then those with a word starting
// with it, those containing it, and those with its letters scattered, as
// "gtl" finds Go to Line, each tier with the letters closest together
// first and otherwise in the order given. With nothing typed, the items
// show in their order, under the headings of their sections.
//
// Up and Down move among the items, as the pointer does, and Enter or a
// click chooses one, which closes the palette; Escape, or a click outside
// it, closes it without choosing. The palette keeps the keyboard while it
// shows, and gives it back to what had it as it closes. Assistive
// technology sees a combo box, and follows the item the arrows are on.
func Show(c *ui.Context, open *bool, query *string, items []Item) Events {
	var ev Events
	if !*open {
		return ev
	}
	t := c.Theme()
	w, h := c.Size()
	ui.Overlay(c, func() {
		back := ui.Column(c).Absolute().Left(0).Top(0).Right(0).Bottom(0).AlignItems(ui.Center).
			Padding(h*0.12, 0, 0, 0).Background(ui.RGBA(0, 0, 0, 0.2)).Modal()
		back.Children(func() {
			panel := ui.Column(c).Width(min(600, w-32)).Radius(12).Clip().Background(t.Background).
				Border(1, t.Border).Shadow(0, 12, 32, 0, ui.RGBA(0, 0, 0, 0.25)).
				Role(ui.RoleDialog).Label("Command palette")
			st := ui.Local(panel, "palette.state", func() state { return state{hovered: -1} })
			rows := find(items, *query)
			if st.query != *query {
				st.query, st.highlight = *query, -1
				st.list.ScrollTo(0, ui.Start)
			}
			// Items that follow the text can change under the row the
			// arrows were on.
			if st.highlight = min(st.highlight, len(rows)-1); st.highlight < 0 || rows[st.highlight].heading != "" {
				st.highlight = next(rows, st.highlight, 1)
			}
			move := func(d int) {
				st.highlight = next(rows, st.highlight, d)
				st.list.ScrollIntoView(st.highlight)
			}
			if panel.OverlayShortcut(0, ui.KeyDown) {
				move(1)
			}
			if panel.OverlayShortcut(0, ui.KeyUp) {
				move(-1)
			}
			if panel.PressedOutside() || panel.OverlayShortcut(0, ui.KeyEscape) {
				*open = false
			}

			panel.Children(func() {
				input := ui.TextInputBase(c, query).AutoFocus().Padding(t.Space(3.5), t.Space(4)).FontSize(t.FontSize + 3).
					Placeholder("Type a command").Label("Command").Role(ui.RoleComboBox).Expanded(true)
				if input.Submitted() && st.highlight >= 0 {
					ev = Events{chosen: rows[st.highlight].item, ok: true}
					*open = false
				}
				ui.Divider(c)
				var active ui.Element
				st.list.Header = func(i int) bool { return rows[i].heading != "" }
				hovered := -1
				list := ui.List(c, &st.list, len(rows), func(i int) {
					r := rows[i]
					if r.heading != "" {
						ui.Text(c, r.heading).FontSize(t.FontSize-2).Bold().TextColor(t.TextMuted).
							Padding(t.Space(2), t.Space(3), t.Space(1), t.Space(3)).Background(t.Background)
						return
					}
					item := items[r.item]
					on := i == st.highlight
					line := ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Padding(t.Space(2), t.Space(3)).Radius(t.Radius)
					if line.Hovered() {
						hovered = i
					}
					if line.Clicked() {
						ev = Events{chosen: r.item, ok: true}
						*open = false
					}
					if on {
						line.Background(t.Accent).TextColor(t.AccentText)
						active = line
					}
					line.Children(func() {
						ui.RichText(c, spans(item.Title, r.matched, on, t)...).SingleLine().Grow(1)
						shortcut.Keys(c, item.Shortcut)
					})
				}).MaxHeight(min(420, h*0.6)).Padding(t.Space(1.5)).KeepFocus()
				list.Children(func() {
					if len(rows) == 0 {
						ui.Text(c, "No matching commands").TextColor(t.TextMuted).Padding(t.Space(3))
					}
				})
				// The pointer moving onto a row highlights it; resting there,
				// it leaves the arrows to move on.
				if hovered != st.hovered && hovered >= 0 {
					st.highlight = hovered
				}
				st.hovered = hovered
				if active.Valid() {
					input.ActiveDescendant(active)
				}
			})
		})
	})
	return ev
}

// next returns the first item row past row i going by d, or i where there
// is none.
func next(rows []row, i, d int) int {
	for j := i + d; j >= 0 && j < len(rows); j += d {
		if rows[j].heading == "" {
			return j
		}
	}
	return i
}

// find returns the rows for items and query: the items matching it, best
// first, or with no query all of them under headings.
func find(items []Item, query string) []row {
	q := []rune(strings.ToLower(strings.TrimSpace(query)))
	if len(q) == 0 {
		var rows []row
		section := ""
		for i, it := range items {
			if it.Section != section {
				section = it.Section
				if section != "" {
					rows = append(rows, row{heading: section})
				}
			}
			rows = append(rows, row{item: i})
		}
		return rows
	}
	type ranked struct {
		row
		tier, span int
	}
	var found []ranked
	for i, it := range items {
		if tier, matched, ok := match([]rune(it.Title), q); ok {
			found = append(found, ranked{row{item: i, matched: matched}, tier, matched[len(matched)-1] - matched[0]})
		}
	}
	slices.SortStableFunc(found, func(a, b ranked) int {
		return cmp.Or(cmp.Compare(a.tier, b.tier), cmp.Compare(a.span, b.span))
	})
	rows := make([]row, len(found))
	for i, f := range found {
		rows[i] = f.row
	}
	return rows
}

// match reports whether title holds the runes of q, in lower case, in
// order, how well, from 0 as its start to 3 scattered, and which runes of
// title they are.
func match(title, q []rune) (tier int, matched []int, ok bool) {
	lower := make([]rune, len(title))
	for i, r := range title {
		lower[i] = unicode.ToLower(r)
	}
	if at := indexFrom(lower, q, 0); at >= 0 {
		tier = 2
		for i := at; i >= 0; i = indexFrom(lower, q, i+1) {
			if wordStart(title, i) {
				at, tier = i, 1
				break
			}
		}
		if at == 0 {
			tier = 0
		}
		for i := range q {
			matched = append(matched, at+i)
		}
		return tier, matched, true
	}
	// Scattered, each rune at its first match, or where a word starts if
	// the runes after it still match after that, as the initials of "Go
	// to Line".
	j := 0
	for k, r := range q {
		at := -1
		for i := j; i < len(lower); i++ {
			if lower[i] != r {
				continue
			}
			if at < 0 {
				at = i
			}
			if wordStart(title, i) && subsequence(lower[i+1:], q[k+1:]) {
				at = i
				break
			}
		}
		if at < 0 {
			return 0, nil, false
		}
		matched = append(matched, at)
		j = at + 1
	}
	return 3, matched, true
}

// subsequence reports whether s holds the runes of q in order.
func subsequence(s, q []rune) bool {
	for _, r := range s {
		if len(q) > 0 && r == q[0] {
			q = q[1:]
		}
	}
	return len(q) == 0
}

// indexFrom returns the first index of q in s from from on, -1 for none.
func indexFrom(s, q []rune, from int) int {
	for i := from; i+len(q) <= len(s); i++ {
		if slices.Equal(s[i:i+len(q)], q) {
			return i
		}
	}
	return -1
}

// wordStart reports whether a word starts at rune i of s: its first rune,
// one after a rune that is not a letter or a digit, or an upper-case
// letter after a lower-case one, as in camelCase.
func wordStart(s []rune, i int) bool {
	if i == 0 {
		return true
	}
	prev, r := s[i-1], s[i]
	if !unicode.IsLetter(prev) && !unicode.IsDigit(prev) {
		return unicode.IsLetter(r) || unicode.IsDigit(r)
	}
	return unicode.IsLower(prev) && unicode.IsUpper(r)
}

// spans returns title in spans, the runes matched bold, and in the accent
// color in a row not highlighted.
func spans(title string, matched []int, on bool, t *ui.Theme) []ui.Span {
	if len(matched) == 0 {
		return []ui.Span{{Text: title}}
	}
	var out []ui.Span
	runes := []rune(title)
	start, k := 0, 0
	for start < len(runes) {
		hit := k < len(matched) && matched[k] == start
		end := start
		for end < len(runes) && (k < len(matched) && matched[k] == end) == hit {
			if hit {
				k++
			}
			end++
		}
		s := ui.Span{Text: string(runes[start:end])}
		if hit {
			s.Weight = 700
			if !on {
				s.Color = t.Accent
			}
		}
		out = append(out, s)
		start = end
	}
	return out
}
