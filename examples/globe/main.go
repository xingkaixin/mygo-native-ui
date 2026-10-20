// Globe shows flights from Dubai to 20 cities on a globe, as great-circle
// arcs from the hub. Drag to turn the globe and flick it to spin it, scroll
// to zoom, and point at a country to see its name.
//
//	go run ./examples/globe
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/globe"
)

var hub = globe.Marker{Place: globe.Place{Lat: 25.25, Lon: 55.36}, Label: "Dubai"}

// cities are where the flights go, at their main airports.
var cities = []globe.Marker{
	{Place: globe.Place{Lat: 51.47, Lon: -0.45}, Label: "London"},
	{Place: globe.Place{Lat: 49.01, Lon: 2.55}, Label: "Paris"},
	{Place: globe.Place{Lat: 50.03, Lon: 8.57}, Label: "Frankfurt"},
	{Place: globe.Place{Lat: 55.41, Lon: 37.9}, Label: "Moscow"},
	{Place: globe.Place{Lat: 41.26, Lon: 28.74}, Label: "Istanbul"},
	{Place: globe.Place{Lat: 30.12, Lon: 31.41}, Label: "Cairo"},
	{Place: globe.Place{Lat: 6.58, Lon: 3.32}, Label: "Lagos"},
	{Place: globe.Place{Lat: -1.32, Lon: 36.93}, Label: "Nairobi"},
	{Place: globe.Place{Lat: -26.14, Lon: 28.24}, Label: "Johannesburg"},
	{Place: globe.Place{Lat: 19.09, Lon: 72.87}, Label: "Mumbai"},
	{Place: globe.Place{Lat: 28.56, Lon: 77.1}, Label: "Delhi"},
	{Place: globe.Place{Lat: 13.69, Lon: 100.75}, Label: "Bangkok"},
	{Place: globe.Place{Lat: 1.36, Lon: 103.99}, Label: "Singapore"},
	{Place: globe.Place{Lat: 22.31, Lon: 113.91}, Label: "Hong Kong"},
	{Place: globe.Place{Lat: 40.08, Lon: 116.58}, Label: "Beijing"},
	{Place: globe.Place{Lat: 35.55, Lon: 139.78}, Label: "Tokyo"},
	{Place: globe.Place{Lat: -33.95, Lon: 151.18}, Label: "Sydney"},
	{Place: globe.Place{Lat: 40.64, Lon: -73.78}, Label: "New York"},
	{Place: globe.Place{Lat: 33.94, Lon: -118.41}, Label: "Los Angeles"},
	{Place: globe.Place{Lat: -23.43, Lon: -46.47}, Label: "São Paulo"},
}

type app struct {
	markers []globe.Marker
	routes  []globe.Arc
	globe   globe.State
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Children(func() {
		ui.Column(c).Gap(t.Space(0.5)).Padding(t.Space(3), t.Space(4)).Children(func() {
			ui.Text(c, "Routes from Dubai").Bold().FontSize(t.FontSize + 4)
			ui.Text(c, "20 nonstop destinations · drag to turn, scroll to zoom").TextColor(t.TextMuted).SingleLine()
		})
		ui.Divider(c)
		globe.View(c, &a.globe, a.markers, a.routes).Grow(1)
	})
}

func main() {
	a := &app{markers: append([]globe.Marker{hub}, cities...)}
	for _, city := range cities {
		a.routes = append(a.routes, globe.Arc{From: hub.Place, To: city.Place})
	}
	a.globe.LookAt(globe.Place{Lat: 30, Lon: 40})
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Routes",
			Width:   900,
			Height:  760,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
