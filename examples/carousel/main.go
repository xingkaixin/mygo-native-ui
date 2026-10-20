// Carousel shows the welcome sheet of a notes app: four pages, each a
// drawing, a title and a few words, which the user drags or scrolls
// through, pages with the arrows, the dots or ← and →, or follows with
// Continue, until Get Started on the last closes the sheet.
//
//	go run ./examples/carousel
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/carousel"
)

// slide is a page of the sheet: what its drawing shows, its title and
// its words, and the hue of its drawing.
type slide struct {
	draw  func(p *ui.Painter, r ui.Rect, k kit)
	title string
	text  string
	hue   float32
}

var slides = []slide{
	{notes, "Welcome to Fieldnotes", "Capture ideas, lists and sketches the moment they come, in notes that open as fast as you can type.", 255},
	{search, "Find anything", "Search every note by a word in it, a tag, or where you wrote it. Results show as you type.", 160},
	{devices, "On all your devices", "Notes sync between your Mac, iPhone and iPad in seconds, and stay with you offline.", 300},
	{shield, "Private by design", "Notes are encrypted on your devices before they sync. Nobody else can read them, not even us.", 45},
}

// app is the state the window shows: whether the welcome sheet is open,
// and its page.
type app struct {
	welcome bool
	page    int
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Surface).Center().Gap(t.Space(3)).Children(func() {
		ui.Text(c, "No notes yet").FontSize(t.FontSize + 4).Bold()
		ui.Text(c, "The notes you write show here.").TextColor(t.TextMuted)
		if ui.Button(c, "Show Welcome").Clicked() {
			a.welcome, a.page = true, 0
		}
	})
	ui.DialogBase(c, &a.welcome, func(backdrop, panel ui.Element) {
		backdrop.Background(ui.RGBA(0, 0, 0, 0.4))
		panel.Radius(t.Space(3)).Background(t.Background).Border(1, t.Border).Clip().Shadow(0, 10, 30, 0, ui.RGBA(0, 0, 0, 0.3))
		carousel.View(c, &a.page, len(slides), func(i int) {
			a.slide(c, i)
		}).Size(480, 430).Padding(0, 0, t.Space(5)).Label("Welcome to Fieldnotes").AutoFocus()
	})
}

// slide builds page i: its drawing, title and words, and the button
// going on to the next page, or closing the sheet on the last.
func (a *app) slide(c *ui.Context, i int) {
	t := c.Theme()
	s := slides[i]
	k := kit{
		accent: ui.Oklch(0.62, 0.15, s.hue),
		card:   t.Background,
		line:   t.Text.Alpha(0.14),
		border: t.Border,
	}
	if t.Dark {
		k.accent = ui.Oklch(0.72, 0.13, s.hue)
		k.card = t.Surface
	}
	ui.Column(c).Fill().AlignItems(ui.Center).Padding(t.Space(8), t.Space(8), 0).Children(func() {
		ui.Box(c).Size(280, 190).Draw(func(p *ui.Painter, r ui.Rect) { s.draw(p, r, k) })
		ui.Text(c, s.title).FontSize(t.FontSize+9).Bold().TextAlign(ui.Center).Margin(t.Space(5), 0, t.Space(2), 0)
		ui.Text(c, s.text).TextColor(t.TextMuted).TextAlign(ui.Center).LineHeight(1.45).MaxWidth(340)
		ui.Spacer(c)
		label := "Continue"
		if i == len(slides)-1 {
			label = "Get Started"
		}
		if ui.PrimaryButton(c, label).Width(200).Clicked() {
			if i == len(slides)-1 {
				a.welcome = false
			} else {
				a.page = i + 1
			}
		}
	})
}

// kit is the colors a drawing is made of.
type kit struct {
	accent, card, line, border ui.Color
}

// backdrop draws the disc each drawing stands on.
func backdrop(p *ui.Painter, r ui.Rect, k kit) (cx, cy float32) {
	cx, cy = r.X+r.W/2, r.Y+r.H/2
	var disc ui.Path
	disc.Circle(cx, cy, r.H/2)
	p.FillPath(&disc, k.accent.Alpha(0.12))
	return cx, cy
}

// card draws a note: a card with a title bar and lines of text.
func card(p *ui.Painter, x, y, w, h float32, k kit) {
	p.Shadow(ui.Rect{X: x, Y: y, W: w, H: h}, 10, 0, 4, 12, 0, ui.RGB(0, 0, 0).Alpha(0.12))
	p.Fill(ui.Rect{X: x, Y: y, W: w, H: h}, k.card, 10)
	p.Stroke(ui.Rect{X: x, Y: y, W: w, H: h}, k.border, 10, 1)
	p.Fill(ui.Rect{X: x + 14, Y: y + 16, W: w * 0.45, H: 8}, k.accent, 4)
	for j, f := range []float32{0.8, 0.65, 0.75, 0.5} {
		if ly := y + 38 + float32(j)*16; ly+6 < y+h-10 {
			p.Fill(ui.Rect{X: x + 14, Y: ly, W: (w - 28) * f, H: 6}, k.line, 3)
		}
	}
}

func notes(p *ui.Painter, r ui.Rect, k kit) {
	cx, cy := backdrop(p, r, k)
	w, h := float32(118), float32(140)
	p.Transform(ui.Rotation(-10), cx, cy, func() { card(p, cx-w/2-28, cy-h/2+6, w, h, k) })
	p.Transform(ui.Rotation(7), cx, cy, func() { card(p, cx-w/2+26, cy-h/2+2, w, h, k) })
	card(p, cx-w/2, cy-h/2, w, h, k)
	var plus ui.Path
	plus.Circle(cx+w/2-4, cy+h/2-4, 18)
	p.FillPath(&plus, k.accent)
	white := ui.RGB(255, 255, 255)
	p.Line(cx+w/2-11, cy+h/2-4, cx+w/2+3, cy+h/2-4, 2.5, white)
	p.Line(cx+w/2-4, cy+h/2-11, cx+w/2-4, cy+h/2+3, 2.5, white)
}

func search(p *ui.Painter, r ui.Rect, k kit) {
	cx, cy := backdrop(p, r, k)
	w, h := float32(170), float32(132)
	x, y := cx-w/2-14, cy-h/2
	card(p, x, y, w, h, k)
	p.Fill(ui.Rect{X: x + 10, Y: y + 50, W: w * 0.7, H: 14}, k.accent.Alpha(0.3), 4)
	mx, my, mr := cx+44, cy+18, float32(30)
	var lens ui.Path
	lens.Circle(mx, my, mr)
	p.FillPath(&lens, k.card.Alpha(0.6))
	p.StrokePath(&lens, 7, k.accent)
	p.Transform(ui.Rotation(45), mx, my, func() { p.Fill(ui.Rect{X: mx + mr + 1, Y: my - 6, W: 30, H: 12}, k.accent, 6) })
}

func devices(p *ui.Painter, r ui.Rect, k kit) {
	cx, cy := backdrop(p, r, k)
	lx, ly, lw, lh := cx-108, cy-50, float32(140), float32(92)
	p.Fill(ui.Rect{X: lx, Y: ly, W: lw, H: lh}, k.card, 8)
	p.Stroke(ui.Rect{X: lx, Y: ly, W: lw, H: lh}, k.border, 8, 1)
	p.Fill(ui.Rect{X: lx + 8, Y: ly + 8, W: lw - 16, H: lh - 16}, k.accent.Alpha(0.16), 3)
	p.Fill(ui.Rect{X: lx + 18, Y: ly + 20, W: 54, H: 7}, k.accent, 3.5)
	p.Fill(ui.Rect{X: lx + 18, Y: ly + 36, W: 92, H: 6}, k.line, 3)
	p.Fill(ui.Rect{X: lx + 18, Y: ly + 50, W: 74, H: 6}, k.line, 3)
	var base ui.Path
	base.MoveTo(lx-14, ly+lh).LineTo(lx+lw+14, ly+lh).LineTo(lx+lw+4, ly+lh+10).LineTo(lx-4, ly+lh+10).Close()
	p.FillPath(&base, k.border)

	px, py, pw, ph := cx+50, cy-38, float32(56), float32(104)
	p.Shadow(ui.Rect{X: px, Y: py, W: pw, H: ph}, 12, 0, 4, 12, 0, ui.RGB(0, 0, 0).Alpha(0.12))
	p.Fill(ui.Rect{X: px, Y: py, W: pw, H: ph}, k.card, 12)
	p.Stroke(ui.Rect{X: px, Y: py, W: pw, H: ph}, k.border, 12, 1)
	p.Fill(ui.Rect{X: px + 5, Y: py + 12, W: pw - 10, H: ph - 24}, k.accent.Alpha(0.16), 3)
	p.Fill(ui.Rect{X: px + 11, Y: py + 22, W: 26, H: 6}, k.accent, 3)
	p.Fill(ui.Rect{X: px + 11, Y: py + 36, W: 34, H: 5}, k.line, 2.5)
	p.Fill(ui.Rect{X: px + 11, Y: py + 48, W: 28, H: 5}, k.line, 2.5)

	ax, ay := cx+20, cy-70
	var arc ui.Path
	arc.MoveTo(ax-34, ay+12).QuadTo(ax, ay-12, ax+34, ay+12)
	p.StrokePath(&arc, 3, k.accent)
	p.StrokePolyline([][2]float32{{ax + 22, ay + 12}, {ax + 34, ay + 12}, {ax + 32, ay}}, 3, k.accent)
}

func shield(p *ui.Painter, r ui.Rect, k kit) {
	cx, cy := backdrop(p, r, k)
	w, h := float32(112), float32(132)
	top := cy - h/2
	var s ui.Path
	s.MoveTo(cx, top).
		CubeTo(cx+w*0.25, top+14, cx+w*0.4, top+16, cx+w/2, top+16).
		CubeTo(cx+w/2, top+h*0.62, cx+w*0.28, top+h*0.86, cx, top+h).
		CubeTo(cx-w*0.28, top+h*0.86, cx-w/2, top+h*0.62, cx-w/2, top+16).
		CubeTo(cx-w*0.4, top+16, cx-w*0.25, top+14, cx, top).Close()
	p.FillPath(&s, k.accent)
	p.StrokePolyline([][2]float32{{cx - 22, cy + 2}, {cx - 6, cy + 18}, {cx + 24, cy - 14}}, 7, ui.RGB(255, 255, 255))
}

func main() {
	a := &app{welcome: true}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Fieldnotes",
			Width:   720,
			Height:  620,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
