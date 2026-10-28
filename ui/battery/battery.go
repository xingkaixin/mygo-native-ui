// Package battery shows a laptop's battery over the last day, as the
// Battery settings of macOS do: the level now with its glyph, whether it
// charges and when it last charged to its fullest; the level in bars of a
// quarter of an hour, green, red at 20% and below, with the time spent
// charging shaded behind them and a bolt over it; and how long the screen
// was on in each hour. The pointer over either chart picks a time,
// marked in both, and reads out the level, and the screen time of its
// hour.
//
//	battery.View(c, app.samples, app.screen)
package battery

import (
	"fmt"
	"math"
	"time"

	"github.com/egoist/mygo/ui"
)

// Sample is the battery at a moment: its level from 0 to 1, and whether
// it was charging.
type Sample struct {
	At       time.Time
	Level    float64
	Charging bool
}

// Usage is how long the screen was on in the hour starting at At.
type Usage struct {
	At     time.Time
	Screen time.Duration
}

const (
	slots = 96
	slot  = 15 * time.Minute
)

type bar struct {
	level    float64
	charging bool
	ok       bool
}

// bin returns the level of each quarter of an hour up to end, the last
// sample in it, and whether it was charging in it.
func bin(samples []Sample, end time.Time) [slots]bar {
	var bars [slots]bar
	start := end.Add(-slots * slot)
	for _, sm := range samples {
		i := int(sm.At.Sub(start) / slot)
		if i < 0 || i >= slots {
			continue
		}
		bars[i].level, bars[i].ok = sm.Level, true
		bars[i].charging = bars[i].charging || sm.Charging
	}
	return bars
}

var bolt = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path d="M13 2L4 14h7l-1 8 9-12h-7z" fill="currentColor"/></svg>`))

// View creates the battery view of samples and screen, oldest first.
func View(c *ui.Context, samples []Sample, screen []Usage) ui.Element {
	t := c.Theme()
	root := ui.Column(c).Gap(t.Space(4))
	if len(samples) == 0 {
		return root
	}
	last := samples[len(samples)-1]
	end := last.At.Truncate(time.Hour).Add(time.Hour)
	bars := bin(samples, end)
	start := end.Add(-slots * slot)
	var usage [24]time.Duration
	for _, u := range screen {
		if i := int(u.At.Sub(start) / time.Hour); i >= 0 && i < 24 {
			usage[i] += u.Screen
		}
	}
	root.Children(func() {
		head(c, samples, last)
		picked := -1
		ui.Text(c, "Battery Level").Bold()
		level := chart(c, "Battery level", t.Space(36))
		ui.Text(c, "Screen On Usage").Bold()
		use := chart(c, "Screen on usage", t.Space(24))
		for _, e := range []ui.Element{level, use} {
			if px, _, in := e.PointerPosition(); in {
				w := e.LayoutBounds().W - axis
				picked = max(0, min(slots-1, int(px/w*slots)))
			}
		}
		level.Draw(func(p *ui.Painter, r ui.Rect) {
			plot := ui.Rect{X: r.X, Y: r.Y, W: r.W - axis, H: r.H - t.Space(5)}
			grid(p, plot, []string{"0%", "50%", "100%"})
			bw := plot.W / slots
			for i := 0; i < slots; i++ {
				if !bars[i].charging {
					continue
				}
				from := i
				for i < slots && bars[i].charging {
					i++
				}
				x := plot.X + float32(from)*bw
				p.Fill(ui.Rect{X: x, Y: plot.Y, W: float32(i-from) * bw, H: plot.H}, t.Success.Alpha(0.14), 0)
				p.Icon(bolt, ui.Rect{X: x + 2, Y: plot.Y + 2, W: 11, H: 11}, t.Success)
			}
			for i, b := range bars {
				x := plot.X + float32(i)*bw
				if !b.ok {
					continue
				}
				col := t.Success
				if b.level <= 0.2 {
					col = t.Danger
				}
				if picked >= 0 && i != picked {
					col = col.Alpha(0.55)
				}
				h := float32(b.level) * plot.H
				p.Fill(ui.Rect{X: x + bw*0.15, Y: plot.Y + plot.H - h, W: bw * 0.7, H: h}, col, bw*0.2)
			}
			hours(p, plot, start)
			mark(p, plot, picked)
		})
		use.Draw(func(p *ui.Painter, r ui.Rect) {
			plot := ui.Rect{X: r.X, Y: r.Y, W: r.W - axis, H: r.H - t.Space(5)}
			grid(p, plot, []string{"0m", "30m", "60m"})
			hw := plot.W / 24
			for i, d := range usage {
				h := float32(min(1, d.Hours())) * plot.H
				col := t.Accent
				if picked >= 0 && i != picked/4 {
					col = col.Alpha(0.55)
				}
				p.Fill(ui.Rect{X: plot.X + float32(i)*hw + hw*0.18, Y: plot.Y + plot.H - h, W: hw * 0.64, H: h}, col, 2)
			}
			hours(p, plot, start)
			mark(p, plot, picked)
		})
		read := " "
		if picked >= 0 {
			at := start.Add(time.Duration(picked) * slot)
			b := bars[picked]
			read = at.Format("3:04 PM")
			if b.ok {
				read += fmt.Sprintf(" · %.0f%%", b.level*100)
			}
			if b.charging {
				read += " · Charging"
			}
			read += " · Screen on " + minutes(usage[picked/4]) + " that hour"
		}
		ui.Text(c, read).FontSize(t.FontSize - 1).TextColor(t.TextMuted).FontFeatures("tnum")
	})
	return root
}

// axis is how wide the labels right of a chart are.
const axis = 40

func chart(c *ui.Context, label string, h float32) ui.Element {
	return ui.Box(c.Key(label)).FillWidth().Height(h + c.Theme().Space(5)).Role(ui.RoleImage).Label(label)
}

// head creates the level now, its glyph, and what the battery is doing.
func head(c *ui.Context, samples []Sample, last Sample) {
	t := c.Theme()
	ui.Row(c).Gap(t.Space(4)).AlignItems(ui.Center).Children(func() {
		ui.Box(c).Size(t.Space(14), t.Space(7)).Draw(func(p *ui.Painter, r ui.Rect) {
			body := ui.Rect{X: r.X, Y: r.Y, W: r.W - 4, H: r.H}
			p.Stroke(body, t.TextMuted, 6, 1.5)
			p.Fill(ui.Rect{X: r.X + r.W - 3, Y: r.Y + r.H*0.33, W: 3, H: r.H * 0.34}, t.TextMuted, 1.5)
			col := t.Text
			switch {
			case last.Charging:
				col = t.Success
			case last.Level <= 0.2:
				col = t.Danger
			}
			in := ui.Rect{X: body.X + 3, Y: body.Y + 3, W: (body.W - 6) * float32(last.Level), H: body.H - 6}
			p.Fill(in, col, 3.5)
			if last.Charging {
				p.Icon(bolt, ui.Rect{X: body.X + body.W/2 - 8, Y: body.Y + body.H/2 - 8, W: 16, H: 16}, t.Text)
			}
		})
		ui.Column(c).Children(func() {
			ui.Text(c, fmt.Sprintf("%.0f%%", last.Level*100)).FontSize(t.FontSize + 10).Bold().FontFeatures("tnum")
			status := "On battery"
			if last.Charging {
				status = "Charging"
			}
			if at, level, ok := lastCharged(samples); ok {
				status += fmt.Sprintf(" · Last charged to %.0f%% at %s", level*100, at.Format("3:04 PM"))
			}
			ui.Text(c, status).TextColor(t.TextMuted)
		})
	})
}

// lastCharged returns when the last charge that has ended ended, and to
// what level.
func lastCharged(samples []Sample) (time.Time, float64, bool) {
	for i := len(samples) - 1; i > 0; i-- {
		if samples[i-1].Charging && !samples[i].Charging {
			return samples[i-1].At, samples[i-1].Level, true
		}
	}
	return time.Time{}, 0, false
}

// grid draws a chart's three lines with their labels on its right.
func grid(p *ui.Painter, plot ui.Rect, labels []string) {
	t := p.Theme()
	for k, l := range labels {
		y := plot.Y + plot.H - plot.H*float32(k)/float32(len(labels)-1)
		p.Line(plot.X, y, plot.X+plot.W, y, 1, t.Border)
		p.Text(plot.X+plot.W+6, y-t.FontSize/2, l, t.FontSize-2, t.TextMuted)
	}
}

// hours labels every third hour under a chart.
func hours(p *ui.Painter, plot ui.Rect, start time.Time) {
	t := p.Theme()
	hw := plot.W / 24
	for h := 0; h < 24; h++ {
		at := start.Add(time.Duration(h) * time.Hour)
		if at.Hour()%3 != 0 {
			continue
		}
		label := at.Format("3")
		if at.Hour()%12 == 0 {
			label = at.Format("3 PM")
		}
		p.Text(plot.X+float32(h)*hw, plot.Y+plot.H+4, label, t.FontSize-3, t.TextMuted)
	}
}

// mark draws the time picked across a chart.
func mark(p *ui.Painter, plot ui.Rect, picked int) {
	if picked < 0 {
		return
	}
	x := plot.X + (float32(picked)+0.5)*plot.W/slots
	p.Line(float32(math.Round(float64(x))), plot.Y, float32(math.Round(float64(x))), plot.Y+plot.H, 1, p.Theme().Text.Alpha(0.5))
}

func minutes(d time.Duration) string {
	return fmt.Sprintf("%dm", int(d.Minutes()))
}
