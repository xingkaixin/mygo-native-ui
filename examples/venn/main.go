// Venn shows which languages the developers of a small company write:
// hover a part of the diagram to name it, click it to list its people on
// the right, and leave Rust out to compare Go and Swift alone.
//
//	go run ./examples/venn
package main

import (
	"log"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/venn"
)

var people = strings.Fields(`Ada Ben Cleo Dev Eli Fay Gus Hana Ivo Jun Kai Lena Milo Nia Omar Pia
Quin Rosa Sami Teo Uma Vik Wren Xia Yuri Zoe Abel Bea Cyd Dora Ezra Fern Gil Hugo Iris Jade Kofi Luz
Mae Nell Otto Pax Remy Sol Tess Vera`)

// pick returns the people whose place in the list satisfies in.
func pick(in func(i int) bool) []string {
	var out []string
	for i, p := range people {
		if in(i) {
			out = append(out, p)
		}
	}
	return out
}

var sets = []venn.Set{
	{Name: "Go", Color: ui.Hex("#0b8fb5"), Items: pick(func(i int) bool { return i < 26 })},
	{Name: "Swift", Color: ui.Hex("#e8571e"), Items: pick(func(i int) bool { return i >= 18 && i < 38 })},
	{Name: "Rust", Color: ui.Hex("#8a5a3c"), Items: pick(func(i int) bool { return i%4 == 0 || i >= 40 })},
}

type app struct {
	diagram venn.State
	rust    bool
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	shown := sets
	if !a.rust {
		shown = sets[:2]
	}
	ui.Row(c).Fill().Background(t.Background).Padding(t.Space(6)).Gap(t.Space(6)).Children(func() {
		ui.Column(c).Grow(1).FillHeight().Gap(t.Space(3)).Children(func() {
			ui.Row(c).AlignItems(ui.Center).Children(func() {
				ui.Text(c, "Languages we write").FontSize(t.FontSize + 3).Bold().Grow(1)
				if ui.Checkbox(c, &a.rust, "Rust").Changed() {
					a.diagram.Chosen = 0
				}
			})
			venn.View(c, &a.diagram, shown).Grow(1)
		})
		ui.Column(c).Width(t.Space(56)).FillHeight().Gap(t.Space(2)).Children(func() {
			if a.diagram.Chosen == 0 {
				ui.Text(c, "Click a part of the diagram to see who is in it.").TextColor(t.TextMuted)
				return
			}
			ui.Text(c, venn.Describe(shown, a.diagram.Chosen)).Bold()
			ui.Scroll(c).Grow(1).Children(func() {
				ui.Column(c).Gap(t.Space(1)).Children(func() {
					for _, p := range venn.Items(shown, a.diagram.Chosen) {
						ui.Text(c.Key(p), p)
					}
				})
			})
		})
	})
}

func main() {
	a := &app{rust: true}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Venn",
			Width:   880,
			Height:  560,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
