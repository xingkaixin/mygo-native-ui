// Package tiles builds the tiles of Control Center for the panels of menu
// bar apps: rounded tiles that switch something on and off, with an icon
// in a circle that fills with the accent color while on, a title and a
// status; tiles of a slider, thick, with its icon on its track; and tiles
// of your own. Lay them out in a ui.Grid, wide ones spanning its columns:
//
//	ui.Grid(c).Columns(2).Gap(10).Children(func() {
//		tiles.Toggle(c, &app.wifi, wifiIcon, "Wi-Fi", app.network)
//		tiles.Toggle(c, &app.focus, moonIcon, "Focus", "")
//		tiles.Slider(c, &app.volume, speakerIcon, "Sound").ColumnSpan(2)
//		tiles.Tile(c).ColumnSpan(2).Children(func() { ... })
//	})
//
// Tiles are the theme's Surface color, which a panel over the vibrancy of
// a flyout can make translucent with Background.
package tiles

import "github.com/egoist/mygo/ui"

// Tile creates an empty tile, to build what it holds with Children.
func Tile(c *ui.Context) ui.Element {
	t := c.Theme()
	return ui.Column(c).Padding(t.Space(2.5)).Gap(t.Space(2)).Radius(t.Space(3.5)).Background(t.Surface)
}

// Toggle creates a tile switching *on, as a click on it, or Space while it
// has the focus, does, which Changed reports: icon in a circle, accented
// while on, then title and, below it, status, muted, as the network
// joined. An empty status leaves the title alone. Assistive technology
// sees a toggle button named title.
func Toggle(c *ui.Context, on *bool, icon *ui.SVG, title, status string) ui.Element {
	t := c.Theme()
	tile := ui.ToggleBase(c, on).Padding(t.Space(2.5)).Gap(t.Space(2.5)).AlignItems(ui.Center).Justify(ui.Start).
		Radius(t.Space(3.5)).Background(t.Surface).Label(title)
	if tile.Hovered() {
		tile.Background(t.SurfaceHover)
	}
	tile.Children(func() {
		circle(c, icon, *on)
		ui.Column(c).Shrink(1).Children(func() {
			ui.Text(c, title).Bold().FontSize(t.FontSize - 1).SingleLine()
			if status != "" {
				ui.Text(c, status).FontSize(t.FontSize - 2).TextColor(t.TextMuted).SingleLine()
			}
		})
	})
	return tile
}

// circle builds icon in a circle, the accent while on.
func circle(c *ui.Context, icon *ui.SVG, on bool) {
	t := c.Theme()
	bg, fg := t.Border, t.Text
	if on {
		bg, fg = t.Accent, t.AccentText
	}
	d := t.Space(7)
	ui.Box(c).Size(d, d).Radius(d / 2).Background(bg).Center().Children(func() {
		ui.Icon(c, icon).Size(d/2, d/2).TextColor(fg)
	})
}

// Slider creates a tile of a slider of *value from 0 to 1, titled title,
// as Control Center's of the sound and the display: a thick track, white
// up to the value, where its knob is, and icon at its start. Dragging it
// or clicking on it sets the value, as do the arrows while it has the
// focus, which Changed reports. Assistive technology sees a slider named
// title.
func Slider(c *ui.Context, value *float64, icon *ui.SVG, title string) ui.Element {
	t := c.Theme()
	h := t.Space(6)
	tile := Tile(c)
	changed := false
	tile.Children(func() {
		ui.Text(c, title).Bold().FontSize(t.FontSize - 1).SingleLine()
		s := ui.SliderBase(c, value, 0, 1).Height(h).PaddingX(h / 2).Label(title)
		changed = s.Changed()
		f := float32(max(0, min(1, *value)))
		s.Draw(func(p *ui.Painter, r ui.Rect) {
			p.Fill(r, t.Text.Alpha(0.12), h/2)
			// The knob is the round end of the fill, which reaches no
			// shorter than it.
			end := r.X + h + (r.W-h)*f
			fill := ui.Rect{X: r.X, Y: r.Y, W: end - r.X, H: h}
			p.Shadow(ui.Rect{X: end - h, Y: r.Y, W: h, H: h}, h/2, 0, 1, 3, 0, ui.RGBA(0, 0, 0, 0.2))
			p.Fill(fill, ui.RGB(255, 255, 255), h/2)
		})
		s.Children(func() {
			ui.Icon(c, icon).Size(h/2, h/2).TextColor(ui.RGB(110, 110, 115)).Absolute().Left(h / 4).Top(h / 4)
		})
	})
	if changed {
		tile.SetChanged()
	}
	return tile
}
