// Package windrose draws a wind rose, as weather stations and airfield
// reports show one: for each of sixteen directions, a wedge as long as the
// share of the time the wind blew from it, stacked by speed from the
// calmest in the middle, the faster the deeper. Rings mark the shares,
// the calm share is written in the middle, and a legend names the speeds.
//
//	windrose.View(c, &app.rose, app.observations, []float64{0.5, 2, 4, 6, 8}, "m/s")
//
// The pointer over a wedge shows its direction, its share, and the share
// of its speed. A click on a speed in the legend hides it, and another
// shows it again.
package windrose

import (
	"fmt"
	"math"
	"strconv"

	"github.com/egoist/mygo/ui"
)

// Obs is an observation of the wind: the direction it blows from, in
// degrees clockwise from north, and its speed.
type Obs struct {
	Dir, Speed float64
}

// State is what a rose keeps: the speeds hidden. Keep it in the app's
// state, one for each rose.
type State struct {
	hidden map[int]bool
}

var names = []string{"N", "NNE", "NE", "ENE", "E", "ESE", "SE", "SSE", "S", "SSW", "SW", "WSW", "W", "WNW", "NW", "NNW"}

var palette = []ui.Color{
	ui.Hex("#c6dbef"), ui.Hex("#9ecae1"), ui.Hex("#6baed6"), ui.Hex("#4292c6"),
	ui.Hex("#2171b5"), ui.Hex("#08519c"), ui.Hex("#08306b"),
}

const sectors = 16

// View creates the rose of obs. Speeds below bands[0] are calm, and each
// band of speed runs from its bound to the next, the last without end,
// named in unit.
func View(c *ui.Context, s *State, obs []Obs, bands []float64, unit string) ui.Element {
	t := c.Theme()
	if s.hidden == nil {
		s.hidden = map[int]bool{}
	}
	nb := len(bands)
	share := make([][]float64, sectors)
	for i := range share {
		share[i] = make([]float64, nb)
	}
	calm := 0.0
	for _, o := range obs {
		b := -1
		for i, lo := range bands {
			if o.Speed >= lo {
				b = i
			}
		}
		if b < 0 {
			calm++
			continue
		}
		k := int(math.Floor(math.Mod(o.Dir+360/sectors/2+360, 360) / (360.0 / sectors)))
		share[k%sectors][b]++
	}
	total := float64(len(obs))
	if total > 0 {
		calm /= total
		for k := range share {
			for b := range share[k] {
				share[k][b] /= total
			}
		}
	}
	top := 0.0
	for _, sec := range share {
		sum := 0.0
		for b, v := range sec {
			if !s.hidden[b] {
				sum += v
			}
		}
		top = max(top, sum)
	}
	step := ringStep(top)
	rings := math.Max(1, math.Ceil(top/step))

	color := func(b int) ui.Color { return palette[min(b*len(palette)/max(nb, 1), len(palette)-1)] }
	label := func(b int) string {
		if b == nb-1 {
			return fmt.Sprintf("≥ %g %s", bands[b], unit)
		}
		return fmt.Sprintf("%g–%g %s", bands[b], bands[b+1], unit)
	}

	return ui.Row(c).Gap(t.Space(6)).AlignItems(ui.Center).Children(func() {
		size := t.Space(110)
		rose := ui.Box(c).Size(size, size).Label("Wind rose")
		cx, cy := float64(size)/2, float64(size)/2
		outer := float64(size)/2 - 28
		inner := outer * 0.12
		radius := func(v float64) float64 { return inner + (outer-inner)*v/(rings*step) }

		hk, hb := -1, -1
		if px, py, ok := rose.PointerPosition(); ok {
			dx, dy := float64(px)-cx, float64(py)-cy
			d := math.Hypot(dx, dy)
			a := math.Mod(math.Atan2(dx, -dy)*180/math.Pi+360+360/sectors/2, 360)
			k := int(a / (360.0 / sectors))
			acc := 0.0
			for b, v := range share[k] {
				if s.hidden[b] {
					continue
				}
				if d >= radius(acc) && d < radius(acc+v) {
					hk, hb = k, b
				}
				acc += v
			}
		}
		if hk >= 0 {
			sum := 0.0
			for b, v := range share[hk] {
				if !s.hidden[b] {
					sum += v
				}
			}
			rose.Tooltip(fmt.Sprintf("%s %.1f%%, %s %.1f%%", names[hk], sum*100, label(hb), share[hk][hb]*100))
		}

		rose.Draw(func(p *ui.Painter, r ui.Rect) {
			ox, oy := float64(r.X)+cx, float64(r.Y)+cy
			for i := 1; i <= int(rings); i++ {
				var ring ui.Path
				rad := radius(float64(i) * step)
				ring.Circle(float32(ox), float32(oy), float32(rad))
				p.StrokePath(&ring, 1, t.Border)
			}
			for k := range 8 {
				a := float64(k) * math.Pi / 4
				p.StrokePolyline([][2]float32{
					{float32(ox + inner*math.Sin(a)), float32(oy - inner*math.Cos(a))},
					{float32(ox + outer*math.Sin(a)), float32(oy - outer*math.Cos(a))},
				}, 1, t.Border)
			}
			width := 2 * math.Pi / sectors
			for k, sec := range share {
				// Angles go clockwise from north; the painter's from east.
				mid := float64(k)*width - math.Pi/2
				a0, a1 := mid-width/2+0.02, mid+width/2-0.02
				acc := 0.0
				for b, v := range sec {
					if s.hidden[b] || v == 0 {
						continue
					}
					r0, r1 := radius(acc), radius(acc+v)
					acc += v
					var w ui.Path
					w.MoveTo(float32(ox+r1*math.Cos(a0)), float32(oy+r1*math.Sin(a0)))
					arcTo(&w, ox, oy, r1, a0, a1)
					w.LineTo(float32(ox+r0*math.Cos(a1)), float32(oy+r0*math.Sin(a1)))
					arcTo(&w, ox, oy, r0, a1, a0)
					w.Close()
					fill := color(b)
					if hk >= 0 && (k != hk || b != hb) {
						fill = fill.Alpha(0.55)
					}
					p.FillPath(&w, fill)
				}
			}
			for i := 1; i <= int(rings); i++ {
				rad := radius(float64(i) * step)
				pct := strconv.FormatFloat(math.Round(float64(i)*step*1000)/10, 'f', -1, 64)
				span := ui.Span{Text: pct + "%", Size: t.FontSize - 3, Color: t.TextMuted}
				a := math.Pi / 8 * 3
				p.RichText(float32(ox+rad*math.Sin(a))+2, float32(oy-rad*math.Cos(a)), 0, span)
			}
			for k := 0; k < sectors; k += 2 {
				span := ui.Span{Text: names[k], Size: t.FontSize - 1, Weight: 600, Color: t.Text}
				if k%4 != 0 {
					span.Weight, span.Color, span.Size = 400, t.TextMuted, t.FontSize-2
				}
				w, h := p.MeasureText(0, span)
				a := float64(k) * math.Pi * 2 / sectors
				x, y := ox+(outer+14)*math.Sin(a), oy-(outer+14)*math.Cos(a)
				p.RichText(float32(x)-w/2, float32(y)-h/2, 0, span)
			}
			var hole ui.Path
			hole.Circle(float32(ox), float32(oy), float32(inner))
			p.FillPath(&hole, t.Background)
			p.StrokePath(&hole, 1, t.Border)
			calmText := ui.Span{Text: fmt.Sprintf("%.0f%%", calm*100), Size: t.FontSize - 2, Weight: 600, Color: t.TextMuted}
			w, h := p.MeasureText(0, calmText)
			p.RichText(float32(ox)-w/2, float32(oy)-h/2, 0, calmText)
		})
		ui.Column(c).Gap(t.Space(1.5)).Children(func() {
			ui.Text(c, "Speed").Bold().FontSize(t.FontSize - 1)
			for b := nb - 1; b >= 0; b-- {
				row := ui.ButtonBase(c.Key(b)).Gap(t.Space(2)).Padding(t.Space(0.5), t.Space(1)).Radius(t.Space(1)).
					Justify(ui.Start).AlignItems(ui.Center).Label(label(b))
				if row.Hovered() {
					row.Background(t.Text.Alpha(0.06))
				}
				if row.Clicked() {
					s.hidden[b] = !s.hidden[b]
				}
				hidden := s.hidden[b]
				row.Children(func() {
					sw := ui.Box(c).Size(12, 12).Radius(2).Background(color(b))
					txt := ui.Text(c, label(b)).FontSize(t.FontSize - 1)
					if hidden {
						sw.Background(ui.Color{}).Border(1, t.Border)
						txt.TextColor(t.TextMuted).Strikethrough()
					}
				})
			}
			ui.Text(c, fmt.Sprintf("Calm %.1f%%", calm*100)).FontSize(t.FontSize-1).TextColor(t.TextMuted).Padding(t.Space(1), t.Space(1), 0)
		})
	})
}

// ringStep returns the share between rings: a round step giving three
// to five rings up to top.
func ringStep(top float64) float64 {
	for _, s := range []float64{0.01, 0.02, 0.025, 0.05, 0.1, 0.2, 0.25} {
		if top/s <= 5 {
			return s
		}
	}
	return 0.5
}

// arcTo adds the arc of radius r around cx, cy from angle from to to, in
// radians, as cubic Béziers of a quarter turn at most.
func arcTo(p *ui.Path, cx, cy, r, from, to float64) {
	n := math.Ceil(math.Abs(to-from) / (math.Pi / 2))
	if n == 0 {
		return
	}
	d := (to - from) / n
	k := 4.0 / 3 * math.Tan(d/4)
	for i := range int(n) {
		a, b := from+float64(i)*d, from+float64(i+1)*d
		ca, sa, cb, sb := math.Cos(a), math.Sin(a), math.Cos(b), math.Sin(b)
		p.CubeTo(float32(cx+r*(ca-k*sa)), float32(cy+r*(sa+k*ca)),
			float32(cx+r*(cb+k*sb)), float32(cy+r*(sb-k*cb)),
			float32(cx+r*cb), float32(cy+r*sb))
	}
}
