// Package frames shows the messages of a WebSocket or a stream of
// server-sent events as they come, as a browser's Messages panel does: a
// row for each, an arrow up for those sent and down for those received,
// what it holds on one line, its length and its time. Frames of the
// connection itself, as its opening and closing, show muted. A search and
// a choice of All, Sent or Received narrow the rows; the list follows the
// end while it shows it. A click on a row shows the message whole under
// the list: JSON indented, other text as it came, and binary data as a
// hex dump.
//
//	win.Update(func() {
//		app.frames = append(app.frames, frames.Frame{Time: time.Now(), Data: msg})
//	})
//	...
//	frames.View(c, &app.log, app.frames).Grow(1)
package frames

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
)

// Frame is a message of a connection: whether the app sent it or received
// it, when, and its data.
type Frame struct {
	Sent bool
	Time time.Time
	// Event is the name of a server-sent event, shown before its data.
	Event string
	Data  []byte
	// Binary shows Data as bytes, not text.
	Binary bool
	// Control is a frame of the connection rather than of data, as its
	// opening, a ping or its closing, which shows muted.
	Control bool
}

// State is what a view keeps: the search, which way the frames shown
// went, the row chosen and how high the list is over the message. Keep it
// in the app's state, one for each view.
type State struct {
	query  string
	dir    int
	chosen int
	split  float32
	shown  []int
	list   ui.ListState
	inited bool
}

// The ways the frames shown went, as the segments name them.
const (
	all = iota
	sent
	received
)

// View creates the view of frames. Give it a size, or grow it in its
// parent.
func View(c *ui.Context, s *State, frames []Frame) ui.Element {
	t := c.Theme()
	if !s.inited {
		s.inited, s.chosen, s.split = true, -1, 60
		s.list.FollowEnd = true
	}
	s.shown = s.shown[:0]
	q := strings.ToLower(s.query)
	for i, f := range frames {
		if s.dir == sent && !f.Sent || s.dir == received && f.Sent {
			continue
		}
		if q != "" && (f.Binary || !strings.Contains(strings.ToLower(f.Event+" "+string(f.Data)), q)) {
			continue
		}
		s.shown = append(s.shown, i)
	}
	row := -1
	for r, i := range s.shown {
		if i == s.chosen {
			row = r
		}
	}
	s.list.Selected = &row
	s.list.Key = func(r int) any { return s.shown[r] }
	view := ui.Column(c)
	view.Children(func() {
		ui.Row(c).Padding(t.Space(2)).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
			ui.SearchField(c, &s.query).Label("Filter messages").Grow(1).Shrink(1).MaxWidth(t.Space(70))
			ui.Segmented(c, &s.dir, "All", "Sent", "Received").Label("Direction")
			ui.Spacer(c)
			ui.Text(c, count(len(s.shown), len(frames))).FontSize(t.FontSize - 1).TextColor(t.TextMuted).FontFeatures("tnum")
		})
		ui.Divider(c)
		list := func() {
			ui.Column(c).Fill().Children(func() {
				header(c)
				ui.Divider(c)
				l := ui.List(c, &s.list, len(s.shown), func(r int) {
					line(c, &frames[s.shown[r]], s.shown[r] == s.chosen)
				}).Grow(1).Label("Messages")
				if l.Changed() {
					s.chosen = -1
					if row >= 0 {
						s.chosen = s.shown[row]
					}
				}
				l.Children(func() {
					if len(s.shown) == 0 {
						ui.Text(c, "No messages").TextColor(t.TextMuted).AlignSelf(ui.Center).MarginY(t.Space(8))
					}
				})
			})
		}
		if s.chosen < 0 || s.chosen >= len(frames) {
			list()
			return
		}
		ui.SplitVerticalPercent(c, &s.split, list, func() {
			detail(c, s, &frames[s.chosen])
		}).Grow(1)
	})
	return view
}

// count says how many frames show of n.
func count(shown, n int) string {
	if shown == n {
		return strconv.Itoa(n) + " messages"
	}
	return strconv.Itoa(shown) + " of " + strconv.Itoa(n) + " messages"
}

// columns are the widths of the length and time columns.
const lengthWidth, timeWidth = 72, 104

// header creates the names of the columns.
func header(c *ui.Context) {
	t := c.Theme()
	name := func(s string) ui.Element {
		return ui.Text(c, s).FontSize(t.FontSize - 1).Bold().TextColor(t.TextMuted)
	}
	ui.Row(c).Background(t.Surface).PaddingY(t.Space(1.5)).PaddingX(t.Space(3)).Gap(t.Space(3)).Children(func() {
		name("Data").Grow(1).MarginStart(t.Space(6))
		name("Length").Width(lengthWidth).TextAlign(ui.End)
		name("Time").Width(timeWidth).TextAlign(ui.End)
	})
}

// line creates the row of f, in the accent's text color when chosen.
func line(c *ui.Context, f *Frame, chosen bool) {
	t := c.Theme()
	small := t.FontSize - 1
	arrow, col := "↓", t.Accent
	if f.Sent {
		arrow, col = "↑", t.Success
	}
	ink, muted := t.Text, t.TextMuted
	if f.Control {
		arrow, col, ink = "•", t.TextMuted, t.TextMuted
	}
	if chosen {
		col, ink, muted = t.AccentText, t.AccentText, t.AccentText.Alpha(0.8)
	}
	r := ui.Row(c).PaddingX(t.Space(3)).PaddingY(t.Space(1)).Gap(t.Space(3)).AlignItems(ui.Center)
	if f.Sent && !f.Control && !chosen {
		r.Background(t.Success.Alpha(0.06))
	}
	r.Children(func() {
		ui.Text(c, arrow).Bold().TextColor(col).Width(t.Space(3)).TextAlign(ui.Center)
		ui.Row(c).Grow(1).Shrink(1).MinWidth(0).Gap(t.Space(1.5)).AlignItems(ui.Center).Children(func() {
			if f.Event != "" {
				ui.Text(c, f.Event).FontSize(small-1).TextColor(muted).Padding(0, t.Space(1.5)).
					Radius(t.Space(1)).Background(ink.Alpha(0.08)).SingleLine()
			}
			text := ui.Text(c, preview(f)).Font("monospace").FontSize(small).TextColor(ink).SingleLine().Shrink(1).MinWidth(0)
			if f.Control {
				text.Italic()
			}
		})
		ui.Text(c, size(len(f.Data))).FontSize(small).TextColor(muted).FontFeatures("tnum").Width(lengthWidth).TextAlign(ui.End)
		ui.Text(c, f.Time.Format("15:04:05.000")).FontSize(small).TextColor(muted).FontFeatures("tnum").Width(timeWidth).TextAlign(ui.End)
	})
}

// preview returns f's data on one line.
func preview(f *Frame) string {
	if f.Binary {
		n := min(len(f.Data), 24)
		s := "Binary: " + hex.EncodeToString(f.Data[:n])
		if n < len(f.Data) {
			s += "…"
		}
		return s
	}
	s := string(f.Data)
	if len(s) > 300 {
		s = s[:300]
		for !utf8.ValidString(s) {
			s = s[:len(s)-1]
		}
	}
	return strings.Join(strings.Fields(s), " ")
}

// size writes n bytes short, as a browser's panels do.
func size(n int) string {
	switch {
	case n < 1000:
		return strconv.Itoa(n) + " B"
	case n < 1000_000:
		return strconv.FormatFloat(float64(n)/1000, 'f', 1, 64) + " kB"
	}
	return strconv.FormatFloat(float64(n)/1000_000, 'f', 1, 64) + " MB"
}

// detail creates the pane of the message f: which way it went, its length
// and time, and its data whole.
func detail(c *ui.Context, s *State, f *Frame) {
	t := c.Theme()
	way := "Received"
	if f.Sent {
		way = "Sent"
	}
	body := string(f.Data)
	switch {
	case f.Binary:
		body = hex.Dump(f.Data)
	case json.Valid(f.Data):
		var out bytes.Buffer
		if json.Indent(&out, f.Data, "", "  ") == nil {
			body = out.String()
		}
	}
	ui.Column(c).Fill().Children(func() {
		ui.Row(c).PaddingX(t.Space(3)).PaddingY(t.Space(1.5)).Gap(t.Space(3)).AlignItems(ui.Center).Background(t.Surface).Children(func() {
			ui.Text(c, way).Bold().FontSize(t.FontSize - 1)
			if f.Event != "" {
				ui.Text(c, "event: "+f.Event).FontSize(t.FontSize - 1).TextColor(t.TextMuted)
			}
			ui.Text(c, size(len(f.Data))+" · "+f.Time.Format("15:04:05.000")).FontSize(t.FontSize - 1).TextColor(t.TextMuted).FontFeatures("tnum")
			ui.Spacer(c)
			if ui.Button(c, "Copy").Clicked() {
				c.WriteClipboard(body)
				c.Toast("Copied")
			}
			if ui.Button(c, "✕").Label("Close").Clicked() {
				s.chosen = -1
			}
		})
		ui.Divider(c)
		ui.ScrollBoth(c).Grow(1).Children(func() {
			ui.Text(c, body).Font("monospace").FontSize(t.FontSize - 1).Selectable().NoWrap().Padding(t.Space(3))
		})
	})
}
