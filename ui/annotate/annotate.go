// Package annotate marks up a picture, as screenshot tools do: boxes and
// arrows to point at things, text to say what they are, and blur to hide
// what should not show, each in a color of the toolbar. Select moves the
// mark under the pointer, and Delete removes the one chosen; Cmd+Z undoes
// and Cmd+Shift+Z redoes. Export draws the picture with its marks at its
// own size, to save or share.
//
//	annotate.View(c, &app.markup, app.shot, &app.marks).Grow(1)
//	...
//	png.Encode(f, annotate.Export(app.shot, app.marks))
package annotate

import (
	"image"
	"math"
	"slices"

	"github.com/egoist/mygo/ui"
)

// Kind is what a mark is.
type Kind uint8

const (
	Box Kind = iota
	Arrow
	Text
	Blur
)

// Mark is a mark on a picture, in its pixels: a box or blur from X0, Y0 to
// X1, Y1, an arrow from X0, Y0 pointing at X1, Y1, or text whose top left
// is at X0, Y0.
type Mark struct {
	Kind           Kind
	X0, Y0, X1, Y1 float32
	Color          ui.Color
	Text           string
}

var colors = []ui.Color{
	ui.RGB(255, 59, 48), ui.RGB(255, 149, 0), ui.RGB(255, 204, 0),
	ui.RGB(52, 199, 89), ui.RGB(0, 122, 255), ui.RGB(255, 255, 255),
}

var tools = []string{"Select", "Box", "Arrow", "Text", "Blur"}

// selecting is the tool that chooses and moves marks; each other draws
// the kind of mark of its index in kinds.
const selecting = 0

var kinds = []Kind{1: Box, 2: Arrow, 3: Text, 4: Blur}

// State is what an editor keeps: the tool and color, the mark chosen and
// the one being drawn or typed, what undo and redo go back to, and the
// picture as it draws it. Keep it in the app's state, one for each
// editor.
type State struct {
	tool, color int
	// chosen, drawing and editing are indexes of marks plus 1, so that the
	// zero State has none.
	chosen, drawing, editing int
	// focus gives the text being typed the focus, and refocus the canvas
	// once it is typed.
	focus, refocus, moved bool
	// down is whether the canvas was pressed as of the last frame.
	down       bool
	undo, redo [][]Mark
	pic        image.Image
	bitmap     *ui.Bitmap
	blurs      map[image.Rectangle]*ui.Bitmap
}

// sizes returns the width of lines and the size of text on a picture of
// size, so that marks look the same on small and large ones.
func sizes(size image.Point) (stroke, text float32) {
	long := float32(max(size.X, size.Y))
	return max(3, long/250), max(18, long/32)
}

func (m Mark) rect() (x, y, w, h float32) {
	return min(m.X0, m.X1), min(m.Y0, m.Y1), abs(m.X1 - m.X0), abs(m.Y1 - m.Y0)
}

func abs(v float32) float32 { return float32(math.Abs(float64(v))) }

// blur returns the blocks the picture shows under r, a block of about
// twelve pixels each, sampled rather than averaged, as it is redone while
// a blur is drawn.
func blur(pic image.Image, r image.Rectangle) *ui.Bitmap {
	r = r.Intersect(pic.Bounds())
	if r.Empty() {
		return nil
	}
	const block = 12
	w, h := (r.Dx()+block-1)/block, (r.Dy()+block-1)/block
	small := image.NewRGBA(image.Rect(0, 0, w, h))
	for by := range h {
		for bx := range w {
			var sr, sg, sb, n uint32
			for sy := range 3 {
				for sx := range 3 {
					x := min(r.Min.X+bx*block+sx*block/3+block/6, r.Max.X-1)
					y := min(r.Min.Y+by*block+sy*block/3+block/6, r.Max.Y-1)
					cr, cg, cb, _ := pic.At(x, y).RGBA()
					sr, sg, sb, n = sr+cr, sg+cg, sb+cb, n+1
				}
			}
			i := small.PixOffset(bx, by)
			small.Pix[i], small.Pix[i+1], small.Pix[i+2], small.Pix[i+3] = uint8(sr/n>>8), uint8(sg/n>>8), uint8(sb/n>>8), 255
		}
	}
	return ui.NewBitmap(small).Pixelated()
}

func (m Mark) area() image.Rectangle {
	x, y, w, h := m.rect()
	return image.Rect(int(x), int(y), int(x+w), int(y+h))
}

// paint draws the picture and marks, all but skip, with the picture's top
// left at x, y and k pixels of the painter a pixel of it.
func paint(p *ui.Painter, pic *ui.Bitmap, size image.Point, marks []Mark, skip int, x, y, k float32, blurOf func(Mark) *ui.Bitmap) {
	p.Image(pic, ui.Rect{X: x, Y: y, W: float32(size.X) * k, H: float32(size.Y) * k}, ui.FillBox)
	stroke, text := sizes(size)
	at := func(px, py float32) (float32, float32) { return x + px*k, y + py*k }
	for i, m := range marks {
		if i == skip {
			continue
		}
		switch m.Kind {
		case Box:
			rx, ry, rw, rh := m.rect()
			x0, y0 := at(rx, ry)
			p.Stroke(ui.Rect{X: x0, Y: y0, W: rw * k, H: rh * k}, m.Color, stroke*k, stroke*k)
		case Blur:
			if b := blurOf(m); b != nil {
				rx, ry, rw, rh := m.rect()
				x0, y0 := at(rx, ry)
				p.ImageRegion(b, ui.Rect{}, ui.Rect{X: x0, Y: y0, W: rw * k, H: rh * k}, ui.ImageOptions{})
			}
		case Arrow:
			x0, y0 := at(m.X0, m.Y0)
			x1, y1 := at(m.X1, m.Y1)
			dx, dy := x1-x0, y1-y0
			l := float32(math.Hypot(float64(dx), float64(dy)))
			if l == 0 {
				continue
			}
			ux, uy := dx/l, dy/l
			head := min(stroke*k*4.5, l*0.6)
			bx, by := x1-ux*head, y1-uy*head
			p.StrokePolyline([][2]float32{{x0, y0}, {bx + ux*head*0.3, by + uy*head*0.3}}, stroke*k, m.Color)
			var tip ui.Path
			w := head * 0.6
			tip.MoveTo(x1, y1).LineTo(bx-uy*w, by+ux*w).LineTo(bx+uy*w, by-ux*w).Close()
			p.FillPath(&tip, m.Color)
		case Text:
			x0, y0 := at(m.X0, m.Y0)
			span := ui.Span{Text: m.Text, Size: text * k, Weight: 700, Color: m.Color}
			// A dark shadow keeps light text readable on light pictures,
			// and the other way round.
			shade := ui.RGB(0, 0, 0).Alpha(0.45)
			if int(m.Color.R)+int(m.Color.G)+int(m.Color.B) < 300 {
				shade = ui.RGB(255, 255, 255).Alpha(0.6)
			}
			back := span
			back.Color = shade
			p.RichText(x0+k, y0+k*1.5, 0, back)
			p.RichText(x0, y0, 0, span)
		}
	}
}

// Export returns pic with marks drawn on it, at its size.
func Export(pic image.Image, marks []Mark) *image.RGBA {
	size := pic.Bounds().Size()
	bitmap := ui.NewBitmap(pic)
	return ui.Render(func(c *ui.Context) {
		ui.Box(c).Fill().Draw(func(p *ui.Painter, r ui.Rect) {
			paint(p, bitmap, size, marks, -1, r.X, r.Y, 1, func(m Mark) *ui.Bitmap { return blur(pic, m.area()) })
		})
	}, size.X, size.Y, 1)
}

func (s *State) save(marks []Mark) {
	s.undo = append(s.undo, slices.Clone(marks))
	s.redo = nil
}

func (s *State) restore(marks *[]Mark, from, to *[][]Mark) {
	if len(*from) == 0 {
		return
	}
	*to = append(*to, slices.Clone(*marks))
	*marks = (*from)[len(*from)-1]
	*from = (*from)[:len(*from)-1]
	s.chosen, s.drawing, s.editing = 0, 0, 0
}

// finish ends typing in a text mark, which goes when it is empty, with
// the change it was.
func (s *State) finish(marks *[]Mark) {
	if s.editing == 0 {
		return
	}
	i := s.editing - 1
	s.editing = 0
	if i < len(*marks) && (*marks)[i].Text == "" {
		*marks = slices.Delete(*marks, i, i+1)
		s.undo = s.undo[:len(s.undo)-1]
		s.chosen = 0
	}
}

// View creates the editor of marks on pic.
func View(c *ui.Context, s *State, pic image.Image, marks *[]Mark) ui.Element {
	t := c.Theme()
	if s.pic != pic {
		s.pic, s.bitmap = pic, ui.NewBitmap(pic)
		s.blurs = nil
	}
	size := pic.Bounds().Size()
	stroke, textSize := sizes(size)
	margin := t.Space(4)
	root := ui.Column(c).Gap(t.Space(3))
	if root.Shortcut(ui.Cmd, ui.KeyZ) {
		s.finish(marks)
		s.restore(marks, &s.undo, &s.redo)
	}
	if root.Shortcut(ui.Cmd|ui.Shift, ui.KeyZ) {
		s.restore(marks, &s.redo, &s.undo)
	}
	root.Children(func() {
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			if ui.Segmented(c, &s.tool, tools...).Changed() {
				s.finish(marks)
			}
			ui.Row(c).Gap(t.Space(1.5)).Children(func() {
				for i, col := range colors {
					b := ui.ButtonBase(c.Key(i)).Size(t.Space(6), t.Space(6)).Radius(t.Space(3)).Background(col).
						Border(1, t.Border).Label("Color")
					if i == s.color {
						b.Border(2.5, t.Accent)
					}
					if b.Clicked() {
						s.color = i
						if s.chosen > 0 {
							s.save(*marks)
							(*marks)[s.chosen-1].Color = col
						}
					}
				}
			})
			ui.Box(c).Grow(1)
			if ui.Button(c, "Delete").Disabled(s.chosen == 0).Clicked() {
				s.remove(marks)
			}
			if ui.Button(c, "Undo").Disabled(len(s.undo) == 0).Clicked() {
				s.finish(marks)
				s.restore(marks, &s.undo, &s.redo)
			}
			if ui.Button(c, "Redo").Disabled(len(s.redo) == 0).Clicked() {
				s.restore(marks, &s.redo, &s.undo)
			}
		})
		canvas := ui.Box(c).Grow(1).AlignSelf(ui.Stretch).Clip().Focusable().Label("Picture").
			Background(t.Surface).Radius(t.Space(2))
		if s.tool != selecting {
			canvas.Cursor(ui.CursorCrosshair)
		}
		if s.refocus {
			canvas.Focus()
			s.refocus = false
		}
		r := canvas.LayoutBounds()
		r.W, r.H = r.W-2*margin, r.H-2*margin
		k := min(r.W/float32(size.X), r.H/float32(size.Y))
		ox, oy := margin+(r.W-float32(size.X)*k)/2, margin+(r.H-float32(size.Y)*k)/2
		measure := func(m Mark) (float32, float32) {
			return c.MeasureText(0, ui.Span{Text: m.Text, Size: textSize, Weight: 700})
		}
		if k > 0 {
			s.input(c, canvas, marks, k, ox, oy, stroke, measure)
		}
		used := map[image.Rectangle]*ui.Bitmap{}
		blurOf := func(m Mark) *ui.Bitmap {
			a := m.area()
			b, ok := used[a]
			if !ok {
				if b, ok = s.blurs[a]; !ok {
					b = blur(pic, a)
				}
				used[a] = b
			}
			return b
		}
		ms := *marks
		chosen, editing := s.chosen-1, s.editing-1
		canvas.Draw(func(p *ui.Painter, r ui.Rect) {
			x, y := r.X+ox, r.Y+oy
			p.Shadow(ui.Rect{X: x, Y: y, W: float32(size.X) * k, H: float32(size.Y) * k}, 0, 0, 2, 12, 0, ui.RGB(0, 0, 0).Alpha(0.25))
			paint(p, s.bitmap, size, ms, editing, x, y, k, blurOf)
			if chosen >= 0 && chosen < len(ms) && chosen != editing {
				m := ms[chosen]
				bx, by, bw, bh := m.rect()
				if m.Kind == Text {
					bw, bh = measure(m)
				}
				pad := stroke * k * 2
				p.StrokeDashed(ui.Rect{X: x + bx*k - pad, Y: y + by*k - pad, W: bw*k + 2*pad, H: bh*k + 2*pad}, t.Accent, 2, 1.5)
			}
			s.blurs = used
		})
		canvas.Children(func() {
			if s.editing == 0 || s.editing > len(*marks) {
				return
			}
			m := &(*marks)[s.editing-1]
			in := ui.TextInputBase(c, &m.Text).Absolute().Left(ox + m.X0*k).Top(oy + m.Y0*k).MinWidth(t.Space(30)).
				FontSize(textSize * k).FontWeight(700).TextColor(m.Color).Label("Text")
			if s.focus {
				in.Focus()
				s.focus = false
			} else if !in.Focused() {
				s.finish(marks)
			}
			in.OnTextKey(func(key ui.TextKey) bool {
				if (key.Key == ui.KeyEnter || key.Key == ui.KeyEscape) && key.Mods == 0 {
					s.finish(marks)
					s.refocus = true
					return true
				}
				return false
			})
		})
	})
	return root
}

func (s *State) remove(marks *[]Mark) {
	if s.chosen == 0 || s.chosen > len(*marks) {
		return
	}
	s.save(*marks)
	*marks = slices.Delete(*marks, s.chosen-1, s.chosen)
	s.chosen = 0
}

// input does what the pointer and keys do on the canvas, whose picture is
// at ox, oy, k DIPs a pixel of it.
func (s *State) input(c *ui.Context, canvas ui.Element, marks *[]Mark, k, ox, oy, stroke float32, measure func(Mark) (float32, float32)) {
	if s.editing == 0 && (canvas.Shortcut(0, ui.KeyBackspace) || canvas.Shortcut(0, ui.KeyDelete)) {
		s.remove(marks)
	}
	start := canvas.Pressed() && !s.down
	// Pressed ends as the pointer leaves the canvas, but the drag goes on
	// until it is let go: coming back is no new press.
	_, _, s.down = canvas.Dragged()
	if start {
		canvas.Focus()
		s.finish(marks)
		px, py, _ := canvas.PointerPosition()
		x, y := (px-ox)/k, (py-oy)/k
		if s.tool == selecting {
			s.chosen, s.moved = s.hit(*marks, x, y, max(8/k, stroke*2), measure)+1, false
		} else {
			s.save(*marks)
			*marks = append(*marks, Mark{Kind: kinds[s.tool], X0: x, Y0: y, X1: x, Y1: y, Color: colors[s.color]})
			s.chosen = len(*marks)
			if kinds[s.tool] == Text {
				s.editing, s.focus = len(*marks), true
			} else {
				s.drawing = len(*marks)
			}
		}
	}
	if dx, dy, ok := canvas.Dragged(); ok {
		dx, dy = dx/k, dy/k
		switch {
		case s.drawing > 0:
			m := &(*marks)[s.drawing-1]
			m.X1, m.Y1 = m.X1+dx, m.Y1+dy
		case s.tool == selecting && s.chosen > 0:
			if !s.moved {
				s.save(*marks)
				s.moved = true
			}
			m := &(*marks)[s.chosen-1]
			m.X0, m.Y0, m.X1, m.Y1 = m.X0+dx, m.Y0+dy, m.X1+dx, m.Y1+dy
		}
	}
	_, _, ended := canvas.DragEnded()
	if (ended || canvas.Clicked()) && s.drawing > 0 {
		i := s.drawing - 1
		s.drawing = 0
		_, _, w, h := (*marks)[i].rect()
		if max(w, h) < 4/k {
			// A click, not a mark.
			*marks = slices.Delete(*marks, i, i+1)
			s.undo = s.undo[:len(s.undo)-1]
			s.chosen = 0
		}
	}
}

// hit returns the index of the topmost mark within tol of x, y, -1 for
// none.
func (s *State) hit(marks []Mark, x, y, tol float32, measure func(Mark) (float32, float32)) int {
	for i := len(marks) - 1; i >= 0; i-- {
		m := marks[i]
		switch m.Kind {
		case Box, Blur:
			rx, ry, rw, rh := m.rect()
			if x >= rx-tol && x <= rx+rw+tol && y >= ry-tol && y <= ry+rh+tol {
				return i
			}
		case Text:
			w, h := measure(m)
			if x >= m.X0-tol && x <= m.X0+w+tol && y >= m.Y0-tol && y <= m.Y0+h+tol {
				return i
			}
		case Arrow:
			dx, dy := m.X1-m.X0, m.Y1-m.Y0
			l2 := dx*dx + dy*dy
			u := float32(0)
			if l2 > 0 {
				u = max(0, min(1, ((x-m.X0)*dx+(y-m.Y0)*dy)/l2))
			}
			if math.Hypot(float64(x-m.X0-u*dx), float64(y-m.Y0-u*dy)) <= float64(tol) {
				return i
			}
		}
	}
	return -1
}
