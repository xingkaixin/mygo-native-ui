// Package sparkline shows a series of values in a word's space, as
// dashboards and tables do beside a number: a line with the area under it
// and a dot on the last value, or bars. The pointer marks the value under
// it and its tooltip says what it is. Card puts one under a number, with
// how it changed.
//
//	sparkline.Line(c, app.latency, ui.Color{}).Size(120, 24)
//	sparkline.Card(c, sparkline.Stat{Label: "Requests", Value: "1,204/s", Change: "+8%", Values: app.requests})
package sparkline

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
)

// Stat is a number of a card: what it is, its value and how it changed,
// written, whether the change is for the worse, and the values before it.
type Stat struct {
	Label  string
	Value  string
	Change string
	Worse  bool
	Values []float64
}

// span returns the lowest and highest of values, apart when they are the
// same.
func span(values []float64) (float64, float64) {
	lo, hi := math.Inf(1), math.Inf(-1)
	for _, v := range values {
		lo, hi = min(lo, v), max(hi, v)
	}
	if lo == hi {
		lo, hi = lo-1, hi+1
	}
	return lo, hi
}

// at returns the index of the value under the pointer of el, n values
// across it, laid out by step from start.
func at(el ui.Element, n int, start, step float32) (int, bool) {
	px, _, ok := el.PointerPosition()
	if !ok || n == 0 {
		return 0, false
	}
	i := int(math.Round(float64((px - start) / step)))
	return max(0, min(n-1, i)), true
}

// Line creates the line of values in color, the theme's accent when it is
// zero. Give it a size.
func Line(c *ui.Context, values []float64, color ui.Color) ui.Element {
	t := c.Theme()
	if color == (ui.Color{}) {
		color = t.Accent
	}
	el := ui.Box(c)
	r := el.LayoutBounds()
	const pad = 3
	step := float32(0)
	if len(values) > 1 {
		step = (r.W - 2*pad) / float32(len(values)-1)
	}
	hot, ok := at(el, len(values), pad, step)
	if ok {
		el.Tooltip(number(values[hot]))
	}
	el.Draw(func(p *ui.Painter, r ui.Rect) {
		if len(values) == 0 {
			return
		}
		lo, hi := span(values)
		step := float32(0)
		if len(values) > 1 {
			step = (r.W - 2*pad) / float32(len(values)-1)
		}
		pt := func(i int) [2]float32 {
			return [2]float32{r.X + pad + float32(i)*step, r.Y + pad + (r.H-2*pad)*float32((hi-values[i])/(hi-lo))}
		}
		pts := make([][2]float32, len(values))
		var area ui.Path
		area.MoveTo(pt(0)[0], r.Y+r.H)
		for i := range values {
			pts[i] = pt(i)
			area.LineTo(pts[i][0], pts[i][1])
		}
		area.LineTo(pts[len(pts)-1][0], r.Y+r.H).Close()
		p.FillPath(&area, color.Alpha(0.12))
		p.StrokePolyline(pts, 1.5, color)
		dot := func(q [2]float32, rad float32) {
			var d ui.Path
			d.Circle(q[0], q[1], rad)
			p.FillPath(&d, color)
		}
		if ok {
			q := pts[hot]
			p.Line(q[0], r.Y, q[0], r.Y+r.H, 1, t.Text.Alpha(0.2))
			dot(q, 3)
		} else {
			dot(pts[len(pts)-1], 2.5)
		}
	})
	return el
}

// Bars creates bars of values in color, the theme's accent when it is
// zero, from 0, or from the lowest when there is one below it. Give it a
// size.
func Bars(c *ui.Context, values []float64, color ui.Color) ui.Element {
	t := c.Theme()
	if color == (ui.Color{}) {
		color = t.Accent
	}
	el := ui.Box(c)
	r := el.LayoutBounds()
	w := float32(0)
	if len(values) > 0 {
		w = r.W / float32(len(values))
	}
	hot, ok := at(el, len(values), w/2, w)
	if ok {
		el.Tooltip(number(values[hot]))
	}
	el.Draw(func(p *ui.Painter, r ui.Rect) {
		if len(values) == 0 {
			return
		}
		lo, hi := span(values)
		lo = min(lo, 0)
		w := r.W / float32(len(values))
		gap := min(2, w*0.25)
		for i, v := range values {
			h := max(1, r.H*float32((v-lo)/(hi-lo)))
			c := color
			if ok && i != hot {
				c = color.Alpha(0.45)
			}
			p.Fill(ui.Rect{X: r.X + float32(i)*w + gap/2, Y: r.Y + r.H - h, W: w - gap, H: h}, c, min(2, (w-gap)/2))
		}
	})
	return el
}

// Card creates a card of stat: its label, its value with the change in
// green, or red when it is for the worse, and the line of its values.
func Card(c *ui.Context, s Stat) ui.Element {
	t := c.Theme()
	return ui.Column(c).Gap(t.Space(2)).Padding(t.Space(4)).Radius(t.Space(2.5)).Background(t.Surface).
		Border(1, t.Border).Children(func() {
		ui.Text(c, s.Label).FontSize(t.FontSize - 1).TextColor(t.TextMuted)
		ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, s.Value).Bold().FontSize(t.FontSize + 8).FontFeatures("tnum")
			if s.Change != "" {
				color := t.Success
				if s.Worse {
					color = t.Danger
				}
				ui.Text(c, s.Change).FontSize(t.FontSize-1).FontFeatures("tnum").TextColor(color).
					Padding(t.Space(0.5), t.Space(1.5)).Radius(t.Space(1)).Background(color.Alpha(0.12))
			}
		})
		Line(c, s.Values, ui.Color{}).Height(t.Space(12)).AlignSelf(ui.Stretch)
	})
}

// number writes v with up to two decimals and commas: 12,400.5.
func number(v float64) string {
	s := strconv.FormatFloat(math.Round(v*100)/100, 'f', -1, 64)
	whole, frac, _ := strings.Cut(s, ".")
	neg := strings.HasPrefix(whole, "-")
	whole = strings.TrimPrefix(whole, "-")
	for i := len(whole) - 3; i > 0; i -= 3 {
		whole = whole[:i] + "," + whole[i:]
	}
	if neg {
		whole = "-" + whole
	}
	if frac != "" {
		return fmt.Sprintf("%s.%s", whole, frac)
	}
	return whole
}
