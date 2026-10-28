// Thermostat sets the heating of two rooms, the living room heating to
// its target and the bedroom kept in a range, each room's temperature
// creeping toward what its thermostat asks for.
//
//	go run ./examples/thermostat
package main

import (
	"log"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/thermostat"
)

type room struct {
	name string
	temp float64
	dial thermostat.State
	set  thermostat.Thermostat
}

type app struct {
	rooms []*room
	last  time.Time
}

// drift moves each room a tenth of a degree toward its target every two
// seconds while its thermostat heats or cools it.
func (a *app) drift(now time.Time) {
	for now.Sub(a.last) >= 2*time.Second {
		a.last = a.last.Add(2 * time.Second)
		for _, r := range a.rooms {
			s := r.set
			switch {
			case (s.Mode == thermostat.Heat || s.Mode == thermostat.Auto) && r.temp < s.Heat-0.05:
				r.temp += 0.1
			case (s.Mode == thermostat.Cool || s.Mode == thermostat.Auto) && r.temp > s.Cool+0.05:
				r.temp -= 0.1
			}
		}
	}
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	now := c.Now()
	if a.last.IsZero() {
		a.last = now
	}
	a.drift(now)
	c.After(2*time.Second - now.Sub(a.last))
	ui.Row(c).Fill().Background(t.Background).Padding(t.Space(10)).Gap(t.Space(12)).Justify(ui.Center).Children(func() {
		for i, r := range a.rooms {
			ui.Column(c.Key(i)).Gap(t.Space(4)).AlignItems(ui.Center).Children(func() {
				ui.Text(c, r.name).FontSize(t.FontSize + 4).Bold()
				thermostat.View(c, &r.dial, r.temp, &r.set).Width(300)
			})
		}
	})
}

func main() {
	a := &app{rooms: []*room{
		{name: "Living room", temp: 18.6, set: thermostat.Thermostat{Mode: thermostat.Heat, Heat: 21, Cool: 25}},
		{name: "Bedroom", temp: 22.4, set: thermostat.Thermostat{Mode: thermostat.Auto, Heat: 18, Cool: 21.5}},
	}}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Thermostat",
			Width:   860,
			Height:  580,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
