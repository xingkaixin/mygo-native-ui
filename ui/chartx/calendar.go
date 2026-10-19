package chartx

import (
	"slices"
	"sort"
	"time"

	"github.com/egoist/mygo/ui"
)

// Calendar builds a calendar of days as GitHub's of contributions: a
// column for each week, Sunday at the top, the months above the weeks
// they start, Mon, Wed and Fri by their rows, and each day a square in the
// color of its level, with a legend of the levels, from Less to More,
// below. values[k] is the value of the day k days after first.
//
// Days of 0 or less take colors[0], and the others the rest of colors by
// the quantiles of the values above 0, as on GitHub: with four colors
// more, each takes a quarter of the days, so a few days of many leave the
// others no paler. Nil colors are GitHub's greens, in light and dark
// windows. A day's tooltip, which assistive technology reads too, is
// describe's of it, "3 contributions on Mar 4":
//
//	chartx.Calendar(c, jan1, app.counts, nil, func(day time.Time, v float64) string {
//		return fmt.Sprintf("%v contributions on %s", v, day.Format("Jan 2"))
//	})
func Calendar(c *ui.Context, first time.Time, values []float64, colors []ui.Color, describe func(day time.Time, v float64) string) ui.Element {
	t := c.Theme()
	if colors == nil {
		colors = githubGreens
		if t.Dark {
			colors = githubGreensDark
		}
	}
	level := levels(values, len(colors)-1)
	const cell, gap, labelH = 11, 3, 16
	lead := int(first.Weekday())
	weeks := (lead + len(values) + 6) / 7
	label := func(s string) ui.Element {
		return ui.Text(c, s).FontSize(11).TextColor(t.TextMuted).NoWrap()
	}

	return ui.Column(c).Gap(t.Space(2)).Children(func() {
		ui.Row(c).Gap(gap).Children(func() {
			ui.Column(c).Gap(gap).Padding(labelH, t.Space(1), 0, 0).Children(func() {
				for d := range 7 {
					row := ui.Box(c).Height(cell).AlignItems(ui.Center)
					if d%2 == 1 {
						row.Children(func() { label(time.Weekday(d).String()[:3]) })
					}
				}
			})
			heads := monthHeads(first, lead, weeks, len(values))
			for w := range weeks {
				ui.Column(c.Key(w)).Gap(gap).Children(func() {
					head := ui.Box(c).Size(cell, labelH-gap)
					if heads[w] != "" {
						head.Children(func() { label(heads[w]).Absolute() })
					}
					for d := range 7 {
						k := w*7 + d - lead
						if k < 0 || k >= len(values) {
							ui.Box(c).Size(cell, cell)
							continue
						}
						day := ui.Box(c).Size(cell, cell).Radius(2).Background(colors[level(values[k])])
						if describe != nil {
							day.Tooltip(describe(first.AddDate(0, 0, k), values[k]))
						}
						if day.Hovered() {
							day.Border(1, t.Text)
						}
					}
				})
			}
		})
		ui.Row(c).Gap(gap).AlignItems(ui.Center).AlignSelf(ui.End).Children(func() {
			label("Less").Margin(0, t.Space(1), 0, 0)
			for _, col := range colors {
				ui.Box(c).Size(cell, cell).Radius(2).Background(col)
			}
			label("More").Margin(0, 0, 0, t.Space(1))
		})
	})
}

// monthHeads returns the names of months over the weeks of a calendar of
// n days: over the first week of each, that of its first day, unless the
// next starts within two weeks, too close for both, and takes the room.
func monthHeads(first time.Time, lead, weeks, n int) []string {
	heads := make([]string, weeks)
	prev, at := time.Month(0), -1
	for w := range weeks {
		k := max(0, w*7-lead)
		if k >= n {
			break
		}
		m := first.AddDate(0, 0, k).Month()
		if m == prev {
			continue
		}
		if at >= 0 && w-at < 3 {
			heads[at] = ""
		}
		heads[w], prev, at = m.String()[:3], m, w
	}
	return heads
}

// levels returns the level of a value among values, from 0 for values of
// 0 or less to n for the greatest, the values above 0 split evenly among
// levels 1 to n.
func levels(values []float64, n int) func(float64) int {
	var above []float64
	for _, v := range values {
		if v > 0 {
			above = append(above, v)
		}
	}
	slices.Sort(above)
	// Level q holds the values up to the q-th of n quantiles.
	bounds := make([]float64, max(0, n-1))
	for q := range bounds {
		if len(above) > 0 {
			bounds[q] = above[max(0, len(above)*(q+1)/n-1)]
		}
	}
	return func(v float64) int {
		if !(v > 0) || n < 1 {
			return 0
		}
		return 1 + sort.SearchFloat64s(bounds, v)
	}
}

var (
	githubGreens     = []ui.Color{ui.Hex("#ebedf0"), ui.Hex("#9be9a8"), ui.Hex("#40c463"), ui.Hex("#30a14e"), ui.Hex("#216e39")}
	githubGreensDark = []ui.Color{ui.Hex("#2d333b"), ui.Hex("#0e4429"), ui.Hex("#006d32"), ui.Hex("#26a641"), ui.Hex("#39d353")}
)
