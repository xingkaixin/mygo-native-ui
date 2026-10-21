// Parallel explores 600 made-up cars from 1970 to 1982 as parallel
// coordinates, from their cylinders to where they were made. Drag along an
// axis to keep the cars in a stretch of it, drag a name to move its axis
// and click it to flip it, and choose what colors the lines; the table
// lists the first 20 cars that pass.
//
//	go run ./examples/parallel
package main

import (
	"log"
	"math"
	"math/rand/v2"
	"slices"
	"strconv"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/parallel"
)

var dims = []parallel.Dim{
	{Name: "Cylinders", Categories: []string{"3", "4", "5", "6", "8"}},
	{Name: "Displacement", Unit: "cu in"},
	{Name: "Horsepower", Unit: "hp"},
	{Name: "Weight", Unit: "lb", Format: grouped},
	{Name: "0–60 mph", Unit: "s"},
	{Name: "MPG"},
	{Name: "Year"},
	{Name: "Origin", Categories: []string{"USA", "Europe", "Japan"}},
}

// The indexes of dims.
const (
	cylinders = iota
	displacement
	horsepower
	weight
	acceleration
	mpg
	year
	origin
)

type app struct {
	cars    []parallel.Row
	chart   parallel.State
	colorBy string
	list    ui.ListState
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	choices := []string{"None"}
	for _, d := range dims {
		choices = append(choices, d.Name)
	}
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(4), t.Space(5)).Gap(t.Space(4)).Children(func() {
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			ui.Column(c).Gap(t.Space(0.5)).Grow(1).Shrink(1).Children(func() {
				ui.Text(c, "Cars, 1970–1982").Bold().FontSize(t.FontSize + 4)
				ui.Text(c, "600 made-up cars across eight dimensions").TextColor(t.TextMuted).SingleLine()
			})
			ui.Text(c, "Color by").TextColor(t.TextMuted)
			ui.Select(c, &a.colorBy, choices).Label("Color by")
			ui.Text(c, "Smooth").TextColor(t.TextMuted)
			ui.Switch(c, &a.chart.Smooth).Label("Smooth")
		})
		var ev parallel.Events
		ui.Column(c).Grow(1).AlignSelf(ui.Stretch).MinHeight(t.Space(80)).Children(func() {
			ev = parallel.View(c, &a.chart, dims, a.cars, slices.Index(choices, a.colorBy)-1)
		})
		shown := ev.Passing()
		ui.Text(c, "The first "+strconv.Itoa(min(20, len(shown)))+" of the "+strconv.Itoa(len(shown))+" cars that pass").
			TextColor(t.TextMuted)
		cols := []ui.TableColumn{{Title: "Name"}}
		for _, d := range dims {
			cols = append(cols, ui.TableColumn{Title: d.Name, Width: t.Space(29), Align: ui.End})
		}
		ui.Table(c, &a.list, cols, min(20, len(shown)), func(row, col int) {
			car := a.cars[shown[row]]
			if col == 0 {
				ui.Text(c, car.Name).SingleLine()
				return
			}
			ui.Text(c, value(col-1, car.Values[col-1])).FontFeatures("tnum")
		}).Height(t.Space(64))
	})
}

// value writes v of dimension d as the table shows it.
func value(d int, v float64) string {
	switch {
	case math.IsNaN(v):
		return "–"
	case dims[d].Categories != nil:
		return dims[d].Categories[int(v)]
	case d == weight:
		return grouped(v)
	case d == acceleration || d == mpg:
		return strconv.FormatFloat(v, 'f', 1, 64)
	}
	return strconv.FormatFloat(v, 'f', 0, 64)
}

// grouped writes v rounded, with commas between groups of three digits.
func grouped(v float64) string {
	s := strconv.FormatFloat(math.Round(v), 'f', 0, 64)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}

// makes are made-up carmakers of each origin.
var makes = [][]string{
	{"Aldridge", "Bellmont", "Carver", "Dunmore", "Elkhart"},
	{"Vauclair", "Heidrun", "Morello", "Strand"},
	{"Kanto", "Shiro", "Hayate", "Nagare"},
}

var bodies = []string{"Sedan", "Coupe", "Wagon", "Hatchback", "Hardtop"}
var trims = []string{"", "", " DL", " GT", " SE", " Deluxe"}

// cars makes up n cars, the same each run: their engines grow with their
// cylinders, power and weight with their engines, and mileage falls with
// weight and power and climbs over the years, as the 1970s' did. One in a
// hundred has no horsepower on record.
func cars(n int) []parallel.Row {
	r := rand.New(rand.NewPCG(1970, 82))
	pick := func(weights ...float64) int {
		x := r.Float64()
		for i, w := range weights {
			if x < w {
				return i
			}
			x -= w
		}
		return len(weights) - 1
	}
	// Cylinders, as indexes of dims[cylinders], and the range of their
	// displacement.
	engines := []struct{ lo, hi float64 }{{70, 90}, {85, 150}, {120, 140}, {145, 260}, {260, 455}}
	out := make([]parallel.Row, n)
	for i := range out {
		from := pick(0.6, 0.2, 0.2)
		y := float64(1970 + r.IntN(13))
		later := (y - 1970) / 12
		var cyl int
		switch from {
		case 0:
			cyl = []int{1, 3, 4}[pick(0.3+0.3*later, 0.3, 0.4-0.3*later)]
		case 1:
			cyl = []int{1, 2, 3}[pick(0.8, 0.06, 0.14)]
		default:
			cyl = []int{0, 1, 3}[pick(0.05, 0.85, 0.1)]
		}
		e := engines[cyl]
		disp := math.Round(e.lo + (e.hi-e.lo)*r.Float64()*(1-0.3*later))
		hp := math.Round(max(46, min(230, 0.33*disp+28+8*r.NormFloat64()-10*later)))
		wt := math.Round(max(1600, min(5100, 1550+7.2*disp+180*r.NormFloat64())))
		if from == 2 {
			wt = math.Round(wt * 0.94)
		}
		acc := math.Round(max(8, min(24.8, 21.5-0.05*hp+wt/2400+1.1*r.NormFloat64()))*10) / 10
		mileage := math.Round(max(9, min(46.6, 51-0.0062*wt-0.03*hp+9*later+2*r.NormFloat64()))*10) / 10
		if r.IntN(100) == 0 {
			hp = math.NaN()
		}
		name := makes[from][r.IntN(len(makes[from]))] + " " + bodies[r.IntN(len(bodies))] + trims[r.IntN(len(trims))] +
			" ’" + strconv.Itoa(int(y)%100)
		out[i] = parallel.Row{Name: name, Values: []float64{float64(cyl), disp, hp, wt, acc, mileage, y, float64(from)}}
	}
	return out
}

func main() {
	a := &app{cars: cars(600), colorBy: "MPG"}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Parallel coordinates",
			Width:   1180,
			Height:  860,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
