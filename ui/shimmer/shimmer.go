// Package shimmer shows text with a band of light sweeping across it, as
// AI apps show "Thinking…" while a model works:
//
//	if app.thinking {
//		shimmer.Text(c, "Thinking…")
//	}
//
// The band is the text drawn again in thin strips over it, brighter
// toward the band's middle, so it costs no shader, and repaints without
// building the view anew.
package shimmer

import (
	"math"

	"github.com/egoist/mygo/ui"
)

// period is how long the band takes to cross the text, in milliseconds,
// and strips how many strips draw it.
const (
	period = 1800
	strips = 12
)

// Text creates text s in the theme's muted color with a band of the text
// color sweeping across it, again and again. With the desktop asking for
// less motion, the text stays muted.
func Text(c *ui.Context, s string) ui.Element {
	t := c.Theme()
	span := ui.Span{Text: s, Size: t.FontSize, Color: t.TextMuted}
	w, h := c.MeasureText(0, span)
	still := c.Preferences().ReduceMotion
	return ui.Box(c).Size(float32(math.Ceil(float64(w))), h).Role(ui.RoleText).Label(s).Draw(func(p *ui.Painter, r ui.Rect) {
		p.RichText(r.X, r.Y, 0, span)
		if still {
			return
		}
		p.AnimationFrame()
		band := max(3*r.H, 0.4*r.W)
		f := float32(p.Now().UnixMilli()%period) / period
		center := r.X - band/2 + f*(r.W+band)
		sw := band / strips
		for k := range strips {
			x := center - band/2 + float32(k)*sw
			a := max(0, 1-abs(x+sw/2-center)/(band/2))
			shine := span
			shine.Color = t.Text.Alpha(a * a * (3 - 2*a))
			p.Clip(ui.Rect{X: x, Y: r.Y, W: sw, H: r.H}, 0, func() { p.RichText(r.X, r.Y, 0, shine) })
		}
	})
}

func abs(v float32) float32 { return max(v, -v) }
