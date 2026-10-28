// Pyramid shows the people of a made-up town by age and sex in 2025,
// with outlines of 2005: the baby boom has moved up into its sixties,
// fewer children are born, and the town draws people in their late
// twenties. Switch between counts and shares, and hover a row.
//
//	go run ./examples/pyramid
package main

import (
	"fmt"
	"log"
	"math"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/pyramid"
)

var ages []string

// town returns the people in each five-year group: a base falling with
// age, a boom centered at boom, and a bump of newcomers at 27.
func town(scale, boom, births float64) []float64 {
	var out []float64
	for g := range 19 {
		age := float64(g*5) + 2.5
		v := 1000 * math.Exp(-math.Pow(age/78, 4))
		v *= 1 + 0.45*math.Exp(-math.Pow((age-boom)/9, 2))
		v *= 1 + 0.3*math.Exp(-math.Pow((age-27)/6, 2))
		if age < 20 {
			v *= births
		}
		out = append(out, math.Round(v*scale))
	}
	return out
}

func init() {
	for g := range 19 {
		if g == 18 {
			ages = append(ages, "90+")
			break
		}
		ages = append(ages, fmt.Sprintf("%d–%d", g*5, g*5+4))
	}
}

var (
	men    = pyramid.Side{Name: "Men", Values: town(1, 62, 0.82), Before: town(0.96, 42, 1)}
	women  = pyramid.Side{Name: "Women", Values: town(1.04, 62, 0.8), Before: town(1, 42, 0.98)}
	shares = 0
)

func view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(6)).Gap(t.Space(4)).Children(func() {
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			ui.Column(c).Gap(t.Space(0.5)).Grow(1).Children(func() {
				ui.Text(c, "People of Larkfield, 2025").Bold().FontSize(t.FontSize + 4)
				ui.Text(c, "Outlines show 2005").TextColor(t.TextMuted)
			})
			ui.Segmented(c, &shares, "Count", "Share")
		})
		pyramid.View(c, ages, men, women, shares == 1).Grow(1)
	})
}

func main() {
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Population pyramid",
			Width:   900,
			Height:  720,
			Content: ui.View(view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
