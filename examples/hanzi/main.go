// Hanzi teaches the stroke order of a few characters: choose one, watch
// its strokes drawn in order, then trace them yourself on the practice
// grid. The strokes here are drawn by hand for this example, on a grid of
// 100.
//
//	go run ./examples/hanzi
package main

import (
	"fmt"
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/hanzi"
)

type character struct {
	han, pinyin, meaning string
	strokes              [][]float32
}

var characters = []character{
	{"永", "yǒng", "forever", [][]float32{
		{44, 9, 55, 19},
		{28, 29, 57, 27, 57, 88, 49, 82},
		{14, 45, 37, 43, 25, 62, 10, 75},
		{84, 31, 71, 44, 59, 52},
		{59, 52, 70, 66, 82, 78, 94, 86},
	}},
	{"十", "shí", "ten", [][]float32{
		{14, 48, 86, 45},
		{50, 10, 50, 92},
	}},
	{"大", "dà", "big", [][]float32{
		{14, 38, 86, 36},
		{50, 8, 50, 40, 42, 62, 26, 80, 10, 91},
		{53, 44, 63, 64, 77, 80, 92, 89},
	}},
	{"人", "rén", "person", [][]float32{
		{51, 8, 49, 35, 39, 62, 23, 82, 8, 92},
		{49, 42, 60, 63, 74, 80, 92, 91},
	}},
	{"口", "kǒu", "mouth", [][]float32{
		{22, 22, 24, 82},
		{22, 22, 78, 20, 76, 82},
		{24, 78, 76, 78},
	}},
	{"中", "zhōng", "middle", [][]float32{
		{20, 30, 22, 68},
		{20, 30, 80, 28, 78, 68},
		{22, 64, 78, 64},
		{50, 6, 50, 96},
	}},
	{"山", "shān", "mountain", [][]float32{
		{50, 12, 50, 80},
		{20, 38, 20, 82, 82, 82},
		{80, 34, 80, 84},
	}},
	{"木", "mù", "tree", [][]float32{
		{12, 32, 88, 30},
		{50, 6, 50, 96},
		{48, 34, 36, 56, 22, 72, 8, 82},
		{52, 36, 66, 56, 80, 70, 92, 78},
	}},
}

func char(ch character) hanzi.Char {
	var out hanzi.Char
	for _, s := range ch.strokes {
		var pts []ui.Point
		for i := 0; i+1 < len(s); i += 2 {
			pts = append(pts, ui.Point{X: s[i] / 100, Y: s[i+1] / 100})
		}
		out.Strokes = append(out.Strokes, pts)
	}
	return out
}

type app struct {
	chosen int
	writer hanzi.State
	result string
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ch := characters[a.chosen]
	ui.Column(c).Fill().Background(t.Surface).Padding(t.Space(8)).Gap(t.Space(5)).AlignItems(ui.Center).Children(func() {
		ui.Row(c).Gap(t.Space(2)).Children(func() {
			for i, x := range characters {
				b := ui.Button(c.Key(i), x.han).FontSize(t.FontSize + 6)
				if i == a.chosen {
					b.Background(t.Accent.Alpha(0.15))
				}
				if b.Clicked() && i != a.chosen {
					a.chosen, a.result = i, ""
					a.writer.Animate()
				}
			}
		})
		ui.Text(c, fmt.Sprintf("%s · %s · %d strokes", ch.pinyin, ch.meaning, len(ch.strokes))).TextColor(t.TextMuted)
		ev := hanzi.View(c, &a.writer, char(ch))
		ev.Size(360, 360).Label(ch.han)
		if ev.Done {
			a.result = fmt.Sprintf("Well written! %d mistakes.", ev.Mistakes)
		}
		ui.Row(c).Gap(t.Space(3)).Children(func() {
			if ui.Button(c, "Animate").Clicked() {
				a.writer.Animate()
				a.result = ""
			}
			if ui.PrimaryButton(c, "Trace it").Clicked() {
				a.writer.Quiz()
				a.result = ""
			}
		})
		ui.Text(c, a.result).TextColor(t.Success)
	})
}

func main() {
	a := &app{}
	a.writer.Animate()
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Stroke Order",
			Width:   560,
			Height:  640,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
