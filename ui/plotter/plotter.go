// Package plotter graphs expressions as they are typed, as Desmos and
// macOS's Grapher do: y = f(x), or f(x) alone; x = g(y); lines such as
// x = 3 and y = 1; r = f(θ) in polar coordinates; points (a, b) and
// parametric curves (f(t), g(t)); equations of x and y, as
// x^2 + y^2 = 4; and constants, as a = 2, which the others use. Each
// curve is drawn in its color over a grid that follows the zoom, finer
// where it bends and broken where it jumps or has no value, as tan(x)
// and sqrt(x) do. The roots and the highest and lowest points of each
// y = f(x), and where two of them cross, are gray dots.
//
// The pointer near a curve traces it, reading out the coordinates of the
// point in the curve's color, and on a gray dot reads out the dot's.
// Dragging pans, and the wheel zooms around the pointer: with Shift held
// along x alone, and with Option (Alt) along y alone. Errors tells what
// is wrong with each expression, to show by its field:
//
//	exprs := []plotter.Expr{{Source: "y = sin(x)", Color: red}, {Source: "r = 1 + cos(θ)", Color: blue}}
//	plotter.View(c, &app.graph, exprs).Grow(1)
//	errs := plotter.Errors(exprs) // nil for each that draws
//	...
//	app.graph.Reset() // back to the origin, x and y to one scale
package plotter

import (
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
)

// Expr is an expression to graph, as typed, and the color of its curve.
//
// It is written as on a calculator: numbers, + - * / and ^, which groups
// to the right, parentheses, and products without *, as 2x, 3sin(x),
// (x+1)(x-1) and 2pi; the functions sin, cos, tan, asin, acos, atan,
// sinh, cosh, tanh, sqrt, cbrt, abs, ln, log (of base 10), exp, floor,
// ceil, round, sign, min and max, their values in parentheses; and pi or
// π, and e. A run of letters is read as the names it begins with, else a
// letter at a time: ax is a times x. A constant is a letter other than x,
// y, r, t and θ, which theta spells too. Polar curves go round once from
// θ = 0, or six times when they don't come back to where they began, as
// a spiral; parametric curves go from t = 0 to 2π.
type Expr struct {
	Source string
	Color  ui.Color
}

// State is what a graph keeps: the region it shows, and the curves drawn
// for it. Keep it in the app's state, one for each graph. The zero State
// shows the origin at the middle, x and y to one scale, 10 units from it
// to the nearer edges.
type State struct {
	// x, y is the point at the middle, and ux, uy the units of x and y a
	// DIP spans; 0 for those of the zero State.
	x, y, ux, uy float64
	exprs        map[string]*parsed
	curves       map[string]*curve
	// meets are where the curves of meetsOf, y = f(x), cross.
	meets   []poi
	meetsOf []*curve
}

// Reset shows the region the zero State does, from the next frame on.
func (s *State) Reset() { s.x, s.y, s.ux, s.uy = 0, 0, 0, 0 }

func (s *State) clamp() {
	s.x, s.y = max(-1e12, min(1e12, s.x)), max(-1e12, min(1e12, s.y))
	// Units smaller than this are lost in the digits of where they are.
	least := max(1e-10, 1e-12*max(math.Abs(s.x), math.Abs(s.y)))
	s.ux, s.uy = max(least, min(1e10, s.ux)), max(least, min(1e10, s.uy))
}

// zoom scales the units of x by fx and of y by fy, keeping the point at
// px, py in a box of w by h DIPs where it is.
func (s *State) zoom(fx, fy, px, py, w, h float64) {
	gx, gy := s.x+(px-w/2)*s.ux, s.y+(h/2-py)*s.uy
	s.ux, s.uy = s.ux*fx, s.uy*fy
	s.clamp()
	s.x, s.y = gx-(px-w/2)*s.ux, gy-(h/2-py)*s.uy
	s.clamp()
}

// mark is the point the pointer traces: on a curve, in its color, or a
// point of interest.
type mark struct {
	x, y  float64
	color ui.Color
	poi   bool
}

// View creates the graph of exprs. Give it a size, or grow it in its
// parent.
func View(c *ui.Context, s *State, exprs []Expr) ui.Element {
	t := c.Theme()
	box := ui.Box(c).Role(ui.RoleImage).Label("Graph")
	b := box.LayoutBounds()
	w, h := float64(b.W), float64(b.H)
	if s.ux == 0 && w > 0 && h > 0 {
		s.ux = 20 / min(w, h)
		s.uy = s.ux
	}

	if dx, dy, ok := box.Dragged(); ok && s.ux != 0 {
		s.x -= float64(dx) * s.ux
		s.y += float64(dy) * s.uy
		s.clamp()
	}
	if box.Pressed() {
		box.Cursor(ui.CursorGrabbing)
	}
	// Scrolling comes between builds, at the box laid out since, which
	// only a handle reads then.
	handle := ui.Local(box, "plotter.handle", func() ui.Handle { return ui.Handle{} })
	box.Bind(handle)
	box.HandleInput(func(ev ui.InputEvent) bool {
		r := handle.Bounds(c)
		if ev.Kind != ui.InputScroll || s.ux == 0 || r.W <= 0 || r.H <= 0 {
			return false
		}
		w, h, px, py := float64(r.W), float64(r.H), float64(ev.X), float64(ev.Y)
		dx, dy := float64(ev.DX), float64(ev.DY)
		// Scrolling 300 DIPs zooms by 2, as candles does.
		switch {
		case ev.Mods&ui.Shift != 0:
			// The window may have turned the wheel sideways, as for
			// scroll containers.
			if dy == 0 {
				dy = dx
			}
			s.zoom(math.Exp2(dy/300), 1, px, py, w, h)
		case ev.Mods&ui.Alt != 0:
			s.zoom(1, math.Exp2(dy/300), px, py, w, h)
		case math.Abs(dx) > math.Abs(dy):
			s.x += dx * s.ux
			s.clamp()
		default:
			f := math.Exp2(dy / 300)
			s.zoom(f, f, px, py, w, h)
		}
		return true
	})

	parsedNow := make(map[string]*parsed, len(exprs))
	ps := make([]*parsed, len(exprs))
	for i, e := range exprs {
		p, ok := s.exprs[e.Source]
		if !ok {
			p = parse(e.Source)
		}
		ps[i], parsedNow[e.Source] = p, p
	}
	s.exprs = parsedNow
	consts, errs := resolve(ps)

	v := view{s.x, s.y, s.ux, s.uy, w, h}
	curves := make([]*curve, len(exprs))
	curvesNow := make(map[string]*curve, len(exprs))
	var fs []*curve
	changed := false
	for i, e := range ps {
		if errs[i] != nil || e.form == formNone || e.form == formConst || v.ux == 0 || w <= 0 || h <= 0 {
			continue
		}
		k := key{v, hash(e.uses, consts)}
		cv := s.curves[exprs[i].Source]
		if cv == nil || cv.key != k {
			cv = plot(e, consts, v)
			cv.key = k
			changed = true
		}
		curves[i], curvesNow[exprs[i].Source] = cv, cv
		if cv.form == formY {
			fs = append(fs, cv)
		}
	}
	s.curves = curvesNow
	if changed || !slices.Equal(fs, s.meetsOf) {
		s.meets, s.meetsOf = nil, fs
		for i, a := range fs {
			for _, b := range fs[i+1:] {
				s.meets = append(s.meets, meets(a, b, v)...)
			}
		}
	}
	pois := slices.Clone(s.meets)
	for _, cv := range fs {
		pois = append(pois, cv.pois...)
	}

	var traced *mark
	if px, py, over := box.PointerPosition(); over && !box.Pressed() && v.ux != 0 {
		traced = trace(float64(px), float64(py), v, curves, exprs, pois)
	}
	if traced != nil {
		box.Value(point(traced, v))
	}

	box.Draw(func(p *ui.Painter, r ui.Rect) {
		if r.W != b.W || r.H != b.H {
			c.Invalidate()
		}
		p.Fill(r, t.Background, 0)
		if v.ux == 0 || w <= 0 || h <= 0 {
			return
		}
		p.Clip(r, 0, func() {
			grid(p, r, v, t)
			for i, cv := range curves {
				if cv == nil || len(cv.pts) == 0 {
					continue
				}
				color := exprs[i].Color
				if cv.form == formPoint {
					dot(p, r.X+cv.pts[0][0], r.Y+cv.pts[0][1], 4.5, color)
					continue
				}
				pts := make([][2]float32, len(cv.pts))
				for j, q := range cv.pts {
					pts[j] = [2]float32{r.X + q[0], r.Y + q[1]}
				}
				p.StrokePolyline(pts, 2.5, color)
			}
			gray := t.TextMuted
			for _, q := range pois {
				x, y := r.X+float32(v.sx(q.x)), r.Y+float32(v.sy(q.y))
				dot(p, x, y, 4.5, t.Background)
				dot(p, x, y, 3, gray)
			}
			if traced == nil {
				return
			}
			x, y := r.X+float32(v.sx(traced.x)), r.Y+float32(v.sy(traced.y))
			color := traced.color
			if traced.poi {
				color = gray
			}
			dot(p, x, y, 6.5, t.Background)
			dot(p, x, y, 4.5, color)
			tag(p, r, x, y, point(traced, v), color, t)
		})
	})
	return box
}

// trace returns the point of interest nearest px, py within a few DIPs,
// or else the point of a curve nearest it within a few more: for y = f(x)
// and x = g(y), the point of the curve at the pointer's x or y.
func trace(px, py float64, v view, curves []*curve, exprs []Expr, pois []poi) *mark {
	var m *mark
	best := 8.0
	for _, q := range pois {
		if d := math.Hypot(v.sx(q.x)-px, v.sy(q.y)-py); d < best {
			best, m = d, &mark{x: q.x, y: q.y, poi: true}
		}
	}
	if m != nil {
		return m
	}
	best = 12.0
	for i, cv := range curves {
		if cv == nil {
			continue
		}
		for j, q := range cv.pts {
			a := q
			if j > 0 && !math.IsNaN(float64(cv.pts[j-1][0])) {
				a = cv.pts[j-1]
			}
			if math.IsNaN(float64(q[0])) || math.IsNaN(float64(a[0])) {
				continue
			}
			x, y := nearest(px, py, a, q)
			if d := math.Hypot(x-px, y-py); d < best {
				best, m = d, &mark{x: v.gx(x), y: v.gy(y), color: exprs[i].Color}
				switch cv.form {
				case formY:
					m.y = cv.at(m.x)
				case formX:
					m.x = cv.at(m.y)
				}
			}
		}
	}
	if m != nil && !finite(m.x, m.y) {
		return nil
	}
	return m
}

// nearest returns the point of the line from a to b nearest x, y.
func nearest(x, y float64, a, b [2]float32) (float64, float64) {
	ax, ay, bx, by := float64(a[0]), float64(a[1]), float64(b[0]), float64(b[1])
	dx, dy := bx-ax, by-ay
	l := dx*dx + dy*dy
	if l == 0 {
		return ax, ay
	}
	f := max(0, min(1, ((x-ax)*dx+(y-ay)*dy)/l))
	return ax + f*dx, ay + f*dy
}

// hash returns a hash of the values of the constants of uses.
func hash(uses []string, consts map[string]float64) uint64 {
	h := uint64(14695981039346656037)
	for _, u := range uses {
		h = (h ^ math.Float64bits(consts[u])) * 1099511628211
	}
	return h
}

// point writes the coordinates of m to the decimals the zoom shows, one
// more for a point of interest: (1.571, 1).
func point(m *mark, v view) string {
	ux, uy := v.ux, v.uy
	if m.poi {
		ux, uy = ux/10, uy/10
	}
	return "(" + coord(m.x, ux) + ", " + coord(m.y, uy) + ")"
}

// coord writes v to the decimals that units u apart tell apart, with a
// power of ten for the very large and the very small: 2.5×10⁻⁷.
func coord(v, u float64) string {
	d := min(15, max(0, int(math.Ceil(-math.Log10(u)))))
	s := trimZeros(strconv.FormatFloat(v, 'f', d, 64))
	if a := math.Abs(v); a >= 1e7 || a < 1e-4 && s != "0" && s != "-0" {
		digits := min(15, max(1, int(math.Floor(math.Log10(a))-math.Floor(math.Log10(u)))+1))
		mant, exp, _ := strings.Cut(strconv.FormatFloat(v, 'e', digits-1, 64), "e")
		e, _ := strconv.Atoi(exp)
		return minus(trimZeros(mant)) + "×10" + superscript(e)
	}
	return minus(s)
}

// label writes n × 10^k, a line of the grid, to as many decimals as k
// needs, with a power of ten for a million and more or less than a
// ten-thousandth: 2×10⁶.
func label(n int64, k int) string {
	neg := n < 0
	digits := strconv.FormatInt(max(n, -n), 10)
	var s string
	switch e := k + len(digits) - 1; {
	case n == 0:
		return "0"
	case e >= 6 || e <= -5:
		mant := trimZeros(digits[:1] + "." + digits[1:])
		s = mant + "×10" + superscript(k+len(digits)-1)
	case k >= 0:
		s = digits + strings.Repeat("0", k)
	default:
		if len(digits) <= -k {
			digits = strings.Repeat("0", -k-len(digits)+1) + digits
		}
		s = trimZeros(digits[:len(digits)+k] + "." + digits[len(digits)+k:])
	}
	if neg {
		s = "−" + s
	}
	return s
}

func trimZeros(s string) string {
	if strings.Contains(s, ".") {
		s = strings.TrimRight(strings.TrimRight(s, "0"), ".")
	}
	return s
}

// minus writes a minus sign for a hyphen, and 0 for -0.
func minus(s string) string {
	if s == "-0" {
		return "0"
	}
	return strings.Replace(s, "-", "−", 1)
}

func superscript(n int) string {
	var b strings.Builder
	for _, r := range strconv.Itoa(n) {
		if r == '-' {
			b.WriteRune('⁻')
		} else {
			b.WriteRune([]rune("⁰¹²³⁴⁵⁶⁷⁸⁹")[r-'0'])
		}
	}
	return b.String()
}

// scale is the grid of an axis: major lines m × 10^k apart, minor lines
// a minor-th of that.
type scale struct {
	m     int64
	k     int
	minor int
}

// gridScale returns the scale of an axis of u units a DIP whose major
// lines are at least least DIPs apart, 1, 2 or 5 times a power of ten.
func gridScale(u, least float64) scale {
	k := int(math.Floor(math.Log10(least * u)))
	for {
		for _, m := range []int64{1, 2, 5} {
			if float64(m)*math.Pow10(k)/u >= least {
				minor := 5
				if m == 2 {
					minor = 4
				}
				return scale{m, k, minor}
			}
		}
		k++
	}
}

func (sc scale) gap() float64 { return float64(sc.m) * math.Pow10(sc.k) / float64(sc.minor) }

// lines returns the first and last minor line from lo to hi.
func (sc scale) lines(lo, hi float64) (int64, int64) {
	g := sc.gap()
	return int64(math.Ceil(lo / g)), int64(math.Floor(hi / g))
}

// label writes the value of minor line i, a major one.
func (sc scale) label(i int64) string { return label(i/int64(sc.minor)*sc.m, sc.k) }

// grid paints the grid of v in r: minor and major lines, the axes, and
// the major lines' values beside the axes, or along the edges the axes
// are past.
func grid(p *ui.Painter, r ui.Rect, v view, t *ui.Theme) {
	size := t.FontSize - 2
	measure := func(s string) (float32, float32) { return p.MeasureText(0, ui.Span{Text: s, Size: size}) }
	x0, x1, y0, y1 := v.gx(0), v.gx(v.w), v.gy(v.h), v.gy(0)

	// The values along x must fit between their lines.
	xs := gridScale(v.ux, 50)
	lw := float32(0)
	for _, x := range []float64{x0, x1} {
		i := int64(math.Round(x/xs.gap())) / int64(xs.minor) * int64(xs.minor)
		w, _ := measure(xs.label(i))
		lw = max(lw, w)
	}
	if float64(lw)+24 > float64(xs.m)*math.Pow10(xs.k)/v.ux {
		xs = gridScale(v.ux, float64(lw)+24)
	}
	ys := xs
	if v.uy != v.ux {
		ys = gridScale(v.uy, 50)
	}

	minor, major := t.Text.Alpha(0.05), t.Text.Alpha(0.13)
	if t.Dark {
		minor, major = t.Text.Alpha(0.07), t.Text.Alpha(0.16)
	}
	i0, i1 := xs.lines(x0, x1)
	for i := i0; i <= i1; i++ {
		x := r.X + float32(v.sx(float64(i)*xs.gap()))
		color := minor
		if i%int64(xs.minor) == 0 {
			color = major
		}
		p.Line(x, r.Y, x, r.Y+r.H, 1, color)
	}
	j0, j1 := ys.lines(y0, y1)
	for j := j0; j <= j1; j++ {
		y := r.Y + float32(v.sy(float64(j)*ys.gap()))
		color := minor
		if j%int64(ys.minor) == 0 {
			color = major
		}
		p.Line(r.X, y, r.X+r.W, y, 1, color)
	}
	ax, ay := float32(v.sx(0)), float32(v.sy(0))
	axis := t.Text.Alpha(0.55)
	showsX, showsY := ay >= 0 && ay <= r.H, ax >= 0 && ax <= r.W
	if showsX {
		p.Line(r.X, r.Y+ay, r.X+r.W, r.Y+ay, 1.5, axis)
	}
	if showsY {
		p.Line(r.X+ax, r.Y, r.X+ax, r.Y+r.H, 1.5, axis)
	}

	gap, pad := t.Space(1.5), float32(2)
	var taken []ui.Rect
	text := func(x, y float32, s string, w, h float32) {
		p.Fill(ui.Rect{X: x - pad, Y: y, W: w + 2*pad, H: h}, t.Background.Alpha(0.85), 3)
		p.Text(x, y, s, size, t.TextMuted)
	}
	origin := showsX && showsY
	for j := j0; j <= j1; j++ {
		if j%int64(ys.minor) != 0 || j == 0 && origin {
			continue
		}
		s := ys.label(j)
		w, h := measure(s)
		x := max(gap, min(r.W-w-gap, ax-gap-w))
		y := float32(v.sy(float64(j)*ys.gap())) - h/2
		if y < pad || y+h > r.H-pad {
			continue
		}
		taken = append(taken, ui.Rect{X: x - gap, Y: y, W: w + 2*gap, H: h})
		text(r.X+x, r.Y+y, s, w, h)
	}
	for i := i0; i <= i1; i++ {
		if i%int64(xs.minor) != 0 || i == 0 && origin {
			continue
		}
		s := xs.label(i)
		w, h := measure(s)
		x := float32(v.sx(float64(i)*xs.gap())) - w/2
		y := max(gap, min(r.H-h-gap, ay+gap))
		hits := func(o ui.Rect) bool { return x < o.X+o.W && o.X < x+w && y < o.Y+o.H && o.Y < y+h }
		if x < pad || x+w > r.W-pad || slices.ContainsFunc(taken, hits) {
			continue
		}
		text(r.X+x, r.Y+y, s, w, h)
	}
	if origin {
		w, h := measure("0")
		text(r.X+ax-gap-w, r.Y+ay+gap, "0", w, h)
	}
}

func dot(p *ui.Painter, x, y, radius float32, c ui.Color) {
	var path ui.Path
	path.Circle(x, y, radius)
	p.FillPath(&path, c)
}

// tag paints s in color on a card beside the point x, y, inside r.
func tag(p *ui.Painter, r ui.Rect, x, y float32, s string, color ui.Color, t *ui.Theme) {
	span := ui.Span{Text: s, Size: t.FontSize - 1, Weight: 600, Color: color}
	tw, th := p.MeasureText(0, span)
	pad, off := t.Space(2), t.Space(2.5)
	card := ui.Rect{W: tw + 2*pad, H: th + pad}
	card.X, card.Y = x+off, y-off-card.H
	if card.X+card.W > r.X+r.W-4 {
		card.X = x - off - card.W
	}
	if card.Y < r.Y+4 {
		card.Y = y + off
	}
	p.Shadow(card, 6, 0, 1, 6, 0, ui.RGBA(0, 0, 0, 0.12))
	p.Fill(card, t.Background, 6)
	p.Stroke(card, t.Border, 6, 1)
	p.RichText(card.X+pad, card.Y+pad/2, 0, span)
}
