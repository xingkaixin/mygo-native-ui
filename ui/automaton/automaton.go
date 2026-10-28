// Package automaton runs a cellular automaton of the Game of Life's kind
// on a board that wraps around at its edges, by any rule written as
// B3/S23: the counts of live neighbors at which a dead cell is born and a
// live one survives. Live cells brighten when born and deepen as they
// age, and dead ones leave a fading ghost. Pressing a cell and dragging
// paints cells alive, or clears them when the first was alive.
//
//	board := automaton.NewBoard(120, 80)
//	board.Place(automaton.Glider, 4, 4)
//	automaton.View(c, &app.life, board, automaton.Life, 15)
package automaton

import (
	"errors"
	"math"
	"math/rand/v2"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
)

// Rule is a rule of Life-like automata: for each count of live
// neighbors, whether a dead cell is born and a live one survives.
type Rule struct {
	Born, Survive [9]bool
}

// ParseRule reads a rule written as B3/S23.
func ParseRule(s string) (Rule, error) {
	var r Rule
	parts := strings.Split(strings.ToUpper(strings.TrimSpace(s)), "/")
	if len(parts) != 2 || !strings.HasPrefix(parts[0], "B") || !strings.HasPrefix(parts[1], "S") {
		return r, errors.New("automaton: rule is not written as B3/S23")
	}
	for i, set := range []*[9]bool{&r.Born, &r.Survive} {
		for _, ch := range parts[i][1:] {
			if ch < '0' || ch > '8' {
				return r, errors.New("automaton: neighbor counts go from 0 to 8")
			}
			set[ch-'0'] = true
		}
	}
	return r, nil
}

func (r Rule) String() string {
	var b strings.Builder
	b.WriteString("B")
	for n, on := range r.Born {
		if on {
			b.WriteByte(byte('0' + n))
		}
	}
	b.WriteString("/S")
	for n, on := range r.Survive {
		if on {
			b.WriteByte(byte('0' + n))
		}
	}
	return b.String()
}

func must(s string) Rule {
	r, err := ParseRule(s)
	if err != nil {
		panic(err)
	}
	return r
}

// Some well-known rules.
var (
	Life     = must("B3/S23")
	HighLife = must("B36/S23")
	Seeds    = must("B2/S")
	DayNight = must("B3678/S34678")
	Maze     = must("B3/S12345")
)

// Some well-known patterns, as Place takes them.
const (
	Glider = ".O.\n..O\nOOO"
	RPento = ".OO\nOO.\n.O."
	Gosper = "" +
		"........................O\n" +
		"......................O.O\n" +
		"............OO......OO............OO\n" +
		"...........O...O....OO............OO\n" +
		"OO........O.....O...OO\n" +
		"OO........O...O.OO....O.O\n" +
		"..........O.....O.......O\n" +
		"...........O...O\n" +
		"............OO"
)

// Board is the cells of an automaton. Make one with NewBoard and keep it
// in the app's state.
type Board struct {
	cols, rows int
	gen        int
	// age is how many generations each cell has lived, 0 when dead, and
	// ghost how recently each died.
	age, next []uint16
	ghost     []uint8
}

// NewBoard returns an empty board of cols by rows cells.
func NewBoard(cols, rows int) *Board {
	n := cols * rows
	return &Board{cols: cols, rows: rows, age: make([]uint16, n), next: make([]uint16, n), ghost: make([]uint8, n)}
}

// Size returns how many columns and rows the board has.
func (b *Board) Size() (cols, rows int) { return b.cols, b.rows }

// Generation returns how many steps the board has taken.
func (b *Board) Generation() int { return b.gen }

// Population returns how many cells are alive.
func (b *Board) Population() int {
	n := 0
	for _, a := range b.age {
		if a > 0 {
			n++
		}
	}
	return n
}

// Alive reports whether the cell at col, row is alive.
func (b *Board) Alive(col, row int) bool { return b.age[b.at(col, row)] > 0 }

// Set makes the cell at col, row alive or dead.
func (b *Board) Set(col, row int, alive bool) {
	i := b.at(col, row)
	switch {
	case alive && b.age[i] == 0:
		b.age[i] = 1
	case !alive && b.age[i] > 0:
		b.age[i], b.ghost[i] = 0, ghostLife
	}
}

func (b *Board) at(col, row int) int {
	col, row = (col%b.cols+b.cols)%b.cols, (row%b.rows+b.rows)%b.rows
	return row*b.cols + col
}

// Place puts a pattern on the board with its top left at col, row: rows
// of O for live cells and . for dead ones, one line each.
func (b *Board) Place(pattern string, col, row int) {
	for y, line := range strings.Split(pattern, "\n") {
		for x, ch := range line {
			b.Set(col+x, row+y, ch == 'O' || ch == '*')
		}
	}
}

// Clear kills every cell and starts counting generations again.
func (b *Board) Clear() {
	clear(b.age)
	clear(b.ghost)
	b.gen = 0
}

// Scatter makes each cell alive with the chance density.
func (b *Board) Scatter(density float64) {
	for i := range b.age {
		b.age[i] = 0
		if rand.Float64() < density {
			b.age[i] = 1
		}
	}
	clear(b.ghost)
	b.gen = 0
}

const ghostLife = 10

// Step moves the board on a generation by rule.
func (b *Board) Step(rule Rule) {
	for y := range b.rows {
		up, down := (y+b.rows-1)%b.rows*b.cols, (y+1)%b.rows*b.cols
		row := y * b.cols
		for x := range b.cols {
			l, r := (x+b.cols-1)%b.cols, (x+1)%b.cols
			n := 0
			for _, i := range [8]int{up + l, up + x, up + r, row + l, row + r, down + l, down + x, down + r} {
				if b.age[i] > 0 {
					n++
				}
			}
			i := row + x
			switch a := b.age[i]; {
			case a > 0 && rule.Survive[n]:
				b.next[i] = min(a, 999) + 1
			case a == 0 && rule.Born[n]:
				b.next[i] = 1
			default:
				b.next[i] = 0
				if a > 0 {
					b.ghost[i] = ghostLife
				} else if b.ghost[i] > 0 {
					b.ghost[i]--
				}
			}
		}
	}
	b.age, b.next = b.next, b.age
	b.gen++
}

// State is what a board's view keeps: when it last stepped and what
// a drag paints. Keep it in the app's state, one for each view.
type State struct {
	last     time.Time
	painting bool
	paint    bool
	px, py   int
}

// View creates the board b, stepping it by rule rate generations a
// second; a rate of 0 pauses it.
func View(c *ui.Context, s *State, b *Board, rule Rule, rate float64) ui.Element {
	t := c.Theme()
	view := ui.Box(c).Clip().Cursor(ui.CursorCrosshair).Role(ui.RoleImage).Label("Cellular automaton")
	r := view.LayoutBounds()
	cell := min(r.W/float32(b.cols), r.H/float32(b.rows))
	ox, oy := (r.W-cell*float32(b.cols))/2, (r.H-cell*float32(b.rows))/2
	cellAt := func(x, y float32) (int, int, bool) {
		if cell <= 0 {
			return 0, 0, false
		}
		col, row := int(math.Floor(float64((x-ox)/cell))), int(math.Floor(float64((y-oy)/cell)))
		return col, row, col >= 0 && row >= 0 && col < b.cols && row < b.rows
	}
	view.HandleInput(func(in ui.InputEvent) bool {
		switch in.Kind {
		case ui.InputPointerDown:
			col, row, ok := cellAt(in.X, in.Y)
			if !ok {
				return false
			}
			s.painting, s.paint, s.px, s.py = true, !b.Alive(col, row), col, row
			b.Set(col, row, s.paint)
		case ui.InputPointerMove:
			if !s.painting {
				return false
			}
			col, row, _ := cellAt(in.X, in.Y)
			col, row = max(0, min(b.cols-1, col)), max(0, min(b.rows-1, row))
			// The pointer skips cells when dragged fast; fill in the line
			// between where it was and where it is.
			steps := max(abs(col-s.px), abs(row-s.py))
			for k := 1; k <= steps; k++ {
				b.Set(s.px+(col-s.px)*k/steps, s.py+(row-s.py)*k/steps, s.paint)
			}
			s.px, s.py = col, row
		case ui.InputPointerUp:
			s.painting = false
		default:
			return false
		}
		return true
	})
	now := c.Now()
	if rate > 0 && !s.painting {
		every := time.Duration(float64(time.Second) / rate)
		if s.last.IsZero() || now.Sub(s.last) > 4*every {
			s.last = now
		}
		for now.Sub(s.last) >= every {
			b.Step(rule)
			s.last = s.last.Add(every)
		}
		c.After(every - now.Sub(s.last))
	} else {
		s.last = time.Time{}
	}
	view.Draw(func(p *ui.Painter, rc ui.Rect) {
		x0, y0 := rc.X+ox, rc.Y+oy
		board := ui.Rect{X: x0, Y: y0, W: cell * float32(b.cols), H: cell * float32(b.rows)}
		p.Fill(board, t.Surface, 0)
		gap := float32(0)
		if cell >= 6 {
			gap = 1
		}
		for i, a := range b.age {
			x, y := x0+float32(i%b.cols)*cell, y0+float32(i/b.cols)*cell
			box := ui.Rect{X: x + gap/2, Y: y + gap/2, W: cell - gap, H: cell - gap}
			switch {
			case a > 0:
				k := float32(min(a-1, 20)) / 20
				p.Fill(box, ui.Oklch(0.82-0.3*k, 0.11+0.05*k, 200+60*k), cell*0.2)
			case b.ghost[i] > 0:
				p.Fill(box, t.Accent.Alpha(0.2*float32(b.ghost[i])/ghostLife), cell*0.2)
			}
		}
		p.Stroke(board, t.Border, 0, 1)
	})
	return view
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
