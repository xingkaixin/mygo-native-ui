package chartx

import "github.com/egoist/mygo/ui"

// Legend builds the legend of a chart's series: a row of an item for
// each, which wraps onto more lines when it runs out of room, with a
// swatch of the series' color, colors[i], by its name, names[i]. Each item
// is a toggle button of shown[i]: a click, or Space while it has the
// focus, shows or hides the series, and a hidden series' swatch is hollow
// and its name muted. The items are one stop of Tab, among which the
// arrows move the focus, and assistive technology reads each as a toggle
// button named by its series, pressed while the series shows. Changed
// reports a toggle; name the legend with Label:
//
//	legend := chartx.Legend(c, names, colors, app.shown).Label("Packages")
//	if legend.Changed() {
//		app.fitY()
//	}
func Legend(c *ui.Context, names []string, colors []ui.Color, shown []bool) ui.Element {
	t := c.Theme()
	row := ui.Row(c).Wrap().Gap(t.Space(1)).FocusGroup(ui.Horizontal)
	toggled := false
	row.Children(func() {
		for i, name := range names {
			item := ui.ToggleBase(c.Key(i), &shown[i]).Gap(t.Space(1.5)).
				Padding(t.Space(1), t.Space(2)).Radius(t.Radius)
			if item.Changed() {
				toggled = true
			}
			if item.Hovered() {
				item.Background(t.SurfaceHover)
			}
			item.Children(func() {
				swatch := ui.Box(c).Size(10, 10).Radius(3)
				label := ui.Text(c, name).NoWrap()
				if shown[i] {
					swatch.Background(colors[i])
				} else {
					swatch.Border(1.5, colors[i])
					label.TextColor(t.TextMuted)
				}
			})
		}
	})
	if toggled {
		row.SetChanged()
	}
	return row
}
