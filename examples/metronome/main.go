// Metronome keeps time in a meter chosen above it, from two beats a bar
// to seven.
//
//	go run ./examples/metronome
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/metronome"
)

var meters = []struct {
	name  string
	beats []metronome.Beat
}{
	{"2/4", []metronome.Beat{metronome.Accent, metronome.Plain}},
	{"3/4", []metronome.Beat{metronome.Accent, metronome.Plain, metronome.Plain}},
	{"4/4", []metronome.Beat{metronome.Accent, metronome.Plain, metronome.Plain, metronome.Plain}},
	{"6/8", []metronome.Beat{metronome.Accent, metronome.Silent, metronome.Silent, metronome.Accent, metronome.Silent, metronome.Silent}},
	{"7/8", []metronome.Beat{metronome.Accent, metronome.Plain, metronome.Accent, metronome.Plain, metronome.Accent, metronome.Plain, metronome.Plain}},
}

type app struct {
	metronome metronome.State
	meter     int
	bpm       float64
	beats     []metronome.Beat
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Padding(t.Space(8)).Gap(t.Space(8)).AlignItems(ui.Center).Background(t.Background).Children(func() {
		names := make([]string, len(meters))
		for i, m := range meters {
			names[i] = m.name
		}
		if ui.Segmented(c, &a.meter, names...).Label("Meter").Changed() {
			a.beats = append([]metronome.Beat(nil), meters[a.meter].beats...)
		}
		metronome.View(c, &a.metronome, &a.bpm, &a.beats).AutoFocus()
	})
}

func main() {
	a := &app{meter: 2, bpm: 96}
	a.beats = append([]metronome.Beat(nil), meters[a.meter].beats...)
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Metronome",
			Width:   560,
			Height:  640,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
