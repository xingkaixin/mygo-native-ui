// Storagebar shows a Mac's disk by kind of file, as Storage settings do,
// and a team's cloud quota by member, nearly full, in a settings-like
// window.
//
//	go run ./examples/storagebar
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/storagebar"
)

const gb = 1_000_000_000

var disk = []storagebar.Part{
	{Name: "Applications", Size: 48_210_000_000},
	{Name: "Documents", Size: 112_730_000_000},
	{Name: "Photos", Size: 64_480_000_000},
	{Name: "Developer", Size: 38_920_000_000},
	{Name: "macOS", Size: 17_440_000_000},
	{Name: "System Data", Size: 31_870_000_000},
}

var quota = []storagebar.Part{
	{Name: "Ada", Size: 412 * gb},
	{Name: "Alan", Size: 288 * gb},
	{Name: "Grace", Size: 651 * gb},
	{Name: "Edsger", Size: 97 * gb},
	{Name: "Shared", Size: 456 * gb},
}

func view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(24).Gap(28).Children(func() {
		storagebar.View(c, "Macintosh HD", 494_384_795_648, disk)
		storagebar.View(c, "Team storage", 2_000*gb, quota)
	})
}

func main() {
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Storage",
			Width:   640,
			Height:  300,
			Content: ui.View(view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
