// Package worldmap shows a value for each country on a map of the world,
// as Our World in Data's maps do: each country filled from light to deep
// by its value, on a linear or a logarithmic scale, those without one
// muted, and a legend of the lowest and highest values below. The pointer
// on a country lights its outline and tells its name and value; a click
// chooses it, which Chosen returns. The wheel zooms around the pointer,
// dragging pans, and a double click shows the whole world again.
//
//	worldmap.View(c, &app.world, app.population, worldmap.Log, formatPeople).Grow(1)
//	if code, ok := app.world.Chosen(); ok {
//		... // show worldmap.Name(code) and app.population[code]
//	}
//
// It draws the 177 countries of Natural Earth's 1:110m Admin 0 –
// Countries, which is in the public domain, in the Equal Earth projection.
package worldmap

import (
	_ "embed"
	"image"
	"image/draw"
	"math"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/egoist/mygo/ui"
	"golang.org/x/image/vector"
)

//go:embed countries.txt
var data string

// country is a country of the map: its code, its name, the rings of its
// borders in units of the projection, y down, and the box around them,
// its left, top, right and bottom.
type country struct {
	code, name string
	rings      [][][2]float32
	box        [4]float32
}

// The projection's width and height: the map is mapW by mapH units, its
// top left at 0, 0.
var mapW, mapH = float32(2 * project(180, 0)[0]), float32(2 * project(0, 90)[1])

// project returns where the point of longitude lon and latitude lat, in
// degrees, is in Equal Earth, as Šavrič, Patterson and Jenny define it,
// with the origin at the middle and y up.
func project(lon, lat float64) [2]float64 {
	const a1, a2, a3, a4 = 1.340264, -0.081106, 0.000893, 0.003796
	m := math.Sqrt(3) / 2
	th := math.Asin(m * math.Sin(lat*math.Pi/180))
	t2 := th * th
	t6 := t2 * t2 * t2
	x := lon * math.Pi / 180 * math.Cos(th) / (m * (a1 + 3*a2*t2 + t6*(7*a3+9*a4*t2)))
	y := th * (a1 + a2*t2 + t6*(a3+a4*t2))
	return [2]float64{x, y}
}

// load parses data: the countries, and their indexes by code.
var load = sync.OnceValues(func() ([]country, map[string]int) {
	var countries []country
	byCode := map[string]int{}
	for line := range strings.Lines(data) {
		line = strings.TrimSpace(line)
		switch {
		case line == "" || line[0] == '#':
		case line[0] >= 'A' && line[0] <= 'Z':
			code, name, _ := strings.Cut(line, " ")
			byCode[code] = len(countries)
			inf := float32(math.Inf(1))
			countries = append(countries, country{code: code, name: name, box: [4]float32{inf, inf, -inf, -inf}})
		default:
			ct := &countries[len(countries)-1]
			nums := strings.Fields(line)
			ring := make([][2]float32, 0, len(nums)/2)
			lon, lat := 0, 0
			for i := 0; i+1 < len(nums); i += 2 {
				dx, _ := strconv.Atoi(nums[i])
				dy, _ := strconv.Atoi(nums[i+1])
				lon, lat = lon+dx, lat+dy
				p := project(float64(lon)/100, float64(lat)/100)
				q := [2]float32{float32(p[0]) + mapW/2, mapH/2 - float32(p[1])}
				ring = append(ring, q)
				ct.box = [4]float32{min(ct.box[0], q[0]), min(ct.box[1], q[1]), max(ct.box[2], q[0]), max(ct.box[3], q[1])}
			}
			ct.rings = append(ct.rings, ring)
		}
	}
	return countries, byCode
})

// contains reports whether x, y is in the country, inside an odd number
// of its rings, so that a hole, as Lesotho in South Africa, is not.
func (ct *country) contains(x, y float32) bool {
	if x < ct.box[0] || y < ct.box[1] || x > ct.box[2] || y > ct.box[3] {
		return false
	}
	in := false
	for _, ring := range ct.rings {
		for i, a := range ring {
			b := ring[(i+len(ring)-1)%len(ring)]
			if (a[1] > y) != (b[1] > y) && x < a[0]+(y-a[1])*(b[0]-a[0])/(b[1]-a[1]) {
				in = !in
			}
		}
	}
	return in
}

// Name returns the English name of the country of code, its ISO 3166-1
// alpha-3 code, as "Norway" of "NOR", or "" for a code the map has not.
func Name(code string) string {
	countries, byCode := load()
	if i, ok := byCode[code]; ok {
		return countries[i].name
	}
	return ""
}

// Scale is how values spread over the colors, from the lowest value's
// to the highest's.
type Scale uint8

const (
	// Linear spreads values by their size.
	Linear Scale = iota
	// Log spreads them by their logarithm, for values of many sizes, as
	// people by country. Values of 0 and less take the lowest's color.
	Log
)

// State is what a view keeps: the country chosen, and how far it is
// zoomed and panned. Keep it in the app's state, one for each view.
type State struct {
	chosen string
	// zoom is how many times the fit the map shows, 0 the fit, and x, y the
	// point of the map at the view's center.
	zoom, x, y float32
	// w, h are the view's size as of the last frame, and moved how far the
	// pointer pressing it went: far, the press pans rather than clicks.
	w, h, moved float32
}

// Chosen returns the code of the country chosen, or false with none.
func (s *State) Chosen() (string, bool) { return s.chosen, s.chosen != "" }

// maxZoom is how far a view zooms in: 16 times the fit.
const maxZoom = 16

// scale returns the DIPs of the view a unit of the map takes.
func (s *State) scale() float32 { return min(s.w/mapW, s.h/mapH) * max(1, s.zoom) }

// zoomAt zooms to z times the fit, keeping the point of the map at px, py
// of the view where it is.
func (s *State) zoomAt(z, px, py float32) {
	k := s.scale()
	if k <= 0 {
		return
	}
	mx, my := s.x+(px-s.w/2)/k, s.y+(py-s.h/2)/k
	s.zoom = max(1, min(maxZoom, z))
	k = s.scale()
	s.x, s.y = mx-(px-s.w/2)/k, my-(py-s.h/2)/k
	s.clamp()
}

// clamp keeps the map's edges outside the view, or centers it along an
// axis it does not fill.
func (s *State) clamp() {
	if k := s.scale(); k > 0 {
		s.x = clampAxis(s.x, mapW, s.w/k)
		s.y = clampAxis(s.y, mapH, s.h/k)
	}
}

// clampAxis keeps the center c of a span of the view seen showing span
// units of the map, size long, within it, or at its middle.
func clampAxis(c, size, span float32) float32 {
	if span >= size {
		return size / 2
	}
	return max(span/2, min(size-span/2, c))
}

// View creates the map of values, by the countries' ISO 3166-1 alpha-3
// codes, spread over the colors by scale, the values formatted with
// format, the numbers with none. Give it a size, or Grow.
func View(c *ui.Context, s *State, values map[string]float64, scale Scale, format func(float64) string) ui.Element {
	t := c.Theme()
	countries, byCode := load()
	if format == nil {
		format = func(v float64) string { return strconv.FormatFloat(v, 'g', 4, 64) }
	}
	lo, hi := extent(values, scale)
	dark := t.Dark
	fills := make([]ui.Color, len(countries))
	for i, ct := range countries {
		fills[i] = t.Text.Alpha(0.1)
		if v, ok := values[ct.code]; ok {
			fills[i] = shade(position(v, lo, hi, scale), dark)
		}
	}
	return ui.Column(c).Children(func() {
		box := ui.Box(c).Grow(1).Clip().Role(ui.RoleImage).Label("World map")
		b := box.LayoutBounds()
		s.w, s.h = b.W, b.H
		if s.zoom == 0 {
			s.x, s.y = mapW/2, mapH/2
		}
		s.clamp()
		if dx, dy, ok := box.Dragged(); ok {
			s.moved += float32(math.Abs(float64(dx)) + math.Abs(float64(dy)))
			if s.moved > 0 && s.zoom > 1 {
				k := s.scale()
				s.x, s.y = s.x-dx/k, s.y-dy/k
				s.clamp()
				box.Cursor(ui.CursorGrabbing)
			}
		}
		if box.DoubleClicked() {
			s.zoom = 0
		}
		// Scrolling comes between builds, at the box laid out since, which
		// only a handle reads then.
		ref := ui.Local(box, "worldmap.handle", func() ui.Handle { return ui.Handle{} })
		box.Bind(ref)
		box.HandleInput(func(ev ui.InputEvent) bool {
			if ev.Kind != ui.InputScroll {
				return false
			}
			b := ref.Bounds(c)
			s.w, s.h = b.W, b.H
			// Scrolling 300 DIPs zooms by 2, as zoomview does.
			s.zoomAt(max(1, s.zoom)*float32(math.Exp2(float64(-ev.DY)/300)), ev.X, ev.Y)
			return true
		})
		k := s.scale()
		hovered := -1
		if x, y, ok := box.PointerPosition(); ok && k > 0 {
			mx, my := s.x+(x-s.w/2)/k, s.y+(y-s.h/2)/k
			for i := range countries {
				if countries[i].contains(mx, my) {
					hovered = i
					break
				}
			}
		}
		// A press that moved a few DIPs is a drag, not a click.
		if box.Clicked() && s.moved < 4 {
			s.chosen = ""
			if hovered >= 0 {
				s.chosen = countries[hovered].code
			}
		}
		if !box.Pressed() {
			s.moved = 0
		}
		chosen := -1
		if i, ok := byCode[s.chosen]; ok {
			chosen = i
		}
		cx, cy, zoom := s.x, s.y, max(1, s.zoom)
		ras := ui.Local(box, "worldmap.raster", func() raster { return raster{} })
		box.Draw(func(p *ui.Painter, r ui.Rect) {
			if r.W != b.W || r.H != b.H {
				c.Invalidate()
			}
			// The bitmap covers r as Painter.Image places it, on whole pixels.
			sc := p.Scale()
			x0, y0 := round(r.X*sc), round(r.Y*sc)
			k := min(r.W/mapW, r.H/mapH) * zoom
			f := frame{
				w: int(round((r.X+r.W)*sc) - x0), h: int(round((r.Y+r.H)*sc) - y0),
				ox: (r.X+r.W/2-cx*k)*sc - x0, oy: (r.Y+r.H/2-cy*k)*sc - y0, k: k * sc, scale: sc,
				hovered: hovered, chosen: chosen, border: t.Background, outline: t.Text,
			}
			if bm := ras.draw(f, fills); bm != nil {
				p.Image(bm, r, ui.FillBox)
			}
		})
		ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Padding(t.Space(2), t.Space(3)).Children(func() {
			line, i := "", hovered
			if i < 0 {
				i = chosen
			}
			if i >= 0 {
				line = countries[i].name + "  ·  No data"
				if v, ok := values[countries[i].code]; ok {
					line = countries[i].name + "  ·  " + format(v)
				}
			}
			ui.Text(c, line).FontSize(t.FontSize - 1).TextColor(t.TextMuted).FontFeatures("tnum").
				SingleLine().Grow(1).Shrink(1)
			if hi < lo {
				return
			}
			small := t.FontSize - 2
			ui.Text(c, format(lo)).FontSize(small).TextColor(t.TextMuted).FontFeatures("tnum")
			stops := make([]ui.GradientStop, 9)
			for i := range stops {
				f := float32(i) / float32(len(stops)-1)
				stops[i] = ui.GradientStop{Color: shade(f, dark), At: f}
			}
			ui.Box(c).Size(t.Space(40), t.Space(2)).Draw(func(p *ui.Painter, r ui.Rect) {
				p.FillGradient(r, ui.LinearGradient{Angle: 90, Stops: stops}, 2)
			})
			ui.Text(c, format(hi)).FontSize(small).TextColor(t.TextMuted).FontFeatures("tnum")
		})
	})
}

// extent returns the lowest and highest of values that scale shows, those
// above 0 on a Log scale, or hi below lo with none.
func extent(values map[string]float64, scale Scale) (lo, hi float64) {
	lo, hi = math.Inf(1), math.Inf(-1)
	for _, v := range values {
		if scale == Log && v <= 0 || math.IsNaN(v) {
			continue
		}
		lo, hi = min(lo, v), max(hi, v)
	}
	return lo, hi
}

// position returns where v is between lo and hi on scale, from 0 to 1.
func position(v, lo, hi float64, scale Scale) float32 {
	if scale == Log {
		if v <= 0 {
			return 0
		}
		v, lo, hi = math.Log(v), math.Log(lo), math.Log(hi)
	}
	if hi <= lo {
		return 1
	}
	return float32(max(0, min(1, (v-lo)/(hi-lo))))
}

// shade returns the color at f of the scale, from 0, the lowest value's,
// light, to 1, the highest's, deep: in the dark theme, from dim to bright.
func shade(f float32, dark bool) ui.Color {
	c := ui.Oklch(0.95-0.62*f, 0.03+0.14*f, 255)
	if dark {
		c = ui.Oklch(0.3+0.62*f, 0.06+0.08*f, 255)
	}
	// The map is drawn in sRGB: the legend shows the same colors.
	return ui.RGB(c.R, c.G, c.B)
}

func round(v float32) float32 { return float32(math.Round(float64(v))) }

// frame is what a map shows: its size in pixels, where the map's 0, 0 is
// in it and the pixels a unit of the map takes, the pixels of a DIP, the
// countries lit, and the colors of their borders and outlines.
type frame struct {
	w, h             int
	ox, oy, k, scale float32
	hovered, chosen  int
	border, outline  ui.Color
}

// raster is a map drawn in pixels on the CPU, as path masks on the GPU
// would fill the atlas zoomed in or on a large screen, kept while what it
// shows holds. It draws in turns into two images, since a bitmap shows an
// image's pixels until the next Update.
type raster struct {
	f     frame
	fills []ui.Color
	bm    *ui.Bitmap
	pix   [2]*image.RGBA
	mask  *image.Alpha
	// z holds a rasterizer for each band of rows, which draw at once.
	z []vector.Rasterizer
}

// draw returns the bitmap of f, the countries filled with fills.
func (ras *raster) draw(f frame, fills []ui.Color) *ui.Bitmap {
	if f.w <= 0 || f.h <= 0 {
		return nil
	}
	if ras.bm != nil && f == ras.f && slices.Equal(fills, ras.fills) {
		return ras.bm
	}
	ras.f, ras.fills = f, fills
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

// paint draws the rows band of img with z: the countries filled, their
// borders, and the outlines of those lit.
func (ras *raster) paint(z *vector.Rasterizer, img *image.RGBA, band image.Rectangle) {
	f := ras.f
	countries, _ := load()
	// box returns the pixels of country i and its outline in band.
	pad := float32(int(f.scale) + 2)
	box := func(i int) image.Rectangle {
		b := countries[i].box
		return image.Rect(int(f.ox+b[0]*f.k-pad), int(f.oy+b[1]*f.k-pad), int(f.ox+b[2]*f.k+pad), int(f.oy+b[3]*f.k+pad)).Intersect(band)
	}
	// add adds the rings of the countries ids to z, which covers r: their
	// areas, or with hw above 0 their outlines, hw pixels to either side.
	add := func(r image.Rectangle, ids []int, hw float32) {
		z.Reset(r.Dx(), r.Dy())
		at := func(q [2]float32) [2]float32 {
			return [2]float32{f.ox - float32(r.Min.X) + q[0]*f.k, f.oy - float32(r.Min.Y) + q[1]*f.k}
		}
		for _, i := range ids {
			for _, ring := range countries[i].rings {
				if hw > 0 {
					for j, q := range ring {
						line(z, at(ring[(j+len(ring)-1)%len(ring)]), at(q), hw)
					}
					continue
				}
				p := at(ring[0])
				z.MoveTo(p[0], p[1])
				for _, q := range ring[1:] {
					p = at(q)
					z.LineTo(p[0], p[1])
				}
				z.ClosePath()
			}
		}
	}
	stroke := func(ids []int, width float32, c ui.Color) {
		var r image.Rectangle
		for _, i := range ids {
			r = r.Union(box(i))
		}
		if !r.Empty() {
			add(r, ids, width*f.scale/2)
			ras.fill(z, img, r, c)
		}
	}
	var shown []int
	for i := range countries {
		if r := box(i); !r.Empty() {
			shown = append(shown, i)
			add(r, []int{i}, 0)
			ras.fill(z, img, r, ras.fills[i])
		}
	}
	stroke(shown, 0.75, f.border)
	// A lit outline goes on a wider one of the borders' color, which shows
	// it on fills as light or as deep as it is.
	if f.hovered >= 0 && f.hovered != f.chosen {
		stroke([]int{f.hovered}, 3.5, f.border)
		stroke([]int{f.hovered}, 1.5, f.outline)
	}
	if f.chosen >= 0 {
		stroke([]int{f.chosen}, 4, f.border)
		stroke([]int{f.chosen}, 2, f.outline)
	}
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
