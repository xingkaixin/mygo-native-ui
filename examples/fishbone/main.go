// Fishbone lays out the causes of a warehouse's late deliveries in six
// categories, two of them marked as likely root causes.
//
//	go run ./examples/fishbone
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/fishbone"
)

func causes(texts ...string) []fishbone.Cause {
	out := make([]fishbone.Cause, len(texts))
	for i, t := range texts {
		out[i].Text = t
	}
	return out
}

type app struct {
	bones   []fishbone.Bone
	diagram fishbone.State
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(4)).Children(func() {
		fishbone.View(c, &a.diagram, "Orders ship late", a.bones).Grow(1)
	})
}

func main() {
	a := &app{bones: []fishbone.Bone{
		{Name: "People", Causes: causes("New pickers still training", "Too few staff on weekends", "Shift handovers miss notes")},
		{Name: "Process", Causes: causes("Orders batched once a day", "Manual address checks", "No rush lane for express")},
		{Name: "Equipment", Causes: causes("Label printer jams", "Scanners lose Wi-Fi", "One forklift for two docks")},
		{Name: "Materials", Causes: causes("Boxes out of stock", "Wrong sizes ordered")},
		{Name: "Environment", Causes: causes("Dock blocked by returns", "Peak season volume", "Carrier pickup at 3 pm")},
		{Name: "Management", Causes: causes("Targets count orders, not on-time", "Delays found only weekly")},
	}}
	a.bones[1].Causes[0].Likely = true
	a.bones[4].Causes[2].Likely = true
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Fishbone",
			Width:   1100,
			Height:  620,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
