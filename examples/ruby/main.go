// Ruby sets readings over Chinese and Japanese: Li Bai's Quiet Night
// Thought with pinyin over each character, and a note on a museum visit
// with furigana over its kanji. A switch shows and hides the readings,
// which then show in a tip over a word under the pointer, and a slider
// sizes the text. The markup at the bottom sets as it is typed; drag
// across any of them to select and copy.
//
//	go run ./examples/ruby
package main

import (
	"fmt"
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/ruby"
)

const poem = `静(jìng)夜(yè)思(sī)　李(lǐ)白(bái)
床(chuáng)前(qián)明(míng)月(yuè)光(guāng)，疑(yí)是(shì)地(dì)上(shàng)霜(shuāng)。
举(jǔ)头(tóu)望(wàng)明(míng)月(yuè)，低(dī)头(tóu)思(sī)故(gù)乡(xiāng)。`

const note = `{日曜日|にちようび}、{上野|うえの}の{美術館|び|じゅつ|かん}で{葛飾北斎|かつしかほくさい}の{浮世絵|うきよえ}を{見|み}ました。{雨上|あめあ}がりの{空|そら}は{澄|す}んでいて、{帰|かえ}り{道|みち}からは{富士山|ふじさん}まで{見|み}えました。`

const markup = `{東京|とうきょう}で{漢字|かん|じ}を{勉強|べんきょう}しています。
我(wǒ)在(zài)学(xué)中(zhōng)文(wén)。`

type app struct {
	show       bool
	size       float64
	poem, note []ruby.Run
	markup     string
	runs       []ruby.Run
	views      [3]ruby.State
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	for i := range a.views {
		a.views[i].Hidden = !a.show
	}
	size := float32(a.size)
	ui.Column(c).Fill().Background(t.Background).Children(func() {
		ui.Row(c).Gap(t.Space(3)).Padding(t.Space(3), t.Space(6)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "Readings")
			ui.Switch(c, &a.show).Label("Readings")
			ui.Box(c).Width(t.Space(4))
			ui.Text(c, "Size")
			ui.Slider(c, &a.size, 14, 40).Label("Size").Width(t.Space(48))
			ui.Text(c, fmt.Sprintf("%.0f", a.size)).TextColor(t.TextMuted).FontFeatures("tnum").Width(t.Space(6))
		})
		ui.Divider(c)
		ui.Scroll(c).Grow(1).Children(func() {
			ui.Column(c).Gap(t.Space(6)).Padding(t.Space(5), t.Space(6), t.Space(8)).Children(func() {
				section(c, "Quiet Night Thought, with pinyin", func() { ruby.View(c, &a.views[0], a.poem, size) })
				section(c, "A Sunday note, with furigana", func() { ruby.View(c, &a.views[1], a.note, size) })
				section(c, "Markup", func() {
					if ui.TextArea(c, &a.markup).Font("monospace").Label("Markup").Height(t.Space(22)).Changed() {
						a.runs = ruby.Parse(a.markup)
					}
					ui.Text(c, "{word|reading} over a word, {word|r|r} over each character, 字(zì) after a Chinese character").
						FontSize(t.FontSize - 1).TextColor(t.TextMuted)
					ui.Column(c).Padding(t.Space(4)).Radius(t.Radius).Border(1, t.Border).Children(func() {
						ruby.View(c, &a.views[2], a.runs, size)
					})
				})
			})
		})
	})
}

// section builds a titled part of the window.
func section(c *ui.Context, title string, body func()) {
	t := c.Theme()
	ui.Column(c).Gap(t.Space(3)).Children(func() {
		ui.Text(c, title).FontSize(t.FontSize - 1).Bold().TextColor(t.TextMuted)
		body()
	})
}

func main() {
	a := &app{show: true, size: 24, poem: ruby.Parse(poem), note: ruby.Parse(note), markup: markup}
	a.runs = ruby.Parse(a.markup)
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Ruby",
			Width:   760,
			Height:  860,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
