// Envfile edits a project's three .env files side by side: fill in what
// production is missing, reveal a secret with its eye, add a variable to
// all three, and see each file as it would be saved below.
//
//	go run ./examples/envfile
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/envfile"
)

type app struct {
	editor envfile.State
	files  []envfile.File
	shown  int
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(6)).Gap(t.Space(5)).Children(func() {
		ui.Scroll(c).Grow(1).Basis(0).Children(func() {
			envfile.View(c, &a.editor, &a.files)
		})
		var names []string
		for _, f := range a.files {
			names = append(names, ".env."+f.Name)
		}
		ui.Segmented(c, &a.shown, names...).Label("File shown").AlignSelf(ui.Start)
		ui.Text(c, envfile.Format(a.files[a.shown].Entries)).Font("monospace").FontSize(t.FontSize - 1).Selectable().
			Padding(t.Space(3)).Radius(t.Radius).Background(t.Surface).Height(t.Space(40))
	})
}

func main() {
	a := &app{files: []envfile.File{
		{Name: "development", Entries: envfile.Parse(`# Local services
DATABASE_URL=postgres://localhost:5432/shop_dev
REDIS_URL=redis://localhost:6379
STRIPE_API_KEY=sk_test_51Hb0cal
LOG_LEVEL=debug
FEATURE_CHECKOUT_V2=true
`)},
		{Name: "staging", Entries: envfile.Parse(`DATABASE_URL=postgres://db.staging.internal:5432/shop
REDIS_URL=redis://cache.staging.internal:6379
STRIPE_API_KEY=sk_test_51HbStaging
LOG_LEVEL=info
LOG_LEVEL=warn
SENTRY_DSN="https://abc123@sentry.example.com/12"
`)},
		{Name: "production", Entries: envfile.Parse(`export DATABASE_URL=postgres://db.prod.internal:5432/shop
STRIPE_API_KEY=sk_live_51HbProd # rotate in March
LOG_LEVEL=warn
SENTRY_DSN='https://def456@sentry.example.com/12'
2FA_ISSUER=Shop
`)},
	}}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   ".env",
			Width:   1040,
			Height:  640,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
