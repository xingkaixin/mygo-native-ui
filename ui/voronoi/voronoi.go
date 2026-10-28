// Package voronoi draws the Voronoi diagram of a set of sites: each
// site's cell is the part of the plane nearer to it than to any other,
// shaded a color of its own, with the cell pointed at lit. Dragging a
// site moves it and its neighbors' borders with it; pressing where
// there is no site adds one there, and a secondary click on a site
// removes it. The Delaunay triangulation, which joins
// the sites whose cells share a border, can be drawn over the cells.
//
//	voronoi.View(c, &app.diagram, &app.sites)
package voronoi

import (
	"math"

	"github.com/egoist/mygo/ui"
)

// Point is a site, from 0 to 1 across the diagram and down it.
type Point struct{ X, Y float64 }

// State is what a diagram keeps: whether it draws the triangulation, and
// the site being dragged. Keep it in the app's state, one for each
// diagram.
type State struct {
	// Delaunay draws the Delaunay triangulation over the cells.
	Delaunay bool

	dragging int
	size     ui.Rect
	edited   bool
}

// cell is a site's cell: its corners, and for the side from each corner
// to the next the site across it, -1 for the edge of the diagram.
type cell struct {
	x, y   []float32
	across []int
}

// clip keeps the part of c where a*x + b*y <= d, the side beyond it
// across from site.
func (c cell) clip(a, b, d float32, site int) cell {
	var out cell
	n := len(c.x)
	for k := range n {
		x0, y0, x1, y1 := c.x[k], c.y[k], c.x[(k+1)%n], c.y[(k+1)%n]
		in0, in1 := a*x0+b*y0 <= d, a*x1+b*y1 <= d
		cross := func() (float32, float32) {
			f := (d - a*x0 - b*y0) / (a*(x1-x0) + b*(y1-y0))
			return x0 + f*(x1-x0), y0 + f*(y1-y0)
		}
		switch {
		case in0 && in1:
			out.add(x0, y0, c.across[k])
		case in0:
			out.add(x0, y0, c.across[k])
			x, y := cross()
			out.add(x, y, site)
		case in1:
			x, y := cross()
			out.add(x, y, c.across[k])
		}
	}
	return out
}

func (c *cell) add(x, y float32, across int) {
	c.x, c.y, c.across = append(c.x, x), append(c.y, y), append(c.across, across)
}

// cells returns the cell of each site in a box w by h.
func cells(sites []Point, w, h float32) []cell {
	out := make([]cell, len(sites))
	for i, p := range sites {
		c := cell{x: []float32{0, w, w, 0}, y: []float32{0, 0, h, h}, across: []int{-1, -1, -1, -1}}
		px, py := float32(p.X)*w, float32(p.Y)*h
		for j, q := range sites {
			if j == i || len(c.x) == 0 {
				continue
			}
			qx, qy := float32(q.X)*w, float32(q.Y)*h
			if qx == px && qy == py {
				continue
			}
			// Nearer to p than to q: (q-p)·x <= (|q|²-|p|²)/2.
			c = c.clip(qx-px, qy-py, (qx*qx+qy*qy-px*px-py*py)/2, j)
		}
		out[i] = c
	}
	return out
}

func (c cell) path(x, y float32) *ui.Path {
	var p ui.Path
	for k := range c.x {
		if k == 0 {
			p.MoveTo(x+c.x[k], y+c.y[k])
		} else {
			p.LineTo(x+c.x[k], y+c.y[k])
		}
	}
	p.Close()
	return &p
}

var hues = []ui.Color{
	ui.Hex("#3E7BFA"), ui.Hex("#30A46C"), ui.Hex("#F5A524"), ui.Hex("#E5484D"),
	ui.Hex("#8E4EC6"), ui.Hex("#12A594"), ui.Hex("#F07A2B"), ui.Hex("#D6409F"),
}

// nearest returns the site nearest to x, y in a box w by h, and how far
// it is.
func nearest(sites []Point, x, y, w, h float32) (int, float32) {
	best, dist := -1, float32(math.Inf(1))
	for i, p := range sites {
		dx, dy := float32(p.X)*w-x, float32(p.Y)*h-y
		if d := dx*dx + dy*dy; d < dist {
			best, dist = i, d
		}
	}
	return best, float32(math.Sqrt(float64(dist)))
}

// View creates the diagram of *sites. It is as tall as Space(110) unless
// given a height.
func View(c *ui.Context, s *State, sites *[]Point) ui.Element {
	t := c.Theme()
	grab := t.Space(3)
	box := ui.Box(c).Height(t.Space(110)).AlignSelf(ui.Stretch).Radius(t.Radius + 2).Background(t.Surface).Label("Voronoi diagram").Cursor(ui.CursorPointer)
	s.size = box.LayoutBounds()
	changed := s.edited
	s.edited = false
	box.HandleInput(func(ev ui.InputEvent) bool {
		w, h := s.size.W, s.size.H
		if w <= 0 || h <= 0 {
			return false
		}
		at := Point{min(max(float64(ev.X/w), 0), 1), min(max(float64(ev.Y/h), 0), 1)}
		switch ev.Kind {
		case ui.InputPointerDown:
			i, d := nearest(*sites, ev.X, ev.Y, w, h)
			hit := i >= 0 && d <= grab*2
			switch {
			case hit && ev.Button == 1:
				*sites = append((*sites)[:i], (*sites)[i+1:]...)
				s.dragging = -1
			case ev.Button != 0:
				return false
			case hit:
				s.dragging = i
			default:
				*sites = append(*sites, at)
				s.dragging = len(*sites) - 1
			}
			s.edited = true
		case ui.InputPointerMove:
			if ev.Button != 0 || s.dragging < 0 || s.dragging >= len(*sites) {
				return false
			}
			(*sites)[s.dragging] = at
			s.edited = true
		case ui.InputPointerUp:
			s.dragging = -1
		default:
			return false
		}
		return true
	})
	if !box.Pressed() {
		s.dragging = -1
	}
	lit := -1
	if x, y, over := box.PointerPosition(); over {
		lit, _ = nearest(*sites, x, y, s.size.W, s.size.H)
	}
	if s.dragging >= 0 {
		lit = s.dragging
	}
	points, delaunay := *sites, s.Delaunay
	box.Draw(func(p *ui.Painter, b ui.Rect) {
		cs := cells(points, b.W, b.H)
		p.Clip(b, t.Radius+2, func() {
			for i, cl := range cs {
				if len(cl.x) == 0 {
					continue
				}
				alpha := float32(0.22)
				if i == lit {
					alpha = 0.45
				}
				path := cl.path(b.X, b.Y)
				p.FillPath(path, hues[i%len(hues)].Alpha(alpha))
				p.StrokePath(path, 1.5, t.Background)
			}
			if delaunay {
				for i, cl := range cs {
					for _, j := range cl.across {
						if j > i {
							var line ui.Path
							line.MoveTo(b.X+float32(points[i].X)*b.W, b.Y+float32(points[i].Y)*b.H)
							line.LineTo(b.X+float32(points[j].X)*b.W, b.Y+float32(points[j].Y)*b.H)
							p.StrokePath(&line, 1, t.Text.Alpha(0.45))
						}
					}
				}
			}
			for i, pt := range points {
				x, y := b.X+float32(pt.X)*b.W, b.Y+float32(pt.Y)*b.H
				r := grab / 2
				if i == lit {
					r *= 1.3
				}
				var dot ui.Path
				dot.Circle(x, y, r)
				p.FillPath(&dot, t.Text)
				p.StrokePath(&dot, 1.5, t.Background)
			}
		})
	})
	if changed {
		box.SetChanged()
	}
	return box
}
