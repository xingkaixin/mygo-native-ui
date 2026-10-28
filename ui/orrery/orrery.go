// Package orrery draws the planets around the Sun on a date, seen from
// above the ecliptic, as an orrery shows them: each on its orbit, with
// distances drawn on a square-root scale so that Mercury and Neptune fit
// together. Scrolling moves the date, a day for every few points, and a
// pinch or Cmd with the wheel zooms in on the inner planets. Hovering a
// planet shows how far it is from the Sun and from the Earth, and how long
// its light takes to reach us.
//
//	orrery.View(c, &app.orrery, &app.date)
//
// Positions come from JPL's Keplerian elements for approximate positions
// of the planets, good to a few minutes of arc between 1800 and 2050.
package orrery

import (
	"fmt"
	"math"
	"time"

	"github.com/egoist/mygo/ui"
)

// Planet is one of the eight planets.
type Planet int

const (
	Mercury Planet = iota
	Venus
	Earth
	Mars
	Jupiter
	Saturn
	Uranus
	Neptune
)

// Planets are the planets from the Sun out.
var Planets = []Planet{Mercury, Venus, Earth, Mars, Jupiter, Saturn, Uranus, Neptune}

func (p Planet) String() string { return planets[p].name }

type planet struct {
	name   string
	radius float32
	color  ui.Color
	// at J2000 and their rates per Julian century: semi-major axis (AU),
	// eccentricity, inclination, mean longitude, longitude of perihelion
	// and longitude of the ascending node (degrees).
	el, rate [6]float64
}

// Earth's elements are those of the Earth–Moon barycenter, as JPL gives
// them; the Earth is never more than 5000 km from it.
var planets = []planet{
	{"Mercury", 3, ui.RGB(160, 154, 148),
		[6]float64{0.38709927, 0.20563593, 7.00497902, 252.25032350, 77.45779628, 48.33076593},
		[6]float64{0.00000037, 0.00001906, -0.00594749, 149472.67411175, 0.16047689, -0.12534081}},
	{"Venus", 4.5, ui.RGB(226, 196, 128),
		[6]float64{0.72333566, 0.00677672, 3.39467605, 181.97909950, 131.60246718, 76.67984255},
		[6]float64{0.00000390, -0.00004107, -0.00078890, 58517.81538729, 0.00268329, -0.27769418}},
	{"Earth", 4.5, ui.RGB(64, 132, 220),
		[6]float64{1.00000261, 0.01671123, -0.00001531, 100.46457166, 102.93768193, 0},
		[6]float64{0.00000562, -0.00004392, -0.01294668, 35999.37244981, 0.32327364, 0}},
	{"Mars", 3.5, ui.RGB(212, 98, 64),
		[6]float64{1.52371034, 0.09339410, 1.84969142, -4.55343205, -23.94362959, 49.55953891},
		[6]float64{0.00001847, 0.00007882, -0.00813131, 19140.30268499, 0.44441088, -0.29257343}},
	{"Jupiter", 8, ui.RGB(206, 168, 128),
		[6]float64{5.20288700, 0.04838624, 1.30439695, 34.39644051, 14.72847983, 100.47390909},
		[6]float64{-0.00011607, -0.00013253, -0.00183714, 3034.74612775, 0.21252668, 0.20469106}},
	{"Saturn", 7, ui.RGB(222, 192, 120),
		[6]float64{9.53667594, 0.05386179, 2.48599187, 49.95424423, 92.59887831, 113.66242448},
		[6]float64{-0.00125060, -0.00050991, 0.00193609, 1222.49362201, -0.41897216, -0.28867794}},
	{"Uranus", 5.5, ui.RGB(130, 206, 214),
		[6]float64{19.18916464, 0.04725744, 0.77263783, 313.23810451, 170.95427630, 74.01692503},
		[6]float64{-0.00196176, -0.00004397, -0.00242939, 428.48202785, 0.40805281, 0.04240589}},
	{"Neptune", 5.5, ui.RGB(72, 112, 220),
		[6]float64{30.06992276, 0.00859048, 1.77004347, -55.12002969, 44.96476227, 131.78422574},
		[6]float64{0.00026291, 0.00005105, 0.00035372, 218.45945325, -0.32241464, -0.00508664}},
}

var j2000 = time.Date(2000, 1, 1, 12, 0, 0, 0, time.UTC)

type orbit struct {
	a, e, i, l, peri, node float64
}

func (p Planet) orbit(at time.Time) orbit {
	cy := at.Sub(j2000).Hours() / 24 / 36525
	pl := planets[p]
	var el [6]float64
	for k := range el {
		el[k] = pl.el[k] + pl.rate[k]*cy
	}
	rad := math.Pi / 180
	return orbit{el[0], el[1], el[2] * rad, el[3] * rad, el[4] * rad, el[5] * rad}
}

// at returns where on the orbit the planet is at eccentric anomaly ea.
func (o orbit) at(ea float64) [3]float64 {
	x := o.a * (math.Cos(ea) - o.e)
	y := o.a * math.Sqrt(1-o.e*o.e) * math.Sin(ea)
	w := o.peri - o.node
	cw, sw := math.Cos(w), math.Sin(w)
	cn, sn := math.Cos(o.node), math.Sin(o.node)
	ci, si := math.Cos(o.i), math.Sin(o.i)
	return [3]float64{
		(cw*cn-sw*sn*ci)*x + (-sw*cn-cw*sn*ci)*y,
		(cw*sn+sw*cn*ci)*x + (-sw*sn+cw*cn*ci)*y,
		sw*si*x + cw*si*y,
	}
}

// Position returns where the planet is at a moment, in AU from the Sun:
// x toward the vernal equinox, y along the ecliptic 90° east of it, z
// toward the ecliptic's north pole.
func (p Planet) Position(at time.Time) [3]float64 {
	o := p.orbit(at)
	m := math.Remainder(o.l-o.peri, 2*math.Pi)
	ea := m + o.e*math.Sin(m)
	for range 8 {
		ea -= (ea - o.e*math.Sin(ea) - m) / (1 - o.e*math.Cos(ea))
	}
	return o.at(ea)
}

// State is what an orrery keeps: how far it is zoomed in. Keep it in the
// app's state, one for each orrery.
type State struct {
	zoom  float64
	hover int
}

// lightMinutes is how many minutes light takes to cross an AU.
const lightMinutes = 149597870.7 / 299792.458 / 60

// View creates the orrery at the date at, which scrolling moves.
func View(c *ui.Context, s *State, at *time.Time) ui.Element {
	t := c.Theme()
	if s.zoom == 0 {
		s.zoom = 1
	}
	view := ui.Box(c).Role(ui.RoleImage).Label("Planets on " + at.Format("2 January 2006")).Clip()
	view.HandleInput(func(in ui.InputEvent) bool {
		switch {
		case in.Kind == ui.InputZoom:
			s.zoom *= float64(in.Scale)
		case in.Kind == ui.InputScroll && in.Mods&ui.Cmd != 0:
			s.zoom *= math.Exp2(-float64(in.DY) / 200)
		case in.Kind == ui.InputScroll:
			d := in.DX
			if math.Abs(float64(in.DY)) > math.Abs(float64(in.DX)) {
				d = in.DY
			}
			*at = at.Add(time.Duration(float64(d) / 3 * float64(24*time.Hour)))
		default:
			return false
		}
		s.zoom = max(1, min(16, s.zoom))
		return true
	})
	b := view.LayoutBounds()
	cx, cy := b.W/2, b.H/2
	// Neptune's aphelion is a little over 30.3 AU.
	k := float32(float64(min(b.W, b.H)/2-t.Space(6)) / math.Sqrt(30.4) * s.zoom)
	project := func(v [3]float64) (float32, float32) {
		r := math.Hypot(v[0], v[1])
		if r == 0 {
			return cx, cy
		}
		f := k * float32(math.Sqrt(r)/r)
		return cx + f*float32(v[0]), cy - f*float32(v[1])
	}
	pos := make([][3]float64, len(Planets))
	xy := make([][2]float32, len(Planets))
	for i, p := range Planets {
		pos[i] = p.Position(*at)
		xy[i][0], xy[i][1] = project(pos[i])
	}
	hover := -1
	if px, py, ok := view.PointerPosition(); ok {
		best := float32(10)
		for i := range xy {
			if d := float32(math.Hypot(float64(px-xy[i][0]), float64(py-xy[i][1]))); d < best {
				best, hover = d, i
			}
		}
	}
	if hover != s.hover {
		s.hover = hover
		c.Invalidate()
	}
	view.Draw(func(p *ui.Painter, r ui.Rect) {
		small := t.FontSize - 2
		for i, pl := range Planets {
			o := pl.orbit(*at)
			pts := make([][2]float32, 0, 181)
			for k := range 181 {
				x, y := project(o.at(float64(k) / 180 * 2 * math.Pi))
				pts = append(pts, [2]float32{r.X + x, r.Y + y})
			}
			col, w := t.Border, float32(1)
			if i == hover {
				col, w = planets[pl].color.Alpha(0.7), 1.5
			}
			p.StrokePolyline(pts, w, col)
		}
		var sun ui.Path
		p.FillPath(sun.Circle(r.X+cx, r.Y+cy, 14), ui.RGB(255, 196, 64).Alpha(0.18))
		sun.Reset()
		p.FillPath(sun.Circle(r.X+cx, r.Y+cy, 7), ui.RGB(255, 190, 48))
		var labels []ui.Rect
		for i, pl := range Planets {
			info := planets[pl]
			x, y := r.X+xy[i][0], r.Y+xy[i][1]
			if pl == Saturn {
				p.StrokePolyline(ellipse(x, y, info.radius*1.9, info.radius*0.7), 1.2, info.color.Alpha(0.8))
			}
			var dot ui.Path
			p.FillPath(dot.Circle(x, y, info.radius), info.color)
			col := t.TextMuted
			if i == hover {
				col = t.Text
			}
			lw, lh := p.MeasureText(0, ui.Span{Text: info.name, Size: small})
			gap := info.radius + t.Space(1.5)
			// Planets near each other in direction would print their names
			// over each other; the first side free of other names wins.
			spots := []ui.Rect{
				{X: x + gap, Y: y - lh/2, W: lw, H: lh},
				{X: x - gap - lw, Y: y - lh/2, W: lw, H: lh},
				{X: x - lw/2, Y: y - gap - lh, W: lw, H: lh},
				{X: x - lw/2, Y: y + gap, W: lw, H: lh},
			}
			spot := spots[0]
			for _, sp := range spots {
				if sp.X >= r.X && sp.X+sp.W <= r.X+r.W && !overlaps(sp, labels) {
					spot = sp
					break
				}
			}
			labels = append(labels, spot)
			p.RichText(spot.X, spot.Y, 0, ui.Span{Text: info.name, Size: small, Color: col})
		}
		p.RichText(r.X+t.Space(4), r.Y+t.Space(3), 0,
			ui.Span{Text: at.Format("2 January 2006"), Size: t.FontSize + 1, Weight: 600, Color: t.Text, Features: "tnum"})
		if hover >= 0 {
			v := pos[hover]
			sun := math.Sqrt(v[0]*v[0] + v[1]*v[1] + v[2]*v[2])
			lines := []string{Planets[hover].String(), fmt.Sprintf("%.3f AU from the Sun", sun)}
			if Planets[hover] != Earth {
				e := pos[Earth]
				d := math.Sqrt((v[0]-e[0])*(v[0]-e[0]) + (v[1]-e[1])*(v[1]-e[1]) + (v[2]-e[2])*(v[2]-e[2]))
				lines = append(lines, fmt.Sprintf("%.3f AU from the Earth", d), "Light takes "+light(d))
			}
			card(p, t, r, r.X+xy[hover][0], r.Y+xy[hover][1], lines, planets[Planets[hover]].color)
		}
	})
	return view
}

// light writes how long light takes to cross d AU.
func light(d float64) string {
	m := d * lightMinutes
	if m < 60 {
		return fmt.Sprintf("%.1f minutes", m)
	}
	return fmt.Sprintf("%dh %02dm", int(m/60), int(m)%60)
}

func overlaps(r ui.Rect, others []ui.Rect) bool {
	for _, o := range others {
		if r.X < o.X+o.W && o.X < r.X+r.W && r.Y < o.Y+o.H && o.Y < r.Y+r.H {
			return true
		}
	}
	return false
}

func ellipse(x, y, rx, ry float32) [][2]float32 {
	pts := make([][2]float32, 0, 41)
	for k := range 41 {
		a := float64(k) / 40 * 2 * math.Pi
		pts = append(pts, [2]float32{x + rx*float32(math.Cos(a)), y + ry*float32(math.Sin(a))})
	}
	return pts
}

// card paints a card of lines beside x, y, the first by a swatch of col.
func card(p *ui.Painter, t *ui.Theme, r ui.Rect, x, y float32, lines []string, col ui.Color) {
	small := t.FontSize - 2
	pad, sw := t.Space(2.5), t.Space(2.5)
	spans := make([]ui.Span, len(lines))
	w, h := float32(0), float32(0)
	for i, l := range lines {
		spans[i] = ui.Span{Text: l, Size: small, Color: t.TextMuted, Features: "tnum"}
		if i == 0 {
			spans[i].Weight, spans[i].Color = 600, t.Text
		}
		lw, lh := p.MeasureText(0, spans[i])
		if i == 0 {
			lw += sw + t.Space(1.5)
		}
		w, h = max(w, lw), h+lh
	}
	w, h = w+2*pad, h+2*pad
	cx, cy := x+t.Space(4), y-h/2
	if cx+w > r.X+r.W {
		cx = x - t.Space(4) - w
	}
	cx, cy = max(r.X, cx), max(r.Y, min(cy, r.Y+r.H-h))
	box := ui.Rect{X: cx, Y: cy, W: w, H: h}
	p.Shadow(box, t.Radius+2, 0, 2, 10, 0, ui.RGBA(0, 0, 0, 0.2))
	p.Fill(box, t.Background, t.Radius+2)
	p.Stroke(box, t.Border, t.Radius+2, 1)
	ty := cy + pad
	for i, sp := range spans {
		tx := cx + pad
		_, lh := p.MeasureText(0, sp)
		if i == 0 {
			p.Fill(ui.Rect{X: tx, Y: ty + (lh-sw)/2, W: sw, H: sw}, col, sw/2)
			tx += sw + t.Space(1.5)
		}
		p.RichText(tx, ty, 0, sp)
		ty += lh
	}
}
