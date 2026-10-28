// Loudness meters a made-up programme as it plays: a hum of chords under
// a voice-like buzz, quiet verses and loud choruses, with a gain to ride
// and three targets to meet.
//
//	go run ./examples/loudness
package main

import (
	"fmt"
	"log"
	"math"
	"math/rand/v2"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/loudness"
)

const rate = 48000

var targets = []float64{-23, -16, -14}

type app struct {
	meter   *loudness.Meter
	view    loudness.State
	playing bool
	last    time.Time
	gain    float64
	target  int
	n       int
	noise   float64
	rng     *rand.Rand
}

// fill writes the next n frames of the programme: twenty-second sections
// alternate between a quiet verse and a loud chorus.
func (a *app) fill(n int) []float32 {
	out := make([]float32, 0, 2*n)
	g := math.Pow(10, a.gain/20)
	for range n {
		t := float64(a.n) / rate
		a.n++
		section := 0.35
		if int(t/20)%2 == 1 {
			section = 1
		}
		chord := 0.0
		for k, f := range []float64{110, 138.6, 164.8, 220} {
			chord += math.Sin(2*math.Pi*f*t+float64(k)) / 4
		}
		a.noise += (a.rng.Float64()*2 - 1 - a.noise) * 0.08
		syllables := 0.5 + 0.5*math.Abs(math.Sin(2*math.Pi*2.2*t))
		voice := (math.Sin(2*math.Pi*180*t+3*math.Sin(2*math.Pi*5*t)) + 2*a.noise) * syllables
		x := g * section * (0.12*chord + 0.1*voice)
		out = append(out, float32(x), float32(x*0.9))
	}
	return out
}

func (a *app) render(c *ui.Context) {
	t := c.Theme()
	now := c.Now()
	if a.playing {
		if !a.last.IsZero() {
			a.meter.Write(a.fill(int(now.Sub(a.last).Seconds() * rate)))
		}
		a.last = now
		c.After(time.Second / 30)
	}
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(5)).Gap(t.Space(5)).Children(func() {
		ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
			label := "Play"
			if a.playing {
				label = "Pause"
			}
			if ui.PrimaryButton(c, label).Clicked() {
				a.playing, a.last = !a.playing, time.Time{}
			}
			if ui.Button(c, "Reset").Clicked() {
				a.meter.Reset()
				a.view = loudness.State{}
			}
			ui.Segmented(c, &a.target, "Broadcast −23", "Podcast −16", "Streaming −14")
			ui.Box(c).Grow(1)
			ui.Text(c, fmt.Sprintf("Gain %+.0f dB", a.gain)).TextColor(t.TextMuted).FontFeatures("tnum")
			ui.Slider(c, &a.gain, -20, 12).Width(t.Space(36))
		})
		loudness.View(c, &a.view, a.meter, targets[a.target]).Grow(1)
	})
}

func main() {
	a := &app{meter: loudness.NewMeter(rate, 2), playing: true, rng: rand.New(rand.NewPCG(1, 2))}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Loudness",
			Width:   860,
			Height:  560,
			Content: ui.View(a.render),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
