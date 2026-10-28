// Forest pools nine made-up trials of a drug: the odds of a stroke on it
// against a placebo, and the change in systolic blood pressure. Switch
// between the two measures, and point at a trial to highlight it.
//
//	go run ./examples/forest
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/forest"
)

var odds = []forest.Study{
	{Name: "Halvorsen 2011", Estimate: 0.72, Lo: 0.48, Hi: 1.08},
	{Name: "Okafor 2013", Estimate: 0.91, Lo: 0.70, Hi: 1.18},
	{Name: "Lindqvist 2014", Estimate: 0.58, Lo: 0.31, Hi: 1.09},
	{Name: "Tanaka 2015", Estimate: 0.80, Lo: 0.66, Hi: 0.97},
	{Name: "Moreau 2016", Estimate: 1.12, Lo: 0.74, Hi: 1.70},
	{Name: "Brennan 2018", Estimate: 0.69, Lo: 0.52, Hi: 0.92},
	{Name: "Szabo 2019", Estimate: 0.85, Lo: 0.45, Hi: 1.61},
	{Name: "Achterberg 2021", Estimate: 0.77, Lo: 0.63, Hi: 0.94},
	{Name: "Ramirez 2023", Estimate: 0.95, Lo: 0.72, Hi: 1.25},
}

var pressure = []forest.Study{
	{Name: "Halvorsen 2011", Estimate: -6.1, Lo: -9.8, Hi: -2.4},
	{Name: "Okafor 2013", Estimate: -3.2, Lo: -5.9, Hi: -0.5},
	{Name: "Lindqvist 2014", Estimate: -8.4, Lo: -14.6, Hi: -2.2},
	{Name: "Tanaka 2015", Estimate: -4.7, Lo: -6.1, Hi: -3.3},
	{Name: "Moreau 2016", Estimate: 0.8, Lo: -3.9, Hi: 5.5},
	{Name: "Brennan 2018", Estimate: -5.5, Lo: -7.9, Hi: -3.1},
	{Name: "Szabo 2019", Estimate: -2.0, Lo: -8.3, Hi: 4.3},
	{Name: "Achterberg 2021", Estimate: -4.1, Lo: -5.6, Hi: -2.6},
	{Name: "Ramirez 2023", Estimate: -1.9, Lo: -4.4, Hi: 0.6},
}

type app struct{ measure int }

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(8)).Gap(t.Space(5)).Children(func() {
		ui.Segmented(c, &a.measure, "Odds of stroke", "Systolic pressure, mmHg").Label("Measure")
		if a.measure == 0 {
			forest.View(c, odds, forest.Ratio)
		} else {
			forest.View(c, pressure, forest.Difference)
		}
	})
}

func main() {
	a := &app{}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Forest Plot",
			Width:   900,
			Height:  520,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
