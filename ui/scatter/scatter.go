// Package scatter shows many points on two axes, as tools for exploring
// data and embeddings do: each point a small translucent dot at its x and
// y, so that where they crowd the plot reads denser, colored by its
// category, with a legend above whose entries show and hide one, or all
// in the accent color. The axes keep round ticks as the view moves, with
// grid lines across the plot and their titles below and beside it. Dots
// grow and strengthen as fewer show.
//
// Scrolling, with the wheel or two fingers on a touchpad, zooms around the
// pointer; dragging with the middle button, or with Space held while the
// plot has the focus, pans. A double click, or Reset, fits the points
// again. Dragging draws a box, and with Option (Alt) held a lasso, that
// selects the points inside it once let go; Shift adds them to the
// selection, and Esc clears it. Selected points keep their color, the
// others fade, and a count below reads "1,284 of 100,000 selected". The
// point nearest the pointer lights, with a card of its label, its
// category, and its x and y. The app reads which points are selected in
// the frame they change:
//
//	ev := scatter.View(c, &app.plot, "UMAP 1", "UMAP 2", topics, points)
//	if ev.Changed() {
//		app.picked = ev.Selected()
//	}
//
// The dots are drawn on the CPU, on as many cores as there are, and kept
// until the view, the size, the legend or the selection changes, and a
// grid of the points finds the one under the pointer and those a box or a
// lasso holds, so that a hundred thousand points pan and zoom at
// interactive rates. Both are made once for each slice of points: give a
// new slice when they change, rather than changing it in place.
package scatter

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

	"github.com/xingkaixin/mygo-native-ui/ui/chartx"
)

// Point is a point of a plot: where it is, its category, an index of the
// plot's categories, and its label, which its card shows as the pointer
// lights it. A point whose X or Y is NaN does not show.
type Point struct {
	X, Y     float64
	Category int
	Label    string
}

// State is what a plot keeps: the stretch of the points it shows, the
// categories shown, the points selected, and the press dragging across
// it. Keep it in the app's state, one for each plot; the zero State fits
// the points, shows every category and selects none, and starts over when
// given another slice of points.
type State struct {
	view, fit view
	zoomed    bool
	shown     []bool
	selected  []int
	// changed is whether the selection changed since the last frame, and
	// version counts the changes of the selection and the legend, for the
	// dots' cache.
	changed bool
	version int
	drag    drag
	space   bool
	// plot is where the plot was laid out in the view, for the input that
	// comes between frames.
	plot ui.Rect
	grid grid
}

// Events is what a plot's selection did in a frame.
type Events struct {
	selected []int
	changed  bool
}

// Selected returns the indexes of the points selected, in order: none
// until a box or a lasso selects some. It is a new slice each time they
// change, so one the app keeps stays as it was.
func (e Events) Selected() []int { return e.selected }

// Changed reports whether the points selected changed in this frame, as
// a box or a lasso was let go, Esc cleared them, the legend hid some, or
// the points changed.
func (e Events) Changed() bool { return e.changed }

// view is a stretch of the points' plane: x from x0 to x1 across, and y
// from y0 to y1 up.
type view struct{ x0, x1, y0, y1 float64 }

// at returns where x, y is in r, in DIPs.
func (v view) at(r ui.Rect, x, y float64) (float32, float32) {
	return r.X + float32((x-v.x0)/(v.x1-v.x0))*r.W, r.Y + float32((v.y1-y)/(v.y1-v.y0))*r.H
}

// point returns the point at px, py of r.
func (v view) point(r ui.Rect, px, py float32) (float64, float64) {
	return v.x0 + float64((px-r.X)/r.W)*(v.x1-v.x0), v.y1 - float64((py-r.Y)/r.H)*(v.y1-v.y0)
}

// fitted returns the stretch that fits the points of extent e, out to the
// round values of about nx ticks across and ny up.
func fitted(e view, nx, ny int) view {
	px, py := (e.x1-e.x0)*0.02, (e.y1-e.y0)*0.02
	x := chart.NewLinear(e.x0-px, e.x1+px).Nice(nx)
	y := chart.NewLinear(e.y0-py, e.y1+py).Nice(ny)
	return view{x.Lo, x.Hi, y.Lo, y.Hi}
}

// valid reports whether p has a place on the plot.
func valid(p Point) bool {
	return !math.IsNaN(p.X) && !math.IsNaN(p.Y) && !math.IsInf(p.X, 0) && !math.IsInf(p.Y, 0)
}

// shows reports whether p's category shows; a point of none shows always.
func (s *State) shows(p Point) bool {
	return p.Category < 0 || p.Category >= len(s.shown) || s.shown[p.Category]
}

// choose makes picked, in order, the points selected.
func (s *State) choose(picked []int) {
	if slices.Equal(picked, s.selected) {
		return
	}
	s.selected, s.changed = picked, true
	s.version++
}

// grid sorts the points into cells across their extent, cols across and
// rows up, to find those about a place without going through them all:
// the points of cell k are order[start[k]:start[k+1]], and the cells of a
// row follow one another. first and n tell the slice of points it sorts
// from another.
type grid struct {
	first      *Point
	n          int
	extent     view
	cols, rows int
	start      []int32
	order      []int32
}

// newGrid sorts points into a grid of about four points a cell.
func newGrid(points []Point) grid {
	inf := math.Inf(1)
	g := grid{n: len(points), extent: view{inf, -inf, inf, -inf}}
	if len(points) > 0 {
		g.first = &points[0]
	}
	for _, p := range points {
		if valid(p) {
			e := &g.extent
			e.x0, e.x1, e.y0, e.y1 = min(e.x0, p.X), max(e.x1, p.X), min(e.y0, p.Y), max(e.y1, p.Y)
		}
	}
	e := &g.extent
	if e.x0 > e.x1 {
		*e = view{0, 1, 0, 1}
	}
	if e.x0 == e.x1 {
		e.x0, e.x1 = e.x0-1, e.x1+1
	}
	if e.y0 == e.y1 {
		e.y0, e.y1 = e.y0-1, e.y1+1
	}
	side := max(1, min(512, int(math.Sqrt(float64(len(points))/4))))
	g.cols, g.rows = side, side
	g.start = make([]int32, side*side+1)
	cells := make([]int32, len(points))
	for i, p := range points {
		cells[i] = -1
		if valid(p) {
			cx, cy := g.cell(p.X, p.Y)
			cells[i] = int32(cy*g.cols + cx)
			g.start[cells[i]+1]++
		}
	}
	for k := 1; k < len(g.start); k++ {
		g.start[k] += g.start[k-1]
	}
	g.order = make([]int32, g.start[len(g.start)-1])
	next := slices.Clone(g.start)
	for i, k := range cells {
		if k >= 0 {
			g.order[next[k]] = int32(i)
			next[k]++
		}
	}
	return g
}

// cell returns the cell of x, y, the nearest for a place outside the
// extent.
func (g *grid) cell(x, y float64) (int, int) {
	e := g.extent
	cx := max(0, min(float64(g.cols-1), math.Floor((x-e.x0)/(e.x1-e.x0)*float64(g.cols))))
	cy := max(0, min(float64(g.rows-1), math.Floor((y-e.y0)/(e.y1-e.y0)*float64(g.rows))))
	return int(cx), int(cy)
}

// each calls fn with each point in the cells over the stretch v, among
// others near it.
func (g *grid) each(v view, fn func(i int)) {
	cx0, cy0 := g.cell(v.x0, v.y0)
	cx1, cy1 := g.cell(v.x1, v.y1)
	for cy := cy0; cy <= cy1; cy++ {
		row := cy * g.cols
		for _, i := range g.order[g.start[row+cx0]:g.start[row+cx1+1]] {
			fn(int(i))
		}
	}
}

// nearest returns the shown point nearest px, py of the plot r, within
// reach DIPs and within r, or -1.
func (s *State) nearest(points []Point, r ui.Rect, px, py, reach float32) int {
	x0, y1 := s.view.point(r, px-reach, py-reach)
	x1, y0 := s.view.point(r, px+reach, py+reach)
	best, at := reach*reach, -1
	s.grid.each(view{x0, x1, y0, y1}, func(i int) {
		p := points[i]
		if !s.shows(p) {
			return
		}
		x, y := s.view.at(r, p.X, p.Y)
		if x < r.X || x > r.X+r.W || y < r.Y || y > r.Y+r.H {
			return
		}
		if d := (x-px)*(x-px) + (y-py)*(y-py); d < best {
			best, at = d, i
		}
	})
	return at
}

// What a press on the plot does as it drags: nothing yet, while it is not
// far enough to tell from a click, draw a box or a lasso, or pan.
const (
	idle = iota
	pressing
	boxing
	lassoing
	panning
)

// drag is a press on the plot while it lasts: what it does, the modifier
// keys held since it began, where the pointer was last taken, in DIPs of
// the view, and what it drew, in the points' units: the corners of a box,
// or the turns of a lasso.
type drag struct {
	mode int
	mods ui.Modifiers
	x, y float32
	path [][2]float64
}

// moveAfter is how far, in DIPs, a press goes before it drags rather than
// clicks, and a lasso before it turns again.
const moveAfter = 3

// input takes the plot's pointer, wheel and Space as they come, so that a
// lasso gets every sample of the pointer and not one a frame.
func (s *State) input(points []Point, ev ui.InputEvent) bool {
	switch ev.Kind {
	case ui.InputKeyDown, ui.InputKeyUp:
		if ev.Key != ui.KeySpace {
			return false
		}
		s.space = ev.Kind == ui.InputKeyDown
	case ui.InputScroll:
		// Scrolling 300 DIPs zooms by 2, as forcegraph does.
		s.zoom(math.Exp2(float64(ev.DY)/300), ev.X, ev.Y)
	case ui.InputPointerDown:
		switch {
		case s.drag.mode != idle:
		case ev.Button == 2 || ev.Button == 0 && s.space:
			s.drag = drag{mode: panning, x: ev.X, y: ev.Y}
		case ev.Button != 0:
			return false
		case ev.Clicks == 2:
			s.zoomed = false
		default:
			s.drag = drag{mode: pressing, mods: ev.Mods, x: ev.X, y: ev.Y}
		}
	case ui.InputPointerMove:
		if s.drag.mode == idle {
			return false
		}
		s.move(ev.X, ev.Y, ev.Mods)
	case ui.InputPointerUp:
		if s.drag.mode == idle {
			return false
		}
		if s.drag.mode == boxing || s.drag.mode == lassoing {
			s.release(points)
		}
		s.drag = drag{}
	default:
		return false
	}
	return true
}

// zoom scales the stretch the plot shows by k around x, y of the view,
// from a ten-thousandth of the stretch that fits the points to four times
// it.
func (s *State) zoom(k float64, x, y float32) {
	r, v := s.plot, s.view
	if r.W <= 0 || r.H <= 0 {
		return
	}
	z := (v.x1 - v.x0) / (s.fit.x1 - s.fit.x0)
	k = max(1e-4, min(4, z*k)) / z
	fx := float64(max(0, min(1, (x-r.X)/r.W)))
	fy := float64(max(0, min(1, (r.Y+r.H-y)/r.H)))
	cx, cy := v.x0+fx*(v.x1-v.x0), v.y0+fy*(v.y1-v.y0)
	w, h := (v.x1-v.x0)*k, (v.y1-v.y0)*k
	s.view = view{cx - fx*w, cx + (1-fx)*w, cy - fy*h, cy + (1-fy)*h}
	s.zoomed = true
}

// move follows the pointer to x, y of the view as the press drags: it
// pans, or draws on the plot, within it, once the press goes far enough,
// a lasso if Alt was held on the way.
func (s *State) move(x, y float32, mods ui.Modifiers) {
	d, r := &s.drag, s.plot
	d.mods |= mods
	if d.mode == panning {
		v := s.view
		dx := float64((x-d.x)/r.W) * (v.x1 - v.x0)
		dy := float64((y-d.y)/r.H) * (v.y1 - v.y0)
		s.view = view{v.x0 - dx, v.x1 - dx, v.y0 + dy, v.y1 + dy}
		s.zoomed = true
		d.x, d.y = x, y
		return
	}
	in := func(x, y float32) [2]float64 {
		px, py := s.view.point(r, max(r.X, min(r.X+r.W, x)), max(r.Y, min(r.Y+r.H, y)))
		return [2]float64{px, py}
	}
	far := abs(x-d.x)+abs(y-d.y) >= moveAfter
	switch {
	case d.mode == pressing && far:
		d.mode = boxing
		if d.mods&ui.Alt != 0 {
			d.mode = lassoing
		}
		d.path = append(d.path[:0], in(d.x, d.y), in(x, y))
		d.x, d.y = x, y
	case d.mode == boxing:
		d.path[1] = in(x, y)
	case d.mode == lassoing && far:
		d.path = append(d.path, in(x, y))
		d.x, d.y = x, y
	}
}

// release selects the shown points inside the box or the lasso drawn, or,
// with Shift held, adds them to the selection.
func (s *State) release(points []Point) {
	d := &s.drag
	var picked []int
	switch {
	case d.mode == boxing:
		a, b := d.path[0], d.path[1]
		box := view{min(a[0], b[0]), max(a[0], b[0]), min(a[1], b[1]), max(a[1], b[1])}
		s.grid.each(box, func(i int) {
			p := points[i]
			if s.shows(p) && p.X >= box.x0 && p.X <= box.x1 && p.Y >= box.y0 && p.Y <= box.y1 {
				picked = append(picked, i)
			}
		})
	case len(d.path) >= 3:
		inf := math.Inf(1)
		box := view{inf, -inf, inf, -inf}
		for _, q := range d.path {
			box = view{min(box.x0, q[0]), max(box.x1, q[0]), min(box.y0, q[1]), max(box.y1, q[1])}
		}
		s.grid.each(box, func(i int) {
			if p := points[i]; s.shows(p) && inside(d.path, p.X, p.Y) {
				picked = append(picked, i)
			}
		})
	}
	slices.Sort(picked)
	if d.mods&ui.Shift != 0 {
		picked = union(s.selected, picked)
	}
	s.choose(picked)
}

// inside reports whether x, y is inside the polygon of path, by the
// even-odd rule.
func inside(path [][2]float64, x, y float64) bool {
	in := false
	b := path[len(path)-1]
	for _, a := range path {
		if (a[1] > y) != (b[1] > y) && x < a[0]+(b[0]-a[0])*(y-a[1])/(b[1]-a[1]) {
			in = !in
		}
		b = a
	}
	return in
}

// union returns the indexes in a or b, both in order, in order.
func union(a, b []int) []int {
	out := make([]int, 0, len(a)+len(b))
	i, j := 0, 0
	for i < len(a) || j < len(b) {
		switch {
		case j == len(b) || i < len(a) && a[i] < b[j]:
			out = append(out, a[i])
			i++
		case i == len(a) || b[j] < a[i]:
			out = append(out, b[j])
			j++
		default:
			out = append(out, a[i])
			i, j = i+1, j+1
		}
	}
	return out
}

// View creates the plot of points, colored by their categories, with a
// legend above that shows and hides them, or all in the accent color
// when categories is nil; x and y are the titles of its axes. It fills
// its parent. The Events returned tell which points are selected.
func View(c *ui.Context, s *State, x, y string, categories []string, points []Point) Events {
	t := c.Theme()
	if s.grid.start == nil || len(points) != s.grid.n || len(points) > 0 && &points[0] != s.grid.first {
		s.grid = newGrid(points)
		s.zoomed, s.drag = false, drag{}
		s.choose(nil)
	}
	if len(s.shown) != len(categories) {
		s.shown = make([]bool, len(categories))
		for i := range s.shown {
			s.shown[i] = true
		}
		s.version++
	}
	colors := palette(t, len(categories))
	ui.Column(c).Fill().Gap(t.Space(3)).Children(func() {
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			if len(categories) == 0 {
				ui.Box(c).Grow(1)
			} else if chartx.Legend(c, categories, colors, s.shown).Label("Categories").Grow(1).Shrink(1).Changed() {
				s.version++
				s.choose(slices.DeleteFunc(slices.Clone(s.selected), func(i int) bool { return !s.shows(points[i]) }))
			}
			if ui.Button(c, "Reset").Disabled(!s.zoomed).Clicked() {
				s.zoomed = false
			}
		})
		s.plotView(c, x, y, categories, points, colors)
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, count(len(s.selected), len(points))).Bold().FontFeatures("tnum")
			alt := "Alt"
			if runtime.GOOS == "darwin" {
				alt = "Option"
			}
			hint := "Drag to select, " + alt + "-drag to lasso, scroll to zoom, Space-drag to pan"
			if len(s.selected) > 0 {
				hint = "Shift-drag to add to the selection, Esc to clear it"
			}
			ui.Text(c, hint).TextColor(t.TextMuted).SingleLine().Grow(1).Shrink(1)
		})
	})
	ev := Events{selected: s.selected, changed: s.changed}
	s.changed = false
	return ev
}

// count reads out how many of n points are selected.
func count(selected, n int) string {
	if selected == 0 {
		return number(n) + " points"
	}
	return number(selected) + " of " + number(n) + " selected"
}

// plotView builds the plot itself: the axes, titled x and y, around the
// dots, which take the pointer, the wheel and Space.
func (s *State) plotView(c *ui.Context, x, y string, categories []string, points []Point, colors []ui.Color) {
	t := c.Theme()
	box := ui.Box(c).Grow(1).AlignSelf(ui.Stretch).Focusable().Label("Scatter plot").
		Value(count(len(s.selected), len(points)))
	b := box.LayoutBounds()
	small, titleSize := t.FontSize-2, t.FontSize-1
	_, th := c.MeasureText(0, ui.Span{Text: "0", Size: small})
	_, tt := c.MeasureText(0, ui.Span{Text: "M", Size: titleSize, Weight: 600})
	gap := t.Space(1.5)
	top, right := th/2+t.Space(1), t.Space(4)
	bottom := gap + th + t.Space(2) + tt
	plotH := max(1, b.H-top-bottom)
	ny := max(2, int(plotH/56))
	s.fit = fitted(s.grid.extent, max(2, int(b.W/96)), ny)
	if !s.zoomed {
		s.view = s.fit
	}
	v := s.view
	yTicks := chart.NewLinear(v.y0, v.y1).Ticks(ny)
	labelW := float32(0)
	for _, tk := range yTicks {
		w, _ := c.MeasureText(0, ui.Span{Text: tk.Label, Size: small})
		labelW = max(labelW, w)
	}
	left := tt + t.Space(2) + labelW + gap
	plot := ui.Rect{X: left, Y: top, W: max(1, b.W-left-right), H: plotH}
	xTicks := chart.NewLinear(v.x0, v.x1).Ticks(max(2, int(plot.W/96)))
	s.plot = plot

	if (len(s.selected) > 0 || s.drag.mode == boxing || s.drag.mode == lassoing) && box.Shortcut(0, ui.KeyEscape) {
		if s.drag.mode != panning {
			s.drag = drag{}
		}
		s.choose(nil)
	}
	if !box.Focused() {
		s.space = false
	}
	box.HandleInput(func(ev ui.InputEvent) bool { return s.input(points, ev) })
	switch {
	case s.drag.mode == panning:
		box.Cursor(ui.CursorGrabbing)
	case s.space:
		box.Cursor(ui.CursorGrab)
	default:
		box.Cursor(ui.CursorCrosshair)
	}

	hovered := -1
	if px, py, over := box.PointerPosition(); over && s.drag.mode <= pressing && !s.space {
		hovered = s.nearest(points, plot, px, py, t.Space(3))
	}
	mode, path := s.drag.mode, s.drag.path
	selected, shown := s.selected, s.shown
	xDec, yDec := decimals(xTicks), decimals(yTicks)
	dc := ui.Local(box, "scatter.dots", func() dots { return dots{} })
	k := key{v: v, first: s.grid.first, n: len(points), version: s.version, dark: t.Dark,
		accent: t.Accent, muted: t.TextMuted, colored: len(categories) > 0}
	box.Draw(func(p *ui.Painter, r ui.Rect) {
		if r.W != b.W || r.H != b.H {
			c.Invalidate()
		}
		pr := ui.Rect{X: r.X + plot.X, Y: r.Y + plot.Y, W: plot.W, H: plot.H}
		line := t.Border.Alpha(0.6)
		for _, tk := range xTicks {
			gx := chart.X(pr, tk.At)
			p.Line(gx, pr.Y, gx, pr.Y+pr.H, 1, line)
		}
		for _, tk := range yTicks {
			gy := chart.Y(pr, tk.At)
			p.Line(pr.X, gy, pr.X+pr.W, gy, 1, line)
		}
		// The bitmap covers pr as Painter.Image places it, on whole
		// pixels.
		sc := p.Scale()
		bx, by := round(pr.X*sc), round(pr.Y*sc)
		k.w, k.h = int(round((pr.X+pr.W)*sc)-bx), int(round((pr.Y+pr.H)*sc)-by)
		k.dw, k.dh, k.scale, k.ox, k.oy = pr.W, pr.H, sc, pr.X*sc-bx, pr.Y*sc-by
		if bm := dc.draw(k, points, shown, selected, colors); bm != nil {
			p.Image(bm, pr, ui.FillBox)
		}
		p.Stroke(pr, t.Border, 0, 1)

		for _, tk := range xTicks {
			w, _ := p.MeasureText(0, ui.Span{Text: tk.Label, Size: small})
			lx := max(r.X, min(r.X+r.W-w, chart.X(pr, tk.At)-w/2))
			p.Text(lx, pr.Y+pr.H+gap, tk.Label, small, t.TextMuted)
		}
		for _, tk := range yTicks {
			w, _ := p.MeasureText(0, ui.Span{Text: tk.Label, Size: small})
			p.Text(pr.X-gap-w, chart.Y(pr, tk.At)-th/2, tk.Label, small, t.TextMuted)
		}
		xs := ui.Span{Text: x, Size: titleSize, Weight: 600, Color: t.Text}
		w, _ := p.MeasureText(0, xs)
		p.RichText(pr.X+(pr.W-w)/2, r.Y+r.H-tt, 0, xs)
		ys := ui.Span{Text: y, Size: titleSize, Weight: 600, Color: t.Text}
		w, _ = p.MeasureText(0, ys)
		cx, cy := r.X+tt/2, pr.Y+pr.H/2
		p.Transform(ui.Rotation(-90), cx, cy, func() {
			p.RichText(cx-w/2, cy-tt/2, 0, ys)
		})

		p.Clip(pr, 0, func() {
			outline(p, t, pr, v, mode, path)
		})
		if hovered < 0 {
			return
		}
		pt := points[hovered]
		col := t.Accent
		if len(categories) > 0 && pt.Category >= 0 && pt.Category < len(colors) {
			col = colors[pt.Category]
		}
		hx, hy := v.at(pr, pt.X, pt.Y)
		dot := max(3.5, dc.radius+1.5)
		p.Fill(ui.Rect{X: hx - dot - 2, Y: hy - dot - 2, W: 2*dot + 4, H: 2*dot + 4}, t.Background, dot+2)
		p.Fill(ui.Rect{X: hx - dot, Y: hy - dot, W: 2 * dot, H: 2 * dot}, col, dot)
		p.Stroke(ui.Rect{X: hx - dot - 2, Y: hy - dot - 2, W: 2*dot + 4, H: 2*dot + 4}, t.Text, dot+2, 1.5)
		label := pt.Label
		if label == "" {
			label = "Point " + number(hovered+1)
		}
		category := ""
		if len(categories) > 0 && pt.Category >= 0 && pt.Category < len(categories) {
			category = categories[pt.Category]
		}
		card(p, t, pr, hx, hy, dot+t.Space(2), label, category, col,
			title(x, "x"), strconv.FormatFloat(pt.X, 'f', xDec, 64),
			title(y, "y"), strconv.FormatFloat(pt.Y, 'f', yDec, 64))
	})
}

// title returns the title of an axis, or name for one untitled.
func title(s, name string) string {
	if strings.TrimSpace(s) == "" {
		return name
	}
	return s
}

// decimals returns how many decimals a value reads out with: one more
// than the ticks need.
func decimals(ticks []chart.Tick) int {
	if len(ticks) < 2 {
		return 2
	}
	step := ticks[1].Value - ticks[0].Value
	return max(0, int(math.Ceil(-math.Log10(step)-1e-9))) + 1
}

// outline paints the box or the lasso of a drag of mode through path,
// in the points' units, which v shows in r.
func outline(p *ui.Painter, t *ui.Theme, r ui.Rect, v view, mode int, path [][2]float64) {
	switch {
	case mode == boxing:
		x0, y0 := v.at(r, path[0][0], path[0][1])
		x1, y1 := v.at(r, path[1][0], path[1][1])
		box := ui.Rect{X: min(x0, x1), Y: min(y0, y1), W: abs(x1 - x0), H: abs(y1 - y0)}
		p.Fill(box, t.Accent.Alpha(0.1), 0)
		p.Stroke(box, t.Accent, 0, 1.25)
	case mode == lassoing && len(path) > 1:
		var lasso ui.Path
		for i, q := range path {
			x, y := v.at(r, q[0], q[1])
			if i == 0 {
				lasso.MoveTo(x, y)
			} else {
				lasso.LineTo(x, y)
			}
		}
		lasso.Close()
		p.FillPath(&lasso, t.Accent.Alpha(0.1))
		p.StrokePath(&lasso, 1.25, t.Accent)
	}
}

// card paints the card of a point at x, y of the plot r, gap from it,
// below and to the right where there is room: its label, its category,
// unless "", by a swatch of col, and its x and y, each after its axis's
// title.
func card(p *ui.Painter, t *ui.Theme, r ui.Rect, x, y, gap float32, label, category string, col ui.Color, xt, xv, yt, yv string) {
	small := t.FontSize - 2
	lines := [][]ui.Span{{{Text: label, Size: t.FontSize - 1, Weight: 600, Color: t.Text}}}
	if category != "" {
		lines = append(lines, []ui.Span{{Text: category, Size: small, Color: t.TextMuted}})
	}
	lines = append(lines, []ui.Span{
		{Text: xt + "  ", Size: small, Color: t.TextMuted},
		{Text: xv, Size: small, Color: t.Text, Features: "tnum"},
		{Text: "    " + yt + "  ", Size: small, Color: t.TextMuted},
		{Text: yv, Size: small, Color: t.Text, Features: "tnum"},
	})
	pad, lead, swatch := t.Space(2.5), t.Space(0.5), t.Space(2)
	maxW := t.Space(80)
	w, h := float32(0), float32(0)
	sizes := make([][2]float32, len(lines))
	for i, l := range lines {
		lw, lh := p.MeasureText(maxW, l...)
		if category != "" && i == 1 {
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
	p.Shadow(box, radius, 0, 2, 10, 0, ui.RGBA(0, 0, 0, 0.18))
	p.Fill(box, t.Background, radius)
	p.Stroke(box, t.Border, radius, 1)
	ty := cy + pad
	for i, l := range lines {
		tx := cx + pad
		if category != "" && i == 1 {
			p.Fill(ui.Rect{X: tx, Y: ty + (sizes[i][1]-swatch)/2, W: swatch, H: swatch}, col, swatch/2)
			tx += swatch + t.Space(1.5)
		}
		p.RichText(tx, ty, maxW, l...)
		ty += sizes[i][1] + lead
	}
}

// hues are the categories' hues, in an order that keeps neighbors apart.
var hues = []float32{255, 45, 150, 330, 90, 200, 20, 290, 120, 170}

// palette returns the colors of n categories, which repeat after ten,
// lighter in the dark theme. The dots are drawn in sRGB: the legend shows
// the same colors.
func palette(t *ui.Theme, n int) []ui.Color {
	l := float32(0.6)
	if t.Dark {
		l = 0.74
	}
	out := make([]ui.Color, n)
	for i := range out {
		c := ui.Oklch(l, 0.16, hues[i%len(hues)])
		out[i] = ui.RGB(c.R, c.G, c.B)
	}
	return out
}

// key is what the dots show: the plot's size in pixels and in DIPs, the
// pixels a DIP and where the plot's 0, 0 falls in its first pixel, the
// stretch of the points shown, the points by their first and their
// number, the version of the legend and the selection, and the colors of
// the theme.
type key struct {
	w, h          int
	dw, dh        float32
	scale, ox, oy float32
	v             view
	first         *Point
	n             int
	version       int
	dark          bool
	accent, muted ui.Color
	colored       bool
}

// dots is the points drawn in pixels on the CPU, as a hundred thousand
// fills a frame would cost more than a frame has, kept while what they
// show holds. It draws in turns into two images, since a bitmap shows an
// image's pixels until the next Update, and keeps the radius of the dots
// last drawn, in DIPs.
type dots struct {
	key    key
	radius float32
	bm     *ui.Bitmap
	pix    [2]*image.RGBA
}

// draw returns the bitmap of points as k shows them, in the colors of
// their categories, or the accent color, with those of the categories not
// shown left out and, while some are selected, the others faded.
func (d *dots) draw(k key, points []Point, shown []bool, selected []int, colors []ui.Color) *ui.Bitmap {
	if k.w <= 0 || k.h <= 0 {
		return nil
	}
	if d.bm != nil && k == d.key {
		return d.bm
	}
	d.key = k
	j := job{points: points, shown: shown, selected: selected, v: k.v, w: k.w, h: k.h,
		sx: float64(k.dw*k.scale) / (k.v.x1 - k.v.x0), sy: float64(k.dh*k.scale) / (k.v.y1 - k.v.y0),
		ox: float64(k.ox), oy: float64(k.oy)}

	// Fewer dots show larger and stronger: their sizes and opacities
	// follow how many are in view, and how many of them are selected.
	seen, picked := 0, 0
	m := 0
	for i, p := range points {
		in := m < len(selected) && selected[m] == i
		if in {
			m++
		}
		if valid(p) && j.shows(p) && p.X >= k.v.x0 && p.X <= k.v.x1 && p.Y >= k.v.y0 && p.Y <= k.v.y1 {
			seen++
			if in {
				picked++
			}
		}
	}
	d.radius = float32(max(1.25, min(3, 90/math.Sqrt(float64(max(1, seen))))))
	strength := func(n int) uint32 {
		return uint32(256 * max(0.22, min(0.9, 45/math.Sqrt(float64(max(1, n))))))
	}
	j.colors = make([]rgba, max(1, len(colors)))
	for i := range j.colors {
		c := k.accent
		if k.colored {
			c = colors[i]
		}
		j.colors[i] = rgba{uint32(c.R), uint32(c.G), uint32(c.B), strength(seen)}
	}
	if len(selected) > 0 {
		j.fade = rgba{uint32(k.muted.R), uint32(k.muted.G), uint32(k.muted.B), strength(seen) * 2 / 5}
		for i := range j.colors {
			j.colors[i].a = strength(picked)
		}
	}
	j.stamp(d.radius * k.scale)

	d.pix[0], d.pix[1] = d.pix[1], d.pix[0]
	img := d.pix[0]
	if img == nil || img.Rect.Dx() != k.w || img.Rect.Dy() != k.h {
		img = image.NewRGBA(image.Rect(0, 0, k.w, k.h))
		d.pix[0] = img
	} else {
		clear(img.Pix)
	}
	n := max(1, min(runtime.NumCPU(), k.h/32))
	var wg sync.WaitGroup
	for i := range n {
		wg.Go(func() { j.band(img, k.h*i/n, k.h*(i+1)/n) })
	}
	wg.Wait()
	if d.bm == nil {
		d.bm = ui.NewBitmap(img)
	} else {
		d.bm.Update(img)
	}
	return d.bm
}

// job is a drawing of dots: the points, those of the categories shown and
// those selected, in order; the stretch v of them that the image w by h
// pixels shows, sx and sy pixels a unit, from ox, oy; the colors of the
// categories, or the one color of all, and the faded one; and the stamps
// of a dot, size pixels across.
type job struct {
	points   []Point
	shown    []bool
	selected []int
	v        view
	w, h     int
	sx, sy   float64
	ox, oy   float64
	colors   []rgba
	fade     rgba
	stamps   []uint32
	size     int
}

// rgba is a color to blend: red, green and blue from 0 to 255, and its
// opacity from 0 to 256.
type rgba struct{ r, g, b, a uint32 }

// shows reports whether p's category shows.
func (j *job) shows(p Point) bool {
	return p.Category < 0 || p.Category >= len(j.shown) || j.shown[p.Category]
}

// color returns the color of p.
func (j *job) color(p Point) rgba {
	if p.Category < 0 || p.Category >= len(j.colors) {
		return j.colors[0]
	}
	return j.colors[p.Category]
}

// sub is how many places across and down a dot's stamps take within a
// pixel, to draw it where it is rather than on whole pixels.
const sub = 4

// stamp makes the stamps of a dot of radius rp pixels: for each of the
// sub by sub places it may take within its pixel, how much of each pixel
// of the size by size around it the dot covers, from 0 to 256, its edge
// smoothed across a pixel.
func (j *job) stamp(rp float32) {
	half := int(math.Ceil(float64(rp) + 0.5))
	j.size = 2*half + 1
	n := j.size * j.size
	j.stamps = make([]uint32, sub*sub*n)
	for qy := range sub {
		for qx := range sub {
			cx := float64(half) + (float64(qx)+0.5)/sub
			cy := float64(half) + (float64(qy)+0.5)/sub
			st := j.stamps[(qy*sub+qx)*n:]
			for v := range j.size {
				for u := range j.size {
					dist := math.Hypot(float64(u)+0.5-cx, float64(v)+0.5-cy)
					st[v*j.size+u] = uint32(256 * max(0, min(1, float64(rp)+0.5-dist)))
				}
			}
		}
	}
}

// band paints the dots in rows y0 to y1 of img: the points not selected,
// faded while some are, then those selected, over them.
func (j *job) band(img *image.RGBA, y0, y1 int) {
	m := 0
	for i, p := range j.points {
		if m < len(j.selected) && j.selected[m] == i {
			m++
			continue
		}
		if !valid(p) || !j.shows(p) {
			continue
		}
		c := j.fade
		if len(j.selected) == 0 {
			c = j.color(p)
		}
		j.dot(img, y0, y1, p, c)
	}
	for _, i := range j.selected {
		j.dot(img, y0, y1, j.points[i], j.color(j.points[i]))
	}
}

// dot blends the stamp of p, in c, into rows y0 to y1 of img.
func (j *job) dot(img *image.RGBA, y0, y1 int, p Point, c rgba) {
	fx := j.ox + (p.X-j.v.x0)*j.sx
	fy := j.oy + (j.v.y1-p.Y)*j.sy
	size := j.size
	half := float64(size / 2)
	if fx < -half-1 || fx > float64(j.w)+half || fy < float64(y0)-half-1 || fy > float64(y1)+half {
		return
	}
	ix, iy := math.Floor(fx), math.Floor(fy)
	bx, by := int(ix)-size/2, int(iy)-size/2
	q := (int((fy-iy)*sub)*sub + int((fx-ix)*sub)) * size * size
	st := j.stamps[q : q+size*size]
	u0, u1 := max(0, -bx), min(size, j.w-bx)
	v0, v1 := max(0, y0-by), min(size, y1-by)
	for v := v0; v < v1; v++ {
		o := (by+v)*img.Stride + (bx+u0)*4
		for u := u0; u < u1; u, o = u+1, o+4 {
			k := st[v*size+u] * c.a >> 8
			if k == 0 {
				continue
			}
			px := img.Pix[o : o+4 : o+4]
			n := 256 - k
			px[0] = uint8((c.r*k + uint32(px[0])*n) >> 8)
			px[1] = uint8((c.g*k + uint32(px[1])*n) >> 8)
			px[2] = uint8((c.b*k + uint32(px[2])*n) >> 8)
			px[3] = uint8((255*k + uint32(px[3])*n) >> 8)
		}
	}
}

// number writes n with commas: 100,000.
func number(n int) string {
	s := strconv.Itoa(n)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}

func abs(v float32) float32 { return max(v, -v) }

func round(v float32) float32 { return float32(math.Round(float64(v))) }
