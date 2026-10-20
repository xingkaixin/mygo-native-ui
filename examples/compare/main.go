// Compare compares two pictures as code review tools show a changed
// image: by a slider, side by side, as an onion skin, or by the pixels
// that changed. Give it two pictures to compare your own; without them,
// it draws two screenshots of a settings card, the second with its save
// button moved, its accent color changed and a setting added.
//
//	go run ./examples/compare [before.png after.png]
package main

import (
	"image"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"os"
	"path/filepath"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/compare"
)

type app struct {
	name          string
	before, after image.Image
	compare       compare.State
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(4)).Gap(t.Space(3)).Children(func() {
		ui.Text(c, a.name).Bold().SingleLine()
		compare.View(c, &a.compare, a.before, a.after).Grow(1)
	})
}

// setting is a row of the card: what it sets, a line about it, and its
// switch.
type setting struct {
	name, detail string
	on           bool
}

// card draws a settings card as a screenshot at twice its size in DIPs;
// edited draws it with its save button on the left, a violet accent
// rather than the blue one, and a setting for quiet hours added.
func card(edited bool) image.Image {
	h := 380
	if edited {
		h = 436
	}
	return ui.Render(func(c *ui.Context) {
		t := *c.Theme()
		if edited {
			t.Accent = ui.RGB(124, 58, 237)
		}
		c.SetTheme(&t)
		ui.Box(c).Fill().Background(t.Surface).Padding(t.Space(5)).Children(func() {
			ui.Column(c).Fill().Background(t.Background).Border(1, t.Border).Radius(t.Space(3)).
				Padding(t.Space(5)).Gap(t.Space(4)).Children(func() {
				ui.Column(c).Gap(t.Space(1)).Children(func() {
					ui.Text(c, "Notifications").Bold().FontSize(t.FontSize + 4)
					ui.Text(c, "Choose what we send you, and where.").TextColor(t.TextMuted)
				})
				ui.Divider(c)
				rows := []setting{
					{"Email digests", "A summary of activity, once a day", true},
					{"Push notifications", "Mentions, replies and reviews", false},
					{"Product updates", "New features, now and then", true},
				}
				if edited {
					rows = append(rows, setting{"Quiet hours", "Hold alerts from 22:00 to 07:00", true})
				}
				for i, r := range rows {
					ui.Row(c.Key(i)).Gap(t.Space(4)).AlignItems(ui.Center).Children(func() {
						ui.Column(c).Grow(1).Gap(t.Space(0.5)).Children(func() {
							ui.Text(c, r.name)
							ui.Text(c, r.detail).FontSize(t.FontSize - 1).TextColor(t.TextMuted)
						})
						ui.Switch(c, &r.on).Label(r.name)
					})
				}
				ui.Box(c).Grow(1)
				ui.Row(c).Gap(t.Space(2)).Children(func() {
					if edited {
						ui.PrimaryButton(c, "Save changes")
						ui.Box(c).Grow(1)
						ui.Button(c, "Cancel")
						return
					}
					ui.Box(c).Grow(1)
					ui.Button(c, "Cancel")
					ui.PrimaryButton(c, "Save changes")
				})
			})
		})
	}, 480, h, 2)
}

// load decodes the picture in the file at path.
func load(path string) image.Image {
	f, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		log.Fatal(err)
	}
	return img
}

func main() {
	a := &app{}
	if len(os.Args) > 2 {
		a.before, a.after = load(os.Args[1]), load(os.Args[2])
		a.name = filepath.Base(os.Args[1]) + " → " + filepath.Base(os.Args[2])
	} else {
		a.before, a.after = card(false), card(true)
		a.name = "settings-card.png"
	}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Compare",
			Width:   1100,
			Height:  760,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
