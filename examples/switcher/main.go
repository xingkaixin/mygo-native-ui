// Switcher goes between open documents with Ctrl+Tab: hold Ctrl and
// press Tab to go on, Shift+Tab to go back, and let go of Ctrl to open
// the document. A quick Ctrl+Tab goes back to the document before.
//
//	go run ./examples/switcher
package main

import (
	"log"
	"slices"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/switcher"
)

func icon(body string) *ui.SVG {
	return ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"><path d="M6 2.5h8l4.5 4.5v14.5H6z"/><path d="M14 2.5V7h4.5"/>` + body + `</svg>`))
}

var (
	code  = icon(`<path d="M10 12l-2 2 2 2M14 12l2 2-2 2"/>`)
	text  = icon(`<path d="M9 12h6M9 15h6M9 18h4"/>`)
	data  = icon(`<path d="M10 11.5c-1 0-1 1-1 1.5s-.5 1-1 1c.5 0 1 .5 1 1s0 1.5 1 1.5M14 11.5c1 0 1 1 1 1.5s.5 1 1 1c-.5 0-1 .5-1 1s0 1.5-1 1.5"/>`)
	image = icon(`<circle cx="10" cy="12" r="1.3"/><path d="M8 18.5l3-3 2 2 2.5-3 1.5 4"/>`)
)

type doc struct {
	name, folder string
	icon         *ui.SVG
	body         string
}

type app struct {
	// docs is most recent first, the open one first of all.
	docs     []doc
	switcher switcher.State
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	items := make([]switcher.Item, len(a.docs))
	for i, d := range a.docs {
		items[i] = switcher.Item{Title: d.name, Detail: d.folder, Icon: d.icon}
	}
	if i, ok := switcher.Show(c, &a.switcher, ui.Ctrl, items).Chosen(); ok && i > 0 {
		d := a.docs[i]
		a.docs = slices.Insert(slices.Delete(a.docs, i, i+1), 0, d)
	}

	open := a.docs[0]
	ui.Column(c).Fill().Background(t.Background).Children(func() {
		ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Padding(t.Space(3), t.Space(4)).Children(func() {
			ui.Icon(c, open.icon).Size(t.Space(4), t.Space(4))
			ui.Text(c, open.name).Bold()
			ui.Text(c, open.folder).TextColor(t.TextMuted)
			ui.Spacer(c)
			ui.Text(c, "Ctrl+Tab to switch").FontSize(t.FontSize - 1).TextColor(t.TextMuted)
		})
		ui.Divider(c)
		ui.Text(c, open.body).Font("Menlo").Padding(t.Space(4)).TextColor(t.Text.Alpha(0.85))
	})
}

func main() {
	a := &app{docs: []doc{
		{"main.go", "cmd/server", code, "package main\n\nfunc main() {\n\tlog.Fatal(serve(\":8080\"))\n}"},
		{"README.md", "", text, "# Server\n\nA small HTTP server for the team's dashboards."},
		{"handlers.go", "internal/http", code, "func health(w http.ResponseWriter, r *http.Request) {\n\tw.WriteHeader(http.StatusOK)\n}"},
		{"config.json", "deploy", data, "{\n  \"port\": 8080,\n  \"timeout\": \"30s\"\n}"},
		{"CHANGELOG.md", "", text, "## 1.4.0\n\n- Health checks\n- Graceful shutdown"},
		{"logo.svg", "assets", image, "<svg viewBox=\"0 0 24 24\">…</svg>"},
		{"store.go", "internal/db", code, "type Store struct {\n\tdb *sql.DB\n}"},
	}}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Switcher",
			Width:   760,
			Height:  480,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
