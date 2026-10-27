// Package radialmenu is a pie menu, as Blender's and Maya's marking
// menus: items round the pointer, each in a wedge of its own. Pressing
// the right button on the element it is attached to opens it at the
// pointer; moving away from the middle lights the wedge the pointer is
// toward, and letting go there chooses its item, so that a quick stroke
// chooses without looking. Letting go in the middle leaves the menu open
// to click an item. An item with items of its own opens them in a ring
// round itself, as the stroke reaches it or a click chooses it.
//
//	area := ui.Box(c).Grow(1)
//	if path, ok := radialmenu.Attach(c, area, app.tools).Chosen(); ok {
//		app.use(path)
//	}
//
// ← and → go round the items while the menu is open, Enter chooses, and
// Escape or a click in the middle goes back a ring, closing the menu at
// the first; a click outside closes it.
package radialmenu

import (
	"math"
	"time"

	"github.com/egoist/mygo/ui"
)

// Item is an item of a menu: its label, an icon beside it, and the items
// of the ring it opens, if any.
type Item struct {
	Label    string
	Icon     *ui.SVG
	Disabled bool
	Items    []Item
}

// Events are what the user did with a menu in a frame.
type Events struct {
	path []int
}

// Chosen returns the path to the item the user chose: its index among
// the items, then among that item's items, and so on. The menu has
// closed.
func (e Events) Chosen() ([]int, bool) { return e.path, e.path != nil }

// ring is a ring of items open round x, y, in the area's coordinates;
// chosen is the item of it that opened the next ring.
type ring struct {
	x, y   float32
	items  []Item
	chosen int
}

// state is what a menu keeps from frame to frame: its open rings, the
// last of them showing; whether the right button that opened it is still
// down, and whether it was let go since the last frame; the pointer, in
// the area's coordinates, and where it was when the arrow keys last went
// to an item, keyed, -1 when the pointer chooses.
type state struct {
	rings    []ring
	held     bool
	released bool
	px, py   float32
	kx, ky   float32
	keyed    int
}

// Attach opens the menu of items as the right button presses area. It
// takes area's input (HandleInput).
func Attach(c *ui.Context, area ui.Element, items []Item) Events {
	var ev Events
	t := c.Theme()
	st := ui.Local(area, "radialmenu.state", func() state { return state{keyed: -1} })
	area.HandleInput(func(in ui.InputEvent) bool {
		switch {
		case in.Kind == ui.InputPointerDown && in.Button == 1 && len(items) > 0:
			*st = state{rings: []ring{{x: in.X, y: in.Y, items: items}}, held: true, px: in.X, py: in.Y, keyed: -1}
			return true
		case !st.held:
			return false
		case in.Kind == ui.InputPointerMove:
			st.px, st.py = in.X, in.Y
			return true
		case in.Kind == ui.InputPointerUp && in.Button == 1:
			st.held, st.released = false, true
			return true
		}
		return false
	})
	if len(st.rings) == 0 {
		return ev
	}

	inner, reach := t.Space(6), t.Space(22)
	ox, oy := area.LocalToWindow(0, 0)
	r := &st.rings[len(st.rings)-1]
	n := len(r.items)
	angle := func(i int) float64 { return -math.Pi/2 + 2*math.Pi*float64(i)/float64(n) }
	aimed := func() int {
		if st.keyed >= 0 {
			return st.keyed
		}
		dx, dy := float64(st.px-r.x), float64(st.py-r.y)
		if math.Hypot(dx, dy) < float64(inner) {
			return -1
		}
		i := int(math.Round((math.Atan2(dy, dx)+math.Pi/2)/(2*math.Pi)*float64(n))) % n
		if r.items[i].Disabled {
			return -1
		}
		return i
	}
	back := func() { st.rings = st.rings[:len(st.rings)-1] }
	choose := func(i int) {
		if r.items[i].Items == nil {
			ev.path = make([]int, 0, len(st.rings))
			for _, g := range st.rings[:len(st.rings)-1] {
				ev.path = append(ev.path, g.chosen)
			}
			ev.path = append(ev.path, i)
			st.rings = nil
			return
		}
		a := angle(i)
		r.chosen = i
		st.rings = append(st.rings, ring{x: r.x + reach*float32(math.Cos(a)), y: r.y + reach*float32(math.Sin(a)), items: r.items[i].Items})
		st.px, st.py, st.keyed = st.rings[len(st.rings)-1].x, st.rings[len(st.rings)-1].y, -1
	}

	if i := aimed(); st.released {
		st.released = false
		if i >= 0 {
			choose(i)
		}
	} else if st.held && i >= 0 && r.items[i].Items != nil &&
		math.Hypot(float64(st.px-r.x), float64(st.py-r.y)) >= float64(reach) {
		choose(i)
	}

	ui.Overlay(c, func() {
		layer := ui.Box(c).Absolute().Left(0).Top(0).Right(0).Bottom(0).Modal().Role(ui.RoleMenu)
		if len(st.rings) == 0 {
			return
		}
		r = &st.rings[len(st.rings)-1]
		n = len(r.items)
		if !st.held {
			if x, y, ok := layer.PointerPosition(); ok && (x != st.kx || y != st.ky) {
				st.px, st.py, st.keyed = x-ox, y-oy, -1
			}
		}
		step := func(d int) {
			from := aimed()
			if from < 0 {
				from = 0
				if d > 0 {
					from = -1
				}
			}
			for k := 1; k <= n; k++ {
				if i := ((from+d*k)%n + n) % n; !r.items[i].Disabled {
					st.keyed, st.kx, st.ky = i, st.px+ox, st.py+oy
					return
				}
			}
		}
		switch {
		case layer.OverlayShortcut(0, ui.KeyRight), layer.OverlayShortcut(0, ui.KeyDown):
			step(1)
		case layer.OverlayShortcut(0, ui.KeyLeft), layer.OverlayShortcut(0, ui.KeyUp):
			step(-1)
		case layer.OverlayShortcut(0, ui.KeyEnter):
			if i := aimed(); i >= 0 {
				choose(i)
			}
		case layer.OverlayShortcut(0, ui.KeyEscape):
			back()
		case layer.RightClicked():
			st.rings = nil
		case layer.Clicked():
			d := math.Hypot(float64(st.px-r.x), float64(st.py-r.y))
			switch i := aimed(); {
			case d < float64(inner):
				back()
			case d > float64(2*reach) || i < 0:
				st.rings = nil
			default:
				choose(i)
			}
		}
		if len(st.rings) == 0 {
			return
		}
		r = &st.rings[len(st.rings)-1]
		n = len(r.items)
		on := aimed()

		layer.Children(func() {
			size := 2 * (reach + t.Space(20))
			cx, cy := ox+r.x, oy+r.y
			wheel := ui.Box(c.Key(len(st.rings))).Absolute().Left(cx-size/2).Top(cy-size/2).Size(size, size).PassThrough().
				Transition(ui.ElementTransition{Enter: &ui.Motion{Scale: 0.85}, Duration: 120 * time.Millisecond})
			toward := math.Atan2(float64(st.py-r.y), float64(st.px-r.x))
			if st.keyed >= 0 {
				toward = angle(st.keyed)
			}
			wheel.Draw(func(p *ui.Painter, b ui.Rect) {
				mx, my := b.X+b.W/2, b.Y+b.H/2
				disc := ui.Rect{X: mx - inner, Y: my - inner, W: 2 * inner, H: 2 * inner}
				p.Fill(disc, t.Background.Alpha(0.9), inner)
				p.Stroke(disc, t.Border, inner, 1.5)
				if on < 0 {
					return
				}
				arc := make([][2]float32, 13)
				for k := range arc {
					b := toward - 0.5 + float64(k)/12
					arc[k] = [2]float32{mx + (inner-1)*float32(math.Cos(b)), my + (inner-1)*float32(math.Sin(b))}
				}
				p.StrokePolyline(arc, 3, t.Accent)
			})
			wheel.Children(func() {
				h := t.Space(8)
				for i, it := range r.items {
					w := pillWidth(c, t, it)
					a := angle(i)
					ax := size/2 + reach*float32(math.Cos(a))
					ay := size/2 + reach*float32(math.Sin(a))
					// Pills on the right start at their point and those on
					// the left end at it, so that wide labels clear the
					// pills above and below.
					x := ax - w/2*(1-float32(math.Cos(a)))
					pill(c.Key(i), t, it, i == on).Absolute().Left(x).Top(ay - h/2).Width(w).Height(h)
				}
			})
		})
	})
	return ev
}

// pillWidth returns how wide the pill of it is.
func pillWidth(c *ui.Context, t *ui.Theme, it Item) float32 {
	w, _ := c.MeasureText(0, ui.Span{Text: it.Label, Size: t.FontSize})
	w += 2 * t.Space(3.5)
	if it.Icon != nil {
		w += t.Space(4) + t.Space(1.5)
	}
	if it.Items != nil {
		w += t.Space(3)
	}
	return w
}

func pill(c *ui.Context, t *ui.Theme, it Item, on bool) ui.Element {
	fg, bg := t.Text, t.Background
	switch {
	case it.Disabled:
		fg = t.TextMuted.Alpha(0.6)
	case on:
		fg, bg = t.AccentText, t.Accent
	}
	row := ui.Row(c).PassThrough().AlignItems(ui.Center).Justify(ui.Center).Gap(t.Space(1.5)).Radius(t.Space(4)).
		Background(bg).Border(1, t.Border).Shadow(0, 2, 8, 0, ui.RGB(0, 0, 0).Alpha(0.18)).
		Role(ui.RoleMenuItem).Label(it.Label).Disabled(it.Disabled).Selected(on)
	row.Children(func() {
		if it.Icon != nil {
			ui.Icon(c, it.Icon).Size(t.Space(4), t.Space(4)).TextColor(fg)
		}
		ui.Text(c, it.Label).TextColor(fg).SingleLine()
		if it.Items != nil {
			ui.Text(c, "›").TextColor(fg.Alpha(0.7))
		}
	})
	return row
}
