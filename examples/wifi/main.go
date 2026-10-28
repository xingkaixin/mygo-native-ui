// Wifi shows the Wi-Fi menu of a menu bar: join a known network at once,
// open Other Networks and join Café Lumière, which asks for no password,
// or The Neighbours, whose password is "correct horse"; anything else
// fails as a wrong password does.
//
//	go run ./examples/wifi
package main

import (
	"log"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/wifi"
)

type app struct {
	menu     wifi.State
	on       bool
	joined   string
	networks []wifi.Network
	// pending is the network the computer is joining, and when it
	// answers.
	pending  string
	password string
	answerAt time.Time
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	if a.pending != "" {
		if wait := a.answerAt.Sub(c.Now()); wait > 0 {
			c.After(wait)
		} else if a.pending == "The Neighbours" && a.password != "correct horse" {
			a.menu.Fail("Incorrect password")
			a.pending = ""
		} else {
			a.joined, a.pending = a.pending, ""
			for i := range a.networks {
				if a.networks[i].SSID == a.joined {
					a.networks[i].Known = true
				}
			}
		}
	}
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(3), t.Space(6)).AlignItems(ui.End).Children(func() {
		ui.Column(c).Radius(t.Space(3)).Background(t.Surface).Border(1, t.Border).Shadow(0, 8, 24, 0, ui.RGBA(0, 0, 0, 0.18)).Children(func() {
			ev := wifi.View(c, &a.menu, &a.on, &a.joined, a.networks)
			if ev.Join != nil {
				a.pending, a.password, a.answerAt = ev.Join.SSID, ev.Password, c.Now().Add(1200*time.Millisecond)
			}
			if ev.Settings {
				c.Toast("Opening Wi-Fi settings")
			}
		})
	})
}

func main() {
	a := &app{on: true, joined: "Home 5G", networks: []wifi.Network{
		{SSID: "Home 5G", Signal: 0.9, Secure: true, Known: true},
		{SSID: "Home", Signal: 0.7, Secure: true, Known: true},
		{SSID: "Office Guest", Signal: 0.2, Secure: true, Known: true},
		{SSID: "Café Lumière", Signal: 0.55},
		{SSID: "The Neighbours", Signal: 0.45, Secure: true},
		{SSID: "DIRECT-42-Printer", Signal: 0.3, Secure: true},
		{SSID: "Airport Free Wi-Fi", Signal: 0.12},
	}}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Wi-Fi",
			Width:   560,
			Height:  640,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
