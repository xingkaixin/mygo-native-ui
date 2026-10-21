// Package reader shows a book a page at a time, as Apple Books and
// e-readers do. The text is set in a book face into pages that fit the
// view, within margins, its lines justified or ragged at the right, the
// first line of each paragraph indented, and each chapter starting a page
// under its title. The last line of a paragraph does not stand alone at
// the top of a page: the page before gives it one more. A view wide
// enough shows two pages side by side.
//
// →, Space and Page Down turn to the next page, and ←, Shift+Space and
// Page Up to the one before, while the view has the keyboard focus; so
// does a click on the right or left third of the pages, and a sideways
// swipe on a trackpad, which the pages follow. They slide, or change at
// once when the desktop asks for less motion. Below them are the
// chapter's title, "Page 12 of 240", and a scrubber with a notch where
// each chapter starts: the pointer over it shows the chapter and page
// under it, and a press or a drag goes there.
//
//	reader.View(c, &app.reader, &app.place, app.book).Grow(1)
//
// *place is the place read, which the app keeps to save it: the offset,
// in runes, of the first character shown, counting through the chapters'
// titles and paragraphs, and one more after each. Turning a page sets it
// to the page's first character, and the app sets it to go elsewhere.
// Another font size, line spacing or size of the view lays the book out
// anew, and shows the page holding *place, so that the first word read
// stays in view.
//
// The text is shaped in the background, once for each font size, and
// laid into pages from that in a frame, anew only as the pages or the
// settings change; until a font size is shaped, the pages keep the last
// one.
package reader

import (
	"cmp"
	"math"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
	"unicode"

	"github.com/egoist/mygo/ui"
)

// family is the book face the text is set in: Apple Books' own, then
// faces most systems have.
const family = "Iowan Old Style, Charter, Georgia, serif"

// The shape of a page, in ems of the text: titles are titleScale times
// as large, first lines are indented by indent, lines are at most measure
// long, and two pages show side by side when each fits lines spread
// long.
const (
	titleScale = 1.6
	indent     = 1.5
	measure    = 34
	spread     = 22
)

// keepSizes is how many font sizes keep their shaped text, to go back to
// at once.
const keepSizes = 3

// scrollIdle is how long after its last scroll a trackpad lets go of the
// pages: scrolling tells no gesture's end, which is when it stops, its
// momentum included.
const scrollIdle = 150 * time.Millisecond

// turnAfter is how far, in widths of the view, a swipe moves the pages
// to turn them once it lets go.
const turnAfter = 0.15

// slide moves the pages as they turn.
var slide = ui.Animation{Spring: ui.SpringOptions{Duration: 350 * time.Millisecond}}

// Chapter is a chapter of a book: its title, which starts a page, and its
// paragraphs, of plain text.
type Chapter struct {
	Title      string
	Paragraphs []string
}

// State is what a reader keeps: the book's text shaped in the last font
// sizes, its pages, the place of the pages as they slide, and the swipe
// moving them. FontSize (18 DIPs when 0), LineSpacing (a multiple of the
// font size, 1.5 when 0) and Justify, which the app sets, set the text.
// Keep it in the app's state, one for each view; the zero State lays out
// the book it is given, and starts over when given another slice of
// chapters.
type State struct {
	FontSize    float32
	LineSpacing float32
	Justify     bool

	// book is the first chapter of the book laid out, and chapters how
	// many it has, to tell another book; blocks are its text.
	book     *Chapter
	chapters int
	blocks   []block
	// shapes are the font sizes shaped or being shaped, the last shown
	// first, and pages the book laid out last.
	shapes []*shaping
	pages  *layout
	// shown is the spread the pages showed, a fraction between two as
	// they slide, from pages.
	shown     float32
	shownFrom *layout
	// scroll is what a trackpad scrolled since the last frame, scrolled
	// when it last did, zero once it let go, and drag how far it moved the
	// pages to the right while it held them.
	scroll   float32
	scrolled time.Time
	held     bool
	drag     float32
	// scrub is the page the scrubber sets, and placed the glyphs of a
	// justified line as it is drawn.
	scrub  float64
	placed []ui.Glyph
}

// block is a chapter's title or one of its paragraphs: its text and
// runes, the offset of its first rune in the book, its chapter, and
// whether it is a title, or the paragraph after one, which starts flush
// left.
type block struct {
	text    string
	runes   []rune
	start   int
	chapter int
	title   bool
	flush   bool
}

// load takes book's text, starting over when it is another book.
func (s *State) load(book []Chapter) {
	if s.book == &book[0] && s.chapters == len(book) {
		return
	}
	for _, sh := range s.shapes {
		sh.stop.Store(true)
	}
	s.book, s.chapters, s.shapes, s.pages, s.shownFrom = &book[0], len(book), nil, nil, nil
	// Work on the last book may still read its blocks.
	s.blocks = nil
	at := 0
	add := func(b block) {
		b.runes, b.start = []rune(b.text), at
		at += len(b.runes) + 1
		s.blocks = append(s.blocks, b)
	}
	for i, ch := range book {
		if ch.Title != "" {
			add(block{text: ch.Title, chapter: i, title: true})
		}
		for j, para := range ch.Paragraphs {
			add(block{text: para, chapter: i, flush: j == 0})
		}
	}
}

// fontSize returns the size of the text, FontSize or its default.
func (s *State) fontSize() float32 {
	if s.FontSize <= 0 {
		return 18
	}
	return s.FontSize
}

// lineSpacing returns the height of lines, in sizes of the text,
// LineSpacing or its default.
func (s *State) lineSpacing() float32 {
	if s.LineSpacing <= 0 {
		return 1.5
	}
	return s.LineSpacing
}

// shaping is the book's blocks shaped in a font size, in the background:
// the glyphs of each and where its lines may break, to read once done is
// set, and the metrics of the text and of the titles. stop tells the work
// to give up, as another size took its place.
type shaping struct {
	size       float32
	glyphs     [][]ui.Glyph
	breaks     [][]brk
	body, head ui.FontMetrics
	done, stop atomic.Bool
}

// brk is where a line may break in a block: the line before it ends at
// glyph end, the spaces after it left out, and the next begins at glyph
// at. space tells that spaces part them, which justified lines widen.
type brk struct {
	end, at int
	space   bool
}

// run shapes blocks, and calls wake once done.
func (sh *shaping) run(blocks []block, wake func()) {
	body := ui.Font{Family: family, Size: sh.size}
	head := ui.Font{Family: family, Size: sh.size * titleScale}
	sh.body, sh.head = body.Metrics(), head.Metrics()
	for i, b := range blocks {
		if sh.stop.Load() {
			return
		}
		f := body
		if b.title {
			f = head
		}
		g := ui.Shape(b.text, f)
		sh.glyphs[i], sh.breaks[i] = g, breaksOf(b.runes, g)
	}
	sh.done.Store(true)
	wake()
}

// breaksOf returns where the lines of a block of runes shaped into g may
// break: after spaces, and after a dash within a word. The last is the
// block's end.
func breaksOf(runes []rune, g []ui.Glyph) []brk {
	var br []brk
	end := 0
	for i := range g {
		r := runes[g[i].Cluster]
		if breakable(r) {
			continue
		}
		if i > 0 {
			last := runes[min(len(runes)-1, g[i-1].Cluster+max(1, g[i-1].Runes)-1)]
			switch {
			case breakable(last):
				br = append(br, brk{end, i, true})
			case (last == '-' || last == '–' || last == '—') && unicode.IsLetter(r):
				br = append(br, brk{i, i, false})
			}
		}
		end = i + 1
	}
	return append(br, brk{end, len(g), false})
}

// breakable reports whether r is a space a line may break at.
func breakable(r rune) bool {
	return unicode.IsSpace(r) && r != '\u00a0' && r != '\u2007' && r != '\u202f'
}

// shaped returns the blocks shaped in size, starting on it if need be;
// while it is under way, those shown last, or nil before any is done.
func (s *State) shaped(c *ui.Context, size float32) *shaping {
	// Sizes passed over on the way, as a slider drags, are not worth
	// finishing.
	s.shapes = slices.DeleteFunc(s.shapes, func(sh *shaping) bool {
		if sh.size != size && !sh.done.Load() {
			sh.stop.Store(true)
			return true
		}
		return false
	})
	i := slices.IndexFunc(s.shapes, func(sh *shaping) bool { return sh.size == size })
	if i < 0 {
		sh := &shaping{size: size, glyphs: make([][]ui.Glyph, len(s.blocks)), breaks: make([][]brk, len(s.blocks))}
		go sh.run(s.blocks, c.Invalidate)
		s.shapes = append(s.shapes, sh)
		i = len(s.shapes) - 1
	}
	if sh := s.shapes[i]; sh.done.Load() {
		s.shapes = slices.Insert(slices.Delete(s.shapes, i, i+1), 0, sh)
		s.shapes = s.shapes[:min(len(s.shapes), keepSizes)]
		return sh
	}
	for _, sh := range s.shapes {
		if sh.done.Load() {
			return sh
		}
	}
	return nil
}

// geometry is where the pages of a view go: one or two side by side, the
// left of the text on each and its width, the top of the text, how many
// lines of text a page holds, and their height.
type geometry struct {
	two   bool
	x     [2]float32
	w     float32
	top   float32
	lines int
	lh    float32
}

// geometryOf lays out pages in a view w by h DIPs, for text of size.
func geometryOf(t *ui.Theme, w, h, size, spacing float32) geometry {
	margin := max(t.Space(6), min(t.Space(16), w*0.06))
	g := geometry{two: w > h && w/2-2*margin >= spread*size, top: t.Space(12), lh: size * spacing}
	pw := w
	if g.two {
		pw = w / 2
	}
	g.w = max(8*size, min(pw-2*margin, measure*size))
	g.x = [2]float32{(pw - g.w) / 2, pw + (pw-g.w)/2}
	g.lines = max(1, int((h-g.top-t.Space(6))/g.lh))
	return g
}

// layout is the book laid out into pages: their lines, the pages, and
// what it was laid out from.
type layout struct {
	sh      *shaping
	w, lh   float32
	n       int
	justify bool
	lines   []line
	pages   []page
}

// line is a line of a block: glyphs g0 to g1, and the breaks k0 to k1
// within, which a justified line widens by gap each. x is where it starts
// on the page, and y its baseline, below the top of the text.
type line struct {
	block  int
	g0, g1 int
	k0, k1 int
	x, y   float32
	gap    float32
	last   bool
}

// page is a page of a layout: its lines, l0 to l1, its chapter, and the
// offset of its first character.
type page struct {
	l0, l1  int
	chapter int
	start   int
}

// layout returns the book shaped in sh laid out as g says, laying it out
// anew when that changed.
func (s *State) layout(sh *shaping, g geometry) *layout {
	if l := s.pages; l != nil && l.sh == sh && l.w == g.w && l.lh == g.lh && l.n == g.lines && l.justify == s.Justify {
		return l
	}
	s.pages = paginate(s.blocks, sh, g.w, g.lh, g.lines, s.Justify)
	return s.pages
}

// paginate lays blocks shaped in sh into pages of n lines lh high and w
// wide.
func paginate(blocks []block, sh *shaping, w, lh float32, n int, justify bool) *layout {
	l := &layout{sh: sh, w: w, lh: lh, n: n, justify: justify}
	size := sh.size
	base := (lh-sh.body.Ascent-sh.body.Descent)/2 + sh.body.Ascent
	titleH := size * titleScale * 1.25
	titleBase := (titleH-sh.head.Ascent-sh.head.Descent)/2 + sh.head.Ascent
	var head, body []line
	slots, start, chapter := 0, 0, -1
	flush := func() {
		if chapter < 0 {
			return
		}
		for i, first := 0, true; first || i < len(body); first = false {
			room, top := n, 0
			if first {
				room, top = max(0, n-slots), slots
			}
			end := min(len(body), i+room)
			// The last line of a paragraph goes over with the one before
			// it rather than alone.
			if end < len(body) && end-1 > i && body[end].last && body[end-1].block == body[end].block {
				end--
			}
			pg := page{l0: len(l.lines), chapter: chapter, start: start}
			if first {
				l.lines = append(l.lines, head...)
			} else if b := body[i]; b.g0 < len(sh.glyphs[b.block]) {
				pg.start = blocks[b.block].start + sh.glyphs[b.block][b.g0].Cluster
			} else {
				pg.start = blocks[b.block].start
			}
			for j := i; j < end; j++ {
				ln := body[j]
				ln.y = float32(top+j-i)*lh + base
				l.lines = append(l.lines, ln)
			}
			pg.l1 = len(l.lines)
			l.pages = append(l.pages, pg)
			i = end
		}
	}
	for i, b := range blocks {
		if b.chapter != chapter {
			flush()
			head, body, slots, start, chapter = head[:0], body[:0], 0, b.start, b.chapter
		}
		g, br := sh.glyphs[i], sh.breaks[i]
		if b.title {
			sink := 2 * lh
			head = wrap(head, i, g, br, w, 0, false, true)
			for j := range head {
				head[j].y = sink + float32(j)*titleH + titleBase
			}
			slots = int(math.Ceil(float64((sink + float32(len(head))*titleH + lh) / lh)))
			continue
		}
		first := float32(0)
		if !b.flush {
			first = indent * size
		}
		body = wrap(body, i, g, br, w, first, justify, false)
	}
	flush()
	return l
}

// wrap breaks block b, shaped into g with breaks br, into lines w wide,
// the first indented by first, and appends them to lines: justified with
// justify, but for the last, or centered with center.
func wrap(lines []line, b int, g []ui.Glyph, br []brk, w, first float32, justify, center bool) []line {
	if len(g) == 0 {
		return append(lines, line{block: b, last: true})
	}
	right := func(e int) float32 { return g[e-1].X + g[e-1].Advance }
	for s, k := 0, 0; s < len(g); {
		room, x := w, float32(0)
		if s == 0 {
			room, x = w-first, first
		}
		left := g[s].X
		ln := line{block: b, g0: s, k0: k, x: x}
		best := -1
		for j := k; j < len(br) && (br[j].end <= s || right(br[j].end)-left <= room); j++ {
			best = j
		}
		if best >= 0 {
			ln.g1, ln.k1 = max(s, br[best].end), best
			s, k = br[best].at, best+1
		} else {
			// A word longer than the line breaks where the line ends.
			e := s + 1
			for e < len(g) && right(e+1)-left <= room {
				e++
			}
			ln.g1, ln.k1 = e, k
			s = e
		}
		ln.last = s >= len(g)
		width := float32(0)
		if ln.g1 > ln.g0 {
			width = right(ln.g1) - left
		}
		switch {
		case center:
			ln.x = (w - width) / 2
		case justify && !ln.last && best >= 0:
			gaps := 0
			for _, b := range br[ln.k0:ln.k1] {
				if b.space {
					gaps++
				}
			}
			if gaps > 0 {
				ln.gap = (room - width) / float32(gaps)
			}
		}
		lines = append(lines, ln)
	}
	return lines
}

// pageAt returns the page holding offset at.
func (l *layout) pageAt(at int) int {
	i, found := slices.BinarySearchFunc(l.pages, at, func(p page, at int) int { return cmp.Compare(p.start, at) })
	if !found {
		i--
	}
	return max(0, min(len(l.pages)-1, i))
}

// View creates a reader of book, of which *place is the place read, and s
// keeps the rest. Give it a size, or grow it in its parent.
func View(c *ui.Context, s *State, place *int, book []Chapter) ui.Element {
	t := c.Theme()
	view := ui.Column(c).Focusable().Label("Book").Background(t.Background)
	if len(book) == 0 {
		return view
	}
	if s.load(book); len(s.blocks) == 0 {
		return view
	}
	sh := s.shaped(c, s.fontSize())
	view.Children(func() {
		area := ui.Box(c).Grow(1).Clip()
		b := area.LayoutBounds()
		area.HandleInput(func(ev ui.InputEvent) bool {
			// A scroll that starts sideways keeps the pages until it stops,
			// as a trackpad's strays up and down.
			if ev.Kind != ui.InputScroll || s.scrolled.IsZero() && abs(ev.DX) <= abs(ev.DY) {
				return false
			}
			s.scroll += ev.DX
			s.scrolled = ev.Time
			return true
		})
		var l *layout
		var g geometry
		if b.W > 0 && b.H > 0 {
			size := s.fontSize()
			if sh != nil {
				size = sh.size
			}
			g = geometryOf(t, b.W, b.H, size, s.lineSpacing())
			if sh != nil {
				l = s.layout(sh, g)
			}
		}
		per := 1
		if g.two {
			per = 2
		}
		pages, spreads, at := 0, 0, 0
		if l != nil {
			pages = len(l.pages)
			spreads, at = (pages+per-1)/per, l.pageAt(*place)/per
		}
		show := func(k int) int {
			if k = max(0, min(spreads-1, k)); l != nil && k != at {
				*place, at = l.pages[k*per].start, k
			}
			return at
		}
		if pressed(view, ui.KeyRight, ui.KeySpace, ui.KeyPageDown) {
			show(at + 1)
		}
		if pressed(view, ui.KeyLeft, ui.KeyPageUp) || view.Shortcut(ui.Shift, ui.KeySpace) {
			show(at - 1)
		}
		if area.Clicked() {
			switch x, _, _ := area.PointerPosition(); {
			case x < b.W/3:
				show(at - 1)
			case x > b.W*2/3:
				show(at + 1)
			}
		}
		shown := s.slide(c, area, l, at, spreads, b.W, show)

		area.Draw(func(p *ui.Painter, r ui.Rect) {
			if r.W != b.W || r.H != b.H {
				c.Invalidate()
			}
			for k := int(math.Floor(float64(shown))); k <= int(math.Ceil(float64(shown))); k++ {
				off := (float32(k) - shown) * r.W
				if k < 0 || k >= spreads || abs(off) >= r.W {
					continue
				}
				x := r.X + off
				for j := range per {
					if i := k*per + j; i < pages {
						s.draw(p, l, l.pages[i], x+g.x[j], r.Y+g.top, t.Text)
					}
				}
				if g.two {
					spine(p, x+r.W/2, r.Y, r.H, t)
				}
			}
		})

		ui.Column(c).Padding(0, b.W-g.x[per-1]-g.w, t.Space(4), g.x[0]).Gap(t.Space(1)).Children(func() {
			scrub := ui.SliderBase(c, &s.scrub, 0, float64(max(1, pages-1))).Step(float64(per)).
				Height(t.Space(6)).PaddingX(t.Space(1.5)).Label("Page")
			if scrub.Changed() {
				show(int(math.Round(s.scrub)) / per)
				if abs(shown-float32(at)) > 1.5 {
					shown = area.Jump("x", float32(at))
				}
			}
			s.scrub = float64(at * per)
			title, count := "", "Paginating…"
			if l != nil {
				title, count = book[l.pages[at*per].chapter].Title, pageText(at*per, per, pages)
				px, _, over := scrub.PointerPosition()
				scrubber(scrub, l, book, per, at*per, px, over || scrub.Pressed(), t)
			}
			scrub.Value(count)
			view.Value(count)
			ui.Row(c).Gap(t.Space(4)).Children(func() {
				ui.Text(c, title).SingleLine().Grow(1).Shrink(1).FontSize(t.FontSize - 1).TextColor(t.TextMuted)
				ui.Text(c, count).FontSize(t.FontSize - 1).TextColor(t.TextMuted).FontFeatures("tnum")
			})
		})
		s.shown = shown
	})
	return view
}

// slide returns the spread the pages show in this frame, a fraction
// between two as they slide to at, of spreads in l, w wide, or as they
// follow a swipe; show turns to another as the swipe lets go, and returns
// the spread shown then.
func (s *State) slide(c *ui.Context, area ui.Element, l *layout, at, spreads int, w float32, show func(int) int) float32 {
	scrolling := !s.scrolled.IsZero()
	if wait := scrollIdle - c.Now().Sub(s.scrolled); scrolling && wait > 0 {
		c.After(wait)
	} else {
		s.scrolled, scrolling = time.Time{}, false
	}
	switch {
	case l == nil:
		s.held, s.drag, s.scroll = false, 0, 0
		return 0
	case l != s.shownFrom:
		// Laid out anew, the spreads are others: no slide goes from the
		// old to the new.
		s.shownFrom, s.held, s.drag, s.scroll = l, false, 0, 0
		return area.Jump("x", float32(at))
	case scrolling:
		s.held = true
		s.drag = max(-w, min(w, s.drag-s.scroll))
		s.scroll = 0
		return area.Jump("x", ui.Rubberband(float32(at)-s.drag/w, 0, float32(spreads-1), 1))
	case s.held:
		moved := -s.drag / w
		s.held, s.drag = false, 0
		if moved > turnAfter {
			at = show(at + 1)
		} else if moved < -turnAfter {
			at = show(at - 1)
		}
		// Jumping in place forgets how fast the scroll went, which had
		// stopped by the time it let go.
		area.Jump("x", s.shown)
		return area.AnimateTo("x", float32(at), slide)
	case abs(s.shown-float32(at)) > 1.5:
		return area.Jump("x", float32(at))
	}
	return area.AnimateTo("x", float32(at), slide)
}

// pressed reports whether any of keys was pressed without modifiers while
// e had the focus.
func pressed(e ui.Element, keys ...ui.Key) bool {
	hit := false
	for _, k := range keys {
		if e.Shortcut(0, k) {
			hit = true
		}
	}
	return hit
}

// pageText reads out the pages from first, per of them, of pages in all.
func pageText(first, per, pages int) string {
	if per > 1 && first+1 < pages {
		return "Pages " + strconv.Itoa(first+1) + "–" + strconv.Itoa(first+2) + " of " + strconv.Itoa(pages)
	}
	return "Page " + strconv.Itoa(first+1) + " of " + strconv.Itoa(pages)
}

// draw draws page pg of l, the left of its text at x and its top at y,
// in color.
func (s *State) draw(p *ui.Painter, l *layout, pg page, x, y float32, color ui.Color) {
	for _, ln := range l.lines[pg.l0:pg.l1] {
		g := l.sh.glyphs[ln.block][ln.g0:ln.g1]
		if len(g) == 0 {
			continue
		}
		left := g[0].X
		if ln.gap == 0 {
			p.Glyphs(g, x+ln.x-left, y+ln.y, color)
			continue
		}
		br := l.sh.breaks[ln.block][ln.k0:ln.k1]
		placed, shift, j := s.placed[:0], -left, 0
		for i, gl := range g {
			for ; j < len(br) && br[j].at <= ln.g0+i; j++ {
				if br[j].space {
					shift += ln.gap
				}
			}
			gl.X += shift
			placed = append(placed, gl)
		}
		s.placed = placed
		p.Glyphs(placed, x+ln.x, y+ln.y, color)
	}
}

// spine shades the fold between two pages at x, from y down h.
func spine(p *ui.Painter, x, y, h float32, t *ui.Theme) {
	w := t.Space(10)
	shade := t.Text.Alpha(0.06)
	p.FillGradient(ui.Rect{X: x - w, Y: y, W: w, H: h}, ui.LinearGradient{From: shade.Alpha(0), To: shade, Angle: 90}, 0)
	p.FillGradient(ui.Rect{X: x, Y: y, W: w, H: h}, ui.LinearGradient{From: shade, To: shade.Alpha(0), Angle: 90}, 0)
	p.Line(x, y, x, y+h, 1, t.Border.Alpha(0.5))
}

// scrubber draws the scrubber of l's pages on bar, per a spread, with the
// spread from page at on it, and a notch where each chapter of book
// starts; hovered, it grows, and a bubble names the chapter and pages at
// px, where a press goes.
func scrubber(bar ui.Element, l *layout, book []Chapter, per, at int, px float32, hovered bool, t *ui.Theme) {
	pages := len(l.pages)
	pad := t.Space(1.5)
	frac := func(i int) float32 { return float32(i) / float32(max(1, pages-1)) }
	bar.Draw(func(p *ui.Painter, r ui.Rect) {
		x0, w, cy := r.X+pad, r.W-2*pad, r.Y+r.H/2
		th := float32(2)
		if hovered {
			th = 4
		}
		p.Fill(ui.Rect{X: x0, Y: cy - th/2, W: w, H: th}, t.Text.Alpha(0.12), th/2)
		p.Fill(ui.Rect{X: x0, Y: cy - th/2, W: w * frac(at), H: th}, t.TextMuted, th/2)
		// Notches closer than a few DIPs, as of many short chapters, would
		// cut the bar to dashes: those go.
		last := x0
		for i := 1; i < pages; i++ {
			if x := x0 + w*frac(i); l.pages[i].chapter != l.pages[i-1].chapter && x-last >= t.Space(3) {
				p.Fill(ui.Rect{X: x - 1, Y: cy - th, W: 2, H: 2 * th}, t.Background, 0)
				last = x
			}
		}
		knob := float32(10)
		if hovered {
			knob = 12
		}
		k := ui.Rect{X: x0 + w*frac(at) - knob/2, Y: cy - knob/2, W: knob, H: knob}
		p.Shadow(k, knob/2, 0, 1, 3, 0, ui.RGB(0, 0, 0).Alpha(0.2))
		p.Fill(k, t.Background, knob/2)
		p.Stroke(k, t.TextMuted, knob/2, 1.5)
	})
	if !hovered {
		return
	}
	bar.DrawOver(func(p *ui.Painter, r ui.Rect) {
		x0, w := r.X+pad, r.W-2*pad
		// The slider steps a spread at a time.
		i := per * int(math.Round(float64(max(0, min(1, (r.X+px-x0)/w))*float32(pages-1))/float64(per)))
		i = min(i, pages-1)
		bg, fg := t.Inverse, t.InverseText
		if bg.A == 0 {
			bg, fg = t.Text, t.Background
		}
		where := strings.TrimSuffix(pageText(i, per, pages), " of "+strconv.Itoa(pages))
		spans := []ui.Span{{Text: where, Size: t.FontSize - 1, Color: fg.Alpha(0.75)}}
		if title := book[l.pages[i].chapter].Title; title != "" {
			spans = []ui.Span{{Text: title, Size: t.FontSize - 1, Weight: 600, Color: fg}, {Text: "  " + where, Size: t.FontSize - 1, Color: fg.Alpha(0.75)}}
		}
		tw, th := p.MeasureText(0, spans...)
		bw, bh := tw+2*t.Space(2.5), th+2*t.Space(1.25)
		bx := max(r.X, min(r.X+r.W-bw, x0+w*frac(i)-bw/2))
		by := r.Y - bh
		p.Shadow(ui.Rect{X: bx, Y: by, W: bw, H: bh}, bh/2, 0, 2, 8, 0, ui.RGB(0, 0, 0).Alpha(0.2))
		p.Fill(ui.Rect{X: bx, Y: by, W: bw, H: bh}, bg, bh/2)
		p.RichText(bx+(bw-tw)/2, by+(bh-th)/2, 0, spans...)
	})
}

func abs(v float32) float32 { return float32(math.Abs(float64(v))) }
