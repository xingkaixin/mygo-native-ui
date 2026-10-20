// Spreadsheet keeps a team's budget for a quarter, as spreadsheet apps
// do: the totals and shares are formulas, worked out again as the months
// change.
//
//	go run ./examples/spreadsheet
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/spreadsheet"
)

type app struct {
	cells map[spreadsheet.Cell]string
	sheet spreadsheet.State
}

// budget fills the sheet, a row a line of it, its cells from column A.
func budget() map[spreadsheet.Cell]string {
	rows := [][]string{
		{"Item", "Jan", "Feb", "Mar", "Total", "Share"},
		{"Salaries", "48000", "48000", "52000", "=SUM(B2:D2)", "=ROUND(E2/E9*100, 1)"},
		{"Cloud", "6200", "6900", "7400", "=SUM(B3:D3)", "=ROUND(E3/E9*100, 1)"},
		{"Software", "1800", "1800", "2100", "=SUM(B4:D4)", "=ROUND(E4/E9*100, 1)"},
		{"Office", "3500", "3500", "3500", "=SUM(B5:D5)", "=ROUND(E5/E9*100, 1)"},
		{"Travel", "900", "2400", "1200", "=SUM(B6:D6)", "=ROUND(E6/E9*100, 1)"},
		{"Marketing", "4000", "5500", "8000", "=SUM(B7:D7)", "=ROUND(E7/E9*100, 1)"},
		{},
		{"Total", "=SUM(B2:B7)", "=SUM(C2:C7)", "=SUM(D2:D7)", "=SUM(E2:E7)"},
		{"Budget", "70000", "70000", "75000", "=SUM(B10:D10)"},
		{"Left", "=B10-B9", "=C10-C9", "=D10-D9", "=E10-E9"},
		{"Over?", "=IF(B11<0, 1, 0)", "=IF(C11<0, 1, 0)", "=IF(D11<0, 1, 0)"},
	}
	cells := map[spreadsheet.Cell]string{}
	for r, row := range rows {
		for k, v := range row {
			if v != "" {
				cells[spreadsheet.Cell{Col: k, Row: r}] = v
			}
		}
	}
	return cells
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(4)).Gap(t.Space(3)).Children(func() {
		ui.Text(c, "Q1 budget").Bold().FontSize(t.FontSize + 4)
		spreadsheet.View(c, &a.sheet, &a.cells, 12, 60).Grow(1).AlignSelf(ui.Stretch)
	})
}

func main() {
	a := &app{cells: budget()}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Spreadsheet",
			Width:   1100,
			Height:  680,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
