// Package tour walks the user through an app's own interface, one element
// at a time, as onboarding tours do: the window dims but for a rounded
// cut-out around the element, which glides there from the step before,
// and a popover beside it, on whichever side has room, shows the step's
// title and text, where the step is in the tour ("2 of 5"), Back, Next,
// Done on the last step, and Skip. An element out of view scrolls into
// it. Escape ends the tour; Left and Right go to the step before and after.
//
// The app binds a ui.Handle to each element a step points at, and keeps
// whether the tour shows and the step it is at:
//
//	ui.Button(c, "Share").Bind(&app.share)
//	...
//	if ui.Button(c, "Take the tour").Clicked() {
//		app.touring, app.step = true, 0
//	}
//	tour.Show(c, &app.touring, &app.step, []tour.Step{
//		{Target: &app.share, Title: "Share a note", Text: "Send a link to the note."},
//	})
package tour

import (
	"fmt"
	"time"

	"github.com/egoist/mygo/ui"
)

// Step is a stop of a tour: the element it points at, which the app binds
// to the handle as it builds it, and the title and text of its popover.
type Step struct {
	Target *ui.Handle
	Title  string
	Text   string
}

// state is what a tour keeps while it shows: the step it last brought into
// view, and its popover, whose size decides where the next one goes.
type state struct {
	shown int
	pop   ui.Handle
}

// windowMargin is how far mygo keeps attached overlays from the window's
// edges.
const windowMargin = 4

// Show shows the tour while *open, at step *step of steps, over the
// window. Build it after the elements the steps point at, in the same
// frame. Back and Next, and Left and Right, change *step; Done, Skip and
// Escape set *open to false. While the tour shows it keeps the keyboard,
// and what is behind it takes neither the pointer nor the window's
// shortcuts; it gives the focus back as it ends. A step whose element does
// not show has its popover in the middle of the window. It returns the
// popover, or nil while the tour does not show.
func Show(c *ui.Context, open *bool, step *int, steps []Step) ui.Element {
	if !*open || len(steps) == 0 {
		return ui.Element{}
	}
	i := max(0, min(*step, len(steps)-1))
	*step = i
	t := c.Theme()
	ww, wh := c.Size()
	s := steps[i]
	var target ui.Element
	hole := ui.Rect{X: ww / 2, Y: wh / 2}
	if s.Target != nil {
		target = c.Resolve(*s.Target)
		if r := s.Target.Bounds(c); target.Valid() && r.W > 0 && r.H > 0 {
			pad := t.Space(1.5)
			hole = ui.Rect{X: r.X - pad, Y: r.Y - pad, W: r.W + 2*pad, H: r.H + 2*pad}
		}
	}
	dim := ui.RGBA(0, 0, 0, 0.45)
	// Over a dark window a lighter dim leaves the cut-out hard to tell apart.
	if t.Dark {
		dim = ui.RGBA(0, 0, 0, 0.65)
	}
	fade := []ui.Keyframe{{Value: 0}, {At: 200 * time.Millisecond, Value: 1}}
	var pop ui.Element
	ui.Overlay(c, func() {
		back := ui.Box(c).Absolute().Left(0).Top(0).Right(0).Bottom(0).Modal()
		st := ui.Local(back, "tour.state", func() state { return state{shown: -1} })
		if st.shown != i {
			st.shown = i
			if target.Valid() {
				target.ScrollIntoView()
			}
		}
		back.Opacity(back.Keyframes("in", fade))
		back.Children(func() {
			// The cut-out's shadow is the dim: it shows only outside the
			// cut-out, and spread past the window's corners it covers the rest.
			cut := ui.Box(c).Absolute().Radius(t.Space(2)).Border(1, t.Text.Alpha(0.25))
			cut.Left(cut.AnimateTo("x", hole.X, ui.Smooth)).Top(cut.AnimateTo("y", hole.Y, ui.Smooth)).
				Size(cut.AnimateTo("w", hole.W, ui.Smooth), cut.AnimateTo("h", hole.H, ui.Smooth)).
				Shadow(0, 0, 0, 2*(ww+wh), dim)

			w := min(t.Space(72), ww-2*windowMargin)
			// A popover of its own for each step fades in on its side,
			// rather than jumping there from the last step's.
			pop = ui.Column(c.Key(i)).Width(w).Gap(t.Space(2)).Padding(t.Space(4)).Radius(t.Space(2.5)).
				Background(t.Background).Border(1, t.Border).Shadow(0, 8, 24, 0, ui.RGBA(0, 0, 0, 0.25)).
				Role(ui.RoleDialog).Label(s.Title).Bind(&st.pop)
			h := st.pop.Bounds(c).H
			if h == 0 {
				h = t.Space(36) // about a popover's height, before one is laid out
			}
			// Attached to the cut-out rather than the target, the popover
			// glides with it and stays in the dialog's focus scope.
			beside(pop, cut, hole, w, h, ww, wh, t.Space(3))
			pop.Opacity(pop.Keyframes("in", fade))

			last := i == len(steps)-1
			if pop.OverlayShortcut(0, ui.KeyEscape) {
				*open = false
			}
			if pop.OverlayShortcut(0, ui.KeyLeft) && i > 0 {
				*step = i - 1
			}
			if pop.OverlayShortcut(0, ui.KeyRight) && !last {
				*step = i + 1
			}
			pop.Children(func() {
				ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
					ui.Text(c, s.Title).Bold().FontSize(t.FontSize + 1).Grow(1).Shrink(1)
					ui.Text(c, fmt.Sprintf("%d of %d", i+1, len(steps))).FontSize(t.FontSize - 1).
						TextColor(t.TextMuted).FontFeatures("tnum").SingleLine()
				})
				ui.Text(c, s.Text).TextColor(t.TextMuted).LineHeight(1.4)
				ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Margin(t.Space(2), 0, 0, 0).Children(func() {
					skip := ui.ButtonBase(c).Padding(t.Space(1), t.Space(2)).Margin(0, 0, 0, -t.Space(2)).Radius(t.Radius)
					if skip.Hovered() {
						skip.Background(t.Text.Alpha(0.06))
					}
					if skip.Clicked() {
						*open = false
					}
					skip.Children(func() { ui.Text(c, "Skip").TextColor(t.TextMuted) })
					ui.Spacer(c)
					if ui.Button(c, "Back").Disabled(i == 0).Clicked() {
						*step = i - 1
					}
					next := "Next"
					if last {
						next = "Done"
					}
					if ui.PrimaryButton(c, next).AutoFocus().Clicked() {
						if last {
							*open = false
						} else {
							*step = i + 1
						}
					}
				})
			})
		})
	})
	return pop
}

// beside attaches pop, w×h, to cut, at r in a window ww×wh, gap apart: below
// r, right of it, above it or left of it, the first with room for pop, or
// inside r along its bottom where none has room, as over a large editor,
// and in the middle of the window where r is empty.
func beside(pop, cut ui.Element, r ui.Rect, w, h, ww, wh, gap float32) {
	fits := func(room, size float32) bool { return room >= size+gap+windowMargin }
	switch {
	case r.W == 0:
		pop.AttachTo(cut, ui.AnchorCenter, ui.AnchorCenter)
	case fits(wh-r.Y-r.H, h):
		pop.AttachTo(cut, ui.AnchorBottom, ui.AnchorTop).Margin(gap, 0, 0, 0)
	case fits(ww-r.X-r.W, w):
		pop.AttachTo(cut, ui.AnchorRight, ui.AnchorLeft).Margin(0, 0, 0, gap)
	case fits(r.Y, h):
		pop.AttachTo(cut, ui.AnchorTop, ui.AnchorBottom).Margin(0, 0, gap, 0)
	case fits(r.X, w):
		pop.AttachTo(cut, ui.AnchorLeft, ui.AnchorRight).Margin(0, gap, 0, 0)
	default:
		pop.AttachTo(cut, ui.AnchorBottom, ui.AnchorBottom).Margin(0, 0, gap, 0)
	}
}
