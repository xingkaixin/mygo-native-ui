// Mixer plays a loop it makes in four stems, drums, bass, keys and a
// lead, each through a strip of a console, metering what is heard of
// each and of the master.
//
//	go run ./examples/mixer
package main

import (
	"log"
	"math"
	"math/rand/v2"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/audio"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/mixer"
)

const (
	rate = 48000
	beat = 0.5 // seconds, at 120 BPM
	bars = 8
)

func hz(midi int) float64 { return 440 * math.Pow(2, float64(midi-69)/12) }

func saw(phase float64) float64 { return 2 * (phase - math.Floor(phase+0.5)) }

// lowpass returns a one-pole filter at cut Hz.
func lowpass(cut float64) func(x float64) float64 {
	a, y := 1-math.Exp(-2*math.Pi*cut/rate), 0.0
	return func(x float64) float64 { y += a * (x - y); return y }
}

// release returns 1 until t reaches length, then falls to 0 over fade
// seconds.
func release(t, length, fade float64) float64 { return max(0, min(1, 1-(t-length)/fade)) }

// note is a note of the lead, in beats from the start of the loop.
type note struct {
	at, length float64
	pitch      int
}

var lead = []note{
	{0, 1, 76}, {1, 0.5, 74}, {1.5, 0.5, 72}, {2, 1, 74}, {3, 1, 76},
	{4, 2, 72}, {6, 1, 69}, {7, 1, 72},
	{8, 1, 67}, {9, 1, 72}, {10, 1, 76}, {11, 1, 74},
	{12, 2, 74}, {14, 1, 71}, {15, 1, 67},
	{16, 1, 81}, {17, 0.5, 79}, {17.5, 0.5, 76}, {18, 1, 79}, {19, 1, 81},
	{20, 2, 84}, {22, 1, 81}, {23, 1, 77},
	{24, 1, 79}, {25, 1, 76}, {26, 1, 72}, {27, 1, 76},
	{28, 3, 74},
}

// stems makes eight bars at 120 BPM of Am, F, C and G, a bar each, in
// four parts: a kick, snare and hats, a bass on the eighths, chords on the
// off-beats and a lead over them. Each part is scaled to its own peak.
func stems() [][]float32 {
	n := int(bars * 4 * beat * rate)
	parts := make([][]float32, 4)
	for i := range parts {
		parts[i] = make([]float32, n)
	}
	drums, bass, keys, melody := parts[0], parts[1], parts[2], parts[3]
	// add adds f from at seconds on for length seconds, going on from the
	// start past the end, so that the loop has no seam.
	add := func(out []float32, at, length float64, f func(t float64) float64) {
		from := int(at * rate)
		for i := range int(length * rate) {
			out[(from+i)%n] += float32(f(float64(i) / rate))
		}
	}
	r := rand.New(rand.NewPCG(3, 5))
	hiss := func() func() float64 {
		last := 0.0
		return func() float64 {
			v := r.Float64()*2 - 1
			d := v - last
			last = v
			return d
		}
	}
	chords := [][]int{{57, 60, 64}, {53, 57, 60}, {48, 52, 55}, {55, 59, 62}}
	roots := []int{33, 29, 36, 31}
	for b := range bars {
		bar := float64(b) * 4 * beat
		chord, root := chords[b%4], roots[b%4]
		for _, k := range []float64{0, 2, 2.5} {
			add(drums, bar+k*beat, 0.6, func(t float64) float64 {
				return math.Sin(2*math.Pi*(48*t+120*0.03*(1-math.Exp(-t/0.03)))) * math.Exp(-t/0.22)
			})
		}
		for _, k := range []float64{1, 3} {
			snap := hiss()
			add(drums, bar+k*beat, 0.4, func(t float64) float64 {
				return 0.45*snap()*math.Exp(-t/0.08) + 0.4*math.Sin(2*math.Pi*185*t)*math.Exp(-t/0.04)
			})
		}
		for k := range 8 {
			tick, level := hiss(), 0.12
			if k%2 == 1 {
				level = 0.2
			}
			add(drums, bar+float64(k)*beat/2, 0.1, func(t float64) float64 {
				return level * tick() * math.Exp(-t/0.02)
			})
		}
		for k, step := range []int{0, 0, 12, 0, 0, 12, 0, 7} {
			f, lp := hz(root+step), lowpass(700)
			add(bass, bar+float64(k)*beat/2, 0.25, func(t float64) float64 {
				x := 0.6*math.Sin(2*math.Pi*f*t) + 0.5*saw(f*t)
				return lp(x) * min(1, t/0.003) * math.Exp(-t/0.3) * release(t, 0.2, 0.03)
			})
		}
		for k := range 4 {
			lp := lowpass(1800)
			add(keys, bar+(float64(k)+0.5)*beat, 0.3, func(t float64) float64 {
				x := 0.0
				for _, m := range chord {
					x += saw(hz(m)*1.004*t) + saw(hz(m)*0.996*t)
				}
				return lp(x) * min(1, t/0.005) * math.Exp(-t/0.12)
			})
		}
	}
	for _, nt := range lead {
		f, lp, phase := hz(nt.pitch), lowpass(3200), 0.0
		length := nt.length*beat - 0.03
		add(melody, nt.at*beat, length+0.1, func(t float64) float64 {
			vibrato := 1 + 0.004*math.Sin(2*math.Pi*5.5*t)*min(1, t/0.3)
			phase += f * vibrato / rate
			x := 0.6*math.Tanh(3*math.Sin(2*math.Pi*phase)) + 0.4*saw(phase)
			return lp(x) * min(1, t/0.01) * release(t, length, 0.08)
		})
	}
	for i, peak := range []float32{0.5, 0.42, 0.3, 0.32} {
		top := float32(0)
		for _, v := range parts[i] {
			top = max(top, v, -v)
		}
		for k := range parts[i] {
			parts[i][k] *= peak / top
		}
	}
	return parts
}

type app struct {
	stems    [][]float32
	players  []*audio.Player
	channels []mixer.Channel
	master   float64
	desk     mixer.State
	// heard is the frame heard at the last frame.
	heard int
}

// apply sets each player's volume and pan from its strip and the master.
func (a *app) apply() {
	gains, master := mixer.Gains(a.channels), math.Pow(10, a.master/20)
	for i, p := range a.players {
		p.SetVolume(gains[i] * master).SetPan(a.channels[i].Pan)
	}
}

// levels returns the peaks of what was heard since the last frame, of
// each stem through its strip and of the master, mixed as the players
// mix them.
func (a *app) levels(playing bool) mixer.Levels {
	n := len(a.stems[0])
	at := int(a.players[0].Position().Seconds()*rate) % n
	from := a.heard
	a.heard = at
	out := mixer.Levels{Channels: make([][2]float32, len(a.stems))}
	if !playing {
		return out
	}
	gains, master := mixer.Gains(a.channels), math.Pow(10, a.master/20)
	sides := make([][2]float64, len(a.stems))
	for i, ch := range a.channels {
		// The players' pan keeps a mono sound's power, as decoded in the
		// middle.
		x := (ch.Pan + 1) * math.Pi / 4
		sides[i] = [2]float64{gains[i] * math.Sqrt2 * math.Cos(x), gains[i] * math.Sqrt2 * math.Sin(x)}
	}
	// After a seek or a stall, as while the window was hidden, only the
	// last moment is metered.
	for f := at - min((at-from+n)%n, rate/10); f < at; f++ {
		var sum [2]float64
		for i, s := range a.stems {
			for k := range 2 {
				v := float64(s[(f+n)%n]) * sides[i][k]
				out.Channels[i][k] = max(out.Channels[i][k], float32(math.Abs(v)))
				sum[k] += v
			}
		}
		for k := range 2 {
			out.Master[k] = max(out.Master[k], float32(math.Abs(sum[k]*master)))
		}
	}
	return out
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	playing := a.players[0].Playing()
	if playing {
		c.AnimationFrame()
	}
	levels := a.levels(playing)
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(6)).Gap(t.Space(4)).Children(func() {
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			label := "Play"
			if playing {
				label = "Pause"
			}
			if ui.PrimaryButton(c, label).Width(t.Space(20)).Clicked() {
				pos := a.players[0].Position()
				for _, p := range a.players {
					if playing {
						p.Pause()
					} else {
						p.Seek(pos)
						p.Play()
					}
				}
			}
			ui.Text(c, "Loop in A minor").Bold()
			length := time.Duration(len(a.stems[0])) * time.Second / rate
			pos := (a.players[0].Position() % length).Truncate(time.Second)
			ui.Text(c, pos.String()+" / "+length.String()).TextColor(t.TextMuted).FontFeatures("tnum")
		})
		if mixer.View(c, &a.desk, a.channels, &a.master, levels).Grow(1).Changed() {
			a.apply()
		}
	})
}

func main() {
	a := &app{
		stems: stems(),
		channels: []mixer.Channel{
			{Name: "Drums", Gain: -3},
			{Name: "Bass", Gain: -3},
			{Name: "Keys", Gain: -6, Pan: -0.35},
			{Name: "Lead", Gain: -4, Pan: 0.25},
		},
	}
	for _, s := range a.stems {
		a.players = append(a.players, audio.NewSound(rate, 1, s).NewPlayer().SetLoop(true))
	}
	a.apply()
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Mixer",
			Width:   520,
			Height:  620,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
