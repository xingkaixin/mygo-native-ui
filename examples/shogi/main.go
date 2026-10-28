// Shogi plays a game of Japanese chess between two people at one board:
// drag a piece or click it and then a square, drop a captured piece from
// the hand the same way, and flip the board or take a move back on the
// right. The example knows how each piece moves, the squares a piece may
// not drop on, two pawns on a file, and a king left in check; it leaves
// out the rule against mating by a pawn drop.
//
//	go run ./examples/shogi
package main

import (
	"log"
	"strconv"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/shogi"
)

type sq = shogi.Square

type game struct {
	pos  shogi.Position
	last shogi.Move
}

func on(f, r int) bool { return f >= 1 && f <= 9 && r >= 1 && r <= 9 }

var (
	king   = [][2]int{{-1, -1}, {0, -1}, {1, -1}, {-1, 0}, {1, 0}, {-1, 1}, {0, 1}, {1, 1}}
	gold   = [][2]int{{-1, -1}, {0, -1}, {1, -1}, {-1, 0}, {1, 0}, {0, 1}}
	silver = [][2]int{{-1, -1}, {0, -1}, {1, -1}, {-1, 1}, {1, 1}}
	rook   = [][2]int{{0, -1}, {0, 1}, {-1, 0}, {1, 0}}
	bishop = [][2]int{{-1, -1}, {1, -1}, {-1, 1}, {1, 1}}
)

// pseudo returns where the piece on from goes by how it moves, leaving
// its own king in check or not. Steps are written for sente, forward
// being toward rank 1.
func (g *game) pseudo(from sq) []sq {
	pc := g.pos.Board[from]
	f, r := from.File(), from.Rank()
	dir := 1
	if pc.Gote {
		dir = -1
	}
	var out []sq
	add := func(nf, nr int) bool {
		if !on(nf, nr) {
			return false
		}
		to := shogi.At(nf, nr)
		q := g.pos.Board[to]
		if q.Kind == 0 || q.Gote != pc.Gote {
			out = append(out, to)
		}
		return q.Kind == 0
	}
	steps := func(ds [][2]int) {
		for _, d := range ds {
			add(f+d[0]*dir, r+d[1]*dir)
		}
	}
	slides := func(ds [][2]int) {
		for _, d := range ds {
			for n := 1; add(f+d[0]*n*dir, r+d[1]*n*dir); n++ {
			}
		}
	}
	switch {
	case pc.Kind == 'K':
		steps(king)
	case pc.Kind == 'G', pc.Promoted && (pc.Kind == 'S' || pc.Kind == 'N' || pc.Kind == 'L' || pc.Kind == 'P'):
		steps(gold)
	case pc.Kind == 'S':
		steps(silver)
	case pc.Kind == 'N':
		steps([][2]int{{-1, -2}, {1, -2}})
	case pc.Kind == 'L':
		slides([][2]int{{0, -1}})
	case pc.Kind == 'P':
		steps([][2]int{{0, -1}})
	case pc.Kind == 'R':
		slides(rook)
		if pc.Promoted {
			steps(bishop)
		}
	case pc.Kind == 'B':
		slides(bishop)
		if pc.Promoted {
			steps(rook)
		}
	}
	return out
}

func (g *game) checked(gote bool) bool {
	k := sq(-1)
	for s, pc := range g.pos.Board {
		if pc.Kind == 'K' && pc.Gote == gote {
			k = sq(s)
		}
	}
	for s, pc := range g.pos.Board {
		if pc.Kind == 0 || pc.Gote == gote {
			continue
		}
		for _, to := range g.pseudo(sq(s)) {
			if to == k {
				return true
			}
		}
	}
	return false
}

func (g game) play(m shogi.Move) game {
	g.pos.Play(m)
	g.last = m
	return g
}

// far returns the rank of to counted from the far side for the side to
// move.
func (g *game) far(to sq) int {
	if g.pos.Gote {
		return 10 - to.Rank()
	}
	return to.Rank()
}

func (g *game) legal(from sq, drop byte) []sq {
	var cands []shogi.Move
	if drop != 0 {
		for to := sq(0); to < 81; to++ {
			if g.pos.Board[to].Kind != 0 ||
				(drop == 'P' || drop == 'L') && g.far(to) == 1 || drop == 'N' && g.far(to) <= 2 {
				continue
			}
			if drop == 'P' && g.pawnOn(to.File()) {
				continue
			}
			cands = append(cands, shogi.Move{To: to, Drop: drop})
		}
	} else {
		pc := g.pos.Board[from]
		if pc.Kind == 0 || pc.Gote != g.pos.Gote {
			return nil
		}
		for _, to := range g.pseudo(from) {
			cands = append(cands, shogi.Move{From: from, To: to})
		}
	}
	var out []sq
	for _, m := range cands {
		if next := g.play(m); !next.checked(g.pos.Gote) {
			out = append(out, m.To)
		}
	}
	return out
}

func (g *game) pawnOn(file int) bool {
	for r := 1; r <= 9; r++ {
		if pc := g.pos.Board[shogi.At(file, r)]; pc.Kind == 'P' && !pc.Promoted && pc.Gote == g.pos.Gote {
			return true
		}
	}
	return false
}

func (g *game) moves() bool {
	for s := sq(0); s < 81; s++ {
		if len(g.legal(s, 0)) > 0 {
			return true
		}
	}
	for _, k := range shogi.Kinds[1:] {
		if len(g.legal(0, byte(k))) > 0 {
			return true
		}
	}
	return false
}

type app struct {
	history []game
	board   shogi.State
	flipped bool
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	g := &a.history[len(a.history)-1]
	check := g.checked(g.pos.Gote)
	ui.Row(c).Fill().Background(t.Background).Padding(t.Space(6)).Gap(t.Space(6)).Children(func() {
		ev := shogi.View(c, &a.board, g.pos, shogi.Options{Flipped: a.flipped, Legal: g.legal, Last: g.last, Check: check})
		ev.Grow(1).FillHeight()
		if ev.Moved {
			a.history = append(a.history, g.play(ev.Move))
		}
		ui.Column(c).Width(t.Space(64)).FillHeight().Gap(t.Space(3)).Children(func() {
			side := "Sente"
			if g.pos.Gote {
				side = "Gote"
			}
			status := side + " to move"
			switch {
			case !g.moves():
				status = side + " is mated"
			case check:
				status = side + " is in check"
			}
			ui.Text(c, status).FontSize(t.FontSize + 4).Bold()
			ui.Row(c).Gap(t.Space(2)).Children(func() {
				if ui.Button(c, "Flip").Clicked() {
					a.flipped = !a.flipped
				}
				if ui.Button(c, "Take back").Disabled(len(a.history) == 1).Clicked() {
					a.history = a.history[:len(a.history)-1]
				}
				if ui.Button(c, "New game").Clicked() {
					a.history = []game{{pos: shogi.Start}}
				}
			})
			ui.Divider(c)
			ui.Scroll(c).Grow(1).Children(func() {
				ui.Grid(c).ColumnTracks(ui.Fixed(t.Space(8)), ui.Fr(1)).GapY(t.Space(1)).Children(func() {
					for i := 1; i < len(a.history); i++ {
						mark := "☗"
						if i%2 == 0 {
							mark = "☖"
						}
						ui.Text(c.Key(2*i), strconv.Itoa(i)+".").TextColor(t.TextMuted).FontFeatures("tnum")
						ui.Text(c.Key(2*i+1), mark+" "+a.history[i].last.String()).Font("monospace")
					}
				})
			})
			ui.Text(c, g.pos.SFEN()).Font("monospace").FontSize(t.FontSize - 2).TextColor(t.TextMuted).Selectable()
		})
	})
}

func main() {
	a := &app{history: []game{{pos: shogi.Start}}}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Shogi",
			Width:   960,
			Height:  760,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
