// Sheet is a map with a sheet of places over it, as Maps on iPhone: drag
// the sheet between a peek, half the window and nearly all of it, flick
// it, or click its grabber. Searching raises it; choosing a place lowers
// it and marks the place on the map.
//
//	go run ./examples/sheet
package main

import (
	"log"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/sheet"
)

type place struct {
	name, kind string
	x, y       float32
}

var places = []place{
	{"Ferry Building", "Market · 0.8 km", 0.78, 0.30},
	{"Golden Gate Park", "Park · 4.1 km", 0.22, 0.58},
	{"Blue Bottle Coffee", "Café · 0.3 km", 0.62, 0.44},
	{"City Lights Books", "Bookstore · 1.2 km", 0.66, 0.22},
	{"Dolores Park", "Park · 2.6 km", 0.48, 0.74},
	{"Tartine Bakery", "Bakery · 2.8 km", 0.54, 0.80},
	{"Coit Tower", "Landmark · 1.5 km", 0.70, 0.14},
	{"Exploratorium", "Museum · 1.4 km", 0.86, 0.20},
}

var detents = []sheet.Detent{sheet.Height(116), sheet.Fraction(0.5), sheet.Fraction(0.92)}

type app struct {
	detent    int
	query     string
	chosen    int
	searching bool
}

func (a *app) view(c *ui.Context) {
	sheet.View(c, &a.detent, detents, func() { a.mapView(c) }, func() { a.places(c) }).Grow(1)
}

func (a *app) mapView(c *ui.Context) {
	chosen := a.chosen
	ui.Box(c).Fill().Draw(func(p *ui.Painter, r ui.Rect) {
		p.Fill(r, ui.RGB(236, 233, 224), 0)
		water := new(ui.Path).MoveTo(r.X, r.Y).LineTo(r.X+r.W, r.Y).LineTo(r.X+r.W, r.Y+r.H*0.55).
			CubeTo(r.X+r.W*0.9, r.Y+r.H*0.4, r.X+r.W*0.95, r.Y+r.H*0.1, r.X+r.W*0.6, r.Y+r.H*0.06).
			CubeTo(r.X+r.W*0.3, r.Y+r.H*0.02, r.X+r.W*0.2, r.Y+r.H*0.12, r.X, r.Y+r.H*0.1).Close()
		p.FillPath(water, ui.RGB(170, 211, 236))
		p.Fill(ui.Rect{X: r.X, Y: r.Y + r.H*0.52, W: r.W * 0.36, H: r.H * 0.12}, ui.RGB(196, 226, 178), 6)
		p.Fill(ui.Rect{X: r.X + r.W*0.44, Y: r.Y + r.H*0.7, W: r.W * 0.1, H: r.H * 0.07}, ui.RGB(196, 226, 178), 4)
		for i := 2; i < 14; i++ {
			y := r.Y + r.H*float32(i)/14
			p.Line(r.X, y, r.X+r.W, y, 3, ui.RGB(255, 255, 255))
		}
		for i := 1; i < 8; i++ {
			x := r.X + r.W*float32(i)/8
			p.StrokePolyline([][2]float32{{x, r.Y + r.H*0.12}, {x + r.W*0.05, r.Y + r.H}}, 3, ui.RGB(255, 255, 255))
		}
		p.StrokePolyline([][2]float32{{r.X, r.Y + r.H*0.9}, {r.X + r.W, r.Y + r.H*0.35}}, 7, ui.RGB(253, 214, 120))
		for i, pl := range places {
			x, y := r.X+r.W*pl.x, r.Y+r.H*pl.y
			size, color := float32(14), ui.RGB(234, 88, 12)
			if i == chosen {
				size, color = 22, ui.RGB(220, 38, 38)
			}
			p.Shadow(ui.Rect{X: x - size/2, Y: y - size/2, W: size, H: size}, size/2, 0, 1, 4, 0, ui.RGB(0, 0, 0).Alpha(0.3))
			p.Fill(ui.Rect{X: x - size/2, Y: y - size/2, W: size, H: size}, ui.RGB(255, 255, 255), size/2)
			p.Fill(ui.Rect{X: x - size/2 + 3, Y: y - size/2 + 3, W: size - 6, H: size - 6}, color, size/2-3)
		}
	})
}

func (a *app) places(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Padding(0, t.Space(5), t.Space(5)).Gap(t.Space(4)).Grow(1).Children(func() {
		search := ui.SearchField(c, &a.query).Placeholder("Search Maps")
		if f := search.Focused(); f != a.searching {
			if f {
				a.detent = len(detents) - 1
			}
			a.searching = f
		}
		ui.Row(c).Gap(t.Space(6)).Children(func() {
			for _, f := range []string{"Home", "Work", "Gym", "Add"} {
				ui.Column(c.Key(f)).Gap(t.Space(1)).AlignItems(ui.Center).Children(func() {
					ui.Box(c).Size(t.Space(13), t.Space(13)).Radius(t.Space(13) / 2).Background(t.Accent.Alpha(0.12)).Center().Children(func() {
						ui.Text(c, f[:1]).Bold().TextColor(t.Accent)
					})
					ui.Text(c, f).FontSize(t.FontSize - 1)
				})
			}
		})
		ui.Text(c, "Recents").Bold().TextColor(t.TextMuted)
		ui.Column(c).Children(func() {
			q := strings.ToLower(a.query)
			for i, pl := range places {
				if q != "" && !strings.Contains(strings.ToLower(pl.name), q) {
					continue
				}
				row := ui.Row(c.Key(i)).Padding(t.Space(2), t.Space(1)).Gap(t.Space(3)).AlignItems(ui.Center).Radius(t.Radius).Cursor(ui.CursorPointer)
				if row.Hovered() {
					row.Background(t.SurfaceHover)
				}
				if row.Clicked() {
					a.chosen, a.detent, a.query = i, 0, ""
				}
				row.Children(func() {
					ui.Box(c).Size(t.Space(8), t.Space(8)).Radius(t.Space(4)).Background(ui.RGB(234, 88, 12)).Center().Children(func() {
						ui.Text(c, pl.name[:1]).Bold().TextColor(ui.RGB(255, 255, 255))
					})
					ui.Column(c).Children(func() {
						ui.Text(c, pl.name)
						ui.Text(c, pl.kind).FontSize(t.FontSize - 1).TextColor(t.TextMuted)
					})
				})
			}
		})
	})
}

func main() {
	a := &app{chosen: -1}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Maps",
			Width:   420,
			Height:  760,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
