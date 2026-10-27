// Package switcher is a switcher as ⌘Tab's between apps and Ctrl+Tab's
// between documents: with a modifier held, Tab goes to the next item and
// Shift+Tab to the one before, and letting go of the modifier chooses
// it. The items show in a row over the window while the modifier stays
// down; a quick press and release goes to the next item without it.
//
//	if i, ok := switcher.Show(c, &app.switcher, ui.Ctrl, app.recent).Chosen(); ok {
//		app.open(i)
//	}
//
// Give the items most recent first, the one open now first of all, so
// that a quick press goes back to the one before it. While the row
// shows, ← and → move too, a click chooses an item, and Escape closes it
// without choosing.
package switcher

import (
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
)

// delay is how long the modifier stays down before the row shows, so
// that a quick switch doesn't flash it.
const delay = 150 * time.Millisecond

// Item is an item to switch to: its title, a detail shown under it while
// it is the one chosen, such as a document's folder, and its icon, its
// title's first letter when nil.
type Item struct {
	Title  string
	Detail string
	Icon   *ui.SVG
}

// State is what a switcher keeps from frame to frame. Keep it in the
// app's state.
type State struct {
	open  bool
	at    int
	since time.Time
}

// Events are what the user did with a switcher in a frame.
type Events struct {
	chosen int
	ok     bool
}

// Chosen returns the item the user chose, as an index into the items
// given; the switcher has closed.
func (e Events) Chosen() (int, bool) { return e.chosen, e.ok }

// Show switches among items while mod is held, as Tab with it pressed
// opens the switcher.
func Show(c *ui.Context, s *State, mod ui.Modifiers, items []Item) Events {
	var ev Events
	n := len(items)
	if n == 0 {
		s.open = false
		return ev
	}
	shown := s.open && c.Now().Sub(s.since) >= delay
	move := func(d int) {
		if !s.open {
			s.open, s.at, s.since = true, 0, c.Now()
		}
		s.at = ((s.at+d)%n + n) % n
		if shown {
			c.Announce(items[s.at].Title)
		}
	}
	switch {
	case c.Shortcut(mod, ui.KeyTab):
		move(1)
	case c.Shortcut(mod|ui.Shift, ui.KeyTab):
		move(-1)
	case !s.open:
		return ev
	case c.Modifiers()&mod == 0:
		s.open = false
		return Events{chosen: min(s.at, n-1), ok: true}
	case c.Shortcut(mod, ui.KeyEscape):
		s.open = false
		return ev
	case c.Shortcut(mod, ui.KeyRight):
		move(1)
	case c.Shortcut(mod, ui.KeyLeft):
		move(-1)
	}
	s.at = min(s.at, n-1)
	if wait := delay - c.Now().Sub(s.since); wait > 0 {
		c.After(wait)
		return ev
	}

	t := c.Theme()
	w, _ := c.Size()
	gap := t.Space(2)
	detailed := false
	for _, it := range items {
		detailed = detailed || it.Detail != ""
	}
	tile := min(t.Space(22), (w-t.Space(16))/float32(n)-gap)
	ui.Overlay(c, func() {
		ui.Column(c).Absolute().Left(0).Top(0).Right(0).Bottom(0).AlignItems(ui.Center).Justify(ui.Center).Children(func() {
			panel := ui.Column(c).AlignItems(ui.Center).Gap(t.Space(3)).Padding(t.Space(4)).Radius(t.Space(5)).
				Background(t.Background.Alpha(0.94)).Border(1, t.Border).Shadow(0, 16, 40, 0, ui.RGB(0, 0, 0).Alpha(0.3)).
				Role(ui.RoleList).Label("Switcher")
			panel.Children(func() {
				ui.Row(c).Gap(gap).Children(func() {
					for i, it := range items {
						on := i == s.at
						cell := ui.Box(c.Key(i)).Size(tile, tile).Radius(t.Space(3.5)).Role(ui.RoleListItem).Label(it.Title).Selected(on)
						switch {
						case on:
							cell.Background(t.Text.Alpha(0.12))
						case cell.Hovered():
							cell.Background(t.Text.Alpha(0.05))
						}
						if cell.Clicked() {
							s.open = false
							ev = Events{chosen: i, ok: true}
						}
						cell.Draw(func(p *ui.Painter, r ui.Rect) {
							drawIcon(p, t, it, r)
						})
					}
				})
				it := items[s.at]
				ui.Column(c).AlignItems(ui.Center).Gap(t.Space(0.5)).MaxWidth(max(t.Space(60), w*0.6)).Children(func() {
					ui.Text(c, it.Title).Bold().SingleLine().Ellipsis("…")
					// The detail keeps its line, empty, so that the panel
					// stays still as the items go by.
					if detailed {
						ui.Text(c, it.Detail).FontSize(t.FontSize - 1).TextColor(t.TextMuted).SingleLine().Ellipsis("…").MinHeight(t.Space(4))
					}
				})
			})
		})
	})
	return ev
}

// drawIcon draws the item's icon in the middle of the cell r: its own, or its
// title's first letter on a tinted square.
func drawIcon(p *ui.Painter, t *ui.Theme, it Item, r ui.Rect) {
	size := r.W * 0.6
	box := ui.Rect{X: r.X + (r.W-size)/2, Y: r.Y + (r.H-size)/2, W: size, H: size}
	if it.Icon != nil {
		p.Icon(it.Icon, box, t.Text)
		return
	}
	first, _ := utf8.DecodeRuneInString(it.Title)
	letter := string(unicode.ToUpper(first))
	p.Fill(box, t.Accent.Alpha(0.16), size*0.22)
	fs := size * 0.5
	lw, lh := p.MeasureText(0, ui.Span{Text: letter, Size: fs, Weight: 600})
	p.RichText(box.X+(size-lw)/2, box.Y+(size-lh)/2, 0, ui.Span{Text: letter, Size: fs, Weight: 600, Color: t.Accent})
}
