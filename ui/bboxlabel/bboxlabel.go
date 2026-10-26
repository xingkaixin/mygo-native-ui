// Package bboxlabel labels the objects in a picture with boxes, as the
// datasets of object detectors are made: a bar of the classes, each with
// its key, its color and how many boxes it has, over the picture with the
// boxes in their classes' colors, each tagged with its class. Dragging
// across the picture draws a box of the class chosen, with lines across
// the picture under the pointer to line it up; a click chooses a box,
// dragging it moves it, and dragging its corners resizes it. The number
// keys choose a class, and give it to the box chosen; Delete removes the
// box chosen and Escape lets it go. Changed reports an edit.
//
//	bboxlabel.View(c, &app.labeler, app.photo, app.classes, &app.boxes).Grow(1)
package bboxlabel

import (
	"fmt"
	"strconv"

	"github.com/egoist/mygo/ui"
)

// Class is a kind of object: its name and the color of its boxes.
type Class struct {
	Name  string
	Color ui.Color
}

// Box is an object in the picture: its edges, in the picture's pixels,
// X0 and Y0 its top left, and the index of its class.
type Box struct {
	X0, Y0, X1, Y1 float32
	Class          int
}

type mode uint8

const (
	idle mode = iota
	drawing
	moving
	resizing
)

// State is what a labeler keeps: the class chosen, the box chosen, and
// what a drag is doing. Keep it in the app's state, one for each labeler.
type State struct {
	class int
	// chosen is the index of the box chosen plus 1, so that the zero State
	// has none.
	chosen int
	mode   mode
	corner int
	from   [2]float32
	was    Box
	down   bool
}

var digits = []ui.Key{ui.Key1, ui.Key2, ui.Key3, ui.Key4, ui.Key5, ui.Key6, ui.Key7, ui.Key8, ui.Key9}

// View creates the labeler of the boxes *boxes in pic.
func View(c *ui.Context, s *State, pic *ui.Bitmap, classes []Class, boxes *[]Box) ui.Element {
	t := c.Theme()
	iw, ih := pic.Size()
	root := ui.Column(c).Gap(t.Space(3))
	changed := false
	if s.chosen > len(*boxes) {
		s.chosen = 0
	}
	root.Children(func() {
		ui.Row(c).Gap(t.Space(2)).Wrap().Role(ui.RoleRadioGroup).Label("Classes").Children(func() {
			for i, cl := range classes {
				n := 0
				for _, b := range *boxes {
					if b.Class == i {
						n++
					}
				}
				chip := ui.ButtonBase(c.Key(i)).Gap(t.Space(2)).Padding(t.Space(1), t.Space(2.5)).Radius(t.Space(1.5)).
					Border(1, t.Border).Role(ui.RoleRadio).Checked(i == s.class).Label(cl.Name)
				if i == s.class {
					chip.BorderColor(cl.Color).Background(cl.Color.Alpha(0.12))
				} else if chip.Hovered() {
					chip.Background(t.SurfaceHover)
				}
				if chip.Clicked() {
					s.class = i
				}
				chip.Children(func() {
					if i < len(digits) {
						ui.Text(c, strconv.Itoa(i+1)).FontSize(t.FontSize - 2).FontFeatures("tnum").TextColor(t.TextMuted)
					}
					ui.Box(c).Size(t.Space(2.5), t.Space(2.5)).Radius(t.Space(0.5)).Background(cl.Color)
					ui.Text(c, cl.Name)
					ui.Text(c, strconv.Itoa(n)).FontSize(t.FontSize - 2).FontFeatures("tnum").TextColor(t.TextMuted)
				})
			}
		})

		canvas := ui.Box(c).Grow(1).MinHeight(t.Space(40)).Radius(t.Space(2)).Background(t.Surface).Clip().
			Focusable().FocusRing(false).AutoFocus().Cursor(ui.CursorCrosshair).Role(ui.RoleImage).
			Label(fmt.Sprintf("Picture, %d boxes", len(*boxes)))
		b := canvas.Bounds()
		k := min(b.W/float32(iw), b.H/float32(ih))
		ox, oy := (b.W-float32(iw)*k)/2, (b.H-float32(ih)*k)/2
		lx, ly, over := canvas.PointerPosition()
		px, py := (lx-ox)/k, (ly-oy)/k
		clamp := func(x, y float32) (float32, float32) {
			return max(0, min(float32(iw), x)), max(0, min(float32(ih), y))
		}
		pressed := canvas.Pressed() && k > 0
		if pressed && !s.down {
			s.mode, s.from = drawing, [2]float32{px, py}
			if s.chosen > 0 {
				if i := corner((*boxes)[s.chosen-1], px, py, 6/k); i >= 0 {
					s.mode, s.corner = resizing, i
				}
			}
			if s.mode == drawing {
				s.chosen = 0
				for i := len(*boxes) - 1; i >= 0; i-- {
					bx := (*boxes)[i]
					if px >= bx.X0 && px <= bx.X1 && py >= bx.Y0 && py <= bx.Y1 {
						s.chosen, s.mode = i+1, moving
						break
					}
				}
			}
			if s.chosen > 0 {
				s.was = (*boxes)[s.chosen-1]
			}
		}
		if _, _, dragged := canvas.Dragged(); pressed && dragged {
			cx, cy := clamp(px, py)
			switch s.mode {
			case drawing:
				fx, fy := clamp(s.from[0], s.from[1])
				nb := Box{min(fx, cx), min(fy, cy), max(fx, cx), max(fy, cy), s.class}
				if s.chosen == 0 {
					*boxes = append(*boxes, nb)
					s.chosen = len(*boxes)
				}
				(*boxes)[s.chosen-1] = nb
			case moving:
				w, h := s.was.X1-s.was.X0, s.was.Y1-s.was.Y0
				x0 := max(0, min(float32(iw)-w, s.was.X0+px-s.from[0]))
				y0 := max(0, min(float32(ih)-h, s.was.Y0+py-s.from[1]))
				(*boxes)[s.chosen-1] = Box{x0, y0, x0 + w, y0 + h, s.was.Class}
			case resizing:
				nb := s.was
				ax, ay := [2]float32{nb.X1, nb.X0}[s.corner%2], [2]float32{nb.Y1, nb.Y0}[s.corner/2]
				nb.X0, nb.X1 = min(ax, cx), max(ax, cx)
				nb.Y0, nb.Y1 = min(ay, cy), max(ay, cy)
				(*boxes)[s.chosen-1] = nb
			}
			changed = true
		}
		if !pressed && s.down && s.mode == drawing && s.chosen > 0 {
			if nb := (*boxes)[s.chosen-1]; (nb.X1-nb.X0)*k < 4 || (nb.Y1-nb.Y0)*k < 4 {
				*boxes = append((*boxes)[:s.chosen-1], (*boxes)[s.chosen:]...)
				s.chosen = 0
			}
		}
		if !pressed {
			s.mode = idle
		}
		s.down = pressed

		for i, d := range digits[:min(len(classes), len(digits))] {
			if canvas.Shortcut(0, d) {
				s.class = i
				if s.chosen > 0 {
					(*boxes)[s.chosen-1].Class = i
					changed = true
				}
			}
		}
		switch {
		case s.chosen > 0 && (canvas.Shortcut(0, ui.KeyDelete) || canvas.Shortcut(0, ui.KeyBackspace)):
			*boxes = append((*boxes)[:s.chosen-1], (*boxes)[s.chosen:]...)
			s.chosen = 0
			changed = true
		case canvas.Shortcut(0, ui.KeyEscape):
			s.chosen = 0
		}

		shown := *boxes
		chosen := s.chosen
		guides := over && s.mode != moving && s.mode != resizing
		canvas.Draw(func(p *ui.Painter, r ui.Rect) {
			k := min(r.W/float32(iw), r.H/float32(ih))
			pr := ui.Rect{X: r.X + (r.W-float32(iw)*k)/2, Y: r.Y + (r.H-float32(ih)*k)/2, W: float32(iw) * k, H: float32(ih) * k}
			p.Image(pic, pr, ui.FillBox)
			at := func(bx Box) ui.Rect {
				return ui.Rect{X: pr.X + bx.X0*k, Y: pr.Y + bx.Y0*k, W: (bx.X1 - bx.X0) * k, H: (bx.Y1 - bx.Y0) * k}
			}
			for i, bx := range shown {
				col := t.Accent
				name := ""
				if bx.Class >= 0 && bx.Class < len(classes) {
					col, name = classes[bx.Class].Color, classes[bx.Class].Name
				}
				br := at(bx)
				fill := float32(0.12)
				if i+1 == chosen {
					fill = 0.25
				}
				p.Fill(br, col.Alpha(fill), 0)
				p.Stroke(br, col, 0, 2)
				size := t.FontSize - 2
				w, h := p.MeasureText(0, ui.Span{Text: name, Size: size, Weight: 600})
				tag := ui.Rect{X: br.X, Y: br.Y - h - 4, W: w + 8, H: h + 4}
				if tag.Y < pr.Y {
					tag.Y = br.Y
				}
				p.Fill(tag, col, 0)
				p.RichText(tag.X+4, tag.Y+2, 0, ui.Span{Text: name, Size: size, Weight: 600, Color: ink(col)})
			}
			if chosen > 0 && chosen <= len(shown) {
				br := at(shown[chosen-1])
				for i := range 4 {
					x, y := br.X+br.W*float32(i%2), br.Y+br.H*float32(i/2)
					h := ui.Rect{X: x - 4, Y: y - 4, W: 8, H: 8}
					p.Fill(h, ui.RGB(255, 255, 255), 1)
					p.Stroke(h, ui.RGB(0, 0, 0).Alpha(0.6), 1, 1)
				}
			}
			if guides {
				gx, gy := r.X+lx, r.Y+ly
				if gx >= pr.X && gx <= pr.X+pr.W && gy >= pr.Y && gy <= pr.Y+pr.H {
					line := ui.RGB(255, 255, 255).Alpha(0.7)
					p.Line(gx, pr.Y, gx, pr.Y+pr.H, 1, line)
					p.Line(pr.X, gy, pr.X+pr.W, gy, 1, line)
				}
			}
		})
	})
	if changed {
		root.SetChanged()
	}
	return root
}

// corner returns which corner of b is within d of x, y: 0 top left, 1 top
// right, 2 bottom left, 3 bottom right, or -1 for none.
func corner(b Box, x, y, d float32) int {
	for i := range 4 {
		cx, cy := [2]float32{b.X0, b.X1}[i%2], [2]float32{b.Y0, b.Y1}[i/2]
		if x >= cx-d && x <= cx+d && y >= cy-d && y <= cy+d {
			return i
		}
	}
	return -1
}

// ink returns black or white, whichever reads on bg.
func ink(bg ui.Color) ui.Color {
	if 0.299*float32(bg.R)+0.587*float32(bg.G)+0.114*float32(bg.B) > 160 {
		return ui.RGB(0, 0, 0)
	}
	return ui.RGB(255, 255, 255)
}
