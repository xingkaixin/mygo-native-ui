// Boxmodel edits the box of a card and shows the card as it changes:
// drag a size sideways, hold Option to change the opposite side with it,
// or double-click it to type.
//
//	go run ./examples/boxmodel
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/boxmodel"
)

type app struct {
	box boxmodel.Box
}

func f(v float64) float32 { return float32(v) }

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	b := &a.box
	ui.Row(c).Fill().Background(t.Background).Padding(t.Space(6)).Gap(t.Space(8)).AlignItems(ui.Start).Children(func() {
		ui.Column(c).Width(380).Gap(t.Space(3)).Children(func() {
			ui.Text(c, "Card").Bold()
			boxmodel.View(c, b)
		})
		ui.Column(c).Grow(1).Background(t.Surface).Radius(t.Radius).Padding(t.Space(4)).AlignItems(ui.Start).Children(func() {
			ui.Column(c).Background(t.Accent.Alpha(0.12)).Children(func() {
				ui.Column(c).Margin(f(b.Margin.Top), f(b.Margin.Right), f(b.Margin.Bottom), f(b.Margin.Left)).
					BorderWidth(f(b.Border.Top), f(b.Border.Right), f(b.Border.Bottom), f(b.Border.Left)).BorderColor(t.Text).
					Padding(f(b.Padding.Top), f(b.Padding.Right), f(b.Padding.Bottom), f(b.Padding.Left)).
					Background(t.Background).Radius(4).Children(func() {
					ui.Column(c).Size(f(b.Width), f(b.Height)).Gap(t.Space(1)).Children(func() {
						ui.Text(c, "Ada Lovelace").Bold()
						ui.Text(c, "Wrote the first program, for a machine not yet built.").TextColor(t.TextMuted).FontSize(t.FontSize - 1)
					})
				})
			})
		})
	})
}

func main() {
	a := &app{box: boxmodel.Box{
		Width: 220, Height: 64,
		Padding: boxmodel.Edges{Top: 12, Right: 16, Bottom: 12, Left: 16},
		Border:  boxmodel.Edges{Top: 1, Right: 1, Bottom: 1, Left: 4},
		Margin:  boxmodel.Edges{Top: 8, Right: 8, Bottom: 24, Left: 8},
	}}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Box model",
			Width:   880,
			Height:  420,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
