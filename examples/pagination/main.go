// Pagination pages through 412 orders in a table, with the rows a page
// chosen in its footer, and through 20 pages of search results by
// number.
//
//	go run ./examples/pagination
package main

import (
	"fmt"
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/pagination"
)

type order struct {
	id       int
	customer string
	total    float64
	status   string
}

var customers = []string{"Ada Lovelace", "Alan Turing", "Grace Hopper", "Edsger Dijkstra", "Barbara Liskov", "Donald Knuth"}

var statuses = []string{"Paid", "Shipped", "Refunded", "Pending"}

type app struct {
	tab     int
	orders  []order
	page    int
	size    int
	results int
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Children(func() {
		ui.Tabs(c, &a.tab, "Orders", "Search").Padding(0, 12)
		ui.Divider(c)
		ui.Column(c).Grow(1).Padding(16).Gap(12).Children(func() {
			if a.tab == 0 {
				a.table(c)
				return
			}
			for i := range 6 {
				n := a.results*6 + i + 1
				ui.Column(c.Key(n)).Gap(2).Children(func() {
					ui.Text(c, fmt.Sprintf("Result %d: Brewing tea at the right temperature", n)).TextColor(t.Accent)
					ui.Text(c, "Green tea wants water well below boiling; black tea wants it at a full boil.").
						FontSize(12).TextColor(t.TextMuted)
				})
			}
			ui.Spacer(c)
			pagination.View(c, &a.results, 20).AlignSelf(ui.Center)
		})
	})
}

func (a *app) table(c *ui.Context) {
	t := c.Theme()
	first := min(a.page*a.size, len(a.orders))
	rows := a.orders[first:min(first+a.size, len(a.orders))]
	columns := []ui.TableColumn{{Title: "Order", Width: 90}, {Title: "Customer"}, {Title: "Total", Width: 100, Align: ui.End}, {Title: "Status", Width: 100}}
	ui.Column(c).Grow(1).Radius(t.Radius).Border(1, t.Border).Clip().Children(func() {
		ui.Table(c.Key(a.page), nil, columns, len(rows), func(row, col int) {
			o := rows[row]
			switch col {
			case 0:
				ui.Text(c, fmt.Sprintf("#%05d", o.id)).FontFeatures("tnum")
			case 1:
				ui.Text(c, o.customer)
			case 2:
				ui.Text(c, fmt.Sprintf("$%.2f", o.total)).FontFeatures("tnum")
			case 3:
				ui.Text(c, o.status)
			}
		}).Grow(1)
		ui.Divider(c)
		pagination.Footer(c, &a.page, &a.size, len(a.orders), 10, 25, 50).Padding(6, 12)
	})
}

func main() {
	a := &app{size: 10}
	for i := range 412 {
		a.orders = append(a.orders, order{
			id:       10000 + i,
			customer: customers[i*7%len(customers)],
			total:    float64(i*37%500) + 0.99,
			status:   statuses[i*3%len(statuses)],
		})
	}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Pages",
			Width:   720,
			Height:  560,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
