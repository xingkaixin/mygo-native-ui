// Pivot sums up a year of sales, as a spreadsheet's pivot table does: 2,000
// orders by region and country across the quarters. Drag the fields
// between Rows, Columns and Values, fold the regions, and sort by a
// quarter's revenue.
//
//	go run ./examples/pivot
package main

import (
	"log"
	"math"
	"math/rand/v2"
	"strconv"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/pivot"
)

var fields = []string{"Region", "Country", "Product", "Category", "Quarter", "Units", "Revenue"}

// sales returns n orders, made up the same each run.
func sales(n int) [][]string {
	regions := []struct {
		name      string
		countries []string
	}{
		{"Americas", []string{"United States", "Canada", "Brazil", "Mexico"}},
		{"Europe", []string{"Germany", "France", "United Kingdom", "Spain", "Italy"}},
		{"Asia Pacific", []string{"Japan", "China", "Australia", "India", "South Korea"}},
		{"Middle East & Africa", []string{"United Arab Emirates", "South Africa", "Saudi Arabia"}},
	}
	products := []struct {
		name, category string
		price          float64
	}{
		{"Laptop Pro 14", "Computers", 1899},
		{"Laptop Air 13", "Computers", 1199},
		{"Desktop Studio", "Computers", 2499},
		{"Phone X", "Phones", 999},
		{"Phone Mini", "Phones", 699},
		{"Tablet 11", "Tablets", 799},
		{"Tablet Mini", "Tablets", 499},
		{"Wireless Earbuds", "Accessories", 179},
		{"Smart Watch", "Accessories", 399},
		{"USB-C Charger", "Accessories", 39},
	}
	r := rand.New(rand.NewPCG(2025, 7))
	records := make([][]string, n)
	for i := range records {
		reg := regions[min(r.IntN(10)/3, len(regions)-1)]
		p := products[r.IntN(len(products))]
		units := 1 + r.IntN(24)
		if p.category == "Accessories" {
			units *= 3
		}
		revenue := math.Round(float64(units) * p.price * (0.85 + 0.15*r.Float64()))
		records[i] = []string{
			reg.name,
			reg.countries[r.IntN(len(reg.countries))],
			p.name,
			p.category,
			"Q" + strconv.Itoa(1+r.IntN(4)),
			strconv.Itoa(units),
			strconv.FormatFloat(revenue, 'f', -1, 64),
		}
	}
	return records
}

type app struct {
	records [][]string
	table   pivot.State
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(4)).Gap(t.Space(3)).Children(func() {
		ui.Text(c, "Sales 2025").Bold().FontSize(t.FontSize + 4)
		pivot.View(c, &a.table, fields, a.records).Grow(1).AlignSelf(ui.Stretch)
	})
}

func main() {
	a := &app{records: sales(2000)}
	a.table.Rows = []string{"Region", "Country"}
	a.table.Columns = []string{"Quarter"}
	a.table.Values = []pivot.Value{{Field: "Revenue", Agg: pivot.Sum}}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Pivot",
			Width:   1200,
			Height:  760,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
