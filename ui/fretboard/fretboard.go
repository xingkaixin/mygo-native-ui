// Package fretboard shows a guitar's neck and chord diagrams. The board
// lays the strings of the tuning the app gives across it, the lowest at
// the bottom with its name by the nut, a column for the open strings
// before the nut, and the frets after it, closer together toward the
// body as on a real neck, with inlays at 3, 5, 7, 9, two at 12, and so on
// up, and the frets' numbers below. Marks show as dots, labeled by their
// notes' names or by how far they are above the root, as ♭3 or 5, the
// root's in the accent color; Neck.Marks makes those of a scale or a
// chord in a key. Hovering a fret shows its note; a click plays it, as a
// plucked string, and marks it or takes its mark away, and the Events
// returned report the string and the fret in that frame.
//
//	neck := fretboard.Neck{Tuning: app.tuning}
//	if changed { // the key, the scale or the tuning
//		app.marks = neck.Marks(app.key, fretboard.Scales[app.scale])
//	}
//	if p, ok := fretboard.View(c, neck, &app.marks, fretboard.Intervals).Clicked(); ok {
//		... // neck.Note(p) was played
//	}
//
// Diagram shows a chord as chord sites draw it, from its shape, as
// "x32010" for C: its strings upright and its frets across, an x or an o
// above each string not fretted, the fingers' numbers in the dots, and a
// barre across. A click strums it.
//
//	fretboard.Diagram(c, nil, fretboard.Chord{Name: "F", Shape: "133211"})
package fretboard

import (
	"cmp"
	"math"
	"math/rand/v2"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/egoist/mygo/audio"
	"github.com/egoist/mygo/ui"
)

// Neck is a guitar's neck: the notes its strings sound open, lowest
// first, as MIDI numbers them, 40 for a guitar's low E, and how many
// frets it has. The zero Neck is a guitar's in standard tuning, EADGBE,
// with 15 frets; a bass's strings are {28, 33, 38, 43}.
type Neck struct {
	Tuning []int
	Frets  int
}

// standard is a guitar's standard tuning, E2 A2 D3 G3 B3 E4.
var standard = []int{40, 45, 50, 55, 59, 64}

func (n Neck) tuning() []int {
	if len(n.Tuning) == 0 {
		return standard
	}
	return n.Tuning
}

func (n Neck) frets() int {
	if n.Frets <= 0 {
		return 15
	}
	return n.Frets
}

// Pos is a place on a neck: a string, 0 for the lowest, and a fret, 0
// for the string played open.
type Pos struct{ String, Fret int }

// Note returns the note p sounds on the neck, as MIDI numbers it.
func (n Neck) Note(p Pos) int { return n.tuning()[p.String] + p.Fret }

// Marks is what a board marks: the places with a dot, and the root, a
// pitch class from 0 for C to 11 for B, whose dots are in the accent
// color and which intervals count from. A click on the board adds a
// place or removes it.
type Marks struct {
	Root   int
	Places map[Pos]bool
}

// Formula is a scale or a chord: its name and its notes, in semitones
// above its root.
type Formula struct {
	Name  string
	Steps []int
}

// Scales are the major and natural minor scales, the pentatonics and the
// blues, then the seven modes.
var Scales = []Formula{
	{"Major", []int{0, 2, 4, 5, 7, 9, 11}},
	{"Natural minor", []int{0, 2, 3, 5, 7, 8, 10}},
	{"Major pentatonic", []int{0, 2, 4, 7, 9}},
	{"Minor pentatonic", []int{0, 3, 5, 7, 10}},
	{"Blues", []int{0, 3, 5, 6, 7, 10}},
	{"Ionian", []int{0, 2, 4, 5, 7, 9, 11}},
	{"Dorian", []int{0, 2, 3, 5, 7, 9, 10}},
	{"Phrygian", []int{0, 1, 3, 5, 7, 8, 10}},
	{"Lydian", []int{0, 2, 4, 6, 7, 9, 11}},
	{"Mixolydian", []int{0, 2, 4, 5, 7, 9, 10}},
	{"Aeolian", []int{0, 2, 3, 5, 7, 8, 10}},
	{"Locrian", []int{0, 1, 3, 5, 6, 8, 10}},
}

// Chords are the chords, named by what follows the root in a chord's
// name, as m7 in Am7, and nothing for a major chord.
var Chords = []Formula{
	{"", []int{0, 4, 7}},
	{"m", []int{0, 3, 7}},
	{"7", []int{0, 4, 7, 10}},
	{"maj7", []int{0, 4, 7, 11}},
	{"m7", []int{0, 3, 7, 10}},
	{"dim", []int{0, 3, 6}},
	{"sus2", []int{0, 2, 7}},
	{"sus4", []int{0, 5, 7}},
}

func mod12(n int) int { return (n%12 + 12) % 12 }

// Marks returns the marks of f in key, a pitch class from 0 for C to 11
// for B: every place on the neck that sounds one of its notes, and key
// the root.
func (n Neck) Marks(key int, f Formula) Marks {
	var in [12]bool
	for _, s := range f.Steps {
		in[mod12(key+s)] = true
	}
	m := Marks{Root: mod12(key), Places: map[Pos]bool{}}
	for s, open := range n.tuning() {
		for fret := 0; fret <= n.frets(); fret++ {
			if in[mod12(open+fret)] {
				m.Places[Pos{s, fret}] = true
			}
		}
	}
	return m
}

// Labels is what the dots of a board read.
type Labels uint8

const (
	// Names labels a dot by its note's name, spelled in the root's key:
	// E♭ in C minor, D♯ in E major.
	Names Labels = iota
	// Intervals labels a dot by how far its note is above the root: R
	// for the root, then ♭2, 2, ♭3, 3, 4, ♭5, 5, ♭6, 6, ♭7 and 7.
	Intervals
)

var (
	// naturals are the pitch classes of the letters C to B.
	naturals = [7]int{0, 2, 4, 5, 7, 9, 11}
	// keys spell the roots as key signatures do, by a letter and a sharp
	// (1) or a flat (-1): D♭ rather than C♯, and F♯ rather than G♭.
	keys = [12]struct{ letter, acc int }{
		{0, 0}, {1, -1}, {1, 0}, {2, -1}, {2, 0}, {3, 0}, {3, 1}, {4, 0}, {5, -1}, {5, 0}, {6, -1}, {6, 0},
	}
	// intervals name the notes 0 to 11 semitones above a root, and
	// degrees count the letters each is above it: ♭3 is two, as E♭ is
	// from C.
	intervals = [12]string{"R", "♭2", "2", "♭3", "3", "4", "♭5", "5", "♭6", "6", "♭7", "7"}
	degrees   = [12]int{0, 1, 1, 2, 2, 3, 4, 4, 5, 5, 6, 6}
)

// spell names pitch class pc in the key of root, by the letter its
// interval counts to: E♭ for the ♭3 of C, D♯ for the 7 of E. Where that
// takes two sharps or flats, or makes E♯, B♯, C♭ or F♭, which read as
// mistakes on a fretboard, the plain name stands, flat in a flat key.
func spell(root, pc int) string {
	k := keys[root]
	l := (k.letter + degrees[mod12(pc-root)]) % 7
	acc := (pc-naturals[l]+18)%12 - 6
	if acc < -1 || acc > 1 || acc == 1 && (l == 2 || l == 6) || acc == -1 && (l == 0 || l == 3) {
		if k.acc < 0 || root == 5 {
			l = degrees[pc]
		} else {
			l = [12]int{0, 0, 1, 1, 2, 3, 3, 4, 4, 5, 5, 6}[pc]
		}
		acc = pc - naturals[l]
	}
	return "CDEFGAB"[l:l+1] + [3]string{"♭", "", "♯"}[acc+1]
}

// Events is what the user did to a board in a frame.
type Events struct {
	at      Pos
	clicked bool
}

// Clicked returns the place clicked, in the frame it was: its note has
// played, and its mark has come or gone.
func (e Events) Clicked() (Pos, bool) { return e.at, e.clicked }

// board is where the parts of a board are in its box: the strings'
// names, the column of the open strings from open, the nut from nut, and
// the frets to end, the strings row apart down to h.
type board struct {
	strings, frets int
	row, h         float32
	open, nut      float32
	nutW, end      float32
	dot            float32
}

func layout(t *ui.Theme, strings, frets int, w float32) board {
	b := board{strings: strings, frets: frets, row: t.Space(7), open: t.Space(6), nutW: t.Space(1.5)}
	b.h = float32(strings) * b.row
	b.nut = b.open + t.Space(8)
	b.end = max(b.nut+b.nutW, w-t.Space(1))
	narrowest := b.wire(frets) - b.wire(frets-1)
	b.dot = max(0, min(b.row*0.38, narrowest/2-2))
	return b
}

// wire returns where fret n's wire is: as on a real neck, the nth is
// 1 - 2^(-n/12) of the scale's length from the nut, the last at end.
func (b board) wire(n int) float32 {
	from := func(n int) float64 { return 1 - math.Pow(2, -float64(n)/12) }
	start := b.nut + b.nutW
	return start + (b.end-start)*float32(from(n)/from(b.frets))
}

// x returns the middle of fret n, where its dots go, 0 being the column
// of the open strings.
func (b board) x(n int) float32 {
	if n == 0 {
		return (b.open + b.nut) / 2
	}
	return (b.wire(n-1) + b.wire(n)) / 2
}

// y returns the line of string i.
func (b board) y(i int) float32 { return (float32(b.strings-1-i) + 0.5) * b.row }

// at returns the place at x, y, or false off the strings and the frets.
func (b board) at(x, y float32) (Pos, bool) {
	if y < 0 || y >= b.h || x < b.open || x >= b.end {
		return Pos{}, false
	}
	s := b.strings - 1 - int(y/b.row)
	if x < b.nut+b.nutW {
		return Pos{s, 0}, true
	}
	for n := 1; n <= b.frets; n++ {
		if x < b.wire(n) {
			return Pos{s, n}, true
		}
	}
	return Pos{}, false
}

// dot is a mark to draw: where it is, what it reads, and whether its note
// is the root.
type dot struct {
	at    Pos
	label string
	root  bool
}

// View creates a board of neck, the places in marks marked, their dots
// labeled as labels says, and returns what was clicked. Hovering a fret
// rings it and shows its note above, as E♭3; a click plucks it and adds
// its place to marks, or removes it.
func View(c *ui.Context, neck Neck, marks *Marks, labels Labels) Events {
	t := c.Theme()
	tuning, frets := neck.tuning(), neck.frets()
	root := mod12(marks.Root)
	el := ui.Box(c).Height(float32(len(tuning))*t.Space(7) + t.Space(7)).Cursor(ui.CursorPointer).Label("Fretboard")
	b := layout(t, len(tuning), frets, el.Bounds().W)
	px, py, over := el.PointerPosition()
	hover, on := b.at(px, py)
	on = on && over
	var ev Events
	if el.Clicked() && on {
		if marks.Places == nil {
			marks.Places = map[Pos]bool{}
		}
		if marks.Places[hover] {
			delete(marks.Places, hover)
		} else {
			marks.Places[hover] = true
		}
		play(neck.Note(hover))
		ev = Events{at: hover, clicked: true}
	}
	var dots []dot
	for p, ok := range marks.Places {
		if !ok || p.String < 0 || p.String >= len(tuning) || p.Fret < 0 || p.Fret > frets {
			continue
		}
		pc := mod12(tuning[p.String] + p.Fret)
		label := spell(root, pc)
		if labels == Intervals {
			label = intervals[mod12(pc-root)]
		}
		dots = append(dots, dot{p, label, pc == root})
	}
	names := make([]string, len(tuning))
	for i, n := range tuning {
		names[i] = spell(mod12(n), mod12(n))
	}
	el.Draw(func(p *ui.Painter, r ui.Rect) {
		draw(p, t, r, layout(t, len(tuning), frets, r.W), names, dots, hover, on)
	})
	if on {
		note := neck.Note(hover)
		x, y, d := b.x(hover.Fret), b.y(hover.String), b.dot
		el.Children(func() {
			spot := ui.Box(c).Absolute().Left(x-d).Top(y-d).Size(2*d, 2*d).PassThrough()
			ui.Overlay(c, func() {
				tip := ui.Row(c).Absolute().PassThrough().Role(ui.RoleTooltip).
					AttachTo(spot, ui.AnchorTop, ui.AnchorBottom).Margin(0, 0, t.Space(1.5), 0).
					Padding(t.Space(1), t.Space(2)).Radius(t.Space(1.5)).
					Background(t.Background).Border(1, t.Border).Shadow(0, 2, 8, 0, ui.RGBA(0, 0, 0, 0.15))
				tip.Children(func() {
					ui.Text(c, spell(root, mod12(note))+strconv.Itoa(note/12-1)).PassThrough().Bold().SingleLine()
				})
			})
		})
	}
	return ev
}

func draw(p *ui.Painter, t *ui.Theme, r ui.Rect, b board, names []string, dots []dot, hover Pos, hovering bool) {
	text := func(x, y float32, s string, size float32, weight int, color ui.Color) {
		span := ui.Span{Text: s, Size: size, Weight: weight, Color: color}
		w, h := p.MeasureText(0, span)
		p.RichText(x-w/2, y-h/2, 0, span)
	}
	p.Fill(ui.Rect{X: r.X + b.nut, Y: r.Y, W: b.end - b.nut, H: b.h}, t.Surface, t.Space(1))
	inlay, mid := t.TextMuted.Alpha(0.3), r.Y+b.h/2
	for n := 1; n <= b.frets; n++ {
		x, rad := r.X+b.x(n), b.row*0.12
		ink, weight := t.TextMuted, 400
		switch n % 12 {
		case 3, 5, 7, 9:
			p.Fill(ui.Rect{X: x - rad, Y: mid - rad, W: 2 * rad, H: 2 * rad}, inlay, rad)
			ink, weight = t.Text, 600
		case 0:
			for _, y := range [2]float32{mid - b.row, mid + b.row} {
				p.Fill(ui.Rect{X: x - rad, Y: y - rad, W: 2 * rad, H: 2 * rad}, inlay, rad)
			}
			ink, weight = t.Text, 600
		}
		w := r.X + b.wire(n)
		p.Line(w, r.Y, w, r.Y+b.h, 2, t.TextMuted.Alpha(0.45))
		text(x, r.Y+b.h+t.Space(3.5), strconv.Itoa(n), t.FontSize-2, weight, ink)
	}
	p.Fill(ui.Rect{X: r.X + b.nut, Y: r.Y, W: b.nutW, H: b.h}, t.Text.Alpha(0.6), 1)
	for i, name := range names {
		y := r.Y + b.y(i)
		w := 1 + 1.5*float32(b.strings-1-i)/float32(max(1, b.strings-1))
		p.Line(r.X+b.open, y, r.X+b.end, y, w, t.TextMuted.Alpha(0.85))
		text(r.X+b.open/2, y, name, t.FontSize-1, 600, t.TextMuted)
	}
	spot := func(at Pos) ui.Rect {
		x, y := r.X+b.x(at.Fret), r.Y+b.y(at.String)
		return ui.Rect{X: x - b.dot, Y: y - b.dot, W: 2 * b.dot, H: 2 * b.dot}
	}
	marked := false
	for _, d := range dots {
		face, ink := t.Text, t.Background
		if d.root {
			face, ink = t.Accent, t.AccentText
		}
		s := spot(d.at)
		if hovering && d.at == hover {
			marked = true
			ring := ui.Rect{X: s.X - 3, Y: s.Y - 3, W: s.W + 6, H: s.H + 6}
			p.Stroke(ring, face.Alpha(0.5), ring.W/2, 2)
		}
		p.Fill(s, face, b.dot)
		text(s.X+b.dot, s.Y+b.dot, d.label, min(t.FontSize-2, b.dot), 600, ink)
	}
	if hovering && !marked {
		s := spot(hover)
		p.Fill(s, t.Background, b.dot)
		p.Stroke(s, t.TextMuted, b.dot, 1.5)
	}
}

// Chord is a chord to draw: its name, and its shape as chord sites write
// it, a fret for each string from the lowest, x for a string not played
// and 0 for one played open, as "x32010" for C, or with commas between
// where a fret is above 9, as "x,x,10,12,12,12". Base is the fret a
// shape's 1 is at, as 3 for G's "133211"; 0 or 1 counts the frets from
// the nut.
type Chord struct {
	Name  string
	Shape string
	Base  int
}

// shape is a chord's frets from the nut, -1 for a string not played, the
// finger on each, 1 for the index to 4 for the little finger, 0 for none,
// and the barre, if from is not -1: its fret and the strings it lies
// across.
type shape struct {
	frets, fingers  []int
	barre, from, to int
}

func parse(ch Chord) shape {
	fields := strings.Split(ch.Shape, "")
	if strings.ContainsAny(ch.Shape, ", ") {
		fields = strings.FieldsFunc(ch.Shape, func(r rune) bool { return r == ',' || r == ' ' })
	}
	s := shape{frets: make([]int, len(fields)), fingers: make([]int, len(fields)), from: -1, to: -1}
	low, held := 0, 0
	for i, f := range fields {
		n, err := strconv.Atoi(f)
		if err != nil || n < 0 {
			s.frets[i] = -1
			continue
		}
		if n > 0 {
			n += max(ch.Base, 1) - 1
			held++
			if low == 0 || n < low {
				low = n
			}
		}
		s.frets[i] = n
	}
	// With more strings held than fingers, the index lies across them at
	// the lowest fret, from the first string there to the last, unless
	// one between rings open.
	if held > 4 {
		first, last := slices.Index(s.frets, low), 0
		for i, f := range s.frets {
			if f == low {
				last = i
			}
		}
		if last > first && !slices.Contains(s.frets[first:last+1], 0) {
			s.barre, s.from, s.to = low, first, last
			for i := first; i <= last; i++ {
				if s.frets[i] == low {
					s.fingers[i] = 1
				}
			}
		}
	}
	var rest []int
	for i, f := range s.frets {
		if f > 0 && s.fingers[i] == 0 {
			rest = append(rest, i)
		}
	}
	slices.SortStableFunc(rest, func(a, b int) int { return cmp.Compare(s.frets[a], s.frets[b]) })
	first := 1
	if s.from >= 0 {
		first = 2
	}
	// The other fingers go in order of fret, each a finger further on for
	// each fret further from the lowest, as a hand lies along the neck;
	// where that runs past the little finger, one after another; and
	// where even that does, the dots go without numbers.
	next := first
	for _, i := range rest {
		f := max(next, 1+s.frets[i]-low)
		s.fingers[i], next = f, f+1
	}
	if next > 5 {
		for k, i := range rest {
			s.fingers[i] = first + k
		}
		if first+len(rest) > 5 {
			for _, i := range rest {
				s.fingers[i] = 0
			}
		}
	}
	return s
}

// Diagram creates the diagram of ch on strings tuned to tuning, lowest
// first as MIDI numbers them, nil for standard: its name above, its
// strings upright, the lowest on the left, and four frets across, or as
// many as it spans. The nut is at the top, or else the number of the top
// fret shows beside it, as 5fr. Above each string an x shows it is not
// played and an o that it rings open; dots on the others hold the number
// of the finger that frets them, 1 for the index, worked out from the
// shape, and a bar lies across the strings the index holds down
// together. A click, Enter or Space strums the strings played, lowest
// first, if the shape has as many strings as the tuning.
func Diagram(c *ui.Context, tuning []int, ch Chord) ui.Element {
	t := c.Theme()
	s := parse(ch)
	if len(tuning) == 0 {
		tuning = standard
	}
	top, hi := 1, 0
	for _, f := range s.frets {
		hi = max(hi, f)
	}
	switch {
	case ch.Base > 1:
		top = ch.Base
	case hi > 5:
		top = hi
		for _, f := range s.frets {
			if f > 0 {
				top = min(top, f)
			}
		}
	}
	rows := max(4, hi-top+1)
	// Room on the left for the top fret's number, and on the right for
	// the dots and the barre past the last string.
	sp, fh, left, right, above := t.Space(4), t.Space(5), t.Space(8), t.Space(3), t.Space(5)
	gw := float32(max(len(s.frets)-1, 0)) * sp
	h := above + float32(rows)*fh + t.Space(1)
	el := ui.ButtonBase(c).Padding(t.Space(2)).Radius(t.Space(2)).Label(ch.Name + " chord")
	if el.Clicked() && len(tuning) == len(s.frets) {
		var notes []int
		for i, f := range s.frets {
			if f >= 0 {
				notes = append(notes, tuning[i]+f)
			}
		}
		strum(notes)
	}
	switch {
	case el.Pressed():
		el.Background(t.SurfacePressed)
	case el.Hovered():
		el.Background(t.Surface)
	}
	el.Children(func() {
		ui.Column(c).Gap(t.Space(1)).AlignItems(ui.Center).Children(func() {
			// The margin centers the name over the strings rather than the box.
			ui.Text(c, ch.Name).Bold().FontSize(t.FontSize + 2).SingleLine().MarginStart(left - right)
			ui.Box(c).Size(left+gw+right, h).Draw(func(p *ui.Painter, r ui.Rect) {
				drawChord(p, t, ui.Rect{X: r.X + left, Y: r.Y + above, W: gw, H: float32(rows) * fh}, s, top, rows)
			})
		})
	})
	return el
}

// drawChord draws shape s in grid, the box of its strings and its rows
// of frets, the top one top.
func drawChord(p *ui.Painter, t *ui.Theme, grid ui.Rect, s shape, top, rows int) {
	n := len(s.frets)
	if n == 0 {
		return
	}
	sp, fh := grid.W/float32(max(n-1, 1)), grid.H/float32(rows)
	rad := min(sp, fh) * 0.44
	line := t.TextMuted.Alpha(0.7)
	x := func(i int) float32 { return grid.X + float32(i)*sp }
	y := func(f int) float32 { return grid.Y + (float32(f-top)+0.5)*fh }
	text := func(x, y float32, s string, size float32, weight int, color ui.Color) (w float32) {
		span := ui.Span{Text: s, Size: size, Weight: weight, Color: color}
		w, h := p.MeasureText(0, span)
		p.RichText(x-w/2, y-h/2, 0, span)
		return w
	}
	for k := 0; k <= rows; k++ {
		ly := grid.Y + float32(k)*fh
		p.Line(grid.X, ly, grid.X+grid.W, ly, 1, line)
	}
	for i := range n {
		p.Line(x(i), grid.Y, x(i), grid.Y+grid.H, 1, line)
	}
	if top == 1 {
		p.Fill(ui.Rect{X: grid.X - 0.5, Y: grid.Y - 3.5, W: grid.W + 1, H: 4}, t.Text, 1)
	} else {
		label := strconv.Itoa(top) + "fr"
		span := ui.Span{Text: label, Size: t.FontSize - 3, Color: t.TextMuted}
		w, h := p.MeasureText(0, span)
		p.RichText(grid.X-rad-t.Space(1)-w, y(top)-h/2, 0, span)
	}
	mark, size := grid.Y-t.Space(2.5), t.Space(1.25)
	for i, f := range s.frets {
		switch f {
		case -1:
			var cross ui.Path
			cross.MoveTo(x(i)-size, mark-size).LineTo(x(i)+size, mark+size).
				MoveTo(x(i)-size, mark+size).LineTo(x(i)+size, mark-size)
			p.StrokePath(&cross, 1.5, t.TextMuted)
		case 0:
			p.Stroke(ui.Rect{X: x(i) - size - 0.5, Y: mark - size - 0.5, W: 2*size + 1, H: 2*size + 1}, t.Text, size+0.5, 1.5)
		}
	}
	if s.from >= 0 {
		by := y(s.barre)
		p.Fill(ui.Rect{X: x(s.from) - rad, Y: by - rad, W: x(s.to) - x(s.from) + 2*rad, H: 2 * rad}, t.Text, rad)
		text(x(s.from), by, "1", t.FontSize-3, 700, t.Background)
	}
	for i, f := range s.frets {
		if f <= 0 || s.from >= 0 && f == s.barre && i >= s.from && i <= s.to {
			continue
		}
		p.Fill(ui.Rect{X: x(i) - rad, Y: y(f) - rad, W: 2 * rad, H: 2 * rad}, t.Text, rad)
		if s.fingers[i] > 0 {
			text(x(i), y(f), strconv.Itoa(s.fingers[i]), t.FontSize-3, 700, t.Background)
		}
	}
}

// rate is the sample rate of the notes played.
const rate = 32000

var plucks [128]struct {
	once    sync.Once
	samples []float32
}

// pluck returns a plucked string's note, as MIDI numbers it, made the
// first time: Karplus and Strong's string, a burst of noise going round a
// loop as long as the note's period, each sample averaged with the one
// before it as it goes round, so that the highs die first, as on a real
// string.
func pluck(note int) []float32 {
	e := &plucks[note]
	e.once.Do(func() {
		f := 440 * math.Pow(2, float64(note-69)/12)
		period := rate / f
		// High notes go round the loop so often that an even average
		// would silence them within a second: above 200 Hz the sample
		// before weighs less.
		w := 0.5 * min(1, 200/f)
		// The average delays the loop w of a sample, and an all-pass
		// filter the fraction of a sample left, without which the high
		// notes would sound out of tune.
		n := max(2, int(period-w-0.1))
		frac := period - w - float64(n)
		a := (1 - frac) / (1 + frac)
		// Each time round the loop loses what fades a note 60 dB in 4
		// seconds, on top of the average's loss.
		loss := math.Pow(0.001, 1/(4*f))
		loop := make([]float64, n)
		rng := rand.New(rand.NewPCG(uint64(note), 1))
		prev, mean := 0.0, 0.0
		for i := range loop {
			v := rng.Float64()*2 - 1
			// Noise softened as a pick softens it, its mean taken out so
			// that the note rings around zero.
			loop[i], prev = (v+prev)/2, v
			mean += loop[i]
		}
		for i := range loop {
			loop[i] -= mean / float64(n)
		}
		out := make([]float64, 2*rate)
		var last, x1, y1 float64
		for i := range out {
			j := i % n
			v := loop[j]
			avg := loss * ((1-w)*v + w*last)
			last = v
			y := a*avg + x1 - a*y1
			x1, y1 = avg, y
			loop[j] = y
			out[i] = v
		}
		peak := 0.0
		for _, v := range out {
			peak = max(peak, math.Abs(v))
		}
		e.samples = make([]float32, len(out))
		for i, v := range out {
			// 2 ms in and 300 ms out, so that neither end clicks.
			g := min(1, float64(i)/(0.002*rate), float64(len(out)-i)/(0.3*rate))
			e.samples[i] = float32(0.3 * g * v / peak)
		}
	})
	return e.samples
}

// play plucks note, as MIDI numbers it.
func play(note int) {
	if note >= 0 && note < len(plucks) {
		audio.NewSound(rate, 1, pluck(note)).Play()
	}
}

// strum plucks notes one after another, 25 ms apart, as a pick sweeping
// down the strings.
func strum(notes []int) {
	const gap = rate / 40
	var mix []float32
	for k, note := range notes {
		if note < 0 || note >= len(plucks) {
			continue
		}
		s := pluck(note)
		if end := k*gap + len(s); end > len(mix) {
			mix = append(mix, make([]float32, end-len(mix))...)
		}
		for i, v := range s {
			mix[k*gap+i] += v
		}
	}
	var peak float32
	for _, v := range mix {
		peak = max(peak, v, -v)
	}
	if peak == 0 {
		return
	}
	g := min(1, 0.5/peak)
	for i := range mix {
		mix[i] *= g
	}
	audio.NewSound(rate, 1, mix).Play()
}
