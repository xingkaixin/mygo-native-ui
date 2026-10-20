// Statechart runs an order's lifecycle from the text under it, as it is
// typed: from the cart through checkout and payment, where a declined card
// can be retried, to shipping, delivery and a return, or a cancellation on
// the way. A payment gateway answers each attempt after a moment, the
// first time with a decline, by sending its event; the buttons under the
// chart send the others. Reset starts the order over.
//
//	go run ./examples/statechart
package main

import (
	"log"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/statechart"
)

const lifecycle = `stateDiagram-v2
%% An order, from the cart to the customer
[*] --> Cart
Cart --> Cart : add item
Cart --> Checkout : check out
Checkout --> Cart : edit cart
Checkout --> Payment : place order
Checkout --> Cancelled : cancel

state Payment {
  [*] --> Authorizing
  Authorizing --> Declined : decline
  Declined --> Authorizing : retry
  Authorizing --> Paid : approve
}
Payment --> Cancelled : cancel

Paid --> Shipped : ship
Shipped --> Delivered : deliver
Delivered --> Returned : return
Delivered --> [*] : close
Returned --> [*] : refund`

// answer is how long the payment gateway takes to answer.
const answer = 1500 * time.Millisecond

type app struct {
	source string
	split  float32
	order  statechart.State
	// attempts is how many times the gateway answered, and due when it
	// answers next, zero while it has nothing to answer.
	attempts int
	due      time.Time
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	if !a.due.IsZero() {
		if wait := a.due.Sub(c.Now()); wait > 0 {
			c.After(wait)
		} else {
			a.due = time.Time{}
			if a.attempts++; a.attempts == 1 {
				a.order.Send("decline")
			} else {
				a.order.Send("approve")
			}
		}
	}
	ui.Column(c).Fill().Background(t.Background).Children(func() {
		ui.Row(c).Gap(t.Space(3)).Padding(t.Space(3), t.Space(4)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "Order #10482").Bold().Grow(1)
			if ui.Button(c, "Reset").Clicked() {
				a.order, a.attempts, a.due = statechart.State{}, 0, time.Time{}
			}
		})
		ui.Divider(c)
		ui.SplitVertical(c, &a.split, func() {
			if st, ok := statechart.View(c, &a.order, a.source).Moved(); ok {
				a.due = time.Time{}
				if st.To == "Authorizing" {
					a.due = c.Now().Add(answer)
					c.After(answer)
				}
			}
		}, func() {
			ui.Column(c).Fill().Padding(t.Space(3)).Children(func() {
				ui.TextArea(c, &a.source).Font("monospace").Label("Source").Grow(1)
			})
		}).Grow(1)
	})
}

func main() {
	a := &app{source: lifecycle, split: 480}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Order lifecycle",
			Width:   1440,
			Height:  940,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
