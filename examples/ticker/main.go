// Ticker runs a tape of made-up stocks whose prices drift every second;
// click a quote to look at it.
//
//	go run ./examples/ticker
package main

import (
	"fmt"
	"log"
	"math"
	"math/rand/v2"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/ticker"
)

var names = map[string]string{
	"NWND": "Northwind Traders", "CNTS": "Contoso", "FABR": "Fabrikam", "TLSP": "Tailspin Toys",
	"WGIM": "Wingtip Imports", "ADVW": "Adventure Works", "LITW": "Litware", "PROS": "Proseware",
	"WWIM": "Wide World Importers", "FRTH": "Fourth Coffee",
}

type app struct {
	tape   ticker.State
	quotes []ticker.Quote
	symbol string
	moved  time.Time
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	now := c.Now()
	if now.Sub(a.moved) >= time.Second {
		a.moved = now
		for range 3 {
			q := &a.quotes[rand.IntN(len(a.quotes))]
			step := math.Round(q.Price*(rand.Float64()-0.5)*0.01*100) / 100
			q.Price += step
			q.Change += step
		}
	}
	c.After(time.Second - now.Sub(a.moved))
	ui.Column(c).Fill().Padding(t.Space(8)).Gap(t.Space(8)).Background(t.Background).Children(func() {
		if ev := ticker.View(c, &a.tape, a.quotes); ev.Chosen != "" {
			a.symbol = ev.Chosen
		}
		for _, q := range a.quotes {
			if q.Symbol != a.symbol {
				continue
			}
			ui.Column(c).Gap(t.Space(1)).Children(func() {
				ui.Text(c, names[q.Symbol]).TextColor(t.TextMuted)
				ui.Text(c, fmt.Sprintf("%s  %.2f", q.Symbol, q.Price)).FontSize(t.FontSize + 18).Bold().FontFeatures("tnum")
				ui.Text(c, fmt.Sprintf("%+.2f today", q.Change)).FontFeatures("tnum")
			})
		}
	})
}

func main() {
	a := &app{symbol: "NWND"}
	for _, s := range []string{"NWND", "CNTS", "FABR", "TLSP", "WGIM", "ADVW", "LITW", "PROS", "WWIM", "FRTH"} {
		price := 20 + rand.Float64()*380
		a.quotes = append(a.quotes, ticker.Quote{Symbol: s, Price: math.Round(price*100) / 100, Change: math.Round(price*(rand.Float64()-0.45)*4) / 100})
	}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Ticker",
			Width:   900,
			Height:  320,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
