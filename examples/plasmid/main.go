// Plasmid maps a made-up expression vector carrying GFP: point at a
// feature, click it to select it, or drag around the ring.
//
//	go run ./examples/plasmid
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/plasmid"
)

var vector = plasmid.Plasmid{
	Name:   "pDemo-GFP",
	Length: 4700,
	Features: []plasmid.Feature{
		{Name: "lac promoter", Start: 120, End: 151, Strand: 1, Color: ui.Hex("#9aa5b1")},
		{Name: "GFP", Start: 300, End: 1017, Strand: 1, Color: ui.Hex("#3cb371")},
		{Name: "T7 terminator", Start: 1060, End: 1108, Color: ui.Hex("#9aa5b1")},
		{Name: "araC", Start: 1200, End: 2078, Strand: -1, Color: ui.Hex("#4a90d9")},
		{Name: "ori", Start: 2350, End: 2939, Color: ui.Hex("#f2c14e")},
		{Name: "AmpR", Start: 3200, End: 4061, Strand: -1, Color: ui.Hex("#e8743b")},
		{Name: "AmpR promoter", Start: 4061, End: 4166, Strand: -1, Color: ui.Hex("#9aa5b1")},
	},
	Sites: []plasmid.Site{
		{Name: "EcoRI", At: 280},
		{Name: "BamHI", At: 1030},
		{Name: "HindIII", At: 1040},
		{Name: "NotI", At: 1052},
		{Name: "XhoI", At: 1110},
		{Name: "NdeI", At: 2200},
		{Name: "PstI", At: 3600},
		{Name: "ScaI", At: 3800},
	},
}

type app struct {
	view      plasmid.State
	selection plasmid.Range
}

func (a *app) render(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Padding(t.Space(6)).Background(t.Background).Children(func() {
		plasmid.View(c, &a.view, vector, &a.selection).Grow(1)
	})
}

func main() {
	a := &app{}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Plasmid",
			Width:   820,
			Height:  720,
			Content: ui.View(a.render),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
