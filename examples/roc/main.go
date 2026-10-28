// ROC scores 800 emails for spam, positives scoring higher on the whole
// but overlapping the rest. Drag the threshold, or press along either
// curve, to trade false alarms for missed spam.
//
//	go run ./examples/roc
package main

import (
	"log"
	"math/rand/v2"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/roc"
)

type app struct {
	roc     roc.State
	samples []roc.Sample
}

func sample() []roc.Sample {
	r := rand.New(rand.NewPCG(3, 4))
	out := make([]roc.Sample, 800)
	for i := range out {
		spam := i%4 == 0
		mean := 0.35
		if spam {
			mean = 0.68
		}
		score := min(max(mean+0.16*r.NormFloat64(), 0), 1)
		out[i] = roc.Sample{Score: score, Positive: spam}
	}
	return out
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(8)).Children(func() {
		roc.View(c, &a.roc, a.samples)
	})
}

func main() {
	a := &app{samples: sample()}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "ROC",
			Width:   900,
			Height:  640,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
