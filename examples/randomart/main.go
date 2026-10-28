// Randomart draws the random art of three made-up host keys, as ssh shows
// it on connecting, and of a fingerprint you paste.
//
//	go run ./examples/randomart
package main

import (
	"encoding/base64"
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/randomart"
)

// keys are the base64 of made-up public keys, as authorized_keys lines
// hold them.
var keys = []struct {
	host, title, b64 string
}{
	{"build.example.com", "ED25519 256", "AAAAC3NzaC1lZDI1NTE5AAAAIGq8yC2xV3m8m1f0Qb7r4t2cV9pZ6w1kX0nQe3yHjT5s"},
	{"git.example.com", "ED25519 256", "AAAAC3NzaC1lZDI1NTE5AAAAIB4tQx7mW2v9Kc1pLr8sN3dF6hJ0yZqU5eGa7oVbXc2R"},
	{"backup.example.com", "ECDSA 256", "AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBN8vR2kQ7mZx3pL1wS9cT6yH4eJ0aU5fG2bV8nK3dX7qM1rP6sY9tW4oC2iE5lA0hB3jF8gD7kN6uQ1vZ2xRw0s="},
}

type app struct {
	pasted string
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Scroll(c).Fill().Background(t.Background).Children(func() {
		ui.Column(c).Padding(t.Space(6)).Gap(t.Space(6)).Children(func() {
			ui.Row(c).Wrap().Gap(t.Space(6)).Children(func() {
				for _, k := range keys {
					key, err := base64.StdEncoding.DecodeString(k.b64)
					if err != nil {
						log.Fatal(err)
					}
					ui.Column(c.Key(k.host)).Gap(t.Space(2)).Children(func() {
						ui.Text(c, k.host).FontWeight(600)
						randomart.View(c, k.title, randomart.Fingerprint(key))
					})
				}
			})
			ui.Column(c).Gap(t.Space(2)).Children(func() {
				ui.Text(c, "Fingerprint").FontWeight(600)
				ui.TextInput(c, &a.pasted).Placeholder("SHA256:…").Font("monospace")
				if fp, err := randomart.ParseFingerprint(a.pasted); err == nil {
					randomart.View(c, "SHA256", fp)
				} else if a.pasted != "" {
					ui.Text(c, err.Error()).TextColor(t.Danger)
				}
			})
		})
	})
}

func main() {
	a := &app{pasted: "SHA256:nThbg6kXUpJWGl7E1IGOCspRomTxdCARLviKw6E5SY8"}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Randomart",
			Width:   760,
			Height:  720,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
