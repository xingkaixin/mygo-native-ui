// Package nonogram plays a nonogram, as picross games do: a grid with
// the runs of filled squares of each row written before it and of each
// column above it, to be worked out and filled. A clue whose line holds
// those runs dims, the row and column under the pointer are tinted, and
// thicker lines mark every five squares. Once every line holds its runs,
// the picture shows on its own.
//
//	ev := nonogram.View(c, &app.play, app.picture)
//	if ev.Solved {
//		c.Toast("Solved!")
//	}
//
// A click fills a square or empties it, and a right click crosses it out
// or clears it; dragging does the same along the row or column it starts
// along. The arrows move the square chosen, Space fills it, and X crosses
// it out.
package nonogram

import (
	"slices"
	"strconv"

	"github.com/egoist/mygo/ui"
)

const (
	empty byte = iota
	filled
	crossed
)

// State is what a game keeps: the squares filled and crossed out, the
// square chosen, and the drag. Keep it in the app's state, one for each
// puzzle.
type State struct {
	picture []string
	cells   [][]byte
	row     int
	col     int
	solved  bool
	// paint is what a drag sets the squares to, from the square at, along
	// a row or a column once it leaves it; painting is false between
	// drags.
	painting bool
	paint    byte
	at       [2]int
	axis     int
	// cell is a square's side, and ox, oy where the grid's top left shows,
	// as of the last layout.
	cell, ox, oy float32
}

// Events is a puzzle, and whether it was solved this frame.
type Events struct {
	ui.Element
	Solved bool
}

// runs returns the lengths of the runs of on squares in line.
func runs(line []bool) []int {
	var out []int
	n := 0
	for _, on := range line {
		if on {
			n++
		} else if n > 0 {
			out = append(out, n)
			n = 0
		}
	}
	if n > 0 {
		out = append(out, n)
	}
	return out
}

// clues returns the runs of each row and column of grid, where on tells
// a square that counts.
func clues[T any](grid [][]T, on func(T) bool) (rows, cols [][]int) {
	if len(grid) == 0 {
		return nil, nil
	}
	for _, r := range grid {
		line := make([]bool, len(r))
		for i, v := range r {
			line[i] = on(v)
		}
		rows = append(rows, runs(line))
	}
	for c := range grid[0] {
		line := make([]bool, len(grid))
		for r := range grid {
			line[r] = on(grid[r][c])
		}
		cols = append(cols, runs(line))
	}
	return rows, cols
}

// View creates the puzzle whose solution is picture, a string for each
// row, with # for a filled square.
func View(c *ui.Context, s *State, picture []string) Events {
	t := c.Theme()
	if !slices.Equal(s.picture, picture) {
		s.picture, s.solved, s.row, s.col = picture, false, 0, 0
		s.cells = make([][]byte, len(picture))
		for r, line := range picture {
			s.cells[r] = make([]byte, len(line))
		}
	}
	grid := ui.Box(c).Focusable().Label("Nonogram")
	ev := Events{Element: grid}
	h := len(s.cells)
	if h == 0 {
		return ev
	}
	w := len(s.cells[0])
	solution := make([][]byte, h)
	for r, line := range picture {
		solution[r] = []byte(line)
	}
	rowClues, colClues := clues(solution, func(b byte) bool { return b == '#' })
	gotRows, gotCols := clues(s.cells, func(b byte) bool { return b == filled })
	solved := true
	for r := range h {
		solved = solved && slices.Equal(rowClues[r], gotRows[r])
	}
	for col := range w {
		solved = solved && slices.Equal(colClues[col], gotCols[col])
	}
	ev.Solved = solved && !s.solved
	s.solved = solved

	most := func(cl [][]int) int {
		n := 1
		for _, c := range cl {
			n = max(n, len(c))
		}
		return n
	}
	left, top := float32(most(rowClues))*0.62+0.4, float32(most(colClues))*0.72+0.4
	b := grid.LayoutBounds()
	s.cell = float32(int(min(b.W/(float32(w)+left), b.H/(float32(h)+top))))
	s.ox = (b.W-s.cell*(float32(w)+left))/2 + s.cell*left
	s.oy = (b.H-s.cell*(float32(h)+top))/2 + s.cell*top

	at := func(x, y float32) (int, int, bool) {
		if s.cell <= 0 || x < s.ox || y < s.oy {
			return 0, 0, false
		}
		col, row := int((x-s.ox)/s.cell), int((y-s.oy)/s.cell)
		return row, col, row < h && col < w
	}
	set := func(r, col int, v byte) {
		if !s.solved {
			s.cells[r][col] = v
		}
	}
	toggle := func(v byte) {
		if s.cells[s.row][s.col] == v {
			v = empty
		}
		set(s.row, s.col, v)
	}
	grid.HandleInput(func(in ui.InputEvent) bool {
		switch in.Kind {
		case ui.InputPointerDown:
			r, col, ok := at(in.X, in.Y)
			if !ok || in.Button > 1 {
				return false
			}
			want := filled
			if in.Button == 1 {
				want = crossed
			}
			s.paint = want
			if s.cells[r][col] == want {
				s.paint = empty
			}
			s.painting, s.at, s.axis, s.row, s.col = true, [2]int{r, col}, -1, r, col
			set(r, col, s.paint)
		case ui.InputPointerMove:
			if !s.painting {
				return false
			}
			r, col, ok := at(in.X, in.Y)
			if !ok {
				return true
			}
			if s.axis < 0 && (r != s.at[0] || col != s.at[1]) {
				s.axis = 0
				if col == s.at[1] {
					s.axis = 1
				}
			}
			switch s.axis {
			case 0:
				r = s.at[0]
			case 1:
				col = s.at[1]
			}
			// A fast drag skips squares between two moves, which it paints
			// too.
			for s.row != r || s.col != col {
				s.row += sign(r - s.row)
				s.col += sign(col - s.col)
				// Squares already crossed out keep their cross under a
				// fill stroke, as only an empty one is free to fill.
				if s.paint != filled || s.cells[s.row][s.col] != crossed {
					set(s.row, s.col, s.paint)
				}
			}
		case ui.InputPointerUp:
			if !s.painting {
				return false
			}
			s.painting = false
		case ui.InputKeyDown:
			switch in.Key {
			case ui.KeyLeft:
				s.col = max(s.col-1, 0)
			case ui.KeyRight:
				s.col = min(s.col+1, w-1)
			case ui.KeyUp:
				s.row = max(s.row-1, 0)
			case ui.KeyDown:
				s.row = min(s.row+1, h-1)
			case ui.KeySpace:
				toggle(filled)
			case ui.KeyX:
				toggle(crossed)
			default:
				return false
			}
		default:
			return false
		}
		return true
	})

	hr, hc := -1, -1
	if px, py, over := grid.PointerPosition(); over {
		if r, col, ok := at(px, py); ok {
			hr, hc = r, col
		}
	}
	focused := grid.Focused()
	grid.Draw(func(p *ui.Painter, rect ui.Rect) {
		cell := s.cell
		x0, y0 := rect.X+s.ox, rect.Y+s.oy
		gw, gh := cell*float32(w), cell*float32(h)
		if !solved {
			if hr >= 0 {
				p.Fill(ui.Rect{X: rect.X + s.ox - cell*left, Y: y0 + float32(hr)*cell, W: gw + cell*left, H: cell}, t.Accent.Alpha(0.08), 0)
				p.Fill(ui.Rect{X: x0 + float32(hc)*cell, Y: rect.Y + s.oy - cell*top, W: cell, H: gh + cell*top}, t.Accent.Alpha(0.08), 0)
			}
		}
		size := cell * 0.44
		for r, cl := range rowClues {
			done := slices.Equal(cl, gotRows[r])
			if len(cl) == 0 {
				cl = []int{0}
			}
			for k, n := range cl {
				span := clueSpan(t, n, size, done)
				tw, th := p.MeasureText(0, span)
				x := x0 - cell*0.3 - (float32(len(cl)-k)-0.5)*cell*0.62
				p.RichText(x-tw/2, y0+(float32(r)+0.5)*cell-th/2, 0, span)
			}
		}
		for col, cl := range colClues {
			done := slices.Equal(cl, gotCols[col])
			if len(cl) == 0 {
				cl = []int{0}
			}
			for k, n := range cl {
				span := clueSpan(t, n, size, done)
				tw, th := p.MeasureText(0, span)
				y := y0 - cell*0.25 - (float32(len(cl)-k)-0.5)*cell*0.72
				p.RichText(x0+(float32(col)+0.5)*cell-tw/2, y-th/2, 0, span)
			}
		}
		for r := range h {
			for col := range w {
				box := ui.Rect{X: x0 + float32(col)*cell, Y: y0 + float32(r)*cell, W: cell, H: cell}
				switch s.cells[r][col] {
				case filled:
					inset := float32(1)
					if solved {
						inset = 0
					}
					p.Fill(ui.Rect{X: box.X + inset, Y: box.Y + inset, W: cell - 2*inset, H: cell - 2*inset}, t.Text, 0)
				case crossed:
					if !solved {
						d := cell * 0.3
						p.StrokePolyline([][2]float32{{box.X + d, box.Y + d}, {box.X + cell - d, box.Y + cell - d}}, 1.5, t.TextMuted)
						p.StrokePolyline([][2]float32{{box.X + cell - d, box.Y + d}, {box.X + d, box.Y + cell - d}}, 1.5, t.TextMuted)
					}
				}
			}
		}
		if !solved {
			for i := 0; i <= w; i++ {
				width, color := float32(1), t.Border
				if i%5 == 0 || i == w {
					width, color = 2, t.TextMuted
				}
				x := x0 + float32(i)*cell
				p.Line(x, y0, x, y0+gh, width, color)
			}
			for i := 0; i <= h; i++ {
				width, color := float32(1), t.Border
				if i%5 == 0 || i == h {
					width, color = 2, t.TextMuted
				}
				y := y0 + float32(i)*cell
				p.Line(x0, y, x0+gw, y, width, color)
			}
			if focused {
				p.Stroke(ui.Rect{X: x0 + float32(s.col)*cell, Y: y0 + float32(s.row)*cell, W: cell, H: cell}, t.Accent, 0, 2.5)
			}
		}
	})
	return ev
}

func sign(n int) int {
	switch {
	case n > 0:
		return 1
	case n < 0:
		return -1
	}
	return 0
}

func clueSpan(t *ui.Theme, n int, size float32, done bool) ui.Span {
	span := ui.Span{Text: strconv.Itoa(n), Size: size, Weight: 600, Color: t.Text, Features: "tnum"}
	if done {
		span.Color, span.Weight = t.TextMuted.Alpha(0.6), 400
	}
	return span
}
