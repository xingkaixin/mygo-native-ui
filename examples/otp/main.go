// OTP asks for the code a sign-in sent by email, as sign-in pages do.
// Type or paste it; the code is 428913, and any other is wrong.
//
//	go run ./examples/otp
package main

import (
	"fmt"
	"log"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/otp"
)

const want = "428913"

type app struct {
	code   string
	wrong  bool
	signed bool
	sent   time.Time
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Justify(ui.Center).AlignItems(ui.Center).Children(func() {
		ui.Column(c).Width(t.Space(110)).Gap(t.Space(4)).Padding(t.Space(8)).Radius(t.Space(3)).
			Background(t.Surface).Border(1, t.Border).AlignItems(ui.Center).Children(func() {
			if a.signed {
				ui.Text(c, "You're signed in").Bold().FontSize(t.FontSize + 6)
				if ui.Button(c, "Sign out").Clicked() {
					*a = app{sent: c.Now()}
				}
				return
			}
			ui.Text(c, "Check your email").Bold().FontSize(t.FontSize + 6)
			ui.Text(c, "Enter the 6-digit code we sent to k•••@example.com").TextColor(t.TextMuted)
			if code, ok := otp.Field(c, &a.code, 6).Done(); ok {
				a.signed = code == want
				a.wrong = !a.signed
			}
			if a.wrong && len(a.code) < 6 {
				a.wrong = false
			}
			if a.wrong {
				ui.Text(c, "That code is not right").TextColor(t.Danger)
			}
			wait := 30*time.Second - c.Now().Sub(a.sent)
			if wait > 0 {
				c.After(wait % time.Second)
				ui.Text(c, fmt.Sprintf("Resend code in %ds", (wait+time.Second-1)/time.Second)).TextColor(t.TextMuted)
			} else if ui.Button(c, "Resend code").Clicked() {
				a.sent, a.code, a.wrong = c.Now(), "", false
			}
		})
	})
}

func main() {
	a := &app{sent: time.Now()}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "OTP",
			Width:   640,
			Height:  480,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
