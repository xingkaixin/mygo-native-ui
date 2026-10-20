// Candles shows two years of simulated daily prices as a candlestick
// chart, with volume and the 20- and 60-day moving averages: 1M, 3M, 1Y
// and All show the last month, three months, year or all of it. The wheel
// zooms, dragging pans, and the arrows move the crosshair.
//
//	go run ./examples/candles
package main

import (
	"log"
	"math"
	"math/rand/v2"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/candles"
)

var spans = []string{"1M", "3M", "1Y", "All"}

type app struct {
	bars  []candles.Bar
	chart candles.State
	span  int
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(16).Gap(12).Children(func() {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "SIM").Bold()
			ui.Text(c, "Simulated daily prices from a seeded random walk").TextColor(t.TextMuted)
			ui.Spacer(c)
			if ui.Segmented(c, &a.span, spans...).Label("Range").Changed() {
				a.show()
			}
		})
		candles.View(c, &a.chart, a.bars, []int{20, 60}).Grow(1)
	})
}

// show shows the stretch of the span chosen, up to the last bar.
func (a *app) show() {
	first, last := a.bars[0].Time, a.bars[len(a.bars)-1].Time
	from := []time.Time{last.AddDate(0, -1, 0), last.AddDate(0, -3, 0), last.AddDate(-1, 0, 0), first}[a.span]
	a.chart.Show(from, last)
}

// walk makes up a bar for each weekday of the two years to end: the
// closes walk at random, seeded so that they come out the same each run,
// their volatility drifting so that calm and wild stretches alternate,
// and more trades on the days that move more.
func walk(end time.Time, seed uint64) []candles.Bar {
	r := rand.New(rand.NewPCG(seed, seed))
	var bars []candles.Bar
	last, vol := 100.0, 0.015
	for d := end.AddDate(-2, 0, 0); !d.After(end); d = d.AddDate(0, 0, 1) {
		if d.Weekday() == time.Saturday || d.Weekday() == time.Sunday {
			continue
		}
		vol = max(0.006, min(0.04, vol+0.08*(0.015-vol)+0.0015*r.NormFloat64()))
		open := last * (1 + 0.004*r.NormFloat64())
		last = open * math.Exp(0.0004+vol*r.NormFloat64())
		bars = append(bars, candles.Bar{
			Time:   d,
			Open:   open,
			High:   max(open, last) * (1 + 0.5*vol*math.Abs(r.NormFloat64())),
			Low:    min(open, last) * (1 - 0.5*vol*math.Abs(r.NormFloat64())),
			Close:  last,
			Volume: 2e6 * (0.5 + math.Abs(last/open-1)/vol) * math.Exp(0.3*r.NormFloat64()),
		})
	}
	return bars
}

func main() {
	a := &app{bars: walk(time.Date(2026, 10, 16, 0, 0, 0, 0, time.UTC), 7), span: 1}
	a.show()
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Candles",
			Width:   1000,
			Height:  640,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
