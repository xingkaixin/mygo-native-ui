// Package permissions walks the user through the system permissions an
// app needs, as an app's first run asks for the camera, the microphone
// or screen recording: a row for each, with its icon in its own color,
// its name, a Required tag and why the app needs it, and at its end
// Allow while the user has not been asked, Open System Settings once
// they said no, and a check once it is allowed. Under the rows, how many
// of the required ones are allowed and Continue, which stays off until
// they all are.
//
//	ev := permissions.View(c, app.perms)
//	if i, ok := ev.Requested(); ok {
//		app.perms[i].Status = app.ask(app.perms[i].Kind) // the system's prompt
//	}
//	if i, ok := ev.OpenedSettings(); ok {
//		app.openSettings(app.perms[i].Kind)
//	}
//
// The app asks the system and reads back each status, as when its
// window comes back to the front from System Settings.
package permissions

import (
	"strconv"

	"github.com/egoist/mygo/ui"
)

// Kind is a permission the system guards.
type Kind uint8

const (
	Camera Kind = iota
	Microphone
	ScreenRecording
	Accessibility
	Notifications
	FullDiskAccess
	Location
	Contacts
	Calendars
)

// Name returns k's name as System Settings shows it.
func (k Kind) Name() string { return kinds[k].name }

// Status is what the user said to a permission.
type Status uint8

const (
	// NotDetermined is a permission the app has not asked for.
	NotDetermined Status = iota
	Granted
	Denied
)

// Permission is a permission the app asks for: its kind, why the app
// needs it, what the user said, and whether the app cannot go on
// without it.
type Permission struct {
	Kind     Kind
	Reason   string
	Status   Status
	Required bool
}

// Events are what the user did with the permissions in a frame. The zero
// Events report nothing.
type Events struct {
	requested, settings int
	continued           bool
}

// Requested reports the index of the permission whose Allow the user
// clicked, for the app to ask the system for it.
func (e Events) Requested() (int, bool) { return e.requested - 1, e.requested > 0 }

// OpenedSettings reports the index of the permission whose Open System
// Settings the user clicked, for the app to open its pane.
func (e Events) OpenedSettings() (int, bool) { return e.settings - 1, e.settings > 0 }

// Continued reports that the user clicked Continue, with every required
// permission allowed.
func (e Events) Continued() bool { return e.continued }

type kind struct {
	name string
	hue  float32
	icon *ui.SVG
}

func svg(body string) *ui.SVG {
	return ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">` + body + `</svg>`))
}

var kinds = [...]kind{
	Camera:          {"Camera", 250, svg(`<path d="M3 8.5A2 2 0 0 1 5 6.5h2L8.5 4.5h7L17 6.5h2a2 2 0 0 1 2 2V17a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/><circle cx="12" cy="12.5" r="3.5"/>`)},
	Microphone:      {"Microphone", 55, svg(`<rect x="9" y="3" width="6" height="11" rx="3"/><path d="M5.5 11a6.5 6.5 0 0 0 13 0M12 17.5V21"/>`)},
	ScreenRecording: {"Screen & System Audio Recording", 300, svg(`<rect x="3" y="4" width="18" height="12" rx="2"/><path d="M8 20h8M12 16v4"/><circle cx="12" cy="10" r="2.2" fill="currentColor"/>`)},
	Accessibility:   {"Accessibility", 240, svg(`<circle cx="12" cy="4.5" r="1.6"/><path d="M5 8.5 12 10l7-1.5M12 10v4.5m0 0L9 20m3-5.5 3 5.5"/>`)},
	Notifications:   {"Notifications", 20, svg(`<path d="M6 16.5V11a6 6 0 0 1 12 0v5.5l1.5 1.5h-15zM10 20.5a2 2 0 0 0 4 0"/>`)},
	FullDiskAccess:  {"Full Disk Access", 215, svg(`<path d="M3.5 7.5a2 2 0 0 1 2-2h4l2 2h7a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2h-13a2 2 0 0 1-2-2z"/>`)},
	Location:        {"Location Services", 265, svg(`<path d="M12 21s-6.5-5.6-6.5-11a6.5 6.5 0 0 1 13 0c0 5.4-6.5 11-6.5 11z"/><circle cx="12" cy="10" r="2.3"/>`)},
	Contacts:        {"Contacts", 60, svg(`<circle cx="12" cy="9" r="3.5"/><path d="M5 20a7 7 0 0 1 14 0"/>`)},
	Calendars:       {"Calendars", 25, svg(`<rect x="3.5" y="5" width="17" height="15.5" rx="2"/><path d="M3.5 10h17M8 3v4M16 3v4"/>`)},
}

var check = svg(`<path d="m6 12.5 4 4 8-9"/>`)

// View creates the rows of perms and the Continue under them. It reports
// the buttons the user clicked.
func View(c *ui.Context, perms []Permission) Events {
	t := c.Theme()
	var ev Events
	required, allowed := 0, 0
	for _, p := range perms {
		if p.Required {
			required++
			if p.Status == Granted {
				allowed++
			}
		}
	}
	ui.Column(c).Gap(t.Space(4)).Role(ui.RoleGroup).Label("Permissions").Children(func() {
		ui.Column(c).Border(1, t.Border).Radius(t.Radius + 4).Background(t.Surface).Clip().Children(func() {
			for i, p := range perms {
				if i > 0 {
					ui.Box(c.Key(i)).Height(1).Background(t.Border).MarginStart(t.Space(16))
				}
				switch row(c.Key(p.Kind), p) {
				case 1:
					ev.requested = i + 1
				case 2:
					ev.settings = i + 1
				}
			}
		})
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			if required > 0 {
				ui.Text(c, strconv.Itoa(allowed)+" of "+strconv.Itoa(required)+" required allowed").
					TextColor(t.TextMuted).FontFeatures("tnum").Grow(1)
			} else {
				ui.Spacer(c)
			}
			if ui.PrimaryButton(c, "Continue").Disabled(allowed < required).Clicked() {
				ev.continued = true
			}
		})
	})
	return ev
}

// row creates p's row and returns 1 when its Allow was clicked, 2 when
// its Open System Settings was.
func row(c *ui.Context, p Permission) int {
	t := c.Theme()
	k := kinds[p.Kind]
	clicked := 0
	ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Padding(t.Space(3), t.Space(4)).Children(func() {
		size := t.Space(9)
		ui.Box(c).Size(size, size).Radius(size*0.25).Center().
			Gradient(ui.Oklch(0.72, 0.15, k.hue), ui.Oklch(0.58, 0.17, k.hue), 180).Children(func() {
			ui.Icon(c, k.icon).Size(size*0.62, size*0.62).TextColor(ui.RGB(255, 255, 255))
		})
		ui.Column(c).Gap(t.Space(0.5)).Grow(1).Shrink(1).Children(func() {
			ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
				ui.Text(c, k.name).FontWeight(600)
				if p.Required {
					ui.Text(c, "Required").FontSize(t.FontSize-3).FontWeight(600).TextColor(t.TextMuted).
						Border(1, t.Border).Padding(0, t.Space(1.5)).Radius(t.Space(1))
				}
			})
			if p.Reason != "" {
				ui.Text(c, p.Reason).FontSize(t.FontSize - 1).TextColor(t.TextMuted)
			}
		})
		switch p.Status {
		case Granted:
			ui.Row(c).Gap(t.Space(1.5)).AlignItems(ui.Center).Role(ui.RoleStatus).Label(k.name + " allowed").Children(func() {
				ui.Box(c).Size(t.Space(5), t.Space(5)).Radius(t.Space(2.5)).Background(t.Success).Center().Children(func() {
					ui.Icon(c, check).Size(t.Space(3.5), t.Space(3.5)).TextColor(ui.RGB(255, 255, 255))
				})
				ui.Text(c, "Allowed").TextColor(t.TextMuted)
			})
		case Denied:
			ui.Column(c).AlignItems(ui.End).Gap(t.Space(1)).Children(func() {
				if ui.Button(c, "Open System Settings").Label("Open System Settings for " + k.name).Clicked() {
					clicked = 2
				}
				ui.Text(c, "Turned off").FontSize(t.FontSize - 2).TextColor(t.Danger)
			})
		default:
			if ui.Button(c, "Allow").Label("Allow " + k.name).Clicked() {
				clicked = 1
			}
		}
	})
	return clicked
}
