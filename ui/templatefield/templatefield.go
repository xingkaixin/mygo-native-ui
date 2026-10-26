// Package templatefield is a text field of one line holding {{variables}},
// as the URL fields of Postman and Insomnia do: a variable the app
// defines shows bold in the accent color, one it does not in the danger
// color. Typing "{{" opens a list of the variables by the field, which the
// letters typed after it narrow, those starting the name first; Up and
// Down move among them, and Enter, Tab or a click puts the one chosen in,
// closing it with "}}". Escape closes the list. With the pointer over the
// field, and the focus elsewhere, a card below it shows the value of each
// variable in it, dots for a secret, and where it comes from.
//
//	templatefield.Input(c, &app.url, app.env).Grow(1).Padding(6, 10).Placeholder("https://")
//
// Ranges colors variables in other inputs, as kvtable's values:
//
//	kvtable.EditorWith(c, &app.headers, kvtable.Options{
//		Ranges: func(v string) []ui.TextRange { return templatefield.Ranges(v, app.env, c.Theme()) },
//	})
package templatefield

import (
	"regexp"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
)

// Resolver is what the app knows of variables: the names of those it
// defines, and the value of one, where it comes from, as an environment's
// name, and whether it is a secret, or not ok when it is not defined.
type Resolver interface {
	Names() []string
	Lookup(name string) (value, source string, secret, ok bool)
}

// variable is a variable as a template writes it, "{{name}}", with spaces
// inside the braces or without.
var variable = regexp.MustCompile(`\{\{ *([\w.-]+) *\}\}`)

// span is a variable in a text: its runes start up to end, and its name.
type span struct {
	start, end int
	name       string
}

// spans returns the variables of text, in order.
func spans(text string) []span {
	var out []span
	for _, m := range variable.FindAllStringSubmatchIndex(text, -1) {
		out = append(out, span{
			start: utf8.RuneCountInString(text[:m[0]]),
			end:   utf8.RuneCountInString(text[:m[1]]),
			name:  text[m[2]:m[3]],
		})
	}
	return out
}

// Ranges returns the runs of text that are variables, styled as the field
// shows them: bold in t's accent color where r defines them, in its danger
// color where it does not.
func Ranges(text string, r Resolver, t *ui.Theme) []ui.TextRange {
	var out []ui.TextRange
	for _, v := range spans(text) {
		tr := ui.TextRange{Start: v.start, End: v.end, Color: t.Danger}
		if _, _, _, ok := r.Lookup(v.name); ok {
			tr.Color, tr.Weight = t.Accent, 600
		}
		out = append(out, tr)
	}
	return out
}

// state is what a field keeps: whether its list shows, for the name typed
// from rune at up to the caret at end, query, the names matching it and
// the one the arrows are on; shut is the list Escape closed, while closed;
// put is where the caret goes once the field shows the text a pick made,
// while putting; seen is the text as the field last left it.
type state struct {
	open    bool
	at, end int
	query   string
	found   []string
	on      int
	closed  bool
	shut    int
	put     int
	putting bool
	seen    string
}

// Input creates a text input without a look editing *text, whose
// variables r resolves. Style it as TextInputBase, and name it with Label.
// Enter is its Submitted while the list is closed.
func Input(c *ui.Context, text *string, r Resolver) ui.Element {
	t := c.Theme()
	in := ui.TextInputBase(c, text)
	s := ui.Local(in, "templatefield.state", func() state { return state{} })
	if s.putting {
		in.SetTextSelection(s.put, s.put)
		s.putting = false
	}
	in.TextRanges(Ranges(*text, r, t)...)
	anchor, caret := in.TextAnchorCaret()
	s.seen = *text
	s.open = in.Focused() && !in.Composing() && anchor == caret && s.suggest([]rune(*text), caret, r)
	in.OnTextKey(func(k ui.TextKey) bool { return s.key(k, text) })
	if !s.open {
		if in.Hovered() && !in.Focused() {
			card(c, in, *text, r)
		}
		return in
	}
	open, picked := true, ""
	ui.PopoverBase(c, in, &open, func(panel ui.Element) {
		panel.Margin(t.Space(1), 0, 0, 0).Padding(t.Space(1)).Radius(t.Space(2.5)).Width(t.Space(80)).
			Background(t.Background).Border(1, t.Border).Shadow(0, 8, 24, 0, ui.RGBA(0, 0, 0, 0.16)).Label("Variables")
		if len(s.found) == 0 {
			ui.Text(c, "No variables").TextColor(t.TextMuted).Padding(t.Space(1.5), t.Space(2))
			return
		}
		for i, name := range s.found {
			if row(c, name, r, i == s.on).Clicked() {
				picked = name
			}
		}
	})
	switch {
	case picked != "":
		*text = s.pick(*text, picked)
		c.Invalidate()
	case !open && !in.Hovered():
		s.closed, s.shut, s.open = true, s.at, false
	}
	return in
}

// key takes the keys moving in the list, choosing from it and closing it,
// while it shows.
func (s *state) key(k ui.TextKey, text *string) bool {
	// What the field found is for the text it last showed, which keys
	// typed since may have changed.
	if !s.open || k.Text() != s.seen || k.Mods != 0 {
		return false
	}
	switch k.Key {
	case ui.KeyDown, ui.KeyUp:
		if len(s.found) == 0 {
			return false
		}
		if k.Key == ui.KeyDown {
			s.on = min(s.on+1, len(s.found)-1)
		} else {
			s.on = max(s.on-1, 0)
		}
		return true
	case ui.KeyEnter, ui.KeyTab:
		if len(s.found) == 0 {
			return false
		}
		*text = s.pick(k.Text(), s.found[s.on])
		k.SetText(*text)
		return true
	case ui.KeyEscape:
		s.closed, s.shut, s.open = true, s.at, false
		return true
	}
	return false
}

// suggest finds the name the caret types, after "{{", and the names
// matching it. It reports whether the list shows.
func (s *state) suggest(text []rune, caret int, r Resolver) bool {
	at := caret
	for at > 0 && nameRune(text[at-1]) {
		at--
	}
	if at < 2 || text[at-1] != '{' || text[at-2] != '{' {
		s.closed = false
		return false
	}
	if s.closed && at == s.shut {
		return false
	}
	s.closed = false
	query := string(text[at:caret])
	if !s.open || s.at != at || s.query != query {
		s.on = 0
	}
	s.at, s.end, s.query = at, caret, query
	s.found = rank(r.Names(), query)
	s.on = min(s.on, max(len(s.found)-1, 0))
	return true
}

// pick returns text with name in place of the name typed after "{{", the
// rest of it after the caret too, closed with "}}", which it takes the
// place of where the text has it already. The caret goes after it.
func (s *state) pick(text, name string) string {
	runes := []rune(text)
	end := s.end
	for end < len(runes) && nameRune(runes[end]) {
		end++
	}
	rest := string(runes[end:])
	if after, ok := strings.CutPrefix(strings.TrimLeft(rest, " "), "}}"); ok {
		rest = after
	}
	with := name + "}}"
	s.put, s.putting, s.open = s.at+utf8.RuneCountInString(with), true, false
	return string(runes[:s.at]) + with + rest
}

// nameRune reports whether r may be in a variable's name.
func nameRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '.' || r == '-'
}

// rank returns the names matching query, ignoring case, best first: those
// starting with it, those containing it, and those with its letters in
// order. With no query, all of them, as given.
func rank(names []string, query string) []string {
	q := strings.ToLower(query)
	tier := func(name string) int {
		l := strings.ToLower(name)
		switch {
		case strings.HasPrefix(l, q):
			return 0
		case strings.Contains(l, q):
			return 1
		}
		rest := l
		for _, r := range q {
			i := strings.IndexRune(rest, r)
			if i < 0 {
				return -1
			}
			rest = rest[i+utf8.RuneLen(r):]
		}
		return 2
	}
	var out []string
	for _, n := range names {
		if tier(n) >= 0 {
			out = append(out, n)
		}
	}
	slices.SortStableFunc(out, func(a, b string) int { return tier(a) - tier(b) })
	return out
}

// row creates the row of name in the list, lit while on: the name, and its
// value dimmed.
func row(c *ui.Context, name string, r Resolver, on bool) ui.Element {
	t := c.Theme()
	b := ui.ButtonBase(c).Gap(t.Space(3)).Padding(t.Space(1.5), t.Space(2)).Radius(t.Space(1.5)).
		Justify(ui.Start).AlignItems(ui.Center).Label(name).KeepFocus()
	if on || b.Hovered() {
		b.Background(t.Text.Alpha(0.07))
	}
	value, _, secret, _ := r.Lookup(name)
	return b.Children(func() {
		ui.Text(c, name).SingleLine().Shrink(1)
		ui.Text(c, shown(value, secret)).FontSize(t.FontSize - 1).TextColor(t.TextMuted).SingleLine().Shrink(100)
	})
}

// card shows below in the value of each variable of text, once, and where
// it comes from.
func card(c *ui.Context, in ui.Element, text string, r Resolver) {
	vs := spans(text)
	if len(vs) == 0 {
		return
	}
	t := c.Theme()
	ui.Overlay(c, func() {
		ui.Column(c).AttachTo(in, ui.AnchorBottomStart, ui.AnchorTopStart).Margin(t.Space(1), 0, 0, 0).
			Padding(t.Space(2), t.Space(3)).Gap(t.Space(1.5)).MaxWidth(t.Space(120)).Radius(t.Space(2.5)).
			Background(t.Background).Border(1, t.Border).Shadow(0, 8, 24, 0, ui.RGBA(0, 0, 0, 0.16)).Children(func() {
			var seen []string
			for _, v := range vs {
				if slices.Contains(seen, v.name) {
					continue
				}
				seen = append(seen, v.name)
				value, source, secret, ok := r.Lookup(v.name)
				ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
					if !ok {
						ui.Text(c, v.name).TextColor(t.Danger).Bold().SingleLine()
						ui.Text(c, "Undefined").TextColor(t.TextMuted).SingleLine()
						return
					}
					ui.Text(c, v.name).TextColor(t.Accent).Bold().SingleLine()
					ui.Text(c, shown(value, secret)).SingleLine().Shrink(1)
					if source != "" {
						ui.Text(c, source).FontSize(t.FontSize - 1).TextColor(t.TextMuted).SingleLine()
					}
				})
			}
		})
	})
}

// shown returns value as the field shows it: dots for a secret.
func shown(value string, secret bool) string {
	if secret && value != "" {
		return "••••••••"
	}
	return value
}
