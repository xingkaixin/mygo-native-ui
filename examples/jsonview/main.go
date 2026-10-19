// Jsonview shows JSON as a tree: the file given as its argument, or a
// sample, an API's response. The bar below says the path of the row
// chosen; Cmd+C and the rows' menus copy values and paths.
//
//	go run ./examples/jsonview [file.json]
package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/jsonview"
)

// app is the state the window shows: the document, or why it did not
// parse, and the tree's state.
type app struct {
	name string
	doc  *jsonview.Document
	err  error
	tree jsonview.State
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Children(func() {
		ui.Text(c, a.name).Bold().SingleLine().Padding(8, 12)
		ui.Divider(c)
		if a.err != nil {
			ui.Text(c, a.err.Error()).TextColor(t.Danger).Padding(12).Grow(1)
			return
		}
		jsonview.View(c, a.doc, &a.tree).Grow(1)
		ui.Divider(c)
		path := a.tree.Path()
		if path == "" {
			path = "Choose a row to see its path"
		}
		ui.Text(c, path).Font("monospace").FontSize(12).TextColor(t.TextMuted).SingleLine().Padding(6, 12)
	})
}

const sample = `{
  "total_count": 3,
  "incomplete_results": false,
  "next_page": null,
  "items": [
    {
      "id": 1296269,
      "full_name": "octocat/Hello-World",
      "description": "My first repository on GitHub, kept as an example of everything a repository can have.",
      "private": false,
      "fork": false,
      "owner": {"login": "octocat", "id": 583231, "type": "User", "site_admin": false},
      "topics": ["octocat", "atom", "electron", "api"],
      "stargazers_count": 80,
      "watchers_count": 80,
      "score": 1.0,
      "license": {"key": "mit", "name": "MIT License", "spdx_id": "MIT"},
      "created_at": "2011-01-26T19:01:12Z"
    },
    {
      "id": 18221276,
      "full_name": "octocat/git-consortium",
      "description": "This repo is for demonstration purposes only.",
      "private": false,
      "fork": true,
      "owner": {"login": "octocat", "id": 583231, "type": "User", "site_admin": false},
      "topics": [],
      "stargazers_count": 26,
      "watchers_count": 26,
      "score": 0.62,
      "license": null,
      "created_at": "2014-03-28T17:55:38Z"
    },
    {
      "id": 20978623,
      "full_name": "octocat/hello-worId",
      "description": null,
      "private": false,
      "fork": false,
      "owner": {"login": "octocat", "id": 583231, "type": "User", "site_admin": false},
      "topics": ["demo"],
      "stargazers_count": 4,
      "watchers_count": 4,
      "score": 0.18,
      "license": {"key": "apache-2.0", "name": "Apache License 2.0", "spdx_id": "Apache-2.0"},
      "created_at": "2014-06-18T21:26:19Z"
    }
  ],
  "rate limit": {"limit": 30, "remaining": 29, "reset": 1.7e9}
}`

func main() {
	a := &app{name: "search/repositories"}
	data := []byte(sample)
	if len(os.Args) > 1 {
		var err error
		if data, err = os.ReadFile(os.Args[1]); err != nil {
			log.Fatal(err)
		}
		a.name = filepath.Base(os.Args[1])
	}
	a.doc, a.err = jsonview.Parse(data)
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "JSON",
			Width:   720,
			Height:  600,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
