// Package sprite plays the animations of a sprite sheet, as Aseprite's
// preview does: the sheet, cut into its frames, with the frames of the
// animation playing tinted and the one showing outlined; and beside it
// that frame enlarged on a checkerboard, with the frame before it faint
// under it as an onion skin, and mirrored if asked. Clicking a frame of
// the sheet pauses on it, and dragging across frames makes them the
// animation's. Space plays and pauses, the arrow keys step a frame, and
// the speed and whether it plays back and forth are set under the
// preview.
//
//	sprite.View(c, &app.preview, app.sheet, 8, 3, &app.animations)
package sprite

import (
	"fmt"
	"time"

	"github.com/egoist/mygo/ui"
)

// Animation is a run of frames of a sheet, counted in reading order from
// 0, From to To; how many it shows a second; and whether it plays back
// and forth instead of starting over.
type Animation struct {
	Name     string
	From, To int
	FPS      float64
	PingPong bool
}

// frame returns the frame showing elapsed into the animation.
func (a Animation) frame(elapsed time.Duration) int {
	n := a.To - a.From + 1
	k := int(elapsed.Seconds() * a.FPS)
	if a.PingPong && n > 1 {
		period := 2*n - 2
		k %= period
		if k >= n {
			k = period - k
		}
		return a.From + k
	}
	return a.From + k%n
}

// State is what a preview keeps: the animation chosen, whether it is
// paused and on which frame, when it started, its onion skin and mirror,
// and a drag across the sheet. Keep it in the app's state, one for each
// preview.
type State struct {
	chosen        int
	paused        bool
	frame         int
	start         time.Time
	onion, mirror bool
	holding       bool
	ranged        bool
	anchor        int
}

func icon(body string) *ui.SVG {
	return ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">` + body + `</svg>`))
}

var (
	playIcon  = icon(`<path d="M7 5v14l11-7z" fill="currentColor"/>`)
	pauseIcon = icon(`<path d="M8 5v14M16 5v14" stroke-width="3"/>`)
	backIcon  = icon(`<path d="M6 5v14M18 5l-9 7 9 7z" fill="currentColor"/>`)
	nextIcon  = icon(`<path d="M18 5v14M6 5l9 7-9 7z" fill="currentColor"/>`)
)

// View creates the preview of the animations of sheet, cut into columns
// and rows of frames of the same size. It edits *animations as frames are
// dragged across and its speed and back and forth are set.
func View(c *ui.Context, s *State, sheet *ui.Bitmap, columns, rows int, animations *[]Animation) ui.Element {
	t := c.Theme()
	now := c.Now()
	if s.start.IsZero() {
		s.start = now
	}
	anims := *animations
	s.chosen = min(max(s.chosen, 0), len(anims)-1)
	a := &anims[s.chosen]
	current := a.frame(now.Sub(s.start))
	if s.paused {
		current = s.frame
	} else {
		step := time.Duration(float64(time.Second) / a.FPS)
		c.After(step - now.Sub(s.start)%step)
	}
	// Restarting the clock at an offset keeps the frame showing when play
	// resumes or the speed changes.
	resume := func(at int) {
		if at < a.From || at > a.To {
			at = a.From
		}
		s.start = now.Add(-time.Duration(float64(at-a.From) / a.FPS * float64(time.Second)))
		s.paused = false
	}
	pause := func(at int) { s.paused, s.frame = true, at }
	stepBy := func(d int) {
		n := a.To - a.From + 1
		pause(a.From + ((current-a.From+d)%n+n)%n)
	}
	switch {
	case c.Shortcut(0, ui.KeySpace):
		if s.paused {
			resume(current)
		} else {
			pause(current)
		}
	case c.Shortcut(0, ui.KeyLeft):
		stepBy(-1)
	case c.Shortcut(0, ui.KeyRight):
		stepBy(1)
	}
	sw, sh := sheet.Size()
	fw, fh := float32(sw/columns), float32(sh/rows)
	frameRect := func(i int) ui.Rect {
		return ui.Rect{X: float32(i%columns) * fw, Y: float32(i/columns) * fh, W: fw, H: fh}
	}
	changed := false
	root := ui.Row(c).Gap(t.Space(6)).AlignItems(ui.Start)
	root.Children(func() {
		scale := float32(max(1, 480/sw))
		board := ui.Box(c).Size(float32(sw)*scale, float32(sh)*scale).Shrink(0).Label("Sprite sheet")
		_, _, held := board.Dragged()
		x, y, _ := board.PointerPosition()
		col := min(max(int(x/(fw*scale)), 0), columns-1)
		row := min(max(int(y/(fh*scale)), 0), rows-1)
		under := row*columns + col
		if held {
			if !s.holding {
				s.holding, s.anchor, s.ranged = true, under, false
			}
			if under != s.anchor {
				s.ranged = true
			}
			if s.ranged {
				a.From, a.To = min(s.anchor, under), max(s.anchor, under)
				changed = true
				pause(under)
			}
		} else if s.holding {
			s.holding = false
			if !s.ranged {
				pause(s.anchor)
			}
		}
		playing, showing := *a, current
		board.Draw(func(p *ui.Painter, r ui.Rect) {
			checker(p, r, 8, t)
			p.Image(sheet, r, ui.FillBox)
			cell := func(i int) ui.Rect {
				f := frameRect(i)
				return ui.Rect{X: r.X + f.X*scale, Y: r.Y + f.Y*scale, W: f.W * scale, H: f.H * scale}
			}
			for i := playing.From; i <= playing.To; i++ {
				p.Fill(cell(i), t.Accent.Alpha(0.12), 0)
			}
			grid := t.Text.Alpha(0.12)
			for k := 1; k < columns; k++ {
				x := r.X + float32(k)*fw*scale
				p.Fill(ui.Rect{X: x, Y: r.Y, W: 1, H: r.H}, grid, 0)
			}
			for k := 1; k < rows; k++ {
				y := r.Y + float32(k)*fh*scale
				p.Fill(ui.Rect{X: r.X, Y: y, W: r.W, H: 1}, grid, 0)
			}
			p.Stroke(cell(showing), t.Accent, 2, 2)
		})
		ui.Column(c).Gap(t.Space(3)).Width(t.Space(64)).Children(func() {
			prev := current - 1
			if prev < a.From {
				prev = a.To
			}
			onion, mirror := s.onion && a.To > a.From, s.mirror
			ui.Box(c).Size(t.Space(64), t.Space(64)).Radius(t.Radius).Label(fmt.Sprintf("Frame %d", current)).Draw(func(p *ui.Painter, r ui.Rect) {
				p.Clip(r, t.Radius, func() { checker(p, r, 12, t) })
				k := float32(int(min(r.W/fw, r.H/fh)))
				dst := ui.Rect{X: r.X + (r.W-fw*k)/2, Y: r.Y + (r.H-fh*k)/2, W: fw * k, H: fh * k}
				if onion {
					p.ImageRegion(sheet, frameRect(prev), dst, ui.ImageOptions{FlipX: mirror, Tint: ui.RGBA(255, 255, 255, 0.3)})
				}
				p.ImageRegion(sheet, frameRect(current), dst, ui.ImageOptions{FlipX: mirror})
			})
			ui.Row(c).Gap(t.Space(1)).AlignItems(ui.Center).Children(func() {
				button := func(ic *ui.SVG, label string, do func()) {
					b := ui.ButtonBase(c.Key(label)).Size(t.Space(8), t.Space(8)).Radius(t.Radius).Center().Label(label)
					if b.Hovered() {
						b.Background(t.SurfaceHover)
					}
					b.Children(func() { ui.Icon(c, ic).Size(t.Space(4), t.Space(4)) })
					if b.Clicked() {
						do()
					}
				}
				button(backIcon, "Previous frame", func() { stepBy(-1) })
				if s.paused {
					button(playIcon, "Play", func() { resume(current) })
				} else {
					button(pauseIcon, "Pause", func() { pause(current) })
				}
				button(nextIcon, "Next frame", func() { stepBy(1) })
				ui.Spacer(c)
				ui.Text(c, fmt.Sprintf("%d / %d", current-a.From+1, a.To-a.From+1)).FontFeatures("tnum").TextColor(t.TextMuted)
			})
			if len(anims) > 1 {
				var names []string
				for _, an := range anims {
					names = append(names, an.Name)
				}
				chosen := s.chosen
				if ui.Segmented(c, &chosen, names...).Label("Animation").Changed() {
					s.chosen, s.paused, s.start = chosen, false, now
				}
			}
			fps := a.FPS
			ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
				ui.Text(c, "Speed").TextColor(t.TextMuted)
				if ui.Slider(c, &fps, 1, 30).Step(1).Grow(1).Label("Frames a second").Changed() {
					a.FPS = fps
					changed = true
					if !s.paused {
						resume(current)
					}
				}
				ui.Text(c, fmt.Sprintf("%.0f fps", a.FPS)).FontFeatures("tnum").Width(t.Space(12))
			})
			pingPong := a.PingPong
			if ui.Checkbox(c, &pingPong, "Back and forth").Changed() {
				a.PingPong = pingPong
				changed = true
			}
			ui.Checkbox(c, &s.onion, "Onion skin")
			ui.Checkbox(c, &s.mirror, "Mirror")
		})
	})
	if changed {
		root.SetChanged()
	}
	return root
}

// checker fills r with squares of size n in two tones, as image editors
// show transparency.
func checker(p *ui.Painter, r ui.Rect, n float32, t *ui.Theme) {
	p.Fill(r, t.Surface, 0)
	dark := t.Text.Alpha(0.06)
	for y, j := r.Y, 0; y < r.Y+r.H; y, j = y+n, j+1 {
		for x, i := r.X, 0; x < r.X+r.W; x, i = x+n, i+1 {
			if (i+j)%2 == 1 {
				p.Fill(ui.Rect{X: x, Y: y, W: min(n, r.X+r.W-x), H: min(n, r.Y+r.H-y)}, dark, 0)
			}
		}
	}
}
