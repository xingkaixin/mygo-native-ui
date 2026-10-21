// Package margincomments shows comments on a document in a margin beside
// it, as Google Docs and Word do. The document is paragraphs of text the
// user can select and copy but not edit. Each thread is anchored to a
// range of the text, which shows tinted, and its card stands in the
// margin level with the first line of the range: the author's avatar and
// name, how long ago they wrote, the comment and the replies under it.
// Where cards would overlap, each pushes those after it down.
//
// A click on a tinted range or on a card makes its thread active: the
// range takes a stronger tint, the card lies exactly level with it, the
// cards above move up to make room, and a reply field shows under the
// replies, which Enter posts. Resolve hides a thread. Escape, or a click
// elsewhere in the text, leaves it.
//
// Selecting text shows "Add comment" in the margin by the selection,
// which opens a card for a new comment, its field focused: Enter posts it
// and Shift+Enter breaks the line, as in the reply field, and Escape
// cancels it. The app keeps the threads, and adds to them what Events
// report in the same frame:
//
//	ev := margincomments.View(c, &app.margin, app.doc, app.threads, app.me)
//	if start, end, text, ok := ev.Added(); ok {
//		app.threads = append(app.threads, margincomments.Thread{Start: start, End: end,
//			Comment: margincomments.Comment{Author: app.me, Time: time.Now(), Text: text}})
//	}
//	if i, text, ok := ev.Replied(); ok {
//		app.threads[i].Replies = append(app.threads[i].Replies,
//			margincomments.Comment{Author: app.me, Time: time.Now(), Text: text})
//	}
//	if i, ok := ev.Resolved(); ok {
//		app.threads[i].Resolved = true
//	}
//
// MyGo does not tell where a range of a Text's words shows. The margin
// lays the paragraphs out again with ui.ShapeText, in the same font and
// at the width the document had in the last frame, and asks that layout
// for the boxes of each range. So the document's text has the theme's
// font at one size and the default line height, which ShapeText cannot
// change, and after the window resizes, the margin follows a frame later.
package margincomments

import (
	"cmp"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"

	"github.com/egoist/mygo/ui"
)

// Comment is what someone wrote, and when.
type Comment struct {
	Author string
	Time   time.Time
	Text   string
}

// Thread is a comment on runes Start up to End of the document's text,
// the replies to it in order, and whether it was resolved, which hides
// it.
type Thread struct {
	Start, End int
	Comment
	Replies  []Comment
	Resolved bool
}

// Events are what the user did in the margin in a frame. The zero Events
// report nothing.
type Events struct {
	start, end        int
	added, reply      string
	replied, resolved int
	hasAdded          bool
	hasReply          bool
	hasResolved       bool
}

// Added returns the comment the user posted on runes start up to end of
// the text. Append a Thread of it in this frame: it comes active.
func (e Events) Added() (start, end int, text string, ok bool) {
	return e.start, e.end, e.added, e.hasAdded
}

// Replied returns the thread the user replied to, and the reply. Append
// it to the thread's Replies in this frame.
func (e Events) Replied() (thread int, text string, ok bool) {
	return e.replied, e.reply, e.hasReply
}

// Resolved returns the thread the user resolved. Set its Resolved in this
// frame, and it goes.
func (e Events) Resolved() (thread int, ok bool) { return e.resolved, e.hasResolved }

// State is what a margin keeps: the active thread, the new comment being
// written, the reply being typed, and the document as it laid it out.
// Keep it in the app's state, one for each document.
type State struct {
	// active is the index of the active thread plus 1, so that the zero
	// State has none.
	active int
	// drafting is whether the card of a new comment on runes from up to
	// to shows; draft is its text, and focus gives its field the focus
	// once it shows.
	drafting bool
	from, to int
	draft    string
	focus    bool
	// reply is what is typed in the active card's reply field.
	reply string
	// queued is what Enter posted between frames, which the next frame
	// reports, and press where the pointer went down in the document.
	queued Events
	press  *ui.Point
	doc    layout
}

// layout is the document's text laid out at a width in a font, its
// paragraphs gap apart.
type layout struct {
	text       string
	font       ui.Font
	width, gap float32
	paras      []para
}

// para is a paragraph of the document: its text, which starts at rune
// start of the document's, its top in the document and its height, its
// shape, and the UTF-16 offset of each of its runes, which the shape
// counts in.
type para struct {
	text        string
	start       int
	top, height float32
	shape       *ui.TextLayout
	units       []int
}

// mark is a highlight in a paragraph: of a thread, or of the new comment
// when thread is -1, as many runes long as size, in rects of the
// paragraph.
type mark struct {
	thread, size int
	rects        []ui.Rect
}

// slot is a card in the margin: of a thread, or of the new comment when
// key is -1, whose range starts at rune at, its first line's top at y in
// the document; its height in the last frame, 0 before it showed, where
// it goes, and the card.
type slot struct {
	key, at   int
	y, h, pos float32
	card      ui.Element
}

// motion is how cards move to their places.
var motion = ui.Animation{Spring: ui.SpringOptions{Duration: 320 * time.Millisecond}}

// View creates the document, text, a paragraph for each of its lines,
// and the margin of threads, as wide as the parent lets it, in a row. me
// is the user, who writes new comments and replies. Put it in a
// ui.Scroll for a document longer than the window.
func View(c *ui.Context, s *State, text string, threads []Thread, me string) Events {
	t := c.Theme()
	ev := s.queued
	s.queued = Events{}
	if s.active > len(threads) || s.active > 0 && threads[s.active-1].Resolved {
		s.active = 0
	}
	if ev.hasAdded {
		s.active = len(threads) + 1
	}
	size := t.FontSize + 2
	font := ui.Font{Family: t.Font, Size: size}
	marginW := t.Space(72)

	root := ui.Row(c).AlignItems(ui.Start).Gap(t.Space(10))
	if root.Shortcut(0, ui.KeyEscape) {
		s.escape()
	}
	root.Children(func() {
		doc := ui.Column(c).Grow(1).Basis(0).Gap(size).FontSize(size).Selectable().Label("Document")
		width := doc.LayoutBounds().W
		doc.Draw(func(_ *ui.Painter, r ui.Rect) {
			if r.W != width {
				c.Invalidate()
			}
		})
		// A press goes on to select text: a paragraph asking for its clicks
		// would leave the selection, as a button's label does.
		doc.HandleInput(func(in ui.InputEvent) bool {
			if in.Kind == ui.InputPointerDown && in.Button == 0 && in.Mods&ui.Shift == 0 {
				s.press = &ui.Point{X: in.X, Y: in.Y}
			}
			return false
		})
		s.doc.lay(text, font, width, size)
		paras := s.doc.paras
		marks := make([][]mark, len(paras))
		var tops []slot
		add := func(key, from, to int) {
			if line, ok := s.doc.mark(marks, key, from, to); ok {
				tops = append(tops, slot{key: key, at: from, y: line.Y})
			}
		}
		for i, th := range threads {
			if !th.Resolved {
				add(i, th.Start, th.End)
			}
		}
		if s.drafting {
			add(-1, s.from, s.to)
		}

		soft, strong := t.Warning.Alpha(0.2), t.Warning.Alpha(0.45)
		if t.Dark {
			soft, strong = t.Warning.Alpha(0.24), t.Warning.Alpha(0.5)
		}
		if s.press != nil {
			s.activate(s.doc.at(marks, *s.press))
			s.dropEmpty()
			s.press = nil
		}
		ids := make([]uint64, len(paras))
		doc.Children(func() {
			for i := range paras {
				p := ui.Text(c, paras[i].text)
				ids[i] = p.ID()
				ms := marks[i]
				p.Draw(func(pt *ui.Painter, r ui.Rect) {
					for _, m := range ms {
						color := soft
						if m.thread < 0 || m.thread == s.active-1 {
							color = strong
						}
						for _, b := range m.rects {
							pt.Fill(ui.Rect{X: r.X + b.X, Y: r.Y + b.Y, W: b.W, H: b.H}, color, 2)
						}
					}
				})
			}
		})
		if width <= 0 {
			return // nothing to line the cards up with before the text is laid out
		}

		margin := ui.Box(c).Width(marginW).Shrink(0)
		var bottom float32
		margin.Children(func() {
			now := c.Now()
			slices.SortStableFunc(tops, func(a, b slot) int {
				return cmp.Or(cmp.Compare(a.y, b.y), cmp.Compare(a.at, b.at))
			})
			var placed []slot
			for _, sl := range tops {
				sl.card = ui.Column(c.Key(sl.key))
				sl.h = sl.card.LayoutBounds().H
				if sl.h == 0 {
					// A card shows once it was laid out, to take its room
					// among the others: until then, it is laid out unseen,
					// and then fades in.
					sl.card.Invisible().Absolute().Top(sl.y).Width(marginW)
					sl.card.Animate("shown", 0, 0)
					style(c, sl.card, s.anchors(sl.key))
					s.fill(c, sl.card, sl.key, threads, me, now, false, &ev)
					c.Invalidate()
					continue
				}
				// Clicking a card, or into the new comment's field, makes it
				// the one the others make room for.
				if sl.card.Clicked() || sl.key < 0 && sl.card.FocusWithin() {
					s.activate(sl.key)
					if sl.key >= 0 {
						s.dropEmpty()
					}
				}
				placed = append(placed, sl)
			}
			anchor := slices.IndexFunc(placed, func(sl slot) bool { return s.anchors(sl.key) })
			place(placed, anchor, t.Space(2))
			for i, sl := range placed {
				on := i == anchor
				left := float32(0)
				if on {
					left = -t.Space(3)
				}
				// A resolved thread's card fades out; the new comment's goes at
				// once, as its thread's card takes its place.
				move := ui.ElementTransition{Animation: &motion, Position: true, Colors: true}
				if sl.key >= 0 {
					move.Exit = &ui.Motion{}
				}
				h := sl.h
				sl.card.Absolute().Left(left).Top(sl.pos).Width(marginW).
					Opacity(sl.card.Animate("shown", 1, 150*time.Millisecond)).Transition(move).
					Draw(func(_ *ui.Painter, r ui.Rect) {
						if r.H != h {
							c.Invalidate()
						}
					})
				style(c, sl.card, on)
				s.fill(c, sl.card, sl.key, threads, me, now, true, &ev)
				bottom = max(bottom, sl.pos+h)
			}
			if from, to, line, ok := s.selection(doc.SelectedTexts(), ids); ok && doc.FocusWithin() {
				bottom = max(bottom, s.addButton(c, from, to, line))
			}
			if len(threads) > 0 {
				c.After(time.Minute) // for the times to say how long ago anew
			}
		})
		margin.Height(bottom)
	})
	return ev
}

// lay lays text out at width in font, its paragraphs gap apart, unless it
// is laid out so already.
func (l *layout) lay(text string, font ui.Font, width, gap float32) {
	if l.paras != nil && l.text == text && l.font == font && l.width == width && l.gap == gap {
		return
	}
	*l = layout{text: text, font: font, width: width, gap: gap}
	var top float32
	start := 0
	for _, line := range strings.Split(text, "\n") {
		p := para{text: line, start: start, top: top, shape: ui.ShapeText(line, font, width)}
		p.units = make([]int, 1, len(line)+1)
		for _, r := range line {
			p.units = append(p.units, p.units[len(p.units)-1]+utf16.RuneLen(r))
		}
		_, p.height = p.shape.Size()
		l.paras = append(l.paras, p)
		top += p.height + gap
		start += len(p.units)
	}
}

// mark adds to marks, by paragraph, the highlight of key on runes from up
// to to of the text, and returns the box of its first line in the
// document.
func (l *layout) mark(marks [][]mark, key, from, to int) (ui.Rect, bool) {
	var first ui.Rect
	found := false
	for i, p := range l.paras {
		n := len(p.units) - 1
		a, b := max(from-p.start, 0), min(to-p.start, n)
		if a >= b {
			continue
		}
		rects := p.shape.SelectionRects(ui.TextInputRange{Start: p.units[a], End: p.units[b]})
		if len(rects) == 0 {
			continue
		}
		marks[i] = append(marks[i], mark{thread: key, size: to - from, rects: rects})
		if !found {
			first, found = slices.MinFunc(rects, func(a, b ui.Rect) int { return cmp.Compare(a.Y, b.Y) }), true
			first.Y += p.top
		}
	}
	return first, found
}

// at returns the thread whose highlight among marks, by paragraph, holds
// pt of the document, the shortest where several do, or -1.
func (l *layout) at(marks [][]mark, pt ui.Point) int {
	hit, size := -1, 0
	for i, p := range l.paras {
		if pt.Y < p.top || pt.Y >= p.top+p.height {
			continue
		}
		for _, m := range marks[i] {
			if m.thread < 0 || hit >= 0 && m.size >= size {
				continue
			}
			for _, r := range m.rects {
				if pt.X >= r.X && pt.X < r.X+r.W && pt.Y-p.top >= r.Y && pt.Y-p.top < r.Y+r.H {
					hit, size = m.thread, m.size
					break
				}
			}
		}
	}
	return hit
}

// place puts the cards of slots, in order down the margin, level with
// their ranges where they can be and gap apart: each pushes those after
// it down. The anchor's lies level with its range, and the cards above it
// move up as far as they must to make room.
func place(slots []slot, anchor int, gap float32) {
	for i := range slots {
		slots[i].pos = slots[i].y
		if i > 0 {
			slots[i].pos = max(slots[i].y, slots[i-1].pos+slots[i-1].h+gap)
		}
	}
	if anchor < 0 {
		return
	}
	slots[anchor].pos = slots[anchor].y
	for i := anchor + 1; i < len(slots); i++ {
		slots[i].pos = max(slots[i].y, slots[i-1].pos+slots[i-1].h+gap)
	}
	for i := anchor - 1; i >= 0; i-- {
		slots[i].pos = min(slots[i].pos, slots[i+1].pos-gap-slots[i].h)
	}
}

// activate makes thread active, or none when it is -1.
func (s *State) activate(thread int) {
	if s.active != thread+1 {
		s.active, s.reply = thread+1, ""
	}
}

// dropEmpty drops the new comment if nothing is written in it, as the
// user turns to something else.
func (s *State) dropEmpty() {
	if s.drafting && strings.TrimSpace(s.draft) == "" {
		s.drafting = false
	}
}

// anchors reports whether the card of key, a thread or the new comment
// (-1), lies level with its range, the others making room: the active
// thread's, or with none active, the new comment's.
func (s *State) anchors(key int) bool {
	return key == s.active-1 || key < 0 && s.active == 0
}

// escape leaves the active thread, or else drops the new comment.
func (s *State) escape() {
	switch {
	case s.active > 0:
		s.active, s.reply = 0, ""
	case s.drafting:
		s.drafting = false
	}
}

// selection returns the runes of the text that the document's selection
// holds, parts of the paragraphs whose elements are ids, and the box of
// its first line.
func (s *State) selection(parts []ui.SelectedText, ids []uint64) (from, to int, line ui.Rect, ok bool) {
	from, to = -1, -1
	for _, part := range parts {
		i := slices.Index(ids, part.ID)
		if i < 0 || part.Start == part.End {
			continue
		}
		if from < 0 {
			from = s.doc.paras[i].start + part.Start
		}
		to = s.doc.paras[i].start + part.End
	}
	if from < 0 || from >= to {
		return 0, 0, ui.Rect{}, false
	}
	line, ok = s.doc.mark(make([][]mark, len(s.doc.paras)), 0, from, to)
	return from, to, line, ok
}

// addButton creates "Add comment" level with the first line, line, of a
// selection of runes from up to to, which opens a card for a new comment
// on them, and returns its bottom.
func (s *State) addButton(c *ui.Context, from, to int, line ui.Rect) float32 {
	t := c.Theme()
	h := t.Space(8)
	top := line.Y + (line.H-h)/2
	b := ui.ButtonBase(c).Absolute().Left(0).Top(top).Height(h).Padding(0, t.Space(3), 0, t.Space(2.5)).
		Gap(t.Space(1.5)).Radius(h/2).Background(t.Background).Border(1, t.Border).
		Shadow(0, 2, 8, 0, ui.RGBA(0, 0, 0, 0.12)).KeepFocus().Label("Add comment")
	if b.Hovered() {
		b.Background(t.SurfaceHover)
	}
	if b.Clicked() {
		s.drafting, s.from, s.to, s.draft, s.focus = true, from, to, "", true
		s.active, s.reply = 0, ""
	}
	b.Children(func() {
		ui.Box(c).Size(t.Space(4), t.Space(4)).Draw(func(p *ui.Painter, r ui.Rect) { bubble(p, r, t.Accent) })
		ui.Text(c, "Add comment").FontWeight(500).SingleLine()
	})
	return top + h
}

// style colors a card: raised from the margin while active.
func style(c *ui.Context, card ui.Element, active bool) {
	t := c.Theme()
	card.Padding(t.Space(3)).Gap(t.Space(3)).Radius(t.Space(2.5)).Focusable()
	fill, hover := t.Surface, t.SurfaceHover
	if t.Dark {
		fill, hover = t.Surface.Alpha(0.5), t.Surface
	}
	switch {
	case active && t.Dark:
		card.Background(t.SurfaceHover).Border(1, t.Border).Shadow(0, 8, 24, 0, ui.RGBA(0, 0, 0, 0.5))
	case active:
		card.Background(t.Background).Border(1, t.Border).Shadow(0, 6, 20, 0, ui.RGBA(0, 0, 0, 0.14))
	case card.Hovered():
		card.Background(hover).Border(1, hover)
	default:
		card.Background(fill).Border(1, fill)
	}
}

// fill builds the content of the card of key: a thread, its replies, and
// while it is active a reply field; or, for -1, the new comment, whose
// field takes the focus once the card is shown.
func (s *State) fill(c *ui.Context, card ui.Element, key int, threads []Thread, me string, now time.Time, shown bool, ev *Events) {
	if key < 0 {
		card.Label("New comment")
		card.Children(func() { s.draftCard(c, me, threads, shown, ev) })
		return
	}
	t := c.Theme()
	th := threads[key]
	active := key == s.active-1
	card.Label("Comment by " + th.Author)
	hovered := card.Hovered()
	card.Children(func() {
		header(c, th.Author, ago(now, th.Time), func() {
			b := resolveButton(c)
			if !active && !hovered && !b.FocusWithin() {
				b.Invisible()
			}
			if b.Clicked() {
				ev.resolved, ev.hasResolved = key, true
			}
		})
		body(c, th.Text)
		for i, r := range th.Replies {
			ui.Column(c.Key(i)).Gap(t.Space(2)).Children(func() {
				header(c, r.Author, ago(now, r.Time), nil)
				body(c, r.Text)
			})
		}
		if !active {
			return
		}
		area := ui.TextArea(c, &s.reply).Lines(1, 6).MinHeight(0).Placeholder("Reply").Label("Reply")
		area.OnTextKey(func(k ui.TextKey) bool {
			switch {
			case k.Key == ui.KeyEnter && k.Mods == 0:
				if text := strings.TrimSpace(k.Text()); text != "" {
					s.queued.replied, s.queued.reply, s.queued.hasReply = key, text, true
					s.reply = ""
					k.SetText("")
				}
			case k.Key == ui.KeyEscape:
				s.escape()
			default:
				return false
			}
			return true
		})
	})
}

// draftCard builds the content of the card of a new comment by me: its
// field, focused once the card shows, Cancel and Comment.
func (s *State) draftCard(c *ui.Context, me string, threads []Thread, shown bool, ev *Events) {
	t := c.Theme()
	header(c, me, "", nil)
	area := ui.TextArea(c, &s.draft).Lines(2, 8).MinHeight(0).Placeholder("Comment").Label("Comment")
	if s.focus && shown {
		area.Focus()
		s.focus = false
	}
	area.OnTextKey(func(k ui.TextKey) bool {
		switch {
		case k.Key == ui.KeyEnter && k.Mods == 0:
			if text := strings.TrimSpace(k.Text()); text != "" {
				s.queued.start, s.queued.end, s.queued.added, s.queued.hasAdded = s.from, s.to, text, true
				s.drafting = false
			}
		case k.Key == ui.KeyEscape:
			s.drafting = false
		default:
			return false
		}
		return true
	})
	ui.Row(c).Gap(t.Space(2)).Justify(ui.End).Children(func() {
		if ui.Button(c, "Cancel").Clicked() {
			s.drafting = false
		}
		text := strings.TrimSpace(s.draft)
		if ui.PrimaryButton(c, "Comment").Disabled(text == "").Clicked() {
			ev.start, ev.end, ev.added, ev.hasAdded = s.from, s.to, text, true
			s.drafting = false
			s.active = len(threads) + 1
		}
	})
}

// header shows who wrote a comment, with their avatar, and when, unless
// it is "", and what trailing builds at the end of the row.
func header(c *ui.Context, author, when string, trailing func()) {
	t := c.Theme()
	ui.Row(c).Gap(t.Space(2.5)).AlignItems(ui.Center).Children(func() {
		ui.Avatar(c, author, nil)
		ui.Column(c).Grow(1).Shrink(1).Children(func() {
			ui.Text(c, author).FontWeight(600).SingleLine()
			if when != "" {
				ui.Text(c, when).FontSize(t.FontSize - 1).TextColor(t.TextMuted).SingleLine()
			}
		})
		if trailing != nil {
			trailing()
		}
	})
}

// body shows the text of a comment.
func body(c *ui.Context, text string) {
	ui.Text(c, text).LineHeight(1.35)
}

// resolveButton creates the round button resolving a thread, a check.
func resolveButton(c *ui.Context) ui.Element {
	t := c.Theme()
	d := t.Space(7)
	b := ui.ButtonBase(c).Size(d, d).Radius(d / 2).Label("Resolve").Tooltip("Resolve")
	if b.Hovered() {
		b.Background(t.Accent.Alpha(0.12))
	}
	return b.Children(func() {
		ui.Box(c).Size(t.Space(4), t.Space(4)).Draw(func(p *ui.Painter, r ui.Rect) { check(p, r, t.Accent) })
	})
}

// ago returns how long before now then was, "5 minutes ago", or from a week
// on, its date.
func ago(now, then time.Time) string {
	d := now.Sub(then)
	unit := func(n int, word string) string {
		if n == 1 {
			return "1 " + word + " ago"
		}
		return strconv.Itoa(n) + " " + word + "s ago"
	}
	switch {
	case d < time.Minute:
		return "Just now"
	case d < time.Hour:
		return unit(int(d.Minutes()), "minute")
	case d < 24*time.Hour:
		return unit(int(d.Hours()), "hour")
	case d < 7*24*time.Hour:
		return unit(int(d.Hours()/24), "day")
	}
	return then.Format("Jan 2")
}

// check draws a check mark in r, a square, in color.
func check(p *ui.Painter, r ui.Rect, color ui.Color) {
	u := r.W / 16
	var path ui.Path
	path.MoveTo(r.X+3*u, r.Y+8.5*u)
	path.LineTo(r.X+6.5*u, r.Y+12*u)
	path.LineTo(r.X+13*u, r.Y+4.5*u)
	p.StrokePath(&path, max(1.5, 1.75*u), color)
}

// bubble draws a speech bubble with a plus in it in r, a square, in
// color.
func bubble(p *ui.Painter, r ui.Rect, color ui.Color) {
	u := r.W / 16
	pt := func(x, y float32) (float32, float32) { return r.X + x*u, r.Y + y*u }
	var path ui.Path
	path.MoveTo(pt(2, 2.5))
	path.LineTo(pt(14, 2.5))
	path.LineTo(pt(14, 11.5))
	path.LineTo(pt(6, 11.5))
	path.LineTo(pt(3, 14.5))
	path.LineTo(pt(3, 11.5))
	path.LineTo(pt(2, 11.5))
	path.Close()
	path.MoveTo(pt(8, 4.75))
	path.LineTo(pt(8, 9.25))
	path.MoveTo(pt(5.75, 7))
	path.LineTo(pt(10.25, 7))
	p.StrokePath(&path, max(1.25, 1.5*u), color)
}
