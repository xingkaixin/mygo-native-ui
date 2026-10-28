// Tablature shows a blues lick in E to play along with or edit: Play runs
// the playhead through it at the tempo, and the tab written as text
// follows the edits below.
//
//	go run ./examples/tablature
package main

import (
	"log"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/tablature"
)

type app struct {
	tab      tablature.Tab
	state    tablature.State
	bpm      float64
	playing  bool
	started  time.Time
	from     float64
	playhead float64
}

func n(str, fret int, tech ...tablature.Tech) tablature.Note {
	note := tablature.Note{String: str, Fret: fret}
	for _, t := range tech {
		note.Tech |= t
	}
	return note
}

func b(length int, notes ...tablature.Note) tablature.Beat {
	return tablature.Beat{Length: length, Notes: notes}
}

func lick() tablature.Tab {
	return tablature.Tab{
		Tuning: []string{"e", "B", "G", "D", "A", "E"},
		Bars: []tablature.Bar{
			{Beats: []tablature.Beat{b(8, n(5, 0)), b(8, n(5, 0)), b(8, n(4, 2)), b(8, n(4, 2)),
				b(8, n(5, 0)), b(8, n(5, 0)), b(8, n(4, 4)), b(8, n(4, 4))}},
			{Beats: []tablature.Beat{b(16, n(3, 0)), b(16, n(3, 2, tablature.Legato)), b(8, n(3, 0)),
				b(16, n(2, 0)), b(16, n(2, 2, tablature.Legato)), b(8, n(2, 4, tablature.Slide)),
				b(4, n(1, 3, tablature.Bend)), b(4, n(0, 0), n(1, 0), n(2, 1))}},
			{Beats: []tablature.Beat{b(8, n(0, 3)), b(8, n(0, 0, tablature.Legato)), b(8, n(1, 3)), b(8, n(1, 0, tablature.Legato)),
				b(8, n(2, 2)), b(8, n(2, 0, tablature.Legato)), b(8, n(3, 2)), b(8, n(4, 2, tablature.Dead))}},
			{Beats: []tablature.Beat{b(2, n(5, 0, tablature.Vibrato), n(4, 2), n(3, 2), n(2, 1), n(1, 0), n(0, 0)), b(2)}},
			{Beats: []tablature.Beat{b(16, n(1, 5)), b(16, n(1, 8, tablature.Legato)), b(16, n(1, 5, tablature.Legato)), b(16, n(2, 7)),
				b(8, n(2, 5)), b(8, n(2, 7, tablature.Bend)), b(4, n(1, 5, tablature.Vibrato)), b(4, n(3, 7, tablature.Slide))}},
		},
	}
}

func (a *app) length() float64 {
	q := 0.0
	for _, bar := range a.tab.Bars {
		for _, bt := range bar.Beats {
			q += 4 / float64(max(1, bt.Length))
		}
	}
	return q
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	if a.playing {
		a.playhead = a.from + c.Now().Sub(a.started).Minutes()*a.bpm
		if a.playhead >= a.length() {
			a.playing, a.playhead = false, -1
		} else {
			c.AnimationFrame()
		}
	}
	ui.Column(c).Fill().Padding(t.Space(8)).Gap(t.Space(5)).Background(t.Background).Children(func() {
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			label := "Play"
			if a.playing {
				label = "Stop"
			}
			if ui.PrimaryButton(c, label).Width(t.Space(20)).Clicked() {
				a.playing = !a.playing
				a.started, a.from = c.Now(), a.state.Cursor(&a.tab)
				if !a.playing {
					a.playhead = -1
				}
			}
			ui.NumberInput(c, &a.bpm, 40, 200, 1).Label("Tempo")
			ui.Text(c, "BPM").TextColor(t.TextMuted)
			ui.Box(c).Grow(1)
			ui.Text(c, "Type a fret · H S B V X for techniques").TextColor(t.TextMuted).FontSize(t.FontSize - 1)
		})
		tablature.View(c, &a.state, &a.tab, a.playhead).AutoFocus()
		ui.Text(c, a.tab.String()).Font("monospace").FontSize(t.FontSize - 2).TextColor(t.TextMuted).Selectable()
	})
}

func main() {
	a := &app{tab: lick(), bpm: 90, playhead: -1}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Tablature",
			Width:   980,
			Height:  620,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
