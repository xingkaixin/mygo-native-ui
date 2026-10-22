// Staff engraves the ABC notation beside it as it is typed, and plays
// it: Greensleeves in A minor with its repeat, Twinkle, Twinkle with a
// variation of triplets, ties and sixteenths, and Frère Jacques in the
// bass clef. Clicking a note shows where it is in the source.
//
//	go run ./examples/staff
package main

import (
	"fmt"
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/staff"
)

var tunes = []struct{ name, abc string }{
	{"Greensleeves", `X:1
T:Greensleeves
C:Traditional
M:6/8
L:1/8
Q:3/8=60
K:Am
A |: c2 d e>fe | d2 B G>AB | c2 A A>^GA | B2 ^G E2 A |
c2 d e>fe | d2 B G>AB | c>BA ^G>^FG | A3 A2 :|
g3 g>fe | d2 B G>AB | c2 A A>^GA | B2 ^G E3 |
g3 g>fe | d2 B G>AB | c>BA ^G>^FG | [A,EA]6 |]`},
	{"Twinkle, Twinkle", `X:2
T:Twinkle, Twinkle, Little Star
C:Traditional, with a variation
M:C
L:1/8
Q:1/4=96
K:C
C2 C2 G2 G2 | A2 A2 G4 | F2 F2 E2 E2 | D2 D2 C4 |
G2 G2 F2 F2 | E2 E2 D4 | G2 G2 F2 F2 | E2 E2 D4 |
% the variation
C/D/C/B,/ C2 G/A/G/F/ G2 | (3AcA (3AcA G2- G2 |
F/G/F/E/ F2 E/F/E/D/ E2 | D>E D>B, [CEGc]4 |]`},
	{"Frère Jacques", `X:3
T:Frère Jacques
C:Traditional round
M:4/4
L:1/4
Q:1/4=112
K:F clef=bass
F, G, A, F, | F, G, A, F, | A, B, C2 | A, B, C2 |
C/D/C/B,/ A, F, | C/D/C/B,/ A, F, | F, C, F, z | [F,,C,F,]4 |]`},
}

var values = map[int]string{1: "whole", 2: "half", 4: "quarter", 8: "eighth", 16: "sixteenth", 32: "32nd", 64: "64th"}

// length names n/d of a whole note, as dotted quarter.
func length(n, d int) string {
	switch {
	case n == 1 && values[d] != "":
		return values[d]
	case n == 3 && values[d/2] != "":
		return "dotted " + values[d/2]
	case n == 1 && d%3 == 0 && values[d/3*2] != "":
		return values[d/3*2] + " of a triplet"
	}
	return fmt.Sprintf("%d/%d", n, d)
}

type app struct {
	tune   int
	source string
	split  float32
	score  staff.State
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Children(func() {
		ui.Row(c).Padding(t.Space(3), t.Space(4)).Children(func() {
			names := make([]string, len(tunes))
			for i, tn := range tunes {
				names[i] = tn.name
			}
			if ui.Segmented(c, &a.tune, names...).Label("Tune").Changed() {
				a.source = tunes[a.tune].abc
			}
		})
		ui.Divider(c)
		ui.Split(c, &a.split, func() {
			ui.Column(c).Fill().Padding(t.Space(3)).Children(func() {
				ui.TextArea(c, &a.source).Font("monospace").Label("ABC").Grow(1)
			})
		}, func() {
			ui.Column(c).Fill().Children(func() {
				staff.View(c, &a.score, a.source).Grow(1)
				ui.Divider(c)
				ui.Row(c).Gap(t.Space(3)).Padding(t.Space(3), t.Space(4)).AlignItems(ui.Center).Children(func() {
					n, ok := a.score.Chosen()
					if !ok {
						ui.Text(c, "Click a note to choose it").TextColor(t.TextMuted)
						return
					}
					ui.Text(c, n.Name).Bold()
					ui.Text(c, length(n.Num, n.Den))
					ui.Text(c, fmt.Sprintf("Line %d, column %d", n.Line, n.Column)).TextColor(t.TextMuted)
				})
			})
		}).Grow(1)
	})
}

func main() {
	a := &app{source: tunes[0].abc, split: 520}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Staff",
			Width:   1400,
			Height:  900,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
