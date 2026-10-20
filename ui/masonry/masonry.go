// Package masonry lays items out in columns of their own heights, as
// Pinterest's boards do: as many columns as fit a width, each item below
// the shortest column. The app gives the height of each item for a
// column's width, from its aspect ratio or what it measures, and builds
// the items; only those in view are built, so a board of thousands
// scrolls as smoothly as one of ten. As the board's width changes, the
// columns reflow and the item at the top stays in view.
//
//	masonry.View(c, &app.board, len(app.pins), 236, func(i int, w float32) float32 {
//		return w/app.pins[i].Ratio + captionHeight // quick: asked each frame
//	}, func(i int) {
//		pin(c, app.pins[i]) // fills the box it is given
//	}).Grow(1)
package masonry

import "github.com/egoist/mygo/ui"

// State is what a board keeps: where it is scrolled, and where its items
// went. Keep it in the app's state, one for each board.
type State struct {
	scroll ui.ScrollState
	places []place
	// colW is the width of the columns of places, 0 before the first
	// layout.
	colW float32
}

// place is where an item went on the board: its left and top, and its
// height.
type place struct{ x, y, h float32 }

// View creates a board of n items, of which s keeps the place: as many
// columns as fit minWidth DIPs wide, each item going below the shortest,
// height(i, w) DIPs high in a column w DIPs wide, and item(i) building it
// in a box of that size. View asks every height each frame, so keep
// height quick; it builds the items in view only. Give it a size, or grow
// it in its parent.
func View(c *ui.Context, s *State, n int, minWidth float32, height func(i int, width float32) float32, item func(i int)) ui.Element {
	gap := c.Theme().Space(4)
	scroll := ui.Scroll(c).TrackScroll(&s.scroll)
	view := scroll.LayoutBounds()
	w := view.W - 2*gap
	cols := max(1, int((w+gap)/(minWidth+gap)))
	colW := (w - gap*float32(cols-1)) / float32(cols)
	if view.W <= 0 {
		n, colW = 0, 0
	}

	top, old := -1, place{}
	if colW != s.colW && s.colW > 0 {
		for i, p := range s.places {
			if p.y+p.h > s.scroll.Y {
				top, old = i, p
				break
			}
		}
	}
	bottoms := make([]float32, cols)
	for k := range bottoms {
		bottoms[k] = gap
	}
	s.places = s.places[:0]
	for i := range n {
		k := 0
		for j, b := range bottoms {
			if b < bottoms[k] {
				k = j
			}
		}
		h := max(0, height(i, colW))
		s.places = append(s.places, place{gap + float32(k)*(colW+gap), bottoms[k], h})
		bottoms[k] += h + gap
	}
	if top >= 0 && top < n {
		p := s.places[top]
		d := s.scroll.Y - old.y
		// Scrolled into the item, the same share of it stays above the
		// view, as it may be shorter now; above it, the same gap.
		if d > 0 && old.h > 0 {
			d *= p.h / old.h
		}
		s.scroll.Y = max(0, p.y+d)
	}
	s.colW = colW
	total := gap
	for _, b := range bottoms {
		total = max(total, b)
	}

	lo, hi := s.scroll.Y-view.H/2, s.scroll.Y+view.H*3/2
	return scroll.Children(func() {
		board := ui.Box(c).Height(total)
		board.Draw(func(p *ui.Painter, r ui.Rect) {
			// The columns were laid out at the width of the last frame.
			if r.W != view.W {
				c.Invalidate()
			}
		})
		board.Children(func() {
			for i, p := range s.places {
				// Each item goes below the shortest column, so tops only
				// go down from one to the next.
				if p.y > hi {
					break
				}
				if p.y+p.h < lo {
					continue
				}
				ui.Box(c.Key(i)).Absolute().Left(p.x).Top(p.y).Size(colW, p.h).Children(func() { item(i) })
			}
		})
	})
}
