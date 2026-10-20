// Flamegraph shows a CPU profile of a made-up notes server as a flame
// graph, or the stacks folded in the file given, as Brendan Gregg's
// stackcollapse scripts write them. Click a function to zoom to it, and
// search to light the functions matching.
//
//	go run ./examples/flamegraph [file.folded]
package main

import (
	"bytes"
	_ "embed"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/flamegraph"
)

//go:embed profile.folded
var profile []byte

type app struct {
	name  string
	root  *flamegraph.Node
	flame flamegraph.State
	query string
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Children(func() {
		ui.Row(c).Gap(t.Space(3)).Padding(t.Space(2.5), t.Space(3)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, a.name).Bold().SingleLine().Grow(1).Shrink(1)
			ui.SearchField(c, &a.query).Placeholder("Search functions").Width(t.Space(64))
		})
		ui.Divider(c)
		flamegraph.View(c, &a.flame, a.root, a.query).Grow(1)
	})
}

func main() {
	var r io.Reader = bytes.NewReader(profile)
	name := "notes-server CPU profile"
	if len(os.Args) > 1 {
		f, err := os.Open(os.Args[1])
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		r, name = f, filepath.Base(os.Args[1])
	}
	root, err := flamegraph.ParseFolded(r)
	if err != nil {
		log.Fatal(err)
	}
	a := &app{name: name, root: root}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Flame Graph",
			Width:   1100,
			Height:  640,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
