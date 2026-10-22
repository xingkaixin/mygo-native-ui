// Package swipeactions puts buttons behind the rows of a list, as Mail and
// Messages do: swiping a row sideways on a trackpad, or dragging it with
// the mouse, slides its content aside and shows the buttons of that edge,
// in their colors.
//
//	ev := swipeactions.Row(c.Key(m.id), &app.swipe,
//		[]swipeactions.Action{read},
//		[]swipeactions.Action{archive, flag},
//		func() { message(c, m) })
//	if a, ok := ev.Chosen(); ok {
//		switch a {
//		case read:
//			m.unread = !m.unread
//		case archive:
//			archived = i // once the loop is done
//		case flag:
//			m.flagged = !m.flagged
//		}
//	}
//
// Let go past half of its buttons, a row springs open, else back closed;
// a flick goes on the way it went. Swiped all the way, the first button of
// the edge stretches across the row with a snap, and letting go there
// chooses it. One row of a State is open at a time: opening another, a
// press anywhere else, or Escape closes it. A swipe takes the axis it
// starts along, so a scroll that starts upright scrolls the list. While
// the desktop asks for less motion, rows jump open and closed.
//
// The buttons are in the row's context menu too, and assistive technology
// finds them as buttons inside the row, open or not.
package swipeactions

import (
	"math"
	"time"

	"github.com/egoist/mygo/ui"
)

// Action is a button behind a row: its label, the icon above it unless
// nil, and its color, which the label and icon show on in white.
type Action struct {
	Label string
	Icon  *ui.SVG
	Color ui.Color
}

// Events are what the user did with a row in a frame. The zero Events
// report nothing.
type Events struct {
	chosen  Action
	ok      bool
	clicked bool
}

// Chosen returns the action the user chose: with its button, its item in
// the context menu, or by swiping all the way. The row has closed.
func (e Events) Chosen() (Action, bool) { return e.chosen, e.ok }

// Clicked reports a click on the row that neither swiped it nor closed it,
// for the app to choose the row.
func (e Events) Clicked() bool { return e.clicked }

// State is what the rows of a list keep together: the row open, or being
// swiped. Keep it in the app's state, one for each list.
type State struct {
	open uint64
}

// scrollIdle is how long after its last scroll a trackpad lets go of a
// row: scrolling tells no gesture's end, which is when it stops, its
// momentum included.
const scrollIdle = 150 * time.Millisecond

// slop is how far a press moves before it takes an axis.
const slop = 3

// Axes a gesture takes.
const (
	none = iota
	across
	upright
)

// row is what a row keeps from frame to frame, its offsets counted from
// the leading edge: where it rests, 0 closed; where it showed; whether a
// press or a scroll holds it, and where that put it; the axis of the press
// under way and how far it moved before taking one; the axis of the scroll
// under way, when it last came, and the DIPs it moved since the last
// frame; and whether it is past where letting go chooses the first action.
type row struct {
	rest, shown float32
	held        bool
	drag        float32
	press       int
	pdx, pdy    float32
	scroll      int
	scrolled    time.Time
	scrollDX    float32
	full        bool
}

// Row creates a row of a list showing what build builds, which a swipe
// slides aside to show leading's buttons at its leading edge, the first
// at the edge, and trailing's at its trailing edge. Give the row a
// Context.Key, as the item it shows, so that it stays open as rows come
// and go above it. Buttons in what build builds take their own presses,
// which do not swipe the row.
func Row(c *ui.Context, st *State, leading, trailing []Action, build func()) Events {
	t := c.Theme()
	var ev Events
	dir := float32(1)
	if c.Direction() == ui.RightToLeft {
		dir = -1
	}
	box := ui.Column(c).Clip().Focusable()
	rs := ui.Local(box, "swipeactions.row", func() row { return row{} })
	id := box.ID()
	choose := func(a Action) {
		ev.chosen, ev.ok = a, true
		rs.rest = 0
	}

	box.HandleInput(func(e ui.InputEvent) bool {
		if e.Kind != ui.InputScroll {
			return false
		}
		if e.Time.Sub(rs.scrolled) > scrollIdle {
			rs.scroll = upright
			if abs(e.DX) > abs(e.DY) {
				rs.scroll = across
			}
		}
		rs.scrolled = e.Time
		if rs.scroll != across {
			return false
		}
		rs.scrollDX -= e.DX * dir
		return true
	})
	box.ContextMenu(func(m *ui.Menu) {
		for _, a := range leading {
			if m.Item(a.Label).Chosen() {
				choose(a)
			}
		}
		if len(leading) > 0 && len(trailing) > 0 {
			m.Separator()
		}
		for _, a := range trailing {
			if m.Item(a.Label).Chosen() {
				choose(a)
			}
		}
	})

	w := max(box.LayoutBounds().W, 1)
	b := t.Space(18)
	lead, trail := float32(len(leading))*b, float32(len(trailing))*b
	lo, hi := float32(0), float32(0)
	if lead > 0 {
		hi = w
	}
	if trail > 0 {
		lo = -w
	}

	scrolling := rs.scroll == across
	if wait := scrollIdle - c.Now().Sub(rs.scrolled); scrolling && wait > 0 {
		c.After(wait)
	} else {
		rs.scroll, scrolling = none, false
	}
	dx, dy, pressing := box.Dragged()
	if pressing && rs.press == none {
		rs.pdx, rs.pdy = rs.pdx+dx, rs.pdy+dy
		if math.Hypot(float64(rs.pdx), float64(rs.pdy)) >= slop {
			rs.press = upright
			if abs(rs.pdx) > abs(rs.pdy) {
				rs.press = across
			}
		}
		dx = rs.pdx
	}
	swiping := pressing && rs.press == across || scrolling
	if box.Clicked() && rs.press == none {
		if rs.rest != 0 {
			rs.rest = 0
		} else {
			ev.clicked = true
		}
	}
	if !pressing {
		rs.press, rs.pdx, rs.pdy = none, 0, 0
	}

	if swiping && !rs.held {
		rs.held, rs.drag = true, rs.shown
		st.open = id
	}
	var x float32
	switch {
	case rs.held && swiping:
		if pressing {
			rs.drag += dx * dir
		}
		rs.drag += rs.scrollDX
		rs.scrollDX = 0
		x = box.Jump("x", ui.Rubberband(rs.drag, lo, hi, w))
		rs.full = lead > 0 && x >= reach(w, lead, b) || trail > 0 && x <= -reach(w, trail, b)
	case rs.held:
		rs.held = false
		vx, _, _ := box.DragEnded()
		v := vx * dir
		switch p := ui.Project(rs.shown, v); {
		case rs.full && rs.shown > 0:
			choose(leading[0])
		case rs.full:
			choose(trailing[0])
		case lead > 0 && p > lead/2:
			rs.rest = lead
		case trail > 0 && p < -trail/2:
			rs.rest = -trail
		default:
			rs.rest = 0
		}
		rs.full = false
		// Jumping in place forgets how fast the last move went, which a
		// pointer resting before it let go no longer goes.
		box.Jump("x", rs.shown)
		a := ui.Snappy
		a.Spring.Velocity = v
		x = box.AnimateTo("x", rs.rest, a)
	default:
		if rs.rest != 0 && (st.open != id || box.PressedOutside() || c.Shortcut(0, ui.KeyEscape)) {
			rs.rest = 0
		}
		x = box.AnimateTo("x", rs.rest, ui.Snappy)
	}
	rs.shown = x
	if !rs.held && rs.rest == 0 && st.open == id {
		st.open = 0
	}
	stretch := float32(0)
	if rs.full {
		stretch = 1
	}
	f := box.AnimateTo("full", stretch, ui.Snappy)

	box.Children(func() {
		ui.Column(c).Translate(x*dir, 0).Children(build)
		if a, ok := strip(c, leading, max(x, 0), f, b, true); ok {
			choose(a)
		}
		if a, ok := strip(c, trailing, max(-x, 0), f, b, false); ok {
			choose(a)
		}
	})
	return ev
}

// strip shows the buttons of an edge across the width the row's content
// left there, the first at the edge stretching across it as full goes from
// 0 to 1, each button b wide or narrower, and returns the action clicked.
// The buttons are built while the row is closed too, for assistive
// technology to find.
func strip(c *ui.Context, actions []Action, width, full, b float32, leading bool) (Action, bool) {
	t := c.Theme()
	var chosen Action
	ok := false
	n := len(actions)
	if n == 0 {
		return chosen, ok
	}
	s := ui.Row(c).Absolute().Top(0).HeightPercent(100).Width(width)
	if leading {
		s.Start(0)
	} else {
		s.End(0)
	}
	white := ui.RGB(255, 255, 255)
	s.Children(func() {
		for k := range n {
			i := k
			if !leading {
				i = n - 1 - k
			}
			a := actions[i]
			share := (1 - full) / float32(n)
			if i == 0 {
				share += full
			}
			sw := width * share
			bg := a.Color
			btn := ui.Row(c.Key(i)).Width(sw).HeightPercent(100).Clip().Role(ui.RoleButton).Label(a.Label)
			if btn.Pressed() {
				bg = bg.Mix(ui.RGB(0, 0, 0), 0.15)
			}
			btn.Background(bg)
			if btn.Clicked() {
				chosen, ok = a, true
			}
			// The label keeps to the content's side of its button, and is
			// centered in it while the button is narrower than b, so that
			// it moves continuously as the button widens.
			off := min(0, (sw-b)/2)
			if leading {
				btn.Justify(ui.End)
			}
			btn.Children(func() {
				label := ui.Column(c).Width(b).Shrink(0).Justify(ui.Center).AlignItems(ui.Center).Gap(t.Space(1))
				if leading {
					label.End(off)
				} else {
					label.Start(off)
				}
				label.Children(func() {
					if a.Icon != nil {
						ui.Icon(c, a.Icon).FontSize(t.FontSize + 5).TextColor(white)
					}
					ui.Text(c, a.Label).FontSize(t.FontSize - 2).FontWeight(500).TextColor(white).SingleLine()
				})
			})
		}
	})
	return chosen, ok
}

// reach is how far a row w wide swipes to an edge whose buttons are open
// wide, each b wide, before letting go chooses the first.
func reach(w, open, b float32) float32 { return max(w/2, open+b/2) }

func abs(v float32) float32 { return float32(math.Abs(float64(v))) }
