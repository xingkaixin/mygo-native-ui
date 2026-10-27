// Goban steps through a short game on a 9×9 board read from SGF, and
// plays on from any move: click a point to play the stone whose turn it
// is, pass, or start an empty board of another size. Captures count on
// the right, and ko is checked.
//
//	go run ./examples/goban
package main

import (
	"fmt"
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/goban"
)

// game is a made-up game between two club players, with a variation
// that the main line leaves out.
const game = `(;GM[1]FF[4]SZ[9]PB[Mei]PW[Ren]
;B[ee];W[gc];B[cg];W[cc];B[gg];W[dc];B[eg];W[hf];B[hg];W[fe]
;B[ef];W[ge];B[ed];W[ec];B[fc];W[fb];B[fd];W[gd];B[gb];W[hb]
(;B[gf];W[db];B[if];W[cd];B[he];W[eb];B[cb];W[bc])
(;B[db];W[eb]))`

// position is the board after a move, with whose turn it is, the stones
// each side took, and the stone played.
type position struct {
	board goban.Board
	turn  goban.Stone
	taken [3]int
	last  *goban.Point
	move  string
}

type app struct {
	history []position
	at      int
	board   goban.State
	size    int
	message string
}

func (a *app) load(sgf string) {
	g, err := goban.ParseSGF(sgf)
	if err != nil {
		log.Fatal(err)
	}
	a.history = []position{{board: g.Setup, turn: goban.Black}}
	for _, m := range g.Moves {
		a.play(m.Point, m.Pass)
	}
	a.at, a.size = 0, g.Size
}

// play plays the stone whose turn it is at p, or passes, after the
// position shown, and reports why it may not.
func (a *app) play(p goban.Point, pass bool) string {
	cur := a.history[a.at]
	next := position{board: cur.board, turn: cur.turn.Other(), taken: cur.taken, move: "Pass"}
	if !pass {
		board, taken, ok := cur.board.Play(p, cur.turn)
		if !ok {
			return "A stone there would have no liberty"
		}
		if a.at > 0 && board.Equal(a.history[a.at-1].board) {
			return "Ko: take back elsewhere first"
		}
		next.board, next.last, next.move = board, &p, p.Name(board.Size)
		next.taken[cur.turn] += taken
	}
	a.history = append(a.history[:a.at+1], next)
	a.at++
	return ""
}

func name(s goban.Stone) string {
	if s == goban.Black {
		return "Black"
	}
	return "White"
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	cur := a.history[a.at]
	ui.Row(c).Fill().Background(t.Background).Padding(t.Space(6)).Gap(t.Space(6)).Children(func() {
		ev := goban.View(c, &a.board, cur.board, goban.Options{Turn: cur.turn, Last: cur.last})
		ev.Grow(1).FillHeight()
		if ev.Played {
			a.message = a.play(ev.Point, false)
		}
		ui.Column(c).Width(t.Space(72)).FillHeight().Gap(t.Space(3)).Children(func() {
			ui.Text(c, name(cur.turn)+" to play").FontSize(t.FontSize + 4).Bold()
			ui.Text(c, fmt.Sprintf("Black took %d · White took %d", cur.taken[goban.Black], cur.taken[goban.White])).TextColor(t.TextMuted)
			ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
				if ui.Button(c, "‹").Disabled(a.at == 0).Label("Previous move").Clicked() {
					a.at--
				}
				ui.Text(c, fmt.Sprintf("Move %d of %d", a.at, len(a.history)-1)).FontFeatures("tnum").Grow(1).TextAlign(ui.Center)
				if ui.Button(c, "›").Disabled(a.at == len(a.history)-1).Label("Next move").Clicked() {
					a.at++
				}
			})
			ui.Row(c).Gap(t.Space(2)).Children(func() {
				if ui.Button(c, "Pass").Clicked() {
					a.message = a.play(goban.Point{}, true)
				}
				for _, n := range []int{9, 13, 19} {
					if ui.Button(c.Key(n), fmt.Sprintf("%d×%d", n, n)).Clicked() {
						a.history, a.at, a.size, a.message = []position{{board: goban.NewBoard(n), turn: goban.Black}}, 0, n, ""
					}
				}
			})
			if a.message != "" {
				ui.Text(c, a.message).TextColor(t.Danger)
			}
			ui.Divider(c)
			ui.Scroll(c).Grow(1).Children(func() {
				ui.Column(c).Gap(t.Space(0.5)).Children(func() {
					for i := 1; i < len(a.history); i++ {
						row := ui.Row(c.Key(i)).Gap(t.Space(3)).Padding(t.Space(0.5), t.Space(2)).Radius(t.Space(1)).Cursor(ui.CursorPointer)
						if i == a.at {
							row.Background(t.Selection)
						}
						if row.Clicked() {
							a.at = i
						}
						row.Children(func() {
							ui.Text(c, fmt.Sprint(i)).Width(t.Space(8)).TextColor(t.TextMuted).FontFeatures("tnum")
							ui.Text(c, name(a.history[i-1].turn)).Width(t.Space(14))
							ui.Text(c, a.history[i].move).FontFeatures("tnum")
						})
					}
				})
			})
		})
	})
}

func main() {
	a := &app{}
	a.load(game)
	a.at = len(a.history) - 1
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Goban",
			Width:   920,
			Height:  660,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
