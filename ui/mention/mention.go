// Package mention is a text field that mentions files, people and other
// things by name, as the prompt fields of Cursor and Claude do. A trigger
// typed at the start or after a space, "@" say, opens a list by it of what
// can be mentioned, which the text typed after the trigger narrows: those
// whose labels hold its letters in order, or else whose details hold it,
// best first, the letters that match in the accent color. Up and Down move
// among them, as the pointer does, and Enter, Tab or a click puts the one
// chosen in the text, followed by a space. Escape closes the list, as does
// a space after which nothing matches.
//
// A mention shows tinted, its kind's icon in place of its trigger, and is
// one piece of the text: the caret steps over it, Backspace and Delete
// take it out whole, and a selection holds all of it or none. It copies as
// the text it is, "@ui/mention/mention.go".
//
//	mention.View(c, &app.prompt, "Ask anything", "@#", app.find)
//	if app.prompt.Submitted() {
//		app.send(app.prompt.Text, app.prompt.Mentions)
//		app.prompt.Text, app.prompt.Mentions = "", nil
//	}
package mention

import (
	"cmp"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
)

// Kind is what an item is, which its icon shows.
type Kind uint8

const (
	File Kind = iota
	Folder
	Symbol
	Person
	Issue
)

// Item is what can be mentioned: its label, which follows the trigger in
// the text, as "ui/mention/mention.go" or "ada"; a detail shown dimmed
// after it in the list, as a person's full name; and its kind.
type Item struct {
	Label, Detail string
	Kind          Kind
}

// Mention is an item mentioned in a field's text: its runes Start up to
// End, the trigger typed for it followed by the item's label.
type Mention struct {
	Start, End int
	Trigger    rune
	Item       Item
}

// String returns m as the text holds it, "@ui/mention/mention.go".
func (m Mention) String() string { return string(m.Trigger) + m.Item.Label }

// State is what a field keeps: its text and the mentions in it, in order,
// which the app may set together, as it empties the field once sent or
// puts a message back to edit (a mention the text does not hold where it
// says is dropped), and whether Enter submitted it in the frame. Keep it
// in the app's state, one for each field.
type State struct {
	Text     string
	Mentions []Mention

	// seen is the text as the field last left it, which tells the app's
	// changes from the user's.
	seen   string
	lines  ui.TextLines
	scroll ui.ScrollState
	list   ui.ListState
	// anchor and caret are the selection as the field last left it; put
	// is where the caret goes once the field shows a text it changed
	// itself, while putting.
	anchor, caret, put int
	putting            bool
	// open is whether menu shows; shut is the trigger whose list the user
	// closed, while closed.
	open   bool
	menu   menu
	shut   int
	closed bool
	// queued is Enter pressed between frames, which the next reports.
	queued    bool
	submitted bool
}

// menu is the list a trigger opened: the trigger, rune at of the text, the
// query typed after it up to the caret at rune end, the items matching
// it, the one the arrows are on, and the one last under the pointer.
type menu struct {
	at, end     int
	trigger     rune
	query       string
	found       []found
	on, hovered int
}

// found is an item matching the query: the runes of its label, or of its
// detail, that match, and how well.
type found struct {
	item          Item
	label, detail []int
	tier, span    int
}

// Submitted reports whether Enter submitted the field in this frame, with
// the list closed and text in it, after View. The app reads Text and
// Mentions, and empties them.
func (s *State) Submitted() bool { return s.submitted }

// View creates the field of s, showing placeholder while empty, from one
// line up to eight, which takes the focus as it appears. Each rune of
// triggers opens the list, as "@" or "@#", of what find gives for it and
// the text typed after it; find runs in each frame the list shows, and
// may give all it has, as the field narrows them.
//
// Enter submits, as Submitted reports, unless the list shows, and
// Shift+Enter breaks the line.
func View(c *ui.Context, s *State, placeholder, triggers string, find func(trigger rune, query string) []Item) ui.Element {
	t := c.Theme()
	// What the app set since takes the place of what the field left.
	s.Mentions, s.seen = valid([]rune(s.Text), s.Mentions), s.Text
	padY, padX := t.Space(1.5), t.Space(2.5)
	font := ui.Font{Family: t.Font, Size: t.FontSize}
	box := ui.Column(c).Radius(t.Radius).Background(t.Surface).Border(1, t.Border)
	box.Children(func() {
		area := ui.TextAreaBase(c, &s.Text).Lines(1, 8).Placeholder(placeholder).Label(placeholder).
			Padding(padY, padX).TrackLines(&s.lines).TrackScroll(&s.scroll).AutoFocus()
		s.submitted, s.queued = s.queued, false
		text, cut := []rune(s.Text), false
		if s.Text != s.seen {
			_, caret := area.TextAnchorCaret()
			text, cut = s.edited([]rune(s.seen), text, caret)
		}
		show := false
		if cut {
			// The field shows what the user left of a mention until the next
			// frame takes the text without it.
			s.anchor, s.caret = s.put, s.put
			c.Invalidate()
		} else {
			if s.putting {
				area.SetTextSelection(s.put, s.put)
				s.putting = false
			}
			anchor, caret := area.TextAnchorCaret()
			if a, k := s.snap(anchor, caret); a != anchor || k != caret {
				area.SetTextSelection(a, k)
				anchor, caret = a, k
			}
			s.anchor, s.caret = anchor, caret
			show = area.Focused() && !area.Composing() && anchor == caret && s.suggest(text, caret, triggers, find)
		}
		s.open = show
		s.seen = s.Text

		if area.Focused() {
			box.Border(1, t.Accent)
		}
		ranges := make([]ui.TextRange, 0, 2*len(s.Mentions))
		for _, m := range s.Mentions {
			ranges = append(ranges,
				ui.TextRange{Start: m.Start, End: m.End, Color: t.Accent},
				// The trigger all but disappears, for the icon to show in its place.
				ui.TextRange{Start: m.Start, End: m.Start + 1, Color: t.Accent.Alpha(0.004)})
		}
		area.TextRanges(ranges...)
		mentions := s.Mentions
		tint := t.Accent.Alpha(0.13)
		if t.Dark {
			tint = t.Accent.Alpha(0.22)
		}
		area.Draw(func(p *ui.Painter, r ui.Rect) {
			p.Clip(r, 0, func() {
				for _, m := range mentions {
					for _, b := range s.rects(text, m.Start, m.End, font) {
						p.Fill(ui.Rect{X: r.X + padX + b.X - 2, Y: r.Y + padY + b.Y - s.scroll.Y + 1, W: b.W + 4, H: b.H - 2}, tint, 4)
					}
				}
			})
		})
		area.DrawOver(func(p *ui.Painter, r ui.Rect) {
			p.Clip(r, 0, func() {
				for _, m := range mentions {
					if b := s.rects(text, m.Start, m.Start+1, font); len(b) > 0 {
						d := min(max(b[0].W, b[0].H*0.56), b[0].H*0.7)
						x, y := r.X+padX+b[0].X+(b[0].W-d)/2, r.Y+padY+b[0].Y-s.scroll.Y+(b[0].H-d)/2
						icon(p, ui.Rect{X: x, Y: y, W: d, H: d}, m.Item.Kind, t.Accent)
					}
				}
			})
		})
		area.OnTextKey(s.key)
		if !show {
			return
		}
		at := s.rects(text, s.menu.at, s.menu.at+1, font)
		// The list goes while the trigger is scrolled out of the field.
		if len(at) == 0 || at[0].Y+at[0].H <= s.scroll.Y || at[0].Y >= s.scroll.Y+area.LayoutBounds().H-2*padY {
			s.open = false
			return
		}
		// The list's icons line up under the trigger, past the panel's
		// padding and the row's.
		mark := ui.Box(c).Absolute().Left(padX+at[0].X-t.Space(3)).Top(padY+at[0].Y-s.scroll.Y).Size(1, at[0].H)
		open, picked := true, -1
		var active ui.Element
		ui.PopoverBase(c, mark, &open, func(panel ui.Element) {
			panel.Margin(t.Space(1), 0, 0, 0).Padding(t.Space(1)).Radius(t.Space(2.5)).Clip().
				Background(t.Background).Border(1, t.Border).Shadow(0, 8, 24, 0, ui.RGBA(0, 0, 0, 0.16)).Label("Suggestions")
			m := &s.menu
			if len(m.found) == 0 {
				ui.Text(c, "No matches").TextColor(t.TextMuted).Padding(t.Space(1.5), t.Space(2))
				return
			}
			panel.Width(t.Space(96))
			hovered := -1
			ui.List(c, &s.list, len(m.found), func(i int) {
				r := row(c, m.found[i], i == m.on)
				if r.Hovered() {
					hovered = i
				}
				if r.Clicked() {
					picked = i
				}
				if i == m.on {
					active = r
				}
			}).MaxHeight(t.Space(64)).KeepFocus()
			// The pointer moving onto a row puts the arrows there; resting
			// there, it leaves the arrows to move on.
			if hovered != m.hovered && hovered >= 0 {
				m.on = hovered
			}
			m.hovered = hovered
		})
		switch {
		case picked >= 0:
			s.pick(s.menu.found[picked].item)
			c.Invalidate()
		case !open && !area.Hovered():
			s.closed, s.shut, s.open = true, s.menu.at, false
		case active.Valid():
			area.ActiveDescendant(active)
		}
	})
	return box
}

// key takes the keys the field acts on itself, as they come: those moving
// in the list and choosing from it or closing it while it shows, Enter
// submitting, and Backspace and Delete next to a mention.
func (s *State) key(k ui.TextKey) bool {
	// What the field found is for the text it last showed, which keys
	// typed since may have changed.
	fresh := k.Text() == s.seen
	m := &s.menu
	plain := k.Mods == 0
	if s.open && fresh {
		switch {
		case plain && (k.Key == ui.KeyDown || k.Key == ui.KeyUp) && len(m.found) > 0:
			if k.Key == ui.KeyDown {
				m.on = min(m.on+1, len(m.found)-1)
			} else {
				m.on = max(m.on-1, 0)
			}
			s.list.ScrollIntoView(m.on)
			return true
		case plain && (k.Key == ui.KeyEnter || k.Key == ui.KeyTab) && len(m.found) > 0:
			s.pick(m.found[m.on].item)
			k.SetText(s.Text)
			return true
		case k.Key == ui.KeyEscape:
			s.closed, s.shut, s.open = true, m.at, false
			return true
		}
	}
	switch k.Key {
	case ui.KeyEnter:
		if !plain {
			return false
		}
		s.queued = strings.TrimSpace(k.Text()) != ""
		return true
	case ui.KeyBackspace, ui.KeyDelete:
		// Alt and Ctrl delete a word, which a mention is.
		if !fresh || s.anchor != s.caret || k.Mods&^(ui.Alt|ui.Ctrl) != 0 {
			return false
		}
		for _, x := range s.Mentions {
			if k.Key == ui.KeyBackspace && x.End == s.caret || k.Key == ui.KeyDelete && x.Start == s.caret {
				s.replace(x.Start, x.End, "", nil)
				k.SetText(s.Text)
				return true
			}
		}
	}
	return false
}

// pick puts item in place of the trigger of the list and the query after
// it, as a mention followed by a space, and closes the list.
func (s *State) pick(it Item) {
	m := Mention{Start: s.menu.at, Trigger: s.menu.trigger, Item: it}
	with := m.String()
	m.End = m.Start + utf8.RuneCountInString(with)
	if rest := []rune(s.Text)[s.menu.end:]; len(rest) == 0 || !unicode.IsSpace(rest[0]) {
		with += " "
	}
	s.replace(m.Start, s.menu.end, with, &m)
	s.put = m.End + 1
	s.open = false
}

// replace puts with in place of runes from up to to of the text, and m
// there as a mention unless nil, moving the mentions after it; the caret
// goes after what it put.
func (s *State) replace(from, to int, with string, m *Mention) {
	text := []rune(s.Text)
	w := []rune(with)
	d := len(w) - (to - from)
	var ms []Mention
	for _, x := range s.Mentions {
		switch {
		case x.End <= from:
			ms = append(ms, x)
		case x.Start >= to:
			x.Start, x.End = x.Start+d, x.End+d
			ms = append(ms, x)
		}
	}
	if m != nil {
		ms = append(ms, *m)
		slices.SortFunc(ms, func(a, b Mention) int { return cmp.Compare(a.Start, b.Start) })
	}
	s.Text = string(slices.Concat(text[:from], w, text[to:]))
	s.Mentions, s.seen = ms, s.Text
	s.put, s.putting = from+len(w), true
}

// valid returns the mentions of ms that text holds where they say, in
// order, as the app may set them: ms itself when they all do.
func valid(text []rune, ms []Mention) []Mention {
	holds := func(m Mention, end int) bool {
		return m.Start >= end && m.Start < m.End && m.End <= len(text) && string(text[m.Start:m.End]) == m.String()
	}
	end := 0
	for _, m := range ms {
		if !holds(m, end) {
			var out []Mention
			end = 0
			for _, m := range slices.SortedFunc(slices.Values(ms), func(a, b Mention) int { return cmp.Compare(a.Start, b.Start) }) {
				if holds(m, end) {
					out = append(out, m)
					end = m.End
				}
			}
			return out
		}
		end = m.End
	}
	return ms
}

// edited takes the user's edit of the text, from before to after, the
// caret then at caret: the mentions around it move with the text, those
// it took out go, and what is left of those it cut into goes too, as a
// mention is one piece. It returns the text, and whether it cut into one,
// for the next frame to show the text without it.
func (s *State) edited(before, after []rune, caret int) ([]rune, bool) {
	// Where the edit could be in either of several places, as a rune typed
	// next to the same rune, it is where the selection was.
	p, q := diff(before, after, len(before), 0)
	lo, hi := min(s.anchor, s.caret, len(before)), min(max(s.anchor, s.caret), len(before))
	if lp, lq := diff(before, after, lo, hi); lp+lq == p+q {
		p, q = lp, lq
	}
	end, newEnd := len(before)-q, len(after)-q
	var kept []Mention
	var cuts [][2]int
	for _, m := range s.Mentions {
		switch {
		case m.End <= p:
			kept = append(kept, m)
		case m.Start >= end:
			m.Start, m.End = m.Start+newEnd-end, m.End+newEnd-end
			kept = append(kept, m)
		default:
			if m.Start < p {
				cuts = append(cuts, [2]int{m.Start, p})
			}
			if m.End > end {
				cuts = append(cuts, [2]int{newEnd, newEnd + m.End - end})
			}
		}
	}
	for i := len(cuts) - 1; i >= 0; i-- {
		a, b := cuts[i][0], cuts[i][1]
		after = slices.Delete(after, a, b)
		for j := range kept {
			if kept[j].Start >= b {
				kept[j].Start, kept[j].End = kept[j].Start-(b-a), kept[j].End-(b-a)
			}
		}
		if caret >= b {
			caret -= b - a
		} else if caret > a {
			caret = a
		}
	}
	s.Mentions = kept
	if len(cuts) == 0 {
		return after, false
	}
	s.Text = string(after)
	s.put, s.putting = caret, true
	return after, true
}

// diff returns how many runes before and after have the same at their
// starts, up to lo, and at their ends past those, up to the end of
// before from hi.
func diff(before, after []rune, lo, hi int) (p, q int) {
	for p < lo && p < len(after) && before[p] == after[p] {
		p++
	}
	for q < len(before)-max(p, hi) && q < len(after)-p && before[len(before)-1-q] == after[len(after)-1-q] {
		q++
	}
	return p, q
}

// snap returns the selection from anchor to caret with neither end inside
// a mention: a caret that came into one from an edge, as an arrow moves
// it, goes on to the other edge, and one put in it otherwise, as by a
// click, to the nearer; an anchor goes out of it away from the caret; and
// a selection inside one holds it whole, as a double click on a word of it
// selects.
func (s *State) snap(anchor, caret int) (int, int) {
	for _, m := range s.Mentions {
		in := func(i int) bool { return m.Start < i && i < m.End }
		edge := m.Start
		if s.caret == m.Start || s.caret != m.End && m.End-caret < caret-m.Start {
			edge = m.End
		}
		switch {
		case anchor == caret && in(caret):
			anchor, caret = edge, edge
		case in(anchor) && in(caret):
			if anchor < caret {
				anchor, caret = m.Start, m.End
			} else {
				anchor, caret = m.End, m.Start
			}
		case in(caret):
			caret = edge
		case in(anchor):
			if anchor < caret {
				anchor = m.Start
			} else {
				anchor = m.End
			}
		}
	}
	return anchor, caret
}

// suggest finds the trigger the caret types after, the nearest before it
// on its line at the start of the text or after a space, with no mention
// between, and what matches the text from it to the caret. A space in that
// text keeps a list showing, while something matches, but opens none. It
// reports whether the list shows.
func (s *State) suggest(text []rune, caret int, triggers string, find func(rune, string) []Item) bool {
	at := -1
	for i := caret - 1; i >= 0 && text[i] != '\n'; i-- {
		if strings.ContainsRune(triggers, text[i]) && (i == 0 || unicode.IsSpace(text[i-1])) {
			at = i
			break
		}
	}
	if s.closed && at != s.shut {
		s.closed = false
	}
	if at < 0 || s.closed || slices.ContainsFunc(s.Mentions, func(m Mention) bool { return m.Start < caret && m.End > at }) {
		return false
	}
	query := string(text[at+1 : caret])
	spaced := strings.ContainsFunc(query, unicode.IsSpace)
	if strings.HasPrefix(query, " ") || spaced && !(s.open && s.menu.at == at) {
		return false
	}
	m := &s.menu
	if !s.open || m.at != at || m.query != query {
		m.on = 0
		s.list.ScrollTo(0, ui.Start)
	}
	m.at, m.end, m.trigger, m.query = at, caret, text[at], query
	m.found = rank(find(text[at], query), query)
	m.on = min(m.on, max(len(m.found)-1, 0))
	return !spaced || len(m.found) > 0
}

// rank returns the items matching query, best first: those whose labels
// start with it, those with a word starting with it, those containing it,
// and those with its letters scattered, each with the letters closest
// together first, and after them those whose details hold it, as prose
// holds its letters scattered by chance. With no query, all of them, as
// given.
func rank(items []Item, query string) []found {
	q := []rune(strings.ToLower(query))
	var out []found
	for _, it := range items {
		f := found{item: it}
		if len(q) > 0 {
			tier, at, ok := match(it.Label, q)
			f.label = at
			if !ok {
				if tier, at, ok = match(it.Detail, q); !ok || tier == 3 {
					continue
				}
				f.label, f.detail = nil, at
				tier += 4
			}
			f.tier, f.span = tier, at[len(at)-1]-at[0]
		}
		out = append(out, f)
	}
	slices.SortStableFunc(out, func(a, b found) int {
		return cmp.Or(cmp.Compare(a.tier, b.tier), cmp.Compare(a.span, b.span))
	})
	return out
}

// match reports whether s holds the runes of q, in lower case, in order,
// how well, from 0 at its start to 3 scattered, and which runes of s they
// are.
func match(s string, q []rune) (int, []int, bool) {
	runes := []rune(s)
	lower := make([]rune, len(runes))
	for i, r := range runes {
		lower[i] = unicode.ToLower(r)
	}
	best, tier := -1, 3
	for i := 0; i+len(q) <= len(lower) && tier > 0; i++ {
		if !slices.Equal(lower[i:i+len(q)], q) {
			continue
		}
		t := 2
		if i == 0 {
			t = 0
		} else if wordStart(runes, i) {
			t = 1
		}
		if t < tier || best < 0 {
			best, tier = i, t
		}
	}
	var at []int
	if best >= 0 {
		for i := range q {
			at = append(at, best+i)
		}
		return tier, at, true
	}
	j := 0
	for _, r := range q {
		i := slices.Index(lower[j:], r)
		if i < 0 {
			return 0, nil, false
		}
		at = append(at, j+i)
		j += i + 1
	}
	return 3, at, true
}

// wordStart reports whether a word starts at rune i of s, after a rune
// that is not a letter or a digit, or as an upper-case letter after a
// lower-case one.
func wordStart(s []rune, i int) bool {
	prev := s[i-1]
	if !unicode.IsLetter(prev) && !unicode.IsDigit(prev) {
		return true
	}
	return unicode.IsLower(prev) && unicode.IsUpper(s[i])
}

// rects returns the boxes of runes from up to to of text, all on one line,
// a box for each row of it they show on, in the field's content before it
// scrolls.
func (s *State) rects(text []rune, from, to int, font ui.Font) []ui.Rect {
	line, start := 0, 0
	for i, r := range text[:from] {
		if r == '\n' {
			line, start = line+1, i+1
		}
	}
	end := start
	for end < len(text) && text[end] != '\n' {
		end++
	}
	rows := s.lines.Rows(line)
	top := s.lines.Top(line)
	h := (s.lines.Top(line+1) - top) / float32(len(rows))
	var out []ui.Rect
	for k, rs := range rows {
		re := end - start
		if k+1 < len(rows) {
			re = rows[k+1]
		}
		a, b := max(from-start, rs), min(to-start, re)
		if a >= b {
			continue
		}
		row := text[start+rs : start+re]
		l := ui.ShapeText(string(row), font, 0)
		x0 := l.Caret(ui.UTF16Len(string(row[:a-rs]))).X
		x1 := l.Caret(ui.UTF16Len(string(row[:b-rs]))).X
		out = append(out, ui.Rect{X: x0, Y: top + float32(k)*h, W: x1 - x0, H: h})
	}
	return out
}

// row creates the row of f in the list, lit while on: its icon, its label
// and its detail, the runes matching in the accent color.
func row(c *ui.Context, f found, on bool) ui.Element {
	t := c.Theme()
	r := ui.ButtonBase(c).Gap(t.Space(2)).Padding(t.Space(1.5), t.Space(2)).Radius(t.Space(1.5)).
		Justify(ui.Start).AlignItems(ui.Center).Label(f.item.Label).KeepFocus()
	color := t.TextMuted
	if on {
		r.Background(t.Text.Alpha(0.07))
		color = t.Text
	}
	return r.Children(func() {
		ui.Box(c).Size(t.Space(4), t.Space(4)).Draw(func(p *ui.Painter, b ui.Rect) { icon(p, b, f.item.Kind, color) })
		ui.RichText(c, spans(f.item.Label, f.label, t.Accent)...).SingleLine().Shrink(1)
		if f.item.Detail != "" {
			ui.RichText(c, spans(f.item.Detail, f.detail, t.Accent)...).FontSize(t.FontSize - 1).
				TextColor(t.TextMuted).SingleLine().Shrink(100)
		}
	})
}

// spans returns s in spans, the runes matched bold in color.
func spans(s string, matched []int, color ui.Color) []ui.Span {
	if len(matched) == 0 {
		return []ui.Span{{Text: s}}
	}
	var out []ui.Span
	runes := []rune(s)
	for i, k := 0, 0; i < len(runes); {
		hit := k < len(matched) && matched[k] == i
		j := i
		for j < len(runes) && (k < len(matched) && matched[k] == j) == hit {
			if hit {
				k++
			}
			j++
		}
		sp := ui.Span{Text: string(runes[i:j])}
		if hit {
			sp.Weight, sp.Color = 700, color
		}
		out = append(out, sp)
		i = j
	}
	return out
}

// icon draws the icon of kind k in r, a square, in color.
func icon(p *ui.Painter, r ui.Rect, k Kind, color ui.Color) {
	u := r.W / 16
	pt := func(x, y float32) (float32, float32) { return r.X + x*u, r.Y + y*u }
	w := max(1, r.W/12)
	var path ui.Path
	poly := func(points ...float32) {
		path.MoveTo(pt(points[0], points[1]))
		for i := 2; i < len(points); i += 2 {
			path.LineTo(pt(points[i], points[i+1]))
		}
	}
	switch k {
	case File:
		poly(3.5, 1.5, 9.5, 1.5, 13, 5, 13, 14.5, 3.5, 14.5)
		path.Close()
		poly(9.5, 1.5, 9.5, 5, 13, 5)
	case Folder:
		poly(1.5, 3.5, 6, 3.5, 7.5, 5.5, 14.5, 5.5, 14.5, 13, 1.5, 13)
		path.Close()
	case Symbol:
		poly(6, 2, 4.5, 2.5, 4.5, 7, 3, 8, 4.5, 9, 4.5, 13.5, 6, 14)
		poly(10, 2, 11.5, 2.5, 11.5, 7, 13, 8, 11.5, 9, 11.5, 13.5, 10, 14)
	case Person:
		cx, cy := pt(8, 5.5)
		path.Circle(cx, cy, 3*u)
		path.MoveTo(pt(2.5, 14.5))
		x1, y1 := pt(2.5, 10)
		x2, y2 := pt(13.5, 10)
		x3, y3 := pt(13.5, 14.5)
		path.CubeTo(x1, y1, x2, y2, x3, y3)
	case Issue:
		cx, cy := pt(8, 8)
		path.Circle(cx, cy, 6.25*u)
		var dot ui.Path
		dot.Circle(cx, cy, 1.75*u)
		p.FillPath(&dot, color)
	}
	p.StrokePath(&path, w, color)
}
