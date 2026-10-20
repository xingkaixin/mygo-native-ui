// Pianokeys plays a small synth from a piano keyboard: press the keys or
// drag across them, or play them from the computer's keys; hold notes
// with Sustain, and light the notes of a scale.
//
//	go run ./examples/pianokeys
package main

import (
	"log"
	"math"
	"sync"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/audio"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/pianokeys"
)

// The keyboard spans three octaves, C3 to C6.
const (
	lowest  = 48
	highest = 84
	rate    = 32000
)

func hz(note int) float64 { return 440 * math.Pow(2, float64(note-69)/12) }

var sounds [highest + 1]struct {
	once sync.Once
	s    *audio.Sound
}

// sound returns the sound of note, made the first time.
func sound(note int) *audio.Sound {
	e := &sounds[note]
	e.once.Do(func() { e.s = audio.NewSound(rate, 1, tone(note)) })
	return e.s
}

// tone makes a plucked synth's note: two saws a few cents apart through a
// two-pole low-pass, each harmonic dying away, the higher the faster, and
// the low notes slower than the high.
func tone(note int) []float32 {
	f := hz(note)
	out := make([]float64, 3*rate)
	pace := math.Pow(2, float64(note-60)/24)
	for k := 1; k <= 32 && float64(k)*f < rate*0.45; k++ {
		r := float64(k) * f / 2400
		amp := 2 / math.Pi / float64(k) / math.Sqrt(1+r*r*r*r)
		decay := math.Exp(-(0.8 + 0.3*float64(k)) * pace / rate)
		for _, detune := range [2]float64{0.998, 1.002} {
			// A rotation for the sine and a product for the decay, which
			// sin and exp for each sample would make slow.
			w := 2 * math.Pi * float64(k) * f * detune / rate
			cw, sw := math.Cos(w), math.Sin(w)
			s, c, e := 0.0, 1.0, amp
			for i := range out {
				out[i] += e * s
				s, c = s*cw+c*sw, c*cw-s*sw
				e *= decay
			}
		}
	}
	peak := 0.0
	for _, v := range out {
		peak = max(peak, math.Abs(v))
	}
	samples := make([]float32, len(out))
	for i, v := range out {
		// 3 ms in and 50 ms out, so that neither end clicks.
		g := min(1, float64(i)/(0.003*rate), float64(len(out)-i)/(0.05*rate))
		samples[i] = float32(0.22 * g * v / peak)
	}
	return samples
}

// release fades a voice out over 150 ms and stops it.
func release(pl *audio.Player) {
	go func() {
		v := pl.Volume()
		for i := 9; i >= 0; i-- {
			pl.SetVolume(v * float64(i) / 10)
			time.Sleep(15 * time.Millisecond)
		}
		pl.Close()
	}()
}

// synth plays the notes the keyboard reports, a voice a note; with
// sustain, a note let go rings on until sustain is let go.
type synth struct {
	voices  map[int]*audio.Player
	held    map[int]bool
	sustain bool
}

func (s *synth) play(e pianokeys.Event) {
	if !e.On {
		delete(s.held, e.Note)
		if pl := s.voices[e.Note]; pl != nil && !s.sustain {
			release(pl)
			delete(s.voices, e.Note)
		}
		return
	}
	if pl := s.voices[e.Note]; pl != nil {
		release(pl)
	}
	pl := sound(e.Note).NewPlayer().SetVolume(0.2 + 0.8*float64(e.Velocity)/127)
	pl.Play()
	s.voices[e.Note], s.held[e.Note] = pl, true
}

// lift lets go of sustain: the notes no key holds fade out.
func (s *synth) lift() {
	for note, pl := range s.voices {
		if !s.held[note] {
			release(pl)
			delete(s.voices, note)
		}
	}
}

var roots = []string{"C", "C♯", "D", "D♯", "E", "F", "F♯", "G", "G♯", "A", "A♯", "B"}

// scales are in semitones above the root.
var scales = []struct {
	name  string
	steps []int
}{
	{"None", nil},
	{"Major", []int{0, 2, 4, 5, 7, 9, 11}},
	{"Minor", []int{0, 2, 3, 5, 7, 8, 10}},
	{"Dorian", []int{0, 2, 3, 5, 7, 9, 10}},
	{"Pentatonic", []int{0, 2, 4, 7, 9}},
	{"Blues", []int{0, 3, 5, 6, 7, 10}},
}

type app struct {
	keys  pianokeys.State
	synth synth
	root  int
	scale int
}

// lit returns the notes of the scale chosen on the keyboard.
func (a *app) lit() []int {
	var out []int
	for n := lowest; n <= highest; n++ {
		for _, step := range scales[a.scale].steps {
			if (n-a.root-step)%12 == 0 {
				out = append(out, n)
			}
		}
	}
	return out
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(6)).Gap(t.Space(5)).Children(func() {
		// The controls leave the focus on the keyboard, which plays from
		// the computer's keys only while it has it.
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).KeepFocus().Children(func() {
			ui.Text(c, "Synth").Bold().FontSize(t.FontSize + 6).Grow(1)
			if ui.Toggle(c, &a.synth.sustain, "Sustain").Changed() && !a.synth.sustain {
				a.synth.lift()
			}
		})
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).KeepFocus().Children(func() {
			ui.Text(c, "Scale").TextColor(t.TextMuted)
			ui.Segmented(c, &a.root, roots...)
			names := make([]string, len(scales))
			for i, s := range scales {
				names[i] = s.name
			}
			ui.Segmented(c, &a.scale, names...)
		})
		for _, e := range pianokeys.View(c, &a.keys, lowest, highest, a.lit()) {
			a.synth.play(e)
		}
		if err := audio.Err(); err != nil {
			ui.Text(c, err.Error()).FontSize(t.FontSize - 1).TextColor(t.TextMuted)
		}
	})
}

func main() {
	a := &app{synth: synth{voices: map[int]*audio.Player{}, held: map[int]bool{}}, root: 2, scale: 1}
	// Make every note's sound ahead, so that none waits for it.
	go func() {
		for n := lowest; n <= highest; n++ {
			sound(n)
		}
	}()
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Piano keys",
			Width:   1000,
			Height:  390,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
