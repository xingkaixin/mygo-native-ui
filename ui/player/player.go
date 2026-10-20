// Package player shows the transport controls of audio or video, as
// podcast apps and video sites do: a scrubber, the part played in the
// accent color, the stretches buffered shaded, and a gap where each
// chapter begins; under it, buttons that go back 10 seconds, play or
// pause, and go forward 10 seconds, the time played and the length, a
// menu of speeds from 0.5× to 2×, and the volume, with a button that
// mutes it.
//
// The pointer over the scrubber shows the time and the chapter under it,
// and a press or a drag there seeks. A click on the time shows the time
// remaining in place of the length, and another the length again.
// Anywhere in the window, unless the focused element takes the key, Space
// plays and pauses, Left and Right go back and forward 5 seconds, Up and
// Down turn the volume up and down, and M mutes.
//
//	a.position, a.playing = a.track.Position(), a.track.Playing()
//	ctl := player.Controls(c, &a.controls, &a.position, &a.playing, player.Media{
//		Duration: a.track.Duration(),
//		Chapters: a.chapters,
//	})
//	if at, ok := ctl.Seeked(); ok {
//		a.track.Seek(at)
//	}
//	if a.playing != a.track.Playing() {
//		... // play or pause a.track
//	}
//	a.track.SetVolume(a.controls.Volume()).SetSpeed(a.controls.Speed())
//
// The app keeps the position and whether it plays, and moves the position
// as it plays, asking for frames while it does. The controls set both as
// the user plays, pauses and seeks, and report a seek in the frame it
// happens, for the app to move its player there.
package player

import (
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/egoist/mygo/ui"
)

// Chapter is a part of the media, from Start to the next chapter's start
// or the end. Give chapters in order.
type Chapter struct {
	Title string
	Start time.Duration
}

// Range is a stretch of the media, as one buffered.
type Range struct{ From, To time.Duration }

// Media is what plays: how long it is, its chapters, and the stretches of
// it buffered, for media that streams.
type Media struct {
	Duration time.Duration
	Chapters []Chapter
	Buffered []Range
}

// State is what controls keep: the volume and whether it is muted, the
// speed, whether the time remaining shows, and the seek of a drag. Keep it
// in the app's state, one for each player. The zero State plays at full
// volume and at normal speed.
type State struct {
	// cut is how far the volume is turned down from full, and speed the
	// speed, 0 for 1, so that the zero State plays at full volume and at
	// normal speed.
	cut, speed       float64
	muted, remaining bool
	// dragging is whether the scrubber is pressed, and at where it last
	// seeked.
	dragging bool
	at       time.Duration
}

// Volume returns the volume to play at, a gain from 0 to 1: 0 while
// muted.
func (s *State) Volume() float64 {
	if s.muted {
		return 0
	}
	return 1 - s.cut
}

// Speed returns the speed to play at, from 0.5 to 2.
func (s *State) Speed() float64 {
	if s.speed == 0 {
		return 1
	}
	return s.speed
}

// setVolume sets the volume to v, unmuted, in hundredths, so that steps
// down come to 0 rather than near it.
func (s *State) setVolume(v float64) {
	s.cut, s.muted = 1-math.Round(max(0, min(1, v))*100)/100, false
}

// toggleMute mutes, or unmutes, back to half the volume when it was
// turned all the way down, which would be silent still.
func (s *State) toggleMute() {
	switch {
	case s.Volume() > 0:
		s.muted = true
	case s.cut >= 1:
		s.setVolume(0.5)
	default:
		s.muted = false
	}
}

// Events are what the user did with controls in a frame. The zero Events
// report nothing.
type Events struct {
	seek   time.Duration
	seeked bool
}

// Seeked returns where the user moved the position, already in
// *position: move the player there in this frame.
func (e Events) Seeked() (time.Duration, bool) { return e.seek, e.seeked }

var speeds = []float64{0.5, 0.75, 1, 1.25, 1.5, 1.75, 2}

// Controls creates the controls of media, which plays at *position, and
// on while *playing. Give it a width.
func Controls(c *ui.Context, s *State, position *time.Duration, playing *bool, media Media) Events {
	t := c.Theme()
	var ev Events
	seek := func(to time.Duration) {
		to = max(0, min(media.Duration, to))
		*position = to
		ev = Events{seek: to, seeked: true}
	}
	switch {
	case c.Shortcut(0, ui.KeySpace):
		*playing = !*playing
	case c.Shortcut(0, ui.KeyLeft):
		seek(*position - 5*time.Second)
	case c.Shortcut(0, ui.KeyRight):
		seek(*position + 5*time.Second)
	case c.Shortcut(0, ui.KeyUp):
		s.setVolume(1 - s.cut + 0.1)
	case c.Shortcut(0, ui.KeyDown):
		s.setVolume(1 - s.cut - 0.1)
	case c.Shortcut(0, ui.KeyM):
		s.toggleMute()
	}
	long := media.Duration >= time.Hour

	ui.Column(c).Gap(t.Space(1)).Children(func() {
		scrubber(c, s, position, media, seek)
		ui.Row(c).Gap(t.Space(1)).AlignItems(ui.Center).Children(func() {
			if button(c, "Back 10 seconds", func(p *ui.Painter, x, y float32, color ui.Color) {
				skip(p, x, y, t.Space(5.5), color, false)
			}).Clicked() {
				seek(*position - 10*time.Second)
			}
			if play(c, *playing).Clicked() {
				*playing = !*playing
			}
			if button(c, "Forward 10 seconds", func(p *ui.Painter, x, y float32, color ui.Color) {
				skip(p, x, y, t.Space(5.5), color, true)
			}).Clicked() {
				seek(*position + 10*time.Second)
			}

			clockButton(c, s, *position, media.Duration, long)
			ui.Spacer(c)

			speed := ui.ButtonBase(c).Height(t.Space(9)).PaddingX(t.Space(2.5)).Radius(t.Space(2)).
				Label("Speed").Value(times(s.Speed())).Tooltip("Speed")
			face(speed, t)
			speed.Menu(func(m *ui.Menu) {
				for _, v := range speeds {
					name := times(v)
					if v == 1 {
						name = "Normal"
					}
					if m.Item(name).Checked(v == s.Speed()).Chosen() {
						s.speed = v
					}
				}
			})
			speed.Children(func() { ui.Text(c, times(s.Speed())).FontFeatures("tnum") })

			label := "Mute"
			if s.Volume() == 0 {
				label = "Unmute"
			}
			if button(c, label, func(p *ui.Painter, x, y float32, color ui.Color) {
				speaker(p, x, y, t.Space(5), color, s.Volume())
			}).Clicked() {
				s.toggleMute()
			}
			volume(c, s)
		})
	})
	return ev
}

// scrubber creates the bar that shows where media plays, at *position,
// and seeks where pressed and dragged.
func scrubber(c *ui.Context, s *State, position *time.Duration, media Media, seek func(time.Duration)) {
	t := c.Theme()
	h, dur := t.Space(6), media.Duration
	long := dur >= time.Hour
	bar := ui.Box(c).Height(h).Focusable().Role(ui.RoleSlider).Label("Seek")
	w := bar.LayoutBounds().W
	at := func(x, w float32) time.Duration {
		return time.Duration(float64(max(0, min(1, x/max(1, w)))) * float64(dur))
	}
	px, _, over := bar.PointerPosition()
	_, _, pressed := bar.Dragged()
	if pressed {
		to := at(px, w)
		if !s.dragging || to != s.at {
			seek(to)
		}
		*position, s.dragging, s.at = to, true, to
	} else {
		s.dragging = false
	}
	pos := *position
	bar.Range(0, dur.Seconds(), pos.Seconds()).Value(clock(pos, long) + " of " + clock(dur, long))
	lit := over || pressed

	if lit {
		hx := max(0, min(w, px))
		hover := at(hx, w)
		bar.Children(func() {
			anchor := ui.Box(c).Absolute().PassThrough().Left(hx).Top(0).Size(1, h)
			tip(c, anchor, chapterAt(media.Chapters, hover), clock(hover, long))
		})
	}

	bar.Draw(func(p *ui.Painter, r ui.Rect) {
		x := func(d time.Duration) float32 {
			if dur <= 0 {
				return r.X
			}
			return r.X + r.W*float32(max(0, min(1, float64(d)/float64(dur))))
		}
		th := t.Space(1)
		if lit {
			th = t.Space(1.5)
		}
		y := r.Y + (r.H-th)/2
		gap := t.Space(0.5)
		edges := []float32{r.X}
		for _, ch := range media.Chapters {
			if ch.Start > 0 && ch.Start < dur {
				edges = append(edges, x(ch.Start))
			}
		}
		edges = append(edges, r.X+r.W)
		for i := range len(edges) - 1 {
			a, b := edges[i], edges[i+1]
			if i > 0 {
				a += gap / 2
			}
			if i < len(edges)-2 {
				b -= gap / 2
			}
			if b <= a {
				continue
			}
			seg := ui.Rect{X: a, Y: y, W: b - a, H: th}
			p.Clip(seg, th/2, func() {
				p.Fill(seg, t.Text.Alpha(0.12), 0)
				for _, rg := range media.Buffered {
					if x0, x1 := x(rg.From), x(rg.To); x1 > x0 {
						p.Fill(ui.Rect{X: x0, Y: y, W: x1 - x0, H: th}, t.Text.Alpha(0.2), 0)
					}
				}
				p.Fill(ui.Rect{X: r.X, Y: y, W: x(pos) - r.X, H: th}, t.Accent, 0)
			})
		}
		k := t.Space(3)
		if lit {
			k = t.Space(3.5)
		}
		knob(p, max(r.X+k/2, min(r.X+r.W-k/2, x(pos))), r.Y+r.H/2, k, t.Accent)
	})
}

// tip shows above anchor the chapter and the time under the pointer.
func tip(c *ui.Context, anchor ui.Element, chapter, at string) {
	t := c.Theme()
	ui.Overlay(c, func() {
		// The tip lets the pointer through to the scrubber under it, which
		// would otherwise lose the hover and the tip.
		card := ui.Column(c).Absolute().PassThrough().Role(ui.RoleTooltip).AlignItems(ui.Center).
			AttachTo(anchor, ui.AnchorTop, ui.AnchorBottom).Margin(0, 0, t.Space(1), 0).
			Padding(t.Space(1), t.Space(2)).Radius(t.Space(1.5)).Background(t.Text).
			Shadow(0, 2, 8, 0, ui.RGBA(0, 0, 0, 0.2))
		card.Children(func() {
			if chapter != "" {
				ui.Text(c, chapter).PassThrough().SingleLine().FontSize(t.FontSize - 1).Bold().TextColor(t.Background)
			}
			ui.Text(c, at).PassThrough().FontSize(t.FontSize - 1).FontFeatures("tnum").TextColor(t.Background)
		})
	})
}

// chapterAt returns the title of the chapter playing at d, "" before the
// first.
func chapterAt(chapters []Chapter, d time.Duration) string {
	title := ""
	for _, ch := range chapters {
		if ch.Start > d {
			break
		}
		title = ch.Title
	}
	return title
}

// clockButton creates the button showing the time played at pos, and the
// length dur or, as a click switches, the time remaining.
func clockButton(c *ui.Context, s *State, pos, dur time.Duration, long bool) {
	t := c.Theme()
	b := ui.ButtonBase(c).Height(t.Space(9)).PaddingX(t.Space(2)).Radius(t.Space(2))
	if b.Clicked() {
		s.remaining = !s.remaining
	}
	face(b, t)
	end := clock(dur, long)
	label := "Show time remaining"
	if s.remaining {
		end = "−" + clock(dur.Truncate(time.Second)-pos.Truncate(time.Second), long)
		label = "Show duration"
	}
	b.Label(label).Value(clock(pos, long) + " / " + end)
	b.Children(func() {
		ui.Text(c, clock(pos, long)).FontFeatures("tnum")
		ui.Text(c, " / "+end).FontFeatures("tnum").TextColor(t.TextMuted)
	})
}

// volume creates the slider of the volume, at 0 while muted, which a drag
// unmutes.
func volume(c *ui.Context, s *State) {
	t := c.Theme()
	v := s.Volume()
	k := t.Space(3)
	sl := ui.SliderBase(c, &v, 0, 1).Width(t.Space(24)).Height(t.Space(9)).PaddingX(k / 2).Label("Volume")
	if sl.Changed() {
		s.setVolume(v)
	}
	at := float32(s.Volume())
	if c.Direction() == ui.RightToLeft {
		at = 1 - at
	}
	sl.Draw(func(p *ui.Painter, r ui.Rect) {
		th := t.Space(1)
		y := r.Y + (r.H-th)/2
		kx := r.X + k/2 + (r.W-k)*at
		p.Fill(ui.Rect{X: r.X, Y: y, W: r.W, H: th}, t.Text.Alpha(0.12), th/2)
		fill := ui.Rect{X: r.X, Y: y, W: kx - r.X, H: th}
		if c.Direction() == ui.RightToLeft {
			fill = ui.Rect{X: kx, Y: y, W: r.X + r.W - kx, H: th}
		}
		p.Fill(fill, t.Accent, th/2)
		knob(p, kx, r.Y+r.H/2, k, t.Accent)
	})
}

// knob draws a knob k wide around (x, y).
func knob(p *ui.Painter, x, y, k float32, color ui.Color) {
	r := ui.Rect{X: x - k/2, Y: y - k/2, W: k, H: k}
	p.Shadow(r, k/2, 0, 1, 3, 0, ui.RGBA(0, 0, 0, 0.25))
	p.Fill(r, color, k/2)
}

// face gives button b the face of a control in a player: none until
// hovered.
func face(b ui.Element, t *ui.Theme) {
	switch {
	case b.Pressed():
		b.Background(t.SurfacePressed)
	case b.Hovered():
		b.Background(t.SurfaceHover)
	}
}

// button creates a button named label showing the icon that draw paints
// around its center.
func button(c *ui.Context, label string, draw func(p *ui.Painter, x, y float32, color ui.Color)) ui.Element {
	t := c.Theme()
	b := ui.ButtonBase(c).Size(t.Space(9), t.Space(9)).Radius(t.Space(2)).Label(label).Tooltip(label)
	face(b, t)
	b.Draw(func(p *ui.Painter, r ui.Rect) { draw(p, r.X+r.W/2, r.Y+r.H/2, t.Text) })
	return b
}

// play creates the button that plays, or pauses while playing.
func play(c *ui.Context, playing bool) ui.Element {
	t := c.Theme()
	label := "Play"
	if playing {
		label = "Pause"
	}
	d := t.Space(10)
	b := ui.ButtonBase(c).Size(d, d).Radius(d/2).Margin(0, t.Space(1)).Label(label).Tooltip(label).Background(t.Accent)
	switch {
	case b.Pressed():
		b.Background(t.AccentPressed)
	case b.Hovered():
		b.Background(t.AccentHover)
	}
	b.Draw(func(p *ui.Painter, r ui.Rect) {
		cx, cy, s := r.X+r.W/2, r.Y+r.H/2, t.Space(4)
		if playing {
			bw := s * 0.3
			p.Fill(ui.Rect{X: cx - s*0.4, Y: cy - s/2, W: bw, H: s}, t.AccentText, 1)
			p.Fill(ui.Rect{X: cx + s*0.4 - bw, Y: cy - s/2, W: bw, H: s}, t.AccentText, 1)
			return
		}
		// The triangle's middle sits right of its box's, to look centered.
		var tri ui.Path
		tri.MoveTo(cx-s*0.35, cy-s/2).LineTo(cx+s*0.55, cy).LineTo(cx-s*0.35, cy+s/2).Close()
		p.FillPath(&tri, t.AccentText)
	})
	return b
}

// arc returns points along the circle of radius r around (cx, cy), from
// angle a0 to a1 in radians, clockwise on the screen.
func arc(cx, cy, r float32, a0, a1 float64) [][2]float32 {
	n := max(2, int(math.Abs(a1-a0)/0.15))
	pts := make([][2]float32, n+1)
	for i := range pts {
		a := a0 + (a1-a0)*float64(i)/float64(n)
		pts[i] = [2]float32{cx + r*float32(math.Cos(a)), cy + r*float32(math.Sin(a))}
	}
	return pts
}

// skip draws the icon of going back, or forward, 10 seconds, s wide around
// (cx, cy): a circle, open where its arrow points, around a 10.
func skip(p *ui.Painter, cx, cy, s float32, color ui.Color, forward bool) {
	r := s / 2
	pts := arc(cx, cy, r, -math.Pi/2, math.Pi*7/6)
	k := s * 0.2
	head := [][2]float32{{cx - k, cy - r}, {cx + k*0.35, cy - r - k*0.8}, {cx + k*0.35, cy - r + k*0.8}}
	if forward {
		for _, q := range [][][2]float32{pts, head} {
			for i := range q {
				q[i][0] = 2*cx - q[i][0]
			}
		}
	}
	p.StrokePolyline(pts, max(1.5, s/12), color)
	var tri ui.Path
	tri.MoveTo(head[0][0], head[0][1]).LineTo(head[1][0], head[1][1]).LineTo(head[2][0], head[2][1]).Close()
	p.FillPath(&tri, color)
	ten := ui.Span{Text: "10", Size: s * 0.42, Weight: 700, Color: color, Features: "tnum"}
	w, h := p.MeasureText(0, ten)
	p.RichText(cx-w/2, cy-h/2+s*0.02, 0, ten)
}

// speaker draws the icon of the volume v, s wide around (cx, cy): a
// speaker with a wave for some and two for more, or crossed out at 0.
func speaker(p *ui.Painter, cx, cy, s float32, color ui.Color, v float64) {
	x := cx - s*0.12
	var body ui.Path
	body.MoveTo(x-s*0.38, cy-s*0.16).LineTo(x-s*0.16, cy-s*0.16).LineTo(x+s*0.1, cy-s*0.4).
		LineTo(x+s*0.1, cy+s*0.4).LineTo(x-s*0.16, cy+s*0.16).LineTo(x-s*0.38, cy+s*0.16).Close()
	p.FillPath(&body, color)
	lw := max(1.5, s/11)
	if v == 0 {
		d := s * 0.14
		ox := x + s*0.42
		p.StrokePolyline([][2]float32{{ox - d, cy - d}, {ox + d, cy + d}}, lw, color)
		p.StrokePolyline([][2]float32{{ox + d, cy - d}, {ox - d, cy + d}}, lw, color)
		return
	}
	p.StrokePolyline(arc(x+s*0.1, cy, s*0.2, -math.Pi/4, math.Pi/4), lw, color)
	if v > 0.5 {
		p.StrokePolyline(arc(x+s*0.1, cy, s*0.4, -math.Pi/4, math.Pi/4), lw, color)
	}
}

// clock writes d as 4:05, or as 1:04:05 when long, for media an hour long
// or longer.
func clock(d time.Duration, long bool) string {
	n := int(max(0, d) / time.Second)
	if long {
		return fmt.Sprintf("%d:%02d:%02d", n/3600, n/60%60, n%60)
	}
	return fmt.Sprintf("%d:%02d", n/60, n%60)
}

// times writes speed v as 1.25×.
func times(v float64) string { return strconv.FormatFloat(v, 'f', -1, 64) + "×" }
