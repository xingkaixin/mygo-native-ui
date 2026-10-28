// Package punnett crosses two parents in a Punnett square: their
// genotypes typed as letter pairs, as AaBb, for up to three genes, each
// letter a gene, capital for the dominant allele. The square crosses
// every gamete of one parent with every one of the other; each child is
// tinted by its phenotype, and the phenotypes and genotypes are counted
// beside it as ratios. Pointing at a child marks every child of its
// genotype and its phenotype in the counts.
//
//	punnett.View(c, &app.square, &app.mother, &app.father)
package punnett

import (
	"cmp"
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode"

	"github.com/egoist/mygo/ui"
)

// Parse reads a genotype, as AaBb, into its genes' allele pairs, each
// with the dominant allele first.
func Parse(s string) ([][2]rune, error) {
	r := []rune(strings.ReplaceAll(strings.TrimSpace(s), " ", ""))
	if len(r) == 0 {
		return nil, errors.New("type a genotype, as AaBb")
	}
	if len(r)%2 != 0 {
		return nil, errors.New("each gene takes two letters")
	}
	var out [][2]rune
	for i := 0; i < len(r); i += 2 {
		a, b := r[i], r[i+1]
		if !unicode.IsLetter(a) || unicode.ToLower(a) != unicode.ToLower(b) {
			return nil, fmt.Errorf("%c%c is not a pair of one gene", a, b)
		}
		if unicode.IsLower(a) && unicode.IsUpper(b) {
			a, b = b, a
		}
		for _, g := range out {
			if unicode.ToLower(g[0]) == unicode.ToLower(a) {
				return nil, fmt.Errorf("%c is there twice", unicode.ToUpper(a))
			}
		}
		out = append(out, [2]rune{a, b})
	}
	if len(out) > 3 {
		return nil, errors.New("three genes at most")
	}
	return out, nil
}

// Gametes returns the gametes of a genotype: one allele of each gene, in
// every way.
func Gametes(genes [][2]rune) []string {
	out := []string{""}
	for _, g := range genes {
		var next []string
		for _, prefix := range out {
			next = append(next, prefix+string(g[0]), prefix+string(g[1]))
		}
		out = next
	}
	return out
}

// child returns the genotype from two gametes, dominant first in each
// gene.
func child(a, b string) string {
	ra, rb := []rune(a), []rune(b)
	var s strings.Builder
	for i := range ra {
		x, y := ra[i], rb[i]
		if unicode.IsLower(x) && unicode.IsUpper(y) {
			x, y = y, x
		}
		s.WriteRune(x)
		s.WriteRune(y)
	}
	return s.String()
}

// Phenotype returns what a genotype shows, gene by gene: A_ for one with
// a dominant allele, aa for one without.
func Phenotype(genotype string) string {
	r := []rune(genotype)
	var s strings.Builder
	for i := 0; i < len(r); i += 2 {
		if unicode.IsUpper(r[i]) {
			s.WriteString(string(unicode.ToUpper(r[i])) + "_")
		} else {
			s.WriteString(string(r[i]) + string(r[i+1]))
		}
	}
	return s.String()
}

// State is what a square keeps: the child pointed at. Keep it in the
// app's state, one for each square.
type State struct {
	hovered string
}

var hues = []ui.Color{
	ui.Hex("#3E7BFA"), ui.Hex("#30A46C"), ui.Hex("#F5A524"), ui.Hex("#E5484D"),
	ui.Hex("#8E4EC6"), ui.Hex("#12A594"), ui.Hex("#F07A2B"), ui.Hex("#8D8D93"),
}

type tally struct {
	key   string
	count int
}

func count(items []string) []tally {
	m := map[string]int{}
	for _, it := range items {
		m[it]++
	}
	var out []tally
	for k, n := range m {
		out = append(out, tally{k, n})
	}
	slices.SortFunc(out, func(a, b tally) int {
		return cmp.Or(b.count-a.count, strings.Compare(a.key, b.key))
	})
	return out
}

// View creates the square of the parents' genotypes *a and *b.
func View(c *ui.Context, s *State, a, b *string) ui.Element {
	t := c.Theme()
	ga, errA := Parse(*a)
	gb, errB := Parse(*b)
	err := cmp.Or(errA, errB)
	if err == nil {
		same := len(ga) == len(gb)
		for i := range ga {
			same = same && i < len(gb) && unicode.ToLower(ga[i][0]) == unicode.ToLower(gb[i][0])
		}
		if !same {
			err = errors.New("both parents need the same genes in the same order")
		}
	}
	changed := false
	root := ui.Column(c).Gap(t.Space(6))
	root.Children(func() {
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.End).Children(func() {
			parent := func(label string, v *string, e error) {
				f := ui.Field(c, label, func() {
					if ui.TextInput(c, v).Font("monospace").Width(t.Space(32)).Changed() {
						changed = true
					}
				})
				if e != nil && *v != "" {
					f.Error(e.Error())
				}
			}
			parent("Parent 1", a, errA)
			ui.Text(c, "×").FontSize(t.FontSize+6).TextColor(t.TextMuted).Padding(0, 0, t.Space(1.5))
			parent("Parent 2", b, errB)
		})
		if err != nil {
			if errA == nil && errB == nil {
				ui.Text(c, err.Error()).TextColor(t.Danger)
			}
			return
		}
		rows, cols := Gametes(ga), Gametes(gb)
		var children, phenos []string
		for _, r := range rows {
			for _, col := range cols {
				g := child(r, col)
				children = append(children, g)
				phenos = append(phenos, Phenotype(g))
			}
		}
		phenoTally, genoTally := count(phenos), count(children)
		hue := map[string]ui.Color{}
		for i, p := range phenoTally {
			hue[p.key] = hues[i%len(hues)]
		}
		hovered := ""
		size := []float32{0, t.Space(20), t.Space(14), t.Space(10)}[len(ga)]
		ui.Row(c).Gap(t.Space(8)).AlignItems(ui.Start).Children(func() {
			ui.Grid(c).Columns(len(cols) + 1).Gap(2).Children(func() {
				ui.Box(c.Key("corner"))
				for i, g := range cols {
					ui.Text(c.Key([2]any{"col", i}), g).Font("monospace").Bold().TextColor(t.TextMuted).AlignSelf(ui.Center).JustifySelf(ui.Center)
				}
				for i, r := range rows {
					ui.Text(c.Key([2]any{"row", i}), r).Font("monospace").Bold().TextColor(t.TextMuted).AlignSelf(ui.Center).Padding(0, t.Space(2), 0, 0)
					for j := range cols {
						g := children[i*len(cols)+j]
						col := hue[Phenotype(g)]
						cell := ui.Box(c.Key([2]any{i, j})).Size(size, size).Radius(4).Center().Label(g).Background(col.Alpha(0.18))
						if cell.Hovered() {
							hovered = g
						}
						// Drawn as painted, after the whole square is built,
						// so that every cell sees the one pointed at.
						cell.Draw(func(p *ui.Painter, r ui.Rect) {
							if s.hovered == g {
								p.Fill(r, col.Alpha(0.3), 4)
								p.Stroke(r, col, 4, 2)
							}
						})
						cell.Children(func() {
							ui.Text(c, g).Font("monospace").FontSize(min(t.FontSize+2, size*0.26))
						})
					}
				}
			})
			s.hovered = hovered
			ui.Column(c).Gap(t.Space(5)).Children(func() {
				list := func(title string, items []tally, color func(string) ui.Color, marked string) {
					ui.Column(c.Key(title)).Gap(t.Space(1)).Children(func() {
						var ratio []string
						for _, it := range items {
							ratio = append(ratio, fmt.Sprint(it.count))
						}
						ui.Text(c, title+"  "+strings.Join(ratio, " : ")).Bold()
						for _, it := range items {
							row := ui.Row(c.Key(it.key)).Gap(t.Space(2)).AlignItems(ui.Center).Padding(t.Space(0.5), t.Space(1.5)).Radius(t.Radius)
							if it.key == marked {
								row.Background(t.SurfaceHover)
							}
							row.Children(func() {
								ui.Box(c).Size(10, 10).Radius(3).Background(color(it.key))
								ui.Text(c, it.key).Font("monospace").Width(t.Space(16))
								ui.Text(c, fmt.Sprintf("%d/%d", it.count, len(children))).FontFeatures("tnum").TextColor(t.TextMuted).Width(t.Space(12))
								ui.Text(c, fmt.Sprintf("%.1f%%", 100*float64(it.count)/float64(len(children)))).FontFeatures("tnum").TextColor(t.TextMuted)
							})
						}
					})
				}
				marked := ""
				if s.hovered != "" {
					marked = Phenotype(s.hovered)
				}
				list("Phenotypes", phenoTally, func(k string) ui.Color { return hue[k] }, marked)
				list("Genotypes", genoTally, func(k string) ui.Color { return hue[Phenotype(k)].Alpha(0.5) }, s.hovered)
			})
		})
	})
	if changed {
		root.SetChanged()
	}
	return root
}
