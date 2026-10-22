// Package streamgraph shows how many series change over time as a
// streamgraph, as Byron and Wattenberg drew the music people listened to:
// a smooth layer for each, as thick at each time as its value, stacked
// around a baseline that flows with them. The baseline wiggles, as
// theirs, to keep the layers as level as it can; or is a silhouette, the
// stack centered on a straight line; or is zero, the stack piled up from
// the bottom. The layers go inside out, those that rise first in the
// middle and the later ones outside them, or in the order given. Each
// layer's name is written inside it, at its widest, where it is thick
// enough, and in a legend below where it is not; the times are along the
// bottom.
//
// Pointing at a layer lights it and dims the others, and a rule at the
// time under the pointer shows every layer's value then. A click on a
// layer isolates it: the others go, and it lies on a zero baseline, as
// tall as the plot; a click again brings them back. The layers ease from
// one shape to the next, as the baseline and the order change too; while
// the desktop asks for less motion, they change at once.
//
//	streamgraph.View(c, &app.stream, months, genres, streamgraph.Wiggle, streamgraph.InsideOut, "hours").Grow(1)
package streamgraph

import (
	"cmp"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/chart"
)

// Layer is a series of a streamgraph: its name and its value at each
// time. Values below 0, and those missing, take no room.
type Layer struct {
	Name   string
	Values []float64
}

// Baseline is the line a streamgraph's layers stack from.
type Baseline int

const (
	// Wiggle moves the baseline to keep the layers as level as it can, as
	// Byron and Wattenberg's streamgraphs.
	Wiggle Baseline = iota
	// Silhouette centers the stack on a straight line.
	Silhouette
	// Zero piles the layers up from the bottom, as a stacked area chart.
	Zero
)

// Order is how a streamgraph's layers stack.
type Order int

const (
	// InsideOut puts the layers whose values peak first in the middle,
	// and the later ones outside them, on either side by turns, as d3's
	// stackOrderInsideOut.
	InsideOut Order = iota
	// Given stacks the layers in their order, the first at the bottom.
	Given
)

// State is what a streamgraph keeps: the layer a click isolates, by its
// name, so that it holds as the app gives it the layers anew, and the
// shapes on their way to new ones. Keep it in the app's state, one for
// each streamgraph.
type State struct {
	isolated string
	from, to []band
	start    time.Time
}

// ease is how long the layers take to change shape.
const ease = 500 * time.Millisecond

// band is a layer's shape: its bottom and top at each time, in fractions
// up the plot, and how much it shows, from 0 to 1.
type band struct {
	lo, hi []float32
	alpha  float32
}

func (b band) equal(o band) bool {
	return b.alpha == o.alpha && slices.Equal(b.lo, o.lo) && slices.Equal(b.hi, o.hi)
}

// value returns layer k's value at time i, or 0 where it has none.
func value(layers []Layer, k, i int) float64 {
	if i < len(layers[k].Values) {
		if v := layers[k].Values[i]; v > 0 && !math.IsInf(v, 1) {
			return v
		}
	}
	return 0
}

// insideOut returns the layers from the bottom of the stack up, the
// first to peak in the middle and each next on the thinner side.
func insideOut(layers []Layer, n int) []int {
	peak, sum := make([]int, len(layers)), make([]float64, len(layers))
	ids := make([]int, len(layers))
	for k := range layers {
		ids[k] = k
		for i := range n {
			v := value(layers, k, i)
			sum[k] += v
			if v > value(layers, k, peak[k]) {
				peak[k] = i
			}
		}
	}
	slices.SortStableFunc(ids, func(a, b int) int { return cmp.Compare(peak[a], peak[b]) })
	var tops, bottoms []int
	top, bottom := 0.0, 0.0
	for _, k := range ids {
		if top < bottom {
			top += sum[k]
			tops = append(tops, k)
		} else {
			bottom += sum[k]
			bottoms = append(bottoms, k)
		}
	}
	slices.Reverse(bottoms)
	return append(bottoms, tops...)
}

// stack returns the shapes of the layers at n times, stacked in order
// from base, in fractions of the stack's height with a little room
// around it. With only not -1, that layer alone shows, from zero, and the
// others lie flat along it.
func stack(layers []Layer, n int, order []int, base Baseline, only int) []band {
	v := func(k, i int) float64 {
		if only >= 0 && k != only {
			return 0
		}
		return value(layers, k, i)
	}
	if only >= 0 {
		base = Zero
	}
	g := make([]float64, n)
	switch base {
	case Silhouette:
		for i := range g {
			for _, k := range order {
				g[i] -= v(k, i) / 2
			}
		}
	case Wiggle:
		// Byron and Wattenberg's: the baseline moves against the change
		// of the layers below each, weighted by its thickness, so that
		// the layers' slopes are as small as they can be overall.
		for i := 1; i < n; i++ {
			var total, moved, below float64
			for _, k := range order {
				vk, d := v(k, i), v(k, i)-v(k, i-1)
				total += vk
				moved += vk * (below + d/2)
				below += d
			}
			g[i] = g[i-1]
			if total > 0 {
				g[i] -= moved / total
			}
		}
	}
	lo, hi := math.Inf(1), math.Inf(-1)
	for i := range n {
		y := g[i]
		for _, k := range order {
			y += v(k, i)
		}
		lo, hi = min(lo, g[i]), max(hi, y)
	}
	if !(hi > lo) {
		lo, hi = 0, 1
	}
	pad := (hi - lo) * 0.04
	if base != Zero {
		lo -= pad
	}
	hi += pad
	at := func(y float64) float32 { return float32((y - lo) / (hi - lo)) }
	out := make([]band, len(layers))
	for _, k := range order {
		out[k] = band{lo: make([]float32, n), hi: make([]float32, n), alpha: 1}
		if only >= 0 && k != only {
			out[k].alpha = 0
		}
	}
	for i := range n {
		y := g[i]
		for _, k := range order {
			out[k].lo[i] = at(y)
			y += v(k, i)
			out[k].hi[i] = at(y)
		}
	}
	return out
}

// shown returns the shapes to show in this frame, on the way to target,
// and whether they are still on their way.
func (s *State) shown(c *ui.Context, target []band) ([]band, bool) {
	now := c.Now()
	same := len(s.to) == len(target)
	for k := 0; same && k < len(target); k++ {
		same = len(s.to[k].lo) == len(target[k].lo)
	}
	switch {
	case !same || c.Preferences().ReduceMotion:
		s.from, s.to, s.start = target, target, now.Add(-ease)
	case !slices.EqualFunc(s.to, target, band.equal):
		s.from, s.to, s.start = s.at(now), target, now
	}
	if now.Sub(s.start) < ease {
		c.AnimationFrame()
		return s.at(now), true
	}
	return s.to, false
}

// at returns the shapes at now, eased from those the change started at
// to those it goes to.
func (s *State) at(now time.Time) []band {
	f := float32(now.Sub(s.start)) / float32(ease)
	if f >= 1 {
		return s.to
	}
	e := ui.EaseInOut(max(0, f))
	lerp := func(a, b float32) float32 { return a + (b-a)*e }
	out := make([]band, len(s.to))
	for k, to := range s.to {
		from := s.from[k]
		b := band{lo: make([]float32, len(to.lo)), hi: make([]float32, len(to.hi)), alpha: lerp(from.alpha, to.alpha)}
		for i := range to.lo {
			b.lo[i], b.hi[i] = lerp(from.lo[i], to.lo[i]), lerp(from.hi[i], to.hi[i])
		}
		out[k] = b
	}
	return out
}

// along returns the value of ys at f along the plot, joining the points
// at xs, in ascending order, with straight lines.
func along(xs, ys []float32, f float32) float32 {
	i := chart.Nearest(xs, f)
	j := i
	if f > xs[i] && i+1 < len(xs) {
		j = i + 1
	} else if f < xs[i] && i > 0 {
		j = i - 1
	}
	if j == i || xs[j] == xs[i] {
		return ys[i]
	}
	u := (f - xs[i]) / (xs[j] - xs[i])
	return ys[i] + (ys[j]-ys[i])*u
}

// place returns where the middle of each layer's name goes in a plot w by
// h, from its top left: at the time the layer is thickest, as high as
// leaves the most room above and below it, if the name, sizes large, fits
// inside the layer there by pad, or ok false.
func place(xs []float32, bands []band, sizes [][2]float32, w, h, pad float32) (at [][2]float32, ok []bool) {
	at, ok = make([][2]float32, len(bands)), make([]bool, len(bands))
	for k, b := range bands {
		if b.alpha < 1 || len(b.lo) == 0 {
			continue
		}
		widest := 0
		for i := range b.lo {
			if b.hi[i]-b.lo[i] > b.hi[widest]-b.lo[widest] {
				widest = i
			}
		}
		tw, th := sizes[k][0]+2*pad, sizes[k][1]+2*pad
		if (b.hi[widest]-b.lo[widest])*h < th || tw > w {
			continue
		}
		cx := min(max(xs[widest]*w, tw/2), w-tw/2)
		bottom, top := float32(0), float32(1)
		for x := cx - tw/2; x <= cx+tw/2; x += 2 {
			bottom, top = max(bottom, along(xs, b.lo, x/w)), min(top, along(xs, b.hi, x/w))
		}
		if (top-bottom)*h >= th {
			at[k], ok[k] = [2]float32{cx, (1 - (top+bottom)/2) * h}, true
		}
	}
	return at, ok
}

// View creates the streamgraph of layers at times, in ascending order,
// stacked from base in order, its values read out in unit. Give it a
// size, or Grow.
func View(c *ui.Context, s *State, times []time.Time, layers []Layer, base Baseline, order Order, unit string) ui.Element {
	t := c.Theme()
	n := len(times)
	small := t.FontSize - 1
	ids := make([]int, len(layers))
	for k := range ids {
		ids[k] = k
	}
	if order == InsideOut {
		ids = insideOut(layers, n)
	}
	only := slices.IndexFunc(layers, func(l Layer) bool { return l.Name == s.isolated })
	xs := make([]float32, n)
	var x chart.Time
	if n > 0 {
		x = chart.NewTime(times[0], times[n-1])
		for i, tm := range times {
			xs[i] = x.At(tm)
		}
	}
	sizes := make([][2]float32, len(layers))
	for k, l := range layers {
		w, h := c.MeasureText(0, ui.Span{Text: l.Name, Size: small, Weight: 600})
		sizes[k] = [2]float32{w, h}
	}
	colors := palette(len(layers), t.Dark)

	col := ui.Column(c).Gap(t.Space(3))
	ticks := []chart.Tick(nil)
	if n > 1 {
		ticks = x.Ticks(max(2, int(col.LayoutBounds().W/110)))
	}
	label := func(_ chart.Side, tk chart.Tick) {
		ui.Text(c, tk.Label).NoWrap().FontSize(small).TextColor(t.TextMuted).FontFeatures("tnum")
	}
	col.Children(func() {
		var named []bool
		chart.Plot(c, chart.Axes{Bottom: ticks, Label: label}, func(area ui.Element) {
			area.Role(ui.RoleImage).Label("Streamgraph")
			bounds := area.LayoutBounds()
			target := stack(layers, n, ids, base, only)
			hot, i := -1, -1
			px, py, over := area.PointerPosition()
			if over && n > 0 && bounds.W > 0 && bounds.H > 0 {
				fx, fy := px/bounds.W, 1-py/bounds.H
				i = chart.Nearest(xs, fx)
				for k, b := range target {
					if b.alpha == 1 && along(xs, b.lo, fx) <= fy && fy <= along(xs, b.hi, fx) {
						hot = k
					}
				}
			}
			if hot >= 0 || only >= 0 {
				area.Cursor(ui.CursorPointer)
			}
			if area.Clicked() && (hot >= 0 || only >= 0) {
				if only >= 0 {
					only, s.isolated = -1, ""
				} else {
					only, s.isolated = hot, layers[hot].Name
				}
				target = stack(layers, n, ids, base, only)
			}
			bands, moving := s.shown(c, target)
			at, ok := place(xs, target, sizes, bounds.W, bounds.H, t.Space(1))
			named = ok

			area.Draw(func(p *ui.Painter, r ui.Rect) {
				// The names were placed at the size of the last frame.
				if r.W != bounds.W || r.H != bounds.H {
					c.Invalidate()
				}
				for _, k := range ids {
					b := bands[k]
					if b.alpha <= 0 {
						continue
					}
					fill := colors[k]
					if hot >= 0 && hot != k && only < 0 {
						fill = fill.Mix(t.Background, 0.6)
					}
					p.FillPath(chart.Area(r, xs, b.lo, b.hi, chart.CurveMonotone), fill.Alpha(b.alpha))
					p.StrokePath(chart.Line(r, xs, b.hi, chart.CurveMonotone), 1, t.Background.Alpha(b.alpha))
				}
				if !moving {
					for k, fits := range ok {
						if !fits {
							continue
						}
						name := ink(colors[k])
						if hot >= 0 && hot != k && only < 0 {
							name = name.Alpha(0.4)
						}
						p.RichText(r.X+at[k][0]-sizes[k][0]/2, r.Y+at[k][1]-sizes[k][1]/2, 0,
							ui.Span{Text: layers[k].Name, Size: small, Weight: 600, Color: name})
					}
				}
				if i >= 0 {
					rx := chart.X(r, xs[i])
					p.Line(rx, r.Y, rx, r.Y+r.H, 1, t.Text.Alpha(0.55))
				}
			})
			if i < 0 {
				return
			}
			shown := slices.Clone(ids)
			slices.Reverse(shown)
			if only >= 0 {
				shown = []int{only}
			}
			h := ui.Start
			if xs[i] > 0.55 {
				h = ui.End
			}
			chart.Anchor(c, xs[i], 1, h, ui.Start, func() {
				card(c, h, times[i].Format(dateLayout(times)), shown, layers, colors, i, hot, unit)
			})
		}).Grow(1).Shrink(1).AlignSelf(ui.Stretch).Gap(t.Space(2))

		ui.Row(c).Wrap().GapX(t.Space(4)).GapY(t.Space(1.5)).Children(func() {
			for _, k := range ids {
				if !named[k] && (only < 0 || k == only) {
					ui.Row(c.Key(k)).Gap(t.Space(1.5)).AlignItems(ui.Center).Children(func() {
						ui.Box(c).Size(10, 10).Radius(3).Background(colors[k])
						ui.Text(c, layers[k].Name).FontSize(small).TextColor(t.TextMuted).NoWrap()
					})
				}
			}
		})
	})
	return col
}

// card shows the values of the layers shown, top first, at time i, under
// the time's title, beside the rule on side h of it, the layer the
// pointer is at, hot, in bold.
func card(c *ui.Context, h ui.Align, title string, shown []int, layers []Layer, colors []ui.Color, i, hot int, unit string) {
	t := c.Theme()
	margin := [4]float32{t.Space(2), 0, 0, t.Space(2)}
	if h == ui.End {
		margin[1], margin[3] = t.Space(2), 0
	}
	ui.Column(c).PassThrough().Margin(margin[0], margin[1], margin[2], margin[3]).
		Padding(t.Space(2), t.Space(3)).Gap(t.Space(1)).Radius(t.Space(2)).
		Background(t.Background).Border(1, t.Border).Shadow(0, 4, 16, 0, ui.RGBA(0, 0, 0, 0.16)).Children(func() {
		ui.Text(c, title).PassThrough().Bold().NoWrap()
		for _, k := range shown {
			ui.Row(c.Key(k)).PassThrough().Gap(t.Space(4)).AlignItems(ui.Center).Children(func() {
				ui.Row(c).PassThrough().Gap(t.Space(1.5)).AlignItems(ui.Center).Grow(1).Children(func() {
					ui.Box(c).PassThrough().Size(8, 8).Radius(2).Background(colors[k])
					name := ui.Text(c, layers[k].Name).PassThrough().NoWrap().FontSize(t.FontSize - 1)
					if k != hot {
						name.TextColor(t.TextMuted)
					}
				})
				v := ui.Text(c, amount(value(layers, k, i), unit)).PassThrough().NoWrap().
					FontSize(t.FontSize - 1).FontFeatures("tnum")
				if k == hot {
					v.Bold()
				}
			})
		}
	})
}

// dateLayout returns how to write one of times: the year alone for
// yearly times, the month and year for monthly ones, else the day.
func dateLayout(times []time.Time) string {
	if len(times) < 2 {
		return "Jan 2, 2006"
	}
	step := times[len(times)-1].Sub(times[0]) / time.Duration(len(times)-1)
	switch {
	case step >= 300*24*time.Hour:
		return "2006"
	case step >= 25*24*time.Hour:
		return "Jan 2006"
	}
	return "Jan 2, 2006"
}

// palette returns the colors of n layers: as many hues evenly around
// the wheel, taken by strides of about three eighths of it so that layers
// near one another differ, every other hue a little darker, and all light
// enough for dark names on them in either theme.
func palette(n int, dark bool) []ui.Color {
	stride := max(1, int(math.Round(float64(n)*0.38)))
	for gcd(stride, n) > 1 {
		stride++
	}
	l := float32(0.8)
	if dark {
		l = 0.74
	}
	out := make([]ui.Color, n)
	for i := range out {
		h := i * stride % n
		out[i] = ui.Oklch(l-float32(h%2)*0.07, 0.12, float32(30+360*h/n))
	}
	return out
}

func gcd(a, b int) int {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

// ink returns the color of a name inside a layer of color col: a dark
// shade of its hue.
func ink(col ui.Color) ui.Color {
	return col.Mix(ui.RGB(0, 0, 0), 0.72)
}

// amount writes v with up to one decimal, with commas, and unit: "12,400.5
// hours".
func amount(v float64, unit string) string {
	s := strconv.FormatFloat(math.Round(v*10)/10, 'f', -1, 64)
	whole, frac, _ := strings.Cut(s, ".")
	for i := len(whole) - 3; i > 0; i -= 3 {
		whole = whole[:i] + "," + whole[i:]
	}
	if frac != "" {
		whole += "." + frac
	}
	if unit != "" {
		whole += " " + unit
	}
	return whole
}
