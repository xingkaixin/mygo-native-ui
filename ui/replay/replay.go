// Package replay plays back the session of a computer-use agent, what it
// saw and what it did, step by step, as agent consoles replay a run. The
// view shows the screenshot of the step fitted in it, and over it, in its
// place on the screenshot and as large as the screenshot shows, what the
// agent did: a ring pulsing where it clicked, two where it double-clicked,
// a ring and a mouse where it right-clicked, a path with an arrow where it
// dragged, a chip of the text it typed or the key it pressed by where they
// went, and an arrow where it scrolled. Beside it, a card says which step
// of how many it is, when, the action in words, as "Click at 412, 300",
// and the agent's reasoning for it.
//
// Under the screenshot are buttons that go to the previous step, play or
// pause, and go to the next, the time of the step into the session and
// its length, a scrubber with a tick for each step, which goes to the step
// pressed or dragged to and names the one under the pointer, and a menu of
// speeds from 0.5× to 4×. Playing, each step shows as long as the agent
// took to the next one, from half a second to three, at the speed chosen,
// and the view stops at the last step, from which Play starts again at the
// first. Below, a filmstrip of thumbnails,
// each with its step number and an icon of its action, outlines the step
// shown and follows it as it moves; a click on a thumbnail goes to its
// step. Anywhere in the window, unless the focused element takes the key,
// Left and Right go to the previous and the next step, and Space plays and
// pauses.
//
//	app.steps = append(app.steps, replay.Step{
//		Screenshot: shot,
//		Time:       time.Now(),
//		Action:     replay.Action{Kind: replay.Click, At: image.Pt(412, 300)},
//		Note:       "The search field is the quickest way to the meetup.",
//	})
//	...
//	replay.View(c, &app.replay, &app.step, app.steps).Grow(1)
//
// *step is the step shown: the view sets it as the user moves and as it
// plays, and the app sets it to go to another. Pass the same screenshots
// each frame: the view keeps a bitmap of the one shown, and thumbnails,
// made as the filmstrip shows them, of those it showed.
package replay

import (
	"fmt"
	"image"
	"math"
	"strconv"
	"time"

	"github.com/egoist/mygo/ui"
	"golang.org/x/image/draw"
)

// Kind is what an action does.
type Kind uint8

const (
	Click Kind = iota
	DoubleClick
	RightClick
	Drag
	Type
	Key
	Scroll
	Wait
)

// Action is what the agent did at a step, in the pixels of the step's
// screenshot, as its Bounds count them.
type Action struct {
	Kind Kind
	// At is where a click or a scroll was, and where typed text or a key
	// went: the field with the keyboard focus.
	At image.Point
	// Path is the points a drag went along, from where it pressed to where
	// it let go.
	Path []image.Point
	// Text is the text typed, or the key pressed, as "⌘C" or "Return".
	Text string
	// Delta is how far a scroll went: down and right are positive.
	Delta image.Point
}

// Step is a step of a session: the screenshot the agent saw, when it saw
// it, what it did then, and the reasoning it noted, if any.
type Step struct {
	Screenshot image.Image
	Time       time.Time
	Action     Action
	Note       string
}

// minGap and maxGap bound how long a step shows while the view plays, at
// normal speed: the seconds an agent thinks would show nothing moving, and
// steps a moment apart would pass unseen.
const (
	minGap = 500 * time.Millisecond
	maxGap = 3 * time.Second
)

// pulse is how long the ring of a click takes to grow and fade.
const pulse = 1400 * time.Millisecond

var speeds = []float64{0.5, 1, 1.5, 2, 4}

// State is what a view keeps: whether it plays and at what speed, since
// when the step shown has shown, a bitmap of its screenshot, thumbnails of
// the screenshots the filmstrip showed, and where the filmstrip is
// scrolled. Keep it in the app's state, one for each view. The zero State
// is paused, at normal speed.
type State struct {
	playing bool
	// speed is the speed, 0 for 1, so that the zero State plays at normal
	// speed.
	speed float64
	// shown is the step shown in the last frame plus one, 0 before the
	// first frame.
	shown int
	since time.Time
	shot  image.Image
	bmp   *ui.Bitmap
	// thumbs are the thumbnails of the steps, by index, each made from
	// the screenshot it holds, nil until the filmstrip shows it.
	thumbs []thumb
	strip  ui.ScrollState
}

// thumb is a thumbnail of shot, a step's screenshot.
type thumb struct {
	shot image.Image
	bmp  *ui.Bitmap
}

// rate returns the speed to play at, from 0.5 to 4.
func (s *State) rate() float64 {
	if s.speed == 0 {
		return 1
	}
	return s.speed
}

// gap returns how long step i shows while the view plays: as long as the
// agent took to the next step, between minGap and maxGap, over the speed.
func (s *State) gap(steps []Step, i int) time.Duration {
	d := minGap
	if i+1 < len(steps) {
		d = max(minGap, min(maxGap, steps[i+1].Time.Sub(steps[i].Time)))
	}
	return time.Duration(float64(d) / s.rate())
}

// toggle plays, from the first step when the last shows, or pauses.
func (s *State) toggle(step *int, n int, now time.Time) {
	if s.playing {
		s.playing = false
		return
	}
	if *step >= n-1 {
		*step = 0
	}
	s.playing, s.since = true, now
}

// play moves *step on while the view plays, once the step has shown for
// its gap, and stops at the last step. It reports whether the step shown
// is another than in the last frame.
func (s *State) play(c *ui.Context, step *int, steps []Step) bool {
	now := c.Now()
	moved := s.shown != *step+1
	if moved {
		s.shown, s.since = *step+1, now
	}
	if !s.playing {
		return moved
	}
	if *step >= len(steps)-1 {
		s.playing = false
		return moved
	}
	if due := s.since.Add(s.gap(steps, *step)); now.Before(due) {
		c.After(due.Sub(now))
		return moved
	}
	*step++
	s.shown, s.since = *step+1, now
	if *step == len(steps)-1 {
		s.playing = false
	} else {
		c.After(s.gap(steps, *step))
	}
	return true
}

// View creates the replay of steps, of which *step shows: the screenshot
// with its action, the controls and the filmstrip under it, and the card
// of the step beside it. Give it a size, or grow it in its parent.
func View(c *ui.Context, s *State, step *int, steps []Step) ui.Element {
	t := c.Theme()
	n := len(steps)
	if n == 0 {
		return ui.Column(c).Justify(ui.Center).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "No steps").TextColor(t.TextMuted)
		})
	}
	show := func(i int) { *step = max(0, min(n-1, i)) }
	show(*step)
	switch {
	case c.Shortcut(0, ui.KeySpace):
		s.toggle(step, n, c.Now())
	case c.Shortcut(0, ui.KeyLeft):
		show(*step - 1)
	case c.Shortcut(0, ui.KeyRight):
		show(*step + 1)
	}
	moved := s.play(c, step, steps)
	if len(s.thumbs) > n {
		s.thumbs = s.thumbs[:n]
	}
	for len(s.thumbs) < n {
		s.thumbs = append(s.thumbs, thumb{})
	}

	return ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Stretch).Children(func() {
		ui.Column(c).Grow(1).Shrink(1).Gap(t.Space(2)).Children(func() {
			s.screen(c, steps, *step)
			s.transport(c, step, steps, show)
			s.filmstrip(c, *step, steps, moved, show)
		})
		card(c, steps, *step)
	})
}

// screen creates the view of step i's screenshot, fitted in it, with the
// step's action over it.
func (s *State) screen(c *ui.Context, steps []Step, i int) {
	t := c.Theme()
	st := steps[i]
	if st.Screenshot != s.shot {
		s.shot = st.Screenshot
		if s.bmp == nil {
			s.bmp = ui.NewBitmap(st.Screenshot)
		} else {
			// Update keeps the texture of a screenshot of the same size,
			// where stepping at speed would make one for each step.
			s.bmp.Update(st.Screenshot)
		}
	}
	bmp, b := s.bmp, st.Screenshot.Bounds()
	motion := !c.Preferences().ReduceMotion
	ui.Box(c).Grow(1).MinHeight(t.Space(40)).Radius(t.Space(2.5)).Background(t.Surface).Clip().
		Role(ui.RoleImage).Label(fmt.Sprintf("Step %d: %s", i+1, describe(steps, i))).
		Draw(func(p *ui.Painter, r ui.Rect) {
			pad := t.Space(3)
			iw, ih := float32(b.Dx()), float32(b.Dy())
			z := min((r.W-2*pad)/iw, (r.H-2*pad)/ih)
			if z <= 0 {
				return
			}
			img := ui.Rect{X: r.X + (r.W-iw*z)/2, Y: r.Y + (r.H-ih*z)/2, W: iw * z, H: ih * z}
			p.Shadow(img, 0, 0, 1, 6, 0, ui.RGBA(0, 0, 0, 0.18))
			p.Image(bmp, img, ui.FillBox)
			at := func(q image.Point) (float32, float32) {
				return img.X + float32(q.X-b.Min.X)*z, img.Y + float32(q.Y-b.Min.Y)*z
			}
			mark(p, t, st.Action, img, at, motion)
		})
}

// halo is the outline drawn under the marks, which keeps them seen over
// screenshots dark and light.
var halo = ui.RGBA(255, 255, 255, 0.9)

// mark draws action a over the screenshot fitted in img, at mapping its
// pixels into img: the marks are as large as the screenshot shows, k DIPs
// a hundredth of its longer side. The ring of a click pulses while motion
// is on.
func mark(p *ui.Painter, t *ui.Theme, a Action, img ui.Rect, at func(image.Point) (float32, float32), motion bool) {
	k := max(img.W, img.H) / 100
	w := max(1.5, 0.35*k)
	color := t.Accent
	switch a.Kind {
	case Click, DoubleClick, RightClick:
		x, y := at(a.At)
		rings := 1
		if a.Kind == DoubleClick {
			rings = 2
		}
		r := 2.2 * k
		if motion {
			p.AnimationFrame()
			period := pulse.Milliseconds()
			f := float32(p.Now().UnixMilli()%period) / float32(period)
			for j := range rings {
				g := f - 0.22*float32(j)
				if g < 0 {
					g++
				}
				grow := 1 - (1-g)*(1-g)
				circle(p, x, y, r+grow*2.6*k, w, color.Alpha(1-g), false)
			}
		}
		for j := range rings {
			circle(p, x, y, r*(1+0.5*float32(j)), w, color, true)
		}
		dot(p, x, y, 0.4*k, color)
		if a.Kind == RightClick {
			cx, cy := x+r+2.4*k, y-r-1.2*k
			pill := ui.Rect{X: cx - 1.6*k, Y: cy - 1.6*k, W: 3.2 * k, H: 3.2 * k}
			p.Fill(pill, color, 1.6*k)
			icon(p, RightClick, cx, cy, 2.2*k, t.AccentText)
		}
	case Drag:
		if len(a.Path) < 2 {
			return
		}
		pts := make([][2]float32, len(a.Path))
		for i, q := range a.Path {
			pts[i][0], pts[i][1] = at(q)
		}
		dot(p, pts[0][0], pts[0][1], 0.8*k, color)
		arrow(p, pts, w, 2.4*k, color)
	case Type, Key:
		x, y := at(a.At)
		circle(p, x, y, 0.8*k, w, color, true)
		text := a.Text
		if a.Kind == Type {
			text = "“" + text + "”"
		}
		chip(p, text, a.Kind == Key, x, y, k, img)
	case Scroll:
		x, y := at(a.At)
		circle(p, x, y, 1.4*k, w, color, true)
		ex, ey := at(a.At.Add(a.Delta))
		dx, dy := ex-x, ey-y
		l := float32(math.Hypot(float64(dx), float64(dy)))
		if l == 0 {
			return
		}
		// The arrow is as long as the scroll, within bounds: a scroll of a
		// few lines still shows its way, and one of pages stays near.
		ux, uy, long := dx/l, dy/l, max(6*k, min(14*k, l))
		arrow(p, [][2]float32{{x + ux*2*k, y + uy*2*k}, {x + ux*long, y + uy*long}}, w, 2.4*k, color)
	}
}

// circle strokes a circle of radius r around (x, y), w wide, on a halo
// if outlined.
func circle(p *ui.Painter, x, y, r, w float32, color ui.Color, outlined bool) {
	var c ui.Path
	c.Circle(x, y, r)
	if outlined {
		p.StrokePath(&c, w+2, halo)
	}
	p.StrokePath(&c, w, color)
}

// dot fills a circle of radius r around (x, y) on a halo.
func dot(p *ui.Painter, x, y, r float32, color ui.Color) {
	var c ui.Path
	p.FillPath(c.Circle(x, y, r+1), halo)
	p.FillPath(c.Reset().Circle(x, y, r), color)
}

// arrow draws a line through pts, w wide, on a halo, with a head long at
// its end.
func arrow(p *ui.Painter, pts [][2]float32, w, long float32, color ui.Color) {
	end := pts[len(pts)-1]
	// The head points along the line's last stretch at least as long as
	// it, so that the jitter of a drag's last points does not turn it.
	from := pts[0]
	for i := len(pts) - 2; i >= 0; i-- {
		if math.Hypot(float64(end[0]-pts[i][0]), float64(end[1]-pts[i][1])) >= float64(long) {
			from = pts[i]
			break
		}
	}
	dx, dy := end[0]-from[0], end[1]-from[1]
	l := float32(math.Hypot(float64(dx), float64(dy)))
	if l == 0 {
		return
	}
	ux, uy := dx/l, dy/l
	bx, by := end[0]-ux*long, end[1]-uy*long
	var head ui.Path
	head.MoveTo(end[0], end[1]).LineTo(bx-uy*long*0.55, by+ux*long*0.55).LineTo(bx+uy*long*0.55, by-ux*long*0.55).Close()
	line := append(pts[:len(pts)-1:len(pts)-1], [2]float32{end[0] - ux*long*0.5, end[1] - uy*long*0.5})
	p.StrokePolyline(line, w+2, halo)
	p.StrokePath(&head, 2, halo)
	p.StrokePolyline(line, w, color)
	p.FillPath(&head, color)
}

// chip draws text in a dark chip below and right of (x, y), or where it
// fits in img, its size following k; keys show bold.
func chip(p *ui.Painter, text string, bold bool, x, y, k float32, img ui.Rect) {
	weight := 500
	if bold {
		weight = 700
	}
	span := ui.Span{Text: text, Size: max(11, 1.5*k), Weight: weight, Color: ui.RGB(255, 255, 255)}
	padX, padY := span.Size*0.6, span.Size*0.35
	tw, th := p.MeasureText(img.W*0.6, span)
	r := ui.Rect{X: x + 0.8*k, Y: y + 1.4*k, W: tw + 2*padX, H: th + 2*padY}
	r.X = max(img.X+k, min(img.X+img.W-r.W-k, r.X))
	if r.Y+r.H > img.Y+img.H-k {
		r.Y = y - 1.4*k - r.H
	}
	p.Shadow(r, r.H/2, 0, 1, 4, 0, ui.RGBA(0, 0, 0, 0.3))
	p.Fill(r, ui.RGBA(24, 24, 28, 0.9), min(r.H/2, span.Size))
	p.RichText(r.X+padX, r.Y+padY, img.W*0.6, span)
}

// transport creates the row under the screenshot: previous, play or
// pause, next, the time, the scrubber and the speed.
func (s *State) transport(c *ui.Context, step *int, steps []Step, show func(int)) {
	t := c.Theme()
	n := len(steps)
	ui.Row(c).Gap(t.Space(1)).AlignItems(ui.Center).Children(func() {
		if button(c, "Previous step", *step > 0, func(p *ui.Painter, x, y float32, color ui.Color) {
			skip(p, x, y, t.Space(3.5), color, false)
		}).Clicked() {
			show(*step - 1)
		}
		if play(c, s.playing).Clicked() {
			s.toggle(step, n, c.Now())
		}
		if button(c, "Next step", *step < n-1, func(p *ui.Painter, x, y float32, color ui.Color) {
			skip(p, x, y, t.Space(3.5), color, true)
		}).Clicked() {
			show(*step + 1)
		}
		long := steps[n-1].Time.Sub(steps[0].Time) >= time.Hour
		ui.Row(c).Margin(0, t.Space(2)).Children(func() {
			ui.Text(c, clock(steps[*step].Time.Sub(steps[0].Time), long)).FontFeatures("tnum")
			ui.Text(c, " / "+clock(steps[n-1].Time.Sub(steps[0].Time), long)).FontFeatures("tnum").TextColor(t.TextMuted)
		})
		s.scrubber(c, *step, steps, show)

		speed := ui.ButtonBase(c).Height(t.Space(9)).PaddingX(t.Space(2.5)).Radius(t.Space(2)).
			Label("Speed").Value(times(s.rate())).Tooltip("Speed")
		face(speed, t)
		speed.Menu(func(m *ui.Menu) {
			for _, v := range speeds {
				name := times(v)
				if v == 1 {
					name = "Normal"
				}
				if m.Item(name).Checked(v == s.rate()).Chosen() {
					s.speed = v
				}
			}
		})
		speed.Children(func() { ui.Text(c, times(s.rate())).FontFeatures("tnum") })
	})
}

// scrubber creates the bar of the steps, a tick each, filled up to the
// step shown, and while playing on toward the next; a press or a drag goes
// to the step nearest, and the pointer names the step under it.
func (s *State) scrubber(c *ui.Context, i int, steps []Step, show func(int)) {
	t := c.Theme()
	n := len(steps)
	h, inset := t.Space(6), t.Space(2)
	bar := ui.Box(c).Height(h).Grow(1).MinWidth(t.Space(20)).Focusable().Role(ui.RoleSlider).Label("Step").
		Range(1, float64(n), float64(i+1)).Value(fmt.Sprintf("Step %d of %d", i+1, n))
	w := bar.LayoutBounds().W
	nearest := func(x float32) int {
		if n == 1 {
			return 0
		}
		f := (x - inset) / max(1, w-2*inset)
		return max(0, min(n-1, int(math.Round(float64(f*float32(n-1))))))
	}
	px, _, over := bar.PointerPosition()
	_, _, pressed := bar.Dragged()
	if pressed {
		show(nearest(px))
	}
	hover := -1
	if over || pressed {
		hover = nearest(px)
		tx := inset
		if n > 1 {
			tx += (w - 2*inset) * float32(hover) / float32(n-1)
		}
		bar.Children(func() {
			anchor := ui.Box(c).Absolute().PassThrough().Left(tx).Top(0).Size(1, h)
			tip(c, anchor, fmt.Sprintf("Step %d", hover+1), describe(steps, hover))
		})
	}
	playing, since, gap := s.playing, s.since, s.gap(steps, i)
	bar.Draw(func(p *ui.Painter, r ui.Rect) {
		x0, span := r.X+inset, r.W-2*inset
		x := func(f float32) float32 {
			if n == 1 {
				return x0
			}
			return x0 + span*f/float32(n-1)
		}
		pos := float32(i)
		if playing && i < n-1 {
			p.AnimationFrame()
			pos += min(1, float32(p.Now().Sub(since))/float32(gap))
		}
		th := t.Space(1)
		y := r.Y + (r.H-th)/2
		p.Fill(ui.Rect{X: x0, Y: y, W: span, H: th}, t.Text.Alpha(0.12), th/2)
		// Ticks closer than this would run together into a band.
		if n == 1 || span/float32(n-1) >= t.Space(1) {
			tick := t.Space(2.5)
			for j := range n {
				color := t.Text.Alpha(0.3)
				switch {
				case j == hover:
					color = t.Text
				case float32(j) <= pos:
					color = t.Accent
				}
				p.Fill(ui.Rect{X: x(float32(j)) - 1, Y: r.Y + (r.H-tick)/2, W: 2, H: tick}, color, 1)
			}
		}
		p.Fill(ui.Rect{X: x0, Y: y, W: x(pos) - x0, H: th}, t.Accent, th/2)
		k := t.Space(3.5)
		knob := ui.Rect{X: x(pos) - k/2, Y: r.Y + (r.H-k)/2, W: k, H: k}
		p.Shadow(knob, k/2, 0, 1, 3, 0, ui.RGBA(0, 0, 0, 0.25))
		p.Fill(knob, t.Accent, k/2)
	})
}

// tip shows above anchor the step under the pointer and its action.
func tip(c *ui.Context, anchor ui.Element, title, action string) {
	t := c.Theme()
	ui.Overlay(c, func() {
		// The tip lets the pointer through to the scrubber under it, which
		// would otherwise lose the hover and the tip.
		ui.Column(c).Absolute().PassThrough().Role(ui.RoleTooltip).AlignItems(ui.Center).
			AttachTo(anchor, ui.AnchorTop, ui.AnchorBottom).Margin(0, 0, t.Space(1), 0).
			Padding(t.Space(1), t.Space(2)).Radius(t.Space(1.5)).Background(t.Text).
			Shadow(0, 2, 8, 0, ui.RGBA(0, 0, 0, 0.2)).Children(func() {
			ui.Text(c, title).PassThrough().SingleLine().FontSize(t.FontSize - 1).Bold().TextColor(t.Background)
			ui.Text(c, action).PassThrough().SingleLine().FontSize(t.FontSize - 1).TextColor(t.Background)
		})
	})
}

// filmstrip creates the row of thumbnails of steps, which scrolls
// sideways, i outlined: only the thumbnails near the view are built, and
// made from their screenshots a few a frame. moved scrolls i into view.
func (s *State) filmstrip(c *ui.Context, i int, steps []Step, moved bool, show func(int)) {
	t := c.Theme()
	n := len(steps)
	// bottom leaves room under the thumbnails for the scroll bar.
	tw, th, gap, pad, bottom := t.Space(26), t.Space(16), t.Space(2), t.Space(1), t.Space(3)
	pitch := tw + gap
	strip := ui.ScrollHorizontal(c).TrackScroll(&s.strip).Height(pad + th + bottom).Shrink(0)
	vw := strip.LayoutBounds().W
	if vw <= 0 {
		vw, _ = c.Size()
	}
	if moved {
		x := pad + float32(i)*pitch
		switch {
		case x-pad < s.strip.X:
			s.strip.ScrollTo(x-pad, 0, ui.Smooth)
		case x+tw+pad > s.strip.X+vw:
			s.strip.ScrollTo(x+tw+pad-vw, 0, ui.Smooth)
		}
	}
	// A screen's width each way is built, so that thumbnails show at once
	// as the strip scrolls.
	first := max(0, int((s.strip.X-vw)/pitch))
	last := min(n-1, int((s.strip.X+2*vw)/pitch))
	made := 0
	strip.Children(func() {
		ui.Box(c).Size(2*pad+float32(n)*pitch-gap, pad+th+bottom).Children(func() {
			for j := first; j <= last; j++ {
				x := pad + float32(j)*pitch
				if s.thumbnail(c.Key(j), steps, j, j == i, x, pad, tw, th, &made).Clicked() {
					show(j)
				}
			}
		})
	})
}

// thumbnail creates the thumbnail of step j at (x, y), w×h, outlined if
// current, with its number and the icon of its action; it makes the
// thumbnail from the screenshot unless two were made this frame, as
// *made counts, and asks for another frame to make it then.
func (s *State) thumbnail(c *ui.Context, steps []Step, j int, current bool, x, y, w, h float32, made *int) ui.Element {
	t := c.Theme()
	tb := &s.thumbs[j]
	if shot := steps[j].Screenshot; tb.shot != shot {
		tb.shot, tb.bmp = shot, nil
	}
	if tb.bmp == nil {
		if *made < 2 {
			// Twice the DIPs keeps thumbnails sharp on displays of two
			// pixels a DIP.
			tb.bmp = ui.NewBitmap(shrink(tb.shot, int(2*w), int(2*h)))
			*made++
		} else {
			c.AnimationFrame()
		}
	}
	bmp, kind := tb.bmp, steps[j].Action.Kind
	words := describe(steps, j)
	b := ui.ButtonBase(c).Absolute().Left(x).Top(y).Size(w, h).Radius(t.Space(1.5)).
		Label(fmt.Sprintf("Step %d: %s", j+1, words)).Tooltip(words).Selected(current)
	hovered := b.Hovered()
	b.Draw(func(p *ui.Painter, r ui.Rect) {
		radius := t.Space(1.5)
		p.Clip(r, radius, func() {
			p.Fill(r, t.Text.Alpha(0.08), 0)
			if bmp != nil {
				p.Image(bmp, r, ui.Contain)
			}
		})
		label := ui.Span{Text: strconv.Itoa(j + 1), Size: t.FontSize - 2, Weight: 600, Color: ui.RGB(255, 255, 255), Features: "tnum"}
		lw, lh := p.MeasureText(0, label)
		pill := ui.Rect{X: r.X + t.Space(1), H: lh + t.Space(1)}
		pill.Y = r.Y + r.H - pill.H - t.Space(1)
		pill.W = t.Space(1.5) + lh + t.Space(1) + lw + t.Space(1.5)
		p.Fill(pill, ui.RGBA(0, 0, 0, 0.62), pill.H/2)
		icon(p, kind, pill.X+t.Space(1.5)+lh/2, pill.Y+pill.H/2, lh, ui.RGB(255, 255, 255))
		p.RichText(pill.X+t.Space(1.5)+lh+t.Space(1), pill.Y+(pill.H-lh)/2, 0, label)
		switch {
		case current:
			p.Stroke(r, t.Accent, radius, 2)
		case hovered:
			p.Stroke(r, t.Text.Alpha(0.35), radius, 1)
		default:
			p.Stroke(r, t.Border, radius, 1)
		}
	})
	return b
}

// shrink returns img scaled to fit in w×h pixels.
func shrink(img image.Image, w, h int) image.Image {
	b := img.Bounds()
	z := min(float64(w)/float64(b.Dx()), float64(h)/float64(b.Dy()))
	if z >= 1 {
		return img
	}
	dst := image.NewRGBA(image.Rect(0, 0, max(1, int(float64(b.Dx())*z)), max(1, int(float64(b.Dy())*z))))
	draw.BiLinear.Scale(dst, dst.Bounds(), img, b, draw.Src, nil)
	return dst
}

// card creates the card beside the screenshot: which step of how many,
// when, the action in words, and the agent's reasoning.
func card(c *ui.Context, steps []Step, i int) {
	t := c.Theme()
	st := steps[i]
	long := steps[len(steps)-1].Time.Sub(steps[0].Time) >= time.Hour
	ui.Column(c).Width(t.Space(72)).Shrink(0).Radius(t.Space(2.5)).Border(1, t.Border).Clip().Children(func() {
		ui.Row(c).Gap(t.Space(2)).Padding(t.Space(2.5), t.Space(3)).Children(func() {
			ui.Text(c, fmt.Sprintf("Step %d of %d", i+1, len(steps))).Bold().SingleLine().Grow(1).Shrink(1)
			ui.Text(c, clock(st.Time.Sub(steps[0].Time), long)).TextColor(t.TextMuted).FontFeatures("tnum")
		})
		ui.Divider(c)
		ui.Column(c).Grow(1).Gap(t.Space(3)).Padding(t.Space(3)).Children(func() {
			ui.Row(c).Gap(t.Space(2.5)).AlignItems(ui.Center).Children(func() {
				kind := st.Action.Kind
				ui.Box(c).Size(t.Space(8), t.Space(8)).Shrink(0).Radius(t.Space(2)).Background(t.Accent.Alpha(0.12)).
					Draw(func(p *ui.Painter, r ui.Rect) {
						icon(p, kind, r.X+r.W/2, r.Y+r.H/2, t.Space(4.5), t.Accent)
					})
				ui.Text(c, describe(steps, i)).Bold().Shrink(1).Selectable()
			})
			ui.Column(c).Grow(1).Gap(t.Space(1)).Children(func() {
				ui.Text(c, "Reasoning").FontSize(t.FontSize - 2).Bold().TextColor(t.TextMuted)
				ui.Scroll(c).Grow(1).Children(func() {
					if st.Note == "" {
						ui.Text(c, "No reasoning noted").TextColor(t.TextMuted)
						return
					}
					ui.Text(c, st.Note).LineHeight(1.45).Selectable()
				})
			})
		})
	})
}

// describe returns step i's action in words, as "Click at 412, 300".
func describe(steps []Step, i int) string {
	a := steps[i].Action
	at := strconv.Itoa(a.At.X) + ", " + strconv.Itoa(a.At.Y)
	switch a.Kind {
	case Click:
		return "Click at " + at
	case DoubleClick:
		return "Double-click at " + at
	case RightClick:
		return "Right-click at " + at
	case Drag:
		if len(a.Path) == 0 {
			return "Drag"
		}
		from, to := a.Path[0], a.Path[len(a.Path)-1]
		return fmt.Sprintf("Drag from %d, %d to %d, %d", from.X, from.Y, to.X, to.Y)
	case Type:
		return "Type “" + a.Text + "”"
	case Key:
		return "Press " + a.Text
	case Scroll:
		var ways []string
		if d := a.Delta.Y; d > 0 {
			ways = append(ways, "down "+strconv.Itoa(d))
		} else if d < 0 {
			ways = append(ways, "up "+strconv.Itoa(-d))
		}
		if d := a.Delta.X; d > 0 {
			ways = append(ways, "right "+strconv.Itoa(d))
		} else if d < 0 {
			ways = append(ways, "left "+strconv.Itoa(-d))
		}
		words := "Scroll"
		for k, w := range ways {
			if k > 0 {
				words += " and"
			}
			words += " " + w
		}
		return words + " at " + at
	case Wait:
		if i+1 < len(steps) {
			d := steps[i+1].Time.Sub(steps[i].Time)
			return "Wait " + strconv.FormatFloat(math.Round(d.Seconds()*10)/10, 'f', -1, 64) + " s"
		}
		return "Wait"
	}
	return ""
}

// icon draws the icon of kind, s wide around (cx, cy): a pointer for a
// click, with rays for a double click, a mouse for a right click, a curve
// with an arrow for a drag, letters for typing, a key cap for a key, a
// double arrow for a scroll, and a clock for a wait.
func icon(p *ui.Painter, kind Kind, cx, cy, s float32, color ui.Color) {
	lw := max(1.25, s/9)
	switch kind {
	case Click, DoubleClick:
		ox, oy, z := cx-0.26*s, cy-0.44*s, s
		if kind == DoubleClick {
			ox, oy, z = cx-0.04*s, cy-0.16*s, 0.72*s
			for _, a := range []float64{math.Pi, math.Pi * 1.25, math.Pi * 1.5} {
				dx, dy := float32(math.Cos(a)), float32(math.Sin(a))
				p.StrokePolyline([][2]float32{{ox + dx*0.13*s, oy + dy*0.13*s}, {ox + dx*0.32*s, oy + dy*0.32*s}}, lw, color)
			}
		}
		pts := [][2]float32{{0, 0}, {0, 0.78}, {0.2, 0.6}, {0.34, 0.9}, {0.46, 0.84}, {0.33, 0.55}, {0.56, 0.55}}
		var arrow ui.Path
		for k, q := range pts {
			if k == 0 {
				arrow.MoveTo(ox+q[0]*z, oy+q[1]*z)
			} else {
				arrow.LineTo(ox+q[0]*z, oy+q[1]*z)
			}
		}
		p.FillPath(arrow.Close(), color)
	case RightClick:
		body := ui.Rect{X: cx - 0.3*s, Y: cy - 0.45*s, W: 0.6 * s, H: 0.9 * s}
		split := body.Y + 0.42*body.H
		p.Clip(body, body.W/2, func() {
			p.Fill(ui.Rect{X: cx, Y: body.Y, W: body.W / 2, H: split - body.Y}, color, 0)
		})
		p.Stroke(body, color, body.W/2, lw)
		p.Line(body.X, split, body.X+body.W, split, lw, color)
		p.Line(cx, body.Y, cx, split, lw, color)
	case Drag:
		var curve ui.Path
		curve.MoveTo(cx-0.4*s, cy+0.36*s).QuadTo(cx-0.36*s, cy-0.3*s, cx+0.22*s, cy-0.32*s)
		p.StrokePath(&curve, lw, color)
		var head ui.Path
		head.MoveTo(cx+0.46*s, cy-0.32*s).LineTo(cx+0.16*s, cy-0.5*s).LineTo(cx+0.16*s, cy-0.14*s).Close()
		p.FillPath(&head, color)
		var start ui.Path
		p.FillPath(start.Circle(cx-0.4*s, cy+0.36*s, 0.12*s), color)
	case Type:
		label := ui.Span{Text: "Aa", Size: 0.72 * s, Weight: 700, Color: color}
		w, h := p.MeasureText(0, label)
		p.RichText(cx-w/2, cy-h/2, 0, label)
	case Key:
		p.Stroke(ui.Rect{X: cx - 0.44*s, Y: cy - 0.44*s, W: 0.88 * s, H: 0.88 * s}, color, 0.2*s, lw)
		label := ui.Span{Text: "⌘", Size: 0.56 * s, Weight: 600, Color: color}
		w, h := p.MeasureText(0, label)
		p.RichText(cx-w/2, cy-h/2, 0, label)
	case Scroll:
		p.Line(cx, cy-0.24*s, cx, cy+0.24*s, lw, color)
		for _, d := range []float32{-1, 1} {
			var head ui.Path
			head.MoveTo(cx, cy+d*0.5*s).LineTo(cx-0.26*s, cy+d*0.2*s).LineTo(cx+0.26*s, cy+d*0.2*s).Close()
			p.FillPath(&head, color)
		}
	case Wait:
		var face ui.Path
		p.StrokePath(face.Circle(cx, cy, 0.42*s), lw, color)
		p.StrokePolyline([][2]float32{{cx, cy - 0.26*s}, {cx, cy}, {cx + 0.2*s, cy}}, lw, color)
	}
}

// face gives button b the face of a control in the transport: none until
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
// around its center, dimmed unless enabled.
func button(c *ui.Context, label string, enabled bool, draw func(p *ui.Painter, x, y float32, color ui.Color)) ui.Element {
	t := c.Theme()
	b := ui.ButtonBase(c).Size(t.Space(9), t.Space(9)).Radius(t.Space(2)).Label(label).Tooltip(label).Disabled(!enabled)
	color := t.Text
	if enabled {
		face(b, t)
	} else {
		color = t.TextMuted.Alpha(0.6)
	}
	b.Draw(func(p *ui.Painter, r ui.Rect) { draw(p, r.X+r.W/2, r.Y+r.H/2, color) })
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
	b := ui.ButtonBase(c).Size(d, d).Radius(d / 2).Label(label).Tooltip(label).Background(t.Accent)
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

// skip draws the icon of going to the next step, or the previous, s wide
// around (cx, cy): a triangle pointing at a bar.
func skip(p *ui.Painter, cx, cy, s float32, color ui.Color, next bool) {
	d := float32(1)
	if !next {
		d = -1
	}
	var tri ui.Path
	tri.MoveTo(cx-d*s*0.45, cy-s/2).LineTo(cx+d*s*0.3, cy).LineTo(cx-d*s*0.45, cy+s/2).Close()
	p.FillPath(&tri, color)
	bw := s * 0.18
	x := cx + d*s*0.3
	if !next {
		x -= bw
	}
	p.Fill(ui.Rect{X: x, Y: cy - s/2, W: bw, H: s}, color, 1)
}

// clock writes d as 4:05, or as 1:04:05 when long, for a session an hour
// long or longer.
func clock(d time.Duration, long bool) string {
	n := int(max(0, d) / time.Second)
	if long {
		return fmt.Sprintf("%d:%02d:%02d", n/3600, n/60%60, n%60)
	}
	return fmt.Sprintf("%d:%02d", n/60, n%60)
}

// times writes speed v as 1.5×.
func times(v float64) string { return strconv.FormatFloat(v, 'f', -1, 64) + "×" }
