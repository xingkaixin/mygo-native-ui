// Morse sends a message in Morse code: play it at the speed set and
// watch each letter light as it sounds, then send it back with the key
// and see it read.
//
//	go run ./examples/morse
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/audio"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/morse"
)

type app struct {
	morse morse.State
	text  string
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(8)).Gap(t.Space(4)).Children(func() {
		morse.View(c, &a.morse, &a.text)
		if err := audio.Err(); err != nil {
			ui.Text(c, err.Error()).FontSize(t.FontSize - 1).TextColor(t.TextMuted)
		}
	})
}

func main() {
	a := &app{text: "What hath God wrought"}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Morse",
			Width:   900,
			Height:  520,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
