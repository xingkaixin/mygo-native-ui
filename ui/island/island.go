// Package island shows live activities in a black capsule at the top of
// the window, as the iPhone's Dynamic Island does: an activity in front
// shows a glance at each end, as a timer's icon and the time left, and
// one more running shows as a circle beside it, its own small glance,
// which a click brings to the front. A click on the capsule springs it
// open to what the activity shows in full; a click outside or Escape
// springs it back. The capsule grows and shrinks along a spring as
// activities come and go, and is gone when none run.
//
//	island.View(c, &app.island, []island.Activity{{
//		Leading:  func() { ui.Icon(c, timerIcon) },
//		Trailing: func() { ui.Text(c, left) },
//		Minimal:  func() { ui.Icon(c, timerIcon) },
//		Expanded: func() { app.timerPanel(c) },
//	}})
//
// Build it last in its parent, over the rest: it takes no room. What the
// activities build shows on black, in white text unless they color it.
package island

import (
	"time"

	"github.com/egoist/mygo/ui"
)

// Activity is a live activity: what it shows at the capsule's start and
// end, in a circle beside it when another is in front, and open.
type Activity struct {
	Leading, Trailing func()
	Minimal           func()
	Expanded          func()
}

// State is what an island keeps: the activity in front and whether it is
// open. Keep it in the app's state, one for each window.
type State struct {
	front int
	open  bool
}

var (
	ink    = ui.RGB(0, 0, 0)
	spring = ui.ElementTransition{Animation: &ui.Snappy, Size: true, Position: true, Enter: &ui.Motion{Scale: 0.5}}
	fade   = ui.ElementTransition{Duration: 220 * time.Millisecond, Enter: &ui.Motion{Opacity: 0}, Exit: &ui.Motion{Opacity: 0}}
	pop    = ui.ElementTransition{Animation: &ui.Snappy, Enter: &ui.Motion{Scale: 0.3}, Exit: &ui.Motion{Scale: 0.3}}
)

// View creates the island of activities, the one at s's front in the
// capsule.
func View(c *ui.Context, s *State, activities []Activity) ui.Element {
	t := c.Theme()
	holder := ui.Box(c).Attach(ui.AnchorTop, ui.AnchorTop).Top(t.Space(2))
	if len(activities) == 0 {
		s.front, s.open = 0, false
		return holder
	}
	s.front = min(s.front, len(activities)-1)
	front := activities[s.front]
	holder.Children(func() {
		pill := ui.ButtonBase(c.Key("island.pill")).Background(ink).TextColor(ui.RGB(255, 255, 255)).Clip().
			FocusRing(false).Transition(spring).Label("Live activity").Expanded(s.open)
		switch {
		case !s.open && pill.Clicked():
			s.open = true
		case s.open && (pill.PressedOutside() || pill.Shortcut(0, ui.KeyEscape)):
			s.open = false
		}
		if s.open {
			pill.Radius(t.Space(8)).Cursor(ui.CursorDefault)
		} else {
			pill.Radius(t.Space(4.5)).Height(t.Space(9))
		}
		pill.Children(func() {
			if s.open {
				ui.Column(c.Key("island.expanded")).Width(t.Space(92)).Padding(t.Space(5)).Transition(fade).Children(func() {
					if front.Expanded != nil {
						front.Expanded()
					}
				})
				return
			}
			ui.Row(c.Key("island.compact")).Height(t.Space(9)).Padding(0, t.Space(3.5)).Gap(t.Space(16)).
				AlignItems(ui.Center).Transition(fade).Children(func() {
				ui.Row(c).AlignItems(ui.Center).Children(func() {
					if front.Leading != nil {
						front.Leading()
					}
				})
				ui.Row(c).AlignItems(ui.Center).Children(func() {
					if front.Trailing != nil {
						front.Trailing()
					}
				})
			})
		})
		if s.open {
			return
		}
		for i, a := range activities {
			if i == s.front {
				continue
			}
			dot := ui.ButtonBase(c.Key(i)).Absolute().AttachTo(pill, ui.AnchorTopRight, ui.AnchorTopLeft).MarginStart(t.Space(1.5)).
				Size(t.Space(9), t.Space(9)).Radius(t.Space(4.5)).Background(ink).
				TextColor(ui.RGB(255, 255, 255)).Center().Clip().FocusRing(false).Transition(pop).Label("Show activity")
			if dot.Clicked() {
				s.front = i
			}
			dot.Children(func() {
				if a.Minimal != nil {
					a.Minimal()
				}
			})
			// One circle shows beside the capsule, as on the iPhone.
			break
		}
	})
	return holder
}
