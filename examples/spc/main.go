// SPC charts made-up diameters of shafts turned on a lathe, one measured
// every 20 minutes, against a target of 25 mm. The first 20 set the
// limits; then the tool wears and the diameters drift up, one shaft is
// measured badly, and two come out near the upper limit. Hover a point
// for its value and the rules it breaks.
//
//	go run ./examples/spc
package main

import (
	"fmt"
	"log"
	"math/rand/v2"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/spc"
)

func shafts() spc.Chart {
	rnd := rand.New(rand.NewPCG(2, 9))
	ch := spc.Chart{Baseline: 20, Unit: "mm"}
	for i := range 48 {
		v := 25 + rnd.NormFloat64()*0.004
		switch {
		case i >= 26 && i < 36:
			v += 0.005
		case i == 38:
			v -= 0.019
		case i == 42 || i == 44:
			v += 0.011
		}
		ch.Values = append(ch.Values, v)
		ch.Labels = append(ch.Labels, fmt.Sprintf("%02d:%02d", 6+i/3, i%3*20))
	}
	return ch
}

var data = shafts()

func view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(6)).Gap(t.Space(4)).Children(func() {
		ui.Column(c).Gap(t.Space(0.5)).Children(func() {
			ui.Text(c, "Shaft diameter, lathe 3").Bold().FontSize(t.FontSize + 4)
			ui.Text(c, "Individuals chart, limits from the first 20 shafts").TextColor(t.TextMuted)
		})
		spc.View(c, data).Grow(1)
	})
}

func main() {
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Control chart",
			Width:   1000,
			Height:  560,
			Content: ui.View(view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
