// Signature signs for a delivery: sign on the pad, then save the
// signature as a PNG on transparency, in black or blue ink.
//
//	go run ./examples/signature
package main

import (
	"image/png"
	"log"
	"os"
	"path/filepath"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/signature"
)

var inks = []ui.Color{ui.RGB(24, 24, 28), ui.RGB(28, 58, 150)}

type app struct {
	pad     signature.State
	strokes []signature.Stroke
	ink     int
	saved   *ui.Bitmap
	path    string
}

func (a *app) save() {
	img := signature.Image(a.strokes, inks[a.ink], 2)
	a.path = filepath.Join(os.TempDir(), "signature.png")
	f, err := os.Create(a.path)
	if err != nil {
		log.Print(err)
		return
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		log.Print(err)
		return
	}
	a.saved = ui.NewBitmap(img)
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(8)).Gap(t.Space(5)).Children(func() {
		ui.Column(c).Gap(t.Space(1)).Children(func() {
			ui.Text(c, "Proof of delivery").Bold().FontSize(t.FontSize + 6)
			ui.Text(c, "Order 20-4471 · 3 parcels · 12.4 kg").TextColor(t.TextMuted)
		})
		if signature.View(c, &a.pad, &a.strokes).Changed() {
			a.saved, a.path = nil, ""
		}
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "Ink").TextColor(t.TextMuted)
			if ui.Segmented(c, &a.ink, "Black", "Blue").Changed() {
				a.saved, a.path = nil, ""
			}
			ui.Box(c).Grow(1)
			if ui.Button(c, "Save PNG").Disabled(len(a.strokes) == 0).Clicked() {
				a.save()
			}
		})
		if a.saved != nil {
			ui.Row(c).Gap(t.Space(4)).AlignItems(ui.Center).Children(func() {
				w, h := a.saved.Size()
				ui.Image(c, a.saved).Size(float32(w)/2, float32(h)/2).Border(1, t.Border).Radius(t.Space(1))
				ui.Text(c, "Saved to "+a.path).FontSize(t.FontSize - 1).TextColor(t.TextMuted).Shrink(1)
			})
		}
	})
}

func main() {
	a := &app{}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Signature",
			Width:   640,
			Height:  600,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
