// Chord shows made-up flows as a chord diagram: people moving between ten
// regions of the world in a year, or messages between ten services of a
// shop in a minute. Point at a group or a ribbon to read it, click a
// group to keep it lit, and press Esc to let go.
//
//	go run ./examples/chord
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/chord"
)

// dataset is what the diagram shows: its title, a line about it, the
// groups, the flows between them, from row to column, and their unit.
type dataset struct {
	title, about string
	names        []string
	flows        [][]float64
	unit         string
}

var datasets = []dataset{
	{
		title: "Migration, 2025",
		about: "People who moved between regions in a year, and within the three where most move across borders",
		names: []string{
			"North America", "Latin America", "Western Europe", "Eastern Europe", "Middle East",
			"North Africa", "Sub-Saharan Africa", "South Asia", "East Asia", "Oceania",
		},
		flows: thousands([][]float64{
			{0, 90, 120, 15, 20, 5, 10, 40, 60, 35},
			{1100, 150, 210, 10, 5, 2, 3, 5, 25, 15},
			{180, 60, 420, 70, 40, 30, 25, 20, 30, 70},
			{90, 8, 650, 0, 15, 3, 2, 5, 8, 10},
			{120, 5, 260, 30, 0, 40, 20, 60, 15, 20},
			{60, 3, 380, 10, 210, 0, 30, 2, 5, 5},
			{150, 5, 320, 10, 160, 70, 380, 10, 25, 20},
			{380, 5, 330, 15, 900, 8, 30, 0, 60, 120},
			{420, 30, 180, 40, 50, 5, 40, 30, 0, 140},
			{40, 3, 60, 2, 5, 1, 3, 10, 30, 0},
		}),
		unit: "people",
	},
	{
		title: "Messages between services",
		about: "Requests and events a shop's services sent one another in the last minute",
		names: []string{"Gateway", "Auth", "Users", "Catalog", "Search", "Cart", "Orders", "Payments", "Events", "Notify"},
		flows: [][]float64{
			{0, 4200, 1800, 6400, 3900, 2600, 900, 0, 0, 0},
			{0, 0, 3800, 0, 0, 0, 0, 0, 300, 0},
			{0, 0, 0, 0, 0, 0, 0, 0, 400, 0},
			{0, 0, 0, 0, 1200, 0, 0, 0, 0, 0},
			{0, 0, 0, 2100, 0, 0, 0, 0, 0, 0},
			{0, 0, 0, 2400, 0, 0, 700, 0, 500, 0},
			{0, 0, 600, 900, 0, 0, 0, 850, 1600, 0},
			{0, 0, 0, 0, 0, 0, 790, 0, 820, 0},
			{0, 0, 0, 0, 1500, 0, 400, 0, 0, 3100},
			{0, 0, 1200, 0, 0, 0, 0, 0, 0, 0},
		},
		unit: "messages",
	},
}

// thousands returns flows given in thousands as they are.
func thousands(flows [][]float64) [][]float64 {
	for _, row := range flows {
		for j := range row {
			row[j] *= 1000
		}
	}
	return flows
}

type app struct {
	shown int
	chord chord.State
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	d := datasets[a.shown]
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(4), t.Space(5)).Gap(t.Space(3)).Children(func() {
		ui.Row(c).Gap(t.Space(4)).AlignItems(ui.Center).Children(func() {
			ui.Column(c).Gap(t.Space(0.5)).Grow(1).Shrink(1).Children(func() {
				ui.Text(c, d.title).Bold().FontSize(t.FontSize + 4)
				ui.Text(c, d.about).TextColor(t.TextMuted).SingleLine()
			})
			ui.Segmented(c, &a.shown, "Migration", "Services").Label("Data")
		})
		chord.View(c, &a.chord, d.names, d.flows, d.unit).Grow(1).AlignSelf(ui.Stretch)
	})
}

func main() {
	a := &app{}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Chord diagram",
			Width:   1100,
			Height:  820,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
