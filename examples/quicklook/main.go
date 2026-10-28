// Quicklook lists a folder's files: choose one and press Space to look at
// it, move through them with the arrows, and press Space again to close.
//
//	go run ./examples/quicklook
package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"log"
	"math/rand/v2"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/quicklook"
)

type app struct {
	look     quicklook.State
	looking  bool
	files    []quicklook.File
	selected int
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	if !a.looking && c.Shortcut(0, ui.KeySpace) {
		a.looking = true
	}
	if c.Shortcut(0, ui.KeyDown) {
		a.selected = min(a.selected+1, len(a.files)-1)
	}
	if c.Shortcut(0, ui.KeyUp) {
		a.selected = max(a.selected-1, 0)
	}
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(3)).Children(func() {
		for i, f := range a.files {
			row := ui.Row(c.Key(i)).Padding(t.Space(1.5), t.Space(3)).Radius(t.Radius).Gap(t.Space(3))
			if i == a.selected {
				row.Background(t.Accent.Alpha(0.16))
			}
			if row.Clicked() {
				a.selected = i
			}
			row.Children(func() {
				ui.Text(c, f.Name).Grow(1)
				ui.Text(c, f.Modified.Format("Jan 2, 3:04 PM")).TextColor(t.TextMuted)
			})
		}
		ui.Spacer(c)
		ui.Text(c, "Press Space to look at the file chosen").TextColor(t.TextMuted).AlignSelf(ui.Center)
	})
	if quicklook.View(c, &a.look, &a.looking, a.files, &a.selected).Open {
		c.Toast("Opening " + a.files[a.selected].Name)
	}
}

func sunrise() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 900, 600))
	for y := range 600 {
		for x := range 900 {
			v := float64(y) / 600
			dx, dy := float64(x-600), float64(y-380)
			if dx*dx+dy*dy < 90*90 {
				img.Set(x, y, color.RGBA{255, 230, 160, 255})
				continue
			}
			img.Set(x, y, color.RGBA{uint8(250 - 120*v), uint8(150 + 40*v), uint8(120 + 110*v), 255})
		}
	}
	var b bytes.Buffer
	png.Encode(&b, img)
	return b.Bytes()
}

const source = `package main

import "fmt"

// fib returns the n-th Fibonacci number.
func fib(n int) int {
	a, b := 0, 1
	for range n {
		a, b = b, a+b
	}
	return a
}

func main() {
	for i := range 10 {
		fmt.Println(i, fib(i))
	}
}
`

func main() {
	day := time.Date(2026, 10, 27, 0, 0, 0, 0, time.Local)
	archive := make([]byte, 48000)
	rng := rand.New(rand.NewPCG(1, 2))
	for i := range archive {
		archive[i] = byte(rng.Uint32())
	}
	a := &app{looking: true, files: []quicklook.File{
		{Name: "Sunrise.png", Modified: day.Add(7 * time.Hour), Data: sunrise()},
		{Name: "fib.go", Modified: day.Add(10 * time.Hour), Data: []byte(source)},
		{Name: "notes.md", Modified: day.Add(13 * time.Hour), Data: []byte("# Trip\n\n- Book the ferry\n- Pack the tent\n- Charge the camera\n")},
		{Name: "photos.zip", Modified: day.Add(15 * time.Hour), Data: archive},
	}}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Files",
			Width:   900,
			Height:  640,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
