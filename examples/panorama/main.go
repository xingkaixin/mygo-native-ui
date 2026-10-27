// Panorama walks between two made-up places drawn as 360° pictures: drag
// to look around, scroll to zoom, and click a hotspot to go to the other
// place. The pictures are painted when the example starts.
//
//	go run ./examples/panorama
package main

import (
	"image"
	"image/color"
	"log"
	"math"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/panorama"
)

// place is a picture to stand in, and its hotspots, which lead to the
// place of the same index in to.
type place struct {
	name  string
	photo *ui.Bitmap
	spots []panorama.Hotspot
	to    []int
}

// scene is how a place is painted: its sky, the sun, the hills' color
// and the ground, a lake or a meadow.
type scene struct {
	zenith, horizon, hills color.RGBA
	sunYaw, sunPitch       float64
	lake                   bool
}

func mix(a, b color.RGBA, t float64) color.RGBA {
	t = max(0, min(1, t))
	m := func(x, y uint8) uint8 { return uint8(float64(x) + (float64(y)-float64(x))*t) }
	return color.RGBA{m(a.R, b.R), m(a.G, b.G), m(a.B, b.B), 255}
}

// paint returns the scene as an equirectangular picture w wide.
func paint(sc scene, w int) *image.RGBA {
	h := w / 2
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	rad := math.Pi / 180
	sun := [3]float64{math.Cos(sc.sunPitch*rad) * math.Sin(sc.sunYaw*rad), math.Sin(sc.sunPitch * rad), math.Cos(sc.sunPitch*rad) * math.Cos(sc.sunYaw*rad)}
	sky := func(lon, lat float64) color.RGBA {
		c := mix(sc.horizon, sc.zenith, math.Pow(max(0, lat)/90, 0.6))
		d := [3]float64{math.Cos(lat*rad) * math.Sin(lon*rad), math.Sin(lat * rad), math.Cos(lat*rad) * math.Cos(lon*rad)}
		angle := math.Acos(max(-1, min(1, d[0]*sun[0]+d[1]*sun[1]+d[2]*sun[2]))) / rad
		c = mix(c, color.RGBA{255, 244, 214, 255}, math.Exp(-angle/12)*0.7)
		if angle < 2.2 {
			c = color.RGBA{255, 252, 240, 255}
		}
		return c
	}
	for y := range h {
		lat := 90 - (float64(y)+0.5)/float64(h)*180
		for x := range w {
			lon := (float64(x)+0.5)/float64(w)*360 - 180
			ridge := 3 + 2.5*math.Sin(lon*rad*3+1) + 1.5*math.Sin(lon*rad*7+2) + 0.8*math.Sin(lon*rad*17)
			var c color.RGBA
			switch {
			case lat >= 0:
				// The hills' edge covers as much of a pixel as it does.
				cover := (ridge-lat)/(180/float64(h)) + 0.5
				c = mix(sky(lon, lat), mix(sc.hills, sc.horizon, 0.35-lat/ridge*0.3), cover)
			case sc.lake:
				// The water mirrors the sky and hills above, darker and
				// rippled further out.
				m := -lat + 0.3*math.Sin(1/math.Max(0.02, -lat*rad)*6)
				cover := (ridge-m)/(180/float64(h)) + 0.5
				c = mix(sky(lon, m), mix(sc.hills, sc.horizon, 0.4), cover)
				c = mix(c, color.RGBA{20, 50, 70, 255}, 0.35+(-lat)/90*0.4)
			default:
				// A meadow mown in stripes, seen on a plane a unit under
				// the eye.
				dist := 1 / math.Tan(-lat*rad)
				stripe := math.Mod(math.Abs(dist*math.Cos(lon*rad))+100, 2) < 1
				c = color.RGBA{86, 140, 62, 255}
				if stripe {
					c = color.RGBA{102, 158, 72, 255}
				}
				c = mix(c, sc.horizon, math.Min(0.7, dist/40))
			}
			img.SetRGBA(x, y, c)
		}
	}
	return img
}

type app struct {
	places []place
	at     int
	pano   panorama.State
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	pl := a.places[a.at]
	ui.Column(c).Fill().Background(t.Background).Children(func() {
		ui.Row(c).Padding(t.Space(3), t.Space(4)).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, pl.name).Bold()
			ui.Spacer(c)
			ui.Text(c, "Drag to look around · scroll to zoom").TextColor(t.TextMuted)
		})
		ev := panorama.View(c, &a.pano, pl.photo, pl.spots)
		ev.Grow(1)
		if ev.Chosen >= 0 {
			yaw := a.pano.Yaw()
			a.at = pl.to[ev.Chosen]
			a.pano.Look(yaw, 0)
		}
	})
}

func main() {
	meadow := scene{zenith: color.RGBA{58, 112, 196, 255}, horizon: color.RGBA{196, 222, 240, 255}, hills: color.RGBA{70, 104, 92, 255}, sunYaw: 40, sunPitch: 35}
	lake := scene{zenith: color.RGBA{70, 70, 150, 255}, horizon: color.RGBA{250, 196, 150, 255}, hills: color.RGBA{60, 60, 90, 255}, sunYaw: -100, sunPitch: 6, lake: true}
	a := &app{places: []place{
		{name: "Meadow", photo: ui.NewBitmap(paint(meadow, 2048)),
			spots: []panorama.Hotspot{{Yaw: 120, Pitch: -12, Label: "Walk down to the lake"}, {Yaw: 40, Pitch: 35, Label: "Sun"}}, to: []int{1, 0}},
		{name: "Lake at dusk", photo: ui.NewBitmap(paint(lake, 2048)),
			spots: []panorama.Hotspot{{Yaw: -60, Pitch: -8, Label: "Back to the meadow"}}, to: []int{0}},
	}}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Panorama",
			Width:   960,
			Height:  620,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
