// Package boxplot shows how the samples of groups spread, side by side,
// as statistics tools do: each group a column with its name under it,
// along an axis of values with round ticks and a title. A box spans the
// middle half of a group's samples, from its first quartile to its third,
// with a line at the median and a diamond at the mean; its whiskers reach
// the furthest samples within 1.5 times the box's length of it, as Tukey
// drew them, and the samples beyond, the outliers, are dots. A violin is
// the samples' density mirrored about the column's middle, as wide as the
// box where they are densest, with a slim box inside; points show every
// sample, scattered across the column, under the box. The columns go
// across and the values up, or, laid on its side, the columns down and the
// values across.
//
// The quartiles and the median are the samples' 25th, 50th and 75th
// percentiles, interpolated between the two samples nearest by rank, as
// NumPy's percentile and Excel's QUARTILE.INC compute them. The density
// is a kernel density estimate with a Gaussian kernel and Silverman's
// bandwidth, from the smallest sample to the largest; on a log scale, of
// the samples' logarithms.
//
// Hovering a group shows a card of it: how many samples it has, the
// smallest, the quartiles and the median, the largest, the mean and how
// many outliers. Hovering a dot shows its value, and its label if it has
// one.
//
//	app.chart.Mode = boxplot.Violin
//	boxplot.View(c, &app.chart, boxplot.Axis{Title: "Latency", Unit: "ms"}, app.groups).Grow(1)
//
// The statistics and densities are kept until the samples change.
package boxplot

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/chart"
)

// Mode is how a chart shows each group's samples.
type Mode uint8

const (
	// Box shows the box, its whiskers and the outliers.
	Box Mode = iota
	// Violin shows the samples' density, with a slim box inside.
	Violin
	// Points shows every sample, scattered across the column, under the
	// box.
	Points
)

// Group is samples shown together: its name, under its column, and its
// values, NaN and the infinities left out. Labels, when given, name the
// values by index, as a dot shows when hovered.
type Group struct {
	Name   string
	Values []float64
	Labels []string
}

// Axis is the axis of values: its title, and the unit that follows it in
// parentheses and the values the cards read out.
type Axis struct {
	Title, Unit string
}

// State is what a chart keeps: the statistics and densities of the
// groups, until their samples change. Mode, which the app sets, is how the
// groups show; Horizontal lays them down the left side with the values
// along the bottom; and Log spaces the values by their logarithms, those
// at or below 0 at the axis's start. Keep it in the app's state, one for
// each chart; the zero State shows box plots, upright, on a linear scale.
type State struct {
	Mode       Mode
	Horizontal bool
	Log        bool

	sums []summary
}

// View creates the chart of groups along axis. Give it a size, or grow it
// in its parent.
func View(c *ui.Context, s *State, axis Axis, groups []Group) ui.Element {
	t := c.Theme()
	if len(s.sums) != len(groups) {
		s.sums = make([]summary, len(groups))
	}
	lo, hi, pos := math.Inf(1), math.Inf(-1), math.Inf(1)
	for i, g := range groups {
		m := &s.sums[i]
		if !m.holds(g.Values, s.Log) {
			*m = summarize(g.Values, s.Log)
		}
		if m.n > 0 {
			lo, hi, pos = min(lo, m.min), max(hi, m.max), min(pos, m.pos)
		}
	}
	title := axis.Title
	if title != "" && axis.Unit != "" {
		title += " (" + axis.Unit + ")"
	}
	name := axis.Title
	if name == "" {
		name = "Box plot"
	}

	box := ui.Box(c).Role(ui.RoleImage).Label(name)
	b := box.LayoutBounds()
	f := s.layout(c, b.W, b.H, title, groups, lo, hi, pos)

	// The pointer over a group's column chooses it, and over one of its
	// dots that dot.
	hot, dot := -1, -1
	reach := t.Space(1.5)
	if px, py, over := box.PointerPosition(); over && len(groups) > 0 {
		pl := f.plot
		if px >= pl.X-reach && px <= pl.X+pl.W+reach && py >= pl.Y-reach && py <= pl.Y+pl.H+reach {
			across := (px - pl.X) / pl.W
			if s.Horizontal {
				across = (py - pl.Y) / pl.H
			}
			if i, ok := f.bands.Index(across); ok {
				hot, dot = i, f.near(&s.sums[i], s.Mode, i, px, py, reach)
			}
		}
	}
	if hot >= 0 {
		box.Value(describe(groups[hot].Name, &s.sums[hot], axis.Unit))
	}

	mode, sums := s.Mode, s.sums
	box.Draw(func(p *ui.Painter, r ui.Rect) {
		if r.W != b.W || r.H != b.H {
			c.Invalidate()
		}
		g := f
		g.plot.X += r.X
		g.plot.Y += r.Y
		g.paintAxes(p, t, r, title, groups)
		if hot >= 0 {
			mid, step := g.mid(hot), g.step()
			p.Fill(g.rect(g.pos(0), g.pos(1), mid-step/2, mid+step/2), t.Text.Alpha(0.04), t.Space(1.5))
		}
		for i := range groups {
			g.paintGroup(p, t, &sums[i], mode, i)
		}
		if dot >= 0 {
			x, y := g.dot(&sums[hot], mode, hot, dot)
			ring := float32(5.5)
			p.Stroke(ui.Rect{X: x - ring, Y: y - ring, W: 2 * ring, H: 2 * ring}, t.Text, ring, 1.5)
		}
	})

	if hot < 0 {
		return box
	}
	m, g := &s.sums[hot], groups[hot]
	box.Children(func() {
		if dot >= 0 {
			x, y := f.dot(m, s.Mode, hot, dot)
			anchor := ui.Box(c).Absolute().PassThrough().Left(x-reach).Top(y-reach).Size(2*reach, 2*reach)
			label := ""
			if dot < len(g.Labels) {
				label = g.Labels[dot]
			}
			dotCard(c, anchor, value(m.values[dot], axis.Unit), label)
			return
		}
		mid, half := f.mid(hot), f.width/2
		a0, a1 := f.pos(0.25), f.pos(0.75)
		if m.n > 0 {
			a0, a1 = f.at(m.q1), f.at(m.q3)
		}
		r := f.rect(a0, a1, mid-half, mid+half)
		anchor := ui.Box(c).Absolute().PassThrough().Left(r.X).Top(r.Y).Size(r.W, r.H)
		groupCard(c, anchor, s.Horizontal, g.Name, m, axis.Unit, hue(hot, t.Dark))
	})
	return box
}

// frame is a chart laid out, in DIPs from its top left corner: the plot,
// the scale of the axis along it, over the values' logarithms when log,
// its ticks, the groups' bands across it, and how wide a group's box is.
type frame struct {
	plot       ui.Rect
	horizontal bool
	log        bool
	scale      chart.Linear
	ticks      []chart.Tick
	bands      chart.Band
	width      float32
}

// layout lays out a chart w by h of groups, its axis titled title, its
// samples from lo to hi, the smallest above 0 pos.
func (s *State) layout(c *ui.Context, w, h float32, title string, groups []Group, lo, hi, pos float64) frame {
	t := c.Theme()
	small, nameSize := t.FontSize-2, t.FontSize-1
	_, th := c.MeasureText(0, ui.Span{Text: "0", Size: small})
	_, nh := c.MeasureText(0, ui.Span{Text: "M", Size: nameSize, Weight: 500})
	titleH := float32(0)
	if title != "" {
		_, tth := c.MeasureText(0, ui.Span{Text: title, Size: nameSize, Weight: 600})
		titleH = tth + t.Space(2)
	}
	names := make([]string, len(groups))
	for i, g := range groups {
		names[i] = g.Name
	}
	f := frame{horizontal: s.Horizontal, log: s.Log, bands: chart.NewBand(names...).Padding(0.3, 0.15)}
	if s.Horizontal {
		nameW := float32(0)
		for _, n := range names {
			nw, _ := c.MeasureText(0, ui.Span{Text: n, Size: nameSize, Weight: 500})
			nameW = max(nameW, nw)
		}
		f.plot.X, f.plot.Y = min(nameW, w*0.3)+t.Space(3), t.Space(1)
		f.plot.W = max(1, w-f.plot.X-t.Space(4))
		f.plot.H = max(1, h-f.plot.Y-th-t.Space(1.5)-titleH)
		f.scale, f.ticks = newScale(lo, hi, pos, s.Log, max(2, int(f.plot.W/90)))
	} else {
		f.plot.Y = th/2 + t.Space(1)
		f.plot.H = max(1, h-f.plot.Y-nh-t.Space(3))
		f.scale, f.ticks = newScale(lo, hi, pos, s.Log, max(2, int(f.plot.H/50)))
		labelW := float32(0)
		for _, tk := range f.ticks {
			lw, _ := c.MeasureText(0, ui.Span{Text: tk.Label, Size: small})
			labelW = max(labelW, lw)
		}
		f.plot.X = titleH + labelW + t.Space(2)
		f.plot.W = max(1, w-f.plot.X-t.Space(2))
	}
	f.width = min(f.bands.Width()*f.across(), t.Space(28))
	return f
}

// newScale returns the scale of samples from lo to hi rounded out for
// about count ticks, and its ticks. With log it is over their logarithms,
// from pos, the smallest sample above 0, out to 1, 2 or 5 times a power of
// ten, its ticks those, or only the powers of ten where those are too
// many.
func newScale(lo, hi, pos float64, log bool, count int) (chart.Linear, []chart.Tick) {
	if !log {
		if lo > hi {
			lo, hi = 0, 1
		}
		if lo == hi {
			lo, hi = lo-1, hi+1
		}
		sc := chart.NewLinear(lo, hi).Nice(count)
		ticks := sc.Ticks(count)
		for i := range ticks {
			ticks[i].Label = grouped(ticks[i].Label)
		}
		return sc, ticks
	}
	if !(pos <= hi) {
		pos, hi = 1, 10
	}
	a, b := roundDown(pos), roundUp(hi)
	if b <= a {
		a, b = a/2, b*2
	}
	sc := chart.NewLinear(math.Log10(a), math.Log10(b))
	sc.Clamp = true
	var ticks []chart.Tick
	for k := math.Floor(sc.Lo); k <= math.Ceil(sc.Hi); k++ {
		for _, m := range []float64{1, 2, 5} {
			if v := m * math.Pow(10, k); v >= a*(1-1e-9) && v <= b*(1+1e-9) {
				ticks = append(ticks, chart.Tick{At: sc.At(math.Log10(v)), Value: v, Label: grouped(strconv.FormatFloat(v, 'f', -1, 64))})
			}
		}
	}
	if len(ticks) > count+count/2 {
		decades := ticks[:0]
		for _, tk := range ticks {
			if d := math.Log10(tk.Value); math.Abs(d-math.Round(d)) < 1e-9 {
				decades = append(decades, tk)
			}
		}
		ticks = decades
	}
	return sc, ticks
}

// roundDown returns the largest of 1, 2 and 5 times a power of ten up to v,
// above 0; roundUp the smallest from v.
func roundDown(v float64) float64 {
	d := math.Pow(10, math.Floor(math.Log10(v)))
	switch m := v / d; {
	case m >= 5:
		return 5 * d
	case m >= 2:
		return 2 * d
	}
	return d
}

func roundUp(v float64) float64 {
	d := math.Pow(10, math.Floor(math.Log10(v)))
	switch m := v / d; {
	case m <= 1:
		return d
	case m <= 2:
		return 2 * d
	case m <= 5:
		return 5 * d
	}
	return 10 * d
}

// across returns how far the plot goes across the groups, in DIPs.
func (f *frame) across() float32 {
	if f.horizontal {
		return f.plot.H
	}
	return f.plot.W
}

// step returns how far apart the groups' middles are.
func (f *frame) step() float32 { return f.across() * f.bands.Width() / (1 - f.bands.Inner) }

// pos returns where the fraction fr of the scale is along the plot: up it,
// or across it from the left when horizontal.
func (f *frame) pos(fr float32) float32 {
	if f.horizontal {
		return f.plot.X + fr*f.plot.W
	}
	return f.plot.Y + (1-fr)*f.plot.H
}

// at returns where value v is along the plot.
func (f *frame) at(v float64) float32 {
	if f.log {
		if v <= 0 {
			return f.pos(0)
		}
		v = math.Log10(v)
	}
	return f.pos(f.scale.At(v))
}

// mid returns where the middle of group i's column is across the plot.
func (f *frame) mid(i int) float32 {
	if f.horizontal {
		return f.plot.Y + f.bands.At(i)*f.plot.H
	}
	return f.plot.X + f.bands.At(i)*f.plot.W
}

// point returns the point a along the plot and b across it.
func (f *frame) point(a, b float32) (x, y float32) {
	if f.horizontal {
		return a, b
	}
	return b, a
}

// rect returns the rectangle from a0 to a1 along the plot and from b0 to
// b1 across it.
func (f *frame) rect(a0, a1, b0, b1 float32) ui.Rect {
	x0, y0 := f.point(min(a0, a1), min(b0, b1))
	x1, y1 := f.point(max(a0, a1), max(b0, b1))
	return ui.Rect{X: x0, Y: y0, W: x1 - x0, H: y1 - y0}
}

// dot returns where sample j of group i, summarized by m, shows as a dot:
// scattered across its column in Points mode, in its middle otherwise.
func (f *frame) dot(m *summary, mode Mode, i, j int) (x, y float32) {
	across := f.mid(i)
	if mode == Points {
		across += jitter(j) * f.width * 0.4
	}
	return f.point(f.at(m.values[j]), across)
}

// near returns the sample of group i, summarized by m, whose dot is
// nearest x, y within reach, or -1: of all of them in Points mode, else of
// the outliers.
func (f *frame) near(m *summary, mode Mode, i int, x, y, reach float32) int {
	best, bestD := -1, reach*reach
	try := func(j int) {
		dx, dy := f.dot(m, mode, i, j)
		if d := (dx-x)*(dx-x) + (dy-y)*(dy-y); d <= bestD {
			best, bestD = j, d
		}
	}
	if mode != Points {
		for _, j := range m.outliers {
			try(j)
		}
		return best
	}
	for j, v := range m.values {
		if finite(v) {
			try(j)
		}
	}
	return best
}

// paintAxes paints the grid and its ticks' labels, the groups' names and
// the axis's title, around the plot of the chart in r.
func (f *frame) paintAxes(p *ui.Painter, t *ui.Theme, r ui.Rect, title string, groups []Group) {
	small := t.FontSize - 2
	pl := f.plot
	grid := t.Border.Alpha(0.6)
	_, th := p.MeasureText(0, ui.Span{Text: "0", Size: small})
	for _, tk := range f.ticks {
		at := f.pos(tk.At)
		w, _ := p.MeasureText(0, ui.Span{Text: tk.Label, Size: small})
		if f.horizontal {
			p.Line(at, pl.Y, at, pl.Y+pl.H, 1, grid)
			p.Text(at-w/2, pl.Y+pl.H+t.Space(1.5), tk.Label, small, t.TextMuted)
		} else {
			p.Line(pl.X, at, pl.X+pl.W, at, 1, grid)
			p.Text(pl.X-t.Space(2)-w, at-th/2, tk.Label, small, t.TextMuted)
		}
	}
	room := f.step() - t.Space(2)
	for i, g := range groups {
		name := ui.Span{Text: g.Name, Size: t.FontSize - 1, Weight: 500, Color: t.Text}
		w, h := p.MeasureText(0, name)
		mid := f.mid(i)
		// A name too long for its room shows its start.
		if f.horizontal {
			end := pl.X - t.Space(3)
			p.Clip(ui.Rect{X: r.X, Y: mid - h/2, W: end - r.X, H: h}, 0, func() {
				p.RichText(max(r.X, end-w), mid-h/2, 0, name)
			})
		} else {
			y := pl.Y + pl.H + t.Space(2)
			p.Clip(ui.Rect{X: mid - room/2, Y: y, W: room, H: h}, 0, func() {
				p.RichText(max(mid-room/2, mid-w/2), y, 0, name)
			})
		}
	}
	if title == "" {
		return
	}
	span := ui.Span{Text: title, Size: t.FontSize - 1, Weight: 600, Color: t.TextMuted}
	w, h := p.MeasureText(0, span)
	if f.horizontal {
		p.RichText(pl.X+(pl.W-w)/2, pl.Y+pl.H+t.Space(1.5)+th+t.Space(2), 0, span)
		return
	}
	cx, cy := r.X+h/2, pl.Y+pl.H/2
	p.Transform(ui.Rotation(-90), cx, cy, func() {
		p.RichText(cx-w/2, cy-h/2, 0, span)
	})
}

// paintGroup paints group i, summarized by m, as mode shows it.
func (f *frame) paintGroup(p *ui.Painter, t *ui.Theme, m *summary, mode Mode, i int) {
	if m.n == 0 {
		return
	}
	col := hue(i, t.Dark)
	mid, half := f.mid(i), f.width/2
	switch mode {
	case Points:
		alpha := float32(max(0.25, min(0.75, 10/math.Sqrt(float64(m.n)))))
		const r = 2.25
		for j, v := range m.values {
			if finite(v) {
				x, y := f.dot(m, mode, i, j)
				p.Fill(ui.Rect{X: x - r, Y: y - r, W: 2 * r, H: 2 * r}, col.Alpha(alpha), r)
			}
		}
		f.paintBox(p, m, mid, half, ui.Color{}, t.Text.Alpha(0.85), t.Background)
		return
	case Violin:
		f.paintViolin(p, m, mid, half, col)
		ink := t.Text.Alpha(0.75)
		slim := max(2, min(half*0.12, t.Space(1.25)))
		x0, y0 := f.point(f.at(m.low), mid)
		x1, y1 := f.point(f.at(m.high), mid)
		p.Line(x0, y0, x1, y1, 1.25, ink)
		p.Fill(f.rect(f.at(m.q1), f.at(m.q3), mid-slim, mid+slim), ink, 1)
		x, y := f.point(f.at(m.median), mid)
		d := slim * 0.8
		p.Fill(ui.Rect{X: x - d, Y: y - d, W: 2 * d, H: 2 * d}, t.Background, d)
	default:
		f.paintBox(p, m, mid, half, col.Alpha(0.2), col, t.Background)
	}
	const r = 3
	for _, j := range m.outliers {
		x, y := f.dot(m, mode, i, j)
		rect := ui.Rect{X: x - r, Y: y - r, W: 2 * r, H: 2 * r}
		p.Fill(rect, t.Background, r)
		p.Stroke(rect, col, r, 1.25)
	}
}

// paintBox paints m's box across mid, half wide to either side: filled
// with fill and outlined in ink, its whiskers and their caps, its median,
// and the diamond at its mean, filled with bg.
func (f *frame) paintBox(p *ui.Painter, m *summary, mid, half float32, fill, ink, bg ui.Color) {
	const lw = 1.25
	line := func(a0, a1, b0, b1 float32, w float32) {
		x0, y0 := f.point(a0, b0)
		x1, y1 := f.point(a1, b1)
		p.Line(x0, y0, x1, y1, w, ink)
	}
	for _, end := range [][2]float64{{m.q1, m.low}, {m.q3, m.high}} {
		from, to := f.at(end[0]), f.at(end[1])
		line(from, to, mid, mid, lw)
		line(to, to, mid-half/2, mid+half/2, lw)
	}
	box := f.rect(f.at(m.q1), f.at(m.q3), mid-half, mid+half)
	if fill.A > 0 {
		p.Fill(box, fill, 2)
	}
	p.Stroke(box, ink, 2, lw)
	md := f.at(m.median)
	line(md, md, mid-half, mid+half, 2.5)

	x, y := f.point(f.at(m.mean), mid)
	const d = 4.5
	var diamond ui.Path
	diamond.MoveTo(x, y-d).LineTo(x+d, y).LineTo(x, y+d).LineTo(x-d, y).Close()
	p.FillPath(&diamond, bg)
	p.StrokePath(&diamond, lw, ink)
}

// paintViolin paints m's density mirrored about mid, as wide to either
// side as half where it is densest, in col.
func (f *frame) paintViolin(p *ui.Painter, m *summary, mid, half float32, col ui.Color) {
	n := len(m.density)
	if n < 2 {
		return
	}
	along := func(k int) float32 {
		return f.pos(f.scale.At(m.dlo + (m.dhi-m.dlo)*float64(k)/float64(n-1)))
	}
	var path ui.Path
	for k, d := range m.density {
		x, y := f.point(along(k), mid+half*d)
		if k == 0 {
			path.MoveTo(x, y)
		} else {
			path.LineTo(x, y)
		}
	}
	for k := n - 1; k >= 0; k-- {
		path.LineTo(f.point(along(k), mid-half*m.density[k]))
	}
	path.Close()
	p.FillPath(&path, col.Alpha(0.3))
	p.StrokePath(&path, 1.25, col)
}

// groupCard shows by anchor, the box of the group named name, the card of
// its summary m, its values in unit and its color col: beside the box, or
// below it when horizontal.
func groupCard(c *ui.Context, anchor ui.Element, horizontal bool, name string, m *summary, unit string, col ui.Color) {
	t := c.Theme()
	ui.Overlay(c, func() {
		// The card and all in it let the pointer through to the chart under
		// it, which would otherwise lose the hover and the card.
		tip := ui.Column(c).Absolute().PassThrough().Role(ui.RoleTooltip).Width(t.Space(50)).
			Padding(t.Space(3)).Gap(t.Space(1)).Radius(t.Space(2)).
			Background(t.Background).Border(1, t.Border).Shadow(0, 4, 16, 0, ui.RGBA(0, 0, 0, 0.18))
		if horizontal {
			tip.AttachTo(anchor, ui.AnchorBottom, ui.AnchorTop).Margin(t.Space(2), 0, 0, 0)
		} else {
			tip.AttachTo(anchor, ui.AnchorRight, ui.AnchorLeft).Margin(0, 0, 0, t.Space(3))
		}
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
			row("n", grouped(strconv.Itoa(m.n)))
			if m.n == 0 {
				return
			}
			row("Min", value(m.min, unit))
			row("Q1", value(m.q1, unit))
			row("Median", value(m.median, unit))
			row("Q3", value(m.q3, unit))
			row("Max", value(m.max, unit))
			row("Mean", value(m.mean, unit))
			row("Outliers", grouped(strconv.Itoa(len(m.outliers))))
		})
	})
}

// dotCard shows above anchor, a dot, the card of its value and its label,
// if it has one.
func dotCard(c *ui.Context, anchor ui.Element, v, label string) {
	t := c.Theme()
	ui.Overlay(c, func() {
		tip := ui.Column(c).Absolute().PassThrough().Role(ui.RoleTooltip).
			AttachTo(anchor, ui.AnchorTop, ui.AnchorBottom).Margin(0, 0, t.Space(1.5), 0).
			Padding(t.Space(1.5), t.Space(2.5)).Gap(t.Space(0.5)).Radius(t.Space(1.5)).
			Background(t.Background).Border(1, t.Border).Shadow(0, 4, 16, 0, ui.RGBA(0, 0, 0, 0.18))
		tip.Children(func() {
			ui.Text(c, v).PassThrough().Bold().FontFeatures("tnum").SingleLine()
			if label != "" {
				ui.Text(c, label).PassThrough().FontSize(t.FontSize - 1).TextColor(t.TextMuted).SingleLine()
			}
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

// describe tells assistive technology about the group named name,
// summarized by m.
func describe(name string, m *summary, unit string) string {
	if m.n == 0 {
		return name + ": no samples"
	}
	return fmt.Sprintf("%s: %s samples, median %s, quartiles %s to %s, range %s to %s, %d outliers", name,
		grouped(strconv.Itoa(m.n)), value(m.median, unit), value(m.q1, unit), value(m.q3, unit),
		value(m.min, unit), value(m.max, unit), len(m.outliers))
}

// value writes v to three significant digits, its thousands grouped, and
// its unit: 0.0123, 4.56 ms, 78.9 ms, 1,234 ms.
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
