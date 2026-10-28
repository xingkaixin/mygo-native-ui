// Duration shows the estimates of a sprint's issues, each in a duration
// field, and their total against the sprint's capacity. Type an estimate
// as you would write it, or press Up and Down in a field.
//
//	go run ./examples/duration
package main

import (
	"log"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/duration"
)

type issue struct {
	key, title string
	field      duration.State
	estimate   time.Duration
}

type app struct {
	issues []*issue
}

const capacity = 3 * 24 * time.Hour

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	var total time.Duration
	for _, is := range a.issues {
		total += is.estimate
	}
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(8)).Gap(t.Space(4)).Children(func() {
		ui.Text(c, "Sprint 42").FontSize(t.FontSize + 8).Bold()
		ui.Column(c).Children(func() {
			for i, is := range a.issues {
				ui.Row(c.Key(i)).Gap(t.Space(3)).AlignItems(ui.Center).Padding(t.Space(2), 0).
					BorderWidth(0, 0, 1, 0).BorderColor(t.Border).Children(func() {
					ui.Text(c, is.key).FontSize(t.FontSize - 1).TextColor(t.TextMuted).Width(t.Space(16))
					ui.Text(c, is.title).Grow(1).SingleLine()
					duration.Field(c, &is.field, &is.estimate).Width(t.Space(36))
				})
			}
		})
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "Total").Bold().Grow(1)
			if total > capacity {
				ui.Text(c, duration.Format(total-capacity)+" over").FontSize(t.FontSize - 1).TextColor(t.Danger)
			} else {
				ui.Text(c, duration.Format(capacity-total)+" left").FontSize(t.FontSize - 1).TextColor(t.TextMuted)
			}
			ui.Text(c, duration.Format(total)).Bold().FontFeatures("tnum").Width(t.Space(36)).Padding(0, t.Space(2.5))
		})
		ui.Box(c).Height(6).Radius(3).Background(t.Text.Alpha(0.08)).Draw(func(p *ui.Painter, r ui.Rect) {
			share := min(float32(total)/float32(capacity), 1)
			color := t.Accent
			if total > capacity {
				color = t.Danger
			}
			p.Fill(ui.Rect{X: r.X, Y: r.Y, W: r.W * share, H: r.H}, color, 3)
		})
	})
}

func main() {
	a := &app{issues: []*issue{
		{key: "WEB-311", title: "Sign in with passkeys", estimate: 14 * time.Hour},
		{key: "WEB-318", title: "Rate limit the export endpoint", estimate: 3*time.Hour + 30*time.Minute},
		{key: "WEB-322", title: "Empty state for new workspaces", estimate: 90 * time.Minute},
		{key: "WEB-325", title: "Fix flaky billing webhook test", estimate: 45 * time.Minute},
		{key: "WEB-327", title: "Migrate avatars to the new bucket"},
	}}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Duration",
			Width:   720,
			Height:  480,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
