// Package zoomview shows content that the user zooms and pans, as an image
// in Preview or a design in Figma:
//
//	zoomview.View(c, &app.zoom, photoW, photoH, func() {
//		ui.Image(c, app.photo).Fill()
//	}).Grow(1).Label("Photo")
//	ui.Textf(c, "%.0f%%", app.zoom.Zoom()*100)
//
// The content is laid out at its own size and scaled with a transform, so
// that the pointer, text and assistive technology follow it as they follow
// any transformed element: it may be an image, a drawing, or controls.
package zoomview

import (
	"math"
	"strconv"

	"github.com/egoist/mygo/ui"
)

// maxZoom is how far a view zooms in: 32 DIPs of the view for a DIP of the
// content.
const maxZoom = 32

// step is how far ZoomIn and ZoomOut zoom: two steps double the size.
var step = float32(math.Sqrt2)

// State is how a view shows its content: how many DIPs of the view a DIP
// of the content takes, and the point of the content at the view's
// center. The zero State fits the content in the view, and follows the
// view's size until the user zooms or pans. Keep it in the app's state,
// one for each view.
type State struct {
	scale, x, y float32
	// fit is the scale fitting the content in the view, and w, h, vw and
	// vh the sizes of the content and the view, as of the last frame.
	fit, w, h, vw, vh float32
}

// Zoom returns how many DIPs of the view a DIP of the content takes, as a
// label shows it: 1 at the content's own size.
func (s *State) Zoom() float32 {
	if s.scale == 0 {
		return s.fit
	}
	return s.scale
}

// Fit fits the content in the view, no larger than its own size, and
// follows the view's size until the user zooms or pans again.
func (s *State) Fit() { s.scale = 0 }

// ActualSize shows the content at its own size, around the view's center.
func (s *State) ActualSize() { s.zoomAt(1, s.vw/2, s.vh/2) }

// ZoomIn zooms in a step around the view's center; two steps double the
// size.
func (s *State) ZoomIn() { s.zoomAt(s.Zoom()*step, s.vw/2, s.vh/2) }

// ZoomOut zooms out a step around the view's center.
func (s *State) ZoomOut() { s.zoomAt(s.Zoom()/step, s.vw/2, s.vh/2) }

// zoomAt sets the scale to z, between the fit and maxZoom, keeping the
// point of the content at (px, py) of the view where it is.
func (s *State) zoomAt(z, px, py float32) {
	from := s.Zoom()
	if s.scale == 0 {
		s.x, s.y = s.w/2, s.h/2
	}
	if from <= 0 {
		return
	}
	z = max(s.fit, min(maxZoom, z))
	cx, cy := s.x+(px-s.vw/2)/from, s.y+(py-s.vh/2)/from
	s.scale, s.x, s.y = z, cx-(px-s.vw/2)/z, cy-(py-s.vh/2)/z
	s.clamp()
}

// pan moves the content by dx, dy DIPs of the view, and reports whether
// it moved.
func (s *State) pan(dx, dy float32) bool {
	if s.scale == 0 {
		s.scale, s.x, s.y = s.fit, s.w/2, s.h/2
	}
	x, y := s.x, s.y
	s.x -= dx / s.scale
	s.y -= dy / s.scale
	s.clamp()
	return s.x != x || s.y != y
}

// clamp keeps the content's edges outside the view, or centers it along
// an axis it does not fill.
func (s *State) clamp() {
	s.x = clampAxis(s.x, s.w, s.vw/s.scale)
	s.y = clampAxis(s.y, s.h, s.vh/s.scale)
}

// clampAxis keeps the center c of a span of the view seen showing span
// DIPs of content that is size long within it, or at its middle.
func clampAxis(c, size, span float32) float32 {
	if span >= size {
		return size / 2
	}
	return max(span/2, min(size-span/2, c))
}

// layout fits s to content w by h in a view vw by vh: the fit's scale,
// and where the content is while it fits.
func (s *State) layout(w, h, vw, vh float32) {
	s.w, s.h, s.vw, s.vh = w, h, vw, vh
	s.fit = 1
	if w > 0 && h > 0 && vw > 0 && vh > 0 {
		s.fit = min(1, vw/w, vh/h)
	}
	if s.scale == 0 {
		s.x, s.y = w/2, h/2
		return
	}
	s.scale = max(s.fit, s.scale)
	s.clamp()
}

// View creates a view of content w by h DIPs, which content builds, as
// the children of an element of that size, and s says how it shows. Give
// the view a size, or grow it in its parent, and name it with Label.
//
// Scrolling with Cmd held, Command on macOS and Control elsewhere, zooms
// around the pointer; scrolling without it, and dragging, pan. A double
// click zooms in where it is, to the content's own size or twice the fit,
// and out to the fit again. While the view has the keyboard focus, Cmd+=
// and Cmd+- zoom in and out, Cmd+0 shows the content at its own size and
// Cmd+9 fits it, as in Preview, and the arrows pan. The view zooms from
// the fit, no larger than the content's own size, to 32 times that size.
func View(c *ui.Context, s *State, w, h float32, content func()) ui.Element {
	view := ui.Box(c).Clip().Focusable()
	b := view.LayoutBounds()
	s.layout(w, h, b.W, b.H)

	if s.w*s.Zoom() > s.vw || s.h*s.Zoom() > s.vh {
		view.Cursor(ui.CursorGrab)
	}
	if dx, dy, ok := view.Dragged(); ok && (dx != 0 || dy != 0) {
		s.pan(dx, dy)
		view.Cursor(ui.CursorGrabbing)
	}
	if view.DoubleClicked() {
		if s.Zoom() > s.fit {
			s.Fit()
		} else {
			px, py, _ := view.PointerPosition()
			s.zoomAt(max(1, 2*s.fit), px, py)
		}
	}
	for _, k := range []struct {
		mods ui.Modifiers
		key  ui.Key
		do   func()
	}{
		{ui.Cmd, ui.KeyEqual, s.ZoomIn},
		{ui.Cmd | ui.Shift, ui.KeyEqual, s.ZoomIn},
		{ui.Cmd, ui.KeyMinus, s.ZoomOut},
		{ui.Cmd, ui.Key0, s.ActualSize},
		{ui.Cmd, ui.Key9, s.Fit},
		{0, ui.KeyLeft, func() { s.pan(40, 0) }},
		{0, ui.KeyRight, func() { s.pan(-40, 0) }},
		{0, ui.KeyUp, func() { s.pan(0, 40) }},
		{0, ui.KeyDown, func() { s.pan(0, -40) }},
	} {
		if view.Shortcut(k.mods, k.key) {
			k.do()
		}
	}
	// Scrolling comes between builds, at the box laid out since, which
	// only a handle reads then.
	ref := ui.Local(view, "zoomview", func() ui.Handle { return ui.Handle{} })
	view.Bind(ref)
	view.HandleInput(func(ev ui.InputEvent) bool {
		if ev.Kind != ui.InputScroll {
			return false
		}
		b := ref.Bounds(c)
		s.vw, s.vh = b.W, b.H
		if ev.Mods&ui.Cmd != 0 {
			// Scrolling 300 DIPs zooms by 2, as chartx.Zoom does.
			s.zoomAt(s.Zoom()/float32(math.Exp2(float64(ev.DY)/300)), ev.X, ev.Y)
			return true
		}
		return s.pan(-ev.DX, -ev.DY)
	})

	z := s.Zoom()
	view.Value(strconv.Itoa(int(math.Round(float64(z)*100))) + "%")
	view.Children(func() {
		ui.Box(c).Absolute().Left(0).Top(0).Size(w, h).TransformOrigin(0, 0).
			Scale(z, z).Translate(s.vw/2-s.x*z, s.vh/2-s.y*z).Children(content)
	})
	return view
}
