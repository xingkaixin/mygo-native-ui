// Embeddings lays out made-up word vectors of six topics. Each word's
// 16-dimensional vector is its topic's center plus noise, a few words
// sitting between two topics, and the plot is the vectors' first two
// principal components, so that neighbors by cosine similarity are not
// always the dots nearest on the plot. Hover a dot, click one to keep
// it, search, and hide topics with their chips.
//
//	go run ./examples/embeddings
package main

import (
	"log"
	"math"
	"math/rand/v2"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/embeddings"
)

var topics = []struct {
	name  string
	words []string
}{
	{"Animals", []string{"cat", "dog", "horse", "rabbit", "fox", "wolf", "owl", "otter", "badger", "deer", "goat", "sheep"}},
	{"Fruit", []string{"apple", "pear", "plum", "cherry", "mango", "banana", "lemon", "lime", "peach", "grape", "kiwi", "fig"}},
	{"Vehicles", []string{"car", "bus", "truck", "bicycle", "tram", "train", "scooter", "van", "taxi", "ferry", "plane", "motorbike"}},
	{"Feelings", []string{"happy", "sad", "angry", "calm", "afraid", "proud", "bored", "excited", "lonely", "grateful", "nervous", "content"}},
	{"Weather", []string{"rain", "snow", "fog", "storm", "sunshine", "hail", "drizzle", "breeze", "thunder", "frost", "heat", "cloud"}},
	{"Cities", []string{"Paris", "Tokyo", "Lagos", "Lima", "Oslo", "Cairo", "Delhi", "Sydney", "Toronto", "Seoul", "Madrid", "Nairobi"}},
}

// between are words whose vectors mix two topics.
var between = map[string]int{"fox": 3, "lemon": 4, "plane": 5, "storm": 3, "calm": 4, "Lima": 1}

const dims = 16

func points() []embeddings.Point {
	rnd := rand.New(rand.NewPCG(3, 5))
	gauss := func(scale float64) []float32 {
		v := make([]float32, dims)
		for i := range v {
			v[i] = float32(rnd.NormFloat64() * scale)
		}
		return v
	}
	centers := make([][]float32, len(topics))
	for i := range centers {
		centers[i] = gauss(1)
	}
	var out []embeddings.Point
	for ti, tp := range topics {
		for _, w := range tp.words {
			v := gauss(0.45)
			for d := range v {
				v[d] += centers[ti][d]
				if o, ok := between[w]; ok {
					v[d] = v[d]*0.55 + centers[o][d]*0.45
				}
			}
			out = append(out, embeddings.Point{Label: w, Group: tp.name, Vec: v})
		}
	}
	pca(out)
	return out
}

// pca sets each point's X and Y to its vector's first two principal
// components, found by power iteration on the covariance.
func pca(pts []embeddings.Point) {
	mean := make([]float64, dims)
	for _, p := range pts {
		for d, v := range p.Vec {
			mean[d] += float64(v) / float64(len(pts))
		}
	}
	centered := make([][]float64, len(pts))
	for i, p := range pts {
		centered[i] = make([]float64, dims)
		for d, v := range p.Vec {
			centered[i][d] = float64(v) - mean[d]
		}
	}
	var axes [][]float64
	for range 2 {
		a := make([]float64, dims)
		a[len(axes)] = 1
		for range 100 {
			next := make([]float64, dims)
			for _, x := range centered {
				dot := 0.0
				for d := range x {
					dot += x[d] * a[d]
				}
				for d := range x {
					next[d] += dot * x[d]
				}
			}
			for _, b := range axes {
				dot := 0.0
				for d := range next {
					dot += next[d] * b[d]
				}
				for d := range next {
					next[d] -= dot * b[d]
				}
			}
			norm := 0.0
			for _, v := range next {
				norm += v * v
			}
			for d := range next {
				a[d] = next[d] / math.Sqrt(norm)
			}
		}
		axes = append(axes, a)
	}
	for i, x := range centered {
		for d := range x {
			pts[i].X += x[d] * axes[0][d]
			pts[i].Y += x[d] * axes[1][d]
		}
	}
}

var (
	pts   = points()
	state embeddings.State
)

func view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(5)).Gap(t.Space(4)).Children(func() {
		ui.Text(c, "Word embeddings").Bold().FontSize(t.FontSize + 4)
		embeddings.View(c, &state, pts, 6).Grow(1)
	})
}

func main() {
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Embeddings",
			Width:   1040,
			Height:  720,
			Content: ui.View(view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
