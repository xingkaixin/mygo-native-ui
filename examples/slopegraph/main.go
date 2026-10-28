// Slopegraph compares each service's build time before and after a
// build cache, as a slope graph or a dumbbell chart.
//
//	go run ./examples/slopegraph
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/slopegraph"
)

var builds = []slopegraph.Item{
	{Name: "web", Before: 120, After: 45},
	{Name: "search", Before: 95, After: 41},
	{Name: "api", Before: 84, After: 31},
	{Name: "billing", Before: 73, After: 70},
	{Name: "auth", Before: 58, After: 24},
	{Name: "worker", Before: 46, After: 39},
	{Name: "cli", Before: 22, After: 20},
	{Name: "docs", Before: 15, After: 18},
}

type app struct {
	chart slopegraph.State
	mode  int
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(8)).Gap(t.Space(5)).Children(func() {
		ui.Row(c).Gap(t.Space(4)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "Build seconds").FontSize(t.FontSize + 4).Bold()
			ui.Segmented(c, &a.mode, "Slope", "Dumbbell")
		})
		if a.mode == 0 {
			slopegraph.Slope(c, &a.chart, "No cache", "Cache", builds)
		} else {
			slopegraph.Dumbbell(c, &a.chart, "No cache", "Cache", builds)
		}
	})
}

func main() {
	a := &app{}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Slope Graph",
			Width:   720,
			Height:  560,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
