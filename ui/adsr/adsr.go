// Package adsr is an editor of the envelope a synthesizer shapes each
// note with, drawn as synths draw it: the level of a note over time,
// rising over the attack, falling over the decay to the sustain level,
// held there for as long as the key is, drawn dashed, and falling over
// the release once the key is let go. Dragging the peak sets the attack;
// dragging the corner after the decay sets the decay, and up and down
// the sustain level; dragging the end sets the release, each time from
// 1 ms to 10 s. The times fit the width; while a handle is dragged the
// scale holds, the handle under the pointer, until the times outgrow it.
// The arrow keys move the handle focused, Shift finely, and a double
// click resets it. The times and the sustain level, in decibels, show
// under the graph, and a dot rides the curve for each of the notes the
// app says are sounding.
//
//	adsr.Editor(c, &app.env, app.notes) // the notes sounding
//	...
//	level := app.env.Level(t, held) // in the synth, for each sample
package adsr

import (
	"math"
	"strconv"
	"time"

	"github.com/egoist/mygo/ui"
)

// Envelope is how the level of a note goes, from 0 to 1: up to 1 over
// Attack from its note-on, down to Sustain over Decay, held there until
// the note is let go, then down to 0 over Release.
//
// It rises as a capacitor charges, toward 1.3 and stopping at 1, as the
// envelopes of analog synths do: fast at first and slowing near the top.
// It falls as one discharges, toward a little below where it ends so
// that it gets there in time: fast at first, then slower.
type Envelope struct {
	Attack, Decay time.Duration
	Sustain       float64
	Release       time.Duration
}

// Level returns the level of a note t after its note-on, let go held
// after it; while the note is held, pass t for held. Once let go, it
// falls from the level it had then to 0 over Release, whatever stage it
// was in.
func (e Envelope) Level(t, held time.Duration) float64 {
	switch {
	case t < 0:
		return 0
	case t <= held:
		return e.holding(t)
	case t-held >= e.Release:
		return 0
	}
	return e.holding(held) * fall(float64(t-held)/float64(e.Release))
}

// holding returns the level of a note held, t after its note-on.
func (e Envelope) holding(t time.Duration) float64 {
	switch {
	case t < 0:
		return 0
	case t < e.Attack:
		return rise(float64(t) / float64(e.Attack))
	case t-e.Attack < e.Decay:
		return e.Sustain + (1-e.Sustain)*fall(float64(t-e.Attack)/float64(e.Decay))
	}
	return e.Sustain
}

// The attack charges toward 1+charge, and a fall discharges toward
// discharge below its end, as shares of how far each goes.
const (
	charge    = 0.3
	discharge = 0.05
)

// rise and fall go from 0 to 1 and from 1 to 0 as x goes from 0 to 1.
func rise(x float64) float64 { return (1 + charge) * (1 - math.Pow(charge/(1+charge), x)) }
func fall(x float64) float64 { return (1+discharge)*math.Pow(discharge/(1+discharge), x) - discharge }

// Note is a note sounding through an envelope: when it was played, and
// when it was let go, zero while it is held.
type Note struct {
	On, Off time.Time
}

// defaults are what a double click resets a handle to.
var defaults = Envelope{
	Attack:  10 * time.Millisecond,
	Decay:   300 * time.Millisecond,
	Sustain: 0.7,
	Release: 400 * time.Millisecond,
}

const (
	shortest = time.Millisecond
	longest  = 10 * time.Second
	// stretch is the share of the width the held stretch takes.
	stretch = 0.2
)

func clamp(d time.Duration) time.Duration { return max(shortest, min(longest, d)) }

// seconds returns v seconds.
func seconds(v float32) time.Duration { return time.Duration(float64(v) * float64(time.Second)) }

// step returns d after the left and right arrows pressed on e, each a
// hundredth of the range from shortest to longest in ratios, as a
// logarithmic knob turns, or a thousandth with Shift.
func step(e ui.Element, d time.Duration) time.Duration {
	var f float64
	switch {
	case e.Shortcut(0, ui.KeyRight):
		f = 0.04
	case e.Shortcut(ui.Shift, ui.KeyRight):
		f = 0.004
	case e.Shortcut(0, ui.KeyLeft):
		f = -0.04
	case e.Shortcut(ui.Shift, ui.KeyLeft):
		f = -0.004
	default:
		return d
	}
	return clamp(time.Duration(float64(max(d, shortest)) * math.Pow(10, f)))
}

// drag is the drag of a handle going on: the scale as it began, held
// while the times fit at it, and where the pointer is in the graph.
type drag struct {
	scale, x, y float32
}

// Editor creates the editor of *env, with a dot on its curve for each of
// notes.
func Editor(c *ui.Context, env *Envelope, notes []Note) ui.Element {
	t := c.Theme()
	w, h, pad := t.Space(120), t.Space(40), t.Space(3)
	span, tall := w-2*pad, h-2*pad
	hold := span * stretch
	var hot [3]bool
	root := ui.Column(c).Gap(t.Space(2))
	root.Children(func() {
		graph := ui.Box(c).Size(w, h).Clip().Role(ui.RoleGroup).Label("Envelope")
		dr := ui.Local(graph, "adsr.drag", func() drag { return drag{} })
		total := max(env.Attack+env.Decay+env.Release, shortest)
		scale := (span - hold) / float32(total.Seconds())
		if dr.scale > 0 {
			scale = min(dr.scale, scale)
		}
		if eased := graph.Animate("adsr.fit", scale, 200*time.Millisecond); dr.scale == 0 {
			scale = eased
		}
		x := func(d time.Duration) float32 { return pad + float32(d.Seconds())*scale }
		y := func(l float64) float32 { return pad + (1-float32(l))*tall }
		// to returns the time of a handle, after the times before and with
		// after behind it, that puts it under the pointer: at the scale
		// held while the total fits at it, else at the scale that fits.
		to := func(before, after time.Duration) time.Duration {
			v := seconds((dr.x-pad)/dr.scale) - before
			if f := float64((dr.x - pad) / (span - hold)); before+v+after > seconds((span-hold)/dr.scale) {
				if f >= 1 {
					return longest
				}
				v = time.Duration(f/(1-f)*float64(after)) - before
			}
			return clamp(v)
		}
		dragging := false
		graph.Children(func() {
			d := t.Space(3.5)
			// handle creates handle k at cx, cy, and reports whether it was
			// dragged since the last frame.
			handle := func(k int, label, value string, cx, cy float32) (ui.Element, bool) {
				e := ui.Box(c.Key(k)).Absolute().Left(cx-d/2).Top(cy-d/2).Size(d, d).Radius(d / 2).
					Focusable().Cursor(ui.CursorGrab).Role(ui.RoleSlider).Label(label).Value(value)
				hot[k] = e.Hovered() || e.Pressed() || e.Focused()
				active := e.Hovered() || e.Pressed()
				e.Draw(func(p *ui.Painter, r ui.Rect) {
					radius := r.W / 2
					if !active {
						radius -= 1.5
					}
					var dot ui.Path
					dot.Circle(r.X+r.W/2, r.Y+r.H/2, radius)
					p.FillPath(&dot, t.Background)
					p.StrokePath(&dot, 2, t.Accent)
				})
				dx, dy, ok := e.Dragged()
				if !ok {
					return e, false
				}
				if dr.scale == 0 {
					*dr = drag{scale, cx, cy}
				}
				dragging = true
				dr.x, dr.y = dr.x+dx, dr.y+dy
				return e, dx != 0 || dy != 0
			}

			a, moved := handle(0, "Attack", duration(env.Attack), x(env.Attack), y(1))
			if moved {
				env.Attack = to(0, env.Decay+env.Release)
			}
			env.Attack = step(a, env.Attack)
			if a.DoubleClicked() {
				env.Attack = defaults.Attack
			}

			ds, moved := handle(1, "Decay and sustain", duration(env.Decay)+", "+decibels(env.Sustain),
				x(env.Attack+env.Decay), y(env.Sustain))
			if moved {
				env.Decay = to(env.Attack, env.Release)
				env.Sustain = 1 - float64((dr.y-pad)/tall)
			}
			env.Decay = step(ds, env.Decay)
			switch {
			case ds.Shortcut(0, ui.KeyUp):
				env.Sustain += 0.01
			case ds.Shortcut(ui.Shift, ui.KeyUp):
				env.Sustain += 0.001
			case ds.Shortcut(0, ui.KeyDown):
				env.Sustain -= 0.01
			case ds.Shortcut(ui.Shift, ui.KeyDown):
				env.Sustain -= 0.001
			case ds.DoubleClicked():
				env.Decay, env.Sustain = defaults.Decay, defaults.Sustain
			}
			env.Sustain = max(0, min(1, env.Sustain))

			r, moved := handle(2, "Release", duration(env.Release), x(env.Attack+env.Decay+env.Release)+hold, y(0))
			if moved {
				// Past the edge, the release goes on growing at the scale
				// held, the handle staying at the edge.
				env.Release = clamp(seconds((dr.x-pad-hold)/dr.scale) - env.Attack - env.Decay)
			}
			env.Release = step(r, env.Release)
			if r.DoubleClicked() {
				env.Release = defaults.Release
			}
		})
		if !dragging {
			dr.scale = 0
		}
		e := *env
		graph.Draw(func(p *ui.Painter, r ui.Rect) {
			px := func(d time.Duration) float32 { return r.X + x(d) }
			py := func(l float64) float32 { return r.Y + y(l) }
			plot := ui.Rect{X: r.X + pad, Y: r.Y + pad, W: span, H: tall}
			p.Fill(r, t.Text.Alpha(0.04), t.Space(2))
			for k := 1; k < 4; k++ {
				gy := py(float64(k) / 4)
				p.Line(plot.X, gy, plot.X+plot.W, gy, 1, t.Text.Alpha(0.06))
			}
			p.Line(plot.X, py(0), plot.X+plot.W, py(0), 1, t.Border)

			ad := e.Attack + e.Decay
			x0, xa, xd := px(0), px(e.Attack), px(ad)
			xr := xd + hold
			for _, gx := range []float32{xa, xd, xr} {
				p.Line(gx, plot.Y, gx, plot.Y+plot.H, 1, t.Text.Alpha(0.08))
			}
			const n = 48
			var on, off [][2]float32
			for i := 0; i <= n; i++ {
				f := float64(i) / n
				on = append(on, [2]float32{x0 + float32(f)*(xa-x0), py(rise(f))})
			}
			for i := 1; i <= n; i++ {
				f := float64(i) / n
				on = append(on, [2]float32{xa + float32(f)*(xd-xa), py(e.Sustain + (1-e.Sustain)*fall(f))})
			}
			rw := float32(e.Release.Seconds()) * scale
			for i := 0; i <= n; i++ {
				f := float64(i) / n
				off = append(off, [2]float32{xr + float32(f)*rw, py(e.Sustain * fall(f))})
			}
			var area ui.Path
			area.MoveTo(x0, py(0))
			for _, pt := range on {
				area.LineTo(pt[0], pt[1])
			}
			for _, pt := range off {
				area.LineTo(pt[0], pt[1])
			}
			area.Close()
			p.FillPath(&area, t.Accent.Alpha(0.14))
			p.StrokePolyline(on, 2.5, t.Accent)
			p.StrokePolyline(off, 2.5, t.Accent)
			sy := py(e.Sustain)
			dash := t.Space(1.5)
			for dx := float32(0); dx < hold; dx += 2 * dash {
				p.Line(xd+dx, sy, xd+min(dx+dash, hold), sy, 2.5, t.Accent)
			}

			moving := false
			now := p.Now()
			for _, nt := range notes {
				at := now.Sub(nt.On)
				let := at
				if !nt.Off.IsZero() {
					let = nt.Off.Sub(nt.On)
				}
				if at < 0 || at > let && at-let >= e.Release {
					continue
				}
				var dx float32
				switch {
				case at > let:
					dx = xr + float32((at-let).Seconds())*scale
					moving = true
				case at > ad:
					// A held note waits at the end of the held stretch,
					// which is where its release starts.
					dx = xd + min(hold, float32((at-ad).Seconds())*scale)
					moving = moving || dx < xd+hold
				default:
					dx = px(at)
					moving = true
				}
				var dot ui.Path
				dot.Circle(dx, py(e.Level(at, let)), t.Space(1.25))
				p.FillPath(&dot, t.Text)
				p.StrokePath(&dot, 1.5, t.Background)
			}
			if moving {
				p.AnimationFrame()
			}
		})
		ui.Grid(c).Columns(4).Width(w).Padding(0, pad).Children(func() {
			for k, rd := range []struct {
				name, value string
				hot         bool
			}{
				{"A", duration(env.Attack), hot[0]},
				{"D", duration(env.Decay), hot[1]},
				{"S", decibels(env.Sustain), hot[1]},
				{"R", duration(env.Release), hot[2]},
			} {
				ui.Row(c.Key(k)).Gap(t.Space(1.5)).Children(func() {
					name := ui.Text(c, rd.name).Bold().TextColor(t.TextMuted)
					if rd.hot {
						name.TextColor(t.Accent)
					}
					ui.Text(c, rd.value).FontFeatures("tnum").SingleLine()
				})
			}
		})
	})
	return root
}

// duration writes d in milliseconds below a second, as "12 ms", and in
// seconds to a tenth above, as "1.2 s".
func duration(d time.Duration) string {
	if ms := math.Round(float64(d) / float64(time.Millisecond)); ms < 1000 {
		return strconv.Itoa(int(ms)) + " ms"
	}
	return strconv.FormatFloat(math.Round(d.Seconds()*10)/10, 'f', -1, 64) + " s"
}

// decibels writes the level v in decibels, as "−6 dB".
func decibels(v float64) string {
	if v <= 0 {
		return "−∞ dB"
	}
	if db := int(math.Round(20 * math.Log10(v))); db < 0 {
		return "−" + strconv.Itoa(-db) + " dB"
	}
	return "0 dB"
}
