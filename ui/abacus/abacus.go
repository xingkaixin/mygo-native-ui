// Package abacus draws a soroban, the Japanese abacus, and reads its
// number: a wooden frame of rods, each with one bead above the beam
// worth five and four below it worth one each, counted when pushed to
// the beam, and a dot on the beam at every third rod from the right,
// where the thousands and millions start. A click on a bead moves it to
// the beam, with the beads between, or away from it, with the beads
// behind it, and the beads slide there. With a rod focused, the up and
// down arrows add and take one, and a digit key sets it. Each rod's
// digit shows over it and the number over the frame. The abacus changes
// the app's number in place and reports the changes:
//
//	if abacus.View(c, &app.total, 13).Changed() {
//		app.check()
//	}
package abacus

import (
	"strconv"
	"time"

	"github.com/egoist/mygo/ui"
)

var slide = ui.Animation{Spring: ui.SpringOptions{Duration: 180 * time.Millisecond}}

// View creates the abacus of rods rods showing *value, which it changes
// in place, reporting the changes. A value beyond what the rods hold is
// shown as its last digits.
func View(c *ui.Context, value *int64, rods int) ui.Element {
	t := c.Theme()
	digits := make([]int, rods)
	v := max(*value, 0)
	for i := rods - 1; i >= 0; i-- {
		digits[i] = int(v % 10)
		v /= 10
	}
	s := t.Space(12)
	bh := s * 0.5
	beam := t.Space(2.5)
	frame := t.Space(3.5)
	inner := t.Space(2)
	wood := ui.Oklch(0.5, 0.09, 55)
	back := ui.Oklch(0.94, 0.025, 80)
	if t.Dark {
		wood, back = ui.Oklch(0.42, 0.07, 55), ui.Oklch(0.27, 0.015, 60)
	}
	changed := false
	root := ui.Column(c).Gap(t.Space(3)).AlignItems(ui.Start).Role(ui.RoleGroup).Label("Abacus")
	root.Children(func() {
		ui.Text(c, group(digits)).Font("monospace").FontSize(t.FontSize+12).FontWeight(600).FontFeatures("tnum").
			Padding(0, frame+inner)
		ui.Column(c).Gap(t.Space(1)).Children(func() {
			ui.Row(c).Padding(0, frame+inner).Children(func() {
				for i, d := range digits {
					l := ui.Text(c.Key(i), strconv.Itoa(d)).Width(s).TextAlign(ui.Center).Font("monospace").FontSize(t.FontSize - 1)
					if d == 0 {
						l.TextColor(t.TextMuted.Alpha(0.5))
					} else {
						l.TextColor(t.TextMuted)
					}
				}
			})
			box := ui.Row(c).Padding(inner).Background(back).Border(frame, wood).Radius(t.Space(2.5))
			box.Children(func() {
				for i := range digits {
					if rod(c.Key(i), &digits[i], rods-1-i, s, bh, beam) {
						changed = true
					}
				}
			})
			box.DrawOver(func(p *ui.Painter, r ui.Rect) {
				y := r.Y + frame + inner + 2*bh
				p.Fill(ui.Rect{X: r.X + frame, Y: y, W: r.W - 2*frame, H: beam}, wood, 0)
				for i := range digits {
					if (rods-1-i)%3 != 0 {
						continue
					}
					d := beam * 0.22
					cx := r.X + frame + inner + s*float32(i) + s/2
					p.Fill(ui.Rect{X: cx - d, Y: y + beam/2 - d, W: 2 * d, H: 2 * d}, ui.Oklch(0.93, 0.03, 80), d)
				}
			})
		})
	})
	if changed {
		var n int64
		for _, d := range digits {
			n = n*10 + int64(d)
		}
		*value = n
		root.SetChanged()
	}
	return root
}

// rod creates the rod of place value 10^place showing *d, and reports
// whether a click or a key changed it.
func rod(c *ui.Context, d *int, place int, s, bh, beam float32) bool {
	t := c.Theme()
	earth := 2*bh + beam
	e := ui.Box(c).Size(s, 7*bh+beam).Focusable().FocusRing(false).Cursor(ui.CursorPointer).
		Role(ui.RoleSlider).Label("Rod " + strconv.Itoa(place+1) + " from the right").Value(strconv.Itoa(*d))
	old := *d
	if e.Clicked() {
		_, y, _ := e.PointerPosition()
		ones := *d % 5
		switch {
		case y < 2*bh && *d >= 5:
			*d -= 5
		case y < 2*bh:
			*d += 5
		case y >= earth:
			for k := range 4 {
				top := earth + float32(k+1)*bh
				if k < ones {
					top -= bh
				}
				if y >= top && y < top+bh {
					if k < ones {
						*d = *d - ones + k
					} else {
						*d = *d - ones + k + 1
					}
				}
			}
		}
	}
	switch {
	case e.Shortcut(0, ui.KeyUp):
		*d = min(*d+1, 9)
	case e.Shortcut(0, ui.KeyDown):
		*d = max(*d-1, 0)
	}
	for k, key := range []ui.Key{ui.Key0, ui.Key1, ui.Key2, ui.Key3, ui.Key4, ui.Key5, ui.Key6, ui.Key7, ui.Key8, ui.Key9} {
		if e.Shortcut(0, key) {
			*d = k
		}
	}
	heaven := e.AnimateTo(-1, pushed(*d >= 5), slide)
	var ones [4]float32
	for k := range ones {
		ones[k] = e.AnimateTo(k, pushed(k < *d%5), slide)
	}
	bead := ui.Oklch(0.36, 0.08, 40)
	if t.Dark {
		bead = ui.Oklch(0.6, 0.09, 55)
	}
	e.Draw(func(p *ui.Painter, r ui.Rect) {
		cx := r.X + r.W/2
		p.Line(cx, r.Y, cx, r.Y+r.H, 2.5, ui.Oklch(0.55, 0.02, 60))
		w := r.W * 0.92
		draw := func(y float32) {
			x0, x1 := cx-w/2, cx+w/2
			g := bh * 0.08
			top, bot, mid := y+g, y+bh-g, y+bh/2
			var path ui.Path
			path.MoveTo(x0, mid).LineTo(x0+w*0.32, top).LineTo(x1-w*0.32, top).LineTo(x1, mid).
				LineTo(x1-w*0.32, bot).LineTo(x0+w*0.32, bot).Close()
			p.FillPath(&path, bead)
			var shine ui.Path
			shine.MoveTo(x0+w*0.1, mid-bh*0.08).LineTo(x0+w*0.34, top+bh*0.08).LineTo(x1-w*0.34, top+bh*0.08).
				LineTo(x1-w*0.1, mid-bh*0.08).Close()
			p.FillPath(&shine, ui.RGBA(255, 255, 255, 0.16))
		}
		draw(r.Y + bh*heaven)
		for k, a := range ones {
			rest := earth + float32(k+1)*bh
			draw(r.Y + rest - bh*a)
		}
		if e.FocusVisible() {
			p.Stroke(ui.Rect{X: r.X + 1, Y: r.Y - 2, W: r.W - 2, H: r.H + 4}, t.Accent, t.Space(1.5), 2)
		}
	})
	return *d != old
}

func pushed(on bool) float32 {
	if on {
		return 1
	}
	return 0
}

// group writes digits as a number with thousands separated, without
// leading zeros.
func group(digits []int) string {
	s := ""
	for _, d := range digits {
		if s != "" || d != 0 {
			s += strconv.Itoa(d)
		}
	}
	if s == "" {
		s = "0"
	}
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}
