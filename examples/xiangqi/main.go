// Xiangqi plays a game of Chinese chess between two people at one board:
// drag a piece or click it and then a point, and flip the board or take
// a move back on the right. The example knows the rules: the horse and
// the elephant blocked, the cannon's screen, the generals that may not
// face each other, and a side without a move losing.
//
//	go run ./examples/xiangqi
package main

import (
	"log"
	"strconv"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/xiangqi"
)

type pt = xiangqi.Point

type game struct {
	pos  xiangqi.Position
	red  bool
	last xiangqi.Move
}

func on(f, r int) bool { return f >= 0 && f <= 8 && r >= 0 && r <= 9 }

func palace(f, r int, red bool) bool {
	if red {
		return f >= 3 && f <= 5 && r >= 0 && r <= 2
	}
	return f >= 3 && f <= 5 && r >= 7 && r <= 9
}

// pseudo returns where the piece on from goes by how it moves, leaving
// its own general in check or not.
func (g *game) pseudo(from pt) []pt {
	pc := g.pos[from]
	red := pc.Red()
	f, r := from.File(), from.Rank()
	var out []pt
	add := func(nf, nr int) {
		if !on(nf, nr) {
			return
		}
		to := xiangqi.At(nf, nr)
		if q := g.pos[to]; q == 0 || q.Red() != red {
			out = append(out, to)
		}
	}
	empty := func(nf, nr int) bool { return on(nf, nr) && g.pos[xiangqi.At(nf, nr)] == 0 }
	straight := [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}
	switch pc | 0x20 {
	case 'k':
		for _, d := range straight {
			if palace(f+d[0], r+d[1], red) {
				add(f+d[0], r+d[1])
			}
		}
	case 'a':
		for _, d := range [][2]int{{1, 1}, {1, -1}, {-1, 1}, {-1, -1}} {
			if palace(f+d[0], r+d[1], red) {
				add(f+d[0], r+d[1])
			}
		}
	case 'b':
		for _, d := range [][2]int{{1, 1}, {1, -1}, {-1, 1}, {-1, -1}} {
			nr := r + 2*d[1]
			if (red && nr > 4) || (!red && nr < 5) || !empty(f+d[0], r+d[1]) {
				continue
			}
			add(f+2*d[0], nr)
		}
	case 'n':
		for _, d := range straight {
			if !empty(f+d[0], r+d[1]) {
				continue
			}
			if d[0] == 0 {
				add(f-1, r+2*d[1])
				add(f+1, r+2*d[1])
			} else {
				add(f+2*d[0], r-1)
				add(f+2*d[0], r+1)
			}
		}
	case 'r', 'c':
		cannon := pc|0x20 == 'c'
		for _, d := range straight {
			screened := false
			for n := 1; on(f+d[0]*n, r+d[1]*n); n++ {
				nf, nr := f+d[0]*n, r+d[1]*n
				q := g.pos[xiangqi.At(nf, nr)]
				switch {
				case !cannon:
					add(nf, nr)
				case !screened && q == 0:
					add(nf, nr)
				case screened && q != 0:
					if q.Red() != red {
						add(nf, nr)
					}
				}
				if q != 0 {
					if !cannon || screened {
						break
					}
					screened = true
				}
			}
		}
	case 'p':
		dir, crossed := 1, r >= 5
		if !red {
			dir, crossed = -1, r <= 4
		}
		add(f, r+dir)
		if crossed {
			add(f-1, r)
			add(f+1, r)
		}
	}
	return out
}

func (g *game) general(red bool) pt {
	k := xiangqi.Piece('k')
	if red {
		k = 'K'
	}
	for p := pt(0); p < 90; p++ {
		if g.pos[p] == k {
			return p
		}
	}
	return -1
}

// checked reports whether the side red's general is attacked, or faces
// the other general across an open file.
func (g *game) checked(red bool) bool {
	k, other := g.general(red), g.general(!red)
	if k < 0 || other < 0 {
		return true
	}
	if k.File() == other.File() {
		open := true
		lo, hi := min(k.Rank(), other.Rank()), max(k.Rank(), other.Rank())
		for r := lo + 1; r < hi; r++ {
			if g.pos[xiangqi.At(k.File(), r)] != 0 {
				open = false
				break
			}
		}
		if open {
			return true
		}
	}
	for p := pt(0); p < 90; p++ {
		if q := g.pos[p]; q != 0 && q.Red() != red {
			for _, to := range g.pseudo(p) {
				if to == k {
					return true
				}
			}
		}
	}
	return false
}

func (g game) play(m xiangqi.Move) game {
	g.pos[m.To], g.pos[m.From] = g.pos[m.From], 0
	g.red, g.last = !g.red, m
	return g
}

// legal returns where the piece on from may go on its side's turn.
func (g *game) legal(from pt) []pt {
	pc := g.pos[from]
	if pc == 0 || pc.Red() != g.red {
		return nil
	}
	var out []pt
	for _, to := range g.pseudo(from) {
		next := g.play(xiangqi.Move{From: from, To: to})
		if !next.checked(g.red) {
			out = append(out, to)
		}
	}
	return out
}

func (g *game) moves() bool {
	for p := pt(0); p < 90; p++ {
		if len(g.legal(p)) > 0 {
			return true
		}
	}
	return false
}

type app struct {
	history []game
	board   xiangqi.State
	flipped bool
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	g := &a.history[len(a.history)-1]
	check := g.checked(g.red)
	general := xiangqi.Piece(0)
	if check {
		general = 'k'
		if g.red {
			general = 'K'
		}
	}
	ui.Row(c).Fill().Background(t.Background).Padding(t.Space(6)).Gap(t.Space(6)).Children(func() {
		ev := xiangqi.View(c, &a.board, g.pos, xiangqi.Options{Flipped: a.flipped, Legal: g.legal, Last: g.last, Check: general})
		ev.Grow(1).FillHeight()
		if ev.Moved {
			a.history = append(a.history, g.play(ev.Move))
		}
		ui.Column(c).Width(t.Space(64)).FillHeight().Gap(t.Space(3)).Children(func() {
			side := "Black"
			if g.red {
				side = "Red"
			}
			status := side + " to move"
			switch {
			case !g.moves():
				status = side + " has no move and loses"
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
					a.history = []game{{pos: xiangqi.Start, red: true}}
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
	a := &app{history: []game{{pos: xiangqi.Start, red: true}}}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Xiangqi",
			Width:   900,
			Height:  720,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
