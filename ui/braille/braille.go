// Package braille writes text in braille, uncontracted as in Unified
// English Braille grade 1, and types it as a Perkins brailler does. The
// text shows as cells of six raised or flat dots, each over what it
// reads, the capital and number signs among them. Below, a cell to write
// with: while it has the focus, hold F D S J K L together for dots 1 to 6
// and let go to write the cell, Space for a space and Backspace to take
// one back; or click its dots and Write. A capital sign makes the next
// letter a capital, and a number sign reads a to j as digits up to a
// space.
//
//	braille.View(c, &app.braille, &app.text)
package braille

import (
	"strings"
	"unicode"

	"github.com/egoist/mygo/ui"
)

var dotsOf = func() map[rune]uint8 {
	m := map[rune]uint8{}
	for r, ds := range map[rune]string{
		'a': "1", 'b': "12", 'c': "14", 'd': "145", 'e': "15", 'f': "124", 'g': "1245", 'h': "125", 'i': "24", 'j': "245",
		'k': "13", 'l': "123", 'm': "134", 'n': "1345", 'o': "135", 'p': "1234", 'q': "12345", 'r': "1235", 's': "234", 't': "2345",
		'u': "136", 'v': "1236", 'w': "2456", 'x': "1346", 'y': "13456", 'z': "1356",
		',': "2", ';': "23", ':': "25", '.': "256", '!': "235", '?': "236", '\'': "3", '-': "36",
	} {
		m[r] = dots(ds)
	}
	return m
}()

var byDots = func() map[uint8]rune {
	m := map[uint8]rune{}
	for r, d := range dotsOf {
		m[d] = r
	}
	return m
}()

var (
	capitalSign = dots("6")
	numberSign  = dots("3456")
	letterSign  = dots("56")
)

// dots returns the cell of the dots numbered in s, dot 1 the lowest bit
// as in Unicode's braille patterns.
func dots(s string) uint8 {
	var d uint8
	for _, n := range s {
		d |= 1 << (n - '1')
	}
	return d
}

type cell struct {
	dots  uint8
	reads string
	sign  bool
}

func cells(text string) []cell {
	var out []cell
	number := false
	for _, r := range text {
		lower := unicode.ToLower(r)
		switch {
		case unicode.IsSpace(r):
			out = append(out, cell{0, " ", false})
			number = false
		case r >= '0' && r <= '9':
			if !number {
				out = append(out, cell{numberSign, "num", true})
				number = true
			}
			out = append(out, cell{dotsOf[rune('a'+(r-'0'+9)%10)], string(r), false})
		case dotsOf[lower] != 0:
			if lower >= 'a' && lower <= 'z' {
				if unicode.IsUpper(r) {
					out = append(out, cell{capitalSign, "cap", true})
				} else if number && lower <= 'j' {
					out = append(out, cell{letterSign, "let", true})
				}
			}
			if lower != '.' && lower != ',' {
				number = false
			}
			out = append(out, cell{dotsOf[lower], string(r), false})
		}
	}
	return out
}

// Encode writes text in Unicode braille. Characters without a cell are
// left out.
func Encode(text string) string {
	var b strings.Builder
	for _, c := range cells(text) {
		b.WriteRune(0x2800 + rune(c.dots))
	}
	return b.String()
}

// Decode reads Unicode braille written as Encode writes it, ? for a cell
// it does not know. Other characters are kept.
func Decode(braille string) string {
	var b strings.Builder
	var rd reader
	for _, r := range braille {
		if r >= 0x2800 && r < 0x2840 {
			b.WriteString(rd.read(uint8(r - 0x2800)))
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// reader reads cells one at a time, keeping the signs that change what
// the next ones read.
type reader struct{ capital, number bool }

func (rd *reader) read(d uint8) string {
	switch d {
	case 0:
		*rd = reader{}
		return " "
	case capitalSign:
		rd.capital, rd.number = true, false
		return ""
	case numberSign:
		rd.number = true
		return ""
	case letterSign:
		rd.number = false
		return ""
	}
	r, ok := byDots[d]
	if !ok {
		*rd = reader{}
		return "?"
	}
	if rd.number {
		if r >= 'a' && r <= 'j' {
			return string(rune('0' + (r-'a'+1)%10))
		}
		if r == '.' || r == ',' {
			return string(r)
		}
		rd.number = false
	}
	if rd.capital {
		rd.capital = false
		r = unicode.ToUpper(r)
	}
	return string(r)
}

// State is what a braille view keeps: the dots held or clicked on the
// writer, and the signs written that the next cell reads by. Keep it in
// the app's state, one for each view.
type State struct {
	held, chord uint8
	reader      reader
	wrote       bool
}

var keys = map[ui.Key]uint8{ui.KeyF: 1, ui.KeyD: 2, ui.KeyS: 4, ui.KeyJ: 8, ui.KeyK: 16, ui.KeyL: 32}

// View creates the braille of *text and the writer that types into it.
func View(c *ui.Context, s *State, text *string) ui.Element {
	t := c.Theme()
	changed := false
	root := ui.Column(c).Gap(t.Space(6))
	root.Children(func() {
		if ui.TextInput(c, text).Placeholder("Type or write below").Label("Text").Changed() {
			changed, s.reader = true, reader{}
		}
		ui.Row(c).Wrap().GapX(t.Space(1.5)).GapY(t.Space(4)).Children(func() {
			for i, cl := range cells(*text) {
				ui.Column(c.Key(i)).Gap(t.Space(1.5)).AlignItems(ui.Center).Width(t.Space(9)).Children(func() {
					ink := t.Text
					if cl.sign {
						ink = t.Accent
					}
					ui.Box(c).Size(t.Space(7), t.Space(10.5)).Draw(func(p *ui.Painter, b ui.Rect) {
						drawCell(p, b, cl.dots, t.Space(1.1), ink, t.Text.Alpha(0.12))
					})
					label := ui.Text(c, cl.reads).FontSize(t.FontSize - 1)
					if cl.sign {
						label.TextColor(t.Accent).FontSize(t.FontSize - 3)
					}
				})
			}
		})
		s.writer(c, text)
	})
	if changed || s.wrote {
		s.wrote = false
		root.SetChanged()
	}
	return root
}

// drawCell draws a cell of six dots of radius r in b, raised dots in ink.
func drawCell(p *ui.Painter, b ui.Rect, d uint8, r float32, ink, flat ui.Color) {
	for i := range 6 {
		x := b.X + b.W*float32(1+2*(i/3))/4
		y := b.Y + b.H*float32(1+2*(i%3))/6
		var dot ui.Path
		if d&(1<<i) != 0 {
			p.FillPath(dot.Circle(x, y, r), ink)
		} else {
			p.FillPath(dot.Circle(x, y, r*0.7), flat)
		}
	}
}

func (s *State) write(text *string, d uint8) {
	*text += s.reader.read(d)
	s.wrote = true
}

func (s *State) backspace(text *string) {
	if r := []rune(*text); len(r) > 0 {
		*text = string(r[:len(r)-1])
		s.wrote = true
	}
	s.reader = reader{}
}

// writer creates the cell to write with and its buttons.
func (s *State) writer(c *ui.Context, text *string) {
	t := c.Theme()
	pitch := t.Space(14)
	ui.Row(c).Gap(t.Space(6)).AlignItems(ui.Center).Children(func() {
		pad := ui.Box(c).Size(pitch*2, pitch*3).Radius(t.Radius+6).Background(t.Surface).Border(1, t.Border).Focusable().Label("Writer").Cursor(ui.CursorPointer)
		if pad.Focused() {
			pad.Border(2, t.Focus)
		} else {
			s.held = 0
		}
		pad.HandleInput(func(ev ui.InputEvent) bool {
			if ev.PhysicalKey == ui.KeyUnknown {
				ev.PhysicalKey = ev.Key
			}
			switch ev.Kind {
			case ui.InputKeyDown:
				if bit, ok := keys[ev.PhysicalKey]; ok {
					if !ev.Repeat {
						s.held |= bit
						s.chord |= bit
					}
					return true
				}
				switch ev.Key {
				case ui.KeySpace:
					s.write(text, 0)
				case ui.KeyBackspace:
					s.backspace(text)
				case ui.KeyEnter:
					if s.chord != 0 {
						s.write(text, s.chord)
						s.chord = 0
					}
				default:
					return false
				}
				return true
			case ui.InputKeyUp:
				bit, ok := keys[ev.PhysicalKey]
				if !ok {
					return false
				}
				s.held &^= bit
				if s.held == 0 && s.chord != 0 {
					s.write(text, s.chord)
					s.chord = 0
				}
				return true
			case ui.InputPointerDown:
				col, row := int(ev.X/pitch), int(ev.Y/pitch)
				dx, dy := ev.X-pitch*(float32(col)+0.5), ev.Y-pitch*(float32(row)+0.5)
				if ev.Button == 0 && col >= 0 && col < 2 && row >= 0 && row < 3 && dx*dx+dy*dy < pitch*pitch*0.16 {
					s.chord ^= 1 << (col*3 + row)
				}
				return true
			}
			return false
		})
		held, chord := s.held, s.chord
		pad.Draw(func(p *ui.Painter, b ui.Rect) {
			for i := range 6 {
				x := b.X + pitch*(float32(i/3)+0.5)
				y := b.Y + pitch*(float32(i%3)+0.5)
				var dot ui.Path
				dot.Circle(x, y, pitch*0.32)
				switch {
				case held&(1<<i) != 0:
					p.FillPath(&dot, t.Accent)
				case chord&(1<<i) != 0:
					p.FillPath(&dot, t.Accent.Alpha(0.7))
				default:
					p.FillPath(&dot, t.Text.Alpha(0.06))
					p.StrokePath(&dot, 1, t.Border)
				}
				key := string("FDSJKL"[i])
				ink := t.TextMuted
				if chord&(1<<i) != 0 {
					ink = t.AccentText
				}
				w, h := p.MeasureText(0, ui.Span{Text: key, Size: t.FontSize})
				p.Text(x-w/2, y-h/2, key, t.FontSize, ink)
			}
		})
		ui.Column(c).Gap(t.Space(3)).Children(func() {
			next := s.reader
			reads := next.read(s.chord)
			switch {
			case s.chord == 0:
				reads = ""
			case s.chord == capitalSign:
				reads = "Capital sign"
			case s.chord == numberSign:
				reads = "Number sign"
			case s.chord == letterSign:
				reads = "Letter sign"
			}
			ui.Text(c, reads).FontSize(t.FontSize + 10).Bold().Height(t.Space(9))
			var mode []string
			if s.reader.capital {
				mode = append(mode, "Capital next")
			}
			if s.reader.number {
				mode = append(mode, "Digits")
			}
			ui.Text(c, strings.Join(mode, " · ")).TextColor(t.Accent)
			ui.Row(c).Gap(t.Space(2)).Children(func() {
				if ui.Button(c, "Write").Disabled(s.chord == 0).Clicked() {
					s.write(text, s.chord)
					s.chord = 0
				}
				if ui.Button(c, "Space").Clicked() {
					s.write(text, 0)
				}
				if ui.Button(c, "Delete").Disabled(*text == "").Clicked() {
					s.backspace(text)
				}
			})
		})
	})
}
