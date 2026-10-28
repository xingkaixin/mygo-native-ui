// Survival follows 300 patients of a made-up trial for up to three years:
// the share of each arm still free of relapse. Point along the plot to
// read each arm's survival at that month.
//
//	go run ./examples/survival
package main

import (
	"log"
	"math"
	"math/rand/v2"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/survival"
)

type app struct{ groups []survival.Group }

// arm returns n patients relapsing after median months on average, some
// dropping out along the way, and the rest still free at 36 months.
func arm(r *rand.Rand, n int, median float64) []survival.Subject {
	out := make([]survival.Subject, n)
	for i := range out {
		relapse := r.ExpFloat64() * median / math.Ln2
		dropout := r.ExpFloat64() * 90
		tm, event := min(relapse, dropout, 36), relapse < dropout && relapse < 36
		out[i] = survival.Subject{Time: math.Round(tm*10) / 10, Event: event}
	}
	return out
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(8)).Children(func() {
		survival.View(c, a.groups, "Months")
	})
}

func main() {
	r := rand.New(rand.NewPCG(7, 8))
	a := &app{groups: []survival.Group{
		{Name: "Placebo", Subjects: arm(r, 150, 11)},
		{Name: "Treatment", Subjects: arm(r, 150, 21)},
	}}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Survival",
			Width:   820,
			Height:  600,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
