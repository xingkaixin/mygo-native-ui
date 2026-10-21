package tex

import (
	"github.com/egoist/mygo/ui"
	"golang.org/x/image/font/sfnt"
)

// style is a TeX style: display, text, script or scriptscript, each with
// a cramped version, one more, whose superscripts sit lower.
type style uint8

const (
	display      style = 0
	textStyle    style = 2
	scriptStyle  style = 4
	scriptScript style = 6
	noStyle      style = 255
)

func (s style) level() int    { return int(s) / 2 }
func (s style) cramp() style  { return s | 1 }
func (s style) cramped() bool { return s&1 != 0 }

// up is the style of s's scripts, num of its numerators: their
// denominators and subscripts are those cramped.
func (s style) up() style  { return style(min(max(s.level()+1, 2), 3)*2) | s&1 }
func (s style) num() style { return style(min(s.level()+1, 3)*2) | s&1 }

// spacing is the room between atoms by their classes: 1 thin, 2 medium,
// 3 thick, negative only in display and text styles.
var spacing = [8][8]int8{
	ord:   {0, 1, -2, -3, 0, 0, 0, -1},
	op:    {1, 1, 0, -3, 0, 0, 0, -1},
	bin:   {-2, -2, 0, 0, -2, 0, 0, -2},
	rel:   {-3, -3, 0, 0, -3, 0, 0, -3},
	open:  {0, 0, 0, 0, 0, 0, 0, 0},
	close: {0, 1, -2, -3, 0, 0, 0, -1},
	punct: {-1, -1, 0, -1, -1, -1, -1, -1},
	inner: {-1, 1, -2, -3, -1, 0, -1, -1},
}

// box is laid out math: w wide, h above its baseline and d below, of
// items placed from its origin, at the left of its baseline.
type box struct {
	w, h, d float32
	// italic is the italic correction of a character.
	italic float32
	items  []item
}

// item is a glyph of the math font at size em, glyphs of the system's, or
// a rule w by h, its top at y.
type item struct {
	x, y float32
	g    *outline
	em   float32
	sys  []ui.Glyph
	w, h float32
	ink  ink
}

// add puts o's items in b with its origin at x, y.
func (b *box) add(o box, x, y float32) {
	for _, it := range o.items {
		it.x, it.y = it.x+x, it.y+y
		b.items = append(b.items, it)
	}
}

// then puts o after b on its baseline.
func (b *box) then(o box) {
	b.add(o, b.w, 0)
	b.w += o.w
	b.h, b.d = max(b.h, o.h), max(b.d, o.d)
}

// lower moves b's content down by dy.
func (b *box) lower(dy float32) {
	for i := range b.items {
		b.items[i].y += dy
	}
	b.h, b.d = b.h-dy, b.d+dy
}

func rule(x, y, w, h float32, k ink) item { return item{x: x, y: y, w: w, h: h, ink: k} }

type layout struct {
	m *mathFont
	// size is the em of display and text styles, in DIPs.
	size float32
}

func (l *layout) konst(i int) float32 {
	if l.m != nil {
		return l.m.k[i]
	}
	return defaults[i]
}

func (l *layout) em(s style) float32 {
	switch s.level() {
	case 2:
		return l.size * l.konst(scriptPercentScaleDown)
	case 3:
		return l.size * l.konst(scriptScriptPercentScaleDown)
	}
	return l.size
}

// k returns constant i at style s, in DIPs.
func (l *layout) k(i int, s style) float32 { return l.konst(i) * l.em(s) }

// list lays out atoms one after another in style st, with TeX's room
// between them.
func (l *layout) list(atoms []*atom, st style) box {
	cls := make([]class, len(atoms))
	prev := -1
	for i, a := range atoms {
		if a.kind == space || a.kind == setStyle {
			continue
		}
		c := a.class
		if c == bin && (prev < 0 || cls[prev] == bin || cls[prev] == op || cls[prev] == rel || cls[prev] == open || cls[prev] == punct) {
			c = ord
		}
		if prev >= 0 && cls[prev] == bin && (c == rel || c == close || c == punct) {
			cls[prev] = ord
		}
		cls[i], prev = c, i
	}
	if prev >= 0 && cls[prev] == bin {
		cls[prev] = ord
	}
	var out box
	prev = -1
	for i, a := range atoms {
		switch a.kind {
		case setStyle:
			st = a.style
			continue
		case space:
			out.w += a.size * l.em(st) / 18
			continue
		}
		if prev >= 0 {
			n := spacing[cls[prev]][cls[i]]
			if n < 0 && st.level() < 2 {
				n = -n
			}
			out.w += [4]float32{0, 3, 4, 5}[max(n, 0)] * l.em(st) / 18
		}
		out.then(l.atom(a, st))
		prev = i
	}
	return out
}

// atom lays out a in style st, with its primes and scripts.
func (l *layout) atom(a *atom, st style) box {
	em := l.em(st)
	var b box
	switch a.kind {
	case sym:
		if a.class == op {
			b = l.bigOp(a, st)
		} else {
			b = l.char(a.r, a.alpha, em, a.ink)
		}
	case text:
		b = l.text(a.text, a.alpha, em, a.ink)
	case group:
		b = l.list(a.body, st)
	case frac:
		b = l.frac(a, st)
	case radical:
		b = l.radical(a, st)
	case fence:
		b = l.fence(a, st)
	case middle:
		b = l.delimiter(a.r, 0, st, a.ink)
	case delim:
		b = l.delimiter(a.r, a.size*em, st, a.ink)
	case accent:
		b = l.accent(a, st)
	case bar:
		b = l.bar(a, st)
	case brace:
		b = l.brace(a, st)
	case array:
		b = l.array(a, st)
	}
	if a.primes > 0 {
		b.w += b.italic
		b.italic = 0
		b.then(l.char([]rune("′″‴⁗")[min(a.primes, 4)-1], roman, em, a.ink))
	}
	if a.sub == nil && a.sup == nil {
		b.w += b.italic
		return b
	}
	if a.limits > 0 || a.limits == 0 && a.class == op && st.level() == 0 {
		return l.limits(b, a, st)
	}
	return l.scripts(b, a, st, a.kind == sym && a.class != op || a.kind == text)
}

// char returns r in alphabet a, em DIPs to the em: the math font's glyph,
// or the system's where it has none.
func (l *layout) char(r rune, a alphabet, em float32, k ink) box {
	if l.m != nil {
		o := l.m.rune(styled(r, a))
		if o == nil {
			o = l.m.rune(r)
		}
		if o != nil {
			return glyphBox(o, em, k)
		}
	}
	if s, ok := spacingAccents[r]; ok {
		r = s
	}
	f := fallback(a, em)
	switch a {
	case normal:
		f.Italic = styled(r, a) != r
	case script, fraktur, doubleStruck:
		r = styled(r, a)
	}
	return system(string(r), f, k)
}

// fallback returns the system's font for alphabet a.
func fallback(a alphabet, em float32) ui.Font {
	f := ui.Font{Family: "serif", Size: em, Italic: a == italic || a == boldItalic}
	switch a {
	case sans:
		f.Family = "sans-serif"
	case mono:
		f.Family = "monospace"
	case bold, boldItalic:
		f.Weight = 700
	}
	return f
}

// spacingAccents are the accents standing alone that the system's fonts
// have, for those of the math font.
var spacingAccents = map[rune]rune{
	0x302: 'ˆ', 0x30C: 'ˇ', 0x303: '˜', 0x301: '´', 0x300: '`', 0x307: '˙', 0x308: '¨',
	0x306: '˘', 0x304: '¯', 0x30A: '˚', 0x20D7: '→', 0x20D6: '←', 0x20E1: '↔',
}

func glyphBox(o *outline, em float32, k ink) box {
	return box{w: o.adv * em, h: -o.y0 * em, d: o.y1 * em, italic: o.italic * em, items: []item{{g: o, em: em, ink: k}}}
}

// system returns s set by the system in font f. Its glyphs say nothing of
// their height, so it takes that of letters.
func system(s string, f ui.Font, k ink) box {
	gs := ui.Shape(s, f)
	var w float32
	for _, g := range gs {
		w = max(w, g.X+g.Advance)
	}
	return box{w: w, h: 0.72 * f.Size, d: 0.22 * f.Size, items: []item{{sys: gs, ink: k, em: f.Size}}}
}

// text returns s in alphabet a, the system setting what the math font
// lacks.
func (l *layout) text(s string, a alphabet, em float32, k ink) box {
	var b box
	var missing []rune
	flush := func() {
		if len(missing) > 0 {
			b.then(system(string(missing), fallback(a, em), k))
			missing = missing[:0]
		}
	}
	for _, r := range s {
		var o *outline
		if l.m != nil {
			o = l.m.rune(styled(r, a))
		}
		if o == nil {
			missing = append(missing, r)
			continue
		}
		flush()
		b.then(glyphBox(o, em, k))
	}
	flush()
	return b
}

// bigOp returns a large operator: in display style its variant at least
// the font's display size, and centered on the axis.
func (l *layout) bigOp(a *atom, st style) box {
	em := l.em(st)
	b := l.char(a.r, a.alpha, em, a.ink)
	if st.level() == 0 {
		if b.items[0].g == nil {
			b = l.char(a.r, a.alpha, em*1.4, a.ink)
		} else if c := l.m.construction(a.r, true); c != nil {
			v := c.variants[len(c.variants)-1]
			for i, size := range c.sizes {
				if size >= l.konst(displayOperatorMinHeight) {
					v = c.variants[i]
					break
				}
			}
			b = glyphBox(v, em, a.ink)
		}
	}
	b.lower((b.h-b.d)/2 - l.k(axisHeight, st))
	return b
}

// scripts puts a's scripts beside base, which is a character or a word
// when char: those of other bases hang from their top and bottom.
func (l *layout) scripts(base box, a *atom, st style, char bool) box {
	k := func(i int) float32 { return l.k(i, st) }
	var u, v float32
	if !char {
		u, v = base.h-k(superscriptBaselineDropMax), base.d+k(subscriptBaselineDropMin)
	}
	var sup, sub box
	if a.sup != nil {
		sup = l.list(a.sup, st.up())
		shift := k(superscriptShiftUp)
		if st.cramped() {
			shift = k(superscriptShiftUpCramped)
		}
		u = max(u, shift, sup.d+k(superscriptBottomMin))
	}
	if a.sub != nil {
		sub = l.list(a.sub, st.up().cramp())
		v = max(v, k(subscriptShiftDown))
		if a.sup == nil {
			v = max(v, sub.h-k(subscriptTopMax))
		}
	}
	if a.sup != nil && a.sub != nil {
		if gap := (u - sup.d) - (sub.h - v); gap < k(subSuperscriptGapMin) {
			v += k(subSuperscriptGapMin) - gap
			if psi := k(superscriptBottomMaxWithSubscript) - (u - sup.d); psi > 0 {
				u, v = u+psi, v-psi
			}
		}
	}
	// The italic correction of a large operator, as of an integral, takes
	// its subscript back under it; that of a letter, its superscript out.
	sx, bx := base.w, base.w+base.italic
	if a.class == op {
		sx, bx = base.w-base.italic, base.w
	}
	out := base
	out.items = append([]item(nil), base.items...)
	var w float32
	if a.sup != nil {
		out.add(sup, bx, -u)
		out.h = max(out.h, u+sup.h)
		w = bx + sup.w
	}
	if a.sub != nil {
		out.add(sub, sx, v)
		out.d = max(out.d, v+sub.d)
		w = max(w, sx+sub.w)
	}
	out.w, out.italic = max(base.w, w)+k(spaceAfterScript), 0
	return out
}

// limits puts a's scripts above and below base.
func (l *layout) limits(base box, a *atom, st style) box {
	k := func(i int) float32 { return l.k(i, st) }
	var sup, sub box
	if a.sup != nil {
		sup = l.list(a.sup, st.up())
	}
	if a.sub != nil {
		sub = l.list(a.sub, st.up().cramp())
	}
	w := max(base.w, sup.w, sub.w)
	out := box{w: w, h: base.h, d: base.d}
	out.add(base, (w-base.w)/2, 0)
	if a.sup != nil {
		rise := max(k(upperLimitBaselineRiseMin), k(upperLimitGapMin)+sup.d)
		out.add(sup, (w-sup.w+base.italic)/2, -base.h-rise)
		out.h = base.h + rise + sup.h
	}
	if a.sub != nil {
		drop := max(k(lowerLimitBaselineDropMin), k(lowerLimitGapMin)+sub.h)
		out.add(sub, (w-sub.w-base.italic)/2, base.d+drop)
		out.d = base.d + drop + sub.d
	}
	return out
}

func (l *layout) frac(a *atom, st style) box {
	fs := st
	if a.style != noStyle {
		fs = a.style
	}
	k := func(i int) float32 { return l.k(i, fs) }
	num, den := l.list(a.body, fs.num()), l.list(a.den, fs.num().cramp())
	axis, disp := k(axisHeight), fs.level() == 0
	var u, v, t float32
	if a.r == 0 {
		t = k(fractionRuleThickness)
		u, v = k(fractionNumeratorShiftUp), k(fractionDenominatorShiftDown)
		gn, gd := k(fractionNumeratorGapMin), k(fractionDenominatorGapMin)
		if disp {
			u, v = k(fractionNumeratorDisplayStyleShiftUp), k(fractionDenominatorDisplayStyleShiftDown)
			gn, gd = k(fractionNumDisplayStyleGapMin), k(fractionDenomDisplayStyleGapMin)
		}
		u = max(u, num.d+axis+t/2+gn)
		v = max(v, den.h-axis+t/2+gd)
	} else {
		u, v = k(stackTopShiftUp), k(stackBottomShiftDown)
		gap := k(stackGapMin)
		if disp {
			u, v, gap = k(stackTopDisplayStyleShiftUp), k(stackBottomDisplayStyleShiftDown), k(stackDisplayStyleGapMin)
		}
		if g := (u - num.d) - (den.h - v); g < gap {
			u, v = u+(gap-g)/2, v+(gap-g)/2
		}
	}
	// TeX puts null delimiters on both sides of a fraction.
	pad := 0.12 * l.em(fs)
	if a.r != 0 {
		pad = 0
	}
	w := max(num.w, den.w)
	out := box{w: w + 2*pad, h: u + num.h, d: v + den.d}
	out.add(num, pad+(w-num.w)/2, -u)
	out.add(den, pad+(w-den.w)/2, v)
	if t > 0 {
		out.items = append(out.items, rule(pad, -axis-t/2, w, t, a.ink))
	}
	if a.r == 0 {
		return out
	}
	size := l.delimSize(out.h, out.d, fs)
	b := l.delimiter(a.r, size, fs, a.ink)
	b.then(out)
	b.then(l.delimiter(a.r2, size, fs, a.ink))
	return b
}

func (l *layout) radical(a *atom, st style) box {
	k := func(i int) float32 { return l.k(i, st) }
	body := l.list(a.body, st.cramp())
	t, gap := k(radicalRuleThickness), k(radicalVerticalGap)
	if st.level() == 0 {
		gap = k(radicalDisplayStyleVerticalGap)
	}
	target := body.h + body.d + gap + t
	sign := l.stretch('√', target, true, l.em(st), a.ink)
	if sign.items[0].sys != nil {
		sign = radicalSign(target, t, l.em(st), a.ink)
	}
	if excess := sign.h + sign.d - target; excess > 0 {
		gap += excess / 2
	}
	top := body.h + gap + t
	sign.lower(sign.h - top)
	out := box{h: top + k(radicalExtraAscender), d: max(body.d, sign.d)}
	if a.den != nil {
		deg := l.list(a.den, scriptScript)
		raise := l.konst(radicalDegreeBottomRaisePercent) * (sign.h + sign.d)
		y := sign.d - raise - deg.d
		x := k(radicalKernBeforeDegree)
		out.add(deg, x, y)
		out.h = max(out.h, deg.h-y)
		out.w = max(0, x+deg.w+k(radicalKernAfterDegree))
	}
	out.add(sign, out.w, 0)
	out.w += sign.w
	out.items = append(out.items, rule(out.w, -top, body.w, t, a.ink))
	out.add(body, out.w, 0)
	out.w += body.w
	return out
}

// radicalSign draws a radical sign reaching size above its baseline, for
// fonts without one, with its rule t thick.
func radicalSign(size, t, em float32, k ink) box {
	w := 0.56 * size / em
	if size > 1.2*em {
		w = 0.56 * 1.2
	}
	s, th := size/em, t/em
	o := &outline{adv: w, x0: 0, y0: -s, x1: w, y1: 0}
	pts := [][2]float32{
		{0, -0.45 * s}, {0.1 * w, -0.5 * s}, {0.38 * w, -0.12 * s}, {w - th, -s},
		{w, -s}, {w, -s + th}, {0.42 * w, 0}, {0.33 * w, 0}, {0.12 * w, -0.4 * s}, {0.02 * w, -0.37 * s},
	}
	for i, p := range pts {
		o.ops = append(o.ops, map[bool]sfnt.SegmentOp{true: sfnt.SegmentOpMoveTo, false: sfnt.SegmentOpLineTo}[i == 0])
		o.pts = append(o.pts, p)
	}
	return box{w: w * em, h: size, items: []item{{g: o, em: em, ink: k}}}
}

// stretch returns glyph r at least size: tall when vertical, else wide.
// That is r, its smallest variant reaching size, or one built of its parts;
// without them, the system's r set larger.
func (l *layout) stretch(r rune, size float32, vertical bool, em float32, k ink) box {
	b := l.char(r, roman, em, k)
	extent := func(b box) float32 {
		if vertical {
			return b.h + b.d
		}
		return b.w
	}
	if extent(b) >= size {
		return b
	}
	if b.items[0].g == nil {
		if vertical {
			return l.char(r, roman, em*size/extent(b), k)
		}
		return b
	}
	c := l.m.construction(r, vertical)
	if c == nil || len(c.variants) == 0 {
		return b
	}
	for i, v := range c.variants {
		if c.sizes[i]*em >= size {
			return glyphBox(v, em, k)
		}
	}
	if len(c.parts) == 0 {
		return glyphBox(c.variants[len(c.variants)-1], em, k)
	}
	return assemble(c.parts, size/em, vertical, em, k)
}

// assemble builds a glyph size ems long of parts, end to end, its
// extenders drawn out to make it up. Parts overlapping, as the font
// allows, would draw their edges twice there, darker.
func assemble(parts []part, size float32, vertical bool, em float32, k ink) box {
	var fixed float32
	n := 0
	for _, p := range parts {
		if p.extender {
			n++
		} else {
			fixed += p.full
		}
	}
	var stretch float32
	if n > 0 {
		stretch = max(0, size-fixed) / float32(n)
	}
	g := &outline{}
	var at float32
	for _, p := range parts {
		s, full := float32(1), p.full
		if p.extender {
			if stretch == 0 {
				continue
			}
			s, full = stretch/p.full, stretch
		}
		for _, q := range p.g.pts {
			if vertical {
				g.pts = append(g.pts, [2]float32{q[0], -at + (q[1]-p.g.y1)*s})
			} else {
				g.pts = append(g.pts, [2]float32{at + q[0]*s, q[1]})
			}
		}
		g.ops = append(g.ops, p.g.ops...)
		g.adv = max(g.adv, p.g.adv)
		at += full
	}
	g.bound()
	b := glyphBox(g, em, k)
	if vertical {
		b.h, b.d = at*em, 0
	} else {
		b.w = at * em
	}
	return b
}

// delimSize returns how tall delimiters around content h high and d deep
// are, as TeX sizes them: covering 90% of it, or all but half an em.
func (l *layout) delimSize(h, d float32, st style) float32 {
	axis := l.k(axisHeight, st)
	delta := max(h-axis, d+axis)
	return max(2*delta*0.901, 2*delta-0.5*l.em(st))
}

// delimiter returns delimiter r at least size tall, centered on the axis;
// 0 is none, only the room TeX leaves for one.
func (l *layout) delimiter(r rune, size float32, st style, k ink) box {
	if r == 0 {
		return box{w: 0.12 * l.em(st)}
	}
	b := l.stretch(r, size, true, l.em(st), k)
	b.lower((b.h-b.d)/2 - l.k(axisHeight, st))
	return b
}

func (l *layout) fence(a *atom, st style) box {
	var parts []box
	var mids []*atom
	start := 0
	for i, x := range a.body {
		if x.kind == middle {
			parts, mids = append(parts, l.list(a.body[start:i], st)), append(mids, x)
			start = i + 1
		}
	}
	parts = append(parts, l.list(a.body[start:], st))
	var h, d float32
	for _, p := range parts {
		h, d = max(h, p.h), max(d, p.d)
	}
	size := l.delimSize(h, d, st)
	out := l.delimiter(a.r, size, st, a.ink)
	for i, p := range parts {
		if i > 0 {
			out.then(l.delimiter(mids[i-1].r, size, st, mids[i-1].ink))
		}
		out.then(p)
	}
	out.then(l.delimiter(a.r2, size, st, a.ink))
	return out
}

func (l *layout) accent(a *atom, st style) box {
	em := l.em(st)
	base := l.list(a.body, st.cramp())
	attach := base.w / 2
	if len(base.items) == 1 && base.items[0].g != nil {
		attach = base.items[0].x + base.items[0].g.accent*em
	}
	acc := l.char(a.r, roman, em, a.ink)
	if a.wide {
		acc = l.stretch(a.r, base.w, false, em, a.ink)
	}
	at := acc.w / 2
	if len(acc.items) == 1 && acc.items[0].g != nil {
		at = acc.items[0].g.accent * em
	}
	shift := max(0, base.h-l.k(accentBaseHeight, st))
	if acc.items[0].sys != nil {
		shift = base.h - 0.45*em
	}
	out := base
	out.add(acc, attach-at, -shift)
	out.h = max(base.h, acc.h+shift)
	return out
}

func (l *layout) bar(a *atom, st style) box {
	k := func(i int) float32 { return l.k(i, st) }
	if a.under {
		out := l.list(a.body, st)
		gap, t := k(underbarVerticalGap), k(underbarRuleThickness)
		out.items = append(out.items, rule(0, out.d+gap, out.w, t, a.ink))
		out.d += gap + t + k(underbarExtraDescender)
		return out
	}
	out := l.list(a.body, st.cramp())
	gap, t := k(overbarVerticalGap), k(overbarRuleThickness)
	out.items = append(out.items, rule(0, -out.h-gap-t, out.w, t, a.ink))
	out.h += gap + t + k(overbarExtraAscender)
	return out
}

func (l *layout) brace(a *atom, st style) box {
	base := l.list(a.body, st)
	b := l.stretch(a.r, base.w, false, l.em(st), a.ink)
	gap := l.k(overbarVerticalGap, st)
	w := max(base.w, b.w)
	out := box{w: w, h: base.h, d: base.d}
	out.add(base, (w-base.w)/2, 0)
	if a.under {
		out.add(b, (w-b.w)/2, base.d+gap+b.h)
		out.d = base.d + gap + b.h + b.d
	} else {
		out.add(b, (w-b.w)/2, -base.h-gap-b.d)
		out.h = base.h + gap + b.d + b.h
	}
	return out
}

// array lays out the cells of an environment in rows and columns:
// centered in a matrix, to the left in cases, and right then left in
// pairs in aligned; centered on the axis.
func (l *layout) array(a *atom, st style) box {
	cs := st
	if a.env == "matrix" || a.env == "cases" {
		cs = max(st, textStyle)
	}
	em := l.em(cs)
	skip := 1.2 * em
	var jot float32
	switch a.env {
	case "cases":
		skip *= 1.2
	case "aligned", "gathered":
		jot = 0.3 * em
	}
	var cols []float32
	cells := make([][]box, len(a.rows))
	hs, ds := make([]float32, len(a.rows)), make([]float32, len(a.rows))
	for i, row := range a.rows {
		hs[i], ds[i] = 0.7*skip, 0.3*skip
		for j, cell := range row {
			if a.env == "aligned" && j%2 == 1 {
				// A {} before the right side of a pair keeps a relation
				// there spaced as one.
				cell = append([]*atom{{kind: group}}, cell...)
			}
			b := l.list(cell, cs)
			cells[i] = append(cells[i], b)
			if j == len(cols) {
				cols = append(cols, 0)
			}
			cols[j] = max(cols[j], b.w)
			hs[i], ds[i] = max(hs[i], b.h), max(ds[i], b.d)
		}
	}
	xs := make([]float32, len(cols))
	var w float32
	for j, cw := range cols {
		if j > 0 {
			switch {
			case a.env != "aligned":
				w += em
			case j%2 == 0:
				w += 2 * em
			}
		}
		xs[j], w = w, w+cw
	}
	var total float32
	for i := range a.rows {
		total += hs[i] + ds[i]
		if i > 0 {
			total += jot
		}
	}
	out := box{w: w, h: total/2 + l.k(axisHeight, st), d: total/2 - l.k(axisHeight, st)}
	y := -out.h
	for i, row := range cells {
		if i > 0 {
			y += jot
		}
		y += hs[i]
		for j, b := range row {
			x := xs[j]
			switch {
			case a.env == "matrix" || a.env == "gathered":
				x += (cols[j] - b.w) / 2
			case a.env == "aligned" && j%2 == 0:
				x += cols[j] - b.w
			}
			out.add(b, x, y)
		}
		y += ds[i]
	}
	return out
}
