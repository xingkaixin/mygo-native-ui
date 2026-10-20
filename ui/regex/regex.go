// Package regex is a regular expression tester, as regex101 is: the
// pattern, its groups in colors, with a toggle for each of its flags,
// which edit the flags at its start, "(?i)"; what is wrong where it does
// not compile; and a sample text whose matches show as the user types,
// each group in its color, with a list of the matches and their groups
// below. Patterns are Go's, RE2's syntax.
//
//	regex.View(c, &app.tester, &app.pattern, &app.sample).Grow(1)
package regex

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
)

// State is what a tester keeps: the pattern it compiled last. Keep it in
// the app's state, one for each tester.
type State struct {
	pattern string
	re      *regexp.Regexp
	err     error
}

// flags are the flags a toggle edits, and what they do.
var flags = []struct {
	flag, name string
}{
	{"i", "Case-insensitive"},
	{"m", "Multi-line: ^ and $ match at lines"},
	{"s", "Single-line: . matches \\n"},
	{"U", "Ungreedy: swap x* and x*?"},
}

// leading is the group of flags a pattern starts with.
var leading = regexp.MustCompile(`^\(\?([imsU]*)\)`)

// withFlag returns pattern with flag on or off in the flags at its start.
func withFlag(pattern, flag string, on bool) string {
	set, rest := "", pattern
	if m := leading.FindStringSubmatch(pattern); m != nil {
		set, rest = m[1], pattern[len(m[0]):]
	}
	set = strings.ReplaceAll(set, flag, "")
	if on {
		set += flag
	}
	if set == "" {
		return rest
	}
	b := []byte(set)
	slices.Sort(b)
	return "(?" + string(b) + ")" + rest
}

// View creates the tester of *pattern on *sample. Give it a height, or
// Grow; the matches scroll when they do not fit.
func View(c *ui.Context, s *State, pattern, sample *string) ui.Element {
	t := c.Theme()
	if s.re == nil && s.err == nil || s.pattern != *pattern {
		s.pattern = *pattern
		s.re, s.err = regexp.Compile(*pattern)
	}
	dark := t.Dark
	return ui.Column(c).Gap(t.Space(3)).Children(func() {
		ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
			in := ui.TextInput(c, pattern).Font("monospace").Placeholder("Regular expression").Label("Pattern").Grow(1)
			in.TextRanges(syntax(*pattern, t, dark)...)
			set := ""
			if m := leading.FindStringSubmatch(*pattern); m != nil {
				set = m[1]
			}
			ui.ToggleGroup(c, func() {
				for k, f := range flags {
					on := strings.Contains(set, f.flag)
					if ui.Toggle(c.Key(k), &on, f.flag).Font("monospace").Tooltip(f.name).Changed() {
						*pattern = withFlag(*pattern, f.flag, on)
					}
				}
			}).Label("Flags")
		})
		if s.err != nil {
			ui.Text(c, problem(s.err)).FontSize(t.FontSize - 1).TextColor(t.Danger)
		}
		var matches [][]int
		if s.re != nil && *pattern != "" {
			matches = s.re.FindAllStringSubmatchIndex(*sample, 1000)
		}
		area := ui.TextArea(c, sample).Font("monospace").Label("Sample").Height(t.Space(40))
		area.TextRanges(highlights(*sample, matches, dark)...)
		if s.re == nil {
			return
		}
		ui.Text(c, summary(len(matches), s.re.NumSubexp())).FontSize(t.FontSize - 1).TextColor(t.TextMuted)
		names := s.re.SubexpNames()
		ui.Scroll(c).Grow(1).Children(func() {
			ui.Column(c).Gap(t.Space(1)).Children(func() {
				for i, m := range matches[:min(len(matches), 100)] {
					match(c.Key(i), i, m, *sample, names, dark)
				}
				if len(matches) > 100 {
					ui.Text(c, fmt.Sprintf("and %d more", len(matches)-100)).FontSize(t.FontSize - 1).TextColor(t.TextMuted)
				}
			})
		})
	})
}

// problem returns what is wrong with a pattern, without the package's
// prefix: "missing closing ): `(ab`".
func problem(err error) string {
	return strings.TrimPrefix(err.Error(), "error parsing regexp: ")
}

func summary(matches, groups int) string {
	plural := func(n int, word string) string {
		if n == 1 {
			return "1 " + word
		}
		return fmt.Sprintf("%d %ss", n, word)
	}
	if matches == 0 {
		return "No match  ·  " + plural(groups, "group")
	}
	return plural(matches, "match") + "  ·  " + plural(groups, "group")
}

// match shows match i, m, in sample: its number, its text and where it
// is, and its groups, each in its color, by name when it has one.
func match(c *ui.Context, i int, m []int, sample string, names []string, dark bool) {
	t := c.Theme()
	ui.Row(c).Wrap().Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
		ui.Text(c, fmt.Sprint(i+1)).FontSize(t.FontSize - 1).FontFeatures("tnum").TextColor(t.TextMuted).Width(t.Space(6))
		chip(c, sample[m[0]:m[1]], color(0, dark))
		ui.Text(c, fmt.Sprintf("%d–%d", runes(sample, m[0]), runes(sample, m[1]))).FontSize(t.FontSize - 2).
			FontFeatures("tnum").TextColor(t.TextMuted)
		for g := 1; g < len(m)/2; g++ {
			name := names[g]
			if name == "" {
				name = fmt.Sprint(g)
			}
			ui.Row(c.Key(g)).Gap(t.Space(1)).AlignItems(ui.Center).Children(func() {
				ui.Text(c, name).FontSize(t.FontSize - 2).TextColor(color(g, dark))
				if m[2*g] < 0 {
					ui.Text(c, "none").FontSize(t.FontSize - 2).TextColor(t.TextMuted)
					return
				}
				chip(c, sample[m[2*g]:m[2*g+1]], color(g, dark))
			})
		}
	})
}

// chip shows text as matched, in monospace on a tint of color, with
// spaces and line breaks shown.
func chip(c *ui.Context, text string, color ui.Color) {
	t := c.Theme()
	text = strings.NewReplacer("\n", "⏎", "\t", "⇥", " ", "·").Replace(text)
	if text == "" {
		text = "empty"
	}
	ui.Text(c, text).Font("monospace").FontSize(t.FontSize-1).SingleLine().Shrink(1).
		Padding(0, t.Space(1)).Radius(t.Space(1)).Background(color.Alpha(0.14)).TextColor(color)
}

// color returns the color of group g, 0 for the whole match.
func color(g int, dark bool) ui.Color {
	l := float32(0.55)
	if dark {
		l = 0.75
	}
	return ui.Oklch(l, 0.16, float32((250+g*67)%360))
}

// runes returns how many runes of s come before byte b.
func runes(s string, b int) int { return utf8.RuneCountInString(s[:b]) }

// highlights returns the ranges of sample that matches cover, bold in the
// match's color, and in them those of groups, in theirs.
func highlights(sample string, matches [][]int, dark bool) []ui.TextRange {
	var out []ui.TextRange
	at, n := 0, 0
	// offset returns the rune offset of byte b, counting on from the last
	// asked, as the offsets mostly come in order.
	offset := func(b int) int {
		if b < at {
			at, n = 0, 0
		}
		n += utf8.RuneCountInString(sample[at:b])
		at = b
		return n
	}
	for _, m := range matches {
		start, end := offset(m[0]), offset(m[1])
		out = append(out, ui.TextRange{Start: start, End: end, Color: color(0, dark), Weight: 600})
		for g := 1; g < len(m)/2; g++ {
			if m[2*g] >= 0 {
				out = append(out, ui.TextRange{Start: offset(m[2*g]), End: offset(m[2*g+1]), Color: color(g, dark)})
			}
		}
		offset(m[1])
	}
	return out
}

// syntax returns the ranges of pattern to color: the parentheses of each
// group in its color, escapes, classes and anchors in one, and
// quantifiers and alternation in another.
func syntax(pattern string, t *ui.Theme, dark bool) []ui.TextRange {
	rs := []rune(pattern)
	meta := ui.Oklch(0.6, 0.14, 300)
	quant := ui.Oklch(0.65, 0.15, 55)
	if dark {
		meta, quant = ui.Oklch(0.75, 0.12, 300), ui.Oklch(0.78, 0.13, 70)
	}
	var out []ui.TextRange
	add := func(start, end int, c ui.Color) { out = append(out, ui.TextRange{Start: start, End: end, Color: c}) }
	var open []ui.Color
	group := 0
	for i := 0; i < len(rs); i++ {
		switch r := rs[i]; {
		case r == '\\' && i+1 < len(rs):
			j := i + 2
			if strings.ContainsRune("pPx", rs[i+1]) && j < len(rs) && rs[j] == '{' {
				for j < len(rs) && rs[j] != '}' {
					j++
				}
				j = min(j+1, len(rs))
			}
			add(i, j, meta)
			i = j - 1
		case r == '[':
			j := i + 1
			if j < len(rs) && rs[j] == '^' {
				j++
			}
			if j < len(rs) && rs[j] == ']' {
				j++
			}
			for j < len(rs) && rs[j] != ']' {
				if rs[j] == '\\' {
					j++
				}
				j++
			}
			j = min(j+1, len(rs))
			add(i, j, meta)
			i = j - 1
		case r == '(':
			c := t.TextMuted
			end := i + 1
			if i+1 < len(rs) && rs[i+1] == '?' {
				rest := string(rs[i+2:])
				switch {
				case strings.HasPrefix(rest, "P<") || strings.HasPrefix(rest, "<"):
					group++
					c = color(group, dark)
					for end < len(rs) && rs[end] != '>' {
						end++
					}
					end = min(end+1, len(rs))
				default:
					for end < len(rs) && rs[end] != ':' && rs[end] != ')' {
						end++
					}
					if end < len(rs) && rs[end] == ':' {
						end++
					}
				}
			} else {
				group++
				c = color(group, dark)
			}
			open = append(open, c)
			add(i, end, c)
			i = end - 1
		case r == ')':
			c := t.TextMuted
			if len(open) > 0 {
				c, open = open[len(open)-1], open[:len(open)-1]
			}
			add(i, i+1, c)
		case r == '*' || r == '+' || r == '?' || r == '|':
			add(i, i+1, quant)
		case r == '{':
			j := i + 1
			for j < len(rs) && (rs[j] >= '0' && rs[j] <= '9' || rs[j] == ',') {
				j++
			}
			if j < len(rs) && rs[j] == '}' && j > i+1 {
				add(i, j+1, quant)
				i = j
			}
		case r == '^' || r == '$' || r == '.':
			add(i, i+1, meta)
		}
	}
	return out
}
