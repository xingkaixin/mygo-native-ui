// Package mosaic draws a Marimekko chart, a mosaic of two ways of
// splitting a whole, as market maps show segments and the companies in
// them: a column for each category, as wide as its share of the total,
// split from the top into a block for each series, as tall as its share
// of the category. A block with room for it shows its series' name and
// its share of the column. Under each column, its name and its share of
// the total; above the chart, a legend of the series. The pointer on a
// block lights it and the other blocks of its series and shows a card of
// its value, its share of its column and of the total.
//
//	mosaic.View(c, []string{"Retail", "Banking"}, []string{"Acme", "Globex"}, values).Grow(1)
package mosaic

import (
	"fmt"
	"strconv"

	"github.com/egoist/mygo/ui"
)

// Color returns the color of series i.
func Color(t *ui.Theme, i int) ui.Color {
	l := float32(0.64)
	if t.Dark {
		l = 0.6
	}
	return ui.Oklch(l, 0.13, 250+float32(i)*67)
}

// View creates the chart of values, values[i][j] being series j in
// category i. Give it a size, or grow it in its parent.
func View(c *ui.Context, categories, series []string, values [][]float64) ui.Element {
	t := c.Theme()
	total := 0.0
	sums := make([]float64, len(categories))
	for i := range categories {
		for j := range series {
			sums[i] += value(values, i, j)
		}
		total += sums[i]
	}
	small := t.FontSize - 2
	return ui.Column(c).Gap(t.Space(3)).Children(func() {
		ui.Row(c).Wrap().Gap(t.Space(4)).Children(func() {
			for j, s := range series {
				ui.Row(c.Key(j)).Gap(t.Space(1.5)).AlignItems(ui.Center).Children(func() {
					ui.Box(c).Size(t.Space(2.5), t.Space(2.5)).Radius(2).Background(Color(t, j))
					ui.Text(c, s).FontSize(t.FontSize - 1)
				})
			}
		})
		_, lineH := c.MeasureText(0, ui.Span{Text: "M", Size: small})
		bottom := 2*lineH + t.Space(3)
		box := ui.Box(c).Grow(1).MinWidth(t.Space(80)).MinHeight(t.Space(60)).Role(ui.RoleImage).Label("Mosaic chart")
		b := box.LayoutBounds()
		gap := float32(2)
		plot := ui.Rect{W: b.W, H: max(1, b.H-bottom)}
		room := plot.W - gap*float32(max(len(categories)-1, 0))
		cols := make([]ui.Rect, len(categories))
		x := float32(0)
		for i := range categories {
			w := float32(0)
			if total > 0 {
				w = room * float32(sums[i]/total)
			}
			cols[i] = ui.Rect{X: x, Y: 0, W: w, H: plot.H}
			x += w + gap
		}
		block := func(i, j int) ui.Rect {
			col := cols[i]
			y := float32(0)
			for k := range j {
				y += share(value(values, i, k), sums[i]) * col.H
			}
			return ui.Rect{X: col.X, Y: y, W: col.W, H: share(value(values, i, j), sums[i]) * col.H}
		}
		hi, hj := -1, -1
		if px, py, over := box.PointerPosition(); over {
			for i, col := range cols {
				if px < col.X || px >= col.X+col.W {
					continue
				}
				for j := range series {
					if r := block(i, j); py >= r.Y && py < r.Y+r.H {
						hi, hj = i, j
					}
				}
			}
		}
		box.Draw(func(p *ui.Painter, r ui.Rect) {
			if r.W != b.W || r.H != b.H {
				c.Invalidate()
			}
			for i := range categories {
				for j := range series {
					br := block(i, j)
					if br.H <= 0 || br.W <= 0 {
						continue
					}
					br = ui.Rect{X: r.X + br.X, Y: r.Y + br.Y, W: br.W, H: br.H - gap}
					col, ink, faint := Color(t, j), ui.RGB(255, 255, 255), ui.RGBA(255, 255, 255, 0.85)
					if hj >= 0 && j != hj {
						col, ink, faint = col.Mix(t.Background, 0.55), t.Text.Alpha(0.6), t.Text.Alpha(0.5)
					}
					p.Fill(br, col, 2)
					if i == hi && j == hj {
						p.Stroke(br, t.Text, 2, 2)
					}
					label := series[j]
					pct := percent(share(value(values, i, j), sums[i]))
					nw, nh := p.MeasureText(0, ui.Span{Text: label, Size: small, Weight: 600})
					pw, _ := p.MeasureText(0, ui.Span{Text: pct, Size: small})
					pad := t.Space(1.5)
					switch {
					case nw+2*pad <= br.W && 2*nh+2*pad <= br.H:
						p.RichText(br.X+pad, br.Y+pad, 0, ui.Span{Text: label, Size: small, Weight: 600, Color: ink})
						p.RichText(br.X+pad, br.Y+pad+nh, 0, ui.Span{Text: pct, Size: small, Color: faint, Features: "tnum"})
					case pw+2*pad <= br.W && nh+2*pad <= br.H:
						p.RichText(br.X+pad, br.Y+pad, 0, ui.Span{Text: pct, Size: small, Color: faint, Features: "tnum"})
					}
				}
				cr := cols[i]
				if cr.W <= 0 {
					continue
				}
				name := ui.Span{Text: categories[i], Size: small, Weight: 600, Color: t.Text}
				if i == hi {
					name.Color = t.Accent
				}
				p.Clip(ui.Rect{X: r.X + cr.X, Y: r.Y + plot.H, W: cr.W, H: bottom}, 0, func() {
					p.RichText(r.X+cr.X+1, r.Y+plot.H+t.Space(1.5), 0, name)
					p.RichText(r.X+cr.X+1, r.Y+plot.H+t.Space(1.5)+lineH, 0,
						ui.Span{Text: percent(share(sums[i], total)), Size: small, Color: t.TextMuted, Features: "tnum"})
				})
			}
			if hi >= 0 {
				v := value(values, hi, hj)
				br := block(hi, hj)
				card(p, t, r, r.X+br.X+br.W/2, r.Y+br.Y+br.H/2, []string{
					series[hj] + " · " + categories[hi],
					number(v),
					percent(share(v, sums[hi])) + " of " + categories[hi],
					percent(share(v, total)) + " of all",
				}, Color(t, hj))
			}
		})
	})
}

func value(values [][]float64, i, j int) float64 {
	if i < len(values) && j < len(values[i]) {
		return max(values[i][j], 0)
	}
	return 0
}

func share(v, of float64) float32 {
	if of <= 0 {
		return 0
	}
	return float32(v / of)
}

func percent(f float32) string {
	if f < 0.1 {
		return fmt.Sprintf("%.1f%%", f*100)
	}
	return fmt.Sprintf("%.0f%%", f*100)
}

func number(v float64) string { return strconv.FormatFloat(v, 'f', -1, 64) }

// card paints a card of lines beside x, y, the first by a swatch of col.
func card(p *ui.Painter, t *ui.Theme, r ui.Rect, x, y float32, lines []string, col ui.Color) {
	small := t.FontSize - 2
	pad, sw := t.Space(2.5), t.Space(2.5)
	spans := make([]ui.Span, len(lines))
	w, h := float32(0), float32(0)
	for i, l := range lines {
		spans[i] = ui.Span{Text: l, Size: small, Color: t.TextMuted, Features: "tnum"}
		switch i {
		case 0:
			spans[i].Weight, spans[i].Color = 600, t.Text
		case 1:
			spans[i].Size, spans[i].Weight, spans[i].Color = t.FontSize, 600, t.Text
		}
		lw, lh := p.MeasureText(0, spans[i])
		if i == 0 {
			lw += sw + t.Space(1.5)
		}
		w, h = max(w, lw), h+lh
	}
	w, h = w+2*pad, h+2*pad
	cx, cy := x+t.Space(4), y-h/2
	if cx+w > r.X+r.W {
		cx = x - t.Space(4) - w
	}
	cx, cy = max(r.X, cx), max(r.Y, min(cy, r.Y+r.H-h))
	box := ui.Rect{X: cx, Y: cy, W: w, H: h}
	p.Shadow(box, t.Radius+2, 0, 2, 10, 0, ui.RGBA(0, 0, 0, 0.2))
	p.Fill(box, t.Background, t.Radius+2)
	p.Stroke(box, t.Border, t.Radius+2, 1)
	ty := cy + pad
	for i, sp := range spans {
		tx := cx + pad
		_, lh := p.MeasureText(0, sp)
		if i == 0 {
			p.Fill(ui.Rect{X: tx, Y: ty + (lh-sw)/2, W: sw, H: sw}, col, 2)
			tx += sw + t.Space(1.5)
		}
		p.RichText(tx, ty, 0, sp)
		ty += lh
	}
}
