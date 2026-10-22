// Package masked is a text field that keeps its text in a pattern, as
// fields for phone numbers, dates and product keys do: the slots not filled
// yet show dimmed after the text, the pattern's other characters come in as
// the user types, the caret steps over them, and Backspace takes back the
// slot before them. A character that does not fit its slot is refused; a
// paste fills the slots with what fits of it and leaves out the rest.
//
// In a pattern, 9 is a slot for a digit, a for a letter, * for a letter or
// a digit, and h for a hexadecimal digit. Every other character shows as it
// is, and \ makes the next one show as it is too, as \9.
//
//	if phone, ok := masked.Field(c, &app.phone, "(999) 999-9999", '_').Done(); ok {
//		app.call(phone) // "4155550132"
//	}
//	masked.Format("(999) 999-9999", app.phone) // "(415) 555-0132"
//
// The app keeps the value: the characters of the slots, without the
// pattern's others. The field is a text input underneath: undo, input
// methods and assistive technology work as in one, and assistive
// technology reads the text as it shows, with the pattern's characters.
package masked

import (
	"slices"
	"strings"
	"unicode"

	"github.com/egoist/mygo/ui"
)

// Events is what the user did to a field in a frame.
type Events struct {
	value         string
	changed, done bool
}

// Changed reports whether the user changed the value in the frame.
func (e Events) Changed() bool { return e.changed }

// Done returns the value, and true in the frame the user changes it and
// it fills every slot.
func (e Events) Done() (string, bool) { return e.value, e.done }

// Format returns value in pattern, as a field shows it: "4155550132" in
// "(999) 999-9999" is "(415) 555-0132". The characters that do not fit
// their slots are left out; a value that fills only some slots ends after
// the last it fills.
func Format(pattern, value string) string {
	m := parse(pattern)
	text, _ := m.format(m.fill(nil, []rune(value), len(m.slots)))
	return text
}

// Complete reports whether value fills every slot of pattern.
func Complete(pattern, value string) bool {
	m := parse(pattern)
	return len(m.fill(nil, []rune(value), len(m.slots))) == len(m.slots)
}

// state is what a field keeps between frames: the last key the input saw,
// whether the selection was empty before it, and where the caret goes
// once the input holds the text the field set, or -1.
type state struct {
	key       ui.Key
	collapsed bool
	caret     int
}

// Field creates the field editing *value in pattern. The slots not filled
// yet show placeholder, or their own letter of the pattern when it is 0.
// The field is as wide as the pattern. A ui.Field around it names it.
func Field(c *ui.Context, value *string, pattern string, placeholder rune) Events {
	t := c.Theme()
	m := parse(pattern)
	n := len(m.slots)
	raw := m.fill(nil, []rune(*value), n)
	*value = string(raw)
	text, end := m.format(raw)
	edited := text
	padY, padX := t.Space(1.5), t.Space(2.5)
	font := ui.Font{Family: "monospace", Size: t.FontSize}
	w, _ := ui.ShapeText(m.rest(0, placeholder), font, 0).Size()
	// Beside the padding and the borders, a DIP keeps the caret after the
	// last slot in view.
	in := ui.TextInputBase(c, &edited).Font("monospace").FontSize(t.FontSize).Padding(padY, padX).
		Width(w+2*padX+2+1).Radius(t.Radius).Background(t.Surface).Border(1, t.Border)
	st := ui.Local(in, "masked.state", func() state { return state{caret: -1} })
	in.OnTextKey(func(k ui.TextKey) bool {
		st.key = k.Key
		return false
	})
	switch {
	case in.Focused():
		in.Border(1, t.Accent)
	case in.Hovered():
		in.Border(1, t.Border.Mix(t.Text, 0.25))
	}

	var ev Events
	anchor, caret := in.TextAnchorCaret()
	holds := true
	if edited != text {
		r, at := m.edit(raw, []rune(text), []rune(edited), caret, st.key, st.collapsed)
		if at >= 0 {
			// The input takes the text in the next build pass of the frame,
			// and the caret with it.
			st.caret, holds = at, false
		}
		if string(r) != *value {
			*value = string(r)
			ev = Events{value: *value, changed: true, done: len(r) == n}
		}
		raw = r
		text, end = m.format(raw)
	}
	if holds {
		if st.caret >= 0 {
			anchor, caret = st.caret, st.caret
			in.SetTextSelection(caret, caret)
			st.caret = -1
		} else if anchor == caret {
			if to := m.snap(caret, len(raw), len([]rune(text)), st.key == ui.KeyRight); to != caret {
				anchor, caret = to, to
				in.SetTextSelection(to, to)
			}
		}
	}
	st.collapsed = !holds || anchor == caret
	st.key = ui.KeyUnknown

	if rest := m.rest(end, placeholder); rest != "" && !in.Composing() {
		x, _ := ui.ShapeText(text, font, 0).Size()
		tail := ui.ShapeText(rest, font, 0)
		muted := t.TextMuted
		in.Draw(func(p *ui.Painter, r ui.Rect) {
			p.TextLayout(tail, r.X+1+padX+x, r.Y+1+padY, muted)
		})
	}
	return ev
}

// item is a character of a pattern: a slot, r its class, or a literal.
type item struct {
	r    rune
	slot bool
}

// mask is a parsed pattern: its items, and the indexes of its slots among
// them, which are those of their runes in the text.
type mask struct {
	items []item
	slots []int
}

func parse(pattern string) mask {
	var m mask
	escaped := false
	for _, r := range pattern {
		if r == '\\' && !escaped {
			escaped = true
			continue
		}
		slot := !escaped && strings.ContainsRune("9a*h", r)
		if slot {
			m.slots = append(m.slots, len(m.items))
		}
		m.items = append(m.items, item{r, slot})
		escaped = false
	}
	return m
}

// fits reports whether r may fill a slot of class.
func fits(class, r rune) bool {
	digit := r >= '0' && r <= '9'
	switch class {
	case '9':
		return digit
	case 'h':
		return digit || r >= 'a' && r <= 'f' || r >= 'A' && r <= 'F'
	case 'a':
		return unicode.IsLetter(r)
	}
	return digit || unicode.IsLetter(r)
}

// fill appends to raw the runes of s that fit the slots after it, in
// order, leaving out those that do not, until raw holds n.
func (m mask) fill(raw, s []rune, n int) []rune {
	for _, r := range s {
		if len(raw) >= n {
			break
		}
		if fits(m.items[m.slots[len(raw)]].r, r) {
			raw = append(raw, r)
		}
	}
	return raw
}

// format returns the text of raw: the pattern up to its last slot raw
// fills, to its end once raw fills them all, and the item it stops at.
func (m mask) format(raw []rune) (string, int) {
	var b strings.Builder
	k := 0
	for i, it := range m.items {
		if k == len(raw) && (it.slot || k < len(m.slots)) {
			return b.String(), i
		}
		if it.slot {
			b.WriteRune(raw[k])
			k++
		} else {
			b.WriteRune(it.r)
		}
	}
	return b.String(), len(m.items)
}

// rest returns the pattern from item i, with placeholder in its slots, or
// their own letters when it is 0.
func (m mask) rest(i int, placeholder rune) string {
	var b strings.Builder
	for _, it := range m.items[i:] {
		if it.slot && placeholder != 0 {
			b.WriteRune(placeholder)
		} else {
			b.WriteRune(it.r)
		}
	}
	return b.String()
}

// after returns where the caret goes once k runes of raw come before it:
// after the k-th, or before the first slot.
func (m mask) after(k int, raw []rune) int {
	switch {
	case k > 0:
		return m.slots[k-1] + 1
	case len(raw) > 0:
		return m.slots[0]
	}
	return 0
}

// snap returns where the caret rests nearest to i, in a text of n runes
// with k slots filled: before the first slot, after a slot's rune, or at
// the end. It looks forward first, as Right moved it, and back otherwise,
// as Left and a click between characters of the pattern do.
func (m mask) snap(i, k, n int, forward bool) int {
	stops := []int{n}
	if k > 0 {
		stops = []int{m.slots[0]}
		for j := range k {
			stops = append(stops, m.slots[j]+1)
		}
		if stops[len(stops)-1] != n {
			stops = append(stops, n)
		}
	}
	if slices.Contains(stops, i) {
		return i
	}
	next := slices.IndexFunc(stops, func(s int) bool { return s > i })
	switch {
	case next < 0:
		return stops[len(stops)-1]
	case next == 0 || forward:
		return stops[next]
	}
	return stops[next-1]
}

// edit returns raw after the user changed its text to edited, the caret
// then at caret, and where the caret goes in the text of what it returns:
// -1 when edited is that text already, the caret staying where it is. key
// is the last key the input saw, and collapsed tells whether the selection
// was empty before it.
func (m mask) edit(raw, text, edited []rune, caret int, key ui.Key, collapsed bool) ([]rune, int) {
	n := len(m.slots)
	var shown []rune
	for _, i := range m.slots {
		if i < len(edited) {
			shown = append(shown, edited[i])
		}
	}
	// Undo and redo put back a text the field showed, and typing at the
	// end often makes one.
	r := m.fill(nil, shown, n)
	if s, _ := m.format(r); s == string(edited) {
		return r, -1
	}

	// What follows the caret was there before, unless an input method
	// replaced text elsewhere.
	s := len(edited) - caret
	if s < 0 || s > len(text) || !slices.Equal(text[len(text)-s:], edited[caret:]) {
		s = 0
		for s < len(text) && s < len(edited) && text[len(text)-1-s] == edited[len(edited)-1-s] {
			s++
		}
	}
	p := 0
	for p < len(text)-s && p < len(edited)-s && text[p] == edited[p] {
		p++
	}
	inserted := edited[p : len(edited)-s]
	from := 0
	for from < len(raw) && m.slots[from] < p {
		from++
	}
	to := from
	for to < len(raw) && m.slots[to] < len(text)-s {
		to++
	}
	if from == to && len(inserted) == 0 && collapsed {
		// Backspace and Delete took a character of the pattern: they take
		// the slot before or after it.
		switch {
		case key == ui.KeyBackspace && from > 0:
			from--
		case key == ui.KeyDelete && to < len(raw):
			to++
		}
	}
	tail := raw[to:]
	r = m.fill(slices.Clone(raw[:from]), inserted, n-len(tail))
	if len(inserted) > 0 && len(r) == from {
		return raw, m.after(from, raw)
	}
	k := len(r)
	r = m.fill(r, tail, n)
	return r, m.after(k, r)
}
