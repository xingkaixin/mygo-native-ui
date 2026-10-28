// Package querybar is a search field of qualifiers, as the searches of
// GitHub and Linear: terms written key:value narrow the search, and the
// other words search the text. A key the app knows shows in the accent
// color with its value bold; a key it does not know, or a value its key
// does not take once the caret leaves it, shows in the danger color. A minus before a term
// excludes what it matches, and quotes hold a value with spaces.
//
//	querybar.Input(c, &app.query, fields).Grow(1).Padding(6, 10).Placeholder("Filter")
//	q := querybar.Parse(app.query)
//
// Typing a word lists the keys starting with it, or holding it, and after
// a key's colon the values of that key, which the letters typed narrow;
// with the field empty, every key. Up and Down move among them, and
// Enter, Tab or a click puts the one chosen in: a key with its colon, a
// value with a space after it. Escape closes the list. Enter is the
// field's Submitted while the list is closed.
package querybar

import (
	"slices"
	"strings"
	"unicode"

	"github.com/egoist/mygo/ui"
)

// Field is a key the field knows: its name, a hint shown by it in the
// list, and the values it takes, or nil for any.
type Field struct {
	Key    string
	Hint   string
	Values []string
}

// Term is a qualifier of a query: Not when a minus excludes what it
// matches.
type Term struct {
	Key, Value string
	Not        bool
}

// Query is a search read: its qualifiers in order, and its other words
// joined by spaces.
type Query struct {
	Terms []Term
	Text  string
}

// Values returns the values of the terms of key that are not excluded.
func (q Query) Values(key string) []string {
	var out []string
	for _, t := range q.Terms {
		if !t.Not && strings.EqualFold(t.Key, key) {
			out = append(out, t.Value)
		}
	}
	return out
}

// token is a word of a query: its runes start up to end, and for a term
// the rune after its minus, if any, and the one after its colon.
type token struct {
	start, end int
	key        int
	value      int
	term       bool
}

// tokens splits text at spaces outside quotes.
func tokens(text []rune) []token {
	var out []token
	for i := 0; i < len(text); {
		if unicode.IsSpace(text[i]) {
			i++
			continue
		}
		t := token{start: i, key: i, value: -1}
		if text[i] == '-' {
			t.key = i + 1
		}
		quoted := false
		for ; i < len(text) && (quoted || !unicode.IsSpace(text[i])); i++ {
			switch {
			case text[i] == '"':
				quoted = !quoted
			case text[i] == ':' && !quoted && t.value < 0 && i > t.key:
				t.value, t.term = i+1, true
			}
		}
		t.end = i
		out = append(out, t)
	}
	return out
}

func unquote(s string) string {
	return strings.Trim(s, `"`)
}

// Parse reads a query.
func Parse(query string) Query {
	runes := []rune(query)
	var q Query
	var words []string
	for _, t := range tokens(runes) {
		if !t.term {
			words = append(words, unquote(string(runes[t.start:t.end])))
			continue
		}
		q.Terms = append(q.Terms, Term{
			Key:   string(runes[t.key : t.value-1]),
			Value: unquote(string(runes[t.value:t.end])),
			Not:   t.key > t.start,
		})
	}
	q.Text = strings.Join(words, " ")
	return q
}

func find(fields []Field, key string) *Field {
	for i := range fields {
		if strings.EqualFold(fields[i].Key, key) {
			return &fields[i]
		}
	}
	return nil
}

// Ranges returns the runs of text that are terms, styled as the field
// shows them.
func Ranges(text string, fields []Field, t *ui.Theme) []ui.TextRange {
	return ranges(text, fields, t, -1)
}

// ranges is Ranges, leaving the value the caret is typing out of the
// danger color.
func ranges(text string, fields []Field, t *ui.Theme, caret int) []ui.TextRange {
	runes := []rune(text)
	var out []ui.TextRange
	for _, tok := range tokens(runes) {
		if !tok.term {
			continue
		}
		f := find(fields, string(runes[tok.key:tok.value-1]))
		if f == nil {
			out = append(out, ui.TextRange{Start: tok.start, End: tok.end, Color: t.Danger})
			continue
		}
		out = append(out, ui.TextRange{Start: tok.start, End: tok.value, Color: t.Accent})
		value := unquote(string(runes[tok.value:tok.end]))
		vr := ui.TextRange{Start: tok.value, End: tok.end, Color: t.Text, Weight: 600}
		if value != "" && f.Values != nil && caret != tok.end && !slices.ContainsFunc(f.Values, func(v string) bool { return strings.EqualFold(v, value) }) {
			vr.Color = t.Danger
		}
		out = append(out, vr)
	}
	return out
}

// item is a row of the list: a key, or a value of one.
type item struct {
	text, hint string
	key        bool
}

// state is what a field keeps: whether its list shows, the runes at up to
// end a pick replaces, the items matching what the caret types and the
// one the arrows are on; shut is where the list Escape closed was, while
// closed; put is where the caret goes once the field shows the text a
// pick made, while putting; seen is the text as the field last left it.
type state struct {
	open    bool
	at, end int
	typed   string
	found   []item
	on      int
	closed  bool
	shut    int
	put     int
	putting bool
	seen    string
}

// Input creates a text input without a look editing *query, whose keys
// fields are. Style it as TextInputBase, and name it with Label.
func Input(c *ui.Context, query *string, fields []Field) ui.Element {
	t := c.Theme()
	in := ui.TextInputBase(c, query)
	s := ui.Local(in, "querybar.state", func() state { return state{} })
	if s.putting {
		in.SetTextSelection(s.put, s.put)
		s.putting = false
	}
	anchor, caret := in.TextAnchorCaret()
	if !in.Focused() {
		caret = -1
	}
	in.TextRanges(ranges(*query, fields, t, caret)...)
	s.seen = *query
	s.open = caret >= 0 && !in.Composing() && anchor == caret && s.suggest([]rune(*query), caret, fields)
	in.OnTextKey(func(k ui.TextKey) bool { return s.key(k, query) })
	if !s.open {
		return in
	}
	open, picked := true, -1
	ui.PopoverBase(c, in, &open, func(panel ui.Element) {
		panel.Margin(t.Space(1), 0, 0, 0).Padding(t.Space(1)).Radius(t.Space(2.5)).Width(t.Space(72)).
			Background(t.Background).Border(1, t.Border).Shadow(0, 8, 24, 0, ui.RGBA(0, 0, 0, 0.16)).Label("Suggestions")
		for i, it := range s.found {
			if row(c, it, i == s.on).Clicked() {
				picked = i
			}
		}
	})
	switch {
	case picked >= 0:
		*query = s.pick(*query, s.found[picked])
		c.Invalidate()
	case !open && !in.Hovered():
		s.closed, s.shut, s.open = true, s.at, false
	}
	return in
}

// key takes the keys moving in the list, choosing from it and closing it,
// while it shows.
func (s *state) key(k ui.TextKey, query *string) bool {
	// What the field found is for the text it last showed, which keys
	// typed since may have changed.
	if !s.open || k.Text() != s.seen || k.Mods != 0 {
		return false
	}
	switch k.Key {
	case ui.KeyDown:
		s.on = min(s.on+1, len(s.found)-1)
		return true
	case ui.KeyUp:
		s.on = max(s.on-1, 0)
		return true
	case ui.KeyEnter, ui.KeyTab:
		*query = s.pick(k.Text(), s.found[s.on])
		k.SetText(*query)
		return true
	case ui.KeyEscape:
		s.closed, s.shut, s.open = true, s.at, false
		return true
	}
	return false
}

// suggest finds the key or value the caret types and the items matching
// it. It reports whether the list shows.
func (s *state) suggest(text []rune, caret int, fields []Field) bool {
	tok := token{start: caret, key: caret, end: caret, value: -1}
	for _, t := range tokens(text) {
		if t.start <= caret && caret <= t.end {
			tok = t
			break
		}
	}
	var found []item
	at, typed := tok.key, ""
	switch {
	case tok.term && caret >= tok.value:
		f := find(fields, string(text[tok.key:tok.value-1]))
		if f == nil {
			break
		}
		at, typed = tok.value, unquote(string(text[tok.value:caret]))
		for _, v := range rank(f.Values, typed) {
			found = append(found, item{text: v})
		}
	case caret > tok.key || len(text) == 0:
		typed = string(text[tok.key:caret])
		keys := make([]string, len(fields))
		for i, f := range fields {
			keys[i] = f.Key
		}
		for _, k := range rank(keys, typed) {
			found = append(found, item{text: k, hint: find(fields, k).Hint, key: true})
		}
	}
	if len(found) == 0 {
		s.closed = false
		return false
	}
	if s.closed && at == s.shut {
		return false
	}
	s.closed = false
	if !s.open || s.at != at || s.typed != typed {
		s.on = 0
	}
	s.at, s.end, s.typed, s.found = at, tok.end, typed, found
	s.on = min(s.on, len(found)-1)
	return true
}

// pick returns text with it in place of what the caret typed: a key with
// its colon, keeping the value after it, or a value, quoted if it holds a
// space, with a space after it. The caret goes after them.
func (s *state) pick(text string, it item) string {
	runes := []rune(text)
	end := s.end
	with := it.text
	if it.key {
		end = s.at
		for end < len(runes) && runes[end] != ':' && !unicode.IsSpace(runes[end]) {
			end++
		}
		if end < len(runes) && runes[end] == ':' {
			end++
		}
		with += ":"
	} else {
		if strings.ContainsFunc(with, unicode.IsSpace) {
			with = `"` + with + `"`
		}
		if end < len(runes) && unicode.IsSpace(runes[end]) {
			end++
		}
		with += " "
	}
	s.put, s.putting, s.open = s.at+len([]rune(with)), true, false
	return string(runes[:s.at]) + with + string(runes[end:])
}

// rank returns the names matching typed, ignoring case: those starting
// with it, then those holding it. With nothing typed, all of them, as
// given.
func rank(names []string, typed string) []string {
	q := strings.ToLower(typed)
	var starts, holds []string
	for _, n := range names {
		l := strings.ToLower(n)
		switch {
		case strings.HasPrefix(l, q):
			starts = append(starts, n)
		case strings.Contains(l, q):
			holds = append(holds, n)
		}
	}
	return append(starts, holds...)
}

// row creates the row of it in the list, lit while on: a key with its
// colon and its hint dimmed, or a value.
func row(c *ui.Context, it item, on bool) ui.Element {
	t := c.Theme()
	b := ui.ButtonBase(c).Gap(t.Space(3)).Padding(t.Space(1.5), t.Space(2)).Radius(t.Space(1.5)).
		Justify(ui.Start).AlignItems(ui.Center).Label(it.text).KeepFocus()
	if on || b.Hovered() {
		b.Background(t.Text.Alpha(0.07))
	}
	return b.Children(func() {
		if !it.key {
			ui.Text(c, it.text).SingleLine().Shrink(1)
			return
		}
		ui.RichText(c, ui.Span{Text: it.text, Color: t.Accent}, ui.Span{Text: ":", Color: t.TextMuted}).SingleLine().Shrink(1)
		ui.Text(c, it.hint).FontSize(t.FontSize - 1).TextColor(t.TextMuted).SingleLine().Shrink(100)
	})
}
