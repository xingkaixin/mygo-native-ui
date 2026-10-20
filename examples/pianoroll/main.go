// Pianoroll edits a riff played in by hand, a little off the beat: draw
// and drag notes, quantize them, and shape the synth with the knobs.
//
//	go run ./examples/pianoroll
package main

import (
	"fmt"
	"log"
	"math"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/knob"
	"github.com/xingkaixin/mygo-native-ui/ui/pianoroll"
)

// riff is an arpeggio over Am, F, C and G with a bass note each bar, each
// note a little early or late, as hands play.
func riff() []pianoroll.Note {
	chords := [][]int{{57, 60, 64}, {53, 57, 60}, {55, 60, 64}, {55, 59, 62}}
	var out []pianoroll.Note
	k := 0
	for bar, ch := range chords {
		out = append(out, pianoroll.Note{Pitch: ch[0] - 12, Start: float64(bar * 4), Length: 3.5})
		for i := range 8 {
			off := 0.07 * math.Sin(float64(k)*2.3)
			k++
			out = append(out, pianoroll.Note{
				Pitch:  ch[[]int{0, 1, 2, 1}[i%4]] + 12*(i/4),
				Start:  max(0, float64(bar*4)+float64(i)*0.5+off),
				Length: 0.42 + 0.05*math.Cos(float64(k)),
			})
		}
	}
	return out
}

func seconds(v float64) string {
	if v < 1 {
		return fmt.Sprintf("%.0f ms", v*1000)
	}
	return fmt.Sprintf("%.2f s", v)
}

func hz(v float64) string {
	if v >= 1000 {
		return fmt.Sprintf("%.1f kHz", v/1000)
	}
	return fmt.Sprintf("%.0f Hz", v)
}

type app struct {
	roll  pianoroll.State
	notes []pianoroll.Note
	bpm   float64
	synth pianoroll.Synth
	wave  int
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(6)).Gap(t.Space(4)).Children(func() {
		ui.Row(c).Gap(t.Space(6)).AlignItems(ui.Center).Children(func() {
			ui.Column(c).Gap(t.Space(2)).Grow(1).Children(func() {
				ui.Text(c, "Riff in A minor").Bold().FontSize(t.FontSize + 6)
				if ui.Segmented(c, &a.wave, "Saw", "Square", "Triangle", "Sine").AlignSelf(ui.Start).Changed() {
					a.synth.Wave = pianoroll.Wave(a.wave)
				}
			})
			knob.View(c, &a.synth.Attack, knob.Knob{Label: "Attack", Min: 0.002, Max: 1, Default: 0.01, Log: true, Format: seconds})
			knob.View(c, &a.synth.Release, knob.Knob{Label: "Release", Min: 0.01, Max: 2, Default: 0.25, Log: true, Format: seconds})
			knob.View(c, &a.synth.Cutoff, knob.Knob{Label: "Cutoff", Min: 100, Max: 12000, Default: 2400, Log: true, Format: hz})
			knob.View(c, &a.synth.Volume, knob.Knob{Label: "Volume", Min: 0, Max: 1, Default: 0.7,
				Format: func(v float64) string { return fmt.Sprintf("%.0f%%", v*100) }})
		})
		pianoroll.View(c, &a.roll, &a.notes, &a.bpm, a.synth).Grow(1)
	})
}

func main() {
	a := &app{
		notes: riff(),
		bpm:   100,
		synth: pianoroll.Synth{Attack: 0.01, Release: 0.25, Cutoff: 2400, Volume: 0.7},
	}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Piano roll",
			Width:   1000,
			Height:  720,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
