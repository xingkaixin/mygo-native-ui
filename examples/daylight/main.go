// Daylight follows the sun over four cities, from the equator's even days
// to the midnight sun of the Arctic: choose a city, and a day on its year.
//
//	go run ./examples/daylight
package main

import (
	"log"
	"time"
	_ "time/tzdata"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/daylight"
)

type city struct {
	name  string
	place daylight.Place
	zone  string
}

var cities = []city{
	{"Singapore", daylight.Place{Lat: 1.29, Lon: 103.85}, "Asia/Singapore"},
	{"Shanghai", daylight.Place{Lat: 31.23, Lon: 121.47}, "Asia/Shanghai"},
	{"Reykjavík", daylight.Place{Lat: 64.15, Lon: -21.94}, "Atlantic/Reykjavik"},
	{"Tromsø", daylight.Place{Lat: 69.65, Lon: 18.96}, "Europe/Oslo"},
}

type app struct {
	sun  daylight.State
	city int
	day  time.Time
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Scroll(c).Fill().Background(t.Background).Children(func() {
		ui.Column(c).Padding(t.Space(8)).Gap(t.Space(5)).Children(func() {
			ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
				ui.Text(c, a.day.Format("Monday, January 2, 2006")).FontSize(t.FontSize + 4).Bold().Grow(1)
				names := make([]string, len(cities))
				for i, ct := range cities {
					names[i] = ct.name
				}
				if ui.Segmented(c, &a.city, names...).Label("City").Changed() {
					a.day = at(a.day, cities[a.city])
				}
			})
			daylight.View(c, &a.sun, cities[a.city].place, &a.day)
		})
	})
}

// at returns the noon of day's date in ct.
func at(day time.Time, ct city) time.Time {
	loc, err := time.LoadLocation(ct.zone)
	if err != nil {
		log.Fatal(err)
	}
	y, m, d := day.Date()
	return time.Date(y, m, d, 12, 0, 0, 0, loc)
}

func main() {
	a := &app{city: 2}
	a.day = at(time.Now(), cities[a.city])
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Daylight",
			Width:   900,
			Height:  720,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
