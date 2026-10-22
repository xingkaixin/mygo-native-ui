// Package ridgeline shows how the samples of groups spread, one above
// another, as a ridgeline plot (a joy plot) does: each group a row with
// its name at the left, its samples' density a filled curve rising from
// the row's baseline into the rows above, along one axis of values with
// round ticks and a title. The rows at the back, at the top, are drawn
// first, so that the ridges in front cover them. Marks rise from a ridge's
// baseline at the quantiles the app asks for, as the median. A ridge takes
// its group's color, or, with a gradient, the color of each value under
// it.
//
// The density is a kernel density estimate with a Gaussian kernel and
// Silverman's bandwidth, 0.9 × min(σ, IQR / 1.34) × n^-⅕, which the app
// may scale. The ridges share one scale of density, so that a group whose
// samples spread wide stands lower. The quantiles are interpolated between
// the two samples nearest by rank, as NumPy's percentile computes them.
//
// Hovering a ridge shows a card of its group at the value under the
// pointer: the value, the density there, and the share of the group's
// samples below it.
//
//	app.chart.Overlap = 1.5
//	app.chart.Quantiles = []float64{0.5}
//	ridgeline.View(c, &app.chart, ridgeline.Axis{Title: "Daily high", Unit: "°C"}, app.months).Grow(1)
//
// The densities are kept until the samples, the bandwidth or the axis's
// stretch change.
package ridgeline

import (
	"cmp"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/chart"

	"github.com/xingkaixin/mygo-native-ui/ui/chartx"
)

// Group is samples shown together: its name, at the left of its row, and
// its values, NaN and the infinities left out.
type Group struct {
	Name   string
	Values []float64
}

// Axis is the axis of values: its title, and the unit that follows it in
// parentheses and the values the card reads out.
type Axis struct {
	Title, Unit string
}

// State is what a chart keeps: the groups' densities, until their samples
// change. Overlap, which the app sets, is how many rows above its own the
// densest ridge rises into: at 0 it reaches the baseline of the row above,
// at 1 the one above that. Bandwidth scales Silverman's bandwidth,
// smoothing the ridges above 1 and sharpening them below; 0 counts as 1.
// Quantiles are marked on each ridge, from 0 to 1: 0.5 is the median.
// Gradient, unless empty, colors the ridges by value, the axis's start its
// first color and its end its last, rather than each in its group's color.
// Sorted lists the groups by their medians, the lowest at the top. Keep it
// in the app's state, one for each chart; the zero State shows ridges in
// their groups' colors and order, unmarked, the densest reaching the row
// above.
type State struct {
	Overlap   float32
	Bandwidth float64
	Quantiles []float64
	Gradient  []ui.Color
	Sorted    bool

	ridges []ridge
}

// View creates the chart of groups along axis. Give it a size, or grow it
// in its parent.
func View(c *ui.Context, s *State, axis Axis, groups []Group) ui.Element {
	t := c.Theme()
	if len(s.ridges) != len(groups) {
		s.ridges = make([]ridge, len(groups))
	}
	bw := s.Bandwidth
	if bw <= 0 {
		bw = 1
	}
	lo, hi := math.Inf(1), math.Inf(-1)
	for i, g := range groups {
		r := &s.ridges[i]
		if !r.holds(g.Values) {
			*r = newRidge(g.Values)
		}
		if n := len(r.sorted); n > 0 {
			// The ridges taper off 3 bandwidths beyond their samples.
			lo, hi = min(lo, r.sorted[0]-3*r.h*bw), max(hi, r.sorted[n-1]+3*r.h*bw)
		}
	}
	order := make([]int, len(groups))
	for i := range order {
		order[i] = i
	}
	if s.Sorted {
		slices.SortStableFunc(order, func(a, b int) int {
			ra, rb := &s.ridges[a], &s.ridges[b]
			if len(ra.sorted) == 0 || len(rb.sorted) == 0 {
				return len(rb.sorted) - len(ra.sorted)
			}
			return cmp.Compare(ra.median, rb.median)
		})
	}
	title := axis.Title
	if title != "" && axis.Unit != "" {
		title += " (" + axis.Unit + ")"
	}
	name := axis.Title
	if name == "" {
		name = "Ridgeline plot"
	}

	box := ui.Box(c).Role(ui.RoleImage).Label(name)
	b := box.LayoutBounds()
	f := layout(c, b.W, b.H, title, groups, max(0, s.Overlap), lo, hi)
	dens := make([][]float64, len(groups))
	for i := range groups {
		dens[i] = s.ridges[i].densities(f.scale.Lo, f.scale.Hi, bw)
		for _, d := range dens[i] {
			f.peak = max(f.peak, d)
		}
	}
	if f.peak == 0 {
		f.peak = 1
	}
	gradient := len(s.Gradient) > 0
	paint := chartx.NewSequential(f.scale.Lo, f.scale.Hi, s.Gradient...)

	// The pointer over a ridge chooses it, the frontmost of those it is
	// over, and over the gap between two the row below.
	hot, at := -1, 0.0
	reach := t.Space(1.5)
	if px, py, over := box.PointerPosition(); over && len(groups) > 0 {
		pl := f.plot
		if px >= pl.X-reach && px <= pl.X+pl.W+reach && py >= pl.Y-reach && py <= pl.Y+pl.H+reach {
			at = f.scale.Invert(max(0, min(1, (px-pl.X)/pl.W)))
			k := int(math.Ceil(float64((py - pl.Y - f.rise) / f.row)))
			hot = order[max(0, min(len(order)-1, k))]
			for k := len(order) - 1; k >= 0; k-- {
				i := order[k]
				if base := f.base(k); py <= base && py >= base-f.height(s.ridges[i].at(at, bw)) {
					hot = i
					break
				}
			}
		}
	}
	if hot >= 0 {
		box.Value(describe(groups[hot].Name, &s.ridges[hot], at, bw, axis.Unit))
	}

	ridges, quantiles := s.ridges, s.Quantiles
	box.Draw(func(p *ui.Painter, r ui.Rect) {
		if r.W != b.W || r.H != b.H {
			c.Invalidate()
		}
		g := f
		g.plot.X += r.X
		g.plot.Y += r.Y
		g.paintAxes(p, t, r, title, groups, order)
		grad := paint.Gradient()
		for k, i := range order {
			ink := hue(i, t.Dark)
			fill := func(path *ui.Path) { p.FillPath(path, t.Background.Mix(ink, 0.4)) }
			if gradient {
				ink = t.Text.Alpha(0.7)
				fill = func(path *ui.Path) { p.FillPathGradient(path, grad) }
			}
			g.paintRidge(p, &ridges[i], dens[i], k, bw, quantiles, fill, ink)
		}
		if hot >= 0 {
			k := slices.Index(order, hot)
			x, base := g.x(at), g.base(k)
			y := base - g.height(ridges[hot].at(at, bw))
			p.Line(x, base, x, y, 1.25, t.Text)
			const ring = 4.5
			dot := ui.Rect{X: x - ring, Y: y - ring, W: 2 * ring, H: 2 * ring}
			p.Fill(dot, t.Background, ring)
			p.Stroke(dot, t.Text, ring, 1.5)
		}
	})

	if hot < 0 {
		return box
	}
	k := slices.Index(order, hot)
	x, base := f.x(at), f.base(k)
	y := base - f.height(s.ridges[hot].at(at, bw))
	box.Children(func() {
		anchor := ui.Box(c).Absolute().PassThrough().Left(x-reach).Top(y-reach).Size(2*reach, 2*reach)
		col := hue(hot, t.Dark)
		if gradient {
			col = paint.At(at)
		}
		card(c, anchor, groups[hot].Name, col, &s.ridges[hot], at, bw, axis)
	})
	return box
}

// frame is a chart laid out, in DIPs from its top left corner: the plot,
// from the top of the back ridge at its highest to the baseline of the
// front row, how tall each row is and how high the densest ridge rises,
// the scale of the axis across the plot and its ticks, and the highest
// density of all, which rises that high.
type frame struct {
	plot      ui.Rect
	row, rise float32
	scale     chart.Linear
	ticks     []chart.Tick
	peak      float64
}

// layout lays out a chart w by h of groups, the densest ridge rising
// overlap rows into those above its own, its axis titled title, its ridges
// from lo to hi.
func layout(c *ui.Context, w, h float32, title string, groups []Group, overlap float32, lo, hi float64) frame {
	t := c.Theme()
	small, nameSize := t.FontSize-2, t.FontSize-1
	_, th := c.MeasureText(0, ui.Span{Text: "0", Size: small})
	titleH := float32(0)
	if title != "" {
		_, tth := c.MeasureText(0, ui.Span{Text: title, Size: nameSize, Weight: 600})
		titleH = tth + t.Space(2)
	}
	nameW := float32(0)
	for _, g := range groups {
		nw, _ := c.MeasureText(0, ui.Span{Text: g.Name, Size: nameSize, Weight: 500})
		nameW = max(nameW, nw)
	}
	var f frame
	f.plot.X, f.plot.Y = min(nameW, w*0.3)+t.Space(3), t.Space(1)
	f.plot.W = max(1, w-f.plot.X-t.Space(4))
	f.plot.H = max(1, h-f.plot.Y-th-t.Space(1.5)-titleH)
	f.row = f.plot.H / (float32(max(1, len(groups))) + overlap)
	f.rise = f.row * (1 + overlap)
	if lo > hi {
		lo, hi = 0, 1
	}
	if lo == hi {
		lo, hi = lo-1, hi+1
	}
	count := max(2, int(f.plot.W/90))
	f.scale = chart.NewLinear(lo, hi).Nice(count)
	f.ticks = f.scale.Ticks(count)
	for i := range f.ticks {
		f.ticks[i].Label = grouped(f.ticks[i].Label)
	}
	return f
}

// base returns where the baseline of row k is down the plot.
func (f *frame) base(k int) float32 { return f.plot.Y + f.rise + float32(k)*f.row }

// x returns where value v is across the plot.
func (f *frame) x(v float64) float32 { return f.plot.X + f.scale.At(v)*f.plot.W }

// height returns how high density d rises from its baseline.
func (f *frame) height(d float64) float32 { return f.rise * float32(d/f.peak) }

// paintAxes paints the grid and its ticks' labels, the groups' names, in
// rows by order, and the axis's title, around the plot of the chart in r.
func (f *frame) paintAxes(p *ui.Painter, t *ui.Theme, r ui.Rect, title string, groups []Group, order []int) {
	small := t.FontSize - 2
	pl := f.plot
	grid := t.Border.Alpha(0.6)
	for _, tk := range f.ticks {
		x := pl.X + tk.At*pl.W
		w, _ := p.MeasureText(0, ui.Span{Text: tk.Label, Size: small})
		p.Line(x, pl.Y, x, pl.Y+pl.H, 1, grid)
		p.Text(x-w/2, pl.Y+pl.H+t.Space(1.5), tk.Label, small, t.TextMuted)
	}
	end := pl.X - t.Space(3)
	for k, i := range order {
		name := ui.Span{Text: groups[i].Name, Size: t.FontSize - 1, Weight: 500, Color: t.Text}
		w, h := p.MeasureText(0, name)
		y := f.base(k) - h
		// A name too long for its room shows its start.
		p.Clip(ui.Rect{X: r.X, Y: y, W: end - r.X, H: h}, 0, func() {
			p.RichText(max(r.X, end-w), y, 0, name)
		})
	}
	if title == "" {
		return
	}
	_, th := p.MeasureText(0, ui.Span{Text: "0", Size: small})
	span := ui.Span{Text: title, Size: t.FontSize - 1, Weight: 600, Color: t.TextMuted}
	w, _ := p.MeasureText(0, span)
	p.RichText(pl.X+(pl.W-w)/2, pl.Y+pl.H+t.Space(1.5)+th+t.Space(2), 0, span)
}

// paintRidge paints m's ridge in row k, of density across the plot and
// its bandwidth times bw: filled by fill, outlined in ink, with a mark at
// each of quantiles.
func (f *frame) paintRidge(p *ui.Painter, m *ridge, density []float64, k int, bw float64, quantiles []float64, fill func(*ui.Path), ink ui.Color) {
	pl, base := f.plot, f.base(k)
	if len(density) < 2 {
		p.Line(pl.X, base, pl.X+pl.W, base, 1.25, ink)
		return
	}
	pts := make([][2]float32, len(density))
	var path ui.Path
	path.MoveTo(pl.X, base)
	for j, d := range density {
		pts[j] = [2]float32{pl.X + pl.W*float32(j)/float32(len(density)-1), base - f.height(d)}
		path.LineTo(pts[j][0], pts[j][1])
	}
	path.LineTo(pl.X+pl.W, base).Close()
	fill(&path)
	p.StrokePolyline(pts, 1.25, ink)
	for _, q := range quantiles {
		v := quantile(m.sorted, q)
		x := f.x(v)
		p.Line(x, base, x, base-f.height(m.at(v, bw)), 1.25, ink)
	}
}

// card shows by anchor, the point of ridge m at value at, the card of the
// group named name there: its color col, the value, the density there,
// its bandwidth times bw, and the share of its samples below.
func card(c *ui.Context, anchor ui.Element, name string, col ui.Color, m *ridge, at, bw float64, axis Axis) {
	t := c.Theme()
	ui.Overlay(c, func() {
		// The card and all in it let the pointer through to the chart under
		// it, which would otherwise lose the hover and the card.
		tip := ui.Column(c).Absolute().PassThrough().Role(ui.RoleTooltip).Width(t.Space(50)).
			AttachTo(anchor, ui.AnchorRight, ui.AnchorLeft).Margin(0, 0, 0, t.Space(3)).
			Padding(t.Space(3)).Gap(t.Space(1)).Radius(t.Space(2)).
			Background(t.Background).Border(1, t.Border).Shadow(0, 4, 16, 0, ui.RGBA(0, 0, 0, 0.18))
		tip.Children(func() {
			ui.Row(c).PassThrough().Gap(t.Space(2)).AlignItems(ui.Center).Margin(0, 0, t.Space(1), 0).Children(func() {
				ui.Box(c).PassThrough().Size(t.Space(2.5), t.Space(2.5)).Radius(2).Background(col)
				ui.Text(c, name).PassThrough().Bold().SingleLine().Shrink(1)
			})
			row := func(label, v string) {
				ui.Row(c).PassThrough().Gap(t.Space(3)).Children(func() {
					ui.Text(c, label).PassThrough().FontSize(t.FontSize - 1).TextColor(t.TextMuted).Grow(1)
					ui.Text(c, v).PassThrough().FontSize(t.FontSize - 1).FontFeatures("tnum")
				})
			}
			label := axis.Title
			if label == "" {
				label = "Value"
			}
			row(label, value(at, axis.Unit))
			row("Density", density(m.at(at, bw), axis.Unit))
			row("Below", share(m, at))
		})
	})
}

// hue returns group i's color, the hues of the groups apart around the
// circle, lighter in the dark theme.
func hue(i int, dark bool) ui.Color {
	l := float32(0.6)
	if dark {
		l = 0.72
	}
	return ui.Oklch(l, 0.14, float32((250+i*137)%360))
}

// describe tells assistive technology about the group named name, of
// ridge m, at value at.
func describe(name string, m *ridge, at, bw float64, unit string) string {
	return fmt.Sprintf("%s at %s: density %s, %s below", name, value(at, unit), density(m.at(at, bw), unit), share(m, at))
}

// density writes density d per unit: 0.0613 per °C.
func density(d float64, unit string) string {
	s := value(d, "")
	if unit != "" {
		s += " per " + unit
	}
	return s
}

// share writes how many of m's samples are below at: 38% (118 of 310).
func share(m *ridge, at float64) string {
	n := len(m.sorted)
	if n == 0 {
		return "no samples"
	}
	k := m.below(at)
	return fmt.Sprintf("%.0f%% (%s of %s)", 100*float64(k)/float64(n), grouped(strconv.Itoa(k)), grouped(strconv.Itoa(n)))
}

// value writes v to three significant digits, its thousands grouped, and
// its unit: 0.0123, 4.56 °C, 78.9 °C, 1,234 °C.
func value(v float64, unit string) string {
	d := 0
	if a := math.Abs(v); a > 0 && a < 100 {
		d = min(6, 2-int(math.Floor(math.Log10(a))))
	}
	s := grouped(strconv.FormatFloat(v, 'f', d, 64))
	if unit != "" {
		s += " " + unit
	}
	return s
}

// grouped puts commas between the thousands of the number s: 12,345.6.
func grouped(s string) string {
	end := strings.IndexByte(s, '.')
	if end < 0 {
		end = len(s)
	}
	start := 0
	if strings.HasPrefix(s, "-") {
		start = 1
	}
	for i := end - 3; i > start; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}
