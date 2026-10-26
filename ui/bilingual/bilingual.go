// Package bilingual edits a translation segment by segment, as the
// editors of translation tools (memoQ, Trados, Crowdin) do: a row for each
// segment with its number, the source text and the translation side by
// side, and a mark of its status, empty, draft or confirmed. The row being
// edited is lit on both sides. Placeholders, as {name}, %d and <b>, show
// in the accent on both sides, and a translation that leaves one out says
// which. Cmd-Enter confirms a segment and goes to the next one not
// confirmed; editing a confirmed one makes it a draft again. A bar above
// shows how many are confirmed, and a choice shows all segments or those
// left to do.
//
//	if bilingual.View(c, &app.editor, "English", "简体中文", app.segments).Changed() {
//		app.save()
//	}
package bilingual

import (
	"regexp"
	"slices"
	"strconv"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
)

// Segment is a sentence or a string to translate: its source, its
// translation, and whether the translation is confirmed.
type Segment struct {
	Source, Target string
	Confirmed      bool
	// Note says something about the segment to the translator, as where
	// the string shows, under the source.
	Note string
}

// State is what an editor keeps: which segments show, the segment being
// edited and the one to go to. Keep it in the app's state, one for each
// editor.
type State struct {
	todo  int
	at    int
	focus int
	refs  []ui.Ref
}

// placeholder matches what must pass into a translation as it is: format
// verbs, braced names and markup tags.
var placeholder = regexp.MustCompile(`\{[^{}\s]+\}|%(?:\[\d+\])?[-+ #0]*\d*(?:\.\d+)?[a-zA-Z%]|</?[a-zA-Z][^<>]*>`)

// View creates the editor of segs, from the language named from to the
// one named to. Changed reports an edit, of a translation or a status.
// Give it a size, or grow it in its parent.
func View(c *ui.Context, s *State, from, to string, segs []Segment) ui.Element {
	t := c.Theme()
	if len(s.refs) != len(segs) {
		s.refs = make([]ui.Ref, len(segs))
		s.focus = -1
	}
	done := 0
	for _, sg := range segs {
		if sg.Confirmed {
			done++
		}
	}
	changed := false
	root := ui.Column(c)
	root.Children(func() {
		ui.Row(c).Padding(t.Space(2), t.Space(3)).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			ui.Segmented(c, &s.todo, "All", "To do").Label("Show")
			ui.Progress(c, float64(done)/float64(max(1, len(segs)))).Grow(1).Label("Confirmed")
			ui.Text(c, strconv.Itoa(done)+" of "+strconv.Itoa(len(segs))+" confirmed").
				FontSize(t.FontSize - 1).TextColor(t.TextMuted).FontFeatures("tnum")
		})
		ui.Divider(c)
		ui.Row(c).Background(t.Surface).PaddingY(t.Space(1.5)).Children(func() {
			ui.Box(c).Width(t.Space(10))
			ui.Text(c, from).Bold().FontSize(t.FontSize - 1).TextColor(t.TextMuted).Grow(1).Basis(0).PaddingX(t.Space(3))
			ui.Text(c, to).Bold().FontSize(t.FontSize - 1).TextColor(t.TextMuted).Grow(1).Basis(0).PaddingX(t.Space(3))
			ui.Box(c).Width(t.Space(10))
		})
		ui.Divider(c)
		ui.Scroll(c).Grow(1).Children(func() {
			shown := 0
			for i := range segs {
				if s.todo == 1 && segs[i].Confirmed && s.at != i {
					continue
				}
				shown++
				if s.row(c.Key(i), i, &segs[i]) {
					changed = true
				}
			}
			if shown == 0 {
				ui.Text(c, "Every segment is confirmed").TextColor(t.TextMuted).AlignSelf(ui.Center).MarginY(t.Space(10))
			}
		})
	})
	if s.at >= 0 && s.at < len(segs) && c.Focused(s.refs[s.at]) && c.Shortcut(ui.Cmd, ui.KeyEnter) {
		segs[s.at].Confirmed, changed = true, true
		s.focus = -1
		for k := 1; k <= len(segs); k++ {
			if j := (s.at + k) % len(segs); !segs[j].Confirmed {
				s.focus = j
				break
			}
		}
	}
	if changed {
		root.SetChanged()
	}
	return root
}

// row creates the row of the i-th segment, sg, and reports an edit.
func (s *State) row(c *ui.Context, i int, sg *Segment) bool {
	t := c.Theme()
	changed := false
	r := ui.Row(c).AlignItems(ui.Stretch)
	editing := r.FocusWithin()
	if editing {
		s.at = i
		r.Background(t.Accent.Alpha(0.07))
	}
	missing := missingFrom(sg.Source, sg.Target)
	r.Children(func() {
		ui.Text(c, strconv.Itoa(i+1)).FontSize(t.FontSize - 1).TextColor(t.TextMuted).FontFeatures("tnum").
			Width(t.Space(10)).TextAlign(ui.End).PaddingY(t.Space(2.5)).PaddingX(t.Space(2))
		ui.Column(c).Grow(1).Basis(0).MinWidth(0).Padding(t.Space(2.5), t.Space(3)).Gap(t.Space(1)).Children(func() {
			ui.RichText(c, spans(sg.Source, t)...).Selectable()
			if sg.Note != "" {
				ui.Text(c, sg.Note).FontSize(t.FontSize - 2).TextColor(t.TextMuted).Italic()
			}
		})
		ui.Column(c).Grow(1).Basis(0).MinWidth(0).Padding(t.Space(1), t.Space(1)).Gap(t.Space(1)).Children(func() {
			in := ui.TextArea(c, &sg.Target).Bind(&s.refs[i]).Placeholder("Translation").Label("Translation of segment "+strconv.Itoa(i+1)).Lines(1, 8).MinHeight(0).
				TextRanges(ranges(sg.Target, t)...)
			if !editing {
				in.Background(ui.Transparent).BorderColor(ui.Transparent)
			}
			if s.focus == i {
				in.Focus().ScrollIntoView()
				s.focus = -1
				// The rows lit follow the focus, which moves after this frame.
				c.Invalidate()
			}
			if in.Changed() {
				sg.Confirmed, changed = false, true
			}
			if len(missing) > 0 && sg.Target != "" {
				text := "Missing "
				for k, m := range missing {
					if k > 0 {
						text += ", "
					}
					text += m
				}
				ui.Text(c, text).FontSize(t.FontSize - 2).TextColor(t.Danger).PaddingX(t.Space(2))
			}
		})
		s.status(c, sg, &changed)
	})
	ui.Divider(c)
	return changed
}

// status creates the mark of sg's status, a button that confirms it or
// makes it a draft again.
func (s *State) status(c *ui.Context, sg *Segment, changed *bool) {
	t := c.Theme()
	glyph, col, label := "", t.TextMuted, "Untranslated"
	switch {
	case sg.Confirmed:
		glyph, col, label = "✓", t.Success, "Confirmed"
	case sg.Target != "":
		glyph, col, label = "✎", t.Warning, "Draft"
	}
	b := ui.ButtonBase(c).Width(t.Space(10)).AlignItems(ui.Start).Justify(ui.Center).PaddingY(t.Space(2)).
		Label(label).Tooltip(label).Disabled(sg.Target == "")
	b.Children(func() {
		ui.Box(c).Size(t.Space(5), t.Space(5)).Radius(t.Space(2.5)).Border(1.5, col.Alpha(0.7)).PassThrough().Children(func() {
			ui.Text(c, glyph).FontSize(t.FontSize - 2).Bold().TextColor(col).Center().PassThrough()
		})
	})
	if b.Clicked() {
		sg.Confirmed, *changed = !sg.Confirmed, true
	}
}

// spans returns text in spans, its placeholders in the accent.
func spans(text string, t *ui.Theme) []ui.Span {
	var out []ui.Span
	at := 0
	for _, m := range placeholder.FindAllStringIndex(text, -1) {
		if m[0] > at {
			out = append(out, ui.Span{Text: text[at:m[0]]})
		}
		out = append(out, ui.Span{Text: text[m[0]:m[1]], Color: t.Accent, Weight: 600})
		at = m[1]
	}
	if at < len(text) || len(out) == 0 {
		out = append(out, ui.Span{Text: text[at:]})
	}
	return out
}

// ranges returns the placeholders of a field's text as ranges of its
// runes, in the accent.
func ranges(text string, t *ui.Theme) []ui.TextRange {
	var out []ui.TextRange
	for _, m := range placeholder.FindAllStringIndex(text, -1) {
		out = append(out, ui.TextRange{
			Start: utf8.RuneCountInString(text[:m[0]]), End: utf8.RuneCountInString(text[:m[1]]),
			Color: t.Accent, Weight: 600,
		})
	}
	return out
}

// missingFrom returns the placeholders of source that target has fewer
// of, once each.
func missingFrom(source, target string) []string {
	have := map[string]int{}
	for _, p := range placeholder.FindAllString(target, -1) {
		have[p]++
	}
	var out []string
	for _, p := range placeholder.FindAllString(source, -1) {
		if have[p] > 0 {
			have[p]--
		} else if !slices.Contains(out, p) {
			out = append(out, p)
		}
	}
	return out
}
