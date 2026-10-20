// Orderbook shows a simulated BTC-USD order book changing five times a
// second, with its depth chart beside it: the price drifts, trades take
// the best levels, and orders come, go and change size. Group the prices
// with the menu, and click a level to make its price the order's.
//
//	go run ./examples/orderbook
package main

import (
	"log"
	"math"
	"math/rand/v2"
	"strconv"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/orderbook"
)

const (
	every  = 200 * time.Millisecond
	levels = 400
)

// market is a made-up book around a fair price, each side best first.
type market struct {
	r          *rand.Rand
	fair       float64
	bids, asks []orderbook.Level
}

func newMarket(fair float64, seed uint64) *market {
	m := &market{r: rand.New(rand.NewPCG(seed, seed)), fair: fair}
	m.bids = m.extend(nil, -1)
	m.asks = m.extend(nil, 1)
	return m
}

func cents(v float64) float64 { return math.Round(v*100) / 100 }

// gap is how far the next level is from one dist from the fair price:
// levels thin out away from it.
func (m *market) gap(dist float64) float64 {
	return max(0.01, cents(m.r.ExpFloat64()*(0.2+0.005*dist)))
}

// size is how much rests at a level dist from the fair price, now and
// then a wall of orders.
func (m *market) size(dist float64) float64 {
	s := 0.01 + m.r.ExpFloat64()*0.35*(1+dist/100)
	if m.r.Float64() < 0.03 {
		s *= 5 + 10*m.r.Float64()
	}
	return math.Round(s*1e4) / 1e4
}

// extend adds levels past the last of side, below the fair price for
// bids (dir -1) and above it for asks, until it has enough.
func (m *market) extend(side []orderbook.Level, dir float64) []orderbook.Level {
	p := cents(m.fair + dir*0.01)
	for len(side) < levels {
		if n := len(side); n > 0 {
			p = cents(side[n-1].Price + dir*m.gap(math.Abs(side[n-1].Price-m.fair)))
		}
		side = append(side, orderbook.Level{Price: p, Size: m.size(math.Abs(p - m.fair))})
	}
	return side
}

// refill takes the levels of side the fair price passed, as trades would,
// and puts new orders between it and the best left.
func (m *market) refill(side []orderbook.Level, dir float64) []orderbook.Level {
	i := 0
	for i < len(side) && (side[i].Price-m.fair)*dir <= 0 {
		i++
	}
	side = side[i:]
	var fresh []orderbook.Level
	p := cents(m.fair + dir*(0.01+0.3*m.r.ExpFloat64()))
	for len(side) > 0 && (side[0].Price-p)*dir > 0.005 {
		fresh = append(fresh, orderbook.Level{Price: p, Size: m.size(0)})
		p = cents(p + dir*m.gap(0))
	}
	return append(fresh, side...)
}

// churn changes a few levels near the best of side: a size goes up or
// down, an order is filled or canceled, or a new one comes between two.
func (m *market) churn(side []orderbook.Level, dir float64) []orderbook.Level {
	for range 4 {
		i := int(m.r.ExpFloat64() * 20)
		if i >= len(side)-1 {
			continue
		}
		switch x := m.r.Float64(); {
		case x < 0.7:
			side[i].Size = max(0.0001, math.Round(side[i].Size*math.Exp(0.6*m.r.NormFloat64())*1e4)/1e4)
		case x < 0.85:
			side = append(side[:i], side[i+1:]...)
		case math.Abs(side[i+1].Price-side[i].Price) >= 0.02:
			p := cents((side[i].Price + side[i+1].Price) / 2)
			l := orderbook.Level{Price: p, Size: m.size(math.Abs(p - m.fair))}
			side = append(side[:i+1], append([]orderbook.Level{l}, side[i+1:]...)...)
		}
	}
	return side
}

// step moves the market on: the fair price drifts, the sides follow it
// and churn, and they keep about the same number of levels.
func (m *market) step() {
	m.fair += 1.5 * m.r.NormFloat64()
	m.bids = m.extend(m.churn(m.refill(m.bids, -1), -1), -1)[:levels]
	m.asks = m.extend(m.churn(m.refill(m.asks, 1), 1), 1)[:levels]
}

type app struct {
	market *market
	book   orderbook.State
	next   time.Time
	limit  string
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	now := c.Now()
	if !now.Before(a.next) {
		if !a.next.IsZero() {
			a.market.step()
		}
		a.next = now.Add(every)
	}
	c.After(a.next.Sub(now))

	m := a.market
	panel := func(e ui.Element) ui.Element {
		return e.Background(t.Background).Border(1, t.Border).Radius(t.Space(2)).Clip()
	}
	ui.Column(c).Fill().Background(t.Background).Padding(16).Gap(12).Children(func() {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "BTC-USD").Bold()
			ui.Text(c, "Simulated order book, five updates a second").TextColor(t.TextMuted)
			ui.Spacer(c)
			ui.Text(c, "Limit price").TextColor(t.TextMuted)
			ui.TextInput(c, &a.limit).Placeholder("Click a level").Width(150).FontFeatures("tnum")
		})
		ui.Row(c).Grow(1).Gap(12).AlignItems(ui.Stretch).Children(func() {
			panel(ui.Column(c).Width(380)).Children(func() {
				ev := orderbook.View(c, &a.book, m.bids, m.asks)
				if p, ok := ev.Clicked(); ok {
					a.limit = strconv.FormatFloat(p, 'f', -1, 64)
				}
			})
			panel(ui.Column(c).Grow(1).Padding(12).Gap(8)).Children(func() {
				ui.Text(c, "Depth").Bold()
				orderbook.Depth(c, m.bids, m.asks).Grow(1)
			})
		})
	})
}

func main() {
	a := &app{market: newMarket(67012.5, 11)}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Order book",
			Width:   1100,
			Height:  720,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
