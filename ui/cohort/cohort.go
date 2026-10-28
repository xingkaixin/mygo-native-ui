// Package cohort shows a retention table, as product analytics tools do:
// a row for each cohort of users who started in the same period, its
// size, and the share of it still active in each period after, deeper in
// the accent color for more, against the highest share after the first
// period; the newest cohorts have fewer periods, so
// the table is a triangle. A row above them weighs all the cohorts
// together.
//
//	if ev := cohort.View(c, &app.table, app.cohorts, "Week"); ev.Row > -2 {
//		app.show(ev.Row, ev.Period)
//	}
//
// The pointer over a share lights its row and column and shows how many
// users it is; a click on one reports it.
package cohort

import (
	"fmt"
	"strconv"

	"github.com/egoist/mygo/ui"
)

// Cohort is a group of users who started in one period: its name, as the
// period's, how many there were, and how many were active in each period
// from the first, Kept[0] usually being Size.
type Cohort struct {
	Name string
	Size int
	Kept []int
}

// State is what a table keeps: the share under the pointer. Keep it in
// the app's state, one for each table.
type State struct {
	lit [2]int
	set bool
}

// Events is a table, and the share clicked this frame: its cohort, -1 for
// the row of all, and its period, or Row -2 for none.
type Events struct {
	ui.Element
	Row, Period int
}

// View creates the table of cohorts, the oldest first, naming the
// periods with period, as "Week 0", "Week 1".
func View(c *ui.Context, s *State, cohorts []Cohort, period string) Events {
	t := c.Theme()
	ev := Events{Row: -2, Period: -1}
	periods := 0
	for _, co := range cohorts {
		periods = max(periods, len(co.Kept))
	}
	all := Cohort{Name: "All users", Kept: make([]int, periods)}
	sizes := make([]int, periods)
	for _, co := range cohorts {
		all.Size += co.Size
		for p, k := range co.Kept {
			all.Kept[p] += k
			sizes[p] += co.Size
		}
	}
	// Shares after the first period are colored against the highest of
	// them, as all of the first are 100% and would wash the rest out.
	top := 0.0
	for _, co := range cohorts {
		for p, k := range co.Kept {
			if p > 0 && co.Size > 0 {
				top = max(top, float64(k)/float64(co.Size))
			}
		}
	}
	if !s.set {
		s.lit, s.set = [2]int{-2, -1}, true
	}
	// The share under the pointer is known once the cells are built, so
	// the headers and the lines to it follow the last frame's.
	lit := s.lit
	hovered := [2]int{-2, -1}
	nameW, sizeW, cellW, rowH := t.Space(26), t.Space(18), t.Space(16), t.Space(8)
	ev.Element = ui.Column(c).Gap(t.Space(0.5)).Children(func() {
		ui.Row(c).Gap(t.Space(0.5)).Children(func() {
			ui.Text(c, "Cohort").Width(nameW).FontSize(t.FontSize - 1).TextColor(t.TextMuted)
			ui.Text(c, "Users").Width(sizeW).FontSize(t.FontSize - 1).TextColor(t.TextMuted).TextAlign(ui.End).PaddingEnd(t.Space(2))
			for p := range periods {
				h := ui.Text(c.Key(p), period+" "+strconv.Itoa(p)).Width(cellW).FontSize(t.FontSize - 1).TextAlign(ui.Center)
				h.TextColor(t.TextMuted)
				if lit[1] == p {
					h.TextColor(t.Text).Bold()
				}
			}
		})
		row := func(i int, co Cohort, sizes []int) {
			ui.Row(c.Key(i)).Gap(t.Space(0.5)).AlignItems(ui.Center).Children(func() {
				name := ui.Text(c, co.Name).Width(nameW).SingleLine()
				if i == -1 {
					name.Bold()
				}
				if lit[0] == i {
					name.TextColor(t.Accent)
				}
				ui.Text(c, group(co.Size)).Width(sizeW).TextAlign(ui.End).PaddingEnd(t.Space(2)).TextColor(t.TextMuted).FontFeatures("tnum")
				for p, k := range co.Kept {
					of := co.Size
					if sizes != nil {
						of = sizes[p]
					}
					share := 0.0
					if of > 0 {
						share = float64(k) / float64(of)
					}
					depth := share
					if p > 0 && top > 0 {
						depth = min(share/top, 1)
					}
					cell := ui.Text(c.Key(p), fmt.Sprintf("%.1f%%", share*100)).Size(cellW, rowH).TextAlign(ui.Center).
						Padding(t.Space(2), 0).FontSize(t.FontSize - 1).FontFeatures("tnum").Radius(2).
						Background(t.Accent.Alpha(float32(0.06 + 0.84*depth))).
						Tooltip(fmt.Sprintf("%s, %s %d: %s of %s users", co.Name, period, p, group(k), group(of)))
					if depth > 0.55 {
						cell.TextColor(t.AccentText)
					}
					if lit[0] == i && p < lit[1] || lit[1] == p && i < lit[0] {
						cell.Border(1, t.Text.Alpha(0.35))
					}
					if cell.Hovered() {
						hovered = [2]int{i, p}
						cell.Border(2, t.Text)
					}
					if cell.Clicked() {
						ev.Row, ev.Period = i, p
					}
				}
			})
		}
		row(-1, all, sizes)
		ui.Box(c).Height(t.Space(1))
		for i, co := range cohorts {
			row(i, co, nil)
		}
	})
	if s.lit != hovered {
		s.lit = hovered
		c.Invalidate()
	}
	return ev
}

// group writes n with commas between thousands.
func group(n int) string {
	s := strconv.Itoa(n)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}
