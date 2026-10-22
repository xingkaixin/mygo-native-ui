package staff

import (
	"math"
	"slices"
	"strconv"

	"github.com/egoist/mygo/ui"
)

// Sizes in staff spaces.
const (
	stemW  = 0.13
	thinW  = 0.16
	thickW = 0.5
	beamH  = 0.5
	// beamGap is from one beam to the next.
	beamGap = 0.75
)

// placed is where an event is engraved: its system, the left of its
// column of heads, and the rest in DIPs from that left, y from the top
// line.
type placed struct {
	sys        int
	x          float32
	kind, dots int
	stem, up   bool
	beamed     bool
	// dx are the heads' offsets, a second apart sitting beside the
	// column; acc the left of each head's accidental.
	dx, acc []float32
	dot     float32
	// l and r are how far its ink reaches to the left and the right.
	l, r  float32
	stemX float32
	// root is where the stem leaves the heads, tip where it ends.
	root, tip float32
	// y0 and y1 are the top and the bottom of its ink.
	y0, y1 float32
}

type sysBar struct {
	x    float32
	kind bar
}

// system is a line of the score: its top line's y, where its staff
// starts and ends, its clef, key and time, its bars and its events.
type system struct {
	y, x0, x1       float32
	clef, key, time float32
	showTime        bool
	bars            []sysBar
	from, to        int
	top, bottom     float32
}

// beam joins the stems of events, its outer edge on y0 at x0 and rising
// slope a DIP across.
type beam struct {
	sys       int
	evs       []int
	up        bool
	x0, y0, k float32
}

func (b beam) at(x float32) float32 { return b.y0 + b.k*(x-b.x0) }

type tieMark struct {
	sys          int
	x0, x1, y, h float32
}

type tupletMark struct {
	sys         int
	x0, x1, y   float32
	up, bracket bool
}

// text is text shaped once, with its width.
type text struct {
	glyphs []ui.Glyph
	w      float32
}

func shape(s string, f ui.Font) text {
	g := ui.Shape(s, f)
	var w float32
	if len(g) > 0 {
		last := g[len(g)-1]
		w = last.X + last.Advance
	}
	return text{g, w}
}

// score is a tune engraved to a width, in DIPs from its top left.
type score struct {
	sp, w, h, pad     float32
	titleY, composerY float32
	tempoX, tempoY    float32
	systems           []system
	evs               []placed
	beams             []beam
	ties              []tieMark
	tuplets           []tupletMark
	upper, lower      text
	three, tempo      text
	tn                *tune
}

// base is the step of the staff's bottom line: E4 in the treble clef,
// G2 in the bass.
func (t *tune) base() int {
	if t.bass {
		return 18
	}
	return 30
}

// y returns the y of pos, in steps from the bottom line, below the top
// line.
func (sc *score) y(pos int) float32 { return float32(8-pos) * sc.sp / 2 }

func (sc *score) headW(kind int) float32 {
	if kind == 1 {
		return ww * sc.sp
	}
	return hw * sc.sp
}

// ideal is the room a note of ticks takes, longer notes more but not in
// proportion, as engravers space them.
func ideal(ticks int) float32 {
	q := float64(ticks) / (whole / 4)
	return float32(max(1.8, 3.5+1.15*math.Log2(q)))
}

type barPart struct {
	kind byte // '|' thin, 'T' thick, ':' dots
	x    float32
}

// parts returns the lines and dots of b, from its left, and its width.
func (sc *score) parts(b bar) ([]barPart, float32) {
	seq := map[bar]string{thin: "|", double: "||", final: "|T", heavy: "T|", repStart: "T|:", repEnd: ":|T", repBoth: ":|T|:"}[b]
	var out []barPart
	var x float32
	for i := range len(seq) {
		if i > 0 {
			x += 0.35 * sc.sp
		}
		out = append(out, barPart{seq[i], x})
		x += map[byte]float32{'|': thinW, 'T': thickW, ':': 0.4}[seq[i]] * sc.sp
	}
	return out, x
}

func (sc *score) barW(b bar) float32 {
	_, w := sc.parts(b)
	return w
}

// measureW is a measure's natural room: what comes before its first
// note, the room after each note, and its width with the bar closing it.
func (sc *score) measureW(m measure, open, close bar) (fixed float32, segs []float32, w float32) {
	sp := sc.sp
	fixed = sp
	if open != noBar {
		fixed = sc.barW(open) + 1.1*sp
	}
	evs := sc.evs[m.from:m.to]
	x := fixed + evs[0].l
	for i, e := range evs {
		var s float32
		if i+1 < len(evs) {
			s = max(ideal(sc.tn.events[m.from+i].ticks)*sp, e.r+evs[i+1].l+0.6*sp)
		} else {
			s = max(ideal(sc.tn.events[m.from+i].ticks)*sp, e.r+1.2*sp)
		}
		segs = append(segs, s)
		x += s
	}
	return fixed + evs[0].l, segs, x + sc.barW(close)
}

// engrave lays tn out on systems width wide, sp DIPs a staff space.
func engrave(c *ui.Context, tn *tune, width float32) *score {
	t := c.Theme()
	sp := t.Space(2.5)
	sc := &score{sp: sp, w: width, tn: tn, titleY: -1, composerY: -1}
	sc.evs = make([]placed, len(tn.events))
	base := tn.base()

	for i, e := range tn.events {
		p := &sc.evs[i]
		p.kind, p.dots, _ = value(e.written)
		p.stem = len(e.heads) > 0 && p.kind > 1
		if len(e.heads) > 0 {
			lo, hi := e.heads[0].step-base, e.heads[len(e.heads)-1].step-base
			p.up = 4-lo > hi-4
		}
	}
	measureOf := make([]int, len(tn.events))
	for mi, m := range tn.measures {
		for i := m.from; i < m.to; i++ {
			measureOf[i] = mi
		}
	}
	beamable := func(i int) bool { return len(tn.events[i].heads) > 0 && sc.evs[i].kind >= 8 }
	for i := 0; i < len(tn.events); {
		if !beamable(i) {
			i++
			continue
		}
		j := i + 1
		for j < len(tn.events) && beamable(j) && tn.events[j].joined && measureOf[j] == measureOf[i] {
			j++
		}
		if j-i >= 2 {
			lo, hi := 99, -99
			for k := i; k < j; k++ {
				hs := tn.events[k].heads
				lo, hi = min(lo, hs[0].step-base), max(hi, hs[len(hs)-1].step-base)
			}
			up := 4-lo > hi-4
			b := beam{up: up}
			for k := i; k < j; k++ {
				sc.evs[k].up, sc.evs[k].beamed = up, true
				b.evs = append(b.evs, k)
			}
			sc.beams = append(sc.beams, b)
		}
		i = j
	}
	for i, e := range tn.events {
		sc.shapeEvent(&sc.evs[i], e, base)
	}

	header := func(first bool) (clef, key, time, end float32) {
		clef = 0.5 * sp
		key = clef + 3.2*sp
		adv := float32(1.05)
		if tn.fifths < 0 {
			adv = 0.9
		}
		n := float32(abs(tn.fifths))
		time = key + n*adv*sp
		if n > 0 {
			time += 0.6 * sp
		}
		end = time
		if first && tn.meter.num > 0 {
			end = time + max(sc.upper.w, sc.lower.w)
			if tn.meter.symbol != 0 {
				end = time + 2.1*sp
			}
			end += 0.4 * sp
		}
		end += 0.5 * sp
		return
	}
	digits := ui.Font{Size: 2.8 * sp, Weight: 800}
	sc.upper = shape(strconv.Itoa(tn.meter.num), digits)
	sc.lower = shape(strconv.Itoa(tn.meter.den), digits)
	sc.three = shape("3", ui.Font{Size: 1.5 * sp, Weight: 600, Italic: true})
	if tn.tempo {
		sc.tempo = shape("= "+strconv.Itoa(tn.bpm), ui.Font{Size: t.FontSize, Weight: 600})
	}

	pad := t.Space(5)
	sc.pad = pad
	x0 := pad
	x1 := max(width-pad, x0+12*sp)
	ms := tn.measures
	for m := 0; m < len(ms); {
		s := len(sc.systems)
		clef, key, time, start := header(s == 0)
		sys := system{x0: x0, clef: x0 + clef, key: x0 + key, time: x0 + time, showTime: s == 0 && tn.meter.num > 0}
		avail := x1 - x0 - start
		type fit struct {
			open, close bar
			fixed, w    float32
			segs        []float32
		}
		var row []fit
		var used float32
		for k := m; k < len(ms); k++ {
			open, close := ms[k].open, ms[k].close
			if k == m && k > 0 {
				if pc := ms[k-1].close; pc == repStart || pc == repBoth {
					open = repStart
				}
			}
			fixed, segs, w := sc.measureW(ms[k], open, close)
			if len(row) > 0 && used+w > avail {
				break
			}
			row = append(row, fit{open, close, fixed, w, segs})
			used += w
		}
		last := m+len(row) == len(ms)
		if !last {
			// A repeat starting at the break shows at the start of the
			// next system.
			f := &row[len(row)-1]
			switch f.close {
			case repStart:
				f.close = thin
			case repBoth:
				f.close = repEnd
			}
		}
		var total, natural float32
		for i := range row {
			f := &row[i]
			_, f.segs, f.w = sc.measureW(ms[m+i], f.open, f.close)
			natural += f.w
			for _, g := range f.segs {
				total += g
			}
		}
		k := float32(1)
		if extra := avail - natural; total > 0 && (!last || natural > 0.7*avail || extra < 0) {
			k = 1 + extra/total
		}
		x := x0 + start
		for i, f := range row {
			mm := ms[m+i]
			if f.open != noBar {
				sys.bars = append(sys.bars, sysBar{x, f.open})
			}
			ex := x + f.fixed
			for j := mm.from; j < mm.to; j++ {
				sc.evs[j].sys, sc.evs[j].x = s, ex
				ex += f.segs[j-mm.from] * k
			}
			end := ex + sc.barW(f.close)
			if f.close != noBar {
				sys.bars = append(sys.bars, sysBar{ex, f.close})
			}
			x = end
		}
		sys.x1 = x
		sys.from, sys.to = ms[m].from, ms[m+len(row)-1].to
		sc.systems = append(sc.systems, sys)
		m += len(row)
	}

	for i, e := range tn.events {
		sc.stemOf(&sc.evs[i], e, base)
	}
	for i := range sc.beams {
		sc.setBeam(&sc.beams[i], base)
	}
	sc.tieAll(base)
	sc.tupletAll(base)
	sc.stack(c)
	return sc
}

// shapeEvent sets the heads' offsets, the accidentals' places and the
// ink around p's column of heads.
func (sc *score) shapeEvent(p *placed, e event, base int) {
	sp := sc.sp
	if len(e.heads) == 0 {
		p.r = restWidth(p.kind) * sp
		if p.dots > 0 {
			p.dot = p.r + 0.3*sp
			p.r = p.dot + float32(p.dots)*0.5*sp
		}
		return
	}
	n := len(e.heads)
	w := sc.headW(p.kind)
	side := w - stemW*sp
	if !p.stem {
		side = w
	}
	p.dx = make([]float32, n)
	if p.up || !p.stem {
		for i := 1; i < n; i++ {
			if e.heads[i].step-e.heads[i-1].step <= 1 && p.dx[i-1] == 0 {
				p.dx[i] = side
			}
		}
	} else {
		for i := n - 2; i >= 0; i-- {
			if e.heads[i+1].step-e.heads[i].step <= 1 && p.dx[i+1] == 0 {
				p.dx[i] = -side
			}
		}
	}
	left, right := slices.Min(p.dx), slices.Max(p.dx)+w
	// Accidentals go in columns leftward, top down, each in the first
	// column where it clears those already there by three spaces.
	var cols [][]int
	colOf := make([]int, n)
	for i := n - 1; i >= 0; i-- {
		if e.heads[i].acc == noAcc {
			continue
		}
		c := 0
		for ; c < len(cols); c++ {
			if !slices.ContainsFunc(cols[c], func(j int) bool { return e.heads[j].step-e.heads[i].step < 6 }) {
				break
			}
		}
		if c == len(cols) {
			cols = append(cols, nil)
		}
		cols[c] = append(cols[c], i)
		colOf[i] = c
	}
	p.acc = make([]float32, n)
	edge := left - 0.22*sp
	rights := make([]float32, len(cols))
	for c, col := range cols {
		rights[c] = edge
		var cw float32
		for _, j := range col {
			cw = max(cw, accWidth[e.heads[j].acc]*sp)
		}
		edge -= cw + 0.12*sp
	}
	for i, h := range e.heads {
		if h.acc != noAcc {
			p.acc[i] = rights[colOf[i]] - accWidth[h.acc]*sp
			left = min(left, p.acc[i])
		}
	}
	p.stemX = stemW * sp / 2
	if p.up {
		p.stemX = w - stemW*sp/2
	}
	p.l = -left
	p.r = right
	if p.dots > 0 {
		p.dot = right + 0.35*sp
		p.r = p.dot + float32(p.dots)*0.5*sp
	}
	if p.stem && !p.beamed && p.kind >= 8 {
		p.r = max(p.r, p.stemX+1.1*sp)
	}
}

// stemOf sets where p's stem runs and the top and bottom of its ink.
func (sc *score) stemOf(p *placed, e event, base int) {
	sp := sc.sp
	if len(e.heads) == 0 {
		top := map[int]float32{1: 1, 2: 1.5, 4: 0.5, 8: 1, 16: 1, 32: 0, 64: 0}[p.kind]
		p.y0, p.y1 = top*sp, (top+3.2)*sp
		return
	}
	hi, lo := sc.y(e.heads[len(e.heads)-1].step-base), sc.y(e.heads[0].step-base)
	p.y0, p.y1 = hi-0.6*sp, lo+0.6*sp
	if !p.stem {
		return
	}
	extra := 0.75 * float32(max(0, beams(p.kind)-2)) * sp
	if p.up {
		p.root, p.tip = lo, min(hi-3.5*sp-extra, 2*sp)
	} else {
		p.root, p.tip = hi, max(lo+3.5*sp+extra, 2*sp)
	}
	p.y0, p.y1 = min(p.y0, p.tip), max(p.y1, p.tip)
}

// setBeam slants b with its notes, a space at most, and moves it off
// them so that every stem is long enough.
func (sc *score) setBeam(b *beam, base int) {
	sp := sc.sp
	evs := sc.evs
	first, last := evs[b.evs[0]], evs[b.evs[len(b.evs)-1]]
	b.sys = first.sys
	xa, xb := first.x+first.stemX, last.x+last.stemX
	near := func(i int) float32 {
		hs := sc.tn.events[i].heads
		if b.up {
			return sc.y(hs[len(hs)-1].step - base)
		}
		return sc.y(hs[0].step - base)
	}
	levels := 1
	for _, i := range b.evs {
		levels = max(levels, beams(evs[i].kind))
	}
	ya, yb := near(b.evs[0]), near(b.evs[len(b.evs)-1])
	dy := max(-sp, min(sp, (yb-ya)/2))
	if xb-xa < 3*sp || first.sys != last.sys {
		dy = 0
	}
	b.x0, b.k = xa, 0
	if xb > xa {
		b.k = dy / (xb - xa)
	}
	// The beam clears every note by a stem's length, and reaches the
	// middle line.
	stem := (3.25 + beamGap*float32(levels-1)) * sp
	if b.up {
		b.y0 = float32(math.Inf(1))
		for _, i := range b.evs {
			off := b.k * (evs[i].x + evs[i].stemX - xa)
			b.y0 = min(b.y0, near(i)-stem-off, 2*sp-off)
		}
	} else {
		b.y0 = float32(math.Inf(-1))
		for _, i := range b.evs {
			off := b.k * (evs[i].x + evs[i].stemX - xa)
			b.y0 = max(b.y0, near(i)+stem-off, 2*sp-off)
		}
	}
	for _, i := range b.evs {
		p := &evs[i]
		p.tip = b.at(p.x + p.stemX)
		p.y0, p.y1 = min(p.y0, p.tip), max(p.y1, p.tip)
	}
}

// tieAll adds the ties from each tied head to the head of the event after
// it at its pitch, across a system's end in two halves.
func (sc *score) tieAll(base int) {
	sp := sc.sp
	tn := sc.tn
	for i := 0; i+1 < len(tn.events); i++ {
		e, next := tn.events[i], tn.events[i+1]
		p, q := sc.evs[i], sc.evs[i+1]
		for hi, h := range e.heads {
			if !h.tie {
				continue
			}
			hj := slices.IndexFunc(next.heads, func(g head) bool { return g.pitch() == h.pitch() })
			if hj < 0 {
				continue
			}
			// A note's tie bows away from its stem; a chord's lower notes
			// tie below and its upper ones above.
			below := p.up
			if n := len(e.heads); n > 1 {
				switch {
				case 2*hi < n-1:
					below = true
				case 2*hi > n-1:
					below = false
				}
			}
			dir := float32(-1)
			if below {
				dir = 1
			}
			y := sc.y(h.step-base) + dir*0.45*sp
			x0 := p.x + p.dx[hi] + sc.headW(p.kind) + 0.12*sp
			if p.dots > 0 {
				x0 = p.x + p.dot + float32(p.dots)*0.5*sp
			}
			x1 := q.x + q.dx[hj] - 0.12*sp
			if p.sys == q.sys {
				sc.ties = append(sc.ties, tieMark{p.sys, x0, x1, y, dir * min(0.35*sp+0.04*(x1-x0), 0.9*sp)})
				continue
			}
			end := sc.systems[p.sys].x1
			sc.ties = append(sc.ties, tieMark{p.sys, x0, end - 0.3*sp, y, dir * 0.6 * sp})
			sc.ties = append(sc.ties, tieMark{q.sys, x1 - 2.5*sp, x1, sc.y(next.heads[hj].step-base) + dir*0.45*sp, dir * 0.6 * sp})
		}
	}
}

// tupletAll adds a 3 over each triplet: by its beam when one beam joins
// it, else on a bracket on the side of the stems.
func (sc *score) tupletAll(base int) {
	sp := sc.sp
	for _, tr := range sc.tn.triplets {
		a, b := tr[0], tr[1]
		for b > a && sc.evs[b].sys != sc.evs[a].sys {
			b--
		}
		p, q := sc.evs[a], sc.evs[b]
		up := 0
		for i := a; i <= b; i++ {
			if len(sc.tn.events[i].heads) > 0 && sc.evs[i].up {
				up++
			} else {
				up--
			}
		}
		m := tupletMark{sys: p.sys, x0: p.x - 0.2*sp, x1: q.x + sc.headW(q.kind) + 0.2*sp, up: up >= 0, bracket: true}
		for _, bm := range sc.beams {
			if slices.Contains(bm.evs, a) && slices.Contains(bm.evs, b) {
				m.up, m.bracket = bm.up, false
				m.x0, m.x1 = p.x+p.stemX, q.x+q.stemX
				mid := (m.x0 + m.x1) / 2
				// y is the 3's baseline, clear of the beam.
				if bm.up {
					m.y = bm.at(mid) - 0.4*sp
				} else {
					m.y = bm.at(mid) + 1.5*sp
				}
			}
		}
		if m.bracket {
			if m.up {
				m.y = 0
				for i := a; i <= b; i++ {
					m.y = min(m.y, sc.evs[i].y0)
				}
				m.y -= 0.9 * sp
			} else {
				m.y = 4 * sp
				for i := a; i <= b; i++ {
					m.y = max(m.y, sc.evs[i].y1)
				}
				m.y += 0.9 * sp
			}
		}
		sc.tuplets = append(sc.tuplets, m)
	}
}

// stack puts the title, the composer and the systems one under another,
// each system as far from the next as what sticks out of them needs.
func (sc *score) stack(c *ui.Context) {
	t := c.Theme()
	sp := sc.sp
	tn := sc.tn
	for i := range sc.systems {
		s := &sc.systems[i]
		// The clefs stick out of the staff: the G clef by more.
		top, bottom := float32(0), 4*sp
		if !tn.bass {
			top, bottom = -1.5*sp, 5.6*sp
		}
		for j := s.from; j < s.to; j++ {
			p := sc.evs[j]
			top, bottom = min(top, p.y0), max(bottom, p.y1)
			for _, h := range tn.events[j].heads {
				if h.acc != noAcc {
					y := sc.y(h.step - tn.base())
					top, bottom = min(top, y-1.8*sp), max(bottom, y+1.5*sp)
				}
			}
		}
		for _, b := range sc.beams {
			if b.sys == i {
				for _, j := range b.evs {
					y := sc.evs[j].tip
					top, bottom = min(top, y-beamH*sp), max(bottom, y+beamH*sp)
				}
			}
		}
		for _, tm := range sc.ties {
			if tm.sys == i {
				top, bottom = min(top, tm.y+tm.h), max(bottom, tm.y+tm.h)
			}
		}
		for _, tm := range sc.tuplets {
			if tm.sys == i {
				top, bottom = min(top, tm.y-1.6*sp), max(bottom, tm.y+0.4*sp)
			}
		}
		s.top, s.bottom = top, bottom
	}
	y := sc.pad
	wide := sc.w - 2*sc.pad
	if tn.title != "" {
		sc.titleY = y
		_, h := c.MeasureText(wide, ui.Span{Text: tn.title, Size: t.FontSize + 7, Weight: 700})
		y += h + t.Space(1)
	}
	if tn.composer != "" {
		sc.composerY = y
		_, h := c.MeasureText(wide, ui.Span{Text: tn.composer, Size: t.FontSize - 1})
		y += h
	}
	if tn.title != "" || tn.composer != "" {
		y += t.Space(2)
	}
	for i := range sc.systems {
		s := &sc.systems[i]
		above := max(2.5*sp, -s.top+0.5*sp)
		if i == 0 && tn.tempo {
			y += 2.6 * sp
			sc.tempoX, sc.tempoY = s.x0, y
		}
		if i > 0 {
			y += sp
		}
		s.y = y + above
		y = s.y + max(4*sp+3*sp, s.bottom+0.6*sp)
	}
	sc.h = y + sc.pad
}
