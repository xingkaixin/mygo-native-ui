// Punnett crosses two pea plants, round and yellow over wrinkled and
// green, as Mendel did: two heterozygous parents give 9 : 3 : 3 : 1.
// Change either genotype, up to three genes, to see the square change.
//
//	go run ./examples/punnett
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/punnett"
)

type app struct {
	square         punnett.State
	mother, father string
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(8)).Children(func() {
		ui.Scroll(c).Grow(1).Children(func() {
			punnett.View(c, &a.square, &a.mother, &a.father)
		})
	})
}

func main() {
	a := &app{mother: "RrYy", father: "RrYy"}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Punnett Square",
			Width:   880,
			Height:  620,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
