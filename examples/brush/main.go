// Brush shows two hours of a service's log a minute to a bar, errors,
// warnings and the rest stacked, with a burst of errors near the end, and
// lists the latest events in the window chosen on them.
//
//	go run ./examples/brush
package main

import (
	"log"
	"math/rand/v2"
	"slices"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/brush"
)

type event struct {
	at    time.Time
	level int
	msg   string
}

var levels = []string{"ERROR", "WARN", "INFO"}

var messages = [][]string{
	{"upstream timed out after 30s", "payment declined: card_expired", "panic: nil map write in cart.Merge"},
	{"retrying request to inventory (2/3)", "slow query took 1.8s", "cache miss rate above 40%"},
	{"GET /api/orders 200", "POST /api/cart 201", "user signed in", "GET /api/products 200", "job export finished"},
}

type app struct {
	start  time.Time
	events []event
	series []brush.Series
	window brush.Range
	brush  brush.State
}

func newApp(now time.Time) *app {
	r := rand.New(rand.NewPCG(1, 2))
	start := now.Truncate(time.Minute).Add(-2 * time.Hour)
	a := &app{start: start}
	counts := make([][]int, 3)
	for i := range counts {
		counts[i] = make([]int, 120)
	}
	for m := range 120 {
		base := 30 + 20*float64(m%60)/60
		burst := m >= 96 && m < 104
		n := []int{r.IntN(2), r.IntN(4), int(base) + r.IntN(15)}
		if burst {
			n[0] += 20 + r.IntN(15)
			n[1] += 8
		}
		for lvl, k := range n {
			counts[lvl][m] += k
			for range k {
				at := start.Add(time.Duration(m)*time.Minute + time.Duration(r.IntN(60000))*time.Millisecond)
				a.events = append(a.events, event{at, lvl, messages[lvl][r.IntN(len(messages[lvl]))]})
			}
		}
	}
	slices.SortFunc(a.events, func(x, y event) int { return x.at.Compare(y.at) })
	a.series = []brush.Series{
		{Name: "Error", Color: ui.Oklch(0.6, 0.18, 25), Counts: counts[0]},
		{Name: "Warning", Color: ui.Oklch(0.75, 0.15, 75), Counts: counts[1]},
		{Name: "Info", Color: ui.Oklch(0.65, 0.1, 250), Counts: counts[2]},
	}
	return a
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(16).Gap(12).Children(func() {
		brush.View(c, &a.brush, a.start, time.Minute, a.series, &a.window)
		ui.Divider(c)
		var shown []event
		for i := len(a.events) - 1; i >= 0 && len(shown) < 200; i-- {
			e := a.events[i]
			if a.window.IsZero() || !e.at.Before(a.window.From) && e.at.Before(a.window.To) {
				shown = append(shown, e)
			}
		}
		ui.Scroll(c).Grow(1).Children(func() {
			ui.Column(c).Children(func() {
				for i, e := range shown {
					ui.Row(c.Key(i)).Gap(12).Padding(3, 4).Children(func() {
						ui.Text(c, e.at.Format("15:04:05.000")).Font("monospace").FontSize(12).TextColor(t.TextMuted)
						ui.Text(c, levels[e.level]).Font("monospace").FontSize(12).Width(44).TextColor(a.series[e.level].Color)
						ui.Text(c, e.msg).Font("monospace").FontSize(12)
					})
				}
			})
		})
	})
}

func main() {
	a := newApp(time.Now())
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Logs",
			Width:   900,
			Height:  620,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
