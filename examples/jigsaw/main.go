// Jigsaw cuts a picture of balloons over hills into 12, 24 or 48 pieces.
//
//	go run ./examples/jigsaw
package main

import (
	"image"
	"log"
	"math"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/jigsaw"
)

const w, h = 600, 400

var grids = [][2]int{{4, 3}, {6, 4}, {8, 6}}

type app struct {
	photo  *image.RGBA
	puzzle jigsaw.State
	grid   int
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Padding(t.Space(6)).Gap(t.Space(4)).AlignItems(ui.Center).Justify(ui.Center).Background(t.Background).Children(func() {
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			ui.Segmented(c, &a.grid, "12", "24", "48").Label("Pieces")
			if ui.Button(c, "Shuffle").Clicked() {
				a.puzzle.Shuffle()
			}
		})
		g := grids[a.grid]
		jigsaw.View(c, &a.puzzle, a.photo, g[0], g[1])
	})
}

// balloons draws hot-air balloons over hills under a morning sky.
func balloons(c *ui.Context) {
	ui.Box(c).Fill().Draw(func(p *ui.Painter, r ui.Rect) {
		p.FillGradient(ui.Rect{W: w, H: h}, ui.LinearGradient{From: ui.Hex("#5AA9E6"), To: ui.Hex("#FFE3B3"), Angle: 180}, 0)
		var sun ui.Path
		sun.Circle(470, 120, 40)
		p.FillPath(&sun, ui.Hex("#FFF4D6"))
		for _, cl := range [][3]float32{{90, 70, 1}, {300, 50, 0.8}, {520, 210, 0.7}} {
			for _, d := range [][3]float32{{0, 0, 22}, {24, -8, 26}, {50, 2, 20}} {
				var puff ui.Path
				puff.Circle(cl[0]+d[0]*cl[2], cl[1]+d[1]*cl[2], d[2]*cl[2])
				p.FillPath(&puff, ui.RGBA(255, 255, 255, 0.85))
			}
		}
		layers := []struct {
			y, amp float32
			col    ui.Color
		}{{280, 30, ui.Hex("#8DB580")}, {320, 36, ui.Hex("#5E9A5A")}, {360, 26, ui.Hex("#3D7041")}}
		for k, l := range layers {
			var hill ui.Path
			hill.MoveTo(0, h)
			for x := float32(0); x <= w; x += 20 {
				y := l.y - l.amp*float32(0.5+0.5*math.Sin(float64(x)/80+float64(k)*2))
				hill.LineTo(x, y)
			}
			hill.LineTo(w, h)
			hill.Close()
			p.FillPath(&hill, l.col)
		}
		for _, b := range []struct {
			x, y, r float32
			a, b    string
		}{{150, 170, 44, "#E63946", "#F1FAEE"}, {330, 130, 34, "#F4A261", "#2A9D8F"}, {450, 250, 26, "#7B2CBF", "#FFD60A"}} {
			for k := range 6 {
				col := ui.Hex(b.a)
				if k%2 == 1 {
					col = ui.Hex(b.b)
				}
				x0 := b.x - b.r + float32(k)*b.r/3
				var stripe ui.Path
				stripe.MoveTo(x0, b.y)
				stripe.CubeTo(x0, b.y-b.r*1.4, x0+b.r/3, b.y-b.r*1.4, x0+b.r/3, b.y)
				stripe.CubeTo(x0+b.r/3, b.y+b.r*0.6, b.x, b.y+b.r*1.1, b.x, b.y+b.r*1.1)
				stripe.CubeTo(b.x, b.y+b.r*1.1, x0, b.y+b.r*0.6, x0, b.y)
				stripe.Close()
				p.FillPath(&stripe, col)
			}
			p.Fill(ui.Rect{X: b.x - b.r*0.18, Y: b.y + b.r*1.35, W: b.r * 0.36, H: b.r * 0.28}, ui.Hex("#6B4226"), 2)
			var ropes ui.Path
			ropes.MoveTo(b.x-b.r*0.3, b.y+b.r*0.95)
			ropes.LineTo(b.x-b.r*0.16, b.y+b.r*1.35)
			ropes.MoveTo(b.x+b.r*0.3, b.y+b.r*0.95)
			ropes.LineTo(b.x+b.r*0.16, b.y+b.r*1.35)
			p.StrokePath(&ropes, 1, ui.Hex("#6B4226"))
		}
	})
}

func main() {
	a := &app{photo: ui.Render(balloons, w, h, 1), grid: 1}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Jigsaw",
			Width:   880,
			Height:  640,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
