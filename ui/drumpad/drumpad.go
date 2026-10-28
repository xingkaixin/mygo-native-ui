// Package drumpad is a grid of sixteen pads to play by hand, as an MPC's:
// pad 1 at the bottom left, each lighting in its color as it is hit, as
// bright as the hit is hard, and fading. A click is harder the nearer the
// middle of the pad, and the keys under 1 to 4 down to Z to V hit the
// pads laid out as they are. Full Level makes every hit as hard as can
// be. Above the pads, the last hit and how hard. The app plays the hits
// in the same frame.
//
//	ev := drumpad.View(c, &app.pads, kit)
//	for _, h := range ev.Hits {
//		sounds[h.Pad].Play().SetVolume(h.Velocity)
//	}
package drumpad

import (
	"fmt"
	"math"
	"time"

	"github.com/egoist/mygo/ui"
)

// Pad is a pad: its name and color.
type Pad struct {
	Name  string
	Color ui.Color
}

// Hit is a pad hit: which, from 0 for pad 1, and how hard, from 0 to 1.
type Hit struct {
	Pad      int
	Velocity float64
}

// State is what a grid keeps: when each pad was last hit and how hard,
// which the pointer is holding, and Full Level. Keep it in the app's
// state, one for each grid.
type State struct {
	hit       [16]time.Time
	velocity  [16]float64
	holding   int
	fullLevel bool
	last      int
	any       bool
}

// Events is a grid, and the hits in the frame, in the order made.
type Events struct {
	ui.Element
	Hits []Hit
}

// keys are the keys of the pads, from pad 1, laid out as the pads are.
var keys = [16]ui.Key{
	ui.KeyZ, ui.KeyX, ui.KeyC, ui.KeyV,
	ui.KeyA, ui.KeyS, ui.KeyD, ui.KeyF,
	ui.KeyQ, ui.KeyW, ui.KeyE, ui.KeyR,
	ui.Key1, ui.Key2, ui.Key3, ui.Key4,
}

var keyNames = [16]string{"Z", "X", "C", "V", "A", "S", "D", "F", "Q", "W", "E", "R", "1", "2", "3", "4"}

const fade = 400 * time.Millisecond

// View creates the grid of pads, sixteen at most.
func View(c *ui.Context, s *State, pads []Pad) Events {
	t := c.Theme()
	now := c.Now()
	var ev Events
	hit := func(i int, v float64) {
		if s.fullLevel {
			v = 1
		}
		s.hit[i], s.velocity[i], s.last, s.any = now, v, i, true
		ev.Hits = append(ev.Hits, Hit{Pad: i, Velocity: v})
	}
	for i := range min(len(pads), 16) {
		if c.Shortcut(0, keys[i]) {
			hit(i, 0.8)
		}
	}
	pressedAny := false
	root := ui.Column(c).Gap(t.Space(3))
	root.Children(func() {
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			readout := "Hit a pad"
			if s.any {
				readout = fmt.Sprintf("%s  ·  %d", pads[s.last].Name, int(math.Round(s.velocity[s.last]*127)))
			}
			ui.Text(c, readout).Font("monospace").FontFeatures("tnum").Grow(1).
				Padding(t.Space(1.5), t.Space(3)).Radius(t.Radius).Background(t.Surface)
			ui.Toggle(c, &s.fullLevel, "Full Level")
		})
		ui.Grid(c).Columns(4).Gap(t.Space(2)).Children(func() {
			for row := 3; row >= 0; row-- {
				for col := range 4 {
					i := row*4 + col
					if i >= len(pads) {
						ui.Box(c.Key(i))
						continue
					}
					b := s.pad(c, i, pads[i], now)
					if b.Pressed() {
						pressedAny = true
						if s.holding != i+1 {
							x, y, _ := b.PointerPosition()
							r := b.LayoutBounds()
							d := math.Hypot(float64(x-r.W/2)/float64(r.W/2), float64(y-r.H/2)/float64(r.H/2))
							s.holding = i + 1
							hit(i, max(0.15, 1-0.85*min(1, d/1.2)))
						}
					}
				}
			}
		})
	})
	if !pressedAny {
		s.holding = 0
	}
	for i := range s.hit {
		if left := fade - now.Sub(s.hit[i]); left > 0 {
			c.After(time.Second / 60)
			break
		}
	}
	if len(ev.Hits) > 0 {
		root.SetChanged()
	}
	ev.Element = root
	return ev
}

// pad creates pad i, lit by its last hit.
func (s *State) pad(c *ui.Context, i int, p Pad, now time.Time) ui.Element {
	t := c.Theme()
	glow := float32(0)
	if since := now.Sub(s.hit[i]); since < fade {
		k := 1 - float32(since)/float32(fade)
		glow = k * k * float32(0.35+0.65*s.velocity[i])
	}
	b := ui.ButtonBase(c.Key(i)).Size(t.Space(24), t.Space(24)).Radius(t.Radius+2).
		Border(1, t.Border).Label(fmt.Sprintf("Pad %d, %s", i+1, p.Name))
	b.Background(t.Surface.Mix(p.Color, 0.12+0.88*glow))
	if glow > 0 {
		b.Shadow(0, 0, 18*glow, 0, p.Color.Alpha(0.7*glow))
	}
	ink := t.Text
	if glow > 0.45 {
		ink = ui.RGB(255, 255, 255)
	}
	b.Children(func() {
		ui.Column(c).Fill().Padding(t.Space(2)).Children(func() {
			ui.Row(c).Children(func() {
				ui.Text(c, fmt.Sprint(i+1)).FontSize(t.FontSize - 2).FontFeatures("tnum").TextColor(ink.Alpha(0.6))
				ui.Spacer(c)
				ui.Text(c, keyNames[i]).Font("monospace").FontSize(t.FontSize - 3).TextColor(ink.Alpha(0.45))
			})
			ui.Spacer(c)
			ui.Text(c, p.Name).FontSize(t.FontSize - 1).Bold().TextColor(ink)
		})
	})
	return b
}
