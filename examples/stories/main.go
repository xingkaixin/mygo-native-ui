// Stories shows four friends' stories: click a ring to watch theirs, the
// right of the card to skip ahead and the left to go back, and hold to
// pause.
//
//	go run ./examples/stories
package main

import (
	"log"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/stories"
)

type post struct {
	from, to ui.Color
	caption  string
}

var posts = [][]post{
	{{ui.Hex("#FF9A8B"), ui.Hex("#FF6A88"), "Sunrise run, 6 km"}, {ui.Hex("#FAD961"), ui.Hex("#F76B1C"), "Coffee first ☕"}},
	{{ui.Hex("#21D4FD"), ui.Hex("#B721FF"), "Made it to the coast"}, {ui.Hex("#08AEEA"), ui.Hex("#2AF598"), "The water is so clear"}, {ui.Hex("#0093E9"), ui.Hex("#80D0C7"), "Sunset swim"}},
	{{ui.Hex("#8EC5FC"), ui.Hex("#E0C3FC"), "New desk setup"}},
	{{ui.Hex("#FBAB7E"), ui.Hex("#F7CE68"), "Sourdough, day 12"}, {ui.Hex("#85FFBD"), ui.Hex("#FFFB7D"), "Garden tomatoes"}},
}

type app struct {
	stories stories.State
	people  []stories.Person
	viewing bool
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Padding(t.Space(6)).Gap(t.Space(6)).Background(t.Background).Children(func() {
		if stories.Tray(c, &a.stories, a.people).Opened {
			a.viewing = true
		}
		ui.Box(c).Grow(1).FillWidth().Center().Children(func() {
			if !a.viewing {
				ui.Text(c, "Click a ring to watch").TextColor(t.TextMuted)
				return
			}
			ev := stories.View(c, &a.stories, &a.people, func(person, i int) {
				p := posts[person][i]
				ui.Box(c).Fill().LinearGradient(ui.LinearGradient{From: p.from, To: p.to, Angle: 160}).Center().Children(func() {
					ui.Text(c, p.caption).FontSize(t.FontSize + 8).Bold().TextColor(ui.RGB(255, 255, 255)).
						TextAlign(ui.Center).MaxWidth(t.Space(60))
				})
			})
			if ev.Closed {
				a.viewing = false
			}
		})
	})
}

func main() {
	now := time.Now()
	names := []string{"Mia", "Leo", "Ava", "Noah"}
	a := &app{}
	for i, name := range names {
		p := stories.Person{Name: name}
		for k := range posts[i] {
			p.Stories = append(p.Stories, stories.Story{
				Length: 5 * time.Second,
				Posted: now.Add(-time.Duration(3*i+k+1) * 47 * time.Minute),
				Seen:   i == 2,
			})
		}
		a.people = append(a.people, p)
	}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Stories",
			Width:   620,
			Height:  860,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
