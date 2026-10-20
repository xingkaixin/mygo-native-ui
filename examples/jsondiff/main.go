// Jsondiff shows how two JSON documents differ: the files given as its
// arguments, or two versions of an API's response for a repository. N
// and Shift+N go to the next and previous change; the bar below says the
// path of the row chosen.
//
//	go run ./examples/jsondiff [a.json b.json]
package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/jsondiff"
)

// app is the state the window shows: the names of the documents, how
// they differ, and the view's state.
type app struct {
	names [2]string
	diff  *jsondiff.Diff
	state jsondiff.State
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Children(func() {
		ui.Row(c).Gap(8).Padding(8, 12).Children(func() {
			ui.Text(c, a.names[0]).Bold().SingleLine()
			ui.Text(c, "→").TextColor(t.TextMuted)
			ui.Text(c, a.names[1]).Bold().SingleLine()
		})
		ui.Divider(c)
		jsondiff.View(c, &a.state, a.diff).Grow(1)
		ui.Divider(c)
		ui.Row(c).Gap(12).Padding(6, 12).Children(func() {
			path := a.state.Path()
			if path == "" {
				path = "Choose a row to see its path"
			}
			ui.Text(c, path).Font("monospace").FontSize(12).TextColor(t.TextMuted).SingleLine().Grow(1).Shrink(1)
			ui.Text(c, "N, Shift+N: next and previous change").FontSize(12).TextColor(t.TextMuted).NoWrap()
		})
	})
}

const before = `{
  "id": 1296269,
  "name": "Hello-World",
  "full_name": "octocat/Hello-World",
  "private": false,
  "owner": {
    "login": "octocat",
    "id": 583231,
    "avatar_url": "https://avatars.githubusercontent.com/u/583231?v=4",
    "gravatar_id": "",
    "type": "User",
    "site_admin": false
  },
  "description": "My first repository on GitHub!",
  "fork": false,
  "homepage": "https://github.com",
  "size": 108,
  "stargazers_count": 1420,
  "watchers_count": 1420,
  "language": null,
  "has_issues": true,
  "has_wiki": true,
  "archived": false,
  "license": null,
  "topics": ["octocat", "atom", "electron"],
  "permissions": {"admin": false, "maintain": false, "push": false, "triage": false, "pull": true},
  "default_branch": "master",
  "created_at": "2011-01-26T19:01:12Z",
  "updated_at": "2024-03-11T08:15:27Z",
  "pushed_at": "2024-02-19T16:20:42Z",
  "network_count": 2310,
  "subscribers_count": 1730
}`

const after = `{
  "id": 1296269,
  "node_id": "MDEwOlJlcG9zaXRvcnkxMjk2MjY5",
  "name": "Hello-World",
  "full_name": "octocat/Hello-World",
  "private": false,
  "owner": {
    "login": "octocat",
    "id": 583231,
    "avatar_url": "https://avatars.githubusercontent.com/u/583231?v=4",
    "html_url": "https://github.com/octocat",
    "type": "Organization",
    "site_admin": false
  },
  "description": "This your first repo!",
  "fork": false,
  "homepage": "https://github.com",
  "size": 108,
  "stargazers_count": 2817,
  "watchers_count": 2817,
  "language": "HTML",
  "has_issues": true,
  "archived": false,
  "license": {"key": "mit", "name": "MIT License", "spdx_id": "MIT"},
  "topics": ["octocat", "atom", "api", "electron"],
  "visibility": "public",
  "permissions": {"admin": false, "maintain": false, "push": false, "triage": false, "pull": true},
  "default_branch": "main",
  "created_at": "2011-01-26T19:01:12Z",
  "updated_at": "2025-06-02T14:41:09Z",
  "pushed_at": "2025-05-30T09:12:55Z",
  "network_count": 2310,
  "subscribers_count": 1730
}`

func main() {
	a := &app{names: [2]string{"hello-world-2024-03.json", "hello-world-2025-06.json"}}
	docs := [2][]byte{[]byte(before), []byte(after)}
	if len(os.Args) > 2 {
		for i, name := range os.Args[1:3] {
			data, err := os.ReadFile(name)
			if err != nil {
				log.Fatal(err)
			}
			docs[i], a.names[i] = data, filepath.Base(name)
		}
	}
	a.diff = jsondiff.Compare(docs[0], docs[1])
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "JSON Diff",
			Width:   720,
			Height:  640,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
