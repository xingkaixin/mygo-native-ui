package chartx

import (
	"math"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/chart"
)

// Zoom makes the plot's area zoom and pan along x. *view is the stretch of
// x's whole domain that the plot shows, in fractions of it: start it at
// chart.Span{Lo: 0, Hi: 1}, all of it. Zoom keeps it within 0 and 1, and
// at least minSpan long. Build the plot's x scale from it with the Invert
// of the whole domain's scale, which makes a Log scale zoom and pan by the
// logarithms of its values:
//
//	all := chart.NewTime(first, last)
//	x := chart.NewTime(all.Invert(app.view.Lo), all.Invert(app.view.Hi))
//	chart.Plot(c, chart.Axes{Bottom: x.Ticks(6)}, func(area ui.Element) {
//		chartx.Zoom(c, area, &app.view, 0.02)
//		...
//	})
//
// Scrolling with Cmd held, Command on macOS and Control elsewhere, as in
// design apps, zooms in and out around the pointer.
// Scrolling sideways, as two fingers do on a touchpad or a wheel does with
// Shift held, pans, as does dragging the area; a double click shows the
// whole domain again. Scrolling up and down without Cmd, and sideways
// where the view can go no further, scrolls what is around the plot.
//
// Zoom clips the marks in the area to its left and right, as zoomed in
// they go past them; clip what a Draw function of the area paints with
// Painter.Clip. Zoomed in, a path through all the points of a line can
// grow too wide to paint: draw those chart.MinMax keeps, the points in
// view and the nearest beyond each side.
func Zoom(c *ui.Context, area ui.Element, view *chart.Span, minSpan float32) {
	area.ClipX()
	w := area.Bounds().W
	if dx, _, ok := area.Dragged(); ok && dx != 0 && w > 0 {
		*view = pan(*view, -dx/w)
	}
	if area.DoubleClicked() {
		*view = chart.Span{Lo: 0, Hi: 1}
	}
	// Scrolling comes between builds, at the box laid out since, which
	// only a handle reads then.
	h := ui.Local(area, "chartx.Zoom", func() ui.Handle { return ui.Handle{} })
	area.Bind(h)
	area.HandleInput(func(ev ui.InputEvent) bool {
		w := h.Bounds(c).W
		if ev.Kind != ui.InputScroll || w <= 0 {
			return false
		}
		if ev.Mods&ui.Cmd != 0 && ev.DY != 0 {
			// Scrolling 300 DIPs zooms by 2: a notch of a mouse's wheel,
			// 40 to 100 DIPs by platform, by 10 to 25%.
			*view = zoom(*view, ev.X/w, float32(math.Exp2(float64(ev.DY)/300)), minSpan)
			return true
		}
		dx, dy := ev.DX, ev.DY
		if ev.Mods&ui.Shift != 0 && dx == 0 {
			// As the window turns a wheel sideways for scroll containers,
			// which it does only once the plot leaves the event.
			dx, dy = dy, 0
		}
		if math.Abs(float64(dx)) <= math.Abs(float64(dy)) {
			return false
		}
		v := pan(*view, dx/w)
		if v == *view {
			return false
		}
		*view = v
		return true
	})
}

// zoom scales the length of view by factor around at, a fraction of it
// that stays where it is, keeping it between minSpan and 1 long.
func zoom(view chart.Span, at, factor, minSpan float32) chart.Span {
	span := view.Hi - view.Lo
	pivot := view.Lo + at*span
	span = max(minSpan, min(1, span*factor))
	return fit(pivot-at*span, span)
}

// pan moves view by d of its length.
func pan(view chart.Span, d float32) chart.Span {
	span := view.Hi - view.Lo
	return fit(view.Lo+d*span, span)
}

// fit returns the stretch span long from lo, moved to keep within 0 and 1.
func fit(lo, span float32) chart.Span {
	lo = max(0, min(1-span, lo))
	return chart.Span{Lo: lo, Hi: lo + span}
}
