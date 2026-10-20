// Package queue shows the messages the user sent while an agent was still
// at work, as Claude Code and Cursor keep them above the field until the
// reply under way is through: "3 queued", which a click folds, and the
// messages in the order they go, each in a line or two. While the pointer
// is over a message, or the keyboard focus in it, buttons show to edit it
// in place, where Enter saves and Escape cancels, to send it now, and to
// remove it. Dragging a message, or Alt+Up and Alt+Down, moves it.
//
//	if text, ok := queue.View(c, &app.queue).SentNow(); ok {
//		app.stop()
//		app.send(text)
//	}
//	promptinput.View(c, &app.prompt, "Ask the agent", app.busy() && app.prompt.Text == "", nil)
//	if m, ok := app.prompt.Sent(); ok && app.busy() {
//		app.queue = append(app.queue, m.Text)
//	}
//	if !app.busy() && len(app.queue) > 0 {
//		app.send(app.queue[0])
//		app.queue = app.queue[1:]
//	}
//
// The app keeps the messages, adds those sent while a reply comes, and
// sends the first as each reply is through; the queue edits, moves and
// removes them. A promptinput busy only while nothing is typed in it
// sends what is typed, and stops the reply while it is empty.
package queue

import (
	"slices"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
)

// Events are what the user did with a queue in a frame. The zero Events
// report nothing.
type Events struct {
	sent string
	ok   bool
}

// SentNow returns the message the user chose to send now, which has left
// the queue, for the app to stop the reply under way and send it.
func (e Events) SentNow() (string, bool) { return e.sent, e.ok }

// state is what a queue keeps: its list, and whether it is folded.
type state struct {
	list   ui.ListState
	folded bool
}

// edit is what a message keeps as the user edits it: whether it is being
// edited, the text so far, and the text Enter saved between frames, which
// the next frame saves.
type edit struct {
	on    bool
	draft string
	saved *string
}

// key is a message's identity in the list, so that its state follows it
// as it moves: its text, and how many before it have the same.
type key struct {
	text string
	n    int
}

// What the user did to a message.
const (
	none = iota
	save
	sendNow
	remove
	up
	down
)

// View creates the queue of *items, the first going first, or nothing
// while it is empty.
func View(c *ui.Context, items *[]string) Events {
	var ev Events
	if len(*items) == 0 {
		return ev
	}
	t := c.Theme()
	box := ui.Column(c).Padding(t.Space(1)).Radius(t.Space(3)).Background(t.Surface).Border(1, t.Border).
		Role(ui.RoleGroup).Label("Queued messages")
	st := ui.Local(box, "queue.state", func() state { return state{} })
	box.Children(func() {
		label := strconv.Itoa(len(*items)) + " queued"
		// The label starts where the messages' text does, past their grips.
		head := ui.ButtonBase(c).Justify(ui.Start).Gap(t.Space(1.5)).Padding(t.Space(1), t.Space(2), t.Space(1), t.Space(0.5)).
			Radius(t.Space(2)).Expanded(!st.folded).Label(label)
		if head.Clicked() {
			st.folded = !st.folded
		}
		if head.Hovered() {
			head.Background(t.Text.Alpha(0.05))
		}
		head.Children(func() {
			chevron(c, !st.folded)
			ui.Text(c, label).FontSize(t.FontSize - 1).TextColor(t.TextMuted)
		})
		if st.folded {
			return
		}
		st.list.Key = func(i int) any { return keyOf(*items, i) }
		st.list.Reorder = func(rows []int, to int) { move(items, rows[0], to) }
		act, at, text := none, -1, ""
		ui.List(c, &st.list, len(*items), func(i int) {
			if a, s := row(c, (*items)[i]); a != none {
				act, at, text = a, i, s
			}
		}).MaxHeight(t.Space(60))
		switch act {
		case save:
			(*items)[at] = text
		case sendNow:
			ev = Events{sent: (*items)[at], ok: true}
			*items = slices.Delete(*items, at, at+1)
		case remove:
			*items = slices.Delete(*items, at, at+1)
		case up:
			move(items, at, at-1)
		case down:
			move(items, at, at+2)
		}
	})
	return ev
}

// row builds a message of text, and returns what the user did to it, and
// with save, the text saved.
func row(c *ui.Context, text string) (int, string) {
	t := c.Theme()
	act, out := none, ""
	r := ui.Row(c).Gap(t.Space(1.5)).Padding(t.Space(1), t.Space(1), t.Space(1), t.Space(1.5)).
		Radius(t.Space(2)).AlignItems(ui.Center).Focusable().Label(text)
	ed := ui.Local(r, "queue.edit", func() edit { return edit{} })
	if ed.saved != nil {
		act, out = save, *ed.saved
		ed.saved = nil
	}
	if ed.on {
		r.Children(func() { editor(c, ed) })
		return act, out
	}
	switch {
	case r.Shortcut(ui.Alt, ui.KeyUp):
		act = up
	case r.Shortcut(ui.Alt, ui.KeyDown):
		act = down
	}
	show := r.Hovered() || r.FocusWithin()
	if r.Hovered() {
		r.Background(t.Text.Alpha(0.05))
	}
	r.Children(func() {
		grip(c, show)
		ui.Text(c, text).MaxLines(2).Grow(1).Shrink(1)
		bar := ui.Row(c).Gap(t.Space(0.5)).Role(ui.RoleToolbar).Label("Message actions")
		if !show {
			bar.Opacity(0)
		}
		bar.Children(func() {
			if button(c, "Edit", pencil).Clicked() {
				ed.on, ed.draft = true, text
			}
			if button(c, "Send now", arrow).Clicked() {
				act = sendNow
			}
			if button(c, "Remove", cross).Clicked() {
				act = remove
			}
		})
	})
	return act, out
}

// editor shows the field editing a message.
func editor(c *ui.Context, ed *edit) {
	t := c.Theme()
	ui.Column(c).Gap(t.Space(1)).Grow(1).Shrink(1).Children(func() {
		area := ui.TextAreaBase(c, &ed.draft).Lines(1, 6).Label("Edit queued message").AutoFocus().
			Padding(t.Space(1.5), t.Space(2)).Radius(t.Space(1.5)).Background(t.Background).Border(1, t.Accent.Alpha(0.6))
		area.OnTextKey(func(k ui.TextKey) bool {
			switch {
			case k.Key == ui.KeyEscape && k.Mods == 0:
				ed.on = false
			case k.Key == ui.KeyEnter && k.Mods == 0:
				if text := strings.TrimSpace(k.Text()); text != "" {
					ed.saved, ed.on = &text, false
				}
			default:
				return false
			}
			return true
		})
		ui.Text(c, "Enter to save · Esc to cancel").FontSize(t.FontSize - 2).TextColor(t.TextMuted)
	})
}

// keyOf returns the key of message i of items.
func keyOf(items []string, i int) any {
	n := 0
	for _, s := range items[:i] {
		if s == items[i] {
			n++
		}
	}
	return key{items[i], n}
}

// move moves message from of items to before message to, counted before
// the move, len for the end, where there is one.
func move(items *[]string, from, to int) {
	if to < 0 || to > len(*items) {
		return
	}
	s := (*items)[from]
	*items = slices.Delete(*items, from, from+1)
	if to > from {
		to--
	}
	*items = slices.Insert(*items, to, s)
}

// button creates a square button of icon, named label, which its tooltip
// shows.
func button(c *ui.Context, label string, icon func(p *ui.Painter, r ui.Rect, color ui.Color)) ui.Element {
	t := c.Theme()
	b := ui.ButtonBase(c).Size(t.Space(6.5), t.Space(6.5)).Radius(t.Space(1.5)).Label(label).Tooltip(label)
	if b.Hovered() {
		b.Background(t.Text.Alpha(0.08))
	}
	return b.Draw(func(p *ui.Painter, r ui.Rect) { icon(p, r, t.TextMuted) })
}

// chevron shows an arrow pointing down, or right while folded.
func chevron(c *ui.Context, open bool) {
	t := c.Theme()
	ui.Box(c).Size(t.Space(3), t.Space(3)).Draw(func(p *ui.Painter, r ui.Rect) {
		cx, cy, k := r.X+r.W/2, r.Y+r.H/2, r.W/5
		points := [][2]float32{{cx - k/2, cy - k}, {cx + k/2, cy}, {cx - k/2, cy + k}}
		if open {
			points = [][2]float32{{cx - k, cy - k/2}, {cx, cy + k/2}, {cx + k, cy - k/2}}
		}
		p.StrokePolyline(points, 1.5, t.TextMuted)
	})
}

// grip shows the dots a message is dragged by, faint unless show.
func grip(c *ui.Context, show bool) {
	t := c.Theme()
	color := t.TextMuted.Alpha(0.35)
	if show {
		color = t.TextMuted
	}
	ui.Box(c).Size(t.Space(2), t.Space(4)).Draw(func(p *ui.Painter, r ui.Rect) {
		for i := range 6 {
			x := r.X + r.W*(0.25+0.5*float32(i%2))
			y := r.Y + r.H*(0.25+0.25*float32(i/2))
			p.Fill(ui.Rect{X: x - 1, Y: y - 1, W: 2, H: 2}, color, 1)
		}
	})
}

// The icons below draw in the middle of r, k half their size.

func pencil(p *ui.Painter, r ui.Rect, color ui.Color) {
	cx, cy, k := r.X+r.W/2, r.Y+r.H/2, r.W/5
	p.StrokePolyline([][2]float32{
		{cx - k, cy + k}, {cx - k*0.85, cy + k*0.4}, {cx + k*0.45, cy - k*0.9},
		{cx + k*0.9, cy - k*0.45}, {cx - k*0.4, cy + k*0.85}, {cx - k, cy + k},
	}, 1.25, color)
	p.StrokePolyline([][2]float32{{cx + k*0.05, cy - k*0.5}, {cx + k*0.5, cy - k*0.05}}, 1.25, color)
}

func arrow(p *ui.Painter, r ui.Rect, color ui.Color) {
	cx, cy, k := r.X+r.W/2, r.Y+r.H/2, r.W/5
	p.StrokePolyline([][2]float32{{cx, cy + k}, {cx, cy - k}}, 1.5, color)
	p.StrokePolyline([][2]float32{{cx - k*0.8, cy - k*0.2}, {cx, cy - k}, {cx + k*0.8, cy - k*0.2}}, 1.5, color)
}

func cross(p *ui.Painter, r ui.Rect, color ui.Color) {
	cx, cy, k := r.X+r.W/2, r.Y+r.H/2, r.W/6
	p.StrokePolyline([][2]float32{{cx - k, cy - k}, {cx + k, cy + k}}, 1.5, color)
	p.StrokePolyline([][2]float32{{cx + k, cy - k}, {cx - k, cy + k}}, 1.5, color)
}
