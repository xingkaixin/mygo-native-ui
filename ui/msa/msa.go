// Package msa shows a multiple sequence alignment, as sequence viewers
// do: the names down the left, and the aligned residues in a grid that
// scrolls sideways, under a ruler of positions and over the consensus,
// with a bar for how many sequences share it at each column. Residues take
// the Clustal colors for proteins, and a color a base for DNA and RNA;
// Identity shades only those that match the consensus, darker the more
// sequences do. Pointing at a residue tells its sequence, its column and
// its place in the sequence without gaps. A click chooses a column, ←
// and → move it, and Escape lets it go.
//
//	msa.View(c, &app.viewer, app.alignment)
package msa

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
)

// Sequence is a sequence of an alignment, its residues with - for gaps.
type Sequence struct {
	Name     string
	Residues string
}

// State is what a viewer keeps: the coloring and the column chosen. Keep
// it in the app's state, one for each viewer.
type State struct {
	coloring int
	// column is the column chosen plus 1, so that the zero State has none.
	column int
	hover  [2]int
}

var (
	protein = map[byte]ui.Color{}
	bases   = map[byte]ui.Color{
		'A': ui.Hex("#5CB85C"), 'C': ui.Hex("#3E7CC4"), 'G': ui.Hex("#F0AD4E"), 'T': ui.Hex("#D9534F"), 'U': ui.Hex("#D9534F"),
	}
)

func init() {
	for _, g := range []struct {
		residues string
		color    string
	}{
		{"AILMFWV", "#80A0F0"}, {"KR", "#F01505"}, {"ED", "#C048C0"}, {"NQST", "#15C015"},
		{"C", "#F08080"}, {"G", "#F09048"}, {"P", "#C0C000"}, {"HY", "#15A4A4"},
	} {
		for i := range len(g.residues) {
			protein[g.residues[i]] = ui.Hex(g.color)
		}
	}
}

// consensus returns the commonest residue of each column and the share of
// sequences that have it.
func consensus(seqs []Sequence, n int) ([]byte, []float32) {
	top := make([]byte, n)
	share := make([]float32, n)
	for col := range n {
		counts := map[byte]int{}
		for _, s := range seqs {
			if col < len(s.Residues) && s.Residues[col] != '-' && s.Residues[col] != '.' {
				counts[s.Residues[col]]++
			}
		}
		best, most := byte('-'), 0
		for r, k := range counts {
			if k > most || k == most && r < best {
				best, most = r, k
			}
		}
		top[col], share[col] = best, float32(most)/float32(max(1, len(seqs)))
	}
	return top, share
}

// View creates the viewer of the alignment seqs.
func View(c *ui.Context, s *State, seqs []Sequence) ui.Element {
	t := c.Theme()
	n := 0
	nucleic := true
	for _, sq := range seqs {
		n = max(n, len(sq.Residues))
		for i := range len(sq.Residues) {
			if r := sq.Residues[i]; r != '-' && r != '.' && !strings.ContainsRune("ACGTUN", rune(r&^0x20)) {
				nucleic = false
			}
		}
	}
	top, share := consensus(seqs, n)
	cw, rh := float32(11), float32(17)
	ruler, foot := float32(20), rh+t.Space(9)
	root := ui.Column(c).Gap(t.Space(3)).Focusable().FocusRing(false).Label("Alignment")
	switch {
	case root.Shortcut(0, ui.KeyLeft) && s.column > 1:
		s.column--
	case root.Shortcut(0, ui.KeyRight) && s.column > 0 && s.column < n:
		s.column++
	case root.Shortcut(0, ui.KeyEscape):
		s.column = 0
	}
	hover := s.hover
	s.hover = [2]int{-1, -1}
	root.Children(func() {
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			ui.Segmented(c, &s.coloring, "Residue", "Identity", "None").Label("Coloring")
			ui.Box(c).Grow(1)
			ui.Text(c, fmt.Sprintf("%d sequences · %d columns", len(seqs), n)).FontSize(t.FontSize - 1).TextColor(t.TextMuted)
		})
		ui.Row(c).FillWidth().Radius(t.Space(2)).Border(1, t.Border).Clip().Children(func() {
			ui.Column(c).Padding(ruler, t.Space(3), 0, t.Space(3)).Background(t.Surface).Children(func() {
				for i, sq := range seqs {
					name := ui.Text(c.Key(i), sq.Name).Height(rh).FontSize(t.FontSize - 1).Ellipsis("…").MaxWidth(t.Space(36))
					if hover[1] == i {
						name.Bold()
					}
				}
				ui.Text(c, "Consensus").Height(foot).FontSize(t.FontSize - 1).TextColor(t.TextMuted)
			})
			ui.ScrollHorizontal(c).Grow(1).Children(func() {
				grid := ui.Box(c).Size(float32(n)*cw+cw, ruler+rh*float32(len(seqs))+foot+t.Space(3)).Cursor(ui.CursorPointer).
					Role(ui.RoleImage).Label("Residues")
				px, py, over := grid.PointerPosition()
				col, row := int(px/cw), int((py-ruler)/rh)
				if over && col < n && py >= ruler && row < len(seqs) {
					s.hover = [2]int{col, row}
				}
				if grid.Clicked() && over && col < n {
					if s.column == col+1 {
						s.column = 0
					} else {
						s.column = col + 1
					}
					root.Focus()
				}
				chosen, coloring := s.column-1, s.coloring
				grid.Draw(func(p *ui.Painter, r ui.Rect) {
					vis := p.Visible()
					first := max(0, int((vis.X-r.X)/cw)-1)
					last := min(n, int((vis.X+vis.W-r.X)/cw)+2)
					for k := first; k < last; k++ {
						x := r.X + float32(k)*cw
						if (k+1)%10 == 0 {
							span := ui.Span{Text: fmt.Sprint(k + 1), Size: 10, Color: t.TextMuted}
							w, _ := p.MeasureText(0, span)
							p.RichText(x+cw/2-w/2, r.Y+3, 0, span)
							p.Fill(ui.Rect{X: x + cw/2 - 0.5, Y: r.Y + ruler - 5, W: 1, H: 4}, t.TextMuted, 0)
						} else if (k+1)%5 == 0 {
							p.Fill(ui.Rect{X: x + cw/2 - 0.5, Y: r.Y + ruler - 3, W: 1, H: 2}, t.TextMuted.Alpha(0.6), 0)
						}
						for i, sq := range seqs {
							y := r.Y + ruler + float32(i)*rh
							res := byte('-')
							if k < len(sq.Residues) {
								res = sq.Residues[k]
							}
							up := res &^ 0x20
							ink := t.Text
							if res == '-' || res == '.' {
								ink = t.TextMuted.Alpha(0.6)
							} else {
								switch {
								case coloring == 0:
									bg, ok := protein[up]
									if nucleic {
										bg, ok = bases[up]
									}
									if ok {
										p.Fill(ui.Rect{X: x, Y: y, W: cw, H: rh}, bg.Alpha(0.8), 0)
										ink = ui.RGB(20, 20, 20)
									}
								case coloring == 1 && up == top[k] && share[k] >= 0.4:
									p.Fill(ui.Rect{X: x, Y: y, W: cw, H: rh}, t.Accent.Alpha(0.2+0.6*(share[k]-0.4)/0.6), 0)
								}
							}
							span := ui.Span{Text: string(res), Size: 11, Font: "monospace", Color: ink}
							w, h := p.MeasureText(0, span)
							p.RichText(x+(cw-w)/2, y+(rh-h)/2, 0, span)
						}
						fy := r.Y + ruler + float32(len(seqs))*rh
						span := ui.Span{Text: string(top[k]), Size: 11, Weight: 700, Font: "monospace", Color: t.Text}
						w, _ := p.MeasureText(0, span)
						p.RichText(x+(cw-w)/2, fy+2, 0, span)
						bar := (foot - rh - 4) * share[k]
						p.Fill(ui.Rect{X: x + 1, Y: fy + foot - 2 - bar, W: cw - 2, H: bar}, t.Accent.Alpha(0.3+0.5*share[k]), 1)
					}
					if chosen >= 0 {
						p.Stroke(ui.Rect{X: r.X + float32(chosen)*cw, Y: r.Y + ruler, W: cw, H: r.H - ruler}, t.Accent, 2, 2)
					}
					if hover[0] >= 0 {
						p.Stroke(ui.Rect{X: r.X + float32(hover[0])*cw, Y: r.Y + ruler + float32(hover[1])*rh, W: cw, H: rh}, t.Text, 1, 1.5)
					}
				})
			})
		})
		info := " "
		focus := hover
		if focus[0] < 0 && s.column > 0 {
			focus = [2]int{s.column - 1, -1}
		}
		switch {
		case focus[0] >= 0 && focus[1] >= 0:
			sq := seqs[focus[1]]
			res := sq.Residues[min(focus[0], len(sq.Residues)-1)]
			place := 0
			for i := 0; i <= focus[0] && i < len(sq.Residues); i++ {
				if sq.Residues[i] != '-' && sq.Residues[i] != '.' {
					place++
				}
			}
			info = fmt.Sprintf("%s · column %d · %c", sq.Name, focus[0]+1, res)
			if res != '-' && res != '.' {
				info += fmt.Sprintf(" %d", place)
			}
		case focus[0] >= 0:
			info = fmt.Sprintf("Column %d · consensus %c in %.0f%%", focus[0]+1, top[focus[0]], share[focus[0]]*100)
		}
		ui.Text(c, info).FontSize(t.FontSize - 1).FontFeatures("tnum").TextColor(t.TextMuted)
	})
	if hover != s.hover {
		c.AnimationFrame()
	}
	return root
}
