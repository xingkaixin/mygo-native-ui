// Waffle shows where a made-up country's electricity came from in 2015,
// 2020 and 2025: coal giving way to wind and solar. Choose a year, and
// hover a source to light its squares.
//
//	go run ./examples/waffle
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/waffle"
)

var (
	coal    = ui.Hex("#57534e")
	gas     = ui.Hex("#f97316")
	nuclear = ui.Hex("#a855f7")
	hydro   = ui.Hex("#0ea5e9")
	wind    = ui.Hex("#14b8a6")
	solar   = ui.Hex("#eab308")
)

var years = []string{"2015", "2020", "2025"}

// mix returns the parts of a year's electricity, in percent by source.
func mix(c, g, n, h, w, s float64) []waffle.Part {
	return []waffle.Part{
		{Name: "Coal", Value: c, Color: coal},
		{Name: "Gas", Value: g, Color: gas},
		{Name: "Nuclear", Value: n, Color: nuclear},
		{Name: "Hydro", Value: h, Color: hydro},
		{Name: "Wind", Value: w, Color: wind},
		{Name: "Solar", Value: s, Color: solar},
	}
}

var mixes = [][]waffle.Part{
	mix(41.3, 24.6, 17.2, 9.8, 5.1, 2.0),
	mix(29.7, 27.4, 16.1, 9.5, 11.2, 6.1),
	mix(14.8, 23.0, 15.4, 9.9, 20.6, 16.3),
}

type app struct {
	year int
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(6)).Gap(t.Space(5)).Children(func() {
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "Electricity by source").Bold().FontSize(t.FontSize + 4).Grow(1)
			ui.Segmented(c, &a.year, years...).Label("Year")
		})
		waffle.View(c, mixes[a.year]).Height(280)
	})
}

func main() {
	a := &app{year: 2}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Waffle chart",
			Width:   580,
			Height:  400,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
