// Transfer chooses the columns of an orders table, and their order, from
// its sixty fields: the order ID always shows, first, and the payment
// token never does. The header above shows the columns as the table would.
//
//	go run ./examples/transfer
package main

import (
	"fmt"
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/transfer"
)

// app is the state the window shows: the fields, the columns chosen and
// the picker.
type app struct {
	fields  []transfer.Item
	titles  map[string]string
	columns []string
	picker  transfer.State
	changes int
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Children(func() {
		ui.Column(c).Gap(6).Padding(12).Children(func() {
			ui.Text(c, fmt.Sprintf("Orders · %d columns", len(a.columns))).Bold()
			a.header(c)
		})
		ui.Divider(c)
		if transfer.View(c, &a.picker, &a.columns, a.fields).Grow(1).Margin(12).Changed() {
			a.changes++
		}
		ui.Divider(c)
		noun := "changes"
		if a.changes == 1 {
			noun = "change"
		}
		ui.Text(c, fmt.Sprintf("%d fields · %d %s", len(a.fields), a.changes, noun)).FontSize(12).TextColor(t.TextMuted).Padding(6, 12)
	})
}

// header shows the columns chosen as the table's header, in their order.
func (a *app) header(c *ui.Context) {
	t := c.Theme()
	ui.ScrollHorizontal(c).Height(32).Radius(6).Border(1, t.Border).Background(t.Surface).Children(func() {
		ui.Row(c).Fill().AlignItems(ui.Stretch).Children(func() {
			for i, k := range a.columns {
				cell := ui.Row(c.Key(k)).Padding(0, 10).AlignItems(ui.Center).Shrink(0)
				if i > 0 {
					cell.Draw(func(p *ui.Painter, r ui.Rect) {
						p.Fill(ui.Rect{X: r.X, Y: r.Y + 6, W: 1, H: r.H - 12}, t.Border, 0)
					})
				}
				cell.Children(func() {
					ui.Text(c, a.titles[k]).FontSize(12).FontWeight(600).SingleLine()
				})
			}
		})
	})
}

// newApp returns the app with the orders table's fields, six of them
// chosen.
func newApp() *app {
	a := &app{titles: map[string]string{}}
	field := func(key, title, kind string) {
		a.fields = append(a.fields, transfer.Item{Key: key, Title: title, Detail: kind})
		a.titles[key] = title
	}
	field("id", "Order ID", "uuid")
	field("number", "Order number", "text")
	field("status", "Status", "enum")
	field("created_at", "Created", "timestamp")
	field("updated_at", "Updated", "timestamp")
	field("placed_at", "Placed", "timestamp")
	field("paid_at", "Paid", "timestamp")
	field("shipped_at", "Shipped", "timestamp")
	field("delivered_at", "Delivered", "timestamp")
	field("cancelled_at", "Cancelled", "timestamp")
	field("customer_id", "Customer ID", "uuid")
	field("customer_name", "Customer", "text")
	field("customer_email", "Email", "text")
	field("customer_phone", "Phone", "text")
	field("customer_since", "Customer since", "date")
	field("customer_orders", "Orders by customer", "int")
	field("segment", "Segment", "enum")
	field("company", "Company", "text")
	field("vat_id", "VAT ID", "text")
	field("ship_name", "Ship to", "text")
	field("ship_line1", "Ship address", "text")
	field("ship_line2", "Ship address 2", "text")
	field("ship_city", "Ship city", "text")
	field("ship_region", "Ship region", "text")
	field("ship_postcode", "Ship postcode", "text")
	field("ship_country", "Ship country", "country")
	field("bill_name", "Bill to", "text")
	field("bill_line1", "Bill address", "text")
	field("bill_city", "Bill city", "text")
	field("bill_postcode", "Bill postcode", "text")
	field("bill_country", "Bill country", "country")
	field("currency", "Currency", "currency")
	field("subtotal", "Subtotal", "money")
	field("discount", "Discount", "money")
	field("shipping", "Shipping", "money")
	field("tax", "Tax", "money")
	field("total", "Total", "money")
	field("refunded", "Refunded", "money")
	field("net", "Net", "money")
	field("items", "Items", "int")
	field("quantity", "Quantity", "int")
	field("weight", "Weight", "decimal")
	field("coupon", "Coupon", "text")
	field("payment_method", "Payment method", "enum")
	field("payment_token", "Payment token", "secret")
	field("card_last4", "Card", "text")
	field("risk_score", "Risk score", "decimal")
	field("carrier", "Carrier", "enum")
	field("service", "Shipping service", "enum")
	field("tracking", "Tracking number", "text")
	field("warehouse", "Warehouse", "enum")
	field("channel", "Channel", "enum")
	field("source", "Source", "text")
	field("campaign", "Campaign", "text")
	field("referrer", "Referrer", "url")
	field("device", "Device", "enum")
	field("locale", "Locale", "text")
	field("gift", "Gift", "bool")
	field("note", "Customer note", "text")
	field("tags", "Tags", "list")
	for i := range a.fields {
		switch a.fields[i].Key {
		case "id":
			a.fields[i].Disabled = true
		case "payment_token":
			a.fields[i].Disabled = true
			a.fields[i].Detail = "restricted"
		}
	}
	a.columns = []string{"id", "number", "customer_name", "status", "total", "created_at"}
	return a
}

func main() {
	a := newApp()
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Columns",
			Width:   880,
			Height:  640,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
