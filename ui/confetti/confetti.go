// Package confetti throws confetti over the window, as apps cheer a task
// done or a goal met: paper squares, strips and dots in a set of colors,
// burst up from a point or showered from the top of the window, which
// tumble and flash their backs as they turn, drift from side to side, slow
// in the air and fall, fading as they go. It draws over everything and
// lets the pointer through, and throws nothing when the desktop asks for
// less motion.
//
//	if b := ui.Button(c, "Ship it"); b.Clicked() {
//		r := b.Bounds()
//		app.party.Burst(ui.Point{X: r.X + r.W/2, Y: r.Y + r.H/2})
//	}
//	confetti.View(c, &app.party)
package confetti

import (
	"math"
	"math/rand/v2"
	"time"

	"github.com/egoist/mygo/ui"
)

// State is what a view keeps: the throws in the air. Colors are the
// colors of the pieces, which the app may set; nil throws a bright set.
// Keep it in the app's state, one for each window.
type State struct {
	Colors []ui.Color
	throws []throw
}

// throw is a burst or a shower: when it started, the zero time until the
// view first shows it, and its pieces.
type throw struct {
	start  time.Time
	pieces []piece
}

// piece is a piece of confetti: where and when it starts, its velocity,
// its drag, its shape, size and color, how it sways and turns, and how
// long it lasts.
type piece struct {
	x, y, vx, vy, drag float64
	delay, life        float64
	shape              int
	w, h               float32
	color              ui.Color
	sway, swayRate     float64
	spin, flip, phase  float64
}

const gravity = 1000.0

const (
	square = iota
	strip
	dot
)

var bright = []ui.Color{
	ui.Hex("#ff5e5b"), ui.Hex("#ffb400"), ui.Hex("#00b2ca"), ui.Hex("#7d5ba6"),
	ui.Hex("#3ec46d"), ui.Hex("#ff7ac6"), ui.Hex("#2f80ed"),
}

// Burst throws a burst of confetti up and out from at, in window DIPs, as
// Element.Bounds gives them.
func (s *State) Burst(at ui.Point) {
	rnd := rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64()))
	pieces := make([]piece, 140)
	for i := range pieces {
		angle := -math.Pi/2 + (rnd.Float64()-0.5)*math.Pi*0.6
		speed := 700 + rnd.Float64()*900
		p := s.piece(rnd)
		p.x, p.y = float64(at.X), float64(at.Y)
		p.vx, p.vy = math.Cos(angle)*speed, math.Sin(angle)*speed
		pieces[i] = p
	}
	s.throws = append(s.throws, throw{pieces: pieces})
}

// Shower throws confetti down from across the top of the window, w DIPs
// wide, over a second.
func (s *State) Shower(w float32) {
	rnd := rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64()))
	pieces := make([]piece, int(max(80, w/5)))
	for i := range pieces {
		p := s.piece(rnd)
		p.x, p.y = rnd.Float64()*float64(w), -20-rnd.Float64()*40
		p.vx, p.vy = (rnd.Float64()-0.5)*200, 50+rnd.Float64()*250
		p.delay = rnd.Float64()
		p.life += 1
		pieces[i] = p
	}
	s.throws = append(s.throws, throw{pieces: pieces})
}

// piece returns a piece of a random shape and color, at rest.
func (s *State) piece(rnd *rand.Rand) piece {
	colors := s.Colors
	if len(colors) == 0 {
		colors = bright
	}
	p := piece{
		drag:     3 + rnd.Float64()*2.5,
		life:     2.4 + rnd.Float64()*1.4,
		color:    colors[rnd.IntN(len(colors))],
		sway:     10 + rnd.Float64()*25,
		swayRate: 2 + rnd.Float64()*3,
		spin:     (rnd.Float64() - 0.5) * 12,
		flip:     4 + rnd.Float64()*8,
		phase:    rnd.Float64() * 2 * math.Pi,
	}
	switch r := rnd.Float64(); {
	case r < 0.55:
		p.shape, p.w, p.h = square, 7+rnd.Float32()*5, 5+rnd.Float32()*3
	case r < 0.85:
		p.shape, p.w, p.h = strip, 3, 12+rnd.Float32()*6
	default:
		p.shape, p.w = dot, 3+rnd.Float32()*2
		p.h = p.w
	}
	return p
}

// at returns where p is t seconds after its start, under gravity and a
// drag on its speed, which brings it to fall at gravity/drag.
func (p piece) at(t float64) (x, y float64) {
	k := p.drag
	e := 1 - math.Exp(-k*t)
	x = p.x + p.vx/k*e + p.sway*math.Sin(p.swayRate*t+p.phase)*math.Min(1, t*2)
	y = p.y + (p.vy-gravity/k)/k*e + gravity/k*t
	return x, y
}

// View creates the confetti in the air over the window, and nothing once
// it has fallen.
func View(c *ui.Context, s *State) ui.Element {
	now := c.Now()
	if c.Preferences().ReduceMotion {
		s.throws = s.throws[:0]
	}
	left := 0.0
	kept := s.throws[:0]
	for _, th := range s.throws {
		if th.start.IsZero() {
			th.start = now
		}
		end := 0.0
		for _, p := range th.pieces {
			end = math.Max(end, p.delay+p.life)
		}
		if rest := end - now.Sub(th.start).Seconds(); rest > 0 {
			kept = append(kept, th)
			left = math.Max(left, rest)
		}
	}
	s.throws = kept
	if len(kept) == 0 {
		return ui.Element{}
	}
	// The view runs again when the last piece is gone, to take the layer
	// away.
	c.After(time.Duration(left * float64(time.Second)))
	throws := append([]throw(nil), kept...)
	w, h := c.Size()
	var layer ui.Element
	ui.Overlay(c, func() {
		layer = ui.Box(c).Absolute().Left(0).Top(0).Size(w, h).PassThrough().Role(ui.RoleNone).
			Draw(func(p *ui.Painter, r ui.Rect) {
				p.AnimationFrame()
				now := p.Now()
				for _, th := range throws {
					t := now.Sub(th.start).Seconds()
					for _, pc := range th.pieces {
						draw(p, pc, t-pc.delay)
					}
				}
			})
	})
	return layer
}

// draw paints pc t seconds after it was thrown.
func draw(p *ui.Painter, pc piece, t float64) {
	if t < 0 || t > pc.life {
		return
	}
	x, y := pc.at(t)
	alpha := float32(math.Min(1, (pc.life-t)/(pc.life*0.3)))
	if pc.shape == dot {
		var d ui.Path
		p.FillPath(d.Circle(float32(x), float32(y), pc.w/2), pc.color.Alpha(alpha))
		return
	}
	angle := pc.spin*t + pc.phase
	// The piece turns over about its long side: its width shrinks to a
	// line and its back, a shade darker, shows.
	turn := math.Cos(pc.flip*t + pc.phase)
	color := pc.color
	if turn < 0 {
		color = color.Mix(ui.RGB(0, 0, 0), 0.25)
	}
	hw, hh := float64(pc.w)/2*math.Abs(turn), float64(pc.h)/2
	sin, cos := math.Sincos(angle)
	corner := func(dx, dy float64) (float32, float32) {
		return float32(x + dx*cos - dy*sin), float32(y + dx*sin + dy*cos)
	}
	var shape ui.Path
	shape.MoveTo(corner(-hw, -hh))
	shape.LineTo(corner(hw, -hh))
	shape.LineTo(corner(hw, hh))
	shape.LineTo(corner(-hw, hh))
	p.FillPath(shape.Close(), color.Alpha(alpha))
}
