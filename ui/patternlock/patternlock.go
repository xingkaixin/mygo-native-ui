// Package patternlock takes a pattern drawn through a three-by-three grid
// of dots, as phones unlock: pressing on a dot and dragging through others
// joins them with a line, which follows the pointer to the next, and a
// dot passed over on the way to another joins too. Each dot joins once.
// Letting go ends the pattern, which the app checks in that frame; Wrong
// turns it red and shakes it before it clears, and otherwise it stays
// until the next press.
//
//	if ev := patternlock.View(c, &app.lock); ev.Drawn != nil && !app.matches(ev.Drawn) {
//		app.lock.Wrong(c)
//	}
package patternlock

import (
	"fmt"
	"math"
	"slices"
	"time"

	"github.com/egoist/mygo/ui"
)

// State is what a grid keeps: the dots joined, in order, and whether and
// when they were found wrong. Keep it in the app's state, one for each
// grid.
type State struct {
	dots  []int
	down  bool
	wrong time.Time
}

// Events is a grid, and the dots of a pattern finished in the frame, by
// row from the top left, 0 to 8, or nil.
type Events struct {
	ui.Element
	Drawn []int
}

const shake = 900 * time.Millisecond

// Wrong shows the pattern drawn as wrong, then clears it.
func (s *State) Wrong(c *ui.Context) {
	s.wrong = c.Now()
	c.After(shake)
}

// View creates the grid.
func View(c *ui.Context, s *State) Events {
	t := c.Theme()
	var ev Events
	now := c.Now()
	wrong := !s.wrong.IsZero()
	if wrong && now.Sub(s.wrong) >= shake {
		s.dots, s.wrong, wrong = nil, time.Time{}, false
	}
	size := t.Space(72)
	cell := size / 3
	e := ui.Box(c).Size(size, size).Role(ui.RoleImage).Label("Pattern")
	px, py, _ := e.PointerPosition()
	pressed := e.Pressed()
	if pressed && !s.down {
		s.dots, s.wrong, wrong = nil, time.Time{}, false
	}
	if pressed {
		col, row := int(px/cell), int(py/cell)
		cx, cy := (float32(col)+0.5)*cell, (float32(row)+0.5)*cell
		if col >= 0 && col < 3 && row >= 0 && row < 3 && math.Hypot(float64(px-cx), float64(py-cy)) < float64(cell)*0.32 {
			dot := row*3 + col
			if !slices.Contains(s.dots, dot) {
				if n := len(s.dots); n > 0 {
					last := s.dots[n-1]
					// A dot halfway along the line joins on the way.
					if (last/3+row)%2 == 0 && (last%3+col)%2 == 0 {
						if mid := (last + dot) / 2; !slices.Contains(s.dots, mid) {
							s.dots = append(s.dots, mid)
						}
					}
				}
				s.dots = append(s.dots, dot)
			}
		}
	}
	if !pressed && s.down && len(s.dots) > 0 {
		ev.Drawn = slices.Clone(s.dots)
		e.SetChanged()
	}
	s.down = pressed
	e.Value(fmt.Sprintf("%d dots", len(s.dots)))
	dots := slices.Clone(s.dots)
	since := s.wrong
	e.Draw(func(p *ui.Painter, r ui.Rect) {
		col := t.Accent
		dx := float32(0)
		if wrong {
			col = ui.Hex("#E03131")
			k := float64(p.Now().Sub(since)) / float64(shake)
			if k < 0.5 {
				dx = float32(math.Sin(k*2*math.Pi*6)*(1-k*2)) * 10
				p.AnimationFrame()
			}
		}
		center := func(d int) (float32, float32) {
			return r.X + (float32(d%3)+0.5)*cell + dx, r.Y + (float32(d/3)+0.5)*cell
		}
		if len(dots) > 0 {
			var line ui.Path
			for i, d := range dots {
				x, y := center(d)
				if i == 0 {
					line.MoveTo(x, y)
				} else {
					line.LineTo(x, y)
				}
			}
			if pressed {
				line.LineTo(r.X+px, r.Y+py)
			}
			p.StrokePath(line.Cap(ui.RoundCap), 5, col.Alpha(0.55))
		}
		for d := range 9 {
			x, y := center(d)
			if slices.Contains(dots, d) {
				var halo ui.Path
				halo.Circle(x, y, cell*0.22)
				p.FillPath(&halo, col.Alpha(0.18))
				var dot ui.Path
				dot.Circle(x, y, 8)
				p.FillPath(&dot, col)
				continue
			}
			var dot ui.Path
			dot.Circle(x, y, 6)
			p.FillPath(&dot, t.TextMuted.Alpha(0.6))
		}
	})
	ev.Element = e
	return ev
}
