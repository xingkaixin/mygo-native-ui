// Package meteogram shows a weather forecast hour by hour, as the
// meteograms of weather services do: the sky as a symbol along the top,
// the temperature as a line, red above freezing and blue below, the
// precipitation as bars against the scale on the right, the wind as
// arrows flying with it and its speed along the bottom, and the nights
// shaded, a line between the days. Hovering shows an hour's numbers.
//
//	meteogram.View(c, app.forecast).Height(320)
//
// Temperatures are in degrees Celsius, precipitation in millimeters an
// hour, wind in meters a second.
package meteogram

import (
	"fmt"
	"math"
	"time"

	"github.com/egoist/mygo/ui"
)

// Sky is the weather of an hour as its symbol shows it.
type Sky int

const (
	Clear Sky = iota
	PartlyCloudy
	Cloudy
	Fog
	Rain
	Snow
	Thunder
)

var skyNames = []string{"Clear", "Partly cloudy", "Cloudy", "Fog", "Rain", "Snow", "Thunder"}

func (s Sky) String() string { return skyNames[s] }

// Hour is the forecast of an hour: when it starts, its temperature,
// precipitation and wind, the direction the wind comes from in degrees,
// 0 from the north and 90 from the east, its sky, and whether it is
// dark, when Clear shows a moon.
type Hour struct {
	Time     time.Time
	Temp     float64
	Precip   float64
	Wind     float64
	WindFrom float64
	Sky      Sky
	Night    bool
}

// ticks returns round values from lo to hi about every step, a whole
// number of steps of 1, 2 or 5 times a power of ten.
func ticks(lo, hi float64, n int) (float64, float64, float64) {
	raw := (hi - lo) / float64(n)
	mag := math.Pow(10, math.Floor(math.Log10(raw)))
	step := mag
	for _, f := range []float64{1, 2, 5, 10} {
		if f*mag >= raw {
			step = f * mag
			break
		}
	}
	return math.Floor(lo/step) * step, math.Ceil(hi/step) * step, step
}

var (
	warm  = ui.RGB(214, 48, 49)
	cold  = ui.RGB(40, 110, 220)
	rainC = ui.RGB(54, 150, 230)
)

// View creates the forecast of hours, one after another. Give it a
// height, or grow it in its parent.
func View(c *ui.Context, hours []Hour) ui.Element {
	t := c.Theme()
	view := ui.Box(c).Role(ui.RoleImage).Label("Weather forecast")
	if len(hours) == 0 {
		return view
	}
	b := view.LayoutBounds()
	const left, right, top, windH, timeH = 36, 36, 34, 36, 34
	plotW := max(1, b.W-left-right)
	dx := plotW / float32(len(hours))
	hover := -1
	if px, py, ok := view.PointerPosition(); ok && px >= left && px < left+plotW && py < b.H-timeH {
		hover = int((px - left) / dx)
	}
	lo, hi := math.Inf(1), math.Inf(-1)
	wet := 0.0
	for _, h := range hours {
		lo, hi = min(lo, h.Temp), max(hi, h.Temp)
		wet = max(wet, h.Precip)
	}
	lo, hi, step := ticks(lo-1, hi+1, 4)
	wetTop, wetStep := 4.0, 1.0
	if wet > 4 {
		_, wetTop, wetStep = ticks(0, wet, 4)
	}
	view.Draw(func(p *ui.Painter, r ui.Rect) {
		x0, y0 := r.X+left, r.Y+top
		plotH := max(1, r.H-top-windH-timeH)
		colX := func(i int) float32 { return x0 + float32(i)*dx }
		tempY := func(v float64) float32 { return y0 + plotH - float32((v-lo)/(hi-lo))*plotH }
		muted, grid := t.TextMuted, t.Text.Alpha(0.08)
		small := t.FontSize - 2
		for i := 0; i < len(hours); {
			if !hours[i].Night {
				i++
				continue
			}
			j := i
			for j < len(hours) && hours[j].Night {
				j++
			}
			p.Fill(ui.Rect{X: colX(i), Y: r.Y, W: colX(j) - colX(i), H: r.H - timeH}, t.Text.Alpha(0.045), 0)
			i = j
		}
		for v := lo; v <= hi+step/2; v += step {
			y := tempY(v)
			col := grid
			if v == 0 {
				col = cold.Alpha(0.5)
			}
			p.Line(x0, y, x0+plotW, y, 1, col)
			label := fmt.Sprintf("%g°", v)
			w, h := p.MeasureText(0, ui.Span{Text: label, Size: small})
			p.RichText(x0-6-w, y-h/2, 0, ui.Span{Text: label, Size: small, Color: muted, Features: "tnum"})
		}
		for v := wetStep; v <= wetTop+wetStep/2; v += wetStep {
			y := y0 + plotH - float32(v/wetTop)*plotH
			label := fmt.Sprintf("%g", v)
			_, h := p.MeasureText(0, ui.Span{Text: label, Size: small})
			p.RichText(x0+plotW+6, y-h/2, 0, ui.Span{Text: label, Size: small, Color: rainC, Features: "tnum"})
		}
		p.RichText(x0+plotW+6, y0+plotH+2, 0, ui.Span{Text: "mm", Size: small, Color: rainC})
		for i, h := range hours {
			if h.Precip <= 0 {
				continue
			}
			bh := float32(min(h.Precip, wetTop)/wetTop) * plotH
			gap := min(2, dx*0.15)
			p.Fill(ui.Rect{X: colX(i) + gap, Y: y0 + plotH - bh, W: dx - 2*gap, H: bh}, rainC.Alpha(0.75), 1)
		}
		pts := make([][2]float32, len(hours))
		for i, h := range hours {
			pts[i] = [2]float32{colX(i) + dx/2, tempY(h.Temp)}
		}
		zero := tempY(0)
		above := ui.Rect{X: x0, Y: r.Y, W: plotW, H: zero - r.Y}
		below := ui.Rect{X: x0, Y: zero, W: plotW, H: r.Y + r.H - zero}
		p.Clip(above, 0, func() { p.StrokePolyline(pts, 2.5, warm) })
		p.Clip(below, 0, func() { p.StrokePolyline(pts, 2.5, cold) })
		every := max(1, int(math.Ceil(float64(36/dx))))
		for i := 0; i < len(hours); i += every {
			n := min(every, len(hours)-i)
			cx := (colX(i) + colX(i+n)) / 2
			symbol(p, hours[i].Sky, hours[i].Night, cx, r.Y+top/2, 13)
		}
		windY := y0 + plotH + windH/2 + 2
		every = max(1, int(math.Ceil(float64(38/dx))))
		for i := 0; i < len(hours); i += every {
			h := hours[i]
			cx := colX(i) + dx/2
			arrow(p, cx-8, windY, h.WindFrom, muted)
			label := fmt.Sprintf("%.0f", h.Wind)
			p.RichText(cx+2, windY-7, 0, ui.Span{Text: label, Size: small, Color: t.Text, Features: "tnum"})
		}
		ty := r.Y + r.H - timeH + 4
		for i, h := range hours {
			x := colX(i)
			if h.Time.Hour() == 0 {
				p.Line(x, r.Y, x, ty+timeH-6, 1, t.Text.Alpha(0.25))
				day := h.Time.Format("Mon 2")
				p.RichText(x+4, ty+14, 0, ui.Span{Text: day, Size: small, Weight: 600, Color: t.Text})
			}
			if h.Time.Hour()%6 == 0 && dx*6 > 24 {
				label := h.Time.Format("15")
				w, _ := p.MeasureText(0, ui.Span{Text: label, Size: small})
				p.RichText(x-w/2, ty, 0, ui.Span{Text: label, Size: small, Color: muted, Features: "tnum"})
			}
		}
		if hover >= 0 && hover < len(hours) {
			h := hours[hover]
			cx := colX(hover) + dx/2
			p.Line(cx, r.Y+top, cx, y0+plotH+windH, 1, t.Text.Alpha(0.4))
			dot := ui.Rect{X: cx - 4, Y: tempY(h.Temp) - 4, W: 8, H: 8}
			col := warm
			if h.Temp < 0 {
				col = cold
			}
			p.Fill(dot, col, 4)
			p.Stroke(dot, t.Background, 4, 1.5)
			tip(p, t, r, cx, y0+8, h)
		}
	})
	return view
}

// tip draws the numbers of h in a box beside x.
func tip(p *ui.Painter, t *ui.Theme, r ui.Rect, x, y float32, h Hour) {
	lines := []ui.Span{
		{Text: h.Time.Format("Mon 15:04") + "\n", Weight: 600},
		{Text: fmt.Sprintf("%s\n%.1f°  ·  %.1f mm\nWind %.0f m/s from %s", h.Sky, h.Temp, h.Precip, h.Wind, compass(h.WindFrom)), Color: t.TextMuted},
	}
	for i := range lines {
		lines[i].Size = t.FontSize - 1
		lines[i].Features = "tnum"
	}
	w, hh := p.MeasureText(0, lines...)
	box := ui.Rect{X: x + 12, Y: y, W: w + 20, H: hh + 14}
	if box.X+box.W > r.X+r.W {
		box.X = x - 12 - box.W
	}
	p.Shadow(box, 6, 0, 2, 10, 0, ui.RGB(0, 0, 0).Alpha(0.18))
	p.Fill(box, t.Surface, 6)
	p.Stroke(box, t.Border, 6, 1)
	p.RichText(box.X+10, box.Y+7, 0, lines...)
}

// compass returns the point of the compass deg is nearest, as "SW".
func compass(deg float64) string {
	points := []string{"N", "NE", "E", "SE", "S", "SW", "W", "NW"}
	return points[int(math.Round(math.Mod(deg+360, 360)/45))%8]
}

// arrow draws an arrow at x, y flying with a wind from deg.
func arrow(p *ui.Painter, x, y float32, deg float64, col ui.Color) {
	p.Transform(ui.Rotation(float32(deg+180)), x, y, func() {
		var path ui.Path
		path.MoveTo(x, y+7).LineTo(x, y-6)
		p.StrokePath(&path, 1.5, col)
		var head ui.Path
		head.MoveTo(x, y-8).LineTo(x-4, y-2).LineTo(x+4, y-2).Close()
		p.FillPath(&head, col)
	})
}

// symbol draws the symbol of sky at x, y, s its radius.
func symbol(p *ui.Painter, sky Sky, night bool, x, y, s float32) {
	sun, moon := ui.RGB(250, 180, 30), ui.RGB(200, 205, 225)
	light, dark := ui.RGB(196, 202, 212), ui.RGB(126, 134, 148)
	body := func(cx, cy, rad float32) {
		if night {
			crescent(p, cx, cy, rad, moon)
			return
		}
		for i := range 8 {
			a := float64(i) * math.Pi / 4
			c, sn := float32(math.Cos(a)), float32(math.Sin(a))
			p.StrokePolyline([][2]float32{{cx + c*rad*1.25, cy + sn*rad*1.25}, {cx + c*rad*1.6, cy + sn*rad*1.6}}, 1.5, sun)
		}
		var disc ui.Path
		p.FillPath(disc.Circle(cx, cy, rad), sun)
	}
	switch sky {
	case Clear:
		body(x, y, s*0.55)
	case PartlyCloudy:
		body(x-s*0.3, y-s*0.3, s*0.45)
		cloud(p, x+s*0.15, y+s*0.2, s*0.8, light)
	case Cloudy:
		cloud(p, x+s*0.25, y-s*0.2, s*0.6, dark)
		cloud(p, x-s*0.1, y+s*0.15, s*0.85, light)
	case Fog:
		for i := range 4 {
			yy := y - s*0.6 + float32(i)*s*0.4
			off := float32(i%2) * s * 0.2
			p.StrokePolyline([][2]float32{{x - s*0.65 + off, yy}, {x + s*0.4 + off, yy}}, 2, dark)
		}
	case Rain, Snow, Thunder:
		cloud(p, x, y-s*0.25, s*0.85, dark)
		for i := range 3 {
			dx := float32(i-1) * s * 0.45
			switch sky {
			case Rain:
				p.StrokePolyline([][2]float32{{x + dx + 2, y + s*0.35}, {x + dx - 1, y + s*0.8}}, 1.8, rainC)
			case Snow:
				var flake ui.Path
				p.FillPath(flake.Circle(x+dx, y+s*0.6, 2), ui.RGB(150, 190, 235))
			}
		}
		if sky == Thunder {
			var bolt ui.Path
			bolt.MoveTo(x+s*0.1, y+s*0.15).LineTo(x-s*0.25, y+s*0.6).LineTo(x, y+s*0.6).
				LineTo(x-s*0.15, y+s*1.0).LineTo(x+s*0.3, y+s*0.45).LineTo(x+s*0.05, y+s*0.45).Close()
			p.FillPath(&bolt, sun)
		}
	}
}

// cloud draws a cloud w wide whose middle is at x, y.
func cloud(p *ui.Painter, x, y, w float32, col ui.Color) {
	var path ui.Path
	path.Circle(x-w*0.45, y+w*0.12, w*0.28)
	path.Circle(x-w*0.08, y-w*0.12, w*0.4)
	path.Circle(x+w*0.4, y+w*0.08, w*0.32)
	path.MoveTo(x-w*0.45, y+w*0.4).LineTo(x-w*0.45, y+w*0.12).LineTo(x+w*0.4, y+w*0.12).LineTo(x+w*0.4, y+w*0.4).Close()
	p.FillPath(&path, col)
}

// crescent draws a waxing moon of radius rad at x, y.
func crescent(p *ui.Painter, x, y, rad float32, col ui.Color) {
	// 4/3 of the radius out puts a cubic's control points where it
	// follows a half circle closely.
	k := rad * 4 / 3
	var path ui.Path
	path.MoveTo(x, y-rad).
		CubeTo(x+k, y-rad, x+k, y+rad, x, y+rad).
		CubeTo(x+k*0.45, y+rad*0.75, x+k*0.45, y-rad*0.75, x, y-rad).Close()
	p.FillPath(&path, col)
}
