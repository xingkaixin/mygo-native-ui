// Package morse turns text into Morse code and back, as a code trainer
// does. The text typed shows as the dots and dashes of each letter, with
// the letter under them; Play sends it as a tone at the speed set, each
// letter lit as it sounds. Below, a key to send by hand: hold Space while
// it has the focus, or press it, and the code sent is read back into
// letters, a short press a dot and a long one a dash, as the gaps after
// them end letters and words.
//
//	morse.View(c, &app.morse, &app.text)
package morse

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/egoist/mygo/audio"
	"github.com/egoist/mygo/ui"
)

// Code is the International Morse code of letters, digits and some
// punctuation.
var Code = map[rune]string{
	'A': ".-", 'B': "-...", 'C': "-.-.", 'D': "-..", 'E': ".", 'F': "..-.", 'G': "--.", 'H': "....",
	'I': "..", 'J': ".---", 'K': "-.-", 'L': ".-..", 'M': "--", 'N': "-.", 'O': "---", 'P': ".--.",
	'Q': "--.-", 'R': ".-.", 'S': "...", 'T': "-", 'U': "..-", 'V': "...-", 'W': ".--", 'X': "-..-",
	'Y': "-.--", 'Z': "--..",
	'0': "-----", '1': ".----", '2': "..---", '3': "...--", '4': "....-",
	'5': ".....", '6': "-....", '7': "--...", '8': "---..", '9': "----.",
	'.': ".-.-.-", ',': "--..--", '?': "..--..", '/': "-..-.", '=': "-...-", '@': ".--.-.",
}

var letters = func() map[string]rune {
	m := map[string]rune{}
	for r, code := range Code {
		m[code] = r
	}
	return m
}()

// Encode writes text in Morse code: letters apart by a space, words by
// " / ". Characters without a code are left out.
func Encode(text string) string {
	var words []string
	for w := range strings.FieldsSeq(strings.ToUpper(text)) {
		var codes []string
		for _, r := range w {
			if code, ok := Code[r]; ok {
				codes = append(codes, code)
			}
		}
		if len(codes) > 0 {
			words = append(words, strings.Join(codes, " "))
		}
	}
	return strings.Join(words, " / ")
}

// Decode reads Morse code written as Encode writes it, ? for a code
// it does not know.
func Decode(code string) string {
	var b strings.Builder
	for i, w := range strings.Split(code, "/") {
		if i > 0 {
			b.WriteByte(' ')
		}
		for c := range strings.FieldsSeq(w) {
			if r, ok := letters[c]; ok {
				b.WriteRune(r)
			} else {
				b.WriteByte('?')
			}
		}
	}
	return b.String()
}

const (
	rate = 44100
	tone = 640
)

// beep returns seconds of the tone, faded in and out over 5 ms so that
// it does not click, unless it is to loop.
func beep(seconds float64, fade bool) []float32 {
	out := make([]float32, int(seconds*rate))
	ramp := 0.005 * rate
	for i := range out {
		g := 1.0
		if fade {
			g = min(1, float64(i)/ramp, float64(len(out)-i)/ramp)
		}
		out[i] = float32(0.3 * g * math.Sin(2*math.Pi*tone*float64(i)/rate))
	}
	return out
}

// mark is an element sounding: from when to when, in units, and the
// letter it belongs to.
type mark struct {
	from, to float64
	letter   int
}

// timeline returns the elements of text's letters and how many units the
// whole message lasts.
func timeline(text string) ([]mark, float64) {
	var out []mark
	at, letter := 0.0, 0
	for wi, w := range strings.Fields(strings.ToUpper(text)) {
		if wi > 0 {
			at += 4
		}
		for _, r := range w {
			code, ok := Code[r]
			if !ok {
				continue
			}
			for _, sym := range code {
				n := 1.0
				if sym == '-' {
					n = 3
				}
				out = append(out, mark{at, at + n, letter})
				at += n + 1
			}
			at += 2
			letter++
		}
	}
	return out, max(0, at-3)
}

// State is what a code view keeps: its speed, playback, and the key's
// press, its letter so far and the text read from it. Keep it in the
// app's state, one for each view.
type State struct {
	wpm     float64
	started time.Time
	playing *audio.Player
	keyTone *audio.Player
	down    bool
	since   time.Time
	lastUp  time.Time
	symbols string
	read    string
}

func (s *State) unit() time.Duration { return time.Duration(float64(1200*time.Millisecond) / s.wpm) }

// View creates the code of *text, its player and the key.
func View(c *ui.Context, s *State, text *string) ui.Element {
	t := c.Theme()
	now := c.Now()
	if s.wpm == 0 {
		s.wpm = 18
	}
	unit := s.unit()
	marks, length := timeline(*text)
	elapsed := -1.0
	if !s.started.IsZero() {
		elapsed = float64(now.Sub(s.started)) / float64(unit)
		if elapsed > length {
			s.started, elapsed = time.Time{}, -1
		} else {
			c.After(time.Second / 60)
		}
	}
	litLetter, sounding := -1, s.down
	for _, m := range marks {
		if elapsed < 0 {
			break
		}
		if elapsed >= m.from-1 && elapsed < m.to+2 {
			litLetter = m.letter
		}
		if elapsed >= m.from && elapsed < m.to {
			sounding = true
		}
	}
	if !s.down && s.symbols != "" {
		gap := now.Sub(s.lastUp)
		if gap >= 3*unit {
			r, ok := letters[s.symbols]
			if !ok {
				r = '?'
			}
			s.read += string(r)
			s.symbols = ""
		} else {
			c.After(3*unit - gap)
		}
	}
	if !s.down && s.symbols == "" && s.read != "" && !strings.HasSuffix(s.read, " ") {
		if gap := now.Sub(s.lastUp); gap >= 7*unit {
			s.read += " "
		} else {
			c.After(7*unit - gap)
		}
	}
	changed := false
	root := ui.Column(c).Gap(t.Space(5))
	root.Children(func() {
		if ui.TextInput(c, text).Placeholder("Type a message").Label("Message").Changed() {
			changed = true
		}
		ui.Row(c).Wrap().Gap(t.Space(4)).Children(func() {
			k := 0
			for wi, w := range strings.Fields(strings.ToUpper(*text)) {
				if wi > 0 {
					ui.Box(c.Key([2]any{"space", wi})).Width(t.Space(4))
				}
				for _, r := range w {
					code, ok := Code[r]
					if !ok {
						continue
					}
					lit := k == litLetter
					ui.Column(c.Key(k)).Gap(t.Space(1)).AlignItems(ui.Center).Children(func() {
						ink := t.Text
						if lit {
							ink = t.Accent
						}
						width := float32(-1.5)
						for _, sym := range code {
							width += map[rune]float32{'.': 3.5, '-': 6.5}[sym]
						}
						ui.Box(c).Size(t.Space(width), t.Space(4)).Draw(func(p *ui.Painter, b ui.Rect) {
							x := b.X
							for _, sym := range code {
								if sym == '.' {
									var dot ui.Path
									p.FillPath(dot.Circle(x+t.Space(1.25), b.Y+b.H/2, t.Space(1)), ink)
									x += t.Space(3.5)
								} else {
									p.Fill(ui.Rect{X: x, Y: b.Y + b.H/2 - t.Space(1), W: t.Space(5), H: t.Space(2)}, ink, t.Space(1))
									x += t.Space(6.5)
								}
							}
						})
						ui.Text(c, string(r)).Font("monospace").Bold().TextColor(ink)
					})
					k++
				}
			}
		})
		ui.Row(c).Gap(t.Space(4)).AlignItems(ui.Center).Children(func() {
			label := "Play"
			if !s.started.IsZero() {
				label = "Stop"
			}
			if ui.Button(c, label).Disabled(len(marks) == 0).Clicked() {
				if s.playing != nil {
					s.playing.Stop()
				}
				if s.started.IsZero() {
					s.playing = audio.NewSound(rate, 1, render(marks, length, unit)).Play()
					s.started = now
				} else {
					s.playing, s.started = nil, time.Time{}
				}
			}
			ui.Text(c, "Speed").TextColor(t.TextMuted)
			ui.Slider(c, &s.wpm, 5, 35).Step(1).Width(t.Space(48)).Label("Words a minute").Disabled(!s.started.IsZero())
			ui.Text(c, fmt.Sprintf("%.0f wpm", s.wpm)).FontFeatures("tnum").Width(t.Space(16))
			lamp(c, sounding)
		})
		s.key(c, unit)
	})
	if changed {
		root.SetChanged()
	}
	return root
}

// render returns the samples of marks over length units of unit.
func render(marks []mark, length float64, unit time.Duration) []float32 {
	u := unit.Seconds()
	out := make([]float32, int(length*u*rate)+1)
	for _, m := range marks {
		copy(out[int(m.from*u*rate):], beep((m.to-m.from)*u, true))
	}
	return out
}

func lamp(c *ui.Context, on bool) {
	t := c.Theme()
	col := t.Text.Alpha(0.12)
	if on {
		col = ui.Hex("#F5A524")
	}
	b := ui.Box(c).Size(t.Space(4), t.Space(4)).Radius(t.Space(2)).Background(col).Label("Tone")
	if on {
		b.Shadow(0, 0, 10, 0, col.Alpha(0.8))
	}
}

// key creates the key to send with and what it read.
func (s *State) key(c *ui.Context, unit time.Duration) {
	t := c.Theme()
	if s.keyTone == nil {
		// A second of whole cycles, to loop without a click.
		s.keyTone = audio.NewSound(rate, 1, beep(1, false)).NewPlayer().SetLoop(true)
	}
	press := func(at time.Time) {
		if !s.down {
			s.down, s.since = true, at
			s.keyTone.Seek(0)
			s.keyTone.Play()
		}
	}
	release := func(at time.Time) {
		if s.down {
			s.down = false
			s.keyTone.Pause()
			if at.Sub(s.since) < 2*unit {
				s.symbols += "."
			} else {
				s.symbols += "-"
			}
			s.lastUp = at
		}
	}
	ui.Column(c).Gap(t.Space(2)).Children(func() {
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			pad := ui.Box(c).Size(t.Space(40), t.Space(16)).Radius(t.Radius + 4).Center().Focusable().Label("Key").Cursor(ui.CursorPointer)
			if s.down {
				pad.Background(t.Accent).Border(1, t.Accent)
			} else {
				pad.Background(t.Surface).Border(1, t.Border)
			}
			if pad.Focused() {
				pad.Border(2, t.Focus)
			}
			if !pad.Focused() && !pad.Pressed() {
				release(c.Now())
			}
			pad.HandleInput(func(ev ui.InputEvent) bool {
				switch {
				case ev.Kind == ui.InputKeyDown && ev.Key == ui.KeySpace:
					if !ev.Repeat {
						press(ev.Time)
					}
				case ev.Kind == ui.InputKeyUp && ev.Key == ui.KeySpace:
					release(ev.Time)
				case ev.Kind == ui.InputPointerDown && ev.Button == 0:
					press(ev.Time)
				case ev.Kind == ui.InputPointerUp && ev.Button == 0:
					release(ev.Time)
				default:
					return false
				}
				return true
			})
			ink := t.TextMuted
			if s.down {
				ink = t.AccentText
			}
			pad.Children(func() {
				ui.Text(c, "Hold Space or press").TextColor(ink)
			})
			ui.Column(c).Gap(t.Space(1)).Grow(1).Children(func() {
				ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Baseline).Children(func() {
					ui.Text(c, s.read).Font("monospace").FontSize(t.FontSize + 6).Bold()
					ui.Text(c, s.symbols).Font("monospace").FontSize(t.FontSize + 6).TextColor(t.Accent)
				})
				if s.read == "" && s.symbols == "" {
					ui.Text(c, "What you send reads here").TextColor(t.TextMuted)
				}
			})
			if ui.Button(c, "Clear").Disabled(s.read == "" && s.symbols == "").Clicked() {
				s.read, s.symbols = "", ""
			}
		})
	})
}
