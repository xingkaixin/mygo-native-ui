// Package timesheet shows a week of hours worked, as time trackers do: a
// row for each project and task, a column for each day with today's
// marked and the weekend's shaded, each row's and each day's total, and
// below them the week's total against the hours due, as a bar. Clicking a
// day's hours opens them for typing, as 1:30, 1.5 or 90m; Enter or
// clicking elsewhere keeps them, and Enter moves on to the row below.
//
//	timesheet.View(c, &app.sheet, monday, &app.rows, 40)
package timesheet

import (
	"cmp"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
)

// Row is a project's task and the hours worked on it each day of the
// week, Monday first.
type Row struct {
	Project, Task string
	Hours         [7]float64
}

// State is what a timesheet keeps: the hours being typed. Keep it in the
// app's state, one for each timesheet.
type State struct {
	// editing is the row times seven plus the day being typed, plus one,
	// so that zero is none.
	editing int
	text    string
	focused bool
	fresh   bool
}

// View creates the timesheet of the week starting on monday, of rows,
// whose hours typing changes, against due hours a week.
func View(c *ui.Context, s *State, monday time.Time, rows *[]Row, due float64) ui.Element {
	t := c.Theme()
	commit := func() {
		if s.editing == 0 {
			return
		}
		r, d := (s.editing-1)/7, (s.editing-1)%7
		if h, ok := Parse(s.text); ok && r < len(*rows) {
			(*rows)[r].Hours[d] = h
		}
		s.editing, s.focused = 0, false
	}
	var days [7]float64
	week := 0.0
	for _, r := range *rows {
		for d, h := range r.Hours {
			days[d] += h
			week += h
		}
	}
	today := -1
	if now := c.Now(); !now.Before(monday) && now.Before(monday.AddDate(0, 0, 7)) {
		today = int(now.Sub(monday).Hours() / 24)
	}
	nameW, cellW := t.Space(56), t.Space(18)
	cellH := t.Space(10)
	shade := func(d int, e ui.Element) ui.Element {
		switch {
		case d == today:
			e.Background(t.Accent.Alpha(0.08))
		case d >= 5:
			e.Background(t.Surface)
		}
		return e
	}
	root := ui.Column(c).Gap(t.Space(4)).AlignItems(ui.Start).Role(ui.RoleGroup).Label("Timesheet for the week of " + monday.Format("2 January"))
	root.Children(func() {
		ui.Column(c).Border(1, t.Border).Radius(t.Radius+2).Clip().Dividers(1, t.Border).Children(func() {
			ui.Row(c).Height(cellH + t.Space(2)).AlignItems(ui.Stretch).Children(func() {
				ui.Text(c, "Project").Width(nameW).PaddingX(t.Space(3)).AlignSelf(ui.Center).FontWeight(600).TextColor(t.TextMuted)
				for d := range 7 {
					day := monday.AddDate(0, 0, d)
					col := ui.Column(c.Key(d)).Width(cellW).AlignItems(ui.Center).Justify(ui.Center)
					shade(d, col).Children(func() {
						name := ui.Text(c, day.Format("Mon")).FontSize(t.FontSize - 1).FontWeight(600)
						if d == today {
							name.TextColor(t.Accent)
						}
						ui.Text(c, day.Format("2 Jan")).FontSize(t.FontSize - 2).TextColor(t.TextMuted)
					})
				}
				ui.Text(c, "Total").Width(cellW).AlignSelf(ui.Center).TextAlign(ui.Center).FontWeight(600).TextColor(t.TextMuted)
			})
			for ri, r := range *rows {
				ui.Row(c.Key(ri)).Height(cellH + t.Space(2)).AlignItems(ui.Stretch).Children(func() {
					ui.Column(c).Width(nameW).PaddingX(t.Space(3)).Justify(ui.Center).Children(func() {
						ui.Text(c, r.Project).FontWeight(600).NoWrap().Ellipsis("…")
						ui.Text(c, r.Task).FontSize(t.FontSize - 2).TextColor(t.TextMuted).NoWrap().Ellipsis("…")
					})
					total := 0.0
					for d, h := range r.Hours {
						total += h
						idx := ri*7 + d + 1
						cell := ui.Box(c.Key(d)).Width(cellW).AlignItems(ui.Center).Justify(ui.Center)
						shade(d, cell)
						if s.editing == idx {
							cell.Children(func() {
								in := ui.TextInput(c, &s.text).Width(cellW - t.Space(3)).TextAlign(ui.Center).FontFeatures("tnum").AutoFocus()
								// Typing replaces the hours rather than adding to them.
								if s.fresh {
									in.SetTextSelection(0, utf8.RuneCountInString(s.text))
									s.fresh = false
								}
								if in.Focused() {
									s.focused = true
								} else if s.focused {
									commit()
								}
								if in.Submitted() {
									commit()
									if ri+1 < len(*rows) {
										s.open(idx+7, (*rows)[ri+1].Hours[d])
									}
								}
							})
							continue
						}
						cell.Cursor(ui.CursorText).Children(func() {
							label, col := Format(h), t.Text
							if h == 0 {
								label, col = "·", t.TextMuted.Alpha(0.6)
							}
							ui.Text(c, label).FontFeatures("tnum").TextColor(col)
						})
						if cell.Hovered() {
							cell.Background(t.Accent.Alpha(0.12))
						}
						if cell.Clicked() {
							commit()
							s.open(idx, h)
						}
					}
					ui.Text(c, Format(total)).Width(cellW).AlignSelf(ui.Center).TextAlign(ui.Center).FontWeight(600).FontFeatures("tnum")
				})
			}
			ui.Row(c).Height(cellH).AlignItems(ui.Stretch).Background(t.Surface).Children(func() {
				ui.Text(c, "Total").Width(nameW).PaddingX(t.Space(3)).AlignSelf(ui.Center).FontWeight(600)
				for d, h := range days {
					txt := ui.Text(c.Key(d), Format(h)).Width(cellW).AlignSelf(ui.Center).TextAlign(ui.Center).FontWeight(600).FontFeatures("tnum")
					switch {
					case h == 0:
						txt.TextColor(t.TextMuted)
					case h > 10:
						txt.TextColor(ui.RGB(224, 72, 72))
					}
				}
				ui.Text(c, Format(week)).Width(cellW).AlignSelf(ui.Center).TextAlign(ui.Center).FontWeight(700).FontFeatures("tnum")
			})
		})
		ui.Column(c).Gap(t.Space(2)).Width(nameW + 8*cellW).Children(func() {
			ui.Row(c).Children(func() {
				ui.Text(c, fmt.Sprintf("%s of %s", Format(week), Format(due))).FontWeight(600).FontFeatures("tnum")
				ui.Box(c).Grow(1)
				left := due - week
				msg := fmt.Sprintf("%s to go", Format(left))
				col := t.TextMuted
				if left <= 0 {
					msg, col = fmt.Sprintf("%s over", Format(-left)), ui.RGB(224, 72, 72)
					if left == 0 {
						msg, col = "Done", ui.RGB(52, 168, 83)
					}
				}
				ui.Text(c, msg).TextColor(col).FontFeatures("tnum")
			})
			bar := ui.Box(c).Height(t.Space(2)).Radius(t.Space(1)).Background(t.Surface)
			bar.Draw(func(p *ui.Painter, r ui.Rect) {
				f := float32(min(1, week/max(due, 1e-9)))
				col := t.Accent
				if week > due {
					col = ui.RGB(224, 72, 72)
				}
				p.Fill(ui.Rect{X: r.X, Y: r.Y, W: r.W * f, H: r.H}, col, r.H/2)
			})
		})
	})
	return root
}

func (s *State) open(idx int, h float64) {
	s.editing, s.text, s.fresh, s.focused = idx, "", true, false
	if h != 0 {
		s.text = Format(h)
	}
}

// Format writes hours as hours and minutes, as 7:30.
func Format(h float64) string {
	m := int(math.Round(h * 60))
	return fmt.Sprintf("%d:%02d", m/60, m%60)
}

// Parse reads hours written as 1:30, 1.5, 90m or 2h, or nothing for none.
func Parse(s string) (float64, bool) {
	s = strings.TrimSpace(strings.ToLower(s))
	switch {
	case s == "" || s == "·":
		return 0, true
	case strings.Contains(s, ":"):
		hs, ms, _ := strings.Cut(s, ":")
		h, err1 := strconv.Atoi(cmp.Or(hs, "0"))
		m, err2 := strconv.Atoi(ms)
		if err1 != nil || err2 != nil || m >= 60 || h < 0 || m < 0 {
			return 0, false
		}
		return float64(h) + float64(m)/60, true
	case strings.HasSuffix(s, "m"):
		m, err := strconv.ParseFloat(strings.TrimSuffix(s, "m"), 64)
		return m / 60, err == nil && m >= 0
	}
	h, err := strconv.ParseFloat(strings.TrimSuffix(s, "h"), 64)
	return h, err == nil && h >= 0 && h <= 24
}
