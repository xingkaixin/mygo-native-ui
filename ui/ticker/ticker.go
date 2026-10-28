// Package ticker runs a tape of quotes across a strip, as markets show
// prices: each symbol with its price and its change for the day, up in
// green with ▲ or down in red with ▼, the tape looping without a seam. A
// quote whose price moves flashes in the color of the move. Pointing at the
// tape stops it, and a click on a quote chooses it. With reduced motion
// the tape stands still and scrolls by hand.
//
//	if ev := ticker.View(c, &app.tape, app.quotes); ev.Chosen != "" {
//		app.symbol = ev.Chosen
//	}
package ticker

import (
	"fmt"
	"math"
	"time"

	"github.com/egoist/mygo/ui"
)

// Quote is a symbol's price and how far it has moved today.
type Quote struct {
	Symbol string
	Price  float64
	Change float64
}

// State is what a tape keeps: how far it has run, and the prices last
// seen, to flash those that move. Keep it in the app's state, one for each
// tape.
type State struct {
	offset float64
	width  float64
	last   time.Time
	prices map[string]float64
	moved  map[string]time.Time
}

// Events is a tape, and the symbol of the quote clicked in the frame, or
// "".
type Events struct {
	ui.Element
	Chosen string
}

const (
	speed = 60 // DIPs a second
	flash = time.Second
)

var (
	up   = ui.Hex("#2F9E44")
	down = ui.Hex("#E03131")
)

// View creates the tape of quotes.
func View(c *ui.Context, s *State, quotes []Quote) Events {
	t := c.Theme()
	var ev Events
	now := c.Now()
	if s.prices == nil {
		s.prices, s.moved = map[string]float64{}, map[string]time.Time{}
	}
	for _, q := range quotes {
		if p, ok := s.prices[q.Symbol]; ok && p != q.Price {
			s.moved[q.Symbol] = now
		}
		s.prices[q.Symbol] = q.Price
	}
	still := c.Preferences().ReduceMotion
	strip := ui.Box(c).FillWidth().Height(t.Space(11)).Clip().Background(t.Surface).
		Border(1, t.Border).Radius(t.Space(2)).Role(ui.RoleList).Label("Quotes")
	dt := min(now.Sub(s.last), 100*time.Millisecond)
	s.last = now
	if !still && !strip.Hovered() && len(quotes) > 0 {
		s.offset += speed * dt.Seconds()
		c.AnimationFrame()
	}
	if s.width > 0 {
		s.offset = math.Mod(s.offset, s.width)
	}
	for _, at := range s.moved {
		if now.Sub(at) < flash {
			c.AnimationFrame()
		}
	}
	tape := func(copies int) ui.Element {
		row := ui.Row(c).AlignItems(ui.Center).FillHeight()
		row.Children(func() {
			for k := range copies {
				for i, q := range quotes {
					if quote(c.Key(fmt.Sprint(k, "/", i)), q, now.Sub(s.moved[q.Symbol])) {
						ev.Chosen = q.Symbol
					}
				}
			}
		})
		return row
	}
	strip.Children(func() {
		if still {
			ui.ScrollHorizontal(c).Fill().Children(func() { tape(1) })
			return
		}
		row := tape(2).Absolute().Left(-float32(s.offset)).Top(0)
		s.width = float64(row.Bounds().W) / 2
	})
	ev.Element = strip
	return ev
}

// quote creates a quote on the tape, flashing for a while after its price
// moved, and reports whether it was clicked.
func quote(c *ui.Context, q Quote, since time.Duration) bool {
	t := c.Theme()
	col, arrow := up, "▲"
	if q.Change < 0 {
		col, arrow = down, "▼"
	}
	pct := 0.0
	if open := q.Price - q.Change; open != 0 {
		pct = q.Change / open * 100
	}
	b := ui.ButtonBase(c).Gap(t.Space(2)).AlignItems(ui.Center).Padding(t.Space(1), t.Space(3)).Radius(t.Space(1.5)).
		Margin(0, t.Space(1)).Cursor(ui.CursorPointer).Label(q.Symbol).
		Value(fmt.Sprintf("%.2f, %+.2f%%", q.Price, pct))
	switch {
	case since >= 0 && since < flash:
		b.Background(col.Alpha(0.25 * float32(1-float64(since)/float64(flash))))
	case b.Hovered():
		b.Background(t.SurfaceHover)
	}
	b.Children(func() {
		ui.Text(c, q.Symbol).Bold()
		ui.Text(c, fmt.Sprintf("%.2f", q.Price)).FontFeatures("tnum")
		ui.Text(c, fmt.Sprintf("%s %.2f (%.2f%%)", arrow, math.Abs(q.Change), math.Abs(pct))).
			FontSize(t.FontSize - 1).FontFeatures("tnum").TextColor(col)
	})
	return b.Clicked()
}
