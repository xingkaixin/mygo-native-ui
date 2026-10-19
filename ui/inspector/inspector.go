// Package inspector builds inspectors, as the panels of Xcode, Figma and
// Sketch: sections that open and close, rows of a label and the controls
// editing what is chosen, and compact number fields, two or three to a
// row, whose labels change them as the pointer drags them sideways.
//
//	inspector.Section(c, "Frame", &app.frameOpen, func() {
//		inspector.Row(c, "Position", func() {
//			inspector.Number(c, "X", &shape.X, -9999, 9999, 1)
//			inspector.Number(c, "Y", &shape.Y, -9999, 9999, 1)
//		})
//		inspector.Row(c, "Rotation", func() {
//			inspector.Number(c, "°", &shape.Angle, -360, 360, 0.5)
//		})
//		inspector.Row(c, "Fill", func() { ui.ColorWell(c, &shape.Fill) })
//	})
//
// Give the sections the width of the panel, one below the other.
package inspector

import (
	"math"
	"strconv"

	"github.com/egoist/mygo/ui"
)

// Section creates a section titled title, a disclosure that a click on the
// title opens and closes, as Return and Space do, showing what build
// builds while *open, with a line below it. Changed reports a click.
func Section(c *ui.Context, title string, open *bool, build func()) ui.Element {
	t := c.Theme()
	changed := false
	section := ui.Column(c)
	section.Children(func() {
		disclosure := ui.Collapsible(c, title, open, func() {
			ui.Column(c).Gap(t.Space(2)).Padding(t.Space(2.5), 0, t.Space(1), 0).FontWeight(400).Children(build)
		}).Bold().Padding(t.Space(3))
		changed = disclosure.Changed()
		ui.Divider(c)
	})
	if changed {
		section.SetChanged()
	}
	return section
}

// Row creates a row of the label label, muted, in a column as wide in
// every row, and of the controls build builds, which share the rest.
func Row(c *ui.Context, label string, build func()) ui.Element {
	t := c.Theme()
	return ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
		ui.Text(c, label).FontSize(t.FontSize - 1).TextColor(t.TextMuted).SingleLine().Width(t.Space(18)).Shrink(0)
		ui.Row(c).Gap(t.Space(1.5)).AlignItems(ui.Center).Grow(1).Basis(0).MinWidth(0).Children(build)
	})
}

// number is what a number field keeps: the text typed, and how far a
// drag has gone toward the next step.
type number struct {
	text  string
	drift float32
}

// Number creates a compact field of *value, from lo to hi in steps of
// step, labeled label within it at its start, as "X" or "W": dragging the
// label sideways changes the value a step for each DIP, ten with Shift
// held and a tenth with Option (Alt), and Up and Down step it while
// typing, by ten with Shift. What is typed applies as soon as it is a
// number in range, and shows rounded to the decimals of step once the
// field loses the keyboard focus. Changed reports a new value. Fields
// share the width of their row.
func Number(c *ui.Context, label string, value *float64, lo, hi, step float64) ui.Element {
	t := c.Theme()
	box := ui.Row(c).Height(t.Space(6.5)).Radius(t.Radius).Background(t.Surface).Border(1, t.Border).
		AlignItems(ui.Center).Grow(1).Basis(0).MinWidth(t.Space(14))
	st := ui.Local(box, "inspector.Number", func() number { return number{} })
	digits := decimals(step)
	changed := false
	set := func(v float64) {
		v = math.Round(max(lo, min(hi, v))/step) * step
		v, _ = strconv.ParseFloat(strconv.FormatFloat(v, 'f', digits, 64), 64)
		if v != *value {
			*value, changed = v, true
		}
	}
	focused := false
	box.Children(func() {
		handle := ui.Text(c, label).FontSize(t.FontSize-1).TextColor(t.TextMuted).NoWrap().
			Padding(0, t.Space(1.5), 0, t.Space(2)).Cursor(ui.CursorResizeEW)
		if dx, _, ok := handle.Dragged(); ok && dx != 0 {
			rate := float32(1)
			switch m := c.Modifiers(); {
			case m&ui.Shift != 0:
				rate = 10
			case m&ui.Alt != 0:
				rate = 0.1
			}
			// Steps go whole: what a slow drag moves less than one stays
			// for the next.
			st.drift += dx * rate
			whole := float32(math.Trunc(float64(st.drift)))
			st.drift -= whole
			set(*value + float64(whole)*step)
		}
		in := ui.TextInputBase(c, &st.text).Grow(1).MinWidth(0).Padding(0, t.Space(2), 0, 0).
			FontFeatures("tnum").Label(label)
		focused = in.Focused()
		if !focused {
			st.text = strconv.FormatFloat(*value, 'f', digits, 64)
			return
		}
		if in.Changed() {
			if v, err := strconv.ParseFloat(st.text, 64); err == nil && v >= lo && v <= hi {
				*value, changed = v, true
			}
		}
		for _, k := range []struct {
			key  ui.Key
			sign float64
		}{{ui.KeyUp, 1}, {ui.KeyDown, -1}} {
			for mods, n := range map[ui.Modifiers]float64{0: 1, ui.Shift: 10} {
				if in.Shortcut(mods, k.key) {
					set(*value + k.sign*n*step)
					st.text = strconv.FormatFloat(*value, 'f', digits, 64)
				}
			}
		}
	})
	if focused {
		box.Border(1, t.Accent)
	}
	if changed {
		box.SetChanged()
	}
	return box
}

// decimals returns the digits after the point that step has, at most 6.
func decimals(step float64) int {
	s := strconv.FormatFloat(step, 'f', -1, 64)
	for i := range s {
		if s[i] == '.' {
			return min(6, len(s)-i-1)
		}
	}
	return 0
}
