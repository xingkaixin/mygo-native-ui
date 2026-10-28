// Windrose shows a season of hourly winds at a coastal station: westerly
// gales in winter, a sea breeze from the south-east in summer. Point at a
// wedge for its shares, and click a speed in the legend to hide it.
//
//	go run ./examples/windrose
package main

import (
	"log"
	"math"
	"math/rand/v2"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/windrose"
)

// wind is a prevailing wind: where it blows from, how widely it varies,
// its usual speed, and how often it blows.
type wind struct {
	dir, spread, speed, weight float64
}

func season(seed uint64, winds []wind) []windrose.Obs {
	r := rand.New(rand.NewPCG(seed, 7))
	total := 0.0
	for _, w := range winds {
		total += w.weight
	}
	out := make([]windrose.Obs, 0, 24*90)
	for range 24 * 90 {
		pick := r.Float64() * total
		w := winds[len(winds)-1]
		for _, x := range winds {
			if pick < x.weight {
				w = x
				break
			}
			pick -= x.weight
		}
		out = append(out, windrose.Obs{
			Dir:   math.Mod(w.dir+r.NormFloat64()*w.spread+360, 360),
			Speed: math.Max(0, w.speed*r.ExpFloat64()),
		})
	}
	return out
}

var (
	winter = season(1, []wind{{250, 20, 7, 5}, {290, 25, 6, 2}, {40, 20, 4, 1}, {180, 60, 2, 1}})
	summer = season(2, []wind{{135, 15, 4, 4}, {225, 30, 3, 2}, {0, 40, 2, 1}})
)

type app struct {
	rose   windrose.State
	season int
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(8)).Gap(t.Space(5)).Children(func() {
		ui.Row(c).Gap(t.Space(4)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "Harbor station").FontSize(t.FontSize + 8).Bold().Grow(1)
			ui.Segmented(c, &a.season, "Winter", "Summer")
		})
		obs := winter
		if a.season == 1 {
			obs = summer
		}
		windrose.View(c, &a.rose, obs, []float64{0.5, 2, 4, 6, 8, 11}, "m/s")
	})
}

func main() {
	a := &app{}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Windrose",
			Width:   760,
			Height:  640,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
