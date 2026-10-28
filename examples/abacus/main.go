// Abacus sets a sum to work on a soroban: add the numbers one by one by
// moving the beads, and the sum is checked as the beads move.
//
//	go run ./examples/abacus
package main

import (
	"log"
	"strconv"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/abacus"
)

var (
	addends = []int64{386, 4_207, 19_055, 642, 73_918}
	total   int64
)

func sum() int64 {
	var s int64
	for _, a := range addends {
		s += a
	}
	return s
}

// grouped writes n with thousands separated, as the abacus reads.
func grouped(n int64) string {
	s := strconv.FormatInt(n, 10)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}

func view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(6)).Gap(t.Space(6)).Children(func() {
		ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "Add on the abacus").Bold().FontSize(t.FontSize + 4).Grow(1)
			if ui.Button(c, "Clear").Clicked() {
				total = 0
			}
		})
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Baseline).Children(func() {
			for i, a := range addends {
				if i > 0 {
					ui.Text(c.Key(-1-i), "+").TextColor(t.TextMuted)
				}
				ui.Text(c.Key(i), grouped(a)).Font("monospace").FontSize(t.FontSize + 2)
			}
		})
		abacus.View(c, &total, 9)
		if total == sum() {
			ui.Text(c, "That is the sum.").TextColor(t.Success).FontWeight(600)
		}
	})
}

func main() {
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Abacus",
			Width:   640,
			Height:  560,
			Content: ui.View(view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
