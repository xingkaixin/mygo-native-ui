// Rack lays out two racks of a small server room: a 42U rack of
// switches, servers, storage and a UPS, and a 24U one for the network.
// Drag a device to free units to move it; hover over one for its model
// and power.
//
//	go run ./examples/rack
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/rack"
)

type dev = rack.Device

type app struct {
	racks  [2]rack.Rack
	states [2]rack.State
	moves  int
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Scroll(c).Fill().Background(t.Background).Children(func() {
		ui.Column(c).Padding(t.Space(6)).Gap(t.Space(4)).Children(func() {
			ui.Row(c).Gap(t.Space(8)).AlignItems(ui.Start).Children(func() {
				for i := range a.racks {
					if rack.View(c.Key(i), &a.states[i], &a.racks[i]).Changed() {
						a.moves++
					}
				}
			})
			if a.moves > 0 {
				ui.Text(c, "Moved devices "+itoa(a.moves)+" times").TextColor(t.TextMuted)
			}
		})
	})
}

func itoa(n int) string {
	if n < 10 {
		return string(rune('0' + n))
	}
	return itoa(n/10) + string(rune('0'+n%10))
}

func main() {
	a := &app{racks: [2]rack.Rack{
		{Name: "A01 Compute", Units: 42, Watts: 6000, Devices: []dev{
			{Name: "sw-a01-1", Model: "48-port 25G switch", Role: "Network", U: 42, Height: 1, Watts: 220},
			{Name: "sw-a01-2", Model: "48-port 25G switch", Role: "Network", U: 41, Height: 1, Watts: 220},
			{Name: "patch-a01", Model: "24-port patch panel", Role: "Cabling", U: 40, Height: 1},
			{Name: "db-1", Model: "2U dual-socket server", Role: "Compute", U: 36, Height: 2, Watts: 650},
			{Name: "db-2", Model: "2U dual-socket server", Role: "Compute", U: 34, Height: 2, Watts: 650},
			{Name: "app-1", Model: "1U server", Role: "Compute", U: 31, Height: 1, Watts: 320},
			{Name: "app-2", Model: "1U server", Role: "Compute", U: 30, Height: 1, Watts: 320},
			{Name: "app-3", Model: "1U server", Role: "Compute", U: 29, Height: 1, Watts: 320},
			{Name: "gpu-1", Model: "4U GPU server", Role: "Compute", U: 22, Height: 4, Watts: 1600},
			{Name: "nas-1", Model: "4U 36-bay storage", Role: "Storage", U: 12, Height: 4, Watts: 480},
			{Name: "ups-a01", Model: "3U UPS 6 kVA", Role: "Power", U: 1, Height: 3},
		}},
		{Name: "A02 Network", Units: 24, Watts: 2000, Devices: []dev{
			{Name: "core-1", Model: "Core router", Role: "Network", U: 22, Height: 2, Watts: 450},
			{Name: "fw-1", Model: "Firewall", Role: "Security", U: 20, Height: 1, Watts: 150},
			{Name: "fw-2", Model: "Firewall", Role: "Security", U: 19, Height: 1, Watts: 150},
			{Name: "patch-a02", Model: "48-port patch panel", Role: "Cabling", U: 17, Height: 2},
			{Name: "oob-1", Model: "Console server", Role: "Management", U: 15, Height: 1, Watts: 40},
			{Name: "pdu-a02", Model: "Switched PDU", Role: "Power", U: 1, Height: 1},
		}},
	}}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Rack",
			Width:   880,
			Height:  860,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
