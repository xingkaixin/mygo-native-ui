// Package reactions shows the reactions to a message, as Slack and GitHub
// show them under one: a chip for each emoji with how many reacted, lit
// while the user is among them, the pointer over it naming who did, and a
// click adding the user's reaction or taking it back. A button after the
// chips opens package emoji's picker, where the emoji picked adds a
// reaction. Changed reports a reaction added or taken back.
//
//	reactions.View(c, &app.picker, "Ada", &msg.Reactions)
package reactions

import (
	"slices"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/emoji"
)

// Reaction is an emoji and who reacted with it, in the order they did.
type Reaction struct {
	Emoji string
	Who   []string
}

// State is what the reactions of an app keep: the picker the add buttons
// open. Keep it in the app's state, one for all the messages.
type State struct {
	// Picker is the state of the picker, whose Recent are the emoji
	// picked lately. Save it to keep them.
	Picker emoji.State
}

// View creates the reactions *rs to a message, for the user named me.
func View(c *ui.Context, s *State, me string, rs *[]Reaction) ui.Element {
	t := c.Theme()
	root := ui.Row(c).Gap(t.Space(1.5)).Wrap().Label("Reactions")
	changed := false
	react := func(e string) {
		i := slices.IndexFunc(*rs, func(r Reaction) bool { return r.Emoji == e })
		switch {
		case i < 0:
			*rs = append(*rs, Reaction{Emoji: e, Who: []string{me}})
		case slices.Contains((*rs)[i].Who, me):
			(*rs)[i].Who = slices.DeleteFunc(slices.Clone((*rs)[i].Who), func(w string) bool { return w == me })
			if len((*rs)[i].Who) == 0 {
				*rs = slices.Delete(*rs, i, i+1)
			}
		default:
			(*rs)[i].Who = append((*rs)[i].Who, me)
		}
		changed = true
	}
	root.Children(func() {
		for _, r := range *rs {
			mine := slices.Contains(r.Who, me)
			chip := ui.ButtonBase(c.Key(r.Emoji)).Gap(t.Space(1)).Padding(t.Space(0.5), t.Space(2)).Radius(t.Space(4)).
				Border(1, t.Border).Background(t.Surface).Label(r.Emoji + " " + strconv.Itoa(len(r.Who))).Checked(mine).
				Tooltip(who(r.Who, me))
			count := t.TextMuted
			if mine {
				chip.Background(t.Accent.Alpha(0.12)).BorderColor(t.Accent.Alpha(0.6))
				count = t.Accent
			} else if chip.Hovered() {
				chip.Background(t.SurfaceHover)
			}
			if chip.Clicked() {
				react(r.Emoji)
			}
			chip.Children(func() {
				ui.Text(c, r.Emoji).FontSize(t.FontSize + 1)
				ui.Text(c, strconv.Itoa(len(r.Who))).FontSize(t.FontSize - 1).Bold().FontFeatures("tnum").TextColor(count).
					ContentTransition(ui.ContentTransition{Numeric: true})
			})
		}
		add := ui.ButtonBase(c).Size(t.Space(9), t.Space(7)).Radius(t.Space(4)).Border(1, t.Border).Label("Add reaction").
			Tooltip("Add reaction").Role(ui.RolePopUpButton)
		open := ui.Local(add, "reactions.open", func() bool { return false })
		if add.Clicked() {
			*open = !*open
		}
		if add.Hovered() || *open {
			add.Background(t.SurfaceHover)
		}
		add.Expanded(*open).Children(func() {
			ui.Box(c).Size(t.Space(4.5), t.Space(4.5)).Draw(func(p *ui.Painter, r ui.Rect) { smiley(p, r, t.TextMuted) })
		})
		ui.Popover(c, add, open, func() {
			emoji.Picker(c, &s.Picker).Size(352, 400)
			if e, ok := s.Picker.Picked(); ok {
				*open = false
				if i := slices.IndexFunc(*rs, func(r Reaction) bool { return r.Emoji == e }); i < 0 || !slices.Contains((*rs)[i].Who, me) {
					react(e)
				}
			}
		})
	})
	if changed {
		root.SetChanged()
	}
	return root
}

// who names those who reacted, "You, Ben and 3 others".
func who(names []string, me string) string {
	var named []string
	for _, n := range names {
		if n == me {
			n = "You"
		}
		named = append(named, n)
	}
	switch n := len(named); {
	case n == 1:
		return named[0]
	case n <= 3:
		return strings.Join(named[:n-1], ", ") + " and " + named[n-1]
	default:
		return strings.Join(named[:2], ", ") + " and " + strconv.Itoa(n-2) + " others"
	}
}

// smiley draws a face with a plus at its top right, for adding a
// reaction.
func smiley(p *ui.Painter, r ui.Rect, color ui.Color) {
	k := r.W / 2
	cx, cy := r.X+k*0.9, r.Y+k*1.1
	rad := k * 0.78
	var face, mouth ui.Path
	face.Circle(cx, cy, rad)
	p.StrokePath(&face, 1.25, color)
	mouth.MoveTo(cx-rad*0.45, cy+rad*0.2).QuadTo(cx, cy+rad*0.7, cx+rad*0.45, cy+rad*0.2)
	p.StrokePath(&mouth, 1.25, color)
	var eyes ui.Path
	eyes.Circle(cx-rad*0.35, cy-rad*0.25, 1).Circle(cx+rad*0.35, cy-rad*0.25, 1)
	p.FillPath(&eyes, color)
	px, py := r.X+r.W-1.5, r.Y+1.5
	p.Fill(ui.Rect{X: px - 4, Y: py - 0.6, W: 5.5, H: 1.2}, color, 0.6)
	p.Fill(ui.Rect{X: px - 1.85, Y: py - 2.75, W: 1.2, H: 5.5}, color, 0.6)
}
