// Package storymap lays out a user story map, as product teams plan one
// on a wall: the user's activities along the top, the steps of each
// below them, and under each step its stories as cards, cut across by
// releases, the first release's slice the smallest that works end to end.
// Dragging a card moves it to another step, release or place in its
// stack, a dashed slot showing where it will land.
//
//	storymap.View(c, &app.board, activities, releases, &app.stories)
package storymap

import (
	"fmt"
	"math"

	"github.com/egoist/mygo/ui"
)

// Activity is something the user does, in steps.
type Activity struct {
	Name  string
	Steps []string
}

// Story is a story card: its step, counted across all activities' steps
// from 0, and its release. Cards under a step in a release stack in the
// order they come in.
type Story struct {
	Title   string
	Step    int
	Release int
}

// State is what a map keeps: the card being dragged. Keep it in the
// app's state, one for each map.
type State struct {
	// dragging is the dragged card's index plus one, so that zero is none.
	dragging int
	dx, dy   float32
	dropped  bool
}

// View creates the map of activities, releases and *stories, which
// dragging rearranges.
func View(c *ui.Context, s *State, activities []Activity, releases []string, stories *[]Story) ui.Element {
	t := c.Theme()
	colW, gap := t.Space(44), t.Space(3)
	cardH, headH, laneH := t.Space(18), t.Space(13), t.Space(9)
	steps := 0
	for _, a := range activities {
		steps += len(a.Steps)
	}
	// The card being dragged leaves its stack; which card that is stays
	// fixed while this frame's cards are built, though a drag may start
	// among them.
	dragging := s.dragging
	stack := func(step, rel int) []int {
		var out []int
		for i, st := range *stories {
			if st.Step == step && st.Release == rel && i+1 != dragging {
				out = append(out, i)
			}
		}
		return out
	}
	// Each lane is as tall as its tallest stack and a slot to drop into.
	laneTop := make([]float32, len(releases)+1)
	y := 2*headH + 2*gap
	for r := range releases {
		laneTop[r] = y
		most := 0
		for st := range steps {
			most = max(most, len(stack(st, r)))
		}
		y += laneH + float32(most+1)*(cardH+gap)
	}
	laneTop[len(releases)] = y
	width := float32(steps)*colW + float32(max(0, steps-1))*gap
	xOf := func(step int) float32 { return float32(step) * (colW + gap) }
	slotY := func(rel, k int) float32 { return laneTop[rel] + laneH + float32(k)*(cardH+gap) }
	// target returns where a card whose top left is at x, y would land.
	target := func(x, y float32) (step, rel, k int) {
		cx, cy := x+colW/2, y+cardH/2
		step = max(0, min(steps-1, int(math.Floor(float64(cx/(colW+gap))))))
		rel = len(releases) - 1
		for r := range releases {
			if cy < laneTop[r+1] {
				rel = r
				break
			}
		}
		rel = max(0, rel)
		k = max(0, min(len(stack(step, rel)), int(math.Floor(float64((cy-laneTop[rel]-laneH)/(cardH+gap))))))
		return step, rel, k
	}
	board := ui.Box(c).Size(width, laneTop[len(releases)]).Role(ui.RoleGroup).Label("Story map")
	var from ui.Rect
	board.Children(func() {
		x := float32(0)
		for ai, a := range activities {
			w := float32(len(a.Steps))*colW + float32(max(0, len(a.Steps)-1))*gap
			note(c, ai, a.Name, ui.RGB(250, 196, 120), x, 0, w, headH, 700)
			x += w + gap
		}
		step := 0
		for _, a := range activities {
			for _, name := range a.Steps {
				note(c, fmt.Sprint("step", step), name, ui.RGB(150, 200, 250), xOf(step), headH+gap, colW, headH, 600)
				step++
			}
		}
		for r, name := range releases {
			n := 0
			for _, st := range *stories {
				if st.Release == r {
					n++
				}
			}
			ui.Row(c.Key(fmt.Sprint("lane", r))).Absolute().Left(0).Top(laneTop[r] + gap).Width(width).Gap(t.Space(2)).AlignItems(ui.Baseline).
				Children(func() {
					ui.Text(c, name).FontWeight(700)
					ui.Text(c, fmt.Sprintf("%d stories", n)).FontSize(t.FontSize - 1).TextColor(t.TextMuted)
				})
		}
		var dragged ui.Element
		for st := range steps {
			for r := range releases {
				for k, i := range stack(st, r) {
					e := card(c, i, (*stories)[i].Title, xOf(st), slotY(r, k), colW, cardH)
					s.track(e, i)
				}
			}
		}
		if dragging > 0 {
			i := dragging - 1
			st := (*stories)[i]
			k := 0
			for _, j := range stack(st.Step, st.Release) {
				if j < i {
					k++
				}
			}
			from = ui.Rect{X: xOf(st.Step), Y: slotY(st.Release, k), W: colW, H: cardH}
			dragged = card(c, i, st.Title, from.X, from.Y, colW, cardH)
			s.track(dragged, i)
			dragged.Translate(s.dx, s.dy).Shadow(0, 8, 20, 0, ui.RGBA(0, 0, 0, 0.25)).Cursor(ui.CursorGrabbing)
		}
	})
	if dragging > 0 {
		step, rel, k := target(from.X+s.dx, from.Y+s.dy)
		board.DrawOver(func(p *ui.Painter, r ui.Rect) {
			slot := ui.Rect{X: r.X + xOf(step), Y: r.Y + slotY(rel, k), W: colW, H: cardH}
			p.StrokeDashed(slot, t.Accent, t.Radius, 1.5)
		})
		if s.dropped {
			i := s.dragging - 1
			moved := (*stories)[i]
			moved.Step, moved.Release = step, rel
			rest := append(append([]Story{}, (*stories)[:i]...), (*stories)[i+1:]...)
			// Insert before the k-th card of the target stack, or after its
			// last card, or at the end when the stack is empty.
			at, seen := len(rest), 0
			for j, st := range rest {
				if st.Step == step && st.Release == rel {
					if seen == k {
						at = j
						break
					}
					seen++
					at = j + 1
				}
			}
			*stories = append(rest[:at], append([]Story{moved}, rest[at:]...)...)
			s.dragging, s.dx, s.dy, s.dropped = 0, 0, 0, false
		}
	}
	board.Draw(func(p *ui.Painter, rc ui.Rect) {
		for _, top := range laneTop[1:len(releases)] {
			p.StrokePolyline([][2]float32{{rc.X, rc.Y + top}, {rc.X + rc.W, rc.Y + top}}, 1.5, t.Border)
		}
	})
	return board
}

// track follows the drag of card i.
func (s *State) track(e ui.Element, i int) {
	if dx, dy, ok := e.Dragged(); ok {
		if s.dragging != i+1 {
			s.dragging, s.dx, s.dy = i+1, 0, 0
		}
		s.dx += dx
		s.dy += dy
	}
	if _, _, ended := e.DragEnded(); ended && s.dragging == i+1 {
		s.dropped = true
	}
}

func note(c *ui.Context, key any, text string, col ui.Color, x, y, w, h float32, weight int) {
	t := c.Theme()
	ui.Box(c.Key(key)).Absolute().Left(x).Top(y).Size(w, h).Padding(t.Space(2.5)).Radius(t.Radius).
		Background(col.Alpha(0.85)).Children(func() {
		ui.Text(c, text).FontWeight(weight).FontSize(t.FontSize - 1).TextColor(ui.RGB(30, 30, 36)).MaxLines(2).Ellipsis("…")
	})
}

func card(c *ui.Context, i int, title string, x, y, w, h float32) ui.Element {
	t := c.Theme()
	e := ui.Box(c.Key(fmt.Sprint("card", i))).Absolute().Left(x).Top(y).Size(w, h).Padding(t.Space(2.5)).Radius(t.Radius).
		Background(ui.RGB(255, 236, 150)).Shadow(0, 1, 2, 0, ui.RGBA(0, 0, 0, 0.12)).Cursor(ui.CursorGrab).Role(ui.RoleGroup).Label(title)
	e.Children(func() {
		ui.Text(c, title).FontSize(t.FontSize - 1).TextColor(ui.RGB(40, 36, 20)).MaxLines(3).Ellipsis("…")
	})
	return e
}
