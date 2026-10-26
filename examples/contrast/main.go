// Contrast checks pairs of a made-up brand's colors against WCAG 2: pick
// a pair, edit either color, or take the text color it suggests.
//
//	go run ./examples/contrast
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/contrast"
)

type pair struct {
	name     string
	text, bg ui.Color
}

type app struct {
	pairs  []pair
	chosen int
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	p := &a.pairs[a.chosen]
	ui.Row(c).Fill().Background(t.Background).AlignItems(ui.Stretch).Children(func() {
		ui.Column(c).Width(200).Padding(t.Space(3)).Gap(t.Space(1)).Background(t.Surface).Children(func() {
			for i, q := range a.pairs {
				row := ui.Row(c).Key(i).Gap(t.Space(2)).AlignItems(ui.Center).Padding(t.Space(1.5), t.Space(2)).Radius(t.Radius).Cursor(ui.CursorPointer)
				if i == a.chosen {
					row.Background(t.Selection)
				}
				row.Children(func() {
					ui.Box(c).Size(22, 22).Radius(5).Background(q.bg).Border(1, t.Border).Children(func() {
						ui.Text(c, "a").TextColor(q.text).Bold().Fill().TextAlign(ui.Center)
					})
					ui.Text(c, q.name)
				})
				if row.Clicked() {
					a.chosen = i
				}
			}
		})
		ui.Column(c).Grow(1).Padding(t.Space(6)).Children(func() {
			contrast.View(c, &p.text, &p.bg)
		})
	})
}

func main() {
	a := &app{pairs: []pair{
		{"Body on paper", ui.Hex("#2B2D33"), ui.Hex("#FAF8F3")},
		{"Caption on paper", ui.Hex("#9A968C"), ui.Hex("#FAF8F3")},
		{"Link", ui.Hex("#3D7FD9"), ui.Hex("#FFFFFF")},
		{"Button", ui.Hex("#FFFFFF"), ui.Hex("#F2994A")},
		{"Badge", ui.Hex("#1E7F4F"), ui.Hex("#DDF3E6")},
		{"Footer", ui.Hex("#8E95A3"), ui.Hex("#1B1E24")},
	}}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Contrast",
			Width:   820,
			Height:  520,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
