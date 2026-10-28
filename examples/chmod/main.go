// Chmod edits the permissions of a few files that each show a common
// mode: a script anyone may run, a private key only its owner may read,
// a shared directory whose files keep its group, a program that runs as
// its owner, and a scratch directory where anyone may write but only
// remove their own files.
//
//	go run ./examples/chmod
package main

import (
	"io/fs"
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/chmod"
)

type file struct {
	name string
	mode fs.FileMode
}

var (
	files = []file{
		{"deploy.sh", 0o755},
		{"id_ed25519", 0o600},
		{"shared", fs.ModeDir | fs.ModeSetgid | 0o775},
		{"passwd", fs.ModeSetuid | 0o755},
		{"tmp", fs.ModeDir | fs.ModeSticky | 0o777},
	}
	chosen int
)

func view(c *ui.Context) {
	t := c.Theme()
	ui.Row(c).Fill().Background(t.Background).Padding(t.Space(6)).Gap(t.Space(8)).AlignItems(ui.Start).Children(func() {
		ui.Column(c).Width(t.Space(64)).Gap(t.Space(0.5)).Children(func() {
			for i, f := range files {
				row := ui.ButtonBase(c.Key(i)).Justify(ui.Start).Gap(t.Space(3)).Padding(t.Space(2), t.Space(3)).Radius(t.Space(1.5)).
					Checked(i == chosen).Label(f.name)
				if i == chosen {
					row.Background(t.Accent.Alpha(0.12))
				}
				if row.Clicked() {
					chosen = i
				}
				row.Children(func() {
					ui.Text(c, chmod.List(f.mode)).Font("monospace").FontSize(t.FontSize - 1).TextColor(t.TextMuted)
					ui.Text(c, f.name).Font("monospace")
				})
			}
		})
		chmod.View(c.Key(chosen), &files[chosen].mode, files[chosen].name)
	})
}

func main() {
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Chmod",
			Width:   900,
			Height:  520,
			Content: ui.View(view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
