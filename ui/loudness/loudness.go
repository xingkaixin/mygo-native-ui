// Package loudness meters loudness as broadcasters and streaming services
// measure it, by EBU R128: two bars for the momentary and short-term
// loudness, green up to the target, amber within 3 LU over it and red
// beyond, the momentary bar holding its peak for two seconds; the
// integrated loudness in large figures with how far it is from the
// target; the loudness range and the peak; and the short-term loudness
// of the last minute as a graph against the target.
//
//	m := loudness.NewMeter(48000, 2)
//	m.Write(samples) // as they play
//	loudness.View(c, &app.meter, m, -23)
package loudness

import (
	"fmt"
	"math"
	"time"

	"github.com/egoist/mygo/ui"
)

// State is what a view keeps: the momentary peak it holds. Keep it in
// the app's state, one for each view.
type State struct {
	hold   float64
	heldAt time.Time
}

const (
	floor = -48.0
	ceil  = 0.0
)

// View creates the meter of m against a target loudness in LUFS, such as
// -23 for broadcast or -14 for streaming.
func View(c *ui.Context, s *State, m *Meter, target float64) ui.Element {
	t := c.Theme()
	now := c.Now()
	mom, short, integ := m.Momentary(), m.ShortTerm(), m.Integrated()
	if mom >= s.hold || now.Sub(s.heldAt) > 2*time.Second {
		s.hold, s.heldAt = mom, now
	}
	green, amber, red := ui.RGB(52, 190, 96), ui.RGB(240, 170, 40), ui.RGB(230, 60, 60)
	zone := func(l float64) ui.Color {
		switch {
		case l > target+3:
			return red
		case l > target:
			return amber
		}
		return green
	}
	root := ui.Row(c).Gap(t.Space(6)).AlignItems(ui.Stretch).Role(ui.RoleGroup).Label("Loudness")
	root.Children(func() {
		bars := ui.Box(c).Width(t.Space(30)).Role(ui.RoleImage).Label(fmt.Sprintf("Momentary %s, short-term %s", lu(mom), lu(short)))
		bars.Draw(func(p *ui.Painter, r ui.Rect) {
			small := t.FontSize - 2
			foot := small * 1.8
			h := r.H - foot - t.Space(2)
			yOf := func(l float64) float32 {
				f := (max(floor, min(ceil, l)) - floor) / (ceil - floor)
				return r.Y + t.Space(2) + h*(1-float32(f))
			}
			scaleW := t.Space(9)
			for l := ceil; l >= floor; l -= 6 {
				sp := ui.Span{Text: fmt.Sprintf("%.0f", l), Size: small, Color: t.TextMuted, Features: "tnum"}
				w, lh := p.MeasureText(0, sp)
				p.RichText(r.X+scaleW-w-t.Space(1.5), yOf(l)-lh/2, 0, sp)
			}
			barW := (r.W - scaleW - t.Space(3)) / 2
			for k, bar := range []struct {
				name string
				l    float64
			}{{"M", mom}, {"S", short}} {
				x := r.X + scaleW + float32(k)*(barW+t.Space(3))
				track := ui.Rect{X: x, Y: yOf(ceil), W: barW, H: yOf(floor) - yOf(ceil)}
				p.Fill(track, t.Surface, 3)
				// Each zone fills only as far as the bar reaches.
				for _, z := range []struct {
					lo, hi float64
					col    ui.Color
				}{{floor, target, green}, {target, target + 3, amber}, {target + 3, ceil, red}} {
					top := min(bar.l, z.hi)
					if top <= z.lo {
						continue
					}
					p.Fill(ui.Rect{X: x, Y: yOf(top), W: barW, H: yOf(z.lo) - yOf(top)}, z.col, 0)
				}
				if k == 0 && s.hold > floor {
					p.Fill(ui.Rect{X: x, Y: yOf(s.hold) - 1, W: barW, H: 2}, zone(s.hold), 0)
				}
				ty := yOf(target)
				p.Fill(ui.Rect{X: x - 3, Y: ty - 0.75, W: barW + 6, H: 1.5}, t.Text, 0)
				sp := ui.Span{Text: bar.name, Size: small, Weight: 600, Color: t.TextMuted}
				w, _ := p.MeasureText(0, sp)
				p.RichText(x+(barW-w)/2, r.Y+r.H-foot+t.Space(1), 0, sp)
			}
		})
		ui.Column(c).Grow(1).Gap(t.Space(5)).Children(func() {
			ui.Column(c).Gap(t.Space(1)).Children(func() {
				ui.Text(c, "Integrated").FontSize(t.FontSize - 1).TextColor(t.TextMuted)
				ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Baseline).Children(func() {
					ui.Text(c, number(integ)).FontSize(t.FontSize + 30).FontWeight(700).FontFeatures("tnum")
					ui.Text(c, "LUFS").TextColor(t.TextMuted)
				})
				off := "—"
				col := t.TextMuted
				if !math.IsInf(integ, -1) {
					off = fmt.Sprintf("%+.1f LU from %.0f", integ-target, target)
					col = zone(integ)
					if math.Abs(integ-target) <= 1 {
						col = green
					}
				}
				ui.Text(c, off).TextColor(col).FontFeatures("tnum").FontWeight(600)
			})
			ui.Row(c).Gap(t.Space(8)).Children(func() {
				for _, f := range []struct{ name, value string }{
					{"Short-term", lu(short)},
					{"Range", fmt.Sprintf("%.1f LU", m.Range())},
					{"Peak", dbfs(m.Peak())},
				} {
					ui.Column(c.Key(f.name)).Gap(t.Space(0.5)).Children(func() {
						ui.Text(c, f.name).FontSize(t.FontSize - 1).TextColor(t.TextMuted)
						ui.Text(c, f.value).FontSize(t.FontSize + 3).FontWeight(600).FontFeatures("tnum")
					})
				}
			})
			graph := ui.Box(c).Grow(1).Background(t.Surface).Radius(t.Radius + 2).Role(ui.RoleImage).Label("Short-term loudness of the last minute")
			graph.Draw(func(p *ui.Painter, r ui.Rect) {
				hist := m.History()
				const span = 600
				lo, hi := target-24, target+12
				yOf := func(l float64) float32 {
					f := (max(lo, min(hi, l)) - lo) / (hi - lo)
					return r.Y + r.H*(1-float32(f))
				}
				for l := hi; l > lo; l -= 6 {
					y := yOf(l)
					p.Fill(ui.Rect{X: r.X, Y: y, W: r.W, H: 1}, t.Border.Alpha(0.5), 0)
					p.RichText(r.X+t.Space(1.5), y+1, 0, ui.Span{Text: fmt.Sprintf("%.0f", l), Size: t.FontSize - 3, Color: t.TextMuted, Features: "tnum"})
				}
				ty := yOf(target)
				p.StrokePolyline([][2]float32{{r.X, ty}, {r.X + r.W, ty}}, 1.5, t.Text.Alpha(0.6))
				from := max(0, len(hist)-span)
				var pts [][2]float32
				for i, l := range hist[from:] {
					x := r.X + r.W*float32(i+span-len(hist[from:]))/span
					pts = append(pts, [2]float32{x, yOf(l)})
				}
				if len(pts) > 1 {
					fill := append([][2]float32{{pts[0][0], r.Y + r.H}}, pts...)
					fill = append(fill, [2]float32{pts[len(pts)-1][0], r.Y + r.H})
					var area ui.Path
					area.MoveTo(fill[0][0], fill[0][1])
					for _, q := range fill[1:] {
						area.LineTo(q[0], q[1])
					}
					p.FillPath(area.Close(), t.Accent.Alpha(0.15))
					p.StrokePolyline(pts, 2, t.Accent)
				}
			})
		})
	})
	return root
}

func number(l float64) string {
	if math.IsInf(l, -1) || l < -70 {
		return "—"
	}
	return fmt.Sprintf("%.1f", l)
}

func lu(l float64) string {
	if math.IsInf(l, -1) || l < -70 {
		return "—"
	}
	return fmt.Sprintf("%.1f LUFS", l)
}

func dbfs(l float64) string {
	if math.IsInf(l, -1) {
		return "—"
	}
	return fmt.Sprintf("%.1f dBFS", l)
}
