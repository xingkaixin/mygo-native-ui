// Package bullet shows measures against their targets as bullet graphs,
// as Stephen Few designed them for dashboards: a row for each measure,
// its name and unit on the left, and on the right bands of gray from dark
// to light for its ranges, as poor, fair and good, a bar of its value
// along them and a mark across them at its target, over an axis of round
// ticks. A value short of its target shows in the warning color. The
// pointer on a row shows its value, target and range.
//
//	bullet.View(c, []bullet.Bullet{
//		{Label: "Revenue", Unit: "$k", Value: 270, Target: 250, Ranges: []float64{150, 225, 300}},
//		{Label: "Satisfaction", Unit: "of 5", Value: 3.6, Target: 4.2, Ranges: []float64{2.5, 3.75, 5}},
//	})
package bullet

import (
	"strconv"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/chart"
)

// Bullet is a measure: its name and unit, its value, its target, and the
// ends of its ranges from the worst, from 0, the last the axis's end
// unless the value or the target go further. RangeNames, when given, name
// the ranges as the pointer reads them out. A Target of 0 shows none.
type Bullet struct {
	Label, Unit string
	Value       float64
	Target      float64
	Ranges      []float64
	RangeNames  []string
	// LowerIsBetter reverses the ranges, the darkest at the end, and warns
	// of a value above its target, as for costs and defects.
	LowerIsBetter bool
}

// View creates the bullet graphs of bullets, one under another. Give it a
// width, or stretch it in its parent.
func View(c *ui.Context, bullets []Bullet) ui.Element {
	t := c.Theme()
	label := t.Space(36)
	return ui.Column(c).Gap(t.Space(4)).Children(func() {
		for i, b := range bullets {
			row(c.Key(i), b, label)
		}
	})
}

// row creates the graph of b, its name label wide.
func row(c *ui.Context, b Bullet, label float32) {
	t := c.Theme()
	hi := max(b.Value, b.Target)
	for _, r := range b.Ranges {
		hi = max(hi, r)
	}
	scale := chart.NewLinear(0, hi).Nice(5)
	ticks := scale.Ticks(5)
	ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Start).Children(func() {
		ui.Column(c).Width(label).AlignItems(ui.End).Children(func() {
			ui.Text(c, b.Label).Bold().SingleLine()
			if b.Unit != "" {
				ui.Text(c, b.Unit).FontSize(t.FontSize - 2).TextColor(t.TextMuted).SingleLine()
			}
		})
		bar := t.Space(7)
		small := t.FontSize - 3
		box := ui.Box(c).Grow(1).Height(bar+small+t.Space(2)).Role(ui.RoleMeter).Label(b.Label).
			Range(0, hi, b.Value).Description(describe(b))
		if box.Hovered() {
			box.Tooltip(describe(b))
		}
		short := b.Target > 0 && (b.Value < b.Target) != b.LowerIsBetter && b.Value != b.Target
		box.Draw(func(p *ui.Painter, r ui.Rect) {
			x := func(v float64) float32 { return r.X + scale.At(v)*r.W }
			n := len(b.Ranges)
			from := 0.0
			for k, end := range b.Ranges {
				shade := k
				if b.LowerIsBetter {
					shade = n - 1 - k
				}
				// The darkest band is the worst range, the lightest the best.
				a := float32(0.32) - float32(shade)*0.22/float32(max(1, n-1))
				p.Fill(ui.Rect{X: x(from), Y: r.Y, W: x(end) - x(from), H: bar}, t.Text.Alpha(a), 0)
				from = end
			}
			col := t.Text
			if short {
				col = t.Warning
			}
			p.Fill(ui.Rect{X: r.X, Y: r.Y + bar/3, W: x(b.Value) - r.X, H: bar / 3}, col, 0)
			if b.Target > 0 {
				tx := x(b.Target)
				p.Fill(ui.Rect{X: tx - 1.5, Y: r.Y + bar*0.12, W: 3, H: bar * 0.76}, t.Text, 0)
			}
			for _, tk := range ticks {
				tx := r.X + tk.At*r.W
				p.Line(tx, r.Y+bar, tx, r.Y+bar+t.Space(1), 1, t.TextMuted)
				w, _ := p.MeasureText(0, ui.Span{Text: tk.Label, Size: small})
				p.Text(min(max(tx-w/2, r.X), r.X+r.W-w), r.Y+bar+t.Space(1.5), tk.Label, small, t.TextMuted)
			}
		})
	})
}

// describe reads out b's value, target and range.
func describe(b Bullet) string {
	s := number(b.Value)
	if b.Unit != "" {
		s += " " + b.Unit
	}
	if b.Target > 0 {
		s += ", target " + number(b.Target)
	}
	from := 0.0
	for k, end := range b.Ranges {
		if b.Value <= end || k == len(b.Ranges)-1 {
			name := number(from) + "–" + number(end)
			if k < len(b.RangeNames) {
				name = b.RangeNames[k] + " (" + name + ")"
			}
			s += ", in " + name
			break
		}
		from = end
	}
	return s
}

// number writes v as short as it reads.
func number(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}
