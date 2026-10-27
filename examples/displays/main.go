// Displays arranges a laptop and two monitors as the Displays settings
// do: drag a display to move it beside the others, drag the white menu
// bar to another display to make it the main one, and click a display to
// change its resolution below.
//
//	go run ./examples/displays
package main

import (
	"fmt"
	"log"
	"slices"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/displays"
)

// modes are the resolutions each display offers, by name.
var modes = map[string][]string{
	"Built-in Display": {"1147 × 745", "1352 × 878", "1512 × 982", "1800 × 1169"},
	"Studio Display":   {"1600 × 900", "2048 × 1152", "2560 × 1440", "2880 × 1620"},
	"DELL U2723QE":     {"1280 × 720", "1920 × 1080", "2560 × 1440", "3008 × 1692"},
}

type app struct {
	displays []displays.Display
	arranger displays.State
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(6)).Gap(t.Space(4)).Children(func() {
		ui.Box(c).Grow(1).Radius(t.Space(3)).Background(t.Surface).Border(1, t.Border).Children(func() {
			displays.View(c, &a.arranger, &a.displays).Fill()
		})
		d := &a.displays[a.arranger.Chosen]
		ui.Row(c).Gap(t.Space(6)).AlignItems(ui.Center).Children(func() {
			ui.Column(c).Gap(t.Space(1)).Width(t.Space(60)).Children(func() {
				ui.Text(c, d.Name).FontSize(t.FontSize + 3).Bold()
				where := fmt.Sprintf("At %d, %d", d.X, d.Y)
				if d.Main {
					where = "Main display"
				}
				ui.Text(c, where).TextColor(t.TextMuted).FontFeatures("tnum")
			})
			ui.Field(c, "Resolution", func() {
				mode := fmt.Sprintf("%d × %d", d.Width, d.Height)
				if ui.Select(c, &mode, modes[d.Name]).Width(t.Space(44)).Changed() {
					fmt.Sscanf(mode, "%d × %d", &d.Width, &d.Height)
					a.rearrange()
				}
			})
			ui.Spacer(c)
			if ui.Button(c, "Use as Main Display").Disabled(d.Main).Clicked() {
				for i := range a.displays {
					a.displays[i].Main = i == a.arranger.Chosen
				}
				a.rearrange()
			}
		})
	})
}

// rearrange puts the displays side by side from the main one after a
// size changed, as the system does when one would cover another.
func (a *app) rearrange() {
	main := slices.IndexFunc(a.displays, func(d displays.Display) bool { return d.Main })
	x := a.displays[main].Width
	for i := range a.displays {
		d := &a.displays[i]
		if i == main {
			d.X, d.Y = 0, 0
			continue
		}
		d.X, d.Y = x, a.displays[main].Height-d.Height
		x += d.Width
	}
}

func main() {
	a := &app{displays: []displays.Display{
		{Name: "Built-in Display", Width: 1512, Height: 982, X: 0, Y: 0, Main: true},
		{Name: "Studio Display", Width: 2560, Height: 1440, X: -500, Y: -1440},
		{Name: "DELL U2723QE", Width: 1920, Height: 1080, X: 2060, Y: -1080},
	}}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Displays",
			Width:   880,
			Height:  620,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
