// Package gradient is an editor of a linear gradient, as design tools'
// fill panels edit one: a strip of its colors with a handle for each stop
// below it, dragged along it to move the stop, or away below to take it
// out; a click on the strip adds a stop of the color there. The stop
// chosen shows its color and its place, which Left and Right nudge and
// Delete takes out, and below them the gradient's angle and whether it
// mixes in Oklab.
//
//	gradient.Editor(c, &app.fillEditor, &app.fill)
//	ui.Box(c).Size(240, 160).LinearGradient(app.fill)
package gradient

import (
	"fmt"
	"math"
	"slices"

	"github.com/egoist/mygo/ui"
)

// State is what an editor keeps: the stop chosen, the identities of the
// stops as they pass one another, and how far below the strip the stop
// being dragged is. Keep it in the app's state, one for each editor.
type State struct {
	chosen int
	ids    []int
	next   int
	pull   float32
}

// Editor creates the editor of *g. A gradient of From and To takes stops
// of them.
func Editor(c *ui.Context, s *State, g *ui.LinearGradient) ui.Element {
	t := c.Theme()
	if len(g.Stops) == 0 {
		start, end := g.Start, g.End
		if start == 0 && end == 0 {
			end = 1
		}
		g.Stops = []ui.GradientStop{{Color: g.From, At: start}, {Color: g.To, At: end}}
	}
	if len(s.ids) != len(g.Stops) {
		s.ids = make([]int, len(g.Stops))
		for i := range s.ids {
			s.ids[i] = i
		}
		s.next = len(s.ids)
	}
	s.chosen = max(0, min(len(g.Stops)-1, s.chosen))
	knob := t.Space(3.5)
	return ui.Column(c).Gap(t.Space(3)).Children(func() {
		ui.Column(c).PaddingX(knob / 2).Children(func() {
			strip := ui.Box(c).Height(t.Space(8)).Radius(t.Space(1.5)).Border(1, t.Border).
				LinearGradient(ui.LinearGradient{Angle: 90, Stops: g.Stops, Oklab: g.Oklab}).
				Cursor(ui.CursorCopy).Role(ui.RoleButton).Label("Add a stop")
			w := strip.LayoutBounds().W
			if strip.Clicked() && w > 0 {
				if x, _, ok := strip.PointerPosition(); ok {
					s.add(g, max(0, min(1, x/w)))
				}
			}
			ui.Box(c).Height(knob + t.Space(1.5)).Children(func() {
				for i := 0; i < len(g.Stops); i++ {
					if s.handle(c.Key(s.ids[i]), g, i, w, knob) {
						i--
					}
				}
			})
		})
		i := s.chosen
		ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
			ui.ColorWell(c, &g.Stops[i].Color).Label("Stop color")
			at := math.Round(float64(g.Stops[i].At) * 100)
			if ui.NumberInput(c, &at, 0, 100, 1).Label("Stop place").Width(t.Space(36)).Changed() {
				g.Stops[i].At = float32(at / 100)
				s.chosen = s.sort(g, i)
			}
			ui.Text(c, "%").TextColor(t.TextMuted)
			ui.Box(c).Grow(1)
			if ui.Button(c, "Remove").Disabled(len(g.Stops) <= 2).Clicked() {
				s.remove(g, i)
			}
		})
		ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "Angle").TextColor(t.TextMuted).Width(t.Space(14))
			angle := float64(g.Angle)
			if ui.Slider(c, &angle, 0, 360).Label("Angle").Grow(1).Changed() {
				g.Angle = float32(math.Round(angle))
			}
			ui.Text(c, fmt.Sprintf("%d°", int(g.Angle))).FontFeatures("tnum").Width(t.Space(10)).TextAlign(ui.End)
		})
		ui.Checkbox(c, &g.Oklab, "Mix in Oklab")
	})
}

// handle shows the handle of stop i, below a strip w wide: a swatch of
// its color pointing up at its place. It reports the stop taken out.
func (s *State) handle(c *ui.Context, g *ui.LinearGradient, i int, w, size float32) bool {
	t := c.Theme()
	stop := g.Stops[i]
	h := ui.ButtonBase(c).Absolute().Left(stop.At*w-size/2).Top(t.Space(1)).Size(size, size).
		Cursor(ui.CursorGrab).Label(fmt.Sprintf("Stop at %d%%", int(math.Round(float64(stop.At)*100))))
	if h.Pressed() {
		s.chosen = i
	}
	if dx, dy, ok := h.Dragged(); ok && w > 0 {
		s.pull += dy
		if s.pull > t.Space(8) && len(g.Stops) > 2 {
			s.remove(g, i)
			s.pull = 0
			return true
		}
		g.Stops[i].At = max(0, min(1, g.Stops[i].At+dx/w))
		s.chosen = s.sort(g, i)
	}
	if _, _, ok := h.DragEnded(); ok {
		s.pull = 0
	}
	left, right := h.Shortcut(0, ui.KeyLeft), h.Shortcut(0, ui.KeyRight)
	if left || right {
		d := float32(0.01)
		if left {
			d = -d
		}
		g.Stops[i].At = max(0, min(1, g.Stops[i].At+d))
		s.chosen = s.sort(g, i)
	}
	if (h.Shortcut(0, ui.KeyDelete) || h.Shortcut(0, ui.KeyBackspace)) && len(g.Stops) > 2 {
		s.remove(g, i)
		return true
	}
	ring := t.Background
	if s.chosen == i {
		ring = t.Accent
	}
	color := stop.Color
	h.Draw(func(p *ui.Painter, r ui.Rect) {
		var tip ui.Path
		cx := r.X + r.W/2
		tip.MoveTo(cx-4, r.Y+1).LineTo(cx, r.Y-4).LineTo(cx+4, r.Y+1).Close()
		p.FillPath(&tip, ring)
		p.Fill(r, ring, 3)
		p.Fill(ui.Rect{X: r.X + 2, Y: r.Y + 2, W: r.W - 4, H: r.H - 4}, color, 2)
		p.Stroke(r, t.Text.Alpha(0.25), 3, 1)
	})
	return false
}

// sort moves stop i to its place among the others, with its identity, and
// returns its index there.
func (s *State) sort(g *ui.LinearGradient, i int) int {
	st := g.Stops
	for i > 0 && st[i].At < st[i-1].At {
		st[i], st[i-1] = st[i-1], st[i]
		s.ids[i], s.ids[i-1] = s.ids[i-1], s.ids[i]
		i--
	}
	for i < len(st)-1 && st[i].At > st[i+1].At {
		st[i], st[i+1] = st[i+1], st[i]
		s.ids[i], s.ids[i+1] = s.ids[i+1], s.ids[i]
		i++
	}
	return i
}

// add adds a stop at at, of the gradient's color there, and chooses it.
func (s *State) add(g *ui.LinearGradient, at float32) {
	k := slices.IndexFunc(g.Stops, func(st ui.GradientStop) bool { return st.At > at })
	if k < 0 {
		k = len(g.Stops)
	}
	color := g.Stops[max(0, k-1)].Color
	if k > 0 && k < len(g.Stops) {
		a, b := g.Stops[k-1], g.Stops[k]
		color = a.Color.Mix(b.Color, (at-a.At)/max(1e-6, b.At-a.At))
	} else if k == 0 {
		color = g.Stops[0].Color
	}
	g.Stops = slices.Insert(g.Stops, k, ui.GradientStop{Color: color, At: at})
	s.ids = slices.Insert(s.ids, k, s.next)
	s.next++
	s.chosen = k
}

// remove takes stop i out, choosing the one before it.
func (s *State) remove(g *ui.LinearGradient, i int) {
	g.Stops = slices.Delete(g.Stops, i, i+1)
	s.ids = slices.Delete(s.ids, i, i+1)
	s.chosen = max(0, i-1)
}
