// Pixelart edits a potted plant of 32 by 32 pixels in a palette of 16
// colors, and saves it as a PNG eight times as large, where the user
// chooses.
//
//	go run ./examples/pixelart
package main

import (
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"os"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/pixelart"
)

const (
	ink = iota
	dusk
	slate
	mist
	paper
	soil
	clayDark
	clay
	clayLight
	leafDark
	leaf
	leafLight
	sprout
	sun
	sky
	rose
)

var palette = []color.RGBA{
	ink:       {0x1E, 0x1B, 0x2B, 0xFF},
	dusk:      {0x45, 0x3F, 0x5C, 0xFF},
	slate:     {0x7D, 0x78, 0x99, 0xFF},
	mist:      {0xC4, 0xC0, 0xD4, 0xFF},
	paper:     {0xF5, 0xF1, 0xE6, 0xFF},
	soil:      {0x5A, 0x34, 0x2B, 0xFF},
	clayDark:  {0x8E, 0x45, 0x30, 0xFF},
	clay:      {0xC2, 0x63, 0x3F, 0xFF},
	clayLight: {0xE9, 0x95, 0x63, 0xFF},
	leafDark:  {0x1D, 0x4A, 0x3B, 0xFF},
	leaf:      {0x2E, 0x7D, 0x4F, 0xFF},
	leafLight: {0x5F, 0xB8, 0x5C, 0xFF},
	sprout:    {0xB9, 0xE3, 0x7D, 0xFF},
	sun:       {0xF0, 0xC9, 0x4A, 0xFF},
	sky:       {0x4A, 0x8C, 0xD8, 0xFF},
	rose:      {0xDB, 0x5C, 0x7F, 0xFF},
}

// plant draws a terracotta pot with four leaves and a flower, outlined in
// ink.
func plant() *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, 32, 32))
	set := func(x, y, i int) { img.SetRGBA(x, y, palette[i]) }
	// A leaf grows from bx, by along angle a, pointed at both ends, with a
	// dark vein, lighter above it.
	leafAt := func(bx, by, a, long, wide float64) {
		cos, sin := math.Cos(a*math.Pi/180), math.Sin(a*math.Pi/180)
		nx, ny := sin, -cos
		if ny > 0 {
			nx, ny = -nx, -ny
		}
		for y := range 32 {
			for x := range 32 {
				dx, dy := float64(x)+0.5-bx, float64(y)+0.5-by
				u, v := dx*cos+dy*sin, dx*nx+dy*ny
				if u < 0 || u > long || math.Abs(v) > wide*math.Sin(math.Pi*u/long) {
					continue
				}
				switch {
				case math.Abs(v) < 0.5 && u < long*0.8:
					set(x, y, leafDark)
				case v > wide*0.45 && u > long*0.3 && u < long*0.65:
					set(x, y, sprout)
				case v > 0:
					set(x, y, leafLight)
				default:
					set(x, y, leaf)
				}
			}
		}
	}
	leafAt(15, 16, 240, 13, 2.8)
	leafAt(17, 16, -60, 13, 2.8)
	leafAt(15, 17.5, 200, 13, 2.8)
	leafAt(17, 17.5, -20, 13, 2.8)
	for y := 8; y <= 17; y++ {
		set(16, y, leafDark)
	}
	flower := []string{
		"..RRR..",
		".RPRRR.",
		"RPRYRRR",
		"RRYYYRR",
		"RRRYRRR",
		".RRRRR.",
		"..RRR..",
	}
	for y, row := range flower {
		for x, ch := range row {
			switch ch {
			case 'R':
				set(13+x, 1+y, rose)
			case 'P':
				set(13+x, 1+y, paper)
			case 'Y':
				set(13+x, 1+y, sun)
			}
		}
	}
	for x := 10; x <= 21; x++ {
		set(x, 18, soil)
	}
	for x := 8; x <= 23; x++ {
		set(x, 19, clayLight)
		set(x, 20, clay)
		set(x, 21, clayDark)
	}
	for y := 22; y <= 29; y++ {
		x0, x1 := 9+(y-22)/3, 22-(y-22)/3
		for x := x0; x <= x1; x++ {
			switch {
			case y == 22 || x >= x1-1:
				set(x, y, clayDark)
			case x == x0+1:
				set(x, y, clayLight)
			case y == 25 || y == 26:
				set(x, y, sky)
			default:
				set(x, y, clay)
			}
		}
	}
	// Outline what is drawn, on the clear pixels next to it.
	edge := image.NewRGBA(img.Rect)
	copy(edge.Pix, img.Pix)
	for y := range 32 {
		for x := range 32 {
			if img.RGBAAt(x, y).A != 0 {
				continue
			}
			for _, d := range []image.Point{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				if img.RGBAAt(x+d.X, y+d.Y).A != 0 {
					edge.SetRGBA(x, y, palette[ink])
					break
				}
			}
		}
	}
	return edge
}

type app struct {
	win    *mygo.Window
	editor pixelart.State
	sprite *image.RGBA
	saved  string
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(5)).Gap(t.Space(4)).Children(func() {
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "Potted plant").Bold().FontSize(t.FontSize + 6)
			ui.Text(c, "32 × 32, 16 colors").TextColor(t.TextMuted).Grow(1)
			ui.Text(c, a.saved).FontSize(t.FontSize - 1).TextColor(t.TextMuted).Shrink(1).SingleLine()
			if ui.PrimaryButton(c, "Save PNG").Clicked() {
				a.save()
			}
		})
		pixelart.View(c, &a.editor, a.sprite, palette).Grow(1).AlignSelf(ui.Stretch)
	})
}

// save asks where to save the sprite at 8×, on a goroutine of its own as
// the dialog waits for the user.
func (a *app) save() {
	img := pixelart.Scale(a.sprite, 8)
	go func() {
		path, err := mygo.Dialog.Save(mygo.SaveDialogOptions{
			Parent:      a.win,
			Title:       "Save PNG",
			DefaultPath: "plant.png",
			Filters:     []mygo.FileFilter{{Name: "PNG image", Extensions: []string{"png"}}},
		})
		if err == nil && path == "" {
			return
		}
		if err == nil {
			err = write(path, img)
		}
		a.win.Update(func() {
			a.saved = "Saved " + path
			if err != nil {
				a.saved = err.Error()
			}
		})
	}()
}

func write(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

func main() {
	a := &app{sprite: plant()}
	mygo.App.WhenReady(func() {
		a.win = mygo.NewWindow(mygo.WindowOptions{
			Title:   "Pixel Art",
			Width:   1040,
			Height:  720,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
