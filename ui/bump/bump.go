// Package bump shows how the ranks of things change over time, as bump
// charts of league tables and of the most used languages do: the periods
// across the top, rank 1 at the top, and a line of its color for each
// series through a disc at its rank in each period, the rank written in
// the disc and the name at both ends. Ranks come from the series' values,
// the highest first, in each period; a series without a value in a period
// leaves a gap there.
//
// The pointer on a disc or a name lights its series and dims the others,
// and shows a card of its value and rank in that period; a click keeps
// the series lit until another click.
//
//	bump.View(c, &app.chart, []string{"2021", "2022", "2023"}, []bump.Series{
//		{Name: "Go", Values: []float64{9.6, 11.2, 13.8}},
//		...
//	}).Grow(1)
package bump

import (
	"math"
	"slices"
	"strconv"

	"github.com/egoist/mygo/ui"
)

// Series is a thing ranked: its name, its value in each period, NaN where
// it has none, and its color; the zero Color is one of a set chosen by its
// place among the series.
type Series struct {
	Name   string
	Values []float64
	Color  ui.Color
}

// State is what a chart keeps: the series kept lit. Keep it in the app's
// state, one for each chart.
type State struct {
	pinned int
}

// View creates the bump chart of series over periods. Give it a size, or
// grow it in its parent.
func View(c *ui.Context, s *State, periods []string, series []Series) ui.Element {
	t := c.Theme()
	ranks := rank(periods, series)
	most := 0
	for _, col := range ranks {
		for _, r := range col {
			most = max(most, r)
		}
	}
	box := ui.Box(c).Role(ui.RoleImage).Label("Bump chart")
	b := box.LayoutBounds()
	small := t.FontSize - 1
	nameW := float32(0)
	for _, sr := range series {
		w, _ := c.MeasureText(0, ui.Span{Text: sr.Name, Size: small, Weight: 600})
		nameW = max(nameW, w)
	}
	nameW = min(nameW+t.Space(3), b.W/5)
	plot := ui.Rect{X: nameW, Y: t.FontSize + t.Space(5), W: b.W - 2*nameW, H: b.H - t.FontSize - t.Space(7)}
	radius := min(t.Space(3.5), plot.H/float32(max(1, most))/2.6)
	px := func(p int) float32 {
		if len(periods) < 2 {
			return plot.X + plot.W/2
		}
		return plot.X + radius + float32(p)*(plot.W-2*radius)/float32(len(periods)-1)
	}
	py := func(r int) float32 {
		if most < 2 {
			return plot.Y + plot.H/2
		}
		return plot.Y + radius + float32(r-1)*(plot.H-2*radius)/float32(most-1)
	}
	hot, at := -1, -1
	if x, y, ok := box.PointerPosition(); ok && box.Hovered() {
		for i := range series {
			for p := range periods {
				if r := ranks[p][i]; r > 0 && math.Hypot(float64(x-px(p)), float64(y-py(r))) <= float64(radius+3) {
					hot, at = i, p
				}
			}
			first, last := ends(ranks, i)
			if first >= 0 && y > py(ranks[first][i])-radius && y < py(ranks[first][i])+radius {
				if x < px(first)-radius {
					hot, at = i, first
				} else if x > px(last)+radius {
					hot, at = i, last
				}
			}
		}
		if hot >= 0 {
			box.Cursor(ui.CursorPointer)
		}
	}
	if box.Clicked() {
		if hot >= 0 && s.pinned != hot+1 {
			s.pinned = hot + 1
		} else {
			s.pinned = 0
		}
	}
	lit := hot
	if lit < 0 {
		lit = s.pinned - 1
	}
	box.Draw(func(p *ui.Painter, r ui.Rect) {
		for k, name := range periods {
			span := ui.Span{Text: name, Size: small, Weight: 600, Color: t.TextMuted}
			w, _ := p.MeasureText(0, span)
			x := r.X + px(k)
			p.RichText(x-w/2, r.Y, 0, span)
			p.Line(x, r.Y+plot.Y-t.Space(1), x, r.Y+plot.Y+plot.H, 1, t.Border.Alpha(0.5))
		}
		order := make([]int, len(series))
		for i := range order {
			order[i] = i
		}
		// The series lit goes over the others.
		if lit >= 0 {
			order = append(slices.Delete(order, lit, lit+1), lit)
		}
		for _, i := range order {
			col := colorOf(series, i, t.Dark)
			if lit >= 0 && i != lit {
				col = col.Alpha(0.15)
			}
			width := float32(3)
			if i == lit {
				width = 4
			}
			var path ui.Path
			open := false
			for k := range periods {
				rk := ranks[k][i]
				if rk == 0 {
					if open {
						p.StrokePath(&path, width, col)
						path.Reset()
						open = false
					}
					continue
				}
				x, y := r.X+px(k), r.Y+py(rk)
				if !open {
					path.MoveTo(x, y)
					open = true
				} else {
					x0 := r.X + px(k-1)
					y0 := r.Y + py(ranks[k-1][i])
					mid := (x0 + x) / 2
					path.CubeTo(mid, y0, mid, y, x, y)
				}
			}
			if open {
				p.StrokePath(&path, width, col)
			}
			for k := range periods {
				rk := ranks[k][i]
				if rk == 0 {
					continue
				}
				x, y := r.X+px(k), r.Y+py(rk)
				p.Fill(ui.Rect{X: x - radius, Y: y - radius, W: 2 * radius, H: 2 * radius}, col.Over(t.Background), radius)
				if lit < 0 || i == lit {
					num := ui.Span{Text: strconv.Itoa(rk), Size: radius * 1.05, Weight: 700, Color: ui.RGB(255, 255, 255), Features: "tnum"}
					w, h := p.MeasureText(0, num)
					p.RichText(x-w/2, y-h/2, 0, num)
				}
			}
			first, last := ends(ranks, i)
			if first < 0 {
				continue
			}
			ink := t.Text
			if lit >= 0 && i != lit {
				ink = t.TextMuted.Alpha(0.5)
			}
			name := ui.Span{Text: series[i].Name, Size: small, Weight: 600, Color: ink}
			w, h := p.MeasureText(0, name)
			p.RichText(r.X+px(first)-radius-t.Space(1.5)-w, r.Y+py(ranks[first][i])-h/2, 0, name)
			p.RichText(r.X+px(last)+radius+t.Space(1.5), r.Y+py(ranks[last][i])-h/2, 0, name)
		}
	})
	if hot >= 0 {
		v := math.NaN()
		if at < len(series[hot].Values) {
			v = series[hot].Values[at]
		}
		x, y := px(at), py(ranks[at][hot])
		box.Children(func() {
			anchor := ui.Box(c).Absolute().PassThrough().Left(x-radius).Top(y-radius).Size(2*radius, 2*radius)
			card(c, anchor, series[hot].Name, periods[at], ranks[at][hot], v, colorOf(series, hot, t.Dark))
		})
	}
	return box
}

// rank returns the rank of each series in each period, by period, the
// highest value 1 and 0 for none.
func rank(periods []string, series []Series) [][]int {
	ranks := make([][]int, len(periods))
	for p := range periods {
		ranks[p] = make([]int, len(series))
		var in []int
		for i, sr := range series {
			if p < len(sr.Values) && !math.IsNaN(sr.Values[p]) {
				in = append(in, i)
			}
		}
		slices.SortStableFunc(in, func(a, b int) int {
			va, vb := series[a].Values[p], series[b].Values[p]
			switch {
			case va > vb:
				return -1
			case va < vb:
				return 1
			}
			return 0
		})
		for k, i := range in {
			ranks[p][i] = k + 1
		}
	}
	return ranks
}

// ends returns the first and last periods series i is ranked in, or -1.
func ends(ranks [][]int, i int) (first, last int) {
	first, last = -1, -1
	for p := range ranks {
		if ranks[p][i] > 0 {
			if first < 0 {
				first = p
			}
			last = p
		}
	}
	return first, last
}

// colorOf returns the color of series i.
func colorOf(series []Series, i int, dark bool) ui.Color {
	if c := series[i].Color; c != (ui.Color{}) {
		return c
	}
	l := float32(0.6)
	if dark {
		l = 0.7
	}
	return ui.Oklch(l, 0.15, float32((250+i*137)%360))
}

// card shows a series' rank and value in a period by anchor.
func card(c *ui.Context, anchor ui.Element, name, period string, rank int, v float64, col ui.Color) {
	t := c.Theme()
	ui.Overlay(c, func() {
		tip := ui.Column(c).Absolute().PassThrough().Role(ui.RoleTooltip).
			AttachTo(anchor, ui.AnchorTop, ui.AnchorBottom).Margin(0, 0, t.Space(1.5), 0).
			Padding(t.Space(1.5), t.Space(2.5)).Gap(t.Space(0.5)).Radius(t.Space(1.5)).
			Background(t.Background).Border(1, t.Border).Shadow(0, 4, 16, 0, ui.RGBA(0, 0, 0, 0.18))
		tip.Children(func() {
			ui.Row(c).Gap(t.Space(1.5)).AlignItems(ui.Center).PassThrough().Children(func() {
				ui.Box(c).Size(t.Space(2), t.Space(2)).Radius(t.Space(1)).Background(col).PassThrough()
				ui.Text(c, name).Bold().PassThrough().SingleLine()
			})
			text := period + " · #" + strconv.Itoa(rank)
			if !math.IsNaN(v) {
				text += " · " + strconv.FormatFloat(v, 'f', -1, 64)
			}
			ui.Text(c, text).FontSize(t.FontSize - 1).TextColor(t.TextMuted).FontFeatures("tnum").PassThrough().SingleLine()
		})
	})
}
