// Package arena compares two answers to one prompt side by side, as model
// arenas ask people to: the answers as Model A and Model B, their names
// hidden, and buttons to vote for A, for B, a tie, or both bad. The
// number keys 1 to 4 vote too, as do Left and Right for A and B, while
// the arena has the focus, which it takes as it shows. Voted, it names
// the models and marks each with the vote. Changed reports the vote.
//
//	arena.View(c, &round.vote, round.models, func() {
//		ui.Text(c, round.answers[0])
//	}, func() {
//		ui.Text(c, round.answers[1])
//	})
package arena

import "github.com/egoist/mygo/ui"

// Vote is the user's vote.
type Vote uint8

const (
	None Vote = iota
	A
	B
	Tie
	BothBad
)

var votes = []struct {
	vote       Vote
	label, key string
	keys       []ui.Key
}{
	{A, "A is better", "1", []ui.Key{ui.Key1, ui.KeyLeft}},
	{Tie, "Tie", "3", []ui.Key{ui.Key3}},
	{BothBad, "Both are bad", "4", []ui.Key{ui.Key4}},
	{B, "B is better", "2", []ui.Key{ui.Key2, ui.KeyRight}},
}

// View creates the arena of the answers a and b build, by the models
// named names, and, while *v is None, the buttons setting it.
func View(c *ui.Context, v *Vote, names [2]string, a, b func()) ui.Element {
	t := c.Theme()
	before := *v
	root := ui.Column(c).Gap(t.Space(4)).Role(ui.RoleGroup).Label("Arena")
	if *v == None {
		root.Focusable().FocusRing(false).AutoFocus()
		for _, o := range votes {
			for _, k := range o.keys {
				if root.Shortcut(0, k) {
					*v = o.vote
				}
			}
		}
	}
	root.Children(func() {
		ui.Row(c).Gap(t.Space(4)).AlignItems(ui.Stretch).Children(func() {
			for i, build := range []func(){a, b} {
				side := [2]Vote{A, B}[i]
				pane := ui.Column(c.Key(i)).Grow(1).Basis(0).Shrink(1).MinWidth(0).Gap(t.Space(3)).Padding(t.Space(4)).
					Radius(t.Space(2.5)).Border(1, t.Border).Label("Model " + "AB"[i:i+1])
				verdict, color := "", t.TextMuted
				switch *v {
				case side:
					verdict, color = "Preferred", t.Success
					pane.BorderColor(t.Success).Background(t.Success.Alpha(0.04))
				case Tie:
					verdict = "Tie"
				case BothBad:
					verdict, color = "Bad", t.Danger
				}
				pane.Children(func() {
					ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
						ui.Text(c, "Model "+"AB"[i:i+1]).Bold()
						if *v != None {
							ui.Text(c, names[i]).TextColor(t.TextMuted).SingleLine().Shrink(1)
						}
						ui.Box(c).Grow(1)
						if verdict != "" {
							ui.Text(c, verdict).FontSize(t.FontSize - 2).Bold().TextColor(color)
						}
					})
					if build != nil {
						build()
					}
				})
			}
		})
		if *v != None {
			return
		}
		ui.Row(c).Gap(t.Space(2)).Justify(ui.Center).Children(func() {
			for _, o := range votes {
				b := ui.Button(c.Key(o.vote), "").Label(o.label).Children(func() {
					ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
						if o.vote == A {
							ui.Text(c, "←")
						}
						ui.Text(c, o.label)
						if o.vote == B {
							ui.Text(c, "→")
						}
						ui.Text(c, o.key).FontSize(t.FontSize-2).FontFeatures("tnum").TextColor(t.TextMuted).
							Padding(0, t.Space(1.5)).Radius(t.Space(1)).Border(1, t.Border)
					})
				})
				if b.Clicked() {
					*v = o.vote
				}
			}
		})
	})
	if *v != before {
		root.SetChanged()
	}
	return root
}
