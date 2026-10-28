// Sampling shows made-up candidates for the token after "Once upon a
// time, there was a", from the likely " little" to the unlikely line
// break. Move the sliders to see which candidates each setting cuts and
// how the rest share what is left; Greedy, Balanced and Creative set
// them as playgrounds often suggest.
//
//	go run ./examples/sampling
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/sampling"
)

var tokens = []string{" little", " young", " king", " princess", " girl", " man", " boy", " small", " beautiful", " very", " dragon", " poor", " wise", " cat", " lonely", " curious", " tiny", " brave", " magical", "\n"}

var logits = []float64{6.1, 5.2, 4.9, 4.8, 4.5, 4.3, 4.1, 3.7, 3.4, 3.2, 2.9, 2.7, 2.4, 2.2, 1.9, 1.6, 1.4, 1.1, 0.8, -0.5}

var candidates = func() []sampling.Candidate {
	out := make([]sampling.Candidate, len(tokens))
	for i, tok := range tokens {
		out[i] = sampling.Candidate{Token: tok, Logit: logits[i]}
	}
	return out
}()

var presets = []struct {
	name   string
	params sampling.Params
}{
	{"Greedy", sampling.Params{Temperature: 0, TopP: 1}},
	{"Balanced", sampling.Params{Temperature: 0.8, TopP: 0.95, MinP: 0.05}},
	{"Creative", sampling.Params{Temperature: 1.3, TopK: 0, TopP: 1, MinP: 0.02}},
}

var params = presets[1].params

func view(c *ui.Context) {
	t := c.Theme()
	ui.Scroll(c).Fill().Background(t.Background).Children(func() {
		ui.Column(c).Padding(t.Space(6)).Gap(t.Space(5)).Children(func() {
			ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
				ui.RichText(c,
					ui.Span{Text: "Once upon a time, there was a", Color: t.TextMuted},
					ui.Span{Text: " ▍", Color: t.Accent},
				).FontSize(t.FontSize + 3).Grow(1)
				for _, ps := range presets {
					if ui.Button(c.Key(ps.name), ps.name).Clicked() {
						params = ps.params
					}
				}
			})
			sampling.View(c, &params, candidates)
		})
	})
}

func main() {
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Sampling",
			Width:   760,
			Height:  860,
			Content: ui.View(view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
