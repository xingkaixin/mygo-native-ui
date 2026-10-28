// Lyrics plays the made-up song Paper Boats, its words timed one by one,
// with an instrumental break between its verses.
//
//	go run ./examples/lyrics
package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/lyrics"
)

// song is the song's lines, each sung over four seconds; an empty line is
// a break.
var song = []string{
	"We folded paper boats from yesterday's news",
	"and set them on the river by the old canal",
	"You said the ink would keep them warm",
	"I said the rain would take them all",
	"",
	"So let them sail, let them sail",
	"past the bridges and the factory wall",
	"Every word we didn't say",
	"is floating out to sea",
	"",
	"Now the river's running higher",
	"and the headlines all are gone",
	"but I still fold a boat for you",
	"and I still sing this song",
}

// lrc writes the song as enhanced LRC, its words spread over each line's
// four seconds.
func lrc() string {
	var b strings.Builder
	b.WriteString("[ti:Paper Boats]\n[ar:The Examples]\n")
	stamp := func(d time.Duration) string {
		return fmt.Sprintf("%02d:%05.2f", int(d.Minutes()), d.Seconds()-float64(int(d.Minutes())*60))
	}
	at := 2 * time.Second
	for _, line := range song {
		fmt.Fprintf(&b, "[%s]", stamp(at))
		words := strings.Fields(line)
		for i, w := range words {
			fmt.Fprintf(&b, "<%s>%s ", stamp(at+time.Duration(i)*3200*time.Millisecond/time.Duration(len(words))), w)
		}
		b.WriteString("\n")
		at += 4 * time.Second
	}
	return b.String()
}

type app struct {
	lines    []lyrics.Line
	lyrics   lyrics.State
	playhead time.Duration
	playing  bool
	last     time.Time
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	now := c.Now()
	end := a.lines[len(a.lines)-1].At + 4*time.Second
	if a.playing {
		if !a.last.IsZero() {
			a.playhead += now.Sub(a.last)
		}
		a.last = now
		if a.playhead >= end {
			a.playhead, a.playing = end, false
		}
		c.After(time.Second / 30)
	}
	ui.Column(c).Fill().Background(t.Background).Children(func() {
		lyrics.View(c, &a.lyrics, a.lines, &a.playhead).Grow(1)
		ui.Row(c).Gap(t.Space(3)).Padding(t.Space(4)).AlignItems(ui.Center).Children(func() {
			label := "Play"
			if a.playing {
				label = "Pause"
			}
			if ui.PrimaryButton(c, label).Clicked() {
				a.playing, a.last = !a.playing, time.Time{}
				if a.playhead >= end {
					a.playhead = 0
				}
			}
			pos := a.playhead.Seconds()
			ui.Slider(c, &pos, 0, end.Seconds()).Grow(1)
			a.playhead = time.Duration(pos * float64(time.Second))
			ui.Text(c, fmt.Sprintf("%d:%02d", int(pos)/60, int(pos)%60)).FontFeatures("tnum").TextColor(t.TextMuted)
		})
	})
}

func main() {
	lines, err := lyrics.ParseLRC(lrc())
	if err != nil {
		log.Fatal(err)
	}
	a := &app{lines: lines, playing: true}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Lyrics",
			Width:   640,
			Height:  720,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
