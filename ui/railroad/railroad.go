// Package railroad draws a regular expression as a railroad diagram, as
// regexper does: the pattern runs from a start mark to an end mark, left
// to right along one track; letters in rounded boxes, a space in them as
// "·"; classes, such as [a-z0-9], "digit" and "any character", in boxes of
// another style; anchors and boundaries, such as ^ and \b, in small pills;
// alternatives on branches that split from the track and rejoin it. What
// ? makes optional has a track around it above, what + repeats one back
// below it, and what * does both; a count, such as {2,5}, labels its loop
// "2–5 times", and a lazy one "lazy". Each group is a dashed frame in its
// color, those of package regex, labeled with its number or name. Where
// the pattern does not compile, what is wrong shows instead. Patterns are
// Go's, RE2's syntax.
//
//	railroad.View(c, &app.diagram, app.pattern).Grow(1)
package railroad

import (
	"fmt"
	"regexp/syntax"
	"strings"
	"unicode"

	"github.com/egoist/mygo/ui"
)

// State is what a view keeps: the diagram of the pattern it drew last.
// Keep it in the app's state, one for each view.
type State struct {
	pattern string
	size    float32
	d       *diagram
	err     error
}

type kind uint8

const (
	literal kind = iota
	class
	anchor
	frame
	note
	start
	end
)

// shape is a part of a diagram: a box, a frame or a mark at r, with its
// text, if any, at x, y, and the group it frames.
type shape struct {
	kind  kind
	r     ui.Rect
	text  string
	x, y  float32
	group int
}

// diagram is a pattern laid out: its size, the track, and the shapes
// over it.
type diagram struct {
	w, h   float32
	track  ui.Path
	shapes []shape
}

// item is a part of a diagram laid out: w wide, up above its track and
// down below; put adds it with its track coming in at x, y.
type item struct {
	w, up, down float32
	put         func(x, y float32)
}

type builder struct {
	c *ui.Context
	t *ui.Theme
	d *diagram
	// r is the radius of a turn, gap the room between parts.
	r, gap float32
}

// View creates the diagram of pattern. Give it a size, or Grow; it
// scrolls when it does not fit.
func View(c *ui.Context, s *State, pattern string) ui.Element {
	t := c.Theme()
	if s.d == nil && s.err == nil || s.pattern != pattern || s.size != t.FontSize {
		s.pattern, s.size = pattern, t.FontSize
		s.d, s.err = nil, nil
		re, err := syntax.Parse(pattern, syntax.Perl)
		if err != nil {
			s.err = err
		} else {
			s.d = layout(c, re)
		}
	}
	return ui.ScrollBoth(c).Children(func() {
		if s.err != nil {
			ui.Text(c, problem(s.err)).TextColor(t.Danger).Padding(t.Space(6))
			return
		}
		d := s.d
		ui.Box(c).Size(d.w, d.h).Draw(func(p *ui.Painter, r ui.Rect) {
			p.Transform(ui.Translation(r.X, r.Y), 0, 0, func() { paint(p, t, d) })
		})
	})
}

// problem returns what is wrong with a pattern, without the package's
// prefix: "missing closing ): `(ab`".
func problem(err error) string {
	return strings.TrimPrefix(err.Error(), "error parsing regexp: ")
}

func layout(c *ui.Context, re *syntax.Regexp) *diagram {
	t := c.Theme()
	b := &builder{c: c, t: t, d: &diagram{}, r: t.Space(2.5), gap: t.Space(3)}
	root := b.node(re)
	pad, dot, lead := t.Space(6), t.Space(1.5), t.Space(3)
	y := pad + root.up
	b.d.w = 2*pad + 4*dot + 2*lead + root.w
	b.d.h = 2*pad + root.up + root.down
	x := pad
	b.d.shapes = append(b.d.shapes, shape{kind: start, r: ui.Rect{X: x, Y: y - dot, W: 2 * dot, H: 2 * dot}})
	x += 2 * dot
	b.line(x, y, x+lead)
	root.put(x+lead, y)
	x += lead + root.w
	b.line(x, y, x+lead)
	x += lead
	b.d.shapes = append(b.d.shapes, shape{kind: end, r: ui.Rect{X: x, Y: y - dot, W: 2 * dot, H: 2 * dot}})
	return b.d
}

func (b *builder) node(re *syntax.Regexp) item {
	switch re.Op {
	case syntax.OpNoMatch:
		return b.box(class, "nothing")
	case syntax.OpLiteral:
		var sb strings.Builder
		for _, r := range re.Rune {
			sb.WriteString(escape(r, false))
		}
		return b.box(literal, sb.String())
	case syntax.OpCharClass:
		return b.box(class, classText(re.Rune))
	case syntax.OpAnyCharNotNL:
		return b.box(class, "any character")
	case syntax.OpAnyChar:
		return b.box(class, "any character or newline")
	case syntax.OpBeginLine:
		return b.box(anchor, "start of line")
	case syntax.OpEndLine:
		return b.box(anchor, "end of line")
	case syntax.OpBeginText:
		return b.box(anchor, "start of text")
	case syntax.OpEndText:
		return b.box(anchor, "end of text")
	case syntax.OpWordBoundary:
		return b.box(anchor, "word boundary")
	case syntax.OpNoWordBoundary:
		return b.box(anchor, "not word boundary")
	case syntax.OpCapture:
		return b.group(re, b.node(re.Sub[0]))
	case syntax.OpStar, syntax.OpPlus, syntax.OpQuest, syntax.OpRepeat:
		return b.repeat(re, b.node(re.Sub[0]))
	case syntax.OpConcat:
		return b.seq(re.Sub)
	case syntax.OpAlternate:
		return b.choice(re.Sub)
	}
	return item{put: func(x, y float32) {}}
}

// span returns the style of text in a shape of kind k.
func span(t *ui.Theme, k kind, text string) ui.Span {
	fs := t.FontSize
	switch {
	case k == literal || k == class && strings.HasPrefix(text, "["):
		return ui.Span{Text: text, Font: "monospace", Size: fs}
	case k == class:
		return ui.Span{Text: text, Size: fs - 1}
	case k == frame:
		return ui.Span{Text: text, Size: fs - 2, Weight: 600}
	}
	return ui.Span{Text: text, Size: fs - 2}
}

func (b *builder) measure(k kind, text string) (float32, float32) {
	if text == "" {
		return 0, 0
	}
	return b.c.MeasureText(0, span(b.t, k, text))
}

// box returns a box of kind k showing text.
func (b *builder) box(k kind, text string) item {
	tw, th := b.measure(k, text)
	padX, h := b.t.Space(2.5), b.t.FontSize+b.t.Space(3.5)
	if k == anchor {
		padX, h = b.t.Space(2), th+b.t.Space(1.5)
	}
	w := tw + 2*padX
	return item{w: w, up: h / 2, down: h / 2, put: func(x, y float32) {
		b.d.shapes = append(b.d.shapes, shape{kind: k, r: ui.Rect{X: x, Y: y - h/2, W: w, H: h}, text: text, x: x + padX, y: y - th/2})
	}}
}

func (b *builder) line(x0, y, x1 float32) {
	if x1 > x0 {
		b.d.track.MoveTo(x0, y).LineTo(x1, y)
	}
}

// turn adds a quarter turn to the track from x0, y0 to x1, y1, leaving
// across when across is set, else up or down.
func (b *builder) turn(x0, y0, x1, y1 float32, across bool) {
	cx, cy := x0, y1
	if across {
		cx, cy = x1, y0
	}
	// 0.552 makes the cubic a quarter of a circle.
	const k = 0.552
	b.d.track.CubeTo(x0+(cx-x0)*k, y0+(cy-y0)*k, x1+(cx-x1)*k, y1+(cy-y1)*k, x1, y1)
}

// fit puts s in the middle of w at x, y, with track on either side.
func (b *builder) fit(s item, x, y, w float32) {
	off := (w - s.w) / 2
	b.line(x, y, x+off)
	s.put(x+off, y)
	b.line(x+off+s.w, y, x+w)
}

func (b *builder) seq(subs []*syntax.Regexp) item {
	items := make([]item, len(subs))
	var out item
	for k, sub := range subs {
		items[k] = b.node(sub)
		if k > 0 {
			out.w += b.gap
		}
		out.w += items[k].w
		out.up, out.down = max(out.up, items[k].up), max(out.down, items[k].down)
	}
	out.put = func(x, y float32) {
		for k, s := range items {
			if k > 0 {
				b.line(x, y, x+b.gap)
				x += b.gap
			}
			s.put(x, y)
			x += s.w
		}
	}
	return out
}

// choice lays out the alternatives, the first on the track and the others
// under it, on branches that split from it and rejoin it.
func (b *builder) choice(subs []*syntax.Regexp) item {
	r := b.r
	items := make([]item, len(subs))
	at := make([]float32, len(subs))
	var inner float32
	for k, sub := range subs {
		items[k] = b.node(sub)
		inner = max(inner, items[k].w)
		if k > 0 {
			at[k] = max(at[k-1]+items[k-1].down+b.t.Space(2)+items[k].up, at[k-1]+2*r)
		}
	}
	last := len(items) - 1
	out := item{w: inner + 4*r, up: items[0].up, down: at[last] + items[last].down}
	out.put = func(x, y float32) {
		w := out.w
		b.line(x, y, x+2*r)
		b.fit(items[0], x+2*r, y, inner)
		b.line(x+w-2*r, y, x+w)
		for k := 1; k <= last; k++ {
			yk := y + at[k]
			b.d.track.MoveTo(x, y)
			b.turn(x, y, x+r, y+r, true)
			b.d.track.LineTo(x+r, yk-r)
			b.turn(x+r, yk-r, x+2*r, yk, false)
			b.fit(items[k], x+2*r, yk, inner)
			b.d.track.MoveTo(x+w-2*r, yk)
			b.turn(x+w-2*r, yk, x+w-r, yk-r, true)
			b.d.track.LineTo(x+w-r, y+r)
			b.turn(x+w-r, y+r, x+w, y, false)
		}
	}
	return out
}

// repeat lays out s as re repeats it: with a track around it above when
// it may be left out, one back below it when it may repeat, and a label
// on the loop, or on the bypass when there is none, for counts and lazy
// repeats.
func (b *builder) repeat(re *syntax.Regexp, s item) item {
	lo, hi := re.Min, re.Max
	switch re.Op {
	case syntax.OpStar:
		lo, hi = 0, -1
	case syntax.OpPlus:
		lo, hi = 1, -1
	case syntax.OpQuest:
		lo, hi = 0, 1
	}
	var label []string
	if re.Op == syntax.OpRepeat {
		switch {
		case lo == hi:
			label = append(label, times(lo))
		case hi < 0:
			label = append(label, fmt.Sprintf("%d+ times", lo))
		default:
			label = append(label, fmt.Sprintf("%d–%d times", lo, hi))
		}
	}
	if re.Flags&syntax.NonGreedy != 0 {
		label = append(label, "lazy")
	}
	text := strings.Join(label, ", ")
	if hi < 0 || hi > 1 {
		s = b.loop(s, text)
		text = ""
	}
	if lo == 0 {
		s = b.skip(s, text)
	}
	return s
}

func times(n int) string {
	if n == 1 {
		return "1 time"
	}
	return fmt.Sprintf("%d times", n)
}

// loop returns s with a track from its end back to its start below it,
// labeled with text.
func (b *builder) loop(s item, text string) item {
	r := b.r
	tw, th := b.measure(note, text)
	w := max(s.w, tw) + 2*r
	below := max(s.down+b.t.Space(2), 2*r)
	out := item{w: w, up: s.up, down: below}
	if text != "" {
		out.down += b.t.Space(1) + th
	}
	out.put = func(x, y float32) {
		b.line(x, y, x+r)
		b.fit(s, x+r, y, w-2*r)
		b.line(x+w-r, y, x+w)
		yl := y + below
		b.d.track.MoveTo(x+w-r, y)
		b.turn(x+w-r, y, x+w, y+r, true)
		b.d.track.LineTo(x+w, yl-r)
		b.turn(x+w, yl-r, x+w-r, yl, false)
		b.d.track.LineTo(x+r, yl)
		b.turn(x+r, yl, x, yl-r, true)
		b.d.track.LineTo(x, y+r)
		b.turn(x, y+r, x+r, y, false)
		if text != "" {
			b.d.shapes = append(b.d.shapes, shape{kind: note, text: text, x: x + (w-tw)/2, y: yl + b.t.Space(1)})
		}
	}
	return out
}

// skip returns s with a track around it above, labeled with text.
func (b *builder) skip(s item, text string) item {
	r := b.r
	tw, th := b.measure(note, text)
	w := max(s.w, tw) + 4*r
	above := max(s.up+b.t.Space(2), 2*r)
	out := item{w: w, up: above, down: s.down}
	if text != "" {
		out.up += b.t.Space(1) + th
	}
	out.put = func(x, y float32) {
		b.line(x, y, x+2*r)
		b.fit(s, x+2*r, y, w-4*r)
		b.line(x+w-2*r, y, x+w)
		yb := y - above
		b.d.track.MoveTo(x, y)
		b.turn(x, y, x+r, y-r, true)
		b.d.track.LineTo(x+r, yb+r)
		b.turn(x+r, yb+r, x+2*r, yb, false)
		b.d.track.LineTo(x+w-2*r, yb)
		b.turn(x+w-2*r, yb, x+w-r, yb+r, true)
		b.d.track.LineTo(x+w-r, y-r)
		b.turn(x+w-r, y-r, x+w, y, false)
		if text != "" {
			b.d.shapes = append(b.d.shapes, shape{kind: note, text: text, x: x + (w-tw)/2, y: yb - b.t.Space(1) - th})
		}
	}
	return out
}

// group returns s in the frame of capture group re, labeled with its name
// or number.
func (b *builder) group(re *syntax.Regexp, s item) item {
	text := re.Name
	if text == "" {
		text = fmt.Sprintf("group %d", re.Cap)
	}
	tw, th := b.measure(frame, text)
	pad := b.t.Space(2)
	w := max(s.w, tw) + 2*pad
	out := item{w: w, up: s.up + pad + th + b.t.Space(1), down: s.down + pad}
	out.put = func(x, y float32) {
		b.d.shapes = append(b.d.shapes, shape{kind: frame, r: ui.Rect{X: x, Y: y - out.up, W: w, H: out.up + out.down},
			text: text, x: x + pad, y: y - out.up + b.t.Space(1), group: re.Cap})
		b.line(x, y, x+pad)
		b.fit(s, x+pad, y, w-2*pad)
		b.line(x+w-pad, y, x+w)
	}
	return out
}

// paint draws d: the frames of groups under the track, and the boxes,
// labels and marks over it.
func paint(p *ui.Painter, t *ui.Theme, d *diagram) {
	text := func(s shape, color ui.Color) {
		sp := span(t, s.kind, s.text)
		sp.Color = color
		p.RichText(s.x, s.y, 0, sp)
	}
	for _, s := range d.shapes {
		if s.kind == frame {
			c := color(s.group, t.Dark)
			p.Fill(s.r, c.Alpha(0.05), t.Space(2))
			p.StrokeDashed(s.r, c.Alpha(0.8), t.Space(2), 1.5)
			text(s, c)
		}
	}
	p.StrokePath(&d.track, 2, t.TextMuted.Alpha(0.7))
	for _, s := range d.shapes {
		switch s.kind {
		case literal:
			p.Fill(s.r, t.Background, t.Space(2))
			p.Stroke(s.r, t.Text.Alpha(0.55), t.Space(2), 1.5)
			text(s, t.Text)
		case class:
			p.Fill(s.r, t.Background, 2)
			p.Fill(s.r, t.Accent.Alpha(0.12), 2)
			p.Stroke(s.r, t.Accent.Alpha(0.6), 2, 1.5)
			text(s, t.Text)
		case anchor:
			p.Fill(s.r, t.Background, s.r.H/2)
			p.Fill(s.r, t.TextMuted.Alpha(0.16), s.r.H/2)
			text(s, t.Text)
		case note:
			text(s, t.TextMuted)
		case start, end:
			var dot ui.Path
			dot.Circle(s.r.X+s.r.W/2, s.r.Y+s.r.H/2, s.r.W/2)
			p.FillPath(&dot, t.TextMuted)
			if s.kind == end {
				var ring ui.Path
				ring.Circle(s.r.X+s.r.W/2, s.r.Y+s.r.H/2, s.r.W/2+t.Space(0.75))
				p.StrokePath(&ring, 1.5, t.TextMuted)
			}
		}
	}
}

// color returns the color of group g, as package regex colors it.
func color(g int, dark bool) ui.Color {
	l := float32(0.55)
	if dark {
		l = 0.75
	}
	return ui.Oklch(l, 0.16, float32((250+g*67)%360))
}

// perl are the classes of Perl's escapes, ASCII only as in RE2, \w
// before \d, which it holds.
var perl = []struct {
	esc, name string
	runes     []rune
}{
	{`\w`, "word character", []rune{'0', '9', 'A', 'Z', '_', '_', 'a', 'z'}},
	{`\d`, "digit", []rune{'0', '9'}},
	{`\s`, "whitespace", []rune{'\t', '\n', '\f', '\r', ' ', ' '}},
}

// classText returns class rs, pairs of the first and last rune of each
// range, as a pattern writes it, with Perl's escapes in it, its ranges
// complemented after "^" when it holds the last rune, or "digit", "word
// character" and "whitespace" for those alone.
func classText(rs []rune) string {
	if len(rs) == 2 && rs[0] == 0 && rs[1] == unicode.MaxRune {
		return "any character"
	}
	neg := len(rs) > 0 && rs[0] == 0 && rs[len(rs)-1] == unicode.MaxRune
	if neg {
		var inv []rune
		for i := 1; i+1 < len(rs); i += 2 {
			inv = append(inv, rs[i]+1, rs[i+1]-1)
		}
		rs = inv
	}
	var ascii [128]bool
	var rest []rune
	for i := 0; i+1 < len(rs); i += 2 {
		for r := rs[i]; r <= min(rs[i+1], 127); r++ {
			ascii[r] = true
		}
		if rs[i+1] >= 128 {
			rest = append(rest, max(rs[i], 128), rs[i+1])
		}
	}
	var parts []string
	name := ""
	for _, k := range perl {
		all := true
		for i := 0; i < len(k.runes); i += 2 {
			for r := k.runes[i]; r <= k.runes[i+1]; r++ {
				all = all && ascii[r]
			}
		}
		if !all {
			continue
		}
		for i := 0; i < len(k.runes); i += 2 {
			for r := k.runes[i]; r <= k.runes[i+1]; r++ {
				ascii[r] = false
			}
		}
		parts, name = append(parts, k.esc), k.name
	}
	var ranges []rune
	for r := rune(0); r < 128; r++ {
		if !ascii[r] {
			continue
		}
		if n := len(ranges); n > 0 && ranges[n-1] == r-1 {
			ranges[n-1] = r
		} else {
			ranges = append(ranges, r, r)
		}
	}
	ranges = append(ranges, rest...)
	if len(parts) == 1 && len(ranges) == 0 {
		if neg {
			return "non-" + name
		}
		return name
	}
	for i := 0; i+1 < len(ranges); i += 2 {
		if len(parts) == 8 {
			parts = append(parts, "…")
			break
		}
		lo, hi := ranges[i], ranges[i+1]
		switch {
		case lo == hi:
			parts = append(parts, escape(lo, true))
		case hi == lo+1:
			parts = append(parts, escape(lo, true)+escape(hi, true))
		default:
			parts = append(parts, escape(lo, true)+"-"+escape(hi, true))
		}
	}
	if neg {
		return "[^" + strings.Join(parts, "") + "]"
	}
	return "[" + strings.Join(parts, "") + "]"
}

// escape returns r as a box shows it: a space as "·", what does not
// show escaped as in a pattern, and in a class what has a meaning there.
func escape(r rune, inClass bool) string {
	switch r {
	case ' ':
		return "·"
	case '\t':
		return `\t`
	case '\n':
		return `\n`
	case '\r':
		return `\r`
	case '\f':
		return `\f`
	case '\v':
		return `\v`
	}
	if inClass && strings.ContainsRune(`\-[]^`, r) {
		return `\` + string(r)
	}
	switch {
	case unicode.IsPrint(r):
	case r < 0x100:
		return fmt.Sprintf(`\x%02x`, r)
	default:
		return fmt.Sprintf(`\x{%x}`, r)
	}
	return string(r)
}
