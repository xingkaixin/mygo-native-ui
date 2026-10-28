// Hypnogram shows a made-up night of sleep in five cycles, deep sleep
// early and REM late, as a night goes; each click makes another.
//
//	go run ./examples/hypnogram
package main

import (
	"log"
	"math/rand/v2"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/hypnogram"
)

// night makes up a night of five sleep cycles of about ninety minutes:
// deep sleep shortens and REM lengthens from cycle to cycle, with brief
// wakings between some.
func night(seed uint64) []hypnogram.Spell {
	rng := rand.New(rand.NewPCG(seed, 7))
	at := time.Date(2026, 10, 27, 23, 10+rng.IntN(20), 0, 0, time.Local)
	var spells []hypnogram.Spell
	add := func(st hypnogram.Stage, minutes float64) {
		if minutes < 1 {
			return
		}
		end := at.Add(time.Duration(minutes * float64(time.Minute)))
		if n := len(spells); n > 0 && spells[n-1].Stage == st {
			spells[n-1].End = end
		} else {
			spells = append(spells, hypnogram.Spell{Stage: st, Start: at, End: end})
		}
		at = end
	}
	jitter := func(m float64) float64 { return m * (0.75 + rng.Float64()*0.5) }
	add(hypnogram.Awake, jitter(12))
	for cycle := range 5 {
		k := float64(cycle)
		add(hypnogram.Core, jitter(22))
		add(hypnogram.Deep, jitter(max(0, 38-11*k)))
		add(hypnogram.Core, jitter(18+4*k))
		if rng.Float64() < 0.4 {
			add(hypnogram.Awake, jitter(3))
			add(hypnogram.Core, jitter(6))
		}
		add(hypnogram.REM, jitter(8+7*k))
		if cycle < 4 && rng.Float64() < 0.5 {
			add(hypnogram.Awake, jitter(2))
		}
	}
	add(hypnogram.Awake, jitter(6))
	return spells
}

type app struct {
	seed  uint64
	sleep hypnogram.State
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(6)).Gap(t.Space(6)).Children(func() {
		hypnogram.View(c, &a.sleep, night(a.seed))
		if ui.Button(c, "Another night").AlignSelf(ui.Start).Clicked() {
			a.seed++
		}
	})
}

func main() {
	a := &app{seed: 1}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Hypnogram",
			Width:   820,
			Height:  560,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
