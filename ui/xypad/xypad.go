// Package xypad is a pad setting two values at once, as synthesizers and
// effects show one for a filter's cutoff and resonance: x runs from left
// to right across it and y from bottom to top, with a puck at the values
// and lines crossing there. A press puts the puck under the pointer and
// dragging moves it, ten times finer with Shift. The arrow keys step x
// and y while the pad has the focus, Shift for fine steps; a double click
// resets both. Each axis's label and value show under the pad. Changed
// reports a move.
//
//	xypad.View(c, &app.cutoff, &app.resonance,
//		xypad.Axis{Label: "Cutoff", Min: 20, Max: 20000, Default: 2000, Log: true},
//		xypad.Axis{Label: "Resonance", Min: 0, Max: 1, Default: 0.2},
//	)
package xypad

import (
	"math"
	"strconv"

	"github.com/egoist/mygo/ui"
)

// Axis is what one way across the pad sets: its range, the value a double
// click resets it to, and how it writes its value.
type Axis struct {
	Label    string
	Min, Max float64
	Default  float64
	// Log runs the axis over the range in ratios rather than steps, as for
	// a frequency or a time; Min must be above 0.
	Log bool
	// Format writes the value; nil writes it with up to two decimals.
	Format func(float64) string
}

// at returns how far across the axis v is, from 0 to 1.
func (a Axis) at(v float64) float64 {
	if a.Max == a.Min {
		return 0
	}
	f := (v - a.Min) / (a.Max - a.Min)
	if a.Log {
		f = math.Log(v/a.Min) / math.Log(a.Max/a.Min)
	}
	return max(0, min(1, f))
}

// value returns the value f of the way across the axis.
func (a Axis) value(f float64) float64 {
	f = max(0, min(1, f))
	if a.Log {
		return a.Min * math.Pow(a.Max/a.Min, f)
	}
	return a.Min + f*(a.Max-a.Min)
}

func (a Axis) text(v float64) string {
	if a.Format != nil {
		return a.Format(v)
	}
	return strconv.FormatFloat(math.Round(v*100)/100, 'f', -1, 64)
}

// View creates the pad setting *x across and *y up.
func View(c *ui.Context, x, y *float64, ax, ay Axis) ui.Element {
	t := c.Theme()
	root := ui.Column(c).Width(t.Space(64)).Gap(t.Space(2))
	root.Children(func() {
		pad := ui.Box(c).AspectRatio(1).Radius(t.Space(2)).Background(t.Surface).Border(1, t.Border).Focusable().
			Cursor(ui.CursorCrosshair).Role(ui.RoleGroup).Label(ax.Label + " and " + ay.Label).
			Value(ax.Label + " " + ax.text(*x) + ", " + ay.Label + " " + ay.text(*y))
		held := ui.Local(pad, "xypad.held", func() bool { return false })
		set := func(fx, fy float64) {
			nx, ny := ax.value(fx), ay.value(fy)
			if nx != *x || ny != *y {
				*x, *y = nx, ny
				root.SetChanged()
			}
		}
		fx, fy := ax.at(*x), ay.at(*y)
		b := pad.Bounds()
		px, py, _ := pad.PointerPosition()
		dx, dy, dragged := pad.Dragged()
		switch {
		case pad.DoubleClicked():
			set(ax.at(ax.Default), ay.at(ay.Default))
		case pad.Pressed() && b.W > 0 && c.Modifiers()&ui.Shift != 0:
			if dragged {
				set(fx+float64(dx/b.W)/10, fy-float64(dy/b.H)/10)
			}
		case pad.Pressed() && b.W > 0 && (!*held || dragged):
			set(float64(px/b.W), 1-float64(py/b.H))
		}
		*held = pad.Pressed()
		for _, mods := range []ui.Modifiers{0, ui.Shift} {
			step := 0.01
			if mods == ui.Shift {
				step = 0.001
			}
			switch {
			case pad.Shortcut(mods, ui.KeyRight):
				set(fx+step, fy)
			case pad.Shortcut(mods, ui.KeyLeft):
				set(fx-step, fy)
			case pad.Shortcut(mods, ui.KeyUp):
				set(fx, fy+step)
			case pad.Shortcut(mods, ui.KeyDown):
				set(fx, fy-step)
			}
		}
		fx, fy = ax.at(*x), ay.at(*y)
		active := pad.Pressed()
		focus := pad.FocusVisible()
		pad.Draw(func(p *ui.Painter, r ui.Rect) {
			for i := 1; i < 4; i++ {
				gx, gy := r.X+r.W*float32(i)/4, r.Y+r.H*float32(i)/4
				p.Line(gx, r.Y, gx, r.Y+r.H, 1, t.Text.Alpha(0.06))
				p.Line(r.X, gy, r.X+r.W, gy, 1, t.Text.Alpha(0.06))
			}
			cx, cy := r.X+r.W*float32(fx), r.Y+r.H*float32(1-fy)
			// Lines a DIP wide on the middle of a pixel stay sharp; across
			// two, they blur to gray.
			lx, ly := float32(math.Floor(float64(cx)))+0.5, float32(math.Floor(float64(cy)))+0.5
			p.Line(lx, r.Y, lx, r.Y+r.H, 1, t.Accent.Alpha(0.6))
			p.Line(r.X, ly, r.X+r.W, ly, 1, t.Accent.Alpha(0.6))
			rad := t.Space(2)
			if active {
				rad = t.Space(2.5)
			}
			puck := ui.Rect{X: cx - rad, Y: cy - rad, W: 2 * rad, H: 2 * rad}
			p.Shadow(puck, rad, 0, 1, 3, 0, ui.RGB(0, 0, 0).Alpha(0.3))
			p.Fill(puck, t.Accent, rad)
			p.Stroke(puck, ui.RGB(255, 255, 255), rad, 2)
			if focus {
				p.FocusRing(r, [4]float32{t.Space(2), t.Space(2), t.Space(2), t.Space(2)})
			}
		})
		ui.Row(c).Gap(t.Space(4)).Children(func() {
			for _, v := range []struct {
				a Axis
				v float64
			}{{ax, *x}, {ay, *y}} {
				ui.Row(c).Gap(t.Space(1.5)).Grow(1).Children(func() {
					ui.Text(c, v.a.Label).TextColor(t.TextMuted).SingleLine()
					ui.Text(c, v.a.text(v.v)).FontFeatures("tnum").SingleLine()
				})
			}
		})
	})
	return root
}
