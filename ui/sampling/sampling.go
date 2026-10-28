// Package sampling shows what a language model's sampling settings do to
// the next token, as a playground's settings panel would if it showed
// their effect: sliders for temperature, top-k, top-p and min-p over the
// candidates for the next token, each a row with its token, a bar of its
// probability at the temperature and over it a bar of its share once the
// others are cut, and both as percentages. The settings apply in the
// order most libraries apply them: temperature, top-k, top-p, then
// min-p. A candidate cut is dimmed and tagged with the setting that cut
// it. Above the rows, how many candidates are left and the entropy of
// what is left, in bits. A temperature of 0 keeps only the likeliest.
//
//	if sampling.View(c, &app.params, app.candidates).Changed() {
//		app.save()
//	}
package sampling

import (
	"cmp"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
)

// Params are the sampling settings. A TopK of 0, a TopP of 1 and a MinP
// of 0 leave every candidate.
type Params struct {
	Temperature float64
	TopK        int
	TopP        float64
	MinP        float64
}

// Candidate is a token the model could write next and its logit.
type Candidate struct {
	Token string
	Logit float64
}

// Cut is the setting that left a candidate out, or Kept.
type Cut uint8

const (
	Kept Cut = iota
	// CutTemperature is a candidate other than the likeliest at a
	// temperature of 0, which samples greedily.
	CutTemperature
	CutTopK
	CutTopP
	CutMinP
)

func (k Cut) String() string {
	return [...]string{"", "greedy", "top-k", "top-p", "min-p"}[k]
}

// Row is a candidate after sampling: its token, its probability at the
// temperature, its share of what is kept, and what cut it.
type Row struct {
	Token string
	Prob  float64
	Final float64
	Cut   Cut
}

// Apply returns the candidates sampled with p, the likeliest first.
func Apply(p Params, cands []Candidate) []Row {
	rows := make([]Row, len(cands))
	order := make([]Candidate, len(cands))
	copy(order, cands)
	slices.SortStableFunc(order, func(a, b Candidate) int { return cmp.Compare(b.Logit, a.Logit) })
	if len(order) == 0 {
		return rows
	}
	temp := max(p.Temperature, 1e-3)
	peak := order[0].Logit / temp
	sum := 0.0
	for i, c := range order {
		rows[i] = Row{Token: c.Token, Prob: math.Exp(c.Logit/temp - peak)}
		sum += rows[i].Prob
	}
	for i := range rows {
		rows[i].Prob /= sum
	}
	for i := range rows {
		switch {
		case p.Temperature <= 0 && i > 0:
			rows[i].Cut = CutTemperature
		case p.TopK > 0 && i >= p.TopK:
			rows[i].Cut = CutTopK
		}
	}
	kept := 0.0
	for _, r := range rows {
		if r.Cut == Kept {
			kept += r.Prob
		}
	}
	cum := 0.0
	for i, r := range rows {
		if r.Cut != Kept {
			continue
		}
		if p.TopP < 1 && cum >= p.TopP*kept {
			rows[i].Cut = CutTopP
			continue
		}
		cum += r.Prob
	}
	for i, r := range rows {
		if r.Cut == Kept && i > 0 && r.Prob < p.MinP*rows[0].Prob {
			rows[i].Cut = CutMinP
		}
	}
	kept = 0
	for _, r := range rows {
		if r.Cut == Kept {
			kept += r.Prob
		}
	}
	for i, r := range rows {
		if r.Cut == Kept {
			rows[i].Final = r.Prob / kept
		}
	}
	return rows
}

// View creates the sliders of *p and the rows of cands sampled with it.
// Changed reports a change to *p.
func View(c *ui.Context, p *Params, cands []Candidate) ui.Element {
	t := c.Theme()
	before := *p
	root := ui.Column(c).Gap(t.Space(4))
	root.Children(func() {
		ui.Grid(c).FillWidth().ColumnTracks(ui.FitContent(), ui.Fr(1), ui.Fixed(t.Space(12))).GapX(t.Space(3)).GapY(t.Space(2)).
			AlignItems(ui.Center).Children(func() {
			slider(c, "Temperature", &p.Temperature, 0, 2, 0.05, strconv.FormatFloat(p.Temperature, 'f', 2, 64))
			k := float64(p.TopK)
			text := strconv.Itoa(p.TopK)
			if p.TopK == 0 {
				text = "off"
			}
			slider(c, "Top-k", &k, 0, float64(max(len(cands), 1)), 1, text)
			p.TopK = int(k)
			slider(c, "Top-p", &p.TopP, 0.05, 1, 0.01, strconv.FormatFloat(p.TopP, 'f', 2, 64))
			slider(c, "Min-p", &p.MinP, 0, 0.5, 0.01, strconv.FormatFloat(p.MinP, 'f', 2, 64))
		})
		rows := Apply(*p, cands)
		left, entropy := 0, 0.0
		for _, r := range rows {
			if r.Cut == Kept {
				left++
				entropy -= r.Final * math.Log2(r.Final)
			}
		}
		ui.Row(c).Gap(t.Space(4)).Children(func() {
			ui.Text(c, fmt.Sprintf("%d of %d tokens left", left, len(rows))).FontWeight(600).FontFeatures("tnum")
			ui.Text(c, fmt.Sprintf("Entropy %.2f bits", entropy)).TextColor(t.TextMuted).FontFeatures("tnum")
		})
		bars(c, rows)
	})
	if *p != before {
		root.SetChanged()
	}
	return root
}

// slider creates a labeled slider of *v from lo to hi, rounded to step,
// with text beside it.
func slider(c *ui.Context, label string, v *float64, lo, hi, step float64, text string) {
	t := c.Theme()
	ui.Text(c, label)
	ui.Slider(c, v, lo, hi).Label(label)
	*v = math.Round(*v/step) * step
	ui.Text(c, text).FontFeatures("tnum").TextColor(t.TextMuted).TextAlign(ui.End)
}

// bars creates a row for each of rows.
func bars(c *ui.Context, rows []Row) {
	t := c.Theme()
	small := t.FontSize - 1
	tokW := t.Space(24)
	for _, r := range rows {
		w, _ := c.MeasureText(0, ui.Span{Text: visible(r.Token), Size: small, Font: "monospace"})
		tokW = max(tokW, w+t.Space(3))
	}
	ui.Column(c).Gap(t.Space(0.5)).Role(ui.RoleList).Label("Candidates").Children(func() {
		for i, r := range rows {
			row := ui.Row(c.Key(i)).Gap(t.Space(3)).AlignItems(ui.Center).Height(t.Space(6.5))
			if r.Cut != Kept {
				row.Opacity(0.5)
			}
			label := fmt.Sprintf("%s %s", visible(r.Token), percent(r.Prob))
			if r.Cut == Kept {
				label += ", " + percent(r.Final) + " after sampling"
			} else {
				label += ", cut by " + r.Cut.String()
			}
			row.Role(ui.RoleListItem).Label(label).Children(func() {
				ui.Text(c, visible(r.Token)).Font("monospace").FontSize(small).Width(tokW).SingleLine().
					Background(t.Text.Alpha(0.06)).Padding(0, t.Space(1.5)).Radius(t.Radius)
				ui.Box(c).Grow(1).Height(t.Space(3.5)).Draw(func(p *ui.Painter, b ui.Rect) {
					p.Fill(b, t.Text.Alpha(0.05), 3)
					p.Fill(ui.Rect{X: b.X, Y: b.Y, W: b.W * float32(r.Prob), H: b.H}, t.Text.Alpha(0.18), 3)
					if r.Cut == Kept {
						p.Fill(ui.Rect{X: b.X, Y: b.Y + b.H*0.25, W: b.W * float32(r.Final), H: b.H * 0.5}, t.Accent, 2)
					}
				})
				ui.Text(c, percent(r.Prob)).FontSize(small).TextColor(t.TextMuted).FontFeatures("tnum").
					Width(t.Space(14)).TextAlign(ui.End)
				if r.Cut == Kept {
					ui.Text(c, percent(r.Final)).FontSize(small).FontWeight(600).FontFeatures("tnum").
						Width(t.Space(16)).TextAlign(ui.End)
				} else {
					ui.Text(c, r.Cut.String()).FontSize(t.FontSize - 3).FontWeight(600).TextColor(t.TextMuted).
						Width(t.Space(16)).TextAlign(ui.End)
				}
			})
		}
	})
}

// visible shows the spaces and line breaks of a token.
func visible(tok string) string {
	return strings.NewReplacer(" ", "␣", "\n", "↵", "\t", "⇥").Replace(tok)
}

func percent(v float64) string {
	switch {
	case v >= 0.1:
		return fmt.Sprintf("%.1f%%", v*100)
	case v >= 0.001:
		return fmt.Sprintf("%.2f%%", v*100)
	}
	return "<0.1%"
}
