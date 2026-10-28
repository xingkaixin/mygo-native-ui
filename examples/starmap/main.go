// Starmap shows tonight's sky over a city chosen above it, the time
// beside it: drag across the sky, or press ← and →, to watch it turn.
//
//	go run ./examples/starmap
package main

import (
	"log"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/starmap"
)

var cities = []struct {
	name  string
	place starmap.Place
	zone  string
}{
	{"Shanghai", starmap.Place{Lat: 31.23, Lon: 121.47}, "Asia/Shanghai"},
	{"Reykjavík", starmap.Place{Lat: 64.15, Lon: -21.94}, "Atlantic/Reykjavik"},
	{"Quito", starmap.Place{Lat: -0.18, Lon: -78.47}, "America/Guayaquil"},
	{"Sydney", starmap.Place{Lat: -33.87, Lon: 151.21}, "Australia/Sydney"},
}

type app struct {
	sky  starmap.State
	city int
	at   time.Time
}

func (a *app) tonight() {
	loc, err := time.LoadLocation(cities[a.city].zone)
	if err != nil {
		loc = time.UTC
	}
	now := time.Now().In(loc)
	a.at = time.Date(now.Year(), now.Month(), now.Day(), 22, 0, 0, 0, loc)
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Padding(t.Space(6)).Gap(t.Space(4)).Background(t.Background).Children(func() {
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			names := make([]string, len(cities))
			for i, city := range cities {
				names[i] = city.name
			}
			if ui.Segmented(c, &a.city, names...).Label("City").Changed() {
				a.tonight()
			}
			ui.Box(c).Grow(1)
			ui.Text(c, a.at.Format("Mon Jan 2, 15:04")).FontFeatures("tnum")
			if ui.Button(c, "Tonight").Clicked() {
				a.tonight()
			}
		})
		starmap.View(c, &a.sky, cities[a.city].place, &a.at).Grow(1).AutoFocus()
	})
}

func main() {
	a := &app{}
	a.tonight()
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Star Map",
			Width:   760,
			Height:  820,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
