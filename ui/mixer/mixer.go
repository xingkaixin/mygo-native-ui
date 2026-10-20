// Package mixer is a mixing console: a strip for each channel and the
// master's at the right. A channel's strip has, from the top, its pan
// knob, its Mute and Solo buttons, its fader on a scale in dB beside a
// meter of each side, and its gain and name. Dragging a fader moves it,
// ten times finer with Shift, the arrows step it while it has the focus,
// and a double click puts it back at 0 dB. The meters are green, amber
// from -12 dB and red from -3 dB, their peaks held a moment before they
// fall; the clip light over a meter lights at full scale and stays lit
// until clicked. While any channel is soloed, the others are silent, their
// M outlined.
//
// The app measures the peaks of each channel, after its fader and pan,
// and of the master, and gives those heard since the last frame to each
// frame, drawing frames while it plays. Gains returns the gain each
// channel is heard at, by the rule the view shows:
//
//	levels := app.peaks() // heard since the last frame
//	if mixer.View(c, &app.desk, app.channels, &app.master, levels).Changed() {
//		app.mix(mixer.Gains(app.channels), app.master)
//	}
package mixer

import (
	"math"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/knob"
)

// Channel is what a strip sets of a channel: its gain in dB, down to
// math.Inf(-1), its pan from the left (-1) to the right (1), and whether
// it is muted or soloed.
type Channel struct {
	Name       string
	Gain       float64
	Pan        float64
	Mute, Solo bool
}

// Levels are the peaks heard since the last frame, left and right, as
// amplitudes, 1 being full scale: those of each channel, in the order of
// the channels, and the master's.
type Levels struct {
	Channels [][2]float32
	Master   [2]float32
}

// Gains returns the gain each channel is heard at, as a factor of its
// samples: that of its fader, or 0 while it is muted, or while another is
// soloed and it is not.
func Gains(channels []Channel) []float64 {
	solo := soloing(channels)
	g := make([]float64, len(channels))
	for i, ch := range channels {
		if !ch.Mute && (ch.Solo || !solo) {
			g[i] = math.Pow(10, ch.Gain/20)
		}
	}
	return g
}

func soloing(channels []Channel) bool {
	return slices.ContainsFunc(channels, func(ch Channel) bool { return ch.Solo })
}

// marks are the marks of the scale and how far up a fader's travel each
// is, closer together toward the top, as on consoles. Below the lowest
// in dB, the travel is in proportion to the amplitude, down to −∞.
var marks = [...]struct {
	db, at float64
	label  string
}{
	{math.Inf(-1), 0, "−∞"}, {-40, 0.12, "−40"}, {-24, 0.32, "−24"},
	{-12, 0.52, "−12"}, {-6, 0.66, "−6"}, {0, 0.8, "0"}, {6, 1, "+6"},
}

// at returns how far up its travel a fader at db is, from 0 to 1.
func at(db float64) float64 {
	if lo := marks[1]; db <= lo.db {
		return lo.at * math.Pow(10, (db-lo.db)/20)
	}
	for i := 2; i < len(marks); i++ {
		if m, p := marks[i], marks[i-1]; db <= m.db {
			return p.at + (m.at-p.at)*(db-p.db)/(m.db-p.db)
		}
	}
	return 1
}

// gain returns the gain in dB of a fader f of the way up its travel.
func gain(f float64) float64 {
	if f <= 0 {
		return math.Inf(-1)
	}
	if lo := marks[1]; f <= lo.at {
		return lo.db + 20*math.Log10(f/lo.at)
	}
	for i := 2; i < len(marks); i++ {
		if m, p := marks[i], marks[i-1]; f <= m.at {
			return p.db + (m.db-p.db)*(f-p.at)/(m.at-p.at)
		}
	}
	return marks[len(marks)-1].db
}

func readout(db float64) string {
	switch {
	case math.IsInf(db, -1):
		return "−∞ dB"
	case math.Abs(db) < 0.05:
		return "0.0 dB"
	case db > 0:
		return "+" + strconv.FormatFloat(db, 'f', 1, 64) + " dB"
	}
	return strings.Replace(strconv.FormatFloat(db, 'f', 1, 64), "-", "−", 1) + " dB"
}

const (
	floor = -60.0
	// fall and drop are how fast meters and their peaks fall, in dB a
	// second; hold is how long a peak stays first.
	fall = 30.0
	drop = 20.0
	hold = 1500 * time.Millisecond
	// inset is how far the ends of a fader's travel are from the ends of
	// its strip's fader, scale and meter, leaving room for the cap and the
	// clip lights.
	inset = 14
)

// meter is a strip's meter: the level of each side as it falls, in dB,
// its peak and since when it is held, and whether it clipped.
type meter struct {
	level, peak [2]float64
	since       [2]time.Time
	clip        bool
}

// State is what a console keeps: its meters as they fall, their peaks
// and clip lights. Keep it in the app's state, one for each console.
type State struct {
	meters []meter
	last   time.Time
}

// update takes the levels heard, letting meters and peaks fall by the
// time since the last frame, and reports whether any is still above the
// floor.
func (s *State) update(now time.Time, levels Levels, n int) bool {
	if len(s.meters) != n+1 {
		s.meters = make([]meter, n+1)
		for i := range s.meters {
			s.meters[i].level = [2]float64{floor, floor}
			s.meters[i].peak = [2]float64{floor, floor}
		}
	}
	dt := 0.0
	if !s.last.IsZero() {
		dt = min(0.1, now.Sub(s.last).Seconds())
	}
	s.last = now
	moving := false
	for i := range s.meters {
		in := levels.Master
		if i < n {
			in = [2]float32{}
			if i < len(levels.Channels) {
				in = levels.Channels[i]
			}
		}
		m := &s.meters[i]
		for k, v := range in {
			m.clip = m.clip || v >= 1
			m.level[k] = max(20*math.Log10(float64(v)), m.level[k]-fall*dt, floor)
			if m.level[k] >= m.peak[k] {
				m.peak[k], m.since[k] = m.level[k], now
			} else if now.Sub(m.since[k]) > hold {
				m.peak[k] = max(floor, m.peak[k]-drop*dt)
			}
			moving = moving || m.level[k] > floor || m.peak[k] > floor
		}
	}
	return moving
}

var panKnob = knob.Knob{Label: "Pan", Min: -1, Max: 1, Format: func(v float64) string {
	switch n := math.Round(v * 100); {
	case n < 0:
		return "L " + strconv.Itoa(int(-n))
	case n > 0:
		return "R " + strconv.Itoa(int(n))
	}
	return "C"
}}

// View creates the console of channels and of the master, at *master dB,
// with the levels heard since the last frame. It sets the channels in
// place and reports a change of any, or of the master, as a change.
func View(c *ui.Context, s *State, channels []Channel, master *float64, levels Levels) ui.Element {
	t := c.Theme()
	if s.update(c.Now(), levels, len(channels)) {
		c.AnimationFrame()
	}
	solo := soloing(channels)
	root := ui.Row(c).Gap(t.Space(1)).AlignItems(ui.Stretch)
	root.Children(func() {
		for i := range channels {
			ch := &channels[i]
			strip(c.Key(i), root, &s.meters[i], ch.Name, &ch.Gain, func() {
				if knob.View(c, &ch.Pan, panKnob).Changed() {
					root.SetChanged()
				}
				ui.Row(c).Gap(t.Space(1.5)).Children(func() {
					button(c, root, &ch.Mute, "M", "Mute", t.Warning, ui.RGB(0, 0, 0).Alpha(0.85), solo && !ch.Solo && !ch.Mute)
					button(c, root, &ch.Solo, "S", "Solo", t.Accent, t.AccentText, false)
				})
			})
		}
		ui.Box(c).Width(t.Space(2))
		strip(c.Key("master"), root, &s.meters[len(channels)], "Master", master, func() {})
	})
	return root
}

// button builds the toggle of *on, lit in color while on, with text in
// ink, and outlined in color while silent, as a channel is by the solo of
// another.
func button(c *ui.Context, root ui.Element, on *bool, text, label string, color, ink ui.Color, silent bool) {
	t := c.Theme()
	b := ui.ToggleBase(c, on).Size(t.Space(7), t.Space(6)).Radius(t.Space(1.5)).Label(label)
	if b.Changed() {
		root.SetChanged()
	}
	bg, fg := t.Text.Alpha(0.07), t.TextMuted
	switch {
	case *on:
		bg, fg = color, ink
	case silent:
		b.Border(1.5, color)
		fg = color
	case b.Hovered():
		bg = t.Text.Alpha(0.13)
	}
	b.Background(bg).Children(func() {
		ui.Text(c, text).Bold().FontSize(t.FontSize - 1).TextColor(fg)
	})
}

// strip builds a strip: what top builds, the fader of *db with the scale
// and the meter m, and the gain and name.
func strip(c *ui.Context, root ui.Element, m *meter, name string, db *float64, top func()) {
	t := c.Theme()
	ui.Column(c).Width(t.Space(22)).Padding(t.Space(3), t.Space(2)).Gap(t.Space(3)).AlignItems(ui.Center).
		Background(t.Surface).Border(1, t.Border).Radius(t.Space(2)).Role(ui.RoleGroup).Label(name).Children(func() {
		ui.Column(c).Height(t.Space(25)).Gap(t.Space(2.5)).AlignItems(ui.Center).Children(top)
		ui.Row(c).Grow(1).MinHeight(t.Space(40)).AlignSelf(ui.Stretch).AlignItems(ui.Stretch).Gap(t.Space(1)).Justify(ui.Center).Children(func() {
			if fader(c, db) {
				root.SetChanged()
			}
			scale(c)
			bars(c, m)
		})
		ui.Text(c, readout(*db)).FontSize(t.FontSize-1).FontFeatures("tnum").TextAlign(ui.Center).SingleLine().
			AlignSelf(ui.Stretch).Padding(t.Space(1), 0).Radius(t.Space(1)).Background(t.Text.Alpha(0.06))
		ui.Text(c, name).Bold().FontSize(t.FontSize - 1).TextAlign(ui.Center).SingleLine().AlignSelf(ui.Stretch)
	})
}

// travel returns where a fader f of the way up its travel is in r.
func travel(r ui.Rect, f float64) float32 {
	return r.Y + inset + float32(1-f)*(r.H-2*inset)
}

// fader builds the fader of *db and reports whether it moved.
func fader(c *ui.Context, db *float64) bool {
	t := c.Theme()
	old := *db
	f := ui.Box(c).Width(t.Space(7)).Focusable().FocusRing(false).Cursor(ui.CursorResizeNS).
		Role(ui.RoleSlider).Label("Gain").Range(0, 1, at(*db)).Value(readout(*db))
	move := func(d float64) { *db = gain(max(0, min(1, at(*db)+d))) }
	if _, dy, ok := f.Dragged(); ok {
		length := float64(f.LayoutBounds().H - 2*inset)
		if length <= 0 {
			length = 200
		}
		d := -float64(dy) / length
		if c.Modifiers()&ui.Shift != 0 {
			d /= 10
		}
		move(d)
	}
	switch {
	case f.DoubleClicked():
		*db = 0
	case f.Shortcut(0, ui.KeyUp), f.Shortcut(0, ui.KeyRight):
		move(0.01)
	case f.Shortcut(ui.Shift, ui.KeyUp), f.Shortcut(ui.Shift, ui.KeyRight):
		move(0.001)
	case f.Shortcut(0, ui.KeyDown), f.Shortcut(0, ui.KeyLeft):
		move(-0.01)
	case f.Shortcut(ui.Shift, ui.KeyDown), f.Shortcut(ui.Shift, ui.KeyLeft):
		move(-0.001)
	}
	pos, active := at(*db), f.Pressed() || f.Focused()
	face := t.Background
	if t.Dark {
		face = t.Surface.Mix(t.Text, 0.2)
	}
	f.Draw(func(p *ui.Painter, r ui.Rect) {
		cx := r.X + r.W/2
		for _, mk := range marks {
			w := float32(1)
			if mk.db == 0 {
				w = 2
			}
			p.Line(r.X+3, travel(r, mk.at), r.X+r.W-3, travel(r, mk.at), w, t.Border)
		}
		top, bottom := travel(r, 1), travel(r, 0)
		p.Fill(ui.Rect{X: cx - 2, Y: top - 2, W: 4, H: bottom - top + 4}, ui.RGB(0, 0, 0).Alpha(0.3), 2)
		y := travel(r, pos)
		thumb := ui.Rect{X: r.X + 1, Y: y - 10, W: r.W - 2, H: 20}
		p.Shadow(thumb, 3, 0, 1, 3, 0, ui.RGB(0, 0, 0).Alpha(0.3))
		p.Fill(thumb, face, 3)
		p.Stroke(thumb, t.Border, 3, 1)
		index := t.Text
		if active {
			index = t.Accent
		}
		p.Line(thumb.X+3, y, thumb.X+thumb.W-3, y, 2, index)
		if f.FocusVisible() {
			p.FocusRing(thumb, [4]float32{3, 3, 3, 3})
		}
	})
	return *db != old
}

// scale builds the scale of the marks between a fader and its meter.
func scale(c *ui.Context) {
	t := c.Theme()
	ui.Box(c).Width(t.Space(5.5)).Draw(func(p *ui.Painter, r ui.Rect) {
		for _, mk := range marks {
			color := t.TextMuted
			if mk.db == 0 {
				color = t.Text
			}
			span := ui.Span{Text: mk.label, Size: t.FontSize - 3, Color: color, Features: "tnum"}
			w, h := p.MeasureText(0, span)
			p.RichText(r.X+(r.W-w)/2, travel(r, mk.at)-h/2, 0, span)
		}
	})
}

// bars builds the meter m, a bar for each side, and its clip light,
// which a click puts out.
func bars(c *ui.Context, m *meter) {
	t := c.Theme()
	zones := [...]struct {
		lo, hi float64
		color  ui.Color
	}{{math.Inf(-1), -12, t.Success}, {-12, -3, t.Warning}, {-3, 6, t.Danger}}
	zone := func(db float64) ui.Color {
		for _, z := range zones[:2] {
			if db < z.hi {
				return z.color
			}
		}
		return zones[2].color
	}
	level, peak := m.level, m.peak
	const bar, gap = 5, 2
	box := ui.Box(c).Width(2*bar + gap + 2)
	box.Draw(func(p *ui.Painter, r ui.Rect) {
		for k := range 2 {
			x := r.X + 1 + float32(k)*(bar+gap)
			for _, z := range zones {
				top, bottom := travel(r, at(z.hi)), travel(r, at(z.lo))
				p.Fill(ui.Rect{X: x, Y: top, W: bar, H: bottom - top}, z.color.Alpha(0.14), 0)
				if v := level[k]; v > floor && v > z.lo {
					lit := travel(r, at(min(v, z.hi)))
					p.Fill(ui.Rect{X: x, Y: lit, W: bar, H: bottom - lit}, z.color, 0)
				}
			}
			if v := peak[k]; v > floor {
				p.Fill(ui.Rect{X: x, Y: travel(r, at(v)) - 1, W: bar, H: 2}, zone(v), 0)
			}
		}
	})
	box.Children(func() {
		light := ui.ButtonBase(c).Absolute().Left(0).Top(0).Size(2*bar+gap+2, inset-4).Label("Clip")
		if light.Clicked() {
			m.clip = false
		}
		lit := m.clip
		light.Draw(func(p *ui.Painter, r ui.Rect) {
			color := t.Text.Alpha(0.12)
			if lit {
				color = t.Danger
			}
			for k := range 2 {
				p.Fill(ui.Rect{X: r.X + 1 + float32(k)*(bar+gap), Y: r.Y + 2, W: bar, H: r.H - 2}, color, 1)
			}
		})
	})
}
