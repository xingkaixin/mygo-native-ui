// Package poll is a poll in a conversation, as chat apps post one: the
// question, and its options to choose from, a circle before each, or a
// box where several may be chosen. Once the user votes, or the poll
// closes, each option fills with its share of the votes, its percentage
// before it and a check after those the user chose, and the votes are
// counted below; a closed poll bolds the leading option.
//
//	ev := poll.View(c, &app.card, &app.poll)
//	if ev.Voted || ev.Retracted {
//		app.send(app.poll.Mine)
//	}
//
// With one choice, a click on an option votes for it. With several, clicks
// tick options and Vote votes for them. Retract takes the vote back, to
// choose again.
package poll

import (
	"fmt"
	"slices"
	"time"

	"github.com/egoist/mygo/ui"
)

// Option is an option of a poll and its votes, the user's among them.
type Option struct {
	Text  string
	Votes int
}

// Poll is a poll: its question and options, whether several may be
// chosen, whether it is closed, and the options the user voted for, by
// index, none before voting.
type Poll struct {
	Question string
	Options  []Option
	Multiple bool
	Closed   bool
	Mine     []int
}

// State is what a poll keeps: the options ticked before voting. Keep it
// in the app's state, one for each poll.
type State struct {
	ticked map[int]bool
}

// Events is a poll, and whether the user voted or took the vote back this
// frame, which the app sends on.
type Events struct {
	ui.Element
	Voted, Retracted bool
}

// View creates the poll p, voting on it as the user does.
func View(c *ui.Context, s *State, p *Poll) Events {
	t := c.Theme()
	if s.ticked == nil {
		s.ticked = map[int]bool{}
	}
	card := ui.Column(c).Gap(t.Space(2)).Padding(t.Space(4)).Radius(t.Space(3)).Background(t.Surface).Border(1, t.Border)
	ev := Events{Element: card}
	results := len(p.Mine) > 0 || p.Closed
	total, top := 0, 0
	for _, o := range p.Options {
		total += o.Votes
		top = max(top, o.Votes)
	}
	vote := func(chosen []int) {
		p.Mine = chosen
		for _, i := range chosen {
			p.Options[i].Votes++
		}
		clear(s.ticked)
		ev.Voted = true
	}
	card.Children(func() {
		ui.Text(c, p.Question).Bold().FontSize(t.FontSize + 1)
		hint := "Choose one"
		switch {
		case p.Closed:
			hint = "Final results"
		case results:
			hint = "You voted"
		case p.Multiple:
			hint = "Choose any"
		}
		ui.Text(c, hint).FontSize(t.FontSize-2).TextColor(t.TextMuted).Padding(0, 0, t.Space(1), 0)
		for i, o := range p.Options {
			row := ui.ButtonBase(c.Key(i)).Gap(t.Space(2.5)).Padding(t.Space(2), t.Space(3)).Radius(t.Space(2)).
				Justify(ui.Start).AlignItems(ui.Center).Label(o.Text)
			share := float32(0)
			if results && total > 0 {
				share = float32(o.Votes) / float32(total)
			}
			fill := row.Animate("poll.share", share, 450*time.Millisecond)
			mine := slices.Contains(p.Mine, i)
			lead := p.Closed && o.Votes == top && top > 0
			if results {
				// A clear border keeps the rows as tall as before the vote.
				row.Cursor(ui.CursorDefault).Border(1, ui.Color{})
				color := t.Text.Alpha(0.08)
				if mine || lead {
					color = t.Accent.Alpha(0.22)
				}
				row.Draw(func(pa *ui.Painter, r ui.Rect) {
					pa.Fill(ui.Rect{X: r.X, Y: r.Y, W: r.W * fill, H: r.H}, color, t.Space(2))
				})
			} else {
				row.Border(1, t.Border)
				if row.Hovered() {
					row.Background(t.Text.Alpha(0.04))
				}
				if s.ticked[i] {
					row.Border(1, t.Accent)
				}
			}
			if row.Clicked() && !results {
				if p.Multiple {
					s.ticked[i] = !s.ticked[i]
				} else {
					vote([]int{i})
				}
			}
			row.Children(func() {
				if results {
					ui.Text(c, fmt.Sprintf("%.0f%%", share*100)).Width(t.Space(10)).FontSize(t.FontSize - 1).FontWeight(600).FontFeatures("tnum")
				} else {
					ticked := s.ticked[i]
					multiple := p.Multiple
					ui.Box(c).Size(16, 16).Draw(func(pa *ui.Painter, r ui.Rect) { choice(pa, r, multiple, ticked, t) })
				}
				txt := ui.Text(c, o.Text).Grow(1)
				if lead {
					txt.Bold()
				}
				if mine {
					ui.Box(c).Size(14, 14).Draw(func(pa *ui.Painter, r ui.Rect) {
						pa.StrokePolyline([][2]float32{{r.X + 2, r.Y + 7}, {r.X + 6, r.Y + 11}, {r.X + 12, r.Y + 3}}, 2, t.Accent)
					})
				}
			})
		}
		ui.Row(c).AlignItems(ui.Center).Gap(t.Space(2)).Padding(t.Space(1), 0, 0).Children(func() {
			count := fmt.Sprintf("%d votes", total)
			if total == 1 {
				count = "1 vote"
			}
			ui.Text(c, count).FontSize(t.FontSize - 1).TextColor(t.TextMuted).Grow(1)
			switch {
			case p.Closed:
			case results:
				back := ui.ButtonBase(c).Padding(t.Space(1), t.Space(2)).Radius(t.Space(1.5)).Label("Retract vote")
				if back.Hovered() {
					back.Background(t.Text.Alpha(0.06))
				}
				back.Children(func() { ui.Text(c, "Retract vote").FontSize(t.FontSize - 1).TextColor(t.Accent) })
				if back.Clicked() {
					for _, i := range p.Mine {
						p.Options[i].Votes--
					}
					p.Mine = nil
					ev.Retracted = true
				}
			case p.Multiple:
				var chosen []int
				for i := range p.Options {
					if s.ticked[i] {
						chosen = append(chosen, i)
					}
				}
				if ui.Button(c, "Vote").Disabled(len(chosen) == 0).Clicked() {
					vote(chosen)
				}
			}
		})
	})
	return ev
}

// choice draws the circle or box before an option, filled when ticked.
func choice(p *ui.Painter, r ui.Rect, box, ticked bool, t *ui.Theme) {
	cx, cy := r.X+r.W/2, r.Y+r.H/2
	if box {
		b := ui.Rect{X: r.X + 1, Y: r.Y + 1, W: r.W - 2, H: r.H - 2}
		if ticked {
			p.Fill(b, t.Accent, 4)
			p.StrokePolyline([][2]float32{{cx - 4, cy}, {cx - 1, cy + 3}, {cx + 4, cy - 3}}, 2, t.AccentText)
			return
		}
		p.Stroke(b, t.TextMuted, 4, 1.5)
		return
	}
	var ring ui.Path
	p.StrokePath(ring.Circle(cx, cy, r.W/2-1), 1.5, t.TextMuted)
}
