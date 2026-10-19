// Settings is a preferences window laid out as macOS System Settings:
// panes chosen in a sidebar, each of groups of rows, a label on the left
// of each and its control on the right.
//
//	go run ./examples/settings
package main

import (
	"fmt"
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/settings"
)

var (
	generalIcon = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="3.5"/><path stroke-linecap="round" d="M12 2.5v3M12 18.5v3M2.5 12h3M18.5 12h3M5.3 5.3l2.1 2.1M16.6 16.6l2.1 2.1M5.3 18.7l2.1-2.1M16.6 7.4l2.1-2.1"/></svg>`))
	dockIcon    = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="4" width="18" height="16" rx="3"/><path d="M7 16h10" stroke-linecap="round"/></svg>`))
	bellIcon    = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linejoin="round"><path d="M6 16V11a6 6 0 0 1 12 0v5l2 2H4z"/><path d="M10 20.5a2 2 0 0 0 4 0" stroke-linecap="round"/></svg>`))
)

// app is the preferences the window changes, and the pane it shows.
type app struct {
	pane string

	name       string
	appearance string
	scrollBars string
	recents    float64

	size        float64
	magnify     bool
	position    int
	minimize    string
	autohide    bool
	showRecents bool

	previews string
	sleeping bool
	locked   bool
	mirror   bool
	detail   string
	blocked  map[string]bool
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Row(c).Fill().Background(t.Background).AlignItems(ui.Stretch).Children(func() {
		ui.Sidebar(c, &a.pane, func() {
			ui.SidebarItem(c, "general", generalIcon, "General")
			ui.SidebarItem(c, "dock", dockIcon, "Desktop & Dock")
			ui.SidebarItem(c, "notifications", bellIcon, "Notifications")
		}).Width(220)
		ui.Divider(c)
		ui.Scroll(c).Grow(1).Children(func() {
			ui.Column(c).Padding(t.Space(5)).Gap(t.Space(5)).Children(func() {
				switch a.pane {
				case "general":
					a.general(c)
				case "dock":
					a.dock(c)
				default:
					a.notifications(c)
				}
			})
		})
	})
}

func (a *app) general(c *ui.Context) {
	ui.Text(c, "General").FontSize(20).Bold()
	settings.Group(c, "", func(g *settings.Rows) {
		g.Row("Computer name", func() { ui.TextInput(c, &a.name).Width(200).Label("Computer name") })
		g.Row("Appearance", func() {
			ui.Select(c, &a.appearance, []string{"Auto", "Light", "Dark"}).Width(140).Label("Appearance")
		})
		g.Row("Show scroll bars", func() {
			ui.Select(c, &a.scrollBars, []string{"Automatically", "When scrolling", "Always"}).Width(160).Label("Show scroll bars")
		})
	})
	settings.Group(c, "Recent items", func(g *settings.Rows) {
		g.Row("Documents, apps and servers", func() {
			ui.Slider(c, &a.recents, 0, 50).Width(160).Label("Recent items")
			ui.Text(c, fmt.Sprintf("%.0f", a.recents)).Width(24).FontFeatures("tnum")
		})
		g.Note("Recent items show in the Apple menu and in Open dialogs.")
	})
}

func (a *app) dock(c *ui.Context) {
	ui.Text(c, "Desktop & Dock").FontSize(20).Bold()
	settings.Group(c, "Dock", func(g *settings.Rows) {
		g.Row("Size", func() { ui.Slider(c, &a.size, 16, 128).Width(200).Label("Size") })
		g.Toggle("Magnification", &a.magnify)
		g.Row("Position on screen", func() {
			ui.Segmented(c, &a.position, "Left", "Bottom", "Right").Label("Position on screen")
		})
		g.Row("Minimize windows using", func() {
			ui.Select(c, &a.minimize, []string{"Genie effect", "Scale effect"}).Width(160).Label("Minimize windows using")
		})
		g.Toggle("Automatically hide and show the Dock", &a.autohide)
		g.Toggle("Show suggested and recent apps in Dock", &a.showRecents)
		g.Note("Hidden, the Dock shows when the pointer reaches the edge of the screen.")
	})
	settings.Group(c, "", func(g *settings.Rows) {
		g.Link("Hot Corners…", "")
	})
}

func (a *app) notifications(c *ui.Context) {
	t := c.Theme()
	if a.detail != "" {
		ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
			if ui.Button(c, "‹").Label("Back").Clicked() {
				a.detail = ""
			}
			ui.Text(c, a.detail).FontSize(20).Bold()
		})
		settings.Group(c, "", func(g *settings.Rows) {
			on := !a.blocked[a.detail]
			if g.Toggle("Allow notifications", &on).Changed() {
				a.blocked[a.detail] = !on
			}
		})
		return
	}
	ui.Text(c, "Notifications").FontSize(20).Bold()
	settings.Group(c, "Notification Center", func(g *settings.Rows) {
		g.Row("Show previews", func() {
			ui.Select(c, &a.previews, []string{"Always", "When unlocked", "Never"}).Width(160).Label("Show previews")
		})
		g.Toggle("Allow notifications when the display is sleeping", &a.sleeping)
		g.Toggle("Allow notifications when the screen is locked", &a.locked)
		g.Toggle("Allow notifications when mirroring or sharing the display", &a.mirror)
	})
	settings.Group(c, "Application Notifications", func(g *settings.Rows) {
		for _, app := range []struct{ name, style string }{
			{"Calendar", "Alerts"}, {"Mail", "Banners"}, {"Messages", "Banners"}, {"Reminders", "Alerts"},
		} {
			style := app.style
			if a.blocked[app.name] {
				style = "Off"
			}
			if g.Link(app.name, style).Clicked() {
				a.detail = app.name
			}
		}
	})
}

func main() {
	a := &app{
		pane: "dock", name: "MacBook Pro", appearance: "Auto", scrollBars: "Automatically", recents: 10,
		size: 48, position: 1, minimize: "Genie effect", showRecents: true,
		previews: "When unlocked", locked: true, blocked: map[string]bool{},
	}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Settings",
			Width:   760,
			Height:  620,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
