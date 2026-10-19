// Package trayicon draws the icons of menu bar and notification area apps
// with native UI, for Tray.SetIcon: live numbers, gauges and graphs, as
// Stats and iStat Menus show, from any view. Render draws a view into a
// PNG; Bars is a graph of recent values to draw, and package ring's
// Progress a gauge:
//
//	png, err := trayicon.Render(28, 18, func(c *ui.Context) {
//		trayicon.Bars(c, app.cpu, ui.RGB(0, 0, 0))
//	})
//	app.tray.SetIcon(png, true)
//
// macOS shows menu bar icons 18 points tall, scaling the PNG to that
// height, and tints template icons, black on transparent, to fit the bar,
// light or dark, and highlights them while their menu or panel is open:
// draw them black, and pass template true. Windows shows icons as drawn,
// 16 points square.
//
// Render builds the view anew for each icon, so call it as the values
// change, not every frame: once or twice a second is what menu bar apps
// do.
package trayicon

import (
	"bytes"
	"image/png"

	"github.com/egoist/mygo/ui"
)

// Render draws view in width×height DIPs, at two pixels per DIP for
// Retina screens, on transparent, and returns it as a PNG. Text in it is
// black, or the color it is given.
func Render(width, height int, view func(c *ui.Context)) ([]byte, error) {
	img := ui.Render(func(c *ui.Context) {
		t := *c.Theme()
		// The root paints the theme's background.
		t.Background, t.Text = ui.Transparent, ui.RGB(0, 0, 0)
		c.SetTheme(&t)
		view(c)
	}, width, height, 2)
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// Bars builds a graph of values, from 0 to 1, as bars of color side by
// side, the oldest at the left, filling its parent: a second's CPU load
// each, as Activity Monitor's Dock icon shows. The bars are a pixel apart
// at least, and the latest values show where there are more than fit.
func Bars(c *ui.Context, values []float64, color ui.Color) ui.Element {
	return ui.Box(c).Fill().Draw(func(p *ui.Painter, r ui.Rect) {
		gap := 1 / p.Scale()
		n := min(len(values), int((r.W+gap)/(2*gap)))
		if n == 0 {
			return
		}
		w := (r.W + gap) / float32(n)
		for i, v := range values[len(values)-n:] {
			h := r.H * float32(max(0, min(1, v)))
			p.Fill(ui.Rect{X: r.X + float32(i)*w, Y: r.Y + r.H - h, W: w - gap, H: h}, color, 0)
		}
	})
}
