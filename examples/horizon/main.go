// Horizon shows an hour of twelve hosts' CPU load, on one scale to 100%,
// one host going hot halfway through, and of how far each host's latency
// is from its usual, above and below, each on its own scale.
//
//	go run ./examples/horizon
package main

import (
	"fmt"
	"log"
	"math"
	"math/rand/v2"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/horizon"
)

type app struct {
	start   time.Time
	tab     int
	cpu     []horizon.Series
	latency []horizon.Series
}

func newApp(now time.Time) *app {
	r := rand.New(rand.NewPCG(3, 4))
	a := &app{start: now.Truncate(time.Minute).Add(-time.Hour)}
	for h := range 12 {
		cpu := make([]float64, 360)
		lat := make([]float64, 360)
		level, drift := 20+r.Float64()*30, 0.0
		for i := range cpu {
			level += r.NormFloat64() * 2
			level = max(5, min(level, 70))
			v := level + 8*math.Sin(float64(i)/20+float64(h))
			if h == 4 && i > 170 && i < 260 {
				v += 40
			}
			cpu[i] = max(0, min(v, 100))
			drift = drift*0.95 + r.NormFloat64()*6
			lat[i] = drift + 10*math.Sin(float64(i)/35+float64(h)*0.7)
		}
		name := fmt.Sprintf("web-%02d", h+1)
		a.cpu = append(a.cpu, horizon.Series{Name: name, Values: cpu})
		a.latency = append(a.latency, horizon.Series{Name: name, Values: lat})
	}
	return a
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(16).Gap(12).Children(func() {
		ui.Segmented(c, &a.tab, "CPU %", "Latency vs usual (ms)").Label("Metric")
		if a.tab == 0 {
			horizon.View(c, a.start, 10*time.Second, a.cpu, 4, 100)
		} else {
			horizon.View(c.Key(1), a.start, 10*time.Second, a.latency, 3, 0)
		}
	})
}

func main() {
	a := newApp(time.Now())
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Hosts",
			Width:   900,
			Height:  560,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
