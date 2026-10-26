// Package rulers puts rulers along the top and left of a zoomview, as a
// design tool's canvas has them: their ticks and numbers count the
// content's DIPs at the view's zoom and pan, and a mark on each follows
// the pointer. Dragging from a ruler into the view pulls out a guide
// across the content, from the top ruler a horizontal one and from the
// left a vertical one; dragging a guide moves it, and dragging it back
// onto its ruler removes it. Guides stop at whole DIPs of the content,
// and snap to its edges and middle, and to the lines in State.Snaps,
// within 6 DIPs of the view.
//
//	rulers.View(c, &app.canvas, 800, 600, &app.guides, func() {
//		app.artboard(c)
//	}).Grow(1)
//
// The view zooms and pans as zoomview.View does; State.Zoom is its
// zoomview.State.
package rulers

import (
	"math"
	"slices"
	"strconv"

	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/zoomview"
)

// size is how thick a ruler is.
const size = 20

// snap is how near, in DIPs of the view, a guide snaps to the content's
// edges and middle.
const snap = 6

// Guide is a line across the content: vertical at X Pos, or horizontal at
// Y Pos, in the content's DIPs.
type Guide struct {
	Vertical bool
	Pos      float32
}

// State is what a view keeps: how it shows the content, and the guide a
// drag from a ruler pulls out. Keep it in the app's state, one for each
// view.
type State struct {
	Zoom zoomview.State
	// Snaps are more lines guides snap to, as the edges of the content's
	// elements, which the app sets.
	Snaps []Guide
	// pulling is the index of the guide a ruler's drag pulls out, plus
	// one.
	pulling int
}

// View creates the rulers and the zoomview of content w by h DIPs, which
// content builds, with the guides *guides over it. Changed reports a
// change of the guides.
func View(c *ui.Context, s *State, w, h float32, guides *[]Guide, content func()) ui.Element {
	t := c.Theme()
	root := ui.Column(c).Label("Canvas")
	changed := false
	root.Children(func() {
		var top, left, area ui.Element
		ui.Row(c).Height(size).AlignItems(ui.Stretch).Children(func() {
			ui.Box(c).Size(size, size).Background(t.Surface).BorderWidth(0, 1, 1, 0).BorderColor(t.Border)
			top = ui.Box(c).Grow(1).Background(t.Surface).BorderWidth(0, 0, 1, 0).BorderColor(t.Border).Cursor(ui.CursorResizeNS).Label("Horizontal ruler")
		})
		ui.Row(c).Grow(1).AlignItems(ui.Stretch).Children(func() {
			left = ui.Box(c).Width(size).Background(t.Surface).BorderWidth(0, 1, 0, 0).BorderColor(t.Border).Cursor(ui.CursorResizeEW).Label("Vertical ruler")
			area = ui.Box(c).Grow(1).Clip()
		})
		z := s.Zoom.Zoom()
		px, py, over := area.PointerPosition()
		// at returns the content's point at the pointer, along x or y,
		// snapped to the nearest of the content's edges and middle and
		// s.Snaps.
		at := func(vertical bool) float32 {
			px, py, _ := area.PointerPosition()
			ox, oy := s.Zoom.Origin()
			v, end := oy+py/z, h
			if vertical {
				v, end = ox+px/z, w
			}
			to, near := float32(math.Round(float64(v))), float32(snap)/z
			try := func(p float32) {
				if d := float32(math.Abs(float64(v - p))); d <= near {
					to, near = p, d
				}
			}
			for _, p := range []float32{0, end / 2, end} {
				try(p)
			}
			for _, g := range s.Snaps {
				if g.Vertical == vertical {
					try(g.Pos)
				}
			}
			return to
		}
		// outside reports whether the pointer is over the ruler of a guide.
		outside := func(vertical bool) bool {
			px, py, _ := area.PointerPosition()
			if vertical {
				return px < 0
			}
			return py < 0
		}
		drop := -1
		// follow moves guide i with the drag of e, and drops it if the
		// drag ends over its ruler.
		follow := func(e ui.Element, i int) {
			g := &(*guides)[i]
			if _, _, ok := e.Dragged(); ok {
				if p := at(g.Vertical); p != g.Pos {
					g.Pos, changed = p, true
				}
			}
			if _, _, ok := e.DragEnded(); ok && outside(g.Vertical) {
				drop, changed = i, true
			}
		}
		for _, r := range []struct {
			e        ui.Element
			vertical bool
		}{{top, false}, {left, true}} {
			if dx, dy, ok := r.e.Dragged(); ok && s.pulling == 0 && (dx != 0 || dy != 0) {
				*guides = append(*guides, Guide{Vertical: r.vertical, Pos: at(r.vertical)})
				s.pulling, changed = len(*guides), true
			}
		}
		// The pull ends as the ruler is let go, rather than at DragEnded,
		// which a press that barely moves never reports.
		if i := s.pulling - 1; i >= len(*guides) {
			s.pulling = 0
		} else if i >= 0 {
			g := &(*guides)[i]
			ruler := top
			if g.Vertical {
				ruler = left
			}
			if _, _, ok := ruler.Dragged(); !ok {
				if outside(g.Vertical) {
					drop, changed = i, true
				}
				s.pulling = 0
			} else if p := at(g.Vertical); p != g.Pos {
				g.Pos, changed = p, true
			}
		}

		area.Children(func() {
			zoomview.View(c, &s.Zoom, w, h, content).Fill().Background(t.Background)
			ox, oy := s.Zoom.Origin()
			b := area.LayoutBounds()
			for i, g := range *guides {
				line := ui.Box(c).Key(i).Absolute().Label("Guide")
				if g.Vertical {
					line.Left((g.Pos-ox)*z-3).Top(0).Size(7, b.H).Cursor(ui.CursorResizeEW)
				} else {
					line.Top((g.Pos-oy)*z-3).Left(0).Size(b.W, 7).Cursor(ui.CursorResizeNS)
				}
				line.Value(strconv.FormatFloat(float64(g.Pos), 'f', -1, 32))
				color := guideColor(t.Dark)
				if line.Hovered() || line.Pressed() {
					color = t.Accent
				}
				line.Draw(func(p *ui.Painter, r ui.Rect) {
					if g.Vertical {
						p.Line(r.X+3.5, r.Y, r.X+3.5, r.Y+r.H, 1, color)
					} else {
						p.Line(r.X, r.Y+3.5, r.X+r.W, r.Y+3.5, 1, color)
					}
				})
				if s.pulling != i+1 {
					follow(line, i)
				}
			}
		})
		if drop >= 0 {
			*guides = slices.Delete(*guides, drop, drop+1)
			s.pulling = 0
		}

		ox, oy := s.Zoom.Origin()
		top.Draw(func(p *ui.Painter, r ui.Rect) {
			ticks(p, t, r, ox, z, false)
			if over {
				p.Line(r.X+px+0.5, r.Y, r.X+px+0.5, r.Y+r.H, 1, t.Accent)
			}
		})
		left.Draw(func(p *ui.Painter, r ui.Rect) {
			ticks(p, t, r, oy, z, true)
			if over {
				p.Line(r.X, r.Y+py+0.5, r.X+r.W, r.Y+py+0.5, 1, t.Accent)
			}
		})
	})
	if changed {
		root.SetChanged()
	}
	return root
}

// guideColor is the color of a guide at rest.
func guideColor(dark bool) ui.Color {
	if dark {
		return ui.Oklch(0.72, 0.16, 200)
	}
	return ui.Oklch(0.62, 0.17, 200)
}

// ticks draws the ticks and numbers of a ruler in r, its start showing
// the content's point from at zoom z, down the ruler if vertical.
func ticks(p *ui.Painter, t *ui.Theme, r ui.Rect, from, z float32, vertical bool) {
	length := r.W
	if vertical {
		length = r.H
	}
	if z <= 0 || length <= 0 {
		return
	}
	major := step(50 / float64(z))
	minor := major / 10
	if minor*float64(z) < 5 {
		minor = major / 5
	}
	if minor*float64(z) < 5 {
		minor = major / 2
	}
	color := t.TextMuted
	first := math.Floor(float64(from) / minor)
	last := math.Ceil((float64(from) + float64(length/z)) / minor)
	per := int(math.Round(major / minor))
	for i := first; i <= last; i++ {
		v := i * minor
		d := (float32(v)-from)*z + 0.5
		n := int(math.Round(i)) % per
		if n < 0 {
			n += per
		}
		tick := float32(size) / 4
		switch {
		case n == 0:
			tick = size
		case per%2 == 0 && n == per/2:
			tick = size / 2
		}
		if vertical {
			p.Line(r.X+r.W-tick, r.Y+d, r.X+r.W, r.Y+d, 1, color.Alpha(0.6))
		} else {
			p.Line(r.X+d, r.Y+r.H-tick, r.X+d, r.Y+r.H, 1, color.Alpha(0.6))
		}
		if n != 0 {
			continue
		}
		label := strconv.FormatFloat(math.Round(v*1e6)/1e6, 'f', -1, 64)
		if vertical {
			x, y := r.X+12, r.Y+d+2
			p.Transform(ui.Rotation(90), x, y, func() {
				p.Text(x, y, label, 9, color)
			})
		} else {
			p.Text(r.X+d+3, r.Y+1, label, 9, color)
		}
	}
}

// step returns the least of 1, 2 or 5 times a power of ten no less than v.
func step(v float64) float64 {
	pow := math.Pow(10, math.Floor(math.Log10(v)))
	for _, m := range []float64{1, 2, 5, 10} {
		if m*pow >= v {
			return m * pow
		}
	}
	return 10 * pow
}
