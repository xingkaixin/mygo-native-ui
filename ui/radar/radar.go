// Package radar compares a few items across several measures as a radar
// chart, as reviews compare laptops and leaderboards compare models: an
// axis for each measure from the middle out, its name beyond its end, and
// each item a closed shape through its value on every axis, filled
// faintly, with a dot at each corner, in a color of its own. Polygons at
// round values ring the middle, their values read along the top axis, or
// along the one the pointer is near. Each axis has a range of its own, or
// the app gives several the same one; an axis where lower is better runs
// the other way, its best values outermost, so that further out is better
// on every axis.
//
// A legend above names the items: a click shows or hides one, and the
// pointer on an entry, or on an item's shape, lights the item and dims the
// others. Near an axis, a card lists every item's value on it, and the
// corner nearest the pointer grows. Items grow out of the middle as they
// show and shrink back as they hide, and move to new values as the data
// changes; while the desktop asks for less motion, they are at once where
// they go.
//
//	radar.View(c, &app.radar, axes, laptops, app.shown).Grow(1)
package radar

import (
	"math"
	"slices"
	"strconv"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/chart"
)

// Axis is a measure, an axis from the middle out: its name, beyond its
// end, and the unit its values read out in. Min and Max are the values at
// the middle and at the end; with neither set, they fit the items' values,
// out to round values, from 0 unless a value is below it. Give axes the
// same Min and Max to compare them on one scale. With LowerIsBetter, the
// axis runs the other way, Max at the middle. Format writes its values,
// or with as many decimals as they need when it is nil.
type Axis struct {
	Name          string
	Unit          string
	Min, Max      float64
	LowerIsBetter bool
	Format        func(v float64) string
}

// Series is an item compared: its name, which tells it from the others,
// its value on each axis, in their order, NaN where it has none, and its
// color, or the zero Color for the chart's own.
type Series struct {
	Name   string
	Values []float64
	Color  ui.Color
}

// State is what a chart keeps: the items it showed, so that one new to it
// grows out of the middle. Keep it in the app's state, one for each chart.
type State struct {
	names []string
}

// The keys of the values a chart animates for each item, by its name: how
// far it has grown out of the middle, where it is along each axis, and
// how it stands out, from 0, dimmed, through 1 to 2, lit.
type (
	growKey  string
	valueKey struct {
		name string
		axis int
	}
	lookKey string
)

// fade is how long an item takes to light or dim.
const fade = 150 * time.Millisecond

// hues are those of the chart's own colors, in their order.
var hues = []float32{255, 45, 150, 330, 290, 195, 95, 20}

// color returns the i-th of the chart's own colors, lighter in a dark
// theme.
func color(i int, dark bool) ui.Color {
	h := hues[i%len(hues)]
	if dark {
		return ui.Oklch(0.72, 0.14, h)
	}
	return ui.Oklch(0.6, 0.16, h)
}

// View creates the chart of series across axes, with its legend above.
// shown says which series show, one for each, and the legend's entries
// turn them on and off; those past its end show. Give it a size, or grow
// it in its parent.
func View(c *ui.Context, s *State, axes []Axis, series []Series, shown []bool) ui.Element {
	t := c.Theme()
	colors := make([]ui.Color, len(series))
	for i, sr := range series {
		colors[i] = sr.Color
		if sr.Color == ui.Transparent {
			colors[i] = color(i, t.Dark)
		}
	}
	view := ui.Column(c).Gap(t.Space(2))
	view.Children(func() {
		lit := legend(c, series, colors, shown)
		s.plot(c, axes, series, colors, shown, lit)
	})
	return view
}

// legend builds an entry for each of series, in colors, which a click
// shows or hides, its swatch hollow while hidden. It returns the series
// whose entry the pointer is on, or -1.
func legend(c *ui.Context, series []Series, colors []ui.Color, shown []bool) int {
	t := c.Theme()
	hovered := -1
	ui.Row(c).Wrap().Justify(ui.Center).Gap(t.Space(1)).FocusGroup(ui.Horizontal).Children(func() {
		for i, sr := range series {
			on := true
			p := &on
			if i < len(shown) {
				p = &shown[i]
			}
			item := ui.ToggleBase(c.Key(sr.Name), p).Gap(t.Space(1.5)).Padding(t.Space(1), t.Space(2)).
				Radius(t.Radius).AlignItems(ui.Center).Label(sr.Name)
			if item.Hovered() {
				item.Background(t.SurfaceHover)
				hovered = i
			}
			item.Children(func() {
				swatch := ui.Box(c).Size(10, 10).Radius(5)
				label := ui.Text(c, sr.Name).NoWrap()
				if *p {
					swatch.Background(colors[i])
				} else {
					swatch.Border(1.5, colors[i])
					label.TextColor(t.TextMuted)
				}
			})
		}
	})
	return hovered
}

// scale is an axis laid out: where its values are, from the middle to the
// end, the values of its rings, out from the middle, and the decimals
// they take.
type scale struct {
	axis   *Axis
	lin    chart.Linear
	ticks  []chart.Tick
	places int
}

// newScale lays out axis a, the d-th, with n rings.
func newScale(a *Axis, series []Series, d, n int) scale {
	lo, hi, _ := fit(a, series, d, n)
	sc := scale{axis: a, lin: chart.Linear{Lo: lo, Hi: hi, Clamp: true}}
	if a.LowerIsBetter {
		sc.lin.Lo, sc.lin.Hi = hi, lo
	}
	for k := 1; k <= n; k++ {
		f := float32(k) / float32(n)
		v := sc.lin.Invert(f)
		sc.places = max(sc.places, places(v))
		sc.ticks = append(sc.ticks, chart.Tick{At: f, Value: v})
	}
	for k := range sc.ticks {
		sc.ticks[k].Label = sc.format(sc.ticks[k].Value, sc.places)
	}
	return sc
}

// format writes v with decimals, by the axis's Format if it has one.
func (sc *scale) format(v float64, decimals int) string {
	if sc.axis.Format != nil {
		return sc.axis.Format(v)
	}
	if v == 0 {
		v = 0 // no -0
	}
	return strconv.FormatFloat(v, 'f', decimals, 64)
}

// value writes v as the card reads it out, with its unit.
func (sc *scale) value(v float64) string {
	if math.IsNaN(v) {
		return "–"
	}
	s := sc.format(v, min(places(v), sc.places+2))
	if sc.axis.Unit != "" {
		s += " " + sc.axis.Unit
	}
	return s
}

// places returns how many decimals v takes, up to 4.
func places(v float64) int {
	for d := range 4 {
		x := v * math.Pow10(d)
		if math.Abs(x-math.Round(x)) < 1e-6*max(1, math.Abs(x)) {
			return d
		}
	}
	return 4
}

// step returns the least round step of at least v: 1, 2, 2.5, 3, 4, 5, 6
// or 8 times a power of ten. Every axis has as many rings, so steps finer
// than d3's 1, 2 and 5 keep each axis's range close to its values.
func step(v float64) float64 {
	p := math.Pow(10, math.Floor(math.Log10(v)))
	for _, m := range []float64{1, 2, 2.5, 3, 4, 5, 6, 8} {
		if m*p >= v*(1-1e-9) {
			return m * p
		}
	}
	return 10 * p
}

// fit returns the values of axis a, the d-th, from the lowest to the
// highest, n rings apart, and how badly they fit: for a range fitted to the
// items' values, the share of it above them, and for one the app gave, 1
// where its rings miss round values.
func fit(a *Axis, series []Series, d, n int) (lo, hi, miss float64) {
	k := float64(n)
	if a.Min != a.Max {
		lo, hi = min(a.Min, a.Max), max(a.Min, a.Max)
		s := (hi - lo) / k
		if math.Abs(step(s)-s) > 1e-9*s || places(lo/s) > 0 {
			miss = 1
		}
		return lo, hi, miss
	}
	var values []float64
	for _, sr := range series {
		if d < len(sr.Values) && !math.IsInf(sr.Values[d], 0) {
			values = append(values, sr.Values[d])
		}
	}
	vlo, vhi := chart.Extent(values...)
	lo = min(0, vlo)
	// Room above the highest value keeps it off the outer ring, and the
	// worst off the middle where lower is better.
	top := vhi + 0.05*(vhi-lo)
	if top <= lo {
		top = lo + 1
	}
	s := step((top - lo) / k)
	for {
		lo = min(0, math.Floor(vlo/s)*s)
		if lo+k*s >= top {
			break
		}
		s = step(s * (1 + 1e-6))
	}
	hi = lo + k*s
	return lo, hi, 1 - (vhi-lo)/(hi-lo)
}

// rings returns how many rings the chart draws, at most most: of 3 to 6,
// the count whose round values fit the axes best, 5 or else 4 where
// several do.
func rings(axes []Axis, series []Series, most int) int {
	best, least := 0, math.Inf(1)
	for _, n := range []int{5, 4, 6, 3} {
		if n > most {
			continue
		}
		miss := 0.0
		for d := range axes {
			_, _, m := fit(&axes[d], series, d, n)
			miss += m
		}
		if miss < least-1e-9 {
			best, least = n, miss
		}
	}
	if best == 0 {
		return max(1, most)
	}
	return best
}

// angle returns the angle of the d-th of n axes, in radians clockwise
// from the top.
func angle(d, n int) float64 { return 2 * math.Pi * float64(d) / float64(n) }

// label is an axis's name laid out beyond its end, with the line under it
// of its unit and whether lower is better: their sizes, where the box is
// from the middle, and how the lines align in it, from 0, left, to 1,
// right.
type label struct {
	name, note   string
	nameW, nameH float32
	noteW        float32
	box          ui.Rect
	align        float32
}

// note returns the line under axis a's name: its unit and whether lower
// is better.
func note(a *Axis) string {
	switch {
	case a.LowerIsBetter && a.Unit != "":
		return a.Unit + " · lower is better"
	case a.LowerIsBetter:
		return "lower is better"
	}
	return a.Unit
}

// plot builds the chart of series across axes, in colors, with lit the
// series whose legend entry the pointer is on, or -1.
func (s *State) plot(c *ui.Context, axes []Axis, series []Series, colors []ui.Color, shown []bool, lit int) {
	t := c.Theme()
	box := ui.Box(c).Grow(1).AlignSelf(ui.Stretch).MinHeight(t.Space(60)).Role(ui.RoleImage).Label("Radar chart")
	b := box.LayoutBounds()
	m := len(axes)
	on := func(i int) bool { return i >= len(shown) || shown[i] }
	small, nameSize := t.FontSize-2, t.FontSize-1
	gap, pad := t.Space(2.5), t.Space(1)

	// The radius is the largest that keeps every name inside the box, each
	// on the side of its axis's end away from the middle.
	radius := min(b.W, b.H)/2 - pad
	labels := make([]label, m)
	for d := range axes {
		l := &labels[d]
		l.name, l.note = axes[d].Name, note(&axes[d])
		l.nameW, l.nameH = c.MeasureText(0, ui.Span{Text: l.name, Size: nameSize, Weight: 600})
		w, h := l.nameW, l.nameH
		if l.note != "" {
			nw, nh := c.MeasureText(0, ui.Span{Text: l.note, Size: small})
			l.noteW, w, h = nw, max(w, nw), h+nh
		}
		a := angle(d, m)
		sx, uy := float32(math.Sin(a)), float32(-math.Cos(a))
		l.align = 0.5
		if sx > 0.2 {
			l.align = 0
		} else if sx < -0.2 {
			l.align = 1
		}
		l.box = ui.Rect{X: -w * l.align, Y: -h * (1 - uy) / 2, W: w, H: h}
		switch {
		case sx > 1e-3:
			radius = min(radius, (b.W/2-pad-l.box.X-w)/sx-gap)
		case sx < -1e-3:
			radius = min(radius, (b.W/2-pad+l.box.X)/-sx-gap)
		}
		switch {
		case uy > 1e-3:
			radius = min(radius, (b.H/2-pad-l.box.Y-h)/uy-gap)
		case uy < -1e-3:
			radius = min(radius, (b.H/2-pad+l.box.Y)/-uy-gap)
		}
	}
	radius = max(1, radius)
	for d := range labels {
		x, y := chart.Polar(0, 0, radius+gap, angle(d, m))
		labels[d].box.X += x
		labels[d].box.Y += y
	}

	n := rings(axes, series, int(radius/t.Space(8)))
	scales := make([]scale, m)
	for d := range axes {
		scales[d] = newScale(&axes[d], series, d, n)
	}

	grow := make([]float32, len(series))
	pts := make([][][2]float32, len(series))
	for i, sr := range series {
		if !slices.Contains(s.names, sr.Name) {
			box.Jump(growKey(sr.Name), 0)
		}
		target := float32(0)
		if on(i) {
			target = 1
		}
		grow[i] = box.AnimateTo(growKey(sr.Name), target, ui.Smooth)
		pts[i] = make([][2]float32, m)
		for d := range axes {
			f := float32(0)
			if d < len(sr.Values) && !math.IsNaN(sr.Values[d]) {
				f = scales[d].lin.At(sr.Values[d])
			}
			f = box.AnimateTo(valueKey{sr.Name, d}, f, ui.Smooth)
			x, y := chart.Polar(0, 0, radius*grow[i]*f, angle(d, m))
			pts[i][d] = [2]float32{x, y}
		}
	}
	s.names = s.names[:0]
	for _, sr := range series {
		s.names = append(s.names, sr.Name)
	}

	// Near an axis, away from the middle where they all meet, or on its
	// name, the pointer reads the values on it and lights the item whose
	// corner is nearest; elsewhere, it lights the smallest shape it is in,
	// as the larger ones show around it.
	px, py, over := box.PointerPosition()
	mx, my := px-b.W/2, py-b.H/2
	near, nearest := -1, -1
	if lit >= 0 && !on(lit) {
		lit = -1
	}
	if over && m > 0 {
		best := t.Space(4)
		for d := range axes {
			a := angle(d, m)
			sx, uy := float32(math.Sin(a)), float32(-math.Cos(a))
			along, across := mx*sx+my*uy, abs(mx*uy-my*sx)
			if along > t.Space(4) && along < radius+gap && across < best {
				best, near = across, d
			}
		}
		for d, l := range labels {
			if mx >= l.box.X && mx < l.box.X+l.box.W && my >= l.box.Y && my < l.box.Y+l.box.H {
				near = d
			}
		}
		if near >= 0 {
			least := float32(math.Inf(1))
			for i := range series {
				if !on(i) {
					continue
				}
				if d := float32(math.Hypot(float64(mx-pts[i][near][0]), float64(my-pts[i][near][1]))); d < least {
					least, nearest = d, i
				}
			}
			lit = nearest
		} else {
			least := float32(math.Inf(1))
			for i := range series {
				if a := area(pts[i]); on(i) && a < least && inside(pts[i], mx, my) {
					least, lit = a, i
				}
			}
		}
	}

	looks := make([]float32, len(series))
	order := make([]int, 0, len(series))
	for i, sr := range series {
		look := float32(1)
		switch {
		case lit == i:
			look = 2
		case lit >= 0:
			look = 0
		}
		looks[i] = box.AnimateTo(lookKey(sr.Name), look, ui.Animation{Duration: fade})
		if i != lit {
			order = append(order, i)
		}
	}
	if lit >= 0 {
		order = append(order, lit)
	}
	reading := max(0, near)

	box.Draw(func(p *ui.Painter, r ui.Rect) {
		// The chart was laid out at the size of the last frame.
		if r.W != b.W || r.H != b.H {
			c.Invalidate()
		}
		if m == 0 {
			return
		}
		cx, cy := r.X+r.W/2, r.Y+r.H/2
		var path ui.Path
		for k := 1; k <= n; k++ {
			ring := radius * float32(k) / float32(n)
			path.Reset()
			for d := range m {
				x, y := chart.Polar(cx, cy, ring, angle(d, m))
				if d == 0 {
					path.MoveTo(x, y)
				} else {
					path.LineTo(x, y)
				}
			}
			p.StrokePath(path.Close(), 1, t.Border)
		}
		for d := range m {
			x, y := chart.Polar(cx, cy, radius, angle(d, m))
			line := t.Border
			if d == near {
				line = t.TextMuted
			}
			p.StrokePolyline([][2]float32{{cx, cy}, {x, y}}, 1, line)
		}

		for _, i := range order {
			if grow[i] < 1e-3 {
				continue
			}
			look := looks[i]
			fill, stroke, width := between(look, 0.04, 0.14, 0.22), between(look, 0.3, 1, 1), between(look, 1.25, 1.75, 2.5)
			path.Reset()
			for d, pt := range pts[i] {
				if d == 0 {
					path.MoveTo(cx+pt[0], cy+pt[1])
				} else {
					path.LineTo(cx+pt[0], cy+pt[1])
				}
			}
			path.Close()
			p.FillPath(&path, colors[i].Alpha(fill))
			p.StrokePath(&path, width, colors[i].Alpha(stroke))
			for d, pt := range pts[i] {
				dot := between(look, 2.25, 2.75, 3.25)
				if i == nearest && d == near {
					dot = 5
				}
				// Dots shrink with their item, so that they do not crowd
				// the middle as it grows out of it.
				dot *= min(1, 2*grow[i])
				halo := dot + 1.5
				p.Fill(ui.Rect{X: cx + pt[0] - halo, Y: cy + pt[1] - halo, W: 2 * halo, H: 2 * halo}, t.Background.Alpha(stroke), halo)
				p.Fill(ui.Rect{X: cx + pt[0] - dot, Y: cy + pt[1] - dot, W: 2 * dot, H: 2 * dot}, colors[i].Alpha(stroke), dot)
			}
		}

		// The rings' values sit beside the axis that reads them, clockwise
		// of it.
		a := angle(reading, m)
		nx, ny := float32(math.Cos(a)), float32(math.Sin(a))
		for _, tk := range scales[reading].ticks {
			x, y := chart.Polar(cx, cy, radius*tk.At, a)
			w, h := p.MeasureText(0, ui.Span{Text: tk.Label, Size: small})
			off := t.Space(2.5) + abs(nx)*w/2 + abs(ny)*h/2
			lx, ly := x+nx*off-w/2, y+ny*off-h/2
			p.Fill(ui.Rect{X: lx - 3, Y: ly, W: w + 6, H: h}, t.Background.Alpha(0.85), 3)
			p.Text(lx, ly, tk.Label, small, t.TextMuted)
		}

		for d, l := range labels {
			x, y := cx+l.box.X, cy+l.box.Y
			name := t.Text
			if near >= 0 && d != near {
				name = t.TextMuted
			}
			p.RichText(x+(l.box.W-l.nameW)*l.align, y, 0, ui.Span{Text: l.name, Size: nameSize, Weight: 600, Color: name})
			if l.note != "" {
				p.Text(x+(l.box.W-l.noteW)*l.align, y+l.nameH, l.note, small, t.TextMuted)
			}
		}

		if near >= 0 {
			// The card sits on the side of the axis away from its values,
			// across from the point on it nearest the pointer, so that it
			// hides neither.
			a := angle(near, m)
			sx, uy := float32(math.Sin(a)), float32(-math.Cos(a))
			along := max(0, min(radius, mx*sx+my*uy))
			card(p, t, r, cx+along*sx, cy+along*uy, -float32(math.Cos(a)), -float32(math.Sin(a)),
				&scales[near], series, colors, on, near, nearest)
		}
	})
}

// card paints the card of every shown series' value on axis d, read by
// sc, off the point x, y in the direction nx, ny, within r; the row of
// series nearest is bold.
func card(p *ui.Painter, t *ui.Theme, r ui.Rect, x, y, nx, ny float32, sc *scale, series []Series, colors []ui.Color, on func(int) bool, d, nearest int) {
	small := t.FontSize - 1
	pad, rowGap, swatch := t.Space(3), t.Space(1.5), float32(8)
	title := ui.Span{Text: sc.axis.Name, Weight: 600, Color: t.Text}
	w, h := p.MeasureText(0, title)
	noteText := note(sc.axis)
	noteW, noteH := float32(0), float32(0)
	if noteText != "" {
		noteW, noteH = p.MeasureText(0, ui.Span{Text: noteText, Size: small})
		w = max(w, noteW)
	}
	type row struct {
		i             int
		name, value   ui.Span
		nameW, valueW float32
	}
	var rows []row
	rowH := float32(0)
	for i, sr := range series {
		if !on(i) {
			continue
		}
		v := math.NaN()
		if d < len(sr.Values) {
			v = sr.Values[d]
		}
		rw := row{i: i,
			name:  ui.Span{Text: sr.Name, Size: small, Color: t.Text},
			value: ui.Span{Text: sc.value(v), Size: small, Color: t.Text, Features: "tnum"},
		}
		if i == nearest {
			rw.name.Weight, rw.value.Weight = 600, 600
		}
		rw.nameW, rowH = p.MeasureText(0, rw.name)
		rw.valueW, _ = p.MeasureText(0, rw.value)
		w = max(w, swatch+t.Space(2)+rw.nameW+t.Space(6)+rw.valueW)
		rows = append(rows, rw)
	}
	cw := w + 2*pad
	ch := 2*pad + h + noteH + float32(len(rows))*(rowH+rowGap)
	if len(rows) > 0 {
		ch += t.Space(1.5)
	}
	off := t.Space(5) + abs(nx)*cw/2 + abs(ny)*ch/2
	box := ui.Rect{X: x + nx*off - cw/2, Y: y + ny*off - ch/2, W: cw, H: ch}
	box.X = max(r.X, min(r.X+r.W-cw, box.X))
	box.Y = max(r.Y, min(r.Y+r.H-ch, box.Y))

	radius := t.Space(2.5)
	shadow := ui.RGBA(0, 0, 0, 0.12)
	if t.Dark {
		shadow = ui.RGBA(0, 0, 0, 0.4)
	}
	p.Shadow(box, radius, 0, 4, 16, 0, shadow)
	p.Fill(box, t.Background, radius)
	p.Stroke(box, t.Border, radius, 1)
	cx, cy := box.X+pad, box.Y+pad
	p.RichText(cx, cy, 0, title)
	cy += h
	if noteText != "" {
		p.Text(cx, cy, noteText, small, t.TextMuted)
		cy += noteH
	}
	cy += t.Space(1.5)
	for _, rw := range rows {
		if rw.i == nearest {
			p.Fill(ui.Rect{X: box.X + pad/2, Y: cy - rowGap/2, W: cw - pad, H: rowH + rowGap}, t.Text.Alpha(0.06), t.Space(1))
		}
		sy := cy + (rowH-swatch)/2
		p.Fill(ui.Rect{X: cx, Y: sy, W: swatch, H: swatch}, colors[rw.i], swatch/2)
		p.RichText(cx+swatch+t.Space(2), cy, 0, rw.name)
		p.RichText(cx+w-rw.valueW, cy, 0, rw.value)
		cy += rowH + rowGap
	}
}

// between returns the value at look of one that is lo when dimmed (0),
// mid as it is (1) and hi when lit (2).
func between(look, lo, mid, hi float32) float32 {
	if look <= 1 {
		return lo + (mid-lo)*max(0, look)
	}
	return mid + (hi-mid)*min(1, look-1)
}

// area returns the area of the polygon through pts.
func area(pts [][2]float32) float32 {
	var a float32
	for i, p := range pts {
		q := pts[(i+1)%len(pts)]
		a += p[0]*q[1] - q[0]*p[1]
	}
	return abs(a) / 2
}

// inside reports whether x, y is inside the polygon through pts.
func inside(pts [][2]float32, x, y float32) bool {
	in := false
	for i, p := range pts {
		q := pts[(i+len(pts)-1)%len(pts)]
		if (p[1] > y) != (q[1] > y) && x < p[0]+(y-p[1])*(q[0]-p[0])/(q[1]-p[1]) {
			in = !in
		}
	}
	return in
}

func abs(v float32) float32 { return max(v, -v) }
