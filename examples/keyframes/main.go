// Keyframes animates a card across a stage over three seconds: play it,
// scrub the playhead, drag keyframes, double-click a row to add one, and
// right-click one to change its ease.
//
//	go run ./examples/keyframes
package main

import (
	"fmt"
	"log"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/keyframes"
)

const length = 3 * time.Second

type app struct {
	keys     keyframes.State
	tracks   []keyframes.Track
	playhead time.Duration
	playing  bool
	started  time.Time
}

func sec(s float64) time.Duration { return time.Duration(s * float64(time.Second)) }

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	if a.playing {
		a.playhead = c.Now().Sub(a.started) % length
		c.AnimationFrame()
	}
	at := func(i int) float32 { return float32(a.tracks[i].At(a.playhead)) }
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(5)).Gap(t.Space(4)).Children(func() {
		ui.Box(c).Grow(1).Radius(t.Space(3)).Background(t.Surface).Clip().Children(func() {
			ui.Box(c).Absolute().Left(t.Space(10)).Top(t.Space(10)).Size(t.Space(24), t.Space(16)).Radius(t.Space(3)).
				Gradient(ui.Hex("#F2994A"), ui.Hex("#EB5757"), 135).Shadow(0, 6, 18, 0, ui.RGBA(0, 0, 0, 0.18)).
				Translate(at(0), at(1)).Scale(at(2), at(2)).Rotate(at(3)).Opacity(at(4))
		})
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			label := "Play"
			if a.playing {
				label = "Pause"
			}
			if ui.Button(c, label).Width(t.Space(20)).Clicked() {
				a.playing = !a.playing
				a.started = c.Now().Add(-a.playhead)
			}
			ui.Text(c, "Double-click a row to add a keyframe, right-click one to change its ease").TextColor(t.TextMuted)
		})
		if keyframes.View(c, &a.keys, a.tracks, &a.playhead, length, 30).Changed() {
			a.started = c.Now().Add(-a.playhead)
		}
	})
}

func main() {
	px := func(v float64) string { return fmt.Sprintf("%.0f px", v) }
	a := &app{tracks: []keyframes.Track{
		{Name: "X", Format: px, Keys: []keyframes.Key{{At: 0, Value: 0, Ease: keyframes.EaseInOut}, {At: sec(1.5), Value: 420, Ease: keyframes.EaseInOut}, {At: length, Value: 0}}},
		{Name: "Y", Format: px, Keys: []keyframes.Key{{At: 0, Value: 0, Ease: keyframes.EaseOut}, {At: sec(0.8), Value: 120, Ease: keyframes.EaseIn}, {At: sec(1.5), Value: 0, Ease: keyframes.Hold}, {At: sec(2.2), Value: 60}, {At: length, Value: 0}}},
		{Name: "Scale", Format: func(v float64) string { return fmt.Sprintf("%.0f%%", v*100) }, Keys: []keyframes.Key{{At: 0, Value: 1}, {At: sec(1.5), Value: 1.3, Ease: keyframes.EaseOut}, {At: length, Value: 1}}},
		{Name: "Rotation", Format: func(v float64) string { return fmt.Sprintf("%.0f°", v) }, Keys: []keyframes.Key{{At: sec(0.5), Value: 0, Ease: keyframes.EaseInOut}, {At: sec(2.5), Value: 360}}},
		{Name: "Opacity", Format: func(v float64) string { return fmt.Sprintf("%.0f%%", v*100) }, Keys: []keyframes.Key{{At: 0, Value: 1}, {At: sec(2.4), Value: 1}, {At: length, Value: 0.2}}},
	}}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Keyframes",
			Width:   960,
			Height:  680,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
