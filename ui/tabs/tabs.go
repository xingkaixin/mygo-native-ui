// Package tabs shows the documents open in a window as a bar of tabs, as
// editors and terminals do, built on ui.TabsBase: a click, the arrows, Home
// and End choose a tab, and assistive technology sees a tab list.
//
//	bar := tabs.Bar(c, &app.selected, len(app.docs), func(i int) tabs.Tab {
//		d := app.docs[i]
//		return tabs.Tab{Title: d.name, Modified: d.modified, Key: d.path}
//	})
//	if i, ok := bar.Closed(); ok {
//		app.docs = slices.Delete(app.docs, i, i+1)
//	}
//	if from, to, ok := bar.Moved(); ok {
//		doc := app.docs[from]
//		app.docs = slices.Insert(slices.Delete(app.docs, from, from+1), to, doc)
//	}
//
// The app keeps the list of documents, and the bar the index of the one
// chosen in it: the bar reports a tab the user closed or moved, which the
// app removes from its list or moves in it in the same frame, and has
// already set the index for the list as the app leaves it.
//
// Tabs share the bar's width, each as wide as its title asks or narrower,
// down to a minimum width, past which the bar scrolls sideways; the tab
// chosen scrolls into view as the choice changes.
package tabs

import "github.com/egoist/mygo/ui"

// Tab is what a tab shows: the title of its document, on one line, cut
// with an ellipsis when the tab is narrow, and a close button, in place of
// which a dot shows that the document has changes not saved while the
// pointer is not over the tab.
type Tab struct {
	Title    string
	Modified bool
	// Key identifies the document, as its path: comparable, and unique
	// among the bar's tabs. Without it, a tab is its index. With it, the
	// keyboard focus stays on the document as tabs close and move, and
	// as the app opens documents among them.
	Key any
}

// Events are what the user did with the tabs of a bar in a frame. The
// zero Events report nothing.
type Events struct {
	closed, from, to    int
	hasClosed, hasMoved bool
}

// Closed returns the tab the user closed, with its close button or the
// middle button of the mouse. Remove it from the list in this frame:
// *selected already indexes the list without it, on the same document when
// another tab closed, else on the tab after it, or on the one before it
// when it was the last, which then takes the keyboard focus if the tab
// closed had it.
func (e Events) Closed() (int, bool) { return e.closed, e.hasClosed }

// Moved returns the tab the user dragged to another place in the bar: its
// index, and the one it takes once deleted from there. Move it in the list
// in this frame: *selected already follows the document it was on.
func (e Events) Moved() (from, to int, ok bool) { return e.from, e.to, e.hasMoved }

// middleButton is InputEvent.Button for the middle button of the mouse.
const middleButton = 2

// state is what a bar keeps from frame to frame: the key of the tab last
// scrolled into view, the tab the middle button clicked since the last
// frame, -1 for none, and whether the tab chosen takes the keyboard focus,
// as the tab closed had it.
type state struct {
	shown   any
	middle  int
	refocus bool
}

// moving is a tab being dragged: the state of its bar and its index.
type moving struct {
	bar  *state
	from int
}

// Bar creates a bar of n tabs, of which *selected is the index of the one
// chosen, and tab(i) tells what tab i shows. Give the bar a Context.Key
// when other bars come and go beside it.
//
// A tab closes with its close button, which assistive technology finds as
// a button named "Close" and the title, or with a click of the middle
// button, and moves when dragged to another place in the bar, a line
// showing where it would go. The Events returned report them.
func Bar(c *ui.Context, selected *int, n int, tab func(i int) Tab) Events {
	t := c.Theme()
	h, minW, button := t.Space(9), t.Space(30), t.Space(5)
	rtl := c.Direction() == ui.RightToLeft
	var ev Events

	bar := ui.ScrollHorizontal(c).Height(h).Background(t.Surface)
	st := ui.Local(bar, "tabs.state", func() state { return state{middle: -1} })
	// Scrolling sideways, the bar lays its content out with no width to
	// share: the tabs share the bar's width in the last frame, and the bar
	// builds again when it changes.
	width := bar.LayoutBounds().W
	bar.Draw(func(_ *ui.Painter, r ui.Rect) {
		if r.W != width {
			c.Invalidate()
		}
	})
	if st.middle >= 0 && st.middle < n {
		ev.closed, ev.hasClosed = st.middle, true
	}
	st.middle = -1
	refocus := st.refocus
	st.refocus = false

	drag, over := ui.DragOver[moving](bar)
	over = over && drag.bar == st && drag.from < n
	drop, dropped := ui.Drop[moving](bar)
	if dropped = dropped && drop.bar == st && drop.from < n; dropped {
		drag = drop
	}
	// gap is where the tab dragged would go, before tab gap, and mark that
	// of the line showing it, -1 for none: set once every tab has seen the
	// pointer, mark is read as the tabs draw.
	gap, mark := 0, -1

	bar.Children(func() {
		parts := ui.TabsBase(c, selected, n)
		// The tabs read the pointer's position against where they were in
		// the last frame: scrolled since, the bar builds again.
		left := parts.List.LayoutBounds().X
		parts.List.Width(max(width, float32(n)*minW)).Height(h).AlignItems(ui.Stretch).Draw(func(p *ui.Painter, r ui.Rect) {
			p.Fill(ui.Rect{X: r.X, Y: r.Y + r.H - 1, W: r.W, H: 1}, t.Border, 0)
			if r.X != left {
				c.Invalidate()
			}
		})
		parts.List.Children(func() {
			for i := range n {
				doc := tab(i)
				on := i == *selected
				key := doc.Key
				if key == nil {
					key = i
				}
				w := ui.Row(c.Key(key)).Shrink(1).MinWidth(minW).AlignItems(ui.Stretch).BorderEndWidth(1).BorderColor(t.Border).
					Drag(moving{bar: st, from: i})
				size := w.LayoutBounds()
				// Hovered holds what was under the pointer when it last
				// moved, which a tab sliding under it as others close or
				// move, or as the bar scrolls, was not.
				px, _, in := w.PointerPosition()
				hovered := in && !over
				if on {
					w.Background(t.Background)
					if key != st.shown {
						w.ScrollIntoView()
						st.shown = key
					}
				}
				if w.Dragging() {
					w.Opacity(0.4)
				}
				// Right to left, the tabs after this one are on its left.
				if (over || dropped) && (px >= size.W/2) != rtl {
					gap = i + 1
				}
				w.DrawOver(func(p *ui.Painter, r ui.Rect) {
					if on {
						p.Fill(ui.Rect{X: r.X, Y: r.Y, W: r.W, H: 2}, t.Accent, 0)
					}
					start, end := r.X, r.X+r.W-2
					if rtl {
						start, end = end, start
					}
					switch {
					case mark == i:
						p.Fill(ui.Rect{X: start, Y: r.Y, W: 2, H: r.H}, t.Accent, 0)
					case mark == n && i == n-1:
						p.Fill(ui.Rect{X: end, Y: r.Y, W: 2, H: r.H}, t.Accent, 0)
					}
				})
				w.HandleInput(func(e ui.InputEvent) bool {
					if e.Button != middleButton {
						return false
					}
					if e.Kind == ui.InputPointerUp && e.X >= 0 && e.Y >= 0 && e.X < size.W && e.Y < size.H {
						st.middle = i
					}
					return e.Kind == ui.InputPointerDown || e.Kind == ui.InputPointerUp
				})

				color := t.TextMuted
				if on || hovered {
					color = t.Text
				}
				w.Children(func() {
					head := parts.Tab(i).Grow(1).Shrink(1).PaddingStart(t.Space(3)).PaddingEnd(button + t.Space(3)).
						TextColor(color).FocusRing(false)
					if on && refocus {
						head.Focus()
					}
					if doc.Modified {
						head.Description("Modified")
					}
					if head.FocusVisible() {
						head.DrawOver(func(p *ui.Painter, r ui.Rect) {
							p.FocusRing(ui.Rect{X: r.X + 5, Y: r.Y + 5, W: r.W - 10, H: r.H - 10}, [4]float32{t.Radius, t.Radius, t.Radius, t.Radius})
						})
					}
					head.Children(func() { ui.Text(c, doc.Title).SingleLine() })

					// Not focusable, the close button keeps the arrows of the
					// tab list on the tabs.
					x := ui.Box(c).Attach(ui.AnchorEnd, ui.AnchorEnd).End(t.Space(1.5)).Size(button, button).Radius(t.Space(1)).
						Role(ui.RoleButton).Label("Close " + doc.Title).KeepFocus()
					if x.Clicked() {
						ev.closed, ev.hasClosed = i, true
					}
					_, _, onX := x.PointerPosition()
					switch {
					case hovered && x.Pressed():
						x.Background(t.SurfacePressed)
					case hovered && onX:
						x.Background(t.SurfaceHover)
					}
					cross := hovered || !doc.Modified
					x.Draw(func(p *ui.Painter, r ui.Rect) {
						cx, cy := r.X+r.W/2, r.Y+r.H/2
						if !cross {
							p.Fill(ui.Rect{X: cx - 4, Y: cy - 4, W: 8, H: 8}, color, 4)
							return
						}
						k := r.W / 5
						var path ui.Path
						path.MoveTo(cx-k, cy-k).LineTo(cx+k, cy+k).MoveTo(cx+k, cy-k).LineTo(cx-k, cy+k)
						p.StrokePath(&path, 1.5, color)
					})
				})
				if ev.hasClosed && ev.closed == i && w.FocusWithin() {
					st.refocus = true
				}
			}
		})
	})

	if over || dropped {
		to := gap
		if to > drag.from {
			to--
		}
		switch {
		case to == drag.from:
		case over:
			mark = gap
		default:
			ev.from, ev.to, ev.hasMoved = drag.from, to, true
			*selected = follow(*selected, drag.from, to)
		}
	}
	if i := ev.closed; ev.hasClosed && (i < *selected || i == *selected && i == n-1) {
		*selected = max(*selected-1, 0)
	}
	if ev.hasClosed || ev.hasMoved {
		// Reported as a change, the frame builds again with the list as
		// the app leaves it, and once more after laying it out, for the
		// tabs that slid under the pointer.
		bar.SetChanged().Changed()
		c.Invalidate()
	}
	return ev
}

// follow returns the index of the tab at selected once the tab at from
// moves to to.
func follow(selected, from, to int) int {
	switch {
	case selected == from:
		return to
	case from < selected && selected <= to:
		return selected - 1
	case to <= selected && selected < from:
		return selected + 1
	}
	return selected
}
