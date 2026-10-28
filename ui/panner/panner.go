// Package panner places a sound in a surround mix, as the surround panners
// of mixing desks: the room seen from above, the listener in the middle
// facing the front, the speakers of a layout around them, and a puck for
// where the sound comes from, which a press or a drag puts anywhere in the
// room, the arrows nudge and a double click returns to the middle.
// Speakers light up as loud as the sound plays from them, and meters beside
// the room read out their gains in decibels.
//
// Gains pans by pairs at constant power: a sound at the edge of the room
// plays from the two speakers either side of its direction, and spreads
// to all of them evenly as it comes to the middle.
//
//	panner.View(c, &app.pan, panner.Surround51)
package panner

import (
	"cmp"
	"fmt"
	"math"
	"slices"

	"github.com/egoist/mygo/ui"
)

// Speaker is a speaker of a layout: its name and its direction from the
// listener, in degrees clockwise from the front.
type Speaker struct {
	Name  string
	Angle float64
}

// Layout is the speakers around the listener. The LFE of a .1 layout takes
// no part in panning, so the layouts leave it out.
type Layout []Speaker

var (
	// Surround51 is the layout of ITU-R BS.775: left, center and right in
	// front, and two surrounds behind the sides.
	Surround51 = Layout{{"L", -30}, {"C", 0}, {"R", 30}, {"Ls", -110}, {"Rs", 110}}
	// Surround71 is the layout of ITU-R BS.2051: the front three, two
	// side surrounds and two rear surrounds.
	Surround71 = Layout{{"L", -30}, {"C", 0}, {"R", 30}, {"Lss", -90}, {"Rss", 90}, {"Lrs", -150}, {"Rrs", 150}}
)

// Position is where a sound comes from: X from -1 at the left to 1 at the
// right, and Y from -1 at the back to 1 at the front.
type Position struct{ X, Y float64 }

// Gains returns how loud the sound at p plays from each speaker of l, from
// 0 to 1, their squares summing to 1.
func Gains(l Layout, p Position) []float64 {
	n := len(l)
	g := make([]float64, n)
	if n == 0 {
		return g
	}
	d := min(1, math.Hypot(p.X, p.Y))
	dir := math.Atan2(p.X, p.Y) * 180 / math.Pi
	order := make([]int, n)
	for i := range order {
		order[i] = i
	}
	slices.SortFunc(order, func(a, b int) int { return cmp.Compare(l[a].Angle, l[b].Angle) })
	for k, i := range order {
		j := order[(k+1)%n]
		a, b := l[i].Angle, l[j].Angle
		if b <= a {
			b += 360
		}
		at := dir
		if at < a {
			at += 360
		}
		if at >= a && at <= b {
			f := (at - a) / (b - a)
			g[i], g[j] = math.Cos(f*math.Pi/2), math.Sin(f*math.Pi/2)
			break
		}
	}
	for i := range g {
		g[i] = math.Sqrt(d*g[i]*g[i] + (1-d)/float64(n))
	}
	return g
}

func decibels(g float64) string {
	if g < 1e-5 {
		return "−∞"
	}
	db := 20 * math.Log10(g)
	if db > -0.05 {
		return "0.0"
	}
	return fmt.Sprintf("−%.1f", -db)
}

// View creates the panner of a sound at *p among the speakers of l, which
// it moves as the user drags it, and reports moving as changes.
func View(c *ui.Context, p *Position, l Layout) ui.Element {
	t := c.Theme()
	view := ui.Row(c).Gap(t.Space(6)).AlignItems(ui.Start)
	return view.Children(func() {
		gains := Gains(l, *p)
		room := ui.Box(c).AspectRatio(1).Grow(1).Basis(0).MinWidth(t.Space(60)).MaxWidth(t.Space(100)).Focusable().
			Cursor(ui.CursorCrosshair).Role(ui.RoleImage).
			Label(fmt.Sprintf("Surround panner, sound at %.2f across, %.2f front to back", p.X, p.Y))
		b := room.Bounds()
		inner := min(b.W, b.H) / 2 * 0.78
		changed := false
		if px, py, _ := room.PointerPosition(); room.Pressed() && inner > 0 {
			x, y := float64((px-b.W/2)/inner), float64((b.H/2-py)/inner)
			if r := math.Hypot(x, y); r > 1 {
				x, y = x/r, y/r
			}
			if x != p.X || y != p.Y {
				p.X, p.Y, changed = x, y, true
			}
		}
		if room.DoubleClicked() {
			*p, changed = Position{}, true
		}
		for _, nudge := range []struct {
			key    ui.Key
			dx, dy float64
		}{{ui.KeyLeft, -0.05, 0}, {ui.KeyRight, 0.05, 0}, {ui.KeyUp, 0, 0.05}, {ui.KeyDown, 0, -0.05}} {
			if room.Shortcut(0, nudge.key) {
				x, y := p.X+nudge.dx, p.Y+nudge.dy
				if r := math.Hypot(x, y); r > 1 {
					x, y = x/r, y/r
				}
				p.X, p.Y, changed = x, y, true
			}
		}
		if changed {
			gains = Gains(l, *p)
			view.SetChanged()
		}

		pos := *p
		room.Draw(func(pt *ui.Painter, r ui.Rect) {
			cx, cy := r.X+r.W/2, r.Y+r.H/2
			half := min(r.W, r.H) / 2
			inner := half * 0.78
			pt.Fill(ui.Rect{X: cx - half, Y: cy - half, W: 2 * half, H: 2 * half}, t.Surface, t.Radius*2)
			var ring ui.Path
			pt.StrokePath(ring.Circle(cx, cy, inner), 1, t.Border)
			var mid ui.Path
			pt.StrokePath(mid.Circle(cx, cy, inner/2), 1, t.Border.Alpha(0.5))
			at := func(angle float64, radius float32) (float32, float32) {
				a := angle * math.Pi / 180
				return cx + radius*float32(math.Sin(a)), cy - radius*float32(math.Cos(a))
			}
			puckX, puckY := cx+float32(pos.X)*inner, cy-float32(pos.Y)*inner
			for i, s := range l {
				sx, sy := at(s.Angle, inner)
				pt.StrokePolyline([][2]float32{{puckX, puckY}, {sx, sy}}, 1+3*float32(gains[i]), t.Accent.Alpha(0.1+0.5*float32(gains[i])))
			}
			for i, s := range l {
				sx, sy := at(s.Angle, inner)
				size := t.Space(7)
				box := ui.Rect{X: sx - size/2, Y: sy - size/2, W: size, H: size}
				pt.Fill(box, t.Background, t.Radius)
				pt.Fill(box, t.Accent.Alpha(float32(gains[i])), t.Radius)
				pt.Stroke(box, t.Border, t.Radius, 1)
				fs := t.FontSize - 2
				w, h := pt.MeasureText(0, ui.Span{Text: s.Name, Size: fs, Weight: 600})
				color := t.Text
				if gains[i] > 0.6 {
					color = t.AccentText
				}
				pt.RichText(sx-w/2, sy-h/2, 0, ui.Span{Text: s.Name, Size: fs, Weight: 600, Color: color})
			}
			var head ui.Path
			head.MoveTo(cx-6, cy-6).LineTo(cx, cy-14).LineTo(cx+6, cy-6).Close()
			pt.FillPath(&head, t.TextMuted)
			var listener ui.Path
			pt.FillPath(listener.Circle(cx, cy, 9), t.TextMuted)
			var puck ui.Path
			puck.Circle(puckX, puckY, t.Space(3))
			pt.Shadow(ui.Rect{X: puckX - t.Space(3), Y: puckY - t.Space(3), W: t.Space(6), H: t.Space(6)}, t.Space(3), 0, 1, 4, 0, ui.RGBA(0, 0, 0, 0.3))
			pt.FillPath(&puck, t.Accent)
			pt.StrokePath(&puck, 2, t.Background)
		})

		ui.Column(c).Gap(t.Space(2)).Width(t.Space(44)).Children(func() {
			for i, s := range l {
				ui.Row(c.Key(i)).Gap(t.Space(2)).AlignItems(ui.Center).Label(fmt.Sprintf("%s: %s dB", s.Name, decibels(gains[i]))).Children(func() {
					ui.Text(c, s.Name).Bold().Width(t.Space(8))
					level := max(0, 1+20*math.Log10(max(gains[i], 1e-5))/40)
					ui.Box(c).Grow(1).Height(t.Space(2)).Radius(t.Space(1)).Background(t.Surface).Draw(func(p *ui.Painter, r ui.Rect) {
						p.Fill(ui.Rect{X: r.X, Y: r.Y, W: r.W * float32(level), H: r.H}, t.Accent, r.H/2)
					})
					ui.Text(c, decibels(gains[i])).FontFeatures("tnum").Width(t.Space(10)).TextAlign(ui.End).NoWrap()
				})
			}
		})
	})
}
