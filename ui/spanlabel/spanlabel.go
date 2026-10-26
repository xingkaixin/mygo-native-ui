// Package spanlabel labels spans of a text, as annotation tools such as
// Prodigy and Label Studio label the names of people, places and things
// for training a model: the labels in a row above the text, each with
// its number key, and each span in the text
// tinted in its label's color, the label's name after it. Spans may
// overlap; each one that does is underlined lower than the one it
// overlaps.
//
// Dragging over words labels them with the label chosen above, from the
// start of the first word to the end of the last; within Chinese and
// Japanese text, from the character the drag starts on to the one it
// ends on. A click on a span selects it: a number key gives it another
// label, and Delete or Backspace removes it. A number key with none
// selected chooses that label; Escape lets the span go.
//
//	if spanlabel.View(c, &app.labeler, doc.Text, app.labels, &doc.Spans).Changed() {
//		app.save(doc)
//	}
package spanlabel

import (
	"cmp"
	"slices"
	"strconv"
	"unicode"
	"unicode/utf16"

	"github.com/egoist/mygo/ui"
)

// Label is a kind of span: its name and color, one of a set chosen by
// its place among the labels when zero.
type Label struct {
	Name  string
	Color ui.Color
}

// Span is runes Start up to End of a text, labeled with a label's Name.
type Span struct {
	Start, End int
	Label      string
}

// State is what a view keeps: the label chosen, the span selected and the
// drag, and the text laid out. Keep it in the app's state, one for each
// view.
type State struct {
	label    int
	selected int // the index of the span selected, plus 1, so that 0 is none
	pressing bool
	anchor   int
	caret    int

	text   string
	spans  []Span
	labels []Label
	runes  []rune
	font   ui.Font
	width  float32
	shape  *ui.TextLayout
	// before and after are the layout runes of each text rune and of its
	// end, before and after the tags of the spans ending there; of the
	// layout's runes, at is the text rune each is, and units where each
	// starts in UTF-16.
	parts         []ui.Span
	line          float32
	tagUnits      map[int]int // the span of each tag, by its UTF-16 offset
	before, after []int
	at, units     []int
	rects         [][]ui.Rect
	tags          []ui.Rect
	levels        []int
	names         map[string]*ui.TextLayout
}

// palette is the hues of labels without a color, in the order of the
// labels.
var palette = []float32{250, 145, 35, 300, 195, 85, 0, 330, 120}

// colorOf returns the color of label i.
func colorOf(labels []Label, i int, dark bool) ui.Color {
	if i < 0 {
		return ui.RGB(128, 128, 128)
	}
	if c := labels[i].Color; c != (ui.Color{}) {
		return c
	}
	l := float32(0.6)
	if dark {
		l = 0.72
	}
	return ui.Oklch(l, 0.15, palette[i%len(palette)])
}

// View creates the labeler of text, its spans in *spans, labeled with
// labels. Changed reports a span added, labeled anew or removed.
func View(c *ui.Context, s *State, text string, labels []Label, spans *[]Span) ui.Element {
	t := c.Theme()
	s.label = min(s.label, max(len(labels)-1, 0))
	if s.selected > len(*spans) {
		s.selected = 0
	}
	changed := false
	el := ui.Column(c).Gap(t.Space(3)).Focusable().FocusRing(false).Label("Spans")
	el.Children(func() {
		ui.Row(c).Gap(t.Space(1.5)).Wrap().Children(func() {
			for i, l := range labels {
				if chip(c.Key(i), l.Name, i, colorOf(labels, i, t.Dark), i == s.label).Clicked() {
					s.label = i
					if s.selected > 0 {
						(*spans)[s.selected-1].Label = l.Name
						changed = true
					}
				}
			}
		})
		changed = s.body(c, el, text, labels, spans) || changed
	})
	for i := range min(len(labels), 9) {
		if el.Shortcut(0, ui.Key1+ui.Key(i)) {
			s.label = i
			if s.selected > 0 {
				(*spans)[s.selected-1].Label = labels[i].Name
				changed = true
			}
		}
	}
	if s.selected > 0 && (el.Shortcut(0, ui.KeyBackspace) || el.Shortcut(0, ui.KeyDelete)) {
		*spans = slices.Delete(*spans, s.selected-1, s.selected)
		s.selected = 0
		changed = true
	}
	if s.selected > 0 && el.Shortcut(0, ui.KeyEscape) {
		s.selected = 0
	}
	if changed {
		el.SetChanged()
		c.Invalidate()
	}
	return el
}

// chip creates the chip of label name, the i-th, in color, lit while
// chosen.
func chip(c *ui.Context, name string, i int, color ui.Color, chosen bool) ui.Element {
	t := c.Theme()
	b := ui.ButtonBase(c).Gap(t.Space(1.5)).Padding(t.Space(1), t.Space(2.5)).Radius(t.Space(4)).
		AlignItems(ui.Center).Border(1, t.Border).Label(name).KeepFocus()
	switch {
	case chosen:
		b.Background(color.Alpha(0.16)).BorderColor(color)
	case b.Hovered():
		b.Background(t.SurfaceHover)
	}
	return b.Children(func() {
		ui.Box(c).Size(t.Space(2), t.Space(2)).Radius(t.Space(1)).Background(color)
		ui.Text(c, name).FontWeight(500)
		if i < 9 {
			ui.Text(c, strconv.Itoa(i+1)).FontSize(t.FontSize-3).TextColor(t.TextMuted).FontFeatures("tnum").
				Padding(0, t.Space(1)).Radius(t.Space(0.75)).Border(1, t.Border)
		}
	})
}

// body creates the text with its spans, under el, and reports a span
// added.
func (s *State) body(c *ui.Context, el ui.Element, text string, labels []Label, spans *[]Span) bool {
	t := c.Theme()
	font := ui.Font{Family: t.Font, Size: t.FontSize + 1}
	if text != s.text || font != s.font || !slices.Equal(*spans, s.spans) || !slices.Equal(labels, s.labels) {
		s.prepare(text, *spans, labels, font)
	}
	box := ui.Box(c).AlignSelf(ui.Stretch).Cursor(ui.CursorText).Role(ui.RoleImage).Label(text)
	if w := box.LayoutBounds().W; w > 0 && w != s.width {
		s.layout(w)
	}
	if s.shape != nil {
		_, h := s.shape.Size()
		box.Height(h)
	}
	added := false
	x, y, in := box.PointerPosition()
	pt := ui.Point{X: x, Y: y}
	switch {
	case box.Pressed() && s.shape != nil:
		el.Focus()
		if !s.pressing {
			s.pressing, s.anchor = true, s.runeAt(pt)
		}
		s.caret = s.runeAt(pt)
	case s.pressing:
		s.pressing = false
		if s.anchor == s.caret {
			if in {
				s.selected = s.spanAt(pt) + 1
			}
			break
		}
		start, end := s.snap(min(s.anchor, s.caret), max(s.anchor, s.caret))
		if start < end && len(labels) > 0 {
			sp := Span{Start: start, End: end, Label: labels[s.label].Name}
			if i := slices.Index(*spans, sp); i >= 0 {
				s.selected = i + 1
				break
			}
			*spans = append(*spans, sp)
			s.selected = len(*spans)
			added = true
		}
	}
	dark := t.Dark
	box.Draw(func(p *ui.Painter, r ui.Rect) {
		if s.shape == nil {
			return
		}
		for i, sp := range s.spans {
			color := colorOf(s.labels, s.labelIndex(sp.Label), dark)
			for _, b := range s.rects[i] {
				b = move(b, r)
				b.Y, b.H = b.Y+b.H-s.line, s.line
				p.Fill(b, color.Alpha(0.16), 3)
				p.Fill(ui.Rect{X: b.X, Y: b.Y + b.H - 2 - float32(s.levels[i])*3, W: b.W, H: 2}, color, 1)
				if i == s.selected-1 {
					p.Stroke(b, color, 3, 1.5)
				}
			}
		}
		if s.pressing && s.anchor != s.caret {
			for _, b := range s.shape.SelectionRects(s.span(min(s.anchor, s.caret), max(s.anchor, s.caret))) {
				p.Fill(move(b, r), t.Selection, 0)
			}
		}
		p.TextLayout(s.shape, r.X, r.Y, t.Text)
		for i, sp := range s.spans {
			b := move(s.tags[i], r)
			k := s.labelIndex(sp.Label)
			color := colorOf(s.labels, k, dark)
			pad := font.Size * 0.15
			p.Fill(ui.Rect{X: b.X + pad, Y: b.Y, W: b.W - 2*pad, H: b.H}, color, b.H/2)
			name := s.names[sp.Label]
			w, h := name.Size()
			fg := ui.RGB(255, 255, 255)
			if dark {
				fg = ui.RGB(20, 20, 20)
			}
			p.TextLayout(name, b.X+(b.W-w)/2, b.Y+(b.H-h)/2, fg)
		}
	})
	return added
}

// move returns b, in the layout, where the layout is drawn in r.
func move(b, r ui.Rect) ui.Rect {
	b.X, b.Y = b.X+r.X, b.Y+r.Y
	return b
}

// labelIndex returns the index of the label named name, or -1.
func (s *State) labelIndex(name string) int {
	return slices.IndexFunc(s.labels, func(l Label) bool { return l.Name == name })
}

// prepare starts over for spans of text, labeled with labels, in font.
func (s *State) prepare(text string, spans []Span, labels []Label, font ui.Font) {
	s.text, s.spans, s.labels, s.font, s.width, s.shape = text, slices.Clone(spans), slices.Clone(labels), font, -1, nil
	s.runes = []rune(text)
	n := len(s.runes)
	for i := range s.spans {
		sp := &s.spans[i]
		sp.Start = min(max(sp.Start, 0), n)
		sp.End = min(max(sp.End, sp.Start), n)
	}
	s.levels = make([]int, len(s.spans))
	order := make([]int, len(s.spans))
	for i := range order {
		order[i] = i
	}
	slices.SortStableFunc(order, func(a, b int) int {
		return cmp.Or(cmp.Compare(s.spans[a].Start, s.spans[b].Start), cmp.Compare(s.spans[b].End, s.spans[a].End))
	})
	// A span goes on the highest level whose spans end before it starts.
	var ends []int
	for _, i := range order {
		k := slices.IndexFunc(ends, func(e int) bool { return e <= s.spans[i].Start })
		if k < 0 {
			k = len(ends)
			ends = append(ends, 0)
		}
		ends[k], s.levels[i] = s.spans[i].End, k
	}
	small := ui.Font{Family: font.Family, Size: font.Size - 4, Weight: 700}
	s.names = map[string]*ui.TextLayout{}
	for _, sp := range s.spans {
		if s.names[sp.Label] == nil {
			s.names[sp.Label] = ui.ShapeText(sp.Label, small, 0)
		}
	}
	// The tags of the spans ending at a rune go before it, in the order
	// of the spans.
	ending := make([][]int, n+1)
	for i, sp := range s.spans {
		ending[sp.End] = append(ending[sp.End], i)
	}
	m := font.Metrics()
	tagH := m.Ascent * 0.85
	s.line = m.Ascent + m.Descent
	// Layouts have no line height of their own: struts, objects taller
	// than the text and too thin to see, after each space and Chinese or
	// Japanese punctuation, open room between lines for the underlines.
	strut := ui.Span{Text: "￼", ObjectWidth: 0.01, ObjectHeight: m.Ascent + font.Size*0.55}
	s.tagUnits = map[int]int{}
	var parts []ui.Span
	s.before, s.after = make([]int, n+1), make([]int, n+1)
	s.at, s.units = s.at[:0], append(s.units[:0], 0)
	add := func(r, units int) {
		s.at = append(s.at, r)
		s.units = append(s.units, s.units[len(s.units)-1]+units)
	}
	from := 0
	for r := 0; r <= n; r++ {
		s.before[r] = len(s.at)
		struts := r == 0 || unicode.IsSpace(s.runes[r-1]) || s.runes[r-1] >= 0x3000 && unicode.IsPunct(s.runes[r-1])
		if (len(ending[r]) > 0 || struts) && from < r {
			parts = append(parts, ui.Span{Text: string(s.runes[from:r])})
			from = r
		}
		for _, i := range ending[r] {
			w, _ := s.names[s.spans[i].Label].Size()
			parts = append(parts, ui.Span{Text: "￼", ObjectWidth: w + font.Size, ObjectHeight: tagH})
			s.tagUnits[s.units[len(s.units)-1]] = i
			add(r, 1)
		}
		if struts {
			parts = append(parts, strut)
			add(r, 1)
		}
		s.after[r] = len(s.at)
		if r < n {
			add(r, utf16.RuneLen(s.runes[r]))
		}
	}
	if from < n {
		parts = append(parts, ui.Span{Text: string(s.runes[from:])})
	}
	s.at = append(s.at, n)
	s.parts = parts
}

// layout lays the text out width wide, and finds the boxes of the spans
// and of their tags.
func (s *State) layout(width float32) {
	s.width = width
	s.shape = ui.ShapeRichText(s.parts, s.font, width)
	s.rects = make([][]ui.Rect, len(s.spans))
	for i, sp := range s.spans {
		s.rects[i] = s.shape.SelectionRects(s.span(sp.Start, sp.End))
	}
	s.tags = make([]ui.Rect, len(s.spans))
	for _, o := range s.shape.Objects() {
		if i, ok := s.tagUnits[o.Index]; ok {
			s.tags[i] = o.Rect
		}
	}
}

// span returns runes start up to end of the text as a range of the
// layout, without the tags at its ends.
func (s *State) span(start, end int) ui.TextInputRange {
	return ui.TextInputRange{Start: s.units[s.after[start]], End: s.units[s.before[end]]}
}

// runeAt returns the place between runes of the text nearest pt.
func (s *State) runeAt(pt ui.Point) int {
	l, _ := slices.BinarySearch(s.units, s.shape.IndexAt(pt))
	return s.at[min(l, len(s.at)-1)]
}

// spanAt returns the index of the span at pt, the shortest of those
// there, or -1.
func (s *State) spanAt(pt ui.Point) int {
	hit := -1
	for i, sp := range s.spans {
		inside := slices.ContainsFunc(s.rects[i], func(b ui.Rect) bool { return b.Contains(pt.X, pt.Y) }) || s.tags[i].Contains(pt.X, pt.Y)
		if inside && (hit < 0 || sp.End-sp.Start < s.spans[hit].End-s.spans[hit].Start) {
			hit = i
		}
	}
	return hit
}

// snap returns start and end widened to whole words, and narrowed to
// leave out spaces and punctuation at either end.
func (s *State) snap(start, end int) (int, int) {
	r := s.runes
	for start > 0 && word(r[start-1]) && word(r[start]) {
		start--
	}
	for end < len(r) && end > 0 && word(r[end]) && word(r[end-1]) {
		end++
	}
	for start < end && !unicode.IsLetter(r[start]) && !unicode.IsDigit(r[start]) {
		start++
	}
	for end > start && !unicode.IsLetter(r[end-1]) && !unicode.IsDigit(r[end-1]) {
		end--
	}
	return start, end
}

// word reports whether r is part of a word that spaces set apart, which
// those of Chinese and Japanese are not.
func word(r rune) bool {
	if unicode.In(r, unicode.Han, unicode.Hiragana, unicode.Katakana) {
		return false
	}
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '\'' || r == '’' || r == '-' || r == '_'
}
