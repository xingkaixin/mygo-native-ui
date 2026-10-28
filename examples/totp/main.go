// Totp lists the one-time codes of six made-up accounts read from their
// otpauth URIs, one of eight digits and one lasting a minute. Click a
// row to copy its code, and type above the list to filter it.
//
//	go run ./examples/totp
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/totp"
)

var uris = []string{
	"otpauth://totp/GitHub:ada@example.com?secret=JBSWY3DPEHPK3PXP&issuer=GitHub",
	"otpauth://totp/Fastmail:ada@example.com?secret=KRSXG5CTMVRXEZLUKN2XAZLSONSWG4TF&issuer=Fastmail",
	"otpauth://totp/Cloudflare:ops@example.com?secret=GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ&issuer=Cloudflare",
	"otpauth://totp/AWS:root?secret=MFRGGZDFMZTWQ2LKNNWG23TPOBYXE43UOV3HO6DZPI&issuer=AWS&digits=8&algorithm=SHA256",
	"otpauth://totp/Bank%20of%20Nowhere:ada?secret=NBSWY3DPO5XXE3DEBI&issuer=Bank%20of%20Nowhere&period=60",
	"otpauth://totp/home-server?secret=ONSWG4TFOQYTEMY",
}

type app struct {
	accounts []totp.Account
	codes    totp.State
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Scroll(c).Fill().Background(t.Background).Children(func() {
		ui.Column(c).Padding(t.Space(6)).Children(func() {
			totp.List(c, &a.codes, a.accounts)
		})
	})
}

func main() {
	a := &app{}
	for _, u := range uris {
		acct, err := totp.ParseURI(u)
		if err != nil {
			log.Fatal(err)
		}
		a.accounts = append(a.accounts, acct)
	}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Codes",
			Width:   480,
			Height:  620,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
