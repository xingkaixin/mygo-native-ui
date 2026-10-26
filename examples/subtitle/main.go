// Subtitle times the subtitles of a made-up clip of 24 seconds, whose
// audio is bursts of a voice-like buzz where someone speaks. Some cues are
// off their lines and one is too fast to read; drag them onto the bursts,
// or play the clip and press [ and ] to set a chosen cue's times. Copy
// SRT puts the result on the clipboard.
//
//	go run ./examples/subtitle
package main

import (
	"log"
	"math"
	"math/rand/v2"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/audio"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/subtitle"
	"github.com/xingkaixin/mygo-native-ui/ui/waveform"
)

const rate = 22050

const srt = `1
00:00:00,800 --> 00:00:03,100
Did you hear that?

2
00:00:03,900 --> 00:00:06,400
It came from the lighthouse.

3
00:00:08,300 --> 00:00:09,000
Nobody has lived there for years.

4
00:00:11,200 --> 00:00:14,600
Then who keeps the lamp burning
every single night?

5
00:00:17,000 --> 00:00:19,800
Let's find out.
`

// lines are when someone speaks in the clip.
var lines = [][2]float64{{0.6, 2.9}, {3.6, 6.2}, {7.4, 10.1}, {11.5, 15.2}, {16.4, 17.6}, {19.2, 21.0}}

// voice returns a clip with a buzz shaped as speech over each line.
func voice() []float32 {
	samples := make([]float32, 24*rate)
	r := rand.New(rand.NewPCG(7, 11))
	for _, ln := range lines {
		pitch := 110 + r.Float64()*80
		for i := int(ln[0] * rate); i < int(ln[1]*rate); i++ {
			at := float64(i)/rate - ln[0]
			span := ln[1] - ln[0]
			env := math.Sin(math.Pi*at/span) * (0.55 + 0.45*math.Sin(2*math.Pi*3.7*at)*math.Sin(2*math.Pi*1.3*at))
			ph := 2 * math.Pi * pitch * float64(i) / rate
			v := 0.5*math.Sin(ph) + 0.3*math.Sin(2*ph) + 0.2*math.Sin(3*ph) + 0.15*(r.Float64()-0.5)
			samples[i] = float32(v * env * 0.6)
		}
	}
	return samples
}

type app struct {
	player *audio.Player
	peaks  []waveform.Peak
	cues   []subtitle.Cue
	subs   subtitle.State
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	playing := a.player.Playing()
	if playing {
		a.subs.Playhead = a.player.Position()
		c.After(time.Second / 30)
	}
	ui.Column(c).Fill().Background(t.Background).Children(func() {
		ui.Row(c).Padding(t.Space(2), t.Space(3)).Gap(t.Space(2)).Children(func() {
			label := "Play"
			if playing {
				label = "Pause"
			}
			if ui.PrimaryButton(c, label).Width(t.Space(18)).Clicked() || c.Shortcut(0, ui.KeySpace) {
				if playing {
					a.player.Pause()
				} else {
					a.player.Seek(a.subs.Playhead)
					a.player.Play()
				}
			}
			ui.Spacer(c)
			if ui.Button(c, "Copy SRT").Clicked() {
				c.WriteClipboard(subtitle.Format(a.cues))
				c.Toast("Copied")
			}
		})
		ui.Divider(c)
		ev := subtitle.View(c, &a.subs, &a.cues, a.peaks, a.player.Duration())
		if at, ok := ev.Seeked(); ok {
			a.player.Seek(at)
		}
	})
}

func main() {
	samples := voice()
	cues, err := subtitle.Parse(srt)
	if err != nil {
		log.Fatal(err)
	}
	a := &app{
		player: audio.NewSound(rate, 1, samples).NewPlayer(),
		peaks:  waveform.Peaks(samples, 1<<14),
		cues:   cues,
	}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Subtitles",
			Width:   900,
			Height:  640,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
