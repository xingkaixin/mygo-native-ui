// Adsr is a small synth played through an envelope you shape: keys A to
// K, with W, E, T, Y and U for the black keys, or a click on the octave
// on screen play notes, as many at once as are held, each a dot riding
// the curve; Pluck, Pad, Organ and Brass set the envelope.
//
//	go run ./examples/adsr
package main

import (
	"log"
	"math"
	"slices"
	"sync"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/audio"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/adsr"
)

const rate = 48000

func frames(d time.Duration) int64 { return int64(d.Seconds() * rate) }

// voice is a note played: its key, as MIDI numbers it, and the frames it
// went down and up at, up -1 while it is held.
type voice struct {
	id       int
	pitch    int
	down, up int64
	note     adsr.Note
}

// synth sounds its voices through env, each a mellow saw. A sample
// depends on nothing before it, so the player decodes again from where
// it is heard as a key goes down or up.
type synth struct {
	mu     sync.Mutex
	env    adsr.Envelope
	voices []voice
	pos    int64
}

func (s *synth) SampleRate() int { return rate }
func (s *synth) Channels() int   { return 1 }
func (s *synth) Len() int64      { return -1 }

func (s *synth) SetPosition(frame int64) error {
	s.mu.Lock()
	s.pos = frame
	s.mu.Unlock()
	return nil
}

func (s *synth) Read(buf []float32) (int, error) {
	s.mu.Lock()
	env, voices, pos := s.env, slices.Clone(s.voices), s.pos
	s.pos += int64(len(buf))
	s.mu.Unlock()
	clear(buf)
	for _, v := range voices {
		f := 440 * math.Pow(2, float64(v.pitch-69)/12)
		step := 2 * math.Pi * f / rate
		var amps []float64
		for k := 1.0; k <= 12 && k*f < rate/2; k++ {
			r := k * f / 2500
			amps = append(amps, 1/k/math.Sqrt(1+r*r*r*r))
		}
		for i := range buf {
			n := pos + int64(i) - v.down
			if n < 0 {
				continue
			}
			held := n
			if v.up >= 0 {
				held = v.up - v.down
			}
			level := env.Level(time.Duration(n)*time.Second/rate, time.Duration(held)*time.Second/rate)
			if level == 0 {
				continue
			}
			x := float64(n) * step
			s0, s1, c := 0.0, math.Sin(x), 2*math.Cos(x)
			sum := 0.0
			for _, a := range amps {
				sum += a * s1
				s0, s1 = s1, c*s1-s0
			}
			buf[i] += float32(sum * level * 0.2)
		}
	}
	for i, x := range buf {
		buf[i] = float32(math.Tanh(float64(x)))
	}
	return len(buf), nil
}

var presets = []struct {
	name string
	env  adsr.Envelope
}{
	{"Pluck", adsr.Envelope{Attack: 2 * time.Millisecond, Decay: 450 * time.Millisecond, Sustain: 0, Release: 250 * time.Millisecond}},
	{"Pad", adsr.Envelope{Attack: 900 * time.Millisecond, Decay: 1200 * time.Millisecond, Sustain: 0.7, Release: 2 * time.Second}},
	{"Organ", adsr.Envelope{Attack: 5 * time.Millisecond, Decay: 20 * time.Millisecond, Sustain: 1, Release: 40 * time.Millisecond}},
	{"Brass", adsr.Envelope{Attack: 80 * time.Millisecond, Decay: 250 * time.Millisecond, Sustain: 0.5, Release: 200 * time.Millisecond}},
}

// The keyboard is the octave from C4, MIDI's 60, with a computer key for
// each note, in semitones up from C: the middle row for the white keys,
// the one above for the black.
const lowest = 60

var computerKeys = []ui.Key{
	ui.KeyA, ui.KeyW, ui.KeyS, ui.KeyE, ui.KeyD, ui.KeyF, ui.KeyT, ui.KeyG, ui.KeyY, ui.KeyH, ui.KeyU, ui.KeyJ, ui.KeyK,
}

const labels = "AWSEDFTGYHUJK"

func black(pitch int) bool { return "010100101010"[pitch%12] == '1' }

type app struct {
	env      adsr.Envelope
	synth    *synth
	player   *audio.Player
	keyboard ui.Handle
	next     int
	keysDown map[ui.Key]int // computer keys held, with the voice each plays
	pressing bool           // whether the pointer presses the keyboard
	pointed  int            // the key it is on, or -1
	pointer  int            // the voice it plays, or 0
}

// play starts a voice of pitch at, and returns its id.
func (a *app) play(pitch int, at time.Time) int {
	pos := a.player.Position()
	a.next++
	a.synth.mu.Lock()
	a.synth.voices = append(a.synth.voices, voice{id: a.next, pitch: pitch, down: frames(pos), up: -1, note: adsr.Note{On: at}})
	a.synth.mu.Unlock()
	// What the player decoded ahead was without the voice.
	a.player.Seek(pos)
	a.player.Play()
	return a.next
}

// letGo lets go of voice id at.
func (a *app) letGo(id int, at time.Time) {
	pos := a.player.Position()
	a.synth.mu.Lock()
	for i, v := range a.synth.voices {
		if v.id == id && v.up < 0 {
			a.synth.voices[i].up = frames(pos)
			a.synth.voices[i].note.Off = at
		}
	}
	a.synth.mu.Unlock()
	a.player.Seek(pos)
}

// The keyboard's geometry, in DIPs.
const (
	whiteWidth  = 60
	whites      = 8
	keysHeight  = 150
	blackWidth  = 34
	blackHeight = 92
)

// keyRect returns where pitch's key is on the keyboard.
func keyRect(pitch int) ui.Rect {
	w := 0
	for p := lowest; p < pitch; p++ {
		if !black(p) {
			w++
		}
	}
	if black(pitch) {
		return ui.Rect{X: float32(w)*whiteWidth - blackWidth/2, W: blackWidth, H: blackHeight}
	}
	return ui.Rect{X: float32(w) * whiteWidth, W: whiteWidth, H: keysHeight}
}

// keyAt returns the pitch of the key at x, y, black keys over white, or -1.
func keyAt(x, y float32) int {
	in := func(r ui.Rect) bool { return x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H }
	for _, b := range []bool{true, false} {
		for p := lowest; p < lowest+len(computerKeys); p++ {
			if black(p) == b && in(keyRect(p)) {
				return p
			}
		}
	}
	return -1
}

func (a *app) input(ev ui.InputEvent) bool {
	switch ev.Kind {
	case ui.InputKeyDown, ui.InputKeyUp:
		if ev.Mods&(ui.Super|ui.Ctrl|ui.Alt) != 0 {
			return false
		}
		k := ev.PhysicalKey
		if k == ui.KeyUnknown {
			k = ev.Key
		}
		i := slices.Index(computerKeys, k)
		if i < 0 {
			return false
		}
		if ev.Repeat {
			return true
		}
		if id, ok := a.keysDown[k]; ok {
			a.letGo(id, ev.Time)
			delete(a.keysDown, k)
		}
		if ev.Kind == ui.InputKeyDown {
			a.keysDown[k] = a.play(lowest+i, ev.Time)
		}
		return true
	case ui.InputPointerDown, ui.InputPointerMove:
		if ev.Kind == ui.InputPointerMove && !a.pressing {
			return false
		}
		a.pressing = true
		// Dragging across the keys plays each it enters.
		if pitch := keyAt(ev.X, ev.Y); pitch != a.pointed {
			a.pointed = pitch
			if a.pointer != 0 {
				a.letGo(a.pointer, ev.Time)
				a.pointer = 0
			}
			if pitch >= 0 {
				a.pointer = a.play(pitch, ev.Time)
			}
		}
		return true
	case ui.InputPointerUp:
		a.pressing, a.pointed = false, -1
		if a.pointer != 0 {
			a.letGo(a.pointer, ev.Time)
			a.pointer = 0
		}
		return true
	}
	return false
}

// sounding hands the synth the envelope, drops the voices whose release
// is over, and returns the notes of those left and the keys held.
func (a *app) sounding(c *ui.Context) ([]adsr.Note, map[int]bool) {
	now := c.Now()
	// The output plays a little behind: a voice goes once surely heard.
	over := a.env.Release + 100*time.Millisecond
	s := a.synth
	s.mu.Lock()
	changed := s.env != a.env
	s.env = a.env
	s.voices = slices.DeleteFunc(s.voices, func(v voice) bool { return v.up >= 0 && now.Sub(v.note.Off) >= over })
	var notes []adsr.Note
	held := map[int]bool{}
	var next time.Duration
	for _, v := range s.voices {
		notes = append(notes, v.note)
		if v.up < 0 {
			held[v.pitch] = true
		} else if d := over - now.Sub(v.note.Off); next == 0 || d < next {
			next = d
		}
	}
	empty := len(s.voices) == 0
	s.mu.Unlock()
	if next > 0 {
		c.After(next)
	}
	switch {
	case empty:
		a.player.Pause()
	case changed:
		a.player.Seek(a.player.Position())
	}
	return notes, held
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	notes, held := a.sounding(c)
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(6)).Gap(t.Space(5)).Children(func() {
		// As wide as the editor and the keyboard, so their edges line up.
		ui.Row(c).Width(whites * whiteWidth).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "Envelope").Bold().FontSize(t.FontSize + 6).Grow(1)
			for _, pr := range presets {
				b := ui.Button(c.Key(pr.name), pr.name)
				if pr.env == a.env {
					b.Border(1, t.Accent).Background(t.Accent.Alpha(0.1))
				}
				if b.Clicked() {
					a.env = pr.env
					a.keyboard.Focus()
				}
			}
		})
		adsr.Editor(c, &a.env, notes)
		kb := ui.Box(c).Size(whites*whiteWidth, keysHeight).Focusable().AutoFocus().Bind(&a.keyboard).
			Label("Keyboard").HandleInput(a.input)
		if !kb.Focused() {
			// Keys let go of while the keyboard does not have the focus
			// never come back up to it.
			for k, id := range a.keysDown {
				a.letGo(id, c.Now())
				delete(a.keysDown, k)
			}
		}
		kb.Draw(func(p *ui.Painter, r ui.Rect) {
			for _, b := range []bool{false, true} {
				for i := range computerKeys {
					pitch := lowest + i
					if black(pitch) != b {
						continue
					}
					k := keyRect(pitch)
					k.X, k.Y = k.X+r.X, k.Y+r.Y
					face, edge, label := ui.Hex("#fbfbf8"), ui.Hex("#a1a1aa"), ui.Hex("#71717a")
					if b {
						face, edge, label = ui.Hex("#18181b"), ui.Hex("#18181b"), ui.Hex("#d4d4d8")
					}
					if held[pitch] {
						face = t.Accent.Mix(face, 0.5)
						if b {
							face = t.Accent
						}
					}
					p.Fill(k, face, 4)
					p.Stroke(k, edge, 4, 1)
					name := labels[i : i+1]
					w, _ := p.MeasureText(0, ui.Span{Text: name, Size: 12})
					p.Text(k.X+(k.W-w)/2, k.Y+k.H-22, name, 12, label)
				}
			}
		})
		hint := "Play with keys A to K, W E T Y U for the black keys, or click the keys"
		if err := audio.Err(); err != nil {
			hint = err.Error()
		}
		ui.Text(c, hint).FontSize(t.FontSize - 2).TextColor(t.TextMuted)
	})
}

func main() {
	s := &synth{}
	a := &app{
		env:      presets[3].env,
		synth:    s,
		player:   audio.NewPlayer(s),
		keysDown: map[ui.Key]int{},
		pointed:  -1,
	}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "ADSR",
			Width:   528,
			Height:  490,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
