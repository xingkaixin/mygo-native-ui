// Package keyframes is the keyframe editor of a motion tool, as the
// timelines of After Effects and Rive: a row for each property that
// animates, its name and its value at the playhead, and its keyframes
// along a ruler of the time, shaped by how the value leaves them, a
// diamond for linear, a circle for eased and a square for a hold, with a
// bar to the next where the value changes. Track.At gives a property's
// value at any time, to play it.
//
// A click on the ruler, or a drag along it, moves the playhead, which
// snaps to the keyframes; Left and Right step it a frame. A click on a
// keyframe chooses it, and Shift-click adds it to the keyframes chosen or
// takes it out; dragging moves those chosen in time, by whole frames, no
// further than the keyframes beside them. A double click on a row adds a
// keyframe there with the value the property has then. A keyframe's
// context menu sets how the value leaves it; Delete removes the
// keyframes chosen. Changed reports an edit.
//
//	keyframes.View(c, &app.keys, app.tracks, &app.playhead, 3*time.Second, 30)
//	box.Translate(float32(app.tracks[0].At(app.playhead)), 0)
package keyframes

import (
	"fmt"
	"math"
	"slices"
	"strconv"
	"time"

	"github.com/egoist/mygo/ui"
)

// Ease is how the value leaves a keyframe for the next, as CSS's
// animation-timing-function on a keyframe.
type Ease uint8

const (
	Linear Ease = iota
	EaseIn
	EaseOut
	EaseInOut
	// Hold keeps the value until the next keyframe, then jumps.
	Hold
)

var eases = []struct {
	name string
	fn   ui.Easing
}{
	{"Linear", ui.Linear},
	{"Ease In", ui.EaseIn},
	{"Ease Out", ui.EaseOut},
	{"Ease In and Out", ui.EaseInOut},
	{"Hold", func(float32) float32 { return 0 }},
}

// Key is a keyframe: the time it is at, the value it sets, and how the
// value leaves it.
type Key struct {
	At    time.Duration
	Value float64
	Ease  Ease
}

// Track is a property that animates: its name, its keyframes in the
// order of their times, and how it writes its value, nil for up to two
// decimals.
type Track struct {
	Name   string
	Keys   []Key
	Format func(float64) string
}

// At returns the value of the track at time d: that of its first
// keyframe before it, of its last after it, and between two the change
// from one to the next as the first's ease shapes it.
func (tr Track) At(d time.Duration) float64 {
	ks := tr.Keys
	if len(ks) == 0 {
		return 0
	}
	for i := 1; i < len(ks); i++ {
		a, b := ks[i-1], ks[i]
		if d < b.At {
			if d <= a.At || b.At == a.At {
				return a.Value
			}
			f := eases[a.Ease].fn(float32(d-a.At) / float32(b.At-a.At))
			return a.Value + float64(f)*(b.Value-a.Value)
		}
	}
	if d < ks[0].At {
		return ks[0].Value
	}
	return ks[len(ks)-1].Value
}

func (tr Track) text(v float64) string {
	if tr.Format != nil {
		return tr.Format(v)
	}
	return strconv.FormatFloat(math.Round(v*100)/100, 'f', -1, 64)
}

// ref is a keyframe by its track's index and its own.
type ref struct{ track, key int }

// State is what an editor keeps: the keyframes chosen and the drag of
// them going on. Keep it in the app's state, one for each editor.
type State struct {
	chosen  map[ref]bool
	last    ref
	drag    bool
	dx      float32
	from    map[ref]time.Duration
	changed bool
}

// Chosen returns the keyframe chosen last, by its track's index and its
// own, while it is chosen.
func (s *State) Chosen() (track, key int, ok bool) {
	return s.last.track, s.last.key, s.chosen[s.last]
}

func (s *State) only(r ref) {
	s.chosen = map[ref]bool{r: true}
	s.last = r
}

// View creates the editor of the keyframes of tracks, which it edits in
// place, over length, in frames of fps a second, the playhead at
// *playhead.
func View(c *ui.Context, s *State, tracks []Track, playhead *time.Duration, length time.Duration, fps int) ui.Element {
	t := c.Theme()
	if fps <= 0 {
		fps = 30
	}
	fr := rate(fps)
	length = max(length, fr.time(1))
	rowH, ruler, nameW, d := t.Space(9), t.Space(7), t.Space(40), t.Space(3.5)
	root := ui.Row(c).AlignItems(ui.Start).Border(1, t.Border).Radius(t.Space(1.5)).Clip().Focusable().FocusRing(false).
		Label("Keyframes")
	if s.changed {
		root.SetChanged()
		s.changed = false
	}
	switch {
	case root.Shortcut(0, ui.KeyBackspace), root.Shortcut(0, ui.KeyDelete):
		for i := range tracks {
			var keep []Key
			for k, key := range tracks[i].Keys {
				if !s.chosen[ref{i, k}] {
					keep = append(keep, key)
				}
			}
			tracks[i].Keys = keep
		}
		if len(s.chosen) > 0 {
			root.SetChanged()
		}
		s.chosen = nil
	case root.Shortcut(0, ui.KeyLeft):
		*playhead = fr.time(max(0, fr.frames(*playhead)-1))
	case root.Shortcut(0, ui.KeyRight):
		*playhead = min(length, fr.time(fr.frames(*playhead)+1))
	}
	root.Children(func() {
		ui.Column(c).Width(nameW).Shrink(0).Background(t.Surface).Children(func() {
			ui.Row(c).Height(ruler).PaddingX(t.Space(3)).AlignItems(ui.Center).BorderWidth(0, 0, 1, 0).BorderColor(t.Border).Children(func() {
				ui.Text(c, clock(*playhead, fr)).Font("monospace").FontSize(t.FontSize - 1)
			})
			for i, tr := range tracks {
				ui.Row(c.Key(i)).Height(rowH).PaddingX(t.Space(3)).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
					ui.Text(c, tr.Name).SingleLine().Grow(1).Shrink(1)
					ui.Text(c, tr.text(tr.At(*playhead))).FontFeatures("tnum").TextColor(t.TextMuted).SingleLine()
				})
			}
		})
		ui.Divider(c).Vertical().AlignSelf(ui.Stretch)
		lanes := ui.Box(c).Grow(1).Shrink(1).Height(ruler + float32(len(tracks))*rowH)
		w := lanes.LayoutBounds().W
		if w == 0 {
			// Before the first layout there is no width to lay the time
			// along; the frame after has it.
			w = 480
			c.Invalidate()
		}
		pad := d
		scale := (w - 2*pad) / float32(length.Seconds())
		x := func(at time.Duration) float32 { return pad + float32(at.Seconds())*scale }
		at := func(px float32) time.Duration {
			return max(0, min(length, fr.time(int(math.Round(float64((px-pad)/scale)*float64(fps))))))
		}
		lanes.Draw(func(p *ui.Painter, r ui.Rect) {
			for i, tr := range tracks {
				y := r.Y + ruler + float32(i)*rowH
				if i%2 == 1 {
					p.Fill(ui.Rect{X: r.X, Y: y, W: r.W, H: rowH}, t.Text.Alpha(0.025), 0)
				}
				for k := 1; k < len(tr.Keys); k++ {
					a, b := tr.Keys[k-1], tr.Keys[k]
					color := t.Accent.Alpha(0.45)
					if a.Value == b.Value || a.Ease == Hold {
						color = t.Text.Alpha(0.12)
					}
					p.Fill(ui.Rect{X: r.X + x(a.At), Y: y + rowH/2 - 1.5, W: x(b.At) - x(a.At), H: 3}, color, 1.5)
				}
			}
			grid(p, t, r, length, fr, x, ruler)
		})
		lanes.DrawOver(func(p *ui.Painter, r ui.Rect) {
			px := r.X + x(*playhead)
			p.Line(px, r.Y, px, r.Y+r.H, 1.5, t.Danger)
			var knob ui.Path
			knob.MoveTo(px-5, r.Y+2).LineTo(px+5, r.Y+2).LineTo(px+5, r.Y+ruler-7).LineTo(px, r.Y+ruler-2).LineTo(px-5, r.Y+ruler-7).Close()
			p.FillPath(&knob, t.Danger)
		})
		lanes.Children(func() {
			scrub := ui.Box(c).Absolute().Left(0).Top(0).Size(w, ruler).Label("Playhead").Cursor(ui.CursorResizeEW)
			if scrub.Pressed() {
				if px, _, ok := scrub.PointerPosition(); ok {
					to := at(px)
					// The playhead snaps to a keyframe within a few DIPs.
					for _, tr := range tracks {
						for _, k := range tr.Keys {
							if math.Abs(float64(x(k.At)-px)) < 6 {
								to = k.At
							}
						}
					}
					*playhead = to
				}
			}
			pressed := false
			for i := range tracks {
				row := ui.Box(c.Key(i)).Absolute().Left(0).Top(ruler+float32(i)*rowH).Size(w, rowH).Label(tracks[i].Name)
				if row.DoubleClicked() {
					if px, _, ok := row.PointerPosition(); ok {
						s.add(tracks, i, at(px))
					}
				} else if row.Clicked() && c.Modifiers()&ui.Shift == 0 {
					s.chosen = nil
				}
				for k := range tracks[i].Keys {
					r := ref{i, k}
					key := tracks[i].Keys[k]
					el := ui.Box(c.Key(r)).Absolute().Left(x(key.At)-d/2).Top(ruler+float32(i)*rowH+(rowH-d)/2).Size(d, d).
						Cursor(ui.CursorPointer).Label(fmt.Sprintf("%s at %s", tracks[i].Name, clock(key.At, fr)))
					if el.Pressed() {
						pressed = true
						if !s.drag {
							s.press(r, tracks, c.Modifiers()&ui.Shift != 0)
						}
					}
					if dx, _, ok := el.Dragged(); ok && s.drag {
						s.dx += dx
						s.move(tracks, length, fr, scale)
					}
					el.ContextMenu(func(m *ui.Menu) {
						for e, it := range eases {
							if m.Item(it.name).Checked(tracks[i].Keys[k].Ease == Ease(e)).Chosen() {
								tracks[i].Keys[k].Ease = Ease(e)
								s.changed = true
							}
						}
						m.Separator()
						if m.Item("Delete").Chosen() {
							tracks[i].Keys = slices.Delete(tracks[i].Keys, k, k+1)
							s.chosen, s.changed = nil, true
						}
					})
					on := s.chosen[r]
					el.Draw(func(p *ui.Painter, b ui.Rect) { shape(p, t, b, key.Ease, on) })
				}
			}
			if s.drag && !pressed {
				s.drag = false
			}
		})
	})
	return root
}

// press starts a press on keyframe r: it chooses r alone, unless shift
// adds it to those chosen or takes it out, and the drag of them.
func (s *State) press(r ref, tracks []Track, shift bool) {
	switch {
	case shift && s.chosen[r]:
		delete(s.chosen, r)
	case shift:
		if s.chosen == nil {
			s.chosen = map[ref]bool{}
		}
		s.chosen[r], s.last = true, r
	case !s.chosen[r]:
		s.only(r)
	default:
		s.last = r
	}
	s.drag, s.dx = true, 0
	s.from = map[ref]time.Duration{}
	for q := range s.chosen {
		s.from[q] = tracks[q.track].Keys[q.key].At
	}
}

// move moves the keyframes chosen by the drag so far, in whole frames,
// within the length and short of the keyframes not chosen beside them.
func (s *State) move(tracks []Track, length time.Duration, fr rate, scale float32) {
	n := int(math.Round(float64(s.dx/scale) * float64(fr)))
	lo, hi := math.MinInt, math.MaxInt
	for r, from := range s.from {
		f := fr.frames(from)
		lo, hi = max(lo, -f), min(hi, fr.frames(length)-f)
		keys := tracks[r.track].Keys
		for j := r.key - 1; j >= 0; j-- {
			if !s.chosen[ref{r.track, j}] {
				lo = max(lo, fr.frames(keys[j].At)+1-f)
				break
			}
		}
		for j := r.key + 1; j < len(keys); j++ {
			if !s.chosen[ref{r.track, j}] {
				hi = min(hi, fr.frames(keys[j].At)-1-f)
				break
			}
		}
	}
	n = max(lo, min(hi, n))
	for r, from := range s.from {
		if at := fr.time(fr.frames(from) + n); tracks[r.track].Keys[r.key].At != at {
			tracks[r.track].Keys[r.key].At = at
			s.changed = true
		}
	}
}

// add adds a keyframe to track i at t, of the value the track has there,
// and chooses it, unless one is there already.
func (s *State) add(tracks []Track, i int, t time.Duration) {
	keys := tracks[i].Keys
	k, found := slices.BinarySearchFunc(keys, t, func(k Key, t time.Duration) int { return int(k.At - t) })
	if found {
		return
	}
	ease := Linear
	if k > 0 {
		ease = keys[k-1].Ease
	}
	tracks[i].Keys = slices.Insert(keys, k, Key{At: t, Value: tracks[i].At(t), Ease: ease})
	s.only(ref{i, k})
	s.changed = true
}

// shape draws a keyframe in b: a diamond for linear, a circle eased and
// a square held, filled in the accent color when chosen.
func shape(p *ui.Painter, t *ui.Theme, b ui.Rect, e Ease, on bool) {
	fill, edge := t.Background, t.Text
	if on {
		fill, edge = t.Accent, t.Accent
	}
	cx, cy, k := b.X+b.W/2, b.Y+b.H/2, b.W/2
	var path ui.Path
	switch e {
	case Linear:
		path.MoveTo(cx, cy-k).LineTo(cx+k, cy).LineTo(cx, cy+k).LineTo(cx-k, cy).Close()
	case Hold:
		k *= 0.8
		path.MoveTo(cx-k, cy-k).LineTo(cx+k, cy-k).LineTo(cx+k, cy+k).LineTo(cx-k, cy+k).Close()
	default:
		path.Circle(cx, cy, k*0.85)
	}
	p.FillPath(&path, fill)
	p.StrokePath(&path, 1.25, edge)
}

// grid draws the ruler of times along the top of r, a label at least 56
// DIPs from the next and a tick at each frame when they are far enough
// apart, and a line down the lanes at each label.
func grid(p *ui.Painter, t *ui.Theme, r ui.Rect, length time.Duration, fr rate, x func(time.Duration) float32, ruler float32) {
	step := 10 * int(fr)
	for _, n := range []int{1, 2, 5, 10, int(fr) / 2, int(fr), 2 * int(fr), 5 * int(fr)} {
		if n > 0 && x(fr.time(n))-x(0) >= 56 {
			step = n
			break
		}
	}
	p.Fill(ui.Rect{X: r.X, Y: r.Y, W: r.W, H: ruler}, t.Surface, 0)
	ticks := x(fr.time(1))-x(0) >= 4
	for n := 0; n <= fr.frames(length); n++ {
		px := r.X + x(fr.time(n))
		switch {
		case n%step == 0:
			p.Line(px, r.Y+ruler*0.6, px, r.Y+ruler, 1, t.TextMuted)
			p.Line(px, r.Y+ruler, px, r.Y+r.H, 1, t.Text.Alpha(0.05))
			label := ui.Span{Text: clock(fr.time(n), fr), Size: t.FontSize - 3, Color: t.TextMuted}
			lx := px + 3
			if w, _ := p.MeasureText(0, label); lx+w > r.X+r.W {
				lx = px - 3 - w
			}
			p.RichText(lx, r.Y+2, 0, label)
		case ticks:
			p.Line(px, r.Y+ruler*0.8, px, r.Y+ruler, 1, t.TextMuted.Alpha(0.5))
		}
	}
	p.Line(r.X, r.Y+ruler, r.X+r.W, r.Y+ruler, 1, t.Border)
}

// rate is frames a second, converting times to frames and back.
type rate int

func (fr rate) frames(d time.Duration) int { return int(math.Round(d.Seconds() * float64(fr))) }

func (fr rate) time(n int) time.Duration {
	return time.Duration(math.Round(float64(n) * float64(time.Second) / float64(fr)))
}

// clock writes d as seconds and frames: 1:12 is 1 second and 12 frames.
func clock(d time.Duration, fr rate) string {
	n := fr.frames(d)
	return fmt.Sprintf("%d:%02d", n/int(fr), n%int(fr))
}
