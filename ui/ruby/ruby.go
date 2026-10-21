// Package ruby sets ruby, small readings above base text: pinyin over
// Chinese characters, zhuyin, or furigana over Japanese kanji. A reading
// shows centered over its base at half its size. One longer than its base
// reaches a little over the text beside it, as far as nothing above that
// is in its way, and spreads the base's characters apart for the rest; a
// shorter one of kana or zhuyin spreads its characters over the base, and
// pinyin sits centered. A base and its reading stay together on one line,
// and the text around them wraps as mygo's text does: Chinese and
// Japanese between any two characters, but not before closing punctuation
// or after opening punctuation.
//
// With Hidden set, the readings go and the lines close up; a base with a
// reading lights under the pointer, and its reading shows in a tip once
// the pointer rests on it.
//
// Dragging selects text, a base and its reading whole. Cmd+C copies the
// bases selected, and the context menu's Copy with Readings copies each
// followed by its reading in parentheses, 漢(hàn)字(zì).
//
//	runs := ruby.Parse("{東京|とうきょう}で{漢字|かん|じ}を{学|まな}ぶ")
//	ruby.View(c, &app.ruby, runs, 20)
//
// Text with kana in it is set as Japanese, with zhuyin as traditional
// Chinese, and with other Chinese characters as simplified Chinese, for
// the forms of its characters.
package ruby

import (
	"slices"
	"unicode"
	"unicode/utf8"

	"github.com/egoist/mygo/transfer"
	"github.com/egoist/mygo/ui"
)

// Run is a stretch of text: Base, and Reading over it, or "" for plain
// text.
type Run struct {
	Base, Reading string
}

// State is what a view keeps: its bases and readings shaped, and where
// they fell in the lines. Hidden, which the app sets, hides the readings.
// Keep it in the app's state, one for each view; the zero State shows
// the readings.
type State struct {
	Hidden bool

	// key and runs are what spans and pairs were shaped for, and lang is
	// the language the runs read in.
	key   key
	runs  []Run
	lang  string
	spans []ui.Span
	pairs []pair
	// at and units are where each run starts in the paragraph's text, in
	// runes and in UTF-16 units, followed by where it ends.
	at, units []int
	// lift is how far a reading's baseline is above its base's, and
	// ascent and descent are the base's font's.
	lift, ascent, descent float32
	// layout is the paragraph laid out at width, as it shows.
	layout *ui.TextLayout
	width  float32
	// plain is the bases, as assistive technology reads them.
	plain string
	// hover is the run under the pointer while the readings hide, or -1.
	hover int
}

// key is what shaping depends on besides the runs.
type key struct {
	size   float32
	family string
	hidden bool
}

// pair is a run with a reading, shaped: its base and its reading placed
// from the left of the run's box, w wide, which stands on the baseline at
// x, y in the paragraph; dx moves the reading within the paragraph. bw and
// rw are how wide the base and the reading are as shaped, the reading with
// room on either side when it is in letters, and n is how many characters
// the reading spreads, or 1 for one that sits centered.
type pair struct {
	base, reading []ui.Glyph
	w, x, y, dx   float32
	bw, rw        float32
	n             int
}

// What sizes a reading and the room around it, as shares of the base's
// size: the reading's size, the room between it and its base, the room
// above it, the room between readings in letters, and how far a reading
// may reach over plain text on each side.
const (
	scale = 0.5
	gap   = 0.06
	above = 0.1
	apart = 0.15
	reach = 0.25
)

// View creates the text of runs, the bases size DIPs high, wrapping at the
// width it gets, as a Text does.
func View(c *ui.Context, s *State, runs []Run, size float32) ui.Element {
	t := c.Theme()
	k := key{size: size, family: t.Font, hidden: s.Hidden}
	if k != s.key || s.spans == nil || !slices.Equal(runs, s.runs) {
		s.shape(runs, k)
	}
	font := ui.Font{Family: k.family, Size: size, Lang: s.lang}

	// The selection is the container's, whose copies CopyData makes: a
	// paragraph alone copies its own text, an object for each base.
	view := ui.Column(c).Selectable()
	view.CopyData(func(parts []ui.SelectedText) transfer.Data { return transfer.TextData(s.copy(parts, false)) })
	view.ContextMenu(func(m *ui.Menu) {
		m.EditItems()
		parts := view.SelectedTexts()
		if m.Item("Copy with Readings").Disabled(len(parts) == 0).Chosen() {
			c.WriteClipboard(s.copy(parts, true))
		}
	})
	s.hover = -1
	view.Children(func() {
		// The paragraph's style is set whole, whatever it would inherit, so
		// that it lays out as the layout drawing its bases and readings
		// does.
		para := ui.RichText(c).Font(k.family).FontSize(size).FontWeight(400).LineHeight(0).LetterSpacing(0).FontFeatures().
			TextAlign(ui.Start).TextColor(t.Text).Lang(s.lang).Label(s.plain)
		para.Children(func() {
			for i, r := range runs {
				if r.Reading == "" {
					ui.Text(c, r.Base)
					continue
				}
				base := ui.RichText(c, s.spans[i])
				if s.Hidden {
					base.Tooltip(r.Reading)
					if base.Hovered() {
						s.hover = i
					}
				}
			}
		})
		para.DrawOver(func(p *ui.Painter, r ui.Rect) {
			s.place(font, r.W)
			for i, run := range s.runs {
				if run.Reading == "" {
					continue
				}
				q := &s.pairs[i]
				x, y := r.X+q.x, r.Y+q.y
				if i == s.hover {
					pad := size * 0.08
					p.Fill(ui.Rect{X: x - pad, Y: y - s.ascent, W: q.w + 2*pad, H: s.ascent + s.descent}, t.Text.Alpha(0.08), t.Radius/2)
				}
				p.Glyphs(q.base, x, y, t.Text)
				p.Glyphs(q.reading, x+q.dx, y-s.lift, t.TextMuted)
			}
		})
	})
	return view
}

// shape shapes runs as k says: a span for each, the runs with a reading an
// inline object as wide as the wider of base and reading, less what the
// reading reaches over the text on either side, with no readings while
// hidden.
func (s *State) shape(runs []Run, k key) {
	s.lang = language(runs)
	base := ui.Font{Family: k.family, Size: k.size, Lang: s.lang}
	read := ui.Font{Family: k.family, Size: k.size * scale, Lang: s.lang}
	m, rm := base.Metrics(), read.Metrics()
	s.ascent, s.descent = m.Ascent, m.Descent
	s.lift = m.Ascent + gap*k.size + rm.Descent
	high := float32(0)
	if !k.hidden {
		high = s.lift + rm.Ascent + above*k.size
	}
	s.key, s.runs = k, append(s.runs[:0], runs...)
	s.pairs, s.layout = s.pairs[:0], nil
	for _, r := range runs {
		var q pair
		if r.Reading != "" {
			q.base = ui.Shape(r.Base, base)
			q.bw, q.n = width(q.base), 1
		}
		if r.Reading != "" && !k.hidden {
			q.reading = ui.Shape(r.Reading, read)
			q.rw = width(q.reading)
			if wide(r.Reading) {
				q.n = clusters(q.reading)
			} else {
				q.rw += apart * k.size
			}
		}
		s.pairs = append(s.pairs, q)
	}

	s.spans = s.spans[:0]
	s.at, s.units = append(s.at[:0], 0), append(s.units[:0], 0)
	for i, r := range runs {
		span := ui.Span{Text: r.Base}
		n, u := utf8.RuneCountInString(r.Base), ui.UTF16Len(r.Base)
		if r.Reading != "" {
			q := &s.pairs[i]
			q.w = q.bw
			if over := q.rw - q.bw; over > 0 {
				left, right := min(over/2, s.room(i-1, true)), min(over/2, s.room(i+1, false))
				q.w = q.rw - left - right
				spread(q.base, q.bw, q.w)
				move(q.reading, (q.rw-width(q.reading))/2-left)
			} else if q.n > 1 {
				spread(q.reading, q.rw, q.w)
			} else {
				move(q.reading, (q.w-width(q.reading))/2)
			}
			span = ui.Span{Text: "\uFFFC", ObjectWidth: q.w, ObjectHeight: high}
			n, u = 1, 1
		}
		s.spans = append(s.spans, span)
		s.at = append(s.at, s.at[i]+n)
		s.units = append(s.units, s.units[i]+u)
	}
	s.plain = s.text(0, s.at[len(runs)], false)
}

// room returns how far a reading may reach over run i beside it, which is
// before it when before: up to a quarter of the base's size over plain
// text but a space, and over the room a narrower reading leaves by the
// edge of its base.
func (s *State) room(i int, before bool) float32 {
	if i < 0 || i >= len(s.runs) {
		return 0
	}
	if r := s.runs[i]; r.Reading == "" {
		c, _ := utf8.DecodeRuneInString(r.Base)
		if before {
			c, _ = utf8.DecodeLastRuneInString(r.Base)
		}
		if r.Base == "" || unicode.IsSpace(c) {
			return 0
		}
		return reach * s.key.size
	}
	q := &s.pairs[i]
	return max(0, (q.bw-q.rw)/float32(2*q.n))
}

// place lays the paragraph out at width w in font f, as it shows, and
// finds where each run with a reading fell; a reading reaching past
// either side of the paragraph moves in.
func (s *State) place(f ui.Font, w float32) {
	if s.layout != nil && s.width == w {
		return
	}
	s.layout, s.width = ui.ShapeRichText(s.spans, f, w), w
	for _, o := range s.layout.Objects() {
		// The run whose object it is is the last starting at it: runs
		// with no text before it start there too.
		i, _ := slices.BinarySearch(s.units, o.Index+1)
		q := &s.pairs[i-1]
		q.x, q.y, q.dx = o.Rect.X, o.Rect.Y+o.Rect.H, 0
		if len(q.reading) == 0 {
			continue
		}
		left, right := q.x+q.reading[0].X, q.x+q.reading[len(q.reading)-1].X+q.reading[len(q.reading)-1].Advance
		switch {
		case left < 0:
			q.dx = -left
		case w > 0 && right > w:
			q.dx = w - right
		}
	}
}

// copy returns the text of parts of the paragraph, as Copy puts it on the
// clipboard: the bases, followed with readings by their readings.
func (s *State) copy(parts []ui.SelectedText, readings bool) string {
	text := ""
	for i, p := range parts {
		if i > 0 {
			text += "\n"
		}
		text += s.text(p.Start, p.End, readings)
	}
	return text
}

// width returns how wide glyphs are together.
func width(glyphs []ui.Glyph) float32 {
	w := float32(0)
	for _, g := range glyphs {
		w = max(w, g.X+g.Advance)
	}
	return w
}

// clusters returns how many characters glyphs show.
func clusters(glyphs []ui.Glyph) int {
	n := 0
	for i := range glyphs {
		if i == 0 || glyphs[i].Cluster != glyphs[i-1].Cluster {
			n++
		}
	}
	return n
}

// move moves glyphs dx along.
func move(glyphs []ui.Glyph, dx float32) {
	for i := range glyphs {
		glyphs[i].X += dx
	}
}

// spread moves glyphs w0 wide together apart across w, the room left
// around each character, half on either side, as CSS's ruby-align:
// space-around spreads them; one character sits centered.
func spread(glyphs []ui.Glyph, w0, w float32) {
	n := clusters(glyphs)
	if n == 0 {
		return
	}
	room := (w - w0) / float32(n)
	k := -1
	for i := range glyphs {
		if i == 0 || glyphs[i].Cluster != glyphs[i-1].Cluster {
			k++
		}
		glyphs[i].X += room/2 + float32(k)*room
	}
}
