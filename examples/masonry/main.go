// Masonry shows a board of two thousand pins in columns, as Pinterest
// does: pictures it draws, of sunsets, orbits, waves, bubbles and
// Bauhaus shapes, each as high as its own proportions and its caption
// make it, the columns reflowing as the window is resized.
//
//	go run ./examples/masonry
package main

import (
	"fmt"
	"log"
	"math"
	"math/rand/v2"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/masonry"
)

// pin is a picture on the board: its caption, who saved it, its
// proportions, width to height, what it shows, and the seed and hue it
// is drawn with; caption is the height its caption and name take, 0
// until measured.
type pin struct {
	title   string
	by      string
	ratio   float32
	kind    int
	seed    uint64
	hue     float32
	caption float32
}

// app is the pins, the board's state, and the column width the pins'
// captions were measured at.
type app struct {
	pins     []pin
	board    masonry.State
	measured float32
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	lineH := func(size float32, weight int) float32 {
		_, h := c.MeasureText(0, ui.Span{Text: "Ag", Size: size, Weight: weight})
		return h
	}
	title, by := lineH(t.FontSize, 600), lineH(t.FontSize-1, 0)
	inset := t.Space(1)
	ui.Column(c).Fill().Background(t.Background).Children(func() {
		ui.Row(c).Padding(t.Space(3), t.Space(4)).Gap(t.Space(2)).AlignItems(ui.End).Children(func() {
			ui.Text(c, "Inspiration").FontSize(t.FontSize + 6).FontWeight(700)
			ui.Text(c, fmt.Sprintf("%d pins", len(a.pins))).TextColor(t.TextMuted)
		})
		ui.Divider(c)
		masonry.View(c, &a.board, len(a.pins), 220, func(i int, w float32) float32 {
			if w != a.measured {
				a.measured = w
				for k := range a.pins {
					a.pins[k].caption = 0
				}
			}
			p := &a.pins[i]
			if p.caption == 0 {
				_, h := c.MeasureText(w-2*inset, ui.Span{Text: p.title, Size: t.FontSize, Weight: 600})
				p.caption = t.Space(2) + min(h, 2*title) + by
			}
			return w/p.ratio + p.caption
		}, func(i int) {
			card(c, &a.pins[i], inset)
		}).Grow(1)
	})
}

// card shows p: its picture over its caption, of two lines at most, and
// who saved it.
func card(c *ui.Context, p *pin, inset float32) {
	t := c.Theme()
	radius := t.Space(4)
	ui.Box(c).Grow(1).Label(p.title).Role(ui.RoleImage).Draw(func(pt *ui.Painter, r ui.Rect) {
		pt.Clip(r, radius, func() { picture(pt, r, p) })
	})
	ui.Column(c).Padding(t.Space(2), inset, 0).Children(func() {
		ui.Text(c, p.title).FontWeight(600).MaxLines(2)
		ui.Text(c, p.by).FontSize(t.FontSize - 1).TextColor(t.TextMuted).SingleLine()
	})
}

// picture draws what p shows in r.
func picture(pt *ui.Painter, r ui.Rect, p *pin) {
	rnd := rand.New(rand.NewPCG(p.seed, 3))
	f := func(lo, hi float32) float32 { return lo + (hi-lo)*rnd.Float32() }
	h := p.hue
	short := min(r.W, r.H)
	switch p.kind {
	case 0:
		pt.FillGradient(r, ui.LinearGradient{Angle: 180, Stops: []ui.GradientStop{
			{Color: ui.Oklch(0.45, 0.12, h+280), At: 0},
			{Color: ui.Oklch(0.72, 0.16, h+20), At: 0.6},
			{Color: ui.Oklch(0.88, 0.12, h+60), At: 1},
		}}, 0)
		sun := (&ui.Path{}).Circle(r.X+r.W*f(0.25, 0.75), r.Y+r.H*f(0.35, 0.55), short*f(0.1, 0.18))
		pt.FillPath(sun, ui.Oklch(0.96, 0.08, h+80))
		for k, l := range []float32{0.42, 0.28} {
			var hills ui.Path
			hills.MoveTo(r.X, r.Y+r.H)
			for x := float32(0); x <= 1.01; x += 0.125 {
				hills.LineTo(r.X+r.W*x, r.Y+r.H*f(0.55, 0.75)+float32(k)*r.H*0.12)
			}
			hills.LineTo(r.X+r.W, r.Y+r.H).Close()
			pt.FillPath(&hills, ui.Oklch(l, 0.06, h+240))
		}
	case 1:
		night := 240 + float32(math.Mod(float64(h), 60))
		pt.FillGradient(r, ui.LinearGradient{Angle: 160, From: ui.Oklch(0.3, 0.08, night), To: ui.Oklch(0.14, 0.04, night+30)}, 0)
		for range 30 {
			star := (&ui.Path{}).Circle(r.X+r.W*f(0, 1), r.Y+r.H*f(0, 1), f(0.5, 1.4))
			pt.FillPath(star, ui.RGB(255, 255, 255).Alpha(f(0.3, 0.8)))
		}
		cx, cy := r.X+r.W*f(0.4, 0.6), r.Y+r.H*f(0.4, 0.6)
		pt.FillPath((&ui.Path{}).Circle(cx, cy, short*0.1), ui.Oklch(0.85, 0.15, h+70))
		for k := 1; k <= 3; k++ {
			radius := short * (0.1 + 0.12*float32(k))
			pt.StrokePath((&ui.Path{}).Circle(cx, cy, radius), 1, ui.RGB(255, 255, 255).Alpha(0.25))
			a := float64(f(0, 2*math.Pi))
			x, y := cx+radius*float32(math.Cos(a)), cy+radius*float32(math.Sin(a))
			pt.FillPath((&ui.Path{}).Circle(x, y, short*f(0.025, 0.05)), ui.Oklch(0.7, 0.15, h+float32(k)*70))
		}
	case 2:
		pt.FillGradient(r, ui.LinearGradient{Angle: 180, From: ui.Oklch(0.93, 0.04, h+200), To: ui.Oklch(0.8, 0.08, h+190)}, 0)
		for k := range 5 {
			base := r.Y + r.H*(0.35+0.13*float32(k))
			amp, phase, freq := r.H*f(0.02, 0.05), f(0, 6), f(1.5, 3)
			var wave ui.Path
			wave.MoveTo(r.X, r.Y+r.H)
			for x := float32(0); x <= 1.001; x += 0.05 {
				wave.LineTo(r.X+r.W*x, base+amp*float32(math.Sin(float64(phase+x*freq*2*math.Pi))))
			}
			wave.LineTo(r.X+r.W, r.Y+r.H).Close()
			pt.FillPath(&wave, ui.Oklch(0.72-0.1*float32(k), 0.1, h+200+float32(k)*8))
		}
	case 3:
		pt.FillGradient(r, ui.LinearGradient{Angle: 135, From: ui.Oklch(0.7, 0.15, h), To: ui.Oklch(0.55, 0.17, h+60), Oklab: true}, 0)
		for range 14 {
			bubble := (&ui.Path{}).Circle(r.X+r.W*f(0, 1), r.Y+r.H*f(0, 1), short*f(0.04, 0.22))
			pt.FillPath(bubble, ui.RGB(255, 255, 255).Alpha(f(0.12, 0.35)))
		}
	default:
		colors := []ui.Color{ui.Oklch(0.6, 0.19, 30), ui.Oklch(0.83, 0.16, 90), ui.Oklch(0.48, 0.15, 260), ui.Oklch(0.25, 0.02, 260)}
		pt.Fill(r, ui.Oklch(0.93, 0.03, 85), 0)
		cols, rows := 2, max(2, int(math.Round(float64(2*r.H/r.W))))
		cw, ch := r.W/float32(cols), r.H/float32(rows)
		for i := range cols * rows {
			cell := ui.Rect{X: r.X + float32(i%cols)*cw, Y: r.Y + float32(i/cols)*ch, W: cw, H: ch}
			color := colors[rnd.IntN(len(colors))]
			size := min(cw, ch) * 0.8
			x, y := cell.X+(cw-size)/2, cell.Y+(ch-size)/2
			switch rnd.IntN(4) {
			case 0:
				pt.FillPath((&ui.Path{}).Circle(x+size/2, y+size/2, size/2), color)
			case 1:
				pt.Fill(ui.Rect{X: x, Y: y, W: size, H: size}, color, 0)
			case 2:
				pt.FillPath((&ui.Path{}).MoveTo(x, y+size).LineTo(x+size/2, y).LineTo(x+size, y+size).Close(), color)
			default:
				pt.Clip(ui.Rect{X: x, Y: y, W: size, H: size / 2}, 0, func() {
					pt.FillPath((&ui.Path{}).Circle(x+size/2, y+size/2, size/2), color)
				})
			}
		}
	}
}

// board returns n pins with captions and proportions made up from seed.
func board(n int, seed uint64) []pin {
	rnd := rand.New(rand.NewPCG(seed, 1))
	pick := func(s []string) string { return s[rnd.IntN(len(s))] }
	places := []string{"the dunes", "Lake Bled", "the Dolomites", "Big Sur", "the old harbor", "Kyoto", "the high desert", "Lofoten", "a quiet bay", "the salt flats"}
	titles := [][]string{
		{"Golden hour over %s", "Last light on %s", "Sunset hike above %s, worth every step", "Evening sky at %s"},
		{"Night sky study", "Orbits, a poster for the studio wall", "Planets in a row", "Small worlds, big quiet"},
		{"Waves off %s", "Morning swell at %s", "Layers of blue", "Tide chart colors for a bedroom"},
		{"Bubbles", "Soft gradient wallpaper", "Color palette for a spring wedding", "Fizz"},
		{"Bauhaus shapes", "Primary colors print", "Geometric quilt pattern, free template", "Shapes for a kids' room"},
	}
	people := []string{"Mara Lind", "Theo Park", "Ines Duarte", "Sam Okafor", "Yuki Mori", "Lena Vogel", "Ravi Shah", "Noor Haddad"}
	ratios := []float32{2.0 / 3, 3.0 / 4, 4.0 / 5, 1, 9.0 / 16, 4.0 / 3, 3.0 / 2, 1.0 / 2}
	pins := make([]pin, n)
	for i := range pins {
		kind := rnd.IntN(len(titles))
		title := pick(titles[kind])
		if strings.Contains(title, "%s") {
			title = fmt.Sprintf(title, pick(places))
		}
		pins[i] = pin{
			title: title,
			by:    fmt.Sprintf("%s · %d saves", pick(people), 3+rnd.IntN(2400)),
			ratio: ratios[rnd.IntN(len(ratios))],
			kind:  kind,
			seed:  rnd.Uint64(),
			hue:   rnd.Float32() * 360,
		}
	}
	return pins
}

func main() {
	a := &app{pins: board(2000, 7)}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Board",
			Width:   1100,
			Height:  760,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
