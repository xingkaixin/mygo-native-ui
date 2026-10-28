// Package hypnogram shows a night's sleep by stage, as sleep trackers
// do: how long was slept and when, a chart with a lane for each stage
// (awake at the top, deep sleep at the bottom) and a bar for each spell
// in its lane, joined to the next spell's, the hours along the bottom,
// and below it how long and what share of the night each stage took.
// Hovering the chart reads out the spell under the pointer.
//
//	hypnogram.View(c, &app.sleep, night)
package hypnogram

import (
	"fmt"
	"time"

	"github.com/egoist/mygo/ui"
)

// Stage is a stage of sleep, or being awake.
type Stage int

const (
	Awake Stage = iota
	REM
	Core
	Deep
)

// Stages are the stages from the top lane to the bottom one.
var Stages = []Stage{Awake, REM, Core, Deep}

func (s Stage) String() string { return [...]string{"Awake", "REM", "Core", "Deep"}[s] }

var colors = [...]ui.Color{ui.RGB(250, 128, 96), ui.RGB(88, 196, 250), ui.RGB(52, 122, 240), ui.RGB(70, 56, 180)}

// Spell is a stretch of the night in one stage.
type Spell struct {
	Stage      Stage
	Start, End time.Time
}

// State is what a hypnogram keeps: the spell under the pointer. Keep it
// in the app's state, one for each hypnogram.
type State struct {
	hover int
}

// View creates the hypnogram of a night, its spells in order.
func View(c *ui.Context, s *State, night []Spell) ui.Element {
	t := c.Theme()
	root := ui.Column(c).Gap(t.Space(4)).Role(ui.RoleGroup).Label("Sleep")
	if len(night) == 0 {
		return root
	}
	var spent [4]time.Duration
	for _, sp := range night {
		spent[sp.Stage] += sp.End.Sub(sp.Start)
	}
	from, to := night[0].Start, night[len(night)-1].End
	inBed := to.Sub(from)
	asleep := inBed - spent[Awake]
	root.Children(func() {
		ui.Column(c).Gap(t.Space(1)).Children(func() {
			ui.Text(c, hm(asleep)+" asleep").FontSize(t.FontSize + 8).FontWeight(700).FontFeatures("tnum")
			ui.Text(c, fmt.Sprintf("In bed %s – %s · %.0f%% asleep", from.Format("15:04"), to.Format("15:04"),
				100*asleep.Seconds()/max(1, inBed.Seconds()))).TextColor(t.TextMuted).FontFeatures("tnum")
		})
		chart := ui.Box(c).Height(t.Space(56)).Role(ui.RoleImage).Label("Stages through the night")
		b := chart.LayoutBounds()
		gutter := t.Space(16)
		plotW := max(1, b.W-gutter)
		xOf := func(at time.Time) float32 { return gutter + float32(at.Sub(from).Seconds()/inBed.Seconds())*plotW }
		hover := -1
		if px, _, ok := chart.PointerPosition(); ok && px >= gutter {
			at := from.Add(time.Duration(float64(px-gutter) / float64(plotW) * float64(inBed)))
			for i, sp := range night {
				if !at.Before(sp.Start) && at.Before(sp.End) {
					hover = i
				}
			}
		}
		if hover+1 != s.hover {
			s.hover = hover + 1
			c.Invalidate()
		}
		chart.Draw(func(p *ui.Painter, r ui.Rect) {
			small := t.FontSize - 2
			axis := t.Space(6)
			lane := (r.H - axis) / float32(len(Stages))
			bar := lane * 0.5
			mid := func(st Stage) float32 { return r.Y + lane*(float32(st)+0.5) }
			for _, st := range Stages {
				p.RichText(r.X, mid(st)-small*0.65, 0, ui.Span{Text: st.String(), Size: small, Color: t.TextMuted})
				y := r.Y + lane*float32(st+1)
				p.StrokePolyline([][2]float32{{r.X + gutter, y}, {r.X + r.W, y}}, 1, t.Border.Alpha(0.5))
			}
			// Hour lines and labels, every hour on the hour.
			for h := from.Truncate(time.Hour).Add(time.Hour); h.Before(to); h = h.Add(time.Hour) {
				x := r.X + xOf(h)
				p.StrokePolyline([][2]float32{{x, r.Y}, {x, r.Y + r.H - axis}}, 1, t.Border.Alpha(0.5))
				sp := ui.Span{Text: h.Format("15:04"), Size: small, Color: t.TextMuted, Features: "tnum"}
				w, _ := p.MeasureText(0, sp)
				p.RichText(x-w/2, r.Y+r.H-axis+t.Space(1.5), 0, sp)
			}
			for i, sp := range night {
				if i == 0 {
					continue
				}
				prev := night[i-1]
				x := r.X + xOf(sp.Start)
				y0, y1 := mid(prev.Stage), mid(sp.Stage)
				if y0 > y1 {
					y0, y1 = y1, y0
				}
				p.FillGradient(ui.Rect{X: x - 0.75, Y: y0, W: 1.5, H: y1 - y0},
					ui.LinearGradient{From: colors[min(prev.Stage, sp.Stage)].Alpha(0.6), To: colors[max(prev.Stage, sp.Stage)].Alpha(0.6), Angle: 180}, 0)
			}
			for i, sp := range night {
				x0, x1 := r.X+xOf(sp.Start), r.X+xOf(sp.End)
				col := colors[sp.Stage]
				if hover >= 0 && i != hover {
					col = col.Alpha(0.45)
				}
				p.Fill(ui.Rect{X: x0, Y: mid(sp.Stage) - bar/2, W: max(2, x1-x0), H: bar}, col, min(bar/2, max(1, (x1-x0)/2)))
			}
			if hover >= 0 {
				sp := night[hover]
				lines := []ui.Span{
					{Text: sp.Stage.String(), Size: small, Weight: 600, Color: t.Text},
					{Text: fmt.Sprintf("%s – %s · %s", sp.Start.Format("15:04"), sp.End.Format("15:04"), hm(sp.End.Sub(sp.Start))),
						Size: small, Color: t.TextMuted, Features: "tnum"},
				}
				pad := t.Space(2)
				w, h := float32(0), float32(0)
				for _, l := range lines {
					lw, lh := p.MeasureText(0, l)
					w, h = max(w, lw), h+lh
				}
				x := r.X + (xOf(sp.Start)+xOf(sp.End))/2 - w/2 - pad
				x = max(r.X+gutter, min(x, r.X+r.W-w-2*pad))
				y := mid(sp.Stage) + bar/2 + t.Space(1.5)
				if sp.Stage == Deep {
					y = mid(sp.Stage) - bar/2 - t.Space(1.5) - h - 2*pad
				}
				box := ui.Rect{X: x, Y: y, W: w + 2*pad, H: h + 2*pad}
				p.Shadow(box, t.Radius, 0, 2, 8, 0, ui.RGBA(0, 0, 0, 0.18))
				p.Fill(box, t.Background, t.Radius)
				p.Stroke(box, t.Border, t.Radius, 1)
				ty := box.Y + pad
				for _, l := range lines {
					_, lh := p.MeasureText(0, l)
					p.RichText(box.X+pad, ty, 0, l)
					ty += lh
				}
			}
		})
		ui.Grid(c).Columns(4).Gap(t.Space(3)).Children(func() {
			for _, st := range Stages {
				share := spent[st].Seconds() / inBed.Seconds()
				ui.Column(c.Key(st)).Gap(t.Space(1)).Children(func() {
					ui.Row(c).Gap(t.Space(1.5)).AlignItems(ui.Center).Children(func() {
						ui.Box(c).Size(t.Space(2.5), t.Space(2.5)).Radius(t.Space(1.25)).Background(colors[st])
						ui.Text(c, st.String()).FontSize(t.FontSize - 1).TextColor(t.TextMuted)
					})
					ui.Text(c, hm(spent[st])).FontWeight(600).FontFeatures("tnum")
					track := ui.Box(c).Height(t.Space(1.5)).Radius(t.Space(0.75)).Background(t.Surface)
					track.Draw(func(p *ui.Painter, r ui.Rect) {
						p.Fill(ui.Rect{X: r.X, Y: r.Y, W: r.W * float32(share), H: r.H}, colors[st], r.H/2)
					})
					ui.Text(c, fmt.Sprintf("%.0f%%", 100*share)).FontSize(t.FontSize - 2).TextColor(t.TextMuted).FontFeatures("tnum")
				})
			}
		})
	})
	return root
}

// hm writes a duration as hours and minutes: 7h 12m, or 28m.
func hm(d time.Duration) string {
	m := int(d.Round(time.Minute).Minutes())
	if m < 60 {
		return fmt.Sprintf("%dm", m)
	}
	return fmt.Sprintf("%dh %02dm", m/60, m%60)
}
