// Package tilemap shows a map of raster tiles in Web Mercator, as
// OpenStreetMap's site does: dragging pans it, and a flick carries it on;
// the wheel zooms around the pointer, as a double click does, and the +
// and − buttons and keys zoom around its center. Between whole zoom
// levels it scales the tiles of the nearest one, and while a tile loads,
// a loaded one of the level above or below stands in. Markers show
// places with labels of the app's, and a click tells where it was. A
// scale bar and the source's attribution sit in the bottom corners.
//
//	app.tiles = &tilemap.Tiles{
//		URL:            tilemap.OpenStreetMap,
//		Attribution:    "© OpenStreetMap contributors",
//		AttributionURL: "https://www.openstreetmap.org/copyright",
//		UserAgent:      "Atlas/1.0 (+https://example.com/atlas)",
//	}
//	...
//	ev := tilemap.View(c, &app.mapView, &app.center, &app.zoom, app.tiles, app.pins)
//	if at, ok := ev.Clicked(); ok {
//		... // at.Lat, at.Lon
//	}
//	...
//	app.mapView.FlyTo(c, tilemap.LatLon{Lat: 48.8566, Lon: 2.3522}, 12)
package tilemap

import (
	"container/list"
	"errors"
	"io"
	"math"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/egoist/mygo/ui"
)

// OpenStreetMap is the URL of OpenStreetMap's standard tiles. Its tile
// usage policy asks for the attribution "© OpenStreetMap contributors",
// linked to https://www.openstreetmap.org/copyright, and a UserAgent
// naming the app.
const OpenStreetMap = "https://tile.openstreetmap.org/{z}/{x}/{y}.png"

// tileSize is how many DIPs a tile takes at its own zoom level.
const tileSize = 256

// maxLat is the latitude, north and south, where Web Mercator's square
// world ends.
const maxLat = 85.0511287798066

// LatLon is a place on Earth, its latitude and longitude in degrees.
type LatLon struct{ Lat, Lon float64 }

// project returns where p is in the world, from 0, 0 at its top left to
// 1, 1 at its bottom right.
func project(p LatLon) (x, y float64) {
	lat := max(-maxLat, min(maxLat, p.Lat)) * math.Pi / 180
	return (p.Lon + 180) / 360, (1 - math.Log(math.Tan(lat)+1/math.Cos(lat))/math.Pi) / 2
}

// unproject returns the place at x, y of the world, its longitude within
// -180 and 180.
func unproject(x, y float64) LatLon {
	x -= math.Floor(x)
	return LatLon{Lat: math.Atan(math.Sinh(math.Pi*(1-2*y))) * 180 / math.Pi, Lon: x*360 - 180}
}

// Marker is a place the map shows, with its label beside it.
type Marker struct {
	At    LatLon
	Label string
}

// client fetches tiles, giving up on a server that does not answer.
var client = &http.Client{Timeout: 30 * time.Second}

// retry is how long a tile that failed to load waits before it is asked
// for again.
const retry = time.Minute

// Tiles loads the tiles of a source in the background, Workers at once,
// those of the latest frame first and nearest its center, and keeps the
// latest shown. Its fields are read once, at the first View; share one
// among the views of a source.
type Tiles struct {
	// URL is where a tile is, with {z}, {x} and {y} for its zoom level,
	// column and row, as OpenStreetMap.
	URL string
	// Attribution credits the source, in the map's corner, as a link to
	// AttributionURL when it has one.
	Attribution, AttributionURL string
	// UserAgent names the app to the server, as "Atlas/1.0
	// (+https://example.com/atlas)": OpenStreetMap's servers refuse a
	// generic one, as Go's own.
	UserAgent string
	// MaxZoom is the deepest zoom level the source has, 19 by default.
	MaxZoom int
	// Cache is how many tiles it keeps, 256 by default.
	Cache int
	// Workers is how many tiles it loads at once, 2 by default.
	Workers int

	mu sync.Mutex
	// tiles are those loaded or failed, by key, in lru, the latest shown
	// first.
	tiles map[tile]*list.Element
	lru   list.List
	// queue is the tiles wanted, the first first, and loading those being
	// fetched.
	queue   []tile
	loading map[tile]bool
	running int
	wake    func()
}

// tile is a tile of the world: its zoom level, column and row.
type tile struct{ z, x, y int }

// entry is a tile loaded, or when it failed to, the time it did.
type entry struct {
	key    tile
	bmp    *ui.Bitmap
	failed time.Time
}

func (l *Tiles) start(c *ui.Context) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.tiles == nil {
		l.tiles, l.loading, l.wake = map[tile]*list.Element{}, map[tile]bool{}, c.Invalidate
		l.MaxZoom, l.Cache, l.Workers = or(l.MaxZoom, 19), or(l.Cache, 256), or(l.Workers, 2)
	}
}

func or(v, def int) int {
	if v <= 0 {
		return def
	}
	return v
}

// bitmap returns the tile of k, or nil while it loads.
func (l *Tiles) bitmap(k tile) *ui.Bitmap {
	l.mu.Lock()
	defer l.mu.Unlock()
	if e, ok := l.tiles[k]; ok {
		l.lru.MoveToFront(e)
		return e.Value.(*entry).bmp
	}
	return nil
}

// want has the tiles of keys loaded, the first first, in place of those
// wanted before, which no longer show.
func (l *Tiles) want(keys []tile) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.queue = l.queue[:0]
	now := time.Now()
	for _, k := range keys {
		if e, ok := l.tiles[k]; ok {
			if e := e.Value.(*entry); e.bmp != nil || now.Sub(e.failed) < retry {
				continue
			}
		}
		if !l.loading[k] && !slices.Contains(l.queue, k) {
			l.queue = append(l.queue, k)
		}
	}
	for l.running < min(l.Workers, len(l.queue)) {
		l.running++
		go l.work()
	}
}

// work loads the tiles wanted until none waits.
func (l *Tiles) work() {
	for {
		l.mu.Lock()
		if len(l.queue) == 0 {
			l.running--
			l.mu.Unlock()
			return
		}
		k := l.queue[0]
		l.queue = l.queue[1:]
		l.loading[k] = true
		l.mu.Unlock()

		e := &entry{key: k}
		var err error
		if e.bmp, err = l.fetch(k); err != nil {
			e.failed = time.Now()
		}

		l.mu.Lock()
		delete(l.loading, k)
		if old, ok := l.tiles[k]; ok {
			l.lru.Remove(old)
		}
		l.tiles[k] = l.lru.PushFront(e)
		for l.lru.Len() > l.Cache {
			delete(l.tiles, l.lru.Remove(l.lru.Back()).(*entry).key)
		}
		wake := l.wake
		l.mu.Unlock()
		wake()
	}
}

func (l *Tiles) fetch(k tile) (*ui.Bitmap, error) {
	url := strings.NewReplacer("{z}", strconv.Itoa(k.z), "{x}", strconv.Itoa(k.x), "{y}", strconv.Itoa(k.y)).Replace(l.URL)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", l.UserAgent)
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, errors.New(url + ": " + res.Status)
	}
	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	return ui.DecodeBitmap(data)
}

// paint draws the tile of k into dst, or while it loads, the part of the
// nearest loaded tile of the levels above that covers it, under the
// loaded tiles of the level below.
func (l *Tiles) paint(p *ui.Painter, k tile, dst ui.Rect) {
	var o ui.ImageOptions
	if bm := l.bitmap(k); bm != nil {
		p.ImageRegion(bm, ui.Rect{}, dst, o)
		return
	}
	// Eight levels up, the part of a tile is a pixel.
	for d := 1; d <= min(k.z, 8); d++ {
		bm := l.bitmap(tile{k.z - d, k.x >> d, k.y >> d})
		if bm == nil {
			continue
		}
		w, h := bm.Size()
		n := float32(int(1) << d)
		i, j := float32(k.x&(1<<d-1)), float32(k.y&(1<<d-1))
		p.ImageRegion(bm, ui.Rect{X: i * float32(w) / n, Y: j * float32(h) / n, W: float32(w) / n, H: float32(h) / n}, dst, o)
		break
	}
	hw, hh := dst.W/2, dst.H/2
	for b := range 2 {
		for a := range 2 {
			if bm := l.bitmap(tile{k.z + 1, 2*k.x + a, 2*k.y + b}); bm != nil {
				p.ImageRegion(bm, ui.Rect{}, ui.Rect{X: dst.X + float32(a)*hw, Y: dst.Y + float32(b)*hh, W: hw, H: hh}, o)
			}
		}
	}
}

// camera is what a view shows: the point of the world at its center, as
// project returns it, its zoom level, its size in DIPs, and the deepest
// level it zooms to.
type camera struct{ x, y, zoom, w, h, top float64 }

// size returns how many DIPs the world takes across.
func (m camera) size() float64 { return tileSize * math.Exp2(m.zoom) }

// clamp keeps the zoom between the level at which the world is as high
// as the view and the deepest, the center within the world's columns,
// and the world's top and bottom outside the view.
func (m *camera) clamp() {
	m.zoom = max(0, math.Log2(m.h/tileSize), min(m.top, m.zoom))
	m.x -= math.Floor(m.x)
	if half := m.h / 2 / m.size(); half >= 0.5 {
		m.y = 0.5
	} else {
		m.y = max(half, min(1-half, m.y))
	}
}

// at returns the point of the world at px, py of the view.
func (m camera) at(px, py float64) (x, y float64) {
	k := m.size()
	return m.x + (px-m.w/2)/k, m.y + (py-m.h/2)/k
}

// pan moves the world by dx, dy DIPs.
func (m *camera) pan(dx, dy float64) {
	k := m.size()
	m.x, m.y = m.x-dx/k, m.y-dy/k
	m.clamp()
}

// zoomAt zooms to level z, keeping the point at px, py of the view where
// it is.
func (m *camera) zoomAt(z, px, py float64) {
	x, y := m.at(px, py)
	m.zoom = z
	m.clamp()
	k := m.size()
	m.x, m.y = x-(px-m.w/2)/k, y-(py-m.h/2)/k
	m.clamp()
}

// grid is how the tiles of level z cover a view: ts DIPs each, from
// column i0 and row j0, with its top left at x0, y0 of the view, to
// column i1 and row j1. Columns past the world's edges wrap around it.
type grid struct {
	z              int
	ts, x0, y0     float64
	i0, i1, j0, j1 int
}

func (m camera) grid() grid {
	z := int(max(0, min(m.top, math.Round(m.zoom))))
	ts := tileSize * math.Exp2(m.zoom-float64(z))
	k := m.size()
	ox, oy := m.x*k-m.w/2, m.y*k-m.h/2
	g := grid{z: z, ts: ts}
	g.i0, g.i1 = int(math.Floor(ox/ts)), int(math.Floor((ox+m.w)/ts))
	g.j0, g.j1 = max(0, int(math.Floor(oy/ts))), min(1<<z-1, int(math.Floor((oy+m.h)/ts)))
	g.x0, g.y0 = float64(g.i0)*ts-ox, float64(g.j0)*ts-oy
	return g
}

// key returns the tile of column i and row j.
func (g grid) key(i, j int) tile {
	n := 1 << g.z
	return tile{g.z, (i%n + n) % n, j}
}

// keys returns the tiles of g, those nearest the view's center w by h
// DIPs first.
func (g grid) keys(w, h float64) []tile {
	type near struct {
		k tile
		d float64
	}
	var all []near
	for j := g.j0; j <= g.j1; j++ {
		for i := g.i0; i <= g.i1; i++ {
			dx := g.x0 + (float64(i-g.i0)+0.5)*g.ts - w/2
			dy := g.y0 + (float64(j-g.j0)+0.5)*g.ts - h/2
			all = append(all, near{g.key(i, j), dx*dx + dy*dy})
		}
	}
	slices.SortStableFunc(all, func(a, b near) int { return int(math.Copysign(1, a.d-b.d)) })
	keys := make([]tile, len(all))
	for i, n := range all {
		keys[i] = n.k
	}
	return keys
}

// glide is how long a fling takes to slow down by e, about as iOS's
// scroll views do.
const glide = 0.35

// flight is a flight from one view to another, along van Wijk and
// Nuij's smooth and efficient path, which zooms out as far as it goes
// and back in, as d3's interpolateZoom does.
type flight struct {
	// to and zoom are where it goes, and pending that it starts in the
	// next frame, on, that it is under way.
	to          LatLon
	zoom        float64
	pending, on bool
	start       time.Time
	dur         time.Duration
	// x0, y0 is the center it starts from, and w0 the width it shows, in
	// units of the world; dx, dy the way to the center it goes to, d its
	// length, and w1 the width it shows there.
	x0, y0, dx, dy, d, w0, w1 float64
	r0, s                     float64
}

// rho is how far a flight zooms out on its way: the larger, the farther.
const rho = math.Sqrt2

func (f *flight) begin(m camera, now time.Time, reduce bool) {
	end := m
	end.x, end.y = project(f.to)
	end.zoom = f.zoom
	end.clamp()
	f.zoom = end.zoom
	f.pending, f.on, f.start = false, true, now
	f.x0, f.y0, f.dx, f.dy = m.x, m.y, end.x-m.x, end.y-m.y
	// The world wraps around: the way east or west, whichever is shorter.
	f.dx -= math.Round(f.dx)
	f.d = math.Hypot(f.dx, f.dy)
	f.w0, f.w1 = m.w/m.size(), m.w/end.size()
	if f.d < 1e-12 {
		f.s = math.Log(f.w1/f.w0) / rho
	} else {
		b0 := (f.w1*f.w1 - f.w0*f.w0 + rho*rho*rho*rho*f.d*f.d) / (2 * f.w0 * rho * rho * f.d)
		b1 := (f.w1*f.w1 - f.w0*f.w0 - rho*rho*rho*rho*f.d*f.d) / (2 * f.w1 * rho * rho * f.d)
		f.r0 = math.Log(math.Sqrt(b0*b0+1) - b0)
		f.s = (math.Log(math.Sqrt(b1*b1+1)-b1) - f.r0) / rho
	}
	// d3 takes S·ρ/√2 seconds, slow across the world: a third of S, held
	// between a step's 0.3 s and 2 s, keeps long flights short.
	f.dur = time.Duration(min(2, max(0.3, math.Abs(f.s)/3)) * float64(time.Second))
	if reduce || m.w <= 0 {
		f.dur = 0
	}
}

// step moves m along the flight to where it is at now, and reports
// whether it is over.
func (f *flight) step(m *camera, now time.Time) bool {
	t := 1.0
	if f.dur > 0 {
		t = min(1, float64(now.Sub(f.start))/float64(f.dur))
	}
	if t >= 1 {
		f.on = false
		m.x, m.y = project(f.to)
		m.zoom = f.zoom
		m.clamp()
		return true
	}
	s := float64(ui.EaseInOut(float32(t))) * f.s
	u, w := 0.0, f.w0*math.Exp(rho*s)
	if f.d >= 1e-12 {
		u = f.w0 / (rho * rho * f.d) * (math.Cosh(f.r0)*math.Tanh(rho*s+f.r0) - math.Sinh(f.r0))
		w = f.w0 * math.Cosh(f.r0) / math.Cosh(rho*s+f.r0)
	}
	m.x, m.y, m.zoom = f.x0+u*f.dx, f.y0+u*f.dy, math.Log2(m.w/w/tileSize)
	m.clamp()
	return false
}

// State is what a view keeps: how the map moves on its own, as after a
// flick or towards a place to fly to. Keep it in the app's state, one
// for each view.
type State struct {
	// w, h are the view's size as of the last frame, and moved how far the
	// pointer pressing it went: far, the press pans rather than clicks.
	w, h, moved float64
	// vx, vy are how fast a flick carries the map, in DIPs per second, as
	// of last.
	vx, vy float64
	last   time.Time
	fly    flight
}

// FlyTo moves the view to center to at zoom level zoom, zooming out on
// the way as far as the distance asks, or at once when the desktop asks
// for less motion. Dragging, scrolling or the keys stop it.
func (s *State) FlyTo(c *ui.Context, to LatLon, zoom float64) {
	s.fly = flight{to: to, zoom: zoom, pending: true}
	c.Invalidate()
}

// zoomBy flies around the view's center to the next whole level in or
// out, d 1 or -1, from where a flight under way goes.
func (s *State) zoomBy(c *ui.Context, m camera, d float64) {
	to, z := unproject(m.x, m.y), m.zoom
	if s.fly.on || s.fly.pending {
		to, z = s.fly.to, s.fly.zoom
	}
	// A level a hair off a whole one, as float math leaves it, counts as it.
	z = math.Floor(d*z+1e-6) + 1
	s.FlyTo(c, to, d*z)
}

// stop stops a flick and a flight.
func (s *State) stop() {
	s.vx, s.vy = 0, 0
	s.fly.on, s.fly.pending = false, false
}

func (s *State) camera(center *LatLon, zoom *float64, top int) camera {
	x, y := project(*center)
	m := camera{x: x, y: y, zoom: *zoom, w: s.w, h: s.h, top: float64(top)}
	m.clamp()
	return m
}

// Events are what the user did with a map in a frame. The zero Events
// report nothing.
type Events struct {
	at              LatLon
	marker          int
	clicked, picked bool
}

// Clicked returns the place the user clicked, off the markers.
func (e Events) Clicked() (LatLon, bool) { return e.at, e.clicked }

// Marker returns the index of the marker the user clicked.
func (e Events) Marker() (int, bool) { return e.marker, e.picked }

// spot is where marker i shows in a view: its point, and the box of its
// label.
type spot struct {
	i     int
	x, y  float32
	label ui.Rect
}

// View creates a map of the tiles of tiles around *center at zoom level
// *zoom, which the user moves, with markers, and s keeps what else it
// shows. It fills its parent. The Events returned tell what the user
// clicked.
//
// Dragging pans the map, and a flick carries it on. A mouse's wheel, or
// a trackpad's scrolling with Cmd held, Command on macOS and Control
// elsewhere, zooms around the pointer, and a double click zooms in a
// level there; a trackpad's scrolling pans. The buttons, and + and −
// while the map has the keyboard focus, zoom a level around its center,
// and the arrows pan. The map zooms from the level at which the world is
// as high as the view to the source's MaxZoom, and wraps around east
// and west.
func View(c *ui.Context, s *State, center *LatLon, zoom *float64, tiles *Tiles, markers []Marker) Events {
	t := c.Theme()
	tiles.start(c)
	top := tiles.MaxZoom
	var ev Events

	view := ui.Box(c).Fill().Clip().Focusable().Role(ui.RoleImage).Label("Map").Background(t.Surface)
	b := view.LayoutBounds()
	s.w, s.h = float64(b.W), float64(b.H)
	m := s.camera(center, zoom, top)
	was := m
	now := c.Now()

	view.Cursor(ui.CursorGrab)
	if view.Pressed() {
		s.stop()
	}
	if dx, dy, ok := view.Dragged(); ok {
		s.moved += math.Abs(float64(dx)) + math.Abs(float64(dy))
		m.pan(float64(dx), float64(dy))
		view.Cursor(ui.CursorGrabbing)
	}
	if vx, vy, ok := view.DragEnded(); ok {
		s.vx, s.vy, s.last = float64(vx), float64(vy), now
	}
	for _, k := range []struct {
		mods   ui.Modifiers
		key    ui.Key
		zoom   float64
		dx, dy float64
	}{
		{0, ui.KeyEqual, 1, 0, 0},
		{ui.Shift, ui.KeyEqual, 1, 0, 0},
		{0, ui.KeyMinus, -1, 0, 0},
		{0, ui.KeyLeft, 0, 100, 0},
		{0, ui.KeyRight, 0, -100, 0},
		{0, ui.KeyUp, 0, 0, 100},
		{0, ui.KeyDown, 0, 0, -100},
	} {
		if !view.Shortcut(k.mods, k.key) {
			continue
		}
		if k.zoom != 0 {
			s.zoomBy(c, m, k.zoom)
		} else {
			s.stop()
			m.pan(k.dx, k.dy)
		}
	}

	if s.vx != 0 || s.vy != 0 {
		dt := now.Sub(s.last).Seconds()
		s.last = now
		f := math.Exp(-dt / glide)
		m.pan(s.vx*glide*(1-f), s.vy*glide*(1-f))
		s.vx, s.vy = s.vx*f, s.vy*f
		if math.Hypot(s.vx, s.vy) < 10 {
			s.vx, s.vy = 0, 0
		} else {
			c.AnimationFrame()
		}
	}
	if s.fly.pending {
		s.fly.begin(m, now, c.Preferences().ReduceMotion)
	}
	if s.fly.on && !s.fly.step(&m, now) {
		c.AnimationFrame()
	}

	// Scrolling comes between builds, at the box laid out since, which
	// only a handle reads then.
	ref := ui.Local(view, "tilemap.handle", func() ui.Handle { return ui.Handle{} })
	view.Bind(ref)
	view.HandleInput(func(ev ui.InputEvent) bool {
		if ev.Kind != ui.InputScroll {
			return false
		}
		b := ref.Bounds(c)
		s.w, s.h = float64(b.W), float64(b.H)
		s.stop()
		m := s.camera(center, zoom, top)
		if ev.Precise && ev.Mods&(ui.Cmd|ui.Ctrl) == 0 {
			m.pan(float64(-ev.DX), float64(-ev.DY))
		} else {
			// A notch of a mouse's wheel, 40 DIPs, zooms half a level; a
			// trackpad's scrolling, many small steps, as zoomview does.
			per := 80.0
			if ev.Precise {
				per = 300
			}
			m.zoomAt(m.zoom-float64(ev.DY)/per, float64(ev.X), float64(ev.Y))
		}
		*center, *zoom = unproject(m.x, m.y), m.zoom
		return true
	})

	spots := m.spots(c, markers)
	hovered := -1
	px, py, pointer := view.PointerPosition()
	if pointer {
		for _, sp := range slices.Backward(spots) {
			dx, dy := px-sp.x, py-sp.y
			if dx*dx+dy*dy <= 100 || sp.label.Contains(px, py) {
				hovered = sp.i
				view.Cursor(ui.CursorPointer)
				break
			}
		}
	}
	switch {
	case pointer && view.DoubleClicked():
		s.stop()
		m.zoomAt(math.Floor(m.zoom)+1, float64(px), float64(py))
	// A press that moved a few DIPs is a drag, not a click.
	case pointer && view.Clicked() && s.moved < 4:
		if hovered >= 0 {
			ev.marker, ev.picked = hovered, true
		} else {
			ev.at, ev.clicked = unproject(m.at(float64(px), float64(py))), true
		}
	}
	if !view.Pressed() {
		s.moved = 0
	}

	if m != was {
		*center, *zoom = unproject(m.x, m.y), m.zoom
	}
	g := m.grid()
	tiles.want(g.keys(m.w, m.h))
	view.Value("Zoom level " + strconv.FormatFloat(m.zoom, 'f', 1, 64))

	pin := t.Accent
	view.Draw(func(p *ui.Painter, r ui.Rect) {
		if r.W != b.W || r.H != b.H {
			c.Invalidate()
		}
		sc := float64(p.Scale())
		// Tiles meet on whole pixels, so that no seam shows between them.
		snap := func(v float64) float32 { return float32(math.Round(v*sc) / sc) }
		// Clip cuts the view's children, not what it draws itself.
		p.Clip(r, 0, func() {
			for j := g.j0; j <= g.j1; j++ {
				y := float64(r.Y) + g.y0 + float64(j-g.j0)*g.ts
				for i := g.i0; i <= g.i1; i++ {
					x := float64(r.X) + g.x0 + float64(i-g.i0)*g.ts
					dst := ui.Rect{X: snap(x), Y: snap(y)}
					dst.W, dst.H = snap(x+g.ts)-dst.X, snap(y+g.ts)-dst.Y
					tiles.paint(p, g.key(i, j), dst)
				}
			}
			size := t.FontSize - 1
			for _, sp := range spots {
				x, y := r.X+sp.x, r.Y+sp.y
				rad := float32(5)
				if sp.i == hovered {
					rad = 6.5
				}
				dot := ui.Rect{X: x - rad - 2, Y: y - rad - 2, W: 2*rad + 4, H: 2*rad + 4}
				p.Shadow(dot, rad+2, 0, 1, 3, 0, ui.RGB(0, 0, 0).Alpha(0.35))
				p.Fill(dot, ui.RGB(255, 255, 255), rad+2)
				p.Fill(ui.Rect{X: x - rad, Y: y - rad, W: 2 * rad, H: 2 * rad}, pin, rad)
				if l := sp.label; l.W > 0 {
					l.X, l.Y = l.X+r.X, l.Y+r.Y
					p.Shadow(l, l.H/2, 0, 1, 2, 0, ui.RGB(0, 0, 0).Alpha(0.2))
					p.Fill(l, t.Background, l.H/2)
					p.Text(l.X+labelPad, l.Y+1, markers[sp.i].Label, size, t.Text)
				}
			}
		})
	})

	view.Children(func() {
		controls(c, s, m, top)
		scaleBar(c, m)
		attribution(c, tiles)
	})
	return ev
}

// labelPad is the room beside a marker's label inside its box.
const labelPad = 6

// spots returns where markers show in the view, on each copy of the world
// across it.
func (m camera) spots(c *ui.Context, markers []Marker) []spot {
	size := c.Theme().FontSize - 1
	k := m.size()
	var spots []spot
	for i, mk := range markers {
		var lw, lh float32
		if mk.Label != "" {
			lw, lh = c.MeasureText(0, ui.Span{Text: mk.Label, Size: size})
			lw, lh = lw+2*labelPad, lh+2
		}
		x, y := project(mk.At)
		sy := float32(m.h/2 + (y-m.y)*k)
		dx := x - m.x
		dx -= math.Round(dx)
		for n := math.Floor(-m.w/2/k - dx); n <= math.Ceil(m.w/2/k-dx); n++ {
			sx := float32(m.w/2 + (dx+n)*k)
			if sx < -lw-20 || sx > float32(m.w)+20 || sy < -20 || sy > float32(m.h)+20 {
				continue
			}
			sp := spot{i: i, x: sx, y: sy}
			if lw > 0 {
				sp.label = ui.Rect{X: sx + 11, Y: sy - lh/2, W: lw, H: lh}
			}
			spots = append(spots, sp)
		}
	}
	return spots
}

var (
	plus  = icon(`<path d="M5 12h14"/><path d="M12 5v14"/>`)
	minus = icon(`<path d="M5 12h14"/>`)
)

func icon(body string) *ui.SVG {
	return ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">` + body + `</svg>`))
}

// controls builds the buttons zooming in and out, at the view's top
// right.
func controls(c *ui.Context, s *State, m camera, top int) {
	t := c.Theme()
	lo := max(0, math.Log2(m.h/tileSize))
	side := t.Space(8)
	ui.Column(c).Absolute().Top(t.Space(3)).Right(t.Space(3)).Radius(t.Radius).Clip().
		Border(1, t.Border).Shadow(0, 1, 4, 0, ui.RGB(0, 0, 0).Alpha(0.15)).Children(func() {
		for _, b := range []struct {
			name string
			svg  *ui.SVG
			d    float64
			off  bool
		}{
			{"Zoom in", plus, 1, m.zoom >= float64(top)},
			{"Zoom out", minus, -1, m.zoom <= lo+1e-9},
		} {
			if b.d < 0 {
				ui.Divider(c)
			}
			btn := ui.Button(c, "").Label(b.name).Tooltip(b.name).Disabled(b.off).Size(side, side).
				Padding(0).Radius(0).Justify(ui.Center).Background(t.Background).Children(func() {
				ui.Icon(c, b.svg)
			})
			if btn.Clicked() {
				s.zoomBy(c, m, b.d)
			}
		}
	})
}

// scaleBar builds the bar of a round distance at the view's center, at
// its bottom left.
func scaleBar(c *ui.Context, m camera) {
	t := c.Theme()
	lat := unproject(m.x, m.y).Lat * math.Pi / 180
	perDIP := 2 * math.Pi * 6378137 * math.Cos(lat) / m.size()
	if perDIP <= 0 {
		return
	}
	maxW := float64(t.Space(25))
	meters := math.Pow(10, math.Floor(math.Log10(maxW*perDIP)))
	for _, f := range []float64{5, 2} {
		if f*meters/perDIP <= maxW {
			meters *= f
			break
		}
	}
	label := strconv.FormatFloat(meters, 'f', -1, 64) + " m"
	if meters >= 1000 {
		label = strconv.FormatFloat(meters/1000, 'f', -1, 64) + " km"
	}
	w := float32(meters / perDIP)
	ui.Column(c).Absolute().Left(t.Space(3)).Bottom(t.Space(3)).Gap(t.Space(0.5)).PassThrough().
		Padding(t.Space(1), t.Space(1.5)).Radius(t.Space(1)).Background(t.Background.Alpha(0.8)).Children(func() {
		ui.Text(c, label).FontSize(t.FontSize - 2).FontFeatures("tnum").SingleLine()
		ui.Box(c).Size(w, t.Space(1.5)).Draw(func(p *ui.Painter, r ui.Rect) {
			p.Line(r.X, r.Y+r.H, r.X+r.W, r.Y+r.H, 1.5, t.Text)
			p.Line(r.X, r.Y, r.X, r.Y+r.H, 1.5, t.Text)
			p.Line(r.X+r.W, r.Y, r.X+r.W, r.Y+r.H, 1.5, t.Text)
		})
	})
}

// attribution builds the credit of tiles' source, at the view's bottom
// right.
func attribution(c *ui.Context, tiles *Tiles) {
	if tiles.Attribution == "" {
		return
	}
	t := c.Theme()
	ui.Row(c).Absolute().Right(0).Bottom(0).Padding(t.Space(0.5), t.Space(1.5)).
		Background(t.Background.Alpha(0.8)).Children(func() {
		if tiles.AttributionURL != "" {
			ui.Link(c, tiles.Attribution, tiles.AttributionURL).FontSize(t.FontSize - 2)
		} else {
			ui.Text(c, tiles.Attribution).FontSize(t.FontSize - 2).TextColor(t.TextMuted)
		}
	})
}
