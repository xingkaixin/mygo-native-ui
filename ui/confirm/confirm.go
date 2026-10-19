// Package confirm asks the user to approve what an agent is about to do,
// as coding agents ask before running a command or writing a file: a
// card with what it will do, and buttons to allow it once, always, or
// deny it, which the card then shows as answered.
//
//	confirm.Ask(c, &call.decision, "Run this command?", "Always allow go test", func() {
//		ui.Text(c, call.command).Font("monospace")
//	})
//	if call.decision == confirm.Allow || call.decision == confirm.AlwaysAllow {
//		... // run it, once
//	}
package confirm

import "github.com/egoist/mygo/ui"

// Decision is the user's answer.
type Decision uint8

const (
	Undecided Decision = iota
	Allow
	AlwaysAllow
	Deny
)

// Ask creates a card asking title, with what body builds below it, as the
// command to run, and, while *d is Undecided, buttons setting it: Deny,
// always unless "", and Allow, which takes the keyboard focus as the card
// shows, so that Enter allows the card that came last; Escape denies.
// Answered, the card shows the answer. Changed reports the answer.
func Ask(c *ui.Context, d *Decision, title, always string, body func()) ui.Element {
	t := c.Theme()
	before := *d
	card := ui.Column(c).Gap(t.Space(2.5)).Padding(t.Space(3)).Radius(t.Space(2.5)).
		Border(1, t.Border).Background(t.Background).Role(ui.RoleGroup).Label(title)
	if *d == Undecided {
		card.Border(1, t.Accent.Alpha(0.6))
		if card.Shortcut(0, ui.KeyEscape) {
			*d = Deny
		}
	}
	card.Children(func() {
		ui.Text(c, title).Bold()
		if body != nil {
			body()
		}
		if *d != Undecided {
			answer(c, *d)
			return
		}
		ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "Enter to allow · Esc to deny").FontSize(t.FontSize - 2).TextColor(t.TextMuted).Grow(1).Shrink(1)
			if ui.Button(c, "Deny").Clicked() {
				*d = Deny
			}
			if always != "" && ui.Button(c, always).Clicked() {
				*d = AlwaysAllow
			}
			if ui.PrimaryButton(c, "Allow").AutoFocus().Clicked() {
				*d = Allow
			}
		})
	})
	if *d != before {
		card.SetChanged()
	}
	return card
}

// answer shows d, a mark and words.
func answer(c *ui.Context, d Decision) {
	t := c.Theme()
	label, color := "Allowed", t.Success
	switch d {
	case AlwaysAllow:
		label = "Always allowed"
	case Deny:
		label, color = "Denied", t.Danger
	}
	ui.Row(c).Gap(t.Space(1.5)).AlignItems(ui.Center).Children(func() {
		ui.Box(c).Size(t.FontSize, t.FontSize).Draw(func(p *ui.Painter, r ui.Rect) {
			cx, cy, k := r.X+r.W/2, r.Y+r.H/2, r.W*0.3
			if d == Deny {
				p.StrokePolyline([][2]float32{{cx - k, cy - k}, {cx + k, cy + k}}, 1.75, color)
				p.StrokePolyline([][2]float32{{cx + k, cy - k}, {cx - k, cy + k}}, 1.75, color)
				return
			}
			p.StrokePolyline([][2]float32{{cx - k, cy}, {cx - k/3, cy + k*0.7}, {cx + k, cy - k*0.7}}, 1.75, color)
		})
		ui.Text(c, label).TextColor(t.TextMuted)
	})
}
