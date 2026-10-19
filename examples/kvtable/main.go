// Kvtable builds an HTTP request as Postman does: its URL, and its query
// parameters and headers in tables of keys and values, which turn on and
// off, from which it writes the curl command that sends it.
//
//	go run ./examples/kvtable
package main

import (
	"log"
	"net/url"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/kvtable"
)

// app is the request the window builds: the method, URL, query
// parameters and headers, and which table shows.
type app struct {
	method  string
	url     string
	params  []kvtable.Pair
	headers []kvtable.Pair
	table   int
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(16).Gap(12).Children(func() {
		ui.Row(c).Gap(8).Children(func() {
			ui.Select(c, &a.method, []string{"GET", "POST", "PUT", "PATCH", "DELETE"}).Label("Method")
			ui.TextInput(c, &a.url).Grow(1).Label("URL").Placeholder("https://")
		})
		ui.Segmented(c, &a.table, "Params", "Headers").Label("Table")
		ui.Scroll(c).Grow(1).Children(func() {
			if a.table == 0 {
				kvtable.Editor(c.Key("params"), &a.params)
			} else {
				kvtable.Editor(c.Key("headers"), &a.headers)
			}
		})
		ui.Row(c).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "curl").Bold().FontSize(12).Grow(1)
			if ui.Button(c, "Copy").Clicked() {
				c.WriteClipboard(a.curl())
			}
		})
		ui.Text(c, a.curl()).Font("monospace").FontSize(12).Selectable().
			Padding(10).Radius(6).Background(t.Surface)
	})
}

// curl returns the command that sends the request, with the parameters
// and headers that are on.
func (a *app) curl() string {
	u, err := url.Parse(a.url)
	if err != nil {
		return "# " + err.Error()
	}
	q := u.Query()
	for _, p := range a.params {
		if p.On && p.Key != "" {
			q.Add(p.Key, p.Value)
		}
	}
	u.RawQuery = q.Encode()
	parts := []string{"curl"}
	if a.method != "GET" {
		parts = append(parts, "-X "+a.method)
	}
	for _, h := range a.headers {
		if h.On && h.Key != "" {
			parts = append(parts, "-H "+quote(h.Key+": "+h.Value))
		}
	}
	parts = append(parts, quote(u.String()))
	return strings.Join(parts, " \\\n  ")
}

// quote quotes s for a POSIX shell.
func quote(s string) string { return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'" }

func main() {
	a := &app{
		method: "GET",
		url:    "https://api.github.com/search/repositories",
		params: []kvtable.Pair{
			{Key: "q", Value: "mygo language:go", On: true},
			{Key: "sort", Value: "stars", On: true},
			{Key: "per_page", Value: "20"},
		},
		headers: []kvtable.Pair{
			{Key: "Accept", Value: "application/vnd.github+json", On: true},
			{Key: "Authorization", Value: "Bearer <token>"},
		},
	}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Request",
			Width:   720,
			Height:  520,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
