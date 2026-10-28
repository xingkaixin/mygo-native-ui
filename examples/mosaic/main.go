// Mosaic maps a made-up market for cloud storage by industry and vendor,
// in millions of dollars a year: retail is the largest industry and
// split among many, banking leans on one vendor, and the public sector
// buys little but from few. Hover a block for its numbers.
//
//	go run ./examples/mosaic
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/mosaic"
)

var (
	industries = []string{"Retail", "Banking", "Healthcare", "Media", "Manufacturing", "Public sector"}
	vendors    = []string{"Nimbus", "Stratus", "Cirrus", "Others"}
	spend      = [][]float64{
		{420, 380, 260, 310},
		{610, 140, 90, 60},
		{220, 260, 180, 120},
		{150, 90, 210, 160},
		{130, 170, 60, 140},
		{40, 110, 10, 20},
	}
)

func view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(6)).Gap(t.Space(4)).Children(func() {
		ui.Column(c).Gap(t.Space(0.5)).Children(func() {
			ui.Text(c, "Cloud storage spend by industry").Bold().FontSize(t.FontSize + 4)
			ui.Text(c, "Columns as wide as each industry's spend, split by vendor").TextColor(t.TextMuted)
		})
		mosaic.View(c, industries, vendors, spend).Grow(1)
	})
}

func main() {
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Mosaic",
			Width:   960,
			Height:  620,
			Content: ui.View(view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
