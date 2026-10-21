// Spectrogram plays a WAV file, mixed to one channel, or, given none, a
// loop it makes of a rising chirp, a chord, a melody and a burst of
// noise, and shows it as it plays: the last ten seconds live, and the
// whole of it below, where a click or a drag seeks. Space or the button
// plays and pauses.
//
//	go run ./examples/spectrogram [file.wav]
package main

import (
	"io"
	"log"
	"math"
	"math/rand/v2"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/audio"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/spectrogram"
)

const rate = 48000

func hz(midi int) float64 { return 440 * math.Pow(2, float64(midi-69)/12) }

// loop makes 8.5 seconds: a sine sweeping up from 80 Hz to 12 kHz, a
// chord of C major with its harmonics, a melody with vibrato and
// harmonics, a burst of white noise, and silence.
func loop() []float32 {
	out := make([]float32, int(8.5*rate))
	// put adds sound(t) from start for length seconds, faded in and out
	// over 10 ms so that no edge clicks.
	put := func(start, length float64, sound func(t float64) float64) {
		at := int(start * rate)
		for i := range int(length * rate) {
			t := float64(i) / rate
			fade := min(1, t/0.01, (length-t)/0.01)
			out[at+i] += float32(fade * sound(t))
		}
	}
	// note is a tone of f with harmonics falling 6 dB an octave.
	note := func(f, t float64) float64 {
		v := 0.0
		for k := 1.0; k <= 8 && k*f < rate/2; k++ {
			v += math.Sin(2*math.Pi*k*f*t) / k
		}
		return v
	}
	const f0, f1, sweep = 80.0, 12000.0, 2.5
	put(0, sweep, func(t float64) float64 {
		k := math.Log(f1 / f0)
		return 0.5 * math.Sin(2*math.Pi*f0*sweep/k*(math.Exp(k*t/sweep)-1))
	})
	put(2.6, 2, func(t float64) float64 {
		v := 0.0
		for _, m := range []int{48, 64, 67, 72} {
			v += note(hz(m), t)
		}
		return 0.12 * v * math.Exp(-t/1.2)
	})
	for i, m := range []int{76, 74, 72, 74, 76, 76, 76, 79} {
		put(4.7+float64(i)*0.36, 0.34, func(t float64) float64 {
			f := hz(m) * (1 + 0.006*math.Sin(2*math.Pi*5.5*t))
			return 0.22 * note(f, t) * math.Exp(-t/0.4)
		})
	}
	r := rand.New(rand.NewPCG(1, 2))
	put(7.7, 0.4, func(t float64) float64 { return 0.5 * (r.Float64()*2 - 1) * math.Exp(-t/0.12) })
	return out
}

// load decodes the file at path, mixed down to one channel.
func load(path string) ([]float32, int, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, 0, err
	}
	defer f.Close()
	d, _, err := audio.NewDecoder(f)
	if err != nil {
		return nil, 0, err
	}
	ch := d.Channels()
	buf := make([]float32, 4096/ch*ch)
	var out []float32
	for {
		n, err := d.Read(buf)
		for i := 0; i+ch <= n; i += ch {
			v := float32(0)
			for _, s := range buf[i : i+ch] {
				v += s
			}
			out = append(out, v/float32(ch))
		}
		if err == io.EOF {
			return out, d.SampleRate(), nil
		}
		if err != nil {
			return nil, 0, err
		}
		if n == 0 {
			return nil, 0, io.ErrNoProgress
		}
	}
}

type app struct {
	title   string
	samples []float32
	rate    int
	player  *audio.Player
	gram    *spectrogram.Analysis
	// heard is the sample the live view has been given up to.
	heard       int
	linear      int
	live, whole spectrogram.State
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	playing := a.player.Playing()
	at := min(len(a.samples), int(a.player.Position().Seconds()*float64(a.rate)))
	var heard []float32
	if playing {
		if at >= a.heard {
			heard = a.samples[a.heard:at]
		} else { // looped
			heard = append(slices.Clip(a.samples[a.heard:]), a.samples[:at]...)
		}
		c.After(time.Second / 60)
	}
	a.heard = at
	a.live.Linear = a.linear == 1
	a.whole.Linear, a.whole.Floor = a.live.Linear, a.live.Floor
	a.whole.Playhead = float64(at) / float64(max(1, len(a.samples)))
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(6)).Gap(t.Space(4)).Children(func() {
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			label := "Play"
			if playing {
				label = "Pause"
			}
			if ui.PrimaryButton(c, label).Width(t.Space(20)).Clicked() || c.Shortcut(0, ui.KeySpace) {
				if playing {
					a.player.Pause()
				} else {
					a.player.Play()
				}
			}
			ui.Text(c, a.title).Bold().SingleLine()
			ui.Text(c, clock(a.player.Position())+" / "+clock(a.player.Duration())).TextColor(t.TextMuted).FontFeatures("tnum")
			ui.Spacer(c)
			ui.Segmented(c, &a.linear, "Log", "Linear").Label("Frequency scale")
			ui.Text(c, "Floor").TextColor(t.TextMuted)
			ui.Slider(c, &a.live.Floor, -140, -40).Step(1).Label("Floor").Width(t.Space(36))
			ui.Text(c, strconv.Itoa(int(a.live.Floor))+" dB").FontFeatures("tnum").Width(t.Space(14))
		})
		spectrogram.Live(c, &a.live, heard, a.rate).Grow(1)
		if spectrogram.Clip(c, &a.whole, a.gram).Height(t.Space(36)).Changed() {
			a.heard = int(a.whole.Playhead * float64(len(a.samples)))
			a.player.Seek(time.Duration(float64(a.heard) / float64(a.rate) * float64(time.Second)))
		}
	})
}

// clock formats d as m:ss.
func clock(d time.Duration) string {
	s := int(d.Seconds())
	return strconv.Itoa(s/60) + ":" + strconv.Itoa(s%60/10) + strconv.Itoa(s%10)
}

func main() {
	a := &app{title: "Test signal", samples: loop(), rate: rate}
	if len(os.Args) > 1 {
		var err error
		if a.samples, a.rate, err = load(os.Args[1]); err != nil {
			log.Fatal(err)
		}
		a.title = filepath.Base(os.Args[1])
	}
	a.live.Floor = -100
	a.gram = spectrogram.Analyze(a.samples, a.rate)
	a.player = audio.NewSound(a.rate, 1, a.samples).NewPlayer().SetLoop(true)
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Spectrogram",
			Width:   960,
			Height:  600,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
