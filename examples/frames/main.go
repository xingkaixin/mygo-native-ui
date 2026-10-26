// Frames shows a made-up market feed's WebSocket as it talks: the
// connection opening, a subscription sent, prices received twice a
// second, pings and pongs, and now and then a snapshot in binary. Click a
// message to see it whole.
//
//	go run ./examples/frames
package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"math/rand/v2"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/frames"
)

type app struct {
	win    *mygo.Window
	frames []frames.Frame
	view   frames.State
}

func (a *app) add(f frames.Frame) {
	f.Time = time.Now()
	a.win.Update(func() { a.frames = append(a.frames, f) })
}

// feed plays the connection, from another goroutine.
func (a *app) feed() {
	a.add(frames.Frame{Control: true, Data: []byte("Connection opened: wss://feed.example.com/v2")})
	time.Sleep(300 * time.Millisecond)
	a.add(frames.Frame{Sent: true, Data: []byte(`{"op":"subscribe","channels":["ticker"],"symbols":["BTC-USD","ETH-USD","SOL-USD"]}`)})
	a.add(frames.Frame{Data: []byte(`{"type":"subscribed","channels":["ticker"],"id":"sub_8c1f"}`)})
	prices := map[string]float64{"BTC-USD": 67210.5, "ETH-USD": 3412.8, "SOL-USD": 171.32}
	symbols := []string{"BTC-USD", "ETH-USD", "SOL-USD"}
	for n := 1; ; n++ {
		time.Sleep(500 * time.Millisecond)
		sym := symbols[rand.IntN(len(symbols))]
		prices[sym] *= 1 + (rand.Float64()-0.5)/500
		a.add(frames.Frame{Data: fmt.Appendf(nil, `{"type":"ticker","symbol":%q,"price":%.2f,"size":%.4f,"seq":%d}`,
			sym, prices[sym], rand.Float64()*2, 41200+n)})
		if n%10 == 0 {
			a.add(frames.Frame{Sent: true, Control: true, Data: []byte("Ping")})
			a.add(frames.Frame{Control: true, Data: []byte("Pong")})
		}
		if n%25 == 0 {
			snap := make([]byte, 0, 80)
			for _, s := range symbols {
				snap = append(snap, s[:3]...)
				snap = binary.BigEndian.AppendUint64(snap, uint64(prices[s]*100))
			}
			a.add(frames.Frame{Binary: true, Data: snap})
		}
	}
}

func (a *app) render(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Children(func() {
		frames.View(c, &a.view, a.frames).Grow(1)
	})
}

func main() {
	a := &app{}
	mygo.App.WhenReady(func() {
		a.win = mygo.NewWindow(mygo.WindowOptions{
			Title:   "Messages",
			Width:   760,
			Height:  560,
			Content: ui.View(a.render),
		})
		go a.feed()
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
