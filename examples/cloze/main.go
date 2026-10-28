// Cloze sets two exercises of a lesson: the water cycle, with words that
// take either spelling, and a Chinese passage of idioms, whose characters
// wrap one by one. Type in the blanks, Enter to go on, then Check.
//
//	go run ./examples/cloze
package main

import (
	"log"
	"strconv"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/cloze"
)

const water = `The sun warms the sea, and water {evaporates} into the air as vapour. As the vapour rises it cools and {condenses} into tiny droplets, which gather as {clouds}.

When the droplets grow too heavy they fall as {precipitation|rain}: rain, snow or hail. Water that runs off the land collects in rivers and flows back to the {sea|ocean}, and the cycle starts {over|again}.`

const idioms = `他做事总是很仔细，从来不会{画蛇添足}。这次比赛他{一马当先}，最后{名列前茅}。老师说，学习要{循序渐进}，不能{拔苗助长}。`

type exercise struct {
	title, text string
	state       cloze.State
	score       string
}

type app struct {
	exercises []*exercise
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Scroll(c).Fill().Background(t.Background).Children(func() {
		ui.Column(c).Padding(t.Space(8)).Gap(t.Space(6)).MaxWidth(t.Space(180)).Children(func() {
			ui.Text(c, "Lesson 4").FontSize(t.FontSize + 8).Bold()
			for i, e := range a.exercises {
				ui.Column(c.Key(i)).Gap(t.Space(4)).Padding(t.Space(5)).Radius(t.Space(3)).Border(1, t.Border).Children(func() {
					ui.Row(c).Children(func() {
						ui.Text(c, e.title).Bold().Grow(1)
						ui.Text(c, e.score).TextColor(t.TextMuted)
					})
					ev := cloze.View(c, &e.state, e.text)
					if ev.Checked {
						e.score = "Best " + strconv.Itoa(ev.Right) + "/" + strconv.Itoa(ev.Blanks)
					}
				})
			}
		})
	})
}

func main() {
	a := &app{exercises: []*exercise{
		{title: "The water cycle", text: water},
		{title: "成语填空", text: idioms},
	}}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Cloze",
			Width:   820,
			Height:  760,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
