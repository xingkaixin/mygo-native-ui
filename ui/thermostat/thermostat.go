// Package thermostat is a thermostat's dial, as smart home apps show one:
// a ring of ticks around the target temperature in large figures, the
// room's below it, and the stretch of ring between the two lit orange
// while heating and blue while cooling, with a mark at the room's
// temperature. Dragging a handle round the ring, or pressing on the ring,
// sets the target; Auto keeps a range, with a handle to heat to and one to
// cool to. The modes, Off, Heat, Cool and Auto, are chosen below.
//
//	app.home = thermostat.Thermostat{Mode: thermostat.Heat, Heat: 21, Cool: 24}
//	thermostat.View(c, &app.dial, room, &app.home).Width(320)
//
// With the dial focused, Up and Right raise the target by a step, Down and
// Left lower it, and scrolling does as well; in Auto they move the handle
// last dragged. Changed reports a change.
package thermostat

import (
	"math"
	"strconv"

	"github.com/egoist/mygo/ui"
)

// Mode is what a thermostat does.
type Mode uint8

const (
	Off Mode = iota
	Heat
	Cool
	Auto
)

// Thermostat is what a thermostat is set to: its mode, the temperature it
// heats to and the one it cools to, the range of the dial, 10 to 32 when
// both are zero, and the step, 0.5 when zero.
type Thermostat struct {
	Mode       Mode
	Heat, Cool float64
	Min, Max   float64
	Step       float64
}

func (th *Thermostat) defaults() {
	if th.Min == 0 && th.Max == 0 {
		th.Min, th.Max = 10, 32
	}
	if th.Step == 0 {
		th.Step = 0.5
	}
}

// at returns where v is along the dial, from 0 to 1.
func (th *Thermostat) at(v float64) float64 {
	return max(0, min(1, (v-th.Min)/(th.Max-th.Min)))
}

// set moves the heat target, which, or the cool one to v, on a step and
// inside the dial, keeping Auto's range at least three steps wide.
func (th *Thermostat) set(cool bool, v float64) {
	v = math.Round(v/th.Step) * th.Step
	v = max(th.Min, min(th.Max, v))
	gap := 3 * th.Step
	if cool {
		th.Cool = v
		if th.Mode == Auto && th.Heat > v-gap {
			th.Heat = max(th.Min, v-gap)
			th.Cool = th.Heat + gap
		}
		return
	}
	th.Heat = v
	if th.Mode == Auto && th.Cool < v+gap {
		th.Cool = min(th.Max, v+gap)
		th.Heat = th.Cool - gap
	}
}

// State is what a dial keeps: the handle being dragged, or last dragged.
// Keep it in the app's state, one for each dial.
type State struct {
	cool     bool
	dragging bool
}

var (
	orange = ui.Hex("#ff7a1a")
	blue   = ui.Hex("#2f8cff")
)

const (
	// start and sweep are where the ring starts, at the bottom left, and
	// how far it goes round, in degrees clockwise from the right.
	start = 135.0
	sweep = 270.0
	ticks = 90
)

// angle returns the angle in radians of f along the ring.
func angle(f float64) float64 { return (start + sweep*f) * math.Pi / 180 }

// along returns how far along the ring the point (dx, dy) from the center
// is, from 0 to 1, the gap at the bottom going to the nearer end.
func along(dx, dy float32) float64 {
	a := math.Atan2(float64(dy), float64(dx)) * 180 / math.Pi
	rel := math.Mod(a-start+720, 360)
	if rel > sweep {
		if rel > sweep+(360-sweep)/2 {
			return 0
		}
		return 1
	}
	return rel / sweep
}

// View creates the dial of th, the room at room.
func View(c *ui.Context, s *State, room float64, th *Thermostat) ui.Element {
	t := c.Theme()
	th.defaults()
	changed := false
	root := ui.Column(c).Gap(t.Space(4)).AlignItems(ui.Center)
	root.Children(func() {
		dial := ui.Box(c).FillWidth().AspectRatio(1).Focusable().FocusRing(false).Role(ui.RoleSlider).
			Label("Target temperature").Cursor(ui.CursorPointer)
		if th.Mode == Off {
			dial.Cursor(ui.CursorDefault)
		}
		b := dial.LayoutBounds()
		cx, cy := b.W/2, b.H/2
		ring := b.W/2 - b.W*0.07
		target := func() float64 {
			if th.Mode == Cool || th.Mode == Auto && s.cool {
				return th.Cool
			}
			return th.Heat
		}
		nudge := func(steps float64) {
			if th.Mode == Off {
				return
			}
			th.set(th.Mode == Cool || th.Mode == Auto && s.cool, target()+steps*th.Step)
			changed = true
		}
		if x, y, _ := dial.PointerPosition(); dial.Pressed() && th.Mode != Off && b.W > 0 {
			f := along(x-cx, y-cy)
			if !s.dragging {
				s.dragging = true
				d := math.Hypot(float64(x-cx), float64(y-cy))
				if d < float64(ring)*0.55 {
					// A press inside the ring's middle is not on the dial.
					s.dragging = false
				}
				switch th.Mode {
				case Auto:
					s.cool = math.Abs(f-th.at(th.Cool)) < math.Abs(f-th.at(th.Heat))
				default:
					s.cool = th.Mode == Cool
				}
			}
			if s.dragging {
				th.set(s.cool, th.Min+f*(th.Max-th.Min))
				changed = true
			}
		} else if !dial.Pressed() {
			s.dragging = false
		}
		switch {
		case dial.Shortcut(0, ui.KeyUp), dial.Shortcut(0, ui.KeyRight):
			nudge(1)
		case dial.Shortcut(0, ui.KeyDown), dial.Shortcut(0, ui.KeyLeft):
			nudge(-1)
		}
		dial.HandleInput(func(ev ui.InputEvent) bool {
			if ev.Kind != ui.InputScroll || ev.DY == 0 {
				return false
			}
			if ev.DY < 0 {
				nudge(1)
			} else {
				nudge(-1)
			}
			c.Invalidate()
			return true
		})
		dial.Value(strconv.FormatFloat(target(), 'f', -1, 64)).Range(th.Min, th.Max, target())

		action := ""
		color := t.TextMuted
		switch {
		case th.Mode == Off:
		case (th.Mode == Heat || th.Mode == Auto) && room < th.Heat-th.Step/2:
			action, color = "Heating", orange
		case (th.Mode == Cool || th.Mode == Auto) && room > th.Cool+th.Step/2:
			action, color = "Cooling", blue
		}
		set := *th
		focused := dial.FocusVisible()
		dial.Draw(func(p *ui.Painter, r ui.Rect) {
			cx, cy := r.X+r.W/2, r.Y+r.H/2
			outer := r.W / 2
			ring := outer - r.W*0.07
			disc := t.Surface
			if action != "" {
				disc = disc.Mix(color, 0.1)
			}
			var bg ui.Path
			p.FillPath(bg.Circle(cx, cy, outer), disc)
			if focused {
				var f ui.Path
				p.StrokePath(f.Circle(cx, cy, outer+3), 2, t.Focus)
			}
			lo, hi := -1.0, -1.0
			switch {
			case action == "Heating":
				lo, hi = set.at(room), set.at(set.Heat)
			case action == "Cooling":
				lo, hi = set.at(set.Cool), set.at(room)
			}
			for i := range ticks + 1 {
				f := float64(i) / ticks
				a := angle(f)
				cos, sin := float32(math.Cos(a)), float32(math.Sin(a))
				in, w, col := ring-r.W*0.035, float32(1.5), t.TextMuted.Alpha(0.35)
				if f >= lo-1e-9 && f <= hi+1e-9 {
					in, w, col = ring-r.W*0.05, 2.5, color
				}
				if set.Mode == Off {
					col = t.TextMuted.Alpha(0.2)
				}
				p.Line(cx+cos*in, cy+sin*in, cx+cos*ring, cy+sin*ring, w, col)
			}
			if set.Mode == Off {
				return
			}
			mark := func(f float64, rad float32, fill, edge ui.Color) {
				a := angle(f)
				x, y := cx+float32(math.Cos(a))*(ring-r.W*0.025), cy+float32(math.Sin(a))*(ring-r.W*0.025)
				var h ui.Path
				h.Circle(x, y, rad)
				p.FillPath(&h, fill)
				p.StrokePath(&h, 2.5, edge)
			}
			// The room's temperature is a small dot, the targets handles.
			mark(set.at(room), r.W*0.012, t.Text, t.Text)
			if set.Mode == Heat || set.Mode == Auto {
				mark(set.at(set.Heat), r.W*0.032, ui.RGB(255, 255, 255), orange)
			}
			if set.Mode == Cool || set.Mode == Auto {
				mark(set.at(set.Cool), r.W*0.032, ui.RGB(255, 255, 255), blue)
			}
		})
		dial.Children(func() {
			ui.Column(c).Absolute().Left(0).Top(0).Right(0).Bottom(0).Center().Gap(t.Space(1)).PassThrough().Children(func() {
				status := action
				switch {
				case th.Mode == Off:
					status = "Off"
				case status != "":
				case th.Mode == Heat:
					status = "Heat set to"
				case th.Mode == Cool:
					status = "Cool set to"
				default:
					status = "Keeping between"
				}
				ui.Text(c, status).TextColor(color).Bold().FontSize(t.FontSize - 1)
				big := b.W * 0.2
				switch th.Mode {
				case Heat:
					ui.Text(c, degrees(th.Heat)).FontSize(big).FontWeight(300).FontFeatures("tnum")
				case Cool:
					ui.Text(c, degrees(th.Cool)).FontSize(big).FontWeight(300).FontFeatures("tnum")
				case Auto:
					ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Baseline).Children(func() {
						ui.Text(c, degrees(th.Heat)).FontSize(big * 0.6).FontWeight(300).FontFeatures("tnum").TextColor(orange)
						ui.Text(c, "–").FontSize(big * 0.4).TextColor(t.TextMuted)
						ui.Text(c, degrees(th.Cool)).FontSize(big * 0.6).FontWeight(300).FontFeatures("tnum").TextColor(blue)
					})
				default:
					ui.Text(c, "—").FontSize(big).FontWeight(300).TextColor(t.TextMuted)
				}
				ui.Text(c, "Indoor "+degrees(room)).TextColor(t.TextMuted).FontFeatures("tnum")
			})
		})

		modes := int(th.Mode)
		if ui.Segmented(c, &modes, "Off", "Heat", "Cool", "Auto").Label("Mode").Changed() {
			th.Mode = Mode(modes)
			if th.Mode == Auto {
				th.set(false, th.Heat)
			}
			changed = true
		}
	})
	if changed {
		root.SetChanged()
	}
	return root
}

// degrees writes v to a tenth of a degree, without a trailing zero, and °.
func degrees(v float64) string {
	return strconv.FormatFloat(math.Round(v*10)/10, 'f', -1, 64) + "°"
}
