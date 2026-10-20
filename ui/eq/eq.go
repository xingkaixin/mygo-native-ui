// Package eq is a parametric equalizer: bands of filters on a graph of
// gain from -18 to +18 dB over a log frequency axis from 20 Hz to 20 kHz,
// each band's own response faint in its color and their total bold, as
// the filters give it. Each band is a numbered handle at its frequency
// and gain, a cut filter's on its curve at its frequency. Dragging a
// handle moves the band's frequency and gain, a cut filter's frequency
// only; the scroll wheel over it changes its Q, and a double click turns
// it off or on, an off band hollow and left out of the total. A click
// chooses a band, which the arrow keys nudge and the readout under the
// graph shows.
//
// The app runs its samples through a Filter, as its audio decoder reads
// them, and sets the filter to the bands each frame; the filter glides to
// them without a click:
//
//	eq.View(c, &app.eq, &app.bands, rate)
//	app.filter.Set(app.bands)
//
//	// In the decoder's Read:
//	app.filter.Process(buf)
package eq

import (
	"math"
	"math/cmplx"
	"strconv"
	"sync/atomic"

	"github.com/egoist/mygo/ui"
)

// Type is the kind of filter a band is.
type Type uint8

// Types of filter.
const (
	// LowCut is a high-pass filter: it cuts what is below its frequency.
	LowCut Type = iota
	LowShelf
	Peak
	HighShelf
	// HighCut is a low-pass filter: it cuts what is above its frequency.
	HighCut
)

var names = [...]string{"Low cut", "Low shelf", "Peak", "High shelf", "High cut"}

// Band is a band of an equalizer: its filter, the frequency it works at,
// in Hz, the gain it gives there, in dB, which a cut filter has none of,
// its Q, which is how narrow a peak is, how steep a shelf and how much a
// cut filter rings at its frequency, and whether it is on.
type Band struct {
	Type          Type
	Freq, Gain, Q float64
	On            bool
}

const (
	low, high = 20.0, 20000.0
	span      = 18.0
)

func (b Band) cut() bool { return b.Type == LowCut || b.Type == HighCut }

func (b *Band) clamp() {
	b.Freq = max(low, min(high, b.Freq))
	b.Gain = max(-span, min(span, b.Gain))
	b.Q = max(0.1, min(18, b.Q))
}

// biquad is the coefficients of a biquad filter, divided by its a0.
type biquad struct{ b0, b1, b2, a1, a2 float64 }

// unit passes samples as they are.
var unit = biquad{b0: 1}

// biquad returns the band's filter at rate frames a second, as Robert
// Bristow-Johnson's Audio EQ Cookbook makes it.
func (b Band) biquad(rate int) biquad {
	w := 2 * math.Pi * max(1, min(b.Freq, 0.49*float64(rate))) / float64(rate)
	cos, sin := math.Cos(w), math.Sin(w)
	alpha := sin / (2 * max(0.01, b.Q))
	a := math.Pow(10, b.Gain/40)
	k := 2 * math.Sqrt(a) * alpha
	var b0, b1, b2, a0, a1, a2 float64
	switch b.Type {
	case LowCut:
		b0, b1, b2 = (1+cos)/2, -(1 + cos), (1+cos)/2
		a0, a1, a2 = 1+alpha, -2*cos, 1-alpha
	case HighCut:
		b0, b1, b2 = (1-cos)/2, 1-cos, (1-cos)/2
		a0, a1, a2 = 1+alpha, -2*cos, 1-alpha
	case Peak:
		b0, b1, b2 = 1+alpha*a, -2*cos, 1-alpha*a
		a0, a1, a2 = 1+alpha/a, -2*cos, 1-alpha/a
	case LowShelf:
		b0 = a * (a + 1 - (a-1)*cos + k)
		b1 = 2 * a * (a - 1 - (a+1)*cos)
		b2 = a * (a + 1 - (a-1)*cos - k)
		a0 = a + 1 + (a-1)*cos + k
		a1 = -2 * (a - 1 + (a+1)*cos)
		a2 = a + 1 + (a-1)*cos - k
	case HighShelf:
		b0 = a * (a + 1 + (a-1)*cos + k)
		b1 = -2 * a * (a - 1 + (a+1)*cos)
		b2 = a * (a + 1 + (a-1)*cos - k)
		a0 = a + 1 - (a-1)*cos + k
		a1 = 2 * (a - 1 - (a+1)*cos)
		a2 = a + 1 - (a-1)*cos - k
	default:
		return unit
	}
	return biquad{b0 / a0, b1 / a0, b2 / a0, a1 / a0, a2 / a0}
}

// gain returns the filter's gain at hz, in dB.
func (q biquad) gain(hz float64, rate int) float64 {
	z := cmplx.Exp(complex(0, -2*math.Pi*hz/float64(rate)))
	num := complex(q.b0, 0) + z*(complex(q.b1, 0)+z*complex(q.b2, 0))
	den := 1 + z*(complex(q.a1, 0)+z*complex(q.a2, 0))
	return 20 * math.Log10(max(1e-9, cmplx.Abs(num/den)))
}

func (q *biquad) toward(to biquad, k float64) {
	q.b0 += (to.b0 - q.b0) * k
	q.b1 += (to.b1 - q.b1) * k
	q.b2 += (to.b2 - q.b2) * k
	q.a1 += (to.a1 - q.a1) * k
	q.a2 += (to.a2 - q.a2) * k
}

// Filter runs samples through bands, a biquad filter for each in turn,
// each channel through filters of its own. When Set gives it other bands,
// its filters glide to them over a few milliseconds rather than jump, so
// that no change clicks. Set it from the UI while Process runs on the
// audio's goroutine.
type Filter struct {
	rate, channels int
	next           atomic.Pointer[[]biquad]
	// What only Process touches: the filters as they glide to next, and
	// the two values each filter of each channel keeps.
	now   []biquad
	state [][2]float64
	glide float64
}

// NewFilter returns a filter of frames of channels interleaved, at rate
// frames a second, which passes them as they are until Set.
func NewFilter(rate, channels int) *Filter {
	// Each frame goes a little of the way to the filters set: two thirds
	// of it in 10 ms, quick to follow a drag and slow enough not to click.
	return &Filter{rate: rate, channels: max(1, channels), glide: 1 - math.Exp(-100/float64(rate))}
}

// Set has the filter follow bands, the off ones passing what comes.
func (f *Filter) Set(bands []Band) {
	qs := make([]biquad, len(bands))
	for i, b := range bands {
		qs[i] = unit
		if b.On {
			qs[i] = b.biquad(f.rate)
		}
	}
	f.next.Store(&qs)
}

// Process filters samples in place: frames of the filter's channels,
// interleaved.
func (f *Filter) Process(samples []float32) {
	var next []biquad
	if p := f.next.Load(); p != nil {
		next = *p
	}
	ch := f.channels
	// Filters of bands gone glide to unit and stay, so that bands taken
	// away fade out too.
	for len(f.now) < len(next) {
		f.now = append(f.now, unit)
		f.state = append(f.state, make([][2]float64, ch)...)
	}
	for i := 0; i+ch <= len(samples); i += ch {
		for k := range f.now {
			to := unit
			if k < len(next) {
				to = next[k]
			}
			f.now[k].toward(to, f.glide)
		}
		for c := range ch {
			x := float64(samples[i+c])
			for k, q := range f.now {
				z := &f.state[k*ch+c]
				y := q.b0*x + z[0]
				z[0] = q.b1*x - q.a1*y + z[1]
				z[1] = q.b2*x - q.a2*y
				x = y
			}
			samples[i+c] = float32(x)
		}
	}
}

// State is what an equalizer keeps: the band chosen and the drag going on.
// Keep it in the app's state, one for each equalizer.
type State struct {
	chosen int
	// drag is the index of the band dragged, plus 1, so that the zero
	// State drags none.
	drag int
	down bool
	x, y float32
	ref  ui.Handle
}

// plot is where the graph draws in a box: its frequency and gain at each
// point, and the bands' handles.
type plot struct {
	ui.Rect
	rad  float32
	rate int
}

func layout(t *ui.Theme, r ui.Rect, rate int) plot {
	left, pad, bottom := t.Space(9), t.Space(3), t.Space(5.5)
	return plot{ui.Rect{X: r.X + left, Y: r.Y + pad, W: max(1, r.W-left-pad), H: max(1, r.H-pad-bottom)}, t.Space(2.5), rate}
}

func (g plot) x(hz float64) float32 {
	return g.X + g.W*float32(math.Log(hz/low)/math.Log(high/low))
}

func (g plot) y(db float64) float32 { return g.Y + g.H*float32((span-db)/(2*span)) }

func (g plot) hz(x float32) float64 { return low * math.Pow(high/low, float64((x-g.X)/g.W)) }

func (g plot) db(y float32) float64 { return span - 2*span*float64((y-g.Y)/g.H) }

// handle returns where b's handle is: at its frequency and gain, or a cut
// filter's on its curve.
func (g plot) handle(b Band) (float32, float32) {
	f := max(low, min(high, b.Freq))
	db := b.Gain
	if b.cut() {
		db = b.biquad(g.rate).gain(f, g.rate)
	}
	return g.x(f), g.y(max(-span, min(span, db)))
}

// at returns the band whose handle is nearest x, y, or -1 for none there.
func (g plot) at(bands []Band, x, y float32) int {
	best, near := -1, float64(g.rad+3)
	for i, b := range bands {
		hx, hy := g.handle(b)
		if d := math.Hypot(float64(x-hx), float64(y-hy)); d <= near {
			best, near = i, d
		}
	}
	return best
}

var hues = [...]float32{30, 80, 150, 210, 265, 315, 355, 115}

func tint(i int) ui.Color { return ui.Oklch(0.72, 0.15, hues[i%len(hues)]) }

// dot draws band i's handle at x, y: filled when on, hollow when off.
func dot(p *ui.Painter, t *ui.Theme, x, y, rad float32, i int, on bool) {
	var circle ui.Path
	circle.Circle(x, y, rad)
	color, ink := tint(i), ui.RGB(20, 20, 24)
	if on {
		p.Shadow(ui.Rect{X: x - rad, Y: y - rad, W: 2 * rad, H: 2 * rad}, rad, 0, 1, 3, 0, ui.RGB(0, 0, 0).Alpha(0.3))
		p.FillPath(&circle, color)
	} else {
		p.FillPath(&circle, t.Background)
		p.StrokePath(&circle, 1.5, color)
		ink = t.Text
	}
	label := ui.Span{Text: strconv.Itoa(i + 1), Size: t.FontSize - 2, Weight: 700, Color: ink}
	w, h := p.MeasureText(0, label)
	p.RichText(x-w/2, y-h/2, 0, label)
}

func hz(f float64) string {
	if math.Round(f) >= 1000 {
		return strconv.FormatFloat(f/1000, 'f', 1, 64) + " kHz"
	}
	return strconv.FormatFloat(f, 'f', 0, 64) + " Hz"
}

func db(g float64) string {
	g = math.Round(g*10) / 10
	if g == 0 {
		g = 0 // not -0
	}
	s := strconv.FormatFloat(g, 'f', 1, 64)
	if g >= 0 {
		s = "+" + s
	}
	return s + " dB"
}

func axis(f float64) string {
	if f >= 1000 {
		return strconv.FormatFloat(f/1000, 'f', -1, 64) + "k"
	}
	return strconv.Itoa(int(f))
}

// View creates the equalizer of *bands, its responses those of filters at
// rate frames a second.
func View(c *ui.Context, s *State, bands *[]Band, rate int) ui.Element {
	t := c.Theme()
	bs := *bands
	s.chosen = max(0, min(len(bs)-1, s.chosen))
	root := ui.Column(c).Gap(t.Space(3))
	root.Children(func() {
		graph := ui.Box(c).AlignSelf(ui.Stretch).Grow(1).MinHeight(t.Space(50)).Focusable().Label("Equalizer").Bind(&s.ref)
		b := graph.LayoutBounds()
		g := layout(t, ui.Rect{W: b.W, H: b.H}, rate)
		px, py, over := graph.PointerPosition()
		hover := -1
		if over {
			hover = g.at(bs, px, py)
		}
		// Pressed ends as the pointer leaves the graph, but the drag goes
		// on until it is let go.
		dx, dy, held := graph.Dragged()
		switch {
		case graph.Pressed() && !s.down:
			s.down, s.drag = true, 0
			if hover >= 0 {
				s.chosen, s.drag = hover, hover+1
				s.x, s.y = g.handle(bs[hover])
			}
		case held && s.down && s.drag > 0:
			s.x, s.y = s.x+dx, s.y+dy
			b := &bs[s.drag-1]
			b.Freq = g.hz(s.x)
			if !b.cut() {
				b.Gain = g.db(s.y)
			}
			b.clamp()
		case !held && s.down:
			s.down, s.drag = false, 0
		}
		if graph.DoubleClicked() && hover >= 0 {
			bs[hover].On = !bs[hover].On
		}
		if len(bs) > 0 {
			b, before := &bs[s.chosen], bs[s.chosen]
			semitone := math.Exp2(1.0 / 12)
			switch {
			case graph.Shortcut(0, ui.KeyLeft):
				b.Freq /= semitone
			case graph.Shortcut(0, ui.KeyRight):
				b.Freq *= semitone
			case graph.Shortcut(0, ui.KeyUp) && !b.cut():
				b.Gain += 0.5
			case graph.Shortcut(0, ui.KeyDown) && !b.cut():
				b.Gain -= 0.5
			}
			if *b != before {
				b.clamp()
			}
		}
		// Scrolling comes between builds, at the box laid out since, which
		// only a handle reads then.
		graph.HandleInput(func(ev ui.InputEvent) bool {
			if ev.Kind != ui.InputScroll {
				return false
			}
			b := s.ref.Bounds(c)
			i := layout(t, ui.Rect{W: b.W, H: b.H}, rate).at(*bands, ev.X, ev.Y)
			if i < 0 {
				return false
			}
			s.chosen = i
			// A notch of a mouse wheel, 40 DIPs, changes Q by a quarter
			// of an octave.
			(*bands)[i].Q *= math.Exp2(-float64(ev.DY) / 160)
			(*bands)[i].clamp()
			return true
		})
		if i := hover; i >= 0 || s.drag > 0 {
			if s.drag > 0 {
				i = s.drag - 1
			}
			if bs[i].cut() {
				graph.Cursor(ui.CursorResizeEW)
			} else {
				graph.Cursor(ui.CursorMove)
			}
		}
		chosen := s.chosen
		graph.Draw(func(p *ui.Painter, r ui.Rect) {
			g := layout(t, r, rate)
			p.Fill(g.Rect, t.Surface, 4)
			for d := -span; d <= span; d += 6 {
				color := t.Border
				if d == 0 {
					color = t.TextMuted.Alpha(0.7)
				}
				p.Line(g.X, g.y(d), g.X+g.W, g.y(d), 1, color)
				label := ui.Span{Text: strconv.Itoa(int(d)), Size: t.FontSize - 3, Color: t.TextMuted}
				if d > 0 {
					label.Text = "+" + label.Text
				}
				w, h := p.MeasureText(0, label)
				p.RichText(g.X-t.Space(2)-w, g.y(d)-h/2, 0, label)
			}
			for _, f := range []float64{50, 100, 200, 500, 1000, 2000, 5000, 10000} {
				p.Line(g.x(f), g.Y, g.x(f), g.Y+g.H, 1, t.Border)
				label := ui.Span{Text: axis(f), Size: t.FontSize - 3, Color: t.TextMuted}
				w, _ := p.MeasureText(0, label)
				p.RichText(g.x(f)-w/2, g.Y+g.H+t.Space(1), 0, label)
			}
			n := max(2, int(g.W/2))
			freqs := make([]float64, n)
			total := make([]float64, n)
			line := make([][2]float32, n)
			for k := range freqs {
				freqs[k] = low * math.Pow(high/low, float64(k)/float64(n-1))
			}
			// Cut filters fall far below the graph: where they go past its
			// edge, the clip cuts them off.
			place := func(k int, d float64) [2]float32 {
				return [2]float32{g.x(freqs[k]), g.y(max(-3*span, min(3*span, d)))}
			}
			p.Clip(g.Rect, 4, func() {
				for i, b := range bs {
					if !b.On {
						continue
					}
					q := b.biquad(rate)
					for k, f := range freqs {
						d := q.gain(f, rate)
						total[k] += d
						line[k] = place(k, d)
					}
					alpha := float32(0.55)
					if i == chosen {
						alpha = 0.9
					}
					p.StrokePolyline(line, 1.5, tint(i).Alpha(alpha))
				}
				var area ui.Path
				area.MoveTo(g.X, g.y(0))
				for k, d := range total {
					line[k] = place(k, d)
					area.LineTo(line[k][0], line[k][1])
				}
				area.LineTo(g.X+g.W, g.y(0)).Close()
				p.FillPath(&area, t.Accent.Alpha(0.14))
				p.StrokePolyline(line, 2.5, t.Accent)
			})
			handle := func(i int) {
				x, y := g.handle(bs[i])
				if i == chosen || i == hover {
					var ring ui.Path
					ring.Circle(x, y, g.rad+3)
					color := t.Text
					if i != chosen {
						color = t.Text.Alpha(0.35)
					}
					p.StrokePath(&ring, 1.5, color)
				}
				dot(p, t, x, y, g.rad, i, bs[i].On)
			}
			for i := range bs {
				if i != chosen {
					handle(i)
				}
			}
			if chosen < len(bs) {
				handle(chosen)
			}
		})
		if len(bs) > 0 {
			readout(c, s.chosen, bs[s.chosen])
		}
	})
	return root
}

// readout shows band i: its type, frequency, gain and Q.
func readout(c *ui.Context, i int, b Band) {
	t := c.Theme()
	ui.Row(c).Gap(t.Space(6)).AlignItems(ui.Center).Children(func() {
		ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Width(t.Space(36)).Children(func() {
			size := t.Space(5)
			ui.Box(c).Size(size, size).Shrink(0).Draw(func(p *ui.Painter, r ui.Rect) {
				dot(p, t, r.X+r.W/2, r.Y+r.H/2, r.W/2-1, i, b.On)
			})
			ui.Text(c, names[b.Type]).Bold()
			if !b.On {
				ui.Text(c, "Off").TextColor(t.TextMuted)
			}
		})
		gain := "—"
		if !b.cut() {
			gain = db(b.Gain)
		}
		for _, f := range [][2]string{{"Frequency", hz(b.Freq)}, {"Gain", gain}, {"Q", strconv.FormatFloat(b.Q, 'f', 2, 64)}} {
			ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
				ui.Text(c, f[0]).TextColor(t.TextMuted)
				ui.Text(c, f[1]).FontFeatures("tnum").Width(t.Space(16)).SingleLine()
			})
		}
	})
}
