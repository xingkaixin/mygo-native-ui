// Package history shows the past conversations of a chat app in a
// sidebar, as ChatGPT and Claude list them: a New chat button and a field
// to search above the conversations, newest first, under Pinned, Today,
// Yesterday, Previous 7 days, Previous 30 days, then a heading a month,
// "March 2026", in the window's language. Each row shows its title on one
// line, the conversation open lit, a spinner while its reply is still
// coming, and, while the pointer is over the row, a "…" button with
// Rename, Pin or Unpin, and Delete, which a right-click on the row opens
// too. Only the rows in view are built, so thousands of conversations
// cost as much as a few.
//
//	ev := history.View(c, &app.history, app.chats, app.active, app.snippet)
//	if i, ok := ev.Opened(); ok {
//		app.active = app.chats[i].ID
//	}
//	if i, title, ok := ev.Renamed(); ok {
//		app.chats[i].Title = title
//	}
//	if i, pinned, ok := ev.Pinned(); ok {
//		app.chats[i].Pinned = pinned
//	}
//	if i, ok := ev.Deleted(); ok {
//		app.chats = slices.Delete(app.chats, i, i+1)
//	}
//	if ev.New() {
//		app.active = app.newChat()
//	}
//
// The app keeps the conversations, in any order, and the ID of the one
// open; the sidebar reports what the user did, which the app does in the
// same frame, deleting last, as the indices the Events hold are into the
// conversations as given.
package history

import (
	"math"
	"slices"
	"strings"
	"time"
	"unicode"

	"github.com/egoist/mygo/i18n"
	"github.com/egoist/mygo/ui"
)

// Chat is a conversation as its row shows it: its title, when its last
// message came, which places it under a heading, whether the user pinned
// it, which puts it under Pinned, and whether its reply is still coming.
type Chat struct {
	// ID identifies the conversation: not empty, and unique among them.
	ID        string
	Title     string
	Updated   time.Time
	Pinned    bool
	Streaming bool
}

// Events are what the user did with the sidebar in a frame, each with the
// index of the conversation in those given. The zero Events report
// nothing.
type Events struct {
	opened, renamed, pinned, deleted             int
	title                                        string
	pin                                          bool
	hasOpened, hasRenamed, hasPinned, hasDeleted bool
	newChat                                      bool
}

// Opened returns the conversation the user opened, with a click or the
// arrows, for the app to make it the one open.
func (e Events) Opened() (int, bool) { return e.opened, e.hasOpened }

// Renamed returns the conversation the user renamed and its new title,
// trimmed and not empty.
func (e Events) Renamed() (int, string, bool) { return e.renamed, e.title, e.hasRenamed }

// Pinned returns the conversation the user pinned, or unpinned, and
// whether it is now pinned.
func (e Events) Pinned() (int, bool, bool) { return e.pinned, e.pin, e.hasPinned }

// Deleted returns the conversation the user deleted, having said so once
// more in its row. Remove it in this frame; when it was the one open, the
// app opens another, or none.
func (e Events) Deleted() (int, bool) { return e.deleted, e.hasDeleted }

// New reports whether the user asked for a new conversation, which the
// app adds and opens.
func (e Events) New() bool { return e.newChat }

// State is what a sidebar keeps: the text searched for, its list, the
// conversation being renamed and the title typed so far, and the one
// asking whether to delete it. Keep it in the app's state, one for each
// sidebar.
type State struct {
	query, searched string
	list            ui.ListState
	renaming, draft string
	// typing is whether the title's field had the keyboard focus, which,
	// once it loses it, saves.
	typing  bool
	asking  string
	refocus bool
	hits    []hit
	rows    []row
}

// hit is a conversation found for the text searched, and the text of its
// messages the app gave for it.
type hit struct {
	chat    int
	snippet string
}

// row is a row of the list: a heading, or a conversation.
type row struct {
	heading string
	hit
}

// heading is a heading's key in the list, apart from conversations' IDs.
type heading string

// View creates the sidebar of chats, of which the one whose ID is active
// is open, growing to its parent's height. Typing in its field keeps the
// conversations whose titles hold what is typed, case aside, and those
// for which snippet, when not nil, returns some text of their messages,
// which shows below the title; what matches is lit in both.
//
// A click opens a conversation, as Up and Down do while the list has the
// keyboard focus. Rename turns the title into a field, where Enter, or
// leaving the field, saves and Escape cancels. Delete, or
// Cmd+Backspace for the one open, asks in the row, where Delete takes the
// keyboard focus and Escape cancels.
func View(c *ui.Context, st *State, chats []Chat, active string, snippet func(i int, query string) string) Events {
	t := c.Theme()
	var ev Events
	query := strings.TrimSpace(st.query)
	q := []rune(strings.ToLower(query))
	rows, at := st.build(c, chats, active, query, q, snippet)

	ui.Column(c).Grow(1).Shrink(1).MinHeight(0).Gap(t.Space(1)).Padding(t.Space(2)).Background(t.Surface).
		Role(ui.RoleGroup).Label("Chat history").Children(func() {
		top := ui.ButtonBase(c).Justify(ui.Start).Gap(t.Space(2)).Padding(t.Space(2)).Radius(t.Radius).Label("New chat")
		if top.Clicked() {
			ev.newChat = true
		}
		if top.Hovered() {
			top.Background(t.Text.Alpha(0.05))
		}
		top.Children(func() {
			ui.Box(c).Size(t.Space(4), t.Space(4)).Draw(func(p *ui.Painter, r ui.Rect) { plus(p, r, t.Text) })
			ui.Text(c, "New chat")
		})
		ui.SearchField(c, &st.query).Label("Search chats")
		if st.searched != query {
			st.searched = query
			st.list.ScrollTo(0, ui.Start)
		}

		st.list.Key = func(i int) any {
			if rows[i].heading != "" {
				return heading(rows[i].heading)
			}
			return chats[rows[i].chat].ID
		}
		st.list.Header = func(i int) bool { return rows[i].heading != "" }
		st.list.Label = func(i int) string {
			if rows[i].heading != "" {
				return rows[i].heading
			}
			return chats[rows[i].chat].Title
		}
		list := ui.List(c, &st.list, len(rows), func(i int) {
			r := rows[i]
			if r.heading != "" {
				ui.Text(c, r.heading).FontSize(t.FontSize-2).Bold().TextColor(t.TextMuted).SingleLine().
					Padding(t.Space(3), t.Space(2), t.Space(1), t.Space(2)).Background(t.Surface)
				return
			}
			ch := chats[r.chat]
			switch ch.ID {
			case st.renaming:
				st.rename(c, &ev, r.chat, ch)
			case st.asking:
				st.ask(c, &ev, r.chat, ch)
			default:
				st.row(c, &ev, r, ch, ch.ID == active, q)
			}
		}).Grow(1).Focusable().Label("Chats")
		list.Children(func() {
			if len(rows) > 0 {
				return
			}
			empty := "No chats yet"
			if query != "" {
				empty = "No chats match"
			}
			ui.Text(c, empty).TextColor(t.TextMuted).Padding(t.Space(3), t.Space(2))
		})

		// The field of a title being typed keeps its arrows and
		// Cmd+Backspace.
		if !st.typing {
			move := 0
			switch {
			case list.Shortcut(0, ui.KeyDown):
				move = 1
			case list.Shortcut(0, ui.KeyUp):
				move = -1
			case list.Shortcut(ui.Cmd, ui.KeyBackspace) && at >= 0:
				st.asking, st.renaming = active, ""
			}
			if j := next(rows, at, move); move != 0 && j >= 0 {
				ev.opened, ev.hasOpened = rows[j].chat, true
				st.list.ScrollIntoView(j)
			}
		}
		if st.refocus {
			st.refocus = false
			list.Focus()
		}
	})
	// The frame builds again with the conversations as the app leaves
	// them.
	if ev.hasOpened || ev.hasRenamed || ev.hasPinned || ev.hasDeleted || ev.newChat {
		c.Invalidate()
	}
	return ev
}

// build returns the rows for chats and what is searched, under their
// headings, and the row of the conversation open, -1 where none shows.
func (st *State) build(c *ui.Context, chats []Chat, active, query string, q []rune, snippet func(int, string) string) ([]row, int) {
	hits := st.hits[:0]
	for i, ch := range chats {
		if len(q) == 0 || len(matches(ch.Title, q)) > 0 {
			hits = append(hits, hit{chat: i})
		} else if snippet != nil {
			if s := snippet(i, query); s != "" {
				hits = append(hits, hit{i, s})
			}
		}
	}
	slices.SortStableFunc(hits, func(a, b hit) int {
		x, y := &chats[a.chat], &chats[b.chat]
		if x.Pinned != y.Pinned {
			if x.Pinned {
				return -1
			}
			return 1
		}
		return y.Updated.Compare(x.Updated)
	})

	now := c.Now()
	today := midnight(now, now.Location())
	locale := c.Region().FormatLocale(c.Locale())
	rows, at, group := st.rows[:0], -1, -1
	for _, h := range hits {
		ch := &chats[h.chat]
		g, title := 0, "Pinned"
		if !ch.Pinned {
			g, title = section(ch.Updated, today, locale)
		}
		if g != group {
			group = g
			rows = append(rows, row{heading: title})
		}
		if ch.ID == active {
			at = len(rows)
		}
		rows = append(rows, row{hit: h})
	}
	st.hits, st.rows = hits, rows
	return rows, at
}

// section returns the heading of a conversation last updated at when,
// and a number for it growing with age, which tells headings apart:
// Today, Yesterday, Previous 7 days, Previous 30 days, then the month,
// in locale.
func section(when, today time.Time, locale string) (int, string) {
	// Rounded, as a day across a change of daylight saving time is 23 or
	// 25 hours long.
	days := int(math.Round(today.Sub(midnight(when, today.Location())).Hours() / 24))
	switch {
	case days <= 0:
		return 1, "Today"
	case days == 1:
		return 2, "Yesterday"
	case days <= 7:
		return 3, "Previous 7 days"
	case days <= 30:
		return 4, "Previous 30 days"
	}
	w := when.In(today.Location())
	months := (today.Year()-w.Year())*12 + int(today.Month()-w.Month())
	return 5 + months, i18n.FormatDate(locale, w, i18n.DateOptions{Skeleton: "yMMMM"})
}

// midnight returns the start of the day of t in loc.
func midnight(t time.Time, loc *time.Location) time.Time {
	y, m, d := t.In(loc).Date()
	return time.Date(y, m, d, 0, 0, 0, 0, loc)
}

// next returns the row of the conversation after row at going by d, at
// itself where there is none, or the first conversation where at is -1.
func next(rows []row, at, d int) int {
	if at < 0 {
		d = 1
	}
	for j := at + d; j >= 0 && j < len(rows); j += d {
		if rows[j].heading == "" {
			return j
		}
	}
	return at
}

// row builds the row of conversation ch, r in the list, lit when on, with
// what matches q lit in its title and snippet.
func (st *State) row(c *ui.Context, ev *Events, r row, ch Chat, on bool, q []rune) {
	t := c.Theme()
	size := t.Space(6)
	w := ui.Row(c).Gap(t.Space(1.5)).AlignItems(ui.Center).Padding(t.Space(1.75), t.Space(2)).Radius(t.Radius)
	if w.Clicked() && !on {
		ev.opened, ev.hasOpened = r.chat, true
	}
	show := w.Hovered()
	switch {
	case on:
		w.Background(t.Text.Alpha(0.09))
	case show:
		w.Background(t.Text.Alpha(0.05))
	}
	if show {
		w.PaddingEnd(size + t.Space(1.5))
	}
	menu := func(m *ui.Menu) {
		if m.Item("Rename").Chosen() {
			st.renaming, st.draft, st.typing, st.asking = ch.ID, ch.Title, false, ""
		}
		pin := "Pin"
		if ch.Pinned {
			pin = "Unpin"
		}
		if m.Item(pin).Chosen() {
			ev.pinned, ev.pin, ev.hasPinned = r.chat, !ch.Pinned, true
		}
		m.Separator()
		if m.Item("Delete").Chosen() {
			st.asking, st.renaming = ch.ID, ""
		}
	}
	w.ContextMenu(menu)
	accent := t.Accent.Alpha(0.25)
	w.Children(func() {
		ui.Column(c).Grow(1).Shrink(1).Children(func() {
			ui.RichText(c, spans(ch.Title, q, accent)...).SingleLine()
			if r.snippet != "" {
				ui.RichText(c, spans(r.snippet, q, accent)...).SingleLine().FontSize(t.FontSize - 2).TextColor(t.TextMuted)
			}
		})
		if ch.Streaming {
			ui.Spinner(c).Size(t.Space(3.5), t.Space(3.5)).Label("Replying")
		}
		// Built while hidden too, as its menu reports the item chosen in
		// the frame after, when the pointer may have left the row.
		more := ui.ButtonBase(c).Attach(ui.AnchorEnd, ui.AnchorEnd).End(t.Space(1)).Size(size, size).Radius(t.Space(1.5)).
			Label("More for " + ch.Title).Tooltip("More").Menu(menu)
		if more.Hovered() {
			more.Background(t.Text.Alpha(0.08))
		}
		if !show {
			more.Opacity(0)
		}
		more.Draw(func(p *ui.Painter, r ui.Rect) { dots(p, r, t.TextMuted) })
	})
}

// rename builds the row of conversation ch, i of those given, as a field
// editing its title.
func (st *State) rename(c *ui.Context, ev *Events, i int, ch Chat) {
	t := c.Theme()
	in := ui.TextInputBase(c, &st.draft).AutoFocus().Label("Rename "+ch.Title).
		Padding(t.Space(1.5), t.Space(2)).Radius(t.Radius).Background(t.Background).Border(1, t.Accent.Alpha(0.6))
	in.OnTextKey(func(k ui.TextKey) bool {
		if k.Key != ui.KeyEscape || k.Mods != 0 {
			return false
		}
		st.renaming, st.typing, st.refocus = "", false, true
		return true
	})
	focused := in.Focused()
	if in.Submitted() || st.typing && !focused {
		if title := strings.TrimSpace(st.draft); title != "" && title != ch.Title {
			ev.renamed, ev.title, ev.hasRenamed = i, title, true
		}
		st.renaming, st.refocus = "", focused
		focused = false
	}
	st.typing = focused
}

// ask builds the row of conversation ch, i of those given, asking whether
// to delete it.
func (st *State) ask(c *ui.Context, ev *Events, i int, ch Chat) {
	t := c.Theme()
	w := ui.Row(c).Gap(t.Space(1)).AlignItems(ui.Center).Padding(t.Space(1), t.Space(1), t.Space(1), t.Space(2)).
		Radius(t.Radius).Background(t.Danger.Alpha(0.1)).Role(ui.RoleGroup).Label("Delete " + ch.Title + "?")
	if w.Shortcut(0, ui.KeyEscape) {
		st.asking, st.refocus = "", true
	}
	w.Children(func() {
		ui.Text(c, "Delete chat?").SingleLine().Grow(1).Shrink(1)
		cancel := small(c, "Cancel", t.Text)
		if cancel.Hovered() {
			cancel.Background(t.Text.Alpha(0.08))
		}
		if cancel.Clicked() {
			st.asking, st.refocus = "", true
		}
		del := small(c, "Delete", ui.RGB(255, 255, 255)).AutoFocus().Background(t.Danger)
		if del.Hovered() {
			del.Background(t.Danger.Mix(ui.RGB(0, 0, 0), 0.15))
		}
		if del.Clicked() {
			ev.deleted, ev.hasDeleted = i, true
			st.asking, st.refocus = "", true
		}
	})
}

// small creates a button of label, in color, that fits in a row.
func small(c *ui.Context, label string, color ui.Color) ui.Element {
	t := c.Theme()
	b := ui.ButtonBase(c).Padding(t.Space(1), t.Space(2)).Radius(t.Space(1.5)).Label(label)
	b.Children(func() { ui.Text(c, label).FontSize(t.FontSize - 1).TextColor(color) })
	return b
}

// matches returns where q, in lower case, is in s, case aside, as ranges
// of runes that do not overlap.
func matches(s string, q []rune) [][2]int {
	if len(q) == 0 {
		return nil
	}
	runes := []rune(s)
	var out [][2]int
	for i := 0; i+len(q) <= len(runes); i++ {
		if equalFold(runes[i:i+len(q)], q) {
			out = append(out, [2]int{i, i + len(q)})
			i += len(q) - 1
		}
	}
	return out
}

func equalFold(s, q []rune) bool {
	for i, r := range s {
		if unicode.ToLower(r) != q[i] {
			return false
		}
	}
	return true
}

// spans returns s in spans, where q matches on a background of color.
func spans(s string, q []rune, color ui.Color) []ui.Span {
	hits := matches(s, q)
	if len(hits) == 0 {
		return []ui.Span{{Text: s}}
	}
	runes := []rune(s)
	var out []ui.Span
	from := 0
	for _, h := range hits {
		if h[0] > from {
			out = append(out, ui.Span{Text: string(runes[from:h[0]])})
		}
		out = append(out, ui.Span{Text: string(runes[h[0]:h[1]]), Background: color, Radius: 3})
		from = h[1]
	}
	if from < len(runes) {
		out = append(out, ui.Span{Text: string(runes[from:])})
	}
	return out
}

// The icons below draw in the middle of r.

func plus(p *ui.Painter, r ui.Rect, color ui.Color) {
	cx, cy, k := r.X+r.W/2, r.Y+r.H/2, r.W*0.4
	p.StrokePolyline([][2]float32{{cx - k, cy}, {cx + k, cy}}, 1.5, color)
	p.StrokePolyline([][2]float32{{cx, cy - k}, {cx, cy + k}}, 1.5, color)
}

func dots(p *ui.Painter, r ui.Rect, color ui.Color) {
	cx, cy, k := r.X+r.W/2, r.Y+r.H/2, r.W/6
	for _, x := range []float32{cx - k, cx, cx + k} {
		p.Fill(ui.Rect{X: x - 1.25, Y: cy - 1.25, W: 2.5, H: 2.5}, color, 1.25)
	}
}
