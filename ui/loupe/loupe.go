// Package loupe inspects the pixels of an image, as the eyedroppers of
// design tools: the image fitted in the view, and over it a loupe
// following the pointer, the pixels around it magnified with a grid
// between them, the one under the pointer outlined, and its color in hex
// below.
//
//	ev := loupe.View(c, &app.loupe, app.photo).Grow(1)
//	if ev.Picked {
//		app.color = ev.Color
//	}
//
// A click picks the color under the pointer. While the view has the
// focus, the arrows move the loupe a pixel, Enter picks, and Cmd+C copies
// the hex of the pixel under it.
package loupe

import (
	"fmt"
	"image"
	"image/color"

	"github.com/egoist/mygo/ui"
)

const (
	span = 11
	cell = 12
)

// State is what a loupe keeps: the bitmap of its image, and the pixel the
// loupe is on. Keep it in the app's state, one for each view.
type State struct {
	img    image.Image
	bmp    *ui.Bitmap
	px, py int
	shown  bool
}

// Events is a loupe's view, and the color picked this frame, if any.
type Events struct {
	ui.Element
	Picked bool
	Color  ui.Color
}

func (s *State) at(x, y int) ui.Color {
	b := s.img.Bounds()
	if x < 0 || y < 0 || x >= b.Dx() || y >= b.Dy() {
		return ui.Color{}
	}
	r, g, bl, a := color.NRGBAModel.Convert(s.img.At(b.Min.X+x, b.Min.Y+y)).RGBA()
	return ui.Color{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(bl >> 8), A: uint8(a >> 8)}
}

// Hex returns c as #rrggbb.
func Hex(c ui.Color) string { return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B) }

// View creates the view of img. Give it a size, or Grow.
func View(c *ui.Context, s *State, img image.Image) Events {
	t := c.Theme()
	if s.img != img {
		s.img, s.bmp, s.shown = img, ui.NewBitmap(img).Pixelated(), false
		s.px, s.py = img.Bounds().Dx()/2, img.Bounds().Dy()/2
	}
	view := ui.Box(c).Focusable().Label("Image").Cursor(ui.CursorCrosshair).Clip()
	ev := Events{Element: view}
	iw, ih := float32(img.Bounds().Dx()), float32(img.Bounds().Dy())
	bounds := view.Bounds()
	scale := min(bounds.W/iw, bounds.H/ih)
	ox, oy := (bounds.W-iw*scale)/2, (bounds.H-ih*scale)/2

	if x, y, ok := view.PointerPosition(); ok && scale > 0 {
		px, py := int((x-ox)/scale), int((y-oy)/scale)
		if px >= 0 && py >= 0 && px < int(iw) && py < int(ih) {
			s.px, s.py, s.shown = px, py, true
		} else {
			s.shown = false
		}
	} else if !view.Focused() {
		s.shown = false
	}
	pick := func() {
		ev.Picked, ev.Color = true, s.at(s.px, s.py)
	}
	view.HandleInput(func(e ui.InputEvent) bool {
		if e.Kind != ui.InputKeyDown {
			return false
		}
		switch {
		case e.Key == ui.KeyC && e.Mods == ui.Cmd:
			c.WriteClipboard(Hex(s.at(s.px, s.py)))
			return true
		case e.Mods != 0:
			return false
		case e.Key == ui.KeyLeft:
			s.px = max(s.px-1, 0)
		case e.Key == ui.KeyRight:
			s.px = min(s.px+1, int(iw)-1)
		case e.Key == ui.KeyUp:
			s.py = max(s.py-1, 0)
		case e.Key == ui.KeyDown:
			s.py = min(s.py+1, int(ih)-1)
		case e.Key == ui.KeyEnter:
			pick()
		default:
			return false
		}
		s.shown = true
		return true
	})
	if view.Clicked() && s.shown {
		view.Focus()
		pick()
	}

	px, py, shown := s.px, s.py, s.shown
	under := s.at(px, py)
	bmp := s.bmp
	view.Draw(func(p *ui.Painter, r ui.Rect) {
		img := ui.Rect{X: r.X + ox, Y: r.Y + oy, W: iw * scale, H: ih * scale}
		p.ImageRegion(bmp, ui.Rect{}, img, ui.ImageOptions{})
		if !shown {
			return
		}
		side := float32(span * cell)
		cx, cy := img.X+(float32(px)+0.5)*scale, img.Y+(float32(py)+0.5)*scale
		lx, ly := cx+20, cy+20
		if lx+side > r.X+r.W {
			lx = cx - 20 - side
		}
		if ly+side+32 > r.Y+r.H {
			ly = cy - 20 - side - 32
		}
		lens := ui.Rect{X: lx, Y: ly, W: side, H: side}
		p.Shadow(lens, side/2, 0, 6, 18, 0, ui.RGBA(0, 0, 0, 0.3))
		p.Clip(lens, side/2, func() {
			p.Fill(lens, t.Surface, 0)
			// Only the part of the square of pixels inside the image draws.
			x0, y0 := max(px-span/2, 0), max(py-span/2, 0)
			x1, y1 := min(px+span/2+1, int(iw)), min(py+span/2+1, int(ih))
			src := ui.Rect{X: float32(x0), Y: float32(y0), W: float32(x1 - x0), H: float32(y1 - y0)}
			dst := ui.Rect{
				X: lens.X + float32(x0-(px-span/2))*cell, Y: lens.Y + float32(y0-(py-span/2))*cell,
				W: src.W * cell, H: src.H * cell,
			}
			p.ImageRegion(bmp, src, dst, ui.ImageOptions{})
			for k := 1; k < span; k++ {
				o := float32(k) * cell
				p.Line(lens.X+o, lens.Y, lens.X+o, lens.Y+side, 1, ui.RGBA(0, 0, 0, 0.12))
				p.Line(lens.X, lens.Y+o, lens.X+side, lens.Y+o, 1, ui.RGBA(0, 0, 0, 0.12))
			}
			mid := ui.Rect{X: lens.X + span/2*cell, Y: lens.Y + span/2*cell, W: cell, H: cell}
			p.Stroke(mid, ui.RGB(0, 0, 0), 0, 3)
			p.Stroke(mid, ui.RGB(255, 255, 255), 0, 1.5)
		})
		p.Stroke(lens, ui.RGB(255, 255, 255), side/2, 3)
		p.Stroke(lens, ui.RGBA(0, 0, 0, 0.25), side/2, 1)

		label := ui.Span{Text: Hex(under), Size: t.FontSize - 1, Weight: 600, Color: ui.RGB(255, 255, 255), Features: "tnum"}
		w, h := p.MeasureText(0, label)
		pill := ui.Rect{X: lens.X + (side-w-28)/2, Y: lens.Y + side + 8, W: w + 28, H: h + 8}
		p.Fill(pill, ui.RGBA(20, 20, 20, 0.85), pill.H/2)
		p.Fill(ui.Rect{X: pill.X + 6, Y: pill.Y + (pill.H-12)/2, W: 12, H: 12}, under, 3)
		p.Stroke(ui.Rect{X: pill.X + 6, Y: pill.Y + (pill.H-12)/2, W: 12, H: 12}, ui.RGBA(255, 255, 255, 0.6), 3, 1)
		p.RichText(pill.X+22, pill.Y+4, 0, label)
	})
	return ev
}
