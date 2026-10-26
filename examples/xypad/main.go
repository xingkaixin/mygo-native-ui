// Xypad sets a filter and a delay with a pad each: press or drag across a
// pad to move both of its values, hold Shift for fine moves, press the
// arrows while it has the focus, and double-click to reset it.
//
//	go run ./examples/xypad
package main

import (
	"fmt"
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/xypad"
)

type app struct {
	cutoff, resonance float64
	time, feedback    float64
}

var (
	cutoff = xypad.Axis{Label: "Cutoff", Min: 20, Max: 20000, Default: 2000, Log: true, Format: func(v float64) string {
		if v >= 1000 {
			return fmt.Sprintf("%.1f kHz", v/1000)
		}
		return fmt.Sprintf("%.0f Hz", v)
	}}
	resonance = xypad.Axis{Label: "Resonance", Min: 0, Max: 100, Default: 20, Format: percent}
	delay     = xypad.Axis{Label: "Time", Min: 10, Max: 2000, Default: 375, Log: true, Format: func(v float64) string {
		return fmt.Sprintf("%.0f ms", v)
	}}
	feedback = xypad.Axis{Label: "Feedback", Min: 0, Max: 100, Default: 35, Format: percent}
)

func percent(v float64) string { return fmt.Sprintf("%.0f%%", v) }

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Row(c).Fill().Background(t.Background).Padding(t.Space(8)).Gap(t.Space(8)).Justify(ui.Center).AlignItems(ui.Start).Children(func() {
		for i, s := range []struct {
			name string
			x, y *float64
			ax   xypad.Axis
			ay   xypad.Axis
		}{
			{"Filter", &a.cutoff, &a.resonance, cutoff, resonance},
			{"Delay", &a.time, &a.feedback, delay, feedback},
		} {
			ui.Column(c.Key(i)).Gap(t.Space(3)).Children(func() {
				ui.Text(c, s.name).Bold().FontSize(t.FontSize + 2)
				xypad.View(c, s.x, s.y, s.ax, s.ay).Width(t.Space(72))
			})
		}
	})
}

func main() {
	a := &app{cutoff: 2000, resonance: 20, time: 375, feedback: 35}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "XY Pad",
			Width:   720,
			Height:  480,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
