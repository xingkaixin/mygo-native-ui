// Package waffle shows the shares of a whole as a waffle chart: a grid
// of a hundred squares, each a percent, colored by the part it belongs to
// in order from the top left, row by row, with a legend of the parts and
// their shares beside it. Shares are rounded so that the squares add up to
// a hundred, the largest remainders rounding up. The pointer on a part, in
// the grid or the legend, lights its squares and dims the others.
//
//	waffle.View(c, []waffle.Part{
//		{Name: "Renewables", Value: 31.4},
//		{Name: "Gas", Value: 22.9},
//		...
//	})
package waffle

import (
	"slices"
	"strconv"

	"github.com/egoist/mygo/ui"
)

// Part is a part of the whole: its name, its value, and its color; the
// zero Color is one of a set chosen by its place among the parts.
type Part struct {
	Name  string
	Value float64
	Color ui.Color
}

// View creates the waffle chart of parts, the grid a square as high as it
// is given and the legend to its right. Give it a height.
func View(c *ui.Context, parts []Part) ui.Element {
	t := c.Theme()
	cells := squares(parts)
	total := 0.0
	for _, p := range parts {
		total += max(p.Value, 0)
	}
	root := ui.Row(c).Gap(t.Space(6)).AlignItems(ui.Stretch)
	// hot is the part under the pointer in the last frame, or -1.
	hot := ui.Local(root, "waffle.hot", func() int { return -1 })
	lit := *hot
	*hot = -1
	h := root.LayoutBounds().H
	root.Children(func() {
		grid := ui.Box(c).Size(h, h).Shrink(0).Role(ui.RoleImage).Label("Waffle chart")
		gap := max(1, h/120)
		side := (h - 9*gap) / 10
		if x, y, ok := grid.PointerPosition(); ok && grid.Hovered() {
			col, row := int(x/(side+gap)), int(y/(side+gap))
			if col >= 0 && col < 10 && row >= 0 && row < 10 && cells[row*10+col] >= 0 {
				*hot = cells[row*10+col]
				p := parts[*hot]
				grid.Tooltip(p.Name + ": " + share(p.Value, total))
			}
		}
		grid.Draw(func(p *ui.Painter, r ui.Rect) {
			for k, part := range cells {
				cell := ui.Rect{X: r.X + float32(k%10)*(side+gap), Y: r.Y + float32(k/10)*(side+gap), W: side, H: side}
				col := t.Text.Alpha(0.08)
				if part >= 0 {
					col = colorOf(parts, part, t.Dark)
					if lit >= 0 && part != lit {
						col = col.Alpha(0.2)
					}
				}
				p.Fill(cell, col, side/6)
			}
		})
		ui.Column(c).Gap(t.Space(2)).Justify(ui.Center).Children(func() {
			for i, p := range parts {
				item := ui.Row(c.Key(i)).Gap(t.Space(2)).AlignItems(ui.Center)
				if item.Hovered() {
					*hot = i
				}
				ink := t.Text
				if lit >= 0 && i != lit {
					ink = t.TextMuted
				}
				item.Children(func() {
					ui.Box(c).Size(t.Space(3), t.Space(3)).Radius(t.Space(0.75)).Background(colorOf(parts, i, t.Dark))
					ui.Text(c, p.Name).TextColor(ink).SingleLine()
					ui.Text(c, share(p.Value, total)).TextColor(t.TextMuted).FontFeatures("tnum")
				})
			}
		})
	})
	if *hot != lit {
		c.Invalidate()
	}
	return root
}

// squares returns the part of each of the hundred squares, in order, or
// -1 for those left when the parts add to nothing.
func squares(parts []Part) []int {
	total := 0.0
	for _, p := range parts {
		total += max(p.Value, 0)
	}
	cells := make([]int, 0, 100)
	if total <= 0 {
		for range 100 {
			cells = append(cells, -1)
		}
		return cells
	}
	counts := make([]int, len(parts))
	rest := make([]int, len(parts))
	given := 0
	for i, p := range parts {
		exact := max(p.Value, 0) / total * 100
		counts[i] = int(exact)
		given += counts[i]
		rest[i] = i
	}
	slices.SortStableFunc(rest, func(a, b int) int {
		ra := max(parts[a].Value, 0)/total*100 - float64(counts[a])
		rb := max(parts[b].Value, 0)/total*100 - float64(counts[b])
		switch {
		case ra > rb:
			return -1
		case ra < rb:
			return 1
		}
		return 0
	})
	for k := 0; given < 100; k++ {
		counts[rest[k%len(rest)]]++
		given++
	}
	for i, n := range counts {
		for range n {
			cells = append(cells, i)
		}
	}
	return cells
}

// share writes v's share of total as a percent.
func share(v, total float64) string {
	if total <= 0 {
		return "0%"
	}
	pct := max(v, 0) / total * 100
	if pct < 10 {
		return strconv.FormatFloat(pct, 'f', 1, 64) + "%"
	}
	return strconv.FormatFloat(pct, 'f', 0, 64) + "%"
}

// colorOf returns the color of part i.
func colorOf(parts []Part, i int, dark bool) ui.Color {
	if c := parts[i].Color; c != (ui.Color{}) {
		return c
	}
	l := float32(0.64)
	if dark {
		l = 0.72
	}
	return ui.Oklch(l, 0.15, float32((250+i*137)%360))
}
