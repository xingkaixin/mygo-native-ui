// Voiceorb is a voice assistant's conversation mode: the orb in the
// middle, and above it what the assistant does, Idle, Listening, Thinking
// or Speaking. Listening and Speaking follow a made-up voice, syllables in
// bursts with pauses between words and longer ones between phrases.
// Speak plays a few seconds of synthesized speech-like tones and feeds
// the orb their level as they sound, then listens again.
//
//	go run ./examples/voiceorb
package main

import (
	"log"
	"math"
	"math/rand/v2"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/audio"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/voiceorb"
)

const rate = 48000

// The modes' labels and the captions under the orb, in voiceorb's order.
var (
	labels   = []string{"Idle", "Listening", "Thinking", "Speaking"}
	captions = []string{"Tap Speak to hear the assistant", "Listening…", "Thinking…", "Speaking…"}
)

// syllable is a burst of a made-up voice: when it starts and how long it
// lasts, in seconds, how loud it gets, and its pitch and vowel, in Hz.
type syllable struct {
	at, length, peak float64
	pitch, vowel     float64
}

// speech returns made-up speech of about secs seconds: phrases of two to
// six words, each of one to three syllables, with short gaps between the
// words and longer pauses between the phrases.
func speech(r *rand.Rand, secs float64) []syllable {
	var out []syllable
	t := 0.2
	for t < secs-1.5 {
		for range 2 + r.IntN(5) {
			for range 1 + r.IntN(3) {
				d := 0.09 + 0.15*r.Float64()
				out = append(out, syllable{
					at: t, length: d, peak: 0.45 + 0.55*r.Float64(),
					pitch: 150 + 70*r.Float64(), vowel: 500 + 700*r.Float64(),
				})
				t += d + 0.02
			}
			t += 0.05 + 0.12*r.Float64()
		}
		t += 0.5 + 0.8*r.Float64()
	}
	return out
}

// envelope returns how loud s is at t seconds: up quickly, down slower.
func (s syllable) envelope(t float64) float64 {
	x := (t - s.at) / s.length
	switch {
	case x <= 0 || x >= 1:
		return 0
	case x < 0.2:
		return s.peak * x / 0.2
	}
	return s.peak * math.Pow((1-x)/0.8, 1.5)
}

// loudness returns the level of speech at t seconds, from 0 to 1, with
// the flutter of a voice.
func loudness(speech []syllable, t float64) float32 {
	v := 0.0
	for _, s := range speech {
		v = max(v, s.envelope(t))
	}
	return float32(v * (0.88 + 0.12*math.Sin(2*math.Pi*11*t)))
}

// synthesize returns speech as samples at rate: each syllable a tone of
// its pitch, falling a little, and its harmonics, those near its vowel
// louder.
func synthesize(speech []syllable, secs float64) []float32 {
	out := make([]float32, int(secs*rate))
	for _, s := range speech {
		from, to := int(s.at*rate), min(len(out), int((s.at+s.length)*rate))
		phase := 0.0
		for i := from; i < to; i++ {
			t := float64(i) / rate
			f := s.pitch * (1 - 0.15*(t-s.at)/s.length)
			phase += 2 * math.Pi * f / rate
			sum := 0.0
			for k := 1.0; k <= 8; k++ {
				near := (k*f - s.vowel) / 250
				sum += math.Sin(k*phase) / k * (0.4 + math.Exp(-near*near))
			}
			out[i] += float32(0.18 * sum * s.envelope(t))
		}
	}
	return out
}

type app struct {
	mode int // a voiceorb.Mode, as the segmented control chooses it
	// voice is the made-up voice Listening and Speaking follow, looped
	// from start, and said the speech Speak plays as sound, by player.
	voice, said []syllable
	start       time.Time
	sound       *audio.Sound
	player      *audio.Player
}

// level returns the level of the voice the orb follows: what the player
// plays, while it does, or else the made-up voice, looped, in Listening
// and Speaking.
func (a *app) level(c *ui.Context) float32 {
	if a.player != nil {
		c.AnimationFrame()
		return loudness(a.said, a.player.Position().Seconds())
	}
	if m := voiceorb.Mode(a.mode); m != voiceorb.Listening && m != voiceorb.Speaking {
		return 0
	}
	c.AnimationFrame()
	return loudness(a.voice, math.Mod(c.Now().Sub(a.start).Seconds(), 30))
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	if a.start.IsZero() {
		a.start = c.Now()
	}
	if a.player != nil && !a.player.Playing() {
		a.player, a.mode = nil, int(voiceorb.Listening)
	}
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(6)).Gap(t.Space(4)).AlignItems(ui.Center).Children(func() {
		if ui.Segmented(c, &a.mode, labels...).Label("Assistant").Changed() && a.player != nil {
			a.player.Stop()
			a.player = nil
		}
		voiceorb.View(c, voiceorb.Mode(a.mode), a.level(c)).Grow(1).AlignSelf(ui.Stretch)
		ui.Text(c, captions[a.mode]).TextColor(t.TextMuted)
		label := "Speak"
		if a.player != nil {
			label = "Stop"
		}
		if ui.Button(c, label).MinWidth(t.Space(24)).Clicked() {
			if a.player != nil {
				a.player.Stop()
				a.player, a.mode = nil, int(voiceorb.Listening)
			} else {
				a.player, a.mode = a.sound.Play(), int(voiceorb.Speaking)
			}
		}
		if err := audio.Err(); err != nil {
			ui.Text(c, err.Error()).FontSize(t.FontSize - 2).TextColor(t.TextMuted)
		}
	})
}

func main() {
	r := rand.New(rand.NewPCG(7, 11))
	said := speech(r, 4)
	a := &app{
		mode:  int(voiceorb.Listening),
		voice: speech(r, 30),
		said:  said,
		sound: audio.NewSound(rate, 1, synthesize(said, 4)),
	}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Voice",
			Width:   460,
			Height:  560,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
