// Package compare shows two pictures, a before and an after, as code
// review tools show a changed image. A segmented control in the view's
// toolbar chooses how: Slider shows the before left of a divider and the
// after right of it, the divider dragged across or moved by the arrow
// keys, a tenth of the way with Shift; Side by side shows them next to
// each other; Onion skin shows the after over the before at the opacity
// of a slider; and Difference shows the pixels whose color changed in a
// strong color over the before dimmed, those in only one picture in a
// lighter tone of it, with how many changed. In every mode the pictures
// zoom and pan together: the wheel zooms around the pointer, dragging
// pans, and Fit and 100% fit them in the view and show them at their own
// size. Pictures of different sizes align at their top left, and the
// toolbar shows both sizes. Pass the same pictures each frame: new ones
// are compared anew.
//
//	compare.View(c, &app.compare, app.before, app.after).Grow(1)
package compare

import (
	"fmt"
	"image"
	"image/draw"
	"math"

	"github.com/egoist/mygo/ui"
)

const (
	slider = iota
	sideBySide
	onion
	difference
)

var modes = []string{"Slider", "Side by side", "Onion skin", "Difference"}

// maxZoom is how far the view zooms in: 32 DIPs for a pixel.
const maxZoom = 32

// tolerance is how far a channel of a pixel, of 255, may change before
// the pixel counts as changed, so that the noise of JPEG and dithering
// does not.
const tolerance = 16

// State is what a view keeps: the mode, where the divider is, the
// opacity of the after, how the pictures are zoomed and panned, and their
// difference, computed when the pictures change. Keep it in the app's
// state, one for each view.
type State struct {
	mode    int
	split   float32
	opacity float64

	before, after image.Image
	pics          [2]*ui.Bitmap
	diff          *ui.Bitmap
	changed, area int

	// scale is how many DIPs of a pane a pixel takes, 0 to fit the
	// pictures in it, and x, y the point of the pictures at its center.
	scale, x, y float32
	// fit is the scale fitting the pictures in a pane, and w, h, vw, vh
	// and pad the sizes of the pictures, the pane and its margin, as of
	// the last frame.
	fit, w, h, vw, vh, pad float32
}

// load makes the bitmaps of before and after, and their difference.
func (s *State) load(before, after image.Image) {
	// The zero State's first pictures start the divider and the opacity
	// half way.
	if s.before == nil {
		s.split, s.opacity = 0.5, 0.5
	}
	s.before, s.after = before, after
	a, b := rgba(before), rgba(after)
	s.pics = [2]*ui.Bitmap{ui.NewBitmap(a).Pixelated(), ui.NewBitmap(b).Pixelated()}
	w, h := max(a.Rect.Dx(), b.Rect.Dx()), max(a.Rect.Dy(), b.Rect.Dy())
	mask := image.NewRGBA(image.Rect(0, 0, w, h))
	s.changed, s.area = 0, w*h
	for y := range h {
		for x := range w {
			if v := change(a, b, x, y); v > 0 {
				s.changed++
				i := mask.PixOffset(x, y)
				copy(mask.Pix[i:i+4], []uint8{v, v, v, v})
			}
		}
	}
	s.diff = ui.NewBitmap(mask).Pixelated()
}

// rgba returns img with its top left at 0, 0.
func rgba(img image.Image) *image.RGBA {
	b := img.Bounds()
	out := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(out, out.Rect, img, b.Min, draw.Src)
	return out
}

// change returns how much of the strong color the pixel at x, y takes:
// none unless it changed beyond the tolerance, all if it did, and a part
// if it is in only one of a and b, so that what it shows shows through.
func change(a, b *image.RGBA, x, y int) uint8 {
	p := image.Pt(x, y)
	inA, inB := p.In(a.Rect), p.In(b.Rect)
	if inA != inB {
		return 80
	}
	if !inA {
		return 0
	}
	i, j := a.PixOffset(x, y), b.PixOffset(x, y)
	for k := range 4 {
		if d := int(a.Pix[i+k]) - int(b.Pix[j+k]); d > tolerance || d < -tolerance {
			return 255
		}
	}
	return 0
}

// zoom returns how many DIPs of a pane a pixel takes.
func (s *State) zoom() float32 {
	if s.scale == 0 {
		return s.fit
	}
	return s.scale
}

// zoomAt sets the scale to z, between the fit and maxZoom, keeping the
// point of the pictures at px, py of a pane where it is.
func (s *State) zoomAt(z, px, py float32) {
	from := s.zoom()
	z = min(maxZoom, z)
	if from <= 0 || z <= s.fit {
		s.scale = 0
		return
	}
	cx, cy := s.x+(px-s.vw/2)/from, s.y+(py-s.vh/2)/from
	s.scale, s.x, s.y = z, cx-(px-s.vw/2)/z, cy-(py-s.vh/2)/z
	s.clamp()
}

// pan moves the pictures by dx, dy DIPs of a pane.
func (s *State) pan(dx, dy float32) {
	if s.scale == 0 {
		return
	}
	s.x -= dx / s.scale
	s.y -= dy / s.scale
	s.clamp()
}

// clamp keeps the pictures' edges no further in than the margin, or
// centers them along an axis they do not fill.
func (s *State) clamp() {
	m := s.pad / s.scale
	s.x = clampAxis(s.x, s.w, s.vw/s.scale, m)
	s.y = clampAxis(s.y, s.h, s.vh/s.scale, m)
}

// clampAxis keeps the center c of a span of a pane, showing span DIPs of
// pictures size long with a margin m, within them, or at their middle.
func clampAxis(c, size, span, m float32) float32 {
	if span-2*m >= size {
		return size / 2
	}
	return max(span/2-m, min(size-span/2+m, c))
}

// layout fits s to pictures w by h in a pane vw by vh with a margin pad.
func (s *State) layout(w, h, vw, vh, pad float32) {
	s.w, s.h, s.vw, s.vh, s.pad = w, h, vw, vh, pad
	s.fit = 1
	if w > 0 && h > 0 && vw > 2*pad && vh > 2*pad {
		s.fit = min(1, (vw-2*pad)/w, (vh-2*pad)/h)
	}
	if s.scale <= s.fit {
		s.scale, s.x, s.y = 0, w/2, h/2
		return
	}
	s.clamp()
}

// panes returns the width of a pane of a view w wide, and how far the
// second starts from the first, 0 when there is one.
func (s *State) panes(w, gap float32) (pw, step float32) {
	if s.mode != sideBySide {
		return w, 0
	}
	pw = (w - gap) / 2
	return pw, pw + gap
}

// View creates the comparison of before and after. Give it a size, or
// Grow.
func View(c *ui.Context, s *State, before, after image.Image) ui.Element {
	t := c.Theme()
	if s.before != before || s.after != after {
		s.load(before, after)
	}
	root := ui.Column(c).Gap(t.Space(3))
	root.Children(func() {
		s.toolbar(c)
		s.canvas(c)
	})
	return root
}

func (s *State) toolbar(c *ui.Context) {
	t := c.Theme()
	ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
		ui.Segmented(c, &s.mode, modes...).Label("Mode")
		ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
			switch s.mode {
			case onion:
				ui.Text(c, "Opacity").TextColor(t.TextMuted)
				ui.Slider(c, &s.opacity, 0, 1).Width(t.Space(32)).Label("Opacity of the after")
				ui.Textf(c, "%.0f%%", s.opacity*100).FontFeatures("tnum").MinWidth(t.Space(9))
			case difference:
				ui.Box(c).Size(t.Space(2.5), t.Space(2.5)).Radius(t.Space(0.5)).Background(t.Danger)
				ui.Textf(c, "%s pixels changed, %.2f%%", thousands(s.changed), 100*float64(s.changed)/float64(max(1, s.area))).
					FontFeatures("tnum")
			}
		})
		ui.Box(c).Grow(1)
		bw, bh := s.pics[0].Size()
		aw, ah := s.pics[1].Size()
		ui.Textf(c, "%d×%d → %d×%d", bw, bh, aw, ah).FontFeatures("tnum").TextColor(t.TextMuted).SingleLine()
		ui.Textf(c, "%.0f%%", s.zoom()*100).FontFeatures("tnum").MinWidth(t.Space(10)).TextAlign(ui.End)
		if ui.Button(c, "Fit").Clicked() {
			s.scale = 0
		}
		if ui.Button(c, "100%").Label("Actual size").Clicked() {
			s.zoomAt(1, s.vw/2, s.vh/2)
		}
	})
}

func (s *State) canvas(c *ui.Context) {
	t := c.Theme()
	radius := t.Space(2)
	view := ui.Box(c).Grow(1).AlignSelf(ui.Stretch).Focusable().Label("Comparison").
		Background(t.Surface).Radius(radius)
	b := view.LayoutBounds()
	gap := t.Space(1)
	pw, step := s.panes(b.W, gap)
	bw, bh := s.pics[0].Size()
	aw, ah := s.pics[1].Size()
	s.layout(float32(max(bw, aw)), float32(max(bh, ah)), pw, b.H, t.Space(5))

	if s.scale > 0 {
		view.Cursor(ui.CursorGrab)
	}
	if dx, dy, ok := view.Dragged(); ok && (dx != 0 || dy != 0) {
		s.pan(dx, dy)
		view.Cursor(ui.CursorGrabbing)
	}
	// Scrolling comes between builds, at the box laid out since, which
	// only a handle reads then.
	ref := ui.Local(view, "compare.handle", func() ui.Handle { return ui.Handle{} })
	view.Bind(ref)
	view.HandleInput(func(ev ui.InputEvent) bool {
		if ev.Kind != ui.InputScroll || ev.DY == 0 {
			return false
		}
		b := ref.Bounds(c)
		pw, step := s.panes(b.W, gap)
		x := ev.X
		if step > 0 && x >= step {
			x -= step
		}
		s.vw, s.vh = pw, b.H
		// Scrolling 300 DIPs zooms by 2, as zoomview does.
		s.zoomAt(s.zoom()/float32(math.Exp2(float64(ev.DY)/300)), x, ev.Y)
		return true
	})
	if s.mode == slider {
		for _, k := range []struct {
			mods ui.Modifiers
			key  ui.Key
			by   float32
		}{{0, ui.KeyLeft, -0.01}, {0, ui.KeyRight, 0.01}, {ui.Shift, ui.KeyLeft, -0.1}, {ui.Shift, ui.KeyRight, 0.1}} {
			if view.Shortcut(k.mods, k.key) {
				s.split = max(0, min(1, s.split+k.by))
			}
		}
		view.Children(func() {
			hit := t.Space(6)
			d := ui.Box(c).Absolute().Left(b.W*s.split-hit/2).Top(0).Size(hit, b.H).Cursor(ui.CursorResizeEW).Label("Divider")
			if dx, _, ok := d.Dragged(); ok && b.W > 0 {
				s.split = max(0, min(1, s.split+dx/b.W))
			}
		})
	}

	z, cx, cy := s.zoom(), s.x, s.y
	mode, split, opacity := s.mode, s.split, float32(s.opacity)
	before, after, diff := s.pics[0], s.pics[1], s.diff
	view.Draw(func(p *ui.Painter, r ui.Rect) {
		// place returns where bm shows in pane.
		place := func(pane ui.Rect, bm *ui.Bitmap) ui.Rect {
			w, h := bm.Size()
			return ui.Rect{X: pane.X + pane.W/2 - cx*z, Y: pane.Y + pane.H/2 - cy*z, W: float32(w) * z, H: float32(h) * z}
		}
		// pic draws bm in pane, outlined, tinted unless tint is zero.
		pic := func(pane ui.Rect, bm *ui.Bitmap, tint ui.Color) {
			at := place(pane, bm)
			p.ImageRegion(bm, ui.Rect{}, at, ui.ImageOptions{Tint: tint})
			p.Stroke(at, t.Border, 0, 1)
		}
		// tag names the picture shown at the top left of area, or its top
		// right, cut by area.
		tag := func(area ui.Rect, name string, right bool) {
			span := ui.Span{Text: name, Size: t.FontSize - 1, Weight: 600, Color: t.Text}
			w, h := p.MeasureText(0, span)
			m, px, py := t.Space(3), t.Space(2), t.Space(0.75)
			box := ui.Rect{X: area.X + m, Y: area.Y + m, W: w + 2*px, H: h + 2*py}
			if right {
				box.X = area.X + area.W - m - box.W
			}
			p.Clip(area, 0, func() {
				p.Fill(box, t.Background.Alpha(0.9), box.H/2)
				p.Stroke(box, t.Border, box.H/2, 1)
				p.RichText(box.X+px, box.Y+py, 0, span)
			})
		}
		p.Clip(r, radius, func() {
			switch mode {
			case slider:
				cut := r.X + r.W*split
				left, right := ui.Rect{X: r.X, Y: r.Y, W: cut - r.X, H: r.H}, ui.Rect{X: cut, Y: r.Y, W: r.X + r.W - cut, H: r.H}
				p.Clip(left, 0, func() { pic(r, before, ui.Color{}) })
				p.Clip(right, 0, func() { pic(r, after, ui.Color{}) })
				tag(left, "Before", false)
				tag(right, "After", true)
				p.Line(cut, r.Y, cut, r.Y+r.H, 2, t.Accent)
				k, mid := t.Space(3.5), r.Y+r.H/2
				p.Shadow(ui.Rect{X: cut - k, Y: mid - k, W: 2 * k, H: 2 * k}, k, 0, 1, 4, 0, ui.RGB(0, 0, 0).Alpha(0.25))
				var knob ui.Path
				p.FillPath(knob.Circle(cut, mid, k), t.Accent)
				a := k * 0.3
				p.StrokePolyline([][2]float32{{cut - a, mid - a}, {cut - 2*a, mid}, {cut - a, mid + a}}, 1.5, t.AccentText)
				p.StrokePolyline([][2]float32{{cut + a, mid - a}, {cut + 2*a, mid}, {cut + a, mid + a}}, 1.5, t.AccentText)
			case sideBySide:
				for i, bm := range []*ui.Bitmap{before, after} {
					pane := ui.Rect{X: r.X + float32(i)*step, Y: r.Y, W: pw, H: r.H}
					p.Clip(pane, 0, func() { pic(pane, bm, ui.Color{}) })
					tag(pane, []string{"Before", "After"}[i], false)
				}
				p.Fill(ui.Rect{X: r.X + pw, Y: r.Y, W: gap, H: r.H}, t.Background, 0)
			case onion:
				pic(r, before, ui.Color{})
				pic(r, after, ui.RGB(255, 255, 255).Alpha(opacity))
			case difference:
				pic(r, after, ui.Color{})
				pic(r, before, ui.Color{})
				p.Fill(place(r, diff), t.Surface.Alpha(0.7), 0)
				p.ImageRegion(diff, ui.Rect{}, place(r, diff), ui.ImageOptions{Tint: t.Danger})
			}
		})
	})
}

// thousands returns n with its thousands apart.
func thousands(n int) string {
	s := fmt.Sprint(n)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}
