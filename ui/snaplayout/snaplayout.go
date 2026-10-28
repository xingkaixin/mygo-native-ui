// Package snaplayout offers the layouts a window can snap into, as the
// zoom button of a window offers them: miniature screens split into
// halves, thirds, quarters and the like, each zone lighting up in the
// accent color as the pointer goes over it, and a click on a zone, or
// Space while it has the focus, choosing it. Frame turns the zone into
// the window's frame on a screen, a gap between windows and around them.
//
//	ev := snaplayout.Picker(c, snaplayout.Layouts)
//	if ev.Chosen != nil {
//		win.frame = snaplayout.Frame(screen, *ev.Chosen, 8)
//	}
package snaplayout

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
)

// Zone is part of the screen, as fractions of its width and height.
type Zone struct {
	X, Y, W, H float64
}

// Layout is a way to split the screen into zones.
type Layout []Zone

// Layouts are the layouts offered by default: halves, thirds, two thirds
// beside a third and back, quarters, a half beside two quarters, a third
// on each side of a wide middle, and the whole screen.
var Layouts = []Layout{
	{{0, 0, 0.5, 1}, {0.5, 0, 0.5, 1}},
	{{0, 0, 1.0 / 3, 1}, {1.0 / 3, 0, 1.0 / 3, 1}, {2.0 / 3, 0, 1.0 / 3, 1}},
	{{0, 0, 2.0 / 3, 1}, {2.0 / 3, 0, 1.0 / 3, 1}},
	{{0, 0, 1.0 / 3, 1}, {1.0 / 3, 0, 2.0 / 3, 1}},
	{{0, 0, 0.5, 0.5}, {0.5, 0, 0.5, 0.5}, {0, 0.5, 0.5, 0.5}, {0.5, 0.5, 0.5, 0.5}},
	{{0, 0, 0.5, 1}, {0.5, 0, 0.5, 0.5}, {0.5, 0.5, 0.5, 0.5}},
	{{0, 0, 0.25, 1}, {0.25, 0, 0.5, 1}, {0.75, 0, 0.25, 1}},
	{{0, 0, 0.5, 0.5}, {0, 0.5, 0.5, 0.5}, {0.5, 0, 0.5, 1}},
	{{0, 0, 1, 1}},
}

// Frame returns where a window in zone z goes on screen, gap apart from
// its neighbors and from the screen's edges.
func Frame(screen ui.Rect, z Zone, gap float32) ui.Rect {
	x0 := screen.X + float32(z.X)*screen.W
	y0 := screen.Y + float32(z.Y)*screen.H
	x1 := screen.X + float32(z.X+z.W)*screen.W
	y1 := screen.Y + float32(z.Y+z.H)*screen.H
	// Inner edges are shared with a neighbor, which takes the other half
	// of the gap.
	inset := func(at, edge float32) float32 {
		if at == edge {
			return gap
		}
		return gap / 2
	}
	x0 += inset(x0, screen.X)
	y0 += inset(y0, screen.Y)
	x1 -= inset(x1, screen.X+screen.W)
	y1 -= inset(y1, screen.Y+screen.H)
	return ui.Rect{X: x0, Y: y0, W: x1 - x0, H: y1 - y0}
}

// name returns how assistive technology hears z, as "Left half" or "Top
// right quarter".
func name(z Zone) string {
	if z.W == 1 && z.H == 1 {
		return "Full screen"
	}
	var words []string
	if z.H < 1 {
		words = append(words, map[bool]string{true: "top", false: "bottom"}[z.Y == 0])
	}
	switch {
	case z.W == 1:
	case z.X == 0:
		words = append(words, "left")
	case z.X+z.W > 0.999:
		words = append(words, "right")
	default:
		words = append(words, "middle")
	}
	part := fmt.Sprintf("%.0f%%", z.W*z.H*100)
	for share, word := range map[float64]string{0.25: "quarter", 1.0 / 3: "third", 0.5: "half", 2.0 / 3: "two thirds"} {
		if math.Abs(z.W*z.H-share) < 0.01 {
			part = word
		}
	}
	s := strings.Join(append(words, part), " ")
	return strings.ToUpper(s[:1]) + s[1:]
}

// Events is a picker, and the zone chosen in it in the frame, nil when
// none was.
type Events struct {
	ui.Element
	Chosen *Zone
}

// Picker creates the miniature screens of layouts, three to a row.
func Picker(c *ui.Context, layouts []Layout) Events {
	t := c.Theme()
	var ev Events
	w := t.Space(26)
	h := w * 0.625
	gap := t.Space(0.75)
	grid := ui.Grid(c).Columns(3).Gap(t.Space(2.5)).Padding(t.Space(3)).Role(ui.RoleGroup).Label("Snap layouts")
	grid.Children(func() {
		for i, layout := range layouts {
			ui.Box(c.Key(i)).Size(w, h).Radius(t.Space(1.5)).Background(t.Text.Alpha(0.06)).Border(1, t.Border).Children(func() {
				for k, z := range layout {
					f := Frame(ui.Rect{W: w, H: h}, z, gap*2)
					zone := ui.ButtonBase(c.Key(k)).Absolute().Left(f.X).Top(f.Y).Size(f.W, f.H).Radius(t.Space(1)).Label(name(z))
					fill := t.Text.Alpha(0.16)
					if zone.Hovered() || zone.FocusVisible() {
						fill = t.Accent
					}
					zone.Background(zone.AnimateColor("snaplayout.fill", fill, ui.Animation{Duration: 120 * time.Millisecond}))
					if zone.Clicked() {
						chosen := z
						ev.Chosen = &chosen
					}
				}
			})
		}
	})
	if ev.Chosen != nil {
		grid.SetChanged()
	}
	ev.Element = grid
	return ev
}
