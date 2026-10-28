// Package crossword plays a crossword, as the New York Times Mini and
// other crossword apps do: a grid with its squares numbered, the word
// being filled lit and its square lit more, the clue of that word above
// the grid, and the clues across and down beside it, those of words
// filled dimmed.
//
//	ev := crossword.View(c, &app.play, app.puzzle)
//	if ev.Solved {
//		c.Toast("Solved!")
//	}
//
// Typing a letter fills the square and moves on through the word, and
// to the next word at its end; Backspace clears the square, or the one
// before. The arrows move between squares, an arrow across the word's
// direction turning it first; Space and a second click on a square turn
// it too, and Tab and Shift+Tab go to the next and the previous word. A
// click on a clue goes to its word. Check marks the letters that are
// wrong, until they are typed over.
package crossword

import (
	"slices"
	"sort"
	"strconv"
	"unicode"

	"github.com/egoist/mygo/ui"
)

// Puzzle is a crossword: its grid as rows of the letters that fill it,
// '#' for a black square, and its clues by the numbers of their words.
type Puzzle struct {
	Grid   []string
	Across map[int]string
	Down   map[int]string
}

type cell struct{ r, c int }

// word is a word of the grid: its number, its direction, and its squares.
type word struct {
	n     int
	down  bool
	cells []cell
}

// State is what a crossword keeps: the letters typed, the square and the
// direction being filled, and the letters marked wrong. Keep it in the
// app's state, one for each puzzle.
type State struct {
	entries map[cell]rune
	wrong   map[cell]bool
	at      cell
	down    bool
	check   bool
	solved  bool
	grid    []string
}

// Check marks the letters typed that are wrong.
func (s *State) Check() { s.check = true }

// Events is a crossword, and whether its last letter was filled in right
// this frame.
type Events struct {
	ui.Element
	Solved bool
}

// layout is a grid read: its words, the numbers of its squares, and the
// words across and down through each square.
type layout struct {
	rows, cols int
	words      []word
	numbers    map[cell]int
	across     map[cell]int
	downs      map[cell]int
}

func read(grid []string) layout {
	l := layout{rows: len(grid), numbers: map[cell]int{}, across: map[cell]int{}, downs: map[cell]int{}}
	for _, row := range grid {
		l.cols = max(l.cols, len(row))
	}
	open := func(r, c int) bool { return r >= 0 && r < l.rows && c >= 0 && c < len(grid[r]) && grid[r][c] != '#' }
	n := 0
	for r := range l.rows {
		for c := range l.cols {
			if !open(r, c) {
				continue
			}
			startA := !open(r, c-1) && open(r, c+1)
			startD := !open(r-1, c) && open(r+1, c)
			if !startA && !startD {
				continue
			}
			n++
			l.numbers[cell{r, c}] = n
			if startA {
				w := word{n: n}
				for k := c; open(r, k); k++ {
					w.cells = append(w.cells, cell{r, k})
					l.across[cell{r, k}] = len(l.words)
				}
				l.words = append(l.words, w)
			}
			if startD {
				w := word{n: n, down: true}
				for k := r; open(k, c); k++ {
					w.cells = append(w.cells, cell{k, c})
					l.downs[cell{k, c}] = len(l.words)
				}
				l.words = append(l.words, w)
			}
		}
	}
	return l
}

// through returns the words across, or down, through each square.
func (l layout) through(down bool) map[cell]int {
	if down {
		return l.downs
	}
	return l.across
}

// current returns the word the state fills: the one in its direction
// through its square, or the other.
func (l layout) current(s *State) int {
	if i, ok := l.downs[s.at]; s.down && ok {
		return i
	}
	if i, ok := l.across[s.at]; ok {
		s.down = false
		return i
	}
	s.down = true
	return l.downs[s.at]
}

// View creates a crossword of p.
func View(c *ui.Context, s *State, p Puzzle) Events {
	t := c.Theme()
	l := read(p.Grid)
	if !slices.Equal(s.grid, p.Grid) {
		*s = State{entries: map[cell]rune{}, wrong: map[cell]bool{}, grid: slices.Clone(p.Grid)}
		if len(l.words) > 0 {
			s.at = l.words[0].cells[0]
		}
	}
	view := ui.Row(c).Gap(t.Space(8)).AlignItems(ui.Start)
	ev := Events{Element: view}
	if len(l.words) == 0 {
		return ev
	}
	if s.check {
		for at, r := range s.entries {
			if r != rune(p.Grid[at.r][at.c]) {
				s.wrong[at] = true
			}
		}
		s.check = false
	}
	cur := l.current(s)

	view.Children(func() {
		ui.Column(c).Gap(t.Space(3)).Children(func() {
			w := l.words[cur]
			dir, clues := "A", p.Across
			if w.down {
				dir, clues = "D", p.Down
			}
			ui.Row(c).Width(t.Space(110)).Gap(t.Space(3)).Padding(t.Space(3)).Radius(t.Radius).Background(t.Accent.Alpha(0.12)).Children(func() {
				ui.Text(c, strconv.Itoa(w.n)+dir).Bold()
				ui.Text(c, clues[w.n]).Grow(1).Shrink(1)
			})
			if board(c, s, p, l, cur) {
				ev.Solved = true
			}
		})
		ui.Column(c).Gap(t.Space(5)).Width(t.Space(70)).Children(func() {
			list(c, s, p, l, cur, false)
			list(c, s, p, l, cur, true)
		})
	})
	return ev
}

// board builds the grid, takes the keys typed on it, and reports whether
// the puzzle was solved this frame.
func board(c *ui.Context, s *State, p Puzzle, l layout, cur int) bool {
	t := c.Theme()
	size := t.Space(110)
	side := size / float32(max(l.rows, l.cols))
	grid := ui.Box(c).Size(side*float32(l.cols), side*float32(l.rows)).Focusable().Label("Crossword grid").Cursor(ui.CursorPointer)
	open := func(at cell) bool { _, a := l.across[at]; _, d := l.downs[at]; return a || d }
	move := func(dr, dc int) {
		for at := (cell{s.at.r + dr, s.at.c + dc}); at.r >= 0 && at.r < l.rows && at.c >= 0 && at.c < l.cols; at = (cell{at.r + dr, at.c + dc}) {
			if open(at) {
				s.at = at
				return
			}
		}
	}
	// Words go across first, then down, as the clues list them.
	order := make([]int, 0, len(l.words))
	for _, down := range []bool{false, true} {
		for i, w := range l.words {
			if w.down == down {
				order = append(order, i)
			}
		}
	}
	jump := func(by int) {
		k := slices.Index(order, l.current(s))
		w := l.words[order[(k+by+len(order))%len(order)]]
		s.at, s.down = w.cells[0], w.down
		for _, at := range w.cells {
			if s.entries[at] == 0 {
				s.at = at
				break
			}
		}
	}
	grid.HandleInput(func(ev ui.InputEvent) bool {
		switch ev.Kind {
		case ui.InputText:
			took := false
			for _, r := range ev.Text {
				if !unicode.IsLetter(r) {
					continue
				}
				took = true
				s.entries[s.at] = unicode.ToUpper(r)
				delete(s.wrong, s.at)
				w := l.words[l.current(s)]
				if k := slices.Index(w.cells, s.at); k < len(w.cells)-1 {
					s.at = w.cells[k+1]
				} else {
					jump(1)
				}
			}
			return took
		case ui.InputKeyDown:
		default:
			return false
		}
		turn := func(down bool) bool {
			if _, ok := l.through(down)[s.at]; ok && s.down != down {
				s.down = down
				return true
			}
			return false
		}
		switch ev.Key {
		case ui.KeyBackspace:
			if s.entries[s.at] == 0 {
				w := l.words[l.current(s)]
				if k := slices.Index(w.cells, s.at); k > 0 {
					s.at = w.cells[k-1]
				}
			}
			delete(s.entries, s.at)
			delete(s.wrong, s.at)
		case ui.KeySpace:
			turn(!s.down)
		case ui.KeyTab:
			if ev.Mods&ui.Shift != 0 {
				jump(-1)
			} else {
				jump(1)
			}
		case ui.KeyLeft:
			if !turn(false) {
				move(0, -1)
			}
		case ui.KeyRight:
			if !turn(false) {
				move(0, 1)
			}
		case ui.KeyUp:
			if !turn(true) {
				move(-1, 0)
			}
		case ui.KeyDown:
			if !turn(true) {
				move(1, 0)
			}
		default:
			return false
		}
		return true
	})
	if grid.Clicked() {
		x, y, _ := grid.PointerPosition()
		at := cell{int(y / side), int(x / side)}
		if open(at) {
			if at == s.at {
				s.down = !s.down
			}
			s.at = at
			grid.Focus()
		}
	}

	solved := len(s.entries) > 0
	for r, row := range p.Grid {
		for col, ch := range row {
			if ch != '#' && s.entries[cell{r, col}] != ch {
				solved = false
			}
		}
	}
	done := solved && !s.solved
	s.solved = solved

	cur = l.current(s)
	lit := l.words[cur].cells
	at, focused := s.at, grid.FocusWithin()
	grid.Draw(func(pa *ui.Painter, r ui.Rect) {
		pa.Fill(r, ui.RGB(20, 20, 20), 0)
		for row := range l.rows {
			for col := range l.cols {
				here := cell{row, col}
				if !open(here) {
					continue
				}
				sq := ui.Rect{X: r.X + float32(col)*side + 0.5, Y: r.Y + float32(row)*side + 0.5, W: side - 1, H: side - 1}
				bg := ui.RGB(255, 255, 255)
				switch {
				case here == at && focused:
					bg = ui.RGB(255, 218, 0)
				case here == at:
					bg = ui.RGB(255, 236, 140)
				case slices.Contains(lit, here):
					bg = ui.RGB(167, 216, 255)
				}
				pa.Fill(sq, bg, 0)
				if n, ok := l.numbers[here]; ok {
					pa.RichText(sq.X+3, sq.Y+2, 0, ui.Span{Text: strconv.Itoa(n), Size: side * 0.22, Color: ui.RGB(30, 30, 30)})
				}
				if e := s.entries[here]; e != 0 {
					color := ui.RGB(20, 20, 20)
					if solved {
						color = ui.RGB(37, 99, 235)
					}
					span := ui.Span{Text: string(e), Size: side * 0.55, Weight: 500, Color: color}
					w, h := pa.MeasureText(0, span)
					pa.RichText(sq.X+(sq.W-w)/2, sq.Y+sq.H*0.58-h/2, 0, span)
				}
				if s.wrong[here] {
					pa.StrokePolyline([][2]float32{{sq.X + 3, sq.Y + sq.H - 3}, {sq.X + sq.W - 3, sq.Y + 3}}, 2, t.Danger)
				}
			}
		}
	})
	return done
}

// list builds the clues across, or down, and goes to a clue's word on a
// click.
func list(c *ui.Context, s *State, p Puzzle, l layout, cur int, down bool) {
	t := c.Theme()
	title, clues := "Across", p.Across
	if down {
		title, clues = "Down", p.Down
	}
	cross := -1
	if down != l.words[cur].down {
		if i, ok := l.through(down)[s.at]; ok {
			cross = i
		}
	}
	ui.Column(c).Gap(t.Space(1)).Children(func() {
		ui.Text(c, title).Bold().FontSize(t.FontSize+1).Padding(0, 0, t.Space(1))
		idx := make([]int, 0, len(l.words))
		for i, w := range l.words {
			if w.down == down {
				idx = append(idx, i)
			}
		}
		sort.Slice(idx, func(a, b int) bool { return l.words[idx[a]].n < l.words[idx[b]].n })
		for _, i := range idx {
			w := l.words[i]
			row := ui.Row(c.Key(w.n)).Gap(t.Space(2)).Padding(t.Space(1.5), t.Space(2)).Radius(t.Radius).Cursor(ui.CursorPointer)
			switch {
			case i == cur:
				row.Background(t.Accent.Alpha(0.2))
			case i == cross:
				row.Background(t.Accent.Alpha(0.08))
			case row.Hovered():
				row.Background(t.SurfaceHover)
			}
			if row.Clicked() {
				s.at, s.down = w.cells[0], w.down
			}
			filled := true
			for _, at := range w.cells {
				if s.entries[at] == 0 {
					filled = false
				}
			}
			ink := t.Text
			if filled {
				ink = t.TextMuted
			}
			row.Children(func() {
				ui.Text(c, strconv.Itoa(w.n)).Bold().Width(t.Space(5)).TextColor(ink)
				ui.Text(c, clues[w.n]).Grow(1).Shrink(1).TextColor(ink)
			})
		}
	})
}
