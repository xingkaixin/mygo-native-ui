// Package chord shows flows between groups as a chord diagram, as d3's
// chord layout draws them: each group an arc around a circle, as long as
// all it sends and receives, with ticks along it at round steps and its
// name outside it, and a ribbon for each flow, as wide at either end as
// the flow. A ribbon takes the color of the group that sends it and ends
// in an arrowhead at the group that receives it; the two ways between a
// pair of groups run side by side, so that their widths read against each
// other, and a flow within a group loops back to its arc. Names keep
// level, beside their arcs, and move apart along the circle where they
// would overlap.
//
// Pointing at a group's arc or name lights its ribbons, fades the others,
// and shows a card of all it sends and receives; pointing at a ribbon
// lights both ways of its pair, with a card of each, "Asia → Europe:
// 1,234" and "Europe → Asia: 567". A click on a group keeps it lit as the
// pointer leaves, and its ribbons alone answer the pointer; a click on it
// again or elsewhere, or Esc, lets go.
//
//	chord.View(c, &app.chord, regions, flows, "people").Grow(1)
package chord

import (
	"cmp"
	"image"
	"image/draw"
	"math"
	"runtime"
	"slices"
	"strconv"
	"sync"

	"github.com/egoist/mygo/ui"
	"golang.org/x/image/vector"
)

// State is what a diagram keeps: the group a click keeps lit, by its
// name, so that it holds as the app gives it the flows anew. Keep it in
// the app's state, one for each diagram.
type State struct {
	pinned string
}

// gap is the angle between groups, in radians.
const gap = 0.04

// shape is a polygon of a diagram, in DIPs from its top left, and its
// bounds.
type shape struct {
	pts            [][2]float32
	x0, y0, x1, y1 float32
}

func newShape(pts [][2]float32) shape {
	s := shape{pts: pts, x0: math.MaxFloat32, y0: math.MaxFloat32, x1: -math.MaxFloat32, y1: -math.MaxFloat32}
	for _, p := range pts {
		s.x0, s.y0 = min(s.x0, p[0]), min(s.y0, p[1])
		s.x1, s.y1 = max(s.x1, p[0]), max(s.y1, p[1])
	}
	return s
}

// contains reports whether x, y is inside s.
func (s *shape) contains(x, y float32) bool {
	if x < s.x0 || x > s.x1 || y < s.y0 || y > s.y1 {
		return false
	}
	in := false
	for i, j := 0, len(s.pts)-1; i < len(s.pts); j, i = i, i+1 {
		a, b := s.pts[i], s.pts[j]
		if (a[1] > y) != (b[1] > y) && x < a[0]+(y-a[1])*(b[0]-a[0])/(b[1]-a[1]) {
			in = !in
		}
	}
	return in
}

// group is a group laid out: all it sends and receives, its arc from a0
// to a1, in radians clockwise from the top, and the box of its name.
type group struct {
	out, in float64
	a0, a1  float64
	name    ui.Rect
}

// shown reports whether g has an arc, sending or receiving anything.
func (g *group) shown() bool { return g.a1 > g.a0 }

// ribbon is a flow laid out: the groups it goes from and to, and its
// amount. Its shape is that of the same index in layout.shapes.
type ribbon struct {
	from, to int
	value    float64
}

// pair reports whether r and o run between the same two groups.
func (r ribbon) pair(o ribbon) bool {
	return r.from == o.from && r.to == o.to || r.from == o.to && r.to == o.from
}

// tick is a mark along an arc at angle a, with the value it marks as its
// label, w wide, on those labeled.
type tick struct {
	a     float64
	label string
	w     float32
}

// key is what a layout depends on besides the groups and their flows:
// the diagram's size and the theme's sizes of text and room.
type key struct {
	w, h, font, space float32
}

// layout is a diagram laid out: its middle; the radii of its ribbons, of
// the inside and outside of its arcs, and of its names; its groups,
// ribbons and ticks; and the shapes of the ribbons, of the arcs, and of
// the rings that mark the group kept lit, in that order. It holds what it
// was laid out for, to be laid out again when that changes.
type layout struct {
	key   key
	names []string
	sizes [][2]float32
	flows [][]float64

	cx, cy         float64
	rr, r0, r1, rn float64
	groups         []group
	ribbons        []ribbon
	ticks          []tick
	shapes         []shape
}

// value returns the flow from group i to j, or 0 where flows has none or
// not a positive number.
func value(flows [][]float64, i, j int) float64 {
	if i < len(flows) && j < len(flows[i]) && flows[i][j] > 0 && !math.IsInf(flows[i][j], 1) {
		return flows[i][j]
	}
	return 0
}

// partners returns the groups whose flows with group i of n meet its arc,
// in their order along it, clockwise: the group before i first and the
// one after it last, so that the ribbons to groups nearby cross no others,
// and i itself halfway.
func partners(i, n int) []int {
	out := make([]int, 0, n)
	for d := 1; d < n; d++ {
		out = append(out, (i-d+n)%n)
	}
	return slices.Insert(out, (n-1)/2, i)
}

// lay lays out the flows between the groups of names, whose names are
// sizes large, for k. The ticks' labels are tickH high, and as wide as
// measure says.
func lay(t *ui.Theme, k key, names []string, sizes [][2]float32, flows [][]float64, tickH float32, measure func(string) float32) *layout {
	n := len(names)
	l := &layout{key: k, names: slices.Clone(names), sizes: slices.Clone(sizes), flows: make([][]float64, len(flows)), groups: make([]group, n)}
	for i, row := range flows {
		l.flows[i] = slices.Clone(row)
	}
	for i := range n {
		for j := range n {
			v := value(flows, i, j)
			l.groups[i].out += v
			l.groups[j].in += v
		}
	}
	shown, total := 0, 0.0
	for _, g := range l.groups {
		if g.out+g.in > 0 {
			shown++
			total += g.out + g.in
		}
	}
	if total == 0 {
		return l
	}
	pad := min(gap, math.Pi/float64(shown))
	per := (2*math.Pi - pad*float64(shown)) / total
	// from and to are the stretches of arc where each flow, i*n+j, leaves
	// and arrives: along each arc, the flow out to each partner, then the
	// flow in from it, so that the two ways of a pair run side by side.
	from, to := make([][2]float64, n*n), make([][2]float64, n*n)
	a := pad / 2
	for i := range l.groups {
		g := &l.groups[i]
		g.a0, g.a1 = a, a
		if g.out+g.in == 0 {
			continue
		}
		for _, j := range partners(i, n) {
			w := value(flows, i, j) * per
			from[i*n+j] = [2]float64{a, a + w}
			a += w
			w = value(flows, j, i) * per
			to[j*n+i] = [2]float64{a, a + w}
			a += w
		}
		g.a1 = a
		a += pad
	}

	// The names go outside the ticks, as far out as keeps each in the
	// diagram beside its arc.
	edge, rim := float64(t.Space(2)), float64(t.Space(4)+tickH)
	l.cx, l.cy = float64(k.w)/2, float64(k.h)/2
	rn := min(l.cx, l.cy) - edge
	for i, g := range l.groups {
		if !g.shown() {
			continue
		}
		at := (g.a0 + g.a1) / 2
		s, c := math.Abs(math.Sin(at)), math.Abs(math.Cos(at))
		w, h := float64(sizes[i][0]), float64(sizes[i][1])
		out := (w*s + h*c) / 2
		if s > 1e-6 {
			rn = min(rn, (l.cx-edge-w/2)/s-out)
		}
		if c > 1e-6 {
			rn = min(rn, (l.cy-edge-h/2)/c-out)
		}
	}
	l.r1 = max(rn-rim, float64(t.Space(10)))
	l.r0 = l.r1 - float64(t.Space(3))
	l.rr = l.r0 - float64(t.Space(0.5))
	l.rn = l.r1 + rim

	for i := range n {
		for j := range n {
			if v := value(flows, i, j); v > 0 {
				l.ribbons = append(l.ribbons, ribbon{i, j, v})
			}
		}
	}
	// The largest go first, under the others.
	slices.SortStableFunc(l.ribbons, func(a, b ribbon) int { return cmp.Compare(b.value, a.value) })
	head := min(float64(t.Space(2.5)), l.rr/8)
	for _, rb := range l.ribbons {
		l.shapes = append(l.shapes, l.ribbon(from[rb.from*n+rb.to], to[rb.from*n+rb.to], head))
	}
	for _, g := range l.groups {
		l.shapes = append(l.shapes, l.band(l.r0, l.r1, g.a0, g.a1))
	}
	ring := float64(t.Space(0.5))
	for _, g := range l.groups {
		l.shapes = append(l.shapes, l.band(l.r0-ring, l.r1+ring, g.a0-ring/l.r1, g.a1+ring/l.r1))
	}

	// Labeled ticks are at least minGap apart along the arcs' outside,
	// and none is labeled so near the end of its arc that its label would
	// meet the next arc's first.
	unit := per * l.r1
	minGap := float64(t.Space(12))
	major, minor := steps(minGap / unit)
	every := int(math.Round(major / minor))
	for _, g := range l.groups {
		if !g.shown() {
			continue
		}
		size := g.out + g.in
		for m := 0; float64(m)*minor <= size*(1+1e-9); m++ {
			v := float64(m) * minor
			tk := tick{a: g.a0 + v*per}
			if m%every == 0 && (size-v)*unit >= minGap/2 {
				tk.label = short(v)
				tk.w = measure(tk.label)
			}
			l.ticks = append(l.ticks, tk)
		}
	}
	l.spread(t.Space(1))
	return l
}

// steps returns the value between labeled ticks, 1, 2 or 5 times a power
// of ten, the least at least x, and that between ticks.
func steps(x float64) (major, minor float64) {
	e := math.Pow(10, math.Floor(math.Log10(x)))
	switch f := x / e; {
	case f <= 1:
		return e, e / 5
	case f <= 2:
		return 2 * e, e / 2
	case f <= 5:
		return 5 * e, e
	}
	return 10 * e, 2 * e
}

// point returns the point at radius r and angle a, clockwise from the
// top.
func (l *layout) point(r, a float64) [2]float32 {
	return [2]float32{float32(l.cx + r*math.Sin(a)), float32(l.cy - r*math.Cos(a))}
}

// arc adds to pts the points along the circle of radius r from angle a0
// to a1, both included.
func (l *layout) arc(pts [][2]float32, r, a0, a1 float64) [][2]float32 {
	n := max(1, int(math.Ceil(math.Abs(a1-a0)*r/4)))
	for k := range n + 1 {
		pts = append(pts, l.point(r, a0+(a1-a0)*float64(k)/float64(n)))
	}
	return pts
}

// curve adds to pts the points of the curve from the last of them to q,
// which bends toward the middle as d3's ribbons do, q included.
func (l *layout) curve(pts [][2]float32, q [2]float32) [][2]float32 {
	p := pts[len(pts)-1]
	cx, cy := float32(l.cx), float32(l.cy)
	const n = 32
	for k := 1; k <= n; k++ {
		u := float32(k) / n
		a, b, c := (1-u)*(1-u), 2*u*(1-u), u*u
		pts = append(pts, [2]float32{a*p[0] + b*cx + c*q[0], a*p[1] + b*cy + c*q[1]})
	}
	return pts
}

// ribbon returns the shape of a ribbon from the stretch of arc from to
// the stretch to, ending in an arrowhead head deep, each end a little
// narrower so that ribbons side by side keep apart.
func (l *layout) ribbon(from, to [2]float64, head float64) shape {
	from, to = narrow(from, 0.5/l.rr), narrow(to, 0.5/l.rr)
	pts := l.arc(nil, l.rr, from[0], from[1])
	pts = l.curve(pts, l.point(l.rr-head, to[0]))
	pts = append(pts, l.point(l.rr, (to[0]+to[1])/2), l.point(l.rr-head, to[1]))
	pts = l.curve(pts, l.point(l.rr, from[0]))
	return newShape(pts[:len(pts)-1])
}

// narrow returns the stretch of arc s less d at either end, or s where
// that would leave less than half of it.
func narrow(s [2]float64, d float64) [2]float64 {
	if s[1]-s[0] < 4*d {
		return s
	}
	return [2]float64{s[0] + d, s[1] - d}
}

// band returns the shape of the ring from radius r0 to r1 between angles
// a0 and a1.
func (l *layout) band(r0, r1, a0, a1 float64) shape {
	return newShape(l.arc(l.arc(nil, r1, a0, a1), r0, a1, a0))
}

// nameBox returns the box of group i's name at angle a: level, just
// outside the circle of the names, on the side of the middle a faces.
func (l *layout) nameBox(i int, a float64) ui.Rect {
	w, h := float64(l.sizes[i][0]), float64(l.sizes[i][1])
	s, c := math.Sin(a), math.Cos(a)
	d := l.rn + (w*math.Abs(s)+h*math.Abs(c))/2
	return ui.Rect{X: float32(l.cx + d*s - w/2), Y: float32(l.cy - d*c - h/2), W: float32(w), H: float32(h)}
}

// spread places the groups' names, each by the middle of its arc, moving
// those that would come within margin of the next apart along the
// circle, a little at a time, until none do.
func (l *layout) spread(margin float32) {
	var ids []int
	at := make([]float64, len(l.groups))
	for i, g := range l.groups {
		if g.shown() {
			ids = append(ids, i)
			at[i] = (g.a0 + g.a1) / 2
		}
	}
	near := func(a, b ui.Rect) bool {
		return a.X < b.X+b.W+margin && b.X < a.X+a.W+margin && a.Y < b.Y+b.H+margin && b.Y < a.Y+a.H+margin
	}
	for range 500 {
		moved := false
		for k, i := range ids {
			j := ids[(k+1)%len(ids)]
			// Two names are next to each other on one side only.
			if j == i || math.Mod(at[j]-at[i]+4*math.Pi, 2*math.Pi) > math.Pi {
				continue
			}
			if near(l.nameBox(i, at[i]), l.nameBox(j, at[j])) {
				at[i], at[j] = at[i]-0.003, at[j]+0.003
				moved = true
			}
		}
		if !moved {
			break
		}
	}
	for _, i := range ids {
		l.groups[i].name = l.nameBox(i, at[i])
	}
}

// groupAt returns the group whose arc, ticks or name are at x, y, or -1.
func (l *layout) groupAt(x, y float32) int {
	dx, dy := float64(x)-l.cx, float64(y)-l.cy
	d, a := math.Hypot(dx, dy), math.Atan2(dx, -dy)
	if a < 0 {
		a += 2 * math.Pi
	}
	for i, g := range l.groups {
		if g.shown() && (d >= l.r0 && d <= l.rn && a >= g.a0 && a <= g.a1 || g.name.Contains(x, y)) {
			return i
		}
	}
	return -1
}

// ribbonAt returns the ribbon on top at x, y, or -1; with pinned not -1,
// of those of group pinned only.
func (l *layout) ribbonAt(x, y float32, pinned int) int {
	for i := len(l.ribbons) - 1; i >= 0; i-- {
		rb := l.ribbons[i]
		if (pinned < 0 || rb.from == pinned || rb.to == pinned) && l.shapes[i].contains(x, y) {
			return i
		}
	}
	return -1
}

// How lit a ribbon is: as all are while the pointer is at none and no
// group is kept lit, faded while others are lit, half lit while the
// pointer is at another of the group kept lit, and lit.
const (
	base = iota
	faded
	half
	lit
)

// View creates the chord diagram of flows between the groups of names,
// flows[i][j] the amount from group i to group j, read out in unit. Give
// it a size, or Grow.
func View(c *ui.Context, s *State, names []string, flows [][]float64, unit string) ui.Element {
	t := c.Theme()
	small := t.FontSize - 3
	box := ui.Box(c).Focusable().Role(ui.RoleImage).Label("Chord diagram")
	b := box.LayoutBounds()
	sizes := make([][2]float32, len(names))
	for i, name := range names {
		w, h := c.MeasureText(0, ui.Span{Text: name, Size: t.FontSize, Weight: 600})
		sizes[i] = [2]float32{w, h}
	}
	_, tickH := c.MeasureText(0, ui.Span{Text: "0", Size: small})
	ch := ui.Local(box, "chord.cache", func() cache { return cache{} })
	k := key{b.W, b.H, t.FontSize, t.Space(1)}
	l := ch.l
	if l == nil || l.key != k || !slices.Equal(l.names, names) || !slices.Equal(l.sizes, sizes) ||
		!slices.EqualFunc(l.flows, flows, slices.Equal) {
		l = lay(t, k, names, sizes, flows, tickH, func(s string) float32 {
			w, _ := c.MeasureText(0, ui.Span{Text: s, Size: small})
			return w
		})
		ch.l = l
	}

	pinned := -1
	if s.pinned != "" {
		pinned = slices.Index(names, s.pinned)
	}
	hotGroup, hotRibbon := -1, -1
	px, py, over := box.PointerPosition()
	if over {
		hotGroup = l.groupAt(px, py)
		if hotGroup < 0 {
			hotRibbon = l.ribbonAt(px, py, pinned)
		}
	}
	if hotGroup >= 0 {
		box.Cursor(ui.CursorPointer)
	}
	if box.Clicked() {
		pinned = -1
		if hotGroup >= 0 && names[hotGroup] != s.pinned {
			pinned = hotGroup
		}
	}
	if pinned >= 0 && box.Shortcut(0, ui.KeyEscape) {
		pinned = -1
	}
	s.pinned = ""
	if pinned >= 0 {
		s.pinned = names[pinned]
	}

	focus := hotGroup
	if focus < 0 {
		focus = pinned
	}
	active := focus >= 0 || hotRibbon >= 0
	level := make([]int, len(l.ribbons))
	involved := make([]bool, len(names))
	for i, rb := range l.ribbons {
		touches := rb.from == focus || rb.to == focus
		switch {
		case hotRibbon >= 0 && rb.pair(l.ribbons[hotRibbon]):
			level[i] = lit
		case touches && hotRibbon >= 0:
			level[i] = half
		case touches:
			level[i] = lit
		case active:
			level[i] = faded
		}
		if level[i] == lit {
			involved[rb.from], involved[rb.to] = true, true
		}
	}
	if focus >= 0 && hotRibbon < 0 {
		involved[focus] = true
	}

	colors := hues(len(names), t.Dark)
	// The ribbons and arcs are drawn in two layers: all of them as they
	// are while the pointer is at none, kept until the layout changes,
	// which fades to show the ribbons faded; and those lit with the arcs
	// over it, which the pointer redraws, a few ribbons rather than all.
	var all, top []ink
	for i, rb := range l.ribbons {
		all = append(all, ink{i, colors[rb.from].Alpha(0.68)})
	}
	if active {
		alpha := [...]float32{half: 0.3, lit: 0.88}
		for lv := half; lv <= lit; lv++ {
			for i, rb := range l.ribbons {
				if level[i] == lv {
					top = append(top, ink{i, colors[rb.from].Alpha(alpha[lv])})
				}
			}
		}
		if pinned >= 0 && l.groups[pinned].shown() {
			top = append(top, ink{len(l.ribbons) + len(names) + pinned, t.Text})
		}
	}
	for i := range l.groups {
		if !l.groups[i].shown() {
			continue
		}
		all = append(all, ink{len(l.ribbons) + i, colors[i]})
		if active {
			col := colors[i]
			if !involved[i] {
				col = col.Mix(t.Background, 0.65)
			}
			top = append(top, ink{len(l.ribbons) + i, col})
		}
	}

	box.Children(func() {
		if hotGroup < 0 && hotRibbon < 0 {
			return
		}
		anchor := ui.Box(c).Absolute().PassThrough().Left(px).Top(py).Size(1, 1)
		if hotGroup >= 0 {
			g := l.groups[hotGroup]
			card(c, anchor, names[hotGroup], colors[hotGroup], []line{
				{label: "Out", value: amount(g.out, unit)},
				{label: "In", value: amount(g.in, unit)},
			})
			return
		}
		rb := l.ribbons[hotRibbon]
		if rb.from == rb.to {
			card(c, anchor, "", ui.Color{}, []line{{colors[rb.from], "Within " + names[rb.from], amount(rb.value, unit)}})
			return
		}
		card(c, anchor, "", ui.Color{}, []line{
			{colors[rb.from], names[rb.from] + " → " + names[rb.to], amount(rb.value, unit)},
			{colors[rb.to], names[rb.to] + " → " + names[rb.from], amount(value(l.flows, rb.to, rb.from), unit)},
		})
	})

	box.Draw(func(p *ui.Painter, r ui.Rect) {
		// The diagram was laid out at the size of the last frame.
		if r.W != l.key.w || r.H != l.key.h {
			c.Invalidate()
		}
		// The bitmap covers r as Painter.Image places it, on whole
		// pixels.
		sc := p.Scale()
		bx, by := round(r.X*sc), round(r.Y*sc)
		f := frame{
			l: l, w: int(round((r.X+r.W)*sc) - bx), h: int(round((r.Y+r.H)*sc) - by),
			scale: sc, ox: r.X*sc - bx, oy: r.Y*sc - by,
		}
		bm := ch.all.draw(f, all)
		switch {
		case bm == nil:
		case active:
			p.ImageRegion(bm, ui.Rect{}, r, ui.ImageOptions{Tint: ui.RGBA(255, 255, 255, 0.12)})
			p.Image(ch.top.draw(f, top), r, ui.FillBox)
		default:
			p.Image(bm, r, ui.FillBox)
		}
		cx, cy, r1 := r.X+float32(l.cx), r.Y+float32(l.cy), float32(l.r1)
		for _, tk := range l.ticks {
			sin, cos := float32(math.Sin(tk.a)), float32(math.Cos(tk.a))
			n := t.Space(1)
			if tk.label != "" {
				n = t.Space(1.5)
			}
			p.Line(cx+sin*r1, cy-cos*r1, cx+sin*(r1+n), cy-cos*(r1+n), 1, t.TextMuted)
			if tk.label == "" {
				continue
			}
			// The label runs along the arc, upright on either half.
			deg, y := float32(tk.a*180/math.Pi), cy-r1-t.Space(2)-tickH
			if cos < 0 {
				deg, y = deg-180, cy+r1+t.Space(2)
			}
			p.Transform(ui.Rotation(deg), cx, cy, func() {
				p.Text(cx-tk.w/2, y, tk.label, small, t.TextMuted)
			})
		}
		for i, g := range l.groups {
			if !g.shown() {
				continue
			}
			col := t.Text
			if active && !involved[i] {
				col = t.TextMuted
			}
			p.RichText(r.X+g.name.X, r.Y+g.name.Y, 0, ui.Span{Text: l.names[i], Size: t.FontSize, Weight: 600, Color: col})
		}
	})
	return box
}

// line is a line of a card: the color of its swatch, none for none, what
// it reads out and its amount.
type line struct {
	color ui.Color
	label string
	value string
}

// card shows by anchor, below and right of it, the lines of what the
// pointer is at, under title and its swatch where there is a title.
func card(c *ui.Context, anchor ui.Element, title string, color ui.Color, lines []line) {
	t := c.Theme()
	swatch := func(col ui.Color) {
		ui.Box(c).PassThrough().Size(t.Space(2.5), t.Space(2.5)).Radius(t.Space(0.5)).Background(col)
	}
	ui.Overlay(c, func() {
		// The card and all in it let the pointer through to the diagram
		// under it, which would otherwise lose the hover and the card.
		tip := ui.Column(c).Absolute().PassThrough().Role(ui.RoleTooltip).
			AttachTo(anchor, ui.AnchorBottomRight, ui.AnchorTopLeft).Margin(t.Space(4), 0, 0, t.Space(3)).
			Padding(t.Space(2.5), t.Space(3)).Gap(t.Space(1.5)).Radius(t.Space(2)).
			Background(t.Background).Border(1, t.Border).Shadow(0, 4, 16, 0, ui.RGBA(0, 0, 0, 0.18))
		tip.Children(func() {
			if title != "" {
				ui.Row(c).PassThrough().Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
					swatch(color)
					ui.Text(c, title).PassThrough().Bold().SingleLine()
				})
			}
			for i, ln := range lines {
				ui.Row(c.Key(i)).PassThrough().Gap(t.Space(5)).AlignItems(ui.Center).Children(func() {
					ui.Row(c).PassThrough().Gap(t.Space(2)).AlignItems(ui.Center).Grow(1).Children(func() {
						if ln.color.A > 0 {
							swatch(ln.color)
						}
						ui.Text(c, ln.label).PassThrough().TextColor(t.TextMuted).SingleLine()
					})
					ui.Text(c, ln.value).PassThrough().FontFeatures("tnum").SingleLine()
				})
			}
		})
	})
}

// hues returns the colors of n groups: as many hues evenly around the
// wheel, taken by strides of about three eighths of it, so that groups
// side by side differ, in sRGB, which the ribbons are drawn in.
func hues(n int, dark bool) []ui.Color {
	stride := max(1, int(math.Round(float64(n)*0.38)))
	for gcd(stride, n) > 1 {
		stride++
	}
	l := float32(0.64)
	if dark {
		l = 0.7
	}
	out := make([]ui.Color, n)
	for i := range out {
		c := ui.Oklch(l, 0.14, float32((250+360*(i*stride%n)/n)%360))
		out[i] = ui.RGB(c.R, c.G, c.B)
	}
	return out
}

func gcd(a, b int) int {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

// amount writes v rounded, with commas, and unit: "12,400 people".
func amount(v float64, unit string) string {
	s := strconv.FormatFloat(math.Round(v), 'f', 0, 64)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	if unit != "" {
		s += " " + unit
	}
	return s
}

// short writes v as a tick reads it: 0, 500, 1.5k, 20k, 3M.
func short(v float64) string {
	suffix := ""
	for _, u := range []struct {
		at float64
		s  string
	}{{1e9, "B"}, {1e6, "M"}, {1e3, "k"}} {
		if v >= u.at {
			v, suffix = v/u.at, u.s
			break
		}
	}
	return strconv.FormatFloat(math.Round(v*100)/100, 'f', -1, 64) + suffix
}

// cache is what a diagram keeps between frames: its layout, and its
// pixels in two layers, all its ribbons and arcs, and those lit over
// them.
type cache struct {
	l        *layout
	all, top raster
}

// frame is how a layout's shapes are drawn: the size of the diagram in
// pixels, the pixels of a DIP, and where its 0, 0 falls in its first
// pixel.
type frame struct {
	l             *layout
	w, h          int
	scale, ox, oy float32
}

// ink is a shape of a layout, by its index, in a color.
type ink struct {
	shape int
	color ui.Color
}

// raster is the ribbons and arcs drawn in pixels on the CPU, as path
// masks on the GPU would fill the atlas with ribbons as large as the
// circle, kept while what it shows holds. It draws in turns into two
// images, since a bitmap shows an image's pixels until the next Update.
type raster struct {
	f    frame
	inks []ink
	bm   *ui.Bitmap
	pix  [2]*image.RGBA
	mask *image.Alpha
	// z holds a rasterizer for each band of rows, which draw at once.
	z []vector.Rasterizer
}

// draw returns the bitmap of the shapes of f's layout in inks, one over
// the other.
func (ras *raster) draw(f frame, inks []ink) *ui.Bitmap {
	if f.w <= 0 || f.h <= 0 {
		return nil
	}
	if ras.bm != nil && f == ras.f && slices.Equal(inks, ras.inks) {
		return ras.bm
	}
	ras.f, ras.inks = f, inks
	ras.pix[0], ras.pix[1] = ras.pix[1], ras.pix[0]
	img := ras.pix[0]
	if img == nil || img.Rect.Dx() != f.w || img.Rect.Dy() != f.h {
		img = image.NewRGBA(image.Rect(0, 0, f.w, f.h))
		ras.pix[0] = img
	} else {
		clear(img.Pix)
	}
	if ras.mask == nil || ras.mask.Rect != img.Rect {
		ras.mask = image.NewAlpha(img.Rect)
	}
	n := max(1, min(runtime.NumCPU(), f.h/64))
	if len(ras.z) != n {
		ras.z = make([]vector.Rasterizer, n)
	}
	var wg sync.WaitGroup
	for i := range n {
		band := image.Rect(0, f.h*i/n, f.w, f.h*(i+1)/n)
		wg.Go(func() { ras.paint(&ras.z[i], img, band) })
	}
	wg.Wait()
	if ras.bm == nil {
		ras.bm = ui.NewBitmap(img)
	} else {
		ras.bm.Update(img)
	}
	return ras.bm
}

// paint draws the rows band of img with z.
func (ras *raster) paint(z *vector.Rasterizer, img *image.RGBA, band image.Rectangle) {
	f := ras.f
	floor := func(v float32) int { return int(math.Floor(float64(v))) }
	for _, k := range ras.inks {
		sh := &f.l.shapes[k.shape]
		if len(sh.pts) < 3 {
			continue
		}
		r := image.Rect(floor(f.ox+sh.x0*f.scale)-1, floor(f.oy+sh.y0*f.scale)-1,
			floor(f.ox+sh.x1*f.scale)+2, floor(f.oy+sh.y1*f.scale)+2).Intersect(band)
		if r.Empty() {
			continue
		}
		z.Reset(r.Dx(), r.Dy())
		dx, dy := f.ox-float32(r.Min.X), f.oy-float32(r.Min.Y)
		z.MoveTo(dx+sh.pts[0][0]*f.scale, dy+sh.pts[0][1]*f.scale)
		for _, q := range sh.pts[1:] {
			z.LineTo(dx+q[0]*f.scale, dy+q[1]*f.scale)
		}
		z.ClosePath()
		ras.fill(z, img, r, k.color)
	}
}

// fill paints c on the rectangle r of img where the shape z holds, as
// large as r, covers it.
func (ras *raster) fill(z *vector.Rasterizer, img *image.RGBA, r image.Rectangle, c ui.Color) {
	z.DrawOp = draw.Src
	z.Draw(ras.mask, r, image.Opaque, image.Point{})
	for y := r.Min.Y; y < r.Max.Y; y++ {
		m := ras.mask.Pix[ras.mask.PixOffset(r.Min.X, y):][:r.Dx()]
		p := img.Pix[img.PixOffset(r.Min.X, y):]
		for x, a := range m {
			if a == 0 {
				continue
			}
			a := uint32(a) * uint32(c.A) / 255
			q := p[4*x : 4*x+4 : 4*x+4]
			q[0] = uint8((uint32(c.R)*a + uint32(q[0])*(255-a)) / 255)
			q[1] = uint8((uint32(c.G)*a + uint32(q[1])*(255-a)) / 255)
			q[2] = uint8((uint32(c.B)*a + uint32(q[2])*(255-a)) / 255)
			q[3] = uint8((255*a + uint32(q[3])*(255-a)) / 255)
		}
	}
}

func round(v float32) float32 { return float32(math.Round(float64(v))) }
