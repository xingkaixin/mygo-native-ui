// Thumbgrid shows the pictures of a folder as a grid of thumbnails, made
// in the background as they scroll into view: the folder given as its
// argument, or ten thousand views of the Mandelbrot set it draws, each
// as slow to make as a photo is to decode. The slider sizes them.
//
//	go run ./examples/thumbgrid [folder]
package main

import (
	"fmt"
	"image"
	"image/color"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"math"
	"math/cmplx"
	"math/rand/v2"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/thumbgrid"
)

// app is the pictures, named by their keys, the grid's state, and the
// thumbnails' size.
type app struct {
	keys   []string
	thumbs *thumbgrid.Loader[string]
	grid   ui.GridState
	chosen int
	size   float64
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	a.grid.Selected = &a.chosen
	ui.Column(c).Fill().Background(t.Background).Children(func() {
		ui.Row(c).Padding(8, 12).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, fmt.Sprintf("%d pictures", len(a.keys))).Bold().Grow(1)
			ui.Slider(c, &a.size, 80, 240).Width(160).Label("Thumbnail size")
		})
		ui.Divider(c)
		thumbgrid.View(c, &a.grid, a.thumbs, a.keys, float32(a.size), filepath.Base).Grow(1)
		ui.Divider(c)
		status := "Nothing chosen"
		if a.chosen >= 0 && a.chosen < len(a.keys) {
			status = a.keys[a.chosen]
		}
		ui.Text(c, status).FontSize(12).TextColor(t.TextMuted).SingleLine().Padding(6, 12)
	})
}

// folder returns the paths of the pictures in dir, by name.
func folder(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var paths []string
	for _, e := range entries {
		switch strings.ToLower(filepath.Ext(e.Name())) {
		case ".jpg", ".jpeg", ".png", ".gif":
			paths = append(paths, filepath.Join(dir, e.Name()))
		}
	}
	slices.Sort(paths)
	return paths, nil
}

func open(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	return img, err
}

// mandelbrot draws view i of the Mandelbrot set, around a point on its
// edge, from a few chosen, at a depth that i makes up.
func mandelbrot(i int) image.Image {
	r := rand.New(rand.NewPCG(uint64(i), 7))
	edges := []complex128{-0.7453 + 0.1127i, -0.1011 + 0.9563i, -1.2500 + 0.0200i, 0.2850 + 0.0100i, -0.7700 - 0.1000i, -1.7600}
	center := edges[r.IntN(len(edges))] + complex(r.NormFloat64(), r.NormFloat64())*0.002
	scale := 3 * math.Pow(0.6, float64(r.IntN(12)))
	hue := r.Float64()
	const w, h, limit = 640, 480, 160
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			c := center + complex((float64(x)/w-0.5)*scale, (float64(y)/h-0.5)*scale*h/w)
			z, n := complex(0, 0), 0
			for ; n < limit && cmplx.Abs(z) < 2; n++ {
				z = z*z + c
			}
			if n == limit {
				img.Set(x, y, color.RGBA{10, 10, 20, 255})
				continue
			}
			f := float64(n) / limit
			img.Set(x, y, hsv(hue+f*2, 0.7, math.Sqrt(f)))
		}
	}
	return img
}

// hsv returns the color of hue, from 0 to 1 and around again, saturation
// and value.
func hsv(hue, s, v float64) color.RGBA {
	hue = math.Mod(hue, 1) * 6
	k := func(n float64) uint8 {
		k := math.Mod(n+hue, 6)
		return uint8(255 * (v - v*s*max(0, min(k, 4-k, 1))))
	}
	return color.RGBA{k(5), k(3), k(1), 255}
}

func main() {
	a := &app{size: 140}
	if len(os.Args) > 1 {
		paths, err := folder(os.Args[1])
		if err != nil {
			log.Fatal(err)
		}
		a.keys, a.thumbs = paths, &thumbgrid.Loader[string]{Load: open}
	} else {
		for i := range 10000 {
			a.keys = append(a.keys, fmt.Sprintf("Mandelbrot %05d.png", i))
		}
		a.thumbs = &thumbgrid.Loader[string]{Load: func(key string) (image.Image, error) {
			var i int
			fmt.Sscanf(key, "Mandelbrot %d.png", &i)
			return mandelbrot(i), nil
		}}
	}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Pictures",
			Width:   900,
			Height:  640,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
