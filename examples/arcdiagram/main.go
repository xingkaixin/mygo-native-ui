// Arcdiagram draws which of two dozen packages of Go's standard library
// import which, as Go 1.27 has them, grouped by what they are for.
//
//	go run ./examples/arcdiagram
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/arcdiagram"
)

var packages = []arcdiagram.Node{
	{Name: "errors", Group: "Runtime"},
	{Name: "sync", Group: "Runtime"},
	{Name: "time", Group: "Runtime"},
	{Name: "context", Group: "Runtime"},
	{Name: "reflect", Group: "Runtime"},
	{Name: "unicode", Group: "Text"},
	{Name: "unicode/utf8", Group: "Text"},
	{Name: "strconv", Group: "Text"},
	{Name: "strings", Group: "Text"},
	{Name: "bytes", Group: "Text"},
	{Name: "regexp", Group: "Text"},
	{Name: "fmt", Group: "Text"},
	{Name: "io", Group: "Files"},
	{Name: "bufio", Group: "Files"},
	{Name: "io/fs", Group: "Files"},
	{Name: "path", Group: "Files"},
	{Name: "path/filepath", Group: "Files"},
	{Name: "os", Group: "Files"},
	{Name: "math", Group: "Data"},
	{Name: "slices", Group: "Data"},
	{Name: "sort", Group: "Data"},
	{Name: "encoding/base64", Group: "Data"},
	{Name: "encoding/json", Group: "Data"},
	{Name: "net/url", Group: "Data"},
}

// imports go from the importing package to the imported one.
var imports = []arcdiagram.Link{
	{From: 12, To: 0}, {From: 12, To: 1}, {From: 17, To: 0}, {From: 17, To: 12}, {From: 17, To: 14},
	{From: 17, To: 19}, {From: 17, To: 1}, {From: 17, To: 2}, {From: 11, To: 0}, {From: 11, To: 12},
	{From: 11, To: 18}, {From: 11, To: 17}, {From: 11, To: 4}, {From: 11, To: 19}, {From: 11, To: 7},
	{From: 11, To: 1}, {From: 11, To: 6}, {From: 8, To: 0}, {From: 8, To: 12}, {From: 8, To: 1},
	{From: 8, To: 5}, {From: 8, To: 6}, {From: 9, To: 0}, {From: 9, To: 12}, {From: 9, To: 5},
	{From: 9, To: 6}, {From: 7, To: 0}, {From: 7, To: 6}, {From: 20, To: 19}, {From: 2, To: 0},
	{From: 2, To: 1}, {From: 13, To: 9}, {From: 13, To: 0}, {From: 13, To: 12}, {From: 13, To: 8},
	{From: 13, To: 6}, {From: 15, To: 0}, {From: 15, To: 6}, {From: 16, To: 0}, {From: 16, To: 14},
	{From: 16, To: 17}, {From: 16, To: 19}, {From: 16, To: 8}, {From: 16, To: 6}, {From: 14, To: 0},
	{From: 14, To: 12}, {From: 14, To: 15}, {From: 14, To: 19}, {From: 14, To: 2}, {From: 14, To: 6},
	{From: 22, To: 9}, {From: 22, To: 0}, {From: 22, To: 11}, {From: 22, To: 12}, {From: 22, To: 4},
	{From: 22, To: 7}, {From: 22, To: 8}, {From: 21, To: 12}, {From: 21, To: 19}, {From: 21, To: 7},
	{From: 23, To: 9}, {From: 23, To: 0}, {From: 23, To: 11}, {From: 23, To: 15}, {From: 23, To: 19},
	{From: 23, To: 7}, {From: 23, To: 8}, {From: 4, To: 0}, {From: 4, To: 18}, {From: 4, To: 7},
	{From: 4, To: 1}, {From: 4, To: 5}, {From: 4, To: 6}, {From: 3, To: 0}, {From: 3, To: 1},
	{From: 3, To: 2}, {From: 10, To: 9}, {From: 10, To: 12}, {From: 10, To: 19}, {From: 10, To: 7},
	{From: 10, To: 8}, {From: 10, To: 1}, {From: 10, To: 5}, {From: 10, To: 6},
}

type app struct {
	arcs arcdiagram.State
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(4)).Children(func() {
		arcdiagram.View(c, &a.arcs, packages, imports).Grow(1)
	})
}

func main() {
	a := &app{}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Arc diagram",
			Width:   900,
			Height:  560,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
