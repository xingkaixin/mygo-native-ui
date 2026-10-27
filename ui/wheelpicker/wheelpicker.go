// Package wheelpicker is the spinning wheels of iOS's pickers: choices
// round a drum, the chosen one in the band across its middle. Dragging
// turns the drum and a flick spins it on, settling on the nearest
// choice; the scroll wheel steps it a choice a notch, and a trackpad
// turns it as dragging does. A click on a choice above or below the band
// turns to it, and ↑ and ↓ step while a wheel has the keyboard focus.
// Wheels side by side in a View share the band, as a date's month, day
// and year do:
//
//	wheelpicker.View(c, func() {
//		wheelpicker.Wheel(c, &app.month, months).Label("Month")
//		wheelpicker.Wheel(c, &app.day, days).Label("Day")
//		wheelpicker.Cycle(c, &app.minute, minutes).Label("Minute")
//	})
//
// A wheel sets *selected as it is let go, and Changed reports it. Its
// choices may change from frame to frame, as the days of a month do;
// *selected then stays among them.
package wheelpicker

import (
	"math"
	"time"

	"github.com/egoist/mygo/ui"
)

// rows is how many rows tall the drum is, its front half-turn.
const rows = 5

// scrollIdle is how long after its last scroll a trackpad lets go of the
// drum: scrolling tells no gesture's end, which is when it stops, its
// momentum included.
const scrollIdle = 150 * time.Millisecond

// state is what a wheel keeps from frame to frame: the choice it turns
// to, counted on past the ends on a cycle; where it showed, in rows;
// whether the pointer or the trackpad holds it, and where it holds it;
// the DIPs a trackpad scrolled since the last frame and when it last
// did, zero once it let go; and the notches of a scroll wheel.
type state struct {
	at       int
	shown    float32
	held     bool
	drag     float32
	scroll   float32
	scrolled time.Time
	notches  int
}

func rowHeight(t *ui.Theme) float32 { return t.Space(8) }

// View creates a row of wheels with the band across them; fn builds the
// wheels.
func View(c *ui.Context, fn func()) ui.Element {
	t := c.Theme()
	h := rowHeight(t)
	return ui.Row(c).Height(rows * h).PaddingX(t.Space(2)).Draw(func(p *ui.Painter, r ui.Rect) {
		p.Fill(ui.Rect{X: r.X, Y: r.Y + (r.H-h)/2, W: r.W, H: h}, t.Text.Alpha(0.07), t.Radius+2)
	}).Children(fn)
}

// Wheel creates a wheel choosing one of items, items[*selected]. It is
// as wide as its widest choice; Width sets it otherwise.
func Wheel(c *ui.Context, selected *int, items []string) ui.Element {
	return wheel(c, selected, items, false)
}

// Cycle creates a wheel whose choices go round, the first after the
// last, as the hours and minutes of a time.
func Cycle(c *ui.Context, selected *int, items []string) ui.Element {
	return wheel(c, selected, items, true)
}

func wheel(c *ui.Context, selected *int, items []string, cycle bool) ui.Element {
	t := c.Theme()
	h := rowHeight(t)
	size := t.FontSize + 5
	n := len(items)
	w := ui.Box(c).FillHeight().Focusable().Role(ui.RoleSlider)
	if n == 0 {
		return w
	}
	*selected = max(0, min(n-1, *selected))
	var widest float32
	for _, s := range items {
		widest = max(widest, first(c.MeasureText(0, ui.Span{Text: s, Size: size})))
	}
	w.Width(widest+t.Space(6)).Range(0, float64(n-1), float64(*selected)).Value(items[*selected])

	st := ui.Local(w, "wheelpicker.state", func() state { return state{at: *selected, shown: float32(*selected)} })
	if d := mod(*selected-st.at, n); d != 0 {
		if cycle && d > n/2 {
			d -= n
		}
		if cycle {
			st.at += d
		} else {
			st.at = *selected
		}
	}
	turn := func(to int) {
		if !cycle {
			to = max(0, min(n-1, to))
		}
		st.at = to
		if k := mod(to, n); k != *selected {
			*selected = k
			w.SetChanged()
		}
	}

	w.HandleInput(func(ev ui.InputEvent) bool {
		if ev.Kind != ui.InputScroll {
			return false
		}
		if ev.Precise {
			st.scroll += ev.DY
			st.scrolled = ev.Time
		} else if ev.DY != 0 {
			st.notches += int(math.Copysign(1, float64(ev.DY)))
		}
		return true
	})
	switch {
	case w.Shortcut(0, ui.KeyUp):
		turn(st.at - 1)
	case w.Shortcut(0, ui.KeyDown):
		turn(st.at + 1)
	case !cycle && w.Shortcut(0, ui.KeyHome):
		turn(0)
	case !cycle && w.Shortcut(0, ui.KeyEnd):
		turn(n - 1)
	}
	if st.notches != 0 {
		turn(st.at + st.notches)
		st.notches = 0
	}

	scrolling := !st.scrolled.IsZero()
	if wait := scrollIdle - c.Now().Sub(st.scrolled); scrolling && wait > 0 {
		c.After(wait)
	} else {
		st.scrolled, scrolling = time.Time{}, false
	}
	radius := rows * h / 2
	_, dy, pressing := w.Dragged()
	clicked := w.Clicked()
	var pos float32
	switch {
	case pressing || scrolling:
		if !st.held {
			st.held, st.drag = true, st.shown
		}
		st.drag += (st.scroll - dy) / h
		st.scroll = 0
		at := st.drag
		if !cycle {
			at = ui.Rubberband(at, 0, float32(n-1), 2)
		}
		pos = w.Jump("y", at)
	case st.held:
		st.held = false
		_, vy, dragged := w.DragEnded()
		v := -vy / h
		to := ui.Project(st.drag, v)
		if _, y, ok := w.PointerPosition(); clicked && !dragged && ok {
			to = float32(math.Round(float64(st.drag))) + rowAt(y-radius, radius, h)
		}
		turn(int(math.Round(float64(to))))
		// Jumping in place forgets how fast the last move went, which a
		// pointer resting before it let go no longer goes.
		w.Jump("y", st.shown)
		a := ui.Smooth
		a.Spring.Velocity = v
		pos = w.AnimateTo("y", float32(st.at), a)
	default:
		pos = w.AnimateTo("y", float32(st.at), ui.Smooth)
	}
	st.shown = pos

	focus := w.FocusVisible()
	w.Draw(func(p *ui.Painter, r ui.Rect) {
		step := h / radius
		cy := r.Y + r.H/2
		reach := int(math.Ceil(math.Pi / 2 / float64(step)))
		mid := int(math.Round(float64(pos)))
		for i := mid - reach; i <= mid+reach; i++ {
			k := i
			if cycle {
				k = mod(i, n)
			} else if i < 0 || i >= n {
				continue
			}
			a := float64((float32(i) - pos) * step)
			if math.Abs(a) >= math.Pi/2 {
				continue
			}
			sy := float32(math.Cos(a))
			y := cy + radius*float32(math.Sin(a))
			tw, th := p.MeasureText(0, ui.Span{Text: items[k], Size: size})
			near := max(0, 1-abs(float32(i)-pos))
			col := t.TextMuted.Alpha(sy*sy).Mix(t.Text, near)
			x := r.X + (r.W-tw)/2
			p.Transform(ui.Scaling(1, sy), x, y, func() {
				p.Text(x, y-th/2, items[k], size, col)
			})
		}
		if focus {
			p.FocusRing(ui.Rect{X: r.X, Y: cy - h/2, W: r.W, H: h}, [4]float32{t.Radius, t.Radius, t.Radius, t.Radius})
		}
	})
	return w
}

// rowAt returns how many rows from the band a point dy DIPs below the
// drum's middle shows, on a drum of the radius.
func rowAt(dy, radius, h float32) float32 {
	s := max(-1, min(1, float64(dy/radius)))
	return float32(math.Round(math.Asin(s) * float64(radius/h)))
}

func mod(a, n int) int { return (a%n + n) % n }

func abs(v float32) float32 { return float32(math.Abs(float64(v))) }

func first(w, _ float32) float32 { return w }
