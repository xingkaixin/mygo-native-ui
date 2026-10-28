// Package hud shows a level as macOS does when a volume or brightness key
// is pressed: a square of Liquid Glass over the bottom of the window, an
// icon and a row of sixteen notches. It comes up as the level changes and
// fades out a moment after the last change; while the pointer is over it,
// it stays, and dragging along the notches sets the level.
//
//	if c.Shortcut(0, ui.KeyUp) {
//		app.volume = min(1, app.volume+1.0/16)
//	}
//	hud.View(c, &app.volumeHUD, speaker(app.volume), &app.volume)
//
// *level is from 0 to 1. The HUD shows whenever *level changes, by the app
// or by a drag, and when the app calls Show, as a volume key pressed at
// full volume does. Build it last in its parent, over the rest: it takes
// no room.
package hud

import (
	"time"

	"github.com/egoist/mygo/plugins/glass"
	"github.com/egoist/mygo/ui"
)

const (
	// linger is how long the HUD shows after the last change.
	linger = 1500 * time.Millisecond
	// notches is how many notches the level fills.
	notches = 16
)

var pop = ui.ElementTransition{
	Duration: 180 * time.Millisecond,
	Enter:    &ui.Motion{Scale: 0.9},
	Exit:     &ui.Motion{Scale: 0.97},
}

// State is what a HUD keeps: when it last came up, and the level it saw.
// Keep it in the app's state, one for each HUD.
type State struct {
	at         time.Time
	level      float64
	seen       bool
	show, hide bool
}

// Show brings the HUD up as if its level had changed.
func (s *State) Show() { s.show, s.hide = true, false }

// Hide takes the HUD down at once, as when another comes up in its place.
func (s *State) Hide() { s.hide, s.show = true, false }

// View creates a HUD showing icon and *level.
func View(c *ui.Context, s *State, icon *ui.SVG, level *float64) ui.Element {
	t := c.Theme()
	now := c.Now()
	*level = max(0, min(1, *level))
	if s.show || s.seen && *level != s.level {
		s.at = now
	}
	if s.hide {
		s.at = time.Time{}
	}
	s.level, s.seen, s.show, s.hide = *level, true, false, false

	holder := ui.Box(c).Attach(ui.AnchorBottom, ui.AnchorBottom).Bottom(t.Space(20))
	left := linger - now.Sub(s.at)
	if s.at.IsZero() || left <= 0 {
		return holder
	}
	c.After(left)
	holder.Children(func() {
		sq := ui.Column(c).Size(196, 196).Radius(32).Material(glass.Glass{}).Center().Gap(t.Space(6)).
			Transition(pop).Role(ui.RoleSlider).Range(0, 1, *level)
		if sq.Hovered() {
			s.at = now
		}
		sq.Children(func() {
			ui.Icon(c, icon).Size(84, 84).TextColor(t.Text.Alpha(0.85))
			bar := ui.Box(c).Size(148, 10).Cursor(ui.CursorPointer)
			if x, _, _ := bar.PointerPosition(); bar.Pressed() {
				*level = max(0, min(1, float64(x/148)))
				s.level, s.at = *level, now
			}
			v := float32(*level)
			bar.Draw(func(p *ui.Painter, r ui.Rect) {
				gap := float32(2)
				w := (r.W - gap*(notches-1)) / notches
				for i := range notches {
					cell := ui.Rect{X: r.X + float32(i)*(w+gap), Y: r.Y, W: w, H: r.H}
					p.Fill(cell, t.Text.Alpha(0.18), 1.5)
					if f := max(0, min(1, v*notches-float32(i))); f > 0 {
						p.Fill(ui.Rect{X: cell.X, Y: cell.Y, W: cell.W * f, H: cell.H}, t.Text.Alpha(0.85), 1.5)
					}
				}
			})
		})
	})
	return holder
}
