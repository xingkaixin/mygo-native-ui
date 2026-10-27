// Package displays arranges displays, as the Displays settings of macOS
// and Windows do: each display a rectangle as large as its resolution
// beside the others, with its name and size. Dragging one moves it, and
// letting go puts it at the nearest place where it touches another
// without covering any, lined up with that one's top, bottom or middle
// when near them; an outline shows where it will go. The white bar on
// the main display is its menu bar, and dragging it to another display
// makes that one the main display. A click chooses a display.
//
//	displays.View(c, &app.arranger, &app.displays).Grow(1)
//	chosen := app.displays[app.arranger.Chosen]
//
// The arrangement keeps the main display's top left at 0, 0, as macOS
// counts the others from it.
package displays

import (
	"fmt"
	"math"

	"github.com/egoist/mygo/ui"
)

// Display is a display in the arrangement: its name, its size and the
// place of its top left in points, and whether it is the main display.
type Display struct {
	Name          string
	Width, Height int
	X, Y          int
	Main          bool
}

func (d Display) overlaps(o Display) bool {
	return d.X < o.X+o.Width && o.X < d.X+d.Width && d.Y < o.Y+o.Height && o.Y < d.Y+d.Height
}

// State is what an arrangement keeps: the display chosen, and the one
// dragged with how far. Keep it in the app's state, one for each view.
type State struct {
	// Chosen is the display chosen, which a click changes.
	Chosen int
	// dragging is the display pressed, and bar whether its menu bar; dx,
	// dy how far it moved, in points.
	dragging int
	bar      bool
	dx, dy   float64
	held     bool
	// scale and ox, oy are how the arrangement shows, kept from the start
	// of a drag so that the others stay still.
	scale, ox, oy float64
}

// snap returns the place nearest x, y where display i touches another
// and covers none, lined up with that one's top, bottom or middle, or
// its left, right or middle, within near points.
func snap(ds []Display, i, x, y int, near float64) (int, int) {
	d := ds[i]
	best, bx, by := math.Inf(1), d.X, d.Y
	try := func(cx, cy int) {
		moved := d
		moved.X, moved.Y = cx, cy
		for j, o := range ds {
			if j != i && moved.overlaps(o) {
				return
			}
		}
		if dist := math.Hypot(float64(cx-x), float64(cy-y)); dist < best {
			best, bx, by = dist, cx, cy
		}
	}
	align := func(v, lo, hi, size int, starts ...int) int {
		// At least an eighth of the shorter side touches.
		touch := max(1, min(size, hi-lo)/8)
		v = max(lo-size+touch, min(hi-touch, v))
		for _, s := range starts {
			if math.Abs(float64(v-s)) < near {
				return s
			}
		}
		return v
	}
	for j, o := range ds {
		if j == i {
			continue
		}
		cy := align(y, o.Y, o.Y+o.Height, d.Height, o.Y, o.Y+o.Height-d.Height, o.Y+(o.Height-d.Height)/2)
		try(o.X-d.Width, cy)
		try(o.X+o.Width, cy)
		cx := align(x, o.X, o.X+o.Width, d.Width, o.X, o.X+o.Width-d.Width, o.X+(o.Width-d.Width)/2)
		try(cx, o.Y-d.Height)
		try(cx, o.Y+o.Height)
	}
	return bx, by
}

// fit returns the scale and the offset that show ds in w by h, a margin
// around.
func fit(ds []Display, w, h, margin float32) (k, ox, oy float64) {
	x0, y0, x1, y1 := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	for _, d := range ds {
		x0, y0 = min(x0, float64(d.X)), min(y0, float64(d.Y))
		x1, y1 = max(x1, float64(d.X+d.Width)), max(y1, float64(d.Y+d.Height))
	}
	k = min(float64(w-2*margin)/(x1-x0), float64(h-2*margin)/(y1-y0))
	k = max(0.001, k)
	ox = float64(w)/2 - (x0+x1)/2*k
	oy = float64(h)/2 - (y0+y1)/2*k
	return k, ox, oy
}

// View creates the arrangement of *ds. Changed reports a change. Give it
// a size, or grow it in its parent.
func View(c *ui.Context, s *State, ds *[]Display) ui.Element {
	t := c.Theme()
	list := *ds
	canvas := ui.Box(c).Role(ui.RoleGroup).Label("Display arrangement")
	if len(list) == 0 {
		return canvas
	}
	s.Chosen = max(0, min(len(list)-1, s.Chosen))
	b := canvas.LayoutBounds()
	if !s.held {
		s.scale, s.ox, s.oy = fit(list, b.W, b.H, t.Space(10))
	}
	k := s.scale
	rect := func(d Display) ui.Rect {
		return ui.Rect{X: float32(float64(d.X)*k + s.ox), Y: float32(float64(d.Y)*k + s.oy), W: float32(float64(d.Width) * k), H: float32(float64(d.Height) * k)}
	}
	barH := func(r ui.Rect) float32 { return max(4, r.H*0.07) }
	px, py, over := canvas.PointerPosition()
	if canvas.Pressed() && !s.held {
		s.dragging, s.bar, s.dx, s.dy = -1, false, 0, 0
		for i := len(list) - 1; i >= 0; i-- {
			r := rect(list[i])
			if r.Contains(px, py) {
				s.dragging, s.Chosen = i, i
				s.bar = list[i].Main && py < r.Y+barH(r)
				break
			}
		}
	}
	if dx, dy, ok := canvas.Dragged(); ok && s.dragging >= 0 {
		s.dx += float64(dx) / k
		s.dy += float64(dy) / k
	}
	pressed := canvas.Pressed()
	released := s.held && !pressed && s.dragging >= 0
	s.held = pressed
	// target is where the display dragged goes, or the display the menu
	// bar is over.
	target, tx, ty := -1, 0, 0
	if s.dragging >= 0 && (pressed || released) {
		d := list[s.dragging]
		if s.bar {
			for i := range list {
				if rect(list[i]).Contains(px, py) {
					target = i
				}
			}
		} else if s.dx != 0 || s.dy != 0 {
			target = s.dragging
			tx, ty = snap(list, s.dragging, d.X+int(math.Round(s.dx)), d.Y+int(math.Round(s.dy)), 12/k)
		}
	}
	if released {
		switch {
		case s.bar && target >= 0 && !list[target].Main:
			for i := range list {
				list[i].Main = i == target
			}
			normalize(list)
			canvas.SetChanged()
		case !s.bar && target >= 0 && (tx != list[target].X || ty != list[target].Y):
			list[target].X, list[target].Y = tx, ty
			normalize(list)
			canvas.SetChanged()
		}
		s.dragging, s.dx, s.dy = -1, 0, 0
		target = -1
	}
	if over && s.dragging < 0 {
		for i := range list {
			if r := rect(list[i]); r.Contains(px, py) {
				canvas.Cursor(ui.CursorPointer)
				if list[i].Main && py < r.Y+barH(r) {
					canvas.Cursor(ui.CursorGrab)
				}
			}
		}
	}
	if s.dragging >= 0 && pressed {
		canvas.Cursor(ui.CursorGrabbing)
	}
	dragging, bar, chosen := s.dragging, s.bar, s.Chosen
	if !pressed {
		dragging = -1
	}
	offset := ui.Point{X: float32(s.dx * k), Y: float32(s.dy * k)}
	canvas.Draw(func(p *ui.Painter, r ui.Rect) {
		at := func(d Display) ui.Rect {
			rc := rect(d)
			rc.X, rc.Y = rc.X+r.X, rc.Y+r.Y
			return rc
		}
		for i, d := range list {
			if i == dragging && !bar {
				continue
			}
			screen(p, t, at(d), d, i, i == chosen, d.Main && !(bar && dragging >= 0), i == target && bar)
		}
		if dragging >= 0 && !bar {
			d := list[dragging]
			if target >= 0 {
				ghost := d
				ghost.X, ghost.Y = tx, ty
				p.StrokeDashed(at(ghost), t.Accent, 4, 2)
			}
			rc := at(d)
			rc.X, rc.Y = rc.X+offset.X, rc.Y+offset.Y
			p.Shadow(rc, 4, 0, 4, 16, 0, ui.RGB(0, 0, 0).Alpha(0.35))
			screen(p, t, rc, d, dragging, true, d.Main, false)
		}
		if dragging >= 0 && bar {
			src := at(list[dragging])
			w := src.W * 0.9
			strip := ui.Rect{X: r.X + px - w/2, Y: r.Y + py - barH(src)/2, W: w, H: barH(src)}
			p.Shadow(strip, 2, 0, 2, 8, 0, ui.RGB(0, 0, 0).Alpha(0.3))
			p.Fill(strip, ui.RGBA(255, 255, 255, 0.95), 2)
		}
	})
	return canvas
}

// normalize moves the arrangement so that the main display's top left is
// at 0, 0.
func normalize(ds []Display) {
	for _, d := range ds {
		if d.Main {
			x, y := d.X, d.Y
			for i := range ds {
				ds[i].X -= x
				ds[i].Y -= y
			}
			return
		}
	}
}

// screen draws display d, the i-th, in r: a desktop of its own hue with
// its name and size, the menu bar when it shows one, lit when chosen or
// when the menu bar is over it.
func screen(p *ui.Painter, t *ui.Theme, r ui.Rect, d Display, i int, chosen, menu, lit bool) {
	hue := float32(250 + 40*i)
	p.FillGradient(r, ui.LinearGradient{From: ui.Oklch(0.62, 0.12, hue), To: ui.Oklch(0.42, 0.1, hue+30), Angle: 135, Oklab: true}, 4)
	if menu {
		p.Clip(r, 4, func() {
			p.Fill(ui.Rect{X: r.X, Y: r.Y, W: r.W, H: max(4, r.H*0.07)}, ui.RGBA(255, 255, 255, 0.92), 0)
		})
	}
	white := ui.RGB(255, 255, 255)
	size := min(t.FontSize, r.H/6)
	label := fmt.Sprintf("%d × %d", d.Width, d.Height)
	w1, h1 := p.MeasureText(0, ui.Span{Text: d.Name, Size: size, Weight: 600})
	w2, h2 := p.MeasureText(0, ui.Span{Text: label, Size: size * 0.85})
	if w1 < r.W-8 {
		y := r.Y + r.H/2 - (h1+h2)/2
		p.RichText(r.X+r.W/2-w1/2, y, w1+1, ui.Span{Text: d.Name, Size: size, Weight: 600, Color: white})
		p.RichText(r.X+r.W/2-w2/2, y+h1, w2+1, ui.Span{Text: label, Size: size * 0.85, Color: white.Alpha(0.8)})
	}
	switch {
	case lit:
		p.Stroke(r, white, 4, 3)
	case chosen:
		p.Stroke(r, t.Accent, 4, 3)
	default:
		p.Stroke(r, ui.RGB(0, 0, 0).Alpha(0.25), 4, 1)
	}
}
