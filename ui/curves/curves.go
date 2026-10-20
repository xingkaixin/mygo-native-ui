// Package curves is the tone curve of a photo editor: a curve from the
// input to the output tone over the histogram of a picture, for its RGB
// together or a channel alone. Pressing the graph adds a point there, and
// dragging a point moves it between its neighbors; a point dragged off
// the graph goes when let go. Hovering reads out the tone in and out.
// Changed reports an edit; Apply maps a picture through the curves.
//
//	if curves.View(c, &app.tone, pic, &app.curves).Changed() {
//		app.after = ui.NewBitmap(curves.Apply(pic, app.curves))
//	}
package curves

import (
	"image"
	"image/draw"
	"math"
	"slices"
	"strconv"

	"github.com/egoist/mygo/ui"
)

// Point is a point of a curve: X a tone in and Y the tone out, from 0
// for black to 1 for white.
type Point struct {
	X, Y float64
}

// Curve is the points a curve goes through, by X. With fewer than two it
// leaves tones as they are.
type Curve []Point

// Curves are the curve of the RGB together, applied after those of each
// channel.
type Curves struct {
	RGB, Red, Green, Blue Curve
}

func (cs *Curves) of(ch int) *Curve {
	return [...]*Curve{&cs.RGB, &cs.Red, &cs.Green, &cs.Blue}[ch]
}

// At returns the tone out for the tone x in, on a monotone cubic through
// the points, so that it never overshoots them, flat past the ends.
func (cv Curve) At(x float64) float64 {
	if len(cv) < 2 {
		return x
	}
	n := len(cv)
	if x <= cv[0].X {
		return cv[0].Y
	}
	if x >= cv[n-1].X {
		return cv[n-1].Y
	}
	d := make([]float64, n-1)
	for k := range d {
		d[k] = (cv[k+1].Y - cv[k].Y) / (cv[k+1].X - cv[k].X)
	}
	m := make([]float64, n)
	m[0], m[n-1] = d[0], d[n-2]
	for k := 1; k < n-1; k++ {
		if d[k-1]*d[k] > 0 {
			m[k] = (d[k-1] + d[k]) / 2
		}
	}
	// Fritsch and Carlson's limit on the slopes keeps each piece monotone.
	for k := range d {
		if d[k] == 0 {
			m[k], m[k+1] = 0, 0
			continue
		}
		a, b := m[k]/d[k], m[k+1]/d[k]
		if s := a*a + b*b; s > 9 {
			t := 3 / math.Sqrt(s)
			m[k], m[k+1] = t*a*d[k], t*b*d[k]
		}
	}
	k, _ := slices.BinarySearchFunc(cv, x, func(p Point, x float64) int {
		switch {
		case p.X < x:
			return -1
		case p.X > x:
			return 1
		}
		return 0
	})
	k = max(1, k) - 1
	h := cv[k+1].X - cv[k].X
	t := (x - cv[k].X) / h
	t2, t3 := t*t, t*t*t
	y := (2*t3-3*t2+1)*cv[k].Y + (t3-2*t2+t)*h*m[k] + (-2*t3+3*t2)*cv[k+1].Y + (t3-t2)*h*m[k+1]
	return max(0, min(1, y))
}

func (cv Curve) lut() [256]uint8 {
	var l [256]uint8
	for i := range l {
		l[i] = uint8(math.Round(cv.At(float64(i)/255) * 255))
	}
	return l
}

// Apply returns pic with its tones mapped through cs.
func Apply(pic image.Image, cs Curves) *image.NRGBA {
	rgb := cs.RGB.lut()
	var ch [3][256]uint8
	for c, cv := range []Curve{cs.Red, cs.Green, cs.Blue} {
		l := cv.lut()
		for i := range l {
			ch[c][i] = rgb[l[i]]
		}
	}
	out := image.NewNRGBA(pic.Bounds())
	draw.Draw(out, out.Rect, pic, pic.Bounds().Min, draw.Src)
	for i := 0; i < len(out.Pix); i += 4 {
		out.Pix[i] = ch[0][out.Pix[i]]
		out.Pix[i+1] = ch[1][out.Pix[i+1]]
		out.Pix[i+2] = ch[2][out.Pix[i+2]]
	}
	return out
}

// State is what the curves keep: the channel shown, the histogram of the
// picture, and the point being dragged. Keep it in the app's state, one
// for each picture.
type State struct {
	channel int
	pic     image.Image
	hist    [4][256]float32
	drag    int
	out     bool
	down    bool
	x, y    float32
}

// histogram counts pic's tones, of luma for RGB, scaled so that the
// highest count but the ends, where clipped tones heap up, is 1.
func (s *State) histogram(pic image.Image) {
	s.pic = pic
	var n [4][256]int
	b := pic.Bounds()
	step := max(1, b.Dx()*b.Dy()/250000)
	k := 0
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if k++; k%step != 0 {
				continue
			}
			r, g, bl, _ := pic.At(x, y).RGBA()
			r, g, bl = r>>8, g>>8, bl>>8
			n[0][(r*299+g*587+bl*114)/1000]++
			n[1][r]++
			n[2][g]++
			n[3][bl]++
		}
	}
	for c := range n {
		top := 1
		for _, v := range n[c][1:255] {
			top = max(top, v)
		}
		for i, v := range n[c] {
			s.hist[c][i] = min(1, float32(v)/float32(top))
		}
	}
}

var names = []string{"RGB", "Red", "Green", "Blue"}

func tint(t *ui.Theme, ch int) ui.Color {
	return [...]ui.Color{t.Text, ui.RGB(232, 72, 72), ui.RGB(56, 178, 92), ui.RGB(64, 120, 236)}[ch]
}

// View creates the curves of pic in *cs.
func View(c *ui.Context, s *State, pic image.Image, cs *Curves) ui.Element {
	t := c.Theme()
	if s.pic != pic {
		s.histogram(pic)
	}
	cv := cs.of(s.channel)
	size := float32(256)
	root := ui.Column(c).Gap(t.Space(3)).Width(size)
	root.Children(func() {
		if ui.Segmented(c, &s.channel, names...).Changed() {
			cv = cs.of(s.channel)
		}
		graph := ui.Box(c).Size(size, size).Background(t.Surface).Border(1, t.Border).Cursor(ui.CursorCrosshair).Label("Curve")
		px, py, over := graph.PointerPosition()
		at := func(px, py float32) Point {
			return Point{max(0, min(1, float64(px/size))), max(0, min(1, 1-float64(py/size)))}
		}
		// Pressed ends as the pointer leaves the graph, but the drag goes
		// on until it is let go.
		dx, dy, held := graph.Dragged()
		switch {
		case graph.Pressed() && !s.down:
			s.down, s.x, s.y = true, px, py
			if len(*cv) < 2 {
				*cv = Curve{{0, 0}, {1, 1}}
			}
			if s.drag = s.grab(cv, at(px, py), size); s.drag > 0 {
				root.SetChanged()
			}
		case held && s.drag > 0:
			s.x, s.y = s.x+dx, s.y+dy
			s.move(cv, at(s.x, s.y), s.x, s.y, size)
			root.SetChanged()
		case !held && s.down:
			s.down = false
			if s.out {
				*cv = slices.Delete(*cv, s.drag-1, s.drag)
				root.SetChanged()
			}
			s.drag, s.out = 0, false
		}
		color := tint(t, s.channel)
		hist := s.hist[s.channel]
		pts := *cv
		graph.Draw(func(p *ui.Painter, r ui.Rect) {
			var h ui.Path
			h.MoveTo(r.X, r.Y+r.H)
			for i, v := range hist {
				h.LineTo(r.X+(float32(i)+0.5)*r.W/256, r.Y+r.H-v*r.H*0.9)
			}
			h.LineTo(r.X+r.W, r.Y+r.H).Close()
			p.FillPath(&h, color.Alpha(0.16))
			for k := 1; k < 4; k++ {
				p.Line(r.X+r.W*float32(k)/4, r.Y, r.X+r.W*float32(k)/4, r.Y+r.H, 1, t.Border)
				p.Line(r.X, r.Y+r.H*float32(k)/4, r.X+r.W, r.Y+r.H*float32(k)/4, 1, t.Border)
			}
			p.StrokePolyline([][2]float32{{r.X, r.Y + r.H}, {r.X + r.W, r.Y}}, 1, t.TextMuted.Alpha(0.35))
			line := make([][2]float32, 129)
			shown := pts
			if s.out && s.drag > 0 {
				shown = slices.Delete(slices.Clone(pts), s.drag-1, s.drag)
			}
			for i := range line {
				x := float64(i) / 128
				line[i] = [2]float32{r.X + float32(x)*r.W, r.Y + r.H - float32(shown.At(x))*r.H}
			}
			p.StrokePolyline(line, 2, color)
			for i, pt := range pts {
				if s.out && s.drag == i+1 {
					continue
				}
				box := ui.Rect{X: r.X + float32(pt.X)*r.W - 4, Y: r.Y + r.H - float32(pt.Y)*r.H - 4, W: 8, H: 8}
				if s.drag == i+1 {
					p.Fill(box, color, 1)
				} else {
					p.Fill(box, t.Surface, 1)
				}
				p.Stroke(box, color, 1, 1.5)
			}
		})
		ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
			read := ""
			if over {
				in := at(px, py).X
				read = "Input " + strconv.Itoa(int(math.Round(in*255))) + "   Output " + strconv.Itoa(int(math.Round(pts.At(in)*255)))
			}
			ui.Text(c, read).FontSize(t.FontSize - 1).FontFeatures("tnum").TextColor(t.TextMuted).Grow(1)
			if ui.Button(c, "Reset "+names[s.channel]).Disabled(len(*cv) == 0).Clicked() {
				*cv = nil
				root.SetChanged()
			}
		})
	})
	return root
}

// grab returns the point near pt to drag, plus 1, adding one at pt when
// there is none.
func (s *State) grab(cv *Curve, pt Point, size float32) int {
	reach := 8 / float64(size)
	for i, q := range *cv {
		if math.Abs(q.X-pt.X) <= reach && math.Abs(q.Y-pt.Y) <= reach {
			return i + 1
		}
	}
	i, _ := slices.BinarySearchFunc(*cv, pt.X, func(p Point, x float64) int {
		switch {
		case p.X < x:
			return -1
		case p.X > x:
			return 1
		}
		return 0
	})
	if i < len(*cv) && (*cv)[i].X-pt.X < 0.01 || i > 0 && pt.X-(*cv)[i-1].X < 0.01 {
		return 0
	}
	*cv = slices.Insert(*cv, i, pt)
	return i + 1
}

// move drags the point between its neighbors, marking it to go when the
// pointer at px, py is well off the graph and it is not an end.
func (s *State) move(cv *Curve, pt Point, px, py, size float32) {
	pts := *cv
	i := s.drag - 1
	lo, hi := 0.0, 1.0
	if i > 0 {
		lo = pts[i-1].X + 0.01
	}
	if i < len(pts)-1 {
		hi = pts[i+1].X - 0.01
	}
	pts[i] = Point{max(lo, min(hi, pt.X)), pt.Y}
	off := px < -24 || py < -24 || px > size+24 || py > size+24
	s.out = off && i > 0 && i < len(pts)-1
}
