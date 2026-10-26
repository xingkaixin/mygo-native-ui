// Package boxmodel edits the box model of an element, as the Computed
// panels of browsers' developer tools show it: the content's width and
// height in the middle, inside its padding, its border and its margin,
// each a box of its color with its name in its corner and its four sides'
// sizes on them. Dragging a size sideways changes it, a step for each
// DIP, ten with Shift held, and with Option (Alt) held changes the
// opposite side with it; a double click types it in, Enter applying it
// and Escape leaving it. The pointer on a box lights it. A side of 0
// shows a dash.
//
//	if boxmodel.View(c, &app.box).Changed() {
//		app.restyle()
//	}
package boxmodel

import (
	"math"
	"strconv"

	"github.com/egoist/mygo/ui"
)

// Edges are the sizes of a box's four sides, in pixels.
type Edges struct {
	Top, Right, Bottom, Left float64
}

// Box is an element's box: its content's size, and the padding, border
// and margin around it. Margins may be negative; the rest may not.
type Box struct {
	Width, Height           float64
	Padding, Border, Margin Edges
}

// layer is a box of the model, from the outside in.
type layer struct {
	name  string
	hue   float32
	edges *Edges
	lo    float64
}

// field is what a size keeps: whether it is being typed and has had the
// keyboard focus since, the text typed, and how far a drag has gone toward
// the next step.
type field struct {
	editing, focused bool
	text             string
	drift            float32
}

// View creates the editor of *b. Changed reports a change of a size.
func View(c *ui.Context, b *Box) ui.Element {
	t := c.Theme()
	layers := []layer{
		{"margin", 55, &b.Margin, math.Inf(-1)},
		{"border", 85, &b.Border, 0},
		{"padding", 130, &b.Padding, 0},
	}
	root := ui.Column(c).Label("Box model")
	// hot is the box under the pointer in the last frame, the content
	// being len(layers), or -1.
	hot := ui.Local(root, "boxmodel.hot", func() int { return -1 })
	lit := *hot
	*hot = -1
	changed := false
	var nest func(i int)
	nest = func(i int) {
		if i == len(layers) {
			content(c, b, lit == i, hot, i, &changed)
			return
		}
		l := layers[i]
		col := tint(l.hue, t.Dark, lit == i)
		box := ui.Column(c).Background(col).Border(1, t.Text.Alpha(0.35)).AlignItems(ui.Stretch).Padding(0, t.Space(1))
		if l.name == "margin" {
			box.BorderStyle(ui.BorderDashed)
		}
		if box.Hovered() {
			*hot = i
		}
		box.Children(func() {
			ui.Row(c).Children(func() {
				ui.Text(c, l.name).FontSize(t.FontSize-3).TextColor(t.Text.Alpha(0.7)).Width(0).NoWrap().Margin(t.Space(0.5), 0, 0, t.Space(0.5))
				ui.Row(c).Grow(1).Justify(ui.Center).Children(func() {
					changed = size(c.Key("top"), &l.edges.Top, &l.edges.Bottom, l.lo) || changed
				})
			})
			ui.Row(c).AlignItems(ui.Center).Children(func() {
				changed = size(c.Key("left"), &l.edges.Left, &l.edges.Right, l.lo) || changed
				ui.Column(c).Grow(1).Margin(0, t.Space(1)).Children(func() { nest(i + 1) })
				changed = size(c.Key("right"), &l.edges.Right, &l.edges.Left, l.lo) || changed
			})
			ui.Row(c).Justify(ui.Center).Children(func() {
				changed = size(c.Key("bottom"), &l.edges.Bottom, &l.edges.Top, l.lo) || changed
			})
		})
	}
	root.Children(func() { nest(0) })
	if *hot != lit {
		c.Invalidate()
	}
	if changed {
		root.SetChanged()
	}
	return root
}

// content creates the content's box, lit while lit, of index i.
func content(c *ui.Context, b *Box, lit bool, hot *int, i int, changed *bool) {
	t := c.Theme()
	box := ui.Row(c).Background(tint(215, t.Dark, lit)).Border(1, t.Text.Alpha(0.35)).
		Justify(ui.Center).AlignItems(ui.Center).Padding(t.Space(2.5), t.Space(3)).Gap(t.Space(1)).MinWidth(t.Space(30))
	if box.Hovered() {
		*hot = i
	}
	box.Children(func() {
		*changed = size(c.Key("width"), &b.Width, nil, 0) || *changed
		ui.Text(c, "×").TextColor(t.Text.Alpha(0.6))
		*changed = size(c.Key("height"), &b.Height, nil, 0) || *changed
	})
}

// tint returns the color of a box of hue, brighter while lit.
func tint(hue float32, dark, lit bool) ui.Color {
	if dark {
		if lit {
			return ui.Oklch(0.48, 0.09, hue)
		}
		return ui.Oklch(0.38, 0.07, hue)
	}
	if lit {
		return ui.Oklch(0.82, 0.11, hue)
	}
	return ui.Oklch(0.9, 0.07, hue)
}

// size creates the field of *v, no less than lo, which a drag with Option
// held moves *opposite with, and reports a change.
func size(c *ui.Context, v, opposite *float64, lo float64) bool {
	t := c.Theme()
	f := ui.Row(c).Height(t.Space(5.5)).MinWidth(t.Space(7)).Justify(ui.Center).AlignItems(ui.Center).Radius(t.Space(1))
	st := ui.Local(f, "boxmodel.field", func() field { return field{} })
	changed := false
	set := func(n float64) {
		n = math.Max(lo, math.Round(n))
		if n != *v {
			if opposite != nil && c.Modifiers()&ui.Alt != 0 {
				*opposite = math.Max(lo, *opposite+n-*v)
			}
			*v, changed = n, true
		}
	}
	f.Children(func() {
		if st.editing {
			in := ui.TextInputBase(c, &st.text).Width(t.Space(10)).TextAlign(ui.Center).FontSize(t.FontSize - 1).
				FontFeatures("tnum").Background(t.Background).Radius(t.Space(1)).AutoFocus().Label("Size")
			if in.Focused() {
				st.focused = true
			}
			switch {
			case in.Submitted():
				if n, err := strconv.ParseFloat(st.text, 64); err == nil {
					set(n)
				}
				st.editing = false
			case in.Shortcut(0, ui.KeyEscape), st.focused && !in.Focused():
				st.editing = false
			}
			return
		}
		text := strconv.FormatFloat(*v, 'f', -1, 64)
		if *v == 0 {
			text = "–"
		}
		label := ui.Text(c, text).FontSize(t.FontSize-1).FontFeatures("tnum").Padding(0, t.Space(1)).
			Cursor(ui.CursorResizeEW).Value(strconv.FormatFloat(*v, 'f', -1, 64))
		if label.Hovered() {
			f.Background(t.Text.Alpha(0.1))
		}
		if dx, _, ok := label.Dragged(); ok && dx != 0 {
			rate := float32(1)
			if c.Modifiers()&ui.Shift != 0 {
				rate = 10
			}
			st.drift += dx * rate
			whole := float32(math.Trunc(float64(st.drift)))
			st.drift -= whole
			set(*v + float64(whole))
		}
		if label.DoubleClicked() {
			st.editing, st.focused, st.text = true, false, strconv.FormatFloat(*v, 'f', -1, 64)
			c.Invalidate()
		}
	})
	if changed {
		f.SetChanged()
	}
	return changed
}
