// Package tablature shows guitar tablature and edits it, as tab editors
// do: six lines for the strings, the highest at the top, each note's fret
// on its string at its time, and the bars laid out in lines as wide as the
// view, each line starting with the strings' names. Beneath the strings,
// each beat's length as a stem, eighths and sixteenths beamed within a
// beat. Hammer-ons and pull-offs arc to the note before them, marked h or
// p; slides slant into the note; bends rise above the strings to how far
// they bend; vibrato waves above the note; and a dead note is an x.
//
// A click puts the cursor on a string at a beat, and the arrows move it.
// Typing a fret puts a note there, two digits typed together making one
// fret, and Delete takes it out. H, S, B and V turn a hammer-on or
// pull-off, a slide, a bend and vibrato on and off for the note, and X
// deadens it. With a playhead, the beat playing is lit and a line stands
// at the playhead.
//
//	tablature.View(c, &app.tabState, &app.tab, app.playhead)
package tablature

import (
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
)

// Tech is how a note is played, as flags.
type Tech uint8

const (
	Legato Tech = 1 << iota // a hammer-on or pull-off from the note before
	Slide                   // slid into from the note before
	Bend                    // bent up a whole tone
	Vibrato
	Dead
)

// Note is a fret on a string, the strings numbered from 0 for the highest.
type Note struct {
	String, Fret int
	Tech         Tech
}

// Beat is the notes struck together and how long they last: 1 a whole
// note, 2 a half, 4 a quarter, 8 an eighth and 16 a sixteenth.
type Beat struct {
	Notes  []Note
	Length int
}

// Bar is a bar's beats.
type Bar struct {
	Beats []Beat
}

// Tab is a part for a guitar: its strings' names, the highest first, and
// its bars.
type Tab struct {
	Tuning []string
	Bars   []Bar
}

// State is what a tab keeps: where the cursor is and the digit typed
// last. Keep it in the app's state, one for each tab.
type State struct {
	bar, beat, str int
	typed          time.Time
}

// Cursor returns where the cursor is, in quarter notes from the start.
func (s *State) Cursor(tab *Tab) float64 {
	q := 0.0
	for b, bar := range tab.Bars {
		for i, bt := range bar.Beats {
			if b == s.bar && i == s.beat {
				return q
			}
			q += 4 / float64(max(1, bt.Length))
		}
	}
	return q
}

const typeWindow = 800 * time.Millisecond

var digits = []ui.Key{ui.Key0, ui.Key1, ui.Key2, ui.Key3, ui.Key4, ui.Key5, ui.Key6, ui.Key7, ui.Key8, ui.Key9}

// View creates the tab of *tab, playing at playhead quarter notes from
// the start, or none when it is below 0.
func View(c *ui.Context, s *State, tab *Tab, playhead float64) ui.Element {
	t := c.Theme()
	strs := max(1, len(tab.Tuning))
	if len(tab.Bars) == 0 {
		tab.Bars = []Bar{{Beats: []Beat{{Length: 4}}}}
	}
	s.bar = max(0, min(len(tab.Bars)-1, s.bar))
	s.beat = max(0, min(len(tab.Bars[s.bar].Beats)-1, s.beat))
	s.str = max(0, min(strs-1, s.str))
	now := c.Now()
	e := ui.Box(c).FillWidth().Focusable().FocusRing(false).Label("Tablature")
	changed := false
	move := func(d int) {
		s.beat += d
		for s.beat < 0 && s.bar > 0 {
			s.bar--
			s.beat += len(tab.Bars[s.bar].Beats)
		}
		for s.beat >= len(tab.Bars[s.bar].Beats) && s.bar < len(tab.Bars)-1 {
			s.beat -= len(tab.Bars[s.bar].Beats)
			s.bar++
		}
		s.beat = max(0, min(len(tab.Bars[s.bar].Beats)-1, s.beat))
		s.typed = time.Time{}
	}
	beat := &tab.Bars[s.bar].Beats[s.beat]
	at := slices.IndexFunc(beat.Notes, func(n Note) bool { return n.String == s.str })
	toggle := func(f Tech) {
		if at >= 0 {
			beat.Notes[at].Tech ^= f
			changed = true
		}
	}
	switch {
	case e.Shortcut(0, ui.KeyLeft):
		move(-1)
	case e.Shortcut(0, ui.KeyRight):
		move(1)
	case e.Shortcut(0, ui.KeyUp):
		s.str = max(0, s.str-1)
		s.typed = time.Time{}
	case e.Shortcut(0, ui.KeyDown):
		s.str = min(strs-1, s.str+1)
		s.typed = time.Time{}
	case e.Shortcut(0, ui.KeyDelete), e.Shortcut(0, ui.KeyBackspace):
		if at >= 0 {
			beat.Notes = slices.Delete(beat.Notes, at, at+1)
			changed = true
		}
	case e.Shortcut(0, ui.KeyH):
		toggle(Legato)
	case e.Shortcut(0, ui.KeyS):
		toggle(Slide)
	case e.Shortcut(0, ui.KeyB):
		toggle(Bend)
	case e.Shortcut(0, ui.KeyV):
		toggle(Vibrato)
	case e.Shortcut(0, ui.KeyX):
		toggle(Dead)
	}
	for d, k := range digits {
		if !e.Shortcut(0, k) {
			continue
		}
		fret := d
		if at >= 0 && now.Sub(s.typed) < typeWindow && beat.Notes[at].Fret*10+d <= 24 {
			fret = beat.Notes[at].Fret*10 + d
		}
		if at >= 0 {
			beat.Notes[at].Fret = fret
		} else {
			beat.Notes = append(beat.Notes, Note{String: s.str, Fret: fret})
		}
		s.typed, changed = now, true
	}

	w := e.Bounds().W
	if w <= 0 {
		w = t.Space(160)
	}
	lay := layout(tab, w, t)
	e.Height(lay.height)
	if e.Clicked() {
		px, py, _ := e.PointerPosition()
		if b, i, str, ok := lay.hit(px, py, strs); ok {
			s.bar, s.beat, s.str, s.typed = b, i, str, time.Time{}
			e.SetChanged()
		}
	}
	focused := e.Focused()
	cur := *s
	e.Draw(func(p *ui.Painter, r ui.Rect) {
		draw(p, r, t, tab, lay, cur, focused, playhead)
	})
	if changed {
		e.SetChanged()
	}
	return e
}

const (
	lineGap   = 13
	above     = 30
	below     = 34
	systemGap = 18
	headWidth = 30
	barPad    = 12
)

type placed struct {
	bar  int
	x, w []float32 // each beat's middle and width
	from float32   // the bar's left edge
	to   float32
	line int
}

type laid struct {
	bars   []placed
	height float32
	starts []float32 // each line's top
}

func beatWidth(length int) float32 {
	return 20 + 44*float32(math.Sqrt(4/float64(max(1, length))))
}

// layout lays the bars out in lines of width w.
func layout(tab *Tab, w float32, t *ui.Theme) laid {
	var l laid
	strs := max(1, len(tab.Tuning))
	sysH := above + float32(strs-1)*lineGap + below
	natural := func(b Bar) float32 {
		sum := float32(barPad * 2)
		for _, bt := range b.Beats {
			sum += beatWidth(bt.Length)
		}
		return sum
	}
	avail := w - headWidth
	var lines [][]int
	var cur []int
	used := float32(0)
	for i, b := range tab.Bars {
		nw := natural(b)
		if len(cur) > 0 && used+nw > avail {
			lines = append(lines, cur)
			cur, used = nil, 0
		}
		cur = append(cur, i)
		used += nw
	}
	if len(cur) > 0 {
		lines = append(lines, cur)
	}
	y := float32(0)
	for li, line := range lines {
		total := float32(0)
		for _, i := range line {
			total += natural(tab.Bars[i])
		}
		k := float32(1)
		if li < len(lines)-1 || total > avail*0.7 {
			k = avail / total
		}
		x := float32(headWidth)
		for _, i := range line {
			b := tab.Bars[i]
			pb := placed{bar: i, from: x, line: li}
			bx := x + barPad*k
			for _, bt := range b.Beats {
				bw := beatWidth(bt.Length) * k
				pb.x = append(pb.x, bx+bw/2)
				pb.w = append(pb.w, bw)
				bx += bw
			}
			x += natural(b) * k
			pb.to = x
			l.bars = append(l.bars, pb)
		}
		l.starts = append(l.starts, y)
		y += sysH + systemGap
	}
	l.height = max(sysH, y-systemGap)
	return l
}

// hit returns the bar, beat and string at x, y.
func (l laid) hit(x, y float32, strs int) (int, int, int, bool) {
	for _, pb := range l.bars {
		top := l.starts[pb.line] + above
		if y < top-lineGap || y > top+float32(strs-1)*lineGap+lineGap {
			continue
		}
		for i, mx := range pb.x {
			if x >= mx-pb.w[i]/2 && x < mx+pb.w[i]/2 {
				str := int(math.Round(float64((y - top) / lineGap)))
				return pb.bar, i, max(0, min(strs-1, str)), true
			}
		}
	}
	return 0, 0, 0, false
}

func draw(p *ui.Painter, r ui.Rect, t *ui.Theme, tab *Tab, l laid, cur State, focused bool, playhead float64) {
	strs := max(1, len(tab.Tuning))
	ink, faint := t.Text, t.Text.Alpha(0.35)
	size := t.FontSize - 1
	lineY := func(line, s int) float32 { return r.Y + l.starts[line] + above + float32(s)*lineGap }
	for li := range l.starts {
		y0, y1 := lineY(li, 0), lineY(li, strs-1)
		x1 := r.X
		for _, pb := range l.bars {
			if pb.line == li {
				x1 = r.X + pb.to
			}
		}
		for s := range strs {
			p.Line(r.X+headWidth-4, lineY(li, s), x1, lineY(li, s), 1, faint)
			name := ui.Span{Text: tab.Tuning[s], Size: t.FontSize - 3, Color: t.TextMuted}
			w, h := p.MeasureText(0, name)
			p.RichText(r.X+headWidth-12-w, lineY(li, s)-h/2, 0, name)
		}
		p.Line(r.X+headWidth-4, y0, r.X+headWidth-4, y1, 2, ink)
	}
	// q is where each beat starts, in quarter notes, for the playhead.
	q := 0.0
	type spot struct{ x, y float32 }
	last := map[int]spot{}
	lastFret := map[int]int{}
	for _, pb := range l.bars {
		bar := tab.Bars[pb.bar]
		y0, y1 := lineY(pb.line, 0), lineY(pb.line, strs-1)
		p.Line(r.X+pb.to, y0, r.X+pb.to, y1, 1.5, ink)
		num := ui.Span{Text: strconv.Itoa(pb.bar + 1), Size: t.FontSize - 4, Color: t.TextMuted}
		p.RichText(r.X+pb.from+2, y0-16, 0, num)
		var group []float32
		groupLen, groupEnd := 0, 0.0
		flush := func() {
			beams(p, group, groupLen, y1+8, ink)
			group = nil
		}
		for i, bt := range bar.Beats {
			x := r.X + pb.x[i]
			dur := 4 / float64(max(1, bt.Length))
			playing := playhead >= q && playhead < q+dur
			if playing {
				p.Fill(ui.Rect{X: x - pb.w[i]/2 + 2, Y: y0 - 8, W: pb.w[i] - 4, H: y1 - y0 + 16}, t.Accent.Alpha(0.12), 4)
				f := float32((playhead - q) / dur)
				px := x - pb.w[i]/2 + pb.w[i]*f
				p.Line(px, y0-10, px, y1+10, 2, t.Accent)
			}
			if focused && pb.bar == cur.bar && i == cur.beat {
				cy := lineY(pb.line, cur.str)
				p.Stroke(ui.Rect{X: x - 11, Y: cy - 8, W: 22, H: 16}, t.Accent, 4, 1.5)
			}
			for _, n := range bt.Notes {
				if n.String < 0 || n.String >= strs {
					continue
				}
				y := lineY(pb.line, n.String)
				label := strconv.Itoa(n.Fret)
				if n.Tech&Dead != 0 {
					label = "x"
				}
				col := ink
				if playing {
					col = t.Accent
				}
				span := ui.Span{Text: label, Size: size, Weight: 600, Color: col, Features: "tnum"}
				w, h := p.MeasureText(0, span)
				p.Fill(ui.Rect{X: x - w/2 - 2, Y: y - h/2, W: w + 4, H: h}, t.Background, 2)
				p.RichText(x-w/2, y-h/2, 0, span)
				prev, ok := last[n.String]
				if n.Tech&Legato != 0 && ok && prev.y == y {
					var arc ui.Path
					arc.MoveTo(prev.x+4, y-8)
					arc.QuadTo((prev.x+x)/2, y-20, x-4, y-8)
					p.StrokePath(&arc, 1, ink)
					mark := "h"
					if n.Fret < lastFret[n.String] {
						mark = "p"
					}
					ms := ui.Span{Text: mark, Size: t.FontSize - 4, Italic: true, Color: t.TextMuted}
					mw, mh := p.MeasureText(0, ms)
					p.RichText((prev.x+x)/2-mw/2, y-21-mh, 0, ms)
				}
				if n.Tech&Slide != 0 {
					dy := float32(4)
					if ok && n.Fret < lastFret[n.String] {
						dy = -4
					}
					var slash ui.Path
					slash.MoveTo(x-w/2-12, y+dy)
					slash.LineTo(x-w/2-4, y-dy)
					p.StrokePath(&slash, 1.2, ink)
				}
				if n.Tech&Bend != 0 {
					var bend ui.Path
					top := lineY(pb.line, 0) - 18
					bend.MoveTo(x+w/2+2, y)
					bend.QuadTo(x+w/2+12, y, x+w/2+12, top+6)
					p.StrokePath(&bend, 1.2, ink)
					var head ui.Path
					head.MoveTo(x+w/2+8, top+8)
					head.LineTo(x+w/2+12, top)
					head.LineTo(x+w/2+16, top+8)
					head.Close()
					p.FillPath(&head, ink)
					fs := ui.Span{Text: "full", Size: t.FontSize - 5, Color: t.TextMuted}
					fw, fh := p.MeasureText(0, fs)
					p.RichText(x+w/2+12-fw/2, top-fh-1, 0, fs)
				}
				if n.Tech&Vibrato != 0 {
					var wave ui.Path
					vy := lineY(pb.line, 0) - 12
					wave.MoveTo(x-6, vy)
					for k := 0; k < 4; k++ {
						wx := x - 6 + float32(k)*5
						wave.QuadTo(wx+1.25, vy-3*float32(1-2*(k%2)), wx+2.5, vy)
						wave.QuadTo(wx+3.75, vy+3*float32(1-2*(k%2)), wx+5, vy)
					}
					p.StrokePath(&wave, 1.2, ink)
				}
				last[n.String] = spot{x, y}
				lastFret[n.String] = n.Fret
			}
			// The stem of the beat, and its beams with the others of the
			// same quarter.
			if bt.Length >= 2 && len(bt.Notes) > 0 {
				top, bottom := y1+8, y1+8+float32(below-14)
				if bt.Length == 2 {
					top = bottom - 10
				}
				p.Line(x, top, x, bottom, 1.2, ink)
			}
			start := q
			if bt.Length >= 8 {
				if len(group) > 0 && (math.Floor(start) != math.Floor(groupEnd-1e-9) || start != groupEnd) {
					flush()
				}
				group = append(group, x)
				groupLen = max(groupLen, bt.Length)
				groupEnd = start + dur
			} else {
				flush()
			}
			q += dur
		}
		flush()
	}
}

// beams joins the stems at xs, one beam for eighths and two for
// sixteenths, or flags a lone stem.
func beams(p *ui.Painter, xs []float32, length int, top float32, col ui.Color) {
	if len(xs) == 0 {
		return
	}
	bottom := top + below - 14
	n := 1
	if length >= 16 {
		n = 2
	}
	for k := range n {
		y := bottom - float32(k)*5
		if len(xs) == 1 {
			var flag ui.Path
			flag.MoveTo(xs[0], y)
			flag.LineTo(xs[0]+7, y-6)
			p.StrokePath(&flag, 2, col)
			continue
		}
		p.Fill(ui.Rect{X: xs[0], Y: y - 1.5, W: xs[len(xs)-1] - xs[0], H: 3}, col, 0)
	}
}

// String writes the tab as ASCII tablature, a line for each string.
func (tab *Tab) String() string {
	var out []string
	for s, name := range tab.Tuning {
		line := fmt.Sprintf("%-2s|", name)
		for _, bar := range tab.Bars {
			for _, bt := range bar.Beats {
				cell := "-"
				for _, n := range bt.Notes {
					if n.String != s {
						continue
					}
					cell = strconv.Itoa(n.Fret)
					switch {
					case n.Tech&Dead != 0:
						cell = "x"
					case n.Tech&Bend != 0:
						cell += "b"
					case n.Tech&Vibrato != 0:
						cell += "~"
					}
					if n.Tech&Legato != 0 {
						cell = "h" + cell
					}
					if n.Tech&Slide != 0 {
						cell = "/" + cell
					}
				}
				pad := max(1, 16/max(1, bt.Length))
				line += cell + strings.Repeat("-", max(1, pad+1-len(cell)))
			}
			line += "|"
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}
