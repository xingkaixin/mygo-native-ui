// Package fontpicker chooses a font family, as the font menus of design
// tools and editors do: a button showing the family chosen, set in it,
// which opens a panel of the families text can use (ui.FontFamilies),
// those the app registered first, each named in its own font, with a
// check by the one chosen. Typing finds families by their names, and
// Monospaced keeps those whose characters all take one width, as a
// terminal or a code editor asks. Up and Down move among them, as the
// pointer does, and Enter or a click chooses one, which closes the panel,
// as Escape or a click outside it does.
//
//	if fontpicker.View(c, &app.font).Changed() {
//		app.save()
//	}
package fontpicker

import (
	"slices"
	"strings"

	"github.com/egoist/mygo/ui"
)

// state is what a picker keeps: whether its panel shows, the text typed
// in it, whether it shows monospaced families alone, the family the
// arrows are on and the one last under the pointer, whether the arrows
// moved to a family not yet scrolled to, and the place of its list.
type state struct {
	open    bool
	query   string
	mono    bool
	on      string
	hovered string
	scroll  bool
	list    ui.ListState
}

// View creates the button choosing *family, "" for the system's font,
// which opens the panel of the families. Changed reports a family chosen.
func View(c *ui.Context, family *string) ui.Element {
	t := c.Theme()
	before := *family
	name := *family
	if name == "" {
		name = "System Font"
	}
	button := ui.ButtonBase(c).Gap(t.Space(2)).Padding(t.Space(1.5), t.Space(2), t.Space(1.5), t.Space(3)).
		Radius(t.Radius).Background(t.Surface).Border(1, t.Border).Role(ui.RolePopUpButton).Label("Font").Value(name)
	st := ui.Local(button, "fontpicker.state", func() state { return state{} })
	if button.Clicked() {
		st.open = !st.open
		st.query, st.on, st.scroll = "", *family, true
	}
	if button.Hovered() {
		button.Background(t.SurfaceHover)
	}
	button.Expanded(st.open).Children(func() {
		ui.Text(c, name).Font(*family).SingleLine().Grow(1).Shrink(1)
		chevron(c)
	})
	ui.PopoverBase(c, button, &st.open, func(panel ui.Element) {
		panel.Width(t.Space(80)).Margin(t.Space(1), 0, 0, 0).Radius(t.Space(2.5)).Clip().Background(t.Background).
			Border(1, t.Border).Shadow(0, 8, 24, 0, ui.RGBA(0, 0, 0, 0.16)).Role(ui.RoleDialog).Label("Fonts")
		found := find(ui.FontFamilies(), st.query, st.mono)
		if !slices.Contains(found, st.on) {
			st.on, st.scroll = "", true
			if len(found) > 0 {
				st.on = found[0]
			}
		}
		choose := func(f string) {
			*family, st.open = f, false
		}
		panel.Children(func() {
			var input, active ui.Element
			ui.Row(c).Gap(t.Space(2)).Padding(t.Space(2), t.Space(2), t.Space(2), t.Space(3)).AlignItems(ui.Center).Children(func() {
				ui.Box(c).Size(t.Space(4), t.Space(4)).Draw(func(p *ui.Painter, r ui.Rect) { magnifier(p, r, t.TextMuted) })
				input = ui.TextInputBase(c, &st.query).AutoFocus().Grow(1).Placeholder("Search fonts").
					Label("Search fonts").Role(ui.RoleComboBox).Expanded(true).OnTextKey(func(k ui.TextKey) bool {
					d := 0
					switch {
					case k.Mods != 0:
					case k.Key == ui.KeyDown:
						d = 1
					case k.Key == ui.KeyUp:
						d = -1
					}
					if d == 0 {
						return false
					}
					if i := slices.Index(found, st.on) + d; i >= 0 && i < len(found) {
						st.on, st.scroll = found[i], true
					}
					return true
				})
				// Pressing the toggle leaves the focus in the field, for
				// typing and the arrows to go on there.
				ui.Toggle(c, &st.mono, "Monospaced").KeepFocus().FontSize(t.FontSize - 1)
			})
			if input.Submitted() && st.on != "" {
				choose(st.on)
			}
			ui.Divider(c)
			if st.scroll {
				if i := slices.Index(found, st.on); i >= 0 {
					st.list.ScrollIntoView(i)
				}
				st.scroll = false
			}
			hovered := ""
			st.list.Label = func(i int) string { return found[i] }
			ui.List(c, &st.list, len(found), func(i int) {
				f := found[i]
				on := f == st.on
				row := ui.Row(c).Height(t.Space(8)).Padding(0, t.Space(2)).Gap(t.Space(2)).Radius(t.Space(1.5)).
					AlignItems(ui.Center).Label(f).Selected(on)
				if row.Hovered() {
					hovered = f
				}
				if row.Clicked() {
					choose(f)
				}
				if on {
					active = row
				}
				row.Children(func() {
					ui.Text(c, f).Font(f).FontSize(t.FontSize + 1).SingleLine().Grow(1).Shrink(1)
					color := t.Accent
					if on {
						color = t.AccentText
					}
					ui.Box(c).Size(t.Space(4), t.Space(4)).Draw(func(p *ui.Painter, r ui.Rect) {
						if f == *family {
							check(p, r, color)
						}
					})
				})
			}).Height(t.Space(90)).Padding(t.Space(1)).Label("Font families").Children(func() {
				if len(found) == 0 {
					ui.Text(c, "No fonts match").TextColor(t.TextMuted).Padding(t.Space(2))
				}
			})
			// The pointer moving onto a family puts the arrows there;
			// resting there, it leaves the arrows to move on.
			if hovered != st.hovered && hovered != "" {
				st.on = hovered
			}
			st.hovered = hovered
			if active.Valid() {
				input.ActiveDescendant(active)
			}
		})
	})
	if *family != before {
		button.SetChanged()
	}
	return button
}

// find returns the names of families holding every word of query, case
// aside, and monospaced ones alone with mono, in the order given.
func find(families []ui.FontFamily, query string, mono bool) []string {
	words := strings.Fields(strings.ToLower(query))
	var found []string
	for _, f := range families {
		if mono && !f.Monospaced {
			continue
		}
		name := strings.ToLower(f.Name)
		if !slices.ContainsFunc(words, func(w string) bool { return !strings.Contains(name, w) }) {
			found = append(found, f.Name)
		}
	}
	return found
}

// chevron shows an arrow pointing down.
func chevron(c *ui.Context) {
	t := c.Theme()
	ui.Box(c).Size(t.Space(3), t.Space(3)).Draw(func(p *ui.Painter, r ui.Rect) {
		cx, cy, k := r.X+r.W/2, r.Y+r.H/2, r.W/5
		p.StrokePolyline([][2]float32{{cx - k, cy - k/2}, {cx, cy + k/2}, {cx + k, cy - k/2}}, 1.5, t.TextMuted)
	})
}

func check(p *ui.Painter, r ui.Rect, color ui.Color) {
	cx, cy, k := r.X+r.W/2, r.Y+r.H/2, r.W*0.3
	p.StrokePolyline([][2]float32{{cx - k, cy}, {cx - k/3, cy + k*0.7}, {cx + k, cy - k*0.7}}, 1.75, color)
}

func magnifier(p *ui.Painter, r ui.Rect, color ui.Color) {
	cx, cy, k := r.X+r.W/2, r.Y+r.H/2, r.W*0.4
	var lens ui.Path
	lens.Circle(cx-k*0.2, cy-k*0.2, k*0.6)
	p.StrokePath(&lens, 1.5, color)
	p.StrokePolyline([][2]float32{{cx + k*0.25, cy + k*0.25}, {cx + k*0.85, cy + k*0.85}}, 1.5, color)
}
