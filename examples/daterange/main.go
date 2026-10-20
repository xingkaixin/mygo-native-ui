// Daterange shows a site's visits over the days chosen, as analytics
// dashboards do: the button above opens the picker.
//
//	go run ./examples/daterange
package main

import (
	"fmt"
	"log"
	"math"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/daterange"
)

type app struct {
	from, to time.Time
	picker   daterange.State
	open     bool
}

// visits makes up a day's visits: more on weekdays, growing over the year.
func visits(d time.Time) int {
	n := 900 + 300*math.Sin(float64(d.YearDay())/9) + float64(d.YearDay())*2
	if wd := d.Weekday(); wd == time.Saturday || wd == time.Sunday {
		n *= 0.6
	}
	return int(n) + (d.Day()*37)%120
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(6)).Gap(t.Space(4)).Children(func() {
		ui.Row(c).AlignItems(ui.Center).Gap(t.Space(3)).Children(func() {
			ui.Text(c, "Visits").Bold().FontSize(t.FontSize + 6).Grow(1)
			b := ui.Button(c, a.from.Format("Jan 2, 2006")+" – "+a.to.Format("Jan 2, 2006")).Label("Range")
			if b.Clicked() {
				a.open = !a.open
			}
			ui.PopoverBase(c, b, &a.open, func(panel ui.Element) {
				panel.Shadow(0, 8, 24, 0, ui.RGB(0, 0, 0).Alpha(0.18)).Radius(t.Space(2))
				if daterange.Picker(c, &a.picker, &a.from, &a.to, c.Now()).Changed() {
					a.open = false
				}
			})
		})
		var days []int
		total := 0
		for d := a.from; !d.After(a.to); d = d.AddDate(0, 0, 1) {
			days = append(days, visits(d))
			total += days[len(days)-1]
		}
		ui.Text(c, fmt.Sprintf("%d visits over %d days", total, len(days))).TextColor(t.TextMuted)
		ui.Box(c).Grow(1).Draw(func(p *ui.Painter, r ui.Rect) {
			top := 1
			for _, n := range days {
				top = max(top, n)
			}
			w := r.W / float32(len(days))
			for i, n := range days {
				h := r.H * float32(n) / float32(top)
				p.Fill(ui.Rect{X: r.X + float32(i)*w + w*0.15, Y: r.Y + r.H - h, W: max(1, w*0.7), H: h}, t.Accent, min(3, w*0.2))
			}
		})
	})
}

func main() {
	y, m, d := time.Now().Date()
	today := time.Date(y, m, d, 0, 0, 0, 0, time.Local)
	a := &app{from: today.AddDate(0, 0, -29), to: today}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Date range",
			Width:   960,
			Height:  640,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
