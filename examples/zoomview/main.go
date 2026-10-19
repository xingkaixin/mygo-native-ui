// Zoomview shows an image to zoom and pan: the file given as its
// argument, or the Mandelbrot set, which it draws at startup. Its pixels
// show as squares once zoomed past their own size.
//
//	go run ./examples/zoomview [image]
package main

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"math/cmplx"
	"os"
	"path/filepath"
	"runtime"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/zoomview"
)

// viewer is the state the window shows: the image, what it is, and how
// the view shows it.
type viewer struct {
	image   *ui.Bitmap
	caption string
	zoom    zoomview.State
}

func (v *viewer) view(c *ui.Context) {
	t := c.Theme()
	w, h := v.image.Size()
	zoomKey := "Ctrl"
	if runtime.GOOS == "darwin" {
		zoomKey = "⌘"
	}
	ui.Column(c).Fill().Background(t.Background).Children(func() {
		ui.Row(c).Padding(8, 12).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, v.caption).Bold().SingleLine().Grow(1)
			if ui.Button(c, "−").Label("Zoom out").Clicked() {
				v.zoom.ZoomOut()
			}
			ui.Textf(c, "%.0f%%", v.zoom.Zoom()*100).FontFeatures("tnum").MinWidth(48).TextAlign(ui.Center)
			if ui.Button(c, "+").Label("Zoom in").Clicked() {
				v.zoom.ZoomIn()
			}
			if ui.Button(c, "Fit").Clicked() {
				v.zoom.Fit()
			}
			if ui.Button(c, "100%").Label("Actual size").Clicked() {
				v.zoom.ActualSize()
			}
		})
		ui.Divider(c)
		zoomview.View(c, &v.zoom, float32(w), float32(h), func() {
			ui.Image(c, v.image).Fill()
		}).Grow(1).Background(t.Surface).Label(v.caption).AutoFocus()
		ui.Divider(c)
		ui.Text(c, fmt.Sprintf("%d × %d. %s-scroll to zoom, scroll or drag to pan, double-click to zoom in and out; %s0 actual size, %s9 fit.",
			w, h, zoomKey, zoomKey, zoomKey)).FontSize(12).TextColor(t.TextMuted).Padding(6, 12)
	})
}

// mandelbrot draws the Mandelbrot set, w by h pixels.
func mandelbrot(w, h int) image.Image {
	m := image.NewRGBA(image.Rect(0, 0, w, h))
	const iterations = 96
	for py := range h {
		for px := range w {
			z, c := complex(0, 0), complex(-2.2+3.2*float64(px)/float64(w), -1+2*float64(py)/float64(h))
			n := 0
			for ; n < iterations && cmplx.Abs(z) <= 2; n++ {
				z = z*z + c
			}
			if n == iterations {
				m.Set(px, py, color.RGBA{A: 255})
				continue
			}
			f := float64(n) / iterations
			m.Set(px, py, color.RGBA{R: uint8(255 * f), G: uint8(255 * f * f), B: uint8(128 + 127*f), A: 255})
		}
	}
	return m
}

func main() {
	v := &viewer{caption: "Mandelbrot set"}
	if len(os.Args) > 1 {
		data, err := os.ReadFile(os.Args[1])
		if err != nil {
			log.Fatal(err)
		}
		if v.image, err = ui.DecodeBitmap(data); err != nil {
			log.Fatal(err)
		}
		v.caption = filepath.Base(os.Args[1])
	} else {
		v.image = ui.NewBitmap(mandelbrot(1600, 1000))
	}
	v.image.Pixelated()
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Zoom",
			Width:   900,
			Height:  640,
			Content: ui.View(v.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
