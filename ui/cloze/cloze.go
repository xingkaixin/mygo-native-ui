// Package cloze is a fill-in-the-blank exercise, as language courses and
// flashcard apps set them: a text with blanks to type into, each as wide
// as its longest answer, Enter going on to the next blank, as Tab does.
// Check marks each blank right in green or wrong in red, a wrong one
// showing its answer under the pointer, and counts those right; typing in
// a blank takes its mark off. Reveal writes the answers in, and Reset
// starts over.
//
//	ev := cloze.View(c, &app.exercise, "Water {evaporates} from the sea.")
//	if ev.Checked {
//		app.score(ev.Right, ev.Blanks)
//	}
//
// A blank is written in braces, its answers parted by a bar, as
// {colour|color}; an answer matches whatever its case and the spaces
// around and between its words. A blank line starts a paragraph.
package cloze

import (
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
)

// Events is the exercise, whether the user checked it this frame, and how
// many blanks it has and are marked right.
type Events struct {
	ui.Element
	Checked       bool
	Right, Blanks int
}

// State is what an exercise keeps: the text parsed, what the user typed in
// each blank, the marks, and the blank focused. Keep it in the app's
// state, one for each exercise.
type State struct {
	src      string
	paras    [][]unit
	answers  [][]string
	typed    []string
	mark     []int8
	revealed bool
	focus    int
}

// unit is a run of tokens a line does not break inside, as a word and the
// punctuation after it, and whether a space follows it.
type unit struct {
	toks []token
	gap  bool
}

// token is text, or blank, the index of a blank, -1 for text.
type token struct {
	text  string
	blank int
}

// parse splits text into paragraphs of units and blanks.
func (s *State) parse(text string) {
	s.src, s.paras, s.answers = text, nil, nil
	for _, para := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n\n") {
		var units []unit
		var last rune
		add := func(tk token, first rune) {
			k := len(units) - 1
			// A line breaks at spaces and around Chinese and Japanese
			// characters, but not before punctuation.
			if k < 0 || units[k].gap || ideograph(first) || ideograph(last) && !unicode.IsPunct(first) {
				units = append(units, unit{})
				k++
			}
			units[k].toks = append(units[k].toks, tk)
		}
		rest := strings.Join(strings.Fields(para), " ")
		for rest != "" {
			r, n := utf8.DecodeRuneInString(rest)
			switch {
			case r == ' ':
				if k := len(units) - 1; k >= 0 {
					units[k].gap = true
				}
				rest = rest[n:]
				continue
			case r == '{' && strings.IndexByte(rest, '}') > 0:
				end := strings.IndexByte(rest, '}')
				var alts []string
				for _, a := range strings.Split(rest[1:end], "|") {
					if a = strings.TrimSpace(a); a != "" {
						alts = append(alts, a)
					}
				}
				add(token{blank: len(s.answers)}, 'x')
				s.answers = append(s.answers, alts)
				rest, last = rest[end+1:], 'x'
				continue
			case !ideograph(r):
				for n < len(rest) {
					r, k := utf8.DecodeRuneInString(rest[n:])
					if r == ' ' || r == '{' || ideograph(r) {
						break
					}
					n += k
				}
			}
			add(token{text: rest[:n], blank: -1}, r)
			last, _ = utf8.DecodeLastRuneInString(rest[:n])
			rest = rest[n:]
		}
		if len(units) > 0 {
			s.paras = append(s.paras, units)
		}
	}
	s.typed = make([]string, len(s.answers))
	s.mark = make([]int8, len(s.answers))
	s.revealed = false
}

func ideograph(r rune) bool {
	return unicode.In(r, unicode.Han, unicode.Hiragana, unicode.Katakana)
}

// right reports whether typed is one of answers.
func right(typed string, answers []string) bool {
	norm := func(s string) string { return strings.Join(strings.Fields(strings.ToLower(s)), " ") }
	for _, a := range answers {
		if norm(typed) == norm(a) {
			return true
		}
	}
	return false
}

// View creates the exercise text and below it Check, Reveal, Reset and,
// once checked, how many blanks are right.
func View(c *ui.Context, s *State, text string) Events {
	t := c.Theme()
	if s.paras == nil || s.src != text {
		s.parse(text)
	}
	ev := Events{Blanks: len(s.answers)}
	check := func() {
		for i := range s.answers {
			s.mark[i] = -1
			if right(s.typed[i], s.answers[i]) {
				s.mark[i] = 1
			}
		}
		ev.Checked = true
	}
	size := t.FontSize + 3
	wide, _ := c.MeasureText(0, ui.Span{Text: "a a", Size: size})
	narrow, _ := c.MeasureText(0, ui.Span{Text: "aa", Size: size})
	col := ui.Column(c).Gap(t.Space(5))
	ev.Element = col
	col.Children(func() {
		ui.Column(c).Gap(t.Space(4)).Children(func() {
			for pi, para := range s.paras {
				ui.Row(c.Key(pi)).Wrap().GapY(t.Space(2.5)).AlignItems(ui.Baseline).Children(func() {
					for _, u := range para {
						run := ui.Row(c).AlignItems(ui.Baseline)
						if u.gap {
							run.MarginEnd(wide - narrow)
						}
						run.Children(func() {
							for _, tk := range u.toks {
								if tk.blank < 0 {
									ui.Text(c, tk.text).FontSize(size)
								} else if s.blank(c, tk.blank, size) {
									check()
								}
							}
						})
					}
				})
			}
		})
		ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
			if ui.PrimaryButton(c, "Check").Disabled(s.revealed || len(s.answers) == 0).Clicked() {
				check()
			}
			if ui.Button(c, "Reveal").Disabled(s.revealed).Clicked() {
				for i, alts := range s.answers {
					if s.mark[i] != 1 && len(alts) > 0 {
						s.typed[i], s.mark[i] = alts[0], 0
					}
				}
				s.revealed = true
			}
			if ui.Button(c, "Reset").Clicked() {
				clear(s.typed)
				clear(s.mark)
				s.revealed, s.focus = false, 1
			}
			marked := 0
			for _, m := range s.mark {
				if m != 0 {
					marked++
				}
				if m == 1 {
					ev.Right++
				}
			}
			if marked > 0 {
				ui.Text(c, strconv.Itoa(ev.Right)+" of "+strconv.Itoa(len(s.answers))+" right").
					FontFeatures("tnum").TextColor(t.TextMuted).Margin(0, 0, 0, t.Space(2))
			}
		})
	})
	return ev
}

// blank creates blank i in text of size, and reports whether Enter in the
// last blank asks to check the exercise.
func (s *State) blank(c *ui.Context, i int, size float32) (check bool) {
	t := c.Theme()
	alts := s.answers[i]
	longest := ""
	for _, a := range alts {
		if utf8.RuneCountInString(a) > utf8.RuneCountInString(longest) {
			longest = a
		}
	}
	w, _ := c.MeasureText(0, ui.Span{Text: longest + "m", Size: size})
	in := ui.TextInputBase(c.Key(i), &s.typed[i]).FocusBind(&s.focus, i+1).
		Width(max(w, t.Space(10))).Padding(t.Space(0.5), t.Space(1)).
		FontSize(size).TextAlign(ui.Center).Radius(t.Space(1), t.Space(1), 0, 0).
		Label("Blank " + strconv.Itoa(i+1)).ReadOnly(s.revealed)
	if in.Changed() {
		s.mark[i] = 0
	}
	if in.Submitted() {
		if i+1 < len(s.answers) {
			s.focus = i + 2
		} else if !s.revealed {
			check = true
		}
	}
	line := t.TextMuted
	switch {
	case s.mark[i] == 1:
		line = t.Success
		in.Background(t.Success.Alpha(0.12)).TextColor(t.Success)
	case s.mark[i] == -1:
		line = t.Danger
		in.Background(t.Danger.Alpha(0.1)).TextColor(t.Danger)
		if len(alts) > 0 {
			in.Tooltip(alts[0])
		}
	case s.revealed:
		in.TextColor(t.Accent)
	case in.Focused():
		line = t.Accent
		in.Background(t.Accent.Alpha(0.06))
	default:
		in.Background(t.Text.Alpha(0.04))
	}
	in.FocusRing(false).DrawOver(func(p *ui.Painter, r ui.Rect) {
		p.Fill(ui.Rect{X: r.X, Y: r.Y + r.H - 1.5, W: r.W, H: 1.5}, line, 0)
	})
	return check
}
