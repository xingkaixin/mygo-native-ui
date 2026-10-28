// Timecode shows the viewer of a two-minute cut with a timecode field
// under it and the cut's markers beside it. Drag the field to scrub, type
// a timecode or +/- an offset and press Enter, or pick a rate.
//
//	go run ./examples/timecode
package main

import (
	"log"
	"math"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/timecode"
)

var rates = []struct {
	name string
	rate timecode.Rate
	fps  float64
}{
	{"23.976", timecode.Rate{FPS: 24}, 24000.0 / 1001},
	{"25", timecode.Rate{FPS: 25}, 25},
	{"29.97 DF", timecode.Rate{FPS: 30, Drop: true}, 30000.0 / 1001},
	{"59.94 DF", timecode.Rate{FPS: 60, Drop: true}, 60000.0 / 1001},
}

var markers = []struct {
	name  string
	at    float64
	color ui.Color
}{
	{"Cold open", 0, ui.Hex("#e15759")},
	{"Title", 12.5, ui.Hex("#f28e2b")},
	{"Interview", 38, ui.Hex("#4e79a7")},
	{"B-roll", 74.2, ui.Hex("#59a14f")},
	{"Credits", 112, ui.Hex("#b07aa1")},
}

const length = 120.0

type app struct {
	tc    timecode.State
	frame int
	rate  int
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	rt := rates[a.rate]
	last := int(length * rt.fps)
	a.frame = min(a.frame, last)
	sec := float64(a.frame) / rt.fps
	scene := 0
	for i, m := range markers {
		if sec >= m.at {
			scene = i
		}
	}
	ui.Row(c).Fill().Background(t.Background).Padding(t.Space(8)).Gap(t.Space(6)).AlignItems(ui.Stretch).Children(func() {
		ui.Column(c).Grow(1).Gap(t.Space(3)).Children(func() {
			ui.Column(c).Grow(1).Radius(t.Radius).Background(ui.Hex("#141414")).Justify(ui.Center).AlignItems(ui.Center).Gap(t.Space(1)).Children(func() {
				ui.Text(c, markers[scene].name).FontSize(t.FontSize + 10).Bold().TextColor(markers[scene].color)
				ui.Text(c, "Interview_v3").FontSize(t.FontSize - 1).TextColor(ui.RGBA(255, 255, 255, 0.5))
			})
			ui.Box(c).Height(4).Radius(2).Background(t.Text.Alpha(0.08)).Draw(func(p *ui.Painter, r ui.Rect) {
				w := r.W * float32(sec/length)
				p.Fill(ui.Rect{X: r.X, Y: r.Y, W: w, H: r.H}, t.Accent, 2)
			})
			ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
				timecode.Field(c, &a.tc, &a.frame, rt.rate)
				ui.Text(c, "/ "+timecode.Format(last, rt.rate)).FontFeatures("tnum").TextColor(t.TextMuted).Grow(1)
				names := make([]string, len(rates))
				for i, r := range rates {
					names[i] = r.name
				}
				prev := a.rate
				ui.Segmented(c, &a.rate, names...)
				if a.rate != prev {
					a.frame = int(math.Round(sec * rates[a.rate].fps))
				}
			})
		})
		ui.Column(c).Width(t.Space(56)).Gap(t.Space(1)).Children(func() {
			ui.Text(c, "Markers").Bold().Padding(0, t.Space(2), t.Space(1))
			for i, m := range markers {
				row := ui.ButtonBase(c.Key(i)).Gap(t.Space(2)).Padding(t.Space(1.5), t.Space(2)).Radius(t.Radius).
					Justify(ui.Start).AlignItems(ui.Center).Label(m.name)
				if row.Hovered() || i == scene {
					row.Background(t.Text.Alpha(0.06))
				}
				if row.Clicked() {
					a.frame = int(math.Round(m.at * rt.fps))
				}
				row.Children(func() {
					ui.Box(c).Size(8, 8).Radius(4).Background(m.color)
					ui.Text(c, m.name).Grow(1)
					ui.Text(c, timecode.Format(int(math.Round(m.at*rt.fps)), rt.rate)).FontFeatures("tnum").FontSize(t.FontSize - 1).TextColor(t.TextMuted)
				})
			}
		})
	})
}

func main() {
	a := &app{rate: 2}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Timecode",
			Width:   1040,
			Height:  560,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
