// Package spinwheel is a wheel of chances, as for picking a name or a
// prize: a slice for each choice in turn of eight colors, with its label
// along it, and a pointer at the top. A click on the wheel, or Space or
// Enter, spins it a few turns, slowing to a stop; the pointer flicks as
// each slice goes past. The slice it stops at stays lit, and the others
// dim, until the next spin.
//
//	ev := spinwheel.View(c, &app.wheel, app.choices)
//	if ev.Landed {
//		app.picked = app.choices[ev.At]
//	}
package spinwheel

import (
	"math"
	"math/rand/v2"
	"time"

	"github.com/egoist/mygo/ui"
)

// State is what a wheel keeps: how far it has turned, and its spin. Keep
// it in the app's state, one for each wheel.
type State struct {
	angle, from, to float64
	start           time.Time
	length          time.Duration
	spun            bool
}

// Events is a wheel, and whether it stopped in the frame, at which slice.
type Events struct {
	ui.Element
	Landed bool
	At     int
}

var colors = []ui.Color{
	ui.Hex("#E63946"), ui.Hex("#F4A261"), ui.Hex("#E9C46A"), ui.Hex("#2A9D8F"),
	ui.Hex("#457B9D"), ui.Hex("#6D597A"), ui.Hex("#B5838D"), ui.Hex("#52B788"),
}

// Spin spins the wheel, unless it is spinning.
func (s *State) Spin(c *ui.Context) {
	if s.spinning(c.Now()) {
		return
	}
	s.from, s.start, s.spun = s.angle, c.Now(), true
	s.to = s.angle + 360*float64(4+rand.IntN(3)) + rand.Float64()*360
	s.length = 4*time.Second + time.Duration(rand.IntN(1500))*time.Millisecond
	if c.Preferences().ReduceMotion {
		s.length = 0
	}
}

func (s *State) spinning(now time.Time) bool {
	return !s.start.IsZero() && now.Sub(s.start) < s.length
}

// at returns the slice of n under the pointer when the wheel has turned by
// angle degrees.
func at(angle float64, n int) int {
	a := math.Mod(360-math.Mod(angle, 360), 360)
	return min(n-1, int(a/(360/float64(n))))
}

// View creates the wheel of choices.
func View(c *ui.Context, s *State, choices []string) Events {
	t := c.Theme()
	var ev Events
	n := max(1, len(choices))
	now := c.Now()
	if !s.start.IsZero() {
		k := 1.0
		if s.length > 0 {
			k = min(1, float64(now.Sub(s.start))/float64(s.length))
		}
		s.angle = s.from + (s.to-s.from)*(1-math.Pow(1-k, 4))
		if k >= 1 {
			s.angle = math.Mod(s.to, 360)
			s.start = time.Time{}
			ev.Landed, ev.At = true, at(s.angle, n)
		} else {
			c.AnimationFrame()
		}
	}
	spinning := !s.start.IsZero()
	lit := -1
	if s.spun && !spinning {
		lit = at(s.angle, n)
	}
	size := t.Space(96)
	label := ""
	if lit >= 0 && lit < len(choices) {
		label = choices[lit]
	}
	e := ui.Box(c).Size(size, size+t.Space(4)).Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton).
		Label("Spin the wheel").Value(label)
	if e.Clicked() || e.Shortcut(0, ui.KeySpace) || e.Shortcut(0, ui.KeyEnter) {
		s.Spin(c)
		e.SetChanged()
	}
	angle := s.angle
	e.Draw(func(p *ui.Painter, r ui.Rect) {
		cx, cy := r.X+r.W/2, r.Y+t.Space(4)+size/2
		rad := size/2 - 4
		seg := 360 / float64(n)
		var rim ui.Path
		rim.Circle(cx, cy, rad+4)
		p.FillPath(&rim, ui.Hex("#2B2D42"))
		point := func(deg float64, d float32) (float32, float32) {
			a := (deg - 90) * math.Pi / 180
			return cx + d*float32(math.Cos(a)), cy + d*float32(math.Sin(a))
		}
		for i := range n {
			a0 := angle + float64(i)*seg
			var wedge ui.Path
			wedge.MoveTo(cx, cy)
			steps := max(2, int(seg/3))
			for k := range steps + 1 {
				wedge.LineTo(point(a0+seg*float64(k)/float64(steps), rad))
			}
			wedge.Close()
			col := colors[i%len(colors)]
			if i == n-1 && n%len(colors) == 1 && n > 1 {
				col = colors[3]
			}
			if lit >= 0 && i != lit {
				col = col.Mix(ui.RGB(40, 40, 50), 0.55)
			}
			p.FillPath(&wedge, col)
			p.StrokePath(&wedge, 1.5, ui.RGBA(255, 255, 255, 0.8))
			if i < len(choices) {
				span := ui.Span{Text: choices[i], Size: 13, Weight: 700, Color: ui.RGB(255, 255, 255)}
				_, h := p.MeasureText(0, span)
				p.Transform(ui.Rotation(float32(a0+seg/2-90)), cx, cy, func() {
					p.RichText(cx+rad*0.3, cy-h/2, rad*0.62, span)
				})
			}
		}
		var hub ui.Path
		hub.Circle(cx, cy, rad*0.17)
		p.FillPath(&hub, ui.RGB(255, 255, 255))
		p.StrokePath(&hub, 3, ui.Hex("#2B2D42"))
		// The pointer flicks back as the edge between two slices passes it.
		flick := float32(0)
		if spinning {
			phase := math.Mod(math.Mod(360-math.Mod(angle, 360), 360), seg) / seg
			flick = float32(max(0, 1-phase/0.3)) * 24
		}
		p.Transform(ui.Rotation(-flick), cx, r.Y+t.Space(2), func() {
			var tip ui.Path
			tip.MoveTo(cx-t.Space(3.5), r.Y)
			tip.LineTo(cx+t.Space(3.5), r.Y)
			tip.LineTo(cx, r.Y+t.Space(9))
			tip.Close()
			p.FillPath(&tip, ui.Hex("#FFFFFF"))
			p.StrokePath(&tip, 2, ui.Hex("#2B2D42"))
		})
	})
	ev.Element = e
	return ev
}
