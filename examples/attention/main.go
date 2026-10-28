// Attention shows made-up attention of a small causal model over one
// sentence, four layers of six heads. The heads follow patterns real
// models learn: one looks at the token before, one at the first token,
// one at itself, one from "it" back to "cat", and the others spread out
// as layers go up. Hover a token, toggle heads, step through the layers,
// and switch to Matrix.
//
//	go run ./examples/attention
package main

import (
	"log"
	"math"
	"math/rand/v2"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/attention"
)

var tokens = []string{"<s>", "The", "cat", "sat", "on", "the", "mat", "because", "it", "was", "tired", "."}

// weights makes layers×heads of causal attention over n tokens, each row
// the softmax of the head's pattern plus noise.
func weights(layers, heads, n int) [][][][]float32 {
	rnd := rand.New(rand.NewPCG(7, 11))
	it, cat := 8, 2
	out := make([][][][]float32, layers)
	for l := range layers {
		out[l] = make([][][]float32, heads)
		for h := range heads {
			m := make([][]float32, n)
			for q := range n {
				logits := make([]float64, q+1)
				for k := range q + 1 {
					var s float64
					switch h {
					case 0:
						s = boost(k == q-1, 4)
					case 1:
						s = boost(k == 0, 3.5)
					case 2:
						s = boost(k == q, 3.5)
					case 3:
						s = boost(q == it && k == cat, 2.5+float64(l)) + boost(k == 0, 1.5)
					default:
						s = -0.15 * float64(q-k) / float64(l+1)
					}
					logits[k] = s + rnd.NormFloat64()*0.35
				}
				m[q] = softmax(logits, n)
			}
			out[l][h] = m
		}
	}
	return out
}

func boost(on bool, by float64) float64 {
	if on {
		return by
	}
	return 0
}

// softmax returns the softmax of logits, padded with zeros to n.
func softmax(logits []float64, n int) []float32 {
	peak := math.Inf(-1)
	for _, v := range logits {
		peak = max(peak, v)
	}
	sum := 0.0
	for _, v := range logits {
		sum += math.Exp(v - peak)
	}
	out := make([]float32, n)
	for k, v := range logits {
		out[k] = float32(math.Exp(v-peak) / sum)
	}
	return out
}

var (
	data  = attention.Attention{Tokens: tokens, Weights: weights(4, 6, len(tokens))}
	state attention.State
)

func view(c *ui.Context) {
	t := c.Theme()
	ui.Scroll(c).Fill().Background(t.Background).Children(func() {
		ui.Column(c).Padding(t.Space(6)).Gap(t.Space(4)).Children(func() {
			ui.Text(c, "Attention").Bold().FontSize(t.FontSize + 4)
			attention.View(c, &state, data)
		})
	})
}

func main() {
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Attention",
			Width:   760,
			Height:  640,
			Content: ui.View(view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
