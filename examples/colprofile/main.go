// Colprofile looks through listings of flats to rent: each column's
// header sums it up, and a click on a bar or a value keeps the rows in
// it, Shift-click on another bar widening that.
//
//	go run ./examples/colprofile
package main

import (
	"fmt"
	"log"
	"math"
	"math/rand/v2"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/colprofile"
)

var titles = []string{"District", "Rooms", "Rent", "Area (m²)", "Built", "Floor", "Elevator", "Agent"}

var (
	districts = []string{"Mitte", "Kreuzberg", "Neukölln", "Prenzlauer Berg", "Charlottenburg", "Wedding", "Friedrichshain", "Spandau"}
	weights   = []float64{1.4, 1.2, 1.3, 1.1, 0.9, 0.8, 1.0, 0.4}
	agents    = []string{"Hausmann & Co", "Spree Living", "Berlin Homes", "Kiez Immobilien", "Altbau Partners"}
)

func listings(n int) [][]string {
	r := rand.New(rand.NewPCG(7, 11))
	pick := func() int {
		total := 0.0
		for _, w := range weights {
			total += w
		}
		x := r.Float64() * total
		for i, w := range weights {
			if x -= w; x < 0 {
				return i
			}
		}
		return len(weights) - 1
	}
	rows := make([][]string, n)
	for i := range rows {
		d := pick()
		rooms := 1 + int(math.Abs(r.NormFloat64()*1.1))
		area := 22 + float64(rooms)*21 + r.NormFloat64()*9
		perM := 14 + float64(len(weights)-d)*1.2 + r.NormFloat64()*3
		built := 1890 + r.IntN(135)
		if r.Float64() < 0.35 {
			built = 1900 + r.IntN(20)
		}
		lift := "No"
		if built > 1960 || r.Float64() < 0.3 {
			lift = "Yes"
		}
		agent := agents[r.IntN(len(agents))]
		if r.Float64() < 0.18 {
			agent = ""
		}
		floor := fmt.Sprint(r.IntN(6))
		if r.Float64() < 0.04 {
			floor = ""
		}
		rows[i] = []string{
			districts[d], fmt.Sprint(rooms), fmt.Sprintf("%.0f", area*perM), fmt.Sprintf("%.0f", area),
			fmt.Sprint(built), floor, lift, agent,
		}
	}
	return rows
}

type app struct {
	rows    [][]string
	profile colprofile.State
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	colprofile.View(c, &a.profile, titles, a.rows).Fill().Background(t.Background)
}

func main() {
	a := &app{rows: listings(1200)}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Colprofile",
			Width:   1180,
			Height:  720,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
