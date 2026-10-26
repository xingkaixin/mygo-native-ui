// Package scope is an oscilloscope, as test benches and audio tools show
// one: a screen of ten divisions across and eight up, 0 V at its middle,
// the traces of up to two channels on it, and the controls of their
// scale below. The screen holds still on a rise of the first channel
// through the trigger level, at its middle, shown by the arrow at its
// right; with no rise, it shows the latest samples as they come, and says
// Auto. Dragging up or down on the screen moves the trigger level. The
// buttons step the time and the volts a division in steps of 1, 2 and 5;
// XY plots the second channel against the first, as for a Lissajous
// figure. Each channel's frequency and its peak-to-peak volts show under
// the screen.
//
//	scope.View(c, &app.scope, 48000, app.left, app.right)
package scope

import (
	"fmt"
	"math"
	"time"

	"github.com/egoist/mygo/ui"
)

// State is what a scope keeps: the time and the volts a division, the
// trigger level, in volts, and whether it plots the channels against each
// other. Set them for the signal, or leave them zero for 1 ms and 0.5 V a
// division and a level of 0 V. Keep it in the app's state, one for each
// scope.
type State struct {
	TimeDiv  time.Duration
	VoltsDiv float64
	Trigger  float64
	XY       bool
}

const across, up = 10, 8

var traces = [2]ui.Color{ui.Hex("#F5D90A"), ui.Hex("#3DD6F5")}

// View creates the scope of channels, each samples at rate a second, the
// latest last.
func View(c *ui.Context, s *State, rate float64, channels ...[]float64) ui.Element {
	t := c.Theme()
	if s.TimeDiv <= 0 {
		s.TimeDiv = time.Millisecond
	}
	if s.VoltsDiv <= 0 {
		s.VoltsDiv = 0.5
	}
	channels = channels[:min(len(channels), 2)]
	xy := s.XY && len(channels) == 2
	n := max(2, int(s.TimeDiv.Seconds()*across*rate))
	start, trigged := -1, false
	if len(channels) > 0 {
		start, trigged = trigger(channels[0], n, s.Trigger)
	}
	root := ui.Column(c).Width(t.Space(160)).Gap(t.Space(3))
	root.Children(func() {
		screen := ui.Box(c).AspectRatio(float32(across) / up).Radius(t.Space(2)).Background(ui.Hex("#0B0F0E")).Clip().
			Cursor(ui.CursorResizeNS).Role(ui.RoleImage).Label("Oscilloscope")
		if _, dy, ok := screen.Dragged(); ok && dy != 0 {
			if b := screen.Bounds(); b.H > 0 {
				s.Trigger -= float64(dy/b.H*up) * s.VoltsDiv
				s.Trigger = max(-up/2*s.VoltsDiv, min(up/2*s.VoltsDiv, s.Trigger))
				root.SetChanged()
			}
		}
		state := *s
		screen.Draw(func(p *ui.Painter, r ui.Rect) {
			grid(p, r)
			dx, dy := r.W/across, r.H/up
			volts := func(v float64) float32 { return r.Y + r.H/2 - float32(v/state.VoltsDiv)*dy }
			if xy {
				a, b := window(channels[0], start, n), window(channels[1], start, n)
				pts := make([][2]float32, min(len(a), len(b)))
				for i := range pts {
					pts[i] = [2]float32{r.X + r.W/2 + float32(a[i]/state.VoltsDiv)*dx, volts(b[i])}
				}
				p.StrokePolyline(pts, 1.5, traces[0])
				return
			}
			for k := len(channels) - 1; k >= 0; k-- {
				w := window(channels[k], start, n)
				pts := make([][2]float32, len(w))
				for i, v := range w {
					pts[i] = [2]float32{r.X + r.W*float32(i)/float32(n-1), volts(v)}
				}
				p.StrokePolyline(pts, 1.5, traces[k])
			}
			ty := volts(state.Trigger)
			k := t.Space(1.5)
			var arrow ui.Path
			arrow.MoveTo(r.X+r.W-1, ty-k).LineTo(r.X+r.W-1-k*1.4, ty).LineTo(r.X+r.W-1, ty+k).Close()
			p.FillPath(&arrow, traces[0])
		})
		ui.Row(c).Gap(t.Space(4)).AlignItems(ui.Center).Children(func() {
			for k, ch := range channels {
				f, vpp := measure(window(ch, start, n), rate)
				ui.Row(c.Key(k)).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
					ui.Box(c).Size(t.Space(2.5), t.Space(2.5)).Radius(t.Space(1.25)).Background(traces[k])
					ui.Text(c, fmt.Sprintf("CH%d", k+1)).Bold()
					ui.Text(c, frequency(f)).FontFeatures("tnum").Width(t.Space(22))
					ui.Text(c, volt(vpp)+" pp").FontFeatures("tnum").Width(t.Space(22))
				})
			}
			ui.Box(c).Grow(1)
			status := "Auto"
			if trigged {
				status = "Trig'd " + volt(s.Trigger)
			}
			ui.Text(c, status).FontFeatures("tnum").TextColor(t.TextMuted).NoWrap()
		})
		ui.Row(c).Gap(t.Space(5)).AlignItems(ui.Center).Children(func() {
			if v, ok := stepper(c, "Time", duration(s.TimeDiv)+"/div"); ok {
				s.TimeDiv = time.Duration(step(float64(s.TimeDiv), v, 1e3, 1e9))
				root.SetChanged()
			}
			if v, ok := stepper(c, "Volts", volt(s.VoltsDiv)+"/div"); ok {
				s.VoltsDiv = step(s.VoltsDiv, v, 1e-3, 10)
				root.SetChanged()
			}
			ui.Box(c).Grow(1)
			if len(channels) == 2 && ui.Toggle(c, &s.XY, "XY").Changed() {
				root.SetChanged()
			}
		})
	})
	return root
}

// trigger returns where to start n samples of ch so that its first rise
// through level, after n/2, is at their middle, and whether there is one;
// without, the latest n.
func trigger(ch []float64, n int, level float64) (int, bool) {
	for i := max(1, n/2); i+n/2 <= len(ch); i++ {
		if ch[i-1] < level && ch[i] >= level {
			return i - n/2, true
		}
	}
	return max(0, len(ch)-n), false
}

// window returns the n samples of ch from start, fewer where ch has none.
func window(ch []float64, start, n int) []float64 {
	start = max(0, min(start, len(ch)))
	return ch[start:min(len(ch), start+n)]
}

// measure returns the frequency of w, from its rises through its middle,
// 0 for fewer than two, and its peak-to-peak volts.
func measure(w []float64, rate float64) (float64, float64) {
	if len(w) == 0 {
		return 0, 0
	}
	lo, hi := w[0], w[0]
	for _, v := range w {
		lo, hi = min(lo, v), max(hi, v)
	}
	mid := (lo + hi) / 2
	// A rise counts once the signal has fallen a tenth of its swing below
	// the middle, so that noise around the middle is not many rises.
	band := (hi - lo) / 10
	first, last, rises := -1.0, 0.0, 0
	armed := false
	for i := 1; i < len(w); i++ {
		if w[i] < mid-band {
			armed = true
		}
		if armed && w[i-1] < mid && w[i] >= mid {
			at := float64(i-1) + (mid-w[i-1])/(w[i]-w[i-1])
			if first < 0 {
				first = at
			}
			last, armed = at, false
			rises++
		}
	}
	if rises < 2 || hi-lo < 1e-9 {
		return 0, hi - lo
	}
	return float64(rises-1) * rate / (last - first), hi - lo
}

// stepper shows name and value between buttons stepping it, and returns
// -1 or 1 for a step down or up.
func stepper(c *ui.Context, name, value string) (int, bool) {
	t := c.Theme()
	dir := 0
	ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Role(ui.RoleGroup).Label(name).Children(func() {
		ui.Text(c, name).TextColor(t.TextMuted)
		if ui.Button(c, "−").Label("Less " + name).Clicked() {
			dir = -1
		}
		ui.Text(c, value).FontFeatures("tnum").Width(t.Space(22)).TextAlign(ui.Center)
		if ui.Button(c, "+").Label("More " + name).Clicked() {
			dir = 1
		}
	})
	return dir, dir != 0
}

// step returns the value of the 1, 2, 5 sequence a step from v in dir,
// within lo and hi.
func step(v float64, dir int, lo, hi float64) float64 {
	e := math.Pow(10, math.Floor(math.Log10(v)+1e-9))
	m := math.Round(v / e)
	seq := []float64{1, 2, 5, 10}
	i := 0
	for i < 3 && seq[i] < m {
		i++
	}
	switch {
	case dir > 0 && i == 2:
		m, e = 1, e*10
	case dir > 0:
		m = seq[i+1]
	case i == 0:
		m, e = 5, e/10
	default:
		m = seq[i-1]
	}
	return max(lo, min(hi, m*e))
}

func grid(p *ui.Painter, r ui.Rect) {
	line := ui.RGB(255, 255, 255).Alpha(0.12)
	for i := 1; i < across; i++ {
		x := r.X + r.W*float32(i)/across
		p.Line(x, r.Y, x, r.Y+r.H, 1, line)
	}
	for i := 1; i < up; i++ {
		y := r.Y + r.H*float32(i)/up
		p.Line(r.X, y, r.X+r.W, y, 1, line)
	}
	axis := ui.RGB(255, 255, 255).Alpha(0.3)
	cx, cy := r.X+r.W/2, r.Y+r.H/2
	for i := range across * 5 {
		x := r.X + r.W*float32(i)/(across*5)
		p.Line(x, cy-3, x, cy+3, 1, axis)
	}
	for i := range up * 5 {
		y := r.Y + r.H*float32(i)/(up*5)
		p.Line(cx-3, y, cx+3, y, 1, axis)
	}
}

func volt(v float64) string {
	if math.Abs(v) < 1 && v != 0 {
		return fmt.Sprintf("%.3g mV", v*1000)
	}
	return fmt.Sprintf("%.3g V", v)
}

func duration(d time.Duration) string {
	switch {
	case d < time.Millisecond:
		return fmt.Sprintf("%g µs", float64(d)/1e3)
	case d < time.Second:
		return fmt.Sprintf("%g ms", float64(d)/1e6)
	default:
		return fmt.Sprintf("%g s", d.Seconds())
	}
}

func frequency(f float64) string {
	switch {
	case f == 0:
		return "— Hz"
	case f < 1000:
		return fmt.Sprintf("%.1f Hz", f)
	default:
		return fmt.Sprintf("%.2f kHz", f/1000)
	}
}
