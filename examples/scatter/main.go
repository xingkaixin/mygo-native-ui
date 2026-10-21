// Scatter explores 100,000 made-up support tickets as a projection of
// their embeddings to two dimensions, where tickets about the same thing
// gather: each a dot colored by its topic, which the legend shows and
// hides. Scroll to zoom, drag with Space held or the middle button to
// pan, drag to select a box of tickets, Option-drag (Alt-drag) to draw a
// lasso, and hold Shift to add to the selection; the panel lists the
// first tickets selected.
//
//	go run ./examples/scatter
package main

import (
	"fmt"
	"log"
	"math"
	"math/rand/v2"
	"strconv"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/scatter"
)

// topic is a kind of ticket: its name, where its tickets gather, how many
// of every hundred are about it, and the subjects they come with.
type topic struct {
	name     string
	x, y     float64
	share    float64
	subjects []subject
}

// subject is a ticket's subject, its %s filled with one of words, or an
// order number without them.
type subject struct {
	format string
	words  []string
}

var (
	months  = []string{"January", "February", "March", "April", "May", "June"}
	plans   = []string{"Starter", "Team", "Business", "Enterprise"}
	devices = []string{"iPhone", "Android", "Windows", "a new Mac"}
	cities  = []string{"Leeds", "Lyon", "Porto", "Graz", "Malmö", "Utrecht"}
	items   = []string{"jacket", "boots", "hoodie", "backpack", "rain shell"}
	screens = []string{"Settings", "the cart", "Search", "Notifications", "Billing"}
	formats = []string{"CSV", "PDF", "Excel", "JSON"}
	tools   = []string{"Slack", "Notion", "Linear", "Zapier", "Jira"}
	people  = []string{"Ana", "Bram", "Chen", "Dalia", "Emre", "Freya"}
	seats   = []string{"3", "5", "10", "25"}
	none    []string
)

var topics = []topic{
	{"Billing", -6.5, 4.5, 20, []subject{
		{"Charged twice for the %s invoice", months},
		{"Card declined on %s plan renewal", plans},
		{"Refund not received for order #%s", none},
		{"VAT ID missing from the %s invoice", months},
	}},
	{"Sign-in", 5.5, 7, 16, []subject{
		{"Password reset email never arrives", none},
		{"Locked out after setting up 2FA on %s", devices},
		{"SSO sends me back to the sign-in page", none},
		{"Can't sign in on %s since the update", devices},
	}},
	{"Shipping", -3, -6, 18, []subject{
		{"Order #%s stuck in transit", none},
		{"Tracking for #%s shows no updates", none},
		{"Parcel left at the wrong address in %s", cities},
		{"Delivery to %s a week late", cities},
	}},
	{"Returns", -9.5, -2.5, 11, []subject{
		{"Return label for #%s won't print", none},
		{"Swap my %s for a larger size", items},
		{"Refund for the returned %s still pending", items},
	}},
	{"Bugs", 8, -2.5, 14, []subject{
		{"App crashes when opening %s", screens},
		{"Export to %s fails with error 500", formats},
		{"Text unreadable in dark mode in %s", screens},
		{"Notifications arrive twice on %s", devices},
	}},
	{"Feature requests", 11.5, 2.5, 11, []subject{
		{"Please add %s export", formats},
		{"Would love a %s integration", tools},
		{"Let me share dashboards with %s", people},
	}},
	{"Account", 1, 0.5, 10, []subject{
		{"Make %s the owner of our account", people},
		{"Delete my account and its data", none},
		{"Add %s seats to our Team plan", seats},
		{"Merge my two accounts", none},
	}},
}

// tickets makes up n tickets, the same each run, as a projection of their
// embeddings would place them: about their topic's place, in a clump for
// each subject, stretched one way, with one in fifty strays far from it.
func tickets(n int) (points []scatter.Point, names []string) {
	r := rand.New(rand.NewPCG(2026, 10))
	type clump struct{ x, y, sa, sb, turn float64 }
	clumps := make([][]clump, len(topics))
	total := 0.0
	for i, tp := range topics {
		names = append(names, tp.name)
		total += tp.share
		for k := range tp.subjects {
			a := 2*math.Pi*float64(k)/float64(len(tp.subjects)) + r.Float64()
			d := 1.2 + 1.4*r.Float64()
			clumps[i] = append(clumps[i], clump{
				x: tp.x + d*math.Cos(a), y: tp.y + d*math.Sin(a),
				sa: 0.7 + 0.7*r.Float64(), sb: 0.35 + 0.35*r.Float64(), turn: math.Pi * r.Float64(),
			})
		}
	}
	points = make([]scatter.Point, n)
	for i := range points {
		t, x := 0, r.Float64()*total
		for t < len(topics)-1 && x >= topics[t].share {
			x -= topics[t].share
			t++
		}
		k := r.IntN(len(topics[t].subjects))
		cl := clumps[t][k]
		a, b := cl.sa*r.NormFloat64(), cl.sb*r.NormFloat64()
		if r.IntN(50) == 0 {
			a, b = 3*a, 3*b
		}
		px := cl.x + a*math.Cos(cl.turn) - b*math.Sin(cl.turn)
		py := cl.y + a*math.Sin(cl.turn) + b*math.Cos(cl.turn)
		points[i] = scatter.Point{X: px, Y: py, Category: t, Label: fill(r, topics[t].subjects[k])}
	}
	return points, names
}

// fill writes subject s, its %s filled.
func fill(r *rand.Rand, s subject) string {
	if !strings.Contains(s.format, "%s") {
		return s.format
	}
	w := strconv.Itoa(10000 + r.IntN(90000))
	if len(s.words) > 0 {
		w = s.words[r.IntN(len(s.words))]
	}
	return fmt.Sprintf(s.format, w)
}

type app struct {
	points  []scatter.Point
	topics  []string
	plot    scatter.State
	colored bool
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(4), t.Space(5)).Gap(t.Space(4)).Children(func() {
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			ui.Column(c).Gap(t.Space(0.5)).Grow(1).Shrink(1).Children(func() {
				ui.Text(c, "Support tickets").Bold().FontSize(t.FontSize + 4)
				ui.Text(c, "100,000 made-up tickets, their embeddings projected to two dimensions").
					TextColor(t.TextMuted).SingleLine()
			})
			ui.Text(c, "Color by topic").TextColor(t.TextMuted)
			ui.Switch(c, &a.colored).Label("Color by topic")
		})
		ui.Row(c).Grow(1).AlignSelf(ui.Stretch).Gap(t.Space(5)).Children(func() {
			var ev scatter.Events
			ui.Column(c).Grow(1).Shrink(1).AlignSelf(ui.Stretch).MinHeight(t.Space(80)).Children(func() {
				var names []string
				if a.colored {
					names = a.topics
				}
				ev = scatter.View(c, &a.plot, "UMAP 1", "UMAP 2", names, a.points)
			})
			a.panel(c, ev.Selected())
		})
	})
}

// shownAtMost is how many of the tickets selected the panel lists.
const shownAtMost = 12

// panel lists the first of the tickets selected, with their topics.
func (a *app) panel(c *ui.Context, selected []int) {
	t := c.Theme()
	ui.Column(c).Width(t.Space(72)).AlignSelf(ui.Stretch).Gap(t.Space(3)).Padding(t.Space(4)).
		Radius(t.Radius + 4).Background(t.Surface).Children(func() {
		ui.Text(c, "Selected").Bold()
		if len(selected) == 0 {
			ui.Text(c, "Drag across the plot to select tickets, and they show here.").TextColor(t.TextMuted)
			return
		}
		for _, i := range selected[:min(shownAtMost, len(selected))] {
			p := a.points[i]
			ui.Column(c).Gap(t.Space(0.5)).Children(func() {
				ui.Text(c, p.Label).SingleLine()
				ui.Text(c, a.topics[p.Category]).FontSize(t.FontSize - 2).TextColor(t.TextMuted)
			})
		}
		if more := len(selected) - shownAtMost; more > 0 {
			ui.Text(c, fmt.Sprintf("and %s more", grouped(more))).TextColor(t.TextMuted)
		}
	})
}

// grouped writes n with commas between groups of three digits.
func grouped(n int) string {
	s := fmt.Sprint(n)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}

func main() {
	points, names := tickets(100_000)
	a := &app{points: points, topics: names, colored: true}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Scatter plot",
			Width:   1240,
			Height:  860,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
