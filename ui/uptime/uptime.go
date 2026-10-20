// Package uptime shows how services have been up, as status pages do: a
// banner of how they are as a whole, such as "All systems operational" or
// "Partial outage", then a row for each service with its name, how it is
// now and its uptime over the period, above a bar for each day of the
// period colored by that day's uptime, muted for days with no data. The
// period is 90 days, or 60 or 30 where the view is too narrow for a bar a
// day. The pointer on a bar shows a card of its day: the date, the uptime
// and the day's incidents with how long they lasted.
//
//	uptime.View(c, app.services, time.Now())
package uptime

import (
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/egoist/mygo/ui"
)

// Status is how a service is doing now.
type Status uint8

const (
	Operational Status = iota
	Degraded
	Outage
)

var names = [...]string{"Operational", "Degraded", "Outage"}

// Incident is what went wrong with a service on a day, and for how long.
type Incident struct {
	Title    string
	Duration time.Duration
}

// Day is a day of a service: the share of it the service was up, from 0
// to 1, and its incidents.
type Day struct {
	Uptime    float64
	Incidents []Incident
}

// Service is a service of the page: its name, how it is doing now, and
// its days, oldest first, the last of them the day the view ends. The
// days before the first have no data, as before the service was watched.
type Service struct {
	Name   string
	Status Status
	Days   []Day
}

// View creates the page of services over the days to the day of end. Give
// it a width.
func View(c *ui.Context, services []Service, end time.Time) ui.Element {
	t := c.Theme()
	pad := t.Space(4)
	page := ui.Column(c).Gap(t.Space(4))
	// The bars read the width of the last frame, the list's less its
	// padding and border: the page builds again when that changes the
	// days that fit.
	n := period(page.LayoutBounds().W-2*pad-2, t)
	page.Draw(func(_ *ui.Painter, r ui.Rect) {
		if period(r.W-2*pad-2, t) != n {
			c.Invalidate()
		}
	})
	page.Children(func() {
		banner(c, services)
		ui.Column(c).Gap(1).Dividers(1, t.Border).Radius(t.Space(2.5)).Border(1, t.Border).Children(func() {
			for i, s := range services {
				row(c.Key(i), s, n, end, pad)
			}
		})
		ui.Row(c).Padding(0, pad).Children(func() {
			ui.Text(c, fmt.Sprintf("%d days ago", n)).FontSize(t.FontSize - 2).TextColor(t.TextMuted).Grow(1)
			ui.Text(c, "Today").FontSize(t.FontSize - 2).TextColor(t.TextMuted)
		})
	})
	return page
}

// period returns the days that show in w, the width of the bars: 90 where
// each bar gets t.Space(1.5), else 60 or 30. With no width yet, as in the
// first frame, it is 90.
func period(w float32, t *ui.Theme) int {
	slot := t.Space(1.5)
	switch {
	case w <= 0 || w >= 90*slot:
		return 90
	case w >= 60*slot:
		return 60
	}
	return 30
}

// row shows service s over its last n days to end, padded by pad on its
// sides: its name, status and uptime, and its bars.
func row(c *ui.Context, s Service, n int, end time.Time, pad float32) {
	t := c.Theme()
	days := make([]*Day, n)
	sum, known := 0.0, 0
	for k := range days {
		if i := len(s.Days) - n + k; i >= 0 {
			days[k] = &s.Days[i]
			sum += s.Days[i].Uptime
			known++
		}
	}
	total := "No data"
	if known > 0 {
		total = percent(sum/float64(known)) + " uptime"
	}
	ui.Column(c).Gap(t.Space(2)).Padding(t.Space(3), pad).Children(func() {
		ui.Row(c).Gap(t.Space(4)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, s.Name).Grow(1).Shrink(1).SingleLine()
			ui.Text(c, total).FontSize(t.FontSize - 1).TextColor(t.TextMuted).FontFeatures("tnum")
			ui.Row(c).Gap(t.Space(1.5)).AlignItems(ui.Center).Children(func() {
				icon(c, s.Status, t.Space(3.5))
				ui.Text(c, names[s.Status]).FontSize(t.FontSize - 1)
			})
		})
		h := t.Space(7)
		bars := ui.Box(c).Height(h).Role(ui.RoleImage).
			Label(fmt.Sprintf("%s, %s over the last %d days", s.Name, total, n))
		slot := bars.LayoutBounds().W / float32(n)
		hot := -1
		if x, _, ok := bars.PointerPosition(); ok && slot > 0 {
			hot = max(0, min(n-1, int(x/slot)))
		}
		bars.Draw(func(p *ui.Painter, r ui.Rect) {
			slot := r.W / float32(n)
			gap := min(slot*0.3, t.Space(0.75))
			for k, d := range days {
				color := shade(d, t)
				if hot >= 0 && k != hot {
					color = color.Alpha(0.45)
				}
				p.Fill(ui.Rect{X: r.X + float32(k)*slot + gap/2, Y: r.Y, W: slot - gap, H: r.H}, color, min(2, (slot-gap)/2))
			}
		})
		if hot >= 0 {
			bars.Children(func() {
				anchor := ui.Box(c).Absolute().PassThrough().Left(float32(hot)*slot).Top(0).Size(slot, h)
				card(c, anchor, days[hot], end.AddDate(0, 0, hot-(n-1)))
			})
		}
	})
}

// card shows by anchor the card of day d on date, d nil with no data.
func card(c *ui.Context, anchor ui.Element, d *Day, date time.Time) {
	t := c.Theme()
	ui.Overlay(c, func() {
		// The card and all in it let the pointer through to the bars under
		// it, which would otherwise lose the hover and the card.
		tip := ui.Column(c).Absolute().PassThrough().Role(ui.RoleTooltip).Width(t.Space(68)).
			AttachTo(anchor, ui.AnchorTop, ui.AnchorBottom).Margin(0, 0, t.Space(1.5), 0).
			Padding(t.Space(3)).Gap(t.Space(2)).Radius(t.Space(2)).
			Background(t.Background).Border(1, t.Border).Shadow(0, 4, 16, 0, ui.RGBA(0, 0, 0, 0.18))
		tip.Children(func() {
			ui.Row(c).PassThrough().Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
				ui.Text(c, date.Format("Mon, Jan 2, 2006")).PassThrough().Bold().Grow(1)
				ui.Box(c).PassThrough().Size(t.Space(2.5), t.Space(2.5)).Radius(2).Background(shade(d, t))
				up := "No data"
				if d != nil {
					up = percent(d.Uptime)
				}
				ui.Text(c, up).PassThrough().FontFeatures("tnum")
			})
			if d == nil {
				return
			}
			if len(d.Incidents) == 0 {
				ui.Text(c, "No incidents").PassThrough().FontSize(t.FontSize - 1).TextColor(t.TextMuted)
				return
			}
			ui.Divider(c).PassThrough()
			for j, in := range d.Incidents {
				ui.Row(c.Key(j)).PassThrough().Gap(t.Space(3)).AlignItems(ui.Start).Children(func() {
					ui.Text(c, in.Title).PassThrough().FontSize(t.FontSize - 1).Grow(1).Shrink(1)
					ui.Text(c, length(in.Duration)).PassThrough().FontSize(t.FontSize - 1).
						TextColor(t.TextMuted).FontFeatures("tnum")
				})
			}
		})
	})
}

// banner shows how the services are as a whole, by the worst of them:
// partial when some are out, major when all are.
func banner(c *ui.Context, services []Service) {
	t := c.Theme()
	worst, out := Operational, 0
	for _, s := range services {
		worst = max(worst, s.Status)
		if s.Status == Outage {
			out++
		}
	}
	title := "All systems operational"
	switch {
	case worst == Degraded:
		title = "Degraded performance"
	case out > 0 && out == len(services):
		title = "Major outage"
	case out > 0:
		title = "Partial outage"
	}
	color := tint(worst, t)
	ui.Row(c).Gap(t.Space(3)).Padding(t.Space(4)).Radius(t.Space(2.5)).AlignItems(ui.Center).
		Background(color.Alpha(0.12)).Border(1, color.Alpha(0.4)).Role(ui.RoleStatus).Label(title).
		Children(func() {
			icon(c, worst, t.Space(6))
			ui.Text(c, title).Bold().FontSize(t.FontSize + 2)
		})
}

// tint returns the color of status.
func tint(s Status, t *ui.Theme) ui.Color {
	switch s {
	case Degraded:
		return t.Warning
	case Outage:
		return t.Danger
	}
	return t.Success
}

// shade returns the color of the bar of day d, muted when d is nil, for
// no data: green from 99.9% up, then lime, amber from 95%, and red.
func shade(d *Day, t *ui.Theme) ui.Color {
	switch {
	case d == nil:
		return t.Text.Alpha(0.1)
	case d.Uptime >= 0.999:
		return t.Success
	case d.Uptime >= 0.99:
		return ui.Oklch(0.77, 0.19, 128)
	case d.Uptime >= 0.95:
		return t.Warning
	}
	return t.Danger
}

// icon shows status in a circle of its color: a check, a bang or a cross.
func icon(c *ui.Context, status Status, size float32) {
	t := c.Theme()
	color := tint(status, t)
	ui.Box(c).Size(size, size).Shrink(0).Draw(func(p *ui.Painter, r ui.Rect) {
		cx, cy, rad := r.X+r.W/2, r.Y+r.H/2, r.W/2
		var circle ui.Path
		circle.Circle(cx, cy, rad)
		p.FillPath(&circle, color)
		white, w := ui.RGB(255, 255, 255), max(1.5, size/9)
		switch status {
		case Operational:
			p.StrokePolyline([][2]float32{{cx - rad*0.42, cy + rad*0.02}, {cx - rad*0.1, cy + rad*0.34}, {cx + rad*0.45, cy - rad*0.32}}, w, white)
		case Degraded:
			p.StrokePolyline([][2]float32{{cx, cy - rad*0.5}, {cx, cy + rad*0.1}}, w, white)
			var dot ui.Path
			dot.Circle(cx, cy+rad*0.42, w*0.6)
			p.FillPath(&dot, white)
		case Outage:
			d := rad * 0.36
			p.StrokePolyline([][2]float32{{cx - d, cy - d}, {cx + d, cy + d}}, w, white)
			p.StrokePolyline([][2]float32{{cx + d, cy - d}, {cx - d, cy + d}}, w, white)
		}
	})
}

// percent writes f, from 0 to 1, as "99.98%", rounded down so that a day
// down for a moment does not read 100%.
func percent(f float64) string {
	if f >= 1 {
		return "100%"
	}
	return strconv.FormatFloat(math.Floor(max(0, f)*10000)/100, 'f', 2, 64) + "%"
}

// length writes d as "1h 12m", "23m" or "40s".
func length(d time.Duration) string {
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	h, m := int(d.Hours()), int(d.Minutes())%60
	if m == 0 {
		return fmt.Sprintf("%dh", h)
	}
	return fmt.Sprintf("%dh %dm", h, m)
}
