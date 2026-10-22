// Package vertical sets Chinese and Japanese in vertical lines (竖排,
// 縦書き), as books in those languages are set: each paragraph in columns
// read top to bottom, the columns placed right to left. Chinese
// characters and kana stand upright, each in a square cell; 。，、 sit in
// the top right of theirs, ：；！？ in the middle, and brackets, dashes,
// ellipses and the long vowel mark turn a quarter clockwise, as their
// vertical forms do. Runs of
// Latin letters lie on their side, turned a quarter clockwise too, and so
// do numbers, but for one of up to three digits, which stands across a
// single cell (縦中横).
//
// A column ends where the next character no longer fits the view's
// height, but never before closing punctuation, as 。」, nor after opening
// punctuation, as 「（: the character before goes on to the next column
// with it. Latin text breaks between words. Each line of the text is a
// paragraph, which starts a column; indent it with ideographic spaces, as
// books do. Columns beyond the view's width scroll sideways, the first at
// the right.
//
//	vertical.View(c, &app.vertical, text, 20).Grow(1)
//
// Text with kana in it is set in a Mincho face, as Japanese, and other
// text in a Song face, as simplified Chinese.
package vertical

import (
	"slices"
	"strings"
	"unicode"

	"github.com/egoist/mygo/ui"
)

// The book faces the text is set in, those of macOS first: their
// punctuation is as wide as a character, which a system face's, shared
// with Latin text, is not.
const (
	chinese  = "Songti SC, STSong, SimSun, Noto Serif CJK SC, serif"
	japanese = "Hiragino Mincho ProN, Yu Mincho, MS Mincho, Noto Serif CJK JP, serif"
)

// What sets the columns, as shares of the text's size: how far apart
// their middles are, how far below the top of its cell a character's
// baseline is, and how far up and right 。，、 move to the top right of
// theirs.
const (
	pitch  = 1.8
	drop   = 0.88
	corner = 0.5
)

// The characters that set otherwise than upright: those that move to the
// top right of their cell, and those that turn a quarter.
const (
	cornered = "、。，．"
	turned   = "「」『』（）《》〈〉【】〔〕〖〗〘〙〚〛｛｝［］｟｠—―…‥～〜ー－＿“”‘’"
)

// upstand sets ：；！？ in their vertical forms, which stand in the middle
// of their cell, where a Song face's sit at the left.
var upstand = strings.NewReplacer("：", "︓", "；", "︔", "！", "︕", "？", "︖")

// What may not start a column, and what may not end one.
const (
	noStart = "、。，．：；！？）」』】〕〗〙〛》〉｝］｠”’・ー々〻ゝゞヽヾぁぃぅぇぉっゃゅょゎァィゥェォッャュョヮヵヶ…‥"
	noEnd   = "（「『【〔〖〘〚《〈｛［｟“‘"
)

// State is what a view keeps: the text shaped into cells, and the cells
// laid into columns as high as the view. Keep it in the app's state, one
// for each view; the zero State sets the text it is given.
type State struct {
	key   key
	cells []cell
	// paras are where each paragraph starts in cells, followed by where
	// the last ends.
	paras []int
	// base is how far a Latin run's baseline is from the middle of its
	// column.
	base float32
	// cols are the cells of each column, laid out height high.
	cols   [][2]int
	height float32
}

// key is what shaping depends on.
type key struct {
	text string
	size float32
}

// setting is how a cell sets in its column.
type setting int

const (
	upright setting = iota
	cornerwise
	turn
	sideways
	across
)

// cell is a character, or the digits standing across one: its glyphs,
// moved to start at 0, w wide as shaped, h long down the column, and its
// first rune. brk tells that a column may start at it, as far as the
// Latin text it is in goes.
type cell struct {
	glyphs []ui.Glyph
	set    setting
	w, h   float32
	r      rune
	brk    bool
}

// View creates text set in vertical lines, its characters size DIPs
// high, in columns as high as the view. Give it a size, or grow it in its
// parent.
func View(c *ui.Context, s *State, text string, size float32) ui.Element {
	t := c.Theme()
	if k := (key{text, size}); k != s.key || s.paras == nil {
		s.shape(k)
	}
	view := ui.ScrollHorizontal(c).Direction(ui.RightToLeft)
	h := view.LayoutBounds().H
	if h != s.height || s.cols == nil {
		s.lay(h)
	}
	view.Draw(func(p *ui.Painter, r ui.Rect) {
		if r.H != s.height {
			c.Invalidate()
		}
	})
	view.Children(func() {
		col := pitch * size
		box := ui.Box(c).Size(float32(len(s.cols))*col, h).Label(text)
		box.Draw(func(p *ui.Painter, r ui.Rect) {
			seen := p.Visible()
			for i, span := range s.cols {
				cx := r.X + r.W - (float32(i)+0.5)*col
				if cx+col < seen.X || cx-col > seen.X+seen.W {
					continue
				}
				y := r.Y
				for _, ce := range s.cells[span[0]:span[1]] {
					s.draw(p, ce, cx, y, t.Text)
					y += ce.h
				}
			}
		})
	})
	return view
}

// draw draws cell ce in the column whose middle is at x, from y down, in
// color.
func (s *State) draw(p *ui.Painter, ce cell, x, y float32, color ui.Color) {
	size := s.key.size
	left, base := x-ce.w/2, y+drop*size
	switch ce.set {
	case upright:
		p.Glyphs(ce.glyphs, left, base, color)
	case cornerwise:
		p.Glyphs(ce.glyphs, left+corner*size, base-corner*size, color)
	case turn:
		p.Transform(ui.Rotation(90), x, y+size/2, func() { p.Glyphs(ce.glyphs, left, base, color) })
	case sideways:
		p.Transform(ui.Rotation(90), x, y, func() { p.Glyphs(ce.glyphs, x, y+s.base, color) })
	case across:
		// Digits wider than the cell narrow to fit it, as fonts' third
		// widths do.
		if fit := 0.9 * size; ce.w > fit {
			p.Transform(ui.Scaling(fit/ce.w, 1), x, base, func() { p.Glyphs(ce.glyphs, left, base, color) })
			return
		}
		p.Glyphs(ce.glyphs, left, base, color)
	}
}

// shape shapes the text of k into cells, a paragraph a line.
func (s *State) shape(k key) {
	family := chinese
	if strings.ContainsFunc(k.text, func(r rune) bool { return unicode.In(r, unicode.Hiragana, unicode.Katakana) }) {
		family = japanese
	}
	font := ui.Font{Family: family, Size: k.size}
	m := font.Metrics()
	s.key, s.base = k, (m.Ascent-m.Descent)/2
	s.cells, s.paras, s.cols = s.cells[:0], s.paras[:0], nil
	for _, para := range strings.Split(k.text, "\n") {
		s.paras = append(s.paras, len(s.cells))
		runes := []rune(para)
		g := ui.Shape(upstand.Replace(para), font)
		for a := 0; a < len(g); {
			b := a + 1
			for b < len(g) && g[b].Cluster == g[a].Cluster {
				b++
			}
			ce := cell{glyphs: append([]ui.Glyph(nil), g[a:b]...), r: runes[g[a].Cluster], h: k.size, brk: true}
			x0 := g[a].X
			for i := range ce.glyphs {
				ce.glyphs[i].X -= x0
				ce.w = max(ce.w, ce.glyphs[i].X+ce.glyphs[i].Advance)
			}
			ce.set = settingOf(ce.r)
			if ce.set == sideways {
				ce.h = ce.w
				if last := len(s.cells) - 1; last >= s.paras[len(s.paras)-1] && s.cells[last].set == sideways {
					ce.brk = s.cells[last].r == ' '
				}
			}
			s.cells = append(s.cells, ce)
			a = b
		}
		s.digits(s.paras[len(s.paras)-1])
	}
	s.paras = append(s.paras, len(s.cells))
}

// digits sets each Latin run from cell from on that is a number of up to
// three digits across a single cell.
func (s *State) digits(from int) {
	for i := from; i < len(s.cells); i++ {
		if s.cells[i].set != sideways {
			continue
		}
		j := i
		for j < len(s.cells) && s.cells[j].set == sideways {
			j++
		}
		// The spaces Latin text keeps around a number go with it.
		a, z := i, j
		for a < z && s.cells[a].r == ' ' {
			a++
		}
		for z > a && s.cells[z-1].r == ' ' {
			z--
		}
		if a == z || z-a > 3 || slices.ContainsFunc(s.cells[a:z], func(d cell) bool { return d.r < '0' || d.r > '9' }) {
			i = j
			continue
		}
		ce := cell{set: across, r: s.cells[a].r, h: s.key.size, brk: true}
		for _, d := range s.cells[a:z] {
			for _, g := range d.glyphs {
				g.X += ce.w
				ce.glyphs = append(ce.glyphs, g)
			}
			ce.w += d.w
		}
		s.cells = append(append(s.cells[:i], ce), s.cells[j:]...)
	}
}

// settingOf returns how a cell starting with r sets: what Chinese and
// Japanese text holds upright, but for punctuation of other forms when
// vertical, and the rest on its side.
func settingOf(r rune) setting {
	switch {
	case strings.ContainsRune(cornered, r):
		return cornerwise
	case strings.ContainsRune(turned, r):
		return turn
	case unicode.In(r, unicode.Han, unicode.Hiragana, unicode.Katakana, unicode.Hangul, unicode.Bopomofo),
		r >= 0x3000 && r <= 0x33ff, r >= 0xfe10 && r <= 0xfe4f, r >= 0xff00 && r <= 0xffef, r >= 0x1f000:
		return upright
	}
	return sideways
}

// lay lays the paragraphs' cells into columns height high.
func (s *State) lay(height float32) {
	s.height, s.cols = height, s.cols[:0]
	for p := 0; p+1 < len(s.paras); p++ {
		i, end := s.paras[p], s.paras[p+1]
		if i == end {
			s.cols = append(s.cols, [2]int{i, i})
		}
		for i < end {
			j, y := i, float32(0)
			// A space past the end of a column hangs there, unseen.
			for j < end && (j == i || y+s.cells[j].h <= height || s.cells[j].r == ' ') {
				y += s.cells[j].h
				j++
			}
			k := j
			for k < end && k > i && !s.breaks(k) {
				k--
			}
			if k == i {
				k = j
			}
			s.cols = append(s.cols, [2]int{i, k})
			i = k
		}
	}
}

// breaks reports whether a column may start at cell k.
func (s *State) breaks(k int) bool {
	return s.cells[k].brk && !strings.ContainsRune(noStart, s.cells[k].r) && !strings.ContainsRune(noEnd, s.cells[k-1].r)
}
