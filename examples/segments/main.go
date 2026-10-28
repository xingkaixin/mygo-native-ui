// Segments shows the displays of a few devices: a bedside clock whose
// colon blinks, a bench meter's reading, a calculator's LCD, a
// fourteen-segment sign in each LED color, and a dot-matrix departure
// board scrolling its message.
//
//	go run ./examples/segments
package main

import (
	"fmt"
	"log"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/segments"
)

type app struct{}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	now := c.Now()
	sep := ":"
	if now.Second()%2 == 1 {
		sep = " "
	}
	c.After(time.Second - time.Duration(now.Nanosecond()))
	labeled := func(label string, build func()) {
		ui.Column(c.Key(label)).Gap(t.Space(2)).Children(func() {
			ui.Text(c, label).FontSize(t.FontSize - 1).TextColor(t.TextMuted)
			build()
		})
	}
	ui.Scroll(c).Fill().Background(t.Background).Children(func() {
		ui.Column(c).Padding(t.Space(6)).Gap(t.Space(5)).Children(func() {
			ui.Row(c).Gap(t.Space(6)).Wrap().Children(func() {
				labeled("Clock", func() {
					segments.View(c, segments.Panel{Kind: segments.Seven, Cells: 4, Height: 64}, now.Format("15")+sep+now.Format("04"))
				})
				labeled("Bench meter", func() {
					segments.View(c, segments.Panel{Kind: segments.Seven, Look: segments.VFD, Cells: 6, Height: 40}, fmt.Sprintf("-%.3f", 12.0+float64(now.Second())/100))
				})
			})
			labeled("Calculator", func() {
				segments.View(c, segments.Panel{Kind: segments.Seven, Look: segments.LCD, Cells: 10, Height: 36}, "3.141592653")
			})
			labeled("Sign", func() {
				ui.Column(c).Gap(t.Space(2)).Children(func() {
					for i, l := range []segments.Look{segments.RedLED, segments.GreenLED, segments.AmberLED} {
						segments.View(c.Key(i), segments.Panel{Kind: segments.Fourteen, Look: l, Cells: 12, Height: 36}, []string{"OPEN 24 HRS", "WAY OUT", "NO VACANCY"}[i])
					}
				})
			})
			labeled("Departures", func() {
				segments.View(c, segments.Panel{Kind: segments.Matrix, Look: segments.AmberLED, Cells: 24, Height: 28},
					"08:42 Harbor Terminal  Platform 2  On time")
			})
		})
	})
}

func main() {
	a := &app{}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Segments",
			Width:   780,
			Height:  680,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
