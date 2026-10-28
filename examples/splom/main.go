// Splom plots four measures of 120 made-up cars, by how many cylinders
// they have: heavier cars have more power, use more fuel and pick up
// speed sooner.
//
//	go run ./examples/splom
package main

import (
	"fmt"
	"log"
	"math/rand/v2"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/splom"
)

var columns = []string{"Weight (kg)", "Power (hp)", "Economy (mpg)", "0–60 (s)"}

func cars() (rows [][]float64, groups []string) {
	rng := rand.New(rand.NewPCG(3, 9))
	for _, g := range []struct {
		cyl    int
		weight float64
		n      int
	}{{4, 1050, 50}, {6, 1450, 35}, {8, 1800, 35}} {
		for range g.n {
			w := g.weight + rng.NormFloat64()*110
			hp := 40 + w*0.075 + float64(g.cyl)*8 + rng.NormFloat64()*12
			mpg := 52 - w*0.016 + rng.NormFloat64()*2.5
			acc := 21 - hp*0.045 + rng.NormFloat64()*1.1
			rows = append(rows, []float64{w, hp, mpg, acc})
			groups = append(groups, fmt.Sprintf("%d cylinders", g.cyl))
		}
	}
	return rows, groups
}

type app struct {
	rows   [][]float64
	groups []string
	matrix splom.State
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(4)).Children(func() {
		splom.View(c, &a.matrix, columns, a.rows, a.groups).Grow(1)
	})
}

func main() {
	a := &app{}
	a.rows, a.groups = cars()
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Scatterplot matrix",
			Width:   720,
			Height:  780,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
