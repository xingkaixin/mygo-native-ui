// Package horizon shows many series over the same time in little height,
// as horizon charts in monitoring tools such as Cubism do: a short row a
// series, each an area chart cut into bands that fold over one another,
// each band deeper than the one below it, blue above zero and red below
// it, folded up the same way. The top band fills its row at a value the
// app gives, as 100 for percentages, so that rows compare; or, without
// one, at the largest value of the row's series, above or below zero, so
// each row reads on its own scale. Each row's name is at its start and
// its value at its end: the last one, or the one under the pointer, which
// a line across every row marks.
//
//	horizon.View(c, app.start, time.Minute, app.hosts, 3, 100)
package horizon

import (
	"math"
	"strconv"
	"time"

	"github.com/egoist/mygo/ui"
)

// Series is a series' name and its values, one a step.
type Series struct {
	Name   string
	Values []float64
}

// View creates the rows of series, whose values are step apart from
// start, each folded into bands, 3 when 0, the top one full at extent, or
// at the largest value of the row when 0.
func View(c *ui.Context, start time.Time, step time.Duration, series []Series, bands int, extent float64) ui.Element {
	t := c.Theme()
	if bands <= 0 {
		bands = 3
	}
	n := 0
	for _, s := range series {
		n = max(n, len(s.Values))
	}
	rowH := t.Space(8)
	small := t.FontSize - 2
	axisH := small*1.4 + t.Space(1.5)
	box := ui.Box(c).AlignSelf(ui.Stretch).Height(float32(len(series))*(rowH+1) + axisH).Role(ui.RoleImage).
		Label(strconv.Itoa(len(series)) + " series")
	hot := -1
	if x, _, ok := box.PointerPosition(); ok && box.Hovered() && n > 1 {
		w := box.LayoutBounds().W
		hot = max(0, min(n-1, int(math.Round(float64(x/w*float32(n-1))))))
	}
	dark := t.Dark
	box.Draw(func(p *ui.Painter, r ui.Rect) {
		if n == 0 {
			return
		}
		dx := r.W / float32(max(n-1, 1))
		for k, s := range series {
			row := ui.Rect{X: r.X, Y: r.Y + float32(k)*(rowH+1), W: r.W, H: rowH}
			p.Fill(row, t.Text.Alpha(0.03), 0)
			peak, high, low := extent, 0.0, 0.0
			for _, v := range s.Values {
				if !math.IsNaN(v) {
					high, low = max(high, v), min(low, v)
				}
			}
			if peak <= 0 {
				peak = max(high, -low)
			}
			if peak > 0 {
				for b := range bands {
					for _, neg := range []bool{false, true} {
						// A band no value reaches draws nothing, not even
						// a sliver along the bottom.
						reach := high
						if neg {
							reach = -low
						}
						if reach <= peak*float64(b)/float64(bands) {
							continue
						}
						var path ui.Path
						path.MoveTo(row.X, row.Y+row.H)
						for i, v := range s.Values {
							if math.IsNaN(v) || (v < 0) != neg {
								v = 0
							}
							f := float32(math.Abs(v)/peak)*float32(bands) - float32(b)
							f = max(0, min(f, 1))
							path.LineTo(row.X+float32(i)*dx, row.Y+row.H-f*row.H)
						}
						path.LineTo(row.X+float32(len(s.Values)-1)*dx, row.Y+row.H).Close()
						hue := float32(250)
						if neg {
							hue = 25
						}
						p.FillPath(&path, shade(hue, b, bands, dark))
					}
				}
			}
			i := len(s.Values) - 1
			if hot >= 0 {
				i = min(hot, len(s.Values)-1)
			}
			label(p, t, row, s.Name, false, small)
			if i >= 0 && !math.IsNaN(s.Values[i]) {
				label(p, t, row, number(s.Values[i]), true, small)
			}
		}
		axisY := r.Y + float32(len(series))*(rowH+1)
		ticks(p, t, ui.Rect{X: r.X, Y: axisY, W: r.W, H: axisH}, start, step, n, small)
		if hot >= 0 {
			x := r.X + float32(hot)*dx
			p.Line(x, r.Y, x, axisY, 1, t.Text.Alpha(0.7))
			tm := start.Add(time.Duration(hot) * step).Format("15:04:05")
			w, _ := p.MeasureText(0, ui.Span{Text: tm, Size: small})
			lx := max(r.X, min(x-w/2-4, r.X+r.W-w-8))
			p.Fill(ui.Rect{X: lx, Y: axisY + 1, W: w + 8, H: axisH - 2}, t.Text, 3)
			p.Text(lx+4, axisY+(axisH-small*1.2)/2, tm, small, t.Background)
		}
	})
	return box
}

// label writes text over the start of row, or its end, on a backing that
// keeps it legible over the bands.
func label(p *ui.Painter, t *ui.Theme, row ui.Rect, text string, end bool, size float32) {
	w, h := p.MeasureText(0, ui.Span{Text: text, Size: size, Weight: 500, Features: "tnum"})
	x := row.X + 4
	if end {
		x = row.X + row.W - w - 8
	}
	y := row.Y + (row.H-h)/2
	p.Fill(ui.Rect{X: x - 2, Y: y - 1, W: w + 4, H: h + 2}, t.Background.Alpha(0.75), 3)
	p.RichText(x, y, 0, ui.Span{Text: text, Size: size, Weight: 500, Features: "tnum", Color: t.Text})
}

// shade returns the color of band b of bands, of hue, deeper for higher.
func shade(hue float32, b, bands int, dark bool) ui.Color {
	f := float32(b+1) / float32(bands)
	if dark {
		return ui.Oklch(0.3+0.4*f, 0.05+0.11*f, hue)
	}
	return ui.Oklch(0.9-0.42*f, 0.04+0.13*f, hue)
}

// ticks draws the times along axis, as far apart as fits labels.
func ticks(p *ui.Painter, t *ui.Theme, axis ui.Rect, start time.Time, step time.Duration, n int, size float32) {
	total := time.Duration(max(n-1, 1)) * step
	every := time.Duration(0)
	for _, d := range []time.Duration{
		time.Second, 5 * time.Second, 15 * time.Second, 30 * time.Second,
		time.Minute, 5 * time.Minute, 15 * time.Minute, 30 * time.Minute,
		time.Hour, 3 * time.Hour, 6 * time.Hour, 12 * time.Hour, 24 * time.Hour,
	} {
		every = d
		if axis.W*float32(d)/float32(total) >= t.Space(22) {
			break
		}
	}
	layout := "15:04"
	if every < time.Minute {
		layout = "15:04:05"
	}
	first := start.Truncate(every)
	if first.Before(start) {
		first = first.Add(every)
	}
	for tm := first; !tm.After(start.Add(total)); tm = tm.Add(every) {
		x := axis.X + axis.W*float32(tm.Sub(start))/float32(total)
		label := tm.Format(layout)
		w, _ := p.MeasureText(0, ui.Span{Text: label, Size: size})
		if x-w/2 < axis.X || x+w/2 > axis.X+axis.W {
			continue
		}
		p.Line(x, axis.Y, x, axis.Y+3, 1, t.Border)
		p.Text(x-w/2, axis.Y+t.Space(1), label, size, t.TextMuted)
	}
}

// number writes v short, with up to three significant digits: 0.042,
// 12.5, 1,204, 3.4M.
func number(v float64) string {
	a := math.Abs(v)
	switch {
	case a >= 1e6:
		return strconv.FormatFloat(v/1e6, 'f', 1, 64) + "M"
	case a >= 1e4:
		return strconv.FormatFloat(v/1e3, 'f', 1, 64) + "k"
	case a >= 100:
		return strconv.FormatFloat(math.Round(v), 'f', 0, 64)
	case a >= 1:
		return strconv.FormatFloat(v, 'f', 1, 64)
	}
	return strconv.FormatFloat(v, 'g', 2, 64)
}
