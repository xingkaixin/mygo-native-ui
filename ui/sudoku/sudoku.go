// Package sudoku plays a sudoku: the givens bold, the digits entered in
// the accent color, small notes of the digits a square may take, and a
// digit that repeats in its row, column or box in the danger color. The
// square chosen is lit, its row, column and box tinted, and the squares
// holding its digit tinted more.
//
//	ev := sudoku.View(c, &app.play, app.puzzle)
//	if ev.Solved {
//		c.Toast("Solved!")
//	}
//
// A click chooses a square, and the arrows move between them. Typing 1 to
// 9 enters the digit, taking it out of the notes of the squares it sees,
// or with Notes on, adds it to the square's notes or takes it out;
// Backspace, Delete or 0 clears the square, and N turns Notes on and off.
// Cmd+Z undoes. Givens never change.
package sudoku

import (
	"slices"
	"strconv"
	"unicode"

	"github.com/egoist/mygo/ui"
)

// State is what a game keeps: the digits entered and the notes, the
// square chosen, and the steps to undo. Notes makes digits typed notes.
// Keep it in the app's state, one for each puzzle.
type State struct {
	Notes   bool
	puzzle  string
	entries [81]uint8
	notes   [81]uint16
	at      int
	history []snapshot
	solved  bool
}

type snapshot struct {
	entries [81]uint8
	notes   [81]uint16
}

// Events is a game, and whether its last square was filled in right this
// frame.
type Events struct {
	ui.Element
	Solved bool
}

func given(puzzle string, i int) uint8 {
	if i < len(puzzle) && puzzle[i] >= '1' && puzzle[i] <= '9' {
		return puzzle[i] - '0'
	}
	return 0
}

// digit returns the digit at square i: given, entered, or 0.
func (s *State) digit(i int) uint8 {
	if g := given(s.puzzle, i); g != 0 {
		return g
	}
	return s.entries[i]
}

// sees reports whether squares a and b share a row, a column or a box.
func sees(a, b int) bool {
	return a/9 == b/9 || a%9 == b%9 || a/27 == b/27 && a%9/3 == b%9/3
}

// clashes reports whether the digit at i repeats in a square it sees.
func (s *State) clashes(i int) bool {
	d := s.digit(i)
	if d == 0 {
		return false
	}
	for j := range 81 {
		if j != i && sees(i, j) && s.digit(j) == d {
			return true
		}
	}
	return false
}

// Count returns how many squares hold d, for a pad of digits to dim those
// placed nine times.
func (s *State) Count(d int) int {
	n := 0
	for i := range 81 {
		if int(s.digit(i)) == d {
			n++
		}
	}
	return n
}

func (s *State) save() {
	s.history = append(s.history, snapshot{s.entries, s.notes})
}

// Put enters d in the square chosen as typing it does, 0 clearing it.
func (s *State) Put(d int) {
	i := s.at
	if given(s.puzzle, i) != 0 || d < 0 || d > 9 {
		return
	}
	if d == 0 {
		if s.entries[i] == 0 && s.notes[i] == 0 {
			return
		}
		s.save()
		s.entries[i], s.notes[i] = 0, 0
		return
	}
	s.save()
	bit := uint16(1) << d
	if s.Notes {
		if s.entries[i] == 0 {
			s.notes[i] ^= bit
		}
		return
	}
	if s.entries[i] == uint8(d) {
		s.entries[i] = 0
		return
	}
	s.entries[i], s.notes[i] = uint8(d), 0
	for j := range 81 {
		if sees(i, j) {
			s.notes[j] &^= bit
		}
	}
}

// Undo takes back the last digit or note.
func (s *State) Undo() {
	if n := len(s.history); n > 0 {
		h := s.history[n-1]
		s.entries, s.notes, s.history = h.entries, h.notes, s.history[:n-1]
	}
}

// View creates the grid of puzzle: 81 digits, row by row, with '.' or '0'
// for the empty squares.
func View(c *ui.Context, s *State, puzzle string) Events {
	t := c.Theme()
	if s.puzzle != puzzle {
		*s = State{puzzle: puzzle, Notes: s.Notes}
		s.at = max(slices.IndexFunc([]byte(puzzle), func(b byte) bool { return b < '1' || b > '9' }), 0)
	}
	size := t.Space(108)
	cell := size / 9
	grid := ui.Box(c).Size(size, size).Focusable().Label("Sudoku grid").Cursor(ui.CursorPointer)
	ev := Events{Element: grid}
	grid.HandleInput(func(e ui.InputEvent) bool {
		switch e.Kind {
		case ui.InputText:
			took := false
			for _, r := range e.Text {
				switch {
				case r >= '0' && r <= '9':
					s.Put(int(r - '0'))
				case unicode.ToLower(r) == 'n':
					s.Notes = !s.Notes
				default:
					continue
				}
				took = true
			}
			return took
		case ui.InputKeyDown:
		default:
			return false
		}
		r, col := s.at/9, s.at%9
		switch {
		case e.Key == ui.KeyZ && e.Mods == ui.Cmd:
			s.Undo()
		case e.Mods != 0:
			return false
		case e.Key == ui.KeyBackspace, e.Key == ui.KeyDelete:
			s.Put(0)
		case e.Key == ui.KeyLeft:
			s.at = r*9 + (col+8)%9
		case e.Key == ui.KeyRight:
			s.at = r*9 + (col+1)%9
		case e.Key == ui.KeyUp:
			s.at = (r+8)%9*9 + col
		case e.Key == ui.KeyDown:
			s.at = (r+1)%9*9 + col
		default:
			return false
		}
		return true
	})
	if grid.Clicked() {
		x, y, _ := grid.PointerPosition()
		s.at = min(int(y/cell), 8)*9 + min(int(x/cell), 8)
		grid.Focus()
	}

	solved := true
	for i := range 81 {
		if s.digit(i) == 0 || s.clashes(i) {
			solved = false
			break
		}
	}
	ev.Solved = solved && !s.solved
	s.solved = solved

	at, focused := s.at, grid.FocusWithin()
	lit := s.digit(at)
	grid.Draw(func(p *ui.Painter, r ui.Rect) {
		p.Fill(r, t.Background, 0)
		for i := range 81 {
			sq := ui.Rect{X: r.X + float32(i%9)*cell, Y: r.Y + float32(i/9)*cell, W: cell, H: cell}
			switch {
			case i == at && focused:
				p.Fill(sq, t.Accent.Alpha(0.35), 0)
			case i == at:
				p.Fill(sq, t.Accent.Alpha(0.2), 0)
			case lit != 0 && s.digit(i) == lit:
				p.Fill(sq, t.Accent.Alpha(0.2), 0)
			case sees(i, at):
				p.Fill(sq, t.Text.Alpha(0.05), 0)
			}
			if d := s.digit(i); d != 0 {
				color, weight := t.Accent, 500
				if given(s.puzzle, i) != 0 {
					color, weight = t.Text, 600
				}
				if s.clashes(i) {
					color = t.Danger
				}
				if solved {
					color = t.Success
				}
				span := ui.Span{Text: strconv.Itoa(int(d)), Size: cell * 0.55, Weight: weight, Color: color}
				w, h := p.MeasureText(0, span)
				p.RichText(sq.X+(cell-w)/2, sq.Y+(cell-h)/2, 0, span)
				continue
			}
			for d := 1; d <= 9; d++ {
				if s.notes[i]&(1<<d) == 0 {
					continue
				}
				color := t.TextMuted
				if int(lit) == d {
					color = t.Accent
				}
				span := ui.Span{Text: strconv.Itoa(d), Size: cell * 0.24, Weight: 500, Color: color}
				w, h := p.MeasureText(0, span)
				third := cell / 3
				p.RichText(sq.X+float32((d-1)%3)*third+(third-w)/2, sq.Y+float32((d-1)/3)*third+(third-h)/2, 0, span)
			}
		}
		for k := 0; k <= 9; k++ {
			width, color := float32(1), t.Border
			if k%3 == 0 {
				width, color = 2, t.Text
			}
			off := float32(k) * cell
			p.Line(r.X+off, r.Y, r.X+off, r.Y+size, width, color)
			p.Line(r.X, r.Y+off, r.X+size, r.Y+off, width, color)
		}
	})
	return ev
}
