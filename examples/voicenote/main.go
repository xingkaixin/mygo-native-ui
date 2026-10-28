// Voicenote shows a chat of two voice messages, one received and not yet
// heard, one sent, their voices made as the example starts.
//
//	go run ./examples/voicenote
package main

import (
	"log"
	"math"
	"math/rand/v2"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/audio"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/voicenote"
)

const rate = 24000

// voice makes seconds of a voice saying syllables at pitch Hz: a buzz of
// harmonics shaped by vowels, each syllable swelling and fading, with
// pauses between words.
func voice(seconds, pitch float64, seed uint64) []float32 {
	r := rand.New(rand.NewPCG(seed, 3))
	out := make([]float32, int(seconds*rate))
	phase := 0.0
	syl, end := 0.0, 0.0
	var vowel [3]float64
	for i := range out {
		t := float64(i) / rate
		if t >= end {
			syl = t
			end = t + 0.12 + r.Float64()*0.18
			if r.Float64() < 0.18 {
				end += 0.25
			}
			vowel = [3]float64{300 + r.Float64()*500, 900 + r.Float64()*1400, 2400}
		}
		f := pitch * (1 + 0.04*math.Sin(2*math.Pi*1.3*t) + 0.06*math.Sin(2*math.Pi*0.3*t))
		phase += 2 * math.Pi * f / rate
		v := 0.0
		for h := 1.0; h <= 14; h++ {
			hf := f * h
			g := 0.0
			for _, fm := range vowel {
				g += math.Exp(-math.Pow((hf-fm)/180, 2))
			}
			v += (g + 0.05) / h * math.Sin(phase*h)
		}
		k := (t - syl) / 0.22
		env := 0.0
		if k < 1 {
			env = math.Sin(math.Pi * k)
		}
		out[i] = float32(0.35 * v * env)
	}
	return out
}

// peaks returns the loudest sample of each of n stretches of samples.
func peaks(samples []float32, n int) []float32 {
	out := make([]float32, n)
	top := float32(0)
	for i := range out {
		for _, s := range samples[i*len(samples)/n : (i+1)*len(samples)/n] {
			out[i] = max(out[i], float32(math.Abs(float64(s))))
		}
		top = max(top, out[i])
	}
	for i := range out {
		out[i] /= max(top, 1e-6)
	}
	return out
}

type app struct {
	theirs, mine voicenote.State
	their, my    voicenote.Note
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Padding(t.Space(6)).Gap(t.Space(4)).Background(t.Background).Children(func() {
		ui.Row(c).Children(func() {
			voicenote.View(c, &a.theirs, &a.their)
		})
		ui.Row(c).Justify(ui.End).Children(func() {
			voicenote.View(c, &a.mine, &a.my)
		})
	})
}

func main() {
	their := voice(7.4, 190, 1)
	mine := voice(3.1, 120, 2)
	a := &app{
		their: voicenote.Note{Peaks: peaks(their, 120), Transcript: "Hey! The train's running late, so I'll be there around seven. Save me a seat?"},
		my:    voicenote.Note{Peaks: peaks(mine, 120), Mine: true, Heard: true},
	}
	a.theirs.Player = audio.NewSound(rate, 1, their).NewPlayer()
	a.mine.Player = audio.NewSound(rate, 1, mine).NewPlayer()
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Voice Notes",
			Width:   560,
			Height:  360,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
