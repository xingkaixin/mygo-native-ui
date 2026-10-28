// Package elevation draws the elevation profile of a route, as cycling
// and hiking apps show one: the ground along the distance, colored by how
// steep it is, from gray going down to red over 9%, with the climbs
// marked above it, their length and average grade.
//
//	ev := elevation.View(c, &app.profile, app.route)
//	ev.Height(240)
//	if ev.At >= 0 {
//		app.marker = app.pointAt(ev.At) // as on a map beside it
//	}
//
// The pointer over the profile shows the distance, elevation and grade
// there, which At reports. Dragging across it chooses a stretch and shows
// its length, its gain and loss and its grade; a click clears it.
package elevation

import (
	"fmt"
	"math"

	"github.com/egoist/mygo/ui"
)

// Point is a point of a route: its distance from the start and its
// elevation, in meters.
type Point struct {
	Dist, Elev float64
}

// State is what a profile keeps: the stretch chosen. Keep it in the app's
// state, one for each profile.
type State struct {
	from, to float64
	pressed  bool
	moved    bool
	anchor   float32
}

// Events is a profile, and the distance under the pointer, or -1.
type Events struct {
	ui.Element
	At float64
}

var grades = []struct {
	below float64
	color ui.Color
	name  string
}{
	{-1, ui.Hex("#9db4c0"), "Down"},
	{3, ui.Hex("#8cc084"), "0–3%"},
	{6, ui.Hex("#f2c14e"), "3–6%"},
	{9, ui.Hex("#f08a4b"), "6–9%"},
	{math.Inf(1), ui.Hex("#d7263d"), "9%+"},
}

func gradeColor(g float64) ui.Color {
	for _, gr := range grades {
		if g < gr.below {
			return gr.color
		}
	}
	return grades[len(grades)-1].color
}

// climb is a stretch that climbs: its first and last point.
type climb struct{ from, to int }

// climbs finds the stretches gaining 40 m or more at 3% or more, a drop
// of less than 20 m not ending one.
func climbs(pts []Point) []climb {
	var out []climb
	low, peak := 0, 0
	end := func() {
		gain := pts[peak].Elev - pts[low].Elev
		if run := pts[peak].Dist - pts[low].Dist; gain >= 40 && run > 0 && gain/run >= 0.03 {
			out = append(out, climb{low, peak})
		}
	}
	for i, p := range pts {
		switch {
		case p.Elev > pts[peak].Elev:
			peak = i
		case p.Elev < pts[peak].Elev-20:
			end()
			low, peak = i, i
		case p.Elev < pts[low].Elev:
			low, peak = i, i
		}
	}
	end()
	return out
}

// grade returns the grade, in percent, around point i, over 100 m or so.
func grade(pts []Point, i int) float64 {
	a, b := i, i
	for a > 0 && pts[i].Dist-pts[a].Dist < 50 {
		a--
	}
	for b < len(pts)-1 && pts[b].Dist-pts[i].Dist < 50 {
		b++
	}
	if run := pts[b].Dist - pts[a].Dist; run > 0 {
		return (pts[b].Elev - pts[a].Elev) / run * 100
	}
	return 0
}

// elevAt returns the elevation at distance d, between the points around
// it.
func elevAt(pts []Point, d float64) (float64, int) {
	for i := 1; i < len(pts); i++ {
		if pts[i].Dist >= d {
			a, b := pts[i-1], pts[i]
			if b.Dist == a.Dist {
				return b.Elev, i
			}
			return a.Elev + (b.Elev-a.Elev)*(d-a.Dist)/(b.Dist-a.Dist), i
		}
	}
	return pts[len(pts)-1].Elev, len(pts) - 1
}

// View creates the profile of pts, in order of distance. Give it a size,
// or Grow.
func View(c *ui.Context, s *State, pts []Point) Events {
	t := c.Theme()
	view := ui.Box(c).Label("Elevation profile").Cursor(ui.CursorCrosshair)
	ev := Events{Element: view, At: -1}
	if len(pts) < 2 {
		return ev
	}
	lo, hi := math.Inf(1), math.Inf(-1)
	for _, p := range pts {
		lo, hi = min(lo, p.Elev), max(hi, p.Elev)
	}
	step := niceStep((hi - lo) / 4)
	lo, hi = math.Floor(lo/step)*step, hi+(hi-lo)*0.15
	total := pts[len(pts)-1].Dist

	const left, bottom, top = 44, 22, 26
	b := view.Bounds()
	plotW, plotH := float64(b.W)-left-8, float64(b.H)-bottom-top
	xOf := func(d float64) float64 { return left + d/total*plotW }
	yOf := func(e float64) float64 { return top + (hi-e)/(hi-lo)*plotH }
	dOf := func(x float32) float64 { return math.Min(math.Max((float64(x)-left)/plotW*total, 0), total) }

	px, _, over := view.PointerPosition()
	if over {
		ev.At = dOf(px)
	}
	switch {
	case view.Pressed():
		if !s.pressed {
			s.pressed, s.moved, s.anchor = true, false, px
		}
		if math.Abs(float64(px-s.anchor)) > 3 {
			s.moved = true
			s.from, s.to = min(dOf(s.anchor), dOf(px)), max(dOf(s.anchor), dOf(px))
		}
	case s.pressed:
		s.pressed = false
		if !s.moved {
			s.from, s.to = 0, 0
		}
	}

	found := climbs(pts)
	from, to := s.from, s.to
	view.Draw(func(p *ui.Painter, r ui.Rect) {
		ox, oy := float64(r.X), float64(r.Y)
		pt := func(x, y float64) [2]float32 { return [2]float32{float32(ox + x), float32(oy + y)} }
		for e := lo; e <= hi+1e-9; e += step {
			y := float32(oy + yOf(e))
			p.Line(r.X+left, y, r.X+r.W-8, y, 1, t.Border)
			span := ui.Span{Text: fmt.Sprintf("%.0f m", e), Size: t.FontSize - 3, Color: t.TextMuted}
			w, h := p.MeasureText(0, span)
			p.RichText(r.X+left-6-w, y-h/2, 0, span)
		}
		kmStep := niceStep(total / 1000 / 6)
		for km := 0.0; km <= total/1000+1e-9; km += kmStep {
			x := float32(ox + xOf(km*1000))
			span := ui.Span{Text: fmt.Sprintf("%g km", km), Size: t.FontSize - 3, Color: t.TextMuted}
			w, _ := p.MeasureText(0, span)
			p.RichText(x-w/2, float32(oy+top+plotH)+6, 0, span)
		}
		// Runs of one color fill as one shape, as shapes side by side show
		// a seam between them.
		base := float32(oy + yOf(lo))
		for i := 1; i < len(pts); {
			color := gradeColor(grade(pts, i))
			j := i
			for j+1 < len(pts) && gradeColor(grade(pts, j+1)) == color {
				j++
			}
			var q ui.Path
			q.MoveTo(float32(ox+xOf(pts[i-1].Dist)), base)
			for k := i - 1; k <= j; k++ {
				q.LineTo(float32(ox+xOf(pts[k].Dist)), float32(oy+yOf(pts[k].Elev)))
			}
			q.LineTo(float32(ox+xOf(pts[j].Dist)), base).Close()
			p.FillPath(&q, color)
			i = j + 1
		}
		line := make([][2]float32, len(pts))
		for i, q := range pts {
			line[i] = pt(xOf(q.Dist), yOf(q.Elev))
		}
		p.StrokePolyline(line, 1.5, t.Text.Alpha(0.7))

		for n, cl := range found {
			a, b := pts[cl.from], pts[cl.to]
			x0, x1 := ox+xOf(a.Dist), ox+xOf(b.Dist)
			peak := oy + yOf(b.Elev) - 14
			p.StrokePolyline([][2]float32{
				{float32(x0), float32(peak + 5)}, {float32(x0), float32(peak)},
				{float32(x1), float32(peak)}, {float32(x1), float32(peak + 5)},
			}, 1, t.Text.Alpha(0.6))
			label := ui.Span{
				Text: fmt.Sprintf("Climb %d · %.1f km · %.1f%%", n+1, (b.Dist-a.Dist)/1000, (b.Elev-a.Elev)/(b.Dist-a.Dist)*100),
				Size: t.FontSize - 2, Weight: 600, Color: t.Text,
			}
			w, h := p.MeasureText(0, label)
			lx := min(max((x0+x1)/2-float64(w)/2, ox+left), float64(r.X+r.W)-8-float64(w))
			p.RichText(float32(lx), float32(peak)-h-2, 0, label)
		}

		if to > from {
			x0, x1 := float32(ox+xOf(from)), float32(ox+xOf(to))
			p.Fill(ui.Rect{X: x0, Y: float32(oy + top), W: x1 - x0, H: float32(plotH)}, t.Accent.Alpha(0.12), 0)
			p.Line(x0, float32(oy+top), x0, float32(oy+top+plotH), 1, t.Accent)
			p.Line(x1, float32(oy+top), x1, float32(oy+top+plotH), 1, t.Accent)
			gain, loss, rise := stretch(pts, from, to)
			text := fmt.Sprintf("%.2f km · +%.0f m −%.0f m · %.1f%%", (to-from)/1000, gain, loss, rise/(to-from)*100)
			box(p, t, text, float64(x0+x1)/2, oy+top+4, r)
		}

		if over {
			d := dOf(px)
			e, i := elevAt(pts, d)
			x := float32(ox + xOf(d))
			p.Line(x, float32(oy+top), x, float32(oy+top+plotH), 1, t.Text.Alpha(0.5))
			var dot ui.Path
			dot.Circle(x, float32(oy+yOf(e)), 4)
			p.FillPath(&dot, t.Background)
			p.StrokePath(&dot, 2, t.Text)
			y := float64(oy+yOf(e)) - 36
			if y < oy+top {
				y += 50
			}
			box(p, t, fmt.Sprintf("%.2f km · %.0f m · %.1f%%", d/1000, e, grade(pts, i)), float64(x), y, r)
		}

		lx := r.X + r.W - 8
		for k := len(grades) - 1; k >= 0; k-- {
			span := ui.Span{Text: grades[k].name, Size: t.FontSize - 3, Color: t.TextMuted}
			w, h := p.MeasureText(0, span)
			lx -= w
			p.RichText(lx, r.Y, 0, span)
			lx -= 13
			p.Fill(ui.Rect{X: lx, Y: r.Y + (h-9)/2, W: 9, H: 9}, grades[k].color, 2)
			lx -= 12
		}
	})
	return ev
}

// stretch returns the gain and the loss between distances from and to,
// and the difference of their elevations.
func stretch(pts []Point, from, to float64) (gain, loss, rise float64) {
	start, i := elevAt(pts, from)
	end, _ := elevAt(pts, to)
	prev := start
	step := func(e float64) {
		gain, loss = gain+max(e-prev, 0), loss+max(prev-e, 0)
		prev = e
	}
	for ; i < len(pts) && pts[i].Dist < to; i++ {
		step(pts[i].Elev)
	}
	step(end)
	return gain, loss, end - start
}

// box draws text in a small box centered at x, its top at y, kept inside
// r.
func box(p *ui.Painter, t *ui.Theme, text string, x, y float64, r ui.Rect) {
	fill, ink := t.Inverse, t.InverseText
	if fill == (ui.Color{}) {
		fill, ink = t.Text, t.Background
	}
	span := ui.Span{Text: text, Size: t.FontSize - 2, Weight: 600, Color: ink, Features: "tnum"}
	w, h := p.MeasureText(0, span)
	bx := math.Min(math.Max(x-float64(w)/2-8, float64(r.X)), float64(r.X+r.W)-float64(w)-16)
	rect := ui.Rect{X: float32(bx), Y: float32(y), W: w + 16, H: h + 8}
	p.Fill(rect, fill, 6)
	p.RichText(rect.X+8, rect.Y+4, 0, span)
}

// niceStep returns a round step near raw: 1, 2 or 5 times a power of ten.
func niceStep(raw float64) float64 {
	if raw <= 0 {
		return 1
	}
	mag := math.Pow(10, math.Floor(math.Log10(raw)))
	for _, m := range []float64{1, 2, 5, 10} {
		if raw <= m*mag {
			return m * mag
		}
	}
	return 10 * mag
}
