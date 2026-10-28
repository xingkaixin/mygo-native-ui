// Package sheet shows a sheet over the bottom of a view, as Maps and Find
// My on iPhone do: it rests at a few heights, its detents, and the user
// drags it between them. Let go, it settles at the nearest, and a flick
// goes on to the next one; past the lowest and the highest it gives less
// and less, and springs back. A click on its grabber raises it to the
// next detent, and from the highest lowers it to the lowest. Above the
// second highest detent the view under it dims, and a click there lowers
// the sheet to the lowest.
//
//	sheet.View(c, &app.detent, []sheet.Detent{sheet.Height(96), sheet.Fraction(0.5), sheet.Fraction(0.92)},
//		func() { mapView(c) },
//		func() { places(c) },
//	).Grow(1)
//
// *detent is the detent it rests at, 0 the lowest: the sheet sets it as
// the user lets it go, and the app sets it to move the sheet, as raising
// it while a search field in it has the focus. The sheet is as tall as
// its highest detent, and shows as much of its top as it is raised.
package sheet

import (
	"math"

	"github.com/egoist/mygo/ui"
)

// dim is how dark the view under the sheet gets at the highest detent.
const dim = 0.3

// Detent is a height a sheet rests at.
type Detent struct {
	dips, part float32
}

// Height is a detent dips DIPs tall.
func Height(dips float32) Detent { return Detent{dips: dips} }

// Fraction is a detent the fraction f of the view's height tall.
func Fraction(f float32) Detent { return Detent{part: f} }

func (d Detent) in(h float32) float32 {
	if d.part > 0 {
		return d.part * h
	}
	return d.dips
}

// state is what a view keeps from frame to frame: the sheet's offset from
// the top of the view, and the detent and the view's height it was set
// for.
type state struct {
	offset ui.Point
	detent int
	height float32
}

// View creates a view showing content filling it, and a sheet over its
// bottom whose content build builds, resting at detents, from the lowest
// to the highest. Give the view a size, or grow it in its parent.
func View(c *ui.Context, detent *int, detents []Detent, content, build func()) ui.Element {
	t := c.Theme()
	view := ui.Box(c).Clip()
	n := len(detents)
	if n == 0 {
		return view.Children(content)
	}
	st := ui.Local(view, "sheet.state", func() state { return state{detent: -1} })
	h := view.LayoutBounds().H
	*detent = max(0, min(n-1, *detent))
	heights := make([]float32, n)
	rests := make([]ui.Point, n)
	for i, d := range detents {
		heights[i] = max(0, min(h, d.in(h)))
		rests[i] = ui.Point{Y: h - heights[i]}
	}
	low, high := heights[0], heights[n-1]
	if st.height == 0 && h > 0 {
		c.DisableAnimations()
	}

	view.Children(func() {
		ui.Column(c).Absolute().Left(0).Top(0).Right(0).Bottom(0).Children(content)
		shade := ui.Box(c).Absolute().Left(0).Top(0).Right(0).Bottom(0).Background(ui.RGB(0, 0, 0))
		// The sheet reaches a quarter of the view below its bottom, so that
		// none of the view shows under it as it is pulled past the highest
		// detent.
		extra := h / 4
		panel := ui.Column(c).Absolute().Left(0).Right(0).Top(0).Height(high+extra).
			Background(t.Background).Radius(t.Radius*3, t.Radius*3, 0, 0).Border(1, t.Border).
			Shadow(0, -2, 16, 0, ui.RGB(0, 0, 0).Alpha(0.15)).Label("Sheet")
		switch {
		case panel.Dragging():
		case *detent != st.detent || h != st.height:
			st.offset = rests[*detent]
		case st.offset != rests[*detent]:
			*detent = nearest(heights, h-st.offset.Y)
		}
		st.detent, st.height = *detent, h
		panel.Draggable(&st.offset, ui.DraggableOptions{
			Axis:   ui.Vertical,
			Bounds: &ui.Rect{Y: h - high, H: high - low},
			Snap:   ui.Nearest(rests...),
		})
		raised := h - panel.DragOffset().Y
		if from := heights[max(0, n-2)]; raised > from && high > from {
			shade.Opacity(dim * min(1, (raised-from)/(high-from)))
			if shade.Clicked() {
				*detent = 0
			}
		} else {
			shade.Opacity(0).PassThrough()
		}

		// The grabber is no button of its own, which would keep the press
		// from dragging the sheet.
		grab := t.Space(6)
		if _, y, ok := panel.PointerPosition(); ok && y < grab && panel.Clicked() {
			*detent = (*detent + 1) % n
		}
		panel.Children(func() {
			ui.Box(c).Height(grab).Center().Children(func() {
				ui.Box(c).Size(t.Space(9), 5).Radius(2.5).Background(t.TextMuted.Alpha(0.4))
			})
			ui.Column(c).Height(max(0, high-grab)).Children(build)
		})
	})
	return view
}

// nearest returns the detent of heights nearest h.
func nearest(heights []float32, h float32) int {
	best := 0
	for i, d := range heights {
		if math.Abs(float64(d-h)) < math.Abs(float64(heights[best]-h)) {
			best = i
		}
	}
	return best
}
