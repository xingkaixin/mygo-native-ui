// Calibration compares three models' probabilities on the same 4,000
// examples: one that means what it says, one too sure of itself and one
// too timid. Choose how many bins to put the predictions in, and point
// at a bin to read it.
//
//	go run ./examples/calibration
package main

import (
	"log"
	"math"
	"math/rand/v2"
	"strconv"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/calibration"
)

var counts = []int{5, 10, 15, 20}

type app struct {
	cal    calibration.State
	models []calibration.Model
	bins   int
}

func logit(p float64) float64   { return math.Log(p / (1 - p)) }
func sigmoid(x float64) float64 { return 1 / (1 + math.Exp(-x)) }

func sample() []calibration.Model {
	r := rand.New(rand.NewPCG(5, 6))
	models := []calibration.Model{{Name: "Logistic regression"}, {Name: "Boosted trees"}, {Name: "Random forest"}}
	// Each model sees the true odds, scaled: 1 means what it says, more
	// than 1 pushes toward the ends, less pulls toward the middle.
	sharpness := []float64{1, 2.2, 0.55}
	for range 4000 {
		q := sigmoid(1.6 * r.NormFloat64())
		positive := r.Float64() < q
		for i := range models {
			p := sigmoid(sharpness[i]*logit(q) + 0.25*r.NormFloat64())
			models[i].Predictions = append(models[i].Predictions, calibration.Prediction{P: p, Positive: positive})
		}
	}
	return models
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(8)).Gap(t.Space(4)).Children(func() {
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "Bins").TextColor(t.TextMuted)
			labels := make([]string, len(counts))
			for i, n := range counts {
				labels[i] = strconv.Itoa(n)
			}
			ui.Segmented(c, &a.bins, labels...).Label("Bins")
		})
		calibration.View(c, &a.cal, a.models, counts[a.bins])
	})
}

func main() {
	a := &app{models: sample(), bins: 1}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Calibration",
			Width:   720,
			Height:  640,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
