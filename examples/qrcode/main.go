// QRcode shares a Wi‑Fi network as a code a phone's camera joins, as
// system settings do. Change the network and the code follows.
//
//	go run ./examples/qrcode
package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/qrcode"
)

var security = []string{"WPA", "WEP", "None"}

type app struct {
	name, password string
	security       int
	hidden         bool
	level          int
}

// payload writes the network as the WIFI: text that phones read, its
// special characters escaped.
func (a *app) payload() string {
	esc := strings.NewReplacer(`\`, `\\`, `;`, `\;`, `,`, `\,`, `:`, `\:`, `"`, `\"`).Replace
	t := security[a.security]
	if t == "None" {
		t = "nopass"
	}
	s := "WIFI:T:" + t + ";S:" + esc(a.name) + ";"
	if t != "nopass" {
		s += "P:" + esc(a.password) + ";"
	}
	if a.hidden {
		s += "H:true;"
	}
	return s + ";"
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Row(c).Fill().Background(t.Background).Padding(t.Space(8)).Gap(t.Space(8)).AlignItems(ui.Center).Children(func() {
		ui.Column(c).Width(t.Space(90)).Gap(t.Space(4)).Children(func() {
			ui.Text(c, "Share a network").Bold().FontSize(t.FontSize + 6)
			ui.Form(c, func() {
				ui.Field(c, "Network", func() { ui.TextInput(c, &a.name) })
				ui.Field(c, "Security", func() { ui.Segmented(c, &a.security, security...) })
				if security[a.security] != "None" {
					ui.Field(c, "Password", func() { ui.TextInput(c, &a.password) })
				}
				ui.Field(c, "Hidden", func() { ui.Checkbox(c, &a.hidden, "Network is hidden") })
				ui.Field(c, "Correction", func() { ui.Segmented(c, &a.level, "L", "M", "Q", "H") }).
					Description("Higher levels still read when part of the code is covered.")
			})
		})
		ui.Column(c).Grow(1).AlignItems(ui.Center).Gap(t.Space(3)).Padding(t.Space(6)).Radius(t.Space(4)).
			Background(t.Surface).Border(1, t.Border).Children(func() {
			code, err := qrcode.Encode(a.payload(), qrcode.Level(a.level))
			if err != nil {
				ui.Text(c, err.Error()).TextColor(t.Danger)
				return
			}
			qrcode.View(c, code).Size(t.Space(64), t.Space(64)).Label("Code to join " + a.name)
			ui.Text(c, "Scan to join “"+a.name+"”").Bold()
			ui.Text(c, fmt.Sprintf("Version %d · %d×%d modules", code.Version(), code.Size, code.Size)).
				FontSize(t.FontSize - 1).FontFeatures("tnum").TextColor(t.TextMuted)
		})
	})
}

func main() {
	a := &app{name: "Studio 5G", password: "correct horse battery", level: 1}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "QR code",
			Width:   900,
			Height:  560,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
