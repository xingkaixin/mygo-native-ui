// Voronoi draws the cells of twenty sites scattered at random. Drag a
// site, press anywhere to add one, secondary-click one to remove it,
// and turn on Delaunay to see which sites are neighbors.
//
//	go run ./examples/voronoi
package main

import (
	"fmt"
	"log"
	"math/rand/v2"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/voronoi"
)

type app struct {
	diagram voronoi.State
	sites   []voronoi.Point
}

func scatter(n int) []voronoi.Point {
	out := make([]voronoi.Point, n)
	for i := range out {
		out[i] = voronoi.Point{X: 0.03 + 0.94*rand.Float64(), Y: 0.03 + 0.94*rand.Float64()}
	}
	return out
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(8)).Gap(t.Space(4)).Children(func() {
		ui.Row(c).Gap(t.Space(4)).AlignItems(ui.Center).Children(func() {
			if ui.Button(c, "Scatter").Clicked() {
				a.sites = scatter(20)
			}
			ui.Checkbox(c, &a.diagram.Delaunay, "Delaunay")
			ui.Text(c, fmt.Sprintf("%d sites", len(a.sites))).TextColor(t.TextMuted)
		})
		voronoi.View(c, &a.diagram, &a.sites).Grow(1)
	})
}

func main() {
	a := &app{sites: scatter(20)}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Voronoi",
			Width:   820,
			Height:  620,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
