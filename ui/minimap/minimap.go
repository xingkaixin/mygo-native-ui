// Package minimap shows a code editor's whole text small beside it, as
// Visual Studio Code's minimap does: each line in blocks of its tokens'
// colors, from the editor's language and theme, and a box over the lines
// the editor shows. Dragging the box scrolls the editor, and a press
// elsewhere scrolls the line pressed to the middle of the editor and drags
// from there. When the text is taller than the minimap, the minimap
// scrolls along with the editor, so that the box goes from its top to its
// bottom as the editor goes from the first line to the last.
//
//	ui.Row(c).Grow(1).AlignItems(ui.Stretch).Children(func() {
//		editor.View(c, app.code).Grow(1)
//		minimap.View(c, app.code, app.marks)
//	})
//
// Marks the app passes show where its search matched, the problems of the
// text and the lines changed, in the editor theme's colors for them.
package minimap

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

// Kind is what a mark shows: a match of a search, an error or a warning,
// over its text, or lines added, modified or deleted, as a bar beside
// them.
type Kind uint8

const (
	Match Kind = iota
	Error
	Warning
	Added
	Modified
	Deleted
)

// Mark marks the text from byte Start to End of the editor's buffer. A
// mark of lines added, modified or deleted marks the lines Start and End
// are in; one of lines deleted marks the line after them.
type Mark struct {
	Kind       Kind
	Start, End int
}

// The size of a line and of a column in the minimap, in DIPs, as Visual
// Studio Code's at its scale 1; the height of a line's blocks, which
// leaves a gap between lines; and the minimap's width.
const (
	rowH  = 2
	colW  = 1
	inkH  = 1.5
	width = 120
)

// cache is the text of a buffer in the blocks a minimap draws, kept until
// the text, its language or the tab size changes.
type cache struct {
	text    editor.Snapshot
	buf     *editor.Buffer
	version uint64
	lang    *editor.Language
	tab     int
	// starts indexes the blocks of each line in blocks, and one past the
	// last line.
	starts []int32
	blocks []block
	// vis is how many lines the editor showed last while it showed lines
	// after them, for when it scrolls past the end, where fewer show.
	vis int
}

// block is a run of a line's characters of one kind of token, without
// spaces, by columns.
type block struct {
	from, to int32
	kind     editor.TokenKind
}

// drag is a drag of the box: the lines in view as it began, which the
// drag keeps so that the box does not jump as lines come partly into view,
// the pointer's height then, and where the box was.
type drag struct {
	active   bool
	vis      int
	y0, top0 float32
}

func (k *cache) refresh(buf *editor.Buffer, tab int) {
	text := buf.Snapshot()
	lang := buf.Language()
	if k.buf == buf && k.version == text.Version() && k.lang == lang && k.tab == tab {
		return
	}
	*k = cache{text: text, buf: buf, version: text.Version(), lang: lang, tab: tab, starts: k.starts[:0], blocks: k.blocks[:0], vis: k.vis}
	var tz editor.Tokenizer
	if lang != nil {
		tz = lang.Tokenizer
	}
	var state uint64
	var toks []editor.Token
	// Cutting the whole text is a hundred times as fast as reading it a
	// line at a time.
	rest := text.String()
	for i := range text.LineCount() {
		k.starts = append(k.starts, int32(len(k.blocks)))
		var line string
		line, rest, _ = strings.Cut(rest, "\n")
		toks = toks[:0]
		if tz != nil {
			toks, state = tz.Tokenize(line, state, toks)
		}
		toks = append(toks, editor.Token{End: len(line), Kind: editor.TokenText})
		col, start := int32(0), 0
		for _, tk := range toks {
			end := min(max(tk.End, start), len(line))
			for _, r := range line[start:end] {
				switch {
				case r == '\t':
					col += int32(tab) - col%int32(tab)
					continue
				case r == ' ':
				case len(k.blocks) > int(k.starts[i]) && k.blocks[len(k.blocks)-1].to == col && k.blocks[len(k.blocks)-1].kind == tk.Kind:
					k.blocks[len(k.blocks)-1].to++
				default:
					k.blocks = append(k.blocks, block{from: col, to: col + 1, kind: tk.Kind})
				}
				col++
			}
			start = end
		}
	}
	k.starts = append(k.starts, int32(len(k.blocks)))
}

// shown returns how many lines the editor shows from first to last of n.
func (k *cache) shown(first, last, n int) int {
	vis := max(1, last-first+1)
	if last < n-1 {
		k.vis = vis
	}
	return max(vis, k.vis)
}

// end returns the column after the last character of line i.
func (k *cache) end(i int) int32 {
	if a, b := k.starts[i], k.starts[i+1]; b > a {
		return k.blocks[b-1].to
	}
	return 0
}

// column returns the column of byte i of a line.
func (k *cache) column(line string, i int) int32 {
	col := int32(0)
	for _, r := range line[:min(max(i, 0), len(line))] {
		if r == '\t' {
			col += int32(k.tab) - col%int32(k.tab)
		} else {
			col++
		}
	}
	return col
}

// geometry is where a minimap h high shows n lines while the editor shows
// vis of them from line first: how far its rows scroll up, and the box.
type geometry struct {
	offset       float32
	boxTop, boxH float32
	// maxFirst is the last line the editor scrolls to the top.
	maxFirst int
	fits     bool
}

func place(n, first, vis int, h float32, pastEnd bool) geometry {
	g := geometry{boxH: float32(vis) * rowH, maxFirst: n - 1}
	if !pastEnd {
		g.maxFirst = n - vis
	}
	g.maxFirst = max(0, g.maxFirst)
	g.fits = float32(n)*rowH <= h || g.maxFirst == 0
	if g.fits {
		g.boxTop = float32(first) * rowH
		return g
	}
	g.boxTop = float32(first) / float32(g.maxFirst) * max(0, h-g.boxH)
	g.offset = float32(first)*rowH - g.boxTop
	return g
}

// firstAt returns the line the editor scrolls to the top for the box's
// top at y, by place's geometry.
func (g geometry) firstAt(y, h float32) int {
	f := y / rowH
	if !g.fits {
		f = y / max(1, h-g.boxH) * float32(g.maxFirst)
	}
	return max(0, min(int(f+0.5), g.maxFirst))
}

// View creates the minimap of the editor of code, with marks over it. Give
// it a height, as by stretching it beside the editor in a row; it is as
// wide as Visual Studio Code's. Build it after the editor in the frame, so
// that it shows the lines the editor shows.
//
// Lines show in the colors of the tokens of the buffer's language, by
// its Tokenizer, and in the theme's text color for a language without
// one. Lines longer than the minimap is wide are cut. It shows a line for
// each of the buffer's, whatever the editor folds or wraps.
func View(c *ui.Context, code *editor.State, marks []Mark) ui.Element {
	t := c.Theme()
	if code.Buffer == nil {
		code.Buffer = editor.NewBuffer("")
	}
	view := ui.Box(c).Width(width).Shrink(0).Clip()
	k := ui.Local(view, "minimap.cache", func() cache { return cache{} })
	d := ui.Local(view, "minimap.drag", func() drag { return drag{} })
	tab := code.TabSize
	if l := code.Buffer.Language(); tab <= 0 && l != nil {
		tab = l.TabSize
	}
	k.refresh(code.Buffer, max(1, tab))
	n := len(k.starts) - 1
	pastEnd := !code.NoScrollPastEnd
	h := view.LayoutBounds().H

	first, last := code.VisibleLines()
	if _, _, ok := view.Dragged(); ok {
		_, y, _ := view.PointerPosition()
		if !d.active {
			vis := k.shown(first, last, n)
			g := place(n, first, vis, h, pastEnd)
			to := first
			if y < g.boxTop || y >= g.boxTop+g.boxH {
				to = max(0, min(int((y+g.offset)/rowH)-vis/2, g.maxFirst))
				code.ScrollTo(to)
			}
			*d = drag{active: true, vis: vis, y0: y, top0: place(n, to, vis, h, pastEnd).boxTop}
		} else if to := place(n, 0, d.vis, h, pastEnd).firstAt(d.top0+y-d.y0, h); to != first {
			code.ScrollTo(to)
		}
	} else {
		d.active = false
	}

	def := editor.DefaultTheme(t)
	ct := code.Theme
	if ct == nil {
		ct = def
	}
	pick := func(c, otherwise ui.Color) ui.Color {
		if c.A > 0 {
			return c
		}
		return otherwise
	}
	text := pick(ct.Text, def.Text)
	var ink [editor.TokenInvalid + 1]ui.Color
	for i := range ink {
		ink[i] = pick(ct.Tokens[i].Color, pick(def.Tokens[i].Color, text)).Alpha(0.8)
	}
	opaque := func(c ui.Color) ui.Color { c.A = 255; return c }
	markColor := [...]ui.Color{
		Match:    opaque(pick(ct.CurrentMatch, def.CurrentMatch)),
		Error:    pick(ct.Error, def.Error),
		Warning:  pick(ct.Warning, def.Warning),
		Added:    pick(ct.Added, def.Added),
		Modified: pick(ct.Modified, def.Modified),
		Deleted:  pick(ct.Deleted, def.Deleted),
	}
	background := pick(ct.Background, def.Background)
	box := t.Text.Alpha(0.08)
	switch {
	case d.active:
		box = t.Text.Alpha(0.2)
	case view.Hovered():
		box = t.Text.Alpha(0.13)
	}
	gutter, pad := t.Space(2), t.Space(1.5)

	view.Draw(func(p *ui.Painter, r ui.Rect) {
		p.Fill(r, background, 0)
		if n <= 0 {
			return
		}
		first, last := code.VisibleLines()
		first, last = min(first, n-1), min(last, n-1)
		g := place(n, first, k.shown(first, last, n), r.H, pastEnd)
		top := max(0, int(g.offset/rowH))
		bottom := min(n, int((g.offset+r.H)/rowH)+1)
		cols := int32((r.W - gutter - pad) / colW)
		y := func(line int) float32 { return r.Y + float32(line)*rowH - g.offset }
		run := func(line int, from, to int32, h float32, color ui.Color) {
			from, to = min(from, cols), min(to, cols)
			if to > from {
				p.Fill(ui.Rect{X: r.X + gutter + float32(from)*colW, Y: y(line), W: float32(to-from) * colW, H: h}, color, 0)
			}
		}
		for i := top; i < bottom; i++ {
			for _, b := range k.blocks[k.starts[i]:k.starts[i+1]] {
				run(i, b.from, b.to, inkH, ink[min(b.kind, editor.TokenInvalid)])
			}
		}
		for _, m := range marks {
			if int(m.Kind) >= len(markColor) {
				continue
			}
			a, b := k.text.Pos(m.Start), k.text.Pos(max(m.Start, m.End))
			if b.Line < top || a.Line >= bottom {
				continue
			}
			color := markColor[m.Kind]
			if m.Kind >= Added {
				p.Fill(ui.Rect{X: r.X + t.Space(0.5), Y: y(a.Line), W: t.Space(0.75), H: float32(b.Line-a.Line+1) * rowH}, color, 0)
				continue
			}
			for line := max(a.Line, top); line <= min(b.Line, bottom-1); line++ {
				from, to := int32(0), k.end(line)
				if line == a.Line {
					from = k.column(k.text.Line(line), a.Col)
				}
				if line == b.Line {
					to = k.column(k.text.Line(line), b.Col)
				}
				run(line, from, max(to, from+2), rowH, color)
			}
		}
		p.Fill(ui.Rect{X: r.X, Y: r.Y + g.boxTop, W: r.W, H: g.boxH}, box, 0)
	})
	return view
}
