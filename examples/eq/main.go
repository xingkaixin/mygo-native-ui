// Eq plays a loop it makes through a parametric equalizer as it is
// shaped, with a bypass switch and the spectrum of what comes out.
//
//	go run ./examples/eq
package main

import (
	"log"
	"math"
	"math/rand/v2"
	"sync"
	"sync/atomic"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/audio"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/eq"
	"github.com/xingkaixin/mygo-native-ui/ui/spectrum"
)

const rate = 48000

func hz(midi int) float64 { return 440 * math.Pow(2, float64(midi-69)/12) }

// loop makes eight bars at 120 BPM: a pad of Am, F, C and G under a
// sweeping filter, a bass on the eighths, a kick on the beats and hats
// between them.
func loop() []float32 {
	const bar = 2.0
	chords := [][]int{{57, 60, 64}, {53, 57, 60}, {48, 55, 64}, {55, 59, 62}}
	n := int(8 * bar * rate)
	out := make([]float32, n)
	var lp float64
	r := rand.New(rand.NewPCG(1, 2))
	var last float64
	saw := func(f, t float64) float64 { p := f * t; return 2 * (p - math.Floor(p+0.5)) }
	for i := range out {
		t := float64(i) / rate
		chord := chords[int(t/(2*bar))%4]
		pad := 0.0
		for _, m := range chord {
			pad += saw(hz(m)*1.003, t) + saw(hz(m)*0.997, t)
		}
		cut := 500 + 3500*(0.5-0.5*math.Cos(2*math.Pi*t/(8*bar)))
		a := 1 - math.Exp(-2*math.Pi*cut/rate)
		lp += a * (pad*0.06 - lp)
		eighth := math.Mod(t, 0.25)
		bass := (math.Sin(2*math.Pi*hz(chord[0]-12)*t) + 0.3*saw(hz(chord[0]-12), t)) * 0.35 * math.Exp(-eighth/0.12)
		beat := math.Mod(t, 0.5)
		kick := math.Sin(2*math.Pi*(45*beat+75*(1-math.Exp(-beat/0.03))*0.03)) * 0.9 * math.Exp(-beat/0.18)
		noise := r.Float64()*2 - 1
		hat := 0.0
		if off := math.Mod(t+0.25, 0.5); off < 0.06 {
			hat = (noise - last) * 0.18 * math.Exp(-off/0.02)
		}
		last = noise
		out[i] = float32(math.Tanh(lp + bass + kick + hat))
	}
	return out
}

// live plays the loop over and over through the filter, keeping what it
// played by where it falls in the loop, for the spectrum.
type live struct {
	loop   []float32
	filter *eq.Filter
	player atomic.Pointer[audio.Player]
	pos    int64

	mu     sync.Mutex
	played []float32
}

func (l *live) SampleRate() int { return rate }
func (l *live) Channels() int   { return 1 }
func (l *live) Len() int64      { return -1 }

func (l *live) SetPosition(frame int64) error {
	l.pos = frame
	return nil
}

func (l *live) Read(buf []float32) (int, error) {
	// A player decodes a second ahead; this keeps a tenth of a second
	// ahead of what plays, so that the EQ is heard as it changes.
	for p := l.player.Load(); p == nil || l.pos-int64(p.Position().Seconds()*rate) > rate/10; p = l.player.Load() {
		time.Sleep(2 * time.Millisecond)
	}
	buf = buf[:min(len(buf), 512)]
	n := int64(len(l.loop))
	for i := range buf {
		buf[i] = l.loop[(l.pos+int64(i))%n]
	}
	l.filter.Process(buf)
	l.mu.Lock()
	for i, x := range buf {
		l.played[(l.pos+int64(i))%n] = x
	}
	l.mu.Unlock()
	l.pos += int64(len(buf))
	return len(buf), nil
}

type app struct {
	live   *live
	player *audio.Player
	filter *eq.Filter
	bands  []eq.Band
	bypass bool
	eq     eq.State
	meter  spectrum.State
	window []float32
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	if a.bypass {
		a.filter.Set(nil)
	} else {
		a.filter.Set(a.bands)
	}
	a.window = a.window[:0]
	playing := a.player.Playing()
	if playing {
		at := int(a.player.Position().Seconds()*rate) % len(a.live.loop)
		a.live.mu.Lock()
		a.window = append(a.window, a.live.played[max(0, at-4096):at]...)
		a.live.mu.Unlock()
		c.After(time.Second / 60)
	}
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(6)).Gap(t.Space(4)).Children(func() {
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			label := "Play"
			if playing {
				label = "Pause"
			}
			if ui.PrimaryButton(c, label).Width(t.Space(20)).Clicked() {
				if playing {
					a.player.Pause()
				} else {
					a.player.Play()
				}
			}
			ui.Text(c, "Loop in A minor").Bold()
			ui.Box(c).Grow(1)
			ui.Text(c, "Bypass").TextColor(t.TextMuted)
			ui.Switch(c, &a.bypass).Label("Bypass")
		})
		eq.View(c, &a.eq, &a.bands, rate).Grow(3)
		ui.Divider(c)
		spectrum.View(c, &a.meter, a.window, rate).Grow(2)
	})
}

func main() {
	samples := loop()
	a := &app{
		filter: eq.NewFilter(rate, 1),
		bands: []eq.Band{
			{Type: eq.LowCut, Freq: 35, Q: 0.71, On: true},
			{Type: eq.LowShelf, Freq: 120, Gain: 2.5, Q: 0.71, On: true},
			{Type: eq.Peak, Freq: 420, Gain: -4, Q: 1.6, On: true},
			{Type: eq.HighShelf, Freq: 9000, Gain: 3.5, Q: 0.71, On: true},
			{Type: eq.HighCut, Freq: 18000, Q: 0.71},
		},
	}
	a.filter.Set(a.bands)
	a.live = &live{loop: samples, filter: a.filter, played: make([]float32, len(samples))}
	a.player = audio.NewPlayer(a.live)
	a.live.player.Store(a.player)
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Equalizer",
			Width:   960,
			Height:  760,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
