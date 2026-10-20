// Package spreadsheet is a sheet of cells, as spreadsheet apps are: a
// grid under lettered columns and numbered rows, each cell a number, text
// or a formula, as =SUM(B2:B9)*1.2, worked out again as the cells it
// refers to change. A formula takes + - * / ^, comparisons, parentheses,
// references and ranges, and SUM, AVERAGE, MIN, MAX, COUNT, ROUND, ABS
// and IF; what it cannot work out shows as #DIV/0!, #VALUE!, #NAME?,
// #CYCLE! or #ERROR!.
//
// A click chooses a cell and a drag a range, whose sum and average show
// below; the arrows move, with Shift to stretch the range, and Tab moves
// right. Typing replaces the cell, and Enter or a double click edits it,
// its references outlined as it is typed; Enter and Tab keep it, and
// Escape leaves it. Delete clears the range, Cmd+C copies it and Cmd+V
// pastes rows of tab-separated cells, as other spreadsheets copy them.
// The bar above edits the cell chosen; the column edges in the header
// change the columns' widths. Changed reports an edit.
//
//	spreadsheet.View(c, &app.sheet, &app.cells, 12, 60).Grow(1)
package spreadsheet

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/egoist/mygo/ui"
)

// State is what a sheet keeps: the range chosen, the cell being edited
// and what is typed in it, the columns' widths and where it is scrolled.
// Keep it in the app's state, one for each sheet.
type State struct {
	anchor, cursor Cell
	editing        bool
	edit           Cell
	draft, bar     string
	widths         map[int]float32
	scroll         ui.ScrollState
	// focus gives the grid the focus, moved scrolls the cursor into view,
	// and copy and changed are what keys did, done in the next frame.
	focus, moved, copy, changed bool
	down                        bool
}

func (s *State) span() (Cell, Cell) {
	a, b := s.anchor, s.cursor
	return Cell{min(a.Col, b.Col), min(a.Row, b.Row)}, Cell{max(a.Col, b.Col), max(a.Row, b.Row)}
}

func (s *State) chosen(c Cell) bool {
	lo, hi := s.span()
	return c.Col >= lo.Col && c.Col <= hi.Col && c.Row >= lo.Row && c.Row <= hi.Row
}

var refs = regexp.MustCompile(`(?i)\b[A-Z]+[0-9]+(?::[A-Z]+[0-9]+)?\b`)

// View creates the sheet of *cells, cols columns by rows rows, each by
// the text typed in it.
func View(c *ui.Context, s *State, cells *map[Cell]string, cols, rows int) ui.Element {
	t := c.Theme()
	if *cells == nil {
		*cells = map[Cell]string{}
	}
	if s.widths == nil {
		s.widths = map[int]float32{}
	}
	cs := *cells
	sh := newSheet(cs)
	rowH, headH, numW := t.Space(7), t.Space(6.5), t.Space(12)
	xs := make([]float32, cols+1)
	for k := range cols {
		w, ok := s.widths[k]
		if !ok {
			w = t.Space(26)
		}
		xs[k+1] = xs[k] + w
	}
	move := func(dc, dr int, stretch bool) {
		s.cursor = Cell{max(0, min(cols-1, s.cursor.Col+dc)), max(0, min(rows-1, s.cursor.Row+dr))}
		if !stretch {
			s.anchor = s.cursor
		}
		s.moved = true
	}
	commit := func(text string) {
		if strings.TrimSpace(text) == "" {
			delete(cs, s.edit)
		} else {
			cs[s.edit] = text
		}
		s.editing, s.draft, s.changed = false, "", true
	}
	if s.copy {
		c.WriteClipboard(s.tsv(sh))
		s.copy = false
	}
	root := ui.Column(c).Gap(t.Space(2))
	if s.changed {
		root.SetChanged()
		s.changed = false
	}
	root.Children(func() {
		ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
			lo, hi := s.span()
			where := lo.Name()
			if lo != hi {
				where += ":" + hi.Name()
			}
			ui.Text(c, where).Font("monospace").FontSize(t.FontSize-1).Width(t.Space(22)).
				Padding(t.Space(1.5), t.Space(2)).Radius(t.Radius).Background(t.Surface)
			ui.Text(c, "ƒx").TextColor(t.TextMuted)
			bar := ui.TextInput(c, &s.bar).Grow(1).Font("monospace").Label("Formula")
			if bar.Focused() {
				if bar.Changed() {
					s.edit = s.cursor
					commit(s.bar)
				}
			} else if s.editing {
				s.bar = s.draft
			} else {
				s.bar = cs[s.cursor]
			}
			bar.OnTextKey(func(k ui.TextKey) bool {
				if k.Key == ui.KeyEnter || k.Key == ui.KeyEscape {
					s.focus = true
					return true
				}
				return false
			})
		})
		ui.Row(c).Grow(1).AlignSelf(ui.Stretch).AlignItems(ui.Stretch).Border(1, t.Border).Clip().Children(func() {
			ui.Column(c).Width(numW).Shrink(0).Children(func() {
				ui.Box(c).Height(headH).Background(t.Surface)
				ui.Box(c).Grow(1).Clip().Background(t.Surface).Draw(func(p *ui.Painter, r ui.Rect) {
					lo, hi := s.span()
					for row := int(s.scroll.Y / rowH); row < rows && float32(row)*rowH-s.scroll.Y < r.H; row++ {
						y := r.Y + float32(row)*rowH - s.scroll.Y
						if row >= lo.Row && row <= hi.Row {
							p.Fill(ui.Rect{X: r.X, Y: y, W: r.W, H: rowH}, t.Accent.Alpha(0.15), 0)
						}
						label := fmt.Sprint(row + 1)
						w, h := p.MeasureText(0, ui.Span{Text: label, Size: t.FontSize - 2})
						p.RichText(r.X+(r.W-w)/2, y+(rowH-h)/2, 0, ui.Span{Text: label, Size: t.FontSize - 2, Color: t.TextMuted})
						p.Line(r.X, y+rowH, r.X+r.W, y+rowH, 1, t.Border)
					}
					p.Line(r.X+r.W-1, r.Y, r.X+r.W-1, r.Y+r.H, 1, t.Border)
				})
			})
			ui.Column(c).Grow(1).Shrink(1).Children(func() {
				head := ui.Box(c).Height(headH).AlignSelf(ui.Stretch).Clip().Background(t.Surface)
				head.Draw(func(p *ui.Painter, r ui.Rect) {
					lo, hi := s.span()
					for k := range cols {
						x := r.X + xs[k] - s.scroll.X
						if x > r.X+r.W || x+xs[k+1]-xs[k] < r.X {
							continue
						}
						w := xs[k+1] - xs[k]
						if k >= lo.Col && k <= hi.Col {
							p.Fill(ui.Rect{X: x, Y: r.Y, W: w, H: r.H}, t.Accent.Alpha(0.15), 0)
						}
						label := column(k)
						lw, lh := p.MeasureText(0, ui.Span{Text: label, Size: t.FontSize - 2})
						p.RichText(x+(w-lw)/2, r.Y+(r.H-lh)/2, 0, ui.Span{Text: label, Size: t.FontSize - 2, Color: t.TextMuted})
						p.Line(x+w, r.Y, x+w, r.Y+r.H, 1, t.Border)
					}
					p.Line(r.X, r.Y+r.H-1, r.X+r.W, r.Y+r.H-1, 1, t.Border)
				})
				head.Children(func() {
					for k := range cols {
						edge := ui.Box(c.Key(k)).Absolute().Left(xs[k+1]-s.scroll.X-3).Top(0).Size(6, headH).
							Cursor(ui.CursorResizeEW).Label("Width of column " + column(k))
						if dx, _, ok := edge.Dragged(); ok {
							s.widths[k] = max(t.Space(10), xs[k+1]-xs[k]+dx)
						}
					}
				})
				scroll := ui.ScrollBoth(c).Grow(1).TrackScroll(&s.scroll)
				height := scroll.LayoutBounds().H
				scroll.Children(func() {
					s.grid(c, sh, cells, cols, rows, xs, rowH, height, move, commit)
				})
			})
		})
		lo, hi := s.span()
		line := ""
		if lo != hi {
			var sum float64
			count := 0
			for r := lo.Row; r <= hi.Row; r++ {
				for k := lo.Col; k <= hi.Col; k++ {
					if v := sh.get(Cell{k, r}); v.isNum {
						sum, count = sum+v.num, count+1
					}
				}
			}
			if count > 0 {
				line = fmt.Sprintf("Sum %s · Average %s · Count %d", n(sum), n(sum/float64(count)), count)
			}
		}
		ui.Text(c, line).FontSize(t.FontSize - 1).FontFeatures("tnum").TextColor(t.TextMuted).AlignSelf(ui.End)
	})
	return root
}

// grid shows the cells, those of the rows in view of a scroll container
// height tall, and the input that takes the keys.
func (s *State) grid(c *ui.Context, sh *sheet, cells *map[Cell]string, cols, rows int, xs []float32, rowH, height float32, move func(dc, dr int, stretch bool), commit func(string)) {
	t := c.Theme()
	cs := *cells
	at := func(x, y float32) Cell {
		k := 0
		for k < cols-1 && x >= xs[k+1] {
			k++
		}
		return Cell{k, max(0, min(rows-1, int(y/rowH)))}
	}
	box := ui.Box(c).Size(xs[cols], float32(rows)*rowH).Background(t.Background)
	pressed := box.Pressed()
	if pressed {
		if x, y, ok := box.PointerPosition(); ok {
			cell := at(x, y)
			if !s.down {
				if s.editing {
					commit(s.draft)
				}
				s.anchor = cell
				s.focus = true
			}
			s.cursor = cell
		}
	}
	s.down = pressed
	if box.DoubleClicked() {
		s.editing, s.edit, s.draft = true, s.cursor, cs[s.cursor]
		s.focus = true
	}
	if !s.editing && s.draft != "" {
		// A key typed while the cell was chosen replaces it.
		s.editing, s.edit = true, s.cursor
	}
	view := s.scroll
	box.Draw(func(p *ui.Painter, r ui.Rect) {
		lo, hi := s.span()
		top, bottom := int(view.Y/rowH), min(rows, int((view.Y+height)/rowH)+1)
		for row := top; row < bottom; row++ {
			y := r.Y + float32(row)*rowH
			for k := range cols {
				cell := Cell{k, row}
				x, w := r.X+xs[k], xs[k+1]-xs[k]
				area := ui.Rect{X: x, Y: y, W: w, H: rowH}
				if s.chosen(cell) && lo != hi {
					p.Fill(area, t.Accent.Alpha(0.1), 0)
				}
				if s.editing && cell == s.edit {
					continue
				}
				v := sh.get(cell)
				text := v.String()
				if text == "" {
					continue
				}
				color := t.Text
				if v.err != "" {
					color = t.Danger
				}
				span := ui.Span{Text: text, Size: t.FontSize - 1, Color: color}
				tw, th := p.MeasureText(0, span)
				tx := x + t.Space(1.5)
				if v.isNum {
					tx = x + w - t.Space(1.5) - tw
				}
				p.Clip(area, 0, func() { p.RichText(tx, y+(rowH-th)/2, 0, span) })
			}
		}
		for row := top; row <= bottom; row++ {
			y := r.Y + float32(row)*rowH
			p.Line(r.X, y, r.X+r.W, y, 1, t.Border.Alpha(0.6))
		}
		for k := 1; k <= cols; k++ {
			p.Line(r.X+xs[k], r.Y, r.X+xs[k], r.Y+r.H, 1, t.Border.Alpha(0.6))
		}
		if s.editing && strings.HasPrefix(s.draft, "=") {
			for i, m := range refs.FindAllString(s.draft, -1) {
				a, b, _ := strings.Cut(m, ":")
				from, ok := parseCell(a)
				to := from
				if b != "" {
					to, _ = parseCell(b)
				}
				if !ok || from.Col >= cols || to.Col >= cols {
					continue
				}
				x0, x1 := xs[min(from.Col, to.Col)], xs[max(from.Col, to.Col)+1]
				y0, y1 := float32(min(from.Row, to.Row))*rowH, float32(max(from.Row, to.Row)+1)*rowH
				color := ui.Oklch(0.62, 0.17, float32((250+i*67)%360))
				p.Fill(ui.Rect{X: r.X + x0, Y: r.Y + y0, W: x1 - x0, H: y1 - y0}, color.Alpha(0.1), 0)
				p.Stroke(ui.Rect{X: r.X + x0, Y: r.Y + y0, W: x1 - x0, H: y1 - y0}, color, 0, 1.5)
			}
		}
		cur := ui.Rect{X: r.X + xs[s.cursor.Col], Y: r.Y + float32(s.cursor.Row)*rowH, W: xs[s.cursor.Col+1] - xs[s.cursor.Col], H: rowH}
		p.Stroke(cur, t.Accent, 0, 2)
	})
	box.Children(func() {
		cur := s.cursor
		mark := ui.Box(c).Absolute().Left(xs[cur.Col]).Top(float32(cur.Row)*rowH).Size(xs[cur.Col+1]-xs[cur.Col], rowH)
		if s.moved {
			mark.ScrollIntoView()
			s.moved = false
		}
		cell := cur
		if s.editing {
			cell = s.edit
		}
		in := ui.TextInputBase(c, &s.draft).Absolute().Left(xs[cell.Col]).Top(float32(cell.Row) * rowH).
			FontSize(t.FontSize - 1).Label("Cell " + cell.Name())
		if s.editing {
			in.MinWidth(xs[cell.Col+1]-xs[cell.Col]).Height(rowH).Padding(0, t.Space(1.5)).
				Background(t.Background).Border(2, t.Accent).Shadow(0, 2, 8, 0, ui.RGB(0, 0, 0).Alpha(0.15))
		} else {
			// Out of the way of the pointer, it still takes the keys.
			in.Size(1, 1).Opacity(0)
		}
		if s.focus {
			in.Focus()
			if s.editing {
				in.SetTextSelection(len([]rune(s.draft)), len([]rune(s.draft)))
			}
			s.focus = false
		}
		in.OnPaste(func(text string) bool {
			if s.editing {
				return false
			}
			s.paste(cs, text, cols, rows)
			return true
		})
		in.OnTextKey(func(k ui.TextKey) bool { return s.key(cs, k, move, commit) })
	})
}

// key does what key k does in the grid, and reports whether it did
// anything.
func (s *State) key(cs map[Cell]string, k ui.TextKey, move func(dc, dr int, stretch bool), commit func(string)) bool {
	shift := k.Mods == ui.Shift
	if s.editing {
		switch {
		case k.Key == ui.KeyEscape:
			s.editing, s.draft = false, ""
			k.SetText("")
		case k.Key == ui.KeyEnter && k.Mods == 0, k.Key == ui.KeyDown && k.Mods == 0:
			commit(k.Text())
			k.SetText("")
			move(0, 1, false)
		case k.Key == ui.KeyUp && k.Mods == 0:
			commit(k.Text())
			k.SetText("")
			move(0, -1, false)
		case k.Key == ui.KeyTab:
			commit(k.Text())
			k.SetText("")
			if shift {
				move(-1, 0, false)
			} else {
				move(1, 0, false)
			}
		default:
			return false
		}
		return true
	}
	if k.Mods != 0 && !shift {
		if k.Mods == ui.Cmd && k.Key == ui.KeyC {
			s.copy = true
			return true
		}
		return false
	}
	switch k.Key {
	case ui.KeyUp:
		move(0, -1, shift)
	case ui.KeyDown:
		move(0, 1, shift)
	case ui.KeyLeft:
		move(-1, 0, shift)
	case ui.KeyRight:
		move(1, 0, shift)
	case ui.KeyTab:
		if shift {
			move(-1, 0, false)
		} else {
			move(1, 0, false)
		}
	case ui.KeyEnter:
		s.editing, s.edit = true, s.cursor
		k.SetText(cs[s.cursor])
	case ui.KeyBackspace, ui.KeyDelete:
		lo, hi := s.span()
		for r := lo.Row; r <= hi.Row; r++ {
			for c := lo.Col; c <= hi.Col; c++ {
				delete(cs, Cell{c, r})
			}
		}
		s.changed = true
	case ui.KeyEscape:
		s.anchor = s.cursor
	default:
		return false
	}
	return true
}

// tsv writes the values of the range chosen, a line a row and a tab
// between cells.
func (s *State) tsv(sh *sheet) string {
	lo, hi := s.span()
	var b strings.Builder
	for r := lo.Row; r <= hi.Row; r++ {
		for c := lo.Col; c <= hi.Col; c++ {
			if c > lo.Col {
				b.WriteByte('\t')
			}
			b.WriteString(sh.get(Cell{c, r}).String())
		}
		b.WriteByte('\n')
	}
	return b.String()
}

// paste puts rows of tab-separated cells from the top left of the range
// chosen, and chooses what they cover.
func (s *State) paste(cs map[Cell]string, text string, cols, rows int) {
	lo, _ := s.span()
	lines := strings.Split(strings.TrimRight(strings.ReplaceAll(text, "\r\n", "\n"), "\n"), "\n")
	end := lo
	for i, line := range lines {
		for j, field := range strings.Split(line, "\t") {
			cell := Cell{lo.Col + j, lo.Row + i}
			if cell.Col >= cols || cell.Row >= rows {
				continue
			}
			if field == "" {
				delete(cs, cell)
			} else {
				cs[cell] = field
			}
			end = Cell{max(end.Col, cell.Col), max(end.Row, cell.Row)}
		}
	}
	s.anchor, s.cursor, s.changed = lo, end, true
}
