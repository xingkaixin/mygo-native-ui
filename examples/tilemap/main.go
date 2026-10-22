// Tilemap shows OpenStreetMap's map with a few cities marked. Choose a
// city in the list, or click its marker, to fly there; click the map to
// see where you clicked. Drag to pan, scroll to zoom.
//
//	go run ./examples/tilemap
package main

import (
	"fmt"
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/tilemap"
)

type city struct {
	name, country string
	at            tilemap.LatLon
}

var cities = []city{
	{"London", "United Kingdom", tilemap.LatLon{Lat: 51.5074, Lon: -0.1278}},
	{"Paris", "France", tilemap.LatLon{Lat: 48.8566, Lon: 2.3522}},
	{"New York", "United States", tilemap.LatLon{Lat: 40.7128, Lon: -74.0060}},
	{"Rio de Janeiro", "Brazil", tilemap.LatLon{Lat: -22.9068, Lon: -43.1729}},
	{"Cape Town", "South Africa", tilemap.LatLon{Lat: -33.9249, Lon: 18.4241}},
	{"Tokyo", "Japan", tilemap.LatLon{Lat: 35.6762, Lon: 139.6503}},
	{"Sydney", "Australia", tilemap.LatLon{Lat: -33.8688, Lon: 151.2093}},
}

// app is the map's place and zoom, the view's state, the city chosen, -1
// for none, and the place clicked last.
type app struct {
	tiles   *tilemap.Tiles
	view    tilemap.State
	center  tilemap.LatLon
	zoom    float64
	markers []tilemap.Marker
	chosen  int
	clicked *tilemap.LatLon
}

func (a *app) build(c *ui.Context) {
	t := c.Theme()
	ui.Row(c).Fill().AlignItems(ui.Stretch).Background(t.Background).Children(func() {
		ui.Column(c).Width(t.Space(56)).Padding(t.Space(3)).Gap(t.Space(0.5)).Children(func() {
			ui.Text(c, "Cities").Bold().Padding(0, t.Space(2), t.Space(2))
			for i, ct := range cities {
				row := ui.ButtonBase(c).Justify(ui.Start).Padding(t.Space(1.5), t.Space(2)).Radius(t.Radius)
				if i == a.chosen {
					row.Background(t.Accent)
				} else if row.Hovered() {
					row.Background(t.SurfaceHover)
				}
				row.Children(func() {
					ui.Column(c).Gap(t.Space(0.5)).Children(func() {
						name, country := ui.Text(c, ct.name), ui.Text(c, ct.country).FontSize(t.FontSize-1)
						if i == a.chosen {
							name.TextColor(t.AccentText)
							country.TextColor(t.AccentText.Alpha(0.8))
						} else {
							country.TextColor(t.TextMuted)
						}
					})
				})
				if row.Clicked() {
					a.fly(c, i)
				}
			}
		})
		ui.Divider(c)
		ui.Column(c).Grow(1).Children(func() {
			ui.Column(c).Grow(1).Children(func() {
				ev := tilemap.View(c, &a.view, &a.center, &a.zoom, a.tiles, a.markers)
				if i, ok := ev.Marker(); ok {
					a.fly(c, i)
				}
				if at, ok := ev.Clicked(); ok {
					a.clicked = &at
				}
			})
			ui.Divider(c)
			ui.Row(c).Padding(t.Space(1.5), t.Space(3)).Gap(t.Space(4)).Children(func() {
				status := "Click the map to see where"
				if a.clicked != nil {
					status = "Clicked " + degrees(*a.clicked)
				}
				ui.Text(c, status).FontSize(t.FontSize - 1).FontFeatures("tnum").SingleLine().Grow(1).Shrink(1)
				ui.Text(c, fmt.Sprintf("%s  ·  zoom %.1f", degrees(a.center), a.zoom)).
					FontSize(t.FontSize - 1).FontFeatures("tnum").TextColor(t.TextMuted).SingleLine()
			})
		})
	})
}

func (a *app) fly(c *ui.Context, i int) {
	a.chosen = i
	a.view.FlyTo(c, cities[i].at, 11)
}

// degrees formats p as "51.5074° N, 0.1278° W".
func degrees(p tilemap.LatLon) string {
	ns, ew := "N", "E"
	if p.Lat < 0 {
		ns = "S"
	}
	if p.Lon < 0 {
		ew = "W"
	}
	return fmt.Sprintf("%.4f° %s, %.4f° %s", abs(p.Lat), ns, abs(p.Lon), ew)
}

func abs(v float64) float64 { return max(v, -v) }

func newApp() *app {
	a := &app{
		tiles: &tilemap.Tiles{
			URL:            tilemap.OpenStreetMap,
			Attribution:    "© OpenStreetMap contributors",
			AttributionURL: "https://www.openstreetmap.org/copyright",
			UserAgent:      "mygo-native-ui-tilemap-example/1.0 (+https://github.com/xingkaixin/mygo-native-ui)",
		},
		center: tilemap.LatLon{Lat: 30, Lon: 10},
		zoom:   2,
		chosen: -1,
	}
	for _, ct := range cities {
		a.markers = append(a.markers, tilemap.Marker{At: ct.at, Label: ct.name})
	}
	return a
}

func main() {
	a := newApp()
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Cities",
			Width:   1100,
			Height:  700,
			Content: ui.View(a.build),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
