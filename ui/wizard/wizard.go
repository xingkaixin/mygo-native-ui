// Package wizard walks the user through a flow of steps, as setup
// assistants and installers do: the step's title, subtitle and content,
// Back and Continue below them, and the steps beside them, listed down the
// side as macOS installers list them, or along the top as numbered dots
// joined by a line. The current step is bold, the steps done are checked,
// and a step whose check failed is marked. A click on a step done goes
// back to it; the steps after the first one not yet done cannot be
// reached.
//
// Continue runs the step's check. When it returns what is wrong, the step
// stays and the message shows by the buttons, until the check passes. On
// the last step Continue reads as the app's label, as "Create", and
// confirms the flow. The new step slides in from the side the user goes
// to, unless the desktop asks for less motion, and the keyboard focus
// moves into it. Enter continues, unless the focus is in a multi-line
// field; a single-line text input takes Enter itself, so continue on its
// Submitted with State.Continue.
//
//	steps := []wizard.Step{
//		{Title: "Name", Build: func() {
//			if ui.TextInput(c, &app.name).AutoFocus().Submitted() {
//				app.wizard.Continue()
//			}
//		}, Check: func() string {
//			if app.name == "" {
//				return "Enter a name."
//			}
//			return ""
//		}},
//		{Title: "Summary", Build: func() { ui.Text(c, app.name) }},
//	}
//	if wizard.View(c, &app.wizard, &app.step, steps, wizard.Sidebar, "Create").Done() {
//		app.create()
//	}
package wizard

import (
	"slices"
	"strconv"
	"time"

	"github.com/egoist/mygo/ui"
)

// Step is a step of a wizard: its title, a subtitle below it unless "",
// Build, which builds its content, and Check, which returns what is wrong
// with what the user gave in the step, or "" when nothing is. A step
// without a Check always passes.
type Step struct {
	Title    string
	Subtitle string
	Build    func()
	Check    func() string
}

// Indicator is where a wizard shows its steps.
type Indicator int

const (
	// Sidebar lists the steps down the left side, as macOS installers do.
	Sidebar Indicator = iota
	// Top shows the steps along the top, numbered dots joined by a line,
	// with their titles below.
	Top
)

// Events are what the user did with a wizard in a frame. The zero Events
// report nothing.
type Events struct{ done bool }

// Done reports that the user confirmed the last step and its check
// passed, for the app to do what the wizard was for.
func (e Events) Done() bool { return e.done }

// State is what a wizard keeps: how many steps are done, what is wrong
// with those whose check failed, and the steps showing as they slide.
// Keep it in the app's state, one for each wizard, and set it to the zero
// State to start over.
type State struct {
	done     int
	problems map[int]string
	pages    []page
	moved    bool
	next     bool
}

// page is a step showing, at its place on a strip that grows by a page to
// the right going forward and to the left going back, however far.
type page struct {
	step int
	at   float32
}

// Continue continues as the Continue button does. Call it as Enter is
// pressed in a single-line text input of the step, which takes Enter
// itself:
//
//	if ui.TextInput(c, &app.name).Submitted() {
//		app.wizard.Continue()
//	}
func (s *State) Continue() { s.next = true }

type status int

const (
	pending status = iota
	current
	done
	problem
)

func (s *State) status(i, cur int) status {
	switch {
	case s.problems[i] != "":
		return problem
	case i == cur:
		return current
	case i < s.done:
		return done
	}
	return pending
}

// check runs the check of step st, number i, keeps what is wrong with
// it, and returns it, "" when nothing is.
func (s *State) check(st Step, i int) string {
	msg := ""
	if st.Check != nil {
		msg = st.Check()
	}
	if msg == "" {
		delete(s.problems, i)
		return ""
	}
	if s.problems == nil {
		s.problems = map[int]string{}
	}
	s.problems[i] = msg
	return msg
}

var slide = ui.Animation{Spring: ui.SpringOptions{Duration: 350 * time.Millisecond}}

// View creates a wizard of steps at step *step, which fills its parent,
// with the steps shown as at says, and finish the label of Continue on
// the last step, "Continue" when "". Back, Continue and clicks on the
// steps change *step; the app sets it to go to a step, and the steps
// before it count as done. While the step shows what is wrong, the wizard
// runs its check again in every frame, and the message goes once it
// passes.
func View(c *ui.Context, s *State, step *int, steps []Step, at Indicator, finish string) Events {
	var ev Events
	t := c.Theme()
	n := len(steps)
	root := ui.Row(c).Fill().AlignItems(ui.Stretch)
	if at == Top {
		root.Column()
	}
	if n == 0 {
		return ev
	}
	cur := max(0, min(*step, n-1))
	*step = cur
	s.done = max(s.done, cur)
	fresh := len(s.pages) == 0
	if fresh {
		s.pages = []page{{cur, 0}}
	} else if last := s.pages[len(s.pages)-1]; last.step != cur {
		d := float32(1)
		if cur < last.step {
			d = -1
		}
		s.pages = slices.DeleteFunc(s.pages, func(p page) bool { return p.step == cur })
		s.pages = append(s.pages, page{cur, last.at + d})
		s.moved = true
	}
	if s.problems[cur] != "" {
		s.check(steps[cur], cur)
	}

	// goTo goes to step to, n to confirm the last, running the checks of
	// the steps it passes going forward; it stops at the first that fails.
	goTo := func(to int) {
		for i := cur; i < to; i++ {
			if msg := s.check(steps[i], i); msg != "" {
				c.Announce(msg)
				*step = i
				return
			}
			s.done = max(s.done, i+1)
		}
		if to == n {
			ev.done = true
			return
		}
		*step = to
	}

	root.Children(func() {
		if at == Top {
			dots(c, s, steps, cur, goTo)
		} else {
			list(c, s, steps, cur, goTo)
		}
		ui.Divider(c)
		ui.Column(c).Grow(1).Shrink(1).MinWidth(0).MinHeight(0).Children(func() {
			area := ui.Box(c).Grow(1).Shrink(1).MinHeight(0).Clip()
			target := s.pages[len(s.pages)-1].at
			var pos float32
			if fresh {
				pos = area.Jump("x", target)
			} else {
				pos = area.AnimateTo("x", target, slide)
			}
			area.Children(func() {
				keep := s.pages[:0]
				for _, p := range s.pages {
					off := p.at - pos
					if p.step != cur && (p.step >= n || off <= -1 || off >= 1) {
						continue
					}
					keep = append(keep, p)
					s.page(c, steps[p.step], p.step, off, p.step == cur)
				}
				s.pages = keep
			})
			ui.Divider(c)
			ui.Row(c).Gap(t.Space(2)).Padding(t.Space(3), t.Space(4)).AlignItems(ui.Center).Children(func() {
				ui.Row(c).Grow(1).Shrink(1).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
					if msg := s.problems[cur]; msg != "" {
						glyph(c, problem, t.Space(4.5))
						ui.Text(c, msg).TextColor(t.Danger).Shrink(1)
					}
				})
				if ui.Button(c, "Back").Disabled(cur == 0).Clicked() {
					goTo(cur - 1)
				}
				label := "Continue"
				if cur == n-1 && finish != "" {
					label = finish
				}
				clicked := ui.PrimaryButton(c, label).Clicked()
				enter := c.Shortcut(0, ui.KeyEnter)
				if clicked || enter || s.next {
					s.next = false
					goTo(cur + 1)
				}
			})
		})
	})
	return ev
}

// page builds step st, number i, off its width to the right of where it
// rests: the current step, or the one it slides in over.
func (s *State) page(c *ui.Context, st Step, i int, off float32, cur bool) {
	t := c.Theme()
	pg := ui.Scroll(c.Key(i)).Absolute().Top(0).StartPercent(off * 100).WidthPercent(100).HeightPercent(100).
		Role(ui.RoleGroup).Label(st.Title).FocusRing(false)
	if cur {
		// Focusable only while it has the focus, the step takes it as it
		// comes in, and Tab goes on into it, past it; a field in it with
		// AutoFocus takes the focus instead.
		if s.moved {
			s.moved = false
			pg.Focus()
		} else if pg.Focused() {
			pg.Focusable()
		}
	}
	pg.Children(func() {
		ui.Column(c).Padding(t.Space(6), t.Space(8)).Gap(t.Space(5)).Children(func() {
			ui.Column(c).Gap(t.Space(1)).Children(func() {
				ui.Text(c, st.Title).FontSize(t.FontSize + 7).Bold()
				if st.Subtitle != "" {
					ui.Text(c, st.Subtitle).TextColor(t.TextMuted)
				}
			})
			if st.Build != nil {
				st.Build()
			}
		})
	})
}

// describe tells assistive technology how a step goes.
func (s *State) describe(e ui.Element, i int, st status) {
	switch st {
	case problem:
		e.Description(s.problems[i])
	case current:
		e.Description("Current step")
	case done:
		e.Description("Done")
	}
}

// list builds the steps down the side, each with a glyph of how it goes.
func list(c *ui.Context, s *State, steps []Step, cur int, goTo func(int)) {
	t := c.Theme()
	ui.Column(c).Width(t.Space(56)).Padding(t.Space(6), t.Space(3)).Gap(t.Space(0.5)).Background(t.Text.Alpha(0.03)).
		FocusGroup(ui.Vertical).Role(ui.RoleGroup).Label("Steps").Children(func() {
		for i, step := range steps {
			st := s.status(i, cur)
			row := ui.ButtonBase(c.Key(i)).Justify(ui.Start).Gap(t.Space(2.5)).Padding(t.Space(1.5), t.Space(2)).
				Radius(t.Radius).Label(step.Title).Disabled(i > s.done)
			s.describe(row, i, st)
			if row.Hovered() && i != cur {
				row.Background(t.Text.Alpha(0.05))
			}
			if row.Clicked() && i != cur {
				goTo(i)
			}
			row.Children(func() {
				glyph(c, st, t.Space(4.5))
				title := ui.Text(c, step.Title).Shrink(1)
				if i == cur {
					title.Bold()
				}
			})
		}
	})
}

// glyph shows how a step goes in a box size wide: a ring for a step to
// come, a ring around a dot for the current one, a check for one done and
// an exclamation mark for one with a problem.
func glyph(c *ui.Context, st status, size float32) {
	t := c.Theme()
	ui.Box(c).Size(size, size).Draw(func(p *ui.Painter, r ui.Rect) {
		cx, cy, rad := r.X+r.W/2, r.Y+r.H/2, r.W*0.42
		var circle ui.Path
		switch st {
		case pending:
			circle.Circle(cx, cy, rad-0.75)
			p.StrokePath(&circle, 1.5, t.Text.Alpha(0.25))
		case current:
			circle.Circle(cx, cy, rad-0.75)
			p.StrokePath(&circle, 1.5, t.Accent)
			p.Fill(ui.Rect{X: cx - rad*0.45, Y: cy - rad*0.45, W: rad * 0.9, H: rad * 0.9}, t.Accent, rad*0.45)
		case done:
			circle.Circle(cx, cy, rad)
			p.FillPath(&circle, t.Accent)
			check(p, cx, cy, rad*0.5, t.AccentText)
		case problem:
			circle.Circle(cx, cy, rad)
			p.FillPath(&circle, t.Danger)
			bang(p, cx, cy, rad*0.5, ui.RGB(255, 255, 255))
		}
	})
}

// dots builds the steps along the top: a numbered dot for each, joined by
// a line, its title below.
func dots(c *ui.Context, s *State, steps []Step, cur int, goTo func(int)) {
	t := c.Theme()
	n := len(steps)
	d, pad, gap := t.Space(7), t.Space(1), t.Space(2)
	ui.Row(c).Padding(t.Space(4), t.Space(3)).FocusGroup(ui.Horizontal).Role(ui.RoleGroup).Label("Steps").Children(func() {
		for i, step := range steps {
			st := s.status(i, cur)
			before, after := t.Border, t.Border
			if i <= s.done {
				before = t.Accent
			}
			if i < s.done {
				after = t.Accent
			}
			// The line is the cell's rather than the button's, which fades
			// it while it is disabled.
			cell := ui.Column(c.Key(i)).Grow(1).Basis(0).AlignItems(ui.Center).Draw(func(p *ui.Painter, r ui.Rect) {
				cx, cy := r.X+r.W/2, r.Y+pad+d/2
				if i > 0 {
					p.Line(r.X, cy, cx-d/2-gap, cy, 2, before)
				}
				if i < n-1 {
					p.Line(cx+d/2+gap, cy, r.X+r.W, cy, 2, after)
				}
			})
			cell.Children(func() {
				b := ui.ButtonBase(c).Column().Gap(t.Space(1.5)).Padding(pad, t.Space(1)).Radius(t.Radius).
					Label(step.Title).Disabled(i > s.done)
				s.describe(b, i, st)
				if b.Clicked() && i != cur {
					goTo(i)
				}
				hover := b.Hovered() && i != cur
				b.Children(func() {
					dot := ui.Row(c).Size(d, d).Radius(d / 2).Justify(ui.Center).AlignItems(ui.Center)
					switch st {
					case pending:
						ring := t.Text.Alpha(0.25)
						if hover {
							ring = t.Text.Alpha(0.5)
						}
						dot.Border(1.5, ring)
					case current:
						dot.Border(2, t.Accent)
					case done:
						dot.Background(t.Accent)
					case problem:
						dot.Background(t.Danger)
					}
					if hover && st != pending {
						dot.Shadow(0, 0, 0, 3, t.Text.Alpha(0.12))
					}
					dot.Children(func() {
						switch st {
						case done, problem:
							ui.Box(c).Size(d, d).Draw(func(p *ui.Painter, r ui.Rect) {
								cx, cy := r.X+r.W/2, r.Y+r.H/2
								if st == done {
									check(p, cx, cy, r.W*0.2, t.AccentText)
								} else {
									bang(p, cx, cy, r.W*0.2, ui.RGB(255, 255, 255))
								}
							})
						default:
							num := ui.Text(c, strconv.Itoa(i+1)).FontSize(t.FontSize - 1).FontFeatures("tnum").SingleLine()
							if st == current {
								num.Bold().TextColor(t.Accent)
							} else {
								num.TextColor(t.TextMuted)
							}
						}
					})
					title := ui.Text(c, step.Title).FontSize(t.FontSize - 1).TextAlign(ui.Center)
					if i == cur {
						title.Bold()
					}
				})
			})
		}
	})
}

// check strokes a check mark k wide on either side of cx, cy.
func check(p *ui.Painter, cx, cy, k float32, color ui.Color) {
	p.StrokePolyline([][2]float32{{cx - k, cy + k*0.05}, {cx - k*0.3, cy + k*0.7}, {cx + k, cy - k*0.65}}, 1.75, color)
}

// bang draws an exclamation mark k high on either side of cx, cy.
func bang(p *ui.Painter, cx, cy, k float32, color ui.Color) {
	p.Line(cx, cy-k, cx, cy+k*0.25, 1.75, color)
	p.Fill(ui.Rect{X: cx - 1, Y: cy + k*0.75 - 1, W: 2, H: 2}, color, 1)
}
