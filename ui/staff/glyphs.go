package staff

import (
	"math"

	"github.com/egoist/mygo/ui"
)

// pen adds to a path in staff spaces, s DIPs each, from an origin at x,
// y, y down.
type pen struct {
	p       *ui.Path
	x, y, s float32
}

func (q pen) at(x, y float32) (float32, float32) { return q.x + x*q.s, q.y + y*q.s }

func (q pen) move(x, y float32) { q.p.MoveTo(q.at(x, y)) }
func (q pen) line(x, y float32) { q.p.LineTo(q.at(x, y)) }

func (q pen) cube(x1, y1, x2, y2, x, y float32) {
	ax, ay := q.at(x1, y1)
	bx, by := q.at(x2, y2)
	cx, cy := q.at(x, y)
	q.p.CubeTo(ax, ay, bx, by, cx, cy)
}

func (q pen) circle(x, y, r float32) {
	cx, cy := q.at(x, y)
	q.p.Circle(cx, cy, r*q.s)
}

// quad adds the four-sided shape through the points.
func (q pen) quad(x0, y0, x1, y1, x2, y2, x3, y3 float32) {
	q.move(x0, y0)
	q.line(x1, y1)
	q.line(x2, y2)
	q.line(x3, y3)
	q.p.Close()
}

// ellipse adds an ellipse at cx, cy, rx by ry, turned a radians
// clockwise; a hole goes round the other way, so that it cuts through a
// shape it is in.
func (q pen) ellipse(cx, cy, rx, ry, a float32, hole bool) {
	const k = 0.5523
	sin, cos := float32(math.Sin(float64(a))), float32(math.Cos(float64(a)))
	pt := func(u, v float32) (float32, float32) {
		if hole {
			v = -v
		}
		return cx + rx*u*cos - ry*v*sin, cy + rx*u*sin + ry*v*cos
	}
	q.move(pt(1, 0))
	arcs := [4][3][2]float32{
		{{1, k}, {k, 1}, {0, 1}},
		{{-k, 1}, {-1, k}, {-1, 0}},
		{{-1, -k}, {-k, -1}, {0, -1}},
		{{k, -1}, {1, -k}, {1, 0}},
	}
	for _, c := range arcs {
		x1, y1 := pt(c[0][0], c[0][1])
		x2, y2 := pt(c[1][0], c[1][1])
		x3, y3 := pt(c[2][0], c[2][1])
		q.cube(x1, y1, x2, y2, x3, y3)
	}
	q.p.Close()
}

// nib is a point of a stroke and how wide the stroke is there.
type nib struct{ x, y, w float32 }

// stroke adds a stroke of a broad pen along a smooth curve through pts,
// as wide at each as its w.
func (q pen) stroke(pts ...nib) {
	var c []nib
	for i := 0; i+1 < len(pts); i++ {
		p0, p1, p2, p3 := pts[max(i-1, 0)], pts[i], pts[i+1], pts[min(i+2, len(pts)-1)]
		for k := range 12 {
			t := float32(k) / 12
			cr := func(a, b, c, d float32) float32 {
				return 0.5 * (2*b + (c-a)*t + (2*a-5*b+4*c-d)*t*t + (3*b-a-3*c+d)*t*t*t)
			}
			c = append(c, nib{cr(p0.x, p1.x, p2.x, p3.x), cr(p0.y, p1.y, p2.y, p3.y), p1.w + (p2.w-p1.w)*t})
		}
	}
	c = append(c, pts[len(pts)-1])
	side := make([][2]float32, 2*len(c))
	for i, n := range c {
		a, b := c[max(i-1, 0)], c[min(i+1, len(c)-1)]
		dx, dy := b.x-a.x, b.y-a.y
		l := float32(math.Hypot(float64(dx), float64(dy)))
		if l == 0 {
			l = 1
		}
		// The outline turns the way Path.Circle does, so that where a
		// stroke runs into a ball the two add up rather than cancel.
		nx, ny := dy/l*n.w/2, -dx/l*n.w/2
		side[i] = [2]float32{n.x + nx, n.y + ny}
		side[len(side)-1-i] = [2]float32{n.x - nx, n.y - ny}
	}
	q.move(side[0][0], side[0][1])
	for _, pt := range side[1:] {
		q.line(pt[0], pt[1])
	}
	q.p.Close()
}

// The glyphs below are drawn in staff spaces from where the music puts
// them.

// trebleClef draws a G clef with its curl on y, the G line.
func trebleClef(q pen) {
	q.circle(0.96, 1.82, 0.43)
	q.stroke(
		nib{1.02, 2.12, 0.16}, nib{1.32, 2.46, 0.19}, nib{1.8, 2.3, 0.2}, nib{1.98, 1.85, 0.19},
		nib{1.93, 1.2, 0.19}, nib{1.72, 0.2, 0.19}, nib{1.52, -1.0, 0.19}, nib{1.38, -2.1, 0.19},
		nib{1.3, -3.05, 0.18}, nib{1.42, -3.9, 0.17}, nib{1.83, -4.36, 0.16}, nib{2.22, -4.0, 0.24},
		nib{2.27, -3.35, 0.32}, nib{2.08, -2.72, 0.44}, nib{1.7, -2.25, 0.54}, nib{1.2, -1.85, 0.6},
		nib{0.68, -1.3, 0.62}, nib{0.4, -0.62, 0.68}, nib{0.36, 0.05, 0.64}, nib{0.55, 0.55, 0.45},
		nib{0.95, 0.84, 0.26}, nib{1.45, 0.88, 0.2}, nib{2.0, 0.66, 0.3}, nib{2.42, 0.15, 0.38}, nib{2.42, -0.45, 0.34},
		nib{2.05, -0.88, 0.24}, nib{1.55, -0.98, 0.2}, nib{1.12, -0.75, 0.19}, nib{0.9, -0.25, 0.18},
		nib{1.0, 0.18, 0.15}, nib{1.36, 0.38, 0.1},
	)
}

// bassClef draws an F clef with its ball and dots around y, the F line.
func bassClef(q pen) {
	q.circle(0.5, 0, 0.5)
	q.stroke(
		nib{0.4, -0.38, 0.14}, nib{0.62, -0.78, 0.2}, nib{1.05, -0.98, 0.22}, nib{1.55, -0.82, 0.3},
		nib{1.86, -0.35, 0.44}, nib{1.9, 0.25, 0.52}, nib{1.7, 0.95, 0.48}, nib{1.25, 1.62, 0.36},
		nib{0.7, 2.05, 0.2}, nib{0.17, 2.28, 0.05},
	)
	q.circle(2.52, -0.5, 0.2)
	q.circle(2.52, 0.5, 0.2)
}

// accWidth is how wide each accidental is.
var accWidth = map[accidental]float32{sharp: 1.0, flat: 0.85, natural: 0.72, doubleSharp: 1.0, doubleFlat: 1.6}

// drawAccidental draws a with its left at x and its note's line or space
// on y.
func drawAccidental(q pen, a accidental) {
	switch a {
	case sharp:
		q.quad(0.22, -1.15, 0.34, -1.2, 0.34, 1.38, 0.22, 1.42)
		q.quad(0.64, -1.4, 0.76, -1.44, 0.76, 1.12, 0.64, 1.16)
		for _, yc := range []float32{-0.48, 0.48} {
			q.quad(0, yc-0.05, 1.0, yc-0.35, 1.0, yc+0.05, 0, yc+0.35)
		}
	case flat:
		flatGlyph(q, 0)
	case doubleFlat:
		flatGlyph(q, 0)
		flatGlyph(q, 0.78)
	case natural:
		q.quad(0, -1.35, 0.12, -1.35, 0.12, 0.62, 0, 0.66)
		q.quad(0.58, -0.66, 0.7, -0.7, 0.7, 1.35, 0.58, 1.35)
		for _, yc := range []float32{-0.35, 0.35} {
			q.quad(0, yc-0.08, 0.7, yc-0.3, 0.7, yc+0.08, 0, yc+0.3)
		}
	case doubleSharp:
		q.stroke(nib{0.2, -0.3, 0.12}, nib{0.8, 0.3, 0.12})
		q.stroke(nib{0.2, 0.3, 0.12}, nib{0.8, -0.3, 0.12})
		for _, c := range [][2]float32{{0.15, -0.35}, {0.85, -0.35}, {0.15, 0.35}, {0.85, 0.35}} {
			q.quad(c[0]-0.15, c[1]-0.15, c[0]+0.15, c[1]-0.15, c[0]+0.15, c[1]+0.15, c[0]-0.15, c[1]+0.15)
		}
	}
}

func flatGlyph(q pen, x float32) {
	q.quad(x, -1.75, x+0.12, -1.75, x+0.12, 0.55, x, 0.55)
	q.move(x+0.1, -0.15)
	q.cube(x+0.35, -0.6, x+0.95, -0.62, x+0.88, -0.12)
	q.cube(x+0.84, 0.15, x+0.45, 0.42, x+0.1, 0.58)
	q.line(x+0.1, 0.36)
	q.cube(x+0.4, 0.18, x+0.66, -0.02, x+0.63, -0.18)
	q.cube(x+0.6, -0.38, x+0.32, -0.3, x+0.1, -0.06)
	q.p.Close()
}

// head sizes: a black or a half note's head is hw wide, a whole note's
// ww.
const (
	hw = 1.18
	ww = 1.64
)

// noteHead draws the head of a note of kind, its left at x.
func noteHead(q pen, kind int) {
	switch {
	case kind == 1:
		q.ellipse(ww/2, 0, 0.82, 0.5, 0, false)
		q.ellipse(ww/2, 0, 0.42, 0.27, -1.0, true)
	case kind == 2:
		q.ellipse(hw/2, 0, 0.6, 0.42, -0.35, false)
		q.ellipse(hw/2, 0, 0.47, 0.18, -0.55, true)
	default:
		q.ellipse(hw/2, 0, 0.6, 0.42, -0.35, false)
	}
}

// restWidth is how wide the rest of kind is.
func restWidth(kind int) float32 {
	switch kind {
	case 1, 2:
		return 1.2
	case 4, 8:
		return 1.05
	}
	return 1.05 + 0.25*float32(beams(kind)-1)
}

// rest draws a rest of kind, its left at x and y on the middle line.
func rest(q pen, kind int) {
	switch kind {
	case 1:
		q.quad(0, -1, 1.2, -1, 1.2, -0.5, 0, -0.5)
	case 2:
		q.quad(0, -0.5, 1.2, -0.5, 1.2, 0, 0, 0)
	case 4:
		q.stroke(nib{0.26, -1.5, 0.1}, nib{0.86, -0.64, 0.38})
		q.stroke(nib{0.88, -0.7, 0.3}, nib{0.6, -0.26, 0.48}, nib{0.3, 0.16, 0.34})
		q.stroke(nib{0.3, 0.1, 0.16}, nib{0.86, 0.86, 0.14})
		q.stroke(nib{0.88, 0.9, 0.08}, nib{0.42, 0.78, 0.3}, nib{0.18, 1.06, 0.32}, nib{0.32, 1.42, 0.2}, nib{0.6, 1.58, 0.06})
	default:
		n := beams(kind)
		top := -0.55 - float32((n-1)/2)
		bottom := top + 1.55 + float32(n-1)
		const lean = 0.32
		x := func(y float32) float32 { return 0.3*float32(n) + 0.68 - (y-top)*lean }
		q.quad(x(top)-0.06, top, x(top)+0.08, top, x(bottom)+0.06, bottom, x(bottom)-0.08, bottom)
		for i := range n {
			y := top + float32(i)
			r := x(y)
			q.circle(r-0.65, y+0.1, 0.24)
			q.stroke(nib{r - 0.62, y + 0.3, 0.14}, nib{r - 0.32, y + 0.3, 0.12}, nib{r, y, 0.08})
		}
	}
}

// beams returns how many beams or flags a note of kind has.
func beams(kind int) int {
	n := 0
	for k := kind; k > 4; k /= 2 {
		n++
	}
	return n
}

// flag draws the n-th flag of a stem, from the stem at x and its tip at
// y, down from an up stem and up from a down one.
func flag(q pen, n int, up bool) {
	d := float32(1)
	if !up {
		d = -1
	}
	o := 0.8 * float32(n) * d
	q.stroke(nib{0, o, 0.3}, nib{0.12, o + 0.55*d, 0.4}, nib{0.6, o + 1.25*d, 0.34}, nib{0.95, o + 2.0*d, 0.2},
		nib{0.85, o + 2.6*d, 0.1}, nib{0.62, o + 3.05*d, 0.03})
}

// commonTime draws C, its left at x and its middle on y; a line through
// it makes it cut time.
func commonTime(q pen, cut bool) {
	var pts []nib
	for i := range 13 {
		a := 0.85 + float64(i)/12*(2*math.Pi-1.7)
		w := 0.12 + 0.3*float32(math.Pow(math.Sin(a/2), 6))
		pts = append(pts, nib{0.95 + float32(math.Cos(a)), -float32(math.Sin(a)), w})
	}
	q.stroke(pts...)
	q.circle(1.42, -0.66, 0.2)
	if cut {
		q.quad(0.88, -1.6, 1.0, -1.6, 1.0, 1.6, 0.88, 1.6)
	}
}

// tie draws a tie from x0 to x1 on y, bowing by h, down when h is above
// 0, th thick in the middle.
func tie(q pen, x0, x1, y, h, th float32) {
	k := (x1 - x0) / 4
	if h < 0 {
		th = -th
	}
	q.move(x0, y)
	q.cube(x0+k, y+h*4/3, x1-k, y+h*4/3, x1, y)
	q.cube(x1-k, y+(h-th)*4/3, x0+k, y+(h-th)*4/3, x0, y)
	q.p.Close()
}
