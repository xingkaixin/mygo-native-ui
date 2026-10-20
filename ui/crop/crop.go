// Package crop lets the user crop a picture, as photo editors do: the
// picture fit in the view, the part kept bright and the rest dimmed, a
// frame around the part with handles at its corners and edges that
// resize it, dragged inside to move it, and its thirds shown while it
// changes. An aspect ratio keeps its shape; the arrow keys move it a
// pixel, ten with Shift.
//
//	crop.View(c, app.picture, &app.area, 16.0/9).Grow(1)
//	... // app.area is the part of the picture kept, in its pixels
package crop

import (
	"math"

	"github.com/egoist/mygo/ui"
)

// edge is the edges of the area a handle moves.
type edge uint8

const (
	left edge = 1 << iota
	top
	right
	bottom
)

// handles are the handles of the frame, by the edges they move, corners
// first, and their cursors.
var handles = []struct {
	edges  edge
	cursor ui.Cursor
	label  string
}{
	{left | top, ui.CursorResizeNWSE, "Top left corner"},
	{right | top, ui.CursorResizeNESW, "Top right corner"},
	{right | bottom, ui.CursorResizeNWSE, "Bottom right corner"},
	{left | bottom, ui.CursorResizeNESW, "Bottom left corner"},
	{top, ui.CursorResizeNS, "Top edge"},
	{right, ui.CursorResizeEW, "Right edge"},
	{bottom, ui.CursorResizeNS, "Bottom edge"},
	{left, ui.CursorResizeEW, "Left edge"},
}

// least is how small the area goes, in the picture's pixels.
const least = 16

// View creates the cropper of pic, *area the part of it kept, in its
// pixels, aspect its width over its height, 0 for any. An empty area
// takes all of the picture, and one of another aspect the largest part
// of it of the aspect, in its middle. Give it a size, or Grow.
func View(c *ui.Context, pic *ui.Bitmap, area *ui.Rect, aspect float32) ui.Element {
	t := c.Theme()
	iw, ih := pic.Size()
	pw, ph := float32(iw), float32(ih)
	bounds := ui.Rect{W: pw, H: ph}
	if area.W <= 0 || area.H <= 0 {
		*area = bounds
	}
	if aspect > 0 && math.Abs(float64(area.W/area.H-aspect)) > 0.001 {
		*area = fit(*area, aspect, bounds)
	}
	box := ui.Box(c).Clip().Focusable().Role(ui.RoleGroup).Label("Crop")
	b := box.LayoutBounds()
	pad := t.Space(6)
	scale := max(1e-3, min((b.W-2*pad)/pw, (b.H-2*pad)/ph))
	ox, oy := (b.W-pw*scale)/2, (b.H-ph*scale)/2
	// view returns where r of the picture is in the box.
	view := func(r ui.Rect) ui.Rect {
		return ui.Rect{X: ox + r.X*scale, Y: oy + r.Y*scale, W: r.W * scale, H: r.H * scale}
	}
	for _, k := range []struct {
		key    ui.Key
		dx, dy float32
	}{{ui.KeyLeft, -1, 0}, {ui.KeyRight, 1, 0}, {ui.KeyUp, 0, -1}, {ui.KeyDown, 0, 1}} {
		if box.Shortcut(0, k.key) {
			*area = move(*area, k.dx, k.dy, bounds)
		}
		if box.Shortcut(ui.Shift, k.key) {
			*area = move(*area, 10*k.dx, 10*k.dy, bounds)
		}
	}
	changing := false
	box.Children(func() {
		v := view(*area)
		mover := ui.Box(c).Absolute().Left(v.X).Top(v.Y).Size(v.W, v.H).Cursor(ui.CursorMove).Label("Move")
		if dx, dy, ok := mover.Dragged(); ok {
			*area = move(*area, dx/scale, dy/scale, bounds)
		}
		changing = changing || mover.Pressed()
		hit := t.Space(5)
		for k, h := range handles {
			x, y := v.X+v.W/2, v.Y+v.H/2
			w, ht := hit, hit
			switch {
			case h.edges&left != 0:
				x = v.X
			case h.edges&right != 0:
				x = v.X + v.W
			default:
				w = max(hit, v.W-2*hit)
			}
			switch {
			case h.edges&top != 0:
				y = v.Y
			case h.edges&bottom != 0:
				y = v.Y + v.H
			default:
				ht = max(hit, v.H-2*hit)
			}
			e := ui.Box(c.Key(k)).Absolute().Left(x-w/2).Top(y-ht/2).Size(w, ht).Cursor(h.cursor).Label(h.label)
			if dx, dy, ok := e.Dragged(); ok {
				*area = resize(*area, h.edges, dx/scale, dy/scale, aspect, bounds)
			}
			changing = changing || e.Pressed()
		}
	})
	shown := view(*area)
	box.Draw(func(p *ui.Painter, r ui.Rect) {
		at := func(q ui.Rect) ui.Rect { return ui.Rect{X: r.X + q.X, Y: r.Y + q.Y, W: q.W, H: q.H} }
		whole := at(view(bounds))
		v := at(shown)
		p.Image(pic, whole, ui.FillBox)
		dim := ui.RGB(0, 0, 0).Alpha(0.55)
		p.Fill(ui.Rect{X: whole.X, Y: whole.Y, W: whole.W, H: v.Y - whole.Y}, dim, 0)
		p.Fill(ui.Rect{X: whole.X, Y: v.Y + v.H, W: whole.W, H: whole.Y + whole.H - v.Y - v.H}, dim, 0)
		p.Fill(ui.Rect{X: whole.X, Y: v.Y, W: v.X - whole.X, H: v.H}, dim, 0)
		p.Fill(ui.Rect{X: v.X + v.W, Y: v.Y, W: whole.X + whole.W - v.X - v.W, H: v.H}, dim, 0)
		white := ui.RGB(255, 255, 255)
		if changing {
			for k := 1; k < 3; k++ {
				f := float32(k) / 3
				p.Line(v.X+v.W*f, v.Y, v.X+v.W*f, v.Y+v.H, 1, white.Alpha(0.5))
				p.Line(v.X, v.Y+v.H*f, v.X+v.W, v.Y+v.H*f, 1, white.Alpha(0.5))
			}
		}
		p.Stroke(v, white.Alpha(0.9), 0, 1)
		arm, w := min(t.Space(5), v.W/3, v.H/3), float32(3)
		for _, cn := range [][4]float32{{v.X, v.Y, 1, 1}, {v.X + v.W, v.Y, -1, 1}, {v.X + v.W, v.Y + v.H, -1, -1}, {v.X, v.Y + v.H, 1, -1}} {
			x, y, sx, sy := cn[0], cn[1], cn[2], cn[3]
			p.Fill(norm(x-sx*w/2, y-sy*w/2, sx*(arm+w/2), sy*w), white, 0)
			p.Fill(norm(x-sx*w/2, y-sy*w/2, sx*w, sy*(arm+w/2)), white, 0)
		}
		mid := min(t.Space(4), v.W/4, v.H/4)
		p.Fill(ui.Rect{X: v.X + v.W/2 - mid/2, Y: v.Y - w/2, W: mid, H: w}, white, 0)
		p.Fill(ui.Rect{X: v.X + v.W/2 - mid/2, Y: v.Y + v.H - w/2, W: mid, H: w}, white, 0)
		p.Fill(ui.Rect{X: v.X - w/2, Y: v.Y + v.H/2 - mid/2, W: w, H: mid}, white, 0)
		p.Fill(ui.Rect{X: v.X + v.W - w/2, Y: v.Y + v.H/2 - mid/2, W: w, H: mid}, white, 0)
	})
	return box
}

// norm returns the rectangle from x, y w wide and h tall, either of which
// may be negative.
func norm(x, y, w, h float32) ui.Rect {
	if w < 0 {
		x, w = x+w, -w
	}
	if h < 0 {
		y, h = y+h, -h
	}
	return ui.Rect{X: x, Y: y, W: w, H: h}
}

// move returns a moved by dx, dy, kept in bounds.
func move(a ui.Rect, dx, dy float32, bounds ui.Rect) ui.Rect {
	a.X = max(bounds.X, min(bounds.X+bounds.W-a.W, a.X+dx))
	a.Y = max(bounds.Y, min(bounds.Y+bounds.H-a.H, a.Y+dy))
	return a
}

// resize returns a with its edges moved by dx, dy, keeping aspect when
// not 0, or a as it is when that would take it out of bounds.
func resize(a ui.Rect, edges edge, dx, dy, aspect float32, bounds ui.Rect) ui.Rect {
	r := a
	if edges&left != 0 {
		r.X, r.W = r.X+dx, r.W-dx
	}
	if edges&right != 0 {
		r.W += dx
	}
	if edges&top != 0 {
		r.Y, r.H = r.Y+dy, r.H-dy
	}
	if edges&bottom != 0 {
		r.H += dy
	}
	if aspect > 0 {
		if edges&(left|right) != 0 {
			h := r.W / aspect
			switch {
			case edges&top != 0:
				r.Y = a.Y + a.H - h
			case edges&bottom == 0:
				r.Y = a.Y + (a.H-h)/2
			}
			r.H = h
		} else {
			w := r.H * aspect
			r.X = a.X + (a.W-w)/2
			r.W = w
		}
	}
	inside := r.X >= bounds.X-0.5 && r.Y >= bounds.Y-0.5 &&
		r.X+r.W <= bounds.X+bounds.W+0.5 && r.Y+r.H <= bounds.Y+bounds.H+0.5
	if r.W < least || r.H < least || !inside {
		return a
	}
	return r
}

// fit returns the largest rectangle of aspect in a, in its middle.
func fit(a ui.Rect, aspect float32, bounds ui.Rect) ui.Rect {
	w, h := a.W, a.W/aspect
	if h > a.H {
		w, h = a.H*aspect, a.H
	}
	cx, cy := a.X+a.W/2, a.Y+a.H/2
	return move(ui.Rect{X: cx - w/2, Y: cy - h/2, W: w, H: h}, 0, 0, bounds)
}
