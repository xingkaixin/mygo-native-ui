// Package plan shows the steps an agent plans to take and how far it has
// got, as coding agents show their to-do lists: a header with how many
// steps are done and a bar of it, and the steps in order, each with an
// icon of how it goes, the one under way shimmering, and those done
// folded away, or struck through.
//
//	plan.View(c, "Fix the failing build", app.steps, &app.showDone)
package plan

import (
	"fmt"

	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/shimmer"
	"github.com/xingkaixin/mygo-native-ui/ui/trace"
)

// Step is a step of a plan, its title and how it goes.
type Step struct {
	Title  string
	Status trace.Status
}

// View creates a card of the plan title, of steps: those done show while
// *showDone is true, which a row above the steps toggles.
func View(c *ui.Context, title string, steps []Step, showDone *bool) ui.Element {
	t := c.Theme()
	done := 0
	for _, s := range steps {
		if s.Status == trace.Done {
			done++
		}
	}
	return ui.Column(c).Radius(t.Space(2.5)).Border(1, t.Border).Clip().Role(ui.RoleGroup).Label(title).Children(func() {
		ui.Row(c).Gap(t.Space(2)).Padding(t.Space(2.5), t.Space(3)).Children(func() {
			ui.Text(c, title).Bold().SingleLine().Grow(1).Shrink(1)
			ui.Text(c, fmt.Sprintf("%d of %d", done, len(steps))).TextColor(t.TextMuted).FontFeatures("tnum")
		})
		f := 0.0
		if len(steps) > 0 {
			f = float64(done) / float64(len(steps))
		}
		ui.Box(c).Height(2).Background(t.Text.Alpha(0.06)).Children(func() {
			ui.Box(c).Height(2).WidthPercent(float32(f * 100)).Background(t.Success)
		})
		ui.Column(c).Padding(t.Space(1.5), t.Space(3), t.Space(2)).Children(func() {
			if done > 0 {
				label := fmt.Sprintf("%d completed", done)
				if *showDone {
					label = "Hide completed"
				}
				toggle := ui.ButtonBase(c).Justify(ui.Start).Gap(t.Space(2)).Padding(t.Space(1), 0).Expanded(*showDone).Label(label)
				if toggle.Clicked() {
					*showDone = !*showDone
				}
				toggle.Children(func() {
					trace.Icon(c, trace.Done, t.FontSize+2)
					ui.Text(c, label).TextColor(t.TextMuted)
				})
			}
			for i, s := range steps {
				if s.Status == trace.Done && !*showDone {
					continue
				}
				ui.Row(c.Key(i)).Gap(t.Space(2)).Padding(t.Space(1), 0).AlignItems(ui.Center).Children(func() {
					trace.Icon(c, s.Status, t.FontSize+2)
					switch s.Status {
					case trace.Running:
						shimmer.Text(c, s.Title)
					case trace.Done:
						ui.Text(c, s.Title).TextColor(t.TextMuted).Strikethrough().Shrink(1)
					default:
						ui.Text(c, s.Title).Shrink(1)
					}
				})
			}
		})
	})
}
