// Package toc shows the table of contents of a document beside it, as
// documentation sites do: its headings, of levels 1 to 4, indented by level
// along a thin guide line, each on one line, cut with an ellipsis when long
// and shown whole in a tooltip. As the document scrolls, the section being
// read lights, that of the last heading above a line near the top of the
// view, or the last heading once the document is at its end, and a bar on
// the guide glides to it. A click on a heading scrolls the document until
// the heading sits near the top, smoothly unless the desktop asks for less
// motion, and lights it at once. With Progress, how much of the document is
// read shows by the title, "38%".
//
// The table reads the headings of a markdown.View, and where the view lays
// them out, through the view's Options.Block, which marks each heading as
// the view builds it. The document scrolls in a container whose
// ui.ScrollState the app keeps:
//
//	app.md.Block = app.toc.Block
//
//	ui.Row(c).Fill().AlignItems(ui.Stretch).Children(func() {
//		ui.Scroll(c).TrackScroll(&app.scroll).Grow(1).Children(func() {
//			markdown.View(c, app.guide, &app.md)
//		})
//		toc.View(c, &app.toc, &app.scroll).Width(240)
//	})
//
// A heading lights, and a click goes to it, from the frame after the view
// first lays it out.
package toc

import (
	"math"
	"strconv"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown/parser"
)

// State is what a table of contents keeps: the headings of its document
// and where they sit, the one lit, and a heading a click went to. MaxLevel,
// which the app sets, is the deepest level listed, 4 when 0, and Progress
// shows how much of the document is read. Keep it in the app's state, one
// for each document.
type State struct {
	MaxLevel int
	Progress bool

	// scroll is the container of the document, as View last had it.
	scroll *ui.ScrollState
	// marks are the headings Block marked in the last build of the
	// document, and headings those View lists; taken tells that View took
	// the marks since.
	marks, headings []heading
	taken           bool
	// seenY is the offset the document was last built at, and laidY the
	// one before, at which the layout Block reads was made.
	seenY, laidY float32
	// active is the heading lit, and shown the one last scrolled into
	// view in the list. While held, active is the heading a click went
	// to, until the document scrolls away from heldY, where it went.
	active, shown int
	held          bool
	heldY         float32
}

// heading is a heading of the document: its level, its text, and the top
// of its box in the content of the document's container, NaN until the
// view has laid it out.
type heading struct {
	level int
	title string
	y     float32
}

// Block is a markdown.Options.Block that marks the headings of the view
// for the table of contents, and builds nothing: it returns false, so that
// the view builds every block as it does without it. Give it to the
// Options of the document's view only, or call it first from a Block of
// the app's own.
func (s *State) Block(c *ui.Context, d *parser.Document, i int) bool {
	if i == first(d) {
		s.marks, s.taken = s.marks[:0], false
		s.laidY = s.seenY
		if s.scroll != nil {
			s.seenY = max(0, min(s.scroll.Y, s.scroll.MaxY))
		}
	}
	b := d.Block(i)
	if b.Kind != parser.Heading || b.Level > s.maxLevel() {
		return false
	}
	h := heading{level: b.Level, title: parser.PlainText(d.Inlines(i)), y: float32(math.NaN())}
	// An empty box first in the block's box sits where the heading's
	// text starts.
	at := ui.Box(c).LayoutBounds()
	if s.scroll != nil {
		if view := s.scroll.Bounds(c); at != (ui.Rect{}) && view != (ui.Rect{}) {
			h.y = at.Y - view.Y + s.laidY
		}
	}
	s.marks = append(s.marks, h)
	return false
}

// first returns the block a view of d builds first, which starts a build:
// the first at the top that is not a footnote's, as footnotes come at the
// end.
func first(d *parser.Document) int {
	for i := 1; i < d.Block(0).Next; i = d.Block(i).Next {
		if d.Block(i).Kind != parser.FootnoteDefinition {
			return i
		}
	}
	return -1
}

// maxLevel returns the deepest level listed.
func (s *State) maxLevel() int {
	if s.MaxLevel <= 0 {
		return 4
	}
	return s.MaxLevel
}

// View creates the table of contents of the document scrolled by the
// container of scroll: "On this page", with the share read by it when
// Progress is set, above the headings, which scroll on their own when
// they do not fit, keeping the one lit in view.
func View(c *ui.Context, s *State, scroll *ui.ScrollState) ui.Element {
	t := c.Theme()
	if !s.taken {
		s.headings, s.marks = s.marks, s.headings
	}
	s.taken, s.scroll = true, scroll
	y := max(0, min(scroll.Y, scroll.MaxY))
	s.light(c, y)
	size := t.FontSize - 1
	root := ui.Column(c).Gap(t.Space(2)).MinHeight(0).Role(ui.RoleGroup).Label("On this page")
	return root.Children(func() {
		ui.Row(c).Gap(t.Space(2)).PaddingStart(t.Space(3)).Children(func() {
			ui.Text(c, "On this page").FontSize(size).FontWeight(600).Grow(1).SingleLine()
			if s.Progress {
				ui.Text(c, read(y, scroll.MaxY)).FontSize(size).TextColor(t.TextMuted).FontFeatures("tnum")
			}
		})
		if len(s.headings) > 0 {
			ui.Scroll(c).Grow(1).MinHeight(0).Children(func() { s.list(c, scroll) })
		}
	})
}

// light finds the heading lit with the document scrolled to y: the one a
// click went to, until the document scrolls away from where it went; else
// the last whose top is above a line near the top of the view, the first
// while none is, and the last at the end of the document.
func (s *State) light(c *ui.Context, y float32) {
	n := len(s.headings)
	s.active = max(0, min(s.active, n-1))
	if s.held && (s.scroll.Scrolling() || math.Abs(float64(y-s.heldY)) < 0.5) {
		return
	}
	s.held = false
	line := y + min(c.Theme().Space(20), s.scroll.Bounds(c).H/3)
	active := 0
	for i, h := range s.headings {
		if h.y > line {
			break
		}
		if !math.IsNaN(float64(h.y)) {
			active = i
		}
	}
	if s.scroll.MaxY > 0 && y >= s.scroll.MaxY-0.5 {
		active = n - 1
	}
	s.active = max(0, active)
}

// list builds the headings along their guide, with the bar on the guide
// by the one lit.
func (s *State) list(c *ui.Context, scroll *ui.ScrollState) {
	t := c.Theme()
	size, rowH := t.FontSize-1, t.Space(7)
	top := s.headings[0].level
	for _, h := range s.headings {
		top = min(top, h.level)
	}
	ui.Column(c).PaddingEnd(t.Space(2)).Children(func() {
		ui.Box(c).Absolute().Start(0).Top(0).Bottom(0).Width(1).Background(t.Border)
		rows := make([]ui.Element, len(s.headings))
		texts := make([]ui.Element, len(s.headings))
		for i, h := range s.headings {
			row := ui.ButtonBase(c).Justify(ui.Start).Height(rowH).PaddingStart(t.Space(3 + 3*float32(h.level-top))).
				PaddingEnd(t.Space(1)).Radius(t.Space(1)).Role(ui.RoleLink).Label(h.title)
			if row.Clicked() && !math.IsNaN(float64(h.y)) {
				s.active, s.held = i, true
				s.heldY = max(0, min(h.y-t.Space(6), scroll.MaxY))
				scroll.ScrollTo(scroll.X, s.heldY, ui.Smooth)
			}
			var text ui.Element
			row.Children(func() { text = ui.Text(c, h.title).FontSize(size).SingleLine().Shrink(1) })
			w, _ := c.MeasureText(0, ui.Span{Text: h.title, Font: t.Font, Size: size})
			if shown := text.LayoutBounds().W; shown > 0 && w > shown+0.5 {
				row.Tooltip(h.title)
			}
			rows[i], texts[i] = row, text
		}
		// Colored once all rows are built: a click lights its row in this
		// frame, after the rows before it were built.
		for i, text := range texts {
			switch {
			case i == s.active:
				text.TextColor(t.Accent)
			case rows[i].Hovered():
				text.TextColor(t.Text)
			default:
				text.TextColor(t.TextMuted)
			}
		}
		if s.active != s.shown {
			rows[s.active].ScrollIntoView()
			s.shown = s.active
		}
		bar := ui.Box(c).Absolute().Start(0).Top(t.Space(1)).Width(2).Height(rowH - t.Space(2)).Radius(1).Background(t.Accent)
		bar.AnimatePose(ui.Pose{Y: float32(s.active) * rowH}, ui.Smooth)
	})
}

// read writes how much of a document scrolled to y of maxY is read, as
// "38%": all of one that does not scroll.
func read(y, maxY float32) string {
	f := float32(1)
	if maxY > 0 {
		f = y / maxY
	}
	return strconv.Itoa(int(math.Round(float64(100*f)))) + "%"
}
