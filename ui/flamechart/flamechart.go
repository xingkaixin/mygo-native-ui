// Package flamechart shows a trace over time, as Chrome's Performance
// panel does: a lane for each thread, under its name, and each call a bar
// from its start to its end, the calls it made in bars below it, colored
// by their category, or their name when the trace has no categories, and
// named as far as the bar has room. Scrolling with Cmd held, Command on
// macOS and Control elsewhere, zooms around the pointer, down to a
// microsecond across the view; scrolling sideways and dragging pan. A
// ruler above follows the zoom. The pointer over a bar shows how long the
// call took and how long it ran itself; a click chooses it, which Chosen
// returns, and a double click zooms to it. A search dims the calls not
// matching it, and a click on a lane's name folds the lane.
//
//	trace, err := flamechart.Parse(f) // once, not every frame
//	...
//	ui.SearchField(c, &app.query)
//	flamechart.View(c, &app.chart, trace, app.query).Grow(1)
//	if thread, call, ok := app.chart.Chosen(); ok {
//		... // show it
//	}
package flamechart

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
)

// minSpan is the shortest stretch of time the view zooms to.
const minSpan = float64(time.Microsecond)

// State is what a view keeps: the stretch of time it shows, the lanes
// folded and the call chosen. Keep it in the app's state, one for each
// view.
type State struct {
	// from and span are the stretch shown, in nanoseconds from the start
	// of the trace; span 0 shows all of it.
	from, span float64
	folded     map[*Thread]bool
	thread     *Thread
	chosen     *Call
	scroll     ui.ScrollState
	// query and trace are those the search last ran on, and matches
	// whether each name matched, matched the calls that did.
	query   string
	trace   *Trace
	matches map[string]bool
	matched int
	// widths are those of the names drawn, at size.
	widths map[string]float32
	size   float32
}

// Chosen returns the call chosen and its thread, or false with none.
func (s *State) Chosen() (*Thread, *Call, bool) {
	return s.thread, s.chosen, s.chosen != nil
}

// window returns the stretch shown of a trace end long.
func (s *State) window(end float64) (from, span float64) {
	if s.span <= 0 {
		return 0, end
	}
	return s.from, s.span
}

// show shows span from from, at least minSpan long, within the trace.
func (s *State) show(end, from, span float64) {
	span = max(minSpan, span)
	if span >= end {
		s.from, s.span = 0, 0
		return
	}
	s.from, s.span = max(0, min(end-span, from)), span
}

// zoom scales the stretch shown by factor around at, a fraction of the
// view that stays where it is.
func (s *State) zoom(end, at, factor float64) {
	from, span := s.window(end)
	pivot := from + at*span
	span *= factor
	s.show(end, pivot-at*span, span)
}

// pan moves the stretch shown by d of its length, and reports whether it
// moved.
func (s *State) pan(end, d float64) bool {
	from, span := s.window(end)
	s.show(end, from+d*span, span)
	return s.from != from
}

// find runs the search for query, lowered, unless it ran on tr already.
func (s *State) find(tr *Trace, query string) {
	if query == s.query && tr == s.trace {
		return
	}
	s.query, s.trace, s.matched = query, tr, 0
	s.matches = map[string]bool{}
	if query == "" {
		return
	}
	for _, th := range tr.Threads {
		for _, row := range th.rows {
			for i := range row {
				m, ok := s.matches[row[i].Name]
				if !ok {
					m = strings.Contains(strings.ToLower(row[i].Name), query)
					s.matches[row[i].Name] = m
				}
				if m {
					s.matched++
				}
			}
		}
	}
}

// View creates the view of tr, dimming the calls whose names do not hold
// search, without regard to case, with a line below of the trace, the
// stretch shown and the calls matching. While it has the keyboard focus,
// W and S zoom in and out and A and D pan, as in Chrome. Give it a height,
// or Grow; the lanes scroll when they do not fit.
func View(c *ui.Context, s *State, tr *Trace, search string) ui.Element {
	t := c.Theme()
	rowH := t.FontSize + t.Space(2)
	end := max(minSpan, float64(tr.End))
	s.find(tr, strings.ToLower(search))
	if s.folded == nil {
		s.folded = map[*Thread]bool{}
	}
	root := ui.Column(c)
	return root.Children(func() {
		s.ruler(c, end, rowH)
		ui.Divider(c)
		var hovered *Call
		ui.Scroll(c).Grow(1).TrackScroll(&s.scroll).Children(func() {
			lanes := ui.Column(c).Focusable().Role(ui.RoleImage).Label("Flame chart")
			w := max(1, lanes.LayoutBounds().W)
			s.input(c, lanes, end)
			lanes.Draw(func(p *ui.Painter, r ui.Rect) {
				from, span := s.window(end)
				for _, tk := range ticks(from, span, r.W, t.Space(24)) {
					p.Line(r.X+tk.x, r.Y, r.X+tk.x, r.Y+r.H, 1, t.Text.Alpha(0.06))
				}
			})
			lanes.Children(func() {
				for i, th := range tr.Threads {
					ui.Column(c.Key(i)).Children(func() {
						s.header(c, th, rowH)
						if s.folded[th] {
							return
						}
						if call := s.lane(c, lanes, th, end, w, rowH); call != nil {
							hovered = call
						}
					})
				}
			})
		})
		ui.Divider(c)
		ui.Text(c, s.status(tr, end)).FontSize(t.FontSize-1).TextColor(t.TextMuted).
			FontFeatures("tnum").SingleLine().Padding(t.Space(1.5), t.Space(3))
		if hovered != nil {
			tooltip(c, root, hovered)
		}
	})
}

// input has lanes zoom and pan with the wheel and the keys.
func (s *State) input(c *ui.Context, lanes ui.Element, end float64) {
	// Scrolling comes between builds, at the box laid out since, which
	// only a handle reads then.
	h := ui.Local(lanes, "flamechart.handle", func() ui.Handle { return ui.Handle{} })
	lanes.Bind(h)
	lanes.HandleInput(func(ev ui.InputEvent) bool {
		w := h.Bounds(c).W
		if ev.Kind != ui.InputScroll || w <= 0 {
			return false
		}
		if ev.Mods&ui.Cmd != 0 && ev.DY != 0 {
			// Scrolling 300 DIPs zooms by 2, as chartx.Zoom does.
			s.zoom(end, float64(ev.X/w), math.Exp2(float64(ev.DY)/300))
			return true
		}
		dx, dy := ev.DX, ev.DY
		if ev.Mods&ui.Shift != 0 && dx == 0 {
			dx, dy = dy, 0
		}
		if math.Abs(float64(dx)) <= math.Abs(float64(dy)) {
			return false
		}
		return s.pan(end, float64(dx/w))
	})
	for _, k := range []struct {
		key ui.Key
		do  func()
	}{
		{ui.KeyW, func() { s.zoom(end, 0.5, 0.8) }},
		{ui.KeyS, func() { s.zoom(end, 0.5, 1.25) }},
		{ui.KeyA, func() { s.pan(end, -0.1) }},
		{ui.KeyD, func() { s.pan(end, 0.1) }},
	} {
		if lanes.Shortcut(0, k.key) {
			k.do()
		}
	}
}

// header shows the name of th and how many calls it made, which a click
// folds and unfolds.
func (s *State) header(c *ui.Context, th *Thread, rowH float32) {
	t := c.Theme()
	open := !s.folded[th]
	b := ui.ButtonBase(c).Height(rowH + t.Space(1)).PaddingStart(t.Space(2)).Gap(t.Space(1.5)).AlignItems(ui.Center).
		Justify(ui.Start).Background(t.Surface)
	if b.Clicked() {
		s.folded[th] = open
	}
	calls := 0
	for _, row := range th.rows {
		calls += len(row)
	}
	color := t.TextMuted
	b.Children(func() {
		ui.Box(c).Size(t.Space(3), t.Space(3)).Shrink(0).Draw(func(p *ui.Painter, r ui.Rect) {
			cx, cy, d := r.X+r.W/2, r.Y+r.H/2, float32(3)
			pts := [][2]float32{{cx - d/2, cy - d}, {cx + d/2, cy}, {cx - d/2, cy + d}}
			if open {
				pts = [][2]float32{{cx - d, cy - d/2}, {cx, cy + d/2}, {cx + d, cy - d/2}}
			}
			p.StrokePolyline(pts, 1.5, color)
		})
		ui.Text(c, th.Name).FontSize(t.FontSize - 1).FontWeight(600).SingleLine().Shrink(1)
		ui.Text(c, count(calls)+" calls").FontSize(t.FontSize - 2).TextColor(t.TextMuted).FontFeatures("tnum").
			SingleLine()
	})
}

// lane shows the calls of th, a row for each depth, in lanes w wide, and
// returns the call under the pointer.
func (s *State) lane(c *ui.Context, lanes ui.Element, th *Thread, end float64, w, rowH float32) *Call {
	t := c.Theme()
	box := ui.Box(c).Height(float32(len(th.rows))*rowH).Margin(0, 0, t.Space(1), 0)
	from, span := s.window(end)
	var hovered *Call
	if x, y, over := box.PointerPosition(); over {
		hovered = th.at(int(y/rowH), from+float64(x/w)*span, 2*span/float64(w))
	}
	// moved is how far the press went, so that a drag ending over a bar
	// does not choose it.
	moved := ui.Local(box, "flamechart.moved", func() float32 { return 0 })
	dx, dy, pressed := box.Dragged()
	if pressed {
		lanes.Focus()
		*moved += float32(math.Abs(float64(dx)) + math.Abs(float64(dy)))
		if dx != 0 || dy != 0 {
			s.pan(end, -float64(dx/w))
			s.scroll.Y -= dy
			box.Cursor(ui.CursorGrabbing)
		}
	}
	if box.Clicked() && *moved < 4 {
		s.thread, s.chosen = th, hovered
		if hovered == nil {
			s.thread = nil
		}
	}
	if box.DoubleClicked() && *moved < 4 {
		if hovered != nil {
			d := float64(hovered.Duration)
			s.show(end, float64(hovered.Start)-d/20, d*1.1)
		} else {
			s.span = 0
		}
	}
	if !pressed {
		*moved = 0
	}
	box.Draw(func(p *ui.Painter, r ui.Rect) { s.paint(p, r, th, end, rowH, hovered) })
	return hovered
}

// at returns the call of th at depth running at t, or one within slop of
// it, if any.
func (th *Thread) at(depth int, t, slop float64) *Call {
	if depth < 0 || depth >= len(th.rows) {
		return nil
	}
	row := th.rows[depth]
	i := sort.Search(len(row), func(i int) bool { return float64(row[i].End()) > t })
	if i < len(row) && float64(row[i].Start) <= t {
		return &row[i]
	}
	i = sort.Search(len(row), func(i int) bool { return float64(row[i].End()) > t-slop })
	if i < len(row) && float64(row[i].Start) <= t+slop {
		return &row[i]
	}
	return nil
}

// paint draws the calls of th in view in r, a pixel's column at most for
// the calls too short to show apart, so that a trace of many draws as
// fast as one of few.
func (s *State) paint(p *ui.Painter, r ui.Rect, th *Thread, end float64, rowH float32, hovered *Call) {
	t := p.Theme()
	from, span := s.window(end)
	perDIP := span / float64(r.W)
	vis := p.Visible()
	size := t.FontSize - 1
	for d, row := range th.rows {
		y := r.Y + float32(d)*rowH
		if y > vis.Y+vis.H || y+rowH < vis.Y {
			continue
		}
		i := sort.Search(len(row), func(i int) bool { return float64(row[i].End()) > from })
		for i < len(row) && float64(row[i].Start) < from+span {
			c := &row[i]
			x0 := float32((float64(c.Start) - from) / perDIP)
			x1 := float32((float64(c.End()) - from) / perDIP)
			fill, dim := s.color(c, t)
			if x1-x0 < 1 {
				x := float32(math.Floor(float64(x0)))
				p.Fill(ui.Rect{X: r.X + x, Y: y, W: 1, H: rowH - 1}, fill, 0)
				next := from + float64(x+1)*perDIP
				rest := row[i+1:]
				i += 1 + sort.Search(len(rest), func(k int) bool { return float64(rest[k].End()) > next })
				continue
			}
			i++
			x0, x1 = max(x0, -2), min(x1, r.W+2)
			rect := ui.Rect{X: r.X + x0, Y: y, W: x1 - x0, H: rowH - 1}
			radius := float32(0)
			if rect.W > 2 {
				rect.W--
			}
			if rect.W > 6 {
				radius = 2
			}
			if c == hovered {
				fill = fill.Mix(t.Text, 0.15)
			}
			p.Fill(rect, fill, radius)
			if c == s.chosen {
				p.Stroke(rect, t.Text, radius, 1.5)
			}
			if rect.W < 2*t.FontSize {
				continue
			}
			ink := ui.RGB(0, 0, 0).Alpha(0.85)
			if dim {
				ink = t.Text.Alpha(0.4)
			}
			lx := max(rect.X, r.X) + 4
			if name := s.fit(p, c.Name, size, rect.X+rect.W-lx-4); name != "" {
				p.Clip(rect, 0, func() {
					p.Text(lx, y+(rowH-1-size*1.2)/2, name, size, ink)
				})
			}
		}
	}
}

// color returns the fill of c's bar, and whether the search dims it.
func (s *State) color(c *Call, t *ui.Theme) (ui.Color, bool) {
	l := float32(0.82)
	if t.Dark {
		l = 0.72
	}
	fill := ui.Oklch(l+c.shade, 0.11, c.hue)
	if s.query != "" && !s.matches[c.Name] {
		return fill.Alpha(0.2), true
	}
	return fill, false
}

// fit returns name, cut short with an ellipsis to be width wide at most,
// or "" when two letters do not fit. It cuts by the name's width on
// average, which the bar's clip makes up for.
func (s *State) fit(p *ui.Painter, name string, size, width float32) string {
	if s.size != size || s.widths == nil {
		s.widths, s.size = map[string]float32{}, size
	}
	w, ok := s.widths[name]
	if !ok {
		w, _ = p.MeasureText(0, ui.Span{Text: name, Size: size})
		s.widths[name] = w
	}
	if w <= width {
		return name
	}
	n := utf8.RuneCountInString(name)
	k := int(float32(n) * (width - size) / w)
	if k < 2 {
		return ""
	}
	return string([]rune(name)[:k]) + "…"
}

// ruler shows the time of the stretch shown, at round times.
func (s *State) ruler(c *ui.Context, end float64, h float32) {
	t := c.Theme()
	ui.Box(c).Height(h).Role(ui.RoleImage).Label("Time").Draw(func(p *ui.Painter, r ui.Rect) {
		from, span := s.window(end)
		for _, tk := range ticks(from, span, r.W, t.Space(24)) {
			x := r.X + tk.x
			p.Line(x, r.Y+r.H-4, x, r.Y+r.H, 1, t.TextMuted)
			if w, _ := p.MeasureText(0, ui.Span{Text: tk.label, Size: t.FontSize - 2}); x+3+w <= r.X+r.W {
				p.Text(x+3, r.Y+(r.H-t.FontSize*1.2)/2, tk.label, t.FontSize-2, t.TextMuted)
			}
		}
	})
}

// tooltip shows beside the pointer over root how long call took and how
// long it ran itself.
func tooltip(c *ui.Context, root ui.Element, call *Call) {
	t := c.Theme()
	x, y, _ := root.PointerPosition()
	tip := ui.Column(c).Absolute().Gap(t.Space(0.5)).Padding(t.Space(1.5), t.Space(2.5)).Radius(6).
		Background(t.Background).Border(1, t.Border).Shadow(0, 2, 8, 0, ui.RGBA(0, 0, 0, 0.15)).MaxWidth(t.Space(100))
	b, rb := tip.LayoutBounds(), root.LayoutBounds()
	left, top := x+14, y+18
	if left+b.W > rb.W {
		left = x - 14 - b.W
	}
	if top+b.H > rb.H {
		top = y - 10 - b.H
	}
	tip.Left(max(0, left)).Top(max(0, top)).Children(func() {
		ui.Text(c, call.Name).FontWeight(600).SingleLine()
		line := long(call.Duration) + "  ·  self " + long(call.Self)
		if call.Category != "" {
			line += "  ·  " + call.Category
		}
		ui.Text(c, line).FontSize(t.FontSize - 1).TextColor(t.TextMuted).FontFeatures("tnum").SingleLine()
	})
}

// status returns the line under the view: the threads, calls and length
// of tr, the stretch shown when zoomed, and the calls the search matches.
func (s *State) status(tr *Trace, end float64) string {
	line := fmt.Sprintf("%d threads  ·  %s calls  ·  %s", len(tr.Threads), count(tr.calls), long(tr.End))
	if s.span > 0 {
		from, span := s.window(end)
		u, name, digits := scale(from+span, span/10)
		line += fmt.Sprintf("  ·  showing %.*f to %.*f %s", digits, from/u, digits, (from+span)/u, name)
	}
	if s.query != "" {
		line += fmt.Sprintf("  ·  %s matching", count(s.matched))
	}
	return line
}

// tick is a mark of the ruler, x from its left.
type tick struct {
	x     float32
	label string
}

// ticks returns the marks of the ruler for span from from, w wide, at
// least gap apart, at round times.
func ticks(from, span float64, w, gap float32) []tick {
	if w <= 0 || span <= 0 {
		return nil
	}
	raw := span * float64(gap/w)
	step := math.Pow(10, math.Floor(math.Log10(raw)))
	for _, m := range []float64{1, 2, 5, 10} {
		if step*m >= raw {
			step *= m
			break
		}
	}
	u, name, digits := scale(from+span, step)
	var out []tick
	for k := math.Ceil(from / step); k*step <= from+span; k++ {
		v := k * step
		label := "0"
		if v >= step/2 {
			label = strconv.FormatFloat(v/u, 'f', digits, 64) + " " + name
		}
		out = append(out, tick{float32((v - from) / span * float64(w)), label})
	}
	return out
}

// unit returns the unit to write ns nanoseconds in, in nanoseconds, and
// its name.
func unit(ns float64) (float64, string) {
	switch {
	case ns >= 1e9:
		return 1e9, "s"
	case ns >= 1e6:
		return 1e6, "ms"
	case ns >= 1e3:
		return 1e3, "µs"
	}
	return 1, "ns"
}

// scale returns the unit to write times up to ns in, step apart, its
// name, and the digits they need after the point: a smaller unit where
// they would need more than 4, as zoomed in far from the start.
func scale(ns, step float64) (u float64, name string, digits int) {
	u, name = unit(ns)
	for {
		digits = max(0, int(math.Ceil(math.Log10(u/step)-1e-9)))
		if digits <= 4 || u == 1 {
			return u, name, digits
		}
		u, name = unit(u / 1000)
	}
}

// long writes d to three figures: "1.24 ms", "312 µs".
func long(d time.Duration) string {
	u, name := unit(float64(d))
	v := float64(d) / u
	digits := 0
	switch {
	case u == 1:
	case v < 10:
		digits = 2
	case v < 100:
		digits = 1
	}
	return strconv.FormatFloat(v, 'f', digits, 64) + " " + name
}

// count writes n with thousands separated, "12,345".
func count(n int) string {
	s := strconv.Itoa(n)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}
