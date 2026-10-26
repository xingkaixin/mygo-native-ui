// Barcode makes a shipping label's Code 128 and a product's EAN-13 from
// what is typed: twelve digits get their check digit, and text a code
// cannot hold says so.
//
//	go run ./examples/barcode
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/barcode"
)

type app struct {
	shipment string
	product  string
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(6)).Gap(t.Space(8)).Children(func() {
		part(c, "Code 128", &a.shipment, func(s string) (*barcode.Code, error) { return barcode.Code128(s) })
		part(c, "EAN-13", &a.product, barcode.EAN13)
	})
}

func part(c *ui.Context, name string, text *string, encode func(string) (*barcode.Code, error)) {
	t := c.Theme()
	code, err := encode(*text)
	ui.Column(c).Gap(t.Space(3)).Children(func() {
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, name).Bold().Width(t.Space(22))
			ui.TextInput(c, text).Font("monospace").Label(name).Grow(1)
		})
		if err != nil {
			ui.Text(c, err.Error()).TextColor(t.Danger).Height(t.Space(24))
			return
		}
		barcode.View(c, code).Size(float32(code.Modules()+20)*2.5, t.Space(24))
	})
}

func main() {
	a := &app{shipment: "1Z-8842-0917-3", product: "590123412345"}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Barcodes",
			Width:   560,
			Height:  360,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
