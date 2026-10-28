// Dice rolls a tray of dice: five six-sided ones to hold and roll again,
// two for a board game, or one of each shape.
//
//	go run ./examples/dice
package main

import (
	"log"
	"strconv"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/dice"
)

var sets = [][]int{
	{6, 6, 6, 6, 6},
	{6, 6},
	{4, 6, 8, 10, 12, 20},
}

type app struct {
	tray  dice.State
	set   int
	dice  []dice.Die
	rolls []int
}

func (a *app) pick() {
	a.dice, a.rolls = nil, nil
	for _, sides := range sets[a.set] {
		a.dice = append(a.dice, dice.Die{Sides: sides, Value: sides})
	}
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Padding(t.Space(8)).Gap(t.Space(10)).AlignItems(ui.Center).Background(t.Background).Children(func() {
		if ui.Segmented(c, &a.set, "Five dice", "Two dice", "One of each").Label("Dice").Changed() {
			a.pick()
		}
		ev := dice.View(c, &a.tray, &a.dice)
		if ev.Rolled {
			total := 0
			for _, d := range a.dice {
				total += d.Value
			}
			a.rolls = append([]int{total}, a.rolls[:min(len(a.rolls), 7)]...)
		}
		if len(a.rolls) > 1 {
			past := make([]string, len(a.rolls)-1)
			for i, r := range a.rolls[1:] {
				past[i] = strconv.Itoa(r)
			}
			ui.Text(c, "Before: "+strings.Join(past, ", ")).TextColor(t.TextMuted).FontFeatures("tnum")
		}
	})
}

func main() {
	a := &app{}
	a.pick()
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Dice",
			Width:   720,
			Height:  480,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
