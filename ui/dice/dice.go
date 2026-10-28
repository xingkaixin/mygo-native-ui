// Package dice rolls dice, as the dice trays of tabletop apps do: a row of
// dice, six-sided ones with pips and the others a shape with a number, and
// the total beside Roll. Roll, or Space while the tray has the focus, sets
// the dice tumbling, each landing a little after the one before; a click
// on a die holds it, ringed, and a held die stays as it is through a roll.
// Where the desktop asks for less motion, the dice land at once.
//
//	ev := dice.View(c, &app.tray, &app.dice)
//	if ev.Rolled {
//		app.score(app.dice)
//	}
package dice

import (
	"math"
	"math/rand/v2"
	"strconv"
	"time"

	"github.com/egoist/mygo/ui"
)

// Die is a die: how many sides it has, 4, 6, 8, 10, 12 or 20, the side
// it shows, and whether it is held.
type Die struct {
	Sides int
	Value int
	Held  bool
}

// State is what a tray keeps: when the roll began and where the dice will
// land. Keep it in the app's state, one for each tray.
type State struct {
	rolled  time.Time
	landing []int
}

// Events is a tray, and whether its dice landed from a roll in the frame,
// their values set.
type Events struct {
	ui.Element
	Rolled bool
}

const (
	tumble  = 700 * time.Millisecond
	stagger = 90 * time.Millisecond
)

// View creates the tray of *dice.
func View(c *ui.Context, s *State, dice *[]Die) Events {
	t := c.Theme()
	var ev Events
	ds := *dice
	now := c.Now()
	rolling := s.landing != nil
	elapsed := now.Sub(s.rolled)
	if rolling && elapsed >= tumble+time.Duration(len(ds))*stagger {
		for i := range ds {
			if i < len(s.landing) && s.landing[i] > 0 {
				ds[i].Value = s.landing[i]
			}
		}
		s.landing, rolling, ev.Rolled = nil, false, true
	}
	if rolling {
		c.After(time.Second / 60)
	}
	roll := func() {
		s.landing = make([]int, len(ds))
		for i, d := range ds {
			if !d.Held {
				s.landing[i] = 1 + rand.IntN(max(1, d.Sides))
			}
		}
		s.rolled = now
		if c.Preferences().ReduceMotion {
			s.rolled = now.Add(-tumble - time.Duration(len(ds))*stagger)
		}
		c.Invalidate()
	}
	root := ui.Column(c).Gap(t.Space(6)).AlignItems(ui.Center).Focusable().FocusRing(false).Label("Dice")
	if root.Shortcut(0, ui.KeySpace) && !rolling {
		roll()
	}
	root.Children(func() {
		ui.Row(c).Gap(t.Space(4)).Wrap().Justify(ui.Center).Children(func() {
			for i := range ds {
				d := &ds[i]
				shown, angle, lift := d.Value, float32(0), float32(0)
				if rolling && !d.Held {
					p := min(1, float64(elapsed-time.Duration(i)*stagger)/float64(tumble))
					p = max(0, p)
					if p < 1 {
						tick := int(elapsed / (70 * time.Millisecond))
						shown = 1 + int(uint32(tick*2654435761+i*40503)>>7)%max(1, d.Sides)
						dir := float32(1 - 2*(i%2))
						angle = dir * float32(math.Pow(1-p, 2)*540)
						lift = -float32(math.Abs(math.Sin(p*math.Pi*2.5)) * (1 - p) * 28)
					} else {
						shown = s.landing[i]
					}
				}
				die := face(c.Key(i), *d, shown).Rotate(angle).Translate(0, lift)
				if die.Clicked() && !rolling {
					d.Held = !d.Held
					die.SetChanged()
				}
			}
		})
		ui.Row(c).Gap(t.Space(4)).AlignItems(ui.Center).Children(func() {
			total := 0
			for i, d := range ds {
				v := d.Value
				if rolling && !d.Held && i < len(s.landing) {
					v = 0
				}
				total += v
			}
			sum := "–"
			if !rolling {
				sum = strconv.Itoa(total)
			}
			ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Baseline).Children(func() {
				ui.Text(c, "Total").TextColor(t.TextMuted)
				ui.Text(c, sum).FontSize(t.FontSize + 10).Bold().FontFeatures("tnum").MinWidth(t.Space(12))
			})
			held := 0
			for _, d := range ds {
				if d.Held {
					held++
				}
			}
			if ui.PrimaryButton(c, "Roll").Width(t.Space(28)).Disabled(rolling || held == len(ds)).Clicked() {
				roll()
			}
		})
	})
	ev.Element = root
	if ev.Rolled {
		root.SetChanged()
	}
	return ev
}

var ink = ui.Hex("#1e2430")

// face creates a die showing value.
func face(c *ui.Context, d Die, value int) ui.Element {
	t := c.Theme()
	size := t.Space(18)
	state := strconv.Itoa(value)
	if d.Held {
		state += ", held"
	}
	e := ui.ButtonBase(c).Size(size, size).Center().Label("d" + strconv.Itoa(d.Sides)).Value(state).Cursor(ui.CursorPointer)
	held, hover := d.Held, e.Hovered()
	e.Draw(func(p *ui.Painter, r ui.Rect) {
		body, edge := ui.Hex("#fbfaf6"), ui.RGBA(0, 0, 0, 0.18)
		if held {
			edge = t.Accent
		}
		if d.Sides == 6 {
			p.Shadow(r, size*0.18, 0, 4, 10, 0, ui.RGBA(0, 0, 0, 0.25))
			p.Fill(r, body, size*0.18)
			if hover && !held {
				p.Fill(r, ui.RGBA(0, 0, 0, 0.03), size*0.18)
			}
			outline(p, r, size*0.18, edge, held)
			for _, at := range pips[value] {
				var dot ui.Path
				dot.Circle(r.X+r.W*at[0], r.Y+r.H*at[1], size*0.085)
				col := ink
				if value == 1 {
					col = ui.Hex("#c8302c")
				}
				p.FillPath(&dot, col)
			}
			return
		}
		var shape ui.Path
		cx, cy, rad := r.X+r.W/2, r.Y+r.H/2, r.W/2
		pts := polygon(d.Sides)
		for k, a := range pts {
			x, y := cx+rad*float32(math.Cos(a)), cy+rad*float32(math.Sin(a))
			if k == 0 {
				shape.MoveTo(x, y)
			} else {
				shape.LineTo(x, y)
			}
		}
		shape.Close()
		// The shadow leaves its rectangle clear, so it is cast from a disc
		// that every shape covers, grown out to the shape's size.
		in := rad * 0.45
		p.Shadow(ui.Rect{X: cx - in, Y: cy - in, W: 2 * in, H: 2 * in}, in, 0, 6, 12, rad*0.4, ui.RGBA(0, 0, 0, 0.25))
		p.FillPath(&shape, colors[d.Sides])
		w := float32(1.5)
		if held {
			w = 3
		}
		p.StrokePath(&shape, w, edge)
		if hover && !held {
			p.FillPath(&shape, ui.RGBA(255, 255, 255, 0.08))
		}
	})
	if d.Sides != 6 {
		e.Children(func() {
			label := strconv.Itoa(value)
			if value == 6 || value == 9 {
				label += "."
			}
			ui.Text(c, label).FontSize(size * 0.3).Bold().TextColor(ui.RGB(255, 255, 255)).PassThrough()
		})
	}
	return e
}

// outline rings a six-sided die, thicker when it is held.
func outline(p *ui.Painter, r ui.Rect, radius float32, col ui.Color, held bool) {
	w := float32(1)
	if held {
		w = 3
	}
	k := w / 2
	var path ui.Path
	x0, y0, x1, y1 := r.X+k, r.Y+k, r.X+r.W-k, r.Y+r.H-k
	rad := radius - k
	path.MoveTo(x0+rad, y0)
	path.LineTo(x1-rad, y0)
	path.QuadTo(x1, y0, x1, y0+rad)
	path.LineTo(x1, y1-rad)
	path.QuadTo(x1, y1, x1-rad, y1)
	path.LineTo(x0+rad, y1)
	path.QuadTo(x0, y1, x0, y1-rad)
	path.LineTo(x0, y0+rad)
	path.QuadTo(x0, y0, x0+rad, y0)
	path.Close()
	p.StrokePath(&path, w, col)
}

// pips are where a six-sided die's pips stand, by value.
var pips = [...][][2]float32{
	1: {{0.5, 0.5}},
	2: {{0.28, 0.28}, {0.72, 0.72}},
	3: {{0.28, 0.28}, {0.5, 0.5}, {0.72, 0.72}},
	4: {{0.28, 0.28}, {0.72, 0.28}, {0.28, 0.72}, {0.72, 0.72}},
	5: {{0.28, 0.28}, {0.72, 0.28}, {0.5, 0.5}, {0.28, 0.72}, {0.72, 0.72}},
	6: {{0.28, 0.25}, {0.72, 0.25}, {0.28, 0.5}, {0.72, 0.5}, {0.28, 0.75}, {0.72, 0.75}},
}

var colors = map[int]ui.Color{
	4:  ui.Hex("#d9534f"),
	8:  ui.Hex("#3a7bd5"),
	10: ui.Hex("#2e9e6a"),
	12: ui.Hex("#8e5bd6"),
	20: ui.Hex("#e0892b"),
}

// polygon returns the angles of the corners of the outline a die of the
// sides given shows: a triangle, a diamond, a kite, a pentagon, a hexagon.
func polygon(sides int) []float64 {
	up := -math.Pi / 2
	regular := func(n int) []float64 {
		var a []float64
		for k := range n {
			a = append(a, up+2*math.Pi*float64(k)/float64(n))
		}
		return a
	}
	switch sides {
	case 4:
		return regular(3)
	case 8:
		return regular(4)
	case 10:
		return []float64{up, up + 1.9, up + math.Pi, up - 1.9}
	case 12:
		return regular(5)
	}
	return regular(6)
}
