// Msa aligns ten members of a made-up protein family, grown from one
// ancestor by substitutions, deletions and an insertion some of them share.
//
//	go run ./examples/msa
package main

import (
	"log"
	"math/rand/v2"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/msa"
)

// groups are amino acids alike enough to stand in for each other.
var groups = []string{"AILMV", "FWY", "KRH", "DE", "NQ", "ST", "G", "P", "C"}

type app struct {
	viewer    msa.State
	alignment []msa.Sequence
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Padding(t.Space(6)).Background(t.Background).Children(func() {
		msa.View(c, &a.viewer, a.alignment)
	})
}

// family returns the aligned members of the family.
func family() []msa.Sequence {
	r := rand.New(rand.NewPCG(7, 11))
	common := "AAAGGLLLLVVVEEKKKRSSTTDDNQIIFPYHWMC"
	root := make([]byte, 260)
	for i := range root {
		root[i] = common[r.IntN(len(common))]
	}
	// Columns 120 to 131 are an insertion only some members carry.
	const ins, insLen = 120, 12
	names := []string{"Aquila", "Borealis", "Corvina", "Dorado", "Egretta", "Falco", "Gadus", "Hyla", "Ibis", "Junco"}
	var out []msa.Sequence
	for k, name := range names {
		seq := []byte(string(root))
		for i := range seq {
			if r.Float64() < 0.04+float64(k)*0.012 {
				for _, g := range groups {
					if strings.IndexByte(g, seq[i]) >= 0 {
						seq[i] = g[r.IntN(len(g))]
					}
				}
			}
			if r.Float64() < 0.01 {
				seq[i] = common[r.IntN(len(common))]
			}
		}
		if k < 4 {
			for i := ins; i < ins+insLen; i++ {
				seq[i] = '-'
			}
		}
		for range r.IntN(3) {
			at, n := 10+r.IntN(220), 2+r.IntN(6)
			for i := at; i < at+n && i < len(seq); i++ {
				seq[i] = '-'
			}
		}
		out = append(out, msa.Sequence{Name: name, Residues: string(seq)})
	}
	return out
}

func main() {
	a := &app{alignment: family()}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Alignment",
			Width:   1000,
			Height:  420,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
