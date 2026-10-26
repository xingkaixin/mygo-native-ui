// Collab is a shared document's header and comments: the people in it
// beside the title, where a click follows one, and reactions under each
// comment, which a click adds or takes back.
//
//	go run ./examples/collab
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/emoji"
	"github.com/xingkaixin/mygo-native-ui/ui/facepile"
	"github.com/xingkaixin/mygo-native-ui/ui/reactions"
)

type comment struct {
	author, ago, text string
	reactions         []reactions.Reaction
}

type app struct {
	people    []facepile.Person
	following string
	comments  []comment
	react     reactions.State
}

const me = "Ada Lovelace"

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Children(func() {
		ui.Row(c).Padding(t.Space(3), t.Space(5)).Gap(t.Space(4)).AlignItems(ui.Center).BorderWidth(0, 0, 1, 0).BorderColor(t.Border).Children(func() {
			ui.Text(c, "Q4 launch plan").Bold().FontSize(t.FontSize + 3).Grow(1)
			if i, ok := facepile.View(c, a.people, 5).Chosen(); ok {
				a.following = a.people[i].Name
			}
			ui.PrimaryButton(c, "Share")
		})
		if a.following != "" {
			ui.Row(c).Padding(t.Space(2), t.Space(5)).Gap(t.Space(3)).AlignItems(ui.Center).Background(t.Accent.Alpha(0.1)).Children(func() {
				ui.Text(c, "Following "+a.following).Grow(1)
				if ui.Button(c, "Stop following").Clicked() {
					a.following = ""
				}
			})
		}
		ui.Scroll(c).Grow(1).Children(func() {
			ui.Column(c).Padding(t.Space(5)).Gap(t.Space(5)).MaxWidth(t.Space(160)).Children(func() {
				for i := range a.comments {
					cm := &a.comments[i]
					ui.Row(c.Key(i)).Gap(t.Space(3)).AlignItems(ui.Start).Children(func() {
						ui.Avatar(c, cm.author, nil).Size(t.Space(9), t.Space(9)).Radius(t.Space(4.5))
						ui.Column(c).Grow(1).Shrink(1).Gap(t.Space(1.5)).Children(func() {
							ui.Row(c).Gap(t.Space(2)).Children(func() {
								ui.Text(c, cm.author).Bold()
								ui.Text(c, cm.ago).TextColor(t.TextMuted)
							})
							ui.Text(c, cm.text)
							reactions.View(c, &a.react, me, &cm.reactions).Margin(t.Space(1), 0, 0, 0)
						})
					})
				}
			})
		})
	})
}

func main() {
	a := &app{
		people: []facepile.Person{
			{Name: "Ada Lovelace", Color: ui.Hex("#2F80ED")},
			{Name: "Grace Hopper", Color: ui.Hex("#EB5757")},
			{Name: "Alan Turing", Color: ui.Hex("#27AE60")},
			{Name: "Katherine Johnson", Color: ui.Hex("#F2994A")},
			{Name: "Linus Pauling", Away: true},
			{Name: "Hedy Lamarr", Color: ui.Hex("#9B51E0")},
			{Name: "Claude Shannon", Away: true},
		},
		comments: []comment{
			{"Grace Hopper", "2h", "Moved the beta to the second week of November so support has a full week with the new docs.",
				[]reactions.Reaction{{Emoji: "👍", Who: []string{"Alan Turing", "Ada Lovelace", "Hedy Lamarr"}}, {Emoji: "🎉", Who: []string{"Katherine Johnson"}}}},
			{"Alan Turing", "1h", "Pricing page copy is ready for review. I kept the three tiers and dropped the annual toggle for now.",
				[]reactions.Reaction{{Emoji: "👀", Who: []string{"Grace Hopper", "Linus Pauling"}}}},
			{"Katherine Johnson", "12m", "Load tests pass at twice last quarter's peak. Numbers are in the appendix.", nil},
		},
		react: reactions.State{Picker: emoji.State{Recent: []string{"👍", "🎉", "❤️", "👀", "🚀"}}},
	}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Collab",
			Width:   800,
			Height:  600,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
