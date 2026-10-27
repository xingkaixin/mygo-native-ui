// Slides presents a short talk on keeping houseplants alive: choose a
// slide in the strip, then Present for the presenter view with notes and
// a timer, or Play for the slides alone. The arrows move between the
// slides and Escape comes back.
//
//	go run ./examples/slides
package main

import (
	"fmt"
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/slides"
)

var (
	ink    = ui.RGB(28, 38, 32)
	leaf   = ui.RGB(46, 125, 80)
	pale   = ui.RGB(232, 243, 235)
	muted  = ui.RGB(110, 120, 114)
	accent = ui.RGB(214, 120, 60)
)

func title(text, sub string) func(p *ui.Painter, r ui.Rect) {
	return func(p *ui.Painter, r ui.Rect) {
		p.Fill(r, pale, 0)
		p.Fill(ui.Rect{X: r.X, Y: r.Y + r.H - 24, W: r.W, H: 24}, leaf, 0)
		p.RichText(r.X+80, r.Y+190, r.W-160, ui.Span{Text: text, Size: 64, Weight: 700, Color: ink})
		p.RichText(r.X+80, r.Y+290, r.W-160, ui.Span{Text: sub, Size: 28, Color: muted})
	}
}

func bullets(head string, items ...string) func(p *ui.Painter, r ui.Rect) {
	return func(p *ui.Painter, r ui.Rect) {
		p.RichText(r.X+80, r.Y+60, r.W-160, ui.Span{Text: head, Size: 44, Weight: 700, Color: ink})
		p.Fill(ui.Rect{X: r.X + 80, Y: r.Y + 130, W: 80, H: 6}, leaf, 3)
		y := r.Y + 180
		for _, it := range items {
			p.Fill(ui.Rect{X: r.X + 84, Y: y + 14, W: 12, H: 12}, accent, 6)
			_, h := p.RichText(r.X+116, y, r.W-200, ui.Span{Text: it, Size: 30, Color: ink})
			y += h + 22
		}
	}
}

// chart draws how often three plants want water, in days.
func chart(p *ui.Painter, r ui.Rect) {
	p.RichText(r.X+80, r.Y+60, 0, ui.Span{Text: "Days between waterings", Size: 44, Weight: 700, Color: ink})
	bars := []struct {
		name string
		days float32
	}{{"Fern", 3}, {"Pothos", 7}, {"Snake plant", 21}, {"Cactus", 28}}
	for i, b := range bars {
		y := r.Y + 170 + float32(i)*80
		p.RichText(r.X+80, y+10, 0, ui.Span{Text: b.name, Size: 26, Color: ink})
		w := b.days / 28 * 520
		p.Fill(ui.Rect{X: r.X + 300, Y: y, W: w, H: 48}, leaf, 6)
		p.RichText(r.X+312+w, y+10, 0, ui.Span{Text: fmt.Sprint(b.days), Size: 26, Weight: 600, Color: leaf})
	}
}

var deck = []slides.Slide{
	{Draw: title("Keeping Houseplants Alive", "Light, water and a little neglect"), Notes: "Welcome everyone. Ask who has killed a plant this year — most hands go up."},
	{Draw: bullets("Why plants die", "Too much water, far more often than too little", "Too little light for the plant chosen", "Pots without a hole to drain"), Notes: "Overwatering rots the roots. The leaves droop, which looks like thirst, so people water more."},
	{Draw: bullets("Light first", "South windows suit cacti and succulents", "East or west suit most leafy plants", "North light: ferns, pothos, snake plants"), Notes: "Move a plant before buying another. A meter from the window halves the light."},
	{Draw: chart, Notes: "Rough numbers for a warm room. Check the soil with a finger: water when the top few centimeters are dry."},
	{Draw: bullets("A routine that works", "Check the soil every Sunday", "Water deeply, then let it drain", "Feed once a month in spring and summer"), Notes: "The habit matters more than the schedule. Pick one day and look at every plant."},
	{Draw: title("Questions?", "Bring a leaf, not the whole plant"), Notes: "Hand out the cuttings at the end."},
}

type app struct {
	deck slides.State
	mode int
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	switch a.mode {
	case 1:
		ev := slides.Presenter(c, &a.deck, deck)
		ev.Fill()
		if ev.Ended {
			a.mode = 0
		}
		return
	case 2:
		ev := slides.Show(c, &a.deck, deck)
		ev.Fill()
		if ev.Ended {
			a.mode = 0
		}
		return
	}
	ui.Row(c).Fill().Background(t.Background).Children(func() {
		slides.Thumbnails(c, &a.deck, deck).Width(t.Space(50)).FillHeight().Padding(t.Space(2), 0)
		ui.Divider(c).Vertical()
		ui.Column(c).Grow(1).FillHeight().Padding(t.Space(6)).Gap(t.Space(4)).Children(func() {
			ui.Row(c).Gap(t.Space(2)).Children(func() {
				ui.Spacer(c)
				if ui.Button(c, "Present").Clicked() {
					a.mode = 1
					a.deck.Restart()
				}
				if ui.PrimaryButton(c, "Play").Clicked() {
					a.mode = 2
				}
			})
			slide := deck[a.deck.Current]
			ui.Box(c).Grow(1).FillWidth().Draw(func(p *ui.Painter, r ui.Rect) { slides.Paint(p, r, slide) })
			ui.Text(c, slide.Notes).TextColor(t.TextMuted)
		})
	})
}

func main() {
	a := &app{}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Slides",
			Width:   1100,
			Height:  680,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
