// Package easing is an editor of a cubic Bézier easing, as CSS's
// cubic-bezier() is and design tools' motion panels edit one: the curve
// of progress over time in a square, its two handles dragged to shape
// it, out above and below the square for a curve that overshoots;
// presets to start from; its values, to copy; and a dot playing it once
// a handle is let go of, and on a click.
//
//	easing.Editor(c, &app.curve)
//	...
//	x := e.AnimateTo("x", target, ui.Animation{Duration: 300 * time.Millisecond, Ease: app.curve.Easing()})
package easing

import (
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/egoist/mygo/ui"
)

// Curve is a cubic Bézier curve from (0, 0) to (1, 1), by its two control
// points, as CSS's cubic-bezier(X1, Y1, X2, Y2): X is time and Y progress.
// X1 and X2 are from 0 to 1.
type Curve struct {
	X1, Y1, X2, Y2 float32
}

// Easing returns the easing of c.
func (c Curve) Easing() ui.Easing { return ui.CubicBezier(c.X1, c.Y1, c.X2, c.Y2) }

// String returns c as CSS writes it, to two decimals:
// "cubic-bezier(0.25, 0.1, 0.25, 1)".
func (c Curve) String() string {
	f := func(v float32) string { return strconv.FormatFloat(math.Round(float64(v)*100)/100, 'f', -1, 64) }
	return fmt.Sprintf("cubic-bezier(%s, %s, %s, %s)", f(c.X1), f(c.Y1), f(c.X2), f(c.Y2))
}

// presets are the curves to start from: CSS's named ones, and two that
// overshoot.
var presets = []struct {
	name  string
	curve Curve
}{
	{"linear", Curve{0, 0, 1, 1}},
	{"ease", Curve{0.25, 0.1, 0.25, 1}},
	{"ease-in", Curve{0.42, 0, 1, 1}},
	{"ease-out", Curve{0, 0, 0.58, 1}},
	{"ease-in-out", Curve{0.42, 0, 0.58, 1}},
	{"back-out", Curve{0.34, 1.56, 0.64, 1}},
	{"back-in-out", Curve{0.68, -0.6, 0.32, 1.6}},
}

// overshoot is how far above and below the square a handle goes, as a
// share of it.
const overshoot = 0.6

// play is how long the dot takes to cross.
const play = 900 * time.Millisecond

// Editor creates the editor of *curve.
func Editor(c *ui.Context, curve *Curve) ui.Element {
	t := c.Theme()
	side := t.Space(44)
	pad := t.Space(3)
	margin := side * overshoot
	return ui.Row(c).Gap(t.Space(6)).AlignItems(ui.Start).Children(func() {
		plot := ui.Box(c).Size(side+2*pad, side+2*margin).Clip().Role(ui.RoleGroup).Label("Curve")
		started := ui.Local(plot, "easing.play", func() time.Time { return time.Time{} })
		// at returns where the point x, y of the curve is in the plot.
		at := func(x, y float32) (float32, float32) { return pad + x*side, margin + (1-y)*side }
		plot.Children(func() {
			for k, h := range []struct{ x, y *float32 }{{&curve.X1, &curve.Y1}, {&curve.X2, &curve.Y2}} {
				hx, hy := at(*h.x, *h.y)
				d := t.Space(3.5)
				handle := ui.Box(c.Key(k)).Absolute().Left(hx-d/2).Top(hy-d/2).Size(d, d).Cursor(ui.CursorGrab).
					Role(ui.RoleSlider).Label(fmt.Sprintf("Handle %d", k+1))
				if dx, dy, ok := handle.Dragged(); ok {
					*h.x = max(0, min(1, *h.x+dx/side))
					*h.y = max(-overshoot, min(1+overshoot, *h.y-dy/side))
				}
				if _, _, ok := handle.DragEnded(); ok {
					*started = c.Now()
				}
				color := t.Accent
				hot := handle.Hovered() || handle.Pressed()
				handle.Draw(func(p *ui.Painter, r ui.Rect) {
					var dot ui.Path
					radius := r.W / 2
					if !hot {
						radius -= 1.5
					}
					dot.Circle(r.X+r.W/2, r.Y+r.H/2, radius)
					p.FillPath(&dot, t.Background)
					p.StrokePath(&dot, 2, color)
				})
			}
		})
		cv := *curve
		plot.Draw(func(p *ui.Painter, r ui.Rect) {
			pt := func(x, y float32) (float32, float32) {
				px, py := at(x, y)
				return r.X + px, r.Y + py
			}
			x0, y0 := pt(0, 0)
			x1, y1 := pt(1, 1)
			square := ui.Rect{X: x0, Y: y1, W: x1 - x0, H: y0 - y1}
			p.Fill(square, t.Text.Alpha(0.04), 0)
			for k := 1; k < 4; k++ {
				f := float32(k) / 4
				gx, _ := pt(f, 0)
				_, gy := pt(0, f)
				p.Line(gx, y1, gx, y0, 1, t.Text.Alpha(0.06))
				p.Line(x0, gy, x1, gy, 1, t.Text.Alpha(0.06))
			}
			p.Stroke(square, t.Border, 0, 1)
			p.StrokePolyline([][2]float32{{x0, y0}, {x1, y1}}, 1, t.Text.Alpha(0.15))
			h1x, h1y := pt(cv.X1, cv.Y1)
			h2x, h2y := pt(cv.X2, cv.Y2)
			p.StrokePolyline([][2]float32{{x0, y0}, {h1x, h1y}}, 1.5, t.Accent.Alpha(0.6))
			p.StrokePolyline([][2]float32{{x1, y1}, {h2x, h2y}}, 1.5, t.Accent.Alpha(0.6))
			var path ui.Path
			path.MoveTo(x0, y0).CubeTo(h1x, h1y, h2x, h2y, x1, y1)
			p.StrokePath(&path, 2.5, t.Text)
		})
		ui.Column(c).Gap(t.Space(4)).Width(t.Space(72)).Children(func() {
			ui.Grid(c).Columns(4).Gap(t.Space(2)).Children(func() {
				for k, pr := range presets {
					b := ui.ButtonBase(c.Key(k)).Label(pr.name).Padding(t.Space(1.5), t.Space(0.5)).Radius(t.Space(1.5)).
						Border(1, t.Border)
					if pr.curve == *curve {
						b.Border(1, t.Accent).Background(t.Accent.Alpha(0.1))
					} else if b.Hovered() {
						b.Background(t.Text.Alpha(0.05))
					}
					if b.Clicked() {
						*curve = pr.curve
						*started = c.Now()
					}
					b.Children(func() {
						ui.Column(c).Gap(t.Space(1)).AlignItems(ui.Center).Children(func() {
							thumb(c, pr.curve)
							ui.Text(c, pr.name).FontSize(t.FontSize - 3).TextColor(t.TextMuted).SingleLine()
						})
					})
				}
			})
			ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
				ui.Text(c, curve.String()).Font("monospace").FontSize(t.FontSize - 1).Selectable().Grow(1).Shrink(1)
				if ui.Button(c, "Copy").Clicked() {
					c.WriteClipboard(curve.String())
				}
			})
			preview(c, *curve, started)
		})
	})
}

// thumb draws curve small, for its preset.
func thumb(c *ui.Context, curve Curve) {
	t := c.Theme()
	s := t.Space(9)
	ui.Box(c).Size(s, s).Draw(func(p *ui.Painter, r ui.Rect) {
		in := s * 0.2
		w := r.W - 2*in
		pt := func(x, y float32) (float32, float32) { return r.X + in + x*w, r.Y + in + (1-y)*w }
		x0, y0 := pt(0, 0)
		x1, y1 := pt(1, 1)
		ax, ay := pt(curve.X1, curve.Y1)
		bx, by := pt(curve.X2, curve.Y2)
		var path ui.Path
		path.MoveTo(x0, y0).CubeTo(ax, ay, bx, by, x1, y1)
		p.StrokePath(&path, 1.5, t.Text)
	})
}

// preview shows a dot crossing a track with curve, from *started; a click
// plays it again.
func preview(c *ui.Context, curve Curve, started *time.Time) {
	t := c.Theme()
	track := ui.ButtonBase(c).Label("Play").Height(t.Space(10)).Radius(t.Space(5)).Background(t.Text.Alpha(0.05))
	if track.Clicked() {
		*started = c.Now()
	}
	f := float32(1)
	if !started.IsZero() {
		if d := c.Now().Sub(*started); d < play {
			f = curve.Easing()(float32(d) / float32(play))
			c.After(time.Second / 60)
		}
	}
	track.Draw(func(p *ui.Painter, r ui.Rect) {
		d := r.H - t.Space(3)
		left, right := r.X+r.H/2, r.X+r.W-r.H/2
		x := left + f*(right-left)
		var dot ui.Path
		dot.Circle(x, r.Y+r.H/2, d/2)
		p.FillPath(&dot, t.Accent)
	})
}
