// Marey draws the morning on a made-up commuter line: locals every 20
// minutes each way stopping everywhere, hourly expresses stopping at four,
// and a freight that waits in a siding at Millbrook for an express to
// pass. Hover over a train for its times, scroll to pan, and pinch or
// Cmd-scroll to zoom.
//
//	go run ./examples/marey
package main

import (
	"fmt"
	"log"
	"slices"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/marey"
)

var stations = []marey.Station{
	{Name: "Harbor Terminal", At: 0}, {Name: "Fish Market", At: 2.1}, {Name: "Old Town", At: 4.6}, {Name: "University", At: 8.3}, {Name: "Millbrook", At: 13.5},
	{Name: "Cedar Park", At: 17.2}, {Name: "Airport Junction", At: 23.8}, {Name: "Hillcrest", At: 27.4}, {Name: "North Valley", At: 33},
}

// run returns a train leaving the station at the end it starts from at
// start, at kmh between stops, waiting dwell at each station it stops at.
func run(name, service string, start time.Duration, north bool, kmh float64, dwell time.Duration, stops []int, wait map[int]time.Duration) marey.Train {
	order := stops
	if !north {
		order = slices.Clone(stops)
		slices.Reverse(order)
	}
	tr := marey.Train{Name: name, Service: service}
	at := start
	for i, st := range order {
		if i > 0 {
			km := stations[st].At - stations[order[i-1]].At
			if km < 0 {
				km = -km
			}
			at += time.Duration(km / kmh * float64(time.Hour))
		}
		d := dwell
		if i == 0 || i == len(order)-1 {
			d = 0
		}
		d += wait[st]
		tr.Stops = append(tr.Stops, marey.Stop{Station: st, Arrive: at, Depart: at + d})
		at += d
	}
	return tr
}

func trains() []marey.Train {
	all := []int{0, 1, 2, 3, 4, 5, 6, 7, 8}
	fast := []int{0, 3, 6, 8}
	h := func(hh, mm int) time.Duration { return time.Duration(hh)*time.Hour + time.Duration(mm)*time.Minute }
	var out []marey.Train
	n := 101
	for t := h(5, 40); t <= h(9, 40); t += 20 * time.Minute {
		out = append(out,
			run(fmt.Sprintf("Local %d", n), "Local", t, true, 52, time.Minute, all, nil),
			run(fmt.Sprintf("Local %d", n+1), "Local", t+10*time.Minute, false, 52, time.Minute, all, nil))
		n += 2
	}
	n = 11
	for t := h(6, 5); t <= h(9, 5); t += time.Hour {
		out = append(out,
			run(fmt.Sprintf("Express %d", n), "Express", t, true, 95, 2*time.Minute, fast, nil),
			run(fmt.Sprintf("Express %d", n+1), "Express", t+30*time.Minute, false, 95, 2*time.Minute, fast, nil))
		n += 2
	}
	out = append(out, run("Freight 7", "Freight", h(6, 20), true, 38, 0, all, map[int]time.Duration{4: 14 * time.Minute}))
	return out
}

type app struct {
	trains []marey.Train
	graph  marey.State
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(6)).Gap(t.Space(3)).Children(func() {
		ui.Text(c, "Harbor Line, weekday morning").FontWeight(600).FontSize(t.FontSize + 2)
		marey.View(c, &a.graph, stations, a.trains).Grow(1)
	})
}

func main() {
	a := &app{trains: trains()}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Marey",
			Width:   1040,
			Height:  640,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
