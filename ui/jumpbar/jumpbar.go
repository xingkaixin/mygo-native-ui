// Package jumpbar shows a path in a tree as a bar of its names, as the
// address bar of Windows Explorer: a click on a name goes up to it, and
// the chevron after a name opens a menu of the items under it, the one in
// the path checked, so that a choice goes to a sibling of any level. The
// chevron before the first name opens the items at the top. Left and Right
// move the focus along the bar, and Enter, Space or Down opens the menu of
// a chevron.
//
//	if jumpbar.Bar(c, &app.path, app.children).Changed() {
//		app.open(app.path)
//	}
//
// The bar is as wide as its parent. When the names do not fit, those after
// the first fold into one "…" from the start, whose menu lists them, and
// the last name is cut with an ellipsis if it must be.
package jumpbar

import (
	"slices"

	"github.com/egoist/mygo/ui"
)

// Bar creates a bar of *path, the names of the items from the top of a
// tree to the one shown, where children(p) returns the names of the items
// under the path p, those at the top for an empty p. A click on a name or
// a choice from a menu sets *path and reports Changed on the bar.
func Bar(c *ui.Context, path *[]string, children func(p []string) []string) ui.Element {
	t := c.Theme()
	p := *path
	bar := ui.Row(c).Height(t.Space(7)).AlignSelf(ui.Stretch).AlignItems(ui.Center).ClipX().
		FocusGroup(ui.Horizontal).Label("Path")
	width := bar.LayoutBounds().W
	bar.Draw(func(_ *ui.Painter, r ui.Rect) {
		if r.W != width {
			c.Invalidate()
		}
	})
	set := func(q []string) {
		*path = q
		bar.SetChanged()
	}

	pad, chev := t.Space(1.5), t.Space(5)
	measure := func(s string) float32 {
		w, _ := c.MeasureText(0, ui.Span{Text: s, Size: t.FontSize})
		return w + 2*pad
	}
	need := chev
	for _, name := range p {
		need += measure(name) + chev
	}
	// The names from, up to to, fold into "…".
	from, to := 1, 1
	for width > 0 && need > width && to < len(p)-1 {
		need -= measure(p[to]) + chev
		if to == from {
			need += measure("…") + chev
		}
		to++
	}

	more := len(children(p)) > 0
	bar.Children(func() {
		chevron(c, p, 0, children, set)
		for i := 0; i < len(p); i++ {
			if i == from && to > from {
				fold := segment(c, "…").Label("Folded path")
				fold.Menu(func(m *ui.Menu) {
					for k := from; k < to; k++ {
						if m.Item(p[k]).Chosen() {
							set(slices.Clone(p[:k+1]))
						}
					}
				})
				chevron(c, p, to, children, set)
				i = to - 1
				continue
			}
			name := segment(c, p[i])
			if i == len(p)-1 {
				name.Shrink(1).MinWidth(0)
			}
			if name.Clicked() && i < len(p)-1 {
				set(slices.Clone(p[:i+1]))
			}
			if i < len(p)-1 || more {
				chevron(c, p, i+1, children, set)
			}
		}
	})
	return bar
}

// segment creates a button of the bar showing s.
func segment(c *ui.Context, s string) ui.Element {
	t := c.Theme()
	b := ui.ButtonBase(c).Height(t.Space(6)).PaddingX(t.Space(1.5)).Radius(t.Radius).Label(s)
	switch {
	case b.Pressed():
		b.Background(t.SurfacePressed)
	case b.Hovered():
		b.Background(t.SurfaceHover)
	}
	b.Children(func() {
		ui.Text(c, s).SingleLine().PassThrough()
	})
	return b
}

// chevron creates the button opening the menu of the items under p[:k],
// with p[k] checked.
func chevron(c *ui.Context, p []string, k int, children func(p []string) []string, set func([]string)) {
	t := c.Theme()
	label := "Items at the top"
	if k > 0 {
		label = "Items in " + p[k-1]
	}
	b := ui.ButtonBase(c).Size(t.Space(5), t.Space(6)).Radius(t.Radius).Label(label)
	switch {
	case b.Pressed():
		b.Background(t.SurfacePressed)
	case b.Hovered():
		b.Background(t.SurfaceHover)
	}
	b.Children(func() {
		ui.Text(c, "›").FontSize(t.FontSize + 2).TextColor(t.TextMuted).PassThrough()
	})
	b.Menu(func(m *ui.Menu) {
		for _, name := range children(p[:k]) {
			if m.Item(name).Checked(k < len(p) && name == p[k]).Chosen() {
				set(append(slices.Clone(p[:k]), name))
			}
		}
	})
}
