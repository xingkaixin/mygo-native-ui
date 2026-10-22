// Quadrant prioritizes a made-up product backlog by impact against
// effort. Drag an item, or focus it with Tab and press the arrows, Shift
// for bigger steps, to move it: the list beside the chart files it under
// its quadrant as it moves, and the Mermaid text below the list is
// written again. Edit the text to change the chart, with what is wrong in
// it listed under it by line.
//
//	go run ./examples/quadrant
package main

import (
	"cmp"
	"log"
	"slices"
	"strconv"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/quadrant"
)

const backlog = `quadrantChart
    title Backlog by impact and effort
    x-axis Low effort --> High effort
    y-axis Low impact --> High impact
    quadrant-1 Big bets
    quadrant-2 Quick wins
    quadrant-3 Fill-ins
    quadrant-4 Money pits
    Onboarding checklist: [0.28, 0.82]
    Slack alerts: [0.12, 0.70]
    Search filters: [0.36, 0.62]
    CSV export: [0.16, 0.44]
    Keyboard shortcuts: [0.22, 0.28]
    Dark mode: [0.42, 0.36]
    SSO login: [0.68, 0.86]
    Mobile app: [0.90, 0.92]
    Usage dashboard: [0.56, 0.66]
    Offline sync: [0.86, 0.58]
    Rewrite billing: [0.84, 0.30]
    Custom themes: [0.62, 0.16]
    Audit log: [0.66, 0.44]
    Security patch: [0.30, 0.95] color: #e5484d
`

// order is the quadrants as the list files them, from the most worth
// doing: quick wins, big bets, fill-ins and money pits.
var order = []int{1, 0, 2, 3}

type app struct {
	source string
	shown  bool
	chart  quadrant.Chart
	points []quadrant.Point
	errs   []quadrant.Error
	matrix quadrant.State
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Row(c).Fill().Background(t.Background).Children(func() {
		ui.Column(c).Grow(1).FillHeight().Padding(t.Space(5)).Children(func() {
			if quadrant.View(c, &a.matrix, a.chart, &a.points).Grow(1).AlignSelf(ui.Stretch).Changed() {
				a.source = quadrant.Format(a.chart, a.points)
			}
		})
		ui.Divider(c)
		ui.Column(c).Width(t.Space(92)).FillHeight().Children(func() {
			ui.Scroll(c).Grow(1).Children(func() {
				ui.Column(c).Gap(t.Space(4)).Padding(t.Space(4)).Children(func() {
					for _, q := range order {
						a.section(c.Key(q), q)
					}
				})
			})
			ui.Divider(c)
			src := ui.TextAreaBase(c, &a.source).Font("monospace").FontSize(t.FontSize-1).
				Padding(t.Space(3), t.Space(4)).Label("Mermaid").Height(t.Space(78))
			if !a.shown {
				// A text area starts with the caret after its text: show the
				// chart's text from the top.
				src.SetTextSelection(0, 0)
				a.shown = true
			}
			if src.Changed() {
				a.chart, a.points, a.errs = quadrant.Parse(a.source)
			}
			if len(a.errs) > 0 {
				ui.Divider(c)
				ui.Column(c).Gap(t.Space(1.5)).Padding(t.Space(3), t.Space(4)).Children(func() {
					for i, e := range a.errs {
						ui.Row(c.Key(i)).Gap(t.Space(2)).Children(func() {
							ui.Text(c, "Line "+strconv.Itoa(e.Line)).FontSize(t.FontSize - 1).FontFeatures("tnum").
								TextColor(t.TextMuted).Shrink(0)
							ui.Text(c, e.Msg).FontSize(t.FontSize - 1).TextColor(t.Danger).Shrink(1)
						})
					}
				})
			}
		})
	})
}

// section lists the items in quadrant q, the most impact first, under
// its title and their count.
func (a *app) section(c *ui.Context, q int) {
	t := c.Theme()
	var in []quadrant.Point
	for _, pt := range a.points {
		if pt.Quadrant() == q {
			in = append(in, pt)
		}
	}
	slices.SortStableFunc(in, func(x, y quadrant.Point) int { return cmp.Compare(y.Y, x.Y) })
	title := a.chart.Quadrants[q]
	if title == "" {
		title = "Quadrant " + strconv.Itoa(q+1)
	}
	ui.Column(c).Gap(t.Space(1.5)).Children(func() {
		ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
			ui.Box(c).Size(10, 10).Radius(5).Background(quadrant.Color(t, q))
			ui.Text(c, title).Bold().Grow(1).Shrink(1)
			ui.Text(c, strconv.Itoa(len(in))).TextColor(t.TextMuted).FontFeatures("tnum")
		})
		if len(in) == 0 {
			ui.Text(c, "Nothing here").FontSize(t.FontSize - 1).TextColor(t.TextMuted).PaddingStart(t.Space(4.5))
		}
		for i, pt := range in {
			ui.Row(c.Key(i)).Gap(t.Space(2)).PaddingStart(t.Space(4.5)).Children(func() {
				ui.Text(c, pt.Name).FontSize(t.FontSize - 1).Grow(1).Shrink(1).SingleLine()
				ui.Text(c, "impact "+strconv.FormatFloat(pt.Y, 'f', 2, 64)+"  effort "+strconv.FormatFloat(pt.X, 'f', 2, 64)).
					FontSize(t.FontSize - 1).TextColor(t.TextMuted).FontFeatures("tnum")
			})
		}
	})
}

func main() {
	a := &app{source: backlog}
	a.chart, a.points, a.errs = quadrant.Parse(a.source)
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Quadrant",
			Width:   1240,
			Height:  800,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
