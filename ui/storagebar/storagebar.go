// Package storagebar shows how a disk or a quota is used, as macOS's
// Storage settings show it: the name and how much of it is used above, a
// bar of parts in their colors, in order, with the space left gray at the
// end, and below it a legend of each part and its size. The pointer on a
// part, in the bar or the legend, lights it there and dims the others.
//
//	storagebar.View(c, "Macintosh HD", 494_384_795_648, []storagebar.Part{
//		{Name: "Applications", Size: 48_200_000_000},
//		{Name: "Documents", Size: 112_700_000_000},
//		...
//	})
package storagebar

import (
	"fmt"
	"strconv"

	"github.com/egoist/mygo/ui"
)

// Part is a part of what is used: its name, its size in bytes, and its
// color, one of a set chosen by its place among the parts when zero.
type Part struct {
	Name  string
	Size  int64
	Color ui.Color
}

// palette is the hues of parts without a color, in the order of the
// parts, as macOS colors its categories.
var palette = []float32{25, 75, 145, 250, 300, 195, 0, 120}

// colorOf returns the color of part i.
func colorOf(parts []Part, i int, dark bool) ui.Color {
	if c := parts[i].Color; c != (ui.Color{}) {
		return c
	}
	if dark {
		return ui.Oklch(0.7, 0.16, palette[i%len(palette)])
	}
	return ui.Oklch(0.66, 0.17, palette[i%len(palette)])
}

// View creates the bar of the parts used of capacity bytes of what is
// named.
func View(c *ui.Context, name string, capacity int64, parts []Part) ui.Element {
	t := c.Theme()
	used := int64(0)
	for _, p := range parts {
		used += max(p.Size, 0)
	}
	total := max(capacity, used, 1)
	el := ui.Column(c).Gap(t.Space(2.5))
	// hot is the part under the pointer in the last frame, or -1.
	hot := ui.Local(el, "storagebar.hot", func() int { return -1 })
	lit := *hot
	*hot = -1
	el.Children(func() {
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.End).Children(func() {
			ui.Text(c, name).Bold().Grow(1).Shrink(1).SingleLine()
			ui.Text(c, fmt.Sprintf("%s of %s used", Bytes(used), Bytes(capacity))).
				FontSize(t.FontSize - 1).TextColor(t.TextMuted).FontFeatures("tnum")
		})
		bar := ui.Box(c).AlignSelf(ui.Stretch).Height(t.Space(5)).Role(ui.RoleMeter).
			Label(name).Description(fmt.Sprintf("%s of %s used", Bytes(used), Bytes(capacity)))
		w := bar.LayoutBounds().W
		widths := make([]float32, len(parts))
		for i, p := range parts {
			if p.Size > 0 {
				widths[i] = max(2, w*float32(p.Size)/float32(total))
			}
		}
		if x, _, ok := bar.PointerPosition(); ok && bar.Hovered() {
			at := float32(0)
			for i, pw := range widths {
				if x >= at && x < at+pw {
					*hot = i
					bar.Tooltip(parts[i].Name + ": " + Bytes(parts[i].Size))
				}
				at += pw
			}
		}
		dark := t.Dark
		bar.Draw(func(p *ui.Painter, r ui.Rect) {
			radius := r.H / 2
			p.Fill(r, t.Text.Alpha(0.1), radius)
			p.Clip(r, radius, func() {
				x := r.X
				for i, pw := range widths {
					if pw == 0 {
						continue
					}
					col := colorOf(parts, i, dark)
					if lit >= 0 && i != lit {
						col = col.Alpha(0.35)
					}
					p.Fill(ui.Rect{X: x, Y: r.Y, W: pw, H: r.H}, col, 0)
					x += pw
					// A line of the background between parts keeps
					// neighbors of like colors apart.
					p.Fill(ui.Rect{X: x - 0.5, Y: r.Y, W: 1, H: r.H}, t.Background, 0)
				}
			})
		})
		ui.Row(c).Wrap().GapX(t.Space(5)).GapY(t.Space(1.5)).Children(func() {
			for i, p := range parts {
				item := ui.Row(c.Key(i)).Gap(t.Space(1.5)).AlignItems(ui.Center)
				if item.Hovered() {
					*hot = i
				}
				color := t.Text
				if lit >= 0 && i != lit {
					color = t.TextMuted
				}
				item.Children(func() {
					ui.Box(c).Size(t.Space(2.5), t.Space(2.5)).Radius(t.Space(1.25)).Background(colorOf(parts, i, t.Dark))
					ui.Text(c, p.Name).FontSize(t.FontSize - 1).TextColor(color)
					ui.Text(c, Bytes(p.Size)).FontSize(t.FontSize - 1).TextColor(t.TextMuted).FontFeatures("tnum")
				})
			}
			if free := capacity - used; free > 0 {
				ui.Row(c).Gap(t.Space(1.5)).AlignItems(ui.Center).Children(func() {
					ui.Box(c).Size(t.Space(2.5), t.Space(2.5)).Radius(t.Space(1.25)).Background(t.Text.Alpha(0.15))
					ui.Text(c, "Available").FontSize(t.FontSize - 1).TextColor(t.TextMuted)
					ui.Text(c, Bytes(free)).FontSize(t.FontSize - 1).TextColor(t.TextMuted).FontFeatures("tnum")
				})
			}
		})
	})
	if *hot != lit {
		c.Invalidate()
	}
	return el
}

// Bytes writes n bytes as macOS does, in powers of 1,000: "512 bytes",
// "4.2 MB", "312.48 GB".
func Bytes(n int64) string {
	if n < 1000 {
		return strconv.FormatInt(n, 10) + " bytes"
	}
	f := float64(n)
	for _, unit := range []string{"KB", "MB", "GB", "TB", "PB"} {
		f /= 1000
		if f < 1000 || unit == "PB" {
			switch {
			case f < 10:
				return strconv.FormatFloat(f, 'f', 2, 64) + " " + unit
			case f < 100:
				return strconv.FormatFloat(f, 'f', 1, 64) + " " + unit
			}
			return strconv.FormatFloat(f, 'f', 0, 64) + " " + unit
		}
	}
	return ""
}
