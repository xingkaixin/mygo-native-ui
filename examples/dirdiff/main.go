// Dirdiff compares two releases of a site, or the two folders given:
// open and close the folders, switch to only what differs, and choose a
// row to read where it stands.
//
//	go run ./examples/dirdiff
//	go run ./examples/dirdiff ~/old ~/new
package main

import (
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/dirdiff"
)

type app struct {
	left, right dirdiff.Dir
	diff        dirdiff.State
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Children(func() {
		dirdiff.View(c, &a.diff, a.left, a.right).Grow(1)
		ui.Divider(c)
		chosen, ok := a.diff.Chosen()
		if !ok {
			chosen = "Choose a file or folder"
		}
		ui.Text(c, chosen).Font("monospace").TextColor(t.TextMuted).Padding(t.Space(2), t.Space(3)).SingleLine()
	})
}

func release() (dirdiff.Dir, dirdiff.Dir) {
	day := func(d int) time.Time { return time.Date(2026, time.October, d, 14, 30, 0, 0, time.Local) }
	both := []dirdiff.Entry{
		{Path: "index.html", Size: 14_230, Modified: day(2)},
		{Path: "about.html", Size: 6_210, Modified: day(2)},
		{Path: "assets/logo.svg", Size: 3_402, Modified: day(1)},
		{Path: "assets/fonts/Inter.woff2", Size: 98_400, Modified: day(1)},
		{Path: "assets/fonts/InterBold.woff2", Size: 101_200, Modified: day(1)},
		{Path: "docs/install.md", Size: 4_810, Modified: day(2)},
		{Path: "docs/api/auth.md", Size: 12_004, Modified: day(2)},
		{Path: "docs/api/users.md", Size: 9_120, Modified: day(2)},
		{Path: "robots.txt", Size: 68, Modified: day(1)},
	}
	left := append(both[:len(both):len(both)],
		dirdiff.Entry{Path: "assets/app.js", Size: 182_300, Modified: day(2)},
		dirdiff.Entry{Path: "assets/app.css", Size: 41_200, Modified: day(2)},
		dirdiff.Entry{Path: "docs/api/billing.md", Size: 7_400, Modified: day(2)},
		dirdiff.Entry{Path: "docs/changelog.md", Size: 22_010, Modified: day(2)},
		dirdiff.Entry{Path: "legacy/old.html", Size: 3_100, Modified: day(1)},
	)
	right := append(both[:len(both):len(both)],
		dirdiff.Entry{Path: "assets/app.js", Size: 176_950, Modified: day(24)},
		dirdiff.Entry{Path: "assets/app.css", Size: 41_200, Modified: day(2)},
		dirdiff.Entry{Path: "assets/charts.js", Size: 64_300, Modified: day(24)},
		dirdiff.Entry{Path: "docs/api/billing.md", Size: 8_950, Modified: day(23)},
		dirdiff.Entry{Path: "docs/api/webhooks.md", Size: 6_020, Modified: day(23)},
		dirdiff.Entry{Path: "docs/changelog.md", Size: 23_870, Modified: day(24)},
		dirdiff.Entry{Path: "pricing.html", Size: 8_800, Modified: day(24)},
	)
	return dirdiff.Dir{Name: "site-1.2", Entries: left}, dirdiff.Dir{Name: "site-1.3", Entries: right}
}

func main() {
	a := &app{}
	a.left, a.right = release()
	if len(os.Args) == 3 {
		for i, d := range []*dirdiff.Dir{&a.left, &a.right} {
			es, err := dirdiff.Read(os.DirFS(os.Args[i+1]))
			if err != nil {
				log.Fatal(err)
			}
			*d = dirdiff.Dir{Name: filepath.Base(os.Args[i+1]), Entries: es}
		}
	}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Folder Comparison",
			Width:   960,
			Height:  620,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
