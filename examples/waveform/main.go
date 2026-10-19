// Waveform shows a drum loop it makes up as its waveform, and plays it
// silently: Space or the button plays and pauses, a click moves the
// playhead, and dragging selects a stretch, which then plays in a loop.
// Cmd and scrolling zoom.
//
//	go run ./examples/waveform
package main

import (
	"fmt"
	"log"
	"math"
	"math/rand/v2"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/waveform"
)

const rate = 44100

// app is the clip, the view's state, and the playback: when it started,
// and from where.
type app struct {
	peaks    []waveform.Peak
	duration float64
	wave     waveform.State
	playing  bool
	started  time.Time
	from     float64
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	if a.playing {
		a.wave.Playhead = a.from + c.Now().Sub(a.started).Seconds()/a.duration
		lo, hi := a.wave.Selection()
		switch {
		case hi > lo && a.wave.Playhead >= hi:
			a.play(c, lo)
		case a.wave.Playhead >= 1:
			a.wave.Playhead, a.playing = 1, false
		default:
			c.After(time.Second / 60)
		}
	}
	ui.Column(c).Fill().Background(t.Background).Padding(16).Gap(12).Children(func() {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			label := "Play"
			if a.playing {
				label = "Pause"
			}
			button := ui.Button(c, label).Width(72)
			if button.Clicked() || c.Shortcut(0, ui.KeySpace) {
				a.toggle(c)
			}
			ui.Text(c, clock(a.wave.Playhead*a.duration)+" / "+clock(a.duration)).FontFeatures("tnum")
			ui.Spacer(c)
			if lo, hi := a.wave.Selection(); hi > lo {
				ui.Text(c, fmt.Sprintf("Selected %s", clock((hi-lo)*a.duration))).TextColor(t.TextMuted).FontFeatures("tnum")
			}
			ui.Text(c, fmt.Sprintf("%.0f×", a.wave.Zoom())).TextColor(t.TextMuted).FontFeatures("tnum")
		})
		wave := waveform.View(c, &a.wave, a.peaks).Grow(1).Label("Drum loop").
			Radius(8).Background(t.Surface)
		if wave.Changed() && a.playing {
			a.play(c, a.wave.Playhead)
		}
	})
}

// toggle plays from the playhead, or from the start at the end, or
// pauses.
func (a *app) toggle(c *ui.Context) {
	if a.playing {
		a.playing = false
		return
	}
	from := a.wave.Playhead
	if from >= 1 {
		from = 0
	}
	a.play(c, from)
}

func (a *app) play(c *ui.Context, from float64) {
	a.playing, a.started, a.from = true, c.Now(), from
	a.wave.Playhead = from
}

// clock formats seconds as m:ss.d.
func clock(s float64) string {
	return fmt.Sprintf("%d:%04.1f", int(s)/60, math.Mod(s, 60))
}

// loop makes up bars of a drum loop at 120 beats a minute: a kick on
// each beat, a snare on two and four, and hi-hats between.
func loop(bars int) []float32 {
	r := rand.New(rand.NewPCG(3, 4))
	beat := rate / 2
	out := make([]float32, bars*4*beat)
	hit := func(at int, length float64, sound func(t float64) float64) {
		for i := 0; i < int(length*rate) && at+i < len(out); i++ {
			t := float64(i) / rate
			out[at+i] += float32(sound(t))
		}
	}
	for b := range bars * 4 {
		at := b * beat
		hit(at, 0.4, func(t float64) float64 {
			return 0.6 * math.Exp(-t*9) * math.Sin(2*math.Pi*(50+90*math.Exp(-t*30))*t)
		})
		if b%2 == 1 {
			hit(at, 0.25, func(t float64) float64 { return 0.35 * math.Exp(-t*18) * (r.Float64()*2 - 1) })
		}
		hit(at+beat/2, 0.08, func(t float64) float64 { return 0.15 * math.Exp(-t*60) * (r.Float64()*2 - 1) })
	}
	for i, v := range out {
		out[i] = max(-1, min(1, v))
	}
	return out
}

func main() {
	samples := loop(8)
	a := &app{
		peaks:    waveform.Peaks(samples, len(samples)/32),
		duration: float64(len(samples)) / rate,
	}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Waveform",
			Width:   900,
			Height:  320,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
