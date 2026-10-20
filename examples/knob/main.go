// Knob is the panel of a guitar amp: drag a knob up or down to turn it,
// with Shift for fine turns, and double-click it to reset it.
//
//	go run ./examples/knob
package main

import (
	"fmt"
	"log"
	"math"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/knob"
)

func db(v float64) string {
	if math.Abs(v) < 0.05 {
		return "0 dB"
	}
	return fmt.Sprintf("%+.1f dB", v)
}

func ten(v float64) string { return fmt.Sprintf("%.1f", v) }

func hz(v float64) string {
	if v >= 1000 {
		return fmt.Sprintf("%.1f kHz", v/1000)
	}
	return fmt.Sprintf("%.0f Hz", v)
}

func ms(v float64) string { return fmt.Sprintf("%.0f ms", v) }

var knobs = []knob.Knob{
	{Label: "Gain", Min: 0, Max: 10, Default: 5, Format: ten},
	{Label: "Bass", Min: -12, Max: 12, Default: 0, Format: db},
	{Label: "Middle", Min: -12, Max: 12, Default: 0, Format: db},
	{Label: "Treble", Min: -12, Max: 12, Default: 0, Format: db},
	{Label: "Tone", Min: 200, Max: 8000, Default: 1600, Log: true, Format: hz},
	{Label: "Delay", Min: 20, Max: 1200, Default: 350, Log: true, Format: ms},
	{Label: "Master", Min: 0, Max: 10, Default: 7, Format: ten},
}

type app struct {
	values []float64
	turned string
}

func defaults() []float64 {
	v := make([]float64, len(knobs))
	for i, k := range knobs {
		v[i] = k.Default
	}
	return v
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(8)).Gap(t.Space(6)).AlignItems(ui.Center).Justify(ui.Center).Children(func() {
		ui.Row(c).Gap(t.Space(4)).Padding(t.Space(6), t.Space(8)).Radius(t.Space(3)).
			Background(t.Surface).Border(1, t.Border).Shadow(0, 2, 8, 0, ui.RGB(0, 0, 0).Alpha(0.12)).Children(func() {
			for i, k := range knobs {
				if knob.View(c.Key(i), &a.values[i], k).Changed() {
					a.turned = k.Label + " " + k.Format(a.values[i])
				}
			}
		})
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, a.turned).FontSize(t.FontSize - 1).TextColor(t.TextMuted).Width(t.Space(40))
			if ui.Button(c, "Reset all").Clicked() {
				a.values, a.turned = defaults(), ""
			}
		})
	})
}

func main() {
	a := &app{values: defaults()}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Knob",
			Width:   760,
			Height:  320,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
