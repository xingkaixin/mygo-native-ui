// Glyph browses the glyphs of Go Regular, the font of the Go project:
// pick one to see it against the font's metrics, turn on Points to see
// its quadratic outline, and search by character, code point or name.
//
//	go run ./examples/glyph
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/sfnt"

	"github.com/xingkaixin/mygo-native-ui/ui/glyph"
)

type app struct {
	glyphs glyph.State
	font   *sfnt.Font
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(6)).Children(func() {
		glyph.View(c, &a.glyphs, a.font)
	})
}

func main() {
	f, err := sfnt.Parse(goregular.TTF)
	if err != nil {
		log.Fatal(err)
	}
	a := &app{font: f}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Glyph",
			Width:   1000,
			Height:  640,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
