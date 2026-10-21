// Package logic shows digital signals over time, as a logic analyzer's
// capture or a WaveDrom timing diagram: a row for each signal under a time
// ruler, its name on the left. A signal of one bit draws as a trace, high
// or low with square edges, a hatched band where it is unknown (x) and a
// line midway where nothing drives it (z). A bus draws as lanes with
// slanted ends where its value changes, each with its value written in,
// in hexadecimal or as its label, when the lane is wide enough for it.
// Changes closer together than the view can show run into a solid block.
// The ruler's ticks follow the zoom, in ns, µs, ms or s as they need.
//
// The wheel zooms around the pointer, and dragging or scrolling sideways
// pans; Fit shows the whole capture again. A click places cursor A and a
// Shift+click cursor B, on the nearest edge of the signal under the
// pointer when one is near; dragging a cursor's flag on the ruler moves
// it. Above the signals, the cursors' times read out, with Δ, B − A, and
// its frequency, 1/Δ. The row under the pointer lights, its value shown
// beside the pointer.
//
//	capture, err := logic.ParseWaveJSON(src, 10e-9) // cycles of 10 ns
//	...
//	logic.View(c, &app.diagram, capture)
//
// Or the app gives what it captured, the changes of each signal:
//
//	logic.View(c, &app.spi, logic.Capture{End: 40e-6, Signals: []logic.Signal{
//		{Name: "CS#", Bits: 1, Changes: []logic.Change{{At: 0, Value: 1}, {At: 2e-6}, {At: 38e-6, Value: 1}}},
//		{Name: "Data", Bits: 8, Changes: []logic.Change{{Level: logic.Floating}, {At: 2e-6, Value: 0x9F, Label: "RDID"}}},
//	}})
package logic

import (
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
)

// minLane is how wide, in DIPs, the stretch between two changes must be
// to draw on its own; narrower ones run together into a block.
const minLane = 2

// snapDIPs is how near an edge, in DIPs, a cursor snaps to it, and
// clickSlop how far a press may move and still be a click.
const (
	snapDIPs  = 6
	clickSlop = 3
)

// Capture is signals over time, from 0 to End seconds.
type Capture struct {
	Signals []Signal
	End     float64
}

// Signal is a row of a capture: its name, how many bits it has, and what
// it changed to when, in order of time. A signal of 1 bit draws as a
// trace; of more, as a bus writing its values in as many hexadecimal
// digits as its bits need; and of 0, as a bus writing only its labels, as
// WaveJSON's do. Gaps are the times where its drawing breaks, as where a
// stretch of time was left out.
type Signal struct {
	Name    string
	Bits    int
	Changes []Change
	Gaps    []float64
}

// Level is how a signal is driven: to its value, to one nobody knows (x),
// or not at all (z).
type Level uint8

const (
	// Driven is a signal at its value.
	Driven Level = iota
	// Unknown is a signal at a value nobody knows, as before a reset.
	Unknown
	// Floating is a signal nothing drives, at high impedance.
	Floating
)

// Change is a signal taking a value At seconds into the capture, which it
// holds until its next change or the end. A bus lane writes Label, when
// there is one, instead of the value, as a decoded byte's character or a
// state's name.
type Change struct {
	At    float64
	Value uint64
	Level Level
	Label string
}

// State is what a view keeps: the stretch of time it shows, the cursors,
// and the drag going on. Keep it in the app's state, one for each view.
// The zero State shows the whole capture, with no cursors.
type State struct {
	// from is the time at the left of the signals and span the time
	// across them, in seconds; a span of 0 fits the capture, whatever its
	// length.
	from, span float64
	// at are the times of cursors A and B, and placed whether they are.
	at     [2]float64
	placed [2]bool
	// pressing is whether the pointer presses the view, and travel how
	// far it went since the press began, to tell a click from a pan.
	pressing bool
	travel   float32
	// grab is the cursor whose flag is dragged plus 1, and grabDX how far
	// right of the cursor the flag was taken.
	grab   int
	grabDX float32
}

// window returns the time at the left of the signals and the time across
// them, of a capture end seconds long.
func (s *State) window(end float64) (from, span float64) {
	if s.span == 0 {
		return 0, end
	}
	return s.from, s.span
}

// clamp keeps the stretch shown inside a capture end seconds long, fitting
// it once it shows it all.
func (s *State) clamp(end float64) {
	if s.span <= 0 || s.span >= end {
		s.from, s.span = 0, 0
		return
	}
	s.from = max(0, min(end-s.span, s.from))
}

// zoomAt shows span seconds, down to a millionth of the capture, keeping
// the time at f, a fraction of the width of the signals, where it is.
func (s *State) zoomAt(span, f, end float64) {
	from, now := s.window(end)
	at := from + f*now
	s.span = max(end*1e-6, span)
	s.from = at - f*s.span
	s.clamp(end)
}

// View creates the view of capture. It is as tall as its rows; give it a
// width, or let it stretch in its parent.
func View(c *ui.Context, s *State, capture Capture) ui.Element {
	t := c.Theme()
	if capture.End > 0 {
		s.clamp(capture.End)
	}
	z := measure(c, capture.Signals)
	pal := colors(t)
	root := ui.Column(c).Gap(t.Space(2))
	root.Children(func() {
		s.readout(c, pal)
		s.plot(c, z, pal, capture)
	})
	return root
}

// cursorNames are the names of the cursors, as their flags show them.
var cursorNames = [2]string{"A", "B"}

// readout builds the row above the signals: the cursors' times, Δ and
// 1/Δ, or how to place them, and Fit.
func (s *State) readout(c *ui.Context, pal palette) {
	t := c.Theme()
	ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
		ui.Row(c).Wrap().Gap(t.Space(4)).Grow(1).AlignItems(ui.Center).MinHeight(t.Space(7)).Children(func() {
			for k, name := range cursorNames {
				if !s.placed[k] {
					continue
				}
				ui.Row(c.Key(k)).Gap(t.Space(1.5)).AlignItems(ui.Center).Children(func() {
					ui.Text(c, name).Bold().FontSize(t.FontSize-1).TextColor(t.Background).Background(pal.cursor[k]).
						Padding(0, t.Space(1.5)).Radius(t.Space(1))
					ui.Text(c, duration(s.at[k])).FontFeatures("tnum")
				})
			}
			field := func(label, value string) {
				ui.Row(c).Gap(t.Space(1.5)).Children(func() {
					ui.Text(c, label).TextColor(t.TextMuted)
					ui.Text(c, value).FontFeatures("tnum")
				})
			}
			switch {
			case s.placed[0] && s.placed[1]:
				d := s.at[1] - s.at[0]
				field("Δ", duration(d))
				if d != 0 {
					field("1/Δ", frequency(1/math.Abs(d)))
				}
			case s.placed[0]:
				ui.Text(c, "Shift+click to place B").TextColor(t.TextMuted)
			case s.placed[1]:
				ui.Text(c, "Click to place A").TextColor(t.TextMuted)
			default:
				ui.Text(c, "Click to place cursor A, Shift+click to place B").TextColor(t.TextMuted)
			}
		})
		if ui.Button(c, "Fit").Disabled(s.span == 0).Clicked() {
			s.span = 0
		}
	})
}

// plot builds the names, the ruler and the signals, and takes the
// pointer over them.
func (s *State) plot(c *ui.Context, z sizes, pal palette, capture Capture) {
	t := c.Theme()
	sigs, end := capture.Signals, capture.End
	n := len(sigs)
	live := end > 0
	box := ui.Box(c).AlignSelf(ui.Stretch).Height(z.rulerH + float32(n)*z.rowH).Label("Timing diagram")
	b := box.LayoutBounds()
	plotW := max(1, b.W-z.nameW)
	from, span := s.window(end)
	tAt := func(x float32) float64 { return from + float64((x-z.nameW)/plotW)*span }
	xAt := func(at float64) float32 { return z.nameW + float32((at-from)/span)*plotW }
	row := func(y float32) int {
		i := int(math.Floor(float64((y - z.rulerH) / z.rowH)))
		if y < z.rulerH || i >= n {
			return -1
		}
		return i
	}
	// snap returns the time at x, or at the nearest edge of the signal at
	// y when one is near.
	snap := func(x, y float32) float64 {
		at := max(0, min(end, tAt(x)))
		if i := row(y); i >= 0 {
			if e, ok := nearest(sigs[i].Changes, at, float64(snapDIPs/plotW)*span); ok {
				return e
			}
		}
		return at
	}

	px, py, over := box.PointerPosition()
	dx, dy, pressing := box.Dragged()
	if pressing && !s.pressing {
		s.travel = 0
	}
	s.pressing = pressing
	if pressing && live {
		s.travel += float32(math.Abs(float64(dx)) + math.Abs(float64(dy)))
		if dx != 0 {
			s.from, s.span = from-float64(dx/plotW)*span, span
			s.clamp(end)
			from, span = s.window(end)
		}
	}
	panning := pressing && s.travel > clickSlop
	inPlot := live && over && px >= z.nameW
	if box.Clicked() && inPlot && s.travel <= clickSlop {
		k := 0
		if box.ClickModifiers()&ui.Shift != 0 {
			k = 1
		}
		s.at[k], s.placed[k] = snap(px, py), true
	}
	switch {
	case panning:
		box.Cursor(ui.CursorGrabbing)
	case inPlot:
		box.Cursor(ui.CursorCrosshair)
	}
	// Scrolling comes between builds, at the box laid out since, which
	// only a handle reads then.
	h := ui.Local(box, "logic.handle", func() ui.Handle { return ui.Handle{} })
	box.Bind(h)
	box.HandleInput(func(ev ui.InputEvent) bool {
		w := h.Bounds(c).W - z.nameW
		if ev.Kind != ui.InputScroll || w <= 0 || !live {
			return false
		}
		from, span := s.window(end)
		if math.Abs(float64(ev.DX)) > math.Abs(float64(ev.DY)) {
			s.from, s.span = from+float64(ev.DX/w)*span, span
			s.clamp(end)
			return true
		}
		// Scrolling 300 DIPs zooms by 2, as candles does.
		s.zoomAt(span*math.Exp2(float64(ev.DY)/300), float64(max(0, min(1, (ev.X-z.nameW)/w))), end)
		return true
	})

	grabbed := 0
	box.Children(func() {
		for k, name := range cursorNames {
			fx := xAt(s.at[k])
			if !live || !s.placed[k] || s.grab != k+1 && (fx < z.nameW || fx > z.nameW+plotW) {
				continue
			}
			flag := ui.Box(c.Key(k)).Absolute().Top(z.rulerH-z.flagH).Size(z.flagW, z.flagH).
				Cursor(ui.CursorResizeEW).Label("Cursor " + name).Value(duration(s.at[k]))
			if fdx, _, ok := flag.Dragged(); ok {
				grabbed = k + 1
				if s.grab != k+1 {
					s.grabDX = px - fx
				}
				if fdx != 0 {
					s.at[k] = snap(max(z.nameW, min(z.nameW+plotW, px-s.grabDX)), py)
					fx = xAt(s.at[k])
				}
			}
			flag.Left(fx - z.flagW/2)
			color := pal.cursor[k]
			flag.Draw(func(p *ui.Painter, r ui.Rect) {
				tip := t.Space(1)
				p.Fill(ui.Rect{X: r.X, Y: r.Y, W: r.W, H: r.H - tip}, color, t.Space(1))
				var arrow ui.Path
				arrow.MoveTo(r.X+r.W/2-tip, r.Y+r.H-tip-0.5).LineTo(r.X+r.W/2+tip, r.Y+r.H-tip-0.5).LineTo(r.X+r.W/2, r.Y+r.H).Close()
				p.FillPath(&arrow, color)
				span := ui.Span{Text: name, Size: z.small, Weight: 700, Color: t.Background}
				w, h := p.MeasureText(0, span)
				p.RichText(r.X+(r.W-w)/2, r.Y+(r.H-tip-h)/2, 0, span)
			})
		}
	})
	s.grab = grabbed

	hot := -1
	if over && !pressing && grabbed == 0 {
		hot = row(py)
	}
	hovering := inPlot && !pressing && grabbed == 0
	var hover float64
	var value string
	if hovering {
		hover = snap(px, py)
		if hot >= 0 {
			value = valueAt(sigs[hot], tAt(px))
		}
	}
	if s.placed[0] || s.placed[1] {
		var parts []string
		for k, name := range cursorNames {
			if s.placed[k] {
				parts = append(parts, name+" "+duration(s.at[k]))
			}
		}
		box.Value(strings.Join(parts, ", "))
	}

	cursors := s.at
	placed := s.placed
	box.Draw(func(p *ui.Painter, r ui.Rect) {
		radius := t.Space(1.5)
		plot := ui.Rect{X: r.X + z.nameW, Y: r.Y + z.rulerH, W: plotW, H: float32(n) * z.rowH}
		column := ui.Rect{X: plot.X, Y: r.Y, W: plotW, H: r.H}
		d := &drawer{p: p, t: t, z: z, pal: pal, plot: plot, from: from, span: span, end: end}
		p.Clip(r, radius, func() {
			p.Fill(ui.Rect{X: r.X, Y: r.Y, W: z.nameW, H: r.H}, t.Text.Alpha(0.03), 0)
			p.Fill(ui.Rect{X: plot.X, Y: r.Y, W: plotW, H: z.rulerH}, t.Text.Alpha(0.03), 0)
			if hot >= 0 {
				p.Fill(ui.Rect{X: r.X, Y: plot.Y + float32(hot)*z.rowH, W: r.W, H: z.rowH}, t.Accent.Alpha(0.08), 0)
			}
			for i := 1; i < n; i++ {
				y := plot.Y + float32(i)*z.rowH
				p.Line(r.X, y, r.X+r.W, y, 1, t.Border.Alpha(0.5))
			}
			p.Line(r.X, plot.Y, r.X+r.W, plot.Y, 1, t.Border)
			p.Line(plot.X, r.Y, plot.X, r.Y+r.H, 1, t.Border)
			for i, sig := range sigs {
				span := ui.Span{Text: sig.Name, Size: t.FontSize, Color: t.Text}
				_, h := p.MeasureText(0, span)
				p.Clip(ui.Rect{X: r.X, Y: r.Y, W: z.nameW - t.Space(2), H: r.H}, 0, func() {
					p.RichText(r.X+t.Space(3), plot.Y+float32(i)*z.rowH+(z.rowH-h)/2, 0, span)
				})
			}
			if !live {
				return
			}
			p.Clip(column, 0, func() {
				d.ticks(ui.Rect{X: plot.X, Y: r.Y, W: plotW, H: z.rulerH})
				if placed[0] && placed[1] {
					x0, x1 := d.x(min(cursors[0], cursors[1])), d.x(max(cursors[0], cursors[1]))
					p.Fill(ui.Rect{X: x0, Y: plot.Y, W: x1 - x0, H: plot.H}, t.Text.Alpha(0.05), 0)
				}
			})
			p.Clip(plot, 0, func() {
				for i, sig := range sigs {
					top := plot.Y + float32(i)*z.rowH
					bg := t.Background
					if i == hot {
						bg = t.Accent.Alpha(0.08).Over(bg)
					}
					d.signal(sig, top+z.inset, top+z.rowH-z.inset, bg)
				}
			})
			p.Clip(column, 0, func() {
				if hovering {
					hx := d.x(hover)
					dashed(p, hx, plot.Y, plot.Y+plot.H, t.TextMuted)
					tag(p, t, z, ui.Rect{X: plot.X, Y: r.Y, W: plotW, H: z.rulerH - z.flagH}, hx, duration(hover), false)
				}
				for k := range placed {
					if placed[k] {
						cx := d.x(cursors[k])
						p.Line(cx, r.Y+z.rulerH-1, cx, r.Y+r.H, 1, pal.cursor[k])
					}
				}
				if hovering && value != "" {
					y := plot.Y + float32(hot)*z.rowH
					tag(p, t, z, ui.Rect{X: plot.X, Y: y, W: plotW, H: z.rowH}, r.X+px, value, true)
				}
			})
		})
		p.Stroke(r, t.Border, radius, 1)
	})
}

// tag draws text on a pill in the inverse of the theme, in the middle of
// box's height: centered on x, or beside it when aside, keeping inside
// box.
func tag(p *ui.Painter, t *ui.Theme, z sizes, box ui.Rect, x float32, text string, aside bool) {
	span := ui.Span{Text: text, Size: z.small, Font: "monospace", Color: t.Background}
	w, h := p.MeasureText(0, span)
	pad := t.Space(1.5)
	w += 2 * pad
	left := x - w/2
	if aside {
		left = x + t.Space(3)
		if left+w > box.X+box.W {
			left = x - t.Space(3) - w
		}
	}
	left = max(box.X, min(box.X+box.W-w, left))
	pill := ui.Rect{X: left, Y: box.Y + (box.H-h-t.Space(1))/2, W: w, H: h + t.Space(1)}
	p.Fill(pill, t.Text, t.Space(1))
	p.RichText(pill.X+pad, pill.Y+t.Space(0.5), 0, span)
}

// dashed paints a dashed line down from y0 to y1 at x.
func dashed(p *ui.Painter, x, y0, y1 float32, c ui.Color) {
	const dash, gap = 4, 3
	for y := y0; y < y1; y += dash + gap {
		p.Fill(ui.Rect{X: x - 0.5, Y: y, W: 1, H: min(dash, y1-y)}, c, 0)
	}
}

// sizes are the sizes of a view's parts, from the theme and the names of
// its signals.
type sizes struct {
	nameW, rowH, rulerH, inset, slant, flagW, flagH, small float32
}

// measure returns the sizes of a view of sigs.
func measure(c *ui.Context, sigs []Signal) sizes {
	t := c.Theme()
	small := t.FontSize - 1
	_, th := c.MeasureText(0, ui.Span{Text: "0", Size: small})
	z := sizes{rowH: t.Space(8), rulerH: th + t.Space(6), inset: t.Space(2), slant: t.Space(1.5),
		flagW: th + t.Space(1), flagH: th + t.Space(2), small: small}
	w := t.Space(10)
	for _, s := range sigs {
		nw, _ := c.MeasureText(0, ui.Span{Text: s.Name, Size: t.FontSize})
		w = max(w, nw)
	}
	z.nameW = w + t.Space(6)
	return z
}

// palette is the colors of a view: of traces, of bus lanes, and of
// cursors A and B.
type palette struct {
	trace, bus ui.Color
	cursor     [2]ui.Color
}

// colors returns the palette for the theme, lighter in a dark one.
func colors(t *ui.Theme) palette {
	l := float32(0.55)
	if t.Dark {
		l = 0.78
	}
	return palette{
		trace:  ui.Oklch(l, 0.15, 150),
		bus:    ui.Oklch(l, 0.12, 245),
		cursor: [2]ui.Color{ui.Oklch(l+0.03, 0.19, 35), ui.Oklch(l, 0.17, 305)},
	}
}

// drawer draws signals in plot, from seconds from to from+span of a
// capture end seconds long.
type drawer struct {
	p               *ui.Painter
	t               *ui.Theme
	z               sizes
	pal             palette
	plot            ui.Rect
	from, span, end float64
}

// x returns where time at is. Far outside the plot, it stays a little
// outside it, so that what is drawn there stays small.
func (d *drawer) x(at float64) float32 {
	v := float64(d.plot.X) + (at-d.from)/d.span*float64(d.plot.W)
	return float32(max(float64(d.plot.X)-50, min(float64(d.plot.X+d.plot.W)+50, v)))
}

// at returns the time at x.
func (d *drawer) at(x float32) float64 {
	return d.from + float64((x-d.plot.X)/d.plot.W)*d.span
}

// ticks draws the ticks of ruler and their times, and faint lines down the
// plot at those with times.
func (d *drawer) ticks(ruler ui.Rect) {
	p, t := d.p, d.t
	step, minor := niceStep(d.span * 100 / float64(d.plot.W))
	unit, name := unitOf(d.from + d.span)
	decimals := max(0, int(math.Ceil(-math.Log10(step/unit)-1e-9)))
	fine := step / float64(minor)
	bottom := ruler.Y + ruler.H
	right := float32(math.Inf(-1))
	for i := math.Ceil(d.from / fine); i*fine <= d.from+d.span; i++ {
		x := d.x(i * fine)
		if int(i)%minor != 0 {
			p.Line(x, bottom-t.Space(1), x, bottom, 1, t.TextMuted.Alpha(0.5))
			continue
		}
		p.Line(x, bottom-t.Space(2), x, bottom, 1, t.TextMuted)
		p.Line(x, bottom, x, d.plot.Y+d.plot.H, 1, t.Text.Alpha(0.06))
		label := ui.Span{Text: strconv.FormatFloat(i*fine/unit, 'f', decimals, 64) + " " + name, Size: d.z.small, Color: t.TextMuted}
		w, _ := p.MeasureText(0, label)
		edge := t.Space(1.5)
		if lx := max(ruler.X+edge, min(ruler.X+ruler.W-edge-w, x-w/2)); lx > right+t.Space(2) {
			p.RichText(lx, ruler.Y+t.Space(1.5), 0, label)
			right = lx + w
		}
	}
}

// signal draws sig between top and bottom, over a row of color bg, from
// the change before the plot's left to the first past its right. Changes
// closer than minLane draw as a block from the first to where they space
// out again.
func (d *drawer) signal(sig Signal, top, bottom float32, bg ui.Color) {
	ch := sig.Changes
	n := len(ch)
	if n == 0 {
		return
	}
	bus := sig.Bits != 1
	right := d.plot.X + d.plot.W
	k := max(0, sort.Search(n, func(i int) bool { return ch[i].At > d.from })-1)
	x := d.x(ch[k].At)
	block := float32(-1)
	edge := false
	for k < n && x < right {
		end := d.end
		if k+1 < n {
			end = ch[k+1].At
		}
		x1 := d.x(end)
		if x1-x >= minLane || k == n-1 {
			if block >= 0 {
				d.block(block, x, top, bottom, bus)
				block, edge = -1, false
			}
			if bus {
				d.lane(sig, ch[k], x, x1, top, bottom)
			} else {
				d.level(ch, k, x, x1, top, bottom, edge)
			}
			x, edge = x1, true
			k++
			continue
		}
		if block < 0 {
			block = x
		}
		lim := d.at(x + minLane)
		k = max(k+1, sort.Search(n, func(i int) bool { return ch[i].At >= lim })) - 1
		x += minLane
	}
	if block >= 0 {
		d.block(block, min(x, right), top, bottom, bus)
	}
	for _, g := range sig.Gaps {
		gx := d.x(g)
		if gx < d.plot.X-d.z.slant || gx > right+d.z.slant {
			continue
		}
		w, y0, y1 := d.z.slant, bottom+d.z.inset/2, top-d.z.inset/2
		var band ui.Path
		band.MoveTo(gx-w, y0).LineTo(gx, y1).LineTo(gx+w/2, y1).LineTo(gx-w/2, y0).Close()
		d.p.FillPath(&band, bg)
		d.p.StrokePolyline([][2]float32{{gx - w, y0}, {gx, y1}}, 1.25, d.t.TextMuted)
		d.p.StrokePolyline([][2]float32{{gx - w/2, y0}, {gx + w/2, y1}}, 1.25, d.t.TextMuted)
	}
}

// block draws changes too close together to tell apart, from x0 to x1.
func (d *drawer) block(x0, x1, top, bottom float32, bus bool) {
	color := d.pal.trace
	if bus {
		color = d.pal.bus
	}
	d.p.Fill(ui.Rect{X: x0, Y: top, W: x1 - x0, H: bottom - top}, color.Alpha(0.5), 0)
}

// extent returns the top and bottom of the line a bit's change draws:
// one height high or low, the middle for z, all of it for x.
func extent(c Change, top, bottom float32) (float32, float32) {
	switch {
	case c.Level == Unknown:
		return top, bottom
	case c.Level == Floating:
		return (top + bottom) / 2, (top + bottom) / 2
	case c.Value&1 == 1:
		return top, top
	}
	return bottom, bottom
}

// level draws change k of a bit's changes from x0 to x1, and the edge
// from the change before at x0 when edge.
func (d *drawer) level(ch []Change, k int, x0, x1, top, bottom float32, edge bool) {
	p := d.p
	const w = 1.5
	c, color := ch[k], d.pal.trace
	y0, y1 := extent(c, top, bottom)
	switch {
	case c.Level == Unknown:
		d.hatch(ui.Rect{X: x0, Y: top, W: x1 - x0, H: bottom - top})
		p.Line(x0, top, x1, top, 1, d.t.TextMuted)
		p.Line(x0, bottom, x1, bottom, 1, d.t.TextMuted)
	case c.Level == Driven && c.Value&1 == 1:
		p.Fill(ui.Rect{X: x0, Y: top, W: x1 - x0, H: bottom - top}, color.Alpha(0.12), 0)
		fallthrough
	default:
		p.Line(x0-w/2, y0, x1+w/2, y0, w, color)
	}
	if edge && k > 0 {
		b0, b1 := extent(ch[k-1], top, bottom)
		if lo, hi := min(y0, b0), max(y1, b1); hi > lo {
			p.Line(x0, lo-w/2, x0, hi+w/2, w, color)
		}
	}
}

// lane draws a bus's change c from x0 to x1: a lane with slanted ends,
// with its value written in when it fits, or a line midway for z.
func (d *drawer) lane(sig Signal, c Change, x0, x1, top, bottom float32) {
	p, t := d.p, d.t
	mid := (top + bottom) / 2
	if c.Level == Floating {
		p.Line(x0, mid, x1, mid, 1.5, d.pal.bus)
		return
	}
	e := min(d.z.slant, (x1-x0)/2)
	var hex ui.Path
	hex.MoveTo(x0, mid).LineTo(x0+e, top).LineTo(x1-e, top).LineTo(x1, mid).LineTo(x1-e, bottom).LineTo(x0+e, bottom).Close()
	color := d.pal.bus
	if c.Level == Unknown {
		color = t.TextMuted
		p.FillPath(&hex, color.Alpha(0.08))
		d.hatch(ui.Rect{X: x0 + e, Y: top, W: x1 - x0 - 2*e, H: bottom - top})
	} else {
		p.FillPath(&hex, color.Alpha(0.14))
	}
	p.StrokePath(&hex, 1.25, color)
	// The text centers in what shows of the lane, to stay in view while
	// the lane is wider than the plot. A label too long for the lane gives
	// way to the value in hexadecimal.
	l, r := max(x0+e, d.plot.X), min(x1-e, d.plot.X+d.plot.W)
	if c.Level == Unknown || r-l < d.z.small {
		return
	}
	texts := [2]string{c.Label}
	if sig.Bits > 1 {
		texts[1] = hexValue(c.Value, sig.Bits)
	}
	for _, text := range texts {
		span := ui.Span{Text: text, Size: d.z.small, Font: "monospace", Color: t.Text}
		if w, h := p.MeasureText(0, span); text != "" && w+t.Space(2) <= r-l {
			p.RichText((l+r-w)/2, mid-h/2, 0, span)
			return
		}
	}
}

// hatch strokes r with lines slanting up to the right, for what is
// unknown.
func (d *drawer) hatch(r ui.Rect) {
	if r.W <= 0 {
		return
	}
	const gap = 6
	var path ui.Path
	for x := float32(math.Floor(float64((r.X-r.H)/gap))) * gap; x < r.X+r.W; x += gap {
		path.MoveTo(x, r.Y+r.H).LineTo(x+r.H, r.Y)
	}
	d.p.Clip(r, 0, func() { d.p.StrokePath(&path, 1, d.t.TextMuted.Alpha(0.6)) })
}

// valueAt returns what sig is at time at, as the pointer reads it out, or
// "" before its first change.
func valueAt(sig Signal, at float64) string {
	i := sort.Search(len(sig.Changes), func(i int) bool { return sig.Changes[i].At > at }) - 1
	if i < 0 {
		return ""
	}
	c := sig.Changes[i]
	switch {
	case c.Level == Unknown:
		return "x"
	case c.Level == Floating:
		return "z"
	case sig.Bits == 1:
		return strconv.FormatUint(c.Value&1, 10)
	case sig.Bits <= 0:
		return c.Label
	}
	v := "0x" + hexValue(c.Value, sig.Bits)
	if c.Label != "" {
		v += "  " + c.Label
	}
	return v
}

// hexValue writes the low bits bits of v in as many hexadecimal digits as
// they need.
func hexValue(v uint64, bits int) string {
	if bits < 64 {
		v &= 1<<bits - 1
	}
	s := strings.ToUpper(strconv.FormatUint(v, 16))
	if pad := (min(bits, 64)+3)/4 - len(s); pad > 0 {
		s = strings.Repeat("0", pad) + s
	}
	return s
}

// nearest returns the time of the change nearest at, if one is within tol
// of it.
func nearest(changes []Change, at, tol float64) (float64, bool) {
	i := sort.Search(len(changes), func(i int) bool { return changes[i].At >= at })
	best, ok := 0.0, false
	for _, j := range [2]int{i - 1, i} {
		if j < 0 || j >= len(changes) {
			continue
		}
		if d := math.Abs(changes[j].At - at); d <= tol && (!ok || d < math.Abs(best-at)) {
			best, ok = changes[j].At, true
		}
	}
	return best, ok
}

// niceStep returns the step of 1, 2 or 5 times a power of ten at or above
// raw, and how many parts its ticks split it into.
func niceStep(raw float64) (float64, int) {
	p := math.Pow(10, math.Floor(math.Log10(raw)))
	switch m := raw / p; {
	case m <= 1:
		return p, 5
	case m <= 2:
		return 2 * p, 4
	case m <= 5:
		return 5 * p, 5
	}
	return 10 * p, 5
}

// unitOf returns the unit to write seconds near v in, and its symbol.
func unitOf(v float64) (float64, string) {
	switch a := math.Abs(v); {
	case a >= 1:
		return 1, "s"
	case a >= 1e-3:
		return 1e-3, "ms"
	case a >= 1e-6:
		return 1e-6, "µs"
	}
	return 1e-9, "ns"
}

// duration writes v seconds in its unit, as 1.25 µs.
func duration(v float64) string {
	unit, name := unitOf(v)
	return number(v/unit, 6) + " " + name
}

// frequency writes f hertz, as 400 kHz.
func frequency(f float64) string {
	for _, u := range []struct {
		at   float64
		name string
	}{{1e9, "GHz"}, {1e6, "MHz"}, {1e3, "kHz"}} {
		if f >= u.at {
			return number(f/u.at, 4) + " " + u.name
		}
	}
	return number(f, 4) + " Hz"
}

// number writes v with sig significant digits, without trailing zeros.
func number(v float64, sig int) string {
	whole := 0
	if a := math.Abs(v); a >= 1 {
		whole = int(math.Floor(math.Log10(a))) + 1
	}
	s := strconv.FormatFloat(v, 'f', max(0, sig-whole), 64)
	if strings.Contains(s, ".") {
		s = strings.TrimRight(strings.TrimRight(s, "0"), ".")
	}
	return s
}
