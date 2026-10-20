// Package waterfall shows when the parts of a run happened, as browsers'
// network panels show a page loading and tracing tools show a request: a
// row for each request or span, nested under the span it was part of,
// with its columns, and a bar from its start to its end, colored by its
// phases. Dragging across the time scale zooms to that stretch, and a
// click on the scale or Escape zooms back out. A click or the arrow keys
// choose a row, which Chosen returns; Left and Right fold and unfold it.
// A line below tells of the row under the pointer, phase by phase.
//
//	spans, err := waterfall.ParseHAR(f) // once, not every frame
//	...
//	waterfall.View(c, &app.network, waterfall.HARColumns, spans).Grow(1)
//	if span, ok := app.network.Chosen(); ok {
//		... // show it
//	}
package waterfall

import (
	"fmt"
	"math"
	"slices"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
)

// Span is a request or a step of a run: its name, the values of the
// view's columns, when it started, from the start of the run, its phases
// one after another, and the spans it was made of, its children.
type Span struct {
	Name     string
	Info     []string
	Start    time.Duration
	Phases   []Phase
	Children []*Span
}

// Phase is a part of a span: its name, how long it took, and its color;
// a phase with none takes one of the view's, the same for the phases of
// one name.
type Phase struct {
	Name     string
	Duration time.Duration
	Color    ui.Color
}

// End returns when s ended, from the start of the run.
func (s *Span) End() time.Duration {
	end := s.Start
	for _, p := range s.Phases {
		end += p.Duration
	}
	return end
}

// State is what a view keeps: the rows folded, the row chosen, and the
// stretch of time it is zoomed to. Keep it in the app's state, one for
// each view.
type State struct {
	list     ui.ListState
	row      int
	rows     []row
	closed   map[*Span]bool
	from, to time.Duration
	drag     struct {
		on     bool
		x0, x1 float32
	}
}

// row is a span showing, and how deep in the tree it is.
type row struct {
	span  *Span
	depth int
}

// Chosen returns the span of the row chosen, after View, or false with
// none.
func (s *State) Chosen() (*Span, bool) {
	if s.row < 0 || s.row >= len(s.rows) {
		return nil, false
	}
	return s.rows[s.row].span, true
}

// View creates the view of spans, with columns, the names of the values
// of their Info, before the bars. Give it a height, or Grow.
func View(c *ui.Context, s *State, columns []string, spans []*Span) ui.Element {
	t := c.Theme()
	if s.list.Selected == nil {
		s.row = -1
		s.list.Selected = &s.row
	}
	if s.closed == nil {
		s.closed = map[*Span]bool{}
	}
	s.rows = s.rows[:0]
	var end time.Duration
	nested := false
	var walk func(spans []*Span, depth int)
	walk = func(spans []*Span, depth int) {
		for _, sp := range spans {
			end = max(end, sp.End())
			nested = nested || len(sp.Children) > 0
			s.rows = append(s.rows, row{sp, depth})
			if !s.closed[sp] {
				walk(sp.Children, depth+1)
			}
		}
	}
	walk(spans, 0)
	end = max(end, time.Millisecond)
	from, to := s.from, s.to
	if to <= from {
		// Room after the last bar keeps it off the edge.
		from, to = 0, end+end/40
	}
	colors := palette(spans, t.Dark)
	rows := s.rows
	s.list.Key = func(i int) any { return rows[i].span }
	s.list.Label = func(i int) string { return rows[i].span.Name }
	rowH := t.FontSize + t.Space(3)
	nameW, infoW := t.Space(46), t.Space(16)
	var hovered *Span

	return ui.Column(c).Children(func() {
		ui.Row(c).Height(rowH).PaddingStart(t.Space(3)).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "Name").FontSize(t.FontSize - 1).TextColor(t.TextMuted).Width(nameW).Shrink(0)
			for k, name := range columns {
				ui.Text(c.Key(k), name).FontSize(t.FontSize - 1).TextColor(t.TextMuted).Width(infoW).Shrink(0).
					TextAlign(ui.End).SingleLine()
			}
			s.ruler(c, from, to, rowH)
		})
		ui.Divider(c)
		list := ui.List(c, &s.list, len(rows), func(i int) {
			r := rows[i]
			e := ui.Row(c).Height(rowH).PaddingStart(t.Space(3)).Gap(t.Space(2)).AlignItems(ui.Center)
			if e.Hovered() {
				hovered = r.span
			}
			e.Children(func() {
				s.name(c, r, nameW, nested)
				for k := range columns {
					v := ""
					if k < len(r.span.Info) {
						v = r.span.Info[k]
					}
					ui.Text(c.Key(k), v).FontSize(t.FontSize - 1).FontFeatures("tnum").Width(infoW).Shrink(0).
						TextAlign(ui.End).SingleLine()
				}
				bar(c, r.span, from, to, colors)
			})
		}).Grow(1)
		if sp, ok := s.Chosen(); ok && len(sp.Children) > 0 {
			if list.Shortcut(0, ui.KeyLeft) {
				s.closed[sp] = true
			}
			if list.Shortcut(0, ui.KeyRight) {
				delete(s.closed, sp)
			}
		}
		if list.Shortcut(0, ui.KeyEscape) {
			s.from, s.to = 0, 0
		}
		ui.Divider(c)
		ui.Row(c).Gap(t.Space(3)).Padding(t.Space(1.5), t.Space(3)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, status(hovered, len(rows), end, s.from, s.to)).FontSize(t.FontSize - 1).TextColor(t.TextMuted).
				FontFeatures("tnum").SingleLine().Grow(1).Shrink(1)
			for k, p := range legend(spans) {
				ui.Row(c.Key(k)).Gap(t.Space(1)).AlignItems(ui.Center).Children(func() {
					ui.Box(c).Size(8, 8).Radius(2).Background(colors(p))
					ui.Text(c, p.Name).FontSize(t.FontSize - 2).TextColor(t.TextMuted).SingleLine()
				})
			}
		})
	})
}

// name shows the name of r, indented by its depth, with a chevron that
// folds and unfolds a span with children, when any is nested.
func (s *State) name(c *ui.Context, r row, w float32, nested bool) {
	t := c.Theme()
	ui.Row(c).Width(w).Shrink(0).PaddingStart(float32(r.depth) * t.Space(4)).Gap(t.Space(1)).AlignItems(ui.Center).
		Children(func() {
			if !nested {
				ui.Text(c, r.span.Name).SingleLine().Shrink(1)
				return
			}
			open := !s.closed[r.span]
			b := ui.ButtonBase(c).Size(t.Space(4), t.Space(4)).Shrink(0)
			if len(r.span.Children) == 0 {
				b.Disabled(true).Opacity(0)
			} else {
				b.Label("Fold")
				if !open {
					b.Label("Unfold")
				}
			}
			if b.Clicked() {
				if open {
					s.closed[r.span] = true
				} else {
					delete(s.closed, r.span)
				}
			}
			color := t.TextMuted
			b.Draw(func(p *ui.Painter, rect ui.Rect) {
				cx, cy, d := rect.X+rect.W/2, rect.Y+rect.H/2, float32(3)
				pts := [][2]float32{{cx - d/2, cy - d}, {cx + d/2, cy}, {cx - d/2, cy + d}}
				if open {
					pts = [][2]float32{{cx - d, cy - d/2}, {cx, cy + d/2}, {cx + d, cy - d/2}}
				}
				p.StrokePolyline(pts, 1.5, color)
			})
			ui.Text(c, r.span.Name).SingleLine().Shrink(1)
		})
}

// ruler shows the time scale from from to to, which dragging across zooms
// to, and a click on zooms back out.
func (s *State) ruler(c *ui.Context, from, to time.Duration, h float32) {
	t := c.Theme()
	box := ui.Box(c).Grow(1).Height(h).MarginStart(t.Space(1)).Cursor(ui.CursorCrosshair).Role(ui.RoleSlider).Label("Time scale")
	w := box.LayoutBounds().W
	if box.Pressed() {
		x, _, ok := box.PointerPosition()
		if ok {
			x = max(0, min(w, x))
			if !s.drag.on {
				s.drag.on, s.drag.x0 = true, x
			}
			s.drag.x1 = x
		}
	} else if s.drag.on {
		s.drag.on = false
		x0, x1 := min(s.drag.x0, s.drag.x1), max(s.drag.x0, s.drag.x1)
		if x1-x0 < 4 || w <= 0 {
			s.from, s.to = 0, 0
		} else {
			at := func(x float32) time.Duration { return from + time.Duration(float64(to-from)*float64(x/w)) }
			s.from, s.to = at(x0), at(x1)
		}
	}
	drag := s.drag
	box.Draw(func(p *ui.Painter, r ui.Rect) {
		for _, tk := range ticks(from, to, r.W, t.Space(24)) {
			x := r.X + tk.x
			p.Line(x, r.Y+r.H-4, x, r.Y+r.H, 1, t.TextMuted)
			p.Text(x+3, r.Y+(r.H-t.FontSize*1.2)/2, tk.label, t.FontSize-2, t.TextMuted)
		}
		if drag.on {
			x0, x1 := min(drag.x0, drag.x1), max(drag.x0, drag.x1)
			p.Fill(ui.Rect{X: r.X + x0, Y: r.Y, W: x1 - x0, H: r.H}, t.Accent.Alpha(0.2), 0)
		}
	})
}

// bar shows the bar of sp in the stretch from from to to, a part a phase,
// over the lines of the time scale.
func bar(c *ui.Context, sp *Span, from, to time.Duration, colors func(Phase) ui.Color) {
	t := c.Theme()
	ui.Box(c).Grow(1).AlignSelf(ui.Stretch).MarginStart(t.Space(1)).Clip().Draw(func(p *ui.Painter, r ui.Rect) {
		for _, tk := range ticks(from, to, r.W, t.Space(24)) {
			p.Line(r.X+tk.x, r.Y, r.X+tk.x, r.Y+r.H, 1, t.Text.Alpha(0.06))
		}
		x := func(d time.Duration) float32 { return r.X + float32(float64(d-from)/float64(to-from))*r.W }
		h := r.H * 0.42
		y := r.Y + (r.H-h)/2
		at := sp.Start
		for _, ph := range sp.Phases {
			x0, x1 := x(at), x(at+ph.Duration)
			at += ph.Duration
			if x1 < r.X || x0 > r.X+r.W {
				continue
			}
			p.Fill(ui.Rect{X: x0, Y: y, W: max(1, x1-x0), H: h}, colors(ph), 0)
		}
	})
}

// tick is a mark of the time scale, x from its left.
type tick struct {
	x     float32
	label string
}

// ticks returns the marks of the time scale from from to to, w wide, at
// least gap apart, at round times.
func ticks(from, to time.Duration, w, gap float32) []tick {
	if w <= 0 || to <= from {
		return nil
	}
	span := float64(to - from)
	raw := span * float64(gap/w)
	step := math.Pow(10, math.Floor(math.Log10(raw)))
	for _, m := range []float64{1, 2, 5, 10} {
		if step*m >= raw {
			step *= m
			break
		}
	}
	var out []tick
	for v := math.Ceil(float64(from)/step) * step; v <= float64(to); v += step {
		out = append(out, tick{float32((v - float64(from)) / span * float64(w)), short(time.Duration(v))})
	}
	return out
}

// short formats d as a time scale does: "250 ms", "1.5 s", "40 µs".
func short(d time.Duration) string {
	switch {
	case d == 0:
		return "0"
	case d < time.Millisecond:
		return trim(float64(d)/float64(time.Microsecond)) + " µs"
	case d < time.Second:
		return trim(float64(d)/float64(time.Millisecond)) + " ms"
	}
	return trim(d.Seconds()) + " s"
}

func trim(v float64) string {
	s := fmt.Sprintf("%.2f", v)
	s = strings.TrimRight(s, "0")
	return strings.TrimSuffix(s, ".")
}

// legend returns the first phase of each name in spans, in the order they
// come.
func legend(spans []*Span) []Phase {
	var out []Phase
	var walk func(spans []*Span)
	walk = func(spans []*Span) {
		for _, sp := range spans {
			for _, ph := range sp.Phases {
				if !slices.ContainsFunc(out, func(p Phase) bool { return p.Name == ph.Name }) {
					out = append(out, ph)
				}
			}
			walk(sp.Children)
		}
	}
	walk(spans)
	return out
}

// palette returns the color of a phase: its own, or for a phase without
// one, a hue for its name, by the order the names come.
func palette(spans []*Span, dark bool) func(Phase) ui.Color {
	names := map[string]int{}
	for _, p := range legend(spans) {
		names[p.Name] = len(names)
	}
	l := float32(0.66)
	if dark {
		l = 0.72
	}
	return func(p Phase) ui.Color {
		if p.Color.A > 0 {
			return p.Color
		}
		return ui.Oklch(l, 0.14, float32((250+names[p.Name]*137)%360))
	}
}

// status returns the line below the rows: of hovered, phase by phase, or
// of the rows and the stretch zoomed to.
func status(hovered *Span, rows int, end, from, to time.Duration) string {
	if hovered != nil {
		parts := []string{hovered.Name, "at " + long(hovered.Start), long(hovered.End() - hovered.Start)}
		var phases []string
		for _, p := range hovered.Phases {
			phases = append(phases, p.Name+" "+long(p.Duration))
		}
		if len(phases) > 1 {
			parts = append(parts, strings.Join(phases, ", "))
		}
		return strings.Join(parts, "  ·  ")
	}
	line := fmt.Sprintf("%d rows  ·  %s", rows, long(end))
	if to > from {
		line += fmt.Sprintf("  ·  showing %s to %s", long(from), long(to))
	}
	return line
}

// long formats d as short does, to the millisecond from 10 ms on.
func long(d time.Duration) string {
	if d >= 10*time.Millisecond && d < time.Second {
		d = d.Round(time.Millisecond)
	}
	return short(d)
}
