// Package matrix shows a square matrix as a grid of tinted cells, as
// machine learning tools show a classifier's confusion matrix and the
// correlations of a dataset's columns: the names down the side and,
// slanted, across the top, each cell with its value, deeper for more.
//
// Confusion counts what a classifier predicted, a row for each class an
// example is and a column for each it was predicted as. The cells it got
// right, down the diagonal, are blue, and those it got wrong red, each
// as deep as the most of its kind; recall ends each row and precision
// each column, with the accuracy in the corner. Above it, it shows the
// counts, or each row's share of its class, or each column's of its
// prediction.
//
//	matrix.Confusion(c, &app.matrix, classes, app.counts)
//	if actual, predicted, ok := app.matrix.Chosen(); ok {
//		... // show the examples of actual predicted as predicted
//	}
//
// Correlation shows coefficients from -1 to 1, blue for positive and red
// for negative.
//
// The pointer on a cell lights its row and column and tells of it; a
// click chooses it, which Chosen returns, and a click on it again lets
// it go.
package matrix

import (
	"fmt"
	"math"
	"strconv"

	"github.com/egoist/mygo/ui"
)

// State is what a view keeps: what the cells show and the cell chosen.
// Keep it in the app's state, one for each view.
type State struct {
	mode   int
	chosen [2]int // the row and column chosen, plus 1, so that 0 is none
}

// Chosen returns the row and column of the cell chosen, or false with
// none.
func (s *State) Chosen() (row, col int, ok bool) {
	return s.chosen[0] - 1, s.chosen[1] - 1, s.chosen[0] > 0
}

// The ways Confusion shows its counts.
const (
	asCounts = iota
	byActual
	byPredicted
)

// grid is a matrix as a view shows it: its names, and for each cell the
// text, color and tip; margins are the texts after each row and below
// each column, and corner what goes where they meet, empty for none.
type grid struct {
	names   []string
	text    func(i, j int) string
	fill    func(i, j int) ui.Color
	tip     func(i, j int) string
	right   []string
	bottom  []string
	corner  string
	rightOf string // what the texts after each row are
	below   string // and those below each column
}

// Confusion creates the confusion matrix of counts, counts[i][j] the
// examples of class i predicted as class j, of the classes named.
func Confusion(c *ui.Context, s *State, classes []string, counts [][]int) ui.Element {
	t := c.Theme()
	n := len(classes)
	at := func(i, j int) int {
		if i < len(counts) && j < len(counts[i]) {
			return counts[i][j]
		}
		return 0
	}
	rows, cols := make([]int, n), make([]int, n)
	total, right := 0, 0
	for i := range n {
		for j := range n {
			rows[i] += at(i, j)
			cols[j] += at(i, j)
			total += at(i, j)
		}
		right += at(i, i)
	}
	share := func(i, j int) float64 {
		switch s.mode {
		case byActual:
			return ratio(at(i, j), rows[i])
		case byPredicted:
			return ratio(at(i, j), cols[j])
		}
		return float64(at(i, j))
	}
	var hit, miss float64
	for i := range n {
		for j := range n {
			if i == j {
				hit = max(hit, share(i, j))
			} else {
				miss = max(miss, share(i, j))
			}
		}
	}
	g := grid{
		names: classes,
		text: func(i, j int) string {
			if s.mode == asCounts {
				return strconv.Itoa(at(i, j))
			}
			return percent(share(i, j))
		},
		fill: func(i, j int) ui.Color {
			if i == j {
				return tint(t, 250, ratioF(share(i, j), hit))
			}
			return tint(t, 25, ratioF(share(i, j), miss))
		},
		tip: func(i, j int) string {
			return fmt.Sprintf("%s predicted as %s: %d, %s of %s, %s of %s",
				classes[i], classes[j], at(i, j), percent(ratio(at(i, j), rows[i])), classes[i],
				percent(ratio(at(i, j), cols[j])), classes[j])
		},
		corner:  percent(ratio(right, total)),
		rightOf: "Recall",
		below:   "Precision",
	}
	for i := range n {
		g.right = append(g.right, percent(ratio(at(i, i), rows[i])))
		g.bottom = append(g.bottom, percent(ratio(at(i, i), cols[i])))
	}
	return ui.Column(c).Gap(t.Space(3)).Children(func() {
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			ui.Segmented(c, &s.mode, "Counts", "By actual", "By predicted").Label("Show")
			ui.Text(c, fmt.Sprintf("Accuracy %s of %d", percent(ratio(right, total)), total)).
				FontSize(t.FontSize - 1).TextColor(t.TextMuted).FontFeatures("tnum")
		})
		view(c, s, g, "Actual", "Predicted")
	})
}

// Correlation creates the matrix of coefficients r, r[i][j] that of the
// columns named names[i] and names[j], from -1 to 1.
func Correlation(c *ui.Context, s *State, names []string, r [][]float64) ui.Element {
	t := c.Theme()
	at := func(i, j int) float64 {
		if i < len(r) && j < len(r[i]) {
			return r[i][j]
		}
		return math.NaN()
	}
	return view(c, s, grid{
		names: names,
		text: func(i, j int) string {
			if math.IsNaN(at(i, j)) {
				return ""
			}
			return strconv.FormatFloat(at(i, j), 'f', 2, 64)
		},
		fill: func(i, j int) ui.Color {
			v := at(i, j)
			switch {
			case math.IsNaN(v):
				return ui.Color{}
			case v < 0:
				return tint(t, 25, float32(min(-v, 1)))
			}
			return tint(t, 250, float32(min(v, 1)))
		},
		tip: func(i, j int) string {
			return fmt.Sprintf("%s and %s: %.3f", names[i], names[j], at(i, j))
		},
	}, "", "")
}

// view creates the grid of g, its rows and columns titled down and
// across, or not when empty.
func view(c *ui.Context, s *State, g grid, down, across string) ui.Element {
	t := c.Theme()
	n := len(g.names)
	small := t.FontSize - 1
	nameW := float32(0)
	for _, name := range g.names {
		w, _ := c.MeasureText(0, ui.Span{Text: name, Size: small})
		nameW = max(nameW, w)
	}
	nameW = min(nameW, t.Space(40))
	slant := nameW*0.71 + t.Space(2)
	cell := t.Space(14)
	if n > 8 {
		cell = t.Space(10)
	}
	left := nameW + t.Space(3)
	top := slant
	if across != "" {
		top += small * 1.6
	}
	extraW, extraH := float32(0), float32(0)
	if len(g.right) > 0 {
		extraW = t.Space(16)
	}
	if len(g.bottom) > 0 {
		extraH = cell * 0.7
	}
	box := ui.Box(c).Size(left+float32(n)*cell+extraW+t.Space(2), top+float32(n)*cell+extraH+slantPad(across, small)).
		Role(ui.RoleImage).Label("Matrix of " + strconv.Itoa(n) + " by " + strconv.Itoa(n))
	hot := [2]int{-1, -1}
	if x, y, ok := box.PointerPosition(); ok && box.Hovered() {
		i, j := int((y-top)/cell), int((x-left)/cell)
		if y >= top && x >= left && i < n && j < n {
			hot = [2]int{i, j}
			box.Tooltip(g.tip(i, j))
			box.Cursor(ui.CursorPointer)
		}
	}
	if box.Clicked() && hot[0] >= 0 {
		pick := [2]int{hot[0] + 1, hot[1] + 1}
		if s.chosen == pick {
			s.chosen = [2]int{}
		} else {
			s.chosen = pick
		}
	}
	chosen := s.chosen
	box.Draw(func(p *ui.Painter, r ui.Rect) {
		x0, y0 := r.X+left, r.Y+top
		if across != "" {
			p.Text(x0, r.Y, across+" →", small, t.TextMuted)
		}
		if down != "" {
			p.Text(r.X, r.Y+top-small*1.6, down+" ↓", small, t.TextMuted)
		}
		for i, name := range g.names {
			name = clip(p, name, nameW, small)
			w, h := p.MeasureText(0, ui.Span{Text: name, Size: small})
			color := t.TextMuted
			if i == hot[0] || i == chosen[0]-1 {
				color = t.Text
			}
			p.Text(x0-t.Space(2)-w, y0+float32(i)*cell+(cell-h)/2, name, small, color)
			color = t.TextMuted
			if i == hot[1] || i == chosen[1]-1 {
				color = t.Text
			}
			cx := x0 + float32(i)*cell + cell/2
			p.Transform(ui.Rotation(-45), cx, y0-t.Space(1.5), func() {
				p.Text(cx, y0-t.Space(1.5)-h/2, name, small, color)
			})
		}
		for i := range n {
			for j := range n {
				rc := ui.Rect{X: x0 + float32(j)*cell, Y: y0 + float32(i)*cell, W: cell, H: cell}
				fill := g.fill(i, j)
				inner := ui.Rect{X: rc.X + 1, Y: rc.Y + 1, W: rc.W - 2, H: rc.H - 2}
				if fill == (ui.Color{}) {
					p.Fill(inner, t.Text.Alpha(0.04), 2)
				} else {
					p.Fill(inner, fill, 2)
				}
				if hot[0] >= 0 && (i == hot[0] || j == hot[1]) && !(i == hot[0] && j == hot[1]) {
					p.Fill(inner, t.Text.Alpha(0.05), 2)
				}
				text := g.text(i, j)
				w, h := p.MeasureText(0, ui.Span{Text: text, Size: small, Features: "tnum"})
				if w < cell-4 {
					p.RichText(rc.X+(cell-w)/2, rc.Y+(cell-h)/2, 0, ui.Span{Text: text, Size: small, Features: "tnum", Color: ink(t, fill)})
				}
				if [2]int{i + 1, j + 1} == chosen || [2]int{i, j} == hot {
					p.Stroke(inner, t.Text.Alpha(0.75), 2, 1.5)
				}
			}
		}
		if len(g.right) > 0 {
			p.Text(x0+float32(n)*cell+t.Space(2), y0-small*1.4, g.rightOf, small-1, t.TextMuted)
			for i, v := range g.right {
				_, h := p.MeasureText(0, ui.Span{Text: v, Size: small})
				p.RichText(x0+float32(n)*cell+t.Space(2), y0+float32(i)*cell+(cell-h)/2, 0, ui.Span{Text: v, Size: small, Features: "tnum", Color: t.TextMuted})
			}
		}
		if len(g.bottom) > 0 {
			by := y0 + float32(n)*cell + t.Space(1)
			w, _ := p.MeasureText(0, ui.Span{Text: g.below, Size: small - 1})
			p.Text(x0-t.Space(2)-w, by, g.below, small-1, t.TextMuted)
			for j, v := range g.bottom {
				w, _ := p.MeasureText(0, ui.Span{Text: v, Size: small})
				p.RichText(x0+float32(j)*cell+(cell-w)/2, by, 0, ui.Span{Text: v, Size: small, Features: "tnum", Color: t.TextMuted})
			}
			if g.corner != "" {
				p.RichText(x0+float32(n)*cell+t.Space(2), by, 0, ui.Span{Text: g.corner, Size: small, Weight: 600, Features: "tnum"})
			}
		}
	})
	return box
}

// slantPad returns room left below the grid when it has a title across.
func slantPad(across string, size float32) float32 {
	if across == "" {
		return 0
	}
	return size * 0.4
}

// clip returns name cut short with an ellipsis to fit w.
func clip(p *ui.Painter, name string, w, size float32) string {
	if tw, _ := p.MeasureText(0, ui.Span{Text: name, Size: size}); tw <= w {
		return name
	}
	runes := []rune(name)
	for len(runes) > 1 {
		runes = runes[:len(runes)-1]
		if tw, _ := p.MeasureText(0, ui.Span{Text: string(runes) + "…", Size: size}); tw <= w {
			break
		}
	}
	return string(runes) + "…"
}

// tint returns the color of a cell of hue, f from 0, pale, to 1, deep.
func tint(t *ui.Theme, hue, f float32) ui.Color {
	f = max(0, min(f, 1))
	if t.Dark {
		return ui.Oklch(0.27+0.35*f, 0.02+0.14*f, hue)
	}
	return ui.Oklch(0.97-0.45*f, 0.01+0.15*f, hue)
}

// ink returns the color of text on fill.
func ink(t *ui.Theme, fill ui.Color) ui.Color {
	if fill == (ui.Color{}) || t.Dark {
		return t.Text
	}
	// Light fills are pale until deep enough for white to read better.
	if luma(fill) < 0.45 {
		return ui.RGB(255, 255, 255)
	}
	return t.Text
}

// luma returns the relative luminance of col, from 0 to 1.
func luma(col ui.Color) float32 {
	return (0.2126*float32(col.R) + 0.7152*float32(col.G) + 0.0722*float32(col.B)) / 255
}

func ratio(a, b int) float64 {
	if b == 0 {
		return 0
	}
	return float64(a) / float64(b)
}

func ratioF(a, b float64) float32 {
	if b == 0 {
		return 0
	}
	return float32(a / b)
}

// percent writes f, from 0 to 1, as a percentage: "87.5%".
func percent(f float64) string {
	return strconv.FormatFloat(math.Round(f*1000)/10, 'f', -1, 64) + "%"
}
