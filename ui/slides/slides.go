// Package slides presents slides, as Keynote and PowerPoint do: a strip
// of thumbnails to choose one, a presenter view with the slide shown,
// the next, the speaker's notes and how long the talk has run, and the
// slide alone filling the screen. The arrows, Space and Page Up and Down
// move between the slides, Home and End to the first and the last, and
// Escape ends the show; a click on the slide alone moves to the next.
//
//	slides.Thumbnails(c, &app.deck, app.slides).Width(180)
//	ev := slides.Presenter(c, &app.deck, app.slides)
//
// A slide draws in Width by Height points, scaled to fit where it shows;
// Paint draws one anywhere else, as in the app's editor.
package slides

import (
	"fmt"
	"time"

	"github.com/egoist/mygo/ui"
)

// Width and Height are the size a slide draws in.
const Width, Height = 960, 540

// Slide is a slide: what it draws in r, Width by Height, and the
// speaker's notes.
type Slide struct {
	Draw  func(p *ui.Painter, r ui.Rect)
	Notes string
}

// State is what a deck keeps: the slide shown, and when the talk
// started. Keep it in the app's state, one for each deck.
type State struct {
	// Current is the slide shown, which the keys and clicks change.
	Current int
	start   time.Time
	list    ui.ListState
}

// Restart counts the talk's time from now.
func (s *State) Restart() { s.start = time.Time{} }

// Events is a view of the deck, and whether Escape ended the show this
// frame.
type Events struct {
	ui.Element
	Ended bool
}

// Paint draws slide in r, scaled to fit, on white.
func Paint(p *ui.Painter, r ui.Rect, slide Slide) {
	k := min(r.W/Width, r.H/Height)
	box := ui.Rect{X: r.X + (r.W-Width*k)/2, Y: r.Y + (r.H-Height*k)/2, W: Width * k, H: Height * k}
	p.Fill(box, ui.RGB(255, 255, 255), 0)
	if slide.Draw == nil {
		return
	}
	p.Clip(box, 0, func() {
		p.Transform(ui.Scaling(k, k), box.X, box.Y, func() {
			slide.Draw(p, ui.Rect{X: box.X, Y: box.Y, W: Width, H: Height})
		})
	})
}

// keys moves s.Current by the keys pressed on el, and reports Escape.
func (s *State) keys(el ui.Element, n int) bool {
	for _, k := range []ui.Key{ui.KeyRight, ui.KeyDown, ui.KeySpace, ui.KeyPageDown} {
		if el.Shortcut(0, k) {
			s.Current++
		}
	}
	for _, k := range []ui.Key{ui.KeyLeft, ui.KeyUp, ui.KeyPageUp} {
		if el.Shortcut(0, k) {
			s.Current--
		}
	}
	if el.Shortcut(0, ui.KeyHome) {
		s.Current = 0
	}
	if el.Shortcut(0, ui.KeyEnd) {
		s.Current = n - 1
	}
	s.Current = max(0, min(n-1, s.Current))
	return el.Shortcut(0, ui.KeyEscape)
}

// Thumbnails creates the strip of the slides, the one shown lit. Changed
// reports another chosen. Give it a width.
func Thumbnails(c *ui.Context, s *State, slides []Slide) ui.Element {
	t := c.Theme()
	before := s.Current
	s.Current = max(0, min(len(slides)-1, s.Current))
	list := ui.List(c, &s.list, len(slides), func(i int) {
		row := ui.Row(c).Padding(t.Space(1.5), t.Space(2)).Gap(t.Space(2)).Cursor(ui.CursorPointer)
		if row.Clicked() {
			s.Current = i
			s.list.Focus(c)
		}
		row.Children(func() {
			num := ui.Text(c, fmt.Sprint(i+1)).Width(t.Space(5)).FontSize(t.FontSize - 1).FontFeatures("tnum").TextAlign(ui.End)
			if i == s.Current {
				num.TextColor(t.Accent).Bold()
			} else {
				num.TextColor(t.TextMuted)
			}
			slide, on := slides[i], i == s.Current
			ui.Box(c).Grow(1).AspectRatio(Width / float32(Height)).Draw(func(p *ui.Painter, r ui.Rect) {
				p.Shadow(r, 2, 0, 1, 3, 0, ui.RGB(0, 0, 0).Alpha(0.2))
				Paint(p, r, slide)
				if on {
					p.Stroke(ui.Rect{X: r.X - 3, Y: r.Y - 3, W: r.W + 6, H: r.H + 6}, t.Accent, 4, 2.5)
				} else {
					p.Stroke(r, t.Border, 0, 1)
				}
			})
		})
	}).Label("Slides")
	s.keys(list, len(slides))
	if s.Current != before {
		list.SetChanged()
	}
	return list
}

// Presenter creates the presenter view: the slide shown, large, the next
// one beside it with the notes below, and over them the time the talk
// has run, the clock and which slide this is.
func Presenter(c *ui.Context, s *State, slides []Slide) Events {
	t := c.Theme()
	if len(slides) == 0 {
		return Events{Element: ui.Box(c)}
	}
	root := ui.Column(c.Key("slides.presenter")).Focusable().FocusRing(false).AutoFocus().Gap(t.Space(4)).Padding(t.Space(6)).Background(ui.RGB(24, 24, 27))
	ended := s.keys(root, len(slides))
	if s.start.IsZero() {
		s.start = c.Now()
	}
	white, muted := ui.RGB(240, 240, 242), ui.RGB(150, 150, 158)
	start := s.start
	root.Children(func() {
		ui.Row(c).Gap(t.Space(6)).AlignItems(ui.Center).Children(func() {
			ui.Box(c).Size(t.Space(30), t.Space(9)).Draw(func(p *ui.Painter, r ui.Rect) {
				d := p.Now().Sub(start)
				span := ui.Span{Text: clock(d), Size: 28, Weight: 600, Color: ui.RGB(255, 196, 0), Features: "tnum"}
				_, h := p.MeasureText(0, span)
				p.RichText(r.X, r.Y+r.H/2-h/2, 0, span)
				p.After(time.Second - d%time.Second)
			})
			ui.Box(c).Size(t.Space(24), t.Space(9)).Draw(func(p *ui.Painter, r ui.Rect) {
				now := p.Now()
				span := ui.Span{Text: now.Format("15:04"), Size: 28, Color: white, Features: "tnum"}
				_, h := p.MeasureText(0, span)
				p.RichText(r.X, r.Y+r.H/2-h/2, 0, span)
				p.After(time.Minute - time.Duration(now.Second())*time.Second)
			})
			if ui.Button(c, "Restart").Clicked() {
				s.start = c.Now()
			}
			ui.Spacer(c)
			ui.Text(c, fmt.Sprintf("Slide %d of %d", s.Current+1, len(slides))).TextColor(muted).FontFeatures("tnum")
		})
		ui.Row(c).Grow(1).Gap(t.Space(6)).Children(func() {
			now := slides[s.Current]
			ui.Box(c).Grow(2).FillHeight().Draw(func(p *ui.Painter, r ui.Rect) { Paint(p, r, now) })
			ui.Column(c).Grow(1).FillHeight().Gap(t.Space(3)).Children(func() {
				ui.Text(c, "Next").TextColor(muted).FontSize(t.FontSize - 1)
				if s.Current+1 < len(slides) {
					next := slides[s.Current+1]
					ui.Box(c).FillWidth().AspectRatio(Width / float32(Height)).Draw(func(p *ui.Painter, r ui.Rect) { Paint(p, r, next) })
				} else {
					ui.Box(c).FillWidth().AspectRatio(Width/float32(Height)).Border(1, muted.Alpha(0.4)).Children(func() {
						ui.Text(c, "End of show").TextColor(muted).Center()
					})
				}
				ui.Text(c, "Notes").TextColor(muted).FontSize(t.FontSize-1).Margin(t.Space(2), 0, 0, 0)
				ui.Scroll(c).Grow(1).Children(func() {
					ui.Text(c, now.Notes).TextColor(white).FontSize(t.FontSize + 4).LineHeight(1.4)
				})
			})
		})
	})
	return Events{Element: root, Ended: ended}
}

// clock returns d as minutes and seconds, with hours past an hour.
func clock(d time.Duration) string {
	sec := int(d / time.Second)
	if sec >= 3600 {
		return fmt.Sprintf("%d:%02d:%02d", sec/3600, sec/60%60, sec%60)
	}
	return fmt.Sprintf("%02d:%02d", sec/60, sec%60)
}

// Show creates the slide shown alone, on black, to fill the screen.
func Show(c *ui.Context, s *State, slides []Slide) Events {
	view := ui.Box(c.Key("slides.show")).Focusable().FocusRing(false).AutoFocus().Background(ui.RGB(0, 0, 0)).Role(ui.RoleImage).Label("Slide")
	if len(slides) == 0 {
		return Events{Element: view}
	}
	ended := s.keys(view, len(slides))
	if view.Clicked() {
		s.Current = min(len(slides)-1, s.Current+1)
	}
	if s.start.IsZero() {
		s.start = c.Now()
	}
	slide := slides[s.Current]
	view.Draw(func(p *ui.Painter, r ui.Rect) { Paint(p, r, slide) })
	return Events{Element: view, Ended: ended}
}
