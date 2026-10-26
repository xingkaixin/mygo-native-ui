// Jwt decodes a JSON Web Token as it is pasted or typed: the buttons sign
// a fresh token and an expired one to start from.
//
//	go run ./examples/jwt
package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"log"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/jwt"
)

type app struct {
	token string
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Row(c).Fill().Background(t.Background).AlignItems(ui.Stretch).Children(func() {
		ui.Column(c).Width(t.Space(90)).Padding(t.Space(5)).Gap(t.Space(3)).BorderWidth(0, 1, 0, 0).BorderColor(t.Border).Children(func() {
			ui.Text(c, "Encoded").Bold()
			ui.TextArea(c, &a.token).Grow(1).Font("monospace").Placeholder("Paste a token").Label("Token")
			ui.Row(c).Gap(t.Space(2)).Children(func() {
				if ui.Button(c, "Fresh token").Clicked() {
					a.token = sign(time.Now(), 2*time.Hour)
				}
				if ui.Button(c, "Expired token").Clicked() {
					a.token = sign(time.Now().Add(-26*time.Hour), time.Hour)
				}
			})
		})
		ui.Scroll(c).Grow(1).Children(func() {
			jwt.View(c, a.token).Padding(t.Space(5))
		})
	})
}

// sign makes a token issued at iat, lasting for, signed with HS256.
func sign(iat time.Time, lasting time.Duration) string {
	enc := func(v any) string {
		b, _ := json.Marshal(v)
		return base64.RawURLEncoding.EncodeToString(b)
	}
	head := enc(struct {
		Alg string `json:"alg"`
		Typ string `json:"typ"`
	}{"HS256", "JWT"})
	body := enc(struct {
		Sub   string   `json:"sub"`
		Name  string   `json:"name"`
		Email string   `json:"email"`
		Roles []string `json:"roles"`
		Iss   string   `json:"iss"`
		Aud   string   `json:"aud"`
		Iat   int64    `json:"iat"`
		Exp   int64    `json:"exp"`
	}{"u_8f3a21", "Ada Lovelace", "ada@example.com", []string{"admin", "billing"}, "https://auth.example.com",
		"api.example.com", iat.Unix(), iat.Add(lasting).Unix()})
	mac := hmac.New(sha256.New, []byte("secret"))
	mac.Write([]byte(head + "." + body))
	return head + "." + body + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func main() {
	a := &app{token: sign(time.Now(), 2*time.Hour)}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "JWT",
			Width:   1000,
			Height:  680,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
