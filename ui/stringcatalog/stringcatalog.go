// Package stringcatalog edits an app's strings in every language it
// speaks, as Xcode's String Catalog does: the languages in a sidebar with
// how much of each is translated, and a row a string with its key, its
// comment, the source text and the translation, typed in place. A string
// that varies by plural has a field for each plural form the language
// has, one and other in English, one, few, many and other in Russian.
// Each translation is New while empty, Translated, or Needs Review, which
// changing the source text marks, and a click on it marks it reviewed.
// A search and All, Missing and Needs Review narrow the rows.
//
//	stringcatalog.View(c, &app.strings, &app.catalog).Grow(1)
package stringcatalog

import (
	"fmt"
	"slices"
	"strings"

	"github.com/egoist/mygo/ui"
	"golang.org/x/text/language"
	"golang.org/x/text/language/display"
)

// Catalog is an app's strings: the language they are written in, the
// languages they are translated to, and the strings.
type Catalog struct {
	Source    string
	Languages []string
	Entries   []Entry
}

// Entry is a string: its key, the comment telling translators where it
// shows, whether it varies by plural, and its text in each language, the
// source's too.
type Entry struct {
	Key, Comment string
	Plural       bool
	Text         map[string]*Translation
}

// Translation is a string's text in a language: Value, or by plural form
// in Forms, and whether it needs review.
type Translation struct {
	Value  string
	Forms  map[string]string
	Review bool
}

// pluralForms are the plural forms of languages as CLDR has them, by the
// language's base; the others have one and other.
var pluralForms = map[string][]string{
	"ar": {"zero", "one", "two", "few", "many", "other"},
	"cs": {"one", "few", "many", "other"},
	"es": {"one", "many", "other"},
	"fr": {"one", "many", "other"},
	"it": {"one", "many", "other"},
	"ja": {"other"},
	"ko": {"other"},
	"pl": {"one", "few", "many", "other"},
	"pt": {"one", "many", "other"},
	"ru": {"one", "few", "many", "other"},
	"uk": {"one", "few", "many", "other"},
	"zh": {"other"},
}

// Forms returns the plural forms of lang, as "one" and "other".
func Forms(lang string) []string {
	base, _, _ := strings.Cut(lang, "-")
	if f, ok := pluralForms[base]; ok {
		return f
	}
	return []string{"one", "other"}
}

// Status is how far a string is translated into a language.
type Status int

const (
	New Status = iota
	Translated
	NeedsReview
)

// Status returns how far e is translated into lang.
func (e *Entry) Status(lang string) Status {
	tr := e.Text[lang]
	if tr == nil {
		return New
	}
	if e.Plural {
		for _, f := range Forms(lang) {
			if tr.Forms[f] == "" {
				return New
			}
		}
	} else if tr.Value == "" {
		return New
	}
	if tr.Review {
		return NeedsReview
	}
	return Translated
}

// translation returns e's translation into lang, made if there is none.
func (e *Entry) translation(lang string) *Translation {
	if e.Text == nil {
		e.Text = map[string]*Translation{}
	}
	tr := e.Text[lang]
	if tr == nil {
		tr = &Translation{}
		e.Text[lang] = tr
	}
	if e.Plural && tr.Forms == nil {
		tr.Forms = map[string]string{}
	}
	return tr
}

// Name returns lang's name in English, as "Simplified Chinese".
func Name(lang string) string {
	tag, err := language.Parse(lang)
	if err != nil {
		return lang
	}
	if name := display.English.Tags().Name(tag); name != "" {
		return name
	}
	return lang
}

// State is what an editor keeps: the language shown, the search and the
// filter. Keep it in the app's state, one for each editor.
type State struct {
	lang   string
	query  string
	filter int
	list   ui.ListState
	// kept are the strings the filter showed since it, the search or the
	// language last changed, which stay while they are translated rather
	// than leave under the typing.
	kept  map[string]bool
	shown string
}

// View creates the editor of *cat. Changed reports a change. Give it a
// size, or grow it in its parent.
func View(c *ui.Context, s *State, cat *Catalog) ui.Element {
	t := c.Theme()
	langs := append([]string{cat.Source}, slices.DeleteFunc(slices.Clone(cat.Languages), func(l string) bool { return l == cat.Source })...)
	if !slices.Contains(langs, s.lang) {
		s.lang = langs[min(1, len(langs)-1)]
	}
	root := ui.Row(c)
	changed := false
	root.Children(func() {
		ui.Column(c).Width(t.Space(60)).FillHeight().Padding(t.Space(2)).Gap(t.Space(0.5)).Background(t.Surface).Children(func() {
			for _, lang := range langs {
				s.language(c, cat, lang)
			}
		})
		ui.Divider(c).Vertical()
		ui.Column(c).Grow(1).FillHeight().Children(func() {
			changed = s.table(c, cat)
		})
	})
	if changed {
		root.SetChanged()
	}
	return root
}

// language creates the sidebar's row of lang: its name and how much of
// it is translated.
func (s *State) language(c *ui.Context, cat *Catalog, lang string) {
	t := c.Theme()
	done := 0
	for i := range cat.Entries {
		if cat.Entries[i].Status(lang) == Translated {
			done++
		}
	}
	share := 1.0
	if len(cat.Entries) > 0 {
		share = float64(done) / float64(len(cat.Entries))
	}
	row := ui.Column(c.Key(lang)).Gap(t.Space(1)).Padding(t.Space(2), t.Space(3)).Radius(t.Space(1.5)).
		Cursor(ui.CursorPointer).Role(ui.RoleButton).Label(Name(lang))
	if lang == s.lang {
		row.Background(t.Selection)
	} else if row.Hovered() {
		row.Background(t.SurfaceHover)
	}
	if row.Clicked() {
		s.lang = lang
	}
	row.Children(func() {
		ui.Row(c).Gap(t.Space(2)).Children(func() {
			ui.Text(c, Name(lang)).Grow(1).SingleLine().Ellipsis("…")
			note := fmt.Sprintf("%.0f%%", share*100)
			if lang == cat.Source {
				note = "Source"
			}
			ui.Text(c, note).TextColor(t.TextMuted).FontFeatures("tnum").FontSize(t.FontSize - 1)
		})
		if lang != cat.Source {
			ui.Box(c).Height(3).Radius(1.5).Background(t.Text.Alpha(0.1)).Children(func() {
				ui.Box(c).Height(3).WidthPercent(float32(share * 100)).Radius(1.5).Background(t.Success)
			})
		}
	})
}

var filters = []string{"All", "Missing", "Needs Review"}

// table creates the strings of the language shown, and reports a change.
func (s *State) table(c *ui.Context, cat *Catalog) bool {
	t := c.Theme()
	var rows []int
	q := strings.ToLower(s.query)
	if shown := fmt.Sprint(s.lang, s.filter, q); shown != s.shown || s.kept == nil {
		s.shown, s.kept = shown, map[string]bool{}
		s.list.ScrollTo(0, ui.Start)
	}
	counts := [3]int{}
	for i := range cat.Entries {
		e := &cat.Entries[i]
		st := e.Status(s.lang)
		src := ""
		if tr := e.Text[cat.Source]; tr != nil {
			src = tr.Value + strings.Join(mapValues(tr.Forms), " ")
		}
		if q != "" && !strings.Contains(strings.ToLower(e.Key+" "+src+" "+e.Comment), q) {
			continue
		}
		counts[0]++
		if st == New {
			counts[1]++
		}
		if st == NeedsReview {
			counts[2]++
		}
		if s.filter == 0 || s.filter == 1 && st == New || s.filter == 2 && st == NeedsReview {
			s.kept[e.Key] = true
		}
		if s.kept[e.Key] {
			rows = append(rows, i)
		}
	}
	ui.Row(c).Padding(t.Space(3), t.Space(4)).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
		ui.SearchField(c, &s.query).Width(t.Space(60)).Label("Search strings")
		labels := make([]string, 3)
		for i, f := range filters {
			labels[i] = fmt.Sprintf("%s %d", f, counts[i])
		}
		ui.Segmented(c, &s.filter, labels...).Label("Show")
	})
	ui.Divider(c)
	// The source language shows its text once, to edit.
	source := s.lang == cat.Source
	cols := []ui.Track{ui.Fr(1), ui.Fr(1.3), ui.Fr(1.6), ui.Fixed(t.Space(30))}
	heads := []string{"Key", Name(cat.Source), Name(s.lang), "State"}
	if source {
		cols, heads = slices.Delete(cols, 1, 2), slices.Delete(heads, 1, 2)
	}
	ui.Grid(c).ColumnTracks(cols...).GapX(t.Space(4)).Padding(t.Space(2), t.Space(4)).Children(func() {
		for _, h := range heads {
			ui.Text(c.Key(h), h).FontSize(t.FontSize - 1).TextColor(t.TextMuted).Bold().SingleLine()
		}
	})
	ui.Divider(c)
	changed := false
	s.list.Key = func(row int) any { return cat.Entries[rows[row]].Key }
	ui.List(c, &s.list, len(rows), func(row int) {
		e := &cat.Entries[rows[row]]
		ui.Grid(c).ColumnTracks(cols...).GapX(t.Space(4)).Padding(t.Space(2.5), t.Space(4)).AlignItems(ui.Start).
			BorderWidth(0, 0, 1, 0).BorderColor(t.Border).Children(func() {
			ui.Column(c).Gap(t.Space(0.5)).Children(func() {
				ui.Text(c, e.Key).Font("monospace").FontSize(t.FontSize - 1).Selectable()
				if e.Comment != "" {
					ui.Text(c, e.Comment).FontSize(t.FontSize - 1).TextColor(t.TextMuted)
				}
			})
			if !source {
				ui.Column(c).Gap(t.Space(1)).Children(func() {
					src := e.Text[cat.Source]
					switch {
					case src == nil:
					case e.Plural:
						for _, f := range Forms(cat.Source) {
							ui.Text(c.Key(f), f+": "+src.Forms[f])
						}
					default:
						ui.Text(c, src.Value)
					}
				})
			}
			if s.edit(c, cat, e) {
				changed = true
			}
			s.status(c, e)
		})
	}).Grow(1).Children(func() {
		if len(rows) == 0 {
			ui.Text(c, "No strings").TextColor(t.TextMuted).Padding(t.Space(6)).AlignSelf(ui.Center)
		}
	})
	return changed
}

func mapValues(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for _, v := range m {
		out = append(out, v)
	}
	return out
}

// edit creates the fields of e's text in the language shown, and reports
// a change. A change to the source text marks the translations for
// review.
func (s *State) edit(c *ui.Context, cat *Catalog, e *Entry) bool {
	t := c.Theme()
	changed := false
	set := func(apply func(tr *Translation)) {
		tr := e.translation(s.lang)
		apply(tr)
		tr.Review = false
		if s.lang == cat.Source {
			for lang, other := range e.Text {
				if lang != cat.Source && e.Status(lang) != New {
					other.Review = true
				}
			}
		}
		changed = true
	}
	tr := e.Text[s.lang]
	ui.Column(c).Gap(t.Space(1)).Children(func() {
		if !e.Plural {
			value := ""
			if tr != nil {
				value = tr.Value
			}
			if ui.TextInput(c, &value).Placeholder("Translation").Label(e.Key).Changed() {
				set(func(tr *Translation) { tr.Value = value })
			}
			return
		}
		for _, f := range Forms(s.lang) {
			value := ""
			if tr != nil {
				value = tr.Forms[f]
			}
			ui.Row(c.Key(f)).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
				ui.Text(c, f).Width(t.Space(10)).FontSize(t.FontSize - 1).TextColor(t.TextMuted)
				if ui.TextInput(c, &value).Grow(1).Placeholder("Translation").Label(e.Key + " " + f).Changed() {
					set(func(tr *Translation) { tr.Forms[f] = value })
				}
			})
		}
	})
	return changed
}

// status creates e's state in the language shown, a click on Needs
// Review marking it reviewed.
func (s *State) status(c *ui.Context, e *Entry) {
	t := c.Theme()
	st := e.Status(s.lang)
	label, color := "New", t.TextMuted
	switch st {
	case Translated:
		label, color = "Translated", t.Success
	case NeedsReview:
		label, color = "Needs Review", t.Warning
	}
	chip := ui.Row(c).Gap(t.Space(1.5)).AlignItems(ui.Center).Padding(t.Space(0.5), t.Space(2)).Radius(t.Space(3)).
		Background(color.Alpha(0.12)).AlignSelf(ui.Start).JustifySelf(ui.Start)
	if st == NeedsReview {
		chip.Cursor(ui.CursorPointer).Role(ui.RoleButton).Tooltip("Mark as reviewed")
		if chip.Clicked() {
			e.Text[s.lang].Review = false
			chip.SetChanged()
		}
	}
	chip.Children(func() {
		ui.Box(c).Size(6, 6).Radius(3).Background(color)
		ui.Text(c, label).FontSize(t.FontSize - 1).TextColor(color).SingleLine()
	})
}
