// Hexview shows the bytes of a file as a hex dump: the file given as its
// argument, or its own program, megabytes long. A field goes to an
// offset, and the bar below says what is selected.
//
//	go run ./examples/hexview [file]
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/hexview"
)

// app is the state the window shows: the file's bytes and name, the
// view's state, and the offset typed to go to.
type app struct {
	data   []byte
	name   string
	hex    hexview.State
	offset string
	err    string
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Children(func() {
		ui.Row(c).Padding(8, 12).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, a.name).Bold().SingleLine().Grow(1)
			ui.Text(c, a.err).TextColor(t.Danger).FontSize(12)
			if ui.TextInput(c, &a.offset).Width(140).Placeholder("Go to offset").Label("Go to offset").Submitted() {
				a.goTo()
			}
		})
		ui.Divider(c)
		hexview.View(c, &a.hex, a.data).Grow(1)
		ui.Divider(c)
		start, end := a.hex.Selection()
		status := fmt.Sprintf("Offset %#x (%d) · Selected %d of %d bytes", start, start, end-start, len(a.data))
		ui.Text(c, status).FontSize(12).FontFeatures("tnum").TextColor(t.TextMuted).Padding(6, 12)
	})
}

// goTo selects the byte at the offset typed, in hex with or without 0x.
func (a *app) goTo() {
	s := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(a.offset)), "0x")
	n, err := strconv.ParseInt(s, 16, 64)
	if err != nil || n < 0 || int(n) >= len(a.data) {
		a.err = "No such offset"
		return
	}
	a.err = ""
	a.hex.Select(int(n), int(n)+1)
}

func main() {
	path := ""
	if len(os.Args) > 1 {
		path = os.Args[1]
	} else {
		exe, err := os.Executable()
		if err != nil {
			log.Fatal(err)
		}
		path = exe
	}
	data, err := os.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}
	a := &app{data: data, name: filepath.Base(path)}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Hex",
			Width:   820,
			Height:  600,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
