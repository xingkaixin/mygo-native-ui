// Metro rides line 2 of a harbour city from Riverside toward the airport,
// a minute every three seconds: choose a stop ahead to get off at.
//
//	go run ./examples/metro
package main

import (
	"log"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/metro"
)

var (
	line1 = metro.Line{Name: "1", Color: ui.Hex("#e4002b")}
	line2 = metro.Line{Name: "2", Color: ui.Hex("#00a650")}
	line4 = metro.Line{Name: "4", Color: ui.Hex("#5f259f")}
	line5 = metro.Line{Name: "5", Color: ui.Hex("#f39800")}
	ferry = metro.Line{Name: "F", Color: ui.Hex("#0072bc")}
)

var stops = []metro.Stop{
	{Name: "Riverside", Minutes: 0},
	{Name: "Mill Lane", Minutes: 2},
	{Name: "Old Market", Minutes: 4},
	{Name: "Central", Transfers: []metro.Line{line1, line4}, Minutes: 7},
	{Name: "Museum", Minutes: 9},
	{Name: "University", Minutes: 11},
	{Name: "Botanic Garden", Minutes: 14},
	{Name: "Stadium", Transfers: []metro.Line{line5}, Minutes: 16},
	{Name: "Harbour", Transfers: []metro.Line{ferry}, Minutes: 19},
	{Name: "Shipyard", Minutes: 22},
	{Name: "Lighthouse", Minutes: 25},
	{Name: "Cargo Terminal", Minutes: 29},
	{Name: "Airport", Minutes: 33},
}

// dwell is how long the train waits at a stop, in minutes of the ride.
const dwell = 0.5

type app struct {
	ride  metro.State
	to    int
	start time.Time
}

// train returns where the train is, m minutes into the ride.
func train(m float64) float64 {
	for i := 0; i < len(stops)-1; i++ {
		m -= dwell
		if m <= 0 {
			return float64(i)
		}
		seg := stops[i+1].Minutes - stops[i].Minutes
		if m < seg {
			return float64(i) + m/seg
		}
		m -= seg
	}
	return float64(len(stops) - 1)
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	if a.start.IsZero() {
		a.start = c.Now()
	}
	m := c.Now().Sub(a.start).Seconds() / 3
	at := train(m)
	if at >= float64(len(stops)-1) && m > stops[len(stops)-1].Minutes+float64(len(stops))*dwell+3 {
		a.start, a.ride, a.to = c.Now(), metro.State{}, 8
	}
	c.After(time.Second / 30)
	ui.Scroll(c).Fill().Background(t.Background).Children(func() {
		ui.Column(c).Padding(t.Space(8)).MaxWidth(t.Space(120)).Children(func() {
			metro.View(c, &a.ride, line2, stops, at, &a.to)
		})
	})
}

func main() {
	a := &app{to: 8}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Metro",
			Width:   520,
			Height:  760,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
