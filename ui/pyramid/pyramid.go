// Package pyramid draws a population pyramid, back-to-back bars as
// censuses show ages by sex: a row for each group, the youngest at the
// bottom, its name in a column down the middle, one side's bar growing
// to the left and the other's to the right, both on one scale with its
// ticks under them, and each side's name above it. A side can carry an
// earlier count, drawn as a step outline over its bars, so that what
// grew and what shrank shows. Shares shows each bar as a share of the
// whole rather than its count. The pointer on a row lights it and shows
// a card of both sides' counts, shares and change.
//
//	pyramid.View(c, ages, pyramid.Side{Name: "Men", Values: men}, pyramid.Side{Name: "Women", Values: women}, true)
package pyramid

import (
	"fmt"
	"math"
	"strconv"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/chart"
)

// Side is one side of a pyramid: its name, a value for each group, and
// the earlier values, or nil.
type Side struct {
	Name   string
	Values []float64
	Before []float64
}

// Colors returns the colors of the left and the right side.
func Colors(t *ui.Theme) (left, right ui.Color) {
	l := float32(0.62)
	if t.Dark {
		l = 0.7
	}
	return ui.Oklch(l, 0.12, 245), ui.Oklch(l, 0.13, 25)
}

// View creates the pyramid of groups, from the youngest, with left and
// right, as shares of the whole when shares is set. Give it a size, or
// grow it in its parent.
func View(c *ui.Context, groups []string, left, right Side, shares bool) ui.Element {
	t := c.Theme()
	n := len(groups)
	total, before := sum(left.Values)+sum(right.Values), sum(left.Before)+sum(right.Before)
	at := func(vs []float64, i int, whole float64) float64 {
		if i >= len(vs) {
			return math.NaN()
		}
		if shares && whole > 0 {
			return vs[i] / whole * 100
		}
		return vs[i]
	}
	peak := 0.0
	for i := range n {
		for _, v := range []float64{at(left.Values, i, total), at(right.Values, i, total), at(left.Before, i, before), at(right.Before, i, before)} {
			if !math.IsNaN(v) {
				peak = max(peak, v)
			}
		}
	}
	scale := chart.NewLinear(0, peak).Nice(4)
	small := t.FontSize - 2
	mid := t.Space(14)
	for _, g := range groups {
		w, _ := c.MeasureText(0, ui.Span{Text: g, Size: small})
		mid = max(mid, w+t.Space(4))
	}
	_, lineH := c.MeasureText(0, ui.Span{Text: "M", Size: small})
	top, bottom := lineH+t.Space(3), lineH+t.Space(3)
	box := ui.Box(c).MinWidth(t.Space(100)).MinHeight(t.Space(70)).Role(ui.RoleImage).Label("Population pyramid")
	b := box.LayoutBounds()
	half := max(1, (b.W-mid)/2)
	rowH := max(1, (b.H-top-bottom)/float32(max(n, 1)))
	hot := -1
	if _, py, over := box.PointerPosition(); over && py >= top && py < top+rowH*float32(n) {
		hot = n - 1 - int((py-top)/rowH)
	}
	lc, rc := Colors(t)
	box.Draw(func(p *ui.Painter, r ui.Rect) {
		if r.W != b.W || r.H != b.H {
			c.Invalidate()
		}
		cx := r.X + half + mid/2
		lx, rx := r.X+half, r.X+half+mid
		rowY := func(i int) float32 { return r.Y + top + float32(n-1-i)*rowH }
		for _, tk := range scale.Ticks(4) {
			for _, x := range []float32{lx - tk.At*half, rx + tk.At*half} {
				p.Line(x, r.Y+top, x, r.Y+top+rowH*float32(n), 1, t.Text.Alpha(0.06))
				label := tk.Label
				if shares {
					label += "%"
				}
				w, _ := p.MeasureText(0, ui.Span{Text: label, Size: small})
				p.Text(x-w/2, r.Y+top+rowH*float32(n)+t.Space(1.5), label, small, t.TextMuted)
			}
		}
		lw, _ := p.MeasureText(0, ui.Span{Text: left.Name, Size: small, Weight: 600})
		p.RichText(lx-lw, r.Y, 0, ui.Span{Text: left.Name, Size: small, Weight: 600, Color: lc})
		p.RichText(rx, r.Y, 0, ui.Span{Text: right.Name, Size: small, Weight: 600, Color: rc})
		gap := min(t.Space(0.75), rowH*0.15)
		for i := range n {
			y := rowY(i)
			if i == hot {
				p.Fill(ui.Rect{X: r.X, Y: y, W: r.W, H: rowH}, t.Text.Alpha(0.05), t.Radius)
			}
			if v := at(left.Values, i, total); !math.IsNaN(v) {
				w := scale.At(v) * half
				p.Fill(ui.Rect{X: lx - w, Y: y + gap, W: w, H: rowH - 2*gap}, lc, 2)
			}
			if v := at(right.Values, i, total); !math.IsNaN(v) {
				w := scale.At(v) * half
				p.Fill(ui.Rect{X: rx, Y: y + gap, W: w, H: rowH - 2*gap}, rc, 2)
			}
			sp := ui.Span{Text: groups[i], Size: small, Color: t.TextMuted, Features: "tnum"}
			if i == hot {
				sp.Color, sp.Weight = t.Text, 600
			}
			w, h := p.MeasureText(0, sp)
			p.RichText(cx-w/2, y+(rowH-h)/2, 0, sp)
		}
		outline := func(vs []float64, x float32, dir float32) {
			if len(vs) == 0 {
				return
			}
			var pts [][2]float32
			for i := range n {
				v := at(vs, i, before)
				if math.IsNaN(v) {
					continue
				}
				ex := x + dir*scale.At(v)*half
				y := rowY(i)
				pts = append(pts, [2]float32{ex, y + rowH}, [2]float32{ex, y})
			}
			p.StrokePolyline(pts, 1.5, t.Text.Alpha(0.75))
		}
		outline(left.Before, lx, -1)
		outline(right.Before, rx, 1)
		if hot >= 0 {
			card(p, t, r, cx, rowY(hot)+rowH/2, groups[hot], []Side{left, right}, hot, total, before, [2]ui.Color{lc, rc})
		}
	})
	return box
}

// card paints the card of group i at x, y: each side's count and share,
// and its change from before.
func card(p *ui.Painter, t *ui.Theme, r ui.Rect, x, y float32, group string, sides []Side, i int, total, before float64, cols [2]ui.Color) {
	small := t.FontSize - 2
	lines := [][]ui.Span{{{Text: group, Size: t.FontSize - 1, Weight: 600, Color: t.Text}}}
	for k, s := range sides {
		if i >= len(s.Values) {
			continue
		}
		v := s.Values[i]
		line := []ui.Span{
			{Text: "● ", Size: small, Color: cols[k]},
			{Text: s.Name + "  ", Size: small, Color: t.TextMuted},
			{Text: number(v), Size: small, Color: t.Text, Weight: 600, Features: "tnum"},
			{Text: fmt.Sprintf("  %.1f%%", v/total*100), Size: small, Color: t.TextMuted, Features: "tnum"},
		}
		if i < len(s.Before) && s.Before[i] > 0 {
			ch := (v - s.Before[i]) / s.Before[i] * 100
			col := t.Success
			if ch < 0 {
				col = t.Danger
			}
			line = append(line, ui.Span{Text: fmt.Sprintf("  %+.1f%%", ch), Size: small, Color: col, Features: "tnum"})
		}
		lines = append(lines, line)
	}
	pad := t.Space(2.5)
	w, h := float32(0), float32(0)
	for _, l := range lines {
		lw, lh := p.MeasureText(0, l...)
		w, h = max(w, lw), h+lh+t.Space(0.5)
	}
	w, h = w+2*pad, h+2*pad
	cx := min(x+t.Space(16), r.X+r.W-w)
	cy := max(r.Y, min(y-h-t.Space(2), r.Y+r.H-h))
	box := ui.Rect{X: cx, Y: cy, W: w, H: h}
	p.Shadow(box, t.Radius+2, 0, 2, 10, 0, ui.RGBA(0, 0, 0, 0.2))
	p.Fill(box, t.Background, t.Radius+2)
	p.Stroke(box, t.Border, t.Radius+2, 1)
	ty := cy + pad
	for _, l := range lines {
		_, lh := p.RichText(cx+pad, ty, 0, l...)
		ty += lh + t.Space(0.5)
	}
}

func sum(vs []float64) float64 {
	s := 0.0
	for _, v := range vs {
		s += v
	}
	return s
}

// number writes v with thousands separated.
func number(v float64) string {
	s := strconv.FormatInt(int64(math.Round(v)), 10)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}
