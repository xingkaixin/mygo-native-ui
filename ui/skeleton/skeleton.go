// Package skeleton shows placeholders in the shape of content still
// loading, as feeds and profiles show them: lines of text, blocks for
// pictures and cards, and circles for avatars, in a faint gray with a
// band of light sweeping across. The band crosses the window rather than
// each placeholder, so the placeholders of a view shine as one; it costs
// no shader, and repaints without building the view anew.
//
//	if app.loading {
//		ui.Row(c).Gap(12).Children(func() {
//			skeleton.Circle(c, 40)
//			skeleton.Text(c, 2).Grow(1)
//		})
//	}
package skeleton

import (
	"github.com/egoist/mygo/ui"
)

// period is how long the band takes to cross the window, in
// milliseconds.
const period = 1600

// Box creates a block placeholder, as for a picture or a card: give it a
// size, or let it stretch.
func Box(c *ui.Context) ui.Element {
	return shape(c, c.Theme().Radius)
}

// Circle creates a round placeholder d across, as for an avatar.
func Circle(c *ui.Context, d float32) ui.Element {
	return shape(c, d/2).Size(d, d)
}

// Line creates a placeholder of a line of text, w wide, as high as the
// text's capitals, with room above and below it as a line has.
func Line(c *ui.Context, w float32) ui.Element {
	t := c.Theme()
	return ui.Box(c).Width(w).PaddingY(t.FontSize * 0.35).Children(func() {
		shape(c, t.FontSize*0.35).Height(t.FontSize * 0.7).FillWidth()
	})
}

// Text creates a placeholder of a paragraph of n lines as wide as it
// is, the last shorter, as a paragraph ends.
func Text(c *ui.Context, n int) ui.Element {
	t := c.Theme()
	return ui.Column(c).Label("Loading").Children(func() {
		for i := range n {
			line := ui.Box(c).PaddingY(t.FontSize * 0.35)
			if i == n-1 && n > 1 {
				line.WidthPercent(60)
			}
			line.Children(func() {
				shape(c, t.FontSize*0.35).Height(t.FontSize * 0.7).FillWidth()
			})
		}
	})
}

// shape creates a placeholder rounded by radius, its band placed by the
// time and the window's width.
func shape(c *ui.Context, radius float32) ui.Element {
	t := c.Theme()
	base := t.Text.Alpha(0.08)
	shine := ui.RGBA(255, 255, 255, 0.55)
	if t.Dark {
		shine = ui.RGBA(255, 255, 255, 0.07)
	}
	w, _ := c.Size()
	still := c.Preferences().ReduceMotion
	return ui.Box(c).Radius(radius).Background(base).Draw(func(p *ui.Painter, r ui.Rect) {
		if still {
			return
		}
		p.AnimationFrame()
		band := max(240, w*0.3)
		f := float32(p.Now().UnixMilli()%period) / period
		x := -band + f*(w+2*band)
		if x+band < r.X || x > r.X+r.W {
			return
		}
		p.Clip(r, radius, func() {
			p.FillGradient(ui.Rect{X: x, Y: r.Y, W: band, H: r.H}, ui.LinearGradient{Angle: 90, Stops: []ui.GradientStop{
				{Color: shine.Alpha(0), At: 0}, {Color: shine, At: 0.5}, {Color: shine.Alpha(0), At: 1},
			}}, 0)
		})
	})
}
