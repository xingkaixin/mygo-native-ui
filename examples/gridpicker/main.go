// Gridpicker inserts tables into a note: click Table in the toolbar,
// move over the grid to size the table, and click to insert it.
//
//	go run ./examples/gridpicker
package main

import (
	"fmt"
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/gridpicker"
)

type table struct{ rows, cols int }

type app struct {
	picking bool
	tables  []table
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Children(func() {
		ui.Toolbar(c, func() {
			button := ui.Button(c, "Table")
			if button.Clicked() {
				a.picking = !a.picking
			}
			ui.Popover(c, button, &a.picking, func() {
				if rows, cols, ok := gridpicker.View(c, 8, 10).Chosen(); ok {
					a.tables = append(a.tables, table{rows, cols})
					a.picking = false
				}
			})
			if ui.Button(c, "Clear").Disabled(len(a.tables) == 0).Clicked() {
				a.tables = nil
			}
		})
		ui.Divider(c)
		ui.Scroll(c).Grow(1).Children(func() {
			ui.Column(c).Padding(t.Space(6)).Gap(t.Space(5)).Children(func() {
				ui.Text(c, "Release checklist").FontSize(t.FontSize + 8).Bold()
				if len(a.tables) == 0 {
					ui.Text(c, "Insert a table from the toolbar.").TextColor(t.TextMuted)
				}
				for i, tb := range a.tables {
					grid(c.Key(i), tb)
				}
			})
		})
	})
}

func grid(c *ui.Context, tb table) {
	t := c.Theme()
	ui.Grid(c).Columns(tb.cols).Border(1, t.Border).Radius(t.Radius).Clip().Dividers(1, t.Border).Children(func() {
		for r := range tb.rows {
			for k := range tb.cols {
				cell := ui.Text(c, "").Padding(t.Space(2), t.Space(3)).MinHeight(t.Space(8))
				if r == 0 {
					cell = ui.Text(c, fmt.Sprintf("Column %d", k+1)).Bold().Padding(t.Space(2), t.Space(3)).Background(t.Surface)
				}
				cell.MinWidth(t.Space(20))
			}
		}
	})
}

func main() {
	a := &app{}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Gridpicker",
			Width:   760,
			Height:  560,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
