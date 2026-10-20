// Package globe shows the world on a globe, as seen from space in the
// orthographic projection: each country's land and borders on the sea,
// lines of latitude and longitude every 15°, dots at places with their
// labels, and great-circle arcs between places, as flight routes. What
// lies beyond the horizon does not show. Dragging turns the globe, which
// goes on turning for a moment once let go, and the wheel zooms in and
// out. The pointer on a country lights it and tells its name.
//
//	app.globe.LookAt(dubai) // once, to start there
//	globe.View(c, &app.globe, app.airports, app.routes).Grow(1)
//
// It draws worldmap's countries.
package globe

import (
	"image"
	"image/draw"
	"math"
	"runtime"
	"sync"
	"time"

	"github.com/egoist/mygo/ui"
	"golang.org/x/image/vector"

	"github.com/xingkaixin/mygo-native-ui/ui/worldmap"
)

// Place is a point of the globe: its latitude and longitude in degrees,
// north and east positive.
type Place struct{ Lat, Lon float64 }

// Marker is a dot at a place, with a label beside it.
type Marker struct {
	Place
	Label string
}

// Arc is the shorter great-circle arc between two places, as a flight
// takes.
type Arc struct{ From, To Place }

// State is what a view keeps: the place at the globe's center, how far
// it is zoomed, and how fast it turns once let go. Keep it in the app's
// state, one for each view.
type State struct {
	// lon, lat are the place at the center, in degrees, and zoom how many
	// times the fit the globe shows, 0 the fit.
	lon, lat, zoom float64
	// vlon, vlat are how fast the globe turns, in degrees a second, as of
	// at.
	vlon, vlat float64
	at         time.Time
}

// LookAt turns the globe so that p is at its center.
func (s *State) LookAt(p Place) {
	s.vlon, s.vlat = 0, 0
	s.turn(p.Lon-s.lon, p.Lat-s.lat)
}

// maxZoom is how far a view zooms in: 8 times the fit.
const maxZoom = 8

// glide is how long, in seconds, a globe let go takes to turn e times
// slower.
const glide = 0.4

// turn moves the center dlon and dlat degrees, no further than the poles.
func (s *State) turn(dlon, dlat float64) {
	s.lon = math.Remainder(s.lon+dlon, 360)
	s.lat = max(-90, min(90, s.lat+dlat))
	if math.Abs(s.lat) == 90 {
		s.vlat = 0
	}
}

// coast turns the globe on as it turned at the last frame, slowing down,
// and reports whether it still turns faster than 5 DIPs a second, k
// degrees a DIP.
func (s *State) coast(now time.Time, k float64) bool {
	d := math.Exp(-now.Sub(s.at).Seconds() / glide)
	s.at = now
	s.turn(s.vlon*glide*(1-d), s.vlat*glide*(1-d))
	s.vlon, s.vlat = s.vlon*d, s.vlat*d
	if math.Hypot(s.vlon, s.vlat) < 5*k {
		s.vlon, s.vlat = 0, 0
	}
	return s.vlon != 0 || s.vlat != 0
}

// unit returns the point of a globe of radius 1 at lon, lat, in degrees:
// x toward 90° east, y toward the North Pole, z toward 0°, 0°.
func unit(lon, lat float64) [3]float32 {
	sl, cl := math.Sincos(lon * math.Pi / 180)
	sf, cf := math.Sincos(lat * math.Pi / 180)
	return [3]float32{float32(cf * sl), float32(sf), float32(cf * cl)}
}

// country is a country of worldmap: its code, the rings of its borders in
// degrees of longitude and latitude and as points of a globe of radius 1,
// and the box of the degrees, west, south, east and north.
type country struct {
	code   string
	lonlat [][][2]float32
	rings  [][][3]float32
	box    [4]float32
}

var load = sync.OnceValue(func() []country {
	var countries []country
	for code, rings := range worldmap.Borders() {
		inf := float32(math.Inf(1))
		ct := country{code: code, lonlat: rings, box: [4]float32{inf, inf, -inf, -inf}}
		for _, ring := range rings {
			pts := make([][3]float32, len(ring))
			for i, q := range ring {
				pts[i] = unit(float64(q[0]), float64(q[1]))
				ct.box = [4]float32{min(ct.box[0], q[0]), min(ct.box[1], q[1]), max(ct.box[2], q[0]), max(ct.box[3], q[1])}
			}
			ct.rings = append(ct.rings, pts)
		}
		countries = append(countries, ct)
	}
	return countries
})

// winding returns how many times ring, in degrees, goes counterclockwise
// around lon, lat: 1 inside an outer ring, -1 inside a hole, 0 outside.
func winding(ring [][2]float32, lon, lat float32) int {
	w := 0
	for i, a := range ring {
		b := ring[(i+1)%len(ring)]
		side := (b[0]-a[0])*(lat-a[1]) - (lon-a[0])*(b[1]-a[1])
		switch {
		case a[1] <= lat && b[1] > lat && side > 0:
			w++
		case a[1] > lat && b[1] <= lat && side < 0:
			w--
		}
	}
	return w
}

// contains reports whether lon, lat is in the country, and not in a hole
// of it, as Lesotho is in South Africa.
func (ct *country) contains(lon, lat float32) bool {
	if lon < ct.box[0] || lat < ct.box[1] || lon > ct.box[2] || lat > ct.box[3] {
		return false
	}
	w := 0
	for _, ring := range ct.lonlat {
		w += winding(ring, lon, lat)
	}
	return w != 0
}

// seam reports whether the edge from a to b of a ring, in degrees, is
// where worldmap cuts the rings, along the antimeridian or the South
// Pole, rather than a border.
func seam(a, b [2]float32) bool {
	return a[0] == b[0] && (a[0] == 180 || a[0] == -180) || a[1] == -90 && b[1] == -90
}

// graticule holds the lines of latitude and longitude every 15°, as
// points of a globe of radius 1 every 2°: the meridians up to 80° from
// the equator, where they would crowd, and the parallels up to 75°.
var graticule = sync.OnceValue(func() [][][3]float32 {
	var lines [][][3]float32
	for lon := -180; lon < 180; lon += 15 {
		var l [][3]float32
		for lat := -80; lat <= 80; lat += 2 {
			l = append(l, unit(float64(lon), float64(lat)))
		}
		lines = append(lines, l)
	}
	for lat := -75; lat <= 75; lat += 15 {
		var l [][3]float32
		for lon := -180; lon <= 180; lon += 2 {
			l = append(l, unit(float64(lon), float64(lat)))
		}
		lines = append(lines, l)
	}
	return lines
})

// eye is how a globe shows: m turns its points to the viewer's axes, x
// right, y up and z toward the viewer, and the globe is r around cx, cy,
// in units whose y goes down.
type eye struct {
	m         [3][3]float32
	cx, cy, r float32
}

// look returns the eye that sees lon, lat, in degrees, at the center.
func look(lon, lat float64, cx, cy, r float32) eye {
	sl, cl := math.Sincos(lon * math.Pi / 180)
	sf, cf := math.Sincos(lat * math.Pi / 180)
	return eye{m: [3][3]float32{
		{float32(cl), 0, float32(-sl)},
		{float32(-sf * sl), float32(cf), float32(-sf * cl)},
		{float32(cf * sl), float32(sf), float32(cf * cl)},
	}, cx: cx, cy: cy, r: r}
}

// turn returns v in the viewer's axes: before the horizon where z > 0.
func (e *eye) turn(v [3]float32) [3]float32 {
	m := &e.m
	return [3]float32{
		m[0][0]*v[0] + m[0][1]*v[1] + m[0][2]*v[2],
		m[1][0]*v[0] + m[1][1]*v[1] + m[1][2]*v[2],
		m[2][0]*v[0] + m[2][1]*v[1] + m[2][2]*v[2],
	}
}

// at returns where x, y of the viewer's axes shows.
func (e *eye) at(x, y float32) [2]float32 { return [2]float32{e.cx + e.r*x, e.cy - e.r*y} }

// place returns the longitude and latitude, in degrees, shown at x, y, or
// false off the globe.
func (e *eye) place(x, y float32) (lon, lat float32, ok bool) {
	vx, vy := (x-e.cx)/e.r, (e.cy-y)/e.r
	d := 1 - vx*vx - vy*vy
	if d < 0 || e.r <= 0 {
		return 0, 0, false
	}
	vz := float32(math.Sqrt(float64(d)))
	m := &e.m
	px := m[0][0]*vx + m[1][0]*vy + m[2][0]*vz
	py := m[0][1]*vx + m[1][1]*vy + m[2][1]*vz
	pz := m[0][2]*vx + m[1][2]*vy + m[2][2]*vz
	return float32(math.Atan2(float64(px), float64(pz)) * 180 / math.Pi), float32(math.Asin(float64(max(-1, min(1, py)))) * 180 / math.Pi), true
}

// horizon returns where the line from a to b, in the viewer's axes, one
// before the horizon and one behind it, crosses it: a point of the rim
// of radius 1.
func horizon(a, b [3]float32) [2]float32 {
	t := a[2] / (a[2] - b[2])
	x, y := a[0]+t*(b[0]-a[0]), a[1]+t*(b[1]-a[1])
	if l := float32(math.Hypot(float64(x), float64(y))); l > 0 {
		x, y = x/l, y/l
	}
	return [2]float32{x, y}
}

// outline appends to pts the ring as it fills the globe: its parts before
// the horizon where they show, and those behind it folded onto the rim,
// where they cover nothing, in steps of step radians. A ring all behind
// the horizon adds nothing. Folded so, a ring would turn inside out if it
// went around the point behind the center, which no ring of worldmap's
// does while it reaches the horizon: none is 90° across.
func (e *eye) outline(pts [][2]float32, ring [][3]float32, step float64) [][2]float32 {
	start := -1
	for i, v := range ring {
		if e.turn(v)[2] > 0 {
			start = i
			break
		}
	}
	if start < 0 {
		return pts
	}
	// a is how far round the rim, in radians, the ring has gone behind the
	// horizon, and dir the way from the center it went last.
	var a float64
	var dir [2]float32
	along := func(x, y float32) {
		if x == 0 && y == 0 {
			return
		}
		to := a + math.Atan2(float64(dir[0]*y-dir[1]*x), float64(dir[0]*x+dir[1]*y))
		if to > a {
			for g := math.Floor(a/step) + 1; g*step < to; g++ {
				pts = append(pts, e.at(float32(math.Cos(g*step)), float32(math.Sin(g*step))))
			}
		} else {
			for g := math.Ceil(a/step) - 1; g*step > to; g-- {
				pts = append(pts, e.at(float32(math.Cos(g*step)), float32(math.Sin(g*step))))
			}
		}
		a, dir = to, [2]float32{x, y}
	}
	var last [3]float32
	for k := range len(ring) + 1 {
		q := e.turn(ring[(start+k)%len(ring)])
		switch {
		case k > 0 && q[2] <= 0 && last[2] > 0:
			h := horizon(last, q)
			pts = append(pts, e.at(h[0], h[1]))
			a, dir = math.Atan2(float64(h[1]), float64(h[0])), h
			along(q[0], q[1])
		case q[2] <= 0:
			along(q[0], q[1])
		case k > 0 && last[2] <= 0:
			h := horizon(q, last)
			along(h[0], h[1])
			pts = append(pts, e.at(h[0], h[1]), e.at(q[0], q[1]))
		default:
			pts = append(pts, e.at(q[0], q[1]))
		}
		last = q
	}
	return pts
}

// pen traces lines of points of the globe as they show, a NaN point where
// they go behind the horizon or lift, for Painter.StrokePolyline.
type pen struct {
	e    *eye
	pts  [][2]float32
	last [3]float32
	down bool
}

var gap = [2]float32{float32(math.NaN()), float32(math.NaN())}

// to draws the line on to v, or starts it there.
func (p *pen) to(v [3]float32) {
	q := p.e.turn(v)
	if p.down && (p.last[2] > 0) != (q[2] > 0) {
		h := horizon(p.last, q)
		p.pts = append(p.pts, p.e.at(h[0], h[1]))
		if q[2] <= 0 {
			p.pts = append(p.pts, gap)
		}
	}
	if q[2] > 0 {
		p.pts = append(p.pts, p.e.at(q[0], q[1]))
	}
	p.last, p.down = q, true
}

// lift ends the line, for the next to start one.
func (p *pen) lift() { p.pts, p.down = append(p.pts, gap), false }

// arc draws the arc as points about 1° apart.
func (p *pen) arc(a Arc) {
	u, v := unit(a.From.Lon, a.From.Lat), unit(a.To.Lon, a.To.Lat)
	dot := float64(u[0]*v[0] + u[1]*v[1] + u[2]*v[2])
	w := math.Acos(max(-1, min(1, dot)))
	if w < 1e-6 || math.Pi-w < 1e-6 {
		return
	}
	n := int(math.Ceil(w * 180 / math.Pi))
	for i := range n + 1 {
		f := float64(i) / float64(n)
		ku, kv := float32(math.Sin((1-f)*w)/math.Sin(w)), float32(math.Sin(f*w)/math.Sin(w))
		p.to([3]float32{ku*u[0] + kv*v[0], ku*u[1] + kv*v[1], ku*u[2] + kv*v[2]})
	}
	p.lift()
}

// View creates the globe, the arcs on it and the markers over them. Give
// it a size, or Grow.
func View(c *ui.Context, s *State, markers []Marker, arcs []Arc) ui.Element {
	t := c.Theme()
	countries := load()
	box := ui.Box(c).Clip().Role(ui.RoleImage).Label("Globe").Cursor(ui.CursorGrab)
	b := box.LayoutBounds()
	rad := max(0, min(b.W, b.H)/2-t.Space(6)) * float32(max(1, s.zoom))
	k := 180 / math.Pi / float64(max(1, rad))
	if dx, dy, ok := box.Dragged(); ok {
		s.turn(-float64(dx)*k, float64(dy)*k)
		box.Cursor(ui.CursorGrabbing)
	}
	if box.Pressed() {
		s.vlon, s.vlat = 0, 0
	}
	now := c.Now()
	if vx, vy, ok := box.DragEnded(); ok && !c.Preferences().ReduceMotion {
		s.vlon, s.vlat, s.at = -float64(vx)*k, float64(vy)*k, now
	}
	if (s.vlon != 0 || s.vlat != 0) && s.coast(now, k) {
		c.AnimationFrame()
	}
	box.HandleInput(func(ev ui.InputEvent) bool {
		if ev.Kind != ui.InputScroll {
			return false
		}
		// Scrolling 300 DIPs zooms by 2, as worldmap does.
		s.zoom = max(1, min(maxZoom, max(1, s.zoom)*math.Exp2(-float64(ev.DY)/300)))
		return true
	})
	lon, lat := s.lon, s.lat
	hovered := -1
	px, py, over := box.PointerPosition()
	if e := look(lon, lat, b.W/2, b.H/2, rad); over && !box.Pressed() {
		if lo, la, ok := e.place(px, py); ok {
			for i := range countries {
				if countries[i].contains(lo, la) {
					hovered = i
					break
				}
			}
		}
	}

	srgb := func(l, ch, h float32) ui.Color { return ui.Oklch(l, ch, h).SRGB() }
	sea, land, border, grid, rim := srgb(0.93, 0.03, 235), srgb(0.99, 0.004, 235), srgb(0.8, 0.025, 240), srgb(0.88, 0.04, 235), srgb(0.78, 0.04, 240)
	if t.Dark {
		sea, land, border, grid, rim = srgb(0.25, 0.04, 255), srgb(0.38, 0.025, 255), srgb(0.25, 0.03, 255), srgb(0.3, 0.045, 255), srgb(0.42, 0.04, 255)
	}
	hot := land.Mix(t.Accent.SRGB(), 0.35)
	ras := ui.Local(box, "globe.raster", func() raster { return raster{} })
	paint := func(p *ui.Painter, r ui.Rect) {
		e := look(lon, lat, r.X+r.W/2, r.Y+r.H/2, rad)
		disc := ui.Rect{X: e.cx - rad, Y: e.cy - rad, W: 2 * rad, H: 2 * rad}
		p.Fill(disc, sea, rad)
		pn := pen{e: &e}
		for _, l := range graticule() {
			for _, v := range l {
				pn.to(v)
			}
			pn.lift()
		}
		p.StrokePolyline(pn.pts, 1, grid)

		// The bitmap covers r as Painter.Image places it, on whole pixels.
		sc := p.Scale()
		x0, y0 := round(r.X*sc), round(r.Y*sc)
		f := frame{
			w: int(round((r.X+r.W)*sc) - x0), h: int(round((r.Y+r.H)*sc) - y0),
			cx: e.cx*sc - x0, cy: e.cy*sc - y0, r: rad * sc, scale: sc, lon: lon, lat: lat,
			lit: hovered, land: land, hot: hot, border: border,
		}
		if bm := ras.draw(f); bm != nil {
			p.Image(bm, r, ui.FillBox)
		}
		p.Stroke(disc, rim, rad, 1)

		pn.pts = pn.pts[:0]
		for _, a := range arcs {
			pn.arc(a)
		}
		p.StrokePolyline(pn.pts, 1.5, t.Accent)

		dot := t.Space(1)
		type spot struct {
			at   [2]float32
			fade float32
			text string
		}
		var dots []spot
		for _, m := range markers {
			q := e.turn(unit(m.Lon, m.Lat))
			if q[2] <= 0 {
				continue
			}
			// A dot fades out as it nears the horizon rather than going at once.
			d := spot{at: e.at(q[0], q[1]), fade: min(1, q[2]/0.1), text: m.Label}
			dots = append(dots, d)
			halo := dot + 1.5
			p.Fill(ui.Rect{X: d.at[0] - halo, Y: d.at[1] - halo, W: 2 * halo, H: 2 * halo}, t.Background.Alpha(d.fade), halo)
			p.Fill(ui.Rect{X: d.at[0] - dot, Y: d.at[1] - dot, W: 2 * dot, H: 2 * dot}, t.Accent.Alpha(d.fade), dot)
		}
		var placed []ui.Rect
		for _, d := range dots {
			if d.text == "" {
				continue
			}
			span := ui.Span{Text: d.text, Size: t.FontSize - 1, Weight: 500, Color: t.Text.Alpha(d.fade)}
			w, h := p.MeasureText(0, span)
			lr := ui.Rect{X: d.at[0] + dot + t.Space(1.5), Y: d.at[1] - h/2, W: w, H: h}
			if overlaps(placed, lr) {
				continue
			}
			placed = append(placed, lr)
			p.RichText(lr.X, lr.Y, 0, span)
		}

		if hovered < 0 {
			return
		}
		span := ui.Span{Text: worldmap.Name(countries[hovered].code), Size: t.FontSize - 1, Weight: 500, Color: t.Background}
		w, h := p.MeasureText(0, span)
		padX, padY := t.Space(2), t.Space(1)
		tip := ui.Rect{X: r.X + px + t.Space(3), Y: r.Y + py - h - 2*padY - t.Space(1), W: w + 2*padX, H: h + 2*padY}
		tip.X = max(r.X+t.Space(1), min(r.X+r.W-tip.W-t.Space(1), tip.X))
		tip.Y = max(r.Y+t.Space(1), tip.Y)
		p.Fill(tip, t.Text, t.Space(1.5))
		p.RichText(tip.X+padX, tip.Y+padY, 0, span)
	}
	box.Draw(func(p *ui.Painter, r ui.Rect) {
		if r.W != b.W || r.H != b.H {
			c.Invalidate()
		}
		// The box's Clip holds in its children, not what it draws, as the sea
		// of a globe zoomed in.
		if rad > 0 {
			p.Clip(r, 0, func() { paint(p, r) })
		}
	})
	return box
}

// overlaps reports whether r overlaps any of rs.
func overlaps(rs []ui.Rect, r ui.Rect) bool {
	for _, o := range rs {
		if r.X < o.X+o.W && o.X < r.X+r.W && r.Y < o.Y+o.H && o.Y < r.Y+r.H {
			return true
		}
	}
	return false
}

func round(v float32) float32 { return float32(math.Round(float64(v))) }

// frame is the land of a globe as it shows in pixels: the size of the
// bitmap, the globe's center and radius in it, the pixels of a DIP, the
// place at the center, the country lit, and the colors of land, of the
// country lit and of borders.
type frame struct {
	w, h             int
	cx, cy, r, scale float32
	lon, lat         float64
	lit              int
	land, hot        ui.Color
	border           ui.Color
}

// shape is how country i shows: its rings in pixels, each ending where
// ends says, and the box of the pixels they cover.
type shape struct {
	i    int
	pts  [][2]float32
	ends []int
	box  image.Rectangle
}

// raster is the land of a globe drawn in pixels on the CPU, as worldmap
// draws its map, since a path mask a frame on the GPU would fill the
// atlas as the globe turns, and kept while what it shows holds. It draws
// in turns into two images, since a bitmap shows an image's pixels until
// the next Update.
type raster struct {
	f    frame
	bm   *ui.Bitmap
	pix  [2]*image.RGBA
	mask *image.Alpha
	// z holds a rasterizer for each band of rows, which draw at once.
	z []vector.Rasterizer
	// shapes holds the countries that show, the first shown of them, and
	// borders the lines of their borders, NaN between them.
	shapes  []shape
	shown   int
	borders [][2]float32
}

// draw returns the bitmap of f.
func (ras *raster) draw(f frame) *ui.Bitmap {
	if f.w <= 0 || f.h <= 0 {
		return nil
	}
	if ras.bm != nil && f == ras.f {
		return ras.bm
	}
	ras.f = f
	ras.project()
	ras.pix[0], ras.pix[1] = ras.pix[1], ras.pix[0]
	img := ras.pix[0]
	if img == nil || img.Rect.Dx() != f.w || img.Rect.Dy() != f.h {
		img = image.NewRGBA(image.Rect(0, 0, f.w, f.h))
		ras.pix[0], ras.mask = img, image.NewAlpha(img.Rect)
	} else {
		clear(img.Pix)
	}
	n := max(1, min(runtime.NumCPU(), f.h/64))
	if len(ras.z) != n {
		ras.z = make([]vector.Rasterizer, n)
	}
	var wg sync.WaitGroup
	for i := range n {
		band := image.Rect(0, f.h*i/n, f.w, f.h*(i+1)/n)
		wg.Go(func() { ras.paint(&ras.z[i], img, band) })
	}
	wg.Wait()
	if ras.bm == nil {
		ras.bm = ui.NewBitmap(img)
	} else {
		ras.bm.Update(img)
	}
	return ras.bm
}

// project finds the shapes and borders of the countries as f shows them.
func (ras *raster) project() {
	f := ras.f
	e := look(f.lon, f.lat, f.cx, f.cy, f.r)
	// Enough points on the rim that the lines between them stray from it
	// less than a fifth of a pixel.
	n := math.Ceil(2 * math.Pi / math.Sqrt(1.6/float64(max(1, f.r))))
	step := 2 * math.Pi / n
	pn := pen{e: &e, pts: ras.borders[:0]}
	bounds := image.Rect(0, 0, f.w, f.h)
	ras.shown = 0
	for i, ct := range load() {
		if ras.shown == len(ras.shapes) {
			ras.shapes = append(ras.shapes, shape{})
		}
		sh := &ras.shapes[ras.shown]
		sh.i, sh.pts, sh.ends = i, sh.pts[:0], sh.ends[:0]
		for k, ring := range ct.rings {
			from := len(sh.pts)
			if sh.pts = e.outline(sh.pts, ring, step); len(sh.pts) > from {
				sh.ends = append(sh.ends, len(sh.pts))
			}
			ll := ct.lonlat[k]
			for j := range len(ring) + 1 {
				if j > 0 && seam(ll[j-1], ll[j%len(ll)]) {
					pn.lift()
				}
				pn.to(ring[j%len(ring)])
			}
			pn.lift()
		}
		if len(sh.pts) == 0 {
			continue
		}
		x0, y0, x1, y1 := sh.pts[0][0], sh.pts[0][1], sh.pts[0][0], sh.pts[0][1]
		for _, q := range sh.pts {
			x0, y0, x1, y1 = min(x0, q[0]), min(y0, q[1]), max(x1, q[0]), max(y1, q[1])
		}
		sh.box = image.Rect(int(x0)-1, int(y0)-1, int(x1)+2, int(y1)+2).Intersect(bounds)
		if !sh.box.Empty() {
			ras.shown++
		}
	}
	ras.borders = pn.pts
}

// paint draws the rows band of img with z: the countries filled, and
// their borders.
func (ras *raster) paint(z *vector.Rasterizer, img *image.RGBA, band image.Rectangle) {
	f := ras.f
	for _, sh := range ras.shapes[:ras.shown] {
		r := sh.box.Intersect(band)
		if r.Empty() {
			continue
		}
		z.Reset(r.Dx(), r.Dy())
		ox, oy := float32(r.Min.X), float32(r.Min.Y)
		from := 0
		for _, end := range sh.ends {
			z.MoveTo(sh.pts[from][0]-ox, sh.pts[from][1]-oy)
			for _, q := range sh.pts[from+1 : end] {
				z.LineTo(q[0]-ox, q[1]-oy)
			}
			z.ClosePath()
			from = end
		}
		c := f.land
		if sh.i == f.lit {
			c = f.hot
		}
		ras.fill(z, img, r, c)
	}
	z.Reset(band.Dx(), band.Dy())
	ox, oy := float32(band.Min.X), float32(band.Min.Y)
	hw := 0.75 * f.scale / 2
	lo, hi := oy-hw-1, float32(band.Max.Y)+hw+1
	for i := 1; i < len(ras.borders); i++ {
		a, b := ras.borders[i-1], ras.borders[i]
		if math.IsNaN(float64(a[0])) || math.IsNaN(float64(b[0])) || max(a[1], b[1]) < lo || min(a[1], b[1]) > hi {
			continue
		}
		line(z, [2]float32{a[0] - ox, a[1] - oy}, [2]float32{b[0] - ox, b[1] - oy}, hw)
	}
	ras.fill(z, img, band, f.border)
}

// line adds to z the line from a to b, hw pixels to either side, reaching
// hw past each end so that the lines of a ring meet at its corners.
func line(z *vector.Rasterizer, a, b [2]float32, hw float32) {
	l := float32(math.Hypot(float64(b[0]-a[0]), float64(b[1]-a[1])))
	if l == 0 {
		return
	}
	ux, uy := (b[0]-a[0])/l*hw, (b[1]-a[1])/l*hw
	z.MoveTo(a[0]-ux-uy, a[1]-uy+ux)
	z.LineTo(b[0]+ux-uy, b[1]+uy+ux)
	z.LineTo(b[0]+ux+uy, b[1]+uy-ux)
	z.LineTo(a[0]-ux+uy, a[1]-uy-ux)
	z.ClosePath()
}

// fill paints c on the rectangle r of img where the shapes z holds, as
// large as r, cover it.
func (ras *raster) fill(z *vector.Rasterizer, img *image.RGBA, r image.Rectangle, c ui.Color) {
	z.DrawOp = draw.Src
	z.Draw(ras.mask, r, image.Opaque, image.Point{})
	for y := r.Min.Y; y < r.Max.Y; y++ {
		m := ras.mask.Pix[ras.mask.PixOffset(r.Min.X, y):][:r.Dx()]
		p := img.Pix[img.PixOffset(r.Min.X, y):]
		for x, a := range m {
			if a == 0 {
				continue
			}
			a := uint32(a) * uint32(c.A) / 255
			q := p[4*x : 4*x+4 : 4*x+4]
			q[0] = uint8((uint32(c.R)*a + uint32(q[0])*(255-a)) / 255)
			q[1] = uint8((uint32(c.G)*a + uint32(q[1])*(255-a)) / 255)
			q[2] = uint8((uint32(c.B)*a + uint32(q[2])*(255-a)) / 255)
			q[3] = uint8((255*a + uint32(q[3])*(255-a)) / 255)
		}
	}
}
