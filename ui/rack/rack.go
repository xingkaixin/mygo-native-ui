// Package rack draws a rack elevation, as NetBox and data center tools
// do: the front of a rack of so many units, numbered from the bottom, and
// each device in the units it fills, in the color of its role, with its
// name, its model and a status light. Above it, how many units are used
// and how much power the devices draw against the rack's budget. Dragging
// a device moves it to free units: the slot it would land in shows green
// where it fits and red where it doesn't, and letting go there moves it.
// Hovering over a device tells its model, units and power.
//
//	if rack.View(c, &app.rackState, &app.rack).Changed() {
//		app.save()
//	}
package rack

import (
	"fmt"
	"math"
	"strconv"

	"github.com/egoist/mygo/ui"
)

// Device is a device in a rack: U is the lowest unit it fills, from 1 at
// the bottom, and Height how many it fills.
type Device struct {
	Name, Model, Role string
	U, Height         int
	Watts             float64
}

// Rack is a rack of Units units and its devices. Watts is the power its
// feeds can supply, none if zero.
type Rack struct {
	Name    string
	Units   int
	Watts   float64
	Devices []Device
}

// State is what a rack view keeps: the device being dragged and how far.
// Keep it in the app's state, one for each rack.
type State struct {
	dragging int
	dy       float32
	dropped  bool
	ready    bool
}

var palette = []ui.Color{
	ui.Oklch(0.6, 0.12, 250), ui.Oklch(0.62, 0.13, 160), ui.Oklch(0.66, 0.13, 70), ui.Oklch(0.6, 0.14, 310),
	ui.Oklch(0.6, 0.13, 25), ui.Oklch(0.62, 0.09, 200), ui.Oklch(0.6, 0.02, 260),
}

// fits reports whether device i fits with its lowest unit at u.
func (r *Rack) fits(i, u int) bool {
	d := r.Devices[i]
	if u < 1 || u+d.Height-1 > r.Units {
		return false
	}
	for j, o := range r.Devices {
		if j != i && u <= o.U+o.Height-1 && o.U <= u+d.Height-1 {
			return false
		}
	}
	return true
}

// View creates the elevation of r, which moves its devices as they are
// dragged and reports when one moved.
func View(c *ui.Context, s *State, r *Rack) ui.Element {
	t := c.Theme()
	uh := t.Space(4)
	width := t.Space(84)
	rail := t.Space(5)
	colors := map[string]ui.Color{}
	var roles []string
	used, watts := 0, 0.0
	for _, d := range r.Devices {
		if _, ok := colors[d.Role]; !ok {
			colors[d.Role] = palette[len(roles)%len(palette)]
			roles = append(roles, d.Role)
		}
		used += d.Height
		watts += d.Watts
	}
	if !s.ready {
		s.dragging, s.ready = -1, true
	}
	target, ok := 0, false
	if s.dragging >= 0 {
		d := r.Devices[s.dragging]
		target = d.U - int(math.Round(float64(s.dy/uh)))
		ok = r.fits(s.dragging, target)
	}
	changed := false
	root := ui.Column(c).Gap(t.Space(3)).AlignItems(ui.Start).Role(ui.RoleGroup).Label(r.Name)
	root.Children(func() {
		ui.Column(c).Gap(t.Space(1.5)).Width(width + 2*rail).Children(func() {
			ui.Text(c, r.Name).FontWeight(600)
			stat(c, fmt.Sprintf("%d of %d U", used, r.Units), float64(used)/float64(r.Units), t.Accent)
			if r.Watts > 0 {
				col := t.Success
				if watts > r.Watts*0.8 {
					col = t.Warning
				}
				if watts > r.Watts {
					col = t.Danger
				}
				stat(c, fmt.Sprintf("%s of %s", power(watts), power(r.Watts)), watts/r.Watts, col)
			}
		})
		frame := ui.Box(c).Size(width+2*rail, uh*float32(r.Units)+2*t.Space(2)).Background(ui.Oklch(0.27, 0.005, 260)).
			Radius(t.Space(1.5))
		frame.Draw(func(p *ui.Painter, rc ui.Rect) {
			top := rc.Y + t.Space(2)
			inner := ui.Rect{X: rc.X + rail, Y: top, W: width, H: uh * float32(r.Units)}
			p.Fill(inner, ui.Oklch(0.2, 0.005, 260), 0)
			for u := 1; u <= r.Units; u++ {
				y := top + uh*float32(r.Units-u)
				p.Fill(ui.Rect{X: inner.X, Y: y + uh - 0.5, W: width, H: 0.5}, ui.RGBA(255, 255, 255, 0.06), 0)
				span := ui.Span{Text: strconv.Itoa(u), Size: t.FontSize - 4, Color: ui.RGBA(255, 255, 255, 0.45), Features: "tnum"}
				w, h := p.MeasureText(0, span)
				p.RichText(rc.X+rail/2-w/2-2, y+uh/2-h/2, 0, span)
				// Each unit has three holes in each rail.
				for k := range 3 {
					hy := y + uh*(float32(k)+0.5)/3 - 1
					p.Fill(ui.Rect{X: rc.X + rail - 5, Y: hy, W: 3, H: 2}, ui.RGBA(0, 0, 0, 0.5), 0.5)
					p.Fill(ui.Rect{X: inner.X + width + 2, Y: hy, W: 3, H: 2}, ui.RGBA(0, 0, 0, 0.5), 0.5)
				}
			}
		})
		// The slot the dragged device would land in shows over the devices
		// already there.
		frame.DrawOver(func(p *ui.Painter, rc ui.Rect) {
			top := rc.Y + t.Space(2)
			inner := ui.Rect{X: rc.X + rail, Y: top, W: width, H: uh * float32(r.Units)}
			if s.dragging >= 0 {
				d := r.Devices[s.dragging]
				lo, hi := max(1, target), min(r.Units, target+d.Height-1)
				if lo <= hi {
					col := t.Success
					if !ok {
						col = t.Danger
					}
					slot := ui.Rect{X: inner.X, Y: top + uh*float32(r.Units-hi), W: width, H: uh * float32(hi-lo+1)}
					p.Fill(slot, col.Alpha(0.2), 2)
					p.Stroke(slot, col, 2, 1.5)
				}
			}
		})
		frame.Children(func() {
			// The device dragged comes last, to show over the others.
			dragging := s.dragging
			for i := range r.Devices {
				if i != dragging {
					device(c.Key(i), s, r, i, colors, uh, width, rail)
				}
			}
			if dragging >= 0 {
				device(c.Key(dragging), s, r, dragging, colors, uh, width, rail)
			}
		})
		if s.dropped {
			d := &r.Devices[s.dragging]
			if u := d.U - int(math.Round(float64(s.dy/uh))); u != d.U && r.fits(s.dragging, u) {
				d.U = u
				changed = true
			}
			s.dragging, s.dy, s.dropped = -1, 0, false
		}
	})
	if changed {
		root.SetChanged()
	}
	return root
}

// device creates device i in its units, dragged by the pointer while held.
func device(c *ui.Context, s *State, r *Rack, i int, colors map[string]ui.Color, uh, width, rail float32) {
	t := c.Theme()
	d := r.Devices[i]
	top := uh * float32(r.Units-(d.U+d.Height-1))
	e := ui.Box(c).Absolute().Left(rail).Top(t.Space(2)+top).Size(width, uh*float32(d.Height)).
		Cursor(ui.CursorGrab).Role(ui.RoleGroup).Label(d.Name).
		Tooltip(fmt.Sprintf("%s\n%s, %s, %s", d.Name, d.Model, units(d), power(d.Watts)))
	if _, dy, ok := e.Dragged(); ok {
		if s.dragging != i {
			s.dragging, s.dy = i, 0
		}
		s.dy += dy
	}
	if _, _, ended := e.DragEnded(); ended && s.dragging == i {
		s.dropped = true
	}
	dragged := s.dragging == i
	if dragged {
		e.Translate(0, s.dy).Shadow(0, 6, 16, 0, ui.RGBA(0, 0, 0, 0.45)).Cursor(ui.CursorGrabbing)
	}
	col := colors[d.Role]
	e.Draw(func(p *ui.Painter, rc ui.Rect) {
		box := ui.Rect{X: rc.X + 1, Y: rc.Y + 1, W: rc.W - 2, H: rc.H - 2}
		p.FillGradient(box, ui.LinearGradient{From: col.Mix(ui.RGB(255, 255, 255), 0.08), To: col.Mix(ui.RGB(0, 0, 0), 0.2), Angle: 180}, 2)
		p.Stroke(box, ui.RGBA(0, 0, 0, 0.35), 2, 1)
		// Vents along the right of each unit, as on a server's bezel.
		for u := range d.Height {
			y := box.Y + uh*float32(u)
			for k := range 6 {
				x := box.X + box.W - t.Space(3) - float32(k)*5
				p.Fill(ui.Rect{X: x, Y: y + uh*0.28, W: 2, H: uh * 0.44}, ui.RGBA(0, 0, 0, 0.25), 1)
			}
		}
		led := ui.Oklch(0.8, 0.2, 145)
		p.Fill(ui.Rect{X: box.X + t.Space(1.5), Y: box.Y + uh/2 - 2.5, W: 5, H: 5}, led, 2.5)
		size := min(t.FontSize-2, uh-4)
		name := ui.Span{Text: d.Name, Size: size, Color: ui.RGB(255, 255, 255), Weight: 600}
		_, h := p.MeasureText(0, name)
		x := box.X + t.Space(4)
		p.RichText(x, box.Y+uh/2-h/2, 0, name, ui.Span{Text: "  " + d.Model, Size: size, Color: ui.RGBA(255, 255, 255, 0.7)})
	})
}

// stat creates a line of text over a bar filled to share.
func stat(c *ui.Context, text string, share float64, col ui.Color) {
	t := c.Theme()
	ui.Column(c).Gap(t.Space(1)).Children(func() {
		ui.Text(c, text).FontSize(t.FontSize - 1).TextColor(t.TextMuted).FontFeatures("tnum")
		ui.Box(c).Height(t.Space(1.5)).Background(t.Border).Radius(t.Space(1)).Draw(func(p *ui.Painter, r ui.Rect) {
			p.Fill(ui.Rect{X: r.X, Y: r.Y, W: r.W * float32(min(1, share)), H: r.H}, col, r.H/2)
		})
	})
}

func units(d Device) string {
	if d.Height == 1 {
		return "U" + strconv.Itoa(d.U)
	}
	return fmt.Sprintf("U%d–U%d", d.U, d.U+d.Height-1)
}

func power(w float64) string {
	if w >= 1000 {
		return strconv.FormatFloat(w/1000, 'f', 1, 64) + " kW"
	}
	return strconv.FormatFloat(w, 'f', 0, 64) + " W"
}
