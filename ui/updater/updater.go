// Package updater asks the user to update the app, as Sparkle does on
// macOS: the new version and the one installed, the release notes of
// each version since, newest first, in Markdown, and Skip This Version,
// Remind Me Later and Install Update. While the update downloads, a bar
// shows how much of it has come with Cancel; once it has, Install and
// Relaunch; when it fails, what went wrong with Try Again. A check box
// under the notes turns on installing updates without asking. Escape
// reminds later.
//
//	switch updater.View(c, app.update, &app.autoUpdate).Action() {
//	case updater.Install, updater.Retry:
//		go app.download() // sets Phase and Downloaded as it goes
//	case updater.Skip:
//		app.skipped = app.update.Releases[0].Version
//	}
//
// The app downloads and installs; the prompt only shows where that is.
package updater

import (
	"fmt"
	"time"

	"github.com/egoist/mygo/i18n"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown"
)

// Release is a version: its number, the day it came out, and its notes
// in Markdown.
type Release struct {
	Version string
	Date    time.Time
	Notes   string
}

// Phase is where an update is.
type Phase uint8

const (
	// Available asks whether to install.
	Available Phase = iota
	// Downloading shows Downloaded of Size.
	Downloading
	// Ready offers to install and relaunch.
	Ready
	// Failed shows Err.
	Failed
)

// Update is what the prompt shows: the app's name, the version
// installed, the releases since, newest first, the size of the download
// in bytes, where the update is, how much of it has downloaded, and what
// went wrong when it failed.
type Update struct {
	App        string
	Current    string
	Releases   []Release
	Size       int64
	Phase      Phase
	Downloaded int64
	Err        string
}

// Action is what the user chose.
type Action uint8

const (
	None Action = iota
	Install
	Later
	Skip
	Cancel
	Relaunch
	Retry
)

// Events are what the user did with the prompt in a frame. The zero
// Events report nothing.
type Events struct {
	action Action
}

// Action returns what the user chose, None for nothing.
func (e Events) Action() Action { return e.action }

var arrow = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path d="M12 4v12m-5-5 5 5 5-5M5 20h14" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/></svg>`))

// View creates the prompt for u, and its check box of *auto. It grows in
// its parent, and the notes take the room left and scroll.
func View(c *ui.Context, u Update, auto *bool) Events {
	t := c.Theme()
	var ev Events
	latest := u.Current
	if len(u.Releases) > 0 {
		latest = u.Releases[0].Version
	}
	root := ui.Column(c).Grow(1).Gap(t.Space(4)).Padding(t.Space(5)).Role(ui.RoleDialog).Label("Software Update")
	if root.Shortcut(0, ui.KeyEscape) && u.Phase != Downloading {
		ev.action = Later
	}
	root.Children(func() {
		ui.Row(c).Gap(t.Space(4)).AlignItems(ui.Start).Children(func() {
			size := t.Space(16)
			ui.Box(c).Size(size, size).Radius(size*0.225).
				Gradient(t.Accent.Mix(ui.RGB(255, 255, 255), 0.25), t.Accent, 180).
				Shadow(0, 1, 3, 0, ui.RGBA(0, 0, 0, 0.2)).Center().Children(func() {
				ui.Icon(c, arrow).Size(size*0.55, size*0.55).TextColor(t.AccentText)
			})
			ui.Column(c).Gap(t.Space(1)).Grow(1).Shrink(1).Children(func() {
				ui.Text(c, headline(u)).Bold().FontSize(t.FontSize + 2).Role(ui.RoleHeading)
				ui.Text(c, fmt.Sprintf("%s %s is now available—you have %s.", u.App, latest, u.Current)).TextColor(t.TextMuted)
			})
		})
		if len(u.Releases) > 0 {
			notes(c, u.Releases)
		}
		switch u.Phase {
		case Available:
			ui.Checkbox(c, auto, "Automatically download and install updates in the future")
			ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
				if ui.Button(c, "Skip This Version").Clicked() {
					ev.action = Skip
				}
				ui.Spacer(c)
				if ui.Button(c, "Remind Me Later").Clicked() {
					ev.action = Later
				}
				if ui.PrimaryButton(c, "Install Update").AutoFocus().Clicked() {
					ev.action = Install
				}
			})
		case Downloading:
			ui.Column(c).Gap(t.Space(1.5)).Children(func() {
				share := 0.0
				if u.Size > 0 {
					share = float64(u.Downloaded) / float64(u.Size)
				}
				ui.Progress(c, share).FillWidth()
				ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
					ui.Text(c, "Downloading… "+bytes(u.Downloaded)+" of "+bytes(u.Size)).
						FontSize(t.FontSize - 1).TextColor(t.TextMuted).FontFeatures("tnum").Grow(1)
					if ui.Button(c, "Cancel").Clicked() {
						ev.action = Cancel
					}
				})
			})
		case Ready:
			ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
				ui.Text(c, "Ready to install. "+u.App+" will relaunch.").TextColor(t.TextMuted).Grow(1).Shrink(1)
				if ui.Button(c, "Install on Quit").Clicked() {
					ev.action = Later
				}
				if ui.PrimaryButton(c, "Install and Relaunch").AutoFocus().Clicked() {
					ev.action = Relaunch
				}
			})
		case Failed:
			ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
				ui.Text(c, u.Err).TextColor(t.Danger).Grow(1).Shrink(1)
				if ui.Button(c, "Remind Me Later").Clicked() {
					ev.action = Later
				}
				if ui.PrimaryButton(c, "Try Again").AutoFocus().Clicked() {
					ev.action = Retry
				}
			})
		}
	})
	return ev
}

// headline returns the prompt's title for where u is.
func headline(u Update) string {
	switch u.Phase {
	case Downloading:
		return "Downloading " + u.App + "…"
	case Ready:
		return "An update is ready to install"
	case Failed:
		return "The update could not be downloaded"
	}
	return "A new version of " + u.App + " is available!"
}

// notes creates the scrolling notes of releases, each under its version
// and day.
func notes(c *ui.Context, releases []Release) {
	t := c.Theme()
	r := c.Region()
	format := r.FormatLocale(c.Locale())
	ui.Scroll(c).Basis(0).Grow(1).MinHeight(t.Space(30)).Border(1, t.Border).Radius(t.Radius).
		Background(t.Surface).Label("Release notes").Children(func() {
		ui.Column(c).Padding(t.Space(4)).Gap(t.Space(5)).Children(func() {
			for i, rl := range releases {
				ui.Column(c.Key(i)).Gap(t.Space(2)).Children(func() {
					ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Baseline).Children(func() {
						ui.Text(c, "Version "+rl.Version).Bold().FontSize(t.FontSize + 1)
						if !rl.Date.IsZero() {
							ui.Text(c, i18n.FormatDate(format, rl.Date, i18n.DateOptions{Date: i18n.MediumStyle})).
								FontSize(t.FontSize - 1).TextColor(t.TextMuted)
						}
					})
					markdown.View(c, rl.Notes, nil)
				})
			}
		})
	})
}

// bytes writes n bytes in MB or GB, as macOS counts them, in thousands.
func bytes(n int64) string {
	switch {
	case n >= 1e9:
		return fmt.Sprintf("%.2f GB", float64(n)/1e9)
	case n >= 1e6:
		return fmt.Sprintf("%.1f MB", float64(n)/1e6)
	}
	return fmt.Sprintf("%d KB", (n+999)/1000)
}
