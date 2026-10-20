// Package candles shows prices as a candlestick chart, as trading apps
// do: a candle for each bar, its wick from the low to the high and its
// body from the open to the close, green for a bar that closed at or
// above its open and red below, the volume of each in a pane below, and
// moving averages of the closes as lines. Prices are on the right, and
// dates along the bottom in days, months or years as the zoom needs.
// Above the chart a line reads out the bar under the crosshair, or the
// last: its open, high, low and close, its change from the close before,
// and its volume; and a legend the value of each average there.
//
// The crosshair follows the pointer from bar to bar, marking the price at
// the pointer on the price axis and the bar's date on the time axis;
// while the chart has the focus, the left and right arrows move it a bar
// at a time. The wheel zooms around the pointer, dragging pans, and the
// prices fit the bars in view. Show shows a stretch of time:
//
//	candles.View(c, &app.chart, app.bars, []int{20, 60}).Grow(1)
//	...
//	app.chart.Show(last.AddDate(0, -3, 0), last) // the last three months
package candles

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/chart"
)

// fewest is the fewest bars a chart zooms in to.
const fewest = 10

const dateLayout = "Mon Jan 2, 2006"

// Bar is the trading of a stretch of time, as a day: when it began, its
// first, highest, lowest and last price, and how much traded.
type Bar struct {
	Time                           time.Time
	Open, High, Low, Close, Volume float64
}

// State is what a chart keeps: the bars it shows and the one under the
// crosshair. Keep it in the app's state, one for each chart. The zero
// State shows all the bars.
type State struct {
	// first is the bar at the left edge, in fractions of bars, and count
	// how many bars the chart shows across; 0 shows all.
	first, count float64
	// cross is the bar under the crosshair plus 1, so that the zero State
	// has none.
	cross int
	// over, px and py are whether the pointer was over the plot and
	// where, and keyed says the arrows moved the crosshair since the
	// pointer last moved.
	px, py      float32
	over, keyed bool
	// from and to are the stretch Show asked for, which the next build,
	// having the bars, turns into first and count.
	from, to time.Time
	show     bool
}

// Show shows the bars from from to to, from the next frame on.
func (s *State) Show(from, to time.Time) { s.from, s.to, s.show = from, to, true }

func (s *State) clamp(n int) {
	s.count = max(min(fewest, float64(n)), min(float64(n), s.count))
	s.first = max(0, min(float64(n)-s.count, s.first))
}

// zoomAt shows count bars, keeping the bar at x, a fraction of the plot's
// width, where it is.
func (s *State) zoomAt(count, x float64, n int) {
	at := s.first + x*s.count
	s.count = count
	s.clamp(n)
	s.first = at - x*s.count
	s.clamp(n)
}

// View creates the chart of bars, in the order of their times, with a
// moving average of the closes of each of averages' periods, in bars.
// Give it a size, or grow it in its parent.
func View(c *ui.Context, s *State, bars []Bar, averages []int) ui.Element {
	t := c.Theme()
	n := len(bars)
	if s.show {
		s.show = false
		i := sort.Search(n, func(i int) bool { return !bars[i].Time.Before(s.from) })
		j := sort.Search(n, func(i int) bool { return bars[i].Time.After(s.to) })
		s.first, s.count = float64(i), float64(max(1, j-i))
	}
	if s.count == 0 {
		s.count = float64(n)
	}
	s.clamp(n)
	if s.cross > n {
		s.cross = 0
	}

	size, pad, gap := t.FontSize-2, t.Space(1.5), t.Space(2)
	top := 0.0
	for _, b := range bars {
		top = max(top, b.High)
	}
	tw, th := c.MeasureText(0, ui.Span{Text: price(top), Size: size})
	axisW, timeH := tw+2*pad, th+2*pad
	lines := make([][]float64, len(averages))
	for k, p := range averages {
		lines[k] = average(bars, p)
	}
	colors := lineColors(t, len(averages))

	// The lines above the chart read the bar under its crosshair, which
	// the chart learns as it is built: it is built first, at the bottom of
	// a column laid out upwards.
	root := ui.Column(c).Gap(t.Space(2)).Reverse()
	root.Children(func() {
		box := ui.Box(c).Grow(1).AlignSelf(ui.Stretch).MinHeight(t.Space(50)).Focusable().Label("Price chart")
		r := box.LayoutBounds()
		plotW := r.W - axisW
		slot := plotW / float32(s.count)

		px, py, over := box.PointerPosition()
		over = over && n > 0 && px < plotW && py < r.H-timeH
		if over && (!s.over || px != s.px || py != s.py) {
			s.keyed = false
		}
		if !over && s.over {
			s.cross = 0
		}
		s.over, s.px, s.py = over, px, py
		if over && !s.keyed {
			s.cross = 1 + max(0, min(n-1, int(s.first+float64(px/slot))))
		}

		move := 0
		if box.Shortcut(0, ui.KeyLeft) {
			move = -1
		}
		if box.Shortcut(0, ui.KeyRight) {
			move = 1
		}
		if move != 0 && n > 0 {
			i := s.cross - 1 + move
			if s.cross == 0 {
				i = int(math.Ceil(s.first+s.count)) - 1
			}
			i = max(0, min(n-1, i))
			s.cross, s.keyed = i+1, true
			s.first = max(min(s.first, float64(i)), float64(i+1)-s.count)
			s.clamp(n)
			c.Announce(describe(bars[i]))
		}

		if dx, _, ok := box.Dragged(); ok && dx != 0 && n > 0 {
			s.first -= float64(dx / slot)
			s.clamp(n)
		}
		if box.Pressed() {
			box.Cursor(ui.CursorGrabbing)
		} else {
			box.Cursor(ui.CursorCrosshair)
		}
		// Scrolling comes between builds, at the box laid out since, which
		// only a handle reads then.
		h := ui.Local(box, "candles.handle", func() ui.Handle { return ui.Handle{} })
		box.Bind(h)
		box.HandleInput(func(ev ui.InputEvent) bool {
			w := h.Bounds(c).W - axisW
			if ev.Kind != ui.InputScroll || w <= 0 || n == 0 {
				return false
			}
			if math.Abs(float64(ev.DX)) > math.Abs(float64(ev.DY)) {
				s.first += float64(ev.DX/w) * s.count
				s.clamp(n)
				return true
			}
			// Scrolling 300 DIPs zooms by 2, as waveform does.
			s.zoomAt(s.count*math.Exp2(float64(ev.DY)/300), float64(max(0, min(1, ev.X/w))), n)
			return true
		})
		if s.cross > 0 {
			box.Value(describe(bars[s.cross-1]))
		}

		first, count, cross, pointed := s.first, s.count, s.cross-1, over && !s.keyed
		i0, i1 := int(first), min(n, int(math.Ceil(first+count)))
		lo, hi, most := math.Inf(1), math.Inf(-1), 0.0
		for _, b := range bars[i0:i1] {
			lo, hi, most = min(lo, b.Low), max(hi, b.High), max(most, b.Volume)
		}
		margin := (hi - lo) * 0.08
		if margin == 0 {
			margin = max(1, math.Abs(hi)*0.01)
		}
		prices := chart.NewLinear(lo-margin, hi+margin)

		box.Draw(func(p *ui.Painter, r ui.Rect) {
			if n == 0 {
				return
			}
			plot := ui.Rect{X: r.X, Y: r.Y, W: r.W - axisW, H: r.H - timeH}
			volume := ui.Rect{X: plot.X, Y: plot.Y + plot.H*4/5, W: plot.W, H: plot.H / 5}
			pane := ui.Rect{X: plot.X, Y: plot.Y, W: plot.W, H: volume.Y - plot.Y - gap}
			slot := plot.W / float32(count)
			x := func(i int) float32 { return plot.X + (float32(float64(i)-first)+0.5)*slot }
			y := func(v float64) float32 { return chart.Y(pane, prices.At(v)) }
			grid := t.Border.Alpha(0.5)

			for _, tick := range prices.Ticks(max(2, int(pane.H/60))) {
				ty := chart.Y(pane, tick.At)
				p.Line(plot.X, ty, plot.X+plot.W, ty, 1, grid)
				if ty-th/2 >= r.Y {
					p.Text(plot.X+plot.W+pad, ty-th/2, tick.Label, size, t.TextMuted)
				}
			}
			prev, right := -1, float32(math.Inf(-1))
			loc := bars[i0].Time.Location()
			for _, tick := range chart.NewTime(bars[i0].Time, bars[i1-1].Time).Ticks(max(2, int(plot.W/110))) {
				at := time.UnixMilli(int64(tick.Value)).In(loc)
				i := sort.Search(n, func(i int) bool { return !bars[i].Time.Before(at) })
				if i >= n || i == prev {
					continue
				}
				// A round time that is no bar's, as a Sunday, marks the bar
				// after it, labeled with its own date unless the tick is a
				// month's or a year's, which the bar begins too.
				label := tick.Label
				if !bars[i].Time.Equal(at) && at.Day() != 1 {
					label = bars[i].Time.Format("Jan 2")
				}
				tx := x(i)
				w, _ := p.MeasureText(0, ui.Span{Text: label, Size: size})
				if tx < plot.X || tx > plot.X+plot.W || tx-w/2 < max(r.X, right+gap) {
					continue
				}
				p.Line(tx, plot.Y, tx, plot.Y+plot.H, 1, grid)
				p.Text(tx-w/2, plot.Y+plot.H+pad, label, size, t.TextMuted)
				prev, right = i, tx+w/2
			}
			p.Line(plot.X, volume.Y-gap/2, plot.X+plot.W, volume.Y-gap/2, 1, t.Border)
			p.Line(plot.X+plot.W, plot.Y, plot.X+plot.W, plot.Y+plot.H, 1, t.Border)
			p.Line(plot.X, plot.Y+plot.H, plot.X+plot.W, plot.Y+plot.H, 1, t.Border)

			body := max(1, slot*0.7)
			p.Clip(plot, 0, func() {
				for i := i0; i < i1; i++ {
					b := bars[i]
					color := t.Success
					if b.Close < b.Open {
						color = t.Danger
					}
					bx := x(i)
					p.Line(bx, y(b.High), bx, y(b.Low), 1, color)
					y0, y1 := y(max(b.Open, b.Close)), y(min(b.Open, b.Close))
					p.Fill(ui.Rect{X: bx - body/2, Y: y0, W: body, H: max(1, y1-y0)}, color, 0)
					if most > 0 {
						vh := volume.H * float32(b.Volume/most)
						p.Fill(ui.Rect{X: bx - body/2, Y: volume.Y + volume.H - vh, W: body, H: vh}, color.Alpha(0.4), 0)
					}
				}
				j0, j1 := max(0, i0-1), min(n, i1+1)
				xs, ys := make([]float32, j1-j0), make([]float32, j1-j0)
				for k, line := range lines {
					for i := j0; i < j1; i++ {
						xs[i-j0], ys[i-j0] = (x(i)-pane.X)/pane.W, prices.At(line[i])
					}
					p.Clip(pane, 0, func() {
						p.StrokePath(chart.Line(pane, xs, ys, chart.CurveLinear), 1.5, colors[k])
					})
				}
			})

			if cross < 0 {
				return
			}
			tag := func(rect ui.Rect, s string) {
				p.Fill(rect, t.Text, 3)
				p.Text(rect.X+pad, rect.Y+(rect.H-th)/2, s, size, t.Background)
			}
			cx := x(cross)
			dashed(p, cx, plot.Y, cx, plot.Y+plot.H, t.TextMuted)
			at, v := y(bars[cross].Close), bars[cross].Close
			if pointed {
				at, v = r.Y+py, prices.Invert(1-py/pane.H)
			}
			if at >= pane.Y && at <= pane.Y+pane.H {
				dashed(p, plot.X, at, plot.X+plot.W, at, t.TextMuted)
				tag(ui.Rect{X: plot.X + plot.W, Y: at - th/2 - pad/2, W: axisW, H: th + pad}, price(v))
			}
			label := bars[cross].Time.Format(dateLayout)
			w, _ := p.MeasureText(0, ui.Span{Text: label, Size: size})
			w += 2 * pad
			tag(ui.Rect{X: max(plot.X, min(plot.X+plot.W-w, cx-w/2)), Y: plot.Y + plot.H, W: w, H: timeH}, label)
		})

		if n == 0 {
			return
		}
		i := n - 1
		if s.cross > 0 {
			i = s.cross - 1
		}
		if len(averages) > 0 {
			ui.Row(c).Wrap().Gap(t.Space(3)).Children(func() {
				for k, period := range averages {
					v := "–"
					if !math.IsNaN(lines[k][i]) {
						v = price(lines[k][i])
					}
					ui.Row(c.Key(k)).Gap(t.Space(1)).Children(func() {
						ui.Text(c, "MA"+strconv.Itoa(period)).TextColor(colors[k])
						ui.Text(c, v).FontFeatures("tnum")
					})
				}
			})
		}
		b := bars[i]
		before := b.Open
		if i > 0 {
			before = bars[i-1].Close
		}
		change := b.Close - before
		changeColor := t.Success
		if change < 0 {
			changeColor = t.Danger
		}
		ui.Row(c).Wrap().Gap(t.Space(3)).Children(func() {
			ui.Text(c, b.Time.Format(dateLayout)).TextColor(t.TextMuted)
			field := func(label, value string) {
				ui.Row(c).Gap(t.Space(1)).Children(func() {
					ui.Text(c, label).TextColor(t.TextMuted)
					ui.Text(c, value).FontFeatures("tnum")
				})
			}
			field("O", price(b.Open))
			field("H", price(b.High))
			field("L", price(b.Low))
			field("C", price(b.Close))
			ui.Text(c, fmt.Sprintf("%+.2f (%+.2f%%)", change, 100*change/before)).TextColor(changeColor).FontFeatures("tnum")
			field("Vol", shortVolume(b.Volume))
		})
	})
	return root
}

// average returns the mean of the closes of each bar and the period-1
// before it, NaN for the bars with fewer before them.
func average(bars []Bar, period int) []float64 {
	out := make([]float64, len(bars))
	sum := 0.0
	for i, b := range bars {
		sum += b.Close
		if i >= period {
			sum -= bars[i-period].Close
		}
		out[i] = math.NaN()
		if period > 0 && i >= period-1 {
			out[i] = sum / float64(period)
		}
	}
	return out
}

// lineColors returns n colors for the averages' lines, in hues apart from
// the candles' green and red.
func lineColors(t *ui.Theme, n int) []ui.Color {
	l := float32(0.6)
	if t.Dark {
		l = 0.75
	}
	hues := []float32{250, 70, 310, 195}
	out := make([]ui.Color, n)
	for i := range out {
		out[i] = ui.Oklch(l, 0.15, hues[i%len(hues)])
	}
	return out
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

func describe(b Bar) string {
	return fmt.Sprintf("%s: open %s, high %s, low %s, close %s, volume %s", b.Time.Format(dateLayout),
		price(b.Open), price(b.High), price(b.Low), price(b.Close), shortVolume(b.Volume))
}

func price(v float64) string { return strconv.FormatFloat(v, 'f', 2, 64) }

// shortVolume writes v with K, M or B for thousands, millions and
// billions: 1.24M.
func shortVolume(v float64) string {
	for _, u := range []struct {
		at   float64
		unit string
	}{{1e9, "B"}, {1e6, "M"}, {1e3, "K"}} {
		if v >= u.at {
			return strconv.FormatFloat(v/u.at, 'f', 2, 64) + u.unit
		}
	}
	return strconv.FormatFloat(v, 'f', 0, 64)
}
