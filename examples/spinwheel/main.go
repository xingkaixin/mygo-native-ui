// Spinwheel picks where to go for lunch, and can take the place picked
// off the wheel for next time.
//
//	go run ./examples/spinwheel
package main

import (
	"log"
	"slices"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/spinwheel"
)

var lunch = []string{"Ramen", "Tacos", "Pho", "Pizza", "Salad bar", "Curry", "Sushi", "Burgers", "Dumplings", "Falafel"}

type app struct {
	wheel   spinwheel.State
	choices []string
	picked  int
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Padding(t.Space(8)).Gap(t.Space(5)).AlignItems(ui.Center).Justify(ui.Center).Background(t.Background).Children(func() {
		ev := spinwheel.View(c, &a.wheel, a.choices)
		if ev.Landed {
			a.picked = ev.At
		}
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Height(t.Space(9)).Children(func() {
			if a.picked < 0 {
				ui.Text(c, "Click the wheel to spin it").TextColor(t.TextMuted)
				return
			}
			ui.Text(c, a.choices[a.picked]).FontSize(t.FontSize + 8).Bold()
			if len(a.choices) > 2 && ui.Button(c, "Take off the wheel").Clicked() {
				a.choices = slices.Delete(a.choices, a.picked, a.picked+1)
				a.picked, a.wheel = -1, spinwheel.State{}
			}
		})
		if ui.Button(c, "Start over").Disabled(len(a.choices) == len(lunch)).Clicked() {
			a.choices, a.picked, a.wheel = slices.Clone(lunch), -1, spinwheel.State{}
		}
	})
}

func main() {
	a := &app{choices: slices.Clone(lunch), picked: -1}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Lunch wheel",
			Width:   560,
			Height:  640,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
