// Mahjong deals a hand from a shuffled wall and draws a tile after each
// discard, the discards laid out above, or sets out a hand that waits on
// many tiles; below, every kind of tile.
//
//	go run ./examples/mahjong
package main

import (
	"fmt"
	"log"
	"math/rand/v2"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/mahjong"
)

type app struct {
	hand  mahjong.State
	deal  int
	wall  []mahjong.Tile
	tiles []mahjong.Tile
	melds []mahjong.Meld
	river []mahjong.Tile
}

func suit(s mahjong.Suit, ranks ...int) []mahjong.Tile {
	var out []mahjong.Tile
	for _, r := range ranks {
		out = append(out, mahjong.Tile{Suit: s, Rank: r})
	}
	return out
}

func (a *app) start() {
	a.river, a.melds = nil, nil
	switch a.deal {
	case 0:
		a.wall = mahjong.Set()
		rand.Shuffle(len(a.wall), func(i, j int) { a.wall[i], a.wall[j] = a.wall[j], a.wall[i] })
		a.tiles, a.wall = a.wall[:13:13], a.wall[13:]
		mahjong.Sort(a.tiles)
		a.draw()
	case 1:
		a.wall = nil
		a.tiles = suit(mahjong.Characters, 1, 1, 1, 2, 3, 4, 5, 6, 7, 8, 9, 9, 9)
	case 2:
		a.wall = nil
		a.tiles = append(suit(mahjong.Bamboo, 2, 3, 4, 6, 6, 6, 7), mahjong.Red, mahjong.Red, mahjong.Red)
		a.melds = []mahjong.Meld{
			{Tiles: suit(mahjong.Dots, 5, 5, 5, 5), Concealed: true},
		}
	}
}

func (a *app) draw() {
	if len(a.wall) > 0 {
		a.tiles, a.wall = append(a.tiles, a.wall[0]), a.wall[1:]
	}
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Padding(t.Space(8)).Gap(t.Space(6)).Background(t.Background).Children(func() {
		ui.Row(c).Gap(t.Space(4)).AlignItems(ui.Center).Children(func() {
			if ui.Segmented(c, &a.deal, "Deal", "Nine gates", "With a kong").Label("Hand").Changed() {
				a.start()
			}
			ui.Box(c).Grow(1)
			if a.deal == 0 {
				ui.Text(c, fmt.Sprintf("%d tiles in the wall", len(a.wall))).TextColor(t.TextMuted)
				if ui.Button(c, "Deal again").Clicked() {
					a.start()
				}
			}
		})
		ui.Row(c).Wrap().Gap(t.Space(0.5)).Height(t.Space(30)).AlignContent(ui.Start).Label("Discards").Children(func() {
			w, h := mahjong.Size(c)
			for i, tile := range a.river {
				mahjong.Face(c.Key(i), tile, true).Size(w*0.7, h*0.7)
			}
		})
		ev := mahjong.Hand(c, &a.hand, &a.tiles, a.melds)
		ev.AutoFocus()
		if ev.Discarded != nil {
			a.river = append(a.river, *ev.Discarded)
			mahjong.Sort(a.tiles)
			a.draw()
		}
		ui.Divider(c)
		ui.Row(c).Wrap().Gap(t.Space(1)).Label("Tiles").Children(func() {
			for i, tile := range mahjong.Set() {
				if i%4 == 0 {
					mahjong.Face(c.Key(i), tile, true)
				}
			}
			mahjong.Face(c.Key("back"), mahjong.East, false)
		})
	})
}

func main() {
	a := &app{}
	a.start()
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Mahjong",
			Width:   880,
			Height:  640,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
