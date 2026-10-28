// Package marey draws a train graph, the time-distance diagram railways
// plan timetables on and Étienne-Jules Marey drew of the Paris–Lyon line:
// the stations down the side as far apart as they are on the line, the
// hours across the top, and each train a line from stop to stop, steep
// where it runs fast, flat where it waits, the trains of each service in
// its color. Lines crossing between stations are trains passing each
// other. Hovering over a train lights it and shows its times at each
// stop. Scrolling pans through the day, and a pinch or Cmd with the wheel
// zooms around the pointer.
//
//	marey.View(c, &app.graph, stations, trains).Grow(1)
package marey

import (
	"fmt"
	"math"
	"time"

	"github.com/egoist/mygo/ui"
)

// Station is a stop on the line, At kilometers from its first.
type Station struct {
	Name string
	At   float64
}

// Stop is a train's stop at a station, by index, its times since
// midnight. A train passing through without stopping arrives as it
// departs.
type Stop struct {
	Station        int
	Arrive, Depart time.Duration
}

// Train is a run along the line, of a service such as "Express", which
// sets its color.
type Train struct {
	Name, Service string
	Stops         []Stop
}

// palette are the services' colors, in the order they first appear.
var palette = []ui.Color{
	ui.Oklch(0.55, 0.15, 255), ui.Oklch(0.6, 0.19, 28), ui.Oklch(0.58, 0.13, 150),
	ui.Oklch(0.6, 0.14, 300), ui.Oklch(0.7, 0.14, 75), ui.Oklch(0.55, 0.09, 200),
}

// State is what a graph keeps: the stretch of the day it shows. Keep it
// in the app's state, one for each graph.
type State struct {
	from, span time.Duration
	ready      bool
	gutter     float32
}

// View creates the graph of trains along the stations, which run in
// order down the line.
func View(c *ui.Context, s *State, stations []Station, trains []Train) ui.Element {
	t := c.Theme()
	view := ui.Box(c).Role(ui.RoleImage).Label("Train graph").Clip()
	if len(stations) < 2 || len(trains) == 0 {
		return view
	}
	lo, hi := time.Duration(math.MaxInt64), time.Duration(0)
	for _, tr := range trains {
		for _, st := range tr.Stops {
			lo, hi = min(lo, st.Arrive), max(hi, st.Depart)
		}
	}
	if !s.ready {
		pad := (hi - lo) / 40
		s.from, s.span, s.ready = lo-pad, hi-lo+2*pad, true
	}
	if s.gutter == 0 {
		s.gutter = t.Space(28)
	}
	head, foot := t.Space(14), t.Space(3)
	b := view.LayoutBounds()
	plotW := max(1, b.W-s.gutter-t.Space(4))
	plotH := max(1, b.H-head-foot)
	first, last := stations[0].At, stations[len(stations)-1].At
	xOf := func(d time.Duration) float32 { return s.gutter + float32(d-s.from)/float32(s.span)*plotW }
	yOf := func(st int) float32 { return head + float32((stations[st].At-first)/(last-first))*plotH }
	view.HandleInput(func(in ui.InputEvent) bool {
		switch {
		case in.Kind == ui.InputZoom:
			s.zoom(float64(in.Scale), in.X, plotW)
		case in.Kind == ui.InputScroll && in.Mods&ui.Cmd != 0:
			s.zoom(math.Exp2(-float64(in.DY)/200), in.X, plotW)
		case in.Kind == ui.InputScroll:
			d := in.DX
			if math.Abs(float64(in.DY)) > math.Abs(float64(in.DX)) {
				d = in.DY
			}
			s.from += time.Duration(float64(d) / float64(plotW) * float64(s.span))
		default:
			return false
		}
		return true
	})
	colors := map[string]ui.Color{}
	var services []string
	for _, tr := range trains {
		if _, ok := colors[tr.Service]; !ok {
			colors[tr.Service] = palette[len(services)%len(palette)]
			services = append(services, tr.Service)
		}
	}
	hover := -1
	if px, py, ok := view.PointerPosition(); ok && px > s.gutter {
		best := float32(7)
		for i, tr := range trains {
			pts := path(tr, xOf, yOf)
			for k := 1; k < len(pts); k++ {
				if d := segDist(px, py, pts[k-1], pts[k]); d < best {
					best, hover = d, i
				}
			}
		}
	}
	view.Draw(func(p *ui.Painter, r ui.Rect) {
		small := t.FontSize - 2
		gutter := float32(0)
		for _, st := range stations {
			w, _ := p.MeasureText(0, ui.Span{Text: st.Name, Size: small})
			gutter = max(gutter, w)
		}
		s.gutter = gutter + t.Space(5)
		at := func(x, y float32) [2]float32 { return [2]float32{r.X + x, r.Y + y} }
		x0, x1 := r.X+s.gutter, r.X+s.gutter+plotW
		for i, st := range stations {
			y := r.Y + yOf(i)
			p.Fill(ui.Rect{X: x0, Y: y - 0.5, W: plotW, H: 1}, t.Border, 0)
			span := ui.Span{Text: st.Name, Size: small, Color: t.TextMuted}
			w, h := p.MeasureText(0, span)
			p.RichText(x0-t.Space(2)-w, y-h/2, 0, span)
		}
		step := tick(s.span, plotW)
		for d := s.from.Truncate(step) + step; d < s.from+s.span; d += step {
			x := r.X + xOf(d)
			col := t.Border.Alpha(0.5)
			if d%time.Hour == 0 {
				col = t.Border
			}
			p.Fill(ui.Rect{X: x - 0.5, Y: r.Y + head, W: 1, H: plotH}, col, 0)
			span := ui.Span{Text: clock(d), Size: small, Color: t.TextMuted, Features: "tnum"}
			w, h := p.MeasureText(0, span)
			p.RichText(x-w/2, r.Y+head-h-t.Space(2), 0, span)
		}
		p.Clip(ui.Rect{X: x0, Y: r.Y, W: x1 - x0, H: r.H}, 0, func() {
			for i, tr := range trains {
				if i == hover {
					continue
				}
				col := colors[tr.Service]
				if hover >= 0 {
					col = col.Alpha(0.25)
				}
				line(p, path(tr, xOf, yOf), at, 1.5, col)
			}
			if hover >= 0 {
				tr := trains[hover]
				pts := path(tr, xOf, yOf)
				line(p, pts, at, 3, colors[tr.Service])
				for _, pt := range pts {
					q := at(pt[0], pt[1])
					p.Fill(ui.Rect{X: q[0] - 3, Y: q[1] - 3, W: 6, H: 6}, colors[tr.Service], 3)
				}
			}
		})
		if hover >= 0 {
			px, py, _ := view.PointerPosition()
			tr := trains[hover]
			lines := []string{tr.Service, tr.Name}
			for _, st := range tr.Stops {
				times := clock(st.Depart)
				if st.Arrive != st.Depart {
					times = clock(st.Arrive) + "–" + clock(st.Depart)
				}
				lines = append(lines, fmt.Sprintf("%s  %s", times, stations[st.Station].Name))
			}
			card(p, t, r, r.X+px, r.Y+py, lines, colors[tr.Service])
		}
		lx := x0
		for _, sv := range services {
			span := ui.Span{Text: sv, Size: small, Color: t.Text}
			w, h := p.MeasureText(0, span)
			y := r.Y + t.Space(1)
			p.Fill(ui.Rect{X: lx, Y: y + h/2 - 1.5, W: t.Space(4), H: 3}, colors[sv], 1.5)
			p.RichText(lx+t.Space(5.5), y, 0, span)
			lx += w + t.Space(10)
		}
	})
	return view
}

// zoom scales the stretch shown by 1/k, keeping the time at x where it is.
func (s *State) zoom(k float64, x float32, plotW float32) {
	at := s.from + time.Duration(float64(x-s.gutter)/float64(plotW)*float64(s.span))
	span := time.Duration(float64(s.span) / k)
	span = max(20*time.Minute, min(36*time.Hour, span))
	s.from = at - time.Duration(float64(at-s.from)*float64(span)/float64(s.span))
	s.span = span
}

// path returns where a train's line bends: its arrival and departure at
// each stop.
func path(tr Train, xOf func(time.Duration) float32, yOf func(int) float32) [][2]float32 {
	pts := make([][2]float32, 0, 2*len(tr.Stops))
	for _, st := range tr.Stops {
		y := yOf(st.Station)
		pts = append(pts, [2]float32{xOf(st.Arrive), y})
		if st.Depart != st.Arrive {
			pts = append(pts, [2]float32{xOf(st.Depart), y})
		}
	}
	return pts
}

func line(p *ui.Painter, pts [][2]float32, at func(x, y float32) [2]float32, w float32, col ui.Color) {
	out := make([][2]float32, len(pts))
	for i, pt := range pts {
		out[i] = at(pt[0], pt[1])
	}
	p.StrokePolyline(out, w, col)
}

// segDist returns how far x, y is from the segment from a to b.
func segDist(x, y float32, a, b [2]float32) float32 {
	dx, dy := b[0]-a[0], b[1]-a[1]
	k := float32(0)
	if l := dx*dx + dy*dy; l > 0 {
		k = max(0, min(1, ((x-a[0])*dx+(y-a[1])*dy)/l))
	}
	return float32(math.Hypot(float64(x-a[0]-k*dx), float64(y-a[1]-k*dy)))
}

// tick returns the step between lines of time across width w, at least
// 64 DIPs apart.
func tick(span time.Duration, w float32) time.Duration {
	for _, m := range []time.Duration{5, 10, 15, 30, 60, 120, 180, 360} {
		if d := m * time.Minute; float32(d)/float32(span)*w >= 64 {
			return d
		}
	}
	return 12 * time.Hour
}

// clock writes a time since midnight as 07:45, past midnight as the next
// day's.
func clock(d time.Duration) string {
	m := int(d/time.Minute) % (24 * 60)
	if m < 0 {
		m += 24 * 60
	}
	return fmt.Sprintf("%02d:%02d", m/60, m%60)
}

// card paints a card of lines beside x, y, the first by a swatch of col.
func card(p *ui.Painter, t *ui.Theme, r ui.Rect, x, y float32, lines []string, col ui.Color) {
	small := t.FontSize - 2
	pad, sw := t.Space(2.5), t.Space(2.5)
	spans := make([]ui.Span, len(lines))
	w, h := float32(0), float32(0)
	for i, l := range lines {
		spans[i] = ui.Span{Text: l, Size: small, Color: t.TextMuted, Features: "tnum"}
		switch i {
		case 0:
			spans[i].Weight, spans[i].Color = 600, t.Text
		case 1:
			spans[i].Size, spans[i].Weight, spans[i].Color = t.FontSize, 600, t.Text
		}
		lw, lh := p.MeasureText(0, spans[i])
		if i == 0 {
			lw += sw + t.Space(1.5)
		}
		w, h = max(w, lw), h+lh
	}
	w, h = w+2*pad, h+2*pad
	cx, cy := x+t.Space(4), y-h/2
	if cx+w > r.X+r.W {
		cx = x - t.Space(4) - w
	}
	cx, cy = max(r.X, cx), max(r.Y, min(cy, r.Y+r.H-h))
	box := ui.Rect{X: cx, Y: cy, W: w, H: h}
	p.Shadow(box, t.Radius+2, 0, 2, 10, 0, ui.RGBA(0, 0, 0, 0.2))
	p.Fill(box, t.Background, t.Radius+2)
	p.Stroke(box, t.Border, t.Radius+2, 1)
	ty := cy + pad
	for i, sp := range spans {
		tx := cx + pad
		_, lh := p.MeasureText(0, sp)
		if i == 0 {
			p.Fill(ui.Rect{X: tx, Y: ty + (lh-sw)/2, W: sw, H: sw}, col, 2)
			tx += sw + t.Space(1.5)
		}
		p.RichText(tx, ty, 0, sp)
		ty += lh
	}
}
