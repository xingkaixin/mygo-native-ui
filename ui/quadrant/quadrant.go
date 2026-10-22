// Package quadrant shows items on two axes split into four quadrants, as
// a priority matrix weighs impact against effort, or Eisenhower's urgent
// against important, and as Mermaid's quadrantChart draws them: both axes
// run from 0 to 1, with their low and high labels at their ends, and the
// quadrants are tinted, each with its title at its top. Each item is a
// dot with its name beside it, on the side where the name covers no other
// name, dot or title and stays inside the chart. A dot is in its item's
// own color, or else in that of the quadrant it is in.
//
// Dragging a dot moves its item; while a dot has the keyboard focus, the
// arrow keys move it by a hundredth, or a tenth with Shift. The pointer
// on a dot grows it and shows a card of its name, its quadrant and its x
// and y. The chart moves the app's points in place and reports the moves
// as changes:
//
//	if quadrant.View(c, &app.matrix, app.chart, &app.points).Grow(1).Changed() {
//		app.source = quadrant.Format(app.chart, app.points)
//	}
//
// Parse reads Mermaid's quadrantChart text into a chart and its points,
// and Format writes them back.
package quadrant

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
)

// Chart is what a quadrant chart shows around its points: its title,
// above it, its axes, and the titles of its quadrants, in Mermaid's
// order: top right, top left, bottom left, bottom right.
type Chart struct {
	Title     string
	X, Y      Axis
	Quadrants [4]string
}

// Axis is an axis from 0 to 1: the labels at its low end and at its high
// end.
type Axis struct {
	Low, High string
}

// Point is an item on a chart: its name, beside its dot, where it is on
// the x and the y axis, each from 0 to 1, and its color, or the zero
// Color for that of its quadrant.
type Point struct {
	Name  string
	X, Y  float64
	Color ui.Color
}

// Quadrant returns the index in Chart.Quadrants of the quadrant p is in.
// The middle of an axis is in its high half.
func (p Point) Quadrant() int {
	switch {
	case p.Y >= 0.5 && p.X >= 0.5:
		return 0
	case p.Y >= 0.5:
		return 1
	case p.X < 0.5:
		return 2
	}
	return 3
}

// hues are the quadrants' hues, in the order of Chart.Quadrants.
var hues = [4]float32{255, 150, 290, 45}

// Color returns the color the chart gives the dots in quadrant q, an
// index in Chart.Quadrants, lighter in a dark theme.
func Color(t *ui.Theme, q int) ui.Color {
	if t.Dark {
		return ui.Oklch(0.72, 0.14, hues[q])
	}
	return ui.Oklch(0.6, 0.16, hues[q])
}

// ink returns the color of quadrant q's title.
func ink(t *ui.Theme, q int) ui.Color {
	if t.Dark {
		return ui.Oklch(0.84, 0.08, hues[q])
	}
	return ui.Oklch(0.44, 0.12, hues[q])
}

// State is what a chart keeps: the point being dragged and where the
// pointer has taken it. Keep it in the app's state, one for each chart.
type State struct {
	// drag is the index of the point dragged, plus 1, and x, y where the
	// pointer has taken its middle, in DIPs of the chart.
	drag int
	x, y float32
}

type growKey struct{}

// View creates the chart of ch with *points, which it moves in place and
// reports the moves as changes. Give it a size, or grow it in its parent.
func View(c *ui.Context, s *State, ch Chart, points *[]Point) ui.Element {
	t := c.Theme()
	name := ch.Title
	if name == "" {
		name = "Quadrant chart"
	}
	box := ui.Box(c).MinWidth(t.Space(60)).MinHeight(t.Space(60)).Role(ui.RoleGroup).Label(name)
	b := box.LayoutBounds()
	l := lay(c, ch, b)
	pts := *points
	hit := t.Space(5)
	xy := make([][2]float32, len(pts))
	radius := make([]float32, len(pts))
	hovered, focused, moved := -1, -1, false
	box.Children(func() {
		for i := range pts {
			pt := &pts[i]
			e := ui.Box(c.Key(i)).Absolute().Size(hit, hit).Radius(hit / 2).Focusable().FocusRing(false).
				Role(ui.RoleSlider)
			if dx, dy, held := e.Dragged(); held {
				if s.drag != i+1 {
					x, y := l.at(*pt)
					s.drag, s.x, s.y = i+1, x, y
				}
				s.x, s.y = s.x+dx, s.y+dy
				x, y := l.value(s.x, s.y)
				if x != pt.X || y != pt.Y {
					pt.X, pt.Y, moved = x, y, true
				}
			} else if s.drag == i+1 {
				s.drag = 0
			}
			if dx, dy := arrows(e); dx != 0 || dy != 0 {
				pt.X, pt.Y = hundredths(pt.X+dx), hundredths(pt.Y+dy)
				moved = true
			}
			x, y := l.at(*pt)
			xy[i] = [2]float32{x, y}
			e.Left(x - hit/2).Top(y - hit/2).Label(pt.Name).Value(reading(ch, *pt))
			if s.drag == i+1 {
				e.Cursor(ui.CursorGrabbing)
			} else {
				e.Cursor(ui.CursorGrab)
			}
			if e.Hovered() {
				hovered = i
			}
			if e.FocusVisible() {
				focused = i
			}
			lit := s.drag == i+1 || e.Hovered() || e.FocusVisible()
			grow := float32(0)
			if lit {
				grow = 1
			}
			radius[i] = t.Space(1.5) + t.Space(0.75)*e.AnimateTo(growKey{}, grow, ui.Smooth)
		}
	})
	if moved {
		box.SetChanged()
	}
	hot := s.drag - 1
	if hot < 0 || hot >= len(pts) {
		hot = hovered
	}
	if hot < 0 {
		hot = focused
	}
	labels := l.names(c, pts, xy, radius)

	box.Draw(func(p *ui.Painter, r ui.Rect) {
		// The chart was laid out at the size of the last frame.
		if r.W != b.W || r.H != b.H {
			c.Invalidate()
		}
		at := func(q ui.Rect) ui.Rect { return ui.Rect{X: r.X + q.X, Y: r.Y + q.Y, W: q.W, H: q.H} }
		plot := at(l.plot)
		p.Clip(plot, t.Radius+2, func() {
			for q := range 4 {
				tint := float32(0.08)
				if t.Dark {
					tint = 0.12
				}
				p.Fill(at(l.quadrant(q)), Color(t, q).Alpha(tint), 0)
			}
		})
		p.Line(plot.X+plot.W/2, plot.Y, plot.X+plot.W/2, plot.Y+plot.H, 2, t.Background)
		p.Line(plot.X, plot.Y+plot.H/2, plot.X+plot.W, plot.Y+plot.H/2, 2, t.Background)
		for q, title := range ch.Quadrants {
			if tb := at(l.titles[q]); title != "" {
				p.RichText(tb.X, tb.Y, tb.W+1, ui.Span{Text: title, Weight: 600, Color: ink(t, q)})
			}
		}
		if ch.Title != "" {
			ts := ui.Span{Text: ch.Title, Size: t.FontSize + 1, Weight: 600, Color: t.Text}
			w, _ := p.MeasureText(0, ts)
			p.RichText(plot.X+(plot.W-w)/2, r.Y+t.Space(1), 0, ts)
		}
		gap := t.Space(2)
		axis(p, t, ch.X, plot.X, plot.X+plot.W, plot.Y+plot.H+gap)
		cx, cy := plot.X-gap-l.line/2, plot.Y+plot.H/2
		p.Transform(ui.Rotation(-90), cx, cy, func() {
			axis(p, t, ch.Y, cx-plot.H/2, cx+plot.H/2, cy-l.line/2)
		})

		for i := range pts {
			if i != hot {
				dot(p, t, pts[i], r.X+xy[i][0], r.Y+xy[i][1], radius[i], false)
			}
		}
		if hot >= 0 {
			dot(p, t, pts[hot], r.X+xy[hot][0], r.Y+xy[hot][1], radius[hot], true)
		}
		for i, lb := range labels {
			if lb.W == 0 {
				continue
			}
			ns := ui.Span{Text: pts[i].Name, Size: t.FontSize - 1, Color: t.Text}
			if i == hot {
				ns.Weight = 600
			}
			p.RichText(r.X+lb.X, r.Y+lb.Y, 0, ns)
		}
		if focused >= 0 {
			x, y, d := r.X+xy[focused][0], r.Y+xy[focused][1], radius[focused]+3
			p.FocusRing(ui.Rect{X: x - d, Y: y - d, W: 2 * d, H: 2 * d}, [4]float32{d, d, d, d})
		}
		if hot >= 0 {
			pt := pts[hot]
			q := pt.Quadrant()
			card(p, t, r, r.X+xy[hot][0], r.Y+xy[hot][1], radius[hot]+t.Space(2), pt, ch.Quadrants[q], dotColor(t, pt))
		}
	})
	return box
}

// arrows returns how far the arrow keys pressed on e move its point: a
// hundredth, or a tenth with Shift.
func arrows(e ui.Element) (dx, dy float64) {
	for _, step := range []struct {
		mods ui.Modifiers
		by   float64
	}{{0, 0.01}, {ui.Shift, 0.1}} {
		switch {
		case e.Shortcut(step.mods, ui.KeyLeft):
			dx -= step.by
		case e.Shortcut(step.mods, ui.KeyRight):
			dx += step.by
		case e.Shortcut(step.mods, ui.KeyDown):
			dy -= step.by
		case e.Shortcut(step.mods, ui.KeyUp):
			dy += step.by
		}
	}
	return dx, dy
}

// hundredths returns v within 0 to 1, rounded to a hundredth, so that
// steps of the keys land on round values.
func hundredths(v float64) float64 { return math.Round(max(0, min(1, v))*100) / 100 }

// reading reads out pt for screen readers: its x, its y and its quadrant.
func reading(ch Chart, pt Point) string {
	s := "x " + decimal(pt.X) + ", y " + decimal(pt.Y)
	if q := ch.Quadrants[pt.Quadrant()]; q != "" {
		s += ", " + q
	}
	return s
}

func decimal(v float64) string { return strconv.FormatFloat(v, 'f', 2, 64) }

func dotColor(t *ui.Theme, pt Point) ui.Color {
	if pt.Color != ui.Transparent {
		return pt.Color
	}
	return Color(t, pt.Quadrant())
}

// dot paints pt's dot of radius at x, y, with a shadow when hot.
func dot(p *ui.Painter, t *ui.Theme, pt Point, x, y, radius float32, hot bool) {
	halo := radius + 2
	ring := ui.Rect{X: x - halo, Y: y - halo, W: 2 * halo, H: 2 * halo}
	if hot {
		p.Shadow(ring, halo, 0, 1, 6, 0, ui.RGBA(0, 0, 0, 0.25))
	}
	p.Fill(ring, t.Background, halo)
	p.Fill(ui.Rect{X: x - radius, Y: y - radius, W: 2 * radius, H: 2 * radius}, dotColor(t, pt), radius)
}

// layout is a chart laid out in its box: the plot, the boxes of the
// quadrants' titles, in the order of Chart.Quadrants, and the height of a
// line of the axes' labels.
type layout struct {
	plot   ui.Rect
	titles [4]ui.Rect
	line   float32
}

// lay lays out ch in a box the size of b.
func lay(c *ui.Context, ch Chart, b ui.Rect) layout {
	t := c.Theme()
	var l layout
	_, l.line = c.MeasureText(0, ui.Span{Text: "M", Size: t.FontSize - 1})
	edge := t.Space(2)
	top, left, bottom := edge, edge, edge
	if ch.Title != "" {
		_, h := c.MeasureText(0, ui.Span{Text: ch.Title, Size: t.FontSize + 1, Weight: 600})
		top = t.Space(1) + h + t.Space(3)
	}
	if ch.Y != (Axis{}) {
		left = l.line + t.Space(4)
	}
	if ch.X != (Axis{}) {
		bottom = l.line + t.Space(4)
	}
	l.plot = ui.Rect{X: left, Y: top, W: max(1, b.W-left-edge), H: max(1, b.H-top-bottom)}
	for q, title := range ch.Quadrants {
		if title == "" {
			continue
		}
		r := l.quadrant(q)
		maxW := max(1, r.W-t.Space(4))
		w, h := c.MeasureText(maxW, ui.Span{Text: title, Weight: 600})
		l.titles[q] = ui.Rect{X: r.X + (r.W-w)/2, Y: r.Y + t.Space(2.5), W: w, H: h}
	}
	return l
}

// quadrant returns the box of quadrant q, an index in Chart.Quadrants.
func (l *layout) quadrant(q int) ui.Rect {
	p := l.plot
	r := ui.Rect{X: p.X, Y: p.Y, W: p.W / 2, H: p.H / 2}
	if q == 0 || q == 3 {
		r.X += p.W / 2
	}
	if q >= 2 {
		r.Y += p.H / 2
	}
	return r
}

// at returns where pt is in the box.
func (l *layout) at(pt Point) (float32, float32) {
	return l.plot.X + float32(pt.X)*l.plot.W, l.plot.Y + float32(1-pt.Y)*l.plot.H
}

// value returns the x and y at x, y of the box, within 0 to 1.
func (l *layout) value(x, y float32) (float64, float64) {
	return max(0, min(1, float64((x-l.plot.X)/l.plot.W))), max(0, min(1, float64(1-(y-l.plot.Y)/l.plot.H)))
}

// names places the name of each of pts beside its dot at xy, of radius,
// on the side where it covers the least of the names placed before it,
// the other dots and the quadrants' titles, and stays inside the plot.
// It returns the boxes of the names, empty for a point without one.
func (l *layout) names(c *ui.Context, pts []Point, xy [][2]float32, radius []float32) []ui.Rect {
	t := c.Theme()
	taken := make([]ui.Rect, 0, 4+len(pts))
	for _, r := range l.titles {
		if r.W > 0 {
			taken = append(taken, r)
		}
	}
	dots := make([]ui.Rect, len(pts))
	for i, p := range xy {
		d := radius[i] + 2
		dots[i] = ui.Rect{X: p[0] - d, Y: p[1] - d, W: 2 * d, H: 2 * d}
	}
	out := make([]ui.Rect, len(pts))
	for i, pt := range pts {
		if pt.Name == "" {
			continue
		}
		w, h := c.MeasureText(0, ui.Span{Text: pt.Name, Size: t.FontSize - 1})
		x, y := xy[i][0], xy[i][1]
		d := radius[i] + t.Space(2)
		e := d * 0.7
		sides := [...][2]float32{
			{x + d, y - h/2}, {x - d - w, y - h/2},
			{x - w/2, y - d - h}, {x - w/2, y + d},
			{x + e, y - e - h}, {x + e, y + e},
			{x - e - w, y - e - h}, {x - e - w, y + e},
		}
		least := float32(math.Inf(1))
		for _, sd := range sides {
			r := ui.Rect{X: sd[0], Y: sd[1], W: w, H: h}
			// Leaving the plot costs more than covering a name, which
			// still reads where it peeks out.
			cost := 4 * (w*h - overlap(r, l.plot))
			for _, o := range taken {
				cost += overlap(r, o)
			}
			for k, o := range dots {
				if k != i {
					cost += overlap(r, o)
				}
			}
			// Below a DIP squared, a cost is the rounding of the areas.
			if cost < least-1 {
				out[i], least = r, cost
			}
			if cost < 1 {
				break
			}
		}
		taken = append(taken, out[i])
	}
	return out
}

// overlap returns the area a and b share.
func overlap(a, b ui.Rect) float32 {
	w := min(a.X+a.W, b.X+b.W) - max(a.X, b.X)
	h := min(a.Y+a.H, b.Y+b.H) - max(a.Y, b.Y)
	if w <= 0 || h <= 0 {
		return 0
	}
	return w * h
}

// axis paints a's labels along the line from x0 to x1, their tops at y:
// the low one from x0, the high one up to x1, and an arrow from one to
// the other between them.
func axis(p *ui.Painter, t *ui.Theme, a Axis, x0, x1, y float32) {
	small := t.FontSize - 1
	lw, lh := p.MeasureText(0, ui.Span{Text: a.Low, Size: small})
	p.Text(x0, y, a.Low, small, t.TextMuted)
	if a.High == "" {
		return
	}
	hw, _ := p.MeasureText(0, ui.Span{Text: a.High, Size: small})
	p.Text(x1-hw, y, a.High, small, t.TextMuted)
	gap := t.Space(2)
	from, to := x0+lw+gap, x1-hw-gap
	if a.Low == "" {
		from = x0
	}
	if to-from < t.Space(6) {
		return
	}
	my, head := y+lh/2, t.Space(1.25)
	line := t.TextMuted.Alpha(0.5)
	p.Line(from, my, to-1, my, 1, line)
	var tip ui.Path
	tip.MoveTo(to, my).LineTo(to-head*1.6, my-head).LineTo(to-head*1.6, my+head).Close()
	p.FillPath(&tip, line)
}

// card paints the card of pt at x, y of r, gap from it, below and to the
// right where there is room: its name, its quadrant by a swatch of col,
// unless quadrant is "", and its x and y.
func card(p *ui.Painter, t *ui.Theme, r ui.Rect, x, y, gap float32, pt Point, quadrant string, col ui.Color) {
	small := t.FontSize - 2
	lines := [][]ui.Span{{{Text: pt.Name, Size: t.FontSize - 1, Weight: 600, Color: t.Text}}}
	if quadrant != "" {
		lines = append(lines, []ui.Span{{Text: quadrant, Size: small, Color: t.TextMuted}})
	}
	lines = append(lines, []ui.Span{
		{Text: "x  ", Size: small, Color: t.TextMuted},
		{Text: decimal(pt.X), Size: small, Color: t.Text, Features: "tnum"},
		{Text: "    y  ", Size: small, Color: t.TextMuted},
		{Text: decimal(pt.Y), Size: small, Color: t.Text, Features: "tnum"},
	})
	pad, lead, swatch := t.Space(2.5), t.Space(0.5), t.Space(2)
	maxW := t.Space(70)
	w, h := float32(0), float32(0)
	sizes := make([][2]float32, len(lines))
	for i, l := range lines {
		lw, lh := p.MeasureText(maxW, l...)
		if quadrant != "" && i == 1 {
			lw += swatch + t.Space(1.5)
		}
		sizes[i] = [2]float32{lw, lh}
		w, h = max(w, lw), h+lh
	}
	w, h = w+2*pad, h+2*pad+lead*float32(len(lines)-1)
	cx, cy := x+gap, y+gap
	if cx+w > r.X+r.W {
		cx = x - gap - w
	}
	if cy+h > r.Y+r.H {
		cy = y - gap - h
	}
	cx, cy = max(r.X, cx), max(r.Y, cy)
	box := ui.Rect{X: cx, Y: cy, W: w, H: h}
	radius := t.Radius + 2
	shadow := ui.RGBA(0, 0, 0, 0.18)
	if t.Dark {
		shadow = ui.RGBA(0, 0, 0, 0.4)
	}
	p.Shadow(box, radius, 0, 2, 10, 0, shadow)
	p.Fill(box, t.Background, radius)
	p.Stroke(box, t.Border, radius, 1)
	ty := cy + pad
	for i, l := range lines {
		tx := cx + pad
		if quadrant != "" && i == 1 {
			p.Fill(ui.Rect{X: tx, Y: ty + (sizes[i][1]-swatch)/2, W: swatch, H: swatch}, col, swatch/2)
			tx += swatch + t.Space(1.5)
		}
		p.RichText(tx, ty, maxW, l...)
		ty += sizes[i][1] + lead
	}
}

// Error is a mistake in the source: the line it is on, from 1, and what
// is wrong.
type Error struct {
	Line int
	Msg  string
}

func (e Error) Error() string { return fmt.Sprintf("line %d: %s", e.Line, e.Msg) }

// Parse reads Mermaid's quadrantChart text into a chart and its points,
// in order, and returns what is wrong in src by line. After the line
// quadrantChart comes a statement a line; lines starting with %% are
// comments:
//
//	title Text                  the chart's title
//	x-axis Low --> High         the labels of the x axis, or its low one only
//	y-axis Low --> High         and those of the y axis
//	quadrant-1 Text             the title of the top right quadrant; 2 is
//	                            the top left, 3 the bottom left, 4 the
//	                            bottom right
//	Name: [0.3, 0.6]            a point, its x then its y, from 0 to 1
//	Name: [0.3, 0.6] color: #e5484d
//	                            one in a color of its own
//
// Text and names may be in double quotes. Mermaid's other styles of a
// point, radius, stroke-color and stroke-width, are skipped. A line with
// a mistake is left out.
func Parse(src string) (Chart, []Point, []Error) {
	var ch Chart
	var points []Point
	var errs []Error
	started := false
	for i, line := range strings.Split(src, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "%%") {
			continue
		}
		if !started {
			started = true
			if line == "quadrantChart" {
				continue
			}
			errs = append(errs, Error{i + 1, "the chart starts with quadrantChart"})
		}
		key, rest, _ := strings.Cut(line, " ")
		rest = strings.TrimSpace(rest)
		switch key {
		case "title":
			ch.Title = unquote(rest)
		case "x-axis":
			ch.X = axisOf(rest)
		case "y-axis":
			ch.Y = axisOf(rest)
		case "quadrant-1", "quadrant-2", "quadrant-3", "quadrant-4":
			ch.Quadrants[key[len(key)-1]-'1'] = unquote(rest)
		default:
			pt, msg := point(line)
			if msg != "" {
				errs = append(errs, Error{i + 1, msg})
				continue
			}
			points = append(points, pt)
		}
	}
	return ch, points, errs
}

func axisOf(s string) Axis {
	low, high, _ := strings.Cut(s, "-->")
	return Axis{unquote(strings.TrimSpace(low)), unquote(strings.TrimSpace(high))}
}

func unquote(s string) string {
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}

// point reads the point of line, or returns what is wrong with it.
func point(line string) (Point, string) {
	var pt Point
	open := strings.IndexByte(line, '[')
	if open < 0 {
		return pt, fmt.Sprintf("%q is neither a statement nor a point", line)
	}
	name, ok := strings.CutSuffix(strings.TrimSpace(line[:open]), ":")
	if !ok {
		return pt, "a point's name ends with a colon"
	}
	if pt.Name = unquote(strings.TrimSpace(name)); pt.Name == "" {
		return pt, "a point needs a name"
	}
	end := strings.IndexByte(line[open:], ']')
	if end < 0 {
		return pt, "a point's [x, y] ends with ]"
	}
	xs, ys, ok := strings.Cut(line[open+1:open+end], ",")
	x, errX := strconv.ParseFloat(strings.TrimSpace(xs), 64)
	y, errY := strconv.ParseFloat(strings.TrimSpace(ys), 64)
	if !ok || errX != nil || errY != nil {
		return pt, "a point is at [x, y], two numbers"
	}
	if !(x >= 0 && x <= 1 && y >= 0 && y <= 1) {
		return pt, "a point's x and y are from 0 to 1"
	}
	pt.X, pt.Y = x, y
	for _, style := range strings.Split(line[open+end+1:], ",") {
		if strings.TrimSpace(style) == "" {
			continue
		}
		k, v, _ := strings.Cut(style, ":")
		switch k, v = strings.TrimSpace(k), strings.TrimSpace(v); k {
		case "color":
			c, ok := hex(v)
			if !ok {
				return pt, fmt.Sprintf("color %q is not #rgb or #rrggbb", v)
			}
			pt.Color = c
		case "radius", "stroke-color", "stroke-width":
		default:
			return pt, fmt.Sprintf("a point has no style %q", k)
		}
	}
	return pt, ""
}

// hex returns the color of s, #rgb or #rrggbb.
func hex(s string) (ui.Color, bool) {
	if len(s) != 4 && len(s) != 7 || s[0] != '#' {
		return ui.Color{}, false
	}
	for _, r := range s[1:] {
		if !strings.ContainsRune("0123456789abcdefABCDEF", r) {
			return ui.Color{}, false
		}
	}
	return ui.Hex(s), true
}

// Format writes ch and points as Mermaid's quadrantChart text, which
// Parse reads back: x and y to two decimals, and the color of a point
// that has its own.
func Format(ch Chart, points []Point) string {
	var b strings.Builder
	b.WriteString("quadrantChart\n")
	if ch.Title != "" {
		fmt.Fprintf(&b, "    title %s\n", ch.Title)
	}
	for _, a := range []struct {
		key string
		Axis
	}{{"x-axis", ch.X}, {"y-axis", ch.Y}} {
		switch {
		case a.High != "":
			fmt.Fprintf(&b, "    %s %s --> %s\n", a.key, a.Low, a.High)
		case a.Low != "":
			fmt.Fprintf(&b, "    %s %s\n", a.key, a.Low)
		}
	}
	for q, title := range ch.Quadrants {
		if title != "" {
			fmt.Fprintf(&b, "    quadrant-%d %s\n", q+1, title)
		}
	}
	for _, pt := range points {
		fmt.Fprintf(&b, "    %s: [%s, %s]", pt.Name, decimal(pt.X), decimal(pt.Y))
		if c := pt.Color; c != ui.Transparent {
			fmt.Fprintf(&b, " color: #%02x%02x%02x", c.R, c.G, c.B)
		}
		b.WriteByte('\n')
	}
	return b.String()
}
