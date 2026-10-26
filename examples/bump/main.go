// Bump ranks eight languages by their share of a made-up developer
// survey from 2019 to 2025: some climb, some slip, and one enters late.
// Hover a disc or a name to follow a language, and click to keep it lit.
//
//	go run ./examples/bump
package main

import (
	"log"
	"math"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/bump"
)

var nan = math.NaN()

var years = []string{"2019", "2020", "2021", "2022", "2023", "2024", "2025"}

var languages = []bump.Series{
	{Name: "JavaScript", Values: []float64{31.2, 30.1, 28.4, 27.0, 25.3, 24.1, 22.8}},
	{Name: "Python", Values: []float64{22.4, 24.8, 26.1, 27.9, 29.6, 31.0, 32.2}},
	{Name: "Java", Values: []float64{19.8, 18.2, 16.9, 15.1, 13.8, 12.4, 11.6}},
	{Name: "TypeScript", Values: []float64{6.1, 8.4, 10.9, 13.2, 15.6, 17.1, 18.9}},
	{Name: "Go", Values: []float64{4.2, 5.0, 6.3, 7.1, 8.4, 9.2, 10.1}},
	{Name: "C#", Values: []float64{10.3, 9.9, 9.6, 9.0, 8.8, 8.1, 7.7}},
	{Name: "PHP", Values: []float64{8.9, 7.6, 6.2, 5.4, 4.6, 4.1, 3.5}},
	{Name: "Rust", Values: []float64{nan, nan, 2.9, 4.4, 6.2, 8.6, 9.4}},
}

type app struct {
	chart bump.State
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(6)).Gap(t.Space(5)).Children(func() {
		ui.Text(c, "Most used languages").Bold().FontSize(t.FontSize + 4)
		bump.View(c, &a.chart, years, languages).Grow(1)
	})
}

func main() {
	a := &app{}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Bump chart",
			Width:   860,
			Height:  520,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
