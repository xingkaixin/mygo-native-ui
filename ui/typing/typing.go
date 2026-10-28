// Package typing is a typing test: a passage to type, in muted text that
// turns to the text's color as it is typed right, and red where it is
// typed wrong, a wrong space shown as a red block; a caret glides from
// character to character and blinks while the typing rests. The clock
// starts at the first key, and a line above keeps the words a minute,
// the accuracy and the time as they go. Once the last character is typed
// the passage gives way to the result: the words a minute, the accuracy,
// the errors and the time, a line of the speed through the test with a
// red mark at each second an error was made, and Again.
//
//	ev := typing.View(c, &app.test, passage)
//	if ev.Finished {
//		app.best = max(app.best, ev.WPM)
//	}
//
// Backspace takes back a character, and Option+Backspace (Ctrl+Backspace
// on Windows and Linux) a word. Escape starts over, as Enter does once the
// result shows.
package typing

import (
	"math"
	"runtime"
	"strconv"
	"time"
	"unicode"

	"github.com/egoist/mygo/ui"
)

// Events is the test, and once it is over its result: whether it finished
// this frame, the words a minute, five characters typed right making a
// word, and the share of keys typed right.
type Events struct {
	ui.Element
	Finished      bool
	WPM, Accuracy float64
}

// State is what a test keeps: the passage laid out, what was typed and
// when, and the speed at each second. Keep it in the app's state, one for
// each test.
type State struct {
	src    string
	font   ui.Font
	runes  []rune
	words  []word
	width  float32
	lines  int
	ascent float32
	typed  []rune
	keys   int
	errors int
	// missed is the seconds an error was made at, and wpm the speed at the
	// end of each second.
	missed   []int
	wpm      []float64
	start    time.Time
	last     time.Time
	end      time.Time
	finished bool
	reported bool
}

// word is a word of the passage with the spaces after it: its first rune,
// its glyphs, where each of its runes starts and its width, and where it
// is laid out.
type word struct {
	from   int
	glyphs []ui.Glyph
	xs     []float32
	w      float32
	x      float32
	line   int
}

// lineHeight is the height of a line, in sizes of the font.
const lineHeight = 1.9

func (s *State) prepare(text string, font ui.Font) {
	s.src, s.font, s.width = text, font, -1
	s.runes = []rune(text)
	s.ascent = font.Metrics().Ascent
	s.words = s.words[:0]
	for i := 0; i < len(s.runes); {
		j := i
		for j < len(s.runes) && !unicode.IsSpace(s.runes[j]) {
			j++
		}
		for j < len(s.runes) && unicode.IsSpace(s.runes[j]) {
			j++
		}
		w := word{from: i, glyphs: ui.Shape(string(s.runes[i:j]), font), xs: make([]float32, j-i+1)}
		for k := range w.xs {
			w.xs[k] = -1
		}
		for _, g := range w.glyphs {
			if w.xs[g.Cluster] < 0 {
				w.xs[g.Cluster] = g.X
			}
			w.w = max(w.w, g.X+g.Advance)
		}
		w.xs[j-i] = w.w
		// A rune inside a ligature starts where its cluster does.
		for k := len(w.xs) - 2; k >= 0; k-- {
			if w.xs[k] < 0 {
				w.xs[k] = w.xs[k+1]
			}
		}
		s.words = append(s.words, w)
		i = j
	}
	s.restart()
}

func (s *State) restart() {
	s.typed, s.keys, s.errors = s.typed[:0], 0, 0
	s.missed, s.wpm = s.missed[:0], s.wpm[:0]
	s.start, s.end, s.finished, s.reported = time.Time{}, time.Time{}, false, false
}

func (s *State) layout(width float32) {
	s.width = width
	x, line := float32(0), 0
	for i := range s.words {
		w := &s.words[i]
		// The spaces at a word's end may hang past the edge.
		ink := w.xs[len(w.xs)-1]
		for k := len(w.xs) - 2; k >= 0 && unicode.IsSpace(s.runes[w.from+k]); k-- {
			ink = w.xs[k]
		}
		if x > 0 && width > 0 && x+ink > width {
			x, line = 0, line+1
		}
		w.x, w.line = x, line
		x += w.w
	}
	s.lines = line + 1
}

// caret returns where rune i starts, its line and across it.
func (s *State) caret(i int) (x float32, line int) {
	for k := len(s.words) - 1; k >= 0; k-- {
		w := s.words[k]
		if w.from <= i {
			return w.x + w.xs[min(i-w.from, len(w.xs)-1)], w.line
		}
	}
	return 0, 0
}

// right returns how many runes typed are right.
func (s *State) right() int {
	n := 0
	for i, r := range s.typed {
		if r == s.runes[i] {
			n++
		}
	}
	return n
}

// speed returns the words a minute typed by now, and the share of keys
// typed right.
func (s *State) speed(now time.Time) (wpm, accuracy float64) {
	if s.start.IsZero() {
		return 0, 1
	}
	if s.finished {
		now = s.end
	}
	min := max(now.Sub(s.start).Minutes(), 1.0/60)
	accuracy = 1
	if s.keys > 0 {
		accuracy = float64(s.keys-s.errors) / float64(s.keys)
	}
	return float64(s.right()) / 5 / min, accuracy
}

// tick notes the speed at the end of each second gone by now.
func (s *State) tick(now time.Time) {
	for sec := len(s.wpm) + 1; time.Duration(sec)*time.Second <= now.Sub(s.start); sec++ {
		wpm, _ := s.speed(s.start.Add(time.Duration(sec) * time.Second))
		s.wpm = append(s.wpm, wpm)
	}
}

// key takes a key typed at now.
func (s *State) key(r rune, now time.Time) {
	if s.finished || len(s.typed) >= len(s.runes) {
		return
	}
	if s.start.IsZero() {
		s.start = now
	}
	s.tick(now)
	s.keys++
	if r != s.runes[len(s.typed)] {
		s.errors++
		sec := int(now.Sub(s.start) / time.Second)
		if n := len(s.missed); n == 0 || s.missed[n-1] != sec {
			s.missed = append(s.missed, sec)
		}
	}
	s.typed = append(s.typed, r)
	if len(s.typed) == len(s.runes) {
		s.finished, s.end = true, now
		wpm, _ := s.speed(now)
		s.wpm = append(s.wpm, wpm)
	}
}

// back takes back a rune, or the word before the caret.
func (s *State) back(word bool) {
	n := len(s.typed)
	if n == 0 || s.finished {
		return
	}
	n--
	if word {
		for n > 0 && unicode.IsSpace(s.runes[n]) {
			n--
		}
		for n > 0 && !unicode.IsSpace(s.runes[n-1]) {
			n--
		}
	}
	s.typed = s.typed[:n]
}

// View creates the test of text: the line of its speed and the passage,
// or once it is typed the result.
func View(c *ui.Context, s *State, text string) Events {
	t := c.Theme()
	font := ui.Font{Family: t.Font, Size: t.FontSize + 8}
	if s.src != text || s.font != font {
		s.prepare(text, font)
	}
	ev := Events{}
	col := ui.Column(c).Gap(t.Space(5))
	ev.Element = col
	col.Children(func() {
		if s.finished {
			if s.result(c) {
				s.restart()
			}
		} else {
			s.passage(c, font)
		}
	})
	if s.finished {
		ev.WPM, ev.Accuracy = s.speed(s.end)
		ev.Finished, s.reported = !s.reported, true
	}
	return ev
}

// passage creates the line of the speed so far and the passage to type.
func (s *State) passage(c *ui.Context, font ui.Font) {
	t := c.Theme()
	now := c.Now()
	wpm, accuracy := s.speed(now)
	elapsed := time.Duration(0)
	if !s.start.IsZero() {
		elapsed = now.Sub(s.start)
		c.After(time.Second - elapsed%time.Second)
	}
	ui.Row(c).Gap(t.Space(6)).AlignItems(ui.Baseline).Children(func() {
		stat(c, strconv.Itoa(int(math.Round(wpm))), "wpm")
		stat(c, strconv.Itoa(int(math.Round(accuracy*100)))+"%", "accuracy")
		stat(c, clock(elapsed), "time")
		ui.Spacer(c)
		ui.Text(c, strconv.Itoa(len(s.typed))+" / "+strconv.Itoa(len(s.runes))).FontFeatures("tnum").TextColor(t.TextMuted)
	})
	lineH := font.Size * lineHeight
	box := ui.Box(c).Focusable().FocusRing(false).AutoFocus().InputMethod(false).Cursor(ui.CursorText).
		Label("Passage to type").Value(string(s.runes[len(s.typed):]))
	if w := box.LayoutBounds().W; w != s.width {
		s.layout(w)
	}
	box.HandleInput(func(ev ui.InputEvent) bool {
		switch ev.Kind {
		case ui.InputText:
			for _, r := range ev.Text {
				if r != '\n' && r != '\r' && r != '\t' {
					s.key(r, time.Now())
				}
			}
			s.last = time.Now()
			return true
		case ui.InputKeyDown:
			word := ui.Alt
			if runtime.GOOS != "darwin" {
				word = ui.Ctrl
			}
			switch {
			case ev.Key == ui.KeyBackspace:
				s.back(ev.Mods&word != 0)
				s.last = time.Now()
				return true
			case ev.Key == ui.KeyEscape:
				s.restart()
				return true
			}
		}
		return false
	})
	box.Height(float32(s.lines) * lineH)
	cx, line := s.caret(len(s.typed))
	x := box.Animate("typing.x", cx, 90*time.Millisecond)
	y := box.Animate("typing.y", float32(line)*lineH, 90*time.Millisecond)
	focused := box.Focused()
	typed, runes, last := s.typed, s.runes, s.last
	box.Draw(func(p *ui.Painter, r ui.Rect) {
		if r.W != s.width {
			c.Invalidate()
		}
		ty := (lineH-font.Size*1.2)/2 + s.ascent
		for _, w := range s.words {
			wx, wy := r.X+w.x, r.Y+float32(w.line)*lineH
			for _, g := range w.glyphs {
				i := w.from + g.Cluster
				color := t.TextMuted.Alpha(0.6)
				if i < len(typed) {
					color = t.Text
					if typed[i] != runes[i] {
						color = t.Danger
						if unicode.IsSpace(runes[i]) {
							p.Fill(ui.Rect{X: wx + g.X + 1, Y: wy + ty - s.ascent*0.8, W: g.Advance - 2, H: s.ascent}, t.Danger.Alpha(0.35), 2)
						}
					}
				}
				if !focused {
					color = color.Alpha(0.5)
				}
				p.Glyphs([]ui.Glyph{g}, wx, wy+ty, color)
			}
		}
		if !focused {
			return
		}
		// The caret shows while typing, and blinks once the typing rests.
		rest := p.Now().Sub(last)
		if rest > 500*time.Millisecond {
			phase := (rest - 500*time.Millisecond) % time.Second
			if phase >= 500*time.Millisecond {
				p.After(time.Second - phase)
				return
			}
			p.After(500*time.Millisecond - phase)
		} else {
			p.After(500*time.Millisecond - rest)
		}
		p.Fill(ui.Rect{X: r.X + x - 1, Y: r.Y + y + ty - s.ascent*1.05, W: 2, H: s.ascent * 1.35}, t.Accent, 1)
	})
}

// stat shows value large over what it is.
func stat(c *ui.Context, value, what string) {
	t := c.Theme()
	ui.Row(c).Gap(t.Space(1.5)).AlignItems(ui.Baseline).Children(func() {
		ui.Text(c, value).FontSize(t.FontSize + 6).Bold().FontFeatures("tnum")
		ui.Text(c, what).TextColor(t.TextMuted)
	})
}

// clock writes d as "0:07".
func clock(d time.Duration) string {
	sec := int(d / time.Second)
	return strconv.Itoa(sec/60) + ":" + strconv.Itoa(sec%60/10) + strconv.Itoa(sec%10)
}

// result creates the result of the test, and reports whether Again was
// asked for.
func (s *State) result(c *ui.Context) (again bool) {
	t := c.Theme()
	wpm, accuracy := s.speed(s.end)
	ui.Row(c).Gap(t.Space(10)).AlignItems(ui.End).Children(func() {
		ui.Column(c).Children(func() {
			ui.Text(c, "wpm").TextColor(t.TextMuted)
			ui.Text(c, strconv.Itoa(int(math.Round(wpm)))).FontSize(t.FontSize * 4).Bold().FontFeatures("tnum").TextColor(t.Accent)
		})
		figure := func(name, value string) {
			ui.Column(c).Gap(t.Space(1)).Padding(0, 0, t.Space(2), 0).Children(func() {
				ui.Text(c, name).TextColor(t.TextMuted)
				ui.Text(c, value).FontSize(t.FontSize + 8).Bold().FontFeatures("tnum")
			})
		}
		figure("accuracy", strconv.Itoa(int(math.Round(accuracy*100)))+"%")
		figure("errors", strconv.Itoa(s.errors))
		figure("time", clock(s.end.Sub(s.start)))
	})
	samples, missed := s.wpm, s.missed
	ui.Box(c).Height(t.Space(40)).Label("Speed through the test").Draw(func(p *ui.Painter, r ui.Rect) {
		top := 10.0
		for _, v := range samples {
			top = max(top, v)
		}
		top = math.Ceil(top/20) * 20
		at := func(i int, v float64) (float32, float32) {
			x := r.X + r.W*float32(i)/float32(max(1, len(samples)-1))
			return x, r.Y + r.H - r.H*float32(v/top)
		}
		for k := 0.0; k <= top; k += top / 2 {
			_, y := at(0, k)
			p.Line(r.X, y, r.X+r.W, y, 1, t.Border)
			p.Text(r.X+2, y-t.FontSize-2, strconv.Itoa(int(k)), t.FontSize-3, t.TextMuted)
		}
		var line ui.Path
		for i, v := range samples {
			x, y := at(i, v)
			if i == 0 {
				line.MoveTo(x, y)
			} else {
				line.LineTo(x, y)
			}
		}
		p.StrokePath(&line, 2, t.Accent)
		for _, sec := range missed {
			x := r.X + r.W*float32(sec)/float32(max(1, len(samples)-1))
			p.Line(x, r.Y+r.H-6, x, r.Y+r.H, 2, t.Danger)
		}
	})
	b := ui.PrimaryButton(c, "Again").AutoFocus().AlignSelf(ui.Start)
	return b.Clicked()
}
