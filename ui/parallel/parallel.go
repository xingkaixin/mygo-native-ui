// Package parallel shows rows of many dimensions as parallel coordinates,
// as tools for exploring data do: an axis for each dimension, from left to
// right, with its name on top and its ticks along it, and each row a line
// across them through its value on each, straight or, with Smooth,
// curving level into every axis. Numbers range over the rows' values, out
// to round ticks; categories are points evenly spaced up their axis, each
// labeled. The lines take the colors of one dimension's values, from low
// to high on a sequential scale, or all the accent color.
//
// Dragging along an axis brushes a stretch of it: the rows within every
// brush pass, drawn in color on top, and the others fade to gray beneath
// them, and a count above reads "128 of 1,000". Dragging inside a brush
// moves it and by its ends resizes it; a click on the axis outside it, or
// a double click, clears it, and Esc clears them all. A name dragged
// sideways moves its axis among the others, and a click on it flips the
// axis. The line nearest the pointer lights, with its value at each axis
// and its name above. The app reads which rows pass in the frame they
// change:
//
//	ev := parallel.View(c, &app.chart, dims, rows, 5) // colored by dims[5]
//	if ev.Changed() {
//		app.shown = ev.Passing()
//	}
//
// The lines are drawn on the CPU, on as many cores as there are, and kept
// until the rows, the size, the order or the brushes change, so that
// thousands of rows brush at interactive rates.
package parallel

import (
	"image"
	"math"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/chart"
)

// Dim is a dimension of the rows, shown on an axis: its name, on top of
// it, and the unit that follows its values where they are read out. With
// Categories, a value is the index of its category, and they show as
// points evenly spaced up the axis, the first at the bottom; without,
// values are numbers, written by Format, or with as many decimals as the
// ticks need when it is nil.
type Dim struct {
	Name       string
	Unit       string
	Format     func(v float64) string
	Categories []string
}

// Row is a record: its value for each dimension, in their order, NaN where
// it has none, which breaks its line there, and its name, which shows as
// the pointer lights it.
type Row struct {
	Name   string
	Values []float64
}

// State is what a chart keeps: the order of its axes, those flipped, the
// brushes on them, and the press dragging one. Smooth, which the app sets,
// draws the rows as curves. Keep it in the app's state, one for each
// chart; the zero State shows the dimensions in their order, unbrushed,
// and starts over when given another number of them.
type State struct {
	Smooth bool

	order   []int
	flipped []bool
	brushes []brush
	drag    drag
	// passing is the rows that passed in the last frame, for Events, and
	// pass, ys, colors and drawn hold what a frame computes, for their
	// storage.
	passing, pass, drawn []int
	ys                   []float32
	colors               []ui.Color
}

// brush is the stretch of a dimension's values from lo to hi that passes,
// when on.
type brush struct {
	lo, hi float64
	on     bool
}

// What a press on an axis does as it drags: brush from where it began,
// move the brush, move one of its ends, or nothing, after a double click
// cleared it.
const (
	fresh = iota
	moving
	sizing
	resting
)

// drag is a press on an axis or a name while it lasts. dim is the
// dimension's index plus 1, so that the zero drag is none, and dist how
// far the pointer went. On an axis, at is the value pressed, and a and b
// the brush's ends as the press began, b the end that moves when sizing;
// on a name, x is where its axis follows the pointer.
type drag struct {
	dim  int
	name bool
	mode int
	at   float64
	a, b float64
	x    float32
	dist float32
}

// moveAfter is how far, in DIPs, a press goes before it drags rather than
// clicks.
const moveAfter = 3

// fit sizes s for n dimensions, starting over for another number.
func (s *State) fit(n int) {
	if len(s.order) == n {
		return
	}
	s.order, s.flipped, s.brushes, s.drag = make([]int, n), make([]bool, n), make([]brush, n), drag{}
	for i := range s.order {
		s.order[i] = i
	}
}

// passes reports whether r is within every brush.
func (s *State) passes(r Row) bool {
	for d, b := range s.brushes {
		if !b.on {
			continue
		}
		v := math.NaN()
		if d < len(r.Values) {
			v = r.Values[d]
		}
		if !(v >= b.lo && v <= b.hi) {
			return false
		}
	}
	return true
}

// Events is what a chart's brushes did in a frame.
type Events struct {
	passing []int
	changed bool
}

// Passing returns the indexes of the rows within every brush, in order:
// all of them while no axis is brushed. It is a new slice each time they
// change, so one the app keeps stays as it was.
func (e Events) Passing() []int { return e.passing }

// Changed reports whether the rows that pass changed in this frame, as a
// brush moved, or the rows did.
func (e Events) Changed() bool { return e.changed }

// axis is a dimension laid out: its values from lo at the bottom to hi at
// the top, or the other way up when flipped, the extent of the rows'
// values, its ticks, and the decimals its values read out with.
type axis struct {
	dim      *Dim
	lo, hi   float64
	min, max float64
	flip     bool
	ticks    []chart.Tick
	decimals int
}

// newAxis lays out dimension d of rows with about count ticks.
func newAxis(dim *Dim, rows []Row, d, count int, flip bool) axis {
	a := axis{dim: dim, flip: flip, min: math.Inf(1), max: math.Inf(-1)}
	for _, r := range rows {
		if d < len(r.Values) && !math.IsNaN(r.Values[d]) && !math.IsInf(r.Values[d], 0) {
			a.min, a.max = min(a.min, r.Values[d]), max(a.max, r.Values[d])
		}
	}
	if n := len(dim.Categories); n > 0 {
		a.lo, a.hi = -0.5, float64(n)-0.5
		if a.min > a.max {
			a.min, a.max = 0, float64(n-1)
		}
		for i, name := range dim.Categories {
			a.ticks = append(a.ticks, chart.Tick{At: a.at(float64(i)), Value: float64(i), Label: name})
		}
		return a
	}
	if a.min > a.max {
		a.min, a.max = 0, 1
	}
	lo, hi := a.min, a.max
	if lo == hi {
		lo, hi = lo-1, hi+1
	}
	scale := chart.NewLinear(lo, hi).Nice(count)
	scale.Format = dim.Format
	a.lo, a.hi, a.ticks = scale.Lo, scale.Hi, scale.Ticks(count)
	for i := range a.ticks {
		a.ticks[i].At = a.at(a.ticks[i].Value)
	}
	if len(a.ticks) > 1 {
		step := a.ticks[1].Value - a.ticks[0].Value
		a.decimals = max(0, int(math.Ceil(-math.Log10(step)-1e-9))) + 1
	}
	return a
}

// at returns where v is up the axis, from 0 at the bottom to 1 at the top.
func (a *axis) at(v float64) float32 {
	f := float32((v - a.lo) / (a.hi - a.lo))
	if a.flip {
		return 1 - f
	}
	return f
}

// value returns the value at f up the axis, within its ends.
func (a *axis) value(f float32) float64 {
	f = max(0, min(1, f))
	if a.flip {
		f = 1 - f
	}
	return a.lo + float64(f)*(a.hi-a.lo)
}

// unit returns where v is among the rows' values, from 0 the lowest to 1
// the highest, for its color.
func (a *axis) unit(v float64) float32 {
	if a.max <= a.min {
		return 0.5
	}
	return float32(max(0, min(1, (v-a.min)/(a.max-a.min))))
}

// format writes v as the dimension's values read out: its category, or
// its number and unit.
func (a *axis) format(v float64) string {
	if n := len(a.dim.Categories); n > 0 {
		return a.dim.Categories[max(0, min(n-1, int(math.Round(v))))]
	}
	s := ""
	if a.dim.Format != nil {
		s = a.dim.Format(v)
	} else {
		s = strconv.FormatFloat(v, 'f', a.decimals, 64)
		if strings.Contains(s, ".") {
			s = strings.TrimRight(strings.TrimRight(s, "0"), ".")
		}
	}
	if a.dim.Unit != "" {
		s += " " + a.dim.Unit
	}
	return s
}

// key tells a name's element, or with axis set its axis's, from the
// others, by its dimension, so that a press follows the axis as it moves.
type key struct {
	dim  int
	axis bool
}

// View creates the chart of rows across dims, colored by the values of
// dims[color], or all in the accent color when color is -1. It fills its
// parent. The Events returned tell which rows pass.
func View(c *ui.Context, s *State, dims []Dim, rows []Row, color int) Events {
	t := c.Theme()
	nd := len(dims)
	s.fit(nd)
	small, nameSize := t.FontSize-2, t.FontSize-1
	_, th := c.MeasureText(0, ui.Span{Text: "0", Size: small})
	_, nh := c.MeasureText(0, ui.Span{Text: "M", Size: nameSize, Weight: 600})
	nameH, hitW, grip := nh+t.Space(2), t.Space(6), t.Space(1.5)

	var ev Events
	// The line above the chart counts the rows that pass and names the one
	// under the pointer, which the chart learns as it is built: it is built
	// first, at the bottom of a column laid out upwards.
	ui.Column(c).Fill().Gap(t.Space(3)).Reverse().Children(func() {
		box := ui.Box(c).Grow(1).AlignSelf(ui.Stretch).Focusable().Label("Parallel coordinates")
		b := box.LayoutBounds()
		top, bottom := nameH+t.Space(4), b.H-th/2-t.Space(1)
		plotH := max(1, bottom-top)
		yAt := func(f float32) float32 { return top + (1-f)*plotH }

		axes := make([]axis, nd)
		labelW := float32(0)
		for d := range dims {
			axes[d] = newAxis(&dims[d], rows, d, max(2, int(plotH/56)), s.flipped[d])
			for _, tk := range axes[d].ticks {
				w, _ := c.MeasureText(0, ui.Span{Text: tk.Label, Size: small})
				labelW = max(labelW, w)
			}
		}
		arrowW := t.Space(2.5)
		headW := make([]float32, nd)
		for d, dm := range dims {
			w, _ := c.MeasureText(0, ui.Span{Text: dm.Name, Size: nameSize, Weight: 600})
			if dm.Unit != "" {
				uw, _ := c.MeasureText(0, ui.Span{Text: dm.Unit, Size: small})
				w += uw + t.Space(1)
			}
			headW[d] = w + t.Space(1) + arrowW + t.Space(4)
		}
		x0, x1 := t.Space(2), max(t.Space(2), b.W-t.Space(2))
		if nd > 0 {
			x0 = max(labelW+t.Space(3), headW[s.order[0]]/2)
			x1 = max(x0, b.W-max(t.Space(2), headW[s.order[nd-1]]/2))
		}
		step := float32(0)
		if nd > 1 {
			step = (x1 - x0) / float32(nd-1)
		}
		slot := func(k int) float32 {
			if nd == 1 {
				return (x0 + x1) / 2
			}
			return x0 + float32(k)*step
		}

		px, py, over := box.PointerPosition()
		lit := -1 // the dimension whose axis the pointer is over or brushes
		box.Children(func() {
			for k, d := range s.order {
				x := slot(k)
				if s.drag.name && s.drag.dim == d+1 && s.drag.dist >= moveAfter {
					x = max(x0, min(x1, s.drag.x))
				}
				w := headW[d]
				if nd > 1 {
					w = min(w, step-t.Space(2))
				}
				s.name(c, &dims[d], d, x, w, nameH, arrowW)
				if s.brushAxis(c, &axes[d], d, x, hitW, grip, top, plotH, py) {
					lit = d
				}
			}
		})
		if s.drag.name && s.drag.dist >= moveAfter && nd > 1 {
			from := slices.Index(s.order, s.drag.dim-1)
			to := int(math.Round(float64((max(x0, min(x1, s.drag.x)) - x0) / step)))
			if from >= 0 && to != from {
				s.order = slices.Insert(slices.Delete(s.order, from, from+1), to, s.drag.dim-1)
			}
		}
		if slices.ContainsFunc(s.brushes, func(b brush) bool { return b.on }) && box.Shortcut(0, ui.KeyEscape) {
			clear(s.brushes)
		}

		xs := make([]float32, nd)
		for k, d := range s.order {
			xs[k] = slot(k)
			if s.drag.name && s.drag.dim == d+1 && s.drag.dist >= moveAfter {
				xs[k] = max(x0, min(x1, s.drag.x))
			}
		}
		ys := s.ys[:0]
		for _, r := range rows {
			for _, d := range s.order {
				// Inf marks a value missing, rather than NaN, so that the
				// lines' cache compares equal.
				y := float32(math.Inf(1))
				if d < len(r.Values) && !math.IsNaN(r.Values[d]) && !math.IsInf(r.Values[d], 0) {
					y = yAt(axes[d].at(r.Values[d]))
				}
				ys = append(ys, y)
			}
		}
		s.ys = ys

		pass := s.pass[:0]
		for i, r := range rows {
			if s.passes(r) {
				pass = append(pass, i)
			}
		}
		s.pass = pass
		if s.passing == nil || !slices.Equal(pass, s.passing) {
			s.passing, ev.changed = append([]int{}, pass...), true
		}
		ev.passing = s.passing

		// Fewer lines show stronger: the lines' opacities fall as their
		// overlaps add up.
		alpha := float32(max(0.18, min(0.9, 10/math.Sqrt(float64(max(1, len(pass)))))))
		fade := t.TextMuted.Alpha(float32(max(0.04, min(0.2, 2/math.Sqrt(float64(max(1, len(rows)-len(pass))))))))
		colors, drawn := s.colors[:0], s.drawn[:0]
		var colorAxis *axis
		if color >= 0 && color < nd {
			colorAxis = &axes[color]
		}
		k := 0
		for i, r := range rows {
			if k < len(pass) && pass[k] == i {
				k++
				colors = append(colors, rowColor(t, colorAxis, color, r).Alpha(alpha))
				continue
			}
			colors = append(colors, fade)
			drawn = append(drawn, i)
		}
		drawn = append(drawn, pass...)
		s.colors, s.drawn = colors, drawn

		// Over an axis, the pointer is there to brush it.
		hovered := -1
		if over && s.drag.dim == 0 && lit < 0 && nd > 1 && py >= top-grip && py <= bottom+grip && px >= xs[0] && px <= xs[nd-1] {
			g := max(0, min(nd-2, int((px-xs[0])/step)))
			u, gw := (px-xs[g])/step, step
			best := t.Space(2)
			for _, i := range pass {
				a, b := ys[i*nd+g], ys[i*nd+g+1]
				if math.IsInf(float64(a), 0) || math.IsInf(float64(b), 0) {
					continue
				}
				y, dy := curve(a, b, u, s.Smooth)
				slope := float64(dy / gw)
				if d := abs(py-y) / float32(math.Sqrt(1+slope*slope)); d < best {
					best, hovered = d, i
				}
			}
		}
		if s.drag.dim > 0 && !s.drag.name {
			lit = s.drag.dim - 1
		}

		box.Value(number(len(pass)) + " of " + number(len(rows)) + " rows pass")
		ras := ui.Local(box, "parallel.lines", func() lines { return lines{} })
		lineW := float32(1)
		if len(pass) < 100 {
			lineW = 1.5
		}
		smooth := s.Smooth
		brushes := s.brushes
		order := s.order
		dragged := -1
		if s.drag.name && s.drag.dist >= moveAfter {
			dragged = s.drag.dim - 1
		}
		box.Draw(func(p *ui.Painter, r ui.Rect) {
			if r.W != b.W || r.H != b.H {
				c.Invalidate()
			}
			// The bitmap covers r as Painter.Image places it, on whole
			// pixels.
			sc := p.Scale()
			bx, by := round(r.X*sc), round(r.Y*sc)
			f := frame{
				w: int(round((r.X+r.W)*sc) - bx), h: int(round((r.Y+r.H)*sc) - by),
				scale: sc, ox: r.X*sc - bx, oy: r.Y*sc - by, width: lineW, smooth: smooth,
			}
			if bm := ras.draw(f, xs, ys, colors, drawn); bm != nil {
				p.Image(bm, r, ui.FillBox)
			}
			halo := t.Background.Alpha(0.8)
			for k, d := range order {
				a := &axes[d]
				x := r.X + xs[k]
				axisColor := t.TextMuted
				if d == dragged {
					axisColor = t.Accent
				}
				p.Line(x, r.Y+top, x, r.Y+bottom, 1, axisColor)
				for _, tk := range a.ticks {
					y := r.Y + yAt(tk.At)
					w, _ := p.MeasureText(0, ui.Span{Text: tk.Label, Size: small})
					p.Line(x-t.Space(1), y, x, y, 1, axisColor)
					lx := x - t.Space(1.5) - w
					p.Fill(ui.Rect{X: lx - 2, Y: y - th/2, W: w + 4, H: th}, halo, 3)
					p.Text(lx, y-th/2, tk.Label, small, t.TextMuted)
				}
				br := brushes[d]
				if !br.on {
					continue
				}
				// A brush outlives the rows, whose ends may since have moved
				// in past it.
				ya := max(r.Y+top, min(r.Y+bottom, r.Y+yAt(a.at(br.lo))))
				yb := max(r.Y+top, min(r.Y+bottom, r.Y+yAt(a.at(br.hi))))
				bw := t.Space(3)
				rect := ui.Rect{X: x - bw/2, Y: min(ya, yb), W: bw, H: max(2, abs(yb-ya))}
				p.Fill(rect, t.Text.Alpha(0.1), 3)
				p.Stroke(rect, t.Text, 3, 1.25)
				if d == lit {
					for _, v := range []float64{br.lo, br.hi} {
						tag(p, t, x, bw/2+t.Space(1.5), r.Y+yAt(a.at(v)), a.format(v), small, th, t.Accent, t.AccentText, r)
					}
				}
			}
			if hovered < 0 {
				return
			}
			pts := path(r.X, r.Y, xs, ys[hovered*nd:(hovered+1)*nd], smooth)
			col := colors[hovered]
			col.A = 255
			p.StrokePolyline(pts, lineW+3, t.Background)
			p.StrokePolyline(pts, lineW+0.5, t.Text)
			for k, d := range order {
				y := ys[hovered*nd+k]
				if math.IsInf(float64(y), 0) {
					continue
				}
				x, y := r.X+xs[k], r.Y+y
				p.Fill(ui.Rect{X: x - 4, Y: y - 4, W: 8, H: 8}, t.Background, 4)
				p.Fill(ui.Rect{X: x - 2.5, Y: y - 2.5, W: 5, H: 5}, col, 2.5)
				tag(p, t, x, t.Space(2), y, axes[d].format(rows[hovered].Values[d]), small, th, t.Text, t.Background, r)
			}
		})

		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, number(len(pass))+" of "+number(len(rows))).Bold().FontFeatures("tnum")
			line, lineColor := "Drag along an axis to filter, a name to move its axis, click a name to flip it", t.TextMuted
			switch {
			case hovered >= 0 && rows[hovered].Name != "":
				line, lineColor = rows[hovered].Name, t.Text
			case hovered >= 0:
				line = "Row " + number(hovered+1)
			case len(pass) < len(rows):
				line = "Click an axis outside its brush or press Esc to clear"
			}
			ui.Text(c, line).TextColor(lineColor).SingleLine().Grow(1).Shrink(1)
			if colorAxis != nil {
				legend(c, colorAxis, t.Dark)
			}
		})
	})
	return ev
}

// name builds the name of dimension d over its axis at x, w wide and h
// high, which a drag moves among the others and a click flips.
func (s *State) name(c *ui.Context, dim *Dim, d int, x, w, h, arrowW float32) {
	t := c.Theme()
	el := ui.Row(c.Key(key{dim: d})).Absolute().Left(x-w/2).Top(0).Size(w, h).Padding(0, t.Space(2)).
		Gap(t.Space(1)).AlignItems(ui.Center).Justify(ui.Center).Radius(t.Space(1.5)).
		Role(ui.RoleButton).Label(dim.Name)
	mine := s.drag.name && s.drag.dim == d+1
	dx, dy, pressing := el.Dragged()
	clicked := el.Clicked()
	switch {
	case pressing && !mine && s.drag.dim == 0:
		s.drag = drag{dim: d + 1, name: true, x: x + dx, dist: abs(dx) + abs(dy)}
	case pressing && mine:
		s.drag.x += dx
		s.drag.dist += abs(dx) + abs(dy)
	case mine:
		if clicked && s.drag.dist < moveAfter {
			s.flipped[d] = !s.flipped[d]
		}
		s.drag = drag{}
	case clicked:
		s.flipped[d] = !s.flipped[d]
	}
	if mine && s.drag.dist >= moveAfter {
		el.Cursor(ui.CursorGrabbing).Background(t.SurfacePressed)
	} else {
		el.Cursor(ui.CursorGrab)
		if el.Hovered() {
			el.Background(t.SurfaceHover)
		}
	}
	flipped := s.flipped[d]
	el.Children(func() {
		ui.Text(c, dim.Name).FontSize(t.FontSize - 1).FontWeight(600).SingleLine().Shrink(1)
		if dim.Unit != "" {
			ui.Text(c, dim.Unit).FontSize(t.FontSize - 2).TextColor(t.TextMuted).SingleLine()
		}
		ui.Box(c).Size(arrowW, arrowW).Draw(func(p *ui.Painter, r ui.Rect) {
			// An arrow the way the values grow.
			cx, k := r.X+r.W/2, r.H*0.32
			tip, tail, head := r.Y+r.H/2-k, r.Y+r.H/2+k, k*0.75
			if flipped {
				tip, tail, head = tail, tip, -head
			}
			p.StrokePolyline([][2]float32{{cx, tail}, {cx, tip}}, 1.25, t.TextMuted)
			p.StrokePolyline([][2]float32{{cx - k*0.7, tip + head}, {cx, tip}, {cx + k*0.7, tip + head}}, 1.25, t.TextMuted)
		})
	})
}

// brushAxis builds the axis of dimension d at x, hitW wide around it, the
// plot from top down plotH, where a press brushes, moves or resizes the
// brush or clears it; py is the pointer's height in the chart. It reports
// whether the pointer is over the axis.
func (s *State) brushAxis(c *ui.Context, a *axis, d int, x, hitW, grip, top, plotH, py float32) bool {
	hit := ui.Box(c.Key(key{dim: d, axis: true})).Absolute().Left(x-hitW/2).Top(top-grip).Size(hitW, plotH+2*grip).
		Label(a.dim.Name + " axis")
	yAt := func(v float64) float32 { return top + (1-a.at(v))*plotH }
	v := a.value(1 - (py-top)/plotH)
	br := s.brushes[d]
	// The brush's ends, top first.
	e0, e1 := br.hi, br.lo
	if yAt(br.lo) < yAt(br.hi) {
		e0, e1 = br.lo, br.hi
	}
	y0, y1 := yAt(e0), yAt(e1)
	const (
		outside = iota
		inside
		topEnd
		bottomEnd
	)
	part := outside
	switch {
	case !br.on:
	case abs(py-y0) <= grip && abs(py-y0) <= abs(py-y1):
		part = topEnd
	case abs(py-y1) <= grip:
		part = bottomEnd
	case py > y0 && py < y1:
		part = inside
	}

	mine := !s.drag.name && s.drag.dim == d+1
	dx, dy, pressing := hit.Dragged()
	clicked := hit.Clicked()
	switch {
	case hit.DoubleClicked():
		s.brushes[d], s.drag = brush{}, drag{dim: d + 1, mode: resting}
	case pressing && !mine && s.drag.dim == 0:
		s.drag = drag{dim: d + 1, at: v}
		switch part {
		case topEnd:
			s.drag.mode, s.drag.a, s.drag.b = sizing, e1, e0
		case bottomEnd:
			s.drag.mode, s.drag.a, s.drag.b = sizing, e0, e1
		case inside:
			s.drag.mode, s.drag.a, s.drag.b = moving, br.lo, br.hi
		}
	case pressing && mine:
		s.drag.dist += abs(dx) + abs(dy)
		delta := v - s.drag.at
		switch s.drag.mode {
		case fresh:
			if s.drag.dist >= moveAfter {
				s.brushes[d] = brush{min(s.drag.at, v), max(s.drag.at, v), true}
			}
		case moving:
			delta = max(a.lo-s.drag.a, min(a.hi-s.drag.b, delta))
			s.brushes[d] = brush{s.drag.a + delta, s.drag.b + delta, true}
		case sizing:
			end := max(a.lo, min(a.hi, s.drag.b+delta))
			s.brushes[d] = brush{min(s.drag.a, end), max(s.drag.a, end), true}
		}
	case mine:
		if s.drag.mode == fresh && s.drag.dist < moveAfter {
			s.brushes[d] = brush{}
		}
		s.drag = drag{}
	case clicked && part == outside:
		// A click too quick for a frame to see it pressing.
		s.brushes[d] = brush{}
	}

	mine = !s.drag.name && s.drag.dim == d+1
	switch {
	case mine && s.drag.mode == moving:
		hit.Cursor(ui.CursorGrabbing)
	case mine && s.drag.mode != resting, !mine && (part == topEnd || part == bottomEnd):
		hit.Cursor(ui.CursorResizeNS)
	case part == inside:
		hit.Cursor(ui.CursorGrab)
	default:
		hit.Cursor(ui.CursorCrosshair)
	}
	return hit.Hovered()
}

// rowColor returns the color of row r by the values of axis by, dimension
// d, or the accent color without one.
func rowColor(t *ui.Theme, by *axis, d int, r Row) ui.Color {
	if by == nil {
		return t.Accent
	}
	if d >= len(r.Values) || math.IsNaN(r.Values[d]) {
		return t.TextMuted
	}
	return shade(by.unit(r.Values[d]), t.Dark)
}

// shade returns the color at f of the lines' scale, from 0, the lowest
// value's, to 1, the highest's: from violet through red to amber, lighter
// all the way, so that the order holds in gray, and none as light as the
// light theme's background or as dark as the dark one's.
func shade(f float32, dark bool) ui.Color {
	l := 0.42 + 0.38*f
	if dark {
		l = 0.56 + 0.32*f
	}
	c := ui.Oklch(l, 0.17, float32(math.Mod(float64(295+135*f), 360)))
	// The lines are drawn in sRGB: the legend shows the same colors.
	return ui.RGB(c.R, c.G, c.B)
}

// legend shows the colors of the lines by axis a's values, from its lowest
// to its highest.
func legend(c *ui.Context, a *axis, dark bool) {
	t := c.Theme()
	small := t.FontSize - 2
	lo, hi := a.format(a.min), a.format(a.max)
	ui.Row(c).Gap(t.Space(1.5)).AlignItems(ui.Center).Children(func() {
		ui.Text(c, a.dim.Name).FontSize(small).TextColor(t.TextMuted)
		ui.Text(c, lo).FontSize(small).FontFeatures("tnum")
		stops := make([]ui.GradientStop, 9)
		for i := range stops {
			f := float32(i) / float32(len(stops)-1)
			stops[i] = ui.GradientStop{Color: shade(f, dark), At: f}
		}
		ui.Box(c).Size(t.Space(24), t.Space(2)).Draw(func(p *ui.Painter, r ui.Rect) {
			p.FillGradient(r, ui.LinearGradient{Angle: 90, Stops: stops}, t.Space(1))
		})
		ui.Text(c, hi).FontSize(small).FontFeatures("tnum")
	})
}

// tag paints label in a pill of fill centered on y, gap right of x, or
// left of it where r ends first.
func tag(p *ui.Painter, t *ui.Theme, x, gap, y float32, label string, size, th float32, fill, text ui.Color, r ui.Rect) {
	pad := t.Space(1.5)
	w, _ := p.MeasureText(0, ui.Span{Text: label, Size: size})
	w, h := w+2*pad, th+pad/2
	x += gap
	if x+w > r.X+r.W {
		x -= 2*gap + w
	}
	y = max(r.Y, min(r.Y+r.H-h, y-h/2))
	p.Fill(ui.Rect{X: x, Y: y, W: w, H: h}, fill, 3)
	p.Text(x+pad, y+pad/4, label, size, text)
}

// curve returns where a line from y0 at one axis to y1 at the next is u of
// the way across, and how fast it climbs there, per gap: straight, or
// smoothly, level at either axis.
func curve(y0, y1, u float32, smooth bool) (y, dy float32) {
	if !smooth {
		return y0 + (y1-y0)*u, y1 - y0
	}
	return y0 + (y1-y0)*u*u*(3-2*u), (y1 - y0) * 6 * u * (1 - u)
}

// curveSteps is how many straight pieces a smooth line takes between two
// axes.
const curveSteps = 12

// path returns the points of a row's line through ys at the axes at xs,
// offset by ox, oy, with Inf where it breaks, as Painter.StrokePolyline
// takes them.
func path(ox, oy float32, xs, ys []float32, smooth bool) [][2]float32 {
	var pts [][2]float32
	for k, y := range ys {
		if math.IsInf(float64(y), 0) {
			pts = append(pts, [2]float32{y, y})
			continue
		}
		if smooth && k > 0 && !math.IsInf(float64(ys[k-1]), 0) {
			for i := 1; i < curveSteps; i++ {
				u := float32(i) / curveSteps
				cy, _ := curve(ys[k-1], y, u, true)
				pts = append(pts, [2]float32{ox + xs[k-1] + (xs[k]-xs[k-1])*u, oy + cy})
			}
		}
		pts = append(pts, [2]float32{ox + xs[k], oy + y})
	}
	return pts
}

// frame is how the lines are drawn: the size of the chart in pixels, the
// pixels a DIP and where the chart's 0, 0 falls in its first pixel, the
// lines' width in DIPs, and whether they curve.
type frame struct {
	w, h          int
	scale, ox, oy float32
	width         float32
	smooth        bool
}

// lines is the rows drawn in pixels on the CPU, as path masks on the GPU
// would fill the atlas with thousands of lines across a large chart, kept
// while what it shows holds. It draws in turns into two images, since a
// bitmap shows an image's pixels until the next Update.
type lines struct {
	f      frame
	xs, ys []float32
	colors []ui.Color
	order  []int
	bm     *ui.Bitmap
	pix    [2]*image.RGBA
}

// draw returns the bitmap of the rows in order, each through ys at the
// axes at xs, in DIPs, in its color.
func (l *lines) draw(f frame, xs, ys []float32, colors []ui.Color, order []int) *ui.Bitmap {
	if f.w <= 0 || f.h <= 0 {
		return nil
	}
	if l.bm != nil && f == l.f && slices.Equal(xs, l.xs) && slices.Equal(ys, l.ys) &&
		slices.Equal(colors, l.colors) && slices.Equal(order, l.order) {
		return l.bm
	}
	l.f = f
	l.xs, l.ys = append(l.xs[:0], xs...), append(l.ys[:0], ys...)
	l.colors, l.order = append(l.colors[:0], colors...), append(l.order[:0], order...)
	l.pix[0], l.pix[1] = l.pix[1], l.pix[0]
	img := l.pix[0]
	if img == nil || img.Rect.Dx() != f.w || img.Rect.Dy() != f.h {
		img = image.NewRGBA(image.Rect(0, 0, f.w, f.h))
		l.pix[0] = img
	} else {
		clear(img.Pix)
	}
	n := max(1, min(runtime.NumCPU(), f.h/64))
	var wg sync.WaitGroup
	for i := range n {
		wg.Go(func() { l.paint(img, f.h*i/n, f.h*(i+1)/n) })
	}
	wg.Wait()
	if l.bm == nil {
		l.bm = ui.NewBitmap(img)
	} else {
		l.bm.Update(img)
	}
	return l.bm
}

// paint draws the rows of img from y0 to y1.
func (l *lines) paint(img *image.RGBA, y0, y1 int) {
	f, k := l.f, len(l.xs)
	if k < 2 {
		return
	}
	hw := f.width * f.scale / 2
	px := func(x float32) float32 { return f.ox + x*f.scale }
	py := func(y float32) float32 { return f.oy + y*f.scale }
	for _, i := range l.order {
		c := l.colors[i]
		col := rgba{uint32(c.R), uint32(c.G), uint32(c.B), float32(c.A) / 255}
		row := l.ys[i*k : (i+1)*k]
		for j := 1; j < k; j++ {
			a, b := row[j-1], row[j]
			if math.IsInf(float64(a), 0) || math.IsInf(float64(b), 0) {
				continue
			}
			if !f.smooth {
				segment(img, y0, y1, px(l.xs[j-1]), py(a), px(l.xs[j]), py(b), hw, col)
				continue
			}
			lx, ly := l.xs[j-1], a
			for s := 1; s <= curveSteps; s++ {
				u := float32(s) / curveSteps
				x := l.xs[j-1] + (l.xs[j]-l.xs[j-1])*u
				y, _ := curve(a, b, u, true)
				segment(img, y0, y1, px(lx), py(ly), px(x), py(y), hw, col)
				lx, ly = x, y
			}
		}
	}
}

// rgba is a color to blend: red, green and blue from 0 to 255, and its
// opacity from 0 to 1.
type rgba struct {
	r, g, b uint32
	a       float32
}

// segment blends into the rows y0 to y1 of img the line from (ax, ay) to
// (bx, by), in pixels, hw to either side, in c. Each pixel along its
// longer axis takes the part of its cross section over it, which smooths
// its edges.
func segment(img *image.RGBA, y0, y1 int, ax, ay, bx, by, hw float32, c rgba) {
	steep := abs(by-ay) > abs(bx-ax)
	if steep {
		ax, ay, bx, by = ay, ax, by, bx
	}
	if ax > bx {
		ax, ay, bx, by = bx, by, ax, ay
	}
	if bx-ax < 1e-3 {
		return
	}
	m := (by - ay) / (bx - ax)
	ht := hw * float32(math.Sqrt(float64(1+m*m)))
	// u runs along the line and v across it: x and y, or y and x when
	// steep, each within the band and the image.
	uMin, uMax, vMin, vMax, du, dv := 0, img.Rect.Dx(), y0, y1, 4, img.Stride
	if steep {
		uMin, uMax, vMin, vMax, du, dv = y0, y1, 0, img.Rect.Dx(), img.Stride, 4
	}
	u0, u1 := max(uMin, int(math.Floor(float64(ax)))), min(uMax, int(math.Ceil(float64(bx))))
	if !steep && m != 0 {
		// The columns where the line reaches the band.
		a, b := ax+(float32(y0)-ht-1-ay)/m, ax+(float32(y1)+ht+1-ay)/m
		if m < 0 {
			a, b = b, a
		}
		u0, u1 = max(u0, int(math.Floor(float64(a)))), min(u1, int(math.Ceil(float64(b))))
	}
	for u := u0; u < u1; u++ {
		fu := float32(u)
		cov := (min(fu+1, bx) - max(fu, ax)) * c.a * 256
		if cov <= 0 {
			continue
		}
		vc := ay + (max(ax, min(bx, fu+0.5))-ax)*m
		vlo, vhi := vc-ht, vc+ht
		v0, v1 := max(vMin, int(math.Floor(float64(vlo)))), min(vMax, int(math.Ceil(float64(vhi))))
		for v := v0; v < v1; v++ {
			fv := float32(v)
			k := uint32(cov*(min(fv+1, vhi)-max(fv, vlo)) + 0.5)
			o := u*du + v*dv
			q := img.Pix[o : o+4 : o+4]
			n := 256 - k
			q[0] = uint8((c.r*k + uint32(q[0])*n) >> 8)
			q[1] = uint8((c.g*k + uint32(q[1])*n) >> 8)
			q[2] = uint8((c.b*k + uint32(q[2])*n) >> 8)
			q[3] = uint8((255*k + uint32(q[3])*n) >> 8)
		}
	}
}

// number writes n with commas: 1,000.
func number(n int) string {
	s := strconv.Itoa(n)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}

func abs(v float32) float32 { return max(v, -v) }

func round(v float32) float32 { return float32(math.Round(float64(v))) }
