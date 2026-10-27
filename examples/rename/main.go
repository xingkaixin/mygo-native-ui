// Rename renames the photos of a trip at once: replace IMG_ with a
// place, add a date before the names, or number them all as Lisbon 01,
// Lisbon 02 and so on. Clashing names stop the rename; Undo puts the old
// names back.
//
//	go run ./examples/rename
package main

import (
	"fmt"
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/rename"
)

type app struct {
	names   []string
	history [][]string
	renamer rename.State
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(6)).Gap(t.Space(4)).Children(func() {
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, fmt.Sprintf("Lisbon Trip — %d items", len(a.names))).FontSize(t.FontSize + 3).Bold().Grow(1)
			if ui.Button(c, "Undo").Disabled(len(a.history) == 0).Clicked() {
				a.names = a.history[len(a.history)-1]
				a.history = a.history[:len(a.history)-1]
			}
		})
		ev := rename.View(c, &a.renamer, a.names)
		ev.Grow(1)
		if ev.Renamed {
			a.history = append(a.history, a.names)
			a.names = ev.Names
			c.Toast(fmt.Sprintf("Renamed %d items", len(ev.Names)))
		}
	})
}

func main() {
	a := &app{names: []string{
		"IMG_4021.HEIC", "IMG_4022.HEIC", "IMG_4023.HEIC", "IMG_4031.HEIC", "IMG_4032.MOV",
		"IMG_4040.HEIC", "IMG_4041.HEIC", "IMG_4057.HEIC", "img_4058.jpg", "IMG_4060.HEIC",
		"Tram 28.HEIC", "Belém tower.HEIC", "IMG_4077.MOV", "IMG_4078.HEIC", "Pastéis.jpg",
		"IMG_4090.HEIC", "IMG_4091.HEIC", "Sunset Miradouro.HEIC",
	}}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Rename",
			Width:   900,
			Height:  640,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
