// Meteogram forecasts three days in a mountain town: a mild day, a front
// that brings rain and a thunderstorm, and a clear frosty night before
// snow showers. Hover the chart for an hour's numbers, and switch between
// two days and three.
//
//	go run ./examples/meteogram
package main

import (
	"fmt"
	"log"
	"math"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/meteogram"
)

// forecast makes up 72 hours from start.
func forecast(start time.Time) []meteogram.Hour {
	var hours []meteogram.Hour
	for i := range 72 {
		at := start.Add(time.Duration(i) * time.Hour)
		h := float64(at.Hour())
		// The day warms toward 15:00 and cools toward dawn, and each is
		// colder than the last as the front passes.
		temp := 6 - 4*float64(i)/24 + 5*math.Sin((h-9)/24*2*math.Pi)
		hour := meteogram.Hour{
			Time:     at,
			Temp:     math.Round(temp*10) / 10,
			Wind:     3 + 2*math.Sin(float64(i)/7),
			WindFrom: 200 + 40*math.Sin(float64(i)/11),
			Night:    h < 7 || h >= 18,
			Sky:      meteogram.PartlyCloudy,
		}
		switch {
		case i < 8:
			hour.Sky = meteogram.Clear
		case i < 14:
			hour.Sky = meteogram.PartlyCloudy
		case i < 20:
			hour.Sky = meteogram.Cloudy
		case i < 34:
			hour.Sky = meteogram.Rain
			hour.Precip = 0.6 + 1.8*math.Sin(float64(i-20)/14*math.Pi)
			hour.Wind += 5
		case i < 37:
			hour.Sky = meteogram.Thunder
			hour.Precip = 4.2 - float64(i-34)
			hour.Wind += 8
		case i < 40:
			hour.Sky = meteogram.Cloudy
			hour.WindFrom = 310
		case i < 50:
			hour.Sky = meteogram.Clear
			hour.WindFrom = 330
		case i < 54:
			hour.Sky = meteogram.Fog
		case i < 62:
			hour.Sky = meteogram.Snow
			hour.Precip = 0.4 + 0.3*math.Sin(float64(i))
		}
		hour.Wind = math.Round(hour.Wind*10) / 10
		hour.Precip = math.Round(max(0, hour.Precip)*10) / 10
		hours = append(hours, hour)
	}
	return hours
}

type app struct {
	hours []meteogram.Hour
	span  int
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(6)).Gap(t.Space(4)).Children(func() {
		now := a.hours[0]
		ui.Row(c).Gap(t.Space(4)).AlignItems(ui.End).Children(func() {
			ui.Column(c).Gap(t.Space(1)).Grow(1).Children(func() {
				ui.Text(c, "Hallstatt").FontSize(t.FontSize + 8).Bold()
				ui.Text(c, fmt.Sprintf("%s, %.0f° now · wind %.0f m/s", now.Sky, now.Temp, now.Wind)).TextColor(t.TextMuted)
			})
			ui.Segmented(c, &a.span, "48 hours", "3 days").Label("Span")
		})
		hours := a.hours
		if a.span == 0 {
			hours = hours[:48]
		}
		meteogram.View(c, hours).Grow(1)
	})
}

func main() {
	start := time.Date(2026, 10, 28, 0, 0, 0, 0, time.Local)
	a := &app{hours: forecast(start), span: 1}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Forecast",
			Width:   1000,
			Height:  480,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
