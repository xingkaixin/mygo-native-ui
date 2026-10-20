// Package signature is a pad to sign on with the pointer: ink that thins
// as the pointer speeds up and smooths its path, a line to sign above,
// and Undo and Clear below. Cmd+Z takes back the last stroke too. Changed
// reports a stroke ended, taken back or cleared; Image draws the strokes
// on transparency, to save as a PNG.
//
//	if signature.View(c, &app.pad, &app.strokes).Changed() {
//		app.signed = len(app.strokes) > 0
//	}
//	png.Encode(f, signature.Image(app.strokes, ui.RGB(20, 40, 120), 2))
package signature

import (
	"image"
	"math"
	"time"

	"github.com/egoist/mygo/ui"
)

// Point is a point of a stroke, in DIPs from the pad's top left, with
// how wide the ink is there.
type Point struct {
	X, Y, W float32
}

// Stroke is the points of one stroke, from where the pointer went down to
// where it went up.
type Stroke []Point

// State is what a pad keeps of the stroke going on. Keep it in the app's
// state, one for each pad.
type State struct {
	down bool
	last time.Time
	x, y float32
}

const (
	widest  = 4.2
	thinest = 1.1
)

// View creates the pad for *strokes.
func View(c *ui.Context, s *State, strokes *[]Stroke) ui.Element {
	t := c.Theme()
	root := ui.Column(c).Gap(t.Space(2))
	root.Children(func() {
		pad := ui.Box(c).AlignSelf(ui.Stretch).Height(t.Space(50)).Radius(t.Space(2)).Background(t.Surface).
			Border(1, t.Border).Cursor(ui.CursorCrosshair).Focusable().Label("Signature").Clip()
		if pad.Shortcut(ui.Cmd, ui.KeyZ) && len(*strokes) > 0 {
			*strokes = (*strokes)[:len(*strokes)-1]
			root.SetChanged()
		}
		s.track(c, pad, strokes, root)
		ink := t.Text
		pad.Draw(func(p *ui.Painter, r ui.Rect) {
			y := r.Y + r.H*0.72
			p.Line(r.X+r.W*0.06, y, r.X+r.W*0.94, y, 1, t.Border)
			p.Text(r.X+r.W*0.06, y-t.FontSize*1.6, "×", t.FontSize+2, t.TextMuted)
			if len(*strokes) == 0 {
				w, _ := p.MeasureText(0, ui.Span{Text: "Sign above the line", Size: t.FontSize - 1})
				p.Text(r.X+(r.W-w)/2, y+t.Space(3), "Sign above the line", t.FontSize-1, t.TextMuted)
			}
			for _, st := range *strokes {
				paint(p, st, r.X, r.Y, ink)
			}
		})
		ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
			ui.Box(c).Grow(1)
			undo := ui.Button(c, "Undo").Disabled(len(*strokes) == 0)
			if undo.Clicked() {
				*strokes = (*strokes)[:len(*strokes)-1]
				root.SetChanged()
			}
			if ui.Button(c, "Clear").Disabled(len(*strokes) == 0).Clicked() {
				*strokes = nil
				root.SetChanged()
			}
		})
	})
	return root
}

// track adds the pointer's points to the stroke it draws while it is down
// on pad, its ink narrower the faster it goes.
func (s *State) track(c *ui.Context, pad ui.Element, strokes *[]Stroke, root ui.Element) {
	x, y, _ := pad.PointerPosition()
	now := c.Now()
	// Pressed ends as the pointer leaves the pad, but the stroke goes on
	// until it is let go, the pointer where its moves take it.
	dx, dy, held := pad.Dragged()
	switch {
	case pad.Pressed() && !s.down:
		s.down, s.last, s.x, s.y = true, now, x, y
		*strokes = append(*strokes, Stroke{{x, y, widest * 0.7}})
	case held && s.down:
		s.x, s.y = s.x+dx, s.y+dy
		x, y := s.x, s.y
		st := &(*strokes)[len(*strokes)-1]
		prev := (*st)[len(*st)-1]
		d := float32(math.Hypot(float64(x-prev.X), float64(y-prev.Y)))
		if d < 1 {
			return
		}
		// Frames that come quick, or a test with no time going by, would
		// make the speed run away.
		dt := max(now.Sub(s.last).Seconds(), 1.0/120)
		s.last = now
		target := max(thinest, widest/(1+float32(float64(d)/dt)/500))
		*st = append(*st, Point{x, y, prev.W*0.6 + target*0.4})
	case !held && s.down:
		s.down = false
		root.SetChanged()
	}
}

// sample returns the points along the Catmull-Rom curve through st, one
// every two DIPs or so, their widths between those of st.
func sample(st Stroke) []Point {
	if len(st) < 3 {
		return st
	}
	var out []Point
	for i := range len(st) - 1 {
		p0, p1, p2, p3 := st[max(i-1, 0)], st[i], st[i+1], st[min(i+2, len(st)-1)]
		n := max(1, int(math.Hypot(float64(p2.X-p1.X), float64(p2.Y-p1.Y))/2))
		for k := range n {
			u := float32(k) / float32(n)
			u2, u3 := u*u, u*u*u
			at := func(a, b, c, d float32) float32 {
				return 0.5 * (2*b + (c-a)*u + (2*a-5*b+4*c-d)*u2 + (3*b-a-3*c+d)*u3)
			}
			out = append(out, Point{at(p0.X, p1.X, p2.X, p3.X), at(p0.Y, p1.Y, p2.Y, p3.Y), p1.W + (p2.W-p1.W)*u})
		}
	}
	return append(out, st[len(st)-1])
}

// paint fills the stroke st offset by x and y as a chain of discs joined
// by quads, all wound the same way so that, filled non-zero, where they
// overlap stays ink.
func paint(p *ui.Painter, st Stroke, x, y float32, ink ui.Color) {
	pts := sample(st)
	var path ui.Path
	for i, a := range pts {
		path.Circle(x+a.X, y+a.Y, a.W/2)
		if i == 0 {
			continue
		}
		b := pts[i-1]
		dx, dy := a.X-b.X, a.Y-b.Y
		l := float32(math.Hypot(float64(dx), float64(dy)))
		if l == 0 {
			continue
		}
		nx, ny := -dy/l, dx/l
		q := [4][2]float32{
			{b.X + nx*b.W/2, b.Y + ny*b.W/2},
			{a.X + nx*a.W/2, a.Y + ny*a.W/2},
			{a.X - nx*a.W/2, a.Y - ny*a.W/2},
			{b.X - nx*b.W/2, b.Y - ny*b.W/2},
		}
		// Path.Circle winds with a positive area, y down; so must the quad.
		if (q[1][0]-q[0][0])*(q[2][1]-q[0][1])-(q[2][0]-q[0][0])*(q[1][1]-q[0][1]) < 0 {
			q[1], q[3] = q[3], q[1]
		}
		path.MoveTo(x+q[0][0], y+q[0][1])
		for _, c := range q[1:] {
			path.LineTo(x+c[0], y+c[1])
		}
		path.Close()
	}
	p.FillPath(&path, ink)
}

// Image draws strokes in ink on transparency, at scale pixels a DIP,
// cropped to them with a margin; nil when there are none.
func Image(strokes []Stroke, ink ui.Color, scale float32) *image.RGBA {
	x0, y0 := float32(math.Inf(1)), float32(math.Inf(1))
	x1, y1 := float32(math.Inf(-1)), float32(math.Inf(-1))
	for _, st := range strokes {
		for _, pt := range st {
			x0, y0 = min(x0, pt.X-pt.W), min(y0, pt.Y-pt.W)
			x1, y1 = max(x1, pt.X+pt.W), max(y1, pt.Y+pt.W)
		}
	}
	if x0 > x1 {
		return nil
	}
	const margin = 8
	w, h := int(math.Ceil(float64(x1-x0)))+2*margin, int(math.Ceil(float64(y1-y0)))+2*margin
	// Render paints on an opaque background, so the ink goes black on
	// white and its darkness becomes how opaque it is.
	img := ui.Render(func(c *ui.Context) {
		ui.Box(c).Fill().Background(ui.RGB(255, 255, 255)).Draw(func(p *ui.Painter, r ui.Rect) {
			for _, st := range strokes {
				paint(p, st, r.X-x0+margin, r.Y-y0+margin, ui.RGB(0, 0, 0))
			}
		})
	}, w, h, scale)
	for i := 0; i < len(img.Pix); i += 4 {
		a := 255 - uint32(img.Pix[i+1])
		img.Pix[i+0] = uint8(uint32(ink.R) * a / 255)
		img.Pix[i+1] = uint8(uint32(ink.G) * a / 255)
		img.Pix[i+2] = uint8(uint32(ink.B) * a / 255)
		img.Pix[i+3] = uint8(a)
	}
	return img
}
