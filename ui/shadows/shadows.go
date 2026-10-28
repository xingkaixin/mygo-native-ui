// Package shadows edits a stack of box shadows, as the effects panels of
// design tools and CSS shadow generators do: a card on a stage casting the
// shadows as edited, and beside it the layers, top first, each with its
// color, its offsets, blur and spread in short, a box to show or hide it
// and a button to remove it. The fields of the layer chosen set its X and
// Y, blur and spread, dragged by their labels or typed, its color, and
// whether it falls inside the card. Dragging across the stage moves the
// chosen layer's offset, as moving the light would. Below, the stack as
// CSS, with Copy.
//
//	shadows.View(c, &app.editor, &app.shadows)
//	css := shadows.CSS(app.shadows)
//
// Changed reports an edit.
package shadows

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/inspector"
)

// Shadow is a layer of a stack: its offset, blur and spread in DIPs, its
// color, whether it falls inside the box, and whether it is hidden.
type Shadow struct {
	X, Y, Blur, Spread float64
	Color              ui.Color
	Inset, Hidden      bool
}

// CSS writes the shadows not hidden as the value of CSS's box-shadow, the
// first on top, or "none".
func CSS(shadows []Shadow) string {
	var parts []string
	for _, s := range shadows {
		if s.Hidden {
			continue
		}
		px := func(v float64) string {
			if v == 0 {
				return "0"
			}
			return strconv.FormatFloat(v, 'f', -1, 64) + "px"
		}
		part := fmt.Sprintf("%s %s %s %s %s", px(s.X), px(s.Y), px(s.Blur), px(s.Spread), rgba(s.Color))
		if s.Inset {
			part = "inset " + part
		}
		parts = append(parts, part)
	}
	if len(parts) == 0 {
		return "none"
	}
	return strings.Join(parts, ",\n            ")
}

func rgba(c ui.Color) string {
	if c.A == 255 {
		return fmt.Sprintf("rgb(%d %d %d)", c.R, c.G, c.B)
	}
	return fmt.Sprintf("rgb(%d %d %d / %s)", c.R, c.G, c.B, strconv.FormatFloat(math.Round(float64(c.A)/255*100)/100, 'f', -1, 64))
}

// State is what an editor keeps: the layer chosen, and when the CSS was
// copied. Keep it in the app's state, one for each editor.
type State struct {
	chosen int
	copied time.Time
}

// View creates the editor of *shadows.
func View(c *ui.Context, s *State, shadows *[]Shadow) ui.Element {
	t := c.Theme()
	changed := false
	if s.chosen >= len(*shadows) {
		s.chosen = len(*shadows) - 1
	}
	s.chosen = max(0, s.chosen)
	root := ui.Column(c).Gap(t.Space(4))
	root.Children(func() {
		ui.Row(c).Gap(t.Space(4)).AlignItems(ui.Stretch).Children(func() {
			if s.stage(c, *shadows) {
				changed = true
			}
			ui.Column(c).Width(t.Space(76)).Gap(t.Space(3)).Children(func() {
				if s.layers(c, shadows) {
					changed = true
				}
				if s.chosen < len(*shadows) && s.fields(c, &(*shadows)[s.chosen]) {
					changed = true
				}
			})
		})
		css := "box-shadow: " + CSS(*shadows) + ";"
		ui.Row(c).Gap(t.Space(3)).Padding(t.Space(3)).Radius(t.Space(2)).Background(t.Surface).AlignItems(ui.Start).Children(func() {
			ui.Text(c, css).Font("monospace").FontSize(t.FontSize - 1).Selectable().Grow(1)
			label := "Copy"
			if left := 1200*time.Millisecond - c.Now().Sub(s.copied); left > 0 {
				label = "Copied"
				c.After(left)
			}
			if ui.Button(c, label).Clicked() {
				c.WriteClipboard(css)
				s.copied = c.Now()
			}
		})
	})
	if changed {
		root.SetChanged()
	}
	return root
}

// stage creates the card casting shadows, and reports whether a drag moved
// the chosen layer.
func (s *State) stage(c *ui.Context, shadows []Shadow) (changed bool) {
	t := c.Theme()
	stage := ui.Column(c).Grow(1).MinHeight(t.Space(80)).Radius(t.Space(3)).Background(t.Surface).Center().
		Cursor(ui.CursorMove).Clip().Label("Stage")
	if dx, dy, ok := stage.Dragged(); ok && s.chosen < len(shadows) && (dx != 0 || dy != 0) {
		sh := &shadows[s.chosen]
		sh.X, sh.Y = math.Round(sh.X+float64(dx)), math.Round(sh.Y+float64(dy))
		changed = true
	}
	stage.Children(func() {
		card := ui.Column(c).Size(t.Space(56), t.Space(36)).Radius(t.Space(4)).Background(t.Background).
			Padding(t.Space(5)).Gap(t.Space(2)).PassThrough()
		for _, sh := range shadows {
			if sh.Hidden {
				continue
			}
			x, y, blur, spread := float32(sh.X), float32(sh.Y), float32(sh.Blur), float32(sh.Spread)
			if sh.Inset {
				card.InsetShadow(x, y, blur, spread, sh.Color)
			} else {
				card.Shadow(x, y, blur, spread, sh.Color)
			}
		}
		card.Children(func() {
			ui.Box(c).Size(t.Space(8), t.Space(8)).Radius(t.Space(4)).Background(t.Accent.Alpha(0.2)).PassThrough()
			ui.Box(c).Height(t.Space(2)).WidthPercent(70).Radius(t.Space(1)).Background(t.Text.Alpha(0.15)).PassThrough()
			ui.Box(c).Height(t.Space(2)).WidthPercent(45).Radius(t.Space(1)).Background(t.Text.Alpha(0.1)).PassThrough()
		})
	})
	return changed
}

// layers creates the list of layers with Add, and reports whether one was
// added, removed, shown or hidden.
func (s *State) layers(c *ui.Context, shadows *[]Shadow) (changed bool) {
	t := c.Theme()
	ui.Row(c).AlignItems(ui.Center).Children(func() {
		ui.Text(c, "Shadows").Bold().Grow(1)
		if ui.Button(c, "Add").Clicked() {
			*shadows = append([]Shadow{{Y: 4, Blur: 12, Color: ui.RGBA(0, 0, 0, 0.15)}}, *shadows...)
			s.chosen, changed = 0, true
		}
	})
	remove := -1
	ui.Column(c).Gap(t.Space(1)).Children(func() {
		for i := range *shadows {
			sh := &(*shadows)[i]
			row := ui.ButtonBase(c.Key(i)).Gap(t.Space(2)).Padding(t.Space(1.5), t.Space(2)).Radius(t.Space(1.5)).
				AlignItems(ui.Center).Selected(i == s.chosen).Label("Layer " + strconv.Itoa(i+1))
			switch {
			case i == s.chosen:
				row.Background(t.Accent.Alpha(0.12)).TextColor(t.Text)
			case row.Hovered():
				row.Background(t.SurfaceHover)
			}
			if row.Clicked() {
				s.chosen = i
			}
			row.Children(func() {
				shown := !sh.Hidden
				if ui.Checkbox(c, &shown, "").Label("Show").Changed() {
					sh.Hidden, changed = !shown, true
				}
				// The color shows opaque: most shadows are a faint black.
				color := ui.RGB(sh.Color.R, sh.Color.G, sh.Color.B)
				ui.Box(c).Size(t.Space(4), t.Space(4)).Radius(t.Space(1)).Border(1, t.Border).Draw(func(p *ui.Painter, r ui.Rect) {
					p.Fill(r, color, t.Space(1))
				})
				summary := fmt.Sprintf("%g %g %g %g", sh.X, sh.Y, sh.Blur, sh.Spread)
				txt := ui.Text(c, summary).Font("monospace").FontSize(t.FontSize - 1).Grow(1)
				if sh.Hidden {
					txt.TextColor(t.TextMuted)
				}
				if sh.Inset {
					ui.Text(c, "Inset").FontSize(t.FontSize - 2).TextColor(t.TextMuted)
				}
				x := ui.ButtonBase(c).Size(t.Space(5), t.Space(5)).Radius(t.Space(1)).Center().Label("Remove")
				if x.Hovered() {
					x.Background(t.Text.Alpha(0.08))
				}
				if x.Clicked() {
					remove = i
				}
				x.Children(func() { ui.Text(c, "×").TextColor(t.TextMuted) })
			})
		}
	})
	if remove >= 0 {
		*shadows = append((*shadows)[:remove], (*shadows)[remove+1:]...)
		changed = true
	}
	return changed
}

// fields creates the fields of sh, and reports a change.
func (s *State) fields(c *ui.Context, sh *Shadow) (changed bool) {
	t := c.Theme()
	ui.Column(c).Gap(t.Space(2)).Padding(t.Space(3), 0, 0, 0).Children(func() {
		inspector.Row(c, "Offset", func() {
			changed = inspector.Number(c, "X", &sh.X, -200, 200, 1).Changed() || changed
			changed = inspector.Number(c, "Y", &sh.Y, -200, 200, 1).Changed() || changed
		})
		inspector.Row(c, "Blur", func() {
			changed = inspector.Number(c, "B", &sh.Blur, 0, 200, 1).Changed() || changed
		})
		inspector.Row(c, "Spread", func() {
			changed = inspector.Number(c, "S", &sh.Spread, -100, 100, 1).Changed() || changed
		})
		inspector.Row(c, "Color", func() {
			changed = ui.ColorWell(c, &sh.Color).Changed() || changed
			ui.Text(c, rgba(sh.Color)).Font("monospace").FontSize(t.FontSize - 2).TextColor(t.TextMuted)
		})
		inspector.Row(c, "Inset", func() {
			changed = ui.Switch(c, &sh.Inset).Label("Inset").Changed() || changed
		})
	})
	return changed
}
