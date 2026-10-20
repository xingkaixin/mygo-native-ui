// Package timeline is the timeline of a video editor: a lane for each
// track, the clips on them as long as they play, a ruler of the time
// above, and the playhead across them all. Dragging a clip moves it, to
// another track too, and dragging its ends trims it, no further than its
// source goes; its edges snap to those of other clips and to the
// playhead, and a clip dropped over another goes back. A click on the
// ruler moves the playhead; S splits the clip chosen at it, and Delete
// removes it. The slider zooms. Changed reports an edit.
//
//	timeline.View(c, &app.edit, tracks, &app.clips, &app.playhead).Grow(1)
package timeline

import (
	"fmt"
	"math"
	"slices"
	"time"

	"github.com/egoist/mygo/ui"
)

// Clip is a clip on a track: its name, its track's index, where it starts
// and how long it plays, where in its source it starts, and how long the
// source is, 0 for a source with no end, as a title.
type Clip struct {
	Name   string
	Track  int
	Start  time.Duration
	Length time.Duration
	In     time.Duration
	Source time.Duration
}

// End returns where the clip ends.
func (c Clip) End() time.Duration { return c.Start + c.Length }

// State is what a timeline keeps: its zoom, the clip chosen and the drag
// going on. Keep it in the app's state, one for each timeline.
type State struct {
	// zoom is DIPs a second, 0 before the first frame sets it.
	zoom float64
	// chosen is the index of the clip chosen, plus 1, so that the zero
	// State chooses none.
	chosen  int
	drag    drag
	scroll  ui.ScrollState
	changed bool
}

type part uint8

const (
	body part = iota
	head
	tail
)

// drag is a drag of part of a clip, by dx and dy DIPs so far, from the
// clip as it started; snap is where an edge snapped, -1 for nowhere.
type drag struct {
	on     bool
	clip   int
	part   part
	dx, dy float32
	from   Clip
	snap   time.Duration
}

// Chosen returns the index of the clip chosen.
func (s *State) Chosen() (int, bool) { return s.chosen - 1, s.chosen > 0 }

const minLength = 100 * time.Millisecond

// Clock writes d as minutes, seconds and hundredths: 1:04.25.
func Clock(d time.Duration) string {
	cs := int(d.Round(10*time.Millisecond) / (10 * time.Millisecond))
	return fmt.Sprintf("%d:%02d.%02d", cs/6000, cs/100%60, cs%100)
}

// split splits clip i at t, when t is inside it.
func split(clips *[]Clip, i int, t time.Duration) bool {
	cl := (*clips)[i]
	if t <= cl.Start+minLength/2 || t >= cl.End()-minLength/2 {
		return false
	}
	left, right := cl, cl
	left.Length = t - cl.Start
	right.Start, right.In, right.Length = t, cl.In+(t-cl.Start), cl.End()-t
	(*clips)[i] = left
	*clips = slices.Insert(*clips, i+1, right)
	return true
}

func overlaps(clips []Clip, i int) bool {
	a := clips[i]
	for j, b := range clips {
		if j != i && b.Track == a.Track && a.Start < b.End() && b.Start < a.End() {
			return true
		}
	}
	return false
}

// View creates the timeline of *clips on tracks, the playhead at
// *playhead.
func View(c *ui.Context, s *State, tracks []string, clips *[]Clip, playhead *time.Duration) ui.Element {
	t := c.Theme()
	cs := *clips
	var end time.Duration
	for _, cl := range cs {
		end = max(end, cl.End())
	}
	end = max(end, *playhead) + 20*time.Second
	laneH, ruler, nameW := t.Space(14), t.Space(9), t.Space(30)
	root := ui.Column(c).Gap(t.Space(2))
	if s.changed {
		root.SetChanged()
		s.changed = false
	}
	if i, ok := s.Chosen(); ok && i < len(cs) {
		switch {
		case root.Shortcut(0, ui.KeyS):
			if split(clips, i, *playhead) {
				s.changed = true
			}
		case root.Shortcut(0, ui.KeyBackspace), root.Shortcut(0, ui.KeyDelete):
			*clips = slices.Delete(*clips, i, i+1)
			s.chosen, s.changed = 0, true
		}
		cs = *clips
	}
	root.Children(func() {
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, Clock(*playhead)).Font("monospace").FontSize(t.FontSize + 2)
			ui.Box(c).Grow(1)
			ui.Text(c, "Zoom").FontSize(t.FontSize - 1).TextColor(t.TextMuted)
			z := math.Log2(max(s.zoom, 1))
			if ui.Slider(c, &z, 2, 9).Width(t.Space(40)).Label("Zoom").Changed() {
				s.zoom = math.Exp2(z)
			}
		})
		ui.Row(c).AlignSelf(ui.Stretch).AlignItems(ui.Start).Border(1, t.Border).Radius(t.Space(1.5)).Clip().Children(func() {
			ui.Column(c).Width(nameW).Shrink(0).Background(t.Surface).Children(func() {
				ui.Box(c).Height(ruler)
				for i, name := range tracks {
					ui.Row(c.Key(i)).Height(laneH).PaddingX(t.Space(3)).AlignItems(ui.Center).Children(func() {
						ui.Box(c).Size(t.Space(2), t.Space(2)).Radius(t.Space(1)).Background(hue(i))
						ui.Text(c, name).FontSize(t.FontSize - 1).PaddingStart(t.Space(2)).SingleLine().Shrink(1)
					})
				}
			})
			ui.Divider(c).Vertical().AlignSelf(ui.Stretch)
			sc := ui.ScrollHorizontal(c).Grow(1).Shrink(1).TrackScroll(&s.scroll)
			if w := sc.LayoutBounds().W; s.zoom == 0 && w > 0 {
				s.zoom = float64(w) / end.Seconds()
			}
			zoom := float32(s.zoom)
			if zoom == 0 {
				// Before the first layout there is no width to fit the cut to.
				zoom = 40
			}
			x := func(d time.Duration) float32 { return float32(d.Seconds()) * zoom }
			sc.Children(func() {
				box := ui.Box(c).Size(x(end), ruler+float32(len(tracks))*laneH)
				box.Draw(func(p *ui.Painter, r ui.Rect) {
					for i := range tracks {
						y := r.Y + ruler + float32(i)*laneH
						if i%2 == 1 {
							p.Fill(ui.Rect{X: r.X, Y: y, W: r.W, H: laneH}, t.Text.Alpha(0.025), 0)
						}
						p.Line(r.X, y, r.X+r.W, y, 1, t.Border)
					}
					grid(p, t, r, end, zoom, ruler)
				})
				box.DrawOver(func(p *ui.Painter, r ui.Rect) {
					if s.drag.on && s.drag.snap >= 0 {
						sx := r.X + x(s.drag.snap)
						p.Line(sx, r.Y+ruler, sx, r.Y+r.H, 1.5, t.Warning)
					}
					px := r.X + x(*playhead)
					p.Line(px, r.Y, px, r.Y+r.H, 2, t.Danger)
					var knob ui.Path
					knob.MoveTo(px-6, r.Y).LineTo(px+6, r.Y).LineTo(px+6, r.Y+ruler-8).LineTo(px, r.Y+ruler-2).LineTo(px-6, r.Y+ruler-8).Close()
					p.FillPath(&knob, t.Danger)
				})
				box.Children(func() {
					scrub := ui.Box(c).Absolute().Left(0).Top(0).Size(x(end), ruler).Label("Playhead").Focusable()
					if scrub.Pressed() {
						if px, _, ok := scrub.PointerPosition(); ok {
							*playhead = max(0, time.Duration(float64(px/zoom)*float64(time.Second)).Round(time.Millisecond))
						}
					}
					for i := range cs {
						s.clip(c.Key(i), clips, i, len(tracks), zoom, ruler, laneH, *playhead)
					}
				})
			})
		})
	})
	return root
}

// clip shows clip i: its body moves it, its ends trim it.
func (s *State) clip(c *ui.Context, clips *[]Clip, i, tracks int, zoom, ruler, laneH float32, playhead time.Duration) {
	t := c.Theme()
	cl := (*clips)[i]
	x := float32(cl.Start.Seconds()) * zoom
	w := max(4, float32(cl.Length.Seconds())*zoom)
	color := hue(cl.Track)
	el := ui.Box(c).Absolute().Left(x).Top(ruler+float32(cl.Track)*laneH+t.Space(1.5)).Size(w, laneH-t.Space(3)).
		Radius(t.Space(1.5)).Background(color.Alpha(0.85)).Clip().Cursor(ui.CursorGrab).Role(ui.RoleButton).Label(cl.Name).Focusable()
	if s.chosen == i+1 {
		el.Border(2, t.Text)
	}
	if el.Pressed() {
		s.chosen = i + 1
	}
	if dx, dy, ok := el.Dragged(); ok {
		s.move(clips, i, body, dx, dy, zoom, laneH, tracks, playhead)
	}
	s.end(clips, el)
	el.Children(func() {
		ui.Column(c).Padding(t.Space(1.5), t.Space(2.5)).Gap(t.Space(0.5)).Children(func() {
			white := ui.RGB(255, 255, 255)
			ui.Text(c, cl.Name).FontSize(t.FontSize - 1).Bold().TextColor(white).SingleLine()
			ui.Text(c, Clock(cl.Length)).FontSize(t.FontSize - 3).FontFeatures("tnum").TextColor(white.Alpha(0.8)).SingleLine()
		})
		grip := min(t.Space(2.5), w/3)
		for k, pt := range []part{head, tail} {
			e := ui.Box(c.Key(k)).Absolute().Top(0).Size(grip, laneH-t.Space(3)).Cursor(ui.CursorResizeEW)
			if pt == head {
				e.Left(0).Label("Start of " + cl.Name)
			} else {
				e.Left(w - grip).Label("End of " + cl.Name)
			}
			if e.Hovered() || s.drag.on && s.drag.clip == i && s.drag.part == pt {
				e.Background(ui.RGB(255, 255, 255).Alpha(0.35))
			}
			if e.Pressed() {
				s.chosen = i + 1
				el.Focus()
			}
			if dx, dy, ok := e.Dragged(); ok {
				s.move(clips, i, pt, dx, dy, zoom, laneH, tracks, playhead)
			}
			s.end(clips, e)
		}
	})
}

// move moves part of clip i by dx and dy more, its edges snapping within
// 8 DIPs of those of other clips and of the playhead.
func (s *State) move(clips *[]Clip, i int, pt part, dx, dy, zoom, laneH float32, tracks int, playhead time.Duration) {
	if dx == 0 && dy == 0 {
		// A click reports a drag of nothing, and no end to it.
		return
	}
	cs := *clips
	if !s.drag.on || s.drag.clip != i || s.drag.part != pt {
		s.drag = drag{on: true, clip: i, part: pt, from: cs[i]}
	}
	s.drag.dx += dx
	s.drag.dy += dy
	from := s.drag.from
	delta := time.Duration(float64(s.drag.dx) / float64(zoom) * float64(time.Second)).Round(time.Millisecond)
	edges := []time.Duration{0, playhead}
	for j, o := range cs {
		if j != i {
			edges = append(edges, o.Start, o.End())
		}
	}
	reach := time.Duration(8 / float64(zoom) * float64(time.Second))
	// nudge returns how far to move d to the edge nearest it within reach.
	nudge := func(d time.Duration) (time.Duration, bool) {
		best, ok := reach+1, false
		for _, e := range edges {
			if dd := e - d; abs(dd) < abs(best) && abs(dd) <= reach {
				best, ok = dd, true
			}
		}
		return best, ok
	}
	s.drag.snap = -1
	cl := &cs[i]
	switch pt {
	case body:
		start := max(0, from.Start+delta)
		a, okA := nudge(start)
		b, okB := nudge(start + from.Length)
		switch {
		case okA && (!okB || abs(a) <= abs(b)):
			start += a
			s.drag.snap = start
		case okB:
			start += b
			s.drag.snap = start + from.Length
		}
		cl.Start = max(0, start)
		cl.Track = max(0, min(tracks-1, from.Track+int(math.Round(float64(s.drag.dy/laneH)))))
	case head:
		start := from.Start + delta
		if d, ok := nudge(start); ok {
			start += d
			s.drag.snap = start
		}
		start = max(start, from.Start-from.In, 0)
		start = min(start, from.End()-minLength)
		cl.Start, cl.In, cl.Length = start, from.In+(start-from.Start), from.End()-start
	case tail:
		e := from.End() + delta
		if d, ok := nudge(e); ok {
			e += d
			s.drag.snap = e
		}
		length := max(minLength, e-from.Start)
		if from.Source > 0 {
			length = min(length, from.Source-from.In)
		}
		cl.Length = length
	}
	s.changed = true
}

// end ends the drag as e lets go, putting the clip back where it was when
// it overlaps another.
func (s *State) end(clips *[]Clip, e ui.Element) {
	if _, _, ok := e.DragEnded(); ok && s.drag.on {
		if s.drag.clip < len(*clips) && overlaps(*clips, s.drag.clip) {
			(*clips)[s.drag.clip] = s.drag.from
		}
		s.drag = drag{}
	}
}

func abs(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
}

// grid draws the ruler of times above the lanes of r, at zoom DIPs a
// second, a label at least 80 DIPs from the next.
func grid(p *ui.Painter, t *ui.Theme, r ui.Rect, end time.Duration, zoom, ruler float32) {
	steps := []float64{0.1, 0.25, 0.5, 1, 2, 5, 10, 15, 30, 60, 120, 300, 600}
	step := steps[len(steps)-1]
	for _, st := range steps {
		if float32(st)*zoom >= 80 {
			step = st
			break
		}
	}
	minor := step / 5
	p.Fill(ui.Rect{X: r.X, Y: r.Y, W: r.W, H: ruler}, t.Surface, 0)
	for k := 0; float64(k)*minor <= end.Seconds(); k++ {
		sec := float64(k) * minor
		x := r.X + float32(sec)*zoom
		if k%5 == 0 {
			p.Line(x, r.Y+ruler*0.45, x, r.Y+ruler, 1, t.TextMuted)
			p.Line(x, r.Y+ruler, x, r.Y+r.H, 1, t.Text.Alpha(0.06))
			label := Clock(time.Duration(sec * float64(time.Second)))
			if step >= 1 {
				label = label[:len(label)-3]
			}
			p.Text(x+3, r.Y+2, label, t.FontSize-3, t.TextMuted)
		} else {
			p.Line(x, r.Y+ruler*0.75, x, r.Y+ruler, 1, t.TextMuted.Alpha(0.6))
		}
	}
	p.Line(r.X, r.Y+ruler, r.X+r.W, r.Y+ruler, 1, t.Border)
}

// hue returns track k's color.
func hue(k int) ui.Color {
	return ui.Oklch(0.6, 0.13, float32((260+k*55)%360))
}
