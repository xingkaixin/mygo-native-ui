// Annotate marks up a screenshot, as screenshot tools do, and exports it
// as annotated.png in the temporary directory. Without a picture, it takes
// one of a made-up settings page.
//
//	go run ./examples/annotate [picture.png|.jpg]
package main

import (
	"image"
	_ "image/jpeg"
	"image/png"
	"log"
	"os"
	"path/filepath"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/annotate"
)

type app struct {
	shot   image.Image
	marks  []annotate.Mark
	markup annotate.State
	saved  string
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(4)).Gap(t.Space(3)).Children(func() {
		annotate.View(c, &a.markup, a.shot, &a.marks).Grow(1).AlignSelf(ui.Stretch)
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, a.saved).TextColor(t.TextMuted).Grow(1)
			if ui.PrimaryButton(c, "Export").Clicked() {
				a.saved = a.export()
			}
		})
	})
}

func (a *app) export() string {
	f, err := os.Create(filepath.Join(os.TempDir(), "annotated.png"))
	if err != nil {
		return err.Error()
	}
	defer f.Close()
	if err := png.Encode(f, annotate.Export(a.shot, a.marks)); err != nil {
		return err.Error()
	}
	return "Saved " + f.Name()
}

// settings draws a made-up settings page, the screenshot to mark up.
func settings(c *ui.Context) {
	t := c.Theme()
	ui.Row(c).Fill().Background(t.Background).AlignItems(ui.Stretch).Children(func() {
		ui.Column(c).Width(200).Padding(16).Gap(4).Background(t.Surface).Children(func() {
			for i, s := range []string{"General", "Account", "Billing", "Notifications", "Security"} {
				row := ui.Text(c, s).Padding(6, 10).Radius(6)
				if i == 2 {
					row.Background(t.Accent.Alpha(0.15)).TextColor(t.Accent)
				}
			}
		})
		ui.Column(c).Grow(1).Padding(28).Gap(18).Children(func() {
			ui.Text(c, "Billing").Bold().FontSize(t.FontSize + 8)
			card := "4242 4242 4242 4242"
			email := "kai@example.com"
			plan := "Team"
			ui.Form(c, func() {
				ui.Field(c, "Plan", func() { ui.Select(c, &plan, []string{"Free", "Team", "Enterprise"}) })
				ui.Field(c, "Card", func() { ui.TextInput(c, &card) })
				ui.Field(c, "Receipts to", func() { ui.TextInput(c, &email) })
			})
			ui.Row(c).Gap(10).Children(func() {
				ui.PrimaryButton(c, "Save changes")
				ui.Button(c, "Cancel")
			})
		})
	})
}

func main() {
	a := &app{}
	if len(os.Args) > 1 {
		f, err := os.Open(os.Args[1])
		if err != nil {
			log.Fatal(err)
		}
		a.shot, _, err = image.Decode(f)
		f.Close()
		if err != nil {
			log.Fatal(err)
		}
	} else {
		a.shot = ui.Render(settings, 900, 560, 2)
	}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Annotate",
			Width:   1100,
			Height:  760,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
