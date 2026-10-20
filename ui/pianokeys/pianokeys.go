// Package pianokeys is a piano keyboard to play: the white and black keys
// of a range the app gives, as C3 to C6. Pressing a key plays it, the
// harder the lower on the key it is pressed, and dragging across the keys
// plays each in turn. While the keyboard has the focus, the computer's
// keys play an octave and a half, from A along the middle row for the
// white keys and the row above for the black, to ;, their letters on the
// keys they play; Z and X move them an octave down and up. Keys held are
// lit, and the notes the app gives, as a scale's or a chord's, lit in
// another color; above the keys shows the name of the chord held, such as
// Cmaj7.
//
// The keyboard makes no sound: it reports each note on and off in the
// frame it comes, for the app to play.
//
//	for _, e := range pianokeys.View(c, &app.keys, 48, 84, app.scale) {
//		if e.On {
//			app.synth.On(e.Note, e.Velocity)
//		} else {
//			app.synth.Off(e.Note)
//		}
//	}
package pianokeys

import (
	"maps"
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
)

// Event is a note started or stopped: its key, as MIDI numbers it, 60 for
// middle C, and for a note on how hard it was pressed, from 1 to 127.
type Event struct {
	On       bool
	Note     int
	Velocity int
}

// typed is the velocity of the notes the computer's keys play.
const typed = 100

// State is what a keyboard keeps: the notes held, by the pointer and by
// the computer's keys, and the octave those keys play. Keep it in the
// app's state, one for each keyboard.
type State struct {
	// held counts what holds each note: the pointer and the keys.
	held map[int]int
	keys map[ui.Key]int
	// pointer is the note the pointer holds, plus 1, so that the zero
	// State holds none.
	pointer int
	octave  int
	events  []Event
	ref     ui.Handle
}

func (s *State) down(note, velocity int) {
	if s.held[note] == 0 {
		s.events = append(s.events, Event{On: true, Note: note, Velocity: velocity})
	}
	s.held[note]++
}

func (s *State) up(note int) {
	if s.held[note] == 0 {
		return
	}
	if s.held[note]--; s.held[note] > 0 {
		return
	}
	delete(s.held, note)
	s.events = append(s.events, Event{Note: note})
}

type letter struct {
	key   ui.Key
	label string
}

// letters are the computer's keys by their place, a semitone apart from
// C, as trackers lay them out.
var letters = [...]letter{
	{ui.KeyA, "A"}, {ui.KeyW, "W"}, {ui.KeyS, "S"}, {ui.KeyE, "E"}, {ui.KeyD, "D"}, {ui.KeyF, "F"},
	{ui.KeyT, "T"}, {ui.KeyG, "G"}, {ui.KeyY, "Y"}, {ui.KeyH, "H"}, {ui.KeyU, "U"}, {ui.KeyJ, "J"},
	{ui.KeyK, "K"}, {ui.KeyO, "O"}, {ui.KeyL, "L"}, {ui.KeyP, "P"}, {ui.KeySemicolon, ";"},
}

// base returns the C the computer's keys start from: the C nearest the
// middle of lo to hi, moved by the octave chosen, as far as the keys
// still play in the range.
func (s *State) base(lo, hi int) int {
	first := lo / 12 * 12
	last := max(first, (hi-12)/12*12)
	return max(first, min(last, (lo+hi)/24*12+12*s.octave))
}

func (s *State) input(c *ui.Context, ev ui.InputEvent, lo, hi int) bool {
	switch ev.Kind {
	case ui.InputPointerDown, ui.InputPointerMove:
		if ev.Button != 0 {
			return false
		}
		r := s.ref.Bounds(c)
		note, velocity := layout(lo, hi, r.W, r.H).at(ev.X, ev.Y)
		if note+1 != s.pointer {
			if s.pointer > 0 {
				s.up(s.pointer - 1)
			}
			s.pointer = note + 1
			if note >= 0 {
				s.down(note, velocity)
			}
		}
		return true
	case ui.InputPointerUp:
		if ev.Button != 0 {
			return false
		}
		if s.pointer > 0 {
			s.up(s.pointer - 1)
			s.pointer = 0
		}
		return true
	case ui.InputKeyDown, ui.InputKeyUp:
		k := ev.PhysicalKey
		// Where the platform does not report the key's place, the key
		// typed stands for it.
		if k == ui.KeyUnknown {
			k = ev.Key
		}
		if ev.Kind == ui.InputKeyUp {
			note, ok := s.keys[k]
			if ok {
				delete(s.keys, k)
				s.up(note)
			}
			return ok
		}
		if ev.Mods&(ui.Super|ui.Ctrl|ui.Alt) != 0 {
			return false
		}
		if k == ui.KeyZ || k == ui.KeyX {
			if !ev.Repeat {
				d := 1
				if k == ui.KeyZ {
					d = -1
				}
				from := s.base(lo, hi)
				if s.octave += d; s.base(lo, hi) == from {
					s.octave -= d
				}
			}
			return true
		}
		i := slices.IndexFunc(letters[:], func(l letter) bool { return l.key == k })
		if i < 0 {
			return false
		}
		if note := s.base(lo, hi) + i; !ev.Repeat && note >= lo && note <= hi {
			s.keys[k] = note
			s.down(note, typed)
		}
		return true
	}
	return false
}

func black(note int) bool { return "010100101010"[note%12] == '1' }

var pitches = [12]string{"C", "C♯", "D", "D♯", "E", "F", "F♯", "G", "G♯", "A", "A♯", "B"}

func name(note int) string { return pitches[note%12] + strconv.Itoa(note/12-1) }

// board is where the keys lo to hi are in a box: the white keys side by
// side, white wide, and the black over the lines between them.
type board struct {
	lo, hi   int
	white, h float32
}

func layout(lo, hi int, w, h float32) board {
	n := 0
	for note := lo; note <= hi; note++ {
		if !black(note) {
			n++
		}
	}
	return board{lo: lo, hi: hi, white: w / float32(max(n, 1)), h: h}
}

// rect returns where note's key is, from the top left of the box.
func (b board) rect(note int) ui.Rect {
	x := float32(0)
	for n := b.lo; n < note; n++ {
		if !black(n) {
			x += b.white
		}
	}
	if black(note) {
		w := b.white * 0.6
		return ui.Rect{X: x - w/2, W: w, H: b.h * 0.62}
	}
	return ui.Rect{X: x, W: b.white, H: b.h}
}

// at returns the note of the key at x, y, black keys over white, and how
// hard a press there plays it; the note is -1 off the keys.
func (b board) at(x, y float32) (note, velocity int) {
	for _, over := range [2]bool{true, false} {
		for n := b.lo; n <= b.hi; n++ {
			r := b.rect(n)
			if black(n) == over && x >= r.X && x < r.X+r.W && y >= 0 && y < r.H {
				return n, max(1, min(127, int(math.Round(float64(127*y/r.H)))))
			}
		}
	}
	return -1, 0
}

// qualities are the chords named, by their notes in semitones above the
// root.
var qualities = []struct {
	suffix string
	notes  []int
}{
	{"", []int{0, 4, 7}}, {"m", []int{0, 3, 7}}, {"dim", []int{0, 3, 6}}, {"aug", []int{0, 4, 8}},
	{"sus2", []int{0, 2, 7}}, {"sus4", []int{0, 5, 7}}, {"5", []int{0, 7}},
	{"6", []int{0, 4, 7, 9}}, {"m6", []int{0, 3, 7, 9}}, {"7", []int{0, 4, 7, 10}},
	{"maj7", []int{0, 4, 7, 11}}, {"m7", []int{0, 3, 7, 10}}, {"m(maj7)", []int{0, 3, 7, 11}},
	{"m7♭5", []int{0, 3, 6, 10}}, {"dim7", []int{0, 3, 6, 9}}, {"7sus4", []int{0, 5, 7, 10}},
	{"add9", []int{0, 2, 4, 7}}, {"madd9", []int{0, 2, 3, 7}}, {"9", []int{0, 2, 4, 7, 10}},
	{"maj9", []int{0, 2, 4, 7, 11}}, {"m9", []int{0, 2, 3, 7, 10}},
}

// chord names the chord of notes, lowest first: on the lowest note when it
// is a root, else on another, over the lowest as C/E is. It returns ""
// for notes of no chord.
func chord(notes []int) string {
	var set uint16
	for _, n := range notes {
		set |= 1 << (n % 12)
	}
	bass := notes[0] % 12
	for i := range 12 {
		root := (bass + i) % 12
		if set&(1<<root) == 0 {
			continue
		}
		rel := (set>>root | set<<(12-root)) & 0xfff
		for _, q := range qualities {
			var m uint16
			for _, n := range q.notes {
				m |= 1 << n
			}
			if rel != m {
				continue
			}
			s := pitches[root] + q.suffix
			if root != bass {
				s += "/" + pitches[bass]
			}
			return s
		}
	}
	return ""
}

// View creates a keyboard of the keys lo to hi, as MIDI numbers them,
// lighting the notes in lit, and returns the notes started and stopped
// since the last frame, in the order played. A black key at either end
// brings in the white key beside it. The keyboard takes the focus as it
// appears.
func View(c *ui.Context, s *State, lo, hi int, lit []int) []Event {
	t := c.Theme()
	if black(lo) {
		lo--
	}
	if black(hi) {
		hi++
	}
	if s.held == nil {
		s.held, s.keys = map[int]int{}, map[ui.Key]int{}
	}
	base := s.base(lo, hi)
	ui.Column(c).Gap(t.Space(3)).Children(func() {
		notes := slices.Sorted(maps.Keys(s.held))
		ui.Row(c).Gap(t.Space(4)).AlignItems(ui.Center).Children(func() {
			label, color := "No notes held", t.TextMuted
			if len(notes) > 0 {
				names := make([]string, len(notes))
				for i, n := range notes {
					names[i] = name(n)
				}
				label = strings.Join(names, " ")
				if ch := chord(notes); ch != "" {
					label, color = ch, t.Text
				}
			}
			ui.Text(c, label).FontSize(t.FontSize + 8).Bold().TextColor(color).Grow(1).Shrink(1).SingleLine()
			from, to := max(lo, base), min(hi, base+len(letters)-1)
			ui.Text(c, "Keys A to ; play "+name(from)+"–"+name(to)+" · Z and X change octave").
				FontSize(t.FontSize - 1).TextColor(t.TextMuted)
		})
		kb := ui.Box(c).Height(t.Space(44)).Focusable().AutoFocus().Cursor(ui.CursorPointer).
			Label("Piano keyboard").Bind(&s.ref)
		// A key let go while another window has the focus never comes up
		// here.
		if !kb.Focused() {
			for k, note := range s.keys {
				delete(s.keys, k)
				s.up(note)
			}
		}
		kb.HandleInput(func(ev ui.InputEvent) bool { return s.input(c, ev, lo, hi) })
		down := maps.Clone(s.held)
		marked := map[int]bool{}
		for _, n := range lit {
			marked[n] = true
		}
		labels := map[int]string{}
		for i, l := range letters {
			labels[base+i] = l.label
		}
		kb.Draw(func(p *ui.Painter, r ui.Rect) { draw(p, t, r, layout(lo, hi, r.W, r.H), down, marked, labels) })
	})
	ev := s.events
	s.events = nil
	return ev
}

func draw(p *ui.Painter, t *ui.Theme, r ui.Rect, b board, down map[int]int, marked map[int]bool, labels map[int]string) {
	// The keys are ivory and ebony in a dark theme too, the ivory dimmed.
	ivory, ebony := ui.RGB(252, 252, 249), ui.RGB(30, 30, 34)
	if t.Dark {
		ivory, ebony = ui.RGB(218, 218, 222), ui.RGB(12, 12, 14)
	}
	white := ui.RGB(255, 255, 255)
	text := func(x, y float32, s string, size float32, weight int, color ui.Color) {
		span := ui.Span{Text: s, Size: size, Weight: weight, Color: color}
		w, _ := p.MeasureText(0, span)
		p.RichText(x-w/2, y, 0, span)
	}
	letter, small := t.FontSize-1, t.FontSize-4
	p.Fill(r, ebony, t.Space(1.5))
	for n := b.lo; n <= b.hi; n++ {
		if black(n) {
			continue
		}
		k := b.rect(n)
		k = ui.Rect{X: r.X + k.X + 0.5, Y: r.Y, W: k.W - 1, H: k.H}
		face, ink := ivory, ui.RGB(110, 110, 118)
		if marked[n] {
			face = t.Success.Mix(ivory, 0.7)
		}
		if _, ok := down[n]; ok {
			face, ink = t.Accent, t.AccentText
		}
		p.Fill(k, face, t.Space(1))
		cx, bottom := k.X+k.W/2, k.Y+k.H-t.Space(2)
		if n%12 == 0 {
			text(cx, bottom-small-2, name(n), small, 400, ink.Alpha(0.8))
		}
		if l, ok := labels[n]; ok {
			text(cx, bottom-small-letter-t.Space(2), l, letter, 600, ink)
		}
	}
	p.Fill(ui.Rect{X: r.X, Y: r.Y, W: r.W, H: t.Space(1)}, ui.RGBA(0, 0, 0, 0.35), 0)
	for n := b.lo; n <= b.hi; n++ {
		if !black(n) {
			continue
		}
		k := b.rect(n)
		k.X, k.Y = r.X+k.X, r.Y
		face := ebony
		if marked[n] {
			face = t.Success.Mix(ebony, 0.4)
		}
		ink := white.Alpha(0.7)
		if _, ok := down[n]; ok {
			face, ink = t.Accent, t.AccentText
		}
		lip := t.Space(1.5)
		p.Fill(k, face.Mix(white, 0.2), t.Space(1))
		p.Fill(ui.Rect{X: k.X + 1, Y: k.Y, W: k.W - 2, H: k.H - lip}, face, t.Space(1))
		if l, ok := labels[n]; ok {
			text(k.X+k.W/2, k.Y+k.H-lip-letter-t.Space(2), l, small+1, 600, ink)
		}
	}
}
