// Pagesetup sets up the page a short article prints on: choose the
// paper and the orientation, drag a margin's line in the preview or type
// it, and scale the text, which flows again as the page changes.
//
//	go run ./examples/pagesetup
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/pagesetup"
)

const title = "Why A4 Folds Into A5"

var paragraphs = []string{
	"Every sheet in the A series has sides in the ratio of one to the square root of two. Cut one in half across its long side and both halves keep that ratio, so A4 folded once is A5, and two A4 pages side by side make A3.",
	"The series starts from A0, which has an area of one square meter. Each step down halves the area: A1 is half a square meter, A4 a sixteenth. Because the shape never changes, a page enlarged from A4 to A3 on a copier fills the new sheet exactly, with no strip of white left over.",
	"US Letter grew from a different habit. Its 8½ by 11 inches came from the size of moulds used by paper makers, and halving it gives a sheet of another shape. That is why a document laid out for Letter and printed on A4 loses a little at the bottom, or gains a band of white on the side.",
	"Margins hide most of the difference. With an inch on every side, text set for one size usually fits the other, which is also why the Normal margins of many word processors are an inch, or 25.4 millimeters, all round.",
}

type app struct {
	setup pagesetup.Setup
	view  pagesetup.State
}

func (a *app) draw(p *ui.Painter, r ui.Rect) {
	ink := ui.RGB(30, 30, 32)
	y := r.Y
	_, h := p.RichText(r.X, y, r.W, ui.Span{Text: title, Size: 22, Weight: 700, Color: ink})
	y += h + 4
	_, h = p.RichText(r.X, y, r.W, ui.Span{Text: "A note on paper sizes", Size: 11, Italic: true, Color: ink.Alpha(0.6)})
	y += h + 14
	for _, para := range paragraphs {
		_, h = p.RichText(r.X, y, r.W, ui.Span{Text: para, Size: 11, Color: ink})
		y += h + 9
	}
}

func (a *app) render(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(6)).Children(func() {
		pagesetup.View(c, &a.view, &a.setup, a.draw).Grow(1)
	})
}

func main() {
	a := &app{setup: pagesetup.Setup{Paper: pagesetup.Papers[4], Top: 72, Right: 72, Bottom: 72, Left: 72, Scale: 1}}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Page Setup",
			Width:   900,
			Height:  680,
			Content: ui.View(a.render),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
