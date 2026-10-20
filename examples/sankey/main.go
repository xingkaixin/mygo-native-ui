// Sankey shows where a month's visitors came from, the pages they landed
// on, and how many signed up, as product analytics do. Point at a flow or
// a stage to read it.
//
//	go run ./examples/sankey
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/sankey"
)

var stages = []string{
	"Search", "Ads", "Referral", "Social", "Direct",
	"Home", "Pricing", "Blog", "Docs",
	"Signed up", "Trial", "Left",
}

var flows = []sankey.Flow{
	f(0, 5, 4200), f(0, 7, 3100), f(0, 8, 2600),
	f(1, 6, 2900), f(1, 5, 1400),
	f(2, 5, 900), f(2, 8, 1200),
	f(3, 7, 2300), f(3, 5, 700),
	f(4, 5, 2100), f(4, 6, 800), f(4, 8, 600),
	f(5, 6, 3100), f(5, 9, 1200), f(5, 11, 4900),
	f(6, 9, 2700), f(6, 10, 1900), f(6, 11, 2200),
	f(7, 9, 400), f(7, 11, 5000),
	f(8, 9, 1300), f(8, 10, 600), f(8, 11, 2500),
}

func f(from, to int, n float64) sankey.Flow { return sankey.Flow{From: from, To: to, Value: n} }

func view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(6)).Gap(t.Space(4)).Children(func() {
		ui.Text(c, "Visitors in October").Bold().FontSize(t.FontSize + 6)
		sankey.View(c, stages, flows, "visitors").Grow(1).AlignSelf(ui.Stretch)
	})
}

func main() {
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Sankey",
			Width:   1080,
			Height:  680,
			Content: ui.View(view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
