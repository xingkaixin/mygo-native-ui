// ECDF compares the response times of three services, 600 requests each:
// an origin with a long tail, a CDN, and an edge cache that answers most
// requests at once and sends the rest on. Point along the plot to read
// each one's share, and turn on QQ to see how far each is from normal.
//
//	go run ./examples/ecdf
package main

import (
	"log"
	"math"
	"math/rand/v2"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/ecdf"
)

type app struct {
	dist   ecdf.State
	series []ecdf.Series
}

func sample() []ecdf.Series {
	r := rand.New(rand.NewPCG(9, 10))
	origin, cdn, cache := make([]float64, 600), make([]float64, 600), make([]float64, 600)
	for i := range 600 {
		origin[i] = 120 * math.Exp(0.45*r.NormFloat64())
		cdn[i] = 60 + 12*r.NormFloat64()
		cache[i] = max(1, 14+4*r.NormFloat64())
		if i%6 == 0 {
			cache[i] = 130 + 30*r.NormFloat64()
		}
	}
	return []ecdf.Series{{Name: "Origin", Values: origin}, {Name: "CDN", Values: cdn}, {Name: "Edge cache", Values: cache}}
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(8)).Children(func() {
		ecdf.View(c, &a.dist, "Response time, ms", a.series)
	})
}

func main() {
	a := &app{series: sample()}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "ECDF",
			Width:   820,
			Height:  560,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
