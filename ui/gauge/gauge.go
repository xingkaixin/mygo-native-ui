// Package gauge is a dial with a needle, as dashboards and a car's
// instruments show a reading. Its arc runs from the least value at the
// bottom left round to the most at the bottom right, in the colors of the
// app's zones, with ticks at round values, and the value with its unit
// and a label shows under the hub. The needle swings to a new value with
// a spring, or goes there at once when the desktop asks for less motion.
// A value past either end pins the needle there, while the readout shows
// it as it is. A peak mark can hold the highest value shown until the app
// resets it.
//
//	gauge.View(c, &app.cpuPeak, app.cpu, gauge.Gauge{
//		Label: "CPU", Unit: "%", Max: 100,
//		Zones: []gauge.Zone{
//			{To: 70, Color: t.Success}, {From: 70, To: 90, Color: t.Warning}, {From: 90, To: 100, Color: t.Danger},
//		},
//	})
//	if ui.Button(c, "Reset peak").Clicked() {
//		app.cpuPeak.Reset()
//	}
//
// A gauge fits the size it is given. Its text keeps to sizes that read
// well however small it gets, labeling fewer ticks where they would
// crowd. Assistive technology sees a meter of the value, named by the
// label.
package gauge

import (
	"math"
	"strconv"

	"github.com/egoist/mygo/ui"
)

// Gauge is what a dial shows: its range, its arc and its name.
type Gauge struct {
	// Label names the gauge, under its value and to assistive technology.
	Label string
	// Unit follows the value, as "km/h" or "%".
	Unit     string
	Min, Max float64
	// Sweep is how far round the arc goes, in degrees, open at the
	// bottom: 240 when 0, and from 180, a semicircle, which takes less
	// height, to 270.
	Sweep float64
	// Zones color stretches of the arc, as green, amber and red bands;
	// the arc is faint elsewhere. Without zones, it fills in the accent
	// color up to the needle.
	Zones []Zone
}

// Zone colors the arc from From to To.
type Zone struct {
	From, To float64
	Color    ui.Color
}

// at returns how far v is along the range, from 0 to 1.
func (g Gauge) at(v float64) float64 {
	if g.Max <= g.Min || math.IsNaN(v) {
		return 0
	}
	return max(0, min(1, (v-g.Min)/(g.Max-g.Min)))
}

// marks returns the multiples of step from Min to Max.
func (g Gauge) marks(step float64) []float64 {
	var vs []float64
	for k := math.Ceil(g.Min/step - 1e-9); k*step <= g.Max+step*1e-9; k++ {
		vs = append(vs, k*step)
	}
	return vs
}

// text writes v to about a hundredth of the range: "88", "3450", "4.5".
func (g Gauge) text(v float64) string {
	d := 0
	if span := g.Max - g.Min; span > 0 {
		d = max(0, 1-int(math.Floor(math.Log10(span))))
	}
	return number(v, d)
}

// number writes v with d decimals, and 0 without a minus sign.
func number(v float64, d int) string {
	k := math.Pow(10, float64(d))
	if v = math.Round(v*k) / k; v == 0 {
		v = 0
	}
	return strconv.FormatFloat(v, 'f', d, 64)
}

// State is what a gauge keeps for its peak mark: the highest value it
// showed. Keep it in the app's state, one for each gauge with a peak mark.
type State struct {
	peak float64
	held bool
}

// Reset clears the peak mark, which starts again from the next value
// shown.
func (s *State) Reset() { *s = State{} }

// View creates a gauge of value. s holds its peak mark; nil shows none.
// It is 192 DIPs wide, or as Width sets, and as high as its sweep needs
// at that width; Size fits it within both.
func View(c *ui.Context, s *State, value float64, g Gauge) ui.Element {
	t := c.Theme()
	sweep := g.Sweep
	if sweep == 0 {
		sweep = 240
	}
	sweep = max(180, min(270, sweep)) * math.Pi / 180
	d := &dial{g: g, t: t, sweep: sweep, start: 1.5*math.Pi - sweep/2, value: g.text(value)}
	readout := d.value
	if g.Unit != "" {
		readout += " " + g.Unit
	}
	f := g.at(value)
	e := ui.Box(c).Width(t.Space(48)).Role(ui.RoleMeter).Label(g.Label).
		Range(g.Min, g.Max, g.Min+f*(g.Max-g.Min)).Value(readout)
	// The text keeps its least size on a small dial, so the height a width
	// needs is found at the width the dial was laid out at, which is known
	// a frame later.
	width := t.Space(48)
	if w := e.LayoutBounds().W; w > 0 {
		width = w
	}
	if laid := ui.Local(e, "gauge.width", func() float32 { return 0 }); *laid != width {
		*laid = width
		c.AnimationFrame()
	}
	e.AspectRatio(width / d.fit(c.MeasureText, width, inf).height())
	needle := e.AnimateTo("needle", float32(f), ui.Snappy)
	peak := -1.0
	if s != nil {
		if !math.IsNaN(value) && (!s.held || value > s.peak) {
			s.peak, s.held = value, true
		}
		if s.held {
			peak = g.at(s.peak)
		}
	}
	e.Draw(func(p *ui.Painter, r ui.Rect) {
		d.paint(p, r, float64(max(0, min(1, needle))), peak)
	})
	return e
}

var inf = float32(math.Inf(1))

// measure is Painter.MeasureText, or Context.MeasureText while building.
type measure func(width float32, spans ...ui.Span) (float32, float32)

// dial is a gauge as it paints, its colors those of the theme it was
// built in.
type dial struct {
	g Gauge
	t *ui.Theme
	// start is the angle of the arc's least value, clockwise from the
	// right, and sweep how far the arc goes from there, in radians.
	start, sweep float64
	value        string
}

// angle returns the angle of the arc f of the way along.
func (d *dial) angle(f float64) float64 { return d.start + d.sweep*f }

// shape is how large a dial's parts are, for an arc r DIPs out.
type shape struct {
	r float32
	// edge is the room outside the arc, for the peak mark; band is the
	// arc's width, gap the room inside it and between ticks and labels,
	// tick a major tick's length, and half half the needle's width.
	edge, band, gap, tick, hub, half float32
	tickSize                         float32
	// step is the value between labeled ticks, divided by minor ticks.
	step    float64
	minor   int
	labeled bool
	// value is the line under the hub, and label the line under it, if
	// any.
	value                          []ui.Span
	label                          ui.Span
	valueW, valueH, labelW, labelH float32
	// top is where the value starts below the center, and bottom how far
	// below it the dial goes.
	top, bottom float32
}

func (s shape) height() float32 { return s.edge + s.r + s.bottom }

// row is the size of a line of the readout.
type row struct{ w, h float32 }

// shape returns the parts of a dial r DIPs out, and where its readout
// goes.
func (d *dial) shape(m measure, r float32) shape {
	t, g := d.t, d.g
	s := shape{r: r, edge: max(3, r*0.07), band: max(3, r*0.07), gap: max(1.5, r*0.025), tick: max(4, r*0.09),
		hub: max(3, r*0.055), half: max(1.2, r*0.022)}
	s.tickSize = max(t.FontSize-3, r*0.11)
	var low []ui.Rect
	if g.Max > g.Min {
		s.step, s.minor, s.labeled = d.scale(m, s)
	}
	if s.labeled {
		for _, v := range g.marks(s.step) {
			if _, b := d.tickLabel(m, s, v); b.Y+b.H > 0 {
				low = append(low, b)
			}
		}
	}
	size := max(t.FontSize+1, r*0.18)
	s.value = []ui.Span{{Text: d.value, Font: t.Font, Size: size, Weight: 600, Color: t.Text, Features: "tnum"}}
	if g.Unit != "" {
		s.value = append(s.value, ui.Span{Text: " " + g.Unit, Font: t.Font, Size: max(t.FontSize-3, size*0.5), Color: t.TextMuted})
	}
	s.valueW, s.valueH = m(0, s.value...)
	s.top = d.place(s, low, row{s.valueW, s.valueH})
	depth := r * float32(math.Sin(d.start))
	if g.Label != "" {
		label := ui.Span{Text: g.Label, Font: t.Font, Size: max(t.FontSize-2, r*0.09), Color: t.TextMuted}
		w, h := m(0, label)
		top := d.place(s, low, row{s.valueW, s.valueH}, row{w, h})
		line := append([]ui.Span{label, {Text: " ", Font: t.Font, Size: label.Size}}, s.value...)
		lw, lh := m(0, line...)
		// Where the value alone goes below the arc, as on a semicircle, a
		// line of its own for the label would make the dial deeper still,
		// so the label leads the value on its line, if that fits.
		if lt := d.place(s, low, row{lw, lh}); top+s.valueH > depth && lw <= 2*(r+s.edge) && lt+lh < top+s.valueH+h {
			s.value, s.valueW, s.valueH, s.top = line, lw, lh, lt
		} else {
			s.label, s.labelW, s.labelH, s.top = label, w, h, top
		}
	}
	s.bottom = max(depth, s.top+s.valueH+s.labelH)
	return s
}

// place returns how far below the center the readout's rows start, so
// that neither the needle, at any value, nor the tick labels in low cross
// them: below the needle's tail, and below the needle at either end of
// the arc, where it points lowest.
func (d *dial) place(s shape, low []ui.Rect, rows ...row) float32 {
	below := (d.sweep - math.Pi) / 2
	sin, cos := float32(math.Sin(below)), float32(math.Cos(below))
	top := s.hub*1.4 + s.half + s.gap
	// Each step down meets one more of the rows' bounds, and can bring
	// another label in the way.
	for range len(rows)*(len(low)+1) + 1 {
		moved := false
		y := top
		for _, rw := range rows {
			need := (s.half + s.gap + rw.w/2*sin) / cos
			for _, b := range low {
				if b.X < rw.w/2+s.gap && -rw.w/2-s.gap < b.X+b.W && b.Y-s.gap < y+rw.h && y < b.Y+b.H+s.gap {
					need = max(need, b.Y+b.H+s.gap)
				}
			}
			if y < need {
				top, moved = top+need-y, true
				break
			}
			y += rw.h
		}
		if !moved {
			break
		}
	}
	return top
}

// fit returns the largest dial that fits w by h.
func (d *dial) fit(m measure, w, h float32) shape {
	r := min(w/2/1.07, w/2-3)
	s := d.shape(m, r)
	// The text keeps its least size as the dial shrinks, so it takes a
	// few steps to fit.
	for range 8 {
		if s.height() <= h || r < 8 {
			break
		}
		r *= 0.99 * h / s.height()
		s = d.shape(m, r)
	}
	return s
}

// scale returns the step between labeled ticks, the minor ticks dividing
// it, and whether its labels fit: the least round step whose labels do
// not crowd along the arc, and of those, one labeling both ends if any
// does.
func (d *dial) scale(m measure, s shape) (step float64, minor int, labeled bool) {
	g := d.g
	span := g.Max - g.Min
	out := s.r - s.band - 2*s.gap - s.tick
	whole := func(v float64) bool { return math.Abs(v-math.Round(v)) < 1e-9 }
	var fit, last float64
	for e := math.Floor(math.Log10(span / 10)); math.Pow(10, e) <= span; e++ {
		for _, k := range []float64{1, 2, 2.5, 4, 5} {
			step := k * math.Pow(10, e)
			if step < span/10 || step > span {
				continue
			}
			last = step
			widest := ""
			for _, v := range g.marks(step) {
				if l := label(v, step); len(l) > len(widest) {
					widest = l
				}
			}
			w, _ := m(0, ui.Span{Text: widest, Font: d.t.Font, Size: s.tickSize, Features: "tnum"})
			if 2*(out-w/2)*float32(math.Sin(d.sweep*step/span/2)) < w+s.tickSize*0.6 {
				continue
			}
			if whole(g.Min/step) && whole(g.Max/step) {
				return step, divisions(step), true
			}
			if fit == 0 {
				fit = step
			}
		}
	}
	if fit > 0 {
		return fit, divisions(fit), true
	}
	return last, divisions(last), false
}

// divisions returns how many minor ticks divide a step: 4 of a step of 2
// or 4, else 5.
func divisions(step float64) int {
	lead := math.Round(step / math.Pow(10, math.Floor(math.Log10(step)+1e-9)))
	if lead == 2 || lead == 4 {
		return 4
	}
	return 5
}

// label writes a tick's value v, of ticks step apart, in thousands from a
// step of 1000: "40", "2.5", "6k".
func label(v, step float64) string {
	suffix := ""
	if step >= 1000 && v != 0 {
		v, step, suffix = v/1000, step/1000, "k"
	}
	d := 0
	for k := step; d < 6 && math.Abs(k-math.Round(k)) > 1e-9; k *= 10 {
		d++
	}
	return number(v, d) + suffix
}

// tickLabel returns the label of the tick at v and its box, from the
// dial's center, inside the tick.
func (d *dial) tickLabel(m measure, s shape, v float64) (ui.Span, ui.Rect) {
	l := ui.Span{Text: label(v, s.step), Font: d.t.Font, Size: s.tickSize, Color: d.t.TextMuted, Features: "tnum"}
	w, h := m(0, l)
	a := d.angle(d.g.at(v))
	cos, sin := float32(math.Cos(a)), float32(math.Sin(a))
	rc := s.r - s.band - 2*s.gap - s.tick - (abs(cos)*w+abs(sin)*h)/2
	return l, ui.Rect{X: rc*cos - w/2, Y: rc*sin - h/2, W: w, H: h}
}

// paint draws the dial within rect, its needle and its peak mark as far
// along as needle and peak; a peak below 0 draws no mark.
func (d *dial) paint(p *ui.Painter, rect ui.Rect, needle, peak float64) {
	t, g := d.t, d.g
	s := d.fit(p.MeasureText, rect.W, rect.H)
	cx := rect.X + rect.W/2
	cy := rect.Y + (rect.H-s.height())/2 + s.edge + s.r
	at := func(f float64, r float32) (float32, float32) {
		a := d.angle(f)
		return cx + r*float32(math.Cos(a)), cy + r*float32(math.Sin(a))
	}
	band := func(from, to float64, c ui.Color) {
		if to > from {
			p.FillPath(sector(cx, cy, s.r-s.band, s.r, d.angle(from), d.angle(to)), c)
		}
	}
	band(0, 1, t.Text.Alpha(0.1))
	if len(g.Zones) == 0 {
		band(0, needle, t.Accent)
	}
	for _, z := range g.Zones {
		band(g.at(z.From), g.at(z.To), z.Color)
	}

	if span := g.Max - g.Min; span > 0 {
		in, minor := s.r-s.band-s.gap, s.minor
		for minor > 1 && float64(in)*d.sweep*s.step/span/float64(minor) < 5 {
			minor /= 2
		}
		var majors, minors ui.Path
		each := s.step / float64(minor)
		for k := int(math.Ceil(g.Min/each - 1e-9)); float64(k)*each <= g.Max+each*1e-9; k++ {
			f := g.at(float64(k) * each)
			length, path := s.tick/2, &minors
			if k%minor == 0 {
				length, path = s.tick, &majors
			}
			path.MoveTo(at(f, in)).LineTo(at(f, in-length))
		}
		p.StrokePath(&minors, max(1, s.r*0.012), t.Text.Alpha(0.35))
		p.StrokePath(&majors, max(1.5, s.r*0.018), t.Text.Alpha(0.7))
		if s.labeled {
			for _, v := range g.marks(s.step) {
				l, b := d.tickLabel(p.MeasureText, s, v)
				p.RichText(cx+b.X, cy+b.Y, 0, l)
			}
		}
	}

	if peak >= 0 {
		a := d.angle(peak)
		cos, sin := float32(math.Cos(a)), float32(math.Sin(a))
		tip, base, half := s.r+1, s.r+s.edge, s.edge*0.6
		var mark ui.Path
		mark.MoveTo(cx+tip*cos, cy+tip*sin).
			LineTo(cx+base*cos-half*sin, cy+base*sin+half*cos).
			LineTo(cx+base*cos+half*sin, cy+base*sin-half*cos).Close()
		p.FillPath(&mark, t.Text)
	}

	p.RichText(cx-s.valueW/2, cy+s.top, 0, s.value...)
	if s.label.Text != "" {
		p.RichText(cx-s.labelW/2, cy+s.top+s.valueH, 0, s.label)
	}

	a := d.angle(needle)
	cos, sin := float32(math.Cos(a)), float32(math.Sin(a))
	length, tail := s.r-s.band-s.gap-s.tick*0.4, s.hub*1.4
	var hand ui.Path
	hand.MoveTo(cx-tail*cos-s.half*sin, cy-tail*sin+s.half*cos).
		LineTo(cx+length*cos-0.5*sin, cy+length*sin+0.5*cos).
		LineTo(cx+length*cos+0.5*sin, cy+length*sin-0.5*cos).
		LineTo(cx-tail*cos+s.half*sin, cy-tail*sin-s.half*cos).Close()
	p.FillPath(&hand, t.Text)
	var hub ui.Path
	p.FillPath(hub.Circle(cx, cy, s.hub), t.Text)
}

// sector returns the ring between radii in and out from angle a0 to a1.
func sector(cx, cy, in, out float32, a0, a1 float64) *ui.Path {
	n := max(2, int(math.Ceil((a1-a0)*float64(out)/3)))
	var p ui.Path
	for i := 0; i <= n; i++ {
		a := a0 + (a1-a0)*float64(i)/float64(n)
		x, y := cx+out*float32(math.Cos(a)), cy+out*float32(math.Sin(a))
		if i == 0 {
			p.MoveTo(x, y)
		} else {
			p.LineTo(x, y)
		}
	}
	for i := n; i >= 0; i-- {
		a := a0 + (a1-a0)*float64(i)/float64(n)
		p.LineTo(cx+in*float32(math.Cos(a)), cy+in*float32(math.Sin(a)))
	}
	return p.Close()
}

func abs(v float32) float32 { return max(v, -v) }
