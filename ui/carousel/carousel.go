// Package carousel shows views side by side, one at a time, as pages the
// user moves between, as an onboarding sheet or a gallery does:
//
//	carousel.View(c, &app.page, len(app.slides), func(i int) {
//		slide(c, app.slides[i])
//	}).Grow(1).Label("Welcome")
//
// Dragging the pages, or scrolling sideways on a trackpad, moves them;
// let go, they settle on the nearest page with a spring, and a flick goes
// on to the next one. Past the first and the last page they give less and
// less, and spring back. Arrows show at the sides while the pointer is
// over the pages, and dots below them show the page, a click on a dot
// going there; while the view has the keyboard focus, ← and → page.
//
// *page is the page shown: the view sets it as the user pages, and the
// app sets it to go to another, as a Continue button does, the pages
// sliding there. Only the page shown and those beside it are built, and
// those the pages pass on the way.
package carousel

import (
	"math"
	"strconv"
	"time"

	"github.com/egoist/mygo/ui"
)

// scrollIdle is how long after its last scroll a trackpad lets go of the
// pages: scrolling tells no gesture's end, which is when it stops, its
// momentum included.
const scrollIdle = 150 * time.Millisecond

// fade shows the arrows and hides them.
var fade = ui.ElementTransition{Colors: true, Enter: &ui.Motion{}, Exit: &ui.Motion{}}

// state is what a view keeps from frame to frame: the page it showed, a
// fraction between two as they move; whether the pointer or the trackpad
// holds the pages, and how many DIPs it moved them to the right of *page;
// and the DIPs scrolled since the last frame and when the last scroll
// came, zero once it let go.
type state struct {
	shown    float32
	held     bool
	drag     float32
	scroll   float32
	scrolled time.Time
}

// View creates a view of n pages, of which *page is the one shown, and
// build(i) builds page i in a column the size of the pages. Give the view
// a size, or grow it in its parent, and name it with Label.
func View(c *ui.Context, page *int, n int, build func(i int)) ui.Element {
	t := c.Theme()
	view := ui.Column(c).Focusable().Gap(t.Space(3))
	if n <= 0 {
		return view
	}
	show := func(i int) { *page = max(0, min(n-1, i)) }
	show(*page)
	if view.Shortcut(0, ui.KeyLeft) {
		show(*page - 1)
	}
	if view.Shortcut(0, ui.KeyRight) {
		show(*page + 1)
	}
	view.Value("Page " + strconv.Itoa(*page+1) + " of " + strconv.Itoa(n))

	view.Children(func() {
		area := ui.Box(c).Grow(1).Clip()
		st := ui.Local(area, "carousel.state", func() state { return state{shown: float32(*page)} })
		area.HandleInput(func(ev ui.InputEvent) bool {
			// A scroll that starts sideways keeps the pages until it stops,
			// as a trackpad's strays up and down; one that starts upright
			// goes on to the scroll views around.
			if ev.Kind != ui.InputScroll || st.scrolled.IsZero() && abs(ev.DX) <= abs(ev.DY) {
				return false
			}
			st.scroll += ev.DX
			st.scrolled = ev.Time
			return true
		})
		w := max(area.LayoutBounds().W, 1)

		scrolling := !st.scrolled.IsZero()
		if wait := scrollIdle - c.Now().Sub(st.scrolled); scrolling && wait > 0 {
			c.After(wait)
		} else {
			st.scrolled, scrolling = time.Time{}, false
		}
		dx, _, pressing := area.Dragged()
		if (pressing || scrolling) && !st.held {
			st.held, st.drag = true, (float32(*page)-st.shown)*w
		}
		var pos float32
		switch {
		case st.held && (pressing || scrolling):
			st.drag = max(-w, min(w, st.drag+dx-st.scroll))
			st.scroll = 0
			pos = area.Jump("x", ui.Rubberband(float32(*page)-st.drag/w, 0, float32(n-1), 1))
		case st.held:
			vx, _, _ := area.DragEnded()
			v := -vx / w
			to := int(math.Round(float64(ui.Project(float32(*page)-st.drag/w, v))))
			show(max(*page-1, min(*page+1, to)))
			st.held, st.drag = false, 0
			// Jumping in place forgets how fast the last move went, which a
			// pointer resting before it let go no longer goes.
			area.Jump("x", st.shown)
			a := ui.Smooth
			a.Spring.Velocity = v
			pos = area.AnimateTo("x", float32(*page), a)
		default:
			pos = area.AnimateTo("x", float32(*page), ui.Smooth)
		}
		st.shown = pos

		area.Children(func() {
			for i := range n {
				off := float32(i) - pos
				if abs(off) >= 1 && (i < *page-1 || i > *page+1) {
					continue
				}
				p := ui.Column(c.Key(i)).Absolute().Top(0).LeftPercent(off * 100).WidthPercent(100).HeightPercent(100)
				if abs(off) >= 1 {
					p.Invisible()
				}
				p.Children(func() { build(i) })
			}
			if area.Hovered() {
				if *page > 0 && arrow(c, false) {
					show(*page - 1)
				}
				if *page < n-1 && arrow(c, true) {
					show(*page + 1)
				}
			}
		})

		dot := t.Space(2)
		ui.Row(c).Justify(ui.Center).Children(func() {
			for i := range n {
				d := ui.Box(c).Size(2*dot, 2*dot).Role(ui.RoleButton).Label("Page " + strconv.Itoa(i+1)).KeepFocus()
				if d.Clicked() {
					show(i)
				}
				color := t.Text.Alpha(0.2)
				if d.Hovered() {
					color = t.Text.Alpha(0.4)
				}
				color = color.Mix(t.Accent, max(0, 1-abs(float32(i)-pos)))
				d.Draw(func(p *ui.Painter, r ui.Rect) {
					p.Fill(ui.Rect{X: r.X + (r.W-dot)/2, Y: r.Y + (r.H-dot)/2, W: dot, H: dot}, color, dot/2)
				})
			}
		})
	})
	return view
}

// arrow builds the button at the left of the pages, or with next at their
// right, and reports whether it was clicked.
func arrow(c *ui.Context, next bool) bool {
	t := c.Theme()
	size, inset := t.Space(9), t.Space(3)
	label, at := "Previous page", ui.AnchorLeft
	if next {
		label, at = "Next page", ui.AnchorRight
	}
	b := ui.Box(c.Key(label)).Attach(at, at).Size(size, size).Radius(size/2).Role(ui.RoleButton).Label(label).KeepFocus().
		Background(t.Background).Border(1, t.Border).Shadow(0, 1, 4, 0, ui.RGB(0, 0, 0).Alpha(0.12)).Transition(fade)
	if next {
		b.Right(inset)
	} else {
		b.Left(inset)
	}
	switch {
	case b.Pressed():
		b.Background(t.SurfacePressed)
	case b.Hovered():
		b.Background(t.SurfaceHover)
	}
	b.Draw(func(p *ui.Painter, r ui.Rect) {
		cx, cy, k := r.X+r.W/2, r.Y+r.H/2, r.W/9
		dir := float32(1)
		if next {
			dir = -1
		}
		p.StrokePolyline([][2]float32{{cx + dir*k/2, cy - k}, {cx - dir*k/2, cy}, {cx + dir*k/2, cy + k}}, 1.75, t.Text)
	})
	return b.Clicked()
}

func abs(v float32) float32 { return float32(math.Abs(float64(v))) }
