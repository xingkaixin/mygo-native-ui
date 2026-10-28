// Package problems is a problems panel, as code editors have under their
// editor: the diagnostics of a workspace grouped by file, each with its
// severity's mark, its message, the tool that found it and its code
// dimmed, and its line and column; each file with its count. Above them,
// a filter narrows them by text, and a toggle for each severity, with its
// count, hides or shows it.
//
//	ev := problems.View(c, &app.problems, app.diagnostics)
//	if ev.Opened >= 0 {
//		app.open(app.diagnostics[ev.Opened])
//	}
//
// Files list in order of their paths, and a file's problems by severity
// then line. A click on a file folds it. A click on a problem, or Enter
// on the one chosen with the arrows, reports it.
package problems

import (
	"cmp"
	"fmt"
	"path"
	"slices"
	"strings"

	"github.com/egoist/mygo/ui"
)

// Severity is how bad a problem is.
type Severity int

const (
	Error Severity = iota
	Warning
	Info
)

// Diagnostic is a problem found in a file: where, how bad, what, and the
// tool that found it with its code for it, as "go vet" and "printf".
type Diagnostic struct {
	Path      string
	Line, Col int
	Severity  Severity
	Message   string
	Source    string
	Code      string
}

// State is what a panel keeps: the filter, the severities hidden, the
// files folded, and the row chosen. Keep it in the app's state, one for
// each panel.
type State struct {
	filter string
	hidden [3]bool
	folded map[string]bool
	list   ui.ListState
	row    int
	setup  bool
}

// Events is a panel, and the index of the diagnostic opened this frame,
// or -1.
type Events struct {
	ui.Element
	Opened int
}

// row is a row of the list: a file's header, by the path, or a
// diagnostic, by its index.
type row struct {
	path string
	diag int
}

// View creates the panel of diags.
func View(c *ui.Context, s *State, diags []Diagnostic) Events {
	t := c.Theme()
	if !s.setup {
		s.setup, s.row, s.folded = true, -1, map[string]bool{}
		s.list.Selected = &s.row
	}
	view := ui.Column(c).Gap(t.Space(2))
	ev := Events{Element: view, Opened: -1}

	var counts [3]int
	for _, d := range diags {
		counts[d.Severity]++
	}
	needle := strings.ToLower(strings.TrimSpace(s.filter))
	var shown []int
	for i, d := range diags {
		if s.hidden[d.Severity] {
			continue
		}
		if needle != "" && !strings.Contains(strings.ToLower(d.Message+" "+d.Source+" "+d.Code+" "+d.Path), needle) {
			continue
		}
		shown = append(shown, i)
	}
	slices.SortStableFunc(shown, func(a, b int) int {
		da, db := diags[a], diags[b]
		return cmp.Or(cmp.Compare(da.Path, db.Path), cmp.Compare(da.Severity, db.Severity),
			cmp.Compare(da.Line, db.Line), cmp.Compare(da.Col, db.Col))
	})
	var rows []row
	perFile := map[string]int{}
	for _, i := range shown {
		p := diags[i].Path
		if perFile[p] == 0 {
			rows = append(rows, row{path: p, diag: -1})
		}
		perFile[p]++
		if !s.folded[p] {
			rows = append(rows, row{path: p, diag: i})
		}
	}
	s.row = min(s.row, len(rows)-1)
	s.list.Key = func(i int) any {
		if rows[i].diag < 0 {
			return rows[i].path
		}
		return rows[i].diag
	}
	s.list.Header = func(i int) bool { return rows[i].diag < 0 }

	view.Children(func() {
		ui.Row(c).Gap(t.Space(1)).AlignItems(ui.Center).Children(func() {
			ui.Row(c).Grow(1).AlignItems(ui.Center).Padding(t.Space(1), t.Space(2)).Radius(t.Space(1.5)).
				Border(1, t.Border).Background(t.Background).Children(func() {
				ui.TextInputBase(c, &s.filter).Grow(1).Placeholder("Filter (e.g. text, unused, src/)").Label("Filter").SingleLine()
			})
			for sev := Error; sev <= Info; sev++ {
				on := !s.hidden[sev]
				b := ui.ToggleBase(c.Key(int(sev)), &on).Gap(t.Space(1)).Padding(t.Space(1), t.Space(1.5)).Radius(t.Space(1)).
					AlignItems(ui.Center).Label(names[sev]).Tooltip(names[sev]).KeepFocus()
				s.hidden[sev] = !on
				if on {
					b.Background(t.Text.Alpha(0.07))
				} else if b.Hovered() {
					b.Background(t.Text.Alpha(0.04))
				}
				n := counts[sev]
				b.Children(func() {
					ui.Box(c).Size(14, 14).Draw(func(p *ui.Painter, r ui.Rect) { mark(p, r, sev, t, !on) })
					txt := ui.Text(c, fmt.Sprint(n)).FontSize(t.FontSize - 1).FontFeatures("tnum")
					if !on {
						txt.TextColor(t.TextMuted)
					}
				})
			}
		})
		if len(rows) == 0 {
			msg := "No problems have been detected in the workspace."
			if len(diags) > 0 {
				msg = "No problems match the filter."
			}
			ui.Text(c, msg).FontSize(t.FontSize-1).TextColor(t.TextMuted).Padding(t.Space(2), t.Space(1))
			return
		}
		list := ui.List(c, &s.list, len(rows), func(i int) {
			r := rows[i]
			if r.diag < 0 {
				header(c, s, r.path, perFile[r.path], i == s.row)
				return
			}
			item(c, diags[r.diag], i == s.row)
		}).Grow(1)
		if (list.Changed() || list.Submitted()) && s.row >= 0 {
			if r := rows[s.row]; r.diag >= 0 {
				ev.Opened = r.diag
			}
		}
	})
	return ev
}

var names = [3]string{"Errors", "Warnings", "Infos"}

// header creates the row of a file: a chevron, its name, its folder
// dimmed, and its count. A click folds or unfolds it.
func header(c *ui.Context, s *State, p string, n int, chosen bool) {
	t := c.Theme()
	ink, muted := t.Text, t.TextMuted
	r := ui.Row(c).Gap(t.Space(1.5)).AlignItems(ui.Center).Padding(t.Space(1), t.Space(1)).Background(t.Background)
	if chosen {
		ink, muted = t.AccentText, t.AccentText
		r.Background(t.Accent).Radius(t.Space(1))
	}
	folded := s.folded[p]
	if r.Clicked() {
		s.folded[p] = !folded
	}
	r.Children(func() {
		ui.Box(c).Size(t.Space(4), t.Space(4)).Draw(func(pa *ui.Painter, rc ui.Rect) { chevron(pa, rc, !folded, muted) })
		ui.Text(c, path.Base(p)).TextColor(ink).SingleLine().Shrink(0)
		if dir := path.Dir(p); dir != "." {
			ui.Text(c, dir).FontSize(t.FontSize - 1).TextColor(muted).SingleLine().Shrink(1)
		}
		ui.Spacer(c)
		ui.Text(c, fmt.Sprint(n)).FontSize(t.FontSize-2).Padding(0, t.Space(1.5)).Radius(999).
			Background(muted.Alpha(0.15)).TextColor(ink)
	})
}

// item creates the row of a diagnostic.
func item(c *ui.Context, d Diagnostic, chosen bool) {
	t := c.Theme()
	ink, muted := t.Text, t.TextMuted
	r := ui.Row(c).Gap(t.Space(1.5)).AlignItems(ui.Center).Padding(t.Space(0.75), t.Space(1), t.Space(0.75), t.Space(6)).
		Radius(t.Space(1))
	if chosen {
		ink, muted = t.AccentText, t.AccentText
		r.Background(t.Accent)
	}
	r.Children(func() {
		ui.Box(c).Size(14, 14).Shrink(0).Draw(func(p *ui.Painter, rc ui.Rect) { mark(p, rc, d.Severity, t, false) })
		spans := []ui.Span{{Text: d.Message, Color: ink}}
		if by := strings.TrimSpace(d.Source + " " + paren(d.Code)); by != "" {
			spans = append(spans, ui.Span{Text: "  " + by, Color: muted})
		}
		ui.RichText(c, spans...).SingleLine().Grow(1).Shrink(1).FontSize(t.FontSize - 1)
		ui.Text(c, fmt.Sprintf("[Ln %d, Col %d]", d.Line, d.Col)).FontSize(t.FontSize - 2).TextColor(muted).FontFeatures("tnum").Shrink(0)
	})
}

func paren(code string) string {
	if code == "" {
		return ""
	}
	return "(" + code + ")"
}

// mark draws the mark of sev in r: a red circle with a cross, a yellow
// triangle with a bang, or a blue circle with an i; faded when off.
func mark(p *ui.Painter, r ui.Rect, sev Severity, t *ui.Theme, off bool) {
	cx, cy, rad := r.X+r.W/2, r.Y+r.H/2, min(r.W, r.H)/2
	white := ui.RGB(255, 255, 255)
	color := [3]ui.Color{t.Danger, t.Warning, t.Accent}[sev]
	if off {
		color = color.Alpha(0.4)
	}
	switch sev {
	case Error:
		var c ui.Path
		p.FillPath(c.Circle(cx, cy, rad), color)
		d := rad * 0.4
		p.StrokePolyline([][2]float32{{cx - d, cy - d}, {cx + d, cy + d}}, 1.5, white)
		p.StrokePolyline([][2]float32{{cx + d, cy - d}, {cx - d, cy + d}}, 1.5, white)
	case Warning:
		var tri ui.Path
		tri.MoveTo(cx, r.Y+1).LineTo(r.X+r.W, r.Y+r.H-1).LineTo(r.X, r.Y+r.H-1).Close()
		p.FillPath(&tri, color)
		p.Line(cx, cy-rad*0.25, cx, cy+rad*0.3, 1.5, white)
		p.Fill(ui.Rect{X: cx - 0.75, Y: cy + rad*0.5, W: 1.5, H: 1.5}, white, 0)
	case Info:
		var c ui.Path
		p.FillPath(c.Circle(cx, cy, rad), color)
		p.Line(cx, cy-rad*0.1, cx, cy+rad*0.5, 1.5, white)
		p.Fill(ui.Rect{X: cx - 0.75, Y: cy - rad*0.5, W: 1.5, H: 1.5}, white, 0)
	}
}

// chevron draws an arrow in r pointing down when open, right when not.
func chevron(p *ui.Painter, r ui.Rect, open bool, color ui.Color) {
	cx, cy, d := r.X+r.W/2, r.Y+r.H/2, float32(3)
	pts := [][2]float32{{cx - d/2, cy - d}, {cx + d/2, cy}, {cx - d/2, cy + d}}
	if open {
		pts = [][2]float32{{cx - d, cy - d/2}, {cx, cy + d/2}, {cx + d, cy - d/2}}
	}
	p.StrokePolyline(pts, 1.4, color)
}
