// Package beeswarm shows every sample of groups as a dot along an axis of
// values, the dots of equal or near values pushed apart across it so that
// none hide another, as a swarm of bees: each group a row with its name on
// the left, over an axis of round ticks with its title. Where a row is too
// crowded for its dots, they shrink until they fit. The pointer on a dot
// lights it and shows its value and label.
//
//	beeswarm.View(c, &app.chart, beeswarm.Axis{Title: "Salary", Unit: "k$"}, []beeswarm.Group{
//		{Name: "Engineering", Values: salaries, Labels: names},
//		...
//	}).Grow(1)
//
// The places of the dots are kept until the samples or the width change.
package beeswarm

import (
	"cmp"
	"math"
	"slices"
	"strconv"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/chart"
)

// Group is samples shown in a row: its name, its values, NaN and the
// infinities left out, labels naming them by index, and its color; the
// zero Color is one of a set chosen by its place among the groups.
type Group struct {
	Name   string
	Values []float64
	Labels []string
	Color  ui.Color
}

// Axis is the axis of values: its title, and the unit that follows it in
// parentheses and the values the cards read out.
type Axis struct {
	Title, Unit string
}

// State is what a chart keeps: where its dots are, until the samples or
// the width change. Keep it in the app's state, one for each chart.
type State struct {
	key    []any
	width  float32
	radius float32
	places [][]float32
}

// View creates the chart of groups along axis. Give it a size, or grow it
// in its parent.
func View(c *ui.Context, s *State, axis Axis, groups []Group) ui.Element {
	t := c.Theme()
	lo, hi := math.Inf(1), math.Inf(-1)
	for _, g := range groups {
		for _, v := range g.Values {
			if !math.IsNaN(v) && !math.IsInf(v, 0) {
				lo, hi = min(lo, v), max(hi, v)
			}
		}
	}
	if lo > hi {
		lo, hi = 0, 1
	}
	scale := chart.NewLinear(lo, hi).Nice(6)
	ticks := scale.Ticks(6)
	title := axis.Title
	if title != "" && axis.Unit != "" {
		title += " (" + axis.Unit + ")"
	}
	box := ui.Box(c).Role(ui.RoleImage).Label(cmp.Or(axis.Title, "Beeswarm"))
	b := box.LayoutBounds()
	small := t.FontSize - 2
	nameW := float32(0)
	for _, g := range groups {
		w, _ := c.MeasureText(0, ui.Span{Text: g.Name, Size: t.FontSize - 1, Weight: 600})
		nameW = max(nameW, w)
	}
	plot := ui.Rect{X: nameW + t.Space(4), Y: t.Space(2), W: b.W - nameW - t.Space(6), H: b.H - small*2 - t.Space(7)}
	if title == "" {
		plot.H += small + t.Space(2)
	}
	band := plot.H / float32(max(1, len(groups)))
	s.lay(groups, scale, plot.W, band, t.Space(1.6))
	r := s.radius
	hot, dot := -1, -1
	if x, y, ok := box.PointerPosition(); ok && box.Hovered() {
		best := float64(r + 3)
		for i, g := range groups {
			mid := plot.Y + band*(float32(i)+0.5)
			for j, v := range g.Values {
				if math.IsNaN(float64(s.places[i][j])) {
					continue
				}
				dx, dy := x-plot.X-scale.At(v)*plot.W, y-mid-s.places[i][j]
				if d := math.Hypot(float64(dx), float64(dy)); d < best {
					best, hot, dot = d, i, j
				}
			}
		}
	}
	places := s.places
	box.Draw(func(p *ui.Painter, rc ui.Rect) {
		pl := plot
		pl.X += rc.X
		pl.Y += rc.Y
		for _, tk := range ticks {
			x := pl.X + tk.At*pl.W
			p.Line(x, pl.Y, x, pl.Y+pl.H, 1, t.Border.Alpha(0.5))
			w, _ := p.MeasureText(0, ui.Span{Text: tk.Label, Size: small})
			p.Text(x-w/2, pl.Y+pl.H+t.Space(1.5), tk.Label, small, t.TextMuted)
		}
		if title != "" {
			span := ui.Span{Text: title, Size: t.FontSize - 1, Weight: 600, Color: t.TextMuted}
			w, _ := p.MeasureText(0, span)
			p.RichText(pl.X+(pl.W-w)/2, pl.Y+pl.H+small+t.Space(3.5), 0, span)
		}
		for i, g := range groups {
			mid := pl.Y + band*(float32(i)+0.5)
			name := ui.Span{Text: g.Name, Size: t.FontSize - 1, Weight: 600, Color: t.Text}
			w, h := p.MeasureText(0, name)
			p.RichText(pl.X-t.Space(4)-w, mid-h/2, 0, name)
			if i > 0 {
				p.Line(pl.X, mid-band/2, pl.X+pl.W, mid-band/2, 1, t.Border.Alpha(0.35))
			}
			col := colorOf(groups, i, t.Dark)
			for j, v := range g.Values {
				y := places[i][j]
				if math.IsNaN(float64(y)) {
					continue
				}
				cx, cy := pl.X+scale.At(v)*pl.W, mid+y
				p.Fill(ui.Rect{X: cx - r, Y: cy - r, W: 2 * r, H: 2 * r}, col.Alpha(0.85), r)
			}
		}
		if hot >= 0 {
			cx := pl.X + scale.At(groups[hot].Values[dot])*pl.W
			cy := pl.Y + band*(float32(hot)+0.5) + places[hot][dot]
			ring := r + 2.5
			p.Stroke(ui.Rect{X: cx - ring, Y: cy - ring, W: 2 * ring, H: 2 * ring}, t.Text, ring, 1.5)
		}
	})
	if hot >= 0 {
		g := groups[hot]
		x := plot.X + scale.At(g.Values[dot])*plot.W
		y := plot.Y + band*(float32(hot)+0.5) + places[hot][dot]
		label := g.Name
		if dot < len(g.Labels) {
			label = g.Labels[dot] + " · " + g.Name
		}
		box.Children(func() {
			anchor := ui.Box(c).Absolute().PassThrough().Left(x-r).Top(y-r).Size(2*r, 2*r)
			card(c, anchor, value(g.Values[dot], axis.Unit), label)
		})
	}
	return box
}

// lay places the dots of groups, across rows band high along a width w
// wide, as big as radius or as fit, unless placed already.
func (s *State) lay(groups []Group, scale chart.Linear, w, band, radius float32) {
	key := make([]any, 0, len(groups)*2+1)
	key = append(key, scale.Lo, scale.Hi)
	for _, g := range groups {
		key = append(key, len(g.Values))
		if len(g.Values) > 0 {
			key = append(key, &g.Values[0])
		}
	}
	if s.width == w && slices.Equal(s.key, key) {
		return
	}
	s.key, s.width = key, w
	for r := radius; ; r *= 0.85 {
		s.radius = r
		s.places = make([][]float32, len(groups))
		fits := true
		for i, g := range groups {
			s.places[i] = swarm(g.Values, scale, w, r)
			for _, y := range s.places[i] {
				if !math.IsNaN(float64(y)) && abs(y)+r > band/2 {
					fits = false
				}
			}
		}
		if fits || r < 1.2 {
			return
		}
	}
}

// swarm returns the offset across the row of each of values' dots of
// radius r, along w, from the row's middle: as near it as a dot goes
// without overlapping those placed before, from the smallest value.
// Values left out are NaN.
func swarm(values []float64, scale chart.Linear, w, r float32) []float32 {
	ys := make([]float32, len(values))
	order := make([]int, 0, len(values))
	for i, v := range values {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			ys[i] = float32(math.NaN())
			continue
		}
		order = append(order, i)
	}
	xs := make([]float32, len(values))
	for _, i := range order {
		xs[i] = scale.At(values[i]) * w
	}
	slices.SortFunc(order, func(a, b int) int {
		switch {
		case xs[a] < xs[b]:
			return -1
		case xs[a] > xs[b]:
			return 1
		}
		return 0
	})
	d := 2 * r
	var placed []int
	var blocked [][2]float32
	for _, i := range order {
		// Only dots within a diameter to the left can overlap, the order
		// being by x.
		k := 0
		for k < len(placed) && xs[placed[k]] < xs[i]-d {
			k++
		}
		placed = placed[k:]
		blocked = blocked[:0]
		for _, j := range placed {
			dx := xs[i] - xs[j]
			dy := float32(math.Sqrt(float64(d*d - dx*dx)))
			blocked = append(blocked, [2]float32{ys[j] - dy, ys[j] + dy})
		}
		cands := []float32{0}
		for _, b := range blocked {
			cands = append(cands, b[0], b[1])
		}
		// Of two places as near the middle, dots take turns above and below
		// it, which keeps the swarm even.
		up := len(placed)%2 == 0
		slices.SortFunc(cands, func(a, b float32) int {
			switch {
			case abs(a) < abs(b):
				return -1
			case abs(a) > abs(b):
				return 1
			case a < b == up:
				return -1
			case a > b == up:
				return 1
			}
			return 0
		})
		for _, y := range cands {
			free := true
			for _, b := range blocked {
				if y > b[0]+0.01 && y < b[1]-0.01 {
					free = false
					break
				}
			}
			if free {
				ys[i] = y
				break
			}
		}
		placed = append(placed, i)
	}
	return ys
}

func abs(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}

// value writes v with its unit.
func value(v float64, unit string) string {
	s := strconv.FormatFloat(v, 'f', -1, 64)
	if unit != "" {
		s += " " + unit
	}
	return s
}

// colorOf returns the color of group i.
func colorOf(groups []Group, i int, dark bool) ui.Color {
	if c := groups[i].Color; c != (ui.Color{}) {
		return c
	}
	l := float32(0.6)
	if dark {
		l = 0.72
	}
	return ui.Oklch(l, 0.14, float32((250+i*137)%360))
}

// card shows a dot's value and label by anchor.
func card(c *ui.Context, anchor ui.Element, v, label string) {
	t := c.Theme()
	ui.Overlay(c, func() {
		tip := ui.Column(c).Absolute().PassThrough().Role(ui.RoleTooltip).
			AttachTo(anchor, ui.AnchorTop, ui.AnchorBottom).Margin(0, 0, t.Space(1.5), 0).
			Padding(t.Space(1.5), t.Space(2.5)).Gap(t.Space(0.5)).Radius(t.Space(1.5)).
			Background(t.Background).Border(1, t.Border).Shadow(0, 4, 16, 0, ui.RGBA(0, 0, 0, 0.18))
		tip.Children(func() {
			ui.Text(c, v).PassThrough().Bold().FontFeatures("tnum").SingleLine()
			ui.Text(c, label).PassThrough().FontSize(t.FontSize - 1).TextColor(t.TextMuted).SingleLine()
		})
	})
}
