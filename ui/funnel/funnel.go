// Package funnel shows how many go on through the steps of a conversion
// funnel, as product analytics tools do: the stages from the first to the
// last, each a bar as long as its count is of the first's, centered, and a
// band narrowing from it to the next. Beside each stage are its name, its
// count and its share of the first; between two stages, the share of the
// one that went on to the next and how many dropped off. The stages run
// down, or across with Horizontal. Numbers are written in the user's
// regional format.
//
// Two or more segments, as this month and the last or each platform, show
// as a group of bars in each stage, each as long as its count is of its
// own first stage's, in the colors of a legend above. Pointing at a stage
// lights it, with a card of its numbers; the app hears of a click on it in
// the frame it happens:
//
//	ev := funnel.View(c, funnel.Vertical, stages, []funnel.Segment{{Name: "October", Counts: october}})
//	if i, ok := ev.Clicked(); ok {
//		app.stage = i
//	}
package funnel

import (
	"strings"

	"github.com/egoist/mygo/i18n"
	"github.com/egoist/mygo/ui"
)

// Direction is which way a funnel's stages run.
type Direction uint8

const (
	// Vertical runs the stages down, each a bar across.
	Vertical Direction = iota
	// Horizontal runs the stages across, each a bar upright.
	Horizontal
)

// Segment is a group of those who went through the funnel, as a month's or
// a platform's: its name, for the legend, and its count at each stage, by
// the stages' indexes. A count missing is 0.
type Segment struct {
	Name   string
	Counts []float64
}

func (s Segment) at(i int) float64 {
	if i < len(s.Counts) {
		return max(0, s.Counts[i])
	}
	return 0
}

// Events is what a funnel did in a frame.
type Events struct {
	clicked int
	ok      bool
}

// Clicked returns the stage the user clicked in this frame.
func (e Events) Clicked() (int, bool) { return e.clicked, e.ok }

// ratio returns n of of, 0 when of is none.
func ratio(n, of float64) float64 {
	if of <= 0 {
		return 0
	}
	return n / of
}

// words is what a funnel writes, by stage and segment: each count and its
// share of the first stage, and of each step to the next stage the share
// that went on and how many dropped off.
type words struct {
	count, share, step, drop [][]string
}

func write(locale string, n int, segments []Segment) words {
	whole := i18n.NumberOptions{Integer: true}
	pct := i18n.NumberOptions{Percent: true, MaxFractionDigits: 1}
	grid := func() [][]string {
		g := make([][]string, n)
		for i := range g {
			g[i] = make([]string, len(segments))
		}
		return g
	}
	w := words{grid(), grid(), grid(), grid()}
	for i := range n {
		for s, seg := range segments {
			w.count[i][s] = i18n.FormatNumber(locale, seg.at(i), whole)
			w.share[i][s] = i18n.FormatNumber(locale, ratio(seg.at(i), seg.at(0)), pct)
			if i+1 < n {
				w.step[i][s] = i18n.FormatNumber(locale, ratio(seg.at(i+1), seg.at(i)), pct)
				w.drop[i][s] = i18n.FormatNumber(locale, max(0, seg.at(i)-seg.at(i+1)), whole)
			}
		}
	}
	return w
}

// geometry is where a funnel's parts go, along its stages, down or across,
// and across them: stage i's bars start at start+i*(group+link), within
// group, a bar of each segment bar thick with gap between them, centered
// on mid and size long at most; link is the room between two stages, which
// the bands between them span.
type geometry struct {
	dir                                     Direction
	start, group, link, bar, gap, mid, size float32
}

// newGeometry lays out n stages of m segments in w×h, leaving room for
// labels of label and steps of step, each a width and a height.
func newGeometry(t *ui.Theme, dir Direction, w, h float32, n, m int, label, step [2]float32) geometry {
	g := geometry{dir: dir}
	along := w
	if dir == Vertical {
		left, right := label[0]+t.Space(5), w-step[0]-t.Space(5)
		g.mid, g.size = (left+right)/2, max(0, right-left)
		g.group = max(label[1], min(t.Space(18), h/(float32(n)+float32(n-1)*0.6)))
		g.link = max(g.group*0.6, step[1]+t.Space(2))
		along = h
	} else {
		top, bottom := label[1]+t.Space(3), h-step[1]-t.Space(3)
		g.mid, g.size = (top+bottom)/2, max(0, bottom-top)
		g.group = min(t.Space(30), w/float32(2*n-1))
		g.link = g.group
	}
	g.start = max(0, (along-float32(n)*g.group-float32(n-1)*g.link)/2)
	if m > 1 {
		g.gap = t.Space(1)
	}
	g.bar = max(1, (g.group-g.gap*float32(m-1))/float32(max(1, m)))
	return g
}

// at returns where stage i starts along the funnel.
func (g geometry) at(i int) float32 { return g.start + float32(i)*(g.group+g.link) }

// rect returns the box from along a, l long, and across c, k wide.
func (g geometry) rect(a, l, c, k float32) ui.Rect {
	if g.dir == Vertical {
		return ui.Rect{X: c, Y: a, W: k, H: l}
	}
	return ui.Rect{X: a, Y: c, W: l, H: k}
}

// point returns the point at a along the funnel and c across it.
func (g geometry) point(a, c float32) (float32, float32) {
	if g.dir == Vertical {
		return c, a
	}
	return a, c
}

// length returns how long a bar of share f of its first stage is: a
// sliver at least, for any count.
func (g geometry) length(f float64) float32 {
	if f <= 0 {
		return 0
	}
	return max(2, g.size*float32(min(1, f)))
}

// barRect returns segment s's bar of stage i, f of its first stage's.
func (g geometry) barRect(i, s int, f float64) ui.Rect {
	l := g.length(f)
	return g.rect(g.at(i)+float32(s)*(g.bar+g.gap), g.bar, g.mid-l/2, l)
}

// band returns the trapezoid between stages i and i+1 of bars f0 and f1 of
// their first stage.
func (g geometry) band(i int, f0, f1 float64) *ui.Path {
	a0, a1 := g.at(i)+g.group, g.at(i+1)
	l0, l1 := g.length(f0)/2, g.length(f1)/2
	var p ui.Path
	p.MoveTo(g.point(a0, g.mid-l0)).LineTo(g.point(a0, g.mid+l0)).
		LineTo(g.point(a1, g.mid+l1)).LineTo(g.point(a1, g.mid-l1)).Close()
	return &p
}

// hit returns the stage nearest x, y along the funnel, -1 beyond its ends.
func (g geometry) hit(x, y float32, n int) int {
	a := y
	if g.dir == Horizontal {
		a = x
	}
	f := a - g.start + g.link/2
	if f < 0 {
		return -1
	}
	if i := int(f / (g.group + g.link)); i < n {
		return i
	}
	return -1
}

// View creates the funnel of stages, through which each segment's counts
// go, in dir. It fills its parent. The Events returned tell which stage
// was clicked.
func View(c *ui.Context, dir Direction, stages []string, segments []Segment) Events {
	t := c.Theme()
	n, m := len(stages), len(segments)
	w := write(c.Region().FormatLocale(c.Locale()), n, segments)
	small := t.FontSize - 1
	dot := t.Space(1.5)
	indent := float32(0)
	if m > 1 {
		indent = dot + t.Space(1.5)
	}
	name := func(i int) ui.Span { return ui.Span{Text: stages[i], Size: t.FontSize, Weight: 600, Color: t.Text} }
	counts := func(i, s int) []ui.Span {
		spans := []ui.Span{{Text: w.count[i][s], Size: small, Color: t.Text, Features: "tnum"}}
		if i > 0 {
			spans = append(spans, ui.Span{Text: "  " + w.share[i][s], Size: small, Color: t.TextMuted, Features: "tnum"})
		}
		return spans
	}
	steps := func(i, s int) []ui.Span {
		return []ui.Span{
			{Text: w.step[i][s], Size: small, Weight: 600, Color: t.Text, Features: "tnum"},
			{Text: "  " + w.drop[i][s] + " dropped", Size: small, Color: t.TextMuted, Features: "tnum"},
		}
	}
	_, nameH := c.MeasureText(0, ui.Span{Text: "M", Size: t.FontSize, Weight: 600})
	_, lineH := c.MeasureText(0, ui.Span{Text: "0", Size: small})
	var label, step [2]float32
	for i := range n {
		lw, _ := c.MeasureText(0, name(i))
		label[0] = max(label[0], lw)
		for s := range segments {
			lw, _ = c.MeasureText(0, counts(i, s)...)
			label[0] = max(label[0], indent+lw)
			if i+1 < n {
				lw, _ = c.MeasureText(0, steps(i, s)...)
				step[0] = max(step[0], indent+lw)
			}
		}
	}
	label[1] = nameH + float32(m)*lineH
	step[1] = float32(m) * lineH

	var summary []string
	if m > 0 {
		for i := range n {
			summary = append(summary, stages[i]+" "+w.count[i][0])
		}
	}

	var ev Events
	ui.Column(c).Fill().Gap(t.Space(3)).Children(func() {
		if m > 1 {
			legend(c, segments)
		}
		box := ui.Box(c).Grow(1).Shrink(1).AlignSelf(ui.Stretch).Label("Funnel").Value(strings.Join(summary, ", "))
		b := box.LayoutBounds()
		g := newGeometry(t, dir, b.W, b.H, n, m, label, step)
		hot := -1
		if px, py, ok := box.PointerPosition(); ok && n > 0 {
			hot = g.hit(px, py, n)
		}
		if hot >= 0 {
			box.Cursor(ui.CursorPointer)
			if box.Clicked() {
				ev = Events{hot, true}
			}
			widest := float32(1)
			for _, seg := range segments {
				widest = max(widest, g.length(ratio(seg.at(hot), seg.at(0))))
			}
			bars := g.rect(g.at(hot), g.group, g.mid-widest/2, widest)
			box.Children(func() {
				anchor := ui.Box(c).Absolute().PassThrough().Left(bars.X).Top(bars.Y).Size(max(1, bars.W), max(1, bars.H))
				card(c, anchor, dir, stages, segments, w, hot)
			})
		}

		box.Draw(func(p *ui.Painter, r ui.Rect) {
			g := newGeometry(t, dir, r.W, r.H, n, m, label, step)
			p.Transform(ui.Translation(r.X, r.Y), 0, 0, func() {
				if hot >= 0 {
					across := r.W
					if dir == Horizontal {
						across = r.H
					}
					p.Fill(g.rect(g.at(hot)-g.link/4, g.group+g.link/2, 0, across), t.Text.Alpha(0.05), t.Space(1.5))
				}
				// Bands of more than one segment overlap, and would blend into
				// a muddy color of their own.
				bandAlpha := float32(0.16)
				if m > 1 {
					bandAlpha = 0.09
				}
				for i := 0; i+1 < n; i++ {
					for s, seg := range segments {
						f0, f1 := ratio(seg.at(i), seg.at(0)), ratio(seg.at(i+1), seg.at(0))
						p.FillPath(g.band(i, f0, f1), hue(s, m, t).Alpha(bandAlpha))
					}
				}
				for i := range n {
					for s, seg := range segments {
						color := hue(s, m, t)
						if hot >= 0 && i != hot {
							color = color.Alpha(0.6)
						}
						p.Fill(g.barRect(i, s, ratio(seg.at(i), seg.at(0))), color, min(t.Space(0.75), g.bar/2))
					}
				}
				// lines draws a line for each segment from x, y, with its dot
				// when there are more than one, centered on x when center.
				lines := func(x, y float32, center bool, spans func(s int) []ui.Span) {
					for s := range segments {
						sp := spans(s)
						lw, _ := p.MeasureText(0, sp...)
						lx := x
						if center {
							lx = x - (indent+lw)/2
						}
						if m > 1 {
							p.Fill(ui.Rect{X: lx, Y: y + (lineH-dot)/2, W: dot, H: dot}, hue(s, m, t), dot/2)
						}
						p.RichText(lx+indent, y, 0, sp...)
						y += lineH
					}
				}
				for i := range n {
					nw, _ := p.MeasureText(0, name(i))
					if dir == Vertical {
						y := g.at(i) + (g.group-label[1])/2
						p.RichText(0, y, 0, name(i))
						lines(0, y+nameH, false, func(s int) []ui.Span { return counts(i, s) })
					} else {
						x := g.at(i) + g.group/2
						p.RichText(x-nw/2, 0, 0, name(i))
						lines(x, nameH, true, func(s int) []ui.Span { return counts(i, s) })
					}
					if i+1 == n {
						continue
					}
					if dir == Vertical {
						lines(r.W-step[0], g.at(i)+g.group+(g.link-step[1])/2, false, func(s int) []ui.Span { return steps(i, s) })
					} else {
						lines(g.at(i)+g.group+g.link/2, r.H-step[1], true, func(s int) []ui.Span { return steps(i, s) })
					}
				}
			})
		})
	})
	return ev
}

// hue returns segment s's color of m: the accent when it is alone, else
// one around the wheel by the golden angle, so that segments near one
// another differ.
func hue(s, m int, t *ui.Theme) ui.Color {
	if m == 1 {
		return t.Accent
	}
	return ui.Oklch(0.66, 0.14, float32((250+s*137)%360))
}

// legend shows the segments' colors and names.
func legend(c *ui.Context, segments []Segment) {
	t := c.Theme()
	ui.Row(c).Gap(t.Space(4)).Wrap().Children(func() {
		for s, seg := range segments {
			ui.Row(c).Gap(t.Space(1.5)).AlignItems(ui.Center).Children(func() {
				ui.Box(c).Size(t.Space(2.5), t.Space(2.5)).Radius(t.Space(0.75)).Background(hue(s, len(segments), t))
				ui.Text(c, seg.Name).FontSize(t.FontSize - 1).TextColor(t.TextMuted)
			})
		}
	})
}

// card shows by anchor, the bars of stage i, the card of its numbers: for
// each segment its count, its share of the first stage, and of the stage
// before it, with how many dropped off since.
func card(c *ui.Context, anchor ui.Element, dir Direction, stages []string, segments []Segment, w words, i int) {
	t := c.Theme()
	at, self := ui.AnchorLeft, ui.AnchorRight
	if dir == Horizontal {
		at, self = ui.AnchorTopRight, ui.AnchorTopLeft
	}
	ui.Overlay(c, func() {
		// The card and all in it let the pointer through to the funnel
		// under it, which would otherwise lose the hover and the card.
		tip := ui.Column(c).Absolute().PassThrough().Role(ui.RoleTooltip).MaxWidth(t.Space(72)).
			AttachTo(anchor, at, self).Margin(t.Space(2)).
			Padding(t.Space(2.5), t.Space(3)).Gap(t.Space(2)).Radius(t.Space(2)).
			Background(t.Background).Border(1, t.Border).Shadow(0, 4, 16, 0, ui.RGBA(0, 0, 0, 0.18))
		tip.Children(func() {
			ui.Text(c, stages[i]).PassThrough().Bold()
			for s, seg := range segments {
				ui.Column(c).PassThrough().Gap(t.Space(0.5)).Children(func() {
					if len(segments) > 1 {
						ui.Row(c).PassThrough().Gap(t.Space(1.5)).AlignItems(ui.Center).Children(func() {
							ui.Box(c).PassThrough().Size(t.Space(2), t.Space(2)).Radius(t.Space(1)).Background(hue(s, len(segments), t))
							ui.Text(c, seg.Name).PassThrough().FontSize(t.FontSize - 1).TextColor(t.TextMuted)
						})
					}
					ui.Row(c).PassThrough().Gap(t.Space(1.5)).AlignItems(ui.End).Children(func() {
						ui.Text(c, w.count[i][s]).PassThrough().FontSize(t.FontSize + 4).FontWeight(600).FontFeatures("tnum")
						if i > 0 {
							ui.Text(c, w.share[i][s]+" of "+stages[0]).PassThrough().TextColor(t.TextMuted).FontFeatures("tnum")
						}
					})
					if i > 0 {
						line := w.drop[i-1][s] + " dropped off"
						if i > 1 {
							line = w.step[i-1][s] + " of " + stages[i-1] + " · " + line
						}
						ui.Text(c, line).PassThrough().FontSize(t.FontSize - 1).TextColor(t.TextMuted).FontFeatures("tnum")
					}
				})
			}
		})
	})
}
