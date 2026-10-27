// Chessboard plays a game of chess between two people at one board: drag
// a piece or click it and then a square, draw arrows with the right
// button, and flip the board or take a move back on the right. The
// example knows the rules, castling, en passant and promotion with them.
//
//	go run ./examples/chessboard
package main

import (
	"log"
	"slices"
	"strconv"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/chessboard"
)

type sq = chessboard.Square

// game is a position with whose move it is, the castlings left, the
// square a pawn may take en passant on, and the move that led to it.
type game struct {
	pos      chessboard.Position
	white    bool
	castling string
	ep       sq
	last     chessboard.Move
}

func newGame() game { return game{pos: chessboard.Start, white: true, castling: "KQkq", ep: -1} }

func (g *game) mine(p chessboard.Piece) bool { return p != 0 && p.White() == g.white }

// attacked reports whether a piece of the side white attacks s.
func (g *game) attacked(s sq, white bool) bool {
	f, r := s.File(), s.Rank()
	is := func(df, dr int, kinds string) bool {
		nf, nr := f+df, r+dr
		if nf < 0 || nf > 7 || nr < 0 || nr > 7 {
			return false
		}
		p := g.pos[chessboard.At(nf, nr)]
		return p != 0 && p.White() == white && strings.ContainsRune(kinds, rune(p|0x20))
	}
	pawn := -1
	if !white {
		pawn = 1
	}
	if is(-1, pawn, "p") || is(1, pawn, "p") {
		return true
	}
	for _, d := range [][2]int{{1, 2}, {2, 1}, {-1, 2}, {-2, 1}, {1, -2}, {2, -1}, {-1, -2}, {-2, -1}} {
		if is(d[0], d[1], "n") {
			return true
		}
	}
	for _, d := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}, {1, 1}, {1, -1}, {-1, 1}, {-1, -1}} {
		if is(d[0], d[1], "k") {
			return true
		}
		kinds := "rq"
		if d[0] != 0 && d[1] != 0 {
			kinds = "bq"
		}
		for n := 1; n < 8; n++ {
			nf, nr := f+d[0]*n, r+d[1]*n
			if nf < 0 || nf > 7 || nr < 0 || nr > 7 {
				break
			}
			if p := g.pos[chessboard.At(nf, nr)]; p != 0 {
				if p.White() == white && strings.ContainsRune(kinds, rune(p|0x20)) {
					return true
				}
				break
			}
		}
	}
	return false
}

func (g *game) king(white bool) sq {
	k := chessboard.Piece('k')
	if white {
		k = 'K'
	}
	return sq(slices.Index(g.pos[:], k))
}

// pseudo returns where the piece on s goes by how it moves, leaving its
// own king in check or not.
func (g *game) pseudo(s sq) []sq {
	p := g.pos[s]
	f, r := s.File(), s.Rank()
	var out []sq
	add := func(nf, nr int, take, quiet bool) bool {
		if nf < 0 || nf > 7 || nr < 0 || nr > 7 {
			return false
		}
		t := chessboard.At(nf, nr)
		q := g.pos[t]
		if q == 0 && quiet || q != 0 && q.White() != p.White() && take {
			out = append(out, t)
		}
		return q == 0
	}
	slide := func(dirs [][2]int, far int) {
		for _, d := range dirs {
			for n := 1; n <= far && add(f+d[0]*n, r+d[1]*n, true, true); n++ {
			}
		}
	}
	straight := [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}
	diagonal := [][2]int{{1, 1}, {1, -1}, {-1, 1}, {-1, -1}}
	switch p | 0x20 {
	case 'p':
		dir, home := 1, 1
		if !p.White() {
			dir, home = -1, 6
		}
		if add(f, r+dir, false, true) && r == home {
			add(f, r+2*dir, false, true)
		}
		for _, df := range []int{-1, 1} {
			add(f+df, r+dir, true, false)
			if f+df >= 0 && f+df <= 7 && chessboard.At(f+df, r+dir) == g.ep {
				out = append(out, g.ep)
			}
		}
	case 'n':
		for _, d := range [][2]int{{1, 2}, {2, 1}, {-1, 2}, {-2, 1}, {1, -2}, {2, -1}, {-1, -2}, {-2, -1}} {
			add(f+d[0], r+d[1], true, true)
		}
	case 'b':
		slide(diagonal, 7)
	case 'r':
		slide(straight, 7)
	case 'q':
		slide(append(straight, diagonal...), 7)
	case 'k':
		slide(append(straight, diagonal...), 1)
		rights, rank := "KQ", 0
		if !p.White() {
			rights, rank = "kq", 7
		}
		if s == chessboard.At(4, rank) && !g.attacked(s, !p.White()) {
			if strings.ContainsRune(g.castling, rune(rights[0])) && g.empty(rank, 5, 6) && !g.attacked(chessboard.At(5, rank), !p.White()) {
				out = append(out, chessboard.At(6, rank))
			}
			if strings.ContainsRune(g.castling, rune(rights[1])) && g.empty(rank, 1, 3) && !g.attacked(chessboard.At(3, rank), !p.White()) {
				out = append(out, chessboard.At(2, rank))
			}
		}
	}
	return out
}

func (g *game) empty(rank, from, to int) bool {
	for f := from; f <= to; f++ {
		if g.pos[chessboard.At(f, rank)] != 0 {
			return false
		}
	}
	return true
}

// legal returns where the piece on s may go on its side's turn.
func (g *game) legal(s sq) []sq {
	if !g.mine(g.pos[s]) {
		return nil
	}
	var out []sq
	for _, to := range g.pseudo(s) {
		next := g.play(chessboard.Move{From: s, To: to, Promotion: 'Q'})
		if !next.attacked(next.king(g.white), !g.white) {
			out = append(out, to)
		}
	}
	return out
}

// play returns the game after m, with castling's rook, en passant's pawn
// and promotion.
func (g game) play(m chessboard.Move) game {
	p := g.pos[m.From]
	g.pos[m.To], g.pos[m.From] = p, 0
	switch {
	case p|0x20 == 'p' && m.To == g.ep:
		g.pos[chessboard.At(m.To.File(), m.From.Rank())] = 0
	case p|0x20 == 'p' && (m.To.Rank() == 0 || m.To.Rank() == 7):
		promo := m.Promotion
		if promo == 0 {
			promo = 'q'
		}
		if p.White() {
			promo &^= 0x20
		} else {
			promo |= 0x20
		}
		g.pos[m.To] = promo
	case p|0x20 == 'k' && m.To.File()-m.From.File() == 2:
		g.pos[m.To-1], g.pos[m.To+1] = g.pos[m.To+1], 0
	case p|0x20 == 'k' && m.From.File()-m.To.File() == 2:
		g.pos[m.To+1], g.pos[m.To-2] = g.pos[m.To-2], 0
	}
	g.ep = -1
	if p|0x20 == 'p' && (m.To.Rank()-m.From.Rank() == 2 || m.From.Rank()-m.To.Rank() == 2) {
		g.ep = (m.From + m.To) / 2
	}
	for _, s := range []sq{m.From, m.To} {
		switch s {
		case chessboard.At(4, 0):
			g.castling = strings.NewReplacer("K", "", "Q", "").Replace(g.castling)
		case chessboard.At(4, 7):
			g.castling = strings.NewReplacer("k", "", "q", "").Replace(g.castling)
		case chessboard.At(7, 0):
			g.castling = strings.ReplaceAll(g.castling, "K", "")
		case chessboard.At(0, 0):
			g.castling = strings.ReplaceAll(g.castling, "Q", "")
		case chessboard.At(7, 7):
			g.castling = strings.ReplaceAll(g.castling, "k", "")
		case chessboard.At(0, 7):
			g.castling = strings.ReplaceAll(g.castling, "q", "")
		}
	}
	g.white = !g.white
	g.last = m
	return g
}

// status returns whether the side to move is in check and whether it has
// a move.
func (g *game) status() (check, moves bool) {
	check = g.attacked(g.king(g.white), !g.white)
	for s := sq(0); s < 64; s++ {
		if len(g.legal(s)) > 0 {
			return check, true
		}
	}
	return check, false
}

type app struct {
	history []game
	board   chessboard.State
	flipped bool
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	g := &a.history[len(a.history)-1]
	check, moves := g.status()
	king := chessboard.Piece(0)
	if check {
		king = 'k'
		if g.white {
			king = 'K'
		}
	}
	ui.Row(c).Fill().Background(t.Background).Padding(t.Space(6)).Gap(t.Space(6)).Children(func() {
		ev := chessboard.View(c, &a.board, g.pos, chessboard.Options{
			Flipped: a.flipped, Legal: g.legal, Last: g.last, Check: king,
		})
		ev.Grow(1).FillHeight()
		if ev.Moved {
			a.history = append(a.history, g.play(ev.Move))
		}
		ui.Column(c).Width(t.Space(64)).FillHeight().Gap(t.Space(3)).Children(func() {
			side := "Black"
			if g.white {
				side = "White"
			}
			status := side + " to move"
			switch {
			case check && !moves:
				status = "Checkmate"
			case !moves:
				status = "Stalemate"
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
					a.history = []game{newGame()}
				}
			})
			ui.Divider(c)
			ui.Scroll(c).Grow(1).Children(func() {
				ui.Grid(c).ColumnTracks(ui.Fixed(t.Space(8)), ui.Fr(1), ui.Fr(1)).GapY(t.Space(1)).Children(func() {
					for i := 1; i < len(a.history); i += 2 {
						ui.Text(c.Key(3*i), strconv.Itoa(i/2+1)+".").TextColor(t.TextMuted).FontFeatures("tnum")
						ui.Text(c.Key(3*i+1), a.history[i].last.String()).Font("monospace")
						black := ""
						if i+1 < len(a.history) {
							black = a.history[i+1].last.String()
						}
						ui.Text(c.Key(3*i+2), black).Font("monospace")
					}
				})
			})
			ui.Text(c, g.pos.FEN()).Font("monospace").FontSize(t.FontSize - 2).TextColor(t.TextMuted).Selectable()
		})
	})
}

func main() {
	a := &app{history: []game{newGame()}}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Chessboard",
			Width:   900,
			Height:  640,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
