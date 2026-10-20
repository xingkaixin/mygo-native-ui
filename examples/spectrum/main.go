// Spectrum plays a loop it makes and shows the spectrum of what is heard.
//
//	go run ./examples/spectrum
package main

import (
	"log"
	"math"
	"math/rand/v2"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/audio"
	"github.com/egoist/mygo/ui"

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

type app struct {
	samples []float32
	player  *audio.Player
	meter   spectrum.State
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	var window []float32
	playing := a.player.Playing()
	if playing {
		at := min(len(a.samples), int(a.player.Position().Seconds()*rate))
		window = a.samples[max(0, at-4096):at]
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
			pos := a.player.Position().Truncate(time.Second)
			ui.Text(c, pos.String()+" / "+a.player.Duration().Truncate(time.Second).String()).TextColor(t.TextMuted).FontFeatures("tnum")
		})
		spectrum.View(c, &a.meter, window, rate).Grow(1)
	})
}

func main() {
	a := &app{samples: loop()}
	a.player = audio.NewSound(rate, 1, a.samples).NewPlayer()
	a.player.SetLoop(true)
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Spectrum",
			Width:   900,
			Height:  480,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
