// Skeleton loads a feed of posts, showing placeholders in their shape for
// the two seconds they take; Reload loads them again.
//
//	go run ./examples/skeleton
package main

import (
	"log"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/skeleton"
)

type post struct {
	name, ago, text string
	picture         ui.Color
}

var posts = []post{
	{"Mira Okafor", "2m", "Finally finished the cabin roof before the first snow. Three weekends, one sprained thumb, zero leaks so far.", ui.Hex("#8FB3A1")},
	{"Tomás Lindqvist", "18m", "Reading group moves to Thursdays from next week. Same café, same corner table, bring the second half of the book.", ui.Color{}},
	{"Hana Ito", "1h", "The night market has a new dumpling stall by the bridge. The chili oil alone is worth the walk.", ui.Hex("#D9A27E")},
}

type app struct {
	until time.Time
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	wait := a.until.Sub(c.Now())
	loading := wait > 0
	if loading {
		c.After(wait)
	}
	ui.Column(c).Fill().Background(t.Background).Children(func() {
		ui.Row(c).Padding(t.Space(3), t.Space(5)).Gap(t.Space(3)).AlignItems(ui.Center).BorderWidth(0, 0, 1, 0).BorderColor(t.Border).Children(func() {
			ui.Text(c, "Neighbors").Bold().FontSize(t.FontSize + 3).Grow(1)
			if ui.Button(c, "Reload").Disabled(loading).Clicked() {
				a.until = c.Now().Add(2 * time.Second)
			}
		})
		ui.Scroll(c).Grow(1).Children(func() {
			ui.Column(c).Padding(t.Space(5)).Gap(t.Space(4)).MaxWidth(t.Space(150)).Children(func() {
				for i, p := range posts {
					ui.Column(c.Key(i)).Gap(t.Space(3)).Padding(t.Space(4)).Radius(t.Space(3)).Background(t.Surface).Children(func() {
						if loading {
							placeholder(c, p.picture.A > 0)
						} else {
							show(c, p)
						}
					})
				}
			})
		})
	})
}

func placeholder(c *ui.Context, picture bool) {
	t := c.Theme()
	ui.Row(c).Gap(t.Space(3)).Children(func() {
		skeleton.Circle(c, t.Space(10))
		ui.Column(c).Children(func() {
			skeleton.Line(c, t.Space(32))
			skeleton.Line(c, t.Space(12))
		})
	})
	skeleton.Text(c, 2)
	if picture {
		skeleton.Box(c).Height(t.Space(40))
	}
}

func show(c *ui.Context, p post) {
	t := c.Theme()
	ui.Row(c).Gap(t.Space(3)).Children(func() {
		ui.Avatar(c, p.name, nil).Size(t.Space(10), t.Space(10))
		ui.Column(c).Children(func() {
			ui.Text(c, p.name).Bold()
			ui.Text(c, p.ago).TextColor(t.TextMuted).FontSize(t.FontSize - 1)
		})
	})
	ui.Text(c, p.text)
	if p.picture.A > 0 {
		ui.Box(c).Height(t.Space(40)).Radius(t.Radius).Gradient(p.picture, p.picture.Mix(t.Background, 0.5), 160)
	}
}

func main() {
	a := &app{until: time.Now().Add(2 * time.Second)}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Skeleton",
			Width:   720,
			Height:  760,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
