// Avionics flies a Cessna 172 through gentle climbing and descending
// turns on its own, its primary flight display following, until you take
// the controls.
//
//	go run ./examples/avionics
package main

import (
	"log"
	"math"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/avionics"
)

type app struct {
	f      avionics.Flight
	manual bool
	start  time.Time
	last   time.Time
}

func (a *app) fly(now time.Time) {
	if a.start.IsZero() {
		a.start, a.last = now, now
	}
	dt := now.Sub(a.last).Seconds()
	a.last = now
	if !a.manual {
		t := now.Sub(a.start).Seconds()
		a.f.Roll = 25 * math.Sin(t/9)
		a.f.Pitch = 2 + 4*math.Sin(t/13)
		a.f.Airspeed = 108 - 2.5*a.f.Pitch + 4*math.Sin(t/7)
	}
	// A coordinated turn's rate in degrees a second is 1091 tan(bank)
	// over the true airspeed in knots; a knot is 101.27 feet a minute.
	if a.f.Airspeed > 0 {
		a.f.Heading = math.Mod(a.f.Heading+1091*math.Tan(a.f.Roll*math.Pi/180)/a.f.Airspeed*dt+360, 360)
	}
	a.f.VerticalSpeed = a.f.Airspeed * 101.27 * math.Sin((a.f.Pitch-2)*math.Pi/180)
	a.f.Altitude = max(0, a.f.Altitude+a.f.VerticalSpeed/60*dt)
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	a.fly(c.Now())
	c.After(time.Second / 60)
	ui.Column(c).Fill().Background(t.Background).Children(func() {
		avionics.View(c, a.f, avionics.C172).Grow(1)
		ui.Row(c).Gap(t.Space(4)).Padding(t.Space(3)).AlignItems(ui.Center).Children(func() {
			ui.Checkbox(c, &a.manual, "Hand-fly")
			if a.manual {
				for _, s := range []struct {
					label  string
					value  *float64
					lo, hi float64
				}{{"Pitch", &a.f.Pitch, -15, 20}, {"Bank", &a.f.Roll, -60, 60}, {"Speed", &a.f.Airspeed, 0, 180}} {
					ui.Row(c.Key(s.label)).Gap(t.Space(2)).AlignItems(ui.Center).Grow(1).Children(func() {
						ui.Text(c, s.label).TextColor(t.TextMuted)
						ui.Slider(c, s.value, s.lo, s.hi).Grow(1)
					})
				}
			}
		})
	})
}

func main() {
	a := &app{f: avionics.Flight{Heading: 270, Airspeed: 105, Altitude: 4500}}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Avionics",
			Width:   900,
			Height:  720,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
