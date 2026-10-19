// Package settings lays out preferences as macOS System Settings does:
// rows in rounded groups, a label on the left of each and its control on
// the right, with lines between the rows, a title above each group and a
// note below it.
//
//	settings.Group(c, "Dock", func(g *settings.Rows) {
//		g.Row("Size", func() { ui.Slider(c, &app.size, 16, 128).Width(200) })
//		g.Toggle("Automatically hide and show the Dock", &app.hide)
//		if g.Link("Hot Corners", "").Clicked() {
//			app.pane = "corners"
//		}
//		g.Note("Hidden, the Dock shows when the pointer reaches it.")
//	})
package settings

import "github.com/egoist/mygo/ui"

// Rows adds rows to a group. Group passes it to the function that
// builds them.
type Rows struct {
	c    *ui.Context
	n    int
	note string
}

// Group creates a group of the rows build adds, with title above it
// unless "". Groups go in a column, a Space(5) apart, as System Settings
// spaces them.
func Group(c *ui.Context, title string, build func(g *Rows)) ui.Element {
	t := c.Theme()
	g := &Rows{c: c}
	return ui.Column(c).Gap(t.Space(1.5)).Children(func() {
		if title != "" {
			ui.Text(c, title).Bold().Padding(0, t.Space(2.5))
		}
		ui.Column(c).Radius(t.Space(2.5)).Background(t.Text.Alpha(0.035)).Border(1, t.Text.Alpha(0.06)).Clip().
			Role(ui.RoleGroup).Label(title).Children(func() { build(g) })
		if g.note != "" {
			ui.Text(c, g.note).FontSize(t.FontSize-1).TextColor(t.TextMuted).Padding(0, t.Space(2.5))
		}
	})
}

// row creates a row, with a line above it unless it is the group's first.
func (g *Rows) row() ui.Element {
	t := g.c.Theme()
	first := g.n == 0
	g.n++
	pad := t.Space(2.5)
	return ui.Row(g.c.Key(g.n)).Gap(t.Space(3)).Padding(t.Space(2), pad).MinHeight(t.Space(10)).AlignItems(ui.Center).
		Draw(func(p *ui.Painter, r ui.Rect) {
			if !first {
				p.Fill(ui.Rect{X: r.X + pad, Y: r.Y, W: r.W - 2*pad, H: 1}, t.Text.Alpha(0.08), 0)
			}
		})
}

// Row adds a row of label and the controls control builds, on the
// right. With label "", control builds the whole row.
func (g *Rows) Row(label string, control func()) ui.Element {
	c := g.c
	return g.row().Children(func() {
		if label != "" {
			ui.Text(c, label).Grow(1).Shrink(1)
		}
		control()
	})
}

// Toggle adds a row of label and a switch turning on, and returns the
// switch, which reports changes.
func (g *Rows) Toggle(label string, on *bool) ui.Element {
	var sw ui.Element
	g.Row(label, func() { sw = ui.Switch(g.c, on).Label(label) })
	return sw
}

// Link adds a row going somewhere else, as to a pane: label, value muted
// on the right unless "", and a chevron. The row is a button; Clicked
// reports clicks.
func (g *Rows) Link(label, value string) ui.Element {
	c := g.c
	t := c.Theme()
	row := g.row().Role(ui.RoleButton).Label(label).Focusable()
	if row.Pressed() {
		row.Background(t.Text.Alpha(0.06))
	}
	return row.Children(func() {
		ui.Text(c, label).Grow(1).Shrink(1)
		if value != "" {
			ui.Text(c, value).TextColor(t.TextMuted).SingleLine()
		}
		ui.Box(c).Size(t.Space(1.5), t.Space(2.5)).Draw(func(p *ui.Painter, r ui.Rect) {
			p.StrokePolyline([][2]float32{{r.X, r.Y}, {r.X + r.W, r.Y + r.H/2}, {r.X, r.Y + r.H}}, 1.5, t.TextMuted)
		})
	})
}

// Note sets the note below the group, in smaller, muted text.
func (g *Rows) Note(text string) { g.note = text }
