// Periodic shows the periodic table: hover an element to name it, click
// it to read its group, period and block below, hover a kind in the
// legend to find its elements, and search by name, symbol or number.
//
//	go run ./examples/periodic
package main

import (
	"fmt"
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/periodic"
)

// block returns the block of the table e is in, as "d".
func block(e periodic.Element) string {
	switch g := e.Group(); {
	case g == 0:
		return "f"
	case g <= 2 || e.Number == 2:
		return "s"
	case g <= 12:
		return "d"
	}
	return "p"
}

type app struct {
	table periodic.State
	query string
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(6)).Gap(t.Space(4)).Children(func() {
		ui.SearchField(c, &a.query).Width(t.Space(60)).Placeholder("Name, symbol or number")
		periodic.View(c, &a.table, a.query).Grow(1)
		ui.Row(c).Gap(t.Space(6)).Height(t.Space(10)).AlignItems(ui.Center).Children(func() {
			if a.table.Chosen == 0 {
				ui.Text(c, "Click an element").TextColor(t.TextMuted)
				return
			}
			e := periodic.Elements[a.table.Chosen-1]
			ui.Text(c, e.Name).FontSize(t.FontSize + 3).Bold()
			group := "—"
			if g := e.Group(); g > 0 {
				group = fmt.Sprint(g)
			}
			for _, f := range [][2]string{
				{"Atomic number", fmt.Sprint(e.Number)},
				{"Atomic weight", e.Mass},
				{"Group", group},
				{"Period", fmt.Sprint(e.Period())},
				{"Block", block(e)},
			} {
				ui.Column(c.Key(f[0])).Children(func() {
					ui.Text(c, f[0]).FontSize(t.FontSize - 2).TextColor(t.TextMuted)
					ui.Text(c, f[1]).FontFeatures("tnum")
				})
			}
		})
	})
}

func main() {
	a := &app{table: periodic.State{Chosen: 26}}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Periodic Table",
			Width:   1100,
			Height:  720,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
