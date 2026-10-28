// Package avionics draws a primary flight display, as glass cockpits show
// an aircraft's flight: the attitude indicator across the whole display,
// its horizon turning with the roll and moving with the pitch behind a
// ladder of pitch lines and under an arc of bank angles; the airspeed tape
// on the left, colored by the aircraft's V-speeds; the altitude tape and
// the vertical speed on the right; and the heading tape along the bottom.
// Each tape scrolls behind a box that reads out its value.
//
//	avionics.View(c, avionics.Flight{Pitch: 3, Roll: -15, Heading: 270,
//		Airspeed: 112, Altitude: 4500, VerticalSpeed: 500}, avionics.C172)
//
// The display only shows; the app moves the flight each frame.
package avionics

import (
	"fmt"
	"math"

	"github.com/egoist/mygo/ui"
)

// Flight is what the display shows.
type Flight struct {
	// Pitch is degrees nose up, and Roll degrees right wing down.
	Pitch, Roll float64
	// Heading is degrees from north.
	Heading float64
	// Airspeed is knots indicated.
	Airspeed float64
	// Altitude is feet, and VerticalSpeed feet per minute up.
	Altitude, VerticalSpeed float64
}

// Speeds are an aircraft's V-speeds in knots, which color the airspeed
// tape: white from stalling with flaps out (VS0) to the top speed for
// flaps (VFE), green from stalling clean (VS1) to the top speed for rough
// air (VNO), yellow to the speed never to exceed (VNE), and red above it.
// The zero value colors nothing.
type Speeds struct {
	VS0, VS1, VFE, VNO, VNE float64
}

// C172 are the V-speeds of a Cessna 172S.
var C172 = Speeds{VS0: 40, VS1: 48, VFE: 85, VNO: 129, VNE: 163}

var (
	sky     = ui.RGB(36, 110, 214)
	skyLow  = ui.RGB(110, 170, 236)
	ground  = ui.RGB(150, 98, 50)
	groundL = ui.RGB(112, 70, 32)
	white   = ui.RGB(255, 255, 255)
	black   = ui.RGB(0, 0, 0)
	yellow  = ui.RGB(255, 204, 0)
	tape    = ui.RGBA(0, 0, 0, 0.38)
	magenta = ui.RGB(232, 72, 220)
)

// View creates the display of f, its airspeed tape colored by v.
func View(c *ui.Context, f Flight, v Speeds) ui.Element {
	e := ui.Box(c).Background(black).Clip().Role(ui.RoleImage).
		Label(fmt.Sprintf("Pitch %.0f°, bank %.0f°, heading %03.0f°, %.0f knots, %.0f feet", f.Pitch, f.Roll, f.Heading, f.Airspeed, f.Altitude))
	e.Draw(func(p *ui.Painter, r ui.Rect) {
		d := display{p: p, r: r, f: f, v: v, font: max(10, r.H/30)}
		d.cx, d.cy = r.X+r.W*0.48, r.Y+r.H*0.44
		d.attitude()
		d.speedTape()
		d.altitudeTape()
		d.headingTape()
	})
	return e
}

type display struct {
	p      *ui.Painter
	r      ui.Rect
	f      Flight
	v      Speeds
	font   float32
	cx, cy float32
}

func (d display) text(s string, size float32, col ui.Color) ui.Span {
	return ui.Span{Text: s, Size: size, Color: col, Weight: 600, Features: "tnum"}
}

// put paints a span with its anchor at x, y: ax and ay are 0, 0.5 or 1 for
// the left, middle or right, and the top, middle or bottom.
func (d display) put(sp ui.Span, x, y, ax, ay float32) {
	w, h := d.p.MeasureText(0, sp)
	d.p.RichText(x-w*ax, y-h*ay, 0, sp)
}

func (d display) attitude() {
	p, r := d.p, d.r
	ppd := r.H / 48
	big := float32(math.Hypot(float64(r.W), float64(r.H))) * 2
	p.Transform(ui.Rotation(float32(-d.f.Roll)), d.cx, d.cy, func() {
		hy := d.cy + float32(d.f.Pitch)*ppd
		p.FillGradient(ui.Rect{X: d.cx - big, Y: hy - big, W: 2 * big, H: big},
			ui.LinearGradient{From: sky, To: skyLow, Angle: 180, Stops: []ui.GradientStop{{At: 0, Color: sky}, {At: 0.9, Color: sky}, {At: 1, Color: skyLow}}}, 0)
		p.FillGradient(ui.Rect{X: d.cx - big, Y: hy, W: 2 * big, H: big},
			ui.LinearGradient{From: ground, To: groundL, Angle: 180, Stops: []ui.GradientStop{{At: 0, Color: ground}, {At: 0.1, Color: groundL}, {At: 1, Color: groundL}}}, 0)
		p.StrokePolyline([][2]float32{{d.cx - big, hy}, {d.cx + big, hy}}, 1.5, white)
		for k := -36; k <= 36; k++ {
			deg := float64(k) * 2.5
			if k == 0 || math.Abs(deg-d.f.Pitch) > 12.5 {
				continue
			}
			y := hy - float32(deg)*ppd
			half := r.W * 0.025
			switch {
			case k%4 == 0:
				half = r.W * 0.1
			case k%2 == 0:
				half = r.W * 0.05
			}
			p.StrokePolyline([][2]float32{{d.cx - half, y}, {d.cx + half, y}}, 1.5, white)
			if k%4 == 0 {
				label := d.text(fmt.Sprintf("%d", int(math.Abs(deg))), d.font*0.75, white)
				d.put(label, d.cx-half-d.font*0.4, y, 1, 0.5)
				d.put(label, d.cx+half+d.font*0.4, y, 0, 0.5)
			}
		}
		// The roll pointer turns with the horizon under the fixed arc.
		rr := r.H * 0.3
		s := d.font * 0.5
		p.FillPath(new(ui.Path).MoveTo(d.cx, d.cy-rr+2).LineTo(d.cx-s, d.cy-rr+2+s*1.4).LineTo(d.cx+s, d.cy-rr+2+s*1.4).Close(), white)
	})
	rr := r.H * 0.3
	var arc [][2]float32
	for a := -60.0; a <= 60; a += 2 {
		x, y := polar(d.cx, d.cy, rr, a)
		arc = append(arc, [2]float32{x, y})
	}
	p.StrokePolyline(arc, 1.5, white)
	for _, a := range []float64{-60, -45, -30, -20, -10, 10, 20, 30, 45, 60} {
		long := float32(0.06)
		if a == 30 || a == -30 || a == 60 || a == -60 {
			long = 0.11
		} else if a == 45 || a == -45 {
			long = 0.04
		}
		x0, y0 := polar(d.cx, d.cy, rr, a)
		x1, y1 := polar(d.cx, d.cy, rr*(1+long), a)
		p.StrokePolyline([][2]float32{{x0, y0}, {x1, y1}}, 1.5, white)
	}
	s := d.font * 0.5
	p.FillPath(new(ui.Path).MoveTo(d.cx, d.cy-rr-1).LineTo(d.cx-s, d.cy-rr-1-s*1.4).LineTo(d.cx+s, d.cy-rr-1-s*1.4).Close(), white)
	// The aircraft symbol: two wing bars and a dot, outlined to read on
	// sky and ground alike.
	w := r.W
	for _, side := range []float32{-1, 1} {
		bar := [][2]float32{{d.cx + side*w*0.2, d.cy}, {d.cx + side*w*0.08, d.cy}, {d.cx + side*w*0.08, d.cy + w*0.02}}
		p.StrokePolyline(bar, 6, black)
		p.StrokePolyline(bar, 3.5, yellow)
	}
	p.Fill(ui.Rect{X: d.cx - 4, Y: d.cy - 4, W: 8, H: 8}, black, 1)
	p.Fill(ui.Rect{X: d.cx - 2.5, Y: d.cy - 2.5, W: 5, H: 5}, yellow, 1)
}

// polar returns the point r from cx, cy at deg from straight up, clockwise.
func polar(cx, cy, r float32, deg float64) (float32, float32) {
	a := deg * math.Pi / 180
	return cx + r*float32(math.Sin(a)), cy - r*float32(math.Cos(a))
}

func (d display) speedTape() {
	p, r := d.p, d.r
	box := ui.Rect{X: r.X + r.W*0.04, Y: d.cy - r.H*0.3, W: r.W * 0.11, H: r.H * 0.6}
	ppk := box.H / 70
	ias := d.f.Airspeed
	y := func(kt float64) float32 { return d.cy - float32(kt-ias)*ppk }
	p.Clip(box, 0, func() {
		p.Fill(box, tape, 0)
		strip := box.X + box.W - 6
		band := func(lo, hi float64, x, w float32, col ui.Color) {
			if hi > lo {
				p.Fill(ui.Rect{X: x, Y: y(hi), W: w, H: y(lo) - y(hi)}, col, 0)
			}
		}
		v := d.v
		if v.VNE > 0 {
			band(v.VS1, v.VNO, strip, 6, ui.RGB(40, 200, 60))
			band(v.VNO, v.VNE, strip, 6, yellow)
			band(v.VNE, v.VNE+200, strip, 6, ui.RGB(230, 40, 40))
			band(v.VS0, v.VFE, strip-4, 3, white)
		}
		for kt := math.Floor((ias-40)/5) * 5; kt <= ias+40; kt += 5 {
			if kt < 20 {
				continue
			}
			ty := y(kt)
			long := float32(6)
			if int(kt)%10 == 0 {
				long = 12
				d.put(d.text(fmt.Sprintf("%.0f", kt), d.font*0.85, white), strip-long-d.font*0.4, ty, 1, 0.5)
			}
			p.StrokePolyline([][2]float32{{strip - long, ty}, {strip, ty}}, 1.5, white)
		}
	})
	label := "---"
	if ias >= 20 {
		label = fmt.Sprintf("%.0f", ias)
	}
	d.readout(ui.Rect{X: box.X, Y: d.cy - d.font*0.95, W: box.W * 0.8, H: d.font * 1.9}, 1, label)
}

// readout paints the box that reads a tape's value, its point toward the
// tape's ticks on the side dir (1 right, -1 left).
func (d display) readout(b ui.Rect, dir float32, label string) {
	p := d.p
	tip := b.H * 0.3
	var path ui.Path
	if dir > 0 {
		path.MoveTo(b.X, b.Y).LineTo(b.X+b.W, b.Y).LineTo(b.X+b.W, d.cy-tip).LineTo(b.X+b.W+tip, d.cy).
			LineTo(b.X+b.W, d.cy+tip).LineTo(b.X+b.W, b.Y+b.H).LineTo(b.X, b.Y+b.H).Close()
	} else {
		path.MoveTo(b.X+b.W, b.Y).LineTo(b.X, b.Y).LineTo(b.X, d.cy-tip).LineTo(b.X-tip, d.cy).
			LineTo(b.X, d.cy+tip).LineTo(b.X, b.Y+b.H).LineTo(b.X+b.W, b.Y+b.H).Close()
	}
	p.FillPath(&path, black)
	p.StrokePath(&path, 1.5, white)
	d.put(d.text(label, d.font*1.15, white), b.X+b.W-d.font*0.4, d.cy, 1, 0.5)
}

func (d display) altitudeTape() {
	p, r := d.p, d.r
	box := ui.Rect{X: r.X + r.W*0.78, Y: d.cy - r.H*0.3, W: r.W * 0.12, H: r.H * 0.6}
	ppf := box.H / 700
	alt := d.f.Altitude
	y := func(ft float64) float32 { return d.cy - float32(ft-alt)*ppf }
	p.Clip(box, 0, func() {
		p.Fill(box, tape, 0)
		for ft := math.Floor((alt-400)/20) * 20; ft <= alt+400; ft += 20 {
			ty := y(ft)
			long := float32(5)
			if int(math.Round(ft))%100 == 0 {
				long = 10
				d.put(d.text(fmt.Sprintf("%.0f", ft), d.font*0.85, white), box.X+long+d.font*0.4, ty, 0, 0.5)
			}
			p.StrokePolyline([][2]float32{{box.X, ty}, {box.X + long, ty}}, 1.5, white)
		}
	})
	d.readout(ui.Rect{X: box.X + box.W*0.12, Y: d.cy - d.font*0.95, W: box.W * 0.88, H: d.font * 1.9}, -1,
		fmt.Sprintf("%.0f", math.Round(alt/10)*10))
	vsi := ui.Rect{X: box.X + box.W + r.W*0.008, Y: box.Y + box.H*0.1, W: r.W * 0.04, H: box.H * 0.8}
	p.Fill(vsi, tape, 0)
	vy := func(fpm float64) float32 {
		return d.cy - float32(max(-2000, min(2000, fpm))/2000)*vsi.H/2
	}
	for _, fpm := range []float64{-2000, -1500, -1000, -500, 0, 500, 1000, 1500, 2000} {
		long := float32(4)
		if int(fpm)%1000 == 0 {
			long = 8
			if fpm != 0 {
				d.put(d.text(fmt.Sprintf("%.0f", math.Abs(fpm)/1000), d.font*0.7, white), vsi.X+long+d.font*0.35, vy(fpm), 0, 0.5)
			}
		}
		p.StrokePolyline([][2]float32{{vsi.X, vy(fpm)}, {vsi.X + long, vy(fpm)}}, 1.2, white)
	}
	py := vy(d.f.VerticalSpeed)
	s := d.font * 0.45
	p.FillPath(new(ui.Path).MoveTo(vsi.X+1, py).LineTo(vsi.X+1+s*1.6, py-s).LineTo(vsi.X+1+s*1.6, py+s).Close(), magenta)
	if math.Abs(d.f.VerticalSpeed) >= 100 {
		at, ay := vsi.Y-d.font*0.3, float32(1)
		if d.f.VerticalSpeed < 0 {
			at, ay = vsi.Y+vsi.H+d.font*0.3, 0
		}
		d.put(d.text(fmt.Sprintf("%.0f", math.Round(d.f.VerticalSpeed/50)*50), d.font*0.7, white), vsi.X+vsi.W/2, at, 0.5, ay)
	}
}

func (d display) headingTape() {
	p, r := d.p, d.r
	box := ui.Rect{X: d.cx - r.W*0.26, Y: r.Y + r.H*0.88, W: r.W * 0.52, H: r.H * 0.08}
	ppd := box.W / 60
	hdg := math.Mod(d.f.Heading+360, 360)
	p.Clip(box, 0, func() {
		p.Fill(box, tape, 0)
		for deg := math.Floor((hdg-35)/5) * 5; deg <= hdg+35; deg += 5 {
			x := d.cx + float32(deg-hdg)*ppd
			long := float32(5)
			n := int(math.Mod(deg+360, 360))
			if n%10 == 0 {
				long = 10
				label := fmt.Sprintf("%d", n/10)
				if name, ok := map[int]string{0: "N", 90: "E", 180: "S", 270: "W"}[n]; ok {
					label = name
				}
				d.put(d.text(label, d.font*0.8, white), x, box.Y+long+d.font*0.2, 0.5, 0)
			}
			p.StrokePolyline([][2]float32{{x, box.Y}, {x, box.Y + long}}, 1.5, white)
		}
	})
	b := ui.Rect{X: d.cx - d.font*1.7, Y: box.Y - d.font*1.9, W: d.font * 3.4, H: d.font * 1.5}
	p.Fill(b, black, 2)
	p.Stroke(b, white, 2, 1.5)
	n := int(math.Round(hdg)) % 360
	if n == 0 {
		n = 360
	}
	d.put(d.text(fmt.Sprintf("%03d°", n), d.font, white), d.cx, b.Y+b.H/2, 0.5, 0.5)
	s := d.font * 0.35
	p.FillPath(new(ui.Path).MoveTo(d.cx, box.Y+s*1.2).LineTo(d.cx-s, box.Y).LineTo(d.cx+s, box.Y).Close(), white)
}
