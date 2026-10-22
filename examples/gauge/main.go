// Gauge is a car's instruments and a computer's load, moving as if live:
// speed, engine speed with its red line, coolant temperature on a
// semicircle, and CPU load with its peak held. Drag the slider to drive;
// past 160 mph the needle pins at its end, and high revs heat the engine.
//
//	go run ./examples/gauge
package main

import (
	"fmt"
	"log"
	"math/rand/v2"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/gauge"
)

const tick = 100 * time.Millisecond

type app struct {
	speed, rpm, coolant, cpu float64
	burst                    int
	cpuPeak                  gauge.State
	rng                      *rand.Rand
	last                     time.Time
}

// step moves the readings on by a tick: the engine follows the speed, the
// coolant warms toward a heat that rises with the revs, and the CPU idles
// between bursts of work.
func (a *app) step() {
	a.rpm = 800 + 42*a.speed + 40*a.rng.NormFloat64()
	a.coolant += (84+a.rpm/250-a.coolant)*0.03 + 0.2*a.rng.NormFloat64()
	target := 18.0
	switch {
	case a.burst > 0:
		a.burst--
		target = 84
	case a.rng.Float64() < 0.04:
		a.burst = 5 + a.rng.IntN(15)
	}
	a.cpu = max(1, min(100, a.cpu+(target-a.cpu)*0.35+6*a.rng.NormFloat64()))
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	now := c.Now()
	if a.last.IsZero() {
		a.last = now
	}
	for ; now.Sub(a.last) >= tick; a.last = a.last.Add(tick) {
		a.step()
	}
	c.After(a.last.Add(tick).Sub(now))

	gauges := []struct {
		peak  *gauge.State
		value float64
		g     gauge.Gauge
	}{
		{nil, a.speed, gauge.Gauge{Label: "Speed", Unit: "mph", Max: 160}},
		{nil, a.rpm, gauge.Gauge{Label: "Engine", Unit: "rpm", Max: 8000, Zones: []gauge.Zone{
			{From: 6000, To: 6500, Color: t.Warning},
			{From: 6500, To: 8000, Color: t.Danger},
		}}},
		{nil, a.coolant, gauge.Gauge{Label: "Coolant", Unit: "°C", Min: 40, Max: 130, Sweep: 180, Zones: []gauge.Zone{
			{From: 40, To: 70, Color: t.Accent},
			{From: 70, To: 105, Color: t.Success},
			{From: 105, To: 130, Color: t.Danger},
		}}},
		{&a.cpuPeak, a.cpu, gauge.Gauge{Label: "CPU", Unit: "%", Max: 100, Zones: []gauge.Zone{
			{To: 70, Color: t.Success},
			{From: 70, To: 90, Color: t.Warning},
			{From: 90, To: 100, Color: t.Danger},
		}}},
	}

	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(6)).Gap(t.Space(5)).Children(func() {
		ui.Row(c).Gap(t.Space(4)).AlignItems(ui.Stretch).Children(func() {
			for i, it := range gauges {
				ui.Column(c.Key(i)).Grow(1).Basis(0).Padding(t.Space(4)).Radius(t.Space(3)).Justify(ui.Center).
					Background(t.Surface).Border(1, t.Border).Children(func() {
					gauge.View(c, it.peak, it.value, it.g).FillWidth()
				})
			}
		})
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "Speed").TextColor(t.TextMuted)
			ui.Slider(c, &a.speed, 0, 180).Label("Speed").Grow(1)
			ui.Text(c, fmt.Sprintf("%.0f mph", a.speed)).FontFeatures("tnum").Width(t.Space(16))
			if ui.Button(c, "Reset CPU peak").Clicked() {
				a.cpuPeak.Reset()
			}
		})
	})
}

func main() {
	a := &app{speed: 45, coolant: 46, cpu: 20, rng: rand.New(rand.NewPCG(1, 2))}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Gauge",
			Width:   960,
			Height:  280,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
