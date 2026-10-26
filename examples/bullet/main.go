// Bullet shows a made-up company's quarter against its targets: revenue
// past its target, profit and new customers short of theirs, satisfaction
// in its fair range, and costs and churn, where lower is better.
//
//	go run ./examples/bullet
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/bullet"
)

var names = []string{"Poor", "Fair", "Good"}

var quarter = []bullet.Bullet{
	{Label: "Revenue", Unit: "US$ k", Value: 270, Target: 250, Ranges: []float64{150, 225, 300}, RangeNames: names},
	{Label: "Profit", Unit: "%", Value: 22.5, Target: 26, Ranges: []float64{20, 25, 30}, RangeNames: names},
	{Label: "Avg order", Unit: "US$", Value: 412, Target: 550, Ranges: []float64{350, 500, 600}, RangeNames: names},
	{Label: "New customers", Unit: "count", Value: 1680, Target: 2100, Ranges: []float64{1400, 2000, 2500}, RangeNames: names},
	{Label: "Satisfaction", Unit: "of 5", Value: 3.6, Target: 4.2, Ranges: []float64{2.5, 3.75, 5}, RangeNames: names},
	{Label: "Support cost", Unit: "US$ per ticket", Value: 7.4, Target: 6, Ranges: []float64{5, 8, 12}, RangeNames: []string{"Good", "Fair", "Poor"}, LowerIsBetter: true},
	{Label: "Churn", Unit: "% a month", Value: 1.8, Target: 2, Ranges: []float64{2, 3, 4}, RangeNames: []string{"Good", "Fair", "Poor"}, LowerIsBetter: true},
}

func view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(6)).Gap(t.Space(5)).Children(func() {
		ui.Column(c).Gap(t.Space(0.5)).Children(func() {
			ui.Text(c, "Q3 at a glance").Bold().FontSize(t.FontSize + 4)
			ui.Text(c, "Bars are this quarter, marks the targets, bands poor, fair and good").TextColor(t.TextMuted)
		})
		bullet.View(c, quarter)
	})
}

func main() {
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Bullet graphs",
			Width:   720,
			Height:  560,
			Content: ui.View(view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
