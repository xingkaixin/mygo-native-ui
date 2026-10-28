// Jumpbar walks the files of a small Go project. Click a name to go up
// to it, open a chevron to go to another item at its level, or click an
// item in the list to go into it. Narrow the window to fold the middle of
// the path.
//
//	go run ./examples/jumpbar
package main

import (
	"log"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/jumpbar"
)

// node is a file, or a folder of nodes.
type node struct {
	name string
	kids []node
}

func dir(name string, kids ...node) node { return node{name, kids} }

func files(names ...string) []node {
	out := make([]node, len(names))
	for i, n := range names {
		out[i] = node{name: n}
	}
	return out
}

var root = []node{
	dir("mygo-native-ui",
		dir("cmd", dir("gallery", files("main.go", "pages.go")...)),
		dir("internal",
			dir("render",
				dir("text", files("shape.go", "wrap.go", "bidi.go")...),
				dir("paint", files("path.go", "stroke.go", "gradient.go")...),
			),
			dir("platform", files("darwin.go", "windows.go", "linux.go")...),
		),
		dir("ui",
			dir("jumpbar", files("jumpbar.go")...),
			dir("tabs", files("tabs.go")...),
			dir("palette", files("palette.go", "match.go")...),
		),
		node{name: "go.mod"},
		node{name: "README.md"},
	),
}

type app struct{ path []string }

func (a *app) children(p []string) []string {
	level := root
	for _, name := range p {
		var next []node
		for _, n := range level {
			if n.name == name {
				next = n.kids
			}
		}
		level = next
	}
	out := make([]string, len(level))
	for i, n := range level {
		out[i] = n.name
	}
	return out
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Children(func() {
		ui.Row(c).AlignSelf(ui.Stretch).PaddingX(t.Space(2)).Background(t.Surface).
			BorderWidth(0, 0, 1, 0).BorderColor(t.Border).Children(func() {
			jumpbar.Bar(c, &a.path, a.children).Grow(1)
		})
		ui.Column(c).Grow(1).AlignSelf(ui.Stretch).Padding(t.Space(4)).Gap(t.Space(1)).Children(func() {
			kids := a.children(a.path)
			if len(kids) == 0 {
				ui.Text(c, strings.Join(a.path, "/")).TextColor(t.TextMuted)
				return
			}
			for _, name := range kids {
				row := ui.ButtonBase(c).AlignSelf(ui.Stretch).Justify(ui.Start).Padding(t.Space(1.5), t.Space(2)).
					Radius(t.Radius).Label(name)
				if row.Hovered() {
					row.Background(t.SurfaceHover)
				}
				row.Children(func() { ui.Text(c, name).PassThrough() })
				if row.Clicked() {
					a.path = append(a.path, name)
				}
			}
		})
	})
}

func main() {
	a := &app{path: []string{"mygo-native-ui", "internal", "render", "text", "shape.go"}}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Jump Bar",
			Width:   720,
			Height:  420,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
