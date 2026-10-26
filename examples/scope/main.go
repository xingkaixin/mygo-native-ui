// Scope watches a live signal on an oscilloscope: a tone of a shape on the
// first channel and a sine at a ratio of it on the second. Drag on the
// screen to move the trigger, step the time and the volts, and switch to
// XY for the Lissajous figure of the ratio, clearest with a sine.
//
//	go run ./examples/scope
package main

import (
	"fmt"
	"log"
	"math"
	"math/rand/v2"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/scope"
)

const rate = 48000

var ratios = []struct{ num, den float64 }{{1, 1}, {3, 2}, {2, 1}, {5, 4}}

type app struct {
	scope       scope.State
	freq, noise float64
	shape       int
	ratio       int
	started     time.Time
	ch1, ch2    []float64
	rng         *rand.Rand
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	c.AnimationFrame()
	a.sample(c.Now().Sub(a.started).Seconds())
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(6)).Gap(t.Space(5)).Children(func() {
		scope.View(c, &a.scope, rate, a.ch1, a.ch2).Width(t.Space(150))
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Wrap().Children(func() {
			ui.Segmented(c, &a.shape, "Sine", "Saw", "Square").Label("Shape")
			ui.Slider(c, &a.freq, 50, 2000).Width(t.Space(40)).Label("Frequency")
			ui.Text(c, fmt.Sprintf("%.0f Hz", a.freq)).FontFeatures("tnum").Width(t.Space(16))
			ui.Text(c, "Noise").TextColor(t.TextMuted)
			ui.Slider(c, &a.noise, 0, 0.3).Width(t.Space(24)).Label("Noise")
			labels := make([]string, len(ratios))
			for i, r := range ratios {
				labels[i] = fmt.Sprintf("%g:%g", r.num, r.den)
			}
			ui.Segmented(c, &a.ratio, labels...).Label("Ratio")
		})
	})
}

// sample fills the channels with the 100 ms of signal up to at seconds.
func (a *app) sample(at float64) {
	n := rate / 10
	a.ch1, a.ch2 = a.ch1[:0], a.ch2[:0]
	r := ratios[a.ratio]
	for i := range n {
		s := at - float64(n-i)/rate
		ph := 2 * math.Pi * a.freq * s
		v := math.Sin(ph)
		for h := 2.0; h <= 9 && a.shape > 0; h++ {
			switch {
			case a.shape == 1:
				v += math.Sin(h*ph) / h
			case int(h)%2 == 1:
				v += math.Sin(h*ph) / h
			}
		}
		a.ch1 = append(a.ch1, 0.8*v+a.noise*a.rng.NormFloat64())
		a.ch2 = append(a.ch2, 0.6*math.Sin(ph*r.den/r.num+math.Pi/4))
	}
}

func main() {
	a := &app{
		scope:   scope.State{TimeDiv: 500 * time.Microsecond, VoltsDiv: 0.5},
		freq:    440,
		noise:   0.03,
		started: time.Now(),
		rng:     rand.New(rand.NewPCG(1, 2)),
	}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Scope",
			Width:   720,
			Height:  700,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
