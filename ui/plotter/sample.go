package plotter

import (
	"math"
	"slices"
)

// view is the region a graph shows: the point at its middle, the units of
// x and y a DIP spans, and its size in DIPs.
type view struct {
	x, y, ux, uy float64
	w, h         float64
}

func (v view) sx(x float64) float64  { return v.w/2 + (x-v.x)/v.ux }
func (v view) sy(y float64) float64  { return v.h/2 - (y-v.y)/v.uy }
func (v view) gx(sx float64) float64 { return v.x + (sx-v.w/2)*v.ux }
func (v view) gy(sy float64) float64 { return v.y + (v.h/2-sy)*v.uy }

const (
	// tolerance is how far, in DIPs, a curve may stray from the chords
	// drawn for it.
	tolerance = 0.25
	// depth is how many times a piece is split in two at most, and budget
	// how many points a curve takes at most, which a curve bending
	// everywhere, as sin(1/x) near 0, would otherwise go past.
	depth  = 10
	budget = 20000
	// margin is how far past the edges, in DIPs, a curve is drawn.
	margin = 4
	// step is the gap, in DIPs, between the values that roots, extrema
	// and intersections are looked for between.
	step = 2.0
)

// curve is an expression drawn in a view: its points in the graph's box,
// NaN between pieces, or the one point of a dot, and for y = f(x) the
// points of interest and f at every step across.
type curve struct {
	key  key
	form form
	f, g *node
	pts  [][2]float32
	pois []poi
	vals []float64
}

// key is what a curve is drawn for: a view, and the values of the
// constants its expression uses.
type key struct {
	v      view
	consts uint64
}

// poi is a point of interest of a curve, in the graph's units.
type poi struct{ x, y float64 }

// at returns the curve's function at s, which is x for y = f(x) and y for
// x = g(y): it goes in every variable, as the function has only one.
func (cv *curve) at(s float64) float64 {
	vars := [3]float64{s, s, s}
	return cv.f.eval(&vars)
}

// plot draws e in v, with the values of consts.
func plot(e *parsed, consts map[string]float64, v view) *curve {
	cv := &curve{form: e.form, f: bind(e.f, consts)}
	if e.g != nil {
		cv.g = bind(e.g, consts)
	}
	var vars [3]float64
	eval := func(n *node, slot int, s float64) float64 {
		vars[slot] = s
		return n.eval(&vars)
	}
	sm := &sampler{w: v.w, h: v.h}
	switch e.form {
	case formY:
		sm.at = func(s float64) (float64, float64) { return s, v.sy(eval(cv.f, 0, v.gx(s))) }
		sm.run(-margin, v.w+margin, int(math.Ceil((v.w+2*margin)/step)))
		cv.vals = make([]float64, int(v.w/step)+1)
		for j := range cv.vals {
			cv.vals[j] = cv.at(v.gx(float64(j) * step))
		}
		var pois []poi
		for _, x := range zeros(cv.at, cv.vals, v) {
			pois = append(pois, poi{x, 0})
		}
		cv.pois = visible(append(pois, extrema(cv.at, cv.vals, v)...), v)
	case formX:
		sm.at = func(s float64) (float64, float64) { return v.sx(eval(cv.f, 1, v.gy(s))), s }
		sm.run(-margin, v.h+margin, int(math.Ceil((v.h+2*margin)/step)))
	case formPolar:
		sm.at = func(s float64) (float64, float64) {
			r := eval(cv.f, 2, s)
			return v.sx(r * math.Cos(s)), v.sy(r * math.Sin(s))
		}
		// A curve that comes back to where it began after a turn, as a
		// rose, is drawn once; one that doesn't, as a spiral, over six.
		turns := 1
		for k := range 8 {
			th := 0.7 + 0.77*float64(k)
			a, b := eval(cv.f, 2, th), eval(cv.f, 2, th+2*math.Pi)
			if !(math.Abs(a-b) <= 1e-9*(1+math.Abs(a)) || math.IsNaN(a) && math.IsNaN(b)) {
				turns = 6
				break
			}
		}
		sm.run(0, 2*math.Pi*float64(turns), 720*turns)
	case formParam:
		sm.at = func(s float64) (float64, float64) {
			vars[2] = s
			return v.sx(cv.f.eval(&vars)), v.sy(cv.g.eval(&vars))
		}
		sm.run(0, 2*math.Pi, 720)
	case formPoint:
		x, y := v.sx(cv.f.eval(nil)), v.sy(cv.g.eval(nil))
		if finite(x, y) {
			cv.pts = [][2]float32{{float32(x), float32(y)}}
		}
		return cv
	case formImplicit:
		cv.pts = contour(cv.f, v)
		return cv
	}
	cv.pts = clip(sm.out, v)
	return cv
}

// sampler samples a curve, at returning where it is at s in the graph's
// box, NaN where it has no value.
type sampler struct {
	at   func(s float64) (x, y float64)
	w, h float64
	out  [][2]float64
	n    int
}

type sample struct{ s, x, y float64 }

func (sm *sampler) point(s float64) sample {
	sm.n++
	x, y := sm.at(s)
	return sample{s, x, y}
}

// run samples from a to b, in pieces split where the curve bends.
func (sm *sampler) run(a, b float64, pieces int) {
	p := sm.point(a)
	sm.out = append(sm.out, [2]float64{p.x, p.y})
	for i := 1; i <= pieces; i++ {
		q := sm.point(a + (b-a)*float64(i)/float64(pieces))
		sm.piece(p, q, 0)
		p = q
	}
}

// piece adds the points of the curve after p up to q, splitting it in
// two where it strays from the chord, and breaking it where, split as
// far as it goes, it still does: a jump, as of floor, or a pole, as of
// tan, where the middle is not between the ends.
func (sm *sampler) piece(p, q sample, d int) {
	m := sm.point((p.s + q.s) / 2)
	bends := sm.bends(p, m, q)
	if bends && d < depth && sm.n < budget {
		sm.piece(p, m, d+1)
		sm.piece(m, q, d+1)
		return
	}
	if bends && p.ok() && q.ok() && !between(p, m, q) {
		sm.out = append(sm.out, [2]float64{math.NaN(), math.NaN()})
	}
	sm.out = append(sm.out, [2]float64{q.x, q.y})
}

func (sm *sampler) bends(p, m, q sample) bool {
	switch {
	case !p.ok() && !q.ok():
		return m.ok()
	case !p.ok() || !q.ok() || !m.ok():
		return true
	}
	// Past one edge, the curve doesn't show however it bends.
	lo, right, bottom := float64(-margin), sm.w+margin, sm.h+margin
	if p.y < lo && m.y < lo && q.y < lo || p.y > bottom && m.y > bottom && q.y > bottom ||
		p.x < lo && m.x < lo && q.x < lo || p.x > right && m.x > right && q.x > right {
		return false
	}
	dx, dy := m.x-(p.x+q.x)/2, m.y-(p.y+q.y)/2
	return dx*dx+dy*dy > tolerance*tolerance
}

// between reports whether m is well between p and q along the chord, as
// on a continuous curve.
func between(p, m, q sample) bool {
	dx, dy := q.x-p.x, q.y-p.y
	l := dx*dx + dy*dy
	if l < 1e-12 || !m.ok() {
		return false
	}
	t := ((m.x-p.x)*dx + (m.y-p.y)*dy) / l
	return t > 0.2 && t < 0.8
}

func (s sample) ok() bool { return finite(s.x, s.y) }

func finite(x, y float64) bool {
	return !math.IsNaN(x) && !math.IsInf(x, 0) && !math.IsNaN(y) && !math.IsInf(y, 0)
}

// clip cuts the lines between the points of pts to just past v's edges:
// the points of a pole get too large to draw, and a painter that draws a
// line as a mask needs one no larger than the view.
func clip(pts [][2]float64, v view) [][2]float32 {
	x0, y0, x1, y1 := float64(-margin), float64(-margin), v.w+margin, v.h+margin
	nan := float32(math.NaN())
	var out [][2]float32
	last := [2]float32{nan, nan}
	for i := 1; i < len(pts); i++ {
		p, q := pts[i-1], pts[i]
		if !finite(p[0], p[1]) || !finite(q[0], q[1]) {
			continue
		}
		// Liang–Barsky: the stretch t0 to t1 of p to q inside the box.
		t0, t1 := 0.0, 1.0
		dx, dy := q[0]-p[0], q[1]-p[1]
		inside := true
		for _, e := range [4][2]float64{{-dx, p[0] - x0}, {dx, x1 - p[0]}, {-dy, p[1] - y0}, {dy, y1 - p[1]}} {
			switch {
			case e[0] == 0:
				inside = inside && e[1] >= 0
			case e[0] < 0:
				t0 = max(t0, e[1]/e[0])
			default:
				t1 = min(t1, e[1]/e[0])
			}
		}
		if !inside || t0 > t1 {
			continue
		}
		a := [2]float32{float32(p[0] + t0*dx), float32(p[1] + t0*dy)}
		b := [2]float32{float32(p[0] + t1*dx), float32(p[1] + t1*dy)}
		if a != last {
			out = append(out, [2]float32{nan, nan}, a)
		}
		out = append(out, b)
		last = b
	}
	return out
}

// cell is the side, in DIPs, of the squares an implicit curve is traced
// through.
const cell = 4.0

// contour traces where f(x, y) = 0 across v by marching squares: a line
// through each square whose corners f gives both signs, between the
// points on its sides where f is 0, found by bisection. A side across
// which f changes sign by a pole, as 1/x = y has at x = 0, has none.
func contour(f *node, v view) [][2]float32 {
	nx, ny := int(math.Ceil(v.w/cell)), int(math.Ceil(v.h/cell))
	var vars [3]float64
	at := func(sx, sy float64) float64 {
		vars[0], vars[1] = v.gx(sx), v.gy(sy)
		return f.eval(&vars)
	}
	vals := make([]float64, (nx+1)*(ny+1))
	for j := 0; j <= ny; j++ {
		for i := 0; i <= nx; i++ {
			vals[j*(nx+1)+i] = at(float64(i)*cell, float64(j)*cell)
		}
	}
	nan := [2]float32{float32(math.NaN()), float32(math.NaN())}
	// cross returns where f is 0 between corners (i, j) and (k, l).
	cross := func(i, j, k, l int) ([2]float32, bool) {
		a, b := vals[j*(nx+1)+i], vals[l*(nx+1)+k]
		if !(a > 0) == !(b > 0) || math.IsNaN(a) || math.IsNaN(b) {
			return nan, false
		}
		x0, y0 := float64(i)*cell, float64(j)*cell
		x1, y1 := float64(k)*cell, float64(l)*cell
		for range 10 {
			mx, my := (x0+x1)/2, (y0+y1)/2
			if m := at(mx, my); !(m > 0) == !(a > 0) {
				x0, y0 = mx, my
			} else {
				x1, y1 = mx, my
			}
		}
		x, y := (x0+x1)/2, (y0+y1)/2
		if m := at(x, y); !(math.Abs(m) <= max(math.Abs(a), math.Abs(b))) {
			return nan, false
		}
		return [2]float32{float32(x), float32(y)}, true
	}
	var out [][2]float32
	line := func(a, b [2]float32, ok bool) {
		if ok {
			out = append(out, nan, a, b)
		}
	}
	for j := range ny {
		for i := range nx {
			top, okT := cross(i, j, i+1, j)
			right, okR := cross(i+1, j, i+1, j+1)
			bottom, okB := cross(i, j+1, i+1, j+1)
			left, okL := cross(i, j, i, j+1)
			if okT && okR && okB && okL {
				// A saddle: the middle tells which corners its sign joins.
				mid := at((float64(i)+0.5)*cell, (float64(j)+0.5)*cell)
				if (mid > 0) == (vals[j*(nx+1)+i] > 0) {
					line(top, right, true)
					line(bottom, left, true)
				} else {
					line(top, left, true)
					line(right, bottom, true)
				}
				continue
			}
			var ends [][2]float32
			for k, ok := range []bool{okT, okR, okB, okL} {
				if ok {
					ends = append(ends, [][2]float32{top, right, bottom, left}[k])
				}
			}
			if len(ends) == 2 {
				line(ends[0], ends[1], true)
			}
		}
	}
	return out
}

// zeros returns where f is 0 across v, vals its values a step apart:
// where one is 0 between two that are not, and where they change sign,
// bisected, but not where f jumps across 0, as tan does at its poles.
func zeros(f func(float64) float64, vals []float64, v view) []float64 {
	var out []float64
	for j := 0; j+1 < len(vals); j++ {
		a, b := vals[j], vals[j+1]
		if a == 0 && j > 0 && vals[j-1] != 0 && b != 0 {
			out = append(out, v.gx(float64(j)*step))
		}
		if !(a*b < 0) {
			continue
		}
		lo, hi := v.gx(float64(j)*step), v.gx(float64(j+1)*step)
		for range 100 {
			m := (lo + hi) / 2
			if m == lo || m == hi {
				break
			}
			if (f(m) < 0) == (a < 0) {
				lo = m
			} else {
				hi = m
			}
		}
		if x := (lo + hi) / 2; math.Abs(f(x)) <= v.uy/2 {
			out = append(out, x)
		}
	}
	return out
}

// extrema returns the highest and lowest points of f across v, vals its
// values a step apart, found by golden-section search around where vals
// turn; not around a pole, where f goes on past what the values about it
// allow.
func extrema(f func(float64) float64, vals []float64, v view) []poi {
	g := (math.Sqrt(5) - 1) / 2
	var out []poi
	for j := 1; j+1 < len(vals); j++ {
		a, m, b := vals[j-1], vals[j], vals[j+1]
		d1, d2 := m-a, b-m
		if !(d1 > 0 && d2 < 0 || d1 < 0 && d2 > 0) {
			continue
		}
		up := d1 > 0
		lo, hi := v.gx(float64(j-1)*step), v.gx(float64(j+1)*step)
		x1, x2 := hi-g*(hi-lo), lo+g*(hi-lo)
		f1, f2 := f(x1), f(x2)
		for range 100 {
			if x2 <= x1 {
				break
			}
			if (f1 > f2) == up {
				hi, x2, f2 = x2, x1, f1
				x1 = hi - g*(hi-lo)
				f1 = f(x1)
			} else {
				lo, x1, f1 = x1, x2, f2
				x2 = lo + g*(hi-lo)
				f2 = f(x2)
			}
		}
		x := (lo + hi) / 2
		y := f(x)
		if (y >= m) != up {
			x, y = v.gx(float64(j)*step), m
		}
		if math.IsNaN(y) || math.Abs(y-m) > math.Abs(d1)+math.Abs(d2) {
			continue
		}
		out = append(out, poi{x, y})
	}
	return out
}

// meets returns where a and b, both y = f(x), cross across v; none for
// the same curve twice.
func meets(a, b *curve, v view) []poi {
	d := make([]float64, len(a.vals))
	same := 0
	for j := range d {
		if d[j] = a.vals[j] - b.vals[j]; d[j] == 0 {
			same++
		}
	}
	if same > len(d)/8 {
		return nil
	}
	var out []poi
	for _, x := range zeros(func(x float64) float64 { return a.at(x) - b.at(x) }, d, v) {
		out = append(out, poi{x, a.at(x)})
	}
	return visible(out, v)
}

// visible returns the points of pois in v, but one of those nearer each
// other than 2 DIPs; and none where there are so many that their dots
// would run together, as for sin(x) zoomed far out.
func visible(pois []poi, v view) []poi {
	var out []poi
	for _, p := range pois {
		x, y := v.sx(p.x), v.sy(p.y)
		if !finite(x, y) || x < 0 || x > v.w || y < 0 || y > v.h {
			continue
		}
		if slices.ContainsFunc(out, func(q poi) bool { return math.Hypot(v.sx(q.x)-x, v.sy(q.y)-y) < 2 }) {
			continue
		}
		out = append(out, p)
	}
	if len(out) > int(v.w/16) {
		return nil
	}
	return out
}
