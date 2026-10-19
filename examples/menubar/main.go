// Menubar is a menu bar app of a made-up system's state, built of this
// module's parts for menu bar panels: its icon a graph of the load of the
// CPU, drawn anew each second with trayicon; its panel Control Center's
// tiles, rings of the load, of the memory used and of a focus timer, and
// a menu. A click on the icon opens and closes the panel, a right click
// shows a menu to quit.
//
//	go run ./examples/menubar
package main

import (
	"fmt"
	"log"
	"math/rand/v2"
	"runtime"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/menupanel"
	"github.com/xingkaixin/mygo-native-ui/ui/ring"
	"github.com/xingkaixin/mygo-native-ui/ui/tiles"
	"github.com/xingkaixin/mygo-native-ui/ui/trayicon"
)

var (
	wifiIcon  = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round"><path d="M2 9a15 15 0 0 1 20 0"/><path d="M5.5 12.5a10 10 0 0 1 13 0"/><path d="M9 16a5 5 0 0 1 6 0"/><circle cx="12" cy="19.5" r="1.2" fill="currentColor" stroke="none"/></svg>`))
	moonIcon  = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path fill="currentColor" d="M20 14.5A8.5 8.5 0 0 1 9.5 4a8.5 8.5 0 1 0 10.5 10.5z"/></svg>`))
	soundIcon = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M4 9h4l5-4v14l-5-4H4z" fill="currentColor"/><path d="M16.5 8.5a5 5 0 0 1 0 7"/><path d="M19.5 5.5a9 9 0 0 1 0 13"/></svg>`))
)

// app is the state of the icon and the panel, all of it read and changed
// on the main thread.
type app struct {
	tray   *mygo.Tray
	panel  *mygo.Window // the open panel, or nil
	menu   menupanel.State
	rand   *rand.Rand
	cpu    []float64 // a second's load each, the latest last
	memory float64
	wifi   bool
	volume float64
	focus  bool
	length int       // of a focus, in minutes
	end    time.Time // when the focus ends
}

func newApp() *app {
	a := &app{rand: rand.New(rand.NewPCG(5, 6)), memory: 0.62, wifi: true, volume: 0.5, length: 25}
	for range 12 {
		a.cpu = append(a.cpu, 0.2+0.3*a.rand.Float64())
	}
	return a
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	if c.Shortcut(ui.Cmd, ui.KeyF) {
		a.setFocus(!a.focus)
	}
	c.OnShortcut(ui.Cmd, ui.KeyQ, mygo.App.Quit)
	c.Root().Background(ui.Transparent)
	box := ui.Column(c).Fill()
	switch {
	case c.Vibrancy(): // over the popover's material on macOS
	case runtime.GOOS == "windows": // which rounds and outlines the flyout
		box.Background(t.Background)
	default:
		box.Background(t.Background).Radius(10).Border(1, t.Border)
	}
	box.Children(func() {
		ui.Grid(c).Columns(2).Gap(10).Padding(12).Children(func() {
			network := "Off"
			if a.wifi {
				network = "Home"
			}
			tiles.Toggle(c, &a.wifi, wifiIcon, "Wi-Fi", network)
			status := "Off"
			if a.focus {
				status = "Until " + a.end.Format("15:04")
			}
			if tiles.Toggle(c, &a.focus, moonIcon, "Focus", status).Changed() {
				a.setFocus(a.focus)
			}
			tiles.Slider(c, &a.volume, soundIcon, "Sound").ColumnSpan(2)
			tiles.Tile(c).ColumnSpan(2).Children(func() {
				ui.Row(c).Justify(ui.SpaceAround).Children(func() {
					gauge(c, "CPU", func() {
						load := a.cpu[len(a.cpu)-1]
						ring.Progress(c, load, t.Accent).Size(52, 52).Children(func() {
							ui.Textf(c, "%.0f%%", 100*load).FontSize(11).Bold()
						})
					})
					gauge(c, "Memory", func() {
						ring.Progress(c, a.memory, ui.Hex("#10b981")).Size(52, 52).Children(func() {
							ui.Textf(c, "%.0f%%", 100*a.memory).FontSize(11).Bold()
						})
					})
					gauge(c, "Focus", func() {
						total := time.Duration(a.length) * time.Minute
						end := a.end
						if !a.focus {
							end = c.Now().Add(total)
						}
						ring.Countdown(c, end, total, ui.Hex("#f59e0b")).Size(52, 52).FontSize(11)
					})
				})
			})
		})
		ui.Divider(c)
		menupanel.View(c, &a.menu, func(m *menupanel.Menu) {
			title := "Start Focus"
			if a.focus {
				title = "Stop Focus"
			}
			if m.Item(title).Shortcut(ui.Cmd, ui.KeyF).Chosen() {
				a.setFocus(!a.focus)
			}
			m.Submenu("Focus Length", func(m *menupanel.Menu) {
				for _, n := range []int{15, 25, 50} {
					if m.Item(fmt.Sprintf("%d minutes", n)).Checked(n == a.length).Chosen() {
						a.length = n
					}
				}
			})
			m.Separator()
			if m.Item("Quit").Shortcut(ui.Cmd, ui.KeyQ).Chosen() {
				mygo.App.Quit()
			}
		})
	})
}

// gauge builds a ring that build builds over its caption.
func gauge(c *ui.Context, caption string, build func()) {
	t := c.Theme()
	ui.Column(c).Gap(6).AlignItems(ui.Center).Children(func() {
		build()
		ui.Text(c, caption).FontSize(11).TextColor(t.TextMuted)
	})
}

// setFocus starts or stops a focus of the length chosen.
func (a *app) setFocus(on bool) {
	a.focus = on
	if on {
		a.end = time.Now().Add(time.Duration(a.length) * time.Minute)
	}
	a.updateTray()
}

// tick moves the made-up state on by a second.
func (a *app) tick() {
	load := a.cpu[len(a.cpu)-1] + 0.15*a.rand.NormFloat64()
	a.cpu = append(a.cpu[1:], max(0.03, min(1, load)))
	a.memory = max(0.4, min(0.9, a.memory+0.01*a.rand.NormFloat64()))
	if a.focus && time.Now().After(a.end) {
		a.focus = false
	}
	a.updateTray()
	if a.panel != nil {
		a.panel.Invalidate()
	}
}

// updateTray draws the icon, a graph of the CPU's load, black for macOS to
// tint and blue elsewhere, and titles it with the time left of a focus.
func (a *app) updateTray() {
	template := runtime.GOOS == "darwin"
	ink := ui.Hex("#3b82f6")
	if template {
		ink = ui.RGB(0, 0, 0)
	}
	png, err := trayicon.Render(24, 18, func(c *ui.Context) { trayicon.Bars(c, a.cpu, ink) })
	if err != nil {
		log.Print("tray icon: ", err)
		return
	}
	a.tray.SetIcon(png, template)
	title := ""
	if left := time.Until(a.end); a.focus && left > 0 {
		s := int(left.Seconds() + 0.999)
		title = fmt.Sprintf("%d:%02d", s/60, s%60)
	}
	a.tray.SetTitle(title)
}

// toggle opens the panel next to the icon, or closes it.
func (a *app) toggle() {
	if a.panel != nil {
		a.panel.Close()
		return
	}
	a.panel = mygo.NewFlyout(mygo.FlyoutOptions{
		Tray:      a.tray,
		Placement: mygo.PlacementBottom,
		Gap:       4,
		Width:     300,
		Height:    420,
		Focusable: true,
		Shadow:    true,
		Popover:   true,
		Content:   ui.View(a.view),
	})
	a.panel.OnClosed(func() { a.panel = nil })
}

func (a *app) ready() {
	var err error
	if a.tray, err = mygo.NewTray(mygo.TrayOptions{ToolTip: "Stats"}); err != nil {
		log.Fatal("tray: ", err)
	}
	a.updateTray()
	menu := mygo.NewMenu([]*mygo.MenuItem{
		{Label: "Open Stats", Click: func(*mygo.MenuItem, *mygo.Window) { a.toggle() }},
		mygo.Separator(),
		{Role: mygo.RoleQuit},
	})
	if runtime.GOOS == "linux" {
		// AppIndicator only shows menus, and tells nothing of clicks.
		a.tray.SetMenu(menu)
	} else {
		a.tray.OnClick(a.toggle)
		a.tray.OnRightClick(func() {
			if a.panel != nil {
				a.panel.Close()
			}
			a.tray.SetMenu(menu)
			defer a.tray.SetMenu(nil)
			a.tray.PopUpMenu()
		})
	}
	go func() {
		for range time.Tick(time.Second) {
			mygo.RunOnMain(a.tick)
		}
	}()
	mygo.App.OnWillQuit(func(*mygo.QuitEvent) { a.tray.Destroy() })
}

func main() {
	a := newApp()
	mygo.App.SetName("Stats")
	mygo.App.SetActivationPolicy(mygo.ActivationPolicyAccessory)
	// The panel is the app's only window: keep running as it closes.
	mygo.App.OnWindowAllClosed(func() {})
	mygo.App.WhenReady(a.ready)
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
