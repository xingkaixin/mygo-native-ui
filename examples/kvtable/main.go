// Kvtable builds an HTTP request as Postman does: its URL, and its query
// parameters, headers and form fields in tables of keys and values, which
// turn on and off, drag into another order, and edit as text, from which
// it writes the curl command that sends it. Header names are suggested as
// they are typed, and a form field is text, a file, a secret or JSON.
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

// headers are the header names suggested as one is typed.
var headers = []string{
	"Accept", "Accept-Encoding", "Accept-Language", "Authorization", "Cache-Control",
	"Content-Type", "Cookie", "If-None-Match", "Origin", "Referer", "User-Agent",
	"X-GitHub-Api-Version", "X-Request-Id",
}

// kinds are what a form field is.
var kinds = []kvtable.Kind{
	{Name: "Text"},
	{Name: "File", Cell: kvtable.File},
	{Name: "Secret", Cell: kvtable.Secret},
	{Name: "JSON", Cell: kvtable.JSON},
}

// app is the request the window builds: the method, URL, query
// parameters, headers and form fields, which table shows, and whether as
// text.
type app struct {
	win     *mygo.Window
	method  string
	url     string
	params  []kvtable.Pair
	headers []kvtable.Pair
	form    []kvtable.Pair
	table   int
	bulk    bool
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(16).Gap(12).Children(func() {
		ui.Row(c).Gap(8).Children(func() {
			ui.Select(c, &a.method, []string{"GET", "POST", "PUT", "PATCH", "DELETE"}).Label("Method")
			ui.TextInput(c, &a.url).Grow(1).Label("URL").Placeholder("https://")
		})
		ui.Row(c).AlignItems(ui.Center).Children(func() {
			ui.Segmented(c, &a.table, "Params", "Headers", "Body").Label("Table")
			ui.Box(c).Grow(1)
			ui.Toggle(c, &a.bulk, "Bulk edit")
		})
		ui.Scroll(c).Grow(1).Children(func() {
			switch a.table {
			case 0:
				kvtable.EditorWith(c.Key("params"), &a.params, kvtable.Options{Reorder: true, Bulk: &a.bulk})
			case 1:
				kvtable.EditorWith(c.Key("headers"), &a.headers, kvtable.Options{
					KeyLabel: "Header", Keys: headers, Values: values, Reorder: true, Bulk: &a.bulk,
				})
			default:
				kvtable.EditorWith(c.Key("form"), &a.form, kvtable.Options{
					KeyLabel: "Field", Kinds: kinds, Choose: a.choose, Reorder: true, Bulk: &a.bulk,
				})
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

// values returns the values suggested for a header named key.
func values(key string) []string {
	switch strings.ToLower(key) {
	case "accept":
		return []string{"application/json", "application/vnd.github+json", "text/html", "*/*"}
	case "content-type":
		return []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data", "text/plain"}
	case "cache-control":
		return []string{"no-cache", "no-store", "max-age=0"}
	}
	return nil
}

// choose shows a dialog choosing a file, and sets the path picked.
func (a *app) choose(set func(path string)) {
	go func() {
		paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{Parent: a.win, Title: "Choose a file"})
		if err != nil || len(paths) == 0 {
			return
		}
		a.win.Update(func() { set(paths[0]) })
	}()
}

// curl returns the command that sends the request, with the parameters,
// headers and form fields that are on.
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
	for _, f := range a.form {
		if !f.On || f.Key == "" {
			continue
		}
		v := f.Value
		if f.Kind == "File" {
			v = "@" + v
		}
		parts = append(parts, "-F "+quote(f.Key+"="+v))
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
		form: []kvtable.Pair{
			{Key: "name", Value: "mygo", On: true, Kind: "Text"},
			{Key: "avatar", Value: "/tmp/avatar.png", On: true, Kind: "File"},
			{Key: "token", Value: "ghp_example", On: true, Kind: "Secret"},
			{Key: "meta", Value: `{"stars": 42}`, On: true, Kind: "JSON"},
		},
	}
	mygo.App.WhenReady(func() {
		a.win = mygo.NewWindow(mygo.WindowOptions{
			Title:   "Request",
			Width:   760,
			Height:  560,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
