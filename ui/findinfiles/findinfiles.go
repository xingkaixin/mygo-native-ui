// Package findinfiles searches and replaces across files, as the Search
// view of VS Code: a find field with toggles to match case, whole words
// and regular expressions, a replace field shown with the arrow beside
// them, how many results in how many files, and the results under each
// file, the match lit in its line. With something to replace, each match
// shows struck out with what replaces it after it.
//
//	ev := findinfiles.View(c, &app.search, &app.files).Grow(1)
//	if ev.Opened.Path != "" {
//		app.open(ev.Opened.Path, ev.Opened.Line, ev.Opened.Col)
//	}
//
// A click, or Enter, on a match reports it Opened. A click on a file's
// row folds its matches. Over a match, a button replaces it; over a
// file's row, all of that file's; and Replace All replaces every match.
// Replacing changes the files' texts. With regular expressions, $1 and
// ${name} put in what the groups matched.
package findinfiles

import (
	"fmt"
	"path"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
)

// File is a file searched: its path, with slashes, and its text.
type File struct {
	Path, Text string
}

// Match is where a match is: its file, its line and its column, from 1,
// in runes.
type Match struct {
	Path      string
	Line, Col int
}

// State is what a search keeps: what it finds and replaces, its toggles,
// and the files folded. Keep it in the app's state, one for each view.
type State struct {
	find, replace string
	matchCase     bool
	word, regex   bool
	replacing     bool
	folded        map[string]bool
	list          ui.ListState
	row           int
	setup         bool
}

// Events is a search view, and what the user did with it this frame:
// Opened a match, and how many matches were Replaced.
type Events struct {
	ui.Element
	Opened   Match
	Replaced int
}

// hit is a match in a file: its bytes in the text, and its line, the
// line's text and its column.
type hit struct {
	start, end int
	line, col  int
	text       string
	lineStart  int
}

// group is a file's matches.
type group struct {
	file int
	hits []hit
}

// row is a row of the results: a file, or a match of one.
type row struct {
	group, hit int
}

// pattern returns the expression the state searches with, or an error
// for an expression not valid.
func (s *State) pattern() (*regexp.Regexp, error) {
	p := s.find
	if !s.regex {
		p = regexp.QuoteMeta(p)
	}
	if s.word {
		p = `\b(?:` + p + `)\b`
	}
	if !s.matchCase {
		p = `(?i)` + p
	}
	return regexp.Compile(p)
}

func search(files []File, re *regexp.Regexp) []group {
	var out []group
	for i, f := range files {
		var g group
		line, lineStart := 1, 0
		at := 0
		for _, m := range re.FindAllStringIndex(f.Text, -1) {
			if m[0] == m[1] {
				continue
			}
			for at < m[0] {
				if f.Text[at] == '\n' {
					line++
					lineStart = at + 1
				}
				at++
			}
			end := strings.IndexByte(f.Text[lineStart:], '\n')
			if end < 0 {
				end = len(f.Text) - lineStart
			}
			g.hits = append(g.hits, hit{
				start: m[0], end: m[1], line: line, lineStart: lineStart,
				col:  utf8.RuneCountInString(f.Text[lineStart:m[0]]) + 1,
				text: f.Text[lineStart : lineStart+end],
			})
		}
		if len(g.hits) > 0 {
			g.file = i
			out = append(out, g)
		}
	}
	return out
}

// with returns what replaces the match h of text.
func (s *State) with(re *regexp.Regexp, text string, h hit) string {
	if !s.regex {
		return s.replace
	}
	m := re.FindStringSubmatchIndex(text[h.start:h.end])
	if m == nil {
		return s.replace
	}
	return string(re.ExpandString(nil, s.replace, text[h.start:h.end], m))
}

// apply replaces the hits of f, which are in order, and returns how many.
func (s *State) apply(re *regexp.Regexp, f *File, hits []hit) int {
	var b strings.Builder
	at := 0
	for _, h := range hits {
		b.WriteString(f.Text[at:h.start])
		b.WriteString(s.with(re, f.Text, h))
		at = h.end
	}
	b.WriteString(f.Text[at:])
	f.Text = b.String()
	return len(hits)
}

// View creates the search of *files. Give it a height, or Grow.
func View(c *ui.Context, s *State, files *[]File) Events {
	t := c.Theme()
	if !s.setup {
		s.setup, s.row, s.folded = true, -1, map[string]bool{}
		s.list.Selected = &s.row
	}
	view := ui.Column(c).Gap(t.Space(2))
	ev := Events{Element: view}

	var groups []group
	var rows []row
	re, err := s.pattern()
	if err == nil && s.find != "" {
		groups = search(*files, re)
		for gi, g := range groups {
			rows = append(rows, row{gi, -1})
			if s.folded[(*files)[g.file].Path] {
				continue
			}
			for hi := range g.hits {
				rows = append(rows, row{gi, hi})
			}
		}
	}
	total := 0
	for _, g := range groups {
		total += len(g.hits)
	}
	s.row = min(s.row, len(rows)-1)
	s.list.Key = func(i int) any {
		r := rows[i]
		g := groups[r.group]
		if r.hit < 0 {
			return (*files)[g.file].Path
		}
		return fmt.Sprint((*files)[g.file].Path, ":", g.hits[r.hit].start)
	}
	s.list.Header = func(i int) bool { return rows[i].hit < 0 }
	replace := s.replacing && s.find != "" && err == nil

	view.Children(func() {
		ui.Row(c).Gap(t.Space(1)).AlignItems(ui.Start).Children(func() {
			arrow := ui.ButtonBase(c).Size(t.Space(5), t.Space(7)).Radius(t.Space(1)).Label("Toggle Replace").KeepFocus()
			if arrow.Clicked() {
				s.replacing = !s.replacing
			}
			if arrow.Hovered() {
				arrow.Background(t.Text.Alpha(0.07))
			}
			open := s.replacing
			arrow.Draw(func(p *ui.Painter, r ui.Rect) { chevron(p, r, open, t.TextMuted) })
			ui.Column(c).Grow(1).Gap(t.Space(1.5)).Children(func() {
				field(c, &s.find, "Search", err != nil, func() {
					toggle(c, &s.matchCase, "Aa", "Match Case")
					toggle(c, &s.word, "ab", "Match Whole Word")
					toggle(c, &s.regex, ".*", "Use Regular Expression")
				})
				if !s.replacing {
					return
				}
				field(c, &s.replace, "Replace", false, func() {
					all := ui.ButtonBase(c).Padding(0, t.Space(1.5)).Height(t.Space(5)).Radius(t.Space(1)).Label("Replace All").
						Disabled(total == 0)
					if all.Hovered() && total > 0 {
						all.Background(t.Text.Alpha(0.07))
					}
					all.Children(func() { ui.Text(c, "Replace All").FontSize(t.FontSize - 1) })
					if all.Clicked() && total > 0 {
						for _, g := range groups {
							ev.Replaced += s.apply(re, &(*files)[g.file], g.hits)
						}
					}
				})
			})
		})
		summary := ""
		switch {
		case err != nil:
			summary = "Not a valid regular expression"
		case s.find == "":
		case total == 0:
			summary = "No results found"
		default:
			summary = fmt.Sprintf("%d %s in %d %s", total, plural(total, "result"), len(groups), plural(len(groups), "file"))
		}
		if summary != "" {
			color := t.TextMuted
			if err != nil {
				color = t.Danger
			}
			ui.Text(c, summary).FontSize(t.FontSize-1).TextColor(color).Padding(0, 0, 0, t.Space(6))
		}

		list := ui.List(c, &s.list, len(rows), func(i int) {
			r := rows[i]
			g := groups[r.group]
			f := &(*files)[g.file]
			if r.hit < 0 {
				if header(c, s, f.Path, len(g.hits), replace, i == s.row) {
					ev.Replaced += s.apply(re, f, g.hits)
				}
				return
			}
			h := g.hits[r.hit]
			by := ""
			if replace {
				by = s.with(re, f.Text, h)
			}
			if line(c, h, by, replace, i == s.row) {
				ev.Replaced += s.apply(re, f, []hit{h})
			}
		}).Grow(1)
		if (list.Changed() || list.Submitted()) && s.row >= 0 {
			if r := rows[s.row]; r.hit >= 0 {
				g := groups[r.group]
				h := g.hits[r.hit]
				ev.Opened = Match{Path: (*files)[g.file].Path, Line: h.line, Col: h.col}
			}
		}
	})
	return ev
}

func plural(n int, word string) string {
	if n == 1 {
		return word
	}
	return word + "s"
}

// field creates a text input on value, its border in the danger color
// while bad, with the buttons end builds at its end.
func field(c *ui.Context, value *string, label string, bad bool, end func()) {
	t := c.Theme()
	border := t.Border
	if bad {
		border = t.Danger
	}
	ui.Row(c).Gap(t.Space(0.5)).AlignItems(ui.Center).Padding(t.Space(0.5), t.Space(1), t.Space(0.5), t.Space(2)).
		Radius(t.Space(1.5)).Border(1, border).Background(t.Background).Children(func() {
		ui.TextInputBase(c, value).Grow(1).Shrink(1).Placeholder(label).Label(label).SingleLine()
		end()
	})
}

// toggle creates a small toggle showing text.
func toggle(c *ui.Context, on *bool, text, label string) {
	t := c.Theme()
	b := ui.ToggleBase(c, on).Size(t.Space(6), t.Space(5)).Radius(t.Space(1)).Label(label).Tooltip(label).KeepFocus()
	ink := t.TextMuted
	switch {
	case *on:
		b.Background(t.Accent.Alpha(0.15)).Border(1, t.Accent.Alpha(0.6))
		ink = t.Accent
	case b.Hovered():
		b.Background(t.Text.Alpha(0.07))
	}
	b.Children(func() {
		ui.Text(c, text).FontSize(t.FontSize-1).TextColor(ink).Bold().Margin(0, ui.Auto)
	})
}

// header creates the row of a file: its name, its folder dimmed, and how
// many matches, or over it a button replacing them. It reports a click on
// the button.
func header(c *ui.Context, s *State, p string, n int, replace, chosen bool) bool {
	t := c.Theme()
	ink, muted := t.Text, t.TextMuted
	if chosen {
		ink, muted = t.AccentText, t.AccentText
	}
	replaced := false
	r := ui.Row(c).Gap(t.Space(1.5)).AlignItems(ui.Center).Padding(t.Space(1), t.Space(1)).Background(t.Background)
	if chosen {
		r.Background(t.Accent).Radius(t.Space(1))
	}
	folded := s.folded[p]
	if r.Clicked() {
		s.folded[p] = !folded
	}
	r.Children(func() {
		ui.Box(c).Size(t.Space(4), t.Space(4)).Draw(func(pa *ui.Painter, rc ui.Rect) { chevron(pa, rc, !folded, muted) })
		ui.Text(c, path.Base(p)).TextColor(ink).SingleLine().Shrink(0)
		if dir := path.Dir(p); dir != "." {
			ui.Text(c, dir).FontSize(t.FontSize - 1).TextColor(muted).SingleLine().Shrink(1)
		}
		ui.Spacer(c)
		if replace && r.Hovered() {
			replaced = button(c, "Replace All in File")
			return
		}
		ui.Text(c, fmt.Sprint(n)).FontSize(t.FontSize-2).Padding(0, t.Space(1.5)).Radius(999).
			Background(muted.Alpha(0.15)).TextColor(ink)
	})
	return replaced
}

// line creates the row of the match h: its line from a little before it,
// the match lit, or struck out with by after it while replacing. It
// reports a click on the button replacing it.
func line(c *ui.Context, h hit, by string, replace, chosen bool) bool {
	t := c.Theme()
	ink := t.Text
	if chosen {
		ink = t.AccentText
	}
	from := h.start - h.lineStart
	// A long line starts a few words before its match, so the match shows.
	lead := ""
	if before := []rune(h.text[:from]); len(before) > 24 {
		lead = "…" + strings.TrimLeft(string(before[len(before)-20:]), " \t")
	} else {
		lead = strings.TrimLeft(string(before), " \t")
	}
	spans := []ui.Span{{Text: lead, Color: ink}}
	match := h.text[from : from+h.end-h.start]
	if replace {
		spans = append(spans,
			ui.Span{Text: match, Color: ink, Strikethrough: true, Background: t.Danger.Alpha(0.2)},
			ui.Span{Text: by, Color: ink, Background: t.Success.Alpha(0.25)})
	} else {
		spans = append(spans, ui.Span{Text: match, Color: ink, Background: t.Warning.Alpha(0.35), Weight: 600})
	}
	spans = append(spans, ui.Span{Text: h.text[from+h.end-h.start:], Color: ink})
	replaced := false
	r := ui.Row(c).Gap(t.Space(1.5)).AlignItems(ui.Center).Padding(t.Space(0.5), t.Space(1), t.Space(0.5), t.Space(6)).Radius(t.Space(1))
	if chosen {
		r.Background(t.Accent)
	}
	r.Children(func() {
		ui.RichText(c, spans...).SingleLine().Grow(1).Shrink(1).FontSize(t.FontSize - 1)
		if replace && r.Hovered() {
			replaced = button(c, "Replace")
		}
	})
	return replaced
}

// button creates a small button named label and reports a click.
func button(c *ui.Context, label string) bool {
	t := c.Theme()
	b := ui.ButtonBase(c).Padding(0, t.Space(1.5)).Height(t.Space(5)).Radius(t.Space(1)).Label(label).Tooltip(label).
		Background(t.Background).Border(1, t.Border)
	b.Children(func() { ui.Text(c, "Replace").FontSize(t.FontSize - 2) })
	return b.Clicked()
}

// chevron draws an arrow in r pointing down when open, right when not.
func chevron(p *ui.Painter, r ui.Rect, open bool, color ui.Color) {
	cx, cy, d := r.X+r.W/2, r.Y+r.H/2, float32(3)
	pts := [][2]float32{{cx - d/2, cy - d}, {cx + d/2, cy}, {cx - d/2, cy + d}}
	if open {
		pts = [][2]float32{{cx - d, cy - d/2}, {cx, cy + d/2}, {cx + d, cy - d/2}}
	}
	p.StrokePolyline(pts, 1.4, color)
}
