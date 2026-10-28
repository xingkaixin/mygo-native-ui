// Drumpad plays sixteen drums made in code: click a pad, harder nearer
// its middle, or play the keys from 1 to 4 down to Z to V. Full Level
// hits every pad as hard as can be.
//
//	go run ./examples/drumpad
package main

import (
	"log"
	"math"
	"math/rand/v2"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/audio"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/drumpad"
)

const rate = 48000

var (
	red    = ui.Hex("#E5484D")
	orange = ui.Hex("#F07A2B")
	yellow = ui.Hex("#E9B020")
	green  = ui.Hex("#30A46C")
	blue   = ui.Hex("#3E7BFA")
	purple = ui.Hex("#8E4EC6")
)

var pads = []drumpad.Pad{
	{Name: "Kick", Color: red}, {Name: "Snare", Color: orange}, {Name: "Closed Hat", Color: yellow}, {Name: "Open Hat", Color: yellow},
	{Name: "Low Tom", Color: green}, {Name: "Mid Tom", Color: green}, {Name: "High Tom", Color: green}, {Name: "Clap", Color: orange},
	{Name: "Rim", Color: blue}, {Name: "Cowbell", Color: blue}, {Name: "Clave", Color: blue}, {Name: "Shaker", Color: blue},
	{Name: "Crash", Color: purple}, {Name: "Low Conga", Color: green}, {Name: "High Conga", Color: green}, {Name: "Sub", Color: red},
}

type app struct {
	pads   drumpad.State
	sounds []*audio.Sound
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(6)).AlignItems(ui.Center).Children(func() {
		ev := drumpad.View(c, &a.pads, pads)
		for _, h := range ev.Hits {
			pl := a.sounds[h.Pad].NewPlayer().SetVolume(0.15 + 0.85*h.Velocity)
			pl.Play()
		}
		if err := audio.Err(); err != nil {
			ui.Text(c, err.Error()).FontSize(t.FontSize - 1).TextColor(t.TextMuted)
		}
	})
}

// kit makes the sixteen drums, in the order of pads.
func kit() [][]float32 {
	r := rand.New(rand.NewPCG(5, 9))
	noise := func() float64 { return r.Float64()*2 - 1 }
	render := func(seconds float64, f func(t float64) float64) []float32 {
		out := make([]float32, int(seconds*rate))
		for i := range out {
			out[i] = float32(f(float64(i) / rate))
		}
		return out
	}
	// tone is a sine whose pitch falls from hi to lo, as a skin's does.
	tone := func(hi, lo, fall float64) func(t float64) float64 {
		phase, last := 0.0, 0.0
		return func(t float64) float64 {
			f := lo + (hi-lo)*math.Exp(-t/fall)
			phase += 2 * math.Pi * f * (t - last)
			last = t
			return math.Sin(phase)
		}
	}
	// hiss is noise with its lows taken out, brighter as k nears 1.
	hiss := func(k float64) func() float64 {
		prev := 0.0
		return func() float64 {
			n := noise()
			out := n - k*prev
			prev = n
			return out / (1 + k)
		}
	}
	decay := func(t, d float64) float64 { return math.Exp(-t / d) }
	kick, snareBody, snare := tone(160, 48, 0.035), tone(220, 180, 0.02), hiss(0.6)
	closed, open, clap := hiss(0.95), hiss(0.95), hiss(0.8)
	low, mid, high := tone(140, 95, 0.05), tone(190, 135, 0.05), tone(250, 180, 0.05)
	rim, shaker, crash := hiss(0.9), hiss(0.97), hiss(0.97)
	lowConga, highConga, sub := tone(330, 230, 0.012), tone(480, 360, 0.01), tone(70, 49, 0.06)
	return [][]float32{
		render(0.45, func(t float64) float64 { return kick(t) * decay(t, 0.14) }),
		render(0.25, func(t float64) float64 {
			return 0.45*snareBody(t)*decay(t, 0.05) + 0.6*snare()*decay(t, 0.07)
		}),
		render(0.06, func(t float64) float64 { return 0.35 * closed() * decay(t, 0.015) }),
		render(0.4, func(t float64) float64 { return 0.3 * open() * decay(t, 0.12) }),
		render(0.4, func(t float64) float64 { return 0.7 * low(t) * decay(t, 0.12) }),
		render(0.35, func(t float64) float64 { return 0.65 * mid(t) * decay(t, 0.11) }),
		render(0.3, func(t float64) float64 { return 0.6 * high(t) * decay(t, 0.1) }),
		render(0.3, func(t float64) float64 {
			// Three hands a few milliseconds apart, then the room.
			env := decay(t, 0.08) * 0.5
			for _, at := range []float64{0, 0.011, 0.022} {
				if t >= at && t < at+0.009 {
					env = math.Max(env, decay(t-at, 0.003))
				}
			}
			return 0.7 * clap() * env
		}),
		render(0.05, func(t float64) float64 {
			return (0.5*math.Sin(2*math.Pi*1700*t) + 0.4*rim()) * decay(t, 0.008)
		}),
		render(0.3, func(t float64) float64 {
			// Two square-ish tones a fifth and a bit apart, as an 808's.
			sq := func(f float64) float64 { return math.Tanh(4 * math.Sin(2*math.Pi*f*t)) }
			return 0.2 * (sq(540) + sq(800)) * decay(t, 0.07)
		}),
		render(0.08, func(t float64) float64 { return 0.6 * math.Sin(2*math.Pi*2500*t) * decay(t, 0.015) }),
		render(0.12, func(t float64) float64 {
			return 0.3 * shaker() * math.Sin(math.Pi*math.Min(1, t/0.12))
		}),
		render(1.6, func(t float64) float64 {
			ring := math.Sin(2*math.Pi*3150*t) + math.Sin(2*math.Pi*4370*t)
			return (0.3*crash() + 0.04*ring) * decay(t, 0.45)
		}),
		render(0.3, func(t float64) float64 { return 0.6 * lowConga(t) * decay(t, 0.09) }),
		render(0.25, func(t float64) float64 { return 0.55 * highConga(t) * decay(t, 0.07) }),
		render(1.0, func(t float64) float64 { return 0.6 * math.Tanh(1.6*sub(t)) * decay(t, 0.35) }),
	}
}

func main() {
	a := &app{}
	for _, samples := range kit() {
		a.sounds = append(a.sounds, audio.NewSound(rate, 1, samples))
	}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Drum Pads",
			Width:   520,
			Height:  560,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
