// Matrix shows how an animal classifier did on its test photos, mixing
// up foxes and wolves, rabbits and hares, and the correlations between
// the columns of a listing of houses.
//
//	go run ./examples/matrix
package main

import (
	"fmt"
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/matrix"
)

var classes = []string{"cat", "dog", "fox", "wolf", "rabbit", "hare"}

var counts = [][]int{
	{161, 9, 4, 0, 2, 1},
	{12, 150, 3, 11, 0, 0},
	{3, 2, 128, 31, 1, 2},
	{0, 9, 22, 139, 0, 0},
	{2, 0, 1, 0, 118, 36},
	{0, 1, 3, 0, 29, 126},
}

var columns = []string{"price", "area", "bedrooms", "bathrooms", "age", "distance to center", "lot size"}

var correlations = [][]float64{
	{1, 0.82, 0.54, 0.61, -0.31, -0.58, 0.37},
	{0.82, 1, 0.71, 0.66, -0.12, -0.22, 0.48},
	{0.54, 0.71, 1, 0.58, 0.05, -0.08, 0.29},
	{0.61, 0.66, 0.58, 1, -0.27, -0.15, 0.24},
	{-0.31, -0.12, 0.05, -0.27, 1, 0.33, 0.18},
	{-0.58, -0.22, -0.08, -0.15, 0.33, 1, 0.52},
	{0.37, 0.48, 0.29, 0.24, 0.18, 0.52, 1},
}

type app struct {
	tab          int
	confusion    matrix.State
	correlations matrix.State
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Children(func() {
		ui.Tabs(c, &a.tab, "Classifier", "Houses").Padding(0, 12)
		ui.Divider(c)
		ui.Column(c).Padding(20).Gap(16).Children(func() {
			if a.tab == 0 {
				matrix.Confusion(c, &a.confusion, classes, counts)
				if i, j, ok := a.confusion.Chosen(); ok {
					ui.Text(c, fmt.Sprintf("%d photos of a %s the model took for a %s", counts[i][j], classes[i], classes[j]))
				}
				return
			}
			matrix.Correlation(c, &a.correlations, columns, correlations)
			if i, j, ok := a.correlations.Chosen(); ok {
				ui.Text(c, fmt.Sprintf("%s and %s: r = %.2f", columns[i], columns[j], correlations[i][j]))
			}
		})
	})
}

func main() {
	a := &app{}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Matrix",
			Width:   720,
			Height:  640,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
