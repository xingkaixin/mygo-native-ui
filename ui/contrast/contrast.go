// Package contrast checks the contrast of text on a background against
// WCAG 2: the two colors in wells to edit and a button swapping them,
// large and normal text and a control drawn in them, their contrast
// ratio, and whether it passes AA and AAA for normal text, large text and
// the outlines of controls. When normal text fails, it offers the text
// color moved toward black or white just far enough to pass, which a
// click takes.
//
//	if contrast.View(c, &app.text, &app.background).Changed() {
//		app.save()
//	}
package contrast

import (
	"fmt"
	"math"

	"github.com/egoist/mygo/ui"
)

// Ratio returns the WCAG 2 contrast ratio of fg over bg, from 1 to 21.
// A translucent fg counts as composited over bg.
func Ratio(fg, bg ui.Color) float64 {
	bg.A = 255
	a, b := luminance(fg.Over(bg)), luminance(bg)
	return (max(a, b) + 0.05) / (min(a, b) + 0.05)
}

// luminance returns the relative luminance of c, as WCAG 2 defines it.
func luminance(c ui.Color) float64 {
	ch := func(v uint8) float64 {
		s := float64(v) / 255
		if s <= 0.04045 {
			return s / 12.92
		}
		return math.Pow((s+0.055)/1.055, 2.4)
	}
	return 0.2126*ch(c.R) + 0.7152*ch(c.G) + 0.0722*ch(c.B)
}

// level is a criterion of WCAG 2: the ratio it asks of what it covers.
type level struct {
	name string
	min  float64
}

var (
	aa       = level{"AA", 4.5}
	aaa      = level{"AAA", 7}
	aaLarge  = level{"AA", 3}
	aaaLarge = level{"AAA", 4.5}
)

// View creates the checker of text in *fg on *bg. Changed reports a new
// color.
func View(c *ui.Context, fg, bg *ui.Color) ui.Element {
	t := c.Theme()
	root := ui.Column(c).Gap(t.Space(4)).Label("Contrast")
	changed := false
	ratio := Ratio(*fg, *bg)
	root.Children(func() {
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			changed = well(c, "Text", fg) || changed
			if ui.Button(c, "⇄").Label("Swap colors").Clicked() {
				*fg, *bg, changed = *bg, *fg, true
			}
			changed = well(c, "Background", bg) || changed
		})

		ui.Column(c).Background(*bg).Radius(t.Radius).Border(1, t.Border).Padding(t.Space(5)).Gap(t.Space(2)).Children(func() {
			ui.Text(c, "Large text, 24 points").FontSize(24).Bold().TextColor(*fg)
			ui.Text(c, "Normal text reads like this, at the size of a paragraph.").FontSize(15).TextColor(*fg)
			ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Margin(t.Space(2), 0, 0, 0).Children(func() {
				ui.Box(c).Padding(t.Space(1.5), t.Space(4)).Radius(t.Radius).Border(1.5, *fg).Children(func() {
					ui.Text(c, "Button").TextColor(*fg)
				})
				ui.Box(c).Size(18, 18).Radius(9).Border(1.5, *fg)
			})
		})

		ui.Row(c).Gap(t.Space(6)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, fmt.Sprintf("%.2f:1", math.Floor(ratio*100)/100)).FontSize(32).Bold().FontFeatures("tnum").TextColor(verdict(t, ratio >= aaLarge.min))
			ui.Grid(c).ColumnTracks(ui.FitContent(), ui.FitContent(), ui.FitContent()).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
				for _, row := range []struct {
					name   string
					levels []level
				}{
					{"Normal text", []level{aa, aaa}},
					{"Large text", []level{aaLarge, aaaLarge}},
					{"Controls", []level{aaLarge}},
				} {
					ui.Text(c, row.name).TextColor(t.TextMuted)
					for i := range 2 {
						if i < len(row.levels) {
							badge(c, row.levels[i], ratio)
						} else {
							ui.Box(c)
						}
					}
				}
			})
		})

		var fixes []level
		for _, l := range []level{aa, aaa} {
			if ratio < l.min {
				fixes = append(fixes, l)
			}
		}
		if len(fixes) > 0 {
			ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
				ui.Text(c, "Text that passes").TextColor(t.TextMuted)
				for _, l := range fixes {
					f, ok := fix(*fg, *bg, l.min)
					if !ok {
						continue
					}
					b := ui.Row(c).Key(l.name).Gap(t.Space(1.5)).AlignItems(ui.Center).Padding(t.Space(1), t.Space(2)).
						Radius(t.Radius).Border(1, t.Border).Cursor(ui.CursorPointer).Label("Use " + hex(f))
					if b.Hovered() {
						b.Background(t.SurfaceHover)
					}
					b.Children(func() {
						ui.Box(c).Size(14, 14).Radius(3).Background(f).Border(1, t.Border)
						ui.Text(c, hex(f)).FontFeatures("tnum")
						ui.Text(c, l.name).FontSize(t.FontSize - 2).TextColor(t.TextMuted)
					})
					if b.Clicked() {
						*fg, changed = f, true
					}
				}
			})
		}
	})
	if changed {
		root.SetChanged()
	}
	return root
}

// well creates the well of *v, named name, with its hex, and reports a
// change.
func well(c *ui.Context, name string, v *ui.Color) bool {
	t := c.Theme()
	changed := false
	ui.Row(c).Key(name).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
		changed = ui.ColorWell(c, v).Label(name).Changed()
		ui.Column(c).Children(func() {
			ui.Text(c, name).FontSize(t.FontSize - 2).TextColor(t.TextMuted)
			ui.Text(c, hex(*v)).FontFeatures("tnum")
		})
	})
	return changed
}

// badge creates the pass or fail of l at ratio.
func badge(c *ui.Context, l level, ratio float64) {
	t := c.Theme()
	pass := ratio >= l.min
	mark := "✕"
	if pass {
		mark = "✓"
	}
	col := verdict(t, pass)
	ui.Row(c).Gap(t.Space(1)).Padding(t.Space(0.5), t.Space(2)).Radius(t.Radius).Background(col.Alpha(0.14)).
		Label(fmt.Sprintf("%s %v", l.name, pass)).Children(func() {
		ui.Text(c, mark).TextColor(col).Bold()
		ui.Text(c, l.name).TextColor(col).Bold().FontSize(t.FontSize - 1)
		ui.Text(c, fmt.Sprintf("%g", l.min)).TextColor(col.Alpha(0.75)).FontSize(t.FontSize - 2)
	})
}

// verdict is the color of a pass, or of a fail.
func verdict(t *ui.Theme, pass bool) ui.Color {
	if pass {
		return t.Success
	}
	return t.Danger
}

// fix returns fg mixed toward black or white, whichever bg contrasts
// with more, the least that brings the ratio to min, and whether any
// does.
func fix(fg, bg ui.Color, min float64) (ui.Color, bool) {
	fg = fg.Over(ui.Color{R: bg.R, G: bg.G, B: bg.B, A: 255})
	to := ui.RGB(0, 0, 0)
	if Ratio(ui.RGB(255, 255, 255), bg) > Ratio(to, bg) {
		to = ui.RGB(255, 255, 255)
	}
	if Ratio(to, bg) < min {
		return ui.Color{}, false
	}
	lo, hi := float32(0), float32(1)
	for range 16 {
		mid := (lo + hi) / 2
		if Ratio(fg.Mix(to, mid), bg) >= min {
			hi = mid
		} else {
			lo = mid
		}
	}
	return fg.Mix(to, hi), true
}

// hex returns c as #RRGGBB.
func hex(c ui.Color) string {
	return fmt.Sprintf("#%02X%02X%02X", c.R, c.G, c.B)
}
