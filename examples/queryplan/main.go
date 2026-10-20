// Queryplan shows how PostgreSQL ran a query of an online shop, read from
// the output of EXPLAIN (ANALYZE, BUFFERS, FORMAT JSON): its top customers
// in Germany by revenue, joining three tables, grouped and sorted, which an
// estimate of recent orders far too low sends into a nested loop; or the
// plan in the file given. Click a node to see all its properties, and
// fold a node by its arrow.
//
//	go run ./examples/queryplan [plan.json]
package main

import (
	_ "embed"
	"log"
	"os"
	"path/filepath"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/queryplan"
)

//go:embed plan.json
var embedded []byte

const query = `SELECT c.name, count(*) AS items, sum(oi.quantity * oi.unit_price) AS revenue
FROM customers c JOIN orders o ON o.customer_id = c.id JOIN order_items oi ON oi.order_id = o.id
WHERE c.country = 'DE' AND o.placed_at >= now() - interval '90 days'
GROUP BY c.id ORDER BY revenue DESC LIMIT 20`

type app struct {
	name, query string
	plan        *queryplan.Plan
	explain     queryplan.State
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Children(func() {
		ui.Column(c).Gap(t.Space(1.5)).Padding(t.Space(3)).Children(func() {
			ui.Text(c, a.name).Bold().SingleLine()
			if a.query != "" {
				ui.Text(c, a.query).Font("monospace").FontSize(t.FontSize - 1).TextColor(t.TextMuted).Selectable()
			}
		})
		ui.Divider(c)
		queryplan.View(c, &a.explain, a.plan).Grow(1)
	})
}

func main() {
	a := &app{name: "Top customers in Germany by revenue, last 90 days", query: query}
	data := embedded
	if len(os.Args) > 1 {
		var err error
		if data, err = os.ReadFile(os.Args[1]); err != nil {
			log.Fatal(err)
		}
		a.name, a.query = filepath.Base(os.Args[1]), ""
	}
	plan, err := queryplan.Parse(data)
	if err != nil {
		log.Fatal(err)
	}
	a.plan = plan
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Query Plan",
			Width:   1440,
			Height:  680,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
