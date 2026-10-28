// Tactics sketches a team playing out from the back against a high
// 4-4-2 press: the keeper finds the left back, who goes long down the
// line for the winger running in behind. Drag the players and the ball,
// draw runs and passes on the grass, or clear the arrows and start over.
//
//	go run ./examples/tactics
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/tactics"
)

func home(n int, name string, x, y float64) tactics.Player {
	return tactics.Player{Number: n, Name: name, X: x, Y: y}
}

func away(n int, name string, x, y float64) tactics.Player {
	return tactics.Player{Number: n, Name: name, Away: true, X: x, Y: y}
}

func start() tactics.Board {
	return tactics.Board{
		Players: []tactics.Player{
			home(1, "Alves", 4, 34),
			home(2, "Reyes", 22, 60),
			home(4, "Costa", 14, 44),
			home(5, "Brandt", 14, 24),
			home(3, "Okafor", 22, 8),
			home(6, "Moretti", 26, 34),
			home(8, "Lindqvist", 38, 46),
			home(10, "Haddad", 38, 22),
			home(7, "Silva", 50, 62),
			home(9, "Novak", 54, 34),
			home(11, "Mbeki", 50, 6),
			away(1, "Petrov", 100, 34),
			away(2, "Ito", 70, 8),
			away(5, "Dupont", 76, 26),
			away(4, "Weber", 76, 42),
			away(3, "Santos", 70, 60),
			away(7, "Ruiz", 46, 10),
			away(8, "Mensah", 42, 28),
			away(6, "Larsen", 42, 42),
			away(11, "Duarte", 46, 58),
			away(9, "Fofana", 20, 30),
			away(10, "Kim", 20, 42),
		},
		Ball: [2]float64{5.5, 34},
		Arrows: []tactics.Arrow{
			{From: [2]float64{5.5, 34}, To: [2]float64{21, 9.5}, Pass: true},
			{From: [2]float64{22, 8}, To: [2]float64{70, 13}, Pass: true},
			{From: [2]float64{50, 6}, To: [2]float64{72, 15}},
			{From: [2]float64{54, 34}, To: [2]float64{80, 30}},
		},
	}
}

var (
	board = start()
	state tactics.State
)

func view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(6)).Gap(t.Space(4)).Children(func() {
		ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "Playing out against a 4-4-2 press").Bold().FontSize(t.FontSize + 4).Grow(1)
			if ui.Button(c, "Clear Arrows").Clicked() {
				board.Arrows = nil
			}
			if ui.Button(c, "Reset").Clicked() {
				board, state = start(), tactics.State{}
			}
		})
		tactics.View(c, &state, &board).Grow(1).Radius(t.Radius)
		ui.Text(c, "Drag on the grass for a run, with Shift for a pass. Double-click an arrow to remove it.").
			FontSize(t.FontSize - 1).TextColor(t.TextMuted)
	})
}

func main() {
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Tactics",
			Width:   1040,
			Height:  760,
			Content: ui.View(view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
