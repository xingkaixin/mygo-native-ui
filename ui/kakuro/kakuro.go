// Package kakuro is a kakuro puzzle: a grid of white cells to fill with
// digits 1 to 9, in runs across and down whose sums stand in the dark
// cells before them, and no digit twice in a run. Click a cell or move
// with the arrow keys, type a digit, and clear it with Backspace. The
// runs through the cell picked are tinted, and beside the grid are the
// sets of digits that make their sums with what is already in them, and
// the digits the cell could take. A digit twice in a run, or a full run
// with the wrong sum, shows in red; a run done right dims its clue.
//
//	kakuro.View(c, &app.kakuro, app.puzzle, &app.digits)
package kakuro

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/egoist/mygo/ui"
)

// Puzzle is a puzzle: its size, which cells are white, and the sums of
// the runs, written in the dark cell before each run.
type Puzzle struct {
	Rows, Cols   int
	White        []bool
	Down, Across []int
	runs         []run
}

type run struct {
	cells  []int
	sum    int
	clue   int
	across bool
}

// Parse reads a puzzle from a solution of it, a row a line: # for a dark
// cell and a digit for a white one. The sums are those of the solution.
func Parse(s string) (Puzzle, error) {
	lines := strings.Fields(s)
	if len(lines) == 0 {
		return Puzzle{}, errors.New("no rows")
	}
	p := Puzzle{Rows: len(lines), Cols: len(lines[0])}
	n := p.Rows * p.Cols
	p.White, p.Down, p.Across = make([]bool, n), make([]int, n), make([]int, n)
	digits := make([]int, n)
	for r, line := range lines {
		if len(line) != p.Cols {
			return Puzzle{}, fmt.Errorf("row %d has %d cells, not %d", r+1, len(line), p.Cols)
		}
		for c, ch := range line {
			switch {
			case ch == '#':
			case ch >= '1' && ch <= '9':
				p.White[r*p.Cols+c], digits[r*p.Cols+c] = true, int(ch-'0')
			default:
				return Puzzle{}, fmt.Errorf("row %d has %q", r+1, ch)
			}
		}
	}
	for i := range n {
		if p.White[i] {
			continue
		}
		for _, across := range []bool{true, false} {
			step := p.Cols
			if across {
				step = 1
			}
			var ru run
			for j := i + step; j < n && p.White[j] && (!across || j/p.Cols == i/p.Cols); j += step {
				ru.cells = append(ru.cells, j)
				ru.sum += digits[j]
			}
			if len(ru.cells) == 0 {
				continue
			}
			if len(ru.cells) == 1 {
				return Puzzle{}, fmt.Errorf("a run of one cell at row %d, column %d", i/p.Cols+1, i%p.Cols+2)
			}
			ru.clue, ru.across = i, across
			if across {
				p.Across[i] = ru.sum
			} else {
				p.Down[i] = ru.sum
			}
			p.runs = append(p.runs, ru)
		}
	}
	return p, nil
}

// runsOf returns the runs through cell i.
func (p Puzzle) runsOf(i int) []run {
	var out []run
	for _, ru := range p.runs {
		if slices.Contains(ru.cells, i) {
			out = append(out, ru)
		}
	}
	return out
}

// check returns whether ru is full and right, and its cells that are
// wrong: a digit twice, or all of them if it is full with the wrong sum.
func check(ru run, digits []int) (done bool, wrong []int) {
	sum, full := 0, true
	for _, i := range ru.cells {
		d := digits[i]
		if d == 0 {
			full = false
			continue
		}
		sum += d
		for _, j := range ru.cells {
			if j != i && digits[j] == d {
				wrong = append(wrong, i)
				break
			}
		}
	}
	if full && sum != ru.sum {
		wrong = ru.cells
	}
	return full && len(wrong) == 0, wrong
}

// Solved reports whether digits fill every run rightly.
func (p Puzzle) Solved(digits []int) bool {
	for _, ru := range p.runs {
		if done, _ := check(ru, digits); !done {
			return false
		}
	}
	return len(p.runs) > 0
}

// combos returns the sets of n different digits that sum to sum and
// hold every digit of have, each as a bit mask of its digits.
func combos(sum, n int, have []int) []uint16 {
	var need uint16
	for _, d := range have {
		need |= 1 << d
	}
	var out []uint16
	for m := uint16(0); m < 1<<10; m += 2 {
		total, count := 0, 0
		for d := 1; d <= 9; d++ {
			if m&(1<<d) != 0 {
				total += d
				count++
			}
		}
		if total == sum && count == n && m&need == need {
			out = append(out, m)
		}
	}
	return out
}

func digitsOf(m uint16) string {
	var b strings.Builder
	for d := 1; d <= 9; d++ {
		if m&(1<<d) != 0 {
			b.WriteByte(byte('0' + d))
		}
	}
	return b.String()
}

// State is what a puzzle keeps: the cell picked, which is never the top
// left, as it is dark. Keep it in the app's state, one for each puzzle.
type State struct {
	selected int
}

// View creates puzzle p with the digits in *digits, a cell each, 0 for
// none.
func View(c *ui.Context, s *State, p Puzzle, digits *[]int) ui.Element {
	t := c.Theme()
	n := p.Rows * p.Cols
	if len(*digits) != n {
		*digits = make([]int, n)
	}
	d := *digits
	if s.selected <= 0 || s.selected >= n || !p.White[s.selected] {
		s.selected = slices.Index(p.White, true)
	}
	changed := false
	set := func(v int) {
		if s.selected >= 0 && d[s.selected] != v {
			d[s.selected], changed = v, true
		}
	}
	for k := 1; k <= 9; k++ {
		if c.Shortcut(0, ui.Key0+ui.Key(k)) {
			set(k)
		}
	}
	if c.Shortcut(0, ui.KeyBackspace) || c.Shortcut(0, ui.KeyDelete) || c.Shortcut(0, ui.Key0) {
		set(0)
	}
	move := func(dr, dc int) {
		r, col := s.selected/p.Cols+dr, s.selected%p.Cols+dc
		for r >= 0 && r < p.Rows && col >= 0 && col < p.Cols {
			if i := r*p.Cols + col; p.White[i] {
				s.selected = i
				return
			}
			r, col = r+dr, col+dc
		}
	}
	switch {
	case c.Shortcut(0, ui.KeyUp):
		move(-1, 0)
	case c.Shortcut(0, ui.KeyDown):
		move(1, 0)
	case c.Shortcut(0, ui.KeyLeft):
		move(0, -1)
	case c.Shortcut(0, ui.KeyRight):
		move(0, 1)
	}
	wrong := map[int]bool{}
	doneClue := map[[2]int]bool{}
	for _, ru := range p.runs {
		done, bad := check(ru, d)
		for _, i := range bad {
			wrong[i] = true
		}
		if done {
			doneClue[[2]int{ru.clue, map[bool]int{false: 0, true: 1}[ru.across]}] = true
		}
	}
	through := p.runsOf(s.selected)
	inRun := map[int]bool{}
	for _, ru := range through {
		for _, i := range ru.cells {
			inRun[i] = true
		}
	}
	block, clueInk := ui.Hex("#3A3A3E"), ui.Hex("#F2F2F2")
	white := t.Background
	if t.Dark {
		block, white = ui.Hex("#0C0C0E"), t.Surface
	}
	size := t.Space(12)
	root := ui.Row(c).Gap(t.Space(8)).AlignItems(ui.Start)
	root.Children(func() {
		ui.Grid(c).Columns(p.Cols).Gap(1).Padding(1).Background(t.Border).Children(func() {
			for i := range n {
				if !p.White[i] {
					down, across := p.Down[i], p.Across[i]
					downDone, acrossDone := doneClue[[2]int{i, 0}], doneClue[[2]int{i, 1}]
					ui.Box(c.Key(i)).Size(size, size).Background(block).Draw(func(pt *ui.Painter, r ui.Rect) {
						if down == 0 && across == 0 {
							return
						}
						pt.Line(r.X, r.Y, r.X+r.W, r.Y+r.H, 1, clueInk.Alpha(0.35))
						fs := size * 0.3
						ink := func(done bool) ui.Color {
							if done {
								return clueInk.Alpha(0.35)
							}
							return clueInk
						}
						if across > 0 {
							label := fmt.Sprint(across)
							w, _ := pt.MeasureText(0, ui.Span{Text: label, Size: fs})
							pt.Text(r.X+r.W-w-size*0.08, r.Y+size*0.06, label, fs, ink(acrossDone))
						}
						if down > 0 {
							pt.Text(r.X+size*0.08, r.Y+r.H-fs*1.35, fmt.Sprint(down), fs, ink(downDone))
						}
					})
					continue
				}
				b := ui.ButtonBase(c.Key(i)).Size(size, size).Center().Label(fmt.Sprintf("Row %d, column %d", i/p.Cols+1, i%p.Cols+1))
				switch {
				case i == s.selected:
					b.Background(t.Accent.Alpha(0.28).Over(white))
				case inRun[i]:
					b.Background(t.Accent.Alpha(0.09).Over(white))
				default:
					b.Background(white)
				}
				if b.Clicked() {
					s.selected = i
				}
				if v := d[i]; v > 0 {
					ink := t.Text
					if wrong[i] {
						ink = t.Danger
					}
					b.Children(func() {
						ui.Text(c, fmt.Sprint(v)).FontSize(size * 0.5).TextColor(ink)
					})
				}
			}
		})
		ui.Column(c).Width(t.Space(56)).Gap(t.Space(4)).Children(func() {
			if p.Solved(d) {
				ui.Text(c, "Solved").Bold().FontSize(t.FontSize + 2).TextColor(t.Success)
			}
			could := uint16(0b1111111110)
			for _, ru := range through {
				var have []int
				for _, i := range ru.cells {
					if d[i] > 0 && i != s.selected {
						have = append(have, d[i])
					}
				}
				sets := combos(ru.sum, len(ru.cells), have)
				union := uint16(0)
				for _, m := range sets {
					union |= m
				}
				for _, v := range have {
					union &^= 1 << v
				}
				could &= union
				dir := "Down"
				if ru.across {
					dir = "Across"
				}
				ui.Column(c.Key(dir)).Gap(t.Space(1)).Children(func() {
					ui.Text(c, fmt.Sprintf("%s %d in %d", dir, ru.sum, len(ru.cells))).Bold()
					if len(sets) == 0 {
						ui.Text(c, "No set fits what is there").TextColor(t.Danger)
						return
					}
					ui.Row(c).Wrap().Gap(t.Space(1.5)).Children(func() {
						for _, m := range sets {
							ui.Text(c.Key(m), digitsOf(m)).Font("monospace").FontFeatures("tnum").Padding(t.Space(0.5), t.Space(1.5)).Radius(t.Radius).Background(t.Surface)
						}
					})
				})
			}
			ui.Column(c).Gap(t.Space(1)).Children(func() {
				ui.Text(c, "Could be").Bold()
				text := digitsOf(could)
				if text == "" {
					text = "—"
				}
				ui.Text(c, strings.Join(strings.Split(text, ""), " ")).Font("monospace").FontSize(t.FontSize + 4).TextColor(t.Accent)
			})
		})
	})
	if changed {
		root.SetChanged()
	}
	return root
}
