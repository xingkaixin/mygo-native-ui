// Classdiagram draws a shop's domain model from the text beside it, as it
// is typed: customers placing orders of line items and products, an order
// shipped and given a status, and payment methods that realize an
// interface, a card and a wallet inheriting from an abstract one.
// Clicking a class lights its relations and shows it under the diagram.
//
//	go run ./examples/classdiagram
package main

import (
	"fmt"
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/classdiagram"
)

const shop = `classDiagram
%% A shop's orders: who places them, what they hold, how they are paid
class Customer {
  +id: UUID
  +name: String
  -email: String
  +orders() List~Order~
}
class Order {
  +number: int
  +placedAt: Time
  -items: List~LineItem~
  +total() Money
  +cancel() bool
  +nextNumber() int$
}
class LineItem {
  +quantity: int
  +subtotal() Money
}
class Product
Product : +String sku
Product : +String title
Product : +Money price
class OrderStatus {
  <<enumeration>>
  PENDING
  PAID
  SHIPPED
  CANCELLED
}
class Shipment {
  +carrier: String
  +tracking: String
  +eta() Date
}
class Payable {
  <<interface>>
  +pay(amount Money) Receipt
  +refund(r Receipt) bool
}
<<abstract>> PaymentMethod
class PaymentMethod {
  +label: String
  +authorize(amount Money) bool*
}
class Card {
  -last4: String
  -expires: Date
  +authorize(amount Money) bool
}
class Wallet {
  -provider: String
  +authorize(amount Money) bool
}

Customer "1" --> "*" Order : places
Order "1" *-- "1..*" LineItem
LineItem "*" --> "1" Product
Order "1" -- "0..1" Shipment
Order ..> OrderStatus
Shipment .. OrderStatus : reports
Customer "1" o-- "*" PaymentMethod : saves
PaymentMethod ..|> Payable
PaymentMethod <|-- Card
Wallet --|> PaymentMethod`

type app struct {
	source  string
	split   float32
	diagram classdiagram.State
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Children(func() {
		ui.Split(c, &a.split, func() {
			ui.Column(c).Fill().Padding(t.Space(3)).Children(func() {
				ui.TextArea(c, &a.source).Font("monospace").Label("Source").Grow(1)
			})
		}, func() {
			ui.Column(c).Fill().Children(func() {
				classdiagram.View(c, &a.diagram, a.source).Grow(1)
				ui.Divider(c)
				ui.Row(c).Gap(t.Space(2)).Padding(t.Space(3), t.Space(4)).AlignItems(ui.Center).Children(func() {
					cl, ok := a.diagram.Chosen()
					if !ok {
						ui.Text(c, "Click a class to choose it").TextColor(t.TextMuted)
						return
					}
					ui.Text(c, cl.Name).Bold()
					if cl.Annotation != "" {
						ui.Text(c, "«"+cl.Annotation+"»").TextColor(t.TextMuted)
					}
					ui.Text(c, fmt.Sprintf("%d fields, %d methods", len(cl.Fields), len(cl.Methods))).TextColor(t.TextMuted)
				})
			})
		}).Grow(1)
	})
}

func main() {
	a := &app{source: shop, split: 420}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Class diagram",
			Width:   1440,
			Height:  940,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
