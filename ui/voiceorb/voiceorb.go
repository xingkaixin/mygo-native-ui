// Package voiceorb shows the orb of a voice assistant's conversation
// mode, as ChatGPT's voice mode shows it: a glowing sphere whose surface
// flows through colors of the theme's accent, slowly drifting. It shows
// what the assistant does: at rest it breathes slowly; listening, it
// swells and ripples with the user's voice; thinking, it turns and
// swirls, calmer; speaking, it pulses outward with its own voice. It goes
// from one to the next, and follows the voice, along springs, so that it
// never jumps.
//
//	voiceorb.View(c, voiceorb.Listening, app.micLevel).Size(240, 240)
//
// The app gives the level of the voice each frame, from 0 to 1: the
// microphone's while the user speaks, the output's while the assistant
// does. The orb is drawn by a shader at the window's resolution, sharp at
// any size, and moves in frames that only paint it, without building the
// view. While the desktop asks for less motion, it is a still sphere of
// the colors' gradient whose size follows the level a little, and asks
// for no frames of its own; where the window draws without a GPU, it is a
// plain disc of them.
package voiceorb

import (
	"math"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/shader"
)

// Mode is what the assistant does, which the orb shows.
type Mode uint8

const (
	Idle Mode = iota
	Listening
	Thinking
	Speaking
)

// names are the modes' names, as assistive technology reads the orb's
// value, and the keys of their springs.
var names = [...]string{"Idle", "Listening", "Thinking", "Speaking"}

// rest is the orb's radius at rest, of the width it is drawn in, as the
// shader's: room around it to swell and glow.
const rest = 0.32

// breathing is how long a breath of the orb at rest takes, in
// milliseconds.
const breathing = 4800

// change is how the orb goes from one mode to another, and heed how it
// follows the voice: quickly, rising a little past it.
var (
	change = ui.Animation{Spring: ui.SpringOptions{Duration: 900 * time.Millisecond}}
	heed   = ui.Animation{Spring: ui.SpringOptions{Duration: 220 * time.Millisecond, Bounce: 0.2}}
)

// look is how the orb shows: how far it breathes, swells and its outline
// ripples, of its radius; how strong the waves going out across it are,
// how much it swirls and how bright it is; and how fast its surface
// drifts, turns and ripples, a second.
type look struct {
	breath, swell, ripple, pulse, twist, energy float32
	flow, spin, wave                            float64
}

// orb is what a view keeps from frame to frame: how far its surface has
// drifted, turned and rippled as of painted; the level it shows while the
// desktop asks for less motion, as of eased; and its shader's values.
type orb struct {
	painted          time.Time
	flow, spin, wave float64
	level            float32
	eased            time.Time
	values           *shader.Values
}

// View creates the orb showing mode, at level, from 0 to 1, of the voice
// it listens to or speaks with, which the other modes leave out. Its
// surface flows through colors, three at most; from one, or none, it
// derives three, or from the theme's accent. The orb is as wide as the
// smaller of the element's width and height, in its middle; give it a
// size, or Grow.
func View(c *ui.Context, mode Mode, level float32, colors ...ui.Color) ui.Element {
	t := c.Theme()
	still := c.Preferences().ReduceMotion
	level = max(0, min(1, level))
	box := ui.Box(c).Role(ui.RoleImage).Label("Assistant").Value(names[mode])
	o := ui.Local(box, "voiceorb.orb", func() orb { return orb{values: orbShader.NewValues()} })
	var w [len(names)]float32
	for i, name := range names {
		target := float32(0)
		if Mode(i) == mode {
			target = 1
		}
		w[i] = box.AnimateTo(name, target, change)
	}
	heard := max(0, box.AnimateTo("level", level, heed))
	idle, listening, thinking, speaking := w[0], w[1], w[2], w[3]
	voice := listening + speaking
	lk := look{
		breath: 0.035*idle + 0.02*thinking,
		swell:  (0.16*listening + 0.12*speaking) * heard,
		ripple: 0.008 + listening*(0.012+0.07*heard),
		pulse:  speaking * (0.3 + 0.7*heard),
		twist:  0.3 + 2.4*thinking,
		energy: 0.15*idle + 0.3*thinking + voice*(0.2+0.8*heard),
		flow:   float64(0.1*idle + 0.07*thinking + voice*(0.16+0.5*heard)),
		spin:   float64(0.05 + 0.85*thinking),
		wave:   float64(1 + listening*(1+5*heard) + speaking*(2+4*heard)),
	}
	if still {
		if !o.ease(c.Now(), level) {
			c.AnimationFrame()
		}
		lk = look{swell: 0.06 * voice * o.level, energy: 0.2}
	}
	pal := palette(t.Accent, colors)
	glow := float32(0.4)
	if t.Dark {
		glow = 0.55
	}
	return box.Draw(func(p *ui.Painter, r ui.Rect) {
		d := min(r.W, r.H)
		sq := ui.Rect{X: r.X + (r.W-d)/2, Y: r.Y + (r.H-d)/2, W: d, H: d}
		if !p.Accelerated() {
			disc(p, sq, pal, 1+lk.swell, glow)
			return
		}
		now := p.Now()
		if !still {
			if !o.painted.IsZero() {
				dt := now.Sub(o.painted).Seconds()
				o.flow = math.Mod(o.flow+lk.flow*dt, 32)
				o.spin = math.Mod(o.spin+lk.spin*dt, 2*math.Pi)
				o.wave = math.Mod(o.wave+lk.wave*dt, 2*math.Pi)
			}
			p.AnimationFrame()
		}
		o.painted = now
		breath := lk.breath * float32(math.Sin(2*math.Pi*float64(now.UnixMilli()%breathing)/breathing))
		v := o.values.Color(uA, pal[0]).Color(uB, pal[1]).Color(uC, pal[2]).
			Float(uScale, 1+breath+lk.swell).Float(uRipple, lk.ripple).
			Float(uFlow, float32(o.flow)).Float(uSpin, float32(o.spin)).Float(uWave, float32(o.wave)).
			Float(uTwist, lk.twist).Float(uPulse, lk.pulse).Float(uEnergy, lk.energy).Float(uGlow, glow).
			Bool(uStill, still)
		orbShader.Draw(p, sq, shader.Options{Values: v})
	})
}

// ease moves the level shown toward level, as of now, and reports whether
// it got there.
func (o *orb) ease(now time.Time, level float32) bool {
	if o.eased.IsZero() {
		o.level = level
	}
	k := float32(1 - math.Exp(-now.Sub(o.eased).Seconds()/0.2))
	o.level += (level - o.level) * min(1, k)
	o.eased = now
	if math.Abs(float64(level-o.level)) < 0.002 {
		o.level = level
	}
	return o.level == level
}

// disc paints the orb in sq where the window draws without a GPU, on which
// its shader would cost too much: a disc of the colors' gradient,
// scale times its size at rest, with a soft glow.
func disc(p *ui.Painter, sq ui.Rect, pal [3]ui.Color, scale, glow float32) {
	d := sq.W * 2 * rest * scale
	b := ui.Rect{X: sq.X + (sq.W-d)/2, Y: sq.Y + (sq.H-d)/2, W: d, H: d}
	p.Shadow(b, d/2, 0, 0, d*0.2, 0, pal[0].Alpha(glow))
	var circle ui.Path
	p.FillPathGradient(circle.Circle(b.X+d/2, b.Y+d/2, d/2), ui.LinearGradient{From: pal[2], To: pal[0], Angle: 135, Oklab: true})
}

// palette returns the three colors the orb flows through: the first
// three of colors, or, from the first of them or else from accent, it, a
// lighter one of a hue 30° on and a pale one 20° back.
func palette(accent ui.Color, colors []ui.Color) [3]ui.Color {
	if len(colors) >= 2 {
		return [3]ui.Color{colors[0], colors[1], colors[min(2, len(colors)-1)]}
	}
	if len(colors) == 1 {
		accent = colors[0]
	}
	l, ch, h := lch(accent)
	l, ch = max(0.5, min(0.68, l)), max(0.12, ch)
	return [3]ui.Color{ui.Oklch(l, ch, h), ui.Oklch(l+0.12, ch*0.85, h+30), ui.Oklch(0.94, ch*0.25, h-20)}
}

// lch returns c's lightness, chroma and hue, in degrees, in OKLCH.
func lch(c ui.Color) (l, ch, h float32) {
	c = c.SRGB()
	linear := func(v uint8) float64 {
		x := float64(v) / 255
		if x <= 0.04045 {
			return x / 12.92
		}
		return math.Pow((x+0.055)/1.055, 2.4)
	}
	r, g, b := linear(c.R), linear(c.G), linear(c.B)
	lm := math.Cbrt(0.4122214708*r + 0.5363325363*g + 0.0514459929*b)
	m := math.Cbrt(0.2119034982*r + 0.6806995451*g + 0.1073969566*b)
	s := math.Cbrt(0.0883024619*r + 0.2817188376*g + 0.6299787005*b)
	ll := 0.2104542553*lm + 0.7936177850*m - 0.0040720468*s
	aa := 1.9779984951*lm - 2.4285922050*m + 0.4505937099*s
	bb := 0.0259040371*lm + 0.7827717662*m - 0.8086757660*s
	return float32(ll), float32(math.Hypot(aa, bb)), float32(math.Atan2(bb, aa) * 180 / math.Pi)
}
