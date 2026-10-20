// Package waveform shows audio as its waveform, as Audacity and Logic
// show a clip: the peaks of its samples, played in the accent color up to
// the playhead, which a click moves, and a stretch that dragging selects.
//
//	peaks := waveform.Peaks(samples, 1<<14) // once, not every frame
//	...
//	if waveform.View(c, &app.wave, peaks).Height(120).Changed() {
//		app.player.Seek(app.wave.Playhead * app.duration)
//	}
//
// Positions are fractions of the clip, from 0 at its start to 1 at its
// end.
package waveform

import (
	"math"
	"strconv"

	"github.com/egoist/mygo/ui"
)

// Peak is the lowest and the highest sample of a stretch of samples,
// from -1 to 1.
type Peak struct{ Lo, Hi float32 }

// Peaks reduces samples to n peaks, or to as many as samples, each of an
// equal stretch of them, for View. A view zooms in until a peak is 8 DIPs
// wide, so more peaks show more detail zoomed in, at the cost of memory.
func Peaks(samples []float32, n int) []Peak {
	n = min(n, len(samples))
	peaks := make([]Peak, n)
	for i := range peaks {
		part := samples[i*len(samples)/n : (i+1)*len(samples)/n]
		p := Peak{Lo: part[0], Hi: part[0]}
		for _, v := range part[1:] {
			p.Lo, p.Hi = min(p.Lo, v), max(p.Hi, v)
		}
		peaks[i] = p
	}
	return peaks
}

// State is what a view shows: the playhead, the stretch selected, and
// how far it is zoomed in and where. Keep it in the app's state, one for
// each view, and move Playhead as the clip plays. The zero State shows
// the whole clip.
type State struct {
	// Playhead is where the clip plays from, a fraction of it.
	Playhead float64
	from, to float64
	// zoom is how many times the view's width the clip is, and offset the
	// fraction of the clip at the view's left.
	zoom, offset float64
	// anchor is where a drag selecting began, and played the playhead the
	// view showed last, to tell the app moving it from the user.
	anchor, played float64
}

// Selection returns the stretch selected, from from to to, empty with
// none.
func (s *State) Selection() (from, to float64) { return s.from, s.to }

// Zoom returns how many times the view's width the clip is: 1 shows it
// whole.
func (s *State) Zoom() float64 { return max(1, s.zoom) }

// zoomAt zooms to z keeping the fraction of the clip at x, a fraction of
// the view's width, where it is; zoomed in no further than until a peak
// of n is 8 DIPs of a view w wide.
func (s *State) zoomAt(z, x float64, n int, w float32) {
	at := s.offset + x/s.Zoom()
	s.zoom = max(1, min(z, float64(n)*8/max(1, float64(w))))
	s.offset = at - x/s.zoom
	s.clamp()
}

func (s *State) clamp() {
	s.offset = max(0, min(1-1/s.Zoom(), s.offset))
}

// View creates a view of peaks, from Peaks, of which s says what shows.
// Give it a size, or grow it in its parent. Changed reports the user
// moving the playhead.
//
// A click moves the playhead, and dragging selects a stretch, with the
// playhead at its start. Scrolling pans, and with Cmd held, Command on
// macOS and Control elsewhere, zooms around the pointer; while the view
// has the keyboard focus, Cmd+= and Cmd+- zoom around the playhead and
// Cmd+0 shows the whole clip. Zoomed in, the view follows the playhead
// as the app moves it.
func View(c *ui.Context, s *State, peaks []Peak) ui.Element {
	t := c.Theme()
	n := len(peaks)
	view := ui.Box(c).Clip().Focusable().Role(ui.RoleSlider).Range(0, 1, s.Playhead)
	w := view.LayoutBounds().W
	// at returns the fraction of the clip at x DIPs from the left of a
	// view w wide.
	at := func(x, w float32) float64 {
		return max(0, min(1, s.offset+float64(x/max(1, w))/s.Zoom()))
	}

	pressed := ui.Local(view, "waveform.pressed", func() bool { return false })
	if _, _, ok := view.Dragged(); ok {
		px, _, _ := view.PointerPosition()
		f := at(px, w)
		if !*pressed {
			s.anchor = f
		}
		s.from, s.to = min(s.anchor, f), max(s.anchor, f)
		if s.Playhead != s.from || !*pressed {
			s.Playhead = s.from
			view.SetChanged()
		}
		*pressed = true
	} else {
		*pressed = false
	}
	if s.Playhead != s.played && !*pressed {
		left, right := s.offset, s.offset+1/s.Zoom()
		if s.Playhead < left || s.Playhead > right {
			s.offset = s.Playhead
			s.clamp()
		}
	}
	s.played = s.Playhead

	for _, k := range []struct {
		key ui.Key
		z   float64
	}{{ui.KeyEqual, s.Zoom() * 2}, {ui.KeyMinus, s.Zoom() / 2}, {ui.Key0, 1}} {
		if view.Shortcut(ui.Cmd, k.key) {
			s.zoomAt(k.z, max(0, min(1, (s.Playhead-s.offset)*s.Zoom())), n, w)
		}
	}
	// Scrolling comes between builds, at the box laid out since, which
	// only a handle reads then.
	ref := ui.Local(view, "waveform.handle", func() ui.Handle { return ui.Handle{} })
	view.Bind(ref)
	view.HandleInput(func(ev ui.InputEvent) bool {
		if ev.Kind != ui.InputScroll {
			return false
		}
		w := ref.Bounds(c).W
		if ev.Mods&ui.Cmd != 0 {
			// Scrolling 300 DIPs zooms by 2, as zoomview does.
			s.zoomAt(s.Zoom()*math.Exp2(float64(ev.DY)/300), float64(ev.X/max(1, w)), n, w)
			return true
		}
		d := ev.DX
		if d == 0 {
			d = ev.DY
		}
		before := s.offset
		s.offset -= float64(d/max(1, w)) / s.Zoom()
		s.clamp()
		return s.offset != before
	})
	view.Value(strconv.Itoa(int(math.Round(s.Playhead*100))) + "%")

	view.Draw(func(p *ui.Painter, r ui.Rect) {
		if n == 0 {
			return
		}
		x := func(f float64) float32 { return r.X + float32((f-s.offset)*s.Zoom())*r.W }
		if x0, x1 := max(r.X, x(s.from)), min(r.X+r.W, x(s.to)); s.to > s.from && x1 > x0 {
			p.Fill(ui.Rect{X: x0, Y: r.Y, W: max(1, x1-x0), H: r.H}, t.Accent.Alpha(0.18), 0)
		}
		mid, half := r.Y+r.H/2, r.H/2
		rest := t.TextMuted.Alpha(0.55)
		for col := float32(0); col < r.W; col++ {
			f0, f1 := at(col, r.W), at(col+1, r.W)
			b0 := min(n-1, int(f0*float64(n)))
			b1 := max(b0+1, min(n, int(f1*float64(n))))
			lo, hi := peaks[b0].Lo, peaks[b0].Hi
			for _, pk := range peaks[b0+1 : b1] {
				lo, hi = min(lo, pk.Lo), max(hi, pk.Hi)
			}
			color := rest
			if (f0+f1)/2 < s.Playhead {
				color = t.Accent
			}
			y0, y1 := mid-hi*half, mid-lo*half
			p.Fill(ui.Rect{X: r.X + col, Y: y0, W: 1, H: max(1, y1-y0)}, color, 0)
		}
		if px := x(s.Playhead); px >= r.X && px <= r.X+r.W {
			p.Fill(ui.Rect{X: px - 1, Y: r.Y, W: 2, H: r.H}, t.Text, 1)
		}
	})
	return view
}
