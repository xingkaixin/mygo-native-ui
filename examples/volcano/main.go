// Volcano plots a made-up test of 6,000 genes in treated cells against
// untreated ones, a few hundred of them changed. Drag the sliders to set
// the thresholds, and point at a gene to read it.
//
//	go run ./examples/volcano
package main

import (
	"cmp"
	"fmt"
	"log"
	"math"
	"math/rand/v2"
	"slices"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/volcano"
)

var symbols = []string{
	"MYC", "FOS", "EGR1", "JUN", "ATF3", "IL6", "CXCL8", "CDKN1A", "GADD45A", "DUSP1",
	"HMOX1", "SERPINE1", "VEGFA", "KLF4", "TXNIP", "CCND1", "E2F1", "MKI67", "TOP2A", "CDC20",
}

type app struct {
	volcano volcano.State
	genes   []volcano.Gene
}

func sample() []volcano.Gene {
	r := rand.New(rand.NewPCG(11, 12))
	genes := make([]volcano.Gene, 6000)
	for i := range genes {
		fold := 0.35 * r.NormFloat64()
		p := r.Float64()
		if i < 400 {
			sign := 1.0
			if i%3 == 0 {
				sign = -1
			}
			fold = sign * (0.6 + math.Abs(1.1*r.NormFloat64()))
			p = math.Pow(10, -(0.8 + math.Abs(fold)*(0.8+2.4*r.Float64())))
		}
		genes[i] = volcano.Gene{Name: fmt.Sprintf("LOC%05d", 10000+i*7), FoldChange: fold, P: p}
	}
	// The most significant genes get names of genes known to respond.
	order := make([]int, len(genes))
	for i := range order {
		order[i] = i
	}
	slices.SortFunc(order, func(a, b int) int { return cmp.Compare(genes[a].P, genes[b].P) })
	for k, name := range symbols {
		genes[order[k]].Name = name
	}
	return genes
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(8)).Children(func() {
		volcano.View(c, &a.volcano, a.genes)
	})
}

func main() {
	a := &app{genes: sample()}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Volcano",
			Width:   860,
			Height:  640,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
