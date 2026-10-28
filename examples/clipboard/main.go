// Clipboard shows a clipboard history of texts, links, colors and an
// image: search it, move with the arrows, pin with ⌘P, and paste with
// Enter or ⌘1 to ⌘9. Copy adds what is typed at the bottom as a new clip.
//
//	go run ./examples/clipboard
package main

import (
	"image"
	"image/color"
	"log"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/clipboard"
)

type app struct {
	history clipboard.State
	clips   []clipboard.Clip
	typed   string
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Children(func() {
		ui.Box(c).Grow(1).Basis(0).Children(func() {
			ev := clipboard.View(c, &a.history, &a.clips)
			if ev.Pasted != nil {
				clip := *ev.Pasted
				clip.Copied = c.Now()
				clipboard.Add(&a.clips, clip, 50)
				c.Toast("Pasted " + clip.Kind().String())
			}
		})
		ui.Divider(c)
		ui.Row(c).Gap(t.Space(2)).Padding(t.Space(3)).Children(func() {
			ui.TextInput(c, &a.typed).Placeholder("Type something to copy").Grow(1)
			if ui.Button(c, "Copy").Disabled(a.typed == "").Clicked() {
				clipboard.Add(&a.clips, clipboard.Clip{Text: a.typed, Source: "Notes", Copied: c.Now()}, 50)
				a.typed = ""
			}
		})
	})
}

func gradient() *ui.Bitmap {
	img := image.NewRGBA(image.Rect(0, 0, 320, 200))
	for y := range 200 {
		for x := range 320 {
			img.Set(x, y, color.RGBA{uint8(40 + x/2), uint8(90 + y/2), 220, 255})
		}
	}
	return ui.NewBitmap(img)
}

func main() {
	now := time.Now()
	ago := func(d time.Duration) time.Time { return now.Add(-d) }
	a := &app{clips: []clipboard.Clip{
		{Text: "git rebase -i origin/main", Source: "Terminal", Copied: ago(20 * time.Second), Pinned: true},
		{Text: "#0A84FF", Source: "Figma", Copied: ago(2 * time.Minute)},
		{Text: "https://github.com/egoist/mygo/releases", Source: "Safari", Copied: ago(6 * time.Minute)},
		{Text: "Thanks for the quick turnaround! I've left two comments on the draft; the rest looks good to ship.", Source: "Mail", Copied: ago(25 * time.Minute)},
		{Image: gradient(), Source: "Preview", Copied: ago(48 * time.Minute)},
		{Text: "rgb(255, 159, 10)", Source: "Figma", Copied: ago(2 * time.Hour)},
		{Text: "func (s *State) order(clips []Clip) []int {\n\tq := strings.ToLower(s.query)\n\t...\n}", Source: "Zed", Copied: ago(3 * time.Hour)},
		{Text: "1 Market Street, San Francisco, CA 94105", Source: "Maps", Copied: ago(26 * time.Hour), Pinned: true},
		{Text: "https://pkg.go.dev/image", Source: "Safari", Copied: ago(50 * time.Hour)},
	}}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Clipboard",
			Width:   860,
			Height:  560,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
