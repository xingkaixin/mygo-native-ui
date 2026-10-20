// Package sequencer is a step sequencer of drums, as drum machines are: a
// row for each drum and a column for each sixteenth of a bar, where a
// click sets a step to sound or not; Play loops the bar at the tempo, the
// column playing lit as it is heard. The drums are made as they play,
// each step on the sample it falls on, and a change is heard at once.
//
//	sequencer.View(c, &app.drums, &app.pattern, &app.bpm)
package sequencer

import (
	"fmt"
	"math"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/egoist/mygo/audio"
	"github.com/egoist/mygo/ui"
)

// Steps is how many steps a bar has: sixteenths.
const Steps = 16

// Drum is a drum of a kit.
type Drum uint8

const (
	Kick Drum = iota
	Snare
	Clap
	ClosedHat
	OpenHat
	LowTom
	HighTom
	Rim
	drums
)

var names = [drums]string{"Kick", "Snare", "Clap", "Closed hat", "Open hat", "Low tom", "High tom", "Rim"}

// Pattern is a bar: for each drum, whether each of its steps sounds.
type Pattern [drums][Steps]bool

const rate = 48000

// kit is each drum's sound, made once.
var kit = sync.OnceValue(func() [drums][]float32 {
	r := rand.New(rand.NewPCG(7, 11))
	noise := func() float64 { return r.Float64()*2 - 1 }
	render := func(seconds float64, f func(t float64) float64) []float32 {
		out := make([]float32, int(seconds*rate))
		for i := range out {
			out[i] = float32(f(float64(i) / rate))
		}
		return out
	}
	// tone returns a sine whose pitch falls from hi to lo, as a skin's
	// does, its phase summed as the pitch changes.
	tone := func(hi, lo, fall float64) func(t float64) float64 {
		phase, last := 0.0, 0.0
		return func(t float64) float64 {
			f := lo + (hi-lo)*math.Exp(-t/fall)
			phase += 2 * math.Pi * f * (t - last)
			last = t
			return math.Sin(phase)
		}
	}
	// hiss returns noise with its lows taken out, brighter as k nears 1.
	hiss := func(k float64) func() float64 {
		prev := 0.0
		return func() float64 {
			n := noise()
			out := n - k*prev
			prev = n
			return out / (1 + k)
		}
	}
	var k [drums][]float32
	kick := tone(160, 48, 0.035)
	k[Kick] = render(0.45, func(t float64) float64 { return kick(t) * math.Exp(-t/0.14) })
	body, snap := tone(220, 180, 0.02), hiss(0.6)
	k[Snare] = render(0.25, func(t float64) float64 {
		return 0.45*body(t)*math.Exp(-t/0.05) + 0.6*snap()*math.Exp(-t/0.07)
	})
	clap := hiss(0.8)
	k[Clap] = render(0.3, func(t float64) float64 {
		// Three hands a few milliseconds apart, then the room.
		env := math.Exp(-t/0.08) * 0.5
		for _, at := range []float64{0, 0.011, 0.022} {
			if t >= at && t < at+0.009 {
				env = math.Max(env, math.Exp(-(t-at)/0.003))
			}
		}
		return 0.7 * clap() * env
	})
	closed := hiss(0.95)
	k[ClosedHat] = render(0.06, func(t float64) float64 { return 0.35 * closed() * math.Exp(-t/0.015) })
	open := hiss(0.95)
	k[OpenHat] = render(0.4, func(t float64) float64 { return 0.3 * open() * math.Exp(-t/0.12) })
	low := tone(140, 95, 0.05)
	k[LowTom] = render(0.4, func(t float64) float64 { return 0.7 * low(t) * math.Exp(-t/0.12) })
	high := tone(230, 165, 0.05)
	k[HighTom] = render(0.35, func(t float64) float64 { return 0.6 * high(t) * math.Exp(-t/0.1) })
	click := hiss(0.9)
	k[Rim] = render(0.05, func(t float64) float64 {
		return (0.5*math.Sin(2*math.Pi*1700*t) + 0.4*click()) * math.Exp(-t/0.008)
	})
	return k
})

// synth makes the sound of a pattern looped at a tempo: each sample is
// what the steps before it make there, so that it starts anywhere.
type synth struct {
	mu      sync.Mutex
	pattern Pattern
	step    float64 // frames a step
	pos     int64
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
	p, step, pos := s.pattern, s.step, s.pos
	s.pos += int64(len(buf))
	s.mu.Unlock()
	k := kit()
	for i := range buf {
		f := pos + int64(i)
		g := int64(float64(f) / step)
		var v float64
		for d := range drums {
			for n := g; n >= 0; n-- {
				off := f - int64(math.Round(float64(n)*step))
				if off >= int64(len(k[d])) {
					break
				}
				if off >= 0 && p[d][n%Steps] {
					v += float64(k[d][off])
				}
			}
		}
		buf[i] = float32(math.Tanh(v))
	}
	return len(buf), nil
}

// frames returns how many frames a step takes at bpm, four steps a beat.
func frames(bpm float64) float64 { return rate * 60 / bpm / 4 }

// State is what a sequencer keeps: its player, and the pattern and tempo
// it plays. Keep it in the app's state, one for each sequencer.
type State struct {
	player *audio.Player
	synth  *synth
	bpm    float64
}

// Playing reports whether the sequencer plays.
func (s *State) Playing() bool { return s.player != nil && s.player.Playing() }

// at returns the step being heard.
func (s *State) at() (int, time.Duration) {
	f := s.player.Position().Seconds() * rate
	n := math.Floor(f / s.synth.step)
	left := (n+1)*s.synth.step - f
	return int(n) % Steps, time.Duration(left / rate * float64(time.Second))
}

// View creates the sequencer of *pattern at *bpm beats a minute.
func View(c *ui.Context, s *State, pattern *Pattern, bpm *float64) ui.Element {
	t := c.Theme()
	*bpm = max(40, min(240, *bpm))
	if s.synth == nil {
		s.synth = &synth{step: frames(*bpm)}
		s.bpm = *bpm
		s.player = audio.NewPlayer(s.synth)
	}
	s.synth.mu.Lock()
	changed := s.synth.pattern != *pattern || s.bpm != *bpm
	s.synth.pattern = *pattern
	old := s.synth.step
	s.synth.step = frames(*bpm)
	s.bpm = *bpm
	s.synth.mu.Unlock()
	playing := s.Playing()
	if changed && playing {
		// What the player decoded ahead was of the old pattern; from where
		// it is heard, in steps, it decodes the new one.
		pos := s.player.Position()
		s.player.Seek(time.Duration(float64(pos) * s.synth.step / old))
	}
	head := -1
	if playing {
		var left time.Duration
		head, left = s.at()
		c.After(left)
	}
	cell, gap := t.Space(9), t.Space(1)
	return ui.Column(c).Gap(t.Space(4)).Children(func() {
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			label := "Play"
			if playing {
				label = "Stop"
			}
			if ui.PrimaryButton(c, label).Width(t.Space(20)).Clicked() {
				if playing {
					s.player.Stop()
				} else {
					s.player.Seek(0)
					s.player.Play()
				}
			}
			ui.NumberInput(c, bpm, 40, 240, 1).Label("Tempo")
			ui.Text(c, "BPM").TextColor(t.TextMuted)
			ui.Box(c).Grow(1)
			if ui.Button(c, "Clear").Clicked() {
				*pattern = Pattern{}
			}
		})
		ui.Column(c).Gap(gap).Children(func() {
			for d := range drums {
				color := ui.Oklch(0.66, 0.15, float32((15+int(d)*45)%360))
				ui.Row(c.Key(d)).Gap(gap).AlignItems(ui.Center).Children(func() {
					ui.Text(c, names[d]).Width(t.Space(26)).FontSize(t.FontSize - 1).TextColor(t.TextMuted)
					for k := range Steps {
						if k > 0 && k%4 == 0 {
							ui.Box(c.Key(-k)).Width(t.Space(2))
						}
						on := pattern[d][k]
						b := ui.ButtonBase(c.Key(k)).Size(cell, cell).Radius(t.Space(1.5)).
							Label(fmt.Sprintf("%s, step %d", names[d], k+1))
						bg := t.Text.Alpha(0.06)
						if k%4 == 0 {
							bg = t.Text.Alpha(0.1)
						}
						switch {
						case on && k == head:
							bg = color.Mix(ui.RGB(255, 255, 255), 0.35)
						case on:
							bg = color
						case k == head:
							bg = t.Text.Alpha(0.2)
						case b.Hovered():
							bg = t.Text.Alpha(0.16)
						}
						b.Background(bg)
						if b.Clicked() {
							pattern[d][k] = !on
						}
					}
				})
			}
		})
	})
}
