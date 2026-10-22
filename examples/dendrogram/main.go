// Dendrogram clusters 44 cities by their climate, their temperature and
// rainfall month by month, as a climatologist's notebook does: the
// linkage to switch, the clusters to cut by count or by dragging the cut,
// and the months beside the tree as a clustered heatmap. The data is
// made up, near enough to the real climates.
//
//	go run ./examples/dendrogram
package main

import (
	"hash/fnv"
	"log"
	"math"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/dendrogram"
)

type app struct {
	items   []dendrogram.Item
	vectors [][]float64
	columns []string
	link    int
	tree    *dendrogram.Tree
	cut     float64
	k       float64
	heat    bool
	across  int
	chart   dendrogram.State
}

var links = []dendrogram.Linkage{dendrogram.Single, dendrogram.Complete, dendrogram.Average, dendrogram.Ward}

func (a *app) cluster() {
	a.tree = dendrogram.Cluster(a.vectors, dendrogram.Euclidean, links[a.link])
	a.cut = a.tree.Cut(int(a.k))
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Children(func() {
		ui.Row(c).Padding(t.Space(3)).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "City climates").Bold()
			ui.Textf(c, "%d cities", len(a.items)).TextColor(t.TextMuted).Grow(1)
			if ui.Segmented(c, &a.link, "Single", "Complete", "Average", "Ward").Label("Linkage").Changed() {
				a.cluster()
			}
			ui.Text(c, "Clusters").TextColor(t.TextMuted)
			if ui.NumberInput(c, &a.k, 1, float64(len(a.items)), 1).Label("Clusters").Changed() {
				a.cut = a.tree.Cut(int(a.k))
			}
			ui.Segmented(c, &a.across, "Left to right", "Top down").Label("Orientation")
			ui.Toggle(c, &a.heat, "Heatmap")
		})
		ui.Divider(c)
		a.chart.TopDown = a.across == 1
		var columns []string
		if a.heat {
			columns = a.columns
		}
		if dendrogram.View(c, &a.chart, a.tree, a.items, columns, &a.cut).Grow(1).Changed() {
			_, k := a.tree.Clusters(a.cut)
			a.k = float64(k)
			// The count above read the count before the cut moved.
			c.Invalidate()
		}
	})
}

// city is a city's climate in a few numbers: its mean temperature, how
// far its months swing from it, its rainfall in a year, how that falls
// over the year, and whether it is south of the equator, where the
// seasons turn the other way.
type city struct {
	name        string
	mean, swing float64
	rain        float64
	season      season
	south       bool
}

// season is when the rain falls.
type season int

const (
	even   season = iota
	summer        // monsoon and continental climates
	winter        // Mediterranean climates
)

var cities = []city{
	{"London", 11.3, 7, 690, even, false},
	{"Paris", 12.5, 8, 640, even, false},
	{"Amsterdam", 10.5, 7, 840, even, false},
	{"Dublin", 10, 5, 760, even, false},
	{"Hamburg", 9.5, 8.5, 770, even, false},
	{"Seattle", 11.5, 7.5, 950, winter, false},
	{"Vancouver", 11, 7, 1450, winter, false},
	{"Moscow", 5.8, 13.5, 700, summer, false},
	{"Warsaw", 8.5, 10.5, 520, summer, false},
	{"Kyiv", 8.4, 12, 620, summer, false},
	{"Chicago", 10, 14, 940, summer, false},
	{"Montreal", 6.8, 15, 1000, even, false},
	{"Toronto", 9, 13, 830, even, false},
	{"Beijing", 12.9, 15, 570, summer, false},
	{"Harbin", 4.2, 21, 520, summer, false},
	{"Rome", 15.5, 8.5, 800, winter, false},
	{"Athens", 18.5, 9, 400, winter, false},
	{"Lisbon", 17.5, 6, 720, winter, false},
	{"Barcelona", 16.5, 7.5, 600, winter, false},
	{"Los Angeles", 18.5, 4.5, 370, winter, false},
	{"San Francisco", 14.5, 3.5, 600, winter, false},
	{"Perth", 18.5, 6, 730, winter, true},
	{"Tokyo", 16, 10.5, 1530, summer, false},
	{"Shanghai", 17, 12, 1180, summer, false},
	{"Hong Kong", 23.5, 6, 2400, summer, false},
	{"Mumbai", 27.5, 2.5, 2200, summer, false},
	{"Kolkata", 26.8, 6, 1580, summer, false},
	{"Bangkok", 28.5, 2, 1500, summer, false},
	{"Houston", 21, 8, 1260, even, false},
	{"Singapore", 27.5, 0.6, 2340, even, false},
	{"Kuala Lumpur", 27.5, 0.5, 2600, even, false},
	{"Jakarta", 27.5, 0.5, 1800, summer, true},
	{"Sydney", 18.5, 4.5, 1150, even, true},
	{"Melbourne", 15, 5, 600, even, true},
	{"Buenos Aires", 18, 6.5, 1200, even, true},
	{"Cape Town", 17, 4.5, 520, winter, true},
	{"Santiago", 14.5, 6.5, 310, winter, true},
	{"Auckland", 15.5, 4, 1200, winter, true},
	{"Cairo", 22.5, 7, 25, winter, false},
	{"Dubai", 28, 8, 80, winter, false},
	{"Riyadh", 26.5, 10, 110, winter, false},
	{"Phoenix", 24, 10, 200, even, false},
	{"Mexico City", 17, 2.5, 820, summer, false},
	{"Nairobi", 19, 1.5, 900, even, true},
}

var months = []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}

// climates returns each city's temperature, in °C, then rainfall, in mm,
// for each month, a little off the even curve as real years are.
func climates() []dendrogram.Item {
	items := make([]dendrogram.Item, len(cities))
	for i, ct := range cities {
		h := fnv.New64a()
		h.Write([]byte(ct.name))
		seed := h.Sum64()
		noise := func() float64 {
			seed = seed*6364136223846793005 + 1442695040888963407
			return float64(seed>>11)/(1<<53)*2 - 1
		}
		values := make([]float64, 24)
		for m := range 12 {
			// January is coldest in the north, July in the south.
			wave := -math.Cos(2 * math.Pi * float64(m) / 12)
			if ct.south {
				wave = -wave
			}
			values[m] = math.Round((ct.mean+ct.swing*wave+0.6*noise())*10) / 10
			share := 1.0
			switch ct.season {
			case summer:
				share = 1 + 0.9*wave
			case winter:
				share = 1 - 0.9*wave
			}
			values[12+m] = math.Round(max(0, ct.rain/12*share*(1+0.2*noise())))
		}
		items[i] = dendrogram.Item{Name: ct.name, Values: values}
	}
	return items
}

// standardize returns the columns of items each scaled to a mean of 0 and
// a standard deviation of 1, so that rainfall, in hundreds of mm, does not
// outweigh temperature.
func standardize(items []dendrogram.Item) [][]float64 {
	out := make([][]float64, len(items))
	for i := range out {
		out[i] = make([]float64, len(items[i].Values))
	}
	for j := range items[0].Values {
		var sum, sq float64
		for _, it := range items {
			sum += it.Values[j]
		}
		mean := sum / float64(len(items))
		for _, it := range items {
			sq += (it.Values[j] - mean) * (it.Values[j] - mean)
		}
		sd := max(1e-9, math.Sqrt(sq/float64(len(items))))
		for i, it := range items {
			out[i][j] = (it.Values[j] - mean) / sd
		}
	}
	return out
}

func newApp() *app {
	a := &app{link: 3, k: 6, heat: true}
	a.items = climates()
	a.vectors = standardize(a.items)
	for _, m := range months {
		a.columns = append(a.columns, m+" °C")
	}
	for _, m := range months {
		a.columns = append(a.columns, m+" mm")
	}
	a.cluster()
	return a
}

func main() {
	a := newApp()
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Dendrogram",
			Width:   1100,
			Height:  760,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
