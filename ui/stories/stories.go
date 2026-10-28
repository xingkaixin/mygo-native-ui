// Package stories shows people's stories, as social apps do. Tray is the
// row of the people with stories, each a picture in a ring, colored while
// some of their stories are unseen and gray once all are; a click opens
// their stories.
//
// View shows one story at a time on a tall card, the app building what it
// shows, under a bar for each of the person's stories that fills as the
// story plays and the person's name and how long ago they posted it. A
// story moves on to the next when its time is up, through the person's
// stories and then the next person's. A click on the right of the card
// goes to the next story, and on the left to the one before; holding the
// pointer down, or Space, pauses it; ← and → step, and Escape or × closes
// it, as does the end of the last story.
//
//	if stories.Tray(c, &app.stories, app.people).Opened {
//		app.viewing = true
//	}
//	if app.viewing && stories.View(c, &app.stories, &app.people, app.story).Closed {
//		app.viewing = false
//	}
package stories

import (
	"fmt"
	"time"

	"github.com/egoist/mygo/ui"
)

// Story is a story: how long it shows, when it was posted, and whether
// the user has seen it.
type Story struct {
	Length time.Duration
	Posted time.Time
	Seen   bool
}

// Person is someone with stories, and their picture, or nil for their
// initials.
type Person struct {
	Name    string
	Picture *ui.Bitmap
	Stories []Story
}

func (p Person) unseen() bool {
	for _, s := range p.Stories {
		if !s.Seen {
			return true
		}
	}
	return false
}

// State is what stories keep: the story shown, how long it has played and
// whether it is paused. Keep it in the app's state, one for each tray and
// its viewer.
type State struct {
	person, story int
	played        time.Duration
	since         time.Time
	paused        bool
	pressed       time.Time
}

// Events is a tray or a viewer, and whether a person's stories were
// opened or the viewer closed in the frame.
type Events struct {
	ui.Element
	Opened, Closed bool
}

const tap = 250 * time.Millisecond

var ringColors = ui.ConicGradient{Stops: []ui.GradientStop{
	{Color: ui.Hex("#F9CE34"), At: 0}, {Color: ui.Hex("#EE2A7B"), At: 0.35},
	{Color: ui.Hex("#6228D7"), At: 0.7}, {Color: ui.Hex("#F9CE34"), At: 1},
}}

// Tray creates the row of people.
func Tray(c *ui.Context, s *State, people []Person) Events {
	t := c.Theme()
	var ev Events
	size := t.Space(16)
	row := ui.Row(c).Gap(t.Space(4)).Label("Stories").Role(ui.RoleList)
	row.Children(func() {
		for i, p := range people {
			unseen := p.unseen()
			e := ui.ButtonBase(c.Key(i)).Column().Gap(t.Space(1.5)).AlignItems(ui.Center).Width(size + t.Space(4)).
				Label(p.Name).Cursor(ui.CursorPointer)
			if unseen {
				e.Description("New stories")
			}
			if e.Clicked() {
				first := 0
				for k, st := range p.Stories {
					if !st.Seen {
						first = k
						break
					}
				}
				*s = State{person: i, story: first, since: c.Now()}
				ev.Opened = true
			}
			e.Children(func() {
				ring := ui.Box(c).Size(size, size).Radius(size / 2).Center()
				if unseen {
					ring.ConicGradient(ringColors)
				} else {
					ring.Background(t.Border)
				}
				ring.Children(func() {
					in := size - 6
					ui.Box(c).Size(in, in).Radius(in / 2).Background(t.Background).Center().Children(func() {
						ui.Avatar(c, p.Name, p.Picture).Size(in-4, in-4).Radius((in - 4) / 2)
					})
				})
				name := ui.Text(c, p.Name).FontSize(t.FontSize - 2).Ellipsis("…").MaxWidth(size + t.Space(4))
				if !unseen {
					name.TextColor(t.TextMuted)
				}
			})
		}
	})
	ev.Element = row
	return ev
}

// View creates the viewer of *people's stories, build making what the
// story of the person shows, on the card.
func View(c *ui.Context, s *State, people *[]Person, build func(person, story int)) Events {
	t := c.Theme()
	var ev Events
	ps := *people
	now := c.Now()
	if s.since.IsZero() {
		s.since = now
	}
	if s.person >= len(ps) || s.story >= len(ps[s.person].Stories) {
		ev.Closed = true
		ev.Element = ui.Box(c)
		return ev
	}
	played := func() time.Duration {
		if s.paused {
			return s.played
		}
		return s.played + now.Sub(s.since)
	}
	step := func(dir int) {
		s.played, s.since = 0, now
		s.story += dir
		switch {
		case s.story < 0 && s.person > 0:
			s.person--
			s.story = len(ps[s.person].Stories) - 1
		case s.story < 0:
			s.story = 0
		case s.story >= len(ps[s.person].Stories):
			s.person, s.story = s.person+1, 0
			if s.person >= len(ps) {
				s.person = len(ps) - 1
				s.story = len(ps[s.person].Stories) - 1
				ev.Closed = true
			}
		}
	}
	story := ps[s.person].Stories[s.story]
	if played() >= story.Length {
		step(1)
	}
	if ev.Closed {
		ev.Element = ui.Box(c)
		return ev
	}
	(*people)[s.person].Stories[s.story].Seen = true
	story = ps[s.person].Stories[s.story]
	h := t.Space(150)
	card := ui.Box(c).Size(h*9/16, h).Radius(t.Space(4)).Clip().Background(ui.Hex("#111111")).
		Focusable().FocusRing(false).AutoFocus().Label(fmt.Sprintf("Story %d of %d by %s", s.story+1, len(ps[s.person].Stories), ps[s.person].Name))
	pause := func(on bool) {
		if on == s.paused {
			return
		}
		s.played, s.since, s.paused = played(), now, on
	}
	switch {
	case card.Shortcut(0, ui.KeyRight):
		step(1)
	case card.Shortcut(0, ui.KeyLeft):
		step(-1)
	case card.Shortcut(0, ui.KeySpace):
		pause(!s.paused)
	case card.Shortcut(0, ui.KeyEscape):
		ev.Closed = true
	}
	held := card.Pressed()
	if held && s.pressed.IsZero() {
		s.pressed = now
	}
	if held && now.Sub(s.pressed) >= tap {
		pause(true)
	}
	if card.Clicked() {
		if now.Sub(s.pressed) < tap {
			px, _, _ := card.PointerPosition()
			if px < card.Bounds().W/3 {
				step(-1)
			} else {
				step(1)
			}
		} else {
			pause(false)
		}
	}
	if !held {
		s.pressed = time.Time{}
	}
	if !s.paused && !ev.Closed {
		c.After(ps[s.person].Stories[s.story].Length - played())
	}
	person, at := s.person, s.story
	count := len(ps[person].Stories)
	base, since, paused := s.played, s.since, s.paused
	card.Children(func() {
		ui.Box(c.Key(fmt.Sprint(person, "/", at))).Fill().Children(func() {
			build(person, at)
		})
		ui.Column(c).Absolute().Top(0).Left(0).Right(0).Padding(t.Space(2.5)).Gap(t.Space(2.5)).
			LinearGradient(ui.LinearGradient{From: ui.RGBA(0, 0, 0, 0.45), To: ui.RGBA(0, 0, 0, 0), Angle: 180}).Children(func() {
			ui.Box(c).FillWidth().Height(3).Draw(func(p *ui.Painter, r ui.Rect) {
				gap := float32(4)
				w := (r.W - gap*float32(count-1)) / float32(count)
				elapsed := base
				if !paused {
					elapsed += p.Now().Sub(since)
					p.AnimationFrame()
				}
				for k := range count {
					x := r.X + float32(k)*(w+gap)
					p.Fill(ui.Rect{X: x, Y: r.Y, W: w, H: r.H}, ui.RGBA(255, 255, 255, 0.35), r.H/2)
					f := float32(0)
					switch {
					case k < at:
						f = 1
					case k == at:
						f = min(1, float32(elapsed)/float32(max(1, story.Length)))
					}
					if f > 0 {
						p.Fill(ui.Rect{X: x, Y: r.Y, W: w * f, H: r.H}, ui.RGB(255, 255, 255), r.H/2)
					}
				}
			})
			ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
				ui.Avatar(c, ps[person].Name, ps[person].Picture).Size(t.Space(8), t.Space(8)).Radius(t.Space(4))
				ui.Text(c, ps[person].Name).Bold().TextColor(ui.RGB(255, 255, 255))
				ui.Text(c, ago(now.Sub(story.Posted))).TextColor(ui.RGBA(255, 255, 255, 0.7)).FontSize(t.FontSize - 1)
				ui.Box(c).Grow(1)
				if paused {
					ui.Text(c, "Paused").FontSize(t.FontSize - 2).TextColor(ui.RGBA(255, 255, 255, 0.8))
				}
				x := ui.ButtonBase(c).Size(t.Space(7), t.Space(7)).Center().Label("Close").Cursor(ui.CursorPointer)
				if x.Clicked() {
					ev.Closed = true
				}
				x.Children(func() {
					ui.Text(c, "×").FontSize(t.FontSize + 8).TextColor(ui.RGB(255, 255, 255))
				})
			})
		})
	})
	ev.Element = card
	return ev
}

// ago writes how long ago d was, as "5m" or "3h".
func ago(d time.Duration) string {
	switch {
	case d < time.Minute:
		return "now"
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d/time.Minute))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d/time.Hour))
	}
	return fmt.Sprintf("%dd", int(d/(24*time.Hour)))
}
