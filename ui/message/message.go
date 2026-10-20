// Package message shows the turns of a chat with a model, as ChatGPT and
// Claude show them: the user's message in a bubble at the end of the
// row, and the model's reply, each with a row of actions below it, Copy,
// Edit for the message and Retry for the reply, and once the user has
// edited the message or retried the reply, arrows between the versions,
// ‹ 2/3 ›.
//
//	ev := message.User(c, &turn.version, len(turn.versions), v.prompt)
//	if text, ok := ev.Edited(); ok {
//		turn.versions = append(turn.versions, app.send(text))
//		turn.version = len(turn.versions) - 1
//	}
//	ev = message.Reply(c, &v.reply, len(v.replies), r.text, r.writing, func() {
//		markdown.View(c, r.text, nil)
//	})
//	if ev.Retried() {
//		v.replies = append(v.replies, app.retry(v))
//		v.reply = len(v.replies) - 1
//	}
//
// The app keeps the versions and which one shows; the arrows move
// between them, and the app adds a version as the user edits or retries.
package message

import (
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
)

// Events are what the user did to a message in a frame. The zero Events
// report nothing.
type Events struct {
	edited    string
	hasEdited bool
	retried   bool
}

// Edited returns the text the user sent editing the message, for the app
// to add as a version of it.
func (e Events) Edited() (string, bool) { return e.edited, e.hasEdited }

// Retried reports a click on Retry, for the app to ask the model for
// another reply and add it as a version.
func (e Events) Retried() bool { return e.retried }

// edit is what a message keeps as the user edits it: whether it is being
// edited, the text so far, and the text Enter sent between frames, which
// the next frame reports.
type edit struct {
	on     bool
	draft  string
	queued *string
}

// copied is how long Copy shows a check once clicked.
const copied = 2 * time.Second

// User creates the user's message text, in a bubble at the end of the
// row, and below it, while the pointer is over the row or the keyboard
// focus in its actions, the arrows between versions, *version the one of
// versions showing, from 0, and Copy and Edit. Edit turns the bubble into
// a field of the text, which Enter or Send sends, as Edited reports, and
// Escape or Cancel leaves; Shift+Enter breaks the line.
func User(c *ui.Context, version *int, versions int, text string) Events {
	t := c.Theme()
	var ev Events
	turn := ui.Column(c).AlignItems(ui.End).Gap(t.Space(1))
	st := ui.Local(turn, "message.edit", func() edit { return edit{} })
	if st.queued != nil {
		ev.edited, ev.hasEdited = *st.queued, true
		st.queued = nil
	}
	turn.Children(func() {
		if st.on {
			if text, ok := editor(c, st); ok {
				ev.edited, ev.hasEdited = text, true
			}
			return
		}
		ui.Text(c, text).Selectable().Shrink(1).MaxWidth(t.Space(120)).
			Padding(t.Space(2), t.Space(3)).Radius(t.Space(3)).Background(t.Surface)
		bar := actions(c)
		if !turn.Hovered() && !bar.FocusWithin() {
			bar.Opacity(0)
		}
		bar.Children(func() {
			if versions > 1 {
				pager(c, version, versions)
			}
			copyButton(c, text)
			if action(c, "Edit", pencil).Clicked() {
				st.on, st.draft = true, text
			}
		})
	})
	return ev
}

// editor shows the field editing a message, with Cancel and Send, and
// returns the text Send sent.
func editor(c *ui.Context, st *edit) (sent string, ok bool) {
	t := c.Theme()
	ui.Column(c).Gap(t.Space(2)).Padding(t.Space(2.5)).Radius(t.Space(3)).Background(t.Surface).
		Width(t.Space(120)).Shrink(1).Children(func() {
		area := ui.TextAreaBase(c, &st.draft).Lines(1, 8).Label("Edit message").AutoFocus()
		area.OnTextKey(func(k ui.TextKey) bool {
			switch {
			case k.Key == ui.KeyEscape && k.Mods == 0:
				st.on = false
			case k.Key == ui.KeyEnter && k.Mods == 0:
				if text := strings.TrimSpace(k.Text()); text != "" {
					st.queued, st.on = &text, false
				}
			default:
				return false
			}
			return true
		})
		ui.Row(c).Gap(t.Space(2)).Justify(ui.End).Children(func() {
			if ui.Button(c, "Cancel").Clicked() {
				st.on = false
			}
			text := strings.TrimSpace(st.draft)
			if ui.PrimaryButton(c, "Send").Disabled(text == "").Clicked() {
				sent, ok, st.on = text, true, false
			}
		})
	})
	return sent, ok
}

// Reply creates the model's reply, what body builds, as its Markdown, and
// below it the arrows between versions, *version the one of versions
// showing, from 0, and Copy, which copies text, and Retry, as Retried
// reports. While writing, as the reply streams, the actions wait.
func Reply(c *ui.Context, version *int, versions int, text string, writing bool, body func()) Events {
	t := c.Theme()
	var ev Events
	ui.Column(c).Gap(t.Space(1)).Children(func() {
		body()
		if writing {
			return
		}
		// The icons line up with the text above rather than their buttons.
		actions(c).MarginStart(-t.Space(1.5)).Children(func() {
			if versions > 1 {
				pager(c, version, versions)
			}
			copyButton(c, text)
			ev.retried = action(c, "Retry", retry).Clicked()
		})
	})
	return ev
}

// actions creates the row of a message's actions.
func actions(c *ui.Context) ui.Element {
	t := c.Theme()
	return ui.Row(c).Gap(t.Space(0.5)).AlignItems(ui.Center).Role(ui.RoleToolbar).Label("Message actions")
}

// pager shows the arrows between versions, and which shows, "2/3".
func pager(c *ui.Context, version *int, versions int) {
	t := c.Theme()
	if style(c, ui.ButtonBase(c).Disabled(*version <= 0), "Previous version", chevron(-1)).Clicked() {
		*version--
	}
	ui.Text(c, strconv.Itoa(*version+1)+"/"+strconv.Itoa(versions)).FontSize(t.FontSize - 1).
		TextColor(t.TextMuted).FontFeatures("tnum")
	if style(c, ui.ButtonBase(c).Disabled(*version >= versions-1), "Next version", chevron(1)).Clicked() {
		*version++
	}
}

// copyButton shows Copy, which copies text and shows a check for a while.
func copyButton(c *ui.Context, text string) {
	b := ui.ButtonBase(c)
	at := ui.Local(b, "message.copied", func() time.Time { return time.Time{} })
	if b.Clicked() {
		c.WriteClipboard(text)
		*at = c.Now()
	}
	if left := copied - c.Now().Sub(*at); left > 0 {
		c.After(left)
		style(c, b, "Copied", func(p *ui.Painter, r ui.Rect, _ ui.Color) { check(p, r, c.Theme().Success) })
		return
	}
	style(c, b, "Copy", clipboard)
}

// action creates a square button of icon, named label, which its tooltip
// shows.
func action(c *ui.Context, label string, icon func(p *ui.Painter, r ui.Rect, color ui.Color)) ui.Element {
	return style(c, ui.ButtonBase(c), label, icon)
}

// style makes b the square button of action.
func style(c *ui.Context, b ui.Element, label string, icon func(p *ui.Painter, r ui.Rect, color ui.Color)) ui.Element {
	t := c.Theme()
	b.Size(t.Space(7), t.Space(7)).Radius(t.Space(1.5)).Label(label).Tooltip(label)
	if b.Hovered() && !b.IsDisabled() {
		b.Background(t.Text.Alpha(0.06))
	}
	color := t.TextMuted
	if b.IsDisabled() {
		color = color.Alpha(0.35)
	}
	return b.Draw(func(p *ui.Painter, r ui.Rect) {
		icon(p, r, color)
	})
}

// The icons below draw in the middle of r, k half their size.

func clipboard(p *ui.Painter, r ui.Rect, color ui.Color) {
	cx, cy, k := r.X+r.W/2, r.Y+r.H/2, r.W/5
	p.Stroke(ui.Rect{X: cx - k*0.55, Y: cy - k*0.55, W: k * 1.55, H: k * 1.55}, color, 2, 1.25)
	p.StrokePolyline([][2]float32{{cx + k*0.3, cy - k}, {cx - k, cy - k}, {cx - k, cy + k*0.3}}, 1.25, color)
}

func check(p *ui.Painter, r ui.Rect, color ui.Color) {
	cx, cy, k := r.X+r.W/2, r.Y+r.H/2, r.W/5
	p.StrokePolyline([][2]float32{{cx - k, cy}, {cx - k/3, cy + k*0.7}, {cx + k, cy - k*0.7}}, 1.5, color)
}

func pencil(p *ui.Painter, r ui.Rect, color ui.Color) {
	cx, cy, k := r.X+r.W/2, r.Y+r.H/2, r.W/5
	p.StrokePolyline([][2]float32{
		{cx - k, cy + k}, {cx - k*0.85, cy + k*0.4}, {cx + k*0.45, cy - k*0.9},
		{cx + k*0.9, cy - k*0.45}, {cx - k*0.4, cy + k*0.85}, {cx - k, cy + k},
	}, 1.25, color)
	p.StrokePolyline([][2]float32{{cx + k*0.05, cy - k*0.5}, {cx + k*0.5, cy - k*0.05}}, 1.25, color)
}

func retry(p *ui.Painter, r ui.Rect, color ui.Color) {
	cx, cy, k := r.X+r.W/2, r.Y+r.H/2, float64(r.W/5)
	// The arc goes round clockwise from a gap at the top right, where an
	// arrow head ends it.
	from, to := -math.Pi/4+0.5, -math.Pi/4-0.2+2*math.Pi
	var arc [][2]float32
	for i := range 21 {
		a := from + float64(i)/20*(to-from)
		arc = append(arc, [2]float32{cx + float32(k*math.Cos(a)), cy + float32(k*math.Sin(a))})
	}
	p.StrokePolyline(arc, 1.25, color)
	tip := arc[len(arc)-1]
	head := func(turn float64) [2]float32 {
		a := to + math.Pi/2 + math.Pi + turn
		return [2]float32{tip[0] + float32(0.7*k*math.Cos(a)), tip[1] + float32(0.7*k*math.Sin(a))}
	}
	p.StrokePolyline([][2]float32{head(-0.7), tip, head(0.7)}, 1.25, color)
}

// chevron returns an icon of an arrow pointing left, dir -1, or right, 1.
func chevron(dir float32) func(p *ui.Painter, r ui.Rect, color ui.Color) {
	return func(p *ui.Painter, r ui.Rect, color ui.Color) {
		cx, cy, k := r.X+r.W/2, r.Y+r.H/2, r.W/7
		p.StrokePolyline([][2]float32{{cx - dir*k/2, cy - k}, {cx + dir*k/2, cy}, {cx - dir*k/2, cy + k}}, 1.5, color)
	}
}
