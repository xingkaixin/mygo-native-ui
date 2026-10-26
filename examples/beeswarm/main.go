// Beeswarm shows the page load times of a made-up site's four regions,
// one dot for each of 150 visits a region: most fast, a long tail, and a
// few that time out. Hover a dot for its visit.
//
//	go run ./examples/beeswarm
package main

import (
	"fmt"
	"log"
	"math"
	"math/rand/v2"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/beeswarm"
)

// region returns n visits' load times, in seconds, around median with a
// long tail, and a share of timeouts near 8 seconds.
func region(r *rand.Rand, name string, n int, median, spread, timeouts float64) beeswarm.Group {
	g := beeswarm.Group{Name: name}
	for i := range n {
		v := median * math.Exp(r.NormFloat64()*spread)
		if r.Float64() < timeouts {
			v = 7.5 + r.Float64()
		}
		g.Values = append(g.Values, math.Round(v*100)/100)
		g.Labels = append(g.Labels, fmt.Sprintf("Visit #%d", 1000+i*7))
	}
	return g
}

type app struct {
	regions []beeswarm.Group
	chart   beeswarm.State
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(6)).Gap(t.Space(4)).Children(func() {
		ui.Text(c, "Page load by region").Bold().FontSize(t.FontSize + 4)
		beeswarm.View(c, &a.chart, beeswarm.Axis{Title: "Load time", Unit: "s"}, a.regions).Grow(1)
	})
}

func main() {
	r := rand.New(rand.NewPCG(3, 5))
	a := &app{regions: []beeswarm.Group{
		region(r, "Europe", 150, 1.2, 0.35, 0.01),
		region(r, "N. America", 150, 1.0, 0.3, 0.005),
		region(r, "Asia", 150, 1.9, 0.45, 0.03),
		region(r, "S. America", 150, 2.4, 0.5, 0.05),
	}}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Beeswarm",
			Width:   900,
			Height:  560,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
