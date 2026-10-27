// Odometer rolls the numbers of a small shop's dashboard: the buttons
// add to and take from the orders, and Live changes the visitors and the
// revenue every second.
//
//	go run ./examples/odometer
package main

import (
	"fmt"
	"log"
	"math/rand/v2"
	"strconv"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/odometer"
)

type app struct {
	orders   int
	visitors int
	revenue  float64
	live     bool
	ticked   time.Time
}

// comma writes n with commas between its thousands.
func comma(n int) string {
	s := strconv.Itoa(n)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	if a.live {
		if c.Now().Sub(a.ticked) >= time.Second {
			a.ticked = c.Now()
			a.visitors = max(0, a.visitors+rand.IntN(40)-12)
			a.revenue = max(0, a.revenue+float64(rand.IntN(9000)-2500)/100)
		}
		c.After(time.Second - c.Now().Sub(a.ticked))
	}

	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(8)).Gap(t.Space(8)).AlignItems(ui.Center).Justify(ui.Center).Children(func() {
		ui.Row(c).Gap(t.Space(6)).Children(func() {
			card(c, "Orders", comma(a.orders))
			card(c, "Visitors now", comma(a.visitors))
			card(c, "Revenue", fmt.Sprintf("$%s.%02d", comma(int(a.revenue)), int(a.revenue*100)%100))
		})
		ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
			for _, d := range []int{1, 9, 120, -37, -500} {
				if ui.Button(c.Key(d), fmt.Sprintf("%+d", d)).Clicked() {
					a.orders = max(0, a.orders+d)
				}
			}
			ui.Spacer(c).Width(t.Space(4))
			if ui.Toggle(c, &a.live, "Live").Changed() && a.live {
				a.ticked = c.Now()
			}
		})
	})
}

func card(c *ui.Context, title, value string) {
	t := c.Theme()
	ui.Column(c.Key(title)).Gap(t.Space(1)).Padding(t.Space(5), t.Space(6)).Width(t.Space(52)).Radius(t.Space(3)).
		Background(t.Surface).Border(1, t.Border).Children(func() {
		ui.Text(c, title).FontSize(t.FontSize - 1).TextColor(t.TextMuted)
		odometer.View(c, value).FontSize(t.FontSize + 20).FontWeight(600)
	})
}

func main() {
	a := &app{orders: 1284, visitors: 87, revenue: 48210.5}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Odometer",
			Width:   820,
			Height:  360,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
