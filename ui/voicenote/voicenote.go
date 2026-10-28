// Package voicenote shows a voice message in a chat, as messaging apps
// do: a bubble with a play button, the note's loudness as bars, the part
// heard in full and the rest faint, and how long it lasts, or how far it
// has played. A click or a drag along the bars plays from there; the speed
// button goes from 1× to 1.5× to 2×; and Aa shows what was said, when the
// note has a transcript. A dot marks a note not yet heard, until it plays.
// The user's own notes are in the accent color, those of others in the
// surface's.
//
//	app.voice.Player = sound.NewPlayer()
//	...
//	voicenote.View(c, &app.voice, &app.note)
package voicenote

import (
	"fmt"
	"math"
	"time"

	"github.com/egoist/mygo/audio"
	"github.com/egoist/mygo/ui"
)

// Note is a voice message: how loud it is along its length, from 0 to 1,
// what it says, whether the user sent it and whether they have heard it.
type Note struct {
	Peaks      []float32
	Transcript string
	Mine       bool
	Heard      bool
}

// State is what a voice note keeps: its Player, which the app makes, and
// the speed, the drag and whether the transcript shows. Keep it in the
// app's state, one for each note.
type State struct {
	Player *audio.Player
	speed  int
	text   bool
	seek   bool
}

var speeds = []float64{1, 1.5, 2}

var playIcon = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path d="M8 5.5v13l10.5-6.5z" fill="currentColor"/></svg>`))
var pauseIcon = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path d="M7 5h3.5v14H7zM13.5 5H17v14h-3.5z" fill="currentColor"/></svg>`))

// View creates the bubble of *note, played by s.Player.
func View(c *ui.Context, s *State, note *Note) ui.Element {
	t := c.Theme()
	p := s.Player
	var pos, length time.Duration
	playing := false
	if p != nil {
		pos, length, playing = p.Position(), p.Duration(), p.Playing()
		if length > 0 && pos >= length {
			pos = 0
		}
	}
	if playing || s.seek {
		c.AnimationFrame()
	}
	fg, bg, faint := t.Text, t.Surface, t.Text.Alpha(0.25)
	if note.Mine {
		fg, bg, faint = t.AccentText, t.Accent, t.AccentText.Alpha(0.4)
	}
	bubble := ui.Column(c).Gap(t.Space(2)).Padding(t.Space(2.5), t.Space(3)).Radius(t.Space(4)).Background(bg).
		Width(t.Space(72)).Label("Voice message").Value(clock(length))
	if !note.Mine {
		bubble.Border(1, t.Border)
	}
	bubble.Children(func() {
		ui.Row(c).Gap(t.Space(2.5)).AlignItems(ui.Center).Children(func() {
			btn := ui.ButtonBase(c).Size(t.Space(9), t.Space(9)).Radius(t.Space(4.5)).Center().Cursor(ui.CursorPointer)
			btn.Background(fg)
			if btn.Hovered() {
				btn.Background(fg.Alpha(0.85))
			}
			icon := playIcon
			btn.Label("Play")
			if playing {
				icon = pauseIcon
				btn.Label("Pause")
			}
			if btn.Clicked() && p != nil {
				if playing {
					p.Pause()
				} else {
					if p.Position() >= length {
						p.Seek(0)
					}
					p.SetSpeed(speeds[s.speed])
					p.Play()
					note.Heard = true
				}
				btn.SetChanged()
			}
			btn.Children(func() {
				ui.Icon(c, icon).Size(t.Space(4.5), t.Space(4.5)).TextColor(bg)
			})
			ui.Column(c).Grow(1).Gap(t.Space(1)).Children(func() {
				bars(c, s, note, pos, length, fg, faint)
				ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
					shown := length
					if playing || pos > 0 {
						shown = pos
					}
					ui.Text(c, clock(shown)).FontSize(t.FontSize - 2).FontFeatures("tnum").TextColor(fg.Alpha(0.75))
					if !note.Heard {
						ui.Box(c).Size(6, 6).Radius(3).Background(t.Accent).Label("Not heard")
					}
					ui.Box(c).Grow(1)
					chip := func(label, name string) ui.Element {
						e := ui.ButtonBase(c.Key(name)).Padding(0, t.Space(1.5)).Height(t.Space(4.5)).Radius(t.Space(2.25)).
							Background(fg.Alpha(0.14)).Label(name).Cursor(ui.CursorPointer)
						if e.Hovered() {
							e.Background(fg.Alpha(0.24))
						}
						e.Children(func() {
							ui.Text(c, label).FontSize(t.FontSize - 3).Bold().TextColor(fg)
						})
						return e
					}
					if chip(fmt.Sprintf("%g×", speeds[s.speed]), "Speed").Clicked() {
						s.speed = (s.speed + 1) % len(speeds)
						if p != nil {
							p.SetSpeed(speeds[s.speed])
						}
					}
					if note.Transcript != "" {
						if chip("Aa", "Transcript").Selected(s.text).Clicked() {
							s.text = !s.text
						}
					}
				})
			})
		})
		if s.text && note.Transcript != "" {
			ui.Text(c, note.Transcript).FontSize(t.FontSize - 1).TextColor(fg).Selectable()
		}
	})
	return bubble
}

// bars draws the note's loudness, the part played in fg, and plays from
// where it is clicked or dragged to.
func bars(c *ui.Context, s *State, note *Note, pos, length time.Duration, fg, faint ui.Color) {
	t := c.Theme()
	e := ui.Box(c).FillWidth().Height(t.Space(7)).Cursor(ui.CursorPointer).Role(ui.RoleSlider).
		Label("Position").Range(0, length.Seconds(), pos.Seconds())
	px, _, _ := e.PointerPosition()
	w := e.Bounds().W
	frac := float32(0)
	if length > 0 {
		frac = float32(pos) / float32(length)
	}
	if e.Pressed() && w > 0 && s.Player != nil {
		frac = max(0, min(1, px/w))
		s.Player.Seek(time.Duration(float64(length) * float64(frac)))
		s.seek = true
		note.Heard = true
	} else {
		s.seek = false
	}
	peaks := note.Peaks
	e.Draw(func(p *ui.Painter, r ui.Rect) {
		const bw, gap = 3, 2
		n := int((r.W + gap) / (bw + gap))
		if n <= 0 {
			return
		}
		for i := range n {
			lo, hi := i*len(peaks)/n, max(i*len(peaks)/n+1, (i+1)*len(peaks)/n)
			v := float32(0)
			for _, pk := range peaks[min(lo, len(peaks)):min(hi, len(peaks))] {
				v = max(v, pk)
			}
			h := max(3, float32(math.Sqrt(float64(v)))*r.H)
			x := r.X + float32(i)*(bw+gap)
			col := faint
			if (float32(i)+0.5)/float32(n) <= frac {
				col = fg
			}
			p.Fill(ui.Rect{X: x, Y: r.Y + (r.H-h)/2, W: bw, H: h}, col, bw/2)
		}
	})
}

// clock writes d as minutes and seconds, as 0:07.
func clock(d time.Duration) string {
	sec := int(d.Round(time.Second) / time.Second)
	return fmt.Sprintf("%d:%02d", sec/60, sec%60)
}
