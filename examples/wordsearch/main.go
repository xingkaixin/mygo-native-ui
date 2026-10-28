// Wordsearch hides the names of thirteen packages of Go's standard
// library in a grid: drag across a name to find it. New Puzzle hides
// them again.
//
//	go run ./examples/wordsearch
package main

import (
	"log"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/wordsearch"
)

var packages = []string{"bufio", "bytes", "context", "errors", "embed", "flag", "image", "math", "slices", "strings", "sync", "time", "unicode"}

type app struct {
	search wordsearch.State
	puzzle wordsearch.Puzzle
}

func (a *app) deal(seed uint64) {
	p, err := wordsearch.New(packages, 12, 12, seed)
	if err != nil {
		log.Fatal(err)
	}
	a.puzzle, a.search = p, wordsearch.State{}
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(8)).Gap(t.Space(5)).Children(func() {
		wordsearch.View(c, &a.search, a.puzzle)
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			if ui.Button(c, "New Puzzle").Clicked() {
				a.deal(uint64(c.Now().UnixNano()))
			}
			if a.search.Found() == len(a.puzzle.Words) {
				ui.Text(c, "All found").Bold().TextColor(t.Success)
			}
		})
	})
}

func main() {
	a := &app{}
	a.deal(uint64(time.Now().UnixNano()))
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Word Search",
			Width:   760,
			Height:  640,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
