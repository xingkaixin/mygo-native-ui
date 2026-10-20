// Package orderbook shows the orders resting on a market, as trading apps
// do beside a chart: the asks in red above, the highest at the top, and
// the bids in green below, the highest at the top, with a row between
// them of the mid price, halfway between the best bid and the best ask,
// and the spread between them. Each level shows its price, its size and
// the running total from the best price to it, in front of a bar as long
// as that total, so that where the book is deep shows at a glance. A size
// that changed flashes, green when it grew and red when it shrank. As
// many levels show as fit.
//
// A menu above the book groups the prices by a tick of 0.01, 0.1, 1 or
// 10, the bids down to it and the asks up to it, summing the sizes of the
// levels in each. A click on a level reports its price, which the app
// reads in the same frame, as to fill in an order's price.
//
// Depth draws the book as a depth chart: the running totals of the bids
// as a green area stepping up to the left of the mid price, and those of
// the asks as a red one to the right. A crosshair follows the pointer,
// marking the price under it and the total bid or asked up to there.
//
//	ev := orderbook.View(c, &app.book, app.bids, app.asks)
//	if price, ok := ev.Clicked(); ok {
//		app.limit = price
//	}
//	orderbook.Depth(c, app.bids, app.asks).Grow(1)
package orderbook

import (
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/chart"
)

// ticks are the steps the menu groups prices by.
var ticks = []float64{0.01, 0.1, 1, 10}

// flashFor is how long a size that changed flashes.
const flashFor = 700 * time.Millisecond

// Level is the orders resting at a price: the price and their size in
// all, as in coins or shares.
type Level struct {
	Price, Size float64
}

// State is what a book keeps: the tick its prices are grouped by, and the
// size of each level as it last showed and when it changed, for the
// flashes. Keep it in the app's state, one for each book. The zero State
// groups by 0.01.
type State struct {
	tick  int
	marks map[key]mark
}

// key is a level as grouped: its side and its price in ticks.
type key struct {
	ask bool
	at  int64
}

// mark is what a level last showed: its size, and when it last changed
// and whether it grew then.
type mark struct {
	size float64
	at   time.Time
	grew bool
}

// Events is what the user did to a book in a frame.
type Events struct {
	price   float64
	clicked bool
}

// Clicked returns the price of the level clicked, as grouped.
func (e Events) Clicked() (float64, bool) { return e.price, e.clicked }

// View creates the book of bids, from the highest price down, and asks,
// from the lowest up. It fills its parent.
func View(c *ui.Context, s *State, bids, asks []Level) Events {
	t := c.Theme()
	var ev Events
	rowH, pad := t.FontSize+t.Space(2.5), t.Space(3)
	middleH := rowH + t.Space(4)
	small := t.FontSize - 1

	ui.Column(c).Fill().Children(func() {
		ui.Row(c).Gap(t.Space(2)).Padding(t.Space(2), pad).AlignItems(ui.Center).Children(func() {
			ui.Spacer(c)
			ui.Text(c, "Group").FontSize(small).TextColor(t.TextMuted)
			ui.MenuButton(c, label(ticks[s.tick]), func(m *ui.Menu) {
				for i, tick := range ticks {
					if m.Item(label(tick)).Checked(i == s.tick).Chosen() && i != s.tick {
						s.tick, s.marks = i, nil
					}
				}
			}).Label("Group prices by")
		})
		tick := ticks[s.tick]
		decimals := max(0, int(math.Round(-math.Log10(tick))))
		groups := [2][]Level{group(bids, tick, false), group(asks, tick, true)}
		s.mark(c.Now(), tick, groups)

		cells(c, ui.Row(c), pad, rowH, func(cell func(string, ui.Align) ui.Element) {
			for i, name := range []string{"Price", "Size", "Total"} {
				cell(name, []ui.Align{ui.Start, ui.End, ui.End}[i]).FontSize(small).TextColor(t.TextMuted)
			}
		})

		list := ui.Column(c).Grow(1).Clip()
		b := list.LayoutBounds()
		list.Draw(func(_ *ui.Painter, r ui.Rect) {
			// As many levels show as fit the list's height in the last
			// frame: changed since, the view builds again.
			if r.H != b.H {
				c.Invalidate()
			}
		})
		n := max(0, int((b.H-middleH)/2/rowH))
		var totals [2][]float64
		most := 0.0
		for k := range groups {
			groups[k] = groups[k][:min(n, len(groups[k]))]
			sum := 0.0
			for _, l := range groups[k] {
				sum += l.Size
				totals[k] = append(totals[k], sum)
			}
			most = max(most, sum)
		}

		level := func(k, i int) {
			l, total := groups[k][i], totals[k][i]
			color := t.Success
			if k == 1 {
				color = t.Danger
			}
			m := s.marks[key{k == 1, ticksAt(l.Price, tick)}]
			row := cells(c, ui.ButtonBase(c.Key(i)), pad, rowH, func(cell func(string, ui.Align) ui.Element) {
				cell(price(l.Price, decimals), ui.Start).TextColor(color)
				size := amount(l.Size)
				flash := t.Success
				if !m.grew {
					flash = t.Danger
				}
				cell(size, ui.End).Draw(func(p *ui.Painter, r ui.Rect) {
					f := 1 - float32(p.Now().Sub(m.at))/float32(flashFor)
					if f <= 0 {
						return
					}
					p.AnimationFrame()
					w, h := p.MeasureText(0, ui.Span{Text: size, Size: t.FontSize, Features: "tnum"})
					in := t.Space(1)
					p.Fill(ui.Rect{X: r.X + r.W - w - in, Y: r.Y + (r.H-h)/2 - 1, W: w + 2*in, H: h + 2}, flash.Alpha(0.35*f), 3)
				})
				cell(amount(total), ui.End).TextColor(t.TextMuted)
			})
			if row.Hovered() {
				row.Background(t.Text.Alpha(0.06))
			}
			if row.Clicked() {
				ev.price, ev.clicked = l.Price, true
			}
			row.Draw(func(p *ui.Painter, r ui.Rect) {
				if most > 0 {
					w := r.W * float32(total/most)
					p.Fill(ui.Rect{X: r.X + r.W - w, Y: r.Y + 1, W: w, H: r.H - 2}, color.Alpha(0.14), 0)
				}
			})
		}

		list.Children(func() {
			ui.Column(c).Grow(1).Justify(ui.End).Children(func() {
				for i := len(groups[1]) - 1; i >= 0; i-- {
					level(1, i)
				}
			})
			middle(c, bids, asks, pad, middleH)
			ui.Column(c).Grow(1).Children(func() {
				for i := range groups[0] {
					level(0, i)
				}
			})
		})
	})
	return ev
}

// cells lays out row as three columns, the first wider, which build fills
// with cell, of text aligned in its column.
func cells(c *ui.Context, row ui.Element, pad, h float32, build func(cell func(string, ui.Align) ui.Element)) ui.Element {
	row.Height(h).Padding(0, pad).Gap(c.Theme().Space(2)).AlignItems(ui.Center).Children(func() {
		build(func(s string, a ui.Align) ui.Element {
			grow := float32(1)
			if a == ui.Start {
				grow = 1.2
			}
			return ui.Text(c, s).Grow(grow).Basis(0).TextAlign(a).FontFeatures("tnum").SingleLine()
		})
	})
	return row
}

// middle creates the row between the asks and the bids: the mid price and
// the spread.
func middle(c *ui.Context, bids, asks []Level, pad, h float32) {
	t := c.Theme()
	mid, spread := "–", "–"
	if len(bids) > 0 && len(asks) > 0 {
		bid, ask := bids[0].Price, asks[0].Price
		mid, spread = price((bid+ask)/2, 2), price(ask-bid, 2)
	}
	ui.Row(c).Height(h).Padding(0, pad).Gap(t.Space(2)).AlignItems(ui.Center).Background(t.Text.Alpha(0.04)).Children(func() {
		ui.Text(c, mid).Bold().FontSize(t.FontSize + 3).FontFeatures("tnum")
		ui.Text(c, "Mid").FontSize(t.FontSize - 1).TextColor(t.TextMuted)
		ui.Spacer(c)
		ui.Text(c, "Spread").FontSize(t.FontSize - 1).TextColor(t.TextMuted)
		ui.Text(c, spread).FontFeatures("tnum")
	})
}

// mark notes the size of each level of groups, the bids and the asks, and
// when it changed. In the first frame, and the first after the tick
// changed, none changed.
func (s *State) mark(now time.Time, tick float64, groups [2][]Level) {
	first := s.marks == nil
	marks := make(map[key]mark, len(groups[0])+len(groups[1]))
	for k, levels := range groups {
		for _, l := range levels {
			id := key{k == 1, ticksAt(l.Price, tick)}
			m := s.marks[id]
			if !first && m.size != l.Size {
				m.at, m.grew = now, l.Size > m.size
			}
			m.size = l.Size
			marks[id] = m
		}
	}
	s.marks = marks
}

// group sums the sizes of levels by tick, in their order, each at its
// price down to a whole tick, or up for asks.
func group(levels []Level, tick float64, up bool) []Level {
	var out []Level
	for _, l := range levels {
		// The slack keeps a price on a tick, as 67012.24 at 0.01, from
		// falling to the tick below in floating point.
		k := math.Floor(l.Price/tick + 1e-6)
		if up {
			k = math.Ceil(l.Price/tick - 1e-6)
		}
		p := k * tick
		if tick < 1 {
			// Dividing by a whole number gives the float nearest the
			// decimal price, 67012.24 rather than 67012.240000000005.
			p = k / math.Round(1/tick)
		}
		if n := len(out); n > 0 && out[n-1].Price == p {
			out[n-1].Size += l.Size
		} else {
			out = append(out, Level{p, l.Size})
		}
	}
	return out
}

// ticksAt returns price, on a tick, in ticks.
func ticksAt(price, tick float64) int64 { return int64(math.Round(price / tick)) }

// Depth creates the depth chart of bids, from the highest price down, and
// asks, from the lowest up, as far from the mid price on each side as the
// nearer of the farthest bid and ask. Give it a size, or grow it in its
// parent.
func Depth(c *ui.Context, bids, asks []Level) ui.Element {
	t := c.Theme()
	size, pad := t.FontSize-2, t.Space(1.5)
	box := ui.Box(c).MinHeight(t.Space(40)).Cursor(ui.CursorCrosshair).Label("Depth chart")
	if len(bids) == 0 && len(asks) == 0 {
		return box
	}
	var mid, half float64
	switch {
	case len(bids) == 0:
		mid, half = asks[0].Price, asks[len(asks)-1].Price-asks[0].Price
	case len(asks) == 0:
		mid, half = bids[0].Price, bids[0].Price-bids[len(bids)-1].Price
	default:
		mid = (bids[0].Price + asks[0].Price) / 2
		half = min(mid-bids[len(bids)-1].Price, asks[len(asks)-1].Price-mid)
	}
	if half <= 0 {
		half = max(1, mid*0.001)
	}
	top := max(total(bids, mid, half), total(asks, mid, half))
	prices := chart.NewLinear(mid-half, mid+half)
	totals := chart.NewLinear(0, max(1, top*1.15))
	tw, th := c.MeasureText(0, ui.Span{Text: amount(totals.Hi), Size: size})
	axisW, axisH := tw+2*pad, th+2*pad

	r := box.LayoutBounds()
	px, py, over := box.PointerPosition()
	over = over && px < r.W-axisW && py < r.H-axisH
	var at, sum float64
	color := t.Success
	if over {
		at = prices.Invert(px / (r.W - axisW))
		side, name := bids, "bid"
		if at > mid {
			side, name, color = asks, "asked", t.Danger
		}
		sum = total(side, mid, math.Abs(at-mid))
		box.Value(price(at, 2) + ", " + amount(sum) + " " + name)
	}

	box.Draw(func(p *ui.Painter, r ui.Rect) {
		plot := ui.Rect{X: r.X, Y: r.Y, W: r.W - axisW, H: r.H - axisH}
		x := func(v float64) float32 { return chart.X(plot, prices.At(v)) }
		y := func(v float64) float32 { return chart.Y(plot, totals.At(v)) }
		grid := t.Border.Alpha(0.5)
		for _, tick := range totals.Ticks(max(2, int(plot.H/50))) {
			ty := chart.Y(plot, tick.At)
			p.Line(plot.X, ty, plot.X+plot.W, ty, 1, grid)
			if ty-th/2 >= r.Y {
				p.Text(plot.X+plot.W+pad, ty-th/2, commas(tick.Label), size, t.TextMuted)
			}
		}
		right := float32(math.Inf(-1))
		for _, tick := range prices.Ticks(max(2, int(plot.W/110))) {
			tx := chart.X(plot, tick.At)
			label := commas(tick.Label)
			w, _ := p.MeasureText(0, ui.Span{Text: label, Size: size})
			if tx-w/2 < max(r.X, right+pad) || tx+w/2 > plot.X+plot.W {
				continue
			}
			p.Line(tx, plot.Y, tx, plot.Y+plot.H, 1, grid)
			p.Text(tx-w/2, plot.Y+plot.H+pad, label, size, t.TextMuted)
			right = tx + w/2
		}
		p.Line(plot.X+plot.W, plot.Y, plot.X+plot.W, plot.Y+plot.H, 1, t.Border)
		p.Line(plot.X, plot.Y+plot.H, plot.X+plot.W, plot.Y+plot.H, 1, t.Border)

		p.Clip(plot, 0, func() {
			for k, levels := range [2][]Level{bids, asks} {
				if len(levels) == 0 {
					continue
				}
				line := t.Success
				if k == 1 {
					line = t.Danger
				}
				pts := steps(levels, mid, half, x, y)
				var area ui.Path
				area.MoveTo(pts[0][0], pts[0][1])
				for _, q := range pts[1:] {
					area.LineTo(q[0], q[1])
				}
				area.LineTo(pts[len(pts)-1][0], pts[0][1]).Close()
				p.FillPath(&area, line.Alpha(0.18))
				p.StrokePolyline(pts, 1.5, line)
			}
		})
		label := price(mid, 2)
		w, _ := p.MeasureText(0, ui.Span{Text: label, Size: size})
		p.Text(x(mid)-w/2, plot.Y+pad, label, size, t.TextMuted)
		dashed(p, x(mid), plot.Y+th+2*pad, x(mid), plot.Y+plot.H, t.TextMuted)

		if !over {
			return
		}
		cx, cy := plot.X+px, y(sum)
		dashed(p, cx, plot.Y, cx, plot.Y+plot.H, t.TextMuted)
		dashed(p, plot.X, cy, plot.X+plot.W, cy, t.TextMuted)
		var dot ui.Path
		dot.Circle(cx, cy, 4)
		p.FillPath(&dot, color)
		p.StrokePath(&dot, 1.5, t.Background)
		tag := func(rect ui.Rect, s string) {
			p.Fill(rect, t.Text, 3)
			p.Text(rect.X+pad, rect.Y+(rect.H-th)/2, s, size, t.Background)
		}
		tag(ui.Rect{X: plot.X + plot.W, Y: cy - th/2 - pad/2, W: axisW, H: th + pad}, amount(sum))
		label = price(at, 2)
		w, _ = p.MeasureText(0, ui.Span{Text: label, Size: size})
		w += 2 * pad
		tag(ui.Rect{X: max(plot.X, min(plot.X+plot.W-w, cx-w/2)), Y: plot.Y + plot.H, W: w, H: axisH}, label)
	})
	return box
}

// total returns the size of levels within within of mid.
func total(levels []Level, mid, within float64) float64 {
	sum := 0.0
	for _, l := range levels {
		if math.Abs(l.Price-mid) > within {
			break
		}
		sum += l.Size
	}
	return sum
}

// steps returns the points of the line of the running totals of levels
// within half of mid, rising at each level's price and ending at the
// edge.
func steps(levels []Level, mid, half float64, x, y func(float64) float32) [][2]float32 {
	edge := mid - half
	if levels[0].Price > mid {
		edge = mid + half
	}
	pts := [][2]float32{{x(levels[0].Price), y(0)}}
	sum := 0.0
	for _, l := range levels {
		if math.Abs(l.Price-mid) > half {
			break
		}
		at := x(l.Price)
		pts = append(pts, [2]float32{at, y(sum)})
		sum += l.Size
		pts = append(pts, [2]float32{at, y(sum)})
	}
	return append(pts, [2]float32{x(edge), y(sum)})
}

// dashed paints a dashed line from (x0, y0) to (x1, y1), across or down.
func dashed(p *ui.Painter, x0, y0, x1, y1 float32, c ui.Color) {
	const dash, gap = 4, 3
	if x0 == x1 {
		for y := y0; y < y1; y += dash + gap {
			p.Fill(ui.Rect{X: x0 - 0.5, Y: y, W: 1, H: min(dash, y1-y)}, c, 0)
		}
		return
	}
	for x := x0; x < x1; x += dash + gap {
		p.Fill(ui.Rect{X: x, Y: y0 - 0.5, W: min(dash, x1-x), H: 1}, c, 0)
	}
}

// label writes a tick as the menu shows it: 0.01.
func label(tick float64) string { return strconv.FormatFloat(tick, 'f', -1, 64) }

// price writes v with decimals decimals and commas: 67,012.25.
func price(v float64, decimals int) string {
	return commas(strconv.FormatFloat(v, 'f', decimals, 64))
}

// amount writes a size with four decimals and commas: 1,204.5000.
func amount(v float64) string { return price(v, 4) }

// commas puts commas between the thousands of a number written out.
func commas(s string) string {
	whole, frac, dot := strings.Cut(s, ".")
	sign := ""
	if strings.HasPrefix(whole, "-") {
		sign, whole = "-", whole[1:]
	}
	for i := len(whole) - 3; i > 0; i -= 3 {
		whole = whole[:i] + "," + whole[i:]
	}
	if dot {
		return sign + whole + "." + frac
	}
	return sign + whole
}
