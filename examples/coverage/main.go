// Coverage shows how much of a small notes package its tests ran, from
// the profile go test -covermode=count -coverpkg=./... -coverprofile wrote
// for it, with the package's source beside it in notes/.
//
//	go run ./examples/coverage
package main

import (
	"embed"
	"errors"
	"log"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/coverage"
)

// notes holds the profile and the package's files, named .go.txt so that
// they are not built with this module.
//
//go:embed notes
var notes embed.FS

// module is the import path of the package the profile names.
const module = "example.com/notes/"

// app is the state the window shows: the profile and the view of it.
type app struct {
	profile  *coverage.Profile
	coverage coverage.State
}

func (a *app) view(c *ui.Context) {
	coverage.View(c, &a.coverage, a.profile, source).Fill()
}

// source returns the file of the notes package at path.
func source(path string) (string, error) {
	name, ok := strings.CutPrefix(path, module)
	if !ok {
		return "", errors.New(path + " is not in " + module)
	}
	b, err := notes.ReadFile("notes/" + name + ".txt")
	return string(b), err
}

func main() {
	out, err := notes.ReadFile("notes/cover.out")
	if err != nil {
		log.Fatal(err)
	}
	a := &app{}
	if a.profile, err = coverage.Parse(string(out)); err != nil {
		log.Fatal(err)
	}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Coverage — example.com/notes",
			Width:   1180,
			Height:  760,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
