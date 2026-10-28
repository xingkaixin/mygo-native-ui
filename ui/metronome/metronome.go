// Package metronome keeps time, as metronome apps do: a pendulum that
// swings to one side and back again each beat, its weight lower on the rod
// the faster the tempo, a dot for each beat of the bar, and the tempo with
// its Italian marking. Start, or Space while the metronome has the focus,
// starts and stops it; − and +, ↑ and ↓, dragging the number up and down,
// and Tap tapped in time change the tempo, Shift with the arrows by ten.
// A click on a beat's dot turns it from accented to plain to silent. The
// clicks are made as they play, higher on an accent, and a change is
// heard at once. Where the desktop asks for less motion, the pendulum
// stays upright and the dots alone keep time.
//
//	metronome.View(c, &app.metronome, &app.bpm, &app.beats)
package metronome

import (
	"fmt"
	"math"
	"slices"
	"strconv"
	"sync"
	"time"

	"github.com/egoist/mygo/audio"
	"github.com/egoist/mygo/ui"
)

// Beat is how a beat of the bar sounds.
type Beat uint8

const (
	Accent Beat = iota
	Plain
	Silent
)

var beatNames = [...]string{"accented", "plain", "silent"}

const (
	rate      = 48000
	slowest   = 30
	fastest   = 300
	swing     = 20 * math.Pi / 180
	tapExpiry = 2 * time.Second
)

// clicks are the sounds of an accented and a plain beat: a wood block's
// knock, a tone and its overtone dying away in a few milliseconds.
var clicks = sync.OnceValue(func() [2][]float32 {
	knock := func(f float64) []float32 {
		out := make([]float32, rate/25)
		for i := range out {
			t := float64(i) / rate
			v := math.Sin(2*math.Pi*f*t) + 0.4*math.Sin(2*math.Pi*f*2.76*t)*math.Exp(-t/0.003)
			out[i] = float32(0.6 * v * math.Exp(-t/0.008))
		}
		return out
	}
	return [2][]float32{knock(1760), knock(1175)}
})

// synth makes the clicks of a bar looped at a tempo. The first beat falls
// half a beat in, so that the pendulum swings out from upright to meet it.
type synth struct {
	mu    sync.Mutex
	beats []Beat
	step  float64 // frames a beat
	pos   int64
}

func (s *synth) SampleRate() int { return rate }
func (s *synth) Channels() int   { return 1 }
func (s *synth) Len() int64      { return -1 }

func (s *synth) SetPosition(frame int64) error {
	s.mu.Lock()
	s.pos = frame
	s.mu.Unlock()
	return nil
}

func (s *synth) Read(buf []float32) (int, error) {
	s.mu.Lock()
	beats, step, pos := s.beats, s.step, s.pos
	s.pos += int64(len(buf))
	s.mu.Unlock()
	k := clicks()
	lead := step / 2
	for i := range buf {
		f := float64(pos+int64(i)) - lead
		var v float32
		for n := int64(math.Floor(f / step)); n >= 0; n-- {
			off := int64(f - math.Round(float64(n)*step))
			if off >= int64(len(k[0])) {
				break
			}
			if b := beats[n%int64(len(beats))]; off >= 0 && b != Silent {
				v += k[b][off]
			}
		}
		buf[i] = v
	}
	return len(buf), nil
}

func frames(bpm float64) float64 { return rate * 60 / bpm }

// State is what a metronome keeps: its player, the tempo it plays, the
// taps of Tap and how far the number was dragged. Keep it in the app's
// state, one for each metronome.
type State struct {
	player *audio.Player
	synth  *synth
	bpm    float64
	taps   []time.Time
	drag   float32
}

// Playing reports whether the metronome plays.
func (s *State) Playing() bool { return s.player != nil && s.player.Playing() }

// beat returns how many beats have been heard, with the part of the next
// one gone by, from the half beat before the first.
func (s *State) beat() float64 {
	return s.player.Position().Seconds()*rate/s.synth.step - 0.5
}

// View creates a metronome at *bpm beats a minute, a bar of *beats; an
// empty bar becomes four beats, the first accented.
func View(c *ui.Context, s *State, bpm *float64, beats *[]Beat) ui.Element {
	t := c.Theme()
	*bpm = math.Round(max(slowest, min(fastest, *bpm)))
	if len(*beats) == 0 {
		*beats = []Beat{Accent, Plain, Plain, Plain}
	}
	if s.synth == nil {
		s.synth = &synth{step: frames(*bpm)}
		s.bpm = *bpm
		s.player = audio.NewPlayer(s.synth)
	}
	s.synth.mu.Lock()
	s.synth.beats = slices.Clone(*beats)
	old := s.synth.step
	s.synth.step = frames(*bpm)
	s.bpm = *bpm
	s.synth.mu.Unlock()
	playing := s.Playing()
	if playing && old != s.synth.step {
		// The player decoded ahead at the old tempo; from where it is
		// heard, in beats, it goes on at the new one.
		s.player.Seek(time.Duration(float64(s.player.Position()) * s.synth.step / old))
	}
	now := c.Now()
	current := -1
	if playing {
		b := s.beat()
		if b >= 0 {
			current = int(b) % len(*beats)
		}
		left := (math.Floor(b) + 1 - b) * s.synth.step / rate
		c.After(time.Duration(left * float64(time.Second)))
	}
	still := c.Preferences().ReduceMotion
	set := func(v float64) { *bpm = math.Round(max(slowest, min(fastest, v))) }
	toggle := func() {
		if playing {
			s.player.Stop()
		} else {
			s.player.Seek(0)
			s.player.Play()
		}
		c.Invalidate()
	}
	tap := func() {
		if n := len(s.taps); n > 0 && now.Sub(s.taps[n-1]) > tapExpiry {
			s.taps = nil
		}
		s.taps = append(s.taps, now)
		if len(s.taps) > 5 {
			s.taps = s.taps[1:]
		}
		if n := len(s.taps); n > 1 {
			set(60 / (s.taps[n-1].Sub(s.taps[0]).Seconds() / float64(n-1)))
		}
	}
	root := ui.Column(c).Gap(t.Space(5)).AlignItems(ui.Center).Focusable().FocusRing(false).Label("Metronome")
	switch {
	case root.Shortcut(0, ui.KeySpace):
		toggle()
	case root.Shortcut(0, ui.KeyUp), root.Shortcut(0, ui.KeyRight):
		set(*bpm + 1)
	case root.Shortcut(0, ui.KeyDown), root.Shortcut(0, ui.KeyLeft):
		set(*bpm - 1)
	case root.Shortcut(ui.Shift, ui.KeyUp):
		set(*bpm + 10)
	case root.Shortcut(ui.Shift, ui.KeyDown):
		set(*bpm - 10)
	case root.Shortcut(0, ui.KeyT):
		tap()
	}
	root.Children(func() {
		pendulum(c, s, *bpm, playing && !still)
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			for i := range *beats {
				dot(c.Key(i), &(*beats)[i], i, i == current)
			}
		})
		ui.Row(c).Gap(t.Space(4)).AlignItems(ui.Center).Children(func() {
			if ui.Button(c, "−").Width(t.Space(10)).Label("Slower").Disabled(*bpm <= slowest).Clicked() {
				set(*bpm - 1)
			}
			ui.Column(c).AlignItems(ui.Center).Width(t.Space(30)).Children(func() {
				n := ui.Text(c, strconv.Itoa(int(*bpm))).FontSize(t.FontSize + 40).Bold().FontFeatures("tnum").
					Cursor(ui.CursorResizeNS).Label("Tempo").Value(fmt.Sprintf("%d beats a minute", int(*bpm)))
				if _, dy, ok := n.Dragged(); ok {
					s.drag -= dy
					if steps := float32(math.Trunc(float64(s.drag / 4))); steps != 0 {
						set(*bpm + float64(steps))
						s.drag -= steps * 4
					}
				} else {
					s.drag = 0
				}
				ui.Text(c, marking(*bpm)+" · BPM").TextColor(t.TextMuted)
			})
			if ui.Button(c, "+").Width(t.Space(10)).Label("Faster").Disabled(*bpm >= fastest).Clicked() {
				set(*bpm + 1)
			}
		})
		ui.Row(c).Gap(t.Space(3)).Children(func() {
			if ui.Button(c, "Tap").Width(t.Space(24)).Tooltip("Tap in time, or press T").Clicked() {
				tap()
			}
			label := "Start"
			if playing {
				label = "Stop"
			}
			if ui.PrimaryButton(c, label).Width(t.Space(24)).Clicked() {
				toggle()
			}
		})
	})
	return root
}

// pendulum draws the metronome's case and its pendulum, swinging while
// it plays.
func pendulum(c *ui.Context, s *State, bpm float64, swinging bool) ui.Element {
	t := c.Theme()
	w, h := t.Space(56), t.Space(62)
	return ui.Box(c).Size(w, h).Draw(func(p *ui.Painter, r ui.Rect) {
		cx, base := r.X+r.W/2, r.Y+r.H
		var body ui.Path
		body.MoveTo(cx-r.W*0.46, base)
		body.LineTo(cx-r.W*0.3, r.Y+r.H*0.06)
		body.QuadTo(cx, r.Y, cx+r.W*0.3, r.Y+r.H*0.06)
		body.LineTo(cx+r.W*0.46, base)
		body.Close()
		p.FillPath(&body, t.Text.Alpha(0.05))
		p.StrokePath(&body, 1, t.Border)
		px, py := cx, r.Y+r.H*0.84
		rod := r.H * 0.66
		var arc ui.Path
		for i := 0; i <= 12; i++ {
			a := -swing + 2*swing*float64(i)/12
			in, out := float64(rod)+4, float64(rod)+8
			if i%6 == 0 {
				out += 4
			}
			arc.MoveTo(px+float32(in*math.Sin(a)), py-float32(in*math.Cos(a)))
			arc.LineTo(px+float32(out*math.Sin(a)), py-float32(out*math.Cos(a)))
		}
		p.StrokePath(&arc, 1, t.TextMuted.Alpha(0.6))
		angle := 0.0
		if swinging {
			angle = swing * math.Cos(math.Pi*s.beat())
			p.AnimationFrame()
		}
		sin, cos := float32(math.Sin(angle)), float32(math.Cos(angle))
		at := func(d float32) (float32, float32) { return px + d*sin, py - d*cos }
		tx, ty := at(rod)
		var rodPath ui.Path
		rodPath.MoveTo(px, py)
		rodPath.LineTo(tx, ty)
		p.StrokePath(&rodPath, 2.5, t.Text)
		// A real metronome slows as its weight is slid up the rod.
		wd := rod * float32(0.3+0.6*(fastest-bpm)/(fastest-slowest))
		wx, wy := at(wd)
		ww, wh := t.Space(7), t.Space(5)
		var weight ui.Path
		corners := [][2]float32{{-ww / 2, -wh / 2}, {ww / 2, -wh / 2}, {ww * 0.35, wh / 2}, {-ww * 0.35, wh / 2}}
		for i, k := range corners {
			x, y := wx+k[0]*cos-k[1]*sin, wy+k[0]*sin+k[1]*cos
			if i == 0 {
				weight.MoveTo(x, y)
			} else {
				weight.LineTo(x, y)
			}
		}
		weight.Close()
		col := t.Text
		if swinging {
			col = t.Accent
		}
		p.FillPath(&weight, col)
		var pivot ui.Path
		pivot.Circle(px, py, 4)
		p.FillPath(&pivot, t.Text)
	})
}

// dot creates the dot of the ith beat, lit while it sounds; a click turns
// it to how the next sounds.
func dot(c *ui.Context, b *Beat, i int, lit bool) ui.Element {
	t := c.Theme()
	size := t.Space(5)
	if *b == Accent {
		size = t.Space(7)
	}
	e := ui.ButtonBase(c).Size(t.Space(8), t.Space(8)).Cursor(ui.CursorPointer).
		Label(fmt.Sprintf("Beat %d", i+1)).Value(beatNames[*b])
	if e.Clicked() {
		*b = (*b + 1) % 3
		e.SetChanged()
	}
	kind, hover := *b, e.Hovered()
	e.Draw(func(p *ui.Painter, r ui.Rect) {
		col := t.Text
		if kind == Accent {
			col = t.Accent
		}
		var d ui.Path
		d.Circle(r.X+r.W/2, r.Y+r.H/2, size/2)
		switch {
		case kind == Silent:
			p.StrokePath(&d, 1.5, t.TextMuted.Alpha(0.6))
		case lit:
			p.FillPath(&d, col)
		case hover:
			p.FillPath(&d, col.Alpha(0.4))
		default:
			p.FillPath(&d, col.Alpha(0.22))
		}
	})
	return e
}

// marking returns the Italian name of a tempo.
func marking(bpm float64) string {
	switch {
	case bpm < 40:
		return "Grave"
	case bpm < 60:
		return "Largo"
	case bpm < 66:
		return "Larghetto"
	case bpm < 76:
		return "Adagio"
	case bpm < 108:
		return "Andante"
	case bpm < 120:
		return "Moderato"
	case bpm < 168:
		return "Allegro"
	case bpm < 200:
		return "Presto"
	}
	return "Prestissimo"
}
