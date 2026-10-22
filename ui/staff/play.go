package staff

import (
	"io"
	"math"
	"sync"
)

const (
	rate = 48000
	// release is how long a note takes to die away once let go, in
	// seconds.
	release = 0.12
)

// sound is a note to play: when it starts and is let go, in seconds, and
// its frequency.
type sound struct {
	start, end, hz float64
}

// span is when event ev sounds, lit.
type span struct {
	start, end float64
	ev         int
}

// schedule is a tune as it plays, repeats and all: its sounds, the spans
// of its events, both by start, and how long it lasts.
type schedule struct {
	sounds []sound
	spans  []span
	length float64
}

// order returns the measures in the order they play: back to the last
// start of a repeat, or the start of the tune, at the end of one, once.
func (t *tune) order() []int {
	var out []int
	from := 0
	done := map[int]bool{}
	for i := 0; i < len(t.measures); {
		m := t.measures[i]
		if m.open == repStart || m.open == repBoth {
			from = i
		}
		out = append(out, i)
		if (m.close == repEnd || m.close == repBoth) && !done[i] {
			done[i] = true
			i = from
			continue
		}
		if m.close == repStart || m.close == repBoth {
			from = i + 1
		}
		i++
	}
	return out
}

func hz(pitch int) float64 { return 440 * math.Pow(2, float64(pitch-69)/12) }

// plan works out when each note of t sounds at its tempo, a tied note
// going on through the note it is tied to.
func plan(t *tune) schedule {
	tick := 60 / float64(t.bpm) / float64(t.beat)
	var s schedule
	var at float64
	held := map[int]int{}
	for _, mi := range t.order() {
		m := t.measures[mi]
		for i := m.from; i < m.to; i++ {
			e := t.events[i]
			d := float64(e.ticks) * tick
			s.spans = append(s.spans, span{at, at + d, i})
			next := map[int]int{}
			for _, h := range e.heads {
				p := h.pitch()
				k, ok := held[p]
				if !ok {
					k = len(s.sounds)
					s.sounds = append(s.sounds, sound{start: at, hz: hz(p)})
				}
				s.sounds[k].end = at + d
				if h.tie {
					next[p] = k
				}
			}
			held = next
			at += d
		}
	}
	// Notes are let go a little early, so that a note played again is
	// heard twice.
	for i := range s.sounds {
		n := &s.sounds[i]
		n.end -= min(0.04, (n.end-n.start)*0.15)
	}
	if len(s.spans) > 0 {
		s.length = at + release
	}
	return s
}

// voice plays the sounds of a schedule, each sample made from the notes
// sounding then, so that it starts anywhere.
type voice struct {
	sounds []sound
	length int64
	mu     sync.Mutex
	pos    int64
}

func (v *voice) SampleRate() int { return rate }
func (v *voice) Channels() int   { return 1 }
func (v *voice) Len() int64      { return v.length }

func (v *voice) SetPosition(frame int64) error {
	v.mu.Lock()
	v.pos = frame
	v.mu.Unlock()
	return nil
}

// level is how loud a note held for held seconds is t seconds after it
// starts: struck, fading toward a third, and dying away once let go.
func level(t, held float64) float64 {
	env := func(t float64) float64 { return min(1, t/0.006) * (0.3 + 0.7*math.Exp(-t/0.3)) }
	if t < held {
		return env(t)
	}
	return env(held) * max(0, 1-(t-held)/release)
}

func (v *voice) Read(buf []float32) (int, error) {
	v.mu.Lock()
	pos := v.pos
	n := int(min(int64(len(buf)), v.length-pos))
	if n <= 0 {
		v.mu.Unlock()
		return 0, io.EOF
	}
	v.pos += int64(n)
	v.mu.Unlock()
	out := buf[:n]
	clear(out)
	t0, t1 := float64(pos)/rate, float64(pos+int64(n))/rate
	for _, s := range v.sounds {
		if s.start >= t1 {
			break
		}
		held := s.end - s.start
		if s.end+release <= t0 {
			continue
		}
		for i := range out {
			t := float64(pos+int64(i))/rate - s.start
			if t < 0 {
				continue
			}
			if t >= held+release {
				break
			}
			ph := 2 * math.Pi * s.hz * t
			w := math.Sin(ph) + 0.5*math.Sin(2*ph) + 0.25*math.Sin(3*ph) + 0.12*math.Sin(4*ph)
			out[i] += float32(0.16 * w * level(t, held))
		}
	}
	for i, x := range out {
		out[i] = float32(math.Tanh(float64(x)))
	}
	return n, nil
}
