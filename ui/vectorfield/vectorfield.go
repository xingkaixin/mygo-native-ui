// Package vectorfield plots a vector field on the plane: a grid of arrows,
// each pointing the field's way and colored by its strength, and specks
// drifting with the field and trailing behind them, so that the flow
// shows at a glance. A click draws the streamline through the point,
// both ways, until it leaves the plot or the flow stops; hovering reads out
// the field at the pointer.
//
//	field := func(x, y float64) (float64, float64) { return -y, x }
//	vectorfield.View(c, &app.plot, field, 2)
package vectorfield

import (
	"fmt"
	"math"
	"math/rand/v2"
	"slices"
	"time"

	"github.com/egoist/mygo/ui"
)

// Field is a vector field: the vector at each point of the plane.
type Field func(x, y float64) (dx, dy float64)

// State is what a plot keeps: its specks and its streamlines. Keep it in
// the app's state, one for each plot.
type State struct {
	// Seeds are the points the streamlines go through, one for each click.
	// Set it to nil to clear them.
	Seeds [][2]float64

	specks []speck
	last   time.Time
}

type speck struct {
	trail [][2]float64
	age   float64
	life  float64
}

const (
	nSpecks = 500
	trail   = 14
	spacing = 30
)

// View creates the plot of f over the square from -span to span across
// the plot's shorter side, centered on the origin.
func View(c *ui.Context, s *State, f Field, span float64) ui.Element {
	t := c.Theme()
	view := ui.Box(c).Clip().Cursor(ui.CursorCrosshair).Role(ui.RoleImage).Label("Vector field")
	b := view.LayoutBounds()
	if b.W <= 0 || b.H <= 0 {
		return view
	}
	half := float64(min(b.W, b.H)) / 2
	toPlane := func(px, py float32) (float64, float64) {
		return (float64(px) - float64(b.W)/2) / half * span, -(float64(py) - float64(b.H)/2) / half * span
	}
	toView := func(x, y float64) (float32, float32) {
		return b.W/2 + float32(x/span*half), b.H/2 - float32(y/span*half)
	}
	sx, sy := float64(b.W)/2/half*span, float64(b.H)/2/half*span
	inside := func(x, y float64) bool { return math.Abs(x) <= sx && math.Abs(y) <= sy }

	cols, rows := int(b.W/spacing), int(b.H/spacing)
	ox, oy := (b.W-float32(cols)*spacing)/2+spacing/2, (b.H-float32(rows)*spacing)/2+spacing/2
	type arrow struct{ x, y, dx, dy, m float64 }
	arrows := make([]arrow, 0, cols*rows)
	var mags []float64
	for j := range rows {
		for i := range cols {
			x, y := toPlane(ox+float32(i)*spacing, oy+float32(j)*spacing)
			dx, dy := f(x, y)
			m := math.Hypot(dx, dy)
			if math.IsNaN(m) || math.IsInf(m, 0) {
				continue
			}
			arrows = append(arrows, arrow{x, y, dx, dy, m})
			mags = append(mags, m)
		}
	}
	// Fields such as Van der Pol's grow without bound toward the edges;
	// measuring against the 90th percentile keeps the middle readable.
	strong := 0.0
	if len(mags) > 0 {
		slices.Sort(mags)
		strong = mags[len(mags)*9/10]
	}

	if view.Clicked() {
		if px, py, ok := view.PointerPosition(); ok {
			x, y := toPlane(px, py)
			s.Seeds = append(s.Seeds, [2]float64{x, y})
		}
	}

	now := c.Now()
	dt := min(0.05, now.Sub(s.last).Seconds())
	if s.last.IsZero() {
		dt = 0
	}
	s.last = now
	if strong > 0 {
		// Specks as strong as most arrows cross a quarter of the plot a
		// second, so that slow fields still flow and fast ones stay
		// readable.
		pace := span * 0.5 / strong
		spawn := func(sp *speck) {
			*sp = speck{trail: [][2]float64{{(rand.Float64()*2 - 1) * sx, (rand.Float64()*2 - 1) * sy}}, life: 1.5 + rand.Float64()*2.5}
		}
		if len(s.specks) != nSpecks {
			s.specks = make([]speck, nSpecks)
			for i := range s.specks {
				spawn(&s.specks[i])
				s.specks[i].age = rand.Float64() * s.specks[i].life
			}
		}
		for i := range s.specks {
			sp := &s.specks[i]
			sp.age += dt
			head := sp.trail[len(sp.trail)-1]
			dx, dy := f(head[0], head[1])
			nx, ny := head[0]+dx*pace*dt, head[1]+dy*pace*dt
			if sp.age > sp.life || !inside(nx, ny) || math.IsNaN(nx+ny) {
				spawn(sp)
				continue
			}
			if dt > 0 {
				sp.trail = append(sp.trail, [2]float64{nx, ny})
				if len(sp.trail) > trail {
					sp.trail = sp.trail[1:]
				}
			}
		}
		c.After(time.Second / 60)
	}

	var lines [][][2]float32
	for _, seed := range s.Seeds {
		lines = append(lines, streamline(f, seed, span, inside, toView))
	}

	view.Draw(func(p *ui.Painter, r ui.Rect) {
		for _, a := range arrows {
			if a.m == 0 {
				continue
			}
			k := min(1, a.m/strong)
			x, y := toView(a.x, a.y)
			x, y = r.X+x, r.Y+y
			l := float32(spacing) * 0.38 * float32(0.35+0.65*k)
			ux, uy := float32(a.dx/a.m), float32(-a.dy/a.m)
			col := ramp(k).Alpha(0.8)
			tx, ty := x+ux*l, y+uy*l
			p.StrokePolyline([][2]float32{{x - ux*l, y - uy*l}, {tx, ty}}, 1.3, col)
			hs := float32(4)
			p.FillPath(new(ui.Path).MoveTo(tx+ux*hs*0.6, ty+uy*hs*0.6).
				LineTo(tx-ux*hs-uy*hs*0.6, ty-uy*hs+ux*hs*0.6).
				LineTo(tx-ux*hs+uy*hs*0.6, ty-uy*hs-ux*hs*0.6).Close(), col)
		}
		ink := t.Text
		for _, sp := range s.specks {
			if len(sp.trail) < 2 {
				continue
			}
			fade := float32(math.Min(1, math.Min(sp.age, sp.life-sp.age)*2))
			pts := make([][2]float32, len(sp.trail))
			for i, q := range sp.trail {
				x, y := toView(q[0], q[1])
				pts[i] = [2]float32{r.X + x, r.Y + y}
			}
			p.StrokePolyline(pts, 1.4, ink.Alpha(0.55*fade))
		}
		for i, l := range lines {
			pts := make([][2]float32, len(l))
			for k, q := range l {
				pts[k] = [2]float32{r.X + q[0], r.Y + q[1]}
			}
			if len(pts) > 1 {
				p.StrokePolyline(pts, 2, t.Accent)
			}
			x, y := toView(s.Seeds[i][0], s.Seeds[i][1])
			var dot ui.Path
			p.FillPath(dot.Circle(r.X+x, r.Y+y, 4), t.Accent)
		}
		if px, py, ok := view.PointerPosition(); ok {
			x, y := toPlane(px, py)
			dx, dy := f(x, y)
			read := fmt.Sprintf("(%.2f, %.2f)  →  (%.2f, %.2f)   |F| %.2f", x, y, dx, dy, math.Hypot(dx, dy))
			sp := ui.Span{Text: read, Size: t.FontSize - 1, Color: t.Text, Features: "tnum"}
			w, h := p.MeasureText(0, sp)
			pad := t.Space(2)
			box := ui.Rect{X: r.X + t.Space(3), Y: r.Y + r.H - t.Space(3) - h - 2*pad, W: w + 2*pad, H: h + 2*pad}
			p.Fill(box, t.Background.Alpha(0.9), t.Radius)
			p.RichText(box.X+pad, box.Y+pad, 0, sp)
		}
	})
	return view
}

// streamline traces the curve through seed along f both ways, in steps of
// equal length, and returns it in the plot's points.
func streamline(f Field, seed [2]float64, span float64, inside func(x, y float64) bool, toView func(x, y float64) (float32, float32)) [][2]float32 {
	h := span / 150
	dir := func(x, y, sign float64) (float64, float64, bool) {
		dx, dy := f(x, y)
		m := math.Hypot(dx, dy)
		if m < 1e-9 || math.IsNaN(m) {
			return 0, 0, false
		}
		return sign * dx / m, sign * dy / m, true
	}
	trace := func(sign float64) [][2]float64 {
		var pts [][2]float64
		x, y := seed[0], seed[1]
		for range 1500 {
			k1x, k1y, ok1 := dir(x, y, sign)
			k2x, k2y, ok2 := dir(x+h/2*k1x, y+h/2*k1y, sign)
			k3x, k3y, ok3 := dir(x+h/2*k2x, y+h/2*k2y, sign)
			k4x, k4y, ok4 := dir(x+h*k3x, y+h*k3y, sign)
			if !ok1 || !ok2 || !ok3 || !ok4 {
				break
			}
			x += h / 6 * (k1x + 2*k2x + 2*k3x + k4x)
			y += h / 6 * (k1y + 2*k2y + 2*k3y + k4y)
			if !inside(x, y) {
				break
			}
			pts = append(pts, [2]float64{x, y})
			// A closed orbit comes back to its seed; stop instead of
			// drawing it over and over.
			if len(pts) > 20 && math.Hypot(x-seed[0], y-seed[1]) < h*0.75 {
				break
			}
		}
		return pts
	}
	back := trace(-1)
	var out [][2]float32
	for i := len(back) - 1; i >= 0; i-- {
		x, y := toView(back[i][0], back[i][1])
		out = append(out, [2]float32{x, y})
	}
	x, y := toView(seed[0], seed[1])
	out = append(out, [2]float32{x, y})
	for _, q := range trace(1) {
		x, y := toView(q[0], q[1])
		out = append(out, [2]float32{x, y})
	}
	return out
}

// ramp colors a strength from 0 to 1, from deep blue through teal to
// yellow.
func ramp(k float64) ui.Color {
	k = max(0, min(1, k))
	return ui.Oklch(float32(0.45+0.45*k), float32(0.12+0.05*k), float32(265-170*k))
}
