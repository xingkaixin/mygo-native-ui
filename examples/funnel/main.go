// Funnel shows how made-up visitors of a product went on to sign up, start
// using it and pay in October, as product analytics do. Choose a platform,
// compare with September, and turn the funnel on its side; point at a
// stage to read its numbers, and click it to note it below.
//
//	go run ./examples/funnel
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/funnel"
)

var stages = []string{"Visited", "Signed up", "Activated", "Paid"}

var platforms = []string{"All", "Web", "iOS", "Android"}

// counts holds each month's counts at each stage, by platform, All left
// out: it is their sum.
var counts = map[string][3][]float64{
	"October":   {{48200, 9640, 5300, 1210}, {21500, 5160, 3410, 980}, {17900, 3220, 1610, 290}},
	"September": {{45100, 8120, 4060, 1020}, {19800, 4360, 2700, 840}, {16300, 2610, 1170, 230}},
}

type app struct {
	platform int
	compare  bool
	side     int
	clicked  string
}

// segment returns month's counts of the platform chosen.
func (a *app) segment(month string) funnel.Segment {
	by := counts[month]
	seg := funnel.Segment{Name: month, Counts: make([]float64, len(stages))}
	for p, cs := range by {
		if a.platform == 0 || a.platform == p+1 {
			for i, n := range cs {
				seg.Counts[i] += n
			}
		}
	}
	return seg
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	segments := []funnel.Segment{a.segment("October")}
	if a.compare {
		segments = append(segments, a.segment("September"))
	}
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(5), t.Space(6)).Gap(t.Space(4)).Children(func() {
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "Signups in October").Bold().FontSize(t.FontSize + 6).Grow(1)
			ui.Segmented(c, &a.platform, platforms...).Label("Platform")
			ui.Text(c, "Compare with September").TextColor(t.TextMuted)
			ui.Switch(c, &a.compare).Label("Compare with September")
			ui.Segmented(c, &a.side, "Down", "Across").Label("Direction")
		})
		dir := funnel.Vertical
		if a.side == 1 {
			dir = funnel.Horizontal
		}
		ui.Column(c).Grow(1).AlignSelf(ui.Stretch).Children(func() {
			if i, ok := funnel.View(c, dir, stages, segments).Clicked(); ok {
				a.clicked = stages[i]
			}
		})
		line := "Click a stage to note it"
		if a.clicked != "" {
			line = "Clicked " + a.clicked
		}
		ui.Text(c, line).TextColor(t.TextMuted)
	})
}

func main() {
	a := &app{}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Funnel",
			Width:   1080,
			Height:  640,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
