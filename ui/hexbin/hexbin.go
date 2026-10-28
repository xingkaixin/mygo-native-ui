// Package hexbin plots many points as hexagonal bins: the plot is tiled
// with hexagons of the radius given, each shaded by how many points fall
// in it, on a log scale so that sparse bins still show, with a key of the
// shades above. Pointing at a bin outlines it and tells its count.
//
//	hexbin.View(c, &app.bins, points, 12)
package hexbin

import (
	"fmt"
	"math"
	"strconv"

	"github.com/egoist/mygo/ui"
)

// Point is a point to plot.
type Point struct{ X, Y float64 }

// State is what a plot keeps: the bin pointed at. Keep it in the app's
// state, one for each plot.
type State struct {
	hovered [2]int
	over    bool
}

// ticks returns round values to mark from lo to hi, about n of them.
func ticks(lo, hi float64, n int) []float64 {
	raw := (hi - lo) / float64(n)
	step := math.Pow(10, math.Floor(math.Log10(raw)))
	for _, m := range []float64{1, 2, 5, 10} {
		if step*m >= raw {
			step *= m
			break
		}
	}
	var out []float64
	for v := math.Floor(lo/step) * step; v <= math.Ceil(hi/step)*step+step/2; v += step {
		out = append(out, v)
	}
	return out
}

// plot is where a plot of a box w by h puts its points and bins.
type plot struct {
	area           ui.Rect
	xticks, yticks []float64
	bins           map[[2]int]int
	most           int
	radius         float32
}

func layout(t *ui.Theme, points []Point, radius, w, h float32) plot {
	g := plot{area: ui.Rect{X: t.Space(12), Y: t.Space(9), W: w - t.Space(15), H: h - t.Space(17)}, bins: map[[2]int]int{}, radius: radius}
	if len(points) == 0 || g.area.W <= 0 || g.area.H <= 0 {
		return g
	}
	x0, x1, y0, y1 := math.Inf(1), math.Inf(-1), math.Inf(1), math.Inf(-1)
	for _, p := range points {
		x0, x1, y0, y1 = min(x0, p.X), max(x1, p.X), min(y0, p.Y), max(y1, p.Y)
	}
	g.xticks, g.yticks = ticks(x0, x1, 6), ticks(y0, y1, 5)
	for _, p := range points {
		g.bins[g.bin(g.at(p))]++
	}
	for _, n := range g.bins {
		g.most = max(g.most, n)
	}
	return g
}

// at returns where p is drawn, relative to the plot area.
func (g plot) at(p Point) (float32, float32) {
	xs, ys := g.xticks, g.yticks
	fx := (p.X - xs[0]) / (xs[len(xs)-1] - xs[0])
	fy := (p.Y - ys[0]) / (ys[len(ys)-1] - ys[0])
	return float32(fx) * g.area.W, float32(1-fy) * g.area.H
}

// bin returns the axial coordinates of the pointy-top hexagon around x,
// y, rounding as cube coordinates do.
func (g plot) bin(x, y float32) [2]int {
	q := (math.Sqrt(3)/3*float64(x) - float64(y)/3) / float64(g.radius)
	r := 2.0 / 3 * float64(y) / float64(g.radius)
	s := -q - r
	rq, rr, rs := math.Round(q), math.Round(r), math.Round(s)
	dq, dr, ds := math.Abs(rq-q), math.Abs(rr-r), math.Abs(rs-s)
	switch {
	case dq > dr && dq > ds:
		rq = -rr - rs
	case dr > ds:
		rr = -rq - rs
	}
	return [2]int{int(rq), int(rr)}
}

func (g plot) center(b [2]int) (float32, float32) {
	q, r := float32(b[0]), float32(b[1])
	return g.radius * float32(math.Sqrt(3)) * (q + r/2), g.radius * 1.5 * r
}

func hexagon(cx, cy, r float32) *ui.Path {
	var p ui.Path
	for k := range 6 {
		a := math.Pi / 180 * float64(60*k-30)
		x, y := cx+r*float32(math.Cos(a)), cy+r*float32(math.Sin(a))
		if k == 0 {
			p.MoveTo(x, y)
		} else {
			p.LineTo(x, y)
		}
	}
	p.Close()
	return &p
}

func label(v float64) string { return strconv.FormatFloat(v, 'f', -1, 64) }

// View creates the plot of points in bins of radius DIPs. It is as tall
// as Space(110) unless given a height.
func View(c *ui.Context, s *State, points []Point, radius float32) ui.Element {
	t := c.Theme()
	radius = max(radius, 3)
	box := ui.Box(c).Height(t.Space(110)).AlignSelf(ui.Stretch).Label(fmt.Sprintf("%d points in hexagonal bins", len(points)))
	lb := box.LayoutBounds()
	x, y, over := box.PointerPosition()
	g := layout(t, points, radius, lb.W, lb.H)
	s.over = false
	if over && g.area.Contains(x, y) {
		s.hovered = g.bin(x-g.area.X, y-g.area.Y)
		_, s.over = g.bins[s.hovered]
	}
	hovered, hovering := s.hovered, s.over
	box.Draw(func(p *ui.Painter, b ui.Rect) {
		g := layout(t, points, radius, b.W, b.H)
		if g.most == 0 {
			return
		}
		area := g.area
		area.X += b.X
		area.Y += b.Y
		low, high := ui.Hex("#DCE7FF"), ui.Hex("#1A3A99")
		if t.Dark {
			low, high = ui.Hex("#1E2A48"), ui.Hex("#9CC0FF")
		}
		shade := func(n int) ui.Color {
			return low.Mix(high, float32(math.Log1p(float64(n))/math.Log1p(float64(g.most))))
		}
		fs := t.FontSize - 2
		for i, v := range g.xticks {
			x := area.X + area.W*float32(i)/float32(len(g.xticks)-1)
			p.Line(x, area.Y, x, area.Y+area.H, 1, t.Border.Alpha(0.5))
			w, _ := p.MeasureText(0, ui.Span{Text: label(v), Size: fs})
			p.Text(x-w/2, area.Y+area.H+t.Space(2), label(v), fs, t.TextMuted)
		}
		for i, v := range g.yticks {
			y := area.Y + area.H*(1-float32(i)/float32(len(g.yticks)-1))
			p.Line(area.X, y, area.X+area.W, y, 1, t.Border.Alpha(0.5))
			w, h := p.MeasureText(0, ui.Span{Text: label(v), Size: fs})
			p.Text(area.X-w-t.Space(2), y-h/2, label(v), fs, t.TextMuted)
		}
		p.Clip(area, 0, func() {
			for bin, n := range g.bins {
				cx, cy := g.center(bin)
				p.FillPath(hexagon(area.X+cx, area.Y+cy, g.radius-0.6), shade(n))
			}
			if hovering {
				cx, cy := g.center(hovered)
				p.StrokePath(hexagon(area.X+cx, area.Y+cy, g.radius), 2, t.Text)
			}
		})
		if hovering {
			cx, cy := g.center(hovered)
			n := g.bins[hovered]
			tag := fmt.Sprintf("%d point", n)
			if n != 1 {
				tag += "s"
			}
			w, h := p.MeasureText(0, ui.Span{Text: tag, Size: t.FontSize - 1})
			r := ui.Rect{X: area.X + cx - w/2 - t.Space(2), Y: area.Y + cy - g.radius - h - t.Space(3), W: w + t.Space(4), H: h + t.Space(1.5)}
			r.X = min(max(r.X, b.X), b.X+b.W-r.W)
			r.Y = max(r.Y, b.Y)
			p.Shadow(r, t.Radius, 0, 1, 4, 0, ui.RGBA(0, 0, 0, 0.2))
			p.Fill(r, t.Surface, t.Radius)
			p.Text(r.X+t.Space(2), r.Y+t.Space(0.75), tag, t.FontSize-1, t.Text)
		}
		steps := 7
		size := t.Space(2.2)
		most := strconv.Itoa(g.most)
		mw, mh := p.MeasureText(0, ui.Span{Text: most, Size: fs})
		x := b.X + b.W - mw - t.Space(3)
		y := b.Y + t.Space(3)
		p.Text(x+t.Space(1.5), y-mh/2, most, fs, t.TextMuted)
		for k := steps - 1; k >= 0; k-- {
			x -= size * 1.8
			n := int(math.Round(math.Expm1(float64(k) / float64(steps-1) * math.Log1p(float64(g.most)))))
			p.FillPath(hexagon(x+size, y, size), shade(max(n, 1)))
		}
		w, h := p.MeasureText(0, ui.Span{Text: "1", Size: fs})
		p.Text(x-w-t.Space(1), y-h/2, "1", fs, t.TextMuted)
	})
	return box
}
