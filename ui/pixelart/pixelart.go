// Package pixelart is a pixel art editor: the app's picture zoomed with
// sharp pixels, a grid once a pixel is 6 DIPs wide, and a checkerboard
// where it is transparent. The left button paints with the primary color
// and the right button with the secondary, with a pencil, an eraser, a
// flood fill, lines and rectangles, or picks a color up with the
// eyedropper; Mirror paints the other half too, about the vertical
// center. A click on a palette color makes it the primary, a right click
// the secondary, and the color well edits the primary. B, E, G, L, U and
// I choose the tools, M mirrors, X swaps the colors, Cmd+Z undoes a
// stroke and Shift+Cmd+Z redoes it. The wheel zooms around the pointer,
// and dragging with Space held pans. Previews show the picture at its
// size and twice it, and a status line the pixel under the pointer and
// its color. Scale enlarges the picture by whole pixels, to save it.
//
//	pixelart.View(c, &app.editor, app.sprite, app.palette).Grow(1)
//	...
//	png.Encode(f, pixelart.Scale(app.sprite, 8))
package pixelart

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"math"
	"slices"

	"github.com/egoist/mygo/ui"
)

const (
	pencil = iota
	eraser
	fill
	line
	rectangle
	eyedropper
)

var tools = []struct {
	name, hint string
	key        ui.Key
}{
	{"Pencil", "B", ui.KeyB},
	{"Eraser", "E", ui.KeyE},
	{"Fill", "G", ui.KeyG},
	{"Line", "L", ui.KeyL},
	{"Rectangle", "U", ui.KeyU},
	{"Eyedropper", "I", ui.KeyI},
}

const (
	maxZoom = 64
	// margin is the room the fit leaves around the picture, and the least
	// of it that panning keeps in the canvas.
	margin = 16
)

// State is what an editor keeps: the tool and the colors, the zoom, the
// stroke going on, and what undo and redo go back to. The zero State fits
// the picture in the canvas, the first color of the palette the primary
// and transparency the secondary. Keep it in the app's state, one for
// each editor.
type State struct {
	tool               int
	mirror             bool
	primary, secondary ui.Color
	// ready is whether the primary was taken from the palette.
	ready bool
	// zoom is the DIPs a pixel takes, rounded as it shows, and x, y the
	// picture's top left in the canvas; a zoom of 0 fits the picture.
	zoom, x, y float32
	// vw and vh are the canvas's size as of the last frame.
	vw, vh float32
	// space is whether Space is held, and px, py where the pointer
	// panning was last.
	space, panning bool
	px, py         float32
	// stroke is the tool of the stroke going on plus 1, so that the zero
	// State has none; ink is its color, right whether the right button
	// draws it, and from and last its first and latest pixels.
	stroke     int
	ink        color.RGBA
	right      bool
	from, last image.Point
	refocus    bool
	undo, redo [][]byte
	img        *image.RGBA
	bitmap     *ui.Bitmap
	// checker is two checks a pixel, white and clear, tinted as it draws.
	checker *ui.Bitmap
}

func round(v float32) float32 { return float32(math.Round(float64(v))) }

// view returns the DIPs a pixel of a picture w by h takes, a whole
// number, and the picture's top left in the canvas.
func (s *State) view(w, h int) (k, x, y float32) {
	if s.zoom == 0 {
		k = float32(math.Floor(float64(min((s.vw-2*margin)/float32(w), (s.vh-2*margin)/float32(h)))))
		k = max(1, min(maxZoom, k))
		return k, round((s.vw - float32(w)*k) / 2), round((s.vh - float32(h)*k) / 2)
	}
	return round(s.zoom), round(s.x), round(s.y)
}

// hold keeps the zoom and place the fit gives, for the user to change.
func (s *State) hold(w, h int) {
	if s.zoom == 0 {
		s.zoom, s.x, s.y = s.view(w, h)
	}
}

func (s *State) clamp(w, h int) {
	k := round(s.zoom)
	s.x = max(margin-float32(w)*k, min(s.vw-margin, s.x))
	s.y = max(margin-float32(h)*k, min(s.vh-margin, s.y))
}

// zoomAt zooms for a scroll of dy, keeping the pixel at px, py where it
// is.
func (s *State) zoomAt(w, h int, px, py, dy float32) {
	s.hold(w, h)
	k, x, y := s.view(w, h)
	// Scrolling 200 DIPs zooms by 2.
	s.zoom = max(1, min(maxZoom, s.zoom*float32(math.Exp2(float64(-dy)/200))))
	k1 := round(s.zoom)
	s.x, s.y = px-(px-x)*k1/k, py-(py-y)*k1/k
	s.clamp(w, h)
}

func (s *State) pixel(img *image.RGBA, x, y float32) image.Point {
	k, ox, oy := s.view(img.Rect.Dx(), img.Rect.Dy())
	return image.Pt(img.Rect.Min.X+int(math.Floor(float64((x-ox)/k))), img.Rect.Min.Y+int(math.Floor(float64((y-oy)/k))))
}

// input takes the canvas's pointer, wheel and Space as they come, so that
// a stroke gets every pointer sample and not one a frame.
func (s *State) input(img *image.RGBA, ev ui.InputEvent) bool {
	w, h := img.Rect.Dx(), img.Rect.Dy()
	switch ev.Kind {
	case ui.InputKeyDown, ui.InputKeyUp:
		if ev.Key != ui.KeySpace {
			return false
		}
		s.space = ev.Kind == ui.InputKeyDown
	case ui.InputScroll:
		s.zoomAt(w, h, ev.X, ev.Y, ev.DY)
	case ui.InputPointerDown:
		if ev.Button > 1 || s.stroke > 0 || s.panning {
			return false
		}
		s.refocus = true
		if s.space {
			s.hold(w, h)
			s.panning, s.px, s.py = true, ev.X, ev.Y
			return true
		}
		s.begin(img, s.pixel(img, ev.X, ev.Y), ev.Button == 1)
	case ui.InputPointerMove:
		switch {
		case s.panning:
			s.x, s.y = s.x+ev.X-s.px, s.y+ev.Y-s.py
			s.px, s.py = ev.X, ev.Y
			s.clamp(w, h)
		case s.stroke > 0:
			s.drag(img, s.pixel(img, ev.X, ev.Y))
		default:
			return false
		}
	case ui.InputPointerUp:
		s.panning = false
		s.end(img)
	default:
		return false
	}
	return true
}

func (s *State) begin(img *image.RGBA, p image.Point, right bool) {
	s.stroke, s.right, s.from, s.last = s.tool+1, right, p, p
	if s.tool == eyedropper {
		s.drag(img, p)
		return
	}
	s.ink = rgba(s.primary)
	if right {
		s.ink = rgba(s.secondary)
	}
	if s.tool == eraser {
		s.ink = color.RGBA{}
	}
	s.undo = append(s.undo, slices.Clone(img.Pix))
	if s.tool == fill {
		flood(img, p, s.ink)
		if s.mirror {
			flood(img, mirror(img, p), s.ink)
		}
		return
	}
	s.drag(img, p)
}

// drag takes the stroke on to pixel p: the pencil and the eraser along the
// line from the last pixel, so that fast moves leave no gaps, and lines
// and rectangles drawn anew from the first.
func (s *State) drag(img *image.RGBA, p image.Point) {
	plot := func(q image.Point) {
		img.SetRGBA(q.X, q.Y, s.ink)
		if s.mirror {
			q = mirror(img, q)
			img.SetRGBA(q.X, q.Y, s.ink)
		}
	}
	switch s.stroke - 1 {
	case pencil, eraser:
		bresenham(s.last, p, plot)
	case line:
		copy(img.Pix, s.undo[len(s.undo)-1])
		bresenham(s.from, p, plot)
	case rectangle:
		copy(img.Pix, s.undo[len(s.undo)-1])
		a, b := s.from, p
		bresenham(a, image.Pt(b.X, a.Y), plot)
		bresenham(image.Pt(b.X, a.Y), b, plot)
		bresenham(b, image.Pt(a.X, b.Y), plot)
		bresenham(image.Pt(a.X, b.Y), a, plot)
	case eyedropper:
		if p.In(img.Rect) {
			c := toUI(img.RGBAAt(p.X, p.Y))
			if s.right {
				s.secondary = c
			} else {
				s.primary = c
			}
		}
	}
	s.last = p
}

// end ends the stroke, forgetting it when it changed nothing.
func (s *State) end(img *image.RGBA) {
	if s.stroke == 0 {
		return
	}
	if s.stroke-1 != eyedropper {
		if bytes.Equal(s.undo[len(s.undo)-1], img.Pix) {
			s.undo = s.undo[:len(s.undo)-1]
		} else {
			s.redo = nil
		}
	}
	s.stroke = 0
}

func (s *State) restore(img *image.RGBA, from, to *[][]byte) {
	if len(*from) == 0 || s.stroke > 0 {
		return
	}
	*to = append(*to, slices.Clone(img.Pix))
	copy(img.Pix, (*from)[len(*from)-1])
	*from = (*from)[:len(*from)-1]
}

func mirror(img *image.RGBA, p image.Point) image.Point {
	return image.Pt(img.Rect.Min.X+img.Rect.Max.X-1-p.X, p.Y)
}

// bresenham calls plot for each pixel of the line from a to b.
func bresenham(a, b image.Point, plot func(image.Point)) {
	dx, dy := b.X-a.X, a.Y-b.Y
	sx, sy := 1, 1
	if dx < 0 {
		dx, sx = -dx, -1
	}
	if dy > 0 {
		dy, sy = -dy, -1
	}
	err := dx + dy
	for {
		plot(a)
		if a == b {
			return
		}
		e := 2 * err
		if e >= dy {
			err += dy
			a.X += sx
		}
		if e <= dx {
			err += dx
			a.Y += sy
		}
	}
}

// flood paints c over the pixels of p's color that p reaches across
// edges, not corners.
func flood(img *image.RGBA, p image.Point, c color.RGBA) {
	if !p.In(img.Rect) {
		return
	}
	old := img.RGBAAt(p.X, p.Y)
	if old == c {
		return
	}
	stack := []image.Point{p}
	for len(stack) > 0 {
		q := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if !q.In(img.Rect) || img.RGBAAt(q.X, q.Y) != old {
			continue
		}
		img.SetRGBA(q.X, q.Y, c)
		stack = append(stack, image.Pt(q.X+1, q.Y), image.Pt(q.X-1, q.Y), image.Pt(q.X, q.Y+1), image.Pt(q.X, q.Y-1))
	}
}

// toUI and rgba convert between the premultiplied colors of the picture
// and the straight ones of the color well.
func toUI(c color.RGBA) ui.Color {
	n := color.NRGBAModel.Convert(c).(color.NRGBA)
	return ui.Color{R: n.R, G: n.G, B: n.B, A: n.A}
}

func rgba(c ui.Color) color.RGBA {
	return color.RGBAModel.Convert(color.NRGBA{R: c.R, G: c.G, B: c.B, A: c.A}).(color.RGBA)
}

func hex(c ui.Color) string {
	if c.A == 255 {
		return fmt.Sprintf("#%02X%02X%02X", c.R, c.G, c.B)
	}
	return fmt.Sprintf("#%02X%02X%02X%02X", c.R, c.G, c.B, c.A)
}

// Scale returns img k times as large, each pixel a square of k by k, as
// pixel art is saved.
func Scale(img *image.RGBA, k int) *image.RGBA {
	b := img.Rect
	out := image.NewRGBA(image.Rect(0, 0, b.Dx()*k, b.Dy()*k))
	for y := range out.Rect.Dy() {
		for x := range out.Rect.Dx() {
			out.SetRGBA(x, y, img.RGBAAt(b.Min.X+x/k, b.Min.Y+y/k))
		}
	}
	return out
}

// start takes the first color of palette as the primary, and starts over
// for a new picture.
func (s *State) start(img *image.RGBA, palette []color.RGBA) {
	if !s.ready {
		s.primary, s.ready = ui.RGB(0, 0, 0), true
		if len(palette) > 0 {
			s.primary = toUI(palette[0])
		}
	}
	if s.img == img {
		return
	}
	s.img, s.bitmap = img, ui.NewBitmap(img).Pixelated()
	s.undo, s.redo, s.stroke = nil, nil, 0
	w, h := 2*img.Rect.Dx(), 2*img.Rect.Dy()
	checks := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			if (x+y)%2 == 0 {
				checks.SetRGBA(x, y, color.RGBA{255, 255, 255, 255})
			}
		}
	}
	s.checker = ui.NewBitmap(checks).Pixelated()
}

// View creates the editor of img, which it paints into, with the colors
// of palette. Give it a size, or Grow it.
func View(c *ui.Context, s *State, img *image.RGBA, palette []color.RGBA) ui.Element {
	t := c.Theme()
	s.start(img, palette)
	root := ui.Row(c).Gap(t.Space(4)).AlignItems(ui.Stretch)
	for i, tl := range tools {
		if root.Shortcut(0, tl.key) {
			s.tool = i
		}
	}
	if root.Shortcut(0, ui.KeyM) {
		s.mirror = !s.mirror
	}
	if root.Shortcut(0, ui.KeyX) {
		s.primary, s.secondary = s.secondary, s.primary
	}
	if root.Shortcut(ui.Cmd, ui.KeyZ) {
		s.restore(img, &s.undo, &s.redo)
	}
	if root.Shortcut(ui.Cmd|ui.Shift, ui.KeyZ) {
		s.restore(img, &s.redo, &s.undo)
	}
	// The app, the strokes and undo change the pixels in place.
	s.bitmap.Update(img)
	root.Children(func() {
		s.panel(c, palette)
		ui.Column(c).Grow(1).Gap(t.Space(2)).Children(func() { s.canvas(c, img) })
		ui.Column(c).Gap(t.Space(2)).Shrink(0).Children(func() {
			heading(c, "Preview")
			w, h := float32(img.Rect.Dx()), float32(img.Rect.Dy())
			ui.Column(c).Padding(t.Space(3)).Gap(t.Space(1.5)).Radius(t.Space(2)).Background(t.Surface).AlignItems(ui.Center).Children(func() {
				ui.Image(c, s.bitmap).Size(2*w, 2*h)
				caption(c, "2×")
				ui.Box(c).Height(t.Space(1.5))
				ui.Image(c, s.bitmap).Size(w, h)
				caption(c, "1×")
			})
		})
	})
	return root
}

func heading(c *ui.Context, s string) {
	t := c.Theme()
	ui.Text(c, s).FontSize(t.FontSize - 1).FontWeight(600).TextColor(t.TextMuted)
}

func caption(c *ui.Context, s string) {
	t := c.Theme()
	ui.Text(c, s).FontSize(t.FontSize - 2).TextColor(t.TextMuted)
}

// panel shows the tools, the colors and the palette.
func (s *State) panel(c *ui.Context, palette []color.RGBA) {
	t := c.Theme()
	ui.Column(c).Width(t.Space(46)).Shrink(0).Gap(t.Space(5)).Children(func() {
		ui.Column(c).Gap(t.Space(0.5)).Children(func() {
			heading(c, "Tools")
			ui.Box(c).Height(t.Space(1))
			for i, tl := range tools {
				row := ui.ButtonBase(c.Key(i)).Padding(t.Space(1.5), t.Space(2.5)).Radius(t.Radius).KeepFocus().Label(tl.name)
				switch {
				case i == s.tool:
					row.Background(t.Accent.Alpha(0.15)).TextColor(t.Accent)
				case row.Hovered():
					row.Background(t.SurfaceHover)
				}
				if row.Clicked() {
					s.tool = i
				}
				row.Children(func() {
					ui.Text(c, tl.name).Grow(1)
					ui.Text(c, tl.hint).TextColor(t.TextMuted)
				})
			}
			ui.Row(c).Padding(t.Space(1.5), t.Space(2.5)).AlignItems(ui.Center).Children(func() {
				ui.Checkbox(c, &s.mirror, "Mirror").KeepFocus().Grow(1)
				ui.Text(c, "M").TextColor(t.TextMuted)
			})
		})
		ui.Column(c).Gap(t.Space(2)).Children(func() {
			heading(c, "Colors")
			ui.Row(c).Gap(t.Space(3)).AlignItems(ui.End).Children(func() {
				ui.Column(c).Gap(t.Space(1)).AlignItems(ui.Center).Children(func() {
					ui.ColorWell(c, &s.primary).Label("Primary color").KeepFocus()
					caption(c, "Primary")
				})
				ui.Column(c).Gap(t.Space(1)).AlignItems(ui.Center).Children(func() {
					back := s.secondary
					ui.Box(c).Padding(t.Space(1)).Radius(t.Radius).Background(t.Surface).Border(1, t.Border).
						Label("Secondary color").Value(hex(back)).Children(func() {
						ui.Box(c).Size(t.Space(9), t.Space(5)).Radius(t.Space(1)).Clip().Draw(func(p *ui.Painter, r ui.Rect) {
							checkers(p, r, t.Space(1.25))
							p.Fill(r, back, 0)
						})
					})
					caption(c, "Secondary")
				})
				ui.Column(c).Gap(t.Space(1)).AlignItems(ui.Center).Children(func() {
					if ui.Button(c, "⇄").KeepFocus().Label("Swap colors").Clicked() {
						s.primary, s.secondary = s.secondary, s.primary
					}
					caption(c, "X")
				})
			})
		})
		ui.Column(c).Gap(t.Space(2)).Children(func() {
			heading(c, "Palette")
			primary, secondary := rgba(s.primary), rgba(s.secondary)
			ui.Grid(c).Columns(8).Gap(t.Space(1)).Children(func() {
				for i, col := range palette {
					sw := ui.ButtonBase(c.Key(i)).AspectRatio(1).Padding(2).Radius(t.Space(1.5)).KeepFocus().Label(hex(toUI(col)))
					switch col {
					case primary:
						sw.Border(2, t.Accent)
					case secondary:
						sw.Border(2, t.TextMuted)
					}
					if sw.Clicked() {
						s.primary = toUI(col)
					}
					if sw.RightClicked() {
						s.secondary = toUI(col)
					}
					sw.Children(func() {
						ui.Box(c).Fill().Radius(t.Space(1)).Background(toUI(col)).Border(1, t.Text.Alpha(0.12))
					})
				}
			})
		})
	})
}

// checkers fills r with the checks that show through a clear color, as
// mygo's color well has them.
func checkers(p *ui.Painter, r ui.Rect, size float32) {
	p.Fill(r, ui.RGB(255, 255, 255), 0)
	for y, row := r.Y, 0; y < r.Y+r.H; y, row = y+size, row+1 {
		for x, col := r.X, 0; x < r.X+r.W; x, col = x+size, col+1 {
			if (row+col)%2 == 0 {
				p.Fill(ui.Rect{X: x, Y: y, W: min(size, r.X+r.W-x), H: min(size, r.Y+r.H-y)}, ui.RGB(204, 204, 204), 0)
			}
		}
	}
}

// canvas shows the picture to paint on, and the status line below it.
func (s *State) canvas(c *ui.Context, img *image.RGBA) {
	t := c.Theme()
	canvas := ui.Box(c).Grow(1).AlignSelf(ui.Stretch).Focusable().AutoFocus().Label("Canvas").
		Background(t.Surface).Radius(t.Space(2))
	switch {
	case s.panning:
		canvas.Cursor(ui.CursorGrabbing)
	case s.space:
		canvas.Cursor(ui.CursorGrab)
	default:
		canvas.Cursor(ui.CursorCrosshair)
	}
	if s.refocus {
		canvas.Focus()
		s.refocus = false
	}
	if !canvas.Focused() {
		// The key going up goes to what has the focus now.
		s.space = false
	}
	canvas.HandleInput(func(ev ui.InputEvent) bool { return s.input(img, ev) })
	b := canvas.LayoutBounds()
	s.vw, s.vh = b.W, b.H
	w, h := img.Rect.Dx(), img.Rect.Dy()
	k, x, y := s.view(w, h)
	bitmap, checker, mirror := s.bitmap, s.checker, s.mirror
	base := t.Background
	if t.Dark {
		// The dark outlines most sprites have would vanish on the window's
		// own dark.
		base = t.Background.Mix(t.Text, 0.25)
	}
	canvas.Draw(func(p *ui.Painter, r ui.Rect) {
		box := ui.Rect{X: r.X + x, Y: r.Y + y, W: float32(w) * k, H: float32(h) * k}
		hair := 1 / p.Scale()
		// Clip clips the children, not what Draw paints.
		p.Clip(r, t.Space(2), func() {
			p.Shadow(box, 0, 0, 1, 8, 0, ui.RGB(0, 0, 0).Alpha(0.2))
			p.Fill(box, base, 0)
			p.ImageRegion(checker, ui.Rect{}, box, ui.ImageOptions{Tint: t.Text.Alpha(0.07)})
			p.Image(bitmap, box, ui.FillBox)
			if k >= 6 {
				grid := t.Text.Alpha(0.1)
				for i := range w + 1 {
					p.Fill(ui.Rect{X: box.X + float32(i)*k, Y: box.Y, W: hair, H: box.H}, grid, 0)
				}
				for j := range h + 1 {
					p.Fill(ui.Rect{X: box.X, Y: box.Y + float32(j)*k, W: box.W, H: hair}, grid, 0)
				}
			}
			if mirror {
				p.Fill(ui.Rect{X: box.X + box.W/2 - hair, Y: box.Y, W: 2 * hair, H: box.H}, t.Accent, 0)
			}
		})
	})

	mx, my, over := canvas.PointerPosition()
	px, py := int(math.Floor(float64((mx-x)/k))), int(math.Floor(float64((my-y)/k)))
	ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Padding(0, t.Space(1)).Children(func() {
		status := func(s string) {
			ui.Text(c, s).FontSize(t.FontSize - 1).FontFeatures("tnum").TextColor(t.TextMuted)
		}
		if over && px >= 0 && py >= 0 && px < w && py < h {
			col := toUI(img.RGBAAt(img.Rect.Min.X+px, img.Rect.Min.Y+py))
			status(fmt.Sprintf("x %d  y %d", px, py))
			ui.Box(c).Size(t.Space(3), t.Space(3)).Radius(t.Space(0.75)).Clip().Border(1, t.Border).Draw(func(p *ui.Painter, r ui.Rect) {
				checkers(p, r, r.W/2)
				p.Fill(r, col, 0)
			})
			status(hex(col))
		} else {
			status(fmt.Sprintf("%d × %d", w, h))
		}
		ui.Box(c).Grow(1)
		status(fmt.Sprintf("%.0f%%", k*100))
	})
}
