// Package promptinput is the field a chat with a model sends messages
// from, as ChatGPT's and Claude's: text that grows as it is typed, files
// attached by dropping or pasting them, a row of the app's controls, as a
// model selector, and a button that sends, or stops the reply coming.
//
//	promptinput.View(c, &app.prompt, "Ask anything", app.streaming, func() {
//		ui.Select(c, &app.model, models)
//	})
//	if m, ok := app.prompt.Sent(); ok {
//		app.send(m.Text, m.Attachments)
//	}
//	if app.prompt.Stopped() {
//		app.cancel()
//	}
package promptinput

import (
	"path/filepath"
	"slices"
	"strings"

	"github.com/egoist/mygo/transfer"
	"github.com/egoist/mygo/ui"
)

// Attachment is a file attached to a message: one dropped, pasted or
// attached by the app, by its path, or an image pasted, its PNG bytes.
type Attachment struct {
	Name string
	Path string
	PNG  []byte
}

// Message is what the field sends.
type Message struct {
	Text        string
	Attachments []Attachment
}

// State is what a field keeps: the text typed and the files attached,
// which the app may change, and what was sent or stopped in the frame.
// Keep it in the app's state, one for each field.
type State struct {
	Text        string
	Attachments []Attachment
	// queued is a message Enter sent between frames, which the next frame
	// reports.
	queued  *Message
	sent    *Message
	stopped bool
}

// Attach attaches the files at paths, as the app's own button for
// attaching files does.
func (s *State) Attach(paths ...string) {
	for _, p := range paths {
		s.Attachments = append(s.Attachments, Attachment{Name: filepath.Base(p), Path: p})
	}
}

// Sent returns the message sent in this frame, by Enter or the button,
// after View. The field is empty again.
func (s *State) Sent() (Message, bool) {
	if s.sent == nil {
		return Message{}, false
	}
	return *s.sent, true
}

// Stopped reports whether the user pressed the button stopping the reply
// in this frame, after View.
func (s *State) Stopped() bool { return s.stopped }

// take empties the field and returns what it held, or nil with nothing in
// it.
func (s *State) take(text string) *Message {
	if strings.TrimSpace(text) == "" && len(s.Attachments) == 0 {
		return nil
	}
	m := &Message{Text: strings.TrimSpace(text), Attachments: s.Attachments}
	s.Text, s.Attachments = "", nil
	return m
}

// View creates the field of s, showing placeholder while empty: the files
// attached, each with a button removing it, the text, from one line up to
// eight, and below it, what tools builds and a button sending the
// message, or while busy, as while a reply comes, stopping it.
//
// Enter sends, unless busy, and Shift+Enter breaks the line. Files dropped
// on the field attach, as do files and images pasted into it.
func View(c *ui.Context, s *State, placeholder string, busy bool, tools func()) ui.Element {
	t := c.Theme()
	s.sent, s.stopped = s.queued, false
	s.queued = nil
	services := c.Services()
	box := ui.Column(c).Gap(t.Space(2)).Padding(t.Space(2.5)).Radius(t.Space(4)).
		Background(t.Background).Border(1, t.Border).Shadow(0, 1, 3, 0, ui.RGB(0, 0, 0).Alpha(0.06))
	if files := box.DroppedFiles(); len(files) > 0 {
		s.Attach(files...)
	}
	if box.FileDragOver() {
		box.Border(1.5, t.Accent)
	}
	box.Children(func() {
		if len(s.Attachments) > 0 {
			ui.Row(c).Gap(t.Space(1.5)).Wrap().Children(func() {
				remove := -1
				for i, a := range s.Attachments {
					if chip(c.Key(i), a) {
						remove = i
					}
				}
				if remove >= 0 {
					s.Attachments = slices.Delete(s.Attachments, remove, remove+1)
				}
			})
		}
		area := ui.TextAreaBase(c, &s.Text).Lines(1, 8).Placeholder(placeholder).Label(placeholder).
			Padding(t.Space(1), t.Space(1)).AutoFocus()
		if area.FocusWithin() {
			box.Border(1, t.Accent.Alpha(0.5))
		}
		area.OnTextKey(func(k ui.TextKey) bool {
			if k.Key != ui.KeyEnter || k.Mods != 0 {
				return false
			}
			if !busy {
				if m := s.take(k.Text()); m != nil {
					s.queued = m
					k.SetText("")
				}
			}
			return true
		})
		area.OnPaste(func(text string) bool {
			data := services.ReadClipboardData(transfer.FileList, transfer.PNG)
			if files, _ := data.Files(); len(files) > 0 {
				s.Attach(files...)
				return true
			}
			if png, err := data.Read(transfer.PNG); err == nil && text == "" {
				s.Attachments = append(s.Attachments, Attachment{Name: "Pasted image.png", PNG: png})
				return true
			}
			return false
		})
		ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
			ui.Row(c).Gap(t.Space(1.5)).AlignItems(ui.Center).Grow(1).Shrink(1).Children(func() {
				if tools != nil {
					tools()
				}
			})
			empty := strings.TrimSpace(s.Text) == "" && len(s.Attachments) == 0
			if sendButton(c, busy, empty) {
				if busy {
					s.stopped = true
				} else {
					s.sent = s.take(s.Text)
				}
			}
		})
	})
	return box
}

// chip shows attachment a, and reports a click on its button removing it.
func chip(c *ui.Context, a Attachment) bool {
	t := c.Theme()
	removed := false
	ui.Row(c).Gap(t.Space(1.5)).Padding(t.Space(1), t.Space(1), t.Space(1), t.Space(2)).AlignItems(ui.Center).
		Radius(t.Space(2)).Background(t.Surface).Border(1, t.Border).Children(func() {
		ui.Box(c).Size(t.Space(3), t.Space(3.5)).Draw(func(p *ui.Painter, r ui.Rect) {
			p.Stroke(r, t.TextMuted, 2, 1.25)
			for i := range 3 {
				y := r.Y + r.H*(0.35+0.18*float32(i))
				p.Fill(ui.Rect{X: r.X + r.W*0.25, Y: y, W: r.W * 0.5, H: 1}, t.TextMuted, 0)
			}
		})
		ui.Text(c, a.Name).FontSize(t.FontSize - 1).SingleLine().MaxWidth(t.Space(45))
		x := ui.ButtonBase(c).Size(t.Space(4.5), t.Space(4.5)).Radius(t.Space(1)).Label("Remove " + a.Name).KeepFocus()
		if x.Hovered() {
			x.Background(t.Text.Alpha(0.08))
		}
		removed = x.Clicked()
		x.Draw(func(p *ui.Painter, r ui.Rect) {
			cx, cy, k := r.X+r.W/2, r.Y+r.H/2, r.W/6
			p.StrokePolyline([][2]float32{{cx - k, cy - k}, {cx + k, cy + k}}, 1.25, t.TextMuted)
			p.StrokePolyline([][2]float32{{cx + k, cy - k}, {cx - k, cy + k}}, 1.25, t.TextMuted)
		})
	})
	return removed
}

// sendButton shows a round button with an arrow up, or a square while
// busy, faint while there is nothing to send, and reports a click doing
// something.
func sendButton(c *ui.Context, busy, empty bool) bool {
	t := c.Theme()
	label := "Send"
	if busy {
		label = "Stop"
	}
	d := t.Space(8)
	b := ui.ButtonBase(c).Size(d, d).Radius(d / 2).Background(t.Text).Label(label).KeepFocus()
	if empty && !busy {
		b.Opacity(0.3)
	}
	b.Draw(func(p *ui.Painter, r ui.Rect) {
		cx, cy, k := r.X+r.W/2, r.Y+r.H/2, r.W/5
		if busy {
			p.Fill(ui.Rect{X: cx - k*0.8, Y: cy - k*0.8, W: k * 1.6, H: k * 1.6}, t.Background, 2)
			return
		}
		p.StrokePolyline([][2]float32{{cx, cy + k*1.2}, {cx, cy - k*1.2}}, 2, t.Background)
		p.StrokePolyline([][2]float32{{cx - k, cy - k*0.2}, {cx, cy - k*1.2}, {cx + k, cy - k*0.2}}, 2, t.Background)
	})
	return b.Clicked() && (busy || !empty)
}
