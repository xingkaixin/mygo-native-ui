// Seatmap books seats: in a cinema, rows by letter and three price zones,
// or on a plane, rows by number and seats by letter.
//
//	go run ./examples/seatmap
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/seatmap"
)

// sold marks the seats of plan whose number is a multiple of every in
// rows that are, as booked by others, and some more by hand.
func sold(rows []string, every int, more ...seatmap.Seat) map[seatmap.Seat]bool {
	out := map[seatmap.Seat]bool{}
	k := -1
	for _, line := range rows {
		n := 0
		for _, ch := range line {
			if ch >= '0' && ch <= '9' {
				n++
			}
		}
		if n == 0 {
			continue
		}
		k++
		for i := 1; i <= n; i++ {
			if (k*7+i*3)%every == 0 {
				out[seatmap.Seat{Row: k, Num: i}] = true
			}
		}
	}
	for _, s := range more {
		out[s] = true
	}
	return out
}

var hall = []string{
	"  00000000000000  ",
	" 0000000000000000 ",
	" 0000000000000000 ",
	"",
	"000000111111000000",
	"000000111111000000",
	"000001111111100000",
	"000001111111100000",
	"",
	" 2222  2222  2222 ",
	" 2222  2222  2222 ",
}

var plane = []string{
	"00 00",
	"00 00",
	"00 00",
	"",
	"111 111",
	"111 111",
	"",
	"222 222",
	"222 222",
	"222 222",
	"222 222",
	"222 222",
	"222 222",
}

var plans = []struct {
	title string
	plan  seatmap.Plan
	limit int
}{
	{"Hall 3 · Tonight 19:30", seatmap.Plan{
		Front: "Screen",
		Rows:  hall,
		Zones: []seatmap.Zone{
			{Name: "Standard", Price: 12},
			{Name: "Premium", Price: 16},
			{Name: "Recliner", Price: 24},
		},
		Sold:     sold(hall, 5, seatmap.Seat{Row: 3, Num: 8}, seatmap.Seat{Row: 3, Num: 9}),
		Currency: "$",
	}, 6},
	{"Flight 482 · Lisbon", seatmap.Plan{
		Front: "Front",
		Rows:  plane,
		Zones: []seatmap.Zone{
			{Name: "Business", Price: 340},
			{Name: "Extra legroom", Price: 45},
			{Name: "Economy", Price: 0},
		},
		Sold:     sold(plane, 4),
		Letters:  true,
		Currency: "€",
	}, 2},
}

type app struct {
	plan  int
	seats []seatmap.Seat
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	p := plans[a.plan]
	ui.Scroll(c).Fill().Background(t.Background).Children(func() {
		ui.Column(c).Padding(t.Space(6)).Gap(t.Space(5)).Children(func() {
			ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
				ui.Text(c, p.title).Bold().FontSize(t.FontSize + 6).Grow(1)
				if ui.Segmented(c, &a.plan, "Cinema", "Plane").Changed() {
					a.seats = nil
				}
			})
			seatmap.View(c, p.plan, &a.seats, p.limit)
		})
	})
}

func main() {
	a := &app{}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Seat map",
			Width:   760,
			Height:  760,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
