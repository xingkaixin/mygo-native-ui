// Package knob is a rotary control, as on a synthesizer. Dragging up or
// right turns it up, down or left turns it down, a full turn in 200
// DIPs, ten times finer with Shift. The arrow keys step it while it has
// the focus, Shift for fine steps, Home and End to its ends; a double
// click resets it. Its label shows below it, and its value while it is
// turned or hovered. Changed reports a turn.
//
//	knob.View(c, &app.cutoff, knob.Knob{
//		Label: "Cutoff", Min: 20, Max: 20000, Default: 2000, Log: true,
//		Format: func(v float64) string { return fmt.Sprintf("%.0f Hz", v) },
//	})
package knob

import (
	"math"
	"strconv"

	"github.com/egoist/mygo/ui"
)

// Knob is what a knob turns: its range, the value a double click resets
// it to, and how it writes its value.
type Knob struct {
	Label    string
	Min, Max float64
	Default  float64
	// Log turns the knob over the range in ratios rather than steps, as
	// for a frequency or a time; Min must be above 0.
	Log bool
	// Format writes the value; nil writes it with up to two decimals.
	Format func(float64) string
}

// at returns how far v is turned, from 0 to 1.
func (k Knob) at(v float64) float64 {
	if k.Max == k.Min {
		return 0
	}
	f := (v - k.Min) / (k.Max - k.Min)
	if k.Log {
		f = math.Log(v/k.Min) / math.Log(k.Max/k.Min)
	}
	return max(0, min(1, f))
}

// value returns the value turned f of the way.
func (k Knob) value(f float64) float64 {
	f = max(0, min(1, f))
	if k.Log {
		return k.Min * math.Pow(k.Max/k.Min, f)
	}
	return k.Min + f*(k.Max-k.Min)
}

func (k Knob) text(v float64) string {
	if k.Format != nil {
		return k.Format(v)
	}
	return strconv.FormatFloat(math.Round(v*100)/100, 'f', -1, 64)
}

// sweep is how far the knob turns, in radians: from the bottom left to
// the bottom right, by the top.
const sweep = 1.5 * math.Pi

// View creates a knob turning *v.
func View(c *ui.Context, v *float64, k Knob) ui.Element {
	t := c.Theme()
	size := t.Space(12)
	root := ui.Column(c).Gap(t.Space(1)).AlignItems(ui.Center)
	root.Children(func() {
		dial := ui.Box(c).Size(size, size).Focusable().Cursor(ui.CursorResizeNS).Role(ui.RoleSlider).
			Label(k.Label).Range(k.Min, k.Max, *v).Value(k.text(*v))
		turn := func(to float64) {
			if to = max(k.Min, min(k.Max, to)); to != *v {
				*v = to
				root.SetChanged()
			}
		}
		if dx, dy, ok := dial.Dragged(); ok {
			d := float64(dx-dy) / 200
			if c.Modifiers()&ui.Shift != 0 {
				d /= 10
			}
			turn(k.value(k.at(*v) + d))
		}
		switch {
		case dial.DoubleClicked():
			turn(k.Default)
		case dial.Shortcut(0, ui.KeyUp), dial.Shortcut(0, ui.KeyRight):
			turn(k.value(k.at(*v) + 0.01))
		case dial.Shortcut(ui.Shift, ui.KeyUp), dial.Shortcut(ui.Shift, ui.KeyRight):
			turn(k.value(k.at(*v) + 0.001))
		case dial.Shortcut(0, ui.KeyDown), dial.Shortcut(0, ui.KeyLeft):
			turn(k.value(k.at(*v) - 0.01))
		case dial.Shortcut(ui.Shift, ui.KeyDown), dial.Shortcut(ui.Shift, ui.KeyLeft):
			turn(k.value(k.at(*v) - 0.001))
		case dial.Shortcut(0, ui.KeyHome):
			turn(k.Min)
		case dial.Shortcut(0, ui.KeyEnd):
			turn(k.Max)
		}
		active := dial.Pressed() || dial.Hovered() || dial.Focused()
		f := k.at(*v)
		dial.Draw(func(p *ui.Painter, r ui.Rect) {
			cx, cy := r.X+r.W/2, r.Y+r.H/2
			rad := r.W/2 - 3
			arc := func(from, to float64, w float32, col ui.Color) {
				n := max(2, int((to-from)*24))
				pts := make([][2]float32, n+1)
				for i := range pts {
					a := 0.75*math.Pi + from + (to-from)*float64(i)/float64(n)
					pts[i] = [2]float32{cx + rad*float32(math.Cos(a)), cy + rad*float32(math.Sin(a))}
				}
				p.StrokePolyline(pts, w, col)
			}
			arc(0, sweep, 4, t.Text.Alpha(0.12))
			from := 0.0
			if k.Min < 0 && k.Max > 0 && !k.Log {
				from = k.at(0)
			}
			if f != from {
				arc(sweep*min(f, from), sweep*max(f, from), 4, t.Accent)
			}
			body := ui.Rect{X: cx - rad + 6, Y: cy - rad + 6, W: 2 * (rad - 6), H: 2 * (rad - 6)}
			p.Shadow(body, body.W/2, 0, 1, 3, 0, ui.RGB(0, 0, 0).Alpha(0.25))
			p.Fill(body, t.Surface, body.W/2)
			p.Stroke(body, t.Border, body.W/2, 1)
			a := 0.75*math.Pi + sweep*f
			cos, sin := float32(math.Cos(a)), float32(math.Sin(a))
			in, out := (rad-6)*0.3, rad-9
			p.StrokePolyline([][2]float32{{cx + in*cos, cy + in*sin}, {cx + out*cos, cy + out*sin}}, 2.5, t.Text)
			if dial.FocusVisible() {
				p.Stroke(ui.Rect{X: r.X - 2, Y: r.Y - 2, W: r.W + 4, H: r.H + 4}, t.Accent.Alpha(0.5), r.W, 2)
			}
		})
		label := k.Label
		if active {
			label = k.text(*v)
		}
		ui.Text(c, label).FontSize(t.FontSize - 2).FontFeatures("tnum").TextColor(t.TextMuted).
			Width(size + t.Space(4)).TextAlign(ui.Center).SingleLine()
	})
	return root
}
