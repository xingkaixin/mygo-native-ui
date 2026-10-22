// Package ragchunks shows how a document was split for retrieval, as RAG
// debugging tools show it: the document with each chunk's range tinted,
// in two colors by turns, the text two neighbors share hatched over both
// tints, and each chunk's number at its start; beside it, the chunks in a
// list, each with its number, its size in characters and tokens, and the
// start of its text. Given what a query retrieved, the list puts the
// chunks retrieved first, highest score first, each with a bar of its
// score, and the document outlines them and dims the rest.
//
// The pointer on a chunk, in the document or in the list, lights it in
// both and scrolls the other to it. A click selects a chunk, which stays
// lit; a click on it again, or beside the text, lets it go.
//
// Split splits a text into chunks of a size, overlapping, as RAG
// libraries do; an app that splits its own passes their ranges instead:
//
//	if app.chunks == nil {
//		app.chunks = ragchunks.Split(app.doc, 500, 100, "\n\n", ". ", " ")
//	}
//	ragchunks.View(c, &app.inspector, app.doc, app.chunks, app.hits).Grow(1)
package ragchunks

import (
	"cmp"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf16"

	"github.com/egoist/mygo/ui"
)

// Chunk is runes Start up to End of a document's text.
type Chunk struct {
	Start, End int
}

// Hit is a chunk a query retrieved, by its index among the chunks, and
// its score, higher for a better match, in any unit: the bars measure the
// scores against the highest.
type Hit struct {
	Chunk int
	Score float64
}

// Split splits text into chunks of at most size characters, each sharing
// up to overlap characters with the one before. Without separators, the
// chunks are windows of size characters, each starting size-overlap after
// the one before. With them, as LangChain's RecursiveCharacterTextSplitter
// does, the text is cut after each of the first separator it holds, the
// pieces in a row that fit in size are joined into chunks as long as size
// lets, the last pieces of each, as many as fit in overlap, starting the
// next, and a piece longer than size is split so by the separators after,
// or into windows when none is left; the chunks leave out the spaces at
// their ends. Pass "\n\n", "\n", ". ", " " for paragraphs, then lines,
// sentences and words.
func Split(text string, size, overlap int, separators ...string) []Chunk {
	size = max(size, 1)
	overlap = min(max(overlap, 0), size-1)
	var seps [][]rune
	for _, sep := range separators {
		if sep != "" {
			seps = append(seps, []rune(sep))
		}
	}
	runes := []rune(text)
	return split(runes, 0, len(runes), size, overlap, seps, nil)
}

// split appends to out the chunks of runes lo up to hi of runes, split as
// Split says by seps.
func split(runes []rune, lo, hi, size, overlap int, seps [][]rune, out []Chunk) []Chunk {
	for k, sep := range seps {
		var cuts []int
		for i := lo; i+len(sep) <= hi; {
			if slices.Equal(runes[i:i+len(sep)], sep) {
				i += len(sep)
				cuts = append(cuts, i)
			} else {
				i++
			}
		}
		if len(cuts) == 0 {
			continue
		}
		if cuts[len(cuts)-1] != hi {
			cuts = append(cuts, hi)
		}
		var fit []Chunk
		from := lo
		for _, cut := range cuts {
			pc := Chunk{from, cut}
			from = cut
			if pc.End-pc.Start <= size {
				fit = append(fit, pc)
				continue
			}
			out = merge(runes, fit, size, overlap, out)
			fit = nil
			out = split(runes, pc.Start, pc.End, size, overlap, seps[k+1:], out)
		}
		return merge(runes, fit, size, overlap, out)
	}
	for start := lo; start < hi; start += size - overlap {
		end := min(start+size, hi)
		out = append(out, Chunk{start, end})
		if end == hi {
			break
		}
	}
	return out
}

// merge appends to out pieces, in order, joined into chunks of at most
// size runes, each starting with the last pieces of the one before that
// fit in overlap.
func merge(runes []rune, pieces []Chunk, size, overlap int, out []Chunk) []Chunk {
	var cur []Chunk
	total := 0
	emit := func() {
		start, end := cur[0].Start, cur[len(cur)-1].End
		for start < end && unicode.IsSpace(runes[start]) {
			start++
		}
		for end > start && unicode.IsSpace(runes[end-1]) {
			end--
		}
		if start < end {
			out = append(out, Chunk{start, end})
		}
	}
	for _, pc := range pieces {
		n := pc.End - pc.Start
		if total+n > size && len(cur) > 0 {
			emit()
			for len(cur) > 0 && (total > overlap || total+n > size) {
				total -= cur[0].End - cur[0].Start
				cur = cur[1:]
			}
		}
		cur = append(cur, pc)
		total += n
	}
	if len(cur) > 0 {
		emit()
	}
	return out
}

// State is what a view keeps: the document laid out with its chunks, the
// chunk under the pointer and the one selected. Tokens, which the app
// sets, counts the tokens of a chunk's text, as the embedding model's
// tokenizer does, once for each chunk as the chunks change; nil counts one
// for each 4 characters. Keep it in the app's state, one for each view.
type State struct {
	Tokens func(text string) int

	text   string
	chunks []Chunk
	font   ui.Font
	// bounds are the chunks kept within the text, order their indices by
	// where they start, tone 0 or 1 by turns in that order, chars and
	// tokens their sizes, previews their text on one line, and numbers
	// their numbers shaped.
	bounds        []Chunk
	order, tone   []int
	chars, tokens []int
	previews      []string
	numbers       []*ui.TextLayout
	// spans are the text with a box for a chunk's number before each
	// chunk's start; at is where each rune of the text, and its end, is
	// among them, before the boxes there, runes which rune of the text
	// each of theirs is, a box the rune after it, and units the UTF-16
	// offset of each of theirs, which a TextLayout counts in.
	spans []ui.Span
	at    []int
	runes []int
	units []int
	// width is the width the text was laid out in, shape the layout and
	// height its height; rects are each chunk's boxes on the lines it
	// takes, shared those of the text neighbors share, and badges the
	// boxes of the chunks' numbers.
	width  float32
	shape  *ui.TextLayout
	height float32
	rects  [][]ui.Rect
	shared []ui.Rect
	badges []ui.Rect
	// hot and selected are the index of the chunk lit by the pointer and
	// of the one selected, plus 1, so that 0 is none; revealDoc and
	// revealList scroll the hot chunk into view in the document or the
	// list in the next frame.
	hot, selected         int
	revealDoc, revealList bool
}

// prepare starts over for chunks of text drawn in font, none of them lit.
func (s *State) prepare(text string, chunks []Chunk, font ui.Font) {
	runes := []rune(text)
	n := len(runes)
	s.text, s.chunks, s.font, s.width, s.shape = text, slices.Clone(chunks), font, -1, nil
	s.hot, s.selected = 0, 0
	s.bounds = make([]Chunk, len(chunks))
	s.order = make([]int, len(chunks))
	s.tone = make([]int, len(chunks))
	s.chars = make([]int, len(chunks))
	s.tokens = make([]int, len(chunks))
	s.previews = make([]string, len(chunks))
	s.numbers = make([]*ui.TextLayout, len(chunks))
	small := ui.Font{Family: font.Family, Size: font.Size - 3, Weight: 600, Features: "tnum"}
	pad := font.Size * 0.4
	for i, ch := range chunks {
		start := min(max(ch.Start, 0), n)
		ch = Chunk{start, min(max(ch.End, start), n)}
		s.bounds[i], s.order[i] = ch, i
		part := string(runes[ch.Start:ch.End])
		s.chars[i] = ch.End - ch.Start
		if s.Tokens != nil {
			s.tokens[i] = s.Tokens(part)
		} else {
			s.tokens[i] = (s.chars[i] + 3) / 4
		}
		s.previews[i] = preview(part)
		s.numbers[i] = ui.ShapeText(strconv.Itoa(i+1), small, 0)
	}
	slices.SortStableFunc(s.order, func(a, b int) int { return cmp.Compare(s.bounds[a].Start, s.bounds[b].Start) })
	for k, i := range s.order {
		s.tone[i] = k % 2
	}

	ascent := font.Metrics().Ascent
	s.spans, s.at, s.runes, s.units = s.spans[:0], make([]int, n+1), s.runes[:0], append(s.units[:0], 0)
	add := func(r int, units int) {
		s.runes = append(s.runes, r)
		s.units = append(s.units, s.units[len(s.units)-1]+units)
	}
	from, k := 0, 0
	for r := 0; r <= n; r++ {
		s.at[r] = len(s.runes)
		for ; k < len(s.order) && s.bounds[s.order[k]].Start == r; k++ {
			if from < r {
				s.spans = append(s.spans, ui.Span{Text: string(runes[from:r])})
				from = r
			}
			w, _ := s.numbers[s.order[k]].Size()
			s.spans = append(s.spans, ui.Span{Text: "￼", ObjectWidth: w + 4*pad, ObjectHeight: ascent * 0.7})
			add(r, 1)
		}
		if r < n {
			add(r, utf16.RuneLen(runes[r]))
		}
	}
	if from < n {
		s.spans = append(s.spans, ui.Span{Text: string(runes[from:])})
	}
	s.runes = append(s.runes, n)
}

// preview returns text on one line, its spaces and line breaks one space,
// cut short where two lines of the list could not show more.
func preview(text string) string {
	words := strings.Fields(text)
	var b strings.Builder
	for _, w := range words {
		if b.Len() > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(w)
		if b.Len() > 240 {
			break
		}
	}
	return b.String()
}

// span returns runes start up to end of the text as a range of the
// layout, the boxes of the numbers of chunks starting at end left out.
func (s *State) span(start, end int) ui.TextInputRange {
	return ui.TextInputRange{Start: s.units[s.at[start]], End: s.units[s.at[end]]}
}

// layout lays the text out width wide, and finds the boxes of the chunks,
// of the text they share, and of their numbers.
func (s *State) layout(width float32) {
	s.width = width
	s.shape = ui.ShapeRichText(s.spans, s.font, width)
	_, s.height = s.shape.Size()
	s.rects = make([][]ui.Rect, len(s.bounds))
	for i, ch := range s.bounds {
		s.rects[i] = s.shape.SelectionRects(s.span(ch.Start, ch.End))
	}
	s.shared = s.shared[:0]
	for k := 1; k < len(s.order); k++ {
		a, b := s.bounds[s.order[k-1]], s.bounds[s.order[k]]
		if end := min(a.End, b.End); b.Start < end {
			s.shared = append(s.shared, s.shape.SelectionRects(s.span(b.Start, end))...)
		}
	}
	s.badges = make([]ui.Rect, len(s.bounds))
	for k, obj := range s.shape.Objects() {
		if k < len(s.order) {
			s.badges[s.order[k]] = obj.Rect
		}
	}
}

// chunkAt returns the chunk at pt of the document, or -1: of two that
// share the text there, the one whose middle is nearer.
func (s *State) chunkAt(pt ui.Point) int {
	l, _ := slices.BinarySearch(s.units, s.shape.IndexAt(pt))
	r := s.runes[min(l, len(s.runes)-1)]
	hit, best := -1, 0
	for i, ch := range s.bounds {
		if !slices.ContainsFunc(s.rects[i], func(b ui.Rect) bool {
			return pt.X >= b.X && pt.X < b.X+b.W && pt.Y >= b.Y && pt.Y < b.Y+b.H
		}) {
			continue
		}
		if d := abs(2*r - ch.Start - ch.End); hit < 0 || d < best {
			hit, best = i, d
		}
	}
	return hit
}

func abs(v int) int { return max(v, -v) }

// toggle selects chunk i, or lets it go when it is selected or -1.
func (s *State) toggle(i int) {
	if s.selected == i+1 {
		s.selected = 0
	} else {
		s.selected = i + 1
	}
}

// View creates the document, text, with its chunks, and beside it the
// list of them, in a row, each scrolling on its own: give it a size, or
// Grow. hits are what a query retrieved, in any order; nil, as before a
// query, shows every chunk alike, and an empty hits, as a query that
// matched nothing, dims them all.
func View(c *ui.Context, s *State, text string, chunks []Chunk, hits []Hit) ui.Element {
	t := c.Theme()
	font := ui.Font{Family: t.Font, Size: t.FontSize + 1}
	if s.text != text || s.font != font || !slices.Equal(s.chunks, chunks) {
		s.prepare(text, chunks, font)
	}
	hits = ranked(hits, len(chunks))
	retrieved := make([]bool, len(chunks))
	for _, h := range hits {
		retrieved[h.Chunk] = true
	}
	return ui.Row(c).AlignItems(ui.Stretch).Children(func() {
		docHot, over := s.document(c, hits != nil, retrieved)
		ui.Divider(c).Vertical()
		listHot := s.list(c, hits, retrieved)
		hot := listHot
		if over {
			hot = docHot
		}
		if hot != s.hot-1 {
			s.hot = hot + 1
			s.revealList, s.revealDoc = hot >= 0 && over, hot >= 0 && !over
			c.Invalidate()
		}
	})
}

// ranked returns the hits of chunks among n, highest score first, nil
// when hits is.
func ranked(hits []Hit, n int) []Hit {
	if hits == nil {
		return nil
	}
	out := make([]Hit, 0, len(hits))
	for _, h := range hits {
		if h.Chunk >= 0 && h.Chunk < n && !slices.ContainsFunc(out, func(o Hit) bool { return o.Chunk == h.Chunk }) {
			out = append(out, h)
		}
	}
	slices.SortStableFunc(out, func(a, b Hit) int { return cmp.Compare(b.Score, a.Score) })
	return out
}

// document creates the text with its chunks, and returns the chunk under
// the pointer, or -1, and whether the pointer is over the text. query
// tells whether a query retrieved the chunks retrieved, which dims the
// rest.
func (s *State) document(c *ui.Context, query bool, retrieved []bool) (hot int, over bool) {
	t := c.Theme()
	hot = -1
	ui.Scroll(c).Grow(1).Basis(0).Background(t.Background).Children(func() {
		ui.Column(c).Padding(t.Space(5), t.Space(6), t.Space(10)).Children(func() {
			box := ui.Box(c).Label(s.text)
			if w := box.LayoutBounds().W; w != s.width {
				s.layout(w)
			}
			box.Height(s.height)
			if x, y, ok := box.PointerPosition(); ok && box.Hovered() {
				over, hot = true, s.chunkAt(ui.Point{X: x, Y: y})
			}
			if hot >= 0 {
				box.Cursor(ui.CursorPointer)
			}
			if box.Clicked() {
				s.toggle(hot)
			}
			box.Draw(func(p *ui.Painter, r ui.Rect) {
				// The text was laid out in the width of the last frame.
				if r.W != s.width {
					c.Invalidate()
				}
				s.paint(p, r, t, query, retrieved)
			})
			if s.revealDoc && s.hot > 0 {
				b := union(s.rects[s.hot-1])
				box.Children(func() {
					ui.Box(c).Absolute().PassThrough().Left(b.X).Top(b.Y).Size(b.W, b.H).ScrollIntoView()
				})
			}
			s.revealDoc = false
		})
	})
	return hot, over
}

// paint draws the document in r: the chunks' tints, the text they share
// hatched, the text, the chunks' numbers, and over them the dimming of
// those a query did not retrieve and the outlines of those it did, of the
// hot chunk and of the selected one.
func (s *State) paint(p *ui.Painter, r ui.Rect, t *ui.Theme, query bool, retrieved []bool) {
	at := func(b ui.Rect) ui.Rect { return ui.Rect{X: r.X + b.X, Y: r.Y + b.Y, W: b.W, H: b.H} }
	hot, sel := s.hot-1, s.selected-1
	for i := range s.bounds {
		color := tint(t, s.tone[i], i == hot || i == sel)
		for _, b := range s.rects[i] {
			p.Fill(at(b), color, 0)
		}
	}
	hatch := t.Text.Alpha(0.3)
	for _, b := range s.shared {
		hatched(p, at(b), hatch)
	}
	p.TextLayout(s.shape, r.X, r.Y, t.Text)
	ascent := s.font.Metrics().Ascent
	for i, b := range s.badges {
		w, h := s.numbers[i].Size()
		ph := h + 2
		pill := ui.Rect{X: r.X + b.X + 1, W: b.W - 2, Y: r.Y + b.Y + b.H - ascent*0.36 - ph/2, H: ph}
		bg, fg := badge(t, s.tone[i], retrieved[i])
		p.Fill(pill, bg, ph/2)
		p.TextLayout(s.numbers[i], pill.X+(pill.W-w)/2, pill.Y+(ph-h)/2, fg)
	}
	if query {
		veil := t.Background.Alpha(0.6)
		for _, b := range s.dimmed(retrieved, hot, sel) {
			p.Fill(at(b), veil, 0)
		}
	}
	for i, on := range retrieved {
		if on && i != hot && i != sel {
			outline(p, s.rects[i], r, t.Accent.Alpha(0.8), 1.5)
		}
	}
	if hot >= 0 && hot != sel {
		outline(p, s.rects[hot], r, t.Text.Alpha(0.6), 1.5)
	}
	if sel >= 0 {
		outline(p, s.rects[sel], r, t.Accent, 2)
	}
}

// dimmed returns the boxes of the text in no chunk retrieved, hot or
// selected.
func (s *State) dimmed(retrieved []bool, hot, sel int) []ui.Rect {
	var kept []Chunk
	for i, ch := range s.bounds {
		if retrieved[i] || i == hot || i == sel {
			kept = append(kept, ch)
		}
	}
	slices.SortFunc(kept, func(a, b Chunk) int { return cmp.Compare(a.Start, b.Start) })
	var ranges []ui.TextInputRange
	from := 0
	for _, ch := range kept {
		if from < ch.Start {
			ranges = append(ranges, s.span(from, ch.Start))
		}
		from = max(from, ch.End)
	}
	if n := len(s.at) - 1; from < n {
		ranges = append(ranges, s.span(from, n))
	}
	if len(ranges) == 0 {
		return nil
	}
	return s.shape.SelectionRects(ranges...)
}

// tint returns the color of a chunk of tone 0 or 1, deeper when lit.
func tint(t *ui.Theme, tone int, lit bool) ui.Color {
	alpha := float32(0.16)
	if t.Dark {
		alpha = 0.2
	}
	if lit {
		alpha *= 2
	}
	return hue(tone).Alpha(alpha)
}

// hue returns the color of chunks of tone 0, blue, or 1, amber.
func hue(tone int) ui.Color {
	if tone == 0 {
		return ui.Oklch(0.66, 0.14, 250)
	}
	return ui.Oklch(0.74, 0.15, 70)
}

// badge returns the colors of a chunk's number of tone 0 or 1: of the
// accent when a query retrieved the chunk.
func badge(t *ui.Theme, tone int, retrieved bool) (bg, fg ui.Color) {
	if retrieved {
		return t.Accent, ui.RGB(255, 255, 255)
	}
	alpha := float32(0.4)
	if t.Dark {
		alpha = 0.5
	}
	return hue(tone).Alpha(alpha), t.Text
}

// hatched draws thin lines across b, rising to the right, in color.
func hatched(p *ui.Painter, b ui.Rect, color ui.Color) {
	var path ui.Path
	for x := b.X - b.H; x < b.X+b.W; x += 4 {
		path.MoveTo(x, b.Y+b.H)
		path.LineTo(x+b.H, b.Y)
	}
	p.Clip(b, 0, func() { p.StrokePath(&path, 1, color) })
}

// outline strokes one line around the boxes of a chunk's lines, rects,
// in r, top to bottom: down their right ends and up their left.
func outline(p *ui.Painter, rects []ui.Rect, r ui.Rect, color ui.Color, width float32) {
	if len(rects) == 0 {
		return
	}
	rs := slices.Clone(rects)
	slices.SortFunc(rs, func(a, b ui.Rect) int { return cmp.Compare(a.Y, b.Y) })
	var path ui.Path
	for i, b := range rs {
		x, y := r.X+b.X+b.W+1, r.Y+b.Y
		if i == 0 {
			path.MoveTo(x, y)
		} else {
			path.LineTo(x, y)
		}
		path.LineTo(x, y+b.H)
	}
	for _, b := range slices.Backward(rs) {
		x, y := r.X+b.X-1, r.Y+b.Y
		path.LineTo(x, y+b.H)
		path.LineTo(x, y)
	}
	path.Close()
	p.StrokePath(&path, width, color)
}

// union returns the box around rects.
func union(rects []ui.Rect) ui.Rect {
	if len(rects) == 0 {
		return ui.Rect{}
	}
	u := rects[0]
	for _, b := range rects[1:] {
		x0, y0 := min(u.X, b.X), min(u.Y, b.Y)
		x1, y1 := max(u.X+u.W, b.X+b.W), max(u.Y+u.H, b.Y+b.H)
		u = ui.Rect{X: x0, Y: y0, W: x1 - x0, H: y1 - y0}
	}
	return u
}

// list creates the list of chunks, those a query retrieved first, and
// returns the chunk under the pointer, or -1.
func (s *State) list(c *ui.Context, hits []Hit, retrieved []bool) (hot int) {
	t := c.Theme()
	hot = -1
	ui.Scroll(c).Width(t.Space(84)).Shrink(0).Children(func() {
		ui.Column(c).Padding(t.Space(4), t.Space(3), t.Space(6)).Gap(t.Space(1)).Children(func() {
			s.header(c)
			if hits != nil {
				section(c, "Retrieved")
				if len(hits) == 0 {
					ui.Text(c, "No chunk matches the query").FontSize(t.FontSize-1).TextColor(t.TextMuted).
						Padding(t.Space(1), t.Space(2.5))
				}
				for _, h := range hits {
					if s.row(c.Key(h.Chunk), h.Chunk, &h, hits[0].Score, false) {
						hot = h.Chunk
					}
				}
				if len(hits) < len(s.bounds) {
					section(c, "Not retrieved")
				}
			}
			for _, i := range s.order {
				if !retrieved[i] && s.row(c.Key(i), i, nil, 0, hits != nil) {
					hot = i
				}
			}
			s.revealList = false
		})
	})
	return hot
}

// header shows how many chunks there are and their mean size.
func (s *State) header(c *ui.Context) {
	t := c.Theme()
	n := len(s.bounds)
	ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Padding(0, t.Space(2.5), t.Space(2)).Children(func() {
		label := strconv.Itoa(n) + " chunks"
		if n == 1 {
			label = "1 chunk"
		}
		ui.Text(c, label).Bold()
		if n == 0 {
			return
		}
		chars, tokens := 0, 0
		for i := range n {
			chars += s.chars[i]
			tokens += s.tokens[i]
		}
		ui.Text(c, "mean "+strconv.Itoa(chars/n)+" chars · "+strconv.Itoa(tokens/n)+" tokens").
			FontSize(t.FontSize - 1).TextColor(t.TextMuted).FontFeatures("tnum").SingleLine().Shrink(1)
	})
}

// section shows the title of a part of the list.
func section(c *ui.Context, title string) {
	t := c.Theme()
	ui.Text(c, title).FontSize(t.FontSize-2).FontWeight(600).TextColor(t.TextMuted).
		Padding(t.Space(2), t.Space(2.5), t.Space(0.5))
}

// row creates the row of chunk i in the list: its number, its size and
// the start of its text, and when a query retrieved it, hit, its score
// with a bar of it against top, the highest. dim fades it while it is not
// lit. It returns whether the pointer is over it.
func (s *State) row(c *ui.Context, i int, hit *Hit, top float64, dim bool) bool {
	t := c.Theme()
	lit, sel := i == s.hot-1, i == s.selected-1
	sizes := strconv.Itoa(s.chars[i]) + " chars · " + strconv.Itoa(s.tokens[i]) + " tokens"
	b := ui.ButtonBase(c).Column().AlignItems(ui.Stretch).Gap(t.Space(1.5)).Padding(t.Space(2), t.Space(2.5)).
		Radius(t.Space(2)).Border(1, t.Accent.Alpha(0)).Label("Chunk " + strconv.Itoa(i+1)).Description(sizes)
	hovered := b.Hovered()
	if b.Clicked() {
		s.toggle(i)
	}
	switch {
	case sel:
		b.Background(t.Accent.Alpha(0.12)).Border(1, t.Accent)
	case lit || hovered:
		b.Background(t.Text.Alpha(0.06))
	case dim:
		b.Opacity(0.5)
	}
	if lit && s.revealList {
		b.ScrollIntoView()
	}
	b.Children(func() {
		ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
			bg, fg := badge(t, s.tone[i], hit != nil)
			h := t.Space(4.5)
			ui.Row(c).Height(h).MinWidth(h).Padding(0, t.Space(1.5)).Radius(h / 2).Background(bg).
				Justify(ui.Center).AlignItems(ui.Center).Children(func() {
				ui.Text(c, strconv.Itoa(i+1)).FontSize(t.FontSize - 2).FontWeight(600).FontFeatures("tnum").TextColor(fg)
			})
			ui.Text(c, sizes).FontSize(t.FontSize - 1).TextColor(t.TextMuted).FontFeatures("tnum").
				SingleLine().Grow(1).Shrink(1)
		})
		if hit != nil {
			score(c, hit.Score, top)
		}
		ui.Text(c, s.previews[i]).FontSize(t.FontSize - 1).LineHeight(1.35).MaxLines(2)
	})
	return hovered
}

// score shows a bar of v against top, the highest score, and v.
func score(c *ui.Context, v, top float64) {
	t := c.Theme()
	share := float32(1)
	if top > 0 {
		share = float32(min(max(v/top, 0), 1))
	}
	ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
		ui.Box(c).Height(t.Space(1.5)).Grow(1).Radius(t.Space(0.75)).Background(t.Text.Alpha(0.08)).
			Draw(func(p *ui.Painter, r ui.Rect) {
				p.Fill(ui.Rect{X: r.X, Y: r.Y, W: r.W * share, H: r.H}, t.Accent, r.H/2)
			})
		ui.Text(c, strconv.FormatFloat(v, 'f', 2, 64)).FontSize(t.FontSize - 2).FontFeatures("tnum").
			TextColor(t.TextMuted).Width(t.Space(9)).TextAlign(ui.End)
	})
}
