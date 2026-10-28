// Updater offers a made-up notes app three releases ahead of the one
// installed. Install Update downloads it, the first time failing halfway
// to show Try Again; once downloaded, Install and Relaunch moves the app
// to the new version. Skip, Remind Me Later and Escape close the prompt,
// and Check for Updates opens it again.
//
//	go run ./examples/updater
package main

import (
	"log"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/updater"
)

func day(m time.Month, d int) time.Time { return time.Date(2026, m, d, 0, 0, 0, 0, time.UTC) }

var releases = []updater.Release{
	{Version: "2.4.0", Date: day(time.October, 21), Notes: `**Linked notes** come to Lantern.

- Type ` + "`[[`" + ` to link to another note, and see its backlinks under it
- The sidebar shows pinned notes first
- Exported PDFs keep their links

Fixed a crash when pasting a table from Numbers.`},
	{Version: "2.3.1", Date: day(time.September, 30), Notes: `- Search finds words with accents when typed without them
- Fixed iCloud sync stalling after waking from sleep`},
	{Version: "2.3.0", Date: day(time.September, 2), Notes: `- **Focus mode** dims every paragraph but the one you are writing
- New themes: Paper and Slate
- Notes open 40% faster`},
}

type app struct {
	open     bool
	auto     bool
	update   updater.Update
	started  time.Time
	attempts int
}

const size = 48_300_000

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	if a.update.Phase == updater.Downloading {
		a.download(c)
	}
	ui.Column(c).Fill().Background(t.Background).Children(func() {
		if !a.open {
			ui.Column(c).Fill().Center().Gap(t.Space(3)).Children(func() {
				ui.Text(c, "Lantern "+a.update.Current).Bold().FontSize(t.FontSize + 4)
				if a.update.Current == releases[0].Version {
					ui.Text(c, "You're up to date.").TextColor(t.TextMuted)
					return
				}
				if ui.Button(c, "Check for Updates…").Clicked() {
					a.open = true
				}
			})
			return
		}
		ev := updater.View(c, a.update, &a.auto)
		switch ev.Action() {
		case updater.Install, updater.Retry:
			a.update.Phase, a.update.Downloaded, a.started = updater.Downloading, 0, c.Now()
			a.attempts++
		case updater.Cancel:
			a.update.Phase = updater.Available
		case updater.Relaunch:
			a.update.Current, a.update.Releases, a.open = releases[0].Version, nil, false
		case updater.Later, updater.Skip:
			a.open = false
		}
	})
}

// download moves the download on at 12 MB a second, failing halfway the
// first time.
func (a *app) download(c *ui.Context) {
	got := int64(c.Now().Sub(a.started).Seconds() * 12e6)
	switch {
	case a.attempts == 1 && got > size/2:
		a.update.Phase, a.update.Err = updater.Failed, "The network connection was lost."
	case got >= size:
		a.update.Phase, a.update.Downloaded = updater.Ready, size
	default:
		a.update.Downloaded = got
		c.After(time.Second / 30)
	}
}

func main() {
	a := &app{open: true, update: updater.Update{App: "Lantern", Current: "2.2.3", Releases: releases, Size: size}}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Software Update",
			Width:   620,
			Height:  520,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
