// Package histslider is a range slider over the values it filters, as the
// price filters of booking sites show one: bars of how many values fall
// in each stretch of the range, those within the range chosen in the
// accent color and the rest faded, over a range slider of two knobs,
// and the two ends below it as the app writes them. Values past the
// range count in its first or last bar, as "$500+" does.
//
//	if histslider.View(c, &app.low, &app.high, prices, histslider.Range{
//		Min: 0, Max: 500, Step: 10,
//		Format: func(v float64) string { return fmt.Sprintf("$%.0f", v) },
//	}).Changed() {
//		app.filter()
//	}
package histslider

import (
	"math"
	"strconv"

	"github.com/egoist/mygo/ui"
)

// Range is what the slider chooses within: its ends, the step its values
// snap to (0 for none), how many bars it shows (32 when 0), and how it
// writes a value (nil writes it with up to two decimals).
type Range struct {
	Min, Max float64
	Step     float64
	Bins     int
	Format   func(float64) string
}

func (r Range) text(v float64) string {
	if r.Format != nil {
		return r.Format(v)
	}
	return strconv.FormatFloat(math.Round(v*100)/100, 'f', -1, 64)
}

// View creates the slider choosing *low and *high within r, over the
// bars of values. Changed reports a knob moved.
func View(c *ui.Context, low, high *float64, values []float64, r Range) ui.Element {
	t := c.Theme()
	n := r.Bins
	if n <= 0 {
		n = 32
	}
	counts := make([]int, n)
	top := 1
	if r.Max > r.Min {
		for _, v := range values {
			i := int((v - r.Min) / (r.Max - r.Min) * float64(n))
			i = max(0, min(n-1, i))
			counts[i]++
			top = max(top, counts[i])
		}
	}
	// The range slider's knobs travel within half a knob of its ends,
	// where the bars must start and stop to stand over the values they
	// count.
	inset := t.Space(2.5)
	root := ui.Column(c).Gap(t.Space(1))
	root.Children(func() {
		ui.Box(c).Height(t.Space(16)).FillWidth().Draw(func(p *ui.Painter, rect ui.Rect) {
			w := (rect.W - 2*inset) / float32(n)
			gap := min(2, w/4)
			for i, k := range counts {
				if k == 0 {
					continue
				}
				h := max(2, rect.H*float32(k)/float32(top))
				mid := r.Min + (float64(i)+0.5)/float64(n)*(r.Max-r.Min)
				color := t.Text.Alpha(0.15)
				if mid >= *low && mid <= *high {
					color = t.Accent
				}
				bar := ui.Rect{X: rect.X + inset + float32(i)*w + gap/2, Y: rect.Y + rect.H - h, W: w - gap, H: h}
				p.Fill(bar, color, min(2, bar.W/2))
			}
		})
		if ui.RangeSlider(c, low, high, r.Min, r.Max, r.Step).FillWidth().Label("Range").Changed() {
			root.SetChanged()
		}
		ui.Row(c).Padding(0, inset).Children(func() {
			ui.Text(c, r.text(*low)).FontFeatures("tnum").TextColor(t.TextMuted).Grow(1)
			ui.Text(c, r.text(*high)).FontFeatures("tnum").TextColor(t.TextMuted)
		})
	})
	return root
}
