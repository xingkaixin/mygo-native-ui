// Package floorplan draws a floor plan to furnish, as home planners do:
// rooms with thick walls, each tinted, named and with its area in square
// meters, doors as gaps in the walls with the arc they swing through,
// a grid every half meter and a meter-long scale bar. The pointer on a
// room writes the length of each of its walls along it.
//
// Furniture is drawn to scale with its name. Dragging a piece moves it
// on a five-centimeter grid; a double click, or R while it has the
// keyboard focus, turns it a quarter, and the arrow keys move it by five
// centimeters, or fifty with Shift. A piece that crosses a wall, leaves
// the rooms or overlaps another is outlined in red. The plan moves the
// app's furniture in place and reports the moves as changes:
//
//	if floorplan.View(c, &app.plan, app.rooms, app.doors, &app.furniture).Grow(1).Changed() {
//		app.save()
//	}
package floorplan

import (
	"fmt"
	"math"

	"github.com/egoist/mygo/ui"
)

// Room is a room: its name and its corners in meters, x to the right and
// y down, in order around it.
type Room struct {
	Name    string
	Corners [][2]float64
}

// Door is a door: where its hinge is, in meters, how wide it is, the
// direction along the wall from the hinge to its latch in degrees, 0 to
// the right and 90 down, and the side it swings to, 1 to the right of
// that direction, -1 to the left.
type Door struct {
	X, Y  float64
	Width float64
	Angle float64
	Swing int
}

// Item is a piece of furniture: its name, its middle in meters, its
// width and depth before turning, and how many quarter turns it is
// turned clockwise.
type Item struct {
	Name string
	X, Y float64
	W, D float64
	Turn int
}

// Size returns its width and depth as turned, along x and y.
func (it Item) Size() (float64, float64) {
	if it.Turn%2 != 0 {
		return it.D, it.W
	}
	return it.W, it.D
}

// State is what a plan keeps: the piece being dragged and where the
// pointer has taken it. Keep it in the app's state, one for each plan.
type State struct {
	drag   int
	dx, dy float64
}

const snap = 0.05

// View creates the plan of rooms and doors with *items, which it moves
// and turns in place, reporting the moves as changes. Give it a size, or
// grow it in its parent.
func View(c *ui.Context, s *State, rooms []Room, doors []Door, items *[]Item) ui.Element {
	t := c.Theme()
	box := ui.Box(c).MinWidth(t.Space(100)).MinHeight(t.Space(80)).Clip().Role(ui.RoleGroup).Label("Floor plan")
	b := box.LayoutBounds()
	lo, hi := [2]float64{math.Inf(1), math.Inf(1)}, [2]float64{math.Inf(-1), math.Inf(-1)}
	for _, r := range rooms {
		for _, p := range r.Corners {
			lo = [2]float64{min(lo[0], p[0]), min(lo[1], p[1])}
			hi = [2]float64{max(hi[0], p[0]), max(hi[1], p[1])}
		}
	}
	if len(rooms) == 0 {
		lo, hi = [2]float64{0, 0}, [2]float64{10, 10}
	}
	pad := t.Space(10)
	scale := min(float64(b.W-2*pad)/(hi[0]-lo[0]), float64(b.H-2*pad)/(hi[1]-lo[1]))
	scale = max(scale, 1)
	ox := float64(b.W)/2 - (lo[0]+hi[0])/2*scale
	oy := float64(b.H)/2 - (lo[1]+hi[1])/2*scale
	toPx := func(x, y float64) (float32, float32) { return float32(ox + x*scale), float32(oy + y*scale) }
	toM := func(x, y float32) (float64, float64) { return (float64(x) - ox) / scale, (float64(y) - oy) / scale }

	its := *items
	moved := false
	box.Children(func() {
		for i := range its {
			it := &its[i]
			w, d := it.Size()
			x, y := toPx(it.X-w/2, it.Y-d/2)
			e := ui.Box(c.Key(i)).Absolute().Left(x).Top(y).Size(float32(w*scale), float32(d*scale)).
				Focusable().FocusRing(false).Role(ui.RoleSlider).Label(it.Name).
				Value(fmt.Sprintf("%.2f m by %.2f m at %.2f, %.2f", w, d, it.X, it.Y))
			if s.drag == i+1 {
				e.Cursor(ui.CursorGrabbing)
			} else {
				e.Cursor(ui.CursorGrab)
			}
			if dx, dy, held := e.Dragged(); held {
				if s.drag != i+1 {
					s.drag, s.dx, s.dy = i+1, it.X, it.Y
				}
				s.dx += float64(dx) / scale
				s.dy += float64(dy) / scale
				nx, ny := round(s.dx), round(s.dy)
				if nx != it.X || ny != it.Y {
					it.X, it.Y, moved = nx, ny, true
				}
			} else if s.drag == i+1 {
				s.drag = 0
			}
			if e.DoubleClicked() || e.Shortcut(0, ui.KeyR) {
				it.Turn, moved = (it.Turn+1)%4, true
			}
			for _, step := range []struct {
				mods ui.Modifiers
				by   float64
			}{{0, snap}, {ui.Shift, 10 * snap}} {
				switch {
				case e.Shortcut(step.mods, ui.KeyLeft):
					it.X, moved = round(it.X-step.by), true
				case e.Shortcut(step.mods, ui.KeyRight):
					it.X, moved = round(it.X+step.by), true
				case e.Shortcut(step.mods, ui.KeyUp):
					it.Y, moved = round(it.Y-step.by), true
				case e.Shortcut(step.mods, ui.KeyDown):
					it.Y, moved = round(it.Y+step.by), true
				}
			}
			bad := clashes(rooms, its, i)
			e.Draw(func(p *ui.Painter, r ui.Rect) {
				fill := t.Background
				edge := t.Text.Alpha(0.55)
				if bad {
					edge = t.Danger
				}
				p.Fill(r, fill, 3)
				p.Fill(r, t.Accent.Alpha(0.1), 3)
				width := float32(1.25)
				if e.FocusVisible() || s.drag == i+1 {
					width = 2
				}
				p.Stroke(r, edge, 3, width)
				sp := ui.Span{Text: it.Name, Size: t.FontSize - 3, Color: t.Text}
				tw, th := p.MeasureText(0, sp)
				if tw <= r.W-4 && th <= r.H {
					p.RichText(r.X+(r.W-tw)/2, r.Y+(r.H-th)/2, 0, sp)
				} else if th <= r.W-4 && tw <= r.H {
					cx, cy := r.X+r.W/2, r.Y+r.H/2
					p.Transform(ui.Rotation(-90), cx, cy, func() { p.RichText(cx-tw/2, cy-th/2, 0, sp) })
				}
			})
		}
	})
	if moved {
		box.SetChanged()
	}
	hotRoom := -1
	if px, py, over := box.PointerPosition(); over && s.drag == 0 {
		mx, my := toM(px, py)
		for i, r := range rooms {
			if inside(r.Corners, mx, my) {
				hotRoom = i
			}
		}
	}
	box.Draw(func(p *ui.Painter, r ui.Rect) {
		if r.W != b.W || r.H != b.H {
			c.Invalidate()
		}
		at := func(x, y float64) [2]float32 { px, py := toPx(x, y); return [2]float32{r.X + px, r.Y + py} }
		for gx := math.Floor(lo[0]*2) / 2; gx <= hi[0]; gx += 0.5 {
			a, z := at(gx, lo[1]), at(gx, hi[1])
			p.Line(a[0], a[1], z[0], z[1], 1, t.Text.Alpha(0.04))
		}
		for gy := math.Floor(lo[1]*2) / 2; gy <= hi[1]; gy += 0.5 {
			a, z := at(lo[0], gy), at(hi[0], gy)
			p.Line(a[0], a[1], z[0], z[1], 1, t.Text.Alpha(0.04))
		}
		for i, rm := range rooms {
			var path ui.Path
			for k, q := range rm.Corners {
				v := at(q[0], q[1])
				if k == 0 {
					path.MoveTo(v[0], v[1])
				} else {
					path.LineTo(v[0], v[1])
				}
			}
			path.Close()
			tint := ui.Oklch(0.9, 0.04, float32(i)*55+40)
			if t.Dark {
				tint = ui.Oklch(0.32, 0.04, float32(i)*55+40)
			}
			p.FillPath(&path, tint)
		}
		wall := float32(max(3, 0.12*scale))
		for _, rm := range rooms {
			ring := make([][2]float32, 0, len(rm.Corners)+2)
			for _, q := range rm.Corners {
				ring = append(ring, at(q[0], q[1]))
			}
			ring = append(ring, ring[0], ring[1])
			p.StrokePolyline(ring, wall, t.Text.Alpha(0.85))
		}
		for _, d := range doors {
			door(p, t, d, at, scale, wall)
		}
		for _, rm := range rooms {
			cx, cy := centroid(rm.Corners)
			v := at(cx, cy)
			name := ui.Span{Text: rm.Name, Size: t.FontSize - 1, Weight: 600, Color: t.Text.Alpha(0.75)}
			area := ui.Span{Text: fmt.Sprintf("%.1f m²", area(rm.Corners)), Size: t.FontSize - 2, Color: t.TextMuted, Features: "tnum"}
			nw, nh := p.MeasureText(0, name)
			aw, _ := p.MeasureText(0, area)
			p.RichText(v[0]-nw/2, v[1]-nh, 0, name)
			p.RichText(v[0]-aw/2, v[1], 0, area)
		}
		bar := float32(scale)
		bx, by := r.X+t.Space(4), r.Y+r.H-t.Space(5)
		p.Fill(ui.Rect{X: bx, Y: by, W: bar, H: 3}, t.Text, 1)
		p.Fill(ui.Rect{X: bx, Y: by - 4, W: 1.5, H: 11}, t.Text, 0)
		p.Fill(ui.Rect{X: bx + bar - 1.5, Y: by - 4, W: 1.5, H: 11}, t.Text, 0)
		p.Text(bx+bar+t.Space(1.5), by-t.Space(1.5), "1 m", t.FontSize-2, t.TextMuted)
	})
	box.DrawOver(func(p *ui.Painter, r ui.Rect) {
		if hotRoom < 0 {
			return
		}
		at := func(x, y float64) [2]float32 { px, py := toPx(x, y); return [2]float32{r.X + px, r.Y + py} }
		lengths(p, t, rooms[hotRoom].Corners, at, float32(max(3, 0.12*scale)))
	})
	return box
}

// door paints d: a gap in the wall, the leaf open at a right angle and
// the arc its latch swings through.
func door(p *ui.Painter, t *ui.Theme, d Door, at func(x, y float64) [2]float32, scale float64, wall float32) {
	a := d.Angle * math.Pi / 180
	ux, uy := math.Cos(a), math.Sin(a)
	sw := float64(d.Swing)
	if sw == 0 {
		sw = 1
	}
	nx, ny := -uy*sw, ux*sw
	hinge := at(d.X, d.Y)
	latch := at(d.X+ux*d.Width, d.Y+uy*d.Width)
	open := at(d.X+nx*d.Width, d.Y+ny*d.Width)
	floor := ui.Oklch(0.9, 0, 0)
	if t.Dark {
		floor = ui.Oklch(0.32, 0, 0)
	}
	p.StrokePolyline([][2]float32{hinge, latch}, wall+2, floor)
	var arc ui.Path
	arc.MoveTo(latch[0], latch[1])
	const steps = 16
	for k := 1; k <= steps; k++ {
		f := float64(k) / steps * math.Pi / 2
		dx := ux*math.Cos(f) + nx*math.Sin(f)
		dy := uy*math.Cos(f) + ny*math.Sin(f)
		v := at(d.X+dx*d.Width, d.Y+dy*d.Width)
		arc.LineTo(v[0], v[1])
	}
	arc.Dash(0, 3, 3)
	p.StrokePath(&arc, 1, t.Text.Alpha(0.5))
	p.StrokePolyline([][2]float32{hinge, open}, 2, t.Text.Alpha(0.85))
}

// lengths paints the length of each wall of corners along its inside.
func lengths(p *ui.Painter, t *ui.Theme, corners [][2]float64, at func(x, y float64) [2]float32, wall float32) {
	cx, cy := centroid(corners)
	middle := at(cx, cy)
	for k, a := range corners {
		z := corners[(k+1)%len(corners)]
		l := math.Hypot(z[0]-a[0], z[1]-a[1])
		pa, pz := at(a[0], a[1]), at(z[0], z[1])
		mx, my := (pa[0]+pz[0])/2, (pa[1]+pz[1])/2
		dx, dy := pz[0]-pa[0], pz[1]-pa[1]
		n := float32(math.Hypot(float64(dx), float64(dy)))
		if n == 0 {
			continue
		}
		nx, ny := -dy/n, dx/n
		if nx*(middle[0]-mx)+ny*(middle[1]-my) < 0 {
			nx, ny = -nx, -ny
		}
		sp := ui.Span{Text: fmt.Sprintf("%.2f m", l), Size: t.FontSize - 3, Weight: 600, Color: t.AccentText, Features: "tnum"}
		w, h := p.MeasureText(0, sp)
		off := wall/2 + t.Space(1) + h/2
		x, y := mx+nx*off, my+ny*off
		p.Fill(ui.Rect{X: x - w/2 - 3, Y: y - h/2 - 1, W: w + 6, H: h + 2}, t.Accent, 3)
		p.RichText(x-w/2, y-h/2, 0, sp)
	}
}

// clashes reports whether item i of its leaves the rooms, crosses a wall
// or overlaps another item.
func clashes(rooms []Room, its []Item, i int) bool {
	it := its[i]
	w, d := it.Size()
	x0, y0, x1, y1 := it.X-w/2, it.Y-d/2, it.X+w/2, it.Y+d/2
	for j, o := range its {
		if j == i {
			continue
		}
		ow, od := o.Size()
		if x0 < o.X+ow/2-1e-9 && o.X-ow/2 < x1-1e-9 && y0 < o.Y+od/2-1e-9 && o.Y-od/2 < y1-1e-9 {
			return true
		}
	}
	for _, r := range rooms {
		if inside(r.Corners, x0+1e-6, y0+1e-6) && inside(r.Corners, x1-1e-6, y0+1e-6) &&
			inside(r.Corners, x0+1e-6, y1-1e-6) && inside(r.Corners, x1-1e-6, y1-1e-6) {
			return false
		}
	}
	return true
}

func round(v float64) float64 { return math.Round(v/snap) * snap }

func inside(poly [][2]float64, x, y float64) bool {
	in := false
	for i, j := 0, len(poly)-1; i < len(poly); j, i = i, i+1 {
		a, b := poly[i], poly[j]
		if (a[1] > y) != (b[1] > y) && x < (b[0]-a[0])*(y-a[1])/(b[1]-a[1])+a[0] {
			in = !in
		}
	}
	return in
}

func area(poly [][2]float64) float64 {
	s := 0.0
	for i, a := range poly {
		b := poly[(i+1)%len(poly)]
		s += a[0]*b[1] - b[0]*a[1]
	}
	return math.Abs(s) / 2
}

func centroid(poly [][2]float64) (float64, float64) {
	var a, cx, cy float64
	for i, p := range poly {
		q := poly[(i+1)%len(poly)]
		cross := p[0]*q[1] - q[0]*p[1]
		a += cross
		cx += (p[0] + q[0]) * cross
		cy += (p[1] + q[1]) * cross
	}
	if a == 0 {
		return poly[0][0], poly[0][1]
	}
	return cx / (3 * a), cy / (3 * a)
}
