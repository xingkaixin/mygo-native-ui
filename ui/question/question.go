// Package question is a question an agent asks the user, as coding agents
// ask before choosing for them: a card with a short header, the question,
// and its options, each a label over a line on what it means, chosen one
// at a time, or several when the question takes several, and a last row
// to answer in the user's own words. An option with a preview shows it
// beside the options while it is under the pointer or the arrows, as the
// code or the layout it would make. Up and Down move among the options,
// Space and the number keys choose them, and Enter or Submit answers; the
// card then shows the answer. Changed reports the answer.
//
//	question.Ask(c, &app.answer, question.Question{
//		Header: "Sign-in",
//		Text:   "Which sign-in should the app offer?",
//		Options: []question.Option{
//			{Label: "Email link", Description: "No passwords to keep"},
//			{Label: "Passkeys", Description: "Face or fingerprint, on each device"},
//		},
//	})
//	if app.answer.Done {
//		... // go on with app.answer
//	}
package question

import (
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
)

// Option is an answer the agent offers: its label, a line on what it
// means, and what shows beside the options while it is under the pointer
// or the arrows, nil for nothing.
type Option struct {
	Label, Description string
	Preview            func()
}

// Question is what the agent asks: a header of a word or two, the
// question, its options, and whether the user may choose several.
type Question struct {
	Header, Text string
	Options      []Option
	Multi        bool
}

// Answer is the user's answer: the options chosen, by index, the answer
// in their own words, "" for none, and whether they gave it. Its zero
// value is no answer yet.
type Answer struct {
	Chosen []int
	Other  string
	Done   bool
}

func (a *Answer) chosen(i int) bool {
	for _, k := range a.Chosen {
		if k == i {
			return true
		}
	}
	return false
}

// choose chooses option i, or takes it back from a question taking
// several; one choice clears the user's own words.
func (a *Answer) choose(i int, multi bool) {
	switch {
	case !multi:
		a.Chosen, a.Other = []int{i}, ""
	case a.chosen(i):
		var kept []int
		for _, k := range a.Chosen {
			if k != i {
				kept = append(kept, k)
			}
		}
		a.Chosen = kept
	default:
		a.Chosen = append(a.Chosen, i)
	}
}

func (a *Answer) any() bool {
	return len(a.Chosen) > 0 || strings.TrimSpace(a.Other) != ""
}

var digits = []ui.Key{ui.Key1, ui.Key2, ui.Key3, ui.Key4, ui.Key5, ui.Key6, ui.Key7, ui.Key8, ui.Key9}

// state is what a card keeps: the row the arrows are on, the last under
// the pointer, and where the focus goes next frame.
type state struct {
	on      int
	hovered int
	toInput bool
	toCard  bool
}

// Ask creates the card asking q, answered into *a.
func Ask(c *ui.Context, a *Answer, q Question) ui.Element {
	t := c.Theme()
	before := a.Done
	card := ui.Column(c).Gap(t.Space(3)).Padding(t.Space(3.5)).Radius(t.Space(2.5)).Border(1, t.Border).
		Background(t.Background).Role(ui.RoleGroup).Label(q.Text)
	st := ui.Local(card, "question.state", func() state { return state{hovered: -1} })
	n := len(q.Options)
	other := n
	submit := func() {
		if a.any() {
			a.Done = true
		}
	}
	if !a.Done {
		card.BorderColor(t.Accent.Alpha(0.6)).Focusable().FocusRing(false).AutoFocus()
		if st.toCard {
			card.Focus()
			st.toCard = false
		}
		switch {
		case card.Shortcut(0, ui.KeyDown):
			st.on = min(st.on+1, other)
			st.toInput = st.on == other
		case card.Shortcut(0, ui.KeyUp):
			st.on = max(st.on-1, 0)
		case card.Shortcut(0, ui.KeySpace) && st.on < n:
			a.choose(st.on, q.Multi)
		case card.Shortcut(0, ui.KeyEnter) && st.on < n:
			if !q.Multi {
				a.choose(st.on, false)
			}
			submit()
		}
		for i, k := range digits[:min(n, len(digits))] {
			if card.Shortcut(0, k) {
				st.on = i
				a.choose(i, q.Multi)
			}
		}
	}
	card.Children(func() {
		ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
			if q.Header != "" {
				ui.Text(c, q.Header).FontSize(t.FontSize-2).Bold().TextColor(t.Accent).Padding(t.Space(0.5), t.Space(2)).
					Radius(t.Space(1)).Background(t.Accent.Alpha(0.12))
			}
			ui.Text(c, q.Text).Bold().Grow(1).Shrink(1)
		})
		if a.Done {
			answered(c, a, q)
			return
		}
		preview := false
		for _, o := range q.Options {
			preview = preview || o.Preview != nil
		}
		hovered := -1
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Stretch).Children(func() {
			list := ui.Column(c).Gap(t.Space(1)).Role(ui.RoleList)
			if preview {
				list.Width(t.Space(72))
			} else {
				list.Grow(1)
			}
			list.Children(func() {
				for i, o := range q.Options {
					row := ui.ButtonBase(c.Key(i)).Justify(ui.Start).AlignItems(ui.Start).Gap(t.Space(2.5)).
						Padding(t.Space(2), t.Space(2.5)).Radius(t.Space(1.5)).Label(o.Label).Checked(a.chosen(i)).
						KeepFocus().FocusRing(false)
					if row.Hovered() {
						hovered = i
					}
					if row.Clicked() {
						st.on = i
						a.choose(i, q.Multi)
						st.toCard = true
					}
					if i == st.on {
						row.Background(t.Text.Alpha(0.06))
					}
					row.Children(func() {
						mark(c, a.chosen(i), q.Multi)
						ui.Column(c).Gap(t.Space(0.5)).Grow(1).Shrink(1).Children(func() {
							ui.RichText(c, ui.Span{Text: strconv.Itoa(i+1) + ". ", Color: t.TextMuted}, ui.Span{Text: o.Label, Weight: 600})
							if o.Description != "" {
								ui.Text(c, o.Description).FontSize(t.FontSize - 1).TextColor(t.TextMuted)
							}
						})
					})
				}
				own := ui.Row(c.Key(other)).Gap(t.Space(2.5)).Padding(t.Space(2), t.Space(2.5)).Radius(t.Space(1.5)).
					AlignItems(ui.Center)
				if own.Hovered() {
					hovered = other
				}
				if st.on == other {
					own.Background(t.Text.Alpha(0.06))
				}
				own.Children(func() {
					mark(c, strings.TrimSpace(a.Other) != "", q.Multi)
					in := ui.TextInputBase(c, &a.Other).Grow(1).Placeholder("Other…").Label("Other answer")
					in.OnTextKey(func(k ui.TextKey) bool {
						if k.Key == ui.KeyUp && k.Mods == 0 {
							st.on, st.toCard = max(n-1, 0), true
							return true
						}
						return false
					})
					if st.toInput {
						in.Focus()
						st.toInput = false
					}
					if in.Focused() {
						st.on = other
					}
					if in.Changed() && !q.Multi && strings.TrimSpace(a.Other) != "" {
						a.Chosen = nil
					}
					if in.Submitted() {
						submit()
					}
				})
			})
			if preview {
				i := st.on
				if i >= n || q.Options[i].Preview == nil {
					i = -1
					for _, k := range a.Chosen {
						if q.Options[k].Preview != nil {
							i = k
							break
						}
					}
				}
				ui.Column(c).Grow(1).Shrink(1).MinWidth(0).Padding(t.Space(3)).Radius(t.Space(2)).Background(t.Surface).
					Clip().Label("Preview").Children(func() {
					if i >= 0 {
						q.Options[i].Preview()
					}
				})
			}
		})
		// The pointer moving onto a row puts the arrows there; resting
		// there, it leaves the arrows to move on.
		if hovered >= 0 && hovered != st.hovered {
			st.on = hovered
		}
		st.hovered = hovered
		ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
			hint := "↑↓ to move · Enter to choose"
			if q.Multi {
				hint = "↑↓ to move · Space to choose · Enter to submit"
			}
			ui.Text(c, hint).FontSize(t.FontSize - 2).TextColor(t.TextMuted).Grow(1).Shrink(1)
			if ui.PrimaryButton(c, "Submit").Disabled(!a.any()).Clicked() {
				submit()
			}
		})
	})
	if a.Done != before {
		card.SetChanged()
	}
	return card
}

// answered shows the answer a gave q.
func answered(c *ui.Context, a *Answer, q Question) {
	t := c.Theme()
	var parts []string
	for _, i := range a.Chosen {
		parts = append(parts, q.Options[i].Label)
	}
	if s := strings.TrimSpace(a.Other); s != "" {
		parts = append(parts, "“"+s+"”")
	}
	ui.Row(c).Gap(t.Space(1.5)).AlignItems(ui.Center).Children(func() {
		ui.Box(c).Size(t.FontSize, t.FontSize).Draw(func(p *ui.Painter, r ui.Rect) {
			cx, cy, k := r.X+r.W/2, r.Y+r.H/2, r.W*0.3
			p.StrokePolyline([][2]float32{{cx - k, cy}, {cx - k/3, cy + k*0.7}, {cx + k, cy - k*0.7}}, 1.75, t.Success)
		})
		ui.Text(c, strings.Join(parts, ", ")).TextColor(t.TextMuted).Shrink(1)
	})
}

// mark shows a radio button, or a checkbox for a question taking several,
// on when on.
func mark(c *ui.Context, on, multi bool) {
	t := c.Theme()
	ui.Box(c).Size(t.Space(4), t.Space(4)).Margin(t.Space(0.5), 0, 0, 0).Draw(func(p *ui.Painter, r ui.Rect) {
		radius := r.W / 2
		if multi {
			radius = t.Space(1)
		}
		if !on {
			p.Stroke(r, t.Border.Mix(t.Text, 0.25), radius, 1.5)
			return
		}
		p.Fill(r, t.Accent, radius)
		cx, cy := r.X+r.W/2, r.Y+r.H/2
		if !multi {
			p.Fill(ui.Rect{X: cx - r.W/5, Y: cy - r.W/5, W: r.W * 0.4, H: r.W * 0.4}, t.AccentText, r.W/5)
			return
		}
		k := r.W * 0.28
		p.StrokePolyline([][2]float32{{cx - k, cy}, {cx - k/3, cy + k*0.7}, {cx + k, cy - k*0.7}}, 1.75, t.AccentText)
	})
}
