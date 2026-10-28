// Package maskpaint paints the mask of a picture for inpainting, as image
// models' editors ask where to paint anew: the picture with what is masked
// tinted over it, a brush and an eraser, and in place of the pointer a
// ring as wide as they paint. Dragging paints or erases a stroke with soft
// edges; Invert masks what was not, Clear unmasks everything, and Undo
// takes back the last stroke or either of those. A line above tells how
// much of the picture is masked.
//
//	app.mask = image.NewAlpha(image.Rect(0, 0, w, h)) // once, the picture's size
//	if maskpaint.View(c, &app.painter, app.photo, app.mask).Grow(1).Changed() {
//		... // app.mask is opaque where to paint anew
//	}
//
// With the picture focused, B and E choose the brush and the eraser, [
// and ] make them smaller and larger, I inverts, and Cmd+Z undoes.
package maskpaint

import (
	"image"
	"image/color"
	"math"
	"strconv"

	"github.com/egoist/mygo/ui"
)

// State is what a painter keeps: the tool and its size, the stroke being
// painted, the tint of the mask shown, and the masks before each stroke,
// to undo. Keep it in the app's state, one for each painter.
type State struct {
	eraser int
	size   float64
	down   bool
	last   [2]float32
	undo   [][]uint8
	of     *image.Alpha
	tint   *image.RGBA
	shown  *ui.Bitmap
	color  ui.Color
	stale  bool
	masked string
}

// keep is how many strokes Undo takes back.
const keep = 20

// View creates the tools over the picture pic, with the mask, which is
// pic's size, tinted over it, and paints into mask as the user does.
func View(c *ui.Context, s *State, pic *ui.Bitmap, mask *image.Alpha) ui.Element {
	t := c.Theme()
	iw, ih := pic.Size()
	if s.size == 0 {
		s.size = float64(max(8, min(iw, ih)/10))
	}
	if s.of != mask || s.tint.Rect != mask.Rect || s.color != t.Accent {
		s.of, s.tint, s.color = mask, image.NewRGBA(mask.Rect), t.Accent
		s.shown = nil
		s.stale = true
	}
	changed := false
	snapshot := func() {
		s.undo = append(s.undo, append([]uint8(nil), mask.Pix...))
		if len(s.undo) > keep {
			s.undo = s.undo[1:]
		}
	}
	undo := func() {
		if n := len(s.undo); n > 0 {
			copy(mask.Pix, s.undo[n-1])
			s.undo = s.undo[:n-1]
			s.stale, changed = true, true
		}
	}
	invert := func() {
		snapshot()
		for i, a := range mask.Pix {
			mask.Pix[i] = 255 - a
		}
		s.stale, changed = true, true
	}
	root := ui.Column(c).Gap(t.Space(3))
	root.Children(func() {
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			ui.Segmented(c, &s.eraser, "Brush", "Eraser").Label("Tool")
			ui.Text(c, "Size").TextColor(t.TextMuted)
			ui.Slider(c, &s.size, 2, float64(max(iw, ih))/4).Width(t.Space(36)).Label("Size")
			ui.Text(c, strconv.Itoa(int(s.size))+" px").FontFeatures("tnum").TextColor(t.TextMuted).Width(t.Space(12))
			ui.Spacer(c)
			if ui.Button(c, "Undo").Disabled(len(s.undo) == 0).Clicked() {
				undo()
			}
			if ui.Button(c, "Invert").Clicked() {
				invert()
			}
			if ui.Button(c, "Clear").Clicked() {
				snapshot()
				clear(mask.Pix)
				s.stale, changed = true, true
			}
		})

		canvas := ui.Box(c).Grow(1).MinHeight(t.Space(40)).Radius(t.Space(2)).Background(t.Surface).Clip().
			Focusable().FocusRing(false).AutoFocus().Role(ui.RoleImage).Label("Picture to mask")
		switch {
		case canvas.Shortcut(0, ui.KeyB):
			s.eraser = 0
		case canvas.Shortcut(0, ui.KeyE):
			s.eraser = 1
		case canvas.Shortcut(0, ui.KeyBracketLeft):
			s.size = max(2, s.size/1.25)
		case canvas.Shortcut(0, ui.KeyBracketRight):
			s.size = min(float64(max(iw, ih))/4, s.size*1.25)
		case canvas.Shortcut(0, ui.KeyI):
			invert()
		case canvas.Shortcut(ui.Cmd, ui.KeyZ):
			undo()
		}
		b := canvas.Bounds()
		k := min(b.W/float32(iw), b.H/float32(ih))
		ox, oy := (b.W-float32(iw)*k)/2, (b.H-float32(ih)*k)/2
		lx, ly, over := canvas.PointerPosition()
		px, py := (lx-ox)/k, (ly-oy)/k
		inside := over && px >= 0 && py >= 0 && px < float32(iw) && py < float32(ih)
		if inside {
			canvas.Cursor(ui.CursorNone)
		}
		pressed := canvas.Pressed() && k > 0
		if pressed {
			if !s.down {
				snapshot()
				s.last = [2]float32{px, py}
			}
			stroke(mask, s.last[0], s.last[1], px, py, float32(s.size)/2, s.eraser == 1)
			s.last = [2]float32{px, py}
			s.stale, changed = true, true
		}
		s.down = pressed
		if s.stale {
			s.refresh(mask)
		}
		if s.shown == nil {
			s.shown = ui.NewBitmap(s.tint)
		}

		shown, ring, eraser := s.shown, float32(s.size)/2*k, s.eraser == 1
		canvas.Draw(func(p *ui.Painter, r ui.Rect) {
			k := min(r.W/float32(iw), r.H/float32(ih))
			pr := ui.Rect{X: r.X + (r.W-float32(iw)*k)/2, Y: r.Y + (r.H-float32(ih)*k)/2, W: float32(iw) * k, H: float32(ih) * k}
			p.Image(pic, pr, ui.FillBox)
			p.Image(shown, pr, ui.FillBox)
			if !inside {
				return
			}
			cx, cy := r.X+lx, r.Y+ly
			var outer, inner ui.Path
			p.StrokePath(outer.Circle(cx, cy, ring+1), 1, ui.RGBA(0, 0, 0, 0.5))
			circle := inner.Circle(cx, cy, ring)
			if eraser {
				circle.Dash(0, 4, 3)
			}
			p.StrokePath(circle, 1.5, ui.RGB(255, 255, 255))
		})
		ui.Text(c, s.masked).FontSize(t.FontSize - 1).FontFeatures("tnum").TextColor(t.TextMuted)
	})
	if changed {
		root.SetChanged()
	}
	return root
}

// refresh tints the mask in the accent color for showing it, and tells
// how much of the picture it masks.
func (s *State) refresh(mask *image.Alpha) {
	s.stale = false
	const shade = 0.55
	r, g, b := float32(s.color.R), float32(s.color.G), float32(s.color.B)
	w, h := mask.Rect.Dx(), mask.Rect.Dy()
	sum := 0
	for y := range h {
		row := mask.Pix[y*mask.Stride : y*mask.Stride+w]
		out := s.tint.Pix[y*s.tint.Stride:]
		for x, a := range row {
			sum += int(a)
			// Pixels of an image.RGBA have their alpha multiplied in.
			f := float32(a) / 255 * shade
			out[x*4], out[x*4+1], out[x*4+2], out[x*4+3] = uint8(r*f), uint8(g*f), uint8(b*f), uint8(255*f)
		}
	}
	if s.shown != nil {
		s.shown.Update(s.tint)
	}
	switch share := float64(sum) / float64(255*max(1, w*h)) * 100; {
	case sum == 0:
		s.masked = "Nothing masked"
	case share < 1:
		s.masked = "Under 1% masked"
	default:
		s.masked = strconv.Itoa(int(math.Round(share))) + "% masked"
	}
}

// stroke paints mask along the line from (x0, y0) to (x1, y1) with a
// round brush of radius rad, its edge a pixel soft, or erases it.
func stroke(mask *image.Alpha, x0, y0, x1, y1, rad float32, erase bool) {
	d := float32(math.Hypot(float64(x1-x0), float64(y1-y0)))
	step := max(1, rad/4)
	n := int(d/step) + 1
	for i := range n {
		f := float32(0)
		if n > 1 {
			f = float32(i) / float32(n-1)
		}
		dab(mask, x0+(x1-x0)*f, y0+(y1-y0)*f, rad, erase)
	}
}

// dab paints a disc of radius rad at (cx, cy), keeping what is masked
// more, or erases one, keeping what is masked less.
func dab(mask *image.Alpha, cx, cy, rad float32, erase bool) {
	b := mask.Rect.Intersect(image.Rect(int(cx-rad-1), int(cy-rad-1), int(cx+rad+2), int(cy+rad+2)))
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			dx, dy := float32(x)+0.5-cx, float32(y)+0.5-cy
			cover := rad + 0.5 - float32(math.Sqrt(float64(dx*dx+dy*dy)))
			if cover <= 0 {
				continue
			}
			v := uint8(255 * min(cover, 1))
			old := mask.AlphaAt(x, y).A
			if erase {
				mask.SetAlpha(x, y, color.Alpha{A: min(old, 255-v)})
			} else {
				mask.SetAlpha(x, y, color.Alpha{A: max(old, v)})
			}
		}
	}
}
