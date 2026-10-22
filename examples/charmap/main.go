// Charmap writes text with characters from a character viewer, as
// macOS's Character Viewer does: a double click or Enter on a character
// inserts it into the field above, where the caret is, and Copy copies it.
//
//	go run ./examples/charmap
package main

import (
	"log"
	"slices"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/charmap"
)

// app is the text being written, where its caret goes in the next frame,
// and the viewer.
type app struct {
	text  string
	caret int
	moved bool
	chars charmap.State
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Children(func() {
		field := ui.TextInput(c, &a.text).Placeholder("Double-click a character to insert it").Label("Text").Margin(12)
		if a.moved {
			field.SetTextSelection(a.caret, a.caret)
			a.moved = false
		}
		ui.Divider(c)
		charmap.View(c, &a.chars).Grow(1)
		if r, ok := a.chars.Picked(); ok {
			start, end := field.TextSelection()
			text := []rune(a.text)
			a.text = string(slices.Concat(text[:start], []rune{r}, text[end:]))
			a.caret, a.moved = start+1, true
		}
	})
}

func main() {
	a := &app{chars: charmap.State{Recent: []rune{'→', '✓', '€', '—', '…', '°', 'π', '⌘'}}}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Charmap",
			Width:   900,
			Height:  620,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
