// Runs compares the loss of four training runs at different learning
// rates: smooth the noise with the slider, put the loss on a log scale,
// read the runs at a step with the pointer, and hide a run from the
// legend.
//
//	go run ./examples/runs
package main

import (
	"log"
	"math"
	"math/rand/v2"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/runs"
)

type app struct {
	runs  []runs.Run
	chart runs.State
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(6)).Children(func() {
		runs.View(c, &a.chart, "train/loss", a.runs).Grow(1)
	})
}

// run makes the loss of a run that falls at speed to floor, noisy, and,
// past diverge steps, climbs again; diverge 0 never does.
func run(name string, seed uint64, speed, floor float64, diverge float64) runs.Run {
	rng := rand.New(rand.NewPCG(seed, 7))
	r := runs.Run{Name: name}
	for step := 0.0; step <= 20000; step += 50 {
		v := floor + 2.3*math.Exp(-step/speed)
		if diverge > 0 && step > diverge {
			v += math.Pow((step-diverge)/4000, 2)
		}
		v *= math.Exp(0.12 * rng.NormFloat64())
		r.Steps, r.Values = append(r.Steps, step), append(r.Values, v)
	}
	return r
}

func main() {
	a := &app{
		runs: []runs.Run{
			run("lr=1e-4", 1, 6000, 0.42, 0),
			run("lr=3e-4", 2, 2500, 0.31, 0),
			run("lr=1e-3", 3, 1200, 0.27, 0),
			run("lr=3e-3", 4, 600, 0.35, 9000),
		},
		chart: runs.State{Smoothing: 0.8},
	}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Runs",
			Width:   900,
			Height:  560,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
