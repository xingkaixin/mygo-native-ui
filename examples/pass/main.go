// Pass shows a boarding pass and a concert ticket side by side; the i
// button turns either over to its details.
//
//	go run ./examples/pass
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/barcode"
	"github.com/xingkaixin/mygo-native-ui/ui/pass"
)

type app struct {
	flight, concert pass.State
	boarding, show  pass.Pass
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Row(c).Fill().Gap(t.Space(10)).AlignItems(ui.Center).Justify(ui.Center).Background(t.Background).Children(func() {
		pass.View(c, &a.flight, a.boarding)
		pass.View(c, &a.concert, a.show)
	})
}

func f(label, value string) pass.Field { return pass.Field{Label: label, Value: value} }

func main() {
	flight, err := barcode.Code128("M1LEE/MARCUS ETK7Q2P SFOJFK 0412")
	if err != nil {
		log.Fatal(err)
	}
	ticket, err := barcode.Code128("TIX-88213-B14")
	if err != nil {
		log.Fatal(err)
	}
	a := &app{
		boarding: pass.Pass{
			Issuer: "Northwind Air",
			Color:  ui.Hex("#1D4E89"),
			Header: []pass.Field{f("Gate", "B22")},
			Main:   []pass.Field{f("San Francisco", "SFO"), f("New York", "JFK")},
			Travel: true,
			Rows: [][]pass.Field{
				{f("Passenger", "Marcus Lee"), f("Seat", "14C")},
				{f("Flight", "NW 412"), f("Boards", "7:35"), f("Group", "3"), f("Departs", "8:05")},
			},
			Back: []pass.Field{
				f("Confirmation", "ETK7Q2P"),
				f("Date", "Thursday, November 5"),
				f("Bags", "1 checked, 1 carry-on"),
				f("Boarding", "Doors close 15 minutes before departure. Have this pass and your ID ready at the gate."),
			},
			Barcode: flight,
		},
		show: pass.Pass{
			Issuer: "The Fillmore",
			Color:  ui.Hex("#F2C14E"),
			Header: []pass.Field{f("Doors", "7 PM")},
			Main:   []pass.Field{f("Tonight", "Lowlight")},
			Rows: [][]pass.Field{
				{f("Section", "Balcony"), f("Row", "B"), f("Seat", "14")},
				{f("Date", "Sat, Nov 7"), f("Holder", "Dana Whitfield")},
			},
			Back: []pass.Field{
				f("Order", "88213"),
				f("Venue", "1805 Geary Blvd, San Francisco"),
				f("Entry", "All ages. Bags larger than 12 by 12 inches are not allowed."),
			},
			Barcode: ticket,
		},
	}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Passes",
			Width:   880,
			Height:  640,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
