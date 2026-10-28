// Package polygonlabel outlines the regions of a picture with polygons, as
// the datasets of segmentation models are made: a bar of the classes, each
// with its key, its color and how many polygons it has, over the picture
// with the polygons filled in their classes' colors, each tagged with its
// class. Clicks on the picture lay down a polygon's points, of the class
// chosen, with a line from the last to the pointer; a click on the first
// point, or Enter, closes it, Backspace takes back the last point, and
// Escape drops it.
//
// A click on a polygon chooses it and shows its points, dragging it moves
// it, and a click off it lets it go. Dragging a point moves the point,
// dragging the dot halfway along an edge adds a point there, and a
// double-click on a point removes it. The number keys choose a class, and
// give it to the polygon chosen; Delete removes the polygon chosen and
// Escape lets it go. Changed reports an edit.
//
//	polygonlabel.View(c, &app.labeler, app.photo, app.classes, &app.polygons).Grow(1)
package polygonlabel

import (
	"fmt"
	"math"
	"slices"
	"strconv"

	"github.com/egoist/mygo/ui"
)

// Class is a kind of region: its name and the color of its polygons.
type Class struct {
	Name  string
	Color ui.Color
}

// Polygon is a region of the picture: its points in order, in the
// picture's pixels, and the index of its class.
type Polygon struct {
	Points [][2]float32
	Class  int
}

type mode uint8

const (
	idle mode = iota
	moving
	dragging
)

// State is what a labeler keeps: the class chosen, the polygon chosen, the
// points of the one being drawn, and what a drag is doing. Keep it in the
// app's state, one for each labeler.
type State struct {
	class int
	// chosen is the index of the polygon chosen plus 1, so that the zero
	// State has none.
	chosen  int
	drawing [][2]float32
	mode    mode
	point   int
	from    [2]float32
	was     [][2]float32
	down    bool
}

var digits = []ui.Key{ui.Key1, ui.Key2, ui.Key3, ui.Key4, ui.Key5, ui.Key6, ui.Key7, ui.Key8, ui.Key9}

// View creates the labeler of the polygons *polygons in pic.
func View(c *ui.Context, s *State, pic *ui.Bitmap, classes []Class, polygons *[]Polygon) ui.Element {
	t := c.Theme()
	iw, ih := pic.Size()
	root := ui.Column(c).Gap(t.Space(3))
	changed := false
	if s.chosen > len(*polygons) {
		s.chosen = 0
	}
	root.Children(func() {
		ui.Row(c).Gap(t.Space(2)).Wrap().Role(ui.RoleRadioGroup).Label("Classes").Children(func() {
			for i, cl := range classes {
				n := 0
				for _, pg := range *polygons {
					if pg.Class == i {
						n++
					}
				}
				chip := ui.ButtonBase(c.Key(i)).Gap(t.Space(2)).Padding(t.Space(1), t.Space(2.5)).Radius(t.Space(1.5)).
					Border(1, t.Border).Role(ui.RoleRadio).Checked(i == s.class).Label(cl.Name)
				if i == s.class {
					chip.BorderColor(cl.Color).Background(cl.Color.Alpha(0.12))
				} else if chip.Hovered() {
					chip.Background(t.SurfaceHover)
				}
				if chip.Clicked() {
					s.class = i
				}
				chip.Children(func() {
					if i < len(digits) {
						ui.Text(c, strconv.Itoa(i+1)).FontSize(t.FontSize - 2).FontFeatures("tnum").TextColor(t.TextMuted)
					}
					ui.Box(c).Size(t.Space(2.5), t.Space(2.5)).Radius(t.Space(0.5)).Background(cl.Color)
					ui.Text(c, cl.Name)
					ui.Text(c, strconv.Itoa(n)).FontSize(t.FontSize - 2).FontFeatures("tnum").TextColor(t.TextMuted)
				})
			}
		})

		canvas := ui.Box(c).Grow(1).MinHeight(t.Space(40)).Radius(t.Space(2)).Background(t.Surface).Clip().
			Focusable().FocusRing(false).AutoFocus().Cursor(ui.CursorCrosshair).Role(ui.RoleImage).
			Label(fmt.Sprintf("Picture, %d polygons", len(*polygons)))
		b := canvas.Bounds()
		k := min(b.W/float32(iw), b.H/float32(ih))
		ox, oy := (b.W-float32(iw)*k)/2, (b.H-float32(ih)*k)/2
		lx, ly, over := canvas.PointerPosition()
		px, py := (lx-ox)/k, (ly-oy)/k
		clamp := func(x, y float32) [2]float32 {
			return [2]float32{max(0, min(float32(iw), x)), max(0, min(float32(ih), y))}
		}
		near := 7 / k
		closable := len(s.drawing) >= 3 && dist(s.drawing[0], [2]float32{px, py}) < near
		hover, mid := -1, -1
		if s.chosen > 0 && len(s.drawing) == 0 {
			pts := (*polygons)[s.chosen-1].Points
			hover = nearest(pts, px, py, near)
			if hover < 0 {
				mid = nearest(midpoints(pts), px, py, near)
			}
		}
		closeUp := func() {
			*polygons = append(*polygons, Polygon{Points: s.drawing, Class: s.class})
			s.chosen, s.drawing = len(*polygons), nil
			changed = true
		}
		pressed := canvas.Pressed() && k > 0
		if pressed && !s.down {
			s.from = [2]float32{px, py}
			switch {
			case closable:
				closeUp()
			case len(s.drawing) > 0:
				s.drawing = append(s.drawing, clamp(px, py))
			case hover >= 0:
				s.mode, s.point = dragging, hover
			case mid >= 0:
				pg := &(*polygons)[s.chosen-1]
				pg.Points = slices.Insert(pg.Points, mid+1, midpoints(pg.Points)[mid])
				s.mode, s.point = dragging, mid+1
				changed = true
			default:
				had := s.chosen > 0
				s.chosen = 0
				for i := len(*polygons) - 1; i >= 0; i-- {
					if inside((*polygons)[i].Points, px, py) {
						s.chosen, s.mode = i+1, moving
						s.was = slices.Clone((*polygons)[i].Points)
						break
					}
				}
				if s.chosen == 0 && !had {
					s.drawing = [][2]float32{clamp(px, py)}
				}
			}
		}
		if _, _, dragged := canvas.Dragged(); pressed && dragged && s.chosen > 0 {
			pg := &(*polygons)[s.chosen-1]
			switch s.mode {
			case moving:
				dx, dy := px-s.from[0], py-s.from[1]
				lo, hi := bounds(s.was)
				dx = max(-lo[0], min(float32(iw)-hi[0], dx))
				dy = max(-lo[1], min(float32(ih)-hi[1], dy))
				for i, p := range s.was {
					pg.Points[i] = [2]float32{p[0] + dx, p[1] + dy}
				}
				changed = true
			case dragging:
				pg.Points[s.point] = clamp(px, py)
				changed = true
			}
		}
		if canvas.DoubleClicked() && hover >= 0 {
			pg := &(*polygons)[s.chosen-1]
			if len(pg.Points) > 3 {
				pg.Points = slices.Delete(pg.Points, hover, hover+1)
				changed = true
			}
		}
		if !pressed {
			s.mode = idle
		}
		s.down = pressed

		for i, d := range digits[:min(len(classes), len(digits))] {
			if canvas.Shortcut(0, d) {
				s.class = i
				if s.chosen > 0 {
					(*polygons)[s.chosen-1].Class = i
					changed = true
				}
			}
		}
		erase := canvas.Shortcut(0, ui.KeyDelete) || canvas.Shortcut(0, ui.KeyBackspace)
		switch {
		case len(s.drawing) > 0 && erase:
			s.drawing = s.drawing[:len(s.drawing)-1]
		case len(s.drawing) >= 3 && canvas.Shortcut(0, ui.KeyEnter):
			closeUp()
		case s.chosen > 0 && erase:
			*polygons = slices.Delete(*polygons, s.chosen-1, s.chosen)
			s.chosen = 0
			changed = true
		case canvas.Shortcut(0, ui.KeyEscape):
			s.chosen, s.drawing = 0, nil
		}
		if hover >= 0 || mid >= 0 {
			canvas.Cursor(ui.CursorPointer)
		} else if s.mode == moving {
			canvas.Cursor(ui.CursorMove)
		}

		shown := *polygons
		chosen, drawing, class := s.chosen, slices.Clone(s.drawing), s.class
		mx, my := lx, ly
		canvas.Draw(func(p *ui.Painter, r ui.Rect) {
			k := min(r.W/float32(iw), r.H/float32(ih))
			pr := ui.Rect{X: r.X + (r.W-float32(iw)*k)/2, Y: r.Y + (r.H-float32(ih)*k)/2, W: float32(iw) * k, H: float32(ih) * k}
			p.Image(pic, pr, ui.FillBox)
			at := func(q [2]float32) (float32, float32) { return pr.X + q[0]*k, pr.Y + q[1]*k }
			path := func(pts [][2]float32, closed bool) *ui.Path {
				var path ui.Path
				for i, q := range pts {
					x, y := at(q)
					if i == 0 {
						path.MoveTo(x, y)
					} else {
						path.LineTo(x, y)
					}
				}
				if closed {
					path.Close()
				}
				return &path
			}
			dot := func(q [2]float32, rad float32, fill, edge ui.Color) {
				x, y := at(q)
				var d ui.Path
				d.Circle(x, y, rad)
				p.FillPath(&d, fill)
				p.StrokePath(&d, 1.5, edge)
			}
			color := func(i int) (ui.Color, string) {
				if i >= 0 && i < len(classes) {
					return classes[i].Color, classes[i].Name
				}
				return t.Accent, ""
			}
			for i, pg := range shown {
				if len(pg.Points) < 2 {
					continue
				}
				col, name := color(pg.Class)
				fill := float32(0.22)
				if i+1 == chosen {
					fill = 0.4
				}
				p.FillPath(path(pg.Points, true), col.Alpha(fill))
				p.StrokePath(path(pg.Points, true), 2, col)
				top := pg.Points[0]
				for _, q := range pg.Points {
					if q[1] < top[1] {
						top = q
					}
				}
				x, y := at(top)
				size := t.FontSize - 2
				w, h := p.MeasureText(0, ui.Span{Text: name, Size: size, Weight: 600})
				tag := ui.Rect{X: x - (w+8)/2, Y: y - h - 8, W: w + 8, H: h + 4}
				if tag.Y < pr.Y {
					tag.Y = y + 4
				}
				p.Fill(tag, col, 3)
				p.RichText(tag.X+4, tag.Y+2, 0, ui.Span{Text: name, Size: size, Weight: 600, Color: ink(col)})
			}
			if chosen > 0 && chosen <= len(shown) {
				pts := shown[chosen-1].Points
				for _, q := range midpoints(pts) {
					dot(q, 3, ui.RGBA(255, 255, 255, 0.7), ui.RGBA(0, 0, 0, 0.4))
				}
				for i, q := range pts {
					rad := float32(4.5)
					if i == hover {
						rad = 6
					}
					dot(q, rad, ui.RGB(255, 255, 255), ui.RGBA(0, 0, 0, 0.6))
				}
			}
			if len(drawing) > 0 {
				col, _ := color(class)
				if len(drawing) >= 2 {
					p.FillPath(path(drawing, true), col.Alpha(0.15))
				}
				p.StrokePath(path(drawing, false), 2, col)
				if over {
					var tail ui.Path
					tail.MoveTo(at(drawing[len(drawing)-1]))
					tail.LineTo(r.X+mx, r.Y+my)
					p.StrokePath(tail.Dash(0, 5, 4), 1.5, col)
				}
				for i, q := range drawing {
					rad := float32(4)
					if i == 0 && closable {
						rad = 7
					}
					dot(q, rad, ui.RGB(255, 255, 255), col)
				}
			}
		})
	})
	if changed {
		root.SetChanged()
	}
	return root
}

// nearest returns the index of the point of pts within d of x, y, or -1.
func nearest(pts [][2]float32, x, y, d float32) int {
	for i, q := range pts {
		if dist(q, [2]float32{x, y}) < d {
			return i
		}
	}
	return -1
}

// midpoints returns the middle of each edge of the polygon pts, the edge
// from point i to the next.
func midpoints(pts [][2]float32) [][2]float32 {
	out := make([][2]float32, len(pts))
	for i, a := range pts {
		b := pts[(i+1)%len(pts)]
		out[i] = [2]float32{(a[0] + b[0]) / 2, (a[1] + b[1]) / 2}
	}
	return out
}

// bounds returns the top left and bottom right of pts.
func bounds(pts [][2]float32) (lo, hi [2]float32) {
	lo, hi = pts[0], pts[0]
	for _, q := range pts {
		lo = [2]float32{min(lo[0], q[0]), min(lo[1], q[1])}
		hi = [2]float32{max(hi[0], q[0]), max(hi[1], q[1])}
	}
	return lo, hi
}

// inside reports whether x, y lies inside the polygon pts.
func inside(pts [][2]float32, x, y float32) bool {
	in := false
	for i, j := 0, len(pts)-1; i < len(pts); j, i = i, i+1 {
		a, b := pts[i], pts[j]
		if (a[1] > y) != (b[1] > y) && x < a[0]+(y-a[1])*(b[0]-a[0])/(b[1]-a[1]) {
			in = !in
		}
	}
	return in
}

func dist(a, b [2]float32) float32 {
	dx, dy := a[0]-b[0], a[1]-b[1]
	return float32(math.Hypot(float64(dx), float64(dy)))
}

// ink returns black or white, whichever reads on bg.
func ink(bg ui.Color) ui.Color {
	if 0.299*float32(bg.R)+0.587*float32(bg.G)+0.114*float32(bg.B) > 160 {
		return ui.RGB(0, 0, 0)
	}
	return ui.RGB(255, 255, 255)
}
