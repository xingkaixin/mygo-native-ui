// Password is a sign-up form: type a password, show it with the eye, and
// watch the meter and the rules; the button creates the account once the
// password meets every rule and is at least Good.
//
//	go run ./examples/password
package main

import (
	"log"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/password"
)

type app struct {
	email, password string
	rules           []password.Rule
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ok := password.Strength(a.password) >= 3
	for _, r := range a.rules {
		ok = ok && r.Met(a.password)
	}
	ui.Column(c).Fill().Background(t.Background).AlignItems(ui.Center).Padding(t.Space(10), t.Space(6)).Children(func() {
		ui.Column(c).Width(t.Space(90)).Gap(t.Space(5)).Children(func() {
			ui.Text(c, "Create your account").Bold().FontSize(t.FontSize + 6)
			ui.Column(c).Gap(t.Space(1.5)).Children(func() {
				ui.Text(c, "Email").Bold()
				ui.TextInput(c, &a.email).Label("Email")
			})
			ui.Column(c).Gap(t.Space(1.5)).Children(func() {
				ui.Text(c, "Password").Bold()
				password.Field(c, &a.password, a.rules)
			})
			if ui.PrimaryButton(c, "Create account").Disabled(!ok).Clicked() {
				c.Toast("Account created for " + a.email)
			}
		})
	})
}

func main() {
	a := &app{email: "ada@example.com"}
	a.rules = []password.Rule{
		{Text: "At least 10 characters", Met: func(s string) bool { return utf8.RuneCountInString(s) >= 10 }},
		{Text: "A number or a symbol", Met: func(s string) bool {
			return strings.IndexFunc(s, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsSpace(r) }) >= 0
		}},
		{Text: "Not your email", Met: func(s string) bool {
			name, _, _ := strings.Cut(a.email, "@")
			return name == "" || !strings.Contains(strings.ToLower(s), strings.ToLower(name))
		}},
	}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Password",
			Width:   560,
			Height:  560,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
