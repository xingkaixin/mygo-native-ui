// Datadiff compares a shop's stock before and after a night's sync,
// matched by SKU: prices and counts change, a product goes, two come, and
// the warehouse column gives way to a location.
//
//	go run ./examples/datadiff
package main

import (
	"encoding/csv"
	"log"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/datadiff"
)

const before = `sku,name,price,stock,warehouse
A-100,Desk lamp,39.00,120,North
A-101,Floor lamp,89.00,34,North
A-102,Clip light,19.50,210,North
B-200,Oak shelf,149.00,12,South
B-201,Pine shelf,99.00,40,South
B-202,Wall hook,4.20,900,South
B-203,Coat rack,59.00,18,South
C-300,Cushion,24.00,75,East
C-301,Throw,49.00,33,East
C-302,Rug,199.00,6,East
C-303,Doormat,29.00,58,East
C-304,Curtain,64.00,22,East
D-400,Mug,9.00,300,West
D-401,Teapot,34.00,41,West
D-402,Kettle,79.00,27,West`

const after = `sku,name,price,stock,location
A-100,Desk lamp,42.00,96,N-3
A-101,Floor lamp,89.00,34,N-3
A-102,Clip light,19.50,180,N-4
A-103,Reading light,54.00,60,N-4
B-200,Oak shelf,149.00,12,S-1
B-201,Pine shelf,99.00,40,S-1
B-202,Wall hook,4.20,900,S-2
B-203,Coat rack,59.00,18,S-2
C-300,Cushion,24.00,75,E-1
C-301,Throw blanket,49.00,33,E-1
C-303,Doormat,29.00,58,E-2
C-304,Curtain,64.00,22,E-2
D-400,Mug,9.00,300,W-1
D-401,Teapot,34.00,41,W-1
D-402,Kettle,74.00,27,W-1
D-403,Milk jug,16.00,80,W-2`

func parse(s string) datadiff.Table {
	rows, err := csv.NewReader(strings.NewReader(s)).ReadAll()
	if err != nil {
		log.Fatal(err)
	}
	return datadiff.Table{Columns: rows[0], Rows: rows[1:]}
}

type app struct {
	diff  datadiff.Diff
	state datadiff.State
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Padding(t.Space(6)).Background(t.Background).Children(func() {
		datadiff.View(c, &a.state, a.diff).Grow(1).AutoFocus()
	})
}

func main() {
	a := &app{diff: datadiff.Compare(parse(before), parse(after), "sku")}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Data Diff",
			Width:   900,
			Height:  560,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
