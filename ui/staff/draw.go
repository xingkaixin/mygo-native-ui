package staff

import (
	"slices"

	"github.com/egoist/mygo/ui"
)

// Where the key's sharps and flats sit in the treble clef, in steps from
// the bottom line; the bass clef has them two steps lower.
var (
	sharpPos = [7]int{8, 5, 9, 6, 3, 7, 4}
	flatPos  = [7]int{4, 7, 3, 6, 2, 5, 1}
)

// paint draws the score with its top left at ox, oy, in the colors of t:
// the events in lit in the accent, and event chosen on a tint.
func (sc *score) paint(p *ui.Painter, t *ui.Theme, ox, oy float32, lit map[int]bool, chosen int) {
	sp := sc.sp
	tn := sc.tn
	ink := t.Text
	// Opaque, so that bars and ledger lines crossing them show no seams.
	lines := t.Background.Mix(t.Text, 0.55)
	var path ui.Path
	fill := func(c ui.Color, x, y float32, f func(q pen)) {
		path.Reset()
		f(pen{&path, x, y, sp})
		p.FillPath(&path, c)
	}
	rect := func(c ui.Color, x0, y0, x1, y1 float32) {
		p.Fill(ui.Rect{X: min(x0, x1), Y: min(y0, y1), W: abs32(x1 - x0), H: abs32(y1 - y0)}, c, 0)
	}

	if chosen >= 0 && chosen < len(sc.evs) {
		e := sc.evs[chosen]
		y := oy + sc.systems[e.sys].y
		pad := 0.35 * sp
		p.Fill(ui.Rect{X: ox + e.x - e.l - pad, Y: y + e.y0 - pad, W: e.l + e.r + 2*pad, H: e.y1 - e.y0 + 2*pad}, t.Accent.Alpha(0.16), 0.6*sp)
	}
	// The title centers over the music, the composer ends at its right;
	// either wraps from the left when it does not fit.
	pad, wide := sc.pad, sc.w-2*sc.pad
	if sc.titleY >= 0 {
		span := ui.Span{Text: tn.title, Size: t.FontSize + 7, Weight: 700, Color: ink}
		w, _ := p.MeasureText(0, span)
		p.RichText(ox+max(pad, (sc.w-w)/2), oy+sc.titleY, wide, span)
	}
	if sc.composerY >= 0 {
		span := ui.Span{Text: tn.composer, Size: t.FontSize - 1, Color: t.TextMuted}
		w, _ := p.MeasureText(0, span)
		p.RichText(ox+max(pad, sc.w-pad-w), oy+sc.composerY, wide, span)
	}
	if tn.tempo && len(sc.systems) > 0 {
		sc.paintTempo(p, ink, ox+sc.tempoX, oy+sc.tempoY)
	}

	base := tn.base()
	for _, s := range sc.systems {
		y := oy + s.y
		x0, x1 := ox+s.x0, ox+s.x1
		for i := range 5 {
			rect(lines, x0, y+float32(i)*sp-0.05*sp, x1, y+float32(i)*sp+0.05*sp)
		}
		if tn.bass {
			fill(ink, ox+s.clef, y+sp, bassClef)
		} else {
			fill(ink, ox+s.clef, y+3*sp, trebleClef)
		}
		shift := 0
		if tn.bass {
			shift = -2
		}
		for i := range abs(tn.fifths) {
			a, pos, adv := sharp, sharpPos[i], float32(1.05)
			if tn.fifths < 0 {
				a, pos, adv = flat, flatPos[i], 0.9
			}
			fill(ink, ox+s.key+float32(i)*adv*sp, y+sc.y(pos+shift), func(q pen) { drawAccidental(q, a) })
		}
		if s.showTime {
			if tn.meter.symbol != 0 {
				fill(ink, ox+s.time, y+2*sp, func(q pen) { commonTime(q, tn.meter.symbol == '|') })
			} else {
				w := max(sc.upper.w, sc.lower.w)
				p.Glyphs(sc.upper.glyphs, ox+s.time+(w-sc.upper.w)/2, y+2*sp, ink)
				p.Glyphs(sc.lower.glyphs, ox+s.time+(w-sc.lower.w)/2, y+4*sp, ink)
			}
		}
		for _, b := range s.bars {
			parts, _ := sc.parts(b.kind)
			for _, pt := range parts {
				x := ox + b.x + pt.x
				switch pt.kind {
				case '|':
					rect(ink, x, y, x+thinW*sp, y+4*sp)
				case 'T':
					rect(ink, x, y, x+thickW*sp, y+4*sp)
				case ':':
					path.Reset()
					path.Circle(x+0.2*sp, y+1.5*sp, 0.2*sp)
					path.Circle(x+0.2*sp, y+2.5*sp, 0.2*sp)
					p.FillPath(&path, ink)
				}
			}
		}
	}

	for i, e := range tn.events {
		pl := sc.evs[i]
		c := ink
		if lit[i] {
			c = t.Accent
		}
		y := oy + sc.systems[pl.sys].y
		x := ox + pl.x
		if len(e.heads) == 0 {
			fill(c, x, y+2*sp, func(q pen) { rest(q, pl.kind) })
			for d := range pl.dots {
				path.Reset()
				path.Circle(x+pl.dot+0.2*sp+float32(d)*0.5*sp, y+sc.y(5), 0.2*sp)
				p.FillPath(&path, c)
			}
			continue
		}
		w := sc.headW(pl.kind)
		lo, hi := e.heads[0].step-base, e.heads[len(e.heads)-1].step-base
		lx0, lx1 := x+slices.Min(pl.dx)-0.4*sp, x+slices.Max(pl.dx)+w+0.4*sp
		for pos := -2; pos >= lo; pos -= 2 {
			rect(lines, lx0, y+sc.y(pos)-0.08*sp, lx1, y+sc.y(pos)+0.08*sp)
		}
		for pos := 10; pos <= hi; pos += 2 {
			rect(lines, lx0, y+sc.y(pos)-0.08*sp, lx1, y+sc.y(pos)+0.08*sp)
		}
		dots := map[int]bool{}
		for k, h := range e.heads {
			pos := h.step - base
			hy := y + sc.y(pos)
			fill(c, x+pl.dx[k], hy, func(q pen) { noteHead(q, pl.kind) })
			if h.acc != noAcc {
				fill(c, x+pl.acc[k], hy, func(q pen) { drawAccidental(q, h.acc) })
			}
			if pos%2 == 0 {
				pos++
			}
			if pl.dots > 0 && !dots[pos] {
				dots[pos] = true
				for d := range pl.dots {
					path.Reset()
					path.Circle(x+pl.dot+0.2*sp+float32(d)*0.5*sp, y+sc.y(pos), 0.2*sp)
					p.FillPath(&path, c)
				}
			}
		}
		if !pl.stem {
			continue
		}
		sx := x + pl.stemX
		rect(c, sx-stemW*sp/2, y+pl.root, sx+stemW*sp/2, y+pl.tip)
		if !pl.beamed {
			for k := range beams(pl.kind) {
				fill(c, sx-stemW*sp/2, y+pl.tip, func(q pen) { flag(q, k, pl.up) })
			}
		}
	}

	for _, b := range sc.beams {
		y := oy + sc.systems[b.sys].y
		d := float32(1)
		if !b.up {
			d = -1
		}
		stems := make([]float32, len(b.evs))
		for k, i := range b.evs {
			stems[k] = sc.evs[i].x + sc.evs[i].stemX
		}
		bar := func(xa, xb float32, level int) {
			off := d * float32(level) * beamGap * sp
			ya, yb := y+b.at(xa)+off, y+b.at(xb)+off
			path.Reset()
			q := pen{&path, ox, 0, 1}
			q.quad(xa, ya, xb, yb, xb, yb+d*beamH*sp, xa, ya+d*beamH*sp)
			p.FillPath(&path, ink)
		}
		half := stemW * sp / 2
		bar(stems[0]-half, stems[len(stems)-1]+half, 0)
		for level := 1; level < 4; level++ {
			for k := 0; k < len(b.evs); {
				if beams(sc.evs[b.evs[k]].kind) <= level {
					k++
					continue
				}
				j := k
				for j+1 < len(b.evs) && beams(sc.evs[b.evs[j+1]].kind) > level {
					j++
				}
				switch {
				case j > k:
					bar(stems[k]-half, stems[j]+half, level)
				case k == 0:
					bar(stems[k]-half, stems[k]+min(1.1*sp, (stems[1]-stems[0])/2), level)
				default:
					bar(stems[k]-min(1.1*sp, (stems[k]-stems[k-1])/2), stems[k]+half, level)
				}
				k = j + 1
			}
		}
	}

	for _, tm := range sc.ties {
		y := oy + sc.systems[tm.sys].y
		path.Reset()
		tie(pen{&path, ox, y, 1}, tm.x0, tm.x1, tm.y, tm.h, 0.18*sp)
		p.FillPath(&path, ink)
	}
	for _, tm := range sc.tuplets {
		y := oy + sc.systems[tm.sys].y + tm.y
		x0, x1 := ox+tm.x0, ox+tm.x1
		mid := (x0 + x1) / 2
		if !tm.bracket {
			p.Glyphs(sc.three.glyphs, mid-sc.three.w/2, y, ink)
			continue
		}
		hook := 0.6 * sp
		if !tm.up {
			hook = -hook
		}
		gap := sc.three.w/2 + 0.3*sp
		lw := 0.1 * sp
		rect(ink, x0, y, mid-gap, y+lw)
		rect(ink, mid+gap, y, x1, y+lw)
		rect(ink, x0, y, x0+lw, y+hook)
		rect(ink, x1-lw, y, x1, y+hook)
		p.Glyphs(sc.three.glyphs, mid-sc.three.w/2, y+0.5*sp, ink)
	}
}

// paintTempo draws the tempo's beat as a small note, = and the beats a
// minute, on the baseline y.
func (sc *score) paintTempo(p *ui.Painter, c ui.Color, x, y float32) {
	sp := sc.sp * 0.75
	kind, dots, ok := value(sc.tn.beat)
	if !ok {
		kind, dots = 4, 0
	}
	var path ui.Path
	hy := y - 0.45*sp
	q := pen{&path, x, hy, sp}
	noteHead(q, kind)
	w := hw * sp
	if kind == 1 {
		w = ww * sp
	}
	if kind > 1 {
		q.quad(hw-stemW, 0, hw, 0, hw, -2.9, hw-stemW, -2.9)
		for k := range beams(kind) {
			flag(pen{&path, x + w - stemW*sp, hy - 2.9*sp, sp}, k, true)
		}
	}
	for d := range dots {
		path.Circle(x+w+0.45*sp+float32(d)*0.5*sp, hy, 0.2*sp)
	}
	p.FillPath(&path, c)
	tx := x + w + 0.6*sp + float32(dots)*0.5*sp
	if kind >= 8 {
		tx += 0.8 * sp
	}
	p.Glyphs(sc.tempo.glyphs, tx, y, c)
}

func abs32(v float32) float32 { return max(v, -v) }
