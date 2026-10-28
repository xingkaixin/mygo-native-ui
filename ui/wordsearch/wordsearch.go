// Package wordsearch is a word search: a grid of letters hiding words
// across, down and diagonally, forward or backward. Dragging from a
// letter draws a band along the nearest of the eight directions; letting
// go on a word hidden there keeps the band in a color of its own and
// strikes the word through in the list beside the grid.
//
//	p, _ := wordsearch.New(words, 12, 12, seed)
//	ev := wordsearch.View(c, &app.search, p)
package wordsearch

import (
	"fmt"
	"math"
	"math/rand/v2"
	"slices"
	"strings"
	"unicode"

	"github.com/egoist/mygo/ui"
)

// Puzzle is a word search: its size, its letters row by row, and the
// words hidden in it.
type Puzzle struct {
	Rows, Cols int
	Letters    []rune
	Words      []string
}

var directions = [8][2]int{{0, 1}, {1, 0}, {1, 1}, {-1, 1}, {0, -1}, {-1, 0}, {-1, -1}, {1, -1}}

// New hides words, in capitals, in a grid of rows by cols at random from
// seed, and fills the rest with random letters. Letters other than A to
// Z are dropped from the words.
func New(words []string, rows, cols int, seed uint64) (Puzzle, error) {
	r := rand.New(rand.NewPCG(seed, seed^0x2545f4914f6cdd1d))
	p := Puzzle{Rows: rows, Cols: cols, Letters: make([]rune, rows*cols)}
	for _, w := range words {
		w = strings.Map(func(r rune) rune {
			r = unicode.ToUpper(r)
			if r < 'A' || r > 'Z' {
				return -1
			}
			return r
		}, w)
		if len(w) < 2 {
			continue
		}
		placed := false
		for try := 0; try < 500 && !placed; try++ {
			d := directions[r.IntN(8)]
			y, x := r.IntN(rows), r.IntN(cols)
			ey, ex := y+d[0]*(len(w)-1), x+d[1]*(len(w)-1)
			if ey < 0 || ey >= rows || ex < 0 || ex >= cols {
				continue
			}
			fits := true
			for k, ch := range w {
				if l := p.Letters[(y+d[0]*k)*cols+x+d[1]*k]; l != 0 && l != ch {
					fits = false
					break
				}
			}
			if !fits {
				continue
			}
			for k, ch := range w {
				p.Letters[(y+d[0]*k)*cols+x+d[1]*k] = ch
			}
			placed = true
		}
		if !placed {
			return Puzzle{}, fmt.Errorf("no room for %s", w)
		}
		p.Words = append(p.Words, w)
	}
	for i, l := range p.Letters {
		if l == 0 {
			p.Letters[i] = rune('A' + r.IntN(26))
		}
	}
	return p, nil
}

// line is a straight run of cells, from one to another.
type line struct{ from, to int }

func (p Puzzle) spell(l line) string {
	fy, fx, ty, tx := l.from/p.Cols, l.from%p.Cols, l.to/p.Cols, l.to%p.Cols
	n := max(abs(ty-fy), abs(tx-fx))
	var b strings.Builder
	for k := 0; k <= n; k++ {
		b.WriteRune(p.Letters[(fy+sign(ty-fy)*k)*p.Cols+fx+sign(tx-fx)*k])
	}
	return b.String()
}

func abs(x int) int { return max(x, -x) }

func sign(x int) int {
	switch {
	case x < 0:
		return -1
	case x > 0:
		return 1
	}
	return 0
}

// State is what a word search keeps: the words found and where, and the
// band being dragged. Keep it in the app's state, one for each puzzle.
type State struct {
	found    map[string]line
	dragging bool
	from, to int
}

// Found reports how many words have been found.
func (s *State) Found() int { return len(s.found) }

// Events is a word search, and the word found in the frame, if any.
type Events struct {
	ui.Element
	Found string
}

var hues = []ui.Color{
	ui.Hex("#F5A524"), ui.Hex("#3E9BFA"), ui.Hex("#30B158"), ui.Hex("#E5484D"),
	ui.Hex("#8E4EC6"), ui.Hex("#12A594"), ui.Hex("#F07A2B"), ui.Hex("#D6409F"),
}

// View creates the word search of p.
func View(c *ui.Context, s *State, p Puzzle) Events {
	t := c.Theme()
	var ev Events
	if s.found == nil {
		s.found = map[string]line{}
	}
	cell := t.Space(10)
	root := ui.Row(c).Gap(t.Space(8)).AlignItems(ui.Start)
	root.Children(func() {
		grid := ui.Box(c).Size(cell*float32(p.Cols), cell*float32(p.Rows)).Radius(t.Radius + 2).Background(t.Surface).Label("Letters")
		x, y, _ := grid.PointerPosition()
		col := min(max(int(x/cell), 0), p.Cols-1)
		row := min(max(int(y/cell), 0), p.Rows-1)
		if _, _, held := grid.Dragged(); held {
			if !s.dragging {
				s.dragging, s.from = true, row*p.Cols+col
			}
			// Snap to the nearest of the eight directions, as far as the
			// pointer has gone along it.
			fy, fx := s.from/p.Cols, s.from%p.Cols
			cx, cy := (float32(fx)+0.5)*cell, (float32(fy)+0.5)*cell
			angle := math.Atan2(float64(y-cy), float64(x-cx))
			k := int(math.Round(angle/(math.Pi/4))+8) % 8
			dx, dy := []int{1, 1, 0, -1, -1, -1, 0, 1}[k], []int{0, 1, 1, 1, 0, -1, -1, -1}[k]
			dist := math.Hypot(float64(x-cx), float64(y-cy)) / float64(cell)
			if dx != 0 && dy != 0 {
				dist /= math.Sqrt2
			}
			n := int(math.Round(dist))
			for n > 0 && (fy+dy*n < 0 || fy+dy*n >= p.Rows || fx+dx*n < 0 || fx+dx*n >= p.Cols) {
				n--
			}
			s.to = (fy+dy*n)*p.Cols + fx + dx*n
		} else if s.dragging {
			s.dragging = false
			l := line{s.from, s.to}
			word := []rune(p.spell(l))
			back := slices.Clone(word)
			slices.Reverse(back)
			for _, w := range p.Words {
				if _, ok := s.found[w]; !ok && l.from != l.to && (w == string(word) || w == string(back)) {
					s.found[w], ev.Found = l, w
				}
			}
		}
		var band *line
		if s.dragging {
			band = &line{s.from, s.to}
		}
		found := make(map[string]line, len(s.found))
		for w, l := range s.found {
			found[w] = l
		}
		words, dragging := p.Words, band
		grid.Draw(func(pt *ui.Painter, r ui.Rect) {
			center := func(i int) (float32, float32) {
				return r.X + (float32(i%p.Cols)+0.5)*cell, r.Y + (float32(i/p.Cols)+0.5)*cell
			}
			capsule := func(l line, col ui.Color) {
				x0, y0 := center(l.from)
				x1, y1 := center(l.to)
				var path ui.Path
				path.MoveTo(x0, y0).LineTo(x1, y1).Cap(ui.RoundCap)
				pt.StrokePath(&path, cell*0.78, col)
			}
			for i, w := range words {
				if l, ok := found[w]; ok {
					capsule(l, hues[i%len(hues)].Alpha(0.35))
				}
			}
			if dragging != nil {
				capsule(*dragging, t.Accent.Alpha(0.3))
			}
			fs := cell * 0.5
			for i, l := range p.Letters {
				cx, cy := center(i)
				w, h := pt.MeasureText(0, ui.Span{Text: string(l), Size: fs})
				pt.Text(cx-w/2, cy-h/2, string(l), fs, t.Text)
			}
		})
		ui.Column(c).Gap(t.Space(1.5)).Width(t.Space(40)).Children(func() {
			ui.Text(c, fmt.Sprintf("%d of %d found", len(s.found), len(p.Words))).Bold().Padding(0, 0, t.Space(2))
			for i, w := range p.Words {
				_, ok := s.found[w]
				ui.Row(c.Key(w)).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
					dot := ui.Box(c).Size(8, 8).Radius(4)
					label := ui.Text(c, w).Font("monospace").FontFeatures("tnum")
					if ok {
						dot.Background(hues[i%len(hues)])
						label.TextColor(t.TextMuted).Strikethrough()
					} else {
						dot.Border(1, t.Border)
					}
				})
			}
		})
	})
	if ev.Found != "" {
		root.SetChanged()
	}
	ev.Element = root
	return ev
}
