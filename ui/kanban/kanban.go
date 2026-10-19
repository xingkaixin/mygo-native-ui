// Package kanban shows cards in lanes, as Trello and Linear's boards do:
// cards drag within a lane to reorder them and to other lanes to move
// them, and the keyboard moves them too.
//
//	board := kanban.View(c, &app.board, app.lanes, func(t *Task) {
//		ui.Text(c, t.Title)
//	}).Grow(1)
//	if board.Changed() {
//		app.save()
//	}
package kanban

import (
	"slices"
	"strconv"

	"github.com/egoist/mygo/ui"
)

// Lane is a column of a board: its title and its cards, from the top.
type Lane[T any] struct {
	Title string
	Cards []T
}

// State is what a board keeps: the card chosen. Keep it in the app's
// state, one for each board. The zero State chooses the first card of the
// first lane.
type State struct {
	lane, card int
	// scroll asks the next frame to scroll the card chosen into view.
	scroll bool
}

// Chosen returns the lane and the index in it of the card chosen; the
// index is past the last card in a lane without cards.
func (s *State) Chosen() (lane, card int) { return s.lane, s.card }

// Choose chooses the card at index card of lane and scrolls it into view.
func (s *State) Choose(lane, card int) {
	s.lane, s.card, s.scroll = lane, card, true
}

// ref is a card dragged: its board, so that boards take only their own
// cards, its lane and its index.
type ref struct {
	board      *State
	lane, card int
}

// View creates a board of lanes side by side, each card built by card
// inside a box of its own. It moves the cards in lanes in place and
// reports the moves as changes. Give it a size, or grow it in its parent.
//
// Dragging a card drops it before or after the card under the pointer,
// or at the end of the lane under it. A click chooses a card; while the
// board has the keyboard focus, the arrows choose the card above, below
// or in the next lane, and Cmd with the arrows moves it there.
func View[T any](c *ui.Context, s *State, lanes []Lane[T], card func(T)) ui.Element {
	t := c.Theme()
	s.lane = max(0, min(len(lanes)-1, s.lane))
	if len(lanes) > 0 {
		s.card = max(0, min(len(lanes[s.lane].Cards), s.card))
	}
	var drop *ref
	to, at := 0, 0
	board := ui.ScrollHorizontal(c).Focusable()
	focused := board.FocusWithin()
	board.Children(func() {
		ui.Row(c).Gap(t.Space(3)).Padding(t.Space(3)).Fill().AlignItems(ui.Stretch).Children(func() {
			for l := range lanes {
				lane := &lanes[l]
				ui.Column(c.Key(l)).Width(t.Space(70)).Radius(t.Space(3)).Background(t.Surface).Children(func() {
					ui.Row(c).Gap(t.Space(2)).Padding(t.Space(3), t.Space(3), t.Space(1)).AlignItems(ui.Center).Children(func() {
						ui.Text(c, lane.Title).Bold().SingleLine().Shrink(1)
						ui.Text(c, strconv.Itoa(len(lane.Cards))).TextColor(t.TextMuted).FontFeatures("tnum")
					})
					cards := ui.Scroll(c).Grow(1).Padding(t.Space(1), t.Space(2))
					if r, ok := ui.Drop[ref](cards); ok && r.board == s {
						drop, to, at = &r, l, len(lane.Cards)
					}
					r, end := ui.DragOver[ref](cards)
					end = end && r.board == s
					cards.Children(func() {
						for i, v := range lane.Cards {
							r, before := slot(c.Key(i), s, l, i, focused, end && i == len(lane.Cards)-1, func() { card(v) })
							if r != nil {
								drop, to, at = r, l, i
								if !before {
									at++
								}
							}
						}
					})
				})
			}
		})
	})
	if drop != nil {
		s.lane, s.card = to, move(lanes, *drop, to, at)
		board.SetChanged().Focus()
	}
	if focused && len(lanes) > 0 && keys(board, s, lanes) {
		board.SetChanged()
	}
	return board
}

// slot builds card i of lane l: a box around it, which drags it and takes
// cards dropped on either half of it, before or after it, and shows where
// with a line in the space around it, below it with end. It returns the
// card dropped on it and whether before.
func slot(c *ui.Context, s *State, l, i int, focused, end bool, build func()) (dropped *ref, before bool) {
	t := c.Theme()
	box := ui.Column(c).Padding(t.Space(1), 0).Drag(ref{board: s, lane: l, card: i})
	_, py, _ := box.PointerPosition()
	before = py < box.Bounds().H/2
	if r, ok := ui.Drop[ref](box); ok && r.board == s {
		dropped = &r
	}
	r, over := ui.DragOver[ref](box)
	over = over && r.board == s
	if box.Clicked() {
		s.lane, s.card = l, i
	}
	chosen := s.lane == l && s.card == i
	if chosen && s.scroll {
		box.ScrollIntoView()
		s.scroll = false
	}
	box.Draw(func(p *ui.Painter, b ui.Rect) {
		switch {
		case over && before:
			p.Fill(ui.Rect{X: b.X, Y: b.Y - 1, W: b.W, H: 2}, t.Accent, 1)
		case over || end:
			p.Fill(ui.Rect{X: b.X, Y: b.Y + b.H - 1, W: b.W, H: 2}, t.Accent, 1)
		}
	})
	box.Children(func() {
		face := ui.Column(c).Gap(t.Space(1.5)).Padding(t.Space(2.5)).Radius(t.Space(2)).
			Background(t.Background).Border(1, t.Border).Shadow(0, 1, 2, 0, ui.RGB(0, 0, 0).Alpha(0.08))
		switch {
		// A ring of shadow thickens the border without moving the content.
		case chosen && focused:
			face.Border(1, t.Accent).Shadow(0, 0, 0, 1, t.Accent)
		case chosen:
			face.Border(1, t.Selection).Shadow(0, 0, 0, 1, t.Selection)
		}
		if box.Dragging() {
			face.Opacity(0.4)
		}
		face.Children(build)
	})
	return dropped, before
}

// move moves the card from refers to to index at of lane to, at counted
// before the move, and returns its index after.
func move[T any](lanes []Lane[T], from ref, to, at int) int {
	v := lanes[from.lane].Cards[from.card]
	lanes[from.lane].Cards = slices.Delete(lanes[from.lane].Cards, from.card, from.card+1)
	if to == from.lane && at > from.card {
		at--
	}
	lanes[to].Cards = slices.Insert(lanes[to].Cards, at, v)
	return at
}

// keys chooses and moves cards with the keys pressed on board, and
// reports whether a card moved.
func keys[T any](board ui.Element, s *State, lanes []Lane[T]) bool {
	l, i := s.lane, s.card
	n := len(lanes[l].Cards)
	last := func(l int) int { return max(0, len(lanes[l].Cards)-1) }
	switch {
	case board.Shortcut(0, ui.KeyUp) && i > 0:
		s.Choose(l, i-1)
	case board.Shortcut(0, ui.KeyDown) && i < n-1:
		s.Choose(l, i+1)
	case board.Shortcut(0, ui.KeyLeft) && l > 0:
		s.Choose(l-1, min(i, last(l-1)))
	case board.Shortcut(0, ui.KeyRight) && l < len(lanes)-1:
		s.Choose(l+1, min(i, last(l+1)))
	}
	if i >= n {
		return false
	}
	from := ref{board: s, lane: l, card: i}
	switch {
	case board.Shortcut(ui.Cmd, ui.KeyUp) && i > 0:
		s.Choose(l, move(lanes, from, l, i-1))
	case board.Shortcut(ui.Cmd, ui.KeyDown) && i < n-1:
		s.Choose(l, move(lanes, from, l, i+2))
	case board.Shortcut(ui.Cmd, ui.KeyLeft) && l > 0:
		s.Choose(l-1, move(lanes, from, l-1, min(i, len(lanes[l-1].Cards))))
	case board.Shortcut(ui.Cmd, ui.KeyRight) && l < len(lanes)-1:
		s.Choose(l+1, move(lanes, from, l+1, min(i, len(lanes[l+1].Cards))))
	default:
		return false
	}
	return true
}
