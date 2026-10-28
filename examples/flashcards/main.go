// Flashcards studies a deck of Chinese words: click a card or press Space
// to see its pinyin and meaning, then throw it right if you knew it or
// left to see it again a few cards later.
//
//	go run ./examples/flashcards
package main

import (
	"fmt"
	"log"
	"slices"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/flashcards"
)

type word struct {
	hanzi, pinyin, meaning string
}

var words = []word{
	{"猫", "māo", "cat"},
	{"学习", "xuéxí", "to study"},
	{"朋友", "péngyou", "friend"},
	{"电脑", "diànnǎo", "computer"},
	{"月亮", "yuèliang", "moon"},
	{"喜欢", "xǐhuan", "to like"},
	{"火车站", "huǒchēzhàn", "railway station"},
	{"图书馆", "túshūguǎn", "library"},
	{"天气", "tiānqì", "weather"},
	{"谢谢", "xièxie", "thank you"},
}

type app struct {
	deck  flashcards.State
	due   []int
	known int
}

func (a *app) restart() {
	a.due, a.known = a.due[:0], 0
	for i := range words {
		a.due = append(a.due, i)
	}
}

func (a *app) grade(known bool) {
	w := a.due[0]
	a.due = a.due[1:]
	if known {
		a.known++
		return
	}
	a.due = slices.Insert(a.due, min(3, len(a.due)), w)
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Surface).Padding(t.Space(10)).Gap(t.Space(6)).AlignItems(ui.Center).Children(func() {
		ui.Text(c, fmt.Sprintf("%d to go · %d known", len(a.due), a.known)).TextColor(t.TextMuted)
		if len(a.due) == 0 {
			ui.Column(c).Grow(1).Center().Gap(t.Space(4)).Children(func() {
				ui.Text(c, "All done for today").FontSize(t.FontSize + 10).Bold()
				if ui.PrimaryButton(c, "Study again").Clicked() {
					a.restart()
				}
			})
			return
		}
		ev := flashcards.Deck(c, &a.deck, len(a.due), func(i int, back bool) {
			w := words[a.due[i]]
			if !back {
				ui.Text(c, w.hanzi).FontSize(72).Bold()
				return
			}
			ui.Text(c, w.hanzi).FontSize(40).Bold()
			ui.Text(c, w.pinyin).FontSize(t.FontSize + 10).TextColor(t.Accent)
			ui.Text(c, w.meaning).FontSize(t.FontSize + 4).TextColor(t.TextMuted)
		})
		ev.Width(t.Space(90)).Grow(1).AutoFocus()
		if ev.Graded {
			a.grade(ev.Known)
		}
		ui.Row(c).Gap(t.Space(3)).Children(func() {
			if !a.deck.Turned() {
				if ui.Button(c, "Show answer").Clicked() {
					a.deck.Turn()
				}
				return
			}
			if ui.Button(c, "← Again").Clicked() {
				a.deck.Throw(false)
			}
			if ui.PrimaryButton(c, "Got it →").Clicked() {
				a.deck.Throw(true)
			}
		})
	})
}

func main() {
	a := &app{}
	a.restart()
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Flashcards",
			Width:   720,
			Height:  640,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
